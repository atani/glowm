package terminal

import (
	"os"

	"golang.org/x/term"
)

func IsTTYFile(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

func StdoutIsTTY() bool {
	return IsTTYFile(os.Stdout)
}

func StdinIsTTY() bool {
	return IsTTYFile(os.Stdin)
}

func StdoutWidth(defaultWidth int) int {
	if !StdoutIsTTY() {
		return defaultWidth
	}
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		return defaultWidth
	}
	return width
}
