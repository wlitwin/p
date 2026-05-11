package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

var rmListCmd = &cobra.Command{
	Use:   "rm-list <project> <list>",
	Short: "Delete a todo list",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		path := todo.ListPath(dir, args[1])
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("todo list %q not found", args[1])
		}

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

		if err := os.Remove(path); err != nil {
			return fmt.Errorf("deleting: %w", err)
		}

		if err := git.CommitAll(dir, fmt.Sprintf("p: delete todo list %s", args[1])); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		fmt.Printf("Deleted todo list %q\n", args[1])
		return nil
	},
}

func init() {
	rmListCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	rootCmd.AddCommand(rmListCmd)
}
