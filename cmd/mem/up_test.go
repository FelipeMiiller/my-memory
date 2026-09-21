package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCliUp_StartsAndStops is spec-mandated (tasks.md T8 done-when
// #9). It spawns the fake mymemoryd, asserts that `mem up` returns
// 0 + emits the readiness line, then calls `mem down` to stop
// the subprocess cleanly.
func TestCliUp_StartsAndStops(t *testing.T) {
	fakeBin := compileFakeMymemoryd(t)
	memDir := t.TempDir()
	writeMemoryDir(t, memDir)
	t.Setenv("FAKE_MEMORY_DIR", memDir)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var upOut, upErr bytes.Buffer
	code := runUpCommand(ctx, []string{"--memory-dir", memDir}, memDir, fakeBin, &upOut, &upErr)
	if code != 0 {
		t.Fatalf("runUpCommand: code=%d stderr=%s stdout=%s", code, upErr.String(), upOut.String())
	}
	if !strings.Contains(upOut.String(), "mymemoryd: ready") {
		t.Fatalf("expected ready line in stdout, got: %s", upOut.String())
	}

	// Read the lock file the fake wrote so we know which PID to
	// stop. We hand-roll JSON parsing to avoid importing the
	// supervisor package from the shared cmd/mem package.
	lockPath := filepath.Join(memDir, "supervisor.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	pid := parseLockPID(t, data)
	if pid <= 0 {
		t.Fatalf("invalid PID in lock: %s", data)
	}

	// mem down should now stop the supervisor.
	var downOut, downErr bytes.Buffer
	code = runDownCommand(ctx, []string{"--memory-dir", memDir}, memDir, &downOut, &downErr)
	if code != 0 {
		t.Fatalf("runDownCommand: code=%d stderr=%s stdout=%s", code, downErr.String(), downOut.String())
	}
	if !strings.Contains(downOut.String(), "Stopped") {
		t.Fatalf("expected Stopped in down output, got: %s", downOut.String())
	}
}

// TestCliUp_AlreadyRunning is spec-mandated (tasks.md T8 done-when
// #10). Two sequential `mem up` calls: the second must observe
// the lock and exit 0 with "Already running on PID <N>".
func TestCliUp_AlreadyRunning(t *testing.T) {
	fakeBin := compileFakeMymemoryd(t)
	memDir := t.TempDir()
	writeMemoryDir(t, memDir)
	t.Setenv("FAKE_MEMORY_DIR", memDir)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// First up: actually starts.
	var firstOut, firstErr bytes.Buffer
	if code := runUpCommand(ctx, []string{"--memory-dir", memDir}, memDir, fakeBin, &firstOut, &firstErr); code != 0 {
		t.Fatalf("first up: code=%d stderr=%s", code, firstErr.String())
	}

	// Second up: must be a no-op exit 0 with the friendly message.
	var secondOut, secondErr bytes.Buffer
	code := runUpCommand(ctx, []string{"--memory-dir", memDir}, memDir, fakeBin, &secondOut, &secondErr)
	if code != 0 {
		t.Fatalf("second up: code=%d stderr=%s stdout=%s", code, secondErr.String(), secondOut.String())
	}
	if !strings.Contains(secondOut.String(), "Already running on PID") {
		t.Fatalf("expected 'Already running on PID' in stdout, got: %s", secondOut.String())
	}

	// Cleanup.
	var downOut, downErr bytes.Buffer
	_ = runDownCommand(ctx, []string{"--memory-dir", memDir}, memDir, &downOut, &downErr)
}

// TestCliUp_NotReadyFails exercises the "exited before READY"
// path: the fake exits 1 without ever printing the readiness
// line. runUpCommand must observe the early exit and return
// non-zero.
func TestCliUp_NotReadyFails(t *testing.T) {
	fakeBin := compileFakeMymemoryd(t)
	memDir := t.TempDir()
	writeMemoryDir(t, memDir)
	t.Setenv("FAKE_MEMORY_DIR", memDir)
	t.Setenv("FAKE_FAIL_IMMEDIATELY", "1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var out, errOut bytes.Buffer
	code := runUpCommand(ctx, []string{"--memory-dir", memDir}, memDir, fakeBin, &out, &errOut)
	if code == 0 {
		t.Fatalf("expected non-zero exit when mymemoryd fails before READY, got 0 (stdout=%s stderr=%s)",
			out.String(), errOut.String())
	}
}

// TestCliDown_NoLockIsNoop verifies the idempotent shutdown: when
// no lock is present, `mem down` exits 0 with "Not running" and
// doesn't attempt to signal any PID.
func TestCliDown_NoLockIsNoop(t *testing.T) {
	memDir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var out, errOut bytes.Buffer
	code := runDownCommand(ctx, []string{"--memory-dir", memDir}, memDir, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d (stderr=%s)", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Not running") {
		t.Fatalf("expected 'Not running' in stdout, got: %s", out.String())
	}
}

// parseLockPID extracts the PID field from a supervisor lock JSON
// payload. We hand-roll the parsing here so the test file doesn't
// import the supervisor package (avoids a cycle risk in the shared
// package main).
func parseLockPID(t *testing.T, data []byte) int {
	t.Helper()
	const marker = `"pid":`
	idx := bytes.Index(data, []byte(marker))
	if idx < 0 {
		return 0
	}
	rest := data[idx+len(marker):]
	for len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t') {
		rest = rest[1:]
	}
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0
	}
	var pid int
	for _, b := range rest[:end] {
		pid = pid*10 + int(b-'0')
	}
	return pid
}
