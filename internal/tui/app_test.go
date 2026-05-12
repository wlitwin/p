package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/walter/p/internal/config"
)

// --- Helpers ---

// sendKey simulates a key press to the App model.
func sendKey(app *App, key string) tea.Cmd {
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return cmd
}

// sendMsg sends a tea.Msg to the App model and returns the resulting command.
func sendMsg(app *App, msg tea.Msg) tea.Cmd {
	_, cmd := app.Update(msg)
	return cmd
}

// runCmd executes a tea.Cmd and returns the resulting message, or nil.
func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// newTestApp creates an App with a temporary project root for testing.
func newTestApp(root string) *App {
	cfg := config.Config{
		ProjectRoot:     root,
		DefaultPriority: "now",
	}
	return NewApp(cfg)
}

// --- App Model Tests ---

func TestNewApp(t *testing.T) {
	cfg := config.Config{
		ProjectRoot:     "/tmp/test",
		DefaultPriority: "now",
	}
	app := NewApp(cfg)

	if app.projectRoot != "/tmp/test" {
		t.Errorf("projectRoot = %q, want %q", app.projectRoot, "/tmp/test")
	}
	if app.activeView != nil {
		t.Error("activeView should be nil before Init()")
	}
	if app.showHelp {
		t.Error("showHelp should be false")
	}
	if app.quitting {
		t.Error("quitting should be false")
	}
	if len(app.viewStack) != 0 {
		t.Error("viewStack should be empty")
	}
}

func TestAppInit(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.Init()

	if app.activeView == nil {
		t.Fatal("activeView should be set after Init()")
	}
	if _, ok := app.activeView.(*ProjectListView); !ok {
		t.Errorf("activeView should be *ProjectListView, got %T", app.activeView)
	}
}

func TestAppStartAtProjectList(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	if _, ok := app.activeView.(*ProjectListView); !ok {
		t.Errorf("activeView should be *ProjectListView, got %T", app.activeView)
	}
}

func TestAppStartAtProject(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProject("myproj", "/tmp/myproj")

	if app.projectName != "myproj" {
		t.Errorf("projectName = %q", app.projectName)
	}
	if app.projectDir != "/tmp/myproj" {
		t.Errorf("projectDir = %q", app.projectDir)
	}
	if _, ok := app.activeView.(*TodoListView); !ok {
		t.Errorf("activeView should be *TodoListView, got %T", app.activeView)
	}
}

func TestAppStartAtList(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtList("myproj", "/tmp/myproj", "backlog")

	if app.projectName != "myproj" {
		t.Errorf("projectName = %q", app.projectName)
	}
	if app.listName != "backlog" {
		t.Errorf("listName = %q", app.listName)
	}
	if _, ok := app.activeView.(*ItemListView); !ok {
		t.Errorf("activeView should be *ItemListView, got %T", app.activeView)
	}
}

func TestAppContentHeight(t *testing.T) {
	app := newTestApp(t.TempDir())

	// Zero height should floor to 5
	app.height = 0
	if h := app.contentHeight(); h != 5 {
		t.Errorf("contentHeight() = %d with height=0, want 5", h)
	}

	// Normal height
	app.height = 24
	if h := app.contentHeight(); h != 22 {
		t.Errorf("contentHeight() = %d with height=24, want 22", h)
	}

	// Small height
	app.height = 4
	if h := app.contentHeight(); h != 5 {
		t.Errorf("contentHeight() = %d with height=4, want 5 (minimum)", h)
	}
}

// --- Navigation Tests ---

func TestNavigateMsg_PushesViewStack(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	if len(app.viewStack) != 0 {
		t.Fatal("viewStack should start empty")
	}

	// Navigate to a TodoListView
	msg := NavigateMsg{
		To:          ViewTodoList,
		ProjectName: "proj1",
		ProjectDir:  "/tmp/proj1",
	}
	sendMsg(app, msg)

	if len(app.viewStack) != 1 {
		t.Fatalf("viewStack should have 1 entry after navigate, got %d", len(app.viewStack))
	}
	if _, ok := app.activeView.(*TodoListView); !ok {
		t.Errorf("activeView should be *TodoListView, got %T", app.activeView)
	}
	if app.projectName != "proj1" {
		t.Errorf("projectName = %q, want proj1", app.projectName)
	}
}

func TestNavigateMsg_UpdatesContext(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	// Navigate sets project context
	sendMsg(app, NavigateMsg{
		To:          ViewTodoList,
		ProjectName: "test-proj",
		ProjectDir:  "/tmp/test-proj",
	})
	if app.projectName != "test-proj" {
		t.Errorf("projectName = %q", app.projectName)
	}
	if app.projectDir != "/tmp/test-proj" {
		t.Errorf("projectDir = %q", app.projectDir)
	}

	// Navigate sets list context
	sendMsg(app, NavigateMsg{
		To:       ViewItemList,
		ListName: "backlog",
	})
	if app.listName != "backlog" {
		t.Errorf("listName = %q", app.listName)
	}
}

func TestGoBackMsg_PopsViewStack(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	// Navigate forward twice
	sendMsg(app, NavigateMsg{
		To:          ViewTodoList,
		ProjectName: "proj1",
		ProjectDir:  "/tmp/proj1",
	})
	sendMsg(app, NavigateMsg{
		To:       ViewItemList,
		ListName: "backlog",
	})

	if len(app.viewStack) != 2 {
		t.Fatalf("viewStack should have 2 entries, got %d", len(app.viewStack))
	}

	// Go back once
	sendMsg(app, GoBackMsg{})

	if len(app.viewStack) != 1 {
		t.Errorf("viewStack should have 1 entry after back, got %d", len(app.viewStack))
	}
	if _, ok := app.activeView.(*TodoListView); !ok {
		t.Errorf("activeView should be *TodoListView after back, got %T", app.activeView)
	}

	// Go back again — should be at ProjectListView
	sendMsg(app, GoBackMsg{})

	if len(app.viewStack) != 0 {
		t.Errorf("viewStack should be empty, got %d", len(app.viewStack))
	}
	if _, ok := app.activeView.(*ProjectListView); !ok {
		t.Errorf("activeView should be *ProjectListView after back, got %T", app.activeView)
	}
}

func TestGoBackMsg_EmptyStack_Quits(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	// Go back from root view
	cmd := sendMsg(app, GoBackMsg{})

	if !app.quitting {
		t.Error("should set quitting=true when stack is empty")
	}
	// Should return tea.Quit
	if cmd == nil {
		t.Error("should return tea.Quit cmd")
	}
}

func TestGoBackMsg_RestoresContext(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	sendMsg(app, NavigateMsg{
		To:          ViewTodoList,
		ProjectName: "proj1",
		ProjectDir:  "/tmp/proj1",
	})

	// Update context during navigation
	sendMsg(app, NavigateMsg{
		To:          ViewItemList,
		ProjectName: "proj2",
		ProjectDir:  "/tmp/proj2",
		ListName:    "sprint",
	})

	// Go back — should restore proj1 context
	sendMsg(app, GoBackMsg{})

	if app.projectName != "proj1" {
		t.Errorf("projectName = %q, want proj1 (restored)", app.projectName)
	}
	if app.projectDir != "/tmp/proj1" {
		t.Errorf("projectDir = %q (restored)", app.projectDir)
	}
}

// --- View Type Navigation ---

func TestNavigateToAllViewTypes(t *testing.T) {
	root := t.TempDir()
	app := newTestApp(root)
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	tests := []struct {
		name    string
		msg     NavigateMsg
		wantTyp string
	}{
		{"TodoList", NavigateMsg{To: ViewTodoList, ProjectName: "p", ProjectDir: root}, "*tui.TodoListView"},
		{"ItemList", NavigateMsg{To: ViewItemList, ListName: "backlog"}, "*tui.ItemListView"},
		{"ItemDetail", NavigateMsg{To: ViewItemDetail, ItemID: "1"}, "*tui.ItemDetailView"},
		{"KnowledgeList", NavigateMsg{To: ViewKnowledgeList}, "*tui.KnowledgeListView"},
		{"KnowledgeView", NavigateMsg{To: ViewKnowledgeView, DocName: "test"}, "*tui.KnowledgeView"},
		{"Search", NavigateMsg{To: ViewSearch}, "*tui.SearchView"},
		{"Status", NavigateMsg{To: ViewStatus}, "*tui.StatusView"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sendMsg(app, tt.msg)
			got := typeName(app.activeView)
			if got != tt.wantTyp {
				t.Errorf("activeView type = %s, want %s", got, tt.wantTyp)
			}
		})
	}
}

func typeName(v any) string {
	if v == nil {
		return "<nil>"
	}
	return "*tui." + typeShortName(v)
}

func typeShortName(v any) string {
	switch v.(type) {
	case *ProjectListView:
		return "ProjectListView"
	case *TodoListView:
		return "TodoListView"
	case *ItemListView:
		return "ItemListView"
	case *ItemDetailView:
		return "ItemDetailView"
	case *KnowledgeListView:
		return "KnowledgeListView"
	case *KnowledgeView:
		return "KnowledgeView"
	case *SearchView:
		return "SearchView"
	case *StatusView:
		return "StatusView"
	default:
		return "unknown"
	}
}

// --- Window Resize Tests ---

func TestWindowResize_PropagatedToActiveView(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.StartAtProjectList()

	sendMsg(app, tea.WindowSizeMsg{Width: 120, Height: 40})

	if app.width != 120 {
		t.Errorf("width = %d, want 120", app.width)
	}
	if app.height != 40 {
		t.Errorf("height = %d, want 40", app.height)
	}

	// Verify active view also received the resize
	plv, ok := app.activeView.(*ProjectListView)
	if !ok {
		t.Fatal("expected ProjectListView")
	}
	if plv.width != 120 || plv.height != 40 {
		t.Errorf("ProjectListView got width=%d height=%d, want 120/40", plv.width, plv.height)
	}
}

func TestWindowResize_PropagatedOnGoBack(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	// Navigate forward
	sendMsg(app, NavigateMsg{
		To:          ViewTodoList,
		ProjectName: "p",
		ProjectDir:  t.TempDir(),
	})

	// Resize while on TodoListView
	sendMsg(app, tea.WindowSizeMsg{Width: 160, Height: 50})

	// Go back to ProjectListView
	sendMsg(app, GoBackMsg{})

	// Restored view should have the new dimensions
	plv := app.activeView.(*ProjectListView)
	if plv.width != 160 || plv.height != 50 {
		t.Errorf("restored view got width=%d height=%d, want 160/50", plv.width, plv.height)
	}
}

// --- Keyboard Handling ---

func TestQuitKey(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.StartAtProjectList()

	sendKey(app, "q")

	if !app.quitting {
		t.Error("pressing 'q' should set quitting=true")
	}
}

func TestCtrlCQuit(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.StartAtProjectList()

	sendMsg(app, tea.KeyMsg{Type: tea.KeyCtrlC})

	if !app.quitting {
		t.Error("Ctrl+C should set quitting=true")
	}
}

func TestHelpToggle(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.StartAtProjectList()

	// Open help
	sendKey(app, "?")
	if !app.showHelp {
		t.Error("'?' should open help")
	}

	// Close help with any key
	sendKey(app, "a")
	if app.showHelp {
		t.Error("any key should close help")
	}
}

func TestHelpView_RendersContent(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	app.showHelp = true
	view := app.View()

	if view == "" {
		t.Error("help view should not be empty")
	}
}

func TestQKey_NotInterceptedDuringInput(t *testing.T) {
	root := t.TempDir()
	app := newTestApp(root)
	app.width = 80
	app.height = 24
	app.StartAtList("proj", root, "backlog")

	// Simulate the view being in input mode by manually setting it
	ilv := app.activeView.(*ItemListView)
	ilv.inputMode = true

	sendKey(app, "q")

	if app.quitting {
		t.Error("'q' should not quit during input mode")
	}
}

// --- Status/Error Messages ---

func TestErrorMsg_DisplayedInView(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	sendMsg(app, ErrorMsg{Err: errors.New("test error")})

	if app.errorMsg != "test error" {
		t.Errorf("errorMsg = %q, want %q", app.errorMsg, "test error")
	}
	if app.statusMsg != "" {
		t.Errorf("statusMsg should be cleared, got %q", app.statusMsg)
	}

	view := app.View()
	if view == "" {
		t.Error("view should render error message")
	}
}

func TestStatusMsg_DisplayedInView(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	sendMsg(app, StatusMsg{Text: "Item marked done"})

	if app.statusMsg != "Item marked done" {
		t.Errorf("statusMsg = %q", app.statusMsg)
	}
	if app.errorMsg != "" {
		t.Errorf("errorMsg should be cleared, got %q", app.errorMsg)
	}
}

func TestClearStatusMsg_ClearsMatchingID(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.StartAtProjectList()

	// Set a status message — increments statusID
	sendMsg(app, StatusMsg{Text: "hello"})
	currentID := app.statusID

	// Clear with matching ID
	sendMsg(app, ClearStatusMsg{ID: currentID})

	if app.statusMsg != "" {
		t.Errorf("statusMsg should be cleared, got %q", app.statusMsg)
	}
}

func TestClearStatusMsg_IgnoresStaleID(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.StartAtProjectList()

	// Set a status
	sendMsg(app, StatusMsg{Text: "first"})
	firstID := app.statusID

	// Set another status — increments ID
	sendMsg(app, StatusMsg{Text: "second"})

	// Try to clear with the old ID
	sendMsg(app, ClearStatusMsg{ID: firstID})

	// Should still show "second" since IDs didn't match
	if app.statusMsg != "second" {
		t.Errorf("statusMsg = %q, want %q (stale clear should be ignored)", app.statusMsg, "second")
	}
}

func TestNavigate_ClearsMessages(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	// Set an error
	app.errorMsg = "some error"
	app.statusMsg = "some status"

	// Navigate
	sendMsg(app, NavigateMsg{
		To:          ViewTodoList,
		ProjectName: "p",
		ProjectDir:  t.TempDir(),
	})

	if app.errorMsg != "" {
		t.Errorf("errorMsg should be cleared on navigate, got %q", app.errorMsg)
	}
	if app.statusMsg != "" {
		t.Errorf("statusMsg should be cleared on navigate, got %q", app.statusMsg)
	}
}

func TestGoBack_ClearsMessages(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	sendMsg(app, NavigateMsg{
		To:          ViewTodoList,
		ProjectName: "p",
		ProjectDir:  t.TempDir(),
	})

	app.errorMsg = "some error"
	app.statusMsg = "some status"

	sendMsg(app, GoBackMsg{})

	if app.errorMsg != "" || app.statusMsg != "" {
		t.Error("messages should be cleared on go back")
	}
}

// --- Quitting Renders Empty ---

func TestQuittingView(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.StartAtProjectList()
	app.quitting = true

	view := app.View()
	if view != "" {
		t.Errorf("quitting view should be empty, got %q", view)
	}
}

// --- StatusView Key: 'S' ---

func TestStatusKey_NavigatesToStatus(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	cmd := sendKey(app, "S")
	if cmd == nil {
		t.Fatal("'S' should return a command")
	}

	msg := runCmd(cmd)
	navMsg, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}
	if navMsg.To != ViewStatus {
		t.Errorf("NavigateMsg.To = %d, want ViewStatus", navMsg.To)
	}
}

// --- activeViewType helper ---

func TestActiveViewType(t *testing.T) {
	tests := []struct {
		model    tea.Model
		wantType ViewType
	}{
		{&ProjectListView{}, ViewProjectList},
		{&TodoListView{}, ViewTodoList},
		{&ItemListView{}, ViewItemList},
		{&ItemDetailView{}, ViewItemDetail},
		{&KnowledgeListView{}, ViewKnowledgeList},
		{&KnowledgeView{}, ViewKnowledgeView},
		{&SearchView{}, ViewSearch},
		{&StatusView{}, ViewStatus},
	}

	for _, tt := range tests {
		got := activeViewType(tt.model)
		if got != tt.wantType {
			t.Errorf("activeViewType(%T) = %d, want %d", tt.model, got, tt.wantType)
		}
	}
}

// --- View Stack Restore for Tab Switching ---

func TestRestoreFromStack_TabSwitching(t *testing.T) {
	app := newTestApp(t.TempDir())
	app.width = 80
	app.height = 24
	app.StartAtProjectList()

	// Navigate to TodoListView
	sendMsg(app, NavigateMsg{
		To:          ViewTodoList,
		ProjectName: "proj",
		ProjectDir:  "/tmp/proj",
	})
	if len(app.viewStack) != 1 {
		t.Fatalf("stack size = %d, want 1", len(app.viewStack))
	}

	// Navigate to KnowledgeListView (Tab from TodoListView)
	sendMsg(app, NavigateMsg{
		To: ViewKnowledgeList,
	})

	// Stack should now have 2 items (ProjectList + TodoList)
	if len(app.viewStack) != 2 {
		t.Fatalf("stack size = %d after navigate to knowledge, want 2", len(app.viewStack))
	}

	// Navigate back to TodoListView via Tab — should restore from stack
	sendMsg(app, NavigateMsg{
		To: ViewTodoList,
	})

	if _, ok := app.activeView.(*TodoListView); !ok {
		t.Errorf("should restore TodoListView from stack, got %T", app.activeView)
	}
	// The TodoListView entry should have been removed from stack
	for _, vs := range app.viewStack {
		if _, ok := vs.model.(*TodoListView); ok {
			t.Error("restored view should be removed from stack")
		}
	}
}
