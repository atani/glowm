package mermaid

import (
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html"
	"strings"
)

// Mermaid JS v10.9.5
//
//go:embed mermaid.min.js
var mermaidJS string

func pdfCSS(bg string) string {
	return "body{font-family:Arial,Helvetica,sans-serif;padding:24px;background:" + bg + ";} .mermaid{margin:24px 0;}"
}

func pngCSS(bg string) string {
	return "body{font-family:Arial,Helvetica,sans-serif;padding:24px;background:" + bg + ";width:100%;} .mermaid{margin:24px 0;width:100%;} svg{width:100%;height:auto;font-size:20px;} svg text{font-size:20px !important;} .label{font-size:20px !important;}"
}

// resolveTheme maps a user-supplied theme name to a canonical Mermaid theme and
// a matching page background. Unknown names fall back to the default theme. The
// returned theme is always one of a fixed set of safe literals, so it is safe to
// interpolate into the init script.
func resolveTheme(name string) (theme, background string) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "dark":
		return "dark", "#1e1e1e"
	case "forest":
		return "forest", "#ffffff"
	case "neutral":
		return "neutral", "#ffffff"
	case "base":
		return "base", "#ffffff"
	default: // "", "light", "default", or anything unrecognized
		return "default", "#ffffff"
	}
}

// IsKnownTheme reports whether name is a recognized Mermaid theme (so callers
// can warn on a typo rather than silently falling back to the default). "auto"
// means "detect from the terminal background" and is resolved by the caller
// before reaching resolveTheme.
func IsKnownTheme(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "auto", "light", "default", "dark", "forest", "neutral", "base":
		return true
	default:
		return false
	}
}

type htmlConfig struct {
	AssignIDs bool
	CSS       string
	// Theme is a canonical Mermaid theme name (from resolveTheme); safe to
	// interpolate into the init script.
	Theme string
}

func buildMermaidHTML(diagrams []string, cfg htmlConfig) (string, []string) {
	nonce := generateNonce()
	var b strings.Builder
	var ids []string

	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\">\n")
	fmt.Fprintf(&b, "<meta http-equiv=\"Content-Security-Policy\" content=\"default-src 'none'; script-src 'nonce-%s'; style-src 'unsafe-inline';\">\n", nonce)
	b.WriteString("<style>")
	b.WriteString(cfg.CSS)
	b.WriteString("</style>\n")
	b.WriteString("</head><body>\n")

	for i, diagram := range diagrams {
		if cfg.AssignIDs {
			id := fmt.Sprintf("mmd-%d", i)
			ids = append(ids, id)
			fmt.Fprintf(&b, "<div class=\"mermaid\" id=\"%s\">\n", id)
		} else {
			b.WriteString("<div class=\"mermaid\">\n")
		}
		b.WriteString(html.EscapeString(diagram))
		b.WriteString("\n</div>\n")
	}

	fmt.Fprintf(&b, "<script nonce=\"%s\">\n", nonce)
	b.WriteString(mermaidJS)
	b.WriteString("\n</script>\n")
	fmt.Fprintf(&b, "<script nonce=\"%s\">\n", nonce)
	b.WriteString(mermaidInitScript(cfg.Theme))
	b.WriteString("</script>\n")
	b.WriteString("</body></html>")

	return b.String(), ids
}

func mermaidInitScript(theme string) string {
	if theme == "" {
		theme = "default"
	}
	return "window.__MERMAID_DONE__ = false; window.__MERMAID_ERROR__ = '';\n" +
		"(async function(){\n" +
		"try { mermaid.initialize({ startOnLoad: false, securityLevel: 'strict', theme: '" + theme + "' }); await mermaid.run({ querySelector: '.mermaid' }); window.__MERMAID_DONE__ = true; }\n" +
		"catch(e){ window.__MERMAID_ERROR__ = (e && e.message) ? e.message : String(e); }\n" +
		"})();\n"
}

func generateNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
