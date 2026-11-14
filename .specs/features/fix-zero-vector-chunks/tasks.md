# Tasks: fix-zero-vector-chunks

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-13
**Issue**: ISSUE-040 (`.specs/ISSUES.md`)

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Design**: inline nesta spec (uma mudança de escrita, uma de leitura e uma de detecção; sem mudança de schema — a coluna `embedding` já é nullable)
**Status**: In Progress

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec. Guidelines found: `AGENTS.md` (regra 4 — testes obrigatórios; regra 8 — quality gate), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`, `.github/workflows/ci.yml` (comandos de teste reais do projeto).

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Gravação do chunk sem embedding (Postgres) | unit | AC1 — `embedding` NULL quando não há vetor; vetor real quando há | `internal/store/chunk_embedding_test.go` | `go test -count=1 -run TestChunkEmbedding ./internal/store/...` |
| Gravação do chunk sem embedding (SQLite) | unit | AC1 — nenhuma linha em `chunks_vec` quando não há vetor | `internal/db/chunk_embedding_test.go` | `go test -count=1 -run TestChunkEmbedding ./internal/db/...` |
| Indexador sem provider | integration | AC1 — o indexador não substitui o vetor zero; o run reporta o documento sem vetor | `cmd/mem/index_no_vector_test.go` | `go test -count=1 -run TestIndexNoVector ./cmd/mem/...` |
| Detecção no doctor | unit | AC2 e AC3 — vetor NULL e vetor zero são dessincronizados; vetor real não é; o score penaliza | `internal/store/doctor_zero_vector_test.go` · `internal/db/doctor_zero_vector_test.go` | `go test -count=1 -run TestZeroVector ./internal/store/... ./internal/db/...` |
| Busca vetorial | unit (SQLite) + integration (Postgres) | AC4 e AC5 — o chunk sem vetor real não sai da fonte vetorial | `internal/db/knn_zero_vector_test.go` · `internal/store/knn_zero_vector_test.go` | `go test -count=1 -run TestKNNZeroVector ./internal/db/... ./internal/store/...` |
| Prova E2E do cofre | integration (ONNX real) | AC6 e AC7 — a nota volta por sentido; reindex do zero deixa 0 chunks sem vetor | `cmd/mem/vault_retrieval_test.go` | `CGO_ENABLED=1 go test -tags "sqlite_fts5 onnx" -count=1 -run TestVaultRetrieval ./cmd/mem/...` |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após cada tarefa de código | `go build ./...` e `go vet ./...` |
| Quick | Após as tarefas de teste | `go test -count=1 -run ZeroVector ./internal/db/... ./internal/store/... ./cmd/mem/...` |
| Full | Gate final (T8) | `go test -count=1 ./...` (com `MY_MEMORY_PG_TEST=1` e `MY_MEMORY_PG_URL` preservada) |
| Tagged | T8 | `$env:PATH="C:\msys64\ucrt64\bin;$env:PATH"; $env:CGO_ENABLED='1'; go test -tags "sqlite_fts5 onnx" -count=1 -run TestVaultRetrieval ./cmd/mem/...` |
| Format | Antes de declarar pronto | `gofmt -l` na cópia LF dos arquivos `.go` tocados (o working tree local mistura CRLF) |
| Spec | Antes de entregar | `python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/fix-zero-vector-chunks/spec.md` e `validate_tasks.py .specs/features/fix-zero-vector-chunks/tasks.md` |
| State | Antes de entregar | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-zero-vector-chunks --root D:\repository\my-memory` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: Não gravar vetor zero

```
T1 -> T2 -> T3
```

### Phase 2: Detectar e excluir

```
T4 -> T5 -> T6
```

### Phase 3: A prova do cofre

```
T6 -> T7 -> T8
```

---

## Task Breakdown

### T1: O insert do Postgres aceita chunk sem embedding

**What**: `InsertChunk` do Postgres grava `embedding` NULL quando o vetor chega
vazio, em vez de gravar um vetor zero.
**Where**: `internal/store/postgres.go`
**Depends on**: None
**Reuses**: `FormatVector` e a query de insert existentes (`:315-323`)
**Requirement**: ZVC-01

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] Vetor vazio → `embedding` NULL (e não `''::vector`, que o cast rejeita)
- [x] Vetor real → gravado como antes (384 dimensões)
- [x] O teste confere o valor gravado no banco, não a chamada

**Status**: Verified (2026-11-14)

**Tests**: unit
**Gate**: quick

**Commit**: `fix(store): chunk sem embedding e gravado com embedding NULL`

---

### T2: O indexador Postgres deixa de substituir o vetor zero

**What**: o caminho de indexação do Postgres para de trocar o embedding ausente
pelo vetor zero e passa a passar o vetor ausente ao insert.
**Where**: `cmd/mem/main.go`
**Depends on**: T1
**Reuses**: o bloco de chunk do `runIndexPostgres` (`:1888-1922`)
**Requirement**: ZVC-01

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] O `make([]float32, embedder.EmbeddingDim)` de substituto **sai** do caminho
      Postgres
- [x] O aviso por documento continua (a degradação é visível)
- [x] O documento indexado sai com os chunks sem embedding

**Status**: Verified (2026-11-14)

**Tests**: integration
**Gate**: quick

**Commit**: `fix(index): o indexador Postgres nao grava mais vetor zero`

---

### T3: O indexador SQLite deixa de substituir o vetor zero

**What**: o mesmo no caminho de indexação do SQLite — onde o `InsertChunk` já
sabe gravar chunk sem linha em `chunks_vec`.
**Where**: `cmd/mem/main.go`
**Depends on**: T2
**Reuses**: o bloco de chunk do `runIndexSQLite` (`:2129-2153`)
**Requirement**: ZVC-01

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] O substituto zero **sai** do caminho SQLite
- [x] O chunk fica sem linha em `chunks_vec` **e** sem linha em
      `chunks_turboquant` (verificado por query)
- [x] O aviso por documento continua

**Tests**: integration
**Gate**: quick

**Status**: Verified (2026-11-14)

**Sensor** (3 mutações, `D:\.minimax\scratch\sensor_t3.py`):

| Mutação | Efeito | Veredito |
| :--- | :--- | :--- |
| M-T3a — o substituto zero volta ao caminho SQLite | o teste acusa `chunks_vec` **e** `chunks_turboquant` com linha | MORTO |
| M-T3b — só o caminho comprimido volta a gravar o zero | o teste acusa `chunks_turboquant` com linha | MORTO |
| M-T3c — a guarda `if len(vec) > 0` é removida | o próprio `Quantize` rejeita dimensão inválida (`esperado 384, recebido 0`), então nada é gravado | SOBREVIVEU — **mutante equivalente** (subsumido pelo check de dimensão do `Quantize`); a guarda fica por explicitar o requisito da AC1 |

**Commit**: `fix(index): o indexador SQLite nao grava mais vetor zero`

---

### T4: O doctor Postgres detecta o chunk sem vetor real

**What**: o check de chunks dessincronizados do Postgres passa a considerar
desincronizado tanto o `embedding IS NULL` quanto o **vetor zero** (igual ao
vetor zero do schema).
**Where**: `internal/store/doctor.go`
**Depends on**: None
**Reuses**: a query de desincronizados existente (`:173-188`)
**Requirement**: ZVC-02, ZVC-03

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] Vetor NULL → dessincronizado
- [x] Vetor zero → dessincronizado
- [x] Vetor real → **não** dessincronizado
- [x] O `Issue` reportado diz qual dos dois casos é

**Tests**: unit
**Gate**: quick

**Status**: Verified (2026-11-14)

**Sensor** (3 mutações, `D:\.minimax\scratch\sensor_t4.py`):

| Mutação | Efeito | Veredito |
| :--- | :--- | :--- |
| M-T4a — a condição volta a `c.embedding IS NULL` (o filtro antigo) | o chunk de vetor zero deixa de ser reportado (só o ausente entra) | MORTO |
| M-T4b — o `CASE` some: todo dessincronizado vira "Missing vector embedding" | o `Issue` deixa de distinguir ausente de zero | MORTO |
| M-T4c — a query volta a dois campos enquanto o `Scan` lê três | o relatório fica vazio sem erro visível (o `if err == nil` engole o erro do `Scan`) | MORTO |

**Nota de implementação**: a comparação do vetor zero no Postgres **não** usa
literal de 384 dimensões — o pgvector dá o produto interno negativo em `<#>`,
e `v <#> v = -||v||²`, que vale zero só quando o vetor É o vetor zero.

**Commit**: `fix(doctor): o Postgres detecta chunk com vetor zero`

---

### T5: O doctor SQLite detecta o chunk sem vetor real

**What**: o mesmo contrato no doctor do SQLite — chunk sem linha em
`chunks_vec` **ou** com vetor zero é dessincronizado.
**Where**: `internal/db/graph.go`
**Depends on**: T4
**Reuses**: o check de dessincronizados do SQLite (na mesma família de queries)
**Requirement**: ZVC-02, ZVC-03

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] Chunk sem linha em `chunks_vec` **e** sem linha em `chunks_turboquant`
      → dessincronizado
- [x] Chunk com vetor zero → dessincronizado
- [x] Chunk com vetor real → **não** dessincronizado

**Tests**: unit
**Gate**: quick

**Status**: Verified (2026-11-14)

**Sensor** (4 mutações, `D:\.minimax\scratch\sensor_t5.py`):

| Mutação | Efeito | Veredito |
| :--- | :--- | :--- |
| M-T5a — `vetorCruzoEhZero` deixa de ver o zero | o chunk com vetor **cru** zero (e só cru) passa a ser reportado como "Missing TurboQuant" | MORTO |
| M-T5b — `turboQuantEhZero` deixa de ver o zero | o chunk com vetor **comprimido** zero (e só comprimido) não é reportado | MORTO |
| M-T5c — o caso da ausência some | o chunk sem vetor nenhum não é reportado | MORTO |
| M-T5d — a regra "vetor cru sem o comprimido" some | o chunk que a busca comprimida não encontraria não é reportado | MORTO |

**Notas de implementação**:

- A comparação do zero ficou em Go (`internal/db/vector_zero.go`) porque o
  `chunks_vec` muda de forma conforme a plataforma: tabela virtual `vec0`
  na Unix com cgo, tabela BLOB no resto. A assinatura do vetor zero no
  caminho comprimido é exata: o `Quantize` grava o valor quantizado como
  nibble deslocado em 7, então o vetor zero é todo byte `0x77` — e um vetor
  real sempre tem um componente que normaliza para ±7 (nibble 0 ou 14).
- A regra anterior ("vetor cru sem o comprimido" → dessincronizado) foi
  **mantida**: ela cobre o caso em que a busca no caminho comprimido — o do
  Windows — não encontraria o chunk.

**Commit**: `fix(doctor): o SQLite detecta chunk com vetor zero`

---

### T6: A busca vetorial exclui o chunk sem vetor real

**What**: a query de KNN do Postgres e a do SQLite deixam de devolver chunk sem
embedding real (ausente ou zero).
**Where**: `internal/store/postgres.go` · `internal/db/graph.go`
**Depends on**: T5
**Reuses**: `SearchKNNWithOptions` dos dois engines e a query de busca híbrida
**Requirement**: ZVC-04, ZVC-05

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] O Postgres não devolve chunk com `embedding` NULL nem com vetor zero
- [x] O SQLite não devolve chunk sem linha em `chunks_vec` nem com vetor zero
- [x] O SQLite não devolve chunk cuja linha TurboQuant é a de um vetor zero
- [x] O híbrido alcança o chunk sem vetor **apenas** pela fonte léxica
- [x] A busca com todos os chunks sem vetor devolve lista vazia, e não erro

**Tests**: unit + integration
**Gate**: quick

**Status**: Verified (2026-11-14)

**Sensor** (4 mutações, `D:\.minimax\scratch\sensor_t6.py` — cada mutação é
rodada nos **três** caminhos: SQLite comprimido, SQLite cru (vec0) e Postgres):

| Mutação | Efeito | Veredito |
| :--- | :--- | :--- |
| M-T6a — os dois filtros do Postgres saem da query do KNN | a busca **erra** (`Scan error ... converting NULL to float64`): o chunk sem embedding entra com distância nula | MORTO |
| M-T6b — o filtro do vetor zero sai do caminho comprimido | o chunk com vetor zero volta pelo TurboQuant; a busca "todos sem vetor" devolve 1 resultado; no híbrido ele aparece na fonte `vector:1` | MORTO |
| M-T6c — o filtro do vetor zero sai do caminho cru (vec0) | o chunk com vetor zero volta pelo KNN do sqlite-vec | MORTO |
| M-T6d — só o filtro `IS NOT NULL` sai do Postgres | o `NULL` continua de fora: `(embedding <#> embedding) <> 0` já o exclui (`NULL <> 0` não é verdadeiro) | SOBREVIVEU — **mutante equivalente**; a cláusula `IS NOT NULL` fica por explicitar o requisito |

**Notas de implementação**:

- O caminho cru (sqlite-vec) precisou de uma decisão: a coluna do vetor
  **não** pode ser filtrada dentro da query do KNN. O `v.embedding` foi
  acrescentado ao `SELECT` e o filtro roda em Go. Isso é seguro porque o
  sqlite-vec devolve o vetor **gravado** fora do `MATCH` (e o `test.sql` do
  próprio upstream seleciona a coluna do vetor em query KNN).
- O caminho vec0 **não existe no Windows** (`HasSqliteVec` é false lá). Para
  verificar localmente, o build tag foi relaxado **temporariamente** (com o
  shim `sqlite3.h` apontando para o header do `mattn/go-sqlite3`) e revertido
  depois: ele não pode ir para o commit, porque quebraria o build do Windows
  sem o shim. A suíte `internal/db` inteira passou com a tag `sqlite_fts5`
  (o mesmo que a CI roda no Ubuntu).

**Commit**: `fix(search): a busca vetorial exclui chunk sem vetor real`

---

### T7: A prova E2E da recuperação por sentido

**What**: teste de integração que escreve uma nota pelo caminho da lição,
indexa com provider e a recupera por uma consulta **sem palavra em comum**,
através da fonte vetorial.
**Where**: `cmd/mem/vault_retrieval_test.go`
**Depends on**: T6
**Reuses**: o caminho de escrita de `mem lesson` (`cmd/mem/lesson.go`) e a busca
híbrida
**Requirement**: ZVC-06

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] Sem a tag `onnx`, o teste **pula** com mensagem explícita (nunca verde
      mentiroso)
- [x] Com a tag, a nota escrita volta na busca por paráfrase
- [x] A origem do hit é a fonte `vector`, não a léxica

**Tests**: integration
**Gate**: tagged

**Status**: Verified (2026-11-14)

**Sensor** (2 mutações, `D:\.minimax\scratch\sensor_t7.py`):

| Mutação | Efeito | Veredito |
| :--- | :--- | :--- |
| M-T7a — `newBuiltinEmbedder` devolve um provider nulo | a nota entra sem vetor e a busca por paráfrase **não** traz a nota | MORTO |
| M-T8a — o vetor real do provider é substituído pelo vetor zero | depois do reindex do zero, 1 chunk continua sem vetor real | MORTO |

**Notas de implementação**:

- O arquivo **não** tem `//go:build onnx`: ele compila nas duas builds e
  **pula com mensagem explícita** quando o provider embutido não existe
  (`newBuiltinEmbedder` devolve erro no binário sem a tag). Sem a tag, o
  resultado observado é `--- SKIP` com o motivo — nunca verde-mentira.
- A nota entra pelo caminho de verdade (`mem lesson`) num cofre
  descartável: a configuração global é redirecionada por
  `MY_MEMORY_GLOBAL_CONFIG_DIR`, então o teste nunca toca no cofre real.
- A consulta é uma paráfrase **sem nenhuma palavra em comum** com a nota
  (nem no título, nem no corpo, nem no frontmatter) — e o FTS5 do SQLite
  casa os termos com AND, então a fonte léxica não traz a nota.

**Commit**: `test(vault): a nota do cofre volta por sentido`

---

### T8: A prova de que o reindex do zero não perde vetor

**What**: o mesmo teste força a reindexação do zero com provider presente e
confirma que **nenhum** chunk fica sem vetor real.
**Where**: `cmd/mem/vault_retrieval_test.go`
**Depends on**: T7
**Reuses**: o `mem index --force` e o doctor
**Requirement**: ZVC-07

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] Depois do `--force`, a contagem de chunks sem vetor real é **zero**
- [x] A contagem de documentos e de chunks é a mesma de antes (o reindex não
      perde conteúdo)
- [x] O doctor reporta zero chunks dessincronizados

**Tests**: integration
**Gate**: full

**Status**: Verified (2026-11-14)

**Nota de implementação**: o teste faz o caminho **do reparo inteiro** —
primeiro indexa o vault **sem** provider (que é o que deixa os chunks sem
vetor, e o doctor reporta), depois reindexa do zero **com** o provider real
(`mem index --force`). A contagem de chunks sem vetor real vem do próprio
doctor (`db.DiagnoseHealth`), que é a mesma definição usada nas T4 e T5.

**Commit**: `test(vault): o reindex do zero preserva os vetores`

---

## Phase Execution Map

```
Phase 1 → Phase 2 → Phase 3

Phase 1:  T1 -> T2 -> T3
Phase 2:  T4 -> T5 -> T6
Phase 3:  T6 -> T7 -> T8
```

Execution is strictly sequential - there is no intra-phase parallelism. A single
agent (or batch worker) works one task at a time, in order.

**Contagem de tarefas**: 8 — cabem em um único lote (≤ ~8 por worker), então a
execução é **inline**, sem sub-agente.

---

## Diagram-Definition Cross-Check

| Task | Depends On (task body) | Diagram Shows | Status |
| ---- | ---------------------- | ------------- | ------ |
| T1 | None | (início da fase 1) | ✅ Match |
| T2 | T1 | T1 -> T2 | ✅ Match |
| T3 | T2 | T2 -> T3 | ✅ Match |
| T4 | None | (início da fase 2) | ✅ Match |
| T5 | T4 | T4 -> T5 | ✅ Match |
| T6 | T5 | T5 -> T6 | ✅ Match |
| T7 | T6 | T6 -> T7 | ✅ Match |
| T8 | T7 | T7 -> T8 | ✅ Match |

Nenhuma dependência aponta para uma fase posterior.

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | Gravação do chunk (Postgres) | unit | unit | ✅ OK |
| T2 | Indexador | integration | integration | ✅ OK |
| T3 | Indexador | integration | integration | ✅ OK |
| T4 | Detecção no doctor | unit | unit | ✅ OK |
| T5 | Detecção no doctor | unit | unit | ✅ OK |
| T6 | Busca vetorial | unit + integration | unit + integration | ✅ OK |
| T7 | Prova E2E | integration | integration | ✅ OK |
| T8 | Prova E2E | integration | integration | ✅ OK |

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: insert Postgres | 1 função | ✅ Granular |
| T2: indexador Postgres | 1 bloco de chunk | ✅ Granular |
| T3: indexador SQLite | 1 bloco de chunk | ✅ Granular |
| T4: doctor Postgres | 1 query | ✅ Granular |
| T5: doctor SQLite | 1 query | ✅ Granular |
| T6: busca vetorial | 1 query por engine | ✅ Granular |
| T7: prova por sentido | 1 teste | ✅ Granular |
| T8: prova do reindex | 1 teste | ✅ Granular |
