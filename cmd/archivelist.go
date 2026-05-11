package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/todo"
)

var archiveListCmd = &cobra.Command{
	Use:   "archive-list <project> [list]",
	Short: "Archive a finished todo list",
	Long: `Moves a todo list to todos/.archive/ so it doesn't clutter
active views but remains accessible. Use --restore to unarchive.

If no list is specified, automatically archives all lists where
every item is done.

Examples:
  p archive-list serviceA feature-a          # archive one list
  p archive-list serviceA                    # auto-archive all 100% done lists
  p archive-list serviceA feature-a --restore`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		restore, _ := cmd.Flags().GetBool("restore")
		ctx := cmd.Context()

		return withProjectLock(args[0], func(dir string) error {
			if len(args) == 2 {
				return archiveOneList(ctx, dir, args[1], restore)
			}
			return autoArchiveDone(ctx, dir)
		})
	},
}

func archiveOneList(ctx context.Context, dir, listName string, restore bool) error {
	archiveDir := filepath.Join(todo.ListDir(dir), ".archive")
	activePath := todo.ListPath(dir, listName)
	archivedPath := filepath.Join(archiveDir, listName+".md")

	if restore {
		if _, err := os.Stat(archivedPath); err != nil {
			return fmt.Errorf("archived list %q not found", listName)
		}
		if err := os.Rename(archivedPath, activePath); err != nil {
			return err
		}
		if err := git.CommitAll(ctx, dir, fmt.Sprintf("p: restore todo list %s from archive", listName)); err != nil {
			return fmt.Errorf("committing: %w", err)
		}
		fmt.Printf("Restored todo list %q\n", listName)
	} else {
		if _, err := os.Stat(activePath); err != nil {
			return fmt.Errorf("todo list %q not found", listName)
		}
		if err := os.MkdirAll(archiveDir, 0o755); err != nil {
			return err
		}
		if err := os.Rename(activePath, archivedPath); err != nil {
			return err
		}
		if err := git.CommitAll(ctx, dir, fmt.Sprintf("p: archive todo list %s", listName)); err != nil {
			return fmt.Errorf("committing: %w", err)
		}
		fmt.Printf("Archived todo list %q\n", listName)
	}
	return nil
}

func autoArchiveDone(ctx context.Context, dir string) error {
	names, err := todo.ListNames(dir)
	if err != nil {
		return err
	}

	var archived []string
	for _, name := range names {
		list, err := todo.LoadList(dir, name)
		if err != nil || len(list.Items) == 0 {
			continue
		}
		if allDone(list.Items) {
			if err := archiveOneList(ctx, dir, name, false); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not archive %s: %v\n", name, err)
				continue
			}
			archived = append(archived, name)
		}
	}

	if len(archived) == 0 {
		fmt.Println("No fully completed lists to archive.")
	}
	return nil
}

func allDone(items []*todo.Item) bool {
	for _, item := range items {
		if item.State != todo.Done {
			return false
		}
		if len(item.Children) > 0 && !allDone(item.Children) {
			return false
		}
	}
	return true
}

func init() {
	archiveListCmd.Flags().Bool("restore", false, "Restore an archived list")
	todoCmd.AddCommand(archiveListCmd)
}
