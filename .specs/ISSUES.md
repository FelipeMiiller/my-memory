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
- **Status:** ✅ resolved (moot — a dependência que o causava não existe mais)
- **Achado em:** debug do fuzzy resolver (sessão 2026-09-17)
- **Contexto:** `node_modules/playwright-core/lib/tools/skills/playwright-*/SKILL.md` e similares foram instalados durante a sessão post-release-v1.3.0-bugfixes para visual validation, mas o conteúdo é do próprio Playwright (não do my-memory).
- **Encerramento (2026-10-18) — moot, sem ADR:** o Playwright era dependência do **viewer Electron**, e o viewer inteiro foi removido do repositório no commit `b477200` (`package.json`, `forge.config.ts`, configs vite/playwright e os testes e2e saíram junto — 112 arquivos, 27.858 linhas). Não existe mais `node_modules/playwright-core/` neste repositório, então os 18 arquivos `.md` que motivaram a issue não podem reaparecer. A questão está resolvida por remoção do vetor, não por convenção: o `.gitignore` continua excluindo `node_modules/`, e a convenção `npm install --no-save` continua válida para qualquer tooling futuro, mas deixa de ser load-bearing aqui.
- **Histórico da resolução original (workaround):**
  - `.gitignore` já exclui `node_modules/` por padrão (entry raiz ou via `**/node_modules/`).
  - Convenção: futuras instalações Playwright usam `npm install --no-save` (evita persistir deps).
  - Verificação: `git ls-files | grep -i playwright-core | wc -l` → **0** (nenhum SKILL.md do Playwright commitado).

**Validação empírica (2026-10-18, após a reversão do viewer):**
```bash
git ls-files | Select-String "package.json|playwright"
# (no output — nem package.json nem config do Playwright no repositório)
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
| ISSUE-007 | 🟢 low | ✅ resolved (moot — dependência removida) | 2026-09-17 |
| ISSUE-008 | 🟠 high | ✅ resolved (bc70c5f + 0137779) | 2026-09-17 |
| ISSUE-009 | 🟡 medium | ✅ resolved (ADR-041, 4c19393) | 2026-09-18 |
| ISSUE-010 | 🟠 high | ✅ resolved | 2026-09-18 |
| ISSUE-011 | 🟡 medium | ✅ resolved (2026-10-17) | 2026-10-03 |
| ISSUE-012 | 🟠 high | ✅ resolved (fix(ci/compiler)) | 2026-10-03 |
| ISSUE-013 | 🟠 high | ✅ resolved | 2026-10-07 |
| ISSUE-014 | 🔴 high | ✅ resolved (2026-10-17) | 2026-10-17 |
| ISSUE-015 | 🔥 high | ✅ resolved (mecanismo entregue) | 2026-10-17 |
| ISSUE-016 | 🔴 high | ✅ resolved (2026-10-17) | 2026-10-17 |
| ISSUE-017 | 🟡 medium | ✅ resolved (mecanismo de prevenção entregue) | 2026-10-18 |

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

## ISSUE-017 — 158 termos sem acento no `central-memory`: a limpeza foi manual, a prevenção não existia
- **Severidade:** 🟡 medium
- **Status:** ✅ resolved (mecanismo de prevenção entregue; a limpeza do vault foi feita em paralelo)
- **Achado em:** sessão 2026-10-18, auditoria do vault `G:\My Drive\central-memory`
- **Família:** qualidade de conteúdo — não é regressão de código anterior

**Sintoma.** 158 termos sem acento espalhados por 14 pastas (`performance` concentrava 94 deles). O vault é gerado por agentes de IA, que escrevem "sao", "tres", "configuracao" em prosa normal.

**Por que não é cosmético.** "sao" e "são" tokenizam diferente no FTS5 e geram embeddings diferentes. Quem busca o termo escrito corretamente não acha o termo que está no documento — e quem busca o que está no documento vê a busca quebrada. O defeito é de recuperabilidade, não de estética.

**Resolução, em duas partes:**

1. **Limpeza** — os 158 termos foram corrigidos no vault, em paralelo, por outro agente.
2. **Prevenção (esta issue)** — sem detector, o próximo lote de notas geradas reintroduz o problema em silêncio. Ninguém perceberia, porque texto sem acento parece texto normal num dashboard. O `mem doctor` ganhou a seção `Ortografia`.

**Mecanismo (`internal/store/orthography.go`, novo).**

- `FindUnaccentedTerms(content string) []TermFix` — função pura, sem banco e sem I/O.
- `portugueseAccentFixes map[string]string` — **97 pares curados à mão**, com palavra completa como chave. Nunca derivado por regex.
- `OrthographyAccumulator` — agrega por documento sem manter o texto indexado em memória.
- Limites de palavra **Unicode** (`\pL`, `\pN`), e não o `\b` do RE2: o `\b` do RE2 é ASCII, então trata "é" como separador e faria "três" casar com "tres".

**Ignora de propósito:** blocos de código (```` ``` ````), linhas indentadas com 4 espaços, code spans, URLs, `[[wikilinks]]` e chaves de frontmatter — exceto os **valores** de `title` e `summary`, que são texto humano e podem ter o mesmo defeito.

**Reporta como aviso, nunca como erro.** A seção entra no relatório do `mem doctor` como aviso e **não** altera o Health Score: perder acento degrada a busca, mas não corrompe dado. Mexer na métrica é decisão de produto, não efeito colateral de um linter de texto.

**Testes de regressão** (`internal/store/orthography_test.go`, 18 funções; o CI roda `./internal/store/...` em `ci.yml:85`, então o gate é exercitado e não decorativo).

Os negativos importam mais que os positivos — um detector que "corrige" palavra válida é pior do que não existir:

| Caso | Teste | O que prova |
|---|---|---|
| `esta` `sim` `limite` `media` `ate` `ha` em prosa | `IgnoresCorrectUnaccentedPortuguese` | palavra portuguesa válida não é sinalizada |
| `NÃO` `ENTÃO` `três` `código` | `IgnoresAlreadyAccentedPortuguese` | texto já correto não vira ruído |
| `cache` `index` `target` `build` `runtime` `commit` `slug` `timeout` | `IgnoresEnglishTechnicalTerms` | inglês técnico não entrou na lista |
| `indices` `notas` | `IgnoresSubstrings` | o casamento é por palavra inteira |
| `indicacoes` `Tresnao` `aposicao` `IndiceBuild` | `IgnoresTermsInsideLongerWords` | termo colado a letra não dispara |
| ``` ```go ... ``` ```, 4 espaços, tab, `` `sao` `` | `IgnoresCodeFences`, `IgnoresIndentedCodeAndCodeSpans` | código não é prosa |
| `[[configuracao]]`, URL, `[rótulo](url)` | `IgnoresWikilinksAndURLs` | identificador e endereço não são português |
| `title:`/`summary:` contra `tags:`/`id:`/`aliases:` | `FrontmatterIgnoraChavesMasLeTitleESummary` | só o texto humano do frontmatter entra |
| `"title": "..."` em JSON | `FrontmatterJSON` | o valor JSON é capturado entre aspas — senão as tags da mesma linha entram |
| `FindUnaccentedTerms("")` | `EmptyInput` | entrada vazia não entra em panic |
| 20 termos da lista curada em prosa | `DetectsCuratedList` | a lista inteira é exercitada termo a termo |

**Decisões registradas, para não virarem surpresa depois:**

- `indices` ficou **fora** do mapa de propósito: é plural de "índice" e aparece como identificador técnico. O resto da família (`variaveis`, `funcoes`, `versoes`...) está no mapa.
- `PortugueseOrthographyIssue` ganhou um campo `Count` além dos três propostos: sem ele não há como dizer quais são os "N piores casos", que é o que a seção promete.
- `so`, `sera`, `era`, `tera`, `fara` ficaram fora: têm leitura ambígua em português (o substantivo "sera", o futuro "será") e sinalizá-los exigiria análise de contexto, que não existe aqui.

**Fora do escopo desta issue (registrado para não parecer esquecido):**

- A ferramenta MCP `memory_doctor` **não** expõe a seção. `internal/mcp/handlers.go` tem um `DoctorReport` espelho, e alterá-lo é um change próprio.
- O Health Score não foi tocado — por decisão, não por esquecimento.

**Commits:** ver `git log` em `develop` a partir de 2026-10-18 (`feat(store):`, `docs:`).

---

## ISSUE-018 — `mem up` sobe um daemon que não faz nada

- **Severidade:** 🟠 high
- **Status:** 🔴 open (decisão do owner: manter o daemon e registrar a lacuna)
- **Achado em:** sessão 2026-10-18, batch 4 da spec `.specs/features/repo-realignment/`
- **Família:** código inalcançado — fundações prontas, wiring ausente

**Sintoma.** `mem up` responde `ready`, o processo fica vivo, e nada acontece: nenhum worker é iniciado, nenhum evento é emitido, nenhum health check é publicado. Para quem usa, o supervisor parece quebrado; para quem lê o código, parece um daemon ocioso.

**Evidência (`file:line`).**

| Fato | Onde |
|---|---|
| `mem up` gera o binário `mymemoryd` e espera a linha de `ready` | `cmd/mem/up.go:81`, `cmd/mem/up.go:117` |
| O entrypoint monta o Manager e chama `Run` | `cmd/mymemoryd/main.go:92`, `cmd/mymemoryd/main.go:102` |
| `NewManager` recebe `nil` no parâmetro `*sql.DB` | `cmd/mymemoryd/main.go:92` |
| `Run` só bloqueia em `ctx.Done()` — não inicia nenhum worker | `internal/supervisor/manager.go:162`–`166` |
| Com `db == nil`, `emitEvent` retorna cedo: emissão é no-op permanente | `internal/supervisor/lifecycle.go:472`–`473` |

**O que existe e está morto.** A camada de lifecycle inteira está implementada e coberta: `Start` (`internal/supervisor/lifecycle.go:61`), `Stop` (`internal/supervisor/lifecycle.go:125`), `Health` (`internal/supervisor/lifecycle.go:145`), `watchWorker` (`internal/supervisor/lifecycle.go:173`), `restartWorker` (`internal/supervisor/lifecycle.go:236`) e `StartAll` (`internal/supervisor/manager.go:234`). São **2.033 linhas de teste em 9 arquivos** (`internal/supervisor/*_test.go`) exercitando essa camada — verdes, mas sem nenhum caller de produção.

**A correção é wiring, não fundação.** O que falta é um `Run` que carregue o manifesto de workers e chame `StartAll`, mais um `*sql.DB` real no lugar do `nil` de `cmd/mymemoryd/main.go:92`. Nenhuma assinatura pública precisa mudar, e nada da camada de lifecycle precisa ser reescrito — por isso a decisão foi manter em vez de remover.

---

## ISSUE-019 — `mem doctor` dá Health Score 100/100 num grafo vazio

- **Severidade:** 🟡 medium
- **Status:** ✅ **resolved** (2026-10-28) — o **Defeito 1** (grafo vazio pontua perfeito) foi **corrigido** pelo fluxo spec-driven; o **Defeito 2** (o vault deste repositório não está indexado) é **operação**, não código, e segue **pendente com o dono**
- **Achado em:** sessão 2026-10-18 — tool MCP `memory_doctor` no vault `FelipeMiiller/my-memory`
- **Relacionada:** `ISSUE-006` (Health Score persistentemente baixo), já aberta. Esta issue **não** a substitui e não a duplica — cross-referência. · ADR-013 (`docs/adr/013-higiene-de-grafo-pruning-e-doctor.md`) · spec `.specs/features/doctor-empty-vault-score/`

**Evidência (saída do próprio MCP do projeto).** `memory_doctor` com `repository: FelipeMiiller/my-memory` retornou:

- **Health Score: 100/100**
- 0 documents · 0 chunks · 0 edges · 0 nodes
- banner de staleness: **8.094 arquivos modificados/novos**

**Defeito 1 — grafo vazio pontua perfeito.** Em `internal/store/doctor.go:112` o score começa em `100`, e todas as penalidades (dead links, órfãs, self-loops, chunks dessincronizados) estão dentro do `if report.TotalDocuments > 0`, de `internal/store/doctor.go:113` a `internal/store/doctor.go:142`. Um vault nunca indexado sai com a nota máxima: a métrica deixa de distinguir "vault saudável" de "vault inexistente" — que é justamente o uso principal dela.

**Defeito 2 — o vault deste repositório está efetivamente não indexado.** Os 8.094 arquivos modificados/novos do banner staleness, contra 0 documentos no grafo, são o mesmo fato visto de dois ângulos: a memória do próprio repositório não existe no banco. `mem index` no vault local resolve este; o Defeito 1 é a correção de código.

**Por que não duplica `ISSUE-006`.** `ISSUE-006` trata o Health Score *persistentemente baixo* num vault indexado — é calibração de pesos. O caso daqui é o oposto e anterior na cadeia: score alto com grafo vazio, por uma guarda que pula o cálculo inteiro. As duas convivem; corrigir uma não arruma a outra.

### Correção (2026-10-28) — fluxo spec-driven

Spec: `.specs/features/doctor-empty-vault-score/` (`spec.md`, `tasks.md`, `validation.md`).

**Causa-raiz completa.** A guarda `if report.TotalDocuments > 0` não estava só no PostgreSQL: a **matemática de penalidades inteira** estava duplicada nos dois motores — `internal/db/graph.go` (SQLite) e `internal/store/doctor.go` (PostgreSQL) — e em ambos o score começava em `100` e o cálculo era pulado quando não havia documentos. A duplicação foi o que escondeu o defeito.

**O que mudou:**

| Item | Antes | Agora |
| :--- | :--- | :--- |
| Regra de aplicabilidade | implícita, duplicada em 2 lugares | **uma** regra: `store.DoctorReport.HealthScoreApplicable()` (`TotalDocuments > 0`) |
| Matemática das penalidades | copiada em 2 lugares | **uma** função: `store.CalculateHealthScore(report)`, usada pelos dois motores |
| Relatório (CLI) | `🟢 … (Health Score: 100/100)` num vault vazio | `⚪ … (Health Score: não se aplica)` + o aviso e a ação (`mem index`) |
| Relatório (MCP, Markdown) | `Health Score: 100/100` | `Health Score: não se aplica` + "vault sem documentos indexados" |
| `structuredContent` | `health_score` sempre presente | `health_score` **anulável** — quando não se aplica, a chave **não viaja** (ausência ≠ 0 ≠ 100) |
| Viewer | — | aceita ausência **e** `null`; o tile mostra "—" e "vault sem documentos" |
| Caminho com documentos | 100 no relatório perfeito | **inalterado**: 100 perfeito, 90 com 2 dead links, 87 com órfã + self-loop; o JSON dos vaults com documentos continua **byte-idêntico** |

**Prova contra o defeito.** O defeito foi reintroduzido em três pontos (o cálculo que voltava a 100, a conversão que voltava a emitir o número, a impressão que voltava a mostrar a nota) e **três camadas de teste falharam com o sintoma exato** — `TestDoctorHealthScore_VazioNaoSeAplica` ("obteve 100"), `TestDoctor_EmptyVault_DoesNotReportPerfectScore` (o binário real imprimindo `🟢 … 100/100` com 0 documentos) e `TestHealthScoreForMCP_VazioNaoEmiteScore`. Restaurado, os três passam.

**Gate:** `go build`/`go vet`/`go test -count=1 ./...` exit 0 (26 pacotes `ok`); `gofmt -l` vazio; front-end — typecheck 0, **98/98** testes, build 0, **144.813/163.840 bytes gzip**; `validate_spec.py` / `validate_tasks.py` / `validate_state.py` sem erro.

**Contrato público.** A ADR-013 continua válida: o Health Score de 0 a 100 com penalidades graduadas **não mudou**. O que mudou é a **precondição de aplicabilidade**, registrada como nota de vigência em `docs/adr/README.md`. O campo do payload viaja **ausente, e não `null`**, porque o contrato do wire do `structuredContent` proíbe `null` (`TestStructuredContentNeverEmitsNullArrays`).

**O que fica pendente (não é código).** O **Defeito 2** — o vault deste repositório está efetivamente não indexado (0 documentos no grafo) — é **operação**: resolver com `mem index` no vault, decisão do dono. A correção de código acima não muda esse fato; ela só faz o relatório dizer a verdade enquanto ele durar.

---

## ISSUE-020 — ADR-058 (vis-network vendorizado) está Accepted, mas o asset e o `go:embed` não existem

- **Severidade:** 🟠 high
- **Status:** 🔵 open
- **Achado em:** sessão 2026-10-19 — auditoria da rodada da viewer (ADR-059/ADR-060), ao verificar o estado real do motor de grafo antes de planejar a etapa seguinte
- **Relacionada:** ADR-058 (`docs/adr/058-renderizador-vis-network-vendorizado-sem-build-step.md`) · feature `graph-view-vis-engine`

**O que a ADR-058 decide.** Renderizar o grafo com **vis-network 10.1.2 vendorizado**, sem build step e sem CDN: o asset `internal/graphview/assets/vis-network.min.js` (652 kB, UMD standalone) embutido com `go:embed`, e `mem graph view --engine vis --out g.html` emitindo **dois arquivos** lado a lado (`g.html` + `vis-network.min.js`), extraídos dos bytes embutidos.

**O que existe hoje.** Nenhum item da decisão foi implementado:

| Item | ADR-058 exige | Estado real |
|---|---|---|
| `internal/graphview/assets/vis-network.min.js` | presente (652 kB, v10.1.2) | **ausente** — `internal/graphview/` tem só 8 arquivos, todos `.go` |
| `go:embed` do asset | embutido no binário | **0 ocorrências** em `internal/graphview/` |
| `LICENSE-vis-network.txt` | ao lado do asset | **ausente** |
| `mem graph view --engine vis` | flag que escolhe o motor | **não existe** — `cmd/mem/` não tem flag `--engine` de grafo |
| Renderizador ativo | vis-network | template HTML autocontido com JS inline (`internal/graphview/template.go:11`, `htmlPageTemplate`) — o motor legado das ADR-020/ADR-024 |

**Por que isso é um defeito, e não só trabalho faltando.** O índice de ADRs (`docs/adr/README.md:61`) registra a ADR-058 como **Accepted**, que no vocabulário do repositório significa decisão em vigor. Quem procurar o motor vis-network no código não encontra, e a ADR-058 sugere que a restrição "sem CDN e sem build step" já está satisfeita no grafo servido pelo MCP — onde, na verdade, ainda é o HTML legado. A feature `graph-view-vis-engine` parou na fase **Specify**: tem `spec.md` e `design.md`, mas **nenhum `tasks.md`**, ou seja, nunca passou pela fase Tasks nem foi executada.

**Impacto.** Nenhum em runtime — o renderizador legado continua funcionando e o gate segue verde. Alto em **rastreabilidade**: o repositório afirma uma decisão que não foi implementada, e a feature correspondente não tem plano de execução.

**Decisão pendente (produto, não documentação).** O escopo do `tasks.md` de `graph-view-vis-engine` é decisão do operador: (a) executar a feature como especificada — vendorizar os 652 kB, embutir, expor `--engine vis`; (b) rebaixar a ADR-058 para "Accepted, não implementada" e manter o motor legado como único; (c) superseder a ADR-058 se o motor vis-network for superado pela viewer React/Vite da ADR-059, que já desenha o grafo no navegador.

**Reprodução:** `grep -r "go:embed" internal/graphview/` → 0 ocorrências; `internal/graphview/` → 8 arquivos `.go`, nenhum subdiretório.

---

## ISSUE-021 — O fallback SPA responde 200 com `index.html` para um asset faltante quando o navegador pede `Accept: text/html`

- **Severidade:** 🟡 medium
- **Status:** 🔵 open
- **Achado em:** sessão 2026-10-19 — verificação adversarial da feature `mcp-static-serving` (verificador independente)
- **Relacionada:** ADR-060 D-5 (a regra do fallback)

**O que a ADR-060 registra.** O fallback SPA só se aplica quando a requisição declara `Accept: text/html`, e a justificativa escrita é que assim *"um asset faltando (`/assets/nao-existe.js`) continue sendo 404 honesto, e não HTML com 200"*.

**O que acontece de verdade.**

| Requisição | Resultado |
|---|---|
| `GET /assets/nao-existe.js` **sem** `Accept: text/html` | `404` + corpo `not found` — o AC é cumprido |
| `GET /assets/nao-existe.js` **com** `Accept: text/html` | **`200` + `index.html`** (480 B, `Content-Type: text/html`) |

O `Accept: text/html` é exatamente o que um **navegador** envia ao pedir um `.js`. Ou seja: a razão documentada não se sustenta para o cliente que importa. Um `404` de asset continua sendo `404` para quem não pede HTML — mas quem pede (o navegador) recebe o entrypoint com status de sucesso.

**Impacto real:** baixo. `X-Content-Type-Options: nosniff` impede o navegador de executar o HTML como script, e o conteúdo servido é o entrypoint público da própria SPA. O custo real é **ruído**: cache e log passam a registrar 200 onde houve 404, e um asset sumido fica invisível para quem monitora status.

**A correção padrão, que é uma mudança de contrato.** A regra usada pela maioria dos servidores SPA é: *só cai no fallback quando o path **não** tem extensão de arquivo* — rota (`/clusters`) cai, asset (`/assets/x.js`) não. Isso tornaria a justificativa da ADR-060 verdadeira. Mas muda o comportamento documentado, então **não foi feito sem decisão**.

**Achado conexo (B8).** `GET /graph/../x` normaliza para fora do prefixo `/graph/` e **cai no fallback** (200 + entrypoint). Benigno — só entrega o entrypoint público, nunca um artefato de fora da raiz —, mas "o prefixo `/graph/` nunca cai no fallback" não é literal: a checagem do prefixo roda **depois** da normalização. `GET /graph/qualquer.html` devolve 404 corretamente.

**Decisão pendente (produto).** (a) manter a regra como está e corrigir a **justificativa** da ADR-060 no índice; (b) adotar a regra da extensão e emendar a ADR-060; (c) deixar como está e aceitar o ruído de log/cache.

---

## ISSUE-022 — `memory_get_impact` e `memory_find_path` não emitem `structuredContent`: dois schemas da viewer nunca validam nada

- **Severidade:** 🟡 medium
- **Status:** 🔵 open
- **Achado em:** sessão 2026-10-19 — verificação adversarial da feature `viewer-dashboard` (verificador independente)
- **Relacionada:** ADR-060 (as 10 tools com `structuredContent`)

**O fato.** Das 24 tools MCP, 10 emitem `structuredContent` (ADR-060). `memory_get_impact` e `memory_find_path` **não** estão entre elas. Mas a viewer tem `impactSchema` (`viewer/src/lib/schemas.ts:454`) e `pathSchema` (`:511`), usados em `viewer/src/lib/queries.ts:192,203` — e como as duas tools nunca emitem o payload estruturado, **esses dois schemas nunca validam nada**: a tela renderiza sempre o texto.

**Por que importa.** A rota `/drift` da viewer promete blast radius e caminho entre dois nós. As duas informações chegam — mas como markdown, sem a estrutura que a tela foi desenhada para consumir. É a mesma classe de degradação que o defeito `null` do drift (corrigido nesta rodada), pelo caminho oposto: lá o payload existia e o schema rejeitava; aqui o schema existe e o payload não.

**Por que não foi corrigido aqui.** A lista de 10 tools com `structuredContent` é o **escopo acordado** na ADR-060 e nas duas specs. Ampliá-la para 12 muda o contrato público do servidor (duas tools a mais passariam a ter `structuredContent`) e exige revisar a forma do payload de cada uma. Isso é decisão de produto.

**Decisão pendente (produto).** (a) estender o `structuredContent` para as duas tools e atualizar a ADR-060 e as specs; (b) remover os dois schemas e a renderização estruturada correspondente da viewer, assumindo que a tela use texto; (c) deixar como está e registrar como limitação conhecida do painel `/drift`.

---

## ISSUE-023 — O servidor anuncia o protocolo MCP `2024-11-05`, mas emite `structuredContent` (campo de `2025-06-18`), e 4 das 10 tools o devolvem como array

- **Severidade:** 🟡 medium
- **Status:** 🔵 open
- **Achado em:** sessão 2026-10-19 — verificação adversarial da feature `mcp-static-serving` (verificador independente)
- **Relacionada:** ADR-060 · ADR-006 (MCP)

**Defeito 1 — a versão anunciada não cobre o campo emitido.** `internal/mcp/server.go:15` declara `ProtocolVersion = "2024-11-05"`, mas `structuredContent` foi introduzido no protocolo MCP **2025-06-18**. Um cliente estrito que valida a versão anunciada não espera o campo; um cliente que o espere pode rejeitar a versão.

**Defeito 2 — a forma do payload não é a que a spec do protocolo exige.** A spec MCP 2025-06-18 define `structuredContent` como **objeto**. Quatro das dez tools devolvem **array**: `memory_list_files`, `memory_note_outline`, `memory_get_hubs` (degree e pagerank) e `memory_get_insights`. Entre a SPA e o servidor isso funciona — o cliente é próprio, e os schemas da viewer aceitam array —, mas diverge da spec.

**Por que não foi corrigido aqui.** Trocar a versão anunciada é uma mudança de contrato com clientes externos (Claude Desktop e afins negociam a versão no `initialize`). Envolver os arrays em um objeto quebra a forma que a viewer já consome e que as specs descrevem. As duas mudanças pedem uma **ADR própria** — é a terceira mudança de protocolo do transporte nesta rodada, e uma ADR é o registro que o repositório usa para esse tipo de decisão.

**Decisão pendente (produto).** (a) emendar a ADR-060: subir o `ProtocolVersion` para `2025-06-18` e passar a embrulhar os arrays (`{"items": [...]}`), com a SPA e as specs atualizadas; (b) subir só a versão, aceitando a divergência de forma; (c) manter como está e registrar a limitação.

---

## ISSUE-024 — Caracteres CJK espúrios no texto de duas ADRs (ADRs são imutáveis)

- **Severidade:** 🔵 low
- **Status:** 🔵 open
- **Achado em:** sessão 2026-10-19 — verificação adversarial (verificador independente)
- **Relacionada:** ADR-036 · ADR-047

**O fato.** Duas ADRs do repositório têm caracteres chineses/japoneses no lugar de palavras portuguesas:

| Arquivo | Linha | Texto atual | Texto pretendido |
|---|---|---|---|
| `docs/adr/036-backlog-pos-release-v1.3.0-e-roadmap-v1.4.0.md` | 233 | `por未来的 leitor` | `por futuros leitores` |
| `docs/adr/047-code-ast-tree-sitter-multi-linguagem.md` | 172 | `` `code_edges`参加的 na CTE `` | `` `code_edges` participa na CTE `` |

**Por que não foi corrigido aqui.** A regra do repositório é que **ADRs são imutáveis**: o corpo registra a decisão como ela foi tomada, e correção de texto numa ADR já aceita reescreve o histórico do registro. O mecanismo correto é superseder ou registrar a correção no índice (`docs/adr/README.md`) — que é onde a ADR-060/059 já registram status e implementações.

**O mesmo padrão apareceu nesta rodada** em `.specs/features/mcp-static-serving/spec.md:19` (`cujo uso尺尺 é fácil confundir` → `cujo uso é fácil`). Uma spec **não** é registro histórico, então essa foi corrigida direto. A lição para a próxima: specs e docs corrige-se; ADRs, só no índice.

**Decisão pendente (produto).** Registrar a correção textual no índice de ADRs, ou superseder as duas ADRs.

---

## ISSUE-025 — 12 testes de events em `cmd/mem` falharam uma vez, sob carga concorrente do mesmo store

- **Severidade:** 🟡 medium
- **Status:** 🔵 open
- **Achado em:** sessão 2026-10-19 — gate final da rodada da viewer
- **Relacionada:** ISSUE-016 (testes de `cmd/mem` liam e escreviam no Postgres real) · ISSUE-011 (teste flaky em `internal/event_runtime`)

**O fato.** Uma execução de `go test -count=1 ./...` reprovou **12 testes**, todos
do mesmo grupo — os de *events*:

```
TestDoctor_EventsFlag_ReportsMetrics
TestDoctor_EventsFlag_WarnsWhenOldestUnackedTooOld
TestDoctor_NoEventsFlag_NoEventsSection
TestCLI_EventsTail_PrintsJSONLines
TestCLI_EventsInspect_NotFound_ExitsNonZero
TestCLI_EventsInspect_Found_PrintsEnvelope
TestCLI_ReplayDryRun_DoesNotApplyEffects
TestCLI_ReplayApply_RequiresConfirmation
TestCLI_Trace_ReturnsAllEventsWithCorrelationID
TestCLI_LastSequence_PrintsMaxSequence
TestCLI_Stats_ReportsOldestUnacked
TestCLI_MissingConfig_Exits2
```

O resto da suíte passou — inclusive `internal/mcp`, que é o pacote tocado nesta
rodada.

**O contexto.** No momento da falha, havia um `mem mcp --port 38400` rodando em
background **e atendendo consultas pesadas** contra o mesmo Postgres: um
`memory_doctor` (varredura completa do banco) e um `memory_get_drift` (varredura
do git). Os 12 testes que falharam são exatamente os que tocam o *event store*.

**Tentativas de reprodução.**

| Cenário | Resultado |
|---|---|
| Suíte completa, com o servidor ainda no ar | **PASS** (exit 0) |
| `TestCLI_Stats_ReportsOldestUnacked` isolado, com o servidor no ar | **PASS** |
| Suíte completa, com o servidor encerrado | **PASS** (74,9 s) |

Ou seja: **não reproduz sob demanda**, e o gate final fechou verde. A falha
correlaciona com uso concorrente do mesmo store, não com uma mudança de código.

**Hipótese (a confirmar).** Os testes de events compartilham um recurso real
com o servidor — a mesma classe da ISSUE-016, em que os testes de `cmd/mem`
liam e escreviam no Postgres do usuário. A correção da ISSUE-016 isolaram
alguns testes, mas o grupo de *events* ficou de fora.

**Impacto.** Quem roda o servidor MCP e a suíte ao mesmo tempo pode ver um
gate vermelho sem causa no código — e o oposto, um gate verde que não exercita
nada. É a mesma armadilha que a ISSUE-014 descreveu (CI vermelho por dias sem
rodar teste nenhum), vista pelo outro lado.

**Direção da correção.** Isolar o event store nos testes de `cmd/mem`, como a
ISSUE-016 fez com o restante — temp dir ou store efêmero por teste, sem tocar
o store real.

**Nota.** A flakiness já tem precedente no repositório: a ISSUE-011 registrou um
teste flaky em `internal/event_runtime` com taxa de falha de ~50%.

---

## ISSUE-026 — Matriz de paridade com o Obsidian: 5 decisões que exigem o dono

- **Severidade:** 🟠 medium
- **Status:** 🔵 open
- **Achado em:** auditoria de paridade de 2026-10-22 (3 frentes independentes: sintaxe/parser, links/embeds/aliases/PDF/propriedades, CLI/MCP/documentação), contra as fontes primárias `obsidian.md/help/{syntax,links,tags,properties,embeds,aliases,pdf,cli}`
- **Medição completa:** [`docs/OBSIDIAN_PARITY_MATRIX.md`](../docs/OBSIDIAN_PARITY_MATRIX.md)

**Contexto.** A ADR-054 decide o escopo de paridade; este issue registra a
distância medida e as decisões que **não são de implementação**. A matriz
substitui a de 2026-10-18 citada na ADR-054:76, que nunca foi registrada no
repositório.

**As 5 decisões:**

| # | Decisão | Alternativas | Bloqueia |
|---|---------|--------------|----------|
| D1 | **Assets não-Markdown**: indexar `.pdf`/imagem/áudio/`.canvas`, ou declará-los fora do escopo? | (a) indexar com `asset_kind`/`mime`; (b) "o My-Memory indexa só Markdown; assets são referenciados mas não ingeridos" | busca dentro de PDF, viewer de PDF, citação por seleção (`#selection=`), export de nota para PDF, import de `.canvas` |
| D2 | **`move`/`rename`**: a ADR-054:33 lista como "a construir" e a :35 lista fora de escopo — a ADR contradiz a si mesma | (a) construir; (b) recusar e corrigir a ADR por supersession | paridade de manipulação de arquivo |
| D3 | **`property:set`/`property:remove`**: a ADR-056:39 prevê escrita de propriedade | (a) construir (destrava `properties`); (b) recusar e manter só leitura | ADR-056 |
| D4 | **`unresolved`/`orphans` isolados**: hoje vivem dentro de `mem doctor` e truncados em 15 itens | (a) comando dedicado; (b) manter como está | ADR-054:33 pede "isolado" |
| D5 | **Comentário `%%` no texto indexado**: removê-lo do FTS deixa a busca igual à do Obsidian (que o esconde), mas muda resultados de busca existentes | (a) só da extração (feito na T1 da feature de tags); (b) também do índice | paridade da busca |

**Além disso, 3 defeitos de parser corrigidos no mesmo ciclo** (feature
`obsidian-tag-parity`, T1): a regex de tag truncava `#ação` para `"a"`; blocos
`~~~` geravam tag e wikilink fantasma; comentários `%%` geravam tag real.

**Direção.** As 5 decisões são de produto e travam o roadmap de paridade.
A medição completa, com evidência `caminho:linha`, está na matriz.

---

## ISSUE-027 — O code graph do vault de repositório está inerte: motor ausente e ligação quebrada

- **Severidade:** 🔴 crítica
- **Status:** 🟢 resolvido (2026-10-24) — 18/18 tarefas implementadas na feature `code-graph-real`; ver **Resolução** no fim desta entrada
- **Achado em:** auditoria comparativa de 2026-10-23 entre o `my-memory` e o [Graphify-Labs/graphify](https://github.com/Graphify-Labs/graphify) `v0.9.94` (clone em `D:\.minimax\scratch\subagents\graphify-study\graphify`), com **verificador independente** que refutou 6 de 7 hipóteses da análise inicial
- **Decisão de arquitetura:** [ADR-061](../docs/adr/061-dois-conceitos-de-memoria-vault-central-e-code-graph.md) — vault central = memória de aprendizagem (Obsidian); vault de repositório = code graph (conceito do Graphify, motor do my-memory)

**Contexto.** A ADR-047 (code AST via tree-sitter) está **Accepted**, mas o
binding real nunca foi implementado: o que roda é um mock regex. A ADR-061
decide o caminho (motor AST real em Go puro, fases de ligação → motor →
superfície). Este issue registra **os 12 gaps concretos** que a execução tem de
fechar, cada um com a evidência.

**Medição empírica (verificador, sobre o próprio repo):** `mem code-index
--include='**/*.go' --lang=go` → **331 arquivos, 2595 símbolos, 120 arestas**;
`db.InitDB`, a função mais chamada do repo, **não tem nenhum vizinho** — não
existe aresta de chamada real em lugar nenhum.

| # | Gap | Evidência | Gravidade |
|---|-----|-----------|-----------|
| **C1** | O motor AST nunca foi implementado: `newBackendParser()` devolve `ErrTreesitterDisabled` mesmo com `-tags treesitter` **e não tem nenhum call site** — o caminho de produção é sempre o `MockParser` regex ("walker regex-by-line") | `internal/codeast/treesitter_enabled.go:57`, `internal/codeast/pipeline.go:69-71`, `internal/codeast/mock_parser.go:82` | 🔴 crítica |
| **C2** | **Não existe call graph.** As arestas são `references` por igualdade de nome; `imports` é explicitamente pulado ("imports tratados separadamente em T+") | `internal/codeast/pipeline.go:279-281, 299` | 🔴 crítica |
| **C3** | **As 2 tools MCP de código estão mortas**: registradas com handler `nil`; `SetCodeSearchHandler`/`SetCodeNeighborsHandler` só são chamados em teste → respondem "indisponível" (CA-14) | `internal/mcp/server.go:116-117`, `internal/mcp/code_handlers_test.go:312` | 🔴 crítica |
| **C4** | **`mem code-index` indexa 0 arquivos por default**: filtra por `cfg.ShouldIndex`, cujo `include` default é `**/*.md`. E `mem index` **nunca** chama o code pipeline (só `mem code-index`) | `internal/codeast/pipeline.go:376`, `internal/config/config.go:116`, `cmd/mem/main.go:864` | 🔴 crítica |
| **C5** | **Em Postgres o code graph fica vazio**: o pipeline abre o store PG e o descarta ("code_* tabelas via SQLite local (v1)"), escrevendo num SQLite local que nada consulta. As tabelas `code_*` são criadas no Postgres (`internal/db/schema.go:250-296`) e **permanecem vazias** | `cmd/mem/codeindex.go:83-95` | 🔴 crítica |
| **C6** | **`code_files`/`code_symbols`/`code_edges` não têm coluna `repository`** — o code graph está fora da federação `repo_id` | `internal/db/schema.go:89-133` | 🟠 alta |
| **C7** | **Dois grafos disjuntos**: `graph_edges` (notas, epistêmicas) e `code_edges` (símbolos) não têm ponte — o "Repository Brain" não existe como grafo único | `internal/graph` não menciona `code_*` (0 ocorrências) | 🟠 alta |
| **C8** | **`code_edges` não tem `epistemic_status`** e a resposta MCP **não expõe `confidence` nem call-site** — só `file`/`line` do símbolo destino. A proveniência que a ADR-011 dá ao grafo de notas não alcança o de código | `internal/db/code_migration.go:118-134`, `internal/mcp/code_handlers.go:41-49` | 🟠 alta |
| **C9** | **`include_code` da busca híbrida é placeholder**: o fan-in RRF code+markdown não existe, só um aviso "agendado para backlog (CA-12 placeholder)" | `internal/mcp/handlers.go:385-399` | 🟠 alta |
| **C10** | **A viewer não tem superfície de código**: nenhuma ocorrência de `code_symbol`/`code_edges`/`code_search` em `viewer/src`; rotas = index/notes/graph/drift. O code graph não tem rosto visual | `viewer/src` (0 ocorrências) | 🟡 média |
| **C11** | **`mem drift` ↔ `ast_hash` desligados**: `internal/drift` não referencia `ast_hash` nem `code_files` — o drift liga nota↔código por git, não por estrutura | `internal/drift` (0 ocorrências) | 🟡 média |
| **C12** | **Flags mortas no CLI de código**: `--repo`, `--ast-hash`, `--no-embed` e `--json` são aceitos e descartados (`_ = repoSlug` etc.). `--json` ainda avisa "sem efeito nesta versão" | `cmd/mem/codeindex.go:39-42` | 🟡 média |

**Armadilha de escopo (o erro mais caro possível aqui).** O `include` global é
compartilhado com o pipeline Markdown (`cmd/mem/main.go:1668`, `:1878`): incluir
`**/*.go` no `include` global faria o Go ser indexado **como nota**, poluindo o
grafo de notas, os embeddings e o drift. O escopo de código tem de ser
**separado** (`code.include`/`code.exclude`), conforme a ADR-061 §4.

**Direção.** A ADR-061 decide o *quê*. Este issue vira a spec
`.specs/features/code-graph-real/` (Specify → Design → Tasks → Execute), com os
12 gaps como requisitos rastreáveis, em três fases: **ligação** (C3, C4, C5, C6,
C12), **motor** (C1, C2, C8) e **superfície** (C7, C9, C10, C11).

### Resolução (2026-10-24)

A feature `code-graph-real` implementou os 12 gaps em 18 tarefas (T1..T18),
distribuídas em 3 fases. Cada gap foi fechado pelo commit indicado:

| Gap | O que era | Fechado por |
| :--- | :--- | :--- |
| **C1** | Motor AST nunca implementado (`newBackendParser` devolvia `ErrTreesitterDisabled` e não tinha call site) | `4f1182a` (T7, backend Go puro) + `a342518` (T8, `newBackendParser` vivo) |
| **C2** | Não existia call graph; só `references` por nome, `imports` pulado | `b986c37` (T9, `calls` com call-site) + `870a323` (T10, `imports`/`inherits`/`uses_type`) |
| **C3** | As 2 tools MCP de código mortas (handler `nil`) | `331be16` (T5, `SetCodeAccess` injeta os executores) |
| **C4** | `mem code-index` indexava 0 arquivos; `mem index` nunca chamava o code pipeline | `1bf7e75` (T1, escopo de código) + `3cd03b5` (T2, code stage no `mem index`) |
| **C5** | Em Postgres o code graph ficava vazio (store PG aberto e descartado) | `33931cb` (T3, engine configurado + camada de dialeto) |
| **C6** | `code_*` sem coluna `repository` (fora da federação `repo_id`) | `0907dbd` (T4, coluna `repository` + filtro por repo) |
| **C7** | Dois grafos disjuntos (`graph_edges` × `code_edges`) | `96a734b` (T13, ponte `references_code` no grafo único) |
| **C8** | `code_edges` sem `epistemic_status`; resposta sem confiança nem call-site | `5f52c2d` (T11, status + aresta ambígua) + `056a3f6` (T12, proveniência/confiança/call-site) |
| **C9** | `include_code` era placeholder | `12f3751` (T14, fan-in RRF real de `code_symbols`) |
| **C10** | A viewer não tinha superfície de código | `2a8d993` (T16, rota `/code`) |
| **C11** | `mem drift` ↔ `ast_hash` desligados | `c1deac3` (T17, drift estrutural) |
| **C12** | Flags mortas no CLI de código | `340c9cb` + `fd95961` (T6, flags reais; `--no-embed` saiu do CLI) |

**Medição depois da correção** (code stage real sobre este repo, SQLite
temporário): 349 arquivos, 6754 símbolos, **6948 arestas** (`calls` 4055,
`imports` 1984, `uses_type` 908, `references` 1, `inherits` 0 — o zero é o
número correto, não há embedding de struct no repo), proveniência
`EXTRACTED` 6367 / `INFERRED` 581, e **10 593 linhas** em
`code_edges_uncertain` (nunca promovidas a aresta resolvida).

**Evidência completa:** `.specs/features/code-graph-real/validation.md` (Fase 2 +
Fase 3 consolidadas num único arquivo). **Estado consolidado e pendências:**
`.specs/features/code-graph-real/HANDOFF.md`.

**Pendência que sobrou (não bloqueia o issue):** a verificação independente
(16 hipóteses) ainda não rodou — ver o item 1 da tabela de pendências do
`HANDOFF.md`.

## ISSUE-028 — O `repo_id` do repositório diverge do `id` no catálogo global

- **Severidade:** 🟠 alta
- **Status:** ✅ **resolved** (2026-10-29) — o dono aprovou: *alinhar o `id` do catálogo com o `repo_id` do repositório e fazer o `mem repos` rejeitar divergência*. As duas metades foram feitas: o catálogo global foi alinhado e o `mem repos` agora acusa a divergência com erro explícito e exit ≠ 0
- **Achado em:** 2026-10-24, durante o planejamento da mudança do repositório de `C:\repository\my-memory` para `D:`
- **Spec**: `.specs/features/repo-id-alignment/` (spec.md, tasks.md, validation.md)

**Contexto.** O `repo_id` é um identificador **criptográfico** (`repo_<12-hex>`,
gerado por `crypto/rand` em `internal/config/config.go:458-465`) e é
**persistido** no `.memory/config.yaml` do próprio repositório. O catálogo global
(`~/.memory/config.yaml`) tem uma lista `repositories:` com uma entrada por
repositório, cada uma com `id` e `path`.

Hoje os dois **não batem**:

| Onde | Valor |
| :--- | :--- |
| `.memory/config.yaml` do repo | `repo_id: "repo_e4c8f3b1a2d5"` |
| `~/.memory/config.yaml` → `repositories[0].id` | `repo_271444ab0133` |

**Por que isso não quebra hoje.** O cascateamento de config casa a entrada do
catálogo por **ID ou caminho** (`internal/config/config.go:695-697`:
`matchesID || matchesPath`). Com os IDs divergentes, o casamento acontece pelo
**caminho** — e a entrada do catálogo só sobrepõe o **storage** (engine + URL
do Postgres), nunca o `repo_id` (`config.go:698-705`).

**Por que isso é um risco.** O casamento por caminho é frágil:

1. Se o repositório for **clonado ou movido** para outro caminho, a entrada do
   catálogo deixa de casar e o override de storage (per-repo) é **perdido em
   silêncio** — o repo passa a usar o storage global sem nenhum aviso.
2. Se o mesmo repo for registrado **em dois caminhos**, a entrada do catálogo
   casa com o primeiro path e o **split-brain de identidade** aparece: um
   `repo_id` apontando para duas árvores de dados.
3. O `id` do catálogo é o que o `mem repos` mostra ao usuário — um `id` que
   não corresponde ao `repo_id` real do repositório é enganoso na hora de
   diagnosticar.

**Proposta de correção (precisa de decisão do dono).** Alinhar o `id` do
catálogo com o `repo_id` do repositório e fazer o `mem repos` **rejeitar** um
`id` que não bate com o `repo_id` do `.memory/config.yaml` do path registrado —
trocando o silencio por um erro explícito.

**Nota.** O `repo_id` **não** muda com a mudança de pasta: ele é persistido no
`.memory/config.yaml`, que move junto com o repositório. O que precisa ser
atualizado no catálogo global é o `path`.

### Correção (2026-10-29) — decisão do dono + fluxo spec-driven

O dono aprovou a proposta: **alinhar o `id` do catálogo com o `repo_id` do
repositório** e **fazer o `mem repos` rejeitar divergência** — trocando o
silêncio por erro explícito.

**O que foi feito:**

| Metade | O quê | Onde |
| :--- | :--- | :--- |
| **A — dado** | O catálogo global (`~/.memory/config.yaml`) passou de `repo_271444ab0133` para **`repo_e4c8f3b1a2d5`**, que é o `repo_id` real do repositório | `C:\Users\Felipe\.memory\config.yaml` → `repositories[0].id` |
| **B — código** | `mem repos` compara o `id` de cada entrada do catálogo com o `repo_id` declarado no vault do path registrado; divergência vira `❌` + erro explícito + **exit ≠ 0** | `cmd/mem/setup.go:274-418` (`auditRepositoryCatalog`, `repoIDDivergenceError`, `printRepoIDDivergences`) + `internal/config/config.go:330-396` (`LoadVaultRepoID`, read-only, restrito ao path registrado) |

**Por que isso fecha o risco 1 (repo clonado/movido).** Com o `id` alinhado, o
cascateamento casa pelo **id** (`matchesID || matchesPath`), e não mais só pelo
caminho: o override de storage (engine + URL do Postgres) **sobrevive à
mudança de path** — provado por `internal/config/config_test.go:942`. O risco 2
(split-brain por registro em dois caminhos) muda de forma: o `repo_id` é
imutável e viaja com o repo, então dois paths do **mesmo** repo continuam
sendo o mesmo repositório — que é o comportamento correto. O caso patológico é
o clone com o `.memory/config.yaml` **copiado**, que compartilha o `repo_id`; é
exatamente aí que a divergência passa a ser acusável pelo `mem repos`.

**Comportamento do `mem repos` depois da correção:**

| Situação | Marcador | Exit |
| :--- | :--- | :--- |
| `id` do catálogo igual ao `repo_id` do disco | ✅ | 0 |
| Entrada sem vault inicializado no path | ⚠️ | 0 |
| Vault sem `repo_id` declarado | ⚠️ | 0 |
| `id` do catálogo **≠** `repo_id` do disco | ❌ + erro com os dois valores, o path e a ação | **≠ 0** |

A auditoria é **read-only** (provado: os bytes do catálogo global e do vault
ficam idênticos e nenhum arquivo é criado).

**Prova contra o defeito.** Dois mutantes injetados no código real (com backup
byte-exato): (a) `Divergent = false` — o comportamento antigo, silêncio + exit
0; (b) listar a entrada divergente como `✅`. Ambos os mutantes **mataram** o
teste `TestRunReposCommand_DivergenciaRepoID` com o sintoma exato. Restaurado:
tudo verde.

**Gate:** `go build`/`go vet`/`go test -count=1 ./...` exit 0 (26 pacotes `ok`, verificado de forma independente pelo gestor); `gofmt -l` vazio na cópia LF; `validate_spec.py` / `validate_tasks.py` / `validate_state.py` sem erro.

**Por que não há `mem repos --fix`.** Decisão registrada na spec: a
auto-correção escreve no arquivo de configuração do usuário e pode **agravar**
o split-brain (o caminho oficial de realinhamento sobrescreve a entrada e
preserva o bloco `storage:`, mas um `--fix` genérico não tem essa garantia).
Ficou como feature própria, com dry-run e backup, se o dono pedir.

**Sem ADR nova.** O cascateamento de config (ADR-040) e a identidade federada
(ADR-033) continuam válidos e intocados; a mudança é de **diagnóstico**.

## ISSUE-029 — 8 defeitos na feature `code-graph-real`, achados pela verificação independente

- **Severidade:** 🔴 alta (2 defeitos altos + 1 alto-médio + 3 médios + 2 baixos)
- **Status:** 🟢 **corrigido e revalidado** (2026-10-26) — os 8 defeitos foram corrigidos e commitados, e a **revalidação independente** (16 hipóteses + 8 refutações das correções + 5 riscos residuais) deu **8/8 correções sobreviveram** — nenhuma refutação consumada. Relatório: `.specs/features/code-graph-real/validation-revalidation.md`
- **Achado em:** verificação independente de 2026-10-24 (agente `verifier`,
  read-only, 16 hipóteses; relatório completo em
  `.specs/features/code-graph-real/verification.md`)
- **Veredito da verificação:** 🔶 **PARTIAL** — o núcleo técnico está sólido e verificadamente
  real, mas não era PASS

| # | Severidade | Defeito | Quebra |
| :--- | :--- | :--- | :--- |
| **D1** | 🔴 alta | `EnsureCodeTables` (`internal/db/code_migration.go:79-101`) migra `code_files`/`code_symbols`/`code_edges` mas **não** `code_edges_uncertain.repository`, que o DDL pré-feature não tinha e que `InsertCodeEdgeUncertain` (`:290`) insere → `mem code-index` falha com `table code_edges_uncertain has no column named repository` em **qualquer vault anterior à feature** | AC-05, AC-26 |
| **D2** | 🔴 alta | `internal/db/hybrid.go:205` usa `RepositoryFromContext(ctx)`, mas `db.WithRepository` só é chamado no caminho do `mem index` (`cmd/mem/main.go:1624`) — **nunca no servidor MCP**. O fan-in de código só existe para `repo_id == "default"`; com `repo=acme`, `include_code=true` devolve resultado idêntico ao `false` | AC-07, AC-20 / CG-19 |
| **D3** | 🟠 alta-média | `internal/codeast/treesitter_enabled.go:44-59` é stub: o build `-tags treesitter` indexa **1 símbolo e 0 arestas** com `files_errors:0` e exit 0 — degradação silenciosa do code graph | AC-14 (emendada) |
| **D4** | 🟡 média | Hub threshold com p99 sobre subgrafos pequenos: `int(n*0.99)-1` cai no penúltimo índice → o nó de **maior grau** é sempre removido; em `depth=2` o vizinho **direto** (hop 1) foi podado e o hop 2 mantido | AC-22 |
| **D5** | 🟡 média | `cmd/mem/codegraph.go:32` declara `--repo` e `:39-40` o descarta (`_ = repoSlug`); sem `--storage`, usa sempre `db.InitDB` local | AC-10 / CG-09 |
| **D6** | 🟡 média | O hash estrutural do drift (`internal/drift/drift.go:548-551`) usa `HashKey()` **com número de linha**: comentário deslocou linhas → **falso positivo**; mudança de assinatura → **falso negativo**. O hash de cache e o estrutural precisam ser **dois hashes** | AC-26 |
| **D7** | 🔵 baixa | `viewer/src/routes/code.tsx:200` e `:321` apontam para a **ADR-047** (deferida) em vez da **ADR-061** | — |
| **D8** | 🔵 baixa | `cmd/mem/main.go:1417-1428` dá precedência a `MY_MEMORY_PG_URL` **sobre** a config global, o que contradiz o `AGENTS.md` §2 ("global > local > env") — foi a causa do incidente com o Postgres real (§Incidente) | — |

**Erros de análise nos próprios documentos da feature** (corrigidos):
a spec tem **29** ACs, não 28; a **AC-14 era insatisfazível** (exigia um backend
tree-sitter inexistente) e foi **emendada** para exigir que a tag *não* degrade o
grafo; `README.md:64` afirmava que o code graph é filtrado por `repository`
"como o grafo de notas" — **falso**, o grafo de notas não tem coluna
`repository` em nenhum engine; `spec.md:188` prometia que o escopo de código
padrão refletia as linguagens do backend ativo — só **Go** tem AST precisa, as
outras 9 seguem no mock heurístico.

**Incidente durante a verificação.** A env var `MY_MEMORY_PG_URL` tem precedência
sobre a config global (D8): em dois testes de `mem lesson`, o binário ignorou a
URL do config de scratch e indexou **2 notas-teste** no **Postgres real do dono**
(`repo_central` do cofre central). Nada foi apagado do store real. **Pendente
para o dono:** remover `licao-sem-postgres.md` e `licao-sem-postgres-3.md` de
`.../central/lessons/`.

## ISSUE-030 — o fallback do `MockParser` só produz aresta entre símbolos de mesmo nome simples

- **Severidade:** 🟡 média (impacto de documentação e de utilidade, não de correção)
- **Status:** 🔵 open — **documentação corrigida** nesta rodada; a decisão de fundo é do dono
- **Achado em:** revalidação independente de 2026-10-26 (achado **R-1**; relatório em
  `.specs/features/code-graph-real/validation-revalidation.md`)

**O que é.** O caminho legado `references` (`internal/codeast/pipeline.go:583-653`)
é a varredida por nome usada nas linguagens **sem** backend AST. O índice `byName`
(`:596-612`) é chaveado pelo `qualified_name` do símbolo **quando ele tem um**, e
pelo `name` quando não tem. A busca (`:614-623`) usa `s.QualifiedName` e, na
ausência dele, `s.Name`.

Um símbolo do `MockParser` é gravado **sem** `qualified_name` — então ele procura
a chave `"Beta"`, enquanto um símbolo Go homônimo está sob `"pkg.Beta"` → **0
candidatos**, nenhuma aresta. O caminho legado só acha alvo quando **outro**
símbolo também não tem `qualified_name` e tem o mesmo nome simples.

**Medido.** 2 arquivos Python → 5 símbolos do `MockParser`, todos com
`qualified_name = NULL`, **0** arestas `references` e **0** linhas em
`code_edges_uncertain`. Nem `Beta` (Python) casou com `pkg.Beta` (Go), que tem o
mesmo nome.

**Por que não é bug do motor.** A semântica "resolução literal de referências"
está na ADR-047 (§CA-01) e no próprio comentário da função (`pipeline.go:568-582`):
são referências **por igualdade de nome**, não chamadas. Casar um símbolo Python
com um símbolo Go de mesmo nome simples seria uma aresta **falsa**. O motor está
correto; a **documentação é que prometia mais do que o motor entrega**.

**Correção de documentação (feita em 2026-10-26).** `README.md`,
`docs/CLI_GUIDE.md`, `docs/ARCHITECTURE.md`, `docs/adr/README.md` e `CHANGELOG.md`
agora dizem que a aresta `references` do fallback é por **igualdade de nome**, sem
ponto de chamada, e que num código sem nomes duplicados o fallback produz poucas
ou nenhuma aresta.

**Decisão de fundo (do dono).** Fazer as linguagens sem AST produzirem arestas
reais exige um parser por linguagem ou o binding tree-sitter da ADR-047 — cujo
deferral segue intacto (0 de 3 gatilhos). Alternativa barata: um parser de
chamada *heurístico* por linguagem no `MockParser` (detectar `x(...)` na linha e
resolver por nome no escopo do arquivo), que daria `references` com call-site
aproximado — mas com proveniência `INFERRED` e risco de falso positivo.

## ISSUE-031 — colisão cross-repo no nó `code:<qualified_name>` em SQLite + Postgres — ✅ RESOLVIDO (P5 e P1)

- **Severidade:** 🟠 alta-média (multi-tenant) / 🟡 média (lacuna de verificação)
- **Status:** ✅ **resolved** — **P5 corrigido** (2026-10-26, `c3c6267`: o dono decidiu
  "alinhar o SQLite com o Postgres") e **P1 fechado** (2026-10-27: verificado contra
  um PostgreSQL real em banco descartável; ver a seção P1 abaixo)
- **Achado em:** revalidação independente de 2026-10-26 (riscos **P5** e **P1**)

### P5 — colisão cross-repo em SQLite

O nó do grafo é `code:<qualified_name>`, **global**. Em SQLite o `graph_nodes`
**não tem** coluna `repository` e o insert é `INSERT OR IGNORE`
(`internal/db/code_bridge.go:317-321`), e `loadBridgeSymbols` só filtra por
`repository` **no Postgres** (`:353-357`).

**Reproduzido.** Dois vaults (`acme` e `globex`) no **mesmo** arquivo SQLite,
ambos com `pkg.Alpha` em paths distintos: `code_symbols` tem só a linha do
*acme*, mas `graph_edges` tem **duas** — a nota do repo A **e** a do repo B
apontando para o **mesmo** nó `code:pkg.Alpha`. Os dois repositórios colapsam no
mesmo nó.

**No Postgres não há colisão**: `graph_nodes` tem `repository` e o conflito é
`(repository, id)` (`code_bridge.go:311-313`); `graph_edges` usa
`(repository, source_id, target_id, relation)` (`:331-335`). O próprio código
declara a assimetria (`code_bridge.go:91-92`).

**Opções (do dono).** (a) nomear o nó por repositório
(`code:<repository>:<qualified_name>`) — quebra a compatibilidade dos ids já
gravados; (b) adicionar `repository` ao `graph_nodes` no SQLite, alinhando os
dois engines; (c) declarar que o SQLite é single-tenant no code graph e
documentar a limitação.

**Decisão do dono (2026-10-26): (a) — o id carrega o repositório.** A opção (b)
foi descartada por um motivo técnico que só apareceu ao medir: a PK de
`graph_nodes` no SQLite é `id TEXT PRIMARY KEY`. Uma coluna `repository` sem
trocar a PK **não separaria** dois repos com o mesmo id — e ainda gravaria um
dado mentiroso (a linha ficaria com o repository de quem criou primeiro).
Trocar a PK exigiria **reconstruir a tabela** e mexer nas ~40 queries do grafo
de notas do SQLite, que é de tenant único por ADR-040: é uma mudança do modelo
de identidade do engine, não um commit.

**Correção aplicada (`c3c6267`).** O id do nó de código passou a ser
`code:<repository>:<qualified_name>` **nos dois engines** — o SQLite ganha a
mesma semântica de identidade do Postgres (cuja identidade é `(repository,
id)`), sem rebuild e sem mexer na PK. Além disso:

- `loadBridgeSymbols` passou a filtrar por `repository` **também no SQLite** —
  era o que fazia a nota do repo B casar com o símbolo do repo A;
- o **cache do índice de símbolos** passou a ser chaveado por repositório nos
  dois engines (ele era chaveado só pelo ponteiro do banco: sem isso, o filtro
  por repo era derrotado pelo cache — defeito **novo**, achado na implementação);
- `CodeSymbolByNodeID` faz o parsing do id novo, aceita o formato legado
  `code:<qn>` e o qualified name puro, e resolve o símbolo dentro do
  repositório do contexto;
- o handler MCP de impacto passou a entregar o id inteiro para o `db` parsear
  (ele removia o prefixo `code:` por conta própria e passaria a resolver
  `acme:pkg.Alpha` como qualified name).

**Prova de que o defeito era real:** o teste de regressão
(`internal/db/code_bridge_repository_test.go`) foi executado com o fix
revertido e **falhou com o sintoma exato do P5** — os dois repositórios geraram
o mesmo id de nó (`code:pkg.Alpha`) e a ponte ofereceu os 2 símbolos aos 2
repos. Com o fix: 2 nós distintos, cada nota ligada só ao nó do próprio repo e
cada repo enxergando só o próprio símbolo.

**Migração: ZERO.** O Postgres real do dono tinha **0** nós `code_symbol`
(verificado antes da mudança), então nenhum id antigo precisou ser migrado.

**O que ainda fica de fora (decisão futura).** O **grafo de notas** do SQLite
continua de tenant único (a tabela `documents` não tem `repository` e `path` é
único) — alinhar isso é mudar o modelo de identidade do engine e pertence a
uma spec própria, não a este hotfix.

### P1 — Postgres real não verificado → ✅ FECHADO (2026-10-27)

**Status:** ✅ **resolved** (2026-10-27) — verificado contra um **PostgreSQL real**
(Neon 16.15), sempre contra um **banco descartável** criado e derrubado pelo
próprio teste. O store real do dono nunca foi alvo de escrita.

#### O que estava sem prova e como foi fechado

| Item | Antes | Evidência (2026-10-27) |
|---|---|---|
| `ON CONFLICT (repository, path)` no Postgres | sem prova | `internal/db/p1_postgres_test.go::TestP1Postgres_OnConflictRepositoryPathAtualizaEIsolaRepositorio` — o 2º upsert mantém o mesmo `id` e muda `language`/`content_hash`/`size_bytes`; o mesmo `path` em outro `repository` vira linha distinta; o SQL que chega ao servidor é `ON CONFLICT (repository, path) DO UPDATE` |
| Índice único composto de `code_*` | sem prova | `::TestP1Postgres_IndiceUnicoCompostoDeCodeFiles` — `pg_catalog` confirma `idx_code_files_repository_path` UNIQUE, válido, com as colunas `(repository, path)` **nessa ordem**; nenhum índice único cobre só `path` |
| `?`→`$1` em runtime real | sem prova | `::TestP1Postgres_PlaceholderOrdinalEmRuntimeReal` — o SQL que chega ao servidor é `SELECT $1 AS "col?x", 'a?b' AS "lit", $2 AS plain`; o `?` dentro de literal e de identificador **não** é convertido; o `INSERT` gravado tem `$1..$7` |
| Metade Postgres da AC-05 (H4) | sem prova | `cmd/mem/codeindex_postgres_test.go::TestCodeIndexCLI_PostgresGravaCodeGraphNoEngineConfigurado` — a CLI real grava `code_files`/`code_symbols`/`code_edges` no **Postgres** (`"engine": "postgres"`), com `content_hash` SHA-256 e a aresta `calls` no ponto de chamada (linha 4, não a da definição, linha 3); nenhum arquivo SQLite é criado no vault |

**Cada asserção foi provada contra o defeito** (executada com o comportamento
quebrado, e restaurada em seguida):

- (a) índice → `ON code_files(path)`: `colunas de idx_code_files_repository_path = [path]; want [repository path]`
- (b) conflict target → `(path)`: `pq: there is no unique or exclusion constraint matching the ON CONFLICT specification (42P10)`
- (c) binding → `?` convertido dentro de literal: `coluna do literal = "a$1b"; want "a?b"` — **dado errado no banco**, não apenas erro
- (d) engine → forçado para SQLite: `engine = "sqlite"; want "postgres"` e `linhas em code_files = 0; want 2`

#### Por que agora foi possível (o que faltava na rodada anterior)

A rodada anterior mediu `127.0.0.1:5432` e não achou Postgres descartável
(sem serviço, sem `psql`, sem `docker`). O que mudou: **o próprio teste cria o
banco descartável** com `CREATE DATABASE mymemory_pgtest_<pid>_<nanos>` e o
derruba com `DROP DATABASE ... WITH (FORCE)` — o `default` role do dono nunca
toca o banco `mymemory`. O servidor Neon (`ep-purple-thunder-a4q5bxac-pooler`)
aceita `CREATE DATABASE` e o harness vive em `internal/pgtest`.

#### Causa-raiz do lixo no store real (corrigida nesta rodada)

O `store real` estava sendo poluído porque **`internal/db` e `internal/store`
não tinham `TestMain`** neutralizando `MY_MEMORY_PG_URL` — que tem precedência
sobre qualquer config. Só `cmd/mem`, `cmd/mymemoryd` e `internal/config` tinham
o guard. Resultado: os testes de integração desses dois pacotes gravavam direto
no banco do dono a cada `go test`.

**Correção:** `internal/db/main_test.go` e `internal/store/main_test.go` agora
usam `pgtest.Capture()` (captura a URL **antes** de neutralizar) +
`pgtest.Neutralize()` (limpa `MY_MEMORY_PG_URL`, `MY_MEMORY_PG_PASSWORD`,
`POSTGRES_URL`, `DATABASE_URL`) e restauram no fim. A integração só roda com
`MY_MEMORY_PG_TEST=1` e sempre contra banco descartável.

**Prova de hermeticidade:** `go test -count=1 ./...` com `MY_MEMORY_PG_URL`
**presente** no ambiente → **exit 0, 27/27 pacotes ok**. As contagens do store
real antes e depois foram **idênticas** (`documents 456 | chunks 455 |
graph_nodes 792 | graph_edges 527 | document_tags 105`, `code_* 0`) e
`0` bancos `mymemory_pgtest_*` ficaram para trás.

#### Limpeza do store real

O lixo acumulado (**126 repositórios**, 370 documentos, 67 chunks, 586
graph_nodes, 5 graph_edges, 105 document_tags) foi removido em **uma transação**,
escopado por uma **lista explícita de 126 repositórios** derivada dos dados
(sem `LIKE`/curinga), com:
- guard de runtime que **aborta** se a lista contiver um repositório legítimo;
- exclusão explícita de `repo_central` e `FelipeMiiller/my-memory` em **cada**
  `DELETE` (cinta e suspensorio);
- `document_tags` deletado explicitamente (não tem FK para `documents`).

**Conferência posterior:** `repo_central` intacto em `86 documents / 388 chunks
/ 204 graph_nodes / 522 graph_edges`, `FelipeMiiller/my-memory` com seus 2 nós,
**0** documentos e **0** graph_nodes de lixo restantes.

## ISSUE-032 — `mem index` aborta no Postgres: a prepared statement "unnamed" do servidor é corrompida (26000 / 08P01)

- **Severidade:** 🟠 alta (impede a indexação do vault num Postgres com pooler)
- **Status:** 🔵 **open** — **workaround aplicado e verificado** (indexação completa em 276 documentos), **correção de código pendente de decisão do dono**
- **Achado em:** 2026-10-29, durante a indexação do vault deste repositório no Postgres real (operação autorizada pelo dono)

**Sintoma.** Contra o endpoint **com pooler** do Neon (`*-pooler.*`), o `mem index`
**aborta** no meio da execução com erro de protocolo:

```
Erro durante a indexação: pq: unnamed prepared statement does not exist (26000)
Erro durante a indexação: pq: bind message supplies 11 parameters, but prepared statement "" requires 4 (08P01)
```

Antes do abort, aparecem avisos do mesmo tipo em gravuras de vetor
(`Aviso: falha ao gravar vetor de "…" (pq: bind message supplies 6 parameters,
but prepared statement "" requires 2 (08P01)); chunk fica sem embedding`).

**Por que é grave.** O `lib/pq` usa o **statement "unnamed"** do servidor para
toda query parametrizada (todo `ExecContext`/`QueryRowContext` com argumentos).
Os dois erros significam que o statement unnamed do servidor **não é mais o que
o cliente acha que é**: ou ele sumiu (`26000`), ou ele tem outra forma
(`requires 4` quando o cliente mandou 11 parâmetros). A partir daí, **toda**
query parametrizada daquela sessão falha, e o `mem index` — que aborta no
primeiro erro fatal — para. O vault fica **parcialmente indexado**, sem
aviso de que sobrou documento de fora.

**Observações importantes.**

1. **Não é concorrência no código.** O `mem index` é **sequencial**
   (`filepath.WalkDir`, sem worker pool — `cmd/mem/main.go:1724`). O erro
   acontece numa sessão de uma conexão só.
2. **É determinístico.** 10 execuções seguidas falharam **no mesmo documento**
   (`.specs/features/federated-central-vault-and-repo-identity/spec.md`),
   sempre com `requires 4`.
3. **O pooler é o gatilho.** Com o endpoint **direto** (sem `-pooler`), o mesmo
   `mem index` passa pelo ponto exato onde falhava e **completa**: `🎉
   Concluído! 276 documentos processados no PostgreSQL (190 indexados, 86 em
   cache, 0 podados)`. O `lib/pq` + pooler em modo transacional é a causa
   clássica desse par de erros: o Parse vai para um backend e o Bind vai para
   outro.
4. **Um probe dedicado não reproduz isolado.** 300 × (SELECT de 2 params +
   INSERT de 4 params) contra o pooler deu `26000=9, 08P01=42` na primeira
   versão do probe (que tinha um bug próprio: 11 argumentos para 4
   placeholders) e `ok=300/300` depois de corrigido, nos **dois** endpoints. Ou
   seja: o gatilho é a **variedade de formas de statement** do caminho de
   indexação (2, 4, 6 e 11 parâmetros), não o volume de queries.

**Workaround aplicado (2026-10-29).** Indexar com o endpoint **direto**:
`MY_MEMORY_PG_URL` apontando para `ep-<id>.<region>.aws.neon.tech` (sem
`-pooler`). A env var tem precedência sobre o config (ADR-040), então **o
`~/.memory/config.yaml` do dono não foi alterado**. Resultado: 276 documentos,
2415 chunks, 151 arestas, 333 nós em `repo_e4c8f3b1a2d5`; `repo_central`
intacto (86/388/522/204); `FelipeMiiller/my-memory` no baseline (0/0/0/2).

**Opções de correção (decisão do dono).**

| # | Abordagem | Custo | Efeito |
| :--- | :--- | :--- | :--- |
| **A** | Usar o endpoint **direto** no `~/.memory/config.yaml` | zero código | resolve para o usuário; o pooler continua disponível para quem precisar de multiplexação |
| **B** | Trocar `lib/pq` por `pgx` com `default_query_exec_mode=simple_protocol` | médio (driver + SQL placeholder) | resolve no código, e é a recomendação canônica para poolers em modo transacional |
| **C** | Tornar o `mem index` **tolerante**: erro de protocolo no documento = documento falhou, seguir o próximo | baixo | deixa de abortar a indexação inteira; **não** conserta a causa |
| **D** | Configurar o pool (`SetMaxOpenConns`, `SetConnMaxLifetime`) | baixo | pode reduzir a exposição; **não** está isolado como causa |

**Recomendação:** **A + C**. A resolve o caso do dono hoje, sem código; C
transforma "a indexação inteira morreu sem aviso" em "um documento falhou e foi
reportado". B é a correção de raiz e pode entrar depois, por fluxo spec-driven.

**Não é regressão do P1.** A integração com Postgres foi provada em 2026-10-27
(ISSUE-031 §P1) contra um banco **descartável** — que usava o mesmo endpoint
com pooler e passou. O que falhou aqui é o caminho de **indexação em volume**,
que faz muito mais queries por documento do que o teste de integração.

## ISSUE-033 — `validate_state.py` reprova 11 das 41 features por desenho, não por conteúdo

- **Severidade:** 🟡 média (dívida de documentação + gate que não distingue relatório de plano)
- **Status:** 🔵 **open** — achado em 2026-10-29, durante a leitura dos 83 alertas de drift
- **Achado em:** verificação do gate do fluxo spec-driven após a indexação do vault
- **Impacto no CI:** **nenhum** — `validate_state.py` **não roda no CI** (o workflow roda `go test`, build e front-end). É um gate **manual**.

**O que acontece.** `validate_state.py` coleta as linhas "candidatas" do
`validation.md` — as que casam com `^#{1,4}\s*validation\b` ou
`\*{0,2}result\*{0,2}\s*:` — e procura `PASS`/`FAIL` **só nelas**. O problema
tem duas faces:

1. **O fallback é errado.** Sem nenhuma linha candidata, o script usa o
   **texto inteiro** como palha (`hay = text`). Um relatório longo que contenha
   as duas palavras — como `viewer-dashboard` (**53 PASS, 2 FAIL**) — é
   classificado como **"template não preenchido"**, embora não seja template
   nenhum. É um **falso positivo do validador**.
2. **O filtro não reconhece o relatório.** O veredito precisa estar **na linha
   do heading ou da linha `Result:`** — não basta estar no documento. Um
   relatório cujo H1 é `# Validation Report: X` e cujo veredito está numa
   tabela 200 linhas abaixo tem 0 candidatos uteis.

**Resultado:** **11 das 41 features** reprovam o cross-check. Elas se dividem
em dois grupos bem diferentes:

| Grupo | Features | Realidade |
| :--- | :--- | :--- |
| **Têm veredito de verdade, o validador não acha** | `embedder-builtin-onnx-minilm` (2 PASS), `event-runtime-event-log` (2 PASS / 10 FAIL), `feat-code-ast` (19/4), `post-release-v1.3.0-bugfixes` (18/0), `triptych-node-inspector` (9/0), `viewer-dashboard` (53/2) | O relatório existe e é real; **só o formato não é lido** |
| **O `validation.md` é um PLANO, não um relatório** | `ai-tool-auto-wiring`, `context-packager-and-subgraph-bundle`, `graph-path-discovery`, `progressive-context-loading`, `staleness-detection-and-banners` | O H1 é `# Validation Plan: X` e o arquivo tem **0 PASS e 0 FAIL** — nunca houve relatório de verificação. Aqui o gate está **certo**: falta a evidência |

As 3 features tocadas em 2026-10-28/29 (`code-graph-real`,
`doctor-empty-vault-score`, `repo-id-alignment`) reprovavam e **passam** agora
pelo bloco canônico `## Validation: PASS` + `**Result**: PASS` — que é
exatamente a forma que o validador lê.

**Opções (decisão do dono).**

| # | Abordagem | Custo | Efeito |
| :--- | :--- | :--- | :--- |
| **A** | Corrigir o **validador**: aceitar linhas `Verdict`/`Veredito`/`Resultado` como candidatas e, **sem** candidatas, devolver "sem veredito" em vez de varrer o texto inteiro | ~5 linhas em `validate_state.py` | elimina o falso "unfilled" e faz 6 features passarem **sem tocar em nenhum relatório** |
| **B** | Escrever o bloco canônico `**Result**: PASS` nos 6 relatórios que **já têm** veredito real | 6 edições de 1 linha | as 6 passam — **só** se o veredito do relatório for de fato PASS (conferir um a um) |
| **C** | Para as 5 features de "# Validation Plan": escrever o relatório de verificação que nunca existiu | fluxo spec-driven por feature | fecha o buraco de verdade, mas é trabalho de verdade |

**Recomendação:** **A + B** para o gate (A corrige o bug do validador, B
normaliza os 6 relatórios reais), e **C** comoitem separado — porque as 5
features do grupo "plan" **nunca foram verificadas**, e fingir que foram é
pior que deixar o gate vermelho.

**Nota de método (2026-10-29):** as 2 features cujo drift era real
(`viewer-dashboard/validation.md` e
`federated-central-vault-and-repo-identity/validation.md`) foram corrigidas
na mesma rodada, porque as minhas mudanças é que as deixaram desatualizadas. O
resto dos 81 alertas de drift era a reação mecânica do detector a uma rodada
grande — ver `.specs/STATE.md`.

---