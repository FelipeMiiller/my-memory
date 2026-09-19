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

// seedEventLogForDoctor creates a tempdir with .memory/config.yaml and an
// event_log containing `nAcked` acked events + `nUnacked` unacked events.
// Returns the rootDir where the binary should run.
func seedEventLogForDoctor(t *testing.T, nAcked, nUnacked int) string {
	t.Helper()
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

	total := nAcked + nUnacked
	sequences := make([]int64, 0, total)
	for i := 0; i < total; i++ {
		env := event_runtime.NewEnvelope("memory.committed", "agg-"+strconv.Itoa(i))
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
		sequences = append(sequences, env.Sequence)
	}

	// ACK the first nAcked events via MarkAcked.
	for i := 0; i < nAcked; i++ {
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			database.Close()
			t.Fatalf("BeginTx: %v", err)
		}
		// Use permanent reject with reason so the event is "acked" (out of unacked queue).
		if err := log.MarkAcked(ctx, tx, sequences[i], ""); err != nil {
			tx.Rollback()
			database.Close()
			t.Fatalf("MarkAcked: %v", err)
		}
		if err := tx.Commit(); err != nil {
			database.Close()
			t.Fatalf("Commit MarkAcked: %v", err)
		}
	}

	database.Close()
	return rootDir
}

// runMemDoctor is a thin wrapper around exec for `mem doctor` tests.
func runMemDoctor(t *testing.T, rootDir string, args ...string) (string, string, int) {
	t.Helper()
	bin := "mem"
	if runtime.GOOS == "windows" {
		bin = "mem.exe"
	}
	cwd, _ := os.Getwd()
	repoRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			t.Fatalf("go.mod não encontrado a partir de %s", cwd)
		}
		repoRoot = parent
	}
	path := filepath.Join(repoRoot, "bin", bin)

	// Always rebuild.
	buildCmd := exec.Command("go", "build", "-o", path, "./cmd/mem")
	buildCmd.Dir = repoRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build mem binary: %v\n%s", err, out)
	}

	cmd := exec.Command(path, args...)
	cmd.Dir = rootDir
	var out, errOut strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	execErr := cmd.Run()
	exit := 0
	if ee, ok := execErr.(*exec.ExitError); ok {
		exit = ee.ExitCode()
	} else if execErr != nil {
		t.Fatalf("run mem: %v", execErr)
	}
	return out.String(), errOut.String(), exit
}

// TestDoctor_EventsFlag_ReportsMetrics: setup 3 events (2 acked, 1 unacked);
// run `mem doctor --events`; assert section contains expected fields.
func TestDoctor_EventsFlag_ReportsMetrics(t *testing.T) {
	rootDir := seedEventLogForDoctor(t, 2, 1)
	stdout, stderr, exit := runMemDoctor(t, rootDir, "doctor", "--events")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s stdout=%s", exit, stderr, stdout)
	}

	// Must contain the Events section header.
	if !strings.Contains(stdout, "Events") {
		t.Errorf("stdout não contém seção 'Events': %s", stdout)
	}

	// Must contain the four expected fields.
	expected := []string{
		"total_events",
		"unacked_count",
		"oldest_unacked_age_seconds",
		"subscribers_active",
	}
	for _, f := range expected {
		if !strings.Contains(stdout, f) {
			t.Errorf("stdout não contém campo %q: %s", f, stdout)
		}
	}

	// Must contain the JSON summary line.
	if !strings.Contains(stdout, "[json]") {
		t.Errorf("stdout não contém '[json]' marker: %s", stdout)
	}

	// Extract the JSON line and assert metrics.
	jsonStart := strings.Index(stdout, "[json] {")
	if jsonStart < 0 {
		t.Fatalf("não encontrou '[json] {' em stdout=%q", stdout)
	}
	jsonPart := stdout[jsonStart+len("[json] "):]
	jsonEnd := strings.Index(jsonPart, "\n")
	if jsonEnd < 0 {
		jsonEnd = len(jsonPart)
	}
	jsonPart = jsonPart[:jsonEnd]

	var report EventsHealthReport
	if err := json.Unmarshal([]byte(jsonPart), &report); err != nil {
		t.Fatalf("JSON inválido: %v (line=%q)", err, jsonPart)
	}
	if report.TotalEvents != 3 {
		t.Errorf("total_events=%d; esperado 3", report.TotalEvents)
	}
	if report.UnackedCount != 1 {
		t.Errorf("unacked_count=%d; esperado 1", report.UnackedCount)
	}
	if report.OldestUnackedAgeSeconds < 0 {
		t.Errorf("oldest_unacked_age_seconds=%d; esperado >= 0", report.OldestUnackedAgeSeconds)
	}
}

// TestDoctor_EventsFlag_WarnsWhenOldestUnackedTooOld: setup 1 unacked event
// with backdated created_at; assert the WARN line is printed.
func TestDoctor_EventsFlag_WarnsWhenOldestUnackedTooOld(t *testing.T) {
	rootDir := seedEventLogForDoctor(t, 0, 1)

	// Update the unacked event's created_at to > 5 minutes ago.
	dbPath := filepath.Join(rootDir, ".memory", "memory.db")
	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	old := time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339Nano)
	if _, err := database.ExecContext(context.Background(),
		`UPDATE event_log SET created_at = ? WHERE acked_at IS NULL`, old); err != nil {
		database.Close()
		t.Fatalf("UPDATE: %v", err)
	}
	database.Close()

	stdout, stderr, exit := runMemDoctor(t, rootDir, "doctor", "--events")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, "WARN") {
		t.Errorf("stdout deveria conter WARN (oldest_unacked > 5min): %s", stdout)
	}
}

// TestDoctor_NoEventsFlag_NoEventsSection: `mem doctor` (without --events)
// must NOT print the Events section.
func TestDoctor_NoEventsFlag_NoEventsSection(t *testing.T) {
	rootDir := seedEventLogForDoctor(t, 1, 0)
	stdout, _, exit := runMemDoctor(t, rootDir, "doctor")
	if exit != 0 {
		t.Fatalf("exit não-zero")
	}
	if strings.Contains(stdout, "[📨 Events") {
		t.Errorf("stdout contém seção Events sem --events: %s", stdout)
	}
	if strings.Contains(stdout, "[json]") {
		t.Errorf("stdout contém linha JSON sem --events: %s", stdout)
	}
}
