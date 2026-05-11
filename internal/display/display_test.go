package display

import (
	"testing"

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

func TestFilterItemsReturnsOriginalSliceWhenNoFilters(t *testing.T) {
	items := []*todo.Item{{Text: "a"}, {Text: "b"}}
	got := FilterItems(items, "", "", "")
	if &got[0] != &items[0] {
		t.Error("FilterItems with no filters should return the original slice")
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
