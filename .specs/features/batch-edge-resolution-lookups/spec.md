# Feature: batch-edge-resolution-lookups

- **Spec ID**: `batch-edge-resolution-lookups`
- **Created**: 2026-10-29
- **Owner**: Felipe Miiller
- **Source**: ISSUE-034 §"Um ponto ficou de fora" — `.specs/ISSUES.md`
- **Related**: ADR-061 (code graph, CG-05/CG-08) · `.specs/features/batch-code-bridge-writes/` (a correção anterior, mesma classe de defeito)

## Problem Statement

A correção da ISSUE-034 pôs em lote as **escritas** do code graph (símbolos,
arestas, bucket incerto e ponte). Ficou de fora o lado das **leituras** da
resolução: o `edgeResolver` fazia **1 `SELECT` por destino** de aresta
(`internal/codeast/pipeline.go`, `lookup` e `lookupInFile`), cacheado por lote.

Medido no Neon (≈268 ms/round-trip), a **resolução** das arestas do repo
custou ~4 500 queries ≈ **20 min** — quase a metade dos 36 min da primeira
população do code graph. O mesmo formato de defeito da ISSUE-034 (uma escrita/
leitura por item), agora do lado da leitura.

### O que muda

Antes de resolver as arestas pendentes, o resolver busca **todos** os destinos
necessários com um punhado de queries (`IN (...)`), preenchendo o **mesmo**
cache que o caminho por destino usaria — chave idêntica, ids na mesma ordem e
cortados em 3.

**POR QUE É SEGURO (a chave da mudança):** a resolução acontece só depois que
**todos** os arquivos do lote foram indexados (`internal/codeast/pipeline.go`,
§Run: *"As arestas acumuladas são resolvidas só agora: com todos os arquivos do
lote indexados, uma chamada para um símbolo de outro arquivo resolve pelo
qualified name (EXTRACTED) em vez de ficar sem destino"*). O conjunto de
símbolos disponível na resolução é, portanto, **o mesmo** que a query por
destino encontraria — um snapshot no início da resolução é equivalente.
Se alguma chave não vier no preload, o resolver **cai no caminho por destino**
(uma query) como antes: o preload é uma otimização, não uma mudança de
semântica.

## Goals

- [x] A resolução SHALL emitir um punhado de queries, em vez de uma por destino.
- [x] O resultado da resolução SHALL ser **idêntico**: mesmas arestas, mesmos ids de destino, mesmo status epistêmico, mesma contagem de candidatos (e a mesma mensagem de ambiguidade).
- [x] O caminho por destino SHALL continuar existindo como fallback.
- [x] O teto de valores por statement SHALL respeitar o limite de parâmetros do engine (SQLite 32 766; Postgres 65 535).
- [x] Teste de equivalência (com o resultado esperado fixado) e teste de contagem de queries no Postgres real.
- [x] Prova de que o preload não é verde vacuamente (mutante).

## Out of Scope

- **O caminho legado por nome** (`resolveAndInsertEdges`) — só roda para linguagens **sem** backend AST; no build padrão (motor Go) ele não é exercitado.
- **Mudar a semântica da resolução** (os 3 níveis, o filtro de kinds, o corte de 3 candidatos) — nada disso muda.
- **Mudar schema, tabelas ou ADR** — nenhum ADR novo é necessário (ver A9).
- **A ponte `references_code`** — já está em lote desde a ISSUE-034.

## Acceptance Criteria

1. WHEN a resolução roda THEN os destinos de todas as arestas pendentes SHALL ter sido buscados com no máximo 2 queries por grupo de 500 valores.
2. IF uma aresta é resolvida THEN o resultado (aresta resolvida ou linha incerta, com id, status epistêmico, confiança e motivo) SHALL ser idêntico ao do caminho por destino.
3. IF a contagem de candidatos por nome é maior que 1 THEN a aresta SHALL ir para o bucket das incertas com o mesmo texto de ambiguidade.
4. IF um destino não está no preload THEN o resolver SHALL fazer a query individual (fallback) e devolver o mesmo resultado.
5. The regression tests SHALL fail against the pre-fix code and pass after the fix, and the PostgreSQL test SHALL count the queries that reach the server.

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale |
| --- | --- | --- |
| O conjunto de símbolos está completo quando a resolução começa | snapshot no início da resolução | o `Run` resolve as arestas só depois de indexar todos os arquivos do lote (`pipeline.go`, §Run) |
| O corte de candidatos por destino é 3 | manter 3 | é o que a query por destino fazia (`LIMIT 3`) — e o texto de ambiguidade depende desse número |
| O teto de 500 valores por statement é seguro | 500 | 2 parâmetros fixos + 1 por valor = ~502 parâmetros: folga larga nos dois engines |
| A ordem das queries do preload não importa para o resultado | ordenar os valores | determinismo: o teste e o comportamento são reprodutíveis |

Open questions: none

## User Stories

- Como dono de um vault grande, quero que a primeira indexação do code graph termine em minutos, não em horas, para poder indexar um repo novo sem medo.
- Como leitor do grafo, quero que a resolução das arestas produza exatamente o mesmo grafo de antes, para não confiar num resultado diferente sem perceber.

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Where |
| --- | --- | --- |
| REL-1 | AC1 — punhado de queries | `internal/codeast/pipeline.go` (`edgeResolver.preload`) |
| REL-2 | AC2 — resultado idêntico | `internal/codeast/preload_test.go` (`TestPreloadDaResolucaoEquivaleAoCaminhoPorDestino`) |
| REL-3 | AC3 — ambiguidade idêntica | `internal/codeast/preload_test.go` (caso "Beta") |
| REL-4 | AC4 — fallback | `internal/codeast/pipeline.go` (`lookup`/`lookupInFile` consultam o cache primeiro) |
| REL-5 | AC5 — contagem de queries no servidor | `internal/codeast/preload_postgres_test.go` (`TestPreloadPostgresReduzAsQueriesDaResolucao`) |
