package termimage

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Format int

const (
	FormatNone Format = iota
	FormatIterm2
	FormatKitty
	FormatSixel
)

var (
	detectOnce  sync.Once
	detectedFmt Format
)

// Detect determines the terminal's image-protocol support. The result is
// cached so the DA1 probe inside isSixel() does not add latency on every
// subsequent call.
func Detect() Format {
	detectOnce.Do(func() {
		detectedFmt = detectUncached()
	})
	return detectedFmt
}

// resetDetectCache clears the cached Detect result. Intended for tests that
// vary terminal env vars between cases.
func resetDetectCache() {
	detectOnce = sync.Once{}
	detectedFmt = FormatNone
}

func detectUncached() Format {
	if isIterm2() {
		return FormatIterm2
	}
	if supportsKittyGraphics() {
		return FormatKitty
	}
	if isSixel() {
		return FormatSixel
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
	case FormatSixel:
		// TODO: widthCells is not honored for Sixel; go-sixel sizes output by
		// source pixels. Scaling would require pre-resizing the PNG.
		return encodeSixel(png)
	default:
		return ""
	}
}

func isIterm2() bool {
	return os.Getenv("TERM_PROGRAM") == "iTerm.app"
}

// supportsKittyGraphics reports whether the current terminal renders images
// via the Kitty graphics protocol. This covers Kitty itself, Ghostty (native),
// and Ghostty wrapped by tmux (which rewrites TERM_PROGRAM, so we rely on
// TMUX + GHOSTTY_RESOURCES_DIR together to avoid false positives when the
// env var leaks into unrelated children such as SSH with SendEnv).
func supportsKittyGraphics() bool {
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return true
	}
	term := os.Getenv("TERM")
	if strings.Contains(term, "xterm-kitty") {
		return true
	}
	if os.Getenv("TERM_PROGRAM") == "ghostty" || strings.Contains(term, "xterm-ghostty") {
		return true
	}
	if os.Getenv("TMUX") != "" && os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
		return true
	}
	return false
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
	params := "f=100,a=T"
	if widthCells > 0 {
		params += ",c=" + strconv.Itoa(widthCells)
	}
	return kittyEmit(png, params)
}

// ImagePixelSize returns the pixel dimensions of a PNG, or (0, 0) if it cannot
// be decoded.
func ImagePixelSize(png []byte) (int, int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(png))
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

// ImageRows returns the number of terminal rows an image occupies when scaled to
// widthCells columns.
func ImageRows(png []byte, widthCells int) int {
	return imageRows(png, widthCells)
}

// EncodeKittyCrop displays a vertical slice of a PNG via the Kitty graphics
// protocol: the slice skips skipRows rows from the top of the (widthCells x
// totalRows) full placement and shows showRows rows. C=1 keeps the cursor
// stationary so the caller controls placement with absolute positioning. This
// lets a pager scroll an image one row at a time instead of all-or-nothing.
func EncodeKittyCrop(png []byte, widthCells, totalRows, skipRows, showRows int) string {
	if showRows <= 0 || totalRows <= 0 {
		return ""
	}
	wpx, hpx := ImagePixelSize(png)
	if wpx <= 0 || hpx <= 0 {
		return ""
	}
	if skipRows < 0 {
		skipRows = 0
	}
	// Map the row range to source pixels, proportionally, preserving aspect.
	y := skipRows * hpx / totalRows
	h := showRows * hpx / totalRows
	if y >= hpx {
		return ""
	}
	if y+h > hpx {
		h = hpx - y
	}
	if h <= 0 {
		return ""
	}
	params := fmt.Sprintf("f=100,a=T,C=1,x=0,y=%d,w=%d,h=%d", y, wpx, h)
	if widthCells > 0 {
		params += ",c=" + strconv.Itoa(widthCells)
	}
	params += ",r=" + strconv.Itoa(showRows)
	return kittyEmit(png, params)
}

// kittyEmit base64-encodes png and writes it as one or more Kitty graphics
// escape sequences, placing firstParams on the first chunk and chunking the
// payload at 4096 bytes (the protocol limit) with the m= continuation flag.
func kittyEmit(png []byte, firstParams string) string {
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
			b.WriteString("\x1b_G")
			b.WriteString(firstParams)
			b.WriteString(",m=")
			b.WriteString(more)
		} else {
			b.WriteString("\x1b_Gm=")
			b.WriteString(more)
		}
		b.WriteString(";")
		b.WriteString(b64[i:end])
		b.WriteString("\x1b\\")
	}
	return b.String()
}
