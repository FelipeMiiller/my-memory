# Tasks: graph-path-discovery

- [x] **T1**: Motor de Pathfinding com Dijkstra e Ponderação Epistêmica (`internal/graph/`)
  - [x] Definir modelos de dados: `PathOptions`, `CostMode`, `PathEdge`, `PathResult`.
  - [x] Implementar fila de prioridade com `container/heap` para algoritmo Dijkstra.
  - [x] Implementar `FindPath(nodes []string, edges []WeightedEdge, sourceID, targetID string, opts PathOptions) (*PathResult, error)`.
  - [x] Suportar cálculo de custos por modo:
    - `epistemic`: $c = 1.0 / w$ (`EXTRACTED` = 1.0, `INFERRED` = 1.67, `TAG` = 3.33).
    - `hops`: $c = 1.0$ fixo por aresta.
  - [x] Suportar modos direcionado (`Directed: true`) e bidirecional (`Directed: false`).
  - [x] Respeitar limite de profundidade `MaxDepth` e garantir terminação em grafos cíclicos.
  - [x] Escrever testes unitários exaustivos em `internal/graph/path_test.go`.

- [x] **T2**: Integração de Persistência SQLite e PostgreSQL (`internal/db/` & `internal/store/`)
  - [x] Implementar `FindPath` em `internal/db/graph.go` usando `ResolveNodeCanonicalID`.
  - [x] Adicionar testes de integração em `internal/db/graph_test.go`.
  - [x] Adicionar método `FindPath` à interface `Store` em `internal/store/store.go`.
  - [x] Implementar `FindPath` em `internal/store/postgres.go`.
  - [x] Adicionar testes em `internal/store/postgres_test.go`.

- [x] **T3**: Subcomando CLI `mem path` (`cmd/mem/`)
  - [x] Criar `cmd/mem/path.go` com flags `--directed`, `--undirected`, `--max-depth`, `--mode` e `--json`.
  - [x] Renderizar saída amigável com ASCII path (`[A] --(rel [status])--> [B]`) e tabela resumo de custos.
  - [x] Integrar no switch de comandos e ajuda em `cmd/mem/main.go`.
  - [x] Escrever testes de CLI em `cmd/mem/path_test.go`.

- [x] **T4**: Ferramenta MCP `memory_find_path` (`internal/mcp/`)
  - [x] Declarar `ToolMemoryFindPath` em `internal/mcp/tools.go` e adicionar a `AllTools`.
  - [x] Criar `internal/mcp/path_handlers.go` com `NewMemoryFindPathHandler` e `SetPathHandler`.
  - [x] Conectar handler nos servidores stdio e HTTP/SSE em `cmd/mem/main.go`.
  - [x] Escrever testes unitários em `internal/mcp/path_handlers_test.go`.

- [x] **T5**: Arquitetura (ADR-027) e Documentação Operacional
  - [x] Criar `docs/adr/027-descoberta-de-rotas-e-caminho-minimo-no-grafo.md` no padrão MADR.
  - [x] Atualizar catálogo em `docs/adr/README.md` e `docs/README.md`.
  - [x] Atualizar documentação em `docs/CLI_GUIDE.md` e `docs/AGENT_INTEGRATION_GUIDE.md`.
  - [x] Atualizar estado do projeto em `.specs/STATE.md`.
