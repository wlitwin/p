package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
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
			return renderShow(content)
		}

		list, err := todo.LoadList(dir, args[1])
		if err != nil {
			// Try as knowledge doc
			content, kerr := knowledge.Read(dir, args[1])
			if kerr != nil {
				return fmt.Errorf("not found as todo list or knowledge doc: %s", args[1])
			}
			return renderShow(content)
		}

		md := todo.Render(list)
		return renderShow(md)
		return nil
	},
}

func renderShow(md string) error {
	rendered, err := ai.RenderMarkdown(md)
	if err != nil {
		fmt.Print(md)
		return nil
	}
	fmt.Print(rendered)
	return nil
}

func init() {
	showCmd.Flags().BoolP("knowledge", "k", false, "Show a knowledge document")
	rootCmd.AddCommand(showCmd)
}
