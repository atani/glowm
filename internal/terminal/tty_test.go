package terminal

import (
	"os"
	"testing"
)

func TestIsTTYFile_Nil(t *testing.T) {
	if IsTTYFile(nil) {
		t.Error("IsTTYFile(nil) should be false")
	}
}

func TestIsTTYFile_RegularFile(t *testing.T) {
	// A regular temp file is never a TTY.
	f, err := os.CreateTemp(t.TempDir(), "notty")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if IsTTYFile(f) {
		t.Error("IsTTYFile(regular file) should be false")
	}
}

func TestStdoutIsTTY_DoesNotPanic(t *testing.T) {
	// Under `go test` stdout is generally not a TTY; the call must simply
	// return a bool without panicking regardless of environment.
	_ = StdoutIsTTY()
	_ = StdinIsTTY()
}

func TestStdoutWidth_NonTTYReturnsDefault(t *testing.T) {
	// When stdout is not a terminal (the usual test environment), the default
	// width must be returned unchanged.
	if !StdoutIsTTY() {
		if got := StdoutWidth(123); got != 123 {
			t.Errorf("StdoutWidth(123) on non-TTY = %d, want 123", got)
		}
	}
}
