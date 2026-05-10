package ai

import (
	"bufio"
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
	ModePlan      Mode = "plan"
)

type mcpConfig struct {
	MCPServers map[string]mcpServerDef `json:"mcpServers"`
}

type mcpServerDef struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// stream-json event types we care about
type streamEvent struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
	Tool    string          `json:"tool,omitempty"`
}

type contentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`
}

type assistantMessage struct {
	Content []contentBlock `json:"content"`
}

func Run(pBinary, claudeBinary, model string, task Task) error {
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
		"--verbose",
		"--output-format", "stream-json",
		"--system-prompt", prompt,
		"--mcp-config", string(mcpJSON),
		"--strict-mcp-config",
		"--tools", "mcp",
		"--no-session-persistence",
		"--dangerously-skip-permissions",
		"--model", model,
		"-p", "Use the p MCP tools to complete the task described in the system prompt. Do not ask clarifying questions — make your best judgment.",
	}

	cmd := exec.Command(claudeBinary, args...)
	cmd.Stderr = nil // suppress claude's stderr warnings

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating stdout pipe: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Running AI agent...\n")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting claude: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		processStreamLine(line)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("claude subprocess failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "AI agent finished.\n")
	return nil
}

func processStreamLine(line string) {
	var event streamEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return
	}

	switch event.Type {
	case "assistant":
		var msg assistantMessage
		if err := json.Unmarshal([]byte(line), &struct {
			Message *assistantMessage `json:"message"`
		}{&msg}); err != nil {
			return
		}
		for _, block := range msg.Content {
			switch block.Type {
			case "tool_use":
				if strings.HasPrefix(block.Name, "mcp__p__") {
					toolName := strings.TrimPrefix(block.Name, "mcp__p__")
					fmt.Fprintf(os.Stderr, "  → %s\n", toolName)
				}
			case "text":
				if text := strings.TrimSpace(block.Text); text != "" {
					fmt.Fprintf(os.Stderr, "\n%s\n", text)
				}
			}
		}

	case "result":
		var result struct {
			Subtype string `json:"subtype"`
		}
		if err := json.Unmarshal([]byte(line), &result); err == nil {
			if result.Subtype == "error_max_turns" {
				fmt.Fprintf(os.Stderr, "  ⚠ AI hit max turns limit\n")
			}
		}
	}
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

	sb.WriteString("## Current project state\n\n")
	sb.WriteString(projectContext(task))

	sb.WriteString("## Task\n\n")

	switch task.Mode {
	case ModeTodo:
		sb.WriteString(todoInstructions(task))
	case ModeKnowledge:
		sb.WriteString(knowledgeInstructions(task))
	case ModePlan:
		sb.WriteString(planInstructions(task))
	}

	return sb.String()
}

func projectContext(task Task) string {
	var sb strings.Builder

	names, err := todo.ListNames(task.ProjectDir)
	if err == nil && len(names) > 0 {
		sb.WriteString("### Todo lists\n\n")
		for _, name := range names {
			list, err := todo.LoadList(task.ProjectDir, name)
			if err != nil {
				continue
			}
			fmt.Fprintf(&sb, "**%s**:\n", name)
			sb.WriteString(todo.Render(list))
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("### Todo lists\n\nNo todo lists exist yet.\n\n")
	}

	files, err := knowledge.ListFiles(task.ProjectDir)
	if err == nil && len(files) > 0 {
		sb.WriteString("### Knowledge docs\n\n")
		for _, f := range files {
			content, err := knowledge.Read(task.ProjectDir, f)
			if err != nil {
				continue
			}
			fmt.Fprintf(&sb, "**%s.md**:\n```\n%s\n```\n\n", f, content)
		}
	} else {
		sb.WriteString("### Knowledge docs\n\nNo knowledge docs exist yet.\n\n")
	}

	meta, err := project.LoadMeta(task.ProjectDir)
	if err == nil && meta.Description != "" {
		fmt.Fprintf(&sb, "### Project description\n\n%s\n\n", meta.Description)
	}

	return sb.String()
}

func todoInstructions(task Task) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Add the following as a todo item:\n\n> %s\n\n", task.Input)

	sb.WriteString("Guidelines:\n")
	sb.WriteString("- Use the `todo_add` tool to add the item.\n")
	if task.ListName != "" {
		fmt.Fprintf(&sb, "- Add it to the list \"%s\".\n", task.ListName)
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

func planInstructions(task Task) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "The user has given you this open-ended task:\n\n> %s\n\n", task.Input)

	sb.WriteString("You have full freedom to explore the project state and make multiple changes.\n\n")

	sb.WriteString("Guidelines:\n")
	sb.WriteString("- Start by reading the current project state using `project_list`, `todo_list`, and `knowledge_read`.\n")
	sb.WriteString("- You can create multiple todo items across multiple lists.\n")
	sb.WriteString("- You can create new todo lists if the work spans different topics.\n")
	sb.WriteString("- You can create or update knowledge docs to capture context, decisions, or plans.\n")
	sb.WriteString("- Group related todos into the same list. Use separate lists for distinct workstreams.\n")
	sb.WriteString("- Word each todo as a clear, actionable task.\n")
	sb.WriteString("- Set priority=backlog for nice-to-haves, priority=now for important items.\n")
	sb.WriteString("- Use [[wiki links]] to connect todos to relevant knowledge docs.\n")
	sb.WriteString("- If the task involves planning, consider creating a knowledge doc that captures the overall plan, then individual todos for execution.\n")
	sb.WriteString("- Think step by step about what needs to happen and break it down into concrete tasks.\n")

	return sb.String()
}

func knowledgeInstructions(task Task) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Add the following information to the project knowledge base:\n\n> %s\n\n", task.Input)

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
