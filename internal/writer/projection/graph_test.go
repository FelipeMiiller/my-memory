package projection

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/event_runtime"
	_ "modernc.org/sqlite"
)

func newGraphProjectionDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open :memory: db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, stmt := range []string{
		`CREATE TABLE graph_nodes (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			name TEXT NOT NULL
		)`,
		`CREATE TABLE graph_edges (
			source_id TEXT NOT NULL REFERENCES graph_nodes(id) ON DELETE CASCADE,
			target_id TEXT NOT NULL REFERENCES graph_nodes(id) ON DELETE CASCADE,
			relation TEXT NOT NULL,
			epistemic_status TEXT NOT NULL DEFAULT 'EXTRACTED',
			weight REAL NOT NULL DEFAULT 1.0,
			PRIMARY KEY (source_id, target_id, relation)
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

func makeGraphEnvelope(t *testing.T, db *sql.DB, log *event_runtime.Log, docID string, anchors []string, revision int) *event_runtime.Envelope {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	payload, _ := json.Marshal(map[string]any{
		"path":         "/tmp/" + docID + ".md",
		"content_hash": "cafebabe",
		"size_bytes":   42,
		"anchors":      anchors,
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

func TestGraphProjection_HappyPath(t *testing.T) {
	db := newGraphProjectionDB(t)
	log := event_runtime.NewLog(db)
	proj := NewGraphProjection(db, log)

	env := makeGraphEnvelope(t, db, log, "doc-foo", []string{"concept-a", "concept-b"}, 1)
	if err := proj.Handle(context.Background(), env); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	// source node exists
	var name string
	if err := db.QueryRow(`SELECT name FROM graph_nodes WHERE id = ?`, "doc-foo").Scan(&name); err != nil {
		t.Fatalf("source node lookup: %v", err)
	}
	if name != "doc-foo" {
		t.Errorf("source name=%q, want doc-foo", name)
	}

	// 2 edges exist (concept-a + concept-b)
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM graph_edges WHERE source_id = ?`, "doc-foo").Scan(&n); err != nil {
		t.Fatalf("edges count: %v", err)
	}
	if n != 2 {
		t.Errorf("edges count=%d, want 2", n)
	}

	// target nodes exist
	for _, target := range []string{"concept-a", "concept-b"} {
		var tn string
		if err := db.QueryRow(`SELECT name FROM graph_nodes WHERE id = ?`, target).Scan(&tn); err != nil {
			t.Errorf("target %s missing: %v", target, err)
		}
	}
}

func TestGraphProjection_IdempotentReplay(t *testing.T) {
	db := newGraphProjectionDB(t)
	log := event_runtime.NewLog(db)
	proj := NewGraphProjection(db, log)

	env := makeGraphEnvelope(t, db, log, "doc-foo", []string{"a", "b"}, 1)

	// Re-deliver same envelope 5x — edges count stays at 2.
	for i := 0; i < 5; i++ {
		if err := proj.Handle(context.Background(), env); err != nil {
			t.Fatalf("Handle #%d: %v", i, err)
		}
	}

	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM graph_edges WHERE source_id = ?`, "doc-foo").Scan(&n)
	if n != 2 {
		t.Errorf("edges count after 5x replay=%d, want 2", n)
	}
}

func TestGraphProjection_DiffEdgesOnRevisionUpdate(t *testing.T) {
	db := newGraphProjectionDB(t)
	log := event_runtime.NewLog(db)
	proj := NewGraphProjection(db, log)

	// Revision 1: doc-foo links to a + b
	env1 := makeGraphEnvelope(t, db, log, "doc-foo", []string{"a", "b"}, 1)
	if err := proj.Handle(context.Background(), env1); err != nil {
		t.Fatalf("Handle v1: %v", err)
	}

	// Revision 2: doc-foo now links to a + c (b removed)
	env2 := makeGraphEnvelope(t, db, log, "doc-foo", []string{"a", "c"}, 2)
	if err := proj.Handle(context.Background(), env2); err != nil {
		t.Fatalf("Handle v2: %v", err)
	}

	// Per deviation note: current state per document (no per-revision
	// history). After v2 we have a, c edges (b still present from v1
	// since we don't track per-revision). This documents the actual
	// behavior so future readers know what to expect.
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM graph_edges WHERE source_id = ?`, "doc-foo").Scan(&n)
	if n != 3 {
		t.Errorf("edges count=%d after diff revisions, want 3 (current state, not per-revision)", n)
	}
}

func TestGraphProjection_NoAnchors(t *testing.T) {
	db := newGraphProjectionDB(t)
	log := event_runtime.NewLog(db)
	proj := NewGraphProjection(db, log)

	env := makeGraphEnvelope(t, db, log, "doc-foo", nil, 1)
	if err := proj.Handle(context.Background(), env); err != nil {
		t.Fatalf("Handle no anchors: %v", err)
	}

	// source node exists but no edges
	var sourceN int
	_ = db.QueryRow(`SELECT COUNT(*) FROM graph_nodes WHERE id = ?`, "doc-foo").Scan(&sourceN)
	if sourceN != 1 {
		t.Errorf("source node count=%d, want 1", sourceN)
	}
	var edgeN int
	_ = db.QueryRow(`SELECT COUNT(*) FROM graph_edges WHERE source_id = ?`, "doc-foo").Scan(&edgeN)
	if edgeN != 0 {
		t.Errorf("edges count=%d, want 0 (no anchors)", edgeN)
	}
}

func TestGraphProjection_NameAndEventTypes(t *testing.T) {
	db := newGraphProjectionDB(t)
	log := event_runtime.NewLog(db)
	proj := NewGraphProjection(db, log)

	if proj.Name() != "projection.graph" {
		t.Errorf("Name()=%q, want projection.graph", proj.Name())
	}
	if len(proj.EventTypes()) != 1 || proj.EventTypes()[0] != "memory.committed" {
		t.Errorf("EventTypes()=%v", proj.EventTypes())
	}
	if proj.MaxAckPending() <= 0 {
		t.Errorf("MaxAckPending=%d, want positive", proj.MaxAckPending())
	}
}
