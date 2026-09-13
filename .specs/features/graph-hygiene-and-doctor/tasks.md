# Tasks: graph-hygiene-and-doctor

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Store & Pruning | unit | Cascading document deletion, chunk & edge expurge | internal/store/*_test.go | go test -v ./internal/store/... |
| DB & Doctor | unit | Dead links, orphan notes, self-loops, health score | internal/store/*_test.go | go test -v ./internal/store/... |
| MCP Engine | unit | memory_doctor tool schema, invocation, markdown output | internal/mcp/*_test.go | go test -v ./internal/mcp/... |
| CLI / ADR | none | CLI dispatch, ascii dashboard, ADR documentation | cmd/mem/main.go | go test -v ./internal/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After pruning or doctor engine updates | go test -v ./internal/store/... |
| Full | After MCP or integration updates | go test -count=1 -v ./internal/store/... ./internal/parser/... ./internal/turboquant/... ./internal/mcp/... ./internal/canvas/... ./internal/repo/... |
| Build | After CLI or ADR updates | go test -v ./internal/... && python .agents/skills/tlc-spec-driven/scripts/validate_state.py graph-hygiene-and-doctor |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T3 -> T4 -> T5 -> T6 -> T7
```

### Phase 1: Pruning e Motor de Diagnóstico

## Task Breakdown

### T1: Estruturas de Dados e Pruning em Cascata no SQLite
**What**: Declarar tipos DeadLink, OrphanNote, SelfLoop, DoctorReport e implementar PruneDeletedDocuments no SQLite
**Where**: internal/db/store.go
**Depends on**: none
**Requirement**: PRUNE-01, PRUNE-02
**Tests**: internal/store/prune_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Declarar structs de diagnóstico em store.go
- [x] Implementar PruneDeletedDocuments em internal/db/store.go apagando documentos, chunks, fts, vec, turboquant e arestas
- [x] Adicionar testes unitários validando a remoção de documentos deletados do SQLite

### T2: Pruning Incremental no PostgreSQL e Interface Store
**What**: Implementar PruneDeletedDocuments no PostgresStore e atualizar contrato da interface Store
**Where**: internal/store/postgres.go
**Depends on**: T1
**Requirement**: PRUNE-01, PRUNE-02
**Tests**: internal/store/postgres_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Declarar método PruneDeletedDocuments na interface Store
- [x] Implementar PruneDeletedDocuments no PostgresStore com transação atômica
- [x] Validar conformidade de interface no pacote store

### T3: Motor de Diagnóstico e Linter no SQLite
**What**: Implementar DiagnoseHealth e FixHealthIssues no SQLite detectando dead links, notas órfãs e self-loops
**Where**: internal/db/graph.go
**Depends on**: T2
**Requirement**: DOCTOR-01, DOCTOR-02
**Tests**: internal/store/doctor_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Implementar DiagnoseHealth no SQLite com detecção de dead links, orphan notes, self loops e health score
- [x] Implementar FixHealthIssues no SQLite expurgando self-loops e links quebrados
- [x] Adicionar testes unitários validando relatórios de diagnóstico e correções

### T4: Motor de Diagnóstico e Linter no PostgreSQL
**What**: Implementar DiagnoseHealth e FixHealthIssues no PostgresStore
**Where**: internal/store/doctor.go
**Depends on**: T3
**Requirement**: DOCTOR-01, DOCTOR-02
**Tests**: internal/store/doctor_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Declarar DiagnoseHealth e FixHealthIssues na interface Store
- [x] Implementar DiagnoseHealth e FixHealthIssues para PostgreSQL em internal/store/doctor.go
- [x] Adicionar testes unitários asserindo paridade comportamental com SQLite

### T5: Ferramenta MCP memory_doctor e Handlers
**What**: Declarar ToolMemoryDoctor, registrar em NewServer e implementar NewMemoryDoctorHandler
**Where**: internal/mcp/handlers.go
**Depends on**: T4
**Requirement**: DOCTOR-04
**Tests**: internal/mcp/handlers_test.go
**Gate**: go test -v ./internal/mcp/...
**Done when**:
- [x] Declarar ToolMemoryDoctor em tools.go
- [x] Implementar NewMemoryDoctorHandler e FormatDoctorReport em handlers.go
- [x] Expor SetDoctorHandler em server.go e registrar ferramenta
- [x] Adicionar testes unitários validando execução via JSON-RPC

### T6: CLI mem index com Pruning e Comando mem doctor
**What**: Integrar pruning no comando mem index (com --no-prune) e adicionar comando mem doctor com dashboard ASCII
**Where**: cmd/mem/main.go
**Depends on**: T5
**Requirement**: PRUNE-01, DOCTOR-03
**Tests**: cmd/mem/main.go
**Gate**: go test -v ./internal/...
**Done when**:
- [ ] Integrar coleta de arquivos e chamada a PruneDeletedDocuments em runIndexSQLite e runIndexPostgres
- [ ] Adicionar flag --no-prune no comando index
- [ ] Implementar comando doctor com dashboard ASCII exibindo score, dead links e notas órfãs
- [ ] Conectar SetDoctorHandler no runMCPServer

### T7: ADR-013, Documentação e Validação Final
**What**: Registrar decisão ADR-013, atualizar documentação e STATE.md
**Where**: docs/adr/013-higiene-de-grafo-pruning-e-doctor.md
**Depends on**: T6
**Requirement**: DOCTOR-01, DOCTOR-03
**Tests**: docs/adr/013-higiene-de-grafo-pruning-e-doctor.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/graph-hygiene-and-doctor/spec.md
**Done when**:
- [ ] Criar docs/adr/013-higiene-de-grafo-pruning-e-doctor.md no formato MADR
- [ ] Atualizar docs/adr/README.md, docs/CLI_GUIDE.md e docs/REPOSITORY_BRAIN.md
- [ ] Atualizar .specs/STATE.md e gerar validation.md
