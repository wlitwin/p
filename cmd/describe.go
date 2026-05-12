package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/ai"
	"github.com/walter/p/internal/git"
	"github.com/walter/p/internal/project"
)

var describeCmd = &cobra.Command{
	Use:   "describe <project> [description...]",
	Short: "Set or update a project's description",
	Long: `Set the description for a project. The description is shown in
'p list' and 'p status' output.

Use --auto to generate a description from project contents using AI.

Examples:
  p project describe serviceA New payments processing service
  p project describe serviceA ""   # clear the description
  p project describe serviceA --auto`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		auto, _ := cmd.Flags().GetBool("auto")

		if !auto && len(args) < 2 {
			return fmt.Errorf("provide a description or use --auto to generate one")
		}

		return withProjectLock(args[0], func(dir string) error {
			meta, err := project.LoadMeta(dir)
			if err != nil {
				return err
			}

			var description string
			if auto {
				fmt.Println("Generating description...")
				generated, err := generateDescription(args[0], dir)
				if err != nil {
					return fmt.Errorf("generating description: %w", err)
				}
				description = generated
			} else {
				description = strings.Join(args[1:], " ")
			}

			meta.Description = description

			if err := project.SaveMeta(dir, meta); err != nil {
				return err
			}

			if err := git.CommitAll(cmd.Context(), dir, fmt.Sprintf("p: update description for %s", args[0])); err != nil {
				return fmt.Errorf("committing: %w", err)
			}

			if meta.Description == "" {
				fmt.Printf("Cleared description for %s\n", args[0])
			} else {
				fmt.Printf("Description for %s: %s\n", args[0], meta.Description)
			}
			return nil
		})
	},
}

func generateDescription(projectName, projectDir string) (string, error) {
	data := ai.BuildTemplateData(projectName, projectDir, "describe", "", "", nil)

	var sb strings.Builder
	sb.WriteString("Based on the project contents below, write a single concise sentence (under 80 characters) describing this project. ")
	sb.WriteString("Output ONLY the description, no quotes, no punctuation at the end, no preamble.\n\n")

	if data.ProjectDescription != "" {
		fmt.Fprintf(&sb, "Current description: %s\n\n", data.ProjectDescription)
	}
	if data.TodoLists != "" {
		fmt.Fprintf(&sb, "Todo lists:\n%s\n", data.TodoLists)
	}
	if data.KnowledgeDocs != "" {
		fmt.Fprintf(&sb, "Knowledge docs:\n%s\n", data.KnowledgeDocs)
	}

	claudePath := cfg.ClaudePath
	if claudePath == "" {
		claudePath = "claude"
	}
	model := cfg.ClaudeModel
	if model == "" {
		model = "claude-opus-4-6"
	}

	claudeArgs := []string{
		"--print",
		"--no-session-persistence",
		"--model", model,
		"-p", sb.String(),
	}

	claudeCmd := exec.Command(claudePath, claudeArgs...)
	claudeCmd.Stderr = claudeStderr()

	out, err := claudeCmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func init() {
	describeCmd.Flags().Bool("auto", false, "Generate description from project contents using AI")
	projectCmd.AddCommand(describeCmd)
}
