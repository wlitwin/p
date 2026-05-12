package todo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseRoundTrip(t *testing.T) {
	input := `---
title: DB Refactor
created: 2026-05-10T12:00:00Z
updated: 2026-05-10T15:00:00Z
---

# DB Refactor

- [ ] Audit current schema priority=now created=2026-05-10
- [ ] Validate optimistic locking priority=now due=2026-05-20 created=2026-05-10
  - [ ] Check conflict rate in logs priority=now created=2026-05-10
  - [ ] Talk to platform team priority=now created=2026-05-10
- [x] Set up migration framework priority=now created=2026-05-08 done=2026-05-08
- [-] Update ORM mappings priority=backlog created=2026-05-09
`

	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if list.Title != "DB Refactor" {
		t.Errorf("title = %q, want %q", list.Title, "DB Refactor")
	}

	if len(list.Items) != 4 {
		t.Fatalf("got %d items, want 4", len(list.Items))
	}

	tests := []struct {
		idx      int
		text     string
		state    State
		priority Priority
		due      string
		children int
	}{
		{0, "Audit current schema", Open, Now, "", 0},
		{1, "Validate optimistic locking", Open, Now, "2026-05-20", 2},
		{2, "Set up migration framework", Done, Now, "", 0},
		{3, "Update ORM mappings", Blocked, Backlog, "", 0},
	}

	for _, tt := range tests {
		item := list.Items[tt.idx]
		if item.Text != tt.text {
			t.Errorf("item[%d].Text = %q, want %q", tt.idx, item.Text, tt.text)
		}
		if item.State != tt.state {
			t.Errorf("item[%d].State = %q, want %q", tt.idx, item.State, tt.state)
		}
		if item.Priority != tt.priority {
			t.Errorf("item[%d].Priority = %q, want %q", tt.idx, item.Priority, tt.priority)
		}
		if item.Due != tt.due {
			t.Errorf("item[%d].Due = %q, want %q", tt.idx, item.Due, tt.due)
		}
		if len(item.Children) != tt.children {
			t.Errorf("item[%d].Children = %d, want %d", tt.idx, len(item.Children), tt.children)
		}
	}

	// Check children
	child0 := list.Items[1].Children[0]
	if child0.Text != "Check conflict rate in logs" {
		t.Errorf("child[0].Text = %q", child0.Text)
	}
}

func TestParseRenderPreservesContent(t *testing.T) {
	list := &List{
		Title: "Test List",
	}
	AddItem(list, "First task", Now, "")
	AddItem(list, "Second task", Backlog, "2026-06-01")

	rendered := Render(list)

	parsed, err := Parse(rendered)
	if err != nil {
		t.Fatalf("Parse(Render()) error: %v", err)
	}

	if parsed.Title != "Test List" {
		t.Errorf("title = %q, want %q", parsed.Title, "Test List")
	}
	if len(parsed.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(parsed.Items))
	}
	if parsed.Items[0].Text != "First task" {
		t.Errorf("item[0] = %q", parsed.Items[0].Text)
	}
	if parsed.Items[1].Priority != Backlog {
		t.Errorf("item[1].Priority = %q, want backlog", parsed.Items[1].Priority)
	}
	if parsed.Items[1].Due != "2026-06-01" {
		t.Errorf("item[1].Due = %q", parsed.Items[1].Due)
	}
}

func TestResolveItem(t *testing.T) {
	list := &List{Title: "Test"}
	item1 := AddItem(list, "Parent", Now, "")
	child := &Item{Text: "Child", State: Open, Priority: Now}
	item1.Children = append(item1.Children, child)
	AddItem(list, "Second", Now, "")

	tests := []struct {
		id   string
		want string
	}{
		{"1", "Parent"},
		{"1.1", "Child"},
		{"2", "Second"},
	}

	for _, tt := range tests {
		got, err := ResolveItem(list, tt.id)
		if err != nil {
			t.Errorf("ResolveItem(%q) error: %v", tt.id, err)
			continue
		}
		if got.Text != tt.want {
			t.Errorf("ResolveItem(%q) = %q, want %q", tt.id, got.Text, tt.want)
		}
	}

	_, err := ResolveItem(list, "5")
	if err == nil {
		t.Error("ResolveItem(5) should error")
	}
}

func TestRenderItemDisplay(t *testing.T) {
	list := &List{Title: "Test"}
	AddItem(list, "Task one", Now, "")
	item2 := AddItem(list, "Task two", Now, "2026-06-15")
	SetState(item2, Done)

	rendered := Render(list)

	if !strings.Contains(rendered, "- [ ] Task one") {
		t.Error("should contain open checkbox for task one")
	}
	if !strings.Contains(rendered, "- [x] Task two") {
		t.Error("should contain done checkbox for task two")
	}
	if !strings.Contains(rendered, "due=2026-06-15") {
		t.Error("should contain due date")
	}
}

// ---------------------------------------------------------------------------
// AddItem tests
// ---------------------------------------------------------------------------

func TestAddItem(t *testing.T) {
	list := &List{Title: "Test"}
	item := AddItem(list, "Write docs", Backlog, "2026-06-01")

	if item.Text != "Write docs" {
		t.Errorf("Text = %q, want %q", item.Text, "Write docs")
	}
	if item.State != Open {
		t.Errorf("State = %q, want %q", item.State, Open)
	}
	if item.Priority != Backlog {
		t.Errorf("Priority = %q, want %q", item.Priority, Backlog)
	}
	if item.Due != "2026-06-01" {
		t.Errorf("Due = %q, want %q", item.Due, "2026-06-01")
	}
	if len(list.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(list.Items))
	}
	if list.Items[0] != item {
		t.Error("item should be appended to list.Items")
	}
}

func TestAddItemDefaults(t *testing.T) {
	list := &List{Title: "Test"}
	item := AddItem(list, "Default task", Now, "")

	if item.State != Open {
		t.Errorf("State = %q, want %q", item.State, Open)
	}
	if item.Created == "" {
		t.Error("Created should be set automatically")
	}
	// Created should be today's date in YYYY-MM-DD format
	today := time.Now().UTC().Format("2006-01-02")
	if item.Created != today {
		t.Errorf("Created = %q, want %q", item.Created, today)
	}
	if item.Due != "" {
		t.Errorf("Due = %q, want empty", item.Due)
	}
	if item.DoneDate != "" {
		t.Errorf("DoneDate = %q, want empty", item.DoneDate)
	}
	if len(item.Tags) != 0 {
		t.Errorf("Tags = %v, want empty", item.Tags)
	}
	if len(item.Children) != 0 {
		t.Errorf("Children = %v, want empty", item.Children)
	}
}

// ---------------------------------------------------------------------------
// RemoveItem tests
// ---------------------------------------------------------------------------

func TestRemoveItemByID(t *testing.T) {
	list := &List{Title: "Test"}
	AddItem(list, "First", Now, "")
	AddItem(list, "Second", Now, "")
	AddItem(list, "Third", Now, "")

	if err := RemoveItem(list, "2"); err != nil {
		t.Fatalf("RemoveItem(2) error: %v", err)
	}

	if len(list.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(list.Items))
	}
	if list.Items[0].Text != "First" {
		t.Errorf("Items[0].Text = %q, want %q", list.Items[0].Text, "First")
	}
	if list.Items[1].Text != "Third" {
		t.Errorf("Items[1].Text = %q, want %q", list.Items[1].Text, "Third")
	}
}

func TestRemoveNestedItem(t *testing.T) {
	list := &List{Title: "Test"}
	parent := AddItem(list, "Parent", Now, "")
	parent.Children = []*Item{
		{Text: "Child A", State: Open, Priority: Now},
		{Text: "Child B", State: Open, Priority: Now},
		{Text: "Child C", State: Open, Priority: Now},
	}

	if err := RemoveItem(list, "1.2"); err != nil {
		t.Fatalf("RemoveItem(1.2) error: %v", err)
	}

	if len(parent.Children) != 2 {
		t.Fatalf("len(Children) = %d, want 2", len(parent.Children))
	}
	if parent.Children[0].Text != "Child A" {
		t.Errorf("Children[0].Text = %q, want %q", parent.Children[0].Text, "Child A")
	}
	if parent.Children[1].Text != "Child C" {
		t.Errorf("Children[1].Text = %q, want %q", parent.Children[1].Text, "Child C")
	}
	// Parent should still exist
	if len(list.Items) != 1 {
		t.Errorf("len(Items) = %d, want 1", len(list.Items))
	}
}

func TestRemoveItemInvalidID(t *testing.T) {
	list := &List{Title: "Test"}
	AddItem(list, "Only item", Now, "")

	tests := []string{"0", "99", "abc", "1.5", ""}
	for _, id := range tests {
		err := RemoveItem(list, id)
		if err == nil {
			t.Errorf("RemoveItem(%q) should error, got nil", id)
		}
	}
}

// ---------------------------------------------------------------------------
// SetState tests
// ---------------------------------------------------------------------------

func TestSetStateDone(t *testing.T) {
	item := &Item{Text: "Task", State: Open, Priority: Now}
	SetState(item, Done)

	if item.State != Done {
		t.Errorf("State = %q, want %q", item.State, Done)
	}
	if item.DoneDate == "" {
		t.Error("DoneDate should be set when state is Done")
	}
	today := time.Now().UTC().Format("2006-01-02")
	if item.DoneDate != today {
		t.Errorf("DoneDate = %q, want %q", item.DoneDate, today)
	}
}

func TestSetStateOpen(t *testing.T) {
	item := &Item{Text: "Task", State: Done, Priority: Now, DoneDate: "2026-05-09"}
	SetState(item, Open)

	if item.State != Open {
		t.Errorf("State = %q, want %q", item.State, Open)
	}
	if item.DoneDate != "" {
		t.Errorf("DoneDate = %q, want empty", item.DoneDate)
	}
}

func TestSetStateWithRecur(t *testing.T) {
	item := &Item{
		Text:     "Standup",
		State:    Open,
		Priority: Now,
		Due:      time.Now().UTC().Format("2006-01-02"),
		Recur:    "daily",
	}
	SetState(item, Done)

	// Recurring items stay open
	if item.State != Open {
		t.Errorf("State = %q, want %q (recurring should stay open)", item.State, Open)
	}
	// Due should advance by 1 day
	expected := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	if item.Due != expected {
		t.Errorf("Due = %q, want %q", item.Due, expected)
	}
	// DoneDate should still be set
	if item.DoneDate == "" {
		t.Error("DoneDate should be set even for recurring items")
	}
}

// ---------------------------------------------------------------------------
// DeepCopyItem tests
// ---------------------------------------------------------------------------

func TestDeepCopyItem(t *testing.T) {
	original := &Item{
		Text:     "Parent",
		State:    Open,
		Priority: Now,
		Due:      "2026-06-01",
		Created:  "2026-05-10",
		Tags:     []string{"urgent", "backend"},
		Recur:    "weekly",
		Children: []*Item{
			{Text: "Child 1", State: Open, Priority: Now, Tags: []string{"sub"}},
			{Text: "Child 2", State: Done, Priority: Backlog},
		},
	}

	cp := DeepCopyItem(original)

	// Verify copy matches
	if cp.Text != original.Text {
		t.Errorf("copy.Text = %q, want %q", cp.Text, original.Text)
	}
	if cp.Due != original.Due {
		t.Errorf("copy.Due = %q, want %q", cp.Due, original.Due)
	}
	if len(cp.Tags) != 2 {
		t.Fatalf("copy.Tags len = %d, want 2", len(cp.Tags))
	}
	if len(cp.Children) != 2 {
		t.Fatalf("copy.Children len = %d, want 2", len(cp.Children))
	}

	// Modify original, verify copy is unchanged
	original.Text = "Modified Parent"
	original.Tags[0] = "changed"
	original.Children[0].Text = "Modified Child"

	if cp.Text != "Parent" {
		t.Errorf("copy.Text = %q after original modification, want %q", cp.Text, "Parent")
	}
	if cp.Tags[0] != "urgent" {
		t.Errorf("copy.Tags[0] = %q after original modification, want %q", cp.Tags[0], "urgent")
	}
	if cp.Children[0].Text != "Child 1" {
		t.Errorf("copy.Children[0].Text = %q after original modification, want %q", cp.Children[0].Text, "Child 1")
	}
}

// ---------------------------------------------------------------------------
// Metadata parsing tests
// ---------------------------------------------------------------------------

func TestParseMetadataTags(t *testing.T) {
	input := `---
title: Tagged
---

# Tagged

- [ ] Task with tags priority=now tags=frontend,urgent created=2026-05-10
`
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(list.Items))
	}
	item := list.Items[0]
	if len(item.Tags) != 2 {
		t.Fatalf("Tags len = %d, want 2", len(item.Tags))
	}
	if item.Tags[0] != "frontend" {
		t.Errorf("Tags[0] = %q, want %q", item.Tags[0], "frontend")
	}
	if item.Tags[1] != "urgent" {
		t.Errorf("Tags[1] = %q, want %q", item.Tags[1], "urgent")
	}
}

func TestParseMetadataRecur(t *testing.T) {
	input := `---
title: Recurring
---

# Recurring

- [ ] Daily standup priority=now due=2026-05-10 recur=weekly created=2026-05-01
`
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(list.Items))
	}
	if list.Items[0].Recur != "weekly" {
		t.Errorf("Recur = %q, want %q", list.Items[0].Recur, "weekly")
	}
}

func TestParseMetadataDoneDate(t *testing.T) {
	input := `---
title: Done
---

# Done

- [x] Completed task priority=now created=2026-05-08 done=2026-05-10
`
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(list.Items))
	}
	item := list.Items[0]
	if item.State != Done {
		t.Errorf("State = %q, want %q", item.State, Done)
	}
	if item.DoneDate != "2026-05-10" {
		t.Errorf("DoneDate = %q, want %q", item.DoneDate, "2026-05-10")
	}
}

// ---------------------------------------------------------------------------
// Edge case tests
// ---------------------------------------------------------------------------

func TestParseEmptyList(t *testing.T) {
	input := `---
title: Empty
created: 2026-05-10T12:00:00Z
updated: 2026-05-10T12:00:00Z
---

# Empty
`
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if list.Title != "Empty" {
		t.Errorf("Title = %q, want %q", list.Title, "Empty")
	}
	if len(list.Items) != 0 {
		t.Errorf("len(Items) = %d, want 0", len(list.Items))
	}
}

func TestParseMalformedMarkdown(t *testing.T) {
	inputs := []string{
		"",
		"just plain text",
		"---\nno closing frontmatter",
		"# No frontmatter heading\nsome content\n",
		"---\n---\n\nno items here, just text\n\nparagraph two",
		"random\n\n\tbytes\x00\x01\x02",
	}

	for i, input := range inputs {
		list, err := Parse(input)
		if err != nil {
			t.Errorf("input[%d]: Parse should not error, got: %v", i, err)
		}
		if list == nil {
			t.Errorf("input[%d]: list should not be nil", i)
		}
	}
}

func TestParseDeeplyNestedItems(t *testing.T) {
	input := `---
title: Deep
---

# Deep

- [ ] Level 1 priority=now created=2026-05-10
  - [ ] Level 2 priority=now created=2026-05-10
    - [ ] Level 3 priority=now created=2026-05-10
      - [ ] Level 4 priority=now created=2026-05-10
`
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("got %d top-level items, want 1", len(list.Items))
	}

	level1 := list.Items[0]
	if level1.Text != "Level 1" {
		t.Errorf("Level 1 text = %q", level1.Text)
	}
	if len(level1.Children) != 1 {
		t.Fatalf("Level 1 children = %d, want 1", len(level1.Children))
	}

	level2 := level1.Children[0]
	if level2.Text != "Level 2" {
		t.Errorf("Level 2 text = %q", level2.Text)
	}
	if len(level2.Children) != 1 {
		t.Fatalf("Level 2 children = %d, want 1", len(level2.Children))
	}

	level3 := level2.Children[0]
	if level3.Text != "Level 3" {
		t.Errorf("Level 3 text = %q", level3.Text)
	}
	if len(level3.Children) != 1 {
		t.Fatalf("Level 3 children = %d, want 1", len(level3.Children))
	}

	level4 := level3.Children[0]
	if level4.Text != "Level 4" {
		t.Errorf("Level 4 text = %q", level4.Text)
	}

	// Verify ResolveItem works with deep nesting
	resolved, err := ResolveItem(list, "1.1.1.1")
	if err != nil {
		t.Fatalf("ResolveItem(1.1.1.1) error: %v", err)
	}
	if resolved.Text != "Level 4" {
		t.Errorf("ResolveItem(1.1.1.1) = %q, want %q", resolved.Text, "Level 4")
	}
}

func TestParseUnicodeText(t *testing.T) {
	input := `---
title: Unicode Tasks
---

# Unicode Tasks

- [ ] Deploy to production priority=now created=2026-05-10
- [ ] Fix Nachricht-Encoding priority=now created=2026-05-10
- [ ] Review by Tanaka-san priority=backlog created=2026-05-10
`
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(list.Items) != 3 {
		t.Fatalf("got %d items, want 3", len(list.Items))
	}

	expected := []string{
		"Deploy to production",
		"Fix Nachricht-Encoding",
		"Review by Tanaka-san",
	}
	for i, want := range expected {
		if list.Items[i].Text != want {
			t.Errorf("Items[%d].Text = %q, want %q", i, list.Items[i].Text, want)
		}
	}

	// Round-trip through Render/Parse
	rendered := Render(list)
	reparsed, err := Parse(rendered)
	if err != nil {
		t.Fatalf("re-Parse error: %v", err)
	}
	for i, want := range expected {
		if reparsed.Items[i].Text != want {
			t.Errorf("after round-trip Items[%d].Text = %q, want %q", i, reparsed.Items[i].Text, want)
		}
	}
}

// ---------------------------------------------------------------------------
// List management tests (filesystem)
// ---------------------------------------------------------------------------

func TestListDir(t *testing.T) {
	got := ListDir("/some/project")
	want := filepath.Join("/some/project", "todos")
	if got != want {
		t.Errorf("ListDir = %q, want %q", got, want)
	}
}

func TestListPath(t *testing.T) {
	got := ListPath("/some/project", "sprint-1")
	want := filepath.Join("/some/project", "todos", "sprint-1.md")
	if got != want {
		t.Errorf("ListPath = %q, want %q", got, want)
	}
}

func TestCreateList(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "todos"), 0o755)

	list, err := CreateList(dir, "backlog", "Backlog Items")
	if err != nil {
		t.Fatalf("CreateList error: %v", err)
	}
	if list.Title != "Backlog Items" {
		t.Errorf("Title = %q, want %q", list.Title, "Backlog Items")
	}

	// File should exist
	path := ListPath(dir, "backlog")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("list file should exist on disk")
	}
}

func TestCreateListDuplicate(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "todos"), 0o755)

	_, err := CreateList(dir, "backlog", "Backlog")
	if err != nil {
		t.Fatalf("first CreateList error: %v", err)
	}

	_, err = CreateList(dir, "backlog", "Backlog Again")
	if err == nil {
		t.Error("duplicate CreateList should error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %q, should contain 'already exists'", err.Error())
	}
}

func TestLoadSaveList(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "todos"), 0o755)

	original := &List{
		Title:   "Round Trip",
		Created: time.Now().UTC(),
	}
	AddItem(original, "Task A", Now, "2026-06-01")
	AddItem(original, "Task B", Backlog, "")

	err := SaveList(dir, "roundtrip", original)
	if err != nil {
		t.Fatalf("SaveList error: %v", err)
	}

	loaded, err := LoadList(dir, "roundtrip")
	if err != nil {
		t.Fatalf("LoadList error: %v", err)
	}

	if loaded.Title != "Round Trip" {
		t.Errorf("Title = %q, want %q", loaded.Title, "Round Trip")
	}
	if len(loaded.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(loaded.Items))
	}
	if loaded.Items[0].Text != "Task A" {
		t.Errorf("Items[0].Text = %q, want %q", loaded.Items[0].Text, "Task A")
	}
	if loaded.Items[0].Due != "2026-06-01" {
		t.Errorf("Items[0].Due = %q, want %q", loaded.Items[0].Due, "2026-06-01")
	}
	if loaded.Items[1].Text != "Task B" {
		t.Errorf("Items[1].Text = %q, want %q", loaded.Items[1].Text, "Task B")
	}
	if loaded.Items[1].Priority != Backlog {
		t.Errorf("Items[1].Priority = %q, want %q", loaded.Items[1].Priority, Backlog)
	}
}

func TestListNames(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "todos"), 0o755)

	_, _ = CreateList(dir, "alpha", "Alpha")
	_, _ = CreateList(dir, "beta", "Beta")
	_, _ = CreateList(dir, "gamma", "Gamma")

	names, err := ListNames(dir)
	if err != nil {
		t.Fatalf("ListNames error: %v", err)
	}

	if len(names) != 3 {
		t.Fatalf("len(names) = %d, want 3", len(names))
	}

	// Names are returned from ReadDir which gives sorted order
	want := []string{"alpha", "beta", "gamma"}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("names[%d] = %q, want %q", i, names[i], w)
		}
	}
}

func TestListNamesEmpty(t *testing.T) {
	dir := t.TempDir()
	// No todos dir yet - should return nil, nil
	names, err := ListNames(dir)
	if err != nil {
		t.Fatalf("ListNames error: %v", err)
	}
	if names != nil {
		t.Errorf("names = %v, want nil for non-existent dir", names)
	}
}

// ---------------------------------------------------------------------------
// Recurrence tests (testing nextDueDate indirectly via SetState)
// ---------------------------------------------------------------------------

func TestNextDueDateDaily(t *testing.T) {
	item := &Item{
		Text:     "Daily standup",
		State:    Open,
		Priority: Now,
		Due:      time.Now().UTC().Format("2006-01-02"),
		Recur:    "daily",
	}
	SetState(item, Done)

	if item.State != Open {
		t.Errorf("State = %q, want %q", item.State, Open)
	}

	expected := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	if item.Due != expected {
		t.Errorf("Due = %q, want %q", item.Due, expected)
	}
}

func TestNextDueDateWeekly(t *testing.T) {
	item := &Item{
		Text:     "Weekly review",
		State:    Open,
		Priority: Now,
		Due:      time.Now().UTC().Format("2006-01-02"),
		Recur:    "weekly",
	}
	SetState(item, Done)

	if item.State != Open {
		t.Errorf("State = %q, want %q", item.State, Open)
	}

	expected := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")
	if item.Due != expected {
		t.Errorf("Due = %q, want %q", item.Due, expected)
	}
}

func TestNextDueDateMonthly(t *testing.T) {
	item := &Item{
		Text:     "Monthly report",
		State:    Open,
		Priority: Now,
		Due:      time.Now().UTC().Format("2006-01-02"),
		Recur:    "monthly",
	}
	SetState(item, Done)

	if item.State != Open {
		t.Errorf("State = %q, want %q", item.State, Open)
	}

	expected := time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02")
	if item.Due != expected {
		t.Errorf("Due = %q, want %q", item.Due, expected)
	}
}

func TestSetStateRecurringCreatesNewDue(t *testing.T) {
	item := &Item{
		Text:     "Sprint planning",
		State:    Open,
		Priority: Now,
		Due:      time.Now().UTC().Format("2006-01-02"),
		Recur:    "weekly",
	}

	originalDue := item.Due
	SetState(item, Done)

	if item.State != Open {
		t.Errorf("State = %q, want %q after recurring done", item.State, Open)
	}
	if item.Due == originalDue {
		t.Error("Due should have changed after completing recurring item")
	}
	if item.DoneDate == "" {
		t.Error("DoneDate should be set after completing recurring item")
	}

	// Complete it again - due should advance again
	prevDue := item.Due
	SetState(item, Done)
	if item.Due == prevDue {
		t.Error("Due should advance on each completion")
	}
	if item.State != Open {
		t.Errorf("State = %q, want %q after second recurring done", item.State, Open)
	}
}

// ---------------------------------------------------------------------------
// ParseState tests
// ---------------------------------------------------------------------------

func TestParseState(t *testing.T) {
	tests := []struct {
		marker string
		want   State
	}{
		{"[x]", Done},
		{"[-]", Blocked},
		{"[ ]", Open},
		{"[?]", Open}, // unknown defaults to Open
		{"", Open},    // empty defaults to Open
		{"xyz", Open}, // garbage defaults to Open
	}

	for _, tt := range tests {
		got := ParseState(tt.marker)
		if got != tt.want {
			t.Errorf("ParseState(%q) = %q, want %q", tt.marker, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Context field tests
// ---------------------------------------------------------------------------

func TestParseContextMultiLine(t *testing.T) {
	input := `---
title: DB Migration
created: 2026-05-11T01:00:00Z
updated: 2026-05-11T01:00:00Z
context:
  - architecture/*
  - decisions/db-*
  - runbooks/deploy
---

# DB Migration

- [ ] Run migration priority=now created=2026-05-11
`
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if list.Title != "DB Migration" {
		t.Errorf("Title = %q, want %q", list.Title, "DB Migration")
	}
	if len(list.Context) != 3 {
		t.Fatalf("Context len = %d, want 3", len(list.Context))
	}
	want := []string{"architecture/*", "decisions/db-*", "runbooks/deploy"}
	for i, w := range want {
		if list.Context[i] != w {
			t.Errorf("Context[%d] = %q, want %q", i, list.Context[i], w)
		}
	}
	if len(list.Items) != 1 {
		t.Errorf("Items len = %d, want 1", len(list.Items))
	}
}

func TestParseContextSingleValue(t *testing.T) {
	input := `---
title: Simple
created: 2026-05-11T01:00:00Z
updated: 2026-05-11T01:00:00Z
context: overview
---

# Simple
`
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(list.Context) != 1 {
		t.Fatalf("Context len = %d, want 1", len(list.Context))
	}
	if list.Context[0] != "overview" {
		t.Errorf("Context[0] = %q, want %q", list.Context[0], "overview")
	}
}

func TestParseContextEmptyList(t *testing.T) {
	input := `---
title: No Docs
created: 2026-05-11T01:00:00Z
updated: 2026-05-11T01:00:00Z
context: []
---

# No Docs
`
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if list.Context == nil {
		t.Fatal("Context should be non-nil (empty slice), got nil")
	}
	if len(list.Context) != 0 {
		t.Errorf("Context len = %d, want 0", len(list.Context))
	}
}

func TestParseContextNil(t *testing.T) {
	input := `---
title: Default
created: 2026-05-11T01:00:00Z
updated: 2026-05-11T01:00:00Z
---

# Default
`
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if list.Context != nil {
		t.Errorf("Context should be nil when not specified, got %v", list.Context)
	}
}

func TestParseContextBareKeyNoItems(t *testing.T) {
	input := `---
title: Bare Context
created: 2026-05-11T01:00:00Z
updated: 2026-05-11T01:00:00Z
context:
---

# Bare Context
`
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	// "context:" with no items should mean empty list (no docs), not nil
	if list.Context == nil {
		t.Fatal("Context should be non-nil (empty slice) for bare 'context:' key, got nil")
	}
	if len(list.Context) != 0 {
		t.Errorf("Context len = %d, want 0", len(list.Context))
	}
}

func TestRenderContextMultiLine(t *testing.T) {
	list := &List{
		Title:   "With Context",
		Context: []string{"architecture/*", "decisions/db-*"},
	}
	rendered := Render(list)

	if !strings.Contains(rendered, "context:\n") {
		t.Error("should contain 'context:' header")
	}
	if !strings.Contains(rendered, "  - architecture/*\n") {
		t.Error("should contain architecture/* pattern")
	}
	if !strings.Contains(rendered, "  - decisions/db-*\n") {
		t.Error("should contain decisions/db-* pattern")
	}
}

func TestRenderContextEmptyList(t *testing.T) {
	list := &List{
		Title:   "Empty Context",
		Context: []string{},
	}
	rendered := Render(list)

	if !strings.Contains(rendered, "context: []\n") {
		t.Errorf("should contain 'context: []', got:\n%s", rendered)
	}
}

func TestRenderContextNil(t *testing.T) {
	list := &List{
		Title: "No Context",
	}
	rendered := Render(list)

	if strings.Contains(rendered, "context") {
		t.Errorf("should not contain 'context' when nil, got:\n%s", rendered)
	}
}

func TestContextRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		context []string
	}{
		{"nil context", nil},
		{"empty context", []string{}},
		{"single pattern", []string{"overview"}},
		{"multiple patterns", []string{"architecture/*", "decisions/db-*", "runbooks/deploy"}},
		{"wildcard patterns", []string{"*", "**"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list := &List{
				Title:   "Round Trip",
				Created: time.Date(2026, 5, 11, 1, 0, 0, 0, time.UTC),
				Updated: time.Date(2026, 5, 11, 2, 0, 0, 0, time.UTC),
				Context: tt.context,
			}
			AddItem(list, "Task", Now, "")

			rendered := Render(list)
			parsed, err := Parse(rendered)
			if err != nil {
				t.Fatalf("Parse(Render()) error: %v", err)
			}

			if tt.context == nil {
				if parsed.Context != nil {
					t.Errorf("expected nil context, got %v", parsed.Context)
				}
			} else {
				if parsed.Context == nil {
					t.Fatalf("expected non-nil context, got nil")
				}
				if len(parsed.Context) != len(tt.context) {
					t.Fatalf("context len = %d, want %d", len(parsed.Context), len(tt.context))
				}
				for i, want := range tt.context {
					if parsed.Context[i] != want {
						t.Errorf("context[%d] = %q, want %q", i, parsed.Context[i], want)
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// YAML frontmatter title sanitization tests
// ---------------------------------------------------------------------------

func TestRenderQuotesSpecialTitles(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		wantQuoted  bool // whether the YAML title value should be double-quoted
	}{
		{"plain title", "My Tasks", false},
		{"colon in title", "Migration: Phase 2", true},
		{"hash in title", "Issue #42 Fix", true},
		{"brackets in title", "[WIP] New Feature", true},
		{"curly braces", "{draft} API Design", true},
		{"ampersand", "R&D Tasks", true},
		{"asterisk", "Fix *critical* bug", true},
		{"double quotes", `The "big" refactor`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list := &List{
				Title:   tt.title,
				Created: time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
				Updated: time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
			}
			rendered := Render(list)

			if tt.wantQuoted {
				if !strings.Contains(rendered, `title: "`) {
					t.Errorf("expected quoted title in YAML, got:\n%s", rendered)
				}
			} else {
				if strings.Contains(rendered, `title: "`) {
					t.Errorf("expected unquoted title in YAML, got:\n%s", rendered)
				}
			}
		})
	}
}

func TestTitleSpecialCharsRoundTrip(t *testing.T) {
	titles := []string{
		"Simple Title",
		"Migration: Phase 2",
		"Issue #42 Fix",
		"[WIP] New Feature",
		`The "big" refactor`,
		"R&D Tasks",
		"Fix *critical* bug",
		"100% complete",
		"Step 1: Fix #42 [urgent]",
		"Compare > contrast",
		"Build | Deploy",
	}

	for _, title := range titles {
		t.Run(title, func(t *testing.T) {
			list := &List{
				Title:   title,
				Created: time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
				Updated: time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
			}
			AddItem(list, "Task", Now, "")

			rendered := Render(list)
			parsed, err := Parse(rendered)
			if err != nil {
				t.Fatalf("Parse(Render()) error: %v", err)
			}

			if parsed.Title != title {
				t.Errorf("title round-trip failed: got %q, want %q", parsed.Title, title)
			}
		})
	}
}

func TestTitleSpecialCharsSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "todos"), 0o755)

	titles := []string{
		"Migration: Phase 2",
		"Issue #42 Fix",
		"[WIP] New Feature",
		`The "big" refactor`,
	}

	for i, title := range titles {
		listName := fmt.Sprintf("test-%d", i)
		t.Run(title, func(t *testing.T) {
			list, err := CreateList(dir, listName, title)
			if err != nil {
				t.Fatalf("CreateList error: %v", err)
			}
			if list.Title != title {
				t.Errorf("CreateList title = %q, want %q", list.Title, title)
			}

			loaded, err := LoadList(dir, listName)
			if err != nil {
				t.Fatalf("LoadList error: %v", err)
			}
			if loaded.Title != title {
				t.Errorf("loaded title = %q, want %q", loaded.Title, title)
			}
		})
	}
}

func TestParseQuotedTitleFromExistingFile(t *testing.T) {
	// Simulate a file that was saved with a quoted title
	input := `---
title: "Migration: Phase 2"
created: 2026-05-11T12:00:00Z
updated: 2026-05-11T12:00:00Z
---

# Migration: Phase 2

- [ ] Run migration priority=now created=2026-05-11
`
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if list.Title != "Migration: Phase 2" {
		t.Errorf("title = %q, want %q", list.Title, "Migration: Phase 2")
	}
}

func TestParseSingleQuotedTitle(t *testing.T) {
	// Ensure single-quoted titles are also handled
	input := `---
title: 'Deploy: Stage 1'
created: 2026-05-11T12:00:00Z
updated: 2026-05-11T12:00:00Z
---

# Deploy: Stage 1
`
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if list.Title != "Deploy: Stage 1" {
		t.Errorf("title = %q, want %q", list.Title, "Deploy: Stage 1")
	}
}

func TestParseUnquotedTitleBackwardCompat(t *testing.T) {
	// Existing files with unquoted plain titles should still work
	input := `---
title: DB Refactor
created: 2026-05-11T12:00:00Z
updated: 2026-05-11T12:00:00Z
---

# DB Refactor
`
	list, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if list.Title != "DB Refactor" {
		t.Errorf("title = %q, want %q", list.Title, "DB Refactor")
	}
}

func TestContextSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "todos"), 0o755)

	original := &List{
		Title:   "Context Test",
		Created: time.Now().UTC(),
		Context: []string{"arch/*", "design/**"},
	}
	AddItem(original, "Do something", Now, "")

	if err := SaveList(dir, "ctx-test", original); err != nil {
		t.Fatalf("SaveList error: %v", err)
	}

	loaded, err := LoadList(dir, "ctx-test")
	if err != nil {
		t.Fatalf("LoadList error: %v", err)
	}

	if len(loaded.Context) != 2 {
		t.Fatalf("Context len = %d, want 2", len(loaded.Context))
	}
	if loaded.Context[0] != "arch/*" {
		t.Errorf("Context[0] = %q, want %q", loaded.Context[0], "arch/*")
	}
	if loaded.Context[1] != "design/**" {
		t.Errorf("Context[1] = %q, want %q", loaded.Context[1], "design/**")
	}
}

// ---------------------------------------------------------------------------
// Subdirectory support tests
// ---------------------------------------------------------------------------

func TestListNamesRecursive(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "todos"), 0o755)

	_, _ = CreateList(dir, "backlog", "Backlog")
	_, _ = CreateList(dir, "sprint/week-1", "Sprint Week 1")
	_, _ = CreateList(dir, "sprint/week-2", "Sprint Week 2")
	_, _ = CreateList(dir, "team/backend/tasks", "Backend Tasks")

	names, err := ListNames(dir)
	if err != nil {
		t.Fatalf("ListNames error: %v", err)
	}

	want := []string{"backlog", "sprint/week-1", "sprint/week-2", "team/backend/tasks"}
	if len(names) != len(want) {
		t.Fatalf("len(names) = %d, want %d; got %v", len(names), len(want), names)
	}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("names[%d] = %q, want %q", i, names[i], w)
		}
	}
}

func TestListNamesSkipsArchive(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "todos"), 0o755)
	os.MkdirAll(filepath.Join(dir, "todos", ".archive"), 0o755)

	_, _ = CreateList(dir, "active", "Active")
	// Manually create an archived list
	archivePath := filepath.Join(dir, "todos", ".archive", "old.md")
	os.WriteFile(archivePath, []byte("---\ntitle: Old\n---\n"), 0o644)

	names, err := ListNames(dir)
	if err != nil {
		t.Fatalf("ListNames error: %v", err)
	}

	if len(names) != 1 || names[0] != "active" {
		t.Errorf("names = %v, want [active]", names)
	}
}

func TestCreateListSubdir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "todos"), 0o755)

	list, err := CreateList(dir, "sprint/week-1", "Sprint Week 1")
	if err != nil {
		t.Fatalf("CreateList error: %v", err)
	}
	if list.Title != "Sprint Week 1" {
		t.Errorf("Title = %q, want %q", list.Title, "Sprint Week 1")
	}

	// Verify file was created in correct subdirectory
	path := ListPath(dir, "sprint/week-1")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("list file not created at expected path: %v", err)
	}

	// Verify it loads back correctly
	loaded, err := LoadList(dir, "sprint/week-1")
	if err != nil {
		t.Fatalf("LoadList error: %v", err)
	}
	if loaded.Title != "Sprint Week 1" {
		t.Errorf("loaded Title = %q, want %q", loaded.Title, "Sprint Week 1")
	}
}

func TestSaveListCreatesSubdirs(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "todos"), 0o755)

	list := &List{
		Title:   "Deep Nested",
		Created: time.Now().UTC(),
	}
	AddItem(list, "Test item", Now, "")

	err := SaveList(dir, "deep/nested/list", list)
	if err != nil {
		t.Fatalf("SaveList error: %v", err)
	}

	// Load back and verify
	loaded, err := LoadList(dir, "deep/nested/list")
	if err != nil {
		t.Fatalf("LoadList error: %v", err)
	}
	if loaded.Title != "Deep Nested" {
		t.Errorf("Title = %q, want %q", loaded.Title, "Deep Nested")
	}
	if len(loaded.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(loaded.Items))
	}
	if loaded.Items[0].Text != "Test item" {
		t.Errorf("Items[0].Text = %q, want %q", loaded.Items[0].Text, "Test item")
	}
}

func TestArchivedListNamesRecursive(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "todos", ".archive")

	// Create archived lists in subdirectories
	os.MkdirAll(filepath.Join(archiveDir, "sprint"), 0o755)
	os.WriteFile(filepath.Join(archiveDir, "old.md"), []byte("---\ntitle: Old\n---\n"), 0o644)
	os.WriteFile(filepath.Join(archiveDir, "sprint", "week-1.md"), []byte("---\ntitle: Week 1\n---\n"), 0o644)

	names, err := ArchivedListNames(dir)
	if err != nil {
		t.Fatalf("ArchivedListNames error: %v", err)
	}

	want := []string{"old", "sprint/week-1"}
	if len(names) != len(want) {
		t.Fatalf("len(names) = %d, want %d; got %v", len(names), len(want), names)
	}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("names[%d] = %q, want %q", i, names[i], w)
		}
	}
}

func TestCleanEmptyParents(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "todos")
	os.MkdirAll(filepath.Join(root, "sprint", "deep"), 0o755)

	// Create a file deep in the tree
	filePath := filepath.Join(root, "sprint", "deep", "test.md")
	os.WriteFile(filePath, []byte("test"), 0o644)

	// Remove the file
	os.Remove(filePath)

	// Clean up empty parents
	CleanEmptyParents(filePath, root)

	// Both "deep" and "sprint" should be removed since they're empty
	if _, err := os.Stat(filepath.Join(root, "sprint", "deep")); !os.IsNotExist(err) {
		t.Error("sprint/deep directory should have been removed")
	}
	if _, err := os.Stat(filepath.Join(root, "sprint")); !os.IsNotExist(err) {
		t.Error("sprint directory should have been removed")
	}
	// Root should still exist
	if _, err := os.Stat(root); err != nil {
		t.Error("root directory should still exist")
	}
}

func TestCleanEmptyParentsStopsAtNonEmpty(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "todos")
	os.MkdirAll(filepath.Join(root, "sprint", "deep"), 0o755)

	// Create a sibling file in sprint/
	os.WriteFile(filepath.Join(root, "sprint", "other.md"), []byte("keep"), 0o644)

	filePath := filepath.Join(root, "sprint", "deep", "test.md")
	os.WriteFile(filePath, []byte("test"), 0o644)
	os.Remove(filePath)

	CleanEmptyParents(filePath, root)

	// "deep" should be removed
	if _, err := os.Stat(filepath.Join(root, "sprint", "deep")); !os.IsNotExist(err) {
		t.Error("sprint/deep directory should have been removed")
	}
	// "sprint" should still exist (has other.md)
	if _, err := os.Stat(filepath.Join(root, "sprint")); err != nil {
		t.Error("sprint directory should still exist (has sibling file)")
	}
}

func TestCheckNameConflictFileVsDir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "todos"), 0o755)

	// Create a list "sprint"
	_, _ = CreateList(dir, "sprint", "Sprint")

	// Trying to create "sprint/week-1" should conflict (sprint.md exists)
	err := CheckNameConflict(dir, "sprint/week-1")
	if err == nil {
		t.Error("expected conflict error when creating sprint/week-1 with sprint.md existing")
	}
}

func TestCheckNameConflictDirVsFile(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "todos"), 0o755)

	// Create a list "sprint/week-1" (creates sprint/ directory)
	_, _ = CreateList(dir, "sprint/week-1", "Week 1")

	// Trying to create "sprint" should conflict (sprint/ directory exists)
	err := CheckNameConflict(dir, "sprint")
	if err == nil {
		t.Error("expected conflict error when creating sprint with sprint/ directory existing")
	}
}

func TestCheckNameConflictNoConflict(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "todos"), 0o755)

	// Create "sprint/week-1"
	_, _ = CreateList(dir, "sprint/week-1", "Week 1")

	// Creating "sprint/week-2" should be fine
	err := CheckNameConflict(dir, "sprint/week-2")
	if err != nil {
		t.Errorf("unexpected conflict error: %v", err)
	}

	// Creating "backlog" should also be fine
	err = CheckNameConflict(dir, "backlog")
	if err != nil {
		t.Errorf("unexpected conflict error: %v", err)
	}
}

func TestCreateListConflictDetection(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "todos"), 0o755)

	// Create "sprint/week-1"
	_, err := CreateList(dir, "sprint/week-1", "Week 1")
	if err != nil {
		t.Fatalf("CreateList error: %v", err)
	}

	// Trying to create "sprint" should fail
	_, err = CreateList(dir, "sprint", "Sprint")
	if err == nil {
		t.Error("expected error when creating list conflicting with existing directory")
	}
}
