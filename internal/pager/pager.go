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

	ttyFile, err := os.Open("/dev/tty")
	if err != nil {
		// No TTY available — print without paging.
		_, err := fmt.Fprint(os.Stdout, output)
		return err
	}
	defer ttyFile.Close()

	fd := int(ttyFile.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		_, err := fmt.Fprint(os.Stdout, output)
		return err
	}
	defer term.Restore(fd, oldState)

	lines := strings.Split(output, "\n")
	bufReader := bufio.NewReader(ttyFile)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	linesPerPage := height - 1
	if linesPerPage <= 0 {
		linesPerPage = 1
	}

	for i := 0; i < len(lines); i++ {
		fmt.Fprint(writer, lines[i])
		fmt.Fprint(writer, "\r\n")
		if (i+1)%linesPerPage == 0 && i < len(lines)-1 {
			fmt.Fprint(writer, "--More--")
			writer.Flush()
			b, err := bufReader.ReadByte()
			fmt.Fprint(writer, "\r\033[K")
			if err != nil || b == 'q' || b == 'Q' {
				return nil
			}
		}
	}
	return nil
}

func terminalHeight() int {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || h <= 0 {
		return 0
	}
	return h
}
