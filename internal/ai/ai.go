package ai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

type Task struct {
	ProjectName   string
	ProjectDir    string
	Input         string
	Mode          Mode
	CommandName   string   // CLI command name (e.g., "ask", "do", "review") for prompt file lookup
	ListName      string   // hint for todo mode, may be empty
	AlsoProjects  []string // additional project dirs for multi-project context
	AlsoNames     []string // names corresponding to AlsoProjects
}

type Mode string

const (
	ModeTodo      Mode = "todo"
	ModeKnowledge Mode = "knowledge"
	ModePlan      Mode = "plan"
	ModeAsk       Mode = "ask"
)

// LoadCustomPrompt reads custom prompt files from a project's .p/ directory.
// It returns the combined content of .p/prompt.md (base) and .p/prompt-{mode}.md
// (mode-specific), separated by newlines. Either or both may be absent.
func LoadCustomPrompt(projectDir string, mode string) string {
	var parts []string

	// Base prompt — always loaded if present
	basePath := filepath.Join(projectDir, ".p", "prompt.md")
	if data, err := os.ReadFile(basePath); err == nil {
		if content := strings.TrimSpace(string(data)); content != "" {
			parts = append(parts, content)
		}
	}

	// Mode-specific prompt — appended if present
	if mode != "" {
		modePath := filepath.Join(projectDir, ".p", fmt.Sprintf("prompt-%s.md", mode))
		if data, err := os.ReadFile(modePath); err == nil {
			if content := strings.TrimSpace(string(data)); content != "" {
				parts = append(parts, content)
			}
		}
	}

	return strings.Join(parts, "\n\n")
}

type MCPServerConfig struct {
	MCPServers map[string]MCPServerDef `json:"mcpServers"`
}

type MCPServerDef struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func MCPConfig(pBinary string) MCPServerConfig {
	return MCPServerConfig{
		MCPServers: map[string]MCPServerDef{
			"p": {
				Command: pBinary,
				Args:    []string{"mcp"},
			},
		},
	}
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

type RunOptions struct {
	Continue bool     // resume last conversation
	Stderr   *os.File // stderr for claude subprocess (nil to suppress)
}

func Run(pBinary, claudeBinary, model string, task Task, opts ...RunOptions) error {
	var opt RunOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	_ = opt
	prompt := buildPrompt(task)

	mcpCfg := MCPServerConfig{
		MCPServers: map[string]MCPServerDef{
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
		"--tools", "mcp,WebFetch,WebSearch",
		"--dangerously-skip-permissions",
		"--model", model,
		"--name", fmt.Sprintf("p-%s", task.ProjectName),
	}

	if opt.Continue {
		args = append(args, "--continue")
	}

	args = append(args, "-p", "Use the p MCP tools to complete the task described in the system prompt. Do not ask clarifying questions — make your best judgment.")

	cmd := exec.Command(claudeBinary, args...)
	cmd.Stderr = opt.Stderr

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
				if toolName, ok := strings.CutPrefix(block.Name, "mcp__p__"); ok {
					fmt.Fprintf(os.Stderr, "  → %s\n", toolName)
				}
			case "text":
				if text := strings.TrimSpace(block.Text); text != "" {
					rendered, err := RenderMarkdown(text)
					if err != nil {
						fmt.Fprintf(os.Stderr, "\n%s\n", text)
					} else {
						fmt.Fprint(os.Stderr, rendered)
					}
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

var mdRenderer *glamour.TermRenderer

func RenderMarkdown(text string) (string, error) {
	if mdRenderer == nil {
		var err error
		mdRenderer, err = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(100),
		)
		if err != nil {
			return "", err
		}
	}
	return mdRenderer.Render(text)
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

	for i, alsoDir := range task.AlsoProjects {
		name := task.AlsoNames[i]
		alsoTask := Task{ProjectName: name, ProjectDir: alsoDir}
		fmt.Fprintf(&sb, "## Related project: %s\n\n", name)
		sb.WriteString(projectContext(alsoTask))
	}

	// Inject custom system prompt from .p/prompt.md and .p/prompt-{mode}.md
	cmdName := task.CommandName
	if cmdName == "" {
		cmdName = string(task.Mode)
	}
	customPrompt := LoadCustomPrompt(task.ProjectDir, cmdName)
	if customPrompt != "" {
		sb.WriteString("## Custom instructions\n\n")
		sb.WriteString(customPrompt)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Task\n\n")

	switch task.Mode {
	case ModeTodo:
		sb.WriteString(todoInstructions(task))
	case ModeKnowledge:
		sb.WriteString(knowledgeInstructions(task))
	case ModePlan:
		sb.WriteString(planInstructions(task))
	case ModeAsk:
		sb.WriteString(askInstructions(task))
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

	// Include recent git history for context
	gitLog := recentGitLog(task.ProjectDir)
	if gitLog != "" {
		sb.WriteString("### Recent changes (git log)\n\n```\n")
		sb.WriteString(gitLog)
		sb.WriteString("```\n\n")
	}

	return sb.String()
}

func recentGitLog(dir string) string {
	cmd := exec.Command("git", "log", "--max-count=15", "--format=%h %cr %s")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
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

func askInstructions(task Task) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "The user has a question about this project:\n\n> %s\n\n", task.Input)

	sb.WriteString("This is a READ-ONLY query. Do NOT create, modify, or delete any todos or knowledge docs.\n\n")
	sb.WriteString("Guidelines:\n")
	sb.WriteString("- Use `project_list`, `todo_list`, and `knowledge_read` to explore the project.\n")
	sb.WriteString("- Answer the question based on the current project state.\n")
	sb.WriteString("- Be specific — reference actual todo items, lists, and knowledge docs by name.\n")
	sb.WriteString("- If the question is about progress, summarize what's done vs. open vs. blocked.\n")
	sb.WriteString("- If the question is about gaps or risks, analyze the current state critically.\n")
	sb.WriteString("- Keep your answer concise and actionable.\n")

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
