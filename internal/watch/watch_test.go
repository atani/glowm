package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStatSigChange(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.md")
	if err := os.WriteFile(p, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	s1 := statSig(p)
	if !s1.exists {
		t.Fatal("expected file to exist")
	}

	// Size change.
	if err := os.WriteFile(p, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	s2 := statSig(p)
	if s1.equal(s2) {
		t.Error("size change should not be equal")
	}

	// Missing file.
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	s3 := statSig(p)
	if s3.exists {
		t.Error("removed file should not exist")
	}
	if s2.equal(s3) {
		t.Error("existing vs missing should not be equal")
	}
}

func TestStatSigSameContentSameSig(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.md")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := statSig(p)
	b := statSig(p)
	if !a.equal(b) {
		t.Error("two stats of an unchanged file should be equal")
	}
}

func TestWatcherEmitsOnChange(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.md")
	if err := os.WriteFile(p, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Fast poll/debounce for a quick test.
	w := New(p, 10*time.Millisecond, 10*time.Millisecond)
	w.Start()
	defer w.Stop()

	// No change yet -> no emit.
	select {
	case <-w.C:
		t.Fatal("unexpected emit before any change")
	case <-time.After(60 * time.Millisecond):
	}

	if err := os.WriteFile(p, []byte("v2 longer"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.C:
		// good
	case <-time.After(2 * time.Second):
		t.Fatal("expected an emit after the file changed")
	}
}
