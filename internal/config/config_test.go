package config

import (
	"os"
	"path/filepath"
	"strings"
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
	if cfg.Pager.Mode != "less" {
		t.Errorf("default mode = %q, want %q", cfg.Pager.Mode, "less")
	}
	if cfg.Mermaid.Theme != "auto" {
		t.Errorf("default theme = %q, want %q", cfg.Mermaid.Theme, "auto")
	}
}

func TestLoad_VimMode(t *testing.T) {
	setupTestConfig(t, `{"pager": {"mode": "vim"}}`)
	cfg := Load()
	if cfg.Pager.Mode != "vim" {
		t.Errorf("mode = %q, want %q", cfg.Pager.Mode, "vim")
	}
}

func TestLoad_EmptyModeDefaultsToLess(t *testing.T) {
	setupTestConfig(t, `{"pager": {"mode": ""}}`)
	cfg := Load()
	if cfg.Pager.Mode != "less" {
		t.Errorf("mode = %q, want %q", cfg.Pager.Mode, "less")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	setupTestConfig(t, `{invalid json}`)
	cfg := Load()
	if cfg.Pager.Mode != "less" {
		t.Errorf("mode = %q, want %q on invalid JSON", cfg.Pager.Mode, "less")
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

func TestLoad_MermaidTheme(t *testing.T) {
	setupTestConfig(t, `{"mermaid": {"theme": "dark"}}`)
	cfg := Load()
	if cfg.Mermaid.Theme != "dark" {
		t.Errorf("theme = %q, want %q", cfg.Mermaid.Theme, "dark")
	}
}

func TestLoad_EmptyConfig(t *testing.T) {
	setupTestConfig(t, `{}`)
	cfg := Load()
	if cfg.Pager.Mode != "less" {
		t.Errorf("mode = %q, want %q", cfg.Pager.Mode, "less")
	}
	if cfg.Mermaid.Theme != "auto" {
		t.Errorf("theme = %q, want %q", cfg.Mermaid.Theme, "auto")
	}
}

func TestConfigPath(t *testing.T) {
	// configPath builds on os.UserConfigDir(). Regardless of platform the
	// result must point at glowm/config.json under the user config dir.
	got, err := configPath()
	if err != nil {
		t.Fatalf("configPath() error: %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(got), "glowm/config.json") {
		t.Errorf("configPath() = %q, want suffix glowm/config.json", got)
	}
}

func TestLoad_ConfigPathError(t *testing.T) {
	origFunc := configPathFunc
	configPathFunc = func() (string, error) {
		return "", os.ErrNotExist
	}
	t.Cleanup(func() { configPathFunc = origFunc })

	cfg := Load()
	if cfg.Pager.Mode != "less" {
		t.Errorf("mode = %q, want %q", cfg.Pager.Mode, "less")
	}
}
