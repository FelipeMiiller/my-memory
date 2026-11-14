# Feature: fix-zero-vector-chunks

- **Spec ID**: `fix-zero-vector-chunks`
- **Created**: 2026-11-13
- **Owner**: Felipe Miiller
- **Source**: ISSUE-040 — `.specs/ISSUES.md` (seção ISSUE-040, aberta nesta rodada)
- **Related**: ADR-013 (chunks dessincronizados no Health Score) · ADR-035 (provider embutido) · ADR-062 §C e §B-4 · ISSUE-037 (`fix-release-onnx-provider`)

## Problem Statement

Quando não há provider de embedding disponível, o indexador **não deixa o chunk
sem vetor**: ele substitui por um vetor **zero** de 384 dimensões
(`cmd/mem/main.go:1913` e o caminho equivalente do SQLite). Três consequências,
todas silenciosas:

1. **O chunk parece saudável.** O check de desincronização do doctor procura
   `embedding IS NULL` (`internal/store/doctor.go:176`) — um vetor zero **não**
   é NULL, então o doctor não vê nada. O Health Score do cofre não cai, e o
   dono lê "vault saudável" sobre um índice sem um único vetor real.
2. **A busca vetorial devolve lixo com cara de resultado.** Um vetor zero
   numa coluna `vector(384)` do pgvector torna a distância de cosseno
   indefinida; os hits ordenados por NaN não distinguem "relevante" de
   "irrelevante", e a resposta parece uma busca normal.
3. **A afirmação "os 481 chunks têm vetor" só vale para quem tem o provider.**
   O ADR-062 mediu isso com o binário local compilado com `-tags onnx`; o
   binário oficial (sem a tag) grava zeros — que é a forma exata de "a
   aplicação não está funcionando".

O SQLite **já sabe** gravar chunk sem vetor: `internal/db/store.go:357` só
insere em `chunks_vec` quando `len(vec) > 0`. O substituto zero em
`cmd/mem/main.go` é que estraga os dois engines.

**A dimensão que faltava (medida em 2026-11-14, pelo sensor da spec
`fix-release-onnx-provider`): o SQLite tem DOIS caminhos de armazenamento
do vetor, e o Postgres tem UM.**

| Engine | Onde o vetor é gravado | Onde a busca vetorial lê |
| :--- | :--- | :--- |
| PostgreSQL | `chunks.embedding` (`vector(384)`, nullable) | `chunks.embedding` (pgvector) |
| SQLite | `chunks_vec` (float32) **e** `chunks_turboquant` (4-bit comprimido) | `chunks_vec` quando `sqlite-vec` está disponível; **`chunks_turboquant`** no Windows, onde `db.HasSqliteVec` é `false` (`internal/db/vec_cgo.go:1` restringe o binding a `darwin` e `linux`) |

Ou seja: **zerar a coluna `chunks.vec` não mata a busca vetorial no
Windows** — o sensor provou que o mutante sobrevive, porque a busca lê as
linhas TurboQuant. E o KNN do Postgres (`internal/store/postgres.go:567-582`)
**não filtra `embedding IS NULL`**: um chunk sem embedding entra no
resultado com distância nula.

Por isso o defeito não é "o vetor zero": é "o vetor zero **e** o
TurboQuant zero **e** o embedding NULL que passa pelo filtro".

## Goals

- [ ] Um chunk indexado sem provider fica **sem embedding**, e não com um vetor
      zero.
- [ ] O doctor detecta o chunk sem vetor real (ausente **ou** zero).
- [ ] A busca vetorial não devolve chunk sem vetor real.
- [ ] O cofre central, reindexado do zero com um provider funcionando, fica com
      **0** chunks sem vetor real.

## Out of Scope

| Item | Reason |
| :--- | :--- |
| Trocar o provider ou o modelo | ADR-035 decide; o provider é fixo e local. |
| Recalcular os vetores já gravados | O reparo é `mem index --force` com provider — a spec `fix-release-onnx-provider` garante que o binário tenha um. |
| Mudar a penalidade do Health Score | A ADR-013 fixa 2 pontos por chunk dessincronizado (teto 15); aqui só muda **o que é dessincronizado**. |
| Gravar um "epoch" por chunk | O `embed_epoch` já existe no documento (SQLite) e resolve a invalidação por troca de embedder; não é o problema desta spec. |
| O `include_code` do RRF | Caminho do vault de repositório (ADR-061), não do cofre central. |

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| Como o chunk fica "sem vetor" no Postgres | `embedding` NULL | a coluna é `vector(384)` **sem** `NOT NULL` (`internal/store/postgres.go:53`); NULL é o valor natural de ausência e já é o que o doctor procura | y |
| Como o chunk fica "sem vetor" no SQLite | **nenhuma linha** em `chunks_vec` **nem** em `chunks_turboquant` | é o que `internal/db/store.go:357` já faz quando `len(vec) == 0`; a linha TurboQuant é a mesma decisão, porque no Windows é ela que a busca vetorial lê | y |
| Como o doctor detecta o vetor zero | compara o vetor com o vetor zero do schema e trata igualdade como ausência | o vetor zero é indistinguível de um vetor real por "existe"; a comparação é o que torna o sintoma visível | y |
| A busca vetorial exclui o chunk sem vetor | a query vetorial filtra o chunk sem vetor real | um chunk sem vetor não tem similaridade; devolvê-lo é devolver ruído com aparência de resultado | y |
| O reparo dos zeros já gravados | `mem index --force` com provider presente | é o mesmo caminho que o ADR-062 já mediu; nenhum reparo novo é necessário | y |
| O teste E2E depende da tag `onnx` | o teste é taggingado e **pula com mensagem explícita** sem a tag | sem provider não há o que provar; o skip explícito evita verde-mentira | y |

Open questions: none

## User Stories

### P1: O chunk sem provider fica visível ⭐ MVP

**User Story**: Como dono do cofre, quero que o doctor me diga quando um chunk
não tem vetor de verdade, para eu saber que a indexação rodou sem provider em
vez de confiar num score que não viu nada.

**Why P1**: hoje o sintoma é invisível — o vetor zero passa por "chunk
sincronizado" e o score não se move.

**Acceptance Criteria**:

1. WHEN a document is indexed without an available embedding provider THEN each
   of its chunks SHALL be written without an embedding (NULL in Postgres, no
   row in `chunks_vec` **and** no row in `chunks_turboquant` in SQLite)
   instead of an all-zero vector.
2. The doctor SHALL report a chunk as desynchronized when it has no vector at
   all (no `embedding` in Postgres, no row in `chunks_vec` and no row in
   `chunks_turboquant` in SQLite) or an all-zero vector of the schema
   dimension.
3. The Health Score SHALL penalize a chunk without a real vector at 2 points,
   with the ADR-013 cap of 15.

**Independent Test**: indexar uma nota com o binário **sem** a tag `onnx` e
rodar o doctor: o chunk aparece como dessincronizado e o score cai.

---

### P1: A busca não devolve chunk sem vetor

**User Story**: Como dono que busca no cofre, quero que a busca vetorial só
traga chunk que tem similaridade de verdade, para não receber resultado que
pareça relevante e não seja.

**Why P1**: vetor zero em `vector(384)` torna a distância de cosseno
indefinida; o resultado sai sem ordem confiável e com aparência de acerto.

**Acceptance Criteria**:

4. WHEN the vector search runs THEN it SHALL NOT return a chunk whose embedding
   is absent or an all-zero vector — in **every** storage path the engine
   reads: the raw vector (`chunks_vec` / `chunks.embedding`) and the
   compressed one (`chunks_turboquant`).
5. WHEN the hybrid search runs THEN the chunk without a real vector SHALL be
   reachable only through its lexical source, never through the vector one.

**Independent Test**: com um chunk de vetor zero no índice, a busca vetorial não
o devolve.

---

### P2: A prova de que o cofre guarda o que aprendi

**User Story**: Como dono do cofre central, quero a prova de que uma nota
escrita por mim volta quando eu pergunto **pelo sentido**, e não só pela mesma
palavra, para que o cofre valha como memória.

**Why P2**: é o critério de prontidão do cofre central na ADR-062 §B-3 e §B-4 —
"recuperável por sentido" e "reindexável do zero sem perder o vetor". Nenhum
dos dois tem prova hoje.

**Acceptance Criteria**:

6. WHEN a note written through the lesson path is indexed with a working
   provider THEN a query that shares no keyword with the note SHALL retrieve it
   through the vector source.
7. WHEN the central vault is reindexed from scratch with a working provider THEN
   the count of chunks without a real vector SHALL be zero.

**Independent Test**: o teste E2E (tag `onnx`) escreve uma nota, indexa, busca
por uma paráfrase e confere a origem do hit; depois força a reindexação e
confere a contagem de chunks sem vetor.

## Edge Cases

- IF the provider is available but a single chunk fails to embed THEN only that
  chunk is written without an embedding, and the run reports it (the failure is
  not silent).
- IF the text of a chunk is empty THEN the chunk is written without an
  embedding and is reported as such.
- IF the provider is unavailable THEN no TurboQuant row SHALL be written
  either — a linha comprimida com todos os coeficientes zerados é o mesmo
  defeito no caminho comprimido, e é por ela que a busca vetorial continua
  devolvendo o chunk no Windows.
- IF a document is reindexed without provider while it already has real vectors
  THEN the real vectors SHALL be preserved (o gate de `embed_epoch` existe:
  `cmd/mem/index_cache_test.go:30`).
- IF the vault is reindexed with `--force` without a provider THEN the chunks
  are written without embeddings, and the doctor reports every one of them.
- IF the model file is missing on disk THEN the provider fails with an error
  naming the file — never a silent zero vector.

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Where |
| :--- | :--- | :--- |
| ZVC-01 | AC1 — chunk sem provider fica sem embedding | `cmd/mem/main.go` · `internal/store/postgres.go` · `internal/db/store.go` |
| ZVC-02 | AC2 — o doctor detecta ausência e vetor zero | `internal/store/doctor.go` · `internal/db/graph.go` |
| ZVC-03 | AC3 — o Health Score penaliza o chunk sem vetor real | `internal/store/store.go` |
| ZVC-04 | AC4 — a busca vetorial exclui o chunk sem vetor real | `internal/store/postgres.go` · `internal/db/graph.go` |
| ZVC-05 | AC5 — o híbrido só alcança o chunk pela fonte léxica | `internal/store/postgres.go` · `internal/db/hybrid.go` |
| ZVC-06 | AC6 — a nota volta por sentido | `cmd/mem/vault_retrieval_test.go` |
| ZVC-07 | AC7 — reindex do zero deixa 0 chunks sem vetor | `cmd/mem/vault_retrieval_test.go` |

**Coverage**: 7 total, 7 mapeados, 0 unmapped.

## Success Criteria

- [ ] Indexar sem provider não grava nenhum vetor zero (verificável por query no
      banco: `embedding IS NULL` para o chunk).
- [ ] O doctor reporta o chunk sem vetor real e o score cai.
- [ ] A busca vetorial não devolve chunk sem vetor real.
- [ ] O teste E2E (com a tag `onnx`) prova a recuperação por sentido e o
      reindex do zero.
