package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"itt-ocr/backend/types"
)

// withTempHome points the config package at an isolated HOME so tests never
// touch the developer's real ~/.itt_ocr_client.
func withTempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}
	return dir
}

// TestRecordExtractionDoesNotDeadlock guards the regression where a compound
// read-modify-write helper took the write lock and then called LoadConfig,
// which tried to take the read lock on the same non-reentrant RWMutex. That
// hung every OCR run forever.
func TestRecordExtractionDoesNotDeadlock(t *testing.T) {
	withTempHome(t)

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := RecordExtraction(120, true); err != nil {
			t.Errorf("RecordExtraction: %v", err)
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("RecordExtraction deadlocked (did not return within 5s)")
	}
}

func TestRecordIngestDoesNotDeadlock(t *testing.T) {
	withTempHome(t)

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := RecordIngest(3); err != nil {
			t.Errorf("RecordIngest: %v", err)
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("RecordIngest deadlocked (did not return within 5s)")
	}
}

func TestLoadConfigMissingFileReturnsDefaults(t *testing.T) {
	withTempHome(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig on fresh home: %v", err)
	}
	if cfg.BaseURL != DefaultConfig().BaseURL {
		t.Errorf("BaseURL = %q, want default %q", cfg.BaseURL, DefaultConfig().BaseURL)
	}
	if cfg.ModelName != DefaultConfig().ModelName {
		t.Errorf("ModelName = %q, want default %q", cfg.ModelName, DefaultConfig().ModelName)
	}
}

func TestLoadConfigDoesNotCreateFile(t *testing.T) {
	home := withTempHome(t)

	if _, err := LoadConfig(); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	path := filepath.Join(home, dirName, "config.json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("LoadConfig created %s; reads must not mutate disk", path)
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	withTempHome(t)

	want := DefaultConfig()
	want.APIKey = "sk-test-secret"
	want.BaseURL = "https://example.test/v1"
	want.ModelName = "some-vision-model"
	want.AutoExtract = false

	if err := SaveConfig(want); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got.APIKey != want.APIKey {
		t.Errorf("APIKey = %q, want %q", got.APIKey, want.APIKey)
	}
	if got.BaseURL != want.BaseURL {
		t.Errorf("BaseURL = %q, want %q", got.BaseURL, want.BaseURL)
	}
	if got.AutoExtract {
		t.Error("AutoExtract = true, want false")
	}
}

// TestConfigFileIsOwnerOnly matters because config.json stores the API key.
func TestConfigFileIsOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not meaningful on Windows")
	}
	home := withTempHome(t)

	if err := SaveConfig(DefaultConfig()); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	info, err := os.Stat(filepath.Join(home, dirName, "config.json"))
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config.json mode = %#o, want 0600 (it holds the API key)", perm)
	}
}

// TestSaveConfigPreservesUsageStats covers the case where the settings dialog
// is open (holding a stale copy of the counters) while an extraction finishes.
// Saving settings must not roll the counters back.
func TestSaveConfigPreservesUsageStats(t *testing.T) {
	withTempHome(t)

	if _, err := RecordExtraction(500, true); err != nil {
		t.Fatalf("RecordExtraction: %v", err)
	}

	stale := DefaultConfig() // UsageStats zeroed, as a stale UI copy would be
	stale.ModelName = "changed-by-user"
	if err := SaveConfig(stale); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got.ModelName != "changed-by-user" {
		t.Errorf("ModelName = %q, want the user's edit to apply", got.ModelName)
	}
	if got.UsageStats.SuccessfulRuns != 1 {
		t.Errorf("SuccessfulRuns = %d, want 1 (stats must survive a settings save)",
			got.UsageStats.SuccessfulRuns)
	}
	if got.UsageStats.TotalCharacters != 500 {
		t.Errorf("TotalCharacters = %d, want 500", got.UsageStats.TotalCharacters)
	}
}

func TestRecordExtractionAccounting(t *testing.T) {
	withTempHome(t)

	if _, err := RecordExtraction(100, true); err != nil {
		t.Fatalf("RecordExtraction success: %v", err)
	}
	if _, err := RecordExtraction(999, false); err != nil {
		t.Fatalf("RecordExtraction failure: %v", err)
	}

	stats, err := UsageStatsSnapshot()
	if err != nil {
		t.Fatalf("UsageStatsSnapshot: %v", err)
	}
	if stats.TotalProcessed != 2 {
		t.Errorf("TotalProcessed = %d, want 2", stats.TotalProcessed)
	}
	if stats.SuccessfulRuns != 1 {
		t.Errorf("SuccessfulRuns = %d, want 1", stats.SuccessfulRuns)
	}
	if stats.FailedRuns != 1 {
		t.Errorf("FailedRuns = %d, want 1", stats.FailedRuns)
	}
	// A failed run must not contribute characters.
	if stats.TotalCharacters != 100 {
		t.Errorf("TotalCharacters = %d, want 100 (failures add none)", stats.TotalCharacters)
	}
}

// TestRecordIngestIsNotAnExtraction pins the accounting split: adding files to
// the queue must not inflate the processed/success counters.
func TestRecordIngestIsNotAnExtraction(t *testing.T) {
	withTempHome(t)

	if _, err := RecordIngest(4); err != nil {
		t.Fatalf("RecordIngest: %v", err)
	}

	stats, err := UsageStatsSnapshot()
	if err != nil {
		t.Fatalf("UsageStatsSnapshot: %v", err)
	}
	if stats.TotalScannedOrUploaded != 4 {
		t.Errorf("TotalScannedOrUploaded = %d, want 4", stats.TotalScannedOrUploaded)
	}
	if stats.TotalProcessed != 0 {
		t.Errorf("TotalProcessed = %d, want 0 (ingest is not a run)", stats.TotalProcessed)
	}
	if stats.SuccessfulRuns != 0 {
		t.Errorf("SuccessfulRuns = %d, want 0", stats.SuccessfulRuns)
	}
}

func TestCorruptConfigFallsBackToDefaults(t *testing.T) {
	home := withTempHome(t)

	dir := filepath.Join(home, dirName)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{not json"), fileMode); err != nil {
		t.Fatalf("write corrupt config: %v", err)
	}

	cfg, err := LoadConfig()
	if err == nil {
		t.Error("expected an error describing the corrupt config")
	}
	if cfg.BaseURL != DefaultConfig().BaseURL {
		t.Errorf("BaseURL = %q, want usable defaults despite corruption", cfg.BaseURL)
	}
}

// TestPartialConfigGetsDefaults covers upgrades from older config files that
// predate newer fields, and hand-edited files with invalid enum values.
func TestPartialConfigGetsDefaults(t *testing.T) {
	home := withTempHome(t)

	dir := filepath.Join(home, dirName)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	partial := `{"api_key":"sk-abc","quality":"bogus","default_output_mode":""}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(partial), fileMode); err != nil {
		t.Fatalf("write partial config: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.APIKey != "sk-abc" {
		t.Errorf("APIKey = %q, want the value from disk", cfg.APIKey)
	}
	if cfg.Quality != string(types.QualityStandard) {
		t.Errorf("Quality = %q, want invalid value replaced with default", cfg.Quality)
	}
	if cfg.DefaultOutputMode != string(types.OutputModeDocument) {
		t.Errorf("DefaultOutputMode = %q, want default", cfg.DefaultOutputMode)
	}
	if cfg.BaseURL == "" {
		t.Error("BaseURL is empty, want default backfilled")
	}
}

func TestHistoryRoundTrip(t *testing.T) {
	withTempHome(t)

	items := []types.QueueItem{
		{ID: "a1", FileName: "one.png", Status: types.StatusDone, Source: types.SourceUpload},
		{ID: "b2", FileName: "two.jpg", Status: types.StatusReady, Source: types.SourceClipboard},
	}
	if err := SaveHistory(items); err != nil {
		t.Fatalf("SaveHistory: %v", err)
	}

	got, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(history) = %d, want 2", len(got))
	}
	if got[0].ID != "a1" || got[1].FileName != "two.jpg" {
		t.Errorf("history round-trip mismatch: %+v", got)
	}
}

// TestLoadHistoryEmptyIsNonNil matters because a nil slice marshals to JSON
// null, which the frontend cannot call .map() on.
func TestLoadHistoryEmptyIsNonNil(t *testing.T) {
	withTempHome(t)

	got, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if got == nil {
		t.Fatal("LoadHistory returned nil; want empty slice so JSON is [] not null")
	}

	blob, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(blob) != "[]" {
		t.Errorf("marshalled empty history = %s, want []", blob)
	}
}

func TestSaveHistoryNilWritesEmptyArray(t *testing.T) {
	withTempHome(t)

	if err := SaveHistory(nil); err != nil {
		t.Fatalf("SaveHistory(nil): %v", err)
	}
	got, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestResetUsageStats(t *testing.T) {
	withTempHome(t)

	if _, err := RecordExtraction(42, true); err != nil {
		t.Fatalf("RecordExtraction: %v", err)
	}
	stats, err := ResetUsageStats()
	if err != nil {
		t.Fatalf("ResetUsageStats: %v", err)
	}
	if stats != (types.UsageStats{}) {
		t.Errorf("stats = %+v, want zero value", stats)
	}
}

// TestConcurrentAccessIsRaceFree is most valuable under `go test -race`: the
// real app calls these helpers from multiple goroutines via Wails bindings.
func TestConcurrentAccessIsRaceFree(t *testing.T) {
	withTempHome(t)

	const workers = 8
	var wg sync.WaitGroup
	wg.Add(workers * 3)

	for i := 0; i < workers; i++ {
		go func() { defer wg.Done(); _, _ = RecordExtraction(10, true) }()
		go func() { defer wg.Done(); _, _ = RecordIngest(1) }()
		go func() { defer wg.Done(); _, _ = LoadConfig() }()
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("concurrent config access deadlocked")
	}

	stats, err := UsageStatsSnapshot()
	if err != nil {
		t.Fatalf("UsageStatsSnapshot: %v", err)
	}
	if stats.TotalProcessed != workers {
		t.Errorf("TotalProcessed = %d, want %d (no lost updates)", stats.TotalProcessed, workers)
	}
	if stats.TotalScannedOrUploaded != workers {
		t.Errorf("TotalScannedOrUploaded = %d, want %d", stats.TotalScannedOrUploaded, workers)
	}
}

// TestAtomicWriteLeavesNoTempFiles verifies the temp-then-rename write cleans up.
func TestAtomicWriteLeavesNoTempFiles(t *testing.T) {
	home := withTempHome(t)

	for i := 0; i < 3; i++ {
		if err := SaveConfig(DefaultConfig()); err != nil {
			t.Fatalf("SaveConfig: %v", err)
		}
	}

	entries, err := os.ReadDir(filepath.Join(home, dirName))
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			t.Errorf("leftover non-config file %q after atomic writes", e.Name())
		}
	}
}
