# Tasks: fix-index-tolerancia

**Spec**: [spec.md](./spec.md)
**Date**: 2026-10-29
**Issue**: ISSUE-032 (`.specs/ISSUES.md:1263`)

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Varredura Postgres (gravação do documento) | unit | Falha de gravação em 1 documento → a varredura segue; documento posterior gravado; aviso nomeia arquivo + erro; resumo quantifica; falho não conta como indexado; falho não é podado | cmd/mem/index_tolerance_test.go | go test -count=1 -run TestMemIndexPostgresTolerante ./cmd/mem/... |
| Varredura SQLite (gravação do documento) | unit | Mesmo contrato no driver real: trigger aborta a gravação de 1 documento → documentos seguintes no índice; aviso + resumo com a contagem | cmd/mem/index_tolerance_test.go | go test -count=1 -run TestMemIndexSQLiteTolerante ./cmd/mem/... |
| Costura de injeção (store falso) | unit | `*store.PostgresStore` satisfaz a interface estreita do package `main`; store falso falha só o documento escolhido | cmd/mem/index_tolerance_test.go | go test -count=1 -run TestMemIndexPostgresTolerante ./cmd/mem/... |
| Contrato do comando (documentação) | manual | `docs/CLI_GUIDE.md` descreve a tolerância por documento e o formato do resumo | docs/CLI_GUIDE.md | — |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após a costura (T1) | go build ./... |
| Quick | Após o teste hermético (T2) | go test -count=1 -run Tolerante ./cmd/mem/... |
| Quick | Após a tolerância no Postgres (T3) | go test -count=1 -run Tolerante ./cmd/mem/... |
| Quick | Após a tolerância no SQLite (T4) | go test -count=1 -run Tolerante ./cmd/mem/... |
| Full | Gate final (T6) | go test -count=1 ./... (com MY_MEMORY_PG_TEST=1 e MY_MEMORY_PG_URL preservada) |
| Build | Antes de declarar pronto | go build ./... e go vet ./... |
| Format | Antes de declarar pronto | gofmt -l na cópia LF dos arquivos .go tocados (o working tree local mistura CRLF) |
| Spec | Antes de entregar | python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/fix-index-tolerancia/spec.md e validate_tasks.py .specs/features/fix-index-tolerancia/tasks.md |
| State | Antes de entregar | python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-index-tolerancia --root D:\repository\my-memory |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2
T2 -> T3
T3 -> T4
T4 -> T5
T5 -> T6
```

| Fase | Tarefas | Objetivo |
| ---- | ------- | -------- |
| 1 | T1 | Costura estreita do store Postgres no package `main` (sem tocar `internal/store`) |
| 2 | T2 | Teste hermético que falha contra o comportamento antigo |
| 3 | T3 | Tolerância a falha por documento no caminho Postgres |
| 4 | T4 | Tolerância a falha por documento no caminho SQLite + erro de varredura que aborta |
| 5 | T5 | Documentação do contrato novo no CLI_GUIDE |
| 6 | T6 | Gate final, prova contra o comportamento antigo e relatório de evidências |

## Task Breakdown

### Phase 1: Costura de injeção

### T1: Interface estreita do store Postgres no package `main`
**What**: Declarar no package `main` a interface `postgresIndexStore` com **apenas** os métodos que a varredura usa (`ListDocumentTitles`, `ListDocumentIDTitles`, `GetDocumentIndexState`, `DeleteDocumentData`, `InsertDocumentWithMeta`, `InsertDocumentTags`, `InsertEdgeWithProps`, `InsertChunk`, `PruneDeletedDocuments`, `DeleteDocumentTags`) e trocar a assinatura de `runIndexPostgres` para recebê-la. `*store.PostgresStore` passa a ser usado pelo contrato; nenhum método de `internal/store` muda
**Where**: cmd/mem/main.go
**Depends on**: none
**Requirement**: TOL-01, TOL-08
**Tests**: cmd/mem/index_tolerance_test.go
**Gate**: go build ./...
**Done when**:
- [x] A interface `postgresIndexStore` existe no package `main` e `*store.PostgresStore` a satisfaz
- [x] `runIndexPostgres` recebe a interface; o comportamento **não** muda nesta fase (a varredura ainda aborta na primeira falha) — é o estado "vermelho" do teste
- [x] `internal/store/**` e `internal/db/**` intocados

### Phase 2: Teste hermético

### T2: Teste de tolerância por documento (store falso + trigger SQLite)
**What**: Escrever `cmd/mem/index_tolerance_test.go` com (a) um store falso que falha a gravação de um documento escolhido e (b) um caso SQLite com trigger que aborta a gravação. Asserções: documento posterior processado, aviso nomeia arquivo + erro, resumo quantifica e nomeia, documento falho não conta como indexado e não é podado
**Where**: cmd/mem/index_tolerance_test.go
**Depends on**: T1
**Requirement**: TOL-01, TOL-02, TOL-03, TOL-04, TOL-05, TOL-06, TOL-07, TOL-08
**Tests**: cmd/mem/index_tolerance_test.go
**Gate**: go test -count=1 -run Tolerante ./cmd/mem/...
**Done when**:
- [x] O teste roda **sem rede, sem Postgres e sem chamar o binário `mem`** (store falso + SQLite temporário)
- [x] Nada é escrito fora do diretório temporário do teste
- [x] O teste **falha** contra o comportamento antigo (rodada vermelha registrada na validation.md)

### Phase 3: Tolerância no caminho Postgres

### T3: Falha de gravação vira aviso + contagem (Postgres)
**What**: Em `runIndexPostgres`, a falha de `InsertDocumentWithMeta` passa a gerar aviso (caminho + erro), incrementar a contagem de falhas, registrar o documento que ficou de fora e **continuar** a varredura; o documento falho não incrementa `indexedCount` e continua em `activeDocIDs`; o resumo final ganha o contador de falhas e o bloco que nomeia os documentos de fora
**Where**: cmd/mem/main.go
**Depends on**: T2
**Requirement**: TOL-01, TOL-02, TOL-03, TOL-04, TOL-05, TOL-06, TOL-07
**Tests**: cmd/mem/index_tolerance_test.go
**Gate**: go test -count=1 -run Tolerante ./cmd/mem/...
**Done when**:
- [x] O aviso por documento nomeia o caminho e o erro original
- [x] A varredura segue para o documento seguinte
- [x] O documento falho não conta como indexado e não é podado
- [x] O resumo final quantifica e nomeia as falhas

### Phase 4: Tolerância no caminho SQLite + erro de varredura

### T4: Mesmo contrato no SQLite e erro de nível de varredura que aborta
**What**: Em `runIndexSQLite`, a falha de `db.InsertDocumentWithMeta` passa a ter o mesmo tratamento (aviso, contagem, continuidade, exclusão da contagem de indexados, `activeDocIDs` preservado). Em **ambos** os caminhos, o erro recebido pelo callback do `filepath.WalkDir` deixa de ser engolido e aborta a varredura com `Erro durante a indexação: ...`
**Where**: cmd/mem/main.go
**Depends on**: T3
**Requirement**: TOL-08, TOL-09, TOL-10
**Tests**: cmd/mem/index_tolerance_test.go
**Gate**: go test -count=1 -run Tolerante ./cmd/mem/...
**Done when**:
- [x] O caminho SQLite tem o mesmo contrato do Postgres (provado com o trigger)
- [x] Erro de nível de varredura aborta e é reportado, sem resumo de sucesso
- [x] A leitura do arquivo que falha continua ignorada em silêncio (comportamento anterior, fora de escopo)

### Phase 5: Documentação

### T5: Contrato da tolerância no CLI_GUIDE
**What**: Documentar em `docs/CLI_GUIDE.md` (§ `mem index`) que a falha de gravação de um documento não aborta a varredura, que o documento que falhou sai do índice mas não é podado, e que o resumo carrega a contagem de falhas
**Where**: docs/CLI_GUIDE.md
**Depends on**: T4
**Requirement**: TOL-02, TOL-05, TOL-06, TOL-07
**Tests**: docs/CLI_GUIDE.md
**Gate**: go test -count=1 ./cmd/mem/...
**Done when**:
- [x] A seção `mem index` descreve o comportamento novo e o formato do resumo

### Phase 6: Gate final e evidências

### T6: Gate completo, prova contra o comportamento antigo e relatório
**What**: Rodar o gate completo (`go test -count=1 ./...` com `MY_MEMORY_PG_TEST=1` e `MY_MEMORY_PG_URL` preservada, `go build ./...`, `go vet ./...`, `gofmt -l` na cópia LF dos arquivos tocados), registrar a prova de que o teste falha contra o comportamento antigo e escrever `validation.md` com evidência `arquivo:linha` por requisito
**Where**: .specs/features/fix-index-tolerancia/validation.md
**Depends on**: T5
**Requirement**: TOL-01, TOL-02, TOL-03, TOL-04, TOL-05, TOL-06, TOL-07, TOL-08, TOL-09, TOL-10
**Tests**: .specs/features/fix-index-tolerancia/validation.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-index-tolerancia --root D:\repository\my-memory
**Done when**:
- [x] `go build ./...`, `go vet ./...` e `go test -count=1 ./...` com exit 0
- [x] `gofmt -l` limpo nos arquivos .go tocados (medido em cópia com LF)
- [x] Comportamento antigo → teste falhando com o sintoma exato → restaurado
- [x] `validation.md` com o bloco canônico `## Validation: PASS` + `**Result**: PASS` e evidência `arquivo:linha` por critério
