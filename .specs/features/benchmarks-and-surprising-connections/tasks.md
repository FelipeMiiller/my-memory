# Tasks: benchmarks-and-surprising-connections

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Benchmarks | benchmark | Throughput, latency, B/op, allocs/op | internal/**/*_bench_test.go | go test -bench=. -benchmem ./internal/... |
| Store & DB | unit | Surprising connections, anti-join, similarity filter | internal/store/*_test.go | go test -v ./internal/store/... |
| MCP Engine | unit | memory_get_insights schema, arguments, response | internal/mcp/*_test.go | go test -v ./internal/mcp/... |
| CLI / ADR | none | CLI dispatch, table formatting, ADR documentation | cmd/mem/main.go | go test -v ./internal/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After benchmark or store tasks | go test -v ./internal/turboquant/... ./internal/store/... |
| Full | After MCP or integration updates | go test -count=1 -v ./internal/store/... ./internal/parser/... ./internal/turboquant/... ./internal/mcp/... ./internal/canvas/... ./internal/repo/... |
| Build | After CLI or ADR updates | go test -v ./internal/... && python .agents/skills/tlc-spec-driven/scripts/validate_state.py benchmarks-and-surprising-connections |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T3 -> T4 -> T5 -> T6
```

### Phase 1: Benchmarks & Algoritmo

## Task Breakdown

### T1: Suíte de Micro-Benchmarks em Go
**What**: Implementar benchmarks de quantização TurboQuant, fusão RRF, hash SHA-256 e parsing de Markdown
**Where**: internal/turboquant/quantizer_bench_test.go
**Depends on**: none
**Requirement**: BENCH-01, BENCH-02, BENCH-03
**Tests**: internal/turboquant/quantizer_bench_test.go
**Gate**: go test -bench=. -benchmem ./internal/turboquant/... ./internal/store/... ./internal/parser/...
**Done when**:
- [x] Implementar BenchmarkQuantize_4Bit, BenchmarkDotProduct_4Bit e BenchmarkDotProduct_Float32
- [x] Implementar BenchmarkFuseRRF_100Items e BenchmarkFuseRRF_1000Items
- [x] Implementar BenchmarkCalculateContentHash em múltiplos tamanhos (1KB, 64KB, 1MB)
- [x] Implementar BenchmarkExtractConnections e BenchmarkChunkText

### T2: Algoritmo de Conexões Inesperadas e Contrato Store
**What**: Adicionar tipo SurprisingConnection, método FindSurprisingConnections no SQLite e PostgreSQL
**Where**: internal/store/store.go
**Depends on**: T1
**Requirement**: SURPR-01
**Tests**: internal/store/store_test.go, internal/store/postgres_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Declarar struct SurprisingConnection e método FindSurprisingConnections na interface Store
- [x] Implementar FindSurprisingConnections no SQLite cruzando similaridade e anti-join em graph_edges
- [x] Implementar FindSurprisingConnections no PostgreSQL com filtro de repositório e min_similarity
- [x] Adicionar testes unitários validando a detecção de conexões não linkadas

### T3: Ferramenta MCP memory_get_insights
**What**: Registrar e implementar a ferramenta memory_get_insights no servidor MCP
**Where**: internal/mcp/handlers.go
**Depends on**: T2
**Requirement**: SURPR-02
**Tests**: internal/mcp/handlers_test.go
**Gate**: go test -v ./internal/mcp/...
**Done when**:
- [x] Declarar ToolMemoryGetInsights no catálogo tools.go
- [x] Implementar NewMemoryGetInsightsHandler e FormatInsights em handlers.go
- [x] Registrar ferramenta em NewServer e expor SetInsightsHandler
- [x] Adicionar testes unitários validando chamadas e respostas da ferramenta

### T4: CLI mem bench e mem insights
**What**: Adicionar subcomandos mem bench e mem insights na CLI
**Where**: cmd/mem/main.go
**Depends on**: T3
**Requirement**: SURPR-03
**Tests**: cmd/mem/main.go
**Gate**: go test -v ./internal/store/... ./internal/mcp/...
**Done when**:
- [x] Implementar comando mem bench executando benchmarks programáticos com saída tabular
- [x] Implementar comando mem insights consultando e exibindo tabela ASCII de conexões latentes
- [x] Conectar SetInsightsHandler no runMCPServer para SQLite e PostgreSQL
- [x] Atualizar documentação do help do CLI

### T5: Relatório de Performance docs/BENCHMARKS.md
**What**: Consolidar métricas aferidas da suíte em docs/BENCHMARKS.md
**Where**: docs/BENCHMARKS.md
**Depends on**: T4
**Requirement**: BENCH-01, BENCH-03
**Tests**: docs/BENCHMARKS.md
**Gate**: go test -bench=. ./internal/...
**Done when**:
- [ ] Documentar metodologia e ambiente de testes
- [ ] Registrar tabelas de throughput (TurboQuant vs Float32, RRF, Hashing, Parsing)
- [ ] Conectar BENCHMARKS.md no docs/README.md

### T6: ADR-012 e Validação Final
**What**: Registrar decisão arquitetural ADR-012, atualizar STATE.md e validation.md
**Where**: docs/adr/012-benchmarks-e-conexoes-inesperadas.md
**Depends on**: T5
**Requirement**: SURPR-01, SURPR-03
**Tests**: docs/adr/012-benchmarks-e-conexoes-inesperadas.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/benchmarks-and-surprising-connections/spec.md
**Done when**:
- [ ] Registrar decisão arquitetural ADR-012
- [ ] Atualizar docs/adr/README.md e .specs/STATE.md
- [ ] Gerar relatório de validação validation.md com cobertura de requisitos
