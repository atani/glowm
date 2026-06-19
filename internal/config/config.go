package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the application configuration.
type Config struct {
	Pager   PagerConfig   `json:"pager"`
	Mermaid MermaidConfig `json:"mermaid"`
}

// PagerConfig holds pager-specific settings.
type PagerConfig struct {
	Mode string `json:"mode"`
}

// MermaidConfig holds Mermaid diagram rendering settings.
type MermaidConfig struct {
	// Theme selects the Mermaid color theme: "light"/"default", "dark",
	// "forest", "neutral", or "base". Empty means default.
	Theme string `json:"theme"`
}

// configPathFunc is the function used to determine the config file path.
// It can be overridden in tests.
var configPathFunc = configPath

// Load reads and returns the application config.
// Returns defaults if the config file is missing or unreadable.
// Warns to stderr if the file exists but is invalid JSON.
func Load() Config {
	defaultCfg := Config{
		Pager: PagerConfig{
			Mode: "less",
		},
		Mermaid: MermaidConfig{
			Theme: "auto",
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
		cfg.Pager.Mode = "less"
	}
	if cfg.Mermaid.Theme == "" {
		cfg.Mermaid.Theme = "auto"
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
