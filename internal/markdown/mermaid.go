package markdown

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

const Placeholder = "[mermaid diagram omitted]"
const MarkerPrefix = "GLOWM_MERMAID_"

type MermaidResult struct {
	Blocks   []string
	Markdown string
	Markers  []string
}

func ExtractMermaid(md string, keepBlocks bool) (MermaidResult, error) {
	return extractMermaid(md, keepBlocks, false)
}

func ExtractMermaidWithMarkers(md string) (MermaidResult, error) {
	return extractMermaid(md, false, true)
}

func extractMermaid(md string, keepBlocks bool, useMarkers bool) (MermaidResult, error) {
	var out strings.Builder
	var blocks []string
	var markers []string

	scanner := bufio.NewScanner(strings.NewReader(md))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	inFence := false
	fence := ""
	isMermaid := false
	var current []string

	// Deferred marker: only written to output when fence closes successfully.
	var pendingLine string
	var originalFenceLine string
	markerIdx := -1

	for scanner.Scan() {
		line := scanner.Text()

		if !inFence {
			stripped := stripIndent(line)
			f, ok := fenceStart(stripped)
			if ok {
				inFence = true
				fence = f
				info := strings.TrimSpace(stripped[len(f):])
				isMermaid = strings.HasPrefix(info, "mermaid")
				if isMermaid && !keepBlocks {
					originalFenceLine = line
					if useMarkers {
						marker := MarkerPrefix + strconv.Itoa(len(markers))
						markerIdx = len(markers)
						markers = append(markers, marker)
						pendingLine = marker + "\n"
					} else {
						pendingLine = Placeholder + "\n"
					}
				} else {
					out.WriteString(line)
					out.WriteString("\n")
				}
				continue
			}
			out.WriteString(line)
			out.WriteString("\n")
			continue
		}

		// in fence
		if isFenceEnd(line, fence) {
			if isMermaid {
				blocks = append(blocks, strings.Join(current, "\n"))
				if pendingLine != "" {
					out.WriteString(pendingLine)
					pendingLine = ""
					originalFenceLine = ""
				}
			}
			if !isMermaid || keepBlocks {
				out.WriteString(line)
				out.WriteString("\n")
			}
			inFence = false
			fence = ""
			isMermaid = false
			current = nil
			markerIdx = -1
			continue
		}

		if inFence && isMermaid {
			current = append(current, line)
			if keepBlocks {
				out.WriteString(line)
				out.WriteString("\n")
			}
			continue
		}

		out.WriteString(line)
		out.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return MermaidResult{}, fmt.Errorf("scanning markdown: %w", err)
	}

	// Unclosed mermaid fence: discard the block, restore original content.
	if inFence && isMermaid {
		if useMarkers && markerIdx >= 0 {
			markers = markers[:markerIdx]
		}
		pendingLine = ""
		if !keepBlocks {
			out.WriteString(originalFenceLine)
			out.WriteString("\n")
			for _, line := range current {
				out.WriteString(line)
				out.WriteString("\n")
			}
		}
	}

	return MermaidResult{Blocks: blocks, Markdown: out.String(), Markers: markers}, nil
}

// stripIndent removes up to 3 leading spaces per CommonMark fence indentation rules.
func stripIndent(line string) string {
	n := 0
	for n < len(line) && n < 3 && line[n] == ' ' {
		n++
	}
	return line[n:]
}

// fenceStart detects the opening of a fenced code block and returns the
// fence string (e.g. "```" or "~~~~") along with true if found.
// The input line should already have leading indentation stripped.
func fenceStart(line string) (string, bool) {
	for _, ch := range []byte{'`', '~'} {
		if len(line) >= 3 && line[0] == ch && line[1] == ch && line[2] == ch {
			n := 3
			for n < len(line) && line[n] == ch {
				n++
			}
			return line[:n], true
		}
	}
	return "", false
}

// isFenceEnd checks whether line is a valid closing fence for the given
// opening fence. Per CommonMark, the closing fence must consist solely of
// the same character as the opening fence (with optional leading indentation
// up to 3 spaces and optional trailing spaces) and be at least as long as
// the opening fence.
func isFenceEnd(line, fence string) bool {
	stripped := stripIndent(line)
	trimmed := strings.TrimRight(stripped, " ")
	if len(trimmed) < len(fence) {
		return false
	}
	ch := fence[0]
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] != ch {
			return false
		}
	}
	return true
}
