package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/x/ansi"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/lock"
	"github.com/walter/p/internal/service"
)

// KnowledgeRenamedMsg signals that the doc was renamed — view updates its title.
type KnowledgeRenamedMsg struct {
	NewName string
}

// KnowledgeContentLoadedMsg carries the loaded and pre-rendered content.
type KnowledgeContentLoadedMsg struct {
	Content string   // raw markdown (kept for re-rendering on resize)
	Lines   []string // pre-rendered lines from glamour
}

// KnowledgeView provides a scrollable read-only viewport for viewing a single
// knowledge document. Uses simple line-based scrolling with Esc to go back.
type KnowledgeView struct {
	projectName string
	projectDir  string
	docName     string

	content string
	lines   []string
	loaded  bool

	scrollOffset int
	width        int
	height       int

	// Confirmation prompt for destructive actions
	confirmMode   bool
	confirmPrompt string
	confirmAction func() tea.Cmd

	// Inline input for rename
	inputMode   bool
	inputPrompt string
	inputValue  string
	inputAction func(value string) tea.Cmd

	// In-doc search
	searchMode    bool
	searchQuery   string
	searchMatches []int // line indices with matches
	searchCurrent int   // index into searchMatches
}

// NewKnowledgeView creates a new knowledge view for the given document.
func NewKnowledgeView(projectName, projectDir, docName string, width, height int) *KnowledgeView {
	return &KnowledgeView{
		projectName: projectName,
		projectDir:  projectDir,
		docName:     docName,
		width:       width,
		height:      height,
	}
}

// IsInputMode reports whether the view is in an interactive mode.
func (v *KnowledgeView) IsInputMode() bool {
	return v.confirmMode || v.inputMode || v.searchMode
}

func (v *KnowledgeView) Init() tea.Cmd {
	return v.loadContent()
}

func (v *KnowledgeView) loadContent() tea.Cmd {
	dir := v.projectDir
	name := v.docName
	width := v.width
	return func() tea.Msg {
		content, err := knowledge.Read(dir, name)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("reading %q: %w", name, err)}
		}
		lines := renderMarkdownContent(content, width)
		return KnowledgeContentLoadedMsg{Content: content, Lines: lines}
	}
}

func (v *KnowledgeView) handleConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (v *KnowledgeView) handleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (v *KnowledgeView) handleSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.searchMode = false
		v.searchQuery = ""
		return v, nil
	case "enter":
		v.searchMode = false
		if v.searchQuery != "" {
			v.findMatches()
			if len(v.searchMatches) > 0 {
				v.scrollToMatch(0)
			}
		}
		return v, nil
	case "backspace":
		if len(v.searchQuery) > 0 {
			v.searchQuery = v.searchQuery[:len(v.searchQuery)-1]
			v.findMatches()
			if len(v.searchMatches) > 0 {
				v.scrollToMatch(0)
			}
		}
		return v, nil
	default:
		if len(msg.String()) == 1 {
			v.searchQuery += msg.String()
			v.findMatches()
			if len(v.searchMatches) > 0 {
				v.scrollToMatch(0)
			}
		}
		return v, nil
	}
}

// findMatches searches rendered lines for the query (case-insensitive).
// ANSI escape codes are stripped before matching so glamour styling doesn't
// interfere with search.
func (v *KnowledgeView) findMatches() {
	v.searchMatches = nil
	v.searchCurrent = 0
	if v.searchQuery == "" {
		return
	}
	query := strings.ToLower(v.searchQuery)
	for i, line := range v.lines {
		plain := strings.ToLower(ansi.Strip(line))
		if strings.Contains(plain, query) {
			v.searchMatches = append(v.searchMatches, i)
		}
	}
}

// scrollToMatch scrolls the viewport so that searchMatches[idx] is visible.
func (v *KnowledgeView) scrollToMatch(idx int) {
	if idx < 0 || idx >= len(v.searchMatches) {
		return
	}
	v.searchCurrent = idx
	line := v.searchMatches[idx]
	vpHeight := v.viewportHeight()

	// Center the match in the viewport if possible
	target := max(0, line-vpHeight/2)
	maxOff := v.maxScroll()
	if target > maxOff {
		target = maxOff
	}
	v.scrollOffset = target
}

// clearSearch removes the active search state.
func (v *KnowledgeView) clearSearch() {
	v.searchQuery = ""
	v.searchMatches = nil
	v.searchCurrent = 0
}

func (v *KnowledgeView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		oldWidth := v.width
		v.width = msg.Width
		v.height = msg.Height
		// Re-render async if width changed (glamour word-wrap depends on width)
		if v.loaded && v.content != "" && msg.Width != oldWidth {
			content := v.content
			width := msg.Width
			return v, func() tea.Msg {
				lines := renderMarkdownContent(content, width)
				return KnowledgeContentLoadedMsg{Content: content, Lines: lines}
			}
		}
		return v, nil

	case KnowledgeContentLoadedMsg:
		v.content = msg.Content
		v.lines = msg.Lines
		v.loaded = true
		if v.scrollOffset > v.maxScroll() {
			v.scrollOffset = v.maxScroll()
		}
		return v, nil

	case DataChangedMsg:
		return v, v.loadContent()

	case KnowledgeRenamedMsg:
		v.docName = msg.NewName
		return v, v.loadContent()

	case tea.KeyMsg:
		// Confirmation mode
		if v.confirmMode {
			return v.handleConfirm(msg)
		}

		// Search mode
		if v.searchMode {
			return v.handleSearch(msg)
		}

		// Input mode (rename)
		if v.inputMode {
			return v.handleInput(msg)
		}

		switch {
		case key.Matches(msg, GlobalKeyMap.Back):
			// If search is active, Esc clears it instead of going back
			if v.searchQuery != "" {
				v.clearSearch()
				return v, nil
			}
			return v, func() tea.Msg { return GoBackMsg{} }

		case key.Matches(msg, GlobalKeyMap.Search):
			v.searchMode = true
			v.searchQuery = ""
			v.searchMatches = nil
			v.searchCurrent = 0
			return v, nil

		case msg.String() == "n" && v.searchQuery != "":
			// Next match
			if len(v.searchMatches) > 0 {
				next := (v.searchCurrent + 1) % len(v.searchMatches)
				v.scrollToMatch(next)
			}
			return v, nil

		case msg.String() == "N" && v.searchQuery != "":
			// Previous match
			if len(v.searchMatches) > 0 {
				prev := v.searchCurrent - 1
				if prev < 0 {
					prev = len(v.searchMatches) - 1
				}
				v.scrollToMatch(prev)
			}
			return v, nil

		case key.Matches(msg, NavKeyMap.Up), msg.String() == "k":
			if v.scrollOffset > 0 {
				v.scrollOffset--
			}

		case key.Matches(msg, NavKeyMap.Down), msg.String() == "j":
			maxOffset := v.maxScroll()
			if v.scrollOffset < maxOffset {
				v.scrollOffset++
			}

		case key.Matches(msg, NavKeyMap.HalfUp):
			pageSize := v.viewportHeight() / 2
			v.scrollOffset -= pageSize
			if v.scrollOffset < 0 {
				v.scrollOffset = 0
			}

		case key.Matches(msg, NavKeyMap.HalfDown):
			pageSize := v.viewportHeight() / 2
			v.scrollOffset += pageSize
			maxOffset := v.maxScroll()
			if v.scrollOffset > maxOffset {
				v.scrollOffset = maxOffset
			}

		case msg.String() == "g":
			v.scrollOffset = 0

		case key.Matches(msg, NavKeyMap.Bottom):
			v.scrollOffset = v.maxScroll()

		case msg.String() == "d":
			v.confirmMode = true
			v.confirmPrompt = fmt.Sprintf("Delete %q? (y/n)", v.docName)
			v.confirmAction = func() tea.Cmd {
				return v.doDeleteDoc()
			}

		case msg.String() == "a":
			return v, v.doArchiveDoc()

		case msg.String() == "r":
			v.inputMode = true
			v.inputPrompt = "Rename to: "
			v.inputValue = v.docName
			v.inputAction = func(newName string) tea.Cmd {
				return v.doRenameDoc(newName)
			}
		}
	}

	return v, nil
}

func (v *KnowledgeView) viewportHeight() int {
	return max(5, v.height-4) // title + help + padding
}

func (v *KnowledgeView) maxScroll() int {
	return max(0, len(v.lines)-v.viewportHeight())
}

func (v *KnowledgeView) doDeleteDoc() tea.Cmd {
	dir := v.projectDir
	name := v.docName
	return func() tea.Msg {
		lk, err := lock.Acquire(dir)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("lock: %w", err)}
		}
		defer lk.Release()

		if err := service.KnowledgeDelete(ctx(), dir, name); err != nil {
			return ErrorMsg{Err: err}
		}

		_ = git.CommitAll(context.Background(), dir, fmt.Sprintf("tui: delete knowledge doc %s", name))
		// Navigate back since the doc no longer exists
		return GoBackMsg{}
	}
}

func (v *KnowledgeView) doArchiveDoc() tea.Cmd {
	dir := v.projectDir
	name := v.docName
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
		return GoBackMsg{}
	}
}

func (v *KnowledgeView) doRenameDoc(newName string) tea.Cmd {
	dir := v.projectDir
	oldName := v.docName
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
		return KnowledgeRenamedMsg{NewName: newName}
	}
}

// Cached glamour renderer — initialized once, reused across all renders.
// Word wrap is set to a reasonable default; content is indented separately.
var (
	glamourMu           sync.Mutex
	glamourRenderer     *glamour.TermRenderer
	glamourRendererWrap int
)

// getGlamourRenderer returns a cached glamour renderer, creating one if needed
// or if the word wrap width has changed significantly (>10 columns).
func getGlamourRenderer(wordWrap int) *glamour.TermRenderer {
	glamourMu.Lock()
	defer glamourMu.Unlock()

	if glamourRenderer != nil && abs(glamourRendererWrap-wordWrap) < 10 {
		return glamourRenderer
	}

	r, err := glamour.NewTermRenderer(
		glamourStyleOption(),
		glamour.WithWordWrap(wordWrap),
	)
	if err != nil {
		return nil
	}

	glamourRenderer = r
	glamourRendererWrap = wordWrap
	return r
}

// prewarmGlamourRenderer initializes the glamour renderer eagerly so it's
// ready by the time a user navigates to a knowledge doc. Uses a default width
// of 80 since the renderer only recreates on >10 column delta.
func prewarmGlamourRenderer() {
	getGlamourRenderer(80)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// glamourStyleOption returns the appropriate glamour rendering option based on
// the GlamourThemeSetting configured by the theme system.
func glamourStyleOption() glamour.TermRendererOption {
	switch GlamourThemeSetting {
	case "dark":
		return glamour.WithStandardStyle("dark")
	case "light":
		return glamour.WithStandardStyle("light")
	case "notty":
		return glamour.WithStandardStyle("notty")
	default:
		return glamour.WithAutoStyle()
	}
}

// renderMarkdownContent renders markdown content using glamour for proper
// formatting of tables, code blocks, lists, etc. YAML frontmatter is styled
// separately. This runs in a background goroutine to avoid blocking the UI.
func renderMarkdownContent(content string, termWidth int) []string {
	frontmatter, markdown := splitFrontmatter(content)

	var lines []string

	// Render frontmatter dimly
	if frontmatter != "" {
		for _, line := range strings.Split(frontmatter, "\n") {
			lines = append(lines, HelpStyle.Render("  "+line))
		}
		lines = append(lines, "")
	}

	if markdown != "" {
		wordWrap := 100
		if termWidth > 0 {
			wordWrap = max(40, termWidth-6)
		}

		renderer := getGlamourRenderer(wordWrap)
		if renderer != nil {
			rendered, err := renderer.Render(markdown)
			if err == nil {
				rendered = strings.TrimRight(rendered, "\n")
				for _, line := range strings.Split(rendered, "\n") {
					lines = append(lines, "  "+line)
				}
				return lines
			}
		}

		// Fallback: plain text if glamour fails
		for _, line := range strings.Split(markdown, "\n") {
			lines = append(lines, "  "+line)
		}
	}

	return lines
}

// splitFrontmatter separates YAML frontmatter (between --- delimiters)
// from the markdown body. Returns ("", content) if no frontmatter is found.
func splitFrontmatter(content string) (frontmatter, body string) {
	if !strings.HasPrefix(content, "---\n") {
		return "", content
	}

	end := strings.Index(content[4:], "\n---")
	if end == -1 {
		return "", content
	}

	// Include both --- delimiters in frontmatter
	fmEnd := 4 + end + 4 // "---\n" + content + "\n---"
	frontmatter = content[:fmEnd]

	body = content[fmEnd:]
	body = strings.TrimLeft(body, "\n")

	return frontmatter, body
}

func (v *KnowledgeView) View() string {
	title := TitleStyle.Render(v.projectName) +
		HelpStyle.Render(" · Knowledge · ") +
		TitleStyle.Render(v.docName)

	if !v.loaded {
		return title + "\n\n" + HelpStyle.Render("  Loading...")
	}

	var sb strings.Builder
	sb.WriteString(title)
	sb.WriteString("\n\n")

	vpHeight := v.viewportHeight()

	// Build a set of matching line indices for fast lookup during rendering
	matchSet := make(map[int]bool, len(v.searchMatches))
	for _, idx := range v.searchMatches {
		matchSet[idx] = true
	}
	currentMatchLine := -1
	if len(v.searchMatches) > 0 && v.searchCurrent < len(v.searchMatches) {
		currentMatchLine = v.searchMatches[v.searchCurrent]
	}

	if len(v.lines) == 0 {
		sb.WriteString(HelpStyle.Render("  (empty document)") + "\n")
	} else {
		startIdx := min(v.scrollOffset, len(v.lines))
		endIdx := min(v.scrollOffset+vpHeight, len(v.lines))

		for i := startIdx; i < endIdx; i++ {
			line := v.lines[i]
			if matchSet[i] {
				// Highlight matching lines with a marker
				if i == currentMatchLine {
					sb.WriteString(NowStyle.Render("▸ "))
				} else {
					sb.WriteString(CursorStyle.Render("│ "))
				}
			}
			sb.WriteString(line)
			sb.WriteByte('\n')
		}
	}

	// Scroll indicator
	if len(v.lines) > vpHeight {
		scrollPct := 0
		maxOff := v.maxScroll()
		if maxOff > 0 {
			scrollPct = v.scrollOffset * 100 / maxOff
		}
		fmt.Fprintf(&sb, "%s\n", HelpStyle.Render(fmt.Sprintf("  ── %d%% ──", scrollPct)))
	}

	if v.searchMode {
		cursorChar := CursorStyle.Render("█")
		matchInfo := ""
		if v.searchQuery != "" {
			matchInfo = HelpStyle.Render(fmt.Sprintf(" (%d matches)", len(v.searchMatches)))
		}
		sb.WriteString("\n" + HelpStyle.Render("  /") + v.searchQuery + cursorChar + matchInfo)
	} else if v.inputMode {
		cursorChar := CursorStyle.Render("█")
		sb.WriteString("\n" + HelpStyle.Render("  "+v.inputPrompt) + v.inputValue + cursorChar)
	} else if v.confirmMode {
		sb.WriteString("\n" + ErrorStyle.Render("  "+v.confirmPrompt))
	} else if v.searchQuery != "" {
		matchInfo := fmt.Sprintf("[%d/%d]", v.searchCurrent+1, len(v.searchMatches))
		sb.WriteString("\n" + HelpStyle.Render(fmt.Sprintf("  /%s %s  n next  N prev  Esc clear  ↑↓ scroll", v.searchQuery, matchInfo)))
	} else {
		sb.WriteString("\n" + HelpStyle.Render("  ↑↓ scroll  / search  d delete  a archive  r rename  Esc back"))
	}

	return sb.String()
}
