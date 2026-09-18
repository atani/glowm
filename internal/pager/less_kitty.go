package pager

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/atani/glowm/internal/termimage"
)

// less_kitty.go implements the less pager for terminals that speak the Kitty
// graphics protocol (Kitty, Ghostty). Unlike the atomic pageLess/pageMore paths,
// it tracks scroll position in display *rows* rather than logical lines, so an
// image can scroll partially off the top or bottom edge: each redraw emits a
// vertically-cropped Kitty placement (EncodeKittyCrop) for only the visible
// slice of each image. This removes the "blank gap then the whole image pops
// into frame" behavior of the atomic pagers.

// kittyDeleteAll removes all transmitted images and placements. Emitted at the
// start of every redraw (and on exit) so re-transmitting cropped slices each
// frame does not leak image memory in the terminal.
const kittyDeleteAll = "\x1b_Ga=d,d=A\x1b\\"

// Synchronized output (DEC mode 2026): the terminal buffers everything between
// begin and end and presents it as a single atomic frame, so a clear-then-
// repaint is never shown half-finished. Ghostty and Kitty support it; on
// terminals that don't, these are ignored. This is the main flicker fix.
const (
	syncBegin = "\x1b[?2026h"
	syncEnd   = "\x1b[?2026l"
)

// PageLessKitty pages output (which still contains the raw marker lines, not
// baked-in image escapes) in less mode with smooth image scrolling. images is
// indexed by marker order; widthCells is the render width used to size images.
func PageLessKitty(output string, markers []string, images [][]byte, widthCells int) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		// Not a terminal: fall back to a plain dump with full images inlined.
		return printOutput(termimage.ReplaceMarkersWithImages(output, markers, images, termimage.FormatKitty, widthCells))
	}
	height := terminalHeight()
	if height <= 0 {
		return printOutput(termimage.ReplaceMarkersWithImages(output, markers, images, termimage.FormatKitty, widthCells))
	}

	reader, shouldClose := openTTYReader()
	if shouldClose {
		defer reader.Close()
	}

	oldState, err := term.MakeRaw(int(reader.Fd()))
	if err != nil {
		return printOutput(termimage.ReplaceMarkersWithImages(output, markers, images, termimage.FormatKitty, widthCells))
	}
	defer term.Restore(int(reader.Fd()), oldState)
	defer setupSignalHandler(int(reader.Fd()), oldState, func() {
		fmt.Fprint(os.Stdout, kittyDeleteAll+ansiAltScreenOff)
	})()

	bufReader := bufio.NewReader(reader)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	fmt.Fprint(writer, ansiAltScreenOn)
	defer fmt.Fprint(os.Stdout, kittyDeleteAll+ansiAltScreenOff)

	p := newLessKittyState(output, markers, images, widthCells, height)
	p.redraw(writer)
	prev := p.viewKey()
	for {
		quit := p.handleKey(readKittyKey(bufReader, writer, p))
		// Coalesce input that already arrived (fast key-repeat / mouse wheel)
		// into a single repaint instead of one repaint per event.
		for !quit && bufReader.Buffered() > 0 {
			quit = p.handleKey(readKittyKey(bufReader, writer, p))
		}
		if quit {
			break
		}
		// Skip the repaint entirely if nothing visible changed (e.g. scrolling
		// past the top/bottom edge).
		if cur := p.viewKey(); cur != prev {
			p.redraw(writer)
			prev = cur
		}
	}
	return nil
}

// viewKey returns a value that changes whenever the visible frame would change,
// so the render loop can skip no-op repaints.
func (p *lessKittyState) viewKey() string {
	return fmt.Sprintf("%d|%s|%s", p.rowTop, p.lastSearch, p.status)
}

// lessSeg is one logical line: either a text line (rows == 1) or an image
// occupying rows terminal rows.
type lessSeg struct {
	isImage bool
	text    string // rendered text (with ANSI), for text segments
	plain   string // ANSI-stripped text, for search/highlight
	imgIdx  int    // index into images, for image segments
	rows    int    // display-row height
}

type lessKittyState struct {
	segs       []lessSeg
	images     [][]byte
	widthCells int
	rowStart   []int // flat display-row index at which each segment begins
	totalRows  int
	height     int
	rowTop     int // top of the viewport, in display rows
	lastSearch string
	lastDir    searchDir
	status     string
}

func newLessKittyState(output string, markers []string, images [][]byte, widthCells, height int) *lessKittyState {
	// Match the marker-replacement convention in termimage: a line is an image
	// if its ANSI-stripped, space-trimmed text equals a marker string.
	markerIdx := make(map[string]int, len(markers))
	for i, m := range markers {
		markerIdx[m] = i
	}

	rawLines := strings.Split(output, "\n")
	segs := make([]lessSeg, 0, len(rawLines))
	for _, line := range rawLines {
		trimmed := strings.TrimSpace(stripANSI(line))
		if idx, ok := markerIdx[trimmed]; ok && idx < len(images) {
			rows := termimage.ImageRows(images[idx], widthCells)
			if rows < 1 {
				rows = 1
			}
			segs = append(segs, lessSeg{isImage: true, imgIdx: idx, rows: rows})
			continue
		}
		segs = append(segs, lessSeg{text: line, plain: trimmed, rows: 1})
	}

	rowStart := make([]int, len(segs))
	total := 0
	for i := range segs {
		rowStart[i] = total
		total += segs[i].rows
	}

	return &lessKittyState{
		segs:       segs,
		images:     images,
		widthCells: widthCells,
		rowStart:   rowStart,
		totalRows:  total,
		height:     height,
	}
}

// applyContent rebuilds the pager from freshly rendered content (used by watch
// mode), preserving the current scroll position and active search.
func (p *lessKittyState) applyContent(c Content) {
	np := newLessKittyState(c.Output, c.Markers, c.Images, c.WidthCells, p.height)
	p.segs = np.segs
	p.images = np.images
	p.widthCells = np.widthCells
	p.rowStart = np.rowStart
	p.totalRows = np.totalRows
	p.status = ""
	p.clampTop()
}

func (p *lessKittyState) linesPerPage() int {
	n := p.height - 1 // reserve the bottom row for the status line
	if n <= 0 {
		return 1
	}
	return n
}

func (p *lessKittyState) maxRowTop() int {
	m := p.totalRows - p.linesPerPage()
	if m < 0 {
		return 0
	}
	return m
}

func (p *lessKittyState) clampTop() {
	if p.rowTop < 0 {
		p.rowTop = 0
	}
	if m := p.maxRowTop(); p.rowTop > m {
		p.rowTop = m
	}
}

// segAt returns the segment index containing display row `row`, and the row
// offset within that segment.
func (p *lessKittyState) segAt(row int) (int, int) {
	for i := range p.segs {
		if row < p.rowStart[i]+p.segs[i].rows {
			return i, row - p.rowStart[i]
		}
	}
	if len(p.segs) == 0 {
		return 0, 0
	}
	last := len(p.segs) - 1
	return last, p.segs[last].rows - 1
}

func (p *lessKittyState) redraw(w *bufio.Writer) {
	fmt.Fprint(w, syncBegin)
	fmt.Fprint(w, kittyDeleteAll)
	fmt.Fprint(w, ansiClearScreen)

	remaining := p.linesPerPage()
	screenRow := 1
	si, off := p.segAt(p.rowTop)
	for si < len(p.segs) && remaining > 0 {
		seg := p.segs[si]
		fmt.Fprintf(w, "\033[%d;1H", screenRow)
		if seg.isImage {
			show := seg.rows - off
			if show > remaining {
				show = remaining
			}
			fmt.Fprint(w, termimage.EncodeKittyCrop(p.images[seg.imgIdx], p.widthCells, seg.rows, off, show))
			screenRow += show
			remaining -= show
		} else {
			line := seg.text
			if p.lastSearch != "" {
				line = highlightLine(line, seg.plain, p.lastSearch)
			}
			fmt.Fprint(w, line)
			screenRow++
			remaining--
		}
		si++
		off = 0
	}
	p.drawStatus(w)
	fmt.Fprint(w, syncEnd)
	w.Flush()
}

func (p *lessKittyState) drawStatus(w *bufio.Writer) {
	width := terminalWidth()
	if width <= 0 {
		width = 80
	}
	status := p.status
	if status == "" {
		bottom := p.rowTop + p.linesPerPage()
		if bottom > p.totalRows {
			bottom = p.totalRows
		}
		percent := 100
		if p.totalRows > 0 {
			percent = bottom * 100 / p.totalRows
		}
		if percent > 100 {
			percent = 100
		}
		first := p.rowTop + 1
		if p.totalRows == 0 {
			first = 0
		}
		status = fmt.Sprintf(
			"glowm  %d/%d (%d%%)  (q quit, j/k line, space/b page, d/u half, g/G top/bottom, / ? search, n/N)",
			first, p.totalRows, percent,
		)
	}
	fmt.Fprintf(w, "\033[%d;1H", p.height)
	fmt.Fprint(w, ansiReverseVideo)
	fmt.Fprint(w, trimPlainToWidth(status, width))
	fmt.Fprint(w, ansiReset)
	fmt.Fprint(w, "\r"+ansiClearToEOL)
}

func (p *lessKittyState) handleKey(k key) bool {
	lpp := p.linesPerPage()
	switch k.typ {
	case keyQuit:
		return true
	case keyDown:
		p.rowTop++
	case keyUp:
		p.rowTop--
	case keyPageDown:
		p.rowTop += lpp
	case keyPageUp:
		p.rowTop -= lpp
	case keyHalfDown:
		p.rowTop += lpp / 2
	case keyHalfUp:
		p.rowTop -= lpp / 2
	case keyTop:
		p.rowTop = 0
	case keyBottom:
		p.rowTop = p.maxRowTop()
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

// search scans text segments for lastSearch in the given direction starting just
// past the segment currently at the top, wrapping around. On a hit, that
// segment's first row becomes the top of the viewport. Image segments are
// skipped.
func (p *lessKittyState) search(dir searchDir) {
	if p.lastSearch == "" || len(p.segs) == 0 {
		p.status = "no previous search"
		return
	}
	cur, _ := p.segAt(p.rowTop)
	n := len(p.segs)
	step := 1
	if dir == dirBackward {
		step = -1
	}
	for i := 1; i <= n; i++ {
		idx := ((cur+i*step)%n + n) % n
		s := p.segs[idx]
		if !s.isImage && s.plain != "" && strings.Contains(s.plain, p.lastSearch) {
			p.rowTop = p.rowStart[idx]
			p.status = ""
			return
		}
	}
	p.status = "pattern not found: " + p.lastSearch
}

func readKittyKey(r io.ByteReader, w *bufio.Writer, p *lessKittyState) key {
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
		text := readKittySearch(r, w, p, "/")
		if text == "" {
			return key{typ: keyUnknown}
		}
		return key{typ: keySearch, text: text}
	case '?':
		text := readKittySearch(r, w, p, "?")
		if text == "" {
			return key{typ: keyUnknown}
		}
		return key{typ: keySearchBackward, text: text}
	case 0x1b:
		return readEscapeKey(r) // arrow keys
	}
	return key{typ: keyUnknown}
}

func readKittySearch(r io.ByteReader, w *bufio.Writer, p *lessKittyState, prefix string) string {
	var buf []byte
	p.status = prefix
	p.drawStatus(w)
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
		p.drawStatus(w)
		w.Flush()
	}
}
