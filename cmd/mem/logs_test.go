package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCliLogs_FilteredByWorker exercises the spec-mandated done-
// when #5: `mem logs <worker>` returns only entries whose
// aggregate_id matches.
func TestCliLogs_FilteredByWorker(t *testing.T) {
	memDir := t.TempDir()
	logsDir := filepath.Join(memDir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("mkdir logs: %v", err)
	}
	logPath := filepath.Join(logsDir, "supervisor.jsonl")

	// Seed with three entries: two for "alpha" and one for "beta".
	entries := []map[string]any{
		{
			"event_type":   "worker.started",
			"aggregate_id": "alpha",
			"created_at":   "2026-09-20T10:00:00Z",
			"payload":      map[string]any{"pid": 111},
		},
		{
			"event_type":   "worker.stopped",
			"aggregate_id": "beta",
			"created_at":   "2026-09-20T10:01:00Z",
			"payload":      map[string]any{"pid": 222},
		},
		{
			"event_type":   "worker.stopped",
			"aggregate_id": "alpha",
			"created_at":   "2026-09-20T10:02:00Z",
			"payload":      map[string]any{"pid": 111, "reason": "requested"},
		},
	}
	var body strings.Builder
	for _, e := range entries {
		b, _ := json.Marshal(e)
		body.Write(b)
		body.WriteByte('\n')
	}
	if err := os.WriteFile(logPath, []byte(body.String()), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var out, errOut bytes.Buffer
	if code := runLogsCommand(ctx, []string{"alpha", "--memory-dir", memDir}, memDir, &out, &errOut); code != 0 {
		t.Fatalf("logs: code=%d stderr=%s", code, errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "alpha") {
		t.Errorf("expected alpha entries, got: %s", got)
	}
	if strings.Contains(got, "beta") {
		t.Errorf("beta entries should be filtered out, got: %s", got)
	}
	// Two alpha entries must both appear.
	if strings.Count(got, "worker=") != 2 {
		t.Errorf("expected 2 alpha lines, got %d in: %s", strings.Count(got, "worker="), got)
	}
}

// TestCliLogs_NoFilterReturnsAll confirms `mem logs` without a
// worker arg enumerates the whole log.
func TestCliLogs_NoFilterReturnsAll(t *testing.T) {
	memDir := t.TempDir()
	logsDir := filepath.Join(memDir, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	logPath := filepath.Join(logsDir, "supervisor.jsonl")
	entries := []string{
		`{"event_type":"worker.started","aggregate_id":"a","created_at":"t1","payload":{"pid":1}}`,
		`{"event_type":"worker.started","aggregate_id":"b","created_at":"t2","payload":{"pid":2}}`,
		`{"event_type":"worker.started","aggregate_id":"c","created_at":"t3","payload":{"pid":3}}`,
	}
	if err := os.WriteFile(logPath, []byte(strings.Join(entries, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var out, errOut bytes.Buffer
	if code := runLogsCommand(ctx, []string{"--memory-dir", memDir}, memDir, &out, &errOut); code != 0 {
		t.Fatalf("logs: code=%d stderr=%s", code, errOut.String())
	}
	if strings.Count(out.String(), "worker=") != 3 {
		t.Errorf("expected 3 lines, got %d in: %s", strings.Count(out.String(), "worker="), out.String())
	}
}

// TestCliLogs_NoFileIsNoop verifies the missing-log graceful path:
// no logs/supervisor.jsonl → exit 0 with a friendly message.
func TestCliLogs_NoFileIsNoop(t *testing.T) {
	memDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var out, errOut bytes.Buffer
	code := runLogsCommand(ctx, []string{"--memory-dir", memDir}, memDir, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit 0 on missing log file, got %d (stderr=%s)", code, errOut.String())
	}
	if !strings.Contains(out.String(), "no log file") {
		t.Errorf("expected 'no log file' message, got: %s", out.String())
	}
}

// silence unused time import when tests don't need it after refactors.
var _ = time.Second
