// Package event_runtime — dispatcher.go
//
// In-process fan-out engine for the event_log. T7 fills the DeliveryLoop
// that T6's registry stub was waiting for.
//
// Responsibilities (ADR-043 §5 + §6):
//   - Poll event_log for unacked envelopes for each registered subscriber.
//   - Match envelopes against subscriber.EventTypes() patterns.
//   - Call Subscriber.Handle in a worker goroutine per subscriber.
//   - On nil return → MarkAcked(seq, ""), advance projection_cursor.
//   - On error → exponential backoff retry (base 100ms, factor 2, jitter ±20%, cap 30s).
//   - On panic → recover, log, treat as transient.
//   - On schema_version > CurrentSchemaVersion → permanent ACK with reason.
//   - Honor MaxAckPending backpressure: pause reads when in-flight count
//     reaches the per-subscriber cap (ADR-050 LLM10).
//   - Stop(ctx) waits up to gracePeriod (default 30s) for in-flight
//     handlers; on timeout, force-cancels and leaves in-flight unacked.
package event_runtime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"path"
	"sync"
	"sync/atomic"
	"time"
)

// Backoff parameters (ADR-043 §6).
const (
	backoffBase   = 100 * time.Millisecond
	backoffFactor = 2.0
	backoffCap    = 30 * time.Second
	jitterPercent = 0.2 // ±20%
)

// DefaultPollInterval is how often an idle worker (no unacked events) wakes
// up to re-check the event_log. The spec recommends 100ms.
const DefaultPollInterval = 100 * time.Millisecond

// DefaultGracePeriod is how long Stop waits for in-flight handlers to drain
// before force-cancelling.
const DefaultGracePeriod = 30 * time.Second

// subState tracks one subscriber's delivery goroutine + in-flight count +
// per-event retry attempts (in-memory only).
type subState struct {
	sub          Subscriber
	patterns     []string
	cancel       context.CancelFunc
	done         chan struct{}
	inflight     int32 // atomic
	retryCounts  map[int64]int
	retryCountsM sync.Mutex
}

// IsReady reports whether the dispatcher has been wired (Start called).
func (d *Dispatcher) IsReady() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db != nil && d.log != nil && d.pollInterval > 0
}

// Attach wires the DB, Log, poll interval and grace period onto an existing
// Dispatcher. Idempotent: calling Attach twice with the same args is a
// no-op (overwrites with the same values).
func (d *Dispatcher) Attach(db *sql.DB, log *Log, pollInterval, gracePeriod time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.db = db
	d.log = log
	if pollInterval <= 0 {
		pollInterval = DefaultPollInterval
	}
	d.pollInterval = pollInterval
	if gracePeriod <= 0 {
		gracePeriod = DefaultGracePeriod
	}
	d.gracePeriod = gracePeriod
}

// Start spawns one worker goroutine per registered subscriber. Each worker
// polls event_log for unacked envelopes matching the subscriber's patterns
// and delivers them through Handle. Already-registered subscribers are
// picked up; subscribers registered AFTER Start require Restart to spawn
// a goroutine (T7 keeps this simple — see Start once, register all).
func (d *Dispatcher) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.started {
		d.mu.Unlock()
		return errors.New("event_runtime: dispatcher already started")
	}
	if d.db == nil || d.log == nil {
		d.mu.Unlock()
		return ErrEventLogNotInitialized
	}
	if !eventLogTableExists(ctx, d.db) {
		d.mu.Unlock()
		return ErrEventLogNotInitialized
	}
	d.started = true
	d.mu.Unlock()

	// Snapshot the subscriber list under lock, then start workers outside.
	d.mu.RLock()
	snapshot := make([]Subscriber, 0, len(d.subs))
	for _, sub := range d.subs {
		snapshot = append(snapshot, sub)
	}
	d.mu.RUnlock()

	for _, sub := range snapshot {
		d.spawnWorker(ctx, sub)
	}
	return nil
}

// Stop signals every worker to drain. Waits up to gracePeriod for in-flight
// handlers (dispatchOne goroutines) to complete; on timeout, force-cancels
// and leaves any unacked events on disk so they get re-delivered on next
// Start.
//
// Calling Stop twice is safe: the second call is a no-op.
func (d *Dispatcher) Stop(ctx context.Context) error {
	d.mu.Lock()
	if !d.started {
		d.mu.Unlock()
		return nil
	}
	if d.stopped {
		d.mu.Unlock()
		return nil
	}
	d.stopped = true
	stopCh := d.stopCh
	workers := make([]*subState, 0, len(d.subStates))
	for _, s := range d.subStates {
		workers = append(workers, s)
	}
	d.mu.Unlock()

	close(stopCh)

	// Compute deadline: min(gracePeriod, ctx.Deadline).
	timeout := d.gracePeriod
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}

	// Wait for ALL goroutines (workers + in-flight dispatchOnes) via d.wg.
	doneCh := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		return nil
	case <-time.After(timeout):
		// Force-cancel worker contexts; cascades to dispatchOnes' parentCtx.
		for _, s := range workers {
			s.cancel()
		}
		// Give them a brief grace to actually unwind.
		select {
		case <-doneCh:
		case <-time.After(1 * time.Second):
		}
		return fmt.Errorf("event_runtime: dispatcher stop timed out after %v; in-flight events remain unacked", timeout)
	}
}

// Subscribe registers a pattern+handler pair. The handler is called with
// any envelope whose EventType matches the glob `pattern` (path.Match
// semantics). Duplicate patterns are rejected.
//
// This is a convenience API for ad-hoc handlers (tests, scripts); production
// subscribers should implement the Subscriber interface and use
// RegisterDispatcher.
func (d *Dispatcher) Subscribe(pattern string, handler HandlerFunc) error {
	if d == nil {
		return fmt.Errorf("event_runtime: dispatcher is nil")
	}
	if pattern == "" {
		return fmt.Errorf("%w: pattern is empty", ErrInvalidEnvelope)
	}
	if handler == nil {
		return ErrNilSubscriber
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.pattern[pattern]; exists {
		return fmt.Errorf("%w: pattern=%q", ErrDuplicateSubscriber, pattern)
	}
	d.pattern[pattern] = &Subscription{Pattern: pattern, Handler: handler}
	return nil
}

// spawnWorker starts a single delivery goroutine for the given subscriber.
// Holds no lock when calling go.
func (d *Dispatcher) spawnWorker(parentCtx context.Context, sub Subscriber) {
	ctx, cancel := context.WithCancel(parentCtx)
	state := &subState{
		sub:         sub,
		patterns:    sub.EventTypes(),
		cancel:      cancel,
		done:        make(chan struct{}),
		retryCounts: make(map[int64]int),
	}
	d.mu.Lock()
	d.subStates[sub.Name()] = state
	d.mu.Unlock()

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer close(state.done)
		d.workerLoop(ctx, state)
	}()
}

// workerLoop is the per-subscriber delivery loop. Polls event_log on a
// fixed interval, applies MaxAckPending backpressure, dispatches matched
// envelopes, and manages ACK/NACK state.
func (d *Dispatcher) workerLoop(ctx context.Context, state *subState) {
	log.Printf("event_runtime: worker started name=%s patterns=%v max_ack=%d",
		state.sub.Name(), state.patterns, state.sub.MaxAckPending())

	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("event_runtime: worker stopping name=%s reason=%v", state.sub.Name(), ctx.Err())
			return
		case <-d.stopCh:
			log.Printf("event_runtime: worker stop signalled name=%s", state.sub.Name())
			return
		case <-ticker.C:
			d.deliverBatch(ctx, state)
		}
	}
}

// deliverBatch reads up to MaxAckPending unacked envelopes for the
// subscriber, matches against patterns, and dispatches each in a fresh
// goroutine so backpressure is per-subscriber not global.
func (d *Dispatcher) deliverBatch(ctx context.Context, state *subState) {
	if atomic.LoadInt32(&state.inflight) >= int32(state.sub.MaxAckPending()) {
		// Backpressure: skip this poll cycle.
		return
	}

	budget := state.sub.MaxAckPending() - int(atomic.LoadInt32(&state.inflight))
	if budget <= 0 {
		return
	}

	envelopes, err := d.log.ReadUnacked(ctx, state.sub.Name(), budget)
	if err != nil {
		log.Printf("event_runtime: read unacked falhou name=%s: %v", state.sub.Name(), err)
		return
	}
	for _, env := range envelopes {
		if !eventMatches(state.patterns, env.EventType) {
			// Permanent ACK: the subscriber does not want this event_type.
			// We mark it acked so it doesn't pollute the queue.
			if err := d.ackPermanent(ctx, state.sub.Name(), env.Sequence, "pattern_mismatch"); err != nil {
				log.Printf("event_runtime: ack pattern_mismatch falhou seq=%d: %v", env.Sequence, err)
			}
			continue
		}

		// unsupported_schema → permanent ACK with reason (no retry).
		if env.SchemaVersion > CurrentSchemaVersion {
			log.Printf("event_runtime.unsupported_schema_acked event_id=%s schema_version=%d current=%d",
				env.EventID, env.SchemaVersion, CurrentSchemaVersion)
			if err := d.ackPermanent(ctx, state.sub.Name(), env.Sequence, "unsupported_schema"); err != nil {
				log.Printf("event_runtime: ack unsupported_schema falhou seq=%d: %v", env.Sequence, err)
			}
			continue
		}

		atomic.AddInt32(&state.inflight, 1)
		env := env // capture
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			d.dispatchOne(ctx, state, &env)
		}()
	}
}

// dispatchOne calls Subscriber.Handle with panic recovery, ACK/NACK
// semantics, and exponential backoff on retry.
func (d *Dispatcher) dispatchOne(parentCtx context.Context, state *subState, env *Envelope) {
	defer atomic.AddInt32(&state.inflight, -1)

	// Per-delivery timeout: stop waiting if handler hangs.
	ctx, cancel := context.WithTimeout(parentCtx, 60*time.Second)
	defer cancel()

	var handleErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("event_runtime: handler panic recovered name=%s event_id=%s correlation_id=%s panic=%v",
					state.sub.Name(), env.EventID, env.CorrelationID, r)
				handleErr = fmt.Errorf("event_runtime: handler panic: %v", r)
			}
		}()
		handleErr = state.sub.Handle(ctx, env)
	}()

	if handleErr == nil {
		if err := d.ackSuccess(ctx, state.sub.Name(), env.Sequence); err != nil {
			log.Printf("event_runtime: ack falhou seq=%d: %v", env.Sequence, err)
		}
		// Clear retry count for this seq.
		state.retryCountsM.Lock()
		delete(state.retryCounts, env.Sequence)
		state.retryCountsM.Unlock()
		return
	}

	// Transient error: compute backoff and either retry in-place or
	// leave the event unacked for next poll cycle.
	state.retryCountsM.Lock()
	attempts := state.retryCounts[env.Sequence] + 1
	state.retryCounts[env.Sequence] = attempts
	state.retryCountsM.Unlock()

	delay := backoffDelay(attempts)
	log.Printf("event_runtime: handler error name=%s seq=%d attempts=%d delay=%v err=%v",
		state.sub.Name(), env.Sequence, attempts, delay, handleErr)

	// Sleep outside the lock so other subscribers can proceed.
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return
	case <-timer.C:
	case <-d.stopCh:
		return
	}

	// If we're still under MaxAckPending and not stopped, retry inline.
	if atomic.LoadInt32(&state.inflight) <= int32(state.sub.MaxAckPending()) && ctx.Err() == nil {
		d.dispatchOne(parentCtx, state, env)
	}
}

// ackSuccess marks the event delivered to this subscriber by advancing the
// per-subscriber cursor. We do NOT set acked_at globally because other
// subscribers may still need to receive the same envelope (fan-out).
func (d *Dispatcher) ackSuccess(ctx context.Context, subName string, seq int64) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("event_runtime: begin tx: %w", err)
	}
	defer tx.Rollback()
	if err := d.log.AdvanceCursor(ctx, tx, subName, seq); err != nil {
		return err
	}
	return tx.Commit()
}

// ackPermanent marks the event acked with a reason (no cursor advance — the
// subscriber didn't really "process" this; we just want it out of the queue).
func (d *Dispatcher) ackPermanent(ctx context.Context, subName string, seq int64, reason string) error {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("event_runtime: begin tx: %w", err)
	}
	defer tx.Rollback()
	if err := d.log.MarkAcked(ctx, tx, seq, reason); err != nil {
		return err
	}
	// Advance cursor so we don't see this event again.
	if err := d.log.AdvanceCursor(ctx, tx, subName, seq); err != nil {
		return err
	}
	return tx.Commit()
}

// backoffDelay computes the wait for the given attempt number:
// base * factor^(attempt-1), capped at backoffCap, with ±jitterPercent
// jitter.
func backoffDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := float64(backoffBase)
	for i := 1; i < attempt; i++ {
		d *= backoffFactor
		if time.Duration(d) >= backoffCap {
			d = float64(backoffCap)
			break
		}
	}
	// Jitter ±20%.
	jitter := (rand.Float64()*2 - 1) * jitterPercent
	d = d * (1.0 + jitter)
	if d < float64(backoffBase) {
		d = float64(backoffBase)
	}
	return time.Duration(d)
}

// eventMatches returns true if eventType matches any of the patterns
// (path.Match glob semantics). A pattern of "*" matches everything.
func eventMatches(patterns []string, eventType string) bool {
	for _, p := range patterns {
		if p == "*" {
			return true
		}
		matched, err := path.Match(p, eventType)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// eventLogTableExists checks for the event_log table via sqlite_master.
// Used by Start to fail-closed when the schema is missing.
func eventLogTableExists(ctx context.Context, db *sql.DB) bool {
	var name string
	err := db.QueryRowContext(ctx,
		`SELECT name FROM sqlite_master WHERE type='table' AND name='event_log'`).Scan(&name)
	if err != nil {
		return false
	}
	return name == "event_log"
}
