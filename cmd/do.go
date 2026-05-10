package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/tui"
)

var doCmd = &cobra.Command{
	Use:   "do <project> [list] [item-ids...]",
	Short: "Have the AI implement todo items in the code repo",
	Long: `Spawns Claude in the project's code directory to implement todo items.
Gathers context from the knowledge base and todo lists, then lets the AI
work on the items. After completion, offers to mark items done and update
the knowledge base.

If no list/items are specified, shows a picker. If no items are specified,
the AI can work on any open items in the list.

Requires code_dir to be set: p set <project> code_dir ~/code/myrepo

Examples:
  p do serviceA                          # pick a list, AI chooses items
  p do serviceA feature-a                # AI works on all open items in feature-a
  p do serviceA feature-a 1 2            # AI works on specific items`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		projectName := args[0]
		dir, err := project.Resolve(cfg.ProjectRoot, projectName)
		if err != nil {
			return err
		}

		meta, err := project.LoadMeta(dir)
		if err != nil {
			return err
		}

		if meta.CodeDir == "" {
			return fmt.Errorf("code_dir not set for project %q — run: p set %s code_dir <path>", projectName, projectName)
		}

		if _, err := os.Stat(meta.CodeDir); err != nil {
			return fmt.Errorf("code directory %q does not exist", meta.CodeDir)
		}

		// Determine which items to work on
		var listName string
		var itemIDs []string
		var userMessage string

		if len(args) >= 2 {
			listName = args[1]
			for _, arg := range args[2:] {
				if looksLikeItemID(arg) {
					itemIDs = append(itemIDs, arg)
				} else {
					userMessage = strings.Join(args[2:], " ")
					itemIDs = nil
					break
				}
			}
		} else {
			listName, err = pickList(dir)
			if err != nil {
				return err
			}
		}

		// Also check --message flag
		if msg, _ := cmd.Flags().GetString("message"); msg != "" {
			userMessage = msg
		}

		list, err := todo.LoadList(dir, listName)
		if err != nil {
			return fmt.Errorf("loading list: %w", err)
		}

		// Build the task description
		taskDesc := buildDoPrompt(projectName, dir, list, listName, itemIDs)

		// Build MCP config for p tools
		pBinary, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolving executable path: %w", err)
		}

		mcpCfg := ai.MCPConfig(pBinary)
		mcpJSON, err := json.Marshal(mcpCfg)
		if err != nil {
			return fmt.Errorf("marshaling MCP config: %w", err)
		}

		claudePath := cfg.ClaudePath
		if claudePath == "" {
			claudePath = "claude"
		}
		model := cfg.ClaudeModel
		if model == "" {
			model = "claude-opus-4-6"
		}

		// Spawn claude in the code directory
		// Fold user message into system prompt so Claude has full context
		if userMessage != "" {
			taskDesc += "\n## Additional instructions from user\n\n" + userMessage + "\n"
		}

		claudeArgs := []string{
			"--system-prompt", taskDesc,
			"--mcp-config", string(mcpJSON),
			"--tools", "default,mcp",
			"--dangerously-skip-permissions",
			"--model", model,
			"--name", fmt.Sprintf("p-do-%s-%s", projectName, listName),
		}

		fmt.Fprintf(os.Stderr, "Spawning Claude in %s to work on %s/%s...\n", meta.CodeDir, projectName, listName)

		claudeCmd := exec.Command(claudePath, claudeArgs...)
		claudeCmd.Dir = meta.CodeDir
		claudeCmd.Stdin = os.Stdin
		claudeCmd.Stdout = os.Stdout
		claudeCmd.Stderr = os.Stderr

		if err := claudeCmd.Run(); err != nil {
			return fmt.Errorf("claude session failed: %w", err)
		}

		// After claude finishes, offer to update project state
		fmt.Println()
		ok, _ := tui.Confirm("Mark completed items as done and update knowledge?")
		if ok {
			return postDoUpdate(pBinary, claudePath, model, projectName, dir, listName)
		}

		return nil
	},
}

func buildDoPrompt(projectName, projectDir string, list *todo.List, listName string, itemIDs []string) string {
	var sb strings.Builder

	sb.WriteString("You are implementing tasks for the project \"")
	sb.WriteString(projectName)
	sb.WriteString("\".\n\n")

	sb.WriteString("## Tasks to implement\n\n")
	sb.WriteString(fmt.Sprintf("Todo list: **%s**\n\n", listName))

	if len(itemIDs) > 0 {
		sb.WriteString("Specific items to work on:\n\n")
		for _, id := range itemIDs {
			item, err := todo.ResolveItem(list, id)
			if err != nil {
				continue
			}
			sb.WriteString(fmt.Sprintf("- #%s: %s\n", id, item.Text))
		}
	} else {
		sb.WriteString("Open items (work on what makes sense):\n\n")
		sb.WriteString(todo.Render(list))
	}

	// Add knowledge context
	sb.WriteString("\n## Project knowledge\n\n")
	files, _ := knowledge.ListFiles(projectDir)
	for _, f := range files {
		content, err := knowledge.Read(projectDir, f)
		if err != nil {
			continue
		}
		sb.WriteString(fmt.Sprintf("### %s\n\n%s\n\n", f, content))
	}

	sb.WriteString("## Instructions\n\n")
	sb.WriteString("- Implement the tasks in this code repository.\n")
	sb.WriteString("- You have full access to read, edit, and run code.\n")
	sb.WriteString("- You also have `p` MCP tools to update the project's todo lists and knowledge base.\n")
	sb.WriteString("- When you complete an item, use the `todo_state` tool to mark it done.\n")
	sb.WriteString("- If you learn something important during implementation, use `knowledge_append` to document it.\n")
	sb.WriteString("- If a task turns out to be more complex than expected, use `todo_add` to break it into sub-tasks.\n")
	sb.WriteString("- Work through the items methodically. Commit your code changes as you go.\n")

	return sb.String()
}

func postDoUpdate(pBinary, claudePath, model, projectName, projectDir, listName string) error {
	task := ai.Task{
		ProjectName: projectName,
		ProjectDir:  projectDir,
		Input: fmt.Sprintf(`Review the todo list "%s" and the recent git history in the code repo.
Mark any items that were just implemented as done.
Add a brief summary of what was implemented to the knowledge base.`, listName),
		Mode: ai.ModePlan,
	}

	if err := ai.Run(pBinary, claudePath, model, task, ai.RunOptions{Stderr: claudeStderr()}); err != nil {
		return err
	}

	diff, err := git.Diff(projectDir)
	if err != nil {
		return err
	}

	if diff == "" {
		fmt.Println("No project updates needed.")
		return nil
	}

	fmt.Fprintf(os.Stderr, "\n--- Project changes ---\n%s\n", diff)

	ok, _ := tui.Confirm("Commit project updates?")
	if ok {
		return git.CommitAll(projectDir, fmt.Sprintf("p: post-implementation update for %s", listName))
	}

	git.RevertChanges(projectDir)
	fmt.Println("Changes reverted.")
	return nil
}

var itemIDRe = regexp.MustCompile(`^\d+(\.\d+)*$`)

func looksLikeItemID(s string) bool {
	return itemIDRe.MatchString(s)
}

func init() {
	doCmd.Flags().StringP("message", "m", "", "Custom instructions for the AI")
	rootCmd.AddCommand(doCmd)
}
