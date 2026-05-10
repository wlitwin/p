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

var planCmd = &cobra.Command{
	Use:   "plan <project> <description>",
	Short: "Open-ended AI planning — create multiple todos, update knowledge",
	Long: `Give the AI an open-ended task and let it explore the project,
create multiple todos, organize knowledge, and plan work.

Examples:
  p plan serviceA "Write up v2 TODOs — v1 is complete"
  p plan serviceA "Break down the auth migration into concrete tasks"
  p plan serviceA "Review the current state and suggest what's missing"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		projectName := args[0]
		input := args[1]

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
			Input:       input,
			Mode:        ai.ModePlan,
		}

		also, _ := cmd.Flags().GetStringSlice("also")
		for _, name := range also {
			aDir, err := project.Resolve(cfg.ProjectRoot, name)
			if err != nil {
				return fmt.Errorf("resolving --also project: %w", err)
			}
			task.AlsoProjects = append(task.AlsoProjects, aDir)
			task.AlsoNames = append(task.AlsoNames, name)
		}

		if err := ai.Run(pBinary, claudePath, model, task); err != nil {
			return err
		}

		diff, err := git.Diff(dir)
		if err != nil {
			return fmt.Errorf("getting diff: %w", err)
		}

		if diff == "" {
			fmt.Println("AI made no changes.")
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

		commitMsg := fmt.Sprintf("p: AI plan — %s", truncate(input, 60))
		if err := git.CommitAll(dir, commitMsg); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		fmt.Println("Changes committed.")
		return nil
	},
}

func init() {
	planCmd.Flags().BoolP("yes", "y", false, "Auto-confirm AI changes without prompting")
	planCmd.Flags().StringSlice("also", nil, "Include context from other projects (comma-separated)")
	rootCmd.AddCommand(planCmd)
}
