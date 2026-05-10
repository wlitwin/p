package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

var statusCmd = &cobra.Command{
	Use:   "status [project]",
	Short: "Show project status overview",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		if len(args) == 1 {
			return projectStatus(args[0])
		}
		return overallStatus()
	},
}

func overallStatus() error {
	projects, err := project.List(cfg.ProjectRoot, false)
	if err != nil {
		return err
	}

	if len(projects) == 0 {
		fmt.Println("No projects. Create one with: p new <name>")
		return nil
	}

	for _, p := range projects {
		dir, err := project.Resolve(cfg.ProjectRoot, p.Name)
		if err != nil {
			continue
		}

		names, err := todo.ListNames(dir)
		if err != nil {
			continue
		}

		totalOpen, totalBlocked, totalDone := 0, 0, 0
		for _, name := range names {
			list, err := todo.LoadList(dir, name)
			if err != nil {
				continue
			}
			o, d, b := countStates(list.Items)
			totalOpen += o
			totalDone += d
			totalBlocked += b
		}

		desc := ""
		if p.Description != "" {
			desc = " — " + p.Description
		}
		fmt.Printf("  %-20s open=%-3d blocked=%-3d done=%-3d%s\n",
			p.Name, totalOpen, totalBlocked, totalDone, desc)
	}
	return nil
}

func projectStatus(name string) error {
	dir, err := project.Resolve(cfg.ProjectRoot, name)
	if err != nil {
		return err
	}

	meta, err := project.LoadMeta(dir)
	if err != nil {
		return err
	}

	fmt.Printf("Project: %s\n", meta.Name)
	if meta.Description != "" {
		fmt.Printf("Description: %s\n", meta.Description)
	}
	fmt.Printf("Created: %s\n", meta.Created.Format("2006-01-02"))
	if meta.Archived {
		fmt.Println("Status: ARCHIVED")
	}
	fmt.Println()

	names, err := todo.ListNames(dir)
	if err != nil {
		return err
	}

	if len(names) == 0 {
		fmt.Println("No todo lists.")
	} else {
		fmt.Println("Todo lists:")
		for _, n := range names {
			list, err := todo.LoadList(dir, n)
			if err != nil {
				fmt.Printf("  %-20s (error loading)\n", n)
				continue
			}
			open, done, blocked := countStates(list.Items)
			fmt.Printf("  %-20s open=%-3d blocked=%-3d done=%-3d\n", n, open, blocked, done)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
