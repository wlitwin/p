package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/lock"
	"github.com/walter/p/internal/service"
	"github.com/walter/p/internal/todo"
)

type filterMode string

const (
	filterAll     filterMode = ""
	filterOpen    filterMode = "open"
	filterDone    filterMode = "done"
	filterBlocked filterMode = "blocked"
)

// filteredItem pairs a todo item with its original positional ID from the
// unfiltered list. This preserves ID stability when filtering.
type filteredItem struct {
	OriginalID string
	Item       *todo.Item
}

// ItemListView displays items in a todo list with state markers and supports
// inline state changes, priority cycling, filtering, and adding new items.
type ItemListView struct {
	projectName string
	projectDir  string
	listName    string

	list   *todo.List
	items  []filteredItem
	cursor int
	filter filterMode

	width  int
	height int
	loaded bool

	// Inline text input state
	inputMode   bool
	inputPrompt string
	inputValue  string
	inputAction func(value string) tea.Cmd
}

// NewItemListView creates a new item list view for the given list.
func NewItemListView(projectName, projectDir, listName string, width, height int) *ItemListView {
	return &ItemListView{
		projectName: projectName,
		projectDir:  projectDir,
		listName:    listName,
		width:       width,
		height:      height,
	}
}

// IsInputMode reports whether the view is currently in text input mode.
// Used by the App to avoid intercepting keys like 'q' during input.
func (v *ItemListView) IsInputMode() bool {
	return v.inputMode
}

func (v *ItemListView) Init() tea.Cmd {
	return v.loadList()
}

func (v *ItemListView) loadList() tea.Cmd {
	dir := v.projectDir
	name := v.listName
	return func() tea.Msg {
		list, err := todo.LoadList(dir, name)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("loading list %q: %w", name, err)}
		}
		return ListLoadedMsg{List: list}
	}
}

func (v *ItemListView) applyFilter() {
	if v.list == nil {
		v.items = nil
		return
	}
	v.items = filterItems(v.list.Items, string(v.filter), "", 1)
}

// filterItems returns items matching the given state filter, annotated with
// their original positional IDs. Empty state matches all items.
func filterItems(items []*todo.Item, state, prefix string, start int) []filteredItem {
	var result []filteredItem
	for i, item := range items {
		id := fmt.Sprintf("%s%d", prefix, start+i)
		if state == "" || string(item.State) == state {
			result = append(result, filteredItem{OriginalID: id, Item: item})
		}
		if len(item.Children) > 0 {
			result = append(result, filterItems(item.Children, state, id+".", 1)...)
		}
	}
	return result
}

func (v *ItemListView) selectedID() string {
	if v.cursor >= 0 && v.cursor < len(v.items) {
		return v.items[v.cursor].OriginalID
	}
	return ""
}

func (v *ItemListView) selectedItem() *todo.Item {
	if v.cursor >= 0 && v.cursor < len(v.items) {
		return v.items[v.cursor].Item
	}
	return nil
}

func (v *ItemListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		return v, nil

	case ListLoadedMsg:
		v.list = msg.List
		v.loaded = true
		v.applyFilter()
		if v.cursor >= len(v.items) {
			v.cursor = max(0, len(v.items)-1)
		}
		return v, nil

	case DataChangedMsg:
		return v, v.loadList()

	case tea.KeyMsg:
		if v.inputMode {
			return v.handleInput(msg)
		}

		switch {
		// Back navigation
		case key.Matches(msg, GlobalKeyMap.Back):
			return v, func() tea.Msg { return GoBackMsg{} }

		// Cursor navigation
		case key.Matches(msg, NavKeyMap.Up):
			if v.cursor > 0 {
				v.cursor--
			}
		case key.Matches(msg, NavKeyMap.Down):
			if v.cursor < len(v.items)-1 {
				v.cursor++
			}

		// State changes
		case key.Matches(msg, ItemListKeyMap.ToggleDone):
			return v, v.toggleDone()
		case key.Matches(msg, ItemListKeyMap.SetOpen):
			return v, v.setState(todo.Open)
		case key.Matches(msg, ItemListKeyMap.SetBlocked):
			return v, v.setState(todo.Blocked)
		case key.Matches(msg, ItemListKeyMap.SetDone):
			return v, v.setState(todo.Done)

		// Priority
		case key.Matches(msg, ItemListKeyMap.Priority):
			return v, v.cyclePriority()

		// New item
		case key.Matches(msg, ItemListKeyMap.New):
			v.startInput("New item: ", v.addItem)

		// Filter controls
		case key.Matches(msg, ItemListKeyMap.CycleFilter):
			v.cycleFilter()
		case key.Matches(msg, ItemListKeyMap.FilterAll):
			v.setFilter(filterAll)
		case key.Matches(msg, ItemListKeyMap.FilterOpen):
			v.setFilter(filterOpen)
		case key.Matches(msg, ItemListKeyMap.FilterDone):
			v.setFilter(filterDone)
		case key.Matches(msg, ItemListKeyMap.FilterBlocked):
			v.setFilter(filterBlocked)

		// Remove
		case key.Matches(msg, ItemListKeyMap.Remove):
			return v, v.removeItem()
		}
	}

	return v, nil
}

// handleInput processes key events during inline text input mode.
func (v *ItemListView) handleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (v *ItemListView) startInput(prompt string, action func(string) tea.Cmd) {
	v.inputMode = true
	v.inputPrompt = prompt
	v.inputValue = ""
	v.inputAction = action
}

// toggleDone toggles the selected item between done and open.
func (v *ItemListView) toggleDone() tea.Cmd {
	item := v.selectedItem()
	if item == nil {
		return nil
	}

	newState := todo.Done
	if item.State == todo.Done {
		newState = todo.Open
	}
	return v.doStateChange(v.selectedID(), newState)
}

// setState sets the selected item to the given state.
func (v *ItemListView) setState(state todo.State) tea.Cmd {
	id := v.selectedID()
	if id == "" {
		return nil
	}
	return v.doStateChange(id, state)
}

// doStateChange performs a state change with locking, saving, and git commit.
func (v *ItemListView) doStateChange(itemID string, state todo.State) tea.Cmd {
	dir := v.projectDir
	listName := v.listName

	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		if err := service.SetItemState(ctx(), dir, listName, itemID, state); err != nil {
			return ErrorMsg{Err: err}
		}

		commitMsg := fmt.Sprintf("tui: set %s #%s %s", listName, itemID, state)
		_ = git.CommitAll(context.Background(), dir, commitMsg)

		return DataChangedMsg{}
	}
}

// cyclePriority toggles the selected item's priority between now and backlog.
func (v *ItemListView) cyclePriority() tea.Cmd {
	item := v.selectedItem()
	if item == nil {
		return nil
	}
	id := v.selectedID()
	dir := v.projectDir
	listName := v.listName

	newPriority := todo.Backlog
	if item.Priority == todo.Backlog {
		newPriority = todo.Now
	}

	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		if err := service.SetItemPriority(ctx(), dir, listName, id, newPriority); err != nil {
			return ErrorMsg{Err: err}
		}

		commitMsg := fmt.Sprintf("tui: set %s #%s priority=%s", listName, id, newPriority)
		_ = git.CommitAll(context.Background(), dir, commitMsg)

		return DataChangedMsg{}
	}
}

// addItem adds a new item to the list.
func (v *ItemListView) addItem(text string) tea.Cmd {
	dir := v.projectDir
	listName := v.listName

	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		if err := service.AddItem(ctx(), dir, listName, text, todo.Now, "", ""); err != nil {
			return ErrorMsg{Err: err}
		}

		commitMsg := fmt.Sprintf("tui: add item to %s", listName)
		_ = git.CommitAll(context.Background(), dir, commitMsg)

		return DataChangedMsg{}
	}
}

// removeItem removes the selected item from the list.
func (v *ItemListView) removeItem() tea.Cmd {
	id := v.selectedID()
	if id == "" {
		return nil
	}
	dir := v.projectDir
	listName := v.listName

	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		if err := service.RemoveItem(ctx(), dir, listName, id); err != nil {
			return ErrorMsg{Err: err}
		}

		commitMsg := fmt.Sprintf("tui: remove %s #%s", listName, id)
		_ = git.CommitAll(context.Background(), dir, commitMsg)

		return DataChangedMsg{}
	}
}

func (v *ItemListView) cycleFilter() {
	switch v.filter {
	case filterAll:
		v.filter = filterOpen
	case filterOpen:
		v.filter = filterDone
	case filterDone:
		v.filter = filterBlocked
	case filterBlocked:
		v.filter = filterAll
	}
	v.applyFilter()
	v.cursor = 0
}

func (v *ItemListView) setFilter(f filterMode) {
	v.filter = f
	v.applyFilter()
	v.cursor = 0
}

func (v *ItemListView) View() string {
	title := TitleStyle.Render(v.projectName) +
		HelpStyle.Render(" · ") +
		TitleStyle.Render(v.listName)

	if !v.loaded {
		return title + "\n\n" + HelpStyle.Render("  Loading...")
	}

	s := title + "\n"

	// Filter indicator
	filterLabel := "all"
	if v.filter != filterAll {
		filterLabel = string(v.filter)
	}
	s += HelpStyle.Render(fmt.Sprintf("  Filter: %s", filterLabel)) + "\n\n"

	if len(v.items) == 0 {
		if v.filter != filterAll {
			s += HelpStyle.Render("  No items match the current filter.")
		} else {
			s += HelpStyle.Render("  No items. Press 'n' to add one.")
		}
		s += "\n"
	} else {
		// Calculate visible area for scrolling
		visibleHeight := v.height - 7 // title + filter + help + padding
		if visibleHeight < 3 {
			visibleHeight = 3
		}

		scrollStart := 0
		if v.cursor >= scrollStart+visibleHeight {
			scrollStart = v.cursor - visibleHeight + 1
		}
		scrollEnd := scrollStart + visibleHeight
		if scrollEnd > len(v.items) {
			scrollEnd = len(v.items)
		}

		for i := scrollStart; i < scrollEnd; i++ {
			fi := v.items[i]
			item := fi.Item

			cursor := "  "
			if v.cursor == i {
				cursor = CursorStyle.Render("▸ ")
			}

			// State marker using the same markers as the markdown format
			marker := "[ ]"
			switch item.State {
			case todo.Done:
				marker = "[x]"
			case todo.Blocked:
				marker = "[-]"
			}

			var styledMarker string
			switch item.State {
			case todo.Done:
				styledMarker = DoneStyle.Render(marker)
			case todo.Blocked:
				styledMarker = BlockedStyle.Render(marker)
			default:
				styledMarker = OpenStyle.Render(marker)
			}

			// Positional ID
			styledID := HelpStyle.Render(fi.OriginalID + ".")

			// Item text — dim if done
			text := item.Text
			if item.State == todo.Done {
				text = DoneStyle.Render(text)
			}

			// Metadata: priority and due date
			var meta []string
			if item.Priority == todo.Now {
				meta = append(meta, NowStyle.Render("now"))
			} else if item.Priority == todo.Backlog {
				meta = append(meta, BacklogStyle.Render("backlog"))
			}
			if item.Due != "" {
				meta = append(meta, Cyan.Render(item.Due))
			}
			if len(item.Tags) > 0 {
				meta = append(meta, HelpStyle.Render("#"+strings.Join(item.Tags, " #")))
			}

			metaStr := ""
			if len(meta) > 0 {
				metaStr = "  " + strings.Join(meta, " ")
			}

			// Indentation for nested items (based on dots in ID)
			indent := ""
			dots := strings.Count(fi.OriginalID, ".")
			for d := 0; d < dots; d++ {
				indent += "  "
			}

			s += fmt.Sprintf("%s%s%s %s %s%s\n", cursor, indent, styledID, styledMarker, text, metaStr)
		}
	}

	// Input area or help bar
	if v.inputMode {
		cursorChar := CursorStyle.Render("█")
		s += "\n" + HelpStyle.Render("  "+v.inputPrompt) + v.inputValue + cursorChar
	} else {
		s += "\n" + HelpStyle.Render("  ↑↓ nav  Space toggle  x done  o open  b block  p priority  n new  f filter  r remove")
	}

	return s
}
