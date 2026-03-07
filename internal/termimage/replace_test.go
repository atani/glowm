package termimage

import (
	"strings"
	"testing"
)

func TestReplaceMarkersWithImages_Basic(t *testing.T) {
	markers := []string{"GLOWM_MERMAID_0"}
	images := [][]byte{[]byte("png")}
	output := "before\nGLOWM_MERMAID_0\nafter"

	result := ReplaceMarkersWithImages(output, markers, images, FormatIterm2, 80)
	if strings.Contains(result, "GLOWM_MERMAID_0") {
		t.Fatal("expected marker to be replaced")
	}
	if !strings.Contains(result, "\x1b]1337;File=") {
		t.Fatal("expected iTerm2 image sequence")
	}
	if !strings.Contains(result, "before") || !strings.Contains(result, "after") {
		t.Fatal("expected surrounding text to be preserved")
	}
}

func TestReplaceMarkersWithImages_Empty(t *testing.T) {
	result := ReplaceMarkersWithImages("hello", nil, nil, FormatIterm2, 80)
	if result != "hello" {
		t.Fatalf("expected unchanged output, got %q", result)
	}
}

func TestReplaceMarkersWithImages_MoreMarkersThanImages(t *testing.T) {
	markers := []string{"GLOWM_MERMAID_0", "GLOWM_MERMAID_1"}
	images := [][]byte{[]byte("png")}
	output := "GLOWM_MERMAID_0\nGLOWM_MERMAID_1"

	result := ReplaceMarkersWithImages(output, markers, images, FormatIterm2, 80)
	// First marker should be replaced, second should remain.
	if strings.Contains(result, "GLOWM_MERMAID_0") {
		t.Fatal("expected first marker to be replaced")
	}
	if !strings.Contains(result, "GLOWM_MERMAID_1") {
		t.Fatal("expected second marker to remain (no matching image)")
	}
}

func TestReplaceMarkersWithImages_FormatNone(t *testing.T) {
	markers := []string{"GLOWM_MERMAID_0"}
	images := [][]byte{[]byte("png")}
	output := "GLOWM_MERMAID_0"

	result := ReplaceMarkersWithImages(output, markers, images, FormatNone, 80)
	if result != output {
		t.Fatalf("expected unchanged output for FormatNone, got %q", result)
	}
}

func TestStripANSI_NoEscapes(t *testing.T) {
	input := "hello world"
	if got := stripANSI(input); got != input {
		t.Fatalf("expected %q, got %q", input, got)
	}
}

func TestStripANSI_CSI(t *testing.T) {
	input := "\x1b[31mred\x1b[0m"
	got := stripANSI(input)
	if got != "red" {
		t.Fatalf("expected 'red', got %q", got)
	}
}

func TestStripANSI_OSC(t *testing.T) {
	input := "\x1b]8;;http://example.com\x07link\x1b]8;;\x07"
	got := stripANSI(input)
	if got != "link" {
		t.Fatalf("expected 'link', got %q", got)
	}
}

func TestStripANSI_APC(t *testing.T) {
	input := "\x1b_Gf=100,a=T,m=0;AAAA\x1b\\"
	got := stripANSI(input)
	if got != "" {
		t.Fatalf("expected empty string for APC sequence, got %q", got)
	}
}

func TestStripANSI_TruncatedEscape(t *testing.T) {
	input := "text\x1b"
	got := stripANSI(input)
	if got != "text" {
		t.Fatalf("expected 'text', got %q", got)
	}
}

func TestStripANSI_Mixed(t *testing.T) {
	input := "\x1b[1mbold\x1b[0m and \x1b]8;;url\x07link\x1b]8;;\x07"
	got := stripANSI(input)
	if got != "bold and link" {
		t.Fatalf("expected 'bold and link', got %q", got)
	}
}

func TestReplaceMarkersWithImages_ANSIWrappedMarker(t *testing.T) {
	markers := []string{"GLOWM_MERMAID_0"}
	images := [][]byte{[]byte("png")}
	// Marker wrapped in ANSI color codes (as glamour might do).
	output := "\x1b[1mGLOWM_MERMAID_0\x1b[0m"

	result := ReplaceMarkersWithImages(output, markers, images, FormatIterm2, 80)
	if strings.Contains(result, "GLOWM_MERMAID_0") {
		t.Fatal("expected ANSI-wrapped marker to be replaced")
	}
}

func TestReplaceMarkersWithImages_KittyFormat(t *testing.T) {
	markers := []string{"GLOWM_MERMAID_0"}
	images := [][]byte{[]byte("png")}
	output := "GLOWM_MERMAID_0"

	result := ReplaceMarkersWithImages(output, markers, images, FormatKitty, 80)
	if strings.Contains(result, "GLOWM_MERMAID_0") {
		t.Fatal("expected marker to be replaced")
	}
	if !strings.Contains(result, "\x1b_G") {
		t.Fatal("expected Kitty APC sequence")
	}
}

func TestStripANSI_TwoByteEscape(t *testing.T) {
	// ESC(B is a common "select character set" two-byte sequence.
	input := "text\x1b(Bmore"
	got := stripANSI(input)
	if got != "textmore" {
		t.Fatalf("expected 'textmore', got %q", got)
	}
}

func TestStripANSI_MalformedCSI(t *testing.T) {
	// CSI with no terminating letter — should not consume following content.
	input := "\x1b[999text"
	got := stripANSI(input)
	// The 't' in 'text' terminates the CSI, so 'ext' remains.
	if got != "ext" {
		t.Fatalf("expected 'ext', got %q", got)
	}
}

func TestStripANSI_DCS(t *testing.T) {
	// DCS sequence: ESC P ... ESC \
	input := "\x1bPsome data\x1b\\visible"
	got := stripANSI(input)
	if got != "visible" {
		t.Fatalf("expected 'visible', got %q", got)
	}
}
