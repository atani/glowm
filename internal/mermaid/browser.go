package mermaid

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
)

func newBrowserContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-plugins", true),
		chromedp.Flag("disable-background-networking", true),
	}
	if os.Getenv("GLOWM_CHROME_NO_SANDBOX") != "" || os.Getuid() == 0 {
		fmt.Fprintln(os.Stderr, "glowm: warning: Chrome sandbox disabled")
		opts = append(opts, chromedp.NoSandbox)
	}
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
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

func writeTempHTML(content string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "glowm-*")
	if err != nil {
		return "", nil, err
	}
	path := filepath.Join(dir, "render.html")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		os.RemoveAll(dir)
		return "", nil, err
	}
	fileURL := url.URL{Scheme: "file", Path: path}
	cleanup := func() {
		_ = os.RemoveAll(dir)
	}
	return fileURL.String(), cleanup, nil
}
