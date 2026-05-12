package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/walter/p/internal/todo"
)

// =======================================================================
// ProjectListView Tests
// =======================================================================

func TestProjectListView_NewAndInit(t *testing.T) {
	v := NewProjectListView("/tmp/root", 80, 24)

	if v.projectRoot != "/tmp/root" {
		t.Errorf("projectRoot = %q", v.projectRoot)
	}
	if v.loaded {
		t.Error("should not be loaded initially")
	}
	if v.cursor != 0 {
		t.Error("cursor should start at 0")
	}

	// Init should return a loadProjects command
	cmd := v.Init()
	if cmd == nil {
		t.Error("Init() should return a command")
	}
}

func TestProjectListView_LoadingView(t *testing.T) {
	v := NewProjectListView("/tmp/root", 80, 24)
	view := v.View()

	if !strings.Contains(view, "Loading") {
		t.Error("view should show 'Loading' before data loads")
	}
}

func TestProjectListView_EmptyProjects(t *testing.T) {
	v := NewProjectListView("/tmp/root", 80, 24)

	// Simulate loaded with no projects
	v.Update(ProjectsLoadedMsg{Projects: nil})

	view := v.View()
	if !strings.Contains(view, "No projects found") {
		t.Errorf("empty project list should show 'No projects found', got:\n%s", view)
	}
}

func TestProjectListView_RendersProjects(t *testing.T) {
	v := NewProjectListView("/tmp/root", 80, 24)

	projects := []ProjectInfo{
		{Name: "alpha", Dir: "/tmp/root/alpha", Open: 5, Done: 3, Blocked: 1},
		{Name: "beta", Dir: "/tmp/root/beta", Open: 0, Done: 10, Blocked: 0},
		{Name: "gamma", Dir: "/tmp/root/gamma", Open: 2, Done: 0, Blocked: 0},
	}
	v.Update(ProjectsLoadedMsg{Projects: projects})

	view := v.View()

	// Check project names appear in the output
	if !strings.Contains(view, "alpha") {
		t.Error("view should contain 'alpha'")
	}
	if !strings.Contains(view, "beta") {
		t.Error("view should contain 'beta'")
	}
	if !strings.Contains(view, "gamma") {
		t.Error("view should contain 'gamma'")
	}
}

func TestProjectListView_CursorNavigation(t *testing.T) {
	v := NewProjectListView("/tmp/root", 80, 24)
	v.Update(ProjectsLoadedMsg{
		Projects: []ProjectInfo{
			{Name: "a"}, {Name: "b"}, {Name: "c"},
		},
	})

	if v.cursor != 0 {
		t.Fatalf("cursor should start at 0, got %d", v.cursor)
	}

	// Move down
	v.Update(tea.KeyMsg{Type: tea.KeyDown})
	if v.cursor != 1 {
		t.Errorf("cursor should be 1 after down, got %d", v.cursor)
	}

	// Move down again
	v.Update(tea.KeyMsg{Type: tea.KeyDown})
	if v.cursor != 2 {
		t.Errorf("cursor should be 2 after second down, got %d", v.cursor)
	}

	// Should not exceed bounds
	v.Update(tea.KeyMsg{Type: tea.KeyDown})
	if v.cursor != 2 {
		t.Errorf("cursor should stay at 2 (max), got %d", v.cursor)
	}

	// Move up
	v.Update(tea.KeyMsg{Type: tea.KeyUp})
	if v.cursor != 1 {
		t.Errorf("cursor should be 1 after up, got %d", v.cursor)
	}

	// Move up past start
	v.Update(tea.KeyMsg{Type: tea.KeyUp})
	v.Update(tea.KeyMsg{Type: tea.KeyUp})
	if v.cursor != 0 {
		t.Errorf("cursor should stay at 0 (min), got %d", v.cursor)
	}
}

func TestProjectListView_CursorNavigation_Vim(t *testing.T) {
	v := NewProjectListView("/tmp/root", 80, 24)
	v.Update(ProjectsLoadedMsg{
		Projects: []ProjectInfo{{Name: "a"}, {Name: "b"}},
	})

	// j moves down
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if v.cursor != 1 {
		t.Errorf("cursor should be 1 after 'j', got %d", v.cursor)
	}

	// k moves up
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if v.cursor != 0 {
		t.Errorf("cursor should be 0 after 'k', got %d", v.cursor)
	}
}

func TestProjectListView_Enter_Navigates(t *testing.T) {
	v := NewProjectListView("/tmp/root", 80, 24)
	v.Update(ProjectsLoadedMsg{
		Projects: []ProjectInfo{
			{Name: "myproj", Dir: "/tmp/root/myproj", Open: 5, Done: 3},
		},
	})

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should return a command")
	}

	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if navMsg.To != ViewTodoList {
		t.Errorf("should navigate to ViewTodoList, got %d", navMsg.To)
	}
	if navMsg.ProjectName != "myproj" {
		t.Errorf("ProjectName = %q", navMsg.ProjectName)
	}
	if navMsg.ProjectDir != "/tmp/root/myproj" {
		t.Errorf("ProjectDir = %q", navMsg.ProjectDir)
	}
}

func TestProjectListView_Enter_EmptyList(t *testing.T) {
	v := NewProjectListView("/tmp/root", 80, 24)
	v.Update(ProjectsLoadedMsg{Projects: nil})

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Enter on empty list should not produce a command")
	}
}

func TestProjectListView_Esc_GoesBack(t *testing.T) {
	v := NewProjectListView("/tmp/root", 80, 24)
	v.Update(ProjectsLoadedMsg{Projects: nil})

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Esc should return a command")
	}

	msg := cmd()
	if _, ok := msg.(GoBackMsg); !ok {
		t.Errorf("expected GoBackMsg, got %T", msg)
	}
}

func TestProjectListView_WindowResize(t *testing.T) {
	v := NewProjectListView("/tmp/root", 80, 24)

	v.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if v.width != 120 || v.height != 40 {
		t.Errorf("got width=%d height=%d, want 120/40", v.width, v.height)
	}
}

func TestProjectListView_DataChanged_Reloads(t *testing.T) {
	v := NewProjectListView("/tmp/root", 80, 24)
	v.loaded = true

	_, cmd := v.Update(DataChangedMsg{})
	if cmd == nil {
		t.Error("DataChangedMsg should trigger a reload command")
	}
}

func TestProjectListView_CursorClampedOnReload(t *testing.T) {
	v := NewProjectListView("/tmp/root", 80, 24)
	v.Update(ProjectsLoadedMsg{
		Projects: []ProjectInfo{{Name: "a"}, {Name: "b"}, {Name: "c"}},
	})
	v.cursor = 2 // Last item

	// Reload with fewer items
	v.Update(ProjectsLoadedMsg{
		Projects: []ProjectInfo{{Name: "a"}},
	})

	if v.cursor != 0 {
		t.Errorf("cursor should be clamped to 0, got %d", v.cursor)
	}
}

// =======================================================================
// ItemListView Tests
// =======================================================================

func TestItemListView_NewAndInit(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)

	if v.projectName != "proj" {
		t.Errorf("projectName = %q", v.projectName)
	}
	if v.listName != "backlog" {
		t.Errorf("listName = %q", v.listName)
	}
	if v.loaded {
		t.Error("should not be loaded initially")
	}
	if v.cursor != 0 {
		t.Error("cursor should start at 0")
	}
	if v.IsInputMode() {
		t.Error("should not be in input mode initially")
	}
}

func TestItemListView_LoadingView(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	view := v.View()

	if !strings.Contains(view, "Loading") {
		t.Error("should show 'Loading' before data loads")
	}
}

func TestItemListView_EmptyList(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{Title: "backlog"}})

	view := v.View()
	if !strings.Contains(view, "No items") {
		t.Errorf("empty list should show 'No items', got:\n%s", view)
	}
}

func TestItemListView_RendersItems(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)

	list := &todo.List{
		Title: "backlog",
		Items: []*todo.Item{
			{Text: "Fix login bug", State: todo.Open, Priority: todo.Now},
			{Text: "Write tests", State: todo.Done, Priority: todo.Backlog},
			{Text: "Update docs", State: todo.Blocked, Priority: todo.Backlog},
		},
	}
	v.Update(ListLoadedMsg{List: list})

	view := v.View()

	// Check state markers
	if !strings.Contains(view, "[ ]") {
		t.Error("should contain [ ] for open items")
	}
	if !strings.Contains(view, "[x]") {
		t.Error("should contain [x] for done items")
	}
	if !strings.Contains(view, "[-]") {
		t.Error("should contain [-] for blocked items")
	}

	// Check item text
	if !strings.Contains(view, "Fix login bug") {
		t.Error("should contain 'Fix login bug'")
	}
	if !strings.Contains(view, "Write tests") {
		t.Error("should contain 'Write tests'")
	}
}

func TestItemListView_StateMarkers(t *testing.T) {
	tests := []struct {
		state  todo.State
		marker string
	}{
		{todo.Open, "[ ]"},
		{todo.Done, "[x]"},
		{todo.Blocked, "[-]"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			v := NewItemListView("proj", "/tmp/proj", "test", 80, 24)
			v.Update(ListLoadedMsg{List: &todo.List{
				Items: []*todo.Item{{Text: "item", State: tt.state}},
			}})

			view := v.View()
			if !strings.Contains(view, tt.marker) {
				t.Errorf("view should contain marker %q for state %q", tt.marker, tt.state)
			}
		})
	}
}

func TestItemListView_CursorNavigation(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "item1", State: todo.Open},
			{Text: "item2", State: todo.Open},
			{Text: "item3", State: todo.Open},
		},
	}})

	if v.cursor != 0 {
		t.Fatalf("cursor should start at 0, got %d", v.cursor)
	}

	v.Update(tea.KeyMsg{Type: tea.KeyDown})
	if v.cursor != 1 {
		t.Errorf("cursor after down = %d, want 1", v.cursor)
	}

	v.Update(tea.KeyMsg{Type: tea.KeyDown})
	v.Update(tea.KeyMsg{Type: tea.KeyDown}) // past end
	if v.cursor != 2 {
		t.Errorf("cursor should clamp at 2, got %d", v.cursor)
	}

	v.Update(tea.KeyMsg{Type: tea.KeyUp})
	if v.cursor != 1 {
		t.Errorf("cursor after up = %d, want 1", v.cursor)
	}
}

func TestItemListView_Enter_NavigatesToDetail(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "item1", State: todo.Open},
			{Text: "item2", State: todo.Open},
		},
	}})

	v.cursor = 1

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should return a command")
	}

	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if navMsg.To != ViewItemDetail {
		t.Errorf("should navigate to ViewItemDetail")
	}
	if navMsg.ItemID != "2" {
		t.Errorf("ItemID = %q, want %q", navMsg.ItemID, "2")
	}
}

func TestItemListView_Esc_GoesBack(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{}})

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Esc should return a command")
	}

	msg := cmd()
	if _, ok := msg.(GoBackMsg); !ok {
		t.Errorf("expected GoBackMsg, got %T", msg)
	}
}

func TestItemListView_WindowResize(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)

	v.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	if v.width != 100 || v.height != 30 {
		t.Errorf("got width=%d height=%d, want 100/30", v.width, v.height)
	}
}

// --- Filtering Tests ---

func TestFilterItems_AllFilter(t *testing.T) {
	items := []*todo.Item{
		{Text: "open1", State: todo.Open},
		{Text: "done1", State: todo.Done},
		{Text: "blocked1", State: todo.Blocked},
	}

	result := filterItems(items, "", "", "", 1)
	if len(result) != 3 {
		t.Errorf("all filter should return 3 items, got %d", len(result))
	}
}

func TestFilterItems_OpenOnly(t *testing.T) {
	items := []*todo.Item{
		{Text: "open1", State: todo.Open},
		{Text: "done1", State: todo.Done},
		{Text: "open2", State: todo.Open},
	}

	result := filterItems(items, "open", "", "", 1)
	if len(result) != 2 {
		t.Errorf("open filter should return 2 items, got %d", len(result))
	}
	for _, fi := range result {
		if fi.Item.State != todo.Open {
			t.Errorf("expected all open, got %q", fi.Item.State)
		}
	}
}

func TestFilterItems_DoneOnly(t *testing.T) {
	items := []*todo.Item{
		{Text: "open1", State: todo.Open},
		{Text: "done1", State: todo.Done},
		{Text: "done2", State: todo.Done},
	}

	result := filterItems(items, "done", "", "", 1)
	if len(result) != 2 {
		t.Errorf("done filter should return 2 items, got %d", len(result))
	}
}

func TestFilterItems_BlockedOnly(t *testing.T) {
	items := []*todo.Item{
		{Text: "open1", State: todo.Open},
		{Text: "blocked1", State: todo.Blocked},
	}

	result := filterItems(items, "blocked", "", "", 1)
	if len(result) != 1 {
		t.Errorf("blocked filter should return 1 item, got %d", len(result))
	}
}

func TestFilterItems_PriorityFilter(t *testing.T) {
	items := []*todo.Item{
		{Text: "now1", State: todo.Open, Priority: todo.Now},
		{Text: "backlog1", State: todo.Open, Priority: todo.Backlog},
		{Text: "now2", State: todo.Open, Priority: todo.Now},
	}

	result := filterItems(items, "", "now", "", 1)
	if len(result) != 2 {
		t.Errorf("now filter should return 2 items, got %d", len(result))
	}

	result = filterItems(items, "", "backlog", "", 1)
	if len(result) != 1 {
		t.Errorf("backlog filter should return 1 item, got %d", len(result))
	}
}

func TestFilterItems_CombinedFilter(t *testing.T) {
	items := []*todo.Item{
		{Text: "open-now", State: todo.Open, Priority: todo.Now},
		{Text: "open-backlog", State: todo.Open, Priority: todo.Backlog},
		{Text: "done-now", State: todo.Done, Priority: todo.Now},
	}

	result := filterItems(items, "open", "now", "", 1)
	if len(result) != 1 {
		t.Errorf("open+now filter should return 1 item, got %d", len(result))
	}
	if result[0].Item.Text != "open-now" {
		t.Errorf("expected 'open-now', got %q", result[0].Item.Text)
	}
}

func TestFilterItems_PreservesOriginalIDs(t *testing.T) {
	items := []*todo.Item{
		{Text: "a", State: todo.Open},
		{Text: "b", State: todo.Done},
		{Text: "c", State: todo.Open},
	}

	result := filterItems(items, "open", "", "", 1)
	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if result[0].OriginalID != "1" {
		t.Errorf("first item ID = %q, want %q", result[0].OriginalID, "1")
	}
	if result[1].OriginalID != "3" {
		t.Errorf("second item ID = %q, want %q", result[1].OriginalID, "3")
	}
}

func TestFilterItems_NestedChildren(t *testing.T) {
	items := []*todo.Item{
		{
			Text: "parent", State: todo.Open,
			Children: []*todo.Item{
				{Text: "child1", State: todo.Open},
				{Text: "child2", State: todo.Done},
			},
		},
	}

	// All filter
	result := filterItems(items, "", "", "", 1)
	if len(result) != 3 {
		t.Errorf("all filter with children: expected 3, got %d", len(result))
	}

	// Verify child IDs
	ids := make(map[string]string)
	for _, fi := range result {
		ids[fi.OriginalID] = fi.Item.Text
	}
	if ids["1"] != "parent" {
		t.Errorf("ID 1 = %q, want 'parent'", ids["1"])
	}
	if ids["1.1"] != "child1" {
		t.Errorf("ID 1.1 = %q, want 'child1'", ids["1.1"])
	}
	if ids["1.2"] != "child2" {
		t.Errorf("ID 1.2 = %q, want 'child2'", ids["1.2"])
	}
}

func TestFilterItems_FilteredChildren(t *testing.T) {
	items := []*todo.Item{
		{
			Text: "parent", State: todo.Done,
			Children: []*todo.Item{
				{Text: "child1", State: todo.Open},
				{Text: "child2", State: todo.Done},
			},
		},
	}

	// Open filter — parent is done but child1 is open
	result := filterItems(items, "open", "", "", 1)
	if len(result) != 1 {
		t.Errorf("open filter: expected 1 item (child1), got %d", len(result))
	}
	if len(result) > 0 && result[0].Item.Text != "child1" {
		t.Errorf("expected 'child1', got %q", result[0].Item.Text)
	}
}

func TestItemListView_CycleFilter(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "open1", State: todo.Open},
			{Text: "done1", State: todo.Done},
		},
	}})

	if v.filter != filterAll {
		t.Fatalf("initial filter = %q, want empty (all)", v.filter)
	}

	// Cycle: all → open → done → blocked → all
	v.cycleFilter()
	if v.filter != filterOpen {
		t.Errorf("after 1st cycle: %q, want open", v.filter)
	}

	v.cycleFilter()
	if v.filter != filterDone {
		t.Errorf("after 2nd cycle: %q, want done", v.filter)
	}

	v.cycleFilter()
	if v.filter != filterBlocked {
		t.Errorf("after 3rd cycle: %q, want blocked", v.filter)
	}

	v.cycleFilter()
	if v.filter != filterAll {
		t.Errorf("after 4th cycle: %q, want empty (all)", v.filter)
	}
}

func TestItemListView_CyclePriorityFilter(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "a", State: todo.Open, Priority: todo.Now},
			{Text: "b", State: todo.Open, Priority: todo.Backlog},
		},
	}})

	if v.priorityFilter != priorityFilterAll {
		t.Fatalf("initial priority filter = %q", v.priorityFilter)
	}

	v.cyclePriorityFilter()
	if v.priorityFilter != priorityFilterNow {
		t.Errorf("after 1st cycle: %q, want now", v.priorityFilter)
	}

	v.cyclePriorityFilter()
	if v.priorityFilter != priorityFilterBacklog {
		t.Errorf("after 2nd cycle: %q, want backlog", v.priorityFilter)
	}

	v.cyclePriorityFilter()
	if v.priorityFilter != priorityFilterAll {
		t.Errorf("after 3rd cycle: %q, want all", v.priorityFilter)
	}
}

func TestItemListView_SetFilter(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "a", State: todo.Open},
			{Text: "b", State: todo.Done},
		},
	}})

	v.cursor = 1 // Set cursor to non-zero
	v.setFilter(filterDone)

	if v.filter != filterDone {
		t.Errorf("filter = %q, want done", v.filter)
	}
	// Cursor should reset on filter change
	if v.cursor != 0 {
		t.Errorf("cursor should reset to 0 on filter change, got %d", v.cursor)
	}
}

func TestItemListView_FilterIndicator(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{{Text: "a", State: todo.Open}},
	}})

	// Default filter label
	view := v.View()
	if !strings.Contains(view, "State: all") {
		t.Error("should show 'State: all' by default")
	}

	// After filtering
	v.setFilter(filterOpen)
	view = v.View()
	if !strings.Contains(view, "State: open") {
		t.Error("should show 'State: open' when filtering")
	}
}

func TestItemListView_EmptyFilteredResults(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "open", State: todo.Open},
		},
	}})

	v.setFilter(filterDone)
	view := v.View()
	if !strings.Contains(view, "No items match") {
		t.Error("should show 'No items match' for empty filter results")
	}
}

// --- Input Mode Tests ---

func TestItemListView_IsInputMode(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)

	if v.IsInputMode() {
		t.Error("should not be in input mode initially")
	}

	v.inputMode = true
	if !v.IsInputMode() {
		t.Error("should be in input mode when inputMode=true")
	}

	v.inputMode = false
	v.confirmMode = true
	if !v.IsInputMode() {
		t.Error("should be in input mode when confirmMode=true")
	}

	v.confirmMode = false
	v.moveMode = true
	if !v.IsInputMode() {
		t.Error("should be in input mode when moveMode=true")
	}
}

func TestItemListView_InputMode_HandleKeys(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)

	var capturedValue string
	v.startInput("Prompt: ", "", func(val string) tea.Cmd {
		capturedValue = val
		return nil
	})

	if !v.inputMode {
		t.Fatal("should be in input mode")
	}

	// Type some characters
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})

	if v.inputValue != "hi" {
		t.Errorf("inputValue = %q, want %q", v.inputValue, "hi")
	}

	// Backspace
	v.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if v.inputValue != "h" {
		t.Errorf("inputValue after backspace = %q, want %q", v.inputValue, "h")
	}

	// Enter submits
	v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if v.inputMode {
		t.Error("should exit input mode on enter")
	}
	if capturedValue != "h" {
		t.Errorf("captured value = %q, want %q", capturedValue, "h")
	}
}

func TestItemListView_InputMode_EscCancels(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)

	called := false
	v.startInput("Prompt: ", "", func(val string) tea.Cmd {
		called = true
		return nil
	})

	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	v.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if v.inputMode {
		t.Error("should exit input mode on esc")
	}
	if called {
		t.Error("action should not be called on esc cancel")
	}
}

func TestItemListView_InputMode_WithInitialValue(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)

	v.startInput("Edit: ", "hello world", func(val string) tea.Cmd {
		return nil
	})

	if v.inputValue != "hello world" {
		t.Errorf("inputValue = %q, want initial value", v.inputValue)
	}
}

func TestItemListView_InputMode_EmptySubmit(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)

	called := false
	v.startInput("Prompt: ", "", func(val string) tea.Cmd {
		called = true
		return nil
	})

	// Submit empty
	v.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if called {
		t.Error("action should not be called for empty submit")
	}
}

// --- Confirm Mode Tests ---

func TestItemListView_ConfirmMode_Yes(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{{Text: "item", State: todo.Open}},
	}})

	confirmed := false
	v.confirmMode = true
	v.confirmPrompt = "Delete? (y/n)"
	v.confirmAction = func() tea.Cmd {
		confirmed = true
		return nil
	}

	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

	if v.confirmMode {
		t.Error("should exit confirm mode after 'y'")
	}
	if !confirmed {
		t.Error("action should be called on 'y'")
	}
}

func TestItemListView_ConfirmMode_No(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)

	confirmed := false
	v.confirmMode = true
	v.confirmPrompt = "Delete? (y/n)"
	v.confirmAction = func() tea.Cmd {
		confirmed = true
		return nil
	}

	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	if v.confirmMode {
		t.Error("should exit confirm mode after 'n'")
	}
	if confirmed {
		t.Error("action should NOT be called on 'n'")
	}
}

func TestItemListView_ConfirmMode_Esc(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)

	confirmed := false
	v.confirmMode = true
	v.confirmAction = func() tea.Cmd {
		confirmed = true
		return nil
	}

	v.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if v.confirmMode {
		t.Error("should exit confirm mode on Esc")
	}
	if confirmed {
		t.Error("action should NOT be called on Esc")
	}
}

// --- Move Mode Tests ---

func TestItemListView_MoveMode_Navigation(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{{Text: "item", State: todo.Open}},
	}})

	v.moveMode = true
	v.moveTargets = []string{"sprint", "bugs", "features"}
	v.moveCursor = 0

	// Navigate down
	v.Update(tea.KeyMsg{Type: tea.KeyDown})
	if v.moveCursor != 1 {
		t.Errorf("moveCursor = %d, want 1", v.moveCursor)
	}

	v.Update(tea.KeyMsg{Type: tea.KeyDown})
	if v.moveCursor != 2 {
		t.Errorf("moveCursor = %d, want 2", v.moveCursor)
	}

	// Clamp at max
	v.Update(tea.KeyMsg{Type: tea.KeyDown})
	if v.moveCursor != 2 {
		t.Errorf("moveCursor should clamp at 2, got %d", v.moveCursor)
	}

	// Navigate up
	v.Update(tea.KeyMsg{Type: tea.KeyUp})
	if v.moveCursor != 1 {
		t.Errorf("moveCursor = %d, want 1", v.moveCursor)
	}
}

func TestItemListView_MoveMode_Esc(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.moveMode = true
	v.moveTargets = []string{"sprint"}

	v.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if v.moveMode {
		t.Error("Esc should exit move mode")
	}
	if v.moveTargets != nil {
		t.Error("moveTargets should be cleared")
	}
}

func TestItemListView_MoveMode_RenderOverlay(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{{Text: "item", State: todo.Open}},
	}})
	v.moveMode = true
	v.moveTargets = []string{"sprint", "bugs"}

	view := v.View()
	if !strings.Contains(view, "Move item") {
		t.Error("move mode view should contain 'Move item'")
	}
	if !strings.Contains(view, "sprint") {
		t.Error("should show target list 'sprint'")
	}
	if !strings.Contains(view, "bugs") {
		t.Error("should show target list 'bugs'")
	}
}

// --- Selected ID / Item helpers ---

func TestItemListView_SelectedID(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "first", State: todo.Open},
			{Text: "second", State: todo.Open},
		},
	}})

	v.cursor = 0
	if id := v.selectedID(); id != "1" {
		t.Errorf("selectedID at cursor 0 = %q, want %q", id, "1")
	}

	v.cursor = 1
	if id := v.selectedID(); id != "2" {
		t.Errorf("selectedID at cursor 1 = %q, want %q", id, "2")
	}
}

func TestItemListView_SelectedItem(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "first", State: todo.Open},
			{Text: "second", State: todo.Done},
		},
	}})

	v.cursor = 0
	item := v.selectedItem()
	if item == nil {
		t.Fatal("selectedItem should not be nil")
	}
	if item.Text != "first" {
		t.Errorf("selectedItem text = %q, want 'first'", item.Text)
	}

	v.cursor = 1
	item = v.selectedItem()
	if item.Text != "second" {
		t.Errorf("selectedItem text = %q, want 'second'", item.Text)
	}
}

func TestItemListView_SelectedID_Empty(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{}})

	if id := v.selectedID(); id != "" {
		t.Errorf("selectedID on empty list = %q, want empty", id)
	}
	if item := v.selectedItem(); item != nil {
		t.Error("selectedItem on empty list should be nil")
	}
}

// --- Data Changed ---

func TestItemListView_DataChanged_Reloads(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.loaded = true

	_, cmd := v.Update(DataChangedMsg{StatusText: "done!"})
	if cmd == nil {
		t.Error("DataChangedMsg should trigger reload")
	}
}

func TestItemListView_CursorClampedOnReload(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "a", State: todo.Open},
			{Text: "b", State: todo.Open},
			{Text: "c", State: todo.Open},
		},
	}})

	v.cursor = 2

	// Reload with fewer items
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{{Text: "a", State: todo.Open}},
	}})

	if v.cursor != 0 {
		t.Errorf("cursor should be clamped to 0, got %d", v.cursor)
	}
}

// --- Priority Display ---

func TestItemListView_PriorityDisplay(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "urgent", State: todo.Open, Priority: todo.Now},
			{Text: "later", State: todo.Open, Priority: todo.Backlog},
		},
	}})

	view := v.View()
	if !strings.Contains(view, "now") {
		t.Error("should display 'now' priority")
	}
	if !strings.Contains(view, "backlog") {
		t.Error("should display 'backlog' priority")
	}
}

// --- Due Date Display ---

func TestItemListView_DueDateDisplay(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "with due", State: todo.Open, Due: "2026-05-20"},
		},
	}})

	view := v.View()
	if !strings.Contains(view, "2026-05-20") {
		t.Error("should display due date")
	}
}

// --- Tags Display ---

func TestItemListView_TagsDisplay(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{Text: "tagged", State: todo.Open, Tags: []string{"bug", "critical"}},
		},
	}})

	view := v.View()
	if !strings.Contains(view, "#bug") {
		t.Error("should display #bug tag")
	}
	if !strings.Contains(view, "#critical") {
		t.Error("should display #critical tag")
	}
}

// --- Nested Item Display ---

func TestItemListView_NestedItemIndentation(t *testing.T) {
	v := NewItemListView("proj", "/tmp/proj", "backlog", 80, 24)
	v.Update(ListLoadedMsg{List: &todo.List{
		Items: []*todo.Item{
			{
				Text: "parent", State: todo.Open,
				Children: []*todo.Item{
					{Text: "child", State: todo.Open},
				},
			},
		},
	}})

	// The child (ID 1.1) should have dots in its ID which triggers indentation
	if len(v.items) != 2 {
		t.Fatalf("expected 2 items (parent + child), got %d", len(v.items))
	}
	if v.items[1].OriginalID != "1.1" {
		t.Errorf("child ID = %q, want '1.1'", v.items[1].OriginalID)
	}
}

// =======================================================================
// TodoListView Tests
// =======================================================================

func TestTodoListView_NewAndInit(t *testing.T) {
	v := NewTodoListView("proj", "/tmp/proj", 80, 24)

	if v.projectName != "proj" {
		t.Errorf("projectName = %q", v.projectName)
	}
	if v.loaded {
		t.Error("should not be loaded initially")
	}
	cmd := v.Init()
	if cmd == nil {
		t.Error("Init should return a command")
	}
}

func TestTodoListView_RendersLists(t *testing.T) {
	v := NewTodoListView("proj", "/tmp/proj", 80, 24)
	v.Update(TodoListsLoadedMsg{Lists: []TodoListInfo{
		{Name: "backlog", Open: 5, Done: 3, Blocked: 1},
		{Name: "sprint", Open: 2, Done: 10},
	}})

	view := v.View()
	if !strings.Contains(view, "backlog") {
		t.Error("should contain 'backlog'")
	}
	if !strings.Contains(view, "sprint") {
		t.Error("should contain 'sprint'")
	}
}

func TestTodoListView_EmptyLists(t *testing.T) {
	v := NewTodoListView("proj", "/tmp/proj", 80, 24)
	v.Update(TodoListsLoadedMsg{Lists: nil})

	view := v.View()
	if !strings.Contains(view, "No todo lists found") {
		t.Error("should show 'No todo lists found' when empty")
	}
}

func TestTodoListView_CursorNavigation(t *testing.T) {
	v := NewTodoListView("proj", "/tmp/proj", 80, 24)
	v.Update(TodoListsLoadedMsg{Lists: []TodoListInfo{
		{Name: "a"}, {Name: "b"}, {Name: "c"},
	}})

	v.Update(tea.KeyMsg{Type: tea.KeyDown})
	if v.cursor != 1 {
		t.Errorf("cursor = %d after down, want 1", v.cursor)
	}

	v.Update(tea.KeyMsg{Type: tea.KeyUp})
	if v.cursor != 0 {
		t.Errorf("cursor = %d after up, want 0", v.cursor)
	}
}

func TestTodoListView_Enter_Navigates(t *testing.T) {
	v := NewTodoListView("proj", "/tmp/proj", 80, 24)
	v.Update(TodoListsLoadedMsg{Lists: []TodoListInfo{
		{Name: "backlog"},
	}})

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter should return a command")
	}

	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if navMsg.To != ViewItemList {
		t.Errorf("should navigate to ViewItemList")
	}
	if navMsg.ListName != "backlog" {
		t.Errorf("ListName = %q", navMsg.ListName)
	}
}

func TestTodoListView_Tab_SwitchesToKnowledge(t *testing.T) {
	v := NewTodoListView("proj", "/tmp/proj", 80, 24)
	v.Update(TodoListsLoadedMsg{Lists: []TodoListInfo{{Name: "a"}}})

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd == nil {
		t.Fatal("Tab should return a command")
	}

	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if navMsg.To != ViewKnowledgeList {
		t.Errorf("Tab should navigate to ViewKnowledgeList, got %d", navMsg.To)
	}
}

func TestTodoListView_Search(t *testing.T) {
	v := NewTodoListView("proj", "/tmp/proj", 80, 24)
	v.Update(TodoListsLoadedMsg{Lists: []TodoListInfo{{Name: "a"}}})

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	if cmd == nil {
		t.Fatal("/ should return a command")
	}

	msg := cmd()
	navMsg, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if navMsg.To != ViewSearch {
		t.Errorf("/ should navigate to ViewSearch")
	}
}

func TestTodoListView_Esc_GoesBack(t *testing.T) {
	v := NewTodoListView("proj", "/tmp/proj", 80, 24)
	v.Update(TodoListsLoadedMsg{Lists: nil})

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("Esc should return a command")
	}

	msg := cmd()
	if _, ok := msg.(GoBackMsg); !ok {
		t.Errorf("expected GoBackMsg, got %T", msg)
	}
}

func TestTodoListView_IsInputMode(t *testing.T) {
	v := NewTodoListView("proj", "/tmp/proj", 80, 24)

	if v.IsInputMode() {
		t.Error("should not be in input mode initially")
	}

	v.inputMode = true
	if !v.IsInputMode() {
		t.Error("should report input mode")
	}

	v.inputMode = false
	v.confirmMode = true
	if !v.IsInputMode() {
		t.Error("should report input mode for confirm")
	}
}

func TestTodoListView_WindowResize(t *testing.T) {
	v := NewTodoListView("proj", "/tmp/proj", 80, 24)
	v.Update(tea.WindowSizeMsg{Width: 120, Height: 50})

	if v.width != 120 || v.height != 50 {
		t.Errorf("got width=%d height=%d, want 120/50", v.width, v.height)
	}
}

func TestTodoListView_DataChanged_Reloads(t *testing.T) {
	v := NewTodoListView("proj", "/tmp/proj", 80, 24)
	v.loaded = true

	_, cmd := v.Update(DataChangedMsg{StatusText: "Created"})
	if cmd == nil {
		t.Error("DataChangedMsg should trigger reload")
	}
}

func TestTodoListView_InputMode_NewList(t *testing.T) {
	v := NewTodoListView("proj", "/tmp/proj", 80, 24)
	v.Update(TodoListsLoadedMsg{Lists: nil})

	// Press 'n' to start new list input
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	if !v.inputMode {
		t.Error("should enter input mode on 'n'")
	}
	if v.inputPrompt != "New list name: " {
		t.Errorf("inputPrompt = %q", v.inputPrompt)
	}
}

func TestTodoListView_ConfirmMode_DeleteList(t *testing.T) {
	v := NewTodoListView("proj", "/tmp/proj", 80, 24)
	v.Update(TodoListsLoadedMsg{Lists: []TodoListInfo{{Name: "backlog"}}})

	// Press 'd' to trigger delete confirmation
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	if !v.confirmMode {
		t.Error("should enter confirm mode on 'd'")
	}
	if !strings.Contains(v.confirmPrompt, "Delete") {
		t.Errorf("confirmPrompt should mention Delete, got %q", v.confirmPrompt)
	}
}

func TestTodoListView_ConfirmMode_ArchiveList(t *testing.T) {
	v := NewTodoListView("proj", "/tmp/proj", 80, 24)
	v.Update(TodoListsLoadedMsg{Lists: []TodoListInfo{{Name: "backlog"}}})

	// Press 'a' to trigger archive confirmation
	v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})

	if !v.confirmMode {
		t.Error("should enter confirm mode on 'a'")
	}
	if !strings.Contains(v.confirmPrompt, "Archive") {
		t.Errorf("confirmPrompt should mention Archive, got %q", v.confirmPrompt)
	}
}

// =======================================================================
// Helper Function Tests
// =======================================================================

func TestFormatCounts(t *testing.T) {
	tests := []struct {
		name    string
		open    int
		done    int
		blocked int
		want    string // substring to check
	}{
		{"all zeros", 0, 0, 0, "empty"},
		{"open only", 5, 0, 0, "5 open"},
		{"done only", 0, 10, 0, "10 done"},
		{"blocked only", 0, 0, 3, "3 blocked"},
		{"mixed", 5, 3, 1, "open"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCounts(tt.open, tt.done, tt.blocked)
			if !strings.Contains(result, tt.want) {
				t.Errorf("formatCounts(%d,%d,%d) = %q, want to contain %q",
					tt.open, tt.done, tt.blocked, result, tt.want)
			}
		})
	}
}

func TestSpaces(t *testing.T) {
	if s := spaces(0); s != "" {
		t.Errorf("spaces(0) = %q", s)
	}
	if s := spaces(-1); s != "" {
		t.Errorf("spaces(-1) = %q", s)
	}
	if s := spaces(3); s != "   " {
		t.Errorf("spaces(3) = %q", s)
	}
}
