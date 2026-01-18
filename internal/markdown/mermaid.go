package markdown

import (
	"bufio"
	"strings"
)

const Placeholder = "[mermaid diagram omitted]"
const MarkerPrefix = "GLOWM_MERMAID_"

type MermaidResult struct {
	Blocks   []string
	Markdown string
	Markers  []string
}

func ExtractMermaid(md string, keepBlocks bool) MermaidResult {
	return extractMermaid(md, keepBlocks, false)
}

func ExtractMermaidWithMarkers(md string) MermaidResult {
	return extractMermaid(md, false, true)
}

func extractMermaid(md string, keepBlocks bool, useMarkers bool) MermaidResult {
	var out strings.Builder
	var blocks []string
	var markers []string

	scanner := bufio.NewScanner(strings.NewReader(md))
	inFence := false
	fence := ""
	isMermaid := false
	var current []string

	for scanner.Scan() {
		line := scanner.Text()

		if !inFence {
			f, ok := fenceStart(line)
			if ok {
				inFence = true
				fence = f
				info := strings.TrimSpace(strings.TrimPrefix(line, fence))
				isMermaid = strings.HasPrefix(info, "mermaid")
				if isMermaid && !keepBlocks {
					if useMarkers {
						marker := MarkerPrefix + itoa(len(markers))
						markers = append(markers, marker)
						out.WriteString(marker)
						out.WriteString("\n")
					} else {
						out.WriteString(Placeholder)
						out.WriteString("\n")
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
		if strings.HasPrefix(line, fence) {
			if isMermaid {
				blocks = append(blocks, strings.Join(current, "\n"))
			}
			if !isMermaid || keepBlocks {
				out.WriteString(line)
				out.WriteString("\n")
			}
			inFence = false
			fence = ""
			isMermaid = false
			current = nil
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

	if inFence && isMermaid {
		blocks = append(blocks, strings.Join(current, "\n"))
	}

	return MermaidResult{Blocks: blocks, Markdown: out.String(), Markers: markers}
}

func fenceStart(line string) (string, bool) {
	if strings.HasPrefix(line, "```") {
		return "```", true
	}
	if strings.HasPrefix(line, "~~~") {
		return "~~~", true
	}
	return "", false
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	n := len(b)
	for i > 0 {
		n--
		b[n] = byte('0' + i%10)
		i /= 10
	}
	return string(b[n:])
}
