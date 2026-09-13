# Tasks: incremental-indexing-cache

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Store Core | unit | SHA-256 vectors; empty slice; store methods | internal/store/*_test.go | go test -v ./internal/store/... |
| DB Engine | unit | Schema migration, hash querying, cascading deletion | internal/db/*_test.go | go test -v ./internal/db/... |
| CLI / ADR | none | End-to-end indexing and caching verification | cmd/mem/main.go | go test -v ./... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After tasks with unit tests only | go test -v ./internal/store/... |
| Full | After tasks modifying DB or integration | go test -v ./internal/store/... ./internal/parser/... ./internal/turboquant/... ./internal/mcp/... ./internal/canvas/... ./internal/repo/... |
| Build | After CLI or ADR updates | go test -v ./internal/... && python .agents/skills/tlc-spec-driven/scripts/validate_state.py incremental-indexing-cache |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T4 -> T5
T1 -> T3 -> T4
```

### Phase 1: Foundation

## Task Breakdown

### T1: Hasher SHA-256 e Interface Store
**What**: Implementar gerador SHA-256 em pure Go e atualizar interface Store
**Where**: internal/store/hash.go
**Depends on**: none
**Requirement**: CACHE-01, CACHE-02
**Tests**: internal/store/hash_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Implementar CalculateContentHash retornando hex de 64 caracteres
- [x] Atualizar assinatura de InsertDocument para aceitar contentHash
- [x] Adicionar GetDocumentHash e DeleteDocumentData à interface Store

### T2: Implementação no SQLite
**What**: Adicionar content_hash no schema e implementar consulta e deleção no SQLite
**Where**: internal/db/store.go
**Depends on**: T1
**Requirement**: CACHE-02, CACHE-03
**Tests**: internal/db/store.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Adicionar coluna content_hash na tabela documents em schema.go
- [x] Atualizar InsertDocument no SQLite para salvar content_hash
- [x] Implementar GetDocumentHash e DeleteDocumentData no SQLite

### T3: Implementação no PostgreSQL
**What**: Adicionar content_hash no PostgreSQL e implementar consulta e deleção
**Where**: internal/store/postgres.go
**Depends on**: T1
**Requirement**: CACHE-02, CACHE-03
**Tests**: internal/store/postgres_test.go
**Gate**: go test -v ./internal/store/...
**Done when**:
- [x] Adicionar content_hash no schema PostgreSQL
- [x] Implementar GetDocumentHash e DeleteDocumentData no PostgresStore
- [x] Garantir conformidade com interface Store

### T4: CLI mem index com Cache Incremental e Flag --force
**What**: Integrar verificação de cache por hash e flag --force no comando mem index
**Where**: cmd/mem/main.go
**Depends on**: T2, T3
**Requirement**: CACHE-04, CACHE-05
**Tests**: cmd/mem/main.go
**Gate**: go test -v ./internal/store/... ./internal/mcp/...
**Done when**:
- [x] Adicionar flag --force no subcomando mem index
- [x] Pular arquivos cujo SHA-256 coincida com o hash persistido
- [x] Limpar chunks antigos antes de reindexar arquivos modificados
- [x] Exibir sumário de indexação com contagem de notas novas vs em cache

### T5: ADR-010 e Fechamento
**What**: Documentar ADR-010 e gerar relatório de validação da feature
**Where**: docs/adr/010-cache-incremental-de-indexacao-com-sha256.md
**Depends on**: T4
**Requirement**: CACHE-01, CACHE-05
**Tests**: docs/adr/010-cache-incremental-de-indexacao-com-sha256.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/incremental-indexing-cache/spec.md
**Done when**:
- [x] Registrar decisão arquitetural ADR-010
- [x] Atualizar .specs/STATE.md
- [x] Gerar relatório validation.md
