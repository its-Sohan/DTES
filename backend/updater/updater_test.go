package updater

import "testing"

func TestSelectAssetPrefersMatchingArch(t *testing.T) {
	assets := []Asset{
		{Name: "itt-ocr-1.2.0-linux-amd64.AppImage", DownloadURL: "u-amd64"},
		{Name: "itt-ocr-1.2.0-linux-arm64.AppImage", DownloadURL: "u-arm64"},
		{Name: "itt-ocr-1.2.0-windows-amd64.exe", DownloadURL: "u-win"},
		{Name: "itt-ocr-1.2.0-macos-arm64.dmg", DownloadURL: "u-mac"},
	}

	tests := []struct {
		goos, goarch string
		want         string
	}{
		{"linux", "amd64", "u-amd64"},
		{"linux", "arm64", "u-arm64"},
		{"windows", "amd64", "u-win"},
		{"darwin", "arm64", "u-mac"},
	}

	for _, tc := range tests {
		got, ok := selectAsset(assets, tc.goos, tc.goarch)
		if !ok {
			t.Errorf("selectAsset(%s/%s): no asset chosen", tc.goos, tc.goarch)
			continue
		}
		if got.DownloadURL != tc.want {
			t.Errorf("selectAsset(%s/%s) = %q, want %q", tc.goos, tc.goarch, got.DownloadURL, tc.want)
		}
	}
}

// TestSelectAssetAcceptsArchAliases covers the many spellings release tooling
// uses for the same architecture.
func TestSelectAssetAcceptsArchAliases(t *testing.T) {
	tests := []struct {
		name   string
		goarch string
	}{
		{"itt-ocr-linux-x86_64.AppImage", "amd64"},
		{"itt-ocr-linux-x64.AppImage", "amd64"},
		{"itt-ocr-linux-aarch64.AppImage", "arm64"},
	}

	for _, tc := range tests {
		got, ok := selectAsset([]Asset{{Name: tc.name, DownloadURL: "u"}}, "linux", tc.goarch)
		if !ok || got.DownloadURL != "u" {
			t.Errorf("selectAsset did not match %q for %s", tc.name, tc.goarch)
		}
	}
}

// TestSelectAssetFallsBackToOSMatch covers single-arch releases whose asset
// names omit the architecture entirely.
func TestSelectAssetFallsBackToOSMatch(t *testing.T) {
	assets := []Asset{{Name: "itt-ocr-setup.exe", DownloadURL: "u-win"}}

	got, ok := selectAsset(assets, "windows", "amd64")
	if !ok {
		t.Fatal("expected the OS-only asset to be accepted")
	}
	if got.DownloadURL != "u-win" {
		t.Errorf("got %q, want u-win", got.DownloadURL)
	}
}

// TestSelectAssetHonoursExtensionPreference: a native installer beats a zip.
func TestSelectAssetHonoursExtensionPreference(t *testing.T) {
	assets := []Asset{
		{Name: "itt-ocr-linux-amd64.tar.gz", DownloadURL: "u-targz"},
		{Name: "itt-ocr-linux-amd64.AppImage", DownloadURL: "u-appimage"},
		{Name: "itt-ocr-linux-amd64.deb", DownloadURL: "u-deb"},
	}

	got, ok := selectAsset(assets, "linux", "amd64")
	if !ok {
		t.Fatal("expected an asset")
	}
	if got.DownloadURL != "u-appimage" {
		t.Errorf("got %q, want the AppImage (most preferred)", got.DownloadURL)
	}
}

func TestSelectAssetNoMatch(t *testing.T) {
	// Only Windows assets published, but we are on Linux.
	assets := []Asset{{Name: "itt-ocr-setup.exe", DownloadURL: "u-win"}}
	if _, ok := selectAsset(assets, "linux", "amd64"); ok {
		t.Error("expected no match so the caller falls back to the release page")
	}

	if _, ok := selectAsset(nil, "linux", "amd64"); ok {
		t.Error("expected no match for an empty asset list")
	}

	if _, ok := selectAsset(assets, "plan9", "amd64"); ok {
		t.Error("expected no match for an unsupported OS")
	}
}

// TestSelectAssetIgnoresSourceArchives keeps GitHub's automatic source tarballs
// from being offered as an installer on Linux.
func TestSelectAssetPrefersRealAssetOverSourceArchive(t *testing.T) {
	assets := []Asset{
		{Name: "itt-ocr-1.0.0-linux-amd64.deb", DownloadURL: "u-deb"},
	}
	got, ok := selectAsset(assets, "linux", "amd64")
	if !ok || got.DownloadURL != "u-deb" {
		t.Errorf("got %+v, want the .deb", got)
	}
}

func TestRepoPattern(t *testing.T) {
	valid := []string{
		"its-Sohan/itt-ocr-release",
		"owner/repo",
		"Owner_1/repo.name",
		"a/b",
	}
	invalid := []string{
		"",
		"noslash",
		"owner/repo/extra",
		"owner//repo",
		"../../etc/passwd",
		"owner/repo?query=1",
		"owner/repo#frag",
		"owner repo/x",
		"https://github.com/owner/repo",
	}

	for _, r := range valid {
		if !repoPattern.MatchString(r) {
			t.Errorf("repoPattern rejected valid repo %q", r)
		}
	}
	for _, r := range invalid {
		if repoPattern.MatchString(r) {
			t.Errorf("repoPattern accepted invalid repo %q", r)
		}
	}
}
