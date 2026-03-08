package input

import (
	"errors"
	"fmt"
	"io"
	"os"
)

const maxInputSize = 10 * 1024 * 1024 // 10 MB

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
	info, err := os.Stat(args[0])
	if err != nil {
		return "", err
	}
	if info.Size() > maxInputSize {
		return "", fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), maxInputSize)
	}
	b, err := os.ReadFile(args[0])
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func readStdin() (string, error) {
	b, err := io.ReadAll(io.LimitReader(os.Stdin, maxInputSize+1))
	if err != nil {
		return "", err
	}
	if len(b) > maxInputSize {
		return "", fmt.Errorf("stdin input too large (max %d bytes)", maxInputSize)
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
