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
chezmoi_config_path: ~/.config/chezmoi/work.toml
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
	if cfg.ChezmoiConfig == "~/.config/chezmoi/work.toml" || cfg.ChezmoiConfig == "" {
		t.Fatalf("expected expanded chezmoi config path, got %q", cfg.ChezmoiConfig)
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

func TestLoadFromParsesDiffBuiltin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`
diff_builtin: true
`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if !cfg.DiffBuiltin {
		t.Fatalf("expected diff_builtin true, got false")
	}
}

func TestLoadFromDiffBuiltinDefaultsFalse(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.DiffBuiltin {
		t.Fatalf("expected diff_builtin default false, got true")
	}
}

func TestLoadFromAutoFetchDefaultsTrue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`
icons: none
`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if !cfg.AutoFetchEnabled() {
		t.Fatal("expected auto_fetch to default to true when the key is absent")
	}
}

func TestLoadFromParsesAutoFetchFalse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`
auto_fetch: false
`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.AutoFetchEnabled() {
		t.Fatal("expected auto_fetch false, got true")
	}
}

func TestLoadFromParsesAutoFetchTrue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`
auto_fetch: true
`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if !cfg.AutoFetchEnabled() {
		t.Fatal("expected auto_fetch true, got false")
	}
}

func TestDefaultAutoFetchTrue(t *testing.T) {
	if !Default().AutoFetchEnabled() {
		t.Fatal("expected Default().AutoFetchEnabled() to be true")
	}
	if (Config{}).AutoFetchEnabled() != true {
		t.Fatal("expected zero-value Config to report auto_fetch enabled")
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

func TestLoadFromParsesEditor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(`
editor: "  code --wait "
`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Editor != "code --wait" {
		t.Fatalf("expected editor %q, got %q", "code --wait", cfg.Editor)
	}
}

func TestLoadFromEditorDefaultsEmpty(t *testing.T) {
	cfg, err := LoadFrom(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Editor != "" {
		t.Fatalf("expected empty editor default, got %q", cfg.Editor)
	}
}
