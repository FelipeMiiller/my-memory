# Feature: fix-index-tolerancia

- **Spec ID**: `fix-index-tolerancia`
- **Created**: 2026-10-29
- **Owner**: Felipe Miiller
- **Source**: ISSUE-032 — `.specs/ISSUES.md:1263` (opção **C** da tabela "Opções de correção")
- **Related**: ADR-001 (SQLite unificado) · ADR-040 (precedência de storage) · ADR-057 (mtime/size) · ISSUE-010 (título do frontmatter)

## Problem Statement

Contra um Postgres com **pooler transacional**, o driver `lib/pq` usa o statement "unnamed" do servidor. Ele se corrompe e passa a responder `26000` (`unnamed prepared statement does not exist`) e `08P01` (`bind message supplies N parameters, but prepared statement "" requires M`). A partir daí **toda** query parametrizada daquela sessão falha.

O `mem index` **aborta a indexação inteira** no primeiro erro fatal: em `cmd/mem/main.go:1828` (Postgres) o `return err` da gravação do documento sai do `filepath.WalkDir`; em `cmd/mem/main.go:2047` (SQLite) o mesmo. O resultado é um vault **parcialmente indexado sem aviso de que sobrou documento de fora** — e sem o resumo final, porque o comando morre antes dele.

O precedente que já existe no próprio arquivo é o do chunk: `InsertChunk` com erro gera aviso e a varredura continua (`cmd/mem/main.go:1876-1882`). O que falta é o mesmo tratamento no passo **anterior** — a gravação do documento.

### O que muda

A falha de gravação de **um** documento deixa de ser fatal e vira **dado reportado**:

- a varredura segue para o próximo documento;
- o documento que falhou aparece com nome de arquivo e erro;
- o resumo final carrega a contagem de falhas;
- o documento que falhou **não** conta como indexado;
- o documento que falhou **não** pode ser podado pelo `--prune` (ele já entra em `activeDocIDs` antes da gravação — `cmd/mem/main.go:1765` e `:1984` — e isso é mantido).

A **opção C não conserta a causa**: ela troca "a indexação inteira morreu sem aviso" por "um documento falhou e foi reportado". A opção A (endpoint direto) já foi aplicada pelo dono fora do repositório; a opção B (trocar `lib/pq` por `pgx`) está **adiada** e não faz parte deste escopo.

## Goals

- [x] A falha de gravação de um documento **não** aborta a varredura: o `mem index` segue para o próximo documento.
- [x] A falha fica **visível**: aviso por documento (nome do arquivo + erro) e resumo final com a contagem.
- [x] Documento que falhou **não** conta como indexado.
- [x] Documento que falhou **não** é podado pelo `--prune`.
- [x] Erros de **nível de varredura** continuam abortando — a tolerância não engole tudo.
- [x] O mesmo comportamento vale no caminho **SQLite**: o contrato é do comando, não do driver.
- [x] Código de saída **zero** quando a varredura termina; a contagem de falhas fica no relatório.
- [x] Teste hermético que **falha contra o comportamento antigo** (sem rede, sem Postgres, sem binário).

## Out of Scope

- **Trocar o driver (`lib/pq` → `pgx`)** — opção B, adiada pelo dono. `go.mod` e `go.sum` não são tocados.
- **Consertar a causa do protocolo** (endpoint direto, `default_query_exec_mode`) — opção A, já aplicada pelo dono fora do repositório.
- **Mudar schema, tabelas ou ADR** — nenhum ADR novo é necessário (ver A10).
- **`internal/store/**` e `internal/db/**`** — nenhuma assinatura muda; a costura de teste vive no package `main`.
- **`internal/watcher/indexer.go`** — o watcher tem o próprio indexador (cópia da lógica de `mem index`); o dono do escopo desta feature é o comando `mem index`.
- **`--prune` de documentos que falharam** — eles **não** são podados; a proteção é mantida, não nova.
- **Falha na leitura do arquivo** (`os.ReadFile`) — continua ignorada em silêncio, como antes (ver A6).
- **Mudar o código de saída** para não-zero quando há documentos falhos (ver A5).
- **`README.md`, `CHANGELOG.md`, `.specs/ISSUES.md` e `.specs/STATE.md`** — fora da propriedade desta execução; pendência registrada para o gestor.

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| A1 — Onde a tolerância vale | **Exclusivamente** na gravação do documento (`InsertDocumentWithMeta`) | A opção C é "erro de protocolo no documento = documento falhou, seguir o próximo". Nenhum outro passo da varredura muda de tratamento | y |
| A2 — Erros de nível de varredura | **Abortam** a varredura e saem como `Erro durante a indexação: ...` | O dono pediu explicitamente que eles "continuem abortando — não engula tudo". Nota: o código anterior **ignorava** o `err` do walk (`return nil` em `cmd/mem/main.go:1725` e `:1944`); esta feature alinha o comportamento ao contrato | y |
| A3 — Documento que falhou e o `--prune` | Continua em `activeDocIDs` (é acrescentado **antes** da gravação) → **nunca** é podado | Um documento que falhou por erro de protocolo não pode ser apagado do índice; apagar seria pior que a falha | y |
| A4 — Documento que falhou e as contagens | Não conta como `indexados`; entra na contagem nova de `falhos` | O índice não tem o documento — anunciá-lo como indexado seria mentira | y |
| A5 — Código de saída | **Zero** quando a varredura termina | Padrão definido pelo dono: "o relatório é que carrega a contagem de falhas". Uma saída não-zero mudaria o contrato do comando para todos os consumidores (CI, scripts) | y |
| A6 — Falha na leitura do arquivo | Continua ignorada em silêncio (comportamento anterior, fora de escopo) | A opção C é sobre **gravação**. Registrar aqui para não vazar escopo; fica como pendência para uma feature própria | n |
| A7 — Onde vive a costura de teste | Interface **estreita** no package `main`, satisfeita por `*store.PostgresStore` | Proibido alterar `internal/store`; a `Store` interface real (`internal/store/store.go:158`) é ampla demais (busca, grafo, doctor) para um teste de varredura | y |
| A8 — Prova do caminho SQLite | Um **trigger** SQLite (`RAISE(ABORT)`) faz a gravação de um documento falhar | Prova o contrato do comando no driver real, sem store falso e sem rede | y |
| A9 — `internal/watcher` | Fora de escopo | O watcher tem indexador próprio (`internal/watcher/indexer.go`) e não é o `mem index` | y |
| A10 — ADR | **Nenhum ADR novo** | Não é mudança de schema, de driver nem de arquitetura: é o contrato de tolerância do comando `mem index` | y |
| A11 — Efeito colateral de um documento que falhou | Pode ficar **sem os chunks/arestas antigos**: a limpeza (`DeleteDocumentData`) roda antes da gravação que falhou | Comportamento anterior (a limpeza também precedia a gravação que abortava tudo); a opção C não muda essa ordem. Reindexar o documento devolve os chunks | y |

**Open questions**: none — as decisões estão resolvidas e registradas acima (A1 a A10).

## User Stories

### P1: Uma falha em um documento não derruba a indexação

**User Story**: Como dono de um vault indexado contra Postgres com pooler, quero que um documento que falhe ao ser gravado não mate a indexação dos outros, so that eu não fique com o vault pela metade sem saber.

**Why P1**: A ISSUE-032 matou a indexação de 276 documentos no primeiro erro de protocolo (`.specs/ISSUES.md:1269-1288`).

**Acceptance Criteria**:
1. WHEN a gravação de um documento falha THEN o `mem index` SHALL seguir para o próximo documento em vez de abortar a varredura.
2. WHEN a gravação de um documento falha THEN o comando SHALL emitir um aviso que nomeie o caminho do arquivo e o erro original.
3. WHEN um documento falhou THEN o comando SHALL NOT contá-lo como documento indexado.
4. WHEN um documento falhou THEN o comando SHALL mantê-lo em `activeDocIDs` para que o `--prune` não o remova.
5. WHEN a varredura termina THEN o comando SHALL sair com código de saída zero e o relatório SHALL carregar a contagem de falhas.

**Independent Test**: Injetar um store falso que falha a gravação do 2º documento de um diretório temporário e conferir que o 3º documento foi gravado, que a saída nomeia o arquivo que falhou e que o resumo contabiliza a falha.

---

### P2: A falha fica visível, não só contada

**User Story**: Como dono do vault, quero saber **quais** documentos ficaram de fora, so that eu reindexe exatamente esses arquivos em vez de desconfiar do índice inteiro.

**Why P2**: Contagem sem nome é um número que ninguém usa; o aviso por documento sem resumo é um log que passa despercebido.

**Acceptance Criteria**:
1. WHEN houve falhas de gravação THEN a linha de resumo `🎉 Concluído!` SHALL incluir a contagem de documentos que falharam.
2. WHEN houve falhas de gravação THEN o relatório SHALL nomear o caminho de cada documento que ficou de fora do índice.
3. WHEN não houve falhas THEN a linha de resumo SHALL manter o formato atual acrescido apenas do contador zerado.

**Independent Test**: Ler a saída do `mem index` de um vault com um documento falho e conferir que a linha de resumo quantifica a falha e que o caminho do arquivo aparece na saída.

---

### P3: O contrato é do comando, não do driver

**User Story**: Como pessoa que usa SQLite localmente, quero o mesmo comportamento do Postgres, so that o `mem index` nãotenha duas semânticas de falha.

**Why P3**: O comando é o mesmo; o engine é detalhe de implementação.

**Acceptance Criteria**:
1. WHEN a gravação de um documento falha no caminho SQLite THEN o `mem index` SHALL seguir para o próximo documento e reportar a falha como no caminho Postgres.
2. WHEN um erro de nível de varredura ocorre THEN o comando SHALL abortar a varredura e reportar o erro, sem converter o erro em aviso de documento.
3. WHEN a varredura é abortada por erro de nível de varredura THEN o comando SHALL NOT emitir um resumo de sucesso.

**Independent Test**: Indexar um vault SQLite temporário cujo trigger aborta a gravação de um documento e conferir que os documentos seguintes entraram no índice e que a saída reporta a falha.

## Edge Cases

- IF a **primeira** gravação da execução falhar THEN o comando SHALL continuar nos documentos seguintes e reportar a falha — a varredura não pode depender do primeiro arquivo.
- IF **todas** as gravações falharem THEN o comando SHALL reportar a contagem de falhas igual ao número de documentos processados e SHALL NOT abortar.
- IF o documento que falhou for **o último** do walk THEN o comando SHALL emitir o mesmo resumo, com a contagem de falhas.
- IF o `--prune` estiver ligado THEN o documento que falhou SHALL NOT ser podado, porque está em `activeDocIDs`.
- IF o erro de varredura for "diretório ilegível" THEN o comando SHALL abortar e reportar — não é tolerância por documento.
- IF o documento que falhou for reindexado numa execução seguinte com sucesso THEN o comando SHALL contá-lo como indexado e zerar a contagem de falhas.
- IF o erro vier do `InsertChunk` (o aviso que já existe) THEN o comportamento SHALL permanecer o de hoje: aviso, chunk sem embedding, documento indexado.

## Requirement Traceability

| Requirement ID | AC | Story | Arquivo | Fase | Status |
| -------------- | -- | ----- | ------- | ---- | ------ |
| TOL-01 | P1.1 | P1: Uma falha em um documento não derruba a indexação | cmd/mem/main.go | Execute | Verified |
| TOL-02 | P1.2 | P1: Uma falha em um documento não derruba a indexação | cmd/mem/main.go | Execute | Verified |
| TOL-03 | P1.3 | P1: Uma falha em um documento não derruba a indexação | cmd/mem/main.go | Execute | Verified |
| TOL-04 | P1.4 | P1: Uma falha em um documento não derruba a indexação | cmd/mem/main.go | Execute | Verified |
| TOL-05 | P1.5 | P1: Uma falha em um documento não derruba a indexação | cmd/mem/main.go | Execute | Verified |
| TOL-06 | P2.1 | P2: A falha fica visível, não só contada | cmd/mem/main.go | Execute | Verified |
| TOL-07 | P2.2 | P2: A falha fica visível, não só contada | cmd/mem/main.go | Execute | Verified |
| TOL-08 | P3.1 | P3: O contrato é do comando, não do driver | cmd/mem/main.go | Execute | Verified |
| TOL-09 | P3.2 | P3: O contrato é do comando, não do driver | cmd/mem/main.go | Execute | Verified |
| TOL-10 | P3.3 | P3: O contrato é do comando, não do driver | cmd/mem/index_tolerance_test.go | Execute | Verified |

**Coverage:** 10 total, 10 mapeados para tarefas, 0 não mapeados.

## Success Criteria

- [x] A gravação de um documento que falha não aborta a varredura, nos dois engines.
- [x] A falha aparece com nome de arquivo + erro, e a contagem entra na linha de resumo.
- [x] Documento que falhou não conta como indexado e não é podado.
- [x] Erro de nível de varredura aborta e é reportado.
- [x] Código de saída zero quando a varredura termina.
- [x] Teste hermético (sem rede, sem Postgres, sem binário) que falha contra o comportamento antigo.
- [x] `go build ./...`, `go vet ./...` e `go test -count=1 ./...` com exit 0.
