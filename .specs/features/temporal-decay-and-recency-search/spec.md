# Feature: temporal-decay-and-recency-search

## Problem Statement
O My-Memory armazena a data de modificação (`updated_at` em segundos epoch) de cada documento Markdown indexado. Contudo, a busca híbrida com Reciprocal Rank Fusion (RRF - ADR-009) avalia exclusivamente a relevância textual (BM25/FTS5), vetorial e estrutural do grafo, atribuindo a mesma pontuação a documentos recém-atualizados e documentos estáticos de anos atrás. Em bases de anotações dinâmicas e vaults de conhecimento, notas recentes ou recentemente revisadas frequentemente contêm respostas mais oportunas para consultas de trabalho. É necessário introduzir um modelo matemático de decaimento temporal exponencial com controle de meia-vida (*half-life*) e peso de modulação sem prejudicar notas clássicas perenes.

## Goals
- [ ] Implementar a função matemática de decaimento temporal exponencial parametrizada por meia-vida e peso de influência em `internal/store/rrf.go`.
- [ ] Enriquecer `SearchResult` com o timestamp `UpdatedAt` nos backends SQLite e PostgreSQL.
- [ ] Implementar `FuseSearchResultsWithDecay` integrando o multiplicador temporal à pontuação RRF.
- [ ] Expor o método `SearchHybridRRFWithDecay` no contrato `Store`.
- [ ] Adicionar os parâmetros `decay`, `half_life` e `decay_weight` na ferramenta MCP `memory_search`.
- [ ] Adicionar as flags `--decay`, `--half-life` e `--decay-weight` no comando CLI `mem search`.
- [ ] Registrar a decisão arquitetural na ADR-015.

## Out of Scope
- Alteração no algoritmo de similaridade cosseno ou no quantizador TurboQuant (o decaimento atua na fusão de rankings, não no embedding).
- Reordenação cronológica pura (o decaimento é um modulador suave de relevância, não uma ordenação `ORDER BY updated_at`).

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Comportamento Padrão | Desativado (`decay = false`) | Garante preservação estrita do ranking RRF existente sem surpresas | y |
| Meia-Vida Padrão ($T_{half}$) | `30.0` dias | Janela representativa para ciclos de atualização em notas e projetos | y |
| Peso de Modulação ($w$) | `0.3` | Reserva 70% do score para relevância semântica e 30% para recência | y |
| Piso de Preservação | Multiplicador $\ge (1 - w)$ | Garante que notas antigas nunca sejam penalizadas a zero | y |
| Notas sem Data (`updated_at == 0`) | Consideradas com decaimento pleno | Evita exceções e trata documentos sem metadados como arquivados | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Motor Matemático de Decaimento Temporal e RRF ⭐ MVP

**User Story**: As a sistema de busca, I want modular a pontuação RRF com base na recência do documento so that notas recentemente modificadas ganhem destaque proporcional.

**Why P1**: Fornece a formulação matemática desacoplada para ponderação temporal de resultados.

**Acceptance Criteria**:
1. The system SHALL provide `CalculateTimeDecay` calculating the exponential multiplier $(1-w) + w \cdot 2^{-\Delta t / T_{half}}$ for a given delta time, half-life, and weight.
2. WHEN `updated_at` equals the reference time ($\Delta t = 0$) THEN the system SHALL return a decay multiplier of exactly $1.0$.
3. WHEN $\Delta t$ equals the half-life duration THEN the system SHALL return a decay multiplier of exactly $1.0 - 0.5w$.
4. WHEN `FuseSearchResultsWithDecay` executes with `DecayOptions.Enabled = true` THEN the system SHALL multiply each fused item's score by its temporal multiplier.
5. IF `DecayOptions.Enabled` is false THEN the system SHALL preserve identical RRF scores to standard `FuseSearchResults`.

**Independent Test**: Testes analíticos em `internal/store/decay_test.go` validando multiplicadores exatos para $\Delta t = 0$, $\Delta t = T_{half}$ e inversão de ranking entre notas com diferentes datas de modificação.

---

### P2: Enriquecimento de Resultados com UpdatedAt no Store

**User Story**: As a desenvolvedor ou agente, I want que os trechos retornados nas buscas contenham o timestamp de modificação da nota de origem so that a recência possa ser auditada.

**Why P2**: Conecta os metadados relacionais de documentos às buscas vetoriais e de texto completo.

**Acceptance Criteria**:
1. The system SHALL include `UpdatedAt` field in `SearchResult` struct in `internal/store`, `internal/db`, and `internal/mcp`.
2. WHEN `SearchFTS` or `SearchKNN` executes in SQLite THEN the system SHALL populate `UpdatedAt` from the corresponding document's `updated_at` column.
3. WHEN `SearchFTS` or `SearchKNN` executes in PostgreSQL THEN the system SHALL populate `UpdatedAt` from `documents.updated_at`.
4. The system SHALL declare and implement `SearchHybridRRFWithDecay` on the `Store` interface.

**Independent Test**: Testes de busca no SQLite e PostgreSQL verificando preenchimento correto de `UpdatedAt` a partir de notas com datas conhecidas.

---

### P3: Protocolo MCP memory_search com Suporte a Decaimento

**User Story**: As a agente de IA, I want solicitar busca com decaimento temporal no MCP so that eu possa priorizar informações recentes do repositório.

**Why P3**: Permite que LLMs consultem notas recentes ativando `decay: true`.

**Acceptance Criteria**:
1. The system SHALL support optional `decay`, `half_life`, and `decay_weight` properties in `memory_search` MCP tool schema.
2. IF `decay` is omitted or false in `memory_search` THEN the system SHALL execute standard hybrid search without time decay.
3. WHEN `decay` is true in `memory_search` THEN the system SHALL forward `DecayOptions` to `SearchHybridRRFWithDecay`.

**Independent Test**: Testes JSON-RPC de `tools/call` em `internal/mcp/handlers_test.go` verificando chamada de `memory_search` com `decay: true`.

---

### P4: Interface CLI mem search com Flags de Recência

**User Story**: As a usuário no terminal, I want executar `mem search --decay --half-life 15` so that eu possa realizar buscas priorizando notas recentes.

**Why P4**: Oferece controle flexível de hiperparâmetros de recência no terminal.

**Acceptance Criteria**:
1. WHEN `mem search` is invoked with `--decay` THEN the system SHALL execute hybrid search applying time decay.
2. WHERE `--half-life` and `--decay-weight` flags are passed THEN the system SHALL configure the corresponding decay hyperparameters.

**Independent Test**: Execução de `mem search --decay --half-life 30 "termo"` no terminal com código de saída 0.

---

## Edge Cases
- IF a document's `updated_at` is in the future ($\Delta t < 0$) due to clock skew THEN the system SHALL clamp $\Delta t$ to $0$, applying a multiplier of $1.0$ without overflow.
- IF a document's `updated_at` is zero or negative THEN the system SHALL treat $\Delta t$ as infinite, applying the floor multiplier $(1 - w)$.
- IF `half_life` is non-positive ($T_{half} \le 0$) THEN the system SHALL fallback to the default half-life of 30 days.

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| DECAY-01 | P1: Motor Matemático de Decaimento Temporal e RRF | Tasks | Pending |
| DECAY-02 | P1: Motor Matemático de Decaimento Temporal e RRF | Tasks | Pending |
| DECAY-03 | P2: Enriquecimento de Resultados com UpdatedAt no Store | Tasks | Pending |
| DECAY-04 | P2: Enriquecimento de Resultados com UpdatedAt no Store | Tasks | Pending |
| DECAY-05 | P3: Protocolo MCP memory_search com Suporte a Decaimento | Tasks | Pending |
| DECAY-06 | P4: Interface CLI mem search com Flags de Recência | Tasks | Pending |

**Coverage:** 6 total, 6 mapped to tasks, 0 unmapped

---

## Success Criteria
- [ ] `CalculateTimeDecay` e `FuseSearchResultsWithDecay` implementados e cobertos por testes unitários.
- [ ] `SearchResult.UpdatedAt` populado no SQLite e PostgreSQL.
- [ ] `SearchHybridRRFWithDecay` implementado no contrato `Store`.
- [ ] Parâmetros `decay`, `half_life` e `decay_weight` operacionais no MCP `memory_search`.
- [ ] Flags `--decay`, `--half-life` e `--decay-weight` disponíveis no CLI `mem search`.
- [ ] Registro de decisão arquitetural na ADR-015.
