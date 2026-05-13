package tui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/glamour"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/walter/p/internal/knowledge"
)

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

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, GlobalKeyMap.Back):
			return v, func() tea.Msg { return GoBackMsg{} }

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

	if len(v.lines) == 0 {
		sb.WriteString(HelpStyle.Render("  (empty document)") + "\n")
	} else {
		startIdx := min(v.scrollOffset, len(v.lines))
		endIdx := min(v.scrollOffset+vpHeight, len(v.lines))

		for _, line := range v.lines[startIdx:endIdx] {
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

	sb.WriteString("\n" + HelpStyle.Render("  ↑↓ scroll  PgUp/PgDn half-page  g/G top/bottom  Esc back"))

	return sb.String()
}
