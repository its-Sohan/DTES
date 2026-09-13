package types

import "time"

// QueueItem represents a document in the OCR queue
type QueueItem struct {
	ID            string        `json:"id"`
	FilePath      string        `json:"file_path"`
	FileName      string        `json:"file_name"`
	FileSizeStr   string        `json:"file_size_str"`
	Status        string        `json:"status"` // "Ready", "Processing", "Done", "Failed"
	ExtractedText string        `json:"extracted_text"`
	ErrorMessage  string        `json:"error_message"`
	Source        string        `json:"source"`      // "upload", "scanner", "clipboard"
	OutputMode    string        `json:"output_mode"` // "document", "spreadsheet", "key_value", "raw_text"
	BlockBoxes    []BoundingBox `json:"block_boxes,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

// BoundingBox represents a 2D bounding box normalized between 0 and 1000
type BoundingBox struct {
	Index int `json:"index"`
	YMin  int `json:"ymin"`
	XMin  int `json:"xmin"`
	YMax  int `json:"ymax"`
	XMax  int `json:"xmax"`
}

// UsageStats stores local analytics
type UsageStats struct {
	TotalScannedOrUploaded int `json:"total_scanned_or_uploaded"`
	TotalProcessed         int `json:"total_processed"`
	TotalCharacters        int `json:"total_characters_extracted"`
	SuccessfulRuns         int `json:"successful_runs"`
	FailedRuns             int `json:"failed_runs"`
}

// Config stores application configuration
type Config struct {
	SessionAccount      string     `json:"session_account"`
	APIKey              string     `json:"api_key"`
	BaseURL             string     `json:"base_url"`
	ModelName           string     `json:"model_name"`
	AutoExtract         bool       `json:"auto_extract"`
	DefaultOutputMode   string     `json:"default_output_mode"`
	Quality             string     `json:"quality"` // "standard" or "high"
	ReleasesRepo        string     `json:"releases_repo"`
	CheckUpdatesStartup bool       `json:"check_updates_on_startup"`
	UsageStats          UsageStats `json:"usage_stats"`
}

// InvoiceValidationResult holds arithmetic calculation audit of tabular invoices
type InvoiceValidationResult struct {
	Matched    bool    `json:"matched"`
	Total      float64 `json:"total"`
	Calculated float64 `json:"calculated"`
	Difference float64 `json:"difference"`
	ItemsCount int     `json:"items_count"`
}
