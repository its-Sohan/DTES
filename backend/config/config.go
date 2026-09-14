// Package config owns on-disk persistence of user settings, usage metrics and
// queue history under the per-user application data directory.
//
// All exported functions are safe for concurrent use. Internally a single
// RWMutex guards the data directory; unexported "unsafe" helpers assume the
// caller already holds the appropriate lock, which keeps compound
// read-modify-write operations (such as RecordExtraction) free of the
// re-entrant locking that Go's sync.RWMutex does not support.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"itt-ocr/backend/types"
)

const (
	// dirName is the application data directory created inside the user's home.
	dirName = ".itt_ocr_client"

	// fileMode is deliberately owner-only: config.json stores the API key.
	fileMode os.FileMode = 0o600
	// dirMode is owner-only for the same reason.
	dirMode os.FileMode = 0o700
)

// mu guards all reads and writes of files in the application data directory.
var mu sync.RWMutex

// Dir returns the application data directory, creating it when absent.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	dir := filepath.Join(home, dirName)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return "", fmt.Errorf("create config directory %s: %w", dir, err)
	}
	return dir, nil
}

func configPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func historyPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}

// DefaultConfig returns the configuration used on first launch and as the
// fallback for any field left empty in a partially written config file.
func DefaultConfig() types.Config {
	return types.Config{
		SessionAccount:      "default_user",
		APIKey:              "",
		BaseURL:             "https://ai.rupic.studio/v1",
		ModelName:           "gemini-3.5-flash-lite",
		AutoExtract:         true,
		DefaultOutputMode:   string(types.OutputModeDocument),
		Quality:             string(types.QualityStandard),
		ReleasesRepo:        "its-Sohan/itt-ocr-release",
		CheckUpdatesStartup: true,
		UsageStats:          types.UsageStats{},
	}
}

// withDefaults backfills any empty or invalid field on cfg from DefaultConfig
// so a hand-edited or older config file can never produce an unusable state.
func withDefaults(cfg types.Config) types.Config {
	def := DefaultConfig()
	if cfg.SessionAccount == "" {
		cfg.SessionAccount = def.SessionAccount
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = def.BaseURL
	}
	if cfg.ModelName == "" {
		cfg.ModelName = def.ModelName
	}
	if !types.IsValidOutputMode(cfg.DefaultOutputMode) {
		cfg.DefaultOutputMode = def.DefaultOutputMode
	}
	if !types.IsValidQuality(cfg.Quality) {
		cfg.Quality = def.Quality
	}
	if cfg.ReleasesRepo == "" {
		cfg.ReleasesRepo = def.ReleasesRepo
	}
	return cfg
}

// writeJSONAtomic serialises v to path via a temporary file in the same
// directory followed by a rename, so a crash or full disk can never leave a
// truncated config behind.
func writeJSONAtomic(path string, v any, mode os.FileMode) error {
	name := filepath.Base(path)

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), name+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file for %s: %w", name, err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup; a successful rename makes this a no-op error.
	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp file for %s: %w", name, err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write %s: %w", name, err)
	}
	// fsync before rename so the rename cannot expose an empty file.
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync %s: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file for %s: %w", name, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %s: %w", name, err)
	}
	return nil
}

// loadConfigUnsafe reads the config file. Caller must hold mu (read or write).
func loadConfigUnsafe() (types.Config, error) {
	def := DefaultConfig()

	path, err := configPath()
	if err != nil {
		return def, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// First launch: report defaults. Persisting is left to SaveConfig so
			// that a read never mutates disk, which keeps LoadConfig safe to
			// call while holding only a read lock.
			return def, nil
		}
		return def, fmt.Errorf("read config: %w", err)
	}

	var loaded types.Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		// A corrupt config must not brick the app: fall back to defaults and
		// surface the problem to the caller.
		return def, fmt.Errorf("parse config (falling back to defaults): %w", err)
	}

	return withDefaults(loaded), nil
}

// saveConfigUnsafe writes the config file. Caller must hold mu for writing.
func saveConfigUnsafe(cfg types.Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	return writeJSONAtomic(path, cfg, fileMode)
}

// LoadConfig reads the persisted configuration, backfilling defaults for any
// missing field. A missing file yields the default configuration and no error.
func LoadConfig() (types.Config, error) {
	mu.RLock()
	defer mu.RUnlock()
	return loadConfigUnsafe()
}

// SaveConfig persists cfg, normalising empty fields to their defaults.
//
// UsageStats are deliberately re-read from disk rather than taken from cfg: the
// settings UI round-trips the whole config object, so trusting its copy of the
// counters would roll back any extraction that finished while the dialog was
// open.
func SaveConfig(cfg types.Config) error {
	mu.Lock()
	defer mu.Unlock()

	existing, _ := loadConfigUnsafe()
	cfg = withDefaults(cfg)
	cfg.UsageStats = existing.UsageStats
	return saveConfigUnsafe(cfg)
}

// UsageStatsSnapshot returns just the usage counters.
func UsageStatsSnapshot() (types.UsageStats, error) {
	cfg, err := LoadConfig()
	return cfg.UsageStats, err
}

// RecordIngest increments the counter of documents added to the queue.
// Ingestion is not an extraction run, so no run or character counter moves.
func RecordIngest(count int) (types.UsageStats, error) {
	if count <= 0 {
		return UsageStatsSnapshot()
	}

	mu.Lock()
	defer mu.Unlock()

	cfg, _ := loadConfigUnsafe()
	cfg.UsageStats.TotalScannedOrUploaded += count
	if err := saveConfigUnsafe(cfg); err != nil {
		return cfg.UsageStats, err
	}
	return cfg.UsageStats, nil
}

// RecordExtraction increments the counters for exactly one completed
// extraction attempt. chars is added to the character total only on success.
func RecordExtraction(chars int, success bool) (types.UsageStats, error) {
	mu.Lock()
	defer mu.Unlock()

	cfg, _ := loadConfigUnsafe()
	stats := &cfg.UsageStats

	stats.TotalProcessed++
	if success {
		stats.SuccessfulRuns++
		if chars > 0 {
			stats.TotalCharacters += chars
		}
	} else {
		stats.FailedRuns++
	}

	if err := saveConfigUnsafe(cfg); err != nil {
		return cfg.UsageStats, err
	}
	return cfg.UsageStats, nil
}

// ResetUsageStats zeroes all usage counters.
func ResetUsageStats() (types.UsageStats, error) {
	mu.Lock()
	defer mu.Unlock()

	cfg, _ := loadConfigUnsafe()
	cfg.UsageStats = types.UsageStats{}
	if err := saveConfigUnsafe(cfg); err != nil {
		return cfg.UsageStats, err
	}
	return cfg.UsageStats, nil
}

// UpdateUsageStats is kept for compatibility with earlier callers that combine
// ingest and extraction accounting into one call.
func UpdateUsageStats(chars int, success bool, scannedOrUploaded int) (types.UsageStats, error) {
	if scannedOrUploaded > 0 {
		if _, err := RecordIngest(scannedOrUploaded); err != nil {
			return UsageStatsSnapshot()
		}
	}
	if chars > 0 || !success {
		return RecordExtraction(chars, success)
	}
	return UsageStatsSnapshot()
}

// LoadHistory reads the persisted queue. A missing or corrupt history file
// yields an empty queue rather than an error, since history is expendable.
func LoadHistory() ([]types.QueueItem, error) {
	mu.RLock()
	defer mu.RUnlock()

	path, err := historyPath()
	if err != nil {
		return []types.QueueItem{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.QueueItem{}, nil
		}
		return []types.QueueItem{}, fmt.Errorf("read history: %w", err)
	}

	var items []types.QueueItem
	if err := json.Unmarshal(data, &items); err != nil {
		return []types.QueueItem{}, nil
	}
	if items == nil {
		items = []types.QueueItem{}
	}
	return items, nil
}

// SaveHistory persists the queue items.
func SaveHistory(items []types.QueueItem) error {
	mu.Lock()
	defer mu.Unlock()

	path, err := historyPath()
	if err != nil {
		return err
	}
	if items == nil {
		items = []types.QueueItem{}
	}
	return writeJSONAtomic(path, items, fileMode)
}
