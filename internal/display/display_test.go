package display

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/walter/p/internal/todo"
)

func TestFilterItems(t *testing.T) {
	items := []*todo.Item{
		{Text: "open item", State: todo.Open, Priority: todo.Now},
		{Text: "done item", State: todo.Done, Priority: todo.Now},
		{Text: "blocked item", State: todo.Blocked, Priority: todo.Backlog, Tags: []string{"bug"}},
		{Text: "backlog item", State: todo.Open, Priority: todo.Backlog, Tags: []string{"bug", "frontend"}},
	}

	tests := []struct {
		name     string
		state    string
		priority string
		tag      string
		want     int
	}{
		{"no filter", "", "", "", 4},
		{"open only", "open", "", "", 2},
		{"done only", "done", "", "", 1},
		{"blocked only", "blocked", "", "", 1},
		{"backlog priority", "", "backlog", "", 2},
		{"now priority", "", "now", "", 2},
		{"bug tag", "", "", "bug", 2},
		{"frontend tag", "", "", "frontend", 1},
		{"nonexistent tag", "", "", "backend", 0},
		{"open+backlog", "open", "backlog", "", 1},
		{"open+bug", "open", "", "bug", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterItems(items, tt.state, tt.priority, tt.tag)
			if len(got) != tt.want {
				t.Errorf("FilterItems(%q, %q, %q) returned %d items, want %d",
					tt.state, tt.priority, tt.tag, len(got), tt.want)
			}
		})
	}
}

func TestFilterItemsPreservesOriginalIDs(t *testing.T) {
	items := []*todo.Item{
		{Text: "done task", State: todo.Done, Priority: todo.Now},
		{Text: "open task 1", State: todo.Open, Priority: todo.Now},
		{Text: "done task 2", State: todo.Done, Priority: todo.Now},
		{Text: "open task 2", State: todo.Open, Priority: todo.Now},
		{Text: "open task 3", State: todo.Open, Priority: todo.Now},
	}

	got := FilterItems(items, "open", "", "")
	if len(got) != 3 {
		t.Fatalf("expected 3 open items, got %d", len(got))
	}

	// Items should have original IDs 2, 4, 5 (not 1, 2, 3)
	wantIDs := []string{"2", "4", "5"}
	for i, fi := range got {
		if fi.OriginalID != wantIDs[i] {
			t.Errorf("filtered item %d: OriginalID = %q, want %q", i, fi.OriginalID, wantIDs[i])
		}
	}
}

func TestFilterItemsPreservesChildIDs(t *testing.T) {
	items := []*todo.Item{
		{Text: "done parent", State: todo.Done, Priority: todo.Now},
		{Text: "open parent", State: todo.Open, Priority: todo.Now, Children: []*todo.Item{
			{Text: "done child", State: todo.Done, Priority: todo.Now},
			{Text: "open child", State: todo.Open, Priority: todo.Now},
		}},
		{Text: "another parent", State: todo.Done, Priority: todo.Now, Children: []*todo.Item{
			{Text: "open nested", State: todo.Open, Priority: todo.Now},
		}},
	}

	got := FilterItems(items, "open", "", "")
	if len(got) != 3 {
		t.Fatalf("expected 3 open items, got %d", len(got))
	}

	// Should be: parent at "2", child at "2.2", and nested at "3.1"
	wantIDs := []string{"2", "2.2", "3.1"}
	for i, fi := range got {
		if fi.OriginalID != wantIDs[i] {
			t.Errorf("filtered item %d: OriginalID = %q, want %q", i, fi.OriginalID, wantIDs[i])
		}
	}
}

func TestFilterItemsNoFilterReturnsAll(t *testing.T) {
	items := []*todo.Item{{Text: "a"}, {Text: "b"}}
	got := FilterItems(items, "", "", "")
	if len(got) != 2 {
		t.Errorf("no filter: expected 2 items, got %d", len(got))
	}
	if got[0].Item != items[0] || got[1].Item != items[1] {
		t.Error("FilterItems with no filters should return the original items")
	}
}

func TestDueFilter(t *testing.T) {
	today := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)

	items := []*todo.Item{
		{Text: "overdue task", State: todo.Open, Due: "2026-05-09"},
		{Text: "due today", State: todo.Open, Due: "2026-05-11"},
		{Text: "due tomorrow", State: todo.Open, Due: "2026-05-12"},
		{Text: "due next week", State: todo.Open, Due: "2026-05-18"},
		{Text: "due next month", State: todo.Open, Due: "2026-06-05"},
		{Text: "no due date", State: todo.Open},
		{Text: "overdue but done", State: todo.Done, Due: "2026-05-01"},
	}

	tests := []struct {
		name     string
		dueRange string
		want     int
		wantIDs  []string
	}{
		{"today", "today", 1, []string{"2"}},
		{"overdue", "overdue", 1, []string{"1"}},          // done items excluded
		{"week", "week", 3, []string{"2", "3", "4"}},      // today + tomorrow + next week (within 7 days)
		{"month", "month", 4, []string{"2", "3", "4", "5"}},
		{"none", "none", 1, []string{"6"}},
		{"specific date", "2026-05-12", 1, []string{"3"}},
		{"no match", "2026-12-25", 0, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DueFilter(items, tt.dueRange, today)
			if len(got) != tt.want {
				t.Errorf("DueFilter(%q) returned %d items, want %d", tt.dueRange, len(got), tt.want)
				for _, fi := range got {
					t.Logf("  got: %s (due=%s, state=%s)", fi.Item.Text, fi.Item.Due, fi.Item.State)
				}
				return
			}
			for i, fi := range got {
				if i < len(tt.wantIDs) && fi.OriginalID != tt.wantIDs[i] {
					t.Errorf("item %d: OriginalID = %q, want %q", i, fi.OriginalID, tt.wantIDs[i])
				}
			}
		})
	}
}

func TestDueFilterWithChildren(t *testing.T) {
	today := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)

	items := []*todo.Item{
		{Text: "parent no due", State: todo.Open, Children: []*todo.Item{
			{Text: "child due today", State: todo.Open, Due: "2026-05-11"},
			{Text: "child no due", State: todo.Open},
		}},
		{Text: "parent due today", State: todo.Open, Due: "2026-05-11"},
	}

	got := DueFilter(items, "today", today)
	if len(got) != 2 {
		t.Fatalf("expected 2 items due today, got %d", len(got))
	}
	if got[0].OriginalID != "1.1" {
		t.Errorf("first match: OriginalID = %q, want %q", got[0].OriginalID, "1.1")
	}
	if got[1].OriginalID != "2" {
		t.Errorf("second match: OriginalID = %q, want %q", got[1].OriginalID, "2")
	}
}

func TestHasTag(t *testing.T) {
	item := &todo.Item{Tags: []string{"bug", "frontend"}}

	if !HasTag(item, "bug") {
		t.Error("HasTag should return true for 'bug'")
	}
	if !HasTag(item, "frontend") {
		t.Error("HasTag should return true for 'frontend'")
	}
	if HasTag(item, "backend") {
		t.Error("HasTag should return false for 'backend'")
	}
}

func TestHasTagEmpty(t *testing.T) {
	item := &todo.Item{}
	if HasTag(item, "anything") {
		t.Error("HasTag should return false for item with no tags")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is a ..."},
		{"", 5, ""},
		{"hello", 0, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Truncate(tt.input, tt.max)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}

func TestTruncateUnicode(t *testing.T) {
	// Ensure truncation works on rune boundaries, not bytes
	got := Truncate("日本語テスト文字列", 4)
	if got != "日本語テ..." {
		t.Errorf("Truncate with unicode = %q, want %q", got, "日本語テ...")
	}
}

func TestLooksLikeURL(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"https://example.com/path?q=1", true},
		{"ftp://example.com", false},
		{"not a url", false},
		{"", false},
		{"httpx://nope", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := LooksLikeURL(tt.input)
			if got != tt.want {
				t.Errorf("LooksLikeURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatchContext(t *testing.T) {
	content := "This is a document about testing Go applications with proper tooling."

	// Basic match
	result := MatchContext(content, "testing")
	if result == "" {
		t.Error("MatchContext should find 'testing'")
	}

	// Case insensitive
	result = MatchContext(content, "TESTING")
	if result == "" {
		t.Error("MatchContext should be case insensitive")
	}

	// No match
	result = MatchContext(content, "python")
	if result != "" {
		t.Errorf("MatchContext should return empty for no match, got %q", result)
	}
}

func TestMatchContextShortContent(t *testing.T) {
	result := MatchContext("hello", "hello")
	if result == "" {
		t.Error("MatchContext should match short content")
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "missing", false},
		{"", "test", false},
		{"test", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"/"+tt.substr, func(t *testing.T) {
			got := ContainsIgnoreCase(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("ContainsIgnoreCase(%q, %q) = %v, want %v",
					tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestDimTextPreservingLinks(t *testing.T) {
	// Just verify it doesn't panic and returns non-empty
	result := DimTextPreservingLinks("text [[link]] more")
	if result == "" {
		t.Error("DimTextPreservingLinks should return non-empty string")
	}

	// No links — should still work
	result = DimTextPreservingLinks("plain text")
	if result == "" {
		t.Error("DimTextPreservingLinks should handle text without links")
	}

	// Multiple links
	result = DimTextPreservingLinks("see [[foo]] and [[bar]]")
	if result == "" {
		t.Error("DimTextPreservingLinks should handle multiple links")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("creating pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout

	out, _ := io.ReadAll(r)
	return string(out)
}

func TestPrintItems(t *testing.T) {
	items := []*todo.Item{
		{Text: "Open task", State: todo.Open, Priority: todo.Now},
		{Text: "Done task", State: todo.Done, Priority: todo.Now, DoneDate: "2026-05-10"},
		{Text: "Backlog task", State: todo.Open, Priority: todo.Backlog, Due: "2026-06-01"},
	}

	output := captureStdout(t, func() {
		PrintItems(items, "", 1)
	})

	if !strings.Contains(output, "Open task") {
		t.Errorf("output should contain 'Open task', got:\n%s", output)
	}
	if !strings.Contains(output, "Done task") {
		t.Errorf("output should contain 'Done task', got:\n%s", output)
	}
	if !strings.Contains(output, "Backlog task") {
		t.Errorf("output should contain 'Backlog task', got:\n%s", output)
	}
}

func TestPrintItemsNested(t *testing.T) {
	items := []*todo.Item{
		{Text: "Parent", State: todo.Open, Priority: todo.Now, Children: []*todo.Item{
			{Text: "Child", State: todo.Done, Priority: todo.Now},
		}},
	}

	output := captureStdout(t, func() {
		PrintItems(items, "", 1)
	})

	if !strings.Contains(output, "Parent") {
		t.Error("output should contain 'Parent'")
	}
	if !strings.Contains(output, "Child") {
		t.Error("output should contain 'Child'")
	}
}

func TestPrintFilteredItems(t *testing.T) {
	filtered := []FilteredItem{
		{OriginalID: "2", Item: &todo.Item{Text: "Second task", State: todo.Open, Priority: todo.Now}},
		{OriginalID: "4", Item: &todo.Item{Text: "Fourth task", State: todo.Open, Priority: todo.Backlog, Due: "2026-07-01"}},
	}

	output := captureStdout(t, func() {
		PrintFilteredItems(filtered)
	})

	if !strings.Contains(output, "Second task") {
		t.Error("output should contain 'Second task'")
	}
	if !strings.Contains(output, "Fourth task") {
		t.Error("output should contain 'Fourth task'")
	}
}

func TestPrintItemsEmpty(t *testing.T) {
	output := captureStdout(t, func() {
		PrintItems(nil, "", 1)
	})
	if output != "" {
		t.Errorf("expected empty output for nil items, got: %q", output)
	}
}

func TestPrintFilteredItemsEmpty(t *testing.T) {
	output := captureStdout(t, func() {
		PrintFilteredItems(nil)
	})
	if output != "" {
		t.Errorf("expected empty output for nil items, got: %q", output)
	}
}
