package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/project"
)

var diffCmd = &cobra.Command{
	Use:   "diff <project>",
	Short: "Show uncommitted changes in a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		gitCmd := exec.Command("git", "diff", "--stat")
		gitCmd.Dir = dir
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr
		if err := gitCmd.Run(); err != nil {
			return err
		}

		gitCmd2 := exec.Command("git", "diff")
		gitCmd2.Dir = dir
		gitCmd2.Stdout = os.Stdout
		gitCmd2.Stderr = os.Stderr
		return gitCmd2.Run()
	},
}

var revertCmd = &cobra.Command{
	Use:   "revert <project>",
	Short: "Undo the last commit",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		// Show what will be reverted
		gitCmd := exec.Command("git", "log", "-1", "--format=%h %s")
		gitCmd.Dir = dir
		out, err := gitCmd.Output()
		if err != nil {
			return fmt.Errorf("reading last commit: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Will revert: %s\n", string(out))

		autoYes, _ := cmd.Flags().GetBool("yes")
		if !autoYes {
			fmt.Fprint(os.Stderr, "Revert this commit? [y/N] ")
			var answer string
			_, _ = fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" && answer != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		revertGit := exec.Command("git", "revert", "--no-edit", "HEAD")
		revertGit.Dir = dir
		revertGit.Stdout = os.Stdout
		revertGit.Stderr = os.Stderr
		if err := revertGit.Run(); err != nil {
			return fmt.Errorf("reverting: %w", err)
		}

		fmt.Println("Reverted.")
		return nil
	},
}

func init() {
	revertCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	projectCmd.AddCommand(diffCmd)
	projectCmd.AddCommand(revertCmd)
}
