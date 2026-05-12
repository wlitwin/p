// Package display provides display, filtering, and text utility functions
// shared across CLI commands. These are general-purpose helpers that operate
// on todo items and text content.
package display

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/tui"
)

// FilteredItem pairs a todo item with its original positional ID from the
// unfiltered list. This preserves ID stability so users can target the correct
// item after viewing a filtered list.
type FilteredItem struct {
	OriginalID string     // e.g. "2", "4", "3.1"
	Item       *todo.Item
}

// FilterItems returns items matching the given state, priority, and tag filters,
// annotated with their original positional IDs. Empty filter strings are treated
// as "match all" for that criterion.
func FilterItems(items []*todo.Item, state, priority, tag string) []FilteredItem {
	return filterItemsRecursive(items, state, priority, tag, "", 1)
}

func filterItemsRecursive(items []*todo.Item, state, priority, tag, prefix string, start int) []FilteredItem {
	var result []FilteredItem
	for i, item := range items {
		id := fmt.Sprintf("%s%d", prefix, start+i)
		matches := true
		if state != "" && string(item.State) != state {
			matches = false
		}
		if priority != "" && string(item.Priority) != priority {
			matches = false
		}
		if tag != "" && !HasTag(item, tag) {
			matches = false
		}

		if matches {
			result = append(result, FilteredItem{OriginalID: id, Item: item})
		}

		// Recurse into children — include matching children even if parent doesn't match
		if len(item.Children) > 0 {
			childResults := filterItemsRecursive(item.Children, state, priority, tag, id+".", 1)
			result = append(result, childResults...)
		}
	}
	return result
}

// DueFilter filters items by due date range, returning matching items with their
// original positional IDs. Supported dueRange values: "today", "overdue",
// "week" (next 7 days inclusive), "month" (next 30 days inclusive),
// "none" (no due date), or a specific "YYYY-MM-DD" date.
// Items matching by due date are always returned; children are recursed into.
func DueFilter(items []*todo.Item, dueRange string, today time.Time) []FilteredItem {
	return dueFilterRecursive(items, dueRange, today, "", 1)
}

func dueFilterRecursive(items []*todo.Item, dueRange string, today time.Time, prefix string, start int) []FilteredItem {
	todayStr := today.Format("2006-01-02")
	weekEnd := today.AddDate(0, 0, 7).Format("2006-01-02")
	monthEnd := today.AddDate(0, 0, 30).Format("2006-01-02")

	var result []FilteredItem
	for i, item := range items {
		id := fmt.Sprintf("%s%d", prefix, start+i)

		if matchesDue(item, dueRange, todayStr, weekEnd, monthEnd) {
			result = append(result, FilteredItem{OriginalID: id, Item: item})
		}

		if len(item.Children) > 0 {
			childResults := dueFilterRecursive(item.Children, dueRange, today, id+".", 1)
			result = append(result, childResults...)
		}
	}
	return result
}

func matchesDue(item *todo.Item, dueRange, todayStr, weekEnd, monthEnd string) bool {
	switch dueRange {
	case "none":
		return item.Due == ""
	case "today":
		return item.Due == todayStr
	case "overdue":
		return item.Due != "" && item.Due < todayStr && item.State != todo.Done
	case "week":
		return item.Due != "" && item.Due >= todayStr && item.Due <= weekEnd
	case "month":
		return item.Due != "" && item.Due >= todayStr && item.Due <= monthEnd
	default:
		// Treat as a specific YYYY-MM-DD date
		return item.Due == dueRange
	}
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

// PrintFilteredItems prints filtered items using their original positional IDs
// rather than sequential numbering. This ensures users can target items by the
// displayed ID even after filtering.
func PrintFilteredItems(items []FilteredItem, projectDir ...string) {
	dir := ""
	if len(projectDir) > 0 {
		dir = projectDir[0]
	}

	for _, fi := range items {
		item := fi.Item
		marker := "[ ]"
		switch item.State {
		case todo.Done:
			marker = "[x]"
		case todo.Blocked:
			marker = "[-]"
		}

		styledMarker := tui.StateStyle(marker)
		styledID := tui.Dim.Render(fi.OriginalID + ".")

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
