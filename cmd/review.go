package cmd

import (
	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/project"
)

const reviewInput = `Review the recent git history and current project state. Then:
1. Summarize what changed recently
2. Identify what's currently in progress (open items)
3. Flag any blockers, risks, or stale items
4. If you find items that appear completed based on recent changes, mark them done
5. If you notice gaps or missing tasks, add new todos
6. Update knowledge docs if recent changes warrant it
7. Suggest prioritized next actions`

var reviewCmd = &cobra.Command{
	Use:   "review <project>",
	Short: "AI reviews project and can update todos/knowledge",
	Long: `AI reviews recent git history, current todos, and knowledge docs,
then can suggest and make changes — marking items done, adding new todos,
updating knowledge based on what it finds.

Unlike 'p summarize' (read-only report), 'p review' can write changes.

Examples:
  p review serviceA`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}
		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}
		return runAIWithCommit(cmd.Context(), aiTaskConfig{
			ProjectName:     args[0],
			Input:           reviewInput,
			Mode:            ai.ModePlan,
			CommandName:     "review",
			CommitMsg:       "p: AI review",
			ContextPatterns: ai.ResolveContext(dir, nil),
		})
	},
}

func init() {
	aiCmd.AddCommand(reviewCmd)
}
