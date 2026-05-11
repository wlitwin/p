package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/validate"
)

var newCmd = &cobra.Command{
	Use:   "new <project>",
	Short: "Create a new project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		name := args[0]
		if err := validate.ProjectName(name); err != nil {
			return err
		}
		desc, _ := cmd.Flags().GetString("description")

		if err := project.Create(cfg.ProjectRoot, name, desc); err != nil {
			return err
		}

		projectDir := filepath.Join(cfg.ProjectRoot, name)
		if err := git.Init(projectDir); err != nil {
			return fmt.Errorf("initializing git: %w", err)
		}

		if err := git.CommitAll(projectDir, fmt.Sprintf("p: create project %q", name)); err != nil {
			return fmt.Errorf("initial commit: %w", err)
		}

		fmt.Printf("Created project %q at %s\n", name, projectDir)
		return nil
	},
}

func init() {
	newCmd.Flags().String("description", "", "Project description")
	projectCmd.AddCommand(newCmd)
}
