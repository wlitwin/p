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

	desc := req.GetString("description", "")

	if err := service.ProjectCreate(s.projectRoot, name, desc); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Created project %q", name)), nil
}

func (s *serverCtx) handleProjectArchive(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	if proj == "" {
		return errResult("project is required")
	}

	archived := req.GetBool("archived", true)

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.ProjectArchive(dir, proj, archived); err != nil {
		return errResult("%v", err)
	}

	action := "archived"
	if !archived {
		action = "unarchived"
	}
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
		totalOpen, totalDone, totalBlocked := service.ProjectTotals(dir)
		fmt.Fprintf(&sb, "%s: open=%d blocked=%d done=%d\n", p.Name, totalOpen, totalBlocked, totalDone)
	}
	return textResult(sb.String()), nil
}

func (s *serverCtx) projectStatus(proj string) (*mcp.CallToolResult, error) {
	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Project: %s\n\n", proj)

	statuses, _ := service.GetProjectListStatuses(dir)
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

func (s *serverCtx) handleSearch(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	if query == "" {
		return errResult("query is required")
	}

	proj := req.GetString("project", "")
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

		matches := service.SearchProject(dir, name, queryLower)
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

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.SetItemPriority(dir, listName, itemID, todo.Priority(priority)); err != nil {
		return errResult("%v", err)
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

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.SetItemDue(dir, listName, itemID, due); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Set %s #%s due to %s", listName, itemID, due)), nil
}

func (s *serverCtx) handleTodoRmList(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	listName := req.GetString("list", "")

	if proj == "" || listName == "" {
		return errResult("project and list are required")
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.RemoveList(dir, listName); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Deleted todo list %q", listName)), nil
}

func (s *serverCtx) handleKnowledgeList(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	if proj == "" {
		return errResult("project is required")
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
