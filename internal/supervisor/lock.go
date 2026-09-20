package supervisor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StaleTimeout is the maximum age of a supervisor.lock file before
// it is considered stale and safe to overwrite. The 30s window is
// wide enough to survive a quick restart while narrow enough to recover
// from a crashed supervisor without operator intervention (ADR-042
// §lock-file).
const StaleTimeout = 30 * time.Second

// LockFileName is the basename of the lock file under the memory
// directory. Exported so `mem status` and tests can reference it
// without hardcoding.
const LockFileName = "supervisor.lock"

// LockFormatVersion is the lock payload format. Bump on breaking
// changes; older payloads are treated as corrupt and overwritten.
const LockFormatVersion = 1

// Sentinel errors. Use errors.Is for matching.
var (
	// ErrAlreadyRunning is returned by AcquireLock when a fresh lock
	// with a live PID is already present in the memory directory.
	ErrAlreadyRunning = errors.New("supervisor: another instance is already running")

	// ErrInvalidProfile is returned when the profile YAML is missing
	// or fails to parse. ADR-050 LLM10 fail-closed semantics apply —
	// the supervisor exits with code 2 rather than falling back to
	// defaults.
	ErrInvalidProfile = errors.New("supervisor: invalid profile")

	// ErrLockCorrupt is returned when the lock file is present but
	// cannot be decoded. The caller treats this as stale and
	// overwrites.
	ErrLockCorrupt = errors.New("supervisor: lock file is corrupt")
)

// LockData is the JSON payload written to supervisor.lock. Fields
// match ADR-042 §lock-file. Format exists for forward-compatibility
// (bump + add fields, never reuse).
type LockData struct {
	Format    int       `json:"format"`
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
	Profile   string    `json:"profile"`
}

// IsAlivePID is the per-process liveness check used by AcquireLock
// when deciding whether an existing lock is held by a live process.
// The default implementation lives in lock_unix.go and lock_windows.go
// (build-tag separated); it uses syscall.Kill(pid, 0) on Unix and
// returns true unconditionally on Windows (Windows has no signal-0
// equivalent, so the StaleTimeout gate is the primary signal there).
//
// The variable is exported so tests can stub it; production callers
// should treat it as read-only.
var IsAlivePID = defaultIsAlivePID

// AcquireLock acquires the supervisor lock at
// <memoryDir>/supervisor.lock. Behavior:
//
//   - When no lock exists, one is written with the current PID and
//     timestamp, then returned.
//   - When a lock exists but is older than StaleTimeout OR its PID
//     is no longer alive, the lock is overwritten (stale cleanup).
//   - When a lock exists, is fresh, AND its PID is alive,
//     ErrAlreadyRunning is returned.
//
// The memory directory is auto-created if missing. memoryDir must
// be non-empty (caller error otherwise).
func AcquireLock(memoryDir, profile string) (*LockData, error) {
	if memoryDir == "" {
		return nil, errors.New("supervisor: memoryDir is empty")
	}
	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		return nil, fmt.Errorf("supervisor: mkdir %s: %w", memoryDir, err)
	}
	path := filepath.Join(memoryDir, LockFileName)

	existing, readErr := readLock(path)
	switch {
	case readErr == nil:
		age := time.Since(existing.StartedAt)
		if age < StaleTimeout && IsAlivePID(existing.PID) {
			return nil, fmt.Errorf("%w: pid=%d age=%s profile=%q",
				ErrAlreadyRunning, existing.PID, age, existing.Profile)
		}
		// Stale or dead PID — fall through and overwrite.
	case errors.Is(readErr, ErrLockCorrupt):
		// Treat as stale — fall through and overwrite.
	case errors.Is(readErr, os.ErrNotExist):
		// No existing lock — fall through and write.
	default:
		return nil, fmt.Errorf("supervisor: read existing lock: %w", readErr)
	}

	lock := &LockData{
		Format:    LockFormatVersion,
		PID:       os.Getpid(),
		StartedAt: time.Now().UTC(),
		Profile:   profile,
	}
	if err := writeLockAtomic(path, lock); err != nil {
		return nil, fmt.Errorf("supervisor: write lock: %w", err)
	}
	return lock, nil
}

// ReleaseLock removes the supervisor.lock file. A missing file is
// not an error (idempotent); other I/O errors are returned.
//
// Callers should defer this right after AcquireLock so every exit
// path releases the lock — including panic-induced crashes.
func ReleaseLock(memoryDir string) error {
	path := filepath.Join(memoryDir, LockFileName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("supervisor: remove lock: %w", err)
	}
	return nil
}

// ReadLock returns the current lock payload, or ErrLockCorrupt when
// the file cannot be decoded. Used by `mem status` and by tests.
func ReadLock(memoryDir string) (*LockData, error) {
	return readLock(filepath.Join(memoryDir, LockFileName))
}

func readLock(path string) (*LockData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lock LockData
	if err := json.NewDecoder(f).Decode(&lock); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLockCorrupt, err)
	}
	return &lock, nil
}

// writeLockAtomic writes the lock payload atomically: write to
// <path>.tmp, fsync, rename. A reader at any moment sees either
// the previous payload (or no file) or the new one — never a
// partial write. Atomicity is essential because mymemoryd may crash
// between fork and write.
func writeLockAtomic(path string, lock *LockData) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	if err := enc.Encode(lock); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// defaultIsAlivePID is a thin wrapper that gates the self-PID
// short-circuit and the platform-specific implementation in
// lock_unix.go / lock_windows.go.
//
// Self-PID is treated as alive so tests can pre-seed a lock with
// their own PID and assert the fresh-blocks branch without poking
// the platform-default liveness check.
func defaultIsAlivePID(pid int) bool {
	if pid <= 0 {
		return false
	}
	if pid == os.Getpid() {
		return true
	}
	return platformIsAlive(pid)
}
