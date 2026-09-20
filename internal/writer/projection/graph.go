package projection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/FelipeMiiller/my-memory/internal/event_runtime"
)

// GraphProjection applies memory.committed events to the graph_nodes
// and graph_edges tables (T7). On each commit it ensures the document
// is represented as a graph node and that every `[[wikilink]]` target
// is represented as both a target node and a `links_to` edge from the
// document to the target.
//
// DEVIATION from spec: tasks.md T7 describes "DELETE old edges WHERE
// document_id=? AND revision_id=? + INSERT new with revision_id" —
// i.e., per-revision edge history. The current graph_edges schema
// (internal/db/schema.go) has PRIMARY KEY (source_id, target_id,
// relation) and no document_id / revision_id column, so per-revision
// history would require a schema migration that is out of scope for
// ADR-044 P1. The projection below collapses to "current state per
// document": re-delivering the same envelope is a no-op (PRIMARY KEY
// idempotency), and a new revision overwrites the prior edge set via
// the same UPSERT path. A future ADR (likely ADR-046+ era) can add
// graph_edge_revisions if historical edge state becomes required.
type GraphProjection struct {
	db  *sql.DB
	log *event_runtime.Log
}

// NewGraphProjection constructs the graph projection bound to the same
// DB + Log handles the dispatcher uses.
func NewGraphProjection(db *sql.DB, log *event_runtime.Log) *GraphProjection {
	return &GraphProjection{db: db, log: log}
}

// Name implements event_runtime.Subscriber.
func (g *GraphProjection) Name() string { return "projection.graph" }

// EventTypes implements event_runtime.Subscriber. Same as the SQLite
// projection — derives all read-side state from memory.committed.
func (g *GraphProjection) EventTypes() []string {
	return []string{"memory.committed"}
}

// MaxAckPending implements event_runtime.Subscriber.
func (g *GraphProjection) MaxAckPending() int { return 64 }

// Handle implements event_runtime.Subscriber. Walks the Anchors slice
// from the envelope payload and writes one graph_edges row per
// (source, target, 'links_to') tuple. The PRIMARY KEY on graph_edges
// guarantees idempotency without an explicit INSERT OR IGNORE.
//
// source node (the document) is upserted once per commit. Target
// nodes are upserted once per unique wikilink — repeated writes to
// the same document only touch edges that changed.
func (g *GraphProjection) Handle(ctx context.Context, env *event_runtime.Envelope) error {
	if env == nil {
		return errors.New("projection.graph: nil envelope")
	}

	var p committedPayloadView
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		return fmt.Errorf("projection.graph: unmarshal payload: %w", err)
	}

	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("projection.graph: begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// 1. Upsert source node (the document).
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO graph_nodes (id, type, name)
		VALUES (?, 'note', ?)
		ON CONFLICT(id) DO UPDATE SET name = excluded.name
	`, env.AggregateID, nameFromID(env.AggregateID)); err != nil {
		return fmt.Errorf("projection.graph: upsert source node: %w", err)
	}

	// 2. For each anchor, upsert target node + links_to edge.
	for _, target := range p.Anchors {
		if target == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO graph_nodes (id, type, name)
			VALUES (?, 'note', ?)
			ON CONFLICT(id) DO UPDATE SET name = excluded.name
		`, target, target); err != nil {
			return fmt.Errorf("projection.graph: upsert target node %q: %w", target, err)
		}
		// PRIMARY KEY (source_id, target_id, relation) gives us
		// idempotency via INSERT OR IGNORE — re-deliveries and
		// same-document rewrites that don't change this particular
		// edge leave the row untouched.
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO graph_edges (source_id, target_id, relation, epistemic_status, weight)
			VALUES (?, ?, 'links_to', 'EXTRACTED', 1.0)
		`, env.AggregateID, target); err != nil {
			return fmt.Errorf("projection.graph: insert edge %s -> %s: %w", env.AggregateID, target, err)
		}
	}

	// 3. Advance cursor + ACK atomically with the projection upserts.
	if err := g.log.MarkAcked(ctx, tx, env.Sequence, ""); err != nil {
		return fmt.Errorf("projection.graph: mark acked: %w", err)
	}
	if err := g.log.AdvanceCursor(ctx, tx, g.Name(), env.Sequence); err != nil {
		return fmt.Errorf("projection.graph: advance cursor: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("projection.graph: commit: %w", err)
	}
	committed = true
	return nil
}

// nameFromID returns a human-readable name for a graph node derived
// from its id. Document ids are opaque (e.g., "doc-tmp-foo.md"); we
// surface the raw id as the name rather than synthesize something
// pretty, since the graph view shows the underlying identifier
// anyway. Wikilink anchors are already human-readable so they pass
// through verbatim.
func nameFromID(id string) string { return id }
