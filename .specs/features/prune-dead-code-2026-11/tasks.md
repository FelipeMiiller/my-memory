# Tasks: prune-dead-code-2026-11

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-14
**Issue**: ISSUE-042 (cada corte com ID rastreável)

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Design**: inline nesta spec (14 cortes mecânicos + o registro; **nenhuma** mudança de comportamento)
**Status**: Ready (2026-11-14)

---

## Test Coverage Matrix

> Guidelines found: `AGENTS.md` (regras 4 e 8), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`.

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Pacotes tocados pela poda | unit (existente) | AC2 e AC3 — o gate completo e o taggeado passam **sem** que nenhum teste seja removido ou alterado | `internal/{codeast,db,embedder,event_runtime,federation,graph,mcp,supervisor,vaultfs}` | `go test -count=1 ./...` |
| Prova de que o que ficou e o que devia ficar | integration | AC4 — o `deadcode` com a tag do release reporta exatamente 3 (as do `Subscriber`) | repo inteiro | `deadcode -test -tags "sqlite_fts5 onnx" ./...` |
| Comportamento do vault | integration | AC6 — o cofre continua 100/100 | Postgres `repo_central` | `mem doctor --storage=postgres --repo repo_central` |

---

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após T1 | `go build ./...` e `go vet ./...` |
| Medido | T1 | `deadcode -test -tags "sqlite_fts5 onnx" ./...` → 3 funções |
| Full | Gate final (T2) | `go test -count=1 ./...` (com `MY_MEMORY_PG_TEST=1` e `MY_MEMORY_PG_URL`) |
| Tagged | Gate final (T2) | `go test -tags "sqlite_fts5 onnx" -count=1 ./...` |
| Medido | T2 | `mem doctor --storage=postgres --repo repo_central` → 100/100 |
| Format | Antes de declarar pronto | `gofmt -l` na cópia LF dos arquivos `.go` tocados |
| Spec | Antes de entregar | `validate_spec.py` e `validate_tasks.py` |
| State | Antes de entregar | `validate_state.py prune-dead-code-2026-11 --root D:\repository\my-memory` |

---

## Execution Plan

### Phase 1: O corte

```
T1
```

### Phase 2: A prova e o fechamento

```
T1 -> T2
```

---

## Task Breakdown

### T1: Os 14 cortes

**What**: remover as 14 funções provadamente sem chamada (a lista está na
ISSUE-042), sem tocar em nenhuma outra linha.

**Where**: `internal/codeast/parser.go` · `internal/db/graph.go` ·
`internal/embedder/model_downloader.go` ·
`internal/event_runtime/{audit_subscriber,dispatcher,log}.go` ·
`internal/federation/search.go` · `internal/graph/inspector.go` ·
`internal/mcp/{handlers,server}.go` · `internal/supervisor/audit.go` ·
`internal/vaultfs/target.go` · `.specs/ISSUES.md`
**Depends on**: None
**Reuses**: o inventário (`.specs/INVENTARIO-2026-11-14.md`) · a prova de chamada
**Requirement**: PDC-01, PDC-05, PDC-07

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] As 14 funções não estão mais no código
- [x] Nenhum outro bloco foi alterado (o diff é só a remoção)
- [x] A ISSUE-042 lista cada corte com `arquivo:linha`, nome e prova

**Status**: Verified (2026-11-14) — 14 blocos removidos; `deadcode` com a tag do release passou a reportar **3** (as do `Subscriber`); gate completo e taggeado verdes; cofre em 100/100.

**Tests**: unit (existente)
**Gate**: quick

**Commit**: `refactor: remove as 14 funcoes provadamente sem chamada`

---

### T2: A prova e o fechamento

**What**: rodar o `deadcode` com a tag do release, os dois gates e o doctor do
cofre; registrar o resultado na ISSUE-042, no `STATE.md` e no `validation.md`.

**Where**: `.specs/ISSUES.md` · `.specs/STATE.md` ·
`.specs/features/prune-dead-code-2026-11/validation.md`
**Depends on**: T1
**Reuses**: o template do `validate.md`
**Requirement**: PDC-02, PDC-03, PDC-04, PDC-06, PDC-08

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] `deadcode -test -tags "sqlite_fts5 onnx"` reporta 3
- [x] Gate completo e taggeado verdes, sem teste removido
- [x] O cofre continua em 100/100
- [x] A ISSUE-042 e o `STATE.md` registram o que saiu e o que ficou
- [x] O `validation.md` existe com veredito e evidência `file:line`

**Status**: Verified (2026-11-14) — veredito PASS; 8/8 ACs; gates verdes; cofre 100/100.

**Tests**: none (documentação)
**Gate**: full

**Commit**: `docs(spec): fecha a poda com a evidencia medida`

---

## Phase Execution Map

```
Phase 1 → Phase 2

Phase 1:  T1
Phase 2:  T1 -> T2
```

**Contagem de tarefas**: 2 — execução **inline**.

---

## Diagram-Definition Cross-Check

| Task | Depends On (task body) | Diagram Shows | Status |
| ---- | ---------------------- | ------------- | ------ |
| T1 | None | (início da fase 1) | ✅ Match |
| T2 | T1 | T1 -> T2 | ✅ Match |

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | Pacotes tocados pela poda | unit (existente) | unit | ✅ OK |
| T2 | Prova + comportamento do vault | integration | none | ✅ OK (a verificação é o gate) |

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: os 14 cortes | 9 arquivos, 14 blocos | ✅ Granular |
| T2: prova e fechamento | artefatos de registro | ✅ Granular |
