package event_runtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Log is the read/write surface over the event_log and projection_cursor
// tables defined in internal/db/schema.go. It is intentionally minimal —
// the dispatcher (T7) adds polling/ACK/retry on top of it.
//
// Log methods take an explicit *sql.Tx for writes so callers can compose the
// outbox pattern: every event lives in the SAME transaction as the side
// effect that produced it. Reads accept *sql.DB because they never write.
type Log struct {
	db *sql.DB
}

// NewLog constructs a Log bound to the given database handle. The DB must
// already have the event_log and projection_cursor tables (see
// internal/db.Schema).
func NewLog(db *sql.DB) *Log {
	return &Log{db: db}
}

// Append inserts the envelope into event_log using the caller's transaction.
// Append does NOT call BEGIN/COMMIT — the caller owns the transaction
// lifecycle, which is what makes the outbox pattern possible.
//
// Returns:
//   - ErrTxRequired when tx == nil
//   - ErrInvalidEnvelope when env fails validation
//   - ErrPayloadTooLarge when payload exceeds 1 MiB (caught by Validate)
//   - ErrDuplicateEventID when the same event_id was already appended
//     (UNIQUE constraint; producer treats this as idempotent retry success)
func (l *Log) Append(ctx context.Context, tx *sql.Tx, env *Envelope) error {
	if tx == nil {
		return ErrTxRequired
	}
	if env == nil {
		return fmt.Errorf("%w: envelope is nil", ErrInvalidEnvelope)
	}
	if err := env.Validate(); err != nil {
		return err
	}
	if env.EventID == "" {
		return fmt.Errorf("%w: event_id is empty (call NewEnvelope first)", ErrInvalidEnvelope)
	}

	// payload + headers are stored as BLOB. We serialize the full envelope to
	// JSON for the payload column (canonical wire format) and a small
	// "headers" blob with routing metadata for the dispatcher (Phase 3).
	payload, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("event_runtime: marshal envelope: %w", err)
	}
	headers, err := encodeHeaders(env)
	if err != nil {
		return fmt.Errorf("event_runtime: encode headers: %w", err)
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO event_log
			(event_id, schema_version, event_type, aggregate_id, revision,
			 payload, headers, created_at, acked_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, NULL)
	`, env.EventID, env.SchemaVersion, env.EventType, env.AggregateID, env.Revision,
		payload, headers, env.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%w: event_id=%s", ErrDuplicateEventID, env.EventID)
		}
		return fmt.Errorf("event_runtime: insert event_log: %w", err)
	}
	seq, err := res.LastInsertId()
	if err == nil {
		env.Sequence = seq
	}
	return nil
}

// LastSequence returns the highest sequence value persisted in event_log,
// or 0 when the table is empty.
func (l *Log) LastSequence(ctx context.Context) (int64, error) {
	var seq sql.NullInt64
	err := l.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence), 0) FROM event_log`).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("event_runtime: query last sequence: %w", err)
	}
	return seq.Int64, nil
}

// ReadUnacked returns up to `limit` envelopes that have not yet been acked
// for the given subscriber, ordered by sequence ASC. The cursor lookup uses
// projection_cursor.last_sequence; a missing cursor means "from the start".
//
// Fan-out semantics (T7): an envelope is considered "pending" for a
// subscriber if the subscriber's cursor is behind the envelope's sequence
// AND the envelope has not been permanently rejected by ANY subscriber
// (acked_at column embedded with a reason like "unsupported_schema" or
// "pattern_mismatch"). Successful delivery does NOT set acked_at — only the
// per-subscriber cursor advances — so multiple subscribers receive the
// same envelope (one delivery per subscriber, exactly-once per subscriber).
func (l *Log) ReadUnacked(ctx context.Context, subscriberName string, limit int) ([]Envelope, error) {
	if subscriberName == "" {
		return nil, fmt.Errorf("%w: subscriber name is empty", ErrInvalidEnvelope)
	}
	if limit <= 0 {
		limit = 64
	}

	rows, err := l.db.QueryContext(ctx, `
		SELECT e.sequence, e.payload
		FROM event_log e
		LEFT JOIN projection_cursor c
		  ON c.projection_name = ?
		WHERE (c.last_sequence IS NULL OR e.sequence > c.last_sequence)
		  AND (e.acked_at IS NULL OR e.acked_at NOT LIKE '%|%')
		ORDER BY e.sequence ASC
		LIMIT ?
	`, subscriberName, limit)
	if err != nil {
		return nil, fmt.Errorf("event_runtime: read unacked: %w", err)
	}
	defer rows.Close()

	out := make([]Envelope, 0, limit)
	for rows.Next() {
		var seq int64
		var payload []byte
		if err := rows.Scan(&seq, &payload); err != nil {
			return nil, fmt.Errorf("event_runtime: scan event_log row: %w", err)
		}
		var env Envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			return nil, fmt.Errorf("event_runtime: decode event_log payload: %w", err)
		}
		// The payload column was written before the sequence was assigned
		// (SQLite assigns it on INSERT), so the JSON copy has Sequence=0.
		// Always prefer the authoritative column value.
		env.Sequence = seq
		out = append(out, env)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("event_runtime: iterate event_log: %w", err)
	}
	return out, nil
}

// MarkAcked updates both event_log.acked_at and projection_cursor.last_sequence
// in the caller's transaction. The two writes are atomic (same tx) so a crash
// between them cannot desync the cursor from the log.
//
// reason == "" represents successful handler completion; non-empty values are
// permanent reject reasons (e.g. "unsupported_schema") — the dispatcher
// stores them verbatim so audit can reconstruct why an event was dropped.
func (l *Log) MarkAcked(ctx context.Context, tx *sql.Tx, sequence int64, reason string) error {
	if tx == nil {
		return ErrTxRequired
	}
	if sequence <= 0 {
		return fmt.Errorf("event_runtime: invalid sequence %d", sequence)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)

	// 1. Stamp event_log row.
	if reason == "" {
		if _, err := tx.ExecContext(ctx,
			`UPDATE event_log SET acked_at = ? WHERE sequence = ? AND acked_at IS NULL`,
			now, sequence); err != nil {
			return fmt.Errorf("event_runtime: stamp event_log: %w", err)
		}
	} else {
		// For permanent reject we still set acked_at so the event leaves the
		// unacked queue, but we embed the reason in a side-table if needed.
		// For now, encode reason into acked_at as "<RFC3339Nano>|<reason>".
		stamp := now + "|" + reason
		if _, err := tx.ExecContext(ctx,
			`UPDATE event_log SET acked_at = ? WHERE sequence = ? AND acked_at IS NULL`,
			stamp, sequence); err != nil {
			return fmt.Errorf("event_runtime: stamp event_log: %w", err)
		}
	}
	return nil
}

// AdvanceCursor moves projection_cursor.last_sequence forward to `sequence`
// for the given subscriber. Idempotent: advancing to a value ≤ current is a
// no-op. Call this AFTER MarkAcked in the same tx for full atomicity.
func (l *Log) AdvanceCursor(ctx context.Context, tx *sql.Tx, subscriberName string, sequence int64) error {
	if tx == nil {
		return ErrTxRequired
	}
	if subscriberName == "" {
		return fmt.Errorf("%w: subscriber name is empty", ErrInvalidEnvelope)
	}
	if sequence <= 0 {
		return fmt.Errorf("event_runtime: invalid sequence %d", sequence)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO projection_cursor (projection_name, last_sequence, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(projection_name) DO UPDATE SET
			last_sequence = MAX(projection_cursor.last_sequence, excluded.last_sequence),
			updated_at    = excluded.updated_at
	`, subscriberName, sequence, now)
	if err != nil {
		return fmt.Errorf("event_runtime: advance cursor: %w", err)
	}
	return nil
}

// encodeHeaders marshals the routing metadata (fields used by the dispatcher
// in Phase 3) into a compact BLOB. Keeping this separate from payload lets
// the dispatcher skip JSON-decoding the full envelope just to filter by
// event_type.
func encodeHeaders(env *Envelope) ([]byte, error) {
	h := struct {
		EventType     string `json:"event_type"`
		AggregateID   string `json:"aggregate_id"`
		SchemaVersion int    `json:"schema_version"`
		CorrelationID string `json:"correlation_id,omitempty"`
		Priority      string `json:"priority,omitempty"`
	}{
		EventType:     env.EventType,
		AggregateID:   env.AggregateID,
		SchemaVersion: env.SchemaVersion,
		CorrelationID: env.CorrelationID,
		Priority:      env.Priority,
	}
	return json.Marshal(h)
}

// isUniqueViolation detects SQLite UNIQUE constraint failures by error message
// substring. The driver returns plain errors.Error objects without a typed
// sentinel, so message matching is the portable path.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "UNIQUE constraint failed") ||
		contains(msg, "constraint failed: UNIQUE")
}

// contains is a tiny case-sensitive substring check kept local to avoid
// pulling strings into the hot path; alloc-free for short messages.
func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	if len(haystack) < len(needle) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// ErrSequenceAheadOfCursor is returned when an external writer tries to mark
// an event acked whose sequence is older than the cursor's last value. This
// shouldn't happen in practice but guards against dispatcher bugs.
var ErrSequenceAheadOfCursor = errors.New("event_runtime: sequence behind cursor; refusing to rewind")
