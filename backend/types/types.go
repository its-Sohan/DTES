// Package types holds the data structures shared between the Go backend and the
// TypeScript frontend. Every exported field carries a snake_case JSON tag,
// which is the contract the generated Wails bindings mirror.
package types

import "time"

// ItemStatus is the lifecycle state of a queue item.
type ItemStatus string

const (
	StatusReady      ItemStatus = "Ready"
	StatusProcessing ItemStatus = "Processing"
	StatusDone       ItemStatus = "Done"
	StatusFailed     ItemStatus = "Failed"
)

// ItemSource records how a document entered the queue.
type ItemSource string

const (
	SourceUpload    ItemSource = "upload"
	SourceScanner   ItemSource = "scanner"
	SourceClipboard ItemSource = "clipboard"
)

// OutputMode selects the prompt family used for extraction, which determines
// how the model is asked to shape its output.
type OutputMode string

const (
	OutputModeDocument    OutputMode = "document"
	OutputModeSpreadsheet OutputMode = "spreadsheet"
	OutputModeKeyValue    OutputMode = "key_value"
	OutputModeRawText     OutputMode = "raw_text"
)

// Quality selects the speed/accuracy tradeoff for extraction.
type Quality string

const (
	QualityStandard Quality = "standard"
	QualityHigh     Quality = "high"
	QualityDocument Quality = "document"
)

// IsValidOutputMode reports whether s names a supported output mode.
func IsValidOutputMode(s string) bool {
	switch OutputMode(s) {
	case OutputModeDocument, OutputModeSpreadsheet, OutputModeKeyValue, OutputModeRawText:
		return true
	default:
		return false
	}
}

// IsValidQuality reports whether s names a supported quality tier.
func IsValidQuality(s string) bool {
	switch Quality(s) {
	case QualityStandard, QualityHigh, QualityDocument:
		return true
	default:
		return false
	}
}

// QueueItem represents a single document in the OCR queue.
type QueueItem struct {
	ID          string `json:"id"`
	FilePath    string `json:"file_path"`
	FileName    string `json:"file_name"`
	FileSizeStr string `json:"file_size_str"`
	// FileSizeBytes is the raw size, kept alongside the display string so the
	// frontend can sort and aggregate without re-parsing "1.4 MB".
	FileSizeBytes int64         `json:"file_size_bytes"`
	Status        ItemStatus    `json:"status"`
	ExtractedText string        `json:"extracted_text"`
	ErrorMessage  string        `json:"error_message"`
	Source        ItemSource    `json:"source"`
	OutputMode    string        `json:"output_mode"`
	BlockBoxes    []BoundingBox `json:"block_boxes,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

// BoundingBox is a text block's location in the source image, with all
// coordinates normalised to the 0..1000 range so they are resolution
// independent. The origin is the top-left corner.
type BoundingBox struct {
	Index int `json:"index"`
	YMin  int `json:"ymin"`
	XMin  int `json:"xmin"`
	YMax  int `json:"ymax"`
	XMax  int `json:"xmax"`
}

// UsageStats holds purely local, never-transmitted counters.
type UsageStats struct {
	// TotalScannedOrUploaded counts documents added to the queue.
	TotalScannedOrUploaded int `json:"total_scanned_or_uploaded"`
	// TotalProcessed counts finished extraction attempts (success or failure).
	TotalProcessed int `json:"total_processed"`
	// TotalCharacters accumulates extracted characters from successful runs.
	TotalCharacters int `json:"total_characters_extracted"`
	SuccessfulRuns  int `json:"successful_runs"`
	FailedRuns      int `json:"failed_runs"`
}

// Config is the persisted user configuration.
type Config struct {
	SessionAccount string `json:"session_account"`
	// APIKey is stored in a 0600 file in the user's home directory and is
	// never included in diagnostic bundles.
	APIKey              string     `json:"api_key"`
	BaseURL             string     `json:"base_url"`
	ModelName           string     `json:"model_name"`
	DocumentModelName   string     `json:"document_model_name,omitempty"`
	AutoExtract         bool       `json:"auto_extract"`
	DefaultOutputMode   string     `json:"default_output_mode"`
	Quality             string     `json:"quality"`
	ReleasesRepo        string     `json:"releases_repo"`
	CheckUpdatesStartup bool       `json:"check_updates_on_startup"`
	UsageStats          UsageStats `json:"usage_stats"`
}

// InvoiceValidationResult is the outcome of the local arithmetic audit of a
// tabular invoice. It is computed entirely offline.
type InvoiceValidationResult struct {
	Matched    bool    `json:"matched"`
	Total      float64 `json:"total"`
	Calculated float64 `json:"calculated"`
	Difference float64 `json:"difference"`
	ItemsCount int     `json:"items_count"`
}

// DocumentPreview carries an image to the webview as a data URL, which avoids
// exposing the filesystem through the asset server.
type DocumentPreview struct {
	DataURL  string `json:"data_url"`
	MimeType string `json:"mime_type"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}
