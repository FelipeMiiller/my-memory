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
