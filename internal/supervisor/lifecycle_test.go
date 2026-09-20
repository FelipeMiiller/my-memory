package supervisor

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// fakeWorkerPath is the absolute path to the compiled test helper
// binary, populated by TestMain. Tests reference this path as the
// WorkerSpec.Command for Start(). The binary is built once per test
// process and shared across all tests.
var fakeWorkerPath string

// TestMain compiles the fakeworker test helper once and stashes the
// resulting binary in a per-process temp directory. The compilation
// costs ~1s on a cold cache and is amortized across the test suite,
// so individual lifecycle tests stay under 100ms.
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "supervisor-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	exe := "fakeworker"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	fakeWorkerPath = filepath.Join(tmpDir, exe)

	srcDir, err := filepath.Abs("testdata/fakeworker")
	if err != nil {
		panic(err)
	}
	cmd := exec.Command("go", "build", "-o", fakeWorkerPath, srcDir)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build fakeworker: " + err.Error())
	}

	os.Exit(m.Run())
}

// newTestDB opens a fresh in-memory SQLite with the minimal event_log
// surface the supervisor needs. We use the pure-Go modernc.org/sqlite
// driver so tests run under CGO_ENABLED=0 (consistent with the
// quality gate baseline). T9's audit log may add additional tables.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, stmt := range []string{
		`CREATE TABLE event_log (
			sequence INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id TEXT UNIQUE NOT NULL,
			schema_version INTEGER NOT NULL,
			event_type TEXT NOT NULL,
			aggregate_id TEXT NOT NULL,
			revision INTEGER,
			payload BLOB NOT NULL,
			headers BLOB NOT NULL,
			created_at TEXT NOT NULL,
			acked_at TEXT
		)`,
		`CREATE TABLE projection_cursor (
			projection_name TEXT PRIMARY KEY,
			last_sequence INTEGER NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("create table: %v\nstmt=%s", err, stmt)
		}
	}
	return db
}

// eventTypesFromDB returns every event_type currently in event_log,
// used to assert lifecycle events without coupling to ordering.
func eventTypesFromDB(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query("SELECT DISTINCT event_type FROM event_log ORDER BY event_type")
	if err != nil {
		t.Fatalf("query event_log: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var et string
		if err := rows.Scan(&et); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, et)
	}
	return out
}

// waitFor polls cond every 25ms until it returns true or timeout
// elapses. Returns true on success, false on timeout. Tests use this
// to absorb the asynchronous lifecycle (subprocess start, watcher
// goroutine) without sleeping for arbitrary durations.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return cond()
}

// ---------------- Tests ----------------

func TestStart_RejectsEmptyName(t *testing.T) {
	mgr := NewManager(t.TempDir(), "default", newTestDB(t))
	_, err := mgr.Start(context.Background(), WorkerSpec{Name: "", Command: fakeWorkerPath})
	if err == nil {
		t.Fatal("expected error on empty Name, got nil")
	}
	if !strings.Contains(err.Error(), "Name") {
		t.Fatalf("expected error mentioning Name, got %v", err)
	}
}

func TestStart_RejectsEmptyCommand(t *testing.T) {
	mgr := NewManager(t.TempDir(), "default", newTestDB(t))
	_, err := mgr.Start(context.Background(), WorkerSpec{Name: "x"})
	if err == nil {
		t.Fatal("expected error on empty Command, got nil")
	}
}

func TestStart_EmitsStartedEvent(t *testing.T) {
	db := newTestDB(t)
	mgr := NewManager(t.TempDir(), "default", db)

	w, err := mgr.Start(context.Background(), WorkerSpec{
		Name:    "alpha",
		Command: fakeWorkerPath,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if w.PID <= 0 {
		t.Fatalf("expected PID > 0, got %d", w.PID)
	}

	ok := waitFor(t, 2*time.Second, func() bool {
		types := eventTypesFromDB(t, db)
		for _, et := range types {
			if et == EventWorkerStarted {
				return true
			}
		}
		return false
	})
	if !ok {
		t.Fatalf("worker.started event not emitted within 2s; events=%v",
			eventTypesFromDB(t, db))
	}
}

func TestStart_RefusesDuplicateID(t *testing.T) {
	mgr := NewManager(t.TempDir(), "default", newTestDB(t))
	spec := WorkerSpec{Name: "alpha", Command: fakeWorkerPath}

	if _, err := mgr.Start(context.Background(), spec); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	_, err := mgr.Start(context.Background(), spec)
	if err == nil {
		t.Fatal("expected ErrWorkerAlreadyStarted on duplicate ID")
	}
	if !errors.Is(err, ErrWorkerAlreadyStarted) {
		t.Fatalf("expected ErrWorkerAlreadyStarted, got %v", err)
	}
}

func TestStop_EmitsStoppedEvent(t *testing.T) {
	db := newTestDB(t)
	mgr := NewManager(t.TempDir(), "default", db)

	w, err := mgr.Start(context.Background(), WorkerSpec{
		Name:    "beta",
		Command: fakeWorkerPath,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Tighter grace than production default so the test stays fast.
	if err := mgr.Stop(context.Background(), w.ID, 3*time.Second); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	ok := waitFor(t, 2*time.Second, func() bool {
		types := eventTypesFromDB(t, db)
		for _, et := range types {
			if et == EventWorkerStopped {
				return true
			}
		}
		return false
	})
	if !ok {
		t.Fatalf("worker.stopped event not emitted within 2s; events=%v",
			eventTypesFromDB(t, db))
	}
}

func TestStop_UnknownIDReturnsError(t *testing.T) {
	mgr := NewManager(t.TempDir(), "default", newTestDB(t))
	err := mgr.Stop(context.Background(), "ghost", 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected error on unknown ID")
	}
	if !errors.Is(err, ErrWorkerNotFound) {
		t.Fatalf("expected ErrWorkerNotFound, got %v", err)
	}
}

func TestWorker_HeartbeatLostOnCrash(t *testing.T) {
	db := newTestDB(t)
	mgr := NewManager(t.TempDir(), "default", db)

	// fakeworker reads FAKEWORKER_EXIT_AFTER_MS and exits 0 after
	// that delay. From the supervisor's perspective this is an
	// abnormal exit (the worker is expected to keep running), so
	// the watcher must emit worker.heartbeat_lost.
	_, err := mgr.Start(context.Background(), WorkerSpec{
		Name:    "gamma",
		Command: fakeWorkerPath,
		Env:     []string{"FAKEWORKER_EXIT_AFTER_MS=200"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	ok := waitFor(t, 3*time.Second, func() bool {
		types := eventTypesFromDB(t, db)
		for _, et := range types {
			if et == EventWorkerHeartbeatLost {
				return true
			}
		}
		return false
	})
	if !ok {
		t.Fatalf("worker.heartbeat_lost event not emitted within 3s; events=%v",
			eventTypesFromDB(t, db))
	}
}

// TestWorker_StartStopRoundtrip — spec-mandated integration test.
// Boots the fakeworker, stops it, and asserts two lifecycle events
// landed in event_log (worker.started + worker.stopped).
func TestWorker_StartStopRoundtrip(t *testing.T) {
	db := newTestDB(t)
	mgr := NewManager(t.TempDir(), "default", db)

	w, err := mgr.Start(context.Background(), WorkerSpec{
		Name:    "roundtrip",
		Command: fakeWorkerPath,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := mgr.Stop(context.Background(), w.ID, 3*time.Second); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	ok := waitFor(t, 3*time.Second, func() bool {
		types := eventTypesFromDB(t, db)
		hasStart, hasStop := false, false
		for _, et := range types {
			if et == EventWorkerStarted {
				hasStart = true
			}
			if et == EventWorkerStopped {
				hasStop = true
			}
		}
		return hasStart && hasStop
	})
	if !ok {
		t.Fatalf("missing lifecycle events; events=%v", eventTypesFromDB(t, db))
	}
}

// TestHealth_AfterStopReturnsNotFound asserts the watcher removes
// the worker from the active map once cmd.Wait returns.
func TestHealth_AfterStopReturnsNotFound(t *testing.T) {
	mgr := NewManager(t.TempDir(), "default", newTestDB(t))
	w, err := mgr.Start(context.Background(), WorkerSpec{Name: "delta", Command: fakeWorkerPath})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := mgr.Stop(context.Background(), w.ID, 3*time.Second); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if !waitFor(t, 2*time.Second, func() bool {
		_, err := mgr.Health(w.ID)
		return errors.Is(err, ErrWorkerNotFound)
	}) {
		t.Fatal("Health still reports worker as known after stop")
	}
}

// silence unused-import warnings from the imports we'll need once
// the heartbeat tests expand (T3). Keeps the file green when those
// tests are temporarily skipped above.
var _ sync.Once
