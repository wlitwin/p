package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/lock"
	"github.com/walter/p/internal/service"
	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/validate"
)

// TodoListView displays all todo lists in a project with item counts.
type TodoListView struct {
	projectName string
	projectDir  string
	lists       []TodoListInfo
	cursor      int
	width       int
	height      int
	loaded      bool

	// Inline text input for creating new lists
	inputMode   bool
	inputPrompt string
	inputValue  string
	inputAction func(value string) tea.Cmd

	// Confirmation prompt for destructive actions
	confirmMode   bool
	confirmPrompt string
	confirmAction func() tea.Cmd
}

// NewTodoListView creates a new todo list view for the given project.
func NewTodoListView(projectName, projectDir string, width, height int) *TodoListView {
	return &TodoListView{
		projectName: projectName,
		projectDir:  projectDir,
		width:       width,
		height:      height,
	}
}

func (v *TodoListView) Init() tea.Cmd {
	return v.loadLists()
}

func (v *TodoListView) loadLists() tea.Cmd {
	dir := v.projectDir
	return func() tea.Msg {
		statuses, err := service.GetProjectListStatuses(ctx(), dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("loading lists: %w", err)}
		}

		var lists []TodoListInfo
		for _, s := range statuses {
			lists = append(lists, TodoListInfo{
				Name:    s.Name,
				Open:    s.Open,
				Done:    s.Done,
				Blocked: s.Blocked,
			})
		}
		return TodoListsLoadedMsg{Lists: lists}
	}
}

// IsInputMode reports whether the view is in an interactive mode.
func (v *TodoListView) IsInputMode() bool {
	return v.inputMode || v.confirmMode
}

func (v *TodoListView) selectedName() string {
	if v.cursor >= 0 && v.cursor < len(v.lists) {
		return v.lists[v.cursor].Name
	}
	return ""
}

func (v *TodoListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		return v, nil

	case TodoListsLoadedMsg:
		v.lists = msg.Lists
		v.loaded = true
		if v.cursor >= len(v.lists) {
			v.cursor = max(0, len(v.lists)-1)
		}
		return v, nil

	case DataChangedMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, v.loadLists())
		if msg.StatusText != "" {
			text := msg.StatusText
			cmds = append(cmds, func() tea.Msg { return StatusMsg{Text: text} })
		}
		return v, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Confirmation mode
		if v.confirmMode {
			return v.handleConfirm(msg)
		}

		// Input mode
		if v.inputMode {
			return v.handleInput(msg)
		}

		switch {
		case key.Matches(msg, GlobalKeyMap.Back):
			return v, func() tea.Msg { return GoBackMsg{} }

		case key.Matches(msg, NavKeyMap.Up):
			if v.cursor > 0 {
				v.cursor--
			}

		case key.Matches(msg, NavKeyMap.Down):
			if v.cursor < len(v.lists)-1 {
				v.cursor++
			}

		case key.Matches(msg, NavKeyMap.HalfDown):
			pageSize := max(1, (v.height-6)/2)
			v.cursor += pageSize
			if v.cursor >= len(v.lists) {
				v.cursor = max(0, len(v.lists)-1)
			}
		case key.Matches(msg, NavKeyMap.HalfUp):
			pageSize := max(1, (v.height-6)/2)
			v.cursor -= pageSize
			if v.cursor < 0 {
				v.cursor = 0
			}
		case key.Matches(msg, NavKeyMap.Bottom):
			v.cursor = max(0, len(v.lists)-1)
		case msg.String() == "g":
			v.cursor = 0

		case key.Matches(msg, NavKeyMap.Enter):
			if len(v.lists) > 0 && v.cursor < len(v.lists) {
				l := v.lists[v.cursor]
				return v, func() tea.Msg {
					return NavigateMsg{
						To:       ViewItemList,
						ListName: l.Name,
					}
				}
			}

		case key.Matches(msg, TodoListKeyMap.Knowledge):
			return v, func() tea.Msg {
				return NavigateMsg{To: ViewKnowledgeList}
			}

		case key.Matches(msg, GlobalKeyMap.Search):
			return v, func() tea.Msg {
				return NavigateMsg{To: ViewSearch}
			}

		case key.Matches(msg, TodoListKeyMap.New):
			v.inputMode = true
			v.inputPrompt = "New list name: "
			v.inputValue = ""
			v.inputAction = v.createList

		case key.Matches(msg, TodoListKeyMap.Delete):
			name := v.selectedName()
			if name != "" {
				v.confirmMode = true
				v.confirmPrompt = fmt.Sprintf("Delete list %q? (y/n)", name)
				v.confirmAction = func() tea.Cmd {
					return v.doDeleteList(name)
				}
			}

		case key.Matches(msg, TodoListKeyMap.Archive):
			name := v.selectedName()
			if name != "" {
				v.confirmMode = true
				v.confirmPrompt = fmt.Sprintf("Archive list %q? (y/n)", name)
				v.confirmAction = func() tea.Cmd {
					return v.doArchiveList(name)
				}
			}
		}
	}

	return v, nil
}

func (v *TodoListView) handleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.inputMode = false
		v.inputValue = ""
		return v, nil
	case "enter":
		v.inputMode = false
		value := v.inputValue
		action := v.inputAction
		v.inputValue = ""
		v.inputAction = nil
		if value != "" && action != nil {
			return v, action(value)
		}
		return v, nil
	case "backspace":
		if len(v.inputValue) > 0 {
			v.inputValue = v.inputValue[:len(v.inputValue)-1]
		}
		return v, nil
	default:
		if len(msg.String()) == 1 {
			v.inputValue += msg.String()
		}
		return v, nil
	}
}

func (v *TodoListView) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		v.confirmMode = false
		action := v.confirmAction
		v.confirmPrompt = ""
		v.confirmAction = nil
		if action != nil {
			return v, action()
		}
		return v, nil
	case "n", "N", "esc":
		v.confirmMode = false
		v.confirmPrompt = ""
		v.confirmAction = nil
		return v, nil
	}
	return v, nil
}

func (v *TodoListView) createList(name string) tea.Cmd {
	dir := v.projectDir
	return func() tea.Msg {
		if err := validate.ListName(name); err != nil {
			return ErrorMsg{Err: err}
		}

		title := strings.ReplaceAll(name, "-", " ")
		if _, err := todo.CreateList(dir, name, title); err != nil {
			return ErrorMsg{Err: fmt.Errorf("creating list: %w", err)}
		}

		_ = git.CommitAll(context.Background(), dir, fmt.Sprintf("tui: create list %s", name))
		return DataChangedMsg{StatusText: fmt.Sprintf("Created list %s", name)}
	}
}

func (v *TodoListView) doDeleteList(name string) tea.Cmd {
	dir := v.projectDir
	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		if err := service.RemoveList(ctx(), dir, name); err != nil {
			return ErrorMsg{Err: err}
		}

		_ = git.CommitAll(context.Background(), dir, fmt.Sprintf("tui: delete list %s", name))
		return DataChangedMsg{StatusText: fmt.Sprintf("Deleted list %s", name)}
	}
}

func (v *TodoListView) doArchiveList(name string) tea.Cmd {
	dir := v.projectDir
	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		activePath := todo.ListPath(dir, name)
		archiveDir := filepath.Join(todo.ListDir(dir), ".archive")
		archivedPath := filepath.Join(archiveDir, name+".md")

		if _, err := os.Stat(activePath); err != nil {
			return ErrorMsg{Err: fmt.Errorf("list %q not found", name)}
		}
		if err := os.MkdirAll(filepath.Dir(archivedPath), 0o755); err != nil {
			return ErrorMsg{Err: fmt.Errorf("creating archive dir: %w", err)}
		}
		if err := os.Rename(activePath, archivedPath); err != nil {
			return ErrorMsg{Err: fmt.Errorf("archiving: %w", err)}
		}
		todo.CleanEmptyParents(activePath, todo.ListDir(dir))

		_ = git.CommitAll(context.Background(), dir, fmt.Sprintf("tui: archive list %s", name))
		return DataChangedMsg{StatusText: fmt.Sprintf("Archived list %s", name)}
	}
}

func (v *TodoListView) View() string {
	title := TitleStyle.Render(v.projectName) + HelpStyle.Render(" · Todo Lists")

	if !v.loaded {
		return title + "\n\n" + HelpStyle.Render("  Loading...")
	}

	if len(v.lists) == 0 {
		s := title + "\n\n" + HelpStyle.Render("  No todo lists found. Press 'n' to create one.")
		s += "\n\n" + HelpStyle.Render("  n new  Tab knowledge  / search  q quit  ? help")
		return s
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n\n")

	// Calculate visible area for scrolling
	visibleHeight := max(3, v.height-5)

	scrollStart := 0
	if v.cursor >= visibleHeight {
		scrollStart = v.cursor - visibleHeight + 1
	}
	scrollEnd := min(scrollStart+visibleHeight, len(v.lists))

	// Adapt name column width to terminal width
	nameWidth := 24
	if v.width > 0 && v.width < 60 {
		nameWidth = max(12, v.width/3)
	}

	for i := scrollStart; i < scrollEnd; i++ {
		l := v.lists[i]

		cursor := "  "
		if v.cursor == i {
			cursor = CursorStyle.Render("▸ ")
		}

		counts := formatCounts(l.Open, l.Done, l.Blocked)

		name := l.Name
		displayName := name
		if len(displayName) > nameWidth {
			displayName = displayName[:nameWidth-1] + "…"
		}
		if v.cursor == i {
			displayName = SelectedStyle.Render(displayName)
		}

		padding := max(2, nameWidth-len(name))

		fmt.Fprintf(&sb, "%s%s%s%s\n", cursor, displayName, spaces(padding), counts)
	}

	// Bottom bar
	if v.inputMode {
		cursorChar := CursorStyle.Render("█")
		sb.WriteString("\n" + HelpStyle.Render("  "+v.inputPrompt) + v.inputValue + cursorChar)
	} else if v.confirmMode {
		sb.WriteString("\n" + ErrorStyle.Render("  "+v.confirmPrompt))
	} else {
		sb.WriteString("\n" + HelpStyle.Render("  ↑↓/jk nav  ^D/^U page  Enter open  n new  d del  a archive  Tab knowledge  ? help"))
	}

	return sb.String()
}
