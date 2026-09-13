// Package bugreport builds an anonymised diagnostic bundle on the user's disk.
//
// The bundle is written locally and never transmitted; the user chooses whether
// to attach it to an issue. It deliberately excludes the API key and every
// character of extracted document text, since those are the two things most
// likely to be sensitive.
package bugreport

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"itt-ocr/backend/config"
	"itt-ocr/backend/types"
	"itt-ocr/backend/version"
)

// historyLimit caps how many recent queue entries are summarised.
const historyLimit = 50

// errorSnippetLimit caps each captured error message.
const errorSnippetLimit = 300

// Report is the diagnostic bundle written to disk.
type Report struct {
	GeneratedAt   string           `json:"generated_at"`
	Build         version.Info     `json:"build"`
	UserInputs    UserInputs       `json:"user_inputs"`
	Configuration Configuration    `json:"configuration"`
	UsageStats    types.UsageStats `json:"usage_stats"`
	RecentQueue   []QueueSummary   `json:"recent_queue_history"`
	Runtime       RuntimeInfo      `json:"runtime"`
}

// UserInputs is what the user typed into the report form.
type UserInputs struct {
	Description string `json:"description"`
	Steps       string `json:"steps"`
}

// Configuration is the redacted settings snapshot.
//
// APIKeyPresent is a boolean rather than any form of the key itself: even a
// prefix or length can help an attacker, and neither helps diagnose a bug.
type Configuration struct {
	BaseURL           string `json:"base_url"`
	ModelName         string `json:"model_name"`
	APIKeyPresent     bool   `json:"api_key_present"`
	AutoExtract       bool   `json:"auto_extract"`
	DefaultOutputMode string `json:"default_output_mode"`
	Quality           string `json:"quality"`
	ReleasesRepo      string `json:"releases_repo"`
	CheckUpdates      bool   `json:"check_updates_on_startup"`
}

// RuntimeInfo describes the host, which is often the key to a platform bug.
type RuntimeInfo struct {
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	NumCPU     int    `json:"num_cpu"`
	GoRoutines int    `json:"goroutines"`
}

// QueueSummary is a redacted queue entry: file names and statuses only, never
// the extracted text.
type QueueSummary struct {
	FileName   string `json:"file_name"`
	FileSize   string `json:"file_size"`
	Status     string `json:"status"`
	Source     string `json:"source"`
	OutputMode string `json:"output_mode"`
	// TextLength records how much text was produced without revealing any of it.
	TextLength int    `json:"extracted_text_length"`
	Error      string `json:"error,omitempty"`
}

// GenerateBugReport is an alias for Generate for backward compatibility.
func GenerateBugReport(description, steps string) (string, error) {
	return Generate(description, steps)
}

// Generate writes a diagnostic bundle and returns its path.
func Generate(description, steps string) (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	reportsDir := filepath.Join(dir, "bug_reports")
	if err := os.MkdirAll(reportsDir, 0o700); err != nil {
		return "", fmt.Errorf("create reports directory: %w", err)
	}

	cfg, _ := config.LoadConfig()
	history, _ := config.LoadHistory()

	report := Report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Build:       version.Current(),
		UserInputs: UserInputs{
			Description: strings.TrimSpace(description),
			Steps:       strings.TrimSpace(steps),
		},
		Configuration: Configuration{
			BaseURL:           RedactURL(cfg.BaseURL),
			ModelName:         cfg.ModelName,
			APIKeyPresent:     strings.TrimSpace(cfg.APIKey) != "",
			AutoExtract:       cfg.AutoExtract,
			DefaultOutputMode: cfg.DefaultOutputMode,
			Quality:           cfg.Quality,
			ReleasesRepo:      cfg.ReleasesRepo,
			CheckUpdates:      cfg.CheckUpdatesStartup,
		},
		UsageStats:  cfg.UsageStats,
		RecentQueue: summariseHistory(history),
		Runtime: RuntimeInfo{
			OS:         runtime.GOOS,
			Arch:       runtime.GOARCH,
			NumCPU:     runtime.NumCPU(),
			GoRoutines: runtime.NumGoroutine(),
		},
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode report: %w", err)
	}

	path := filepath.Join(reportsDir,
		fmt.Sprintf("bug_report_%s.json", time.Now().UTC().Format("20060102-150405")))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write report: %w", err)
	}
	return path, nil
}

// summariseHistory redacts the most recent queue entries.
func summariseHistory(history []types.QueueItem) []QueueSummary {
	if len(history) > historyLimit {
		history = history[len(history)-historyLimit:]
	}

	out := make([]QueueSummary, 0, len(history))
	for _, it := range history {
		msg := it.ErrorMessage
		if r := []rune(msg); len(r) > errorSnippetLimit {
			msg = string(r[:errorSnippetLimit]) + "…"
		}
		out = append(out, QueueSummary{
			FileName:   it.FileName,
			FileSize:   it.FileSizeStr,
			Status:     string(it.Status),
			Source:     string(it.Source),
			OutputMode: it.OutputMode,
			TextLength: len([]rune(it.ExtractedText)),
			Error:      msg,
		})
	}
	return out
}

// RedactURL strips credentials, query strings and fragments from a URL so a
// key-in-query-parameter endpoint cannot leak through a diagnostic bundle.
//
// On a parse failure the input is discarded rather than passed through, because
// echoing an unparseable string risks leaking exactly what we meant to remove.
func RedactURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "<unparseable url redacted>"
	}

	if u.User != nil {
		u.User = url.User("***")
	}
	if u.RawQuery != "" {
		u.RawQuery = "<redacted>"
	}
	u.Fragment = ""

	return u.String()
}
