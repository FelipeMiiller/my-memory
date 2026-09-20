# threat-model-owasp-controls Tasks

## Execution Protocol (MANDATORY — do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path.

**If the skill cannot be activated, STOP and tell the user — do not proceed without it.**

---

**Spec**: `.specs/features/threat-model-owasp-controls/spec.md`
**Design**: [ADR-050](../../../docs/adr/050-threat-model-owasp-llm-aplicado-ao-mymemory.md)
**Status**: Draft → Approved → In Progress → Done

---

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| --- | --- | --- | --- | --- |
| Service (redaction, tool gateway, loop limiter) | unit + integration | All branches; PII detection + LLM05/LLM06/LLM10 controls | `internal/redaction/*_test.go`, `internal/tool/*_test.go`, `internal/agent/*_test.go` | `go test -count=1 ./internal/redaction/... ./internal/tool/... ./internal/agent/...` |
| Domain (TrustLevel, RedactionRule, ToolSpec) | unit | Enum parsing; regex compilation | `internal/redaction/types_test.go` | `go test -count=1 ./internal/redaction/...` |
| Model integrity (SHA-256 verify) | integration | Mismatch rejection | `internal/model/integrity_test.go` | `go test -count=1 ./internal/model/...` |
| Viewer CSP + Markdown sanitizer | integration | CSP header; remote image blocking; script stripping | `internal/viewer/csp_test.go` | `go test -count=1 ./internal/viewer/...` |
| CLI (`mem security audit`) | integration | Probe ordering + JSON output | `cmd/mem/security_audit_test.go` | `go test -count=1 ./cmd/mem/...` |

---

## Gate Check Commands

| Gate Level | When to Use | Command |
| --- | --- | --- |
| Quick | After unit-only tasks (T2, T4) | `go test -count=1 -short ./internal/redaction/... ./internal/tool/...` |
| Full | After integration tasks (T1, T3, T5, T6, T7, T8, T9, T10) | `go test -count=1 ./internal/redaction/... ./internal/tool/... ./internal/agent/... ./internal/model/... ./internal/viewer/... ./cmd/mem/...` |
| Build | After phase completion | `gofmt -l . && go build ./... && go test -count=1 ./...` |

---

## Execution Plan

### Phase 1: Trust level + redaction LLM02

PII detection + redaction before event_log.Append + provenance.trust_level.

```
T1 -> T2
```

### Phase 2: Tool gateway LLM05/LLM06 + Loop limiter LLM10

Tool call validation, approval flow, runaway protection.

```
T3 -> T4
T4 -> T5
T5 -> T6
```

### Phase 3: Model integrity + Viewer CSP

SHA-256 verification + CSP strict + Markdown sanitization.

```
T7 -> T8
```

### Phase 4: Security audit subcommand + garak probe integration

`mem security audit` CLI + adapted garak probes.

```
T8 -> T10
```

Total: **10 tasks across 4 phases**. Fits 2 batches at Execute time.

---

## Task Breakdown

### Phase 1

### T1: Provenance.trust_level + envelope extension

**What**: Extend `event_runtime.Envelope` with `Provenance.TrustLevel` field (enum: `owner|trusted|unverified|untrusted`). Default to `unverified` when caller doesn't set. Update event_runtime tests to validate backward compat (existing envelopes still parse).
**Where**: `internal/event_runtime/envelope.go` (modify), `internal/event_runtime/envelope_test.go` (modify)
**Depends on**: None
**Reuses**: Existing envelope struct from event-runtime-event-log.
**Requirement**: SEC-07
**Status**: Pending

**Done when**:

- [ ] `Envelope.Provenance.TrustLevel` exported enum field
- [ ] Default `unverified` applied when not set
- [ ] Round-trip JSON: marshal → unmarshal preserves TrustLevel
- [ ] Reject unknown enum values with `ErrInvalidTrustLevel`
- [ ] Test `TestEnvelope_TrustLevelRoundtrip`
- [ ] Test `TestEnvelope_UnknownTrustLevel_Rejected`
- [ ] Backward compat: existing test fixtures still parse

**Tests**: unit
**Gate**: quick
**Commit**: `feat(event-runtime): add Provenance.TrustLevel to envelope`

---

### T2: Redaction LLM02 (PII detection + redaction)

**What**: Create `internal/redaction/` package with `Apply(ctx, content []byte) (redacted []byte, hash string, error)` detecting and redacting: email, phone (BR + international), CPF, CNPJ, credit card (Luhn-validated), JWT-shaped strings. Replace with `[REDACTED:<type>]`. Compute SHA-256 of redacted content. Compile regexes at startup; fail-closed on compilation error. NEVER log raw payload to stdout/stderr (validated by debug-mode test).
**Where**: `internal/redaction/redaction.go` (new), `internal/redaction/redaction_test.go` (new), `internal/redaction/pii_test.go` (new)
**Depends on**: T1
**Reuses**: `regexp` stdlib.
**Requirement**: SEC-01, SEC-02, SEC-03, SEC-04, SEC-05, SEC-06
**Status**: Pending

**Done when**:

- [ ] `Apply` detects: email, phone (BR + intl), CPF, CNPJ, credit card (Luhn), JWT
- [ ] Replaces matches with `[REDACTED:<type>]`
- [ ] Returns redacted bytes + SHA-256 hash of redacted content
- [ ] Regex compilation failure at startup → fail-closed
- [ ] No raw payload ever written to stdout/stderr (debug-mode test asserts)
- [ ] Positive tests: 1 per PII type (PII detected)
- [ ] Negative tests: 1 per PII type (clean text untouched)
- [ ] Luhn validation: invalid card numbers NOT redacted
- [ ] Test `TestRedaction_TranscriptWithCPF_Email_Redacted`
- [ ] Test `TestRedaction_NeverLogsUnredacted` forces debug mode + checks stdout

**Tests**: unit + integration
**Gate**: full
**Commit**: `feat(redaction): LLM02 PII detection + redaction before event_log.Append`

---

### Phase 2

### T3: Tool gateway skeleton + namespace allowlist

**What**: Create `internal/tool/gateway.go` with `Gateway` struct. `Invoke(ctx, name string, args json.RawMessage) (Result, error)`. Validates `name` against allowlist `memory.*|session.*|agent.*|approval.*`. Rejects others with `ErrToolNotInAllowlist`. Validates `args` against tool's JSON Schema with `additionalProperties: false`.
**Where**: `internal/tool/gateway.go` (new), `internal/tool/gateway_test.go` (new)
**Depends on**: None
**Reuses**: JSON Schema validation via `github.com/xeipuuv/gojsonschema`.
**Requirement**: SEC-08, SEC-09
**Status**: Pending

**Done when**:

- [ ] `Gateway.Invoke` validates tool name against allowlist regex
- [ ] Tool name not in allowlist → `ErrToolNotInAllowlist`
- [ ] Tool args validated against JSON Schema with `additionalProperties: false`
- [ ] Unknown fields in args → `ErrInvalidToolArgs`
- [ ] Logs every invocation to audit with `tool.name`, `tool.args_hash`, `policy.decision_id`
- [ ] Test `TestToolGateway_RejectsToolNotInAllowlist`
- [ ] Test `TestToolGateway_RejectsAdditionalProperties`

**Tests**: unit
**Gate**: quick
**Commit**: `feat(tool): gateway with namespace allowlist + JSON Schema validation`

---

### T4: SafeResolvePath + size limits (LLM05)

**What**: Add to `Gateway.Invoke`: `SafeResolvePath` on any path arg (reject `../`, absolute paths outside vault). Enforce per-arg size limit (default 1 MiB) + per-request limit (default 5 MiB). Oversized → `ErrToolArgTooLarge`.
**Where**: `internal/tool/gateway.go` (modify), `internal/tool/path_test.go` (new)
**Depends on**: T3
**Reuses**: Existing `SafeResolvePath` from ADR-018 (internal/compiler or wherever it lives).
**Requirement**: SEC-10, SEC-11
**Status**: Pending

**Done when**:

- [ ] Path arg `../etc/passwd` → `ErrPathTraversal`
- [ ] Absolute path outside vault → `ErrPathTraversal`
- [ ] Per-arg > 1 MiB → `ErrToolArgTooLarge`
- [ ] Per-request > 5 MiB → `ErrToolArgTooLarge`
- [ ] Test `TestSafeResolvePath_RejectsTraversal`
- [ ] Test `TestSafeResolvePath_RejectsAbsoluteOutsideVault`
- [ ] Test `TestSizeLimits_PerArg`
- [ ] Test `TestSizeLimits_PerRequest`

**Tests**: unit
**Gate**: quick
**Commit**: `feat(tool): SafeResolvePath + size limits (LLM05)`

---

### T5: Approval flow for mutations (LLM06)

**What**: Extend `Gateway.Invoke`: tools that mutate state (write, exec, network) require `approval.granted` event before execution. Emit `approval.requested` event with `payload.tool_name`, `payload.tool_args_hash`, TTL 30min. Without grant → `ErrApprovalRequired`. Audit log includes `policy.decision_id`.
**Where**: `internal/tool/gateway.go` (modify), `internal/tool/approval_test.go` (new)
**Depends on**: T4
**Reuses**: `event_runtime.Log.Append` for approval events.
**Requirement**: SEC-12, SEC-13
**Status**: Pending

**Done when**:

- [ ] Tool marked `mutates_state: true` in spec → `approval.requested` event emitted
- [ ] Without `approval.granted` within 30min → `ErrApprovalRequired`
- [ ] `approval.granted` → tool executes
- [ ] Audit log entry includes `policy.decision_id`
- [ ] Test `TestApproval_MutationToolRequiresGrant`
- [ ] Test `TestApproval_TimesOutAfter30Min`
- [ ] Test `TestApproval_GrantedExecutesTool`

**Tests**: integration
**Gate**: full
**Commit**: `feat(tool): approval flow for mutating tool calls (LLM06)`

---

### T6: Loop limiter + rate limit (LLM10)

**What**: Create `internal/agent/loop_limiter.go` with `Limiter` struct tracking `steps_used`, `tokens_used`, `wall_clock_started` per `run_id`. Configurable thresholds: `max_steps` (default 20), `max_tokens` (default 8000), `max_wall_clock_seconds` (default 120). On threshold exceeded → halt with `agent.halted` event. Also add rate limit per `session_id` (default 100 req/min).
**Where**: `internal/agent/loop_limiter.go` (new), `internal/agent/loop_limiter_test.go` (new), `internal/agent/rate_limit_test.go` (new)
**Depends on**: T5
**Reuses**: Token counting via `tiktoken-go` (or simpler word count for MVP).
**Requirement**: SEC-15, SEC-16, SEC-17, SEC-18, SEC-19, SEC-20
**Status**: Pending

**Done when**:

- [ ] `Limiter.Check(runID)` returns nil if all counters below thresholds
- [ ] Counter exceeded → `agent.halted` event + error with `reason: "loop_limit_exceeded"`
- [ ] Rate limit: 101st request in 60s → `429 Too Many Requests`
- [ ] `agent.throttled` event emitted on rate limit trigger
- [ ] Thresholds configurable via `.memory/policy/*.yaml`
- [ ] Test `TestLoopLimiter_HaltsRunawayRun` forces 25 steps (limit 20)
- [ ] Test `TestRateLimit_PerSession` forces 101 requests in 60s

**Tests**: unit + integration
**Gate**: full
**Commit**: `feat(agent): loop limiter + rate limit (LLM10)`

---

### Phase 3

### T7: Model integrity (SHA-256 verify)

**What**: Create `internal/model/integrity.go` with `Verify(ctx, modelPath, expectedHashPath) error`. Computes SHA-256 of model file, compares against `.memory/models/<name>.sha256`. Mismatch → `ErrModelIntegrity` and refuse to load. Generates `.sha256` on first successful download (called by `mem asr download` from nemotron spec).
**Where**: `internal/model/integrity.go` (new), `internal/model/integrity_test.go` (new)
**Depends on**: None
**Reuses**: `crypto/sha256`.
**Requirement**: SEC-21, SEC-22, SEC-26
**Status**: Pending

**Done when**:

- [ ] `Verify` returns nil on SHA-256 match
- [ ] Returns `ErrModelIntegrity` on mismatch
- [ ] Returns nil + generates `.sha256` if hash file missing (first-time setup)
- [ ] Test `TestIntegrity_Mismatch_Rejects`
- [ ] Test `TestIntegrity_FirstTime_GeneratesHash`

**Tests**: integration
**Gate**: full
**Commit**: `feat(model): SHA-256 integrity verify on load`

---

### T8: Audit log structured (every model load attempt)

**What**: Create `internal/audit/` writing JSONL to `.memory/logs/security-audit.jsonl`. Records every model load attempt (success/fail) with `model.name`, `model.sha256`, `model.size_bytes`, `actor`, `ts`. Log rotation: 100 MiB or 7 days. Audit write failure → `audit.degraded` event.
**Where**: `internal/audit/audit.go` (new), `internal/audit/audit_test.go` (new)
**Depends on**: T7
**Reuses**: `encoding/json` for JSONL; existing rotation pattern from mymemoryd-supervisor T9.
**Requirement**: SEC-26
**Status**: Pending

**Done when**:

- [ ] `audit.Log(event, payload)` writes JSONL entry
- [ ] Entries include all required fields
- [ ] Rotation at 100 MiB or 7 days
- [ ] Write failure → `audit.degraded` event, continues
- [ ] Test `TestAudit_LogsModelLoadAttempts`
- [ ] Test `TestAudit_RotatesAtSizeLimit`

**Tests**: integration
**Gate**: full
**Commit**: `feat(audit): structured JSONL for model load attempts + rotation`

---

### T9: Viewer CSP strict + Markdown sanitizer

**What**: Create `internal/viewer/csp.go` setting CSP header: `default-src 'none'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self';`. Create `internal/viewer/sanitize.go` stripping `<script>`, `<iframe>`, `<object>`, `<embed>`, event handlers, `javascript:` URLs, remote `<img src="http...">`.
**Where**: `internal/viewer/csp.go` (new), `internal/viewer/sanitize.go` (new), `internal/viewer/sanitize_test.go` (new)
**Depends on**: None
**Reuses**: `html/template` for safe rendering; `golang.org/x/net/html` for parsing.
**Requirement**: SEC-23, SEC-24, SEC-25
**Status**: Pending

**Done when**:

- [ ] CSP header set on every HTTP response from viewer
- [ ] Markdown sanitizer strips `<script>`, `<iframe>`, event handlers, `javascript:`
- [ ] `<img src="http://evil.com/x.png">` → src stripped
- [ ] `<img src="data:image/png;base64,...">` preserved
- [ ] Test `TestCSP_HeaderSetOnResponse`
- [ ] Test `TestSanitize_StripsScriptTag`
- [ ] Test `TestSanitize_StripsRemoteImage`
- [ ] Test `TestSanitize_PreservesDataImage`

**Tests**: integration
**Gate**: full
**Commit**: `feat(viewer): CSP strict + Markdown sanitizer (LLM05)`

---

### Phase 4

### T10: `mem security audit` subcommand + garak probe integration

**What**: Add `cmd/mem/security_audit.go` with `mem security audit [--ci] [--vector LLM0X]`. Runs probes in order: SBOM check → model integrity → tool gateway fuzz → redaction fuzz → citations check → loop limit stress. Output JSON with `severity`, `vector`, `description`, `remediation_hint`. Exit 3 on critical/high findings.
**Where**: `cmd/mem/security_audit.go` (new), `cmd/mem/security_audit_test.go` (new), `internal/security/probe.go` (new)
**Depends on**: T8
**Reuses**: All previous tasks (T1-T9) as probe targets.
**Requirement**: SEC-27, SEC-28, SEC-29, SEC-30, SEC-31, SEC-32
**Status**: Pending

**Done when**:

- [ ] `mem security audit` runs all probes in order
- [ ] JSON output includes severity, vector, description, remediation
- [ ] Exit 3 on critical/high findings
- [ ] `--ci` flag → compact one-line-per-finding output
- [ ] `--vector LLM02` scopes to single OWASP vector
- [ ] Report written to `.memory/logs/security-audit-<ts>.json`
- [ ] Test `TestAudit_RunsAllProbes`
- [ ] Test `TestAudit_FailsClosedOnCriticalFinding`
- [ ] Test `TestAudit_CiCompactOutput`
- [ ] Test `TestAudit_VectorScoped`

**Tests**: integration
**Gate**: full
**Commit**: `feat(cli): mem security audit + garak probe integration`
