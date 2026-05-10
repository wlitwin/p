package todo

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var itemRe = regexp.MustCompile(`^(\s*)- \[([ x\-])\] (.+)$`)
var metaRe = regexp.MustCompile(`\b(priority|due|created|done|blocked-by|tags|recur)=(\S+)`)

func Parse(content string) (*List, error) {
	lines := strings.Split(content, "\n")
	list := &List{}

	inFrontmatter := false
	frontmatterDone := false
	var bodyLines []string
	var frontmatterLines []string

	for _, line := range lines {
		if !frontmatterDone {
			trimmed := strings.TrimSpace(line)
			if trimmed == "---" {
				if inFrontmatter {
					frontmatterDone = true
					for _, fl := range frontmatterLines {
						parseFrontmatterLine(list, fl)
					}
					continue
				}
				inFrontmatter = true
				continue
			}
			if inFrontmatter {
				frontmatterLines = append(frontmatterLines, trimmed)
				continue
			}
		}
		bodyLines = append(bodyLines, line)
	}

	// If frontmatter was never closed, treat all buffered lines as body
	if inFrontmatter && !frontmatterDone {
		bodyLines = append([]string{"---"}, append(frontmatterLines, bodyLines...)...)
	}

	list.Items = parseItems(bodyLines, 0)
	return list, nil
}

func parseFrontmatterLine(list *List, line string) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return
	}
	key := strings.TrimSpace(parts[0])
	val := strings.TrimSpace(parts[1])

	switch key {
	case "title":
		list.Title = val
	case "created":
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			list.Created = t
		}
	case "updated":
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			list.Updated = t
		}
	}
}

func parseItems(lines []string, baseIndent int) []*Item {
	var items []*Item
	i := 0
	for i < len(lines) {
		line := lines[i]
		m := itemRe.FindStringSubmatch(line)
		if m == nil {
			i++
			continue
		}

		indent := len(m[1])
		if indent < baseIndent {
			break
		}
		if indent > baseIndent {
			i++
			continue
		}

		stateChar := m[2]
		rest := m[3]

		item := &Item{
			State:    ParseState(fmt.Sprintf("[%s]", stateChar)),
			Priority: Now,
		}

		metas := metaRe.FindAllStringSubmatchIndex(rest, -1)
		textEnd := len(rest)
		if len(metas) > 0 {
			textEnd = metas[0][0]
		}
		item.Text = strings.TrimSpace(rest[:textEnd])

		for _, loc := range metaRe.FindAllStringSubmatch(rest, -1) {
			switch loc[1] {
			case "priority":
				item.Priority = Priority(loc[2])
			case "due":
				item.Due = loc[2]
			case "created":
				item.Created = loc[2]
			case "done":
				item.DoneDate = loc[2]
			case "tags":
				item.Tags = strings.Split(loc[2], ",")
			case "recur":
				item.Recur = loc[2]
			}
		}

		// collect children
		childStart := i + 1
		childEnd := childStart
		for childEnd < len(lines) {
			cm := itemRe.FindStringSubmatch(lines[childEnd])
			if cm != nil {
				childIndent := len(cm[1])
				if childIndent <= indent {
					break
				}
			} else if strings.TrimSpace(lines[childEnd]) != "" {
				trimmed := lines[childEnd]
				leadingSpaces := len(trimmed) - len(strings.TrimLeft(trimmed, " "))
				if leadingSpaces <= indent {
					break
				}
			}
			childEnd++
		}

		if childEnd > childStart {
			item.Children = parseItems(lines[childStart:childEnd], indent+2)
		}

		items = append(items, item)
		i = childEnd
	}
	return items
}

func Render(list *List) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("title: %s\n", list.Title))
	sb.WriteString(fmt.Sprintf("created: %s\n", list.Created.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("updated: %s\n", list.Updated.Format(time.RFC3339)))
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %s\n", list.Title))

	if len(list.Items) > 0 {
		sb.WriteString("\n")
		renderItems(&sb, list.Items, 0)
	}

	return sb.String()
}

func renderItems(sb *strings.Builder, items []*Item, indent int) {
	prefix := strings.Repeat("  ", indent)
	for _, item := range items {
		sb.WriteString(fmt.Sprintf("%s- %s %s", prefix, stateMarker(item.State), item.Text))

		var meta []string
		if item.Priority != "" && item.Priority != Now {
			meta = append(meta, fmt.Sprintf("priority=%s", item.Priority))
		} else if indent == 0 {
			meta = append(meta, fmt.Sprintf("priority=%s", item.Priority))
		}
		if item.Due != "" {
			meta = append(meta, fmt.Sprintf("due=%s", item.Due))
		}
		if item.Created != "" {
			meta = append(meta, fmt.Sprintf("created=%s", item.Created))
		}
		if item.DoneDate != "" {
			meta = append(meta, fmt.Sprintf("done=%s", item.DoneDate))
		}
		if len(item.Tags) > 0 {
			meta = append(meta, fmt.Sprintf("tags=%s", strings.Join(item.Tags, ",")))
		}
		if item.Recur != "" {
			meta = append(meta, fmt.Sprintf("recur=%s", item.Recur))
		}

		if len(meta) > 0 {
			sb.WriteString(" " + strings.Join(meta, " "))
		}
		sb.WriteString("\n")

		if len(item.Children) > 0 {
			renderItems(sb, item.Children, indent+1)
		}
	}
}
