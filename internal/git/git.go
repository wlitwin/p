package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Init(dir string) error {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		ensureGitignore(dir)
		return nil
	}
	if err := run(dir, "init"); err != nil {
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

func CommitAll(dir, message string) error {
	if err := run(dir, "add", "-A"); err != nil {
		return err
	}

	out, err := output(dir, "status", "--porcelain")
	if err != nil {
		return err
	}
	if len(out) == 0 {
		return nil
	}

	return run(dir, "commit", "-m", message)
}

func Diff(dir string) (string, error) {
	// Stage everything first so we can show a unified diff
	if err := run(dir, "add", "-A"); err != nil {
		return "", err
	}
	staged, err := output(dir, "diff", "--cached")
	if err != nil {
		// If no HEAD yet (first commit scenario), diff against empty tree
		staged, err = output(dir, "diff", "--cached", "--diff-filter=A")
		if err != nil {
			return "", err
		}
	}
	// Unstage so the user can still choose to revert
	_ = runQuiet(dir, "reset", "HEAD")
	return staged, nil
}

func DiffStat(dir string) (string, error) {
	return output(dir, "diff", "--stat", "HEAD")
}

func RevertChanges(dir string) error {
	if err := run(dir, "checkout", "."); err != nil {
		return err
	}
	return run(dir, "clean", "-fd")
}

func run(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %v: %w", args, err)
	}
	return nil
}

func runQuiet(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %v: %w", args, err)
	}
	return nil
}

func output(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %v: %w", args, err)
	}
	return string(out), nil
}
