# Feature: interactive-html-graph-visualizer

## Problem Statement
O My-Memory atualmente oferece suporte à exportação de subgrafos para o formato aberto JSON Canvas 1.0 (`.canvas`, ADR-008). Contudo, essa experiência visual apresenta barreiras operacionais:
1. **Dependência do Ecossistema Obsidian:** Apenas usuários com o aplicativo Obsidian instalado e configurado conseguem visualizar e interagir com os arquivos `.canvas`.
2. **Ausência de Visualização Global da Memória:** Não há mecanismo para inspecionar visualmente o grafo completo do repositório/vault no navegador sem ferramentas externas.
3. **Falta de Recursos Interativos Dinâmicos:** A visualização não conta com busca de nós em tempo real, filtros dinâmicos por tipo/tags, inspeção de PageRank e exploração interativa de conexões em ambientes offline.
4. **Agentes Sem Capacidade de Renderização Visual:** Agentes de IA via MCP não conseguem gerar mapas conceituais navegáveis em HTML para apresentação rápida aos usuários humanos.

## Goals
- [ ] Criar pacote `internal/graphview` para modelagem e extração do grafo global e subgrafos locais a partir do SQLite e PostgreSQL.
- [ ] Integrar cálculo de autoridade estrutural via algoritmo PageRank ponderado (`internal/graph`), refletindo relevância nos raios dos nós.
- [ ] Implementar motor de renderização HTML5/SVG/JavaScript 100% autocontido (Zero-CDN) com simulação de forças gravitacionais (Force-Directed Graph), zoom contínuo, pan e arraste de nós.
- [ ] Implementar recursos de busca em tempo real com destaque visual, filtros por tipo de nota e painel lateral de inspeção de nós e conexões.
- [ ] Adicionar subcomando CLI `mem graph view` (com abertura no navegador padrão) e `mem graph export`.
- [ ] Adicionar flag `--html` no comando existente `mem export` para retrocompatibilidade.
- [ ] Expor nova ferramenta MCP `memory_visualize_graph` para agentes de IA.
- [ ] Registrar decisão de arquitetura na ADR-020.

## Out of Scope
- Edição bidirecional de nós e arestas diretamente no canvas HTML com persistência no banco (o visualizador é somente-leitura para observabilidade e exploração).
- Renderização 3D baseada em WebGL ou Three.js (a renderização 2D SVG/Canvas com física de forças atende com leveza e compatibilidade total).

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Autonomia Offline | Arquivo HTML 100% autocontido sem CDN | Garante funcionamento em ambientes corporativos, air-gapped e offline | y |
| Escopo de Visualização | Grafo global por padrão, ou subgrafo focado com `--root <nota>` e `--depth <N>` | Oferece tanto a visão macro do repositório quanto análise contextual de um tópico | y |
| Abertura de Browser | `mem graph view` abre automaticamente no navegador padrão do SO | Facilidade imediata de inspeção visual com um único comando | y |
| Tamanho dos Nós | Proporcional ao PageRank ponderado com base mínima de 8px até 32px | Destaca naturalmente os God Nodes / Hubs de conhecimento | y |
| Coloração de Nós | Paleta distinta baseada no tipo da nota (`concept`, `decision`, `guide`, `reference`, etc.) | Facilita a segmentação visual de conceitos arquiteturais | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Modelo Topológico e Renderização HTML Autocontida ⭐ MVP

**User Story**: As a desenvolvedor ou analista de arquitetura, I want gerar uma página HTML autocontida com a representação gráfica das notas e conexões do repositório so that eu possa explorar visualmente a teia de conhecimento no navegador sem depender de softwares proprietários.

**Why P1**: Entrega o núcleo da capacidade visual e o motor de física determinístico sem dependências de rede.

**Acceptance Criteria**:
1. The system SHALL extract all nodes and directed edges from the database into a `GraphView` structure.
2. The system SHALL calculate weighted PageRank and node degrees (`InDegree`, `OutDegree`) to scale node radius.
3. The system SHALL render a standalone HTML file embedding CSS and JavaScript force simulation without external CDN dependencies.
4. The system SHALL support dragging nodes, zooming via wheel and panning the canvas viewport.

**Independent Test**: Testes unitários em `internal/graphview/builder_test.go` e `internal/graphview/template_test.go`.

---

### P2: Interatividade Avançada, Filtros e Painel Lateral

**User Story**: As a pesquisador da base de memória, I want pesquisar termos em tempo real, filtrar por tipos de nota e inspecionar detalhes ao clicar em um nó so that eu navegue de forma fluida pelas dependências e decisões registradas.

**Why P2**: Agrega grande valor de usabilidade para exploração e análise de impacto arquitetural.

**Acceptance Criteria**:
1. WHEN a query is typed in the search input THEN the system SHALL highlight matching nodes and dim unrelated nodes.
2. WHEN note type chips are toggled THEN the system SHALL filter node visibility dynamically.
3. WHEN a node is clicked THEN the system SHALL open a sidebar displaying note title, type, tags, PageRank and in/out connections with navigation links.
4. The system SHALL provide an Obsidian deep link button (`obsidian://open?file=...`) for the selected node.

**Independent Test**: Testes unitários do template HTML validando a presença dos elementos de busca, chips e painel lateral.

---

### P3: Subcomando CLI mem graph e Integração em mem export

**User Story**: As a usuário de linha de comando, I want executar `mem graph view` para abrir o grafo no navegador ou `mem graph export` para salvar o HTML em disco so that eu integre a visualização ao meu fluxo de trabalho no terminal.

**Why P3**: Disponibiliza a interface primária de consumo humano.

**Acceptance Criteria**:
1. The system SHALL provide `mem graph view` compiling the graph and launching the default system browser.
2. The system SHALL provide `mem graph export` writing the HTML file to a specified path or default `graph.html`.
3. The system SHALL support `--html` flag in `mem export` for backward compatibility.
4. The system SHALL support `--root` and `--depth` to isolate a focused subgraph.

**Independent Test**: Testes unitários em `cmd/mem/graph_test.go`.

---

### P4: Ferramenta MCP memory_visualize_graph

**User Story**: As a agente de IA (Claude, Cursor, Antigravity), I want invocar `memory_visualize_graph` so that eu gere relatórios visuais espaciais sob demanda para o usuário durante conversas.

**Why P4**: Permite que a IA complemente respostas textuais com artefatos visuais interativos.

**Acceptance Criteria**:
1. The system SHALL register `memory_visualize_graph` in the MCP tools catalog.
2. WHEN `memory_visualize_graph` is called THEN the system SHALL generate the standalone HTML file and return file path and graph topology metrics.

**Independent Test**: Testes de integração JSON-RPC em `internal/mcp/visualizer_handlers_test.go`.

---

## Requirements (EARS)

### VIZ-01: Extração e Construção do Grafo Visual
The system SHALL provide `BuildGlobalGraph` and `BuildSubGraph` in `internal/graphview` extracting nodes, directed edges and repository metadata from SQLite and PostgreSQL stores.
- User story: P1
- Task: T1

### VIZ-02: Cálculo de Métricas Estruturais e PageRank
The system SHALL compute weighted PageRank and connection degrees for each node, calculating appropriate visual radius and color themes based on note types.
- User story: P1
- Task: T1

### VIZ-03: Renderização HTML Standalone Autocontida
The system SHALL render a single, standalone HTML5 document embedding force-directed graph simulation, SVG markers and CSS styles without any external network or CDN dependencies.
- User story: P1
- Task: T2

### VIZ-04: Recursos Interativos de Navegação e Inspeção
The system SHALL provide real-time search filtering, note type toggles, zoom/pan navigation, draggable nodes and an interactive sidebar showing note details, backlinks and Obsidian deep links.
- User story: P2
- Task: T2

### VIZ-05: Interface de Linha de Comando (mem graph e mem export)
The system SHALL provide `mem graph view`, `mem graph export` and `mem export --html` supporting `--root`, `--depth`, `--out` and `--open` options.
- User story: P3
- Task: T3

### VIZ-06: Ferramenta MCP de Visualização de Grafo
The system SHALL expose `memory_visualize_graph` via Model Context Protocol returning absolute HTML path, node counts, edge counts and key hub nodes.
- User story: P4
- Task: T4

---

## Requirement Traceability

| Requirement ID | User Story | Status | Test / Validation |
| -------------- | ---------- | ------ | ----------------- |
| VIZ-01 | P1: Modelo Topológico e Renderização HTML Autocontida | Verified | Tasks |
| VIZ-02 | P1: Modelo Topológico e Renderização HTML Autocontida | Verified | Tasks |
| VIZ-03 | P1: Modelo Topológico e Renderização HTML Autocontida | Verified | Tasks |
| VIZ-04 | P2: Interatividade Avançada, Filtros e Painel Lateral | Verified | Tasks |
| VIZ-05 | P3: Subcomando CLI mem graph e Integração em mem export | Verified | Tasks |
| VIZ-06 | P4: Ferramenta MCP memory_visualize_graph | Verified | Tasks |

**Coverage:** 6 total, 6 mapped to tasks, 0 unmapped

---

## Success Criteria
- [ ] Pacote `internal/graphview` implementado com 100% dos testes unitários passando.
- [ ] Renderizador de grafos SVG/JS standalone funcionando 100% offline (Zero-CDN).
- [ ] Subcomando `mem graph view` e `mem graph export` funcionais e testados.
- [ ] Ferramenta MCP `memory_visualize_graph` integrada e testada via JSON-RPC.
- [ ] Decisão registrada na ADR-020 e documentação atualizada.
