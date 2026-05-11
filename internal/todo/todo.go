// Package todo provides parsing, rendering, and CRUD operations for markdown-based
// todo lists with YAML frontmatter, checkbox items, inline metadata, and nested sub-items.
package todo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StateMarker returns the markdown checkbox marker for a state.
func StateMarker(s State) string {
	return stateMarker(s)
}

// State represents the completion state of a todo item.
type State string

const (
	Open    State = "open"
	Blocked State = "blocked"
	Done    State = "done"
)

// Priority indicates the urgency of a todo item.
type Priority string

const (
	Now     Priority = "now"
	Backlog Priority = "backlog"
)

// Item represents a single todo item with optional metadata and nested children.
type Item struct {
	Text     string
	State    State
	Priority Priority
	Due      string
	Created  string
	DoneDate string
	Tags     []string
	Recur    string
	Children []*Item
}

// List represents a todo list with YAML frontmatter metadata and a tree of items.
type List struct {
	Title   string
	Created time.Time
	Updated time.Time
	Context []string // knowledge doc glob patterns for AI context scoping
	Items   []*Item
}

func stateMarker(s State) string {
	switch s {
	case Done:
		return "[x]"
	case Blocked:
		return "[-]"
	default:
		return "[ ]"
	}
}

// ParseState converts a markdown checkbox marker (e.g. "[x]", "[-]", "[ ]") to a State.
func ParseState(marker string) State {
	switch marker {
	case "[x]":
		return Done
	case "[-]":
		return Blocked
	default:
		return Open
	}
}

// ListDir returns the absolute path to the todos directory within a project.
func ListDir(projectDir string) string {
	return filepath.Join(projectDir, "todos")
}

// ListNames returns the names of all non-archived todo lists in the project,
// derived from .md filenames in the todos directory.
func ListNames(projectDir string) ([]string, error) {
	dir := ListDir(projectDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
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

// ArchivedListNames returns the names of all archived todo lists in the
// project's .archive directory.
func ArchivedListNames(projectDir string) ([]string, error) {
	dir := filepath.Join(ListDir(projectDir), ".archive")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
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

// ListPath returns the absolute file path for a named todo list.
func ListPath(projectDir, listName string) string {
	return filepath.Join(ListDir(projectDir), listName+".md")
}

// LoadList reads and parses a todo list from disk by name.
func LoadList(projectDir, listName string) (*List, error) {
	data, err := os.ReadFile(ListPath(projectDir, listName))
	if err != nil {
		return nil, err
	}
	return Parse(string(data))
}

// SaveList renders a todo list to markdown and writes it to disk,
// updating the list's Updated timestamp.
func SaveList(projectDir, listName string, list *List) error {
	list.Updated = time.Now().UTC()
	data := Render(list)
	return os.WriteFile(ListPath(projectDir, listName), []byte(data), 0o644)
}

// CreateList creates a new todo list with the given title and writes it to disk.
// Returns an error if a list with that name already exists.
func CreateList(projectDir, listName, title string) (*List, error) {
	path := ListPath(projectDir, listName)
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("list %q already exists", listName)
	}

	list := &List{
		Title:   title,
		Created: time.Now().UTC(),
		Updated: time.Now().UTC(),
	}
	if err := SaveList(projectDir, listName, list); err != nil {
		return nil, err
	}
	return list, nil
}

// AddItem appends a new open item to the list and returns it.
func AddItem(list *List, text string, priority Priority, due string) *Item {
	item := &Item{
		Text:     text,
		State:    Open,
		Priority: priority,
		Due:      due,
		Created:  time.Now().UTC().Format("2006-01-02"),
	}
	list.Items = append(list.Items, item)
	return item
}

// ResolveItem finds an item by its positional ID (e.g. "2", "3.1").
// IDs are 1-based and dot-separated for nested children.
func ResolveItem(list *List, id string) (*Item, error) {
	parts := strings.Split(id, ".")
	items := list.Items

	for i, part := range parts {
		idx := 0
		if _, err := fmt.Sscanf(part, "%d", &idx); err != nil || idx < 1 || idx > len(items) {
			return nil, fmt.Errorf("invalid item id %q", id)
		}
		item := items[idx-1]
		if i == len(parts)-1 {
			return item, nil
		}
		items = item.Children
	}
	return nil, fmt.Errorf("invalid item id %q", id)
}

// RemoveItem deletes an item from the list by its positional ID,
// including nested children if the target is a parent.
func RemoveItem(list *List, id string) error {
	parts := strings.Split(id, ".")
	if len(parts) == 1 {
		idx := 0
		if _, err := fmt.Sscanf(parts[0], "%d", &idx); err != nil || idx < 1 || idx > len(list.Items) {
			return fmt.Errorf("invalid item id %q", id)
		}
		list.Items = append(list.Items[:idx-1], list.Items[idx:]...)
		return nil
	}

	parentID := strings.Join(parts[:len(parts)-1], ".")
	parent, err := ResolveItem(list, parentID)
	if err != nil {
		return err
	}

	childIdx := 0
	if _, err := fmt.Sscanf(parts[len(parts)-1], "%d", &childIdx); err != nil || childIdx < 1 || childIdx > len(parent.Children) {
		return fmt.Errorf("invalid item id %q", id)
	}
	parent.Children = append(parent.Children[:childIdx-1], parent.Children[childIdx:]...)
	return nil
}

// DeepCopyItem returns a deep copy of an item including its tags and children.
func DeepCopyItem(item *Item) *Item {
	cp := *item
	if len(item.Tags) > 0 {
		cp.Tags = make([]string, len(item.Tags))
		copy(cp.Tags, item.Tags)
	}
	if len(item.Children) > 0 {
		cp.Children = make([]*Item, len(item.Children))
		for i, child := range item.Children {
			cp.Children[i] = DeepCopyItem(child)
		}
	}
	return &cp
}

// CountStates recursively counts items by state.
func CountStates(items []*Item) (open, done, blocked int) {
	for _, item := range items {
		switch item.State {
		case Open:
			open++
		case Done:
			done++
		case Blocked:
			blocked++
		}
		co, cd, cb := CountStates(item.Children)
		open += co
		done += cd
		blocked += cb
	}
	return
}

// SearchResult holds a matched item with its location metadata.
type SearchResult struct {
	ProjectName string
	ListName    string
	ItemID      string
	Item        *Item
}

// SearchItems recursively searches items for a query string (case-insensitive)
// and returns matching results with positional IDs.
func SearchItems(items []*Item, projectName, listName, prefix string, start int, queryLower string) []SearchResult {
	var results []SearchResult
	for i, item := range items {
		id := fmt.Sprintf("%s%d", prefix, start+i)
		if strings.Contains(strings.ToLower(item.Text), queryLower) {
			results = append(results, SearchResult{
				ProjectName: projectName,
				ListName:    listName,
				ItemID:      id,
				Item:        item,
			})
		}
		if len(item.Children) > 0 {
			results = append(results, SearchItems(item.Children, projectName, listName, id+".", 1, queryLower)...)
		}
	}
	return results
}

// SetState changes an item's state. For recurring tasks marked done, the item
// is reopened with its due date advanced to the next occurrence.
func SetState(item *Item, state State) {
	if state == Done && item.Recur != "" {
		// Recurring task: mark done but immediately reopen
		item.DoneDate = time.Now().UTC().Format("2006-01-02")
		item.State = Open
		item.Due = nextDueDate(item.Recur, item.Due)
		return
	}

	item.State = state
	if state == Done {
		item.DoneDate = time.Now().UTC().Format("2006-01-02")
	} else {
		item.DoneDate = ""
	}
}

func nextDueDate(recur, currentDue string) string {
	now := time.Now().UTC()
	base := now
	if currentDue != "" {
		if t, err := time.Parse("2006-01-02", currentDue); err == nil {
			base = t
		}
	}

	var next time.Time
	switch recur {
	case "daily":
		next = base.AddDate(0, 0, 1)
	case "weekly":
		next = base.AddDate(0, 0, 7)
	case "monthly":
		next = base.AddDate(0, 1, 0)
	default:
		return currentDue
	}

	// If the computed next date is still in the past, advance to the next
	// occurrence from today instead
	if next.Before(now) {
		switch recur {
		case "daily":
			next = now.AddDate(0, 0, 1)
		case "weekly":
			next = now.AddDate(0, 0, 7)
		case "monthly":
			next = now.AddDate(0, 1, 0)
		}
	}

	return next.Format("2006-01-02")
}
