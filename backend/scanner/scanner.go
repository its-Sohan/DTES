package scanner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ScanDocument triggers a document scan.
// Returns (path, error).
func ScanDocument() (string, error) {
	if runtime.GOOS == "windows" {
		return scanWindows()
	}

	// Dev / simulated scan mode
	if os.Getenv("ITT_OCR_ALLOW_SIMULATED_SCAN") == "1" {
		tempDir := filepath.Join(os.TempDir(), "itt_ocr_scans")
		_ = os.MkdirAll(tempDir, 0755)
		outPath := filepath.Join(tempDir, fmt.Sprintf("simulated_scan_%d.png", time.Now().Unix()))
		// Minimal valid PNG header
		minimalPNG := []byte{
			0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
			0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x64, 0x00, 0x00, 0x00, 0x64,
			0x08, 0x02, 0x00, 0x00, 0x00, 0xff, 0x80, 0x02, 0x03, 0x00, 0x00, 0x00,
			0x1b, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0xfc, 0xcf, 0x80, 0x01,
			0x00, 0x00, 0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00,
			0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
		}
		if err := os.WriteFile(outPath, minimalPNG, 0644); err != nil {
			return "", err
		}
		return outPath, nil
	}

	return "", fmt.Errorf("Direct hardware scanning requires Windows (WIA). On Linux/macOS, please scan with your scanner's native utility, then load the file via Browse or Paste.")
}

func scanWindows() (string, error) {
	tempDir := filepath.Join(os.TempDir(), "itt_ocr_scans")
	_ = os.MkdirAll(tempDir, 0755)
	outPath := filepath.Join(tempDir, fmt.Sprintf("scan_%d.jpg", time.Now().Unix()))

	psScript := fmt.Sprintf(`
    $ErrorActionPreference = 'Stop'
    try {
        $dialog = New-Object -ComObject WIA.CommonDialog
        $jpgFormat = '{B96B3CAE-0728-11D3-9D7B-0000F81EF32E}'
        $image = $null

        try {
            $image = $dialog.ShowAcquireImage(1, 0, 131072, $jpgFormat, $true, $true, $false)
        } catch {
            try {
                $image = $dialog.ShowAcquireImage(0, 0, 131072, $jpgFormat, $true, $true, $false)
            } catch {
                [Console]::Out.WriteLine("STATUS:NO_DEVICE:No WIA hardware scanner is currently connected.")
                exit 0
            }
        }

        if ($image -ne $null) {
            $dest = '%s'
            if (Test-Path $dest) { Remove-Item -Force $dest }
            $image.SaveFile($dest)
            [Console]::Out.WriteLine("STATUS:SUCCESS:$dest")
        } else {
            [Console]::Out.WriteLine("STATUS:CANCELLED:Scan prompt was cancelled.")
        }
    } catch {
        [Console]::Out.WriteLine("STATUS:ERROR:" + $_.Exception.Message)
    }
`, strings.ReplaceAll(outPath, `\`, `\\`))

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", psScript)
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("PowerShell execution failed: %w", err)
	}

	lines := strings.Split(string(outputBytes), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "STATUS:SUCCESS:") {
			dest := strings.TrimPrefix(trimmed, "STATUS:SUCCESS:")
			if _, err := os.Stat(dest); err == nil {
				return dest, nil
			}
		}
		if strings.HasPrefix(trimmed, "STATUS:CANCELLED:") {
			return "", fmt.Errorf("Scan cancelled by user")
		}
		if strings.HasPrefix(trimmed, "STATUS:NO_DEVICE:") {
			return "", fmt.Errorf("%s", strings.TrimPrefix(trimmed, "STATUS:NO_DEVICE:"))
		}
		if strings.HasPrefix(trimmed, "STATUS:ERROR:") {
			return "", fmt.Errorf("%s", strings.TrimPrefix(trimmed, "STATUS:ERROR:"))
		}
	}

	if _, err := os.Stat(outPath); err == nil {
		return outPath, nil
	}
	return "", fmt.Errorf("Scanner returned no output")
}
