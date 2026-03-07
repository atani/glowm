package termimage

import (
	"encoding/base64"
	"os"
	"strconv"
	"strings"
)

type Format int

const (
	FormatNone Format = iota
	FormatIterm2
	FormatKitty
)

func Detect() Format {
	if isIterm2() {
		return FormatIterm2
	}
	if isKitty() {
		return FormatKitty
	}
	return FormatNone
}

func Encode(format Format, png []byte) string {
	return EncodeWithWidth(format, png, 0)
}

func EncodeWithWidth(format Format, png []byte, widthCells int) string {
	switch format {
	case FormatIterm2:
		return encodeIterm2(png, widthCells)
	case FormatKitty:
		return encodeKitty(png, widthCells)
	default:
		return ""
	}
}

func isIterm2() bool {
	return os.Getenv("TERM_PROGRAM") == "iTerm.app"
}

func isKitty() bool {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	return strings.Contains(os.Getenv("TERM"), "xterm-kitty")
}

func encodeIterm2(png []byte, widthCells int) string {
	b64 := base64.StdEncoding.EncodeToString(png)
	meta := "inline=1;preserveAspectRatio=1"
	if widthCells > 0 {
		meta += ";width=" + strconv.Itoa(widthCells)
	}
	return "\x1b]1337;File=" + meta + ":" + b64 + "\x07"
}

func encodeKitty(png []byte, widthCells int) string {
	b64 := base64.StdEncoding.EncodeToString(png)
	const chunkSize = 4096
	var b strings.Builder
	for i := 0; i < len(b64); i += chunkSize {
		end := i + chunkSize
		if end > len(b64) {
			end = len(b64)
		}
		more := "0"
		if end < len(b64) {
			more = "1"
		}
		if i == 0 {
			// First chunk: include full control parameters.
			b.WriteString("\x1b_Gf=100,a=T,")
			if widthCells > 0 {
				b.WriteString("c=")
				b.WriteString(strconv.Itoa(widthCells))
				b.WriteString(",")
			}
			b.WriteString("m=")
			b.WriteString(more)
		} else {
			// Subsequent chunks: only continuation flag.
			b.WriteString("\x1b_Gm=")
			b.WriteString(more)
		}
		b.WriteString(";")
		b.WriteString(b64[i:end])
		b.WriteString("\x1b\\")
	}
	return b.String()
}
