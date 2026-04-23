package mermaid

import (
	"context"
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

	var (
		renderErr string
		pollDone  bool
	)
	if err := chromedp.Run(ctx,
		chromedp.Navigate(pageURL),
		chromedp.Poll("window.__MERMAID_DONE__ === true || !!window.__MERMAID_ERROR__", &pollDone, chromedp.WithPollingInterval(100*time.Millisecond)),
		chromedp.Evaluate("window.__MERMAID_ERROR__", &renderErr),
	); err != nil {
		return nil, err
	}
	if strings.TrimSpace(renderErr) != "" {
		return nil, fmt.Errorf("mermaid render failed: %s", renderErr)
	}

	var pdfBytes []byte
	if err := chromedp.Run(ctx,
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
	return pdfBytes, nil
}
