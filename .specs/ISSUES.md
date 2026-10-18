# ISSUES.md — Tracker de Achados do my-memory

**Propósito:** registro rastreável de bugs, débitos técnicos, achados de auditoria, code smells e outras pendências encontradas durante desenvolvimento, validação ou operação do projeto my-memory. Cada issue tem ID único, severidade, status e referência ao contexto onde foi achada.

**Convenção de ID:** `ISSUE-NNN` (zero-padded, sequencial). Severidades seguem escala:
- 🔴 **critical**: bloqueia funcionalidade principal ou causa perda de dados
- 🟠 **high**: degrada UX/performance significativamente
- 🟡 **medium**: problema funcional mas tem workaround
- 🟢 **low**: cosmético ou melhoria opcional

**Status:**
- 🔵 **open**: identificado, ainda não resolvido
- ✅ **resolved**: corrigido, commit referenciado
- ⏸️ **deferred**: aceito como débtio, registrado para trabalho futuro
- 🚫 **wontfix**: decisão consciente de não corrigir

---

## ISSUE-001 — Dead links residuais (8) por conceitos sem nota
- **Severidade:** 🟡 medium
- **Status:** ✅ resolved (commit `849bea3`)
- **Achado em:** sessão 2026-09-17 via `mem doctor --db .memory/memory.db` (também via MCP `memory_doctor`)
- **Contexto:** após fix parser EARS (74804c4) + fuzzy resolve de tags (b86a4ce, ae8547b), restavam 8 dead links. Mix de tags de frontmatter genéricas + categoria duplicada como tag + `#tag` em prosa do ADR-005.

**Lista original (8 dead links):**
| Tag | Origem | Causa raiz |
|---|---|---|
| `algorithms` | `[[COMO_FUNCIONA]]` | Frontmatter: tag genérica sem nota-casa |
| `internal` | `[[COMO_FUNCIONA]]` | Frontmatter: tag genérica sem nota-casa |
| `resource` | `[[COMO_FUNCIONA]]`, `[[ARCHITECTURE]]` | Bug: `category: resource` duplicado como tag |
| `howto` | `[[COMO_USAR]]` | Frontmatter: tag genérica sem nota-casa |
| `commands` | `[[CLI_GUIDE]]` | Frontmatter: tag genérica sem nota-casa |
| `flags` | `[[CLI_GUIDE]]` | Frontmatter: tag genérica sem nota-casa |
| `Tags` | `[[ADR-005]]` | `#Tags` em prosa (linha 22) parseado como tag — fix: inline code |
| `operational-guide` | `[[AGENTS]]` | Resolvido em `ae8547b` (token match) |

**Resolução (sem ADR — bug trivial de tagging):**
- Removidas tags genéricas (`algorithms`, `internal`, `howto`, `commands`, `flags`) → substituídas por tags específicas com casa (`turboquant`, `rrf`, `pagerank`) ou removidas.
- Removido `resource` da lista `tags:` (já está em `category:` nos mesmos docs — duplicação semântica).
- `Tags` no ADR-005 → wrappado em inline code (padrão Obsidian para exemplos de sintaxe) para o parser `tagRegex` ignorar.

**Validação empírica (`mem doctor --db .memory/memory.db` pós-fix):**

| Métrica | ANTES | DEPOIS | Delta |
|---|---:|---:|---:|
| Dead links | 8 | **0** | **-100%** |
| Health Score | 30/100 | **70/100** | **+40** |
| Documentos | 165 | 167 | +2 (re-index forçado) |

**Commits:**
- `849bea3` fix(tags): prune dead-link tags + wrap `#Tags` inline code (8→0 dead links)

**Referência:** commit `ae8547b` (redução anterior: 30 → 8, -73%).

---

## ISSUE-002 — `preCollectDocTitles` retornava basename enquanto DB retorna title do frontmatter (mismatch)
- **Severidade:** 🟠 high (causava fuzzy resolver 60% cego)
- **Status:** ✅ resolved (ae8547b)
- **Achado em:** debug durante resolução de ISSUE-001
- **Contexto:** `cmd/mem/main.go::preCollectDocTitles` retornava apenas basenames (`README`, `AGENTS`, etc.) enquanto `db.ListDocumentTitles` retornava title do frontmatter (slug tipo `001-uso-de-sqlite-...`). Mismatch 74 basenames vs 165 titles → fuzzy resolve perdia 60% dos candidatos.
- **Fix:** função agora extrai `frontmatter.title` (mesmo critério do DB), fallback para basename se ausente.
- **Lição cross-project:** salva em agent memory (`MEMORY.md`, regra "Fuzzy resolver: title do frontmatter vs basename").

---

## ISSUE-003 — `actions/setup-go@v5` flake com cache incompatível (Go 1.23 declarado vs 1.25 exigido)
- **Severidade:** 🟠 high (quebrou CI por 2 commits consecutivos)
- **Status:** ✅ resolved (aff45ca)
- **Achado em:** sessão 2026-09-17 ao investigar CI failure após commit 74804c4
- **Contexto:** `go.mod` declara `go 1.25.0` mas workflow `ci.yml` declarava `go-version: "1.23"`. Setup-go@v5 baixava toolchain 1.25 automaticamente mas extração tar falhava com `Cannot open: File exists` contra cache 1.23.
- **Fix:** bump `go-version: "1.23"` → `"1.25"` em todos os 4 jobs do workflow.
- **Lição cross-project:** salva em agent memory (`MEMORY.md`, regra "Windows path hardcoded em testes Go").

---

## ISSUE-004 — `mem index` (sem `--db`) não cria `memory.db` quando ausente
- **Severidade:** 🟡 medium
- **Status:** ✅ resolved (commit `ec1298b`)
- **Achado em:** debug durante validação do fuzzy resolve (sessão 2026-09-17)
- **Contexto:** ao deletar `.memory/memory.db`, `mem index` sem flag `--db` criava `memory.db` no CWD (working directory) em vez de respeitar o vault canônico `.memory/memory.db`. Causa raiz tinha duas camadas:
  1. `cmd/mem/main.go::resolveStorageAndRepo` — fallback de dbPath quando vazio caía em CWD `memory.db` se `.memory/memory.db` não existia.
  2. `internal/config/config.go::LoadCascadingConfig` — `DefaultConfig()` setava `SQLitePath = "memory.db"` como default; quando YAML local não declarava `storage.sqlite_path`, esse default "vazava" e a heurística de `.memory/` em `resolveStorageAndRepo` nunca disparava.
- **Resolução (ADR-016 Auto-Scoping, sem ADR novo — bug trivial):**
  - `resolveStorageAndRepo`: se `dbPath == ""` e `.memory/` é diretório, usa `.memory/memory.db` (criado por `db.InitDB`); fallback CWD só se `.memory/` ausente.
  - `LoadCascadingConfig`: ao final do cascade, se `SQLitePath` ficou no default CWD `memory.db` mas `<repoDir>/.memory/` existe, promove para `<repoDir>/.memory/memory.db`. Não toca em paths explícitos do usuário.
- **Validação empírica (smoke test E2E):**
  - Setup: removido `.memory/memory.db`, estado limpo.
  - `bin/mem.exe index` (sem `--db`): DB criado em `.memory/memory.db` ✅, sem vazamento para CWD ✅.
- **Commits:**
  - `ec1298b` fix(cli): mem index (sem --db) cria DB em .memory/ quando vault existe
- **Lição durável:** duas camadas de default (config global + storage resolver) precisam estar coerentes. ADR-040 (proposto em 2026-09-18) substitui essa heurística por centralização em `~/.memory/config.yaml` + SQLite opt-in via flag explícita — eliminando a necessidade desse fallback.

---

## ISSUE-005 — Drift em 70.8/100 com 111 críticos (resolvida via ISSUE-008)
- **Severidade:** 🟡 medium (sintoma — problema real é ISSUE-008)
- **Status:** ✅ resolved (commits `bc70c5f` + `0137779`)
- **Achado em:** MCP `memory_get_drift` em sessão 2026-09-17 (`C:/repository/my-memory/.memory/logs/drift-full.json`)
- **Contexto:** drift reportado em `HEAD~5..HEAD`: 5 commits, 9 arquivos, **score 70.8/100**, **111 críticos, 0 high, 0 medium, 0 low** — distribuição degenerada que sugere bug no threshold/calibração.
- **Investigação:** amostrei 8 docs (incluindo `README.md`, `docs/CLI_GUIDE.md`, `docs/REPOSITORY_BRAIN.md`, `docs/adr/031-semantic-drift-...` — o próprio ADR que define o algoritmo) — **0 mencionam** os 3 arquivos alterados (`cmd/mem/main.go`, `internal/parser/fuzzy.go`, `internal/parser/fuzzy_test.go`). Os 111 críticos são **falso positivo em massa**.
- **Resolução:** ISSUE-008 calibrou o detector (ADR-039). Pós-fix: 0 CRITICAL, 35 HIGH, score 62.3. Gate `--strict` liberou.

---

## ISSUE-008 — Detector de drift super-reporta CRITICAL por calibração do PageRank
- **Severidade:** 🟠 high (invalida métrica central do CI — `--strict` bloquearia PRs por falso positivo)
- **Status:** ✅ resolved (commits `bc70c5f` + `0137779`)
- **Achado em:** investigação de ISSUE-005 (sessão 2026-09-17)
- **Contexto:** fórmula do `internal/drift` (ADR-031):
  ```
  Score = min(100, (Δcommits × 12) + (PageRank × 250) + (ln(1+LinesChanged) × 6))
  ```
  Com `Δcommits=5, LinesChanged=183, ln(184)≈5.22`:
  ```
  Score = min(100, 60 + 250×PR + 31.3) = min(100, 91.3 + 250×PR)
  ```
  Qualquer nota com **PageRank > 0.035** satura em 100. Como a maioria das notas linkadas tem PR ≥ 0.5, **TUDO vira CRITICAL** quando há 5+ commits. Resultado: 111 críticos sem correlação semântica real.

**Resolução (ADR-039):**
- Match estrito: removido match por basename (palavras comuns tipo `main.go` inflavam). Agora exige path completo. Adicionado `MinMatchOccurrences=2` para filtrar listas genéricas que mencionam o path só 1x.
- Fórmula recalibrada: `PageRank × 30` (cap 15) em vez de × 250 (cap 30). Redução de 88% no peso. `lines × 5` (cap 25), `commits × 12` (cap 40).
- Limiares: `CRITICAL ≥ 75` (era 65), `HIGH ≥ 55` (era 45), `MEDIUM ≥ 30` (era 25).

**Validação empírica (MCP `memory_get_drift --since HEAD~5..HEAD`):**

| Métrica | ANTES | DEPOIS | Delta |
|---|---:|---:|---:|
| Score | 70.8 | 62.3 | -8.5 |
| CRITICAL | 111 | **0** | **-100%** |
| HIGH | 0 | 35 | +35 |
| Total flagged | 111 | 35 | -68% |

Gate `--strict` liberado: 0 CRITICAL permite uso em CI sem bloqueios falsos.

**Commits:**
- `bc70c5f` docs(adr): add ADR-039 drift detector calibration
- `0137779` fix(drift): calibrate PageRank weight + tighten path matching

**Lição cross-project:** detector de drift baseado em PageRank sem correlação semântica explícita é armadilha clássica. Salva em agent memory (regra "Drift detector: combine PageRank with explicit path-mention correlation").

---

## ISSUE-006 — Health Score persistentemente baixo (74/100)
- **Severidade:** 🟡 medium (métrica informativa, não bloqueia funcionalidade)
- **Status:** ✅ resolved (commit pending — ISSUE-010 antecedente já estava zerando dead links)
- **Achado em:** `mem doctor --db .memory/memory.db` (sessão 2026-09-17)
- **Contexto:** após ISSUE-010 zerar dead links (74/100), restava um cap de penalty de órfãs em 26 pontos. Análise dos 119 órfãos mostrou que **88% (105/119) são docs by-design** — ADRs, `.specs/*/tasks.md`+`validation.md`, SKILL.md de agents — que existem para leitura isolada, não como hubs do grafo. Fórmula `orphanRatio × 40` (cap 30) foi calibrada pra grafos densos; penaliza estruturalmente este repo.
- **Root cause:** a query de órfãos em `internal/db/graph.go` (SQLite) e `internal/store/doctor.go` (Postgres) contava TUDO que não tivesse edge — incluindo ADRs (32 órfãos), specs/tasks/validation (64 órfãos), skills (9 órfãos), `.memory/*` (config), `.github/pull_request_template.md`. Fórmula treats-by-design as if fosse signal.

**Resolução (path-prefix, sem ADR — calibração trivial):**
- Adicionado `path NOT LIKE` filter na query de órfãos dos DO stores (SQLite + Postgres), excluindo buckets by-design:
  - `docs\adr\` / `docs/adr/` (ADRs)
  - `.specs\` / `.specs/` (specs, tasks.md, validation.md)
  - `.agents\skills\` / `.agents/skills/` (SKILL.md)
  - `.memory\` / `.memory/` (config do vault)
  - `.github\` / `.github/` (PR templates, workflows .md)
- Paths cobrem Windows (`\`) e Unix (`/`) pra rodar em CI matrix.
- Fórmula do score (`orphanRatio × 40`, cap 30) **inalterada** — correção é só no numerador.

**Validação empírica (`mem doctor --db .memory/memory.db`):**

| Métrica | ANTES | DEPOIS | Delta |
|---|---:|---:|---:|
| Health Score | 74/100 | **98/100** | **+24** |
| Dead links | 0 | 0 | 0 |
| Orphans reportados | 119 | **10** | **-91%** |
| Self-loops | 0 | 0 | 0 |
| Desynced chunks | 0 | 0 | 0 |

**10 órfãos restantes (sinal real, não bug):**
- `README.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md` — arquivos raiz que deveriam linkar entre si
- `docs/BENCHMARKS.md`, `docs/REFERENCES.md`, `docs/REPOSITORY_BRAIN.md` — concept docs que deveriam ser linkados do `README.md`
- `.agents/rules/always-{quality-gate,test,update-docs-and-specs-on-pr}.md` — regras de agente que deveriam ser cross-referenciadas com `AGENTS.md`

Decisão consciente: deixar como signal (não adicionar `.agents/rules/` ao exclude) — esses 10 são "should-be-linked" e o doctor deve continuar apontando.

**Commits:**
- `f51c342` (antecedente): fix(indexer): ISSUE-010 — `documents.title` usa frontmatter.title
- (pendente): fix(doctor): ISSUE-006 — exclui by-design paths do orphan count (SQLite + Postgres)

**Trade-off aceito:** hardcoded path prefixes são brittle se a estrutura de pastas mudar. Mitigação: comentário ISSUE-006 em ambas as funções de query sinaliza o porquê. Migração futura pra `frontmatter.category` (opção B da ISSUE-006) continua disponível sem retrabalho — basta adicionar `OR (path NOT LIKE ...)` ou trocar o filtro pra `category != 'adr' AND category != 'spec' ...`.

**Lição durável:** métrica "ratio of bad things / total" precisa ter denominador consistente com o universo sendo medido. Quando a maioria do corpus é by-design, o ratio captura a estrutura do repo, não a saúde. Salva em agent memory (`MEMORY.md`, regra sobre "ratio metrics: filter numerator and denominator consistently").

---

## ISSUE-007 — 18 .md files em `node_modules/playwright-core/` desnecessários no repo
- **Severidade:** 🟢 low (artefatos de Playwright instalados pra validação visual)
- **Status:** ✅ resolved (workaround aplicado + convenção documentada)
- **Achado em:** debug do fuzzy resolver (sessão 2026-09-17)
- **Contexto:** `node_modules/playwright-core/lib/tools/skills/playwright-*/SKILL.md` e similares foram instalados durante a sessão post-release-v1.3.0-bugfixes para visual validation, mas o conteúdo é do próprio Playwright (não do my-memory).
- **Resolução (workaround, sem ADR — bug trivial):**
  - `.gitignore` já exclui `node_modules/` por padrão (entry raiz ou via `**/node_modules/`).
  - Convenção: futuras instalações Playwright usam `npm install --no-save` (evita persistir deps).
  - Verificação: `git ls-files | grep -i playwright-core | wc -l` → **0** (nenhum SKILL.md do Playwright commitado).

**Validação empírica:**
```bash
git ls-files | Select-String "playwright-core.*SKILL\.md"
# (no output)
```

## ISSUE-009 — Indexer gera dead links sistemicamente: tag values viram graph nodes
- **Severidade:** 🟡 medium (cosmético mas recorrente — cada novo doc com tags genéricos adiciona dead links)
- **Status:** ✅ resolved (ADR-041, commit `4c19393`)
- **Achado em:** validação E2E pós-ADR-040 (sessão 2026-09-18, `mem doctor --db .memory/memory.db` mostrou 19 dead links antes do cleanup parcial, 2 restantes após)
- **Contexto:** o parser trata QUALQUER valor de tag (frontmatter `tags: [...]` ou inline `#tag`) como nó do grafo e cria edge `tagged_as` do doc para esse nó. Se nenhum doc tem esse exato título, a edge vira dead link. ISSUE-001 limpou 8 tags específicas mas o problema é estrutural: cada novo doc com tags descritivas (ex: `architecture`, `storage`, `federation`) cria dead links potenciais.

**Root cause:**

1. `internal/parser/wikilinks.go::ExtractConnections` linha 278: para cada tag em `conn.Tags`, cria `addEdge(t, "tagged_as", ...)`.
2. `internal/parser/wikilinks.go::ExtractConnections` linha 235-244: regex `tagRegex` extrai `#tag` de body.
3. Tags viram nós implícitos; não há verificação se o nó existe como doc title antes de criar a edge.
4. `internal/parser/fuzzy.go::ResolveTagConnections` (commit `ae8547b`) faz fuzzy resolve quando o título do doc tem match, mas não cria nó novo pra tags não-casadas — só remove ou renomeia a edge.

**Estado pós ADR-040 + ISSUE-001 cleanup (2026-09-18):**

| Doc | Tag morta (sem nota-casa) | Relação |
|---|---|---|
| `ADR-040` | `federation`, `architecture`, `config`, `storage`, `central-vault`, `sqlite`, `postgres`, `fallback` | 8 dead links |
| `.specs/040-config-global-unica/spec.md` | `Spec 040: Implementação ADR-040 — Config global única + SQLite auto-scope + Postgres opt-in` | 1 dead link (self-reference — root cause a investigar) |

**Possíveis correções (decisão arquitetural pendente):**

1. **Não criar `tagged_as` edge se tag não casa com doc title** — parser filtra antes de criar edge. Tags "conceituais" (categorias) deixam de poluir o grafo.
2. **Aceitar como débtio cosmético** — tags conceituais ainda servem pra `mem search --tag X`; dead link é ruído visual mas funcional.
3. **Renomear nodes de tag pra `tag:<valor>`** — namespacing explícito (ex: `tag:architecture` vs doc title `architecture`) evita colisão semântica. Decisão arquitetural — ADR-041+.
4. **Híbrido**: tags com match → edge `tagged_as` normal; tags sem match → ainda cria edge mas marca `target_kind=tag` (cosmético, não dead link).

**Recomendação:** opção 1 (não criar edge pra tags sem casa) — mais limpa, mas pode quebrar `mem search --tag X` que depende do nó existir. Validar impacto primeiro.

**Workaround atual:** adicionar tags só quando há doc title correspondente (convenção manual).

**Referência:** ADR-040 (config centralizada), ISSUE-001 (limpeza específica anterior).


---

## ISSUE-010 — `documents.title` usa basename mas `availableTitles` usa frontmatter.title (inconsistência)

- **Severidade:** 🟠 high (causa dead links sistemicos a cada `--force`; só `mem doctor --fix` resolve como paliativo)
- **Status:** ✅ resolved (commit `<TBD>`, sessão 2026-09-18)
- **Achado em:** validação E2E do T1 da Spec 041 (`mem doctor --db .memory/memory.db` mostrou 4 dead links pós-fix do ISSUE-009, com auto-referências via `tagged_as`)
- **Contexto:** após ADR-041 remover tags órfãs, 4 dead links persistiram em `.specs/040/spec.md`, `.specs/041/spec.md`, `docs/adr/041-...md` com target = `[[Spec 040: Implementação ADR-040 — Config global única + SQLite auto-scope + Postgres opt-in]]` (= frontmatter.title longo do próprio source).

**Root cause (3 lugares com mesmo bug):**

| Lugar | Linha | O que usava como `title` |
|---|---|---|
| `cmd/mem/main.go::runIndexSQLite` | ~1457 (Postgres) + ~1617 (SQLite) | `strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))` → **basename** |
| `cmd/mem/main.go::preCollectDocTitles` | ~1369 | `fm.Title` → **frontmatter.title** (correto) |
| `internal/watcher/indexer.go::IndexSingleFileSQLite` | ~47 + ~182 | `strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))` → **basename** |

**Cadeia do bug:**

1. Parser extrai `conn.Tags = [spec, adr-040]` do frontmatter (spec.md 040).
2. Cria edge `tagged_as` com `target = "spec"`.
3. `ResolveTagConnections` faz fuzzy match contra `availableTitles` (= frontmatter.title `"Spec 040: Implementação..."`).
4. Substring match (passo 2): `strings.Contains("spec040:implementacao...", "spec")` → **TRUE**.
5. `fuzzyResolveTag` retorna o **original** disponível: `"Spec 040: Implementação ADR-040 — Config global única + SQLite auto-scope + Postgres opt-in"`.
6. Edge gravada no DB com target longo.
7. `mem doctor` consulta `documents.title` (que é basename `"spec"`) procurando pelo target longo → **não acha** → reporta dead link.

**Resolução (fix direto em código, sem ADR — bug trivial de coerência):**

Em `cmd/mem/main.go::runIndexSQLite` (Postgres + SQLite) e `internal/watcher/indexer.go::IndexSingleFileSQLite` (Postgres + SQLite):

```go
title := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
// ISSUE-010 (2026-09-18): usar frontmatter.title quando existir, alinhando
// com preCollectDocTitles e ListDocumentTitles. Caso contrário, o fuzzy
// substring match [...] grava edges tagged_as com target longo enquanto
// documents.title no DB é o basename.
if content != "" {
    if fmProbe, _ := parser.ExtractFrontmatter(content); fmProbe != nil && fmProbe.Title != "" {
        title = fmProbe.Title
    }
}
```

**Teste novo:** `internal/watcher/indexer_test.go::TestIndexSingleFileUsesFrontmatterTitle` — indexa nota com frontmatter.title longo e valida que `SELECT title FROM documents WHERE id = ?` retorna o frontmatter.title, não o basename.

**Validação empírica (sessão 2026-09-18):**

| Cenário | Health Score | Dead links | documents.title |
|---|---:|---:|---|
| Baseline com `mem doctor --fix` (T1) | 74/100 | 0 | basename (incorreto) |
| Pós-fix 1ª execução (`--force`) | 34/100 | 12 | frontmatter.title (estado intermediário) |
| Pós-fix 2ª execução (`--force`) | **74/100** | **0** | frontmatter.title (correto, idempotente) |

**Commits:**
- `<TBD>` fix(indexer): ISSUE-010 — `documents.title` usa frontmatter.title quando existir (4 lugares)

**Lição cross-project:** salva em agent memory (`MEMORY.md`, regra sobre "consistência entre parser, DB e pre-collect para fuzzy match").

---

## Métricas

| Issue | Severidade | Status | Achado em |
|---|---|---|---|
| ISSUE-001 | 🟡 medium | ✅ resolved | 2026-09-18 |
| ISSUE-002 | 🟠 high | ✅ resolved (ae8547b) | 2026-09-17 |
| ISSUE-003 | 🟠 high | ✅ resolved (aff45ca) | 2026-09-17 |
| ISSUE-004 | 🟡 medium | ✅ resolved (ec1298b) | 2026-09-18 |
| ISSUE-005 | 🟡 medium | ✅ resolved (bc70c5f + 0137779) | 2026-09-17 |
| ISSUE-006 | 🟡 medium | 🔵 open | 2026-09-17 |
| ISSUE-007 | 🟢 low | ⏸️ deferred | 2026-09-17 |
| ISSUE-008 | 🟠 high | ✅ resolved (bc70c5f + 0137779) | 2026-09-17 |
| ISSUE-009 | 🟡 medium | ✅ resolved (ADR-041, 4c19393) | 2026-09-18 |
| ISSUE-010 | 🟠 high | ✅ resolved | 2026-09-18 |
| ISSUE-011 | 🟡 medium | ✅ resolved (2026-10-17) | 2026-10-03 |
| ISSUE-014 | 🔴 high | ✅ resolved (2026-10-17) | 2026-10-17 |
| ISSUE-016 | 🔴 high | ✅ resolved (2026-10-17) | 2026-10-17 |

## ISSUE-011 — Teste flaky `TestSelector_StreamUntilCancel_ConveysContextCancel` (falha ~50%)
- **Severidade:** 🟡 medium
- **Status:** ✅ resolved (2026-10-17)
- **Achado em:** sessão 2026-10-03, quality gate pós-merge dos PRs #6 e #7
- **Local:** `internal/asr/selector_test.go:166` e `internal/asr/selector.go:157-163`

**Sintoma:** `go test -count=1 ./internal/asr/` falha de forma intermitente (3 de 6 runs observados). Passa isolado com `-run`. Não reprovou nos runners Linux do CI, mas falha ~50% das vezes no Windows.

**Causa raiz:** corrida sorteada pelo `select` do Go. O teste chama `cancel()` *antes* de `StreamUntilCancel`, e uma goroutine fecha o canal `in` ao observar `ctx.Done()`. Com os dois casos prontos ao mesmo tempo, o `select` de `selector.go:157-163` escolhe uniformemente ao acaso:

- caminho A — `<-gctx.Done()`: retorna `gctx.Err()`, teste **passa**
- caminho B — `<-in` já fechado: retorna `nil`, teste **falha**

O assert `errors.Is(err, context.Canceled)` só cobre o caminho A.

**Impacto:** o CI pode ficar vermelho de forma intermitente e não-determinística, o que treina o time a ignorar falha de teste.

**Correção aplicada (2026-10-17) — no código de produção, não no teste.** A análise original classificou os dois caminhos como igualmente legítimos ("não é bug de produção") e propôs consertar o teste. Isso foi reavaliado e é incorreto: a assinatura é `StreamUntilCancel(ctx, in)` — o contrato é "stream até o chamador cancelar **ou** `in` fechar". Quando `ctx` está cancelado, devolver `nil` afirma ao chamador que o stream **terminou com sucesso**, quando na verdade foi interrompido no meio. É a mesma classe da armadilha 1 do `armadilhas-verificadas` (silêncio onde deveria haver sinal): o cancelamento é o motivo mais específico e precisa vencer.

A correção desambigua o `select` no próprio `selector.go`:

```go
case pcm, ok := <-in:
    if !ok {
        // `in` fechado e `ctx` cancelado chegam prontos ao mesmo tempo, e o
        // `select` escolhe um caso aleatoriamente. Sem esta checagem, ~50% das
        // execuções devolvem nil.
        if err := gctx.Err(); err != nil {
            return err
        }
        return nil
    }
```

Verificação: `go test ./internal/asr/... -run TestSelector_StreamUntilCancel_ConveysContextCancel -count=30` → **30/30 passando**. Antes da correção, 4 falhas em 10.

Efeito colateral benéfico: o teste existente passou a ser um teste de regressão de verdade, sem precisar ser reescrito. As duas opções originais (sinalizar readiness no teste, ou aceitar os dois desfechos) teriam escondido a ambiguidade em vez de removê-la.

**Observação registrada:** o drain `select { case <-out: default: }` nas linhas 188-194 do teste não verifica nada e pode ser removido em uma passada de limpeza.

## ISSUE-014 — CI vermelho por 4 dias sem rodar nenhum teste
- **Severidade:** 🔴 high
- **Status:** ✅ resolved (2026-10-17)
- **Achado em:** sessão 2026-10-17, ao auditar por que o `make([]float32, 768)` quebrado no teste de integração do Postgres sobreviveu à migração 768 → 384

**Sintoma:** os dois últimos runs do `CI Pipeline` em `develop` terminaram em `failure` (2026-10-13 15:27 e 19:33). O último verde foi 2026-10-13 11:22 — **antes** do commit `f38477c` (schema 384d).

**Causa raiz:** todos os jobs usavam `actions/setup-go` com `go-version: "1.25"`, enquanto o `go.mod` declara `go 1.26.1`. O setup-go instalava 1.25, o Go detectava a diretiva mais nova e baixava o toolchain como módulo (`golang.org/toolchain@v0.0.1-go1.26.1.linux-amd64`), que entrava no cache junto das dependências. O restore desse cache no runner falhava com:

```
! Failed to restore: "/usr/bin/tar" failed with error: exit code 2
/usr/bin/tar: .../go1.26.1.linux-amd64/src/go/scanner/errors.go: Cannot open: File exists
```

A falha acontecia no job `lint`, **antes de qualquer teste rodar**. Como `unit-tests` e `database-tests` têm `needs: lint`, nenhum deles executou. O `768` quebrado do `internal/store/postgres_test.go` só apareceu dias depois, quando a suíte foi rodada localmente contra o Neon.

**Resolução:** os 6 jobs com Go passaram a usar `go-version-file: "go.mod"`, fazendo o setup-go instalar exatamente a versão que o módulo pede — assim o toolchain nunca é baixado como módulo e não envenena o cache. Como a versão do Go entra na chave do cache, a mudança também invalida o cache corrompido. YAML validado; os 7 jobs foram preservados.

**Lição registrada:** em `.agents/rules/always-test.md`, novos itens 8 ("regra que o CI não exercita é decorativa") e 9 ("teste flaky é teste quebrado"). Verificar `gh run list` ao encerrar tarefa, em vez de confiar na memória de que "rodou antes".

## ISSUE-012 — Testes de `cmd/mem` escreviam no config global real do usuário
- **Severidade:** 🟠 high
- **Status:** ✅ resolved (commit `fix(ci/compiler)` — ver seção "Resolução")
- **Achado em:** sessão 2026-10-03, ao auditar `~/.memory/config.yaml` para a arquitetura central-vault/Postgres

**Sintoma:** `~/.memory/config.yaml` acumulou **~160 entradas** de `repositories:` apontando para `C:\Users\Felipe\AppData\Local\Temp\TestRunInit*` — diretórios temporários de teste que já não existem.

**Causa raiz:** `cmd/mem/init_test.go` tem dois testes (`TestRunInit`, `TestRunInit_IDEs`) que usam `t.TempDir()` mas **não definem `MY_MEMORY_GLOBAL_CONFIG_DIR`** antes de chamar `runInit()`, que por dentro chama `config.RegisterRepositoryInGlobalConfig()`. Sem a variável de ambiente, o registry escreve no `~/.memory/config.yaml` real. Cada `go test ./cmd/mem/...` local acrescentava **2 entradas**, acumulando ao longo de meses.

Note que os testes de `internal/config` já faziam certo (`t.Setenv(config.GlobalConfigDirEnv, tmpDir)`) — o vazamento vinha só do pacote `cmd/mem`.

**Impacto:** o catálogo global do usuário fica inutilizável (centenas de entradas mortas), e qualquer agente ou ferramenta que leia o config recebe lixo que nunca limpa sozinho.

**Resolução:** criado `cmd/mem/main_test.go` com `TestMain` que define `MY_MEMORY_GLOBAL_CONFIG_DIR` para um temp dir válido a vida inteira do pacote de testes. Isolar uma vez no pacote é mais robusto do que lembrar de `t.Setenv` em cada teste novo. A configuração global do usuário foi limpa: de ~160 entradas para 1 (o repositório real).

**Prevenção:** qualquer teste novo em `cmd/mem` que grave config global herda o isolamento automaticamente.

## ISSUE-013 — BOM do Windows gerava frontmatter duplicado e descartava flags da CLI
- **Severidade:** 🟠 high
- **Status:** ✅ resolved
- **Achado em:** sessão 2026-10-07, teste real do fluxo "IA da IDE → markdown → vault central"

**Sintoma 1 — frontmatter duplicado.** `mem note create` alimentado por pipe do PowerShell produzia arquivo com **dois blocos de frontmatter** e caracteres BOM no meio do arquivo. O próprio docstring de `WriteAtomicNote` promete "UTF-8 sem BOM" (AGENTS.md regra 1).

**Sintoma 2 — flags silenciosamente descartadas.** Passando `--title "X" --tags "a,b" --type summary` para conteúdo que já tinha frontmatter, o comando reportava sucesso mas gravava o frontmatter **original intacto**: título, tags e type passados desapareciam sem aviso nem erro.

**Causa raiz:** em `internal/compiler/note.go`, a detecção de frontmatter existente era `strings.HasPrefix(cleanContent, "---")` após `strings.TrimSpace(content)`. Mas **`strings.TrimSpace` não remove U+FEFF** — BOM é categoria Unicode `Cf`, não `White_Space`, então `unicode.IsSpace(U+FEFF) == false`. O PowerShell 5.1 injeta BOM em todo pipe, a detecção falhava, caía no `else` e montava um segundo frontmatter por cima. Além disso o comentário dizia "preservamos ou mesclamos" mas a implementação só preservava — nunca mesclava.

O mesmo bug existia em `internal/parser/frontmatter.go` (`ExtractFrontmatter`), com efeito mais grave: **qualquer nota gravada pelo PowerShell entrava no índice sem título, sem tags e sem categoria**.

**Resolução:**
- `stripBOM()` remove U+FEFF antes de qualquer trim em `note.go`.
- `splitFrontmatter()` localiza o bloco de abertura e devolve (yaml, corpo).
- `mergeFrontmatter()` aplica os argumentos explícitos da CLI sobre o frontmatter existente: `title`/`type` da CLI vencem, `tags`/`aliases` são **unidos** (união sem duplicatas, normalizando `#`), chaves desconhecidas (`category`, `summary`, ...) são **preservadas**, `created_at`/`updated_at` garantidos.
- `parser.ExtractFrontmatter` passou a remover BOM também.

**Evidência (teste com BOM forçado na entrada):**
```yaml
# entrada tinha BOM + frontmatter próprio; saída:
category: resource            # preservada
tags:
    - tag-do-conteudo         # união, não substituição
    - tag-da-cli              # flag da CLI preservada
title: TITULO_DA_CLI          # flag da CLI venceu
type: summary                 # flag da CLI aplicada
```

---

## ISSUE-015 — Não existia como pedir limpeza do índice sem reindexar tudo
- **Severidade:** 🔥 high
- **Status:** ✅ resolved (mecanismo entregue) — limpeza do `central-memory` legado depende de aval humano
- **Achado em:** sessão 2026-10-17, via `mem doctor --repo central-memory` no vault `G:\My Drive\central-memory`

**Sintoma.** O `mem doctor` reportava 316 documentos, 1420 chunks, 34 dead links, **218 notas órfãs** e Health Score **33/100**. As órfãs são quase todas de `catalog-skills\skills\**` — ~700 arquivos de material BRUTO indexados antes de a exclusão `catalog-skills/skills/**` existir no `.memory/config.yaml`.

**Causa raiz (o que realmente estava faltando).** A única limpeza existente era `internal/store/postgres.go` → `PruneDeletedDocuments`, acoplada a `mem index --prune`: o conjunto `activeDocIDs` vinha de `cmd/mem/main.go` dos arquivos que passaram em `cfg.ShouldIndex()`, e tudo sob `rootDir` que não estivesse nele era apagado. Ou seja, **só rodava como efeito colateral de uma indexação completa**, que reconstrói o índice e regenera embeddings. Quem só queria limpar não tinha como pedir.

**Resolução — a ferramenta é o mecanismo, não a limpeza.** Entregue a MCP tool `memory_reconcile` (`internal/mcp/reconcile_handlers.go`) sobre `store.Reconciler` (`internal/store/reconcile.go`), que compara o inventário do banco (`ListDocumentPaths`) com uma varredura de disco respeitando `cfg.ShouldIndex()`, e só então chama `PruneStaleDocuments`. As categorias de relatório são disjuntas:

| Categoria | Significado | Remove? |
|---|---|---|
| Presentes | disco + banco | não |
| Ausentes no disco | banco sem arquivo | **sim** |
| No disco, fora do banco | arquivo indexável sem linha | não |
| Fora do índice por exclusão | existe mas a config exclui | **nunca** |

`catalog-skills/**` é material bruto: estar ausente do índice é o estado **correto**, então ele é reportado como "fora do índice por exclusão" e nunca entra na lista de remoção (proteção dupla: classificação no relatório + `ProtectedPrefixes` no prune).

**Guardas de segurança implementadas e testadas** (`internal/store/reconcile_test.go`):
1. `root_dir` inexistente / não é diretório / ausente → erro, zero DELETE.
2. Config do vault ausente → erro, zero DELETE.
3. **Varredura do disco com 0 arquivos indexáveis → erro, zero DELETE.** Este é o teste mais importante: sem ele, um vault não montado vira "o banco tem 316 documentos, portanto apague 316".
4. `NotIndexed > Present` com ≥ 50 presentes → remoção suspensa até `confirm=true` (vault pequeno é tolerado).
5. Toda remoção vai para o log com `id` e `path`.
6. Dry-run não emite nenhum DELETE — verificado contra um vault real em `t.TempDir()` através do pipeline real do `database/sql` (driver registrador em `internal/store/reconcile_test.go`), não por mock da função.

**Correção de comportamento descoberta durante os testes.** A extração do helper compartilhado de normalização de path endureceu duas coisas em `PruneDeletedDocuments`: (a) comparação por id/path passou a ser case-insensitive e a tolerar `\` vs `/`; (b) o recorte por `rootDir` passou a respeitar fronteira de segmento — antes `/repo/notes` casava com `/repo/notes2` por simples `HasPrefix` de string, o que podia apagar documento de vault vizinho.

**Pendente — depende de aval humano.** A limpeza efetiva do `central-memory` legado **não** foi executada. Requer: (1) confirmar que os ~700 arquivos de `catalog-skills/skills/**` são mesmo material bruto descartável; (2) rodar `memory_reconcile` em dry-run contra o vault real; (3) só então aplicar com `dry_run=false`. Nenhuma operação foi feita no Neon nem no vault `G:\My Drive\central-memory` nesta sessão.

**Commits:** ver `git log` em `develop` a partir de 2026-10-17 (`feat(mcp):`, `test(mcp):`, `docs:`).

---

**Próxima revisão:** quando iniciar próxima sessão (sessão `mvs_b361ffb514ef434a8c0c0d9342dfede6`).
**Mantido por:** Mavis (Mavis orchestrator) + Felipe Miiller (review).
**Localização:** `.specs/ISSUES.md` (commitado no repo).

## ISSUE-016 — Testes de `cmd/mem` liam e escreviam no Postgres real do usuário (regressão da ISSUE-012)
- **Severidade:** 🔴 high
- **Status:** ✅ resolved (2026-10-17)
- **Achado em:** sessão 2026-10-17, ao investigar 12 testes de `cmd/mem` falhando
- **Mesma família da:** ISSUE-012 (teste tocando estado global do usuário) — lá foi o *config*, aqui é o *storage*

**Sintoma.** 12 testes de `cmd/mem` falhavam, e o motivo era o mesmo em todos:

```
--- FAIL: TestClustersCLI_EmptyAndDegraded
    Total de nós: 11 | Total de arestas: 30
    1  C:\Users\Felipe\AppData\Local\Temp\TestRunCompileCLI2950196861\001\syntheses\microservices.md  6  synthesis
--- FAIL: TestDoctor_EventsFlag_ReportsMetrics
    🩺 Auditando integridade do grafo no PostgreSQL [001]...
    Documentos: 11 | Chunks: 14 | Arestas: 30 | Nós: 11
```

Os nós lidos eram de diretórios temporários de teste **já deletados**, gravados no banco do usuário. A lista completa: `TestClustersCLI_EmptyAndDegraded`, `TestClustersCLI_WithSQLiteData`, `TestDoctor_EventsFlag_ReportsMetrics`, `TestDoctor_EventsFlag_WarnsWhenOldestUnackedTooOld`, `TestRunDriftCommand_JSONAndTerminal`, `TestCLI_EventsTail_PrintsJSONLines`, `TestCLI_EventsInspect_Found_PrintsEnvelope`, `TestCLI_ReplayDryRun_DoesNotApplyEffects`, `TestCLI_Trace_ReturnsAllEventsWithCorrelationID`, `TestCLI_LastSequence_PrintsMaxSequence`, `TestCLI_Stats_ReportsOldestUnacked`, `TestResolveStorageAndRepo`.

**Causa raiz.** A ISSUE-012 já tinha criado `cmd/mem/main_test.go` com um `TestMain` que isola duas camadas: `MY_MEMORY_GLOBAL_CONFIG_DIR` para um temp dir, e `os.Unsetenv` de `MY_MEMORY_PG_URL`, `MY_MEMORY_PG_PASSWORD`, `POSTGRES_URL`, `DATABASE_URL`. **A camada de env não segurava**, por três motivos encadeados:

1. **`cmd/mem/main.go:48` e `cmd/mem/main.go:1280`** chamam `config.FindAndLoadDotEnv(".")` em **todo** comando (via `resolveConfig()`). A função sobe do diretório de trabalho até a raiz do git procurando `.env`, e `internal/config/env.go:12` lista `.memory/.env` como **primeiro** candidato.
2. **`internal/config/env.go:97`** — `LoadDotEnvFile` injeta as variáveis com `os.Setenv` sempre que `os.Getenv(key) == ""`. Como o `TestMain` tinha *apagado* a variável (não definido um valor sentinela), a condição era satisfeita e a URL real voltava para o ambiente do processo.
3. O `.memory/.env` do próprio repositório contém `MY_MEMORY_PG_URL` e `DATABASE_URL` (são o `.env.example` versionado em `.memory/.env.example`, com a URL real preenchida). O arquivo achado era `C:\repository\my-memory\.memory\.env`.

Daí para o roteamento: `resolveStorageAndRepo` (`cmd/mem/main.go:1344-1349`) lê `MY_MEMORY_PG_URL` **antes** de qualquer config, então `resolvedPG` vinha preenchido e o branch Postgres (`main.go:131`) ganhava do `--db <temp>/test.db` que o teste passava. **Por isso os testes liam o Postgres do usuário e ignoravam o banco temporário que acabavam de criar.**

**Alcance do vazamento — por que metade dos testes falhava e a outra metade não.** Os 12 não são dois bugs:
- **Em processo** (`TestClustersCLI_*`, `TestRunDriftCommand_*`, `TestResolveStorageAndRepo`): a chamada a `resolveConfig()` dentro do próprio teste repovoava o env antes de `resolveStorageAndRepo` ler. Por isso `TestClustersCLI_EmptyAndDegraded` falhava **mesmo rodando sozinho** (`-run`).
- **Em subprocesso** (`TestDoctor_*`, `TestCLI_*`): `runMemDoctor`/`runMem` (`doctor_test.go:116`, `events_test.go:113`) executam `bin/mem.exe` com `cmd.Env == nil`, herdando `os.Environ()`. O `TestMain` rodava antes e tinha limpado as variáveis, mas **um teste em processo anterior já tinha repovulado o env do processo-pai** — os `mem.exe` nasciam com a URL real. Daí o vazamento em cascata, dependente de ordem.

**Impacto.** Além de ler o banco do usuário, os testes que indexam (`TestRunNoteCLI_*`, `TestRunCompileCLI`) **gravavam** nele — daí os 11 documentos e os nós apontando para `TestRunCompileCLI*`/`TestRunNoteCLI*` no banco do desenvolvedor. Um `go test` é uma escrita no vault real.

**Correção (2026-10-17).** Opt-out explícito da descoberta de `.env`, na raiz do problema:

- `internal/config/env.go` — nova constante `SkipDotEnvEnv = "MY_MEMORY_SKIP_DOTENV"` e `DotEnvDisabled()`; `FindAndLoadDotEnv` retorna `os.ErrNotExist` sem tocar no disco nem no ambiente quando ligada.
- `cmd/mem/main_test.go` — `TestMain` liga `MY_MEMORY_SKIP_DOTENV=1` (terceira camada de isolamento) e restaura o valor original ao final.

A variável resolve os dois casos de uma vez porque é **herdada pelos `mem.exe`**: os subprocessos nascem com o opt-out ligado, então nunca encontram `.env` subindo do temp dir nem no ambiente.

**Por que não bastou "definir um valor sentinela" nas variáveis de storage.** `LoadDotEnvFile` só escreve quando `os.Getenv(key) == ""`. Um sentinela não-vazio impediria a reinjeção, mas `resolveStorageAndRepo` o leria como uma URL e tentaria `store.NewPostgresStore(sentinela)`, falhando o teste com um erro de conexão em vez de rotear para SQLite. Isso exigiria um segundo entalhe no código de produção com um valor mágico — o opt-out é explícito e não depende de valor mágico.

**Testes de regressão.**
- `cmd/mem/storage_isolation_test.go:19` — `TestDotEnvDoesNotReinjectStorageEnvVar`: reproduz o estado pós-`TestMain` e afirma que `FindAndLoadDotEnv(".")` não repopula nenhuma das 4 variáveis de storage.
- `cmd/mem/storage_isolation_test.go:42` — `TestResolveConfigAndStorage_NeverReachUserGlobalState`: roda a cadeia real do CLI (`resolveConfig()` → `resolveStorageAndRepo`) e falha se o storage resolver para Postgres ou para dentro de `os.UserHomeDir()`.
- `internal/config/env_test.go:90` — `TestFindAndLoadDotEnv_SkipDotEnvEnv` e `TestDotEnvDisabled` cobrem o contrato do opt-out no pacote que o implementa.

Os dois primeiros **falhavam antes da correção** (red confirmado: `MY_MEMORY_PG_URL foi repopulada por C:\repository\my-memory\.memory\.env (86 bytes)`).

**Efeito colateral perigoso registrado.** Mover o `.env` da raiz para `.memory/.env` **piorou** o problema: `CandidateEnvFileNames[0]` é `.memory/.env`, então o arquivo agora é encontrado em **uma** aplicação de `filepath.Join` em vez de duas. A hipótese inicial da investigação (de que mover o arquivo resolveria) estava errada — por isso o teste de regressão importa mais que a mudança de arquivo.

**Pendência fora do escopo deste fix (decisão do humano).** O banco real do usuário continua com o lixo acumulado — 11 documentos e 30 arestas de diretórios temporários de teste no Postgres, e `C:\Users\Felipe\.memory\memory.db` com 15.3 MB (última escrita 2026-10-13). **Nada foi apagado**: `~/.memory/memory.db` e `~/.memory/config.yaml` são estado real do usuário. A limpeza do Postgres (via `tmpdbtool -action=reset`, ferramenta de manutenção do dono do repo) e do SQLite central é decisão do Felipe.

**Commits:** ver `git log` em `develop` a partir de 2026-10-17 (`fix(config):`, `test(ci):`, `docs:`).

---
