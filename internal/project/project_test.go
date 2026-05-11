package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreate(t *testing.T) {
	root := t.TempDir()
	name := "my-project"

	if err := Create(root, name, ""); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	for _, sub := range []string{"knowledge", "todos", "assets", ".p"} {
		path := filepath.Join(root, name, sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected directory %s to exist: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", sub)
		}
	}

	// config.yaml should exist
	configPath := filepath.Join(root, name, ".p", "config.yaml")
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config.yaml to exist: %v", err)
	}
}

func TestCreateDuplicate(t *testing.T) {
	root := t.TempDir()
	name := "dup-project"

	if err := Create(root, name, "first"); err != nil {
		t.Fatalf("first Create failed: %v", err)
	}

	err := Create(root, name, "second")
	if err == nil {
		t.Fatal("expected error when creating duplicate project, got nil")
	}
}

func TestLoadMeta(t *testing.T) {
	root := t.TempDir()
	name := "load-test"
	desc := "A test project"
	before := time.Now().UTC().Add(-time.Second)

	if err := Create(root, name, desc); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	after := time.Now().UTC().Add(time.Second)

	meta, err := LoadMeta(filepath.Join(root, name))
	if err != nil {
		t.Fatalf("LoadMeta failed: %v", err)
	}

	if meta.Name != name {
		t.Errorf("Name = %q, want %q", meta.Name, name)
	}
	if meta.Description != desc {
		t.Errorf("Description = %q, want %q", meta.Description, desc)
	}
	if meta.Created.Before(before) || meta.Created.After(after) {
		t.Errorf("Created = %v, expected between %v and %v", meta.Created, before, after)
	}
	if meta.Archived {
		t.Error("expected Archived to be false for new project")
	}
}

func TestSaveMeta(t *testing.T) {
	root := t.TempDir()
	name := "save-test"

	if err := Create(root, name, "original"); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	projectDir := filepath.Join(root, name)
	meta, err := LoadMeta(projectDir)
	if err != nil {
		t.Fatalf("LoadMeta failed: %v", err)
	}

	meta.Description = "updated description"
	meta.Archived = true
	meta.CodeDir = "/some/code/path"

	if err := SaveMeta(projectDir, meta); err != nil {
		t.Fatalf("SaveMeta failed: %v", err)
	}

	reloaded, err := LoadMeta(projectDir)
	if err != nil {
		t.Fatalf("LoadMeta after save failed: %v", err)
	}

	if reloaded.Description != "updated description" {
		t.Errorf("Description = %q, want %q", reloaded.Description, "updated description")
	}
	if !reloaded.Archived {
		t.Error("expected Archived to be true after save")
	}
	if reloaded.CodeDir != "/some/code/path" {
		t.Errorf("CodeDir = %q, want %q", reloaded.CodeDir, "/some/code/path")
	}
	if reloaded.Name != name {
		t.Errorf("Name = %q, want %q", reloaded.Name, name)
	}
}

func TestList(t *testing.T) {
	root := t.TempDir()
	names := []string{"alpha", "bravo", "charlie"}

	for _, n := range names {
		if err := Create(root, n, ""); err != nil {
			t.Fatalf("Create(%q) failed: %v", n, err)
		}
	}

	projects, err := List(root, false)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(projects) != len(names) {
		t.Fatalf("List returned %d projects, want %d", len(projects), len(names))
	}

	found := make(map[string]bool)
	for _, p := range projects {
		found[p.Name] = true
	}
	for _, n := range names {
		if !found[n] {
			t.Errorf("project %q not found in List results", n)
		}
	}
}

func TestListExcludesArchived(t *testing.T) {
	root := t.TempDir()

	if err := Create(root, "active", ""); err != nil {
		t.Fatalf("Create(active) failed: %v", err)
	}
	if err := Create(root, "archived", ""); err != nil {
		t.Fatalf("Create(archived) failed: %v", err)
	}

	// Archive the second project
	archivedDir := filepath.Join(root, "archived")
	meta, err := LoadMeta(archivedDir)
	if err != nil {
		t.Fatalf("LoadMeta failed: %v", err)
	}
	meta.Archived = true
	if err := SaveMeta(archivedDir, meta); err != nil {
		t.Fatalf("SaveMeta failed: %v", err)
	}

	projects, err := List(root, false)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("List(includeArchived=false) returned %d projects, want 1", len(projects))
	}
	if projects[0].Name != "active" {
		t.Errorf("expected remaining project to be %q, got %q", "active", projects[0].Name)
	}
}

func TestListIncludesArchived(t *testing.T) {
	root := t.TempDir()

	if err := Create(root, "active", ""); err != nil {
		t.Fatalf("Create(active) failed: %v", err)
	}
	if err := Create(root, "archived", ""); err != nil {
		t.Fatalf("Create(archived) failed: %v", err)
	}

	// Archive the second project
	archivedDir := filepath.Join(root, "archived")
	meta, err := LoadMeta(archivedDir)
	if err != nil {
		t.Fatalf("LoadMeta failed: %v", err)
	}
	meta.Archived = true
	if err := SaveMeta(archivedDir, meta); err != nil {
		t.Fatalf("SaveMeta failed: %v", err)
	}

	projects, err := List(root, true)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(projects) != 2 {
		t.Fatalf("List(includeArchived=true) returned %d projects, want 2", len(projects))
	}

	found := make(map[string]bool)
	for _, p := range projects {
		found[p.Name] = true
	}
	if !found["active"] {
		t.Error("expected 'active' in List results")
	}
	if !found["archived"] {
		t.Error("expected 'archived' in List results")
	}
}

func TestResolveValid(t *testing.T) {
	root := t.TempDir()
	name := "resolvable"

	if err := Create(root, name, ""); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	dir, err := Resolve(root, name)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	expected := filepath.Join(root, name)
	if dir != expected {
		t.Errorf("Resolve = %q, want %q", dir, expected)
	}
}

func TestResolveMissing(t *testing.T) {
	root := t.TempDir()

	_, err := Resolve(root, "nonexistent")
	if err == nil {
		t.Fatal("expected error when resolving nonexistent project, got nil")
	}
}

func TestListEmpty(t *testing.T) {
	root := t.TempDir()

	projects, err := List(root, false)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("List on empty root returned %d projects, want 0", len(projects))
	}
}

func TestCreateWithDescription(t *testing.T) {
	root := t.TempDir()
	name := "described"
	desc := "This project has a description"

	if err := Create(root, name, desc); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	meta, err := LoadMeta(filepath.Join(root, name))
	if err != nil {
		t.Fatalf("LoadMeta failed: %v", err)
	}

	if meta.Description != desc {
		t.Errorf("Description = %q, want %q", meta.Description, desc)
	}
}

func TestDefaultContextSaveLoad(t *testing.T) {
	root := t.TempDir()
	name := "ctx-project"

	if err := Create(root, name, ""); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	projectDir := filepath.Join(root, name)
	meta, err := LoadMeta(projectDir)
	if err != nil {
		t.Fatalf("LoadMeta failed: %v", err)
	}

	// Initially nil
	if meta.DefaultContext != nil {
		t.Errorf("DefaultContext should be nil initially, got %v", meta.DefaultContext)
	}

	// Set and save
	meta.DefaultContext = []string{"overview", "architecture/*"}
	if err := SaveMeta(projectDir, meta); err != nil {
		t.Fatalf("SaveMeta failed: %v", err)
	}

	reloaded, err := LoadMeta(projectDir)
	if err != nil {
		t.Fatalf("LoadMeta after save failed: %v", err)
	}

	if len(reloaded.DefaultContext) != 2 {
		t.Fatalf("DefaultContext len = %d, want 2", len(reloaded.DefaultContext))
	}
	if reloaded.DefaultContext[0] != "overview" {
		t.Errorf("DefaultContext[0] = %q, want %q", reloaded.DefaultContext[0], "overview")
	}
	if reloaded.DefaultContext[1] != "architecture/*" {
		t.Errorf("DefaultContext[1] = %q, want %q", reloaded.DefaultContext[1], "architecture/*")
	}
}

func TestDefaultContextOmittedWhenEmpty(t *testing.T) {
	root := t.TempDir()
	name := "no-ctx"

	if err := Create(root, name, ""); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	projectDir := filepath.Join(root, name)

	// Read raw config to verify default_context is not present
	data, err := os.ReadFile(filepath.Join(projectDir, ".p", "config.yaml"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if strings.Contains(string(data), "default_context") {
		t.Errorf("config.yaml should not contain default_context when nil, got:\n%s", string(data))
	}
}

func TestDefaultContextClearRoundTrip(t *testing.T) {
	root := t.TempDir()
	name := "clear-ctx"

	if err := Create(root, name, ""); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	projectDir := filepath.Join(root, name)
	meta, err := LoadMeta(projectDir)
	if err != nil {
		t.Fatalf("LoadMeta failed: %v", err)
	}

	// Set context
	meta.DefaultContext = []string{"docs/*"}
	if err := SaveMeta(projectDir, meta); err != nil {
		t.Fatalf("SaveMeta failed: %v", err)
	}

	// Clear context
	meta.DefaultContext = nil
	if err := SaveMeta(projectDir, meta); err != nil {
		t.Fatalf("SaveMeta (clear) failed: %v", err)
	}

	reloaded, err := LoadMeta(projectDir)
	if err != nil {
		t.Fatalf("LoadMeta after clear failed: %v", err)
	}

	if reloaded.DefaultContext != nil {
		t.Errorf("DefaultContext should be nil after clear, got %v", reloaded.DefaultContext)
	}
}
