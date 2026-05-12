package cmd

import (
	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
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
		projectName := args[0]
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
			Input:           "Generate a comprehensive status summary for this project.",
			Mode:            ai.ModeAsk,
			CommandName:     "summarize",
			ContextPatterns: ai.ResolveContext(dir, nil),
		}

		return ai.Run(cmd.Context(), pBinary, claudePath, model, task, ai.RunOptions{Stderr: claudeStderr()})
	},
}

func init() {
	aiCmd.AddCommand(summarizeCmd)
}
