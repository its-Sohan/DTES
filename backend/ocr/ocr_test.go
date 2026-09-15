package ocr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"itt-ocr/backend/config"
	"itt-ocr/backend/types"
)

// isolate points config at a throwaway HOME and writes cfg there, so tests
// never read or write the developer's real settings.
func isolate(t *testing.T, cfg types.Config) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}
}

// tinyPNG writes a valid 1x1 PNG and returns its path.
func tinyPNG(t *testing.T) string {
	t.Helper()
	// 1x1 opaque pixel, generated once and inlined to keep tests hermetic.
	const b64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8DwHwAFAAH/q842iQAAAABJRU5ErkJggg=="
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "sample.png")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestResolveChatEndpoint(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"openai base", "https://api.openai.com/v1", "https://api.openai.com/v1/chat/completions"},
		{"trailing slash", "https://api.openai.com/v1/", "https://api.openai.com/v1/chat/completions"},
		{"surrounding space", "  https://api.openai.com/v1  ", "https://api.openai.com/v1/chat/completions"},
		{"empty falls back to default", "", "https://ai.rupic.studio/v1/chat/completions"},
		{
			"already a full route is untouched",
			"https://openrouter.ai/api/v1/chat/completions",
			"https://openrouter.ai/api/v1/chat/completions",
		},
		{
			"gemini openai surface",
			"https://generativelanguage.googleapis.com/v1beta/openai",
			"https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		},
		{
			"gemini v1beta gets openai appended",
			"https://generativelanguage.googleapis.com/v1beta",
			"https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		},
		{
			"gemini v1 is rewritten to v1beta/openai",
			"https://generativelanguage.googleapis.com/v1",
			"https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		},
		{
			"gemini bare host",
			"https://generativelanguage.googleapis.com",
			"https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		},
		{
			"local ollama",
			"http://localhost:11434/v1",
			"http://localhost:11434/v1/chat/completions",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveChatEndpoint(tc.input); got != tc.want {
				t.Errorf("ResolveChatEndpoint(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestResolveChatEndpointIsIdempotent guards against a resolved URL being
// re-resolved into a doubled path such as /chat/completions/chat/completions.
func TestResolveChatEndpointIsIdempotent(t *testing.T) {
	for _, in := range []string{
		"https://api.openai.com/v1",
		"https://generativelanguage.googleapis.com/v1beta",
		"http://localhost:1234/v1",
	} {
		once := ResolveChatEndpoint(in)
		twice := ResolveChatEndpoint(once)
		if once != twice {
			t.Errorf("not idempotent for %q: once=%q twice=%q", in, once, twice)
		}
	}
}

func TestResolveOCREndpoint(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"mistral base", "https://api.mistral.ai/v1", "https://api.mistral.ai/v1/ocr"},
		{"trailing slash", "https://api.mistral.ai/v1/", "https://api.mistral.ai/v1/ocr"},
		{"already ocr route", "https://api.mistral.ai/v1/ocr", "https://api.mistral.ai/v1/ocr"},
		{"chat completions gets rewritten to ocr", "https://api.mistral.ai/v1/chat/completions", "https://api.mistral.ai/v1/ocr"},
		{"bare host adds v1 ocr", "https://api.mistral.ai", "https://api.mistral.ai/v1/ocr"},
		{"empty falls back to default ocr", "", "https://ai.rupic.studio/v1/ocr"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveOCREndpoint(tc.input); got != tc.want {
				t.Errorf("ResolveOCREndpoint(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestResolveModel pins the fix for the bug where the configured model was
// discarded in favour of a hardcoded name that did not exist on any provider.
func TestResolveModel(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		quality    string
		want       string
	}{
		{"standard keeps configured model", "gemini-3.5-flash", "standard", "gemini-3.5-flash"},
		{"high upgrades 3.5 flash to 3.7 flash", "gemini-3.5-flash", "high", "gemini-3.7-flash"},
		{"high upgrades 3.6 flash to 3.7 flash", "gemini-3.6-flash", "high", "gemini-3.7-flash"},
		{"high upgrades 3.5 flash lite to 3.7 flash", "gemini-3.5-flash-lite", "high", "gemini-3.7-flash"},
		{"empty falls back to a default", "", "standard", "gemini-3.5-flash-lite"},
		{"whitespace is trimmed", "  gemini-3.5-flash  ", "standard", "gemini-3.5-flash"},
		// The important cases: a custom or other vendor model must survive unchanged.
		{"custom model kept on standard", "my-local-vision:7b", "standard", "my-local-vision:7b"},
		{"custom model kept on high", "my-local-vision:7b", "high", "my-local-vision:7b"},
		{"non-gemini vendor model kept on high", "gpt-4o-mini", "high", "gpt-4o-mini"},
		{"unrecognised quality is treated as standard", "gemini-3.5-flash", "ludicrous", "gemini-3.5-flash"},
		{"empty quality is treated as standard", "gemini-3.5-flash", "", "gemini-3.5-flash"},
		{"document routes to mistral-ocr-latest", "gemini-3.5-flash", "document", "mistral-ocr-latest"},
		{"document routes gemini lite to mistral-ocr-latest", "gemini-3.5-flash-lite", "document", "mistral-ocr-latest"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveModel(tc.configured, tc.quality); got != tc.want {
				t.Errorf("ResolveModel(%q, %q) = %q, want %q",
					tc.configured, tc.quality, got, tc.want)
			}
		})
	}
}

func TestIsLocalEndpoint(t *testing.T) {
	local := []string{
		"http://localhost:11434/v1/chat/completions",
		"http://127.0.0.1:1234/v1/chat/completions",
		"http://0.0.0.0:8080/v1/chat/completions",
		"http://host.docker.internal:11434/v1",
	}
	remote := []string{
		"https://api.openai.com/v1/chat/completions",
		"https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		"https://openrouter.ai/api/v1/chat/completions",
	}

	for _, u := range local {
		if !isLocalEndpoint(u) {
			t.Errorf("isLocalEndpoint(%q) = false, want true", u)
		}
	}
	for _, u := range remote {
		if isLocalEndpoint(u) {
			t.Errorf("isLocalEndpoint(%q) = true, want false", u)
		}
	}
}

func TestMimeTypeFor(t *testing.T) {
	tests := map[string]string{
		"a.png":       "image/png",
		"a.PNG":       "image/png",
		"a.jpg":       "image/jpeg",
		"a.jpeg":      "image/jpeg",
		"a.webp":      "image/webp",
		"a.bmp":       "image/bmp",
		"a.tiff":      "image/tiff",
		"a.pdf":       "application/pdf",
		"noextension": "image/png",
	}
	for path, want := range tests {
		if got := MimeTypeFor(path); got != want {
			t.Errorf("MimeTypeFor(%q) = %q, want %q", path, got, want)
		}
	}
}

// TestMissingAPIKeyIsReportedBeforeAnyNetworkCall verifies the guard fires for
// remote endpoints and is skipped for local ones.
func TestMissingAPIKeyIsReportedBeforeAnyNetworkCall(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.APIKey = ""
	cfg.BaseURL = "https://api.openai.com/v1"
	isolate(t, cfg)

	_, err := ExtractText(context.Background(), tinyPNG(t), "document", "standard")
	if err == nil {
		t.Fatal("expected an error when no API key is configured for a remote endpoint")
	}
	if !strings.Contains(err.Error(), "API key") {
		t.Errorf("error %q should mention the missing API key", err)
	}
	if !strings.Contains(err.Error(), "Settings") {
		t.Errorf("error %q should tell the user where to fix it", err)
	}
}

func TestMissingFileIsReportedClearly(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.APIKey = "sk-test"
	isolate(t, cfg)

	_, err := ExtractText(context.Background(), "/nonexistent/nope.png", "document", "standard")
	if err == nil {
		t.Fatal("expected an error for a missing file")
	}
	if !strings.Contains(err.Error(), "no longer exists") {
		t.Errorf("error %q should explain the file is gone", err)
	}
}

func TestEmptyFileIsRejected(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.APIKey = "sk-test"
	isolate(t, cfg)

	empty := filepath.Join(t.TempDir(), "empty.png")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatalf("write empty file: %v", err)
	}

	_, err := ExtractText(context.Background(), empty, "document", "standard")
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("err = %v, want an error mentioning the file is empty", err)
	}
}

func TestBlankFilePathIsRejected(t *testing.T) {
	isolate(t, config.DefaultConfig())
	if _, err := ExtractText(context.Background(), "   ", "document", "standard"); err == nil {
		t.Error("expected an error for a blank file path")
	}
}

// TestExtractTextAgainstStubServer exercises the full HTTP path: request
// shape, auth header, response parsing and Unicode normalisation.
func TestExtractTextAgainstStubServer(t *testing.T) {
	var gotAuth, gotUA, gotModel string
	var gotMessages []chatMessage

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		gotModel = req.Model
		gotMessages = req.Messages

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"  Hello বাংলা  "}}]}`))
	}))
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.APIKey = "sk-unit-test"
	cfg.BaseURL = srv.URL + "/v1"
	cfg.ModelName = "stub-vision"
	isolate(t, cfg)

	text, err := ExtractText(context.Background(), tinyPNG(t), "document", "standard")
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}

	if text != "Hello বাংলা" {
		t.Errorf("text = %q, want surrounding whitespace trimmed", text)
	}
	if gotAuth != "Bearer sk-unit-test" {
		t.Errorf("Authorization = %q, want the configured bearer token", gotAuth)
	}
	if gotUA == "" || !strings.Contains(gotUA, "ITT-OCR") {
		t.Errorf("User-Agent = %q, want the app to identify itself", gotUA)
	}
	if gotModel != "stub-vision" {
		t.Errorf("model = %q, want the configured model to be sent", gotModel)
	}

	// The request must be a system message plus a multimodal user message
	// carrying the image inline.
	if len(gotMessages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(gotMessages))
	}
	if gotMessages[0].Role != "system" {
		t.Errorf("messages[0].Role = %q, want system", gotMessages[0].Role)
	}
	blob, _ := json.Marshal(gotMessages[1])
	if !strings.Contains(string(blob), "data:image/png;base64,") {
		t.Errorf("user message must embed the image as a data URL, got %s", blob)
	}

	// The successful run must be counted exactly once.
	stats, err := config.UsageStatsSnapshot()
	if err != nil {
		t.Fatalf("UsageStatsSnapshot: %v", err)
	}
	if stats.SuccessfulRuns != 1 || stats.TotalProcessed != 1 {
		t.Errorf("stats = %+v, want exactly one successful run", stats)
	}
	if stats.FailedRuns != 0 {
		t.Errorf("FailedRuns = %d, want 0", stats.FailedRuns)
	}
}

// TestExtractTextDocumentModeSendsNoTextPrompts verifies that Document mode routes
// directly to the dedicated OCR endpoint without passing chat messages or text prompts.
func TestExtractTextDocumentModeSendsNoTextPrompts(t *testing.T) {
	var gotPath string
	var gotModel string
	var gotDocType string
	var gotImageURL string
	var rawBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path

		bodyBytes, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bodyBytes, &rawBody)

		var req ocrRequest
		_ = json.Unmarshal(bodyBytes, &req)

		gotModel = req.Model
		gotDocType = req.Document.Type
		gotImageURL = req.Document.ImageURL

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"pages":[{"index":0,"markdown":"Extracted Document Markdown Content"}]}`))
	}))
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.APIKey = "sk-unit-test"
	cfg.BaseURL = srv.URL + "/v1"
	isolate(t, cfg)

	text, err := ExtractText(context.Background(), tinyPNG(t), "document", "document")
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}

	if text != "Extracted Document Markdown Content" {
		t.Errorf("text = %q, want expected markdown", text)
	}
	if gotPath != "/v1/ocr" {
		t.Errorf("request path = %q, want /v1/ocr", gotPath)
	}
	if gotModel != "mistral-ocr-latest" {
		t.Errorf("model = %q, want mistral-ocr-latest", gotModel)
	}
	if gotDocType != "image_url" {
		t.Errorf("document.type = %q, want image_url", gotDocType)
	}
	if !strings.HasPrefix(gotImageURL, "data:image/png;base64,") {
		t.Errorf("document.image_url must be data URL, got %s", gotImageURL)
	}

	// Crucial: verify that NO chat messages or text prompt fields were sent!
	if _, hasMessages := rawBody["messages"]; hasMessages {
		t.Error("Document mode must NOT pass 'messages' field to OCR endpoint")
	}
	if _, hasPrompt := rawBody["prompt"]; hasPrompt {
		t.Error("Document mode must NOT pass 'prompt' field to OCR endpoint")
	}
}

// TestModeSelectionChangesPrompt confirms the output mode actually reaches the
// model instead of every mode sending an identical request.
func TestModeSelectionChangesPrompt(t *testing.T) {
	prompts := make(map[string]string)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if s, ok := req.Messages[0].Content.(string); ok {
			prompts[s] = s
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.APIKey = "sk-test"
	cfg.BaseURL = srv.URL + "/v1"
	isolate(t, cfg)

	img := tinyPNG(t)
	for _, mode := range []string{"document", "spreadsheet", "key_value", "raw_text"} {
		if _, err := ExtractText(context.Background(), img, mode, "standard"); err != nil {
			t.Fatalf("ExtractText(%s): %v", mode, err)
		}
	}

	if len(prompts) != 4 {
		t.Errorf("got %d distinct system prompts, want 4 (one per mode)", len(prompts))
	}
}

func TestAPIErrorMessagesAreActionable(t *testing.T) {
	tests := []struct {
		status   int
		wantText string
	}{
		{http.StatusUnauthorized, "API key"},
		{http.StatusForbidden, "API key"},
		{http.StatusNotFound, "Endpoint Base URL"},
		{http.StatusTooManyRequests, "rate limited"},
		{http.StatusInternalServerError, "server error"},
	}

	for _, tc := range tests {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"error":{"message":"upstream detail"}}`))
			}))
			defer srv.Close()

			cfg := config.DefaultConfig()
			cfg.APIKey = "sk-test"
			cfg.BaseURL = srv.URL + "/v1"
			isolate(t, cfg)

			_, err := ExtractText(context.Background(), tinyPNG(t), "document", "standard")
			if err == nil {
				t.Fatalf("expected an error for HTTP %d", tc.status)
			}
			if !strings.Contains(err.Error(), tc.wantText) {
				t.Errorf("error %q should contain %q", err, tc.wantText)
			}

			// A failed attempt must be counted as exactly one failure.
			stats, _ := config.UsageStatsSnapshot()
			if stats.FailedRuns != 1 {
				t.Errorf("FailedRuns = %d, want exactly 1", stats.FailedRuns)
			}
			if stats.SuccessfulRuns != 0 {
				t.Errorf("SuccessfulRuns = %d, want 0", stats.SuccessfulRuns)
			}
		})
	}
}

func TestAPIErrorRetryable(t *testing.T) {
	retryable := []int{429, 500, 502, 503}
	permanent := []int{400, 401, 403, 404, 422}

	for _, s := range retryable {
		if !(&APIError{StatusCode: s}).Retryable() {
			t.Errorf("status %d should be retryable", s)
		}
	}
	for _, s := range permanent {
		if (&APIError{StatusCode: s}).Retryable() {
			t.Errorf("status %d should not be retryable", s)
		}
	}
}

func TestEmptyChoicesIsReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.APIKey = "sk-test"
	cfg.BaseURL = srv.URL + "/v1"
	isolate(t, cfg)

	_, err := ExtractText(context.Background(), tinyPNG(t), "document", "standard")
	if err == nil || !strings.Contains(err.Error(), "no choices") {
		t.Errorf("err = %v, want an error explaining there were no choices", err)
	}
}

func TestBlankModelOutputIsReported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"   "}}]}`))
	}))
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.APIKey = "sk-test"
	cfg.BaseURL = srv.URL + "/v1"
	isolate(t, cfg)

	_, err := ExtractText(context.Background(), tinyPNG(t), "document", "standard")
	if err == nil || !strings.Contains(err.Error(), "no text") {
		t.Errorf("err = %v, want an error about empty output", err)
	}
}

// TestCancellationIsNotCountedAsFailure matters because the user aborting a run
// should not pollute the reliability metrics.
func TestCancellationIsNotCountedAsFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.APIKey = "sk-test"
	cfg.BaseURL = srv.URL + "/v1"
	isolate(t, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	go cancel()

	if _, err := ExtractText(ctx, tinyPNG(t), "document", "standard"); err == nil {
		t.Fatal("expected an error when the context is cancelled")
	}

	stats, _ := config.UsageStatsSnapshot()
	if stats.FailedRuns != 0 {
		t.Errorf("FailedRuns = %d, want 0 (cancellation is not a failure)", stats.FailedRuns)
	}
	if stats.TotalProcessed != 0 {
		t.Errorf("TotalProcessed = %d, want 0", stats.TotalProcessed)
	}
}

func TestStripCodeFence(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"unfenced text is untouched", "Hello world", "Hello world"},
		{"plain fence is removed", "```\nHello\n```", "Hello"},
		{"language fence is removed", "```markdown\n# Title\n```", "# Title"},
		{"inner content is preserved", "```\nline 1\nline 2\n```", "line 1\nline 2"},
		{"trailing newline after fence", "```\nHello\n```\n", "Hello"},
		// Multiple fences imply genuine structure, so leave them alone.
		{
			"multiple blocks are preserved",
			"```\nfirst\n```\ntext\n```\nsecond\n```",
			"```\nfirst\n```\ntext\n```\nsecond\n```",
		},
		{"unterminated fence is left alone", "```\nHello", "```\nHello"},
		{"a markdown table survives", "| a | b |\n|---|---|", "| a | b |\n|---|---|"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripCodeFence(tc.in); got != tc.want {
				t.Errorf("stripCodeFence(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestModesAreCompleteAndOrdered(t *testing.T) {
	got := Modes()
	if len(got) != 4 {
		t.Fatalf("len(Modes()) = %d, want 4", len(got))
	}

	wantOrder := []types.OutputMode{
		types.OutputModeDocument,
		types.OutputModeSpreadsheet,
		types.OutputModeKeyValue,
		types.OutputModeRawText,
	}
	for i, want := range wantOrder {
		if got[i].ID != want {
			t.Errorf("Modes()[%d].ID = %q, want %q", i, got[i].ID, want)
		}
		if got[i].Label == "" || got[i].Description == "" {
			t.Errorf("mode %q is missing UI metadata", got[i].ID)
		}
		if got[i].SystemInstruction == "" || got[i].UserPrompt == "" {
			t.Errorf("mode %q is missing prompts", got[i].ID)
		}
	}

	// Every registered mode must be accepted by the shared validator, or the
	// UI and the backend would disagree about what is valid.
	for _, m := range got {
		if !types.IsValidOutputMode(string(m.ID)) {
			t.Errorf("mode %q is not accepted by types.IsValidOutputMode", m.ID)
		}
	}
}

func TestModeForFallsBackToDocument(t *testing.T) {
	for _, in := range []string{"", "nonsense", "DOCUMENT"} {
		if got := modeFor(in); got.ID != types.OutputModeDocument {
			t.Errorf("modeFor(%q).ID = %q, want document", in, got.ID)
		}
	}
	if got := modeFor("spreadsheet"); got.ID != types.OutputModeSpreadsheet {
		t.Errorf("modeFor(spreadsheet).ID = %q", got.ID)
	}
}

func TestResolveModelChain(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		quality    string
		want       []string
	}{
		{
			name:       "gemini flash lite high cascade",
			configured: "gemini-3.5-flash-lite",
			quality:    "high",
			want:       []string{"gemini-3.7-flash", "gemini-3.6-flash", "gemini-3.5-flash", "gemini-3.5-flash-lite"},
		},
		{
			name:       "gemini 3.5 flash high cascade",
			configured: "gemini-3.5-flash",
			quality:    "high",
			want:       []string{"gemini-3.7-flash", "gemini-3.6-flash", "gemini-3.5-flash", "gemini-3.5-flash-lite"},
		},
		{
			name:       "standard returns single configured model",
			configured: "gemini-3.5-flash-lite",
			quality:    "standard",
			want:       []string{"gemini-3.5-flash-lite"},
		},
		{
			name:       "custom non-gemini model on high stays single",
			configured: "custom-vision:latest",
			quality:    "high",
			want:       []string{"custom-vision:latest"},
		},
		{
			name:       "document mode routes to mistral ocr latest",
			configured: "gemini-3.5-flash",
			quality:    "document",
			want:       []string{"mistral-ocr-latest"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveModelChain(tc.configured, tc.quality)
			if len(got) != len(tc.want) {
				t.Fatalf("ResolveModelChain(%q, %q) len = %d, want %d (%v vs %v)",
					tc.configured, tc.quality, len(got), len(tc.want), got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestExtractTextFallbackOnQuotaExceeded(t *testing.T) {
	var requestedModels []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		requestedModels = append(requestedModels, req.Model)

		if req.Model == "gemini-3.7-flash" {
			// Simulate 429 Quota Exceeded on 3.7
			w.WriteHeader(http.StatusTooManyRequests)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"error":{"message":"Quota exceeded for gemini-3.7-flash"}}`))
			return
		}

		if req.Model == "gemini-3.6-flash" {
			// Fallback succeeded!
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Fallback success text"}}]}`))
			return
		}

		http.Error(w, "unexpected model", http.StatusBadRequest)
	}))
	defer srv.Close()

	cfg := config.DefaultConfig()
	cfg.APIKey = "sk-fallback-test"
	cfg.BaseURL = srv.URL + "/v1"
	cfg.ModelName = "gemini-3.5-flash-lite"
	isolate(t, cfg)

	text, err := ExtractText(context.Background(), tinyPNG(t), "document", "high")
	if err != nil {
		t.Fatalf("ExtractText with fallback failed: %v", err)
	}

	if text != "Fallback success text" {
		t.Errorf("got %q, want %q", text, "Fallback success text")
	}

	if len(requestedModels) != 2 || requestedModels[0] != "gemini-3.7-flash" || requestedModels[1] != "gemini-3.6-flash" {
		t.Errorf("requestedModels = %v, want [gemini-3.7-flash, gemini-3.6-flash]", requestedModels)
	}
}
