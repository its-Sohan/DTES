package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"itt-ocr/backend/types"
	"itt-ocr/backend/version"
)

// maxImageBytes caps the source file size. Images are base64-encoded inline
// into the request body, which inflates them by ~33%; beyond this size the
// request is very likely to be rejected by the provider anyway, and a clear
// local error is far more useful than an opaque HTTP 413.
const maxImageBytes = 20 << 20 // 20 MiB

// requestTimeout bounds a single vision call. Large scans on a slow connection
// legitimately take tens of seconds, so this is generous.
const requestTimeout = 120 * time.Second

// ErrMissingAPIKey is returned when a remote endpoint is configured without
// credentials. It is a sentinel so the UI can offer to open Settings.
var ErrMissingAPIKey = errors.New("no API key configured")

// APIError describes a non-2xx response from the vision provider.
type APIError struct {
	StatusCode int
	Endpoint   string
	Message    string
}

func (e *APIError) Error() string {
	switch {
	case e.StatusCode == http.StatusNotFound:
		return fmt.Sprintf("endpoint not found (404) at %s — check the Endpoint Base URL in Settings", e.Endpoint)
	case e.StatusCode == http.StatusUnauthorized, e.StatusCode == http.StatusForbidden:
		return fmt.Sprintf("authentication failed (%d) — check your API key in Settings", e.StatusCode)
	case e.StatusCode == http.StatusTooManyRequests:
		return fmt.Sprintf("rate limited (429) by the provider — wait a moment and retry: %s", e.Message)
	case e.StatusCode >= 500:
		return fmt.Sprintf("the vision provider reported a server error (%d): %s", e.StatusCode, e.Message)
	case e.Message != "":
		return fmt.Sprintf("vision API error (%d): %s", e.StatusCode, e.Message)
	default:
		return fmt.Sprintf("vision API error (%d)", e.StatusCode)
	}
}

// Retryable reports whether repeating the request could plausibly succeed.
func (e *APIError) Retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= 500
}

// --- Wire format (OpenAI-compatible chat completions) ---

type messageContent struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	ImageURL *imageURLField `json:"image_url,omitempty"`
}

type imageURLField struct {
	URL string `json:"url"`
}

type chatMessage struct {
	Role string `json:"role"`
	// Content is either a plain string (system messages) or a
	// []messageContent (multimodal user messages), per the OpenAI schema.
	Content any `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

// Endpoint holds the resolved connection settings for one vision call.
type Endpoint struct {
	URL    string
	APIKey string
	Model  string
	// base is the normalised configured base URL, retained for error messages
	// and provider-specific header decisions.
	base string
}

// ResolveChatEndpoint normalises a configured base URL into a full
// /chat/completions route.
//
// Google's Generative Language API needs special handling: its
// OpenAI-compatible surface lives under /v1beta/openai, so a bare host or a
// plain /v1 path is rewritten accordingly.
func ResolveChatEndpoint(baseURL string) string {
	clean := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if clean == "" {
		clean = defaultBaseURL
	}

	if strings.HasSuffix(clean, "/chat/completions") {
		return clean
	}

	if strings.Contains(clean, "generativelanguage.googleapis.com") {
		if !strings.HasSuffix(clean, "/openai") {
			// Strip a trailing version segment so we can append the correct one.
			if strings.HasSuffix(clean, "/v1") || strings.HasSuffix(clean, "/v1beta") {
				clean = clean[:strings.LastIndex(clean, "/")]
			}
			clean += "/v1beta/openai"
		}
		return clean + "/chat/completions"
	}

	return clean + "/chat/completions"
}

// isLocalEndpoint reports whether the URL points at the developer's machine, in
// which case a missing API key is expected rather than a misconfiguration
// (Ollama, LM Studio, llama.cpp and friends need no credentials).
func isLocalEndpoint(rawURL string) bool {
	host := rawURL
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		host = u.Hostname()
	}
	host = strings.ToLower(host)

	for _, local := range []string{"localhost", "127.0.0.1", "::1", "0.0.0.0", "host.docker.internal"} {
		if host == local || strings.Contains(host, local) {
			return true
		}
	}
	return false
}

// resolveEndpoint turns the persisted configuration plus a requested quality
// tier into concrete connection settings.
func resolveEndpoint(cfg types.Config, quality string) (Endpoint, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = defaultBaseURL
	}

	ep := Endpoint{
		URL:    ResolveChatEndpoint(base),
		APIKey: strings.TrimSpace(cfg.APIKey),
		Model:  ResolveModel(cfg.ModelName, quality),
		base:   strings.TrimRight(base, "/"),
	}

	if ep.APIKey == "" && !isLocalEndpoint(ep.URL) {
		return ep, fmt.Errorf("%w for %s: add one in Settings, or point the Endpoint Base URL at a local vision server",
			ErrMissingAPIKey, ep.base)
	}
	return ep, nil
}

// MimeTypeFor returns the image MIME type for a file path, defaulting to PNG
// when the extension is unknown.
func MimeTypeFor(filePath string) string {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".tif", ".tiff":
		return "image/tiff"
	case ".pdf":
		return "application/pdf"
	default:
		if m := mime.TypeByExtension(strings.ToLower(filepath.Ext(filePath))); m != "" {
			return m
		}
		return "image/png"
	}
}

// encodeFileAsDataURL reads an image and returns it as an RFC 2397 data URL,
// which is how the OpenAI-compatible image_url field carries inline bytes.
func encodeFileAsDataURL(filePath string) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file no longer exists: %s", filePath)
		}
		return "", fmt.Errorf("inspect %s: %w", filepath.Base(filePath), err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory, not a document", filePath)
	}
	if info.Size() == 0 {
		return "", fmt.Errorf("%s is empty", filepath.Base(filePath))
	}
	if info.Size() > maxImageBytes {
		return "", fmt.Errorf("%s is %.1f MB, which exceeds the %d MB limit for inline upload",
			filepath.Base(filePath), float64(info.Size())/(1<<20), maxImageBytes>>20)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", filepath.Base(filePath), err)
	}

	return fmt.Sprintf("data:%s;base64,%s",
		MimeTypeFor(filePath), base64.StdEncoding.EncodeToString(data)), nil
}

// httpClient is shared so connections are pooled across calls.
var httpClient = &http.Client{Timeout: requestTimeout}

// complete performs one chat-completion request and returns the assistant's
// message content.
func complete(ctx context.Context, ep Endpoint, systemPrompt, userPrompt, imageDataURL string) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model: ep.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: []messageContent{
				{Type: "text", Text: userPrompt},
				{Type: "image_url", ImageURL: &imageURLField{URL: imageDataURL}},
			}},
		},
		// Near-zero temperature: transcription should be deterministic, not creative.
		Temperature: 0.1,
	})
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.URL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", version.UserAgent())
	if ep.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+ep.APIKey)
	}
	// OpenRouter attributes traffic using these headers.
	if strings.Contains(strings.ToLower(ep.base), "openrouter") {
		req.Header.Set("HTTP-Referer", "https://github.com/its-Sohan/DTES")
		req.Header.Set("X-Title", version.AppName)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return "", context.Canceled
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return "", fmt.Errorf("the vision request to %s timed out after %s", ep.base, requestTimeout)
		}
		return "", fmt.Errorf("could not reach the vision endpoint %s: %w", ep.base, err)
	}
	defer func() {
		// Drain before closing so the connection can be reused.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
		_ = resp.Body.Close()
	}()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response from %s: %w", ep.base, err)
	}

	var parsed chatResponse
	// Unmarshal errors are tolerated here: a non-2xx response may not be JSON,
	// and the status code below is the more informative signal.
	_ = json.Unmarshal(payload, &parsed)

	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(payload))
		if parsed.Error != nil && parsed.Error.Message != "" {
			msg = parsed.Error.Message
		}
		if len(msg) > 500 {
			msg = msg[:500] + "…"
		}
		return "", &APIError{StatusCode: resp.StatusCode, Endpoint: ep.URL, Message: msg}
	}

	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("vision API reported: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("the model at %s returned no choices; the image may have been rejected by a content filter", ep.base)
	}

	return parsed.Choices[0].Message.Content, nil
}
