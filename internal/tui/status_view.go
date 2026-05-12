package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/service"
)

// StatusDataLoadedMsg carries loaded status data.
type StatusDataLoadedMsg struct {
	Projects []projectStatus
}

type projectStatus struct {
	Name    string
	Open    int
	Done    int
	Blocked int
	Lists   []service.ListStatus
}

// StatusView shows an overview of all projects or a single project with
// open/done/blocked counts per list.
type StatusView struct {
	projectRoot string
	projectName string
	projectDir  string

	projects []projectStatus
	loaded   bool

	scrollOffset int
	width        int
	height       int
}

// NewStatusView creates a status view. If projectName is non-empty, it shows
// status for just that project; otherwise it shows all projects.
func NewStatusView(projectRoot, projectName, projectDir string, width, height int) *StatusView {
	return &StatusView{
		projectRoot: projectRoot,
		projectName: projectName,
		projectDir:  projectDir,
		width:       width,
		height:      height,
	}
}

func (v *StatusView) Init() tea.Cmd {
	return v.loadStatus()
}

func (v *StatusView) loadStatus() tea.Cmd {
	root := v.projectRoot
	projectName := v.projectName
	projectDir := v.projectDir
	return func() tea.Msg {
		var projects []projectStatus

		if projectName != "" {
			// Single project status
			statuses, err := service.GetProjectListStatuses(ctx(), projectDir)
			if err != nil {
				return ErrorMsg{Err: fmt.Errorf("loading status: %w", err)}
			}
			ps := projectStatus{Name: projectName, Lists: statuses}
			for _, s := range statuses {
				ps.Open += s.Open
				ps.Done += s.Done
				ps.Blocked += s.Blocked
			}
			projects = append(projects, ps)
		} else {
			// All projects
			metas, err := project.List(root, false)
			if err != nil {
				return ErrorMsg{Err: fmt.Errorf("loading projects: %w", err)}
			}
			for _, m := range metas {
				dir := root + "/" + m.Name
				statuses, _ := service.GetProjectListStatuses(ctx(), dir)
				ps := projectStatus{Name: m.Name, Lists: statuses}
				for _, s := range statuses {
					ps.Open += s.Open
					ps.Done += s.Done
					ps.Blocked += s.Blocked
				}
				projects = append(projects, ps)
			}
		}

		return StatusDataLoadedMsg{Projects: projects}
	}
}

func (v *StatusView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		return v, nil

	case StatusDataLoadedMsg:
		v.projects = msg.Projects
		v.loaded = true
		return v, nil

	case DataChangedMsg:
		return v, v.loadStatus()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, GlobalKeyMap.Back):
			return v, func() tea.Msg { return GoBackMsg{} }

		case key.Matches(msg, NavKeyMap.Up):
			if v.scrollOffset > 0 {
				v.scrollOffset--
			}

		case key.Matches(msg, NavKeyMap.Down):
			v.scrollOffset++
		}
	}

	return v, nil
}

func (v *StatusView) View() string {
	title := TitleStyle.Render("Status")
	if v.projectName != "" {
		title = TitleStyle.Render(v.projectName) + HelpStyle.Render(" · Status")
	}

	if !v.loaded {
		return title + "\n\n" + HelpStyle.Render("  Loading...")
	}

	if len(v.projects) == 0 {
		return title + "\n\n" + HelpStyle.Render("  No projects found.")
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n\n")

	var lines []string
	for _, p := range v.projects {
		// Project header with total counts
		counts := formatCounts(p.Open, p.Done, p.Blocked)
		lines = append(lines, fmt.Sprintf("  %s  %s", TitleStyle.Render(p.Name), counts))

		// Per-list breakdown
		for _, l := range p.Lists {
			listCounts := formatCounts(l.Open, l.Done, l.Blocked)
			padding := max(2, 24-len(l.Name))
			lines = append(lines, fmt.Sprintf("    %s%s%s",
				HelpStyle.Render(l.Name), spaces(padding), listCounts))
		}
		lines = append(lines, "") // blank line between projects
	}

	// Apply scrolling
	vpHeight := max(5, v.height-5)
	maxScroll := max(0, len(lines)-vpHeight)
	if v.scrollOffset > maxScroll {
		v.scrollOffset = maxScroll
	}

	endIdx := min(v.scrollOffset+vpHeight, len(lines))
	for _, line := range lines[v.scrollOffset:endIdx] {
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	// Scroll indicator
	if len(lines) > vpHeight && maxScroll > 0 {
		scrollPct := v.scrollOffset * 100 / maxScroll
		sb.WriteString(HelpStyle.Render(fmt.Sprintf("  ── %d%% ──", scrollPct)))
		sb.WriteByte('\n')
	}

	sb.WriteString("\n" + HelpStyle.Render("  ↑↓ scroll  Esc back  q quit"))

	return sb.String()
}
