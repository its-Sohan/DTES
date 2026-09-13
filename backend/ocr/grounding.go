package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"itt-ocr/backend/config"
	"itt-ocr/backend/types"
)

// coordMax is the upper bound of the normalised coordinate space. Boxes are
// expressed as integers in 0..coordMax on both axes so they are independent of
// the source image's pixel dimensions.
const coordMax = 1000

// maxGroundingBlocks caps how many text blocks are sent for localisation. Very
// long documents would otherwise produce a prompt large enough to crowd out the
// image itself, degrading the result for every block.
const maxGroundingBlocks = 60

// blockPreviewRunes limits each block's excerpt in the prompt; the model only
// needs enough text to locate the block, not to re-read it.
const blockPreviewRunes = 120

// jsonArrayPattern finds the JSON array in a response that may be wrapped in
// prose or a code fence despite instructions to the contrary.
var jsonArrayPattern = regexp.MustCompile(`(?s)\[\s*\{.*}\s*]`)

const groundingSystemPrompt = "You are an expert document visual grounding and layout analysis engine. " +
	"Your task is to detect the exact 2D bounding box for each given text block in the image. " +
	"Coordinates must be normalised integers from 0 to 1000, where:\n" +
	"- y: 0 is the top edge, 1000 is the bottom edge of the image.\n" +
	"- x: 0 is the left edge, 1000 is the right edge of the image.\n" +
	"Return strictly a JSON array of objects with keys 'index' and 'box_2d':\n" +
	"[{\"index\": 0, \"box_2d\": [ymin, xmin, ymax, xmax]}, ...]\n" +
	"Ensure ymin < ymax and xmin < xmax. Include one entry per block, using the " +
	"index given for that block. Omit blocks you cannot locate rather than guessing. " +
	"Output only the JSON array, with no markdown fence or commentary."

// groundingBox is the wire shape the model is asked to produce.
type groundingBox struct {
	Index int    `json:"index"`
	Box   []int  `json:"box_2d"`
	Label string `json:"label,omitempty"`
}

// AlignBlocksWithAI locates each supplied text block within the source image
// and returns their normalised bounding boxes.
//
// This powers Audit Mode, where selecting a block of extracted text highlights
// the matching region of the original scan. It is strictly auxiliary: callers
// should treat any error as non-fatal and simply omit the highlights.
//
// Blocks that the model cannot locate, or that it returns with degenerate
// coordinates, are dropped rather than approximated.
func AlignBlocksWithAI(ctx context.Context, filePath string, blocks []string) ([]types.BoundingBox, error) {
	if len(blocks) == 0 {
		return []types.BoundingBox{}, nil
	}
	if strings.TrimSpace(filePath) == "" {
		return nil, fmt.Errorf("no document was supplied")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Grounding always uses the configured model directly. The high-quality
	// upgrade is reserved for transcription, where accuracy is user-visible;
	// paying more for auxiliary highlights is not a good trade.
	ep, err := resolveEndpoint(cfg, string(types.QualityStandard))
	if err != nil {
		return nil, err
	}

	dataURL, err := encodeFileAsDataURL(filePath)
	if err != nil {
		return nil, err
	}

	limit := min(len(blocks), maxGroundingBlocks)
	userPrompt := buildGroundingPrompt(blocks[:limit])

	raw, err := complete(ctx, ep, groundingSystemPrompt, userPrompt, dataURL)
	if err != nil {
		return nil, err
	}

	return parseBoundingBoxes(raw, limit), nil
}

// buildGroundingPrompt renders the block excerpts the model must locate.
func buildGroundingPrompt(blocks []string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Detect the normalised 2D bounding boxes [ymin, xmin, ymax, xmax] (0 to 1000) for these %d text blocks in the image:\n\n", len(blocks))

	for i, b := range blocks {
		// Collapse internal whitespace so a multi-line block stays on one
		// prompt line, keeping the block list unambiguous.
		excerpt := strings.Join(strings.Fields(b), " ")
		if r := []rune(excerpt); len(r) > blockPreviewRunes {
			excerpt = string(r[:blockPreviewRunes]) + "…"
		}
		fmt.Fprintf(&sb, "Block %d: %q\n", i, excerpt)
	}

	sb.WriteString("\nOutput strictly the JSON array.")
	return sb.String()
}

// parseBoundingBoxes extracts valid boxes from a model response. It is
// deliberately lenient about surrounding text but strict about the coordinates
// themselves: a bad box would draw a highlight over the wrong region, which is
// worse than drawing none.
func parseBoundingBoxes(content string, blockCount int) []types.BoundingBox {
	match := jsonArrayPattern.FindString(strings.TrimSpace(content))
	if match == "" {
		return []types.BoundingBox{}
	}

	var parsed []groundingBox
	if err := json.Unmarshal([]byte(match), &parsed); err != nil {
		return []types.BoundingBox{}
	}

	out := make([]types.BoundingBox, 0, len(parsed))
	seen := make(map[int]struct{}, len(parsed))

	for fallbackIdx, item := range parsed {
		if len(item.Box) != 4 {
			continue
		}

		idx := item.Index
		if idx < 0 || idx >= blockCount {
			// The model omitted or invented an index; fall back to position.
			idx = fallbackIdx
		}
		if idx >= blockCount {
			continue
		}
		// Keep only the first box for any block.
		if _, dup := seen[idx]; dup {
			continue
		}

		ymin, xmin := clampCoord(item.Box[0]), clampCoord(item.Box[1])
		ymax, xmax := clampCoord(item.Box[2]), clampCoord(item.Box[3])

		// Some models emit the pair reversed; a swap is unambiguous and safe.
		if ymin > ymax {
			ymin, ymax = ymax, ymin
		}
		if xmin > xmax {
			xmin, xmax = xmax, xmin
		}
		// A zero-area box cannot be rendered meaningfully.
		if ymax <= ymin || xmax <= xmin {
			continue
		}

		seen[idx] = struct{}{}
		out = append(out, types.BoundingBox{
			Index: idx, YMin: ymin, XMin: xmin, YMax: ymax, XMax: xmax,
		})
	}

	return out
}

// clampCoord constrains a coordinate to the normalised 0..coordMax space.
func clampCoord(v int) int {
	if v < 0 {
		return 0
	}
	if v > coordMax {
		return coordMax
	}
	return v
}
