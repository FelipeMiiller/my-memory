# Tasks: fix-progressive-context-test-coverage

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-15
**Issue**: ISSUE-044 (`.specs/ISSUES.md`) · fecha os 3 gaps da feature `progressive-context-loading`

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Design**: inline nesta spec (3 testes novos + o ciclo de re-verificação; **nenhum arquivo de produção muda**)
**Status**: Draft

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec. Guidelines found: `AGENTS.md` (regra 4 — testes obrigatórios; regra 8 — quality gate), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`, `.github/workflows/ci.yml` (comandos de teste reais do projeto).

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Schema declarado da tool MCP | unit | 1:1 com PCT-01 e PCT-02: o catálogo **servido** (`GetTools()`) declara `detail_level` (enum l0/l1/l2) e `category` (enum resource/memory/skill) | `internal/mcp/tools_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestTools_SearchSchema ./internal/mcp/...` |
| Renderização L0 da CLI (SQLite) | integration | 1:1 com PCT-03: badge de categoria em MAIÚSCULO, micro-abstract presente, corpo bruto ausente, e o nível `l1` como controle negativo | `cmd/mem/search_level_l0_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestSearch_L0 ./cmd/mem/...` |
| Renderização L0 da CLI (Postgres) | integration (PostgreSQL real) | 1:1 com PCT-04: o mesmo contrato no engine de produção, contra o banco descartável do `pgtest` — é o ramo onde o mutante M6 foi injetado | `cmd/mem/search_level_l0_postgres_test.go` | `MY_MEMORY_PG_TEST=1 go -C D:\repository\my-memory test -count=1 -run TestSearch_L0Postgres ./cmd/mem/...` |
| Escolha do `abstract` no indexador | unit | 1:1 com PCT-05, PCT-06 e PCT-07: os três ramos da cadeia `internal/watcher/indexer.go:81-88` (summary / abstract / fallback de 160 caracteres) | `internal/watcher/indexer_abstract_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestIndexSingleFile_Abstract ./internal/watcher/...` |
| Verificação independente + sensor | integration | 1:1 com PCT-08 e PCT-09: as 3 mutações que sobreviveram (M2, M5, M6) morrem e o veredito da feature corrigida é PASS com 9/9 ACs | `.specs/features/progressive-context-loading/validation.md` | `python3 .agents/skills/tlc-spec-driven/scripts/validate_state.py progressive-context-loading --root D:\repository\my-memory` |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após cada task de teste | `go -C D:\repository\my-memory build ./...` e `go -C D:\repository\my-memory vet ./...` |
| Quick | Após T1, T2 e T4 (pacotes sem Postgres) | `go -C D:\repository\my-memory test -count=1 ./internal/mcp/... ./internal/watcher/... ./cmd/mem/...` |
| Full | Após T3 (Postgres) e no gate final | `go -C D:\repository\my-memory test -count=1 ./...` (com `MY_MEMORY_PG_TEST=1` e `MY_MEMORY_PG_URL` preservada) |
| Full | Gate final taggeado | `go -C D:\repository\my-memory test -tags "sqlite_fts5 onnx" -count=1 ./...` |
| Build | Antes de declarar pronto | `go -C D:\repository\my-memory build ./...` |
| Format | Antes de declarar pronto | `gofmt -l` nos `.go` tocados, medido em **cópia LF** (o working tree local mistura CRLF — o `gofmt` do repo aponta 134 arquivos por isso, e a CI roda em Linux) |
| Spec | Antes de entregar | `python3 .agents/skills/tlc-spec-driven/scripts/validate_spec.py fix-progressive-context-test-coverage --root D:\repository\my-memory` e `validate_tasks.py fix-progressive-context-test-coverage --root D:\repository\my-memory` |
| State | Antes de entregar | `python3 .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-progressive-context-test-coverage --root D:\repository\my-memory` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: Os 3 testes que faltam

As 4 tasks são **independentes** (pacotes diferentes, nenhum arquivo compartilhado) — nenhuma declara dependência de outra.

```
T1 · T2 · T3 · T4
```

### Phase 2: A re-verificação independente

O ciclo fix → re-verificar da ISSUE-044: só depois dos 4 testes é que o Verifier independente e o sensor podem rodar.

```
T1 -> T5
T2 -> T5
T3 -> T5
T4 -> T5
```

---

## Task Breakdown

### T1: O schema da `memory_search` é asserido no catálogo servido

**What**: um teste que lê o catálogo de tools **servido** pelo servidor MCP
(`Server.GetTools()`), localiza `memory_search` e afirma que o `inputSchema`
declara `detail_level` (com `enum` = `l0`, `l1`, `l2`) e `category` (com `enum` =
`resource`, `memory`, `skill`) — as duas chaves que o mutante M5 remove.
**Where**: `internal/mcp/tools_test.go`
**Depends on**: None
**Reuses**: `Server.GetTools()` (`internal/mcp/tools.go:792`) · o padrão de asserção
de schema já existente em `TestTools_DefaultSchemas` (mesmo arquivo) ·
`ToolMemorySearch.InputSchema` (`internal/mcp/tools.go:54` e `internal/mcp/tools.go:59`)
**Requirement**: PCT-01, PCT-02
**Gap**: PCL-02

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O teste lê `GetTools()` (o catálogo **servido**) e **não** a variável de pacote `ToolMemorySearch`
- [ ] `detail_level` está em `properties` e o `enum` é exatamente `["l0","l1","l2"]`
- [ ] `category` está em `properties` e o `enum` é exatamente `["resource","memory","skill"]`
- [ ] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestTools_SearchSchema ./internal/mcp/...`
- [ ] Test count: os testes de `internal/mcp` passam, sem nenhuma deleção silenciosa
- [ ] Só `internal/mcp/tools_test.go` mudou — nenhum arquivo de produção foi tocado

**Tests**: unit
**Gate**: quick

**Commit**: `test(mcp): assere o schema de detail_level e category da memory_search`

---

### T2: A saída L0 da CLI no SQLite é asserida

**What**: um teste que monta um banco SQLite temporário com uma nota cujo
frontmatter declara `category: memory` e `summary:`, roda a busca no nível `l0`
capturando o stdout e afirma que a saída traz o badge `[MEMORY]`, o
micro-abstract (o `summary`) e **não** traz o corpo bruto do chunk — com o mesmo
fixture no nível `l1` como **controle negativo** (lá o corpo aparece).
**Where**: `cmd/mem/search_level_l0_test.go`
**Depends on**: None
**Reuses**: `captureStdout` (`cmd/mem/search_tag_test.go:330`) · `runSearchSQLite`
(`cmd/mem/main.go:2448`) · o ramo L0 de `cmd/mem/main.go:2517` · o par
`db.InitDB` + `watcher.IndexSingleFileSQLite` (o mesmo par de
`internal/watcher/indexer_test.go:14`)
**Requirement**: PCT-03
**Gap**: PCL-01

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] A saída L0 contém o badge de categoria em **MAIÚSCULO** (`[MEMORY]`) e o `summary` como micro-abstract
- [ ] A saída L0 **não** contém o corpo bruto do chunk
- [ ] O controle negativo passa: no nível `l1` a saída **contém** o corpo bruto
- [ ] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestSearch_L0 ./cmd/mem/...`
- [ ] Test count: os testes de `cmd/mem` passam, sem nenhuma deleção silenciosa
- [ ] O teste usa `t.TempDir()` e não escreve em nenhum vault nem em `.memory/memory.db`

**Tests**: integration
**Gate**: quick

**Commit**: `test(cli): assere a saida L0 do mem search no SQLite`

---

### T3: A saída L0 da CLI no Postgres é asserida

**What**: o mesmo contrato de T2 no **engine de produção**, contra o banco
descartável do `pgtest` — cobrindo o ramo L0 de `runSearchPostgres`
(`cmd/mem/main.go:2368`), que é **exatamente** onde o mutante M6 foi injetado e
sobreviveu. Um teste só no SQLite (T2) **não** mataria M6.
**Where**: `cmd/mem/search_level_l0_postgres_test.go`
**Depends on**: None
**Reuses**: `pgtest.Capture()` no `init()` e `pgtest.New(t)` (o padrão de
`cmd/mem/codeindex_postgres_test.go:32` e `cmd/mem/codeindex_postgres_test.go:57`)
· `store.NewPostgresStore` · o par `store.InsertDocument` + `store.InsertChunk`
(`internal/store/postgres_test.go:231`) · `captureStdout`
**Requirement**: PCT-04
**Gap**: PCL-01

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O teste usa `pgtest.New(t)` (banco descartável) e **nunca** o Postgres configurado do dono
- [ ] Sem `MY_MEMORY_PG_TEST=1` no ambiente o teste **pulsa** (skip) em vez de falhar
- [ ] Com Postgres: a saída L0 contém o badge `[MEMORY]` e o `summary` como micro-abstract
- [ ] Com Postgres: a saída L0 **não** contém o corpo bruto, e no nível `l1` **contém** (controle negativo)
- [ ] O gate completo passa: `MY_MEMORY_PG_TEST=1 go -C D:\repository\my-memory test -count=1 ./cmd/mem/...`
- [ ] Test count: os testes de `cmd/mem` passam, sem nenhuma deleção silenciosa
- [ ] Só o arquivo de teste mudou — nenhum arquivo de produção foi tocado

**Tests**: integration
**Gate**: full

**Commit**: `test(cli): assere a saida L0 do mem search no Postgres`

---

### T4: A escolha do `abstract` no indexador é asserida

**What**: um teste na suíte do `watcher` que indexa **3** documentos — um com
`summary`, um com `abstract` e um sem nenhum dos dois — e afirma o valor gravado
em `documents.abstract` em cada um dos três ramos da cadeia de escolha de
`internal/watcher/indexer.go:81-88`.
**Where**: `internal/watcher/indexer_abstract_test.go`
**Depends on**: None
**Reuses**: `IndexSingleFileSQLite` e `db.InitDB` (o padrão de
`internal/watcher/indexer_test.go:14`) · `parser.ExtractMicroAbstract` (o fallback
de 160 caracteres)
**Requirement**: PCT-05, PCT-06, PCT-07
**Gap**: PCL-03

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] Documento com `summary` → `documents.abstract` é **exatamente** o `summary`, verbatim
- [ ] Documento com `abstract` e sem `summary` → `documents.abstract` é o `abstract` do frontmatter
- [ ] Documento sem nenhum dos dois → `documents.abstract` é o fallback do parser, com **no máximo 160** caracteres
- [ ] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestIndexSingleFile_Abstract ./internal/watcher/...`
- [ ] Test count: os testes de `internal/watcher` passam, sem nenhuma deleção silenciosa
- [ ] Só o arquivo de teste mudou — nenhum arquivo de produção foi tocado

**Tests**: unit
**Gate**: quick

**Commit**: `test(watcher): assere o summary virando abstract no indexador`

---

### T5: A feature corrigida é re-verificada por um Verifier independente

**What**: despachar um Verifier **independente** (≠ o autor de T1..T4) que
(a) re-derive a cobertura da feature `progressive-context-loading` ancorada na
spec, com evidência `file:line` para as 9 ACs; (b) rode o **sensor de
discriminação** em worktree isolado re-injetando as 3 mutações que sobreviveram
— M2 em `internal/watcher/indexer.go:82`, M5 em `internal/mcp/tools.go:54` e M6
em `cmd/mem/main.go:2368` — e confirme que **cada uma** é morta pelos testes
novos; (c) escreva o relatório
`.specs/features/progressive-context-loading/validation.md` com o veredito novo.
**Where**: `.specs/features/progressive-context-loading/validation.md`
**Depends on**: T1, T2, T3, T4
**Reuses**: o sensor anterior (6 alvos, worktree isolado) · `internal/pgtest` (o
gate do sensor precisa do Postgres para o M6) · o fallback standalone do
`tlc-spec-driven` para a análise ancorada na spec
**Requirement**: PCT-08, PCT-09
**Gap**: nenhum novo — fecha a ressalva herdada da ISSUE-044 (o sensor roda no pai)

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O Verifier é um agente **distinto** do autor de T1..T4 (autor ≠ verificador)
- [ ] O sensor re-injeta M2, M5 e M6 e **cada uma** derruba a suíte
- [ ] As 6 mutações do sensor original (M1, M3, M4 + M2, M5, M6) constam no relatório: **6/6 mortas**
- [ ] As 9 ACs da feature `progressive-context-loading` têm evidência `file:line` no relatório
- [ ] O veredito do relatório é **PASS** e `validate_state.py progressive-context-loading` sai **exit 0**
- [ ] A árvore real está intacta no fim (baseline e final idênticos, `git status --porcelain` limpo) — **sem `git stash`**
- [ ] O relatório final desta feature de correção é escrito pelo Verifier da fase final, com o diff range

**Tests**: integration
**Gate**: full

**Commit**: `docs(specs): re-verificacao independente da feature progressive-context-loading`

---

## Phase Execution Map

```
Phase 1 → Phase 2

Phase 1:  T1 · T2 · T3 · T4
Phase 2:  T5  (depende de T1, T2, T3 e T4)
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
| T2 | None | (início da fase 1) | ✅ Match |
| T3 | None | (início da fase 1) | ✅ Match |
| T4 | None | (início da fase 1) | ✅ Match |
| T5 | T1, T2, T3, T4 | T1 -> T5 · T2 -> T5 · T3 -> T5 · T4 -> T5 | ✅ Match |

Nenhuma dependência aponta para uma fase posterior.

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | Schema declarado da tool MCP | unit | unit | ✅ OK |
| T2 | Renderização L0 da CLI (SQLite) | integration | integration | ✅ OK |
| T3 | Renderização L0 da CLI (Postgres) | integration | integration | ✅ OK |
| T4 | Escolha do `abstract` no indexador | unit | unit | ✅ OK |
| T5 | Verificação independente + sensor | integration | integration | ✅ OK |

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: schema da `memory_search` | 1 arquivo de teste | ✅ Granular |
| T2: saída L0 no SQLite | 1 arquivo de teste | ✅ Granular |
| T3: saída L0 no Postgres | 1 arquivo de teste | ✅ Granular |
| T4: `abstract` no indexador | 1 arquivo de teste | ✅ Granular |
| T5: re-verificação independente | 1 relatório de verificação | ✅ Granular |
