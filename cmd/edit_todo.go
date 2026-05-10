package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
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
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		listName := args[1]
		text := args[2]

		list, err := todo.LoadList(dir, listName)
		if err != nil {
			list, err = todo.CreateList(dir, listName, listName)
			if err != nil {
				return err
			}
		}

		priority, _ := cmd.Flags().GetString("priority")
		if priority == "" {
			priority = "now"
		}
		due, _ := cmd.Flags().GetString("due")
		parentID, _ := cmd.Flags().GetString("parent")

		item := todo.AddItem(list, text, todo.Priority(priority), due)

		if parentID != "" {
			parent, err := todo.ResolveItem(list, parentID)
			if err != nil {
				return fmt.Errorf("resolving parent: %w", err)
			}
			// Remove from top level and add as child
			list.Items = list.Items[:len(list.Items)-1]
			parent.Children = append(parent.Children, item)
		}

		if err := todo.SaveList(dir, listName, list); err != nil {
			return err
		}

		if err := git.CommitAll(dir, fmt.Sprintf("p: add todo %q to %s", text, listName)); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		fmt.Printf("Added: %s\n", text)
		return nil
	},
}

var editTodoUpdateCmd = &cobra.Command{
	Use:   "update <project> <list> <item-id> <new-text>",
	Short: "Update todo item text",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		list, err := todo.LoadList(dir, args[1])
		if err != nil {
			return err
		}

		item, err := todo.ResolveItem(list, args[2])
		if err != nil {
			return err
		}

		item.Text = args[3]

		if err := todo.SaveList(dir, args[1], list); err != nil {
			return err
		}

		if err := git.CommitAll(dir, fmt.Sprintf("p: update %s #%s text", args[1], args[2])); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		fmt.Printf("Updated %s #%s\n", args[1], args[2])
		return nil
	},
}

var editTodoStateCmd = &cobra.Command{
	Use:   "state <project> <list> <item-id> <open|blocked|done>",
	Short: "Set todo item state",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setItemState(args[0], args[1], args[2], todo.State(args[3]))
	},
}

var editTodoRemoveCmd = &cobra.Command{
	Use:   "remove <project> <list> <item-id>",
	Short: "Remove a todo item",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		list, err := todo.LoadList(dir, args[1])
		if err != nil {
			return err
		}

		if err := todo.RemoveItem(list, args[2]); err != nil {
			return err
		}

		if err := todo.SaveList(dir, args[1], list); err != nil {
			return err
		}

		if err := git.CommitAll(dir, fmt.Sprintf("p: remove %s #%s", args[1], args[2])); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		fmt.Printf("Removed %s #%s\n", args[1], args[2])
		return nil
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
