# Validation: batch-code-bridge-writes

**Spec**: [spec.md](./spec.md)
**Tasks**: [tasks.md](./tasks.md)
**Date**: 2026-10-29
**Branch**: `develop` · **Base**: `adfb8cc` · **Diff**: working tree (sem commit — o git é do gestor)

## Validation: PASS

**Result**: PASS

Escopo verificado: 21 requisitos (BR-01..05, SYM-01..05, EDG-01..04, UNC-01..04, PRV-01..03), as **três** metades do defeito (a ponte `notes↔código`, o pipeline de símbolos/arestas e o bucket de arestas incertas), a prova de que os testes novos **falham contra o comportamento antigo** (quatro mutantes, sintoma exato), o Postgres **real** (banco descartável do `internal/pgtest`) e o gate completo.

**Arquivos tocados** (todos UTF-8 sem BOM; o índice normaliza para LF via `.gitattributes`):

| Arquivo | Mudança |
| ------- | ------- |
| `internal/db/code_bridge.go` | +150 / −45 — a ponte grava **1 INSERT de nós + 1 INSERT de arestas por documento** (era 2 statements por par documento→símbolo) |
| `internal/db/code_migration.go` | +430 — `UpsertCodeSymbols` (1 INSERT multi-linha + 1 SELECT de ids por arquivo), `UpsertCodeEdges` e `InsertCodeEdgeUncertains` (1 INSERT multi-linha por arquivo), dedup dos lotes e as construções de statement |
| `internal/codeast/cache.go` | +25 / −13 — `PersistSymbols` passa a gravar o arquivo inteiro em um lote |
| `internal/codeast/pipeline.go` | +175 / −60 — `FlushEdges` grava os DOIS buckets (`code_edges` e `code_edges_uncertain`) em 1 statement por arquivo; a resolução foi separada da escrita (`resolveEdgeRow` → `resolvedEdge`, `uncertainRow`) |
| `internal/codeast/main_test.go` | +33 — `TestMain` com o guard de storage (`pgtest.Capture` + `Neutralize`), o mesmo de `internal/db/main_test.go`: sem ele o teste de integração do codeast não alcança o Postgres descartável |
| `internal/db/code_bridge_batch_test.go` | +400 — contagem de statements, equivalência com o caminho por símbolo, idempotência e forma do SQL |
| `internal/db/code_batch_postgres_test.go` | +400 — ponte, símbolos, arestas e **arestas incertas** no Postgres real, com o `pgtest.Recorder` contando o SQL que chega ao servidor + a medição do custo do `UpsertCodeFile` |
| `internal/codeast/batch_test.go` | +420 — equivalência de linhas/ids, mapeamento de ids, idempotência, estado das arestas e do bucket incerto |
| `internal/codeast/batch_postgres_test.go` | +185 — o `FlushEdges` no Postgres real: 1 statement por arquivo em cada bucket |
| `.specs/features/batch-code-bridge-writes/` | `spec.md`, `tasks.md`, `validation.md` |

---

## Números: antes e depois (268 ms por round-trip no Neon)

### Metade 1 — a ponte `notes↔código`

| | Antes | Depois |
| - | ---- | ----- |
| Statements por documento (N símbolos casados) | **2 × N** (1 nó + 1 aresta por par) | **2** (1 nós + 1 arestas), constante |
| Statements no vault (~276 docs, ~34 000 arestas `references_code`) | **≈68 000** | **552** |
| Tempo no Neon (268 ms/round-trip) | **≈4,7 h** (medi pelo dono) | **≈2,5 min** (552 × 268 ms) |
| `BuildCodeBridgeForDocument` (1 documento, N=123) | **246 statements ≈ 66 s** | **2 statements ≈ 0,5 s** (+ as 2 queries de carga, que não mudaram) |

### Metade 2 — o pipeline de símbolos/arestas

| | Antes | Depois |
| - | ---- | ----- |
| Round-trips por símbolo | **2** (INSERT + SELECT do id — o `lib/pq` não tem `LastInsertId`) | **1 INSERT + 1 SELECT por ARQUIVO** |
| Round-trips por arquivo (N símbolos, M arestas) | **2N + M + 2** (`code_files` não muda) | **5** (2 de símbolos + 2 de `code_files` + 1 de arestas) |
| Round-trips no repo (~6 750 símbolos, ~375 arquivos .go, ~1 aresta por símbolo) | **≈13 500 + 6 750 + 750 ≈ 21 000** | **≈1 875** |
| Tempo no Neon (268 ms/round-trip) | **≈2 h** (0,78 símbolos/s medido pelo dono) | **≈8,4 min** |

### Metade 3 — o bucket de arestas incertas (complemento de 29/10)

Medição do dono **em produção** depois da primeira entrega (Neon, run de 08:24→08:41): o bucket era gravado a **~1,6 aresta/s** (1 statement por aresta) → **≈110 min** só para popular `code_edges_uncertain` do repo inteiro, contra ~5 min dos símbolos. A decisão original (A11, "volume residual") partia do **código**, não da **medição**: o bucket é pequeno em linhas, mas cada linha custava 1 round-trip.

| | Antes | Depois |
| - | ---- | ----- |
| Statements por arquivo (M arestas incertas) | **M** (1 por aresta) | **1** (multi-linha), constante |
| Round-trips do bucket no repo (~374 arquivos) | **~1 statement por aresta incerta** (≈10,5 mil arestas pela taxa de 1,6/s) | **~374** (1 por arquivo) |
| Tempo no Neon | **≈110 min** (medido) | **≈1,7 min** (374 × 268 ms) |
| Taxa | ~1,6 aresta/s | o gargalo migra do bucket para as **queries de resolução** (cacheadas por destino) |

### O code stage inteiro, depois do complemento

| Cenário | Round-trips | Tempo no Neon (268 ms) |
| ------- | ----------- | ----------------------- |
| Primeira população (~374 arquivos) | ~374 × 6 (2 `code_files` + 2 símbolos + 1 arestas + 1 incertas) ≈ **2 244** | **≈10 min** |
| Rotina com cache quente (tudo indexado) | a ponte: ~276 docs × 2 ≈ **552** (o code stage não escreve nada) | **≈2,5 min** |

### Custo real do `UpsertCodeFile` (a pergunta do dono)

**2 round-trips por arquivo no Postgres** — o `INSERT` e um `SELECT` extra para resolver o id, porque o `lib/pq` não implementa `LastInsertId` (`internal/db/code_migration.go:220`). Medido com o `pgtest.Recorder`:

```
UpsertCodeFile no Postgres: 2 round-trip(s) por arquivo
  SELECT id FROM code_files WHERE repository = $1 AND path = $2
```

No **SQLite** são **1** (o `LastInsertId` do modernc devolve o id da linha gravada). O código **não foi mexido** (decisão A13): o custo só pesa na primeira população (~375 × 2 ≈ 750 round-trips ≈ 3,4 min) e **não** pesa no `mem index` de rotina, em que o cache quente faz o arquivo ser pulado antes de qualquer escrita.

---

## Evidência por requisito

| Req | Critério | Evidência | Comando | Saída real |
| --- | -------- | --------- | ------- | ---------- |
| BR-01 | 2 statements por documento, e não 2×N | `internal/db/code_bridge.go:263` (nós), `:266` (arestas), `:442`/`:458` (escrita em lote) | go test -count=1 -run PonteLote ./internal/db/... | `ok …/internal/db` — 8 símbolos casados → **2 statements** (`internal/db/code_bridge_batch_test.go:60`) |
| BR-02 | Nós gravados **antes** das arestas (FK `graph_edges → graph_nodes`) | `internal/db/code_bridge.go:263` antes de `:266` | go test -count=1 -run TestPonteLoteGravaDoisStatementsPorDocumento ./internal/db/... | o 1º statement capturado é `graph_nodes`, o 2º é `graph_edges` (`internal/db/code_bridge_batch_test.go:79`) |
| BR-03 | Idempotência: 2× não duplica e não muda o grafo | `internal/db/code_bridge.go:269`/`:270` (contagens) | go test -count=1 -run 'TestPonteLoteIdempotente\|TestPonteLoteEquivaleAoCaminhoPorSimbolo' ./internal/db/... | grafo final idêntico ao do caminho por símbolo (`internal/db/code_bridge_batch_test.go:129`) e 2 statements nas duas rodadas (`:193`) |
| BR-04 | Mesmo SQL por engine (`INSERT OR IGNORE` / `ON CONFLICT`) | `internal/db/code_bridge.go:386` (nós), `:414` (arestas) | go test -count=1 -run TestPonteLoteConstrutoresDeStatement ./internal/db/... | `ok` — os literais (`'code_symbol'`, `'references_code'`) intactos depois do `Bind`, e `$1..$N` numerado (`internal/db/code_bridge_batch_test.go:225`) |
| BR-05 | Lote acima do teto é partido (nós antes das arestas) | `internal/db/code_bridge.go:59` (`bridgeMaxBatchRows = 2000`), `:442`/`:458` | leitura do código + `go vet ./internal/db/...` | `exit=0` — o corte é de **parâmetros**, não de conteúdo |
| SYM-01 | 1 statement de escrita por arquivo | `internal/db/code_migration.go:409` (`UpsertCodeSymbols`), `:429` (o INSERT), `:434` (o SELECT de ids) | MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgres ./internal/db/... | `ok` — `INSERTs em code_symbols = 1` (`internal/db/code_batch_postgres_test.go:162`) |
| SYM-02 | O id devolvido é o do símbolo `i` (é dele que as arestas dependem) | `internal/codeast/cache.go:186` (o lote na ordem dos símbolos) | go test -count=1 -run TestLoteSimbolosMapeiaIds ./internal/codeast/... | `ok` — o id de cada posição aponta para o símbolo correspondente (`internal/codeast/batch_test.go:135`) |
| SYM-03 | Mesmoas linhas e mesmos ids que o caminho por símbolo | `internal/db/code_migration.go:335` (11 colunas + o `ON CONFLICT` de `:349`), `:369` (a resolução) | go test -count=1 -run TestLoteSimbolosEquivaleAoCaminhoPorSimbolo ./internal/codeast/... | `ok` — bancos comparados linha a linha: ids e conteúdo idênticos (`internal/codeast/batch_test.go:74`) |
| SYM-04 | Rodar 2× não duplica nem muda o `ON CONFLICT` | `internal/db/code_migration.go:349` (o alvo é o mesmo de `:244`) | go test -count=1 -run 'TestLoteSimbolosIdempotente\|LotePostgresSimbolos' ./internal/... | `ok` — 4 símbolos, 2 rodadas, 4 linhas, os mesmos ids (`internal/codeast/batch_test.go:173`) e no Postgres (`internal/db/code_batch_postgres_test.go:162`) |
| SYM-05 | Chave duplicada no lote → uma linha, payload do último | `internal/db/code_migration.go:314` (`dedupCodeSymbolRows`) | go test -count=1 -run TestLoteSimbolosChaveDuplicada ./internal/codeast/... | `ok` — 2 linhas no banco, os dois ids iguais, `name = AlphaV2` (`internal/codeast/batch_test.go:212`) |
| EDG-01 | 1 statement de arestas por arquivo | `internal/codeast/pipeline.go:393` (`FlushEdges`), `:738` (caminho legado), `internal/db/code_migration.go:543` | MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgresArestas ./internal/db/... | `ok` — `INSERTs em code_edges = 1` (`internal/db/code_batch_postgres_test.go:233`) |
| EDG-02 | Mesmo estado de `code_edges` (menor posição, payload do último) | `internal/db/code_migration.go:475` (a fusão), `:511`/`:524` (o `CASE WHEN` do `ON CONFLICT`) | go test -count=1 -run TestLoteArestasEstadoIgualAoCaminhoPorAresta ./internal/codeast/... | `ok` — as linhas do lote e as do caminho por aresta são idênticas (`internal/codeast/batch_test.go:262`) |
| EDG-03 | `confidence < 0.5` continua recusado com o mesmo texto | `internal/db/code_migration.go:551`/`:552` (idêntico a `:596`/`:597`) | go test -count=1 -run TestLoteArestasRecusaBaixaConfianca ./internal/codeast/... | `ok` — erro `UseInsertCodeEdgeUncertain para confidence < 0.5` (`internal/codeast/batch_test.go:328`) |
| PRV-01 | Os testes novos **falham** contra o comportamento antigo | três mutantes (abaixo) | mutante + go test | os testes morrem com o sintoma exato do defeito |
| PRV-02 | Postgres real: `INSERT` multi-linha com `ON CONFLICT` aceito pelo servidor | `internal/db/code_batch_postgres_test.go:37` (harness + `pgtest.Recorder`), `:72`, `:162`, `:233` | MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgres ./internal/db/... | `ok …/internal/db 37.596s` — a integração rodou de verdade (banco descartável `mymemory_pgtest_*`) |
| PRV-03 | Gate completo verde | aba Gate abaixo | go test/build/vet/gofmt | todos com `exit=0` |
| UNC-01 | 1 statement multi-linha por arquivo no bucket incerto | `internal/db/code_migration.go:720` (`InsertCodeEdgeUncertains`), `:688` (o construtor), `:701` (o `ON CONFLICT`) | MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgresIncertas ./internal/db/... | `ok` — `INSERTs em code_edges_uncertain = 1` com 3 linhas no lote (`internal/db/code_batch_postgres_test.go:273`) |
| UNC-02 | O `FlushEdges` grava os DOIS buckets em 1 statement por arquivo | `internal/codeast/pipeline.go:403` (`code_edges`) e `:408` (`code_edges_uncertain`), agrupados por arquivo em `:377` | MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgresFlushEdges ./internal/codeast/... | `ok` — 2 arquivos → **2 + 2** statements (`internal/codeast/batch_postgres_test.go:53`) |
| UNC-03 | Mesma chave → uma linha, payload do último; mesmo `ON CONFLICT`; idempotente | `internal/db/code_migration.go:664` (a fusão), `:701` (o `ON CONFLICT`, igual ao de `:634`) | go test -count=1 -run TestLoteIncertas ./internal/codeast/... | `ok` — os dois bancos (caminho por aresta × caminho em lote) ficam com as mesmas 2 linhas (`internal/codeast/batch_test.go:330`) |
| UNC-04 | Prova no Postgres real, com o número de statements por arquivo | `internal/codeast/batch_postgres_test.go:37` (o contador), `:53` (o teste) | MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgresFlushEdges ./internal/codeast/... | `ok …/internal/codeast 16.4s` — banco descartável, `Recorder` contando o SQL que chega ao servidor |
| EDG-04 | Confirmação do número: 1 statement por arquivo nas arestas **resolvidas** | `internal/codeast/pipeline.go:403` | MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgresFlushEdges ./internal/codeast/... | `ok` — `INSERTs em code_edges = 2` para 2 arquivos (`internal/codeast/batch_postgres_test.go:119`) |

---

## Prova contra o defeito (comportamento antigo)

Três mutantes, cada um **voltando ao 1 statement por linha** (o defeito original) sem tocar em mais nada. Em todos os casos os testes novos **morrem** com o sintoma exato.

### Mutante 1 — a ponte (`bridgeMaxBatchRows = 1` ⇒ 1 statement por linha)

```
--- FAIL: TestPonteLoteGravaDoisStatementsPorDocumento (0.14s)
    code_bridge_batch_test.go:74: statements de escrita = 16; want 2 (1 graph_nodes + 1 graph_edges)
--- FAIL: TestPonteLoteNumeroDeStatementsConstante (0.44s)
        code_bridge_batch_test.go:120: statements = 10 com 5 símbolos; want 2 (constante, não O(N))
        code_bridge_batch_test.go:120: statements = 50 com 25 símbolos; want 2 (constante, não O(N))
--- FAIL: TestPonteLoteIdempotente (0.11s)
        code_bridge_batch_test.go:206: statements (rodada 1) = 8; want 2
FAIL	github.com/FelipeMiiller/my-memory/internal/db	1.142s
```

O sintoma é o do defeito medido: **16 statements para 8 símbolos** (2×N) e **50 para 25** — a contagem cresce com N. Depois do revert: `ok …/internal/db`.

### Mutante 2 — os símbolos (`codeMaxBatchRows = 1` ⇒ 1 INSERT por símbolo), no Postgres real

```
    code_batch_postgres_test.go:197: INSERTs em code_symbols = 5; want 1 (um statement por arquivo)
--- FAIL: TestLotePostgresSimbolosGravaUmStatementPorArquivo (13.65s)
FAIL	github.com/FelipeMiiller/my-memory/internal/db	24.789s
```

### Mutante 3 — as arestas (`codeMaxBatchRows = 1` ⇒ 1 INSERT por aresta), no Postgres real

```
    code_batch_postgres_test.go:270: INSERTs em code_edges = 2; want 1 (um statement por arquivo)
--- FAIL: TestLotePostgresArestasGravaUmStatementPorArquivo (11.65s)
FAIL	github.com/FelipeMiiller/my-memory/internal/db	11.823s
```

### Mutante 4 — o bucket de arestas incertas (`codeMaxBatchRows = 1` ⇒ 1 statement por aresta), no Postgres real

```
    code_batch_postgres_test.go:273: INSERTs em code_edges_uncertain = 2; want 1 (um statement por arquivo)
--- FAIL: TestLotePostgresIncertasGravaUmStatementPorArquivo (12.17s)
FAIL	github.com/FelipeMiiller/my-memory/internal/db	12.330s
```

E, no **pipeline** (2 arquivos × 2 arestas incertas cada):

```
    batch_postgres_test.go:126: INSERTs em code_edges_uncertain = 4; want 2 (um statement por arquivo)
    batch_postgres_test.go:165: INSERTs em code_edges_uncertain (2a vez) = 4; want 2
--- FAIL: TestLotePostgresFlushEdgesGravaUmStatementPorArquivo (17.62s)
FAIL	github.com/FelipeMiiller/my-memory/internal/codeast	17.783s
```

O sintoma é o do gargalo medido: **4 statements** (1 por aresta) em vez de **2** (1 por arquivo). Depois do revert, a mesma rodada verde (`ok …/internal/codeast`).

Depois dos quatro reverts, a mesma rodada verde (`ok …/internal/db`, `ok …/internal/codeast`).

---

## Postgres real (banco descartável do `internal/pgtest`)

O job rodou de verdade: `MY_MEMORY_PG_TEST=1` com `MY_MEMORY_PG_URL` preservada (o harness `pgtest.New` cria e derruba o `mymemory_pgtest_*`; o Postgres do dono nunca é alvo).

```
MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgres -v ./internal/db/
--- PASS: TestLotePostgresPonteGravaDoisStatementsPorDocumento (16.08s)
--- PASS: TestLotePostgresSimbolosGravaUmStatementPorArquivo (11.98s)
--- PASS: TestLotePostgresIncertasGravaUmStatementPorArquivo (10.62s)
--- PASS: TestLotePostgresArestasGravaUmStatementPorArquivo (11.00s)
ok  	github.com/FelipeMiiller/my-memory/internal/db	49.856s

MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgresFlushEdges -v ./internal/codeast/
--- PASS: TestLotePostgresFlushEdgesGravaUmStatementPorArquivo (16.26s)
ok  	github.com/FelipeMiiller/my-memory/internal/codeast	16.424s
```

O `pgtest.Recorder` grava o SQL que chega ao servidor: ele é o que prova que o `INSERT` multi-linha com `ON CONFLICT` foi **aceito pelo Postgres** e que o número de statements por documento/arquivo é 1+1 / 1 / 1 em cada bucket.

O `internal/codeast` não tinha `TestMain` (o `pgtest.Capture()` que autoriza o teste de integração vivia só em `internal/db` e `internal/store`): sem o guard de storage, o teste de integração do codeast nem alcançava o banco descartável. O guard novo (`internal/codeast/main_test.go:28`) é o mesmo de `internal/db/main_test.go` — capturar a URL, neutralizar as variáveis de storage e restaurá-las no fim.

---

## Gate

| Gate | Comando | Saída |
| ---- | ------- | ----- |
| Testes (suíte completa) | `$env:MY_MEMORY_PG_TEST='1'; go -C D:\repository\my-memory test -count=1 ./...` | `exit=0` — 26 pacotes `ok`, nenhum `FAIL` (`cmd/mem` 96.7s, `internal/db` 98.6s, `internal/store` 40.3s, `internal/codeast` 19.4s — a integração com Postgres rodou de verdade) |
| Build | `go -C D:\repository\my-memory build ./...` | `exit=0` |
| Vet | `go -C D:\repository\my-memory vet ./...` | `exit=0` |
| Formatação | `gofmt -l` na **cópia LF** dos 7 arquivos `.go` tocados (`D:\.minimax\scratch\subagents\issue-bridge-batch\gofmt-check`) | saída vazia — formatação limpa |
| Spec | `python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/batch-code-bridge-writes/spec.md` | `0 error(s), 0 warning(s)` · `exit=0` |
| Tasks | `python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py .specs/features/batch-code-bridge-writes/tasks.md` | `0 error(s), 0 warning(s)` · `exit=0` |
| Estado | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py batch-code-bridge-writes --root D:\repository\my-memory` | `0 error(s)` · `exit=0` |

---

## Decisões que ficaram registradas (e o porquê)

- **Rota (b) para os ids: `INSERT` multi-linha + 1 `SELECT` por arquivo** (A4). O `RETURNING` foi **medido** nos dois engines (SQLite 3.50 devolve as linhas na ordem do `VALUES`; o Postgres também, mas a ordem **não é garantida** pelo contrato) e seria 1 round-trip em vez de 2 — mas o `INSERT ... RETURNING` via `QueryContext` não está coberto pelo build CGO (`mattn/go-sqlite3`) que o CI roda, e o casamento por chave em Go é o mesmo algoritmo nos dois engines. O custo são ~375 round-trips a mais num code stage inteiro.
- **A chave duplicada é removida ANTES do statement** (A6). O Postgres **rejeita** `ON CONFLICT DO UPDATE` que afete a mesma linha duas vezes (`ON CONFLICT DO UPDATE command cannot affect row a second time` — medido no Postgres descartável); o SQLite aceita. Sem a remoção, o mesmo código teria comportamento diferente nos dois engines. A fusão reproduz o caminho por símbolo: uma linha, payload do último, mesmo id.
- **O `id` da aresta vem da linha gravada, e não do `LastInsertId`** (A5, A10). Medido no SQLite (modernc): depois de um upsert que **atualiza**, o `LastInsertId` devolve o `last_insert_rowid` **velho** (o da última linha inserida), não o da linha atualizada — ou seja, o caminho por símbolo pode devolver o id de **outro** símbolo. O lote resolve o id pela chave (`kind`, `qualified_name`, `start_line`), que é o que as arestas precisam. Essa é a única diferença de comportamento em relação ao caminho antigo, e ela é uma **correção**: o id devolvido passa a ser sempre o da linha do símbolo.
- **Nenhum ADR novo** (A9). Não há mudança de schema, de driver nem de arquitetura: é a **forma** de emitir as mesmas escritas. `go.mod`/`go.sum` intocados.
- **O teto de 2000 linhas por statement** (A3) é de **parâmetros**, não de desempenho: `MAX_VARIABLE_NUMBER=32766` no SQLite (medido no driver deste repo) e 65 535 no Postgres; o insert de símbolo usa 11 colunas → 2000 × 11 = 22 000. Documentos/arquivos acima do teto são partidos em pedaços, e o conteúdo gravado não muda.
- **O casamento da ponte não foi tocado** (restrição do dono): literais por qualified name, por path e o match fraco por nome continuam exatamente como estavam — a feature muda só **como as escritas são emitidas** (`internal/db/code_bridge.go:146` mantém o bloco de casamento original).
- **`UpsertCodeFile` e `code_edges_uncertain` ficaram como estavam** — ~2 round-trips por arquivo e um volume residual de arestas incertas, irrelevantes ao lado do resto.

> A segunda linha é a que a **medição de 29/10 refutou**: o bucket de incertas é justamente o que faltava. A decisão original está registrada aqui como histórico (A11/A11b na spec) — não apagada.

- **O bucket de arestas incertas entra no escopo, com o mesmo `ON CONFLICT`** (A12). A chave `UNIQUE(src_symbol_id, dst_candidate_name, kind)` **existe nos dois engines** (`internal/db/schema.go:141` e `internal/store/postgres.go:153`) — nenhum índice novo, nenhuma coluna nova, nenhum schema alterado. A duplicata do próprio lote é fundida antes do statement (o Postgres rejeita `ON CONFLICT DO UPDATE` da mesma linha duas vezes), com o payload do **último** — que é o que o `ON CONFLICT` grava em todas as colunas.
- **A resolução parou de gravar no meio do caminho** (UNC-02). `resolveEdgeRow` devolve um `resolvedEdge` (ou `code_edges`, ou `code_edges_uncertain`) e quem grava é o `FlushEdges`, em 1 statement por arquivo em cada bucket. O caminho por aresta (`resolveEdge`, para o processor não-batched) grava 1 statement por aresta, como antes — é o fallback, não o caminho de produção.
- **`UpsertCodeFile` não foi mexido** (A13). O custo real é **2 round-trips por arquivo no Postgres** (o `INSERT` + o `SELECT` do id, porque o `lib/pq` não implementa `LastInsertId`) e 1 no SQLite; o teste que mede isso (`internal/db/code_batch_postgres_test.go:244`) falha só se o custo **crescer**. Reduzir para 1 exigiria `RETURNING id` — decisão do dono, não do agente.
- **`internal/codeast` ganhou `TestMain`** (`internal/codeast/main_test.go:28`): sem o guard de storage, o teste de integração do codeast não alcança o banco descartável (o `pgtest.Capture()` vivia só em `internal/db` e `internal/store`). O guard é o mesmo dos outros dois pacotes: capturar a URL, neutralizar as variáveis de storage e restaurá-las no fim.
- **`internal/watcher/indexer.go`** não foi tocado: ele consome `BuildCodeBridgeForDocument`, cuja assinatura não mudou — o ganho vem sozinho.

## Pendências para o gestor

- `.specs/ISSUES.md` e `.specs/STATE.md` não foram tocados (fora da propriedade desta execução) — se a issue do code stage remoto deve ser atualizada, é o gestor.
- Nenhum commit e nenhum push — o git é do gestor.
- Os arquivos de outro agente que estavam no working tree (`cmd/mem/main.go`, `docs/CLI_GUIDE.md`, `.agents/skills/tlc-spec-driven/**`, `.specs/features/fix-*/**`) **não** foram tocados por esta execução.
