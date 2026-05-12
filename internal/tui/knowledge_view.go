package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/walter/p/internal/knowledge"
)

// KnowledgeContentLoadedMsg carries the loaded content of a knowledge doc.
type KnowledgeContentLoadedMsg struct {
	Content string
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
	return func() tea.Msg {
		content, err := knowledge.Read(dir, name)
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("reading %q: %w", name, err)}
		}
		return KnowledgeContentLoadedMsg{Content: content}
	}
}

func (v *KnowledgeView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		return v, nil

	case KnowledgeContentLoadedMsg:
		v.content = msg.Content
		v.lines = v.renderContent(msg.Content)
		v.loaded = true
		v.scrollOffset = 0
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

		case msg.String() == "pgup", msg.String() == "ctrl+u":
			pageSize := v.viewportHeight() / 2
			v.scrollOffset -= pageSize
			if v.scrollOffset < 0 {
				v.scrollOffset = 0
			}

		case msg.String() == "pgdown", msg.String() == "ctrl+d":
			pageSize := v.viewportHeight() / 2
			v.scrollOffset += pageSize
			maxOffset := v.maxScroll()
			if v.scrollOffset > maxOffset {
				v.scrollOffset = maxOffset
			}

		case msg.String() == "home", msg.String() == "g":
			v.scrollOffset = 0

		case msg.String() == "end", msg.String() == "G":
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

// renderContent parses markdown content into styled terminal lines.
func (v *KnowledgeView) renderContent(content string) []string {
	raw := strings.Split(content, "\n")
	var lines []string
	inFrontmatter := false

	for _, line := range raw {
		trimmed := strings.TrimSpace(line)

		// YAML frontmatter: style dimly
		if trimmed == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				lines = append(lines, HelpStyle.Render("  "+line))
				continue
			}
			inFrontmatter = false
			lines = append(lines, HelpStyle.Render("  "+line))
			continue
		}
		if inFrontmatter {
			lines = append(lines, HelpStyle.Render("  "+line))
			continue
		}

		// Headings: bold + colored
		if strings.HasPrefix(trimmed, "#") {
			lines = append(lines, "  "+TitleStyle.Render(line))
			continue
		}

		// Bullet points: highlight marker
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			lines = append(lines, "  "+line)
			continue
		}

		// Code blocks (fenced)
		if strings.HasPrefix(trimmed, "```") {
			lines = append(lines, "  "+HelpStyle.Render(line))
			continue
		}

		// Regular text
		lines = append(lines, "  "+line)
	}

	return lines
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
