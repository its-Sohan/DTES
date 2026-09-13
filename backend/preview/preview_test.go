package preview

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// onePixelPNG is a valid 1x1 opaque PNG.
const onePixelPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFAAH/q842iQAAAABJRU5ErkJggg=="

func writeFixture(t *testing.T, name, b64 string) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestLoadReturnsDataURLAndDimensions(t *testing.T) {
	got, err := Load(writeFixture(t, "pixel.png", onePixelPNG))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !strings.HasPrefix(got.DataURL, "data:image/png;base64,") {
		t.Errorf("DataURL = %q..., want a PNG data URL", truncate(got.DataURL))
	}
	if got.MimeType != "image/png" {
		t.Errorf("MimeType = %q, want image/png", got.MimeType)
	}
	if got.Width != 1 || got.Height != 1 {
		t.Errorf("dimensions = %dx%d, want 1x1", got.Width, got.Height)
	}
}

// TestLoadDataURLDecodesToTheOriginalBytes is the core contract: the webview
// must receive the exact file, not a re-encoded copy.
func TestLoadDataURLDecodesToTheOriginalBytes(t *testing.T) {
	path := writeFixture(t, "pixel.png", onePixelPNG)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	_, payload, found := strings.Cut(got.DataURL, ",")
	if !found {
		t.Fatal("data URL has no comma separator")
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("decode data URL payload: %v", err)
	}
	if string(decoded) != string(original) {
		t.Error("decoded preview bytes differ from the file on disk")
	}
}

func TestLoadRejectsMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "absent.png"))
	if err == nil {
		t.Fatal("expected an error for a missing file")
	}
	if !strings.Contains(err.Error(), "no longer on disk") {
		t.Errorf("error %q should explain the file is gone", err)
	}
}

func TestLoadRejectsEmptyPath(t *testing.T) {
	if _, err := Load("   "); err == nil {
		t.Error("expected an error for a blank path")
	}
}

func TestLoadRejectsEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.png")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("err = %v, want an error mentioning the file is empty", err)
	}
}

func TestLoadRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	// Give it an image extension so the check under test is the stat, not the ext.
	imgDir := filepath.Join(dir, "looks-like.png")
	if err := os.Mkdir(imgDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	_, err := Load(imgDir)
	if err == nil || !strings.Contains(err.Error(), "directory") {
		t.Errorf("err = %v, want an error mentioning a directory", err)
	}
}

// TestLoadExplainsPDFLimitation matters because the file picker accepts PDFs,
// so users will select one and need to know extraction still works.
func TestLoadExplainsPDFLimitation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doc.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected an error for a PDF")
	}
	if !strings.Contains(err.Error(), "PDF") {
		t.Errorf("error %q should name the format", err)
	}
	if !strings.Contains(err.Error(), "extraction still works") {
		t.Errorf("error %q should reassure the user extraction is unaffected", err)
	}
}

func TestLoadRejectsUnsupportedExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Error("expected an error for an unsupported extension")
	}
}

func TestLoadIsCaseInsensitiveOnExtension(t *testing.T) {
	if _, err := Load(writeFixture(t, "PIXEL.PNG", onePixelPNG)); err != nil {
		t.Errorf("uppercase extension should be accepted: %v", err)
	}
}

// TestLoadUndecodableHeaderStillPreviews: formats the standard library cannot
// inspect must still round-trip to the webview, just without dimensions.
func TestLoadUndecodableHeaderStillPreviews(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.webp")
	if err := os.WriteFile(path, []byte("RIFF????WEBPVP8 not-really"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load should tolerate an uninspectable header: %v", err)
	}
	if !strings.HasPrefix(got.DataURL, "data:image/webp;base64,") {
		t.Errorf("DataURL = %q..., want a webp data URL", truncate(got.DataURL))
	}
	if got.Width != 0 || got.Height != 0 {
		t.Errorf("dimensions = %dx%d, want 0x0 when the header cannot be read", got.Width, got.Height)
	}
}

func TestLoadMimeTypes(t *testing.T) {
	// A valid PNG body under various extensions: only the declared MIME type
	// should change, since that is derived from the extension.
	tests := map[string]string{
		"a.png":  "image/png",
		"a.jpg":  "image/jpeg",
		"a.jpeg": "image/jpeg",
		"a.gif":  "image/gif",
		"a.bmp":  "image/bmp",
		"a.webp": "image/webp",
		"a.tiff": "image/tiff",
	}

	for name, want := range tests {
		got, err := Load(writeFixture(t, name, onePixelPNG))
		if err != nil {
			t.Errorf("Load(%s): %v", name, err)
			continue
		}
		if got.MimeType != want {
			t.Errorf("Load(%s).MimeType = %q, want %q", name, got.MimeType, want)
		}
	}
}

func truncate(s string) string {
	if len(s) > 48 {
		return s[:48]
	}
	return s
}
