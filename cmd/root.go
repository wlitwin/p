package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/walter/p/internal/config"
	"github.com/walter/p/internal/lock"
	"github.com/walter/p/internal/project"
)

// resolveClaudeConfig returns the claude binary path and model,
// applying defaults from config.
func resolveClaudeConfig() (claudePath, model string) {
	claudePath = cfg.ClaudePath
	if claudePath == "" {
		claudePath = "claude"
	}
	model = cfg.ClaudeModel
	if model == "" {
		model = "claude-opus-4-6"
	}
	return
}

// resolvePBinary returns the path to the running p executable.
func resolvePBinary() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolving executable path: %w", err)
	}
	return p, nil
}

// resolveProjectDir validates config and resolves a project name to its directory.
func resolveProjectDir(projectName string) (string, error) {
	if err := requireProjectRoot(); err != nil {
		return "", err
	}
	return project.Resolve(cfg.ProjectRoot, projectName)
}

var cfg config.Config
var verbose bool

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

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
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Show claude subprocess stderr output")
	cobra.OnInitialize(loadConfig)
}

func claudeStderr() *os.File {
	if verbose {
		return os.Stderr
	}
	return nil
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

func withProjectLock(projectName string, fn func(dir string) error) error {
	if err := requireProjectRoot(); err != nil {
		return err
	}

	dir, err := project.Resolve(cfg.ProjectRoot, projectName)
	if err != nil {
		return err
	}

	lk, err := lock.Acquire(dir)
	if err != nil {
		return err
	}
	defer lk.Release()

	return fn(dir)
}
