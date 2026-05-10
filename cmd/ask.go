package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/project"
)

var askCmd = &cobra.Command{
	Use:   "ask <project> <question>",
	Short: "Ask the AI a question about the project",
	Long: `Query the project state using AI. The AI reads todos, knowledge docs,
and project metadata to answer your question. Read-only — no changes are made.

Examples:
  p ask serviceA "What's the current status of the DB refactor?"
  p ask serviceA "What are the biggest risks right now?"
  p ask serviceA "Summarize what we've decided so far"
  p ask serviceA "What's left before we can ship v1?"`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		projectName := args[0]
		question := args[1]

		dir, err := project.Resolve(cfg.ProjectRoot, projectName)
		if err != nil {
			return err
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
			Input:       question,
			Mode:        ai.ModeAsk,
		}

		cont, _ := cmd.Flags().GetBool("continue")
		return ai.Run(pBinary, claudePath, model, task, ai.RunOptions{Continue: cont, Stderr: claudeStderr()})
	},
}

func init() {
	askCmd.Flags().BoolP("continue", "c", false, "Continue the previous conversation")
	rootCmd.AddCommand(askCmd)
}
