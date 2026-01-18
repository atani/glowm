package termimage

import (
	"strings"
)

func ReplaceMarkersWithImages(output string, markers []string, images [][]byte, format Format, widthCells int) string {
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
			i += 2
			for i < len(s) {
				ch := s[i]
				if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
					break
				}
				i++
			}
			continue
		}
		if next == ']' {
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
	}
	return b.String()
}
