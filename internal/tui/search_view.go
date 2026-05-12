package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/walter/p/internal/service"
	"github.com/walter/p/internal/todo"
)

// searchResult is a unified result that can be either a todo item or knowledge doc.
type searchResult struct {
	// Type is "todo" or "knowledge"
	Type string
	// For todo results
	ListName string
	ItemID   string
	ItemText string
	State    todo.State
	// For knowledge results
	DocName string
}

// SearchResultsMsg carries search results back to the view.
type SearchResultsMsg struct {
	Results []searchResult
	Query   string
}

// SearchView provides interactive search across todo items and knowledge docs
// in a project. It features a text input at the top with real-time filtered
// results below. Enter jumps to the selected result.
type SearchView struct {
	projectName string
	projectDir  string

	query   string
	results []searchResult
	cursor  int

	width  int
	height int

	// Track whether we've done at least one search
	searched bool
}

// NewSearchView creates a new search view for the given project.
func NewSearchView(projectName, projectDir string, width, height int) *SearchView {
	return &SearchView{
		projectName: projectName,
		projectDir:  projectDir,
		width:       width,
		height:      height,
	}
}

// IsInputMode always returns true since the search view is always in input mode.
func (v *SearchView) IsInputMode() bool {
	return true
}

func (v *SearchView) Init() tea.Cmd {
	return nil
}

func (v *SearchView) doSearch() tea.Cmd {
	if v.query == "" {
		return func() tea.Msg {
			return SearchResultsMsg{Query: ""}
		}
	}

	dir := v.projectDir
	projectName := v.projectName
	query := v.query
	return func() tea.Msg {
		queryLower := strings.ToLower(query)
		matches := service.SearchProject(ctx(), dir, projectName, queryLower)

		var results []searchResult
		for _, m := range matches {
			switch m.Type {
			case "todo":
				for _, r := range m.TodoResults {
					results = append(results, searchResult{
						Type:     "todo",
						ListName: r.ListName,
						ItemID:   r.ItemID,
						ItemText: r.Item.Text,
						State:    r.Item.State,
					})
				}
			case "knowledge":
				results = append(results, searchResult{
					Type:    "knowledge",
					DocName: m.File,
				})
			}
		}

		return SearchResultsMsg{Results: results, Query: query}
	}
}

func (v *SearchView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		return v, nil

	case SearchResultsMsg:
		// Only update if results match the current query (ignore stale results)
		if msg.Query == v.query {
			v.results = msg.Results
			v.searched = true
			if v.cursor >= len(v.results) {
				v.cursor = max(0, len(v.results)-1)
			}
		}
		return v, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, GlobalKeyMap.Back):
			return v, func() tea.Msg { return GoBackMsg{} }

		case msg.String() == "up":
			if v.cursor > 0 {
				v.cursor--
			}
			return v, nil

		case msg.String() == "down":
			if v.cursor < len(v.results)-1 {
				v.cursor++
			}
			return v, nil

		case msg.String() == "enter":
			if v.cursor < len(v.results) {
				r := v.results[v.cursor]
				switch r.Type {
				case "todo":
					return v, func() tea.Msg {
						return NavigateMsg{
							To:       ViewItemList,
							ListName: r.ListName,
						}
					}
				case "knowledge":
					return v, func() tea.Msg {
						return NavigateMsg{
							To:      ViewKnowledgeView,
							DocName: r.DocName,
						}
					}
				}
			}
			return v, nil

		case msg.String() == "backspace":
			if len(v.query) > 0 {
				v.query = v.query[:len(v.query)-1]
				v.cursor = 0
				return v, v.doSearch()
			}
			return v, nil

		default:
			if len(msg.String()) == 1 {
				v.query += msg.String()
				v.cursor = 0
				return v, v.doSearch()
			}
		}
	}

	return v, nil
}

func (v *SearchView) View() string {
	title := TitleStyle.Render(v.projectName) + HelpStyle.Render(" · Search")

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n\n")

	// Search input
	cursorChar := CursorStyle.Render("█")
	sb.WriteString("  " + HelpStyle.Render("Search: ") + v.query + cursorChar + "\n\n")

	if v.query == "" && !v.searched {
		sb.WriteString(HelpStyle.Render("  Type to search todos and knowledge docs...") + "\n")
	} else if len(v.results) == 0 && v.searched {
		sb.WriteString(HelpStyle.Render("  No results found.") + "\n")
	} else {
		// Calculate visible area for scrolling
		visibleHeight := max(3, v.height-8)

		scrollStart := 0
		if v.cursor >= scrollStart+visibleHeight {
			scrollStart = v.cursor - visibleHeight + 1
		}
		scrollEnd := min(scrollStart+visibleHeight, len(v.results))

		for i := scrollStart; i < scrollEnd; i++ {
			r := v.results[i]

			cursor := "  "
			if v.cursor == i {
				cursor = CursorStyle.Render("▸ ")
			}

			switch r.Type {
			case "todo":
				// State marker
				marker := "[ ]"
				var styledMarker string
				switch r.State {
				case todo.Done:
					marker = "[x]"
					styledMarker = DoneStyle.Render(marker)
				case todo.Blocked:
					marker = "[-]"
					styledMarker = BlockedStyle.Render(marker)
				default:
					styledMarker = OpenStyle.Render(marker)
				}

				listRef := HelpStyle.Render(r.ListName + " #" + r.ItemID + ":")
				text := r.ItemText
				if r.State == todo.Done {
					text = DoneStyle.Render(text)
				}

				fmt.Fprintf(&sb, "%s%s %s %s %s\n", cursor, listRef, styledMarker, text, "")

			case "knowledge":
				docRef := HelpStyle.Render("knowledge/")
				name := r.DocName
				if v.cursor == i {
					name = SelectedStyle.Render(name)
				}
				fmt.Fprintf(&sb, "%s%s%s\n", cursor, docRef, name)
			}
		}
	}

	sb.WriteString("\n" + HelpStyle.Render("  ↑↓ navigate  Enter jump to result  Esc back"))

	return sb.String()
}
