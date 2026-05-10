package todo

import (
	"strings"
	"testing"
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
