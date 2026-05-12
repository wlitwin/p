// Package config handles loading and saving the global application configuration
// from the XDG config directory.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ColorConfig holds individual color overrides. Values can be ANSI 256 codes
// ("242"), hex codes ("#7C6F64"), or empty strings (use preset default).
type ColorConfig struct {
	Dim         string `yaml:"dim,omitempty"`
	Done        string `yaml:"done,omitempty"`
	Help        string `yaml:"help,omitempty"`
	Accent      string `yaml:"accent,omitempty"`
	Open        string `yaml:"open,omitempty"`
	Green       string `yaml:"green,omitempty"`
	Yellow      string `yaml:"yellow,omitempty"`
	Red         string `yaml:"red,omitempty"`
	Cyan        string `yaml:"cyan,omitempty"`
	Blocked     string `yaml:"blocked,omitempty"`
	PriorityNow string `yaml:"priority_now,omitempty"`
	Error       string `yaml:"error,omitempty"`
}

// Config holds the global application configuration, persisted as YAML in
// the XDG config directory (~/.config/p/config.yaml).
type Config struct {
	ProjectRoot     string      `yaml:"project_root"`
	ClaudePath      string      `yaml:"claude_path"`
	ClaudeModel     string      `yaml:"claude_model"`
	DefaultPriority string      `yaml:"default_priority"`
	Theme           string      `yaml:"theme,omitempty"`
	Colors          ColorConfig `yaml:"colors,omitempty"`
	GlamourTheme    string      `yaml:"glamour_theme,omitempty"`
}

// DefaultConfig returns a Config with sensible defaults for all fields.
func DefaultConfig() Config {
	return Config{
		ClaudePath:      "claude",
		ClaudeModel:     "claude-opus-4-6",
		DefaultPriority: "now",
	}
}

func configDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "p"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "p"), nil
}

// ConfigPath returns the absolute path to the config file, respecting
// XDG_CONFIG_HOME if set.
func ConfigPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads the config file from disk and returns it merged over defaults.
// Returns defaults without error if the config file does not exist.
func Load() (Config, error) {
	cfg := DefaultConfig()
	path, err := ConfigPath()
	if err != nil {
		return cfg, err
	}

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
	return cfg, nil
}

// Save writes the config to disk, creating the config directory if needed.
func Save(cfg Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}
