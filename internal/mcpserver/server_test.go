package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

func setupTestRoot(t *testing.T) string {
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

	return root
}

func callTool(t *testing.T, ctx *serverCtx, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) string {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		for _, c := range result.Content {
			if tc, ok := c.(mcp.TextContent); ok {
				t.Fatalf("tool error: %s", tc.Text)
			}
		}
	}
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func TestProjectList(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	text := callTool(t, ctx, ctx.handleProjectList, nil)
	if !strings.Contains(text, "test-project") {
		t.Errorf("expected project name in output: %s", text)
	}
}

func TestTodoList(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	// List all todo lists
	text := callTool(t, ctx, ctx.handleTodoList, map[string]any{
		"project": "test-project",
	})
	if !strings.Contains(text, "tasks") {
		t.Errorf("expected 'tasks' in list: %s", text)
	}

	// List items in a specific list
	text = callTool(t, ctx, ctx.handleTodoList, map[string]any{
		"project": "test-project",
		"list":    "tasks",
	})
	if !strings.Contains(text, "First task") {
		t.Errorf("expected 'First task' in items: %s", text)
	}
}

func TestTodoAdd(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	text := callTool(t, ctx, ctx.handleTodoAdd, map[string]any{
		"project":  "test-project",
		"list":     "tasks",
		"text":     "Third task",
		"priority": "backlog",
	})
	if !strings.Contains(text, "Third task") {
		t.Errorf("expected confirmation: %s", text)
	}

	// Verify it was added
	dir := filepath.Join(root, "test-project")
	list, _ := todo.LoadList(dir, "tasks")
	if len(list.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(list.Items))
	}
}

func TestTodoState(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	callTool(t, ctx, ctx.handleTodoState, map[string]any{
		"project": "test-project",
		"list":    "tasks",
		"item_id": "1",
		"state":   "done",
	})

	dir := filepath.Join(root, "test-project")
	list, _ := todo.LoadList(dir, "tasks")
	if list.Items[0].State != todo.Done {
		t.Errorf("expected done state, got %s", list.Items[0].State)
	}
}

func TestTodoRemove(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	callTool(t, ctx, ctx.handleTodoRemove, map[string]any{
		"project": "test-project",
		"list":    "tasks",
		"item_id": "1",
	})

	dir := filepath.Join(root, "test-project")
	list, _ := todo.LoadList(dir, "tasks")
	if len(list.Items) != 1 {
		t.Errorf("expected 1 item after remove, got %d", len(list.Items))
	}
}

func TestKnowledgeRead(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	// List docs
	text := callTool(t, ctx, ctx.handleKnowledgeRead, map[string]any{
		"project": "test-project",
	})
	if !strings.Contains(text, "overview") {
		t.Errorf("expected 'overview' in list: %s", text)
	}

	// Read specific doc
	text = callTool(t, ctx, ctx.handleKnowledgeRead, map[string]any{
		"project":  "test-project",
		"filename": "overview",
	})
	if !strings.Contains(text, "overview content") {
		t.Errorf("expected content in doc: %s", text)
	}
}

func TestKnowledgeCreate(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	callTool(t, ctx, ctx.handleKnowledgeCreate, map[string]any{
		"project":  "test-project",
		"filename": "decisions",
		"title":    "Decision Log",
		"tags":     "decisions,arch",
	})

	dir := filepath.Join(root, "test-project")
	content, err := knowledge.Read(dir, "decisions")
	if err != nil {
		t.Fatalf("reading created doc: %v", err)
	}
	if !strings.Contains(content, "Decision Log") {
		t.Error("expected title in created doc")
	}
}

func TestKnowledgeAppend(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	callTool(t, ctx, ctx.handleKnowledgeAppend, map[string]any{
		"project":  "test-project",
		"filename": "overview",
		"content":  "Appended by test.",
	})

	dir := filepath.Join(root, "test-project")
	content, _ := knowledge.Read(dir, "overview")
	if !strings.Contains(content, "Appended by test.") {
		t.Error("appended content not found")
	}
}

func TestKnowledgeDelete(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	callTool(t, ctx, ctx.handleKnowledgeDelete, map[string]any{
		"project":  "test-project",
		"filename": "overview",
	})

	dir := filepath.Join(root, "test-project")
	path := knowledge.FilePath(dir, "overview")
	if _, err := os.Stat(path); err == nil {
		t.Error("doc should be deleted")
	}
}

func TestTodoMove(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	callTool(t, ctx, ctx.handleTodoMove, map[string]any{
		"project":     "test-project",
		"list":        "tasks",
		"item_id":     "1",
		"target_list": "done-tasks",
	})

	dir := filepath.Join(root, "test-project")
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

// callToolAllowError calls a handler and returns the text result plus whether it was an error.
// Unlike callTool, it does not fatal on tool-level errors (IsError=true).
func callToolAllowError(t *testing.T, ctx *serverCtx, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) (string, bool) {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			return tc.Text, result.IsError
		}
	}
	return "", result.IsError
}

// --- Missing handler tests ---

func TestTodoUpdate(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	callTool(t, ctx, ctx.handleTodoUpdate, map[string]any{
		"project": "test-project",
		"list":    "tasks",
		"item_id": "1",
		"text":    "Updated first task",
	})

	dir := filepath.Join(root, "test-project")
	list, _ := todo.LoadList(dir, "tasks")
	if list.Items[0].Text != "Updated first task" {
		t.Errorf("expected updated text, got %q", list.Items[0].Text)
	}
}

func TestKnowledgeReplace(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}
	dir := filepath.Join(root, "test-project")

	// The "overview" doc created by setupTestRoot has a section "# Overview".
	// Append a subsection so we can replace it.
	knowledge.Append(dir, "overview", "## Details\n\nOld details content.", "")

	callTool(t, ctx, ctx.handleKnowledgeReplace, map[string]any{
		"project":  "test-project",
		"filename": "overview",
		"section":  "Details",
		"content":  "New details content.",
	})

	content, _ := knowledge.Read(dir, "overview")
	if strings.Contains(content, "Old details content") {
		t.Errorf("old content should be gone, got:\n%s", content)
	}
	if !strings.Contains(content, "New details content.") {
		t.Errorf("new content should be present, got:\n%s", content)
	}
}

func TestKnowledgeRename(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	callTool(t, ctx, ctx.handleKnowledgeRename, map[string]any{
		"project":      "test-project",
		"old_filename": "overview",
		"new_filename": "architecture",
	})

	dir := filepath.Join(root, "test-project")

	// Old name should not exist
	oldPath := knowledge.FilePath(dir, "overview")
	if _, err := os.Stat(oldPath); err == nil {
		t.Error("old file should not exist after rename")
	}

	// New name should exist
	content, err := knowledge.Read(dir, "architecture")
	if err != nil {
		t.Fatalf("reading renamed doc: %v", err)
	}
	if !strings.Contains(content, "overview content") {
		t.Errorf("renamed doc should preserve content, got:\n%s", content)
	}
}

func TestTodoPriority(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	// Item 1 starts with priority=now, change to backlog
	callTool(t, ctx, ctx.handleTodoPriority, map[string]any{
		"project":  "test-project",
		"list":     "tasks",
		"item_id":  "1",
		"priority": "backlog",
	})

	dir := filepath.Join(root, "test-project")
	list, _ := todo.LoadList(dir, "tasks")
	if list.Items[0].Priority != todo.Backlog {
		t.Errorf("expected backlog priority, got %s", list.Items[0].Priority)
	}
}

func TestTodoDue(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	callTool(t, ctx, ctx.handleTodoDue, map[string]any{
		"project": "test-project",
		"list":    "tasks",
		"item_id": "1",
		"due":     "2026-12-31",
	})

	dir := filepath.Join(root, "test-project")
	list, _ := todo.LoadList(dir, "tasks")
	if list.Items[0].Due != "2026-12-31" {
		t.Errorf("expected due 2026-12-31, got %q", list.Items[0].Due)
	}
}

func TestTodoRmList(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	callTool(t, ctx, ctx.handleTodoRmList, map[string]any{
		"project": "test-project",
		"list":    "tasks",
	})

	dir := filepath.Join(root, "test-project")
	path := todo.ListPath(dir, "tasks")
	if _, err := os.Stat(path); err == nil {
		t.Error("todo list file should be deleted")
	}
}

func TestKnowledgeList(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	text := callTool(t, ctx, ctx.handleKnowledgeList, map[string]any{
		"project": "test-project",
	})

	if !strings.Contains(text, "overview") {
		t.Errorf("expected 'overview' in list output: %s", text)
	}
	if !strings.Contains(text, "bytes") {
		t.Errorf("expected size in bytes in output: %s", text)
	}
}

func TestKnowledgeSearch(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	// The overview doc contains "overview content" — search for it
	text := callTool(t, ctx, ctx.handleKnowledgeSearch, map[string]any{
		"project": "test-project",
		"query":   "overview content",
	})

	if !strings.Contains(text, "overview") {
		t.Errorf("expected overview in search results: %s", text)
	}

	// Search for something that doesn't exist
	text = callTool(t, ctx, ctx.handleKnowledgeSearch, map[string]any{
		"project": "test-project",
		"query":   "xyznonexistent",
	})
	if !strings.Contains(text, "No matches") {
		t.Errorf("expected no matches: %s", text)
	}
}

func TestStatus(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	// Status for a specific project
	text := callTool(t, ctx, ctx.handleStatus, map[string]any{
		"project": "test-project",
	})

	if !strings.Contains(text, "test-project") {
		t.Errorf("expected project name in status: %s", text)
	}
	if !strings.Contains(text, "open=") {
		t.Errorf("expected open count in status: %s", text)
	}

	// Status for all projects (no project param)
	text = callTool(t, ctx, ctx.handleStatus, map[string]any{})
	if !strings.Contains(text, "test-project") {
		t.Errorf("expected project name in all-project status: %s", text)
	}
}

func TestSearch(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	// Search for known todo text
	text := callTool(t, ctx, ctx.handleSearch, map[string]any{
		"query":   "First task",
		"project": "test-project",
	})
	if !strings.Contains(text, "First task") {
		t.Errorf("expected 'First task' in search results: %s", text)
	}

	// Search for known knowledge content
	text = callTool(t, ctx, ctx.handleSearch, map[string]any{
		"query":   "overview content",
		"project": "test-project",
	})
	if !strings.Contains(text, "overview") {
		t.Errorf("expected 'overview' in search results: %s", text)
	}

	// Search with no matches
	text = callTool(t, ctx, ctx.handleSearch, map[string]any{
		"query":   "absolutelynothingtofind",
		"project": "test-project",
	})
	if !strings.Contains(text, "No matches") {
		t.Errorf("expected 'No matches' for empty search: %s", text)
	}
}

// --- Error case tests ---

func TestTodoAddMissingParams(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	// Empty project
	text, isErr := callToolAllowError(t, ctx, ctx.handleTodoAdd, map[string]any{
		"project": "",
		"list":    "tasks",
		"text":    "some task",
	})
	if !isErr {
		t.Errorf("expected error for empty project, got: %s", text)
	}

	// Empty list
	text, isErr = callToolAllowError(t, ctx, ctx.handleTodoAdd, map[string]any{
		"project": "test-project",
		"list":    "",
		"text":    "some task",
	})
	if !isErr {
		t.Errorf("expected error for empty list, got: %s", text)
	}

	// Empty text
	text, isErr = callToolAllowError(t, ctx, ctx.handleTodoAdd, map[string]any{
		"project": "test-project",
		"list":    "tasks",
		"text":    "",
	})
	if !isErr {
		t.Errorf("expected error for empty text, got: %s", text)
	}
}

func TestTodoStateMissingParams(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	text, isErr := callToolAllowError(t, ctx, ctx.handleTodoState, map[string]any{
		"project": "",
		"list":    "",
		"item_id": "",
		"state":   "",
	})
	if !isErr {
		t.Errorf("expected error for empty params, got: %s", text)
	}
}

func TestTodoStateInvalidState(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	text, isErr := callToolAllowError(t, ctx, ctx.handleTodoState, map[string]any{
		"project": "test-project",
		"list":    "tasks",
		"item_id": "1",
		"state":   "invalid",
	})
	if !isErr {
		t.Errorf("expected error for invalid state, got: %s", text)
	}
	if !strings.Contains(text, "invalid") {
		t.Errorf("expected error message to mention invalid state: %s", text)
	}
}

func TestTodoRemoveMissingParams(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	text, isErr := callToolAllowError(t, ctx, ctx.handleTodoRemove, map[string]any{
		"project": "",
		"list":    "",
		"item_id": "",
	})
	if !isErr {
		t.Errorf("expected error for empty params, got: %s", text)
	}
}

func TestTodoUpdateMissingParams(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	text, isErr := callToolAllowError(t, ctx, ctx.handleTodoUpdate, map[string]any{
		"project": "",
		"list":    "",
		"item_id": "",
		"text":    "",
	})
	if !isErr {
		t.Errorf("expected error for empty params, got: %s", text)
	}
}

func TestKnowledgeCreateMissingParams(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	text, isErr := callToolAllowError(t, ctx, ctx.handleKnowledgeCreate, map[string]any{
		"project":  "",
		"filename": "",
		"title":    "",
	})
	if !isErr {
		t.Errorf("expected error for empty params, got: %s", text)
	}
}

func TestTodoAddInvalidProject(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	text, isErr := callToolAllowError(t, ctx, ctx.handleTodoAdd, map[string]any{
		"project": "nonexistent-project",
		"list":    "tasks",
		"text":    "some task",
	})
	if !isErr {
		t.Errorf("expected error for nonexistent project, got: %s", text)
	}
}

func TestTodoResolveInvalidID(t *testing.T) {
	root := setupTestRoot(t)
	ctx := &serverCtx{projectRoot: root}

	// The list has only 2 items, so item_id=99 should fail
	text, isErr := callToolAllowError(t, ctx, ctx.handleTodoState, map[string]any{
		"project": "test-project",
		"list":    "tasks",
		"item_id": "99",
		"state":   "done",
	})
	if !isErr {
		t.Errorf("expected error for invalid item ID, got: %s", text)
	}
}
