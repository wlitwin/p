package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/service"
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
			resultTags, err := service.SetItemTags(cmd.Context(), dir, args[1], args[2], tags, remove)
			if err != nil {
				return err
			}

			action := "tagged"
			if remove {
				action = "untagged"
			}
			commitMsg := fmt.Sprintf("p: %s %s #%s with %s", action, args[1], args[2], strings.Join(tags, ","))
			if err := service.Commit(cmd.Context(), dir, commitMsg); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			fmt.Printf("Tags %s on %s #%s: %s\n", action, args[1], args[2], strings.Join(resultTags, ", "))
			return nil
		})
	},
}

func init() {
	tagCmd.Flags().Bool("remove", false, "Remove the specified tags instead of adding")
	todoCmd.AddCommand(tagCmd)
}
