package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/display"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
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
		also, _ := cmd.Flags().GetStringSlice("also")
		listName, _ := cmd.Flags().GetString("list")

		// Resolve context patterns from the target list if specified
		var contextPatterns []string
		if listName != "" {
			dir, err := project.Resolve(cfg.ProjectRoot, args[0])
			if err != nil {
				return err
			}
			list, err := todo.LoadList(dir, listName)
			if err != nil {
				return fmt.Errorf("loading list for context: %w", err)
			}
			contextPatterns = ai.ResolveContext(dir, list)
		} else {
			// No specific list — use project default context
			dir, err := project.Resolve(cfg.ProjectRoot, args[0])
			if err != nil {
				return err
			}
			contextPatterns = ai.ResolveContext(dir, nil)
		}

		return runAIWithCommit(aiTaskConfig{
			ProjectName:     args[0],
			Input:           args[1],
			Mode:            ai.ModePlan,
			CommandName:     "plan",
			CommitMsg:       fmt.Sprintf("p: AI plan — %s", display.Truncate(args[1], 60)),
			AlsoNames:       also,
			ContextPatterns: contextPatterns,
		})
	},
}

func init() {
	planCmd.Flags().StringSlice("also", nil, "Include context from other projects (comma-separated)")
	planCmd.Flags().StringP("list", "l", "", "Target todo list (uses its context patterns for knowledge filtering)")
	rootCmd.AddCommand(planCmd)
}
