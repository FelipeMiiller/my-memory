# Tasks — Spec 041

## T1 — Validação E2E da fix do ISSUE-009

- [ ] Rodar `bin/mem.exe index --force` em repo limpo
- [ ] Rodar `bin/mem.exe doctor --db .memory/memory.db`
- [ ] Confirmar dead links ≤ 2 (down from 19 pré-fix)
- [ ] Confirmar Health Score subiu de 33 → ≥ 60

**Verificação**: smoke test E2E documentado em `validation.md`

## T2 — Investigar spec.md self-reference dead link

- [ ] Inspecionar SQL: `SELECT source_id, target_id FROM graph_edges WHERE target_id LIKE '%Spec 040%'`
- [ ] Verificar se ainda existe após `mem index --force` com fix
- [ ] Se persistir: rastrear se é bug separado (potencial ISSUE-010) ou se spec.md precisa de ajuste
- [ ] Documentar root cause + fix em ISSUE-010 ou comentário no spec.md

**Verificação**: SQL query retorna 0 rows após fix

## T3 — Avaliar criação de stubs para tags conceituais recorrentes

- [ ] Listar tags mais usadas em frontmatter (grep recursivo `tags: \[[^]]*\]` em `.md`)
- [ ] Identificar tags conceituais que aparecem em ≥ 3 docs (provavelmente querem ser nós do grafo)
- [ ] Decidir quais criar como stubs vs aceitar como filtro (sem nó)
- [ ] Para cada stub candidato, criar `docs/concepts/<tag>.md` com frontmatter `title: "<tag>"` e corpo mínimo

**Decisão editorial**: usuário (Felipe) decide quais criar

## T4 — Migration path docs (Como criar stub pra tag conceitual)

- [ ] Adicionar seção em `docs/CLI_GUIDE.md` ou `docs/REPOSITORY_BRAIN.md`:
  > Se você quer que uma tag conceitual (ex: `architecture`) seja um nó do grafo, crie um stub:
  > ```bash
  > echo '# architecture' > docs/concepts/architecture.md
  > ```
  > O frontmatter `title: "architecture"` (via `mem init` template) vira nó do grafo automaticamente.

**Verificação**: doc atualizada, linkada de ADR-041

## T5 — Quality gate final

- [ ] `gofmt -l .` → vazio
- [ ] `go build ./...` → OK
- [ ] `go test -count=1 ./...` → 19/19 pacotes OK
- [ ] Commit atômico por fase (T1, T2, T3, T4)
