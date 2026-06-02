package pager

import (
	"testing"

	"golang.org/x/term"
)

// TestSetupSignalHandler verifies the handler installs and the returned
// cleanup function stops the handler goroutine without firing a signal.
// The signal-receive branch is not exercised because it calls os.Exit.
func TestSetupSignalHandler_InstallAndCleanup(t *testing.T) {
	var restored bool
	cleanup := setupSignalHandler(-1, &term.State{}, func() { restored = true })
	if cleanup == nil {
		t.Fatal("expected a non-nil cleanup function")
	}
	// Calling cleanup must stop the goroutine and return promptly.
	cleanup()
	// extraCleanup is only invoked on signal receipt, so it must not have run.
	if restored {
		t.Error("extraCleanup should not run during normal cleanup")
	}
}

func TestSetupSignalHandler_NilExtraCleanup(t *testing.T) {
	// A nil extraCleanup must be accepted; cleanup must still work.
	cleanup := setupSignalHandler(-1, &term.State{}, nil)
	cleanup()
}

// TestOpenTTYReader verifies openTTYReader returns a usable file in both the
// /dev/tty-available and fallback-to-stdin cases. The shouldClose flag tells
// the caller whether it owns the returned file.
func TestOpenTTYReader(t *testing.T) {
	f, shouldClose := openTTYReader()
	if f == nil {
		t.Fatal("expected a non-nil file")
	}
	if shouldClose {
		// We own /dev/tty; close it to avoid leaking the descriptor.
		f.Close()
	}
	// When shouldClose is false the returned file is os.Stdin, which the
	// caller must not close.
}
