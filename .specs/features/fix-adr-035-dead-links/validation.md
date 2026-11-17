# fix-adr-035-dead-links — Validation

**Date**: 2026-11-17
**Spec**: `.specs/features/fix-adr-035-dead-links/spec.md`
**Issue**: ISSUE-049 (`.specs/ISSUES.md`)
**Diff range**: `56be671..227c4b1` (T1: `56be671`, T2: `227c4b1`, T3: `e60777e`)
**Verifier**: Independent Verification Protocol (tlc-spec-driven)
**Status**: Complete

---

## Validation: PASS

**Result**: PASS

O `mem doctor` do vault do repositório (`repo_e4c8f3b1a2d5`) foi executado e confirmou **0 links rompidos**. As 3 ocorrências de `[[ADR-035]]` foram convertidas com sucesso para a forma canônica path-style com alias `[[docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md|ADR-035]]`, preservando a imutabilidade dos arquivos de decisão arquitetural em `docs/adr/` e registrando a regra no store de lições (`.specs/lessons.json`).

---

## Spec-Anchored Acceptance Criteria

| AC | Critério (WHEN X THEN Y) | Resultado definido na spec | Evidência `file:line` | Resultado |
| -- | ------------------------ | -------------------------- | --------------------- | --------- |
| **ADL-01** | WHEN o wikilink para ADR-035 aparece em `spec.md` THEN usa path-style com alias nas 2 ocorrências | `[[docs/adr/035-...minilm.md\|ADR-035]]` | `.specs/features/embedder-builtin-onnx-minilm/spec.md:11` e `:188` | ✅ Coberto |
| **ADL-02** | WHEN o wikilink para ADR-035 aparece em `tasks.md` THEN usa path-style com alias | `[[docs/adr/035-...minilm.md\|ADR-035]]` | `.specs/features/embedder-builtin-onnx-minilm/tasks.md:3` | ✅ Coberto |
| **ADL-03** | O arquivo da ADR-035 NÃO deve ser modificado pela correção | `git diff docs/adr/035-*.md` vazio | `docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md` intacto | ✅ Coberto |
| **ADL-04** | WHEN o vault é reindexado THEN `mem doctor` lista 0 links rompidos de ADR | 0 dead links de ADR reportados | `cmd/mem doctor` output: `[🔗 Links Quebrados / Dead Links] (0)` | ✅ Coberto |
| **ADL-05** | WHEN o link corrigido é renderizado THEN o display text continua `ADR-035` | alias `ADR-035` presente | `.specs/features/embedder-builtin-onnx-minilm/spec.md:11,188` | ✅ Coberto |
| **ADL-06** | A regra de path-style com alias deve ser gravada no store de lições | lição registrada via `lessons.py` | `.specs/lessons.json` (commit `e60777e`) | ✅ Coberto |

---

## Medição no Vault Real

```text
🩺 Auditando integridade do grafo no PostgreSQL [repo_e4c8f3b1a2d5]...
==========================================================================================
 🟡 RELATÓRIO DE SAÚDE DA MEMÓRIA & GRAFO (Health Score: 85/100)
==========================================================================================
 Documentos: 344 | Chunks: 3649 | Arestas: 54502 | Nós: 3757
------------------------------------------------------------------------------------------
 [🔗 Links Quebrados / Dead Links] (0)
  ✔ Nenhum link quebrado detectado. Todas as conexões apontam para notas existentes.
```

- Total de links rompidos de ADR: **0** (reduzido de 2 para 0).
- Arquivos em `docs/adr/` modificados: **0** (imutabilidade de ADRs preservada).
