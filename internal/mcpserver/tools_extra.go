package mcpserver

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/service"
	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/validate"
)

// --- Tool definitions ---

func projectCreateTool() mcp.Tool {
	return mcp.NewTool("project_create",
		mcp.WithDescription("Create a new project with directory structure and git init."),
		mcp.WithString("name", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Project description")),
	)
}

func projectArchiveTool() mcp.Tool {
	return mcp.NewTool("project_archive",
		mcp.WithDescription("Archive or unarchive a project."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithBoolean("archived", mcp.Description("true to archive, false to unarchive"), mcp.Required()),
	)
}

func statusTool() mcp.Tool {
	return mcp.NewTool("status",
		mcp.WithDescription("Get aggregate status of a project or all projects. Shows open/blocked/done counts per list."),
		mcp.WithString("project", mcp.Description("Project name. If omitted, shows all projects.")),
	)
}

func searchTool() mcp.Tool {
	return mcp.NewTool("search",
		mcp.WithDescription("Full-text search across todos and knowledge docs."),
		mcp.WithString("query", mcp.Description("Search query"), mcp.Required()),
		mcp.WithString("project", mcp.Description("Project name. If omitted, searches all projects.")),
	)
}

func todoPriorityTool() mcp.Tool {
	return mcp.NewTool("todo_priority",
		mcp.WithDescription("Set the priority of a todo item."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("list", mcp.Description("Todo list name"), mcp.Required()),
		mcp.WithString("item_id", mcp.Description("Item ID (e.g. '1' or '2.1')"), mcp.Required()),
		mcp.WithString("priority", mcp.Description("Priority: now or backlog"), mcp.Required()),
	)
}

func todoDueTool() mcp.Tool {
	return mcp.NewTool("todo_due",
		mcp.WithDescription("Set the due date of a todo item."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("list", mcp.Description("Todo list name"), mcp.Required()),
		mcp.WithString("item_id", mcp.Description("Item ID (e.g. '1' or '2.1')"), mcp.Required()),
		mcp.WithString("due", mcp.Description("Due date in YYYY-MM-DD format"), mcp.Required()),
	)
}

func todoRmListTool() mcp.Tool {
	return mcp.NewTool("todo_rm_list",
		mcp.WithDescription("Delete an entire todo list."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("list", mcp.Description("Todo list name to delete (supports subdirectory paths like 'sprint/week-1')"), mcp.Required()),
	)
}

func knowledgeListTool() mcp.Tool {
	return mcp.NewTool("knowledge_list",
		mcp.WithDescription("List knowledge documents with tags and sizes."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("tag", mcp.Description("Filter by tag")),
	)
}

func knowledgeSearchTool() mcp.Tool {
	return mcp.NewTool("knowledge_search",
		mcp.WithDescription("Full-text search across knowledge documents."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("query", mcp.Description("Search query"), mcp.Required()),
	)
}

func assetAddTool() mcp.Tool {
	return mcp.NewTool("asset_add",
		mcp.WithDescription("Add a file to the project's assets directory. Copies the file from the given source path."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("source_path", mcp.Description("Absolute path to the file to add"), mcp.Required()),
	)
}

func assetListTool() mcp.Tool {
	return mcp.NewTool("asset_list",
		mcp.WithDescription("List all assets in a project with file sizes."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
	)
}

func assetRemoveTool() mcp.Tool {
	return mcp.NewTool("asset_remove",
		mcp.WithDescription("Remove an asset from a project."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("filename", mcp.Description("Asset filename to remove"), mcp.Required()),
	)
}

func todoArchiveListTool() mcp.Tool {
	return mcp.NewTool("todo_archive_list",
		mcp.WithDescription("Archive a completed todo list, or restore one from the archive. If no list is specified, auto-archives all lists where every item is done."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("list", mcp.Description("Todo list name. If omitted, auto-archives all fully-done lists.")),
		mcp.WithBoolean("restore", mcp.Description("If true, restores the list from the archive instead of archiving it")),
	)
}

func todoTagTool() mcp.Tool {
	return mcp.NewTool("todo_tag",
		mcp.WithDescription("Add or remove tags on a todo item."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("list", mcp.Description("Todo list name"), mcp.Required()),
		mcp.WithString("item_id", mcp.Description("Item ID (e.g. '1' or '2.1')"), mcp.Required()),
		mcp.WithString("tags", mcp.Description("Comma-separated tags to add or remove"), mcp.Required()),
		mcp.WithBoolean("remove", mcp.Description("If true, removes the specified tags instead of adding them")),
	)
}

func knowledgeArchiveTool() mcp.Tool {
	return mcp.NewTool("knowledge_archive",
		mcp.WithDescription("Archive a knowledge document to .archive/, or restore one from the archive."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("filename", mcp.Description("Knowledge doc filename (without .md)"), mcp.Required()),
		mcp.WithBoolean("restore", mcp.Description("If true, restores from archive instead of archiving")),
	)
}

func projectRenameTool() mcp.Tool {
	return mcp.NewTool("project_rename",
		mcp.WithDescription("Rename a project directory and update its metadata."),
		mcp.WithString("old_name", mcp.Description("Current project name"), mcp.Required()),
		mcp.WithString("new_name", mcp.Description("New project name"), mcp.Required()),
	)
}

// --- Handlers ---

func (s *serverCtx) handleProjectCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	name := p.require("name")
	if r := p.error(); r != nil {
		return r, nil
	}

	desc := p.optional("description", "")

	if err := service.ProjectCreate(ctx, s.projectRoot, name, desc); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Created project %q", name)), nil
}

func (s *serverCtx) handleProjectArchive(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	if r := p.error(); r != nil {
		return r, nil
	}

	archived := p.optionalBool("archived", true)

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.ProjectArchive(ctx, dir, proj, archived); err != nil {
		return errResult("%v", err)
	}

	action := "archived"
	if !archived {
		action = "unarchived"
	}
	return textResult(fmt.Sprintf("Project %q %s", proj, action)), nil
}

func (s *serverCtx) handleStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.optional("project", "")

	if proj != "" {
		return s.projectStatus(ctx, proj)
	}

	projects, err := project.List(s.projectRoot, false)
	if err != nil {
		return errResult("%v", err)
	}

	var sb strings.Builder
	for _, p := range projects {
		dir, err := project.Resolve(s.projectRoot, p.Name)
		if err != nil {
			continue
		}
		totalOpen, totalDone, totalBlocked := service.ProjectTotals(ctx, dir)
		fmt.Fprintf(&sb, "%s: open=%d blocked=%d done=%d\n", p.Name, totalOpen, totalBlocked, totalDone)
	}
	return textResult(sb.String()), nil
}

func (s *serverCtx) projectStatus(ctx context.Context, proj string) (*mcp.CallToolResult, error) {
	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Project: %s\n\n", proj)

	statuses, _ := service.GetProjectListStatuses(ctx, dir)
	if len(statuses) == 0 {
		sb.WriteString("No todo lists.\n")
	} else {
		for _, ls := range statuses {
			fmt.Fprintf(&sb, "  %-20s open=%d blocked=%d done=%d\n", ls.Name, ls.Open, ls.Blocked, ls.Done)
		}
	}

	files, _ := knowledge.ListFiles(dir)
	if len(files) > 0 {
		sb.WriteString("\nKnowledge docs: " + strings.Join(files, ", ") + "\n")
	}

	return textResult(sb.String()), nil
}

func (s *serverCtx) handleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	query := p.require("query")
	if r := p.error(); r != nil {
		return r, nil
	}

	proj := p.optional("project", "")
	queryLower := strings.ToLower(query)

	var projectNames []string
	if proj != "" {
		projectNames = []string{proj}
	} else {
		projects, _ := project.List(s.projectRoot, false)
		for _, p := range projects {
			projectNames = append(projectNames, p.Name)
		}
	}

	var sb strings.Builder
	for _, name := range projectNames {
		dir, err := project.Resolve(s.projectRoot, name)
		if err != nil {
			continue
		}

		matches := service.SearchProject(ctx, dir, name, queryLower)
		for _, m := range matches {
			if m.Type == "todo" {
				for _, r := range m.TodoResults {
					fmt.Fprintf(&sb, "%s/%s#%s %s %s\n",
						r.ProjectName, r.ListName, r.ItemID,
						todo.StateMarker(r.Item.State), r.Item.Text)
				}
			} else {
				fmt.Fprintf(&sb, "%s/knowledge/%s.md: matches query\n", name, m.File)
			}
		}
	}

	if sb.Len() == 0 {
		return textResult("No matches found."), nil
	}
	return textResult(sb.String()), nil
}

func (s *serverCtx) handleTodoPriority(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	listName := p.require("list")
	itemID := p.require("item_id")
	priority := p.require("priority")
	if r := p.error(); r != nil {
		return r, nil
	}

	if err := validate.Priority(priority); err != nil {
		return errResult("%v", err)
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.SetItemPriority(ctx, dir, listName, itemID, todo.Priority(priority)); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Set %s #%s priority to %s", listName, itemID, priority)), nil
}

func (s *serverCtx) handleTodoDue(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	listName := p.require("list")
	itemID := p.require("item_id")
	due := p.require("due")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.SetItemDue(ctx, dir, listName, itemID, due); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Set %s #%s due to %s", listName, itemID, due)), nil
}

func (s *serverCtx) handleTodoRmList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	listName := p.require("list")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.RemoveList(ctx, dir, listName); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Deleted todo list %q", listName)), nil
}

func (s *serverCtx) handleKnowledgeList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	files, err := knowledge.ListFiles(dir)
	if err != nil {
		return errResult("%v", err)
	}
	if len(files) == 0 {
		return textResult("No knowledge documents."), nil
	}

	tagFilter := p.optional("tag", "")

	var sb strings.Builder
	for _, f := range files {
		content, _ := knowledge.Read(dir, f)
		tags := knowledge.ExtractTags(content)

		if tagFilter != "" && !slices.Contains(tags, tagFilter) {
			continue
		}

		info, _ := os.Stat(knowledge.FilePath(dir, f))
		size := 0
		if info != nil {
			size = int(info.Size())
		}

		tagStr := ""
		if len(tags) > 0 {
			tagStr = " tags=[" + strings.Join(tags, ",") + "]"
		}
		fmt.Fprintf(&sb, "%s (%d bytes)%s\n", f, size, tagStr)
	}

	return textResult(sb.String()), nil
}

func (s *serverCtx) handleKnowledgeSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	query := p.require("query")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	matches, err := knowledge.Search(dir, query)
	if err != nil {
		return errResult("%v", err)
	}

	if len(matches) == 0 {
		return textResult("No matches found."), nil
	}

	var sb strings.Builder
	for _, f := range matches {
		fmt.Fprintf(&sb, "%s.md: matches\n", f)
	}
	return textResult(sb.String()), nil
}

func (s *serverCtx) handleAssetAdd(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	srcPath := p.require("source_path")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	filename, err := service.AssetAdd(ctx, dir, srcPath)
	if err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Added assets/%s", filename)), nil
}

func (s *serverCtx) handleAssetList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	infos, err := service.AssetList(ctx, dir)
	if err != nil {
		return errResult("%v", err)
	}

	if len(infos) == 0 {
		return textResult("No assets."), nil
	}

	var sb strings.Builder
	for _, info := range infos {
		fmt.Fprintf(&sb, "%s (%d bytes)\n", info.Name, info.Size)
	}
	return textResult(sb.String()), nil
}

func (s *serverCtx) handleAssetRemove(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	filename := p.require("filename")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.AssetDelete(ctx, dir, filename); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Removed assets/%s", filename)), nil
}

func (s *serverCtx) handleTodoArchiveList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	listName := p.optional("list", "")
	restore := p.optionalBool("restore", false)

	if listName == "" {
		archived, err := service.AutoArchiveDone(ctx, dir)
		if err != nil {
			return errResult("%v", err)
		}
		if len(archived) == 0 {
			return textResult("No fully completed lists to archive."), nil
		}
		return textResult(fmt.Sprintf("Auto-archived %d list(s): %s", len(archived), strings.Join(archived, ", "))), nil
	}

	if err := service.ArchiveList(ctx, dir, listName, restore); err != nil {
		return errResult("%v", err)
	}

	action := "Archived"
	if restore {
		action = "Restored"
	}
	return textResult(fmt.Sprintf("%s todo list %q", action, listName)), nil
}

func (s *serverCtx) handleTodoTag(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	listName := p.require("list")
	itemID := p.require("item_id")
	tagsStr := p.require("tags")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	remove := p.optionalBool("remove", false)

	var tags []string
	for _, t := range strings.Split(tagsStr, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}

	resultTags, err := service.SetItemTags(ctx, dir, listName, itemID, tags, remove)
	if err != nil {
		return errResult("%v", err)
	}

	action := "Added"
	if remove {
		action = "Removed"
	}
	return textResult(fmt.Sprintf("%s tags on %s #%s. Current tags: %s", action, listName, itemID, strings.Join(resultTags, ", "))), nil
}

func (s *serverCtx) handleKnowledgeArchive(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	filename := p.require("filename")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	restore := p.optionalBool("restore", false)

	if !restore {
		if refs := service.KnowledgeArchiveRefs(dir, filename); len(refs) > 0 {
			// Warn but proceed
			_ = refs
		}
	}

	if err := service.KnowledgeArchive(ctx, dir, filename, restore); err != nil {
		return errResult("%v", err)
	}

	action := "Archived"
	if restore {
		action = "Restored"
	}
	return textResult(fmt.Sprintf("%s knowledge/%s.md", action, filename)), nil
}

func (s *serverCtx) handleProjectRename(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	oldName := p.require("old_name")
	newName := p.require("new_name")
	if r := p.error(); r != nil {
		return r, nil
	}

	if err := service.ProjectRename(ctx, s.projectRoot, oldName, newName); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Renamed project %q to %q", oldName, newName)), nil
}
