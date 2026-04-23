package termimage

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-sixel"
	"golang.org/x/term"
)

const da1QueryTimeout = 200 * time.Millisecond

var (
	sixelOnce    sync.Once
	sixelSupport bool
)

// isSixel returns true if the terminal supports Sixel graphics.
// The result is cached across calls via sync.Once so the DA1 query (up to
// ~200ms) runs at most once per process.
// Detection order:
//  1. GLOWM_SIXEL=1 env var forces Sixel mode.
//  2. Known Sixel-capable terminal env vars (TERM_PROGRAM, TERM).
//  3. DA1 (Primary Device Attributes) terminal query for capability 4.
func isSixel() bool {
	sixelOnce.Do(func() {
		sixelSupport = detectSixelUncached()
	})
	return sixelSupport
}

// resetSixelCache clears the cached isSixel result. Intended for tests that
// flip GLOWM_SIXEL or terminal env vars between cases.
func resetSixelCache() {
	sixelOnce = sync.Once{}
	sixelSupport = false
}

func detectSixelUncached() bool {
	if os.Getenv("GLOWM_SIXEL") == "1" {
		return true
	}
	if isKnownSixelTerminal() {
		return true
	}
	return querySixelViaDA1()
}

// isKnownSixelTerminal checks environment variables for terminals known to
// support Sixel that may not reliably advertise it via DA1.
func isKnownSixelTerminal() bool {
	switch os.Getenv("TERM_PROGRAM") {
	case "WezTerm":
		return true
	}
	switch os.Getenv("TERM") {
	case "mlterm", "yaft-256color", "foot", "foot-direct", "contour":
		return true
	}
	return false
}

// querySixelViaDA1 queries the terminal via DA1 (Primary Device Attributes)
// and returns true if the terminal reports Sixel capability (attribute 4).
func querySixelViaDA1() bool {
	debug := sixelDebugEnabled()
	stdoutTTY := term.IsTerminal(int(os.Stdout.Fd()))
	stdinTTY := term.IsTerminal(int(os.Stdin.Fd()))
	if debug {
		fmt.Fprintf(os.Stderr, "glowm: stdout TTY=%v stdin TTY=%v\n", stdoutTTY, stdinTTY)
	}
	if !stdoutTTY || !stdinTTY {
		return false
	}

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return false
	}
	defer term.Restore(fd, oldState) //nolint:errcheck

	if _, err := os.Stdout.WriteString("\x1b[c"); err != nil {
		return false
	}

	// NOTE: If the terminal never sends a DA1 response, the reader goroutine
	// below remains blocked on os.Stdin.Read. Because isSixel() caches its
	// result via sync.Once, at most one such goroutine can leak per process.
	resp, ok := readDA1Response(da1QueryTimeout)
	if !ok {
		if debug {
			fmt.Fprintln(os.Stderr, "glowm: DA1 query timed out (no response within 200ms)")
		}
		return false
	}
	if debug {
		fmt.Fprintf(os.Stderr, "glowm: DA1 response (%d bytes): %q\n", len(resp), resp)
	}
	return parseSixelSupport(string(resp))
}

// readDA1Response reads from os.Stdin until it sees the 'c' terminator of a
// DA1 response or the given timeout elapses. Returns (nil, false) on timeout.
func readDA1Response(timeout time.Duration) ([]byte, bool) {
	ch := make(chan []byte, 1)
	go func() {
		var responseBuf []byte
		buf := make([]byte, 64)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				responseBuf = append(responseBuf, buf[:n]...)
				if bytes.ContainsRune(responseBuf, 'c') {
					ch <- responseBuf
					return
				}
			}
			if err != nil {
				if len(responseBuf) > 0 {
					ch <- responseBuf
				} else {
					ch <- nil
				}
				return
			}
		}
	}()

	select {
	case resp := <-ch:
		return resp, resp != nil
	case <-time.After(timeout):
		return nil, false
	}
}

// parseSixelSupport checks the DA1 response for Sixel capability (attribute 4).
// Expected format: ESC [ ? P1 ; P2 ; ... c
func parseSixelSupport(resp string) bool {
	start := strings.Index(resp, "\x1b[?")
	if start == -1 {
		return false
	}
	end := strings.Index(resp[start:], "c")
	if end == -1 {
		return false
	}
	params := resp[start+3 : start+end]
	for _, p := range strings.Split(params, ";") {
		if p == "4" {
			return true
		}
	}
	return false
}

// encodeSixel converts PNG data to a Sixel image escape sequence.
// Note: widthCells is intentionally not accepted; go-sixel sizes output by
// source pixels. Scaling to a target cell count would require pre-resizing
// the source image before encoding (not yet implemented).
func encodeSixel(png []byte) string {
	debug := sixelDebugEnabled()
	img, _, err := image.Decode(bytes.NewReader(png))
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "glowm: Sixel decode failed: %v\n", err)
		}
		return ""
	}
	var buf bytes.Buffer
	enc := sixel.NewEncoder(&buf)
	if err := enc.Encode(img); err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "glowm: Sixel encode failed: %v\n", err)
		}
		return ""
	}
	return buf.String()
}

func sixelDebugEnabled() bool {
	return os.Getenv("GLOWM_DEBUG_SIXEL") == "1"
}
