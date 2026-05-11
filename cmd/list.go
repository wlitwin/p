package cmd

import (
	"fmt"

	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/display"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/tui"
)

var listCmd = &cobra.Command{
	Use:   "list [project] [list]",
	Short: "List projects, todo lists, or items",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		switch len(args) {
		case 0:
			return listProjects(cmd)
		case 1:
			return listTodoLists(args[0])
		case 2:
			if args[1] == "all" || args[1] == "*" {
				return listAllItems(args[0], cmd)
			}
			return listItems(args[0], args[1], cmd)
		}
		return nil
	},
}

func listProjects(cmd *cobra.Command) error {
	all, _ := cmd.Flags().GetBool("all")
	projects, err := project.List(cfg.ProjectRoot, all)
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		fmt.Println("No projects found. Create one with: p new <name>")
		return nil
	}
	fmt.Printf("%s\n\n", tui.Dim.Render(cfg.ProjectRoot))
	for _, p := range projects {
		dir, _ := project.Resolve(cfg.ProjectRoot, p.Name)

		status := ""
		if p.Archived {
			status = tui.Dim.Render(" (archived)")
		}
		desc := ""
		if p.Description != "" {
			desc = " — " + p.Description
		}

		created := tui.Dim.Render("created=" + p.Created.Format("2006-01-02"))
		lastMod := lastCommitDate(dir)
		updated := ""
		if lastMod != "" {
			updated = " " + tui.Dim.Render("updated="+lastMod)
		}

		fmt.Printf("  %s%s%s  %s%s\n", tui.Bold.Render(p.Name), desc, status, created, updated)
	}
	return nil
}

func lastCommitDate(dir string) string {
	cmd := exec.Command("git", "log", "-1", "--format=%cs")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func listTodoLists(projectName string) error {
	dir, err := project.Resolve(cfg.ProjectRoot, projectName)
	if err != nil {
		return err
	}

	names, err := todo.ListNames(dir)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		fmt.Printf("No todo lists in %s. Add one with: p add %s \"task text\"\n", projectName, projectName)
		return nil
	}

	fmt.Println(tui.Bold.Render("Todo lists:"))
	for _, name := range names {
		list, err := todo.LoadList(dir, name)
		if err != nil {
			fmt.Printf("  %s (error loading)\n", name)
			continue
		}
		open, done, blocked := todo.CountStates(list.Items)
		fmt.Printf("  %-20s  open=%-3d blocked=%-3d done=%-3d\n", name, open, blocked, done)
	}

	archived, _ := todo.ArchivedListNames(dir)
	if len(archived) > 0 {
		fmt.Printf("\n%s\n", tui.Dim.Render(fmt.Sprintf("  + %d archived list(s) — use p archive-list --restore to restore", len(archived))))
	}

	// Also show knowledge docs
	kFiles, _ := knowledge.ListFiles(dir)
	if len(kFiles) > 0 {
		fmt.Println()
		fmt.Println(tui.Bold.Render("Knowledge docs:"))
		for _, f := range kFiles {
			path := knowledge.FilePath(dir, f)
			info, err := os.Stat(path)
			size := ""
			if err == nil {
				size = tui.Dim.Render(fmt.Sprintf("(%d bytes)", info.Size()))
			}
			fmt.Printf("  %-20s  %s\n", f, size)
		}
	}
	return nil
}

func listAllItems(projectName string, cmd *cobra.Command) error {
	dir, err := project.Resolve(cfg.ProjectRoot, projectName)
	if err != nil {
		return err
	}

	names, err := todo.ListNames(dir)
	if err != nil {
		return err
	}

	stateFilter, _ := cmd.Flags().GetString("state")
	priorityFilter, _ := cmd.Flags().GetString("priority")
	tagFilter, _ := cmd.Flags().GetString("tag")

	for i, name := range names {
		list, err := todo.LoadList(dir, name)
		if err != nil {
			continue
		}

		filtered := display.FilterItems(list.Items, stateFilter, priorityFilter, tagFilter)
		if len(filtered) == 0 {
			continue
		}

		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("%s\n\n", tui.Bold.Render("# "+name))
		display.PrintItems(filtered, "", 1, dir)
	}
	return nil
}

func listItems(projectName, listName string, cmd *cobra.Command) error {
	dir, err := project.Resolve(cfg.ProjectRoot, projectName)
	if err != nil {
		return err
	}

	list, err := todo.LoadList(dir, listName)
	if err != nil {
		return err
	}

	stateFilter, _ := cmd.Flags().GetString("state")
	priorityFilter, _ := cmd.Flags().GetString("priority")
	tagFilter, _ := cmd.Flags().GetString("tag")

	filtered := display.FilterItems(list.Items, stateFilter, priorityFilter, tagFilter)

	fmt.Printf("# %s\n\n", list.Title)
	display.PrintItems(filtered, "", 1, dir)
	return nil
}

func init() {
	listCmd.Flags().Bool("all", false, "Include archived projects")
	listCmd.Flags().String("state", "", "Filter by state: open, blocked, done")
	listCmd.Flags().String("priority", "", "Filter by priority: now, backlog")
	listCmd.Flags().String("tag", "", "Filter by tag")
	rootCmd.AddCommand(listCmd)
}
