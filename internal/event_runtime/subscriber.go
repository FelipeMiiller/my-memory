// Package event_runtime — subscriber.go
//
// Subscriber interface and the Dispatcher stub that T7 extends with the
// fan-out / ACK / retry / panic-recovery engine.
//
// The interface is the contract every projection future-proofs against —
// the audit subscriber (T9), the SQLite FTS projection (ADR-044), the
// graph projection, and the vector projection all conform to it. Keeping
// the contract narrow (4 methods, all obvious) makes adding a new
// subscriber a 30-line job.
//
// References:
//   - ADR-043 §5 (in-process dispatcher, fan-out per event_type)
//   - ADR-050 §LLM02 (audit subscriber MUST redact PII — see redaction.go)
package event_runtime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrNilSubscriber is returned by RegisterDispatcher when the caller passes
// a nil Subscriber.
var ErrNilSubscriber = errors.New("event_runtime: subscriber is nil")

// HandlerFunc is the type-erased handler signature accepted by
// Dispatcher.Subscribe (T7). It deliberately mirrors Subscriber.Handle so
// adapters that wrap a HandlerFunc into a Subscriber are trivial.
type HandlerFunc func(ctx context.Context, env *Envelope) error

// Subscription is a pattern+handler pair registered via Dispatcher.Subscribe.
// T7 fills the dispatch loop that drives these.
type Subscription struct {
	Pattern string
	Handler HandlerFunc
}

// Subscriber is the contract every projection implements to receive events
// from the in-process dispatcher (ADR-043 §5).
//
// Implementations MUST be safe to call from multiple goroutines because the
// dispatcher runs one delivery goroutine per registered subscriber; the
// implementation owns its own concurrency primitives (e.g. *AuditSubscriber
// guards writes with a sync.Mutex).
type Subscriber interface {
	// Name returns the unique subscriber identifier. Used as the
	// projection_cursor.projection_name key, so renaming a subscriber in
	// production effectively resets its cursor (events replayed from the
	// beginning).
	Name() string

	// EventTypes returns the list of event_type patterns this subscriber
	// wants to receive. Each entry is matched against env.EventType via
	// path.Match semantics; "*" subscribes to every event.
	//
	// Returning an empty slice means "no subscription" — the dispatcher
	// will skip this subscriber when polling.
	EventTypes() []string

	// Handle processes a single event. Returning nil triggers ACK;
	// returning a non-nil error triggers NACK + exponential backoff retry
	// (per ADR-043 §6). The dispatcher treats a panic inside Handle as
	// a transient error.
	Handle(ctx context.Context, env *Envelope) error

	// MaxAckPending is the maximum number of unacknowledged events the
	// subscriber is willing to have in flight. The dispatcher uses this
	// for backpressure (ADR-043 §6 / ADR-050 LLM10 unbounded consumption):
	// when in-flight count reaches MaxAckPending, the delivery goroutine
	// pauses reads from event_log until at least one handler returns.
	MaxAckPending() int
}

// Dispatcher is the fan-out engine that reads from the event_log and
// delivers envelopes to every registered Subscriber whose EventTypes()
// match the envelope's event_type. T6 defined the registry; T7 added the
// engine fields below.
//
// Field semantics:
//   - subs:        named subscribers registered via RegisterDispatcher.
//   - pattern:     pattern-keyed subscriptions registered via Subscribe.
//   - subStates:   per-subscriber delivery state (cancel func, done chan,
//     in-flight counter, retry counts).
//   - stopCh:      closed by Stop() to signal worker goroutines to drain.
//   - started:     true after Start has spawned workers.
//   - stopped:     true after Stop has run (Start is rejected after this).
//   - db, log:     wired by Attach before Start; nil until then.
//   - pollInterval: how often idle workers re-check the event_log.
//   - gracePeriod:  how long Stop waits for in-flight handlers.
//   - mu:           guards every field above; use RLock for read paths.
//
// The zero value is NOT ready — callers must use NewDispatcher + Attach + Start.
type Dispatcher struct {
	mu           sync.RWMutex
	subs         map[string]Subscriber
	pattern      map[string]*Subscription
	subStates    map[string]*subState
	stopCh       chan struct{}
	started      bool
	stopped      bool
	db           *sql.DB
	log          *Log
	pollInterval time.Duration
	gracePeriod  time.Duration
	wg           sync.WaitGroup
}

// NewDispatcher constructs an empty dispatcher. T7 wires the DB/Log
// handles and the poll interval when Start is called.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		subs:      make(map[string]Subscriber),
		pattern:   make(map[string]*Subscription),
		subStates: make(map[string]*subState),
		stopCh:    make(chan struct{}),
	}
}

// RegisterDispatcher validates and registers a Subscriber on the dispatcher.
// Returns:
//   - ErrNilSubscriber when d == nil or sub == nil
//   - ErrDuplicateSubscriber when a subscriber with the same Name() was
//     already registered
//   - ErrInvalidEnvelope when Name() returns "" (subscribers MUST be named
//     so the projection_cursor can persist their progress)
//
// RegisterDispatcher does NOT spawn goroutines — that's Dispatcher.Start.
// It is safe to call before or after Start (before Start is the canonical
// pattern; after Start it will pick up the new subscriber on the next
// poll cycle).
func RegisterDispatcher(d *Dispatcher, sub Subscriber) error {
	if d == nil {
		return fmt.Errorf("event_runtime: dispatcher is nil")
	}
	if sub == nil {
		return ErrNilSubscriber
	}
	name := sub.Name()
	if name == "" {
		return fmt.Errorf("%w: subscriber name is empty", ErrInvalidEnvelope)
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.subs[name]; exists {
		return fmt.Errorf("%w: name=%q", ErrDuplicateSubscriber, name)
	}
	d.subs[name] = sub
	return nil
}

// Subscribers returns a snapshot of the registered subscriber names. Useful
// for `mem doctor --events` to count subscribers_active without exposing
// the underlying map. Order is not guaranteed.
func (d *Dispatcher) Subscribers() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]string, 0, len(d.subs))
	for name := range d.subs {
		out = append(out, name)
	}
	return out
}

// HasSubscriber reports whether a subscriber with the given name is
// registered. Used by T7 and tests; cheap O(1) lookup.
func (d *Dispatcher) HasSubscriber(name string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.subs[name]
	return ok
}
