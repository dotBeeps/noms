package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotBeeps/noms/internal/config"
)

func TestXDGConfigPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	got := config.ConfigDir()
	want := filepath.Join(home, ".config", "noms")
	if got != want {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}

func TestXDGDataPath(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	got := config.DataDir()
	want := filepath.Join(home, ".local", "share", "noms")
	if got != want {
		t.Errorf("DataDir() = %q, want %q", got, want)
	}
}

func TestXDGCachePath(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	got := config.CacheDir()
	want := filepath.Join(home, ".cache", "noms")
	if got != want {
		t.Errorf("CacheDir() = %q, want %q", got, want)
	}
}

func TestXDGCustomOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "cfg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))

	if got := config.ConfigDir(); !strings.HasPrefix(got, filepath.Join(tmp, "cfg")) {
		t.Errorf("ConfigDir() = %q, want prefix %q", got, filepath.Join(tmp, "cfg"))
	}
	if got := config.DataDir(); !strings.HasPrefix(got, filepath.Join(tmp, "data")) {
		t.Errorf("DataDir() = %q, want prefix %q", got, filepath.Join(tmp, "data"))
	}
	if got := config.CacheDir(); !strings.HasPrefix(got, filepath.Join(tmp, "cache")) {
		t.Errorf("CacheDir() = %q, want prefix %q", got, filepath.Join(tmp, "cache"))
	}
}

func TestAutoCreateDirectories(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "cfg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))

	if err := config.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error: %v", err)
	}

	for _, dir := range []string{config.ConfigDir(), config.DataDir(), config.CacheDir()} {
		if info, err := os.Stat(dir); err != nil {
			t.Errorf("directory %q not created: %v", dir, err)
		} else if !info.IsDir() {
			t.Errorf("%q is not a directory", dir)
		}
	}
}
