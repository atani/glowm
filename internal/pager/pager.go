package pager

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// Mode represents the pager display mode.
type Mode string

const (
	ModeMore Mode = "more"
	ModeVim  Mode = "vim"
)

// ValidMode returns true if the given mode is recognized.
func ValidMode(m Mode) bool {
	switch m {
	case ModeMore, ModeVim:
		return true
	default:
		return false
	}
}

// ParseMode normalizes and validates a mode string.
// Returns the mode and true if valid, or ModeMore and false if not.
func ParseMode(s string) (Mode, bool) {
	m := Mode(strings.ToLower(s))
	if ValidMode(m) {
		return m, true
	}
	return ModeMore, false
}

// Page displays output using the default more-style pager.
func Page(output string) error {
	return PageWithMode(output, ModeMore)
}

// PageWithMode displays output using the specified pager mode.
func PageWithMode(output string, mode Mode) error {
	switch mode {
	case ModeVim:
		return pageVim(output)
	default:
		return pageMore(output)
	}
}

func terminalHeight() int {
	_, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || h <= 0 {
		return 0
	}
	return h
}

func terminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 0
	}
	return w
}

// openTTYReader opens /dev/tty for direct terminal input.
// Returns the file and true if /dev/tty was opened (caller should close),
// or os.Stdin and false if /dev/tty is unavailable (caller must NOT close).
func openTTYReader() (*os.File, bool) {
	f, err := os.Open("/dev/tty")
	if err != nil {
		return os.Stdin, false
	}
	return f, true
}

func printOutput(output string) error {
	_, err := fmt.Fprint(os.Stdout, output)
	return err
}
