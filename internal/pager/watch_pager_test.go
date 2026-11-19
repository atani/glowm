package pager

import (
	"strings"
	"testing"
)

func TestLessApplyContentPreservesScroll(t *testing.T) {
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = "old line"
	}
	p := newLessState(strings.Join(lines, "\n"), 6) // lpp = 5
	p.top = 10

	newLines := make([]string, 40)
	for i := range newLines {
		newLines[i] = "new line"
	}
	p.applyContent(Content{Output: strings.Join(newLines, "\n")})

	if p.top != 10 {
		t.Errorf("top after reload = %d, want 10 (preserved)", p.top)
	}
	if len(p.lines) != 40 {
		t.Errorf("lines after reload = %d, want 40", len(p.lines))
	}
	if p.status != "" {
		t.Errorf("status after reload = %q, want empty", p.status)
	}
}

func TestLessApplyContentClampsScrollWhenShorter(t *testing.T) {
	long := make([]string, 30)
	for i := range long {
		long[i] = "x"
	}
	p := newLessState(strings.Join(long, "\n"), 6) // lpp = 5
	p.top = 25

	// New content is much shorter; top must clamp to the new maxTop.
	p.applyContent(Content{Output: "a\nb\nc"})
	if p.top > p.maxTop() {
		t.Errorf("top=%d exceeds maxTop=%d after shrinking", p.top, p.maxTop())
	}
}

func TestKittyApplyContentRebuildsSegments(t *testing.T) {
	img := makeKittyTestPNG(t, 100, 40) // 2 rows at width 10
	markers := []string{"@@IMG0@@"}
	images := [][]byte{img}

	// Start with a text-only document.
	p := newLessKittyState("a\nb\nc\nd", nil, nil, 10, 5)
	if p.totalRows != 4 {
		t.Fatalf("initial totalRows=%d, want 4", p.totalRows)
	}
	p.rowTop = 2

	// Reload with a document that now contains a diagram.
	p.applyContent(Content{Output: "intro\n@@IMG0@@\noutro", Markers: markers, Images: images, WidthCells: 10})
	if len(p.segs) != 3 {
		t.Fatalf("segs after reload = %d, want 3", len(p.segs))
	}
	if !p.segs[1].isImage || p.segs[1].rows != 2 {
		t.Errorf("segs[1] = %+v, want image with 2 rows", p.segs[1])
	}
	if p.totalRows != 4 { // 1 + 2 + 1
		t.Errorf("totalRows after reload = %d, want 4", p.totalRows)
	}
	if p.rowTop > p.maxRowTop() {
		t.Errorf("rowTop=%d exceeds maxRowTop=%d", p.rowTop, p.maxRowTop())
	}
}

func TestChanReaderPushback(t *testing.T) {
	ch := make(chan byte, 3)
	ch <- 'b'
	ch <- 'c'
	close(ch)
	r := &chanReader{ch: ch, pending: 'a', havePending: true}
	got := []byte{}
	for {
		b, err := r.ReadByte()
		if err != nil {
			break
		}
		got = append(got, b)
	}
	if string(got) != "abc" {
		t.Errorf("chanReader yielded %q, want %q", got, "abc")
	}
}
