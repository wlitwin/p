package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/config"
)

var cfg config.Config

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "p",
	Short:   "Project knowledge & task manager",
	Version: Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(loadConfig)
}

func loadConfig() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: loading config: %v\n", err)
	}
}

func requireProjectRoot() error {
	if cfg.ProjectRoot == "" {
		return fmt.Errorf("project root not configured — run `p init` first")
	}
	return nil
}
