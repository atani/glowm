package mermaid

import (
	"context"
	"net/url"
	"os"
	"time"

	"github.com/chromedp/chromedp"
)

func newBrowserContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(),
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.NoSandbox,
	)
	ctx, cancel := chromedp.NewContext(allocCtx)
	if timeout > 0 {
		ctxWithTimeout, timeoutCancel := context.WithTimeout(ctx, timeout)
		return ctxWithTimeout, func() {
			timeoutCancel()
			cancel()
			allocCancel()
		}
	}
	return ctx, func() {
		cancel()
		allocCancel()
	}
}

func writeTempHTML(html string) (string, func(), error) {
	tmp, err := os.CreateTemp("", "glowm-*.html")
	if err != nil {
		return "", nil, err
	}
	if _, err := tmp.WriteString(html); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", nil, err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", nil, err
	}
	fileURL := url.URL{Scheme: "file", Path: tmp.Name()}
	cleanup := func() {
		_ = os.Remove(tmp.Name())
	}
	return fileURL.String(), cleanup, nil
}
