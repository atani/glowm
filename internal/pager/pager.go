package pager

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

func Page(output string) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		_, err := fmt.Fprint(os.Stdout, output)
		return err
	}

	height := terminalHeight()
	if height <= 0 {
		_, err := fmt.Fprint(os.Stdout, output)
		return err
	}

	lines := strings.Split(output, "\n")
	plainLines := make([]string, len(lines))
	for i := range lines {
		plainLines[i] = stripANSI(lines[i])
	}

	reader := openTTYReader()
	defer reader.Close()

	oldState, err := term.MakeRaw(int(reader.Fd()))
	if err != nil {
		_, err := fmt.Fprint(os.Stdout, output)
		return err
	}
	defer term.Restore(int(reader.Fd()), oldState)

	bufReader := bufio.NewReader(reader)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	p := &pagerState{
		lines:      lines,
		plain:      plainLines,
		height:     height,
		top:        0,
		lastSearch: "",
		status:     "",
	}

	p.redraw(writer)
	for {
		key := readKey(bufReader, writer, p)
		if p.handleKey(key) {
			break
		}
		p.redraw(writer)
	}
	fmt.Fprint(writer, "\r\033[K")
	return nil
}

func terminalHeight() int {
	h, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || h <= 0 {
		return 0
	}
	return h
}

func openTTYReader() *os.File {
	f, err := os.Open("/dev/tty")
	if err != nil {
		return os.Stdin
	}
	return f
}

type pagerState struct {
	lines      []string
	plain      []string
	height     int
	top        int
	cursor     int
	cursorCol  int
	lastSearch string
	lastDir    searchDir
	status     string
	history    []string
	histIndex  int
	pendingG   bool
}

func (p *pagerState) redraw(w *bufio.Writer) {
	linesPerPage := p.height - 1
	if linesPerPage <= 0 {
		linesPerPage = 1
	}
	if p.top < 0 {
		p.top = 0
	}
	if p.top > len(p.lines)-1 {
		if len(p.lines) > 0 {
			p.top = len(p.lines) - 1
		} else {
			p.top = 0
		}
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor > len(p.lines)-1 {
		if len(p.lines) > 0 {
			p.cursor = len(p.lines) - 1
		} else {
			p.cursor = 0
		}
	}

	fmt.Fprint(w, "\033[H\033[2J")
	end := p.top + linesPerPage
	if end > len(p.lines) {
		end = len(p.lines)
	}
	for i := p.top; i < end; i++ {
		line := p.lines[i]
		if p.lastSearch != "" {
			line = highlightLine(line, p.plain[i], p.lastSearch)
		}
		if i == p.cursor {
			fmt.Fprint(w, "\033[7m")
			fmt.Fprint(w, line)
			fmt.Fprintln(w, "\033[0m")
		} else {
			fmt.Fprintln(w, line)
		}
	}
	for i := end - p.top; i < linesPerPage; i++ {
		fmt.Fprintln(w, "")
	}
	p.drawStatus(w)
	w.Flush()
}

func (p *pagerState) drawStatus(w *bufio.Writer) {
	total := len(p.lines)
	line := p.top + 1
	if total == 0 {
		line = 0
	}
	status := p.status
	if status == "" {
		status = fmt.Sprintf("glowm pager  %d/%d  (q quit, / ? search, n/N next/prev, * # word, gg/G top/bottom, ctrl-f/ctrl-b page, h/l col)", line, total)
	}
	fmt.Fprintf(w, "\033[%d;1H", p.height)
	fmt.Fprintf(w, "\033[7m%s\033[0m", truncateStatus(status))
	fmt.Fprint(w, "\r\033[K")
}

func (p *pagerState) handleKey(k key) bool {
	linesPerPage := p.height - 1
	if linesPerPage <= 0 {
		linesPerPage = 1
	}

	if p.pendingG && k.typ != keyG {
		p.pendingG = false
	}

	switch k.typ {
	case keyQuit:
		return true
	case keyPageDown:
		p.top += linesPerPage
		p.cursor += linesPerPage
	case keyPageUp:
		p.top -= linesPerPage
		p.cursor -= linesPerPage
	case keyDown:
		p.cursor++
		if p.cursor >= p.top+linesPerPage {
			p.top++
		}
	case keyUp:
		p.cursor--
		if p.cursor < p.top {
			p.top--
		}
	case keyLeft:
		p.cursorCol--
	case keyRight:
		p.cursorCol++
	case keyTop:
		p.top = 0
		p.cursor = 0
	case keyBottom:
		if len(p.lines) > 0 {
			p.top = len(p.lines) - 1
			p.cursor = len(p.lines) - 1
		}
	case keyG:
		if p.pendingG {
			p.top = 0
			p.cursor = 0
			p.pendingG = false
		} else {
			p.pendingG = true
		}
	case keySearch:
		p.lastSearch = k.text
		p.lastDir = searchForward
		p.searchForward()
	case keySearchBackward:
		p.lastSearch = k.text
		p.lastDir = searchBackward
		p.searchBackward()
	case keySearchNext:
		if p.lastDir == searchBackward {
			p.searchBackward()
		} else {
			p.searchForward()
		}
	case keySearchPrev:
		if p.lastDir == searchBackward {
			p.searchForward()
		} else {
			p.searchBackward()
		}
	case keySearchWord:
		word := p.wordUnderCursor()
		if word != "" {
			p.lastSearch = word
			p.lastDir = searchForward
			p.searchForward()
		} else {
			p.status = "no word under cursor"
		}
	case keySearchWordBackward:
		word := p.wordUnderCursor()
		if word != "" {
			p.lastSearch = word
			p.lastDir = searchBackward
			p.searchBackward()
		} else {
			p.status = "no word under cursor"
		}
	}
	p.clampCursor()
	return false
}

func (p *pagerState) searchForward() {
	if p.lastSearch == "" || len(p.lines) == 0 {
		p.status = "no previous search"
		return
	}
	start := p.cursor + 1
	if start < 0 {
		start = 0
	}
	for i := start; i < len(p.plain); i++ {
		if strings.Contains(p.plain[i], p.lastSearch) {
			p.top = i
			p.cursor = i
			p.status = ""
			return
		}
	}
	p.status = "pattern not found"
}

func (p *pagerState) searchBackward() {
	if p.lastSearch == "" || len(p.lines) == 0 {
		p.status = "no previous search"
		return
	}
	start := p.cursor - 1
	if start >= len(p.plain) {
		start = len(p.plain) - 1
	}
	for i := start; i >= 0; i-- {
		if strings.Contains(p.plain[i], p.lastSearch) {
			p.top = i
			p.cursor = i
			p.status = ""
			return
		}
	}
	p.status = "pattern not found"
}

type keyType int

const (
	keyUnknown keyType = iota
	keyQuit
	keyPageDown
	keyPageUp
	keyDown
	keyUp
	keyTop
	keyBottom
	keyG
	keySearch
	keySearchBackward
	keySearchNext
	keySearchPrev
	keySearchWord
	keySearchWordBackward
	keyLeft
	keyRight
)

type key struct {
	typ  keyType
	text string
}

type searchDir int

const (
	searchForward searchDir = iota
	searchBackward
)

func readKey(r *bufio.Reader, w *bufio.Writer, p *pagerState) key {
	b, err := r.ReadByte()
	if err != nil {
		return key{typ: keyQuit}
	}
	switch b {
	case 'q', 'Q':
		return key{typ: keyQuit}
	case ' ':
		return key{typ: keyPageDown}
	case 'b':
		return key{typ: keyPageUp}
	case 0x06: // ctrl-f
		return key{typ: keyPageDown}
	case 0x02: // ctrl-b
		return key{typ: keyPageUp}
	case 'j', '\n':
		return key{typ: keyDown}
	case 'k':
		return key{typ: keyUp}
	case 'g':
		return key{typ: keyG}
	case 'G':
		return key{typ: keyBottom}
	case 'h':
		return key{typ: keyLeft}
	case 'l':
		return key{typ: keyRight}
	case 'n':
		return key{typ: keySearchNext}
	case 'N':
		return key{typ: keySearchPrev}
	case '*':
		return key{typ: keySearchWord}
	case '#':
		return key{typ: keySearchWordBackward}
	case '/':
		text := readSearch(r, w, p, "/")
		if text == "" {
			return key{typ: keyUnknown}
		}
		p.appendHistory(text)
		return key{typ: keySearch, text: text}
	case '?':
		text := readSearch(r, w, p, "?")
		if text == "" {
			return key{typ: keyUnknown}
		}
		p.appendHistory(text)
		return key{typ: keySearchBackward, text: text}
	case 0x1b:
		if seq, _ := r.ReadByte(); seq == '[' {
			if code, _ := r.ReadByte(); code == 'A' {
				return key{typ: keyUp}
			} else if code == 'B' {
				return key{typ: keyDown}
			} else if code == '5' { // PageUp
				_, _ = r.ReadByte()
				return key{typ: keyPageUp}
			} else if code == '6' { // PageDown
				_, _ = r.ReadByte()
				return key{typ: keyPageDown}
			} else if code == 'C' {
				return key{typ: keyRight}
			} else if code == 'D' {
				return key{typ: keyLeft}
			}
		}
	}
	return key{typ: keyUnknown}
}

func readSearch(r *bufio.Reader, w *bufio.Writer, p *pagerState, prefix string) string {
	var buf []byte
	p.status = prefix
	p.histIndex = len(p.history)
	p.drawStatus(w)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return ""
		}
		switch b {
		case '\n', '\r':
			p.status = ""
			return string(buf)
		case 0x7f, 0x08:
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
			p.status = prefix + string(buf)
			p.drawStatus(w)
		default:
			if b >= 0x20 {
				buf = append(buf, b)
				p.status = prefix + string(buf)
				p.drawStatus(w)
			}
		}
		if b == 0x1b {
			if seq, _ := r.ReadByte(); seq == '[' {
				if code, _ := r.ReadByte(); code == 'A' {
					buf = p.historyPrev(buf)
					p.status = prefix + string(buf)
					p.drawStatus(w)
					continue
				} else if code == 'B' {
					buf = p.historyNext(buf)
					p.status = prefix + string(buf)
					p.drawStatus(w)
					continue
				}
			}
			p.status = ""
			return ""
		}
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			b.WriteByte(s[i])
			continue
		}
		if i+1 >= len(s) {
			break
		}
		next := s[i+1]
		if next == '[' {
			i += 2
			for i < len(s) {
				c := s[i]
				if c >= 0x40 && c <= 0x7e {
					break
				}
				i++
			}
			continue
		}
		if next == ']' {
			i += 2
			for i < len(s) {
				if s[i] == 0x07 {
					break
				}
				if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
					i++
					break
				}
				i++
			}
			continue
		}
	}
	return b.String()
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

	type span struct {
		start int
		end   int
	}
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
		spans = append(spans, span{start: start, end: end})
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
		b.WriteString("\033[7m")
		b.WriteString(original[sp.start:sp.end])
		b.WriteString("\033[0m")
		last = sp.end
	}
	b.WriteString(original[last:])
	return b.String()
}

func findAllMatches(s, pattern string) []int {
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

func visibleIndexMap(s string) []int {
	var idx []int
	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			idx = append(idx, i)
			continue
		}
		if i+1 >= len(s) {
			break
		}
		next := s[i+1]
		if next == '[' {
			i += 2
			for i < len(s) {
				c := s[i]
				if c >= 0x40 && c <= 0x7e {
					break
				}
				i++
			}
			continue
		}
		if next == ']' {
			i += 2
			for i < len(s) {
				if s[i] == 0x07 {
					break
				}
				if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
					i++
					break
				}
				i++
			}
			continue
		}
	}
	return idx
}

func truncateStatus(s string) string {
	if len(s) <= 200 {
		return s
	}
	return s[:200]
}

func (p *pagerState) clampCursor() {
	if len(p.lines) == 0 {
		p.cursor = 0
		p.cursorCol = 0
		p.top = 0
		return
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.lines) {
		p.cursor = len(p.lines) - 1
	}
	lineLen := len(p.plain[p.cursor])
	if p.cursorCol < 0 {
		p.cursorCol = 0
	}
	if p.cursorCol > lineLen {
		p.cursorCol = lineLen
	}
	linesPerPage := p.height - 1
	if linesPerPage <= 0 {
		linesPerPage = 1
	}
	if p.cursor < p.top {
		p.top = p.cursor
	}
	if p.cursor >= p.top+linesPerPage {
		p.top = p.cursor - linesPerPage + 1
	}
	if p.top < 0 {
		p.top = 0
	}
}

func (p *pagerState) appendHistory(s string) {
	if s == "" {
		return
	}
	if len(p.history) > 0 && p.history[len(p.history)-1] == s {
		return
	}
	p.history = append(p.history, s)
}

func (p *pagerState) historyPrev(cur []byte) []byte {
	if len(p.history) == 0 {
		return cur
	}
	if p.histIndex > 0 {
		p.histIndex--
	}
	return []byte(p.history[p.histIndex])
}

func (p *pagerState) historyNext(cur []byte) []byte {
	if len(p.history) == 0 {
		return cur
	}
	if p.histIndex < len(p.history)-1 {
		p.histIndex++
		return []byte(p.history[p.histIndex])
	}
	p.histIndex = len(p.history)
	return []byte{}
}

func (p *pagerState) wordUnderCursor() string {
	if len(p.lines) == 0 {
		return ""
	}
	line := p.plain[p.cursor]
	if line == "" {
		return ""
	}
	col := p.cursorCol
	if col >= len(line) && len(line) > 0 {
		col = len(line) - 1
	}
	if col < 0 || col >= len(line) {
		return ""
	}
	if !isWordChar(line[col]) {
		return ""
	}
	start := col
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}
	end := col + 1
	for end < len(line) && isWordChar(line[end]) {
		end++
	}
	return line[start:end]
}

func isWordChar(b byte) bool {
	if b >= 'a' && b <= 'z' {
		return true
	}
	if b >= 'A' && b <= 'Z' {
		return true
	}
	if b >= '0' && b <= '9' {
		return true
	}
	return b == '_'
}
