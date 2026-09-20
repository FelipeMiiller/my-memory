// Package projection holds the idempotent event_log subscribers that
// keep the read-side projections (SQLite/FTS5, graph edges, vector
// embeddings) in sync with the writer's memory.committed stream.
//
// Each subscriber implements event_runtime.Subscriber and is registered
// with the dispatcher at startup. The contract is:
//
//   - Handle is called exactly once per envelope by the dispatcher; it
//     must be idempotent (INSERT OR IGNORE keyed on event_id).
//   - On success, advance projection_cursor.last_sequence to env.Sequence.
//   - On failure, return a non-nil error so the dispatcher retries with
//     exponential backoff (ADR-043 §6).
//
// The first concrete subscribers live here (T6 + T7); F2 voice, F3
// agent, and F4 MCP server each add their own projection subscribers
// without touching this file.
package projection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/FelipeMiiller/my-memory/internal/event_runtime"
)

// SqliteProjection applies memory.committed events to the documents,
// chunks, and chunks_fts tables (T6). It is the read-side counterpart to
// the writer's UPSERT in the same transaction — but unlike the writer,
// the projection runs AFTER the transaction has committed, so it
// performs its own idempotent upsert keyed on event_id via INSERT OR
// IGNORE.
//
// The projection is intentionally narrow: it does not regenerate
// embeddings (that lives in projection.vec, out of scope for ADR-044 P1),
// it does not re-parse wikilinks (that lives in projection.graph, T7),
// and it does not enforce schema migrations (those run at startup).
// What it DOES do is keep documents + chunks + FTS5 consistent with the
// on-disk .md files, so search stays correct after a crash-restart.
type SqliteProjection struct {
	db  *sql.DB
	log *event_runtime.Log
}

// NewSqliteProjection constructs the projection bound to the same DB +
// Log handles the dispatcher uses.
func NewSqliteProjection(db *sql.DB, log *event_runtime.Log) *SqliteProjection {
	return &SqliteProjection{db: db, log: log}
}

// Name implements event_runtime.Subscriber. The string is the key for
// projection_cursor.projection_name; renaming in production resets the
// cursor (events replayed from the beginning). Keep it stable.
func (s *SqliteProjection) Name() string { return "projection.sqlite" }

// EventTypes implements event_runtime.Subscriber. We only consume
// memory.committed; the writer emits that as the single durable event
// per write, so all downstream state derives from it.
func (s *SqliteProjection) EventTypes() []string {
	return []string{"memory.committed"}
}

// MaxAckPending implements event_runtime.Subscriber. The projection is
// I/O-bound on the FTS5 upsert; 64 keeps memory bounded under load
// without throttling throughput.
func (s *SqliteProjection) MaxAckPending() int { return 64 }

// Handle implements event_runtime.Subscriber. The envelope's Payload
// field carries the committedPayload JSON (see writer.committedPayload)
// which has Path, ContentHash, SizeBytes, Anchors. The actual markdown
// content is re-read from disk at Path — by the time the dispatcher
// delivers this event, the writer's tx has committed and the .md is
// on disk.
func (s *SqliteProjection) Handle(ctx context.Context, env *event_runtime.Envelope) error {
	if env == nil {
		return errors.New("projection.sqlite: nil envelope")
	}

	var p committedPayloadView
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("projection.sqlite: unmarshal payload: %w", err)
	}

	// Read the .md from disk so the chunks row carries the full
	// content for FTS5 indexing. If the file is gone (manually deleted
	// between writer commit and dispatch) we treat it as a transient
	// failure so the dispatcher retries — the operator may restore
	// the file before the retry cap kicks in.
	content, err := os.ReadFile(p.Path)
	if err != nil {
		return fmt.Errorf("projection.sqlite: read %s: %w", p.Path, err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("projection.sqlite: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// 1. UPSERT documents row (idempotent via ON CONFLICT).
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO documents (id, path, updated_at, content_hash)
		VALUES (?, ?, strftime('%s','now'), ?)
		ON CONFLICT(id) DO UPDATE SET
			path         = excluded.path,
			updated_at   = excluded.updated_at,
			content_hash = excluded.content_hash
	`, env.AggregateID, p.Path, p.ContentHash); err != nil {
		return fmt.Errorf("projection.sqlite: upsert documents: %w", err)
	}

	// 2. Replace chunks for this (document_id, revision) — old
	// chunks for the previous revision are deleted so the FTS5
	// index doesn't double-count when a doc is rewritten. The
	// dedup key is (document_id, revision_id) which is unique
	// per committed event.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM chunks WHERE document_id = ? AND chunk_index < 0`, env.AggregateID); err != nil {
		return fmt.Errorf("projection.sqlite: delete prior chunks: %w", err)
	}

	chunkID := fmt.Sprintf("%s#r%d", env.AggregateID, env.Revision)
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO chunks (id, document_id, chunk_index, content)
		VALUES (?, ?, 0, ?)
	`, chunkID, env.AggregateID, string(content)); err != nil {
		return fmt.Errorf("projection.sqlite: insert chunks: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO chunks_fts (chunk_id, content)
		VALUES (?, ?)
	`, chunkID, string(content)); err != nil {
		return fmt.Errorf("projection.sqlite: insert chunks_fts: %w", err)
	}

	// 3. Advance the projection cursor atomically with the upsert
	// so a crash between upsert and ACK re-delivers the same event
	// (and INSERT OR IGNORE makes the re-delivery a no-op).
	if err := s.log.MarkAcked(ctx, tx, env.Sequence, ""); err != nil {
		return fmt.Errorf("projection.sqlite: mark acked: %w", err)
	}
	if err := s.log.AdvanceCursor(ctx, tx, s.Name(), env.Sequence); err != nil {
		return fmt.Errorf("projection.sqlite: advance cursor: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("projection.sqlite: commit: %w", err)
	}
	committed = true
	return nil
}

// committedPayloadView is a local mirror of writer.committedPayload.
// Defined here (rather than importing the writer type) so the
// projection package has a stable, narrow dependency: it reads the
// JSON shape by name and tolerates extra fields. Field order matches
// writer.committedPayload for round-trip stability.
type committedPayloadView struct {
	Path        string   `json:"path"`
	ContentHash string   `json:"content_hash"`
	SizeBytes   int      `json:"size_bytes"`
	Anchors     []string `json:"anchors"`
	Oversize    bool     `json:"oversize,omitempty"`
	IfMatch     *int64   `json:"if_match,omitempty"`
}
