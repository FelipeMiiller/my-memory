# Feature: batch-code-bridge-writes

- **Spec ID**: `batch-code-bridge-writes`
- **Created**: 2026-10-29
- **Owner**: Felipe Miiller
- **Source**: medição do dono no Postgres remoto (Neon, us-east-1) — ponte `notes↔código` e pipeline de símbolos do code stage
- **Related**: ADR-001 (SQLite unificado) · ADR-040 (precedência de storage) · ADR-047 (code AST) · ADR-061 (code graph no engine configurado) · ADR-011 (status epistêmico)

## Problem Statement

O dono habilitou o escopo de código (`code:` no `.memory/config.yaml`) e mediu, contra o **Postgres remoto (Neon, us-east-1)**, o custo do caminho de escrita do code graph: **≈268 ms por round-trip**.

Há **dois** pontos de escrita com o mesmo defeito de classe — um statement por linha, N linhas por arquivo/documento.

### Ponto 1 — a ponte `notes↔código` (`internal/db/code_bridge.go`)

`buildCodeBridge` casa o texto de cada documento contra os `code_symbols` (em memória, barato) e então grava **uma linha por vez**: `upsertBridgeNode` (1 `INSERT ... ON CONFLICT DO NOTHING`) e `upsertBridgeEdge` (1 `INSERT ... ON CONFLICT DO UPDATE`) por par documento→símbolo. São **2 round-trips por par**.

Neste vault: ~276 documentos e ~34 000 arestas `references_code` → **≈68 000 round-trips** → **≈4,7 horas** só na ponte. O mesmo defeito torna `BuildCodeBridgeForDocument` (usado por `internal/watcher/indexer.go:154` e `:293`) custar **≈1 minuto por documento**.

### Ponto 2 — o pipeline de símbolos (`internal/codeast/cache.go:160` → `internal/db/code_migration.go`)

`PersistSymbols` chama `db.UpsertCodeSymbol` **uma vez por símbolo**. Cada chamada faz 1 `INSERT ... ON CONFLICT` e, como o `lib/pq` não implementa `LastInsertId`, um `SELECT` extra para resolver o id → **2 round-trips por símbolo**. Medição do dono: **0,78 símbolos/s**; com ~6 750 símbolos no repo → **≈2 horas** só para popular `code_symbols`.

O mesmo vale para as arestas: `FlushEdges`/`resolveAndInsertEdges` chamam `db.UpsertCodeEdgeWithStatus` **uma vez por aresta** (`code_edges` tem ≈1 aresta por símbolo no repo).

### Ponto 3 — o bucket de arestas incertas (`internal/codeast/pipeline.go` → `db.InsertCodeEdgeUncertain`)

**Medido pelo dono em produção depois da primeira entrega desta feature** (run no Neon de 08:24→08:41, ≈268 ms/round-trip):

| bucket | escrita | taxa medida | fila estimada (repo inteiro) |
| :--- | :--- | :--- | :--- |
| `code_symbols` | 2 round-trips por **símbolo** (antes) | 0,78/s (antes) | ~2 h → **~5 min** (com o lote) ✅ |
| `code_edges_uncertain` | **1 statement por aresta** | **~1,6/s** | **~110 min** ❌ |

Ou seja: o lote dos símbolos destravou ~95% do custo, mas o bucket de incertas ficou no caminho antigo e sozinho leva mais de 1h50 no Neon. Como as arestas **resolvidas** só são gravadas no `FlushEdges` final, uma interrupção ali perde o `code_edges` inteiro — não dá para "deixar rodando".

Este ponto **estava fora do escopo** na primeira entrega (decisão A11, "volume residual") e foi **refutado pela medição** — ver A12.

### Por que os testes não pegaram isso

No SQLite local cada statement é praticamente instantâneo, e a suíte usa volumes minúsculos. O defeito só aparece quando o engine é **remoto** e o lote é **grande** — a combinação que o dono acabou de ativar.

### O que muda

- A ponte passa a gravar, **por documento**, **1** `INSERT` multi-linha em `graph_nodes` + **1** `INSERT` multi-linha em `graph_edges` (nós **antes** das arestas, pela FK `graph_edges → graph_nodes`).
- O pipeline passa a gravar, **por arquivo**, **1** `INSERT` multi-linha em `code_symbols` + **no máximo 1** `SELECT` para resolver os ids, e **1** `INSERT` multi-linha em `code_edges` por arquivo.
- **O que é gravado não muda**: mesmas linhas, mesmas colunas, mesmo `ON CONFLICT`, mesmo `epistemic_status`, mesma idempotência, mesmo mapeamento símbolo→id (que é o que as arestas usam).

## Goals

- [x] A ponte grava **2 statements por documento** (1 nós + 1 arestas), independentemente de quantos símbolos o documento casa.
- [x] O pipeline de símbolos grava **1 statement por arquivo** (mais, no máximo, 1 `SELECT` de resolução de ids).
- [x] O flush de arestas grava **1 statement por arquivo**, em vez de 1 por aresta.
- [x] O bucket de arestas **incertas** (`code_edges_uncertain`) grava **1 statement por arquivo**, em vez de 1 por aresta (decisão refutada pela medição de 29/10 — ver A12).
- [x] Mesma semântica: mesmas linhas, mesmas colunas, mesmo alvo de `ON CONFLICT`, mesmo `epistemic_status`, mesma idempotência.
- [x] O id devolvido para o símbolo `i` é o id **daquele** símbolo (o das arestas), nos dois engines.
- [x] SQLite e Postgres com o mesmo caminho de código (só o dialeto muda).
- [x] Teste que **falha contra o comportamento antigo** (prova de que não é verde vacuamente).
- [x] Postgres real (banco descartável do `internal/pgtest`) provando o `INSERT` multi-linha com `ON CONFLICT` nos dois engines.

## Out of Scope

- **Schema** — nenhuma tabela, coluna, índice ou restrição muda. Nenhum ADR novo (ver A9).
- **Driver** — `lib/pq` continua; `go.mod`/`go.sum` intocados.
- **O cálculo do casamento da ponte** — literais por qualified name, por path e o match fraco por nome ficam **intocados**. Muda **só** como as escritas são emitidas.
- **`UpsertCodeFile`** — as ~2 round-trips por arquivo ficam como estão (irrelevantes ao lado do resto).
- **`code_edges_uncertain`** — ~~as arestas de baixa confiança continuam uma por uma (volume residual)~~. **Decisão REFUTADA pela medição de 29/10**: o bucket é o último gargalo do code stage no engine remoto (≈110 min). Entra no escopo (ver A12).
- **`internal/store/**`, `internal/mcp/**`, `cmd/mem/**`, `viewer/**`, `.memory/**`** — fora da propriedade desta execução.
- **`.specs/ISSUES.md` e `.specs/STATE.md`** — fora da propriedade desta execução; pendência registrada para o gestor.
- **Rodar o pipeline contra o Postgres do dono** — o store real está em uso; o único Postgres tocado é o banco descartável do `internal/pgtest`.

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| A1 — Escopo | Os **dois** pontos de escrita: a ponte (`graph_nodes`/`graph_edges`) e o pipeline (`code_symbols`/`code_edges`) | Mesma classe de defeito, mesmo engine, mesma medição (268 ms/round-trip). O dono autorizou a ampliação no meio da execução | y |
| A2 — Como o `?` vira `$N` | O SQL gerado usa **`?`** e o `Dialect.Bind` converte (o caminho já validado em `internal/db/dialect_test.go`) | O repositório já tem `?` + `Bind` como convenção; trocar para `$N` na mão seria um segundo dialecto dentro do mesmo arquivo | y |
| A3 — Teto de linhas por statement | **2000 linhas** por statement (teto de segurança, não desempenho) | O limite que aperta é o de **parâmetros** do SQLite: `MAX_VARIABLE_NUMBER=32766` (medido) e o insert de símbolo usa 11 colunas → 2000×11 = 22 000 < 32 766. No Postgres o teto é 65 535 parâmetros. Acima de 2000 linhas o lote é partido em pedaços — o número de statements deixa de ser 1 e passa a ser `ceil(N/2000)` | y |
| A4 — Resolução dos ids dos símbolos | **1 `INSERT` multi-linha + 1 `SELECT` por arquivo**, com o casamento feito em Go (rota (b) do briefing) | O `RETURNING` funciona nos dois engines (medido: SQLite ≥ 3.35 e Postgres), mas a ordem das linhas devolvidas **não é garantida** pelo Postgres e o `INSERT ... RETURNING` por `QueryContext` não está coberto pelo build CGO (`mattn/go-sqlite3`) que o CI roda. A rota (b) usa só SQL portátil e dá o mesmo resultado | y |
| A5 — Qual id o `SELECT` devolve para uma chave com mais de uma linha | O **maior id** (a linha que o lote acabou de gravar) | A chave `(file_id, kind, qualified_name, start_line)` é única quando o `qualified_name` é preenchido; o único caso com mais de uma linha é o `qualified_name` vazio (NULL), em que o `INSERT` **acrescenta** uma linha. O id novo é o da linha viva | y |
| A6 — `ON CONFLICT` e o conflito dentro do próprio lote | Chaves duplicadas no mesmo statement são **removidas antes** (o payload que vence é o do **último** símbolo com a mesma chave) | O Postgres **rejeita** um `ON CONFLICT DO UPDATE` que afete a mesma linha duas vezes (`ON CONFLICT DO UPDATE command cannot affect row a second time`); o SQLite aceita. Remover a duplicata deixa os dois engines iguais e reproduz o caminho por símbolo (onde o segundo upsert atualiza a linha do primeiro) | y |
| A7 — Ordem das arestas gravadas | A ordem de `code_edges` é a ordem de resolução (como hoje); o conteúdo gravado não muda | A ordem não é contrato do schema (não há `id` de ordem nem leitura que dependa dela) e o `ON CONFLICT` decide por chave | y |
| A8 — Onde fica a costura de contagem de statements | Uma interface estreita de escrita (`bridgeWriter`) no `internal/db`, usada só pela ponte | O `*sql.DB` concreto não dá hook; a interface deixa o teste contar os statements **sem** mudar o `*sql.DB` nem o driver. No Postgres a contagem real é feita pelo `pgtest.Recorder`, que grava o SQL que chega ao servidor | y |
| A9 — ADR | **Nenhum ADR novo** | Não é mudança de schema, de driver nem de arquitetura: é a **forma** de emitir as mesmas escritas | y |
| A10 — `LastInsertId` no caminho por símbolo (SQLite) | O caminho em lote **não** usa `LastInsertId`: ele resolve o id pelo `SELECT` | Medido: no SQLite (modernc) o `LastInsertId` depois de um upsert que **atualiza** devolve o `last_insert_rowid` **velho** (o da última linha inserida), não o da linha atualizada — o caminho por símbolo pode devolver o id de outro símbolo. O lote resolve o id pela chave, que é o que as arestas precisam | y |
| A11 — Documentos/arquivos acima do teto de 2000 linhas | O lote é partido em pedaços de 2000 linhas, **nós antes das arestas** | Um documento com mais de 2000 símbolos casados é patológico; o corte é de segurança, não de desempenho, e não muda o que é gravado | y |
| A11b — `code_edges_uncertain` gravado aresta por aresta | **REFUTADA em 2026-10-29** — o bucket entra no escopo e é gravado em lote (1 statement por arquivo) | A decisão original ("volume residual") partiu do **código**, não da **medição**: o bucket é pequeno em número de linhas, mas cada linha custava 1 round-trip. No Neon (≈268 ms) isso é ~1,6 aresta/s → **≈110 min** para o repo inteiro, contra ~5 min dos símbolos. A medição do dono (run de 08:24→08:41) refutou a premissa: o gargalo é o **número de round-trips**, não o volume | y |
| A12 — A chave do bucket incerto | `UNIQUE(src_symbol_id, dst_candidate_name, kind)` — **verificada nos DOIS engines** (`internal/db/schema.go` e `internal/store/postgres.go`) | O `ON CONFLICT` do caminho atual aponta para uma chave que **existe** no schema: o lote pode manter o mesmo `ON CONFLICT` (mesma idempotência) sem criar índice e sem mudar schema. O lote funde a duplicata do próprio statement (o Postgres rejeita `ON CONFLICT DO UPDATE` da mesma linha duas vezes) | y |
| A13 — `UpsertCodeFile` | **Não mexido** — o custo real foi medido e reportado: **2 round-trips por arquivo no Postgres** (o `INSERT` + o `SELECT` do id, porque o `lib/pq` não implementa `LastInsertId`); 1 no SQLite | O dono pediu "só olhe e me diga o custo real". O custo importa só na **primeira** população (~375 arquivos × 2 ≈ 750 round-trips ≈ 3,4 min no Neon); no `mem index` de rotina (cache quente) o arquivo nem é gravado. Otimizar exigiria `RETURNING id` (1 round-trip) — decisão do dono, não do agente | y |

**Open questions**: none — as decisões estão resolvidas e registradas acima (A1 a A13).

## User Stories

### P1: A ponte não pode custar horas no engine remoto

**User Story**: Como dono de um vault indexado contra Neon, quero que a ponte notas↔código grave em lote, so that o `mem index` termine em minutos em vez de horas.

**Why P1**: ≈68 000 round-trips medidos (≈4,7 h) para um único passe da ponte, com o `mem index` já habilitado no escopo `code:`.

**Acceptance Criteria**:
1. WHEN um documento casa com N símbolos de código THEN a ponte SHALL emitir 2 statements de escrita (1 `INSERT` multi-linha em `graph_nodes` e 1 `INSERT` multi-linha em `graph_edges`), e não 2×N.
2. WHEN a ponte grava as arestas de um documento THEN o `INSERT` de `graph_nodes` SHALL ter sido executado antes do `INSERT` de `graph_edges`, porque `graph_edges` tem FK para `graph_nodes`.
3. WHEN o mesmo documento é indexado duas vezes THEN a ponte SHALL produzir exatamente o mesmo grafo (mesmos nós, mesmas arestas, mesmo `epistemic_status`) e SHALL NOT duplicar linhas.
4. WHEN a ponte roda nos dois engines THEN o `INSERT` multi-linha SHALL usar `INSERT OR IGNORE` no SQLite e `ON CONFLICT (repository, id) DO NOTHING` no Postgres para os nós, e `ON CONFLICT(source_id, target_id, relation)` no SQLite e `ON CONFLICT (repository, source_id, target_id, relation)` no Postgres para as arestas.
5. WHEN o documento casa com mais de 2000 símbolos THEN a ponte SHALL partir o lote em pedaços de 2000 linhas, mantendo os nós antes das arestas.

**Independent Test**: Semear N símbolos, indexar um documento que os menciona, contar os statements emitidos e conferir que o número é 2 (e não 2×N), que o grafo final tem N nós e N arestas, e que a segunda rodada não muda nada.

---

### P2: O pipeline de símbolos também grava em lote

**User Story**: Como dono de um vault com o escopo `code:` habilitado, quero que o code stage grave os símbolos de um arquivo em um statement, so that `mem code-index` populate `code_symbols` em minutos em vez de horas.

**Why P2**: 0,78 símbolos/s medidos no Neon (2 round-trips por símbolo) → ≈2 h para ~6 750 símbolos.

**Acceptance Criteria**:
1. WHEN um arquivo tem N símbolos THEN `PersistSymbols` SHALL emitir 1 `INSERT` multi-linha em `code_symbols` (mais, no máximo, 1 `SELECT` de resolução de ids), e não N `INSERT`s.
2. WHEN `PersistSymbols` devolve os ids THEN o id da posição `i` SHALL ser o id do símbolo `i` do arquivo, para que as arestas `src_symbol_id`/`dst_symbol_id` apontem para o símbolo certo.
3. WHEN o caminho em lote grava um arquivo THEN ele SHALL produzir as mesmas linhas (mesmas colunas, mesmo conteúdo) e os mesmos ids que o caminho por símbolo produz no mesmo estado inicial do banco.
4. WHEN o mesmo arquivo é indexado duas vezes THEN o caminho em lote SHALL NOT duplicar linhas de `code_symbols` e SHALL NOT alterar o `ON CONFLICT` já usado (`ON CONFLICT(file_id, kind, qualified_name, start_line)`).
5. WHEN dois símbolos do mesmo arquivo têm a mesma chave de conflito THEN o caminho em lote SHALL gravá-los como uma linha só (o payload do último), reproduzindo o que o caminho por símbolo faz.

**Independent Test**: Gravar um arquivo com N símbolos pelo caminho por símbolo num banco e pelo caminho em lote noutro, e comparar o conteúdo das linhas e o vetor de ids.

---

### P3: As arestas do code graph também saem em lote

**User Story**: Como dono do vault, quero que o flush das arestas do code stage grave em lote, so that o `code_edges` (~1 aresta por símbolo) não seja o novo gargalo.

**Why P3**: Com os símbolos em lote, 1 statement por aresta passaria a ser o gargalo restante (~6 750 round-trips).

**Acceptance Criteria**:
1. WHEN o lote de arestas é resolvido THEN cada arquivo SHALL ter suas arestas gravadas em 1 `INSERT` multi-linha em `code_edges`.
2. WHEN a mesma aresta aparece duas vezes no mesmo lote THEN o caminho em lote SHALL manter a menor `start_line`/`end_line` e o payload do último, como o `ON CONFLICT` por aresta já faz.
3. WHEN uma aresta tem `confidence < 0.5` THEN o caminho em lote SHALL recusá-la com o mesmo erro do caminho por aresta (`UseInsertCodeEdgeUncertain`).

**Independent Test**: Resolver um lote com arestas repetidas e conferir o estado final de `code_edges` (mesma linha, mesma posição, mesmo status epistêmico).

---

### P4: A prova é real, não verde por acidente

**User Story**: Como dono que confia no gate, quero que os testes novos falhem contra o código antigo, so that o verde signifique que o defeito foi corrigido.

**Why P4**: Um teste que passa nos dois comportamentos não prova nada.

**Acceptance Criteria**:
1. WHEN o caminho de escrita volta a emitir 1 statement por linha THEN o teste novo de contagem de statements SHALL falhar.
2. WHEN o caminho em lote é exercitado contra o Postgres real THEN o teste SHALL rodar contra o banco descartável do `internal/pgtest` e SHALL provar que o `INSERT` multi-linha com `ON CONFLICT` foi aceito pelo servidor.
3. WHEN o gate final roda THEN `go test -count=1 ./...`, `go build ./...` e `go vet ./...` SHALL sair com código zero, e `gofmt -l` SHALL ficar vazio nos arquivos tocados.

**Independent Test**: Voltar a emitir 1 statement por linha (mutante), rodar o teste novo e ver o sintoma exato da regressão.

---

### P5: O bucket de arestas incertas também sai em lote

**User Story**: Como dono de um vault com o escopo `code:` habilitado, quero que as arestas de baixa confiança sejam gravadas em lote, so that o `mem index` feche em minutos num Postgres remoto em vez de mais de 1h50.

**Why P5**: Medido pelo dono em produção (Neon, run de 08:24→08:41): o bucket `code_edges_uncertain` é gravado a ~1,6 aresta/s (1 statement por aresta) → **≈110 min** só para popular o repo. Era o último gargalo do code stage.

**Acceptance Criteria**:
1. WHEN um arquivo tem M arestas incertas THEN o bucket `code_edges_uncertain` SHALL ser gravado com 1 statement multi-linha por arquivo, e não M statements.
2. WHEN o lote tem a mesma chave de conflito duas vezes THEN a gravação SHALL produzir uma linha só, com o payload do ÚLTIMO (é o que o `ON CONFLICT` do caminho atual grava).
3. WHEN o mesmo arquivo é reprocessado THEN o bucket SHALL NOT duplicar linhas e o `ON CONFLICT(src_symbol_id, dst_candidate_name, kind)` SHALL ser o mesmo do caminho por aresta.
4. WHEN o `FlushEdges` roda THEN as arestas resolvidas (`code_edges`) e as incertas (`code_edges_uncertain`) SHALL ser gravadas em 1 statement por arquivo **em cada bucket**.

**Independent Test**: Semear 2 arquivos com arestas incertas, rodar o `FlushEdges` contra o Postgres descartável e contar os `INSERT` que chegam ao servidor (2 em cada bucket, não 4+4).

## Edge Cases

- IF o documento **não** casar com nenhum símbolo THEN a ponte SHALL NOT emitir nenhum statement de escrita (como hoje).
- IF o arquivo **não** tiver símbolos THEN `PersistSymbols` SHALL devolver uma lista vazia e SHALL NOT emitir statement.
- IF o símbolo tiver `qualified_name` **vazio** (NULL no banco) THEN o caminho em lote SHALL resolver o id pela chave coalescida (`COALESCE(qualified_name, '')`), como o caminho por símbolo já faz.
- IF dois símbolos do mesmo lote tiverem a mesma chave THEN o lote SHALL emitir a linha uma vez só (o Postgres rejeita o conflito dentro do statement).
- IF o lote passar de 2000 linhas THEN o caminho SHALL partir o statement, mantendo a ordem nós→arestas.
- IF o `SELECT` de resolução não encontrar a chave THEN o caminho em lote SHALL devolver erro explícito, em vez de devolver id 0 silencioso.
- IF o `ON CONFLICT` de `graph_edges` for atingido THEN o `epistemic_status` SHALL ser atualizado como hoje (`excluded.epistemic_status`).
- IF a mesma aresta incerta (mesmo `src_symbol_id`, mesmo candidato, mesmo `kind`) aparecer duas vezes no lote THEN o bucket SHALL ficar com **uma** linha, com o payload do **último** (é o que o `ON CONFLICT DO UPDATE` grava em todas as colunas).
- IF o símbolo de origem da aresta incerta for apagado THEN o `ON DELETE CASCADE` da tabela SHALL continuar apagando a aresta incerta (o schema não muda).
- IF o lote de arestas incertas for **vazio** THEN o caminho em lote SHALL NOT emitir nenhum statement.

## Requirement Traceability

| Requirement ID | AC | Story | Arquivo | Fase | Status |
| -------------- | -- | ----- | ------- | ---- | ------ |
| BR-01 | P1.1 | P1: A ponte não pode custar horas no engine remoto | internal/db/code_bridge.go | Execute | Verified |
| BR-02 | P1.2 | P1: A ponte não pode custar horas no engine remoto | internal/db/code_bridge.go | Execute | Verified |
| BR-03 | P1.3 | P1: A ponte não pode custar horas no engine remoto | internal/db/code_bridge.go | Execute | Verified |
| BR-04 | P1.4 | P1: A ponte não pode custar horas no engine remoto | internal/db/code_bridge.go | Execute | Verified |
| BR-05 | P1.5 | P1: A ponte não pode custar horas no engine remoto | internal/db/code_bridge.go | Execute | Verified |
| SYM-01 | P2.1 | P2: O pipeline de símbolos também grava em lote | internal/db/code_migration.go | Execute | Verified |
| SYM-02 | P2.2 | P2: O pipeline de símbolos também grava em lote | internal/codeast/cache.go | Execute | Verified |
| SYM-03 | P2.3 | P2: O pipeline de símbolos também grava em lote | internal/db/code_migration.go | Execute | Verified |
| SYM-04 | P2.4 | P2: O pipeline de símbolos também grava em lote | internal/db/code_migration.go | Execute | Verified |
| SYM-05 | P2.5 | P2: O pipeline de símbolos também grava em lote | internal/db/code_migration.go | Execute | Verified |
| EDG-01 | P3.1 | P3: As arestas do code graph também saem em lote | internal/codeast/pipeline.go | Execute | Verified |
| EDG-02 | P3.2 | P3: As arestas do code graph também saem em lote | internal/db/code_migration.go | Execute | Verified |
| EDG-03 | P3.3 | P3: As arestas do code graph também saem em lote | internal/db/code_migration.go | Execute | Verified |
| PRV-01 | P4.1 | P4: A prova é real, não verde por acidente | internal/db/code_bridge_batch_test.go | Execute | Verified |
| PRV-02 | P4.2 | P4: A prova é real, não verde por acaso | internal/db/code_batch_postgres_test.go | Execute | Verified |
| PRV-03 | P4.3 | P4: A prova é real, não verde por acaso | .specs/features/batch-code-bridge-writes/validation.md | Execute | Verified |
| UNC-01 | P5.1 | P5: O bucket de arestas incertas também sai em lote | internal/db/code_migration.go | Execute | Verified |
| UNC-02 | P5.4 | P5: O bucket de arestas incertas também sai em lote | internal/codeast/pipeline.go | Execute | Verified |
| UNC-03 | P5.2, P5.3 | P5: O bucket de arestas incertas também sai em lote | internal/db/code_migration.go | Execute | Verified |
| UNC-04 | P5.1, P5.3 | P5: O bucket de arestas incertas também sai em lote | internal/codeast/batch_postgres_test.go | Execute | Verified |
| EDG-04 | P5.4 | P5: O bucket de arestas incertas também sai em lote | internal/codeast/batch_postgres_test.go | Execute | Verified |

**Coverage**: 21 total, 21 mapeados para tarefas, 0 não mapeados.

## Success Criteria

- [x] A ponte emite 2 statements por documento (qualquer que seja o número de símbolos casados).
- [x] `PersistSymbols` emite 1 `INSERT` (+1 `SELECT`) por arquivo, com os ids na ordem da entrada.
- [x] O flush de arestas emite 1 statement por arquivo.
- [x] O bucket de arestas incertas emite 1 statement por arquivo, com o mesmo `ON CONFLICT` e a mesma idempotência do caminho por aresta.
- [x] O custo real do `UpsertCodeFile` foi medido e reportado (2 round-trips por arquivo no Postgres), sem mexer no código dele.
- [x] O grafo final e as tabelas `code_*` ficam idênticos aos do caminho por símbolo.
- [x] Postgres real (banco descartável) provando o `INSERT` multi-linha com `ON CONFLICT` nos dois engines.
- [x] Prova de que o teste novo falha contra o comportamento antigo (mutante registrado na `validation.md`).
- [x] `go test -count=1 ./...` (com `MY_MEMORY_PG_TEST=1`), `go build ./...` e `go vet ./...` com exit 0; `gofmt -l` vazio.
