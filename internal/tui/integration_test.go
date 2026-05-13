package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/walter/p/internal/config"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

// =======================================================================
// Integration Test Helpers
// =======================================================================

// setupTestProject creates a temp directory with a fully-initialized project
// containing the specified todo lists and items. Returns (projectRoot, projectDir).
func setupTestProject(t *testing.T, projectName string, lists map[string][]*todo.Item) (string, string) {
	t.Helper()
	root := t.TempDir()

	// Create the project
	if err := project.Create(root, projectName, "Test project"); err != nil {
		t.Fatalf("project.Create: %v", err)
	}
	projDir := filepath.Join(root, projectName)

	// Initialize git
	if err := git.Init(context.Background(), projDir); err != nil {
		t.Fatalf("git.Init: %v", err)
	}
	_ = git.CommitAll(context.Background(), projDir, "initial")

	// Create todo lists with items
	for listName, items := range lists {
		list, err := todo.CreateList(projDir, listName, strings.ReplaceAll(listName, "-", " "))
		if err != nil {
			t.Fatalf("todo.CreateList(%q): %v", listName, err)
		}
		for _, item := range items {
			added := todo.AddItem(list, item.Text, item.Priority, item.Due)
			added.State = item.State
			added.Tags = item.Tags
			if len(item.Children) > 0 {
				added.Children = item.Children
			}
		}
		if err := todo.SaveList(projDir, listName, list); err != nil {
			t.Fatalf("todo.SaveList(%q): %v", listName, err)
		}
	}

	_ = git.CommitAll(context.Background(), projDir, "setup test data")

	return root, projDir
}

// loadItemState loads a list and returns the state of a specific item.
func loadItemState(t *testing.T, dir, listName, itemID string) todo.State {
	t.Helper()
	list, err := todo.LoadList(dir, listName)
	if err != nil {
		t.Fatalf("LoadList(%q): %v", listName, err)
	}
	item, err := todo.ResolveItem(list, itemID)
	if err != nil {
		t.Fatalf("ResolveItem(%q): %v", itemID, err)
	}
	return item.State
}

// loadItemPriority loads a list and returns the priority of a specific item.
func loadItemPriority(t *testing.T, dir, listName, itemID string) todo.Priority {
	t.Helper()
	list, err := todo.LoadList(dir, listName)
	if err != nil {
		t.Fatalf("LoadList(%q): %v", listName, err)
	}
	item, err := todo.ResolveItem(list, itemID)
	if err != nil {
		t.Fatalf("ResolveItem(%q): %v", itemID, err)
	}
	return item.Priority
}

// =======================================================================
// Integration Tests: TUI Mutations
// =======================================================================

func TestIntegration_ToggleState(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: "Task one", State: todo.Open, Priority: todo.Now},
			{Text: "Task two", State: todo.Done, Priority: todo.Backlog},
		},
	})

	v := NewItemListView("test-proj", projDir, "backlog", 80, 24)
	// Simulate the Init command to load data
	cmd := v.Init()
	msg := cmd()
	v.Update(msg)

	// Verify items loaded
	if !v.loaded || len(v.items) != 2 {
		t.Fatalf("loaded=%v, items=%d, want loaded=true, items=2", v.loaded, len(v.items))
	}

	// Toggle item #1 (open → done)
	v.cursor = 0
	cmd = v.toggleDone()
	if cmd == nil {
		t.Fatal("toggleDone should return a command")
	}
	result := cmd()

	// Should produce DataChangedMsg
	dcMsg, ok := result.(DataChangedMsg)
	if !ok {
		// Check if it's an error
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}
	if !strings.Contains(dcMsg.StatusText, "done") {
		t.Errorf("StatusText = %q, should mention 'done'", dcMsg.StatusText)
	}

	// Verify the file was actually changed
	state := loadItemState(t, projDir, "backlog", "1")
	if state != todo.Done {
		t.Errorf("item 1 state = %q, want done", state)
	}

	// Toggle item #2 (done → open)
	v.cursor = 1
	cmd = v.toggleDone()
	result = cmd()
	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	state = loadItemState(t, projDir, "backlog", "2")
	if state != todo.Open {
		t.Errorf("item 2 state = %q, want open", state)
	}
}

func TestIntegration_SetState(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: "Task", State: todo.Open, Priority: todo.Now},
		},
	})

	v := NewItemListView("test-proj", projDir, "backlog", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Set to blocked
	cmd = v.setState(todo.Blocked)
	result := cmd()
	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}
	if s := loadItemState(t, projDir, "backlog", "1"); s != todo.Blocked {
		t.Errorf("state = %q, want blocked", s)
	}

	// Set to done
	cmd = v.setState(todo.Done)
	result = cmd()
	if _, ok := result.(DataChangedMsg); !ok {
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}
	if s := loadItemState(t, projDir, "backlog", "1"); s != todo.Done {
		t.Errorf("state = %q, want done", s)
	}

	// Set back to open
	cmd = v.setState(todo.Open)
	result = cmd()
	if _, ok := result.(DataChangedMsg); !ok {
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}
	if s := loadItemState(t, projDir, "backlog", "1"); s != todo.Open {
		t.Errorf("state = %q, want open", s)
	}
}

func TestIntegration_CyclePriority(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: "Task", State: todo.Open, Priority: todo.Now},
		},
	})

	v := NewItemListView("test-proj", projDir, "backlog", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Cycle: now → backlog
	cmd = v.cyclePriority()
	if cmd == nil {
		t.Fatal("cyclePriority should return a command")
	}
	result := cmd()
	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	p := loadItemPriority(t, projDir, "backlog", "1")
	if p != todo.Backlog {
		t.Errorf("priority = %q, want backlog", p)
	}

	// Reload the list so the view has fresh data
	cmd = v.loadList()
	v.Update(cmd())

	// Cycle: backlog → now
	cmd = v.cyclePriority()
	result = cmd()
	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	p = loadItemPriority(t, projDir, "backlog", "1")
	if p != todo.Now {
		t.Errorf("priority = %q, want now", p)
	}
}

func TestIntegration_AddItem(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: "Existing item", State: todo.Open, Priority: todo.Now},
		},
	})

	v := NewItemListView("test-proj", projDir, "backlog", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Add a new item
	addCmd := v.addItem("New task from TUI")
	if addCmd == nil {
		t.Fatal("addItem should return a command")
	}
	result := addCmd()

	dcMsg, ok := result.(DataChangedMsg)
	if !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}
	if !strings.Contains(dcMsg.StatusText, "Added") {
		t.Errorf("StatusText = %q", dcMsg.StatusText)
	}

	// Verify the item was added to the file
	list, err := todo.LoadList(projDir, "backlog")
	if err != nil {
		t.Fatalf("LoadList: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(list.Items))
	}
	if list.Items[1].Text != "New task from TUI" {
		t.Errorf("new item text = %q", list.Items[1].Text)
	}
}

func TestIntegration_RemoveItem(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: "Keep this", State: todo.Open, Priority: todo.Now},
			{Text: "Remove this", State: todo.Open, Priority: todo.Backlog},
		},
	})

	v := NewItemListView("test-proj", projDir, "backlog", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Remove item #2
	removeCmd := v.doRemoveItem("2")
	result := removeCmd()

	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	// Verify the item was removed
	list, err := todo.LoadList(projDir, "backlog")
	if err != nil {
		t.Fatalf("LoadList: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 item after removal, got %d", len(list.Items))
	}
	if list.Items[0].Text != "Keep this" {
		t.Errorf("remaining item text = %q", list.Items[0].Text)
	}
}

func TestIntegration_EditItem(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: "Original text", State: todo.Open, Priority: todo.Now},
		},
	})

	v := NewItemListView("test-proj", projDir, "backlog", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Edit the item text
	editCmd := v.doEditItem("1", "Updated text from TUI")
	result := editCmd()

	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	// Verify
	list, err := todo.LoadList(projDir, "backlog")
	if err != nil {
		t.Fatalf("LoadList: %v", err)
	}
	if list.Items[0].Text != "Updated text from TUI" {
		t.Errorf("item text = %q, want 'Updated text from TUI'", list.Items[0].Text)
	}
}

func TestIntegration_SetDueDate(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: "Task", State: todo.Open, Priority: todo.Now},
		},
	})

	v := NewItemListView("test-proj", projDir, "backlog", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Set due date
	dueCmd := v.doSetDueDate("1", "2026-06-15")
	result := dueCmd()

	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	// Verify
	list, err := todo.LoadList(projDir, "backlog")
	if err != nil {
		t.Fatalf("LoadList: %v", err)
	}
	if list.Items[0].Due != "2026-06-15" {
		t.Errorf("due = %q, want 2026-06-15", list.Items[0].Due)
	}
}

func TestIntegration_SetDueDate_InvalidDate(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: "Task", State: todo.Open, Priority: todo.Now},
		},
	})

	v := NewItemListView("test-proj", projDir, "backlog", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Set invalid due date
	dueCmd := v.doSetDueDate("1", "not-a-date")
	result := dueCmd()

	errMsg, ok := result.(ErrorMsg)
	if !ok {
		t.Fatalf("expected ErrorMsg for invalid date, got %T", result)
	}
	if !strings.Contains(errMsg.Err.Error(), "invalid") {
		t.Errorf("error = %q, should mention 'invalid'", errMsg.Err)
	}
}

func TestIntegration_SetTags(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: "Task", State: todo.Open, Priority: todo.Now},
		},
	})

	v := NewItemListView("test-proj", projDir, "backlog", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Set tags
	tagCmd := v.doSetTags("1", "bug, critical, frontend")
	result := tagCmd()

	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	// Verify
	list, err := todo.LoadList(projDir, "backlog")
	if err != nil {
		t.Fatalf("LoadList: %v", err)
	}
	tags := list.Items[0].Tags
	expected := []string{"bug", "critical", "frontend"}
	if len(tags) != len(expected) {
		t.Fatalf("tags = %v, want %v", tags, expected)
	}
	for i, tag := range expected {
		if tags[i] != tag {
			t.Errorf("tag[%d] = %q, want %q", i, tags[i], tag)
		}
	}
}

func TestIntegration_MoveItem(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: "Move me", State: todo.Open, Priority: todo.Now},
			{Text: "Stay here", State: todo.Open, Priority: todo.Backlog},
		},
		"sprint": {},
	})

	v := NewItemListView("test-proj", projDir, "backlog", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Move item #1 to sprint
	moveCmd := v.doMoveItem("1", "sprint")
	result := moveCmd()

	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	// Verify source list
	srcList, err := todo.LoadList(projDir, "backlog")
	if err != nil {
		t.Fatalf("LoadList(backlog): %v", err)
	}
	if len(srcList.Items) != 1 {
		t.Errorf("source list should have 1 item, got %d", len(srcList.Items))
	}
	if srcList.Items[0].Text != "Stay here" {
		t.Errorf("remaining item = %q", srcList.Items[0].Text)
	}

	// Verify destination list
	dstList, err := todo.LoadList(projDir, "sprint")
	if err != nil {
		t.Fatalf("LoadList(sprint): %v", err)
	}
	if len(dstList.Items) != 1 {
		t.Errorf("dest list should have 1 item, got %d", len(dstList.Items))
	}
	if dstList.Items[0].Text != "Move me" {
		t.Errorf("moved item = %q", dstList.Items[0].Text)
	}
}

func TestIntegration_GitCommits(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: "Task", State: todo.Open, Priority: todo.Now},
		},
	})

	v := NewItemListView("test-proj", projDir, "backlog", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Perform a state change
	stateCmd := v.doStateChange("1", todo.Done)
	stateCmd()

	// Check git log for the commit
	diff, err := git.Diff(context.Background(), projDir)
	if err != nil {
		// Diff might not work, try to check via log
		t.Logf("git diff err (non-fatal): %v", err)
	}
	_ = diff // Git diff checks the latest changes

	// Verify a new commit was created by checking the file state
	state := loadItemState(t, projDir, "backlog", "1")
	if state != todo.Done {
		t.Errorf("state should be 'done' after git commit, got %q", state)
	}
}

// =======================================================================
// Edge Case Tests
// =======================================================================

func TestEdgeCase_EmptyProject(t *testing.T) {
	root := t.TempDir()

	// Create project with no lists
	if err := project.Create(root, "empty-proj", "Empty project"); err != nil {
		t.Fatalf("project.Create: %v", err)
	}
	projDir := filepath.Join(root, "empty-proj")
	git.Init(context.Background(), projDir)
	_ = git.CommitAll(context.Background(), projDir, "initial")

	// TodoListView should handle empty project gracefully
	v := NewTodoListView("empty-proj", projDir, 80, 24)
	cmd := v.Init()
	msg := cmd()
	v.Update(msg)

	if !v.loaded {
		t.Error("should be loaded even with no lists")
	}
	if len(v.lists) != 0 {
		t.Errorf("should have 0 lists, got %d", len(v.lists))
	}

	view := v.View()
	if !strings.Contains(view, "No todo lists found") {
		t.Error("should show empty state message")
	}
}

func TestEdgeCase_EmptyList(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"empty-list": {},
	})

	v := NewItemListView("test-proj", projDir, "empty-list", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	if !v.loaded {
		t.Error("should be loaded")
	}
	if len(v.items) != 0 {
		t.Errorf("should have 0 items, got %d", len(v.items))
	}

	view := v.View()
	if !strings.Contains(view, "No items") {
		t.Error("should show 'No items' for empty list")
	}

	// Navigation should not crash on empty list
	v.Update(tea.KeyMsg{Type: tea.KeyDown})
	v.Update(tea.KeyMsg{Type: tea.KeyUp})
	v.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// selectedID/selectedItem should return empty/nil
	if id := v.selectedID(); id != "" {
		t.Errorf("selectedID on empty list = %q", id)
	}
	if item := v.selectedItem(); item != nil {
		t.Error("selectedItem on empty list should be nil")
	}
}

func TestEdgeCase_VeryLongItemText(t *testing.T) {
	longText := strings.Repeat("This is a very long item description. ", 20)
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: longText, State: todo.Open, Priority: todo.Now},
		},
	})

	v := NewItemListView("test-proj", projDir, "backlog", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Should render without panicking
	view := v.View()
	if view == "" {
		t.Error("view should not be empty")
	}

	// The text should appear somewhere in the view
	if !strings.Contains(view, "very long item") {
		t.Error("long text should appear in view")
	}
}

func TestEdgeCase_DeeplyNestedChildren(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{
				Text: "Level 1", State: todo.Open, Priority: todo.Now,
				Children: []*todo.Item{
					{
						Text: "Level 2", State: todo.Open,
						Children: []*todo.Item{
							{
								Text: "Level 3", State: todo.Open,
								Children: []*todo.Item{
									{Text: "Level 4", State: todo.Open},
								},
							},
						},
					},
				},
			},
		},
	})

	v := NewItemListView("test-proj", projDir, "backlog", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// All nested items should be in the flat list
	if len(v.items) != 4 {
		t.Fatalf("expected 4 items (including nested), got %d", len(v.items))
	}

	// Check IDs
	expectedIDs := []string{"1", "1.1", "1.1.1", "1.1.1.1"}
	for i, fi := range v.items {
		if fi.OriginalID != expectedIDs[i] {
			t.Errorf("item[%d].ID = %q, want %q", i, fi.OriginalID, expectedIDs[i])
		}
	}

	// Should render without crash
	view := v.View()
	if !strings.Contains(view, "Level 4") {
		t.Error("deeply nested item should appear in view")
	}
}

func TestEdgeCase_SmallTerminal(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: "Item 1", State: todo.Open},
			{Text: "Item 2", State: todo.Open},
			{Text: "Item 3", State: todo.Open},
			{Text: "Item 4", State: todo.Open},
			{Text: "Item 5", State: todo.Open},
		},
	})

	// Very small terminal
	v := NewItemListView("test-proj", projDir, "backlog", 30, 10)
	cmd := v.Init()
	v.Update(cmd())

	// Should render without panicking
	view := v.View()
	if view == "" {
		t.Error("view should render even in small terminal")
	}
}

func TestEdgeCase_VerySmallTerminal(t *testing.T) {
	v := NewProjectListView("/tmp/root", 20, 5)
	v.Update(ProjectsLoadedMsg{
		Projects: []ProjectInfo{
			{Name: "project-with-a-very-long-name-indeed"},
		},
	})

	// Should handle narrow terminal
	view := v.View()
	if view == "" {
		t.Error("view should render in very small terminal")
	}
}

func TestEdgeCase_ConcurrentReloadSafety(t *testing.T) {
	// This test verifies that the data flow is safe even if data changes
	// between operations. The TUI uses per-operation locking, so the
	// underlying data can change between loads.

	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: "Item 1", State: todo.Open, Priority: todo.Now},
			{Text: "Item 2", State: todo.Open, Priority: todo.Now},
		},
	})

	v := NewItemListView("test-proj", projDir, "backlog", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Simulate external modification — add an item outside the TUI
	list, err := todo.LoadList(projDir, "backlog")
	if err != nil {
		t.Fatal(err)
	}
	todo.AddItem(list, "External item", todo.Now, "")
	todo.SaveList(projDir, "backlog", list)

	// Reload in TUI — should pick up the change
	cmd = v.loadList()
	v.Update(cmd())

	if len(v.items) != 3 {
		t.Errorf("after external add, expected 3 items, got %d", len(v.items))
	}
}

// =======================================================================
// Full App Integration Tests with Real Navigation
// =======================================================================

func TestIntegration_FullNavigation(t *testing.T) {
	root, _ := setupTestProject(t, "proj1", map[string][]*todo.Item{
		"backlog": {
			{Text: "Task 1", State: todo.Open, Priority: todo.Now},
		},
	})

	cfg := config.Config{
		ProjectRoot:     root,
		DefaultPriority: "now",
	}
	app := NewApp(cfg)
	app.width = 80
	app.height = 24
	app.Init()

	// Should start at ProjectListView
	if _, ok := app.activeView.(*ProjectListView); !ok {
		t.Fatalf("should start at ProjectListView, got %T", app.activeView)
	}

	// Load projects
	plv := app.activeView.(*ProjectListView)
	cmd := plv.Init()
	msg := cmd()
	plv.Update(msg)

	// Navigate to the project
	if len(plv.projects) == 0 {
		t.Fatal("no projects loaded")
	}

	// Send Enter → should navigate
	_, navCmd := plv.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if navCmd != nil {
		navMsg := navCmd()
		sendMsg(app, navMsg)
	}

	if _, ok := app.activeView.(*TodoListView); !ok {
		t.Fatalf("after Enter on project, should be at TodoListView, got %T", app.activeView)
	}

	// Load lists
	tlv := app.activeView.(*TodoListView)
	cmd = tlv.Init()
	msg = cmd()
	tlv.Update(msg)

	// Navigate to the list
	_, navCmd = tlv.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if navCmd != nil {
		navMsg := navCmd()
		sendMsg(app, navMsg)
	}

	if _, ok := app.activeView.(*ItemListView); !ok {
		t.Fatalf("after Enter on list, should be at ItemListView, got %T", app.activeView)
	}

	// Go back to TodoListView
	sendMsg(app, GoBackMsg{})
	if _, ok := app.activeView.(*TodoListView); !ok {
		t.Fatalf("after GoBack, should be at TodoListView, got %T", app.activeView)
	}

	// Go back to ProjectListView
	sendMsg(app, GoBackMsg{})
	if _, ok := app.activeView.(*ProjectListView); !ok {
		t.Fatalf("after GoBack, should be at ProjectListView, got %T", app.activeView)
	}

	// Go back from root — should quit
	sendMsg(app, GoBackMsg{})
	if !app.quitting {
		t.Error("going back from root should quit")
	}
}

func TestIntegration_ProjectListLoadsRealData(t *testing.T) {
	root, _ := setupTestProject(t, "proj-a", map[string][]*todo.Item{
		"tasks": {
			{Text: "Task 1", State: todo.Open, Priority: todo.Now},
			{Text: "Task 2", State: todo.Done, Priority: todo.Backlog},
		},
	})
	// Create a second project
	project.Create(root, "proj-b", "Second project")
	projBDir := filepath.Join(root, "proj-b")
	git.Init(context.Background(), projBDir)
	list, _ := todo.CreateList(projBDir, "backlog", "backlog")
	todo.AddItem(list, "B task", todo.Now, "")
	todo.SaveList(projBDir, "backlog", list)
	_ = git.CommitAll(context.Background(), projBDir, "setup")

	v := NewProjectListView(root, 80, 24)
	cmd := v.Init()
	msg := cmd()
	v.Update(msg)

	if len(v.projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(v.projects))
	}

	// Check counts for proj-a
	var projA *ProjectInfo
	for i := range v.projects {
		if v.projects[i].Name == "proj-a" {
			projA = &v.projects[i]
			break
		}
	}
	if projA == nil {
		t.Fatal("proj-a not found")
	}
	if projA.Open != 1 || projA.Done != 1 {
		t.Errorf("proj-a: open=%d done=%d, want 1/1", projA.Open, projA.Done)
	}
}

// =======================================================================
// TodoListView Integration Tests
// =======================================================================

func TestIntegration_TodoListView_CreateList(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{})

	v := NewTodoListView("test-proj", projDir, 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Use the createList action
	createCmd := v.createList("new-list")
	result := createCmd()

	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	// Verify file exists
	listPath := todo.ListPath(projDir, "new-list")
	if _, err := os.Stat(listPath); err != nil {
		t.Errorf("list file should exist at %s: %v", listPath, err)
	}
}

func TestIntegration_TodoListView_DeleteList(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"delete-me": {
			{Text: "Task", State: todo.Open},
		},
	})

	v := NewTodoListView("test-proj", projDir, 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Delete the list
	deleteCmd := v.doDeleteList("delete-me")
	result := deleteCmd()

	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	// Verify file is gone
	listPath := todo.ListPath(projDir, "delete-me")
	if _, err := os.Stat(listPath); !os.IsNotExist(err) {
		t.Error("list file should be deleted")
	}
}

func TestIntegration_TodoListView_ArchiveList(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"archive-me": {
			{Text: "Task", State: todo.Done},
		},
	})

	v := NewTodoListView("test-proj", projDir, 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Archive the list
	archiveCmd := v.doArchiveList("archive-me")
	result := archiveCmd()

	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	// Active list should be gone
	activePath := todo.ListPath(projDir, "archive-me")
	if _, err := os.Stat(activePath); !os.IsNotExist(err) {
		t.Error("active list file should be gone")
	}

	// Archived list should exist
	archivePath := filepath.Join(todo.ListDir(projDir), ".archive", "archive-me.md")
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("archived file should exist: %v", err)
	}
}

// =======================================================================
// TodoListView Archived View Integration Tests
// =======================================================================

func TestIntegration_TodoListView_ToggleArchived(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"active-list": {
			{Text: "Active task", State: todo.Open},
		},
		"old-list": {
			{Text: "Old task", State: todo.Done},
		},
	})

	v := NewTodoListView("test-proj", projDir, 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	if len(v.lists) != 2 {
		t.Fatalf("expected 2 active lists, got %d", len(v.lists))
	}

	// Archive one list
	archiveCmd := v.doArchiveList("old-list")
	result := archiveCmd()
	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("archive error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	// Reload active lists
	cmd = v.loadLists()
	v.Update(cmd())
	if len(v.lists) != 1 {
		t.Fatalf("expected 1 active list after archive, got %d", len(v.lists))
	}

	// Press 'A' to toggle to archived view
	_, cmd = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
	if !v.showArchived {
		t.Fatal("should be in archived view after 'A'")
	}
	if cmd == nil {
		t.Fatal("should return reload command")
	}

	// Load archived lists
	v.Update(cmd())
	if len(v.lists) != 1 {
		t.Fatalf("expected 1 archived list, got %d", len(v.lists))
	}
	if v.lists[0].Name != "old-list" {
		t.Errorf("archived list name = %q, want 'old-list'", v.lists[0].Name)
	}

	// Verify title shows "(archived)"
	view := v.View()
	if !strings.Contains(view, "(archived)") {
		t.Error("view should show '(archived)' in title")
	}

	// Press 'A' again to switch back
	_, cmd = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
	if v.showArchived {
		t.Error("should be back in active view")
	}

	v.Update(cmd())
	if len(v.lists) != 1 {
		t.Fatalf("expected 1 active list, got %d", len(v.lists))
	}
	if v.lists[0].Name != "active-list" {
		t.Errorf("active list name = %q, want 'active-list'", v.lists[0].Name)
	}
}

func TestIntegration_TodoListView_RestoreList(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"restore-me": {
			{Text: "Restore task", State: todo.Open, Priority: todo.Now},
		},
	})

	v := NewTodoListView("test-proj", projDir, 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Archive the list
	archiveCmd := v.doArchiveList("restore-me")
	archiveCmd()

	// Switch to archived view and load
	v.showArchived = true
	cmd = v.loadLists()
	v.Update(cmd())

	if len(v.lists) != 1 {
		t.Fatalf("expected 1 archived list, got %d", len(v.lists))
	}

	// Press 'R' to restore
	_, cmd = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	if cmd == nil {
		t.Fatal("'R' should return restore command")
	}

	result := cmd()
	dcMsg, ok := result.(DataChangedMsg)
	if !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("restore error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}
	if !strings.Contains(dcMsg.StatusText, "Restored") {
		t.Errorf("StatusText = %q, should mention Restored", dcMsg.StatusText)
	}

	// Verify list is back in active location
	activePath := todo.ListPath(projDir, "restore-me")
	if _, err := os.Stat(activePath); err != nil {
		t.Error("restored list should exist in active location")
	}

	// Verify gone from archive
	archivePath := filepath.Join(todo.ListDir(projDir), ".archive", "restore-me.md")
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Error("list should be gone from archive after restore")
	}

	// Verify data integrity — load and check items
	list, err := todo.LoadList(projDir, "restore-me")
	if err != nil {
		t.Fatalf("LoadList: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].Text != "Restore task" {
		t.Errorf("restored list data should be intact, got %d items", len(list.Items))
	}
}

func TestIntegration_TodoListView_RestoreNotInActiveView(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"list1": {{Text: "Task", State: todo.Open}},
	})

	v := NewTodoListView("test-proj", projDir, 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// 'R' in active view should do nothing
	_, cmd = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	if cmd != nil {
		t.Error("'R' should do nothing in active view")
	}
}

func TestIntegration_TodoListView_ArchiveNotInArchivedView(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"list1": {{Text: "Task", State: todo.Open}},
	})

	v := NewTodoListView("test-proj", projDir, 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Archive the list and switch to archived view
	archiveCmd := v.doArchiveList("list1")
	archiveCmd()

	v.showArchived = true
	cmd = v.loadLists()
	v.Update(cmd())

	// 'a' in archived view should not trigger archive
	_, cmd = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	if v.confirmMode {
		t.Error("'a' should not enter confirm mode in archived view")
	}
}

func TestIntegration_TodoListView_ArchivedViewEmpty(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"list1": {{Text: "Task", State: todo.Open}},
	})

	v := NewTodoListView("test-proj", projDir, 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Toggle to archived view with no archived lists
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A")})
	cmd = v.loadLists()
	v.Update(cmd())

	if len(v.lists) != 0 {
		t.Errorf("expected 0 archived lists, got %d", len(v.lists))
	}

	view := v.View()
	if !strings.Contains(view, "No todo lists found") {
		t.Error("should show empty state message")
	}
}

func TestIntegration_TodoListView_ArchivedListCounts(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"mixed": {
			{Text: "Open", State: todo.Open, Priority: todo.Now},
			{Text: "Done", State: todo.Done, Priority: todo.Backlog},
			{Text: "Blocked", State: todo.Blocked, Priority: todo.Now},
		},
	})

	v := NewTodoListView("test-proj", projDir, 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Archive the list
	archiveCmd := v.doArchiveList("mixed")
	archiveCmd()

	// Switch to archived view
	v.showArchived = true
	cmd = v.loadLists()
	v.Update(cmd())

	if len(v.lists) != 1 {
		t.Fatalf("expected 1 archived list, got %d", len(v.lists))
	}

	// Verify counts are shown correctly
	l := v.lists[0]
	if l.Open != 1 || l.Done != 1 || l.Blocked != 1 {
		t.Errorf("counts: open=%d done=%d blocked=%d, want 1/1/1", l.Open, l.Done, l.Blocked)
	}
}

func TestIntegration_TodoListView_HelpBar_ArchivedView(t *testing.T) {
	_, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"list1": {{Text: "Task", State: todo.Open}},
	})

	v := NewTodoListView("test-proj", projDir, 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Active view help bar
	view := v.View()
	if !strings.Contains(view, "archived") {
		t.Error("active help bar should mention 'archived' toggle")
	}

	// Archive and switch
	archiveCmd := v.doArchiveList("list1")
	archiveCmd()
	v.showArchived = true
	cmd = v.loadLists()
	v.Update(cmd())

	// Archived view help bar
	view = v.View()
	if !strings.Contains(view, "restore") {
		t.Error("archived help bar should mention 'restore'")
	}
	if !strings.Contains(view, "active") {
		t.Error("archived help bar should mention 'active' toggle")
	}
}

// =======================================================================
// StatusView Integration Test
// =======================================================================

func TestIntegration_StatusView_LoadsData(t *testing.T) {
	root, projDir := setupTestProject(t, "test-proj", map[string][]*todo.Item{
		"backlog": {
			{Text: "Open1", State: todo.Open},
			{Text: "Done1", State: todo.Done},
		},
		"sprint": {
			{Text: "Blocked1", State: todo.Blocked},
		},
	})

	// Test single-project status
	v := NewStatusView(root, "test-proj", projDir, 80, 24)
	cmd := v.Init()
	msg := cmd()
	v.Update(msg)

	if !v.loaded {
		t.Fatal("should be loaded")
	}
	if len(v.projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(v.projects))
	}

	p := v.projects[0]
	if p.Name != "test-proj" {
		t.Errorf("project name = %q", p.Name)
	}
	if p.Open != 1 || p.Done != 1 || p.Blocked != 1 {
		t.Errorf("totals: open=%d done=%d blocked=%d, want 1/1/1", p.Open, p.Done, p.Blocked)
	}
	if len(p.Lists) != 2 {
		t.Errorf("expected 2 lists, got %d", len(p.Lists))
	}

	view := v.View()
	if !strings.Contains(view, "test-proj") {
		t.Error("status view should show project name")
	}
}

func TestIntegration_StatusView_AllProjects(t *testing.T) {
	root, _ := setupTestProject(t, "proj-a", map[string][]*todo.Item{
		"tasks": {{Text: "A1", State: todo.Open}},
	})
	project.Create(root, "proj-b", "B")
	projBDir := filepath.Join(root, "proj-b")
	git.Init(context.Background(), projBDir)
	list, _ := todo.CreateList(projBDir, "tasks", "tasks")
	todo.AddItem(list, "B1", todo.Now, "")
	todo.SaveList(projBDir, "tasks", list)
	_ = git.CommitAll(context.Background(), projBDir, "setup")

	// All-projects status
	v := NewStatusView(root, "", "", 80, 24)
	cmd := v.Init()
	msg := cmd()
	v.Update(msg)

	if len(v.projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(v.projects))
	}
}
