package pager

import (
	"testing"
)

func newTestState(lines []string, height int) *pagerState {
	plain := make([]string, len(lines))
	for i := range lines {
		plain[i] = stripANSI(lines[i])
	}
	return &pagerState{
		lines:  lines,
		plain:  plain,
		height: height,
	}
}

func TestClampCursor(t *testing.T) {
	tests := []struct {
		name       string
		lines      []string
		height     int
		cursor     int
		top        int
		wantCursor int
		wantTop    int
	}{
		{"empty lines", nil, 10, 5, 3, 0, 0},
		{"cursor negative", []string{"a", "b", "c"}, 10, -1, 0, 0, 0},
		{"cursor beyond end", []string{"a", "b", "c"}, 10, 10, 0, 2, 0},
		{"cursor below viewport", []string{"a", "b", "c", "d", "e"}, 3, 4, 0, 4, 3},
		{"cursor above viewport", []string{"a", "b", "c", "d", "e"}, 3, 0, 3, 0, 0},
		{"cursor within viewport", []string{"a", "b", "c", "d", "e"}, 3, 1, 0, 1, 0},
		{"negative top", []string{"a", "b"}, 10, 0, -5, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestState(tt.lines, tt.height)
			p.cursor = tt.cursor
			p.top = tt.top
			p.clampCursor()
			if p.cursor != tt.wantCursor {
				t.Errorf("cursor = %d, want %d", p.cursor, tt.wantCursor)
			}
			if p.top != tt.wantTop {
				t.Errorf("top = %d, want %d", p.top, tt.wantTop)
			}
		})
	}
}

func TestHandleKey(t *testing.T) {
	lines := []string{"line0", "line1", "line2", "line3", "line4"}

	t.Run("keyQuit returns true", func(t *testing.T) {
		p := newTestState(lines, 4) // 3 lines per page
		if !p.handleKey(key{typ: keyQuit}) {
			t.Error("expected handleKey(keyQuit) to return true")
		}
	})

	t.Run("keyDown increments cursor", func(t *testing.T) {
		p := newTestState(lines, 4)
		p.handleKey(key{typ: keyDown})
		if p.cursor != 1 {
			t.Errorf("cursor = %d, want 1", p.cursor)
		}
	})

	t.Run("keyUp decrements cursor", func(t *testing.T) {
		p := newTestState(lines, 4)
		p.cursor = 2
		p.handleKey(key{typ: keyUp})
		if p.cursor != 1 {
			t.Errorf("cursor = %d, want 1", p.cursor)
		}
	})

	t.Run("keyPageDown moves by page", func(t *testing.T) {
		p := newTestState(lines, 4) // linesPerPage = 3
		p.handleKey(key{typ: keyPageDown})
		if p.cursor != 3 {
			t.Errorf("cursor = %d, want 3", p.cursor)
		}
	})

	t.Run("keyPageUp moves by page", func(t *testing.T) {
		p := newTestState(lines, 4)
		p.cursor = 4
		p.top = 2
		p.handleKey(key{typ: keyPageUp})
		if p.cursor != 1 {
			t.Errorf("cursor = %d, want 1", p.cursor)
		}
	})

	t.Run("keyBottom goes to last line", func(t *testing.T) {
		p := newTestState(lines, 4)
		p.handleKey(key{typ: keyBottom})
		if p.cursor != 4 {
			t.Errorf("cursor = %d, want 4", p.cursor)
		}
		// Top should show last line at bottom of viewport
		if p.top != 2 {
			t.Errorf("top = %d, want 2", p.top)
		}
	})

	t.Run("keyG twice goes to top (gg)", func(t *testing.T) {
		p := newTestState(lines, 4)
		p.cursor = 3
		p.top = 1
		p.handleKey(key{typ: keyG})
		if !p.awaitingGG {
			t.Error("expected awaitingGG to be true after first g")
		}
		p.handleKey(key{typ: keyG})
		if p.cursor != 0 || p.top != 0 {
			t.Errorf("after gg: cursor=%d top=%d, want 0,0", p.cursor, p.top)
		}
		if p.awaitingGG {
			t.Error("expected awaitingGG to be false after gg")
		}
	})

	t.Run("keyG then other key cancels gg", func(t *testing.T) {
		p := newTestState(lines, 4)
		p.cursor = 3
		p.handleKey(key{typ: keyG})
		if !p.awaitingGG {
			t.Error("expected awaitingGG true")
		}
		p.handleKey(key{typ: keyDown})
		if p.awaitingGG {
			t.Error("expected awaitingGG false after non-g key")
		}
		if p.cursor != 4 {
			t.Errorf("cursor = %d, want 4", p.cursor)
		}
	})
}

func TestSearchForward(t *testing.T) {
	lines := []string{"apple", "banana", "cherry", "apple pie", "date"}

	t.Run("finds match ahead", func(t *testing.T) {
		p := newTestState(lines, 10)
		p.lastSearch = "cherry"
		p.searchForward()
		if p.cursor != 2 {
			t.Errorf("cursor = %d, want 2", p.cursor)
		}
		if p.status != "" {
			t.Errorf("status = %q, want empty", p.status)
		}
	})

	t.Run("wraps around", func(t *testing.T) {
		p := newTestState(lines, 10)
		p.cursor = 3
		p.lastSearch = "banana"
		p.searchForward()
		if p.cursor != 1 {
			t.Errorf("cursor = %d, want 1", p.cursor)
		}
		if p.status != "search hit BOTTOM, continuing at TOP" {
			t.Errorf("status = %q", p.status)
		}
	})

	t.Run("pattern not found", func(t *testing.T) {
		p := newTestState(lines, 10)
		p.lastSearch = "zzz"
		p.searchForward()
		if p.status != "pattern not found" {
			t.Errorf("status = %q, want 'pattern not found'", p.status)
		}
	})

	t.Run("no previous search", func(t *testing.T) {
		p := newTestState(lines, 10)
		p.searchForward()
		if p.status != "no previous search" {
			t.Errorf("status = %q, want 'no previous search'", p.status)
		}
	})
}

func TestSearchBackward(t *testing.T) {
	lines := []string{"apple", "banana", "cherry", "apple pie", "date"}

	t.Run("finds match behind", func(t *testing.T) {
		p := newTestState(lines, 10)
		p.cursor = 3
		p.lastSearch = "banana"
		p.searchBackward()
		if p.cursor != 1 {
			t.Errorf("cursor = %d, want 1", p.cursor)
		}
	})

	t.Run("wraps around", func(t *testing.T) {
		p := newTestState(lines, 10)
		p.cursor = 1
		p.lastSearch = "date"
		p.searchBackward()
		if p.cursor != 4 {
			t.Errorf("cursor = %d, want 4", p.cursor)
		}
		if p.status != "search hit TOP, continuing at BOTTOM" {
			t.Errorf("status = %q", p.status)
		}
	})

	t.Run("pattern not found", func(t *testing.T) {
		p := newTestState(lines, 10)
		p.cursor = 2
		p.lastSearch = "zzz"
		p.searchBackward()
		if p.status != "pattern not found" {
			t.Errorf("status = %q", p.status)
		}
	})
}

func TestWordUnderCursor(t *testing.T) {
	tests := []struct {
		name   string
		lines  []string
		cursor int
		want   string
	}{
		{"empty lines", nil, 0, ""},
		{"empty line", []string{""}, 0, ""},
		{"first word", []string{"hello world"}, 0, "hello"},
		{"word with underscore", []string{"foo_bar baz"}, 0, "foo_bar"},
		{"no word chars", []string{"---"}, 0, ""},
		{"word after spaces", []string{"  hello"}, 0, "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestState(tt.lines, 10)
			p.cursor = tt.cursor
			got := p.wordUnderCursor()
			if got != tt.want {
				t.Errorf("wordUnderCursor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHistory(t *testing.T) {
	t.Run("appendHistory deduplicates consecutive", func(t *testing.T) {
		p := &pagerState{}
		p.appendHistory("foo")
		p.appendHistory("foo")
		p.appendHistory("bar")
		if len(p.history) != 2 {
			t.Errorf("history len = %d, want 2", len(p.history))
		}
	})

	t.Run("appendHistory ignores empty", func(t *testing.T) {
		p := &pagerState{}
		p.appendHistory("")
		if len(p.history) != 0 {
			t.Errorf("history len = %d, want 0", len(p.history))
		}
	})

	t.Run("historyPrev at beginning stays at 0", func(t *testing.T) {
		p := &pagerState{history: []string{"a", "b"}, histIndex: 0}
		got := p.historyPrev(nil)
		if string(got) != "a" {
			t.Errorf("got %q, want 'a'", got)
		}
		if p.histIndex != 0 {
			t.Errorf("histIndex = %d, want 0", p.histIndex)
		}
	})

	t.Run("historyNext past end returns empty", func(t *testing.T) {
		p := &pagerState{history: []string{"a", "b"}, histIndex: 1}
		got := p.historyNext(nil)
		if len(got) != 0 {
			t.Errorf("got %q, want empty", got)
		}
		if p.histIndex != 2 {
			t.Errorf("histIndex = %d, want 2", p.histIndex)
		}
	})

	t.Run("full cycle", func(t *testing.T) {
		p := &pagerState{history: []string{"x", "y", "z"}, histIndex: 3}
		got := p.historyPrev(nil)
		if string(got) != "z" {
			t.Errorf("prev1 = %q, want z", got)
		}
		got = p.historyPrev(nil)
		if string(got) != "y" {
			t.Errorf("prev2 = %q, want y", got)
		}
		got = p.historyNext(nil)
		if string(got) != "z" {
			t.Errorf("next1 = %q, want z", got)
		}
	})
}

func TestLinesPerPage(t *testing.T) {
	tests := []struct {
		height int
		want   int
	}{
		{10, 9},
		{2, 1},
		{1, 1},
		{0, 1},
		{-1, 1},
	}
	for _, tt := range tests {
		p := &pagerState{height: tt.height}
		got := p.linesPerPage()
		if got != tt.want {
			t.Errorf("linesPerPage(height=%d) = %d, want %d", tt.height, got, tt.want)
		}
	}
}

func TestValidMode(t *testing.T) {
	if !ValidMode(ModeMore) {
		t.Error("ModeMore should be valid")
	}
	if !ValidMode(ModeVim) {
		t.Error("ModeVim should be valid")
	}
	if ValidMode("emacs") {
		t.Error("emacs should not be valid")
	}
	if ValidMode("") {
		t.Error("empty should not be valid")
	}
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		input    string
		wantMode Mode
		wantOK   bool
	}{
		{"more", ModeMore, true},
		{"vim", ModeVim, true},
		{"MORE", ModeMore, true},
		{"VIM", ModeVim, true},
		{"emacs", ModeMore, false},
		{"", ModeMore, false},
	}
	for _, tt := range tests {
		m, ok := ParseMode(tt.input)
		if m != tt.wantMode || ok != tt.wantOK {
			t.Errorf("ParseMode(%q) = (%q, %v), want (%q, %v)", tt.input, m, ok, tt.wantMode, tt.wantOK)
		}
	}
}
