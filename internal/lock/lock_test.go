package lock

import (
	"os"
	"path/filepath"
	"testing"
)

func setupLockTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".p"), 0o755)
	return dir
}

func TestAcquireAndRelease(t *testing.T) {
	dir := setupLockTest(t)

	lk, err := Acquire(dir)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	info, err := Read(dir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if info.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", info.PID, os.Getpid())
	}

	lk.Release()

	if _, err := os.Stat(lockPath(dir)); !os.IsNotExist(err) {
		t.Error("lock file should be removed after release")
	}
}

func TestDoubleAcquireFails(t *testing.T) {
	dir := setupLockTest(t)

	lk, err := Acquire(dir)
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	defer lk.Release()

	_, err = Acquire(dir)
	if err == nil {
		t.Error("second Acquire should fail while locked")
	}
}

func TestStaleLockRecovery(t *testing.T) {
	dir := setupLockTest(t)

	// Write a lock with a non-existent PID
	content := "999999\n2026-01-01T00:00:00Z\n"
	os.WriteFile(lockPath(dir), []byte(content), 0o644)

	lk, err := Acquire(dir)
	if err != nil {
		t.Fatalf("Acquire should recover stale lock: %v", err)
	}
	lk.Release()
}
