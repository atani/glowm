package termimage

import (
	"bytes"
	"image"
	_ "image/png"
	"math"
	"strconv"
	"strings"
)

func ReplaceMarkersWithImages(output string, markers []string, images [][]byte, format Format, widthCells int) string {
	return replaceMarkersWithImages(output, markers, images, format, widthCells, false)
}

func ReplaceMarkersWithImagesForPager(output string, markers []string, images [][]byte, format Format, widthCells int) string {
	return replaceMarkersWithImages(output, markers, images, format, widthCells, true)
}

func replaceMarkersWithImages(output string, markers []string, images [][]byte, format Format, widthCells int, padToImageRows bool) string {
	if len(markers) == 0 || len(images) == 0 {
		return output
	}
	lookup := make(map[string]string, len(markers))
	for i, marker := range markers {
		if i >= len(images) {
			break
		}
		img := EncodeWithWidth(format, images[i], widthCells)
		if img == "" {
			continue
		}
		if padToImageRows {
			img = pagerRowsMarker(imageRows(images[i], widthCells)) + img
		}
		lookup[marker] = img
	}

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(stripANSI(line))
		if img, ok := lookup[trimmed]; ok {
			lines[i] = img
		}
	}
	return strings.Join(lines, "\n")
}

func pagerRowsMarker(rows int) string {
	return "\x1b]1337;glowm-rows=" + strconv.Itoa(rows) + "\x07"
}

func imageRows(png []byte, widthCells int) int {
	if widthCells <= 0 {
		return 1
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(png))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return 1
	}
	const cellAspect = 2.0 // terminal cells are roughly twice as tall as wide.
	rows := int(math.Ceil((float64(cfg.Height) / float64(cfg.Width)) * float64(widthCells) / cellAspect))
	if rows < 1 {
		return 1
	}
	return rows
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != 0x1b {
			b.WriteByte(c)
			continue
		}
		if i+1 >= len(s) {
			continue
		}
		next := s[i+1]
		if next == '[' {
			// CSI sequence: ESC [ ... <letter>
			i += 2
			for i < len(s) {
				ch := s[i]
				if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
					break
				}
				if ch == 0x1b {
					// Malformed: new escape before terminator.
					i--
					break
				}
				i++
			}
			continue
		}
		if next == ']' {
			// OSC sequence: ESC ] ... (BEL | ESC \)
			i += 2
			for i < len(s) {
				if s[i] == 0x07 {
					break
				}
				if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
					i++
					break
				}
				i++
			}
			continue
		}
		if next == '_' || next == 'P' || next == '^' {
			// APC / DCS / PM sequence: ESC <type> ... ESC \
			i += 2
			for i < len(s) {
				if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '\\' {
					i++
					break
				}
				i++
			}
			continue
		}
		// Two-byte or three-byte escape sequences.
		// ESC( ESC) ESC* ESC+ are followed by a character set designator byte.
		if next == '(' || next == ')' || next == '*' || next == '+' {
			i += 2 // skip ESC + type + designator
			continue
		}
		// Other two-byte: ESC + single byte (e.g. ESC 7, ESC 8, ESC =).
		i++
		continue
	}
	return b.String()
}
