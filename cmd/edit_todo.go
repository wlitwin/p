package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/service"
	"github.com/walter/p/internal/todo"
)

var editTodoCmd = &cobra.Command{
	Use:   "todo",
	Short: "Todo edit primitives",
}

var editTodoAddCmd = &cobra.Command{
	Use:   "add <project> <list> <text>",
	Short: "Add a todo item",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withProjectLock(args[0], func(dir string) error {
			priority, _ := cmd.Flags().GetString("priority")
			if priority == "" {
				priority = "now"
			}
			due, _ := cmd.Flags().GetString("due")
			parentID, _ := cmd.Flags().GetString("parent")

			if err := service.AddItem(cmd.Context(), dir, args[1], args[2], todo.Priority(priority), due, parentID); err != nil {
				return err
			}

			if err := service.Commit(cmd.Context(), dir, fmt.Sprintf("p: add todo %q to %s", args[2], args[1])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Printf("Added: %s\n", args[2])
			return nil
		})
	},
}

var editTodoUpdateCmd = &cobra.Command{
	Use:   "update <project> <list> <item-id> <new-text>",
	Short: "Update todo item text",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withProjectLock(args[0], func(dir string) error {
			if err := service.UpdateItemText(cmd.Context(), dir, args[1], args[2], args[3]); err != nil {
				return err
			}
			if err := service.Commit(cmd.Context(), dir, fmt.Sprintf("p: update %s #%s text", args[1], args[2])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}
			fmt.Printf("Updated %s #%s\n", args[1], args[2])
			return nil
		})
	},
}

var editTodoStateCmd = &cobra.Command{
	Use:   "state <project> <list> <item-id> <open|blocked|done>",
	Short: "Set todo item state",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withProjectLock(args[0], func(dir string) error {
			if err := service.SetItemState(cmd.Context(), dir, args[1], args[2], todo.State(args[3])); err != nil {
				return err
			}
			if err := service.Commit(cmd.Context(), dir, fmt.Sprintf("p: mark %s #%s as %s", args[1], args[2], args[3])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}
			fmt.Printf("Marked %s #%s as %s\n", args[1], args[2], args[3])
			return nil
		})
	},
}

var editTodoRemoveCmd = &cobra.Command{
	Use:   "remove <project> <list> <item-id>",
	Short: "Remove a todo item",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withProjectLock(args[0], func(dir string) error {
			if err := service.RemoveItem(cmd.Context(), dir, args[1], args[2]); err != nil {
				return err
			}
			if err := service.Commit(cmd.Context(), dir, fmt.Sprintf("p: remove %s #%s", args[1], args[2])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}
			fmt.Printf("Removed %s #%s\n", args[1], args[2])
			return nil
		})
	},
}

func init() {
	editTodoAddCmd.Flags().String("priority", "now", "Priority: now or backlog")
	editTodoAddCmd.Flags().String("due", "", "Due date: YYYY-MM-DD")
	editTodoAddCmd.Flags().String("parent", "", "Parent item ID to nest under")

	editTodoCmd.AddCommand(editTodoAddCmd)
	editTodoCmd.AddCommand(editTodoUpdateCmd)
	editTodoCmd.AddCommand(editTodoStateCmd)
	editTodoCmd.AddCommand(editTodoRemoveCmd)
	editCmd.AddCommand(editTodoCmd)
}
