# Tasks: fix-ears-gate-blindness

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-16
**Issue**: ISSUE-050 (`.specs/ISSUES.md`) · fecha a cegueira do gate de EARS

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Design**: inline nesta spec (o coletor volta a ver o bloco inteiro · teste de regressão no padrão da skill · medição republicável · remediação das specs reveladas)
**Status**: Draft

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec. Guidelines found: `AGENTS.md` (regra 4 — testes obrigatórios; regra 8 — quality gate), `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py` (convenção de teste de script da skill: `assert` + `main()` próprio, sem pytest), `.agents/rules/always-quality-gate.md`.

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Gate de EARS (o validador) | unit | 1:1 com EAG-01 a EAG-06 e EAG-09: os **dois** formatos de bloco, o AC sem SHALL (ERROR), o AC com SHALL e sem keyword (WARNING), o placeholder (sem erro), e o reset reintroduzido (o teste falha) | `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py` | `python3 .agents/skills/tlc-spec-driven/scripts/test_validate_spec.py` |
| Medição da cegueira | integration | 1:1 com EAG-07: **0** blocos cegos e "ACs vistos = ACs presentes" no repo inteiro | `.agents/skills/tlc-spec-driven/scripts/count_ears_visibility.py` | `python3 .agents/skills/tlc-spec-driven/scripts/count_ears_visibility.py --root D:\repository\my-memory` |
| Remediação das specs | integration | 1:1 com EAG-08: nenhuma spec do repo reprova no gate sem correção ou dispensa escrita dentro dela | `.specs/features/` | `python3 .agents/skills/tlc-spec-driven/scripts/validate_spec.py <feature> --root D:\repository\my-memory` (por feature) |
| Contrato (template da skill + exemplar) | none | build gate only — o contrato é a **leitura cruzada**: o template e o exemplar continuam no formato atual e continuam passando | `.agents/skills/tlc-spec-driven/references/specify.md` | build gate only |
| Verificação independente + sensor | integration | 1:1 com EAG-10: a correção do gate é verificada por um Verifier distinto do autor, com o reset reintroduzido como mutante | `.specs/features/fix-ears-gate-blindness/validation.md` | `python3 .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-ears-gate-blindness --root D:\repository\my-memory` |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após T1 (o coletor) | `python3 .agents/skills/tlc-spec-driven/scripts/validate_spec.py fix-doctor-unindexed-links --root D:\repository\my-memory` (o exemplar **ccego** passa a ser lido — é o contra-teste) |
| Quick | Após T2 (o teste do gate) | `python3 .agents/skills/tlc-spec-driven/scripts/test_validate_spec.py` |
| Quick | Após T3 (a medição) | `python3 .agents/skills/tlc-spec-driven/scripts/count_ears_visibility.py --root D:\repository\my-memory` (0 blocos cegos) |
| Full | Gate final (T5) | `go -C D:\repository\my-memory test -count=1 ./...` (o repo segue verde) e o gate rodado em **todas** as specs do repo |
| Build | Antes de declarar pronto | `go -C D:\repository\my-memory build ./...` |
| Format | Antes de declarar pronto | `gofmt -l` — nenhum `.go` é tocado; os scripts são Python (UTF-8 sem BOM, LF) |
| Spec | Antes de entregar | `python3 .agents/skills/tlc-spec-driven/scripts/validate_spec.py fix-ears-gate-blindness --root D:\repository\my-memory` e `validate_tasks.py fix-ears-gate-blindness --root D:\repository\my-memory` |
| State | Antes de entregar | `python3 .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-ears-gate-blindness --root D:\repository\my-memory` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: O coletor e a sua prova

```
T1 -> T2
T1 -> T3
```

### Phase 2: A remediação do que o gate passa a enxergar

```
T2 -> T4
T3 -> T4
```

### Phase 3: A re-verificação

```
T1 -> T5
T2 -> T5
T3 -> T5
T4 -> T5
```

---

## Task Breakdown

### T1: O coletor volta a ver o bloco inteiro

**What**: a condição que encerra o bloco de Acceptance Criteria deixa de zerar
`in_ac` em linha vazia — o bloco passa a terminar **só** em heading (`#{1,3}`)
ou em linha iniciada por `**`. É a correção de uma linha que a ISSUE-050
apontou.
**Where**: `.agents/skills/tlc-spec-driven/scripts/validate_spec.py`
**Depends on**: None
**Reuses**: o coletor existente (`validate_spec.py:156-176`) · a função
`classify_ears` (que já sabe dizer "no SHALL") · a decisão D1/D2 da spec
**Requirement**: EAG-01, EAG-02, EAG-06
**Gap**: ISSUE-050

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] A condição de fim de bloco não inclui mais `stripped == ""`
- [x] O bloco termina em heading (`#{1,3}`) ou em linha que começa com `**`
- [x] Uma spec com AC malformado **depois** de uma linha em branco faz o gate sair com código **!= 0**, nomeando a linha e o critério
- [x] O gate continua ignorando listas numeradas que **não** estão dentro de um bloco de AC
- [x] As demais seções do validador (assumptions, traceability, placeholders) **não** mudaram de comportamento
- [x] O script continua em UTF-8 sem BOM e LF

**Tests**: unit
**Gate**: quick

**Commit**: `fix(skill): o gate de EARS volta a ver o bloco de criterios`

---

### T2: O teste de regressão do gate

**What**: um `test_validate_spec.py` no padrão da skill (`assert` + `main()`
próprio, sem pytest) que prova os **dois** formatos de bloco, o AC sem SHALL
(ERROR), o AC com SHALL e sem keyword (WARNING), o placeholder (sem erro) e o
reset reintroduzido (o teste **falha**).
**Where**: `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py`
**Depends on**: T1
**Reuses**: a convenção do `test_validate_state.py` (mesma pasta, mesmo padrão de
`main()` e de `sys.path.insert`) · os fixtures em `tempfile` que o gate aceita
**Requirement**: EAG-03, EAG-04, EAG-05, EAG-09
**Gap**: ISSUE-050

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O teste cobre o bloco **com** linha em branco depois do cabeçalho e o bloco **sem** ela
- [ ] O teste cobre o AC sem SHALL (erro), o AC com SHALL e sem keyword (warning) e o placeholder (sem erro)
- [ ] O teste **falha** se a condição de linha em branco voltar ao coletor
- [ ] O script roda sozinho (`python3 test_validate_spec.py`) e sai com 0
- [ ] O script está em UTF-8 sem BOM e LF

**Tests**: unit
**Gate**: quick

**Commit**: `test(skill): o gate de EARS ganha teste de regressao`

---

### T3: A medição republicável da cegueira

**What**: um script que percorre as specs do repo, conta os ACs **presentes** e
os ACs **vistos** pelo gate, e imprime os **blocos cegos** (os que o gate não
enxerga) — para que o número da ISSUE-050 (311 vistos vs 752 presentes, 441
cegos, 75 blocos) deixe de depender de um scratch descartado.
**Where**: `.agents/skills/tlc-spec-driven/scripts/count_ears_visibility.py`
**Depends on**: T1
**Reuses**: a lógica de coleta do `validate_spec.py` (mesma pasta, importável) ·
o formato de saída do `lessons.py` (mesma família de scripts da skill)
**Requirement**: EAG-07
**Gap**: ISSUE-050

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O script imprime: ACs vistos, ACs presentes, ACs cegos, specs com bloco cego e total de blocos cegos
- [ ] Ele sai com código **!= 0** quando há bloco cego (para servir de gate)
- [ ] Rodado no estado **antes** da correção, reproduz o baseline da ISSUE-050
- [ ] Rodado **depois** da correção, imprime 0 blocos cegos e vistos = presentes
- [ ] O script está em UTF-8 sem BOM e LF

**Tests**: integration
**Gate**: quick

**Commit**: `feat(skill): script que mede a cegueira do gate de EARS`

---

### T4: A remediação das specs que o gate passa a apontar

**What**: rodar o gate corrigido em **todas** as specs do repo e corrigir os ACs
que ele reprovar — sem afrouxar o gate. Para spec **histórica ou arquivada**, a
dispensa vai **escrita dentro da spec**, com o motivo.
**Where**: `.specs/features/`
**Depends on**: T2, T3
**Reuses**: o `validate_spec.py` corrigido (T1) · a medição (T3) como lista de
trabalho · a convenção de dispensa registrada dentro da spec
**Requirement**: EAG-08
**Gap**: ISSUE-050 (a revelação)

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O gate corrigido roda em **todas** as specs do repo e a lista de falhas está na medição
- [ ] Toda spec reprovada tem os ACs **corrigidos** (SHALL + padrão EARS) — nenhum gate foi afrouxado
- [ ] Toda spec dispensada tem a dispensa **escrita dentro dela**, com o motivo
- [ ] Nenhuma spec continua passando por cegueira: a contagem de ACs vistos é igual à de ACs presentes
- [ ] O gate completo do repo continua verde: `go -C D:\repository\my-memory test -count=1 ./...`

**Tests**: integration
**Gate**: full

**Commit**: `fix(specs): os ACs que o gate cego deixava passar`

---

### T5: A correção do gate é re-verificada por um Verifier independente

**What**: despachar um Verifier **independente** (≠ o autor de T1..T4) que
(a) rode a bateria de specs do repo com o gate corrigido e confirme que a
**prova empírica** da ISSUE-050 se inverte (a spec com AC sem SHALL agora
falha); (b) rode o **sensor de discriminação** do gate, injetando a condição de
linha em branco de volta no coletor, e confirme que o `test_validate_spec.py`
**falha**; (c) escreva o relatório
`.specs/features/fix-ears-gate-blindness/validation.md` com o veredito novo.
**Where**: `.specs/features/fix-ears-gate-blindness/validation.md`
**Depends on**: T1, T2, T3, T4
**Reuses**: a spec de fixture da ISSUE-050 (copia do `spec.md` do exemplar com um
AC sem SHALL) · o worktree isolado (sem `git stash`)
**Requirement**: EAG-10
**Gap**: ressalva do ciclo fix → re-verificar

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O Verifier é um agente **distinto** do autor de T1..T4 (autor ≠ verificador)
- [ ] A prova empírica da ISSUE-050 se inverte: a spec de fixture com AC sem SHALL faz o gate sair com código **!= 0**
- [ ] O sensor reintroduz o reset e o `test_validate_spec.py` **falha**
- [ ] A medição final reporta 0 blocos cegos e vistos = presentes
- [ ] O veredito do relatório é **PASS** e `validate_state.py fix-ears-gate-blindness` sai **exit 0**
- [ ] A árvore real está intacta no fim (baseline e final idênticos, `git status --porcelain` limpo) — **sem `git stash`**

**Tests**: integration
**Gate**: full

**Commit**: `docs(specs): re-verificacao independente da correcao do gate de EARS`

---

## Phase Execution Map

```
Phase 1 → Phase 2 → Phase 3

Phase 1:  T1 · T2 · T3
Phase 2:  T4
Phase 3:  T5
```

Execution is strictly sequential - there is no intra-phase parallelism. A single agent (or batch worker) works one task at a time, in order.

**Contagem de tarefas**: 5 — cabem em um único lote (≤ ~8 por worker), então a
execução é **inline**, sem sub-agente de lote. A T5 despacha o **Verifier**
independente, que é parte do fluxo do `tlc-spec-driven` (nunca opcional, nunca
perguntado).

---

## Diagram-Definition Cross-Check

| Task | Depends On (task body) | Diagram Shows | Status |
| ---- | ---------------------- | ------------- | ------ |
| T1 | None | (início da fase 1) | ✅ Match |
| T2 | T1 | T1 -> T2 | ✅ Match |
| T3 | T1 | T1 -> T3 | ✅ Match |
| T4 | T2, T3 | T2 -> T4 · T3 -> T4 | ✅ Match |
| T5 | T1, T2, T3, T4 | T1 -> T5 · T2 -> T5 · T3 -> T5 · T4 -> T5 | ✅ Match |

Nenhuma dependência aponta para uma fase posterior.

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | Gate de EARS (o validador) | unit | unit | ✅ OK |
| T2 | Gate de EARS (o validador) | unit | unit | ✅ OK |
| T3 | Medição da cegueira | integration | integration | ✅ OK |
| T4 | Remediação das specs | integration | integration | ✅ OK |
| T5 | Verificação independente + sensor | integration | integration | ✅ OK |

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: o coletor | 1 condição em 1 script | ✅ Granular |
| T2: o teste do gate | 1 arquivo de teste | ✅ Granular |
| T3: a medição | 1 script | ✅ Granular |
| T4: a remediação | as specs apontadas pela medição (1 lotecoeso, com a lista no relatório) | ✅ Granular |
| T5: re-verificação independente | 1 relatório de verificação | ✅ Granular |
