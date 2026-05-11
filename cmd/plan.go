package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
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
		also, _ := cmd.Flags().GetStringSlice("also")
		return runAIWithCommit(aiTaskConfig{
			ProjectName: args[0],
			Input:       args[1],
			Mode:        ai.ModePlan,
			CommandName: "plan",
			CommitMsg:   fmt.Sprintf("p: AI plan — %s", truncate(args[1], 60)),
			AlsoNames:   also,
		})
	},
}

func init() {
	planCmd.Flags().StringSlice("also", nil, "Include context from other projects (comma-separated)")
	rootCmd.AddCommand(planCmd)
}
