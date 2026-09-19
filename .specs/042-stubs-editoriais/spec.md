---
title: "Spec 042 — Stubs editoriais para conceitos recorrentes + migration path"
category: resource
summary: "Convenção para criar stubs de conceitos recorrentes (ex: architecture) como nós do grafo, decisão editorial explícita, e migration doc explicando o fluxo. Herda T3+T4 deferred do Spec 041."
tags: [spec, concepts, stubs, migration, docs]
---

# Spec 042 — Stubs editoriais + migration path

**Status**: pending (criação)
**ADR de referência**: nenhum novo — extensão do ADR-041 (Accepted)
**Issue relacionada**: nenhuma aberta (ISSUE-009 + ISSUE-010 já resolveram dead links tecnicamente)
**Specs relacionadas**: [Spec 041](../041-issue-009-tag-resolution/spec.md) (T3 + T4 deferred → esta spec)

## Contexto

Após Spec 041 (ADR-041 + ISSUE-010), o sistema não gera mais dead links automaticamente — tags sem casa viram filtros, não edges. **Mas conceitos recorrentes merecem ser nós do grafo**:

| Tag | Docs | Categoria |
|---|---:|---|
| `architecture` | 5 | candidato a stub (conceito central) |
| `agents`, `ai`, `memory`, `turboquant`, `cli`, `skill`, `federation`, `storage`, `parser`, `graph`, `dead-links`, `tag-resolution`, `sqlite`, `postgres` | 2 cada | específicos (não justificam stub) |
| Demais (≥1 doc) | 1 cada | single-doc tags (não justificam stub) |

**Problema atual:** não existe convenção formal pra criar "stub" de conceito. Usuário novo não sabe:
1. Onde colocar stub (`docs/concepts/`? raiz?)
2. Qual template seguir
3. Quando vale criar vs aceitar como filtro
4. Como verificar que funcionou

**Resultado:** conceitos centrais (`architecture`) ficam órfãos do grafo, empobrecendo busca semântica e conexões.

## Escopo

1. **Convenção `docs/concepts/`**: pasta canônica pra stubs editoriais
2. **Template mínimo**: frontmatter `title: "<slug>"` + corpo curto (descrição 1-2 frases + lista de backlinks)
3. **Stub exemplo**: criar `docs/concepts/architecture.md` (única tag em ≥3 docs)
4. **Migration doc**: seção em `docs/REPOSITORY_BRAIN.md` (tópico "Concept Stubs: Quando e Como") com decision matrix
5. **Validação**: `mem doctor` mostra `architecture` como nó resolvido (não dead link) após indexar

## Fora do escopo

- CLI helper `mem concept create <slug>` (Nice-to-have, deferrable — `New-Item`/edit tool resolve)
- Auto-geração de stubs (decisão 100% editorial)
- Schema change (`graph_nodes.kind = 'concept'`)
- Migration de tags em massa (só 1 stub neste repo)

## Critérios EARS

- **EARS-1**: After creating `docs/concepts/<slug>.md` with `title: "<slug>"`, the system MUST treat it as candidate target for `tagged_as` edges from any doc whose tags include `<slug>` (verified by `mem doctor` no longer reporting dead link to that tag).
- **EARS-2**: The stub doc MUST have frontmatter `title` matching its filename slug exactly (no fuzzy fallthrough — exact title match for stubs).
- **EARS-3**: The migration doc MUST include a decision matrix: criar stub | aceitar como filtro | renomear tag, com critério objetivo (≥3 docs vs 1-2 vs ambiguidade).
- **EARS-4**: After `mem concept audit` (CLI), the system MUST list tags used in ≥3 docs that lack a corresponding `docs/concepts/<tag>.md` stub. (Pode ser `mem doctor` reportando órfãos; não precisa novo comando.)
- **EARS-5**: After committing `architecture.md` stub and re-indexing, `mem doctor` MUST NOT report `architecture` as dead link; Health Score MUST NOT regress.

## Tarefas

Ver `tasks.md`.

## Validação

Ver `validation.md`.
