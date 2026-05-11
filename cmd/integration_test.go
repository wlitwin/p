package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/walter/p/internal/config"
	"github.com/walter/p/internal/display"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/validate"
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

func TestIntegrationStatusEmpty(t *testing.T) {
	root := setupIntegrationTest(t)

	// Create a project with no todos
	if err := project.Create(root, "empty-proj", "Empty project"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// List projects — the project should exist
	projects, err := project.List(root, false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}

	// Verify no todo lists exist
	dir := filepath.Join(root, "empty-proj")
	names, err := todo.ListNames(dir)
	if err != nil {
		t.Fatalf("ListNames: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0 todo lists, got %d", len(names))
	}
}

func TestIntegrationSearch(t *testing.T) {
	root := setupIntegrationTest(t)
	project.Create(root, "search-proj", "")
	dir := filepath.Join(root, "search-proj")

	// Add todo items with searchable text
	list, _ := todo.CreateList(dir, "tasks", "Tasks")
	todo.AddItem(list, "Implement authentication module", todo.Now, "")
	todo.AddItem(list, "Fix database connection pooling", todo.Backlog, "")
	todo.SaveList(dir, "tasks", list)

	// Add knowledge with searchable text
	knowledge.Create(dir, "design", "Design Doc", []string{"architecture"})
	knowledge.Append(dir, "design", "## Overview\n\nThis system uses microservices architecture.", "")

	// Search todos by rendering and checking content
	list, _ = todo.LoadList(dir, "tasks")
	foundAuth := false
	for _, item := range list.Items {
		if strings.Contains(strings.ToLower(item.Text), "authentication") {
			foundAuth = true
		}
	}
	if !foundAuth {
		t.Error("expected to find 'authentication' in todo items")
	}

	// Search knowledge
	content, _ := knowledge.Read(dir, "design")
	if !strings.Contains(strings.ToLower(content), "microservices") {
		t.Error("expected to find 'microservices' in knowledge content")
	}

	// Verify no matches for nonexistent text
	foundNonexistent := false
	for _, item := range list.Items {
		if strings.Contains(strings.ToLower(item.Text), "nonexistent-xyz-term") {
			foundNonexistent = true
		}
	}
	if foundNonexistent {
		t.Error("should not find nonexistent text in todos")
	}
	if strings.Contains(strings.ToLower(content), "nonexistent-xyz-term") {
		t.Error("should not find nonexistent text in knowledge")
	}
}

func TestIntegrationMove(t *testing.T) {
	root := setupIntegrationTest(t)
	project.Create(root, "move-proj", "")
	dir := filepath.Join(root, "move-proj")

	// Create two lists
	srcList, _ := todo.CreateList(dir, "backlog", "Backlog")
	todo.AddItem(srcList, "Task to move", todo.Now, "")
	todo.AddItem(srcList, "Task to keep", todo.Backlog, "")
	todo.SaveList(dir, "backlog", srcList)

	dstList, _ := todo.CreateList(dir, "sprint", "Sprint")
	todo.AddItem(dstList, "Existing task", todo.Now, "")
	todo.SaveList(dir, "sprint", dstList)

	// Reload and simulate move using DeepCopyItem + RemoveItem
	srcList, _ = todo.LoadList(dir, "backlog")
	dstList, _ = todo.LoadList(dir, "sprint")

	item, _ := todo.ResolveItem(srcList, "1")
	itemCopy := todo.DeepCopyItem(item)

	dstList.Items = append(dstList.Items, itemCopy)
	todo.SaveList(dir, "sprint", dstList)

	todo.RemoveItem(srcList, "1")
	todo.SaveList(dir, "backlog", srcList)

	// Verify source list has item removed
	srcList, _ = todo.LoadList(dir, "backlog")
	if len(srcList.Items) != 1 {
		t.Errorf("source list: expected 1 item, got %d", len(srcList.Items))
	}
	if srcList.Items[0].Text != "Task to keep" {
		t.Errorf("source list: wrong remaining item: %q", srcList.Items[0].Text)
	}

	// Verify destination list has item added
	dstList, _ = todo.LoadList(dir, "sprint")
	if len(dstList.Items) != 2 {
		t.Errorf("dest list: expected 2 items, got %d", len(dstList.Items))
	}
	found := false
	for _, it := range dstList.Items {
		if it.Text == "Task to move" {
			found = true
		}
	}
	if !found {
		t.Error("moved item not found in destination list")
	}
}

func TestIntegrationArchiveUnarchive(t *testing.T) {
	root := setupIntegrationTest(t)
	project.Create(root, "archivable", "Test archive")
	dir := filepath.Join(root, "archivable")

	// Archive the project
	meta, _ := project.LoadMeta(dir)
	meta.Archived = true
	project.SaveMeta(dir, meta)

	// Verify excluded from non-archived list
	projects, _ := project.List(root, false)
	if len(projects) != 0 {
		t.Error("archived project should be excluded from List(false)")
	}

	// Unarchive the project
	meta.Archived = false
	project.SaveMeta(dir, meta)

	// Verify included again
	projects, _ = project.List(root, false)
	if len(projects) != 1 {
		t.Error("unarchived project should be included in List(false)")
	}
	if projects[0].Name != "archivable" {
		t.Errorf("expected project name 'archivable', got %q", projects[0].Name)
	}
}

func TestIntegrationRmList(t *testing.T) {
	root := setupIntegrationTest(t)
	project.Create(root, "rmlist-proj", "")
	dir := filepath.Join(root, "rmlist-proj")

	// Create a todo list
	todo.CreateList(dir, "ephemeral", "Ephemeral List")
	todo.CreateList(dir, "keeper", "Keeper List")

	names, _ := todo.ListNames(dir)
	if len(names) != 2 {
		t.Fatalf("expected 2 lists, got %d", len(names))
	}

	// Delete the list file directly
	listPath := todo.ListPath(dir, "ephemeral")
	if err := os.Remove(listPath); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Verify ListNames no longer includes the deleted list
	names, _ = todo.ListNames(dir)
	if len(names) != 1 {
		t.Errorf("expected 1 list after deletion, got %d", len(names))
	}
	for _, n := range names {
		if n == "ephemeral" {
			t.Error("deleted list 'ephemeral' should not appear in ListNames")
		}
	}
}

func TestIntegrationKnowledgeSubcommands(t *testing.T) {
	root := setupIntegrationTest(t)
	project.Create(root, "kdoc-proj", "")
	dir := filepath.Join(root, "kdoc-proj")

	// Create knowledge doc with tags
	knowledge.Create(dir, "setup-guide", "Setup Guide", []string{"onboarding", "devops"})

	// List files, verify present
	files, _ := knowledge.ListFiles(dir)
	if len(files) != 1 || files[0] != "setup-guide" {
		t.Errorf("expected ['setup-guide'], got %v", files)
	}

	// Read and search content
	content, err := knowledge.Read(dir, "setup-guide")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !strings.Contains(content, "Setup Guide") {
		t.Error("expected to find 'Setup Guide' in content")
	}

	// Verify tags
	tags := knowledge.ExtractTags(content)
	if len(tags) != 2 || tags[0] != "onboarding" || tags[1] != "devops" {
		t.Errorf("expected tags [onboarding, devops], got %v", tags)
	}

	// Delete doc, verify gone
	knowledge.Delete(dir, "setup-guide")
	files, _ = knowledge.ListFiles(dir)
	if len(files) != 0 {
		t.Error("knowledge doc should be deleted")
	}
}

func TestIntegrationFilterItems(t *testing.T) {
	// Build items with different states and priorities
	items := []*todo.Item{
		{Text: "Open now task", State: todo.Open, Priority: todo.Now},
		{Text: "Done task", State: todo.Done, Priority: todo.Now},
		{Text: "Blocked task", State: todo.Blocked, Priority: todo.Backlog},
		{Text: "Open backlog task", State: todo.Open, Priority: todo.Backlog, Tags: []string{"bug"}},
	}

	// Filter by state: open
	filtered := display.FilterItems(items, "open", "", "")
	if len(filtered) != 2 {
		t.Errorf("filter state=open: expected 2 items, got %d", len(filtered))
	}
	for _, item := range filtered {
		if item.State != todo.Open {
			t.Errorf("filter state=open: got item with state %q", item.State)
		}
	}

	// Filter by priority: backlog
	filtered = display.FilterItems(items, "", "backlog", "")
	if len(filtered) != 2 {
		t.Errorf("filter priority=backlog: expected 2 items, got %d", len(filtered))
	}
	for _, item := range filtered {
		if item.Priority != todo.Backlog {
			t.Errorf("filter priority=backlog: got item with priority %q", item.Priority)
		}
	}

	// Filter by tag: bug
	filtered = display.FilterItems(items, "", "", "bug")
	if len(filtered) != 1 {
		t.Errorf("filter tag=bug: expected 1 item, got %d", len(filtered))
	}
	if len(filtered) > 0 && filtered[0].Text != "Open backlog task" {
		t.Errorf("filter tag=bug: wrong item %q", filtered[0].Text)
	}

	// No filter — all items returned
	filtered = display.FilterItems(items, "", "", "")
	if len(filtered) != 4 {
		t.Errorf("no filter: expected 4 items, got %d", len(filtered))
	}
}

func TestIntegrationCountStates(t *testing.T) {
	items := []*todo.Item{
		{Text: "Open 1", State: todo.Open},
		{Text: "Done 1", State: todo.Done},
		{Text: "Blocked 1", State: todo.Blocked, Children: []*todo.Item{
			{Text: "Child open", State: todo.Open},
			{Text: "Child done", State: todo.Done},
		}},
		{Text: "Open 2", State: todo.Open},
	}

	open, done, blocked := todo.CountStates(items)
	if open != 3 {
		t.Errorf("open: expected 3, got %d", open)
	}
	if done != 2 {
		t.Errorf("done: expected 2, got %d", done)
	}
	if blocked != 1 {
		t.Errorf("blocked: expected 1, got %d", blocked)
	}
}

func TestIntegrationHasTag(t *testing.T) {
	item := &todo.Item{
		Text: "Tagged item",
		Tags: []string{"bug", "frontend"},
	}

	if !display.HasTag(item, "bug") {
		t.Error("hasTag should return true for 'bug'")
	}
	if !display.HasTag(item, "frontend") {
		t.Error("hasTag should return true for 'frontend'")
	}
	if display.HasTag(item, "backend") {
		t.Error("hasTag should return false for 'backend'")
	}
}

func TestIntegrationConfigGetSet(t *testing.T) {
	setupIntegrationTest(t)

	// getConfigValue for valid keys
	val, err := getConfigValue("project_root")
	if err != nil {
		t.Errorf("getConfigValue project_root: %v", err)
	}
	if val != cfg.ProjectRoot {
		t.Errorf("project_root: expected %q, got %q", cfg.ProjectRoot, val)
	}

	val, err = getConfigValue("claude_model")
	if err != nil {
		t.Errorf("getConfigValue claude_model: %v", err)
	}
	if val != "claude-opus-4-6" {
		t.Errorf("claude_model: expected 'claude-opus-4-6', got %q", val)
	}

	// getConfigValue for invalid key
	_, err = getConfigValue("invalid_key")
	if err == nil {
		t.Error("getConfigValue should return error for invalid key")
	}

	// setConfigKey — redirect config to temp dir to avoid polluting real config
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err = setConfigKey("claude_model", "test-model")
	if err != nil {
		t.Errorf("setConfigKey claude_model: %v", err)
	}
	if cfg.ClaudeModel != "test-model" {
		t.Errorf("after set, claude_model = %q", cfg.ClaudeModel)
	}

	// setConfigKey with invalid key
	err = setConfigKey("nonexistent_key", "value")
	if err == nil {
		t.Error("setConfigKey should return error for invalid key")
	}
}

func TestIntegrationInputValidation(t *testing.T) {
	// ProjectName validation
	if err := validate.ProjectName("valid-name"); err != nil {
		t.Errorf("valid project name rejected: %v", err)
	}
	if err := validate.ProjectName("also_valid123"); err != nil {
		t.Errorf("valid project name rejected: %v", err)
	}
	if err := validate.ProjectName(""); err == nil {
		t.Error("empty project name should be rejected")
	}
	if err := validate.ProjectName("has spaces"); err == nil {
		t.Error("project name with spaces should be rejected")
	}
	if err := validate.ProjectName("special!chars"); err == nil {
		t.Error("project name with special chars should be rejected")
	}

	// ListName validation
	if err := validate.ListName("valid-list"); err != nil {
		t.Errorf("valid list name rejected: %v", err)
	}
	if err := validate.ListName(""); err == nil {
		t.Error("empty list name should be rejected")
	}
	if err := validate.ListName("bad name!"); err == nil {
		t.Error("list name with spaces/special chars should be rejected")
	}

	// Date validation
	if err := validate.Date("2026-05-10"); err != nil {
		t.Errorf("valid date rejected: %v", err)
	}
	if err := validate.Date(""); err != nil {
		t.Errorf("empty date should be valid: %v", err)
	}
	if err := validate.Date("05-10-2026"); err == nil {
		t.Error("invalid date format should be rejected")
	}
	if err := validate.Date("not-a-date"); err == nil {
		t.Error("non-date string should be rejected")
	}

	// Priority validation
	if err := validate.Priority("now"); err != nil {
		t.Errorf("valid priority 'now' rejected: %v", err)
	}
	if err := validate.Priority("backlog"); err != nil {
		t.Errorf("valid priority 'backlog' rejected: %v", err)
	}
	if err := validate.Priority("urgent"); err == nil {
		t.Error("invalid priority should be rejected")
	}
	if err := validate.Priority(""); err == nil {
		t.Error("empty priority should be rejected")
	}
}

func TestIntegrationLooksLikeURL(t *testing.T) {
	if !display.LooksLikeURL("http://example.com") {
		t.Error("http://example.com should look like a URL")
	}
	if !display.LooksLikeURL("https://example.com") {
		t.Error("https://example.com should look like a URL")
	}
	if display.LooksLikeURL("not a url") {
		t.Error("'not a url' should not look like a URL")
	}
	if display.LooksLikeURL("ftp://example.com") {
		t.Error("ftp://example.com should not look like a URL")
	}
}

func TestIntegrationTruncate(t *testing.T) {
	if result := display.Truncate("short", 10); result != "short" {
		t.Errorf("display.Truncate('short', 10) = %q, want 'short'", result)
	}

	result := display.Truncate("this is a long string", 10)
	if !strings.HasSuffix(result, "...") {
		t.Errorf("truncated string should end with '...', got %q", result)
	}
	// 10 runes + "..." = 13 runes total
	runes := []rune(result)
	if len(runes) != 13 {
		t.Errorf("truncated string should be 13 runes, got %d: %q", len(runes), result)
	}
}

func TestIntegrationContainsIgnoreCase(t *testing.T) {
	if !display.ContainsIgnoreCase("Hello World", "hello") {
		t.Error("containsIgnoreCase should find 'hello' in 'Hello World'")
	}
	if !display.ContainsIgnoreCase("Hello World", "WORLD") {
		t.Error("containsIgnoreCase should find 'WORLD' in 'Hello World'")
	}
	if display.ContainsIgnoreCase("Hello World", "missing") {
		t.Error("containsIgnoreCase should not find 'missing' in 'Hello World'")
	}
}

func TestIntegrationDimTextPreservingLinks(t *testing.T) {
	result := display.DimTextPreservingLinks("text [[link]] more")
	if !strings.Contains(result, "[[link]]") {
		t.Errorf("wiki link should be preserved in output, got %q", result)
	}
}

func TestIntegrationExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	result := expandHome("~/test")
	if !strings.HasPrefix(result, home) {
		t.Errorf("expandHome('~/test') = %q, should start with %q", result, home)
	}
	if !strings.HasSuffix(result, "test") {
		t.Errorf("expandHome('~/test') = %q, should end with 'test'", result)
	}

	result = expandHome("/absolute/path")
	if result != "/absolute/path" {
		t.Errorf("expandHome('/absolute/path') = %q, should be unchanged", result)
	}
}

func configureGitUser(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config user.email: %v", err)
	}
	cmd = exec.Command("git", "config", "user.name", "test")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config user.name: %v", err)
	}
}

func TestIntegrationLogAfterMutations(t *testing.T) {
	root := setupIntegrationTest(t)
	project.Create(root, "log-proj", "")
	dir := filepath.Join(root, "log-proj")

	// Init git and configure user
	if err := git.Init(dir); err != nil {
		t.Fatalf("git.Init: %v", err)
	}
	configureGitUser(t, dir)

	// Add a todo item, save, and commit
	list, err := todo.CreateList(dir, "tasks", "Tasks")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}
	todo.AddItem(list, "First task", todo.Now, "")
	todo.SaveList(dir, "tasks", list)
	if err := git.CommitAll(dir, "Add first task"); err != nil {
		t.Fatalf("CommitAll 1: %v", err)
	}

	// Add another item, save, and commit again
	list, _ = todo.LoadList(dir, "tasks")
	todo.AddItem(list, "Second task", todo.Backlog, "")
	todo.SaveList(dir, "tasks", list)
	if err := git.CommitAll(dir, "Add second task"); err != nil {
		t.Fatalf("CommitAll 2: %v", err)
	}

	// Run git log --oneline
	cmd := exec.Command("git", "log", "--oneline")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log: %v", err)
	}

	logOutput := string(out)
	if !strings.Contains(logOutput, "Add first task") {
		t.Errorf("git log should contain 'Add first task', got:\n%s", logOutput)
	}
	if !strings.Contains(logOutput, "Add second task") {
		t.Errorf("git log should contain 'Add second task', got:\n%s", logOutput)
	}

	lines := strings.Split(strings.TrimSpace(logOutput), "\n")
	if len(lines) < 2 {
		t.Errorf("expected at least 2 log lines, got %d:\n%s", len(lines), logOutput)
	}
}

func TestIntegrationDiffUncommitted(t *testing.T) {
	root := setupIntegrationTest(t)
	project.Create(root, "diff-proj", "")
	dir := filepath.Join(root, "diff-proj")

	// Init git and configure user
	if err := git.Init(dir); err != nil {
		t.Fatalf("git.Init: %v", err)
	}
	configureGitUser(t, dir)

	// Create initial commit with a todo list
	list, err := todo.CreateList(dir, "tasks", "Tasks")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}
	todo.AddItem(list, "Initial item", todo.Now, "")
	todo.SaveList(dir, "tasks", list)
	if err := git.CommitAll(dir, "Initial commit"); err != nil {
		t.Fatalf("CommitAll: %v", err)
	}

	// Modify the list (add a new item) but do NOT commit
	list, _ = todo.LoadList(dir, "tasks")
	todo.AddItem(list, "Uncommitted item", todo.Backlog, "")
	todo.SaveList(dir, "tasks", list)

	// git.Diff should return non-empty diff containing the new item text
	diff, err := git.Diff(dir)
	if err != nil {
		t.Fatalf("git.Diff: %v", err)
	}
	if diff == "" {
		t.Error("git.Diff should return non-empty diff for uncommitted changes")
	}
	if !strings.Contains(diff, "Uncommitted item") {
		t.Errorf("diff should contain 'Uncommitted item', got:\n%s", diff)
	}

	// git.DiffStat should return non-empty stat output
	stat, err := git.DiffStat(dir)
	if err != nil {
		t.Fatalf("git.DiffStat: %v", err)
	}
	if stat == "" {
		t.Error("git.DiffStat should return non-empty stat for uncommitted changes")
	}

	// Commit and verify Diff returns empty (clean)
	if err := git.CommitAll(dir, "Commit uncommitted item"); err != nil {
		t.Fatalf("CommitAll after modify: %v", err)
	}
	diff, err = git.Diff(dir)
	if err != nil {
		t.Fatalf("git.Diff after commit: %v", err)
	}
	if diff != "" {
		t.Errorf("git.Diff should return empty after commit, got:\n%s", diff)
	}
}

func TestIntegrationDiffClean(t *testing.T) {
	root := setupIntegrationTest(t)
	project.Create(root, "clean-proj", "")
	dir := filepath.Join(root, "clean-proj")

	// Init git and configure user
	if err := git.Init(dir); err != nil {
		t.Fatalf("git.Init: %v", err)
	}
	configureGitUser(t, dir)

	// Create and commit a file
	list, err := todo.CreateList(dir, "tasks", "Tasks")
	if err != nil {
		t.Fatalf("CreateList: %v", err)
	}
	todo.AddItem(list, "Committed item", todo.Now, "")
	todo.SaveList(dir, "tasks", list)
	if err := git.CommitAll(dir, "Initial commit"); err != nil {
		t.Fatalf("CommitAll: %v", err)
	}

	// DiffStat should return empty on clean working tree
	stat, err := git.DiffStat(dir)
	if err != nil {
		t.Fatalf("git.DiffStat: %v", err)
	}
	if stat != "" {
		t.Errorf("git.DiffStat should return empty on clean tree, got:\n%s", stat)
	}
}
