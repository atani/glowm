package markdown

import (
	"strings"
	"testing"
)

func TestExtractMermaidKeepBlocks(t *testing.T) {
	md := "# Title\n\n```mermaid\nflowchart TD\n  A-->B\n```\n\nText\n"
	res, err := ExtractMermaid(md, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	res, err := ExtractMermaid(md, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	res, err := ExtractMermaidWithMarkers(md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
	res, err := ExtractMermaid(md, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(res.Blocks))
	}
}

func TestExtractMermaidUnterminatedFence(t *testing.T) {
	md := "```mermaid\nA-->B\n"
	res, err := ExtractMermaid(md, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Blocks) != 0 {
		t.Fatalf("expected 0 blocks for unterminated fence, got %d", len(res.Blocks))
	}
	// Original content should be preserved in output (no orphan placeholder).
	if strings.Contains(res.Markdown, Placeholder) {
		t.Fatal("expected no placeholder for unterminated fence")
	}
	if !strings.Contains(res.Markdown, "```mermaid") {
		t.Fatal("expected original fence line restored in output")
	}
	if !strings.Contains(res.Markdown, "A-->B") {
		t.Fatal("expected original content restored in output")
	}
}

func TestExtractMermaidUnterminatedFenceMarkers(t *testing.T) {
	md := "```mermaid\nA-->B\n"
	res, err := ExtractMermaidWithMarkers(md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Blocks) != 0 {
		t.Fatalf("expected 0 blocks for unterminated fence, got %d", len(res.Blocks))
	}
	if len(res.Markers) != 0 {
		t.Fatalf("expected 0 markers for unterminated fence, got %d", len(res.Markers))
	}
	// No orphan marker text in output.
	if strings.Contains(res.Markdown, MarkerPrefix) {
		t.Fatal("expected no orphan marker in output")
	}
}

func TestExtractMermaidTildeFence(t *testing.T) {
	md := "~~~mermaid\nA-->B\n~~~\n"
	res, err := ExtractMermaid(md, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(res.Blocks))
	}
	if res.Blocks[0] != "A-->B" {
		t.Fatalf("expected 'A-->B', got %q", res.Blocks[0])
	}
}

func TestExtractMermaidEmptyBlock(t *testing.T) {
	md := "```mermaid\n```\n"
	res, err := ExtractMermaid(md, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(res.Blocks))
	}
	if res.Blocks[0] != "" {
		t.Fatalf("expected empty block, got %q", res.Blocks[0])
	}
}

func TestExtractMermaidNoBlocks(t *testing.T) {
	md := "# Hello\n\nSome text.\n\n```go\nfmt.Println(\"hi\")\n```\n"
	res, err := ExtractMermaid(md, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(res.Blocks))
	}
}

func TestExtractMermaidLongerClosingFence(t *testing.T) {
	// A closing fence with more backticks than the opener is valid.
	md := "```mermaid\nA-->B\n`````\n"
	res, err := ExtractMermaid(md, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(res.Blocks))
	}
}

func TestExtractMermaidFenceWithInfoString(t *testing.T) {
	md := "```mermaid {theme: dark}\nA-->B\n```\n"
	res, err := ExtractMermaid(md, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(res.Blocks))
	}
}

func TestExtractMermaidIndentedFence(t *testing.T) {
	// CommonMark allows up to 3 spaces of indentation before a fence.
	md := "   ```mermaid\nA-->B\n   ```\n"
	res, err := ExtractMermaid(md, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(res.Blocks))
	}
	if res.Blocks[0] != "A-->B" {
		t.Fatalf("expected 'A-->B', got %q", res.Blocks[0])
	}
}

func TestExtractMermaidIndentedFenceClosing(t *testing.T) {
	// Opening at column 0, closing indented.
	md := "```mermaid\nA-->B\n  ```\n"
	res, err := ExtractMermaid(md, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(res.Blocks))
	}
}
