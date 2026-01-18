package pager

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

func Page(output string) error {
	height := terminalHeight()
	if height <= 0 {
		_, err := fmt.Fprint(os.Stdout, output)
		return err
	}

	lines := strings.Split(output, "\n")
	reader := openTTYReader()
	defer reader.Close()

	bufReader := bufio.NewReader(reader)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	linesPerPage := height - 1
	if linesPerPage <= 0 {
		linesPerPage = 1
	}

	for i := 0; i < len(lines); i++ {
		fmt.Fprintln(writer, lines[i])
		if (i+1)%linesPerPage == 0 && i < len(lines)-1 {
			fmt.Fprint(writer, "--More--")
			writer.Flush()
			b, _ := bufReader.ReadByte()
			fmt.Fprint(writer, "\r\033[K")
			if b == 'q' || b == 'Q' {
				return nil
			}
		}
	}
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
