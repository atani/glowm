package termimage

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"
)

// withStdin temporarily replaces os.Stdin with a file containing the given
// bytes, restoring the original on cleanup. Returns the replacement file.
func withStdin(t *testing.T, data []byte) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	orig := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		os.Stdin = orig
		f.Close()
	})
}

func TestIsSixel_EnvOverride(t *testing.T) {
	resetSixelCache()
	t.Cleanup(resetSixelCache)
	t.Setenv("GLOWM_SIXEL", "1")
	if !isSixel() {
		t.Fatal("expected isSixel()=true when GLOWM_SIXEL=1")
	}
}

func TestIsSixel_Cached(t *testing.T) {
	resetSixelCache()
	t.Cleanup(resetSixelCache)
	t.Setenv("GLOWM_SIXEL", "1")
	if !isSixel() {
		t.Fatal("first call: expected true")
	}
	// Flip the env var; the cached result must still win.
	t.Setenv("GLOWM_SIXEL", "0")
	if !isSixel() {
		t.Fatal("cached isSixel() result should remain true")
	}
}

func TestIsKnownSixelTerminal(t *testing.T) {
	tests := []struct {
		termProgram string
		term        string
		want        bool
	}{
		{"WezTerm", "", true},
		{"", "mlterm", true},
		{"", "foot", true},
		{"", "foot-direct", true},
		{"", "yaft-256color", true},
		{"", "contour", true},
		{"", "xterm-256color", false},
		{"iTerm.app", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.termProgram+"/"+tt.term, func(t *testing.T) {
			t.Setenv("TERM_PROGRAM", tt.termProgram)
			t.Setenv("TERM", tt.term)
			if got := isKnownSixelTerminal(); got != tt.want {
				t.Fatalf("isKnownSixelTerminal() = %v, want %v (TERM_PROGRAM=%q TERM=%q)",
					got, tt.want, tt.termProgram, tt.term)
			}
		})
	}
}

func TestParseSixelSupport(t *testing.T) {
	tests := []struct {
		name string
		resp string
		want bool
	}{
		{
			name: "sixel supported (attribute 4 present)",
			resp: "\x1b[?62;4;6c",
			want: true,
		},
		{
			name: "sixel not supported",
			resp: "\x1b[?62;1;2c",
			want: false,
		},
		{
			name: "only attribute 4",
			resp: "\x1b[?4c",
			want: true,
		},
		{
			name: "attribute 4 at start",
			resp: "\x1b[?4;22;29c",
			want: true,
		},
		{
			name: "attribute 14 must not match as 4",
			resp: "\x1b[?14;22c",
			want: false,
		},
		{
			name: "attribute 40 must not match as 4",
			resp: "\x1b[?40;22c",
			want: false,
		},
		{
			name: "no DA1 response",
			resp: "garbage",
			want: false,
		},
		{
			name: "empty response",
			resp: "",
			want: false,
		},
		{
			name: "no closing c",
			resp: "\x1b[?4;22",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSixelSupport(tt.resp)
			if got != tt.want {
				t.Fatalf("parseSixelSupport(%q) = %v, want %v", tt.resp, got, tt.want)
			}
		})
	}
}

func TestQuerySixelViaDA1_NonTTY(t *testing.T) {
	// Under `go test`, stdin/stdout are typically not TTYs, so the DA1 query
	// must short-circuit to false without any side effects. This guards the
	// early-return path for CI environments.
	if got := querySixelViaDA1(); got != false {
		t.Fatalf("querySixelViaDA1() on non-TTY = %v, want false", got)
	}
}

func TestReadDA1Response_FullResponse(t *testing.T) {
	// A complete DA1 response (terminated by 'c') is read back verbatim.
	withStdin(t, []byte("\x1b[?62;4;6c"))
	resp, ok := readDA1Response(time.Second)
	if !ok {
		t.Fatal("expected ok=true for a terminated response")
	}
	if !bytes.ContainsRune(resp, 'c') {
		t.Errorf("response %q missing terminator", resp)
	}
	if !parseSixelSupport(string(resp)) {
		t.Errorf("expected attribute 4 to be detected in %q", resp)
	}
}

func TestReadDA1Response_EOFWithoutTerminator(t *testing.T) {
	// EOF reached after partial data (no 'c') still returns the buffered bytes.
	withStdin(t, []byte("\x1b[?62;1"))
	resp, ok := readDA1Response(time.Second)
	if !ok {
		t.Fatal("expected ok=true when partial data was buffered before EOF")
	}
	if string(resp) != "\x1b[?62;1" {
		t.Errorf("resp = %q, want the buffered partial bytes", resp)
	}
}

func TestReadDA1Response_EmptyEOF(t *testing.T) {
	// Empty stdin reaches EOF with no data: ok must be false.
	withStdin(t, nil)
	resp, ok := readDA1Response(time.Second)
	if ok {
		t.Fatalf("expected ok=false on empty input, got resp=%q", resp)
	}
}

// Note: the timeout path of readDA1Response is intentionally not unit-tested.
// On timeout the reader goroutine remains blocked on os.Stdin.Read (documented
// in sixel.go), which would race with restoring the swapped os.Stdin under the
// -race detector. The timeout select branch is exercised indirectly in real
// terminals where no DA1 response arrives.

func TestDetectSixelUncached_EnvOverride(t *testing.T) {
	t.Setenv("GLOWM_SIXEL", "1")
	if !detectSixelUncached() {
		t.Fatal("expected true when GLOWM_SIXEL=1")
	}
}

func TestDetectSixelUncached_KnownTerminal(t *testing.T) {
	t.Setenv("GLOWM_SIXEL", "")
	t.Setenv("TERM_PROGRAM", "WezTerm")
	if !detectSixelUncached() {
		t.Fatal("expected true for WezTerm")
	}
}

func TestEncodeSixel_ValidPNG(t *testing.T) {
	// Create a minimal valid PNG image (1x1 red pixel).
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatal(err)
	}

	got := encodeSixel(pngBuf.Bytes())
	if got == "" {
		t.Fatal("expected non-empty Sixel output")
	}
	// Sixel sequences start with DCS (ESC P).
	if !strings.Contains(got, "\x1bP") {
		preview := got
		if len(preview) > 20 {
			preview = preview[:20]
		}
		t.Errorf("expected DCS (ESC P) in Sixel output, got %q", preview)
	}
}

func TestEncodeSixel_InvalidPNG(t *testing.T) {
	got := encodeSixel([]byte("not-a-png"))
	if got != "" {
		t.Fatalf("expected empty string for invalid PNG, got %q", got)
	}
}

func TestEncodeSixel_Nil(t *testing.T) {
	got := encodeSixel(nil)
	if got != "" {
		t.Fatalf("expected empty string for nil input, got %q", got)
	}
}

func TestEncodeWithWidth_Sixel(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatal(err)
	}
	got := EncodeWithWidth(FormatSixel, pngBuf.Bytes(), 0)
	if got == "" {
		t.Fatal("expected non-empty output for FormatSixel")
	}
}

func TestSixelDebugEnabled(t *testing.T) {
	os.Unsetenv("GLOWM_DEBUG_SIXEL")
	if sixelDebugEnabled() {
		t.Fatal("expected false when GLOWM_DEBUG_SIXEL is unset")
	}
	t.Setenv("GLOWM_DEBUG_SIXEL", "1")
	if !sixelDebugEnabled() {
		t.Fatal("expected true when GLOWM_DEBUG_SIXEL=1")
	}
	t.Setenv("GLOWM_DEBUG_SIXEL", "0")
	if sixelDebugEnabled() {
		t.Fatal("expected false when GLOWM_DEBUG_SIXEL=0")
	}
}
