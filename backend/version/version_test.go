package version

import "testing"

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "2.0.0", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.2.0", "1.10.0", -1}, // numeric, not lexicographic
		{"1.9.0", "1.10.0", -1},
		// A "v" prefix is cosmetic.
		{"v1.0.0", "1.0.0", 0},
		{"1.0.0", "v1.0.1", -1},
		// Shorter versions are zero-padded.
		{"1.2", "1.2.0", 0},
		{"1.2", "1.2.1", -1},
		{"1.2.1", "1.2", 1},
		// Pre-release suffixes are ignored for ordering.
		{"1.0.0-rc1", "1.0.0", 0},
		{"1.0.0", "1.0.1-beta", -1},
		// Whitespace is tolerated.
		{" 1.0.0 ", "1.0.0", 0},
		// Degenerate input must not panic.
		{"", "1.0.0", -1},
		{"1.0.0", "", 1},
		{"", "", 0},
		{"garbage", "1.0.0", -1},
	}

	for _, tc := range tests {
		if got := Compare(tc.a, tc.b); got != tc.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		current, latest string
		want            bool
	}{
		{"1.0.0", "1.0.1", true},
		{"1.0.0", "1.1.0", true},
		{"1.0.0", "2.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.1", "1.0.0", false},
		{"2.0.0", "1.9.9", false},
		{"1.0.0", "v1.0.1", true},
		// The historical bug: len(latest) > len(current) reported a bogus
		// update, so 1.0.0 -> "1.0.0.0" or equal-but-longer tags nagged users.
		{"1.0.0", "1.0.0.0", false},
		{"1.0", "1.0.0", false},
		// A pre-release of the same version is not an upgrade.
		{"1.0.0", "1.0.0-rc2", false},
		// Empty/garbage from a bad API response must not claim an update.
		{"1.0.0", "", false},
		{"1.0.0", "not-a-version", false},
	}

	for _, tc := range tests {
		if got := IsNewer(tc.current, tc.latest); got != tc.want {
			t.Errorf("IsNewer(current=%q, latest=%q) = %v, want %v",
				tc.current, tc.latest, got, tc.want)
		}
	}
}

func TestCurrentIsPopulated(t *testing.T) {
	info := Current()
	if info.Version == "" {
		t.Error("Version is empty")
	}
	if info.GoVersion == "" {
		t.Error("GoVersion is empty")
	}
	if info.OS == "" || info.Arch == "" {
		t.Errorf("OS/Arch incomplete: %+v", info)
	}
}

func TestUserAgentIncludesVersion(t *testing.T) {
	ua := UserAgent()
	if ua == "" {
		t.Fatal("UserAgent is empty")
	}
	if !contains(ua, Version) {
		t.Errorf("UserAgent %q does not include version %q", ua, Version)
	}
	// A User-Agent header value must not contain raw spaces in the product
	// token; the app name is hyphenated for that reason.
	if contains(ua, "ITT OCR") {
		t.Errorf("UserAgent %q contains an unhyphenated product token", ua)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		func() bool {
			for i := 0; i+len(needle) <= len(haystack); i++ {
				if haystack[i:i+len(needle)] == needle {
					return true
				}
			}
			return false
		}()
}
