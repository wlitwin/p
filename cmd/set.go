package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
)

var setCmd = &cobra.Command{
	Use:   "set <project> <key> <value...>",
	Short: "Set a project metadata field",
	Long: `Set a metadata field on a project.

Supported keys:
  description    Project description
  code_dir       Path to the code repository for this project

Examples:
  p set serviceA description New payments processing service
  p set serviceA code_dir ~/code/serviceA`,
	Args: cobra.MinimumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withProjectLock(args[0], func(dir string) error {
			meta, err := project.LoadMeta(dir)
			if err != nil {
				return err
			}

			key := args[1]
			value := strings.Join(args[2:], " ")

			switch key {
			case "description":
				meta.Description = value
			case "code_dir":
				meta.CodeDir = expandHome(value)
			default:
				return fmt.Errorf("unknown key %q — supported: description, code_dir", key)
			}

			if err := project.SaveMeta(dir, meta); err != nil {
				return err
			}

			if err := git.CommitAll(dir, fmt.Sprintf("p: set %s=%s for %s", key, value, args[0])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Printf("Set %s = %s\n", key, value)
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(setCmd)
}
