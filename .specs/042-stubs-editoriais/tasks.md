# Tasks — Spec 042

## T1 — Convenção `docs/concepts/` ✅ Done (nesta sessão)

- [x] Criar pasta `docs/concepts/` (canônica para stubs editoriais)
- [x] Criar `docs/concepts/README.md` explicando a convenção, template mínimo, e regras de ouro

**Resultado**:
- `docs/concepts/README.md` documenta:
  - Quando criar stub (≥3 docs referenciam a tag)
  - Quando NÃO criar (1-2 docs, tag específica)
  - Template mínimo (frontmatter + 1-2 frases + lista de backlinks)
  - Regra: `title` do frontmatter deve casar exatamente com o slug do arquivo
- Convenção segue padrão Obsidian Flavored Markdown (skill `my-memory-format`)

**Verificação**: arquivo presente, renderiza OK no Obsidian, segue convenção.

## T2 — Stub exemplo: `architecture` ✅ Done (nesta sessão)

- [x] Criar `docs/concepts/architecture.md` com:
  - frontmatter `title: "architecture"`
  - descrição 1-2 frases
  - lista de backlinks explícitos (5 docs que usam a tag)
- [x] Rodar `bin/mem.exe index --force`
- [x] Rodar `bin/mem.exe doctor --db .memory/memory.db`
- [x] Confirmar que `architecture` NÃO aparece mais em dead links
- [x] Confirmar que Health Score **não regrediu** (98/100 mantido)

**Resultado**:
- Stub criado: `docs/concepts/architecture.md`
- Indexado sem warnings
- `mem doctor` pós-fix: Health Score **98/100** mantido, dead links = 0
- Tag `architecture` agora casa via exact title match → nó real do grafo

**Verificação** (a ser documentada em validation.md):
- SQL: `SELECT source_id, target_id FROM graph_edges WHERE relation = 'tagged_as' AND target_id = 'architecture'` → retorna edges resolvidas
- `mem doctor` não reporta `architecture` como dead link

## T3 — Migration doc: seção em `docs/REPOSITORY_BRAIN.md` ✅ Done (nesta sessão)

- [x] Adicionar seção "Concept Stubs: Quando e Como" em `docs/REPOSITORY_BRAIN.md`
- [x] Decision matrix:
  | Situação | Ação |
  |---|---|
  | Tag usada em ≥3 docs | Criar stub em `docs/concepts/<tag>.md` |
  | Tag usada em 1-2 docs | Aceitar como filtro (não precisa stub) |
  | Tag ambígua (significados diferentes em docs diferentes) | Renomear pra ser específica |
- [x] Walkthrough: criar `architecture.md` do zero até validação no `mem doctor`
- [x] Cross-link com `docs/CLI_GUIDE.md` (seção relevante) e ADR-041

**Resultado**: `docs/REPOSITORY_BRAIN.md` agora tem seção dedicada. Decisão editorial fica explícita e versionada no repo.

## T4 — Quality gate final ✅ Done (nesta sessão)

- [x] `gofmt -l .` → vazio (0 arquivos)
- [x] `go build ./...` → OK
- [x] `go test -count=1 ./...` → 19/19 pacotes OK
- [x] `mem index --force` → sucesso
- [x] `mem doctor` → Health Score 98/100, dead links 0, órfãos 9 (não-regrediu: era 10 antes; `architecture` saiu do orphan list porque agora é resolvido como nó)
- [x] Commits atômicos (1-3 commits por task, mensagens conventional)

## Commits esperados

- `docs(concepts): add docs/concepts/ convention + architecture stub`
- `docs(repobrain): add "Concept Stubs: Quando e Como" migration section`
- `docs(specs): add Spec 042 — stubs editoriais + migration path`
