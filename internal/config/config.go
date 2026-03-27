package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/atani/glowm/internal/pager"
)

// Config holds the application configuration.
type Config struct {
	Pager PagerConfig `json:"pager"`
}

// PagerConfig holds pager-specific settings.
type PagerConfig struct {
	Mode pager.Mode `json:"mode"`
}

// configPathFunc is the function used to determine the config file path.
// It can be overridden in tests.
var configPathFunc = configPath

// Load reads and returns the application config.
// Returns defaults if the config file is missing or unreadable.
// Warns to stderr if the file exists but is invalid.
func Load() Config {
	defaultCfg := Config{
		Pager: PagerConfig{
			Mode: pager.ModeMore,
		},
	}

	path, err := configPathFunc()
	if err != nil {
		return defaultCfg
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultCfg
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "glowm: warning: failed to parse %s: %v\n", path, err)
		return defaultCfg
	}
	if cfg.Pager.Mode == "" {
		cfg.Pager.Mode = pager.ModeMore
	} else if !pager.ValidMode(cfg.Pager.Mode) {
		fmt.Fprintf(os.Stderr, "glowm: unknown pager mode %q, using more\n", cfg.Pager.Mode)
		cfg.Pager.Mode = pager.ModeMore
	}
	return cfg
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "glowm", "config.json"), nil
}
