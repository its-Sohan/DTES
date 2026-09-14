// Package version is the single source of truth for the application's identity.
//
// Version is intended to be overridden at build time so that a release binary
// reports the tag it was built from rather than a hardcoded constant:
//
//	go build -ldflags "-X itt-ocr/backend/version.Version=1.2.3"
//
// The Makefile and CI pipeline both derive the value from `git describe`.
package version

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// Values injected via -ldflags. The defaults describe a local dev build.
var (
	// Version is the semantic version, without a leading "v".
	Version = "0.1.3"
	// Commit is the short git SHA the binary was built from.
	Commit = "unknown"
	// BuildDate is an RFC 3339 timestamp of the build.
	BuildDate = "unknown"
)

// AppName is the product name shown in the UI and window title.
const AppName = "ITT OCR"

// UserAgent identifies this client to the update and vision endpoints.
func UserAgent() string {
	return fmt.Sprintf("%s/%s (%s; %s)",
		strings.ReplaceAll(AppName, " ", "-"), Version, runtime.GOOS, runtime.GOARCH)
}

// Info describes the running build, surfaced in the UI and bug reports.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Current returns the build information for this binary.
func Current() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// Compare orders two dotted numeric version strings, ignoring any leading "v"
// and any pre-release suffix such as "-rc1". It returns -1 if a < b, 0 if they
// are equal and +1 if a > b.
//
// Versions of differing length compare as if the shorter were zero-padded, so
// "1.2" == "1.2.0" and "1.2" < "1.2.1".
func Compare(a, b string) int {
	as, bs := parse(a), parse(b)

	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}

	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		switch {
		case av < bv:
			return -1
		case av > bv:
			return 1
		}
	}
	return 0
}

// IsNewer reports whether latest is strictly newer than current.
func IsNewer(current, latest string) bool {
	return Compare(latest, current) > 0
}

// parse splits a version string into its numeric components, tolerating a "v"
// prefix, surrounding whitespace and a pre-release/build suffix.
func parse(v string) []int {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")

	// Drop any pre-release or build metadata: 1.2.3-rc1+abc -> 1.2.3
	if i := strings.IndexAny(v, "-+ "); i >= 0 {
		v = v[:i]
	}
	if v == "" {
		return nil
	}

	fields := strings.Split(v, ".")
	out := make([]int, 0, len(fields))
	for _, f := range fields {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			// Stop at the first non-numeric component rather than guessing.
			break
		}
		out = append(out, n)
	}
	return out
}
