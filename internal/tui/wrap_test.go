package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/walter/p/internal/todo"
)

// =======================================================================
// wrapLine Tests
// =======================================================================

func TestWrapLine_BasicWrapping(t *testing.T) {
	// "hello world foo" at width 11 should wrap to two lines
	lines := wrapLine("hello world foo", 11, 0)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "hello world" {
		t.Errorf("line 0 = %q, want %q", lines[0], "hello world")
	}
	if lines[1] != "foo" {
		t.Errorf("line 1 = %q, want %q", lines[1], "foo")
	}
}

func TestWrapLine_NoWrapNeeded(t *testing.T) {
	lines := wrapLine("short text", 50, 0)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
	}
	if lines[0] != "short text" {
		t.Errorf("line 0 = %q", lines[0])
	}
}

func TestWrapLine_ExactWidth(t *testing.T) {
	// Text exactly fills the width — should not wrap
	lines := wrapLine("12345", 5, 0)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
	}
	if lines[0] != "12345" {
		t.Errorf("line 0 = %q", lines[0])
	}
}

func TestWrapLine_EmptyText(t *testing.T) {
	lines := wrapLine("", 50, 0)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0] != "" {
		t.Errorf("line 0 = %q, want empty", lines[0])
	}
}

func TestWrapLine_WhitespaceOnly(t *testing.T) {
	lines := wrapLine("   ", 50, 0)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0] != "" {
		t.Errorf("line 0 = %q, want empty (Fields trims whitespace)", lines[0])
	}
}

func TestWrapLine_LongSingleWordHardWrap(t *testing.T) {
	// A single word longer than maxWidth should be hard-wrapped
	lines := wrapLine("abcdefghijklmnop", 5, 0)
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "abcde" {
		t.Errorf("line 0 = %q", lines[0])
	}
	if lines[1] != "fghij" {
		t.Errorf("line 1 = %q", lines[1])
	}
	if lines[2] != "klmno" {
		t.Errorf("line 2 = %q", lines[2])
	}
	if lines[3] != "p" {
		t.Errorf("line 3 = %q", lines[3])
	}
}

func TestWrapLine_MixedShortAndLongWords(t *testing.T) {
	lines := wrapLine("hi abcdefghij world", 5, 0)
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "hi" {
		t.Errorf("line 0 = %q", lines[0])
	}
	if lines[1] != "abcde" {
		t.Errorf("line 1 = %q", lines[1])
	}
	if lines[2] != "fghij" {
		t.Errorf("line 2 = %q", lines[2])
	}
	if lines[3] != "world" {
		t.Errorf("line 3 = %q", lines[3])
	}
}

func TestWrapLine_ContinuationIndent(t *testing.T) {
	lines := wrapLine("hello world foo bar", 11, 4)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "hello world" {
		t.Errorf("line 0 = %q, want %q", lines[0], "hello world")
	}
	// Continuation line should be indented by 4 spaces
	if !strings.HasPrefix(lines[1], "    ") {
		t.Errorf("line 1 should have 4-space indent, got %q", lines[1])
	}
	if lines[1] != "    foo bar" {
		t.Errorf("line 1 = %q, want %q", lines[1], "    foo bar")
	}
}

func TestWrapLine_MultipleContinuationLines(t *testing.T) {
	text := "one two three four five six seven eight nine ten"
	lines := wrapLine(text, 15, 3)
	// All continuation lines should have 3-space indent
	for i := 1; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "   ") {
			t.Errorf("line %d should have 3-space indent, got %q", i, lines[i])
		}
	}
	// First line should not have indent
	if strings.HasPrefix(lines[0], " ") {
		t.Errorf("first line should not have indent, got %q", lines[0])
	}
}

func TestWrapLine_ZeroWidth(t *testing.T) {
	lines := wrapLine("hello", 0, 0)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0] != "hello" {
		t.Errorf("got %q, want %q", lines[0], "hello")
	}
}

func TestWrapLine_NegativeWidth(t *testing.T) {
	lines := wrapLine("hello", -5, 0)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
}

func TestWrapLine_SingleWord(t *testing.T) {
	lines := wrapLine("hello", 10, 0)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0] != "hello" {
		t.Errorf("got %q", lines[0])
	}
}

// =======================================================================
// truncateText Tests
// =======================================================================

func TestTruncateText_ShortText(t *testing.T) {
	result := truncateText("hello", 20)
	if result != "hello" {
		t.Errorf("got %q, want %q", result, "hello")
	}
}

func TestTruncateText_ExactWidth(t *testing.T) {
	result := truncateText("hello", 5)
	if result != "hello" {
		t.Errorf("got %q, want %q", result, "hello")
	}
}

func TestTruncateText_Truncated(t *testing.T) {
	result := truncateText("hello world", 8)
	if lipgloss.Width(result) > 8 {
		t.Errorf("result %q exceeds maxWidth 8 (width=%d)", result, lipgloss.Width(result))
	}
	if !strings.HasSuffix(result, "…") {
		t.Errorf("truncated text should end with ellipsis, got %q", result)
	}
	// Should keep as much text as possible
	if !strings.HasPrefix(result, "hello w") {
		t.Errorf("should preserve max text, got %q", result)
	}
}

func TestTruncateText_EmptyMaxWidth(t *testing.T) {
	result := truncateText("hello", 0)
	if result != "" {
		t.Errorf("got %q, want empty", result)
	}
}

func TestTruncateText_NegativeMaxWidth(t *testing.T) {
	result := truncateText("hello", -1)
	if result != "" {
		t.Errorf("got %q, want empty", result)
	}
}

func TestTruncateText_Width1(t *testing.T) {
	result := truncateText("hello", 1)
	if result != "…" {
		t.Errorf("got %q, want %q", result, "…")
	}
}

func TestTruncateText_Width2(t *testing.T) {
	result := truncateText("hello", 2)
	if lipgloss.Width(result) > 2 {
		t.Errorf("result %q exceeds maxWidth 2", result)
	}
	if !strings.HasSuffix(result, "…") {
		t.Errorf("should end with ellipsis, got %q", result)
	}
}

func TestTruncateText_EmptyText(t *testing.T) {
	result := truncateText("", 10)
	if result != "" {
		t.Errorf("got %q, want empty", result)
	}
}

// =======================================================================
// calcScrollOffset Tests
// =======================================================================

func TestCalcScrollOffset_CursorVisible(t *testing.T) {
	// Cursor already visible — no change
	heights := []int{1, 1, 1, 1, 1}
	offset := calcScrollOffset(heights, 2, 0, 5)
	if offset != 0 {
		t.Errorf("got offset %d, want 0 (cursor already visible)", offset)
	}
}

func TestCalcScrollOffset_ScrollDown(t *testing.T) {
	// Cursor is below viewport — scroll down
	heights := []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	offset := calcScrollOffset(heights, 8, 0, 5)
	// Cursor item starts at line 8, ends at line 9
	// Need to scroll so line 9 is at bottom: offset = 9 - 5 = 4
	if offset != 4 {
		t.Errorf("got offset %d, want 4", offset)
	}
}

func TestCalcScrollOffset_ScrollUp(t *testing.T) {
	// Cursor is above viewport — scroll up
	heights := []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	offset := calcScrollOffset(heights, 2, 5, 5)
	// Cursor item starts at line 2 — scroll to show it
	if offset != 2 {
		t.Errorf("got offset %d, want 2", offset)
	}
}

func TestCalcScrollOffset_MultiLineItem(t *testing.T) {
	// Item at cursor takes 3 lines, viewport is 5
	heights := []int{1, 1, 3, 1, 1}
	offset := calcScrollOffset(heights, 2, 0, 5)
	// Cursor item starts at line 2, ends at line 5
	// Viewport starts at 0, ends at 5 — item fits!
	if offset != 0 {
		t.Errorf("got offset %d, want 0 (item fits in viewport)", offset)
	}
}

func TestCalcScrollOffset_MultiLineItemScrollDown(t *testing.T) {
	// Item at cursor takes 3 lines, needs scrolling
	heights := []int{1, 1, 1, 1, 3, 1}
	offset := calcScrollOffset(heights, 4, 0, 5)
	// Cursor item starts at line 4, ends at line 7
	// Need offset = 7 - 5 = 2
	if offset != 2 {
		t.Errorf("got offset %d, want 2", offset)
	}
}

func TestCalcScrollOffset_ItemTallerThanViewport(t *testing.T) {
	// Item is taller than viewport — show from item start
	heights := []int{1, 1, 8, 1}
	offset := calcScrollOffset(heights, 2, 0, 5)
	// Cursor item starts at line 2, is 8 lines tall
	// Should show from top of item
	if offset != 2 {
		t.Errorf("got offset %d, want 2 (top of tall item)", offset)
	}
}

func TestCalcScrollOffset_Empty(t *testing.T) {
	offset := calcScrollOffset(nil, 0, 0, 5)
	if offset != 0 {
		t.Errorf("got offset %d, want 0", offset)
	}
}

func TestCalcScrollOffset_NegativeScrollClamped(t *testing.T) {
	heights := []int{1, 1, 1}
	offset := calcScrollOffset(heights, 0, -5, 5)
	if offset != 0 {
		t.Errorf("got offset %d, want 0 (clamped from -5)", offset)
	}
}

// =======================================================================
// Compact Mode Tests
// =======================================================================

func TestItemListView_CompactMode_SelectedItemWraps(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 50, 24)
	longText := "This is a very long item text that should wrap when selected in compact mode"
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: longText, State: todo.Open},
			{Text: "Short item", State: todo.Open},
		},
	}})

	v.cursor = 0 // Select the long item
	view := v.View()

	// The selected item text should appear fully (not truncated)
	if !strings.Contains(view, "wrap when selected") {
		t.Errorf("selected item should show full wrapped text, got:\n%s", view)
	}
}

func TestItemListView_CompactMode_NonSelectedTruncated(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 40, 24)
	longText := "This is a very long item text that definitely exceeds the terminal width"
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "Short item", State: todo.Open},
			{Text: longText, State: todo.Open},
		},
	}})

	v.cursor = 0 // Short item selected, long item is not
	view := v.View()

	// The non-selected long item should be truncated with ellipsis
	if strings.Contains(view, "exceeds the terminal width") {
		t.Errorf("non-selected item should be truncated, but full text appears in:\n%s", view)
	}
	if !strings.Contains(view, "…") {
		t.Errorf("truncated text should contain ellipsis, got:\n%s", view)
	}
}

func TestItemListView_CompactMode_CursorMoveExpandsNewItem(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 40, 24)
	longText1 := "First long item that needs wrapping when displayed in the TUI view"
	longText2 := "Second long item that also needs wrapping when displayed in the TUI view"
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: longText1, State: todo.Open},
			{Text: longText2, State: todo.Open},
		},
	}})

	// Start at first item — it should be expanded (wrapping across multiple lines)
	v.cursor = 0
	view1 := v.View()
	// Full text should appear — "TUI view" is at the end so it only shows when expanded
	if !strings.Contains(view1, "TUI view") {
		t.Errorf("first item should be expanded when selected, got:\n%s", view1)
	}

	// Move to second item — it should now be expanded
	v.cursor = 1
	view2 := v.View()
	if !strings.Contains(view2, "TUI view") && !strings.Contains(view2, "in the TUI") {
		t.Errorf("second item should be expanded when selected, got:\n%s", view2)
	}
	// First item should be truncated (ellipsis)
	if !strings.Contains(view2, "…") {
		t.Errorf("first item should be truncated when not selected, got:\n%s", view2)
	}
}

// =======================================================================
// Wrapped Mode Tests
// =======================================================================

func TestItemListView_WrappedMode_AllItemsWrap(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 50, 24)
	v.wrapMode = true

	longText1 := "First item with text that is too long to fit on one line"
	longText2 := "Second item with text that is also too long to fit on one line"
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: longText1, State: todo.Open},
			{Text: longText2, State: todo.Open},
		},
	}})

	view := v.View()

	// Both items should show their full text (text wraps across lines)
	// Check for end-of-text fragments that only appear when fully rendered
	if !strings.Contains(view, "on one line") {
		t.Errorf("wrapped mode should show full text of first item, got:\n%s", view)
	}
	if !strings.Contains(view, "on one line") && !strings.Contains(view, "also too") {
		t.Errorf("wrapped mode should show full text of second item, got:\n%s", view)
	}
	// In wrapped mode, there should be no truncation ellipsis
	if strings.Contains(view, "…") {
		t.Errorf("wrapped mode should not have truncation, got:\n%s", view)
	}
}

func TestItemListView_WrappedMode_ShortItemsSingleLine(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.wrapMode = true

	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "Short", State: todo.Open},
			{Text: "Also short", State: todo.Open},
		},
	}})

	view := v.View()
	if !strings.Contains(view, "Short") || !strings.Contains(view, "Also short") {
		t.Errorf("short items should display normally, got:\n%s", view)
	}
}

// =======================================================================
// Display Mode Toggle Tests
// =======================================================================

func TestItemListView_WrapToggle(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{{Text: "item", State: todo.Open}},
	}})

	if v.wrapMode {
		t.Error("wrapMode should default to false (compact)")
	}

	// Press 'w' to toggle
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")})
	if !v.wrapMode {
		t.Error("wrapMode should be true after pressing 'w'")
	}

	// Press 'w' again to toggle back
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")})
	if v.wrapMode {
		t.Error("wrapMode should be false after pressing 'w' again")
	}
}

func TestItemListView_WrapToggle_ResetsScrollOffset(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{{Text: "item", State: todo.Open}},
	}})

	v.scrollOffset = 10 // Simulate a scroll position

	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")})
	if v.scrollOffset != 0 {
		t.Errorf("scrollOffset should reset to 0 on wrap toggle, got %d", v.scrollOffset)
	}
}

func TestItemListView_WrapToggle_ViewChanges(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 40, 24)
	longText := "This is a very long item text that should look different in compact vs wrapped mode"
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "Short", State: todo.Open},
			{Text: longText, State: todo.Open},
		},
	}})

	// Compact mode: cursor on first item, second item truncated
	v.cursor = 0
	compactView := v.View()

	// Switch to wrapped mode
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("w")})
	wrappedView := v.View()

	// Views should differ
	if compactView == wrappedView {
		t.Error("compact and wrapped views should differ")
	}

	// Wrapped mode should show more of the long text (may wrap across lines)
	if !strings.Contains(wrappedView, "look different") {
		t.Errorf("wrapped view should show full text, got:\n%s", wrappedView)
	}
}

// =======================================================================
// Display Mode Indicator Tests
// =======================================================================

func TestItemListView_DisplayModeIndicator_Compact(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{{Text: "item", State: todo.Open}},
	}})

	view := v.View()
	if !strings.Contains(view, "Display: compact") {
		t.Errorf("should show 'Display: compact' in compact mode, got:\n%s", view)
	}
}

func TestItemListView_DisplayModeIndicator_Wrapped(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.wrapMode = true
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{{Text: "item", State: todo.Open}},
	}})

	view := v.View()
	if !strings.Contains(view, "Display: wrapped") {
		t.Errorf("should show 'Display: wrapped' in wrapped mode, got:\n%s", view)
	}
}

// =======================================================================
// Help Bar Tests
// =======================================================================

func TestItemListView_HelpBar_WrapHint(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{{Text: "item", State: todo.Open}},
	}})

	view := v.View()
	if !strings.Contains(view, "w wrap") {
		t.Errorf("help bar should show 'w wrap' in compact mode, got:\n%s", view)
	}

	v.wrapMode = true
	view = v.View()
	if !strings.Contains(view, "w unwrap") {
		t.Errorf("help bar should show 'w unwrap' in wrapped mode, got:\n%s", view)
	}
}

// =======================================================================
// Help Overlay Tests
// =======================================================================

func TestHelpOverlay_IncludesWrapKeybinding(t *testing.T) {
	help := renderContextHelp(ViewItemList)
	if !strings.Contains(help, "wrap") {
		t.Errorf("help overlay should mention wrap keybinding, got:\n%s", help)
	}
}

// =======================================================================
// Scroll Offset Integration Tests
// =======================================================================

func TestItemListView_ScrollOffset_CursorMoveBelowViewport(t *testing.T) {
	// Create a view with height=10 and many items
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 10)
	items := make([]*todo.Item, 20)
	for i := range items {
		items[i] = &todo.Item{Text: "item", State: todo.Open}
	}
	v.Update(ListLoadedMsg{List: &todo.List{Items: items}})

	// Move cursor to bottom
	for i := 0; i < 19; i++ {
		v.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	// Render — scrollOffset should be adjusted so cursor is visible
	view := v.View()
	_ = view // just ensure no panic

	// The scroll offset should have been adjusted
	// With height=10, viewportHeight = 10-7 = 3
	// Cursor at 19, so scrollOffset should be >= 17 (19 - 3 + 1)
	if v.scrollOffset < 17 {
		t.Errorf("scrollOffset = %d, should be >= 17 to show cursor at item 19", v.scrollOffset)
	}
}

func TestItemListView_ScrollOffset_CursorMoveAboveViewport(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 10)
	items := make([]*todo.Item, 20)
	for i := range items {
		items[i] = &todo.Item{Text: "item", State: todo.Open}
	}
	v.Update(ListLoadedMsg{List: &todo.List{Items: items}})

	// Move cursor to bottom then back to top
	for i := 0; i < 19; i++ {
		v.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	v.View() // trigger scroll calculation
	for i := 0; i < 19; i++ {
		v.Update(tea.KeyMsg{Type: tea.KeyUp})
	}
	v.View() // trigger scroll calculation

	// Scroll offset should be back near 0
	if v.scrollOffset > 0 {
		t.Errorf("scrollOffset = %d, should be 0 when cursor is at top", v.scrollOffset)
	}
}

func TestItemListView_ScrollOffset_WrappedItemVisible(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 40, 12)
	v.wrapMode = true

	longText := "This is a very long item that will definitely need to wrap across multiple lines in a 40-char-wide terminal"
	items := []*todo.Item{
		{Text: "Item 1", State: todo.Open},
		{Text: "Item 2", State: todo.Open},
		{Text: "Item 3", State: todo.Open},
		{Text: longText, State: todo.Open},
		{Text: "Item 5", State: todo.Open},
	}
	v.Update(ListLoadedMsg{List: &todo.List{Items: items}})

	// Move cursor to the long item
	v.cursor = 3
	view := v.View()
	_ = view

	// The long item text should be visible (fully or at least starting)
	if !strings.Contains(view, "very long item") {
		t.Errorf("wrapped long item should be visible when cursor is on it, got:\n%s", view)
	}
}

// =======================================================================
// Child Item Wrapping Tests
// =======================================================================

func TestItemListView_ChildItems_ContinuationIndent(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 50, 24)
	v.wrapMode = true

	longChildText := "This child item has a very long description that needs wrapping to fit"
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{
				Text: "Parent item", State: todo.Open,
				Children: []*todo.Item{
					{Text: longChildText, State: todo.Open},
				},
			},
		},
	}})

	view := v.View()

	// Child text should appear (wrapped)
	if !strings.Contains(view, "child item has") {
		t.Errorf("child item text should be visible, got:\n%s", view)
	}
	// The parent should also be visible
	if !strings.Contains(view, "Parent item") {
		t.Errorf("parent item should be visible, got:\n%s", view)
	}
}

// =======================================================================
// Edge Cases
// =======================================================================

func TestItemListView_NarrowTerminal(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 20, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "A task with some text", State: todo.Open},
		},
	}})

	// Should not panic with very narrow terminal
	view := v.View()
	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestItemListView_WrapModeWithMetadata(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 50, 24)
	v.wrapMode = true

	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{
				Text:     "A long item that wraps with priority and due date metadata",
				State:    todo.Open,
				Priority: todo.Now,
				Due:      "2026-05-20",
				Tags:     []string{"urgent"},
			},
		},
	}})

	view := v.View()

	// Metadata should be present
	if !strings.Contains(view, "now") {
		t.Error("priority 'now' should be visible")
	}
	if !strings.Contains(view, "2026-05-20") {
		t.Error("due date should be visible")
	}
	if !strings.Contains(view, "#urgent") {
		t.Error("tag should be visible")
	}
}

func TestItemListView_DoneItemInWrappedMode(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 40, 24)
	v.wrapMode = true

	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "A completed item with enough text to wrap across lines", State: todo.Done},
		},
	}})

	view := v.View()
	// Done marker should be present
	if !strings.Contains(view, "[x]") {
		t.Error("done marker should be visible")
	}
	// Text should still appear
	if !strings.Contains(view, "completed item") {
		t.Errorf("done item text should be visible in wrapped mode, got:\n%s", view)
	}
}

func TestItemListView_ActionsWorkInBothModes(t *testing.T) {
	// Verify that state change commands still work after the View() refactor
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "item", State: todo.Open},
		},
	}})

	// Verify toggleDone returns a command in compact mode
	cmd := v.toggleDone()
	if cmd == nil {
		t.Error("toggleDone should return a command in compact mode")
	}

	// Switch to wrapped mode
	v.wrapMode = true

	// Verify toggleDone still returns a command
	cmd = v.toggleDone()
	if cmd == nil {
		t.Error("toggleDone should return a command in wrapped mode")
	}
}

func TestItemListView_FilterAndWrapInteraction(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 50, 24)
	v.wrapMode = true

	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "Open item with long text that wraps", State: todo.Open},
			{Text: "Done item with long text that wraps", State: todo.Done},
		},
	}})

	// Filter to open only
	v.setFilter(filterOpen)
	view := v.View()

	// Only open item should be visible
	if !strings.Contains(view, "Open item") {
		t.Error("open item should be visible after filter")
	}
	if strings.Contains(view, "Done item") {
		t.Error("done item should not be visible after open filter")
	}
}
