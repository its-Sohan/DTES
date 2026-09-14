// Package updater checks GitHub Releases for a newer version of the app.
//
// The updater is deliberately advisory only: it never downloads or executes
// anything. It reports what is available and hands the user a URL, which keeps
// the security surface of a desktop app that ships no code-signing story to a
// minimum.
package updater

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"itt-ocr/backend/config"
	"itt-ocr/backend/version"
)

// checkTimeout keeps a startup update check from delaying the UI.
const checkTimeout = 10 * time.Second

// maxBodyBytes bounds the response we will parse from the release API.
const maxBodyBytes = 2 << 20 // 2 MiB

// repoPattern validates an "owner/name" GitHub repository reference before it
// is interpolated into a request URL.
var repoPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// Asset is a downloadable file attached to a release.
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

// release mirrors the subset of the GitHub Releases payload we consume.
type release struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	Draft       bool    `json:"draft"`
	Prerelease  bool    `json:"prerelease"`
	PublishedAt string  `json:"published_at"`
	HTMLURL     string  `json:"html_url"`
	Assets      []Asset `json:"assets"`
}

// CheckResult is the outcome of an update check.
//
// Error is a user-facing string rather than a Go error because a failed update
// check is informational, not exceptional: the UI shows it inline and the app
// continues to work normally. Callers therefore receive a populated result with
// Error set instead of a non-nil error for network and API problems.
type CheckResult struct {
	HasUpdate      bool   `json:"has_update"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	ReleaseName    string `json:"release_name"`
	ReleaseNotes   string `json:"release_notes"`
	ReleaseURL     string `json:"release_url"`
	PublishedAt    string `json:"published_at"`
	// DownloadURL points at the asset matching the running platform when one
	// exists, and otherwise at the release page.
	DownloadURL string `json:"download_url"`
	// AssetName is the file name behind DownloadURL, empty when falling back
	// to the release page.
	AssetName string `json:"asset_name"`
	Error     string `json:"error,omitempty"`
}

// UpdateCheckResult is an alias for CheckResult for backward compatibility.
type UpdateCheckResult = CheckResult

// httpClient is shared so repeated checks reuse connections.
var httpClient = &http.Client{Timeout: checkTimeout}

// CheckForUpdates is a convenience wrapper around Check using a background context.
func CheckForUpdates() (CheckResult, error) {
	return Check(context.Background())
}

// Check queries the configured repository's latest release.
//
// It returns an error only for programming-level problems; network failures,
// rate limits and missing releases are reported in CheckResult.Error so a
// transient outage never surfaces as a crash or a scary dialog.
func Check(ctx context.Context) (CheckResult, error) {
	result := CheckResult{CurrentVersion: version.Version}

	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	repo := strings.Trim(strings.TrimSpace(cfg.ReleasesRepo), "/")
	if repo == "" {
		repo = config.DefaultConfig().ReleasesRepo
	}
	if !repoPattern.MatchString(repo) {
		result.Error = fmt.Sprintf("%q is not a valid owner/repository reference.", repo)
		return result, nil
	}

	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return result, fmt.Errorf("build update request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", version.UserAgent())

	resp, err := httpClient.Do(req)
	if err != nil {
		result.Error = "Could not reach the update server. Check your internet connection."
		return result, nil
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
		_ = resp.Body.Close()
	}()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		result.Error = fmt.Sprintf("No published releases found for %s.", repo)
		return result, nil
	case resp.StatusCode == http.StatusForbidden, resp.StatusCode == http.StatusTooManyRequests:
		// GitHub rate-limits unauthenticated requests per IP.
		result.Error = "Update checks are temporarily rate limited by GitHub. Try again later."
		return result, nil
	case resp.StatusCode != http.StatusOK:
		result.Error = fmt.Sprintf("The update server returned HTTP %d.", resp.StatusCode)
		return result, nil
	}

	var rel release
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodyBytes)).Decode(&rel); err != nil {
		result.Error = "The update server returned a response this version could not read."
		return result, nil
	}

	latest := strings.TrimSpace(rel.TagName)
	if latest == "" {
		result.Error = "The latest release has no version tag."
		return result, nil
	}

	result.LatestVersion = strings.TrimPrefix(latest, "v")
	result.ReleaseName = rel.Name
	result.ReleaseNotes = rel.Body
	result.ReleaseURL = rel.HTMLURL
	result.PublishedAt = rel.PublishedAt
	// Drafts and pre-releases are never offered as updates.
	result.HasUpdate = !rel.Draft && !rel.Prerelease &&
		version.IsNewer(version.Version, result.LatestVersion)

	if asset, ok := selectAsset(rel.Assets, runtime.GOOS, runtime.GOARCH); ok {
		result.DownloadURL = asset.DownloadURL
		result.AssetName = asset.Name
	} else {
		result.DownloadURL = rel.HTMLURL
	}

	return result, nil
}

// platformExtensions lists installer/package suffixes per OS, most preferred
// first.
var platformExtensions = map[string][]string{
	"windows": {".exe", ".msi", ".zip"},
	"darwin":  {".dmg", ".pkg", ".zip"},
	"linux":   {".appimage", ".deb", ".rpm", ".tar.gz"},
}

// archAliases lists the tokens a release asset might use for an architecture.
var archAliases = map[string][]string{
	"amd64": {"amd64", "x86_64", "x64"},
	"arm64": {"arm64", "aarch64"},
	"386":   {"386", "i386", "x86"},
}

// selectAsset picks the release asset best matching the running platform.
//
// An asset naming the correct architecture is preferred over one that only
// matches the OS, so a user on arm64 is not handed an amd64 build. When nothing
// matches, callers fall back to the release page rather than guessing.
func selectAsset(assets []Asset, goos, goarch string) (Asset, bool) {
	// For Windows, prefer an installer package over standalone executable if both exist.
	if goos == "windows" {
		for _, a := range assets {
			name := strings.ToLower(a.Name)
			if strings.HasSuffix(name, ".exe") && (strings.Contains(name, "installer") || strings.Contains(name, "setup")) {
				for _, alias := range archAliases[goarch] {
					if strings.Contains(name, alias) {
						return a, true
					}
				}
			}
		}
	}

	exts, ok := platformExtensions[goos]
	if !ok {
		return Asset{}, false
	}

	var osOnly *Asset
	// Iterate extensions outermost so preference order is honoured.
	for _, ext := range exts {
		for i := range assets {
			name := strings.ToLower(assets[i].Name)
			if ext != "" && !strings.HasSuffix(name, ext) {
				continue
			}
			for _, alias := range archAliases[goarch] {
				if strings.Contains(name, alias) {
					return assets[i], true
				}
			}
			if osOnly == nil {
				osOnly = &assets[i]
			}
		}
	}

	// For Linux, also check bare binaries without standard extension
	if goos == "linux" {
		for _, a := range assets {
			name := strings.ToLower(a.Name)
			if !strings.Contains(name, ".") || strings.HasSuffix(name, ".zip") {
				for _, alias := range archAliases[goarch] {
					if strings.Contains(name, alias) {
						return a, true
					}
				}
			}
		}
	}

	if osOnly != nil {
		return *osOnly, true
	}
	return Asset{}, false
}

// DownloadAndInstall streams the update package from downloadURL to a secure
// temporary file, emitting progress (0-100) via onProgress, and launches the installer.
func DownloadAndInstall(ctx context.Context, downloadURL string, onProgress func(int)) error {
	u, err := url.Parse(downloadURL)
	if err != nil {
		return fmt.Errorf("invalid download url: %w", err)
	}

	// Security guard: Only allow downloads from official github endpoints
	host := strings.ToLower(u.Host)
	if !strings.HasSuffix(host, "github.com") &&
		!strings.HasSuffix(host, "githubusercontent.com") {
		return fmt.Errorf("untrusted download source %q (must be github.com)", u.Host)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("prepare download request: %w", err)
	}
	req.Header.Set("User-Agent", version.UserAgent())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with HTTP %d", resp.StatusCode)
	}

	filename := filepath.Base(u.Path)
	if filename == "" || filename == "/" || filename == "." {
		filename = "itt-ocr-update.exe"
	}

	tempDir := os.TempDir()
	tempPath := filepath.Join(tempDir, fmt.Sprintf("itt-ocr-update-%d-%s", time.Now().Unix(), filename))
	out, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("create temporary update file: %w", err)
	}

	totalSize := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	lastReport := time.Now()

	for {
		n, rErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := out.Write(buf[:n]); wErr != nil {
				_ = out.Close()
				_ = os.Remove(tempPath)
				return fmt.Errorf("write update file: %w", wErr)
			}
			downloaded += int64(n)
			if totalSize > 0 && onProgress != nil {
				if time.Since(lastReport) > 100*time.Millisecond {
					pct := int(float64(downloaded) / float64(totalSize) * 100)
					if pct > 100 {
						pct = 100
					}
					onProgress(pct)
					lastReport = time.Now()
				}
			}
		}
		if rErr == io.EOF {
			break
		}
		if rErr != nil {
			_ = out.Close()
			_ = os.Remove(tempPath)
			return fmt.Errorf("read update stream: %w", rErr)
		}
	}

	if err := out.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close downloaded update file: %w", err)
	}

	if onProgress != nil {
		onProgress(100)
	}

	// Make executable on unix platforms
	if runtime.GOOS != "windows" {
		_ = os.Chmod(tempPath, 0755)
	}

	runPath := tempPath

	// If the downloaded asset is a zip archive, extract it and locate the executable/installer inside
	if strings.HasSuffix(strings.ToLower(tempPath), ".zip") {
		extractDir := filepath.Join(tempDir, fmt.Sprintf("itt-ocr-extracted-%d", time.Now().Unix()))
		if err := unzipFile(tempPath, extractDir); err != nil {
			return fmt.Errorf("unzip update archive: %w", err)
		}
		foundExe := findExecutableInDir(extractDir)
		if foundExe != "" {
			runPath = foundExe
			if runtime.GOOS != "windows" {
				_ = os.Chmod(runPath, 0755)
			}
		}
	}

	// Launch installer / executable detached
	if runtime.GOOS == "windows" {
		cmd := exec.Command(runPath)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("launch installer: %w", err)
		}
	} else if runtime.GOOS == "darwin" {
		_ = exec.Command("open", runPath).Start()
	} else {
		// Linux: open containing directory or run
		if strings.HasSuffix(strings.ToLower(runPath), ".appimage") || !strings.Contains(filepath.Base(runPath), ".") {
			_ = exec.Command(runPath).Start()
		} else {
			_ = exec.Command("xdg-open", filepath.Dir(runPath)).Start()
		}
	}

	return nil
}

func unzipFile(srcZip, destDir string) error {
	r, err := zip.OpenReader(srcZip)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	for _, f := range r.File {
		targetPath := filepath.Join(destDir, f.Name)
		// Protect against Zip Slip vulnerabilities
		cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)
		if !strings.HasPrefix(filepath.Clean(targetPath)+string(os.PathSeparator), cleanDest) &&
			filepath.Clean(targetPath) != filepath.Clean(destDir) {
			continue
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(targetPath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func findExecutableInDir(dir string) string {
	var candidate string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := strings.ToLower(info.Name())
		if runtime.GOOS == "windows" {
			if strings.HasSuffix(name, ".exe") {
				// Strongly prefer installer/setup executable
				if strings.Contains(name, "installer") || strings.Contains(name, "setup") {
					candidate = path
					return filepath.SkipAll
				}
				if candidate == "" {
					candidate = path
				}
			}
		} else {
			if info.Mode()&0111 != 0 {
				candidate = path
				return filepath.SkipAll
			}
		}
		return nil
	})
	return candidate
}
