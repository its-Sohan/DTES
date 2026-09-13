package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"itt-ocr/backend/config"
)

const AppVersion = "1.0.0"

type ReleaseInfo struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

type UpdateCheckResult struct {
	HasUpdate      bool        `json:"has_update"`
	CurrentVersion string      `json:"current_version"`
	LatestVersion  string      `json:"latest_version"`
	ReleaseNotes   string      `json:"release_notes"`
	ReleaseURL     string      `json:"release_url"`
	DownloadURL    string      `json:"download_url"`
	Error          string      `json:"error,omitempty"`
}

// CheckForUpdates queries GitHub Releases API for updates
func CheckForUpdates() (UpdateCheckResult, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return UpdateCheckResult{CurrentVersion: AppVersion}, err
	}

	repo := strings.Trim(strings.TrimSpace(cfg.ReleasesRepo), "/")
	if repo == "" {
		repo = "its-Sohan/itt-ocr-release"
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return UpdateCheckResult{CurrentVersion: AppVersion}, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "ITT-OCR-Desktop-Client")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return UpdateCheckResult{
			CurrentVersion: AppVersion,
			HasUpdate:      false,
			Error:          "Unable to connect to update server. Check your internet connection.",
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return UpdateCheckResult{
			CurrentVersion: AppVersion,
			HasUpdate:      false,
			Error:          "No releases found for this repository.",
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return UpdateCheckResult{
			CurrentVersion: AppVersion,
			HasUpdate:      false,
			Error:          fmt.Sprintf("Update check returned HTTP %d", resp.StatusCode),
		}, nil
	}

	var rel ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return UpdateCheckResult{CurrentVersion: AppVersion}, err
	}

	latestClean := strings.TrimPrefix(strings.TrimSpace(rel.TagName), "v")
	hasUpdate := isNewerVersion(AppVersion, latestClean)

	downloadURL := rel.HTMLURL
	for _, a := range rel.Assets {
		if strings.HasSuffix(strings.ToLower(a.Name), ".exe") ||
			strings.HasSuffix(strings.ToLower(a.Name), ".deb") ||
			strings.HasSuffix(strings.ToLower(a.Name), ".appimage") ||
			strings.HasSuffix(strings.ToLower(a.Name), ".dmg") {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}

	return UpdateCheckResult{
		HasUpdate:      hasUpdate,
		CurrentVersion: AppVersion,
		LatestVersion:  latestClean,
		ReleaseNotes:   rel.Body,
		ReleaseURL:     rel.HTMLURL,
		DownloadURL:    downloadURL,
	}, nil
}

func isNewerVersion(current, latest string) bool {
	cParts := strings.Split(current, ".")
	lParts := strings.Split(latest, ".")

	for i := 0; i < len(cParts) && i < len(lParts); i++ {
		var cNum, lNum int
		fmt.Sscanf(cParts[i], "%d", &cNum)
		fmt.Sscanf(lParts[i], "%d", &lNum)
		if lNum > cNum {
			return true
		} else if lNum < cNum {
			return false
		}
	}
	return len(lParts) > len(cParts)
}
