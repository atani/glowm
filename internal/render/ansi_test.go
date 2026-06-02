package render

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestANSI_URLNotBroken(t *testing.T) {
	// Long URL that would be broken by word wrap if not handled properly
	longURL := "https://github.com/charmbracelet/glamour/issues/149#issuecomment-1234567890"
	md := "Check this link: " + longURL + " for more info."

	output, err := ANSI(md, RenderOptions{
		Width: 40, // Narrow width to force potential wrapping
		Style: "notty",
		TTY:   false,
	})
	if err != nil {
		t.Fatalf("ANSI() error: %v", err)
	}

	// The URL should appear intact (not split across lines)
	// With OSC 8 support, URLs are wrapped in escape sequences
	if !strings.Contains(output, longURL) {
		// Check if URL was broken by newlines
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "github.com") && !strings.Contains(line, longURL) {
				t.Errorf("URL appears to be broken across lines.\nExpected URL intact: %s\nGot output:\n%s", longURL, output)
				return
			}
		}
	}
}

func TestANSI_MarkdownLinkNotBroken(t *testing.T) {
	longURL := "https://example.com/very/long/path/that/would/normally/be/wrapped/by/word/wrap"
	md := "[Click here](" + longURL + ") to visit."

	output, err := ANSI(md, RenderOptions{
		Width: 40,
		Style: "notty",
		TTY:   false,
	})
	if err != nil {
		t.Fatalf("ANSI() error: %v", err)
	}

	// Verify the URL is present and not broken
	if !strings.Contains(output, longURL) {
		lines := strings.Split(output, "\n")
		urlParts := 0
		for _, line := range lines {
			if strings.Contains(line, "example.com") || strings.Contains(line, "/very/long/") {
				urlParts++
			}
		}
		if urlParts > 1 {
			t.Errorf("URL appears to be broken across multiple lines.\nExpected URL intact: %s\nGot output:\n%s", longURL, output)
		}
	}
}

func TestANSI_MultipleURLsNotBroken(t *testing.T) {
	md := `Here are some links:
- [Link1](https://github.com/charmbracelet/glamour/pull/411)
- [Link2](https://github.com/charmbracelet/glow/issues/286)
`

	output, err := ANSI(md, RenderOptions{
		Width: 50,
		Style: "notty",
		TTY:   false,
	})
	if err != nil {
		t.Fatalf("ANSI() error: %v", err)
	}

	// Both URLs should be present
	if !strings.Contains(output, "glamour/pull/411") {
		t.Errorf("First URL missing or broken in output:\n%s", output)
	}
	if !strings.Contains(output, "glow/issues/286") {
		t.Errorf("Second URL missing or broken in output:\n%s", output)
	}
}

func TestFormatError_Nil(t *testing.T) {
	if got := FormatError(nil); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestFormatError_WithError(t *testing.T) {
	got := FormatError(errors.New("boom"))
	if got != "error: boom" {
		t.Errorf("expected 'error: boom', got %q", got)
	}
}

func TestANSI_DarkStyle(t *testing.T) {
	md := "# Hello\n\nWorld\n"
	output, err := ANSI(md, RenderOptions{Width: 80, Style: "dark", TTY: true})
	if err != nil {
		t.Fatalf("ANSI() error: %v", err)
	}
	if output == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(output, "Hello") {
		t.Fatal("expected heading text in output")
	}
}

func TestANSI_LightStyle(t *testing.T) {
	md := "# Hello\n\nWorld\n"
	output, err := ANSI(md, RenderOptions{Width: 80, Style: "light", TTY: true})
	if err != nil {
		t.Fatalf("ANSI() error: %v", err)
	}
	if output == "" {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(output, "Hello") {
		t.Fatal("expected heading text in output")
	}
}

func TestANSI_InvalidStyleFile(t *testing.T) {
	md := "# Hello\n"
	_, err := ANSI(md, RenderOptions{Width: 80, Style: "/nonexistent/style.json", TTY: true})
	if err == nil {
		t.Fatal("expected error for invalid style file path")
	}
}

func TestANSI_ZeroWidth(t *testing.T) {
	md := "# Hello\n\nWorld\n"
	output, err := ANSI(md, RenderOptions{Width: 0, Style: "notty", TTY: false})
	if err != nil {
		t.Fatalf("ANSI() error: %v", err)
	}
	if output == "" {
		t.Fatal("expected non-empty output")
	}
}

func TestANSI_AutoStyleWithTTY(t *testing.T) {
	// auto/empty style with TTY=true exercises the termenv background
	// detection branch (dark vs light) and the heading-prefix stripping.
	md := "# Hello\n\nWorld\n"
	for _, style := range []string{"", "auto"} {
		t.Run("style="+style, func(t *testing.T) {
			output, err := ANSI(md, RenderOptions{Width: 80, Style: style, TTY: true})
			if err != nil {
				t.Fatalf("ANSI() error: %v", err)
			}
			if !strings.Contains(output, "Hello") {
				t.Errorf("expected heading text in output, got %q", output)
			}
			// withoutHeadingPrefix removes the leading "# " glamour prefix.
			if strings.Contains(output, "# Hello") {
				t.Errorf("heading prefix should be stripped, got %q", output)
			}
		})
	}
}

func TestANSI_StandardNamedStyles(t *testing.T) {
	md := "# Title\n\nbody text\n"
	for _, style := range []string{"notty", "ascii", "dracula", "pink"} {
		t.Run(style, func(t *testing.T) {
			output, err := ANSI(md, RenderOptions{Width: 80, Style: style, TTY: true})
			if err != nil {
				t.Fatalf("ANSI(style=%q) error: %v", style, err)
			}
			if !strings.Contains(output, "Title") {
				t.Errorf("style %q output missing title: %q", style, output)
			}
		})
	}
}

func TestANSI_ValidStyleFile(t *testing.T) {
	// A well-formed style JSON file must be loaded via WithStylesFromJSONFile.
	dir := t.TempDir()
	stylePath := filepath.Join(dir, "style.json")
	if err := os.WriteFile(stylePath, []byte(`{"document":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	md := "# Hello\n\nWorld\n"
	output, err := ANSI(md, RenderOptions{Width: 80, Style: stylePath, TTY: true})
	if err != nil {
		t.Fatalf("ANSI() with valid style file error: %v", err)
	}
	if !strings.Contains(output, "Hello") {
		t.Errorf("expected heading text in output, got %q", output)
	}
}
