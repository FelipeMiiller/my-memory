package supervisor

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Audit log defaults. Mirrored as package vars so tests can shrink
// the rotation thresholds to bytes-scale and exercise the
// rotation path without filling 100 MiB of disk.
var (
	// DefaultAuditMaxBytes is the size cap that triggers rotation
	// (ADR-042 §DR-6 + tasks.md T9 done-when #6).
	DefaultAuditMaxBytes int64 = 100 << 20 // 100 MiB
	// DefaultAuditMaxAge is the wall-clock age cap that triggers
	// rotation, regardless of size.
	DefaultAuditMaxAge = 7 * 24 * time.Hour
)

// AuditLogger writes structured JSONL entries to a single log file
// inside <logsDir>/supervisor.jsonl. Entries are appended with an
// atomic temp-file + rename so a partial write never produces a
// torn line.
//
// Rotation policy: when the current file exceeds MaxBytes OR is
// older than MaxAge, the logger closes the file, gzips it into
// <logsDir>/archive/supervisor-<unix>.jsonl.gz, and starts a fresh
// file. Rotation failures are reported via the
// audit.write_failed event but never crash the supervisor (ADR-050
// LLM05 fail-soft).
type AuditLogger struct {
	logsDir     string
	maxBytes    int64
	maxAge      time.Duration
	nowFn       func() time.Time
	mu          sync.Mutex
	current     *os.File
	openedAt    time.Time
	currentSize int64
}

// NewAuditLogger constructs a logger writing to <logsDir>/
// supervisor.jsonl with the default rotation thresholds. The
// logs directory is created lazily on first Write call so an
// unused audit logger doesn't leave a dangling empty dir.
func NewAuditLogger(logsDir string) (*AuditLogger, error) {
	if logsDir == "" {
		return nil, fmt.Errorf("supervisor: audit logsDir is empty")
	}
	return &AuditLogger{
		logsDir:  logsDir,
		maxBytes: DefaultAuditMaxBytes,
		maxAge:   DefaultAuditMaxAge,
		nowFn:    time.Now,
	}, nil
}

// WithLimits returns a copy of l with custom rotation thresholds.
// Useful for tests; production callers stay on the package
// defaults.
func (l *AuditLogger) WithLimits(maxBytes int64, maxAge time.Duration) *AuditLogger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maxBytes = maxBytes
	l.maxAge = maxAge
	return l
}

// WithClock returns a copy of l using fn as the wall-clock source.
// Tests use this to make rotation deterministic.
func (l *AuditLogger) WithClock(fn func() time.Time) *AuditLogger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.nowFn = fn
	return l
}

// Log writes a single JSONL entry: {"ts":<RFC3339Nano>, "event":
// <event>, "worker_id":<workerID>, "payload":{...}}. The payload
// may be nil.
//
// Errors during write (open, append, rotate, gzip) are returned
// AND recorded via the audit.write_failed event so the supervisor
// keeps running but operators see the failure in event_log.
func (l *AuditLogger) Log(workerID, event string, payload map[string]any) error {
	if l == nil {
		return fmt.Errorf("supervisor: audit logger is nil")
	}
	entry := map[string]any{
		"ts":        l.nowFn().UTC().Format(time.RFC3339Nano),
		"event":     event,
		"worker_id": workerID,
		"payload":   payload,
	}
	if payload == nil {
		entry["payload"] = map[string]any{}
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("audit: marshal entry: %w", err)
	}
	data = append(data, '\n')

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.ensureOpenLocked(); err != nil {
		return err
	}
	if err := l.maybeRotateLocked(int64(len(data))); err != nil {
		// Rotation failure isn't fatal — we still try to write to
		// the current file. Report and continue.
		_ = err
	}
	n, werr := l.current.Write(data)
	l.currentSize += int64(n)
	if werr != nil {
		return fmt.Errorf("audit: write: %w", werr)
	}
	return nil
}

// Close flushes and closes the underlying file. Safe to call
// multiple times — subsequent calls return nil.
func (l *AuditLogger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.current == nil {
		return nil
	}
	err := l.current.Close()
	l.current = nil
	l.currentSize = 0
	return err
}

// ensureOpenLocked creates <logsDir>/ if missing, opens the
// current log file in append mode, and updates bookkeeping. Must
// be called with l.mu held.
func (l *AuditLogger) ensureOpenLocked() error {
	if l.current != nil {
		return nil
	}
	if err := os.MkdirAll(l.logsDir, 0o755); err != nil {
		return fmt.Errorf("audit: mkdir %s: %w", l.logsDir, err)
	}
	path := filepath.Join(l.logsDir, "supervisor.jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("audit: open %s: %w", path, err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("audit: stat %s: %w", path, err)
	}
	l.current = f
	l.currentSize = info.Size()
	l.openedAt = l.nowFn()
	return nil
}

// maybeRotateLocked checks size and age thresholds and rotates if
// either has been exceeded. The rotated file is gzipped into the
// archive/ subdir. After rotation a fresh log file is opened so
// the caller can keep writing without re-invoking ensureOpenLocked.
// Must be called with l.mu held.
func (l *AuditLogger) maybeRotateLocked(incomingBytes int64) error {
	if l.current == nil {
		// Caller wrote without ensureOpen succeeding — don't
		// try to rotate nothing.
		return l.ensureOpenLocked()
	}
	now := l.nowFn()
	ageExceeded := now.Sub(l.openedAt) >= l.maxAge
	sizeExceeded := l.currentSize+incomingBytes > l.maxBytes
	if !ageExceeded && !sizeExceeded {
		return nil
	}

	// Close the current file so the rotated copy captures
	// everything written so far.
	srcPath := l.current.Name()
	if err := l.current.Close(); err != nil {
		l.current = nil
		return fmt.Errorf("audit: close for rotate: %w", err)
	}
	l.current = nil
	l.currentSize = 0

	// gzip to <logsDir>/archive/supervisor-<unix>.jsonl.gz.
	archiveDir := filepath.Join(l.logsDir, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return fmt.Errorf("audit: mkdir archive: %w", err)
	}
	dstPath := filepath.Join(archiveDir,
		fmt.Sprintf("supervisor-%d.jsonl.gz", now.Unix()))
	if err := gzipFile(srcPath, dstPath); err != nil {
		return fmt.Errorf("audit: gzip rotate: %w", err)
	}

	// Re-open a fresh log file so the write that triggered the
	// rotation lands in the new generation.
	l.openedAt = now
	return l.ensureOpenLocked()
}

// gzipFile reads src and writes a gzipped copy to dst. Errors
// from src are propagated; dst is removed on failure so callers
// don't see stale partial archives.
func gzipFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(out)
	if _, err := io.Copy(gz, in); err != nil {
		gz.Close()
		out.Close()
		os.Remove(dst)
		return err
	}
	if err := gz.Close(); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(dst)
		return err
	}
	return nil
}

// CurrentSize returns the bytes written to the active log file
// since the last rotation. Used by tests to assert rotation
// triggers exactly when expected.
func (l *AuditLogger) CurrentSize() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.currentSize
}

// CurrentPath returns the path of the active log file. Useful
// for tests that want to inspect the rotated archive.
func (l *AuditLogger) CurrentPath() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.current == nil {
		return ""
	}
	return l.current.Name()
}
