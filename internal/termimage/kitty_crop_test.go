package termimage

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func makeTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return b.Bytes()
}

func TestImagePixelSize(t *testing.T) {
	p := makeTestPNG(t, 100, 40)
	w, h := ImagePixelSize(p)
	if w != 100 || h != 40 {
		t.Fatalf("ImagePixelSize = (%d,%d), want (100,40)", w, h)
	}
	if w, h := ImagePixelSize([]byte("not a png")); w != 0 || h != 0 {
		t.Fatalf("ImagePixelSize(bad) = (%d,%d), want (0,0)", w, h)
	}
}

func TestEncodeKittyCrop(t *testing.T) {
	p := makeTestPNG(t, 100, 40)
	// totalRows=4 over a 40px-tall image => 10px/row.
	// skip 1 row, show 2 rows => source y=10, h=20.
	got := EncodeKittyCrop(p, 10, 4, 1, 2)
	for _, want := range []string{"\x1b_G", "y=10", "w=100", "h=20", "c=10", "r=2", "C=1", "a=T"} {
		if !strings.Contains(got, want) {
			t.Errorf("crop escape missing %q\n got: %q", want, got[:min(len(got), 120)])
		}
	}
}

func TestEncodeKittyCropDegenerate(t *testing.T) {
	p := makeTestPNG(t, 100, 40)
	if s := EncodeKittyCrop(p, 10, 4, 0, 0); s != "" {
		t.Errorf("showRows=0 should yield empty, got %q", s)
	}
	if s := EncodeKittyCrop(p, 10, 4, 4, 1); s != "" {
		t.Errorf("skip beyond image should yield empty, got %q", s)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
