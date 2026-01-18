package input

import (
	"errors"
	"io"
	"os"
)

var ErrNoInput = errors.New("input is required")

func Read(args []string) (string, error) {
	if len(args) == 0 {
		if stdinHasData() {
			return readStdin()
		}
		return "", ErrNoInput
	}
	if len(args) > 1 {
		return "", errors.New("only one input is supported")
	}
	if args[0] == "-" {
		return readStdin()
	}
	b, err := os.ReadFile(args[0])
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func readStdin() (string, error) {
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func stdinHasData() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}
