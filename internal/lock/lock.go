package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Lock struct {
	path string
}

type Info struct {
	PID       int
	Timestamp time.Time
}

func lockPath(projectDir string) string {
	return filepath.Join(projectDir, ".p", "lock")
}

func Acquire(projectDir string) (*Lock, error) {
	path := lockPath(projectDir)

	if info, err := Read(projectDir); err == nil {
		if isProcessRunning(info.PID) {
			return nil, fmt.Errorf("project is locked by PID %d (since %s) — if this is stale, remove %s",
				info.PID, info.Timestamp.Format("15:04:05"), path)
		}
		// Stale lock — process is dead, clean it up
		os.Remove(path)
	}

	content := fmt.Sprintf("%d\n%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("creating lock: %w", err)
	}

	return &Lock{path: path}, nil
}

func (l *Lock) Release() {
	if l != nil {
		os.Remove(l.path)
	}
}

func Read(projectDir string) (*Info, error) {
	data, err := os.ReadFile(lockPath(projectDir))
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("malformed lock file")
	}

	pid, err := strconv.Atoi(lines[0])
	if err != nil {
		return nil, fmt.Errorf("malformed lock PID: %w", err)
	}

	ts, err := time.Parse(time.RFC3339, lines[1])
	if err != nil {
		return nil, fmt.Errorf("malformed lock timestamp: %w", err)
	}

	return &Info{PID: pid, Timestamp: ts}, nil
}

func isProcessRunning(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
