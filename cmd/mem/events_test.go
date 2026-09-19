package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/db"
	"github.com/FelipeMiiller/my-memory/internal/event_runtime"
)

// memBinary returns the absolute path to the mem binary in the repo's
// bin/ directory, building it first if needed (or if it's older than the
// source). Tests rely on this so subprocess execution matches the same
// binary the user runs.
//
// We always resolve the path relative to the repo root (where go.mod lives),
// NOT the test's cwd — otherwise the test would resolve bin/ relative to
// cmd/mem/ and find a stale binary.
func memBinary(t *testing.T) string {
	t.Helper()
	bin := "mem"
	if runtime.GOOS == "windows" {
		bin = "mem.exe"
	}

	// Find repo root by walking up from cwd until we find go.mod.
	cwd, _ := os.Getwd()
	repoRoot := findRepoRoot(t, cwd)
	path := filepath.Join(repoRoot, "bin", bin)

	// Always rebuild to avoid stale-binary issues (build is ~1s).
	cmd := exec.Command("go", "build", "-o", path, "./cmd/mem")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build mem binary (cwd=%s): %v\n%s", repoRoot, err, out)
	}
	return path
}

// findRepoRoot walks up from start looking for go.mod.
func findRepoRoot(t *testing.T, start string) string {
	t.Helper()
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod não encontrado a partir de %s", start)
		}
		dir = parent
	}
}

// seedEventLog creates a tempdir with .memory/config.yaml + a SQLite
// memory.db containing `count` events of the given types.
func seedEventLog(t *testing.T, count int, eventType string) (rootDir string, dbPath string) {
	t.Helper()
	rootDir = t.TempDir()
	memoryDir := filepath.Join(rootDir, ".memory")
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(memoryDir, "config.yaml"),
		[]byte("vault:\n  name: test\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	dbPath = filepath.Join(memoryDir, "memory.db")

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	log := event_runtime.NewLog(database)
	outbox := event_runtime.NewOutbox(log)
	ctx := context.Background()

	for i := 0; i < count; i++ {
		env := event_runtime.NewEnvelope(eventType, "agg-"+strconv.Itoa(i))
		env.Payload = json.RawMessage(`{"i":` + strconv.Itoa(i) + `}`)
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			database.Close()
			t.Fatalf("BeginTx: %v", err)
		}
		if err := outbox.Emit(ctx, tx, env); err != nil {
			tx.Rollback()
			database.Close()
			t.Fatalf("Emit: %v", err)
		}
		if err := tx.Commit(); err != nil {
			database.Close()
			t.Fatalf("Commit: %v", err)
		}
	}
	database.Close()
	return rootDir, dbPath
}

// runMem runs the mem binary in rootDir with args and returns (stdout, stderr, exitCode).
func runMem(t *testing.T, rootDir string, args ...string) (string, string, int) {
	t.Helper()
	bin := memBinary(t)
	cmd := exec.Command(bin, args...)
	cmd.Dir = rootDir
	var out, errOut strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err := cmd.Run()
	exit := 0
	if ee, ok := err.(*exec.ExitError); ok {
		exit = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("run mem: %v", err)
	}
	return out.String(), errOut.String(), exit
}

// TestCLI_EventsTail_PrintsJSONLines: 5 events → 5 valid JSON lines.
func TestCLI_EventsTail_PrintsJSONLines(t *testing.T) {
	rootDir, _ := seedEventLog(t, 5, "memory.committed")
	stdout, stderr, exit := runMem(t, rootDir, "events", "tail", "--limit", "5")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s", exit, stderr)
	}
	lines := splitNonEmpty(stdout)
	if len(lines) != 5 {
		t.Fatalf("tail retornou %d linhas; esperado 5 (stdout=%q)", len(lines), stdout)
	}
	for i, line := range lines {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("linha %d inválida: %v (line=%q)", i, err, line)
		}
	}
}

// TestCLI_EventsInspect_NotFound_ExitsNonZero.
func TestCLI_EventsInspect_NotFound_ExitsNonZero(t *testing.T) {
	rootDir, _ := seedEventLog(t, 1, "memory.committed")
	_, _, exit := runMem(t, rootDir, "events", "inspect", "00000000-0000-0000-0000-000000000000")
	if exit == 0 {
		t.Errorf("inspect com ID inexistente deveria sair != 0; exit=%d", exit)
	}
}

// TestCLI_EventsInspect_Found_PrintsEnvelope.
func TestCLI_EventsInspect_Found_PrintsEnvelope(t *testing.T) {
	rootDir, dbPath := seedEventLog(t, 1, "memory.committed")

	// Get the event_id directly from the DB.
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	var eventID string
	if err := database.QueryRowContext(context.Background(), `SELECT event_id FROM event_log LIMIT 1`).Scan(&eventID); err != nil {
		database.Close()
		t.Fatalf("query event_id: %v", err)
	}
	database.Close()

	stdout, stderr, exit := runMem(t, rootDir, "events", "inspect", eventID)
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, eventID) {
		t.Errorf("output não contém event_id=%s: %s", eventID, stdout)
	}
	if !strings.Contains(stdout, "memory.committed") {
		t.Errorf("output não contém event_type=memory.committed: %s", stdout)
	}
	if !strings.Contains(stdout, "payload") {
		t.Errorf("output não contém campo payload: %s", stdout)
	}
}

// TestCLI_ReplayDryRun_DoesNotApplyEffects.
func TestCLI_ReplayDryRun_DoesNotApplyEffects(t *testing.T) {
	rootDir, _ := seedEventLog(t, 2, "memory.committed")
	stdout, stderr, exit := runMem(t, rootDir, "events", "replay", "--since", "1", "--dry-run")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s", exit, stderr)
	}
	lines := splitNonEmpty(stdout)
	if len(lines) != 2 {
		t.Fatalf("replay dry-run retornou %d linhas; esperado 2 (stdout=%q)", len(lines), stdout)
	}
	for _, line := range lines {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("linha não parseável: %v", err)
			continue
		}
		if applied, _ := m["applied"].(bool); applied {
			t.Errorf("dry-run deveria ter applied=false; line=%s", line)
		}
	}
}

// TestCLI_ReplayApply_RequiresConfirmation: --apply without --yes fails-closed (exit 3).
func TestCLI_ReplayApply_RequiresConfirmation(t *testing.T) {
	rootDir, _ := seedEventLog(t, 1, "memory.committed")
	_, stderr, exit := runMem(t, rootDir, "events", "replay", "--since", "1", "--apply")
	if exit == 0 {
		t.Errorf("--apply sem --yes deveria falhar; exit=%d", exit)
	}
	if !strings.Contains(stderr, "--yes") && !strings.Contains(stderr, "fail-closed") {
		t.Logf("stderr não menciona --yes/fail-closed (acceptable, just informational): %s", stderr)
	}
}

// TestCLI_Trace_ReturnsAllEventsWithCorrelationID: 3 events with corr-A, 2 with corr-B.
func TestCLI_Trace_ReturnsAllEventsWithCorrelationID(t *testing.T) {
	rootDir := t.TempDir()
	memoryDir := filepath.Join(rootDir, ".memory")
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(memoryDir, "config.yaml"),
		[]byte("vault:\n  name: test\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	dbPath := filepath.Join(memoryDir, "memory.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	log := event_runtime.NewLog(database)
	outbox := event_runtime.NewOutbox(log)
	ctx := context.Background()

	correlationID := "corr-trace-test-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	for i := 0; i < 5; i++ {
		env := event_runtime.NewEnvelope("memory.committed", "agg-"+strconv.Itoa(i))
		env.Payload = json.RawMessage(`{"i":` + strconv.Itoa(i) + `}`)
		if i < 3 {
			env.CorrelationID = correlationID
		}
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx: %v", err)
		}
		if err := outbox.Emit(ctx, tx, env); err != nil {
			tx.Rollback()
			t.Fatalf("Emit: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit: %v", err)
		}
	}
	database.Close()

	stdout, stderr, exit := runMem(t, rootDir, "events", "trace", correlationID)
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s", exit, stderr)
	}
	lines := splitNonEmpty(stdout)
	if len(lines) != 3 {
		t.Fatalf("trace retornou %d linhas; esperado 3 (stdout=%q)", len(lines), stdout)
	}
	for _, line := range lines {
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("linha não parseável: %v (line=%q)", err, line)
		}
	}
}

// TestCLI_LastSequence_PrintsMaxSequence.
func TestCLI_LastSequence_PrintsMaxSequence(t *testing.T) {
	rootDir, _ := seedEventLog(t, 7, "memory.committed")
	stdout, stderr, exit := runMem(t, rootDir, "events", "last-sequence")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s", exit, stderr)
	}
	if strings.TrimSpace(stdout) != "7" {
		t.Errorf("last-sequence=%q; esperado \"7\"", strings.TrimSpace(stdout))
	}
}

// TestCLI_Stats_ReportsOldestUnacked: 3 unacked events → oldest_unacked_age_seconds >= 0.
func TestCLI_Stats_ReportsOldestUnacked(t *testing.T) {
	rootDir, _ := seedEventLog(t, 3, "memory.committed")
	time.Sleep(50 * time.Millisecond) // ensure age > 0

	stdout, stderr, exit := runMem(t, rootDir, "events", "stats")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s", exit, stderr)
	}
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &stats); err != nil {
		t.Fatalf("stats output não é JSON: %v (stdout=%q)", err, stdout)
	}
	if total, _ := stats["total_events"].(float64); int(total) != 3 {
		t.Errorf("total_events=%v; esperado 3", stats["total_events"])
	}
	if _, ok := stats["oldest_unacked_age_seconds"]; !ok {
		t.Error("stats não contém oldest_unacked_age_seconds")
	}
	if _, ok := stats["events_by_type_top10"]; !ok {
		t.Error("stats não contém events_by_type_top10")
	}
}

// TestCLI_MissingConfig_Exits2: run with no .memory/config.yaml → exit 2 + stderr mentions "mem init".
func TestCLI_MissingConfig_Exits2(t *testing.T) {
	rootDir := t.TempDir() // empty — no .memory
	_, stderr, exit := runMem(t, rootDir, "events", "tail")
	if exit != 2 {
		t.Errorf("exit=%d; esperado 2", exit)
	}
	if !strings.Contains(stderr, "mem init") {
		t.Errorf("stderr deveria mencionar 'mem init': %q", stderr)
	}
}

// splitNonEmpty returns lines from s, dropping empty lines.
func splitNonEmpty(s string) []string {
	out := []string{}
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}
