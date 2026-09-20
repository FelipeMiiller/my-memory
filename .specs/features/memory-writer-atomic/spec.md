# Feature: memory-writer-atomic

> **Status**: Specify phase — design document is [ADR-044](../../../docs/adr/044-memory-writer-atomico-outbox-e-reprojecao-idempotente.md); this spec captures WHAT (testable requirements), not the architectural discussion.

## Problem Statement

O padrão atual de escrita (`ADR-018` Compile-not-Retrieve) é síncrono: `mem note create` grava o `.md`, dispara reindexação inline, e retorna. Funciona para CLI/MCP pontual, mas tem três fragilidades que ficam críticas quando voz/agente entram:

1. **Sem crash-safety** — se o processo morre entre `.md` gravado e indexação completa, estado fica inconsistente até o próximo `mem index`.
2. **Sem idempotência forte** — replay ou retry duplica arestas/chunks.
3. **Sem precondition de versão** — edições concorrentes sobrescrevem silenciosamente.

O **ADR-044** decidiu a arquitetura (outbox SQLite + event_runtime + idempotência + precondition por revision). Falta o **WHAT testável**: schema exato, contrato do writer, formato dos eventos `memory.committed` / `memory.write.rejected` / `conflict.detected`, e invariantes de replay.

Sem este spec, Foundation não pode começar — todo write de voz/agente vai reinventar atomicidade a cada caller.

## Goals

- [ ] Tabela `event_log` (ADR-043) + tabela `projection_cursor` (idempotência de subscribers) já criadas pela spec `event-runtime-event-log`.
- [ ] Colunas `documents.revision INTEGER NOT NULL DEFAULT 0` + `documents.last_event_id TEXT` adicionadas via `ALTER TABLE` não-destrutivo.
- [ ] `internal/writer/writer.go` orquestrando write + outbox na **mesma transação SQLite WAL** (atomicidade real entre `.md` no disco e `event_log` entry).
- [ ] Precondition checker (`internal/writer/precondition.go`) comparando `expected_revision` com `documents.revision` read-Committed.
- [ ] Policy engine stub (`internal/policy/engine.go`) lendo `.memory/policy/*.yaml` com 3 profiles: `strict`, `balanced`, `permissive-dev` (per ADR-050).
- [ ] Projeção `projection.sqlite` (subscriber idempotente) upsert em `documents`, `chunks`, `chunks_fts` via `event_log` replay.
- [ ] Projeção `projection.graph` (subscriber idempotente) fazendo diff de edges (delete old + insert new por `(document_id, revision_id)`).
- [ ] Eventos `memory.write.requested`, `memory.write.rejected`, `memory.committed`, `conflict.detected` emitidos via `event_runtime.Log.Append`.
- [ ] CLI ganha `--if-match <revision>` flag; default ignora (sempre grava); `--strict` exige match.
- [ ] MCP tools `memory_write_note` / `memory_append_section` / `memory_compile_note` ganham parâmetro `if_match` opcional.
- [ ] Cobrir invariantes: `TestWrite_CommitIsAtomic`, `TestWrite_IdempotentReplay`, `TestWrite_PreconditionConflict_409`, `TestWrite_CrashBetweenCommitAndDispatch`, `TestWrite_PolicyBlocksHighRiskActor`.

## Out of Scope

| Feature | Reason |
| --- | --- |
| Projeção `projection.vec` (embeddings) | Depende do modelo (ADR-035) + Nemotron ASR (ADR-045) — implementação separada após vetor foundation estiver estável |
| Projeção `viewer.timeline` | Viewer rewrite é ADR-037/ADR-049; esta spec entrega apenas SQLite + graph |
| Audit subscriber estruturado (`mem trace`) | Já existe subscriber básico do event_runtime; refinamento é enhancement |
| CRDT / merge automático de Markdown | ADR-044 §Considered Options C descartou — overhead alto para Markdown humano |
| Banco operacional write-through separado | ADR-044 §Considered Options D — quebraria ADR-005 (Markdown source-of-truth) |
| Optimistic UI / undo do caller | Caller-side, não responsabilidade do writer |
| Versionamento semântico do Markdown | ADR-019 cobre release-version; document-level revision é só contador monotônico |

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --- | --- | --- | --- |
| Atomicidade transacional | `.md` no disco + `event_log` na mesma transação SQLite WAL | `tmp + rename` no filesystem + `BEGIN IMMEDIATE` no SQLite evita race; ordem canônica | y |
| Precondition encoding | Header `If-Match: revision=N` (HTTP-like) — extensível a outros campos depois | Inspirado em ETag HTTP; simples de implementar e testar | y |
| `expected_revision == 0` (criação) | Aceitar como "criar novo documento" | Caso comum de `mem note create` sem ler antes; evita flag separada | y |
| Política default | `balanced` é o default; `strict` exige `--strict-policy` flag; `permissive-dev` opt-in via config | ADR-050 §Negative Consequences — 3 profiles dá tunability sem complicar UX | y |
| `projection.graph` strategy | Delete all edges de `(document_id, revision_id-1)` + insert edges de `revision_id` | Replay precisa produzir mesmo estado; diff é mais simples que delta | y |
| Crash recovery | Startup lê `projection_cursor.last_sequence` e replay a partir de `last_sequence+1` até `MAX(sequence)` em `event_log` | Idempotente porque subscribers checam `event_id` antes de aplicar | y |
| Idempotency key | `event_id` (UUID v7) é o dedup key; `UNIQUE` constraint em `event_log.event_id` | ADR-043 §Decision Outcome já definiu | y |
| Backpressure | `MaxAckPending = 64` por subscriber; backoff exponencial capped 30s | ADR-043 §6 já fixou; reuso de contrato | y |
| Write conflict resolution | Não resolver automaticamente — emitir `conflict.detected` event + retornar `409 Conflict` + `current_revision` no payload | Conflito semântico (qual versão prevalece) precisa decisão humana | y |
| Quarantine | Documentos com `provenance.trust_level = "untrusted"` vão pra `documents.quarantine` table separada | ADR-050 LLM04: conteúdo importado não vira canônico direto | n (verificar se tabela existe ou criar) |
| Storage de `.md` durante write | `tmp` em mesma pasta + `os.Rename` (atomic no mesmo filesystem) | `rename(2)` é atômico em Linux/macOS; no Windows usa `MoveFileEx` com flag replace | y |
| Limite de tamanho do write | 5 MiB hard limit; warning em >1 MiB | Markdown típico < 100 KiB; 5 MiB cobre anexos via `![[file]]` | n (decidir antes da T1) |

**Open questions:** 2 logged above — quarantine table location + size limit. Both must resolve before T1.

---

## User Stories

### P1: Write Atômico + Outbox ⭐ MVP

**User Story**: As a caller (CLI, MCP tool, voz ou agente), I want cada `mem note create` gravar o `.md` e o evento de commit na mesma transação SQLite so that crash-safety seja real e o event_log nunca fique fora de sync com o filesystem.

**Why P1**: É o substrato sobre o qual idempotência (P2), precondition (P3) e policy (P4) constroem. Sem atomicidade real, nenhum dos outros P-stories tem garantia verificável.

**Acceptance Criteria**:

1. WHEN `writer.Write(ctx, request)` is called with valid request THEN the system SHALL `BEGIN IMMEDIATE` SQLite transaction, write `.md` via `tmp + os.Rename`, compute SHA-256 `content_hash`, insert `event_log` row with `event_type="memory.committed"`, and `COMMIT` in a single atomic step. (event-driven)
2. IF `.md` write fails (disk full, permission denied) THEN the system SHALL rollback SQLite transaction and return `ErrWriteFailed` with no event published. (unwanted-behavior)
3. IF SQLite `BEGIN` fails THEN the system SHALL remove the `.md` file (compensating action) and return `ErrDatabaseUnavailable`. (unwanted-behavior)
4. The system SHALL emit `memory.committed` envelope with `aggregate_id=<document_id>`, `revision=<N+1>`, `payload.content_hash=<sha256>`, `payload.anchors=[wikilinks]`, `payload.actor=<actor>`. (ubiquitous)
5. The system SHALL emit `memory.write.requested` envelope **before** the transaction begins with `payload.actor` and `payload.intended_path`. (event-driven)
6. The system SHALL auto-assign `revision` as `documents.revision + 1` and write back to `documents.revision` in the same transaction. (ubiquitous)

**Independent Test**: `TestWrite_CommitIsAtomic` simula crash (kill -9) entre `.md` rename e SQLite COMMIT — verifica que próximo startup detecta estado inconsistente e rollback; `TestWrite_NoEventIfMdFails` força erro de filesystem e verifica 0 rows em `event_log`.

---

### P2: Idempotência + Replay

**User Story**: As a operador, I want rodar `mem replay --since <seq>` e o sistema reconstruir todas as projeções a partir do event log so that eu possa recuperar de corrupção de SQLite ou restaurar backups.

**Why P2**: Sem idempotência forte, replay duplica arestas no grafo e chunks no FTS. ADR-044 §Decision Outcome garante isso como requisito.

**Acceptance Criteria**:

1. The system SHALL advance `projection_cursor.last_sequence` only after a subscriber successfully applies the event and acknowledges. (ubiquitous)
2. WHEN `mem replay --since <seq>` is invoked THEN the system SHALL re-emit every event from `seq` to `MAX(sequence)` in order, and subscribers SHALL apply them idempotently via `INSERT OR IGNORE` keyed on `event_id`. (event-driven)
3. The system SHALL guarantee that running `mem replay --since 0` followed by current state produces **byte-identical** SQLite state as a fresh `--rebuild-from-event-log` run. (ubiquitous)
4. The system SHALL reject duplicate `event_id` insertions with `ErrDuplicateEventID` (UNIQUE constraint on `event_log.event_id`). (unwanted-behavior)
5. IF a subscriber crashes mid-handler THEN on restart it SHALL re-receive the same `event_id` (redelivery) without duplicating effects, because `INSERT OR IGNORE` on `event_id` is the dedup primitive. (unwanted-behavior)

**Independent Test**: `TestWrite_IdempotentReplay` aplica 100 eventos, replay 5x, verifica state byte-equal; `TestReplay_RestoresFromBackup` deleta SQLite, restaura event_log de backup, replay desde 0, verifica estado completo reconstruído.

---

### P3: Precondition de Versão

**User Story**: As a agente concorrente, I want detectar conflito de versão antes do write so that duas edições simultâneas não sobrescrevam silenciosamente.

**Why P3**: ADR-044 §DR-3 — sem precondition, agente 1 lê doc com revision=5, agente 2 grava incrementando pra 6, agente 1 grava revision=6 e perde a mudança do agente 2 sem aviso.

**Acceptance Criteria**:

1. WHEN `writer.Write` is called with `expected_revision=5` and `documents.revision=5` THEN the system SHALL proceed with write and increment to `revision=6`. (event-driven)
2. WHEN `writer.Write` is called with `expected_revision=5` and `documents.revision=7` THEN the system SHALL return `ErrPreconditionFailed` (HTTP 409 equivalent) and emit `conflict.detected` event with `payload.expected=5`, `payload.current=7`. (unwanted-behavior)
3. WHEN `writer.Write` is called with `expected_revision=0` and the document already exists THEN the system SHALL return `ErrPreconditionFailed` (creation conflict). (unwanted-behavior)
4. WHEN `writer.Write` is called with no `expected_revision` (CLI default) THEN the system SHALL proceed unconditionally, similar to status quo. (state-driven)
5. The system SHALL include `If-Match` header in HTTP-style metadata for future HTTP transport compatibility. (ubiquitous)

**Independent Test**: `TestWrite_PreconditionConflict_409` valida 3 cenários: match ok, mismatch rejeita, no-revision default aceita.

---

### P4: Policy Engine + 3 Profiles

**User Story**: As a operador, I want configurar `.memory/policy/balanced.yaml` com allowlist de actors, scopes de curta duração, e risk levels so that writes arriscados (actor externo, path fora do vault) exijam aprovação.

**Why P4**: ADR-050 LLM06 (Excessive Agency) — política fora do LLM; ADR-042 §policy-engine é o componente.

**Acceptance Criteria**:

1. WHERE `.memory/policy/strict.yaml` is active THEN every external actor (`actor != "user:<owner>"`) SHALL require approval before write. (optional-feature)
2. WHERE `.memory/policy/balanced.yaml` is active THEN writes within vault scope (`path matches vault.glob`) SHALL proceed; writes outside scope SHALL require approval. (optional-feature)
3. WHERE `.memory/policy/permissive-dev.yaml` is active THEN all writes proceed without approval (development mode). (optional-feature)
4. WHEN policy requires approval THEN the system SHALL emit `approval.requested` event and block the write until `approval.granted` or `30 minutes` timeout. (event-driven)
5. IF policy decision fails to load THEN the system SHALL fail-closed with `ErrPolicyUnavailable` (ADR-050 fail-closed posture). (unwanted-behavior)
6. The system SHALL record `policy.decision_id` in every `memory.committed` envelope for audit traceability. (ubiquitous)

**Independent Test**: `TestWrite_PolicyBlocksHighRiskActor` valida actor `agent:<external>` rejeitado em `strict`; `TestWrite_PolicyRequiresApprovalForExternalScope` valida path fora do vault em `balanced`.

---

## Edge Cases

- IF `.md` file is being read by another process during write (Windows file lock) THEN the system SHALL retry rename up to 3x with 100ms backoff. (unwanted-behavior)
- WHEN two writers race for the same `document_id` THEN the second SHALL receive `ErrPreconditionFailed` (UNIQUE on `documents.document_id` + version check). (state-driven)
- IF `documents.revision` overflows (int64 max ≈ 9.2 × 10^18) THEN the system SHALL refuse new writes and emit `revision.overflow` event. (unwanted-behavior)
- WHEN `expected_revision` is negative THEN the system SHALL reject with `ErrInvalidRevision`. (unwanted-behavior)
- IF `.memory/config.yaml` is missing THEN the system SHALL default to `balanced` policy and warn at startup. (unwanted-behavior)

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| --- | --- | --- | --- |
| WRTR-01 | P1 | Design | Pending |
| WRTR-02 | P1 | Design | Pending |
| WRTR-03 | P1 | Design | Pending |
| WRTR-04 | P1 | Design | Pending |
| WRTR-05 | P1 | Design | Pending |
| WRTR-06 | P1 | Design | Pending |
| WRTR-07 | P2 | Design | Pending |
| WRTR-08 | P2 | Design | Pending |
| WRTR-09 | P2 | Design | Pending |
| WRTR-10 | P2 | Design | Pending |
| WRTR-11 | P2 | Design | Pending |
| WRTR-12 | P3 | Design | Pending |
| WRTR-13 | P3 | Design | Pending |
| WRTR-14 | P3 | Design | Pending |
| WRTR-15 | P3 | Design | Pending |
| WRTR-16 | P3 | Design | Pending |
| WRTR-17 | P4 | Design | Pending |
| WRTR-18 | P4 | Design | Pending |
| WRTR-19 | P4 | Design | Pending |
| WRTR-20 | P4 | Design | Pending |
| WRTR-21 | P4 | Design | Pending |
| WRTR-22 | P4 | Design | Pending |

**Coverage:** 22 total, 0 mapped to tasks, 22 unmapped ⚠️ (tasks.md pending).

---

## Success Criteria

- [ ] `mem note create ./foo.md --content "..."` followed by `kill -9` mid-write leaves filesystem and SQLite in consistent state (next startup detects + rolls back).
- [ ] `mem replay --since 0` after fresh DB produces byte-identical state.
- [ ] `writer.Write` with stale `expected_revision` returns 409 + emits `conflict.detected` event.
- [ ] Policy `strict` blocks all writes from non-owner actors (regression test).
- [ ] `go test -count=1 ./internal/writer/...` passes 100%.
- [ ] No `payload.transcript` or other LLM02-sensitive content reaches `event_log.payload` without redaction (validated by grep test).
- [ ] CLI `--if-match` flag works for both create and update paths.
- [ ] MCP tools accept `if_match` parameter and emit appropriate error envelope.
