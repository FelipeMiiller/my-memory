package event_runtime

import (
	"context"
	"errors"
	"testing"
)

func TestOutbox_EmitRequiresTx(t *testing.T) {
	conn, cleanup := newTestDB(t)
	defer cleanup()

	outbox := NewOutbox(NewLog(conn))
	ctx := context.Background()
	env := NewEnvelope("memory.committed", "documents/foo.md")

	err := outbox.Emit(ctx, nil, env)
	if err == nil {
		t.Fatal("Emit com tx=nil deveria falhar")
	}
	if !errors.Is(err, ErrTxRequired) {
		t.Errorf("Emit deveria retornar ErrTxRequired; obteve %v", err)
	}
}

func TestOutbox_EmitBatchRequiresTx(t *testing.T) {
	conn, cleanup := newTestDB(t)
	defer cleanup()

	outbox := NewOutbox(NewLog(conn))
	ctx := context.Background()
	envs := []*Envelope{
		NewEnvelope("a.x", "agg-1"),
		NewEnvelope("b.x", "agg-2"),
	}

	err := outbox.EmitBatch(ctx, nil, envs)
	if err == nil {
		t.Fatal("EmitBatch com tx=nil deveria falhar")
	}
	if !errors.Is(err, ErrTxRequired) {
		t.Errorf("EmitBatch deveria retornar ErrTxRequired; obteve %v", err)
	}
}

// EmitBatch wraps the first failure with event_type context so the caller
// can pinpoint which envelope blew up.
func TestOutbox_EmitBatchShortCircuitsOnError(t *testing.T) {
	conn, cleanup := newTestDB(t)
	defer cleanup()

	log := NewLog(conn)
	outbox := NewOutbox(log)
	ctx := context.Background()

	// Build a batch where the SECOND envelope has an empty event_type,
	// which Validate will reject.
	envs := []*Envelope{
		NewEnvelope("memory.committed", "documents/foo.md"),
		NewEnvelope("", "documents/bar.md"), // invalid
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx falhou: %v", err)
	}
	defer tx.Rollback()

	err = outbox.EmitBatch(ctx, tx, envs)
	if err == nil {
		t.Fatal("EmitBatch deveria falhar no segundo envelope")
	}
	if !errors.Is(err, ErrInvalidEnvelope) {
		t.Errorf("erro deveria envolver ErrInvalidEnvelope; obteve %v", err)
	}
	// The error message should mention the offending event_type (which is
	// empty, but the wrapper still embeds it).
	if !contains(err.Error(), "event_type=") {
		t.Errorf("erro deveria mencionar event_type no contexto; obteve %q", err.Error())
	}

	// First envelope must NOT have been committed (tx still open, will rollback).
	// Verify by rolling back and checking the table is empty.
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback falhou: %v", err)
	}
	seq, err := log.LastSequence(ctx)
	if err != nil {
		t.Fatalf("LastSequence falhou: %v", err)
	}
	if seq != 0 {
		t.Errorf("LastSequence = %d; esperado 0 após rollback", seq)
	}
}

// Emit succeeds when given a valid envelope and a live tx.
func TestOutbox_Emit_HappyPath(t *testing.T) {
	conn, cleanup := newTestDB(t)
	defer cleanup()

	outbox := NewOutbox(NewLog(conn))
	ctx := context.Background()
	env := NewEnvelope("memory.committed", "documents/foo.md")

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx falhou: %v", err)
	}
	if err := outbox.Emit(ctx, tx, env); err != nil {
		tx.Rollback()
		t.Fatalf("Emit falhou: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit falhou: %v", err)
	}

	log := NewLog(conn)
	seq, err := log.LastSequence(ctx)
	if err != nil {
		t.Fatalf("LastSequence falhou: %v", err)
	}
	if seq != 1 {
		t.Errorf("LastSequence = %d; esperado 1", seq)
	}
	if env.Sequence != 1 {
		t.Errorf("env.Sequence = %d; esperado 1 (Append should populate)", env.Sequence)
	}
}

// TestOutbox_AtomicWithEffect proves that rolling back the caller's
// transaction hides BOTH the side effect (INSERT into a fictitious
// `effects` table) AND the outbox event_log insert. This is the core
// guarantee that makes the outbox pattern safe for crash recovery.
func TestOutbox_AtomicWithEffect(t *testing.T) {
	conn, cleanup := newTestDB(t)
	defer cleanup()

	// Fictitious table simulating a side-effect (e.g. document write,
	// graph edge insertion). UNIQUE on event_id simulates idempotency.
	if _, err := conn.Exec(`
		CREATE TABLE effects (
			event_id TEXT PRIMARY KEY,
			payload  TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)
	`); err != nil {
		t.Fatalf("CREATE effects falhou: %v", err)
	}

	outbox := NewOutbox(NewLog(conn))
	ctx := context.Background()
	env := NewEnvelope("memory.committed", "documents/foo.md")

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx falhou: %v", err)
	}

	// 1. Side effect in the SAME tx.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO effects (event_id, payload, applied_at) VALUES (?, ?, ?)`,
		env.EventID, "side-effect", "now"); err != nil {
		tx.Rollback()
		t.Fatalf("INSERT effects falhou: %v", err)
	}

	// 2. Outbox emit in the SAME tx.
	if err := outbox.Emit(ctx, tx, env); err != nil {
		tx.Rollback()
		t.Fatalf("Emit falhou: %v", err)
	}

	// 3. ROLLBACK — both writes must be invisible afterwards.
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback falhou: %v", err)
	}

	// event_log must be empty.
	var eventCount int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_log`).Scan(&eventCount); err != nil {
		t.Fatalf("COUNT event_log falhou: %v", err)
	}
	if eventCount != 0 {
		t.Errorf("event_log COUNT = %d; esperado 0 após rollback", eventCount)
	}

	// effects must be empty.
	var effectCount int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM effects`).Scan(&effectCount); err != nil {
		t.Fatalf("COUNT effects falhou: %v", err)
	}
	if effectCount != 0 {
		t.Errorf("effects COUNT = %d; esperado 0 após rollback", effectCount)
	}

	// And LastSequence should report 0.
	seq, err := NewLog(conn).LastSequence(ctx)
	if err != nil {
		t.Fatalf("LastSequence falhou: %v", err)
	}
	if seq != 0 {
		t.Errorf("LastSequence = %d; esperado 0 após rollback", seq)
	}
}

// TestOutbox_RetryDoesNotDuplicateEventID simulates the retry-after-SQLITE_BUSY
// scenario: caller commits tx1 successfully, then crashes and retries the
// same operation in tx2. The second Emit must NOT create a duplicate
// event_log row (UNIQUE on event_id), and the effects table must NOT be
// re-applied (simulated by catching the duplicate error and treating it as
// idempotent success).
func TestOutbox_RetryDoesNotDuplicateEventID(t *testing.T) {
	conn, cleanup := newTestDB(t)
	defer cleanup()

	if _, err := conn.Exec(`
		CREATE TABLE effects (
			event_id TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		)
	`); err != nil {
		t.Fatalf("CREATE effects falhou: %v", err)
	}

	outbox := NewOutbox(NewLog(conn))
	log := NewLog(conn)
	ctx := context.Background()

	// Stable event_id so tx1 and tx2 emit the same envelope.
	env := NewEnvelope("memory.committed", "documents/foo.md")
	env.EventID = "stable-event-id-retry-test"

	// tx1: side effect + emit + commit.
	tx1, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx 1 falhou: %v", err)
	}
	if _, err := tx1.ExecContext(ctx,
		`INSERT INTO effects (event_id, applied_at) VALUES (?, ?)`,
		env.EventID, "tx1"); err != nil {
		tx1.Rollback()
		t.Fatalf("INSERT effects tx1 falhou: %v", err)
	}
	if err := outbox.Emit(ctx, tx1, env); err != nil {
		tx1.Rollback()
		t.Fatalf("Emit tx1 falhou: %v", err)
	}
	if err := tx1.Commit(); err != nil {
		t.Fatalf("Commit tx1 falhou: %v", err)
	}

	// tx2: retry — same event_id, same side effect. The retry should be
	// recognized as a duplicate and NOT create new rows.
	tx2, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx 2 falhou: %v", err)
	}

	// Side-effect attempt: should fail because event_id is UNIQUE in effects.
	_, sideEffectErr := tx2.ExecContext(ctx,
		`INSERT INTO effects (event_id, applied_at) VALUES (?, ?)`,
		env.EventID, "tx2-retry")
	if sideEffectErr == nil {
		tx2.Rollback()
		t.Fatal("INSERT effects em retry deveria falhar por UNIQUE constraint")
	}

	// Outbox emit: should return ErrDuplicateEventID. Caller treats as success.
	emitErr := outbox.Emit(ctx, tx2, env)
	if emitErr == nil {
		tx2.Rollback()
		t.Fatal("Emit em retry deveria falhar com ErrDuplicateEventID")
	}
	if !errors.Is(emitErr, ErrDuplicateEventID) {
		t.Errorf("Emit em retry deveria retornar ErrDuplicateEventID; obteve %v", emitErr)
	}

	// Rollback tx2 (nothing was persisted in tx2).
	if err := tx2.Rollback(); err != nil {
		t.Fatalf("Rollback tx2 falhou: %v", err)
	}

	// Final invariants:
	// 1. event_log has exactly ONE row for this event_id.
	var eventCount int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM event_log WHERE event_id = ?`, env.EventID,
	).Scan(&eventCount); err != nil {
		t.Fatalf("COUNT event_log falhou: %v", err)
	}
	if eventCount != 1 {
		t.Errorf("event_log COUNT para event_id = %d; esperado 1 (sem duplicatas)", eventCount)
	}

	// 2. effects has exactly ONE row.
	var effectCount int
	if err := conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM effects WHERE event_id = ?`, env.EventID,
	).Scan(&effectCount); err != nil {
		t.Fatalf("COUNT effects falhou: %v", err)
	}
	if effectCount != 1 {
		t.Errorf("effects COUNT para event_id = %d; esperado 1 (idempotente)", effectCount)
	}

	// 3. LastSequence is 1.
	seq, err := log.LastSequence(ctx)
	if err != nil {
		t.Fatalf("LastSequence falhou: %v", err)
	}
	if seq != 1 {
		t.Errorf("LastSequence = %d; esperado 1", seq)
	}
}
