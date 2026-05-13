package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/lock"
	"github.com/walter/p/internal/service"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// KnowledgeDocsLoadedMsg carries loaded knowledge doc metadata.
type KnowledgeDocsLoadedMsg struct {
	Docs []KnowledgeDocInfo
}

// KnowledgeDocInfo holds summary information about a knowledge doc.
type KnowledgeDocInfo struct {
	Name string
	Size int64
	Tags []string
}

// KnowledgeListView displays all knowledge docs in a project with file sizes
// and tags. Supports Enter to view, `/` to search/filter, `n` to create, and
// Tab to switch back to the todo list view.
type KnowledgeListView struct {
	projectName string
	projectDir  string
	docs        []KnowledgeDocInfo
	cursor      int
	width       int
	height      int
	loaded      bool

	// Inline text input for creating new docs
	inputMode   bool
	inputPrompt string
	inputValue  string
	inputAction func(value string) tea.Cmd

	// Search/filter mode
	searchMode  bool
	searchQuery string
	filtered    []KnowledgeDocInfo

	// Confirmation prompt for destructive actions
	confirmMode   bool
	confirmPrompt string
	confirmAction func() tea.Cmd

	// Archive view toggle
	showArchived bool
}

// NewKnowledgeListView creates a new knowledge list view for the given project.
func NewKnowledgeListView(projectName, projectDir string, width, height int) *KnowledgeListView {
	return &KnowledgeListView{
		projectName: projectName,
		projectDir:  projectDir,
		width:       width,
		height:      height,
	}
}

// IsInputMode reports whether the view is in text input mode.
func (v *KnowledgeListView) IsInputMode() bool {
	return v.inputMode || v.searchMode || v.confirmMode
}

func (v *KnowledgeListView) Init() tea.Cmd {
	return v.loadDocs()
}

func (v *KnowledgeListView) loadDocs() tea.Cmd {
	dir := v.projectDir
	showArchived := v.showArchived
	return func() tea.Msg {
		var names []string
		var err error

		if showArchived {
			names, err = listArchivedKnowledgeDocs(dir)
		} else {
			names, err = knowledge.ListFiles(dir)
		}
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("loading knowledge docs: %w", err)}
		}

		var docs []KnowledgeDocInfo
		for _, name := range names {
			info := KnowledgeDocInfo{Name: name}

			// Get file size
			var path string
			if showArchived {
				path = filepath.Join(knowledge.Dir(dir), ".archive", name+".md")
			} else {
				path = knowledge.FilePath(dir, name)
			}
			if stat, err := os.Stat(path); err == nil {
				info.Size = stat.Size()
			}

			// Extract tags from content
			if content, err := os.ReadFile(path); err == nil {
				info.Tags = knowledge.ExtractTags(string(content))
			}

			docs = append(docs, info)
		}
		return KnowledgeDocsLoadedMsg{Docs: docs}
	}
}

// listArchivedKnowledgeDocs returns names of docs in knowledge/.archive/.
func listArchivedKnowledgeDocs(projectDir string) ([]string, error) {
	archiveDir := filepath.Join(knowledge.Dir(projectDir), ".archive")
	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".md"))
	}
	return names, nil
}

func (v *KnowledgeListView) visibleDocs() []KnowledgeDocInfo {
	if v.searchMode && v.searchQuery != "" {
		return v.filtered
	}
	return v.docs
}

func (v *KnowledgeListView) applySearch() {
	if v.searchQuery == "" {
		v.filtered = nil
		return
	}
	query := strings.ToLower(v.searchQuery)
	v.filtered = nil

	// Tag search: #tag prefix filters by matching doc tags
	if tagQuery, ok := strings.CutPrefix(query, "#"); ok {
		if tagQuery == "" {
			return
		}
		for _, doc := range v.docs {
			for _, tag := range doc.Tags {
				if strings.Contains(strings.ToLower(tag), tagQuery) {
					v.filtered = append(v.filtered, doc)
					break
				}
			}
		}
	} else {
		for _, doc := range v.docs {
			nameMatch := strings.Contains(strings.ToLower(doc.Name), query)
			tagMatch := false
			for _, tag := range doc.Tags {
				if strings.Contains(strings.ToLower(tag), query) {
					tagMatch = true
					break
				}
			}
			if nameMatch || tagMatch {
				v.filtered = append(v.filtered, doc)
			}
		}
	}

	if v.cursor >= len(v.filtered) {
		v.cursor = max(0, len(v.filtered)-1)
	}
}

func (v *KnowledgeListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		return v, nil

	case KnowledgeDocsLoadedMsg:
		v.docs = msg.Docs
		v.loaded = true
		if v.searchMode {
			v.applySearch()
		}
		visible := v.visibleDocs()
		if v.cursor >= len(visible) {
			v.cursor = max(0, len(visible)-1)
		}
		return v, nil

	case DataChangedMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, v.loadDocs())
		if msg.StatusText != "" {
			text := msg.StatusText
			cmds = append(cmds, func() tea.Msg { return StatusMsg{Text: text} })
		}
		return v, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Confirmation mode
		if v.confirmMode {
			return v.handleConfirm(msg)
		}

		// Input mode for creating new docs
		if v.inputMode {
			return v.handleInput(msg)
		}

		// Search/filter mode
		if v.searchMode {
			return v.handleSearch(msg)
		}

		switch {
		case key.Matches(msg, GlobalKeyMap.Back):
			return v, func() tea.Msg { return GoBackMsg{} }

		case key.Matches(msg, NavKeyMap.Up):
			if v.cursor > 0 {
				v.cursor--
			}

		case key.Matches(msg, NavKeyMap.Down):
			visible := v.visibleDocs()
			if v.cursor < len(visible)-1 {
				v.cursor++
			}

		case key.Matches(msg, NavKeyMap.HalfDown):
			visible := v.visibleDocs()
			pageSize := max(1, (v.height-6)/2)
			v.cursor += pageSize
			if v.cursor >= len(visible) {
				v.cursor = max(0, len(visible)-1)
			}
		case key.Matches(msg, NavKeyMap.HalfUp):
			pageSize := max(1, (v.height-6)/2)
			v.cursor -= pageSize
			if v.cursor < 0 {
				v.cursor = 0
			}
		case key.Matches(msg, NavKeyMap.Bottom):
			visible := v.visibleDocs()
			v.cursor = max(0, len(visible)-1)
		case msg.String() == "g":
			v.cursor = 0

		case key.Matches(msg, NavKeyMap.Enter):
			visible := v.visibleDocs()
			if len(visible) > 0 && v.cursor < len(visible) {
				doc := visible[v.cursor]
				archived := v.showArchived
				return v, func() tea.Msg {
					return NavigateMsg{
						To:       ViewKnowledgeView,
						DocName:  doc.Name,
						Archived: archived,
					}
				}
			}

		case key.Matches(msg, TodoListKeyMap.Knowledge):
			// Tab switches back to todo list view
			return v, func() tea.Msg {
				return NavigateMsg{To: ViewTodoList}
			}

		case key.Matches(msg, GlobalKeyMap.Search):
			v.searchMode = true
			v.searchQuery = ""
			v.filtered = nil

		case msg.String() == "n":
			v.inputMode = true
			v.inputPrompt = "New doc name: "
			v.inputValue = ""
			v.inputAction = v.createDoc

		case msg.String() == "d":
			visible := v.visibleDocs()
			if len(visible) > 0 && v.cursor < len(visible) {
				doc := visible[v.cursor]
				v.confirmMode = true
				v.confirmPrompt = fmt.Sprintf("Delete %q? (y/n)", doc.Name)
				v.confirmAction = func() tea.Cmd {
					return v.doDeleteDoc(doc.Name)
				}
			}

		case msg.String() == "a":
			if !v.showArchived {
				visible := v.visibleDocs()
				if len(visible) > 0 && v.cursor < len(visible) {
					doc := visible[v.cursor]
					return v, v.doArchiveDoc(doc.Name)
				}
			}

		case msg.String() == "r":
			visible := v.visibleDocs()
			if len(visible) > 0 && v.cursor < len(visible) {
				doc := visible[v.cursor]
				v.inputMode = true
				v.inputPrompt = "Rename to: "
				v.inputValue = doc.Name
				v.inputAction = func(newName string) tea.Cmd {
					return v.doRenameDoc(doc.Name, newName)
				}
			}

		case msg.String() == "A":
			v.showArchived = !v.showArchived
			v.cursor = 0
			v.searchMode = false
			v.searchQuery = ""
			v.filtered = nil
			return v, v.loadDocs()

		case msg.String() == "R":
			if v.showArchived {
				visible := v.visibleDocs()
				if len(visible) > 0 && v.cursor < len(visible) {
					doc := visible[v.cursor]
					return v, v.doRestoreDoc(doc.Name)
				}
			}
		}
	}

	return v, nil
}

func (v *KnowledgeListView) handleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (v *KnowledgeListView) handleSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.searchMode = false
		v.searchQuery = ""
		v.filtered = nil
		v.cursor = 0
		return v, nil
	case "enter":
		// Select the currently highlighted item if there is one
		visible := v.visibleDocs()
		if len(visible) > 0 && v.cursor < len(visible) {
			doc := visible[v.cursor]
			archived := v.showArchived
			v.searchMode = false
			v.searchQuery = ""
			v.filtered = nil
			return v, func() tea.Msg {
				return NavigateMsg{
					To:       ViewKnowledgeView,
					DocName:  doc.Name,
					Archived: archived,
				}
			}
		}
		return v, nil
	case "backspace":
		if len(v.searchQuery) > 0 {
			v.searchQuery = v.searchQuery[:len(v.searchQuery)-1]
			v.applySearch()
		}
		return v, nil
	case "up":
		if v.cursor > 0 {
			v.cursor--
		}
		return v, nil
	case "down":
		visible := v.visibleDocs()
		if v.cursor < len(visible)-1 {
			v.cursor++
		}
		return v, nil
	default:
		if len(msg.String()) == 1 {
			v.searchQuery += msg.String()
			v.applySearch()
		}
		return v, nil
	}
}

func (v *KnowledgeListView) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	return v, nil
}

func (v *KnowledgeListView) createDoc(name string) tea.Cmd {
	dir := v.projectDir
	return func() tea.Msg {
		title := strings.ReplaceAll(name, "-", " ")
		title = cases.Title(language.English).String(title)
		if err := knowledge.Create(dir, name, title, nil); err != nil {
			return ErrorMsg{Err: fmt.Errorf("creating doc: %w", err)}
		}
		return DataChangedMsg{StatusText: fmt.Sprintf("Created %s", name)}
	}
}

func (v *KnowledgeListView) doDeleteDoc(name string) tea.Cmd {
	dir := v.projectDir
	archived := v.showArchived
	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		if archived {
			path := filepath.Join(knowledge.Dir(dir), ".archive", name+".md")
			if err := os.Remove(path); err != nil {
				return ErrorMsg{Err: fmt.Errorf("deleting archived doc: %w", err)}
			}
		} else {
			if err := service.KnowledgeDelete(ctx(), dir, name); err != nil {
				return ErrorMsg{Err: err}
			}
		}

		_ = git.CommitAll(context.Background(), dir, fmt.Sprintf("tui: delete knowledge doc %s", name))
		return DataChangedMsg{StatusText: fmt.Sprintf("Deleted %s", name)}
	}
}

func (v *KnowledgeListView) doArchiveDoc(name string) tea.Cmd {
	dir := v.projectDir
	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		src := knowledge.FilePath(dir, name)
		archiveDir := filepath.Join(knowledge.Dir(dir), ".archive")
		dst := filepath.Join(archiveDir, filepath.Base(src))

		if _, err := os.Stat(src); err != nil {
			return ErrorMsg{Err: fmt.Errorf("doc %q not found", name)}
		}
		if err := os.MkdirAll(archiveDir, 0o755); err != nil {
			return ErrorMsg{Err: fmt.Errorf("creating archive dir: %w", err)}
		}
		if err := os.Rename(src, dst); err != nil {
			return ErrorMsg{Err: fmt.Errorf("archiving: %w", err)}
		}

		_ = git.CommitAll(context.Background(), dir, fmt.Sprintf("tui: archive knowledge doc %s", name))
		return DataChangedMsg{StatusText: fmt.Sprintf("Archived %s", name)}
	}
}

func (v *KnowledgeListView) doRenameDoc(oldName, newName string) tea.Cmd {
	dir := v.projectDir
	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		if err := service.KnowledgeRename(ctx(), dir, oldName, newName); err != nil {
			return ErrorMsg{Err: err}
		}

		_ = git.CommitAll(context.Background(), dir, fmt.Sprintf("tui: rename %s → %s", oldName, newName))
		return DataChangedMsg{StatusText: fmt.Sprintf("Renamed to %s", newName)}
	}
}

func (v *KnowledgeListView) doRestoreDoc(name string) tea.Cmd {
	dir := v.projectDir
	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		archiveDir := filepath.Join(knowledge.Dir(dir), ".archive")
		src := filepath.Join(archiveDir, name+".md")
		dst := knowledge.FilePath(dir, name)

		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return ErrorMsg{Err: fmt.Errorf("creating dir: %w", err)}
		}
		if err := os.Rename(src, dst); err != nil {
			return ErrorMsg{Err: fmt.Errorf("restoring: %w", err)}
		}

		_ = git.CommitAll(context.Background(), dir, fmt.Sprintf("tui: restore knowledge doc %s", name))
		return DataChangedMsg{StatusText: fmt.Sprintf("Restored %s", name)}
	}
}

func (v *KnowledgeListView) View() string {
	titleSuffix := " · Knowledge"
	if v.showArchived {
		titleSuffix = " · Knowledge (archived)"
	}
	title := TitleStyle.Render(v.projectName) + HelpStyle.Render(titleSuffix)

	if !v.loaded {
		return title + "\n\n" + HelpStyle.Render("  Loading...")
	}

	visible := v.visibleDocs()

	if len(v.docs) == 0 {
		s := title + "\n\n" + HelpStyle.Render("  No knowledge docs found. Press 'n' to create one.")
		s += "\n\n" + HelpStyle.Render("  Tab todos  n new  q quit  ? help")
		return s
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n")

	// Search indicator
	if v.searchMode {
		cursorChar := CursorStyle.Render("█")
		sb.WriteString(HelpStyle.Render("  Search: ") + v.searchQuery + cursorChar + "\n\n")
	} else {
		sb.WriteString("\n")
	}

	if len(visible) == 0 && v.searchMode {
		sb.WriteString(HelpStyle.Render("  No docs match the search query.") + "\n")
	} else {
		// Calculate visible area for scrolling
		visibleHeight := max(3, v.height-6)

		scrollStart := 0
		if v.cursor >= scrollStart+visibleHeight {
			scrollStart = v.cursor - visibleHeight + 1
		}
		scrollEnd := min(scrollStart+visibleHeight, len(visible))

		// Adapt name column width to terminal width
		nameWidth := 30
		if v.width > 0 && v.width < 80 {
			nameWidth = max(16, v.width/3)
		}
		showTags := v.width == 0 || v.width >= 60

		for i := scrollStart; i < scrollEnd; i++ {
			doc := visible[i]

			cursor := "  "
			if v.cursor == i {
				cursor = CursorStyle.Render("▸ ")
			}

			name := doc.Name
			displayName := name
			if len(displayName) > nameWidth {
				displayName = displayName[:nameWidth-1] + "…"
			}
			if v.cursor == i {
				displayName = SelectedStyle.Render(displayName)
			}

			// File size
			sizeStr := formatSize(doc.Size)

			// Tags (hidden on narrow terminals)
			tagStr := ""
			if showTags && len(doc.Tags) > 0 {
				tagStr = HelpStyle.Render(strings.Join(doc.Tags, ", "))
			}

			padding := max(2, nameWidth-len(name))
			sizePad := max(2, 10-len(sizeStr))

			fmt.Fprintf(&sb, "%s%s%s%s%s%s\n", cursor, displayName, spaces(padding),
				HelpStyle.Render(sizeStr), spaces(sizePad), tagStr)
		}
	}

	// Bottom bar
	if v.inputMode {
		cursorChar := CursorStyle.Render("█")
		sb.WriteString("\n" + HelpStyle.Render("  "+v.inputPrompt) + v.inputValue + cursorChar)
	} else if v.confirmMode {
		sb.WriteString("\n" + ErrorStyle.Render("  "+v.confirmPrompt))
	} else if v.searchMode {
		sb.WriteString("\n" + HelpStyle.Render("  ↑↓ navigate  Enter view  Esc cancel search"))
	} else if v.showArchived {
		sb.WriteString("\n" + HelpStyle.Render("  ↑↓/jk nav  Enter view  R restore  d delete  A active  / search  ? help"))
	} else {
		sb.WriteString("\n" + HelpStyle.Render("  ↑↓/jk nav  Enter view  d del  a archive  r rename  A archived  / search  n new  ? help"))
	}

	return sb.String()
}

// formatSize returns a human-readable file size string.
func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(bytes)/1024/1024)
	case bytes >= 1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
