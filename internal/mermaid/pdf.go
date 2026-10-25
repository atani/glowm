package mermaid

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

const (
	paperWidthIn  = 8.27
	paperHeightIn = 11.69
)

func RenderPDF(diagrams []string) ([]byte, error) {
	if len(diagrams) == 0 {
		return nil, errors.New("no mermaid blocks found")
	}

	htmlDoc, _ := buildMermaidHTML(diagrams, htmlConfig{CSS: pdfCSS})
	pageURL, cleanup, err := serveHTML(htmlDoc)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	ctx, cancel := newBrowserContext(30 * time.Second)
	defer cancel()

	if err := chromedp.Do(ctx, chromedp.Navigate(pageURL)); err != nil {
		return nil, err
	}
	if _, err := chromedp.Run(ctx, chromedp.Poll[bool]("window.__MERMAID_DONE__ === true || !!window.__MERMAID_ERROR__", chromedp.WithPollingInterval(100*time.Millisecond))); err != nil {
		return nil, err
	}
	renderErr, err := chromedp.Run(ctx, chromedp.Evaluate[string]("window.__MERMAID_ERROR__"))
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(renderErr) != "" {
		return nil, fmt.Errorf("mermaid render failed: %s", renderErr)
	}

	result, err := chromedp.Call(ctx, page.PrintToPDF, page.PrintToPDFParams{
		PrintBackground: new(true),
		PaperWidth:      paperWidthIn,
		PaperHeight:     paperHeightIn,
	})
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}
