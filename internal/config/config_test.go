package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("LoadFrom returned error: %v", err)
	}
	if cfg.Mode != ModeWrite {
		t.Fatalf("expected default mode write, got %q", cfg.Mode)
	}
	if cfg.Icons != "nerdfont" {
		t.Fatalf("expected default icons nerdfont, got %q", cfg.Icons)
	}
}

func TestLoadFromParsesFlatConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`
mode: read_only
binary_path: ~/bin/chezmoi-edge
commit_presets:
  - "from chezmoi"
`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Mode != ModeReadOnly {
		t.Fatalf("expected mode read_only, got %q", cfg.Mode)
	}
	if cfg.BinaryPath == "~/bin/chezmoi-edge" || cfg.BinaryPath == "" {
		t.Fatalf("expected expanded binary path, got %q", cfg.BinaryPath)
	}
	if len(cfg.CommitPresets) != 1 || cfg.CommitPresets[0] != "from chezmoi" {
		t.Fatalf("unexpected commit presets: %#v", cfg.CommitPresets)
	}
}

func TestLoadFromInvalidMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`
mode: bad
`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := LoadFrom(path); err == nil {
		t.Fatalf("expected error for invalid mode")
	}
}

func TestLoadFromParsesTopLevelCommitPresets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`
commit_presets:
  - "top-level preset"
`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if len(cfg.CommitPresets) != 1 || cfg.CommitPresets[0] != "top-level preset" {
		t.Fatalf("expected top-level commit presets, got %#v", cfg.CommitPresets)
	}
}

func TestNormalizeIconsTrimsAndLowercases(t *testing.T) {
	cfg := Config{
		Icons: "  NerdFont  ",
	}

	cfg.Normalize()

	if cfg.Icons != "nerdfont" {
		t.Fatalf("expected normalized icons nerdfont, got %q", cfg.Icons)
	}
}
