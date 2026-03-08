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

const (
	pdfCSS = "body{font-family:Arial,Helvetica,sans-serif;padding:24px;background:#fff;} .mermaid{margin:24px 0;}"
	pngCSS = "body{font-family:Arial,Helvetica,sans-serif;padding:24px;background:#fff;width:100%;} .mermaid{margin:24px 0;width:100%;} svg{width:100%;height:auto;font-size:20px;} svg text{font-size:20px !important;} .label{font-size:20px !important;}"
)

type htmlConfig struct {
	AssignIDs bool
	CSS       string
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
	b.WriteString(mermaidInitScript())
	b.WriteString("</script>\n")
	b.WriteString("</body></html>")

	return b.String(), ids
}

func mermaidInitScript() string {
	return "window.__MERMAID_DONE__ = false; window.__MERMAID_ERROR__ = '';\n" +
		"(async function(){\n" +
		"try { mermaid.initialize({ startOnLoad: false, securityLevel: 'strict' }); await mermaid.run({ querySelector: '.mermaid' }); window.__MERMAID_DONE__ = true; }\n" +
		"catch(e){ window.__MERMAID_ERROR__ = (e && e.message) ? e.message : String(e); }\n" +
		"})();\n"
}

func generateNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
