package pager

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/term"
)

func pageVim(output string) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return printOutput(output)
	}

	height := terminalHeight()
	if height <= 0 {
		return printOutput(output)
	}

	lines := strings.Split(output, "\n")
	plainLines := make([]string, len(lines))
	for i := range lines {
		plainLines[i] = stripANSI(lines[i])
	}

	reader, shouldClose := openTTYReader()
	if shouldClose {
		defer reader.Close()
	}

	oldState, err := term.MakeRaw(int(reader.Fd()))
	if err != nil {
		return printOutput(output)
	}
	defer term.Restore(int(reader.Fd()), oldState)

	// Restore terminal state on SIGINT/SIGTERM to prevent leaving raw mode
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprint(os.Stdout, ansiAltScreenOff)
		term.Restore(int(reader.Fd()), oldState)
		os.Exit(0)
	}()
	defer signal.Stop(sigCh)

	bufReader := bufio.NewReader(reader)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	// Enter alternate screen buffer to preserve shell history
	fmt.Fprint(writer, ansiAltScreenOn)
	defer fmt.Fprint(os.Stdout, ansiAltScreenOff)

	p := &pagerState{
		lines:  lines,
		plain:  plainLines,
		height: height,
	}

	p.redraw(writer)
	for {
		key := readKey(bufReader, writer, p)
		if p.handleKey(key) {
			break
		}
		p.redraw(writer)
	}
	return nil
}

type pagerState struct {
	lines      []string
	plain      []string
	height     int
	top        int
	cursor     int
	lastSearch string
	lastDir    searchDir
	status     string
	history    []string
	histIndex  int
	awaitingGG bool // true after first 'g' press, waiting for second 'g' to go to top
}

func (p *pagerState) linesPerPage() int {
	n := p.height - 1
	if n <= 0 {
		return 1
	}
	return n
}

func (p *pagerState) redraw(w *bufio.Writer) {
	width := terminalWidth()
	if width <= 0 {
		width = 80
	}

	fmt.Fprint(w, ansiClearScreen)
	lpp := p.linesPerPage()
	end := p.top + lpp
	if end > len(p.lines) {
		end = len(p.lines)
	}
	for i := p.top; i < end; i++ {
		line := p.lines[i]
		if p.lastSearch != "" {
			line = highlightLine(line, p.plain[i], p.lastSearch)
		}
		line = trimANSIToWidth(line, width)
		if i == p.cursor {
			fmt.Fprint(w, "\r"+ansiReverseVideo)
			fmt.Fprint(w, line)
			fmt.Fprint(w, ansiReset+"\r\n")
		} else {
			fmt.Fprint(w, "\r")
			fmt.Fprint(w, line)
			fmt.Fprint(w, "\r\n")
		}
	}
	// Fill remaining lines
	for i := end - p.top; i < lpp; i++ {
		fmt.Fprint(w, "\r\n")
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
		status = fmt.Sprintf(
			"glowm pager  %d/%d  (q quit, / ? search, n/N next/prev, * # word, gg/G top/bottom, ctrl-f/ctrl-b page)",
			line, total,
		)
	}
	width := terminalWidth()
	if width <= 0 {
		width = 80
	}
	fmt.Fprintf(w, "\033[%d;1H", p.height)
	fmt.Fprint(w, ansiReverseVideo)
	fmt.Fprint(w, trimPlainToWidth(status, width))
	fmt.Fprint(w, ansiReset)
	fmt.Fprint(w, "\r"+ansiClearToEOL)
}

func (p *pagerState) handleKey(k key) bool {
	lpp := p.linesPerPage()

	if p.awaitingGG && k.typ != keyG {
		p.awaitingGG = false
	}

	switch k.typ {
	case keyQuit:
		return true
	case keyPageDown:
		p.top += lpp
		p.cursor += lpp
	case keyPageUp:
		p.top -= lpp
		p.cursor -= lpp
	case keyDown:
		p.cursor++
		if p.cursor >= p.top+lpp {
			p.top++
		}
	case keyUp:
		p.cursor--
		if p.cursor < p.top {
			p.top--
		}
	case keyTop:
		p.top = 0
		p.cursor = 0
	case keyBottom:
		if len(p.lines) > 0 {
			p.cursor = len(p.lines) - 1
			p.top = p.cursor - lpp + 1
			if p.top < 0 {
				p.top = 0
			}
		}
	case keyG:
		if p.awaitingGG {
			p.top = 0
			p.cursor = 0
			p.awaitingGG = false
		} else {
			p.awaitingGG = true
		}
	case keySearch:
		p.lastSearch = k.text
		p.lastDir = dirForward
		p.searchForward()
	case keySearchBackward:
		p.lastSearch = k.text
		p.lastDir = dirBackward
		p.searchBackward()
	case keySearchNext:
		if p.lastDir == dirBackward {
			p.searchBackward()
		} else {
			p.searchForward()
		}
	case keySearchPrev:
		if p.lastDir == dirBackward {
			p.searchForward()
		} else {
			p.searchBackward()
		}
	case keySearchWord:
		word := p.wordUnderCursor()
		if word != "" {
			p.lastSearch = word
			p.lastDir = dirForward
			p.searchForward()
		} else {
			p.status = "no word under cursor"
		}
	case keySearchWordBackward:
		word := p.wordUnderCursor()
		if word != "" {
			p.lastSearch = word
			p.lastDir = dirBackward
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
	// Search forward from cursor+1
	for i := p.cursor + 1; i < len(p.plain); i++ {
		if strings.Contains(p.plain[i], p.lastSearch) {
			p.top = i
			p.cursor = i
			p.status = ""
			return
		}
	}
	// Wrap around
	for i := 0; i <= p.cursor; i++ {
		if strings.Contains(p.plain[i], p.lastSearch) {
			p.top = i
			p.cursor = i
			p.status = "search hit BOTTOM, continuing at TOP"
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
	// Search backward from cursor-1
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
	// Wrap around
	for i := len(p.plain) - 1; i >= p.cursor; i-- {
		if strings.Contains(p.plain[i], p.lastSearch) {
			p.top = i
			p.cursor = i
			p.status = "search hit TOP, continuing at BOTTOM"
			return
		}
	}
	p.status = "pattern not found"
}

func (p *pagerState) clampCursor() {
	if len(p.lines) == 0 {
		p.cursor = 0
		p.top = 0
		return
	}
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.lines) {
		p.cursor = len(p.lines) - 1
	}
	lpp := p.linesPerPage()
	if p.cursor < p.top {
		p.top = p.cursor
	}
	if p.cursor >= p.top+lpp {
		p.top = p.cursor - lpp + 1
	}
	if p.top < 0 {
		p.top = 0
	}
}

func (p *pagerState) wordUnderCursor() string {
	if len(p.lines) == 0 {
		return ""
	}
	line := p.plain[p.cursor]
	if line == "" {
		return ""
	}
	// Find the first word on the current line
	start := 0
	for start < len(line) && !isWordChar(line[start]) {
		start++
	}
	if start >= len(line) {
		return ""
	}
	end := start + 1
	for end < len(line) && isWordChar(line[end]) {
		end++
	}
	return line[start:end]
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

// Key types and input handling

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
)

type key struct {
	typ  keyType
	text string
}

type searchDir int

const (
	dirForward searchDir = iota
	dirBackward
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
		return readEscapeKey(r)
	}
	return key{typ: keyUnknown}
}

func readEscapeKey(r *bufio.Reader) key {
	seq, _ := r.ReadByte()
	if seq != '[' {
		return key{typ: keyUnknown}
	}
	code, _ := r.ReadByte()
	switch code {
	case 'A':
		return key{typ: keyUp}
	case 'B':
		return key{typ: keyDown}
	case '5': // PageUp
		_, _ = r.ReadByte()
		return key{typ: keyPageUp}
	case '6': // PageDown
		_, _ = r.ReadByte()
		return key{typ: keyPageDown}
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
		case 0x1b:
			handled, newBuf := handleSearchEscape(r, w, p, prefix, buf)
			if handled {
				buf = newBuf
				continue
			}
			p.status = ""
			return ""
		default:
			if b >= 0x20 {
				buf = append(buf, b)
				p.status = prefix + string(buf)
				p.drawStatus(w)
			}
		}
	}
}

// handleSearchEscape processes escape sequences during search input.
// Returns true and updated buffer if an arrow key was handled,
// or false if the search should be cancelled.
func handleSearchEscape(r *bufio.Reader, w *bufio.Writer, p *pagerState, prefix string, buf []byte) (bool, []byte) {
	seq, _ := r.ReadByte()
	if seq != '[' {
		return false, buf
	}
	code, _ := r.ReadByte()
	switch code {
	case 'A': // Up arrow - history prev
		buf = p.historyPrev(buf)
		p.status = prefix + string(buf)
		p.drawStatus(w)
		return true, buf
	case 'B': // Down arrow - history next
		buf = p.historyNext(buf)
		p.status = prefix + string(buf)
		p.drawStatus(w)
		return true, buf
	}
	return false, buf
}
