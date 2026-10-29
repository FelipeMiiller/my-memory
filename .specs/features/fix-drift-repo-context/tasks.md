# Tasks: fix-drift-repo-context

**Spec**: [spec.md](./spec.md)
**Date**: 2026-10-29
**Issue**: ISSUE-035 (`.specs/ISSUES.md`)

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| CLI do drift (escopo de repositório) | integration (PostgreSQL real) | Arquivo indexado e inalterado NÃO vira mudança estrutural; o escopo do repositório chega à análise | cmd/mem/drift_postgres_test.go | go test -count=1 -run TestDriftCLI_Postgres ./cmd/mem/... |
| Drift no vault real (comportamento) | manual | Mudança cosmética → 0 drift; mudança de assinatura → detectada e atribuída; revertido → 0 | (vault `repo_e4c8f3b1a2d5`) | mem drift --storage=postgres --repo repo_e4c8f3b1a2d5 --json |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após o fix (T1) | go build ./... |
| Quick | Após o teste (T2) | go test -count=1 -run TestDriftCLI_Postgres ./cmd/mem/... |
| Full | Gate final (T3) | go test -count=1 ./... (com MY_MEMORY_PG_TEST=1 e MY_MEMORY_PG_URL preservada) |
| Build | Antes de declarar pronto | go build ./... e go vet ./... |
| Format | Antes de declarar pronto | gofmt -l na cópia LF dos arquivos .go tocados (o working tree local mistura CRLF) |
| Spec | Antes de entregar | python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/fix-drift-repo-context/spec.md e validate_tasks.py .specs/features/fix-drift-repo-context/tasks.md |
| State | Antes de entregar | python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-drift-repo-context --root D:\repository\my-memory |

## Execution Plan

```
T1 (fix do escopo) ──▶ T2 (teste de regressão) ──▶ T3 (gate final)
```

## Task Breakdown

### T1: O `mem drift` roda com escopo de repositório

**Where**: cmd/mem/drift.go
**Depends on**: —
**Tests**: go test -count=1 -run TestDriftCLI_Postgres ./cmd/mem/...
**Gate**: go build ./... e go vet ./...

Capturar o repositório devolvido por `resolveStorageAndRepo` e colocá-lo no contexto com `db.WithRepository(ctx, resolvedRepo)` antes de `drift.AnalyzeDrift` — o mesmo padrão de `cmd/mem/codeindex.go:84`.

### T2: Teste de regressão em PostgreSQL real

**Where**: cmd/mem/drift_postgres_test.go
**Depends on**: T1
**Tests**: go test -count=1 -run TestDriftCLI_Postgres ./cmd/mem/...
**Gate**: o teste falha com o fix revertido (sintoma: "acusou 1 mudança(s) ESTRUTURAL(is) em arquivo INALTERADO") e passa com o fix

Indexar um arquivo Go no banco descartável do `internal/pgtest` e rodar o drift com um runner de git falso: o arquivo é o mesmo indexado, então não pode virar mudança estrutural.

### T3: Gate final e evidência

**Where**: .specs/features/fix-drift-repo-context/validation.md
**Depends on**: T2
**Tests**: go test -count=1 ./... e o experimento no vault real (cosmética vs estrutural)
**Gate**: os três validadores do fluxo spec-driven sem erro
