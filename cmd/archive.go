package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
)

var archiveCmd = &cobra.Command{
	Use:   "archive <project>",
	Short: "Archive a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setArchived(args[0], true)
	},
}

var unarchiveCmd = &cobra.Command{
	Use:   "unarchive <project>",
	Short: "Unarchive a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return setArchived(args[0], false)
	},
}

func setArchived(name string, archived bool) error {
	if err := requireProjectRoot(); err != nil {
		return err
	}

	dir, err := project.Resolve(cfg.ProjectRoot, name)
	if err != nil {
		return err
	}

	meta, err := project.LoadMeta(dir)
	if err != nil {
		return err
	}

	meta.Archived = archived
	if err := project.SaveMeta(dir, meta); err != nil {
		return err
	}

	action := "archived"
	if !archived {
		action = "unarchived"
	}

	if err := git.CommitAll(dir, fmt.Sprintf("p: %s project %q", action, name)); err != nil {
		return fmt.Errorf("committing: %w", err)
	}

	fmt.Printf("Project %q %s\n", name, action)
	return nil
}

func init() {
	projectCmd.AddCommand(archiveCmd)
	projectCmd.AddCommand(unarchiveCmd)
}
