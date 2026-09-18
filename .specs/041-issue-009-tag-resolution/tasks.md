# Tasks — Spec 041

## T1 — Validação E2E da fix do ISSUE-009 ✅ Done (2026-09-18)

- [x] Rodar `bin/mem.exe index --force` em repo limpo
- [x] Rodar `bin/mem.exe doctor --db .memory/memory.db`
- [x] Confirmar dead links ≤ 2 (down from 19 pré-fix) — **atingiu 0** após `--fix`
- [x] Confirmar Health Score subiu de 33 → ≥ 60 — **atingiu 74/100**

**Resultado**:
- Baseline inicial (com fix ADR-041): Health 54/100, **4 dead links** tagged_as auto-referências
- Após `doctor --fix`: Health **74/100**, **0 dead links** (limpeza completa)
- Cache SHA-256 funcionando: 175/175 docs inalterados → 0 indexados na 2ª run

**Verificação**: smoke test E2E documentado em `validation.md`

## T2 — Investigar spec.md self-reference dead link ✅ Done (2026-09-18) — ISSUE-010

- [x] Inspecionar SQL: `SELECT source_id, target_id FROM graph_edges WHERE target_id LIKE '%Spec 040%'` → **4 rows órfãs confirmadas**
- [x] Verificar se ainda existe após `mem index --force` com fix → **SIM, persiste** (4 edges órfãs tagged_as com target = frontmatter.title longo)
- [x] Rastrear root cause: **ISSUE-010** (novo) — `documents.title` usa **basename**, mas `preCollectDocTitles` usa **frontmatter.title**. Fuzzy substring match em `ResolveTagConnections` reescreve target para o título longo, gerando dead links
- [x] Documentar root cause + fix em ISSUE-010

**Cadeia do bug (ISSUE-010):**
1. `runIndexSQLite` linha 1605: `title := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))` (basename)
2. `preCollectDocTitles` linha 1369: `title = fm.Title` (frontmatter.title completo)
3. Fuzzy substring match casa tag curta "spec" → contém em "spec040:implementacao..." → resolve target = frontmatter.title longo
4. Edge gravada no DB com target longo; `mem doctor` consulta `documents.title` (basename curto) → não acha → **dead link**
5. Após `--force`, ciclo se repete. `doctor --fix` é o único paliativo.

**Fix aplicado (commit `<TBD>`):**
- `cmd/mem/main.go::runIndexSQLite` (linha ~1457 + ~1617) — usa `fm.Title` quando existir
- `internal/watcher/indexer.go::IndexSingleFileSQLite` (linha ~47 + ~182) — mesmo fix
- Teste novo: `TestIndexSingleFileUsesFrontmatterTitle` valida que `documents.title` = `frontmatter.title`

**Resultado pós-fix**:
- `documents.title` AGORA = frontmatter.title (verificado via SQL: AGENTS.md → "Guia Operacional para Agentes de IA")
- `mem index --force` 2ª execução: Health **74/100**, **0 dead links** ✓
- Idempotente: 3ª execução mantém 0 dead links (cache SHA-256)

**Verificação**: SQL query `SELECT ... WHERE target_id NOT IN (SELECT title FROM documents)` → 0 rows

## T3 — Avaliar criação de stubs para tags conceituais recorrentes ⏸ Deferred

- [ ] Listar tags mais usadas em frontmatter (grep recursivo `tags: \[[^]]*\]` em `.md`)
- [ ] Identificar tags conceituais que aparecem em ≥ 3 docs (provavelmente querem ser nós do grafo)
- [ ] Decidir quais criar como stubs vs aceitar como filtro (sem nó)
- [ ] Para cada stub candidato, criar `docs/concepts/<tag>.md` com frontmatter `title: "<tag>"` e corpo mínimo

**Status**: Adiado para Spec 042 (decisão editorial + criação de stubs). Sem urgência técnica após fix do ISSUE-010.

## T4 — Migration path docs (Como criar stub pra tag conceitual) ⏸ Deferred

- [ ] Adicionar seção em `docs/CLI_GUIDE.md` ou `docs/REPOSITORY_BRAIN.md`:
  > Se você quer que uma tag conceitual (ex: `architecture`) seja um nó do grafo, crie um stub:
  > ```bash
  > echo '# architecture' > docs/concepts/architecture.md
  > ```
  > O frontmatter `title: "architecture"` (via `mem init` template) vira nó do grafo automaticamente.

**Status**: Adiado para Spec 042.

## T5 — Quality gate final ✅ Done (2026-09-18)

- [x] `gofmt -l .` → vazio (0 arquivos)
- [x] `go build ./...` → OK
- [x] `go test -count=1 ./...` → 19/19 pacotes OK
- [x] Commits atômicos (T1+T2 num commit único "ISSUE-010: title usa frontmatter.title")
