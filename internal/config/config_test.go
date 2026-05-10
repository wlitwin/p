package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ClaudePath != "claude" {
		t.Errorf("ClaudePath = %q, want %q", cfg.ClaudePath, "claude")
	}
	if cfg.ClaudeModel != "claude-opus-4-6" {
		t.Errorf("ClaudeModel = %q, want %q", cfg.ClaudeModel, "claude-opus-4-6")
	}
	if cfg.DefaultPriority != "now" {
		t.Errorf("DefaultPriority = %q, want %q", cfg.DefaultPriority, "now")
	}
	if cfg.ProjectRoot != "" {
		t.Errorf("ProjectRoot = %q, want empty string", cfg.ProjectRoot)
	}
}

func TestConfigPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error: %v", err)
	}

	want := filepath.Join(tmp, "p", "config.yaml")
	if path != want {
		t.Errorf("ConfigPath() = %q, want %q", path, want)
	}
}

func TestConfigPathFallback(t *testing.T) {
	// When XDG_CONFIG_HOME is unset, configDir should use ~/.config/p
	t.Setenv("XDG_CONFIG_HOME", "")

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error: %v", err)
	}

	want := filepath.Join(home, ".config", "p", "config.yaml")
	if path != want {
		t.Errorf("ConfigPath() = %q, want %q", path, want)
	}
}

func TestLoadNoFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	def := DefaultConfig()
	if cfg != def {
		t.Errorf("Load() = %+v, want defaults %+v", cfg, def)
	}
}

func TestLoadExistingFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "p")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	content := []byte("project_root: /my/projects\nclaude_path: /usr/bin/claude\nclaude_model: sonnet\ndefault_priority: later\n")
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.ProjectRoot != "/my/projects" {
		t.Errorf("ProjectRoot = %q, want %q", cfg.ProjectRoot, "/my/projects")
	}
	if cfg.ClaudePath != "/usr/bin/claude" {
		t.Errorf("ClaudePath = %q, want %q", cfg.ClaudePath, "/usr/bin/claude")
	}
	if cfg.ClaudeModel != "sonnet" {
		t.Errorf("ClaudeModel = %q, want %q", cfg.ClaudeModel, "sonnet")
	}
	if cfg.DefaultPriority != "later" {
		t.Errorf("DefaultPriority = %q, want %q", cfg.DefaultPriority, "later")
	}
}

func TestLoadPartialFile(t *testing.T) {
	// A file that sets only some fields should leave the rest as defaults.
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "p")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	content := []byte("project_root: /data/projects\n")
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.ProjectRoot != "/data/projects" {
		t.Errorf("ProjectRoot = %q, want %q", cfg.ProjectRoot, "/data/projects")
	}
	// Defaults should be preserved for unset fields.
	def := DefaultConfig()
	if cfg.ClaudePath != def.ClaudePath {
		t.Errorf("ClaudePath = %q, want default %q", cfg.ClaudePath, def.ClaudePath)
	}
	if cfg.ClaudeModel != def.ClaudeModel {
		t.Errorf("ClaudeModel = %q, want default %q", cfg.ClaudeModel, def.ClaudeModel)
	}
	if cfg.DefaultPriority != def.DefaultPriority {
		t.Errorf("DefaultPriority = %q, want default %q", cfg.DefaultPriority, def.DefaultPriority)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	original := Config{
		ProjectRoot:     "/home/user/projects",
		ClaudePath:      "/opt/claude",
		ClaudeModel:     "haiku",
		DefaultPriority: "backlog",
	}

	if err := Save(original); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded != original {
		t.Errorf("round-trip mismatch:\n  saved:  %+v\n  loaded: %+v", original, loaded)
	}
}

func TestLoadMalformedYAML(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "p")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	content := []byte(":\n  bad:\n  - :\n  {{invalid yaml}}\n")
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should return error for malformed YAML, got nil")
	}
}

func TestSaveCreatesDirectories(t *testing.T) {
	tmp := t.TempDir()
	// Point XDG_CONFIG_HOME to a subdirectory that does not yet exist.
	nested := filepath.Join(tmp, "deep", "nested")
	t.Setenv("XDG_CONFIG_HOME", nested)

	cfg := DefaultConfig()
	cfg.ProjectRoot = "/somewhere"

	if err := Save(cfg); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify the file was actually created.
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created at %s: %v", path, err)
	}

	// Verify we can load what was saved.
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.ProjectRoot != "/somewhere" {
		t.Errorf("ProjectRoot = %q, want %q", loaded.ProjectRoot, "/somewhere")
	}
}

func TestSaveFilePermissions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	if err := Save(DefaultConfig()); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s): %v", path, err)
	}

	perm := info.Mode().Perm()
	if perm != 0o644 {
		t.Errorf("file permissions = %o, want 644", perm)
	}
}
