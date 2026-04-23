package termimage

import (
	"encoding/base64"
	"strings"
	"testing"
)

// termEnv captures the subset of environment variables Detect() consults.
// Zero values clear the variable for the test scope.
type termEnv struct {
	termProgram         string
	kittyWindowID       string
	term                string
	ghosttyResourcesDir string
	tmux                string
	glowmSixel          string
}

// setTermEnv resets the Detect/isSixel caches and applies e for the current
// test, restoring caches automatically when the test finishes.
func setTermEnv(t *testing.T, e termEnv) {
	t.Helper()
	resetDetectCache()
	resetSixelCache()
	t.Cleanup(func() {
		resetDetectCache()
		resetSixelCache()
	})
	t.Setenv("TERM_PROGRAM", e.termProgram)
	t.Setenv("KITTY_WINDOW_ID", e.kittyWindowID)
	t.Setenv("TERM", e.term)
	t.Setenv("GHOSTTY_RESOURCES_DIR", e.ghosttyResourcesDir)
	t.Setenv("TMUX", e.tmux)
	t.Setenv("GLOWM_SIXEL", e.glowmSixel)
}

func TestEncode_FormatNone(t *testing.T) {
	got := Encode(FormatNone, []byte("png-data"))
	if got != "" {
		t.Fatalf("expected empty string for FormatNone, got %q", got)
	}
}

func TestEncodeIterm2_Basic(t *testing.T) {
	data := []byte("fake-png")
	got := encodeIterm2(data, 0)
	if !strings.HasPrefix(got, "\x1b]1337;File=") {
		t.Fatalf("expected iTerm2 OSC prefix, got %q", got[:20])
	}
	if !strings.HasSuffix(got, "\x07") {
		t.Fatalf("expected BEL terminator")
	}
	if !strings.Contains(got, "inline=1") {
		t.Fatal("expected inline=1")
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	if !strings.Contains(got, b64) {
		t.Fatal("expected base64 payload")
	}
}

func TestEncodeIterm2_WithWidth(t *testing.T) {
	got := encodeIterm2([]byte("x"), 42)
	if !strings.Contains(got, "width=42") {
		t.Fatalf("expected width=42 in output, got %q", got)
	}
}

func TestEncodeKitty_SingleChunk(t *testing.T) {
	data := []byte("small")
	got := encodeKitty(data, 0)
	if !strings.HasPrefix(got, "\x1b_G") {
		t.Fatal("expected Kitty APC prefix")
	}
	if !strings.Contains(got, "f=100,a=T,") {
		t.Fatal("expected f=100,a=T in first chunk")
	}
	if !strings.Contains(got, "m=0;") {
		t.Fatal("expected m=0 for single/last chunk")
	}
	if !strings.HasSuffix(got, "\x1b\\") {
		t.Fatal("expected ST terminator")
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	if !strings.Contains(got, b64) {
		t.Fatal("expected base64 payload")
	}
}

func TestEncodeKitty_MultiChunk(t *testing.T) {
	// Generate data that will produce >4096 bytes of base64.
	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i % 256)
	}
	got := encodeKitty(data, 80)

	chunks := strings.Split(got, "\x1b\\")
	// Last element after final split is empty.
	chunks = chunks[:len(chunks)-1]
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// First chunk should have full control params.
	if !strings.Contains(chunks[0], "f=100,a=T,") {
		t.Fatalf("expected f=100,a=T in first chunk: %q", chunks[0][:60])
	}
	if !strings.Contains(chunks[0], "c=80,") {
		t.Fatalf("expected c=80 in first chunk: %q", chunks[0][:60])
	}
	if !strings.Contains(chunks[0], "m=1;") {
		t.Fatalf("expected m=1 in first chunk")
	}

	// Subsequent chunks should NOT have f= or a=.
	for i := 1; i < len(chunks); i++ {
		if strings.Contains(chunks[i], "f=100") {
			t.Fatalf("chunk %d should not have f=100: %q", i, chunks[i][:40])
		}
	}

	// Last chunk should have m=0.
	last := chunks[len(chunks)-1]
	if !strings.Contains(last, "m=0;") {
		t.Fatalf("expected m=0 in last chunk: %q", last[:40])
	}
}

func TestEncodeKitty_WithWidth(t *testing.T) {
	got := encodeKitty([]byte("x"), 100)
	if !strings.Contains(got, "c=100,") {
		t.Fatalf("expected c=100 in output, got %q", got)
	}
}

func TestEncodeWithWidth_RoundTrip(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header bytes
	for _, format := range []Format{FormatIterm2, FormatKitty} {
		got := EncodeWithWidth(format, data, 0)
		if got == "" {
			t.Fatalf("expected non-empty output for format %d", format)
		}
	}
}

func TestDetect_ITerm2(t *testing.T) {
	setTermEnv(t, termEnv{termProgram: "iTerm.app", term: "xterm-256color"})
	if got := Detect(); got != FormatIterm2 {
		t.Fatalf("expected FormatIterm2, got %d", got)
	}
}

func TestDetect_Kitty(t *testing.T) {
	setTermEnv(t, termEnv{kittyWindowID: "1"})
	if got := Detect(); got != FormatKitty {
		t.Fatalf("expected FormatKitty, got %d", got)
	}
}

func TestDetect_KittyByTerm(t *testing.T) {
	setTermEnv(t, termEnv{term: "xterm-kitty"})
	if got := Detect(); got != FormatKitty {
		t.Fatalf("expected FormatKitty, got %d", got)
	}
}

func TestDetect_Ghostty(t *testing.T) {
	setTermEnv(t, termEnv{termProgram: "ghostty", term: "xterm-ghostty"})
	if got := Detect(); got != FormatKitty {
		t.Fatalf("expected FormatKitty for Ghostty, got %d", got)
	}
}

func TestDetect_GhosttyByTerm(t *testing.T) {
	// Some setups expose Ghostty only via TERM=xterm-ghostty. Guard that the
	// TERM check also triggers detection independently of TERM_PROGRAM.
	setTermEnv(t, termEnv{term: "xterm-ghostty"})
	if got := Detect(); got != FormatKitty {
		t.Fatalf("expected FormatKitty for xterm-ghostty, got %d", got)
	}
}

func TestDetect_GhosttyInTmux(t *testing.T) {
	setTermEnv(t, termEnv{
		termProgram:         "tmux",
		term:                "xterm-256color",
		ghosttyResourcesDir: "/Applications/Ghostty.app/Contents/Resources/ghostty",
		tmux:                "/tmp/tmux-1000/default,123,0",
	})
	if got := Detect(); got != FormatKitty {
		t.Fatalf("expected FormatKitty for Ghostty in tmux, got %d", got)
	}
}

func TestDetect_GhosttyResourcesDirLeakedWithoutTmux(t *testing.T) {
	// GHOSTTY_RESOURCES_DIR can leak into non-Ghostty children (SSH SendEnv,
	// detached tmux reattached elsewhere, stray shell exports). Without TMUX
	// set, Detect must not assume Kitty graphics are supported.
	setTermEnv(t, termEnv{
		term:                "xterm-256color",
		ghosttyResourcesDir: "/Applications/Ghostty.app/Contents/Resources/ghostty",
	})
	if got := Detect(); got != FormatNone {
		t.Fatalf("leaked GHOSTTY_RESOURCES_DIR without TMUX must not trigger Kitty; got %d", got)
	}
}

func TestDetect_None(t *testing.T) {
	setTermEnv(t, termEnv{term: "xterm-256color"})
	if got := Detect(); got != FormatNone {
		t.Fatalf("expected FormatNone, got %d", got)
	}
}

func TestDetect_Sixel(t *testing.T) {
	setTermEnv(t, termEnv{term: "xterm-256color", glowmSixel: "1"})
	if got := Detect(); got != FormatSixel {
		t.Fatalf("expected FormatSixel, got %d", got)
	}
}

func TestDetect_Cached(t *testing.T) {
	setTermEnv(t, termEnv{termProgram: "iTerm.app", term: "xterm-256color"})
	if got := Detect(); got != FormatIterm2 {
		t.Fatalf("first call: expected FormatIterm2, got %d", got)
	}
	// Flip env vars; cached result must still win.
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("KITTY_WINDOW_ID", "1")
	if got := Detect(); got != FormatIterm2 {
		t.Fatalf("expected cached FormatIterm2, got %d", got)
	}
}

func TestEncodeKitty_EmptyData(t *testing.T) {
	got := encodeKitty(nil, 0)
	if got != "" {
		t.Fatalf("expected empty string for nil data, got %q", got)
	}
}
