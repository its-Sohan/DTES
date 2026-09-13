package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"itt-ocr/backend/bugreport"
	"itt-ocr/backend/clipboard"
	"itt-ocr/backend/config"
	"itt-ocr/backend/ocr"
	"itt-ocr/backend/scanner"
	"itt-ocr/backend/transforms"
	"itt-ocr/backend/types"
	"itt-ocr/backend/updater"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func formatFileSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024.0)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024.0*1024.0))
}

// GetConfig returns the application configuration
func (a *App) GetConfig() (types.Config, error) {
	return config.LoadConfig()
}

// SaveConfig persists updated configuration
func (a *App) SaveConfig(cfg types.Config) error {
	return config.SaveConfig(cfg)
}

// GetUsageStats returns current local metrics
func (a *App) GetUsageStats() types.UsageStats {
	cfg, _ := config.LoadConfig()
	return cfg.UsageStats
}

// LoadHistory returns queue items saved from previous sessions
func (a *App) LoadHistory() ([]types.QueueItem, error) {
	return config.LoadHistory()
}

// SaveHistory persists queue items
func (a *App) SaveHistory(items []types.QueueItem) error {
	return config.SaveHistory(items)
}

// PickFiles opens the native file dialog to select images or documents
func (a *App) PickFiles() ([]types.QueueItem, error) {
	selection, err := wailsRuntime.OpenMultipleFilesDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select document or image",
		Filters: []wailsRuntime.FileFilter{
			{
				DisplayName: "Images & Documents (*.png;*.jpg;*.jpeg;*.webp;*.bmp;*.pdf)",
				Pattern:     "*.png;*.jpg;*.jpeg;*.webp;*.bmp;*.pdf",
			},
		},
	})
	if err != nil {
		return nil, err
	}

	cfg, _ := config.LoadConfig()
	var items []types.QueueItem

	for _, path := range selection {
		fi, err := os.Stat(path)
		sizeStr := "Unknown"
		if err == nil {
			sizeStr = formatFileSize(fi.Size())
		}

		item := types.QueueItem{
			ID:          uuid.New().String()[:8],
			FilePath:    path,
			FileName:    filepath.Base(path),
			FileSizeStr: sizeStr,
			Status:      "Ready",
			Source:      "upload",
			OutputMode:  cfg.DefaultOutputMode,
			CreatedAt:   time.Now(),
		}
		items = append(items, item)
	}

	if len(items) > 0 {
		_, _ = config.UpdateUsageStats(0, true, len(items))
	}

	return items, nil
}

// GetClipboardImage extracts an image from system clipboard and returns an item
func (a *App) GetClipboardImage() (*types.QueueItem, error) {
	path, err := clipboard.GetClipboardImage()
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(path)
	sizeStr := "Unknown"
	if err == nil {
		sizeStr = formatFileSize(fi.Size())
	}

	cfg, _ := config.LoadConfig()
	item := &types.QueueItem{
		ID:          uuid.New().String()[:8],
		FilePath:    path,
		FileName:    filepath.Base(path),
		FileSizeStr: sizeStr,
		Status:      "Ready",
		Source:      "clipboard",
		OutputMode:  cfg.DefaultOutputMode,
		CreatedAt:   time.Now(),
	}

	_, _ = config.UpdateUsageStats(0, true, 1)
	return item, nil
}

// ScanDocument triggers native scanner acquisition
func (a *App) ScanDocument() (*types.QueueItem, error) {
	path, err := scanner.ScanDocument()
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(path)
	sizeStr := "Unknown"
	if err == nil {
		sizeStr = formatFileSize(fi.Size())
	}

	cfg, _ := config.LoadConfig()
	item := &types.QueueItem{
		ID:          uuid.New().String()[:8],
		FilePath:    path,
		FileName:    filepath.Base(path),
		FileSizeStr: sizeStr,
		Status:      "Ready",
		Source:      "scanner",
		OutputMode:  cfg.DefaultOutputMode,
		CreatedAt:   time.Now(),
	}

	_, _ = config.UpdateUsageStats(0, true, 1)
	return item, nil
}

// ExtractText executes the vision LLM transcription
func (a *App) ExtractText(filePath string, mode string, quality string) (string, error) {
	return ocr.ExtractText(filePath, mode, quality)
}

// AlignBlocks detects normalized 2D bounding boxes for text blocks
func (a *App) AlignBlocks(filePath string, blocks []string) ([]types.BoundingBox, error) {
	return ocr.AlignBlocksWithAI(filePath, blocks)
}

// TransformText executes deterministic post-processing transforms
func (a *App) TransformText(text string, transformType string) (string, error) {
	switch transformType {
	case "digits_to_english":
		return transforms.ConvertDigitsToEnglish(text), nil
	case "digits_to_bengali":
		return transforms.ConvertDigitsToBengali(text), nil
	case "unwrap_lines":
		return transforms.UnwrapBrokenLines(text), nil
	case "clean_whitespace":
		return transforms.CleanWhitespaceAndMargins(text), nil
	case "clean_tables":
		return transforms.CleanTableFormatting(text), nil
	default:
		return text, fmt.Errorf("unknown transform type: %s", transformType)
	}
}

// CheckInvoiceMath parses and validates invoice totals locally
func (a *App) CheckInvoiceMath(text string) *types.InvoiceValidationResult {
	return transforms.CheckInvoiceMath(text)
}

// SaveExportFile opens native save dialog and saves exported content
func (a *App) SaveExportFile(defaultFilename string, content string) (string, error) {
	path, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		DefaultFilename: defaultFilename,
		Title:           "Export OCR Document",
	})
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil // user cancelled
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}

	return path, nil
}

// CheckForUpdates checks GitHub Releases for new updates
func (a *App) CheckForUpdates() (updater.UpdateCheckResult, error) {
	return updater.CheckForUpdates()
}

// GenerateBugReport generates a redacted diagnostic report
func (a *App) GenerateBugReport(description string, steps string) (string, error) {
	return bugreport.GenerateBugReport(description, steps)
}
