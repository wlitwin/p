package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/service"
)

// ProjectListView displays all non-archived projects with summary statistics.
type ProjectListView struct {
	projectRoot string
	projects    []ProjectInfo
	cursor      int
	width       int
	height      int
	loaded      bool
}

// NewProjectListView creates a new project list view.
func NewProjectListView(projectRoot string, width, height int) *ProjectListView {
	return &ProjectListView{
		projectRoot: projectRoot,
		width:       width,
		height:      height,
	}
}

func (v *ProjectListView) Init() tea.Cmd {
	return v.loadProjects()
}

func (v *ProjectListView) loadProjects() tea.Cmd {
	root := v.projectRoot
	return func() tea.Msg {
		metas, err := project.List(root, false)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("loading projects: %w", err)}
		}

		var projects []ProjectInfo
		for _, m := range metas {
			dir := root + "/" + m.Name
			open, done, blocked := service.ProjectTotals(ctx(), dir)
			projects = append(projects, ProjectInfo{
				Name:    m.Name,
				Dir:     dir,
				Open:    open,
				Done:    done,
				Blocked: blocked,
			})
		}
		return ProjectsLoadedMsg{Projects: projects}
	}
}

func (v *ProjectListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		return v, nil

	case ProjectsLoadedMsg:
		v.projects = msg.Projects
		v.loaded = true
		if v.cursor >= len(v.projects) {
			v.cursor = max(0, len(v.projects)-1)
		}
		return v, nil

	case DataChangedMsg:
		return v, v.loadProjects()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, GlobalKeyMap.Back):
			return v, func() tea.Msg { return GoBackMsg{} }

		case key.Matches(msg, NavKeyMap.Up):
			if v.cursor > 0 {
				v.cursor--
			}

		case key.Matches(msg, NavKeyMap.Down):
			if v.cursor < len(v.projects)-1 {
				v.cursor++
			}

		case key.Matches(msg, NavKeyMap.Enter):
			if len(v.projects) > 0 && v.cursor < len(v.projects) {
				p := v.projects[v.cursor]
				return v, func() tea.Msg {
					return NavigateMsg{
						To:          ViewTodoList,
						ProjectName: p.Name,
						ProjectDir:  p.Dir,
					}
				}
			}

		case key.Matches(msg, GlobalKeyMap.Search):
			// Search not available at project level — need a project context
		}
	}

	return v, nil
}

func (v *ProjectListView) View() string {
	title := TitleStyle.Render("Projects")

	if !v.loaded {
		return title + "\n\n" + HelpStyle.Render("  Loading...")
	}

	if len(v.projects) == 0 {
		return title + "\n\n" + HelpStyle.Render("  No projects found. Use 'p project new' to create one.")
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
	if scrollEnd > len(v.projects) {
		scrollEnd = len(v.projects)
	}

	for i := scrollStart; i < scrollEnd; i++ {
		p := v.projects[i]

		cursor := "  "
		if v.cursor == i {
			cursor = CursorStyle.Render("▸ ")
		}

		counts := formatCounts(p.Open, p.Done, p.Blocked)

		name := p.Name
		if v.cursor == i {
			name = SelectedStyle.Render(name)
		}

		padding := 20 - len(p.Name)
		if padding < 2 {
			padding = 2
		}
		pad := spaces(padding)

		s += fmt.Sprintf("%s%s%s%s\n", cursor, name, pad, counts)
	}

	s += "\n" + HelpStyle.Render("  ↑↓ navigate  Enter select  q quit  ? help")

	return s
}

// formatCounts renders open/done/blocked counts as a styled string.
func formatCounts(open, done, blocked int) string {
	var parts []string
	if open > 0 {
		parts = append(parts, CountOpenStyle.Render(fmt.Sprintf("%d open", open)))
	}
	if done > 0 {
		parts = append(parts, CountDoneStyle.Render(fmt.Sprintf("%d done", done)))
	}
	if blocked > 0 {
		parts = append(parts, CountBlockedStyle.Render(fmt.Sprintf("%d blocked", blocked)))
	}
	if len(parts) == 0 {
		return HelpStyle.Render("empty")
	}
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += HelpStyle.Render(" · ")
		}
		result += part
	}
	return result
}

// spaces returns a string of n spaces.
func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
