package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
)

var saveCmd = &cobra.Command{
	Use:   "save <project> [message...]",
	Short: "Commit any uncommitted changes in a project",
	Long: `Commits all pending changes (new files, modifications, deletions)
in a project directory. Useful after manual edits in Obsidian or a text editor.

If no message is provided, defaults to "p: manual save".

Examples:
  p save serviceA
  p save serviceA updated architecture docs`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		msg := "p: manual save"
		if len(args) > 1 {
			msg = "p: " + strings.Join(args[1:], " ")
		}

		if err := git.CommitAll(dir, msg); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		fmt.Println("Saved.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(saveCmd)
}
