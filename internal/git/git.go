// Package git provides helpers for managing git repositories used as
// project storage backends. All public functions accept a context.Context
// to support cancellation and deadline propagation.
package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Init initialises a new git repository in dir if one does not already exist.
func Init(ctx context.Context, dir string) error {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		ensureGitignore(dir)
		return nil
	}
	if err := run(ctx, dir, "init"); err != nil {
		return err
	}
	ensureGitignore(dir)
	return nil
}

func ensureGitignore(dir string) {
	path := filepath.Join(dir, ".gitignore")
	entry := ".p/lock"
	data, err := os.ReadFile(path)
	if err == nil {
		s := string(data)
		if strings.Contains(s, entry) {
			return
		}
		if !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		_ = os.WriteFile(path, []byte(s+entry+"\n"), 0o644)
		return
	}
	_ = os.WriteFile(path, []byte(entry+"\n"), 0o644)
}

// CommitAll stages all changes and creates a commit with the given message.
// If there are no changes to commit, it returns nil without creating a commit.
func CommitAll(ctx context.Context, dir, message string) error {
	if err := run(ctx, dir, "add", "-A"); err != nil {
		return err
	}

	out, err := output(ctx, dir, "status", "--porcelain")
	if err != nil {
		return err
	}
	if len(out) == 0 {
		return nil
	}

	return run(ctx, dir, "commit", "-m", message)
}

// Diff stages all changes and returns the unified diff of staged content.
func Diff(ctx context.Context, dir string) (string, error) {
	// Stage everything first so we can show a unified diff
	if err := run(ctx, dir, "add", "-A"); err != nil {
		return "", err
	}
	staged, err := output(ctx, dir, "diff", "--cached")
	if err != nil {
		// If no HEAD yet (first commit scenario), diff against empty tree
		staged, err = output(ctx, dir, "diff", "--cached", "--diff-filter=A")
		if err != nil {
			return "", err
		}
	}
	// Unstage so the user can still choose to revert
	_ = runQuiet(ctx, dir, "reset", "HEAD")
	return staged, nil
}

// DiffStat returns a summary of changes relative to HEAD.
func DiffStat(ctx context.Context, dir string) (string, error) {
	return output(ctx, dir, "diff", "--stat", "HEAD")
}

// RevertChanges discards all uncommitted changes in the working tree.
func RevertChanges(ctx context.Context, dir string) error {
	if err := run(ctx, dir, "checkout", "."); err != nil {
		return err
	}
	return run(ctx, dir, "clean", "-fd")
}

func run(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %v: %w", args, err)
	}
	return nil
}

func runQuiet(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %v: %w", args, err)
	}
	return nil
}

func output(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %v: %w", args, err)
	}
	return string(out), nil
}
