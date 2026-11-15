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
- **Status:** 🟡 **parcial** — **opção A entregue** (2026-10-29: endpoint direto no config global **e** na env var de escopo User) e **opção C entregue** (2026-10-29: `mem index` tolerante, `.specs/features/fix-index-tolerancia/`); **opção B (pgx) adiada** pelo dono — fica para feature própria com ADR
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

**Correção A aplicada (2026-10-29).** O endpoint **direto** foi configurado em
**dois** lugares, porque a env var tem precedência sobre o config (ADR-040) e
*sombreava* o config global:

1. `~/.memory/config.yaml` → `storage.postgres_url` sem `-pooler.`;
2. variável de ambiente `MY_MEMORY_PG_URL` no **escopo User do Windows**, que
   apontava para o pooler → realinhada para o endpoint direto.

A existência da env var de escopo User apontando para o pooler foi **achado
novo**: trocar **só** o config não entregaria a opção A. Verificado: `env ==
config (host)`.

Resultado (endpoint direto): 276 documentos, 2424 chunks, 151 arestas, 333 nós
em `repo_e4c8f3b1a2d5` — `mem doctor --storage=postgres --repo
repo_e4c8f3b1a2d5` → Health Score **75/100**; `repo_central` intacto
(86/388/522/204); `FelipeMiiller/my-memory` no baseline (0/0/0/2).

⚠️ **Mudança fora do repositório:** o config global do dono e a env var de
escopo User foram alterados. Reverter é voltar ao host com `-pooler.` **nos
dois** lugares.

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

**Resolução (2026-10-29).** O dono aprovou "A + C" e adiou "B".

| # | Abordagem | Decisão | Entrega |
| :--- | :--- | :--- | :--- |
| **A** | endpoint direto | ✅ **aplicada** | config global + env var de escopo User apontando para o endpoint direto; verificado por `mem doctor` (75/100) |
| **B** | `lib/pq` → `pgx` | ⏸ **adiada** | fica para feature própria com ADR; **não** foi feita |
| **C** | `mem index` tolerante | ✅ **entregue** | `.specs/features/fix-index-tolerancia/` — a falha de um documento não aborta a varredura: o documento falho é registrado, o index segue para o próximo e o comando fecha com um resumo "N falhos" |
| **D** | pool configurado | — | não escolhida |

**B fica registrada como pendência.** A causa de raiz (o statement "unnamed" do
servidor) continua lá: **A** contorna e **C** contém o estrago, mas quem
corrige a causa é **B**.

**Fora do escopo desta ISSUE, mas verificado na mesma rodada:** o
`internal/watcher/indexer.go` tem indexador próprio, **por documento** (o
watcher indexa um arquivo por evento) — e **não** tem o defeito de "abortar a
indexação inteira": o loop do watcher (`cmd/mem/main.go:1324`) **reporta o erro
do documento e segue**. O caminho Postgres do watcher já põe o repositório no
contexto para a ponte (`internal/watcher/indexer.go:293`). **Nada a corrigir
aqui.**

**Não é regressão do P1.** A integração com Postgres foi provada em 2026-10-27
(ISSUE-031 §P1) contra um banco **descartável** — que usava o mesmo endpoint
com pooler e passou. O que falhou aqui é o caminho de **indexação em volume**,
que faz muito mais queries por documento do que o teste de integração.

## ISSUE-033 — `validate_state.py` reprova 11 das 41 features por desenho, não por conteúdo

- **Severidade:** 🟡 média (dívida de documentação + gate que não distingue relatório de plano)
- **Status:** 🟢 **A + B entregues** (2026-10-29, `.specs/features/fix-validate-state-verdict/`); **C adiada** pelo dono — as 5 features de plano continuam **vermelhas de propósito**
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

**Resolução (2026-10-29).** O dono aprovou "A + B"; "C" foi **adiada**.

| # | Abordagem | Decisão | Entrega |
| :--- | :--- | :--- | :--- |
| **A** | corrigir o validador | ✅ **entregue** | `validate_state.py` reconhece `Verdict:`/`Veredito:`/`Resultado:`; **sem** candidata → "sem veredito" (o fallback que varria o texto inteiro saiu); `PARTIAL` passou a ser veredito próprio |
| **B** | bloco canônico nos 6 relatórios | ✅ **entregue** | 5 relatórios com veredito PASS real ganharam `## Validation: PASS` + `**Result**: PASS` (com evidência `file:line` conferida); `feat-code-ast` ficou **PARTIAL** |
| **C** | escrever os 5 relatórios de plano | ⏸ **adiada** | continuam vermelhos — falta a evidência |

**Antes → depois: 11 erros → 6** (os 5 de plano + `feat-code-ast` como
PARTIAL), com **zero regressões** entre as features que já passavam. O teste
novo do validador (17 casos, incluindo 2 mutantes) está em
`.agents/skills/tlc-spec-driven/scripts/test_validate_state.py`.

**`feat-code-ast` continua vermelha, de propósito.** O relatório diz, na seção
"6. Verdict Summary": `**Verdict final**: **PARTIAL**. Não PASS puro (CA-01
stub + CA-12 placeholder)`. Carimbar `PASS` ali seria verde por cima de 3 ACs
com SPEC-GAP (CA-01 / CA-12 / CA-16) mais o sub-item CA-14a. O validador agora
trata `PARTIAL` como veredito próprio e reporta a feature como não-promovível,
com a mensagem de fechamento. O relatório recomenda abrir fix-tasks para
GAP-1/2/3 antes de promover a feature.

**Nota de método (2026-10-29):** as 2 features cujo drift era real
(`viewer-dashboard/validation.md` e
`federated-central-vault-and-repo-identity/validation.md`) foram corrigidas
na mesma rodada, porque as minhas mudanças é que as deixaram desatualizadas. O
resto dos 81 alertas de drift era a reação mecânica do detector a uma rodada
grande — ver `.specs/STATE.md`.

---

## ISSUE-034 — O code graph grava uma linha por round-trip: com o escopo `code:` ligado, o `mem index` levaria ~6 h num Postgres remoto

- **Severidade:** 🟠 alta (torna o ADR-061 inutilizável em engine remoto)
- **Status:** 🟢 **corrigida** (2026-10-29, `.specs/features/batch-code-bridge-writes/`) — as três escritas foram posto em lote; restou **um** ponto medido como próximo candidato
- **Achado em:** 2026-10-29, ao habilitar o escopo `code:` no vault deste repositório

**Sintoma.** Com o escopo `code:` habilitado (ADR-061), o `mem index` passa a rodar
o code stage **e** a ponte `notes↔código`. Os dois caminhos de escrita gravam
**uma linha por round-trip**:

| Caminho | Escritas por rodada | Round-trips |
| :--- | :--- | :--- |
| `internal/codeast/cache.go:160` → `PersistSymbols` | 1 `INSERT` + 1 `SELECT` de id por **símbolo** (o `lib/pq` não implementa `LastInsertId`) | ~2 × ~6 750 símbolos |
| `internal/db/code_bridge.go` → `buildCodeBridge` | 1 `INSERT` por nó + 1 por aresta, por **(documento, símbolo)** | ~2 × ~34 000 arestas |

**Medição no Neon** (us-east-1, desta máquina): **~268 ms por round-trip** →
**0,78 símbolos/s**. Popular o `code_symbols` do repo levaria **~2 h**; a ponte,
**~4,7 h** por execução do `mem index`.

**Por que é grave.** O defeito nunca apareceu nos testes porque o SQLite local
faz as mesmas escritas em milissegundos. Em engine remoto a mesma lógica é
inviável: um `mem index` de rotina levaria horas, e o dono não pode adotar um
custo assim.

**Correção entregue (2026-10-29).** `.specs/features/batch-code-bridge-writes/` —
as **três** escritas que gravavam uma linha por round-trip passaram a gravar
**uma linha por documento/arquivo**, com as mesmas colunas, o mesmo
`ON CONFLICT` e a mesma idempotência. **Sem** mudança de schema, **sem** troca
de driver, **sem** índice novo.

| Escrita | Antes | Depois | No Neon (268 ms/round-trip) |
| :--- | :--- | :--- | :--- |
| `PersistSymbols` → `code_symbols` | 2 round-trips por **símbolo** | 2 por **arquivo** | ~2 h → **~3,4 min** |
| `FlushEdges` → `code_edges` | 1 por **aresta** | 1 por **arquivo** | ~30 min → **~1,7 min** |
| `FlushEdges` → `code_edges_uncertain` | 1 por **aresta incerta** | 1 por **arquivo** | ~110 min → **~1,7 min** |
| `buildCodeBridge` (ponte) | 2 por **(documento, símbolo)** | 2 por **documento** | ~4,7 h → **~2,5 min** |

Medido no run real: a primeira população do code graph deste repo (381 arquivos
/ 7 418 símbolos) saiu em **2 173 s (36,2 min)** — antes, >2 h e impraticável. O
`mem index` de rotina, com cache quente, fica em **~2,5 min**. (A fase de
resolução, que sozinha pagava ~20 min, foi para <1 min depois — ver
`.specs/features/batch-edge-resolution-lookups/`.)

**O ponto que ficou de fora foi fechado depois (2026-11-13).** As **lookups de
resolução** (1 `SELECT` por destino distinto) foram posto em lote por
`.specs/features/batch-edge-resolution-lookups/`: antes de resolver as arestas,
o resolver busca **todos** os destinos necessários com um punhado de queries
(`IN (...)`, blocos de 500 valores), preenchendo o mesmo cache do caminho por
destino. **A chave da segurança:** a resolução acontece só depois que todos os
arquivos do lote foram indexados (`internal/codeast/pipeline.go`, §Run), então o
conjunto de símbolos é o mesmo — o resultado é idêntico, provado aresta a
aresta, e o caminho por destino continua como fallback. **Medido:** a fase de
resolução saiu de **~20 min para <1 min** e a população completa do code graph
de **2 173 s (36,2 min) → 1 226 s (20,4 min)**; o número de queries que chega
ao servidor cai de **≥10 para ≤2** (10 destinos) e para **zero** durante a
resolução com o cache cheio.

---

## ISSUE-035 — O caminho estrutural do `mem drift` está morto no Postgres: o repositório não chega no contexto

- **Severidade:** 🟠 alta (o detector de drift estrutural — ADR-061 AC-26 — nunca engage no engine de produção)
- **Status:** 🟢 **corrigida** (2026-10-29, `.specs/features/fix-drift-repo-context/`) — o `mem drift` roda com escopo de repositório; mudança cosmética é suprimida e a estrutural é detectada e atribuída
- **Achado em:** 2026-10-29, ao medir o drift com o code graph populado pela primeira vez

**O bug.** `cmd/mem/drift.go:45` resolve o repositório e **o descarta**:

```go
_, resolvedDB, resolvedPG := resolveStorageAndRepo(cfg, *targetRepo, *dbPath, *pgURL, defaultRepo)
```

O `ctx` que chega em `drift.AnalyzeDrift(ctx, ...)` (`:81`) **não carrega o
`repository`** — ao contrário de `cmd/mem/codeindex.go:84`,
`cmd/mem/codegraph.go:68` e `cmd/mem/main.go:1620`, que fazem
`ctx = db.WithRepository(ctx, repository)`.

**Consequência no Postgres.** `internal/db/code_bridge.go` →
`CodeSymbolSignatures` filtra por repositório **só no Postgres** (`f.repository
= ? AND s.repository = ?`); com o contexto vazio a consulta devolve **0 linhas**.
Então `internal/drift/drift.go:561` calcula o hash estrutural gravado como `""`,
contra um hash atual não-vazio → **`res.Structural = true` para todo arquivo
indexado**. E a atribuição pela ponte erra o alvo pelo mesmo motivo: o id do nó
é prefixado com o repositório (`db.CodeSymbolNodeID`).

**Prova medida.** `internal/config/config_test.go`, cujo `content_hash` gravado
era **idêntico** ao SHA-256 do arquivo em disco (o arquivo não tinha mudado):

| Contexto | Lado gravado | Lado atual | Veredito |
| :--- | :--- | :--- | :--- |
| com `WithRepository` | 37 símbolos · `1ef039df…` | 37 símbolos · `1ef039df…` | **iguais** → cosmético → fora do drift |
| sem repositório | 0 símbolos · `""` | 37 símbolos · `1ef039df…` | divergentes → **estrutural** (falso) |

**Por que ninguém pegou.** No **SQLite** não há filtro de repositório em
`CodeSymbolSignatures` — os testes de `internal/drift` (fixture SQLite) passam
e o defeito só aparece no Postgres, que é o engine de produção.

**Sintoma no relatório.** Todo arquivo de código indexado aparece como
`structural_changes` e as notas que o mencionam são acusadas como drift
(**15 HIGH + 3 MÉDIO** no range `adfb8cc~5..adfb8cc`) — inclusive arquivos que
**não mudaram** desde o último índice.

---

## ISSUE-036 — O `mem doctor` conta 38 306 links `references_code` como quebridos, e a rotina de correção apagaria o code graph

- **Severidade:** 🟠 alta (o Health Score do vault cai 15 pontos e o "fix" do doctor destrói as arestas do code graph)
- **Status:** 🟢 **corrigida** (2026-10-29, `.specs/features/fix-doctor-dead-links-code-nodes/`) — o nó de código não é mais contado como link quebrado, e a rotina de correção não apaga mais o code graph
- **Achado em:** 2026-10-29, ao rodar o `mem doctor` depois de popular o code graph pela primeira vez

**O defeito.** O check de dead links do `mem doctor` considera "vivo" **só** o
alvo que resolve para um **documento** (id ou title) — as tags (`#%`) são
ignoradas por desenho. Os nós do code graph (tipo `code_symbol`, ADR-061) **não
são documentos**, e o id deles é `code:<repo>:<qualified_name>`
(`db.CodeSymbolNodeID`). Então **toda** aresta `references_code` é contada como
link quebrado:

- **Postgres** — `internal/store/doctor.go:96-101`;
- **SQLite** — `internal/db/graph.go:271-278` (mesma forma, sem filtro de repositório).

**E o caminho de correção apaga o code graph.** `internal/db/graph.go:420-426`
(`DELETE FROM graph_edges WHERE target_id NOT LIKE '#%' AND … NOT IN (SELECT id
FROM documents) AND … NOT IN (SELECT title FROM documents)`) classifica as
**38 304 arestas `references_code`** como mortas e **as apaga**. Quem rodar essa
rotina num vault com code graph destrói a ponte notas↔código.

**Prova medida (Neon, `repo_e4c8f3b1a2d5`).**

| Métrica | Valor |
| :--- | :--- |
| nós `code:*` no grafo | **2 959** (existem) |
| arestas `references_code` | **38 304** |
| arestas `references_code` **sem** nó de destino | **0** → nenhum link está realmente quebrado |
| `mem doctor` reportando | **38 306** dead links (falso positivo) |
| Health Score | **60/100** (antes do code graph: **75/100**) |

**Por que é grave.** A configuração recomendada (habilitar o escopo `code:`)
**rebaixa o Health Score do vault em 15 pontos** com 38 mil falsos positivos — e
o "fix" do doctor, se rodar, leva o code graph junto.

---

## ISSUE-037 — O binário distribuído nunca carrega o provider semântico: a busca vetorial do cofre decai em silêncio

- **Severidade:** 🟠 alta (a ADR-035 fica sem efeito prático no artefato que o dono usa)
- **Status:** 🟢 **corrigida** (2026-11-13, `.specs/features/fix-release-onnx-provider/`) — o release e a CI compilam com a tag `onnx`, `mem embed download` traz o modelo e a DLL, o instalador invoca o comando e a degradação é explícita e consolidada. Spec com **11 ACs, 15 tarefas, 4 fases**
- **Achado em:** 2026-11-13, ao diagnosticar por que o dono "não consegue documentar aprendizados no cofre central"

**O defeito.** O provider de embedding embutido (ONNX MiniLM, ADR-035) mora atrás
de `//go:build onnx` (`internal/embedder/builtin.go:1`). Mas:

- o **release** compila com `CGO_ENABLED=1 go build -tags "sqlite_fts5"`
  (`.github/workflows/release.yml:118,131`) — **sem** a tag `onnx`;
- a **CI** não tem job com a tag `onnx` (tem o do `treesitter`,
  `codeast-cgo-on`), ou seja, **o provider nunca é compilado nem testado** no
  pipeline.

Resultado: **todo binário oficial roda em FTS lexical** e avisa
"Aviso: provider builtin indisponível: este binário foi compilado sem a tag
`onnx`; indexando em modo léxico FTS" — aviso que passa batido, porque a
busca **funciona**.

**Por que é grave.** A reindexação regrava o chunk **sem embedding** → cada nota
tocada pelo binário oficial **perde o vetor** → a busca semântica do cofre
central **decai nota a nota, em silêncio**. É exatamente "a aplicação não está
funcionando" sem erro em lugar nenhum.

**Prova medida (2026-11-13, cofre central `repo_central`).**

| Binário | Resposta da busca | Fonte RRF |
| :--- | :--- | :--- |
| build padrão (sem tag) | `Aviso: provider builtin indisponível` | `[fts:1]` |
| `CGO_ENABLED=1 go build -tags onnx` | funciona, 35.634.166 bytes | **`[vector:1]`** |

O toolchain **existe** nesta máquina: `C:\msys64\ucrt64\bin\gcc.exe` (MSYS2
ucrt64), apenas **fora do PATH** — por isso `CGO_ENABLED=0` é o default aqui.
O modelo (`~/.memory/models/paraphrase-multilingual-MiniLM-L12-v2/model_int8.onnx`,
118 MB) e a DLL do ONNX Runtime
(`~/.memory/models/onnxruntime/onnxruntime.dll`, 16 MB) **já estão no disco**.

**Correção entregue (2026-11-13, `.specs/features/fix-release-onnx-provider/`).**

(a) O release passa a compilar com a tag `onnx` **sempre** — no Linux
(`release.yml:123`) e no Windows (`release.yml:170`), com `CGO_ENABLED=1`. No
Windows o job tem um passo **"Verify C Toolchain"** que **falha com mensagem
explícita** (`::error::Toolchain C ausente: ...`) quando o runner não tem
toolchain C: nunca se publica binário mudo. (b) A CI ganhou o job
`embedder-onnx` (`ci.yml:385-413`), que compila `cmd/mem` com a tag e roda a
suíte do provider sob ela — **gate duro**, sem `continue-on-error`: um job que
não pode falhar não protege nada. (c) O instalador (ADR-032) invoca
`mem embed download` e imprime o diretório que o download **realmente usou**.

**Os artefatos que faltavam.** `internal/embedder/model_downloader.go` baixa o
modelo, o tokenizer e a biblioteca do ONNX Runtime **1.29.0** para
`~/.memory/models/`, conferindo o **tamanho** de cada arquivo antes do rename
atômico (uma falha não deixa parcial, e o erro nomeia a URL tentada e o
destino). As revisões dos artefatos são fixadas por **commit**, não por `main`:
o vetor gravado no índice depende do arquivo exato, e o Epoch cobre nome e
dimensão, não conteúdo. `mem embed doctor` diagnostica o ambiente.

**A degradação ficou explícita.** O aviso por documento
(`sem provider de embedding; <docID> indexado em modo léxico FTS`) foi
substituído por **um** aviso por execução, que nomeia a tag ausente e diz
quantos documentos ficaram sem vetor; o `mem index` sai com código **0**. A
busca híbrida declara o fallback léxico (`Resultado apenas LEXICAL — ... NÃO
usou similaridade vetorial`) e o `mem search --mode vector` sem provider
continua sendo erro de uso.

**Medido depois da correção.** O binário com a tag responde `[vector:1]` na
fonte RRF (antes: `[fts:1]`), e `mem embed doctor` reporta `Status: ATIVO`.

**Referência:** ADR-062 §C · ADR-035 (provider embutido) · ADR-032 (instalador).

---

## ISSUE-038 — O doctor conta a aresta de tag (`tagged_as`) como link morto: o Health Score do cofre central é 60/100 sem motivo

- **Severidade:** 🟡 média (medição errada que penaliza o vault mais saudável)
- **Status:** 🟢 **corrigida** (2026-11-13, `.specs/features/fix-doctor-tag-edges/`; fechada em 2026-11-14 por `.specs/features/fix-doctor-unindexed-links/`) — a tag que existe em `document_tags` deixou de ser dead link, e a parte que dependia da checagem em disco chegou na spec do doctor: um alvo que existe em disco (mesmo sendo uma tag) é **fora do índice por decisão**, nunca link rompido.
- **Achado em:** 2026-11-13, ao rodar o `mem doctor` do cofre central

**O defeito.** O check de dead links considera "vivo" só o alvo que resolve para
um **documento** (id ou title) e isenta os alvos que começam com `#`:

```sql
AND e.target_id NOT LIKE '#%'
AND e.target_id NOT IN (SELECT id FROM documents WHERE repository = $1)
AND e.target_id NOT IN (SELECT title FROM documents WHERE repository = $1)
```

Mas o caminho de **tag do frontmatter** grava o alvo da tag **sem** o prefixo
`#` (o prefixo só existe no caminho da tag inline `#tag`). Resultado: **toda**
aresta `tagged_as` cujo nome de tag não exista como documento é contada como
link morto.

**Prova medida (cofre central `repo_central`, 2026-11-13).**

| Origem | Count | Realidade |
| :--- | :--- | :--- |
| `tagged_as` | **80** | **falso positivo** — o alvo é uma tag ou uma referência a nota; **10** delas têm a tag em `document_tags` (corrigido em 2026-11-13) e as outras **70** apontam para material fora do índice → ISSUE-039 |
| `links_to` | **68** | material bruto fora do índice (existe no disco) → ISSUE-039 |
| **Total reportado pelo doctor** | **148** | Health Score **60/100** (a penalidade de dead link tem teto de 40 pontos) |

Sem os 148, o cofre sairia em **100/100** — o valor compatível com um vault com 0
órfãos.

**Correção entregue (2026-11-13, `.specs/features/fix-doctor-tag-edges/`).** O
check de dead link agora conhece a tag: o alvo está vivo se for um documento
(id/título), uma **tag conhecida** (`document_tags`, por `tag` ou `tag_key`) ou um
**símbolo de código** (`graph_nodes` com `type='code_symbol'`). Medido: cofre
central **148 → 138** dead links; vault do repo **2** (os dois `[[ADR-035]]`
reais, com Health Score 90/100).

> ⚠️ **Um contra-teste que mudou a solução:** a primeira versão aceitava
> "qualquer nó do grafo" como vivo — e o teste **passou**. Era errado: o grafo
> cria **nós-placeholder** para todo link citado (no cofre central, 238 nós
> para 94 documentos), então a regra zerava o contador de dead links **e**
> mascarava links realmente quebrados. A regra final não aceita placeholder.

**O que falta (ISSUE-039).** Os **138** restantes são referências a material
que **existe no disco** e está fora do índice por decisão — a distinção que o
dono definiu ("fora do índice por decisão" ≠ "link rompido") exige a checagem em
disco, e o `doctor` precisa do caminho do vault.

**Referência:** ADR-062 §D-2.

---

## ISSUE-039 — 125 dos 138 links do cofre central apontam para material que existe em disco mas está fora do índice

- **Severidade:** 🟡 média (o vault parece quebrado no Obsidian e no grafo ao mesmo tempo)
- **Status:** 🟢 **corrigida** (2026-11-14, `.specs/features/fix-doctor-unindexed-links/`) — o doctor confere cada link suspeito **contra o disco do vault** e classifica em **três** categorias: **rompido** (penaliza o score e a `--fix` apaga), **fora do índice por decisão** (não penaliza e a `--fix` preserva) e **não verificável** (sem raiz legível: penaliza como link morto e marca o score como **provisório**). Medido no `repo_central`: **125** na categoria nova e **13** rompidos; o `--fix` passou de 138 para 13 arestas. Spec com **9 ACs, 9 tarefas, 4 fases**.
- **Achado em:** 2026-11-13, ao decompor os 148 "dead links" do cofre central

**O fato.** O `config.yaml` do cofre exclui do índice o material bruto
(`catalog-skills/skills/**`, 704 arquivos) por decisão registrada: *"Material
BRUTO das skills: é FONTE para destilação, não conhecimento. Sem esta exclusão,
os ~700 arquivos de references/rules em inglês competem na busca semântica"* (o
Health Score caiu de 100 para 57 quando isso aconteceu, em 2026-10-17).

Mas as notas do catálogo **linkam** esse material (`[[SKILL]]`,
`[[review-standards]]`, `[[mcp-guide]]`…). No Obsidian esses links resolvem (os
arquivos existem no disco); no **índice** eles não (os alvos não são
documentos indexados) → o doctor os conta como links mortos.

**Prova medida (corrigida em 2026-11-14 — a medição original estava errada).**
A hipótese original era "os 138 dead links restantes apontam para caminhos que
existem em disco". **Não é verdade.** Medido no Postgres real (`repo_central`,
dados de 2026-11-14), os **138** se dividem em:

| Classe | Arestas | Como o alvo resolve | Onde vive o alvo |
| :--- | ---: | :--- | :--- |
| Existe em disco, fora do índice por decisão | **125** | 112 por basename · **13 por `title:` do frontmatter** | **todas** em `catalog-skills/skills/**` (a área excluída) |
| **Não existe em disco** | **13** | — | 2 notas de origem, todas `links_to` |

Por relação: **63** `links_to` + **75** `tagged_as` no total; dos 125 que
existem em disco, 50 são `links_to` e 75 são `tagged_as`.

Os **13 que não existem** são **conceitos sem nota** (`Docker`, `Kubernetes`,
`Rollback`, `Observabilidade`, `Banco de Dados`, `Busca Vetorial e RAG`,
`SRE e SLOs`, `Transações e Concorrência`, `Índices SQL`,
`Backup e Recuperação`, `Filas e Processamento Assíncrono`,
`Observabilidade de Banco de Dados`, `Expand and Contract`), todos linkados
por apenas 2 notas: `cloud/blue-green-deployment-obsidian.md` e
`database/PostgreSQL — Visão Geral de Recursos e Ferramentas.md`. São
**links rompidos de verdade** e continuam penalizando o Health Score.

**Duas consequências que a medição mudou:**

1. **O Health Score real é 45/100**, não 60/100 — os 138 dead links já
   saturavam a penalidade (teto de 40) e os 101 chunks sem vetor saturavam a
   outra (teto de 15): `100 - 40 - 15 = 45`.
2. **O `--fix` de hoje apagaria 138 arestas, das quais 125 apontam para
   material que existe em disco** — ele apagaria links vivos. Esse é o defeito
   que a spec corrige: a rotina passa a apagar só o que o relatório provou
   rompido (13).

**Semântica de resolução (medida).** 13 dos 125 resolvem **só** pelo `title:`
do frontmatter — a identidade de um documento no índice é
`documents.title` (`COALESCE(frontmatter.title, basename)`, ISSUE-010), e o
indexador reescreve wikilink/tag contra os **títulos** indexados
(`parser.ResolveTagConnections`). Sem o `title:` na checagem em disco, esses
13 seriam classificados como rompidos e o `--fix` os apagaria.

**O que falta (o escopo desta issue).** Implementar a checagem em disco no
`doctor`: distinguir "rompido" / "fora do índice por decisão" / "não
verificável". A correção de **conteúdo** dos 13 conceitos sem nota (criar nota
stub ou remover o wikilink) é decisão do dono e **não** é escopo da spec — sem
ela, o critério "≥ 90/100" da ADR-062 §B-2 não fecha (o score honesto do
cofre é **60/100** depois do reindex com provider).

> **✅ FECHADA (2026-11-14).** A checagem em disco foi entregue pela spec
> `fix-doctor-unindexed-links` (ADR-063) e a decisão de **conteúdo** foi tomada
> pelo dono: **criar 13 notas stub, uma por conceito**. Executado na spec
> `central-vault-concept-notes` (ADR-064): as 13 notas existem, com o nome do
> arquivo igual ao texto do wikilink, e o cofre central foi reindexado com o
> provider ONNX. **Medido:** `mem doctor` → **Health Score 100/100**, **0 link
> rompido**, 0 órfã, 0 chunk sem vetor real. **O critério "≥ 90/100" da
> ADR-062 §B-2 fecha.**

**As três saídas possíveis (decisão do dono).**

1. **Resolver contra o disco** no check de dead link: o alvo existe como
   arquivo → não é link morto. Hmm: o doctor passaria a depender do filesystem
   do vault (que em Postgres pode não estar no mesmo host).
2. **Aceitar como "não indexado por decisão"**: o doctor distingue
   "rompido" de "fora do índice por política" — hmm, e o Health Score para de
   punir um link que funciona no Obsidian.
3. **Remover os links** do material bruto das notas do catálogo (o catálogo
   aponta para a skill pelo **nome**, não por `[[wikilink]]`).

**Nota.** As três opções mexem em **conteúdo** ou em **semântica de medição** —
não é bug de código puro. Hmm: a opção 1 e a 2 sãochanges no doctor (código);
a 3 é uma edição de conteúdo do cofre.

### Resolução (2026-11-14)

**O que foi corrigido** (spec `fix-doctor-unindexed-links`, 9 tarefas, 4 fases, 14 commits em `58c8c0e..22dd4fe`):

| O que | Onde | Como |
| :--- | :--- | :--- |
| A raiz do vault | `internal/vaultfs/root.go` | flag `--vault` › **catálogo do repositório auditado** › `central_vault.path` (só o cofre central) › diretório atual; e diz se a raiz é **acessível** |
| O alvo do wikilink | `internal/vaultfs/target.go` | basename em qualquer profundidade, caminho relativo (com/sem `.md`), `title:` e `alias:` — match **exato**, nunca fuzzy — devolvendo **como** casou |
| A classificação | `internal/store/doctor_links.go` | classificador **compartilhado** (`ClassifyLinks` + `NewVaultResolver`): rompido / fora do índice por decisão / não verificável |
| O Health Score | `internal/store/doctor.go` | a categoria nova **não** penaliza; a não verificável penaliza e marca o score **provisório** |
| A rotina de correção | `internal/db/graph.go` · `internal/store/doctor.go` | apaga **só** a lista do relatório (no Postgres, um statement com `unnest`) |
| A superfície | `cmd/mem/main.go` · `internal/mcp/handlers.go` | `mem doctor --vault` e as três categorias no payload do `memory_doctor` |

**A regra que fecha a ISSUE-038**: o relatório e a rotina de correção consomem a
**mesma** lista — o classificador é o único que decide o que é link rompido.

**Verificação**: Verifier independente em duas rodadas (a 1ª achou 3 defeitos
reais — o gate vermelho, o `gofmt` quebrado e um número stale na spec — e a 2ª
confirmou os dois gates verdes e a medição real), sensor de 6 mutações
**6/6 mortas**, gate full (1325 testes) e gate taggeado ONNX + Postgres (1337
testes) verdes. Relatório: `.specs/features/fix-doctor-unindexed-links/validation.md`.

**O que NÃO fecha**: os **13 conceitos sem nota** continuam rompidos — a correção
de **conteúdo** (criar 13 notas stub ou remover os 13 wikilinks) é decisão do
dono. O score honesto do cofre é **60/100** depois do reindex com o provider, e o
critério **≥ 90/100** da ADR-062 §B-2 depende dessa escolha.

**Referência:** ADR-062 §D-3.

---

## ISSUE-040 — O chunk indexado sem provider grava vetor zero: ninguém detecta e a busca vetorial devolve ruído

- **Severidade:** 🟠 alta (a degradação é **invisível**: o doctor não vê, o score não cai, e a busca "responde")
- **Status:** 🟢 **corrigida** (2026-11-14, `.specs/features/fix-zero-vector-chunks/`) — o chunk sem provider é gravado **sem vetor** (não vetor zero), o doctor passa a ver o vetor zero **e** a ausência nos **dois** caminhos de armazenamento, a busca vetorial exclui o chunk sem vetor real, e há prova E2E de recuperação **por sentido** e de reindex do zero. Spec com **7 ACs, 8 tarefas, 3 fases**
- **Achado em:** 2026-11-13, ao ler o caminho de indexação do `cmd/mem/main.go` depois de escrever as specs do Vault Central

**O defeito.** Quando não há provider de embedding, o indexador **não deixa o
chunk sem vetor**: ele substitui por um vetor **zero** de 384 dimensões
(`cmd/mem/main.go:1913` no caminho Postgres, e o bloco equivalente no SQLite).
Três consequências, todas silenciosas:

1. **O doctor não vê.** O check de chunks dessincronizados procura
   `embedding IS NULL` (`internal/store/doctor.go:176`) — vetor zero **não** é
   NULL. O Health Score do cofre não se move, e o dono lê "vault saudável"
   sobre um índice sem um único vetor real.
2. **A busca vetorial devolve ruído com cara de resultado.** Vetor zero numa
   coluna `vector(384)` (pgvector) deixa a distância de cosseno indefinida
   (NaN): a ordenação dos hits deixa de distinguir relevante de irrelevante, e
   a resposta tem a aparência de uma busca normal.
3. **O SQLite já sabia fazer certo.** `internal/db/store.go:357` só insere em
   `chunks_vec` quando `len(vec) > 0` — o caminho "chunk sem vetor" **já
   existia**. Foi o substituto zero no `cmd/mem/main.go` que estragou os dois
   engines.

**Por que isso importa agora.** A ADR-062 mediu "os 481 chunks têm vetor" com o
binário **local** compilado com `-tags onnx`. O binário oficial (sem a tag)
grava zeros: o mesmo cofre, indexado pelo artefato distribuído, fica com 481
chunks de vetor zero — e nada no produto diz isso.

**Correção proposta.** (a) o indexador para de substituir o vetor zero e grava
o chunk **sem embedding** (NULL no Postgres — a coluna `embedding vector(384)`
é nullable, `internal/store/postgres.go:53`; nenhuma linha em `chunks_vec` no
SQLite); (b) o doctor passa a tratar vetor zero como chunk dessincronizado,
além do NULL; (c) a busca vetorial deixa de devolver chunk sem vetor real;
(d) prova E2E de que a nota do cofre volta **por sentido** e de que o reindex
do zero não perde vetor.

**Nota.** A correção (a) muda o **dado gravado**: chunks já gravados com vetor
zero ficam assim até um `mem index --force` com provider presente. O doctor
passa a listá-los — que é exatamente o ponto.

**Referência:** ADR-062 §C e §B-4 · ADR-013 (penalidade de chunk dessincronizado) · ISSUE-037.

### Resolução (2026-11-14)

**O que foi corrigido** (spec `fix-zero-vector-chunks`, 8 tarefas, 3 fases, 9 commits de código em `d523d91..d9c5ebb`):

| O que | Onde | Como |
| :--- | :--- | :--- |
| O indexador para de substituir o vetor zero | `cmd/mem/main.go` (Postgres **e** SQLite) | o chunk sem provider é gravado **sem embedding**: `embedding` NULL no Postgres, **nenhuma** linha em `chunks_vec` **nem** em `chunks_turboquant` no SQLite |
| O doctor passa a ver o vetor zero | `internal/store/doctor.go` · `internal/db/graph.go` | o Postgres compara com `v <#> v = -||v||²` (zero só quando o vetor É o vetor zero); o SQLite compara o BLOB cru e a assinatura comprimida (todo nibble 7) |
| A busca vetorial exclui o chunk sem vetor real | `internal/store/postgres.go` · `internal/db/graph.go` | os **três** caminhos de leitura (Postgres, SQLite cru, SQLite comprimido) |
| Prova E2E | `cmd/mem/vault_retrieval_test.go` | a nota escrita por `mem lesson` volta na busca por uma **paráfrase sem palavra em comum**, pela fonte `vector`; e o reindex do zero deixa **0** chunks sem vetor real |

**A dimensão que a spec emendou**: o SQLite tem **dois** caminhos de vetor e
o Postgres **um** — e a busca vetorial do Windows lê o caminho **comprimido**
(`chunks_turboquant`), não o cru. Zerar a coluna `chunks.vec` não matava a
busca vetorial: por isso a spec cobre os dois caminhos.

**O dado já gravado**: chunks com vetor zero continuam assim até um
`mem index --force` com provider presente — e, a partir de agora, o doctor
**os lista** (é o ponto). O reparo é o mesmo caminho que a ADR-062 mediu.

**Verificação**: 5 sensores de mutação (16 mutações, 14 mortas + 2 mutantes
equivalentes registrados), gate full e gate taggeado verdes, e as duas provas
E2E rodando com o **provider ONNX real**. Relatório:
`.specs/features/fix-zero-vector-chunks/validation.md`.
---

## ISSUE-041 — A suite de `cmd/mem` sobrescreve `bin/mem.exe` com um build SEM a tag `onnx`: o proximo `mem index` grava chunk sem vetor

- **Severidade:** 🟡 media (o estrago e **anunciado** — o `mem index` imprime o aviso consolidado e o doctor lista os chunks — mas a suite **derruba o artefato que o dono usa**, sem ter obrigação sobre isso)
- **Status:** 🔵 **open — spec escrita** (2026-11-14, `.specs/features/fix-test-binary-clobber/`) — 3 ACs, 7 requisitos, 3 tarefas, 2 fases; o conserto (helper compilando num temp dir + prova de nao-regressao) ainda **nao** foi implementado. Reparo **provado** no cofre (rebuild taggeado + reindex), mas o defeito do codigo continua ate a T2 da spec.
- **Achado em:** 2026-11-14, durante a medicao do "atualizar so o conteudo que mudou" (spec `central-vault-concept-notes`)

**O defeito.** Dois helpers de teste de `cmd/mem` compilam o binario **por cima do
binario do usuario**, sem a tag `onnx`:

- `cmd/mem/doctor_test.go:111` — `exec.Command("go", "build", "-o", path, "./cmd/mem")`
- `cmd/mem/events_test.go:40` — idem

O caminho e `<repoRoot>/bin/mem.exe` (`doctor_test.go:108`, `events_test.go:37`).
O comentario dos testes diz *"Always rebuild to avoid stale-binary issues"* — mas
o problema de binario velho se resolve compilando **novo num temp dir**, nao
sobrescrevendo o artefato que o dono usa.

**A cadeia medida (Cofre Central, 2026-11-14):**

1. `bin/mem.exe` foi compilado com `-tags "sqlite_fts5 onnx"` (37.191.134 bytes) e
   o `mem embed doctor` reportava `Status: ATIVO`.
2. A suite completa rodou as 16:23 e sobrescreveu o binario por um build sem a
   tag (26.481.664 bytes). O `mem embed doctor` passou a dizer
   `Status: INDISPONIVEL - este binario foi compilado sem a tag 'onnx'`.
3. O `mem index` seguinte rodou em **modo lexico**: os chunks da nota alterada
   foram gravados **sem embedding** (ISSUE-040: sem provider, o chunk e gravado
   sem vetor — nunca vetor zero).
4. O `mem doctor` reportou **96/100** (2 chunks sem vetor real = 4 pontos).
   **Correcao de um registro anterior deste issue:** a degradacao **nao** e
   silenciosa — o `mem index` imprime o aviso consolidado do ISSUE-037
   (`cmd/mem/main.go:1503`, que nomeia a tag ausente e a causa probable) e o
   doctor lista os chunks sem vetor real. O que faltava era o binario do
   usuario deixar de ser efeito colateral da suite.

**Por que o epoch nao salvou.** O gate do cache incremental exige hash **e**
epoch iguais (`internal/db/store.go:151-170`). No modo lexico o epoch gravado e
`""`; quando o provider volta, `"" != "paraphrase-multilingual-MiniLM-L12-v2:384:v1"`
e o documento **nao** passa no gate — ou seja, o epoch **se auto-cura**: provado
neste caso (1 documento reprocessado, 107 em cache, vetor de volta, score 100).
O que falta e evitar o estrago: a suite nao deve trocar o binario do dono.

**Correcao proposta.**
(a) Os helpers de teste compilam num **temp dir** (`t.TempDir()`) e nunca em
`bin/` — o binario do usuario deixa de ser efeito colateral da suite. **Este e o
escopo da spec `fix-test-binary-clobber`.**
(b) ~~O `mem index` com o provider ausente avisa por execução~~ — **ja existe**
(ISSUE-037, `cmd/mem/main.go:1503`), com teste proprio em
`cmd/mem/embed_provider_degradation_test.go`. Mantido como contrato a preservar.
(c) ~~O `mem embed doctor` distingue "sem tag" de "artefato faltando"~~ — **ja
existe**: o stub sem tag responde *"este binário foi compilado sem a tag
`onnx`"* e o provider real responde que o modelo nao foi encontrado.

**Referência:** ADR-035 (o provider atras da tag) · ADR-062 §B-4 (o cofre com
vetor) · ISSUE-037 (binario oficial sem provider) · ISSUE-040 (chunk sem vetor
real) · ADR-063 (as tres categorias de link).

### Resolução (2026-11-14)

**O que foi corrigido** (spec `fix-test-binary-clobber`, 3 tarefas, 2 fases, 3 commits):

| O que | Onde | Como |
| :--- | :--- | :--- |
| A suite nao compila para dentro de `bin/` | `cmd/mem/events_test.go` (`memBinary`) · `cmd/mem/doctor_test.go` (`runMemDoctor`) | helper **unico** que compila em `t.TempDir()`; o `doctor_test.go` passou a consumi-lo (nao ha mais um segundo `go build`) |
| A regressao nao volta | `cmd/mem/test_binary_untouched_test.go` | 3 testes: o helper fora de `bin/`, o SHA-256 de `bin/mem.exe` intacto antes e depois da suite, e um **guard estatico** que aponta qualquer fonte de teste que volte a compilar para dentro de `bin/` |

**Medido**: a suite `cmd/mem` **inteira** rodou (exit 0, 123,8 s) e o
`bin/mem.exe` do dono ficou **byte a byte** igual — hash
`2336f7e84273ceaa9ee3285ad9d77cd9701f01693bbcb871c70cbb751df30e60`,
37.191.134 bytes, antes e depois. Antes da correcao, a mesma suite deixava o
binario em 26.481.664 bytes (build sem a tag).

**O contrato do aviso consolidado (ISSUE-037) nao regrediu**: o `mem index` sem
provider continua imprimindo o aviso uma vez por execucao
(`cmd/mem/main.go:1503`), coberto por
`cmd/mem/embed_provider_degradation_test.go`, e o gate taggeado prova que com
o provider presente o aviso nao aparece.

**Referência da spec:** `.specs/features/fix-test-binary-clobber/`.
---

## ISSUE-042 — 14 funcoes sem chamada no repo: a fase 2 da ADR-065 as remove

- **Severidade:** 🟢 baixa (codigo morto: nao custa runtime, custa leitura e manutencao)
- **Status:** 🟢 **corrigida** (2026-11-14, `.specs/features/prune-dead-code-2026-11/`) — 14 blocos removidos, **223 linhas**; `deadcode` com a tag do release passou de 17 para **3** (as do `Subscriber`, que sao chamadas via interface). Spec: 8 ACs, 8 requisitos, 2 tarefas, 2 fases.
- **Achado em:** 2026-11-14, no inventario read-only da ADR-065 (fase 1) — relatorio em `.specs/INVENTARIO-2026-11-14.md`

**A correcao do metodo (o que quase deu errado).** O `deadcode` e **indicador**,
nao veredito, e erra nas duas pontas:

| Medida | Resultado | Por que |
| :--- | :---: | :--- |
| `deadcode -test` **sem** a tag `onnx` | 19 | o provider so existe na build taggeada: `NewTokenizerFromDir` e `Tokenizer.Close` aparecem como mortos |
| `deadcode -test` **com** a tag do release | 17 | some os 2 falso-positivos do provider |
| **prova de chamada** (busca de chamada real em `.go`, fora do arquivo de definicao) | **14** | `AuditSubscriber.Name`, `.EventTypes` e `.MaxAckPending` sao chamadas **via a interface `Subscriber`** — o `deadcode` nao ve dispatch por interface |

O repo **nao usa reflexao** (nenhum `reflect.ValueOf` / `MethodByName`), e uma
chamada por interface aparece como `.Metodo(` no codigo — e por isso que a prova
salvou as 3.

**Os 14 cortes (cada um com a prova):**

| # | `arquivo:linha` | Funcao removida | Prova |
| :--: | :--- | :--- | :--- |
| 1 | `internal/codeast/parser.go:40` | `ValidKinds` | nenhuma chamada em `.go` |
| 2 | `internal/codeast/parser.go:272` | `IsCamelOrPascalSegment` | nenhuma chamada em `.go` |
| 3 | `internal/db/graph.go:119` | `SearchTurboQuant` | wrapper de conveniencia; o caminho vivo e `SearchTurboQuantWithOptions` |
| 4 | `internal/embedder/model_downloader.go:233` | `DownloadProviderArtifacts` | o `mem embed download` chama `DownloadArtifact` em loop |
| 5 | `internal/event_runtime/audit_subscriber.go:129` | `auditLogPathForTesting` | nenhuma chamada (nem dos testes) |
| 6 | `internal/event_runtime/dispatcher.go:61` | `Dispatcher.IsReady` | nenhuma chamada |
| 7 | `internal/event_runtime/log.go:97` | `Log.ReadRange` | nenhuma chamada (66 linhas) |
| 8 | `internal/federation/search.go:289` | `ReplaceDatabaseInPostgresURL` | nenhuma chamada |
| 9 | `internal/graph/inspector.go:85` | `DefaultInspectorOptions` | nenhuma chamada |
| 10 | `internal/mcp/handlers.go:545` | `NewDBNeighborsHandler` | a tool viva usa `NewMemoryNeighborsHandler` |
| 11 | `internal/mcp/server.go:158` | `Server.StalenessDetector` | o setter e usado; o getter nao |
| 12 | `internal/supervisor/audit.go:250` | `AuditLogger.CurrentSize` | nenhuma chamada |
| 13 | `internal/supervisor/audit.go:259` | `AuditLogger.CurrentPath` | nenhuma chamada |
| 14 | `internal/vaultfs/target.go:127` | `Resolver.Root` | nenhuma chamada (metodo criado na spec `fix-doctor-unindexed-links`, 2026-11-14) |

**O que NAO foi podado (e por que):**

| Item | Motivo |
| :--- | :--- |
| 6 ADRs obsoletas (949 linhas) | sao o **registro das decisoes revertidas** — a regra *supersede-not-delete* da ADR-062 as preserva de proposito |
| 5 `validation.md` que sao plano | divida de **verificacao** (o codigo existe, o Verifier nunca rodou), nao de corte |
| 6 specs planejadas | roadmap: a decisao e do dono (ADR-065 §7b) |
| 3 funcoes do `Subscriber` | **vivas** — chamadas via interface |
| `NewTokenizerFromDir`, `Tokenizer.Close` | **vivas** — chamadas pelo provider na build taggeada |

**Verificacao:** `go build ./...` e `go vet ./...` exit 0; `deadcode -test -tags
"sqlite_fts5 onnx" ./...` reporta **3** (as do `Subscriber`); gate completo e
gate taggeado verdes; o Cofre Central continua em **100/100** (0 link rompido,
0 chunk sem vetor real).

**Referencia:** ADR-065 (fase 2) · ADR-050 (LLM02/auditoria) · ISSUE-036 (o
codigo graph que o `--fix` apagava por engano — corte errado e custo alto) ·
`.specs/INVENTARIO-2026-11-14.md`.

## ISSUE-043 — O banner de staleness nunca aparece nos comandos de consulta: `FormatCLIBanner` é código morto

- **Severidade:** 🟠 high (a feature entrega o motor e deixa de fora o gancho que a torna visível — o usuário não é avisado de que o índice está desatualizado)
- **Status:** 🔵 **open** (achado pela verificação independente de 2026-11-15; `tasks.md` marca T3 como `[x]` e o checkbox mente)
- **Achado em:** 2026-11-15, na verificação da feature `staleness-detection-and-banners` — `.specs/features/staleness-detection-and-banners/validation.md`

**O defeito.** `FormatCLIBanner` (`internal/staleness/detector.go:309`) foi
implementado e tem teste unitário, mas **nenhum comando de consulta o chama**.
A única construção de `staleness.Detector` na CLI é a de `cmd/mem/status.go:75-90`.
`cmd/mem/main.go:203` (`search`), `cmd/mem/path.go`, `cmd/mem/inspect.go` e
`cmd/mem/impact.go` nunca constroem o detector — logo, o aviso de desatualização
**não aparece** em nenhum deles.

**Os 4 gaps (ranqueados pelo Verifier):**

| # | ID | Sev. | Gap | Onde |
| :--: | :--- | :--- | :--- | :--- |
| 1 | P3-1 | 🔴 Blocker | o aviso não sai em `search`/`path`/`inspect`/`impact`; `FormatCLIBanner` é código morto | `internal/staleness/detector.go:309`, `cmd/mem/status.go:75-90` |
| 2 | P1-1 | 🟠 Major | o literal emitido (`> ⚠️ **AVISO: Memória Desatualizada (Stale Data)**`) **não** é o do spec (`spec.md:48`) nem o da ADR-028 Accepted (`docs/adr/028-...md:45`); a documentação foi reescrita para acompanhar o código (`docs/AGENT_INTEGRATION_GUIDE.md:397`) | `internal/staleness/detector.go:302` |
| 3 | P2-2 | 🟠 Major | a listagem de arquivos modificados/novos/deletados não tem asserção, e "novo" não é distinguível de "modificado" no modelo (`StalenessReport` só tem `StaleFiles` e `DeletedFiles`) | `internal/staleness/detector.go:25-34,227,245`, `cmd/mem/status.go:133` |
| 4 | P2-3/P2-4 | 🟡 Minor | literais de status divergem do spec (`⚠️ DESATUALIZADO (Stale)` / `✅ ATUALIZADO (Fresh)` vs `⚠️  Desatualizado (Stale Data)` / `✅ Atualizado (In Sync)`) | `cmd/mem/status.go:124,129` |

**Sensor:** 6/6 mutantes mortos — o motor de detecção é sólido. Os gaps são de
**gancho e de literal**, não de lógica.

**O que a correção precisa decidir (não é implementação, é escolha):**
1. Um **literal canônico** do banner — alinhar o código ao ADR-028 (que cita
   `mem watch`) ou emendar a ADR-028 com a tradução para PT-BR. O teste MCP
   (`internal/mcp/staleness_handlers_test.go:94`) deve asseverar o literal
   **canônico**, não o acidental.
2. Se "novos" é categoria própria (`NewFiles` + heading próprio) ou se o
   requisito sai do spec.

**Referência:** ADR-028 · spec `staleness-detection-and-banners` ·
`.specs/features/staleness-detection-and-banners/validation.md` (§ Fix Plans).

## ISSUE-044 — 3 caminhos do carregamento progressivo não têm teste: o default da flag, o schema da tool e o indexador

- **Severidade:** 🟡 medium (o comportamento existe e o gate está verde; falta a prova que liga as unidades — o padrão é "testado na unidade, encano sem prova")
- **Status:** 🔵 **open** (achado pela verificação independente de 2026-11-15)
- **Achado em:** 2026-11-15, na verificação da feature `progressive-context-loading` — `.specs/features/progressive-context-loading/validation.md`

**Os 3 gaps:**

| ID | O que não tem teste | Onde |
| :--- | :--- | :--- |
| PCL-01 | a renderização **L0** pela CLI — o teste pega o default da **flag**, não a **saída** | `cmd/mem/` |
| PCL-02 | o **schema JSON** da tool MCP — o teste pega o **comportamento**, não o **schema** | `internal/mcp/` |
| PCL-03 | o indexador `summary` → `abstract` — o teste pega o **parser**, não o **indexador** | indexador |

**Veredito:** ⚠️ **PARTIAL** — 8/9 ACs com evidência `file:line`; o comportamento
está implementado nos dois motores (taxonomia `category`, `abstract` L0, filtro
por categoria na busca, projeção L0 sem corpo bruto, tool serializando por nível).

**Sensor:** 6 mutações, **3 mortas / 3 sobreviveram** — as 3 que sobreviveram
são exatamente as 3 bordas sem teste acima.

**Ressalva de independência:** o sensor foi executado **no pai**, porque o Verifier
independente encerrou o turno antes dele. O gate e a análise ancorada na spec
foram do sub-agente. A re-verificação independente da spec continua pendente.

**Referência:** ADR-025 · spec `progressive-context-loading` ·
`.specs/features/progressive-context-loading/validation.md`.

## ISSUE-045 — O `mem pack` estoura o orçamento de tokens e a ordem de prioridade não tem resultado definido

- **Severidade:** 🟠 high (o bundle gravado tem **1370 tokens** contra um orçamento de **1000** — a restrição que a ADR-029 e o `docs/CLI_GUIDE.md` §21 prometem é violada em uso real)
- **Status:** 🔵 **open** (achado pela verificação independente de 2026-11-15)
- **Achado em:** 2026-11-15, na verificação da feature `context-packager-and-subgraph-bundle` — `.specs/features/context-packager-and-subgraph-bundle/validation.md`

**Os 7 gaps (ranqueados):**

| # | Gap | Detalhe |
| :--: | :--- | :--- |
| 1 | o bundle emitido **estoura `max_tokens`** | medido **1370** tokens contra orçamento de 1000; a contabilidade ignora o bloco Mermaid, o cabeçalho e os metadados por nó (R2, ADR-029:22, `docs/CLI_GUIDE.md` §21) |
| 2 | o filtro de direção de R1 **não tem teste discriminante** | mutante M3 sobrevive à suíte inteira |
| 3 | a **ordem de prioridade não tem resultado definido** no spec e não é testada | mutante M4 sobrevive (SP2) |
| 4 | o **peso epistêmico é carregado e descartado** | T1 e a ADR-029 superestimam o que o código faz |
| 5 | o tratamento preferencial da raiz e os headers R3.3/R3.4 **não são asseverados** | mutante M2 sobrevive |
| 6 | **código morto e defaults duplicados** | `availableBudget` (`internal/graph/pack.go:221-224`) é calculado e nunca lido; `nodeIDs`; `graph.DefaultPackOptions()` só é usado em teste |
| 7 | a CLI reporta os **tokens contabilizados**, não o tamanho do bundle que gravou | `mem pack` diz um número e escreve outro |

**Sensor:** 3/6 mutantes mortos (M1, M5, M6). M2, M3 e M4 sobreviveram à suíte
inteira — nenhum teste os distingue.

**O que funciona:** a expansão do subgrafo, a degradação em 3 níveis
(core / fringe / omitted), a topologia Mermaid, a CLI (`mem pack` com `--out` /
`--json`), a tool MCP com entrada no catálogo de JSON-Schema, o banner de
staleness no server, as duas camadas de persistência, a ADR e a documentação
operacional. O motor é determinístico, local e sem dependência externa, como o
spec exige.

**Referência:** ADR-029 · spec `context-packager-and-subgraph-bundle` ·
`.specs/features/context-packager-and-subgraph-bundle/validation.md`.

## ISSUE-046 — 7 gaps na feature `graph-path-discovery`: resolução ambígua, defaults mortos e ausência de asserção

- **Severidade:** 🟡 medium (o motor de caminho está correto; os gaps são de gancho, de default e de prova — a feature é inutilizável no vault real por causa do ISSUE-047)
- **Status:** 🔵 **open** (achado pela verificação independente de 2026-11-15)
- **Achado em:** 2026-11-15, na verificação da feature `graph-path-discovery` — `.specs/features/graph-path-discovery/validation.md`

**Os 7 gaps (ranqueados):**

| # | ID | Gap |
| :--: | :--- | :--- |
| 1 | FIX-2 | **falha ponta a ponta em dado real** — ver ISSUE-047 (defeito de produto, causa fora do diff da feature) |
| 2 | FIX-3 | a resolução de nó é **ambígua e arbitrária**: `LIMIT 1` sem `ORDER BY` sobre 7 nós de mesmo nome; `"README"` resolveu silenciosamente para o README de um skill |
| 3 | FIX-1 | mutante M7 **sobreviveu**: o ramo de nó idêntico é redundante e sem guarda |
| 4 | FIX-4 | `EpistemicStatus` e `Weight` das arestas devolvidas **nunca são asseverados** |
| 5 | FIX-5 | o caminho PostgreSQL só é coberto por um self-path; o **isolamento por repositório nunca é asseverado** |
| 6 | FIX-6 | a mensagem de não-encontrado, o bloco CLI de not-found, os headings de rota/tabela MCP e o código `InvalidParams` **não são asseverados** |
| 7 | FIX-7 | `DefaultPathOptions()` é **morto em produção**; os defaults são duplicados em outros dois lugares |

**Spec precision:** 7 gaps (S1–S7), incluindo a promessa P3 "Markdown e JSON" que
a resposta só-texto não cumpre (já rastreada pela ISSUE-022).

**O que funciona:** o motor Dijkstra está correto; o *pruning* por dominância
consciente de profundidade é sólido (um estado só é descartado quando já existe
um estado com `depth ≤ d` **e** `cost ≤ c`); o modo `hops` é exatamente
equivalente a BFS; ciclos terminam; e a inversão epistêmica `c = 1/w` se
comporta como a ADR-027 especifica (`EXTRACTED` 1.0, `INFERRED` ≈1.67, `TAG`
≈3.33). A CLI está completa (flags, rota ASCII, tabela de segmentos, JSON,
dois motores) e foi verificada ao vivo. A tool MCP tem JSON Schema formal
(`required: [source, target]`), aparece em `tools/list` e está ligada nos
transportes stdio e HTTP/SSE pelo mesmo `runMCPServer`.

**Referência:** ADR-027 · spec `graph-path-discovery` ·
`.specs/features/graph-path-discovery/validation.md`.

## ISSUE-047 — `mem path` não funciona no vault real: o indexador dá duas identidades a cada nota

- **Severidade:** 🔴 critical (a feature `graph-path-discovery` está entregue e verificada, e ainda assim o comando principal dela **não acha caminho nenhum** entre duas notas ligadas por wikilink)
- **Status:** 🔵 **open** (achado pela verificação independente de 2026-11-15; a causa é **anterior** à feature)
- **Achado em:** 2026-11-15, na verificação da feature `graph-path-discovery` — `.specs/features/graph-path-discovery/validation.md` (FIX-2)

**O defeito.** O indexador dá **duas identidades** a cada nota: a **origem** da
aresta é gravada com o **id de path** do documento, e o **destino** com o **id de
título**. Toda aresta atravessa essa fenda. `ResolveNodeCanonicalID` cai
justamente no lado que **nenhuma** aresta conecta.

**Medido no vault real (`repo_central`):** **0 de 40.243 arestas** têm as duas
pontas em id de path. `mem path "CONTRIBUTING.md" "README"` devolve
**"Não encontrado"**. O `README` tem grau **3** por id de path e **66** por id
de título — o mesmo nó, dois graus.

**A causa está fora do diff da feature** (commit `171df1e`, um dia antes do
commit da feature). O motor de caminho **não precisa de mudança** — ele está
correto (ver ISSUE-046). O que precisa consertar é a **identidade do nó** no
grafo, o que afeta qualquer consumidor que resolva nó por id canônico.

**Por que isso é uma issue separada:** não é um gap da feature — é um defeito de
produto que precisa de **spec própria**, porque a correção toca o indexador (a
gravação da aresta) e o resolvedor canônico, e a decisão "qual identidade é a
canônica" é de arquitetura.

**Referência:** ADR-027 · ADR-040 (identidade de repositório) ·
spec `graph-path-discovery` (FIX-2) ·
`.specs/features/graph-path-discovery/validation.md`.

## ISSUE-048 — O auto-wiring promete comandos e detecção que o HEAD não tem: `mem setup` não é alias, e o `mem install` cria config para ferramentas ausentes

- **Severidade:** 🔴 critical (o comando é **documentado** e **não faz o que a doc promete** — e falha em silêncio, com exit 0)
- **Status:** 🔵 **open** (achado pela verificação independente de 2026-11-15)
- **Achado em:** 2026-11-15, na verificação da feature `ai-tool-auto-wiring` — `.specs/features/ai-tool-auto-wiring/validation.md`

**Veredito:** ❌ **FAIL** — 8/8 ACs com evidência `file:line`; 1 spec-precision
gap (P2.1); 3 goals divergentes (G1/G2/G4) e **G3 não cumprido**. Sensor: 9
mutações, 7 mortas, **2 sobreviveram** (M7, M8). `validate_state.py` exit 1.

**Os gaps (ranqueados pelo Verifier):**

| # | Sev. | Gap | Onde |
| :--: | :--- | :--- | :--- |
| 1 | 🔴 Blocker | **`mem setup` não é alias do auto-wiring** — `cmd/mem/main.go:836` roteia `case "install":` e `cmd/mem/main.go:794` roteia `case "setup":` para o assistente interativo de cofre central (ADR-033). O alias existiu entre `db02542` e `8685ed8` e foi repurposado. A doc continua anunciando-o: `docs/CLI_GUIDE.md:625`, `:650-651` (`./bin/mem.exe setup --dry-run` → erro real `flag provided but not defined: -dry-run`), `docs/adr/026-...md:1,18,27,57` | `cmd/mem/main.go:794,836` |
| 2 | 🟠 Major | **P3.1/P3.2 sem discriminação na camada CLI** — mutantes M7 (`cmd/mem/install.go:85` ignora `--target`) e M8 (`cmd/mem/install.go:39-45` ignora `--workspace`/`--global`) sobreviveram à suíte inteira: uma regressão que escreva em todos os clientes, ou no escopo errado, **não quebra nenhum teste**. `cmd/mem/install_test.go:38-83` só afirma o lado positivo; não há teste de `--global` | `cmd/mem/install.go:39-45,85` |
| 3 | 🟠 Major | **o payload aponta para o SQLite enquanto a CLI declara que o Postgres venceu** — `cmd/mem/install.go:63` descarta a URL do Postgres resolvida e monta o payload só com `--db`/`--repo` (`:71-77`). Observado ao vivo: o log diz *"Postgres detectado... será ignorado nesta sessão"* e duas linhas abaixo, `Database: C:\Users\Felipe\.memory\memory.db`. Se o editor não herdar `MY_MEMORY_PG_URL`, `mem mcp --db <esse caminho>` abre um **store diferente** do vault vivo | `cmd/mem/install.go:63,71-77`, `cmd/mem/main.go:403-422` |
| 4 | 🟠 Major | **`--target claude-desktop` documentado e não aceito** — `docs/CLI_GUIDE.md:631` lista o valor; `internal/autowire/detector.go:222` só casa `claude`. Execução real → "Nenhum cliente MCP compatível encontrado", **exit 0, sem erro** | `internal/autowire/detector.go:222` |
| 5 | 🟠 Major | **"auto-detecta ferramentas instaladas" não é implementado** — o detector devolve **todos** os candidatos (`internal/autowire/detector.go:69-158`); `Exists`/`ParentDirOK` (`:199-217`) **nunca** são consultados por `cmd/mem/install.go:109-142`; o instalador cria o diretório e escreve o arquivo de qualquer jeito (`internal/autowire/installer.go:95-101`). Execução real em diretório vazio criou `.cursor/mcp.json` e `.vscode/mcp.json` | `internal/autowire/detector.go:199-217`, `cmd/mem/install.go:109-142` |
| 6 | 🟡 Minor | o backup `.bak` **não é atômico** — `os.WriteFile` sem temporário + rename; a spec (G2) e a ADR-026:19 exigem atomicidade | `internal/autowire/installer.go:105-108` |
| 7 | 🟡 Minor | indentação e ordem das chaves **não são preservadas** — `json.MarshalIndent(cfgMap, "", "  ")` sobre `map[string]interface{}` reordena em alfabético. O *conteúdo* foi preservado (a AC P1.2 **não** é violada); o que não se sustenta é o checkbox de T2 | `internal/autowire/installer.go:75` |
| 8 | 🟡 Minor | a ADR-026 descreve uma arquitetura que **não existe** — registro declarativo `ClientSpec` (o código tem `if` fixos e constantes `ClientType`) e escopo **Workspace** do Windsurf (só existe o global). A ADR também repete o alias `mem setup` | `docs/adr/026-...md:31-35,42` |
| 9 | 🟡 Minor | **T4 parcial** — a linha `mem install` foi adicionada ao `docs/AGENT_INTEGRATION_GUIDE.md` por `db02542` e removida por `f5646fd`; hoje o arquivo só menciona `mem hook install` | `docs/AGENT_INTEGRATION_GUIDE.md:271` |
| 10 | 🟡 Minor | o **plano de validação anterior listava 8 testes que nunca existiram** (de 12 nomes marcados como feitos, só 4 existem hoje) e afirmava "15 pacotes" (hoje são 27) | `.specs/features/ai-tool-auto-wiring/validation.md` (plano anterior) |

**O que o Verifier confirma que funciona:** o detector monta os candidatos, a
injeção preserva servidores e chaves paralelas do arquivo existente (AC P1.2
cumprida), o `--dry-run` mostra o que faria sem escrever, e o `mem mcp` sobe
nos dois transportes.

**Higiene:** sensor em worktree isolado (`D:\.minimax\scratch\subagents\verifier-atw\wt`,
sem `git stash`); o Postgres do dono nunca foi contatado; baseline e final do
`git status --porcelain` comparados.

**Referência:** ADR-026 · ADR-040 (precedência de storage) · ADR-033 ·
spec `ai-tool-auto-wiring` ·
`.specs/features/ai-tool-auto-wiring/validation.md`.
