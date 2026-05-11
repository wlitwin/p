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
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		count, _ := cmd.Flags().GetInt("count")

		gitCmd := exec.Command("git", "log",
			fmt.Sprintf("--max-count=%d", count),
			"--format=%C(yellow)%h%C(reset) %C(dim)%cr%C(reset) %s",
		)
		gitCmd.Dir = dir
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		return gitCmd.Run()
	},
}

func init() {
	logCmd.Flags().IntP("count", "n", 20, "Number of commits to show")
	projectCmd.AddCommand(logCmd)
}
