// Package lock provides file-based mutual exclusion for project directories,
// with automatic stale lock detection via PID checking.
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

// Lock represents an acquired file-based project lock. Call Release when done.
type Lock struct {
	path string
}

// Info contains metadata about an existing lock: the owning PID and when
// the lock was acquired.
type Info struct {
	PID       int
	Timestamp time.Time
}

func lockPath(projectDir string) string {
	return filepath.Join(projectDir, ".p", "lock")
}

// Acquire attempts to create a project lock file atomically. If a lock file
// already exists, it checks whether the owning process is still running and
// breaks stale locks automatically. Returns an error if the lock is held by
// a live process.
func Acquire(projectDir string) (*Lock, error) {
	path := lockPath(projectDir)

	content := fmt.Sprintf("%d\n%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339))

	// Try atomic creation first (O_CREATE|O_EXCL fails if file exists)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err == nil {
		_, _ = f.WriteString(content)
		_ = f.Close()
		return &Lock{path: path}, nil
	}

	// File exists — check if it's a stale lock
	info, readErr := Read(projectDir)
	if readErr != nil {
		// Can't read lock file — remove and retry
		_ = os.Remove(path)
		return Acquire(projectDir)
	}

	if isProcessRunning(info.PID) {
		return nil, fmt.Errorf("project is locked by PID %d (since %s) — if this is stale, remove %s",
			info.PID, info.Timestamp.Format("15:04:05"), path)
	}

	// Stale lock — remove and retry atomically
	_ = os.Remove(path)
	f, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, fmt.Errorf("acquiring lock (race): %w", err)
	}
	_, _ = f.WriteString(content)
	_ = f.Close()
	return &Lock{path: path}, nil
}

func (l *Lock) Release() {
	if l != nil {
		_ = os.Remove(l.path)
	}
}

// Read parses an existing lock file and returns its metadata.
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
