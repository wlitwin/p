package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

var addCmd = &cobra.Command{
	Use:   "add <project> [list] <text>",
	Short: "Add a todo item or knowledge entry",
	Long: `Add a todo item to a project. If the list name is omitted, you'll be
prompted to choose one.

Use --knowledge (-k) to add to the knowledge base instead.`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		isKnowledge, _ := cmd.Flags().GetBool("knowledge")
		if isKnowledge {
			// Knowledge addition will be handled by AI in a later phase.
			// For now, stub it out.
			fmt.Println("Knowledge addition via AI is not yet implemented.")
			fmt.Println("Use `p edit knowledge` subcommands for manual edits.")
			return nil
		}

		var projectName, listName, text string
		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}
		projectName = args[0]

		if len(args) == 3 {
			listName = args[1]
			text = args[2]
		} else {
			// 2 args: project + text, need to pick a list
			text = args[1]
			names, err := todo.ListNames(dir)
			if err != nil {
				return err
			}
			if len(names) == 0 {
				fmt.Print("No todo lists exist. Enter name for new list: ")
				if _, err := fmt.Scanln(&listName); err != nil {
					return err
				}
			} else {
				fmt.Println("Choose a list:")
				for i, n := range names {
					fmt.Printf("  %d. %s\n", i+1, n)
				}
				fmt.Printf("  %d. (create new)\n", len(names)+1)
				fmt.Print("Choice: ")
				var choice int
				if _, err := fmt.Scanln(&choice); err != nil {
					return err
				}
				if choice < 1 || choice > len(names)+1 {
					return fmt.Errorf("invalid choice")
				}
				if choice == len(names)+1 {
					fmt.Print("New list name: ")
					if _, err := fmt.Scanln(&listName); err != nil {
						return err
					}
				} else {
					listName = names[choice-1]
				}
			}
		}

		// Load or create list
		list, err := todo.LoadList(dir, listName)
		if err != nil {
			list, err = todo.CreateList(dir, listName, listName)
			if err != nil {
				return err
			}
		}

		priority, _ := cmd.Flags().GetString("priority")
		if priority == "" {
			priority = cfg.DefaultPriority
			if priority == "" {
				priority = "now"
			}
		}
		dueDate, _ := cmd.Flags().GetString("due")

		todo.AddItem(list, text, todo.Priority(priority), dueDate)

		if err := todo.SaveList(dir, listName, list); err != nil {
			return err
		}

		if err := git.CommitAll(dir, fmt.Sprintf("p: add todo %q to %s", text, listName)); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		fmt.Printf("Added to %s/%s: %s\n", projectName, listName, text)
		return nil
	},
}

func init() {
	addCmd.Flags().BoolP("knowledge", "k", false, "Add to knowledge base instead of todos")
	addCmd.Flags().String("priority", "", "Priority: now or backlog (default: now)")
	addCmd.Flags().String("due", "", "Due date: YYYY-MM-DD")
	rootCmd.AddCommand(addCmd)
}
