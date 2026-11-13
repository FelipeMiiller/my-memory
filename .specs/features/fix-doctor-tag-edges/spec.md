# Feature: fix-doctor-tag-edges

- **Spec ID**: `fix-doctor-tag-edges`
- **Created**: 2026-11-13
- **Owner**: Felipe Miiller
- **Source**: ISSUE-038 — `.specs/ISSUES.md` (seção ISSUE-038)
- **Related**: ADR-041/ADR-055 (tags) · ADR-013 (Health Score) · ISSUE-036 (nó de código como alvo vivo) · ADR-062

## Problem Statement

O `mem doctor` conta **toda** aresta `tagged_as` cujo alvo não resolve para um
documento como **link morto**. Mas o alvo de uma aresta de tag é uma **tag** — e
a tag vive em `document_tags` (ADR-055), não em `documents`. O check só isenta
alvos que começam com `#`:

```sql
AND e.target_id NOT LIKE '#%'
AND e.target_id NOT IN (SELECT id FROM documents WHERE repository = $1)
AND e.target_id NOT IN (SELECT title FROM documents WHERE repository = $1)
```

O caminho da tag **inline** (`#tag`) grava o alvo prefixado; o caminho da tag
de **frontmatter** (`tags: [tag]`) grava o nome **cru** → o check de dead link
não reconhece e marca como quebrado.

**Prova medida (cofre central `repo_central`, 2026-11-13).**

| Origem | Count | Realidade |
| :--- | :--- | :--- |
| `tagged_as` | **80** | **falso positivo** — a tag existe em `document_tags` |
| `links_to` | **68** | real no índice (material bruto fora do índice — ISSUE-039) |
| Total reportado | **148** | Health Score **60/100** (teto de 40 pontos de penalidade) |

Sem os 80 falsos positivos, o cofre sai em **~90/100** — o valor compatível com
um vault com 0 órfãos e 2 links realmente quebrados.

## Goals

- [x] O alvo de uma aresta `tagged_as` SHALL NOT ser contado como link morto quando a tag existe.
- [x] Uma aresta de tag cuja tag **não** existe em `document_tags` SHALL continuar contando como link morto.
- [x] A rotina de correção (`FixHealthIssues` / `FixGraphIssues`) SHALL usar o **mesmo** critério do relatório — para não apagar uma aresta de tag viva.
- [x] O comportamento SHALL ser o mesmo nos **dois** engines (SQLite e Postgres).
- [x] Nenhum dado gravado muda: a correção é **só de medição**.

## Out of Scope

- **Os 68 `links_to` para o material bruto** — é a ISSUE-039 e depende de decisão de política do dono.
- **Gravar o alvo da tag com `#`** — alinharia os dois caminhos de tag, mas muda o **dado** gravado e exigiria reindexar os dois vaults. A decisão (mudar o dado ou a medição) fica para a spec do dono; aqui a medição é corrigida.
- **Mudar o Health Score** — a fórmula não muda; o que muda é a lista de anomalias reais.

## Acceptance Criteria

1. WHEN o doctor audita um vault THEN o alvo de uma aresta `tagged_as` SHALL NOT ser reportado como dead link se a tag existir em `document_tags` (por `tag` ou `tag_key`).
2. IF a tag de uma aresta `tagged_as` não existir em `document_tags` THEN a aresta SHALL continuar sendo reportada como dead link.
3. WHEN a rotina de correção roda THEN ela SHALL apagar exatamente o mesmo conjunto de arestas que o relatório aponta como morto.
4. The regression tests SHALL fail against the pre-fix code and pass after the fix, on both engines.

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| A identidade da tag é `tag` (display) ou `tag_key` (normalizada) | comparar com **as duas** | o alvo da aresta é o nome como escrito; o `tag_key` cobre a forma normalizada | y |
| O filtro é por relação (`tagged_as`) ou por existência do alvo | por **existência do alvo** em `document_tags` | é o mesmo mecanismo do ISSUE-036 (nó de código como alvo vivo) e não depende da relação | y |
| O escopo no Postgres é por `repository` | sim | `document_tags` tem `repository` na PK | y |

Open questions: none

## User Stories

- Como dono do cofre central, quero que o `mem doctor` não me diga que minhas notas têm links quebrados quando o que existe é tag, para que o Health Score signifique alguma coisa.
- Como dono do vault, quero que a rotina de correção apague só o que está realmente quebrado, para não perder arestas de tag.

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Where |
| :--- | :--- | :--- |
| DTE-1 | AC1 — tag viva não é link morto | `internal/store/doctor.go` · `internal/db/graph.go` |
| DTE-2 | AC2 — tag morta continua morta | `internal/store/doctor_code_nodes_test.go` · `internal/db/doctor_code_nodes_test.go` |
| DTE-3 | AC3 — a correção usa o mesmo critério | `internal/db/doctor_code_nodes_test.go` |
| DTE-4 | AC4 — vermelho contra o código antigo | `internal/db/doctor_code_nodes_test.go` |
