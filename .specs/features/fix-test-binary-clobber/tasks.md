# Tasks: fix-test-binary-clobber

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-14
**Issue**: ISSUE-041 (`.specs/ISSUES.md`)

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Design**: inline nesta spec (um helper compartilhado que compila no temp dir + um teste de regressão; **nenhuma** mudança de runtime)
**Status**: Ready (2026-11-14)

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec. Guidelines found: `AGENTS.md` (regra 4 — testes obrigatórios; regra 8 — quality gate), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`, e o contrato do próprio vault (`standards/contrato-de-skill.md`).

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Helper de build do binário de teste | unit | 1:1 com os ACs 2 e 4: compila em `t.TempDir()`, devolve caminho absoluto, não toca `bin/` | `cmd/mem/events_test.go` (helper) | `go test -count=1 -run TestMemBinary ./cmd/mem/...` |
| Não-regressão do binário do usuário | integration | AC1 e AC3: `bin/mem.exe` (se existir) fica byte a byte igual depois da suíte | `cmd/mem/` | `go test -count=1 ./cmd/mem/...` |
| Contrato do aviso consolidado | unit (existente) | AC5, AC6, AC7 — **preservado**, sem código novo | `cmd/mem/embed_provider_degradation_test.go` | `go test -count=1 -run TestEmbed ./cmd/mem/...` |

---

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após T1 | `go test -count=1 ./cmd/mem/...` |
| Full | Gate final (T3) | `go test -count=1 ./...` (com `MY_MEMORY_PG_TEST=1` e `MY_MEMORY_PG_URL` preservada) |
| Tagged | Gate final (T3) | `go test -tags "sqlite_fts5 onnx" -count=1 ./...` |
| Format | Antes de declarar pronto | `gofmt -l` na cópia LF dos arquivos `.go` tocados |
| Spec | Antes de entregar | `python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/fix-test-binary-clobber/spec.md` e `validate_tasks.py .specs/features/fix-test-binary-clobber/tasks.md` |
| State | Antes de entregar | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-test-binary-clobber --root D:\repository\my-memory` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: O conserto

```
T1 -> T2
```

### Phase 2: O fechamento

```
T2 -> T3
```

---

## Task Breakdown

### T1: O helper de teste compila no temp dir

**What**: os dois helpers que compilam o binário para os testes (`cmd/mem/doctor_test.go` e `cmd/mem/events_test.go`) passam a compilar num **diretório temporário próprio**, em vez de `<repoRoot>/bin/`. Vira **um** helper compartilhado (`memBinary(t)`), usado pelos dois — duplicar o build em dois lugares foi o que criou a divergência.

**Where**: `cmd/mem/events_test.go` (helper único) · `cmd/mem/doctor_test.go` (consome o helper)
**Depends on**: None
**Reuses**: o `runtime.GOOS` para o sufixo `.exe` · o `t.TempDir()` do Go · `findRepoRoot` (o `go.mod` como âncora do repo)
**Requirement**: TBC-01, TBC-02, TBC-04

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] O helper compila em `t.TempDir()` e **não** em `<repoRoot>/bin/`
- [x] `cmd/mem/doctor_test.go` usa o helper compartilhado (sem segundo `go build`)
- [x] Nenhum outro arquivo de teste do repo escreve em `<repoRoot>/bin/`
- [x] O binário continua sendo executado por caminho absoluto, com o cwd que o teste definir

**Status**: Verified (2026-11-14) — helper único em `cmd/mem/events_test.go:27` (`memBinary`), `doctor_test.go` passou a consumi-lo; o build vai para `t.TempDir()`.

**Tests**: unit
**Gate**: quick

**Commit**: `fix(test): a suite de cmd/mem compila o binario num temp dir`

---

### T2: A regressão não volta — o binário do usuário sobrevive à suíte

**What**: um teste que prova o AC3: **antes** e **depois** da suíte, o `bin/mem.exe` do usuário (se existir) tem o **mesmo SHA-256**. E um teste que prova o AC1: nenhum caminho de teste do repo escreve em `<repoRoot>/bin/`.

**Where**: `cmd/mem/test_binary_untouched_test.go` (novo)
**Depends on**: T1
**Reuses**: `crypto/sha256` · `os.ReadFile` · o helper de T1
**Requirement**: TBC-01, TBC-03

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] O teste compara o SHA-256 de `bin/mem.exe` antes e depois de rodar a
      suíte que compila o binário
- [x] O teste prova que o helper **não** devolve um caminho dentro de
      `<repoRoot>/bin/`
- [x] O teste passa com o `bin/mem.exe` ausente (repo recém-clonado)

**Status**: Verified (2026-11-14) — `cmd/mem/test_binary_untouched_test.go`: o hash do binário do usuário é conferido antes/depois, e o caminho devolvido pelo helper é provado fora de `bin/`.

**Tests**: integration
**Gate**: quick

**Commit**: `test(cmd/mem): a suite preserva o binario do usuario`

---

### T3: O fechamento

**What**: conferir o contrato do aviso consolidado (AC5, AC6, AC7 — **sem código novo**), rodar os gates full e taggeado, atualizar a ISSUE-041 (status + o conserto), o CHANGELOG e o `STATE.md`, e escrever o `validation.md`.

**Where**: `.specs/ISSUES.md` · `CHANGELOG.md` · `.specs/STATE.md` · `.specs/features/fix-test-binary-clobber/validation.md`
**Depends on**: T2
**Reuses**: `cmd/mem/embed_provider_degradation_test.go` (o contrato existente) · o template do `validate.md`
**Requirement**: TBC-05, TBC-06, TBC-07

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O aviso consolidado continua aparecendo **uma** vez por execução quando o
      provider falta, e **não** aparece quando ele existe
- [x] Gate full e gate taggeado verdes
- [x] A ISSUE-041 reflete o conserto e o status fechado
- [x] O `validation.md` existe com veredito e evidência `file:line`
- [x] `validate_spec` / `validate_tasks` / `validate_state` sem erro

**Status**: Verified (2026-11-14) — contrato do aviso preservado (suite existente verde + gate taggeado), gates full e taggeado verdes, ISSUE-041 fechada, `validation.md` com veredito PASS.

**Tests**: none (documentação)
**Gate**: full

**Commit**: `docs(spec): fecha a ISSUE-041 com a evidencia medida`

---

## Phase Execution Map

```
Phase 1 → Phase 2

Phase 1:  T1 -> T2
Phase 2:  T2 -> T3
```

**Contagem de tarefas**: 3 — cabem em um único lote (≤ ~8 por worker), então a
execução é **inline**, sem sub-agente.

---

## Diagram-Definition Cross-Check

| Task | Depends On (task body) | Diagram Shows | Status |
| ---- | ---------------------- | ------------- | ------ |
| T1 | None | (início da fase 1) | ✅ Match |
| T2 | T1 | T1 -> T2 | ✅ Match |
| T3 | T2 | T2 -> T3 | ✅ Match |

Nenhuma dependência aponta para uma fase posterior.

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | Helper de build do binário de teste | unit | unit | ✅ OK |
| T2 | Não-regressão do binário do usuário | integration | integration | ✅ OK |
| T3 | Contrato do aviso + documentação | unit (existente) | none | ✅ OK (o contrato já tem teste; a task só o confere) |

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: helper no temp dir | 2 arquivos de teste | ✅ Granular |
| T2: teste de regressão | 1 arquivo novo | ✅ Granular |
| T3: fechamento | artefatos de registro | ✅ Granular |
