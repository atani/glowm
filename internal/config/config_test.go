package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/atani/glowm/internal/pager"
)

func setupTestConfig(t *testing.T, content string) {
	t.Helper()
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.json")
	if content != "" {
		if err := os.WriteFile(configFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	origFunc := configPathFunc
	configPathFunc = func() (string, error) { return configFile, nil }
	t.Cleanup(func() { configPathFunc = origFunc })
}

func TestLoad_DefaultsWhenNoFile(t *testing.T) {
	origFunc := configPathFunc
	configPathFunc = func() (string, error) {
		return filepath.Join(t.TempDir(), "nonexistent", "config.json"), nil
	}
	t.Cleanup(func() { configPathFunc = origFunc })

	cfg := Load()
	if cfg.Pager.Mode != pager.ModeMore {
		t.Errorf("default mode = %q, want %q", cfg.Pager.Mode, pager.ModeMore)
	}
}

func TestLoad_VimMode(t *testing.T) {
	setupTestConfig(t, `{"pager": {"mode": "vim"}}`)
	cfg := Load()
	if cfg.Pager.Mode != pager.ModeVim {
		t.Errorf("mode = %q, want %q", cfg.Pager.Mode, pager.ModeVim)
	}
}

func TestLoad_EmptyModeDefaultsToMore(t *testing.T) {
	setupTestConfig(t, `{"pager": {"mode": ""}}`)
	cfg := Load()
	if cfg.Pager.Mode != pager.ModeMore {
		t.Errorf("mode = %q, want %q", cfg.Pager.Mode, pager.ModeMore)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	setupTestConfig(t, `{invalid json}`)
	cfg := Load()
	if cfg.Pager.Mode != pager.ModeMore {
		t.Errorf("mode = %q, want %q on invalid JSON", cfg.Pager.Mode, pager.ModeMore)
	}
}

func TestLoad_UnknownMode(t *testing.T) {
	setupTestConfig(t, `{"pager": {"mode": "emacs"}}`)
	cfg := Load()
	if cfg.Pager.Mode != pager.ModeMore {
		t.Errorf("mode = %q, want %q on unknown mode", cfg.Pager.Mode, pager.ModeMore)
	}
}

func TestLoad_MoreMode(t *testing.T) {
	setupTestConfig(t, `{"pager": {"mode": "more"}}`)
	cfg := Load()
	if cfg.Pager.Mode != pager.ModeMore {
		t.Errorf("mode = %q, want %q", cfg.Pager.Mode, pager.ModeMore)
	}
}

func TestLoad_EmptyConfig(t *testing.T) {
	setupTestConfig(t, `{}`)
	cfg := Load()
	if cfg.Pager.Mode != pager.ModeMore {
		t.Errorf("mode = %q, want %q", cfg.Pager.Mode, pager.ModeMore)
	}
}
