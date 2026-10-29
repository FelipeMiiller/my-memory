# Tasks: batch-code-bridge-writes

**Spec**: [spec.md](./spec.md)
**Date**: 2026-10-29
**Source**: medição do dono no Postgres remoto (Neon) — ponte `notes↔código` e pipeline de símbolos

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Ponte — montagem do statement (SQL puro) | unit | N linhas → N tuplas no `VALUES`, número de `?` correto, `Bind` numerando `$1..$N` sem tocar literal | internal/db/code_bridge_batch_test.go | go test -count=1 -run PonteLote ./internal/db/... |
| Ponte — contagem de statements (SQLite) | integration | Documento com N símbolos casados → exatamente 2 statements (1 nós + 1 arestas), independentemente do N; nós antes das arestas; estado final igual ao caminho por símbolo | internal/db/code_bridge_batch_test.go | go test -count=1 -run PonteLote ./internal/db/... |
| Ponte — Postgres real | integration | `INSERT` multi-linha com `ON CONFLICT` aceito pelo servidor; número de statements por documento contados pelo `pgtest.Recorder`; idempotência | internal/db/code_batch_postgres_test.go | MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgres ./internal/db/... |
| Símbolos — equivalência de linhas e ids | integration | Caminho em lote × caminho por símbolo: mesmas linhas (mesmas colunas) e mesmos ids para um arquivo com N símbolos | internal/codeast/batch_test.go | go test -count=1 -run LoteSimbolos ./internal/codeast/... |
| Símbolos — mapeamento de ids | integration | O id devolvido na posição `i` é o id do símbolo `i` (lido do banco), e as arestas apontam para ele | internal/codeast/batch_test.go | go test -count=1 -run LoteSimbolos ./internal/codeast/... |
| Símbolos — Postgres real | integration | `INSERT` multi-linha com `ON CONFLICT` + `SELECT` de resolução no servidor; 1 statement de escrita por arquivo | internal/db/code_batch_postgres_test.go | MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgres ./internal/db/... |
| Arestas — flush em lote | integration | Estado final de `code_edges` igual ao caminho por aresta (mesma chave, menor `start_line`/`end_line`, payload do último) | internal/codeast/batch_test.go | go test -count=1 -run LoteArestas ./internal/codeast/... |
| Arestas incertas — equivalência | integration | `code_edges_uncertain` fica igual ao caminho por aresta (mesma chave → uma linha, payload do último) | internal/codeast/batch_test.go | go test -count=1 -run LoteIncertas ./internal/codeast/... |
| Arestas incertas — statements por arquivo (Postgres real) | integration | 1 `INSERT` em `code_edges_uncertain` por arquivo; `FlushEdges` grava 1 statement por arquivo em CADA bucket (2 arquivos → 2 + 2) | internal/codeast/batch_postgres_test.go | MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgres ./internal/codeast/... |
| Custo real do `UpsertCodeFile` | integration | Mede os round-trips por arquivo no Postgres (INSERT + SELECT do id) sem fixar o número | internal/db/code_batch_postgres_test.go | MY_MEMORY_PG_TEST=1 go test -count=1 -run TestCustoRealUpsertCodeFilePostgres ./internal/db/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após a costura da ponte (T1) | go build ./... |
| Quick | Após o lote da ponte (T2) | go test -count=1 -run PonteLote ./internal/db/... |
| Quick | Após os testes da ponte (T3) | go test -count=1 -run PonteLote ./internal/db/... |
| Quick | Após o lote de símbolos (T5) | go test -count=1 -run LoteSimbolos ./internal/codeast/... |
| Quick | Após o lote de arestas (T7) | go test -count=1 -run LoteArestas ./internal/codeast/... |
| Quick | Após o pipeline em lote (T8) | go test -count=1 ./internal/codeast/... |
| Quick | Após o lote do bucket incerto (T11) | go test -count=1 -run 'LoteIncertas\|LotePostgresIncertas' ./internal/... |
| Quick | Após o `FlushEdges` com os dois buckets (T12) | MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgresFlushEdges ./internal/codeast/... |
| Full | Gate final (T14) | go test -count=1 ./... (com MY_MEMORY_PG_TEST=1 e MY_MEMORY_PG_URL preservada) |
| Build | Antes de declarar pronto | go build ./... e go vet ./... |
| Format | Antes de declarar pronto | gofmt -l na cópia LF dos arquivos .go tocados (o working tree local mistura CRLF) |
| Spec | Antes de entregar | python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/batch-code-bridge-writes/spec.md e validate_tasks.py .specs/features/batch-code-bridge-writes/tasks.md |
| State | Antes de entregar | python .agents/skills/tlc-spec-driven/scripts/validate_state.py batch-code-bridge-writes --root D:\repository\my-memory |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2
T2 -> T3
T3 -> T4
T4 -> T5
T5 -> T6
T6 -> T7
T7 -> T8
T8 -> T9
T9 -> T10
T10 -> T11
T11 -> T12
T12 -> T13
T13 -> T14
```

| Fase | Tarefas | Objetivo |
| ---- | ------- | -------- |
| 1 | T1 | Costura de escrita da ponte (interface estreita + construtores puros do statement) |
| 2 | T2 | Ponte grava em lote: 1 `INSERT` de nós + 1 `INSERT` de arestas por documento |
| 3 | T3 | Testes da ponte no SQLite (contagem de statements + equivalência de estado) |
| 4 | T4 | Testes da ponte, dos símbolos e das arestas no Postgres real (pgtest) |
| 5 | T5 | `code_symbols` em lote: 1 `INSERT` multi-linha + 1 `SELECT` de ids por arquivo |
| 6 | T6 | `PersistSymbols` passa a usar o lote |
| 7 | T7 | `code_edges` em lote: 1 `INSERT` multi-linha por arquivo |
| 8 | T8 | `FlushEdges`/`resolveAndInsertEdges` passam a usar o lote |
| 9 | T9 | Testes do pipeline: equivalência de linhas/ids, mapeamento de ids, arestas |
| 10 | T10 | Mutante (prova vermelha), gate completo e `validation.md` |
| 11 | T11 | Complemento de 29/10: `code_edges_uncertain` em lote (decisão A11 refutada pela medição) |
| 12 | T12 | `FlushEdges` grava os DOIS buckets em 1 statement por arquivo |
| 13 | T13 | Testes do bucket incerto: equivalência (SQLite), statements por arquivo (Postgres real), mutante |
| 14 | T14 | Gate final do complemento e evidência no `validation.md` |

## Task Breakdown

### Phase 1: Costura de escrita da ponte

### T1: Costura de escrita e construtores puros do statement da ponte
**What**: Em `internal/db/code_bridge.go`, introduzir a interface estreita `bridgeWriter` (um método `ExecContext`), o núcleo `buildCodeBridgeWithWriter` (o `*sql.DB` satisfaz a interface) e os construtores puros `buildNodeInsert`/`buildEdgeInsert` (query + args), com os mesmos textos de SQL dos helpers por símbolo (`INSERT OR IGNORE` no SQLite; `ON CONFLICT (repository, id) DO NOTHING` e `ON CONFLICT (repository, source_id, target_id, relation)` no Postgres). `upsertBridgeNode`/`upsertBridgeEdge` passam a ser implementados sobre os mesmos construtores (uso unitário); o casamento não é tocado
**Where**: internal/db/code_bridge.go
**Depends on**: none
**Requirement**: BR-04
**Tests**: internal/db/code_bridge_batch_test.go
**Gate**: go build ./...
**Done when**:
- [x] A interface `bridgeWriter` existe e `*sql.DB` a satisfaz
- [x] `buildNodeInsert`/`buildEdgeInsert` devolvem query e args, sem tocar o banco
- [x] O texto de SQL de cada engine é **idêntico** ao do helper por símbolo
- [x] Nenhuma assinatura pública mudou

### Phase 2: Lote da ponte

### T2: A ponte grava em lote (1 nós + 1 arestas por documento)
**What**: O laço de escrita de `buildCodeBridge` deixa de chamar `upsertBridgeNode`/`upsertBridgeEdge` por símbolo: monta as linhas casadas e emite **1** `INSERT` multi-linha em `graph_nodes` **e depois** **1** `INSERT` multi-linha em `graph_edges` por documento (partindo em pedaços de 2000 linhas como teto de segurança de parâmetros). As contagens de `CodeBridgeResult` (Nodes/Edges/Matched) continuam iguais
**Where**: internal/db/code_bridge.go
**Depends on**: T1
**Requirement**: BR-01, BR-02, BR-03, BR-05
**Tests**: internal/db/code_bridge_batch_test.go
**Gate**: go test -count=1 -run PonteLote ./internal/db/...
**Done when**:
- [x] O caminho de produção usa o lote (os helpers por símbolo ficam para uso unitário)
- [x] Os nós são gravados antes das arestas
- [x] As contagens de `CodeBridgeResult` não mudaram
- [x] O cálculo do casamento (literais por qualified name, por path, nome fraco) está intocado

### Phase 3: Testes da ponte (SQLite)

### T3: Teste da ponte em lote — contagem de statements e equivalência de estado
**What**: Escrever `internal/db/code_bridge_batch_test.go` (package `db`) com: (a) um writer que conta os statements; (b) um teste que semeia N símbolos, indexa um documento que os menciona e exige **exatamente 2** statements (1 nós + 1 arestas), com o statement dos nós antes do das arestas; (c) um teste que mostra o número de statements **constante** entre N=1 e N=25; (d) um teste que compara o estado final do grafo com o do caminho por símbolo; (e) testes dos construtores puros (N tuplas, N×params, `Bind` → `$1..$N` sem tocar literal de string)
**Where**: internal/db/code_bridge_batch_test.go
**Depends on**: T2
**Requirement**: BR-01, BR-02, BR-03, BR-04, BR-05, PRV-01
**Tests**: internal/db/code_bridge_batch_test.go
**Gate**: go test -count=1 -run PonteLote ./internal/db/...
**Done when**:
- [x] O teste conta statements **de verdade** (writer que intercepta o `ExecContext` da ponte)
- [x] O número de statements é 2 para N=1 e para N=25 (constante, não O(N))
- [x] O grafo final é o mesmo do caminho por símbolo
- [x] O `Bind` numera `$1..$N` e nenhum `?` de literal é convertido

### Phase 4: Postgres real

### T4: Ponte, símbolos e arestas em lote no Postgres real
**What**: Escrever `internal/db/code_batch_postgres_test.go` (package `db_test`) contra o banco descartável do `internal/pgtest`, com o `pgtest.Recorder` contando o SQL que chega ao servidor: (a) a ponte emite 2 statements por documento e o `INSERT` multi-linha com `ON CONFLICT` é aceito; (b) o `code_symbols` em lote grava as mesmas linhas e resolve os mesmos ids; (c) o `code_edges` em lote é aceito e é idempotente
**Where**: internal/db/code_batch_postgres_test.go
**Depends on**: T3
**Requirement**: PRV-02, SYM-01, SYM-03, SYM-04, EDG-01, EDG-02
**Tests**: internal/db/code_batch_postgres_test.go
**Gate**: MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgres ./internal/db/...
**Done when**:
- [x] O teste roda contra o banco descartável (nunca o do dono)
- [x] O `pgtest.Recorder` mostra o número de statements por documento/arquivo
- [x] O `INSERT` multi-linha com `ON CONFLICT` é aceito pelo servidor real

### Phase 5: Símbolos em lote

### T5: `code_symbols` em lote — 1 INSERT + 1 SELECT de ids por arquivo
**What**: Em `internal/db/code_migration.go`, adicionar `CodeSymbolRow` e `UpsertCodeSymbols(ctx, db, fileID, rows)`: um `INSERT` multi-linha com as **mesmas 11 colunas** e o **mesmo** `ON CONFLICT(file_id, kind, qualified_name, start_line)` de `UpsertCodeSymbol`, com as chaves duplicadas do próprio lote removidas antes (o payload do último vence) e, depois, **um** `SELECT id, kind, COALESCE(qualified_name,''), start_line FROM code_symbols WHERE file_id = ?` que resolve os ids por chave (id maior = linha gravada). Devolve os ids na ordem da entrada. `UpsertCodeSymbol` continua existindo com a mesma assinatura
**Where**: internal/db/code_migration.go
**Depends on**: T4
**Requirement**: SYM-01, SYM-03, SYM-04, SYM-05
**Tests**: internal/codeast/batch_test.go
**Gate**: go test -count=1 -run LoteSimbolos ./internal/codeast/...
**Done when**:
- [x] As 11 colunas e o alvo de `ON CONFLICT` são os de `UpsertCodeSymbol`
- [x] A chave duplicada no lote é removida antes do statement (o Postgres rejeita o conflito dentro do statement)
- [x] O `SELECT` de resolução devolve o id da linha gravada, na ordem da entrada
- [x] `UpsertCodeSymbol` segue disponível e sem mudança de assinatura

### Phase 6: `PersistSymbols` em lote

### T6: `PersistSymbols` passa a usar o lote
**What**: Em `internal/codeast/cache.go`, `PersistSymbols` converte os `[]Symbol` em `[]db.CodeSymbolRow` e chama `db.UpsertCodeSymbols` uma vez por arquivo; a assinatura pública (`([]int64, error)`, ids na ordem dos símbolos) e a mensagem de erro por símbolo ficam as mesmas
**Where**: internal/codeast/cache.go
**Depends on**: T5
**Requirement**: SYM-01, SYM-02
**Tests**: internal/codeast/batch_test.go
**Gate**: go test -count=1 -run LoteSimbolos ./internal/codeast/...
**Done when**:
- [x] `PersistSymbols` emite 1 statement de escrita por arquivo
- [x] Os ids saem na ordem dos símbolos (o mapeamento das arestas continua válido)
- [x] A assinatura pública não mudou

### Phase 7: Arestas em lote

### T7: `code_edges` em lote — 1 INSERT multi-linha
**What**: Em `internal/db/code_migration.go`, adicionar `CodeEdgeRow` e `UpsertCodeEdges(ctx, db, rows)`: um `INSERT` multi-linha com as **mesmas 9 colunas** e o **mesmo** `ON CONFLICT(src_symbol_id, dst_symbol_id, kind)` de `UpsertCodeEdgeWithStatus` (inclusive o `CASE WHEN` que mantém a menor `start_line`/`end_line`), com as chaves duplicadas do lote fundidas (menor `start_line`/`end_line`, payload do último) e a recusa de `confidence < 0.5` preservada
**Where**: internal/db/code_migration.go
**Depends on**: T6
**Requirement**: EDG-01, EDG-02, EDG-03
**Tests**: internal/codeast/batch_test.go
**Gate**: go test -count=1 -run LoteArestas ./internal/codeast/...
**Done when**:
- [x] As 9 colunas e o alvo de `ON CONFLICT` são os de `UpsertCodeEdgeWithStatus`
- [x] A duplicata no lote vira uma linha só, com a menor posição
- [x] `confidence < 0.5` continua recusado com o mesmo erro

### Phase 8: Pipeline em lote

### T8: `FlushEdges` e `resolveAndInsertEdges` gravam em lote
**What**: Em `internal/codeast/pipeline.go`, separar a **resolução** da aresta da **escrita**: `resolveEdge` passa a devolver a aresta resolvida (sem gravar) e quem chama grava — `FlushEdges` agrupa as arestas resolvidas por arquivo e grava 1 statement por arquivo; `resolveAndInsertEdges` (caminho legado) também coleta e grava em lote. O caminho não-batched (`collectEdges` com `batched == false`) continua gravando aresta por aresta, como hoje
**Where**: internal/codeast/pipeline.go
**Depends on**: T7
**Requirement**: EDG-01
**Tests**: internal/codeast/batch_test.go
**Gate**: go test -count=1 ./internal/codeast/...
**Done when**:
- [x] O `FlushEdges` grava 1 statement por arquivo
- [x] O caminho legado também grava em lote
- [x] A resolução (níveis 1/2/3, arestas incertas) não mudou

### Phase 9: Testes do pipeline

### T9: Testes do pipeline — equivalência de linhas/ids, mapeamento de ids e arestas
**What**: Escrever `internal/codeast/batch_test.go` com: (a) equivalência — o mesmo arquivo com N símbolos gravado pelo caminho por símbolo e pelo caminho em lote produz as mesmas linhas e os mesmos ids; (b) mapeamento — o id devolvido na posição `i` é o id do símbolo `i` lido do banco; (c) idempotência — reindexar o arquivo não duplica linhas nem muda os ids; (d) arestas — o estado final de `code_edges` é o mesmo com e sem lote
**Where**: internal/codeast/batch_test.go
**Depends on**: T8
**Requirement**: SYM-02, SYM-03, SYM-05, EDG-02, EDG-03, PRV-01
**Tests**: internal/codeast/batch_test.go
**Gate**: go test -count=1 -run Lote ./internal/codeast/...
**Done when**:
- [x] O teste de equivalência compara banco a banco (linhas e ids)
- [x] O teste de mapeamento liga o id devolvido ao símbolo correspondente
- [x] O teste de idempotência prova que rodar 2× não duplica

### Phase 10: Gate final e evidências

### T10: Mutante, gate completo e `validation.md`
**What**: Registrar a prova de que os testes novos **falham** contra o comportamento antigo (voltar a emitir 1 statement por linha e rodar os testes), rodar o gate completo (`go test -count=1 ./...` com `MY_MEMORY_PG_TEST=1` e `MY_MEMORY_PG_URL` preservada, `go build ./...`, `go vet ./...`, `gofmt -l` na cópia LF dos arquivos tocados) e escrever `validation.md` com o bloco canônico e evidência `arquivo:linha` por requisito
**Where**: .specs/features/batch-code-bridge-writes/validation.md
**Depends on**: T9
**Requirement**: PRV-01, PRV-02, PRV-03
**Tests**: .specs/features/batch-code-bridge-writes/validation.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_state.py batch-code-bridge-writes --root D:\repository\my-memory
**Done when**:
- [x] A rodada vermelha (mutante) está registrada com o sintoma exato
- [x] `go build ./...`, `go vet ./...` e `go test -count=1 ./...` com exit 0
- [x] `gofmt -l` limpo nos arquivos .go tocados (medido em cópia com LF)
- [x] `validation.md` com `## Validation: PASS` + `**Result**: PASS` e evidência `arquivo:linha` por requisito

### Phase 11: Complemento — o bucket de arestas incertas em lote

### T11: `code_edges_uncertain` em lote no `internal/db`
**What**: Em `internal/db/code_migration.go`, adicionar `CodeEdgeUncertainRow` e `InsertCodeEdgeUncertains`: um `INSERT` multi-linha com as **mesmas 8 colunas** e o **mesmo** `ON CONFLICT(src_symbol_id, dst_candidate_name, kind)` de `InsertCodeEdgeUncertain` (a chave existe nos DOIS engines — ver A12), com as chaves duplicadas do lote fundidas (o payload do **último** vence, porque o `ON CONFLICT` grava `excluded` em todas as colunas). `InsertCodeEdgeUncertain` continua existindo, com a mesma assinatura, implementado sobre o mesmo construtor
**Where**: internal/db/code_migration.go
**Depends on**: T10
**Requirement**: UNC-01, UNC-03
**Tests**: internal/codeast/batch_test.go
**Gate**: go test -count=1 -run LoteIncertas ./internal/codeast/...
**Done when**:
- [x] As 8 colunas e o alvo de `ON CONFLICT` são os de `InsertCodeEdgeUncertain`
- [x] A chave duplicada no lote vira uma linha só, com o payload do último
- [x] `InsertCodeEdgeUncertain` segue disponível e sem mudança de assinatura
- [x] Nenhum índice novo, nenhuma coluna nova, nenhum schema alterado

### Phase 12: O `FlushEdges` grava os dois buckets

### T12: `FlushEdges` grava arestas resolvidas E incertas em 1 statement por arquivo
**What**: Em `internal/codeast/pipeline.go`, a resolução passa a devolver a linha incerta (sem gravar): `resolveEdgeRow` devolve um `resolvedEdge` (ou `code_edges`, ou `code_edges_uncertain`) e `uncertainRow` monta a linha do bucket. `FlushEdges` agrupa os DOIS buckets por arquivo e grava 1 statement por arquivo em cada um; o caminho por aresta (`resolveEdge`, usado quando `batched == false`) e o caminho legado (`resolveAndInsertEdges`) também gravam o bucket incerto em lote
**Where**: internal/codeast/pipeline.go
**Depends on**: T11
**Requirement**: UNC-02, EDG-04
**Tests**: internal/codeast/batch_postgres_test.go
**Gate**: MY_MEMORY_PG_TEST=1 go test -count=1 -run LotePostgresFlushEdges ./internal/codeast/...
**Done when**:
- [x] O `FlushEdges` grava 1 statement por arquivo em `code_edges` e 1 em `code_edges_uncertain`
- [x] A aresta incerta não é mais gravada na hora da resolução
- [x] A resolução (níveis 1/2/3 e o texto do `reason`) não mudou

### Phase 13: Testes do bucket incerto

### T13: Testes do bucket incerto — equivalência, statements por arquivo e mutante
**What**: Escrever os testes do bucket incerto: (a) equivalência — o mesmo conjunto de arestas incertas gravado pelo caminho por aresta e pelo caminho em lote deixa `code_edges_uncertain` igual; (b) Postgres real — 1 `INSERT` por arquivo no bucket, a chave duplicada fundida e a idempotência; (c) o `FlushEdges` no Postgres real com 2 arquivos → 2 + 2 statements (um por bucket por arquivo); (d) o mutante que volta a 1 statement por linha, com a rodada vermelha registrada
**Where**: internal/codeast/batch_postgres_test.go
**Depends on**: T12
**Requirement**: UNC-01, UNC-04, EDG-04, PRV-01
**Tests**: internal/codeast/batch_postgres_test.go
**Gate**: MY_MEMORY_PG_TEST=1 go test -count=1 -run 'LotePostgresIncertas|LotePostgresFlushEdges' ./internal/...
**Done when**:
- [x] O `pgtest.Recorder` conta 1 statement por arquivo em cada bucket
- [x] O estado final do bucket é o mesmo nos dois caminhos
- [x] O mutante morre com o sintoma exato (rodada vermelha registrada na `validation.md`)

### Phase 14: Gate final do complemento

### T14: Gate completo e evidência do complemento no `validation.md`
**What**: Rodar o gate completo (`go test -count=1 ./...` com `MY_MEMORY_PG_TEST=1` e `MY_MEMORY_PG_URL` preservada, `go build ./...`, `go vet ./...`, `gofmt -l` na cópia LF dos arquivos tocados) e registrar no `validation.md` a evidência do complemento: os números antes/depois do bucket incerto, o custo real do `UpsertCodeFile` e a confirmação do `FlushEdges` por arquivo
**Where**: .specs/features/batch-code-bridge-writes/validation.md
**Depends on**: T13
**Requirement**: PRV-03
**Tests**: .specs/features/batch-code-bridge-writes/validation.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_state.py batch-code-bridge-writes --root D:\repository\my-memory
**Done when**:
- [x] `go build ./...`, `go vet ./...` e `go test -count=1 ./...` com exit 0
- [x] `gofmt -l` limpo nos arquivos .go tocados (medido em cópia com LF)
- [x] `validation.md` com o bloco canônico e a evidência `arquivo:linha` do complemento
