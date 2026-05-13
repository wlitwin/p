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

// ItemDetailView displays full metadata for a single item and provides
// inline action keys for state changes, editing, and other mutations.
type ItemDetailView struct {
	projectName string
	projectDir  string
	listName    string
	itemID      string

	item   *todo.Item
	loaded bool

	width  int
	height int

	// Scroll offset for long content
	scrollOffset int

	// Inline text input state
	inputMode   bool
	inputPrompt string
	inputValue  string
	inputAction func(value string) tea.Cmd
}

// NewItemDetailView creates a new detail view for a specific item.
func NewItemDetailView(projectName, projectDir, listName, itemID string, width, height int) *ItemDetailView {
	return &ItemDetailView{
		projectName: projectName,
		projectDir:  projectDir,
		listName:    listName,
		itemID:      itemID,
		width:       width,
		height:      height,
	}
}

// IsInputMode reports whether the view is in text input mode.
func (v *ItemDetailView) IsInputMode() bool {
	return v.inputMode
}

func (v *ItemDetailView) Init() tea.Cmd {
	return v.loadItem()
}

func (v *ItemDetailView) loadItem() tea.Cmd {
	dir := v.projectDir
	listName := v.listName
	itemID := v.itemID
	return func() tea.Msg {
		list, err := todo.LoadList(dir, listName)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("loading list %q: %w", listName, err)}
		}
		item, err := todo.ResolveItem(list, itemID)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return ListLoadedMsg{List: &todo.List{Items: []*todo.Item{item}}}
	}
}

func (v *ItemDetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		return v, nil

	case ListLoadedMsg:
		v.loaded = true
		if len(msg.List.Items) > 0 {
			v.item = msg.List.Items[0]
		}
		return v, nil

	case DataChangedMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, v.loadItem())
		if msg.StatusText != "" {
			text := msg.StatusText
			cmds = append(cmds, func() tea.Msg { return StatusMsg{Text: text} })
		}
		return v, tea.Batch(cmds...)

	case tea.KeyMsg:
		if v.inputMode {
			return v.handleInput(msg)
		}

		switch {
		// Back navigation
		case key.Matches(msg, GlobalKeyMap.Back):
			return v, func() tea.Msg { return GoBackMsg{} }

		// Scroll
		case key.Matches(msg, NavKeyMap.Up):
			if v.scrollOffset > 0 {
				v.scrollOffset--
			}
		case key.Matches(msg, NavKeyMap.Down):
			v.scrollOffset++
		case key.Matches(msg, NavKeyMap.HalfUp):
			pageSize := max(1, (v.height-4)/2)
			v.scrollOffset -= pageSize
			if v.scrollOffset < 0 {
				v.scrollOffset = 0
			}
		case key.Matches(msg, NavKeyMap.HalfDown):
			pageSize := max(1, (v.height-4)/2)
			v.scrollOffset += pageSize
		case msg.String() == "g":
			v.scrollOffset = 0

		// State changes
		case key.Matches(msg, ItemListKeyMap.ToggleDone):
			return v, v.toggleDone()
		case key.Matches(msg, ItemListKeyMap.SetOpen):
			return v, v.doStateChange(todo.Open)
		case key.Matches(msg, ItemListKeyMap.SetBlocked):
			return v, v.doStateChange(todo.Blocked)
		case key.Matches(msg, ItemListKeyMap.SetDone):
			return v, v.doStateChange(todo.Done)

		// Priority
		case key.Matches(msg, ItemListKeyMap.Priority):
			return v, v.cyclePriority()

		// Edit text
		case key.Matches(msg, ItemListKeyMap.Edit):
			if v.item != nil {
				v.startInput("Edit: ", v.item.Text, func(text string) tea.Cmd {
					return v.doEditItem(text)
				})
			}

		// Due date
		case key.Matches(msg, ItemListKeyMap.DueDate):
			if v.item != nil {
				v.startInput("Due date (YYYY-MM-DD): ", v.item.Due, func(date string) tea.Cmd {
					return v.doSetDueDate(date)
				})
			}

		// Tags
		case key.Matches(msg, ItemListKeyMap.Tag):
			if v.item != nil {
				currentTags := strings.Join(v.item.Tags, ", ")
				v.startInput("Tags (comma-separated): ", currentTags, func(input string) tea.Cmd {
					return v.doSetTags(input)
				})
			}
		}
	}

	return v, nil
}

// handleInput processes key events during inline text input mode.
func (v *ItemDetailView) handleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (v *ItemDetailView) startInput(prompt, initialValue string, action func(string) tea.Cmd) {
	v.inputMode = true
	v.inputPrompt = prompt
	v.inputValue = initialValue
	v.inputAction = action
}

// --- Mutations ---

func (v *ItemDetailView) toggleDone() tea.Cmd {
	if v.item == nil {
		return nil
	}
	newState := todo.Done
	if v.item.State == todo.Done {
		newState = todo.Open
	}
	return v.doStateChange(newState)
}

func (v *ItemDetailView) doStateChange(state todo.State) tea.Cmd {
	dir := v.projectDir
	listName := v.listName
	itemID := v.itemID

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

func (v *ItemDetailView) cyclePriority() tea.Cmd {
	if v.item == nil {
		return nil
	}
	dir := v.projectDir
	listName := v.listName
	itemID := v.itemID

	newPriority := todo.Backlog
	if v.item.Priority == todo.Backlog {
		newPriority = todo.Now
	}

	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		if err := service.SetItemPriority(ctx(), dir, listName, itemID, newPriority); err != nil {
			return ErrorMsg{Err: err}
		}

		commitMsg := fmt.Sprintf("tui: set %s #%s priority=%s", listName, itemID, newPriority)
		_ = git.CommitAll(context.Background(), dir, commitMsg)

		return DataChangedMsg{StatusText: fmt.Sprintf("Set #%s priority=%s", itemID, newPriority)}
	}
}

func (v *ItemDetailView) doEditItem(text string) tea.Cmd {
	dir := v.projectDir
	listName := v.listName
	itemID := v.itemID

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

func (v *ItemDetailView) doSetDueDate(date string) tea.Cmd {
	dir := v.projectDir
	listName := v.listName
	itemID := v.itemID

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

func (v *ItemDetailView) doSetTags(input string) tea.Cmd {
	dir := v.projectDir
	listName := v.listName
	itemID := v.itemID

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

// --- View rendering ---

func (v *ItemDetailView) View() string {
	title := TitleStyle.Render(v.projectName) +
		HelpStyle.Render(" · ") +
		TitleStyle.Render(v.listName) +
		HelpStyle.Render(" · ") +
		TitleStyle.Render("#"+v.itemID)

	if !v.loaded || v.item == nil {
		return title + "\n\n" + HelpStyle.Render("  Loading...")
	}

	item := v.item

	// Build content lines for scrolling
	var lines []string

	// Item text (prominent)
	lines = append(lines, "")
	lines = append(lines, "  "+item.Text)
	lines = append(lines, "")

	// State with marker
	marker, markerStyle := stateDisplay(item.State)
	lines = append(lines, fmt.Sprintf("  %-10s %s %s", HelpStyle.Render("State:"), markerStyle, string(item.State)))
	_ = marker

	// Priority
	priStr := string(item.Priority)
	if priStr == "" {
		priStr = "(none)"
	}
	var styledPri string
	switch item.Priority {
	case todo.Now:
		styledPri = NowStyle.Render(priStr)
	case todo.Backlog:
		styledPri = BacklogStyle.Render(priStr)
	default:
		styledPri = HelpStyle.Render(priStr)
	}
	lines = append(lines, fmt.Sprintf("  %-10s %s", HelpStyle.Render("Priority:"), styledPri))

	// Due date
	if item.Due != "" {
		lines = append(lines, fmt.Sprintf("  %-10s %s", HelpStyle.Render("Due:"), Cyan.Render(item.Due)))
	}

	// Created date
	if item.Created != "" {
		lines = append(lines, fmt.Sprintf("  %-10s %s", HelpStyle.Render("Created:"), HelpStyle.Render(item.Created)))
	}

	// Done date
	if item.DoneDate != "" {
		lines = append(lines, fmt.Sprintf("  %-10s %s", HelpStyle.Render("Done:"), HelpStyle.Render(item.DoneDate)))
	}

	// Recurrence
	if item.Recur != "" {
		lines = append(lines, fmt.Sprintf("  %-10s %s", HelpStyle.Render("Recur:"), HelpStyle.Render(item.Recur)))
	}

	// Tags
	if len(item.Tags) > 0 {
		tagStr := "#" + strings.Join(item.Tags, " #")
		lines = append(lines, fmt.Sprintf("  %-10s %s", HelpStyle.Render("Tags:"), HelpStyle.Render(tagStr)))
	}

	// Children
	if len(item.Children) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+TitleStyle.Render("Children:"))
		for i, child := range item.Children {
			childID := fmt.Sprintf("%s.%d", v.itemID, i+1)
			childMarker, _ := stateDisplay(child.State)
			var childMeta []string
			switch child.Priority {
			case todo.Now:
				childMeta = append(childMeta, NowStyle.Render("now"))
			case todo.Backlog:
				childMeta = append(childMeta, BacklogStyle.Render("backlog"))
			}
			if child.Due != "" {
				childMeta = append(childMeta, Cyan.Render(child.Due))
			}
			metaStr := ""
			if len(childMeta) > 0 {
				metaStr = "  " + strings.Join(childMeta, " ")
			}
			lines = append(lines, fmt.Sprintf("    %s %s %s%s",
				HelpStyle.Render(childID+"."),
				childMarker,
				child.Text,
				metaStr,
			))

			// Grandchildren (one level deeper)
			for j, gc := range child.Children {
				gcID := fmt.Sprintf("%s.%d", childID, j+1)
				gcMarker, _ := stateDisplay(gc.State)
				lines = append(lines, fmt.Sprintf("      %s %s %s",
					HelpStyle.Render(gcID+"."),
					gcMarker,
					gc.Text,
				))
			}
		}
	}

	// Apply scroll offset
	visibleHeight := v.height - 4 // title + help bar + padding
	if visibleHeight < 5 {
		visibleHeight = 5
	}
	if v.scrollOffset > len(lines)-visibleHeight {
		v.scrollOffset = max(0, len(lines)-visibleHeight)
	}

	endIdx := v.scrollOffset + visibleHeight
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	s := title + "\n"
	for _, line := range lines[v.scrollOffset:endIdx] {
		s += line + "\n"
	}

	// Scroll indicator
	if len(lines) > visibleHeight {
		scrollPct := 0
		if len(lines)-visibleHeight > 0 {
			scrollPct = v.scrollOffset * 100 / (len(lines) - visibleHeight)
		}
		s += HelpStyle.Render(fmt.Sprintf("  ── %d%% ──", scrollPct)) + "\n"
	}

	// Bottom bar
	if v.inputMode {
		cursorChar := CursorStyle.Render("█")
		s += "\n" + HelpStyle.Render("  "+v.inputPrompt) + v.inputValue + cursorChar
	} else {
		s += "\n" + HelpStyle.Render("  d toggle  o/b/x state  p priority  e edit  D due  t tags  Esc back")
	}

	return s
}

// stateDisplay returns a styled state marker and the style itself.
func stateDisplay(state todo.State) (string, string) {
	switch state {
	case todo.Done:
		return "[x]", DoneStyle.Render("[x]")
	case todo.Blocked:
		return "[-]", BlockedStyle.Render("[-]")
	default:
		return "[ ]", OpenStyle.Render("[ ]")
	}
}
