# Validation — Spec 041

## Critérios EARS (do spec.md)

| ID | Critério | Status |
|---|---|---|
| EARS-1 | `mem doctor` ≤ 2 dead links após `mem index --force` | ✅ **0 dead links** (sessão 2026-09-18) |
| EARS-2 | Stub `title: "architecture"` resolve `tagged_as` edges | 🔄 (depende T3 — deferred para Spec 042) |
| EARS-3 | Testes `TestResolveTagConnections/ISSUE-009:*` passam sem regressão | ✅ (validado em `4c19393`) |
| EARS-4 | `mem search <query>` funciona por FTS independente de tagged_as | ✅ (validado por design) |

## Evidência E2E (sessão 2026-09-18)

### T1 — Smoke Test Inicial (pós ADR-041, pré ISSUE-010 fix)

**Comando:**
```bash
bin/mem.exe index --force
bin/mem.exe doctor --db .memory/memory.db
```

**Resultado:**
- Health Score: **54/100**
- Dead links: **4** tagged_as auto-referências (spec.md/adr.md → próprio frontmatter.title)
- 175 documentos indexados
- 1022 chunks
- 42 arestas

**Ação corretiva via `mem doctor --fix`:**
- Health Score: **74/100** (+20)
- Dead links: **0** (auto-cura completa)

### T2 — Root Cause Investigation → ISSUE-010

**Investigação SQL:**
```sql
SELECT source_id, target_id FROM graph_edges
WHERE relation = 'tagged_as'
  AND (target_id LIKE '%Spec 04%' OR target_id LIKE '%ADR-04%')
```

**4 dead edges confirmadas:**
| source_id | target_id |
|---|---|
| `.specs/040-config-global-unica/spec.md` | `Spec 040: Implementação ADR-040 — Config global única + SQLite auto-scope + Postgres opt-in` |
| `.specs/041-issue-009-tag-resolution/spec.md` | `ADR-041: Tags sem doc-casa são removidas do grafo (não viram dead links)` |
| `.specs/041-issue-009-tag-resolution/spec.md` | `Spec 041: ADR-041 — Tag sem casa removida do grafo (validação + stubs opcionais)` |
| `docs/adr/041-issue-009-tag-sem-casa-removida-do-grafo.md` | `ADR-041: Tags sem doc-casa são removidas do grafo (não viram dead links)` |

**Root cause (3 lugares com mesmo bug):**
- `cmd/mem/main.go::runIndexSQLite` (Postgres + SQLite) usa **basename** como `documents.title`
- `cmd/mem/main.go::preCollectDocTitles` usa **frontmatter.title** como `availableTitles`
- `internal/watcher/indexer.go::IndexSingleFileSQLite` (Postgres + SQLite) também usa basename

**Cadeia:** tag `"spec"` no frontmatter → fuzzy substring match em `"spec040:implementacaoadr040..."` → resolve target = frontmatter.title longo → grava no DB → `mem doctor` consulta `documents.title` (basename) → não acha → dead link.

### T2 — Fix Aplicado (commit `<TBD>`)

**Mudança em 4 lugares:** `cmd/mem/main.go::runIndexSQLite` (Postgres + SQLite) + `internal/watcher/indexer.go::IndexSingleFileSQLite` (Postgres + SQLite). Cada lugar agora lê `fm.Title` antes de gravar `documents.title`.

**Teste novo:** `TestIndexSingleFileUsesFrontmatterTitle` em `internal/watcher/indexer_test.go` valida que `documents.title` = `frontmatter.title` (não basename).

**Validação E2E pós-fix:**

| Execução | Health Score | Dead links | Estado |
|---|---:|---:|---|
| Baseline (pré-fix, pós `--fix`) | 74/100 | 0 | limpo |
| Pós-fix 1ª execução (`--force`) | 34/100 | 12 | estado intermediário (DB com dados antigos + novos misturados) |
| Pós-fix 2ª execução (`--force`) | **74/100** | **0** | **correto, idempotente** |

**Verificação SQL pós-fix:**
```sql
SELECT e.source_id, e.target_id FROM graph_edges e
LEFT JOIN documents d ON d.title = e.target_id
WHERE e.relation = 'tagged_as' AND d.title IS NULL
-- retorna 0 rows
```

### Quality Gate Final

- `gofmt -l .` → **0** arquivos
- `go build ./...` → OK
- `go test -count=1 ./...` → **19/19 pacotes OK**
- `go test -run TestIndexSingleFileUsesFrontmatterTitle ./internal/watcher/...` → OK

## T3 — Stubs para tags conceituais ⏸ Deferred

Adiado para Spec 042 (decisão editorial + criação de stubs). Sem urgência técnica após fix do ISSUE-010.

## T4 — Migration path docs ⏸ Deferred

Adiado para Spec 042.

## Commits

- `4c19393` — `fix(parser): ISSUE-009 — tags sem casa removidas do grafo`
- `d8b274d` — `docs(adr+issues): ADR-041 + ISSUE-009 → resolved`
- `<TBD>` — `fix(indexer): ISSUE-010 — documents.title usa frontmatter.title`
- `<TBD>` — `test(watcher): ISSUE-010 — TestIndexSingleFileUsesFrontmatterTitle`
- `<TBD>` — `docs(specs+issues): ISSUE-010 spec.md/tasks.md/validation.md + ISSUES.md`
