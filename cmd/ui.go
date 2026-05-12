package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/tui"
)

var uiCmd = &cobra.Command{
	Use:   "ui [project] [list]",
	Short: "Launch interactive TUI mode",
	Long: `Launch a full-screen terminal interface for browsing and managing
projects, todo lists, and items.

Navigate with arrow keys or j/k, select with Enter, go back with Esc.
Press ? for full keybinding help.`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		app := tui.NewApp(cfg)

		switch len(args) {
		case 0:
			app.StartAtProjectList()
		case 1:
			dir, err := project.Resolve(cfg.ProjectRoot, args[0])
			if err != nil {
				return fmt.Errorf("resolving project: %w", err)
			}
			app.StartAtProject(args[0], dir)
		case 2:
			dir, err := project.Resolve(cfg.ProjectRoot, args[0])
			if err != nil {
				return fmt.Errorf("resolving project: %w", err)
			}
			app.StartAtList(args[0], dir, args[1])
		}

		p := tea.NewProgram(app, tea.WithAltScreen())
		_, err := p.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
