package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/walter/p/internal/config"
)

// ViewState captures the state needed to restore a view when navigating back.
type ViewState struct {
	model       tea.Model
	projectName string
	projectDir  string
	listName    string
}

// App is the root bubbletea model that manages view navigation and shared state.
type App struct {
	// Navigation stack for back navigation
	viewStack []ViewState

	// Shared state
	config      config.Config
	projectRoot string

	// Current context
	projectName string
	projectDir  string
	listName    string

	// Active view model
	activeView tea.Model

	// Window dimensions
	width  int
	height int

	// Help overlay
	showHelp bool

	// Status/error display with auto-clear
	statusMsg string
	errorMsg  string
	statusID  int // monotonic counter for clear timer dedup

	// Quitting flag
	quitting bool
}

// NewApp creates a new App model with the given configuration.
func NewApp(cfg config.Config) *App {
	return &App{
		config:      cfg,
		projectRoot: cfg.ProjectRoot,
	}
}

// StartAtProjectList initializes the app to show the project list.
func (a *App) StartAtProjectList() {
	a.activeView = NewProjectListView(a.projectRoot, a.width, a.contentHeight())
}

// StartAtProject initializes the app to show a specific project's todo lists.
func (a *App) StartAtProject(projectName, projectDir string) {
	a.projectName = projectName
	a.projectDir = projectDir
	a.activeView = NewTodoListView(projectName, projectDir, a.width, a.contentHeight())
}

// StartAtList initializes the app to show a specific todo list's items.
func (a *App) StartAtList(projectName, projectDir, listName string) {
	a.projectName = projectName
	a.projectDir = projectDir
	a.listName = listName
	a.activeView = NewItemListView(projectName, projectDir, listName, a.width, a.contentHeight())
}

func (a *App) contentHeight() int {
	h := a.height - 2 // reserve space for status bar
	if h < 5 {
		h = 5
	}
	return h
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	if a.activeView == nil {
		a.StartAtProjectList()
	}
	go prewarmGlamourRenderer()
	return a.activeView.Init()
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Forward resize to active view
		if a.activeView != nil {
			var cmd tea.Cmd
			a.activeView, cmd = a.activeView.Update(msg)
			return a, cmd
		}
		return a, nil

	case tea.KeyMsg:
		// Help overlay: any key closes it
		if a.showHelp {
			a.showHelp = false
			return a, nil
		}

		// ctrl+c always quits
		if msg.String() == "ctrl+c" {
			a.quitting = true
			return a, tea.Quit
		}

		// Check if view is in input mode — if so, don't intercept q/?
		inInput := false
		if iv, ok := a.activeView.(interface{ IsInputMode() bool }); ok {
			inInput = iv.IsInputMode()
		}

		if !inInput {
			switch msg.String() {
			case "q":
				a.quitting = true
				return a, tea.Quit
			case "?":
				a.showHelp = !a.showHelp
				return a, nil
			case "S":
				// Jump to status view from any context
				return a, func() tea.Msg {
					return NavigateMsg{To: ViewStatus}
				}
			case "T":
				// Cycle through theme presets
				return a, a.cycleTheme()
			}
		}

		// Forward all other keys to active view
		if a.activeView != nil {
			var cmd tea.Cmd
			a.activeView, cmd = a.activeView.Update(msg)
			return a, cmd
		}

	case NavigateMsg:
		return a, a.navigate(msg)

	case GoBackMsg:
		return a, a.goBack()

	case ErrorMsg:
		a.errorMsg = msg.Err.Error()
		a.statusMsg = ""
		a.statusID++
		id := a.statusID
		return a, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return ClearStatusMsg{ID: id}
		})

	case StatusMsg:
		a.statusMsg = msg.Text
		a.errorMsg = ""
		a.statusID++
		id := a.statusID
		return a, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return ClearStatusMsg{ID: id}
		})

	case ClearStatusMsg:
		if msg.ID == a.statusID {
			a.statusMsg = ""
			a.errorMsg = ""
		}
		return a, nil

	default:
		// Forward other messages (e.g. data loaded) to active view
		if a.activeView != nil {
			var cmd tea.Cmd
			a.activeView, cmd = a.activeView.Update(msg)
			return a, cmd
		}
	}

	return a, nil
}

// cycleTheme switches to the next theme preset and applies it immediately.
func (a *App) cycleTheme() tea.Cmd {
	names := ThemePresetNames
	if len(names) == 0 || ThemeApplyFunc == nil {
		return func() tea.Msg {
			return StatusMsg{Text: "Theme cycling not available"}
		}
	}

	// Find current theme index
	current := a.config.Theme
	if current == "" {
		current = "default"
	}
	idx := 0
	for i, name := range names {
		if name == current {
			idx = i
			break
		}
	}

	// Cycle to next
	next := names[(idx+1)%len(names)]
	a.config.Theme = next

	// Apply the theme — styles update immediately, next View() picks them up
	ThemeApplyFunc(a.config)

	// Invalidate cached glamour renderer so it picks up new glamour theme
	glamourMu.Lock()
	glamourRenderer = nil
	glamourMu.Unlock()

	return func() tea.Msg {
		return StatusMsg{Text: "Theme: " + next}
	}
}

func (a *App) navigate(msg NavigateMsg) tea.Cmd {
	// Push current state onto the view stack
	if a.activeView != nil {
		a.viewStack = append(a.viewStack, ViewState{
			model:       a.activeView,
			projectName: a.projectName,
			projectDir:  a.projectDir,
			listName:    a.listName,
		})
	}

	// Update navigation context
	if msg.ProjectName != "" {
		a.projectName = msg.ProjectName
	}
	if msg.ProjectDir != "" {
		a.projectDir = msg.ProjectDir
	}
	if msg.ListName != "" {
		a.listName = msg.ListName
	}

	// Clear messages on navigation
	a.statusMsg = ""
	a.errorMsg = ""

	ch := a.contentHeight()

	switch msg.To {
	case ViewProjectList:
		a.activeView = NewProjectListView(a.projectRoot, a.width, ch)
	case ViewTodoList:
		// Check if we're Tab-switching from KnowledgeListView — if so,
		// pop the stack instead of pushing a new view so we preserve state.
		if a.restoreFromStack(ViewTodoList) {
			// Send DataChanged to refresh data while keeping cursor position
			var cmd tea.Cmd
			a.activeView, cmd = a.activeView.Update(DataChangedMsg{})
			return cmd
		}
		a.activeView = NewTodoListView(a.projectName, a.projectDir, a.width, ch)
	case ViewItemList:
		a.activeView = NewItemListView(a.projectName, a.projectDir, a.listName, a.width, ch)
	case ViewItemDetail:
		a.activeView = NewItemDetailView(a.projectName, a.projectDir, a.listName, msg.ItemID, a.width, ch)
	case ViewKnowledgeList:
		// Check if we're Tab-switching from TodoListView — if so,
		// pop the stack instead of pushing a new view so we preserve state.
		if a.restoreFromStack(ViewKnowledgeList) {
			// Send DataChanged to refresh data while keeping cursor position
			var cmd tea.Cmd
			a.activeView, cmd = a.activeView.Update(DataChangedMsg{})
			return cmd
		}
		a.activeView = NewKnowledgeListView(a.projectName, a.projectDir, a.width, ch)
	case ViewKnowledgeView:
		a.activeView = NewKnowledgeView(a.projectName, a.projectDir, msg.DocName, a.width, ch)
	case ViewSearch:
		a.activeView = NewSearchView(a.projectName, a.projectDir, a.width, ch)
	case ViewStatus:
		a.activeView = NewStatusView(a.projectRoot, a.projectName, a.projectDir, a.width, ch)
	default:
		return nil
	}

	return a.activeView.Init()
}

func (a *App) goBack() tea.Cmd {
	if len(a.viewStack) == 0 {
		a.quitting = true
		return tea.Quit
	}

	// Pop from view stack
	prev := a.viewStack[len(a.viewStack)-1]
	a.viewStack = a.viewStack[:len(a.viewStack)-1]

	a.activeView = prev.model
	a.projectName = prev.projectName
	a.projectDir = prev.projectDir
	a.listName = prev.listName

	// Clear messages
	a.statusMsg = ""
	a.errorMsg = ""

	// Reload data and resize the restored view
	var cmds []tea.Cmd
	if a.width > 0 && a.height > 0 {
		var cmd tea.Cmd
		a.activeView, cmd = a.activeView.Update(tea.WindowSizeMsg{
			Width: a.width, Height: a.height,
		})
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Trigger a data reload so restored view shows fresh data
	var cmd tea.Cmd
	a.activeView, cmd = a.activeView.Update(DataChangedMsg{})
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

// View implements tea.Model.
func (a *App) View() string {
	if a.quitting {
		return ""
	}

	if a.showHelp {
		return a.renderHelp()
	}

	var content string
	if a.activeView != nil {
		content = a.activeView.View()
	}

	// Status bar — always present to avoid layout shift when messages appear.
	// The contentHeight() reserves 2 lines for this area.
	if a.errorMsg != "" {
		content += "\n" + ErrorStyle.Render("  ⚠ "+a.errorMsg)
	} else if a.statusMsg != "" {
		content += "\n" + StatusStyle.Render("  "+a.statusMsg)
	} else {
		content += "\n"
	}

	return content
}

func (a *App) renderHelp() string {
	vt := activeViewType(a.activeView)
	help := renderContextHelp(vt)

	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center,
		BorderStyle.Render(help))
}

// restoreFromStack searches the view stack for a matching view type and
// restores it if found, removing it from the stack. This is used for Tab
// switching between TodoListView and KnowledgeListView so that each view's
// state is preserved across switches.
func (a *App) restoreFromStack(viewType ViewType) bool {
	for i := len(a.viewStack) - 1; i >= 0; i-- {
		vs := a.viewStack[i]
		var isMatch bool
		switch viewType {
		case ViewTodoList:
			_, isMatch = vs.model.(*TodoListView)
		case ViewKnowledgeList:
			_, isMatch = vs.model.(*KnowledgeListView)
		}
		if isMatch {
			// Restore this view and remove it from the stack
			a.activeView = vs.model
			a.projectName = vs.projectName
			a.projectDir = vs.projectDir
			a.listName = vs.listName
			// Remove from stack
			a.viewStack = append(a.viewStack[:i], a.viewStack[i+1:]...)
			// Resize
			if a.width > 0 && a.height > 0 {
				a.activeView, _ = a.activeView.Update(tea.WindowSizeMsg{
					Width: a.width, Height: a.height,
				})
			}
			return true
		}
	}
	return false
}

// ctx returns a background context for service operations.
func ctx() context.Context {
	return context.Background()
}
