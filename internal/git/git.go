package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Init(dir string) error {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return nil
	}
	return run(dir, "init")
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
	staged, err := output(dir, "diff", "--cached")
	if err != nil {
		return "", err
	}
	unstaged, err := output(dir, "diff")
	if err != nil {
		return "", err
	}
	untracked, err := output(dir, "status", "--porcelain")
	if err != nil {
		return "", err
	}
	return staged + unstaged + untracked, nil
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
