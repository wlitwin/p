package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/lock"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/service"
	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/validate"
)

func NewServer(projectRoot string) *server.MCPServer {
	s := server.NewMCPServer(
		"p",
		"0.1.0",
		server.WithToolCapabilities(true),
	)

	ctx := &serverCtx{projectRoot: projectRoot}

	// Read-only tools
	s.AddTool(projectListTool(), ctx.handleProjectList)
	s.AddTool(statusTool(), ctx.handleStatus)
	s.AddTool(searchTool(), ctx.handleSearch)
	s.AddTool(todoListTool(), ctx.handleTodoList)
	s.AddTool(knowledgeReadTool(), ctx.handleKnowledgeRead)
	s.AddTool(knowledgeListTool(), ctx.handleKnowledgeList)
	s.AddTool(knowledgeSearchTool(), ctx.handleKnowledgeSearch)

	// Project mutation tools
	s.AddTool(projectCreateTool(), ctx.handleProjectCreate)
	s.AddTool(projectArchiveTool(), ctx.locked(ctx.handleProjectArchive))

	// Todo mutation tools
	s.AddTool(todoAddTool(), ctx.locked(ctx.handleTodoAdd))
	s.AddTool(todoUpdateTool(), ctx.locked(ctx.handleTodoUpdate))
	s.AddTool(todoStateTool(), ctx.locked(ctx.handleTodoState))
	s.AddTool(todoPriorityTool(), ctx.locked(ctx.handleTodoPriority))
	s.AddTool(todoDueTool(), ctx.locked(ctx.handleTodoDue))
	s.AddTool(todoRemoveTool(), ctx.locked(ctx.handleTodoRemove))
	s.AddTool(todoMoveTool(), ctx.locked(ctx.handleTodoMove))
	s.AddTool(todoRmListTool(), ctx.locked(ctx.handleTodoRmList))
	s.AddTool(todoContextTool(), ctx.locked(ctx.handleTodoContext))

	// Knowledge mutation tools
	s.AddTool(knowledgeCreateTool(), ctx.locked(ctx.handleKnowledgeCreate))
	s.AddTool(knowledgeAppendTool(), ctx.locked(ctx.handleKnowledgeAppend))
	s.AddTool(knowledgeReplaceTool(), ctx.locked(ctx.handleKnowledgeReplace))
	s.AddTool(knowledgeRenameTool(), ctx.locked(ctx.handleKnowledgeRename))
	s.AddTool(knowledgeDeleteTool(), ctx.locked(ctx.handleKnowledgeDelete))

	return s
}

type serverCtx struct {
	projectRoot string
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: text},
		},
	}
}

func errResult(format string, args ...any) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(fmt.Sprintf(format, args...)), nil
}

type toolHandler = func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)

func (s *serverCtx) locked(handler toolHandler) toolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		proj := req.GetString("project", "")
		if proj == "" {
			return errResult("project is required")
		}

		dir, err := project.Resolve(s.projectRoot, proj)
		if err != nil {
			return errResult("%v", err)
		}

		lk, err := lock.Acquire(dir)
		if err != nil {
			return errResult("%v", err)
		}
		defer lk.Release()

		return handler(ctx, req)
	}
}

// resolve resolves a project name to its directory, returning an errResult on failure.
func (s *serverCtx) resolve(proj string) (string, *mcp.CallToolResult, error) {
	if proj == "" {
		r, err := errResult("project is required")
		return "", r, err
	}
	dir, err := project.Resolve(s.projectRoot, proj)
	if err != nil {
		r, e := errResult("%v", err)
		return "", r, e
	}
	return dir, nil, nil
}

// --- Tool definitions --- Project tools ---

func projectListTool() mcp.Tool {
	return mcp.NewTool("project_list",
		mcp.WithDescription("List all projects. Returns project names, descriptions, and archived status."),
	)
}

func todoListTool() mcp.Tool {
	return mcp.NewTool("todo_list",
		mcp.WithDescription("List todo lists in a project, or items in a specific list."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("list", mcp.Description("Todo list name. If omitted, lists all todo lists in the project.")),
	)
}

func knowledgeReadTool() mcp.Tool {
	return mcp.NewTool("knowledge_read",
		mcp.WithDescription("Read knowledge documents. If filename is omitted, lists all knowledge files."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("filename", mcp.Description("Knowledge doc filename (without .md). If omitted, lists all docs.")),
	)
}

func todoAddTool() mcp.Tool {
	return mcp.NewTool("todo_add",
		mcp.WithDescription("Add a todo item to a list. Creates the list if it doesn't exist."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("list", mcp.Description("Todo list name"), mcp.Required()),
		mcp.WithString("text", mcp.Description("Todo item text"), mcp.Required()),
		mcp.WithString("priority", mcp.Description("Priority: now or backlog"), mcp.DefaultString("now")),
		mcp.WithString("due", mcp.Description("Due date in YYYY-MM-DD format")),
		mcp.WithString("parent_id", mcp.Description("Parent item ID to nest under (e.g. '1' or '2.1')")),
	)
}

func todoUpdateTool() mcp.Tool {
	return mcp.NewTool("todo_update",
		mcp.WithDescription("Update the text of a todo item."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("list", mcp.Description("Todo list name"), mcp.Required()),
		mcp.WithString("item_id", mcp.Description("Item ID (e.g. '1' or '2.1')"), mcp.Required()),
		mcp.WithString("text", mcp.Description("New text for the item"), mcp.Required()),
	)
}

func todoStateTool() mcp.Tool {
	return mcp.NewTool("todo_state",
		mcp.WithDescription("Change the state of a todo item."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("list", mcp.Description("Todo list name"), mcp.Required()),
		mcp.WithString("item_id", mcp.Description("Item ID (e.g. '1' or '2.1')"), mcp.Required()),
		mcp.WithString("state", mcp.Description("New state: open, blocked, or done"), mcp.Required()),
	)
}

func todoRemoveTool() mcp.Tool {
	return mcp.NewTool("todo_remove",
		mcp.WithDescription("Remove a todo item from a list."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("list", mcp.Description("Todo list name"), mcp.Required()),
		mcp.WithString("item_id", mcp.Description("Item ID (e.g. '1' or '2.1')"), mcp.Required()),
	)
}

func knowledgeCreateTool() mcp.Tool {
	return mcp.NewTool("knowledge_create",
		mcp.WithDescription("Create a new knowledge document."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("filename", mcp.Description("Filename (without .md)"), mcp.Required()),
		mcp.WithString("title", mcp.Description("Document title"), mcp.Required()),
		mcp.WithString("tags", mcp.Description("Comma-separated tags")),
	)
}

func knowledgeAppendTool() mcp.Tool {
	return mcp.NewTool("knowledge_append",
		mcp.WithDescription("Append content to a knowledge document, optionally under a specific section."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("filename", mcp.Description("Knowledge doc filename (without .md)"), mcp.Required()),
		mcp.WithString("content", mcp.Description("Content to append (markdown)"), mcp.Required()),
		mcp.WithString("section", mcp.Description("Section heading to append under. If omitted, appends to end.")),
	)
}

func knowledgeReplaceTool() mcp.Tool {
	return mcp.NewTool("knowledge_replace",
		mcp.WithDescription("Replace the content of a section in a knowledge document."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("filename", mcp.Description("Knowledge doc filename (without .md)"), mcp.Required()),
		mcp.WithString("section", mcp.Description("Section heading to replace"), mcp.Required()),
		mcp.WithString("content", mcp.Description("New content for the section (markdown)"), mcp.Required()),
	)
}

func knowledgeRenameTool() mcp.Tool {
	return mcp.NewTool("knowledge_rename",
		mcp.WithDescription("Rename a knowledge document."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("old_filename", mcp.Description("Current filename (without .md)"), mcp.Required()),
		mcp.WithString("new_filename", mcp.Description("New filename (without .md)"), mcp.Required()),
	)
}

func todoMoveTool() mcp.Tool {
	return mcp.NewTool("todo_move",
		mcp.WithDescription("Move a todo item from one list to another."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("list", mcp.Description("Source todo list name"), mcp.Required()),
		mcp.WithString("item_id", mcp.Description("Item ID to move (e.g. '1' or '2.1')"), mcp.Required()),
		mcp.WithString("target_list", mcp.Description("Destination todo list name"), mcp.Required()),
	)
}

func knowledgeDeleteTool() mcp.Tool {
	return mcp.NewTool("knowledge_delete",
		mcp.WithDescription("Delete a knowledge document."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("filename", mcp.Description("Knowledge doc filename (without .md)"), mcp.Required()),
	)
}

func todoContextTool() mcp.Tool {
	return mcp.NewTool("todo_context",
		mcp.WithDescription("Set or clear context patterns on a todo list. Context patterns control which knowledge docs are included in AI prompts for this list. Pass patterns to set, omit patterns with clear=true to remove."),
		mcp.WithString("project", mcp.Description("Project name"), mcp.Required()),
		mcp.WithString("list", mcp.Description("Todo list name"), mcp.Required()),
		mcp.WithString("patterns", mcp.Description("Comma-separated glob patterns (e.g. 'architecture/*,decisions/db-*'). Omit to clear.")),
		mcp.WithBoolean("clear", mcp.Description("If true, removes the context field (reverts to project default or all)")),
	)
}

// --- Handlers ---

func (s *serverCtx) handleProjectList(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	projects, err := project.List(s.projectRoot, true)
	if err != nil {
		return errResult("listing projects: %v", err)
	}

	var sb strings.Builder
	for _, p := range projects {
		status := ""
		if p.Archived {
			status = " (archived)"
		}
		fmt.Fprintf(&sb, "- %s%s", p.Name, status)
		if p.Description != "" {
			fmt.Fprintf(&sb, " — %s", p.Description)
		}
		sb.WriteString("\n")
	}
	if sb.Len() == 0 {
		return textResult("No projects found."), nil
	}
	return textResult(sb.String()), nil
}

func (s *serverCtx) handleTodoList(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	listName := req.GetString("list", "")
	if listName == "" {
		names, err := todo.ListNames(dir)
		if err != nil {
			return errResult("listing todos: %v", err)
		}
		if len(names) == 0 {
			return textResult("No todo lists in this project."), nil
		}
		return textResult("Todo lists:\n- " + strings.Join(names, "\n- ")), nil
	}

	list, err := todo.LoadList(dir, listName)
	if err != nil {
		return errResult("loading list: %v", err)
	}
	return textResult(todo.Render(list)), nil
}

func (s *serverCtx) handleKnowledgeRead(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	filename := req.GetString("filename", "")
	if filename == "" {
		files, err := knowledge.ListFiles(dir)
		if err != nil {
			return errResult("listing knowledge: %v", err)
		}
		if len(files) == 0 {
			return textResult("No knowledge documents in this project."), nil
		}
		return textResult("Knowledge docs:\n- " + strings.Join(files, "\n- ")), nil
	}

	content, err := knowledge.Read(dir, filename)
	if err != nil {
		return errResult("reading %s: %v", filename, err)
	}
	return textResult(content), nil
}

func (s *serverCtx) handleTodoAdd(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	listName := req.GetString("list", "")
	text := req.GetString("text", "")

	if proj == "" || listName == "" || text == "" {
		return errResult("project, list, and text are required")
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	priority := todo.Priority(req.GetString("priority", "now"))
	due := req.GetString("due", "")
	parentID := req.GetString("parent_id", "")

	if err := service.AddItem(dir, listName, text, priority, due, parentID); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Added todo: %s", text)), nil
}

func (s *serverCtx) handleTodoUpdate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	listName := req.GetString("list", "")
	itemID := req.GetString("item_id", "")
	text := req.GetString("text", "")

	if proj == "" || listName == "" || itemID == "" || text == "" {
		return errResult("project, list, item_id, and text are required")
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.UpdateItemText(dir, listName, itemID, text); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Updated %s #%s", listName, itemID)), nil
}

func (s *serverCtx) handleTodoState(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	listName := req.GetString("list", "")
	itemID := req.GetString("item_id", "")
	state := req.GetString("state", "")

	if proj == "" || listName == "" || itemID == "" || state == "" {
		return errResult("project, list, item_id, and state are required")
	}

	if err := validate.State(state); err != nil {
		return errResult("%v", err)
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.SetItemState(dir, listName, itemID, todo.State(state)); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Set %s #%s to %s", listName, itemID, state)), nil
}

func (s *serverCtx) handleTodoRemove(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	listName := req.GetString("list", "")
	itemID := req.GetString("item_id", "")

	if proj == "" || listName == "" || itemID == "" {
		return errResult("project, list, and item_id are required")
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.RemoveItem(dir, listName, itemID); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Removed %s #%s", listName, itemID)), nil
}

func (s *serverCtx) handleKnowledgeCreate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	filename := req.GetString("filename", "")
	title := req.GetString("title", "")

	if proj == "" || filename == "" || title == "" {
		return errResult("project, filename, and title are required")
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	var tags []string
	if t := req.GetString("tags", ""); t != "" {
		tags = strings.Split(t, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	if err := service.KnowledgeCreate(dir, filename, title, tags); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Created knowledge/%s.md", filename)), nil
}

func (s *serverCtx) handleKnowledgeAppend(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	filename := req.GetString("filename", "")
	content := req.GetString("content", "")

	if proj == "" || filename == "" || content == "" {
		return errResult("project, filename, and content are required")
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	section := req.GetString("section", "")

	if err := service.KnowledgeAppend(dir, filename, content, section); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Appended to knowledge/%s.md", filename)), nil
}

func (s *serverCtx) handleKnowledgeReplace(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	filename := req.GetString("filename", "")
	section := req.GetString("section", "")
	content := req.GetString("content", "")

	if proj == "" || filename == "" || section == "" || content == "" {
		return errResult("project, filename, section, and content are required")
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.KnowledgeReplace(dir, filename, section, content); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Replaced section %q in knowledge/%s.md", section, filename)), nil
}

func (s *serverCtx) handleKnowledgeRename(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	oldName := req.GetString("old_filename", "")
	newName := req.GetString("new_filename", "")

	if proj == "" || oldName == "" || newName == "" {
		return errResult("project, old_filename, and new_filename are required")
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.KnowledgeRename(dir, oldName, newName); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Renamed knowledge/%s.md to knowledge/%s.md", oldName, newName)), nil
}

func (s *serverCtx) handleTodoMove(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	listName := req.GetString("list", "")
	itemID := req.GetString("item_id", "")
	targetList := req.GetString("target_list", "")

	if proj == "" || listName == "" || itemID == "" || targetList == "" {
		return errResult("project, list, item_id, and target_list are required")
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.MoveItem(dir, listName, itemID, targetList); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Moved %s #%s to %s", listName, itemID, targetList)), nil
}

func (s *serverCtx) handleKnowledgeDelete(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	filename := req.GetString("filename", "")

	if proj == "" || filename == "" {
		return errResult("project and filename are required")
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.KnowledgeDelete(dir, filename); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Deleted knowledge/%s.md", filename)), nil
}

func (s *serverCtx) handleTodoContext(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	proj := req.GetString("project", "")
	listName := req.GetString("list", "")

	if proj == "" || listName == "" {
		return errResult("project and list are required")
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	clearFlag := req.GetBool("clear", false)

	if clearFlag {
		if err := service.SetListContext(dir, listName, nil); err != nil {
			return errResult("%v", err)
		}
		return textResult(fmt.Sprintf("Cleared context on %s (will use project default or all docs)", listName)), nil
	}

	patternsStr := req.GetString("patterns", "")
	if patternsStr == "" {
		return errResult("either patterns or clear=true is required")
	}

	var patterns []string
	for _, p := range strings.Split(patternsStr, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			patterns = append(patterns, p)
		}
	}

	if err := service.SetListContext(dir, listName, patterns); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Set context on %s: %s", listName, strings.Join(patterns, ", "))), nil
}
