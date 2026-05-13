package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// wrapLine wraps text to fit within maxWidth characters per line.
// It wraps at word boundaries when possible, and hard-wraps single words
// that exceed the available width. Continuation lines are prepended with
// continuationIndent spaces. The text portion of every line (excluding
// the indent) fits within maxWidth characters, so all output lines have
// the same total visual width when the caller prepends its prefix to
// the first line.
func wrapLine(text string, maxWidth, continuationIndent int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	currentLine := ""

	for _, word := range words {
		if currentLine == "" {
			// Start of a new line — hard-wrap words longer than maxWidth
			for len(word) > maxWidth {
				lines = append(lines, word[:maxWidth])
				word = word[maxWidth:]
			}
			currentLine = word
		} else if len(currentLine)+1+len(word) <= maxWidth {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			// Hard-wrap words longer than maxWidth
			for len(word) > maxWidth {
				lines = append(lines, word[:maxWidth])
				word = word[maxWidth:]
			}
			currentLine = word
		}
	}
	if currentLine != "" || len(lines) == 0 {
		lines = append(lines, currentLine)
	}

	// Prepend continuation indent to lines after the first
	if continuationIndent > 0 {
		indent := strings.Repeat(" ", continuationIndent)
		for i := 1; i < len(lines); i++ {
			lines[i] = indent + lines[i]
		}
	}

	return lines
}

// truncateText truncates text with an ellipsis when it exceeds maxWidth.
// Width is measured using lipgloss.Width to correctly handle ANSI escape
// sequences in styled text.
func truncateText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= maxWidth {
		return text
	}
	if maxWidth <= 1 {
		return "…"
	}
	// Truncate rune-by-rune to find the longest prefix that fits with ellipsis
	runes := []rune(text)
	for i := len(runes) - 1; i >= 0; i-- {
		candidate := string(runes[:i]) + "…"
		if lipgloss.Width(candidate) <= maxWidth {
			return candidate
		}
	}
	return "…"
}

// calcScrollOffset returns an adjusted scroll offset that ensures the cursor
// item is fully visible within the viewport. itemHeights contains the number
// of rendered lines for each item.
func calcScrollOffset(itemHeights []int, cursor, scrollOffset, viewportHeight int) int {
	if len(itemHeights) == 0 || cursor < 0 || cursor >= len(itemHeights) {
		return 0
	}

	// Clamp to non-negative before calculations
	if scrollOffset < 0 {
		scrollOffset = 0
	}

	// Calculate line offset of cursor item
	cursorLineStart := 0
	for i := 0; i < cursor; i++ {
		cursorLineStart += itemHeights[i]
	}
	cursorLineEnd := cursorLineStart + itemHeights[cursor]

	// If item is taller than viewport, show from top of item
	if itemHeights[cursor] >= viewportHeight {
		return cursorLineStart
	}

	// Scroll up if cursor is above viewport
	if cursorLineStart < scrollOffset {
		return cursorLineStart
	}

	// Scroll down if cursor extends below viewport
	if cursorLineEnd > scrollOffset+viewportHeight {
		return cursorLineEnd - viewportHeight
	}

	return scrollOffset
}
