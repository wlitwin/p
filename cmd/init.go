package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/config"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up p — configure project root directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		current := cfg.ProjectRoot
		if current != "" {
			fmt.Printf("Current project root: %s\n", current)
		}

		fmt.Print("Project root directory: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		input = strings.TrimSpace(input)
		if input == "" && current != "" {
			input = current
		}
		if input == "" {
			return fmt.Errorf("project root is required")
		}

		expanded := expandHome(input)
		abs, err := filepath.Abs(expanded)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(abs, 0o755); err != nil {
			return fmt.Errorf("creating directory: %w", err)
		}

		cfg.ProjectRoot = abs
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		fmt.Printf("Project root set to: %s\n", abs)
		configPath, _ := config.ConfigPath()
		fmt.Printf("Config saved to: %s\n", configPath)
		return nil
	},
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func init() {
	rootCmd.AddCommand(initCmd)
}
