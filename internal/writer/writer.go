package writer

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/FelipeMiiller/my-memory/internal/event_runtime"
)

// MaxWriteBytes is the hard size limit for a single .md write (ADR-044
// §Edge Cases — markdown typical is < 100 KiB; this covers `![[file]]`
// embeds). Writers exceeding this limit are rejected up-front with
// ErrWriteFailed, before any disk or database work happens.
const MaxWriteBytes = 5 << 20 // 5 MiB

// WarnWriteBytes is the size above which the writer emits a warning
// event payload field. Warnings are informational — they do not fail
// the write — but they surface to the audit subscriber so operators can
// flag unusually large writes.
const WarnWriteBytes = 1 << 20 // 1 MiB

// Writer is the single ingress point for vault mutations (ADR-044). It
// guarantees that a successful Write commits the `.md` file to disk and
// the corresponding `memory.committed` envelope to the event_log in the
// same SQLite WAL transaction. On any failure between the .md write and
// the COMMIT, the .md file is removed as a compensating action so the
// filesystem and the database cannot diverge.
//
// The Writer does NOT hold the .md file open or stream its contents.
// Atomicity comes from the standard `tmp + rename` pattern documented
// in the spec; the rename is atomic on POSIX and on Windows when using
// os.Rename (Go runtime handles MoveFileEx with replace-existing flag).
type Writer struct {
	db  *sql.DB
	log *event_runtime.Log
}

// New constructs a Writer bound to the given database handle and event
// log. Both must be initialized (event_log and projection_cursor tables
// must exist — see internal/db.Schema and the event-runtime-event-log
// spec).
func New(db *sql.DB, log *event_runtime.Log) *Writer {
	return &Writer{db: db, log: log}
}

// WriteRequest captures everything the writer needs to perform a single
// vault mutation. The fields are explicit (rather than variadic) so the
// caller cannot accidentally omit a required value.
type WriteRequest struct {
	// DocumentID is the vault document id. When empty, the writer
	// derives it from Path via filepath-to-id hashing so the caller
	// can write a new document without first generating an id.
	DocumentID string

	// Path is the on-disk location of the .md file. Must be absolute
	// (relative paths would make the precondition check racy across
	// the cwd of concurrent processes).
	Path string

	// Content is the new markdown body. Replaces any existing content
	// at Path atomically (the writer does not merge with the existing
	// file — that is the caller's responsibility via a read-modify-write
	// loop guarded by ExpectedRevision).
	Content []byte

	// Actor identifies who is performing the write. ADR-050 §DR-3
	// mandates this for audit; common values are "user:<owner>",
	// "agent:<model>", or "system:<component>".
	Actor string

	// CorrelationID propagates the upstream turn or tool-call id into
	// the emitted envelope, so replay traces the full causal chain.
	CorrelationID string

	// CausationID points to the immediate parent event (optional).
	CausationID string

	// ExpectedRevision enforces the precondition check (ADR-044 §P3).
	// When nil, the write proceeds unconditionally — this preserves
	// backward compatibility with status-quo callers (mem note create
	// from CLI without --if-match).
	//
	// Pointer-to-int64 is used (rather than int64 + a sentinel value)
	// so "no precondition" is unambiguous in the type signature.
	ExpectedRevision *int64

	// Provenance is optional audit metadata that flows into the
	// emitted envelope. ADR-050 §LLM04 mandates this for any content
	// imported from outside the vault (quarantine decisions, trust
	// levels, etc.).
	Provenance *event_runtime.Provenance
}

// WriteResult is what a successful Write returns. Callers use this for
// the ACK back to the user/MCP tool.
type WriteResult struct {
	// EventID is the UUID v7 of the memory.committed envelope.
	EventID string
	// Sequence is the event_log.sequence value assigned to the commit.
	Sequence int64
	// Revision is the new documents.revision value (always > previous).
	Revision int64
	// ContentHash is the SHA-256 hex of the .md content (ADR-010).
	ContentHash string
	// Path is the filesystem path that was written to (echoed for
	// caller convenience; equal to req.Path).
	Path string
	// SizeBytes is len(req.Content), captured for the audit payload.
	SizeBytes int
	// OversizeWarning is true when Content > WarnWriteBytes. The write
	// itself succeeds; this flag exists so the audit subscriber can
	// surface a warning event without parsing the payload itself.
	OversizeWarning bool
}

// committedPayload is the JSON body of the memory.committed envelope.
// Stable field order is part of the wire contract — reordering fields
// would break subscribers that pattern-match on position.
type committedPayload struct {
	Path        string   `json:"path"`
	ContentHash string   `json:"content_hash"`
	SizeBytes   int      `json:"size_bytes"`
	Anchors     []string `json:"anchors"`
	Oversize    bool     `json:"oversize,omitempty"`
}

// Write performs the atomic commit described in ADR-044 §P1 AC 1. The
// call flow is:
//
//  1. validate inputs (fail fast before any disk or DB work)
//  2. compute content_hash
//  3. check precondition (if ExpectedRevision is set)
//  4. BEGIN IMMEDIATE
//  5. write .md atomically (tmp + os.Rename)
//  6. INSERT memory.committed envelope via event_runtime.Log.Append
//  7. UPSERT documents row (revision = expected+1, last_event_id, content_hash)
//  8. COMMIT
//
// On any failure between step 5 and step 8, the .md file is removed as
// a compensating action so the filesystem and the database cannot
// diverge. Step 3 (precondition failure) is reported via
// *PreconditionError and does NOT touch disk.
func (w *Writer) Write(ctx context.Context, req WriteRequest) (WriteResult, error) {
	// --- 1. validate ---
	if req.Path == "" {
		return WriteResult{}, fmt.Errorf("%w: path is empty", ErrWriteFailed)
	}
	if !filepath.IsAbs(req.Path) {
		return WriteResult{}, fmt.Errorf("%w: path must be absolute (got %q)", ErrWriteFailed, req.Path)
	}
	if len(req.Content) == 0 {
		return WriteResult{}, fmt.Errorf("%w: content is empty", ErrWriteFailed)
	}
	if len(req.Content) > MaxWriteBytes {
		return WriteResult{}, fmt.Errorf("%w: content size %d exceeds hard limit %d MiB",
			ErrWriteFailed, len(req.Content), MaxWriteBytes/(1<<20))
	}

	docID := req.DocumentID
	if docID == "" {
		docID = pathToDocID(req.Path)
	}

	// --- 2. compute content_hash ---
	sum := sha256.Sum256(req.Content)
	contentHash := hex.EncodeToString(sum[:])

	// --- 3. precondition check ---
	var expected int64
	if req.ExpectedRevision != nil {
		expected = *req.ExpectedRevision
		if err := Check(ctx, w.db, docID, expected); err != nil {
			return WriteResult{}, err
		}
	}

	// --- 4. BEGIN IMMEDIATE ---
	tx, err := w.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return WriteResult{}, fmt.Errorf("%w: begin tx: %v", ErrDatabaseUnavailable, err)
	}
	// rollback is a no-op after Commit; safe to defer unconditionally.
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// --- 5. write .md atomically ---
	if err := writeMarkdownAtomic(req.Path, req.Content); err != nil {
		return WriteResult{}, fmt.Errorf("%w: %v", ErrWriteFailed, err)
	}

	// If any subsequent step fails, remove the .md we just wrote so
	// filesystem and database cannot diverge. This is the compensating
	// action documented in ADR-044 §Decision Outcome.
	compensate := func() { _ = os.Remove(req.Path) }

	// --- 6. emit memory.committed envelope ---
	committedEnv := event_runtime.NewEnvelope("memory.committed", docID)
	committedEnv.Actor = req.Actor
	committedEnv.CorrelationID = req.CorrelationID
	committedEnv.CausationID = req.CausationID
	committedEnv.Revision = int(expected) + 1
	committedEnv.Payload, _ = json.Marshal(committedPayload{
		Path:        req.Path,
		ContentHash: contentHash,
		SizeBytes:   len(req.Content),
		Anchors:     extractWikilinks(req.Content),
		Oversize:    len(req.Content) > WarnWriteBytes,
	})
	committedEnv.Provenance = req.Provenance

	if err := w.log.Append(ctx, tx, committedEnv); err != nil {
		compensate()
		// ErrDuplicateEventID is a near-impossible race (UUID v7
		// collision); surface as ErrDatabaseUnavailable so callers
		// retry rather than treating it as success.
		if errors.Is(err, event_runtime.ErrDuplicateEventID) {
			return WriteResult{}, fmt.Errorf("%w: %v", ErrDatabaseUnavailable, err)
		}
		return WriteResult{}, fmt.Errorf("%w: append event: %v", ErrDatabaseUnavailable, err)
	}

	// --- 7. UPSERT documents row ---
	newRev := int64(committedEnv.Revision)
	now := time.Now().Unix()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO documents (id, path, revision, last_event_id, updated_at, content_hash)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			path         = excluded.path,
			revision     = excluded.revision,
			last_event_id= excluded.last_event_id,
			updated_at   = excluded.updated_at,
			content_hash = excluded.content_hash
	`, docID, req.Path, newRev, committedEnv.EventID, now, contentHash)
	if err != nil {
		compensate()
		return WriteResult{}, fmt.Errorf("%w: upsert document: %v", ErrDatabaseUnavailable, err)
	}

	// --- 8. COMMIT ---
	if err := tx.Commit(); err != nil {
		compensate()
		return WriteResult{}, fmt.Errorf("%w: commit: %v", ErrDatabaseUnavailable, err)
	}
	committed = true

	return WriteResult{
		EventID:         committedEnv.EventID,
		Sequence:        committedEnv.Sequence,
		Revision:        newRev,
		ContentHash:     contentHash,
		Path:            req.Path,
		SizeBytes:       len(req.Content),
		OversizeWarning: len(req.Content) > WarnWriteBytes,
	}, nil
}

// writeMarkdownAtomic writes content to path via the standard tmp+rename
// pattern. The tmp file lives in the same directory as path so the
// rename stays on the same filesystem (cross-filesystem rename is not
// atomic). On Windows, os.Rename uses MoveFileEx with replace-existing
// semantics — equivalent to the POSIX behavior for our purposes.
//
// The function does NOT create parent directories — that is the
// caller's responsibility (vaults that need mkdir should do it before
// calling Write, so a missing directory produces a clear ErrWriteFailed
// rather than a half-created tree).
func writeMarkdownAtomic(path string, content []byte) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".write-*.md.tmp")
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	tmpName := tmp.Name()

	// Ensure cleanup on any failure path.
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = tmp.Write(content); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("sync tmp: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close tmp: %w", err)
	}
	if err = os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmpName, path, err)
	}
	return nil
}

// pathToDocID derives a stable document id from an absolute filesystem
// path. It is used when WriteRequest.DocumentID is empty so the caller
// can create a new document without first generating an id externally.
// The current implementation lowercases the path and strips the
// filesystem-specific prefix (e.g. /tmp/foo.md -> tmp/foo.md on Linux);
// this is a deliberate simplification for T3 and will be revisited in
// the policy-engine task (T9) when path scoping lands.
func pathToDocID(path string) string {
	// Normalize: lowercase, strip leading slashes, replace separators
	// with single hyphen. Result is opaque to callers — they should
	// pass DocumentID explicitly when they have one.
	p := filepath.ToSlash(path)
	p = strings.TrimPrefix(p, "/")
	p = strings.ToLower(p)
	p = strings.ReplaceAll(p, "/", "-")
	return "doc-" + p
}

// extractWikilinks returns the set of [[wikilink]] targets found in
// content. The implementation is intentionally lightweight (single-pass
// scan for "[[" + "]]" pairs) — full parser integration lives in
// internal/parser and is out of scope for T3. The slice may contain
// duplicates; downstream consumers dedupe as needed.
func extractWikilinks(content []byte) []string {
	const open, close = "[[", "]]"
	var out []string
	s := string(content)
	for {
		i := strings.Index(s, open)
		if i < 0 {
			break
		}
		s = s[i+len(open):]
		j := strings.Index(s, close)
		if j < 0 {
			break
		}
		target := strings.TrimSpace(s[:j])
		if target != "" {
			out = append(out, target)
		}
		s = s[j+len(close):]
	}
	return out
}
