package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/service"
	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/validate"
)

func makeStateCmd(name string, state todo.State, short string) *cobra.Command {
	return &cobra.Command{
		Use:   fmt.Sprintf("%s <project> <list> <item-id> [item-id...]", name),
		Short: short,
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withProjectLock(args[0], func(dir string) error {
				ids := args[2:]
				for _, id := range ids {
					if err := service.SetItemState(dir, args[1], id, state); err != nil {
						return err
					}
					fmt.Printf("Marked %s #%s as %s\n", args[1], id, state)
				}
				msg := fmt.Sprintf("p: mark %s #%s as %s", args[1], ids[0], state)
				if len(ids) > 1 {
					msg = fmt.Sprintf("p: mark %s #%s as %s", args[1], strings.Join(ids, ",#"), state)
				}
				if err := service.Commit(dir, msg); err != nil {
					return fmt.Errorf("committing: %w", err)
				}
				return nil
			})
		},
	}
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
			if err := service.SetItemPriority(dir, args[1], args[2], todo.Priority(args[3])); err != nil {
				return err
			}
			if err := service.Commit(dir, fmt.Sprintf("p: set %s #%s priority to %s", args[1], args[2], args[3])); err != nil {
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
			if err := service.SetItemDue(dir, args[1], args[2], args[3]); err != nil {
				return err
			}
			if err := service.Commit(dir, fmt.Sprintf("p: set %s #%s due date to %s", args[1], args[2], args[3])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}
			fmt.Printf("Set %s #%s due date to %s\n", args[1], args[2], args[3])
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(makeStateCmd("done", todo.Done, "Mark a todo item as done"))
	todoCmd.AddCommand(makeStateCmd("block", todo.Blocked, "Mark a todo item as blocked"))
	todoCmd.AddCommand(makeStateCmd("open", todo.Open, "Reopen a todo item"))
	todoCmd.AddCommand(priorityCmd)
	todoCmd.AddCommand(dueCmd)
}
