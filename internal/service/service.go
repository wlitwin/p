// Package service provides shared business logic used by both the CLI and MCP
// server. It encapsulates the common load-modify-save patterns to avoid
// duplication between the two entry points.
//
// Functions in this package perform the data operation (load, modify, save) but
// do NOT commit to git. Callers are responsible for committing when appropriate
// — the CLI commits after each operation, while the MCP server lets the caller
// manage the git lifecycle.
package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/walter/p/internal/asset"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/validate"
)

// SetItemState loads a list, changes an item's state, and saves.
func SetItemState(_ context.Context, dir, listName, itemID string, state todo.State) error {
	list, err := todo.LoadList(dir, listName)
	if err != nil {
		return err
	}

	item, err := todo.ResolveItem(list, itemID)
	if err != nil {
		return err
	}

	todo.SetState(item, state)

	return todo.SaveList(dir, listName, list)
}

// SetItemPriority loads a list, changes an item's priority, and saves.
func SetItemPriority(_ context.Context, dir, listName, itemID string, priority todo.Priority) error {
	list, err := todo.LoadList(dir, listName)
	if err != nil {
		return err
	}

	item, err := todo.ResolveItem(list, itemID)
	if err != nil {
		return err
	}

	item.Priority = priority

	return todo.SaveList(dir, listName, list)
}

// SetItemDue loads a list, changes an item's due date, and saves.
func SetItemDue(_ context.Context, dir, listName, itemID, due string) error {
	list, err := todo.LoadList(dir, listName)
	if err != nil {
		return err
	}

	item, err := todo.ResolveItem(list, itemID)
	if err != nil {
		return err
	}

	item.Due = due

	return todo.SaveList(dir, listName, list)
}

// UpdateItemText loads a list, changes an item's text, and saves.
func UpdateItemText(_ context.Context, dir, listName, itemID, text string) error {
	list, err := todo.LoadList(dir, listName)
	if err != nil {
		return err
	}

	item, err := todo.ResolveItem(list, itemID)
	if err != nil {
		return err
	}

	item.Text = text

	return todo.SaveList(dir, listName, list)
}

// RemoveItem loads a list, removes an item, and saves.
func RemoveItem(_ context.Context, dir, listName, itemID string) error {
	list, err := todo.LoadList(dir, listName)
	if err != nil {
		return err
	}

	if err := todo.RemoveItem(list, itemID); err != nil {
		return err
	}

	return todo.SaveList(dir, listName, list)
}

// AddItem loads (or creates) a list, adds an item, and saves.
// If parentID is non-empty, the item is nested under the specified parent.
func AddItem(_ context.Context, dir, listName, text string, priority todo.Priority, due, parentID string) error {
	if err := validate.ListName(listName); err != nil {
		return err
	}

	list, err := todo.LoadList(dir, listName)
	if err != nil {
		list, err = todo.CreateList(dir, listName, listName)
		if err != nil {
			return fmt.Errorf("creating list: %w", err)
		}
	}

	item := todo.AddItem(list, text, priority, due)

	if parentID != "" {
		parent, err := todo.ResolveItem(list, parentID)
		if err != nil {
			return fmt.Errorf("resolving parent: %w", err)
		}
		list.Items = list.Items[:len(list.Items)-1]
		parent.Children = append(parent.Children, item)
	}

	return todo.SaveList(dir, listName, list)
}

// MoveItem moves an item from one list to another and saves both lists.
func MoveItem(_ context.Context, dir, srcListName, itemID, dstListName string) error {
	srcList, err := todo.LoadList(dir, srcListName)
	if err != nil {
		return fmt.Errorf("loading source list: %w", err)
	}

	item, err := todo.ResolveItem(srcList, itemID)
	if err != nil {
		return err
	}

	itemCopy := todo.DeepCopyItem(item)

	// Write to destination first -- source still has the item if this fails
	dstList, err := todo.LoadList(dir, dstListName)
	if err != nil {
		dstList, err = todo.CreateList(dir, dstListName, dstListName)
		if err != nil {
			return fmt.Errorf("creating target list: %w", err)
		}
	}
	dstList.Items = append(dstList.Items, itemCopy)
	if err := todo.SaveList(dir, dstListName, dstList); err != nil {
		return fmt.Errorf("saving target: %w", err)
	}

	// Only remove from source after destination is safely written
	if err := todo.RemoveItem(srcList, itemID); err != nil {
		return fmt.Errorf("removing from source: %w", err)
	}
	if err := todo.SaveList(dir, srcListName, srcList); err != nil {
		return fmt.Errorf("saving source: %w", err)
	}

	return nil
}

// RemoveList deletes a todo list file and cleans up empty parent directories.
func RemoveList(_ context.Context, dir, listName string) error {
	path := todo.ListPath(dir, listName)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("todo list %q not found", listName)
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleting: %w", err)
	}

	// Clean up empty parent directories for subdirectory lists
	todo.CleanEmptyParents(path, todo.ListDir(dir))

	return nil
}

// ListStatus holds per-list item counts.
type ListStatus struct {
	Name    string
	Open    int
	Done    int
	Blocked int
}

// GetProjectListStatuses returns per-list item counts for a project.
func GetProjectListStatuses(_ context.Context, dir string) ([]ListStatus, error) {
	names, err := todo.ListNames(dir)
	if err != nil {
		return nil, err
	}

	var statuses []ListStatus
	for _, name := range names {
		list, err := todo.LoadList(dir, name)
		if err != nil {
			continue
		}
		open, done, blocked := todo.CountStates(list.Items)
		statuses = append(statuses, ListStatus{
			Name:    name,
			Open:    open,
			Done:    done,
			Blocked: blocked,
		})
	}
	return statuses, nil
}

// ProjectTotals returns aggregate item counts across all lists in a project.
func ProjectTotals(ctx context.Context, dir string) (open, done, blocked int) {
	statuses, err := GetProjectListStatuses(ctx, dir)
	if err != nil {
		return
	}
	for _, s := range statuses {
		open += s.Open
		done += s.Done
		blocked += s.Blocked
	}
	return
}

// SearchMatch holds a search match across todos or knowledge docs.
type SearchMatch struct {
	// Type is "todo" or "knowledge"
	Type string
	// ListName is the todo list name (for todo matches)
	ListName string
	// File is the knowledge doc filename (for knowledge matches)
	File string
	// TodoResults holds matched todo items (for todo matches)
	TodoResults []todo.SearchResult
}

// SearchProject searches all todos and knowledge docs in a project for a query.
func SearchProject(_ context.Context, dir, projectName, queryLower string) []SearchMatch {
	var matches []SearchMatch

	// Search todos
	lists, _ := todo.ListNames(dir)
	for _, listName := range lists {
		list, err := todo.LoadList(dir, listName)
		if err != nil {
			continue
		}
		results := todo.SearchItems(list.Items, projectName, listName, "", 1, queryLower)
		if len(results) > 0 {
			matches = append(matches, SearchMatch{
				Type:        "todo",
				ListName:    listName,
				TodoResults: results,
			})
		}
	}

	// Search knowledge
	kFiles, _ := knowledge.Search(dir, queryLower)
	for _, f := range kFiles {
		matches = append(matches, SearchMatch{
			Type: "knowledge",
			File: f,
		})
	}

	return matches
}

// KnowledgeCreate creates a knowledge doc.
func KnowledgeCreate(_ context.Context, dir, filename, title string, tags []string) error {
	if err := validate.Filename(filename); err != nil {
		return err
	}
	return knowledge.Create(dir, filename, title, tags)
}

// KnowledgeAppend appends content to a knowledge doc.
func KnowledgeAppend(_ context.Context, dir, filename, content, section string) error {
	return knowledge.Append(dir, filename, content, section)
}

// KnowledgeReplace replaces a section in a knowledge doc.
func KnowledgeReplace(_ context.Context, dir, filename, section, content string) error {
	return knowledge.ReplaceSection(dir, filename, section, content)
}

// KnowledgeRename renames a knowledge doc.
func KnowledgeRename(_ context.Context, dir, oldName, newName string) error {
	return knowledge.Rename(dir, oldName, newName)
}

// KnowledgeDelete deletes a knowledge doc.
func KnowledgeDelete(_ context.Context, dir, filename string) error {
	return knowledge.Delete(dir, filename)
}

// ProjectCreate creates a new project with directory structure and git init.
func ProjectCreate(ctx context.Context, projectRoot, name, description string) error {
	if err := validate.ProjectName(name); err != nil {
		return err
	}

	if err := project.Create(projectRoot, name, description); err != nil {
		return err
	}

	dir := fmt.Sprintf("%s/%s", projectRoot, name)
	if err := git.Init(ctx, dir); err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	_ = git.CommitAll(ctx, dir, fmt.Sprintf("p: create project %q", name))
	return nil
}

// ProjectArchive sets the archived state on a project and commits.
func ProjectArchive(ctx context.Context, dir, projectName string, archived bool) error {
	meta, err := project.LoadMeta(dir)
	if err != nil {
		return err
	}

	meta.Archived = archived
	if err := project.SaveMeta(dir, meta); err != nil {
		return err
	}

	action := "archived"
	if !archived {
		action = "unarchived"
	}
	_ = git.CommitAll(ctx, dir, fmt.Sprintf("p: %s project %q", action, projectName))
	return nil
}

// SetItemTags loads a list, modifies tags on an item, and saves.
func SetItemTags(_ context.Context, dir, listName, itemID string, tags []string, remove bool) ([]string, error) {
	list, err := todo.LoadList(dir, listName)
	if err != nil {
		return nil, err
	}

	item, err := todo.ResolveItem(list, itemID)
	if err != nil {
		return nil, err
	}

	if remove {
		item.Tags = removeTags(item.Tags, tags)
	} else {
		item.Tags = addTags(item.Tags, tags)
	}

	if err := todo.SaveList(dir, listName, list); err != nil {
		return nil, err
	}

	return item.Tags, nil
}

func addTags(existing, toAdd []string) []string {
	set := make(map[string]bool)
	for _, t := range existing {
		set[t] = true
	}
	for _, t := range toAdd {
		if !set[t] {
			existing = append(existing, t)
			set[t] = true
		}
	}
	return existing
}

func removeTags(existing, toRemove []string) []string {
	remove := make(map[string]bool)
	for _, t := range toRemove {
		remove[t] = true
	}
	var result []string
	for _, t := range existing {
		if !remove[t] {
			result = append(result, t)
		}
	}
	return result
}

// SetListContext loads a list, sets (or clears) its context patterns, and saves.
// If patterns is nil, the context field is removed (reverts to default/all).
func SetListContext(_ context.Context, dir, listName string, patterns []string) error {
	list, err := todo.LoadList(dir, listName)
	if err != nil {
		return err
	}

	list.Context = patterns
	return todo.SaveList(dir, listName, list)
}

// SetDefaultContext loads project meta, sets (or clears) the default context
// patterns, and saves.
func SetDefaultContext(_ context.Context, dir string, patterns []string) error {
	meta, err := project.LoadMeta(dir)
	if err != nil {
		return err
	}

	meta.DefaultContext = patterns
	return project.SaveMeta(dir, meta)
}

// AssetAdd copies a file into the project's assets directory.
func AssetAdd(_ context.Context, dir, srcPath string) (string, error) {
	return asset.Copy(dir, srcPath)
}

// AssetList returns the names of all assets in a project.
func AssetList(_ context.Context, dir string) ([]asset.Info, error) {
	return asset.ListWithInfo(dir)
}

// AssetDelete removes an asset from a project.
func AssetDelete(_ context.Context, dir, filename string) error {
	return asset.Delete(dir, filename)
}

// Commit is a convenience wrapper around git.CommitAll for callers (like the
// CLI) that need to commit after a service operation.
func Commit(ctx context.Context, dir, message string) error {
	return git.CommitAll(ctx, dir, message)
}

// ArchiveList moves a todo list to the .archive directory, or restores it.
func ArchiveList(ctx context.Context, dir, listName string, restore bool) error {
	archiveDir := filepath.Join(todo.ListDir(dir), ".archive")
	activePath := todo.ListPath(dir, listName)
	archivedPath := filepath.Join(archiveDir, listName+".md")

	if restore {
		if _, err := os.Stat(archivedPath); err != nil {
			return fmt.Errorf("archived list %q not found", listName)
		}
		if err := os.MkdirAll(filepath.Dir(activePath), 0o755); err != nil {
			return err
		}
		if err := os.Rename(archivedPath, activePath); err != nil {
			return err
		}
		todo.CleanEmptyParents(archivedPath, archiveDir)
		return nil
	}

	if _, err := os.Stat(activePath); err != nil {
		return fmt.Errorf("todo list %q not found", listName)
	}
	if err := os.MkdirAll(filepath.Dir(archivedPath), 0o755); err != nil {
		return err
	}
	if err := os.Rename(activePath, archivedPath); err != nil {
		return err
	}
	todo.CleanEmptyParents(activePath, todo.ListDir(dir))
	return nil
}

// AutoArchiveDone archives all todo lists where every item is done.
// Returns the names of archived lists.
func AutoArchiveDone(ctx context.Context, dir string) ([]string, error) {
	names, err := todo.ListNames(dir)
	if err != nil {
		return nil, err
	}

	var archived []string
	for _, name := range names {
		list, err := todo.LoadList(dir, name)
		if err != nil || len(list.Items) == 0 {
			continue
		}
		if allDone(list.Items) {
			if err := ArchiveList(ctx, dir, name, false); err != nil {
				continue
			}
			archived = append(archived, name)
		}
	}
	return archived, nil
}

func allDone(items []*todo.Item) bool {
	for _, item := range items {
		if item.State != todo.Done {
			return false
		}
		if len(item.Children) > 0 && !allDone(item.Children) {
			return false
		}
	}
	return true
}

// KnowledgeArchive moves a knowledge doc to the .archive directory, or restores it.
func KnowledgeArchive(ctx context.Context, dir, filename string, restore bool) error {
	archiveDir := filepath.Join(knowledge.Dir(dir), ".archive")
	activePath := knowledge.FilePath(dir, filename)
	archivedPath := filepath.Join(archiveDir, filename+".md")

	if restore {
		if _, err := os.Stat(archivedPath); err != nil {
			return fmt.Errorf("archived doc %q not found", filename)
		}
		if err := os.MkdirAll(filepath.Dir(activePath), 0o755); err != nil {
			return err
		}
		return os.Rename(archivedPath, activePath)
	}

	if _, err := os.Stat(activePath); err != nil {
		return fmt.Errorf("knowledge doc %q not found", filename)
	}
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return err
	}
	return os.Rename(activePath, archivedPath)
}

// KnowledgeArchiveRefs returns todo lists that reference a knowledge doc in their context patterns.
func KnowledgeArchiveRefs(dir, filename string) []string {
	return knowledge.FindReferencingLists(dir, filename)
}

// ProjectRename renames a project directory and updates its metadata.
func ProjectRename(ctx context.Context, projectRoot, oldName, newName string) error {
	if oldName == newName {
		return fmt.Errorf("old and new names are the same")
	}
	if err := validate.ProjectName(newName); err != nil {
		return fmt.Errorf("invalid new name: %w", err)
	}

	oldDir, err := project.Resolve(projectRoot, oldName)
	if err != nil {
		return err
	}

	newDir := filepath.Join(projectRoot, newName)
	if _, err := os.Stat(newDir); err == nil {
		return fmt.Errorf("project %q already exists", newName)
	}

	if err := os.Rename(oldDir, newDir); err != nil {
		return fmt.Errorf("renaming directory: %w", err)
	}

	meta, err := project.LoadMeta(newDir)
	if err != nil {
		_ = os.Rename(newDir, oldDir)
		return fmt.Errorf("loading metadata: %w", err)
	}

	meta.Name = newName
	if err := project.SaveMeta(newDir, meta); err != nil {
		_ = os.Rename(newDir, oldDir)
		return fmt.Errorf("saving metadata: %w", err)
	}

	return nil
}
