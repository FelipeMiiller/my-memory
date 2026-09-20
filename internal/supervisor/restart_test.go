package supervisor

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// countEventTypeByType returns the number of event_log rows whose
// event_type matches et. Used by restart tests to assert "how many
// restarts happened" without coupling to ordering.
func countEventTypeByType(t *testing.T, db *sql.DB, et string) int {
	t.Helper()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM event_log WHERE event_type = ?", et).Scan(&n); err != nil {
		t.Fatalf("count %q events: %v", et, err)
	}
	return n
}

// pollUntil polls cond every 25ms until it returns true or timeout
// elapses. Returns true on success.
func pollUntil(t *testing.T, timeout time.Duration, cond func() bool) bool {
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

// fixedRNG returns a closure suitable for assigning to rngFn that
// always yields the given value. Tests use this to pin the jitter
// contribution so backoff is reproducible.
func fixedRNG(v float64) func() float64 {
	return func() float64 { return v }
}

// ---------------- Tests ----------------

func TestRestart_DisabledByDefault(t *testing.T) {
	db := newTestDB(t)
	mgr := NewManager(t.TempDir(), "default", db)

	// Spec WITHOUT a RestartPolicy — MaxRetries=0 means no restart.
	// The watcher still emits heartbeat_lost but never schedules
	// another Start().
	_, err := mgr.Start(context.Background(), WorkerSpec{
		Name:    "noretry",
		Command: fakeWorkerPath,
		Env:     []string{"FAKEWORKER_EXIT_AFTER_MS=100"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if !pollUntil(t, 3*time.Second, func() bool {
		return countEventTypeByType(t, db, EventWorkerHeartbeatLost) >= 1
	}) {
		t.Fatal("heartbeat_lost never emitted")
	}

	// Give plenty of time for any spurious restart to fire. None
	// should because policy.MaxRetries is zero.
	time.Sleep(500 * time.Millisecond)
	if got := countEventTypeByType(t, db, EventWorkerStarted); got != 1 {
		t.Fatalf("expected exactly 1 worker.started (no retries), got %d", got)
	}
	if got := countEventTypeByType(t, db, EventWorkerRestarted); got != 0 {
		t.Fatalf("expected 0 worker.restarted when policy disabled, got %d", got)
	}
}

func TestRestart_ExponentialBackoff(t *testing.T) {
	db := newTestDB(t)
	mgr := NewManager(t.TempDir(), "default", db)

	policy := RestartPolicy{
		MaxRetries: 3,
		BaseDelay:  20 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		Jitter:     0,
	}
	prevRNG := rngFn
	rngFn = fixedRNG(0.5)
	defer func() { rngFn = prevRNG }()

	_, err := mgr.Start(context.Background(), WorkerSpec{
		Name:          "expo",
		Command:       fakeWorkerPath,
		Env:           []string{"FAKEWORKER_EXIT_AFTER_MS=50"},
		RestartPolicy: policy,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Expect ≥3 worker.restarted events. Total wall time is bounded
	// by 50ms (fakeworker exit) + 20ms*3 (backoff for retries 1..3)
	// ≈ 350ms.
	if !pollUntil(t, 5*time.Second, func() bool {
		return countEventTypeByType(t, db, EventWorkerRestarted) >= 3
	}) {
		t.Fatalf("expected ≥3 worker.restarted, got %d",
			countEventTypeByType(t, db, EventWorkerRestarted))
	}

	// worker_failed must NOT have fired — we're still under the
	// retry budget.
	if got := countEventTypeByType(t, db, EventWorkerFailed); got != 0 {
		t.Fatalf("unexpected worker_failed event (got %d)", got)
	}
}

func TestRestart_MaxRetriesThenFailed(t *testing.T) {
	db := newTestDB(t)
	mgr := NewManager(t.TempDir(), "default", db)

	policy := RestartPolicy{
		MaxRetries: 5,
		BaseDelay:  10 * time.Millisecond,
		MaxDelay:   50 * time.Millisecond,
		Jitter:     0,
	}
	prevRNG := rngFn
	rngFn = fixedRNG(0.5)
	defer func() { rngFn = prevRNG }()

	_, err := mgr.Start(context.Background(), WorkerSpec{
		Name:          "failandstop",
		Command:       fakeWorkerPath,
		Env:           []string{"FAKEWORKER_EXIT_AFTER_MS=30"},
		RestartPolicy: policy,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if !pollUntil(t, 5*time.Second, func() bool {
		return countEventTypeByType(t, db, EventWorkerFailed) >= 1
	}) {
		t.Fatalf("worker.failed never emitted; restarts=%d",
			countEventTypeByType(t, db, EventWorkerRestarted))
	}

	if got := countEventTypeByType(t, db, EventWorkerRestarted); got < 5 {
		t.Fatalf("expected at least 5 restarts before giving up, got %d", got)
	}

	// Give extra time to make sure no spurious restart fires
	// after worker_failed.
	time.Sleep(200 * time.Millisecond)
	if got := countEventTypeByType(t, db, EventWorkerRestarted); got > 6 {
		t.Fatalf("too many restarts after failure (%d) — restart loop didn't stop", got)
	}
}

func TestRestart_GracefulStopSuppressesRestart(t *testing.T) {
	db := newTestDB(t)
	mgr := NewManager(t.TempDir(), "default", db)

	policy := RestartPolicy{
		MaxRetries: 5,
		BaseDelay:  50 * time.Millisecond,
		MaxDelay:   100 * time.Millisecond,
		Jitter:     0,
	}
	prevRNG := rngFn
	rngFn = fixedRNG(0.5)
	defer func() { rngFn = prevRNG }()

	w, err := mgr.Start(context.Background(), WorkerSpec{
		Name:          "graceful",
		Command:       fakeWorkerPath,
		RestartPolicy: policy,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := mgr.Stop(context.Background(), w.ID, 1*time.Second); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	if got := countEventTypeByType(t, db, EventWorkerRestarted); got != 0 {
		t.Fatalf("graceful Stop should not trigger restart, got %d", got)
	}
	if got := countEventTypeByType(t, db, EventWorkerFailed); got != 0 {
		t.Fatalf("graceful Stop should not trigger worker_failed, got %d", got)
	}
}

func TestRestart_ComputeBackoffMonotonic(t *testing.T) {
	prevRNG := rngFn
	rngFn = fixedRNG(0.5) // delta = 0 (rng - 0.5 = 0)
	defer func() { rngFn = prevRNG }()

	policy := RestartPolicy{
		MaxRetries: 10,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   30 * time.Second,
		Jitter:     0,
	}

	expected := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1600 * time.Millisecond,
		3200 * time.Millisecond,
		6400 * time.Millisecond,
		12800 * time.Millisecond,
		25600 * time.Millisecond,
		30000 * time.Millisecond, // capped at MaxDelay
	}
	for i, want := range expected {
		got := computeBackoff(i, policy)
		if got != want {
			t.Fatalf("backoff[%d] = %v, want %v", i, got, want)
		}
	}
}

func TestRestart_ComputeBackoffCapRespected(t *testing.T) {
	prevRNG := rngFn
	rngFn = fixedRNG(0.5)
	defer func() { rngFn = prevRNG }()

	policy := RestartPolicy{
		MaxRetries: 100,
		BaseDelay:  1 * time.Second,
		MaxDelay:   5 * time.Second,
		Jitter:     0,
	}
	for i := 0; i < 100; i++ {
		got := computeBackoff(i, policy)
		if got > 5*time.Second {
			t.Fatalf("backoff[%d] = %v exceeds cap 5s", i, got)
		}
	}
}

func TestRestart_DefaultPolicySensible(t *testing.T) {
	p := DefaultRestartPolicy()
	if p.MaxRetries != 5 {
		t.Fatalf("default MaxRetries = %d, want 5", p.MaxRetries)
	}
	if p.BaseDelay != 100*time.Millisecond {
		t.Fatalf("default BaseDelay = %v, want 100ms", p.BaseDelay)
	}
	if p.MaxDelay != 30*time.Second {
		t.Fatalf("default MaxDelay = %v, want 30s", p.MaxDelay)
	}
	if p.Jitter != 0.2 {
		t.Fatalf("default Jitter = %v, want 0.2", p.Jitter)
	}
}

func TestRestart_JitterSpreadWithinBounds(t *testing.T) {
	policy := RestartPolicy{
		MaxRetries: 1,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   1 * time.Second,
		Jitter:     0.2,
	}
	prevRNG := rngFn
	defer func() { rngFn = prevRNG }()

	// Sample jitter extremes.
	for _, sample := range []float64{0.0, 0.5, 0.99} {
		rngFn = fixedRNG(sample)
		got := computeBackoff(0, policy)
		// Jitter range is [base * (1 - jitter), base * (1 + jitter)]
		// = [80ms, 120ms].
		if got < 80*time.Millisecond || got > 120*time.Millisecond {
			t.Fatalf("jitter sample %.2f produced %v, want within [80ms, 120ms]", sample, got)
		}
	}
}
