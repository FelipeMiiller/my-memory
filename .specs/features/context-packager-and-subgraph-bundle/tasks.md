# Tasks: context-packager-and-subgraph-bundle

- [x] **T1**: Motor de Empacotamento de Subgrafo e Orçamento de Tokens (`internal/graph/`)
  - [x] Definir modelos de dados: `PackOptions`, `PackedNode`, `PackTier`, `PackResult`, `PackGraphNode`.
  - [x] Implementar `EstimateTokens(text string) int` com heurística determinística de contagem.
  - [x] Implementar `PackContext(nodes []PackGraphNode, edges []WeightedEdge, rootID string, opts PackOptions) (*PackResult, error)`.
  - [x] Implementar expansão BFS respeitando `MaxDepth` e `Direction` (`both`, `outbound`, `inbound`).
  - [x] Implementar ordenação prioritária por distância, PageRank e pesos epistêmicos de arestas.
  - [x] Implementar alocação gananciosa com orçamento: `TierCore` (L2) $\to$ `TierFringe` (L0/L1) $\to$ `TierOmitted`.
  - [x] Implementar gerador de Mermaid topológico e renderizador do Markdown consolidado.
  - [x] Escrever testes unitários exaustivos em `internal/graph/pack_test.go`.

- [x] **T2**: Integração de Persistência SQLite e PostgreSQL (`internal/db/` & `internal/store/`)
  - [x] Implementar `PackContext` em `internal/db/graph.go` usando `ResolveNodeCanonicalID`.
  - [x] Adicionar testes de integração em `internal/db/graph_test.go`.
  - [x] Adicionar método `PackContext` à interface `Store` em `internal/store/store.go`.
  - [x] Implementar `PackContext` em `internal/store/postgres.go`.
  - [x] Adicionar testes em `internal/store/postgres_test.go`.


- [x] **T3**: Subcomando CLI `mem pack` (`cmd/mem/`)
  - [x] Criar `cmd/mem/pack.go` com flags `--depth`, `--max-tokens`, `--direction`, `--out` e `--json`.
  - [x] Renderizar resumo no terminal (tokens consumidos, nós incluídos vs omitidos) e gravação em arquivo / stdout.
  - [x] Integrar no switch de comandos e ajuda em `cmd/mem/main.go`.
  - [x] Escrever testes de CLI em `cmd/mem/pack_test.go`.


- [x] **T4**: Ferramenta MCP `memory_pack_context` (`internal/mcp/`)
  - [x] Declarar `ToolMemoryPackContext` em `internal/mcp/tools.go` e adicionar a `AllTools`.
  - [x] Criar `internal/mcp/pack_handlers.go` com `NewMemoryPackContextHandler` e `SetPackHandler`.
  - [x] Conectar handler nos servidores stdio e HTTP/SSE em `cmd/mem/main.go`.
  - [x] Escrever testes unitários em `internal/mcp/pack_handlers_test.go`.


- [ ] **T5**: Arquitetura (ADR-029) e Documentação Operacional
  - [ ] Criar `docs/adr/029-context-packager-e-subgraph-bundle.md` no padrão MADR.
  - [ ] Atualizar catálogo em `docs/adr/README.md` e `docs/README.md`.
  - [ ] Atualizar documentação em `docs/CLI_GUIDE.md` e `docs/AGENT_INTEGRATION_GUIDE.md`.
  - [ ] Atualizar estado do projeto em `.specs/STATE.md`.
