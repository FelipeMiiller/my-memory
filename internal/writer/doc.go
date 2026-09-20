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
package writer
