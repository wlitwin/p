package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
)

var describeCmd = &cobra.Command{
	Use:   "describe <project> <description...>",
	Short: "Set or update a project's description",
	Long: `Set the description for a project. The description is shown in
'p list' and 'p status' output.

Examples:
  p describe serviceA New payments processing service
  p describe serviceA ""   # clear the description`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withProjectLock(args[0], func(dir string) error {
			meta, err := project.LoadMeta(dir)
			if err != nil {
				return err
			}

			meta.Description = strings.Join(args[1:], " ")

			if err := project.SaveMeta(dir, meta); err != nil {
				return err
			}

			if err := git.CommitAll(cmd.Context(), dir, fmt.Sprintf("p: update description for %s", args[0])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			if meta.Description == "" {
				fmt.Printf("Cleared description for %s\n", args[0])
			} else {
				fmt.Printf("Description for %s: %s\n", args[0], meta.Description)
			}
			return nil
		})
	},
}

func init() {
	projectCmd.AddCommand(describeCmd)
}
