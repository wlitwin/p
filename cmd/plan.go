package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/display"
	"github.com/walter/p/internal/todo"
)

var planCmd = &cobra.Command{
	Use:   "plan <project> [description]",
	Short: "Open-ended AI planning — create multiple todos, update knowledge",
	Long: `Give the AI an open-ended task and let it explore the project,
create multiple todos, organize knowledge, and plan work.

If no description is provided, starts an interactive planning session with
full project context pre-loaded.

Examples:
  p plan serviceA "Write up v2 TODOs — v1 is complete"
  p plan serviceA "Break down the auth migration into concrete tasks"
  p plan serviceA                                          # interactive session`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		also, _ := cmd.Flags().GetStringSlice("also")
		listName, _ := cmd.Flags().GetString("list")
		cont, _ := cmd.Flags().GetBool("continue")

		input := ""
		if len(args) >= 2 {
			input = args[1]
		}

		// Resolve context patterns from the target list if specified
		dir, err := resolveProjectDir(args[0])
		if err != nil {
			return err
		}

		var contextPatterns []string
		if listName != "" {
			list, err := todo.LoadList(dir, listName)
			if err != nil {
				return fmt.Errorf("loading list for context: %w", err)
			}
			contextPatterns = ai.ResolveContext(dir, list)
		} else {
			contextPatterns = ai.ResolveContext(dir, nil)
		}

		commitMsg := "p: AI interactive planning session"
		if input != "" {
			commitMsg = fmt.Sprintf("p: AI plan — %s", display.Truncate(input, 60))
		}

		return runAIWithCommit(cmd.Context(), aiTaskConfig{
			ProjectName:     args[0],
			Input:           input,
			Mode:            ai.ModePlan,
			CommandName:     "plan",
			CommitMsg:       commitMsg,
			Continue:        cont,
			AlsoNames:       also,
			ContextPatterns: contextPatterns,
		})
	},
}

func init() {
	planCmd.Flags().BoolP("continue", "c", false, "Continue the previous conversation")
	planCmd.Flags().StringSlice("also", nil, "Include context from other projects (comma-separated)")
	planCmd.Flags().StringP("list", "l", "", "Target todo list (uses its context patterns for knowledge filtering)")
	rootCmd.AddCommand(planCmd)
}
