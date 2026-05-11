package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/service"
)

var editKnowledgeCmd = &cobra.Command{
	Use:     "knowledge",
	Aliases: []string{"kb"},
	Short:   "Knowledge base edit primitives",
}

var editKnowledgeCreateCmd = &cobra.Command{
	Use:   "create <project> <filename> <title>",
	Short: "Create a knowledge document",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withProjectLock(args[0], func(dir string) error {
			tagsStr, _ := cmd.Flags().GetString("tags")
			var tags []string
			if tagsStr != "" {
				tags = strings.Split(tagsStr, ",")
				for i := range tags {
					tags[i] = strings.TrimSpace(tags[i])
				}
			}

			if err := service.KnowledgeCreate(dir, args[1], args[2], tags); err != nil {
				return err
			}

			if err := service.Commit(dir, fmt.Sprintf("p: create knowledge doc %q", args[2])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Printf("Created knowledge/%s.md\n", args[1])
			return nil
		})
	},
}

var editKnowledgeAppendCmd = &cobra.Command{
	Use:   "append <project> <filename> <content>",
	Short: "Append content to a knowledge document",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withProjectLock(args[0], func(dir string) error {
			section, _ := cmd.Flags().GetString("section")

			if err := service.KnowledgeAppend(dir, args[1], args[2], section); err != nil {
				return err
			}

			if err := service.Commit(dir, fmt.Sprintf("p: append to knowledge/%s", args[1])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Printf("Appended to knowledge/%s.md\n", args[1])
			return nil
		})
	},
}

var editKnowledgeReplaceCmd = &cobra.Command{
	Use:   "replace <project> <filename> <new-content>",
	Short: "Replace a section in a knowledge document",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withProjectLock(args[0], func(dir string) error {
			section, _ := cmd.Flags().GetString("section")
			if section == "" {
				return fmt.Errorf("--section is required for replace")
			}

			if err := service.KnowledgeReplace(dir, args[1], section, args[2]); err != nil {
				return err
			}

			if err := service.Commit(dir, fmt.Sprintf("p: replace section %q in knowledge/%s", section, args[1])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Printf("Replaced section %q in knowledge/%s.md\n", section, args[1])
			return nil
		})
	},
}

var editKnowledgeRenameCmd = &cobra.Command{
	Use:   "rename <project> <old-filename> <new-filename>",
	Short: "Rename a knowledge document",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withProjectLock(args[0], func(dir string) error {
			if err := service.KnowledgeRename(dir, args[1], args[2]); err != nil {
				return err
			}

			if err := service.Commit(dir, fmt.Sprintf("p: rename knowledge/%s to knowledge/%s", args[1], args[2])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Printf("Renamed knowledge/%s.md to knowledge/%s.md\n", args[1], args[2])
			return nil
		})
	},
}

func init() {
	editKnowledgeCreateCmd.Flags().String("tags", "", "Comma-separated tags")
	editKnowledgeAppendCmd.Flags().String("section", "", "Section heading to append under")
	editKnowledgeReplaceCmd.Flags().String("section", "", "Section heading to replace")

	editKnowledgeCmd.AddCommand(editKnowledgeCreateCmd)
	editKnowledgeCmd.AddCommand(editKnowledgeAppendCmd)
	editKnowledgeCmd.AddCommand(editKnowledgeReplaceCmd)
	editKnowledgeCmd.AddCommand(editKnowledgeRenameCmd)
	editCmd.AddCommand(editKnowledgeCmd)
}
