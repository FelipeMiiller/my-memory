# Feature: triptych-node-inspector

## Problem Statement
No fluxo de trabalho assistido por IA e na navegação de grafos de conhecimento, agentes e desenvolvedores frequentemente sofrem com a exploração cega (*blind exploration*), abrindo e lendo dezenas de arquivos inteiros apenas para descobrir quem os referencia e para onde eles apontam.
Inspirado na arquitetura do **CodeGraph** (Colby McHenry), a **Visualização Cirúrgica em 3 Colunas (Triptych Node Inspector)** fornece um modelo de inspeção instantâneo estruturado em três colunas espaciais:
1. **Inbound Links (Chamadores e Dependentes Reversos)**: Quem aponta para o nó, tipos de relação (`implements`, `depends_on`, `links_to`), severidade de impacto de quebra e autoridade PageRank.
2. **Nó Central (Núcleo e Risco)**: Metadados canônicos, PageRank, comunidade/cluster LPA (ADR-022), score de risco de destruição (ADR-023), tags e preview cirúrgico de conteúdo (*Zero File Reads*).
3. **Outbound Links (Referências de Saída)**: Para onde o nó aponta, status de existência (detectando *dead links* e nós órfãos) e pesos epistêmicos (`EXTRACTED` vs `INFERRED`).

## Goals
- [ ] Implementar as estruturas de dados e motor de montagem do tríptico em `internal/graph/inspector.go`.
- [ ] Implementar métodos `InspectNode` de alta performance na camada de persistência (`internal/db/` e `internal/store/`).
- [ ] Criar o subcomando de terminal `mem inspect <node_id> [--depth 1] [--json] [--full]` em `cmd/mem/inspect.go`.
- [ ] Registrar a ferramenta MCP `memory_inspect_node` para agentes consumirem o contexto cirúrgico sem ler múltiplos arquivos.
- [ ] Aprimorar o visualizador HTML/SVG standalone (`internal/graphview/template.go`) com um modal/painel interativo de 3 colunas.
- [ ] Formalizar as decisões na `ADR-024` e manter os artefatos de especificação e estado sincronizados.

## Out of Scope
- Edição direta de arquivos a partir do visualizador do terminal (a ferramenta é para inspeção e contexto cirúrgico de leitura).
- Parsing sintático específico de ASTs de código-fonte (opera sobre os nós e arestas unificados do grafo de notas e decisões).

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Direção das 3 colunas | `In-links` (Esquerda) \| `Nó Central` (Centro) \| `Out-links` (Direita) | Alinhamento espacial padrão estabelecido pelo CodeGraph | y |
| Truncamento de Preview | Padrão de 500 caracteres / primeiras linhas com opção `--full` | Economiza tokens para agentes de IA mantendo o contexto semântico chave | y |
| Resolução de Identificadores | Resolução flexível por ID, caminho de arquivo, título ou busca difusa | Ergonômico tanto para digitação humana no terminal quanto para chamadas de IA | y |
| Integração com Blast Radius | Exibição do `RiskScore` (0-100) e `RiskLevel` calculados via ADR-023 | O desenvolvedor/agente vê imediatamente o perigo de refatorar o nó inspecionado | y |
| Integração com Comunidades | Exibição do `CommunityID` e rótulo dominante via ADR-022 | Localiza o nó dentro do seu agrupamento conceitual macro | y |

---

## User Stories

### P1: Motor de Montagem do Tríptico em Go ⭐ MVP
**User Story**: As a desenvolvedor ou agente, I want uma função que agregue in-links, nó central e out-links so that eu tenha um modelo consolidado e determinístico do tríptico.
**Acceptance Criteria**:
1. WHEN `BuildTriptychView` runs THEN the model SHALL be built from the edges and metadata with ordering consistent by severity and PageRank.
2. WHEN the tríptico is built THEN the system SHALL identify outgoing links to non-existent destinations (*dangling/dead links*).
3. The unit tests with canonical topologies SHALL live in `internal/graph/inspector_test.go`.

### P2: Consulta de Inspeção no SQLite e PostgreSQL
**User Story**: As a sistema de persistência, I want consultar e enriquecer dados de um nó por ID/caminho so that as conexões e conteúdo sejam extraídos cirurgicamente.
**Acceptance Criteria**:
1. WHEN `db.InspectNode` is called on SQLite or `pgStore.InspectNode` on PostgreSQL THEN the system SHALL support flexible identifiers.
2. WHEN a node is inspected THEN the system SHALL retrieve the chunks/content, the topological metrics and the adjacent edges with precision.

### P3: Subcomando CLI mem inspect
**User Story**: As a desenvolvedor no terminal, I want executar `mem inspect <node_id>` so that eu veja o tríptico formatado ou em JSON.
**Acceptance Criteria**:
1. The `mem inspect <node_id> [--depth N] [--json] [--full]` command SHALL be supported.
2. WHEN `mem inspect` runs THEN the output SHALL show a clear tabular/block view with in-links, central metrics and out-links.

### P4: Ferramenta MCP memory_inspect_node, Visualizador HTML e ADR-024
**User Story**: As a agente de IA ou usuário web, I want chamar `memory_inspect_node` ou inspecionar no HTML standalone so that eu obtenha o contexto cirúrgico sem ler arquivos avulsos.
**Acceptance Criteria**:
1. WHEN the MCP catalog is listed THEN `memory_inspect_node` SHALL be registered with a formatted Markdown output.
2. The triptych panel/modal SHALL live in `internal/graphview/template.go`.
3. The ADR-024 SHALL be formalized.

---

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Where | Status |
| :--- | :--- | :--- | :--- |
| TNI-01 | P1 AC1-2 — modelo montado a partir das arestas e metadados, com links de saída inexistentes identificados | `internal/graph/inspector.go` | Verified |
| TNI-02 | P1 AC3 — testes unitários com topologias canônicas | `internal/graph/inspector_test.go` | Verified |
| TNI-03 | P2 AC1-2 — `InspectNode` com identificadores flexíveis, chunks, métricas e arestas adjacentes | `internal/db/` · `internal/store/` | Verified |
| TNI-04 | P3 AC1-2 — `mem inspect <node_id> [--depth N] [--json] [--full]` com saída tabular | `cmd/mem/inspect.go` | Verified |
| TNI-05 | P4 AC1 — `memory_inspect_node` no catálogo MCP com saída Markdown | `internal/mcp/inspect_handlers.go` | Verified |
| TNI-06 | P4 AC2 — painel/modal tríptico no visualizador | `internal/graphview/template.go` | Verified |
| TNI-07 | P4 AC3 — ADR-024 formalizada | `docs/adr/024-visualizacao-cirurgica-em-3-colunas-triptych-inspector.md` | Verified |

**Coverage**: 7 total, 7 mapeados, 0 unmapped.
