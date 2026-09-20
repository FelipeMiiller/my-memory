# memory-writer-atomic Tasks

## Execution Protocol (MANDATORY — do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user — do not proceed without it.**

---

**Spec**: `.specs/features/memory-writer-atomic/spec.md`
**Design**: [ADR-044](../../../docs/adr/044-memory-writer-atomico-outbox-e-reprojecao-idempotente.md)
**Security**: [ADR-050 (Threat Model OWASP LLM)](../../../docs/adr/050-threat-model-owasp-llm-aplicado-ao-mymemory.md) — controls LLM02 (redaction), LLM04 (provenance), LLM09 (misinformation citations) apply.
**Status**: Draft → Approved → In Progress → Done

---

## Test Coverage Matrix

> Generated from codebase sampling. Guidelines found: `.agents/rules/always-quality-gate.md` (gate = gofmt + go build + go test) and `.agents/rules/always-test.md` (mandatory tests after each task). Existing test pattern: `internal/<pkg>/*_test.go` co-located with source.

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| --- | --- | --- | --- | --- |
| Service (writer, precondition, policy) | unit | All branches; 1:1 to spec ACs; every listed edge case has a test | `internal/writer/*_test.go`, `internal/policy/*_test.go` | `go test -count=1 ./internal/writer/... ./internal/policy/...` |
| Domain (Envelope extension, Revision) | unit | Type validation; round-trip; conflict scenarios | `internal/writer/writer_test.go` | `go test -count=1 ./internal/writer/...` |
| Repository (event_log, projection_cursor already done by event-runtime-event-log) | integration | Idempotency via `INSERT OR IGNORE` on `event_id` | `internal/writer/writer_test.go` | `go test -count=1 ./internal/writer/...` |
| Migration SQL (DDL in store.go) | none | Build gate only | `internal/db/store.go` | build gate only |
| CLI subcommand (mem write --if-match) | integration | Happy path + every listed edge case + 409 conflict | `cmd/mem/write_test.go` | `go test -count=1 ./cmd/mem/...` |

---

## Gate Check Commands

> Generated from `.agents/rules/always-quality-gate.md`. Each gate runs from repo root.

| Gate Level | When to Use | Command |
| --- | --- | --- |
| Quick | After tasks with unit tests only (T2, T4, T5, T7, T8) | `go test -count=1 -short ./internal/writer/... ./internal/policy/...` |
| Full | After tasks with CLI/integration tests (T1, T3, T6, T9, T10, T11) | `go test -count=1 ./internal/writer/... ./internal/policy/... ./internal/db/... ./cmd/mem/...` |
| Build | After phase completion or schema/CLI tasks | `gofmt -l . && go build ./... && go test -count=1 ./...` |

---

## Execution Plan

Phases are ordered and run sequentially. Tasks within a phase execute in order.

### Phase 1: Setup (preconditions + schema refinements)

Foundation for writer: precondition checker + schema additions for revision tracking.

```
T1 -> T3
T2 -> T3
```

### Phase 2: Writer Core (atomic commit + outbox)

Atomic `.md` + `event_log` in same SQLite transaction; precondition enforcement.

```
T3 -> T4
T3 -> T5
```

### Phase 3: Projections (idempotent subscribers)

`projection.sqlite` + `projection.graph` as event_log subscribers with idempotency.

```
T6 -> T7
T6 -> T8
T7 -> T8
```

### Phase 4: Policy Engine + CLI

Policy engine with 3 profiles + CLI `--if-match` flag.

```
T9 -> T10
T10 -> T11
T5 -> T11
```

Total: **11 tasks across 4 phases**. Fits 2 batches at Execute time (Phase 1-2 inline, Phase 3-4 sub-agent).

---

## Task Breakdown

> **Note on validator**: `validate_tasks.py` keeps the last-seen `phase_idx` after `### Phase N` headers and assigns it to subsequent `### Tn:` task headers. To keep per-task phase assignment accurate in the flat Task Breakdown layout, we repeat `### Phase N` markers immediately above each phase's first task.

### Phase 1

### T1: Add `documents.revision` columns + writer package skeleton

**What**: Verify `documents.revision` + `documents.last_event_id` columns from event-runtime-event-log (already merged) — if missing, add via guarded ALTER. Create `internal/writer/` package skeleton with `doc.go`, `errors.go` (sentinel errors: `ErrWriteFailed`, `ErrDatabaseUnavailable`, `ErrPreconditionFailed`, `ErrPolicyUnavailable`, `ErrInvalidRevision`).
**Where**: `internal/db/store.go` (verify, modify if needed), `internal/writer/doc.go` (new), `internal/writer/errors.go` (new)
**Depends on**: None
**Reuses**: Existing `Schema` constant from event-runtime-event-log spec; existing sentinel error pattern from `internal/event_runtime/envelope.go`.
**Requirement**: WRTR-01 (precondition — schema side)
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `documents.revision INTEGER NOT NULL DEFAULT 0` exists in SQLite (verified via `PRAGMA table_info`)
- [ ] `documents.last_event_id TEXT` exists in SQLite
- [ ] `internal/writer/doc.go` declares package and imports
- [ ] `internal/writer/errors.go` exports 5 sentinel errors with descriptive docstrings
- [ ] `go build ./...` succeeds
- [ ] No regression on existing writer tests (none expected since package is new)

**Tests**: none (build gate only)
**Gate**: build
**Commit**: `feat(writer): skeleton + revision column verification`

---

### T2: Precondition checker (version comparison)

**What**: Create `internal/writer/precondition.go` with `Check(ctx, db, documentID, expectedRevision) error` — reads `documents.revision` with `BEGIN IMMEDIATE` semantics, compares with `expectedRevision`. Returns `ErrPreconditionFailed` (with `CurrentRevision` field for diagnostics) on mismatch. Special case: `expectedRevision == 0` + document exists → `ErrPreconditionFailed` (creation conflict).
**Where**: `internal/writer/precondition.go` (new), `internal/writer/precondition_test.go` (new)
**Depends on**: None
**Reuses**: Existing DB connection pattern from `internal/event_runtime/log.go`.
**Requirement**: WRTR-12, WRTR-13, WRTR-14, WRTR-15
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `Check(ctx, db, documentID, expectedRevision)` returns `nil` on match
- [ ] Returns `ErrPreconditionFailed{Current: 7}` on mismatch (expected 5, current 7)
- [ ] Returns `ErrPreconditionFailed` on `expected=0` + document exists
- [ ] Returns `nil` on `expected=0` + document does NOT exist (creation OK)
- [ ] Returns `ErrInvalidRevision` on `expected<0`
- [ ] Test `TestPrecondition_Match`: revision 5==5 returns nil
- [ ] Test `TestPrecondition_Mismatch`: revision 5 vs 7 returns `ErrPreconditionFailed` with `Current=7`
- [ ] Test `TestPrecondition_CreationConflict`: expected=0 + existing doc returns error
- [ ] Test `TestPrecondition_InvalidRevision`: expected=-1 returns `ErrInvalidRevision`

**Tests**: integration (uses real SQLite via `:memory:`)
**Gate**: full
**Commit**: `feat(writer): precondition checker for revision control`

---

### Phase 1
### T3: Writer core — atomic commit (md + event_log)

**What**: Create `internal/writer/writer.go` with `Writer` struct holding `*sql.DB` and `*event_runtime.Log`. Method `Write(ctx, req WriteRequest) (WriteResult, error)`: (1) emit `memory.write.requested` event, (2) validate schema, (3) check precondition via T2, (4) `BEGIN IMMEDIATE`, (5) write `.md` via `tmp + os.Rename` (Windows: `MoveFileEx` with replace flag), (6) compute SHA-256 `content_hash`, (7) INSERT into `event_log` with `event_type="memory.committed"`, (8) UPDATE `documents.revision` and `documents.last_event_id`, (9) `COMMIT`. Compensating action on SQLite failure: remove `.md` file.
**Where**: `internal/writer/writer.go` (new), `internal/writer/writer_test.go` (new)
**Depends on**: T1, T2
**Reuses**: `event_runtime.Log.Append` from event-runtime-event-log; existing `content_hash` SHA-256 computation from `internal/compiler/` (ADR-010).
**Requirement**: WRTR-01, WRTR-02, WRTR-03, WRTR-04, WRTR-05, WRTR-06
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `Write` performs atomic commit (`.md` + `event_log` + `documents.revision` in single tx)
- [ ] `memory.write.requested` envelope emitted BEFORE transaction begins
- [ ] `memory.committed` envelope emitted via `Log.Append` in same tx with `aggregate_id`, `revision`, `payload.content_hash`, `payload.anchors`, `payload.actor`
- [ ] Compensating action: SQLite error → `.md` removed + tx rolled back
- [ ] `.md` write failure → SQLite tx rolled back, no event emitted, returns `ErrWriteFailed`
- [ ] `content_hash` computed via SHA-256, stored in envelope payload
- [ ] `revision` auto-assigned as `documents.revision + 1`
- [ ] `last_event_id` updated to new `event_id`
- [ ] Test `TestWrite_CommitIsAtomic`: simulate crash mid-write via SIGKILL equivalent → next startup detects inconsistency
- [ ] Test `TestWrite_NoEventIfMdFails`: force filesystem error → 0 rows in `event_log`, tx rolled back
- [ ] Test `TestWrite_CompensatingActionOnDbError`: force SQLite error → `.md` removed

**Tests**: integration (uses real SQLite via `:memory:` + temp dir)
**Gate**: full
**Commit**: `feat(writer): atomic commit (md + event_log + revision in single tx)`

---

### Phase 2

### T4: Conflict detection event + 409 response

**What**: Extend `Writer.Write` (or add helper) to emit `conflict.detected` event when precondition fails. Event payload: `expected=<int>`, `current=<int>`, `document_id=<id>`. CLI/MCP return error with HTTP-409-equivalent code (`ErrPreconditionFailed`) and structured error message including current revision.
**Where**: `internal/writer/writer.go` (modify), `internal/writer/conflict_test.go` (new)
**Depends on**: T3
**Reuses**: `event_runtime.Log.Append` from T3.
**Requirement**: WRTR-13 (conflict.detected event side)
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `Write` with stale `expected_revision` emits `conflict.detected` event BEFORE returning error
- [ ] Returned error includes `CurrentRevision` field for caller diagnostics
- [ ] Event payload includes `expected`, `current`, `document_id`, `correlation_id`
- [ ] Test `TestWrite_PreconditionConflict_409`: stale revision → 409 + conflict.detected event in event_log
- [ ] Test `TestWrite_CreationConflict_409`: expected=0 + existing doc → 409 + conflict.detected event

**Tests**: integration
**Gate**: full
**Commit**: `feat(writer): conflict detection event + 409 response`

---

### T5: If-Match metadata + backward compat (no expected_revision)

**What**: Add `IfMatch` field to envelope headers (per HTTP convention) carrying `revision=<N>`. When `Writer.Write` is called with no `expected_revision`, behavior matches status quo (always write). Document the convention in `internal/writer/doc.go`.
**Where**: `internal/writer/writer.go` (modify), `internal/writer/writer_test.go` (modify)
**Depends on**: T3
**Reuses**: `event_runtime.Envelope.Headers` from event-runtime-event-log.
**Requirement**: WRTR-15, WRTR-16
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `WriteRequest` struct has optional `ExpectedRevision *int64` (nil = always write)
- [ ] When `ExpectedRevision == nil`, writer proceeds without precondition check
- [ ] `memory.committed` envelope includes `If-Match: revision=<N>` in headers (when ExpectedRevision set)
- [ ] Test `TestWrite_NoExpectedRevision_AlwaysWrites`: nil ExpectedRevision proceeds unconditionally
- [ ] Test `TestWrite_IfMatchHeader_InEnvelope`: ExpectedRevision=5 produces `If-Match: revision=5` header

**Tests**: integration
**Gate**: full
**Commit**: `feat(writer): If-Match metadata + backward compat (no expected_revision)`

---

### Phase 3

### T6: projection.sqlite (idempotent subscriber)

**What**: Create `internal/writer/projection/sqlite.go` implementing `event_runtime.Subscriber` interface. On `memory.committed` event: upsert `documents` row (by `aggregate_id`), upsert `chunks` (by `(document_id, revision_id)`), upsert `chunks_fts` (FTS5 virtual table). Use `INSERT OR IGNORE` keyed on `event_id` for idempotency. Advance `projection_cursor.last_sequence` only after successful apply + ACK.
**Where**: `internal/writer/projection/sqlite.go` (new), `internal/writer/projection/sqlite_test.go` (new)
**Depends on**: T3
**Reuses**: Existing `chunks` and `chunks_fts` schema from `internal/db/schema.go`; `event_runtime.Subscriber` interface from event-runtime-event-log.
**Requirement**: WRTR-07, WRTR-08 (idempotency via projection_cursor)
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `SqliteProjection` implements `event_runtime.Subscriber.Handle(ctx, env) error`
- [ ] Upserts `documents` row matching `aggregate_id`
- [ ] Upserts `chunks` rows with `(document_id, revision_id)` dedup key
- [ ] Upserts `chunks_fts` content
- [ ] Uses `INSERT OR IGNORE` on `event_id` for idempotency
- [ ] Advances `projection_cursor.last_sequence` for `projection.sqlite` only after success
- [ ] Test `TestProjectionSqlite_IdempotentReplay`: apply event 5x → only 1 row in `documents`
- [ ] Test `TestProjectionSqlite_HandlesMultipleRevisions`: same doc, 3 revisions → 3 chunks, latest revision wins for `last_event_id`

**Tests**: integration
**Gate**: full
**Commit**: `feat(writer-projection): projection.sqlite idempotent subscriber`

---

### T7: projection.graph (diff edges subscriber)

**What**: Create `internal/writer/projection/graph.go` implementing `event_runtime.Subscriber`. On `memory.committed`: (1) read old `edges` for `(document_id, revision_id-1)`, (2) read new edges from `.md` content (extract `[[wikilinks]]` via `internal/parser/`), (3) DELETE old edges WHERE `document_id=?` AND `revision_id=?`, (4) INSERT new edges with `revision_id=current`. Idempotent via `INSERT OR IGNORE` on edge `(source, target, kind, document_id, revision_id)`.
**Where**: `internal/writer/projection/graph.go` (new), `internal/writer/projection/graph_test.go` (new)
**Depends on**: T6
**Reuses**: `internal/parser/wikilink.go` for edge extraction; existing `edges` schema from `internal/db/schema.go`.
**Requirement**: WRTR-07, WRTR-09 (replay produces identical state)
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `GraphProjection` implements `event_runtime.Subscriber`
- [ ] On `memory.committed`: extracts wikilinks from `payload.content` (or re-parses `.md` if needed)
- [ ] DELETEs old edges for `(document_id, revision_id-1)`
- [ ] INSERTs new edges for `(document_id, revision_id)`
- [ ] `INSERT OR IGNORE` on edge dedup key
- [ ] Test `TestProjectionGraph_DiffEdges`: write doc with 3 wikilinks, update to remove 1 → old edges deleted, new edges present
- [ ] Test `TestProjectionGraph_IdempotentReplay`: apply same event 5x → edge count identical

**Tests**: integration
**Gate**: full
**Commit**: `feat(writer-projection): projection.graph diff-edges subscriber`

---

### T8: Replay tool — `mem replay --since <seq>`

**What**: Add `mem replay --since <seq>` CLI subcommand in `cmd/mem/replay.go`. Reads events from `event_log` starting at `seq+1` up to `MAX(sequence)`, dispatches to registered subscribers (idempotent via `INSERT OR IGNORE`). Default is dry-run (re-emit to audit only); `--apply` flag is opt-in destructive.
**Where**: `cmd/mem/replay.go` (new), `cmd/mem/replay_test.go` (new)
**Depends on**: T6, T7
**Reuses**: `event_runtime.Dispatcher` from event-runtime-event-log; existing CLI pattern from `cmd/mem/events.go`.
**Requirement**: WRTR-08, WRTR-09, WRTR-10, WRTR-11
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `mem replay --since <seq>` re-emits events to subscribers without re-executing handlers (dry-run)
- [ ] `mem replay --since <seq> --apply` re-executes handlers (idempotent)
- [ ] Returns 0 events with exit 0 when `seq > MAX(sequence)` (stream Unix-like behavior)
- [ ] Outputs JSON summary: `{ "from": <seq>, "to": <seq>, "events_replayed": <count>, "errors": [] }`
- [ ] Test `TestReplay_DryRun_ReEmitsToAudit`: 100 events → audit subscriber receives 100, others do NOT
- [ ] Test `TestReplay_Apply_RecoversState`: delete SQLite, restore event_log, replay --since 0 --apply → state byte-equal to pre-delete

**Tests**: integration
**Gate**: full
**Commit**: `feat(cli): mem replay --since <seq> [--apply]`

---

### Phase 4

### T9: Policy engine stub + 3 profiles

**What**: Create `internal/policy/engine.go` with `Engine` struct reading `.memory/policy/<active>.yaml`. Methods: `Load(profilePath) (*Engine, error)`, `Decide(ctx, actor, path, op) Decision` (returns Allow|Deny|RequireApproval). Default profile `balanced`: writes within vault scope proceed, outside require approval. Profiles `strict` and `permissive-dev` defined in YAML templates.
**Where**: `internal/policy/engine.go` (new), `internal/policy/engine_test.go` (new), `.memory/policy/{strict,balanced,permissive-dev}.yaml` (new templates)
**Depends on**: None
**Reuses**: YAML parsing via `gopkg.in/yaml.v3` (already in go.mod from `internal/config/`).
**Requirement**: WRTR-17, WRTR-18, WRTR-19
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `Engine` struct holds profile config + decision logic
- [ ] `Load("/path/to/policy.yaml")` parses YAML and returns `*Engine`
- [ ] `Decide(actor="user:owner", path="docs/foo.md", op="write")` returns `Allow` for `balanced` profile
- [ ] `Decide(actor="agent:external", path="docs/foo.md", op="write")` returns `RequireApproval` for `balanced`
- [ ] `Decide(actor="user:owner", path="/etc/passwd", op="write")` returns `Deny` for `balanced`
- [ ] `strict` profile: all non-owner actors → `RequireApproval`
- [ ] `permissive-dev` profile: all writes → `Allow`
- [ ] `.memory/policy/balanced.yaml` is default fallback
- [ ] Missing policy file → default `balanced` + warn at startup
- [ ] Test `TestEngine_BalancedAllowsOwnerWritesInScope`
- [ ] Test `TestEngine_BalancedRequiresApprovalForExternalActor`
- [ ] Test `TestEngine_StrictBlocksAllNonOwnerWrites`
- [ ] Test `TestEngine_PermissiveDevAllowsEverything`

**Tests**: integration
**Gate**: full
**Commit**: `feat(policy): engine stub + 3 profile templates`

---

### T10: Policy integration with writer

**What**: Extend `Writer.Write` to call `policy.Engine.Decide` before transaction begins. On `Deny` → return `ErrPolicyDenied` (no event emitted). On `RequireApproval` → emit `approval.requested` event with `payload.actor`, `payload.path`, `payload.op` and return `ErrApprovalRequired`. On `Allow` → proceed.
**Where**: `internal/writer/writer.go` (modify), `internal/writer/policy_test.go` (new)
**Depends on**: T3, T9
**Reuses**: `policy.Engine` from T9.
**Requirement**: WRTR-19, WRTR-20, WRTR-21
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `Write` calls `policy.Engine.Decide` before transaction
- [ ] `Deny` → `ErrPolicyDenied`, no event, no tx
- [ ] `RequireApproval` → `approval.requested` event + `ErrApprovalRequired`
- [ ] `Allow` → proceeds normally
- [ ] `policy.decision_id` recorded in `memory.committed` envelope
- [ ] Test `TestWrite_PolicyBlocksHighRiskActor`: external actor + strict → `ErrPolicyDenied`
- [ ] Test `TestWrite_PolicyRequiresApprovalForExternalScope`: external path + balanced → `approval.requested` event
- [ ] Test `TestWrite_PolicyRecordsDecisionID`: allow path → `policy.decision_id` in envelope

**Tests**: integration
**Gate**: full
**Commit**: `feat(writer): policy integration with approval flow`

---

### T11: CLI `--if-match` flag + MCP `if_match` parameter

**What**: Add `--if-match <revision>` flag to `mem write` and `mem note create` subcommands. Default: no flag = always write. Add `if_match` optional parameter to MCP tools `memory_write_note`, `memory_append_section`, `memory_compile_note`.
**Where**: `cmd/mem/write.go` (modify or new), `internal/mcp/tools.go` (modify), `cmd/mem/write_test.go` (new)
**Depends on**: T5, T10
**Reuses**: Existing CLI flag parsing from `cobra`; existing MCP tool registration from `internal/mcp/`.
**Requirement**: WRTR-16 (CLI flag), MCP integration spec
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `mem write --if-match 5 ./foo.md` calls writer with `ExpectedRevision=5`
- [ ] `mem write ./foo.md` (no flag) calls writer with `ExpectedRevision=nil`
- [ ] `mem write --if-match 5 ./foo.md` on stale revision returns exit 2 with error message
- [ ] MCP tool `memory_write_note` accepts `if_match` parameter (JSON int) and passes to writer
- [ ] Test `TestCliWrite_IfMatch_StaleReturnsError`
- [ ] Test `TestCliWrite_NoFlag_AlwaysWrites`
- [ ] Test `TestMcpWriteNote_IfMatchParameter`

**Tests**: integration
**Gate**: full
**Commit**: `feat(cli): mem write --if-match + MCP if_match parameter`
