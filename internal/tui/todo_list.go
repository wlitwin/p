package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/walter/p/internal/service"
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
		return v, v.loadLists()

	case tea.KeyMsg:
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
		}
	}

	return v, nil
}

func (v *TodoListView) View() string {
	title := TitleStyle.Render(v.projectName) + HelpStyle.Render(" · Todo Lists")

	if !v.loaded {
		return title + "\n\n" + HelpStyle.Render("  Loading...")
	}

	if len(v.lists) == 0 {
		return title + "\n\n" + HelpStyle.Render("  No todo lists found. Use 'p add' to create one.")
	}

	s := title + "\n\n"

	// Calculate visible area for scrolling
	visibleHeight := v.height - 5
	if visibleHeight < 3 {
		visibleHeight = 3
	}

	scrollStart := 0
	if v.cursor >= visibleHeight {
		scrollStart = v.cursor - visibleHeight + 1
	}
	scrollEnd := scrollStart + visibleHeight
	if scrollEnd > len(v.lists) {
		scrollEnd = len(v.lists)
	}

	for i := scrollStart; i < scrollEnd; i++ {
		l := v.lists[i]

		cursor := "  "
		if v.cursor == i {
			cursor = CursorStyle.Render("▸ ")
		}

		counts := formatCounts(l.Open, l.Done, l.Blocked)

		name := l.Name
		if v.cursor == i {
			name = SelectedStyle.Render(name)
		}

		padding := 24 - len(l.Name)
		if padding < 2 {
			padding = 2
		}
		pad := spaces(padding)

		s += fmt.Sprintf("%s%s%s%s\n", cursor, name, pad, counts)
	}

	s += "\n" + HelpStyle.Render("  ↑↓ navigate  Enter open  Tab knowledge  q quit  ? help")

	return s
}
