// Package writer implements the atomic Markdown writer for the my-memory
// vault (ADR-044). It guarantees that every successful write commits the
// `.md` file to disk and the corresponding `memory.committed` event to the
// event log in the same SQLite WAL transaction, with crash-safety,
// idempotency, and revision precondition enforcement.
//
// The writer is the single ingress point for vault mutations from any
// caller (CLI, MCP tools, voice pipeline, agent runtime). It sits on top of
// the event_runtime envelope (ADR-043) and is hosted by `mymemoryd`
// (ADR-042).
//
// # Precondition: If-Match semantics
//
// Every write carries an optional ExpectedRevision pointer (nil when no
// precondition was requested, e.g. CLI without --if-match). When set,
// the writer enforces the revision via Check before touching disk;
// mismatch returns *PreconditionError AND emits a conflict.detected
// envelope for forensics.
//
// The HTTP-style convention "If-Match: revision=N" is recorded in the
// committedPayload so future HTTP transports (ADR-021 MCP HTTP/SSE)
// can reconstruct precondition semantics from the envelope alone.
// Subscribers that don't care can ignore the field; subscribers that
// implement their own retry-with-precondition loop use it to know
// which revision the caller expected.
package writer
