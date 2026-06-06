package pager

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestDrawStatus(t *testing.T) {
	t.Run("default status shows position and total", func(t *testing.T) {
		p := newTestState([]string{"a", "b", "c"}, 10)
		p.top = 0
		var buf bytes.Buffer
		w := bufio.NewWriter(&buf)
		p.drawStatus(w, 200)
		w.Flush()
		out := buf.String()
		if !strings.Contains(out, "1/3") {
			t.Errorf("status output missing position 1/3: %q", out)
		}
		if !strings.Contains(out, "glowm pager") {
			t.Errorf("status output missing label: %q", out)
		}
	})

	t.Run("empty document reports line 0", func(t *testing.T) {
		p := newTestState(nil, 10)
		var buf bytes.Buffer
		w := bufio.NewWriter(&buf)
		p.drawStatus(w, 200)
		w.Flush()
		if !strings.Contains(buf.String(), "0/0") {
			t.Errorf("empty doc status missing 0/0: %q", buf.String())
		}
	})

	t.Run("custom status overrides default", func(t *testing.T) {
		p := newTestState([]string{"a"}, 10)
		p.status = "pattern not found"
		var buf bytes.Buffer
		w := bufio.NewWriter(&buf)
		p.drawStatus(w, 200)
		w.Flush()
		out := buf.String()
		if !strings.Contains(out, "pattern not found") {
			t.Errorf("custom status not shown: %q", out)
		}
		if strings.Contains(out, "glowm pager") {
			t.Errorf("default status leaked when custom set: %q", out)
		}
	})
}

func TestRedraw(t *testing.T) {
	t.Run("renders visible window with reverse-video cursor", func(t *testing.T) {
		p := newTestState([]string{"line0", "line1", "line2"}, 10)
		p.cursor = 1
		var buf bytes.Buffer
		w := bufio.NewWriter(&buf)
		p.redraw(w)
		w.Flush()
		out := buf.String()
		for _, l := range []string{"line0", "line1", "line2"} {
			if !strings.Contains(out, l) {
				t.Errorf("redraw output missing %q: %q", l, out)
			}
		}
		// Cursor line is highlighted with reverse video.
		if !strings.Contains(out, ansiReverseVideo) {
			t.Errorf("redraw missing reverse-video for cursor line")
		}
		if !strings.HasPrefix(out, ansiClearScreen) {
			t.Errorf("redraw should clear screen first")
		}
	})

	t.Run("highlights search matches", func(t *testing.T) {
		p := newTestState([]string{"hello world", "no match here"}, 10)
		p.lastSearch = "world"
		var buf bytes.Buffer
		w := bufio.NewWriter(&buf)
		p.redraw(w)
		w.Flush()
		// A highlighted match wraps the term in reverse video then reset.
		if !strings.Contains(buf.String(), ansiReverseVideo+"world"+ansiReset) {
			t.Errorf("search match not highlighted: %q", buf.String())
		}
	})

	t.Run("pads short documents to fill the page", func(t *testing.T) {
		p := newTestState([]string{"only line"}, 5) // linesPerPage = 4
		var buf bytes.Buffer
		w := bufio.NewWriter(&buf)
		p.redraw(w)
		w.Flush()
		// One content line + padding "\r\n" rows should fill remaining space.
		if strings.Count(buf.String(), "\r\n") < 4 {
			t.Errorf("expected page padding to fill window, got %q", buf.String())
		}
	})
}

func TestPrintOutput(t *testing.T) {
	// printOutput writes to os.Stdout; just verify it returns no error for
	// normal content (the write target is the process stdout).
	if err := printOutput(""); err != nil {
		t.Errorf("printOutput(empty) error: %v", err)
	}
}

func TestPageEnd_AccountsForDisplayRows(t *testing.T) {
	heights := []int{1, 1, 5, 1}

	if got := pageEnd(heights, 0, 4); got != 2 {
		t.Fatalf("pageEnd() = %d, want 2 so tall image starts next page", got)
	}
	if got := pageEnd(heights, 2, 4); got != 3 {
		t.Fatalf("pageEnd() = %d, want 3 so oversized first item still renders", got)
	}
	if got := pageEnd(heights, len(heights), 4); got != len(heights) {
		t.Fatalf("pageEnd() = %d, want len(heights)", got)
	}
	if got := pageEnd([]int{1, 1}, 0, 4); got != 2 {
		t.Fatalf("pageEnd() = %d, want all lines when they fit", got)
	}
}

func TestSplitDisplayLines_ConsumesPagerRowsMarker(t *testing.T) {
	lines, heights := splitDisplayLines("a\n\x1b]1337;glowm-rows=7\x07image\nb")

	if len(lines) != 3 || lines[1] != "image" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
	if heights[0] != 1 || heights[1] != 7 || heights[2] != 1 {
		t.Fatalf("unexpected heights: %#v", heights)
	}
}

func TestConsumePagerRowsMarker_InvalidMarkers(t *testing.T) {
	tests := []string{
		"plain",
		"\x1b]1337;glowm-rows=\x07image",
		"\x1b]1337;glowm-rows=0\x07image",
		"\x1b]1337;glowm-rows=7image",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, rows := consumePagerRowsMarker(input)
			if rows != 1 {
				t.Fatalf("rows = %d, want 1", rows)
			}
		})
	}
}
