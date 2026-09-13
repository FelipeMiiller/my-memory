# Tasks: epistemic-edges-and-god-nodes

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Parser | unit | Typed wikilinks, tags, edge extraction, backward compatibility | internal/parser/*_test.go | go test -v ./internal/parser/... |
| Store & DB | unit | GodNode type, InsertEdgeWithProps, GetGodNodes query | internal/store/*_test.go | go test -v ./internal/store/... |
| MCP Engine | unit | memory_get_hubs catalog, input parsing, JSON formatting | internal/mcp/*_test.go | go test -v ./internal/mcp/... |
| CLI / ADR | none | CLI dispatch, hubs table output, ADR documentation | cmd/mem/main.go | go test -v ./internal/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After parser or store tasks | go test -v ./internal/parser/... ./internal/store/... |
| Full | After MCP or integration updates | go test -v ./internal/store/... ./internal/parser/... ./internal/turboquant/... ./internal/mcp/... ./internal/canvas/... ./internal/repo/... |
| Build | After CLI or ADR updates | go test -v ./internal/... && python .agents/skills/tlc-spec-driven/scripts/validate_state.py epistemic-edges-and-god-nodes |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T3 -> T4 -> T5 -> T6
```

### Phase 1: Foundation

## Task Breakdown

### T1: Parser de Conexões Tipadas e Arestas Epistêmicas
**What**: Implementar suporte a wikilinks tipados e tags na struct ExtractedConnections
**Where**: internal/parser/wikilinks.go
**Depends on**: none
**Requirement**: EDGE-01
**Tests**: internal/parser/parser_test.go
**Gate**: go test -v ./internal/parser/...
**Done when**:
- [x] Definir EdgeConnection e adicionar Edges à ExtractedConnections
- [x] Suportar prefixos de relação [[relation:Target]] e aliases [[Target|rel:relation]]
- [x] Mapear tags como arestas tagged_as
- [x] Garantir 100% de retrocompatibilidade com OutgoingLinks

### T2: Contrato Store e Implementação no SQLite
**What**: Adicionar tipo GodNode, método GetGodNodes e InsertEdgeWithProps no SQLite
**Where**: internal/db/store.go
**Depends on**: T1
**Requirement**: EDGE-02, EDGE-03
**Tests**: internal/store/store.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Definir struct GodNode no pacote store
- [x] Adicionar colunas epistemic_status e weight em graph_edges no schema SQLite
- [x] Implementar InsertEdgeWithProps e GetGodNodes no SQLite

### T3: Implementação no PostgreSQL
**What**: Implementar InsertEdgeWithProps e GetGodNodes no PostgresStore
**Where**: internal/store/postgres.go
**Depends on**: T2
**Requirement**: EDGE-02, EDGE-03
**Tests**: internal/store/postgres_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [ ] Adicionar colunas epistemic_status e weight em PostgresSchema
- [ ] Implementar InsertEdgeWithProps e GetGodNodes no PostgresStore
- [ ] Garantir conformidade com a interface Store

### T4: Ferramenta MCP memory_get_hubs
**What**: Registrar e implementar a ferramenta memory_get_hubs no servidor MCP
**Where**: internal/mcp/handlers.go
**Depends on**: T3
**Requirement**: EDGE-04
**Tests**: internal/mcp/handlers_test.go
**Gate**: go test -v ./internal/mcp/...
**Done when**:
- [ ] Registrar ToolMemoryGetHubs em tools.go
- [ ] Implementar handleMemoryGetHubs no handlers.go
- [ ] Adicionar testes unitários validando chamadas e respostas

### T5: CLI mem hubs e Indexação com Arestas Tipadas
**What**: Adicionar comando mem hubs e salvar arestas tipadas durante mem index
**Where**: cmd/mem/main.go
**Depends on**: T4
**Requirement**: EDGE-05
**Tests**: cmd/mem/main.go
**Gate**: go test -v ./internal/store/... ./internal/mcp/...
**Done when**:
- [ ] Salvar Edges extraídas via InsertEdgeWithProps em runIndexPostgres e runIndexSQLite
- [ ] Adicionar comando mem hubs exibindo tabela de centralidade de nós
- [ ] Atualizar documentação do help do CLI

### T6: ADR-011 e Validação
**What**: Documentar ADR-011, atualizar STATE.md e gerar relatório validation.md
**Where**: docs/adr/011-arestas-epistemicas-e-god-nodes.md
**Depends on**: T5
**Requirement**: EDGE-01, EDGE-05
**Tests**: docs/adr/011-arestas-epistemicas-e-god-nodes.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/epistemic-edges-and-god-nodes/spec.md
**Done when**:
- [ ] Registrar decisão arquitetural ADR-011
- [ ] Atualizar .specs/STATE.md
- [ ] Gerar relatório de validação validation.md
