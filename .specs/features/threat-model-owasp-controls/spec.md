# Feature: threat-model-owasp-controls

> **Status**: Specify phase — design document is [ADR-050](../../../docs/adr/050-threat-model-owasp-llm-aplicado-ao-mymemory.md); this spec captures WHAT (testable controls) for the OWASP LLM Top 10 mapping applied across features.

## Problem Statement

A partir do momento em que MyMemory ganhar voz, agente runtime, MCP server autoral e workers Python sidecar, a superfície de ataque deixa de ser "arquivo Markdown local" e passa a incluir áudio hostil, transcrições parciais, saídas de LLM, tool results, embeddings externos e plugins de terceiros. O [ADR-050](../../../docs/adr/050-threat-model-owasp-llm-aplicado-ao-mymemory.md) mapeia o OWASP LLM Top 10 para controles concretos no MyMemory e define postura deny-by-default / fail-closed / audit-tudo.

Falta o **WHAT testável**: como cada controle LLM01-LLM10 é implementado, validado por teste adversarial (garak ou probe manual), e auditado. Sem este spec, voz/agente/MCP entram sem âncora de segurança — exatamente quando mais precisam.

## Goals

- [ ] Adicionar campo `provenance.trust_level` (enum: `owner`, `trusted`, `unverified`, `untrusted`) no envelope de eventos (ADR-043) — small refactor em `internal/event_runtime/envelope.go`.
- [ ] `internal/policy/engine.go` carregando `.memory/policy/{strict,balanced,permissive-dev}.yaml` com allowlist de actors, scopes de curta duração, risk thresholds.
- [ ] Tool gateway (`internal/tool/gateway.go`) validando todo tool call antes de execução: JSON Schema com `additionalProperties: false`, `SafeResolvePath`, size limit, namespace check (`memory.*`, `session.*`, `agent.*`, `approval.*`).
- [ ] Audit subscriber (`internal/audit/`) já existente via event_runtime, ampliado para incluir `policy.decision_id`, `redacted_payload_hash`, `provenance.trust_level`.
- [ ] Redaction LLM02 (`internal/redaction/`) aplicada automaticamente a `payload.transcript`, `payload.llm_output`, `payload.tool_result` antes de `event_log.Append`.
- [ ] SBOM + hash check pra modelos ML: `internal/model/integrity.go` valida SHA-256 do modelo no startup contra `.memory/models/<name>.sha256`; falha = `ErrModelIntegrity`.
- [ ] CSP strict no viewer: `default-src 'none'; img-src 'self' data:;` + Markdown sanitizado (sem `<script>`, `<iframe>`, `<img src="http...">`).
- [ ] Loop limiter (`internal/agent/loop_limiter.go`): max steps, max tokens, max wall-clock por `run_id`.
- [ ] Garak probes adaptados pra LLM01 (prompt injection), LLM02 (info disclosure), LLM06 (excessive agency), LLM09 (misinformation) — viram parte do quality gate F5+.
- [ ] `mem security audit` subcommand: roda SBOM + hash check + probes básicos + reporta findings.
- [ ] Cobrir invariantes: `TestLlm01_PromptInjectionRejected`, `TestLlm02_TranscriptRedactedBeforeLog`, `TestLlm06_ToolGatewayBlocksUnsafeArgs`, `TestLlm09_CitationsRequired`, `TestLlm10_LoopLimiterHaltsRunaway`.

## Out of Scope

| Feature | Reason |
| --- | --- |
| Implementação específica de cada worker Python (ASR, TTS) | Workers vêm de specs separadas; esta spec entrega controles transversais |
| Runtime sandbox completo (gVisor, Firecracker) | ADR-042 §DR-5 é best-effort; sandbox perfeito é F5+ |
| Adversarial training do LLM | Não é nosso LLM; Ollama/Piper são tools externas |
| SIEM integration (Splunk, ELK) | F5+; por ora JSONL + `jq` |
| Compliance certifications (SOC2, ISO27001) | F5+; controles estão definidos, certificações são processo |
| Bug bounty program | F5+; fora do escopo técnico |
| Code signing de releases | ADR-019 cobre signing; segurança de supply chain separada |
| Encryption at rest | ADR-050 §Considered — LLM02 via redaction; SQLCipher é mudança de stack, defer |

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --- | --- | --- | --- |
| Default policy | `balanced` | ADR-050 §Negative Consequences — 3 profiles dá tunability | y |
| `provenance.trust_level` no envelope | Adicionar campo no ADR-043 com backward compat (default `unverified` se ausente) | Refactor pequeno, sem quebrar callers existentes | y |
| Tool gateway namespace | Allowlist: `memory.*`, `session.*`, `agent.*`, `approval.*` | ADR-050 §MCP table — namespaces determinísticos | y |
| Redaction strategy | PII detection via regex (email, telefone, CPF, CNPJ, cartão de crédito) + LLM opt-in pra NER | MVP sem LLM dependency; regex cobre 80% dos casos | n (decidir se adicionar NER em F2 ou F5) |
| Audit log retention | 7 dias rolling, gzip após 24h, JSONL format | ADR-050 §Redaction — retenção configurável default conservador | y |
| Garak probe scope | F5+ — probes viram parte do quality gate nightly; bloqueia `promote`, não bloqueia `commit` | ADR-050 §Implementation Notes — balanceia segurança e velocity | y |
| Loop limiter defaults | max_steps=20, max_tokens=8000, max_wall_clock=120s | Limites conservadores; tuning via `.memory/policy/*.yaml` | y |
| CSP enforcement location | Middleware HTTP no viewer; markdown sanitization antes do render | ADR-050 §Viewer — defense in depth | y |
| SBOM generation | `go.mod` + `go list -m -json all` na release; CycloneDX format | Go ecosystem standard; tooling maduro | y |
| Model integrity check | SHA-256 em `.memory/models/<name>.sha256`; falha = refuse to load | ADR-050 §Embedder row — SBOM + hash | y |
| Egress deny implementation | Best-effort via Linux network namespace + macOS sandbox-exec fallback | ADR-042 §DR-5; perfeito cross-platform não é viável | y |
| Approval flow | Tool call de mutação → `approval.requested` event com TTL 30min; sem `approval.granted` = no-op | ADR-050 §LLM06 — policy engine fora do LLM | y |
| Security audit retention | Findings mantidos por 30 dias; resolved findings mantidos por 1 ano pra auditoria | Compliance-friendly sem ser pesado | y |

**Open questions:** 1 — redaction strategy: regex-only pra MVP ou adicionar NER LLM-based em F2? Resolver antes da T1.

---

## User Stories

### P1: Trust Level + Redaction LLM02 ⭐ MVP

**User Story**: As a operador, I want cada envelope carregando `provenance.trust_level` e todo `payload.transcript`/`llm_output` redacted antes do `event_log.Append` so that conteúdo sensível nunca vaza via log ou replay.

**Why P1**: LLM02 (Sensitive Information Disclosure) é o vetor mais crítico — PII em áudio/transcript é default no caso de uso de voz. Sem redaction, replay vira leak de PII.

**Acceptance Criteria**:

1. WHERE the envelope has `payload.transcript` THEN the system SHALL apply `internal/redaction.Apply()` before `event_log.Append`, replacing detected PII with `[REDACTED:<type>]`. (optional-feature)
2. WHERE the envelope has `payload.llm_output` THEN the system SHALL apply the same redaction. (optional-feature)
3. The system SHALL detect and redact: email, phone (BR + international), CPF, CNPJ, credit card (Luhn-validated), and JWT-shaped strings. (ubiquitous)
4. The system SHALL compute `payload.redacted_payload_hash = sha256(redacted_content)` and store alongside `payload` for forensics. (ubiquitous)
5. IF redaction regex compilation fails at startup THEN the system SHALL fail-closed (ADR-050 §DR-2). (unwanted-behavior)
6. The system SHALL NEVER log the unredacted payload to stdout/stderr, even in debug mode. (ubiquitous)
7. The system SHALL add `provenance.trust_level` field to envelope (enum: `owner`, `trusted`, `unverified`, `untrusted`) with default `unverified`. (ubiquitous)

**Independent Test**: `TestLlm02_TranscriptRedactedBeforeLog` injeta transcript com CPF+email, valida `event_log.payload` contém `[REDACTED:cpf]` mas audit log NÃO contém raw; `TestLlm02_NeverLogsUnredacted` força debug mode e verifica stdout limpo.

---

### P2: Tool Gateway LLM05/LLM06 ⭐ MVP

**User Story**: As a agente runtime, I want cada tool call validado pelo `internal/tool/gateway.go` antes de executar so that path traversal, size overflow, e argumentos adversariais sejam bloqueados antes de afetar estado.

**Why P2**: LLM05 (Improper Output Handling) e LLM06 (Excessive Agency) são os vetores mais exploráveis — tool calls são onde o agente toca o mundo externo.

**Acceptance Criteria**:

1. WHEN `tool_gateway.Invoke(name, args)` is called THEN the system SHALL validate `name` against allowlist `memory.*|session.*|agent.*|approval.*` and reject otherwise with `ErrToolNotInAllowlist`. (unwanted-behavior)
2. The system SHALL validate `args` against the tool's JSON Schema with `additionalProperties: false`; unknown fields rejected with `ErrInvalidToolArgs`. (unwanted-behavior)
3. The system SHALL apply `SafeResolvePath` (ADR-018) to any path arg; traversal attempts (`../`, absolute paths outside vault) rejected with `ErrPathTraversal`. (unwanted-behavior)
4. The system SHALL enforce size limit per arg (default 1 MiB) and per total request (default 5 MiB); oversized rejected with `ErrToolArgTooLarge`. (unwanted-behavior)
5. WHERE the tool mutates state (write, exec, network) THEN the system SHALL require `approval.granted` event before executing. (optional-feature)
6. The system SHALL log every tool invocation to audit with `tool.name`, `tool.args_hash`, `tool.result_hash`, `policy.decision_id`. (ubiquitous)
7. IF validation fails THEN the system SHALL return error envelope AND emit `tool.rejected` event with `payload.reason`. (unwanted-behavior)

**Independent Test**: `TestLlm06_ToolGatewayBlocksUnsafeArgs` tenta path traversal `../../etc/passwd`, valida rejeição; `TestLlm06_RequiresApprovalForMutations` valida que tool de write precisa `approval.granted`.

---

### P3: Loop Limiter + Rate Limit LLM10 ⭐ MVP

**User Story**: As a operador, I want cada `agent.run` ter limite máximo de steps, tokens, e wall-clock so that loops infinitos ou DoS não derrubem o sistema.

**Why P3**: LLM10 (Unbounded Consumption) é o vetor mais fácil de explorar — basta um prompt que confunda o agente em loop.

**Acceptance Criteria**:

1. WHEN `agent.run` starts THEN the system SHALL initialize counters: `steps_used`, `tokens_used`, `wall_clock_started`. (event-driven)
2. WHEN any counter exceeds configured threshold THEN the system SHALL halt the run with `agent.halted` event and return error envelope with `reason: "loop_limit_exceeded"`. (unwanted-behavior)
3. The system SHALL enforce rate limit per `session_id` (default 100 requests/minute) — over-limit requests rejected with `429 Too Many Requests` equivalent. (state-driven)
4. The system SHALL enforce `MaxAckPending` per subscriber (default 64, ADR-043 §6) to prevent event_log exhaustion. (ubiquitous)
5. The system SHALL emit `agent.throttled` event when rate limit triggers, with `session_id`, `count`, `limit`. (event-driven)
6. Loop limiter thresholds SHALL be configurable via `.memory/policy/*.yaml` (`max_steps`, `max_tokens`, `max_wall_clock_seconds`). (optional-feature)

**Independent Test**: `TestLlm10_LoopLimiterHaltsRunaway` força agente a tentar step 25 (limite 20), valida halt e `agent.halted` event; `TestLlm10_RateLimitPerSession` força 101 requests em 60s, valida 429 no request 101.

---

### P4: Model Integrity LLM08 + Viewer CSP LLM05

**User Story**: As a operador, I want modelos ML verificados por SHA-256 no startup e viewer com CSP strict + Markdown sanitizado so that modelo poisoned ou Markdown hostil não cheguem ao usuário.

**Why P4**: LLM08 (Vector and Embedding Weaknesses) e viewer-side LLM05 são vetores de supply chain e content injection.

**Acceptance Criteria**:

1. WHEN model is loaded THEN the system SHALL compute SHA-256 and compare against `.memory/models/<name>.sha256`; mismatch → `ErrModelIntegrity` and refuse to load. (unwanted-behavior)
2. The system SHALL generate `.memory/models/<name>.sha256` on first download via `mem asr download` or `mem model install`. (event-driven)
3. WHERE the viewer is served via HTTP THEN the system SHALL set CSP header: `default-src 'none'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self';`. (optional-feature)
4. WHERE Markdown is rendered THEN the system SHALL sanitize: strip `<script>`, `<iframe>`, `<object>`, `<embed>`, event handlers (`onclick` etc.), and `javascript:` URLs. (optional-feature)
5. The system SHALL reject `<img src="http...">` (remote images) — only `self` and `data:` URIs allowed. (unwanted-behavior)
6. The system SHALL log every model load attempt (success or fail) to audit with `model.name`, `model.sha256`, `model.size_bytes`. (ubiquitous)

**Independent Test**: `TestLlm08_ModelIntegrityMismatch` modifica modelo em disco, valida `ErrModelIntegrity` no próximo load; `TestLlm05_ViewerBlocksRemoteImage` renderiza Markdown com `<img src="http://evil.com/x.png">`, valida que src é stripped.

---

### P5: Garak Probe Integration + Security Audit Subcommand

**User Story**: As a operador, I want rodar `mem security audit` e a ferramenta executar probes adaptados do garak contra os vetores LLM01/LLM02/LLM06/LLM09 + SBOM check + reportar findings so that tenho sinal contínuo de postura de segurança.

**Why P5**: ADR-050 §Implementation Notes — `make security-probe` é parte do quality gate em F5+.

**Acceptance Criteria**:

1. WHEN `mem security audit` is invoked THEN the system SHALL run probes in order: SBOM check → model integrity → tool gateway fuzz → redaction fuzz → citations check → loop limit stress. (event-driven)
2. The system SHALL report findings as JSON with `severity` (`critical|high|medium|low|info`), `vector` (LLM01..10), `description`, `remediation_hint`. (ubiquitous)
3. The system SHALL fail-closed (exit 3) if any `critical` or `high` finding is detected. (unwanted-behavior)
4. The system SHALL write report to `.memory/logs/security-audit-<ts>.json`. (event-driven)
5. WHERE `--ci` flag is passed THEN the system SHALL use compact output (one line per finding) suitable for CI logs. (optional-feature)
6. The system SHALL support `--vector LLM02` flag to scope the audit to specific OWASP vectors. (optional-feature)

**Independent Test**: `TestLlm01_PromptInjectionRejected` injeta prompt com `<system>` override, valida rejeição; `TestLlm09_CitationsRequired` valida tool result sem `source_document` é rejeitado pelo agente.

---

## Edge Cases

- IF `provenance.trust_level` is unknown enum THEN system rejects envelope with `ErrInvalidTrustLevel`. (unwanted-behavior)
- WHEN audit log write fails repeatedly (3x) THEN system SHALL emit `audit.degraded` event and continue in degraded mode. (state-driven)
- IF `.memory/policy/*.yaml` is missing THEN system SHALL default to `balanced` and warn at startup. (unwanted-behavior)
- WHEN viewer receives Markdown with both inline image and remote URL THEN system SHALL keep inline (`data:` or `self`) and strip remote. (unwanted-behavior)
- IF loop limiter counter overflows int64 THEN system SHALL halt run defensively and emit `loop_limit.overflow` event. (unwanted-behavior)
- WHEN CSP header conflicts with embedded styles THEN strict CSP wins (defense in depth). (state-driven)
- IF `mem security audit` runs out of memory during probe THEN system SHALL write partial report and exit 4 (OOM). (unwanted-behavior)

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| --- | --- | --- | --- |
| SEC-01 | P1 | Design | Pending |
| SEC-02 | P1 | Design | Pending |
| SEC-03 | P1 | Design | Pending |
| SEC-04 | P1 | Design | Pending |
| SEC-05 | P1 | Design | Pending |
| SEC-06 | P1 | Design | Pending |
| SEC-07 | P1 | Design | Pending |
| SEC-08 | P2 | Design | Pending |
| SEC-09 | P2 | Design | Pending |
| SEC-10 | P2 | Design | Pending |
| SEC-11 | P2 | Design | Pending |
| SEC-12 | P2 | Design | Pending |
| SEC-13 | P2 | Design | Pending |
| SEC-14 | P2 | Design | Pending |
| SEC-15 | P3 | Design | Pending |
| SEC-16 | P3 | Design | Pending |
| SEC-17 | P3 | Design | Pending |
| SEC-18 | P3 | Design | Pending |
| SEC-19 | P3 | Design | Pending |
| SEC-20 | P3 | Design | Pending |
| SEC-21 | P4 | Design | Pending |
| SEC-22 | P4 | Design | Pending |
| SEC-23 | P4 | Design | Pending |
| SEC-24 | P4 | Design | Pending |
| SEC-25 | P4 | Design | Pending |
| SEC-26 | P4 | Design | Pending |
| SEC-27 | P5 | Design | Pending |
| SEC-28 | P5 | Design | Pending |
| SEC-29 | P5 | Design | Pending |
| SEC-30 | P5 | Design | Pending |
| SEC-31 | P5 | Design | Pending |
| SEC-32 | P5 | Design | Pending |

**Coverage:** 32 total, 0 mapped to tasks, 32 unmapped ⚠️ (tasks.md pending).

---

## Success Criteria

- [ ] Redaction LLM02 applied: transcript with PII never reaches `event_log.payload` raw (validated by `TestLlm02_TranscriptRedactedBeforeLog`).
- [ ] Tool gateway blocks path traversal and unauthorized tool namespaces (validated by `TestLlm06_ToolGatewayBlocksUnsafeArgs`).
- [ ] Loop limiter halts runaway runs at configured threshold (validated by `TestLlm10_LoopLimiterHaltsRunaway`).
- [ ] Model integrity check refuses to load tampered models (validated by `TestLlm08_ModelIntegrityMismatch`).
- [ ] Viewer CSP blocks remote images and inline scripts (validated by `TestLlm05_ViewerBlocksRemoteImage`).
- [ ] `mem security audit --ci` exits 0 if no critical/high findings; exits 3 otherwise.
- [ ] All redaction rules have at least 1 positive test (PII detected) and 1 negative test (clean text untouched).
- [ ] Garak probe adapted for LLM01/LLM02/LLM06/LLM09 runs in CI nightly without false positives.
- [ ] `go test -count=1 ./internal/redaction/... ./internal/tool/... ./internal/agent/... ./internal/policy/... ./internal/model/...` passes 100%.
