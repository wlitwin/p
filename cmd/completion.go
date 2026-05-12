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
	// Top-level commands that take a project name as first arg
	for _, cmd := range []*cobra.Command{
		addCmd, askCmd, planCmd, doCmd, showCmd, saveCmd, searchCmd,
	} {
		cmd.ValidArgsFunction = projectCompletionFunc
	}

	// p project subcommands that take a project name
	for _, cmd := range []*cobra.Command{
		archiveCmd, unarchiveCmd, logCmd, diffCmd, revertCmd,
		describeCmd, setCmd,
	} {
		cmd.ValidArgsFunction = projectCompletionFunc
	}

	// p ai subcommands that take a project name
	for _, cmd := range []*cobra.Command{summarizeCmd, reviewCmd} {
		cmd.ValidArgsFunction = projectCompletionFunc
	}

	// p knowledge subcommands that take a project name
	for _, cmd := range []*cobra.Command{
		knowledgeDeleteCmd, knowledgeSearchCmd, knowledgeListCmd,
		knowledgeCreateFromTemplateCmd, knowledgeArchiveCmd,
	} {
		cmd.ValidArgsFunction = projectCompletionFunc
	}

	// Commands that take project + list (list completion covers both)
	for _, cmd := range []*cobra.Command{
		agentCmd, moveCmd, rmListCmd, archiveListCmd,
		priorityCmd, dueCmd, tagCmd,
	} {
		cmd.ValidArgsFunction = listCompletionFunc
	}

	// State commands created via makeStateCmd — find by name
	if sub, _, err := rootCmd.Find([]string{"done"}); err == nil {
		sub.ValidArgsFunction = listCompletionFunc
	}
	for _, name := range []string{"block", "open"} {
		if sub, _, err := todoCmd.Find([]string{name}); err == nil {
			sub.ValidArgsFunction = listCompletionFunc
		}
	}
}
