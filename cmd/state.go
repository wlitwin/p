package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/validate"
)

func makeStateCmd(name string, state todo.State, short string) *cobra.Command {
	return &cobra.Command{
		Use:   fmt.Sprintf("%s <project> <list> <item-id> [item-id...]", name),
		Short: short,
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, id := range args[2:] {
				if err := setItemState(args[0], args[1], id, state); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func setItemState(projectName, listName, itemID string, state todo.State) error {
	return withProjectLock(projectName, func(dir string) error {
		list, err := todo.LoadList(dir, listName)
		if err != nil {
			return err
		}

		item, err := todo.ResolveItem(list, itemID)
		if err != nil {
			return err
		}

		todo.SetState(item, state)

		if err := todo.SaveList(dir, listName, list); err != nil {
			return err
		}

		if err := git.CommitAll(dir, fmt.Sprintf("p: mark %s #%s as %s", listName, itemID, state)); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		fmt.Printf("Marked %s #%s as %s\n", listName, itemID, state)
		return nil
	})
}

var priorityCmd = &cobra.Command{
	Use:   "priority <project> <list> <item-id> <now|backlog>",
	Short: "Set item priority",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validate.Priority(args[3]); err != nil {
			return err
		}
		return withProjectLock(args[0], func(dir string) error {
			list, err := todo.LoadList(dir, args[1])
			if err != nil {
				return err
			}
			item, err := todo.ResolveItem(list, args[2])
			if err != nil {
				return err
			}
			item.Priority = todo.Priority(args[3])
			if err := todo.SaveList(dir, args[1], list); err != nil {
				return err
			}
			if err := git.CommitAll(dir, fmt.Sprintf("p: set %s #%s priority to %s", args[1], args[2], args[3])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}
			fmt.Printf("Set %s #%s priority to %s\n", args[1], args[2], args[3])
			return nil
		})
	},
}

var dueCmd = &cobra.Command{
	Use:   "due <project> <list> <item-id> <YYYY-MM-DD>",
	Short: "Set item due date",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validate.Date(args[3]); err != nil {
			return err
		}
		return withProjectLock(args[0], func(dir string) error {
			list, err := todo.LoadList(dir, args[1])
			if err != nil {
				return err
			}
			item, err := todo.ResolveItem(list, args[2])
			if err != nil {
				return err
			}
			item.Due = args[3]
			if err := todo.SaveList(dir, args[1], list); err != nil {
				return err
			}
			if err := git.CommitAll(dir, fmt.Sprintf("p: set %s #%s due date to %s", args[1], args[2], args[3])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}
			fmt.Printf("Set %s #%s due date to %s\n", args[1], args[2], args[3])
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(makeStateCmd("done", todo.Done, "Mark a todo item as done"))
	rootCmd.AddCommand(makeStateCmd("block", todo.Blocked, "Mark a todo item as blocked"))
	rootCmd.AddCommand(makeStateCmd("open", todo.Open, "Reopen a todo item"))
	rootCmd.AddCommand(priorityCmd)
	rootCmd.AddCommand(dueCmd)
}
