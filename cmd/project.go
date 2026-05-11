package cmd

import "github.com/spf13/cobra"

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Project lifecycle commands",
	Long: `Manage project lifecycle — create, configure, archive, and
view project history.

Examples:
  p project new serviceA --description "Payments service"
  p project set serviceA code_dir ~/code/serviceA
  p project log serviceA
  p project archive serviceA`,
}

func init() {
	rootCmd.AddCommand(projectCmd)
}
