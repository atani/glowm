package input

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRead_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte("# Hello"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := Read([]string{path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "# Hello" {
		t.Fatalf("expected '# Hello', got %q", got)
	}
}

func TestRead_NonexistentFile(t *testing.T) {
	_, err := Read([]string{"/nonexistent/file.md"})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestRead_MultipleArgs(t *testing.T) {
	_, err := Read([]string{"a.md", "b.md"})
	if err == nil {
		t.Fatal("expected error for multiple arguments")
	}
	if !strings.Contains(err.Error(), "only one input") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRead_NoArgsNoStdin(t *testing.T) {
	// When args is empty and stdin is a terminal (no data), should return ErrNoInput.
	// In test environment, stdin is typically not a pipe, so stdinHasData() returns false.
	_, err := Read(nil)
	if err != ErrNoInput {
		t.Fatalf("expected ErrNoInput, got %v", err)
	}
}

func TestRead_StdinDash(t *testing.T) {
	// Simulate reading from stdin by replacing os.Stdin with a pipe.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	go func() {
		w.WriteString("piped content")
		w.Close()
	}()

	got, err := Read([]string{"-"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "piped content" {
		t.Fatalf("expected 'piped content', got %q", got)
	}
}

func TestRead_FileTooLarge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.md")
	// Create a file that exceeds maxInputSize by writing a header.
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	// Truncate to maxInputSize + 1 byte to trigger the limit.
	if err := f.Truncate(maxInputSize + 1); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	_, err = Read([]string{path})
	if err == nil {
		t.Fatal("expected error for large file")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("unexpected error: %v", err)
	}
}
