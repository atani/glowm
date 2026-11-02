package mermaid

import (
	"errors"
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

	if err := renderPage(ctx, pageURL); err != nil {
		return nil, err
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
