package terminal

import (
	"math"
	"testing"
)

func TestParseOSC11Color(t *testing.T) {
	cases := []struct {
		name                string
		resp                string
		wantR, wantG, wantB float64
		wantOK              bool
	}{
		{"black 16-bit BEL", "\x1b]11;rgb:0000/0000/0000\x07", 0, 0, 0, true},
		{"white 16-bit BEL", "\x1b]11;rgb:ffff/ffff/ffff\x07", 1, 1, 1, true},
		{"white 8-bit", "\x1b]11;rgb:ff/ff/ff\x07", 1, 1, 1, true},
		{"mid gray", "\x1b]11;rgb:8080/8080/8080\x07", 0x8080 / 65535.0, 0x8080 / 65535.0, 0x8080 / 65535.0, true},
		{"ST terminator", "\x1b]11;rgb:1234/5678/9abc\x1b\\", 0x1234 / 65535.0, 0x5678 / 65535.0, 0x9abc / 65535.0, true},
		{"no rgb", "\x1b]11;garbage\x07", 0, 0, 0, false},
		{"too few parts", "\x1b]11;rgb:ffff/ffff\x07", 0, 0, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, g, b, ok := parseOSC11Color(c.resp)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if !ok {
				return
			}
			const eps = 1e-9
			if math.Abs(r-c.wantR) > eps || math.Abs(g-c.wantG) > eps || math.Abs(b-c.wantB) > eps {
				t.Errorf("got (%v,%v,%v), want (%v,%v,%v)", r, g, b, c.wantR, c.wantG, c.wantB)
			}
		})
	}
}

// TestLuminanceThreshold documents the dark/light decision used by
// detectBackground for representative backgrounds.
func TestLuminanceThreshold(t *testing.T) {
	lum := func(r, g, b float64) float64 { return 0.2126*r + 0.7152*g + 0.0722*b }
	if lum(0, 0, 0) >= 0.5 {
		t.Error("black should be dark")
	}
	if lum(1, 1, 1) < 0.5 {
		t.Error("white should be light")
	}
	// A typical dark-terminal background (#1e1e1e).
	v := 0x1e / 255.0
	if lum(v, v, v) >= 0.5 {
		t.Error("#1e1e1e should be dark")
	}
}
