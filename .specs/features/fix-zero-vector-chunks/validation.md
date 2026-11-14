# fix-zero-vector-chunks Validation

**Date**: 2026-11-14
**Spec**: `.specs/features/fix-zero-vector-chunks/spec.md`
**Diff range**: `78dda3f..d9c5ebb` (branch `develop`, 9 commits)
**Round**: 1
**Verdict**: PASS (7/7 ACs com evidência; 14 mutações mortas + 2 mutantes equivalentes registrados; gate full e gate taggeado verdes)

> [!IMPORTANT]
> **A verificação desta rodada foi feita pelo autor do código, e isso está declarado aqui de propósito.**
> Um sub-agente `verifier` independente **não** foi despachado para esta spec: na spec
> `fix-release-onnx-provider` (a anterior, mesma rodada) o verificador **morreu duas vezes
> no meio da execução** (turno encerrado com `finish_reason: stop` e saída vazia — assinatura
> de limite de uso do provider). Consequência honesta: a separação autor ≠ verificador
> **não vale** para o que está neste relatório. O que **vale** é que **toda** AC tem uma
> asserção automatizada que **morre** quando o comportamento é quebrado (5 sensores, 14
> mutações mortas), e que as duas provas E2E rodaram com o **provider ONNX real** — não com
> dublê.
> **Recomendação: reexecutar um Verifier independente antes de publicar o release.**

---

## O que a spec cobre (e por que a emenda foi feita antes de executar)

A spec foi **emendada antes** da execução (commit `d523d91`), porque o sensor da spec
anterior tinha medido uma dimensão que a spec original não cobria:

| Engine | Onde o vetor é gravado | Onde a busca vetorial lê |
| :--- | :--- | :--- |
| PostgreSQL | `chunks.embedding` (`vector(384)`, nullable) | `chunks.embedding` (pgvector) |
| SQLite | `chunks_vec` (float32) **e** `chunks_turboquant` (4-bit comprimido) | `chunks_vec` quando há `sqlite-vec`; **`chunks_turboquant`** no Windows, onde `db.HasSqliteVec` é `false` |

Ou seja: **zerar a coluna `chunks.vec` não matava a busca vetorial no Windows** — a busca
lê as linhas TurboQuant. E o KNN do Postgres **não filtrava `embedding IS NULL`**. A
emenda cobriu os dois caminhos de armazenamento do SQLite e o `embedding` NULL do
Postgres, sem mudar o escopo do defeito.

---

## Spec-Anchored Acceptance Criteria

| AC | Resultado definido na spec | Evidência (`file:line` + asserção) | Resultado |
| --- | --- | --- | --- |
| **AC1** — o chunk indexado sem provider é gravado **sem** embedding (NULL no Postgres; **nenhuma** linha em `chunks_vec` **e** `chunks_turboquant` no SQLite) | sem vetor zero | **Postgres**: `internal/store/postgres.go:329-332` — `var vecValue any` + `if len(vec) > 0` (o `$6::vector` rejeita `'[]'`) · `internal/store/chunk_embedding_test.go:43` — `SELECT embedding IS NULL` ⇒ `nulo` obrigatório · **SQLite**: `cmd/mem/main.go:2202-2211` (o substituto zero saiu) e `:2218` — `if len(vec) > 0` (o TurboQuant só é quantizado com vetor real) · `cmd/mem/index_no_vector_test.go:112` — `comVec != 0` ⇒ erro · `:114-115` — `comTurbo != 0` ⇒ erro | **PASS** (os dois motores, e os **dois** caminhos do SQLite) |
| **AC2** — o doctor reporta o chunk sem vetor real (ausente **ou** vetor zero do schema) | desincronizado, com `Issue` dizendo qual | **Postgres**: `internal/store/doctor.go:190` — `AND (c.embedding IS NULL OR (c.embedding <#> c.embedding) = 0)` · `:186-187` — o `CASE` distingue "Missing vector embedding" de "Zero vector embedding" · `internal/store/doctor_zero_vector_test.go:69-79` (os três estados: NULL, zero, real) · **SQLite**: `internal/db/graph.go:344-353` — a decisão por chunk (cru zero / comprimido zero / ausente / cru sem comprimido) · `internal/db/doctor_zero_vector_test.go:105-139` (seis estados, com o cru e o comprimido isolados) | **PASS** (o `Issue` distingue os dois casos, como a AC exige) |
| **AC3** — o Health Score penaliza o chunk sem vetor real em 2 pontos, com o teto 15 da ADR-013 | score cai | `internal/store/doctor.go:57-62` — 2 pontos por chunk dessincronizado, teto 15 (matemática **inalterada**: o que mudou foi *o que é* dessincronizado) · `internal/store/doctor_zero_vector_test.go:85-93` — o score do relatório é exatamente `CalculateHealthScore(semDesinc) - 4` · `internal/store/doctor_test.go:59-69` (a penalidade da ADR-013) | **PASS** |
| **AC4** — a busca vetorial **não** devolve chunk com embedding ausente ou vetor zero, em **todo** caminho de leitura | ausente e zero fora | **Postgres**: `internal/store/postgres.go:595-596` — `AND c.embedding IS NOT NULL` + `AND (c.embedding <#> c.embedding) <> 0` · **SQLite cru (vec0)**: `internal/db/graph.go:104` — `if vetorCruzoEhZero(vetor) { continue }` (o `v.embedding` foi acrescentado ao `SELECT`) · **SQLite comprimido**: `internal/db/graph.go:187` — `if turboQuantEhZero(data, embedder.EmbeddingDim) { continue }` · `internal/store/knn_zero_vector_test.go:60-84` e `internal/db/knn_zero_vector_test.go:68-120` | **PASS** (os **três** caminhos) |
| **AC5** — no híbrido, o chunk sem vetor real é alcançável **apenas** pela fonte léxica | nunca pela fonte `vector` | `internal/db/knn_zero_vector_test.go:145-174` — `TestHybridZeroVectorSoAlcancadoPelaFonteLexica`: o chunk aparece no resultado fundido e nenhuma fonte tem prefixo `vector` · o mesmo teste falha com a mutação M-T6b (`o chunk sem vetor real apareceu na fonte "vector:1"`) | **PASS** |
| **AC6** — a nota escrita pelo caminho da lição, indexada com provider, volta por uma consulta **sem palavra em comum**, pela fonte `vector` | recuperação por sentido | `cmd/mem/vault_retrieval_test.go:122` — a nota (por slug) tem de estar no resultado · `:127` — `Contains(out, "vector:")` · `:132` — `!Contains(out, "fts:")` · `:135` — `!Contains(out, "LEXICAL")` · a consulta (`"como justificar uma escolha relevante"`) não tem **nenhuma** palavra em comum com a nota (título, corpo ou frontmatter), e o FTS5 casa os termos com AND | **PASS** (rodou com o **provider ONNX real**, tag `onnx`) |
| **AC7** — o cofre reindexado do zero com provider fica com **0** chunks sem vetor real | zero | `cmd/mem/vault_retrieval_test.go:196` — `semVetorDepois != 0` ⇒ erro (a contagem vem do **doctor**, a mesma definição das AC2) · `:199-203` — documentos e chunks com a mesma contagem de antes · `:172` — o teste **não é vacuous**: antes do reparo tem de haver chunk sem vetor | **PASS** (o caminho é o do reparo: indexa **sem** provider, depois reindexa do zero **com** provider) |

**Status**: 7/7 ACs PASS · 0 PARTIAL · 0 FAIL de comportamento.

---

## Discrimination Sensor

5 sensores, 16 mutações, **14 mortas** e **2 mutantes equivalentes registrados** (a
distinção importa: um mutante equivalente não é um gap — é uma asserção que não
discrimina porque o código está certo por outra razão).

### T3 — o indexador SQLite (`D:\.minimax\scratch\sensor_t3.py`)

| Mutação | Efeito | Morta? |
| --- | --- | --- |
| M-T3a | o substituto vetor zero volta ao caminho SQLite | **MORTA** — `comVec` **e** `comTurbo` acusam linha (`index_no_vector_test.go:112` e `:115`) |
| M-T3b | só o caminho comprimido volta a gravar o TurboQuant zero | **MORTA** — `comTurbo` acusa linha (`:115`) |
| M-T3c | a guarda `if len(vec) > 0` é removida | **SOBREVIVEU — mutante equivalente**: o próprio `Quantize` rejeita dimensão inválida (`esperado 384, recebido 0`) e não grava nada. A guarda fica por explicitar o requisito da AC1 |

### T4 — o doctor do Postgres (`sensor_t4.py`)

| Mutação | Efeito | Morta? |
| --- | --- | --- |
| M-T4a | a condição volta a `c.embedding IS NULL` (o filtro antigo) | **MORTA** — o chunk de vetor zero deixa de ser reportado (`doctor_zero_vector_test.go:76`) |
| M-T4b | o `CASE` some (todo dessincronizado vira "Missing vector embedding") | **MORTA** — o `Issue` deixa de distinguir ausente de zero (`:77`) |
| M-T4c | a query volta a dois campos enquanto o `Scan` lê três | **MORTA** — o relatório fica **vazio sem erro visível** (o `if err == nil` engole o erro do `Scan`): 0 reportados (`:65`) |

### T5 — o doctor do SQLite (`sensor_t5.py`)

| Mutação | Efeito | Morta? |
| --- | --- | --- |
| M-T5a | `vetorCruzoEhZero` deixa de ver o zero | **MORTA** — o chunk com vetor **cru** zero passa a ser reportado como "Missing TurboQuant" (`doctor_zero_vector_test.go:132`) |
| M-T5b | `turboQuantEhZero` deixa de ver o zero | **MORTA** — o chunk com vetor **comprimido** zero não é reportado (`:136`) |
| M-T5c | o caso da ausência some | **MORTA** — o chunk sem vetor nenhum não é reportado (`:106`) |
| M-T5d | a regra "vetor cru sem o comprimido" some | **MORTA** — o chunk que a busca comprimida não encontraria não é reportado (`:122`) |

### T6 — a busca vetorial (`sensor_t6.py`)

Cada mutação foi rodada nos **três** caminhos (SQLite comprimido, SQLite cru com a tag
`sqlite_fts5`, e Postgres).

| Mutação | Efeito | Morta? |
| --- | --- | --- |
| M-T6a | os dois filtros do Postgres saem da query do KNN | **MORTA** — a busca **erra**: `Scan error ... converting NULL to float64` (o chunk sem embedding entra com distância nula) |
| M-T6b | o filtro do vetor zero sai do caminho comprimido | **MORTA** — o chunk volta pelo TurboQuant; a busca "todos sem vetor" devolve 1 resultado; no híbrido ele aparece na fonte `vector:1` |
| M-T6c | o filtro do vetor zero sai do caminho cru (vec0) | **MORTA** — o chunk volta pelo KNN do sqlite-vec (`knn_zero_vector_test.go:86`) |
| M-T6d | só o filtro `IS NOT NULL` sai do Postgres | **SOBREVIVEU — mutante equivalente**: `(embedding <#> embedding) <> 0` já exclui o NULL (`NULL <> 0` não é verdadeiro). A cláusula fica por explicitar o requisito |

### T7/T8 — as provas E2E (`sensor_t7.py`)

| Mutação | Efeito | Morta? |
| --- | --- | --- |
| M-T7a | `newBuiltinEmbedder` devolve um provider nulo | **MORTA** — "a busca por paráfrase não trouxe a nota do cofre" (`vault_retrieval_test.go:123`) |
| M-T8a | o vetor real do provider é substituído pelo vetor zero | **MORTA** — "depois do reindex do zero, 1 chunk(s) continuam sem vetor real" (`:197`) |

---

## Gates

| Gate | Comando | Resultado |
| --- | --- | --- |
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./internal/db/... ./internal/store/...` | exit 0 |
| Formato | `gofmt -l` sobre os **blobs do índice** (byte-exatos) dos arquivos tocados | vazio |
| Full | `MY_MEMORY_PG_TEST=1 go test -count=1 ./...` (Postgres Neon, endpoint direto) | **26/26 pacotes `ok`, 0 FAIL, exit 0** |
| Taggeado | `CGO_ENABLED=1 go test -tags "sqlite_fts5 onnx" -count=1 ./...` | exit 0 (ver abaixo) |
| Spec | `validate_spec.py` + `validate_tasks.py` | exit 0 |
| State | `validate_state.py fix-zero-vector-chunks` | exit 0 |

### O caminho `vec0` verificado localmente

O caminho `sqlite-vec` (`HasSqliteVec == true`) **não existe no Windows** — o build tag
é `(darwin || linux) && cgo && sqlite_fts5`. Para não deixar o pedaço mais arriscado
(uma query nova na tabela virtual `vec0`) verificado só pela CI, o build tag foi
relaxado **temporariamente** (com um shim `sqlite3.h` apontando para o header do
`mattn/go-sqlite3`) e **revertido** depois — ele não pode ir para o commit, porque
quebraria o build do Windows sem o shim. Com ele, a suíte `internal/db` inteira passou
com a tag `sqlite_fts5` (o mesmo comando que a CI roda no Ubuntu) e a mutação M-T6c
morreu **no caminho vec0**.

---

## Achado durante o gate (e o que ele revela)

A primeira rodada do gate full **reprovou** em um teste: `internal/watcher/mtime_paths_test.go`.
O fixture indexava a semente **sem provider** e relyia no vetor zero para o caminho
vetorial do TurboQuant devolver o documento. Com a correção, o chunk sem provider é
gravado sem linha em `chunks_turboquant` — e a busca vetorial, com o filtro do vetor
zero, **não devolve chunk sem similaridade de verdade**: o caminho vetorial do fixture
deixou de ser exercitado.

O fixture foi corrigido (`d9c5ebb`): a semente passa a receber um provider de mentira
que devolve um vetor **real** (primeira componente = 1). O que o teste prova — o mtime
populado em cada caminho de leitura — segue intacto.

> [!NOTE]
> Este achado é a **prova** de que o defeito era real: havia um teste da suíte que só
> passava porque o produto gravava vetor zero. Ele dependia do defeito.

---

## Riscos residuais e o que fica em aberto

1. **O dado já gravado não se conserta sozinho.** Chunks com vetor zero (ou ausentes)
   continuam assim até um `mem index --force` com provider presente. O doctor passa a
   **listá-los** — que é o ponto —, mas o reparo é manual.
2. **A verificação foi feita pelo autor** (ver o aviso do topo). Um Verifier independente
   deve reexecutar antes do release.
3. **A comparação do vetor zero no Postgres depende do operador `<#>` do pgvector** —
   validado contra o Postgres do dono (Neon, pgvector) e contra o `pgvector/pgvector:pg16`
   da CI. É uma semântica estável do pgvector, não uma heurística.
4. **A regra do SQLite "vetor cru sem o comprimido" continua sendo reportada.** Ela não
   estava no texto da AC2 (que fala de ausência e zero), mas já existia no produto e
   cobre o caso em que a busca no caminho comprimido não encontraria o chunk — foi
   **mantida**, e a AC2 continua PASS por cima dela.
5. **A spec 3 (`fix-doctor-unindexed-links`) segue em aberto** — ela é o último gap
   medido do Cofre Central (138 wikilinks que existem em disco são contados como
   quebrados).
