# Feature: event-runtime-event-log

> **Status**: Specify phase — design document is [ADR-043](../../../docs/adr/043-envelope-de-eventos-canonico-event-runtime.md); this spec captures WHAT (testable requirements + acceptance criteria), not the architectural discussion (already done).

## Problem Statement

O My-Memory hoje opera como CLI síncrono (`mem index`, `mem search`) e como servidor MCP sob demanda (ADR-021). Não existe um **barramento interno de eventos** que conecte captura de áudio, sessão conversacional, tool calls do agente, escrita de memória, projeções assíncronas, viewer de timeline e auditoria.

A pesquisa autoral de 2026-09-18 (§4.1 + §4.2) define um envelope JSON canônico (`event_id`, `schema_version`, `event_type`, `monotonic_ns`, `correlation_id`, `causation_id`, `actor`, `payload`) e lista os eventos de voz que o sistema precisa emitir. O ADR-043 transformou essa visão em decisão arquitetural. Falta o **WHAT testável**: o que exatamente o event_runtime faz, em que ordem, com que garantias, e como verificar.

Sem esse barramento, F2 (voz), F3 (agente) e F4 (MCP server autoral) vão reinventar canais de comunicação a cada subsistema — exatamente o que o ADR-043 quer evitar.

## Goals

- [ ] Criar tabela `event_log` em SQLite WAL com `sequence PRIMARY KEY AUTOINCREMENT`, `event_id UNIQUE`, `schema_version`, `event_type`, `payload BLOB`, `headers BLOB`, `aggregate_id`, `revision`, `created_at`, `acked_at`.
- [ ] Criar tabela `projection_cursor` para idempotência de subscribers (`projection_name PRIMARY KEY`, `last_sequence`, `updated_at`).
- [ ] Adicionar colunas `revision INTEGER DEFAULT 0` e `last_event_id TEXT` em `documents` (sem migração destrutiva; `ALTER TABLE ... ADD COLUMN`).
- [ ] Definir struct `event_runtime.Envelope` em Go com JSON canônico conforme ADR-043 §4.1.
- [ ] Implementar producer com outbox na mesma transação SQLite (atomicidade real entre commit e evento).
- [ ] Implementar dispatcher in-process com fan-out por `event_type`, ACK/NACK, retry com backoff exponencial capped em 30s e `MaxAckPending=64`.
- [ ] Expor CLI: `mem events tail`, `mem events inspect <event_id>`, `mem replay --since <seq>`, `mem trace <correlation_id>`.
- [ ] Cobrir invariantes com testes: `TestEventLog_OrderingIsCausal`, `TestReplay_ReproducesState`, `TestOutbox_AtomicWithEffect`.

## Out of Scope

| Feature | Reason |
| --- | --- |
| NATS JetStream / Redis Streams / ZeroMQ adapter | ADR-043 §3.9: defer para F5 (hardening), envelope já é o contrato. |
| Multi-node consensus / cluster mode | ADR-043 §3.6 + single-node MVP first; replay distribuído é F5+. |
| HTTP/SSE event bridge pra viewer remoto | ADR-043 §3.6 + ADR-021: MCP HTTP é default; bridge próprio só quando viewer remoto virar requisito. |
| WebSocket push pro viewer em tempo real | ADR-043 §3.6; viewer de timeline recebe via polling inicial (Fase 1). Push vem com viewer rewrite (ADR-037). |
| Schema versionamento dinâmico (v2+) | Schema_version=1 fixo nesta fase; migração formal de envelope é feature separada. |
| Projeções específicas (sqlite/fts/graph/vec) | ADR-044 é quem implementa as projeções concretas. Esta spec entrega apenas a interface `Subscriber` + 1 exemplo de subscriber (audit) para validar o dispatcher. |
| Compressão do payload (gzip/snappy) | Eventos típicos < 1 KB; fricção > ganho. Defer se profiling mostrar gargalo. |
| Criptografia em repouso do `event_log` | ADR-050 (LLM02): privacy por redaction no payload, não por criptografia. SQLCipher é mudança de stack; defer. |

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --- | --- | --- | --- |
| Persistência do event log | SQLite WAL na mesma conexão do storage unificado (ADR-001) | Outbox na mesma transação exige mesmo DB; simplifica backup e replay | y |
| Formato do `event_id` | UUID v7 (sortable + collision-safe) | v7 carrega timestamp + random; melhor que v4 pra ordem causal e indexação | y |
| `schema_version` inicial | `1` | Sem migração de envelope ainda; quebrar = nova major + ADR de migração | y |
| Backpressure padrão | `MaxAckPending = 64` por subscriber | ADR-043 §6: capacidade típica de subscriber; configurável via `.memory/profiles/*.yaml` | y |
| Backoff de retry | Exponencial com jitter, base 100ms, max 30s | ADR-043 §6: producer pausa se log > limite; retry infinito com backoff capped é fail-soft | y |
| ACK explícito vs implícito | Explícito: subscriber chama `Ack(event_id)` após sucesso; timeout = NACK + redelivery | Crash entre handler e ACK = redelivery idempotente (subscribers checam `projection_cursor`) | y |
| Replay tool default | `mem replay --since 0 --dry-run` re-emite todos os eventos sem chamar handlers (auditoria) | `--apply` é opt-in destrutivo; default dry-run evita replay acidental | y |
| `actor` no envelope | String tipada `user:<id>` \| `agent:<model>` \| `system:<component>` | ADR-050 LLM02/LLM09: proveniência explícita em todo evento | y |
| `priority` no envelope | Enum fixo: `low` \| `normal` \| `high` | Dispatcher respeita ordem de prioridade apenas em caso de backpressure extrema | y |
| `payload` size limit | 1 MiB hard limit; warning em >256 KiB | Payload grande indica tool result ou chunk; melhor serializar externamente e referenciar por `aggregate_id` | y |
| Replay end-of-stream | Se `--since <seq>` ultrapassa último sequence, retorna 0 eventos com exit 0 | Comportamento de stream Unix-like; usuário checa `--last-sequence` separadamente | y |

**Open questions:** none — all resolved or logged above.

---

## User Stories

### P1: Envelope Canônico + Event Log SQLite ⭐ MVP

**User Story**: As a núcleo `mymemoryd`, I want registrar cada evento autorizado num log SQLite durável e ordenado so that replay, auditoria e outbox funcionem sem infra externa.

**Why P1**: É o substrato sobre o qual producer, dispatcher e subscribers constroem. Sem log durável e ordenado, nenhuma garantia (crash-safety, idempotência, replay) é verificável.

**Acceptance Criteria**:

1. The system SHALL persist an `event_log` table with `sequence INTEGER PRIMARY KEY AUTOINCREMENT`, `event_id TEXT UNIQUE NOT NULL`, `schema_version INTEGER NOT NULL DEFAULT 1`, `event_type TEXT NOT NULL`, `aggregate_id TEXT NOT NULL`, `revision INTEGER`, `payload BLOB NOT NULL`, `headers BLOB NOT NULL`, `created_at TEXT NOT NULL`, `acked_at TEXT`. (ubiquitous)
2. The system SHALL define Go struct `event_runtime.Envelope` matching ADR-043 §4.1: `EventID`, `SchemaVersion`, `EventType`, `Producer`, `CreatedAt`, `MonotonicNS`, `SessionID`, `ConversationID`, `AggregateID`, `Epoch`, `Sequence`, `CorrelationID`, `CausationID`, `Actor`, `Locale`, `Priority`, `ContentType`, `Payload json.RawMessage`, `Provenance`, `Trace`. (ubiquitous)
3. WHEN `Envelope` is serialized to JSON THEN the field order SHALL match ADR-043 §4.1 (stable across versions). (event-driven)
4. The system SHALL reject envelopes with empty `event_type` or empty `aggregate_id` returning `ErrInvalidEnvelope` before any DB write. (unwanted-behavior)
5. The system SHALL auto-assign `event_id` (UUID v7) when the caller does not provide one. (ubiquitous)
6. The system SHALL enforce payload size ≤ 1 MiB; IF a producer submits payload > 1 MiB THEN the system SHALL reject with `ErrPayloadTooLarge` and log a warning. (unwanted-behavior)
7. The system SHALL expose `Log.Append(ctx, tx, env)` which writes the envelope in the **caller's transaction** (no implicit `BEGIN`/`COMMIT`). (ubiquitous)

**Independent Test**: `TestEventLog_AppendInTransaction_AndReorder` valida ordem causal via `ORDER BY sequence`; `TestEventLog_DuplicateEventID_IsIdempotent` valida `UNIQUE` constraint; `TestEventLog_PayloadTooLarge_IsRejected`.

---

### P2: Producer com Outbox Atômico

**User Story**: As a qualquer subsistema que queira emitir um efeito colateral, I want helper `Outbox.Emit(ctx, tx, env)` que grava o efeito e o evento na mesma transação so que crash entre os dois não deixe estado inconsistente.

**Why P2**: É o que transforma o log em outbox real. Sem atomicidade transacional, "commit no disco + evento no log" é duas escritas com janela de inconsistência.

**Acceptance Criteria**:

1. The system SHALL provide `event_runtime.Outbox` exposing `Emit(ctx, tx, env)` and `EmitBatch(ctx, tx, envs)`. (ubiquitous)
2. WHEN `Outbox.Emit` is called inside an existing `*sql.Tx` THEN the system SHALL insert into `event_log` using that transaction and SHALL NOT call `COMMIT` itself. (event-driven)
3. IF the caller-provided transaction is `nil` THEN the system SHALL reject with `ErrTxRequired` before any DB write. (unwanted-behavior)
4. IF the underlying `COMMIT` (called by the caller) fails THEN the system SHALL guarantee that no event from this batch is visible to subscribers (atomicidade real). (unwanted-behavior)
5. WHEN the caller retries the entire transaction (e.g., after `SQLITE_BUSY`) THEN the system SHALL reject any duplicate `event_id` from the retry without re-applying the effect (idempotência por chave única). (event-driven)

**Independent Test**: `TestOutbox_AtomicWithEffect` (paralelo: 1 transação com INSERT em `documents` + outbox emit; rollback explícito; assert que event_log está vazio). `TestOutbox_RetryDoesNotDuplicateEventID` (insere event_id; retry da transação; assert `UNIQUE constraint failed` é tratado como sucesso idempotente).

---

### P3: Dispatcher In-Process com Fan-out, ACK e Retry

**User Story**: As a core runtime, I want um dispatcher que entrega eventos a subscribers registrados com ACK explícito, retry exponencial e backpressure so que efeitos colaterais sejam entregues ao menos uma vez mesmo com crashes parciais.

**Why P3**: Outbox sozinho não entrega nada — só registra. Dispatcher é o motor que lê do log e chama subscribers. Sem ele, event_log é tabela morta.

**Acceptance Criteria**:

1. The system SHALL provide `event_runtime.Dispatcher` exposing `Subscribe(eventTypePattern, handler)`, `Start(ctx)`, `Stop(ctx)`. (ubiquitous)
2. WHEN `Dispatcher.Start` is called THEN the system SHALL spawn worker goroutines that read from `event_log WHERE acked_at IS NULL ORDER BY sequence ASC` and deliver to matching subscribers. (event-driven)
3. WHILE `MaxAckPending` per subscriber is reached, the dispatcher SHALL pause reading from `event_log` for that subscriber until at least one in-flight event is acked. (state-driven)
4. WHEN a subscriber handler returns `nil` THEN the dispatcher SHALL mark the event as acked (`UPDATE event_log SET acked_at = ? WHERE sequence = ?`). (event-driven)
5. WHEN a subscriber handler returns a non-nil error THEN the dispatcher SHALL re-queue the event with exponential backoff (base 100ms, factor 2, jitter ±20%, cap 30s) and SHALL NOT mark as acked. (event-driven)
6. WHEN a subscriber handler panics THEN the dispatcher SHALL recover, log the panic with `event_id` and `correlation_id`, and treat as a transient error (NACK + retry). (unwanted-behavior)
7. IF an event has `schema_version > 1` (unknown) THEN the dispatcher SHALL reject it permanently (mark as acked with `acked_reason = 'unsupported_schema'`) to avoid infinite retry. (unwanted-behavior)
8. IF the event log is empty THEN the dispatcher SHALL block on a poll interval (default 100ms) until at least one event appears. (state-driven)
9. The system SHALL persist per-subscriber `projection_cursor.last_sequence` in the `projection_cursor` table, updated atomically with the ACK, to support crash-safe resume. (ubiquitous)

**Independent Test**: `TestDispatcher_DeliversToSubscribers` (3 subscribers, 1 evento, assert todos recebem). `TestDispatcher_ACKUpdatesCursor` (handler OK, assert `projection_cursor.last_sequence` cresce). `TestDispatcher_RetryOnError` (handler retorna erro 3x; assert 3 tentativas com backoff; após sucesso, ACKed). `TestDispatcher_PanicRecoveredAsTransient` (handler panic; assert log de pânico + retry). `TestDispatcher_UnsupportedSchemaIsAckedPermanently` (schema_version=99; assert `acked_at` setado e `acked_reason='unsupported_schema'`). `TestDispatcher_MaxAckPending_PausesRead` (subscriber lento com MaxAckPending=1; producer emite 5 eventos; assert não acumula > 1 in-flight).

---

### P4: CLI de Inspeção e Replay

**User Story**: As a desenvolvedor ou auditor, I want inspecionar o event log e re-executar replay a partir de um sequence so that eu possa depurar sessões passadas, auditar efeitos e validar invariantes após deploy.

**Why P4**: Sem CLI, o event log é uma caixa-preta. Replay é a única forma de validar atomicidade/idempotência em produção.

**Acceptance Criteria**:

1. The system SHALL provide CLI subcommand `mem events tail [--session <id>] [--type <pattern>] [--limit <n>]` printing the last N events (default 20) as one JSON object per line, ordered by `sequence DESC`. (ubiquitous)
2. The system SHALL provide CLI subcommand `mem events inspect <event_id>` printing the full envelope (header + payload pretty-printed) for a single event, exiting non-zero if not found. (ubiquitous)
3. The system SHALL provide CLI subcommand `mem replay --since <seq> [--to <seq>] [--dry-run] [--apply]` re-emitting events to subscribers; default `--dry-run` calls `Subscriber.InspectOnly` (não aplica efeito). (ubiquitous)
4. WHEN `mem replay --apply` is invoked THEN the system SHALL require interactive confirmation (`--yes` flag bypasses) and SHALL log a structured warning before delivery. (unwanted-behavior)
5. The system SHALL provide CLI subcommand `mem trace <correlation_id>` returning the DAG of events whose `correlation_id` matches, ordered by `sequence ASC`, with one line per event showing `sequence`, `event_type`, `aggregate_id`, `actor`. (ubiquitous)
6. The system SHALL provide CLI subcommand `mem events last-sequence` printing the current highest `sequence` in `event_log`, useful as upper bound for `--since`. (ubiquitous)
7. The system SHALL provide CLI subcommand `mem events stats` printing: total events, events by `event_type` (top 10), oldest unacked `sequence`, oldest unacked age in seconds. (ubiquitous)
8. IF the user invokes any of the above commands without a configured vault (`.memory/config.yaml` missing) THEN the CLI SHALL exit 2 with a clear error pointing to `mem init`. (unwanted-behavior)

**Independent Test**: `TestCLI_EventsTail_PrintsJSONLines` (popula 5 eventos; assert saída tem 5 linhas, formato JSON válido, ordem DESC). `TestCLI_EventsInspect_NotFound_ExitsNonZero`. `TestCLI_ReplayDryRun_DoesNotApplyEffects` (subscriber conta chamadas; assert count=0 após `--dry-run`). `TestCLI_ReplayApply_RequiresConfirmation` (sem `--yes`, assert prompt de confirmação; com `--yes`, assert execução). `TestCLI_Trace_ReturnsAllEventsWithCorrelationID`. `TestCLI_EventsStats_ReportsOldestUnacked`.

---

### P5: Interface `Subscriber` + Subscriber de Exemplo (Audit)

**User Story**: As a autor de uma projeção futura (sqlite/fts/graph/vec), I want uma interface `event_runtime.Subscriber` clara e um exemplo concreto (audit) so que eu possa registrar novas projeções sem mexer no dispatcher.

**Why P5**: Sem interface + exemplo, cada subsistema vai reinventar acoplamento com o dispatcher. O exemplo `audit` prova que o pipeline funciona ponta-a-ponta.

**Acceptance Criteria**:

1. The system SHALL define Go interface `event_runtime.Subscriber` with methods `Name() string`, `EventTypes() []string` (ou pattern), `Handle(ctx, env) error`, `MaxAckPending() int`. (ubiquitous)
2. The system SHALL ship `event_runtime/audit_subscriber.go` persisting each event as a structured JSONL line at `.memory/audit.log` with `event_id`, `sequence`, `event_type`, `actor`, `redacted_payload_hash` (SHA-256 do payload original, **nunca** o payload em si). (ubiquitous)
3. WHEN the audit subscriber is registered THEN `event_log` SHALL receive a `audit.persisted` event per processed envelope, with `provenance.tool_call_id` and `provenance.audit_subscriber=true`. (event-driven)
4. The audit subscriber SHALL NOT log any field classified as PII by ADR-050 LLM02 (audio URLs, transcription text, full prompts, tokens). The redaction map SHALL be configurable via `internal/event_runtime/redaction.go` with default rules: replace `payload.transcript` with `[REDACTED:transcript]`, `payload.audio_url` with `[REDACTED:audio_url]`, `payload.prompt` with `[REDACTED:prompt]`. (ubiquitous)
5. The system SHALL provide helper `event_runtime.RegisterSubscriber(dispatcher, sub)` that validates the `Name()` is non-empty and unique within the running dispatcher. (ubiquitous)
6. IF a subscriber registration is attempted with duplicate name THEN the system SHALL reject with `ErrDuplicateSubscriber` before any goroutine starts. (unwanted-behavior)

**Independent Test**: `TestSubscriber_AuditSubscriber_RedactsPII` (emite evento com `payload.transcript="hello world"`; assert `.memory/audit.log` NÃO contém "hello world" e contém `[REDACTED:transcript]`). `TestSubscriber_RegisterDuplicateName_IsRejected`. `TestSubscriber_HandleError_PropagatesToDispatcher` (handler retorna erro; assert dispatcher faz retry).

---

## Edge Cases

- IF the event log table is missing on startup (`event_log` não existe) THEN the dispatcher SHALL refuse to start with `ErrEventLogNotInitialized` (fail-closed, fail-loud). (unwanted-behavior)
- IF the SQLite WAL file is corrupt (`PRAGMA integrity_check` retorna falha) THEN the dispatcher SHALL refuse to start and SHALL log full diagnostics. (unwanted-behavior)
- IF `event_id` collides between two producers (UUID v7 collision) THEN the second insert SHALL fail with `UNIQUE constraint failed`; the producer SHALL treat this as idempotent retry and NOT raise an error to the caller. (unwanted-behavior)
- IF the dispatcher is stopped with N events in-flight THEN the system SHALL wait up to 30s for graceful shutdown (handlers complete), then force-cancel context and NACK remaining events (re-delivered on next start). (state-driven)
- IF a replay target range contains events with `schema_version > current` THEN the replay tool SHALL skip them and emit a warning per event (do not crash, do not apply). (unwanted-behavior)
- IF the audit log file is on a read-only filesystem THEN the audit subscriber SHALL fail-closed (NACK + retry with backoff) instead of silently dropping events. (unwanted-behavior)

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| --- | --- | --- | --- |
| EVT-01 | P1: Envelope Canônico + Event Log SQLite | Tasks | Pending |
| EVT-02 | P1: Envelope Canônico + Event Log SQLite | Tasks | Pending |
| EVT-03 | P1: Envelope Canônico + Event Log SQLite | Tasks | Pending |
| EVT-04 | P1: Envelope Canônico + Event Log SQLite | Tasks | Pending |
| EVT-05 | P2: Producer com Outbox Atômico | Tasks | Pending |
| EVT-06 | P2: Producer com Outbox Atômico | Tasks | Pending |
| EVT-07 | P2: Producer com Outbox Atômico | Tasks | Pending |
| EVT-08 | P3: Dispatcher In-Process | Tasks | Pending |
| EVT-09 | P3: Dispatcher In-Process | Tasks | Pending |
| EVT-10 | P3: Dispatcher In-Process | Tasks | Pending |
| EVT-11 | P3: Dispatcher In-Process | Tasks | Pending |
| EVT-12 | P3: Dispatcher In-Process | Tasks | Pending |
| EVT-13 | P3: Dispatcher In-Process | Tasks | Pending |
| EVT-14 | P4: CLI de Inspeção e Replay | Tasks | Pending |
| EVT-15 | P4: CLI de Inspeção e Replay | Tasks | Pending |
| EVT-16 | P4: CLI de Inspeção e Replay | Tasks | Pending |
| EVT-17 | P5: Interface Subscriber + Audit Subscriber | Tasks | Pending |
| EVT-18 | P5: Interface Subscriber + Audit Subscriber | Tasks | Pending |
| EVT-19 | P5: Interface Subscriber + Audit Subscriber | Tasks | Pending |

**Coverage:** 19 total, 19 mapped, 0 unmapped.

**ID format:** `EVT-NN` — kebab-case no feature dir (`event-runtime-event-log`), SCREAMING-KEBAB nos requirement IDs.

**Status values:** Pending → In Design → In Tasks → Implementing → Verified.

---

## Success Criteria

- [ ] Tabelas `event_log` + `projection_cursor` criadas via migração SQL idempotente.
- [ ] Pacote `internal/event_runtime/` com sub-pacotes `envelope`, `log`, `outbox`, `dispatcher`, `subscriber`, `audit`, `redaction`.
- [ ] CLI: `mem events tail|inspect|replay|trace|last-sequence|stats` funcionais.
- [ ] 100% dos acceptance criteria cobertos por testes Go (`go test -count=1 ./internal/event_runtime/...`).
- [ ] Quality gate: `gofmt -l .` limpo, `go build ./...` limpo, `go test -count=1 ./...` verde.
- [ ] ADR-043 referenciado em código (`doc.go` do pacote) e na doc do CLI (`cmd/mem/events.go`).
- [ ] `mem doctor` ganha check opcional `--events` reportando `total_events`, `oldest_unacked_age_seconds`, `subscribers_active`.
- [ ] ADR-050 (Threat Model): controle LLM02 (redaction) implementado em `redaction.go` com testes.

---

## Architectural Reference (do not duplicate)

O design completo deste feature vive em [ADR-043: Envelope de Eventos Canônico](../../../docs/adr/043-envelope-de-eventos-canonico-event-runtime.md). Esta spec captura apenas o **WHAT testável** (ACs EARS + IDs + edge cases). Decisões arquiteturais (por que SQLite WAL, por que in-process, por que ADR-043 rejeitou NATS/Redis/ZeroMQ) estão no ADR e não precisam ser reproduzidas aqui.
