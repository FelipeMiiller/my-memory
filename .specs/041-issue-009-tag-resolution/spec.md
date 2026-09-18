---
title: "Spec 041: ADR-041 — Tag sem casa removida do grafo (validação + stubs opcionais)"
category: resource
summary: "Spec de implementação do ADR-041. Cobre validação E2E da fix do parser, criação de stubs opcionais para tags conceituais recorrentes, e edge cases de migration."
tags: [spec, parser, graph, dead-links, tag-resolution, stubs]
---

# Spec 041 — ADR-041 implementação

**Status**: pending (criação)
**ADR de referência**: [ADR-041](../../docs/adr/041-issue-009-tag-sem-casa-removida-do-grafo.md) (Accepted)
**Issue relacionada**: ISSUE-009 (resolved em `4c19393`)
**Implementation commit**: `4c19393`

## Escopo

Esta spec cobre validação pós-implementação do ADR-041 e tarefas de polish/migration:

1. **Validação E2E**: confirmar redução de dead links para 0-1 no `mem doctor`
2. **Stubs sugeridos**: para tags conceituais recorrentes (`architecture`, `federation`, `storage`, `config`, `sqlite`, `postgres`), avaliar criação de notas stub como nós do grafo (decisão editorial, não automática)
3. **Migration path docs**: tutorial "como criar stub pra tag conceitual"
4. **Investigação do spec.md self-reference**: ainda existe 1 dead link misterioso — investigar se é caso coberto por esta fix ou ISSUE separada

## Fora do escopo

- Refactor do `mem search --tag X` (não existe atualmente — só `--tags` em `mem note create`)
- Mudança de schema (`graph_edges.target_kind`)
- ADR-041 option 3 (namespace `tag:<valor>`) — descartado por decisão arquitetural

## Critérios EARS

- **EARS-1**: After `mem index --force` with ADR-041 fix applied, the system MUST report 0-2 dead links in `mem doctor` (down from 19 pré-fix).
- **EARS-2**: After creating a stub doc with `title: "architecture"`, the system MUST resolve `tagged_as(doc, "architecture")` edges to that stub (verified via `mem doctor`).
- **EARS-3**: After ADR-041 fix, the `TestResolveTagConnections/ISSUE-009:*` tests MUST pass without regression in existing tests.
- **EARS-4**: When `mem search <query>` runs, the system MUST find docs by tag text content (FTS), independently of whether tagged_as edge exists.

## Tarefas (a serem preenchidas)

Ver `tasks.md` (skeleton).

## Validação

Ver `validation.md` (skeleton).
