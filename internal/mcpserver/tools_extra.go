package mcpserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
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
		mcp.WithString("list", mcp.Description("Todo list name to delete"), mcp.Required()),
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

// --- Handlers ---

func (s *serverCtx) handleProjectCreate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if name == "" {
		return errResult("name is required")
	}

	if err := validate.ProjectName(name); err != nil {
		return errResult("%v", err)
	}

	desc := req.GetString("description", "")

	if err := project.Create(s.projectRoot, name, desc); err != nil {
		return errResult("%v", err)
	}

	dir := filepath.Join(s.projectRoot, name)
	if err := git.Init(dir); err != nil {
		return errResult("git init: %v", err)
	}
	_ = git.CommitAll(dir, fmt.Sprintf("p: create project %q", name))

	return textResult(fmt.Sprintf("Created project %q", name)), nil
}

func (s *serverCtx) handleProjectArchive(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	if proj == "" {
		return errResult("project is required")
	}

	archived := req.GetBool("archived", true)

	dir, err := project.Resolve(s.projectRoot, proj)
	if err != nil {
		return errResult("%v", err)
	}

	meta, err := project.LoadMeta(dir)
	if err != nil {
		return errResult("%v", err)
	}

	meta.Archived = archived
	if err := project.SaveMeta(dir, meta); err != nil {
		return errResult("%v", err)
	}

	action := "archived"
	if !archived {
		action = "unarchived"
	}
	_ = git.CommitAll(dir, fmt.Sprintf("p: %s project %q", action, proj))

	return textResult(fmt.Sprintf("Project %q %s", proj, action)), nil
}

func (s *serverCtx) handleStatus(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")

	if proj != "" {
		return s.projectStatus(proj)
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
		totalOpen, totalBlocked, totalDone := 0, 0, 0
		names, _ := todo.ListNames(dir)
		for _, name := range names {
			list, err := todo.LoadList(dir, name)
			if err != nil {
				continue
			}
			o, d, b := countItems(list.Items)
			totalOpen += o
			totalDone += d
			totalBlocked += b
		}
		fmt.Fprintf(&sb, "%s: open=%d blocked=%d done=%d\n", p.Name, totalOpen, totalBlocked, totalDone)
	}
	return textResult(sb.String()), nil
}

func (s *serverCtx) projectStatus(proj string) (*mcp.CallToolResult, error) {
	dir, err := project.Resolve(s.projectRoot, proj)
	if err != nil {
		return errResult("%v", err)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Project: %s\n\n", proj)

	names, _ := todo.ListNames(dir)
	if len(names) == 0 {
		sb.WriteString("No todo lists.\n")
	} else {
		for _, name := range names {
			list, err := todo.LoadList(dir, name)
			if err != nil {
				continue
			}
			open, done, blocked := countItems(list.Items)
			fmt.Fprintf(&sb, "  %-20s open=%d blocked=%d done=%d\n", name, open, blocked, done)
		}
	}

	files, _ := knowledge.ListFiles(dir)
	if len(files) > 0 {
		sb.WriteString("\nKnowledge docs: " + strings.Join(files, ", ") + "\n")
	}

	return textResult(sb.String()), nil
}

func (s *serverCtx) handleSearch(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	if query == "" {
		return errResult("query is required")
	}

	proj := req.GetString("project", "")
	queryLower := strings.ToLower(query)

	var sb strings.Builder
	var projectNames []string

	if proj != "" {
		projectNames = []string{proj}
	} else {
		projects, _ := project.List(s.projectRoot, false)
		for _, p := range projects {
			projectNames = append(projectNames, p.Name)
		}
	}

	for _, name := range projectNames {
		dir, err := project.Resolve(s.projectRoot, name)
		if err != nil {
			continue
		}

		lists, _ := todo.ListNames(dir)
		for _, listName := range lists {
			list, err := todo.LoadList(dir, listName)
			if err != nil {
				continue
			}
			searchItemsMCP(&sb, list.Items, name, listName, "", 1, queryLower)
		}

		files, _ := knowledge.ListFiles(dir)
		for _, f := range files {
			content, err := knowledge.Read(dir, f)
			if err != nil {
				continue
			}
			if strings.Contains(strings.ToLower(content), queryLower) {
				fmt.Fprintf(&sb, "%s/knowledge/%s.md: matches query\n", name, f)
			}
		}
	}

	if sb.Len() == 0 {
		return textResult("No matches found."), nil
	}
	return textResult(sb.String()), nil
}

func searchItemsMCP(sb *strings.Builder, items []*todo.Item, projectName, listName, prefix string, start int, query string) {
	for i, item := range items {
		id := fmt.Sprintf("%s%d", prefix, start+i)
		if strings.Contains(strings.ToLower(item.Text), query) {
			marker := "[ ]"
			switch item.State {
			case todo.Done:
				marker = "[x]"
			case todo.Blocked:
				marker = "[-]"
			}
			fmt.Fprintf(sb, "%s/%s#%s %s %s\n", projectName, listName, id, marker, item.Text)
		}
		if len(item.Children) > 0 {
			searchItemsMCP(sb, item.Children, projectName, listName, id+".", 1, query)
		}
	}
}

func (s *serverCtx) handleTodoPriority(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	listName := req.GetString("list", "")
	itemID := req.GetString("item_id", "")
	priority := req.GetString("priority", "")

	if proj == "" || listName == "" || itemID == "" || priority == "" {
		return errResult("project, list, item_id, and priority are required")
	}

	if err := validate.Priority(priority); err != nil {
		return errResult("%v", err)
	}

	dir, err := project.Resolve(s.projectRoot, proj)
	if err != nil {
		return errResult("%v", err)
	}

	list, err := todo.LoadList(dir, listName)
	if err != nil {
		return errResult("loading list: %v", err)
	}

	item, err := todo.ResolveItem(list, itemID)
	if err != nil {
		return errResult("%v", err)
	}

	item.Priority = todo.Priority(priority)

	if err := todo.SaveList(dir, listName, list); err != nil {
		return errResult("saving: %v", err)
	}

	return textResult(fmt.Sprintf("Set %s #%s priority to %s", listName, itemID, priority)), nil
}

func (s *serverCtx) handleTodoDue(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	listName := req.GetString("list", "")
	itemID := req.GetString("item_id", "")
	due := req.GetString("due", "")

	if proj == "" || listName == "" || itemID == "" || due == "" {
		return errResult("project, list, item_id, and due are required")
	}

	dir, err := project.Resolve(s.projectRoot, proj)
	if err != nil {
		return errResult("%v", err)
	}

	list, err := todo.LoadList(dir, listName)
	if err != nil {
		return errResult("loading list: %v", err)
	}

	item, err := todo.ResolveItem(list, itemID)
	if err != nil {
		return errResult("%v", err)
	}

	item.Due = due

	if err := todo.SaveList(dir, listName, list); err != nil {
		return errResult("saving: %v", err)
	}

	return textResult(fmt.Sprintf("Set %s #%s due to %s", listName, itemID, due)), nil
}

func (s *serverCtx) handleTodoRmList(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	listName := req.GetString("list", "")

	if proj == "" || listName == "" {
		return errResult("project and list are required")
	}

	dir, err := project.Resolve(s.projectRoot, proj)
	if err != nil {
		return errResult("%v", err)
	}

	path := todo.ListPath(dir, listName)
	if _, err := os.Stat(path); err != nil {
		return errResult("todo list %q not found", listName)
	}

	if err := os.Remove(path); err != nil {
		return errResult("deleting: %v", err)
	}

	return textResult(fmt.Sprintf("Deleted todo list %q", listName)), nil
}

func (s *serverCtx) handleKnowledgeList(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	if proj == "" {
		return errResult("project is required")
	}

	dir, err := project.Resolve(s.projectRoot, proj)
	if err != nil {
		return errResult("%v", err)
	}

	files, err := knowledge.ListFiles(dir)
	if err != nil {
		return errResult("%v", err)
	}
	if len(files) == 0 {
		return textResult("No knowledge documents."), nil
	}

	tagFilter := req.GetString("tag", "")

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

func (s *serverCtx) handleKnowledgeSearch(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	query := req.GetString("query", "")

	if proj == "" || query == "" {
		return errResult("project and query are required")
	}

	dir, err := project.Resolve(s.projectRoot, proj)
	if err != nil {
		return errResult("%v", err)
	}

	files, err := knowledge.ListFiles(dir)
	if err != nil {
		return errResult("%v", err)
	}

	queryLower := strings.ToLower(query)
	var sb strings.Builder
	for _, f := range files {
		content, err := knowledge.Read(dir, f)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(content), queryLower) {
			fmt.Fprintf(&sb, "%s.md: matches\n", f)
		}
	}

	if sb.Len() == 0 {
		return textResult("No matches found."), nil
	}
	return textResult(sb.String()), nil
}

func countItems(items []*todo.Item) (open, done, blocked int) {
	for _, item := range items {
		switch item.State {
		case todo.Open:
			open++
		case todo.Done:
			done++
		case todo.Blocked:
			blocked++
		}
		co, cd, cb := countItems(item.Children)
		open += co
		done += cd
		blocked += cb
	}
	return
}
