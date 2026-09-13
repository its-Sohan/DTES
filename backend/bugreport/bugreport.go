package bugreport

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"itt-ocr/backend/config"
	"itt-ocr/backend/updater"
)

var redactURLRegex = regexp.MustCompile(`://[^/@]*@`)

func redactURL(url string) string {
	u := strings.TrimSpace(url)
	if u == "" {
		return ""
	}
	if strings.Contains(u, "?") {
		u = strings.Split(u, "?")[0] + "?<redacted>"
	}
	if strings.Contains(u, "#") {
		u = strings.Split(u, "#")[0]
	}
	return redactURLRegex.ReplaceAllString(u, "://***@")
}

// GenerateBugReport builds an anonymized diagnostic bundle
func GenerateBugReport(description string, steps string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	reportsDir := filepath.Join(home, ".itt_ocr_client", "bug_reports")
	_ = os.MkdirAll(reportsDir, 0755)

	cfg, _ := config.LoadConfig()
	history, _ := config.LoadHistory()

	redactedKey := ""
	if cfg.APIKey != "" {
		redactedKey = "***set***"
	}

	type historySummary struct {
		FileName string `json:"file_name"`
		FileSize string `json:"file_size"`
		Status   string `json:"status"`
		Source   string `json:"source"`
		Mode     string `json:"mode"`
		Error    string `json:"error"`
	}

	var histList []historySummary
	limit := 50
	start := 0
	if len(history) > limit {
		start = len(history) - limit
	}
	for _, it := range history[start:] {
		errSnippet := it.ErrorMessage
		if len(errSnippet) > 300 {
			errSnippet = errSnippet[:300]
		}
		histList = append(histList, historySummary{
			FileName: it.FileName,
			FileSize: it.FileSizeStr,
			Status:   it.Status,
			Source:   it.Source,
			Mode:     it.OutputMode,
			Error:    errSnippet,
		})
	}

	report := map[string]interface{}{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"app_version":  updater.AppVersion,
		"system": map[string]string{
			"os":       runtime.GOOS,
			"arch":     runtime.GOARCH,
			"compiler": runtime.Compiler,
		},
		"user_inputs": map[string]string{
			"description": description,
			"steps":       steps,
		},
		"configuration": map[string]interface{}{
			"model_name":          cfg.ModelName,
			"base_url":            redactURL(cfg.BaseURL),
			"api_key":             redactedKey,
			"auto_extract":        cfg.AutoExtract,
			"default_output_mode": cfg.DefaultOutputMode,
			"quality":             cfg.Quality,
			"releases_repo":       cfg.ReleasesRepo,
			"usage_stats":         cfg.UsageStats,
		},
		"recent_queue_history": histList,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}

	filePath := filepath.Join(reportsDir, fmt.Sprintf("bug_report_%d.json", time.Now().Unix()))
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}

	return filePath, nil
}
