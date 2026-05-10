package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
)

var knowledgeCmd = &cobra.Command{
	Use:     "knowledge",
	Aliases: []string{"kb"},
	Short:   "Knowledge base commands",
}

var knowledgeDeleteCmd = &cobra.Command{
	Use:   "delete <project> <filename>",
	Short: "Delete a knowledge document",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		path := knowledge.FilePath(dir, args[1])
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("knowledge doc %q not found", args[1])
		}

		autoYes, _ := cmd.Flags().GetBool("yes")
		if !autoYes {
			fmt.Fprintf(os.Stderr, "Delete knowledge/%s.md? [y/N] ", args[1])
			var answer string
			fmt.Scanln(&answer)
			if answer != "y" && answer != "Y" && answer != "yes" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		if err := os.Remove(path); err != nil {
			return fmt.Errorf("deleting: %w", err)
		}

		if err := git.CommitAll(dir, fmt.Sprintf("p: delete knowledge/%s", args[1])); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		fmt.Printf("Deleted knowledge/%s.md\n", args[1])
		return nil
	},
}

var knowledgeSearchCmd = &cobra.Command{
	Use:   "search <project> <query>",
	Short: "Search knowledge documents",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		files, err := knowledge.ListFiles(dir)
		if err != nil {
			return err
		}

		query := args[1]
		found := false
		for _, f := range files {
			content, err := knowledge.Read(dir, f)
			if err != nil {
				continue
			}
			if containsIgnoreCase(content, query) {
				fmt.Printf("  %s  %s\n", f+".md", matchContext(content, query))
				found = true
			}
		}

		if !found {
			fmt.Println("No matches found.")
		}
		return nil
	},
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		len(substr) > 0 &&
		findIgnoreCase(s, substr) >= 0
}

func findIgnoreCase(s, substr string) int {
	s = toLower(s)
	substr = toLower(substr)
	return indexOf(s, substr)
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func init() {
	knowledgeDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	knowledgeCmd.AddCommand(knowledgeDeleteCmd)
	knowledgeCmd.AddCommand(knowledgeSearchCmd)
	rootCmd.AddCommand(knowledgeCmd)
}
