// Package mcpserver implements an MCP (Model Context Protocol) stdio server
// that exposes project management tools for AI agents.
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

// NewServer creates and configures an MCP server with all p tool handlers
// registered. The projectRoot is used to resolve project directories.
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

	// Asset mutation tools
	s.AddTool(assetAddTool(), ctx.locked(ctx.handleAssetAdd))
	s.AddTool(assetListTool(), ctx.handleAssetList)
	s.AddTool(assetRemoveTool(), ctx.locked(ctx.handleAssetRemove))

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
		mcp.WithString("list", mcp.Description("Todo list name (supports subdirectory paths like 'sprint/week-1'). If omitted, lists all todo lists in the project.")),
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
		mcp.WithString("list", mcp.Description("Todo list name (supports subdirectory paths like 'sprint/week-1')"), mcp.Required()),
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
		mcp.WithString("list", mcp.Description("Source todo list name (supports subdirectory paths like 'sprint/week-1')"), mcp.Required()),
		mcp.WithString("item_id", mcp.Description("Item ID to move (e.g. '1' or '2.1')"), mcp.Required()),
		mcp.WithString("target_list", mcp.Description("Destination todo list name (supports subdirectory paths)"), mcp.Required()),
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

func (s *serverCtx) handleProjectList(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func (s *serverCtx) handleTodoList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	if r := p.error(); r != nil {
		return r, nil
	}

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

func (s *serverCtx) handleKnowledgeRead(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	if r := p.error(); r != nil {
		return r, nil
	}

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

func (s *serverCtx) handleTodoAdd(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	listName := p.require("list")
	text := p.require("text")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	priority := todo.Priority(p.optional("priority", "now"))
	due := p.optional("due", "")
	parentID := p.optional("parent_id", "")

	if err := service.AddItem(ctx, dir, listName, text, priority, due, parentID); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Added todo: %s", text)), nil
}

func (s *serverCtx) handleTodoUpdate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	listName := p.require("list")
	itemID := p.require("item_id")
	text := p.require("text")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.UpdateItemText(ctx, dir, listName, itemID, text); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Updated %s #%s", listName, itemID)), nil
}

func (s *serverCtx) handleTodoState(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	listName := p.require("list")
	itemID := p.require("item_id")
	state := p.require("state")
	if r := p.error(); r != nil {
		return r, nil
	}

	if err := validate.State(state); err != nil {
		return errResult("%v", err)
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.SetItemState(ctx, dir, listName, itemID, todo.State(state)); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Set %s #%s to %s", listName, itemID, state)), nil
}

func (s *serverCtx) handleTodoRemove(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	listName := p.require("list")
	itemID := p.require("item_id")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.RemoveItem(ctx, dir, listName, itemID); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Removed %s #%s", listName, itemID)), nil
}

func (s *serverCtx) handleKnowledgeCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	filename := p.require("filename")
	title := p.require("title")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	var tags []string
	if t := p.optional("tags", ""); t != "" {
		tags = strings.Split(t, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	if err := service.KnowledgeCreate(ctx, dir, filename, title, tags); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Created knowledge/%s.md", filename)), nil
}

func (s *serverCtx) handleKnowledgeAppend(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	filename := p.require("filename")
	content := p.require("content")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	section := p.optional("section", "")

	if err := service.KnowledgeAppend(ctx, dir, filename, content, section); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Appended to knowledge/%s.md", filename)), nil
}

func (s *serverCtx) handleKnowledgeReplace(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	filename := p.require("filename")
	section := p.require("section")
	content := p.require("content")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.KnowledgeReplace(ctx, dir, filename, section, content); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Replaced section %q in knowledge/%s.md", section, filename)), nil
}

func (s *serverCtx) handleKnowledgeRename(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	oldName := p.require("old_filename")
	newName := p.require("new_filename")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.KnowledgeRename(ctx, dir, oldName, newName); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Renamed knowledge/%s.md to knowledge/%s.md", oldName, newName)), nil
}

func (s *serverCtx) handleTodoMove(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p := newParams(req)
	proj := p.require("project")
	listName := p.require("list")
	itemID := p.require("item_id")
	targetList := p.require("target_list")
	if r := p.error(); r != nil {
		return r, nil
	}

	dir, r, err := s.resolve(proj)
	if r != nil {
		return r, err
	}

	if err := service.MoveItem(ctx, dir, listName, itemID, targetList); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Moved %s #%s to %s", listName, itemID, targetList)), nil
}

func (s *serverCtx) handleKnowledgeDelete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	// Check for referencing lists before deleting
	refs := knowledge.FindReferencingLists(dir, filename)

	if err := service.KnowledgeDelete(ctx, dir, filename); err != nil {
		return errResult("%v", err)
	}

	msg := fmt.Sprintf("Deleted knowledge/%s.md", filename)
	if len(refs) > 0 {
		msg += fmt.Sprintf("\n⚠ Warning: this doc was referenced by context patterns in: %s", strings.Join(refs, ", "))
	}
	return textResult(msg), nil
}

func (s *serverCtx) handleTodoContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	clearFlag := p.optionalBool("clear", false)

	if clearFlag {
		if err := service.SetListContext(ctx, dir, listName, nil); err != nil {
			return errResult("%v", err)
		}
		return textResult(fmt.Sprintf("Cleared context on %s (will use project default or all docs)", listName)), nil
	}

	patternsStr := p.optional("patterns", "")
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

	if err := service.SetListContext(ctx, dir, listName, patterns); err != nil {
		return errResult("%v", err)
	}

	return textResult(fmt.Sprintf("Set context on %s: %s", listName, strings.Join(patterns, ", "))), nil
}
