package tui

import "github.com/charmbracelet/bubbles/key"

// GlobalKeyMap defines keybindings available in all views.
var GlobalKeyMap = struct {
	Quit   key.Binding
	Back   key.Binding
	Help   key.Binding
	Search key.Binding
}{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
}

// NavKeyMap defines navigation keybindings shared across list views.
var NavKeyMap = struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
}{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
}

// TodoListKeyMap defines keybindings for the todo list view.
var TodoListKeyMap = struct {
	New       key.Binding
	Delete    key.Binding
	Archive   key.Binding
	Knowledge key.Binding
}{
	New: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new list"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "delete"),
	),
	Archive: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "archive"),
	),
	Knowledge: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("Tab", "knowledge"),
	),
}

// ItemListKeyMap defines keybindings for the item list view.
var ItemListKeyMap = struct {
	ToggleDone     key.Binding
	SetOpen        key.Binding
	SetBlocked     key.Binding
	SetDone        key.Binding
	Priority       key.Binding
	New            key.Binding
	Edit           key.Binding
	DueDate        key.Binding
	Tag            key.Binding
	Move           key.Binding
	Remove         key.Binding
	FilterAll      key.Binding
	FilterOpen     key.Binding
	FilterDone     key.Binding
	FilterBlocked  key.Binding
	CycleFilter    key.Binding
	PriorityFilter key.Binding
	WrapToggle     key.Binding
}{
	ToggleDone: key.NewBinding(
		key.WithKeys(" ", "d"),
		key.WithHelp("space/d", "toggle done"),
	),
	SetOpen: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open"),
	),
	SetBlocked: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "block"),
	),
	SetDone: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "done"),
	),
	Priority: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "priority"),
	),
	New: key.NewBinding(
		key.WithKeys("n", "a"),
		key.WithHelp("n", "new item"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	DueDate: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "due date"),
	),
	Tag: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "tags"),
	),
	Move: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "move"),
	),
	Remove: key.NewBinding(
		key.WithKeys("r", "delete"),
		key.WithHelp("r/Del", "remove"),
	),
	FilterAll: key.NewBinding(
		key.WithKeys("0"),
		key.WithHelp("0", "all"),
	),
	FilterOpen: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "open only"),
	),
	FilterDone: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "done only"),
	),
	FilterBlocked: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "blocked only"),
	),
	CycleFilter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "cycle filter"),
	),
	PriorityFilter: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "priority filter"),
	),
	WrapToggle: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "wrap/unwrap"),
	),
}
