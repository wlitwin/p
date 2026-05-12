package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/display"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/service"
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
		projectName := args[0]
		dir, err := resolveProjectDir(projectName)
		if err != nil {
			return err
		}

		isKnowledge, _ := cmd.Flags().GetBool("knowledge")
		useAI, _ := cmd.Flags().GetBool("ai")

		// Detect URLs — default to knowledge mode with AI
		var text string
		if len(args) == 3 {
			text = args[2]
		} else {
			text = args[1]
		}

		if !isKnowledge && display.LooksLikeURL(text) && !cmd.Flags().Changed("knowledge") {
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

			pBinary, err := resolvePBinary()
			if err != nil {
				return err
			}

			claudePath, model := resolveClaudeConfig()

			// Resolve context patterns from the target list if known.
			// ResolveContext handles the fallback chain internally:
			// list.Context > project.DefaultContext > nil (all docs).
			var listForContext *todo.List
			if listName != "" {
				if l, err := todo.LoadList(dir, listName); err == nil {
					listForContext = l
				}
			}
			contextPatterns := ai.ResolveContext(dir, listForContext)

			task := ai.Task{
				ProjectName:     projectName,
				ProjectDir:      dir,
				Input:           text,
				Mode:            mode,
				CommandName:     "add",
				ListName:        listName,
				ContextPatterns: contextPatterns,
			}

			if err := ai.Run(cmd.Context(), pBinary, claudePath, model, task, ai.RunOptions{Stderr: claudeStderr()}); err != nil {
				return err
			}

			diff, err := git.Diff(cmd.Context(), dir)
			if err != nil {
				return fmt.Errorf("getting diff: %w", err)
			}

			if diff == "" {
				fmt.Println("AI made no changes.")
				return nil
			}

			fmt.Fprintf(os.Stderr, "\n--- Changes ---\n%s\n", diff)

			commitMsg := fmt.Sprintf("p: AI added %s content from: %s", mode, display.Truncate(text, 60))
			if err := git.CommitAll(cmd.Context(), dir, commitMsg); err != nil {
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

		priority, _ := cmd.Flags().GetString("priority")
		if priority == "" {
			priority = cfg.DefaultPriority
			if priority == "" {
				priority = "now"
			}
		}
		dueDate, _ := cmd.Flags().GetString("due")

		if err := service.AddItem(cmd.Context(), dir, listName, text, todo.Priority(priority), dueDate, ""); err != nil {
			return err
		}

		if err := service.Commit(cmd.Context(), dir, fmt.Sprintf("p: add todo %q to %s", text, listName)); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		fmt.Printf("Added to %s/%s: %s\n", projectName, listName, text)
		return nil
	},
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
			open, _, blocked := todo.CountStates(list.Items)
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
	addCmd.Flags().StringP("list", "l", "", "Target todo list (skips interactive picker)")
	addCmd.Flags().String("priority", "", "Priority: now or backlog (default: now)")
	addCmd.Flags().String("due", "", "Due date: YYYY-MM-DD")
	rootCmd.AddCommand(addCmd)
}
