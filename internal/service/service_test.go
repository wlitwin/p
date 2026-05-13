package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

func setupTestProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := project.Create(root, "test-project", "A test project"); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "test-project")

	list, _ := todo.CreateList(dir, "tasks", "Tasks")
	todo.AddItem(list, "First task", todo.Now, "")
	todo.AddItem(list, "Second task", todo.Backlog, "2026-06-01")
	todo.SaveList(dir, "tasks", list)

	knowledge.Create(dir, "overview", "Overview", []string{"arch"})
	knowledge.Append(dir, "overview", "This is the overview content.", "")

	return dir
}

func TestSetItemState(t *testing.T) {
	dir := setupTestProject(t)

	if err := SetItemState(context.Background(), dir, "tasks", "1", todo.Done); err != nil {
		t.Fatal(err)
	}

	list, _ := todo.LoadList(dir, "tasks")
	if list.Items[0].State != todo.Done {
		t.Errorf("expected done, got %s", list.Items[0].State)
	}
}

func TestSetItemPriority(t *testing.T) {
	dir := setupTestProject(t)

	if err := SetItemPriority(context.Background(), dir, "tasks", "1", todo.Backlog); err != nil {
		t.Fatal(err)
	}

	list, _ := todo.LoadList(dir, "tasks")
	if list.Items[0].Priority != todo.Backlog {
		t.Errorf("expected backlog, got %s", list.Items[0].Priority)
	}
}

func TestSetItemDue(t *testing.T) {
	dir := setupTestProject(t)

	if err := SetItemDue(context.Background(), dir, "tasks", "1", "2026-12-31"); err != nil {
		t.Fatal(err)
	}

	list, _ := todo.LoadList(dir, "tasks")
	if list.Items[0].Due != "2026-12-31" {
		t.Errorf("expected 2026-12-31, got %s", list.Items[0].Due)
	}
}

func TestUpdateItemText(t *testing.T) {
	dir := setupTestProject(t)

	if err := UpdateItemText(context.Background(), dir, "tasks", "1", "Updated text"); err != nil {
		t.Fatal(err)
	}

	list, _ := todo.LoadList(dir, "tasks")
	if list.Items[0].Text != "Updated text" {
		t.Errorf("expected updated text, got %q", list.Items[0].Text)
	}
}

func TestRemoveItem(t *testing.T) {
	dir := setupTestProject(t)

	if err := RemoveItem(context.Background(), dir, "tasks", "1"); err != nil {
		t.Fatal(err)
	}

	list, _ := todo.LoadList(dir, "tasks")
	if len(list.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(list.Items))
	}
}

func TestAddItem(t *testing.T) {
	dir := setupTestProject(t)

	if err := AddItem(context.Background(), dir, "tasks", "Third task", todo.Now, "", ""); err != nil {
		t.Fatal(err)
	}

	list, _ := todo.LoadList(dir, "tasks")
	if len(list.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(list.Items))
	}
}

func TestAddItemWithParent(t *testing.T) {
	dir := setupTestProject(t)

	if err := AddItem(context.Background(), dir, "tasks", "Subtask", todo.Now, "", "1"); err != nil {
		t.Fatal(err)
	}

	list, _ := todo.LoadList(dir, "tasks")
	if len(list.Items) != 2 {
		t.Errorf("expected 2 top-level items, got %d", len(list.Items))
	}
	if len(list.Items[0].Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(list.Items[0].Children))
	}
}

func TestAddItemCreatesNewList(t *testing.T) {
	dir := setupTestProject(t)

	if err := AddItem(context.Background(), dir, "new-list", "New task", todo.Now, "", ""); err != nil {
		t.Fatal(err)
	}

	list, err := todo.LoadList(dir, "new-list")
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(list.Items))
	}
}

func TestMoveItem(t *testing.T) {
	dir := setupTestProject(t)

	if err := MoveItem(context.Background(), dir, "tasks", "1", "done-tasks"); err != nil {
		t.Fatal(err)
	}

	srcList, _ := todo.LoadList(dir, "tasks")
	if len(srcList.Items) != 1 {
		t.Errorf("source should have 1 item, got %d", len(srcList.Items))
	}

	dstList, _ := todo.LoadList(dir, "done-tasks")
	if len(dstList.Items) != 1 {
		t.Errorf("dest should have 1 item, got %d", len(dstList.Items))
	}
	if dstList.Items[0].Text != "First task" {
		t.Errorf("moved item text = %q", dstList.Items[0].Text)
	}
}

func TestRemoveList(t *testing.T) {
	dir := setupTestProject(t)

	if err := RemoveList(context.Background(), dir, "tasks"); err != nil {
		t.Fatal(err)
	}

	_, err := todo.LoadList(dir, "tasks")
	if err == nil {
		t.Error("expected error loading deleted list")
	}
}

func TestRemoveListNotFound(t *testing.T) {
	dir := setupTestProject(t)

	err := RemoveList(context.Background(), dir, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent list")
	}
}

func TestGetProjectListStatuses(t *testing.T) {
	dir := setupTestProject(t)

	statuses, err := GetProjectListStatuses(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 list, got %d", len(statuses))
	}
	if statuses[0].Name != "tasks" {
		t.Errorf("expected 'tasks', got %q", statuses[0].Name)
	}
	if statuses[0].Open != 2 {
		t.Errorf("expected 2 open, got %d", statuses[0].Open)
	}
}

func TestProjectTotals(t *testing.T) {
	dir := setupTestProject(t)

	// Mark one as done to have a mix
	SetItemState(context.Background(), dir, "tasks", "1", todo.Done)

	open, done, blocked := ProjectTotals(context.Background(), dir)
	if open != 1 {
		t.Errorf("expected 1 open, got %d", open)
	}
	if done != 1 {
		t.Errorf("expected 1 done, got %d", done)
	}
	if blocked != 0 {
		t.Errorf("expected 0 blocked, got %d", blocked)
	}
}

func TestSearchProject(t *testing.T) {
	dir := setupTestProject(t)

	// Search for a todo item
	matches := SearchProject(context.Background(), dir, "test-project", "first")
	found := false
	for _, m := range matches {
		if m.Type == "todo" {
			for _, r := range m.TodoResults {
				if r.Item.Text == "First task" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Error("expected to find 'First task' in search results")
	}

	// Search for knowledge content
	matches = SearchProject(context.Background(), dir, "test-project", "overview content")
	found = false
	for _, m := range matches {
		if m.Type == "knowledge" && m.File == "overview" {
			found = true
		}
	}
	if !found {
		t.Error("expected to find 'overview' knowledge doc in search results")
	}

	// Search for non-existent
	matches = SearchProject(context.Background(), dir, "test-project", "nonexistent_xyz")
	if len(matches) != 0 {
		t.Errorf("expected no matches, got %d", len(matches))
	}
}

func TestKnowledgeCRUD(t *testing.T) {
	dir := setupTestProject(t)

	// Create
	if err := KnowledgeCreate(context.Background(), dir, "new-doc", "New Doc", []string{"test"}); err != nil {
		t.Fatal(err)
	}

	content, err := knowledge.Read(dir, "new-doc")
	if err != nil {
		t.Fatal(err)
	}
	if content == "" {
		t.Error("expected content in new doc")
	}

	// Append
	if err := KnowledgeAppend(context.Background(), dir, "new-doc", "Appended content.", ""); err != nil {
		t.Fatal(err)
	}

	content, _ = knowledge.Read(dir, "new-doc")
	if content == "" {
		t.Error("expected non-empty content after append")
	}

	// Rename
	if err := KnowledgeRename(context.Background(), dir, "new-doc", "renamed-doc"); err != nil {
		t.Fatal(err)
	}

	_, err = knowledge.Read(dir, "renamed-doc")
	if err != nil {
		t.Fatalf("expected renamed doc to be readable: %v", err)
	}

	// Delete
	if err := KnowledgeDelete(context.Background(), dir, "renamed-doc"); err != nil {
		t.Fatal(err)
	}

	_, err = knowledge.Read(dir, "renamed-doc")
	if err == nil {
		t.Error("expected error reading deleted doc")
	}
}

func TestSetItemTags(t *testing.T) {
	dir := setupTestProject(t)

	// Add tags
	tags, err := SetItemTags(context.Background(), dir, "tasks", "1", []string{"bug", "frontend"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}

	// Remove tags
	tags, err = SetItemTags(context.Background(), dir, "tasks", "1", []string{"bug"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 {
		t.Errorf("expected 1 tag after removal, got %d", len(tags))
	}
}

func TestKnowledgeReplace(t *testing.T) {
	dir := setupTestProject(t)

	knowledge.Append(dir, "overview", "## Details\n\nOld details content.", "")

	if err := KnowledgeReplace(context.Background(), dir, "overview", "Details", "New details content."); err != nil {
		t.Fatal(err)
	}

	content, _ := knowledge.Read(dir, "overview")
	if !strings.Contains(content, "New details content.") {
		t.Errorf("expected new content, got:\n%s", content)
	}
}

func TestSetListContext(t *testing.T) {
	dir := setupTestProject(t)

	patterns := []string{"architecture/*", "decisions/db-*"}
	if err := SetListContext(context.Background(), dir, "tasks", patterns); err != nil {
		t.Fatal(err)
	}

	list, _ := todo.LoadList(dir, "tasks")
	if len(list.Context) != 2 {
		t.Fatalf("expected 2 context patterns, got %d", len(list.Context))
	}
	if list.Context[0] != "architecture/*" {
		t.Errorf("context[0] = %q, want %q", list.Context[0], "architecture/*")
	}
	if list.Context[1] != "decisions/db-*" {
		t.Errorf("context[1] = %q, want %q", list.Context[1], "decisions/db-*")
	}
}

func TestSetListContextClear(t *testing.T) {
	dir := setupTestProject(t)

	SetListContext(context.Background(), dir, "tasks", []string{"old-pattern"})

	if err := SetListContext(context.Background(), dir, "tasks", nil); err != nil {
		t.Fatal(err)
	}

	list, _ := todo.LoadList(dir, "tasks")
	if list.Context != nil {
		t.Errorf("expected nil context after clear, got %v", list.Context)
	}
}

func TestSetDefaultContext(t *testing.T) {
	dir := setupTestProject(t)

	patterns := []string{"overview", "api-*"}
	if err := SetDefaultContext(context.Background(), dir, patterns); err != nil {
		t.Fatal(err)
	}

	meta, _ := project.LoadMeta(dir)
	if len(meta.DefaultContext) != 2 {
		t.Fatalf("expected 2 default context patterns, got %d", len(meta.DefaultContext))
	}
	if meta.DefaultContext[0] != "overview" {
		t.Errorf("default context[0] = %q, want %q", meta.DefaultContext[0], "overview")
	}
}

func TestSetDefaultContextClear(t *testing.T) {
	dir := setupTestProject(t)

	SetDefaultContext(context.Background(), dir, []string{"old"})

	if err := SetDefaultContext(context.Background(), dir, nil); err != nil {
		t.Fatal(err)
	}

	meta, _ := project.LoadMeta(dir)
	if meta.DefaultContext != nil {
		t.Errorf("expected nil after clear, got %v", meta.DefaultContext)
	}
}

func TestAssetAddListDelete(t *testing.T) {
	dir := setupTestProject(t)

	// Create a temp source file
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmpFile, []byte("test data"), 0o644)

	// Add
	filename, err := AssetAdd(context.Background(), dir, tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if filename != "test.txt" {
		t.Errorf("expected filename 'test.txt', got %q", filename)
	}

	// List
	infos, err := AssetList(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(infos))
	}
	if infos[0].Name != "test.txt" {
		t.Errorf("asset name = %q, want %q", infos[0].Name, "test.txt")
	}

	// Delete
	if err := AssetDelete(context.Background(), dir, "test.txt"); err != nil {
		t.Fatal(err)
	}

	infos, _ = AssetList(context.Background(), dir)
	if len(infos) != 0 {
		t.Errorf("expected 0 assets after delete, got %d", len(infos))
	}
}

func TestSetItemStateInvalidItem(t *testing.T) {
	dir := setupTestProject(t)

	err := SetItemState(context.Background(), dir, "tasks", "99", todo.Done)
	if err == nil {
		t.Error("expected error for invalid item ID")
	}
}

func TestSetItemStateInvalidList(t *testing.T) {
	dir := setupTestProject(t)

	err := SetItemState(context.Background(), dir, "nonexistent", "1", todo.Done)
	if err == nil {
		t.Error("expected error for nonexistent list")
	}
}
