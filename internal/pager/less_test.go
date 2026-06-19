package pager

import "testing"

// newLessState builds a less-mode state from plain text lines (each 1 row tall)
// with a viewport tall enough for `visible` content rows.
func newLessState(visible int, lines []string) *lessState {
	heights := make([]int, len(lines))
	plain := make([]string, len(lines))
	isImage := make([]bool, len(lines))
	for i, l := range lines {
		heights[i] = 1
		plain[i] = l
	}
	return &lessState{
		lines:   lines,
		heights: heights,
		plain:   plain,
		isImage: isImage,
		height:  visible + 1, // +1 for the status row
	}
}

func TestLessLineScroll(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e", "f"}
	p := newLessState(3, lines) // shows 3 lines per page

	p.handleKey(key{typ: keyDown})
	if p.top != 1 {
		t.Fatalf("keyDown: top=%d, want 1", p.top)
	}
	p.handleKey(key{typ: keyDown})
	p.handleKey(key{typ: keyUp})
	if p.top != 1 {
		t.Fatalf("down,down,up: top=%d, want 1", p.top)
	}
	// Cannot scroll above the first line.
	p.handleKey(key{typ: keyUp})
	p.handleKey(key{typ: keyUp})
	if p.top != 0 {
		t.Fatalf("clamped top: top=%d, want 0", p.top)
	}
}

func TestLessPageAndHalfPage(t *testing.T) {
	lines := []string{"l0", "l1", "l2", "l3", "l4", "l5", "l6", "l7", "l8", "l9"}
	p := newLessState(4, lines) // 4 rows per page

	p.handleKey(key{typ: keyPageDown})
	if p.top != 4 {
		t.Fatalf("page down: top=%d, want 4", p.top)
	}
	p.handleKey(key{typ: keyHalfUp}) // half of 4 = 2
	if p.top != 2 {
		t.Fatalf("half up: top=%d, want 2", p.top)
	}
	p.handleKey(key{typ: keyHalfDown})
	if p.top != 4 {
		t.Fatalf("half down: top=%d, want 4", p.top)
	}
}

func TestLessTopBottomClamp(t *testing.T) {
	lines := []string{"l0", "l1", "l2", "l3", "l4", "l5"}
	p := newLessState(3, lines) // 3 rows per page; last page starts at index 3

	p.handleKey(key{typ: keyBottom})
	if p.top != 3 {
		t.Fatalf("bottom: top=%d, want 3 (last page)", p.top)
	}
	// Paging/scrolling past the end must not exceed maxTop.
	p.handleKey(key{typ: keyPageDown})
	if p.top != 3 {
		t.Fatalf("page down past end: top=%d, want 3", p.top)
	}
	p.handleKey(key{typ: keyTop})
	if p.top != 0 {
		t.Fatalf("top: top=%d, want 0", p.top)
	}
}

func TestLessImageHeightAware(t *testing.T) {
	// Line 1 is an image occupying 5 rows; the rest are 1-row text lines.
	lines := []string{"text0", "IMG", "text2", "text3", "text4"}
	p := newLessState(6, lines) // 6 rows per page
	p.heights[1] = 5
	p.isImage[1] = true

	// maxTop must account for the tall image: from the end, rows are
	// text4,text3,text2(=3) then IMG(=5) would exceed 6, so the last page
	// starts at index 2.
	if got := p.maxTop(); got != 2 {
		t.Fatalf("maxTop with tall image: got %d, want 2", got)
	}
}

func TestLessSearch(t *testing.T) {
	// Trailing padding ensures the second match is not in the final page, so it
	// can rest at the top of the viewport without being clamped to maxTop.
	lines := []string{"alpha", "beta", "gamma", "delta", "beta again", "p1", "p2", "p3"}
	p := newLessState(2, lines)

	p.handleKey(key{typ: keySearch, text: "beta"})
	if p.top != 1 {
		t.Fatalf("search 'beta': top=%d, want 1", p.top)
	}
	p.handleKey(key{typ: keySearchNext})
	if p.top != 4 {
		t.Fatalf("search next 'beta': top=%d, want 4", p.top)
	}
	// Backward from the match wraps to the earlier occurrence.
	p.handleKey(key{typ: keySearchPrev})
	if p.top != 1 {
		t.Fatalf("search prev 'beta': top=%d, want 1", p.top)
	}
}

func TestLessSearchSkipsImages(t *testing.T) {
	lines := []string{"intro", "PAYLOAD", "match here"}
	p := newLessState(2, lines)
	// Mark line 1 as an image whose raw payload happens to contain the pattern;
	// search must not match or highlight it.
	p.isImage[1] = true
	p.plain[1] = ""
	p.heights[1] = 3

	p.handleKey(key{typ: keySearch, text: "match"})
	if p.top != 2 {
		t.Fatalf("search must skip image line: top=%d, want 2", p.top)
	}
}
