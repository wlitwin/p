package lock

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
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

func TestConcurrentAcquire(t *testing.T) {
	dir := setupLockTest(t)

	var wg sync.WaitGroup
	var holding int64
	var maxHolding int64

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			lk, err := Acquire(dir)
			if err != nil {
				// Another goroutine holds the lock; that's expected.
				return
			}

			cur := atomic.AddInt64(&holding, 1)
			// Track the maximum concurrent holders we ever see.
			for {
				old := atomic.LoadInt64(&maxHolding)
				if cur <= old || atomic.CompareAndSwapInt64(&maxHolding, old, cur) {
					break
				}
			}

			// Simulate a brief critical section.
			// Use a busy-loop to avoid importing time.
			for j := 0; j < 1000; j++ {
				_ = j
			}

			atomic.AddInt64(&holding, -1)
			lk.Release()
		}()
	}

	wg.Wait()

	if m := atomic.LoadInt64(&maxHolding); m > 1 {
		t.Errorf("max concurrent holders = %d, want at most 1", m)
	}
}

func TestConcurrentAcquireRelease(t *testing.T) {
	dir := setupLockTest(t)

	for i := 0; i < 20; i++ {
		lk, err := Acquire(dir)
		if err != nil {
			t.Fatalf("iteration %d: Acquire failed: %v", i, err)
		}

		// Simulate a small amount of work by writing a file.
		marker := filepath.Join(dir, ".p", "work")
		if err := os.WriteFile(marker, []byte("busy"), 0o644); err != nil {
			t.Fatalf("iteration %d: WriteFile: %v", i, err)
		}

		lk.Release()
	}
}

func TestLockReadMalformed(t *testing.T) {
	dir := setupLockTest(t)

	// Write garbage to the lock file.
	if err := os.WriteFile(lockPath(dir), []byte("not-a-valid-lock\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Read(dir)
	if err == nil {
		t.Error("Read should return an error for malformed lock file")
	}
}

func TestLockReadMissingFile(t *testing.T) {
	dir := setupLockTest(t)

	_, err := Read(dir)
	if err == nil {
		t.Error("Read should return an error when lock file does not exist")
	}
}

func TestLockReleaseTwice(t *testing.T) {
	dir := setupLockTest(t)

	lk, err := Acquire(dir)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// First release removes the file.
	lk.Release()
	// Second release should not panic (os.Remove on missing file is a no-op error).
	lk.Release()
}

func TestLockReleaseNil(t *testing.T) {
	// Calling Release on a nil *Lock should not panic.
	var lk *Lock
	lk.Release()
}
