package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var aliasesCmd = &cobra.Command{
	Use:   "aliases",
	Short: "Print shell aliases for common p commands",
	Long: `Output shell alias definitions for quick access to common commands.

Add to your shell profile:
  # bash (~/.bashrc)
  eval "$(p aliases bash)"

  # zsh (~/.zshrc)
  eval "$(p aliases zsh)"

  # fish (~/.config/fish/config.fish)
  p aliases fish | source

Aliases provided:
  pa   → p add            pl   → p list
  pd   → p done           pst  → p status
  psh  → p show           psr  → p search
  pdo  → p do             ppn  → p plan
  psk  → p ask            psv  → p save
  pag  → p agent`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := detectShell()
		if len(args) == 1 {
			shell = args[0]
		}

		switch shell {
		case "bash", "zsh":
			fmt.Print(bashAliases())
		case "fish":
			fmt.Print(fishAliases())
		default:
			return fmt.Errorf("unsupported shell %q — use bash, zsh, or fish", shell)
		}
		return nil
	},
	ValidArgs: []string{"bash", "zsh", "fish"},
}

type shellAlias struct {
	short string
	full  string
}

var aliases = []shellAlias{
	{"pa", "p add"},
	{"pl", "p list"},
	{"pd", "p done"},
	{"pst", "p status"},
	{"psh", "p show"},
	{"psr", "p search"},
	{"pdo", "p do"},
	{"ppn", "p plan"},
	{"psk", "p ask"},
	{"psv", "p save"},
	{"pag", "p agent"},
}

func bashAliases() string {
	var sb strings.Builder
	for _, a := range aliases {
		fmt.Fprintf(&sb, "alias %s='%s'\n", a.short, a.full)
	}
	return sb.String()
}

func fishAliases() string {
	var sb strings.Builder
	for _, a := range aliases {
		fmt.Fprintf(&sb, "abbr -a %s '%s'\n", a.short, a.full)
	}
	return sb.String()
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "bash"
	}
	base := filepath.Base(shell)
	switch base {
	case "zsh":
		return "zsh"
	case "fish":
		return "fish"
	default:
		return "bash"
	}
}

func init() {
	rootCmd.AddCommand(aliasesCmd)
}
