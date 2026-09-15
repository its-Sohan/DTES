// Package ocr turns document images into text using an OpenAI-compatible
// multimodal (vision) chat endpoint.
//
// The provider is user-configurable: anything speaking the
// /chat/completions schema works, including OpenAI, Google Gemini's
// compatibility layer, OpenRouter and local servers such as Ollama or LM Studio.
package ocr

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/text/unicode/norm"

	"itt-ocr/backend/config"
	"itt-ocr/backend/types"
)

// defaultBaseURL is used when no endpoint has been configured.
const defaultBaseURL = "https://ai.rupic.studio/v1"

// basePrompt states the transcription contract shared by every output mode.
const basePrompt = "You are an expert high-precision OCR and document transcription engine. " +
	"Transcribe all visible text, handwritten notes, numbers, tables, and punctuation from this image accurately. " +
	"Support multilingual scripts including English, Bengali (বাংলা), Assamese, Hindi, and others accurately with correct conjuncts and diacritics. " +
	"Never invent, correct, translate or summarise content that is not visibly present. " +
	"If a region is genuinely illegible, mark it [illegible] rather than guessing. " +
	"Output clean text or Markdown only, with no introductory pleasantries or trailing commentary."

// Mode describes one output shape and the prompts that elicit it.
type Mode struct {
	// ID is the stable key used by config and the frontend.
	ID types.OutputMode `json:"id"`
	// Label is the human-readable name shown in the UI.
	Label string `json:"label"`
	// Description explains when to pick this mode.
	Description string `json:"description"`
	// SystemInstruction is appended to basePrompt.
	SystemInstruction string `json:"-"`
	// UserPrompt accompanies the image in the user message.
	UserPrompt string `json:"-"`
}

// modes is the authoritative registry of output modes.
var modes = map[types.OutputMode]Mode{
	types.OutputModeDocument: {
		ID:          types.OutputModeDocument,
		Label:       "Document",
		Description: "Prose, headings and mixed layouts. Preserves reading order and structure.",
		SystemInstruction: "Preserve structural elements such as headings, lists, tables, and paragraphs where applicable. " +
			"Maintain natural reading order and document hierarchy.",
		UserPrompt: "Transcribe all text and layout elements present in this image, preserving the original structure.",
	},
	types.OutputModeSpreadsheet: {
		ID:          types.OutputModeSpreadsheet,
		Label:       "Spreadsheet",
		Description: "Invoices, ledgers and tables. Emits Markdown tables ready for CSV export.",
		SystemInstruction: "You are a specialised financial and tabular document extractor. " +
			"Identify all tables, itemised billing rows, quantities, rates, unit prices, descriptions, and numerical totals. " +
			"Format all tabular sections strictly as clean Markdown tables with header rows (`| Col 1 | Col 2 |`) so they can be exported to CSV or pasted into a spreadsheet. " +
			"For non-table document metadata (such as invoice number, date, vendor name, buyer name, total amount), format them as a concise two-column key-value table (`| Field | Value |`). " +
			"Never merge separate columns into combined text paragraphs. " +
			"Transcribe numbers exactly as printed; do not recompute or correct totals.",
		UserPrompt: "Extract all tabular data, line items, and document metadata from this image strictly as formatted Markdown tables suitable for spreadsheets.",
	},
	types.OutputModeKeyValue: {
		ID:          types.OutputModeKeyValue,
		Label:       "Key-Value",
		Description: "Forms and applications. Emits `Field: Value` pairs grouped by section.",
		SystemInstruction: "You are a structured data and form extractor. " +
			"Extract every form field, label, identifier, and value present in the image. " +
			"Format strictly as clean key-value pairs (`Field Name: Value`). " +
			"Group related fields under concise Markdown headings. " +
			"Where a field is present but blank, emit the field with an empty value rather than omitting it. " +
			"Do not output conversational commentary.",
		UserPrompt: "Extract all form fields, labels, and corresponding values from this image as structured key-value pairs.",
	},
	types.OutputModeRawText: {
		ID:          types.OutputModeRawText,
		Label:       "Raw Text",
		Description: "Verbatim plain text with no Markdown at all.",
		SystemInstruction: "You are a pure OCR transcription engine. " +
			"Transcribe all text in natural reading order. " +
			"Output pure plain text only, with zero Markdown formatting, zero table pipes, zero bold asterisks, and zero commentary.",
		UserPrompt: "Transcribe all text from this image as raw, unformatted plain text.",
	},
}

// modeOrder fixes the display order, since map iteration is random.
var modeOrder = []types.OutputMode{
	types.OutputModeDocument,
	types.OutputModeSpreadsheet,
	types.OutputModeKeyValue,
	types.OutputModeRawText,
}

// Modes returns the available output modes in display order.
func Modes() []Mode {
	out := make([]Mode, 0, len(modeOrder))
	for _, id := range modeOrder {
		out = append(out, modes[id])
	}
	return out
}

// modeFor returns the requested mode, falling back to Document.
func modeFor(mode string) Mode {
	if m, ok := modes[types.OutputMode(mode)]; ok {
		return m
	}
	return modes[types.OutputModeDocument]
}

// geminiHighPrecisionOrder defines the cascading fallback preference sequence
// for High Precision mode when using Gemini family models.
var geminiHighPrecisionOrder = []string{
	"gemini-3.7-flash",
	"gemini-3.6-flash",
	"gemini-3.5-flash",
	"gemini-3.5-flash-lite",
}

// defaultDocumentOCRModel is the official Mistral OCR model identifier used
// when Document mode is selected.
const defaultDocumentOCRModel = "mistral-ocr-latest"

// isGeminiModel reports whether the model belongs to the Gemini family.
func isGeminiModel(model string) bool {
	lower := strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(lower, "gemini-")
}

// ResolveModelChain returns an ordered slice of candidate models for a request.
// For High Precision mode with Gemini models, it returns a cascading list
// starting from the most capable model (3.7) down to reliable fallbacks (3.6, 3.5, lite).
// For standard or custom models, it returns a single-item slice with the target model.
func ResolveModelChain(configured, quality string) []string {
	model := strings.TrimSpace(configured)
	if model == "" {
		model = "gemini-3.5-flash-lite"
	}

	switch types.Quality(quality) {
	case types.QualityHigh:
		if isGeminiModel(model) {
			chain := make([]string, len(geminiHighPrecisionOrder))
			copy(chain, geminiHighPrecisionOrder)
			return chain
		}
		return []string{model}
	case types.QualityDocument:
		return []string{defaultDocumentOCRModel}
	default:
		return []string{model}
	}
}

// ResolveModel picks the primary model for a request.
//
// The configured model is always the baseline. The "high" tier upgrades it only
// when a known better sibling exists, so a custom or self-hosted model name is
// never silently replaced with something the provider has never heard of.
// The "document" tier routes the request directly to the dedicated OCR model.
func ResolveModel(configured, quality string) string {
	chain := ResolveModelChain(configured, quality)
	if len(chain) > 0 {
		return chain[0]
	}
	return "gemini-3.5-flash-lite"
}

// ExtractText transcribes the document at filePath.
//
// mode selects the output shape (see Modes) and quality selects the
// speed/accuracy tradeoff. The returned text is NFC-normalised so that Bengali
// and other Indic conjuncts and diacritics compare and render correctly.
//
// Every completed attempt is recorded in the local usage counters exactly once.
func ExtractText(ctx context.Context, filePath, mode, quality string) (string, error) {
	text, err := extract(ctx, filePath, mode, quality)

	// Record the outcome once, here, rather than at each early return — that
	// duplication previously made the counters unreliable. A cancelled request
	// is a user action, not an attempt, so it is not counted.
	if err != nil {
		if !isCancellation(err) {
			_, _ = config.RecordExtraction(0, false)
		}
		return "", err
	}
	_, _ = config.RecordExtraction(len([]rune(text)), true)
	return text, nil
}

func extract(ctx context.Context, filePath, mode, quality string) (string, error) {
	if strings.TrimSpace(filePath) == "" {
		return "", fmt.Errorf("no document was supplied")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		// A corrupt config still yields usable defaults, so continue rather
		// than failing the extraction outright.
		cfg = config.DefaultConfig()
	}

	ep, err := resolveEndpoint(cfg, quality)
	if err != nil {
		return "", err
	}

	dataURL, err := encodeFileAsDataURL(filePath)
	if err != nil {
		return "", err
	}

	var raw string
	if types.Quality(quality) == types.QualityDocument {
		raw, err = completeOCR(ctx, ep, dataURL, MimeTypeFor(filePath))
		if err != nil {
			return "", err
		}
	} else {
		m := modeFor(mode)
		systemPrompt := fmt.Sprintf("%s\n\n[OUTPUT FORMAT DIRECTIVE: %s]\n%s",
			basePrompt, strings.ToUpper(m.Label), m.SystemInstruction)

		candidates := ResolveModelChain(cfg.ModelName, quality)
		var lastErr error
		for i, candidateModel := range candidates {
			candidateEP := ep
			candidateEP.Model = candidateModel

			raw, err = complete(ctx, candidateEP, systemPrompt, m.UserPrompt, dataURL)
			if err == nil {
				lastErr = nil
				break
			}
			lastErr = err

			// If the user cancelled, abort immediately without trying fallback models.
			if errors.Is(err, context.Canceled) {
				return "", err
			}

			// If the error is not eligible for fallback, or if we exhausted all candidates, stop.
			if !isFallbackEligible(err) || i == len(candidates)-1 {
				break
			}
		}
		if lastErr != nil {
			return "", lastErr
		}
	}

	text := norm.NFC.String(strings.TrimSpace(raw))
	text = stripCodeFence(text)
	if text == "" {
		return "", fmt.Errorf("the model returned no text; the document may be blank or unreadable")
	}
	return text, nil
}

// stripCodeFence removes a single wrapping ``` fence, which models add despite
// being told not to. Fences *within* the text are left alone, since a genuine
// multi-block response should keep its structure.
func stripCodeFence(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}

	lines := strings.Split(s, "\n")
	if len(lines) < 2 {
		return s
	}
	// The closing fence must be the final non-empty line.
	last := len(lines) - 1
	for last > 0 && strings.TrimSpace(lines[last]) == "" {
		last--
	}
	if last == 0 || strings.TrimSpace(lines[last]) != "```" {
		return s
	}
	// Only unwrap when there is exactly one fence pair.
	for _, l := range lines[1:last] {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			return s
		}
	}
	return strings.TrimSpace(strings.Join(lines[1:last], "\n"))
}

func isCancellation(err error) bool {
	return err != nil && (err == context.Canceled ||
		strings.Contains(err.Error(), context.Canceled.Error()))
}
