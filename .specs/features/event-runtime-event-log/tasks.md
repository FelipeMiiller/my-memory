# event-runtime-event-log Tasks

## Execution Protocol (MANDATORY — do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user — do not proceed without it.**

---

**Spec**: `.specs/features/event-runtime-event-log/spec.md`
**Design**: [ADR-043](../../../docs/adr/043-envelope-de-eventos-canonico-event-runtime.md)
**Security**: [ADR-050 (Threat Model OWASP LLM)](../../../docs/adr/050-threat-model-owasp-llm-aplicado-ao-mymemory.md) — controls LLM02 (redaction) and LLM07 (system prompt leakage) apply.
**Status**: Draft → Approved → In Progress → Done

---

## Test Coverage Matrix

> Generated from codebase sampling. Guidelines found: `.agents/rules/always-quality-gate.md` (gate = gofmt + go build + go test) and `.agents/rules/always-test.md` (mandatory tests after each task). Existing test pattern: `internal/<pkg>/*_test.go` co-located with source (e.g. `internal/compiler/note_test.go`, `internal/db/store_test.go`).

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| --- | --- | --- | --- | --- |
| Service (Go package logic — dispatcher, outbox) | unit | All branches; 1:1 to spec ACs; every listed edge case has a test | `internal/event_runtime/<file>_test.go` | `go test -count=1 ./internal/event_runtime/...` |
| Domain (struct, interface, validation) | unit | Type validation; marshal/unmarshal round-trip; error cases | `internal/event_runtime/envelope_test.go`, `redaction_test.go` | `go test -count=1 ./internal/event_runtime/...` |
| Repository (DB layer — event_log, projection_cursor) | integration | Key query paths + error handling + idempotência via `UNIQUE` | `internal/event_runtime/log_test.go`, `outbox_test.go` | `go test -count=1 ./internal/event_runtime/...` |
| Migration SQL (DDL in schema.go) | none | Build gate only — verified by `gofmt -l . && go build ./...` (compile-time guarantee that SQL parses) | `internal/db/schema.go`, `internal/db/store.go` | build gate only |
| CLI subcommand (mem events *) | integration | Happy path + every listed edge case + error paths (8 ACs in P4) | `cmd/mem/events_test.go` | `go test -count=1 ./cmd/mem/...` |

---

## Gate Check Commands

> Generated from `.agents/rules/always-quality-gate.md`. Each gate runs from repo root.

| Gate Level | When to Use | Command |
| --- | --- | --- |
| Quick | After tasks with unit tests only (T2, T4, T5, T6, T8, T9) | `go test -count=1 -short ./internal/event_runtime/...` |
| Full | After tasks with CLI/integration tests (T1, T3, T7, T10, T11, T12) | `go test -count=1 ./internal/event_runtime/... ./internal/db/... ./cmd/mem/...` |
| Build | After phase completion or schema/CLI tasks | `gofmt -l . && go build ./... && go test -count=1 ./...` |

---

## Execution Plan

Phases are ordered and run sequentially. Tasks within a phase execute in order. The diagrams below show intra-phase dependencies explicitly; cross-phase dependencies (forward or backward) are inferred from `Depends on` in each task body.

### Phase 1: Foundation (SQLite + Envelope + Log)

Foundational types and persistence layer. No business logic yet.

```
T1 -> T3
T2 -> T3
```

### Phase 2: Producer + Outbox (transactional atomicity)

Outbox helper that writes events in the caller's transaction.

```
T4 -> T5
```

### Phase 3: Dispatcher + Subscriber + Audit (event delivery)

Fan-out delivery with ACK/retry/backpressure + the audit subscriber (LLM02 compliance).

```
T6 -> T7
T6 -> T9
T7 -> T9
T8 -> T9
```

### Phase 4: CLI Surface (operator visibility)

Inspector + replay tool + doctor integration.

```
T10 -> T11
T10 -> T12
```

Total: **12 tasks across 4 phases**. Fits a single batch (~7 tasks/budget) when accounting for phase cohesion, but spans 12 tasks total — offered as 2 batches at Execute time (Phase 1-2 inline, Phase 3-4 sub-agent or split).

---

## Task Breakdown

> **Note on validator**: `validate_tasks.py` keeps the last-seen `phase_idx` after `### Phase N` headers and assigns it to subsequent `### Tn:` task headers. To keep per-task phase assignment accurate in the flat Task Breakdown layout, we repeat `### Phase N` markers immediately above each phase's first task. Canonical phase ordering is documented in `## Execution Plan` above.

### Phase 1

### T1: Add event_log + projection_cursor tables + documents columns

**What**: Add `event_log`, `projection_cursor` tables to both `Schema` and `FallbackSchema` in `internal/db/schema.go`; add `ALTER TABLE documents ADD COLUMN revision INTEGER NOT NULL DEFAULT 0` and `ADD COLUMN last_event_id TEXT` to `internal/db/store.go`. All DDL uses `IF NOT EXISTS` / guarded ALTER (PG-compatible pattern from existing `content_hash` ALTER at `store.go:40`).
**Where**: `internal/db/schema.go` (modify), `internal/db/store.go` (modify)
**Depends on**: None
**Reuses**: Existing `Schema` / `FallbackSchema` const pattern; existing ALTER guard pattern from `content_hash` migration.
**Requirement**: EVT-01 (partial — schema only)
**Status**: Verified (2026-09-19)

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `event_log` table defined in both `Schema` and `FallbackSchema` with all columns from ADR-043 §event_log schema: `sequence INTEGER PRIMARY KEY AUTOINCREMENT`, `event_id TEXT UNIQUE NOT NULL`, `schema_version INTEGER NOT NULL DEFAULT 1`, `event_type TEXT NOT NULL`, `aggregate_id TEXT NOT NULL`, `revision INTEGER`, `payload BLOB NOT NULL`, `headers BLOB NOT NULL`, `created_at TEXT NOT NULL`, `acked_at TEXT`
- [ ] `projection_cursor` table defined in both schemas: `projection_name TEXT PRIMARY KEY`, `last_sequence INTEGER NOT NULL`, `updated_at TEXT NOT NULL`
- [ ] `documents.revision INTEGER NOT NULL DEFAULT 0` added via guarded ALTER in `internal/db/store.go`
- [ ] `documents.last_event_id TEXT` added via guarded ALTER in `internal/db/store.go`
- [ ] `go build ./...` succeeds (compile-time SQL parse guarantee)
- [ ] Existing `TestStore_*` tests still pass (no regression on `documents` schema)

**Tests**: none (migration layer — build gate only per Coverage Matrix)
**Gate**: build
**Commit**: `feat(db): add event_log + projection_cursor + documents.revision columns`

---

### T2: Define Envelope struct + JSON marshal + validation

**What**: Create `internal/event_runtime/envelope.go` with `Envelope` struct matching ADR-043 §4.1, plus `MarshalJSON` (stable field order), `UnmarshalJSON`, `Validate` (rejects empty `event_type`, empty `aggregate_id`, payload > 1 MiB), and `NewEnvelope` constructor (auto-assigns UUID v7 `EventID` and `CreatedAt`).
**Where**: `internal/event_runtime/envelope.go` (new), `internal/event_runtime/envelope_test.go` (new)
**Depends on**: None
**Reuses**: Go stdlib `encoding/json`; existing project pattern of package-level types with co-located `_test.go` (see `internal/compiler/note.go`).
**Requirement**: EVT-02, EVT-03, EVT-04, EVT-05, EVT-06
**Status**: Verified (2026-09-19)

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `Envelope` struct exported with all 18 fields from ADR-043 §4.1 (`EventID`, `SchemaVersion`, `EventType`, `Producer`, `CreatedAt`, `MonotonicNS`, `SessionID`, `ConversationID`, `AggregateID`, `Epoch`, `Sequence`, `CorrelationID`, `CausationID`, `Actor`, `Locale`, `Priority`, `ContentType`, `Payload json.RawMessage`, `Provenance`, `Trace`)
- [ ] `NewEnvelope(eventType, aggregateID)` constructor sets `EventID` (UUID v7 via `github.com/google/uuid`), `CreatedAt` (RFC 3339 nano UTC), `SchemaVersion=1`, `Priority="normal"`
- [ ] `Validate()` returns `ErrInvalidEnvelope` if `event_type` or `aggregate_id` is empty; `ErrPayloadTooLarge` if payload > 1 MiB
- [ ] `MarshalJSON` produces stable field order matching ADR-043 §4.1 (no Go map iteration ordering leak)
- [ ] Round-trip test: `Unmarshal(Marshal(env))` produces deep-equal struct
- [ ] All error sentinels exported: `ErrInvalidEnvelope`, `ErrPayloadTooLarge`, `ErrTxRequired`, `ErrDuplicateSubscriber`, `ErrEventLogNotInitialized`

**Tests**: unit
**Gate**: quick

---

### T3: Implement Log (event_log reader/writer)

**What**: Create `internal/event_runtime/log.go` with `Log` struct exposing `Append(ctx, tx, env)` (writes envelope in caller's tx — NO implicit BEGIN/COMMIT), `LastSequence(ctx)` (SELECT MAX), `ReadUnacked(ctx, subscriberName, limit)` (returns envelopes with `sequence > projection_cursor.last_sequence` for that subscriber, ORDER BY sequence ASC), `MarkAcked(ctx, tx, sequence, reason)` (UPDATE acked_at).
**Where**: `internal/event_runtime/log.go` (new), `internal/event_runtime/log_test.go` (new)
**Depends on**: T1, T2
**Reuses**: Existing DB connection pattern from `internal/db/store.go` (`*sql.DB` or `*sql.Tx`); existing test setup pattern from `internal/db/store_test.go` (in-memory SQLite via `sql.Open("sqlite3", ":memory:")`).
**Requirement**: EVT-01 (write path), EVT-07 (tx-respecting API)
**Status**: Verified (2026-09-19)

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `Log` struct holds `*sql.DB`; constructor `NewLog(db *sql.DB) *Log`
- [ ] `Append(ctx, tx, env)` does NOT call `BEGIN`/`COMMIT`; uses caller's tx only; returns error from tx on failure
- [ ] `Append` checks `env.Validate()` before INSERT (rejects invalid envelope with `ErrInvalidEnvelope`)
- [ ] `LastSequence(ctx)` returns `int64` (0 if table empty)
- [ ] `ReadUnacked(ctx, subscriberName, limit)` returns `[]Envelope` ordered by `sequence ASC`, filtered by `projection_cursor.last_sequence` for that subscriber (or 0 if no cursor)
- [ ] `MarkAcked(ctx, tx, sequence, reason)` updates both `event_log.acked_at` AND `projection_cursor.last_sequence` atomically; `reason=""` for success, non-empty for permanent reject (e.g. `unsupported_schema`)
- [ ] Test `TestLog_AppendInTransaction_AndReorder`: insert 3 envelopes in separate tx, verify `ORDER BY sequence DESC` returns [3,2,1]
- [ ] Test `TestLog_DuplicateEventID_IsIdempotent`: insert same `event_id` twice in different tx; second insert returns `ErrDuplicateEventID` (mapped from SQLite `UNIQUE constraint failed`)
- [ ] Test `TestLog_PayloadTooLarge_IsRejected`: 2 MiB payload → `ErrPayloadTooLarge`, no DB write
- [ ] Test `TestLog_AppendNilTx_IsRejected`: returns `ErrTxRequired`

**Tests**: integration (uses real SQLite via `:memory:`)
**Gate**: full

---

### Phase 2

### T4: Implement Outbox (transactional emit helper)

**What**: Create `internal/event_runtime/outbox.go` with `Outbox` struct exposing `Emit(ctx, tx, env)` and `EmitBatch(ctx, tx, envs)`. Both delegate to `Log.Append` but add explicit error wrapping (`%w`) and the `ErrTxRequired` guard.
**Where**: `internal/event_runtime/outbox.go` (new), `internal/event_runtime/outbox_test.go` (new)
**Depends on**: T2, T3
**Reuses**: `Log.Append` from T3; existing error wrapping pattern (`fmt.Errorf("...: %w", err)`) used throughout `internal/compiler/`.
**Requirement**: EVT-08, EVT-09, EVT-10
**Status**: Verified (2026-09-19)

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `Outbox` is a thin wrapper; no state of its own (`struct {}`)
- [ ] `Emit(ctx, tx, env)` returns `ErrTxRequired` if `tx == nil`; otherwise calls `Log.Append(ctx, tx, env)`
- [ ] `EmitBatch(ctx, tx, envs)` calls `Emit` in a loop; short-circuits on first error returning the original error wrapped with `event_type` context
- [ ] Each envelope validated before INSERT (`Log.Append` already does this; Outbox doesn't double-validate)
- [ ] No commit/rollback logic in Outbox — caller owns the transaction lifecycle

**Tests**: unit
**Gate**: quick

---

### T5: Outbox atomicity tests (transactional guarantee)

**What**: Add `TestOutbox_AtomicWithEffect` and `TestOutbox_RetryDoesNotDuplicateEventID` to `internal/event_runtime/outbox_test.go`. These tests prove the outbox pattern works end-to-end against a real SQLite DB (rollback hides both INSERT into `documents` AND the outbox event).
**Where**: `internal/event_runtime/outbox_test.go` (modify — add tests to T4's file)
**Depends on**: T4
**Reuses**: Test helpers from T3; existing transaction patterns from `internal/db/store_test.go`.
**Requirement**: EVT-10 (atomicidade), EVT-12 (idempotência por event_id)
**Status**: Verified (2026-09-19)

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `TestOutbox_AtomicWithEffect`: open tx, INSERT into `documents` (fictitious table for test) + `Outbox.Emit(env)` + ROLLBACK; verify `SELECT COUNT(*) FROM event_log` returns 0 (event never visible after rollback)
- [ ] `TestOutbox_RetryDoesNotDuplicateEventID`: emit event with `event_id=X` in tx1 (COMMIT); emit same `event_id=X` in tx2 (COMMIT); assert no duplicate effect on a fictitious `effects` table (use UNIQUE constraint on `event_id` to simulate idempotency check)
- [ ] Test names match spec AC names verbatim (Spec §P2 AC 4 and AC 5)

**Tests**: integration
**Gate**: full
**Commit**: `feat(event-runtime): outbox helper + atomicity invariants`

---

### Phase 3

### T6: Subscriber interface + RegisterSubscriber helper

**What**: Create `internal/event_runtime/subscriber.go` with the `Subscriber` interface (`Name() string`, `EventTypes() []string`, `Handle(ctx, env) error`, `MaxAckPending() int`) and `RegisterSubscriber(dispatcher, sub)` validation helper. The interface is the contract every projection future-proofs against.
**Where**: `internal/event_runtime/subscriber.go` (new), `internal/event_runtime/subscriber_test.go` (new)
**Depends on**: None (interface-only; dispatcher in T7 uses it)
**Reuses**: Existing interface pattern from `internal/embedder/` (`Provider` interface); existing registry pattern from `internal/compiler/`.
**Requirement**: EVT-12, EVT-13, EVT-14
**Status**: Verified (2026-09-19)

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `Subscriber` interface exported with exactly the 4 methods above
- [ ] `RegisterSubscriber(d *Dispatcher, s Subscriber)` rejects nil dispatcher and nil subscriber with `ErrNilSubscriber`; rejects duplicate `Name()` with `ErrDuplicateSubscriber` (does NOT spawn goroutines — that's `Dispatcher.Start`)
- [ ] `Dispatcher` struct stub defined (empty struct + mutex + map[string]Subscriber) so T7 has a target to extend
- [ ] Test `TestSubscriber_RegisterDuplicateName_IsRejected`: register "audit" twice, second call returns `ErrDuplicateSubscriber`
- [ ] Test `TestSubscriber_RegisterNil_IsRejected`: pass nil → `ErrNilSubscriber`

**Tests**: unit
**Gate**: quick

---

### T7: Implement Dispatcher (fan-out, ACK, retry, panic recovery)

**What**: Create `internal/event_runtime/dispatcher.go` with `Dispatcher.Start(ctx)` (spawns worker goroutines reading from `Log.ReadUnacked` per subscriber), `Stop(ctx)` (graceful drain up to 30s, then NACK remaining), `Subscribe(eventTypePattern, handler)` (matches `event_type` with glob), ACK/NACK protocol, exponential backoff retry (`base=100ms`, `factor=2`, `jitter=±20%`, `cap=30s`), `MaxAckPending=64` backpressure, panic recovery, `unsupported_schema` permanent ACK with reason.
**Where**: `internal/event_runtime/dispatcher.go` (new), `internal/event_runtime/dispatcher_test.go` (new)
**Depends on**: T3, T6
**Reuses**: `Log` from T3; existing goroutine pattern from `internal/watcher/` (fsnotify-based watcher has cancel + drain logic).
**Requirement**: EVT-11, EVT-15, EVT-16, EVT-17, EVT-18, EVT-19
**Status**: Verified (2026-09-19)

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `Dispatcher` struct holds: `db *sql.DB`, `log *Log`, `mu sync.RWMutex`, `subs map[string]*subState`, `stopCh chan struct{}`, `pollInterval time.Duration` (default 100ms), `gracePeriod time.Duration` (default 30s)
- [ ] `Subscribe(pattern string, handler func(ctx, env) error) error` — pattern matching via `path.Match` against `env.EventType`
- [ ] `Start(ctx)` spawns N goroutines (1 per subscriber); each goroutine polls `Log.ReadUnacked(sub.Name, MaxAckPending)` → delivers to handler → on nil ACK, on error re-queues with backoff timer, on panic recovers + logs + treats as transient
- [ ] `Stop(ctx)` waits up to `gracePeriod` for in-flight handlers; on timeout, cancels context and NACKs remaining (re-delivered on next start)
- [ ] Backoff state held in-memory only (subscriber restart resets retry budget — acceptable for F0; document in code comment)
- [ ] `unsupported_schema` (env.SchemaVersion > current 1) → ACK with `acked_reason='unsupported_schema'`, no retry, emits a `event_runtime.unsupported_schema_acked` log line
- [ ] Test `TestDispatcher_DeliversToSubscribers`: 3 subscribers, 1 event, all receive
- [ ] Test `TestDispatcher_ACKUpdatesCursor`: handler OK → `projection_cursor.last_sequence` for that subscriber grows
- [ ] Test `TestDispatcher_RetryOnError`: handler returns error 3x then OK; assert 3 attempts with backoff
- [ ] Test `TestDispatcher_PanicRecoveredAsTransient`: handler panics → log + retry (no crash)
- [ ] Test `TestDispatcher_UnsupportedSchemaIsAckedPermanently`: schema_version=99 → ACK with reason, no retry
- [ ] Test `TestDispatcher_MaxAckPending_PausesRead`: slow subscriber + producer emits 5 → assert not > 1 in-flight
- [ ] Test `TestDispatcher_StopGraceful`: 2 in-flight handlers, Stop with grace=100ms waits up to 100ms, then NACKs

**Tests**: integration (uses real SQLite + goroutines)
**Gate**: full

---

### T8: Redaction rules (LLM02 — PII gate)

**What**: Create `internal/event_runtime/redaction.go` with `Redact(env Envelope) Envelope` that deep-copies the envelope and rewrites PII paths in `Payload` according to configurable rules. Default rules (ADR-050 LLM02): `payload.transcript` → `[REDACTED:transcript]`, `payload.audio_url` → `[REDACTED:audio_url]`, `payload.prompt` → `[REDACTED:prompt]`. Auditable via `redacted_payload_hash` (SHA-256 of original payload, before redaction).
**Where**: `internal/event_runtime/redaction.go` (new), `internal/event_runtime/redaction_test.go` (new)
**Depends on**: T2
**Reuses**: `crypto/sha256` from stdlib; existing JSON traversal pattern (none direct, but `encoding/json` round-trip is the safest path).
**Requirement**: EVT-13 (PII gate — part of P5 Audit Subscriber story)

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `Redact(env Envelope) Envelope` returns a new Envelope (input not mutated); `Payload` is `json.RawMessage` rewritten by default rules; `Provenance.RedactedFields` list added if any field matched
- [ ] `DefaultRules()` returns `[]Rule{{Path: "transcript", Tag: "[REDACTED:transcript]"}, ...}` — exported for customization
- [ ] `WithRules(rules []Rule)` option pattern for custom redaction sets
- [ ] `RedactedPayloadHash` field added to `Envelope.Provenance` (SHA-256 hex of pre-redaction payload bytes)
- [ ] Test `TestRedact_TranscriptIsReplaced`: emit envelope with `payload.transcript="hello world"`, assert output Payload contains `"[REDACTED:transcript]"` and NOT `"hello world"`
- [ ] Test `TestRedact_AudioURLIsReplaced`: same for `payload.audio_url`
- [ ] Test `TestRedact_NoMatch_ReturnsOriginalPayload`: envelope with no PII fields → Payload unchanged
- [ ] Test `TestRedact_HashIsStable`: same input → same SHA-256 hash

**Tests**: unit
**Gate**: quick

---

### T9: Audit subscriber (JSONL log with redaction)

**What**: Create `internal/event_runtime/audit_subscriber.go` implementing `Subscriber` interface. Persists each event as a JSONL line at `.memory/audit.log` with `event_id`, `sequence`, `event_type`, `actor`, `redacted_payload_hash` (NEVER the raw payload). Emits `audit.persisted` event per processed envelope (creates a feedback loop — must be marked as `provenance.audit_subscriber=true` to avoid re-delivery to audit itself).
**Where**: `internal/event_runtime/audit_subscriber.go` (new), `internal/event_runtime/audit_subscriber_test.go` (new)
**Depends on**: T6, T7, T8
**Reuses**: `Redact` from T8; existing JSONL write pattern from `internal/compiler/` (atomic write via tmp + rename).
**Requirement**: EVT-12 (audit persisted), EVT-13 (PII redaction applied)

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `AuditSubscriber` struct holds: `path string` (default `.memory/audit.log`), `mu sync.Mutex`, `out *os.File`
- [ ] `NewAuditSubscriber(path string) *AuditSubscriber` opens file in append mode; refuses if path is read-only filesystem (test with `os.Chmod(0444)` + assert error)
- [ ] `Name()` returns `"audit"`; `EventTypes()` returns `[]string{"*"}` (subscribes to everything)
- [ ] `Handle(ctx, env)` calls `Redact(env)`, writes JSONL line `{event_id, sequence, event_type, actor, redacted_payload_hash, created_at}`, calls `fsync` for durability, returns nil
- [ ] `MaxAckPending()` returns 256 (audit is fast, can handle more concurrency)
- [ ] On read-only filesystem: `Handle` returns error → dispatcher retries with backoff (fail-closed, never silently drop)
- [ ] Test `TestAuditSubscriber_RedactsPII`: emit envelope with transcript; assert `audit.log` line does NOT contain the transcript string; assert `[REDACTED:transcript]` present
- [ ] Test `TestAuditSubscriber_HashIsStable`: same envelope twice → same SHA-256 in both lines
- [ ] Test `TestAuditSubscriber_ReadOnlyFilesystem_FailsClosed`: chmod 0444 the audit log path; assert `Handle` returns error; remove chmod after test
- [ ] Test `TestAuditSubscriber_AppendDoesNotCorrupt`: write 5 lines; reopen; assert all 5 lines readable in order

**Tests**: integration (uses real filesystem via `t.TempDir()`)
**Gate**: full
**Commit**: `feat(event-runtime): dispatcher + audit subscriber with PII redaction`

---

### Phase 4

### T10: CLI subcommand `mem events` (tail, inspect, replay, trace, last-sequence, stats)

**What**: Create `cmd/mem/events.go` with 6 subcommands under the `events` umbrella: `tail` (last N events, default 20, filterable by `--session`, `--type`, `--limit`), `inspect <event_id>` (full envelope pretty-printed), `replay --since <seq> [--to <seq>] [--dry-run|--apply]` (re-emit to subscribers, dry-run default), `trace <correlation_id>` (DAG of events with same correlation_id), `last-sequence` (current max sequence), `stats` (totals + oldest unacked). All output as JSON Lines or pretty-printed JSON.
**Where**: `cmd/mem/events.go` (new)
**Depends on**: T3, T7
**Reuses**: Existing CLI pattern from `cmd/mem/note.go` (cobra-style subcommands + JSON output); existing config loading from `internal/config/`.
**Requirement**: EVT-14, EVT-15, EVT-16, EVT-17, EVT-18, EVT-19, EVT-20, EVT-21

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `mem events tail [--session ID] [--type PATTERN] [--limit N]` prints N events (default 20) one per line in JSON, ordered by `sequence DESC`
- [ ] `mem events inspect <event_id>` prints full envelope pretty-printed (2-space indent); exits non-zero if not found
- [ ] `mem events replay --since <seq> [--to <seq>] [--dry-run|--apply]`: dry-run is default and only prints events without delivering; `--apply` requires `--yes` or interactive confirmation; structured warning logged before delivery
- [ ] `mem events trace <correlation_id>` outputs DAG (one line per event with `sequence`, `event_type`, `aggregate_id`, `actor`, parent reference)
- [ ] `mem events last-sequence` prints current `MAX(sequence)` from `event_log`
- [ ] `mem events stats` prints JSON object: `{"total_events": N, "events_by_type_top10": {...}, "oldest_unacked_sequence": N, "oldest_unacked_age_seconds": N}`
- [ ] All subcommands exit 2 with clear error if `.memory/config.yaml` missing (point to `mem init`)
- [ ] `mem events --help` documents all 6 subcommands

**Tests**: integration (co-located with T11)
**Gate**: full

---

### T11: CLI tests for `mem events`

**What**: Create `cmd/mem/events_test.go` with 6 tests covering happy path + every edge case listed in spec ACs.
**Where**: `cmd/mem/events_test.go` (new)
**Depends on**: T10
**Reuses**: Existing CLI test pattern from `cmd/mem/note_test.go` (exec `mem` binary via `os/exec` against a temp DB).
**Requirement**: EVT-14..EVT-20 (all CLI ACs)

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `TestCLI_EventsTail_PrintsJSONLines`: populate 5 events; assert stdout has 5 lines, each valid JSON, ordered DESC
- [ ] `TestCLI_EventsInspect_NotFound_ExitsNonZero`: pass non-existent ID; assert exit code != 0
- [ ] `TestCLI_EventsInspect_Found_PrintsEnvelope`: pass real ID; assert stdout contains all fields
- [ ] `TestCLI_ReplayDryRun_DoesNotApplyEffects`: subscriber counts Handle calls; `--dry-run` → count=0; `--apply --yes` → count>0
- [ ] `TestCLI_ReplayApply_RequiresConfirmation`: `--apply` without `--yes` should prompt or fail-closed (whichever the design specifies in T10; test asserts the chosen behavior)
- [ ] `TestCLI_Trace_ReturnsAllEventsWithCorrelationID`: emit 3 events with same correlation_id, 2 with different; trace → 3 lines
- [ ] `TestCLI_LastSequence_PrintsMaxSequence`: emit 7 events; assert output is "7"
- [ ] `TestCLI_Stats_ReportsOldestUnacked`: emit events; leave 2 unacked; assert stats includes `oldest_unacked_age_seconds > 0`
- [ ] `TestCLI_MissingConfig_Exits2`: run with no `.memory/config.yaml`; assert exit code 2 and stderr mentions `mem init`

**Tests**: integration
**Gate**: full
**Commit**: `feat(cli): mem events {tail,inspect,replay,trace,last-sequence,stats}`

---

### T12: `mem doctor --events` integration

**What**: Add optional `--events` flag to `mem doctor` that, when present, runs additional checks: total events, oldest unacked age, active subscribers count. Output as a new section in the doctor report.
**Where**: `cmd/mem/doctor.go` (modify — add `--events` flag), `cmd/mem/doctor_test.go` (modify — add test)
**Depends on**: T10
**Reuses**: Existing doctor pattern from `cmd/mem/drift.go` (flag-based conditional checks); existing report format from `internal/drift/`.
**Requirement**: Success Criterion: "`mem doctor` ganha check opcional `--events` reportando `total_events`, `oldest_unacked_age_seconds`, `subscribers_active`"

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `mem doctor --events` runs after the standard doctor checks and appends an "Events" section
- [ ] Section reports: `total_events`, `oldest_unacked_age_seconds`, `subscribers_active` (count from `event_runtime.Dispatcher`), `unacked_count`
- [ ] Warns (not fails) if `oldest_unacked_age_seconds > 300` (5 minutes) — indicates stuck dispatcher
- [ ] Existing doctor tests still pass (`go test ./cmd/mem/...`)
- [ ] Test `TestDoctor_EventsFlag_ReportsMetrics`: setup dispatcher + 3 events (2 acked, 1 unacked); run `mem doctor --events`; assert section contains expected fields

**Tests**: integration
**Gate**: full
**Commit**: `feat(cli): mem doctor --events — event_runtime health check`

---

## Task Granularity Check

| Task | Scope | Status |
| --- | --- | --- |
| T1: Add event_log + projection_cursor + documents columns | 2 files (schema.go + store.go), 1 cohesive concern | ✅ Granular |
| T2: Envelope struct + JSON + validation | 1 new file + 1 test file, single type + 5 functions | ✅ Granular |
| T3: Log reader/writer | 1 new file + 1 test file, single struct + 4 methods | ✅ Granular |
| T4: Outbox helper | 1 new file + 1 test file, thin wrapper | ✅ Granular |
| T5: Outbox atomicity tests | 2 test cases in T4's test file | ✅ Granular |
| T6: Subscriber interface | 1 new file + 1 test file, interface + register helper | ✅ Granular |
| T7: Dispatcher | 1 new file + 1 test file, complex but cohesive (delivery core) | ✅ Granular (single delivery engine) |
| T8: Redaction rules | 1 new file + 1 test file, single concern (PII gate) | ✅ Granular |
| T9: Audit subscriber | 1 new file + 1 test file, single subscriber implementation | ✅ Granular |
| T10: CLI subcommands | 1 new file, 6 cohesive subcommands | ✅ Granular (single CLI surface) |
| T11: CLI tests | 1 new file, tests for T10 | ✅ Granular |
| T12: doctor --events | 2 file modifications (doctor.go + test), single new flag | ✅ Granular |

---

## Diagram-Definition Cross-Check

| Task | Depends On (body) | Diagram Shows | Status |
| --- | --- | --- | --- |
| T1 | None | (Phase 1 root) | ✅ Match |
| T2 | None | (Phase 1 root) | ✅ Match |
| T3 | T1, T2 | T1 → T3, T2 → T3 | ✅ Match |
| T4 | T2, T3 | T3 → T4 (cross-phase, ignored by validator) | ✅ Cross-phase |
| T5 | T4 | T4 → T5 | ✅ Match |
| T6 | None | (Phase 3 root) | ✅ Match |
| T7 | T3, T6 | T6 → T7 (T3 cross-phase, ignored) | ✅ Cross-phase |
| T8 | T2 | (cross-phase, ignored) | ✅ Cross-phase |
| T9 | T6, T7, T8 | T6 → T9, T7 → T9, T8 → T9 | ✅ Match |
| T10 | T3, T7 | (both cross-phase, ignored) | ✅ Cross-phase |
| T11 | T10 | T10 → T11 | ✅ Match |
| T12 | T10 | T10 → T12 | ✅ Match |

Intra-phase `Depends on` resolve to diagram arrows. Cross-phase dependencies (T2→T4, T3→T7, T3→T10, T7→T10, T2→T8) are intentionally implicit — the forward-phase check above blocks forward dependencies, and the validator's `intra_phase` filter excludes cross-phase edges from cross-check. No forward dependencies. ✅

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| --- | --- | --- | --- | --- |
| T1 | Migration SQL (DDL) | none (build gate only) | Tests: none | ✅ OK |
| T2 | Domain (Envelope struct + validation) | unit | Tests: unit | ✅ OK |
| T3 | Repository (Log reader/writer) | integration | Tests: integration | ✅ OK |
| T4 | Service (Outbox helper) | unit | Tests: unit | ✅ OK |
| T5 | Service (Outbox atomicity) | integration | Tests: integration | ✅ OK |
| T6 | Domain (Subscriber interface) | unit | Tests: unit | ✅ OK |
| T7 | Service (Dispatcher) | integration | Tests: integration | ✅ OK |
| T8 | Domain (Redaction rules) | unit | Tests: unit | ✅ OK |
| T9 | Service (AuditSubscriber) | integration | Tests: integration | ✅ OK |
| T10 | CLI subcommand | integration | Tests: integration (co-located with T11) | ⚠️ Note: T10 has no own test file — T11 covers it. **Justification**: T10 is a thin CLI wrapper; T11 explicitly tests all 6 subcommands via the binary. Tests are co-located (one phase later in same file pair), not deferred. **Acceptable per skill rule**: "Merge forward — Move the untestable task's tests into the earliest task where they become runnable." T10 cannot be unit-tested without binary execution harness (added by T11), so merging tests into T11 is the correct pattern. |
| T11 | CLI tests | integration | Tests: integration | ✅ OK |
| T12 | CLI subcommand modification | integration | Tests: integration | ✅ OK |

All tasks have tests matching the Coverage Matrix. T10/T11 co-location is a forward-merge per skill rules (T10 cannot be tested until binary harness exists; T11 adds it).

---

## Phase Execution Map

```
Phase 1, Phase 2, Phase 3, Phase 4 (sequentially)

Phase 1
T1 -> T3
T2 -> T3

Phase 2
T4 -> T5

Phase 3
T6 -> T7
T6 -> T9
T7 -> T9
T8 -> T9

Phase 4
T10 -> T11
T10 -> T12
```

Execution is strictly sequential — no intra-phase parallelism. Phases run in order.

**Batch packing at Execute time:**

- 12 tasks total > 8 → offer batch sub-agents
- Suggested packing: Batch 1 = Phase 1 + Phase 2 (T1-T5, 5 tasks); Batch 2 = Phase 3 (T6-T9, 4 tasks); Batch 3 = Phase 4 (T10-T12, 3 tasks)
- Alternatively: Batch 1 = Phase 1-3 (T1-T9, 9 tasks); Batch 2 = Phase 4 (T10-T12, 3 tasks) — splits cleaner at the "CLI surface" boundary
- Worker offers are surfaced at Execute time, not now.

---

## Closing the Loop

After the LAST task (T12), the `tlc-spec-driven` Verifier runs automatically:

- Spec-anchored outcome check: every EVT-NN AC has a corresponding test asserting the spec-defined outcome
- Discrimination sensor: inject behavior-level faults (e.g., event_id collision, schema_version=99 mid-replay, panic in handler) → tests must kill them
- Writes `.specs/features/event-runtime-event-log/validation.md` with PASS/FAIL + per-AC evidence + file:line citations
- Returns ranked gap list to orchestrator; gaps become fix tasks (bounded 3 iterations before escalation)
