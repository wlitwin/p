package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

var moveCmd = &cobra.Command{
	Use:   "move <project> <list> <item-id> <target-list>",
	Short: "Move a todo item to another list",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		projectName := args[0]
		srcListName := args[1]
		itemID := args[2]
		dstListName := args[3]

		dir, err := project.Resolve(cfg.ProjectRoot, projectName)
		if err != nil {
			return err
		}

		srcList, err := todo.LoadList(dir, srcListName)
		if err != nil {
			return fmt.Errorf("loading source list: %w", err)
		}

		item, err := todo.ResolveItem(srcList, itemID)
		if err != nil {
			return err
		}

		itemCopy := todo.DeepCopyItem(item)

		// Add to destination first (safe — source still has the item)
		dstList, err := todo.LoadList(dir, dstListName)
		if err != nil {
			dstList, err = todo.CreateList(dir, dstListName, dstListName)
			if err != nil {
				return fmt.Errorf("creating target list: %w", err)
			}
		}
		dstList.Items = append(dstList.Items, itemCopy)
		if err := todo.SaveList(dir, dstListName, dstList); err != nil {
			return fmt.Errorf("saving target: %w", err)
		}

		// Only remove from source after destination is safely written
		if err := todo.RemoveItem(srcList, itemID); err != nil {
			return fmt.Errorf("removing from source: %w", err)
		}
		if err := todo.SaveList(dir, srcListName, srcList); err != nil {
			return err
		}

		if err := git.CommitAll(dir, fmt.Sprintf("p: move %s #%s to %s", srcListName, itemID, dstListName)); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		fmt.Printf("Moved %s #%s → %s\n", srcListName, itemID, dstListName)
		return nil
	},
}

func init() {
	todoCmd.AddCommand(moveCmd)
}
