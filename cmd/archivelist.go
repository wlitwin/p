package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/todo"
)

var archiveListCmd = &cobra.Command{
	Use:   "archive-list <project> <list>",
	Short: "Archive a finished todo list",
	Long: `Moves a todo list to todos/.archive/ so it doesn't clutter
active views but remains accessible. Use --restore to unarchive.

Examples:
  p archive-list serviceA feature-a
  p archive-list serviceA feature-a --restore`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		restore, _ := cmd.Flags().GetBool("restore")

		return withProjectLock(args[0], func(dir string) error {
			archiveDir := filepath.Join(todo.ListDir(dir), ".archive")
			activePath := todo.ListPath(dir, args[1])
			archivedPath := filepath.Join(archiveDir, args[1]+".md")

			if restore {
				if _, err := os.Stat(archivedPath); err != nil {
					return fmt.Errorf("archived list %q not found", args[1])
				}
				if err := os.Rename(archivedPath, activePath); err != nil {
					return err
				}
				if err := git.CommitAll(dir, fmt.Sprintf("p: restore todo list %s from archive", args[1])); err != nil {
					return fmt.Errorf("committing: %w", err)
				}
				fmt.Printf("Restored todo list %q\n", args[1])
			} else {
				if _, err := os.Stat(activePath); err != nil {
					return fmt.Errorf("todo list %q not found", args[1])
				}
				if err := os.MkdirAll(archiveDir, 0o755); err != nil {
					return err
				}
				if err := os.Rename(activePath, archivedPath); err != nil {
					return err
				}
				if err := git.CommitAll(dir, fmt.Sprintf("p: archive todo list %s", args[1])); err != nil {
					return fmt.Errorf("committing: %w", err)
				}
				fmt.Printf("Archived todo list %q\n", args[1])
			}
			return nil
		})
	},
}

func init() {
	archiveListCmd.Flags().Bool("restore", false, "Restore an archived list")
	rootCmd.AddCommand(archiveListCmd)
}
