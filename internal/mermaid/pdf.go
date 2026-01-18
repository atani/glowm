package mermaid

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// Mermaid JS v10.9.5
//
//go:embed mermaid.min.js
var mermaidJS string

const (
	paperWidthIn  = 8.27
	paperHeightIn = 11.69
)

func RenderPDF(diagrams []string) ([]byte, error) {
	if len(diagrams) == 0 {
		return nil, errors.New("no mermaid blocks found")
	}

	htmlDoc := buildHTML(diagrams)
	fileURL, cleanup, err := writeTempHTML(htmlDoc)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	ctx, cancel := newBrowserContext(10 * time.Second)
	defer cancel()

	var (
		renderErr string
		pdfBytes  []byte
	)
	if err := chromedp.Run(ctx,
		chromedp.Navigate(fileURL),
		chromedp.Poll("window.__MERMAID_DONE__ === true || !!window.__MERMAID_ERROR__", nil, chromedp.WithPollingInterval(100*time.Millisecond)),
		chromedp.Evaluate("window.__MERMAID_ERROR__", &renderErr),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(paperWidthIn).
				WithPaperHeight(paperHeightIn).
				Do(ctx)
			if err != nil {
				return err
			}
			pdfBytes = buf
			return nil
		}),
	); err != nil {
		return nil, err
	}
	if strings.TrimSpace(renderErr) != "" {
		return nil, fmt.Errorf("mermaid render failed: %s", renderErr)
	}
	return pdfBytes, nil
}

func buildHTML(diagrams []string) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\">\n")
	b.WriteString("<style>body{font-family:Arial,Helvetica,sans-serif;padding:24px;background:#fff;} .mermaid{margin:24px 0;}</style>\n")
	b.WriteString("</head><body>\n")
	for _, diagram := range diagrams {
		b.WriteString("<div class=\"mermaid\">\n")
		b.WriteString(html.EscapeString(diagram))
		b.WriteString("\n</div>\n")
	}
	b.WriteString("<script>\n")
	b.WriteString(mermaidJS)
	b.WriteString("\n</script>\n")
	b.WriteString("<script>\n")
	b.WriteString("window.__MERMAID_DONE__ = false; window.__MERMAID_ERROR__ = '';\n")
	b.WriteString("(async function(){\n")
	b.WriteString("try { mermaid.initialize({ startOnLoad: false }); await mermaid.run({ querySelector: '.mermaid' }); window.__MERMAID_DONE__ = true; }\n")
	b.WriteString("catch(e){ window.__MERMAID_ERROR__ = (e && e.message) ? e.message : String(e); }\n")
	b.WriteString("})();\n")
	b.WriteString("</script>\n")
	b.WriteString("</body></html>")
	return b.String()
}
