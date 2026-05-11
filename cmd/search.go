package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/display"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/service"
	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/tui"
)

var searchCmd = &cobra.Command{
	Use:   "search [project] <query>",
	Short: "Search across todos and knowledge docs",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		var projectName, query string
		if len(args) == 2 {
			projectName = args[0]
			query = args[1]
		} else {
			query = args[0]
		}

		queryLower := strings.ToLower(query)

		ctx := cmd.Context()
		if projectName != "" {
			return searchProject(ctx, projectName, queryLower)
		}

		projects, err := project.List(cfg.ProjectRoot, false)
		if err != nil {
			return err
		}
		for _, p := range projects {
			_ = searchProject(ctx, p.Name, queryLower)
		}
		return nil
	},
}

func searchProject(ctx context.Context, name, queryLower string) error {
	dir, err := project.Resolve(cfg.ProjectRoot, name)
	if err != nil {
		return err
	}

	matches := service.SearchProject(ctx, dir, name, queryLower)
	if len(matches) == 0 {
		return nil
	}

	fmt.Printf("%s\n", tui.Bold.Render(name))

	for _, m := range matches {
		if m.Type == "todo" {
			for _, r := range m.TodoResults {
				marker := todo.StateMarker(r.Item.State)
				fmt.Printf("  %s %s %s\n",
					tui.Dim.Render(r.ListName+"#"+r.ItemID),
					tui.StateStyle(marker),
					r.Item.Text,
				)
			}
		} else {
			path := filepath.Join("knowledge", m.File+".md")
			content, _ := knowledge.Read(dir, m.File)
			fmt.Printf("  %s %s\n", tui.Cyan.Render(path), display.MatchContext(content, queryLower))
		}
	}

	fmt.Println()
	return nil
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
