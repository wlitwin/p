package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/service"
)

var rmListCmd = &cobra.Command{
	Use:   "rm-list <project> <list>",
	Short: "Delete a todo list",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withProjectLock(args[0], func(dir string) error {
			autoYes, _ := cmd.Flags().GetBool("yes")
			if !autoYes {
				fmt.Fprintf(os.Stderr, "Delete todo list %q? [y/N] ", args[1])
				var answer string
				_, _ = fmt.Scanln(&answer)
				if answer != "y" && answer != "Y" && answer != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			if err := service.RemoveList(dir, args[1]); err != nil {
				return err
			}

			if err := service.Commit(dir, fmt.Sprintf("p: delete todo list %s", args[1])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Printf("Deleted todo list %q\n", args[1])
			return nil
		})
	},
}

func init() {
	rmListCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	todoCmd.AddCommand(rmListCmd)
}
