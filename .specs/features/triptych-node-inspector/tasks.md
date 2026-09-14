# Tasks: triptych-node-inspector

- [x] **T1**: Implementação do Motor de Montagem do Tríptico (`internal/graph/inspector.go` e `inspector_test.go`)
  - [x] Definir modelos `TriptychView`, `InboundLink`, `OutboundLink`, `NodeSummary`, `InspectorOptions`.
  - [x] Implementar `BuildTriptychView` com ordenação por severidade, autoridade PageRank e detecção de links quebrados.
  - [x] Escrever testes unitários cobrindo grafos cíclicos, nós isolados e links dangling.

- [x] **T2**: Integração de Consulta e Enriquecimento na Persistência (`internal/db/` e `internal/store/`)
  - [x] Implementar `InspectNode` em `internal/db/graph.go` com resolução canônica de ID/título/path e leitura de preview de conteúdo.
  - [x] Implementar `InspectNode` em `internal/store/postgres.go` para suporte a PostgreSQL multi-repo.
  - [x] Testes de integração em `internal/db/graph_test.go` e `internal/store/postgres_test.go`.

- [x] **T3**: Subcomando CLI mem inspect (`cmd/mem/inspect.go` e `inspect_test.go`)
  - [x] Implementar `runInspectCommand` com flags `--depth`, `--json`, `--full`, `--db`, `--postgres`, `--repo`.
  - [x] Renderizador em formato de 3 colunas / blocos de tríptico com badges ANSI/Unicode.
  - [x] Registrar comando no switch de comandos em `cmd/mem/main.go`.
  - [x] Testes automatizados de CLI com captura de stdout.

- [x] **T4**: Ferramenta MCP memory_inspect_node, Visualizador HTML e ADR-024
  - [x] Definir `ToolMemoryInspectNode` em `internal/mcp/tools.go`.
  - [x] Implementar `NewMemoryInspectNodeHandler` e `SetInspectHandler` em `internal/mcp/inspect_handlers.go`.
  - [x] Conectar o handler nos despachantes do `cmd/mem/main.go` para SQLite e Postgres.
  - [x] Escrever testes unitários em `internal/mcp/inspect_handlers_test.go`.
  - [x] Aprimorar `internal/graphview/template.go` para exibir o layout 3-colunas no visualizador web interativo.
  - [x] Criar `docs/adr/024-visualizacao-cirurgica-em-3-colunas-triptych-inspector.md` e atualizar índices.
