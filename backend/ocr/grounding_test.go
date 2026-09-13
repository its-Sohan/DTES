package ocr

import (
	"strings"
	"testing"
)

func TestParseBoundingBoxes(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		blockCount int
		want       []int // flattened index,ymin,xmin,ymax,xmax per box
	}{
		{
			name:       "clean array",
			content:    `[{"index":0,"box_2d":[10,20,30,40]}]`,
			blockCount: 1,
			want:       []int{0, 10, 20, 30, 40},
		},
		{
			name:       "wrapped in a code fence",
			content:    "```json\n[{\"index\":0,\"box_2d\":[10,20,30,40]}]\n```",
			blockCount: 1,
			want:       []int{0, 10, 20, 30, 40},
		},
		{
			name:       "surrounded by prose",
			content:    `Sure! Here are the boxes: [{"index":0,"box_2d":[1,2,3,4]}] Hope that helps.`,
			blockCount: 1,
			want:       []int{0, 1, 2, 3, 4},
		},
		{
			name:       "multiple blocks",
			content:    `[{"index":0,"box_2d":[0,0,100,500]},{"index":1,"box_2d":[110,0,200,500]}]`,
			blockCount: 2,
			want:       []int{0, 0, 0, 100, 500, 1, 110, 0, 200, 500},
		},
		{
			name:       "coordinates are clamped to 0..1000",
			content:    `[{"index":0,"box_2d":[-50,-10,5000,2000]}]`,
			blockCount: 1,
			want:       []int{0, 0, 0, 1000, 1000},
		},
		{
			name:       "reversed pairs are swapped",
			content:    `[{"index":0,"box_2d":[300,400,100,200]}]`,
			blockCount: 1,
			want:       []int{0, 100, 200, 300, 400},
		},
		{
			name:       "zero-area boxes are dropped",
			content:    `[{"index":0,"box_2d":[100,100,100,100]}]`,
			blockCount: 1,
			want:       nil,
		},
		{
			name:       "boxes with the wrong arity are dropped",
			content:    `[{"index":0,"box_2d":[1,2,3]}]`,
			blockCount: 1,
			want:       nil,
		},
		{
			name:       "an out-of-range index falls back to position",
			content:    `[{"index":99,"box_2d":[10,20,30,40]}]`,
			blockCount: 1,
			want:       []int{0, 10, 20, 30, 40},
		},
		{
			name:       "indexes beyond the block count are dropped",
			content:    `[{"index":0,"box_2d":[1,2,3,4]},{"index":1,"box_2d":[5,6,7,8]}]`,
			blockCount: 1,
			want:       []int{0, 1, 2, 3, 4},
		},
		{
			name:       "duplicate indexes keep only the first",
			content:    `[{"index":0,"box_2d":[1,2,3,4]},{"index":0,"box_2d":[9,9,99,99]}]`,
			blockCount: 2,
			want:       []int{0, 1, 2, 3, 4},
		},
		{name: "no JSON at all", content: "I cannot help with that.", blockCount: 1, want: nil},
		{name: "empty string", content: "", blockCount: 1, want: nil},
		{name: "malformed JSON", content: `[{"index":0,"box_2d":[1,2,`, blockCount: 1, want: nil},
		{name: "empty array", content: `[]`, blockCount: 1, want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseBoundingBoxes(tc.content, tc.blockCount)

			var flat []int
			for _, b := range got {
				flat = append(flat, b.Index, b.YMin, b.XMin, b.YMax, b.XMax)
			}

			if len(flat) != len(tc.want) {
				t.Fatalf("got %d boxes %v, want %d %v", len(got), flat, len(tc.want)/5, tc.want)
			}
			for i := range flat {
				if flat[i] != tc.want[i] {
					t.Fatalf("boxes = %v, want %v", flat, tc.want)
				}
			}
		})
	}
}

// TestParseBoundingBoxesNeverReturnsNilSlice keeps the JSON contract with the
// frontend: null would break .map() in the preview overlay.
func TestParseBoundingBoxesNeverReturnsNilSlice(t *testing.T) {
	if got := parseBoundingBoxes("nothing here", 3); got == nil {
		t.Error("parseBoundingBoxes returned nil; want an empty slice")
	}
}

// TestParsedBoxesStayInCoordinateSpace is the invariant the preview overlay
// relies on to convert coordinates into CSS percentages.
func TestParsedBoxesStayInCoordinateSpace(t *testing.T) {
	content := `[{"index":0,"box_2d":[-999,-999,99999,99999]},{"index":1,"box_2d":[0,0,1000,1000]}]`
	for _, b := range parseBoundingBoxes(content, 2) {
		if b.YMin < 0 || b.XMin < 0 || b.YMax > coordMax || b.XMax > coordMax {
			t.Errorf("box %+v escapes the 0..%d space", b, coordMax)
		}
		if b.YMax <= b.YMin || b.XMax <= b.XMin {
			t.Errorf("box %+v has non-positive area", b)
		}
	}
}

func TestClampCoord(t *testing.T) {
	tests := map[int]int{-1: 0, -1000: 0, 0: 0, 500: 500, 1000: 1000, 1001: 1000, 99999: 1000}
	for in, want := range tests {
		if got := clampCoord(in); got != want {
			t.Errorf("clampCoord(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestBuildGroundingPrompt(t *testing.T) {
	blocks := []string{"First block of text", "Second block"}
	got := buildGroundingPrompt(blocks)

	if !strings.Contains(got, "Block 0:") || !strings.Contains(got, "Block 1:") {
		t.Errorf("prompt must enumerate every block, got:\n%s", got)
	}
	if !strings.Contains(got, "First block of text") {
		t.Errorf("prompt must include block text, got:\n%s", got)
	}
	if !strings.Contains(got, "2 text blocks") {
		t.Errorf("prompt should state the block count, got:\n%s", got)
	}
}

// TestBuildGroundingPromptFlattensWhitespace keeps one block on one prompt
// line, so the enumerated list cannot be misread by the model.
func TestBuildGroundingPromptFlattensWhitespace(t *testing.T) {
	got := buildGroundingPrompt([]string{"line one\nline two\n\tindented"})

	body := strings.SplitN(got, "\n\n", 2)[1]
	firstLine := strings.SplitN(body, "\n", 2)[0]
	if !strings.Contains(firstLine, "line one line two indented") {
		t.Errorf("expected whitespace collapsed onto one line, got %q", firstLine)
	}
}

// TestBuildGroundingPromptTruncatesLongBlocks keeps the prompt from crowding
// out the image on text-heavy documents.
func TestBuildGroundingPromptTruncatesLongBlocks(t *testing.T) {
	long := strings.Repeat("word ", 400)
	got := buildGroundingPrompt([]string{long})

	if len([]rune(got)) > blockPreviewRunes+400 {
		t.Errorf("prompt is %d runes; long blocks should be truncated", len([]rune(got)))
	}
	if !strings.Contains(got, "…") {
		t.Error("expected a truncation ellipsis")
	}
}

func TestBuildGroundingPromptEscapesQuotes(t *testing.T) {
	got := buildGroundingPrompt([]string{`he said "hi" then left`})
	// %q escaping keeps the enumerated list parseable.
	if !strings.Contains(got, `\"hi\"`) {
		t.Errorf("inner quotes should be escaped, got:\n%s", got)
	}
}
