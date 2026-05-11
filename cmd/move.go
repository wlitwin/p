package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/service"
)

var moveCmd = &cobra.Command{
	Use:   "move <project> <list> <item-id> <target-list>",
	Short: "Move a todo item to another list",
	Args:  cobra.ExactArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withProjectLock(args[0], func(dir string) error {
			if err := service.MoveItem(dir, args[1], args[2], args[3]); err != nil {
				return err
			}
			if err := service.Commit(dir, fmt.Sprintf("p: move %s #%s to %s", args[1], args[2], args[3])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}
			fmt.Printf("Moved %s #%s → %s\n", args[1], args[2], args[3])
			return nil
		})
	},
}

func init() {
	todoCmd.AddCommand(moveCmd)
}
