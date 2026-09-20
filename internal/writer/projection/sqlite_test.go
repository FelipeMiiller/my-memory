package projection

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/event_runtime"
	_ "modernc.org/sqlite"
)

// newProjectionDB returns a fresh :memory: SQLite with the full schema
// the projection touches (documents, chunks, chunks_fts, event_log,
// projection_cursor). Mirrors the schema used by internal/db.Schema.
func newProjectionDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, stmt := range []string{
		`CREATE TABLE documents (
			id TEXT PRIMARY KEY,
			path TEXT NOT NULL UNIQUE,
			title TEXT,
			updated_at INTEGER NOT NULL,
			content_hash TEXT,
			abstract TEXT,
			category TEXT DEFAULT 'resource',
			revision INTEGER NOT NULL DEFAULT 0,
			last_event_id TEXT
		)`,
		`CREATE TABLE chunks (
			id TEXT PRIMARY KEY,
			document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
			chunk_index INTEGER NOT NULL,
			content TEXT NOT NULL
		)`,
		`CREATE VIRTUAL TABLE chunks_fts USING fts5(
			chunk_id UNINDEXED,
			content,
			tokenize = 'porter unicode61'
		)`,
		`CREATE TABLE event_log (
			sequence INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id TEXT UNIQUE NOT NULL,
			schema_version INTEGER NOT NULL,
			event_type TEXT NOT NULL,
			aggregate_id TEXT NOT NULL,
			revision INTEGER,
			payload BLOB NOT NULL,
			headers BLOB NOT NULL,
			created_at TEXT NOT NULL,
			acked_at TEXT
		)`,
		`CREATE TABLE projection_cursor (
			projection_name TEXT PRIMARY KEY,
			last_sequence INTEGER NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("schema: %v", err)
		}
	}
	return db
}

// makeCommittedEnvelope constructs a memory.committed envelope with
// the given payload and persists it via event_runtime.Log.Append so the
// sequence column is populated. The projection needs Sequence to be
// set when calling MarkAcked / AdvanceCursor.
func makeCommittedEnvelope(t *testing.T, db *sql.DB, log *event_runtime.Log, docID, path, content string, revision int) *event_runtime.Envelope {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	payload, _ := json.Marshal(map[string]any{
		"path":         path,
		"content_hash": "deadbeef",
		"size_bytes":   len(content),
		"anchors":      []string{},
	})

	env := event_runtime.NewEnvelope("memory.committed", docID)
	env.Actor = "user:test"
	env.Revision = revision
	env.Payload = payload

	if err := log.Append(context.Background(), tx, env); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	return env
}

func TestSqliteProjection_HappyPath(t *testing.T) {
	db := newProjectionDB(t)
	log := event_runtime.NewLog(db)
	proj := NewSqliteProjection(db, log)

	dir := t.TempDir()
	path := filepath.Join(dir, "foo.md")
	content := "# Hello\nSome markdown."
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write tmp md: %v", err)
	}

	env := makeCommittedEnvelope(t, db, log, "doc-foo", path, content, 1)
	if err := proj.Handle(context.Background(), env); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	// documents row exists with right content_hash
	var gotHash string
	if err := db.QueryRow(`SELECT content_hash FROM documents WHERE id = ?`, "doc-foo").Scan(&gotHash); err != nil {
		t.Fatalf("documents lookup: %v", err)
	}
	if gotHash != "deadbeef" {
		t.Errorf("content_hash=%q, want deadbeef", gotHash)
	}

	// chunks row populated with full content
	var gotContent string
	if err := db.QueryRow(`SELECT content FROM chunks WHERE document_id = ?`, "doc-foo").Scan(&gotContent); err != nil {
		t.Fatalf("chunks lookup: %v", err)
	}
	if gotContent != content {
		t.Errorf("chunks.content=%q, want %q", gotContent, content)
	}

	// chunks_fts indexes the content (FTS5 MATCH sanity check)
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM chunks_fts WHERE chunks_fts MATCH ?`, "markdown").Scan(&n); err != nil {
		t.Fatalf("chunks_fts lookup: %v", err)
	}
	if n != 1 {
		t.Errorf("chunks_fts MATCH markdown count=%d, want 1", n)
	}

	// projection_cursor advanced to env.Sequence
	var cursor int64
	if err := db.QueryRow(`SELECT last_sequence FROM projection_cursor WHERE projection_name = ?`,
		proj.Name()).Scan(&cursor); err != nil {
		t.Fatalf("cursor lookup: %v", err)
	}
	if cursor != env.Sequence {
		t.Errorf("projection_cursor=%d, want %d", cursor, env.Sequence)
	}
}

func TestSqliteProjection_IdempotentReplay(t *testing.T) {
	db := newProjectionDB(t)
	log := event_runtime.NewLog(db)
	proj := NewSqliteProjection(db, log)

	dir := t.TempDir()
	path := filepath.Join(dir, "foo.md")
	if err := os.WriteFile(path, []byte("hello world"), 0o644); err != nil {
		t.Fatalf("write tmp md: %v", err)
	}

	env := makeCommittedEnvelope(t, db, log, "doc-foo", path, "hello world", 1)

	// Re-deliver the same envelope 5 times — only one chunks row
	// should remain (idempotency via INSERT OR IGNORE on chunk id).
	for i := 0; i < 5; i++ {
		if err := proj.Handle(context.Background(), env); err != nil {
			t.Fatalf("Handle #%d: %v", i, err)
		}
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM chunks WHERE document_id = ?`, "doc-foo").Scan(&n); err != nil {
		t.Fatalf("count chunks: %v", err)
	}
	if n != 1 {
		t.Errorf("chunks row count=%d after 5x replay, want 1", n)
	}
}

func TestSqliteProjection_MultipleRevisions(t *testing.T) {
	db := newProjectionDB(t)
	log := event_runtime.NewLog(db)
	proj := NewSqliteProjection(db, log)

	dir := t.TempDir()
	path := filepath.Join(dir, "foo.md")

	// revision 1
	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatalf("write v1: %v", err)
	}
	env1 := makeCommittedEnvelope(t, db, log, "doc-foo", path, "v1", 1)
	if err := proj.Handle(context.Background(), env1); err != nil {
		t.Fatalf("Handle v1: %v", err)
	}

	// revision 2 — different chunk id (revision in id), same document_id
	if err := os.WriteFile(path, []byte("v2 longer"), 0o644); err != nil {
		t.Fatalf("write v2: %v", err)
	}
	env2 := makeCommittedEnvelope(t, db, log, "doc-foo", path, "v2 longer", 2)
	if err := proj.Handle(context.Background(), env2); err != nil {
		t.Fatalf("Handle v2: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM chunks WHERE document_id = ?`, "doc-foo").Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Errorf("chunks count=%d after 2 revisions, want 2", n)
	}

	// latest revision wins for last_event_id on documents row
	// (UPSERT overwrites the field; we don't store last_event_id in
	// documents here, so just verify projection_cursor advanced to
	// the latest sequence).
	var cursor int64
	_ = db.QueryRow(`SELECT last_sequence FROM projection_cursor WHERE projection_name = ?`,
		proj.Name()).Scan(&cursor)
	if cursor != env2.Sequence {
		t.Errorf("projection_cursor=%d, want %d (env2.Sequence)", cursor, env2.Sequence)
	}
}

func TestSqliteProjection_MissingFileReturnsError(t *testing.T) {
	db := newProjectionDB(t)
	log := event_runtime.NewLog(db)
	proj := NewSqliteProjection(db, log)

	// Use a path that points at a file we'll then delete so ReadFile fails
	dir := t.TempDir()
	path := filepath.Join(dir, "foo.md")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	env := makeCommittedEnvelope(t, db, log, "doc-foo", path, "hello", 1)
	_ = os.Remove(path)

	err := proj.Handle(context.Background(), env)
	if err == nil {
		t.Fatal("expected error when .md is missing")
	}
}

func TestSqliteProjection_NameAndEventTypes(t *testing.T) {
	db := newProjectionDB(t)
	log := event_runtime.NewLog(db)
	proj := NewSqliteProjection(db, log)

	if proj.Name() != "projection.sqlite" {
		t.Errorf("Name()=%q, want projection.sqlite", proj.Name())
	}
	types := proj.EventTypes()
	if len(types) != 1 || types[0] != "memory.committed" {
		t.Errorf("EventTypes()=%v, want [memory.committed]", types)
	}
	if proj.MaxAckPending() <= 0 {
		t.Errorf("MaxAckPending must be positive, got %d", proj.MaxAckPending())
	}
}
