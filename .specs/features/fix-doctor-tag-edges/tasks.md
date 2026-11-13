# Tasks: fix-doctor-tag-edges

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-13
**Issue**: ISSUE-038 (`.specs/ISSUES.md`)

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Doctor SQLite (detecção) | unit | aresta `tagged_as` de tag viva **não** é dead link; de tag inexistente **é** | internal/db/doctor_tag_edges_test.go | go test -count=1 -run TestDoctorTag ./internal/db/... |
| Correção SQLite | unit | a correção apaga a aresta de tag morta e **preserva** a viva | internal/db/doctor_tag_edges_test.go | go test -count=1 -run TestDoctorTag ./internal/db/... |
| Doctor Postgres (detecção) | integration (PostgreSQL real) | o mesmo contrato no engine de produção | internal/store/doctor_tag_edges_test.go | go test -count=1 -run TestPostgresDoctorTag ./internal/store/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após o fix (T1) | go build ./... e go vet ./... |
| Quick | Após os testes (T2) | go test -count=1 -run Tag ./internal/db/... ./internal/store/... |
| Full | Gate final (T3) | go test -count=1 ./... (com MY_MEMORY_PG_TEST=1 e MY_MEMORY_PG_URL preservada) |
| Format | Antes de declarar pronto | gofmt -l na cópia LF dos arquivos .go tocados (o working tree local mistura CRLF) |
| Spec | Antes de entregar | python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/fix-doctor-tag-edges/spec.md e validate_tasks.py .specs/features/fix-doctor-tag-edges/tasks.md |
| State | Antes de entregar | python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-doctor-tag-edges --root D:\repository\my-memory |

## Execution Plan

```
T1 (SQLite) ──▶ T2 (Postgres) ──▶ T3 (gate final)
```

## Task Breakdown

### T1: O doctor SQLite reconhece a tag como alvo vivo

**Where**: internal/db/graph.go

**Depends on**: —

**Tests**: go test -count=1 -run Tag ./internal/db/...

**Gate**: go build ./... e go vet ./...

A detecção e a rotina de correção passam a excluir do dead link o alvo que
existe em `document_tags` (por `tag` ou `tag_key`).

### T2: O mesmo no Postgres

**Where**: internal/store/doctor.go

**Depends on**: T1

**Tests**: MY_MEMORY_PG_TEST=1 go test -count=1 -run TestPostgresDoctorTag ./internal/store/...

**Gate**: o mesmo contrato contra o engine de produção, num banco descartável.

### T3: Gate final e evidência

**Where**: .specs/features/fix-doctor-tag-edges/validation.md

**Depends on**: T2

**Tests**: go test -count=1 ./... e o `mem doctor` do cofre central antes/depois

**Gate**: os três validadores do fluxo spec-driven sem erro
