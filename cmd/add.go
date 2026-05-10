package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/tui"
)

var addCmd = &cobra.Command{
	Use:   "add <project> [list] <text>",
	Short: "Add a todo item or knowledge entry",
	Long: `Add a todo item to a project. If the list name is omitted, you'll be
prompted to choose one.

Use --knowledge (-k) to add to the knowledge base instead.
Use --ai to have the AI agent decide placement and wording.`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		projectName := args[0]
		dir, err := project.Resolve(cfg.ProjectRoot, projectName)
		if err != nil {
			return err
		}

		isKnowledge, _ := cmd.Flags().GetBool("knowledge")
		useAI, _ := cmd.Flags().GetBool("ai")
		autoYes, _ := cmd.Flags().GetBool("yes")

		// Detect URLs — default to knowledge mode with AI
		var text string
		if len(args) == 3 {
			text = args[2]
		} else {
			text = args[1]
		}

		if !isKnowledge && looksLikeURL(text) && !cmd.Flags().Changed("knowledge") {
			fmt.Fprintf(os.Stderr, "Detected URL — defaulting to knowledge mode with AI.\n")
			isKnowledge = true
			useAI = true
		}

		// AI path
		if useAI || isKnowledge {
			mode := ai.ModeTodo
			if isKnowledge {
				mode = ai.ModeKnowledge
			}

			listName := ""
			if len(args) == 3 && !isKnowledge {
				listName = args[1]
			}

			pBinary, err := os.Executable()
			if err != nil {
				return fmt.Errorf("resolving executable path: %w", err)
			}

			claudePath := cfg.ClaudePath
			if claudePath == "" {
				claudePath = "claude"
			}
			model := cfg.ClaudeModel
			if model == "" {
				model = "claude-opus-4-6"
			}

			task := ai.Task{
				ProjectName: projectName,
				ProjectDir:  dir,
				Input:       text,
				Mode:        mode,
				ListName:    listName,
			}

			if err := ai.Run(pBinary, claudePath, model, task); err != nil {
				return err
			}

			// Show diff and confirm
			diff, err := git.Diff(dir)
			if err != nil {
				return fmt.Errorf("getting diff: %w", err)
			}

			if diff == "" {
				fmt.Println("AI made no changes.")
				return nil
			}

			fmt.Fprintf(os.Stderr, "\n--- Changes ---\n%s\n", diff)

			if !autoYes {
				ok, err := tui.Confirm("Commit these changes?")
				if err != nil || !ok {
					if revertErr := git.RevertChanges(dir); revertErr != nil {
						return fmt.Errorf("reverting changes: %w", revertErr)
					}
					fmt.Println("Changes reverted.")
					return nil
				}
			}

			commitMsg := fmt.Sprintf("p: AI added %s content from: %s", mode, truncate(text, 60))
			if err := git.CommitAll(dir, commitMsg); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Println("Changes committed.")
			return nil
		}

		// Manual (non-AI) path
		var listName string
		if l, _ := cmd.Flags().GetString("list"); l != "" {
			listName = l
		} else if len(args) == 3 {
			listName = args[1]
		} else {
			listName, err = pickList(dir)
			if err != nil {
				return err
			}
		}

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

func looksLikeURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func pickList(projectDir string) (string, error) {
	names, err := todo.ListNames(projectDir)
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return tui.Input("New list name:")
	}

	items := make([]tui.PickerItem, 0, len(names)+1)
	for _, n := range names {
		list, loadErr := todo.LoadList(projectDir, n)
		desc := ""
		if loadErr == nil {
			open, _, blocked := countStates(list.Items)
			desc = fmt.Sprintf("open=%d blocked=%d", open, blocked)
		}
		items = append(items, tui.PickerItem{Label: n, Desc: desc})
	}
	items = append(items, tui.PickerItem{Label: "+ Create new list"})

	choice, err := tui.Pick("Choose a todo list:", items)
	if err != nil {
		return "", err
	}

	if choice == len(names) {
		return tui.Input("New list name:")
	}
	return names[choice], nil
}

func init() {
	addCmd.Flags().BoolP("knowledge", "k", false, "Add to knowledge base instead of todos")
	addCmd.Flags().Bool("ai", false, "Use AI agent for smart placement and wording")
	addCmd.Flags().BoolP("yes", "y", false, "Auto-confirm AI changes without prompting")
	addCmd.Flags().StringP("list", "l", "", "Target todo list (skips interactive picker)")
	addCmd.Flags().String("priority", "", "Priority: now or backlog (default: now)")
	addCmd.Flags().String("due", "", "Due date: YYYY-MM-DD")
	rootCmd.AddCommand(addCmd)
}
