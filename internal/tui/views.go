package tui

import "github.com/walter/p/internal/todo"

// ViewType identifies which view is currently active in the TUI.
type ViewType int

const (
	ViewProjectList ViewType = iota
	ViewTodoList
	ViewItemList
	ViewItemDetail
	ViewKnowledgeList
	ViewKnowledgeView
	ViewStatus
	ViewSearch
)

// NavigateMsg requests navigation to a different view.
type NavigateMsg struct {
	To          ViewType
	ProjectName string
	ProjectDir  string
	ListName    string
	ItemID      string
	DocName     string
}

// GoBackMsg requests navigation back to the previous view.
type GoBackMsg struct{}

// DataChangedMsg signals that underlying data has changed and views should reload.
// StatusText optionally carries a success message to display after reload.
type DataChangedMsg struct {
	StatusText string
}

// ClearStatusMsg clears the status bar after a timeout.
type ClearStatusMsg struct {
	ID int
}

// ErrorMsg carries an error to be displayed in the status bar.
type ErrorMsg struct {
	Err error
}

// StatusMsg carries a transient status message.
type StatusMsg struct {
	Text string
}

// ItemStateChangedMsg signals that an item's state was changed.
type ItemStateChangedMsg struct {
	ListName string
	ItemID   string
	NewState todo.State
}

// ListLoadedMsg carries a freshly loaded todo list.
type ListLoadedMsg struct {
	List *todo.List
}

// ProjectsLoadedMsg carries loaded project data for the project list view.
type ProjectsLoadedMsg struct {
	Projects []ProjectInfo
}

// ProjectInfo holds summary information about a project.
type ProjectInfo struct {
	Name    string
	Dir     string
	Open    int
	Done    int
	Blocked int
}

// TodoListsLoadedMsg carries loaded todo list summaries.
type TodoListsLoadedMsg struct {
	Lists []TodoListInfo
}

// TodoListInfo holds summary information about a todo list.
type TodoListInfo struct {
	Name    string
	Open    int
	Done    int
	Blocked int
}
