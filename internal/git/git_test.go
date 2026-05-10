package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// configureGit sets user.email and user.name in the given repo so commits work
// even without global git config.
func configureGit(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "test"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
}

// initRepo calls Init and configures git user info for testing.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	configureGit(t, dir)
	return dir
}

// gitOutput runs a git command in dir and returns trimmed stdout.
func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(out))
}

func TestInit(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		t.Fatalf(".git directory does not exist after Init: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf(".git is not a directory")
	}
}

func TestInitIdempotent(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("first Init: %v", err)
	}
	if err := Init(dir); err != nil {
		t.Fatalf("second Init: %v", err)
	}

	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		t.Fatalf(".git directory does not exist after double Init: %v", err)
	}
}

func TestCommitAll(t *testing.T) {
	dir := initRepo(t)

	// Create a file to commit.
	testFile := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(testFile, []byte("hello world\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := CommitAll(dir, "add hello"); err != nil {
		t.Fatalf("CommitAll: %v", err)
	}

	// Verify the working tree is clean.
	status := gitOutput(t, dir, "status", "--porcelain")
	if status != "" {
		t.Fatalf("expected clean working tree, got:\n%s", status)
	}

	// Verify the commit message.
	msg := gitOutput(t, dir, "log", "-1", "--format=%s")
	if msg != "add hello" {
		t.Fatalf("expected commit message %q, got %q", "add hello", msg)
	}
}

func TestCommitAllNoChanges(t *testing.T) {
	dir := initRepo(t)

	// Make an initial commit so the repo is not empty.
	testFile := filepath.Join(dir, "init.txt")
	if err := os.WriteFile(testFile, []byte("init\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := CommitAll(dir, "initial"); err != nil {
		t.Fatalf("CommitAll (initial): %v", err)
	}

	// Commit again with no changes — should be a no-op and not error.
	if err := CommitAll(dir, "no changes"); err != nil {
		t.Fatalf("CommitAll (no changes) returned error: %v", err)
	}

	// Verify no new commit was created.
	count := gitOutput(t, dir, "rev-list", "--count", "HEAD")
	if count != "1" {
		t.Fatalf("expected 1 commit, got %s", count)
	}
}

func TestDiff(t *testing.T) {
	dir := initRepo(t)

	// Create a file so there's something to diff.
	testFile := filepath.Join(dir, "data.txt")
	if err := os.WriteFile(testFile, []byte("some content\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	diff, err := Diff(dir)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}

	if !strings.Contains(diff, "some content") {
		t.Fatalf("expected diff to contain 'some content', got:\n%s", diff)
	}
	if !strings.Contains(diff, "data.txt") {
		t.Fatalf("expected diff to mention 'data.txt', got:\n%s", diff)
	}

	// In a fresh repo with no HEAD, git reset HEAD is a no-op, so the file
	// may remain staged. We only verify the diff content was correct above.
}

func TestDiffWithExistingCommits(t *testing.T) {
	dir := initRepo(t)

	// Create initial file and commit.
	testFile := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(testFile, []byte("line one\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := CommitAll(dir, "initial"); err != nil {
		t.Fatalf("CommitAll: %v", err)
	}

	// Modify the file.
	if err := os.WriteFile(testFile, []byte("line one\nline two\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	diff, err := Diff(dir)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}

	if !strings.Contains(diff, "line two") {
		t.Fatalf("expected diff to contain 'line two', got:\n%s", diff)
	}
}

func TestDiffStat(t *testing.T) {
	dir := initRepo(t)

	// Create initial commit.
	testFile := filepath.Join(dir, "stats.txt")
	if err := os.WriteFile(testFile, []byte("original\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := CommitAll(dir, "initial"); err != nil {
		t.Fatalf("CommitAll: %v", err)
	}

	// Modify the file (unstaged change).
	if err := os.WriteFile(testFile, []byte("original\nmodified\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	stat, err := DiffStat(dir)
	if err != nil {
		t.Fatalf("DiffStat: %v", err)
	}

	if !strings.Contains(stat, "stats.txt") {
		t.Fatalf("expected diff stat to mention 'stats.txt', got:\n%s", stat)
	}
	// The stat line should contain insertion/deletion indicators.
	if !strings.Contains(stat, "+") {
		t.Fatalf("expected diff stat to contain '+' for insertions, got:\n%s", stat)
	}
}

func TestRevertChanges(t *testing.T) {
	dir := initRepo(t)

	// Create an initial commit so checkout has a base to revert to.
	trackedFile := filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(trackedFile, []byte("keep me\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := CommitAll(dir, "initial"); err != nil {
		t.Fatalf("CommitAll: %v", err)
	}

	// Modify the tracked file.
	if err := os.WriteFile(trackedFile, []byte("modified\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Create an untracked file.
	untrackedFile := filepath.Join(dir, "untracked.txt")
	if err := os.WriteFile(untrackedFile, []byte("should be removed\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := RevertChanges(dir); err != nil {
		t.Fatalf("RevertChanges: %v", err)
	}

	// Verify the tracked file is restored to its committed content.
	data, err := os.ReadFile(trackedFile)
	if err != nil {
		t.Fatalf("ReadFile (tracked): %v", err)
	}
	if string(data) != "keep me\n" {
		t.Fatalf("expected tracked file to be reverted, got %q", string(data))
	}

	// Verify the untracked file was cleaned up.
	if _, err := os.Stat(untrackedFile); !os.IsNotExist(err) {
		t.Fatalf("expected untracked file to be removed, but it still exists")
	}

	// Verify the working tree is clean.
	status := gitOutput(t, dir, "status", "--porcelain")
	if status != "" {
		t.Fatalf("expected clean working tree after revert, got:\n%s", status)
	}
}
