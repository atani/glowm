package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atani/glowm/internal/pager"
	"github.com/atani/glowm/internal/termimage"
)

func TestParseFlags_Defaults(t *testing.T) {
	opts, err := parseFlags(nil)
	if err != nil {
		t.Fatalf("parseFlags(nil) error: %v", err)
	}
	if opts.width != 0 || opts.style != "auto" || opts.usePager || opts.noPager || opts.pdf || opts.showVersion || opts.showLinkURLs {
		t.Errorf("unexpected defaults: %+v", opts)
	}
}

func TestParseFlags_ShowLinkURLs(t *testing.T) {
	opts, err := parseFlags([]string{"-show-link-urls", "file.md"})
	if err != nil {
		t.Fatalf("parseFlags error: %v", err)
	}
	if !opts.showLinkURLs {
		t.Errorf("showLinkURLs = false, want true")
	}
}

func TestParseFlags_AllSet(t *testing.T) {
	opts, err := parseFlags([]string{"-w", "100", "-s", "dark", "-p", "-no-pager", "file.md"})
	if err != nil {
		t.Fatalf("parseFlags error: %v", err)
	}
	if opts.width != 100 {
		t.Errorf("width = %d, want 100", opts.width)
	}
	if opts.style != "dark" {
		t.Errorf("style = %q, want dark", opts.style)
	}
	if !opts.usePager || !opts.noPager {
		t.Errorf("pager flags not set: %+v", opts)
	}
	if len(opts.positional) != 1 || opts.positional[0] != "file.md" {
		t.Errorf("positional = %v, want [file.md]", opts.positional)
	}
}

func TestParseFlags_Invalid(t *testing.T) {
	_, err := parseFlags([]string{"-nonexistent-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestRun_Version(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"-version"}, &out, &errBuf)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "glowm") {
		t.Errorf("version output missing 'glowm': %q", out.String())
	}
}

func TestRun_InvalidFlag(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{"-bogus"}, &out, &errBuf)
	if code != 2 {
		t.Errorf("exit code = %d, want 2 for invalid flag", code)
	}
}

func TestRun_RendersFileToStdout(t *testing.T) {
	// Non-TTY path: a markdown file is rendered and written to the provided
	// stdout writer. Under `go test` stdout is not a terminal, so this
	// exercises the common text-only branch end to end.
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(mdPath, []byte("# Heading\n\nbody paragraph\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errBuf bytes.Buffer
	code := run([]string{"-s", "notty", mdPath}, &out, &errBuf)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "Heading") {
		t.Errorf("output missing heading: %q", out.String())
	}
	if !strings.Contains(out.String(), "body paragraph") {
		t.Errorf("output missing body: %q", out.String())
	}
}

func TestRun_MissingFile(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := run([]string{filepath.Join(t.TempDir(), "does-not-exist.md")}, &out, &errBuf)
	if code != 1 {
		t.Errorf("exit code = %d, want 1 for missing file", code)
	}
	if errBuf.Len() == 0 {
		t.Error("expected an error message on stderr")
	}
}

func TestRun_NoInput(t *testing.T) {
	// No positional args and stdin is not a pipe (interactive test env) ->
	// input.Read returns ErrNoInput -> exit 1. Skip when stdin happens to be
	// a pipe (e.g. CI feeding stdin) to avoid blocking on a read.
	stat, err := os.Stdin.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		t.Skip("stdin is a pipe in this environment; skipping no-input case")
	}
	var out, errBuf bytes.Buffer
	code := run(nil, &out, &errBuf)
	if code != 1 {
		t.Errorf("exit code = %d, want 1 when no input", code)
	}
}

func TestFail(t *testing.T) {
	var errBuf bytes.Buffer
	code := fail(&errBuf, errors.New("boom"))
	if code != 1 {
		t.Errorf("fail() code = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "boom") {
		t.Errorf("fail() stderr = %q, want to contain 'boom'", errBuf.String())
	}
}

func TestRun_PDFFlagNoBlocks(t *testing.T) {
	// The -pdf flag with no mermaid blocks exits 1 via the runPDF path.
	dir := t.TempDir()
	mdPath := filepath.Join(dir, "plain.md")
	if err := os.WriteFile(mdPath, []byte("# just text\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errBuf bytes.Buffer
	code := run([]string{"-pdf", mdPath}, &out, &errBuf)
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(errBuf.String(), "no mermaid blocks") {
		t.Errorf("stderr = %q", errBuf.String())
	}
}

func TestRunPDF_NoMermaidBlocks(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := runPDF("# Just text, no diagrams\n", "", &out, &errBuf)
	if code != 1 {
		t.Errorf("code = %d, want 1 when no mermaid blocks", code)
	}
	if !strings.Contains(errBuf.String(), "no mermaid blocks") {
		t.Errorf("stderr = %q, want 'no mermaid blocks'", errBuf.String())
	}
}

func TestReplaceMarkersForPagerMode(t *testing.T) {
	markers := []string{"GLOWM_MERMAID_0"}
	images := [][]byte{[]byte("png")}
	output := "GLOWM_MERMAID_0"

	more := replaceMarkersForPagerMode(output, markers, images, termimage.FormatKitty, 80, pager.ModeMore)
	if !strings.Contains(more, "glowm-rows=1") {
		t.Fatalf("more mode should include pager row metadata: %q", more)
	}

	vim := replaceMarkersForPagerMode(output, markers, images, termimage.FormatKitty, 80, pager.ModeVim)
	if strings.Contains(vim, "glowm-rows=") {
		t.Fatalf("vim mode should not include pager row metadata: %q", vim)
	}
}
