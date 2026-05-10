package cmd

import (
	"github.com/spf13/cobra"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

func projectCompletionFunc(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	if cfg.ProjectRoot == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	projects, err := project.List(cfg.ProjectRoot, false)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, p := range projects {
		names = append(names, p.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func listCompletionFunc(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if cfg.ProjectRoot == "" || len(args) < 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	dir, err := project.Resolve(cfg.ProjectRoot, args[0])
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names, err := todo.ListNames(dir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	// Register completions for commands that take project/list args
	for _, cmd := range []*cobra.Command{
		addCmd, archiveCmd, unarchiveCmd, logCmd, diffCmd,
		revertCmd, askCmd, planCmd, summarizeCmd, reviewCmd,
	} {
		cmd.ValidArgsFunction = projectCompletionFunc
	}

	for _, cmd := range []*cobra.Command{moveCmd, rmListCmd} {
		cmd.ValidArgsFunction = listCompletionFunc
	}

	// State commands use list completion for the second arg
	for _, name := range []string{"done", "block", "open"} {
		if sub, _, err := rootCmd.Find([]string{name}); err == nil {
			sub.ValidArgsFunction = listCompletionFunc
		}
	}
}
