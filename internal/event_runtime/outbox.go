package event_runtime

import (
	"context"
	"database/sql"
	"fmt"
)

// Outbox is the transactional emit helper described in ADR-043. It is a
// stateless wrapper over Log.Append that adds:
//
//   - The ErrTxRequired guard (so callers fail fast before touching the DB).
//   - Batched emission with per-event error context (event_type in the wrap).
//
// IMPORTANT: Outbox NEVER calls BEGIN, COMMIT or ROLLBACK. The caller owns
// the transaction lifecycle. This is what gives the outbox pattern its
// atomicity guarantee — the event_log insert and the side effect either both
// commit or both roll back together.
type Outbox struct {
	log *Log
}

// NewOutbox binds an Outbox to a Log. The Log's *sql.DB must already be
// initialized with the event_runtime schema.
func NewOutbox(log *Log) *Outbox {
	return &Outbox{log: log}
}

// Emit appends a single envelope inside the caller's transaction. Returns
// ErrTxRequired when tx == nil; otherwise delegates to Log.Append which
// validates the envelope and inserts into event_log.
//
// Idempotent retry safety: if the same event_id was already persisted (for
// example, after the caller retries a SQLITE_BUSY transaction), Append
// returns ErrDuplicateEventID. The caller should treat this as success.
func (o *Outbox) Emit(ctx context.Context, tx *sql.Tx, env *Envelope) error {
	if tx == nil {
		return ErrTxRequired
	}
	if o == nil || o.log == nil {
		return fmt.Errorf("event_runtime: outbox is not initialized")
	}
	return o.log.Append(ctx, tx, env)
}

// EmitBatch appends multiple envelopes in order, using the caller's
// transaction. On the first error it short-circuits and returns the wrapped
// error annotated with the failing event's type so the caller can log it
// meaningfully. Already-persisted envelopes from a partially-applied batch
// are rolled back when the caller rolls back its transaction.
func (o *Outbox) EmitBatch(ctx context.Context, tx *sql.Tx, envs []*Envelope) error {
	if tx == nil {
		return ErrTxRequired
	}
	if o == nil || o.log == nil {
		return fmt.Errorf("event_runtime: outbox is not initialized")
	}
	for _, env := range envs {
		if err := o.Emit(ctx, tx, env); err != nil {
			etype := ""
			if env != nil {
				etype = env.EventType
			}
			return fmt.Errorf("event_runtime: outbox batch failed on event_type=%q: %w", etype, err)
		}
	}
	return nil
}
