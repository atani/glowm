package pager

import "strings"

// ANSI escape sequence constants.
const (
	ansiClearScreen  = "\033[H\033[2J"
	ansiReverseVideo = "\033[7m"
	ansiReset        = "\033[0m"
	ansiClearToEOL   = "\033[K"
	ansiAltScreenOn  = "\033[?1049h"
	ansiAltScreenOff = "\033[?1049l"
)

// stripANSI removes all ANSI escape sequences (CSI and OSC) from s,
// returning only the visible text.
func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			b.WriteByte(s[i])
			continue
		}
		i = skipEscapeSequence(s, i)
	}
	return b.String()
}

// visibleIndexMap returns a mapping from visible character index to byte
// position in the original string, skipping ANSI escape sequences.
func visibleIndexMap(s string) []int {
	var idx []int
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			idx = append(idx, i)
			continue
		}
		i = skipEscapeSequence(s, i)
	}
	return idx
}

// skipEscapeSequence advances past an ANSI escape sequence starting at s[i]
// (where s[i] == 0x1b). Returns the index of the last byte consumed.
func skipEscapeSequence(s string, i int) int {
	if i+1 >= len(s) {
		return i
	}
	switch s[i+1] {
	case '[': // CSI sequence: ESC [ ... <final byte 0x40-0x7e>
		i += 2
		for i < len(s) {
			if isCSITerminator(s[i]) {
				return i
			}
			i++
		}
		return i - 1
	case ']': // OSC sequence: ESC ] ... (BEL or ST)
		i += 2
		for i < len(s) {
			if s[i] == 0x07 { // BEL terminator
				return i
			}
			if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' { // ST terminator
				return i + 1
			}
			i++
		}
		return i - 1
	default:
		return i + 1
	}
}

// isCSITerminator returns true if b is a CSI final byte (0x40-0x7e, i.e. '@' through '~').
func isCSITerminator(b byte) bool {
	return b >= 0x40 && b <= 0x7e
}

func highlightLine(original, plain, pattern string) string {
	if pattern == "" || plain == "" {
		return original
	}

	matchStarts := findAllMatches(plain, pattern)
	if len(matchStarts) == 0 {
		return original
	}

	indexMap := visibleIndexMap(original)
	if len(indexMap) != len(plain) {
		return original
	}

	type span struct{ start, end int }
	var spans []span
	for _, s := range matchStarts {
		if s < 0 || s+len(pattern) > len(indexMap) {
			continue
		}
		start := indexMap[s]
		end := indexMap[s+len(pattern)-1] + 1
		if start < 0 || end <= start || end > len(original) {
			continue
		}
		spans = append(spans, span{start, end})
	}
	if len(spans) == 0 {
		return original
	}

	var b strings.Builder
	last := 0
	for _, sp := range spans {
		if sp.start < last {
			continue
		}
		b.WriteString(original[last:sp.start])
		b.WriteString(ansiReverseVideo)
		b.WriteString(original[sp.start:sp.end])
		b.WriteString(ansiReset)
		last = sp.end
	}
	b.WriteString(original[last:])
	return b.String()
}

func findAllMatches(s, pattern string) []int {
	if pattern == "" {
		return nil
	}
	var matches []int
	for i := 0; i+len(pattern) <= len(s); {
		idx := strings.Index(s[i:], pattern)
		if idx < 0 {
			break
		}
		pos := i + idx
		matches = append(matches, pos)
		i = pos + len(pattern)
	}
	return matches
}

func trimPlainToWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	if len(s) <= width {
		return s
	}
	return s[:width]
}

func trimANSIToWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	indexMap := visibleIndexMap(s)
	if len(indexMap) <= width {
		return s
	}
	cut := indexMap[width-1] + 1
	if cut < 0 || cut > len(s) {
		return s
	}
	return s[:cut]
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}
