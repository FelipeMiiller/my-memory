package supervisor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// withStubbedAlive overrides IsAlivePID for the duration of t and
// restores the previous implementation on cleanup. Tests that need
// deterministic PID-aliveness (e.g. simulating a dead prior process)
// MUST use this — the default impl uses syscall.Kill(0) which is
// platform-dependent.
func isAliveStub(t *testing.T, fn func(pid int) bool) {
	t.Helper()
	prev := IsAlivePID
	IsAlivePID = fn
	t.Cleanup(func() { IsAlivePID = prev })
}

func TestLock_FreshBlocks(t *testing.T) {
	dir := t.TempDir()
	const livePID = 12345

	// Stub liveness so we don't depend on the test runner's PIDs.
	isAliveStub(t, func(p int) bool { return p == livePID })

	// Pre-write a fresh lock for pid=12345.
	existing := LockData{
		Format:    LockFormatVersion,
		PID:       livePID,
		StartedAt: time.Now().Add(-5 * time.Second),
		Profile:   "default",
	}
	if err := writeLockAtomic(filepath.Join(dir, LockFileName), &existing); err != nil {
		t.Fatalf("seed lock: %v", err)
	}

	lock, err := AcquireLock(dir, "default")
	if err == nil {
		t.Fatalf("expected ErrAlreadyRunning, got nil (lock=%+v)", lock)
	}
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("expected ErrAlreadyRunning, got %v", err)
	}

	// Lock file should still be the original (not overwritten).
	cur, rerr := ReadLock(dir)
	if rerr != nil {
		t.Fatalf("read lock: %v", rerr)
	}
	if cur.PID != livePID {
		t.Fatalf("lock should not have been overwritten, got pid=%d (want %d)", cur.PID, livePID)
	}
}

func TestLock_StaleRemoved(t *testing.T) {
	dir := t.TempDir()
	const deadPID = 999999

	// Pretend the prior PID is dead regardless of reality.
	isAliveStub(t, func(p int) bool { return false })

	stale := LockData{
		Format:    LockFormatVersion,
		PID:       deadPID,
		StartedAt: time.Now().Add(-2 * time.Minute),
		Profile:   "old",
	}
	if err := writeLockAtomic(filepath.Join(dir, LockFileName), &stale); err != nil {
		t.Fatalf("seed stale lock: %v", err)
	}

	lock, err := AcquireLock(dir, "default")
	if err != nil {
		t.Fatalf("expected success on stale lock, got %v", err)
	}
	if lock.PID != os.Getpid() {
		t.Fatalf("expected current PID %d, got %d", os.Getpid(), lock.PID)
	}
	if lock.Profile != "default" {
		t.Fatalf("expected profile=default, got %q", lock.Profile)
	}
	if lock.Format != LockFormatVersion {
		t.Fatalf("expected format=%d, got %d", LockFormatVersion, lock.Format)
	}

	// The lock file on disk should now reflect the new owner.
	cur, rerr := ReadLock(dir)
	if rerr != nil {
		t.Fatalf("read lock: %v", rerr)
	}
	if cur.PID != lock.PID {
		t.Fatalf("disk lock PID=%d, expected %d", cur.PID, lock.PID)
	}
}

func TestLock_NoExistingCreatesNew(t *testing.T) {
	dir := t.TempDir()

	lock, err := AcquireLock(dir, "headless")
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if lock.Profile != "headless" {
		t.Fatalf("expected profile=headless, got %q", lock.Profile)
	}
	if lock.PID != os.Getpid() {
		t.Fatalf("expected PID=%d, got %d", os.Getpid(), lock.PID)
	}
}

func TestLock_ReleaseIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	if _, err := AcquireLock(dir, "default"); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if err := ReleaseLock(dir); err != nil {
		t.Fatalf("first release: %v", err)
	}
	if err := ReleaseLock(dir); err != nil {
		t.Fatalf("second release should be no-op, got %v", err)
	}
}

func TestLock_AutoCreatesMemoryDir(t *testing.T) {
	root := t.TempDir()
	memoryDir := filepath.Join(root, "deep", "nested", ".memory")

	_, err := AcquireLock(memoryDir, "default")
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if _, err := os.Stat(filepath.Join(memoryDir, LockFileName)); err != nil {
		t.Fatalf("lock file missing after acquire: %v", err)
	}
}

func TestLock_RejectsEmptyMemoryDir(t *testing.T) {
	_, err := AcquireLock("", "default")
	if err == nil {
		t.Fatal("expected error on empty memoryDir, got nil")
	}
}

func TestLock_CorruptOverwritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, LockFileName)
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatalf("seed corrupt lock: %v", err)
	}

	lock, err := AcquireLock(dir, "default")
	if err != nil {
		t.Fatalf("expected corrupt lock to be overwritten, got %v", err)
	}
	if lock.PID != os.Getpid() {
		t.Fatalf("expected current PID, got %d", lock.PID)
	}
}
