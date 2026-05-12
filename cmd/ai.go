package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Specialized AI commands",
	Long: `AI-powered project analysis and updates.

For everyday AI use, see the top-level commands:
  p ask    — ask questions about project state
  p plan   — open-ended AI planning
  p do     — AI implements todo items in code

These specialized commands provide deeper analysis:
  p ai review    — review recent changes, update todos/knowledge
  p ai summarize — generate a status report`,
}

func init() {
	rootCmd.AddCommand(aiCmd)
}

// aiTaskConfig holds the parameters for creating and running an AI task
// that auto-commits its changes. Used by plan and review commands.
type aiTaskConfig struct {
	ProjectName     string
	Input           string
	Mode            ai.Mode
	CommandName     string
	CommitMsg       string
	Continue        bool     // resume last conversation
	AlsoNames       []string // additional project names for multi-project context
	ContextPatterns []string // knowledge doc glob patterns; nil means include all
}

// runAIWithCommit resolves a project, runs an AI task, and commits any changes.
// This is the shared flow used by plan and review commands. The context
// enables cancellation of the claude subprocess.
func runAIWithCommit(ctx context.Context, taskCfg aiTaskConfig) error {
	dir, err := resolveProjectDir(taskCfg.ProjectName)
	if err != nil {
		return err
	}

	pBinary, err := resolvePBinary()
	if err != nil {
		return err
	}

	claudePath, model := resolveClaudeConfig()

	task := ai.Task{
		ProjectName:     taskCfg.ProjectName,
		ProjectDir:      dir,
		Input:           taskCfg.Input,
		Mode:            taskCfg.Mode,
		CommandName:     taskCfg.CommandName,
		ContextPatterns: taskCfg.ContextPatterns,
	}

	// Resolve --also projects for multi-project context
	for _, name := range taskCfg.AlsoNames {
		aDir, err := project.Resolve(cfg.ProjectRoot, name)
		if err != nil {
			return fmt.Errorf("resolving --also project: %w", err)
		}
		task.AlsoProjects = append(task.AlsoProjects, aDir)
		task.AlsoNames = append(task.AlsoNames, name)
	}

	if err := ai.Run(ctx, pBinary, claudePath, model, task, ai.RunOptions{Continue: taskCfg.Continue, Stderr: claudeStderr()}); err != nil {
		return err
	}

	diff, err := git.Diff(ctx, dir)
	if err != nil {
		return fmt.Errorf("getting diff: %w", err)
	}

	if diff == "" {
		fmt.Println("AI made no changes.")
		return nil
	}

	fmt.Fprintf(os.Stderr, "\n--- Changes ---\n%s\n", diff)

	if err := git.CommitAll(ctx, dir, taskCfg.CommitMsg); err != nil {
		return fmt.Errorf("committing: %w", err)
	}

	fmt.Println("Changes committed.")
	return nil
}
