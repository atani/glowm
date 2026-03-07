package mermaid

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestRenderPDF(t *testing.T) {
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

func TestRenderPDF_MultipleDiagrams(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("chrome dependency not supported on this platform")
	}
	if !chromeAvailable() {
		t.Skip("chrome/chromium not available")
	}

	pdf, err := RenderPDF([]string{"flowchart TD\n  A-->B", "flowchart TD\n  C-->D"})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if len(pdf) < 4 || string(pdf[:4]) != "%PDF" {
		t.Fatalf("expected PDF header, got %q", string(pdf[:4]))
	}
}

func TestRenderPDF_EmptyDiagrams(t *testing.T) {
	_, err := RenderPDF(nil)
	if err == nil {
		t.Fatal("expected error for empty diagrams")
	}
	if !strings.Contains(err.Error(), "no mermaid blocks found") {
		t.Fatalf("unexpected error: %v", err)
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
