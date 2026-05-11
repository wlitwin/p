package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/tui"
)

var reviewCmd = &cobra.Command{
	Use:   "review <project>",
	Short: "AI reviews project and can update todos/knowledge",
	Long: `AI reviews recent git history, current todos, and knowledge docs,
then can suggest and make changes — marking items done, adding new todos,
updating knowledge based on what it finds.

Unlike 'p summarize' (read-only report), 'p review' can write changes.

Examples:
  p review serviceA`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		projectName := args[0]
		dir, err := project.Resolve(cfg.ProjectRoot, projectName)
		if err != nil {
			return err
		}

		pBinary, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolving executable path: %w", err)
		}

		claudePath := cfg.ClaudePath
		if claudePath == "" {
			claudePath = "claude"
		}
		model := cfg.ClaudeModel
		if model == "" {
			model = "claude-opus-4-6"
		}

		task := ai.Task{
			ProjectName: projectName,
			ProjectDir:  dir,
			CommandName: "review",
			Input: `Review the recent git history and current project state. Then:
1. Summarize what changed recently
2. Identify what's currently in progress (open items)
3. Flag any blockers, risks, or stale items
4. If you find items that appear completed based on recent changes, mark them done
5. If you notice gaps or missing tasks, add new todos
6. Update knowledge docs if recent changes warrant it
7. Suggest prioritized next actions`,
			Mode: ai.ModePlan,
		}

		if err := ai.Run(pBinary, claudePath, model, task, ai.RunOptions{Stderr: claudeStderr()}); err != nil {
			return err
		}

		diff, err := git.Diff(dir)
		if err != nil {
			return fmt.Errorf("getting diff: %w", err)
		}

		if diff == "" {
			fmt.Println("No changes made.")
			return nil
		}

		fmt.Fprintf(os.Stderr, "\n--- Changes ---\n%s\n", diff)

		autoYes, _ := cmd.Flags().GetBool("yes")
		if !autoYes {
			ok, err := tui.Confirm("Commit these changes?")
			if err != nil || !ok {
				if revertErr := git.RevertChanges(dir); revertErr != nil {
					return fmt.Errorf("reverting changes: %w", revertErr)
				}
				fmt.Println("Changes reverted.")
				return nil
			}
		}

		if err := git.CommitAll(dir, "p: AI review"); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		fmt.Println("Changes committed.")
		return nil
	},
}

func init() {
	reviewCmd.Flags().BoolP("yes", "y", false, "Auto-confirm AI changes without prompting")
	rootCmd.AddCommand(reviewCmd)
}
