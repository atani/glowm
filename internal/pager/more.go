package pager

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
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

	lines, heights := splitDisplayLines(output)
	reader, shouldClose := openTTYReader()
	if shouldClose {
		defer reader.Close()
	}

	oldState, err := term.MakeRaw(int(reader.Fd()))
	if err != nil {
		return printOutput(output)
	}
	defer term.Restore(int(reader.Fd()), oldState)
	defer setupSignalHandler(int(reader.Fd()), oldState, nil)()

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
		end := pageEnd(heights, linePos, linesPerPage)
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

func splitDisplayLines(output string) ([]string, []int) {
	lines := strings.Split(output, "\n")
	heights := make([]int, len(lines))
	for i, line := range lines {
		clean, rows := consumePagerRowsMarker(line)
		lines[i] = clean
		heights[i] = rows
	}
	return lines, heights
}

func pageEnd(heights []int, start, linesPerPage int) int {
	if start >= len(heights) {
		return len(heights)
	}
	used := 0
	for end := start; end < len(heights); end++ {
		rows := heights[end]
		if rows < 1 {
			rows = 1
		}
		if used > 0 && used+rows > linesPerPage {
			return end
		}
		used += rows
		if used >= linesPerPage {
			return end + 1
		}
	}
	return len(heights)
}

func consumePagerRowsMarker(line string) (string, int) {
	const prefix = "\x1b]1337;glowm-rows="
	start := strings.Index(line, prefix)
	if start == -1 {
		return line, 1
	}
	valueStart := start + len(prefix)
	valueEnd := valueStart
	for valueEnd < len(line) && line[valueEnd] >= '0' && line[valueEnd] <= '9' {
		valueEnd++
	}
	if valueEnd == valueStart || valueEnd >= len(line) || line[valueEnd] != '\x07' {
		return line, 1
	}
	rows, err := strconv.Atoi(line[valueStart:valueEnd])
	if err != nil || rows < 1 {
		rows = 1
	}
	return line[:start] + line[valueEnd+1:], rows
}
