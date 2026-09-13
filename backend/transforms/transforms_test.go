package transforms

import (
	"testing"
)

func TestConvertDigits(t *testing.T) {
	bengali := "ইনভয়েস নং: ১২৩৪৫, মোট: ৯৮৭.৫০"
	englishExpected := "ইনভয়েস নং: 12345, মোট: 987.50"

	engResult := ConvertDigitsToEnglish(bengali)
	if engResult != englishExpected {
		t.Fatalf("expected %q, got %q", englishExpected, engResult)
	}

	benResult := ConvertDigitsToBengali(engResult)
	if benResult != bengali {
		t.Fatalf("expected %q, got %q", bengali, benResult)
	}
}

func TestUnwrapBrokenLines(t *testing.T) {
	input := "This is the first line of\na broken sentence.\n\n# Heading Should Remain\n- Item 1\n- Item 2"
	expected := "This is the first line of a broken sentence.\n\n# Heading Should Remain\n- Item 1\n- Item 2"

	res := UnwrapBrokenLines(input)
	if res != expected {
		t.Fatalf("expected:\n%s\ngot:\n%s", expected, res)
	}
}

func TestCleanTableFormatting(t *testing.T) {
	input := "|Item|Qty|Price|\n|---|---|---|\n|Apple|10|100|\n|Banana|5|25|"
	res := CleanTableFormatting(input)
	if res == "" {
		t.Fatalf("expected formatted table, got empty string")
	}
}

func TestCheckInvoiceMath(t *testing.T) {
	table := `| Description | Amount |
|---|---|
| Domain Registration | 15.00 |
| Cloud Hosting | 85.00 |
| Subtotal | 100.00 |
| VAT | 15.00 |
| Grand Total | 115.00 |`

	res := CheckInvoiceMath(table)
	if res == nil {
		t.Fatalf("expected non-nil invoice math result")
	}
	if !res.Matched {
		t.Fatalf("expected matched math, got calculated=%.2f, total=%.2f", res.Calculated, res.Total)
	}
	if res.Total != 115.00 {
		t.Fatalf("expected total 115.00, got %.2f", res.Total)
	}
}
