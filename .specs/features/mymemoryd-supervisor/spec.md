# Feature: mymemoryd-supervisor

> **Status**: Specify phase — design document is [ADR-042](../../../docs/adr/042-nucleo-local-mymemoryd-com-workers-sidecar-isolados.md); this spec captures WHAT (testable requirements) for the `mymemoryd` Go core + supervisor + worker sidecar topology.

## Problem Statement

O MyMemory hoje opera como CLI pontual e servidor MCP sob demanda — cada invocação abre SQLite, lê, escreve, fecha. A pesquisa de 2026-09-18 demanda voz contínua, agente runtime, viewer contínuo, replay, e projeções assíncronas. Esses componentes precisam de processo de longa duração, workers residentes, barramento de eventos, sandbox explícito, e recovery pós-crash.

O **ADR-042** decidiu a topologia: `mymemoryd` em Go como núcleo + workers Python sidecar via JSON-RPC sobre stdio/Unix socket. Falta o **WHAT testável**: entrypoint, contrato do supervisor, formato do manifesto de workers, profiles, modos (up/down/status/logs), e garantias de sandbox.

Sem este spec, F0/F1 (Foundation) não pode começar — todo caller (voz, agente, viewer) vai reinventar lifecycle de processo.

## Goals

- [ ] `cmd/mymemoryd/main.go` entrypoint lendo `.memory/profiles/<active>.yaml` manifesto de workers e subindo supervisor.
- [ ] `internal/supervisor/` com lifecycle: start, stop, health-check (heartbeat 5s), restart com backoff exponencial (max 30s), crash detection (worker exited ≠ 0).
- [ ] `internal/ipc/jsonrpc/` framing JSON-RPC 2.0 sobre stdio OU Unix socket com 4-byte length prefix pra delimitar frames binários.
- [ ] Profiles `.memory/profiles/{default,voice,headless}.yaml` declarando workers obrigatórios vs opcionais + ordem de start.
- [ ] CLI `mem up`, `mem down`, `mem status`, `mem logs <worker>`, `mem profiles list/use <name>`.
- [ ] Modo standalone (sem `mymemoryd`) continua funcionando — CLI verifica `mymemoryd` rodando e usa path rápido in-process; fallback pra standalone se supervisor indisponível.
- [ ] Sandbox explícito: workers sem rede por default (egress deny); `internal/supervisor/sandbox.go` configura cgroup/namespace quando disponível, no-op warning em plataformas sem suporte.
- [ ] Audit log estruturado: `mymemoryd` loga cada worker start/stop/crash/restart com `worker_id`, `pid`, `reason`, `correlation_id` (quando aplicável).
- [ ] Cobrir invariantes: `TestSupervisor_RestartCrashedWorker`, `TestSupervisor_FailsClosedOnProfileError`, `TestIpc_FrameDelimitsBinaryAudio`, `TestProfile_AppliesManifestOrder`, `TestStandalone_FallbackWhenDaemonDown`.

## Out of Scope

| Feature | Reason |
| --- | --- |
| Implementar workers Python específicos (ASR, TTS, embedder) | Workers vêm de specs separadas (`nemotron-asr-streaming`, Piper TTS ADR-047+, embedder ADR-035) — esta spec entrega o supervisor + contrato, não os workers |
| HTTP/SSE transport pro `mymemoryd` | ADR-021 MCP HTTP é F5+; esta spec é stdio-first |
| Auto-update / versionamento do `mymemoryd` | ADR-019 cobre semver; update mechanism é enhancement |
| mTLS / autenticação entre supervisor e workers | Unix socket + filesystem permissions é suficiente pra MVP local; remote auth é F5 |
| Hot-reload de profiles sem restart | Mudança de profile requer `mem down && mem up --profile <new>`; reload dinâmico é enhancement |
| Distributed supervisor (multi-host) | Single-host MVP; cross-host é F5+ (cluster mode) |
| Web dashboard pro supervisor | `mem status` em CLI é suficiente; dashboard é viewer rewrite (ADR-049) |
| Métricas Prometheus / OpenTelemetry | F5+; por ora `mem status --json` tem contadores básicos |

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --- | --- | --- | --- |
| Ativação do `mymemoryd` | Auto-start opcional via `MEM_AUTOSTART=1` env var; default é `mem up` manual | Não forçar daemon permanente; CLI standalone continua sendo zero-config | y |
| Heartbeat interval | 5s entre supervisor e worker; 3 misses = unhealthy | ADR-042 §DR-4 crash-safety — detecção rápida sem ruído | y |
| Restart backoff | Exponencial 100ms → 30s cap, jitter ±20%; max 5 retries antes de `supervisor.worker_failed` event | ADR-043 §6 já fixou retry contract; mesmo padrão | y |
| Workers Python location | `.memory/workers/<name>/` com entrypoint `__main__.py` | Path convencionado; permite workers de terceiros sem compilar | y |
| JSON-RPC transport | stdio para workers leves (CPU bound); Unix socket para workers de áudio (binário PCM) | ADR-042 §DR-1 — framing binário quando volume alto | y |
| Profile schema version | `schema_version: 1` no YAML; migração é feature separada | YAGNI; quebra = bump major + ADR | y |
| Worker health check payload | JSON `{ "status": "ready" \| "busy" \| "degraded", "metrics": {...} }` | Contrato simples; evolui com métricas | y |
| Sandbox enforcement | Best-effort: `setrlimit` + `prctl(PR_SET_NO_NEW_PRIVS)` quando disponível; warning em sandbox parcial | ADR-042 §DR-5 egress deny-by-default; perfeito não é viável cross-platform | y |
| Graceful shutdown timeout | 10s pra workers honrarem SIGTERM; depois SIGKILL | ADR-042 §DR-7 compatibilidade CLI | y |
| Standalone fallback | Quando supervisor indisponível, CLI abre SQLite direto e executa in-process; loga `running standalone` | ADR-042 §DR-7 — modo standalone é first-class | y |
| Lock file | `.memory/supervisor.lock` com PID; stale lock (>30s sem heartbeat) = removido na próxima start | Evita dois `mymemoryd` rodando simultaneamente | n (verificar compatibilidade com outras ferramentas) |
| Profile inheritance | Profiles podem herdar de `default` via `extends: default` no YAML | DRY; evita duplicação | y |
| Audit retention | 7 dias rolling, gzip após 24h | ADR-050 §Redaction — retenção configurável, default conservador | y |

**Open questions:** 1 — formato do lock file e compatibilidade com multi-CLI simultâneo. Resolver antes da T1.

---

## User Stories

### P1: Supervisor Lifecycle + Crash Recovery ⭐ MVP

**User Story**: As a operador, I want rodar `mem up` e o `mymemoryd` subir todos os workers declarados no profile ativo so that voz/agente/viewer tenham processo de longa duração pra se conectar.

**Why P1**: É o substrato sobre o qual IPC (P2), profiles (P3), sandbox (P4) constroem. Sem supervisor funcional, nenhum worker consegue rodar.

**Acceptance Criteria**:

1. WHEN `mem up` is invoked THEN the system SHALL spawn `mymemoryd` process which reads `.memory/profiles/default.yaml`, starts workers in declared order, and reports `READY` on stdout when all mandatory workers are alive. (event-driven)
2. WHEN a worker process exits unexpectedly (code ≠ 0) THEN the system SHALL restart it with exponential backoff (100ms base, 30s cap, jitter ±20%) up to 5 retries before emitting `supervisor.worker_failed` event with `payload.worker_id`, `payload.exit_code`, `payload.retries`. (unwanted-behavior)
3. The system SHALL emit `worker.started`, `worker.stopped`, `worker.crashed`, `worker.restarted`, `worker.heartbeat_lost` events to `event_log` for audit. (ubiquitous)
4. IF profile YAML fails to parse THEN the system SHALL fail-closed with `ErrInvalidProfile` (ADR-050 LLM10 fail-closed) and exit 2. (unwanted-behavior)
5. WHEN `mem down` is invoked THEN the system SHALL send SIGTERM to all workers, wait 10s, send SIGKILL to survivors, and shutdown supervisor with exit 0. (event-driven)
6. The system SHALL write `.memory/supervisor.lock` with PID at startup; on second `mem up`, refuse if lock is fresh (<30s old) and PID is alive. (state-driven)

**Independent Test**: `TestSupervisor_RestartCrashedWorker` mata worker com SIGKILL, verifica restart automático em <30s; `TestSupervisor_FailsClosedOnProfileError` passa YAML inválido, verifica exit 2 e nenhum worker spawnado.

---

### P2: IPC JSON-RPC + Framing Binário

**User Story**: As a worker Python ou Go, I want comunicar com `mymemoryd` via JSON-RPC com framing binário when audio chunks estão envolvidos so that latência de IPC seja mínima e contratos sejam tipados.

**Why P2**: Sem IPC padronizado, cada worker reinventa socket e parsing. ADR-042 §DR-1 — `mem <sub>` CLI deve responder em <15ms.

**Acceptance Criteria**:

1. WHEN worker connects via stdio THEN the system SHALL negotiate JSON-RPC 2.0 handshake (`initialize` request, `initialized` notification) within 5s or fail-closed. (event-driven)
2. WHEN worker connects via Unix socket THEN the system SHALL use 4-byte big-endian length prefix per frame to delimit binary payloads (audio PCM). (ubiquitous)
3. The system SHALL enforce request timeout of 30s; timeout emits `ipc.timeout` event and closes the connection. (unwanted-behavior)
4. The system SHALL reject malformed JSON-RPC frames (missing `jsonrpc: "2.0"`, missing `id`/`method`) with `-32600 Invalid Request` error. (unwanted-behavior)
5. The system SHALL support batch requests (JSON-RPC 2.0 §6) but reject batches > 100 requests with `-32613 Batch limit exceeded`. (state-driven)

**Independent Test**: `TestIpc_FrameDelimitsBinaryAudio` envia 10 chunks PCM de 512 bytes via Unix socket, verifica framing correto e zero perda; `TestIpc_RejectsMalformedFrame` envia JSON inválido, verifica error code -32600.

---

### P3: Profiles YAML + Herança

**User Story**: As a operador, I want definir profiles `default`, `voice`, `headless` em `.memory/profiles/*.yaml` declarando workers obrigatórios vs opcionais so que diferentes deployments (laptop com mic, servidor sem display) usem o profile certo.

**Why P3**: Sem profiles, todo usuário roda o mesmo conjunto de workers — desperdício em headless server ou falta de capability em laptop sem mic.

**Acceptance Criteria**:

1. WHERE `.memory/profiles/voice.yaml` declares `workers: [asr, tts, embedder, session_manager]` THEN `mem up --profile voice` SHALL start exactly these workers in declared order. (optional-feature)
2. WHERE a profile has `extends: default` THEN the system SHALL merge the parent's `workers` list and override `config` keys. (optional-feature)
3. WHERE a profile declares a worker with `required: true` THEN the system SHALL fail-closed (exit 2) if the worker fails to start within 30s. (optional-feature)
4. WHERE a profile declares a worker with `required: false` THEN the system SHALL start it best-effort; failure logs `worker.optional_failed` event but does not abort. (optional-feature)
5. WHEN `mem profiles list` is invoked THEN the system SHALL print all profiles found in `.memory/profiles/` with their effective worker list. (event-driven)
6. WHEN `mem profiles use <name>` is invoked THEN the system SHALL write `.memory/config.yaml: active_profile = <name>`. (event-driven)

**Independent Test**: `TestProfile_AppliesManifestOrder` valida ordem de start respeitada; `TestProfile_FailsClosedOnRequiredWorkerMissing` valida exit 2 quando worker required não existe.

---

### P4: Sandbox Best-Effort + Audit Log

**User Story**: As a operador, I want cada worker rodar com sandbox best-effort (egress deny, no-new-privs) e cada start/stop/crash auditado so que incidentes tenham rastro forense.

**Why P4**: ADR-050 LLM05/LLM06 + LLM10. Sem sandbox, worker Python comprometido pode fazer request HTTP arbitrário. Sem audit, incidente fica opaco.

**Acceptance Criteria**:

1. WHEN worker starts THEN the system SHALL apply best-effort sandbox: `setrlimit(NOFILE)` cap, `prctl(PR_SET_NO_NEW_PRIVS)` on Linux, no-op warning on macOS/Windows. (event-driven)
2. WHERE worker declares `egress: deny` (default) THEN the system SHALL NOT configure any network namespace; worker has only Unix socket / stdio available. (optional-feature)
3. WHERE worker declares `egress: allow` (opt-in, audit-flagged) THEN the system SHALL configure network namespace with egress allowlist from `policy.yaml`. (optional-feature)
4. The system SHALL log every `worker.started` / `worker.stopped` / `worker.crashed` / `worker.restarted` to structured audit log (JSONL) at `.memory/logs/supervisor.jsonl` with `worker_id`, `pid`, `ts`, `reason`. (ubiquitous)
5. IF audit log write fails THEN the system SHALL continue (don't crash supervisor) but emit `audit.write_failed` event. (unwanted-behavior)
6. The system SHALL rotate audit log when size > 100 MiB OR age > 7 days; older logs gzipped to `.memory/logs/archive/`. (state-driven)

**Independent Test**: `TestSandbox_DenyEgressByDefault` valida que worker sem `egress: allow` flag não consegue abrir TCP socket; `TestAudit_LogsAllLifecycleEvents` valida que 100 start/stop cycles produzem 100+ entries em supervisor.jsonl.

---

## Edge Cases

- IF profile is empty (no workers) THEN the system SHALL start `mymemoryd` with empty worker set and exit immediately. (unwanted-behavior)
- WHEN worker takes >30s to respond to health check THEN system marks as degraded (not dead); heartbeat recovery auto-restores. (state-driven)
- IF `.memory/supervisor.lock` exists but PID is dead (stale) THEN system SHALL remove lock and proceed. (unwanted-behavior)
- WHEN `mem up` is invoked while `mymemoryd` is already running THEN system SHALL be idempotent: detect via lock + PID, return `Already running on PID <N>` exit 0. (state-driven)
- IF worker has `required: true` but its binary is missing (Python module not installed) THEN system SHALL fail-closed with explicit hint `Install worker: pip install mymemory-<worker>` or build instructions. (unwanted-behavior)
- WHEN shutdown is interrupted (SIGINT to `mem down`) THEN system SHALL attempt graceful shutdown but exit non-zero if cleanup incomplete. (unwanted-behavior)

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| --- | --- | --- | --- |
| SUPR-01 | P1 | Design | Pending |
| SUPR-02 | P1 | Design | Pending |
| SUPR-03 | P1 | Design | Pending |
| SUPR-04 | P1 | Design | Pending |
| SUPR-05 | P1 | Design | Pending |
| SUPR-06 | P1 | Design | Pending |
| SUPR-07 | P2 | Design | Pending |
| SUPR-08 | P2 | Design | Pending |
| SUPR-09 | P2 | Design | Pending |
| SUPR-10 | P2 | Design | Pending |
| SUPR-11 | P2 | Design | Pending |
| SUPR-12 | P3 | Design | Pending |
| SUPR-13 | P3 | Design | Pending |
| SUPR-14 | P3 | Design | Pending |
| SUPR-15 | P3 | Design | Pending |
| SUPR-16 | P3 | Design | Pending |
| SUPR-17 | P3 | Design | Pending |
| SUPR-18 | P4 | Design | Pending |
| SUPR-19 | P4 | Design | Pending |
| SUPR-20 | P4 | Design | Pending |
| SUPR-21 | P4 | Design | Pending |
| SUPR-22 | P4 | Design | Pending |
| SUPR-23 | P4 | Design | Pending |

**Coverage:** 23 total, 0 mapped to tasks, 23 unmapped ⚠️ (tasks.md pending).

---

## Success Criteria

- [ ] `mem up` followed by `ps aux | grep mymemoryd` shows supervisor running with N worker child processes.
- [ ] Worker crash (SIGKILL) is auto-recovered in <30s with audit log entry.
- [ ] `mem down` shuts down all workers cleanly in <10s (graceful) or <11s (forced).
- [ ] CLI standalone mode works without supervisor (`mem search "foo"` works even if `mymemoryd` is down).
- [ ] Unix socket IPC delivers 1000 audio chunks with zero loss and <5ms p99 latency.
- [ ] Profile inheritance works: `voice.yaml extends default.yaml` correctly merges.
- [ ] Sandbox prevents worker from opening outbound TCP socket when `egress: deny`.
- [ ] `go test -count=1 ./internal/supervisor/... ./internal/ipc/...` passes 100%.
- [ ] Audit log `.memory/logs/supervisor.jsonl` is queryable via `jq` for forensics.
- [ ] Lock file prevents accidental double-start of `mymemoryd`.
