package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/tui"
)

var showCmd = &cobra.Command{
	Use:   "show <project> <list-or-doc>",
	Short: "Show a todo list or knowledge document",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		isKnowledge, _ := cmd.Flags().GetBool("knowledge")

		if isKnowledge {
			content, err := knowledge.Read(dir, args[1])
			if err != nil {
				return fmt.Errorf("reading knowledge doc: %w", err)
			}
			fmt.Print(tui.RenderWikiLinks(content, dir))
			return nil
		}

		list, err := todo.LoadList(dir, args[1])
		if err != nil {
			// Try as knowledge doc
			content, kerr := knowledge.Read(dir, args[1])
			if kerr != nil {
				return fmt.Errorf("not found as todo list or knowledge doc: %s", args[1])
			}
			fmt.Print(tui.RenderWikiLinks(content, dir))
			return nil
		}

		fmt.Printf("# %s\n\n", list.Title)
		printItems(list.Items, "", 1, dir)
		return nil
	},
}

func init() {
	showCmd.Flags().BoolP("knowledge", "k", false, "Show a knowledge document")
	rootCmd.AddCommand(showCmd)
}
