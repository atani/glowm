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

	if _, err := chromedp.Call(ctx, emulation.SetDeviceMetricsOverride, emulation.SetDeviceMetricsOverrideParams{
		Width: viewportWidth(widthCells), Height: 900, DeviceScaleFactor: 1, Mobile: false,
	}); err != nil {
		return nil, err
	}
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

	results := make([][]byte, 0, len(ids))
	for _, id := range ids {
		buf, err := chromedp.Run(ctx, chromedp.Screenshot(chromedp.ID(id), chromedp.NodeVisible))
		if err != nil {
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
