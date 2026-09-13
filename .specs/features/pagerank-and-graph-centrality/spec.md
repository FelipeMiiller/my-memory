# Feature: pagerank-and-graph-centrality

## Problem Statement
Atualmente, a identificação de nós centrais (*God Nodes* / Hubs - ADR-011) no My-Memory baseia-se exclusivamente na contagem de conexões brutas (`total_degree = in_degree + out_degree`). Essa abordagem não considera a autoridade relativa dos nós apontadores (um link de uma nota central tem o mesmo peso de um link de uma nota marginal) e não aproveita a tipagem epistêmica de arestas (links explícitos `EXTRACTED` vs inferidos `INFERRED`). É necessário introduzir um algoritmo de PageRank iterativo ponderado para identificar os verdadeiros polos conceituais do grafo de conhecimento.

## Goals
- [ ] Implementar motor matemático de PageRank com amortecimento ($d=0.85$), redistribuição de *dangling nodes* e convergência determinística em Go puro (`internal/graph`).
- [ ] Suportar pesos epistêmicos diferenciados por tipo de aresta (`EXTRACTED` vs `INFERRED`).
- [ ] Integrar o cálculo de PageRank ao contrato `Store` para SQLite e PostgreSQL.
- [ ] Atualizar a ferramenta MCP `memory_get_hubs` com suporte ao parâmetro `algorithm: "pagerank" | "degree"`.
- [ ] Atualizar o comando CLI `mem hubs` com as flags `--algorithm`, `--damping` e `--iter`.
- [ ] Registrar a decisão arquitetural na ADR-014.

## Out of Scope
- Alteração no algoritmo RRF de busca híbrida neste momento (PageRank funcionará primeiramente na camada estrutural e MCP/CLI).
- Algoritmos de detecção de comunidades (ex: Louvain / Leiden), que pertencem a frentes futuras.

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Fator de Amortecimento ($d$) | `0.85` | Padrão da literatura que balanceia navegação aleatória e autoridade estrutural | y |
| Tolerância de Convergência | `1e-6` | Precisão suficiente para ordenação de notas com convergência em poucas iterações | y |
| Máximo de Iterações Padrão | `30` | Garante convergência estável em tempo $< 5$ms para grafos típicos de vaults | y |
| Peso Epistêmico de Arestas | EXTRACTED=1.0, INFERRED=0.6, Outros=0.3 | Arestas explícitas refletem intencionalidade humana superior à similaridade estatística | y |
| Algoritmo Padrão no CLI/MCP | `degree` | Preserva retrocompatibilidade absoluta com scripts e integrações existentes | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Motor Matemático de PageRank Ponderado ⭐ MVP

**User Story**: As a engenheiro de conhecimento, I want calcular o PageRank de um grafo direcionado ponderado so that eu possa identificar os nós com maior autoridade topológica.

**Why P1**: Fornece o núcleo computacional desacoplado para ordenação de autoridade estrutural.

**Acceptance Criteria**:
1. The system SHALL provide `ComputePageRank` in `internal/graph` returning a normalized score map for all graph nodes.
2. WHEN `ComputePageRank` executes with damping factor $d$ THEN the system SHALL redistribute authority such that dangling nodes distribute their rank uniformly across all nodes.
3. WHEN `ComputePageRank` converges THEN the system SHALL ensure the sum of all node scores equals $1.0 \pm 10^{-5}$.
4. WHERE edges declare weights THEN the system SHALL weight out-neighbor transition probabilities proportionally to edge weights.

**Independent Test**: Testes analíticos em grafo estrela, grafo em anel e grafo bipartido validando propriedades matemáticas exatas.

---

### P2: Contrato Store e Implementações SQLite e PostgreSQL

**User Story**: As a sistema My-Memory, I want consultar o PageRank de documentos armazenados so that as notas mais autoritativas possam ser recuperadas via Store.

**Why P2**: Conecta o motor analítico às bases relacionais persistentes do sistema.

**Acceptance Criteria**:
1. The system SHALL declare `ComputePageRank` method on `Store` interface returning a slice of `PageRankNode`.
2. WHEN `ComputePageRank` executes on SQLite THEN the system SHALL load active edges and nodes and return results sorted in descending order of PageRank score.
3. WHEN `ComputePageRank` executes on PostgreSQL with a repository parameter THEN the system SHALL isolate graph edges and nodes to the specified repository.
4. IF a database contains zero nodes THEN the system SHALL return an empty slice without error.

**Independent Test**: Testes unitários no SQLite e PostgreSQL validando ordenação por score e preenchimento de `Rank`, `Score`, `InDegree` e `OutDegree`.

---

### P3: Protocolo MCP memory_get_hubs com PageRank

**User Story**: As a agente de IA, I want solicitar os nós centrais via PageRank no MCP so that eu possa priorizar documentos de referência na navegação autônoma.

**Why P3**: Permite que assistentes como Claude e Cursor escolham entre hubs de volume (`degree`) ou autoridade conceitual (`pagerank`).

**Acceptance Criteria**:
1. The system SHALL support `algorithm` property in `memory_get_hubs` accepting values `degree` and `pagerank`.
2. IF `algorithm` is omitted in `memory_get_hubs` THEN the system SHALL default to `degree` calculation.
3. WHEN `algorithm` is set to `pagerank` THEN the system SHALL format output displaying PageRank score and relative percentage.

**Independent Test**: Testes JSON-RPC de `tools/call` com `name: "memory_get_hubs"` e `algorithm: "pagerank"`.

---

### P4: Interface CLI mem hubs com Suporte a PageRank

**User Story**: As a usuário no terminal, I want executar `mem hubs --algorithm pagerank` so that eu possa inspecionar visualmente os nós mais autoritativos.

**Why P4**: Oferece inspeção de arquitetura e navegação rápida no terminal.

**Acceptance Criteria**:
1. WHEN `mem hubs` is invoked with `--algorithm pagerank` THEN the system SHALL display an ASCII table including node rank, PageRank score, and degree metrics.
2. WHERE `--damping` and `--iter` flags are passed to `mem hubs` THEN the system SHALL forward customized hyperparameters to the computation.

**Independent Test**: Execução de `mem hubs --algorithm pagerank --top 5` asserindo saída tabular e código de saída 0.

---

## Edge Cases
- IF the graph contains isolated nodes with zero incoming and outgoing edges THEN the system SHALL assign them the base teleportation probability $(1-d)/N$.
- IF the graph has no edges at all THEN the system SHALL assign equal probability $1/N$ to every node.
- IF all nodes form a directed cycle ($A \to B \to C \to A$) THEN the system SHALL assign identical scores $1/3$ to all nodes.

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| RANK-01 | P1: Motor Matemático de PageRank Ponderado | Tasks | Pending |
| RANK-02 | P1: Motor Matemático de PageRank Ponderado | Tasks | Pending |
| RANK-03 | P2: Contrato Store e Implementações SQLite e PostgreSQL | Tasks | Pending |
| RANK-04 | P3: Protocolo MCP memory_get_hubs com PageRank | Tasks | Pending |
| RANK-05 | P4: Interface CLI mem hubs com Suporte a PageRank | Tasks | Pending |
| RANK-06 | P4: Interface CLI mem hubs com Suporte a PageRank | Tasks | Pending |

**Coverage:** 6 total, 6 mapped to tasks, 0 unmapped

---

## Success Criteria
- [ ] Motor `internal/graph/pagerank.go` implementado com precisão analítica.
- [ ] Ponderação epistêmica validada (`EXTRACTED` vs `INFERRED`).
- [ ] `ComputePageRank` implementado em SQLite e PostgreSQL.
- [ ] MCP `memory_get_hubs` com suporte retrocompatível a `algorithm`.
- [ ] CLI `mem hubs` atualizado com `--algorithm`, `--damping`, `--iter`.
- [ ] Registro formal na ADR-014.
