# ADR-043: Envelope de Eventos Canônico (event_runtime)

- **Date**: 2026-09-19
- **Status**: Proposed
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: events, event-driven, envelope, outbox, replay, idempotency, observability

## Context and Problem Statement

Com a introdução de voz, agente runtime e projeções assíncronas (ADR-042), o MyMemory precisa de um **barramento interno de eventos** que conecte:

- Captura de áudio (VAD/wake/ASR)
- Sessão conversacional (turnos, cancelamento, barge-in)
- Tool calls do agente (policy + approval)
- Escrita de memória (writer atômico)
- Projeções (SQLite/FTS5/grafo/vetor/stats/drift)
- Viewer (timeline, diffs, aprovações)
- Auditoria (logs, replay forense)

Hoje cada subsistema se comunica de forma ad-hoc (chamada direta de função, channel Go in-process, mutex compartilhado). Isso funciona para o CLI síncrono, mas **não escala** quando:

1. Um agente dispara uma tool call que precisa ser autorizada antes de executar (precisa de aprovação assíncrona).
2. Uma transcrição final precisa acionar reindexação sem bloquear o áudio do próximo turno.
3. Um crash durante uma tool call precisa ser retomável sem re-executar efeitos colaterais.
4.Um auditor precisa reconstruir **exatamente** o que aconteceu em uma sessão de 30 minutos.

A pesquisa de 2026-09-18 define um envelope JSON canônico (§4.1) com `event_id`, `schema_version`, `event_type`, `monotonic_ns`, `correlation_id`, `causation_id`, `actor`, `payload` etc. Falta decidir o **transporte** e a **persistência** desse envelope.

## Decision Drivers

- **DR-1**: Replay nativo — `mymemoryd replay --since <seq>` reconstrói estado a partir do event log.
- **DR-2**: Outbox pattern — eventos são gravados na mesma transação do efeito (ADR-044), garantindo que commit só é visível depois do ACK do envelope.
- **DR-3**: Idempotência — cada evento tem `event_id` único + `sequence` monotônico; consumers deduplicam por `(producer, event_id)`.
- **DR-4**: Correlação cross-subsistema — `correlation_id` (request) + `causation_id` (evento-pai) permitem reconstruir DAG causal.
- **DR-5**: Crash-safety — log durável em SQLite WAL (já é o storage unificado, ADR-001); mesma transação do efeito.
- **DR-6**: Performance — broadcast in-process via channels Go; >10k eventos/s em SQLite WAL local é trivial.
- **DR-7**: Versionamento — `schema_version` em todo envelope; consumers rejeitam versão desconhecida.
- **DR-8**: Simplicidade — MVP não precisa de cluster. Escalar horizontalmente é defer para Fase 5 (NATS/Redis opcional).

## Considered Options

- **Opção A: in-process em Go + log SQLite WAL (event log no mesmo DB operacional)**
- **Opção B: NATS JetStream externo**
- **Opção C: Redis Streams externo**
- **Opção D: ZeroMQ + log custom**

## Decision Outcome

Chosen option: **"Opção A: in-process em Go + log SQLite WAL"**, porque (1) mantém ADR-001 (SQLite unificado), (2) evita dependência externa para o MVP, (3) garante outbox atômico via mesma transação, (4) replay nativo sem infra adicional, (5) Fase 5 pode trocar o transporte sem mudar o envelope (cliente NATS/Redis é adapter).

### Envelope canônico (JSON)

```json
{
  "event_id": "uuid-v7",
  "schema_version": 1,
  "event_type": "memory.committed",
  "producer": "mymemory-writer",
  "created_at": "2026-09-19T00:00:00Z",
  "monotonic_ns": 1234567890,
  "session_id": "session-abc",
  "conversation_id": "conv-xyz",
  "aggregate_id": "documents/2026-09-19-arquitetura-autoral.md",
  "epoch": 2,
  "sequence": 17,
  "correlation_id": "request-uuid",
  "causation_id": "event-uuid",
  "actor": "user:felipe",
  "locale": "pt-BR",
  "priority": "normal",
  "content_type": "application/json",
  "payload": {},
  "provenance": {
    "tool_call_id": "tool-uuid",
    "model_run_id": "run-uuid",
    "approval_id": "approval-uuid"
  },
  "trace": {
    "span_id": "...",
    "parent_span_id": "..."
  }
}
```

### Eventos de voz (subset)

```text
session.created
audio.started
audio.chunk
audio.stopped
vad.started
vad.stopped
wake.detected
stt.started
stt.partial
stt.final
intent.proposed
intent.resolved
run.started
model.delta
tool.proposed
approval.required
tool.started
tool.progress
tool.finished
tts.started
tts.chunk
tts.synthesized
playback.started
playback.interrupted
playback.finished
turn.completed
session.cancelled
session.failed
```

Áudio usa payload binário (framing Wyoming-inspired) — eventos de áudio são separados dos eventos de controle.

### Camadas do event_runtime

| Camada | Responsabilidade |
|---|---|
| **Producer** | Emite envelope, atribui `event_id`/`sequence`, persiste no outbox na mesma transação do efeito. |
| **Event Log** | Tabela `event_log` em SQLite WAL: `(sequence PRIMARY KEY, event_id UNIQUE, payload, headers, created_at)`. |
| **Dispatcher** | Lê do log em ordem, entrega a subscribers registrados, gerencia ACK/NACK/retry. |
| **Subscriber** | Filtra por `event_type`, executa handler, ACK ao terminar; NACK + backoff em falha. |
| **Outbox relay** | Se processo caiu entre commit e ACK, republica sem duplicar efeito (idempotência por `event_id`). |
| **Replay tool** | `mymemoryd replay --since <seq>` re-emite eventos do log para subscribers dry-run. |

### Garantias

- **At-least-once delivery**: subscribers idempotentes (checam `event_id` antes de aplicar efeito).
- **Ordem causal**: `sequence` monotônico por `aggregate_id`; consumers globais respeitam `monotonic_ns`.
- **Durabilidade**: WAL + checkpoint; replay até último sequence confirmado.
- **Backpressure**: `MaxAckPending` por subscriber (padrão: 64); producer pausa se log > limite.

### Negative Consequences

- **Replay manual para escalar**: por padrão, single-node. Mitigação: Fase 5 introduz adapter NATS/Redis, mantendo envelope.
- **SQLite WAL tem limite de escrita**: ~10k tx/s em hardware comum. Mitigação: sharding por aggregate_id se necessário (defer).
- **Debugging mais sutil**: trace cross-subsistema exige correlação manual. Mitigação: `mem trace <correlation_id>` reconstrói DAG.

## Pros and Cons of the Options

### Opção A: in-process Go + SQLite WAL ✅ Chosen

- ✅ Zero dependência externa; aproveita ADR-001.
- ✅ Outbox na mesma transação do efeito (atomicidade real).
- ✅ Replay nativo via `SELECT * FROM event_log WHERE sequence > ?`.
- ✅ Migração para transporte externo é adapter, não rewrite.
- ❌ Single-node por padrão (defer escala).
- ❌ Debugging cross-process exige correlação (mitigável).

### Opção B: NATS JetStream externo

- ✅ Cluster nativo, replay, consumer groups, pull consumers.
- ✅ Padrão forte em event-driven (CNCF).
- ❌ Dependência operacional pesada para MVP.
- ❌ Outbox precisa de two-phase (escrita no SQLite + publish no NATS).
- ❌ Documento de pesquisa §3.9 lista como "defer após benchmark".

### Opção C: Redis Streams

- ✅ Simples, XADD/XREAD/XACK.
- ❌ Licença do Redis Server (SSPL desde 7.4) — incompatível com algumas distribuições.
- ❌ Sem replay durável equivalente a JetStream.
- ❌ Memória RAM (custo em vaults grandes).

### Opção D: ZeroMQ + log custom

- ✅ Brokerless, leve.
- ❌ Sem cursor/ACK nativo — reconstruiríamos JetStream do zero.
- ❌ Documento de pesquisa §3.9 alerta: "o core não fornece cursor, ACK ou log durável."

## Implementation Notes (Fase 0 + Fase 1 do roadmap)

1. `internal/event_runtime/envelope.go` — struct `Envelope` + JSON schema.
2. `internal/event_runtime/log.go` — writer/reader sobre `event_log` (SQLite WAL).
3. `internal/event_runtime/dispatcher.go` — fan-out + ACK + retry + backoff.
4. `internal/event_runtime/outbox.go` — helper para atomicidade transacional.
5. Subscrições built-in: `projection.sqlite`, `projection.fts`, `projection.graph`, `projection.vec`, `viewer.timeline`, `audit.log`.
6. CLI: `mem events tail`, `mem events inspect <id>`, `mem replay --since <seq>`, `mem trace <correlation_id>`.
7. Testes de invariante: ordem causal, idempotência, replay produz mesmo estado final.

## Links

- **ADR-001**: SQLite como Camada Unificada — `event_log` mora no mesmo DB.
- **ADR-018**: Compile-not-Retrieve — escrita atômica (refinado por ADR-044 com outbox).
- **ADR-042**: `mymemoryd` — núcleo que hospeda event_runtime.
- **ADR-044**: Memory Writer + Outbox — outbox na mesma transação.
- [ADR-050: Threat Model OWASP LLM](050-threat-model-owasp-llm-aplicado-ao-mymemory.md) — auditoria via event log.
- **Documento de pesquisa** `pesquisa-infraestrutura-autoral-mymemory.md` §4.1 (envelope) e §4.2 (eventos de voz).
- Referências externas: [NATS JetStream](https://docs.nats.io/concepts/jetstream), [Wyoming](https://github.com/OHF-Voice/wyoming) (framing binário p/ áudio), [pi-mono](https://github.com/badlogic/pi-mono) (eventos de execução serializáveis).
