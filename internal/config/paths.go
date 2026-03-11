package config

import (
	"os"
	"path/filepath"
)

// ConfigDir returns the XDG-compliant config directory for noms.
// Defaults to ~/.config/noms/ if XDG_CONFIG_HOME is not set.
func ConfigDir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "noms")
}

// DataDir returns the XDG-compliant data directory for noms.
// Defaults to ~/.local/share/noms/ if XDG_DATA_HOME is not set.
func DataDir() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "noms")
}

// CacheDir returns the XDG-compliant cache directory for noms.
// Defaults to ~/.cache/noms/ if XDG_CACHE_HOME is not set.
func CacheDir() string {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "noms")
}

// EnsureDirs creates all three XDG directories (config, data, cache) if they
// don't already exist.
func EnsureDirs() error {
	for _, dir := range []string{ConfigDir(), DataDir(), CacheDir()} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}
