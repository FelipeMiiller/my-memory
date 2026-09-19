# ADR-044: Memory Writer Atômico, Outbox e Reprojeção Idempotente

- **Date**: 2026-09-19
- **Status**: Proposed
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: writer, atomicity, outbox, idempotency, projection, markdown, crash-safety

## Context and Problem Statement

O **ADR-018** introduziu o padrão Compile-not-Retrieve e as tools MCP `memory_write_note`, `memory_append_section`, `memory_compile_note`, todas mediadas pelo `SyncEngine` que dispara reindexação **síncrona** após cada write. Esse modelo funcionou para a fase atual (CLI/MCP pontual), mas tem três fragilidades estruturais que ficam críticas quando voz e agente runtime entram em cena:

1. **Sem crash-safety**: se `mem note create` grava o `.md` mas o processo morre **antes** de a indexação completar, o arquivo existe mas o SQLite/grafo/vetor não refletem o conteúdo. Próximo `mem index` corrige, mas o estado intermediário é inconsistente.

2. **Sem idempotência forte**: se o mesmo tool_call for retentado (network blip, timeout), há risco de duplicar arestas no grafo ou chunks no FTS. O `content_hash` da ADR-010 cobre indexação, mas não cobre o ato de escrever no disco.

3. **Sem precondition de versão**: dois agentes podem tentar editar o mesmo `.md` simultaneamente; o último a gravar vence silenciosamente, sem detecção de conflito.

A pesquisa de 2026-09-18 (§4.3) define o padrão desejado: `memory.write.requested → validação + precondition + policy → Markdown atomicamente gravado → memory.committed {revision_id, hash, anchors} → projeções SQLite/FTS/grafo/vetor`. O **outbox** garante que a publicação do evento de commit não se perca se o processo cair entre o commit e o ACK (ADR-043).

## Decision Drivers

- **DR-1**: Crash-safety — se o processo morre após gravar o `.md` e antes do ACK, próximo startup detecta e reprocessa.
- **DR-2**: Idempotência — replay do event log produz o mesmo estado final (sem duplicar chunks, edges, embeddings).
- **DR-3**: Precondition de versão — `expected_revision` em todo write; conflito detectado e sinalizado via `conflict.detected` event.
- **DR-4**: Atomicidade transacional — `.md` no disco + `event_log` no mesmo ponto de commit lógico (eventual consistency via outbox).
- **DR-5**: Projeções reconstruíveis — qualquer projeção (SQLite/FTS/grafo/vetor/stats) pode ser refeita do zero a partir do event log + Markdown canônico.
- **DR-6**: Compatibilidade retroativa — `mem note create`, `mem note append`, `mem compile` continuam funcionando; `MemoryWriteNote`, `MemoryAppendSection`, `MemoryCompileNote` MCP tools ganham precondition opcional.
- **DR-7**: Auditoria — todo write registra `actor`, `provenance`, `correlation_id`, `revision_id`, `content_hash`, `created_at`.

## Considered Options

- **Opção A: Outbox SQLite + event_runtime + projeções idempotentes + precondition por revision**
- **Opção B: Write síncrono + delta de projeções (status quo do ADR-018)**
- **Opção C: CRDT/merge automático no Markdown**
- **Opção D: Banco operacional separado do vault (write-through cache)**

## Decision Outcome

Chosen option: **"Opção A: Outbox + event_runtime + projeções idempotentes + precondition"**, porque endereça as três fragilidades estruturais (crash-safety, idempotência, conflito) com baixo custo e prepara o terreno para voz/agente sem mudar o contrato externo do CLI/MCP.

### Fluxo canônico de write

```text
memory.write.requested
    ├─ 1. Validar schema (frontmatter, wikilinks, paths)
    ├─ 2. Checar precondition (expected_revision == current_revision)
    │     └─ conflito → memory.write.rejected {reason: "conflict", current_revision}
    ├─ 3. Checar policy (policy-engine ADR-042: actor, scope, risco)
    │     └─ risco alto → approval.required → memory.write.deferred
    ├─ 4. BEGIN TRANSACTION
    │     ├─ Gravar .md atomicamente (tmp + rename)
    │     ├─ Calcular content_hash (SHA-256, ADR-010)
    │     ├─ Inserir em event_log (envelope memory.committed)
    │     └─ Atualizar revision counter (documents.revision)
    │     COMMIT
    ├─ 5. Dispatcher entrega memory.committed aos subscribers
    │     ├─ projection.sqlite: upsert document + chunks + fts
    │     ├─ projection.graph: extrair edges + adicionar/remover
    │     ├─ projection.vec: gerar embedding (ADR-035) + upsert
    │     ├─ projection.stats: atualizar métricas
    │     └─ viewer.timeline: append no feed
    └─ 6. ACK ao caller (tool_call ou CLI)
```

### Garantias

- **Atomicidade do commit**: `.md` + `event_log` na **mesma transação SQLite** (WAL). Se a gravação do `.md` falhar, nada é publicado. Se a publicação no event_log falhar, a transação inteira faz rollback.
- **Outbox**: dispatcher lê do `event_log` em ordem. Se processo cair entre commit e ACK, próximo startup retoma do último sequence confirmado. Subscribers idempotentes (checam `event_id` antes de aplicar efeito).
- **Precondition**: header `If-Match: revision=N` em toda chamada de write. Conflito gera `409 Conflict` + evento `conflict.detected` (não corrompe estado).
- **Idempotência**: `revision_id` único por commit; projeção upsert por `(document_id, revision_id)`. Replay produz mesmo estado.

### Componentes

| Componente | Responsabilidade |
|---|---|
| `memory-writer` | Validação + precondition + commit transacional + outbox. |
| `precondition-checker` | Compara `expected_revision` com `documents.revision` (read Committed). |
| `policy-engine` | ADR-042 — escopo, risco, approval. |
| `projection.sqlite` | Subscriber idempotente: upsert em `documents`, `chunks`, `chunks_fts`. |
| `projection.graph` | Subscriber idempotente: diff edges (delete old + insert new). |
| `projection.vec` | Subscriber idempotente: gera embedding, upsert em `chunks_vec`. |
| `viewer.timeline` | Subscriber: append no feed do viewer. |
| `audit.log` | Subscriber: append em log estruturado para `mem trace`. |

### Schema additions

```sql
-- event_log (ADR-043)
CREATE TABLE event_log (
  sequence INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id TEXT UNIQUE NOT NULL,
  schema_version INTEGER NOT NULL,
  event_type TEXT NOT NULL,
  payload BLOB NOT NULL,           -- JSON do envelope
  headers BLOB NOT NULL,           -- correlation_id, causation_id, etc.
  aggregate_id TEXT NOT NULL,
  revision INTEGER,
  created_at TEXT NOT NULL,
  acked_at TEXT
);

-- documents (refinamento)
ALTER TABLE documents ADD COLUMN revision INTEGER NOT NULL DEFAULT 0;
ALTER TABLE documents ADD COLUMN last_event_id TEXT;

-- projection cursor (idempotência)
CREATE TABLE projection_cursor (
  projection_name TEXT PRIMARY KEY,
  last_sequence INTEGER NOT NULL,
  updated_at TEXT NOT NULL
);
```

### Negative Consequences

- **Mais uma camada**: outbox + dispatcher + subscribers adicionam complexidade. Mitigação: testes de invariante cobrem ordem causal, idempotência, replay; primeira projeção entregue é `projection.sqlite` (mais simples).
- **Latência ligeiramente maior**: write passa por transação adicional (event_log). Mitigação: WAL é local; overhead <5ms em hardware comum.
- **Projeções inicialmente mais lentas**: subscriber `projection.vec` ainda depende do modelo (ADR-035). Mitigação: `projection.vec` é opt-in na Fase 1; FTS e grafo são obrigatórios.
- **Refinamento do ADR-018**: este ADR **estende**, não substitui. As tools MCP/CLI permanecem, mas ganham header `If-Match` opcional.

## Pros and Cons of the Options

### Opção A: Outbox + event_runtime + idempotência + precondition ✅ Chosen

- ✅ Crash-safety real (transação única).
- ✅ Replay produz estado idêntico (idempotência).
- ✅ Detecta conflitos de versão (precondition).
- ✅ Compatível com o envelope do ADR-043.
- ✅ Projeções reconstruíveis do event log.
- ❌ Complexidade operacional maior (subscribers, projection_cursor).

### Opção B: Write síncrono + delta (status quo ADR-018)

- ✅ Mais simples.
- ❌ Sem crash-safety — crash entre write e index deixa estado inconsistente.
- ❌ Sem idempotência forte — replay duplica efeitos.
- ❌ Sem precondition — edições concorrentes silenciosamente sobrescrevem.

### Opção C: CRDT/merge automático

- ✅ Resolve conflitos sem coordenação.
- ❌ Overhead alto para escrita canônica (Markdown humano, não é dado replicado).
- ❌ Conflito semântico (qual versão do texto prevalece) não tem resposta automática.

### Opção D: Banco operacional separado do vault (write-through cache)

- ✅ Latência de leitura menor.
- ❌ Quebra ADR-005 (Markdown é source-of-truth).
- ❌ Adiciona dependência operacional (banco extra).

## Implementation Notes (Fase 1 do roadmap)

1. `internal/writer/writer.go` — orquestra write + outbox.
2. `internal/writer/precondition.go` — version check.
3. `internal/writer/projection/` — subscribers (sqlite, fts, graph, vec).
4. Migração `0007_writer_outbox.sql` — tabelas `event_log`, `projection_cursor`, colunas em `documents`.
5. Refinar `internal/compiler/` (ADR-018) para emitir outbox em vez de chamar `SyncEngine` direto.
6. CLI ganha `mem write --if-match <revision>` (default: ignorar = sempre gravar; `--strict` exige match).
7. Testes: `TestWrite_CommitIsAtomic`, `TestWrite_IdempotentReplay`, `TestWrite_PreconditionConflict`, `TestWrite_CrashBetweenCommitAndDispatch`.

## Links

- **ADR-001**: SQLite como Camada Unificada — `event_log` mora aqui.
- **ADR-005**: Markdown com Wikilinks como Fonte de Verdade — `.md` é canônico.
- **ADR-010**: Cache Incremental com SHA-256 — `content_hash` continua sendo a chave de invalidação.
- **ADR-018**: Compile-not-Retrieve — base, **refinado por este ADR** (write passa a ser outbox + projection).
- **ADR-042**: `mymemoryd` — hospeda writer, dispatcher e subscribers.
- **ADR-043**: Envelope de Eventos — formato canônico do `event_log`.
- [ADR-050: Threat Model OWASP LLM](050-threat-model-owasp-llm-aplicado-ao-mymemory.md) — auditoria via event log.
- **Documento de pesquisa** `pesquisa-infraestrutura-autoral-mymemory.md` §4.3 (commit canônico) e §5 (security: redaction, retenção).
- Referências externas: [LangGraph checkpoints](https://github.com/langchain-ai/langgraph), [Graphiti temporal graphs](https://github.com/getzep/graphiti) (validade/invalidação não destrutiva).
