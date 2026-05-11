package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/service"
)

var setCmd = &cobra.Command{
	Use:   "set <project> [key] [value...]",
	Short: "View or set project metadata",
	Long: `View all project settings, get a specific value, or set a value.

Supported keys:
  description      Project description
  code_dir         Path to the code repository for this project
  default-context  Default knowledge doc patterns for AI context scoping

For default-context, pass patterns as separate args. Use --clear to remove.

Examples:
  p set serviceA                                        # show all settings
  p set serviceA code_dir                               # show one setting
  p set serviceA code_dir ~/code/serviceA               # set a value
  p set serviceA description New payments service       # set description
  p set serviceA default-context overview architecture/*
  p set serviceA default-context --clear`,
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
			fmt.Printf("  %-18s %s\n", "name", meta.Name)
			fmt.Printf("  %-18s %s\n", "description", meta.Description)
			fmt.Printf("  %-18s %s\n", "code_dir", meta.CodeDir)
			fmt.Printf("  %-18s %s\n", "created", meta.Created.Format("2006-01-02"))
			fmt.Printf("  %-18s %v\n", "archived", meta.Archived)
			if meta.DefaultContext != nil {
				fmt.Printf("  %-18s %s\n", "default-context", strings.Join(meta.DefaultContext, ", "))
			} else {
				fmt.Printf("  %-18s (not set)\n", "default-context")
			}
			return nil

		case 2:
			// Get one setting (or clear default-context with --clear)
			key := args[1]
			if key == "default-context" {
				clearFlag, _ := cmd.Flags().GetBool("clear")
				if clearFlag {
					return withProjectLock(args[0], func(dir string) error {
						if err := service.SetDefaultContext(dir, nil); err != nil {
							return err
						}
						if err := service.Commit(dir, "p: clear default-context"); err != nil {
							return fmt.Errorf("committing: %w", err)
						}
						fmt.Println("Cleared default-context")
						return nil
					})
				}
				if meta.DefaultContext != nil {
					fmt.Println(strings.Join(meta.DefaultContext, ", "))
				} else {
					fmt.Println("(not set)")
				}
				return nil
			}
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
				return fmt.Errorf("unknown key %q — supported: description, code_dir, default-context, name, created, archived", key)
			}
			return nil

		default:
			// Set a value
			return withProjectLock(args[0], func(dir string) error {
				key := args[1]

				// default-context takes multiple patterns, not a joined string
				if key == "default-context" {
					clearFlag, _ := cmd.Flags().GetBool("clear")
					var patterns []string
					if !clearFlag {
						patterns = args[2:]
					}
					if err := service.SetDefaultContext(dir, patterns); err != nil {
						return err
					}
					if err := service.Commit(dir, fmt.Sprintf("p: set default-context for %s", args[0])); err != nil {
						return fmt.Errorf("committing: %w", err)
					}
					if clearFlag {
						fmt.Println("Cleared default-context")
					} else {
						fmt.Printf("Set default-context = %s\n", strings.Join(patterns, ", "))
					}
					return nil
				}

				meta, err := project.LoadMeta(dir)
				if err != nil {
					return err
				}

				value := strings.Join(args[2:], " ")

				switch key {
				case "description":
					meta.Description = value
				case "code_dir":
					meta.CodeDir = expandHome(value)
				default:
					return fmt.Errorf("unknown key %q — settable: description, code_dir, default-context", key)
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
	setCmd.Flags().Bool("clear", false, "Clear the setting (for default-context)")
	projectCmd.AddCommand(setCmd)
}
