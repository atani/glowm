package pager

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"

	"github.com/atani/glowm/internal/termimage"
)

// watch_pager.go drives the less pagers in watch mode: an event loop that waits
// on both keystrokes and a reload signal, so the view can refresh in place when
// the source file changes. The non-watch pagers (pageLess/pageLessKitty) keep
// their simple blocking loops; this code is additive.

// Content is a freshly rendered document handed to a watch-mode pager. For the
// Kitty path, Output still contains the raw marker lines and Images/Markers are
// populated; for the text path, Output is the final rendered text and the image
// fields are empty.
type Content struct {
	Output     string
	Markers    []string
	Images     [][]byte
	WidthCells int
}

// RenderFunc produces the current Content (e.g. by re-reading and re-rendering
// the source file). It is called once per reload signal.
type RenderFunc func() (Content, error)

// chanReader adapts a byte channel to io.ByteReader, with a one-byte pushback so
// the event loop can peek the first byte of a key (via select) and still hand a
// complete reader to the key parser.
type chanReader struct {
	ch          <-chan byte
	pending     byte
	havePending bool
}

func (c *chanReader) ReadByte() (byte, error) {
	if c.havePending {
		c.havePending = false
		return c.pending, nil
	}
	b, ok := <-c.ch
	if !ok {
		return 0, io.EOF
	}
	return b, nil
}

// runReload is the shared event loop. A goroutine pumps raw terminal bytes into
// byteCh; the loop selects between a reload signal and the next keystroke. While
// parsing a single key (which may read several bytes, e.g. an escape sequence or
// a search prompt) it reads straight from byteCh, so a reload only interrupts
// between keys. All terminal writes happen here on the main goroutine.
func runReload(
	bufReader *bufio.Reader,
	reload <-chan struct{},
	render RenderFunc,
	parseAndHandle func(r io.ByteReader) (quit bool),
	apply func(Content),
	setStatus func(string),
	redraw func(),
	viewKey func() string,
) {
	byteCh := make(chan byte, 1024)
	done := make(chan struct{})
	go func() {
		for {
			b, err := bufReader.ReadByte()
			if err != nil {
				close(byteCh)
				return
			}
			select {
			case byteCh <- b:
			case <-done:
				return
			}
		}
	}()
	defer close(done)

	redraw()
	prev := viewKey()
	for {
		select {
		case <-reload:
			c, err := render()
			if err != nil {
				setStatus("reload failed: " + err.Error())
			} else {
				apply(c)
			}
			redraw()
			prev = viewKey()
		case b, ok := <-byteCh:
			if !ok {
				return
			}
			r := &chanReader{ch: byteCh, pending: b, havePending: true}
			quit := parseAndHandle(r)
			// Coalesce input that already arrived into a single repaint.
			for !quit && len(byteCh) > 0 {
				quit = parseAndHandle(r)
			}
			if quit {
				return
			}
			if cur := viewKey(); cur != prev {
				redraw()
				prev = cur
			}
		}
	}
}

// PageLessKittyWatch runs the smooth Kitty pager in watch mode.
func PageLessKittyWatch(initial Content, reload <-chan struct{}, render RenderFunc) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return printOutput(termimage.ReplaceMarkersWithImages(initial.Output, initial.Markers, initial.Images, termimage.FormatKitty, initial.WidthCells))
	}
	height := terminalHeight()
	if height <= 0 {
		return printOutput(termimage.ReplaceMarkersWithImages(initial.Output, initial.Markers, initial.Images, termimage.FormatKitty, initial.WidthCells))
	}

	reader, shouldClose := openTTYReader()
	if shouldClose {
		defer reader.Close()
	}
	oldState, err := term.MakeRaw(int(reader.Fd()))
	if err != nil {
		return printOutput(termimage.ReplaceMarkersWithImages(initial.Output, initial.Markers, initial.Images, termimage.FormatKitty, initial.WidthCells))
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

	p := newLessKittyState(initial.Output, initial.Markers, initial.Images, initial.WidthCells, height)
	runReload(bufReader, reload, render,
		func(r io.ByteReader) bool { return p.handleKey(readKittyKey(r, writer, p)) },
		func(c Content) { p.applyContent(c) },
		func(s string) { p.status = s },
		func() { p.redraw(writer) },
		p.viewKey,
	)
	return nil
}

// PageLessWatch runs the text-mode less pager in watch mode.
func PageLessWatch(initial Content, reload <-chan struct{}, render RenderFunc) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return printOutput(initial.Output)
	}
	height := terminalHeight()
	if height <= 0 {
		return printOutput(initial.Output)
	}

	reader, shouldClose := openTTYReader()
	if shouldClose {
		defer reader.Close()
	}
	oldState, err := term.MakeRaw(int(reader.Fd()))
	if err != nil {
		return printOutput(initial.Output)
	}
	defer term.Restore(int(reader.Fd()), oldState)
	defer setupSignalHandler(int(reader.Fd()), oldState, func() {
		fmt.Fprint(os.Stdout, ansiAltScreenOff)
	})()

	bufReader := bufio.NewReader(reader)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()
	fmt.Fprint(writer, ansiAltScreenOn)
	defer fmt.Fprint(os.Stdout, ansiAltScreenOff)

	p := newLessState(initial.Output, height)
	runReload(bufReader, reload, render,
		func(r io.ByteReader) bool { return p.handleKey(readLessKey(r, writer, p)) },
		func(c Content) { p.applyContent(c) },
		func(s string) { p.status = s },
		func() { p.redraw(writer) },
		p.viewKey,
	)
	return nil
}
