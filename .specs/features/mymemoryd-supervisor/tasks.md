# mymemoryd-supervisor Tasks

## Execution Protocol (MANDATORY — do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path.

**If the skill cannot be activated, STOP and tell the user — do not proceed without it.**

---

**Spec**: `.specs/features/mymemoryd-supervisor/spec.md`
**Design**: [ADR-042](../../../docs/adr/042-nucleo-local-mymemoryd-com-workers-sidecar-isolados.md)
**Security**: [ADR-050 (Threat Model OWASP LLM)](../../../docs/adr/050-threat-model-owasp-llm-aplicado-ao-mymemory.md) — controls LLM05 (egress), LLM06 (sandbox), LLM10 (resource consumption) apply.
**Status**: Draft → Approved → In Progress → Done

---

## Test Coverage Matrix

> Generated from codebase sampling. Guidelines found: `.agents/rules/always-quality-gate.md`, `.agents/rules/always-test.md`. Existing pattern: `internal/<pkg>/*_test.go` co-located.

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| --- | --- | --- | --- | --- |
| Service (supervisor lifecycle, IPC) | unit + integration | All branches; 1:1 to spec ACs; crash + restart scenarios | `internal/supervisor/*_test.go`, `internal/ipc/jsonrpc/*_test.go` | `go test -count=1 ./internal/supervisor/... ./internal/ipc/...` |
| Domain (WorkerSpec, Profile, SandboxPolicy) | unit | YAML parse round-trip; validation | `internal/supervisor/profile_test.go` | `go test -count=1 ./internal/supervisor/...` |
| Process management (start/stop/kill) | integration | Real subprocess; heartbeat timeout; backoff restart | `internal/supervisor/manager_test.go` | `go test -count=1 ./internal/supervisor/...` |
| CLI (mem up/down/status/profiles) | integration | Happy path + edge cases + idempotency | `cmd/mem/up_test.go`, `cmd/mem/down_test.go` | `go test -count=1 ./cmd/mem/...` |

---

## Gate Check Commands

| Gate Level | When to Use | Command |
| --- | --- | --- |
| Quick | After tasks with unit tests only (T2, T4, T6) | `go test -count=1 -short ./internal/supervisor/... ./internal/ipc/...` |
| Full | After tasks with integration tests (T1, T3, T5, T7, T8, T9) | `go test -count=1 ./internal/supervisor/... ./internal/ipc/... ./cmd/mem/...` |
| Build | After phase completion | `gofmt -l . && go build ./... && go test -count=1 ./...` |

---

## Execution Plan

Phases are ordered and run sequentially. Tasks within a phase execute in order.

### Phase 1: Supervisor skeleton + lock file

Foundation: lifecycle loop, lock file, heartbeat, restart backoff.

```
T1 -> T2
T2 -> T3
```

### Phase 2: Profile parsing + workers manifest

YAML parsing for profiles, worker declaration, inheritance.

```
T4 -> T5
```

### Phase 3: IPC + framing binário

JSON-RPC 2.0 over stdio + Unix socket with binary framing.

```
T6 -> T7
```

### Phase 4: CLI + sandbox + audit log

CLI subcommands, sandbox best-effort, audit JSONL.

```
T2 -> T9
T5 -> T9
```

Total: **9 tasks across 4 phases**. Fits 2 batches at Execute time (Phase 1-2 inline, Phase 3-4 sub-agent).

---

## Task Breakdown

> **Note on validator**: `validate_tasks.py` keeps the last-seen `phase_idx` after `### Phase N` headers. We repeat `### Phase N` markers immediately above each phase's first task to keep per-task phase assignment accurate.

### Phase 1

### T1: Supervisor entrypoint + lock file

**What**: Create `cmd/mymemoryd/main.go` (entrypoint reading `.memory/profiles/default.yaml`, starting supervisor). Create `internal/supervisor/manager.go` with `Manager` struct holding workers map + lock. Create `internal/supervisor/lock.go` reading/writing `.memory/supervisor.lock` with PID + timestamp. Stale lock (>30s without heartbeat) removed on next start.
**Where**: `cmd/mymemoryd/main.go` (new), `internal/supervisor/manager.go` (new), `internal/supervisor/lock.go` (new)
**Depends on**: None
**Reuses**: Existing CLI entrypoint pattern from `cmd/mem/main.go`; existing PID handling from `os.Getpid()`.
**Requirement**: SUPR-01, SUPR-06
**Status**: Verified (2026-09-20)

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [x] `mymemoryd` binary reads `--profile <name>` flag (default `default`)
- [x] On startup: checks `.memory/supervisor.lock`, refuses if fresh + PID alive, removes if stale
- [x] Writes lock with `{pid, started_at}` JSON on successful start
- [x] Removes lock on graceful shutdown (SIGTERM/SIGINT)
- [x] Exits with code 2 on `ErrInvalidProfile` (fail-closed)
- [x] Test `TestLock_StaleRemoved`: create lock with old timestamp + dead PID → new start removes and proceeds
- [x] Test `TestLock_FreshBlocks`: create lock with alive PID → new start returns `ErrAlreadyRunning`

**Tests**: integration
**Gate**: full
**Commit**: `feat(supervisor): entrypoint + lock file + stale lock cleanup`

---

### T2: Worker lifecycle (start, stop, health check)

**What**: Extend `internal/supervisor/manager.go` with worker lifecycle: `Start(ctx, workerSpec) (Worker, error)`, `Stop(ctx, worker, timeout) error`, `HealthCheck(ctx, worker) (status, error)`. Health check via 5s heartbeat with 3 misses = unhealthy. Emit `worker.started`, `worker.stopped`, `worker.heartbeat_lost` events to event_log.
**Where**: `internal/supervisor/manager.go` (modify), `internal/supervisor/manager_test.go` (new)
**Depends on**: T1
**Reuses**: `os/exec` for subprocess; `event_runtime.Log.Append` for events.
**Requirement**: SUPR-01, SUPR-03
**Status**: Verified (2026-09-20)

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [x] `Start` launches worker subprocess with declared command + args
- [x] `Stop` sends SIGTERM, waits 10s, sends SIGKILL to survivors
- [x] `HealthCheck` reads worker heartbeat channel; 3 misses = unhealthy
- [x] `worker.started` event emitted with `worker_id`, `pid`, `ts`
- [x] `worker.stopped` event emitted with `worker_id`, `reason`, `ts`
- [x] `worker.heartbeat_lost` event emitted after 3 misses
- [x] Test `TestWorker_StartStopRoundtrip`: start fake worker → stop → 2 events in audit
- [x] Test `TestWorker_HeartbeatLost`: kill worker's heartbeat goroutine → event after 15s

**Tests**: integration (uses fake worker binary)
**Gate**: full
**Commit**: `feat(supervisor): worker lifecycle + heartbeat + events`

---

### T3: Restart with exponential backoff

**What**: Add `Restart(ctx, worker) error` to `Manager`. Detect unexpected exit (code ≠ 0) and restart with exponential backoff (100ms base, 30s cap, jitter ±20%). After 5 retries, emit `supervisor.worker_failed` event and stop trying.
**Where**: `internal/supervisor/manager.go` (modify), `internal/supervisor/restart_test.go` (new)
**Depends on**: T2
**Reuses**: `time.AfterFunc` for backoff; existing retry contract from ADR-043 §6.
**Requirement**: SUPR-02
**Status**: Verified (2026-09-20)

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [x] `Restart` detects worker exit (channel from `cmd.Wait()`)
- [x] Applies backoff: 100ms × 2^n + jitter, capped 30s
- [x] After 5 retries: emits `supervisor.worker_failed` event with `worker_id`, `exit_code`, `retries=5`
- [x] After 5 retries: stops trying (worker marked `failed`)
- [x] Test `TestRestart_ExponentialBackoff`: crash 3 times → 3 restarts with increasing intervals
- [x] Test `TestRestart_MaxRetriesThenFailed`: crash 6 times → `worker_failed` event after 5th

**Tests**: integration
**Gate**: full
**Commit**: `feat(supervisor): restart with exponential backoff + max retries`

---

### Phase 2

### T4: Profile YAML schema + parser

**What**: Define `WorkerSpec` struct (`name`, `command`, `args`, `required`, `egress`, `config`). Define `Profile` struct (`schema_version`, `extends`, `workers`). Create `internal/supervisor/profile.go` with `LoadProfile(path) (*Profile, error)` parsing YAML with `gopkg.in/yaml.v3`.
**Where**: `internal/supervisor/profile.go` (new), `internal/supervisor/profile_test.go` (new)
**Depends on**: None
**Reuses**: YAML parsing from `internal/config/`.
**Requirement**: SUPR-12, SUPR-13
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `WorkerSpec` + `Profile` structs exported with validation
- [ ] `LoadProfile` parses YAML, rejects unknown fields, returns error on malformed
- [ ] Round-trip test: write profile → parse → fields equal
- [ ] Test `TestProfile_LoadValid`: parse `.memory/profiles/default.yaml` template
- [ ] Test `TestProfile_LoadMalformed_ReturnsErrInvalidProfile`

**Tests**: integration
**Gate**: full
**Commit**: `feat(supervisor): profile YAML schema + parser`

---

### T5: Profile inheritance + required vs optional

**What**: Extend `LoadProfile` to support `extends: default` — merge parent's `workers` list and override `config` keys. Implement `required: true|false` semantics: required worker failure → exit 2; optional failure → emit `worker.optional_failed` event but continue.
**Where**: `internal/supervisor/profile.go` (modify), `internal/supervisor/inheritance_test.go` (new)
**Depends on**: T4
**Reuses**: T4 parser.
**Requirement**: SUPR-12, SUPR-13, SUPR-14, SUPR-15
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `voice.yaml extends default.yaml` → effective workers = merged list (parent first, child overrides)
- [ ] `required: true` worker fails to start → manager exits with code 2 after 30s timeout
- [ ] `required: false` worker fails to start → emits `worker.optional_failed` event, continues
- [ ] Test `TestProfile_Inheritance_MergesWorkers`
- [ ] Test `TestProfile_RequiredWorkerMissing_FailsClosed`
- [ ] Test `TestProfile_OptionalWorkerMissing_ContinuesWithEvent`

**Tests**: integration
**Gate**: full
**Commit**: `feat(supervisor): profile inheritance + required/optional semantics`

---

### Phase 3

### T6: JSON-RPC 2.0 over stdio

**What**: Create `internal/ipc/jsonrpc/server.go` implementing JSON-RPC 2.0 server over stdio. Handshake: worker sends `initialize` request → server responds with capabilities. Reject malformed frames. Enforce 30s request timeout. Support batch (limit 100).
**Where**: `internal/ipc/jsonrpc/server.go` (new), `internal/ipc/jsonrpc/server_test.go` (new)
**Depends on**: None
**Reuses**: `net/rpc` or hand-rolled JSON-RPC parser; `bufio.Scanner` for stdio.
**Requirement**: SUPR-07, SUPR-09, SUPR-10, SUPR-11
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] Server accepts JSON-RPC 2.0 frames over stdio
- [ ] `initialize` handshake completes within 5s or fail-closed
- [ ] Malformed frame → response with `code: -32600 Invalid Request`
- [ ] Request timeout 30s → `ipc.timeout` event + connection close
- [ ] Batch > 100 → response with `code: -32613 Batch limit exceeded`
- [ ] Test `TestJsonRpc_InitializeHandshake`
- [ ] Test `TestJsonRpc_RejectsMalformedFrame`
- [ ] Test `TestJsonRpc_RejectsBatchOver100`

**Tests**: unit + integration
**Gate**: full
**Commit**: `feat(ipc): JSON-RPC 2.0 server over stdio`

---

### T7: Unix socket + 4-byte binary framing

**What**: Add Unix socket transport to `internal/ipc/jsonrpc/server.go`. Each frame prefixed with 4-byte big-endian length. Used for binary audio PCM payloads between supervisor and audio workers.
**Where**: `internal/ipc/jsonrpc/server.go` (modify), `internal/ipc/jsonrpc/unix_test.go` (new)
**Depends on**: T6
**Reuses**: `net.UnixListener`; existing JSON-RPC parser from T6.
**Requirement**: SUPR-08
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] Unix socket listener accepts connections on `/tmp/mymemoryd-<pid>.sock`
- [ ] Each frame: 4-byte big-endian length + JSON-RPC payload
- [ ] Binary payloads (audio PCM) preserved end-to-end with zero loss
- [ ] Test `TestUnixSocket_FrameDelimitsBinaryAudio`: 10 chunks × 512 bytes PCM, zero loss
- [ ] Test `TestUnixSocket_RejectsTruncatedFrame`: send 2 bytes instead of 4 → error

**Tests**: integration
**Gate**: full
**Commit**: `feat(ipc): Unix socket + 4-byte binary framing`

---

### Phase 4

### T8: CLI `mem up` / `mem down` / `mem status` / `mem logs` / `mem profiles`

**What**: Add CLI subcommands in `cmd/mem/`: `mem up [--profile <name>]` (spawns mymemoryd), `mem down` (graceful shutdown), `mem status [--json]` (worker health snapshot), `mem logs <worker>` (tail structured audit), `mem profiles list` / `mem profiles use <name>`.
**Where**: `cmd/mem/up.go`, `cmd/mem/down.go`, `cmd/mem/status.go`, `cmd/mem/logs.go`, `cmd/mem/profiles.go` (all new)
**Depends on**: T1
**Reuses**: `cobra` framework (already in use); existing CLI patterns from `cmd/mem/events.go`.
**Requirement**: SUPR-01, SUPR-05, SUPR-16, SUPR-17
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `mem up` spawns `mymemoryd` subprocess, waits for `READY` on stdout, exits 0
- [ ] `mem down` sends SIGTERM, waits 10s, SIGKILL, exits 0
- [ ] `mem status` prints table: worker_id, pid, status, uptime, last_heartbeat
- [ ] `mem status --json` outputs JSON parseable
- [ ] `mem logs <worker>` tails `.memory/logs/supervisor.jsonl` filtered by worker_id
- [ ] `mem profiles list` prints all profiles + effective worker count
- [ ] `mem profiles use <name>` writes `.memory/config.yaml: active_profile`
- [ ] `mem up` when already running → exit 0 with "Already running on PID <N>"
- [ ] Test `TestCliUp_StartsAndStops`: `mem up` then `mem down` → supervisor exits 0
- [ ] Test `TestCliUp_AlreadyRunning`: 2× `mem up` → second exit 0 with message
- [ ] Test `TestCliProfiles_ListAndUse`

**Tests**: integration
**Gate**: full
**Commit**: `feat(cli): mem up/down/status/logs/profiles`

---

### T9: Sandbox best-effort + audit log JSONL

**What**: Create `internal/supervisor/sandbox.go` applying best-effort sandbox to worker subprocess: `setrlimit(NOFILE)` cap, `prctl(PR_SET_NO_NEW_PRIVS)` on Linux, no-op warning elsewhere. Create `internal/supervisor/audit.go` writing structured JSONL to `.memory/logs/supervisor.jsonl` for every lifecycle event. Rotation: 100 MiB size OR 7 days age → gzip to `.memory/logs/archive/`.
**Where**: `internal/supervisor/sandbox.go` (new), `internal/supervisor/audit.go` (new), `internal/supervisor/sandbox_test.go` (new), `internal/supervisor/audit_test.go` (new)
**Depends on**: T2, T5
**Reuses**: `syscall.Setrlimit`, `syscall.Prctl` (Linux); `encoding/json` for JSONL.
**Requirement**: SUPR-18, SUPR-19, SUPR-20, SUPR-21, SUPR-22, SUPR-23
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `sandbox.Apply(cmd *exec.Cmd)` sets `NOFILE` rlimit + `PR_SET_NO_NEW_PRIVS` on Linux
- [ ] macOS/Windows: no-op + warning logged
- [ ] `egress: deny` worker has no network access (Unix socket / stdio only)
- [ ] `egress: allow` worker with allowlist config can open TCP
- [ ] `audit.Log(workerID, event, payload)` writes JSONL entry to `.memory/logs/supervisor.jsonl`
- [ ] Audit entries include `worker_id`, `pid`, `ts`, `event`, `payload`
- [ ] Audit log rotates at 100 MiB or 7 days; old files gzipped
- [ ] Audit write failure → emits `audit.write_failed` event but continues
- [ ] Test `TestSandbox_LinuxAppliesPrctlNoNewPrivs`: Linux runner sets PR_SET_NO_NEW_PRIVS
- [ ] Test `TestSandbox_EgressDenyBlocksTcp`: deny worker fails on `net.Dial("tcp", ...)`
- [ ] Test `TestAudit_LogsAllLifecycleEvents`: 100 start/stop → 200+ entries in JSONL
- [ ] Test `TestAudit_RotatesAtSizeLimit`: write 100 MiB → rotation triggered

**Tests**: integration
**Gate**: full
**Commit**: `feat(supervisor): sandbox best-effort + audit log JSONL with rotation`
