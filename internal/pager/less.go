package pager

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// less.go implements a less-like pager that scrolls a viewport (rather than
// moving a cursor as vim mode does) while preserving inline terminal images.
//
// Image safety: like more mode, it never re-flows or width-trims a line. Each
// display line is reprinted verbatim, and image lines carry a glowm-rows height
// (parsed by splitDisplayLines) so paging math reserves the right number of
// terminal rows and never splits an image across a screen boundary. Navigation
// repaints the screen, which re-emits the image escapes intact.

// keyType values local to less mode (half-page motions). Offset well past the
// shared keyType constants defined in vim.go to avoid collisions.
const (
	keyHalfDown keyType = iota + 1000
	keyHalfUp
)

func pageLess(output string) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return printOutput(output)
	}

	height := terminalHeight()
	if height <= 0 {
		return printOutput(output)
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
	defer setupSignalHandler(int(reader.Fd()), oldState, func() {
		fmt.Fprint(os.Stdout, ansiAltScreenOff)
	})()

	bufReader := bufio.NewReader(reader)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	// Alternate screen buffer preserves the shell's scrollback.
	fmt.Fprint(writer, ansiAltScreenOn)
	defer fmt.Fprint(os.Stdout, ansiAltScreenOff)

	p := newLessState(output, height)

	p.redraw(writer)
	prev := p.viewKey()
	for {
		quit := p.handleKey(readLessKey(bufReader, writer, p))
		// Coalesce already-arrived input (fast key-repeat / mouse wheel) into a
		// single repaint.
		for !quit && bufReader.Buffered() > 0 {
			quit = p.handleKey(readLessKey(bufReader, writer, p))
		}
		if quit {
			break
		}
		if cur := p.viewKey(); cur != prev {
			p.redraw(writer)
			prev = cur
		}
	}
	return nil
}

// viewKey returns a value that changes whenever the visible frame would change,
// so the render loop can skip no-op repaints.
func (p *lessState) viewKey() string {
	return fmt.Sprintf("%d|%s|%s", p.top, p.lastSearch, p.status)
}

type lessState struct {
	lines      []string
	heights    []int
	plain      []string
	isImage    []bool
	height     int
	top        int // index of the logical line at the top of the viewport
	lastSearch string
	lastDir    searchDir
	status     string
}

// newLessState builds a text-mode less pager from rendered output. Image lines
// (height > 1, or carrying a raw image escape) are flagged so they are never
// searched or highlighted, since injecting reverse-video into image payload
// bytes would corrupt them.
func newLessState(output string, height int) *lessState {
	lines, heights := splitDisplayLines(output)
	plain := make([]string, len(lines))
	isImage := make([]bool, len(lines))
	for i, line := range lines {
		if heights[i] > 1 || strings.Contains(line, "\x1bP") || strings.Contains(line, "\x1b]1337") {
			isImage[i] = true
			continue
		}
		plain[i] = stripANSI(line)
	}
	return &lessState{
		lines:   lines,
		heights: heights,
		plain:   plain,
		isImage: isImage,
		height:  height,
	}
}

// applyContent rebuilds the pager from freshly rendered output (used by watch
// mode), preserving the current scroll position.
func (p *lessState) applyContent(c Content) {
	np := newLessState(c.Output, p.height)
	p.lines = np.lines
	p.heights = np.heights
	p.plain = np.plain
	p.isImage = np.isImage
	p.status = ""
	p.clampTop()
}

func (p *lessState) linesPerPage() int {
	n := p.height - 1 // reserve the bottom row for the status line
	if n <= 0 {
		return 1
	}
	return n
}

// h returns the display-row height of line i (clamped to at least 1).
func (p *lessState) h(i int) int {
	if i < 0 || i >= len(p.heights) {
		return 1
	}
	if p.heights[i] < 1 {
		return 1
	}
	return p.heights[i]
}

func (p *lessState) redraw(w *bufio.Writer) {
	fmt.Fprint(w, syncBegin)
	fmt.Fprint(w, ansiClearScreen)
	end := pageEnd(p.heights, p.top, p.linesPerPage())
	for i := p.top; i < end; i++ {
		line := p.lines[i]
		if p.lastSearch != "" && !p.isImage[i] {
			line = highlightLine(line, p.plain[i], p.lastSearch)
		}
		fmt.Fprint(w, "\r")
		fmt.Fprint(w, line)
		fmt.Fprint(w, "\r\n")
	}
	p.drawStatus(w, end)
	fmt.Fprint(w, syncEnd)
	w.Flush()
}

func (p *lessState) drawStatus(w *bufio.Writer, end int) {
	width := terminalWidth()
	if width <= 0 {
		width = 80
	}
	total := len(p.lines)
	status := p.status
	if status == "" {
		percent := 100
		if total > 0 {
			percent = end * 100 / total
		}
		if percent > 100 {
			percent = 100
		}
		first := p.top + 1
		if total == 0 {
			first = 0
		}
		status = fmt.Sprintf(
			"glowm  %d/%d (%d%%)  (q quit, j/k line, space/b page, d/u half, g/G top/bottom, / ? search, n/N)",
			first, total, percent,
		)
	}
	fmt.Fprintf(w, "\033[%d;1H", p.height)
	fmt.Fprint(w, ansiReverseVideo)
	fmt.Fprint(w, trimPlainToWidth(status, width))
	fmt.Fprint(w, ansiReset)
	fmt.Fprint(w, "\r"+ansiClearToEOL)
}

func (p *lessState) handleKey(k key) bool {
	lpp := p.linesPerPage()
	switch k.typ {
	case keyQuit:
		return true
	case keyPageDown:
		p.top = pageEnd(p.heights, p.top, lpp)
	case keyPageUp:
		p.top = p.retreat(p.top, lpp)
	case keyHalfDown:
		p.top = p.advance(p.top, lpp/2)
	case keyHalfUp:
		p.top = p.retreat(p.top, lpp/2)
	case keyDown:
		p.top++
	case keyUp:
		p.top--
	case keyTop:
		p.top = 0
	case keyBottom:
		p.top = p.maxTop()
	case keySearch:
		p.lastSearch = k.text
		p.lastDir = dirForward
		p.search(dirForward)
	case keySearchBackward:
		p.lastSearch = k.text
		p.lastDir = dirBackward
		p.search(dirBackward)
	case keySearchNext:
		p.search(p.lastDir)
	case keySearchPrev:
		p.search(p.lastDir.reverse())
	}
	p.clampTop()
	return false
}

// advance moves the top index forward until at least `rows` display rows have
// been passed, returning the new top.
func (p *lessState) advance(from, rows int) int {
	if rows < 1 {
		rows = 1
	}
	used := 0
	i := from
	for i < len(p.lines) && used < rows {
		used += p.h(i)
		i++
	}
	return i
}

// retreat moves the top index backward until at least `rows` display rows have
// been passed, returning the new top.
func (p *lessState) retreat(from, rows int) int {
	if rows < 1 {
		rows = 1
	}
	used := 0
	i := from
	for i > 0 && used < rows {
		i--
		used += p.h(i)
	}
	return i
}

// maxTop returns the largest top index that still fills the final page, so that
// scrolling stops with the last line resting at the bottom of the viewport.
func (p *lessState) maxTop() int {
	lpp := p.linesPerPage()
	used := 0
	i := len(p.lines)
	for i > 0 {
		h := p.h(i - 1)
		if used+h > lpp && used > 0 {
			break
		}
		used += h
		i--
	}
	if i >= len(p.lines) {
		i = len(p.lines) - 1
	}
	if i < 0 {
		i = 0
	}
	return i
}

func (p *lessState) clampTop() {
	if p.top < 0 {
		p.top = 0
	}
	if mt := p.maxTop(); p.top > mt {
		p.top = mt
	}
}

// search scans for lastSearch starting just past the current top, in the given
// direction, wrapping around. Image lines are skipped. On a hit, the matching
// line is moved to the top of the viewport.
func (p *lessState) search(dir searchDir) {
	if p.lastSearch == "" || len(p.lines) == 0 {
		p.status = "no previous search"
		return
	}
	n := len(p.lines)
	step := 1
	if dir == dirBackward {
		step = -1
	}
	for i := 1; i <= n; i++ {
		idx := ((p.top+i*step)%n + n) % n
		if !p.isImage[idx] && p.plain[idx] != "" && strings.Contains(p.plain[idx], p.lastSearch) {
			p.top = idx
			p.status = ""
			return
		}
	}
	p.status = "pattern not found: " + p.lastSearch
}

func readLessKey(r io.ByteReader, w *bufio.Writer, p *lessState) key {
	b, err := r.ReadByte()
	if err != nil {
		return key{typ: keyQuit}
	}
	switch b {
	case 'q', 'Q':
		return key{typ: keyQuit}
	case ' ', 'f', 0x06: // space, f, ctrl-f
		return key{typ: keyPageDown}
	case 'b', 0x02: // b, ctrl-b
		return key{typ: keyPageUp}
	case 'd', 0x04: // d, ctrl-d
		return key{typ: keyHalfDown}
	case 'u', 0x15: // u, ctrl-u
		return key{typ: keyHalfUp}
	case 'j', '\n', '\r', 'e':
		return key{typ: keyDown}
	case 'k', 'y':
		return key{typ: keyUp}
	case 'g', '<':
		return key{typ: keyTop}
	case 'G', '>':
		return key{typ: keyBottom}
	case 'n':
		return key{typ: keySearchNext}
	case 'N':
		return key{typ: keySearchPrev}
	case '/':
		text := readLessSearch(r, w, p, "/")
		if text == "" {
			return key{typ: keyUnknown}
		}
		return key{typ: keySearch, text: text}
	case '?':
		text := readLessSearch(r, w, p, "?")
		if text == "" {
			return key{typ: keyUnknown}
		}
		return key{typ: keySearchBackward, text: text}
	case 0x1b:
		return readEscapeKey(r) // arrow keys -> up/down/page up/page down
	}
	return key{typ: keyUnknown}
}

// readLessSearch reads a search pattern at the bottom prompt until Enter, or
// returns "" if cancelled with Esc.
func readLessSearch(r io.ByteReader, w *bufio.Writer, p *lessState, prefix string) string {
	width := terminalWidth()
	if width <= 0 {
		width = 80
	}
	var buf []byte
	p.status = prefix
	p.drawStatus(w, pageEnd(p.heights, p.top, p.linesPerPage()))
	w.Flush()
	for {
		b, err := r.ReadByte()
		if err != nil {
			p.status = ""
			return ""
		}
		switch b {
		case '\n', '\r':
			p.status = ""
			return string(buf)
		case 0x7f, 0x08: // backspace / DEL
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
			p.status = prefix + string(buf)
		case 0x1b: // Esc cancels; swallow a trailing arrow-key sequence if present.
			if next, _ := r.ReadByte(); next == '[' {
				_, _ = r.ReadByte()
			}
			p.status = ""
			return ""
		default:
			if b >= 0x20 {
				buf = append(buf, b)
				p.status = prefix + string(buf)
			}
		}
		p.drawStatus(w, pageEnd(p.heights, p.top, p.linesPerPage()))
		w.Flush()
	}
}
