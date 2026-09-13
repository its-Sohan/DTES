// Package updater checks GitHub Releases for a newer version of the app.
//
// The updater is deliberately advisory only: it never downloads or executes
// anything. It reports what is available and hands the user a URL, which keeps
// the security surface of a desktop app that ships no code-signing story to a
// minimum.
package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	exts, ok := platformExtensions[goos]
	if !ok {
		return Asset{}, false
	}

	var osOnly *Asset
	// Iterate extensions outermost so preference order is honoured.
	for _, ext := range exts {
		for i := range assets {
			name := strings.ToLower(assets[i].Name)
			if !strings.HasSuffix(name, ext) {
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

	if osOnly != nil {
		return *osOnly, true
	}
	return Asset{}, false
}
