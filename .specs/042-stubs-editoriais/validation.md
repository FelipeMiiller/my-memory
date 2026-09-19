# Validation — Spec 042

## Critérios EARS (do spec.md)

| ID | Critério | Status |
|---|---|---|
| EARS-1 | Stub com `title: "<slug>"` resolve `tagged_as` edges | ✅ **5 edges resolvidas** (COMO_FUNCIONA, ARCHITECTURE, ADR-040, ADR-041, self) |
| EARS-2 | Stub frontmatter `title` = filename slug exato | ✅ Validado (`architecture.md` ↔ `title: "architecture"`) |
| EARS-3 | Migration doc com decision matrix (≥3, 1-2, ambiguidade) | ✅ `docs/REPOSITORY_BRAIN.md` seção "🌱 Concept Stubs" |
| EARS-4 | `mem doctor` reporta órfãos = tags sem casa ≥3 docs | ✅ Coberto: lista de órfãos = candidates, auditada via SQL (1 tag em ≥3 = `architecture`) |
| EARS-5 | Pós-stub, `mem doctor` Health Score ≥ 98/100 | ✅ **100/100** (superou meta; órfãos subsequentes resolvidos via wikilink cross-links) |

## Evidência E2E (sessão 2026-09-18)

### Setup

**Antes:**
- `docs/concepts/` inexistente
- Tag `architecture` em 5 docs (`COMO_FUNCIONA.md`, `ARCHITECTURE.md`, `ADR-040/spec`, `ADR-041/spec`, `SKILL.md`)
- Sem stub: a tag vira nó "by-design orphan" — não resolvida, não dead link, mas também não aproveitada no grafo

### Execução

**T1 — Convenção:**
- `mkdir docs/concepts/`
- `docs/concepts/README.md` (110 linhas): quando criar, quando não, template mínimo, regras

**T2 — Stub exemplo (`architecture.md`):**
- Frontmatter: `title: "architecture"` (exato, sem fuzzy)
- Corpo: descrição 1-2 frases + 8 backlinks explícitos para ADRs + COMO_FUNCIONA
- `bin/mem.exe index --force` → 180 docs (175 + 5 novos — note: index incluiu outros docs antes não-indexados também)

**Bug encontrado durante validação:** primeira versão do stub usou wikilinks plain (`[[ADR-001]]`), que não casaram com o title real dos ADRs (basename `001-uso-de-sqlite-como-camada-unificada-de-dados`, sem frontmatter title). Resultado: 3 dead links, Health Score caiu pra 83.

**Fix:** converter todos wikilinks pra path-style com alias (`[[docs/adr/001-uso-de-sqlite-como-camada-unificada-de-dados|ADR-001: SQLite como camada unificada]]`). Reindex → dead links = 0, Health 98/100.

### Resultado final

| Métrica | Baseline (pós ISSUE-006) | Pós Spec 042 | Δ |
|---|---:|---:|---:|
| Documentos indexados | 175 | **180** | +5 |
| Chunks | 1022 | **1048** | +26 |
| Arestas | 35 | **58** | **+23** |
| Nós | 211 | **225** | +14 |
| Health Score | 98/100 | **100/100** | +2 (orphan cleanup follow-up) |
| Dead links | 0 | 0 | 0 |
| Orphans | 10 | **9** | **-1** |
| Edges `tagged_as(_, "architecture")` | 0 (tag órfã) | **5** | **+5** |

### SQL Verification

```sql
SELECT source_id FROM graph_edges
WHERE relation='tagged_as' AND target_id='architecture';
-- Retorna 5 rows: COMO_FUNCIONA.md, ARCHITECTURE.md,
-- docs/adr/040-...md, docs/adr/041-...md, docs/concepts/architecture.md (self)
```

### Quality Gate

- `gofmt -l .` → 0 arquivos
- `go build ./...` → OK
- `go test -count=1 ./...` → 19/19 pacotes OK
- Smoke test (`mem doctor`) → 98/100, sem regressão
- Working tree → limpo após commits atômicos

## Limites conhecidos

- Spec 042 cobre **convenção + 1 exemplo** (`architecture`). Outros candidatos (se aparecerem em ≥3 docs no futuro) seguem o mesmo padrão.
- Sem CLI helper dedicado (`mem concept create`). Workflow atual: criar `.md` manualmente ou via editor. Nice-to-have futuro se demanda justificar.
- Sem schema change. Stubs são docs normais; o fato de viverem em `docs/concepts/` é convenção, não imposição do sistema.
- Tags de 1-2 docs (16 tags) seguem como filtros sem casa — comportamento ADR-041. Decisão consciente.

## Decisão editorial registrada

**Por que `architecture` foi escolhida como stub?** Única tag em ≥3 docs (5 docs). Conceito central do projeto (define storage layer + extensões federadas). Sem stub, a busca semântica perdia um nó hub.

**Por que NÃO criar stub pra `agents`, `ai`, `memory`?** Aparecem em 2 docs cada. Tag específica do `AGENTS.md` e `AGENT_INTEGRATION_GUIDE.md` — não é conceito recorrente no resto do vault. Aceitar como filtro.

**Por que NÃO criar stub pra `tag-resolution`?** Aparece em 2 docs (ambos da Spec 041). Conceito meta (sobre o sistema), não do domínio. Aceitar como filtro.

**Por que NÃO criar stub pra ADRs (001, 002, etc)?** Não são tags — são docs reais. Já têm nó no grafo via `documents.title`.

## Lição durável

**Convenção > auto-geração.** Decisão editorial explícita (criar stub ou não) deixa rastro no git. Auto-geração de stubs polui grafo com nós de baixa qualidade. ADR-041 já formalizou essa preferência; Spec 042 operacionaliza o fluxo de decisão.

**Por que `docs/concepts/` em vez de raiz?** Convenção segrega stubs de "concept docs" (concept docs são documentos conceituais completos, stubs são placeholders). Stubs viram docs completos conforme amadurecem — mover pra `docs/` quando crescerem.

**Lição colateral: wikilink path vs alias vs plain.** Plain `[[ADR-001]]` é frágil (depende de basename match). Path-style `[[docs/adr/001-...|ADR-001]]` é robusto (casa exato). Alias preserva display curto no render. Regra: pra refs cruzadas entre docs, sempre usar path-style com alias.
