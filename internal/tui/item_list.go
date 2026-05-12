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
	"github.com/walter/p/internal/validate"
)

type filterMode string

const (
	filterAll     filterMode = ""
	filterOpen    filterMode = "open"
	filterDone    filterMode = "done"
	filterBlocked filterMode = "blocked"
)

type priorityFilterMode string

const (
	priorityFilterAll     priorityFilterMode = ""
	priorityFilterNow     priorityFilterMode = "now"
	priorityFilterBacklog priorityFilterMode = "backlog"
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

	list           *todo.List
	items          []filteredItem
	cursor         int
	filter         filterMode
	priorityFilter priorityFilterMode

	width  int
	height int
	loaded bool

	// Inline text input state
	inputMode   bool
	inputPrompt string
	inputValue  string
	inputAction func(value string) tea.Cmd

	// Confirmation prompt state
	confirmMode   bool
	confirmPrompt string
	confirmAction func() tea.Cmd

	// Move-to-list selection state
	moveMode    bool
	moveTargets []string
	moveCursor  int
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

// IsInputMode reports whether the view is currently in an interactive mode
// (text input, confirmation, or list selection). Used by the App to avoid
// intercepting keys like 'q' during input.
func (v *ItemListView) IsInputMode() bool {
	return v.inputMode || v.confirmMode || v.moveMode
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
	v.items = filterItems(v.list.Items, string(v.filter), string(v.priorityFilter), "", 1)
}

// filterItems returns items matching the given state and priority filters,
// annotated with their original positional IDs. Empty filter values match all.
func filterItems(items []*todo.Item, state, priority, prefix string, start int) []filteredItem {
	var result []filteredItem
	for i, item := range items {
		id := fmt.Sprintf("%s%d", prefix, start+i)
		stateMatch := state == "" || string(item.State) == state
		priorityMatch := priority == "" || string(item.Priority) == priority
		if stateMatch && priorityMatch {
			result = append(result, filteredItem{OriginalID: id, Item: item})
		}
		if len(item.Children) > 0 {
			result = append(result, filterItems(item.Children, state, priority, id+".", 1)...)
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
		var cmds []tea.Cmd
		cmds = append(cmds, v.loadList())
		if msg.StatusText != "" {
			text := msg.StatusText
			cmds = append(cmds, func() tea.Msg { return StatusMsg{Text: text} })
		}
		return v, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Confirmation mode: only y/n/esc are meaningful
		if v.confirmMode {
			return v.handleConfirm(msg)
		}

		// Move-to-list selection mode
		if v.moveMode {
			return v.handleMoveMode(msg)
		}

		// Inline text input mode
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
			v.startInput("New item: ", "", v.addItem)

		// Edit item text
		case key.Matches(msg, ItemListKeyMap.Edit):
			item := v.selectedItem()
			if item == nil {
				break
			}
			id := v.selectedID()
			v.startInput("Edit: ", item.Text, func(text string) tea.Cmd {
				return v.doEditItem(id, text)
			})

		// Due date
		case key.Matches(msg, ItemListKeyMap.DueDate):
			item := v.selectedItem()
			if item == nil {
				break
			}
			id := v.selectedID()
			v.startInput("Due date (YYYY-MM-DD): ", item.Due, func(date string) tea.Cmd {
				return v.doSetDueDate(id, date)
			})

		// Tags
		case key.Matches(msg, ItemListKeyMap.Tag):
			item := v.selectedItem()
			if item == nil {
				break
			}
			id := v.selectedID()
			currentTags := strings.Join(item.Tags, ", ")
			v.startInput("Tags (comma-separated): ", currentTags, func(input string) tea.Cmd {
				return v.doSetTags(id, input)
			})

		// Move to another list
		case key.Matches(msg, ItemListKeyMap.Move):
			return v, v.startMoveMode()

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

		// Priority filter
		case key.Matches(msg, ItemListKeyMap.PriorityFilter):
			v.cyclePriorityFilter()

		// Remove with confirmation
		case key.Matches(msg, ItemListKeyMap.Remove):
			v.startRemoveConfirm()
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

// handleConfirm processes key events during confirmation prompt mode.
func (v *ItemListView) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	// Ignore other keys during confirmation
	return v, nil
}

// handleMoveMode processes key events during move-to-list selection mode.
func (v *ItemListView) handleMoveMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "esc":
		v.moveMode = false
		v.moveTargets = nil
		return v, nil

	case key.Matches(msg, NavKeyMap.Up):
		if v.moveCursor > 0 {
			v.moveCursor--
		}
		return v, nil

	case key.Matches(msg, NavKeyMap.Down):
		if v.moveCursor < len(v.moveTargets)-1 {
			v.moveCursor++
		}
		return v, nil

	case key.Matches(msg, NavKeyMap.Enter):
		if v.moveCursor < len(v.moveTargets) {
			target := v.moveTargets[v.moveCursor]
			id := v.selectedID()
			v.moveMode = false
			v.moveTargets = nil
			return v, v.doMoveItem(id, target)
		}
		return v, nil
	}

	return v, nil
}

func (v *ItemListView) startInput(prompt, initialValue string, action func(string) tea.Cmd) {
	v.inputMode = true
	v.inputPrompt = prompt
	v.inputValue = initialValue
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

		return DataChangedMsg{StatusText: fmt.Sprintf("Marked #%s %s", itemID, state)}
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

		return DataChangedMsg{StatusText: fmt.Sprintf("Set #%s priority=%s", id, newPriority)}
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

		return DataChangedMsg{StatusText: fmt.Sprintf("Added item to %s", listName)}
	}
}

// startRemoveConfirm initiates an inline confirmation prompt for item removal.
func (v *ItemListView) startRemoveConfirm() {
	id := v.selectedID()
	if id == "" {
		return
	}
	v.confirmMode = true
	v.confirmPrompt = fmt.Sprintf("Remove item #%s? (y/n)", id)
	v.confirmAction = func() tea.Cmd {
		return v.doRemoveItem(id)
	}
}

// doRemoveItem removes the specified item from the list.
func (v *ItemListView) doRemoveItem(itemID string) tea.Cmd {
	dir := v.projectDir
	listName := v.listName

	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		if err := service.RemoveItem(ctx(), dir, listName, itemID); err != nil {
			return ErrorMsg{Err: err}
		}

		commitMsg := fmt.Sprintf("tui: remove %s #%s", listName, itemID)
		_ = git.CommitAll(context.Background(), dir, commitMsg)

		return DataChangedMsg{StatusText: fmt.Sprintf("Removed #%s", itemID)}
	}
}

// doEditItem updates the text of the specified item.
func (v *ItemListView) doEditItem(itemID, text string) tea.Cmd {
	dir := v.projectDir
	listName := v.listName

	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		if err := service.UpdateItemText(ctx(), dir, listName, itemID, text); err != nil {
			return ErrorMsg{Err: err}
		}

		commitMsg := fmt.Sprintf("tui: edit %s #%s", listName, itemID)
		_ = git.CommitAll(context.Background(), dir, commitMsg)

		return DataChangedMsg{StatusText: fmt.Sprintf("Updated #%s text", itemID)}
	}
}

// doSetDueDate validates and sets the due date for the specified item.
func (v *ItemListView) doSetDueDate(itemID, date string) tea.Cmd {
	dir := v.projectDir
	listName := v.listName

	return func() tea.Msg {
		if err := validate.Date(date); err != nil {
			return ErrorMsg{Err: fmt.Errorf("invalid date: %w", err)}
		}

		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		if err := service.SetItemDue(ctx(), dir, listName, itemID, date); err != nil {
			return ErrorMsg{Err: err}
		}

		commitMsg := fmt.Sprintf("tui: set %s #%s due=%s", listName, itemID, date)
		_ = git.CommitAll(context.Background(), dir, commitMsg)

		return DataChangedMsg{StatusText: fmt.Sprintf("Set #%s due=%s", itemID, date)}
	}
}

// doSetTags parses comma-separated tags and replaces the item's tag list.
func (v *ItemListView) doSetTags(itemID, input string) tea.Cmd {
	dir := v.projectDir
	listName := v.listName

	// Parse comma-separated tags, trimming whitespace
	var tags []string
	for _, t := range strings.Split(input, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}

	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		// Load, resolve, replace tags, save
		list, err := todo.LoadList(dir, listName)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		item, err := todo.ResolveItem(list, itemID)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		item.Tags = tags
		if err := todo.SaveList(dir, listName, list); err != nil {
			return ErrorMsg{Err: err}
		}

		commitMsg := fmt.Sprintf("tui: set %s #%s tags", listName, itemID)
		_ = git.CommitAll(context.Background(), dir, commitMsg)

		tagStr := strings.Join(tags, ", ")
		if tagStr == "" {
			tagStr = "(cleared)"
		}
		return DataChangedMsg{StatusText: fmt.Sprintf("Set #%s tags: %s", itemID, tagStr)}
	}
}

// startMoveMode loads available lists and enters move selection mode.
func (v *ItemListView) startMoveMode() tea.Cmd {
	id := v.selectedID()
	if id == "" {
		return nil
	}

	names, err := todo.ListNames(v.projectDir)
	if err != nil {
		return func() tea.Msg { return ErrorMsg{Err: fmt.Errorf("listing targets: %w", err)} }
	}

	// Filter out the current list
	var targets []string
	for _, n := range names {
		if n != v.listName {
			targets = append(targets, n)
		}
	}

	if len(targets) == 0 {
		return func() tea.Msg { return ErrorMsg{Err: fmt.Errorf("no other lists to move to")} }
	}

	v.moveMode = true
	v.moveTargets = targets
	v.moveCursor = 0
	return nil
}

// doMoveItem moves the specified item to the target list.
func (v *ItemListView) doMoveItem(itemID, targetList string) tea.Cmd {
	dir := v.projectDir
	listName := v.listName

	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		if err := service.MoveItem(ctx(), dir, listName, itemID, targetList); err != nil {
			return ErrorMsg{Err: err}
		}

		commitMsg := fmt.Sprintf("tui: move %s #%s to %s", listName, itemID, targetList)
		_ = git.CommitAll(context.Background(), dir, commitMsg)

		return DataChangedMsg{StatusText: fmt.Sprintf("Moved #%s to %s", itemID, targetList)}
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

func (v *ItemListView) cyclePriorityFilter() {
	switch v.priorityFilter {
	case priorityFilterAll:
		v.priorityFilter = priorityFilterNow
	case priorityFilterNow:
		v.priorityFilter = priorityFilterBacklog
	case priorityFilterBacklog:
		v.priorityFilter = priorityFilterAll
	}
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

	// Filter indicator line
	var filters []string
	filterLabel := "all"
	if v.filter != filterAll {
		filterLabel = string(v.filter)
	}
	filters = append(filters, fmt.Sprintf("State: %s", filterLabel))

	if v.priorityFilter != priorityFilterAll {
		filters = append(filters, fmt.Sprintf("Priority: %s", string(v.priorityFilter)))
	}
	s += HelpStyle.Render(fmt.Sprintf("  Filter: %s", strings.Join(filters, " · "))) + "\n\n"

	// Move-to-list overlay
	if v.moveMode {
		s += v.renderMoveMode()
		return s
	}

	if len(v.items) == 0 {
		if v.filter != filterAll || v.priorityFilter != priorityFilterAll {
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
			switch item.Priority {
			case todo.Now:
				meta = append(meta, NowStyle.Render("now"))
			case todo.Backlog:
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

	// Bottom bar: input, confirmation, or help
	if v.inputMode {
		cursorChar := CursorStyle.Render("█")
		s += "\n" + HelpStyle.Render("  "+v.inputPrompt) + v.inputValue + cursorChar
	} else if v.confirmMode {
		s += "\n" + ErrorStyle.Render("  "+v.confirmPrompt)
	} else {
		s += "\n" + HelpStyle.Render("  ↑↓ nav  Space toggle  o/b/x state  p priority  n new  e edit  f/P filter  r remove")
	}

	return s
}

// renderMoveMode draws the move-to-list selection overlay.
func (v *ItemListView) renderMoveMode() string {
	s := "  " + TitleStyle.Render(fmt.Sprintf("Move item #%s to:", v.selectedID())) + "\n\n"

	for i, target := range v.moveTargets {
		cursor := "    "
		name := target
		if i == v.moveCursor {
			cursor = CursorStyle.Render("  ▸ ")
			name = SelectedStyle.Render(target)
		}
		s += cursor + name + "\n"
	}

	s += "\n" + HelpStyle.Render("  ↑↓ select  Enter confirm  Esc cancel")
	return s
}
