package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/knowledge"
	"github.com/walter/p/internal/project"
	"github.com/walter/p/internal/todo"
)

var editOpenCmd = &cobra.Command{
	Use:   "open <project> <name>",
	Short: "Open a todo list or knowledge doc in $EDITOR",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireProjectRoot(); err != nil {
			return err
		}

		dir, err := project.Resolve(cfg.ProjectRoot, args[0])
		if err != nil {
			return err
		}

		isKnowledge, _ := cmd.Flags().GetBool("knowledge")

		var path string
		if isKnowledge {
			path = knowledge.FilePath(dir, args[1])
		} else {
			path = todo.ListPath(dir, args[1])
			if _, err := os.Stat(path); err != nil {
				// Try knowledge
				kpath := knowledge.FilePath(dir, args[1])
				if _, err := os.Stat(kpath); err == nil {
					path = kpath
				}
			}
		}

		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("file not found: %s", path)
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		editorCmd := exec.Command(editor, path)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr
		return editorCmd.Run()
	},
}

func init() {
	editOpenCmd.Flags().BoolP("knowledge", "k", false, "Open a knowledge doc")
	editCmd.AddCommand(editOpenCmd)
}
