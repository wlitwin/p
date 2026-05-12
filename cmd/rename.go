package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/validate"
)

var renameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename a project",
	Long: `Rename a project directory and update its metadata.

Examples:
  p project rename old-service new-service`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		oldName := args[0]
		newName := args[1]

		if oldName == newName {
			return fmt.Errorf("old and new names are the same")
		}

		if err := validate.ProjectName(newName); err != nil {
			return fmt.Errorf("invalid new name: %w", err)
		}

		// Resolve old project
		oldDir, err := project.Resolve(cfg.ProjectRoot, oldName)
		if err != nil {
			return err
		}

		// Check new name doesn't already exist
		newDir := fmt.Sprintf("%s/%s", cfg.ProjectRoot, newName)
		if _, err := os.Stat(newDir); err == nil {
			return fmt.Errorf("project %q already exists", newName)
		}

		// Rename the directory
		if err := os.Rename(oldDir, newDir); err != nil {
			return fmt.Errorf("renaming directory: %w", err)
		}

		// Update meta.yaml
		meta, err := project.LoadMeta(newDir)
		if err != nil {
			// Roll back directory rename on failure
			_ = os.Rename(newDir, oldDir)
			return fmt.Errorf("loading metadata: %w", err)
		}

		meta.Name = newName
		if err := project.SaveMeta(newDir, meta); err != nil {
			_ = os.Rename(newDir, oldDir)
			return fmt.Errorf("saving metadata: %w", err)
		}

		if err := git.CommitAll(cmd.Context(), newDir, fmt.Sprintf("p: rename project %q to %q", oldName, newName)); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		fmt.Printf("Renamed %s → %s\n", oldName, newName)
		return nil
	},
}

func init() {
	projectCmd.AddCommand(renameCmd)
}
