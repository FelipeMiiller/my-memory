# Tasks: fix-adr-035-dead-links

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-16
**Issue**: ISSUE-049 (`.specs/ISSUES.md`) · fecha os 2 links rompidos do vault do repositório

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Design**: inline nesta spec (correção de **conteúdo** em 2 artefatos + a regra registrada como lição + a prova pelo `mem doctor`; **nenhum arquivo de ADR é tocado**)
**Status**: Complete

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec. Guidelines found: `AGENTS.md` (regra 1 — UTF-8 sem BOM; regra 4 — testes; regra 7 — docs/specs sincronizados), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`.

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Conteúdo dos artefatos da feature | integration | 1:1 com ADL-01, ADL-02 e ADL-05: as **3** ocorrências estão na forma path-style com alias e o display continua `ADR-035` | `.specs/features/embedder-builtin-onnx-minilm/spec.md` · `.specs/features/embedder-builtin-onnx-minilm/tasks.md` | `grep -rn "\[\[ADR-035\]\]" .specs/` (deve sair **vazio**) |
| Medição no vault | integration | 1:1 com ADL-03 e ADL-04: o `mem doctor` não reporta link rompido de ADR e a ADR-035 **não** foi tocada pelo diff | vault do repositório (`repo_e4c8f3b1a2d5`) | `mem doctor --storage=postgres --repo repo_e4c8f3b1a2d5` |
| Store de lições | none | o gate é a **listagem** do store — não é camada de código | `.specs/lessons.json` | `python3 .agents/skills/tlc-spec-driven/scripts/lessons.py list --status confirmed` |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após T1, T2 e T3 (conteúdo) | `grep -rn "\[\[ADR-035\]\]" .specs/` (deve sair **vazio**) e `git diff --name-only docs/adr/` (deve sair **vazio**) |
| Quick | Após T3 (lição) | `python3 .agents/skills/tlc-spec-driven/scripts/lessons.py list --status confirmed` (a regra de path-style aparece) |
| Full | Gate final (T4) | `mem index --force` + `mem doctor --storage=postgres --repo repo_e4c8f3b1a2d5` (0 links rompidos de ADR) e `go -C D:\repository\my-memory test -count=1 ./...` (o repo segue verde — a correção é de conteúdo) |
| Build | Antes de declarar pronto | `go -C D:\repository\my-memory build ./...` |
| Format | Antes de declarar pronto | `gofmt -l` nos `.go` tocados — nenhum é tocado nesta correção |
| Spec | Antes de entregar | `python3 .agents/skills/tlc-spec-driven/scripts/validate_spec.py fix-adr-035-dead-links --root D:\repository\my-memory` e `validate_tasks.py fix-adr-035-dead-links --root D:\repository\my-memory` |
| State | Antes de entregar | `python3 .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-adr-035-dead-links --root D:\repository\my-memory` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: A correção do texto e a regra

```
T1 -> T2
T1 -> T3
```

### Phase 2: A prova

```
T1 -> T4
T2 -> T4
```

---

## Task Breakdown

### T1: As 2 ocorrências do `spec.md` viram path-style

**What**: as 2 ocorrências de `[[ADR-035]]` na spec da feature
`embedder-builtin-onnx-minilm` passam a ser
`[[docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md|ADR-035]]`.
**Where**: `.specs/features/embedder-builtin-onnx-minilm/spec.md`
**Depends on**: None
**Reuses**: a regra de path-style do próprio repo (`.specs/042-stubs-editoriais/validation.md:90`) ·
o alias preserva o display curto `ADR-035`
**Requirement**: ADL-01, ADL-05
**Gap**: ISSUE-049

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] A linha 11 (`> **Tipo:** Feature. Executa o plano de implementação do [[…]]`) usa a forma path-style com alias
- [x] A linha 188 (`- [[…]] — a decisão que esta spec executa`) usa a forma path-style com alias
- [x] O display continua sendo `ADR-035` (o alias)
- [x] `grep -rn "\[\[ADR-035\]\]" .specs/` não devolve ocorrência **neste** arquivo
- [x] O arquivo continua em UTF-8 sem BOM e LF

**Tests**: integration
**Gate**: quick

**Commit**: `docs(spec): o link da ADR-035 vira path-style com alias`

---

### T2: A ocorrência do `tasks.md` vira path-style

**What**: a ocorrência de `[[ADR-035]]` na linha 3 do `tasks.md` da feature
(`**ADR:** [[ADR-035]]`) passa a ser
`[[docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md|ADR-035]]`.
**Where**: `.specs/features/embedder-builtin-onnx-minilm/tasks.md`
**Depends on**: T1
**Reuses**: a mesma forma do T1
**Requirement**: ADL-02
**Gap**: ISSUE-049

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] A linha 3 usa a forma path-style com alias, com o display `ADR-035`
- [x] `grep -rn "\[\[ADR-035\]\]" .specs/` não devolve ocorrência **neste** arquivo
- [x] O arquivo continua em UTF-8 sem BOM e LF

**Tests**: integration
**Gate**: quick

**Commit**: `docs(spec): o link da ADR-035 no tasks vira path-style com alias`

---

### T3: A regra de path-style vai para o store de lições

**What**: registrar, pelo script `lessons.py`, a lição de que **refs cruzadas
entre docs usam path-style com alias** — a regra já escrita no relatório de
2026-09-09, que nunca entrou no store (que é o que a skill carrega).
**Where**: `.specs/lessons.json`
**Depends on**: T1
**Reuses**: `lessons.py add` (o store é machine-owned — o `LESSONS.md` é
renderizado, não editado à mão)
**Requirement**: ADL-06
**Gap**: ISSUE-049

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] A lição entra no store pelo `lessons.py` (não por edição manual)
- [x] `lessons.py list --status confirmed` mostra a regra de path-style
- [x] A lição cita a evidência (`internal/parser/parser_test.go:13` — o parser trata a forma com alias) e a regra de origem (`.specs/042-stubs-editoriais/validation.md:90`)

**Tests**: none (store de lições — o gate é a listagem do `lessons.py`)
**Gate**: quick

**Commit**: `docs(lessons): refs cruzadas usam path-style com alias`

---

### T4: A prova — reindex e doctor

**What**: reindexar o vault do repositório e rodar o `mem doctor`, para provar
que os 2 links rompidos sumiram — e que **nenhum** arquivo de ADR foi tocado.
**Where**: — (operação no vault do repositório; nenhum arquivo do repo muda)
**Depends on**: T1, T2
**Reuses**: `mem index --force` · `mem doctor --storage=postgres --repo
repo_e4c8f3b1a2d5` (o mesmo comando que mediu os 2 links em 2026-11-15)
**Requirement**: ADL-03, ADL-04
**Gap**: ISSUE-049

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] `mem index --force` rodou no vault do repositório
- [x] O `mem doctor` **não** reporta nenhum link rompido cujo alvo seja um wikilink de ADR (hoje: 2 arestas)
- [x] `git diff --name-only docs/adr/` sai **vazio** — a ADR-035 não foi tocada
- [x] O Health Score do vault subiu: os **10 pontos** consumidos por esses 2 links não são mais perdidos
- [x] O gate completo do repo continua verde: `go -C D:\repository\my-memory test -count=1 ./...`

**Tests**: integration
**Gate**: full

**Commit**: `docs(specs): a correcao dos links da ADR-035 e a medicao do doctor`

---

## Phase Execution Map

```
Phase 1 → Phase 2

Phase 1:  T1 · T2 · T3
Phase 2:  T4
```

Execution is strictly sequential - there is no intra-phase parallelism. A single agent (or batch worker) works one task at a time, in order.

**Contagem de tarefas**: 4 — cabem em um único lote (≤ ~8 por worker), então a
execução é **inline**, sem sub-agente de lote.

---

## Diagram-Definition Cross-Check

| Task | Depends On (task body) | Diagram Shows | Status |
| ---- | ---------------------- | ------------- | ------ |
| T1 | None | (início da fase 1) | ✅ Match |
| T2 | T1 | T1 -> T2 | ✅ Match |
| T3 | T1 | T1 -> T3 | ✅ Match |
| T4 | T1, T2 | T1 -> T4 · T2 -> T4 | ✅ Match |

Nenhuma dependência aponta para uma fase posterior.

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | Conteúdo dos artefatos da feature | integration | integration | ✅ OK |
| T2 | Conteúdo dos artefatos da feature | integration | integration | ✅ OK |
| T3 | Store de lições | none | none | ✅ OK (o gate é a listagem do `lessons.py`) |
| T4 | Medição no vault | integration | integration | ✅ OK |

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: 2 ocorrências do spec.md | 1 arquivo, 2 linhas | ✅ Granular |
| T2: 1 ocorrência do tasks.md | 1 arquivo, 1 linha | ✅ Granular |
| T3: lição no store | 1 entrada | ✅ Granular |
| T4: reindex + doctor | 1 medição | ✅ Granular |
