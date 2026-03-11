package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotBeeps/noms/internal/config"
)

func TestConfigLoadDefaults(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Theme.Name != "default" {
		t.Errorf("Theme.Name = %q, want %q", cfg.Theme.Name, "default")
	}
	if cfg.ImageProtocol != "auto" {
		t.Errorf("ImageProtocol = %q, want %q", cfg.ImageProtocol, "auto")
	}
}

func TestConfigLoadFromFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Create the noms config dir and write a TOML file.
	cfgDir := filepath.Join(tmp, "noms")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	tomlContent := `
default_account = "did:plc:testuser"
image_protocol = "kitty"

[theme]
name = "dracula"
`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(tomlContent), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.DefaultAccount != "did:plc:testuser" {
		t.Errorf("DefaultAccount = %q, want %q", cfg.DefaultAccount, "did:plc:testuser")
	}
	if cfg.Theme.Name != "dracula" {
		t.Errorf("Theme.Name = %q, want %q", cfg.Theme.Name, "dracula")
	}
	if cfg.ImageProtocol != "kitty" {
		t.Errorf("ImageProtocol = %q, want %q", cfg.ImageProtocol, "kitty")
	}
}

func TestConfigSaveToFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	cfg := &config.Config{
		DefaultAccount: "did:plc:savetest",
		Theme:          config.ThemeConfig{Name: "nord"},
		ImageProtocol:  "sixel",
	}

	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	cfgPath := filepath.Join(tmp, "noms", "config.toml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "did:plc:savetest") {
		t.Errorf("saved file missing DefaultAccount value; got:\n%s", content)
	}
	if !strings.Contains(content, "nord") {
		t.Errorf("saved file missing theme name; got:\n%s", content)
	}
	if !strings.Contains(content, "sixel") {
		t.Errorf("saved file missing image_protocol; got:\n%s", content)
	}
}
