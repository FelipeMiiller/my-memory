# Tasks: batch-edge-resolution-lookups

**Spec**: [spec.md](./spec.md)
**Date**: 2026-10-29
**Issue**: ISSUE-034 §"Um ponto ficou de fora" (`.specs/ISSUES.md`)

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Preload da resolução (equivalência) | unit | 8 arestas cobrindo os 3 níveis da resolução (nível 1 único, nível 2 único, nível 2 ambíguo, nível 3 inexistente), o filtro de kinds e o lookup escopado por arquivo: o resultado com preload é IDÊNTICO ao sem preload, e o resultado ESPERADO de cada aresta é fixado | internal/codeast/preload_test.go | go test -count=1 -run TestPreload ./internal/codeast/... |
| Preload (cache) | unit | O preload preenche o cache com a chave do nível 1; a consulta por destino deixa de rodar | internal/codeast/preload_test.go | go test -count=1 -run TestPreloadPreenche ./internal/codeast/... |
| Contagem de queries no servidor | integration (PostgreSQL real) | 10 destinos distintos: o caminho por destino faz ≥10 SELECTs; o preload faz ≤2; durante a resolução com o cache cheio, **zero** | internal/codeast/preload_postgres_test.go | go test -count=1 -run TestPreloadPostgres ./internal/codeast/... |
| Comportamento no vault real | manual | A primeira população do code graph (code graph zerado) sai mais rápido, com o mesmo grafo | (vault `repo_e4c8f3b1a2d5`) | mem index |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após o preload (T1) | go build ./... e go vet ./... |
| Quick | Após os testes (T2) | go test -count=1 -run TestPreload ./internal/codeast/... |
| Quick | Após a integração (T3) | MY_MEMORY_PG_TEST=1 go test -count=1 -run TestPreloadPostgres ./internal/codeast/... |
| Full | Gate final (T4) | go test -count=1 ./... (com MY_MEMORY_PG_TEST=1 e MY_MEMORY_PG_URL preservada) |
| Build | Antes de declarar pronto | go build ./... e go vet ./... |
| Format | Antes de declarar pronto | gofmt -l na cópia LF dos arquivos .go tocados (o working tree local mistura CRLF) |
| Spec | Antes de entregar | python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/batch-edge-resolution-lookups/spec.md e validate_tasks.py .specs/features/batch-edge-resolution-lookups/tasks.md |
| State | Antes de entregar | python .agents/skills/tlc-spec-driven/scripts/validate_state.py batch-edge-resolution-lookups --root D:\repository\my-memory |

## Execution Plan

```
T1 (preload) ──▶ T2 (teste de equivalência) ──▶ T3 (integração + contagem) ──▶ T4 (gate final)
```

## Task Breakdown

### T1: O resolver busca os destinos com um punhado de queries

**Where**: internal/codeast/pipeline.go

**Depends on**: —

**Tests**: go test -count=1 -run TestPreload ./internal/codeast/...

**Gate**: go build ./... e go vet ./...

`edgeResolver.preload` coleta os destinos de todas as arestas pendentes (por
`qualified_name`, por `name` e por arquivo, no caso de `imports`) e os busca com
`IN (...)` em blocos de 500 valores, preenchendo o mesmo cache do caminho por
destino. `FlushEdges` chama o preload antes do laço de resolução.

### T2: O resultado da resolução é idêntico

**Where**: internal/codeast/preload_test.go

**Depends on**: T1

**Tests**: go test -count=1 -run TestPreloadDaResolucao ./internal/codeast/...

**Gate**: o teste fixa o resultado esperado de cada uma das 8 arestas (para a equivalência não ser verde por vacuidade) e falha com o preload mutado (corte de 3 → 1: o caso ambíguo deixa de ser ambíguo).

### T3: A contagem de queries no servidor

**Where**: internal/codeast/preload_postgres_test.go

**Depends on**: T2

**Tests**: MY_MEMORY_PG_TEST=1 go test -count=1 -run TestPreloadPostgres ./internal/codeast/...

**Gate**: ≥10 SELECTs no caminho por destino, ≤2 no preload e 0 durante a resolução com o cache cheio.

### T4: Gate final, medição no vault e evidência

**Where**: .specs/features/batch-edge-resolution-lookups/validation.md

**Depends on**: T3

**Tests**: go test -count=1 ./... e a primeira população do code graph no vault real (code graph zerado) medida antes/depois

**Gate**: os três validadores do fluxo spec-driven sem erro
