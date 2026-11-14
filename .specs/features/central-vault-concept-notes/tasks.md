# Tasks: central-vault-concept-notes

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-14
**Issue**: fecha o último obstáculo do critério §B-2 da ADR-062 (doctor ≥ 90)
**Vault**: `G:\My Drive\central-memory` · **repo**: `repo_central` (Postgres)

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Design**: inline nesta spec (13 notas de conteúdo no vault + 2 READMEs + um reindex; **nenhuma** mudança de código)
**Status**: Ready (2026-11-14)

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec. Guidelines found: `AGENTS.md` (regra 4 — testes obrigatórios; regra 8 — quality gate), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`, `.agents/skills/my-memory-format/SKILL.md` (frontmatter canônico) e a nota `standards/navegacao-moc.md` do próprio cofre (regra do MOC).

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Notas do cofre | verificação de conteúdo | 1:1 com os ACs 3, 4, 5, 9, 10: nome do arquivo = texto do wikilink; `title` = conceito; frontmatter canônico; `# Título` no corpo; backlink para o README e para a nota de origem; "O que é" + "Onde entra aqui"; ao menos uma relacionada | `cloud/*.md` · `database/*.md` | script de verificação (T5) |
| Índice dos READMEs | verificação de conteúdo | AC6: as 13 notas listadas no README da pasta; `database/README.md` existe e segue o padrão do MOC | `cloud/README.md` · `database/README.md` | script de verificação (T5) |
| Índice do cofre | integração | AC1, AC2, AC7, AC8: os 13 wikilinks resolvem, 0 link rompido, 0 chunk sem vetor, score ≥ 90 | Postgres `repo_central` | `mem doctor --storage=postgres --repo repo_central` |

---

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após cada tarefa de conteúdo | script de verificação das 13 notas (T5) — frontmatter, nome, backlinks, seções |
| Medido | T4 | `mem index -dir "G:\My Drive\central-memory" -storage postgres -repo repo_central -force` (binário com a tag `onnx`) |
| Medido | T5 | `bin/mem.exe doctor --storage=postgres --repo repo_central` |
| Spec | Antes de entregar | `python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/central-vault-concept-notes/spec.md` e `validate_tasks.py .specs/features/central-vault-concept-notes/tasks.md` |
| State | Antes de entregar | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py central-vault-concept-notes --root D:\repository\my-memory` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: As notas

```
T1 -> T2
```

### Phase 2: O índice e a prova

```
T2 -> T3 -> T4 -> T5
```

### Phase 3: O fechamento

```
T5 -> T6
```

---

## Task Breakdown

### T1: As 6 notas de `cloud/`

**What**: criar a nota de cada conceito linkado por `cloud/blue-green-deployment-obsidian.md` — `Docker`, `Kubernetes`, `Observabilidade`, `Rollback`, `Expand and Contract`, `SRE e SLOs` — com o nome do arquivo **igual** ao texto do wikilink.

**Where**: `G:\My Drive\central-memory\cloud\*.md`
**Depends on**: None
**Reuses**: o frontmatter canônico da skill `my-memory-format` · a estrutura de nota do cofre (`category: resource`, `summary`, `tags`, `status: ativo`) · a regra do MOC (`standards/navegacao-moc.md`)
**Requirement**: CCN-01, CCN-03, CCN-04, CCN-05, CCN-09, CCN-10, CCN-11

**Tools**:
- MCP: NONE
- Skill: `my-memory-format`

**Done when**:

- [x] Os 6 arquivos existem com o nome do texto do wikilink (`Docker.md`, …)
- [x] O `title` de cada um é exatamente o conceito
- [x] O frontmatter tem `title`, `category: resource`, `summary`, `tags`, `status: ativo`
- [x] O corpo começa com `# <título>` e tem "O que é", "Onde entra aqui" e "Relacionado"
- [x] Cada uma tem o backlink para `[[README]]` da pasta e para a nota que a linka

**Status**: Verified (2026-11-14) — 6 notas criadas com `mem note create` (escrita atômica, UTF-8 sem BOM); verificação por script: nome, `title`, frontmatter, seções e backlinks.

**Tests**: none (conteúdo — a verificação é o script do T5)
**Gate**: quick

**Commit**: conteúdo no vault (fora do repo) — sem commit no repo

---

### T2: As 7 notas de `database/`

**What**: criar a nota de cada conceito linkado por `database/PostgreSQL — Visão Geral de Recursos e Ferramentas.md` — `Banco de Dados`, `Índices SQL`, `Busca Vetorial e RAG`, `Filas e Processamento Assíncrono`, `Transações e Concorrência`, `Observabilidade de Banco de Dados`, `Backup e Recuperação`.

**Where**: `G:\My Drive\central-memory\database\*.md`
**Depends on**: T1
**Reuses**: idem T1
**Requirement**: CCN-01, CCN-03, CCN-04, CCN-05, CCN-09, CCN-10, CCN-11

**Tools**:
- MCP: NONE
- Skill: `my-memory-format`

**Done when**:

- [x] Os 7 arquivos existem com o nome do texto do wikilink
- [x] O `title` de cada um é exatamente o conceito
- [x] O frontmatter canônico está completo
- [x] O corpo tem "O que é", "Onde entra aqui" e "Relacionado"
- [x] Cada uma tem o backlink para o README da pasta e para a nota que a linka

**Status**: Verified (2026-11-14) — 7 notas criadas; verificação por script.

**Tests**: none (conteúdo)
**Gate**: quick

**Commit**: conteúdo no vault (fora do repo) — sem commit no repo

---

### T3: Os READMEs das pastas

**What**: listar as 6 notas novas no README de `cloud/` e **criar** o README de `database/` (que não existia — a regra 1 do MOC exige uma pasta com README, e a pasta já tinha conteúdo).

**Where**: `G:\My Drive\central-memory\cloud\README.md` · `G:\My Drive\central-memory\database\README.md`
**Depends on**: T2
**Reuses**: o padrão do README de `cloud/` e o template do MOC
**Requirement**: CCN-06

**Tools**:
- MCP: NONE
- Skill: `my-memory-format`

**Done when**:

- [x] O README de `cloud/` lista as 6 notas novas, com uma linha de "o que resolve" cada
- [x] `database/README.md` existe, com o frontmatter padrão e a lista das 8 notas da pasta
- [x] Os dois READMEs têm a seção "Relacionado" com o índice-pai

**Status**: Verified (2026-11-14) — os dois READMEs conferidos por script (frontmatter + entradas).

**Tests**: none (conteúdo)
**Gate**: quick

**Commit**: conteúdo no vault (fora do repo) — sem commit no repo

---

### T4: O reindex do cofre com o provider

**What**: reindexar o cofre central **com o binário taggeado** (`-tags "sqlite_fts5 onnx"`), para (a) as 13 notas entrarem no índice e (b) os **101 chunks** que estavam sem vetor real serem re-embaralhados pelo provider.

**Where**: Postgres `repo_central` (a operação; o binário é `bin/mem.exe` do repo)
**Depends on**: T3
**Reuses**: `mem index -dir <vault> -storage postgres -repo repo_central -force`
**Requirement**: CCN-01, CCN-07

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] O reindex termina sem erro, com o provider ONNX presente
- [x] O índice passa a ter os 13 documentos novos (94 → 107)
- [x] Nenhum chunk fica sem vetor real

**Status**: Verified (2026-11-14) — reindex com `-tags "sqlite_fts5 onnx"`: **108 documentos** (94 + 13 notas + o README novo de `database/`), 108 indexados, 0 podados, 0 falhos, **0 chunk sem vetor real** (eram 101).

**Tests**: none (operação — a verificação é o doctor do T5)
**Gate**: medido

**Commit**: operação — sem commit no repo

---

### T5: A medição

**What**: medir o `repo_central` com o doctor e conferir os 11 ACs.

**Where**: Postgres `repo_central` · `G:\My Drive\central-memory`
**Depends on**: T4
**Reuses**: `mem doctor --storage=postgres --repo repo_central`
**Requirement**: CCN-01, CCN-02, CCN-06, CCN-07, CCN-08, CCN-11

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] O doctor reporta **0 link rompido**
- [x] O Health Score é **≥ 90**
- [x] O doctor reporta **0 chunk sem vetor real**
- [x] As 13 notas não aparecem como órfãs
- [x] Um script confere nome, `title`, frontmatter, seções e backlinks das 13 notas

**Status**: Verified (2026-11-14) — **0 link rompido, Health Score 100/100, 0 chunk sem vetor real, 0 órfã**; script de verificação: 13/13 notas conformes.

**Tests**: integration (o doctor contra o Postgres real)
**Gate**: medido

**Commit**: medição — sem commit no repo

---

### T6: O fechamento

**What**: registrar a decisão no repositório — ADR (a convenção "conceito linkado tem nota"), `STATE.md` (decisões + handoff), `ISSUES.md` (a ISSUE-039 fecha de vez) e o `validation.md` da spec.

**Where**: `docs/adr/064-*.md` · `docs/adr/README.md` · `docs/README.md` · `.specs/STATE.md` · `.specs/ISSUES.md` · `.specs/features/central-vault-concept-notes/validation.md`
**Depends on**: T5
**Reuses**: o formato MADR da skill `create-adr` · o template do `validate.md`
**Requirement**: CCN-08

**Tools**:
- MCP: NONE
- Skill: `create-adr` · `tlc-spec-driven`

**Done when**:

- [x] A ADR registra a convenção "todo conceito linkado tem nota"
- [x] O `STATE.md` registra a decisão do dono e o resultado medido
- [x] O `ISSUES.md` reflete o fechamento
- [x] O `validation.md` existe com veredito e evidência `file:line`
- [x] `validate_spec` / `validate_tasks` / `validate_state` sem erro

**Status**: Verified (2026-11-14) — ADR-064, STATE.md, ISSUES.md e validation.md commitados; validadores com 0 erro.

**Tests**: none (documentação)
**Gate**: spec

**Commit**: `docs(spec): fecha o cofre central com as 13 notas de conceito`

---

## Phase Execution Map

```
Phase 1 → Phase 2 → Phase 3

Phase 1:  T1 -> T2
Phase 2:  T2 -> T3 -> T4 -> T5
Phase 3:  T5 -> T6
```

**Contagem de tarefas**: 6 — cabem em um único lote (≤ ~8 por worker), então a
execução é **inline**, sem sub-agente.

---

## Diagram-Definition Cross-Check

| Task | Depends On (task body) | Diagram Shows | Status |
| ---- | ---------------------- | ------------- | ------ |
| T1 | None | (início da fase 1) | ✅ Match |
| T2 | T1 | T1 -> T2 | ✅ Match |
| T3 | T2 | T2 -> T3 | ✅ Match |
| T4 | T3 | T3 -> T4 | ✅ Match |
| T5 | T4 | T4 -> T5 | ✅ Match |
| T6 | T5 | T5 -> T6 | ✅ Match |

Nenhuma dependência aponta para uma fase posterior.

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | Notas do cofre | verificação de conteúdo | none | ✅ OK (a verificação é o script do T5) |
| T2 | Notas do cofre | verificação de conteúdo | none | ✅ OK |
| T3 | Índice dos READMEs | verificação de conteúdo | none | ✅ OK |
| T4 | Índice do cofre | integração | none | ✅ OK (a verificação é o doctor do T5) |
| T5 | Índice do cofre | integração | integration | ✅ OK |
| T6 | Documentação | none | none | ✅ OK |

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: notas de cloud | 6 notas, 1 pasta | ✅ Granular |
| T2: notas de database | 7 notas, 1 pasta | ✅ Granular |
| T3: READMEs | 2 arquivos de índice | ✅ Granular |
| T4: reindex | 1 operação | ✅ Granular |
| T5: medição | 1 medição + 1 script | ✅ Granular |
| T6: fechamento | artefatos de registro | ✅ Granular |
