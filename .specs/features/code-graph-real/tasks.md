# Tasks: Code Graph Real

- **Spec**: [spec.md](spec.md) (28 requisitos, CG-01..CG-28)
- **Decisão**: [ADR-061](../../docs/adr/061-dois-conceitos-de-memoria-vault-central-e-code-graph.md)
- **Issue**: [ISSUE-027](../ISSUES.md) (12 gaps: C1..C12)
- **Fases**: ligação → motor → superfície (ADR-061 §Implementation Plan)

## Test Coverage Matrix

| Camada | Testes | Cobre |
|---|---|---|
| Config / escopo de código | `internal/config/config_test.go`, `internal/codeast/pipeline_test.go` | CG-01, CG-02, CG-04 |
| CLI `mem index` + code stage | `cmd/mem/codeindex_test.go` | CG-03, CG-05, CG-09 |
| Schema `code_*` (SQLite + Postgres) | `internal/db/code_migration_test.go` | CG-06, CG-07, CG-14, CG-15, CG-17 |
| Store de código (filtro por repo) | `internal/store/code_test.go` | CG-07 |
| MCP de código | `internal/mcp/code_handlers_test.go`, `internal/mcp/server_test.go` | CG-08, CG-16, CG-21, CG-22 |
| Parser AST Go | `internal/codeast/go_parser_test.go`, `internal/codeast/parser_test.go` | CG-10, CG-11, CG-12, CG-13 |
| Busca híbrida com código | `internal/store/rrf_test.go`, `internal/mcp/handlers_test.go` | CG-19, CG-20 |
| Drift de código | `internal/drift/drift_test.go` | CG-25, CG-26 |
| Vault central (lição) | `cmd/mem/lesson_test.go` | CG-27, CG-28 |
| Viewer | `viewer/src/routes/code.test.tsx` | CG-23, CG-24 |

## Gate Check Commands

```bash
# Build e vet (o gate de código é o Go; a viewer tem gate próprio)
go build ./... && go vet ./...

# Suíte completa — o gate só fecha com 25/25 pacotes verdes
go test -count=1 ./...

# Benchmark obrigatório nos pacotes tocados (o gate exige benchmark onde falta)
go test '-bench=.' '-benchmem' '-run=^$' ./internal/codeast ./internal/db ./internal/mcp ./internal/store ./internal/drift

# Formatação: prova por cópia normalizada em LF (o checkout local é CRLF)
gofmt -l <arquivos tocados>

# Gate de frontend, quando a viewer for tocada (orçamento: 160 kB gzip)
cd viewer && npm run typecheck && npm run test && npm run build

# Gates estruturais da spec
python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/code-graph-real --strict
python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py .specs/features/code-graph-real
```

## Execution Plan

### Phase 1 — Ligação (o code graph passa a existir e ser alcançável)

```
T1 -> T2
T1 -> T3
T1 -> T6
T3 -> T4
T3 -> T5
```

### T1: Escopo de código separado do escopo de notas

**Where**: `internal/config/config.go`
**Depends on**: none
**Tests**: `internal/config/config_test.go` — `TestCodeScopeIndependenteDoIncludeGlobal`; `internal/codeast/pipeline_test.go` — `TestListCodeFilesUsaCodeScopeNaoShouldIndex`; `cmd/mem/index_code_scope_test.go` — `TestMemIndexNaoIndexaCodigoComoNota`
**Gate**: `go test -count=1 ./internal/config/... ./internal/codeast/...`
- **Do**: adicionar `CodeInclude`/`CodeExclude` ao config, com default igual às extensões de código do backend ativo; `ListCodeFiles` passa a filtrar pelo escopo de código e nunca por `cfg.ShouldIndex`; o `include` global de Markdown não pode selecionar arquivo de código para o pipeline de notas. Cobre CG-01, CG-02, CG-04.
**Done when**:
- [x] Seção `code.include`/`code.exclude` no config, independente de `include`/`exclude` (ADR-061 §4)
- [x] Default do escopo de código = extensões do backend ativo (guardado por teste de paridade com a tabela de linguagens)
- [x] `ListCodeFiles` filtra pelo escopo de código e nunca por `cfg.ShouldIndex`
- [x] Arquivo de código nunca vira nota no pipeline Markdown, mesmo quando o `include` global casa com ele (CG-04)

### T2: O `mem index` executa o code stage

**Where**: `cmd/mem/main.go`
**Depends on**: T1
**Tests**: `cmd/mem/index_code_scope_test.go` — `TestMemIndexExecutaCodeStage`; `TestMemIndexSemEscopoDeCodigoNaoRodaCodeStage`
**Gate**: `go test -count=1 ./cmd/mem/...`
- **Do**: `mem index` passa a rodar o code pipeline como estágio (opt-in por flag, default ligado quando há escopo de código configurado), no mesmo repositório resolvido pelo pipeline Markdown; o behaviour atual (só Markdown) permanece quando o escopo de código está vazio. Cobre CG-03.
**Done when**:
- [x] `mem index` roda o code stage no mesmo repositório resolvido pelo pipeline Markdown (persistindo `code_files`/`code_symbols`/`code_edges`)
- [x] Escopo de código configurado liga o estágio por default; a flag `--code-ast[=true|false]` liga ou desliga explicitamente
- [x] Escopo de código vazio: o `mem index` continua sendo byte-a-byte o pipeline Markdown de antes
- [x] Falha do code stage é logada e não derruba o `mem index` (mesma política do embedder ausente)

### T3: O code pipeline escreve no engine configurado

**Where**: `cmd/mem/codeindex.go`
**Depends on**: T1
**Tests**: `cmd/mem/codeindex_test.go` — `TestCodeIndexEscreveNoEngineConfigurado`
**Gate**: `go test -count=1 ./cmd/mem/...`
- **Do**: quando o engine configurado é Postgres, o pipeline grava `code_files`/`code_symbols`/`code_edges` no Postgres, em vez de abrir e descartar o store e escrever num SQLite local; o caminho `--storage=sqlite` continua existindo e é o usado em teste/CI. Cobre CG-05.
- **Done when**:
  - [x] O code stage grava `code_files`/`code_symbols`/`code_edges` no engine configurado (Postgres quando Postgres, SQLite quando SQLite)
  - [x] As tabelas `code_*` existem no Postgres (DDL no schema do store) e o mesmo caminho de escrita roda nos dois engines
  - [x] Engine Postgres sem URL é erro explícito — nunca um fallback silencioso para um SQLite local
  - [x] O caminho `--storage=sqlite` continua existindo e é o usado em teste/CI

### T4: `repository` nas tabelas `code_*` e filtro por repositório

**Where**: `internal/db/code_migration.go`
**Depends on**: T3
**Tests**: `internal/db/code_migration_test.go` — `TestCodeTablesTemRepository`; `internal/store/code_test.go` — `TestFiltroPorRepository`
**Gate**: `go test -count=1 ./internal/db/... ./internal/store/...`
- **Do**: migração idempotente adicionando `repository` em `code_files`, `code_symbols` e `code_edges` (SQLite e Postgres), preenchida com o `repo_id` resolvido; as queries de código passam a filtrar por repositório, como já acontece com o grafo de notas. Cobre CG-06, CG-07.

### T5: Os handlers MCP de código são injetados no servidor

**Where**: `internal/mcp/server.go`
**Depends on**: T3
**Tests**: `internal/mcp/code_handlers_test.go` — `TestServerInjetaHandlersDeCodigo`
**Gate**: `go test -count=1 ./internal/mcp/...`
- **Do**: `SetCodeSearchHandler`/`SetCodeNeighborsHandler` deixam de ser chamados só em teste: o servidor MCP, tanto no CLI quanto no daemon, injeta os executores de código resolvidos a partir do engine ativo; `memory_code_search` e `memory_code_neighbors` passam a devolver resultados em vez da mensagem "indisponível". Cobre CG-08.

### T6: As flags do CLI de código deixam de ser descartadas

**Where**: `cmd/mem/codeindex.go`
**Depends on**: T1
**Tests**: `cmd/mem/codeindex_test.go` — `TestFlagsDeCodeIndexNaoSaoDescartadas`
**Gate**: `go test -count=1 ./cmd/mem/...`
- **Do**: `--repo` passa a valer de verdade; `--ast-hash` e `--no-embed` honram o contrato declarado no help ou saem do help; `--json` sai de "sem efeito nesta versão" ou passa a emitir JSON. Nenhuma flag pode continuar sendo aceitada e descartada. Cobre CG-09.

### Phase 2 — Motor (arestas estruturais reais, com call-site e proveniência)

```
T7 -> T8
T8 -> T9
T9 -> T10
T10 -> T11
T11 -> T12
```

### T7: Backend AST em Go puro

**Where**: `internal/codeast/go_parser.go`
**Depends on**: T1
**Tests**: `internal/codeast/go_parser_test.go` — `TestGoParserExtraiSimbolosComPosicao`
**Gate**: `go test -count=1 ./internal/codeast/...`
- **Do**: novo backend que usa `go/parser`/`go/ast` da stdlib para extrair símbolos com posição exata (kind, nome, qualified name, assinatura, linha/coluna, doc comment) — Go puro, sem CGO, sem build tag. Cobre CG-10.

### T8: `newBackendParser()` deixa de ser código morto

**Where**: `internal/codeast/treesitter_disabled.go`
**Depends on**: T7
**Tests**: `internal/codeast/parser_test.go` — `TestBackendPadraoNaoEStub`
**Gate**: `go test -count=1 ./internal/codeast/...`
- **Do**: o build default passa a devolver o backend Go em `newBackendParser()`, e não `ErrTreesitterDisabled`; o caminho com a tag `treesitter` continua isolado e, quando houver toolchain C, tem precedência. O `MockParser` deixa de ser o caminho de produção. Cobre CG-13.

### T9: Arestas `calls` com call-site

**Where**: `internal/codeast/go_parser.go`
**Depends on**: T8
**Tests**: `internal/codeast/go_parser_test.go` — `TestGoParserExtraiCallsComCallSite`
**Gate**: `go test -count=1 ./internal/codeast/... ./internal/db/...`
- **Do**: o backend extrai chamadas do AST e grava arestas `calls` carregando o **ponto de chamada** (arquivo e linha da expressão de chamada), e não apenas a posição da definição; a resolução por escopo de pacote gera `EXTRACTED`, e a resolução apenas por nome gera `INFERRED`. Cobre CG-11.

### T10: Arestas `imports` e `inherits`

**Where**: `internal/codeast/go_parser.go`
**Depends on**: T9
**Tests**: `internal/codeast/go_parser_test.go` — `TestGoParserExtraiImportsEInherits`
**Gate**: `go test -count=1 ./internal/codeast/...`
- **Do**: o backend passa a emitir `imports` (arquivo importador → pacote importado) e `inherits`/`uses_type` (embedding de struct e satisfação de interface em Go); os `kind` disponíveis passam a ser `calls`, `imports`, `inherits`, `uses_type` e `references`, e não só `references`. Cobre CG-12.

### T11: `epistemic_status` e aresta ambígua

**Where**: `internal/db/code_migration.go`
**Depends on**: T10
**Tests**: `internal/db/code_migration_test.go` — `TestCodeEdgesTemEpistemicStatus`; `internal/codeast/go_parser_test.go` — `TestArestaAmbiguaVaiParaUncertain`
**Gate**: `go test -count=1 ./internal/db/... ./internal/codeast/...`
- **Do**: `code_edges` ganha `epistemic_status` (`EXTRACTED`/`INFERRED`), alinhado a `graph_edges` (ADR-011), por migração idempotente; aresta cujo destino não resolve unicamente vai para `code_edges_uncertain` com confiança e razão, e não aparece como resolvida. Cobre CG-14, CG-15.

### T12: A resposta MCP de código traz proveniência, confiança e call-site

**Where**: `internal/mcp/code_handlers.go`
**Depends on**: T11
**Tests**: `internal/mcp/code_handlers_test.go` — `TestRespostaDeCodeTrazProvenienciaECallSite`
**Gate**: `go test -count=1 ./internal/mcp/...`
- **Do**: `CodeNeighborResult` passa a expor `epistemic_status`, `confidence` e o call-site da aresta, além do `file`/`line` do símbolo destino; os campos novos são aditivos e não quebram o contrato das tools existentes. Cobre CG-16.

### Phase 3 — Superfície (o code graph aparece onde o agente e o humano olham)

```
T13 -> T15
T13 -> T16
```

### T13: A ponte `documents` ↔ `code_symbols`

**Where**: `internal/db/code_migration.go`
**Depends on**: T12
**Tests**: `internal/db/code_migration_test.go` — `TestDocumentCodeSymbolEdge`
**Gate**: `go test -count=1 ./internal/db/...`
- **Do**: quando um documento referencia um símbolo existente em `code_symbols` (por `qualified_name`), a indexação cria uma aresta tipada do documento para o símbolo; o grafo de notas e o de código deixam de ser disjuntos e o "Repository Brain" passa a existir como grafo único. Cobre CG-17, CG-18.

### T14: Fan-in RRF real de `code_symbols`

**Where**: `internal/store/rrf.go`
**Depends on**: T12
**Tests**: `internal/store/rrf_test.go` — `TestFanInDeCodeEntraNoRRF`; `internal/mcp/handlers_test.go` — `TestIncludeCodeSemPlaceHolder`
**Gate**: `go test -count=1 ./internal/store/... ./internal/mcp/...`
- **Do**: `memory_search` com `include_code=true` deixa de emitir o aviso "CA-12 placeholder" e passa a fundir `code_symbols` no RRF como uma fonte real; com `include_code` ausente ou falso, o ranking atual permanece byte-a-byte igual (teste de regressão explícito). Cobre CG-19, CG-20.

### T15: Hub threshold e orçamento de tokens na travessia

**Where**: `internal/mcp/code_handlers.go`
**Depends on**: T13
**Tests**: `internal/mcp/code_handlers_test.go` — `TestTravessiaAplicaHubThreshold`; `TestTravessiaRespeitaOrcamentoDeTokens`
**Gate**: `go test -count=1 ./internal/mcp/...`
- **Do**: a travessia de código aplica um hub threshold por percentil de grau (para que poucos nós centrais não dominem o subgrafo), respeita um orçamento de tokens na resposta, ordena por distância em saltos a partir da semente e, ao truncar, emite a dica de estreitar a query. Cobre CG-21, CG-22.

### T16: Rota de código na viewer

**Where**: `viewer/src/routes/code.tsx`
**Depends on**: T13
**Tests**: `viewer/src/routes/code.test.tsx`
**Gate**: `cd viewer && npm run typecheck && npm run test && npm run build`
- **Do**: a viewer ganha uma rota de código que lista os símbolos do repositório e, ao selecionar um, mostra os vizinhos estruturais com o tipo da aresta e o call-site; sem dependência de CDN e dentro do orçamento de 160 kB gzip do CI. Cobre CG-23, CG-24.

### T17: Drift de mudança estrutural

**Where**: `internal/drift/drift.go`
**Depends on**: T12
**Tests**: `internal/drift/drift_test.go` — `TestDriftDetectaMudancaEstrutural`
**Gate**: `go test -count=1 ./internal/drift/...`
- **Do**: o detector de drift passa a ler `ast_hash` de `code_files` e a reportar mudança estrutural de código, atribuída aos documentos que referenciam os símbolos afetados; o drift continua funcionando quando o code stage não rodou na mesma sessão. Cobre CG-25, CG-26.

### T18: Escrita de lição no vault central

**Where**: `cmd/mem/lesson.go`
**Depends on**: T5
**Tests**: `cmd/mem/lesson_test.go` — `TestMemLessonGravaNotaNoVaultCentral`
**Gate**: `go test -count=1 ./cmd/mem/...`
- **Do**: novo caminho de escrita que registra uma lição no vault central como nota com `category: memory`, `summary` (para recuperação L0) e timestamp, indexando-a na hora; a nota fica como Markdown puro no vault e não exige nada do vault de repositório. Cobre CG-27, CG-28.

## Task Breakdown

| # | Tarefa | Fase | Onde | Depende de | Requisitos |
|---|---|---|---|---|---|
| T1 | Escopo de código separado | 1 | `internal/config/config.go` | — | CG-01, CG-02, CG-04 |
| T2 | `mem index` executa o code stage | 1 | `cmd/mem/main.go` | T1 | CG-03 |
| T3 | Escrita no engine configurado | 1 | `cmd/mem/codeindex.go` | T1 | CG-05 |
| T4 | `repository` em `code_*` | 1 | `internal/db/code_migration.go` | T3 | CG-06, CG-07 |
| T5 | Handlers MCP injetados | 1 | `internal/mcp/server.go` | T3 | CG-08 |
| T6 | Flags do CLI de código | 1 | `cmd/mem/codeindex.go` | T1 | CG-09 |
| T7 | Backend AST Go puro | 2 | `internal/codeast/go_parser.go` | T1 | CG-10 |
| T8 | `newBackendParser()` vivo | 2 | `internal/codeast/treesitter_disabled.go` | T7 | CG-13 |
| T9 | `calls` com call-site | 2 | `internal/codeast/go_parser.go` | T8 | CG-11 |
| T10 | `imports` e `inherits` | 2 | `internal/codeast/go_parser.go` | T9 | CG-12 |
| T11 | `epistemic_status` + uncertain | 2 | `internal/db/code_migration.go` | T10 | CG-14, CG-15 |
| T12 | Resposta MCP com proveniência | 2 | `internal/mcp/code_handlers.go` | T11 | CG-16 |
| T13 | Ponte `documents` ↔ `code_symbols` | 3 | `internal/db/code_migration.go` | T12 | CG-17, CG-18 |
| T14 | Fan-in RRF real | 3 | `internal/store/rrf.go` | T12 | CG-19, CG-20 |
| T15 | Hub threshold + orçamento | 3 | `internal/mcp/code_handlers.go` | T13 | CG-21, CG-22 |
| T16 | Rota de código na viewer | 3 | `viewer/src/routes/code.tsx` | T13 | CG-23, CG-24 |
| T17 | Drift de mudança estrutural | 3 | `internal/drift/drift.go` | T12 | CG-25, CG-26 |
| T18 | Escrita de lição no vault central | 3 | `cmd/mem/lesson.go` | T5 | CG-27, CG-28 |

**Ordem de execução**: as fases rodam em sequência; dentro de cada fase, as tarefas
que não têm dependência entre si podem ser executadas em paralelo por workers
disjuntos. Cada tarefa é um commit atômico Conventional Commit, com o gate
indicado verde **antes** do commit, e `tasks.md` atualizado no mesmo commit.
