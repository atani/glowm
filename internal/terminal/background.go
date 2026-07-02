package terminal

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

const bgQueryTimeout = 200 * time.Millisecond

var (
	bgOnce sync.Once
	bgDark bool
	bgOK   bool
)

// BackgroundIsDark reports whether the terminal background is dark, querying it
// once via OSC 11 ("report background color"). The second return value is false
// if the background could not be determined (not a TTY, no response, or an
// unparseable reply), in which case callers should fall back to a light default.
// The result is cached for the process.
func BackgroundIsDark() (dark, ok bool) {
	bgOnce.Do(func() {
		bgDark, bgOK = detectBackground()
	})
	return bgDark, bgOK
}

func detectBackground() (bool, bool) {
	// Querying needs a writable terminal (to send the query) and a readable one
	// (to receive the reply). Mirror the Sixel DA1 probe and use stdout/stdin.
	if !term.IsTerminal(int(os.Stdout.Fd())) || !term.IsTerminal(int(os.Stdin.Fd())) {
		return false, false
	}

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return false, false
	}
	defer term.Restore(fd, oldState) //nolint:errcheck

	// OSC 11 with a "?" asks the terminal to report its background color.
	if _, err := os.Stdout.WriteString("\x1b]11;?\x07"); err != nil {
		return false, false
	}

	resp, ok := readOSCResponse(bgQueryTimeout)
	if !ok {
		return false, false
	}
	r, g, b, ok := parseOSC11Color(string(resp))
	if !ok {
		return false, false
	}
	// Perceptual (Rec. 709) relative luminance; below 0.5 reads as a dark theme.
	lum := 0.2126*r + 0.7152*g + 0.0722*b
	return lum < 0.5, true
}

// readOSCResponse reads from stdin until an OSC terminator (BEL or ST) or the
// timeout. Returns (nil, false) on timeout.
func readOSCResponse(timeout time.Duration) ([]byte, bool) {
	ch := make(chan []byte, 1)
	go func() {
		var buf []byte
		tmp := make([]byte, 64)
		for {
			n, err := os.Stdin.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
				// BEL terminator, or ST (ESC backslash).
				if bytes.IndexByte(buf, 0x07) >= 0 || bytes.Contains(buf, []byte{0x1b, '\\'}) {
					ch <- buf
					return
				}
			}
			if err != nil {
				if len(buf) > 0 {
					ch <- buf
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

// parseOSC11Color parses an OSC 11 reply of the form
// "...rgb:RRRR/GGGG/BBBB<terminator>" into red/green/blue fractions in [0,1].
// Each component may be 1-4 hex digits; per X11 convention it is normalized by
// the maximum value for its digit width (so "ff" and "ffff" both mean 1.0).
func parseOSC11Color(resp string) (r, g, b float64, ok bool) {
	i := strings.Index(resp, "rgb:")
	if i < 0 {
		return 0, 0, 0, false
	}
	s := resp[i+len("rgb:"):]
	if j := strings.IndexAny(s, "\x07\x1b"); j >= 0 {
		s = s[:j]
	}
	parts := strings.Split(s, "/")
	if len(parts) < 3 {
		return 0, 0, 0, false
	}
	conv := func(p string) (float64, bool) {
		p = strings.TrimSpace(p)
		if p == "" || len(p) > 8 {
			return 0, false
		}
		v, err := strconv.ParseUint(p, 16, 64)
		if err != nil {
			return 0, false
		}
		max := float64(uint64(1)<<(4*uint(len(p))) - 1)
		if max <= 0 {
			return 0, false
		}
		return float64(v) / max, true
	}
	var okr, okg, okb bool
	r, okr = conv(parts[0])
	g, okg = conv(parts[1])
	b, okb = conv(parts[2])
	if !okr || !okg || !okb {
		return 0, 0, 0, false
	}
	return r, g, b, true
}
