# Tasks: pagerank-and-graph-centrality

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Graph Engine | unit | Convergence, star graph, ring graph, dangling nodes, weights | internal/graph/*_test.go | go test -v ./internal/graph/... |
| Store & DB | unit | ComputePageRank in SQLite and PostgreSQL, sorting | internal/store/*_test.go | go test -v ./internal/store/... |
| MCP Engine | unit | memory_get_hubs tool schema, algorithm param, output | internal/mcp/*_test.go | go test -v ./internal/mcp/... |
| CLI / ADR | none | CLI dispatch, table rendering, ADR documentation | cmd/mem/main.go | go test -v ./internal/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After graph engine updates | go test -v ./internal/graph/... |
| Store | After Store & DB implementations | go test -v ./internal/store/... |
| Full | After MCP or integration updates | go test -count=1 -v ./internal/graph/... ./internal/store/... ./internal/mcp/... |
| Build | After CLI or ADR updates | go test -v ./internal/... && python .agents/skills/tlc-spec-driven/scripts/validate_state.py pagerank-and-graph-centrality |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T3 -> T4 -> T5 -> T6
```

### Phase 1: Motor Matemático e Armazenamento

## Task Breakdown

### T1: Motor Matemático de PageRank em Go Puro
**What**: Implementar algoritmo iterativo de PageRank com amortecimento, dangling nodes e ponderação epistêmica
**Where**: internal/graph/pagerank.go
**Depends on**: none
**Requirement**: RANK-01, RANK-02
**Tests**: internal/graph/pagerank_test.go
**Gate**: go test -v ./internal/graph/...
**Done when**:
- [x] Implementar ComputePageRank com fator de amortecimento e tolerância de convergência
- [x] Tratar dangling nodes com redistribuição uniforme
- [x] Ponderar arestas conforme pesos definidos
- [x] Adicionar testes unitários validando invariantes matemáticos e grafos canônicos

### T2: Estrutura PageRankNode e Método ComputePageRank no SQLite
**What**: Declarar PageRankNode e implementar ComputePageRank no SQLite
**Where**: internal/db/store.go
**Depends on**: T1
**Requirement**: RANK-03
**Tests**: internal/store/pagerank_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Declarar PageRankNode e método ComputePageRank no pacote store
- [x] Implementar ComputePageRank em internal/db/store.go carregando nós e arestas ativas
- [x] Adicionar testes unitários validando ordenação decrescente por score no SQLite

### T3: Implementação de ComputePageRank no PostgreSQL
**What**: Implementar ComputePageRank no PostgresStore com isolamento por repositório
**Where**: internal/store/postgres.go
**Depends on**: T2
**Requirement**: RANK-03
**Tests**: internal/store/postgres_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Implementar ComputePageRank em internal/store/postgres.go
- [x] Validar conformidade de interface Store
- [x] Adicionar testes unitários com filtro de repositório

### T4: Suporte a PageRank na Ferramenta MCP memory_get_hubs
**What**: Adicionar parâmetro algorithm no schema e handler de memory_get_hubs
**Where**: internal/mcp/handlers.go
**Depends on**: T3
**Requirement**: RANK-04
**Tests**: internal/mcp/handlers_test.go
**Gate**: go test -v ./internal/mcp/...
**Done when**:
- [x] Atualizar schema de ToolMemoryGetHubs com propriedade algorithm (degree | pagerank)
- [x] Atualizar NewMemoryGetHubsHandler para executar cálculo de PageRank quando solicitado
- [x] Formatar saída Markdown detalhada com scores e ranking
- [x] Adicionar testes JSON-RPC para tool call com algorithm=pagerank

### T5: Suporte a PageRank no Comando CLI mem hubs
**What**: Integrar flags --algorithm, --damping e --iter no subcomando mem hubs
**Where**: cmd/mem/main.go
**Depends on**: T4
**Requirement**: RANK-05
**Tests**: cmd/mem/main.go
**Gate**: go test -v ./internal/...
**Done when**:
- [ ] Adicionar flags --algorithm, --damping e --iter no flagset hubs
- [ ] Atualizar displayHubsTable para exibir coluna de score do PageRank
- [ ] Roteamento para SQLite e PostgreSQL

### T6: ADR-014, Benchmarks de PageRank e Documentação
**What**: Registrar decisão ADR-014, micro-benchmark e atualizar manuais
**Where**: docs/adr/014-centralidade-de-grafo-com-pagerank-ponderado.md
**Depends on**: T5
**Requirement**: RANK-06
**Tests**: docs/adr/014-centralidade-de-grafo-com-pagerank-ponderado.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/pagerank-and-graph-centrality/spec.md
**Done when**:
- [ ] Criar docs/adr/014-centralidade-de-grafo-com-pagerank-ponderado.md no formato MADR
- [ ] Implementar micro-benchmark de convergência de PageRank em internal/store/pagerank_bench_test.go
- [ ] Atualizar CLI_GUIDE.md, REPOSITORY_BRAIN.md, README.md e STATE.md
