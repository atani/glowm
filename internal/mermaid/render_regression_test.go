package mermaid

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRenderPageProtocolErrors(t *testing.T) {
	if !chromeAvailable() {
		t.Skip("chrome/chromium not available")
	}
	for _, tc := range []struct{ name, script, want string }{
		{"poll", `Object.defineProperty(window, '__MERMAID_DONE__', {get() {throw new Error('poll failed')}})`, "poll failed"},
		{"evaluate", `window.__MERMAID_DONE__ = true; Object.defineProperty(window, '__MERMAID_ERROR__', {get() {throw new Error('evaluate failed')}})`, "evaluate failed"},
		{"decode", `window.__MERMAID_DONE__ = true; window.__MERMAID_ERROR__ = 42`, "string"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			url, cleanup, err := serveHTML("<script>" + tc.script + "</script>")
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()
			ctx, cancel := newBrowserContext(5 * time.Second)
			defer cancel()
			if err := renderPage(ctx, url); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

// Exercise the typed chromedp result APIs with both valid and invalid input.
func TestRenderInvalidDiagram(t *testing.T) {
	if !chromeAvailable() {
		t.Skip("chrome/chromium not available")
	}
	for _, format := range []string{"PDF", "PNG"} {
		t.Run(format, func(t *testing.T) {
			diagrams := []string{"this is not a mermaid diagram"}
			var err error
			if format == "PDF" {
				_, err = RenderPDF(diagrams)
			} else {
				_, err = RenderPNGs(diagrams, 80)
			}
			if err == nil || !strings.Contains(err.Error(), "mermaid render failed:") {
				t.Fatalf("expected mermaid parser error, got %v", err)
			}
		})
	}
}

func TestRenderPNGsMultipleDiagrams(t *testing.T) {
	if !chromeAvailable() {
		t.Skip("chrome/chromium not available")
	}
	images, err := RenderPNGs([]string{"flowchart TD\n A-->B", "flowchart LR\n C-->D"}, 120)
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 2 {
		t.Fatalf("got %d images, want 2", len(images))
	}
	for i, data := range images {
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("image %d is not a valid PNG: %v", i, err)
		}
		if img.Bounds().Dx() == 0 || img.Bounds().Dy() == 0 {
			t.Fatalf("image %d is empty", i)
		}
	}
}

func TestRenderBrowserStartupFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test executable requires a POSIX shell")
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "chromium"), []byte("#!/bin/sh\necho intentional browser startup failure >&2\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	for _, format := range []string{"PDF", "PNG"} {
		t.Run(format, func(t *testing.T) {
			diagrams := []string{"flowchart TD\n A-->B"}
			var err error
			if format == "PDF" {
				_, err = RenderPDF(diagrams)
			} else {
				_, err = RenderPNGs(diagrams, 80)
			}
			if err == nil || !strings.Contains(err.Error(), "intentional browser startup failure") {
				t.Fatalf("expected browser startup error, got %v", err)
			}
		})
	}
}
