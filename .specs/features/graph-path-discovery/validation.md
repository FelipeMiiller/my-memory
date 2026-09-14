# Validation Plan: graph-path-discovery

## 1. Testes Automatizados Unitários e de Integração

### Algoritmo de Pathfinding (`internal/graph/`)
- [x] `TestFindPath_IdenticalSourceAndTarget`: Valida que rota entre um nó e ele mesmo retorna 0 saltos, custo 0.0 e `found: true`.
- [x] `TestFindPath_NodeNotFound`: Valida erro apropriado ao consultar nós inexistentes.
- [x] `TestFindPath_Unreachable`: Valida retorno `found: false` para componentes desconexos.
- [x] `TestFindPath_LinearChain`: Valida rota direta $A \to B \to C$ com saltos e sequência de nós exata.
- [x] `TestFindPath_EpistemicCostTieBreaker`: Valida que entre um caminho de 2 saltos via `EXTRACTED` (custo $1.0 + 1.0 = 2.0$) e um de 1 salto via `TAG` (custo $3.33$), o algoritmo prioriza o caminho mais confiável.
- [x] `TestFindPath_DirectedVsUndirected`: Valida que arestas no sentido oposto são ignoradas em modo direcionado e transitadas em modo não-direcionado (`Directed: false`).
- [x] `TestFindPath_MaxDepthCutoff`: Valida que caminhos que excedem `max_depth` não são retornados.
- [x] `TestFindPath_CyclicGraph`: Valida terminação e ausência de loops infinitos em grafos cíclicos densos.

### Camadas de Persistência SQLite e Postgres (`internal/db/` & `internal/store/`)
- [x] `TestDB_FindPath`: Executa `FindPath` contra banco SQLite real com múltiplos nós e arestas tipadas.
- [x] `TestPostgres_FindPath`: Executa `FindPath` contra mock/store do PostgreSQL validando isolamento por repositório.

### Subcomando CLI `mem path` (`cmd/mem/`)
- [x] `TestPathCLI_BasicRoute`: Executa `mem path "nodeA" "nodeB"` e valida renderização ASCII no stdout.
- [x] `TestPathCLI_JSONOutput`: Executa `mem path "nodeA" "nodeB" --json` e valida parsing JSON do payload.
- [x] `TestPathCLI_FlagsValidation`: Valida parsing de flags `--undirected`, `--max-depth`, `--mode`.

### Ferramenta MCP `memory_find_path` (`internal/mcp/`)
- [x] `TestMCP_MemoryFindPath_Valid`: Chama `memory_find_path` com argumentos válidos e valida resposta em Markdown.
- [x] `TestMCP_MemoryFindPath_MissingParams`: Valida erro com código `InvalidParams` na ausência de `source` ou `target`.
- [x] `TestMCP_MemoryFindPath_ToolRegistration`: Valida presença da ferramenta no catálogo retornado por `tools/list`.

---

## 2. Critérios de Conclusão e Regressão
- [x] 100% dos testes novos passando.
- [x] Todos os pacotes Go continuam passando com 100% de sucesso (`go test -count=1 ./...`).
- [x] Recompilação de `bin/mem.exe` bem-sucedida.
