package ocr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"itt-ocr/backend/config"
	"itt-ocr/backend/types"
)

var jsonArrayRegex = regexp.MustCompile(`\[\s*\{.*\}\s*\]`)

// AlignBlocksWithAI detects normalized 2D bounding boxes [ymin, xmin, ymax, xmax] (0-1000) for text blocks
func AlignBlocksWithAI(filePath string, blocks []string) ([]types.BoundingBox, error) {
	if len(blocks) == 0 {
		return []types.BoundingBox{}, nil
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	apiKey := strings.TrimSpace(cfg.APIKey)
	rawBase := strings.TrimSpace(cfg.BaseURL)
	if rawBase == "" {
		rawBase = "https://api.openai.com/v1"
	}
	url := ResolveChatEndpoint(rawBase)
	cleanBase := strings.TrimRight(rawBase, "/")

	modelName := cfg.ModelName
	if modelName == "" {
		modelName = "gpt-4o-mini"
	}

	if apiKey == "" && !strings.Contains(cleanBase, "localhost") && !strings.Contains(cleanBase, "127.0.0.1") {
		return nil, fmt.Errorf("API Key is missing. Please configure your API Key in Settings.")
	}

	imageData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read image file: %w", err)
	}

	mimeType := getMimeType(filePath)
	base64Str := base64.StdEncoding.EncodeToString(imageData)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Str)

	var blockPreviews []string
	for idx, b := range blocks {
		words := strings.Fields(b)
		snip := strings.Join(words, " ")
		if len(snip) > 120 {
			snip = snip[:120]
		}
		blockPreviews = append(blockPreviews, fmt.Sprintf("Block %d: %q", idx, snip))
	}
	blocksText := strings.Join(blockPreviews, "\n")

	systemPrompt := "You are an expert document visual grounding and layout analysis engine. " +
		"Your task is to detect the exact 2D bounding boxes for each given text block in the image. " +
		"Coordinates must be normalized integers from 0 to 1000, where:\n" +
		"- 0 is the top edge, 1000 is the bottom edge of the image.\n" +
		"- 0 is the left edge, 1000 is the right edge of the image.\n" +
		"Return strictly a JSON array of objects with keys 'index' and 'box_2d':\n" +
		"[{\"index\": 0, \"box_2d\": [ymin, xmin, ymax, xmax]}, ...]\n" +
		"Ensure ymin < ymax and xmin < xmax. Do not include markdown commentary."

	userPrompt := fmt.Sprintf("Detect the normalized 2D bounding boxes [ymin, xmin, ymax, xmax] (0 to 1000) for these %d text blocks in the image:\n\n%s\n\nOutput strictly the JSON array.", len(blocks), blocksText)

	messages := []chatMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role: "user",
			Content: []chatMessageContent{
				{
					Type: "text",
					Text: userPrompt,
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
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	if strings.Contains(strings.ToLower(cleanBase), "openrouter") {
		req.Header.Set("HTTP-Referer", "https://github.com/its-Sohan/itt_ocr_client")
		req.Header.Set("X-Title", "ITT OCR Client")
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error during grounding: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read grounding response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("grounding API error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp chatCompletionResponse
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse grounding response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("empty choices from grounding API")
	}

	content := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	match := jsonArrayRegex.FindString(content)
	if match == "" {
		return []types.BoundingBox{}, nil
	}

	var rawList []map[string]interface{}
	if err := json.Unmarshal([]byte(match), &rawList); err != nil {
		return []types.BoundingBox{}, nil
	}

	clamp := func(val int) int {
		if val < 0 {
			return 0
		}
		if val > 1000 {
			return 1000
		}
		return val
	}

	var result []types.BoundingBox
	for idx, item := range rawList {
		itemIndex := idx
		if fIdx, ok := item["index"].(float64); ok {
			itemIndex = int(fIdx)
		}

		if rawBox, ok := item["box_2d"].([]interface{}); ok && len(rawBox) == 4 {
			ymin := clamp(int(rawBox[0].(float64)))
			xmin := clamp(int(rawBox[1].(float64)))
			ymax := clamp(int(rawBox[2].(float64)))
			xmax := clamp(int(rawBox[3].(float64)))
			if ymax > ymin && xmax > xmin {
				result = append(result, types.BoundingBox{
					Index: itemIndex,
					YMin:  ymin,
					XMin:  xmin,
					YMax:  ymax,
					XMax:  xmax,
				})
			}
		}
	}

	return result, nil
}
