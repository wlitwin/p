package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/project"
)

var summarizeCmd = &cobra.Command{
	Use:   "summarize <project>",
	Short: "AI-generated project status summary",
	Long: `Generate a status report covering project health, progress,
blockers, and suggested next steps.

Example:
  p summarize serviceA`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		projectName := args[0]
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
			Input:       "Generate a comprehensive status summary for this project.",
			Mode:        ai.ModeAsk,
			CommandName: "summarize",
		}

		return ai.Run(pBinary, claudePath, model, task, ai.RunOptions{Stderr: claudeStderr()})
	},
}

func init() {
	aiCmd.AddCommand(summarizeCmd)
}
