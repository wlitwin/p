package cmd

import (
	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
)

var askCmd = &cobra.Command{
	Use:   "ask <project> [question]",
	Short: "Ask the AI a question about the project",
	Long: `Query the project state using AI. The AI reads todos, knowledge docs,
and project metadata to answer your question. Read-only — no changes are made.

If no question is provided, starts an interactive chat session with full
project context pre-loaded.

Examples:
  p ask serviceA "What's the current status of the DB refactor?"
  p ask serviceA "What are the biggest risks right now?"
  p ask serviceA                                          # interactive session`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		question := ""
		if len(args) >= 2 {
			question = args[1]
		}

		dir, err := resolveProjectDir(projectName)
		if err != nil {
			return err
		}

		pBinary, err := resolvePBinary()
		if err != nil {
			return err
		}

		claudePath, model := resolveClaudeConfig()

		task := ai.Task{
			ProjectName:     projectName,
			ProjectDir:      dir,
			Input:           question,
			Mode:            ai.ModeAsk,
			CommandName:     "ask",
			ContextPatterns: ai.ResolveContext(dir, nil),
		}

		cont, _ := cmd.Flags().GetBool("continue")
		return ai.Run(cmd.Context(), pBinary, claudePath, model, task, ai.RunOptions{Continue: cont, Stderr: claudeStderr()})
	},
}

func init() {
	askCmd.Flags().BoolP("continue", "c", false, "Continue the previous conversation")
	rootCmd.AddCommand(askCmd)
}
