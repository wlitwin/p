package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

type Task struct {
	ProjectName string
	ProjectDir  string
	Input       string
	Mode        Mode
	ListName    string // hint for todo mode, may be empty
}

type Mode string

const (
	ModeTodo      Mode = "todo"
	ModeKnowledge Mode = "knowledge"
)

type mcpConfig struct {
	MCPServers map[string]mcpServerDef `json:"mcpServers"`
}

type mcpServerDef struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func Run(pBinary, claudeBinary string, task Task) error {
	prompt := buildPrompt(task)

	mcpCfg := mcpConfig{
		MCPServers: map[string]mcpServerDef{
			"p": {
				Command: pBinary,
				Args:    []string{"mcp"},
			},
		},
	}
	mcpJSON, err := json.Marshal(mcpCfg)
	if err != nil {
		return fmt.Errorf("marshaling MCP config: %w", err)
	}

	args := []string{
		"--print",
		"--system-prompt", prompt,
		"--mcp-config", string(mcpJSON),
		"--strict-mcp-config",
		"--tools", "mcp",
		"--no-session-persistence",
		"-p", "Use the p MCP tools to complete the task described in the system prompt. Do not ask clarifying questions — make your best judgment.",
	}

	cmd := exec.Command(claudeBinary, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	fmt.Fprintf(os.Stderr, "Running AI agent...\n")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude subprocess failed: %w", err)
	}

	return nil
}

func buildPrompt(task Task) string {
	var sb strings.Builder

	sb.WriteString("You are a project knowledge manager for the project \"")
	sb.WriteString(task.ProjectName)
	sb.WriteString("\".\n\n")

	sb.WriteString("## Your tools\n\n")
	sb.WriteString("You have MCP tools to manage todos and knowledge docs. ")
	sb.WriteString("Always use the project name \"")
	sb.WriteString(task.ProjectName)
	sb.WriteString("\" when calling tools.\n\n")

	// Add project context
	sb.WriteString("## Current project state\n\n")
	sb.WriteString(projectContext(task))

	// Add task-specific instructions
	sb.WriteString("## Task\n\n")

	switch task.Mode {
	case ModeTodo:
		sb.WriteString(todoInstructions(task))
	case ModeKnowledge:
		sb.WriteString(knowledgeInstructions(task))
	}

	return sb.String()
}

func projectContext(task Task) string {
	var sb strings.Builder

	// List todo lists
	names, err := todo.ListNames(task.ProjectDir)
	if err == nil && len(names) > 0 {
		sb.WriteString("### Todo lists\n\n")
		for _, name := range names {
			list, err := todo.LoadList(task.ProjectDir, name)
			if err != nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("**%s**:\n", name))
			sb.WriteString(todo.Render(list))
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("### Todo lists\n\nNo todo lists exist yet.\n\n")
	}

	// List knowledge docs
	files, err := knowledge.ListFiles(task.ProjectDir)
	if err == nil && len(files) > 0 {
		sb.WriteString("### Knowledge docs\n\n")
		for _, f := range files {
			content, err := knowledge.Read(task.ProjectDir, f)
			if err != nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("**%s.md**:\n```\n%s\n```\n\n", f, content))
		}
	} else {
		sb.WriteString("### Knowledge docs\n\nNo knowledge docs exist yet.\n\n")
	}

	// Project metadata
	meta, err := project.LoadMeta(task.ProjectDir)
	if err == nil && meta.Description != "" {
		sb.WriteString(fmt.Sprintf("### Project description\n\n%s\n\n", meta.Description))
	}

	return sb.String()
}

func todoInstructions(task Task) string {
	var sb strings.Builder

	sb.WriteString("Add the following as a todo item:\n\n")
	sb.WriteString(fmt.Sprintf("> %s\n\n", task.Input))

	sb.WriteString("Guidelines:\n")
	sb.WriteString("- Use the `todo_add` tool to add the item.\n")
	if task.ListName != "" {
		sb.WriteString(fmt.Sprintf("- Add it to the list \"%s\".\n", task.ListName))
	} else {
		sb.WriteString("- Choose the most appropriate existing list, or create a new one if none fit.\n")
	}
	sb.WriteString("- If it makes sense as a sub-item of an existing todo, nest it using parent_id.\n")
	sb.WriteString("- Word the todo clearly and concisely as an actionable task.\n")
	sb.WriteString("- Set priority to 'now' unless it sounds like a future/low-priority idea.\n")
	sb.WriteString("- If the input references related knowledge, add a [[wiki link]] in the todo text.\n")
	sb.WriteString("- Do NOT modify existing todos unless the new item is clearly a sub-task of one.\n")

	return sb.String()
}

func knowledgeInstructions(task Task) string {
	var sb strings.Builder

	sb.WriteString("Add the following information to the project knowledge base:\n\n")
	sb.WriteString(fmt.Sprintf("> %s\n\n", task.Input))

	sb.WriteString("Guidelines:\n")
	sb.WriteString("- If the input is a URL, fetch and summarize the content at that URL.\n")
	sb.WriteString("- Decide where this information belongs in the knowledge base.\n")
	sb.WriteString("- If an existing knowledge doc covers this topic, append to it (use `knowledge_append`).\n")
	sb.WriteString("- If no existing doc fits, create a new one (use `knowledge_create` then `knowledge_append`).\n")
	sb.WriteString("- Write clear, concise markdown. Use headings, lists, and links as appropriate.\n")
	sb.WriteString("- Use [[wiki links]] to cross-reference other knowledge docs.\n")
	sb.WriteString("- Preserve existing content — append or update sections, don't overwrite unrelated content.\n")
	sb.WriteString("- If the information relates to a decision, add it under a 'Decisions' section.\n")

	return sb.String()
}
