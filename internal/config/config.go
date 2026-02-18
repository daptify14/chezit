package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Mode string

const (
	ModeWrite    Mode = "write"
	ModeReadOnly Mode = "read_only"
)

var validModes = []Mode{ModeWrite, ModeReadOnly}

func ParseMode(s string) (Mode, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return ModeWrite, nil
	}
	for _, m := range validModes {
		if string(m) == s {
			return m, nil
		}
	}
	return "", fmt.Errorf("invalid mode %q (valid: write, read_only)", s)
}

type Config struct {
	Panel         string   `yaml:"panel"` // "auto" (default), "show", "hide"
	Icons         string   `yaml:"icons"` // "nerdfont" (default), "unicode", "none"
	Mode          Mode     `yaml:"mode"`
	BinaryPath    string   `yaml:"binary_path"`
	CommitPresets []string `yaml:"commit_presets"`
}

func Default() Config {
	return Config{
		Icons: "nerdfont",
		Mode:  ModeWrite,
	}
}

func (c *Config) Normalize() {
	if strings.TrimSpace(string(c.Mode)) == "" {
		c.Mode = ModeWrite
	}

	c.Icons = strings.TrimSpace(strings.ToLower(c.Icons))

	c.BinaryPath = strings.TrimSpace(c.BinaryPath)
	if c.BinaryPath != "" {
		c.BinaryPath = expandPath(c.BinaryPath)
	}
	if len(c.CommitPresets) > 0 {
		c.CommitPresets = normalizeStringList(c.CommitPresets)
	}
}

func (c Config) Validate() error {
	if c.Icons != "" {
		switch c.Icons {
		case "nerdfont", "unicode", "none":
		default:
			return fmt.Errorf("invalid icons %q (valid: nerdfont, unicode, none)", c.Icons)
		}
	}
	if _, err := ParseMode(string(c.Mode)); err != nil {
		return err
	}
	return nil
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "chezit", "config.yaml")
}

func Load() (Config, error) {
	return LoadFrom(DefaultPath())
}

// LoadFrom returns Default() if path doesn't exist.
func LoadFrom(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config: %w", err)
	}

	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, strings.TrimPrefix(path, "~"))
	}
	return path
}

func normalizeStringList(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
