package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
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

		query = strings.ToLower(query)

		if projectName != "" {
			return searchProject(projectName, query)
		}

		projects, err := project.List(cfg.ProjectRoot, false)
		if err != nil {
			return err
		}
		for _, p := range projects {
			_ = searchProject(p.Name, query)
		}
		return nil
	},
}

func searchProject(name, query string) error {
	dir, err := project.Resolve(cfg.ProjectRoot, name)
	if err != nil {
		return err
	}

	headerPrinted := false
	printHeader := func() {
		if !headerPrinted {
			fmt.Printf("%s\n", tui.Bold.Render(name))
			headerPrinted = true
		}
	}

	// Search todos
	lists, _ := todo.ListNames(dir)
	for _, listName := range lists {
		list, err := todo.LoadList(dir, listName)
		if err != nil {
			continue
		}
		searchItems(list.Items, name, listName, "", 1, query, printHeader)
	}

	// Search knowledge
	files, _ := knowledge.ListFiles(dir)
	for _, f := range files {
		content, err := knowledge.Read(dir, f)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(content), query) {
			printHeader()
			path := filepath.Join("knowledge", f+".md")
			fmt.Printf("  %s %s\n", tui.Cyan.Render(path), matchContext(content, query))
		}
	}

	if headerPrinted {
		fmt.Println()
	}
	return nil
}

func searchItems(items []*todo.Item, projectName, listName, prefix string, start int, query string, printHeader func()) {
	for i, item := range items {
		id := fmt.Sprintf("%s%d", prefix, start+i)
		if strings.Contains(strings.ToLower(item.Text), query) {
			printHeader()
			marker := "[ ]"
			switch item.State {
			case todo.Done:
				marker = "[x]"
			case todo.Blocked:
				marker = "[-]"
			}
			fmt.Printf("  %s %s %s %s\n",
				tui.Dim.Render(listName+"#"+id),
				tui.StateStyle(marker),
				item.Text,
				"",
			)
		}
		if len(item.Children) > 0 {
			searchItems(item.Children, projectName, listName, id+".", 1, query, printHeader)
		}
	}
}

func matchContext(content, query string) string {
	runes := []rune(content)
	lowerRunes := []rune(strings.ToLower(content))
	queryRunes := []rune(strings.ToLower(query))

	idx := -1
	for i := 0; i <= len(lowerRunes)-len(queryRunes); i++ {
		if string(lowerRunes[i:i+len(queryRunes)]) == string(queryRunes) {
			idx = i
			break
		}
	}
	if idx == -1 {
		return ""
	}

	start := max(idx-30, 0)
	end := min(idx+len(queryRunes)+30, len(runes))

	snippet := strings.ReplaceAll(string(runes[start:end]), "\n", " ")
	snippet = strings.TrimSpace(snippet)

	prefix := ""
	if start > 0 {
		prefix = "..."
	}
	suffix := ""
	if end < len(runes) {
		suffix = "..."
	}

	return tui.Dim.Render(prefix + snippet + suffix)
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
