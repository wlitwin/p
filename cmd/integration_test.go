package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/walter/p/internal/config"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

func setupIntegrationTest(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	cfg = config.Config{
		ProjectRoot:     root,
		ClaudePath:      "claude",
		ClaudeModel:     "claude-opus-4-6",
		DefaultPriority: "now",
	}

	return root
}

func TestIntegrationProjectLifecycle(t *testing.T) {
	root := setupIntegrationTest(t)

	// Create project
	if err := project.Create(root, "test-proj", "Integration test project"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Verify directory structure
	for _, dir := range []string{"knowledge", "todos", "assets", ".p"} {
		path := filepath.Join(root, "test-proj", dir)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing directory %s: %v", dir, err)
		}
	}

	// Load and verify metadata
	meta, err := project.LoadMeta(filepath.Join(root, "test-proj"))
	if err != nil {
		t.Fatalf("LoadMeta: %v", err)
	}
	if meta.Name != "test-proj" {
		t.Errorf("name = %q", meta.Name)
	}
	if meta.Description != "Integration test project" {
		t.Errorf("description = %q", meta.Description)
	}
	if meta.Archived {
		t.Error("should not be archived")
	}

	// Archive
	meta.Archived = true
	if err := project.SaveMeta(filepath.Join(root, "test-proj"), meta); err != nil {
		t.Fatalf("SaveMeta: %v", err)
	}

	// List should exclude archived
	projects, _ := project.List(root, false)
	if len(projects) != 0 {
		t.Error("archived project should be hidden")
	}

	// List with --all should include
	projects, _ = project.List(root, true)
	if len(projects) != 1 {
		t.Error("archived project should appear with --all")
	}
}

func TestIntegrationTodoCRUD(t *testing.T) {
	root := setupIntegrationTest(t)
	project.Create(root, "proj", "")
	dir := filepath.Join(root, "proj")

	// Create list and add items
	list, err := todo.CreateList(dir, "tasks", "Tasks")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}

	todo.AddItem(list, "First", todo.Now, "")
	todo.AddItem(list, "Second", todo.Backlog, "2026-06-01")
	todo.AddItem(list, "Third", todo.Now, "")
	todo.SaveList(dir, "tasks", list)

	// Reload and verify
	list, _ = todo.LoadList(dir, "tasks")
	if len(list.Items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(list.Items))
	}

	// State changes
	todo.SetState(list.Items[0], todo.Done)
	todo.SetState(list.Items[2], todo.Blocked)
	todo.SaveList(dir, "tasks", list)

	list, _ = todo.LoadList(dir, "tasks")
	if list.Items[0].State != todo.Done {
		t.Error("item 1 should be done")
	}
	if list.Items[2].State != todo.Blocked {
		t.Error("item 3 should be blocked")
	}

	// Remove
	todo.RemoveItem(list, "2")
	todo.SaveList(dir, "tasks", list)

	list, _ = todo.LoadList(dir, "tasks")
	if len(list.Items) != 2 {
		t.Errorf("expected 2 items after remove, got %d", len(list.Items))
	}

	// Nested items
	child := &todo.Item{Text: "Sub-task", State: todo.Open, Priority: todo.Now}
	list.Items[0].Children = append(list.Items[0].Children, child)
	todo.SaveList(dir, "tasks", list)

	list, _ = todo.LoadList(dir, "tasks")
	if len(list.Items[0].Children) != 1 {
		t.Error("expected 1 child")
	}

	// Resolve nested item
	item, err := todo.ResolveItem(list, "1.1")
	if err != nil {
		t.Fatalf("ResolveItem 1.1: %v", err)
	}
	if item.Text != "Sub-task" {
		t.Errorf("child text = %q", item.Text)
	}
}

func TestIntegrationKnowledgeCRUD(t *testing.T) {
	root := setupIntegrationTest(t)
	project.Create(root, "proj", "")
	dir := filepath.Join(root, "proj")

	// Create
	knowledge.Create(dir, "arch", "Architecture", []string{"architecture", "design"})

	// Read
	content, err := knowledge.Read(dir, "arch")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "Architecture") {
		t.Error("missing title")
	}

	// Append
	knowledge.Append(dir, "arch", "## Database\n\nWe use PostgreSQL.", "")
	content, _ = knowledge.Read(dir, "arch")
	if !strings.Contains(content, "PostgreSQL") {
		t.Error("append failed")
	}

	// Append to section
	knowledge.Append(dir, "arch", "With read replicas.", "Database")
	content, _ = knowledge.Read(dir, "arch")
	if !strings.Contains(content, "read replicas") {
		t.Error("section append failed")
	}

	// Replace section
	knowledge.ReplaceSection(dir, "arch", "Database", "We migrated to CockroachDB.")
	content, _ = knowledge.Read(dir, "arch")
	if strings.Contains(content, "PostgreSQL") {
		t.Error("old content still present after replace")
	}
	if !strings.Contains(content, "CockroachDB") {
		t.Error("new content missing after replace")
	}

	// Extract tags
	tags := knowledge.ExtractTags(content)
	if len(tags) != 2 || tags[0] != "architecture" {
		t.Errorf("tags = %v", tags)
	}

	// List files
	files, _ := knowledge.ListFiles(dir)
	if len(files) != 1 || files[0] != "arch" {
		t.Errorf("files = %v", files)
	}

	// Rename
	knowledge.Rename(dir, "arch", "architecture")
	files, _ = knowledge.ListFiles(dir)
	if len(files) != 1 || files[0] != "architecture" {
		t.Errorf("after rename, files = %v", files)
	}

	// Delete
	knowledge.Delete(dir, "architecture")
	files, _ = knowledge.ListFiles(dir)
	if len(files) != 0 {
		t.Error("file should be deleted")
	}
}

func TestIntegrationTodoParseRender(t *testing.T) {
	root := setupIntegrationTest(t)
	project.Create(root, "proj", "")
	dir := filepath.Join(root, "proj")

	list, _ := todo.CreateList(dir, "work", "Work Items")
	item1 := todo.AddItem(list, "Build feature", todo.Now, "2026-07-01")
	item1.Tags = []string{"frontend", "urgent"}
	todo.AddItem(list, "Write docs", todo.Backlog, "")
	todo.SaveList(dir, "work", list)

	// Reload and verify round-trip
	list, _ = todo.LoadList(dir, "work")
	if list.Items[0].Due != "2026-07-01" {
		t.Errorf("due = %q", list.Items[0].Due)
	}
	if len(list.Items[0].Tags) != 2 {
		t.Errorf("tags = %v", list.Items[0].Tags)
	}
	if list.Items[1].Priority != todo.Backlog {
		t.Errorf("priority = %q", list.Items[1].Priority)
	}
}
