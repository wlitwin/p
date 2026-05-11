// Package display provides display, filtering, and text utility functions
// shared across CLI commands. These are general-purpose helpers that operate
// on todo items and text content.
package display

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/tui"
)

// FilterItems returns items matching the given state, priority, and tag filters.
// Empty filter strings are treated as "match all" for that criterion.
func FilterItems(items []*todo.Item, state, priority, tag string) []*todo.Item {
	if state == "" && priority == "" && tag == "" {
		return items
	}

	var result []*todo.Item
	for _, item := range items {
		if state != "" && string(item.State) != state {
			continue
		}
		if priority != "" && string(item.Priority) != priority {
			continue
		}
		if tag != "" && !HasTag(item, tag) {
			continue
		}
		result = append(result, item)
	}
	return result
}

// HasTag returns true if the item contains the given tag.
func HasTag(item *todo.Item, tag string) bool {
	return slices.Contains(item.Tags, tag)
}

var wikiLinkSplit = regexp.MustCompile(`\[\[[^\]]+\]\]`)

// DimTextPreservingLinks dims text segments between wiki links, leaving
// [[...]] markers untouched so they can be rendered as clickable links.
func DimTextPreservingLinks(text string) string {
	parts := wikiLinkSplit.Split(text, -1)
	links := wikiLinkSplit.FindAllString(text, -1)

	var sb strings.Builder
	for i, part := range parts {
		sb.WriteString(tui.Dim.Render(part))
		if i < len(links) {
			sb.WriteString(links[i])
		}
	}
	return sb.String()
}

// PrintItems prints todo items to stdout with colored markers, IDs, and
// metadata. It recurses into children, building dotted IDs (e.g. "1.2.1").
// If projectDir is provided, wiki links in item text are rendered as
// clickable terminal hyperlinks.
func PrintItems(items []*todo.Item, prefix string, start int, projectDir ...string) {
	dir := ""
	if len(projectDir) > 0 {
		dir = projectDir[0]
	}

	for i, item := range items {
		id := fmt.Sprintf("%s%d", prefix, start+i)
		marker := "[ ]"
		switch item.State {
		case todo.Done:
			marker = "[x]"
		case todo.Blocked:
			marker = "[-]"
		}

		styledMarker := tui.StateStyle(marker)
		styledID := tui.Dim.Render(id + ".")

		var meta string
		if item.Priority == todo.Backlog {
			meta += " " + tui.Dim.Render("priority=backlog")
		}
		if item.Due != "" {
			meta += " " + tui.Cyan.Render("due="+item.Due)
		}
		if item.DoneDate != "" {
			meta += " " + tui.Green.Render("done="+item.DoneDate)
		}

		text := item.Text
		if item.State == todo.Done {
			text = DimTextPreservingLinks(text)
		}
		if dir != "" {
			text = tui.RenderWikiLinks(text, dir)
		}

		fmt.Printf("  %s %s %s%s\n", styledID, styledMarker, text, meta)

		if len(item.Children) > 0 {
			PrintItems(item.Children, id+".", 1, dir)
		}
	}
}

// Truncate shortens s to max runes, appending "..." if truncated.
func Truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

// LooksLikeURL returns true if s starts with http:// or https://.
func LooksLikeURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// MatchContext returns a short snippet of content surrounding the first
// occurrence of query, with "..." ellipsis if the snippet is truncated.
// Both content and query are matched case-insensitively.
func MatchContext(content, query string) string {
	runes := []rune(content)
	lowerRunes := []rune(strings.ToLower(content))
	queryRunes := []rune(strings.ToLower(query))

	idx := -1
	for i := 0; i <= len(lowerRunes)-len(queryRunes); i++ {
		if string(lowerRunes[i:i+len(queryRunes)]) == string(queryRunes) {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ""
	}

	start := max(idx-30, 0)
	end := min(idx+len(queryRunes)+30, len(runes))

	snippet := strings.ReplaceAll(string(runes[start:end]), "\n", " ")
	snippet = strings.TrimSpace(snippet)

	prefix := ""
	if start > 0 {
		prefix = "..."
	}
	suffix := ""
	if end < len(runes) {
		suffix = "..."
	}

	return tui.Dim.Render(prefix + snippet + suffix)
}

// ContainsIgnoreCase reports whether s contains substr, ignoring case.
func ContainsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
