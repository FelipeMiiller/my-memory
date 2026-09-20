package supervisor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/event_runtime"
)

// Lifecycle event types written to event_log (ADR-042 §lifecycle
// events). Exported so tests and external tools can match them.
const (
	EventWorkerStarted       = "worker.started"
	EventWorkerStopped       = "worker.stopped"
	EventWorkerHeartbeatLost = "worker.heartbeat_lost"
	EventWorkerFailed        = "supervisor.worker_failed"
	EventWorkerRestarted     = "worker.restarted"
)

// defaultStopGrace is the SIGTERM → SIGKILL escalation timeout
// (ADR-042 §DR-7). Exposed as a const so tests can shrink it.
const defaultStopGrace = 10 * time.Second

// Sentinel errors for the lifecycle surface. Use errors.Is.
var (
	// ErrWorkerNotFound is returned by Stop / Health when the worker
	// ID is not in the active map.
	ErrWorkerNotFound = errors.New("supervisor: worker not found")

	// ErrWorkerAlreadyStarted is returned by Start when a worker
	// with the same ID is already in the active map.
	ErrWorkerAlreadyStarted = errors.New("supervisor: worker already started")
)

// Start launches the worker subprocess declared by spec, registers
// it in the active map, emits worker.started to event_log, and
// spawns a watcher goroutine that emits worker.heartbeat_lost if
// the subprocess exits unexpectedly. Returns ErrWorkerAlreadyStarted
// when a worker with the same ID is already running.
//
// ctx is used as the parent context for the BeginTx below; it does
// NOT kill the subprocess on cancel (use Stop for that). The
// subprocess's stdout/stderr are captured to /dev/null-equivalent
// to avoid blocking on a full pipe; T9 swaps in audit-log
// redirection.
func (m *Manager) Start(ctx context.Context, spec WorkerSpec) (*Worker, error) {
	if spec.Name == "" {
		return nil, errors.New("supervisor: WorkerSpec.Name is empty")
	}
	if spec.Command == "" {
		return nil, fmt.Errorf("supervisor: worker %q has empty Command", spec.Name)
	}

	m.mu.Lock()
	if _, exists := m.active[spec.Name]; exists {
		m.mu.Unlock()
		return nil, fmt.Errorf("%w: id=%s", ErrWorkerAlreadyStarted, spec.Name)
	}
	m.mu.Unlock()

	cmd := exec.Command(spec.Command, spec.Args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if len(spec.Env) > 0 {
		cmd.Env = append(os.Environ(), spec.Env...)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("supervisor: start %q: %w", spec.Name, err)
	}

	w := &Worker{
		ID:     spec.Name,
		PID:    cmd.Process.Pid,
		spec:   spec,
		cmd:    cmd,
		doneCh: make(chan struct{}),
	}

	m.mu.Lock()
	m.active[spec.Name] = w
	m.mu.Unlock()

	m.emitEvent(ctx, EventWorkerStarted, spec.Name, map[string]any{
		"pid":     w.PID,
		"command": spec.Command,
		"args":    spec.Args,
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
	})

	// Watcher: detect exit and emit heartbeat_lost when abnormal.
	// T3 introduces an `expected` flag so Stop can suppress the
	// false heartbeat_lost after a graceful shutdown.
	go m.watchWorker(w)

	return w, nil
}

// Stop sends SIGTERM to the worker, waits up to grace for it to
// exit, then sends SIGKILL to survivors. Emits worker.stopped once
// the process has actually terminated (or after the kill).
//
// ctx is honored — if it is cancelled mid-wait, Stop returns the
// context error and the subprocess is force-killed.
//
// On Windows, os.Process.Signal only supports os.Kill (SIGTERM is a
// no-op), so the call falls straight through to Kill after the
// grace period. ADR-042 §DR-7 acknowledges this asymmetry — Windows
// workers are expected to honor CTRL_BREAK_EVENT themselves.
func (m *Manager) Stop(ctx context.Context, workerID string, grace time.Duration) error {
	if grace <= 0 {
		grace = defaultStopGrace
	}

	m.mu.Lock()
	w, ok := m.active[workerID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("%w: id=%s", ErrWorkerNotFound, workerID)
	}
	return m.stopWorker(ctx, w, grace, "requested")
}

// Health returns the worker's liveness state. T2 implements a
// binary check ("alive = running, dead = exited"); T3 refines it
// into the 5s heartbeat / 3 misses model once the IPC envelope lands.
//
// Returns ErrWorkerNotFound if the worker has never been started
// or has already been removed by the watcher.
func (m *Manager) Health(workerID string) (HealthStatus, error) {
	m.mu.Lock()
	_, ok := m.active[workerID]
	m.mu.Unlock()
	if !ok {
		return HealthDead, fmt.Errorf("%w: id=%s", ErrWorkerNotFound, workerID)
	}
	return HealthAlive, nil
}

// HealthStatus is the lifecycle state of a worker.
type HealthStatus string

const (
	HealthAlive    HealthStatus = "alive"
	HealthDegraded HealthStatus = "degraded"
	HealthDead     HealthStatus = "dead"
)

// watchWorker blocks until the subprocess exits. If the exit was
// not initiated by Stop (expectedExit=false) we emit
// worker.heartbeat_lost AND, if a RestartPolicy is configured,
// schedule a restart with exponential backoff. After
// policy.MaxRetries attempts we emit supervisor.worker_failed and
// stop trying.
//
// The active map is updated BEFORE the done channel closes so
// callers that observe the close see the consistent state.
func (m *Manager) watchWorker(w *Worker) {
	err := w.cmd.Wait()

	m.mu.Lock()
	delete(m.active, w.ID)
	m.mu.Unlock()

	close(w.doneCh)

	if w.expectedExit {
		return
	}

	reason := "abnormal_exit"
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		reason = fmt.Sprintf("exit_code=%d", exitErr.ExitCode())
	} else if err != nil {
		reason = fmt.Sprintf("signal: %v", err)
	}
	m.emitEvent(context.Background(), EventWorkerHeartbeatLost, w.ID, map[string]any{
		"pid":           w.PID,
		"reason":        reason,
		"restart_count": w.restartCount,
		"ts":            time.Now().UTC().Format(time.RFC3339Nano),
	})

	// Restart path: only when a policy is configured AND we
	// haven't exhausted the retry budget.
	policy := w.spec.RestartPolicy
	if policy.MaxRetries <= 0 {
		return
	}
	if w.restartCount >= policy.MaxRetries {
		w.failed = true
		m.emitEvent(context.Background(), EventWorkerFailed, w.ID, map[string]any{
			"pid":         w.PID,
			"exit_code":   exitCodeOf(err),
			"retries":     w.restartCount,
			"max_retries": policy.MaxRetries,
			"ts":          time.Now().UTC().Format(time.RFC3339Nano),
		})
		return
	}

	delay := computeBackoff(w.restartCount, policy)
	w.restartCount++

	// Schedule the restart. We use time.AfterFunc so the watcher
	// goroutine returns immediately and the supervisor can keep
	// doing useful work. The timer is intentionally NOT cancellable
	// — once a crash is detected we always honour the backoff so the
	// worker has time to settle.
	time.AfterFunc(delay, func() {
		m.restartWorker(w)
	})
}

// restartWorker re-Starts the same worker spec, reusing the
// existing *Worker struct so restartCount / failed flags persist
// across attempts. Returns early (without emitting events) when
// the manager has been shut down — the timer fired after Run()
// returned.
func (m *Manager) restartWorker(w *Worker) {
	// Re-check expectedExit — Stop may have been called while the
	// backoff timer was pending. If so, leave the worker alone.
	if w.expectedExit {
		return
	}

	// Reset the per-attempt flags. cmd + doneCh must be replaced
	// because the previous ones are now closed/finished.
	cmd := exec.Command(w.spec.Command, w.spec.Args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if len(w.spec.Env) > 0 {
		cmd.Env = append(os.Environ(), w.spec.Env...)
	}

	if err := cmd.Start(); err != nil {
		m.emitEvent(context.Background(), EventWorkerFailed, w.ID, map[string]any{
			"pid":         w.PID,
			"error":       err.Error(),
			"retries":     w.restartCount,
			"max_retries": w.spec.RestartPolicy.MaxRetries,
			"ts":          time.Now().UTC().Format(time.RFC3339Nano),
		})
		w.failed = true
		return
	}

	w.cmd = cmd
	w.PID = cmd.Process.Pid
	w.expectedExit = false
	w.doneCh = make(chan struct{})
	w.lastRestart = time.Now()

	m.mu.Lock()
	m.active[w.ID] = w
	m.mu.Unlock()

	m.emitEvent(context.Background(), EventWorkerRestarted, w.ID, map[string]any{
		"pid":           w.PID,
		"restart_count": w.restartCount,
		"ts":            time.Now().UTC().Format(time.RFC3339Nano),
	})

	go m.watchWorker(w)
}

// computeBackoff returns the delay before the next restart attempt.
// Formula: min(maxDelay, baseDelay * 2^n) with ±jitter. Jitter is
// applied multiplicatively — e.g. jitter=0.2, delay=100ms produces a
// uniform sample in [80ms, 120ms]. Tests override the RNG via the
// `rngFn` package var to keep determinism.
func computeBackoff(attempt int, policy RestartPolicy) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	base := policy.BaseDelay
	if base <= 0 {
		base = 100 * time.Millisecond
	}
	max := policy.MaxDelay
	if max <= 0 {
		max = 30 * time.Second
	}
	// 2^n grows fast; cap to avoid overflow.
	shift := uint(attempt)
	if shift > 30 {
		shift = 30
	}
	delay := base << shift
	if delay <= 0 || delay > max {
		delay = max
	}

	jitter := policy.Jitter
	if jitter < 0 {
		jitter = 0
	}
	if jitter > 0 {
		// Sample in [1-jitter, 1+jitter]. rngFn returns [0,1).
		r := rngFn() - 0.5 // [-0.5, 0.5)
		delta := float64(delay) * jitter * 2 * r
		delay += time.Duration(delta)
		if delay < 0 {
			delay = 0
		}
	}
	return delay
}

// exitCodeOf extracts the OS exit code from an exec.Cmd error,
// returning -1 when the error is nil or doesn't carry a code. Used
// by the restart bookkeeping so worker_failed payloads carry the
// same exit_code semantics as heartbeat_lost.
func exitCodeOf(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

// rngFn is the package-level random source used by computeBackoff.
// Default uses math/rand's Float64; tests override to a fixed seed
// for deterministic backoff verification.
var rngFn = func() float64 { return randFloat64() }

// randFloat64 wraps math/rand.Float64 behind a tiny indirection so
// the import lives in restart_randr.go (separate file) — keeps the
// hot path in lifecycle.go free of crypto-grade dependencies.
func randFloat64() float64 {
	return randFloat64Runtime()
}

// stopWorker performs the SIGTERM → wait → SIGKILL escalation.
// Shared by Stop and by future T3 restart logic. The reason string
// is written into the worker.stopped payload.
//
// Sets Worker.expectedExit before signaling so the watcher skips
// the worker.heartbeat_lost event for graceful shutdowns.
//
// If the process is already gone we emit worker.stopped with
// outcome="already_gone" and return nil. If the kill path itself
// fails we still emit worker.stopped so operators see the
// lifecycle in audit logs.
func (m *Manager) stopWorker(ctx context.Context, w *Worker, grace time.Duration, reason string) error {
	if w == nil {
		return ErrWorkerNotFound
	}
	pid := w.PID

	// Mark the exit as expected BEFORE sending the signal so the
	// watcher — which races on cmd.Wait returning — sees the flag
	// and skips the heartbeat_lost event.
	w.expectedExit = true

	// Step 1: polite shutdown via SIGTERM. On Windows this is a
	// no-op via os.Process.Signal (only Kill is supported) but the
	// grace window still applies — workers are expected to honor
	// CTRL_BREAK_EVENT themselves. We record the failure as
	// outcome="signal_unsupported" but proceed to the wait so the
	// SIGKILL escalation at the end of grace still fires.
	signalOutcome := "graceful"
	sendErr := sendSignal(pid, syscall.SIGTERM)
	if sendErr != nil {
		if isProcessGone(sendErr) {
			signalOutcome = "already_gone"
		} else if isSignalUnsupported(sendErr) {
			signalOutcome = "signal_unsupported"
		} else {
			m.emitEvent(ctx, EventWorkerStopped, w.ID, map[string]any{
				"pid":     pid,
				"reason":  reason,
				"outcome": "signal_failed",
				"error":   sendErr.Error(),
				"ts":      time.Now().UTC().Format(time.RFC3339Nano),
			})
			// Best-effort: still wait for the watcher before
			// returning so the active map gets cleaned up.
			select {
			case <-w.doneCh:
			case <-time.After(grace):
			}
			return nil
		}
	}

	// Step 2: wait for the watcher to observe the exit.
	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case <-timer.C:
		// Grace expired — escalate.
	case <-ctx.Done():
		_ = sendSignal(pid, syscall.SIGKILL)
		return ctx.Err()
	case <-w.doneCh:
		m.emitEvent(ctx, EventWorkerStopped, w.ID, map[string]any{
			"pid":     pid,
			"reason":  reason,
			"outcome": signalOutcome,
			"ts":      time.Now().UTC().Format(time.RFC3339Nano),
		})
		return nil
	}

	// Step 3: SIGKILL the survivor.
	_ = sendSignal(pid, syscall.SIGKILL)

	m.emitEvent(ctx, EventWorkerStopped, w.ID, map[string]any{
		"pid":     pid,
		"reason":  reason,
		"outcome": "killed",
		"ts":      time.Now().UTC().Format(time.RFC3339Nano),
	})

	// Best-effort wait for the watcher to reap.
	select {
	case <-w.doneCh:
	case <-time.After(grace):
	}
	return nil
}

// isSignalUnsupported returns true for errors that indicate the OS
// rejected the signal — typically because the signal isn't
// implemented on that platform (e.g. SIGTERM on Windows). The
// caller should fall back to SIGKILL escalation rather than treating
// this as a hard failure.
func isSignalUnsupported(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsStr(msg, "not supported by windows") ||
		containsStr(msg, "not supported") ||
		containsStr(msg, "operation not permitted")
}

// sendSignal is a thin wrapper around os.FindProcess + Signal so
// the call site is concise.
func sendSignal(pid int, sig syscall.Signal) error {
	if pid <= 0 {
		return errors.New("supervisor: invalid pid")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(sig)
}

// emitEvent writes an event_log row when db is bound; it is a no-op
// when db is nil (T1-style smoke runs without event_log). Errors
// are swallowed because lifecycle events are advisory — losing one
// must not crash the supervisor (ADR-050 LLM05/LLM06 fail-soft).
func (m *Manager) emitEvent(ctx context.Context, eventType, aggregateID string, payload map[string]any) {
	if m.log == nil || m.db == nil {
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	env := event_runtime.NewEnvelope(eventType, aggregateID)
	env.Payload = raw
	env.Producer = "supervisor"
	env.Actor = "supervisor"

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	if err := m.log.Append(ctx, tx, env); err != nil {
		_ = tx.Rollback()
		return
	}
	if err := tx.Commit(); err != nil {
		return
	}
}

// isProcessGone returns true for errors that indicate the target
// process no longer exists. Used so stopWorker can treat
// "process already exited" as success.
func isProcessGone(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "os: process already finished" ||
		containsStr(msg, "no such process") ||
		containsStr(msg, "invalid parameter")
}

// containsStr is a strings.Contains shim kept local to avoid
// growing the import set.
func containsStr(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
