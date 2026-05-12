package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/project"
)

var logCmd = &cobra.Command{
	Use:   "log <project>",
	Short: "Show git history for a project",
	Long: `Show git history for a project with optional filtering.

Examples:
  p project log myproject
  p project log myproject -n 50
  p project log myproject --since 2026-05-01
  p project log myproject --since 2026-05-01 --until 2026-05-10
  p project log myproject --grep "knowledge"
  p project log myproject --since 2026-05-01 --grep "add"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		count, _ := cmd.Flags().GetInt("count")
		since, _ := cmd.Flags().GetString("since")
		until, _ := cmd.Flags().GetString("until")
		grep, _ := cmd.Flags().GetString("grep")

		gitArgs := []string{"log",
			fmt.Sprintf("--max-count=%d", count),
			"--format=%C(yellow)%h%C(reset) %C(dim)%cr%C(reset) %s",
		}

		if since != "" {
			gitArgs = append(gitArgs, "--since="+since)
		}
		if until != "" {
			gitArgs = append(gitArgs, "--until="+until)
		}
		if grep != "" {
			gitArgs = append(gitArgs, "--grep="+grep, "--regexp-ignore-case")
		}

		gitCmd := exec.Command("git", gitArgs...)
		gitCmd.Dir = dir
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		return gitCmd.Run()
	},
}

func init() {
	logCmd.Flags().IntP("count", "n", 20, "Number of commits to show")
	logCmd.Flags().String("since", "", "Show commits after date (e.g. 2026-05-01, '2 weeks ago')")
	logCmd.Flags().String("until", "", "Show commits before date (e.g. 2026-05-10, 'yesterday')")
	logCmd.Flags().String("grep", "", "Filter commits by message content (case-insensitive)")
	projectCmd.AddCommand(logCmd)
}
