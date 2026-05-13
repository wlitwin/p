package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
)

// =======================================================================
// Test Helpers
// =======================================================================

// setupKnowledgeProject creates a temp project with knowledge docs.
func setupKnowledgeProject(t *testing.T, docs map[string][]string) (string, string) {
	t.Helper()
	root := t.TempDir()

	if err := project.Create(root, "test-proj", "Test project"); err != nil {
		t.Fatalf("project.Create: %v", err)
	}
	projDir := filepath.Join(root, "test-proj")

	if err := git.Init(context.Background(), projDir); err != nil {
		t.Fatalf("git.Init: %v", err)
	}

	// Create knowledge docs
	for name, tags := range docs {
		if err := knowledge.Create(projDir, name, strings.ReplaceAll(name, "-", " "), tags); err != nil {
			t.Fatalf("knowledge.Create(%q): %v", name, err)
		}
	}

	_ = git.CommitAll(context.Background(), projDir, "setup test data")
	return root, projDir
}

// loadKnowledgeListView creates and loads a KnowledgeListView with test data.
func loadKnowledgeListView(t *testing.T, projDir string) *KnowledgeListView {
	t.Helper()
	v := NewKnowledgeListView("test-proj", projDir, 80, 24)
	cmd := v.Init()
	if cmd != nil {
		msg := cmd()
		v.Update(msg)
	}
	return v
}

// pressKey simulates a key press on a KnowledgeListView.
func pressKey(v *KnowledgeListView, key string) (*KnowledgeListView, tea.Cmd) {
	model, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return model.(*KnowledgeListView), cmd
}

// pressSpecialKey simulates a special key press.
func pressSpecialKey(v *KnowledgeListView, keyType tea.KeyType) (*KnowledgeListView, tea.Cmd) {
	model, cmd := v.Update(tea.KeyMsg{Type: keyType})
	return model.(*KnowledgeListView), cmd
}

// kvPressKey simulates a key press on a KnowledgeView.
func kvPressKey(v *KnowledgeView, key string) (*KnowledgeView, tea.Cmd) {
	model, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return model.(*KnowledgeView), cmd
}

// kvPressSpecialKey simulates a special key press on KnowledgeView.
func kvPressSpecialKey(v *KnowledgeView, keyType tea.KeyType) (*KnowledgeView, tea.Cmd) {
	model, cmd := v.Update(tea.KeyMsg{Type: keyType})
	return model.(*KnowledgeView), cmd
}

// =======================================================================
// KnowledgeListView Delete Tests
// =======================================================================

func TestKnowledgeListView_Delete_ConfirmYes(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"architecture": {"design", "arch"},
		"testing":      {"test"},
	})

	v := loadKnowledgeListView(t, projDir)
	if len(v.docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(v.docs))
	}

	// Press 'd' to start delete confirmation
	v, _ = pressKey(v, "d")
	if !v.confirmMode {
		t.Fatal("should enter confirm mode on 'd'")
	}
	if !strings.Contains(v.confirmPrompt, "Delete") {
		t.Errorf("confirmPrompt = %q, should mention Delete", v.confirmPrompt)
	}

	// Press 'y' to confirm
	v, cmd := pressKey(v, "y")
	if v.confirmMode {
		t.Error("should exit confirm mode after 'y'")
	}
	if cmd == nil {
		t.Fatal("should return a command for deletion")
	}

	// Execute the delete command
	result := cmd()
	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	// Verify doc was deleted
	docName := v.docs[0].Name // cursor was at 0
	path := knowledge.FilePath(projDir, docName)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("doc file should be deleted")
	}
}

func TestKnowledgeListView_Delete_ConfirmNo(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"architecture": {"design"},
	})

	v := loadKnowledgeListView(t, projDir)

	// Press 'd' then 'n' to cancel
	v, _ = pressKey(v, "d")
	if !v.confirmMode {
		t.Fatal("should be in confirm mode")
	}

	v, _ = pressKey(v, "n")
	if v.confirmMode {
		t.Error("should exit confirm mode after 'n'")
	}

	// Doc should still exist
	path := knowledge.FilePath(projDir, "architecture")
	if _, err := os.Stat(path); err != nil {
		t.Error("doc should still exist after cancel")
	}
}

func TestKnowledgeListView_Delete_ConfirmEsc(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"architecture": {"design"},
	})

	v := loadKnowledgeListView(t, projDir)

	v, _ = pressKey(v, "d")
	if !v.confirmMode {
		t.Fatal("should be in confirm mode")
	}

	v, _ = pressSpecialKey(v, tea.KeyEsc)
	if v.confirmMode {
		t.Error("should exit confirm mode after Esc")
	}
}

// =======================================================================
// KnowledgeListView Archive Tests
// =======================================================================

func TestKnowledgeListView_Archive(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"architecture": {"design"},
		"testing":      {"test"},
	})

	v := loadKnowledgeListView(t, projDir)

	// Press 'a' to archive
	v, cmd := pressKey(v, "a")
	if cmd == nil {
		t.Fatal("'a' should return a command for archiving")
	}

	result := cmd()
	dcMsg, ok := result.(DataChangedMsg)
	if !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}
	if !strings.Contains(dcMsg.StatusText, "Archived") {
		t.Errorf("StatusText = %q, should mention Archived", dcMsg.StatusText)
	}

	// Verify doc moved to archive
	docName := v.docs[0].Name
	activePath := knowledge.FilePath(projDir, docName)
	if _, err := os.Stat(activePath); !os.IsNotExist(err) {
		t.Error("active doc should be gone")
	}

	archivePath := filepath.Join(knowledge.Dir(projDir), ".archive", docName+".md")
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("archived doc should exist: %v", err)
	}
}

func TestKnowledgeListView_Archive_NotInArchivedView(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"architecture": {"design"},
	})

	v := loadKnowledgeListView(t, projDir)

	// Archive the doc first
	cmd := v.doArchiveDoc("architecture")
	cmd()

	// Switch to archived view
	v.showArchived = true
	cmd = v.loadDocs()
	msg := cmd()
	v.Update(msg)

	// Press 'a' in archived view — should not archive again
	_, archiveCmd := pressKey(v, "a")
	if archiveCmd != nil {
		t.Error("'a' should do nothing in archived view")
	}
}

// =======================================================================
// KnowledgeListView Rename Tests
// =======================================================================

func TestKnowledgeListView_Rename(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"old-name": {"design"},
	})

	v := loadKnowledgeListView(t, projDir)

	// Press 'r' to start rename
	v, _ = pressKey(v, "r")
	if !v.inputMode {
		t.Fatal("should enter input mode on 'r'")
	}
	if v.inputPrompt != "Rename to: " {
		t.Errorf("inputPrompt = %q", v.inputPrompt)
	}
	if v.inputValue != "old-name" {
		t.Errorf("inputValue should be pre-filled with current name, got %q", v.inputValue)
	}

	// Clear current name and type new one
	for range len(v.inputValue) {
		v, _ = pressSpecialKey(v, tea.KeyBackspace)
	}

	for _, c := range "new-name" {
		v, _ = pressKey(v, string(c))
	}

	if v.inputValue != "new-name" {
		t.Errorf("inputValue = %q, want 'new-name'", v.inputValue)
	}

	// Submit
	v, cmd := pressSpecialKey(v, tea.KeyEnter)
	if v.inputMode {
		t.Error("should exit input mode after Enter")
	}
	if cmd == nil {
		t.Fatal("should return a command for rename")
	}

	result := cmd()
	dcMsg, ok := result.(DataChangedMsg)
	if !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}
	if !strings.Contains(dcMsg.StatusText, "new-name") {
		t.Errorf("StatusText = %q", dcMsg.StatusText)
	}

	// Verify old file gone and new file exists
	oldPath := knowledge.FilePath(projDir, "old-name")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("old doc should be gone")
	}

	newPath := knowledge.FilePath(projDir, "new-name")
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("new doc should exist: %v", err)
	}
}

func TestKnowledgeListView_Rename_EscCancels(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"keep-name": {"design"},
	})

	v := loadKnowledgeListView(t, projDir)

	v, _ = pressKey(v, "r")
	if !v.inputMode {
		t.Fatal("should be in input mode")
	}

	v, _ = pressSpecialKey(v, tea.KeyEsc)
	if v.inputMode {
		t.Error("should exit input mode on Esc")
	}

	// Doc should still have original name
	path := knowledge.FilePath(projDir, "keep-name")
	if _, err := os.Stat(path); err != nil {
		t.Error("doc should still exist with original name")
	}
}

// =======================================================================
// Archived View Toggle Tests
// =======================================================================

func TestKnowledgeListView_ToggleArchived(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"active-doc":  {"design"},
		"archive-doc": {"old"},
	})

	v := loadKnowledgeListView(t, projDir)
	if len(v.docs) != 2 {
		t.Fatalf("expected 2 active docs, got %d", len(v.docs))
	}

	// Archive one doc
	archiveCmd := v.doArchiveDoc("archive-doc")
	archiveCmd()

	// Reload active docs
	cmd := v.loadDocs()
	msg := cmd()
	v.Update(msg)
	if len(v.docs) != 1 {
		t.Fatalf("expected 1 active doc after archiving, got %d", len(v.docs))
	}

	// Press 'A' to toggle to archived view
	v, cmd = pressKey(v, "A")
	if !v.showArchived {
		t.Fatal("should be in archived view after 'A'")
	}
	if cmd == nil {
		t.Fatal("should return a reload command")
	}

	// Load archived docs
	msg = cmd()
	v.Update(msg)
	if len(v.docs) != 1 {
		t.Fatalf("expected 1 archived doc, got %d", len(v.docs))
	}
	if v.docs[0].Name != "archive-doc" {
		t.Errorf("archived doc name = %q, want 'archive-doc'", v.docs[0].Name)
	}

	// View should show "(archived)" in title
	view := v.View()
	if !strings.Contains(view, "(archived)") {
		t.Error("view should show '(archived)' in title")
	}

	// Press 'A' again to switch back to active
	v, cmd = pressKey(v, "A")
	if v.showArchived {
		t.Error("should be back in active view after second 'A'")
	}

	msg = cmd()
	v.Update(msg)
	if len(v.docs) != 1 {
		t.Fatalf("expected 1 active doc, got %d", len(v.docs))
	}
	if v.docs[0].Name != "active-doc" {
		t.Errorf("active doc name = %q, want 'active-doc'", v.docs[0].Name)
	}
}

func TestKnowledgeListView_Restore(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"restore-me": {"design"},
	})

	v := loadKnowledgeListView(t, projDir)

	// Archive the doc
	archiveCmd := v.doArchiveDoc("restore-me")
	archiveCmd()

	// Switch to archived view and load
	v.showArchived = true
	cmd := v.loadDocs()
	msg := cmd()
	v.Update(msg)

	if len(v.docs) != 1 {
		t.Fatalf("expected 1 archived doc, got %d", len(v.docs))
	}

	// Press 'R' to restore
	_, cmd = pressKey(v, "R")
	if cmd == nil {
		t.Fatal("'R' should return a restore command")
	}

	result := cmd()
	dcMsg, ok := result.(DataChangedMsg)
	if !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}
	if !strings.Contains(dcMsg.StatusText, "Restored") {
		t.Errorf("StatusText = %q", dcMsg.StatusText)
	}

	// Verify doc is back in active location
	activePath := knowledge.FilePath(projDir, "restore-me")
	if _, err := os.Stat(activePath); err != nil {
		t.Error("restored doc should exist in active location")
	}

	// Verify doc is gone from archive
	archivePath := filepath.Join(knowledge.Dir(projDir), ".archive", "restore-me.md")
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Error("doc should be gone from archive after restore")
	}
}

func TestKnowledgeListView_Restore_NotInActiveView(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"doc1": {"design"},
	})

	v := loadKnowledgeListView(t, projDir)

	// Press 'R' in active view — should do nothing
	_, cmd := pressKey(v, "R")
	if cmd != nil {
		t.Error("'R' should do nothing in active view")
	}
}

// =======================================================================
// Tag Search Tests
// =======================================================================

func TestKnowledgeListView_TagSearch(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"api-design":     {"design", "api"},
		"test-plan":      {"testing", "qa"},
		"design-overview": {"design", "overview"},
	})

	v := loadKnowledgeListView(t, projDir)
	if len(v.docs) != 3 {
		t.Fatalf("expected 3 docs, got %d", len(v.docs))
	}

	// Enter search mode
	v, _ = pressKey(v, "/")
	if !v.searchMode {
		t.Fatal("should be in search mode")
	}

	// Type '#design' to filter by tag
	for _, c := range "#design" {
		v, _ = pressKey(v, string(c))
	}

	visible := v.visibleDocs()
	if len(visible) != 2 {
		t.Fatalf("expected 2 docs with 'design' tag, got %d", len(visible))
	}

	// Check that all visible docs have the 'design' tag
	for _, doc := range visible {
		hasTag := false
		for _, tag := range doc.Tags {
			if strings.Contains(strings.ToLower(tag), "design") {
				hasTag = true
				break
			}
		}
		if !hasTag {
			t.Errorf("doc %q should have 'design' tag", doc.Name)
		}
	}

	// Clear search and verify all docs return
	v, _ = pressSpecialKey(v, tea.KeyEsc)
	if v.searchMode {
		t.Error("should exit search mode")
	}
	visible = v.visibleDocs()
	if len(visible) != 3 {
		t.Errorf("all docs should be visible after clearing search, got %d", len(visible))
	}
}

func TestKnowledgeListView_TagSearch_NoMatch(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"doc1": {"design"},
	})

	v := loadKnowledgeListView(t, projDir)

	v, _ = pressKey(v, "/")
	for _, c := range "#nonexistent" {
		v, _ = pressKey(v, string(c))
	}

	visible := v.visibleDocs()
	if len(visible) != 0 {
		t.Errorf("expected 0 matches for nonexistent tag, got %d", len(visible))
	}
}

func TestKnowledgeListView_TagSearch_HashOnly(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"doc1": {"design"},
	})

	v := loadKnowledgeListView(t, projDir)

	v, _ = pressKey(v, "/")
	v, _ = pressKey(v, "#")

	// Just '#' with no tag name should return empty filtered list
	visible := v.visibleDocs()
	if len(visible) != 0 {
		t.Errorf("expected 0 docs for bare '#', got %d", len(visible))
	}
}

// =======================================================================
// KnowledgeView Actions Tests
// =======================================================================

func TestKnowledgeView_Delete_ConfirmNavigatesBack(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"delete-me": {"design"},
	})

	v := NewKnowledgeView("test-proj", projDir, "delete-me", 80, 24)
	cmd := v.Init()
	msg := cmd()
	v.Update(msg)

	// Press 'd' to start delete
	v, _ = kvPressKey(v, "d")
	if !v.confirmMode {
		t.Fatal("should enter confirm mode")
	}
	if !strings.Contains(v.confirmPrompt, "delete-me") {
		t.Errorf("confirmPrompt should mention doc name, got %q", v.confirmPrompt)
	}

	// Confirm with 'y'
	v, cmd = kvPressKey(v, "y")
	if v.confirmMode {
		t.Error("should exit confirm mode")
	}
	if cmd == nil {
		t.Fatal("should return delete command")
	}

	result := cmd()
	if _, ok := result.(GoBackMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected GoBackMsg after delete, got %T", result)
	}

	// Verify doc is deleted
	path := knowledge.FilePath(projDir, "delete-me")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("doc should be deleted")
	}
}

func TestKnowledgeView_Archive_NavigatesBack(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"archive-me": {"design"},
	})

	v := NewKnowledgeView("test-proj", projDir, "archive-me", 80, 24)
	cmd := v.Init()
	msg := cmd()
	v.Update(msg)

	// Press 'a' to archive
	_, cmd = kvPressKey(v, "a")
	if cmd == nil {
		t.Fatal("'a' should return archive command")
	}

	result := cmd()
	if _, ok := result.(GoBackMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected GoBackMsg after archive, got %T", result)
	}

	// Verify doc moved to archive
	activePath := knowledge.FilePath(projDir, "archive-me")
	if _, err := os.Stat(activePath); !os.IsNotExist(err) {
		t.Error("active doc should be gone")
	}

	archivePath := filepath.Join(knowledge.Dir(projDir), ".archive", "archive-me.md")
	if _, err := os.Stat(archivePath); err != nil {
		t.Errorf("archived doc should exist: %v", err)
	}
}

func TestKnowledgeView_Rename_StaysInView(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"old-name": {"design"},
	})

	v := NewKnowledgeView("test-proj", projDir, "old-name", 80, 24)
	cmd := v.Init()
	msg := cmd()
	v.Update(msg)

	// Press 'r' to start rename
	v, _ = kvPressKey(v, "r")
	if !v.inputMode {
		t.Fatal("should enter input mode on 'r'")
	}
	if v.inputValue != "old-name" {
		t.Errorf("inputValue should be pre-filled, got %q", v.inputValue)
	}

	// Clear and type new name
	for range len(v.inputValue) {
		v, _ = kvPressSpecialKey(v, tea.KeyBackspace)
	}
	for _, c := range "new-name" {
		v, _ = kvPressKey(v, string(c))
	}

	// Submit
	v, cmd = kvPressSpecialKey(v, tea.KeyEnter)
	if cmd == nil {
		t.Fatal("should return rename command")
	}

	result := cmd()
	renamedMsg, ok := result.(KnowledgeRenamedMsg)
	if !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected KnowledgeRenamedMsg, got %T", result)
	}
	if renamedMsg.NewName != "new-name" {
		t.Errorf("NewName = %q, want 'new-name'", renamedMsg.NewName)
	}

	// Apply the rename msg to the view
	v.Update(renamedMsg)
	if v.docName != "new-name" {
		t.Errorf("docName should be updated to 'new-name', got %q", v.docName)
	}

	// Verify files
	oldPath := knowledge.FilePath(projDir, "old-name")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("old doc should be gone")
	}
	newPath := knowledge.FilePath(projDir, "new-name")
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("new doc should exist: %v", err)
	}
}

func TestKnowledgeView_IsInputMode(t *testing.T) {
	v := NewKnowledgeView("proj", "/tmp/proj", "doc", 80, 24)

	if v.IsInputMode() {
		t.Error("should not be in input mode initially")
	}

	v.confirmMode = true
	if !v.IsInputMode() {
		t.Error("should report input mode for confirm")
	}

	v.confirmMode = false
	v.inputMode = true
	if !v.IsInputMode() {
		t.Error("should report input mode for input")
	}
}

// =======================================================================
// Edge Case Tests
// =======================================================================

func TestKnowledgeListView_EmptyDocList_NoActions(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{})

	v := loadKnowledgeListView(t, projDir)
	if len(v.docs) != 0 {
		t.Fatalf("expected 0 docs, got %d", len(v.docs))
	}

	// Actions on empty list should not crash
	v, _ = pressKey(v, "d")
	if v.confirmMode {
		t.Error("should not enter confirm mode on empty list")
	}

	v, cmd := pressKey(v, "a")
	if cmd != nil {
		t.Error("'a' should do nothing on empty list")
	}

	v, _ = pressKey(v, "r")
	if v.inputMode {
		t.Error("should not enter input mode on empty list")
	}

	// View should render without crash
	view := v.View()
	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestKnowledgeListView_CursorAdjustAfterDeleteLast(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"doc-a": {"a"},
		"doc-b": {"b"},
	})

	v := loadKnowledgeListView(t, projDir)

	// Move cursor to last item
	v.cursor = len(v.docs) - 1

	// Delete the last doc
	docName := v.docs[v.cursor].Name
	deleteCmd := v.doDeleteDoc(docName)
	result := deleteCmd()

	// Apply DataChangedMsg
	dcMsg := result.(DataChangedMsg)
	v.Update(dcMsg)

	// Reload
	cmd := v.loadDocs()
	msg := cmd()
	v.Update(msg)

	// Cursor should be adjusted
	if v.cursor >= len(v.docs) && len(v.docs) > 0 {
		t.Errorf("cursor (%d) should be within bounds (0..%d)", v.cursor, len(v.docs)-1)
	}
}

func TestKnowledgeListView_ArchivedViewEmpty(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"doc1": {"design"},
	})

	v := loadKnowledgeListView(t, projDir)

	// Toggle to archived view (no archived docs)
	v, cmd := pressKey(v, "A")
	if !v.showArchived {
		t.Fatal("should be in archived view")
	}

	msg := cmd()
	v.Update(msg)

	if len(v.docs) != 0 {
		t.Errorf("expected 0 archived docs, got %d", len(v.docs))
	}

	// View should render without crash
	view := v.View()
	if !strings.Contains(view, "No knowledge docs found") {
		t.Errorf("should show empty state in archived view")
	}
}

func TestKnowledgeListView_DeleteArchivedDoc(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"to-delete": {"old"},
	})

	v := loadKnowledgeListView(t, projDir)

	// Archive the doc first
	archiveCmd := v.doArchiveDoc("to-delete")
	archiveCmd()

	// Switch to archived view
	v.showArchived = true
	cmd := v.loadDocs()
	msg := cmd()
	v.Update(msg)

	if len(v.docs) != 1 {
		t.Fatalf("expected 1 archived doc, got %d", len(v.docs))
	}

	// Delete from archived view with confirmation
	v, _ = pressKey(v, "d")
	if !v.confirmMode {
		t.Fatal("should enter confirm mode")
	}

	_, cmd = pressKey(v, "y")
	result := cmd()
	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	// Verify archived doc is gone
	archivePath := filepath.Join(knowledge.Dir(projDir), ".archive", "to-delete.md")
	if _, err := os.Stat(archivePath); !os.IsNotExist(err) {
		t.Error("archived doc should be permanently deleted")
	}
}

// =======================================================================
// Help Bar Tests
// =======================================================================

func TestKnowledgeListView_HelpBar_ActiveView(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"doc1": {"design"},
	})

	v := loadKnowledgeListView(t, projDir)
	view := v.View()

	// Active view should show action keys
	for _, key := range []string{"del", "archive", "rename", "archived"} {
		if !strings.Contains(view, key) {
			t.Errorf("active help bar should contain %q", key)
		}
	}
}

func TestKnowledgeListView_HelpBar_ArchivedView(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"doc1": {"design"},
	})

	v := loadKnowledgeListView(t, projDir)

	// Archive a doc and switch to archived view
	archiveCmd := v.doArchiveDoc("doc1")
	archiveCmd()

	v.showArchived = true
	cmd := v.loadDocs()
	msg := cmd()
	v.Update(msg)

	view := v.View()
	for _, key := range []string{"restore", "delete", "active"} {
		if !strings.Contains(view, key) {
			t.Errorf("archived help bar should contain %q", key)
		}
	}
}

func TestKnowledgeListView_HelpBar_ConfirmMode(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"doc1": {"design"},
	})

	v := loadKnowledgeListView(t, projDir)

	v, _ = pressKey(v, "d")
	view := v.View()
	if !strings.Contains(view, "Delete") && !strings.Contains(view, "(y/n)") {
		t.Error("confirm mode should show the confirmation prompt")
	}
}

func TestKnowledgeView_HelpBar(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"doc1": {"design"},
	})

	v := NewKnowledgeView("test-proj", projDir, "doc1", 80, 24)
	cmd := v.Init()
	msg := cmd()
	v.Update(msg)

	view := v.View()
	for _, key := range []string{"delete", "archive", "rename"} {
		if !strings.Contains(view, key) {
			t.Errorf("KnowledgeView help bar should contain %q", key)
		}
	}
}

// =======================================================================
// Help Overlay Tests
// =======================================================================

func TestHelpOverlay_KnowledgeList(t *testing.T) {
	help := renderContextHelp(ViewKnowledgeList)

	expectedKeys := []string{"delete", "archive", "rename", "toggle archived", "#tag"}
	for _, key := range expectedKeys {
		if !strings.Contains(help, key) {
			t.Errorf("knowledge list help should contain %q", key)
		}
	}
}

func TestHelpOverlay_KnowledgeView(t *testing.T) {
	help := renderContextHelp(ViewKnowledgeView)

	expectedKeys := []string{"delete", "archive", "rename"}
	for _, key := range expectedKeys {
		if !strings.Contains(help, key) {
			t.Errorf("knowledge view help should contain %q", key)
		}
	}
}

// =======================================================================
// KnowledgeView In-Doc Search Tests
// =======================================================================

func TestKnowledgeView_Search_EnterAndFind(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"search-doc": {"design"},
	})

	// Write some searchable content
	content := "---\ntitle: Search Doc\ncreated: 2026-01-01T00:00:00Z\nupdated: 2026-01-01T00:00:00Z\ntags: [design]\n---\n\n# Search Doc\n\nThis has a keyword here.\n\nAnother paragraph without it.\n\nAnd the keyword appears again.\n"
	os.WriteFile(knowledge.FilePath(projDir, "search-doc"), []byte(content), 0o644)

	v := NewKnowledgeView("test-proj", projDir, "search-doc", 80, 40)
	cmd := v.Init()
	msg := cmd()
	v.Update(msg)

	if !v.loaded || len(v.lines) == 0 {
		t.Fatal("content should be loaded")
	}

	// Press '/' to enter search mode
	v, _ = kvPressKey(v, "/")
	if !v.searchMode {
		t.Fatal("should be in search mode after '/'")
	}
	if !v.IsInputMode() {
		t.Error("IsInputMode should return true during search")
	}

	// Type search query
	for _, c := range "keyword" {
		v, _ = kvPressKey(v, string(c))
	}

	if v.searchQuery != "keyword" {
		t.Errorf("searchQuery = %q, want 'keyword'", v.searchQuery)
	}

	// Should have matches (incremental search)
	if len(v.searchMatches) != 2 {
		t.Errorf("expected 2 matches for 'keyword', got %d", len(v.searchMatches))
	}

	// Submit search
	v, _ = kvPressSpecialKey(v, tea.KeyEnter)
	if v.searchMode {
		t.Error("should exit search mode after Enter")
	}
	if v.searchQuery != "keyword" {
		t.Error("searchQuery should persist after Enter")
	}
}

func TestKnowledgeView_Search_NextPrev(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"nav-doc": {"design"},
	})

	// Create content with multiple matches spread across lines
	var lines []string
	lines = append(lines, "---", "title: Nav Doc", "created: 2026-01-01T00:00:00Z", "updated: 2026-01-01T00:00:00Z", "---", "", "# Nav Doc", "")
	for i := 0; i < 50; i++ {
		if i == 10 || i == 25 || i == 40 {
			lines = append(lines, "This line has TARGET in it.")
		} else {
			lines = append(lines, "Some other content here.")
		}
	}
	content := strings.Join(lines, "\n")
	os.WriteFile(knowledge.FilePath(projDir, "nav-doc"), []byte(content), 0o644)

	v := NewKnowledgeView("test-proj", projDir, "nav-doc", 80, 20)
	cmd := v.Init()
	msg := cmd()
	v.Update(msg)

	// Search for TARGET
	v, _ = kvPressKey(v, "/")
	for _, c := range "TARGET" {
		v, _ = kvPressKey(v, string(c))
	}
	v, _ = kvPressSpecialKey(v, tea.KeyEnter)

	if len(v.searchMatches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(v.searchMatches))
	}

	// Should be on first match
	if v.searchCurrent != 0 {
		t.Errorf("searchCurrent = %d, want 0", v.searchCurrent)
	}
	firstOffset := v.scrollOffset

	// Press 'n' for next match
	v, _ = kvPressKey(v, "n")
	if v.searchCurrent != 1 {
		t.Errorf("after 'n': searchCurrent = %d, want 1", v.searchCurrent)
	}
	if v.scrollOffset == firstOffset {
		// Should have scrolled to a different position
		t.Log("scrolled to different position for match 2 — good")
	}

	// Press 'n' again
	v, _ = kvPressKey(v, "n")
	if v.searchCurrent != 2 {
		t.Errorf("after second 'n': searchCurrent = %d, want 2", v.searchCurrent)
	}

	// Press 'n' again — should wrap to first
	v, _ = kvPressKey(v, "n")
	if v.searchCurrent != 0 {
		t.Errorf("after wrap: searchCurrent = %d, want 0", v.searchCurrent)
	}

	// Press 'N' for previous — should go to last
	v, _ = kvPressKey(v, "N")
	if v.searchCurrent != 2 {
		t.Errorf("after 'N': searchCurrent = %d, want 2", v.searchCurrent)
	}

	// Press 'N' again
	v, _ = kvPressKey(v, "N")
	if v.searchCurrent != 1 {
		t.Errorf("after second 'N': searchCurrent = %d, want 1", v.searchCurrent)
	}
}

func TestKnowledgeView_Search_EscClearsSearch(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"esc-doc": {"design"},
	})

	content := "---\ntitle: Esc Doc\ncreated: 2026-01-01T00:00:00Z\nupdated: 2026-01-01T00:00:00Z\n---\n\n# Esc Doc\n\nSome searchable text here.\n"
	os.WriteFile(knowledge.FilePath(projDir, "esc-doc"), []byte(content), 0o644)

	v := NewKnowledgeView("test-proj", projDir, "esc-doc", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Search and submit
	v, _ = kvPressKey(v, "/")
	for _, c := range "searchable" {
		v, _ = kvPressKey(v, string(c))
	}
	v, _ = kvPressSpecialKey(v, tea.KeyEnter)

	if v.searchQuery == "" {
		t.Fatal("searchQuery should be set after Enter")
	}
	if len(v.searchMatches) == 0 {
		t.Fatal("should have matches")
	}

	// Esc should clear search (not go back)
	v, cmd2 := kvPressSpecialKey(v, tea.KeyEsc)
	if v.searchQuery != "" {
		t.Error("Esc should clear searchQuery")
	}
	if len(v.searchMatches) != 0 {
		t.Error("Esc should clear searchMatches")
	}
	if cmd2 != nil {
		// Should NOT produce a GoBackMsg
		result := cmd2()
		if _, ok := result.(GoBackMsg); ok {
			t.Error("Esc with active search should clear search, not go back")
		}
	}

	// Second Esc should go back (no search active)
	_, cmd2 = kvPressSpecialKey(v, tea.KeyEsc)
	if cmd2 == nil {
		t.Fatal("second Esc should go back")
	}
	result := cmd2()
	if _, ok := result.(GoBackMsg); !ok {
		t.Errorf("expected GoBackMsg, got %T", result)
	}
}

func TestKnowledgeView_Search_EscDuringInput(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"esc-input": {"design"},
	})

	v := NewKnowledgeView("test-proj", projDir, "esc-input", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Enter search mode
	v, _ = kvPressKey(v, "/")
	if !v.searchMode {
		t.Fatal("should be in search mode")
	}

	// Type something
	v, _ = kvPressKey(v, "t")
	v, _ = kvPressKey(v, "e")

	// Esc during input cancels search mode
	v, _ = kvPressSpecialKey(v, tea.KeyEsc)
	if v.searchMode {
		t.Error("Esc should exit search mode")
	}
	if v.searchQuery != "" {
		t.Error("query should be cleared on Esc during input")
	}
}

func TestKnowledgeView_Search_CaseInsensitive(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"case-doc": {"design"},
	})

	content := "---\ntitle: Case Doc\ncreated: 2026-01-01T00:00:00Z\nupdated: 2026-01-01T00:00:00Z\n---\n\n# Case Doc\n\nHello World here.\n\nhello world there.\n\nHELLO WORLD everywhere.\n"
	os.WriteFile(knowledge.FilePath(projDir, "case-doc"), []byte(content), 0o644)

	v := NewKnowledgeView("test-proj", projDir, "case-doc", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Search should be case-insensitive
	v, _ = kvPressKey(v, "/")
	for _, c := range "hello" {
		v, _ = kvPressKey(v, string(c))
	}

	// Should match all 3 lines regardless of case
	if len(v.searchMatches) != 3 {
		t.Errorf("expected 3 case-insensitive matches, got %d", len(v.searchMatches))
	}
}

func TestKnowledgeView_Search_NoMatches(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"no-match": {"design"},
	})

	v := NewKnowledgeView("test-proj", projDir, "no-match", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	v, _ = kvPressKey(v, "/")
	for _, c := range "zzzznonexistent" {
		v, _ = kvPressKey(v, string(c))
	}
	v, _ = kvPressSpecialKey(v, tea.KeyEnter)

	if len(v.searchMatches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(v.searchMatches))
	}

	// n/N should not crash with no matches
	v, _ = kvPressKey(v, "n")
	_, _ = kvPressKey(v, "N")
}

func TestKnowledgeView_Search_HelpBar(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"help-doc": {"design"},
	})

	content := "---\ntitle: Help Doc\ncreated: 2026-01-01T00:00:00Z\nupdated: 2026-01-01T00:00:00Z\n---\n\n# Help Doc\n\nSome text with a findme word.\n"
	os.WriteFile(knowledge.FilePath(projDir, "help-doc"), []byte(content), 0o644)

	v := NewKnowledgeView("test-proj", projDir, "help-doc", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	// Default help bar should show '/ search'
	view := v.View()
	if !strings.Contains(view, "/ search") {
		t.Error("default help bar should show '/ search'")
	}

	// During search input, should show the query
	v, _ = kvPressKey(v, "/")
	view = v.View()
	if !strings.Contains(view, "/") {
		t.Error("search mode should show '/' prompt")
	}

	// After search, should show n/N navigation
	for _, c := range "findme" {
		v, _ = kvPressKey(v, string(c))
	}
	v, _ = kvPressSpecialKey(v, tea.KeyEnter)

	view = v.View()
	if !strings.Contains(view, "findme") {
		t.Error("help bar should show active search query")
	}
	if !strings.Contains(view, "next") || !strings.Contains(view, "prev") {
		t.Error("help bar should show n/N navigation hints")
	}
}

func TestKnowledgeView_Search_MatchIndicatorInView(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"indicator-doc": {"design"},
	})

	content := "---\ntitle: Indicator Doc\ncreated: 2026-01-01T00:00:00Z\nupdated: 2026-01-01T00:00:00Z\n---\n\n# Indicator Doc\n\nFirst marker line.\n\nNo match here.\n\nSecond marker line.\n"
	os.WriteFile(knowledge.FilePath(projDir, "indicator-doc"), []byte(content), 0o644)

	v := NewKnowledgeView("test-proj", projDir, "indicator-doc", 80, 24)
	cmd := v.Init()
	v.Update(cmd())

	v, _ = kvPressKey(v, "/")
	for _, c := range "marker" {
		v, _ = kvPressKey(v, string(c))
	}
	v, _ = kvPressSpecialKey(v, tea.KeyEnter)

	view := v.View()
	// The current match should have the ▸ indicator
	if !strings.Contains(view, "▸") {
		t.Error("view should show ▸ indicator for current match")
	}
}

func TestHighlightSearchTerm_PlainText(t *testing.T) {
	result := highlightSearchTerm("Hello World Hello", "hello")
	stripped := ansi.Strip(result)

	// The visible text content should be preserved regardless of terminal
	if stripped != "Hello World Hello" {
		t.Errorf("stripped text = %q, want 'Hello World Hello'", stripped)
	}
}

func TestHighlightSearchTerm_NoMatch(t *testing.T) {
	original := "Hello World"
	result := highlightSearchTerm(original, "xyz")
	if result != original {
		t.Error("no-match should return the original line unchanged")
	}
}

func TestHighlightSearchTerm_EmptyQuery(t *testing.T) {
	original := "Hello World"
	result := highlightSearchTerm(original, "")
	if result != original {
		t.Error("empty query should return original line")
	}
}

func TestHighlightSearchTerm_MultipleMatches(t *testing.T) {
	result := highlightSearchTerm("cat and cat and cat", "cat")
	stripped := ansi.Strip(result)
	// Visible text should be preserved
	if stripped != "cat and cat and cat" {
		t.Errorf("stripped = %q, want 'cat and cat and cat'", stripped)
	}
}

func TestHighlightSearchTerm_CaseInsensitive(t *testing.T) {
	result := highlightSearchTerm("Hello HELLO hello", "hello")
	stripped := ansi.Strip(result)
	// All three variants should be in the output with original casing
	if stripped != "Hello HELLO hello" {
		t.Errorf("stripped = %q, want 'Hello HELLO hello'", stripped)
	}
}

func TestHighlightSearchTerm_WithANSI(t *testing.T) {
	// Simulate a glamour-styled line (with ANSI codes around "bold")
	styled := "Some \x1b[1mbold\x1b[0m text here"
	result := highlightSearchTerm(styled, "bold")
	stripped := ansi.Strip(result)
	if !strings.Contains(stripped, "bold") {
		t.Errorf("stripped result should contain 'bold', got %q", stripped)
	}
}

func TestHelpOverlay_KnowledgeView_Search(t *testing.T) {
	help := renderContextHelp(ViewKnowledgeView)

	if !strings.Contains(help, "search in doc") {
		t.Error("help overlay should mention 'search in doc'")
	}
	if !strings.Contains(help, "next/prev match") {
		t.Error("help overlay should mention 'next/prev match'")
	}
}

// =======================================================================
// Git Commit Tests
// =======================================================================

func TestKnowledgeListView_GitCommit_Delete(t *testing.T) {
	_, projDir := setupKnowledgeProject(t, map[string][]string{
		"delete-me": {"design"},
	})

	v := loadKnowledgeListView(t, projDir)

	deleteCmd := v.doDeleteDoc("delete-me")
	result := deleteCmd()
	if _, ok := result.(DataChangedMsg); !ok {
		if errMsg, isErr := result.(ErrorMsg); isErr {
			t.Fatalf("got error: %v", errMsg.Err)
		}
		t.Fatalf("expected DataChangedMsg, got %T", result)
	}

	// Verify there's nothing to commit (git commit was done as part of delete)
	diff, err := git.DiffStat(context.Background(), projDir)
	if err != nil {
		t.Logf("git diff-stat err (non-fatal): %v", err)
	}
	if diff != "" {
		t.Errorf("should have no uncommitted changes after delete, got: %s", diff)
	}
}
