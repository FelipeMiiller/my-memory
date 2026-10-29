# Tasks: fix-doctor-dead-links-code-nodes

**Spec**: [spec.md](./spec.md)
**Date**: 2026-10-29
**Issue**: ISSUE-036 (`.specs/ISSUES.md`)

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Doctor SQLite (detecção) | unit | Nó `code_symbol` não é dead link; alvo que não é nota nem código continua morto | internal/db/doctor_code_nodes_test.go | go test -count=1 -run 'TestDoctorNaoConta\|TestDoctorContinua' ./internal/db/... |
| Correção SQLite | unit | A correção não apaga a ponte e apaga a órfã | internal/db/doctor_code_nodes_test.go | go test -count=1 -run TestCorrecaoDeDeadLinks ./internal/db/... |
| Doctor PostgreSQL (detecção) | integration (PostgreSQL real) | Nó `code_symbol` não é dead link; alvo inexistente continua morto | internal/store/doctor_code_nodes_test.go | go test -count=1 -run 'TestPostgresDoctor' ./internal/store/... |
| Correção PostgreSQL | integration (PostgreSQL real) | A correção não apaga a ponte e apaga a órfã | internal/store/doctor_code_nodes_test.go | go test -count=1 -run TestPostgresFixHealthIssues ./internal/store/... |
| Comportamento no vault real | manual | Health Score e contagem de dead links antes/depois | (vault `repo_e4c8f3b1a2d5`) | mem doctor --storage=postgres --repo repo_e4c8f3b1a2d5 |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após o fix (T1) | go build ./... |
| Quick | Após o fix (T2) | go test -count=1 -run 'DeadLink\|CodeSymbol' ./internal/db/... ./internal/store/... |
| Full | Gate final (T4) | go test -count=1 ./... (com MY_MEMORY_PG_TEST=1 e MY_MEMORY_PG_URL preservada) |
| Build | Antes de declarar pronto | go build ./... e go vet ./... |
| Format | Antes de declarar pronto | gofmt -l na cópia LF dos arquivos .go tocados (o working tree local mistura CRLF) |
| Spec | Antes de entregar | python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/fix-doctor-dead-links-code-nodes/spec.md e validate_tasks.py .specs/features/fix-doctor-dead-links-code-nodes/tasks.md |
| State | Antes de entregar | python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-doctor-dead-links-code-nodes --root D:\repository\my-memory |

## Execution Plan

```
T1 (SQLite: detecção) ──▶ T2 (SQLite: correção) ──▶ T3 (PostgreSQL) ──▶ T4 (gate final)
```

## Task Breakdown

### T1: O doctor SQLite reconhece o nó de código

**Where**: internal/db/graph.go
**Depends on**: —
**Tests**: go test -count=1 -run 'TestDoctorNaoConta\|TestDoctorContinua' ./internal/db/...
**Gate**: go build ./...

Incluir o nó `code_symbol` na lista de alvos vivos da query de dead links, com o mesmo critério da query de correção.

### T2: A correção SQLite não apaga o code graph

**Where**: internal/db/graph.go
**Depends on**: T1
**Tests**: go test -count=1 -run TestCorrecaoDeDeadLinks ./internal/db/...
**Gate**: o teste falha com o critério antigo (a ponte seria apagada) e passa com o novo

### T3: O doctor PostgreSQL reconhece o nó de código

**Where**: internal/store/doctor.go
**Depends on**: T2
**Tests**: go test -count=1 -run 'TestPostgresDoctor\|TestPostgresFixHealthIssues' ./internal/store/...
**Gate**: o mesmo critério, escopado pelo repositório da aresta

### T4: Gate final e evidência

**Where**: .specs/features/fix-doctor-dead-links-code-nodes/validation.md
**Depends on**: T3
**Tests**: go test -count=1 ./... e o `mem doctor` no vault real
**Gate**: os três validadores do fluxo spec-driven sem erro
