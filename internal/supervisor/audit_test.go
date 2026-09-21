package supervisor

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAudit_LogsAllLifecycleEvents is spec-mandated (tasks.md
// T9 done-when #10). 100 start/stop cycles → 200+ entries in the
// JSONL log. Each entry carries worker_id, pid, ts, event, payload.
func TestAudit_LogsAllLifecycleEvents(t *testing.T) {
	logsDir := filepath.Join(t.TempDir(), "logs")
	logger, err := NewAuditLogger(logsDir)
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	defer logger.Close()

	const cycles = 100
	for i := 0; i < cycles; i++ {
		workerID := "worker-" + itoaForAudit(i%5)
		_ = logger.Log(workerID, "worker.started", map[string]any{
			"pid":     1000 + i,
			"command": "/bin/true",
		})
		_ = logger.Log(workerID, "worker.stopped", map[string]any{
			"pid":     1000 + i,
			"reason":  "requested",
			"outcome": "graceful",
		})
	}

	path := filepath.Join(logsDir, "supervisor.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	lines := splitLines(string(data))
	if len(lines) < cycles*2 {
		t.Fatalf("got %d lines, want >=%d", len(lines), cycles*2)
	}

	// Spot-check one entry: every line must be valid JSON with
	// the expected keys.
	for i, line := range lines {
		var e map[string]any
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("line %d: not valid JSON: %v\n%s", i, err, line)
		}
		for _, key := range []string{"ts", "event", "worker_id", "payload"} {
			if _, ok := e[key]; !ok {
				t.Errorf("line %d missing key %q: %s", i, key, line)
				break
			}
		}
	}
}

// TestAudit_RotatesAtSizeLimit is spec-mandated (tasks.md T9
// done-when #11). Configure a tiny rotation threshold so the
// test exercises rotation without filling 100 MiB.
func TestAudit_RotatesAtSizeLimit(t *testing.T) {
	logsDir := filepath.Join(t.TempDir(), "logs")
	logger, err := NewAuditLogger(logsDir)
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	defer logger.Close()
	// Force rotation after ~1 KiB. Each entry is ~120 bytes
	// (typical JSONL row); 30 entries should trip the cap.
	logger.WithLimits(1024, 1*time.Hour)

	for i := 0; i < 50; i++ {
		_ = logger.Log("rotator", "worker.started", map[string]any{
			"pid":     i,
			"command": "/bin/true",
			"args":    []string{"--flag", "value"},
		})
	}

	// At least one rotated file must exist under archive/.
	archiveDir := filepath.Join(logsDir, "archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no rotated files in archive/")
	}

	// Rotated file must be a valid gzip containing the original
	// JSONL lines.
	var found bool
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".jsonl.gz") {
			continue
		}
		path := filepath.Join(archiveDir, e.Name())
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("open rotated: %v", err)
		}
		gz, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			t.Fatalf("gzip reader: %v", err)
		}
		data, _ := io.ReadAll(gz)
		gz.Close()
		f.Close()
		if !strings.Contains(string(data), `"event":"worker.started"`) {
			t.Errorf("rotated file missing expected entries: %s", data)
		}
		found = true
	}
	if !found {
		t.Fatal("no .jsonl.gz files in archive/")
	}
}

// TestAudit_RotatesAtAgeLimit exercises the wall-clock age
// threshold (done-when #6 second clause).
func TestAudit_RotatesAtAgeLimit(t *testing.T) {
	logsDir := filepath.Join(t.TempDir(), "logs")
	logger, err := NewAuditLogger(logsDir)
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	defer logger.Close()

	// Pin the clock so we can advance time deterministically.
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	current := base
	logger.WithClock(func() time.Time { return current })

	if err := logger.Log("alpha", "worker.started", map[string]any{"pid": 1}); err != nil {
		t.Fatalf("first log: %v", err)
	}
	// Jump 8 days into the future.
	current = base.Add(8 * 24 * time.Hour)
	if err := logger.Log("alpha", "worker.started", map[string]any{"pid": 2}); err != nil {
		t.Fatalf("second log: %v", err)
	}

	archiveDir := filepath.Join(logsDir, "archive")
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("age threshold didn't trigger rotation")
	}
}

// TestAudit_WriteFailureSurfaces confirms the audit.write_failed
// contract (done-when #7): the logger doesn't panic when the
// underlying file can't be written. We point logsDir at a path
// under a non-existent parent so MkdirAll fails predictably.
func TestAudit_WriteFailureSurfaces(t *testing.T) {
	root := t.TempDir()
	// Make a regular file at <root>/blocker so MkdirAll(<root>/blocker/x) fails.
	blocker := filepath.Join(root, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	logger, err := NewAuditLogger(filepath.Join(blocker, "logs"))
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	defer logger.Close()

	err = logger.Log("alpha", "worker.started", map[string]any{"pid": 1})
	if err == nil {
		t.Fatal("expected write failure when logsDir parent is a regular file")
	}
	if !strings.Contains(err.Error(), "audit:") {
		t.Errorf("expected audit-prefixed error, got: %v", err)
	}
}

// TestAudit_LazyDirCreate confirms the logsDir is created lazily
// on first Write so an unused audit logger doesn't leave a
// dangling empty directory.
func TestAudit_LazyDirCreate(t *testing.T) {
	root := t.TempDir()
	logsDir := filepath.Join(root, "nested", "logs")
	logger, err := NewAuditLogger(logsDir)
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	defer logger.Close()
	// Pre-condition: logsDir does NOT exist yet.
	if _, err := os.Stat(logsDir); err == nil {
		t.Fatal("logsDir should not exist before first Write")
	}
	if err := logger.Log("alpha", "worker.started", map[string]any{"pid": 1}); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if _, err := os.Stat(logsDir); err != nil {
		t.Fatalf("logsDir should exist after first Write: %v", err)
	}
}

// TestAudit_PayloadAlwaysObject confirms the payload field is
// always an object (never null), even when the caller passes
// nil. The spec doesn't mandate this but downstream consumers
// (jq, log-shippers) expect a stable shape.
func TestAudit_PayloadAlwaysObject(t *testing.T) {
	logsDir := filepath.Join(t.TempDir(), "logs")
	logger, err := NewAuditLogger(logsDir)
	if err != nil {
		t.Fatalf("NewAuditLogger: %v", err)
	}
	defer logger.Close()
	if err := logger.Log("alpha", "worker.started", nil); err != nil {
		t.Fatalf("log: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(logsDir, "supervisor.jsonl"))
	if !strings.Contains(string(data), `"payload":{}`) {
		t.Fatalf("expected payload={} when nil passed, got: %s", data)
	}
}

// silence unused-import warnings when bufio isn't referenced.
var _ = bufio.NewScanner

// itoaForAudit is a tiny int-to-string helper so the audit
// tests don't pull in strconv just for one call site.
func itoaForAudit(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}

// splitLines is a no-allocation string splitter that drops the
// trailing empty line that bufio.Scanner adds on a final \n.
func splitLines(s string) []string {
	out := make([]string, 0, 16)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if line != "" {
				out = append(out, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
