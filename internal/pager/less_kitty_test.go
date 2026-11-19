package pager

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func makeKittyTestPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 120, B: 200, A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return b.Bytes()
}

// newKittyStateManual builds a lessKittyState directly from segment heights,
// bypassing PNG decoding, so scroll math can be tested in isolation.
func newKittyStateManual(visibleRows int, segs []lessSeg) *lessKittyState {
	rowStart := make([]int, len(segs))
	total := 0
	for i := range segs {
		if segs[i].rows < 1 {
			segs[i].rows = 1
		}
		rowStart[i] = total
		total += segs[i].rows
	}
	return &lessKittyState{
		segs:      segs,
		rowStart:  rowStart,
		totalRows: total,
		height:    visibleRows + 1, // +1 for the status row
	}
}

func TestKittySegAt(t *testing.T) {
	// text(1), image(5), text(1) => rows: [0],[1..5],[6]
	p := newKittyStateManual(3, []lessSeg{
		{plain: "a", rows: 1},
		{isImage: true, rows: 5},
		{plain: "b", rows: 1},
	})
	cases := []struct {
		row, wantSeg, wantOff int
	}{
		{0, 0, 0},
		{1, 1, 0}, // top of image
		{3, 1, 2}, // 2 rows into image
		{5, 1, 4}, // last row of image
		{6, 2, 0}, // text after image
	}
	for _, c := range cases {
		gotSeg, gotOff := p.segAt(c.row)
		if gotSeg != c.wantSeg || gotOff != c.wantOff {
			t.Errorf("segAt(%d) = (%d,%d), want (%d,%d)", c.row, gotSeg, gotOff, c.wantSeg, c.wantOff)
		}
	}
}

func TestKittyRowScroll(t *testing.T) {
	// Smooth scroll must step one row at a time, even into a tall image.
	p := newKittyStateManual(4, []lessSeg{
		{plain: "a", rows: 1},
		{isImage: true, rows: 6}, // rows 1..6
		{plain: "b", rows: 1},    // row 7
	}) // totalRows = 8, lpp = 4, maxRowTop = 4

	for want := 1; want <= 4; want++ {
		p.handleKey(key{typ: keyDown})
		if p.rowTop != want {
			t.Fatalf("after %d downs: rowTop=%d, want %d", want, p.rowTop, want)
		}
	}
	// Clamped at maxRowTop = 8 - 4 = 4.
	p.handleKey(key{typ: keyDown})
	if p.rowTop != 4 {
		t.Fatalf("rowTop should clamp to maxRowTop 4, got %d", p.rowTop)
	}
	p.handleKey(key{typ: keyTop})
	if p.rowTop != 0 {
		t.Fatalf("keyTop: rowTop=%d, want 0", p.rowTop)
	}
	p.handleKey(key{typ: keyBottom})
	if p.rowTop != 4 {
		t.Fatalf("keyBottom: rowTop=%d, want 4", p.rowTop)
	}
}

func TestKittyHalfAndFullPage(t *testing.T) {
	segs := make([]lessSeg, 20)
	for i := range segs {
		segs[i] = lessSeg{plain: "x", rows: 1}
	}
	p := newKittyStateManual(4, segs) // lpp=4, totalRows=20, maxRowTop=16

	p.handleKey(key{typ: keyPageDown})
	if p.rowTop != 4 {
		t.Fatalf("page down: rowTop=%d, want 4", p.rowTop)
	}
	p.handleKey(key{typ: keyHalfDown}) // +2
	if p.rowTop != 6 {
		t.Fatalf("half down: rowTop=%d, want 6", p.rowTop)
	}
	p.handleKey(key{typ: keyHalfUp}) // -2
	if p.rowTop != 4 {
		t.Fatalf("half up: rowTop=%d, want 4", p.rowTop)
	}
}

func TestKittySearchJumpsToSegmentRow(t *testing.T) {
	// Trailing padding keeps the match out of the final page so it can sit at
	// the top of the viewport without being clamped to maxRowTop.
	p := newKittyStateManual(2, []lessSeg{
		{plain: "intro", rows: 1},  // row 0
		{isImage: true, rows: 4},   // rows 1..4
		{plain: "target", rows: 1}, // row 5
		{plain: "p1", rows: 1},     // row 6
		{plain: "p2", rows: 1},     // row 7
	})
	p.handleKey(key{typ: keySearch, text: "target"})
	if p.rowTop != 5 {
		t.Fatalf("search jumped to rowTop=%d, want 5 (segment start row)", p.rowTop)
	}
}

func TestKittyNewStateParsesMarkers(t *testing.T) {
	// A 100x40 PNG at width 10 => imageRows = ceil((40/100)*10/2) = 2 rows.
	img := makeKittyTestPNG(t, 100, 40)
	markers := []string{"@@IMG0@@"}
	images := [][]byte{img}
	output := "line one\n@@IMG0@@\nline three"

	p := newLessKittyState(output, markers, images, 10, 5)
	if len(p.segs) != 3 {
		t.Fatalf("segs=%d, want 3", len(p.segs))
	}
	if !p.segs[1].isImage || p.segs[1].rows != 2 {
		t.Fatalf("segs[1]=%+v, want image with 2 rows", p.segs[1])
	}
	if p.totalRows != 4 { // 1 + 2 + 1
		t.Fatalf("totalRows=%d, want 4", p.totalRows)
	}
}
