package markdown

import (
	"strings"
	"testing"
)

func TestExtractMermaidKeepBlocks(t *testing.T) {
	md := "# Title\n\n```mermaid\nflowchart TD\n  A-->B\n```\n\nText\n"
	res := ExtractMermaid(md, true)
	if len(res.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(res.Blocks))
	}
	if res.Blocks[0] == "" {
		t.Fatalf("expected block content")
	}
	if res.Markdown == "" {
		t.Fatalf("expected markdown output")
	}
	if !strings.Contains(res.Markdown, "```mermaid") {
		t.Fatalf("expected mermaid block preserved")
	}
}

func TestExtractMermaidReplaceBlocks(t *testing.T) {
	md := "# Title\n\n```mermaid\nflowchart TD\n  A-->B\n```\n\nText\n"
	res := ExtractMermaid(md, false)
	if len(res.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(res.Blocks))
	}
	if strings.Contains(res.Markdown, "```mermaid") {
		t.Fatalf("expected mermaid block removed")
	}
	if !strings.Contains(res.Markdown, Placeholder) {
		t.Fatalf("expected placeholder in output")
	}
}

func TestExtractMermaidWithMarkers(t *testing.T) {
	md := "```mermaid\nA-->B\n```\n"
	res := ExtractMermaidWithMarkers(md)
	if len(res.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(res.Blocks))
	}
	if len(res.Markers) != 1 {
		t.Fatalf("expected 1 marker, got %d", len(res.Markers))
	}
	if !strings.Contains(res.Markdown, res.Markers[0]) {
		t.Fatalf("expected marker in output")
	}
}

func TestExtractMermaidMultipleBlocks(t *testing.T) {
	md := "```mermaid\nA-->B\n```\n\n```mermaid\nB-->C\n```\n"
	res := ExtractMermaid(md, false)
	if len(res.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(res.Blocks))
	}
}
