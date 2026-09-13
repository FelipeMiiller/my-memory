# Tasks: temporal-decay-and-recency-search

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Store & RRF | unit | CalculateTimeDecay, half-life math, FuseSearchResultsWithDecay | internal/store/*_test.go | go test -v ./internal/store/... |
| DB Layer | unit | UpdatedAt in SearchResult, SearchHybridRRFWithDecay SQLite | internal/db/*_test.go | go test -v ./internal/db/... |
| Store Postgres | unit | Postgres SearchHybridRRFWithDecay, interface compliance | internal/store/*_test.go | go test -v ./internal/store/... |
| MCP Engine | unit | memory_search schema, decay options, handlers | internal/mcp/*_test.go | go test -v ./internal/mcp/... |
| CLI / ADR | none | CLI dispatch, decay flags, ADR documentation | cmd/mem/main.go | go test -v ./internal/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After RRF decay engine updates | go test -v -run "TestTimeDecay|TestFuseSearchResults" ./internal/store/... |
| Store | After DB & Postgres implementations | go test -v ./internal/store/... ./internal/db/... |
| Full | After MCP or integration updates | go test -count=1 -v ./internal/store/... ./internal/db/... ./internal/mcp/... |
| Build | After CLI or ADR updates | go test -v ./internal/... && python .agents/skills/tlc-spec-driven/scripts/validate_state.py temporal-decay-and-recency-search |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T3 -> T4 -> T5 -> T6
```

### Phase 1: Motor Matemático e Armazenamento

## Task Breakdown

### T1: Motor Matemático de Decaimento Temporal e Fusão RRF Ponderada
**What**: Implementar DecayOptions, CalculateTimeDecay e FuseSearchResultsWithDecay em internal/store/rrf.go
**Where**: internal/store/rrf.go
**Depends on**: none
**Requirement**: DECAY-01, DECAY-02
**Tests**: internal/store/decay_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Declarar DecayOptions e DefaultDecayOptions em rrf.go
- [x] Implementar CalculateTimeDecay com meia-vida e peso
- [x] Implementar FuseSearchResultsWithDecay multiplicando score RRF por fator temporal
- [x] Adicionar testes unitários validando a curva matemática e inversão de ranking por recência

### T2: Enriquecimento de SearchResult com UpdatedAt e Queries SQLite
**What**: Adicionar UpdatedAt em SearchResult e atualizar queries FTS e KNN no SQLite
**Where**: internal/db/hybrid.go
**Depends on**: T1
**Requirement**: DECAY-03, DECAY-04
**Tests**: internal/db/hybrid_test.go
**Gate**: go test -v ./internal/db/...
**Done when**:
- [x] Adicionar campo UpdatedAt em SearchResult no pacote db e store
- [x] Atualizar queries SQL em internal/db/hybrid.go e internal/db/graph.go com JOIN em documents.updated_at
- [x] Implementar SearchHybridRRFWithDecay no SQLite
- [x] Validar preenchimento de UpdatedAt em testes

### T3: Suporte a Decaimento Temporal no PostgreSQL e Interface Store
**What**: Implementar SearchHybridRRFWithDecay no PostgresStore e atualizar contrato da interface Store
**Where**: internal/store/postgres.go
**Depends on**: T2
**Requirement**: DECAY-03, DECAY-04
**Tests**: internal/store/postgres_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Declarar SearchHybridRRFWithDecay na interface Store
- [x] Atualizar queries de SearchFTS e SearchKNN no PostgresStore populando UpdatedAt
- [x] Implementar SearchHybridRRFWithDecay no PostgresStore
- [x] Validar conformidade de interface Store

### T4: Parâmetros de Decaimento na Ferramenta MCP memory_search
**What**: Adicionar propriedades decay, half_life e decay_weight no schema e handler de memory_search
**Where**: internal/mcp/handlers.go
**Depends on**: T3
**Requirement**: DECAY-05
**Tests**: internal/mcp/handlers_test.go
**Gate**: go test -v ./internal/mcp/...
**Done when**:
- [x] Adicionar decay, half_life e decay_weight em ToolMemorySearch em tools.go
- [x] Atualizar NewMemorySearchHandler para repassar DecayOptions ao executor de busca
- [x] Adicionar testes JSON-RPC validando execução com decay=true

### T5: Flags de Decaimento no Comando CLI mem search
**What**: Adicionar flags --decay, --half-life e --decay-weight no subcomando mem search
**Where**: cmd/mem/main.go
**Depends on**: T4
**Requirement**: DECAY-06
**Tests**: cmd/mem/main.go
**Gate**: go test -v ./internal/...
**Done when**:
- [x] Adicionar flags --decay, --half-life e --decay-weight no flagset de busca
- [x] Repassar DecayOptions nos fluxos de busca SQLite e PostgreSQL
- [x] Atualizar printHelp e documentação do CLI

### T6: ADR-015, Validação Final e Documentação
**What**: Registrar decisão ADR-015 e atualizar manuais de documentação
**Where**: docs/adr/015-decaimento-temporal-exponencial-na-busca-hibrida.md
**Depends on**: T5
**Requirement**: DECAY-01, DECAY-06
**Tests**: docs/adr/015-decaimento-temporal-exponencial-na-busca-hibrida.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/temporal-decay-and-recency-search/spec.md
**Done when**:
- [x] Criar docs/adr/015-decaimento-temporal-exponencial-na-busca-hibrida.md no formato MADR
- [x] Atualizar docs/adr/README.md, docs/CLI_GUIDE.md, docs/REPOSITORY_BRAIN.md e README.md
- [x] Atualizar STATE.md e gerar validation.md com veredicto PASS
