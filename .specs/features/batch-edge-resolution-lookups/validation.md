# Validation: batch-edge-resolution-lookups

- **Feature**: batch-edge-resolution-lookups (ISSUE-034 §"Um ponto ficou de fora")
- **Date**: 2026-10-29
- **Spec**: [spec.md](./spec.md)
- **Verdict**: PASS

## Validation: PASS

**Result**: PASS

A resolução das arestas deixou de custar **uma query por destino** e passou a
custar **um punhado** — com o **mesmo resultado**, provado por equivalência
caso a caso e pela contagem de queries que chegam ao servidor.

### O que o teste de equivalência prova

`internal/codeast/preload_test.go` — `TestPreloadDaResolucaoEquivaleAoCaminhoPorDestino`:
8 arestas cobrindo **todos** os caminhos da resolução (nível 1 por qualified name
único, nível 2 por nome único, nível 2 **ambíguo**, nível 3 sem destino, o
**filtro de kinds** que exclui um símbolo, e o **lookup escopado por arquivo**
do `imports`), cada uma comparada entre o caminho por destino e o caminho com
preload.

O teste **fixa o resultado esperado** de cada aresta — sem isso a equivalência
seria verde por vacuidade (se os dois caminhos não resolvessem nada, dariam
iguais):

| Aresta | Resultado esperado |
| :--- | :--- |
| `Alfa` (qualified name único) | aresta · `EXTRACTED` · conf 1.0 |
| `Beta` (dois símbolos com o mesmo nome) | **incerta** · conf 0.5 |
| `Orfao` (nome único) | aresta · `INFERRED` · conf 0.8 |
| `NaoExiste` | **incerta** · conf 0.3 |
| `Gamma` (tipo) | aresta · `EXTRACTED` · conf 1.0 |
| `Alfa` como destino de `uses_type` | **incerta** · conf 0.3 (o kind exclui) |
| `Importado` (mesmo arquivo) | aresta · `EXTRACTED` · conf 1.0 |
| `Importado` (outro arquivo) | aresta · `EXTRACTED` · conf 1.0 (nível 1) |

**Prova de que não é verde vacuamente:** com o preload mutado (o corte de 3
candidatos → 1), o teste morre com o sintoma exato — o caso ambíguo deixa de ser
ambíguo e vira uma aresta `INFERRED`:

```
--- FAIL: TestPreloadDaResolucaoEquivaleAoCaminhoPorDestino (0.11s)
    preload_test.go:101: aresta 1 (Beta): com preload="aresta dst=2 kind=calls conf=0.8 status=INFERRED linha=11";
      sem preload="incerta dst=\"Beta\" kind=calls conf=0.5 motivo=\"ambíguo: 2 símbolos com o nome \\\"Beta\\\"\""
```

### O que o teste no Postgres prova

`internal/codeast/preload_postgres_test.go` — `TestPreloadPostgresReduzAsQueriesDaResolucao`
(PostgreSQL **real**, banco descartável do `internal/pgtest`, com o
`pgtest.Recorder` contando o SQL que chega ao servidor), 10 destinos distintos:

| Caminho | SELECTs sobre `code_symbols` |
| :--- | :--- |
| por destino (o de antes) | **≥ 10** (um por destino) |
| o preload | **≤ 2** (um punhado) |
| durante a resolução, com o cache cheio | **0** |

### A medição no vault real (Neon, `repo_e4c8f3b1a2d5`)

Code graph **zerado** (`code_files`, `code_symbols`, `code_edges`,
`code_edges_uncertain` apagados) e repopulado do zero, com o mesmo escopo:

| | Antes (por destino) | Depois (com preload) |
| :--- | :--- | --- |
| **Fase de resolução** (entre o fim do parse e o flush) | **~20 min** (~4 500 queries) | **< 1 min** |
| **População completa** (code graph do zero até a ponte) | **2 173 s (36,2 min)** | **1 226 s (20,4 min)** |

O grafo final é o mesmo formato: **381 arquivos · 7 418 símbolos** e a ponte com
**39 376 arestas** em 293 documentos; o `mem index` fechou com
`293 documentos processados (4 indexados, 289 em cache, 0 podados, 0 falhos)`.

### AC por AC

| AC | Status | Evidência |
| --- | --- | --- |
| AC1 — punhado de queries | PASS | `internal/codeast/preload_postgres_test.go` (≤2 no preload) + a medição no vault |
| AC2 — resultado idêntico | PASS | `internal/codeast/preload_test.go` (8 arestas, resultado esperado fixado) |
| AC3 — ambiguidade idêntica | PASS | o caso "Beta" (2 candidatos → incerta, conf 0.5) no mesmo teste |
| AC4 — fallback por destino | PASS | `lookup`/`lookupInFile` consultam o cache primeiro e caem na query individual (`internal/codeast/pipeline.go`) |
| AC5 — contagem de queries no servidor + prova de não-vacuidade | PASS | o teste do `pgtest.Recorder` e a mutante reproduzidos acima |

### Gate

- `go test -count=1 -run TestPreload ./internal/codeast/` → exit 0 (3 testes)
- `MY_MEMORY_PG_TEST=1 go test -count=1 -run TestPreloadPostgres ./internal/codeast/` → exit 0 (Postgres real)
- `go test -count=1 ./...` → exit 0 (26 pacotes `ok`)
- `go build ./...` → exit 0 · `go vet ./...` → exit 0
- `gofmt -l` na cópia LF de `internal/codeast/pipeline.go` e dos dois testes → vazio
- `validate_spec.py` 0 erros · `validate_tasks.py` 0 erros · `validate_state.py` 0 erros

### Riscos / pendências

- **Nenhum ADR novo**: nenhuma query, coluna, índice ou regra de resolução
  mudou — só o **número** de queries. A ADR-061 (code graph) continua válida.
- O **caminho legado por nome** (`resolveAndInsertEdges`, que só roda para
  linguagens **sem** backend AST) não foi tocado: no build padrão (motor Go)
  ele não é exercitado.
- O teto de **500 valores por statement** vem do limite de parâmetros do
  engine (SQLite 32 766; Postgres 65 535). Acima do teto o lote é partido — o
  conteúdo gravado não muda.
- A **ponte** (`references_code`) segue sendo o maior bloco isolado da
  população (≈2,5 min): ela grava 39 376 arestas em 293 documentos, 2
  statements por documento.
