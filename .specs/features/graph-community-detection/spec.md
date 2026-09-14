# Feature: graph-community-detection

## Problem Statement
O My-Memory atualmente analisa o grafo de conhecimento através de métricas de centralidade individual, como grau de conexões (ADR-011) e PageRank ponderado (ADR-014). No entanto, o sistema ainda carece de uma compreensão macroestrutural da memória:
1. **Incapacidade de Detecção Modular:** Não há mecanismo para identificar automaticamente "ilhas", "módulos conceituais" ou domínios temáticos agregados que emergem das interconexões densas entre notas.
2. **Navegação Visual Desarticulada:** No visualizador interativo (ADR-020), as notas são coloridas apenas por tipo estático (`concept`, `decision`), impedindo que o desenvolvedor visualize os limites entre subsistemas de software ou tópicos de negócio.
3. **Agentes Sem Visão de Domínio:** Agentes de IA via MCP precisam navegar nota a nota ou realizar buscas abertas, sem conseguir consultar "quais são os principais clusters conceituais desta memória e seus tópicos nucleares".

## Goals
- [ ] Implementar algoritmo de detecção de comunidades ponderado (*Label Propagation Algorithm* - LPA) em `internal/graph/community.go` com complexidade linear próxima a O(V + E) e desempate determinístico.
- [ ] Calcular métrica de qualidade de partição modular (Modularidade Newman-Girvan \(Q\)) para quantificar a coesão dos clusters identificados.
- [ ] Identificar para cada comunidade o nó mais representativo (líder/hub com maior PageRank local) e o tipo dominante de nota.
- [ ] Integrar identificador e rótulo de comunidade ao modelo `internal/graphview` e adicionar alternância visual por cluster no visualizador interativo HTML/SVG.
- [ ] Implementar subcomando CLI `mem clusters` com filtros de tamanho mínimo (`--min-size`), formatação tabular e suporte a `--json`.
- [ ] Expor nova ferramenta MCP `memory_get_clusters` para agentes de IA consultarem módulos temáticos do repositório.
- [ ] Formalizar as decisões na ADR-022 e atualizar guias técnicos.

## Out of Scope
- Algoritmos estocásticos que não garantam convergência determinística para o mesmo estado do grafo (o algoritmo adotará ordenação determinística de nós).
- Modificação física nos arquivos Markdown para gravar tags de cluster (a detecção é calculada dinamicamente sobre a topologia em memória e banco).

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Algoritmo Principal | Weighted Label Propagation Algorithm (LPA) | Alta eficiência O(V+E), detecção automática de K clusters sem parametrização rígida | y |
| Desempate Determinístico | Ordem lexicográfica de identificadores de nós em caso de empate de pesos | Garante reprodutibilidade exata em testes e execuções repetidas | y |
| Representação de Cluster | Rótulo baseado no nó com maior PageRank ou grau dentro da comunidade | Facilita a identificação humana imediata do tema do cluster | y |
| Limiar Mínimo CLI | `--min-size 2` por padrão (oculta nós isolados de tamanho 1 por padrão) | Foca nos agrupamentos densos e reduz ruído visual | y |
| Cores no Visualizador | Paleta categórica dinâmica associada ao `CommunityID` | Permite alternar entre visualização por tipo de nota e por comunidade temática | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Algoritmo de Detecção de Comunidades (LPA) em Go ⭐ MVP

**User Story**: As a engenheiro de dados da memória, I want executar detecção de comunidades sobre a estrutura de nós e arestas so that o sistema identifique partições densamente conectadas de conhecimento sem requerer bibliotecas externas de grafos em C.

**Why P1**: Núcleo computacional determinístico indispensável para toda a feature.

**Acceptance Criteria**:
1. The system SHALL implement `DetectCommunities(nodes, edges, opts)` in `internal/graph` returning a list of communities with assigned members.
2. The system SHALL compute cumulative edge weights using epistemic weights (`EXTRACTED`, `INFERRED`, `TAG`) during label propagation.
3. The system SHALL apply deterministic tie-breaking based on node identifier sorting when multiple neighbor labels have identical weights.
4. The system SHALL calculate the Newman-Girvan Modularity score Q representing graph partition density.
5. IF a graph has disconnected subgraphs THEN the system SHALL partition them into separate independent communities.

**Independent Test**: Testes unitários com topologias canônicas (cliques interligados, nós isolados, anéis) em `internal/graph/community_test.go`.

---

### P2: Extração de Metadados e Integração com SQLite/Postgres

**User Story**: As a desenvolvedor consultando o repositório, I want carregar comunidades diretamente dos bancos SQLite e PostgreSQL so that os clusters reflitam a topologia real de notas do vault.

**Why P2**: Conecta o algoritmo puro com os mecanismos de persistência existentes.

**Acceptance Criteria**:
1. The system SHALL provide functions in `internal/db` and `internal/store` to extract graph topology and build community partitions.
2. The system SHALL determine the core node (highest PageRank) and dominant note type for each community.

**Independent Test**: Testes de extração em `internal/db` e `internal/graph`.

---

### P3: Subcomando CLI mem clusters

**User Story**: As a usuário no terminal, I want rodar `mem clusters` so that eu veja uma tabela com os clusters do projeto, quantidade de notas, nós centrais e membros.

**Why P3**: Interface de uso imediata para humanos inspecionarem a modularidade da base de notas.

**Acceptance Criteria**:
1. The system SHALL provide `mem clusters` displaying an aligned terminal table with cluster ID, core node, size, dominant type and top member notes.
2. The system SHALL support `--min-size <N>` filtering out clusters smaller than N nodes.
3. WHERE `--json` is passed THEN the system SHALL output structured JSON containing all communities and modularity score.

**Independent Test**: Testes do comando CLI em `cmd/mem/clusters_test.go`.

---

### P4: Ferramenta MCP e Integração com Visualizador Interativo

**User Story**: As a agente de IA ou usuário do browser, I want consultar `memory_get_clusters` via MCP e alternar cores por cluster no visualizador HTML so that a exploração visual e semântica reflita os domínios do repositório.

**Why P4**: Entrega observabilidade gráfica e integração transparente com assistentes inteligentes.

**Acceptance Criteria**:
1. The system SHALL register `memory_get_clusters` in the MCP tools catalog returning cluster summaries and modularity.
2. WHERE communities are computed THEN the system SHALL enrich `GraphView` nodes with `community_id` and assign harmonious cluster colors.
3. The system SHALL maintain ADR-022 formalizando as decisões de arquitetura de detecção de comunidades.

**Independent Test**: Testes de MCP em `internal/mcp` e teste de integração no visualizador HTML.

---

## Requirement Traceability

| Requirement ID | User Story / Feature Area | Status |
| -------------- | ------------------------- | ------ |
| COMM-01 | Algoritmo LPA ponderado em Go puro (internal/graph) | verified |
| COMM-02 | Desempate determinístico e convergência de rótulos | verified |
| COMM-03 | Cálculo de Modularidade Newman-Girvan Q | verified |
| COMM-04 | Identificação de nó central/líder e tipo dominante | verified |
| COMM-05 | Subcomando CLI mem clusters com tabela e --json | verified |
| COMM-06 | Filtro de tamanho mínimo (--min-size) | verified |
| COMM-07 | Ferramenta MCP memory_get_clusters | pending |
| COMM-08 | Integração com visualizador de grafo e ADR-022 | pending |
