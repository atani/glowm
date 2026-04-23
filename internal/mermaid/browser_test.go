package mermaid

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestServeHTML_ServesContent(t *testing.T) {
	const body = "<html><body>hello</body></html>"
	url, cleanup, err := serveHTML(body)
	if err != nil {
		t.Fatalf("serveHTML: %v", err)
	}
	defer cleanup()

	if !strings.HasPrefix(url, "http://127.0.0.1:") {
		t.Fatalf("expected loopback URL, got %q", url)
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("http GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html prefix", ct)
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(got) != body {
		t.Errorf("body = %q, want %q", got, body)
	}
}

func TestServeHTML_CleanupShutsDownServer(t *testing.T) {
	url, cleanup, err := serveHTML("ok")
	if err != nil {
		t.Fatalf("serveHTML: %v", err)
	}

	// Prove it was up first.
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		cleanup()
		t.Fatalf("pre-cleanup GET failed: %v", err)
	}
	resp.Body.Close()

	cleanup()

	// After cleanup the server must reject further connections.
	done := make(chan error, 1)
	go func() {
		_, err := client.Get(url)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected GET after cleanup to fail")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("GET after cleanup hung; server still accepting connections")
	}
}

func TestServeHTML_ServesSameContentOnAnyPath(t *testing.T) {
	const body = "payload"
	url, cleanup, err := serveHTML(body)
	if err != nil {
		t.Fatalf("serveHTML: %v", err)
	}
	defer cleanup()

	client := &http.Client{Timeout: 3 * time.Second}
	// Any path under the server should return the same content, mirroring
	// how Chrome may fetch favicons or relative assets during navigation.
	for _, path := range []string{"", "favicon.ico", "assets/x.js"} {
		resp, err := client.Get(url + path)
		if err != nil {
			t.Fatalf("GET %q: %v", path, err)
		}
		got, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(got) != body {
			t.Errorf("GET %q: body = %q, want %q", path, got, body)
		}
	}
}
