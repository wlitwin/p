package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/project"
)

var reviewCmd = &cobra.Command{
	Use:   "review <project>",
	Short: "AI reviews recent changes and suggests next actions",
	Args:  cobra.ExactArgs(1),
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
			Input: `Review the recent git history and current project state. Provide:
1. A summary of what changed recently
2. What's currently in progress (open items)
3. Any blockers or risks you notice
4. Suggested next actions based on priorities and dependencies`,
			Mode: ai.ModeAsk,
		}

		return ai.Run(pBinary, claudePath, model, task)
	},
}

func init() {
	rootCmd.AddCommand(reviewCmd)
}
