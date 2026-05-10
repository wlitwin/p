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
