package event_runtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	// Pure-Go SQLite driver — works without cgo in test binaries.
	// Production code uses mattn/go-sqlite3 (cgo) when sqlite-vec is
	// available, or modernc.org/sqlite as fallback; both share the same SQL.
	_ "modernc.org/sqlite"
)

// eventRuntimeSchema is the minimal DDL the event_runtime tests need.
// It mirrors the event_log + projection_cursor definitions from
// internal/db/schema.go (both Schema and FallbackSchema), keeping tests
// independent of sqlite-vec which isn't available in modernc.org/sqlite.
const eventRuntimeSchema = `
CREATE TABLE IF NOT EXISTS event_log (
    sequence INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT UNIQUE NOT NULL,
    schema_version INTEGER NOT NULL DEFAULT 1,
    event_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    revision INTEGER,
    payload BLOB NOT NULL,
    headers BLOB NOT NULL,
    created_at TEXT NOT NULL,
    acked_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_event_log_aggregate ON event_log(aggregate_id);
CREATE INDEX IF NOT EXISTS idx_event_log_unacked ON event_log(sequence) WHERE acked_at IS NULL;

CREATE TABLE IF NOT EXISTS projection_cursor (
    projection_name TEXT PRIMARY KEY,
    last_sequence INTEGER NOT NULL,
    updated_at TEXT NOT NULL
);
`

// newTestDB opens an in-memory SQLite with the event_runtime schema applied.
// Returns (db, closeFn). Tests should defer closeFn() to release the connection.
func newTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open falhou: %v", err)
	}
	// Belt-and-suspenders: only one in-memory connection so concurrent tests
	// don't see each other's writes.
	conn.SetMaxOpenConns(1)
	if _, err := conn.Exec(eventRuntimeSchema); err != nil {
		conn.Close()
		t.Fatalf("aplicar Schema falhou: %v", err)
	}
	return conn, func() { conn.Close() }
}

func TestLog_AppendInTransaction_AndReorder(t *testing.T) {
	conn, cleanup := newTestDB(t)
	defer cleanup()

	log := NewLog(conn)
	ctx := context.Background()

	// Append 3 envelopes in 3 separate transactions and verify they got
	// monotonic sequences, then read them back ordered DESC.
	envs := []*Envelope{
		NewEnvelope("memory.committed", "documents/foo.md"),
		NewEnvelope("tool.finished", "documents/foo.md"),
		NewEnvelope("session.ended", "session-abc"),
	}

	for _, env := range envs {
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx falhou: %v", err)
		}
		if err := log.Append(ctx, tx, env); err != nil {
			tx.Rollback()
			t.Fatalf("Append falhou: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit falhou: %v", err)
		}
	}

	// Verify monotonic sequence.
	last, err := log.LastSequence(ctx)
	if err != nil {
		t.Fatalf("LastSequence falhou: %v", err)
	}
	if last != 3 {
		t.Errorf("LastSequence = %d; esperado 3", last)
	}

	// Read all back unacked (no cursor yet → from start).
	got, err := log.ReadUnacked(ctx, "test-subscriber", 10)
	if err != nil {
		t.Fatalf("ReadUnacked falhou: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("ReadUnacked retornou %d envelopes; esperado 3", len(got))
	}
	// ReadUnacked returns ASC.
	for i := 1; i < len(got); i++ {
		if got[i].Sequence <= got[i-1].Sequence {
			t.Errorf("ordem violada em %d: %d <= %d", i, got[i].Sequence, got[i-1].Sequence)
		}
	}
	// Expected ordering matches insertion order.
	for i, env := range envs {
		if got[i].EventType != env.EventType {
			t.Errorf("ReadUnacked[%d].EventType = %q; esperado %q", i, got[i].EventType, env.EventType)
		}
		if got[i].Sequence != int64(i+1) {
			t.Errorf("ReadUnacked[%d].Sequence = %d; esperado %d", i, got[i].Sequence, i+1)
		}
	}
}

func TestLog_DuplicateEventID_IsIdempotent(t *testing.T) {
	conn, cleanup := newTestDB(t)
	defer cleanup()

	log := NewLog(conn)
	ctx := context.Background()

	env := NewEnvelope("memory.committed", "documents/foo.md")

	// First insert succeeds.
	tx1, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx 1 falhou: %v", err)
	}
	if err := log.Append(ctx, tx1, env); err != nil {
		tx1.Rollback()
		t.Fatalf("Append 1 falhou: %v", err)
	}
	if err := tx1.Commit(); err != nil {
		t.Fatalf("Commit 1 falhou: %v", err)
	}

	// Second insert with the same event_id must fail with ErrDuplicateEventID.
	tx2, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx 2 falhou: %v", err)
	}
	defer tx2.Rollback()
	err = log.Append(ctx, tx2, env)
	if err == nil {
		t.Fatal("Append 2 deveria falhar em event_id duplicado")
	}
	if !errors.Is(err, ErrDuplicateEventID) {
		t.Errorf("Append 2 deveria retornar ErrDuplicateEventID; obteve %v", err)
	}
}

func TestLog_PayloadTooLarge_IsRejected(t *testing.T) {
	conn, cleanup := newTestDB(t)
	defer cleanup()

	log := NewLog(conn)
	ctx := context.Background()

	env := NewEnvelope("memory.committed", "documents/foo.md")
	// 2 MiB > 1 MiB hard limit.
	env.Payload = make([]byte, 2*MaxPayloadBytes)
	for i := range env.Payload {
		env.Payload[i] = 'x'
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx falhou: %v", err)
	}
	defer tx.Rollback()

	err = log.Append(ctx, tx, env)
	if err == nil {
		t.Fatal("Append deveria rejeitar payload > 1 MiB")
	}
	if !errors.Is(err, ErrPayloadTooLarge) {
		t.Errorf("Append deveria retornar ErrPayloadTooLarge; obteve %v", err)
	}

	// Verify nothing was written by rolling back and querying LastSequence.
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback falhou: %v", err)
	}
	seq, err := log.LastSequence(ctx)
	if err != nil {
		t.Fatalf("LastSequence falhou: %v", err)
	}
	if seq != 0 {
		t.Errorf("LastSequence = %d; esperado 0 após rejeição", seq)
	}
}

func TestLog_AppendNilTx_IsRejected(t *testing.T) {
	conn, cleanup := newTestDB(t)
	defer cleanup()

	log := NewLog(conn)
	ctx := context.Background()

	env := NewEnvelope("memory.committed", "documents/foo.md")
	err := log.Append(ctx, nil, env)
	if err == nil {
		t.Fatal("Append com tx=nil deveria falhar")
	}
	if !errors.Is(err, ErrTxRequired) {
		t.Errorf("Append deveria retornar ErrTxRequired; obteve %v", err)
	}
}

// Sanity check: ReadUnacked with cursor only returns envelopes past cursor.
func TestLog_ReadUnacked_RespectsCursor(t *testing.T) {
	conn, cleanup := newTestDB(t)
	defer cleanup()

	log := NewLog(conn)
	ctx := context.Background()

	// Append 4 envelopes.
	envs := []*Envelope{
		NewEnvelope("a.x", "agg-1"),
		NewEnvelope("b.x", "agg-2"),
		NewEnvelope("c.x", "agg-3"),
		NewEnvelope("d.x", "agg-4"),
	}
	for _, env := range envs {
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx falhou: %v", err)
		}
		if err := log.Append(ctx, tx, env); err != nil {
			tx.Rollback()
			t.Fatalf("Append falhou: %v", err)
		}
		if err := tx.Commit(); err != nil {
			t.Fatalf("Commit falhou: %v", err)
		}
	}

	// Advance cursor for "sub-A" to sequence 2.
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx cursor falhou: %v", err)
	}
	if err := log.AdvanceCursor(ctx, tx, "sub-A", 2); err != nil {
		tx.Rollback()
		t.Fatalf("AdvanceCursor falhou: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit cursor falhou: %v", err)
	}

	// Read for sub-A: should skip seq 1+2 → only seq 3,4.
	got, err := log.ReadUnacked(ctx, "sub-A", 10)
	if err != nil {
		t.Fatalf("ReadUnacked sub-A falhou: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ReadUnacked sub-A retornou %d; esperado 2", len(got))
	}
	if got[0].Sequence != 3 || got[1].Sequence != 4 {
		t.Errorf("sub-A retornou seq [%d, %d]; esperado [3, 4]", got[0].Sequence, got[1].Sequence)
	}

	// Read for sub-B (no cursor) → all 4.
	got, err = log.ReadUnacked(ctx, "sub-B", 10)
	if err != nil {
		t.Fatalf("ReadUnacked sub-B falhou: %v", err)
	}
	if len(got) != 4 {
		t.Errorf("sub-B retornou %d; esperado 4", len(got))
	}
}

// MarkAcked + AdvanceCursor in the same tx hides events from ReadUnacked.
func TestLog_MarkAcked_AndAdvanceCursor_AreAtomic(t *testing.T) {
	conn, cleanup := newTestDB(t)
	defer cleanup()

	log := NewLog(conn)
	ctx := context.Background()

	env := NewEnvelope("memory.committed", "documents/foo.md")
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx falhou: %v", err)
	}
	if err := log.Append(ctx, tx, env); err != nil {
		tx.Rollback()
		t.Fatalf("Append falhou: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit falhou: %v", err)
	}

	// Without ack, ReadUnacked returns it.
	got, _ := log.ReadUnacked(ctx, "audit", 10)
	if len(got) != 1 {
		t.Fatalf("antes do ACK, ReadUnacked retornou %d; esperado 1", len(got))
	}

	// ACK + advance cursor in same tx.
	tx2, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx 2 falhou: %v", err)
	}
	if err := log.MarkAcked(ctx, tx2, env.Sequence, ""); err != nil {
		tx2.Rollback()
		t.Fatalf("MarkAcked falhou: %v", err)
	}
	if err := log.AdvanceCursor(ctx, tx2, "audit", env.Sequence); err != nil {
		tx2.Rollback()
		t.Fatalf("AdvanceCursor falhou: %v", err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatalf("Commit 2 falhou: %v", err)
	}

	// Now ReadUnacked returns nothing for "audit".
	got, _ = log.ReadUnacked(ctx, "audit", 10)
	if len(got) != 0 {
		t.Errorf("após ACK, ReadUnacked retornou %d; esperado 0", len(got))
	}
}

// LastSequence returns 0 on empty table.
func TestLog_LastSequence_Empty(t *testing.T) {
	conn, cleanup := newTestDB(t)
	defer cleanup()

	log := NewLog(conn)
	ctx := context.Background()
	seq, err := log.LastSequence(ctx)
	if err != nil {
		t.Fatalf("LastSequence falhou: %v", err)
	}
	if seq != 0 {
		t.Errorf("LastSequence em tabela vazia = %d; esperado 0", seq)
	}
}

// Belt-and-suspenders: ensure the unused imports / helpers don't bitrot.
func TestLog_SmokeTest_KeyTypes(t *testing.T) {
	// We don't assert behavior, only that round-trip through the SQL row
	// preserves Payload bytes.
	conn, cleanup := newTestDB(t)
	defer cleanup()
	log := NewLog(conn)
	ctx := context.Background()
	env := NewEnvelope("memory.committed", "documents/foo.md")
	env.Payload = json.RawMessage(`{"k":"v"}`)
	tx, _ := conn.BeginTx(ctx, nil)
	if err := log.Append(ctx, tx, env); err != nil {
		t.Fatalf("Append falhou: %v", err)
	}
	tx.Commit()
	got, _ := log.ReadUnacked(ctx, "smoke", 10)
	if len(got) != 1 {
		t.Fatalf("ReadUnacked retornou %d", len(got))
	}
	if !strings.Contains(string(got[0].Payload), `"k":"v"`) {
		t.Errorf("Payload não preservado: %s", got[0].Payload)
	}
}
