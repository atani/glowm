package mermaid

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
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

// renderPage waits for Mermaid and propagates navigation, protocol and parser errors.
func renderPage(ctx context.Context, pageURL string) error {
	if err := chromedp.Do(ctx, chromedp.Navigate(pageURL)); err != nil {
		return err
	}
	if _, err := chromedp.Run(ctx, chromedp.Poll[bool]("window.__MERMAID_DONE__ === true || !!window.__MERMAID_ERROR__", chromedp.WithPollingInterval(100*time.Millisecond))); err != nil {
		return err
	}
	renderErr, err := chromedp.Run(ctx, chromedp.Evaluate[string]("window.__MERMAID_ERROR__"))
	if err != nil {
		return err
	}
	if strings.TrimSpace(renderErr) != "" {
		return fmt.Errorf("mermaid render failed: %s", renderErr)
	}
	return nil
}

// serveHTML starts a loopback HTTP server that serves content as a single page,
// and returns the URL and a cleanup function. Using HTTP avoids file:// access
// restrictions in sandboxed Chrome environments (e.g. snap packages).
func serveHTML(content string) (string, func(), error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, content)
	})
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { _ = srv.Serve(ln) }()
	url := "http://" + ln.Addr().String() + "/"
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}
	return url, cleanup, nil
}
