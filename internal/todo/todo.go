package todo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type State string

const (
	Open    State = "open"
	Blocked State = "blocked"
	Done    State = "done"
)

type Priority string

const (
	Now     Priority = "now"
	Backlog Priority = "backlog"
)

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

type List struct {
	Title   string
	Created time.Time
	Updated time.Time
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

func ListDir(projectDir string) string {
	return filepath.Join(projectDir, "todos")
}

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

func ListPath(projectDir, listName string) string {
	return filepath.Join(ListDir(projectDir), listName+".md")
}

func LoadList(projectDir, listName string) (*List, error) {
	data, err := os.ReadFile(ListPath(projectDir, listName))
	if err != nil {
		return nil, err
	}
	return Parse(string(data))
}

func SaveList(projectDir, listName string, list *List) error {
	list.Updated = time.Now().UTC()
	data := Render(list)
	return os.WriteFile(ListPath(projectDir, listName), []byte(data), 0o644)
}

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
