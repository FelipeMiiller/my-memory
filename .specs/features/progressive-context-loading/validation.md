# progressive-context-loading Validation

**Date**: 2026-11-15
**Spec**: `.specs/features/progressive-context-loading/spec.md`
**Diff range**: `9158da3` (T1) → `dd108b9` (T5) · **spec da feature**: 9 ACs (P1..P4)
**Verifier**: sub-agente `verifier` independente (gate e análise) + **sensor retomado no pai** (o sub-agente encerrou o turno antes do sensor e **antes** de escrever o relatório)

## Validation: PARTIAL

**Result**: PARTIAL

> **Ressalva de independência.** O sub-agente Verifier fez a análise ancorada na
> spec, o gate e o mapeamento da superfície de diff — e **morreu** (o turno
> encerrou) ao preparar o sensor, sem escrever o relatório. O **sensor de
> discriminação foi executado no pai**, pelo fallback standalone do próprio
> `tlc-spec-driven` ("without sub-agents, run validate.md as an independent
> fresh-eyes pass ... including the discrimination sensor"), sobre o worktree
> isolado que o sub-agente deixou pronto, com os **6 alvos de mutação que ele
> havia localizado**. O veredito é **PARTIAL** porque o sensor achou **3
> caminhos sem teste** — e essa é a informação mais importante deste relatório.

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1 — esquema + extração de L0/category | ✅ Done | migração `abstract`+`category` nos dois motores; parser extrai `category`, `summary`, `abstract` |
| T2 — níveis e filtro de categoria no motor | ✅ Done | filtro `category` no SQL; projeção L0 no RRF (omite o corpo bruto) |
| T3 — CLI `mem search --level/--category` | ⚠️ **Parcial** | as **flags** existem e são testadas; a **formatação L0 da saída não tem asserção** (gap PCL-01) |
| T4 — tool MCP `memory_search` com `detail_level`/`category` | ⚠️ **Parcial** | o **handler** serializa por nível (testado); o **schema JSON não tem asserção** (gap PCL-02) |
| T5 — documentação | ✅ Done | `docs/CLI_GUIDE.md` e a tool schema descritos |

**Cobertura**: 5 tasks, 3 fechadas, 2 com lacuna de **teste** (não de comportamento).

---

## Spec-Anchored Acceptance Criteria

| Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion | Result |
| ------------------------- | -------------------- | ----------------------- | ------ |
| **P1-1** — migração adiciona `abstract` e `category` | as duas colunas existem nos dois motores, sem perda de dado | `internal/db/store_test.go:206` — `TestSchemaMigration_AbstractAndCategory` (SQLite, banco antigo sem as colunas); `internal/store/postgres_test.go:68` — `TestPostgresSchema_AbstractAndCategory` | ✅ PASS |
| **P1-2** — o parser extrai `category` (default `resource`) | `category` vem do frontmatter, normalizada | `internal/parser/parser_test.go:255` — `TestFrontmatter_CategoryAndAbstract`; implementação em `internal/parser/frontmatter.go:76` (`fm.Category = strings.ToLower(strings.TrimSpace(s))`) | ✅ PASS |
| **P1-3** — `summary`/`abstract` do frontmatter **ou** o 1º parágrafo sem formatação até 160 caracteres | o `abstract` gravado é o summary, ou o fallback truncado | `internal/parser/parser_test.go:286` — `TestExtractMicroAbstract`; `:315` — `TestExtractMicroAbstract_Fallback`; o indexador consome em `internal/watcher/indexer.go:82` | ⚠️ **GAP (PCL-03)** — o **parser** é testado, mas o **caminho do indexador** (usar o `summary` como `abstract` ao gravar) não tem teste: o mutante M2 sobreviveu |
| **P2-1** — `SearchFTS`/`SearchVector`/`SearchHybrid` aceitam `category` | o filtro por categoria funciona nos dois motores | `internal/db/hybrid_test.go:156` — `TestSearch_CategoryFilter`; `internal/store/rrf_test.go:190` — `TestFuseSearchResults_CategoryFilter`; implementação em `internal/db/hybrid.go:42` | ✅ PASS |
| **P2-2** — o resultado inclui `Abstract`, `Category` e a formatação L0 (só título, ID, score e micro-abstract) | em L0 o corpo bruto é omitido | `internal/store/rrf_test.go:234` — `TestFuseSearchResults_LevelL0`; `internal/db/hybrid_test.go:240` — `TestSearch_LevelL0Projection`; implementação em `internal/store/rrf.go:258` (`res.Content = ""`) | ✅ PASS |
| **P3-1** — as flags `--level l0\|l1\|l2` e `--category resource\|memory\|skill` | as flags são aceitas e viram default | `cmd/mem/search_defaults_test.go:119` — `TestSearchDefaultsFromConfig_LevelAndCategory`; `:139` — `TestSearchDefaults_DefaultLevel` | ✅ PASS |
| **P3-2** — formatação visual com **badges de categoria e nível** | a saída do L0 traz o badge de categoria | — **sem evidência**: o mutante M6 (desligar o ramo L0 da renderização) **sobreviveu** | ❌ **GAP (PCL-01)** |
| **P4-1** — o **schema JSON** da tool `memory_search` tem `detail_level` e `category` | as duas chaves estão no schema | `internal/mcp/tools.go:54` — o schema declara `detail_level`; mas **sem teste**: o mutante M5 (remover a chave do schema) **sobreviveu** | ❌ **GAP (PCL-02)** |
| **P4-2** — o handler serializa Markdown proporcional ao nível | o `detail_level` muda a saída da tool | `internal/mcp/handlers_test.go:1088` — `TestServer_ToolsCall_MemorySearch_DetailLevelL0`; `:1154` — `TestServer_ToolsCall_MemorySearch_CategoryFilter` | ✅ PASS |

**Status**: ⚠️ **8/9 ACs com evidência** · **3 lacunas** (2 ACs sem teste + 1 caminho de implementação sem teste) · 0 spec-precision gap.

---

## Discrimination Sensor

Executado no worktree isolado `D:\.minimax\scratch\subagents\verifier-pcl\wt`
(`git worktree add`, HEAD `9e8cd05`), com os **6 alvos localizados pelo
sub-agente**. Cada mutação foi revertida com `git checkout --` depois do teste.
**Nenhum `git stash`**; a árvore real permaneceu intocada (baseline e final
idênticos) e o worktree ficou limpo.

| Mutation | File:line | Description | Killed? |
| -------- | --------- | ----------- | ------- |
| M1 | `internal/parser/frontmatter.go:76` | A categoria deixa de ser extraída do frontmatter | ✅ Killed — `TestFrontmatter_CategoryAndAbstract` |
| M2 | `internal/watcher/indexer.go:82` | O indexador deixa de usar o `summary` como `abstract` | ❌ **Sobreviveu** — nenhum teste cobre o caminho; gap **PCL-03** |
| M3 | `internal/db/hybrid.go:42` | A busca deixa de filtrar por `category` | ✅ Killed — `TestSearch_CategoryFilter` |
| M4 | `internal/store/rrf.go:258` | O L0 deixa de omitir o corpo bruto | ✅ Killed — `TestFuseSearchResults_LevelL0` |
| M5 | `internal/mcp/tools.go:54` | O schema da tool perde `detail_level` | ❌ **Sobreviveu** — o teste comportamental não olha o schema; gap **PCL-02** |
| M6 | `cmd/mem/main.go:2368` | A CLI deixa de renderizar o nível L0 | ❌ **Sobreviveu** — só o *parsing* do default é testado, não a *saída*; gap **PCL-01** |

**Sensor depth**: lightweight (6 mutações, uma por AC — a cobertura exata do
conjunto de ACs).
**Resultado**: **3/6 mortas** — PASS no que cobre, **3 lacunas reais**.

### O padrão dos 3 sobreviventes

Os três são o **mesmo erro de RACIOCÍNIO**, não três bugs diferentes: o
comportamento foi testado **no ponto errado**.

| Onde o teste olha | Onde o comportamento está |
|:---|:---|
| o default da flag (`search_defaults_test.go`) | a **renderização** da saída (`cmd/mem/main.go:2368`) |
| o **comportamento** da tool (`handlers_test.go:1088`) | o **schema** declarado (`internal/mcp/tools.go:54`) |
| o **parser** (`parser_test.go:286`) | o **indexador** que grava (`internal/watcher/indexer.go:82`) |

Ou seja: a feature foi verificada nas suas **unidades**, e o que ficou sem
prova são os **encanos entre elas**. É o mesmo mecanismo da ISSUE-038
(relatório e rotina com critérios diferentes), agora na forma "teste e
comportamento em camadas diferentes".

---

## Gate Check

- **Gate completo**: `go test -count=1 ./...` (com Postgres) → **27/27 pacotes `ok`**, exit 0
- **Build**: `go build ./...` → exit 0
- **Vet**: `go vet ./...` → exit 0
- **Format**: `gofmt -l` aponta 134 arquivos no worktree local — **é o artefato de CRLF do checkout do Windows**, não formatação quebrada: o sub-agente comparou `internal/store/rrf.go` (CRLF=342 no worktree real, LF=342 no worktree do scratch) e o diff do `gofmt` é só a mudança de fim de linha. A CI roda em Linux (LF) e está verde.
- **State**: `validate_state.py progressive-context-loading` → exit **1** (o veredito é PARTIAL — o gate funcionando, não uma falha).

---

## Code Quality

| Principle | Status |
| --------- | ------ |
| Minimum code | ✅ — a feature resolve o problema no ponto certo (schema + parser + filtro + projeção) |
| Surgical changes | ✅ — 5 tasks, sem refatoração lateral |
| No scope creep | ✅ — nada além de L0/L1/L2 e `category` |
| Matches patterns | ✅ — o `category` reusa a taxonomia da ADR-025 e o `abstract` o campo já previsto |
| Spec-anchored outcome check | ✅ — 8/9 ACs com `file:line`; a 9ª está **marcada como gap**, não passada |
| Per-layer coverage expectation | ⚠️ — **3 caminhos sem teste** (PCL-01..03) |
| Every test maps to a spec AC | ✅ — nenhum teste órfão nos pacotes tocados |
| Documented guidelines followed | ✅ — `AGENTS.md` (regras 1, 4, 7), a spec e o `validate.md` |

---

## Edge Cases

- [x] `category` ausente do frontmatter → default `resource`
      (`cmd/mem/main.go:1896`) — coberto pelo gate.
- [x] Documento sem texto → `abstract` vazio, sem quebrar a indexação — coberto
      por `TestExtractMicroAbstract_Fallback`.
- [x] Banco antigo sem as colunas → migração sem perda (`TestSchemaMigration_AbstractAndCategory`).
- [x] Filtro `category` vazio → **não** filtra (`internal/db/hybrid.go:42`, o `? = ''`).
- [x] `category` inválida (fora de `resource|memory|skill`) → normalizada para
      `resource` (`cmd/mem/main.go:1896`).
- [ ] Saída L0 da CLI com badge de categoria → **sem asserção** (PCL-01).
- [ ] Schema da tool sem `detail_level` → **sem asserção** (PCL-02).

---

## Fix Plans (os 3 gaps viram tarefas de correção)

### PCL-01 — a renderização L0 da CLI não tem asserção

- **Root cause**: o `cmd/mem` testa o *parsing* do default (`search_defaults_test.go`), mas ninguém verifica a *saída* do L0.
- **Fix**: um teste em `cmd/mem` que roda `mem search --level l0` contra um vault de teste e afirma que a linha contém o badge de categoria, o nível e **não** contém o corpo bruto.
- **Priority**: Major

### PCL-02 — o schema JSON da tool não tem asserção

- **Root cause**: `TestServer_ToolsCall_MemorySearch_DetailLevelL0` verifica o *comportamento*; o *schema* (`internal/mcp/tools.go:54`) não é inspecionado por nenhum teste.
- **Fix**: um teste que pegue `GetTools()` e afirme que `memory_search` declara `detail_level` e `category` no schema.
- **Priority**: Major

### PCL-03 — o indexador não tem teste para o `summary` virar `abstract`

- **Root cause**: `internal/watcher/indexer.go:82` decide entre `fm.Summary`, `fm.Abstract` e o fallback do parser — e a suíte do `watcher` não cobre essa escolha.
- **Fix**: um teste que indexe um documento com `summary` no frontmatter e afirme que o `abstract` gravado é **o summary**, não o fallback truncado.
- **Priority**: Minor

> Os 3 gaps são de **cobertura**, não de comportamento: o caminho funcional
> existe e o gate passa; o que falta é a prova de que ele continua funcionando.
> Por isso o veredito é **PARTIAL** e não FAIL — mas a feature **fica vermelha**
> até os 3 testes existirem (regra do `validate_state.py`).

---

## Requirement Traceability Update

| Requirement | Previous Status | New Status |
| ----------- | --------------- | ---------- |
| P1-1 (schema `abstract`+`category`) | Implementing | ✅ Verified |
| P1-2 (parser extrai `category`) | Implementing | ✅ Verified |
| P1-3 (`summary`/`abstract`/fallback) | Implementing | ⚠️ Parcial — o caminho do indexador sem teste (PCL-03) |
| P2-1 (filtro `category` na busca) | Implementing | ✅ Verified |
| P2-2 (projeção L0) | Implementing | ✅ Verified |
| P3-1 (flags `--level`/`--category`) | Implementing | ✅ Verified |
| P3-2 (badges na saída) | Implementing | ❌ Sem teste (PCL-01) |
| P4-1 (schema da tool) | Implementing | ❌ Sem teste (PCL-02) |
| P4-2 (handler por nível) | Implementing | ✅ Verified |

---

## Summary

**Overall**: ⚠️ **PARTIAL** — o comportamento está implementado e o gate está
verde, mas **3 caminhos não têm teste**.

**Spec-anchored check**: 8/9 ACs com evidência `file:line` · 3 lacunas
**Sensor**: 6 mutações · **3 mortas / 3 sobreviveram**
**Gate**: `go test` 27/27 ok (exit 0) · `go build` ok · `go vet` ok
**Report**: `.specs/features/progressive-context-loading/validation.md`

**O que funciona**: a taxonomia `category` e o `abstract` (L0) existem nos
**dois** motores, o filtro por categoria funciona na busca, a projeção L0 omite
o corpo bruto, e a tool MCP serializa por nível.

**O que falta**: 3 testes (PCL-01, PCL-02, PCL-03). O padrão é o mesmo nos
três: **o comportamento foi testado na unidade, e o encano entre as unidades
ficou sem prova** — o default da flag em vez da saída, o comportamento da tool
em vez do schema, o parser em vez do indexador.

**Nota de honestidade**: o sensor foi executado **no pai**, porque o Verifier
independente encerrou o turno antes dele. O gate e a análise ancorada na spec
foram do sub-agente.
