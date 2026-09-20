package writer

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// documents schema (subset of what ADR-044 requires; revision + id
	// + path are the only columns precondition tests need).
	_, err = db.Exec(`
		CREATE TABLE documents (
			id TEXT PRIMARY KEY,
			path TEXT NOT NULL UNIQUE,
			revision INTEGER NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		t.Fatalf("create documents table: %v", err)
	}
	return db
}

func upsertDoc(t *testing.T, db *sql.DB, id, path string, revision int64) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO documents (id, path, revision) VALUES (?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET revision = excluded.revision`,
		id, path, revision,
	)
	if err != nil {
		t.Fatalf("upsert doc: %v", err)
	}
}

func TestCheck_Match(t *testing.T) {
	db := newTestDB(t)
	upsertDoc(t, db, "doc-1", "/tmp/foo.md", 5)

	if err := Check(context.Background(), db, "doc-1", 5); err != nil {
		t.Errorf("Check(rev=5, current=5) = %v, want nil", err)
	}
}

func TestCheck_Mismatch(t *testing.T) {
	db := newTestDB(t)
	upsertDoc(t, db, "doc-1", "/tmp/foo.md", 7)

	err := Check(context.Background(), db, "doc-1", 5)
	if err == nil {
		t.Fatal("expected error on mismatch, got nil")
	}
	if !errors.Is(err, ErrPreconditionFailed) {
		t.Errorf("expected ErrPreconditionFailed, got %v", err)
	}
	var pe *PreconditionError
	if !errors.As(err, &pe) {
		t.Fatal("expected *PreconditionError via errors.As")
	}
	if pe.Expected != 5 {
		t.Errorf("Expected=%d, want 5", pe.Expected)
	}
	if pe.Current != 7 {
		t.Errorf("Current=%d, want 7", pe.Current)
	}
	if pe.DocumentID != "doc-1" {
		t.Errorf("DocumentID=%q, want doc-1", pe.DocumentID)
	}
}

func TestCheck_CreationOnExisting(t *testing.T) {
	db := newTestDB(t)
	upsertDoc(t, db, "doc-1", "/tmp/foo.md", 3)

	err := Check(context.Background(), db, "doc-1", 0)
	if err == nil {
		t.Fatal("creation-on-existing must fail")
	}
	if !errors.Is(err, ErrPreconditionFailed) {
		t.Errorf("expected ErrPreconditionFailed, got %v", err)
	}
	var pe *PreconditionError
	if !errors.As(err, &pe) {
		t.Fatal("expected *PreconditionError")
	}
	if pe.Expected != 0 || pe.Current != 3 {
		t.Errorf("Expected=%d Current=%d, want 0/3", pe.Expected, pe.Current)
	}
}

func TestCheck_CreationOnMissing(t *testing.T) {
	db := newTestDB(t)
	// document "doc-new" does not exist; expected=0 means create
	if err := Check(context.Background(), db, "doc-new", 0); err != nil {
		t.Errorf("creation-on-missing must succeed, got %v", err)
	}
}

func TestCheck_UpdateOnMissing(t *testing.T) {
	db := newTestDB(t)
	// caller asked for rev=5 on a non-existent doc; should fail with Current=0
	err := Check(context.Background(), db, "doc-ghost", 5)
	if err == nil {
		t.Fatal("update-on-missing must fail")
	}
	var pe *PreconditionError
	if !errors.As(err, &pe) {
		t.Fatal("expected *PreconditionError")
	}
	if pe.Current != 0 {
		t.Errorf("Current=%d, want 0", pe.Current)
	}
	if pe.Expected != 5 {
		t.Errorf("Expected=%d, want 5", pe.Expected)
	}
}

func TestCheck_NegativeRevision(t *testing.T) {
	db := newTestDB(t)
	upsertDoc(t, db, "doc-1", "/tmp/foo.md", 1)

	err := Check(context.Background(), db, "doc-1", -1)
	if !errors.Is(err, ErrInvalidRevision) {
		t.Errorf("expected ErrInvalidRevision, got %v", err)
	}
	if errors.Is(err, ErrPreconditionFailed) {
		t.Errorf("ErrInvalidRevision must NOT match ErrPreconditionFailed")
	}
}

func TestCheck_ContextCancelled(t *testing.T) {
	db := newTestDB(t)
	upsertDoc(t, db, "doc-1", "/tmp/foo.md", 1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Check(ctx, db, "doc-1", 1)
	// We don't pin to a specific error here — sqlite may return
	// "context canceled" or "interrupted" depending on platform; the
	// only invariant is that it is NOT ErrPreconditionFailed.
	if errors.Is(err, ErrPreconditionFailed) {
		t.Errorf("cancelled context must not be reported as a precondition failure")
	}
}
