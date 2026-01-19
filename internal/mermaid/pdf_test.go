package mermaid

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

func TestRenderPDF(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("skipping chrome-dependent test in CI environment")
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("chrome dependency not supported on this platform")
	}
	if !chromeAvailable() {
		t.Skip("chrome/chromium not available")
	}

	pdf, err := RenderPDF([]string{"flowchart TD\n  A-->B"})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if len(pdf) < 4 || string(pdf[:4]) != "%PDF" {
		t.Fatalf("expected PDF header, got %q", string(pdf[:4]))
	}
}

func chromeAvailable() bool {
	candidates := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return true
		}
	}
	return false
}
