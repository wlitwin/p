package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "View or set configuration",
	Long: `View all config values, get a specific value, or set a value.

Examples:
  p config                           # show all config
  p config project_root              # show one value
  p config claude_model sonnet       # set a value`,
	Args: cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch len(args) {
		case 0:
			return showAllConfig()
		case 1:
			return showConfigKey(args[0])
		case 2:
			return setConfigKey(args[0], args[1])
		}
		return nil
	},
}

func showAllConfig() error {
	path, _ := config.ConfigPath()
	fmt.Printf("Config file: %s\n\n", path)
	fmt.Printf("  %-20s %s\n", "project_root", cfg.ProjectRoot)
	fmt.Printf("  %-20s %s\n", "claude_path", cfg.ClaudePath)
	fmt.Printf("  %-20s %s\n", "claude_model", cfg.ClaudeModel)
	fmt.Printf("  %-20s %s\n", "default_priority", cfg.DefaultPriority)
	return nil
}

func showConfigKey(key string) error {
	val, err := getConfigValue(key)
	if err != nil {
		return err
	}
	fmt.Println(val)
	return nil
}

func setConfigKey(key, value string) error {
	switch key {
	case "project_root":
		cfg.ProjectRoot = expandHome(value)
	case "claude_path":
		cfg.ClaudePath = value
	case "claude_model":
		cfg.ClaudeModel = value
	case "default_priority":
		cfg.DefaultPriority = value
	default:
		return fmt.Errorf("unknown config key: %s\nValid keys: project_root, claude_path, claude_model, default_priority", key)
	}

	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}

func getConfigValue(key string) (string, error) {
	switch key {
	case "project_root":
		return cfg.ProjectRoot, nil
	case "claude_path":
		return cfg.ClaudePath, nil
	case "claude_model":
		return cfg.ClaudeModel, nil
	case "default_priority":
		return cfg.DefaultPriority, nil
	default:
		return "", fmt.Errorf("unknown config key: %s\nValid keys: project_root, claude_path, claude_model, default_priority", key)
	}
}

func init() {
	rootCmd.AddCommand(configCmd)
}
