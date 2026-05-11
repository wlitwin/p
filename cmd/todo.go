package cmd

import "github.com/spf13/cobra"

var todoCmd = &cobra.Command{
	Use:   "todo",
	Short: "Todo item management commands",
	Long: `Manage todo items beyond basic add/done — change state, set
priority, due dates, tags, move between lists, and archive.

Examples:
  p todo block serviceA feature-a 3
  p todo priority serviceA feature-a 1 now
  p todo tag serviceA feature-a 1 bug frontend
  p todo move serviceA feature-a 2 backlog
  p todo archive-list serviceA feature-a`,
}

func init() {
	rootCmd.AddCommand(todoCmd)
}
