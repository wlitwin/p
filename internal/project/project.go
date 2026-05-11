package project

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type ProjectMeta struct {
	Name           string    `yaml:"name"`
	Created        time.Time `yaml:"created"`
	Archived       bool      `yaml:"archived"`
	Description    string    `yaml:"description,omitempty"`
	CodeDir        string    `yaml:"code_dir,omitempty"`
	DefaultContext []string  `yaml:"default_context,omitempty"`
}

func Create(root, name, description string) error {
	dir := filepath.Join(root, name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("project %q already exists", name)
	}

	for _, sub := range []string{"knowledge", "todos", "assets", ".p"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}
	}

	meta := ProjectMeta{
		Name:        name,
		Created:     time.Now().UTC(),
		Description: description,
	}
	data, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, ".p", "config.yaml"), data, 0o644)
}

func LoadMeta(projectDir string) (ProjectMeta, error) {
	var meta ProjectMeta
	data, err := os.ReadFile(filepath.Join(projectDir, ".p", "config.yaml"))
	if err != nil {
		return meta, err
	}
	return meta, yaml.Unmarshal(data, &meta)
}

func SaveMeta(projectDir string, meta ProjectMeta) error {
	data, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(projectDir, ".p", "config.yaml"), data, 0o644)
}

func List(root string, includeArchived bool) ([]ProjectMeta, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("reading project root: %w", err)
	}

	var projects []ProjectMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		configPath := filepath.Join(root, e.Name(), ".p", "config.yaml")
		if _, err := os.Stat(configPath); err != nil {
			continue
		}
		meta, err := LoadMeta(filepath.Join(root, e.Name()))
		if err != nil {
			continue
		}
		if !includeArchived && meta.Archived {
			continue
		}
		projects = append(projects, meta)
	}
	return projects, nil
}

func Resolve(root, name string) (string, error) {
	dir := filepath.Join(root, name)
	configPath := filepath.Join(dir, ".p", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		return "", fmt.Errorf("project %q not found — run `p new %s` to create it, or `p list` to see existing projects", name, name)
	}
	return dir, nil
}
