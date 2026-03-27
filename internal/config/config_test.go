package config

import (
	"os"
	"path/filepath"
	"testing"
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
	if cfg.Pager.Mode != "more" {
		t.Errorf("default mode = %q, want %q", cfg.Pager.Mode, "more")
	}
}

func TestLoad_VimMode(t *testing.T) {
	setupTestConfig(t, `{"pager": {"mode": "vim"}}`)
	cfg := Load()
	if cfg.Pager.Mode != "vim" {
		t.Errorf("mode = %q, want %q", cfg.Pager.Mode, "vim")
	}
}

func TestLoad_EmptyModeDefaultsToMore(t *testing.T) {
	setupTestConfig(t, `{"pager": {"mode": ""}}`)
	cfg := Load()
	if cfg.Pager.Mode != "more" {
		t.Errorf("mode = %q, want %q", cfg.Pager.Mode, "more")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	setupTestConfig(t, `{invalid json}`)
	cfg := Load()
	if cfg.Pager.Mode != "more" {
		t.Errorf("mode = %q, want %q on invalid JSON", cfg.Pager.Mode, "more")
	}
}

func TestLoad_UnknownModePassedThrough(t *testing.T) {
	setupTestConfig(t, `{"pager": {"mode": "emacs"}}`)
	cfg := Load()
	// Config does not validate mode; that is the caller's responsibility
	if cfg.Pager.Mode != "emacs" {
		t.Errorf("mode = %q, want %q", cfg.Pager.Mode, "emacs")
	}
}

func TestLoad_MoreMode(t *testing.T) {
	setupTestConfig(t, `{"pager": {"mode": "more"}}`)
	cfg := Load()
	if cfg.Pager.Mode != "more" {
		t.Errorf("mode = %q, want %q", cfg.Pager.Mode, "more")
	}
}

func TestLoad_EmptyConfig(t *testing.T) {
	setupTestConfig(t, `{}`)
	cfg := Load()
	if cfg.Pager.Mode != "more" {
		t.Errorf("mode = %q, want %q", cfg.Pager.Mode, "more")
	}
}

func TestLoad_ConfigPathError(t *testing.T) {
	origFunc := configPathFunc
	configPathFunc = func() (string, error) {
		return "", os.ErrNotExist
	}
	t.Cleanup(func() { configPathFunc = origFunc })

	cfg := Load()
	if cfg.Pager.Mode != "more" {
		t.Errorf("mode = %q, want %q", cfg.Pager.Mode, "more")
	}
}
