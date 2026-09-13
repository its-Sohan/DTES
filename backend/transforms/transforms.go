package transforms

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"itt-ocr/backend/types"
)

var (
	bengaliToEnglishMap = map[rune]rune{
		'০': '0', '১': '1', '২': '2', '৩': '3', '৪': '4',
		'৫': '5', '৬': '6', '৭': '7', '৮': '8', '৯': '9',
	}
	englishToBengaliMap = map[rune]rune{
		'0': '০', '1': '১', '2': '২', '3': '৩', '4': '৪',
		'5': '৫', '6': '৬', '7': '৭', '8': '৮', '9': '৯',
	}
	terminators = []string{"।", ".", "!", "?", ":", ";", "—", "\"", "'", "”", "’"}

	listRegex      = regexp.MustCompile(`^[-*+•]\s+`)
	orderedRegex   = regexp.MustCompile(`^\d+[\.\)]\s+`)
	numRegex       = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)
	redundantNL    = regexp.MustCompile(`\n{3,}`)
	redundantSpace = regexp.MustCompile(`[ \t]{2,}`)
)

// ConvertDigitsToEnglish converts all Bengali numerals (০-৯) to Arabic numerals (0-9)
func ConvertDigitsToEnglish(text string) string {
	var sb strings.Builder
	for _, r := range text {
		if eng, ok := bengaliToEnglishMap[r]; ok {
			sb.WriteRune(eng)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// ConvertDigitsToBengali converts all Arabic numerals (0-9) to Bengali numerals (০-৯)
func ConvertDigitsToBengali(text string) string {
	var sb strings.Builder
	for _, r := range text {
		if ben, ok := englishToBengaliMap[r]; ok {
			sb.WriteRune(ben)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func isSpecialLine(stripped string) bool {
	if len(stripped) == 0 {
		return false
	}
	for _, prefix := range []string{"#", "|", ">", "```", "---", "===", "***"} {
		if strings.HasPrefix(stripped, prefix) {
			return true
		}
	}
	return listRegex.MatchString(stripped) || orderedRegex.MatchString(stripped)
}

func endsWithTerminator(s string) bool {
	trimmed := strings.TrimRight(s, " \t\r\n")
	for _, t := range terminators {
		if strings.HasSuffix(trimmed, t) {
			return true
		}
	}
	return false
}

// UnwrapBrokenLines unwraps artificial line wraps while preserving headings, lists, and tables
func UnwrapBrokenLines(text string) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	var out []string
	i := 0

	for i < len(lines) {
		line := strings.TrimRight(lines[i], " \r\t")
		if line == "" {
			out = append(out, "")
			i++
			continue
		}

		stripped := strings.TrimSpace(line)
		if isSpecialLine(stripped) {
			out = append(out, line)
			i++
			continue
		}

		for i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if nextLine == "" {
				break
			}
			if isSpecialLine(nextLine) {
				break
			}
			if endsWithTerminator(line) {
				break
			}

			line = line + " " + nextLine
			i++
		}

		out = append(out, line)
		i++
	}

	return strings.Join(out, "\n")
}

// CleanWhitespaceAndMargins collapses redundant spaces and excess blank lines
func CleanWhitespaceAndMargins(text string) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	for idx, l := range lines {
		lines[idx] = strings.TrimRight(l, " \r\t")
	}
	joined := strings.Join(lines, "\n")
	joined = redundantNL.ReplaceAllString(joined, "\n\n")

	rawLines := strings.Split(joined, "\n")
	var result []string
	for _, l := range rawLines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			result = append(result, l)
		} else {
			result = append(result, redundantSpace.ReplaceAllString(l, " "))
		}
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

func isSeparatorCell(cell string) bool {
	c := strings.TrimSpace(cell)
	if c == "" {
		return false
	}
	for _, r := range c {
		if r != '-' && r != ':' && r != ' ' {
			return false
		}
	}
	return true
}

// CleanTableFormatting parses Markdown tables and aligns vertical pipes into a uniform ASCII grid
func CleanTableFormatting(text string) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	var out []string
	i := 0

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			var tableLines []string
			for i < len(lines) {
				tl := strings.TrimSpace(lines[i])
				if strings.HasPrefix(tl, "|") && strings.HasSuffix(tl, "|") {
					tableLines = append(tableLines, tl)
					i++
				} else {
					break
				}
			}

			var rows [][]string
			maxCols := 0
			for _, tl := range tableLines {
				inner := strings.Trim(tl, "|")
				rawCells := strings.Split(inner, "|")
				var cells []string
				for _, c := range rawCells {
					cells = append(cells, strings.TrimSpace(c))
				}
				if len(cells) > maxCols {
					maxCols = len(cells)
				}
				rows = append(rows, cells)
			}

			for rIdx := range rows {
				for len(rows[rIdx]) < maxCols {
					rows[rIdx] = append(rows[rIdx], "")
				}
			}

			colWidths := make([]int, maxCols)
			for _, r := range rows {
				for cIdx, cell := range r {
					if isSeparatorCell(cell) {
						continue
					}
					width := len([]rune(cell))
					if width > colWidths[cIdx] {
						colWidths[cIdx] = width
					}
				}
			}
			for cIdx := range colWidths {
				if colWidths[cIdx] < 3 {
					colWidths[cIdx] = 3
				}
			}

			for _, r := range rows {
				isSep := false
				for _, c := range r {
					if isSeparatorCell(c) {
						isSep = true
						break
					}
				}

				var formattedCells []string
				for cIdx, cell := range r {
					w := colWidths[cIdx]
					if isSep {
						formattedCells = append(formattedCells, strings.Repeat("-", w+2))
					} else {
						runes := []rune(cell)
						pad := w - len(runes)
						if pad < 0 {
							pad = 0
						}
						formattedCells = append(formattedCells, fmt.Sprintf(" %s%s ", cell, strings.Repeat(" ", pad)))
					}
				}
				out = append(out, "|"+strings.Join(formattedCells, "|")+"|")
			}
		} else {
			out = append(out, line)
			i++
		}
	}

	return strings.Join(out, "\n")
}

func parseNumeric(val string) (float64, bool) {
	eng := ConvertDigitsToEnglish(val)
	cleaned := strings.ReplaceAll(eng, ",", "")
	cleaned = strings.TrimSpace(cleaned)
	m := numRegex.FindString(cleaned)
	if m == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(m, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// CheckInvoiceMath validates arithmetic totals within invoice tables
func CheckInvoiceMath(text string) *types.InvoiceValidationResult {
	if text == "" {
		return nil
	}

	rawLines := strings.Split(text, "\n")
	var tableLines []string
	for _, l := range rawLines {
		tl := strings.TrimSpace(l)
		if strings.HasPrefix(tl, "|") && strings.HasSuffix(tl, "|") {
			tableLines = append(tableLines, tl)
		}
	}
	if len(tableLines) < 3 {
		return nil
	}

	var (
		totalVal    *float64
		subtotalVal *float64
		taxVal      float64
		discountVal float64
		itemVals    []float64
	)

	totalKeywords := []string{"grand total", "net payable", "net total", "total amount", "total", "সর্বমোট", "মোট টাকা", "মোট"}
	subtotalKeywords := []string{"subtotal", "sub total", "উপমোট", "মোট মূল্য"}
	taxKeywords := []string{"vat", "tax", "gst", "ভ্যাট", "কর", "ট্যাক্স"}
	discountKeywords := []string{"discount", "ছাড়", "কমিশন"}

	containsAny := func(s string, keywords []string) bool {
		for _, k := range keywords {
			if strings.Contains(s, k) {
				return true
			}
		}
		return false
	}

	for _, l := range tableLines {
		inner := strings.Trim(l, "|")
		rawCells := strings.Split(inner, "|")
		var cells []string
		allSep := true
		for _, c := range rawCells {
			cTrim := strings.TrimSpace(c)
			cells = append(cells, cTrim)
			if !isSeparatorCell(cTrim) && cTrim != "" {
				allSep = false
			}
		}
		if allSep {
			continue
		}

		rowStr := strings.ToLower(ConvertDigitsToEnglish(strings.Join(cells, " ")))

		if containsAny(rowStr, totalKeywords) {
			for i := len(cells) - 1; i >= 0; i-- {
				if n, ok := parseNumeric(cells[i]); ok {
					totalVal = &n
					break
				}
			}
		} else if containsAny(rowStr, subtotalKeywords) {
			for i := len(cells) - 1; i >= 0; i-- {
				if n, ok := parseNumeric(cells[i]); ok {
					subtotalVal = &n
					break
				}
			}
		} else if containsAny(rowStr, taxKeywords) {
			for i := len(cells) - 1; i >= 0; i-- {
				if n, ok := parseNumeric(cells[i]); ok {
					taxVal = n
					break
				}
			}
		} else if containsAny(rowStr, discountKeywords) {
			for i := len(cells) - 1; i >= 0; i-- {
				if n, ok := parseNumeric(cells[i]); ok {
					discountVal = n
					break
				}
			}
		} else {
			for _, c := range cells {
				if n, ok := parseNumeric(c); ok {
					itemVals = append(itemVals, n)
				}
			}
		}
	}

	if totalVal != nil {
		itemsSum := 0.0
		for _, v := range itemVals {
			itemsSum += v
		}

		expected := itemsSum + taxVal - discountVal
		if subtotalVal != nil {
			expected = *subtotalVal + taxVal - discountVal
		}

		diff := math.Round(math.Abs(expected-*totalVal)*100) / 100
		matched := diff < 0.05 || (math.Round(math.Abs(itemsSum-*totalVal)*100)/100 < 0.05)

		return &types.InvoiceValidationResult{
			Matched:    matched,
			Total:      *totalVal,
			Calculated: expected,
			Difference: diff,
			ItemsCount: len(itemVals),
		}
	}

	return nil
}
