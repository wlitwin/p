package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
)

var setCmd = &cobra.Command{
	Use:   "set <project> [key] [value...]",
	Short: "View or set project metadata",
	Long: `View all project settings, get a specific value, or set a value.

Supported keys:
  description    Project description
  code_dir       Path to the code repository for this project

Examples:
  p set serviceA                                    # show all settings
  p set serviceA code_dir                           # show one setting
  p set serviceA code_dir ~/code/serviceA           # set a value
  p set serviceA description New payments service   # set description`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		meta, err := project.LoadMeta(dir)
		if err != nil {
			return err
		}

		switch len(args) {
		case 1:
			// Show all settings
			fmt.Printf("  %-15s %s\n", "name", meta.Name)
			fmt.Printf("  %-15s %s\n", "description", meta.Description)
			fmt.Printf("  %-15s %s\n", "code_dir", meta.CodeDir)
			fmt.Printf("  %-15s %s\n", "created", meta.Created.Format("2006-01-02"))
			fmt.Printf("  %-15s %v\n", "archived", meta.Archived)
			return nil

		case 2:
			// Get one setting
			key := args[1]
			switch key {
			case "description":
				fmt.Println(meta.Description)
			case "code_dir":
				fmt.Println(meta.CodeDir)
			case "name":
				fmt.Println(meta.Name)
			case "created":
				fmt.Println(meta.Created.Format("2006-01-02"))
			case "archived":
				fmt.Println(meta.Archived)
			default:
				return fmt.Errorf("unknown key %q — supported: description, code_dir, name, created, archived", key)
			}
			return nil

		default:
			// Set a value
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
					return fmt.Errorf("unknown key %q — settable: description, code_dir", key)
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
		}
	},
}

func init() {
	projectCmd.AddCommand(setCmd)
}
