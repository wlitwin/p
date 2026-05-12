package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/display"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
	"github.com/walter/p/internal/tui"
)

var dueViewCmd = &cobra.Command{
	Use:   "due <project> [range]",
	Short: "Show items due today, overdue, or within a date range",
	Long: `Cross-list view of items with due dates. Shows items due today and
overdue by default. Accepts an optional range argument.

Ranges:
  today     Items due today (default, along with overdue)
  overdue   Only overdue items (past due date, still open)
  week      Items due within the next 7 days
  month     Items due within the next 30 days
  YYYY-MM-DD  Items due on a specific date

Examples:
  p due myproject              # today + overdue
  p due myproject week         # due within 7 days
  p due myproject overdue      # only overdue items
  p due myproject 2026-05-20   # due on specific date`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		projectName := args[0]
		dueRange := ""
		if len(args) >= 2 {
			dueRange = args[1]
		}

		dir, err := project.Resolve(cfg.ProjectRoot, projectName)
		if err != nil {
			return err
		}

		names, err := todo.ListNames(dir)
		if err != nil {
			return err
		}

		now := time.Now()
		first := true

		// Default: show both today and overdue
		if dueRange == "" {
			for _, name := range names {
				list, err := todo.LoadList(dir, name)
				if err != nil {
					continue
				}

				overdue := display.DueFilter(list.Items, "overdue", now)
				today := display.DueFilter(list.Items, "today", now)

				// Merge, avoiding duplicates (overdue items might also be "today" items)
				seen := make(map[string]bool)
				var combined []display.FilteredItem
				for _, fi := range overdue {
					seen[fi.OriginalID] = true
					combined = append(combined, fi)
				}
				for _, fi := range today {
					if !seen[fi.OriginalID] {
						combined = append(combined, fi)
					}
				}

				if len(combined) == 0 {
					continue
				}

				if !first {
					fmt.Println()
				}
				first = false
				fmt.Printf("%s\n\n", tui.Bold.Render("# "+name))
				display.PrintFilteredItems(combined, dir)
			}
		} else {
			for _, name := range names {
				list, err := todo.LoadList(dir, name)
				if err != nil {
					continue
				}

				filtered := display.DueFilter(list.Items, dueRange, now)
				if len(filtered) == 0 {
					continue
				}

				if !first {
					fmt.Println()
				}
				first = false
				fmt.Printf("%s\n\n", tui.Bold.Render("# "+name))
				display.PrintFilteredItems(filtered, dir)
			}
		}

		if first {
			fmt.Println("No items match the due date filter.")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dueViewCmd)
}
