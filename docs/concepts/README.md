---
title: "Concept Stubs Convention"
category: resource
summary: "Convenção para criar stubs editoriais de conceitos recorrentes no vault My-Memory. Define pasta canônica, template mínimo, e regras de ouro."
tags: [concepts, stubs, convention, docs]
---

# Concept Stubs — Convenção

**Stubs editoriais** são notas curtas que servem como "casa" para conceitos recorrentes no grafo. Quando uma tag é usada em **≥3 documentos** e representa um conceito central, criar um stub garante que esse conceito vire um nó real do grafo (não apenas um filtro solto).

## Quando criar um stub

| Situação | Ação |
|---|---|
| Tag usada em **≥3 docs** e representa conceito central | ✅ **Criar stub** |
| Tag usada em 1-2 docs (específica) | ❌ Aceitar como filtro |
| Tag ambígua (significados diferentes em docs diferentes) | 🔧 Renomear pra ser específica |

**Por que ≥3?** Limiar arbitrário mas útil: abaixo disso, a tag é específica demais pra ser hub do grafo. Acima, é conceito recorrente e merece nó.

## Onde colocar

Pasta canônica: **`docs/concepts/<slug>.md`**

- `slug` = versão kebab-case do conceito (`architecture`, `machine-learning`, `graph-theory`)
- Arquivo deve ter `title: "<slug>"` no frontmatter (casa exata, sem fuzzy fallthrough)
- Convenção segrega stubs de "concept docs" completos em `docs/` raiz

## Template mínimo

```markdown
---
title: "<slug>"
category: resource
summary: "<descrição de 1 linha>"
tags: [concept, <tags-relacionadas>]
---

# <slug>

<descrição de 1-2 frases do conceito>

## Onde aparece neste vault

- [[doc-que-referencia]] — contexto
- [[outro-doc]] — contexto
```

## Regras de ouro

1. **`title` exato** = slug do arquivo (sem espaços, sem aliases). O parser casa via exact title match pra stubs.
2. **Corpo curto**: 1-2 frases + lista de backlinks. Stubs NÃO são docs completos — esboço curto é o ponto.
3. **Backlinks explícitos**: liste docs que referenciam a tag. Mantém o stub útil como índice de navegação.
4. **Cresça gradualmente**: se o stub virar doc conceitual maduro (>200 linhas, tem seções próprias), promova pra `docs/<slug>.md` raiz.
5. **Decisão editorial**: humano decide o que é stub. Sistema **não** auto-gera.

## Quando NÃO criar

- **Tags de domínio específico** (`pipeline-de-ci`, `ci-go-version`) — vivem em 1-2 docs só
- **Tags de trabalho** (`adr-040`, `spec-041`) — versionadas com o trabalho, não conceitos
- **Tags-meta** (`parser`, `graph`) — descrevem o sistema, não o domínio

## Verificação pós-criação

```bash
bin/mem.exe index --force
bin/mem.exe doctor --db .memory/memory.db
```

Esperado:
- Documentos indexados: +1
- Dead links: **0** (sem regressão)
- Orphans: pode cair 1 (a tag stub sai da lista de by-design... não, espera, ela vira nó resolvido, não órfão)
- Health Score: **não** regride

## Ver também

- [ADR-041](../../adr/041-issue-009-tag-sem-casa-removida-do-grafo.md) — regra original: tags sem casa viram filtros, não edges
- [Spec 042](../../specs/042-stubs-editoriais/spec.md) — escopo + EARS
- `docs/REPOSITORY_BRAIN.md` — seção "Concept Stubs: Quando e Como" (migration walkthrough)
