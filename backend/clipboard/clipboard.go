package clipboard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// GetClipboardImage extracts an image from system clipboard and saves to temporary file
func GetClipboardImage() (string, error) {
	tempDir := filepath.Join(os.TempDir(), "itt_ocr_scans")
	_ = os.MkdirAll(tempDir, 0755)
	outFile := filepath.Join(tempDir, fmt.Sprintf("clip_%d.png", time.Now().UnixMilli()))

	switch runtime.GOOS {
	case "windows":
		return getClipboardWindows(outFile)
	case "linux":
		return getClipboardLinux(outFile)
	case "darwin":
		return getClipboardDarwin(outFile)
	default:
		return "", fmt.Errorf("unsupported operating system for clipboard image extraction: %s", runtime.GOOS)
	}
}

func getClipboardWindows(outFile string) (string, error) {
	psScript := fmt.Sprintf(`
    Add-Type -AssemblyName System.Windows.Forms
    $clip = [System.Windows.Forms.Clipboard]::GetImage()
    if ($clip -ne $null) {
        $dest = '%s'
        $clip.Save($dest, [System.Drawing.Imaging.ImageFormat]::Png)
        [Console]::Out.WriteLine("SUCCESS")
    } else {
        [Console]::Out.WriteLine("NO_IMAGE")
    }
`, strings.ReplaceAll(outFile, `\`, `\\`))

	cmd := exec.Command("powershell", "-NoProfile", "-STA", "-ExecutionPolicy", "Bypass", "-Command", psScript)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("clipboard read error: %w", err)
	}
	if strings.Contains(string(out), "SUCCESS") {
		if _, err := os.Stat(outFile); err == nil {
			return outFile, nil
		}
	}
	return "", fmt.Errorf("No image found on clipboard")
}

func getClipboardLinux(outFile string) (string, error) {
	// Try wl-paste (Wayland)
	if path, err := exec.LookPath("wl-paste"); err == nil {
		cmd := exec.Command(path, "--type", "image/png")
		out, err := cmd.Output()
		if err == nil && len(out) > 50 {
			if err := os.WriteFile(outFile, out, 0644); err == nil {
				return outFile, nil
			}
		}
	}

	// Try xclip (X11)
	if path, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command(path, "-selection", "clipboard", "-t", "image/png", "-o")
		out, err := cmd.Output()
		if err == nil && len(out) > 50 {
			if err := os.WriteFile(outFile, out, 0644); err == nil {
				return outFile, nil
			}
		}
	}

	// Try xsel
	if path, err := exec.LookPath("xsel"); err == nil {
		cmd := exec.Command(path, "-b")
		out, err := cmd.Output()
		if err == nil && len(out) > 50 && bytesArePNG(out) {
			if err := os.WriteFile(outFile, out, 0644); err == nil {
				return outFile, nil
			}
		}
	}

	return "", fmt.Errorf("No image data found on clipboard or clipboard utilities (wl-paste, xclip) missing.")
}

func getClipboardDarwin(outFile string) (string, error) {
	if path, err := exec.LookPath("pngpaste"); err == nil {
		cmd := exec.Command(path, outFile)
		if err := cmd.Run(); err == nil {
			if _, err := os.Stat(outFile); err == nil {
				return outFile, nil
			}
		}
	}
	return "", fmt.Errorf("No image on clipboard (or pngpaste not installed)")
}

func bytesArePNG(b []byte) bool {
	if len(b) < 8 {
		return false
	}
	return b[0] == 0x89 && b[1] == 'P' && b[2] == 'N' && b[3] == 'G'
}
