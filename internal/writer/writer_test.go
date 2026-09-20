package writer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FelipeMiiller/my-memory/internal/event_runtime"
	_ "modernc.org/sqlite"
)

// newWriterDB returns a fresh in-memory SQLite with the minimal schema
// surface the writer needs (documents + event_log + projection_cursor)
// — same shape as internal/db.Schema but scoped to the writer tests.
func newWriterDB(t *testing.T) *sql.DB {
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
			t.Fatalf("create table: %v\nstmt=%s", err, stmt)
		}
	}
	return db
}

func tempDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("", "writer-test-*")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(d) })
	return d
}

func writeOK(t *testing.T, w *Writer, path string, content []byte, expected *int64) WriteResult {
	t.Helper()
	req := WriteRequest{
		Path:             path,
		Content:          content,
		Actor:            "user:test",
		CorrelationID:    "corr-test",
		ExpectedRevision: expected,
	}
	res, err := w.Write(context.Background(), req)
	if err != nil {
		t.Fatalf("Write(%q): %v", path, err)
	}
	return res
}

func TestWrite_HappyPath(t *testing.T) {
	db := newWriterDB(t)
	log := event_runtime.NewLog(db)
	w := New(db, log)
	dir := tempDir(t)
	path := filepath.Join(dir, "foo.md")

	res := writeOK(t, w, path, []byte("# Hello\n[[wikilink-target]]"), nil)

	if res.EventID == "" {
		t.Error("EventID must be set")
	}
	if res.Sequence <= 0 {
		t.Errorf("Sequence must be > 0, got %d", res.Sequence)
	}
	if res.Revision != 1 {
		t.Errorf("Revision=%d, want 1", res.Revision)
	}
	if len(res.ContentHash) != 64 {
		t.Errorf("ContentHash must be 64 hex chars, got %d", len(res.ContentHash))
	}

	// Disk: file exists with the right content
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "# Hello\n[[wikilink-target]]" {
		t.Errorf("file content mismatch: %q", got)
	}

	// SQLite: documents row reflects the write
	var rev int64
	var lastEv string
	var hash string
	if err := db.QueryRow(`SELECT revision, last_event_id, content_hash FROM documents WHERE id = ?`,
		res.EventID).Scan(&rev, &lastEv, &hash); err != nil {
		// DocumentID is empty in test; documents.id is computed from path
		// Re-query by path
		_ = db.QueryRow(`SELECT revision, last_event_id, content_hash FROM documents WHERE path = ?`,
			path).Scan(&rev, &lastEv, &hash)
	}
	if rev != 1 {
		t.Errorf("documents.revision=%d, want 1", rev)
	}
	if lastEv != res.EventID {
		t.Errorf("documents.last_event_id=%q, want %q", lastEv, res.EventID)
	}
	if hash != res.ContentHash {
		t.Errorf("documents.content_hash=%q, want %q", hash, res.ContentHash)
	}

	// event_log: exactly one memory.committed envelope. The payload
	// column stores the FULL envelope JSON (event_runtime contract),
	// so we unmarshal as Envelope and then look at the inner Payload.
	var et string
	var rawPayload []byte
	if err := db.QueryRow(`SELECT event_type, payload FROM event_log WHERE event_id = ?`,
		res.EventID).Scan(&et, &rawPayload); err != nil {
		t.Fatalf("event_log lookup: %v", err)
	}
	if et != "memory.committed" {
		t.Errorf("event_type=%q, want memory.committed", et)
	}
	var env event_runtime.Envelope
	if err := json.Unmarshal(rawPayload, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.EventType != "memory.committed" {
		t.Errorf("envelope.event_type=%q, want memory.committed", env.EventType)
	}
	var p committedPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		t.Fatalf("unmarshal committedPayload: %v\nenv.Payload=%s", err, env.Payload)
	}
	if p.Path != path {
		t.Errorf("payload.path=%q, want %q", p.Path, path)
	}
	if p.ContentHash != res.ContentHash {
		t.Errorf("payload.content_hash mismatch: got %q want %q", p.ContentHash, res.ContentHash)
	}
	if len(p.Anchors) != 1 || p.Anchors[0] != "wikilink-target" {
		t.Errorf("payload.anchors=%v, want [wikilink-target]", p.Anchors)
	}
}

func TestWrite_PreconditionMatch(t *testing.T) {
	db := newWriterDB(t)
	log := event_runtime.NewLog(db)
	w := New(db, log)
	dir := tempDir(t)
	path := filepath.Join(dir, "foo.md")

	rev0 := int64(0)
	res := writeOK(t, w, path, []byte("v1"), &rev0)
	if res.Revision != 1 {
		t.Errorf("first write revision=%d, want 1", res.Revision)
	}

	rev1 := int64(1)
	res2 := writeOK(t, w, path, []byte("v2"), &rev1)
	if res2.Revision != 2 {
		t.Errorf("second write revision=%d, want 2", res2.Revision)
	}
}

func TestWrite_PreconditionMismatch(t *testing.T) {
	db := newWriterDB(t)
	log := event_runtime.NewLog(db)
	w := New(db, log)
	dir := tempDir(t)
	path := filepath.Join(dir, "foo.md")

	rev0 := int64(0)
	writeOK(t, w, path, []byte("v1"), &rev0)

	// Now ask for rev=99 — must fail with PreconditionError{Current: 1}
	stale := int64(99)
	_, err := w.Write(context.Background(), WriteRequest{
		Path:             path,
		Content:          []byte("v2-stale"),
		Actor:            "user:test",
		ExpectedRevision: &stale,
	})
	if err == nil {
		t.Fatal("expected precondition error")
	}
	if !errors.Is(err, ErrPreconditionFailed) {
		t.Errorf("expected ErrPreconditionFailed, got %v", err)
	}
	var pe *PreconditionError
	if !errors.As(err, &pe) || pe.Current != 1 || pe.Expected != 99 {
		t.Errorf("PreconditionError mismatch: %+v", pe)
	}

	// Compensating action: file MUST still be v1, not v2-stale
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "v1" {
		t.Errorf("file should be v1 after rejected write, got %q", got)
	}

	// T4: conflict.detected event must have been emitted (T4 contract)
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM event_log WHERE event_type = 'conflict.detected'`).Scan(&n); err != nil {
		t.Fatalf("count conflict.detected: %v", err)
	}
	if n != 1 {
		t.Errorf("conflict.detected count=%d, want 1", n)
	}

	// Validate the conflict.detected payload
	var raw []byte
	if err := db.QueryRow(`SELECT payload FROM event_log WHERE event_type = 'conflict.detected' LIMIT 1`).Scan(&raw); err != nil {
		t.Fatalf("read conflict payload: %v", err)
	}
	var env event_runtime.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	var cp conflictPayload
	if err := json.Unmarshal(env.Payload, &cp); err != nil {
		t.Fatalf("unmarshal conflictPayload: %v", err)
	}
	if cp.Expected != 99 || cp.Current != 1 {
		t.Errorf("conflict payload: expected=%d current=%d, want 99/1", cp.Expected, cp.Current)
	}
	if cp.DocumentID == "" {
		t.Error("conflict payload: document_id missing")
	}
	if cp.Path != path {
		t.Errorf("conflict payload: path=%q want %q", cp.Path, path)
	}
	if cp.Reason != "precondition_failed" {
		t.Errorf("conflict payload: reason=%q want precondition_failed", cp.Reason)
	}
}

func TestWrite_CreationConflictEmitsConflict(t *testing.T) {
	db := newWriterDB(t)
	log := event_runtime.NewLog(db)
	w := New(db, log)
	dir := tempDir(t)
	path := filepath.Join(dir, "foo.md")

	rev0 := int64(0)
	writeOK(t, w, path, []byte("v1"), &rev0)

	// Now ask to create (expected=0) again — must fail
	_, err := w.Write(context.Background(), WriteRequest{
		Path:             path,
		Content:          []byte("v2-create-conflict"),
		Actor:            "user:test",
		ExpectedRevision: &rev0,
	})
	if !errors.Is(err, ErrPreconditionFailed) {
		t.Errorf("creation conflict must fail with ErrPreconditionFailed, got %v", err)
	}

	// conflict.detected must still be emitted for creation conflicts
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM event_log WHERE event_type = 'conflict.detected'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("conflict.detected count=%d, want 1 for creation conflict", n)
	}

	// File unchanged
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "v1" {
		t.Errorf("file should still be v1 after creation conflict, got %q", got)
	}
}

func TestWrite_NoMdFailsRollsBack(t *testing.T) {
	db := newWriterDB(t)
	log := event_runtime.NewLog(db)
	w := New(db, log)
	dir := tempDir(t)
	path := filepath.Join(dir, "foo.md")

	// Use a path under a directory we then delete so the rename will
	// fail with a not-exist error. This simulates "disk full" / "dir
	// removed" without needing real filesystem fault injection.
	_ = os.RemoveAll(dir)

	_, err := w.Write(context.Background(), WriteRequest{
		Path:    path,
		Content: []byte("will fail"),
		Actor:   "user:test",
	})
	if err == nil {
		t.Fatal("expected error on missing dir")
	}
	if !errors.Is(err, ErrWriteFailed) {
		t.Errorf("expected ErrWriteFailed, got %v", err)
	}

	// No documents row should have been created
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM documents WHERE path = ?`, path).Scan(&n)
	if n != 0 {
		t.Errorf("documents row count=%d, want 0 (tx rolled back)", n)
	}
	// No event_log row either
	_ = db.QueryRow(`SELECT COUNT(*) FROM event_log`).Scan(&n)
	if n != 0 {
		t.Errorf("event_log count=%d, want 0 (tx rolled back)", n)
	}
}

func TestWrite_NoEventEmittedIfMdFails(t *testing.T) {
	db := newWriterDB(t)
	log := event_runtime.NewLog(db)
	w := New(db, log)
	dir := tempDir(t)
	path := filepath.Join(dir, "subdir", "foo.md") // subdir does not exist

	_, err := w.Write(context.Background(), WriteRequest{
		Path:    path,
		Content: []byte("will fail"),
		Actor:   "user:test",
	})
	if err == nil {
		t.Fatal("expected error on missing parent dir")
	}

	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM event_log`).Scan(&n)
	if n != 0 {
		t.Errorf("event_log count=%d after failed .md write, want 0", n)
	}
}

func TestWrite_EmptyContentRejected(t *testing.T) {
	db := newWriterDB(t)
	log := event_runtime.NewLog(db)
	w := New(db, log)
	dir := tempDir(t)
	path := filepath.Join(dir, "foo.md")

	_, err := w.Write(context.Background(), WriteRequest{
		Path:    path,
		Content: nil,
		Actor:   "user:test",
	})
	if !errors.Is(err, ErrWriteFailed) {
		t.Errorf("empty content must return ErrWriteFailed, got %v", err)
	}
}

func TestWrite_RelativePathRejected(t *testing.T) {
	db := newWriterDB(t)
	log := event_runtime.NewLog(db)
	w := New(db, log)

	_, err := w.Write(context.Background(), WriteRequest{
		Path:    "relative/foo.md",
		Content: []byte("nope"),
		Actor:   "user:test",
	})
	if !errors.Is(err, ErrWriteFailed) {
		t.Errorf("relative path must return ErrWriteFailed, got %v", err)
	}
}

func TestWrite_OversizeContentRejected(t *testing.T) {
	db := newWriterDB(t)
	log := event_runtime.NewLog(db)
	w := New(db, log)
	dir := tempDir(t)
	path := filepath.Join(dir, "foo.md")

	huge := make([]byte, MaxWriteBytes+1)
	for i := range huge {
		huge[i] = 'x'
	}
	_, err := w.Write(context.Background(), WriteRequest{
		Path:    path,
		Content: huge,
		Actor:   "user:test",
	})
	if !errors.Is(err, ErrWriteFailed) {
		t.Errorf("oversize content must return ErrWriteFailed, got %v", err)
	}
}

func TestWrite_OversizeWarningFlag(t *testing.T) {
	db := newWriterDB(t)
	log := event_runtime.NewLog(db)
	w := New(db, log)
	dir := tempDir(t)
	path := filepath.Join(dir, "foo.md")

	// WarnWriteBytes+1 — over the warning threshold but under the hard limit
	content := make([]byte, WarnWriteBytes+1)
	for i := range content {
		content[i] = 'a'
	}
	res, err := w.Write(context.Background(), WriteRequest{
		Path:    path,
		Content: content,
		Actor:   "user:test",
	})
	if err != nil {
		t.Fatalf("over-warning write should succeed, got %v", err)
	}
	if !res.OversizeWarning {
		t.Errorf("OversizeWarning=true expected for content > %d bytes", WarnWriteBytes)
	}

	// Same on payload (the BLOB stores the full envelope JSON; unwrap
	// to get to the inner committedPayload).
	var rawPayload []byte
	if err := db.QueryRow(`SELECT payload FROM event_log WHERE event_id = ?`,
		res.EventID).Scan(&rawPayload); err != nil {
		t.Fatalf("payload lookup: %v", err)
	}
	var env event_runtime.Envelope
	if err := json.Unmarshal(rawPayload, &env); err != nil {
		t.Fatalf("envelope unmarshal: %v", err)
	}
	var p committedPayload
	_ = json.Unmarshal(env.Payload, &p)
	if !p.Oversize {
		t.Error("payload.Oversize=true expected")
	}
}

func TestExtractWikilinks(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"plain text", nil},
		{"[[single]]", []string{"single"}},
		{"[[a]] and [[b]]", []string{"a", "b"}},
		{"no close", nil},
		{"[[]] empty target skipped", nil},
		{"[[ spaced ]]", []string{"spaced"}},
		// Documents the lightweight greedy behavior: nested [[link]]
		// inside another [[ ]] is treated as one big target. The
		// proper parser (internal/parser) handles nesting; this
		// extractor is intentionally simple for T3.
		{"[[outer [[inner]] more]]", []string{"outer [[inner"}},
	}
	for _, tc := range cases {
		got := extractWikilinks([]byte(tc.in))
		if !equalStringSlices(got, tc.want) {
			t.Errorf("extractWikilinks(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestWriteMarkdownAtomic_Rename(t *testing.T) {
	dir := tempDir(t)
	target := filepath.Join(dir, "dest.md")
	if err := writeMarkdownAtomic(target, []byte("payload")); err != nil {
		t.Fatalf("writeMarkdownAtomic: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "payload" {
		t.Errorf("file content=%q, want payload", got)
	}
}

func TestWriteMarkdownAtomic_CleansUpTmpOnFailure(t *testing.T) {
	dir := tempDir(t)
	// Target lives in a deleted dir → rename will fail; tmp must be
	// cleaned up so the dir does not litter with .write-*.md.tmp files.
	subdir := filepath.Join(dir, "gone")
	bad := filepath.Join(subdir, "dest.md")
	_ = os.MkdirAll(subdir, 0o755)
	_ = os.RemoveAll(subdir)

	if err := writeMarkdownAtomic(bad, []byte("x")); err == nil {
		t.Fatal("expected error when target dir is missing")
	}

	// Count leftover tmp files in the original tempDir
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".write-") {
			t.Errorf("leftover tmp file: %s", e.Name())
		}
	}
}

func TestPathToDocID(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/tmp/foo.md", "doc-tmp-foo.md"},
		{"/var/vault/My Note.md", "doc-var-vault-my note.md"},
		{"C:\\Users\\foo\\bar.md", "doc-c:-users-foo-bar.md"},
	}
	for _, tc := range cases {
		got := pathToDocID(tc.in)
		if got != tc.want {
			t.Errorf("pathToDocID(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
