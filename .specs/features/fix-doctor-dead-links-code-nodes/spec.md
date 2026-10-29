# Feature: fix-doctor-dead-links-code-nodes

- **Spec ID**: `fix-doctor-dead-links-code-nodes`
- **Created**: 2026-10-29
- **Owner**: Felipe Miiller
- **Source**: ISSUE-036 — `.specs/ISSUES.md` (seção ISSUE-036)
- **Related**: ADR-061 (code graph, CG-17/CG-18) · ADR-013 (Health Score) · ISSUE-034 (code graph em lote)

## Problem Statement

O check de dead links do `mem doctor` considera "vivo" **só** o alvo que resolve
para um **documento** (id ou title); as tags (`#%`) são ignoradas por desenho. O
alvo de uma aresta `references_code` (ADR-061) é um **nó de código**
(`type='code_symbol'`, id `code:<repo>:<qualified_name>`) — que não é documento,
mas **existe** no grafo. Então **toda** aresta da ponte contava como link
quebrado:

- **PostgreSQL** — `internal/store/doctor.go:96-101` (e o caminho de correção em `:269-274`);
- **SQLite** — `internal/db/graph.go:271-278` (e o caminho de correção em `:421-427`).

**E o caminho de correção apagava o code graph**: o `DELETE FROM graph_edges`
classificava as **38 304** arestas `references_code` como mortas e as removia.

**Medido** (Neon, `repo_e4c8f3b1a2d5`):

| Métrica | Valor |
| --- | --- |
| nós `code:*` no grafo | **2 959** (existem) |
| arestas `references_code` | **38 304** |
| arestas `references_code` **sem** nó de destino | **0** → nenhum link está realmente quebrado |
| `mem doctor` reportando | **38 306** dead links (falso positivo) |
| Health Score | **60/100** (antes do code graph: **75/100**) |

## Goals

- [x] O nó de código SHALL NOT ser contado como dead link.
- [x] O alvo que não é documento **nem** nó de código SHALL continuar contando como dead link.
- [x] A rotina de correção SHALL NOT apagar arestas de code graph.
- [x] O comportamento SHALL ser o mesmo nos **dois** engines.
- [x] No PostgreSQL, o escopo SHALL ser o **mesmo repositório** da aresta.
- [x] Teste de regressão nos dois engines, com prova de que morre contra o código antigo.

## Out of Scope

- **Mudar o que é dead link** (a noção "o alvo resolve para um documento") — só o nó de código entra na lista de alvos vivos.
- **Os nós de `type='concept'`** — continuam contando como dead link, como antes.
- **Mudar schema, tabelas ou ADR** — nenhum ADR novo é necessário (ver A8).
- **O `mem drift`** — é a ISSUE-035, feature separada.

## Acceptance Criteria

1. IF o alvo de uma aresta é um nó `code_symbol` THEN o relatório SHALL NOT listar essa aresta como dead link.
2. IF o alvo de uma aresta não é documento nem nó `code_symbol` THEN o relatório SHALL listar essa aresta como dead link.
3. IF a rotina de correção roda num grafo com code graph THEN ela SHALL NOT apagar as arestas `references_code`.
4. IF a rotina de correção roda num grafo com aresta órfã THEN ela SHALL apagar essa aresta.
5. WHEN o motor é PostgreSQL THEN a busca do nó SHALL ser escopada pelo **mesmo** repositório da aresta.
6. The regression tests SHALL fail against the pre-fix code and pass after the fix, on both engines.

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale |
| --- | --- | --- |
| Só o nó `code_symbol` entra na lista de alvos vivos | filtrar por `type = 'code_symbol'` | é o tipo criado pela ponte da ADR-061 (`db.CodeSymbolNodeType`); os demais tipos mantêm o comportamento anterior |
| O `internal/store` não pode importar o `internal/db` | repetir o literal `"code_symbol"` com comentário | `internal/db` importa `internal/store` — o import viraria ciclo |
| No SQLite o `graph_nodes` não tem `repository` | busca sem filtro de repositório | é a identidade do engine (`internal/db/schema.go:47`) |

Open questions: none

## User Stories

- Como dono do vault, quero que o `mem doctor` não accuse links quebrados que existem, para que o Health Score signifique alguma coisa.
- Como dono do vault, quero que a rotina de correção não apague o code graph, para não perder o trabalho de indexação.

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Where |
| --- | --- | --- |
| DDL-1 | AC1 — nó de código não é dead link | `internal/db/graph.go:277` · `internal/store/doctor.go:116` |
| DDL-2 | AC2 — alvo que não é nota nem código continua morto | `internal/db/doctor_code_nodes_test.go:75` · `internal/store/doctor_code_nodes_test.go:118` |
| DDL-3 | AC3 — a correção não apaga o code graph | `internal/db/doctor_code_nodes_test.go:100` · `internal/store/doctor_code_nodes_test.go:196` |
| DDL-4 | AC4 — a correção apaga a órfã | `internal/db/doctor_code_nodes_test.go:100` · `internal/store/doctor_code_nodes_test.go:196` |
| DDL-5 | AC5 — escopo de repositório no Postgres | `internal/store/doctor.go:116` |
| DDL-6 | AC6 — os testes morrem contra o código antigo | `internal/db/doctor_code_nodes_test.go:35` |
