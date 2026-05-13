package tui

import (
	"fmt"
	"strings"
)

// helpEntry represents a single key → description pair in the help view.
type helpEntry struct {
	key  string
	desc string
}

// formatHelpColumns renders help entries as aligned two-column rows within
// the given width. Entries are laid out left-to-right, top-to-bottom in
// the specified number of columns.
func formatHelpColumns(entries []helpEntry, cols, totalWidth int) string {
	if len(entries) == 0 {
		return ""
	}

	// Find the widest key across all entries for consistent alignment
	maxKeyWidth := 0
	for _, e := range entries {
		if len(e.key) > maxKeyWidth {
			maxKeyWidth = len(e.key)
		}
	}

	colWidth := totalWidth / cols
	if colWidth < 20 {
		colWidth = 20
	}

	var sb strings.Builder
	for i := 0; i < len(entries); i += cols {
		var parts []string
		for c := 0; c < cols; c++ {
			idx := i + c
			if idx >= len(entries) {
				break
			}
			e := entries[idx]
			// Right-pad the key to align descriptions
			paddedKey := fmt.Sprintf("%-*s", maxKeyWidth, e.key)
			cell := fmt.Sprintf("%s  %s", paddedKey, e.desc)
			// Pad cell to column width
			if c < cols-1 {
				cell = fmt.Sprintf("%-*s", colWidth, cell)
			}
			parts = append(parts, cell)
		}
		sb.WriteString("    " + strings.Join(parts, "") + "\n")
	}
	return sb.String()
}

// renderContextHelp returns help text tailored to the given view type.
// It always includes global keybindings plus the section relevant to the
// current view.
func renderContextHelp(viewType ViewType) string {
	var sb strings.Builder

	sb.WriteString(TitleStyle.Render("Keyboard Shortcuts"))
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	// Global — always shown
	sb.WriteString("  " + TitleStyle.Render("Global") + "\n")
	globalEntries := []helpEntry{
		{"q", "quit"},
		{"Esc", "back"},
		{"?", "toggle help"},
		{"T", "cycle theme"},
	}
	sb.WriteString(HelpStyle.Render(formatHelpColumns(globalEntries, 2, 52)))
	sb.WriteByte('\n')

	sb.WriteString("  " + TitleStyle.Render("Navigation") + "\n")
	navEntries := []helpEntry{
		{"↑/k", "up"},
		{"↓/j", "down"},
		{"Ctrl+U", "half page up"},
		{"Ctrl+D", "half page down"},
		{"g", "go to top"},
		{"G", "go to bottom"},
		{"Enter", "select"},
	}
	sb.WriteString(HelpStyle.Render(formatHelpColumns(navEntries, 2, 52)))
	sb.WriteByte('\n')

	// View-specific sections
	switch viewType {
	case ViewProjectList:
		sb.WriteString("  " + TitleStyle.Render("Projects") + "\n")
		entries := []helpEntry{
			{"Enter", "open project"},
			{"S", "status overview"},
		}
		sb.WriteString(HelpStyle.Render(formatHelpColumns(entries, 2, 52)))

	case ViewTodoList:
		sb.WriteString("  " + TitleStyle.Render("Todo Lists") + "\n")
		entries := []helpEntry{
			{"Enter", "open list"},
			{"Tab", "knowledge view"},
			{"n", "new list"},
			{"d", "delete list"},
			{"a", "archive list"},
			{"/", "search"},
			{"S", "project status"},
		}
		sb.WriteString(HelpStyle.Render(formatHelpColumns(entries, 2, 52)))

	case ViewItemList:
		sb.WriteString("  " + TitleStyle.Render("State") + "\n")
		stateEntries := []helpEntry{
			{"Space/d", "toggle done"},
			{"o", "set open"},
			{"b", "set blocked"},
			{"x", "set done"},
		}
		sb.WriteString(HelpStyle.Render(formatHelpColumns(stateEntries, 2, 52)))
		sb.WriteByte('\n')

		sb.WriteString("  " + TitleStyle.Render("Edit") + "\n")
		editEntries := []helpEntry{
			{"p", "cycle priority"},
			{"n", "new item"},
			{"e", "edit text"},
			{"D", "due date"},
			{"t", "tags"},
			{"m", "move to list"},
			{"r", "remove item"},
		}
		sb.WriteString(HelpStyle.Render(formatHelpColumns(editEntries, 2, 52)))
		sb.WriteByte('\n')

		sb.WriteString("  " + TitleStyle.Render("Filter & Display") + "\n")
		filterEntries := []helpEntry{
			{"f", "cycle filter"},
			{"P", "priority filter"},
			{"0", "show all"},
			{"1", "open only"},
			{"2", "done only"},
			{"3", "blocked only"},
			{"w", "wrap/unwrap"},
			{"Enter", "item detail"},
		}
		sb.WriteString(HelpStyle.Render(formatHelpColumns(filterEntries, 2, 52)))

	case ViewItemDetail:
		sb.WriteString("  " + TitleStyle.Render("Item Detail") + "\n")
		entries := []helpEntry{
			{"d", "toggle done"},
			{"o", "set open"},
			{"b", "set blocked"},
			{"x", "set done"},
			{"p", "cycle priority"},
			{"e", "edit text"},
			{"D", "due date"},
			{"t", "tags"},
		}
		sb.WriteString(HelpStyle.Render(formatHelpColumns(entries, 2, 52)))

	case ViewKnowledgeList:
		sb.WriteString("  " + TitleStyle.Render("Knowledge") + "\n")
		entries := []helpEntry{
			{"Enter", "view doc"},
			{"Tab", "switch to todos"},
			{"/", "search/filter (#tag)"},
			{"n", "new doc"},
			{"d", "delete doc"},
			{"a", "archive doc"},
			{"r", "rename doc"},
			{"A", "toggle archived view"},
			{"R", "restore (archived)"},
		}
		sb.WriteString(HelpStyle.Render(formatHelpColumns(entries, 2, 52)))

	case ViewKnowledgeView:
		sb.WriteString("  " + TitleStyle.Render("Knowledge Viewer") + "\n")
		entries := []helpEntry{
			{"↑↓", "scroll"},
			{"PgUp/PgDn", "half page"},
			{"g", "top"},
			{"G", "bottom"},
			{"d", "delete doc"},
			{"a", "archive doc"},
			{"r", "rename doc"},
		}
		sb.WriteString(HelpStyle.Render(formatHelpColumns(entries, 2, 52)))

	case ViewSearch:
		sb.WriteString("  " + TitleStyle.Render("Search") + "\n")
		entries := []helpEntry{
			{"Type", "search"},
			{"↑↓", "navigate results"},
			{"Enter", "jump to result"},
			{"Esc", "back"},
		}
		sb.WriteString(HelpStyle.Render(formatHelpColumns(entries, 2, 52)))

	case ViewStatus:
		sb.WriteString("  " + TitleStyle.Render("Status") + "\n")
		entries := []helpEntry{
			{"↑↓", "scroll"},
			{"Esc", "back"},
		}
		sb.WriteString(HelpStyle.Render(formatHelpColumns(entries, 2, 52)))
	}

	sb.WriteByte('\n')
	sb.WriteString(HelpStyle.Render("  Press any key to close"))

	return sb.String()
}

// activeViewType returns the ViewType for the currently active view model.
func activeViewType(model any) ViewType {
	switch model.(type) {
	case *ProjectListView:
		return ViewProjectList
	case *TodoListView:
		return ViewTodoList
	case *ItemListView:
		return ViewItemList
	case *ItemDetailView:
		return ViewItemDetail
	case *KnowledgeListView:
		return ViewKnowledgeList
	case *KnowledgeView:
		return ViewKnowledgeView
	case *SearchView:
		return ViewSearch
	case *StatusView:
		return ViewStatus
	default:
		return ViewProjectList
	}
}
