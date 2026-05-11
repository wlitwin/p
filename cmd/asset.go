package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/asset"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
)

var assetCmd = &cobra.Command{
	Use:   "asset",
	Short: "Manage project assets (images, files, attachments)",
}

var assetAddCmd = &cobra.Command{
	Use:   "add <project> <file>",
	Short: "Add a file to the project's assets",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withProjectLock(args[0], func(dir string) error {
			filename, err := asset.Copy(dir, args[1])
			if err != nil {
				return err
			}

			if err := git.CommitAll(cmd.Context(), dir, fmt.Sprintf("p: add asset %s", filename)); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Printf("Added assets/%s\n", filename)
			return nil
		})
	},
}

var assetListCmd = &cobra.Command{
	Use:   "list <project>",
	Short: "List project assets",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		infos, err := asset.ListWithInfo(dir)
		if err != nil {
			return err
		}
		if len(infos) == 0 {
			fmt.Println("No assets.")
			return nil
		}

		for _, info := range infos {
			fmt.Printf("  %-30s  %s\n", info.Name, formatSize(info.Size))
		}
		return nil
	},
}

var assetRemoveCmd = &cobra.Command{
	Use:   "remove <project> <filename>",
	Short: "Remove an asset from the project",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withProjectLock(args[0], func(dir string) error {
			if err := asset.Delete(dir, args[1]); err != nil {
				return err
			}

			if err := git.CommitAll(cmd.Context(), dir, fmt.Sprintf("p: remove asset %s", args[1])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Printf("Removed assets/%s\n", args[1])
			return nil
		})
	},
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func init() {
	assetCmd.AddCommand(assetAddCmd)
	assetCmd.AddCommand(assetListCmd)
	assetCmd.AddCommand(assetRemoveCmd)
	rootCmd.AddCommand(assetCmd)
}
