package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/display"
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
			_, _ = fmt.Scanln(&answer)
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
			if display.ContainsIgnoreCase(content, query) {
				fmt.Printf("  %s  %s\n", f+".md", display.MatchContext(content, query))
				found = true
			}
		}

		if !found {
			fmt.Println("No matches found.")
		}
		return nil
	},
}

var knowledgeListCmd = &cobra.Command{
	Use:   "list <project>",
	Short: "List knowledge documents with details",
	Args:  cobra.ExactArgs(1),
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
		if len(files) == 0 {
			fmt.Println("No knowledge docs.")
			return nil
		}

		tagFilter, _ := cmd.Flags().GetString("tag")

		for _, f := range files {
			content, err := knowledge.Read(dir, f)
			if err != nil {
				continue
			}
			tags := knowledge.ExtractTags(content)

			if tagFilter != "" && !slices.Contains(tags, tagFilter) {
				continue
			}

			info, _ := os.Stat(knowledge.FilePath(dir, f))
			size := ""
			if info != nil {
				size = fmt.Sprintf("%d bytes", info.Size())
			}

			tagStr := ""
			if len(tags) > 0 {
				tagStr = " [" + strings.Join(tags, ", ") + "]"
			}
			fmt.Printf("  %-20s  %s%s\n", f, size, tagStr)
		}
		return nil
	},
}

var knowledgeCreateFromTemplateCmd = &cobra.Command{
	Use:   "create <project> <filename> <title>",
	Short: "Create a knowledge doc, optionally from a template",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		template, _ := cmd.Flags().GetString("template")
		tagsStr, _ := cmd.Flags().GetString("tags")
		var tags []string
		if tagsStr != "" {
			for t := range strings.SplitSeq(tagsStr, ",") {
				tags = append(tags, strings.TrimSpace(t))
			}
		}

		if err := knowledge.Create(dir, args[1], args[2], tags); err != nil {
			return err
		}

		if template != "" {
			content := templateContent(template)
			if content != "" {
				if err := knowledge.Append(dir, args[1], content, ""); err != nil {
					return fmt.Errorf("applying template: %w", err)
				}
			}
		}

		if err := git.CommitAll(dir, fmt.Sprintf("p: create knowledge doc %q", args[2])); err != nil {
			return fmt.Errorf("committing: %w", err)
		}

		fmt.Printf("Created knowledge/%s.md\n", args[1])
		return nil
	},
}

func templateContent(name string) string {
	switch name {
	case "decision-record":
		return `## Context

What is the issue that we're seeing that is motivating this decision or change?

## Decision

What is the change that we're proposing and/or doing?

## Consequences

What becomes easier or more difficult to do because of this change?`

	case "meeting-notes":
		return `## Attendees

-

## Agenda

1.

## Notes

## Action Items

- [ ]`

	case "runbook":
		return `## Overview

## Prerequisites

## Steps

1.

## Troubleshooting

## Rollback`

	default:
		return ""
	}
}

var knowledgeArchiveCmd = &cobra.Command{
	Use:   "archive <project> [filename]",
	Short: "Archive a knowledge document",
	Long: `Moves a knowledge doc to knowledge/.archive/. Use --restore to unarchive.
If no filename is given, shows archived docs available to restore.

Examples:
  p knowledge archive serviceA old-decisions
  p knowledge archive serviceA old-decisions --restore
  p knowledge archive serviceA                           # list archived docs`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		restore, _ := cmd.Flags().GetBool("restore")

		return withProjectLock(args[0], func(dir string) error {
			archiveDir := filepath.Join(knowledge.Dir(dir), ".archive")

			if len(args) == 1 {
				// List archived docs
				entries, err := os.ReadDir(archiveDir)
				if err != nil {
					fmt.Println("No archived knowledge docs.")
					return nil
				}
				fmt.Println("Archived knowledge docs:")
				for _, e := range entries {
					if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
						fmt.Printf("  %s\n", strings.TrimSuffix(e.Name(), ".md"))
					}
				}
				return nil
			}

			filename := args[1]
			activePath := knowledge.FilePath(dir, filename)
			archivedPath := filepath.Join(archiveDir, filename+".md")

			if restore {
				if _, err := os.Stat(archivedPath); err != nil {
					return fmt.Errorf("archived doc %q not found", filename)
				}
				if err := os.Rename(archivedPath, activePath); err != nil {
					return err
				}
				if err := git.CommitAll(dir, fmt.Sprintf("p: restore knowledge/%s from archive", filename)); err != nil {
					return fmt.Errorf("committing: %w", err)
				}
				fmt.Printf("Restored knowledge/%s.md\n", filename)
			} else {
				if _, err := os.Stat(activePath); err != nil {
					return fmt.Errorf("knowledge doc %q not found", filename)
				}
				if err := os.MkdirAll(archiveDir, 0o755); err != nil {
					return err
				}
				if err := os.Rename(activePath, archivedPath); err != nil {
					return err
				}
				if err := git.CommitAll(dir, fmt.Sprintf("p: archive knowledge/%s", filename)); err != nil {
					return fmt.Errorf("committing: %w", err)
				}
				fmt.Printf("Archived knowledge/%s.md\n", filename)
			}
			return nil
		})
	},
}

func init() {
	knowledgeDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
	knowledgeArchiveCmd.Flags().Bool("restore", false, "Restore an archived doc")
	knowledgeListCmd.Flags().String("tag", "", "Filter by tag")
	knowledgeCreateFromTemplateCmd.Flags().String("template", "", "Template: decision-record, meeting-notes, runbook")
	knowledgeCreateFromTemplateCmd.Flags().String("tags", "", "Comma-separated tags")

	knowledgeCmd.AddCommand(knowledgeDeleteCmd)
	knowledgeCmd.AddCommand(knowledgeSearchCmd)
	knowledgeCmd.AddCommand(knowledgeListCmd)
	knowledgeCmd.AddCommand(knowledgeCreateFromTemplateCmd)
	knowledgeCmd.AddCommand(knowledgeArchiveCmd)
	rootCmd.AddCommand(knowledgeCmd)
}
