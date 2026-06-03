package pager

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

// newReader wraps a byte string in a *bufio.Reader for feeding to the
// key/search parsing functions.
func newReader(s string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(s))
}

func TestReadKey(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantTyp keyType
	}{
		{"quit q", "q", keyQuit},
		{"quit Q", "Q", keyQuit},
		{"space page down", " ", keyPageDown},
		{"b page up", "b", keyPageUp},
		{"ctrl-f page down", "\x06", keyPageDown},
		{"ctrl-b page up", "\x02", keyPageUp},
		{"j down", "j", keyDown},
		{"newline down", "\n", keyDown},
		{"k up", "k", keyUp},
		{"g", "g", keyG},
		{"G bottom", "G", keyBottom},
		{"n search next", "n", keySearchNext},
		{"N search prev", "N", keySearchPrev},
		{"* search word", "*", keySearchWord},
		{"# search word backward", "#", keySearchWordBackward},
		{"unknown char", "x", keyUnknown},
		{"EOF returns quit", "", keyQuit},
		{"arrow up", "\x1b[A", keyUp},
		{"arrow down", "\x1b[B", keyDown},
		{"page up esc 5", "\x1b[5~", keyPageUp},
		{"page down esc 6", "\x1b[6~", keyPageDown},
		{"esc without bracket", "\x1bX", keyUnknown},
		{"esc unknown code", "\x1b[Z", keyUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newReader(tt.input)
			var w bytes.Buffer
			bw := bufio.NewWriter(&w)
			p := newTestState([]string{"a", "b"}, 10)
			got := readKey(r, bw, p)
			if got.typ != tt.wantTyp {
				t.Errorf("readKey(%q).typ = %d, want %d", tt.input, got.typ, tt.wantTyp)
			}
		})
	}
}

func TestReadKey_SearchForward(t *testing.T) {
	r := newReader("/foo\n")
	var w bytes.Buffer
	bw := bufio.NewWriter(&w)
	p := newTestState([]string{"foo", "bar"}, 10)
	got := readKey(r, bw, p)
	if got.typ != keySearch {
		t.Fatalf("typ = %d, want keySearch", got.typ)
	}
	if got.text != "foo" {
		t.Errorf("text = %q, want 'foo'", got.text)
	}
	if len(p.history) != 1 || p.history[0] != "foo" {
		t.Errorf("history = %v, want [foo]", p.history)
	}
}

func TestReadKey_SearchBackward(t *testing.T) {
	r := newReader("?bar\r")
	var w bytes.Buffer
	bw := bufio.NewWriter(&w)
	p := newTestState([]string{"foo", "bar"}, 10)
	got := readKey(r, bw, p)
	if got.typ != keySearchBackward {
		t.Fatalf("typ = %d, want keySearchBackward", got.typ)
	}
	if got.text != "bar" {
		t.Errorf("text = %q, want 'bar'", got.text)
	}
}

func TestReadKey_EmptySearchIsUnknown(t *testing.T) {
	// Pressing '/' then immediately Enter yields an empty search and must
	// not be dispatched as a search.
	r := newReader("/\n")
	var w bytes.Buffer
	bw := bufio.NewWriter(&w)
	p := newTestState([]string{"a"}, 10)
	got := readKey(r, bw, p)
	if got.typ != keyUnknown {
		t.Errorf("empty search typ = %d, want keyUnknown", got.typ)
	}
	if len(p.history) != 0 {
		t.Errorf("empty search should not be added to history, got %v", p.history)
	}
}

func TestReadSearch(t *testing.T) {
	t.Run("plain text terminated by enter", func(t *testing.T) {
		r := newReader("hello\n")
		var w bytes.Buffer
		bw := bufio.NewWriter(&w)
		p := newTestState([]string{"a"}, 10)
		got := readSearch(r, bw, p, "/")
		if got != "hello" {
			t.Errorf("got %q, want 'hello'", got)
		}
		if p.status != "" {
			t.Errorf("status = %q, want empty after enter", p.status)
		}
	})

	t.Run("backspace deletes last char", func(t *testing.T) {
		r := newReader("abc\x7f\n") // type abc, delete c -> "ab"
		var w bytes.Buffer
		bw := bufio.NewWriter(&w)
		p := newTestState([]string{"a"}, 10)
		got := readSearch(r, bw, p, "/")
		if got != "ab" {
			t.Errorf("got %q, want 'ab'", got)
		}
	})

	t.Run("backspace on empty buffer is a no-op", func(t *testing.T) {
		r := newReader("\x7f\n")
		var w bytes.Buffer
		bw := bufio.NewWriter(&w)
		p := newTestState([]string{"a"}, 10)
		got := readSearch(r, bw, p, "/")
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("escape cancels search", func(t *testing.T) {
		r := newReader("ab\x1bX") // escape not followed by '[' cancels
		var w bytes.Buffer
		bw := bufio.NewWriter(&w)
		p := newTestState([]string{"a"}, 10)
		got := readSearch(r, bw, p, "/")
		if got != "" {
			t.Errorf("got %q, want empty (cancelled)", got)
		}
	})

	t.Run("control chars below 0x20 ignored", func(t *testing.T) {
		r := newReader("a\x01b\n") // \x01 (ctrl-a) must be ignored
		var w bytes.Buffer
		bw := bufio.NewWriter(&w)
		p := newTestState([]string{"a"}, 10)
		got := readSearch(r, bw, p, "/")
		if got != "ab" {
			t.Errorf("got %q, want 'ab'", got)
		}
	})

	t.Run("EOF returns empty", func(t *testing.T) {
		r := newReader("ab")
		var w bytes.Buffer
		bw := bufio.NewWriter(&w)
		p := newTestState([]string{"a"}, 10)
		got := readSearch(r, bw, p, "/")
		if got != "" {
			t.Errorf("got %q, want empty on EOF", got)
		}
	})

	t.Run("up arrow recalls history", func(t *testing.T) {
		r := newReader("\x1b[A\n") // up arrow then enter
		var w bytes.Buffer
		bw := bufio.NewWriter(&w)
		p := newTestState([]string{"a"}, 10)
		p.history = []string{"prev"}
		got := readSearch(r, bw, p, "/")
		if got != "prev" {
			t.Errorf("got %q, want 'prev' from history", got)
		}
	})

	t.Run("down arrow navigates history forward", func(t *testing.T) {
		r := newReader("\x1b[A\x1b[A\x1b[B\n") // up up down
		var w bytes.Buffer
		bw := bufio.NewWriter(&w)
		p := newTestState([]string{"a"}, 10)
		p.history = []string{"first", "second"}
		got := readSearch(r, bw, p, "/")
		// up,up -> "first", down -> "second"
		if got != "second" {
			t.Errorf("got %q, want 'second'", got)
		}
	})
}

func TestHandleSearchEscape(t *testing.T) {
	t.Run("non-bracket sequence cancels", func(t *testing.T) {
		r := newReader("X")
		var w bytes.Buffer
		bw := bufio.NewWriter(&w)
		p := newTestState([]string{"a"}, 10)
		handled, _ := handleSearchEscape(r, bw, p, "/", []byte("ab"), 80)
		if handled {
			t.Error("expected non-bracket escape to be unhandled (cancel)")
		}
	})

	t.Run("unknown code cancels", func(t *testing.T) {
		r := newReader("[Z")
		var w bytes.Buffer
		bw := bufio.NewWriter(&w)
		p := newTestState([]string{"a"}, 10)
		handled, _ := handleSearchEscape(r, bw, p, "/", nil, 80)
		if handled {
			t.Error("expected unknown code to be unhandled")
		}
	})
}

func TestSearchWord(t *testing.T) {
	lines := []string{"apple", "banana", "apple pie", "cherry"}

	t.Run("finds next occurrence of word under cursor", func(t *testing.T) {
		p := newTestState(lines, 10)
		p.cursor = 0 // word "apple"
		p.searchWord(dirForward)
		if p.lastSearch != "apple" {
			t.Errorf("lastSearch = %q, want 'apple'", p.lastSearch)
		}
		if p.cursor != 2 {
			t.Errorf("cursor = %d, want 2", p.cursor)
		}
	})

	t.Run("no word under cursor sets status", func(t *testing.T) {
		p := newTestState([]string{"---", "banana"}, 10)
		p.cursor = 0
		p.searchWord(dirForward)
		if p.status != "no word under cursor" {
			t.Errorf("status = %q, want 'no word under cursor'", p.status)
		}
	})

	t.Run("backward direction recorded", func(t *testing.T) {
		p := newTestState(lines, 10)
		p.cursor = 2 // "apple pie" -> apple
		p.searchWord(dirBackward)
		if p.lastDir != dirBackward {
			t.Error("expected lastDir to be dirBackward")
		}
		if p.cursor != 0 {
			t.Errorf("cursor = %d, want 0", p.cursor)
		}
	})
}

func TestHandleKey_SearchNavigation(t *testing.T) {
	lines := []string{"apple", "banana", "apple pie", "cherry"}

	t.Run("keySearchNext repeats last search", func(t *testing.T) {
		p := newTestState(lines, 10)
		p.lastSearch = "apple"
		p.lastDir = dirForward
		p.cursor = 0
		p.handleKey(key{typ: keySearchNext})
		if p.cursor != 2 {
			t.Errorf("cursor = %d, want 2", p.cursor)
		}
	})

	t.Run("keySearchPrev reverses direction", func(t *testing.T) {
		p := newTestState(lines, 10)
		p.lastSearch = "apple"
		p.lastDir = dirForward
		p.cursor = 2
		p.handleKey(key{typ: keySearchPrev})
		if p.cursor != 0 {
			t.Errorf("cursor = %d, want 0", p.cursor)
		}
	})

	t.Run("keySearchWord searches word under cursor", func(t *testing.T) {
		p := newTestState(lines, 10)
		p.cursor = 0
		p.handleKey(key{typ: keySearchWord})
		if p.cursor != 2 || p.lastSearch != "apple" {
			t.Errorf("cursor=%d lastSearch=%q, want 2/'apple'", p.cursor, p.lastSearch)
		}
	})

	t.Run("keySearchWordBackward searches word backward", func(t *testing.T) {
		p := newTestState(lines, 10)
		p.cursor = 2
		p.handleKey(key{typ: keySearchWordBackward})
		if p.cursor != 0 {
			t.Errorf("cursor = %d, want 0", p.cursor)
		}
	})

	t.Run("unknown key is a no-op", func(t *testing.T) {
		p := newTestState(lines, 10)
		p.cursor = 1
		if p.handleKey(key{typ: keyUnknown}) {
			t.Error("keyUnknown should not signal exit")
		}
		if p.cursor != 1 {
			t.Errorf("cursor moved to %d on unknown key", p.cursor)
		}
	})
}
