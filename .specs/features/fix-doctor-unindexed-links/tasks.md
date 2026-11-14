# Tasks: fix-doctor-unindexed-links

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-13
**Issue**: ISSUE-039 (`.specs/ISSUES.md`) · fecha a ISSUE-038

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Design**: inline nesta spec (um pacote novo de resolução em disco + a classificação nos dois engines; sem mudança de schema — ADR-013 continua válida)
**Status**: Emended (2026-11-14) — a spec foi emendada **antes** de executar, porque a
medição real no Postgres do cofre central mostrou que a premissa original ("138 links,
todos existem em disco", "60/100") era falsa: são **125** que existem em disco (112 por
basename, **13 por `title:` do frontmatter**) e **13** que não existem (conceitos sem
nota); o Health Score real é **45/100**. O contrato (ACs DUL-01..DUL-09) **não** mudou —
mudou a semântica de resolução do alvo (T2) e os números esperados.

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec. Guidelines found: `AGENTS.md` (regra 4 — testes obrigatórios; regra 8 — quality gate), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`, `.github/workflows/ci.yml` (comandos de teste reais do projeto).

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Resolução da raiz do vault | unit | 1:1 com o AC9: flag > config global > cwd; path inexistente é "não verificável" | `internal/vaultfs/root_test.go` | `go test -count=1 -run TestVaultRoot ./internal/vaultfs/...` |
| Resolução do alvo em disco | unit | 1:1 com os ACs 1 e 2: basename em qualquer profundidade, caminho relativo, com/sem `.md`, **`title:` e `alias:` do frontmatter**, match exato (não fuzzy), alvo inexistente, ambiguidade, tag viva | `internal/vaultfs/target_test.go` | `go test -count=1 -run TestVaultTarget ./internal/vaultfs/...` |
| Classificação no SQLite | unit | 1:1 com os ACs 1, 2, 4 e 6: em disco → categoria nova; ausente → dead link; `--fix` apaga só o morto | `internal/db/doctor_unindexed_test.go` | `go test -count=1 -run TestDoctorUnindexed ./internal/db/...` |
| Classificação no Postgres | integration (PostgreSQL real) | o mesmo contrato no engine de produção (ACs 1, 2, 4, 5, 6, 7) | `internal/store/doctor_unindexed_test.go` | `MY_MEMORY_PG_TEST=1 go test -count=1 -run TestPostgresDoctorUnindexed ./internal/store/...` |
| Relatório e Health Score | unit | AC3, AC4 e AC5: a categoria nova não penaliza; não verificável penaliza e marca o score como provisório | `internal/store/doctor_score_test.go` | `go test -count=1 -run TestHealthScore ./internal/store/...` |
| Superfície CLI e MCP | integration | AC8 e AC9: `--vault` muda a raiz; o payload do MCP traz as três categorias | `cmd/mem/doctor_vault_test.go` | `go test -count=1 -run TestDoctorVault ./cmd/mem/...` |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após cada tarefa de código | `go build ./...` e `go vet ./...` |
| Quick | Após as tarefas de teste | `go test -count=1 -run Vault ./internal/vaultfs/... ./internal/db/... ./internal/store/...` |
| Full | Gate final (T7) | `go test -count=1 ./...` (com `MY_MEMORY_PG_TEST=1` e `MY_MEMORY_PG_URL` preservada) |
| Medido | T7 | `bin/mem.exe doctor --storage=postgres --repo repo_central` (antes/depois) e `bin/mem.exe doctor --storage=postgres --repo repo_e4c8f3b1a2d5` |
| Format | Antes de declarar pronto | `gofmt -l` na cópia LF dos arquivos `.go` tocados (o working tree local mistura CRLF) |
| Spec | Antes de entregar | `python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/fix-doctor-unindexed-links/spec.md` e `validate_tasks.py .specs/features/fix-doctor-unindexed-links/tasks.md` |
| State | Antes de entregar | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-doctor-unindexed-links --root D:\repository\my-memory` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: O verificador de disco

```
T1 -> T2
```

### Phase 2: A classificação no doctor

```
T2 -> T3 -> T4 -> T5
```

### Phase 3: A superfície

```
T5 -> T6 -> T7
```

---

## Task Breakdown

### T1: Resolver a raiz do vault

**What**: novo pacote `internal/vaultfs` com a função que decide a raiz do
vault: flag explícita > entrada do catálogo de repositórios (do repo auditado) >
`central_vault.path` do config global (cofre central) > diretório de trabalho
atual; e que diz se essa raiz é **acessível**.
**Where**: `internal/vaultfs/root.go`
**Depends on**: None
**Reuses**: `internal/config/config.go` (`CentralVault.Path` e `Repositories`,
já lidos no cascade) · `config.ExpandPath`
**Requirement**: DUL-05, DUL-09

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] A precedência flag > catálogo do repo > cofre central > cwd está
      implementada e testada
- [x] Uma raiz inexistente ou ilegível é reportada como **não acessível**, com
      o motivo
- [x] Nenhum acesso de escrita: a função é read-only

**Status**: Verified (2026-11-14) — 8 testes, gate `go build ./...` + `go vet`
verdes

**Tests**: unit
**Gate**: quick

**Commit**: `feat(vaultfs): resolve a raiz do vault e sua acessibilidade`

---

### T2: Resolver o alvo de wikilink em disco

**What**: no mesmo pacote, a função que resolve o `target_id` de uma aresta
para um arquivo real dentro da raiz do vault — **basename** em qualquer
profundidade, **caminho relativo** com e sem `.md`, **`title:` do frontmatter**
e **`alias:` do frontmatter** — e devolve o path encontrado (e como ele casou)
ou "não encontrado".
**Where**: `internal/vaultfs/target.go`
**Depends on**: T1
**Reuses**: a identidade de link do próprio produto — `documents.title` é
`COALESCE(frontmatter.title, basename)` (ISSUE-010) e o indexador reescreve
wikilink/tag contra os **títulos** indexados (`parser.ResolveTagConnections`).
O `title:` entra porque 13 das 138 arestas do cofre central resolvem **só**
por ele.
**Requirement**: DUL-01, DUL-02

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] `[[review-standards]]` casa com `catalog-skills/skills/x/review-standards.md`
- [x] `[[pasta/nota]]` casa pelo caminho relativo e cai no basename se falhar
- [x] `[[React 19 API Changes]]` casa com o arquivo cujo `title:` é esse
- [x] `[[Alias]]` casa com o arquivo que declara esse `alias:`
- [x] O match é **exato** (case-insensitive), nunca fuzzy: `[[Kubernetes]]` não
      casa por token com um arquivo qualquer
- [x] Um alvo inexistente devolve "não encontrado" (e **não** erro)
- [x] A ambiguidade (dois arquivos com o mesmo basename) devolve o path encontrado,
      para o relatório mostrar qual foi
- [x] A resolução devolve **como** casou (basename / caminho relativo / title /
      alias), para o relatório nomear o caminho resolvido

**Status**: Verified (2026-11-14) — 9 testes, gate `go build ./...` + `go vet`
verdes. **Provado contra o vault real**: os 138 alvos do `repo_central`
resolvem em 112 por basename + 13 por `title:` e ficam 13 sem resolver — os
mesmos 13 conceitos sem nota da medição.

**Tests**: unit
**Gate**: quick

**Commit**: `feat(vaultfs): resolve alvo de wikilink contra o disco do vault`

---

### T3: As categorias novas no relatório e no Health Score

**What**: acrescentar ao `DoctorReport` a categoria "fora do índice por decisão"
(alvo, origem, relação, path em disco) e a categoria "não verificável", e
ajustar a fórmula do score: a categoria nova não penaliza; a não verificável
penaliza como dead link e marca o score como provisório.
**Where**: `internal/store/store.go`
**Depends on**: T2
**Reuses**: `CalculateHealthScore` (a matemática única dos dois engines)
**Requirement**: DUL-03, DUL-04, DUL-05

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] As duas listas novas existem no `DoctorReport` e são inicializadas vazias
- [ ] A categoria nova não altera o score
- [ ] A não verificável penaliza 5 pontos cada (teto de 40, como a ADR-013)
- [ ] O relatório expõe um sinalizador de score provisório
- [ ] O teste de fórmula cobre os quatro casos (0 / só nova / não verificável /
      dead link real)

**Tests**: unit
**Gate**: quick

**Commit**: `feat(doctor): novas categorias e Health Score provisório`

---

### T4: A classificação no SQLite

**What**: o doctor do SQLite passa a classificar cada link suspeito consultando
o disco do vault, e a rotina de correção passa a apagar **exatamente** a lista
de dead links do relatório.
**Where**: `internal/db/graph.go`
**Depends on**: T3
**Reuses**: a query de dead link existente (a que já conhece `document_tags` e
`code_symbol` da ISSUE-038)
**Requirement**: DUL-01, DUL-02, DUL-06, DUL-07

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] Um link cujo alvo existe em disco vai para a categoria nova
- [ ] Um link sem alvo em disco e sem documento indexado continua dead link
- [ ] A rotina de correção apaga só as arestas da lista de dead links
- [ ] A rotina de correção preserva as arestas não verificáveis
- [ ] O teste prova que o código pré-fix falha com o sintoma exato

**Tests**: unit
**Gate**: quick

**Commit**: `fix(doctor): o SQLite distingue fora-do-indice de link rompido`

---

### T5: A classificação no Postgres

**What**: o mesmo contrato no engine de produção — mesma classificação, mesma
lista de correção.
**Where**: `internal/store/doctor.go`
**Depends on**: T4
**Reuses**: a query de dead link do Postgres (a que já conhece `document_tags` e
`code_symbol`)
**Requirement**: DUL-01, DUL-02, DUL-06, DUL-07

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] O contrato é idêntico ao do SQLite (mesmo teste, mesmo resultado)
- [ ] O filtro é por `repository`, como o resto do doctor
- [ ] O teste roda contra um Postgres real (skip sem `MY_MEMORY_PG_TEST`)

**Tests**: integration
**Gate**: quick

**Commit**: `fix(doctor): o Postgres distingue fora-do-indice de link rompido`

---

### T6: A flag `--vault` e o relatório no CLI

**What**: `mem doctor` aceita `--vault` (raiz alternativa para a checagem em
disco) e o relatório passa a mostrar as três categorias com as contagens.
**Where**: `cmd/mem/main.go`
**Depends on**: T5
**Reuses**: `displayDoctorReport` (o formatador já existente) e
`convertStoreDoctorReportToMCP`
**Requirement**: DUL-09, DUL-03

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `--vault` aparece na ajuda do comando
- [ ] O relatório mostra as três categorias e avisa quando o score é provisório
- [ ] O MCP converter leva as categorias novas

**Tests**: integration
**Gate**: quick

**Commit**: `feat(cli): mem doctor aceita --vault e mostra as tres categorias`

---

### T7: O espelho das categorias no MCP

**What**: o `DoctorReport` do MCP (`internal/mcp/handlers.go`) ganha as mesmas
categorias, para que `memory_doctor` devolva os mesmos números que a CLI.
**Where**: `internal/mcp/handlers.go`
**Depends on**: T6
**Reuses**: o espelho parcial já existente do `DoctorReport`
**Requirement**: DUL-08

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] O payload traz as três categorias com as mesmas contagens da CLI
- [ ] O contrato do wire continua valendo (sem `null`, arrays sempre presentes)
- [ ] O teste compara a contagem do MCP com a do relatório da CLI

**Tests**: integration
**Gate**: full

**Commit**: `feat(mcp): memory_doctor expoe as tres categorias de link`

---

## Phase Execution Map

```
Phase 1 → Phase 2 → Phase 3

Phase 1:  T1 -> T2
Phase 2:  T2 -> T3 -> T4 -> T5
Phase 3:  T5 -> T6 -> T7
```

Execution is strictly sequential - there is no intra-phase parallelism. A single
agent (or batch worker) works one task at a time, in order.

**Contagem de tarefas**: 7 — cabem em um único lote (≤ ~8 por worker), então a
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
| T7 | T6 | T6 -> T7 | ✅ Match |

Nenhuma dependência aponta para uma fase posterior.

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | Resolução da raiz do vault | unit | unit | ✅ OK |
| T2 | Resolução do alvo em disco | unit | unit | ✅ OK |
| T3 | Relatório e Health Score | unit | unit | ✅ OK |
| T4 | Classificação no SQLite | unit | unit | ✅ OK |
| T5 | Classificação no Postgres | integration | integration | ✅ OK |
| T6 | Superfície CLI e MCP | integration | integration | ✅ OK |
| T7 | Superfície CLI e MCP | integration | integration | ✅ OK |

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: raiz do vault | 1 função nova | ✅ Granular |
| T2: alvo em disco | 1 função nova | ✅ Granular |
| T3: categorias + score | 1 struct + 1 função | ✅ Granular |
| T4: classificação SQLite | 1 arquivo (2 funções) | ✅ Granular |
| T5: classificação Postgres | 1 arquivo (2 funções) | ✅ Granular |
| T6: flag + relatório | 1 arquivo | ✅ Granular |
| T7: espelho MCP | 1 arquivo | ✅ Granular |
