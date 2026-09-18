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
- `Tags` no ADR-005 → wrappado em inline code `#Tags` → `` `#Tags` `` para o parser `tagRegex` ignorar (padrão Obsidian para exemplos de sintaxe).

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

## ISSUE-006 — Health Score persistentemente baixo (30/100)
- **Severidade:** 🟡 medium (métrica informativa, não bloqueia funcionalidade)
- **Status:** 🔵 open (consequência de outras issues)
- **Achado em:** `mem doctor --db .memory/memory.db` (sessão 2026-09-17)
- **Contexto:** mesmo após reduções (44 → 8 dead links, +30 edges recuperadas), Health Score permanece em 30/100. Fórmula do score provavelmente pesa notas órfãs (144) que existem por design (ADRs e skills não devem ser linkados ativamente).
- **Possível correção:** revisar fórmula do Health Score ou categorizar "órfão" como aceitável pra docs leaf.

---

## ISSUE-007 — 18 .md files em `node_modules/playwright-core/` desnecessários no repo
- **Severidade:** 🟢 low (artefatos de Playwright instalados pra validação visual)
- **Status:** ⏸️ deferred (já existem no `.gitignore` para novas instalações)
- **Achado em:** debug do fuzzy resolver (sessão 2026-09-17)
- **Contexto:** `node_modules/playwright-core/lib/tools/skills/playwright-*/SKILL.md` e similares foram instalados durante a sessão post-release-v1.3.0-bugfixes para visual validation, mas o conteúdo é do próprio Playwright (não do my-memory).
- **Workaround:** já estão excluídos do `.gitignore`. Futuras instalações do Playwright devem usar `npm install --no-save`.

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

---

**Próxima revisão:** quando iniciar próxima sessão (sessão `mvs_b361ffb514ef434a8c0c0d9342dfede6`).
**Mantido por:** Mavis (Mavis orchestrator) + Felipe Miiller (review).
**Localização:** `.specs/ISSUES.md` (commitado no repo).
