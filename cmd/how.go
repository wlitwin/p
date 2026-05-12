package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
)

var howCmd = &cobra.Command{
	Use:   "how <question...>",
	Short: "Ask the AI how to do something with p",
	Long: `Ask a natural language question about how to use p and get
a helpful answer with the right commands.

Examples:
  p how do I move items between lists
  p how to set up a new project with a code repo
  p how can I search for todos tagged as bugs`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		question := strings.Join(args, " ")

		claudePath := cfg.ClaudePath
		if claudePath == "" {
			claudePath = "claude"
		}
		model := cfg.ClaudeModel
		if model == "" {
			model = "claude-opus-4-6"
		}

		helpText := buildHelpPrompt()

		claudeArgs := []string{
			"--print",
			"--system-prompt", helpText,
			"--no-session-persistence",
			"--model", model,
			"-p", question,
		}

		claudeCmd := exec.Command(claudePath, claudeArgs...)
		claudeCmd.Stderr = claudeStderr()

		out, err := claudeCmd.Output()
		if err != nil {
			return err
		}

		rendered, renderErr := ai.RenderMarkdown(string(out))
		if renderErr != nil {
			fmt.Print(string(out))
		} else {
			fmt.Print(rendered)
		}
		return nil
	},
}

func buildHelpPrompt() string {
	var sb strings.Builder

	sb.WriteString("You are a helpful assistant for the `p` CLI tool — a project knowledge & task manager.\n\n")
	sb.WriteString("Answer the user's question with specific `p` commands they should run. Be concise.\n\n")
	sb.WriteString("## Available commands\n\n")

	commands := []struct{ cmd, desc string }{
		// Top-level commands (daily use)
		{"p add <project> [list] '<text>' [--ai] [-k] [-l LIST] [--priority now|backlog] [--due YYYY-MM-DD]", "Add a todo item or knowledge entry"},
		{"p list", "List projects (shows path, created/updated dates)"},
		{"p list <project>", "List todo lists and knowledge docs in a project"},
		{"p list <project> <list|all> [--state open|blocked|done] [--priority now|backlog] [--tag TAG]", "List items with optional filters (use 'all' for all lists)"},
		{"p show <project> <list-or-doc> [-k]", "Show a todo list or knowledge document"},
		{"p status [project]", "Show open/blocked/done counts"},
		{"p done <project> <list> <id> [id...]", "Mark items done (supports multiple IDs)"},
		{"p search [project] <query>", "Full-text search across todos and knowledge"},
		{"p do <project> [list] [ids...] [-m 'message']", "Have AI implement todo items in the code repo"},
		{"p agent <project> <list> [-i MAX] [-m 'message']", "Autonomous agent loop — works through a todo list until done"},
		{"p plan <project> '<description>' [--also=other-project]", "Open-ended AI planning — creates multiple todos/knowledge"},
		{"p ask <project> '<question>' [-c]", "Ask the AI about project state (read-only). -c continues last conversation"},
		{"p save <project> [message...]", "Commit manual edits (Obsidian, text editor)"},
		{"p how <question>", "This command — ask how to do something"},
		{"p aliases [bash|zsh|fish]", "Print shell aliases (eval \"$(p aliases)\" in your profile)"},
		{"p init", "Set up p — configure project root directory"},
		{"p config [key] [value]", "View or set global config (project_root, claude_path, claude_model)"},

		// p project — project lifecycle
		{"p project new <project> [--description '']", "Create a new project"},
		{"p project rename <old-name> <new-name>", "Rename a project"},
		{"p project archive/unarchive <project>", "Archive or unarchive a project"},
		{"p project set <project> [key] [value]", "View or set project settings (description, code_dir)"},
		{"p project describe <project> <text...> [--auto]", "Set project description (--auto generates from contents)"},
		{"p project log <project> [-n COUNT] [--since DATE] [--until DATE] [--grep TEXT]", "Show git history with optional filters"},
		{"p project diff <project>", "Show uncommitted changes"},
		{"p project revert <project> [-y]", "Undo the last commit"},

		// p todo — item management
		{"p todo block <project> <list> <id> [id...]", "Mark items blocked"},
		{"p todo open <project> <list> <id> [id...]", "Reopen items"},
		{"p todo priority <project> <list> <id> now|backlog", "Set item priority"},
		{"p todo due <project> <list> <id> YYYY-MM-DD", "Set item due date"},
		{"p todo tag <project> <list> <id> <tags...> [--remove]", "Add or remove tags"},
		{"p todo move <project> <list> <id> <target-list>", "Move item to another list"},
		{"p todo rm-list <project> <list> [-y]", "Delete a todo list"},
		{"p todo archive-list <project> [list] [--restore]", "Archive a finished list (or auto-archive all 100% done)"},

		// p ai — specialized AI commands
		{"p ai review <project>", "AI reviews project and can update todos/knowledge"},
		{"p ai summarize <project>", "AI-generated status report (read-only)"},

		// p knowledge — docs
		{"p knowledge create <project> <name> <title> [--template decision-record|meeting-notes|runbook] [--tags a,b]", "Create a knowledge doc"},
		{"p knowledge delete <project> <doc> [-y]", "Delete a knowledge document"},
		{"p knowledge search <project> <query>", "Search knowledge documents"},
		{"p knowledge list <project> [--tag TAG]", "List knowledge docs with tags and sizes"},
		{"p knowledge archive <project> <doc>", "Archive a knowledge document"},

		// Internal / advanced
		{"p edit todo add/update/state/remove <project> <list> ...", "Deterministic todo edit primitives"},
		{"p edit knowledge create/append/replace/rename <project> ...", "Deterministic knowledge edit primitives"},
		{"p edit open <project> <name> [-k]", "Open a file in $EDITOR"},
		{"p mcp", "Run as MCP server (22 tools, used internally by AI commands)"},
	}

	for _, c := range commands {
		fmt.Fprintf(&sb, "- `%s` — %s\n", c.cmd, c.desc)
	}

	sb.WriteString("\n## Key concepts\n\n")
	sb.WriteString("- **Projects** are directories under the project root, each with its own git repo\n")
	sb.WriteString("- **Todo lists** are markdown files in `todos/`, containing checkbox items with inline metadata\n")
	sb.WriteString("- **Knowledge docs** are markdown files in `knowledge/`, organized wiki-style with [[links]]\n")
	sb.WriteString("- **Item IDs** are positional (1, 2, 2.1, 2.2) — shown in `p list` output\n")
	sb.WriteString("- **Tags** are inline metadata: `tags=bug,frontend`\n")
	sb.WriteString("- **Priorities**: `now` (default) or `backlog`\n")
	sb.WriteString("- **States**: `open`, `blocked`, `done`\n")
	sb.WriteString("- Every mutation auto-commits to git\n")
	sb.WriteString("- `code_dir` links a project to a code repository for `p do`\n")
	sb.WriteString("- **Custom AI prompts**: Create `.p/prompt.md` in a project for base AI instructions that apply to all AI commands. Optionally create `.p/prompt-do.md`, `.p/prompt-ask.md`, `.p/prompt-plan.md`, `.p/prompt-review.md`, `.p/prompt-summarize.md`, or `.p/prompt-add.md` for mode-specific instructions (appended to the base prompt).\n")
	sb.WriteString("- **Prompt templates**: To fully replace the default prompt (not just append), create `.p/template-{mode}.md` (e.g. `.p/template-ask.md`). Uses Go text/template syntax with variables: `{{.ProjectName}}`, `{{.ProjectDir}}`, `{{.ProjectDescription}}`, `{{.Mode}}`, `{{.Input}}`, `{{.ListName}}`, `{{.TodoLists}}`, `{{.TodoList}}`, `{{.KnowledgeDocs}}`, `{{.GitLog}}`, `{{.CustomPrompt}}`. Falls back to default prompt if template has errors.\n")

	return sb.String()
}

func init() {
	rootCmd.AddCommand(howCmd)
}
