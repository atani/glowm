package render

import (
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
