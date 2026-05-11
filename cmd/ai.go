package cmd

import "github.com/spf13/cobra"

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Specialized AI commands",
	Long: `AI-powered project analysis and updates.

For everyday AI use, see the top-level commands:
  p ask    — ask questions about project state
  p plan   — open-ended AI planning
  p do     — AI implements todo items in code

These specialized commands provide deeper analysis:
  p ai review    — review recent changes, update todos/knowledge
  p ai summarize — generate a status report`,
}

func init() {
	rootCmd.AddCommand(aiCmd)
}
