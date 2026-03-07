package mermaid

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestViewportWidth_Zero(t *testing.T) {
	if got := viewportWidth(0); got != 800 {
		t.Fatalf("expected 800 for 0 cells, got %d", got)
	}
}

func TestViewportWidth_Negative(t *testing.T) {
	if got := viewportWidth(-10); got != 800 {
		t.Fatalf("expected 800 for negative cells, got %d", got)
	}
}

func TestViewportWidth_SmallValue(t *testing.T) {
	if got := viewportWidth(50); got != 800 {
		t.Fatalf("expected 800 for small cell count, got %d", got)
	}
}

func TestViewportWidth_Normal(t *testing.T) {
	if got := viewportWidth(120); got != 1080 {
		t.Fatalf("expected 1080 for 120 cells, got %d", got)
	}
}

func TestBuildMermaidHTML_WithIDs(t *testing.T) {
	diagrams := []string{"A-->B", "C-->D"}
	html, ids := buildMermaidHTML(diagrams, htmlConfig{AssignIDs: true, CSS: pngCSS})

	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d", len(ids))
	}
	if ids[0] != "mmd-0" || ids[1] != "mmd-1" {
		t.Fatalf("unexpected IDs: %v", ids)
	}
	if !strings.Contains(html, `id="mmd-0"`) {
		t.Fatal("expected mmd-0 ID in HTML")
	}
	if !strings.Contains(html, `id="mmd-1"`) {
		t.Fatal("expected mmd-1 ID in HTML")
	}
	if !strings.Contains(html, "A--&gt;B") {
		t.Fatal("expected HTML-escaped diagram content")
	}
	if !strings.Contains(html, "securityLevel: 'strict'") {
		t.Fatal("expected securityLevel: 'strict' in mermaid init")
	}
	if !strings.Contains(html, "Content-Security-Policy") {
		t.Fatal("expected CSP meta tag")
	}
	if !strings.Contains(html, "nonce=") {
		t.Fatal("expected nonce attribute on script tags")
	}
	if strings.Contains(html, "'unsafe-inline'") && strings.Contains(html, "script-src") {
		// script-src should use nonce, not unsafe-inline
		if strings.Contains(html, "script-src 'unsafe-inline'") {
			t.Fatal("expected nonce-based CSP, not unsafe-inline for script-src")
		}
	}
}

func TestBuildMermaidHTML_PDF(t *testing.T) {
	diagrams := []string{"X-->Y"}
	html, ids := buildMermaidHTML(diagrams, htmlConfig{CSS: pdfCSS})

	if len(ids) != 0 {
		t.Fatalf("expected 0 IDs for PDF mode, got %d", len(ids))
	}
	if !strings.Contains(html, "X--&gt;Y") {
		t.Fatal("expected HTML-escaped diagram content")
	}
	if !strings.Contains(html, "securityLevel: 'strict'") {
		t.Fatal("expected securityLevel: 'strict' in mermaid init")
	}
	if strings.Contains(html, `id="mmd-`) {
		t.Fatal("PDF mode should not assign element IDs")
	}
	if !strings.Contains(html, "nonce=") {
		t.Fatal("expected nonce attribute on script tags")
	}
}

func TestRenderPNGs_EmptyDiagrams(t *testing.T) {
	_, err := RenderPNGs(nil, 80)
	if err == nil {
		t.Fatal("expected error for empty diagrams")
	}
	if !strings.Contains(err.Error(), "no mermaid blocks found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRenderPNGs(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("chrome dependency not supported on this platform")
	}
	if !chromeAvailableForPNG() {
		t.Skip("chrome/chromium not available")
	}

	pngs, err := RenderPNGs([]string{"flowchart TD\n  A-->B"}, 80)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if len(pngs) != 1 {
		t.Fatalf("expected 1 PNG, got %d", len(pngs))
	}
	if len(pngs[0]) < 4 || string(pngs[0][:4]) != "\x89PNG" {
		t.Fatalf("expected PNG magic bytes")
	}
}

func chromeAvailableForPNG() bool {
	candidates := []string{
		"google-chrome", "google-chrome-stable", "chromium", "chromium-browser",
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
