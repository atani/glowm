package pager

import (
	"testing"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"no ansi", "hello world", "hello world"},
		{"single CSI color", "\033[31mred\033[0m", "red"},
		{"bold and color", "\033[1m\033[34mblue\033[0m", "blue"},
		{"multiple sequences", "\033[31mred\033[0m and \033[32mgreen\033[0m", "red and green"},
		{"OSC with BEL", "\033]8;;https://example.com\007link\033]8;;\007", "link"},
		{"OSC with ST", "\033]8;;https://example.com\033\\link\033]8;;\033\\", "link"},
		{"unterminated ESC at end", "hello\033", "hello"},
		{"consecutive CSI sequences", "\033[1m\033[31m\033[42mtext\033[0m", "text"},
		{"mixed text and escapes", "a\033[1mb\033[0mc", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.input)
			if got != tt.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestVisibleIndexMap(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []int
	}{
		{"empty", "", nil},
		{"plain ASCII", "abc", []int{0, 1, 2}},
		{"single CSI", "\033[31ma\033[0m", []int{5}},
		{"mixed", "a\033[31mb\033[0mc", []int{0, 6, 11}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := visibleIndexMap(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("visibleIndexMap(%q) len = %d, want %d", tt.input, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("visibleIndexMap(%q)[%d] = %d, want %d", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFindAllMatches(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		pattern string
		want    []int
	}{
		{"empty pattern", "hello", "", nil},
		{"empty string", "", "hello", nil},
		{"no match", "hello", "xyz", nil},
		{"single match", "hello", "ell", []int{1}},
		{"multiple matches", "abcabc", "abc", []int{0, 3}},
		{"pattern at end", "hello world", "world", []int{6}},
		{"pattern at start", "hello world", "hello", []int{0}},
		{"overlapping not returned", "aaa", "aa", []int{0}},
		{"pattern longer than string", "hi", "hello", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findAllMatches(tt.s, tt.pattern)
			if len(got) != len(tt.want) {
				t.Fatalf("findAllMatches(%q, %q) = %v, want %v", tt.s, tt.pattern, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("findAllMatches(%q, %q)[%d] = %d, want %d", tt.s, tt.pattern, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestHighlightLine(t *testing.T) {
	tests := []struct {
		name     string
		original string
		plain    string
		pattern  string
		want     string
	}{
		{"empty pattern", "hello", "hello", "", "hello"},
		{"empty plain", "hello", "", "hello", "hello"},
		{"no match", "hello", "hello", "xyz", "hello"},
		{"plain text match", "hello world", "hello world", "world", "hello \033[7mworld\033[0m"},
		{"match with ANSI", "\033[31mhello\033[0m", "hello", "hello", "\033[31m\033[7mhello\033[0m\033[0m"},
		{"multiple matches", "abc abc", "abc abc", "abc", "\033[7mabc\033[0m \033[7mabc\033[0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := highlightLine(tt.original, tt.plain, tt.pattern)
			if got != tt.want {
				t.Errorf("highlightLine(%q, %q, %q) = %q, want %q", tt.original, tt.plain, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestTrimPlainToWidth(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{"within width", "hello", 10, "hello"},
		{"exact width", "hello", 5, "hello"},
		{"exceeds width", "hello world", 5, "hello"},
		{"zero width", "hello", 0, "hello"},
		{"width 1", "hello", 1, "h"},
		{"empty string", "", 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimPlainToWidth(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("trimPlainToWidth(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
			}
		})
	}
}

func TestTrimANSIToWidth(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		width int
		want  string
	}{
		{"plain within width", "hello", 10, "hello"},
		{"plain exceeds width", "hello world", 5, "hello"},
		{"ANSI within width", "\033[31mhello\033[0m", 10, "\033[31mhello\033[0m"},
		{"ANSI exceeds width", "\033[31mhello world\033[0m", 5, "\033[31mhello"},
		{"zero width", "hello", 0, "hello"},
		{"only ANSI codes", "\033[31m\033[0m", 5, "\033[31m\033[0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimANSIToWidth(tt.s, tt.width)
			if got != tt.want {
				t.Errorf("trimANSIToWidth(%q, %d) = %q, want %q", tt.s, tt.width, got, tt.want)
			}
		})
	}
}

func TestIsWordChar(t *testing.T) {
	trueChars := "abcABC019_"
	falseChars := " .-!@#$%^&*()+"
	for _, c := range trueChars {
		if !isWordChar(byte(c)) {
			t.Errorf("isWordChar(%q) = false, want true", c)
		}
	}
	for _, c := range falseChars {
		if isWordChar(byte(c)) {
			t.Errorf("isWordChar(%q) = true, want false", c)
		}
	}
}

func TestIsCSITerminator(t *testing.T) {
	// 0x40 = '@', 0x7e = '~'
	if !isCSITerminator('@') {
		t.Error("expected '@' to be CSI terminator")
	}
	if !isCSITerminator('m') {
		t.Error("expected 'm' to be CSI terminator")
	}
	if !isCSITerminator('~') {
		t.Error("expected '~' to be CSI terminator")
	}
	if isCSITerminator('3') {
		t.Error("expected '3' not to be CSI terminator")
	}
	if isCSITerminator(';') {
		t.Error("expected ';' not to be CSI terminator")
	}
}

func TestSkipEscapeSequence(t *testing.T) {
	tests := []struct {
		name string
		s    string
		i    int
		want int
	}{
		{"CSI sequence", "\033[31m", 0, 4},
		{"CSI with params", "\033[1;31m", 0, 6},
		{"OSC with BEL", "\033]0;title\007", 0, 9},
		{"lone ESC at end", "\033", 0, 0},
		{"ESC with unknown", "\033X", 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := skipEscapeSequence(tt.s, tt.i)
			if got != tt.want {
				t.Errorf("skipEscapeSequence(%q, %d) = %d, want %d", tt.s, tt.i, got, tt.want)
			}
		})
	}
}
