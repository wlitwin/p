package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/service"
	"github.com/walter/p/internal/todo"
)

var contextCmd = &cobra.Command{
	Use:   "context <project> <list> [patterns...]",
	Short: "Set or view knowledge context patterns for a todo list",
	Long: `Set which knowledge docs are included in AI prompts when working
on a particular todo list. Patterns use glob syntax matching knowledge
doc names (without .md extension).

With --clear, removes the context field (reverts to project default or all).
With --show, displays the resolved context: what patterns apply and which
docs they match.

Pattern syntax:
  overview        Exact name match
  architecture/*  All docs directly in architecture/
  architecture/** All docs recursively under architecture/
  db-*            Prefix match
  *               All top-level docs
  **              Everything

Examples:
  p todo context myproj sprint-1 architecture/* decisions/db-*
  p todo context myproj sprint-1 --show
  p todo context myproj sprint-1 --clear`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		projectName := args[0]
		listName := args[1]
		patterns := args[2:]

		dir, err := project.Resolve(cfg.ProjectRoot, projectName)
		if err != nil {
			return err
		}

		showFlag, _ := cmd.Flags().GetBool("show")
		clearFlag, _ := cmd.Flags().GetBool("clear")

		if showFlag {
			return showContext(dir, listName)
		}

		if clearFlag {
			return withProjectLock(projectName, func(dir string) error {
				if err := service.SetListContext(cmd.Context(), dir, listName, nil); err != nil {
					return err
				}
				if err := service.Commit(cmd.Context(), dir, fmt.Sprintf("p: clear context on %s", listName)); err != nil {
					return fmt.Errorf("committing: %w", err)
				}
				fmt.Printf("Cleared context on %s (will use project default or all docs)\n", listName)
				return nil
			})
		}

		if len(patterns) == 0 {
			return showContext(dir, listName)
		}

		return withProjectLock(projectName, func(dir string) error {
			if err := service.SetListContext(cmd.Context(), dir, listName, patterns); err != nil {
				return err
			}
			if err := service.Commit(cmd.Context(), dir, fmt.Sprintf("p: set context on %s: %s", listName, strings.Join(patterns, ", "))); err != nil {
				return fmt.Errorf("committing: %w", err)
			}
			fmt.Printf("Set context on %s: %s\n", listName, strings.Join(patterns, ", "))
			return nil
		})
	},
}

func showContext(dir, listName string) error {
	list, err := todo.LoadList(dir, listName)
	if err != nil {
		return fmt.Errorf("loading list: %w", err)
	}

	resolved := ai.ResolveContext(dir, list)

	// Show context source
	if list.Context != nil {
		fmt.Printf("Context source: list %q\n", listName)
		if len(list.Context) == 0 {
			fmt.Println("Patterns: [] (no knowledge docs)")
		} else {
			fmt.Printf("Patterns: %s\n", strings.Join(list.Context, ", "))
		}
	} else {
		meta, _ := project.LoadMeta(dir)
		if meta.DefaultContext != nil {
			fmt.Println("Context source: project default")
			fmt.Printf("Patterns: %s\n", strings.Join(meta.DefaultContext, ", "))
		} else {
			fmt.Println("Context source: none (all docs included)")
		}
	}

	// Show matched files
	if resolved != nil {
		if len(resolved) == 0 {
			fmt.Println("\nMatched docs: (none)")
			return nil
		}
		files, err := knowledge.MatchFiles(dir, resolved)
		if err != nil {
			return err
		}
		if len(files) == 0 {
			fmt.Println("\nMatched docs: (none)")
		} else {
			fmt.Printf("\nMatched docs (%d):\n", len(files))
			for _, f := range files {
				fmt.Printf("  - %s\n", f)
			}
		}
	} else {
		files, _ := knowledge.ListFiles(dir)
		fmt.Printf("\nAll docs (%d):\n", len(files))
		for _, f := range files {
			fmt.Printf("  - %s\n", f)
		}
	}

	return nil
}

func init() {
	contextCmd.Flags().Bool("show", false, "Display the resolved context (patterns + matched docs)")
	contextCmd.Flags().Bool("clear", false, "Remove context patterns (revert to project default or all)")
	todoCmd.AddCommand(contextCmd)
}
