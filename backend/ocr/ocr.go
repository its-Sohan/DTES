package ocr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"

	"itt-ocr/backend/config"
)

const (
	OCRBasePrompt = "You are an expert high-precision OCR and document transcription engine. " +
		"Transcribe all visible text, handwritten notes, numbers, tables, and punctuation from this image accurately. " +
		"Support multilingual scripts including English, Bengali (বাংলা), Assamese, Hindi, and others accurately with correct conjuncts and diacritics. " +
		"Output clean text or Markdown only without introductory pleasantries or commentary."
)

type ModeInfo struct {
	Label             string
	SystemInstruction string
	UserPrompt        string
}

var OutputModes = map[string]ModeInfo{
	"document": {
		Label: "Document",
		SystemInstruction: "Preserve structural elements such as headings, lists, tables, and paragraphs where applicable. " +
			"Maintain natural reading order and document hierarchy.",
		UserPrompt: "Please transcribe and extract all text and layout elements present in this image preserving original structure.",
	},
	"spreadsheet": {
		Label: "Spreadsheet",
		SystemInstruction: "You are a specialized financial and tabular document extractor. " +
			"Identify all tables, itemized billing rows, quantities, rates, unit prices, descriptions, and numerical totals. " +
			"Format all tabular sections strictly as clean Markdown tables with header rows (`| Col 1 | Col 2 |`) so they can be exported to CSV or pasted into Excel. " +
			"For non-table document metadata (such as invoice number, date, vendor name, buyer name, total amount), format them as a concise 2-column key-value table (`| Field | Value |`). " +
			"Do NOT merge separate columns into combined text paragraphs.",
		UserPrompt: "Extract all tabular data, line items, and document metadata from this image strictly into formatted tables suitable for spreadsheets.",
	},
	"key_value": {
		Label: "Key-Value Form",
		SystemInstruction: "You are a structured data and form extractor. " +
			"Extract every form field, label, identifier, and value present in the image. " +
			"Format strictly as clean key-value pairs (`Field Name: Value`). " +
			"Group related fields under concise markdown headings. " +
			"Do not output conversational commentary.",
		UserPrompt: "Extract all form fields, labels, and corresponding values from this image as structured key-value pairs.",
	},
	"raw_text": {
		Label: "Raw Text",
		SystemInstruction: "You are a pure OCR transcription engine. " +
			"Transcribe all text in natural reading order. " +
			"Output pure plain text only with zero markdown formatting, zero table pipes, zero bold asterisks, and zero commentary.",
		UserPrompt: "Transcribe all text from this image as raw unformatted plain text.",
	},
}

var QualityModels = map[string]string{
	"standard": "gemini-3.5-flash-lite",
	"high":     "gemini-3.7-flash",
}

// ResolveChatEndpoint normalizes base endpoint URLs into a valid /chat/completions route
func ResolveChatEndpoint(baseURL string) string {
	clean := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if clean == "" {
		clean = "https://api.openai.com/v1"
	}

	if strings.Contains(clean, "generativelanguage.googleapis.com") {
		if strings.HasSuffix(clean, "/chat/completions") {
			return clean
		}
		if !strings.HasSuffix(clean, "/openai") {
			if strings.HasSuffix(clean, "/v1") || strings.HasSuffix(clean, "/v1beta") {
				lastSlash := strings.LastIndex(clean, "/")
				clean = clean[:lastSlash]
			}
			clean = fmt.Sprintf("%s/v1beta/openai", clean)
		}
		return fmt.Sprintf("%s/chat/completions", clean)
	}

	if strings.HasSuffix(clean, "/chat/completions") {
		return clean
	}
	return fmt.Sprintf("%s/chat/completions", clean)
}

func getMimeType(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
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
	default:
		m := mime.TypeByExtension(ext)
		if m != "" {
			return m
		}
		return "image/jpeg"
	}
}

type chatMessageContent struct {
	Type     string         `json:"type"`
	Text     string         `json:"text,omitempty"`
	ImageURL *imageURLField `json:"image_url,omitempty"`
}

type imageURLField struct {
	URL string `json:"url"`
}

type chatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type chatCompletionPayload struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ExtractText sends document image to multimodal vision endpoint
func ExtractText(filePath string, mode string, quality string) (string, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	apiKey := strings.TrimSpace(cfg.APIKey)
	rawBase := strings.TrimSpace(cfg.BaseURL)
	if rawBase == "" {
		rawBase = "https://api.openai.com/v1"
	}
	url := ResolveChatEndpoint(rawBase)

	modelName := cfg.ModelName
	if qModel, ok := QualityModels[quality]; ok && quality != "" {
		modelName = qModel
	}
	if modelName == "" {
		modelName = "gpt-4o-mini"
	}

	cleanBase := strings.TrimRight(rawBase, "/")
	if apiKey == "" && !strings.Contains(cleanBase, "localhost") && !strings.Contains(cleanBase, "127.0.0.1") {
		return "", fmt.Errorf("API Key is missing. Please configure your API Key in Settings.")
	}

	imageData, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("could not read image file: %w", err)
	}

	mimeType := getMimeType(filePath)
	base64Str := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str)

	modeInfo, exists := OutputModes[mode]
	if !exists {
		modeInfo = OutputModes["document"]
	}

	effectiveSysPrompt := fmt.Sprintf("%s\n\n[OUTPUT FORMAT DIRECTIVE: %s]\n%s",
		OCRBasePrompt, strings.ToUpper(modeInfo.Label), modeInfo.SystemInstruction)

	messages := []chatMessage{
		{
			Role:    "system",
			Content: effectiveSysPrompt,
		},
		{
			Role: "user",
			Content: []chatMessageContent{
				{
					Type: "text",
					Text: modeInfo.UserPrompt,
				},
				{
					Type: "image_url",
					ImageURL: &imageURLField{
						URL: dataURL,
					},
				},
			},
		},
	}

	payload := chatCompletionPayload{
		Model:       modelName,
		Messages:    messages,
		Temperature: 0.1,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to encode request payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	if strings.Contains(strings.ToLower(cleanBase), "openrouter") {
		req.Header.Set("HTTP-Referer", "https://github.com/its-Sohan/itt_ocr_client")
		req.Header.Set("X-Title", "ITT OCR Client")
	}

	client := &http.Client{
		Timeout: 90 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		_, _ = config.UpdateUsageStats(0, false, 0)
		return "", fmt.Errorf("network error while contacting vision API (%s): %w", cleanBase, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		_, _ = config.UpdateUsageStats(0, false, 0)
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_, _ = config.UpdateUsageStats(0, false, 0)
		var errResp chatCompletionResponse
		if err := json.Unmarshal(bodyBytes, &errResp); err == nil && errResp.Error != nil {
			return "", fmt.Errorf("LLM API error (%d): %s", resp.StatusCode, errResp.Error.Message)
		}
		if resp.StatusCode == 404 {
			return "", fmt.Errorf("Endpoint Not Found (404) at %s. Please verify your Endpoint URL in Settings.", url)
		}
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return "", fmt.Errorf("Authentication/Authorization Error (%d). Please check your API key in Settings.", resp.StatusCode)
		}
		return "", fmt.Errorf("LLM API error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp chatCompletionResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		_, _ = config.UpdateUsageStats(0, false, 0)
		return "", fmt.Errorf("failed to parse API JSON reply: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		_, _ = config.UpdateUsageStats(0, false, 0)
		return "", fmt.Errorf("LLM API returned an empty choices list")
	}

	rawText := chatResp.Choices[0].Message.Content
	// Apply Unicode NFC normalization to preserve Bengali conjuncts & diacritics
	normalizedText := norm.NFC.String(strings.TrimSpace(rawText))

	_, _ = config.UpdateUsageStats(len(normalizedText), true, 0)
	return normalizedText, nil
}
