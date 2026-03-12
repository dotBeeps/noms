package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ThemeConfig holds theme-related configuration.
type ThemeConfig struct {
	Name string `toml:"name"`
}

// Config is the top-level configuration structure for noms.
type Config struct {
	DefaultAccount string      `toml:"default_account"`
	Theme          ThemeConfig `toml:"theme"`
	ImageProtocol  string      `toml:"image_protocol"`
}

// defaultConfig returns a Config populated with sensible defaults.
func defaultConfig() *Config {
	return &Config{
		DefaultAccount: "",
		Theme:          ThemeConfig{Name: "default"},
		ImageProtocol:  "auto",
	}
}

// Load reads the config file from ConfigDir()/config.toml.
// If the file does not exist, a default Config is returned and written to disk.
func Load() (*Config, error) {
	cfgPath := filepath.Join(ConfigDir(), "config.toml")

	cfg := defaultConfig()

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Write defaults so the file exists for future edits.
			if saveErr := Save(cfg); saveErr != nil {
				return nil, saveErr
			}
			return cfg, nil
		}
		return nil, err
	}

	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes cfg to ConfigDir()/config.toml, creating the directory if needed.
func Save(cfg *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	cfgPath := filepath.Join(dir, "config.toml")

	// Write to a temp file then rename for atomicity.
	tmp, err := os.CreateTemp(dir, "config-*.toml.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if err := toml.NewEncoder(tmp).Encode(cfg); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, cfgPath)
}
