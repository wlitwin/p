package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/todo"
)

var tagCmd = &cobra.Command{
	Use:   "tag <project> <list> <item-id> <tags...>",
	Short: "Add or remove tags on a todo item",
	Long: `Add tags to a todo item. Use --remove to remove tags instead.

Examples:
  p tag serviceA tasks 1 bug frontend
  p tag serviceA tasks 1 --remove bug`,
	Args: cobra.MinimumNArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		remove, _ := cmd.Flags().GetBool("remove")
		tags := args[3:]

		return withProjectLock(args[0], func(dir string) error {
			list, err := todo.LoadList(dir, args[1])
			if err != nil {
				return err
			}

			item, err := todo.ResolveItem(list, args[2])
			if err != nil {
				return err
			}

			if remove {
				item.Tags = removeTags(item.Tags, tags)
			} else {
				item.Tags = addTags(item.Tags, tags)
			}

			if err := todo.SaveList(dir, args[1], list); err != nil {
				return err
			}

			action := "tagged"
			if remove {
				action = "untagged"
			}
			commitMsg := fmt.Sprintf("p: %s %s #%s with %s", action, args[1], args[2], strings.Join(tags, ","))
			if err := git.CommitAll(dir, commitMsg); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Printf("Tags %s on %s #%s: %s\n", action, args[1], args[2], strings.Join(item.Tags, ", "))
			return nil
		})
	},
}

func addTags(existing, toAdd []string) []string {
	set := make(map[string]bool)
	for _, t := range existing {
		set[t] = true
	}
	for _, t := range toAdd {
		if !set[t] {
			existing = append(existing, t)
			set[t] = true
		}
	}
	return existing
}

func removeTags(existing, toRemove []string) []string {
	remove := make(map[string]bool)
	for _, t := range toRemove {
		remove[t] = true
	}
	var result []string
	for _, t := range existing {
		if !remove[t] {
			result = append(result, t)
		}
	}
	return result
}

func init() {
	tagCmd.Flags().Bool("remove", false, "Remove the specified tags instead of adding")
	rootCmd.AddCommand(tagCmd)
}
