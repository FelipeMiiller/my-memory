# Tasks: hybrid-search-rrf

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Store Core | unit | All formula branches; empty and single inputs; edge cases | internal/store/*_test.go | go test -v ./internal/store/... |
| DB Engine | unit | BM25 ranking, query sanitization, hybrid fusion | internal/db/*_test.go | go test -v ./internal/db/... |
| MCP Handlers | unit | All modes (hybrid, vector, fts), fallback, JSON formatting | internal/mcp/*_test.go | go test -v ./internal/mcp/... |
| CLI / ADR | none | Build and manual execution verification | cmd/mem/main.go | go test -v ./... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After tasks with unit tests only | go test -v ./internal/store/... ./internal/mcp/... |
| Full | After tasks modifying DB or integration | go test -v ./internal/parser/... ./internal/turboquant/... ./internal/mcp/... ./internal/canvas/... ./internal/repo/... ./internal/store/... |
| Build | After CLI or ADR updates | go test -v ./internal/... && python .agents/skills/tlc-spec-driven/scripts/validate_state.py hybrid-search-rrf |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T4 -> T5 -> T6
T1 -> T3 -> T4
```

### Phase 1: Foundation

## Task Breakdown

### T1: Algoritmo RRF e Tipos no Store
**What**: Implementar algoritmo de fusão RRF em pure Go e estender SearchResult
**Where**: internal/store/rrf.go
**Depends on**: none
**Requirement**: RRF-01, RRF-02
**Tests**: internal/store/rrf_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Implementar FuseRRF genérico e determinístico com suporte a k=60
- [x] Tratar listas vazias, empates e itens ausentes
- [x] Atualizar struct SearchResult com campos Score e Sources
- [x] Adicionar assinaturas de busca híbrida na interface Store

### T2: Busca FTS5 e Híbrida no SQLite
**What**: Implementar FTS5 e busca híbrida no SQLite combinando vetores e grafo
**Where**: internal/db/hybrid.go
**Depends on**: T1
**Requirement**: RRF-03, RRF-04
**Tests**: internal/db/hybrid_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Implementar sanitização de termos para FTS5
- [x] Implementar SearchFTS contra a tabela chunks_fts
- [x] Implementar SearchHybridRRF combinando FTS5, vetores (sqlite-vec ou TurboQuant) e CTE de vizinhos do grafo

### T3: Busca FTS e Híbrida no PostgreSQL
**What**: Implementar busca FTS e híbrida no PostgreSQL com pgvector
**Where**: internal/store/postgres.go
**Depends on**: T1
**Requirement**: RRF-05
**Tests**: internal/store/postgres_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Implementar SearchFTS usando tsvector/plainto_tsquery e ts_rank
- [x] Implementar SearchHybridRRF combinando FTS, SearchKNN e CTE GetNodeNeighbors filtrados por repositório

### T4: Integração no Servidor MCP
**What**: Expor busca híbrida como padrão na tool memory_search
**Where**: internal/mcp/handlers.go
**Depends on**: T2, T3
**Requirement**: RRF-06, RRF-07
**Tests**: internal/mcp/handlers_test.go
**Gate**: go test -v ./internal/mcp/...
**Done when**:
- [x] Atualizar tool schema memory_search para aceitar mode e k
- [x] Atualizar handler memory_search para chamar busca híbrida por padrão
- [x] Formatar saída detalhando score RRF e ranking individual das fontes

### T5: Integração no CLI mem search
**What**: Adicionar suporte a busca híbrida no comando de terminal mem search
**Where**: cmd/mem/main.go
**Depends on**: T4
**Requirement**: RRF-06, RRF-07
**Tests**: cmd/mem/main.go
**Gate**: go test -v ./internal/store/... ./internal/mcp/...
**Done when**:
- [x] Adicionar flags --mode e --k no comando mem search
- [x] Exibir scores RRF, fontes e vizinhos no terminal

### T6: ADR-009 e Fechamento
**What**: Documentar ADR-009 e gerar relatório de validação da feature
**Where**: docs/adr/009-busca-hibrida-com-reciprocal-rank-fusion-rrf.md
**Depends on**: T5
**Requirement**: RRF-01, RRF-07
**Tests**: docs/adr/009-busca-hibrida-com-reciprocal-rank-fusion-rrf.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/hybrid-search-rrf/spec.md
**Done when**:
- [ ] Registrar decisão arquitetural ADR-009
- [ ] Atualizar .specs/STATE.md
- [ ] Gerar relatório validation.md
