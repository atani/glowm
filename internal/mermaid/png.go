package mermaid

import (
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

func RenderPNGs(diagrams []string, widthCells int) ([][]byte, error) {
	if len(diagrams) == 0 {
		return nil, errors.New("no mermaid blocks found")
	}

	htmlDoc, ids := buildHTMLWithIDs(diagrams)
	fileURL, cleanup, err := writeTempHTML(htmlDoc)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	ctx, cancel := newBrowserContext(10 * time.Second)
	defer cancel()

	var renderErr string
	if err := chromedp.Run(ctx,
		emulation.SetDeviceMetricsOverride(viewportWidth(widthCells), 900, 1, false),
		chromedp.Navigate(fileURL),
		chromedp.Poll("window.__MERMAID_DONE__ === true || !!window.__MERMAID_ERROR__", nil, chromedp.WithPollingInterval(100*time.Millisecond)),
		chromedp.Evaluate("window.__MERMAID_ERROR__", &renderErr),
	); err != nil {
		return nil, err
	}
	if strings.TrimSpace(renderErr) != "" {
		return nil, fmt.Errorf("mermaid render failed: %s", renderErr)
	}

	results := make([][]byte, 0, len(ids))
	for _, id := range ids {
		var buf []byte
		sel := "#" + id
		if err := chromedp.Run(ctx,
			chromedp.Screenshot(sel, &buf, chromedp.NodeVisible, chromedp.ByID),
		); err != nil {
			return nil, err
		}
		results = append(results, buf)
	}

	return results, nil
}

func buildHTMLWithIDs(diagrams []string) (string, []string) {
	var b strings.Builder
	var ids []string
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\">\n")
	b.WriteString("<style>body{font-family:Arial,Helvetica,sans-serif;padding:24px;background:#fff;width:100%;} .mermaid{margin:24px 0;width:100%;} svg{width:100%;height:auto;font-size:20px;} svg text{font-size:20px !important;} .label{font-size:20px !important;}</style>\n")
	b.WriteString("</head><body>\n")
	for i, diagram := range diagrams {
		id := fmt.Sprintf("mmd-%d", i)
		ids = append(ids, id)
		b.WriteString("<div class=\"mermaid\" id=\"")
		b.WriteString(id)
		b.WriteString("\">\n")
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
	return b.String(), ids
}

func viewportWidth(widthCells int) int64 {
	const minWidth = 800
	if widthCells <= 0 {
		return minWidth
	}
	// rough cell->px estimate; adjust if needed
	px := widthCells * 9
	if px < minWidth {
		px = minWidth
	}
	return int64(px)
}
