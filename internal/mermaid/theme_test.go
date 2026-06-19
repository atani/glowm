package mermaid

import (
	"strings"
	"testing"
)

func TestResolveTheme(t *testing.T) {
	cases := []struct {
		in        string
		wantTheme string
		wantBG    string
	}{
		{"", "default", "#ffffff"},
		{"light", "default", "#ffffff"},
		{"default", "default", "#ffffff"},
		{"dark", "dark", "#1e1e1e"},
		{"DARK", "dark", "#1e1e1e"},
		{" forest ", "forest", "#ffffff"},
		{"neutral", "neutral", "#ffffff"},
		{"base", "base", "#ffffff"},
		{"bogus", "default", "#ffffff"},
	}
	for _, c := range cases {
		gotTheme, gotBG := resolveTheme(c.in)
		if gotTheme != c.wantTheme || gotBG != c.wantBG {
			t.Errorf("resolveTheme(%q) = (%q,%q), want (%q,%q)", c.in, gotTheme, gotBG, c.wantTheme, c.wantBG)
		}
	}
}

func TestIsKnownTheme(t *testing.T) {
	for _, ok := range []string{"", "light", "default", "dark", "forest", "neutral", "base", "DARK"} {
		if !IsKnownTheme(ok) {
			t.Errorf("IsKnownTheme(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"bogus", "midnight", "solarized"} {
		if IsKnownTheme(bad) {
			t.Errorf("IsKnownTheme(%q) = true, want false", bad)
		}
	}
}

func TestBuildMermaidHTML_ThemeAndBackground(t *testing.T) {
	themeName, bg := resolveTheme("dark")
	html, _ := buildMermaidHTML([]string{"A-->B"}, htmlConfig{CSS: pngCSS(bg), Theme: themeName})
	if !strings.Contains(html, "theme: 'dark'") {
		t.Errorf("html missing dark theme in init script")
	}
	if !strings.Contains(html, "background:#1e1e1e") {
		t.Errorf("html missing dark background")
	}
}
