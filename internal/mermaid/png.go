package mermaid

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
)

func RenderPNGs(diagrams []string, widthCells int) ([][]byte, error) {
	if len(diagrams) == 0 {
		return nil, errors.New("no mermaid blocks found")
	}

	htmlDoc, ids := buildMermaidHTML(diagrams, htmlConfig{AssignIDs: true, CSS: pngCSS})
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
		emulation.SetDeviceMetricsOverride(viewportWidth(widthCells), 900, 1, false),
		chromedp.Navigate(pageURL),
		chromedp.Poll("window.__MERMAID_DONE__ === true || !!window.__MERMAID_ERROR__", &pollDone, chromedp.WithPollingInterval(100*time.Millisecond)),
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

// viewportWidth converts terminal cell width to browser viewport pixels.
// Uses a 9 px/cell estimate. Minimum 800px.
func viewportWidth(widthCells int) int64 {
	const minWidth = 800
	if widthCells <= 0 {
		return minWidth
	}
	px := widthCells * 9
	if px < minWidth {
		px = minWidth
	}
	return int64(px)
}
