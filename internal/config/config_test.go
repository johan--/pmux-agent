package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Nonexistent(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/config.toml")
	if err != nil {
		t.Fatalf("LoadConfig() returned error for nonexistent file: %v", err)
	}
	if cfg.Name != "" {
		t.Errorf("expected empty name, got %q", cfg.Name)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	want := Config{Name: "my-workstation"}
	if err := SaveConfig(path, want); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	got, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if got.Name != want.Name {
		t.Errorf("Name = %q, want %q", got.Name, want.Name)
	}
}

func TestSaveConfig_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	if err := SaveConfig(path, Config{Name: "test"}); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}

func TestDefaultHostName(t *testing.T) {
	name := DefaultHostName()
	if name == "" {
		t.Error("DefaultHostName() returned empty string")
	}
}
