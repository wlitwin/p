package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

var agentCmd = &cobra.Command{
	Use:   "agent <project> <list>",
	Short: "Autonomously work through a todo list",
	Long: `Runs an AI agent loop that works through a todo list, implementing
items one at a time. After each AI session, the agent checks the list
state and continues until all items are done or no progress is made.

Stopping conditions:
  - All items in the list are done
  - An iteration makes no progress (no items marked done)
  - Maximum iterations reached (default 10)
  - AI session fails

Requires code_dir to be set: p set <project> code_dir ~/code/myrepo

Examples:
  p agent serviceA feature-list
  p agent serviceA sprint-1 --max-iterations 5
  p agent serviceA bugs --message "Focus on the critical ones first"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		listName := args[1]
		maxIter, _ := cmd.Flags().GetInt("max-iterations")
		userMessage, _ := cmd.Flags().GetString("message")

		dir, err := resolveProjectDir(projectName)
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

		// Verify list exists
		if _, err := todo.LoadList(dir, listName); err != nil {
			return fmt.Errorf("loading list: %w", err)
		}

		pBinary, err := resolvePBinary()
		if err != nil {
			return err
		}

		claudePath, model := resolveClaudeConfig()

		mcpJSON, err := ai.MCPConfigJSON(pBinary)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "🤖 Starting agent loop for %s/%s (max %d iterations)\n\n", projectName, listName, maxIter)

		var lastOpenCount = -1

		for iter := 1; iter <= maxIter; iter++ {
			// Load current list state
			list, err := todo.LoadList(dir, listName)
			if err != nil {
				return fmt.Errorf("iteration %d: loading list: %w", iter, err)
			}

			open, done, blocked := todo.CountStates(list.Items)
			total := open + done + blocked

			// Stop if all items are done
			if open == 0 && blocked == 0 {
				fmt.Fprintf(os.Stderr, "✅ All %d items done! Agent finished after %d iteration(s).\n", done, iter-1)
				return nil
			}

			// Stop if no progress since last iteration (not first iteration)
			if iter > 1 && open == lastOpenCount {
				fmt.Fprintf(os.Stderr, "⚠️  No progress — %d items still open after iteration %d. Stopping.\n", open, iter-1)
				return nil
			}

			lastOpenCount = open

			fmt.Fprintf(os.Stderr, "── Iteration %d/%d ── %d open, %d done, %d blocked (of %d total) ──\n\n",
				iter, maxIter, open, done, blocked, total)

			// Build system prompt for this iteration
			prompt := buildAgentPrompt(projectName, dir, list, listName, iter, maxIter, userMessage)

			claudeArgs := []string{
				"--append-system-prompt", prompt,
				"--mcp-config", mcpJSON,
				"--dangerously-skip-permissions",
				"--model", model,
				"--name", fmt.Sprintf("p-agent-%s-%s", projectName, listName),
			}

			claudeCmd := exec.Command(claudePath, claudeArgs...)
			claudeCmd.Dir = meta.CodeDir
			claudeCmd.Stdin = os.Stdin
			claudeCmd.Stdout = os.Stdout
			claudeCmd.Stderr = os.Stderr

			if err := claudeCmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "\n❌ AI session failed on iteration %d: %v\n", iter, err)
				return fmt.Errorf("agent iteration %d failed: %w", iter, err)
			}

			// Commit any project data changes from this iteration
			commitMsg := fmt.Sprintf("p: agent iteration %d for %s", iter, listName)
			if err := git.CommitAll(cmd.Context(), dir, commitMsg); err != nil {
				return fmt.Errorf("committing iteration %d: %w", iter, err)
			}

			fmt.Fprintf(os.Stderr, "\n")
		}

		// Ran out of iterations — report final state
		list, err := todo.LoadList(dir, listName)
		if err != nil {
			return fmt.Errorf("loading final state: %w", err)
		}
		open, done, blocked := todo.CountStates(list.Items)
		fmt.Fprintf(os.Stderr, "⚠️  Reached max iterations (%d). Final state: %d open, %d done, %d blocked.\n",
			maxIter, open, done, blocked)

		return nil
	},
}

func buildAgentPrompt(projectName, projectDir string, list *todo.List, listName string, iteration, maxIter int, userMessage string) string {
	// Check for a full template replacement first
	if tmpl := ai.LoadTemplate(projectDir, "agent"); tmpl != "" {
		contextPatterns := ai.ResolveContext(projectDir, list)
		data := ai.BuildTemplateData(projectName, projectDir, "agent", userMessage, listName, contextPatterns)
		if result, err := ai.ExecuteTemplate(tmpl, data); err == nil {
			return result
		}
		fmt.Fprintf(os.Stderr, "warning: prompt template error, using default prompt\n")
	}

	var sb strings.Builder

	sb.WriteString("You are implementing tasks for the project \"")
	sb.WriteString(projectName)
	sb.WriteString("\".\n\n")

	fmt.Fprintf(&sb, "Project knowledge base: `%s`\n", projectDir)
	fmt.Fprintf(&sb, "Knowledge docs are in: `%s/knowledge/`\n", projectDir)
	fmt.Fprintf(&sb, "Todo lists are in: `%s/todos/`\n\n", projectDir)

	fmt.Fprintf(&sb, "## Agent loop — iteration %d of %d\n\n", iteration, maxIter)
	sb.WriteString("You are part of an automated agent loop working through a todo list.\n")
	sb.WriteString("After this session ends, the loop will check progress and may invoke you again for the next item(s).\n\n")
	sb.WriteString("**Focus on completing 1-3 open items this iteration.** Don't try to do everything at once.\n")
	sb.WriteString("Mark items done with `todo_state` as you complete them.\n\n")

	sb.WriteString("## Tasks to implement\n\n")
	fmt.Fprintf(&sb, "Todo list: **%s**\n\n", listName)
	sb.WriteString("Open items (work on what makes sense):\n\n")
	sb.WriteString(todo.Render(list))

	// Add knowledge context
	sb.WriteString("\n## Project knowledge\n\n")
	contextPatterns := ai.ResolveContext(projectDir, list)
	var files []string
	if contextPatterns != nil {
		if len(contextPatterns) > 0 {
			files, _ = knowledge.MatchFiles(projectDir, contextPatterns)
		}
	} else {
		files, _ = knowledge.ListFiles(projectDir)
	}
	for _, f := range files {
		content, err := knowledge.Read(projectDir, f)
		if err != nil {
			continue
		}
		fmt.Fprintf(&sb, "### %s\n\n%s\n\n", f, content)
	}

	// Inject custom prompt
	customPrompt := ai.LoadCustomPrompt(projectDir, "agent")
	if customPrompt != "" {
		sb.WriteString("## Custom instructions\n\n")
		sb.WriteString(customPrompt)
		sb.WriteString("\n\n")
	}

	if userMessage != "" {
		sb.WriteString("## Additional instructions from user\n\n")
		sb.WriteString(userMessage)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Instructions\n\n")
	sb.WriteString("- Implement the tasks in this code repository.\n")
	sb.WriteString("- You have full access to read, edit, and run code.\n")
	sb.WriteString("- You also have `p` MCP tools to update the project's todo lists and knowledge base.\n")
	sb.WriteString("- When you complete an item, use the `todo_state` tool to mark it done.\n")
	sb.WriteString("- If you learn something important during implementation, use `knowledge_append` to document it.\n")
	sb.WriteString("- If a task turns out to be more complex than expected, use `todo_add` to break it into sub-tasks.\n")
	sb.WriteString("- Work through the items methodically. Commit your code changes as you go.\n")
	sb.WriteString("- You can use `todo_context` to update which knowledge docs are relevant to this list, keeping future AI sessions focused.\n")
	sb.WriteString("- Focus on making concrete progress — the agent loop will handle re-invocation.\n")

	return sb.String()
}

func init() {
	agentCmd.Flags().IntP("max-iterations", "i", 10, "Maximum number of agent iterations")
	agentCmd.Flags().StringP("message", "m", "", "Custom instructions for the AI")
	rootCmd.AddCommand(agentCmd)
}
