package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	PagerModeMore = "more"
	PagerModeVim  = "vim"
)

type Config struct {
	Pager PagerConfig `json:"pager"`
}

type PagerConfig struct {
	Mode string `json:"mode"`
}

func Load() Config {
	cfg := Config{
		Pager: PagerConfig{
			Mode: PagerModeMore,
		},
	}

	path, err := configPath()
	if err != nil {
		return cfg
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg
	}
	if cfg.Pager.Mode == "" {
		cfg.Pager.Mode = PagerModeMore
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
