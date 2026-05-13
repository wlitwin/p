package tui

import "strings"

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
	sb.WriteString(HelpStyle.Render("    q/Ctrl+C  quit        Esc  back        ?  toggle help") + "\n")
	sb.WriteString(HelpStyle.Render("    T  cycle theme") + "\n\n")

	sb.WriteString("  " + TitleStyle.Render("Navigation") + "\n")
	sb.WriteString(HelpStyle.Render("    ↑/k  up    ↓/j  down    Enter  select") + "\n\n")

	// View-specific sections
	switch viewType {
	case ViewProjectList:
		sb.WriteString("  " + TitleStyle.Render("Projects") + "\n")
		sb.WriteString(HelpStyle.Render("    Enter  open project    S  status overview") + "\n\n")

	case ViewTodoList:
		sb.WriteString("  " + TitleStyle.Render("Todo Lists") + "\n")
		sb.WriteString(HelpStyle.Render("    Enter  open list     Tab  switch to knowledge") + "\n")
		sb.WriteString(HelpStyle.Render("    n  new list          d  delete list") + "\n")
		sb.WriteString(HelpStyle.Render("    a  archive list      /  search") + "\n")
		sb.WriteString(HelpStyle.Render("    S  project status") + "\n\n")

	case ViewItemList:
		sb.WriteString("  " + TitleStyle.Render("Item List") + "\n")
		sb.WriteString(HelpStyle.Render("    Space/d  toggle done  o  open    b  block    x  done") + "\n")
		sb.WriteString(HelpStyle.Render("    p  cycle priority     n  new item    e  edit text") + "\n")
		sb.WriteString(HelpStyle.Render("    D  due date   t  tags   m  move to list   r  remove") + "\n")
		sb.WriteString(HelpStyle.Render("    f  cycle filter  0-3  filter by state  P  priority filter") + "\n")
		sb.WriteString(HelpStyle.Render("    w  wrap/unwrap        Enter  open item detail") + "\n\n")

	case ViewItemDetail:
		sb.WriteString("  " + TitleStyle.Render("Item Detail") + "\n")
		sb.WriteString(HelpStyle.Render("    d  toggle done  o/b/x  state  p  priority") + "\n")
		sb.WriteString(HelpStyle.Render("    e  edit text  D  due date  t  tags  ↑↓  scroll") + "\n\n")

	case ViewKnowledgeList:
		sb.WriteString("  " + TitleStyle.Render("Knowledge") + "\n")
		sb.WriteString(HelpStyle.Render("    Enter  view doc      Tab  switch to todos") + "\n")
		sb.WriteString(HelpStyle.Render("    /  search/filter     n  new doc") + "\n\n")

	case ViewKnowledgeView:
		sb.WriteString("  " + TitleStyle.Render("Knowledge Viewer") + "\n")
		sb.WriteString(HelpStyle.Render("    ↑↓  scroll           PgUp/PgDn  half-page") + "\n")
		sb.WriteString(HelpStyle.Render("    g  top  G  bottom    Esc  back") + "\n\n")

	case ViewSearch:
		sb.WriteString("  " + TitleStyle.Render("Search") + "\n")
		sb.WriteString(HelpStyle.Render("    Type to search       ↑↓  navigate results") + "\n")
		sb.WriteString(HelpStyle.Render("    Enter  jump to result  Esc  back") + "\n\n")

	case ViewStatus:
		sb.WriteString("  " + TitleStyle.Render("Status") + "\n")
		sb.WriteString(HelpStyle.Render("    ↑↓  scroll    Esc  back") + "\n\n")
	}

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
