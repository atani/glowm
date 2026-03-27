package pager

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

func pageMore(output string) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return printOutput(output)
	}

	height := terminalHeight()
	if height <= 0 {
		return printOutput(output)
	}

	lines := strings.Split(output, "\n")
	reader, shouldClose := openTTYReader()
	if shouldClose {
		defer reader.Close()
	}

	oldState, err := term.MakeRaw(int(reader.Fd()))
	if err != nil {
		return printOutput(output)
	}
	defer term.Restore(int(reader.Fd()), oldState)

	bufReader := bufio.NewReader(reader)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	linesPerPage := height - 1
	if linesPerPage <= 0 {
		linesPerPage = 1
	}

	linePos := 0
	shouldClearScreen := false
	for {
		if shouldClearScreen {
			fmt.Fprint(writer, ansiClearScreen)
			shouldClearScreen = false
		}
		end := linePos + linesPerPage
		if end > len(lines) {
			end = len(lines)
		}
		for j := linePos; j < end; j++ {
			fmt.Fprint(writer, "\r")
			fmt.Fprint(writer, lines[j])
			fmt.Fprint(writer, "\r\n")
		}
		linePos = end
		if linePos >= len(lines) {
			return nil
		}

		percent := (linePos * 100) / len(lines)
		fmt.Fprintf(writer, "--More--(%d%%)", percent)
		writer.Flush()
		b, _ := bufReader.ReadByte()
		fmt.Fprint(writer, "\r"+ansiClearToEOL)

		switch b {
		case 'q', 'Q':
			return nil
		case 'b', 'B':
			linePos -= linesPerPage * 2
			if linePos < 0 {
				linePos = 0
			}
			shouldClearScreen = true
		case '\r', '\n':
			if linePos < len(lines) {
				fmt.Fprint(writer, "\r")
				fmt.Fprint(writer, lines[linePos])
				fmt.Fprint(writer, "\r\n")
				linePos++
			}
		case ' ':
			// page forward (already printed one page)
		default:
			// ignore other keys
		}
	}
}
