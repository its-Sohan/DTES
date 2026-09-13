package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"itt-ocr/backend/types"
)

var (
	mu sync.RWMutex
)

func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".itt_ocr_client")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func getConfigFile() (string, error) {
	dir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func getHistoryFile() (string, error) {
	dir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}

// DefaultConfig returns the standard initial configuration
func DefaultConfig() types.Config {
	return types.Config{
		SessionAccount:      "default_user",
		APIKey:              "",
		BaseURL:             "https://api.openai.com/v1",
		ModelName:           "gpt-4o-mini",
		AutoExtract:         true,
		DefaultOutputMode:   "document",
		Quality:             "standard",
		ReleasesRepo:        "its-Sohan/itt-ocr-release",
		CheckUpdatesStartup: true,
		UsageStats: types.UsageStats{
			TotalScannedOrUploaded: 0,
			TotalProcessed:         0,
			TotalCharacters:        0,
			SuccessfulRuns:         0,
			FailedRuns:             0,
		},
	}
}

// LoadConfig reads config from ~/.itt_ocr_client/config.json with default fallbacks
func LoadConfig() (types.Config, error) {
	mu.RLock()
	defer mu.RUnlock()

	cfg := DefaultConfig()
	filePath, err := getConfigFile()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			_ = saveConfigUnsafe(cfg)
			return cfg, nil
		}
		return cfg, err
	}

	var loaded types.Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		return cfg, err
	}

	// Ensure defaults if empty
	if loaded.BaseURL == "" {
		loaded.BaseURL = cfg.BaseURL
	}
	if loaded.ModelName == "" {
		loaded.ModelName = cfg.ModelName
	}
	if loaded.DefaultOutputMode == "" {
		loaded.DefaultOutputMode = cfg.DefaultOutputMode
	}
	if loaded.Quality == "" {
		loaded.Quality = cfg.Quality
	}
	if loaded.ReleasesRepo == "" {
		loaded.ReleasesRepo = cfg.ReleasesRepo
	}

	return loaded, nil
}

func saveConfigUnsafe(cfg types.Config) error {
	filePath, err := getConfigFile()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// SaveConfig persists the given configuration
func SaveConfig(cfg types.Config) error {
	mu.Lock()
	defer mu.Unlock()
	return saveConfigUnsafe(cfg)
}

// UpdateUsageStats safely increments usage counters
func UpdateUsageStats(chars int, success bool, scannedOrUploaded int) (types.UsageStats, error) {
	mu.Lock()
	defer mu.Unlock()

	cfg, _ := LoadConfig()
	stats := cfg.UsageStats

	if scannedOrUploaded > 0 {
		stats.TotalScannedOrUploaded += scannedOrUploaded
	}
	stats.TotalProcessed++
	if success {
		stats.SuccessfulRuns++
		stats.TotalCharacters += chars
	} else {
		stats.FailedRuns++
	}

	cfg.UsageStats = stats
	_ = saveConfigUnsafe(cfg)
	return stats, nil
}

// LoadHistory reads the persisted queue from ~/.itt_ocr_client/history.json
func LoadHistory() ([]types.QueueItem, error) {
	mu.RLock()
	defer mu.RUnlock()

	filePath, err := getHistoryFile()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.QueueItem{}, nil
		}
		return nil, err
	}

	var items []types.QueueItem
	if err := json.Unmarshal(data, &items); err != nil {
		return []types.QueueItem{}, nil
	}
	return items, nil
}

// SaveHistory writes the queue items to ~/.itt_ocr_client/history.json
func SaveHistory(items []types.QueueItem) error {
	mu.Lock()
	defer mu.Unlock()

	filePath, err := getHistoryFile()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}
