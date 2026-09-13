// Package preview loads local document images for display in the webview.
//
// The webview cannot load arbitrary local paths: the Wails asset server only
// serves the embedded frontend bundle, so a src="/home/user/scan.png" simply
// fails. Rather than exposing the filesystem through a custom asset handler,
// images are read here and handed to the frontend as data URLs. That leaves the
// webview with no filesystem reach at all, at the cost of holding one image in
// memory while it is on screen.
package preview

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	// Registered for their side effects: image.DecodeConfig needs a format to be
	// registered before it can report that format's dimensions. Only the
	// standard library decoders are pulled in; formats it cannot inspect still
	// preview fine, just without known dimensions.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"itt-ocr/backend/types"
)

// maxPreviewBytes bounds what will be inlined into the webview. Base64 inflates
// payloads by roughly a third and the string is held in JS memory, so very
// large scans are refused with a clear message instead of freezing the UI.
const maxPreviewBytes = 32 << 20 // 32 MiB

// previewable maps the extensions the preview pane can display to their MIME
// type. PDFs are accepted by the file picker for extraction, but rasterising a
// page needs a PDF engine, so they are reported as unpreviewable.
var previewable = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".gif":  "image/gif",
	".tif":  "image/tiff",
	".tiff": "image/tiff",
}

// Load reads the image at filePath and returns it as a data URL together with
// its pixel dimensions where those can be determined.
//
// Dimensions let the frontend size the preview and place bounding-box overlays
// before the image finishes decoding. A file whose header the standard library
// cannot parse (WebP, BMP, TIFF) still returns a usable data URL with zero
// dimensions, because the webview can render formats Go cannot inspect.
func Load(filePath string) (types.DocumentPreview, error) {
	var out types.DocumentPreview

	if strings.TrimSpace(filePath) == "" {
		return out, fmt.Errorf("no document was supplied")
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == ".pdf" {
		return out, fmt.Errorf("PDF pages cannot be previewed in this version; text extraction still works")
	}
	mimeType, ok := previewable[ext]
	if !ok {
		return out, fmt.Errorf("%q files cannot be previewed", strings.TrimPrefix(ext, "."))
	}

	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return out, fmt.Errorf("this file is no longer on disk: %s", filePath)
		}
		return out, fmt.Errorf("inspect %s: %w", filepath.Base(filePath), err)
	}
	if info.IsDir() {
		return out, fmt.Errorf("%s is a directory, not a document", filePath)
	}
	if info.Size() == 0 {
		return out, fmt.Errorf("%s is empty", filepath.Base(filePath))
	}
	if info.Size() > maxPreviewBytes {
		return out, fmt.Errorf("%s is %.1f MB, which is too large to preview (limit %d MB)",
			filepath.Base(filePath), float64(info.Size())/(1<<20), maxPreviewBytes>>20)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return out, fmt.Errorf("read %s: %w", filepath.Base(filePath), err)
	}

	out.MimeType = mimeType
	out.DataURL = "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)

	// Best-effort: a decode failure must never block the preview itself.
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
		out.Width = cfg.Width
		out.Height = cfg.Height
	}

	return out, nil
}
