# Tasks: compile-not-retrieve-and-bilateral-mcp

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Compiler Core | unit | SafeResolvePath, WriteAtomicNote, YAML frontmatter | internal/compiler/*_test.go | go test -v ./internal/compiler/... |
| Compiler Actions | unit | AppendSection, CompileTopicNote | internal/compiler/*_test.go | go test -v ./internal/compiler/... |
| Compiler Sync | unit | Surgical indexing trigger after file write | internal/compiler/*_test.go | go test -v ./internal/compiler/... |
| MCP Writer Tools | integration | memory_write_note, memory_append_section, memory_compile_note | internal/mcp/*_test.go | go test -v ./internal/mcp/... |
| CLI Commands | unit | mem note create, mem note append, mem compile | cmd/mem/*_test.go | go test -v ./cmd/mem/... |
| Documentation / ADR | none | Documentation, ADR-018, README updates | docs/adr/* | go test -v ./cmd/mem/... ./internal/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After compiler core changes | go test -v ./internal/compiler/... |
| MCP | After MCP tools implementation | go test -v ./internal/mcp/... |
| CLI | After CLI note commands implementation | go test -v ./cmd/mem/... |
| Full | After all integrations | go test -count=1 -v ./cmd/mem/... ./internal/... |
| Build | Before completing feature | go test -v ./cmd/mem/... ./internal/... && python .agents/skills/tlc-spec-driven/scripts/validate_spec.py compile-not-retrieve-and-bilateral-mcp |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T3 -> T4 -> T5 -> T6
```

### Phase 1: Núcleo de Compilação, MCP e Interface CLI

## Task Breakdown

### T1: Núcleo de Resolução Segura de Caminhos e Formatação de Notas Atômicas
**What**: Implementar SafeResolvePath e WriteAtomicNote em internal/compiler/note.go
**Where**: internal/compiler/note.go
**Depends on**: none
**Requirement**: CNR-01, CNR-02, CNR-03
**Tests**: internal/compiler/note_test.go
**Gate**: go test -v ./internal/compiler/...
**Done when**:
- [x] Implementar SafeResolvePath garantindo isolamento dentro da raiz do vault contra path traversal
- [x] Implementar WriteAtomicNote formatando frontmatter YAML limpo (title, type, tags, aliases)
- [x] Suportar injeção de arestas epistêmicas tipadas no corpo do Markdown ([[rel:relation:Target]])
- [x] Validar recusa de sobrescrita caso o arquivo já exista e overwrite seja falso
- [x] Adicionar testes unitários em internal/compiler/note_test.go

### T2: Apensamento Inteligente de Seções e Compilação de Busca
**What**: Implementar AppendSection e CompileTopicNote em internal/compiler/compile.go
**Where**: internal/compiler/compile.go
**Depends on**: T1
**Requirement**: CNR-04, CNR-05
**Tests**: internal/compiler/compile_test.go
**Gate**: go test -v ./internal/compiler/...
**Done when**:
- [x] Implementar AppendSection localizando cabeçalhos Markdown ou criando nova seção
- [x] Implementar CompileTopicNote gerando notas estruturadas com seção de síntese e backlinks para fontes
- [x] Preservar integridade de formatação Markdown sem quebra de linhas existentes
- [x] Adicionar testes unitários em internal/compiler/compile_test.go

### T3: Sincronização Cirúrgica Automática de Notas no Banco de Dados
**What**: Implementar rotina de sincronização cirúrgica em internal/compiler/sync.go
**Where**: internal/compiler/sync.go
**Depends on**: T2
**Requirement**: CNR-05
**Tests**: internal/compiler/sync_test.go
**Gate**: go test -v ./internal/compiler/...
**Done when**:
- [x] Integrar chamada de reindexação cirúrgica utilizando rotinas de watcher.IndexSingleFile
- [x] Tratar gracioso em caso de indisponibilidade momentânea do embedder Ollama
- [x] Atualizar tabela de documentos, chunks, FTS e arestas de grafo instantaneamente
- [x] Adicionar testes unitários em internal/compiler/sync_test.go

### T4: Ferramentas de Escrita Bilateral no Servidor MCP
**What**: Declarar schemas e implementar handlers MCP para escrita e compilação em internal/mcp/writer_handlers.go
**Where**: internal/mcp/writer_handlers.go
**Depends on**: T3
**Requirement**: CNR-06
**Tests**: internal/mcp/writer_handlers_test.go
**Gate**: go test -v ./internal/mcp/...
**Done when**:
- [x] Declarar schemas para memory_write_note, memory_append_section e memory_compile_note em internal/mcp/tools.go
- [x] Implementar handlers com validação de parâmetros e integração ao internal/compiler
- [x] Registrar ferramentas no Server em internal/mcp/server.go
- [x] Adicionar testes de integração JSON-RPC em internal/mcp/writer_handlers_test.go

### T5: Comandos CLI mem note e mem compile
**What**: Adicionar subcomandos mem note e mem compile em cmd/mem/note.go e main.go
**Where**: cmd/mem/note.go
**Depends on**: T4
**Requirement**: CNR-07
**Tests**: cmd/mem/note_test.go
**Gate**: go test -v ./cmd/mem/...
**Done when**:
- [ ] Implementar subcomandos note create e note append com flags completas
- [ ] Implementar subcomando compile buscando tópicos e gerando nota atômica compilada
- [ ] Integrar subcomandos ao switch principal em cmd/mem/main.go e atualizar printHelp
- [ ] Adicionar testes unitários em cmd/mem/note_test.go

### T6: ADR-018, Validação Final e Documentação
**What**: Registrar decisão de arquitetura na ADR-018 e atualizar documentação do projeto
**Where**: docs/adr/018-padrao-compile-not-retrieve-e-escrita-bilateral-mcp.md
**Depends on**: T5
**Requirement**: CNR-01, CNR-02, CNR-03, CNR-04, CNR-05, CNR-06, CNR-07
**Tests**: none
**Gate**: go test -v ./cmd/mem/... ./internal/...
**Done when**:
- [ ] Redigir ADR-018 no formato MADR detalhando o padrão Compile-not-Retrieve e escrita bilateral
- [ ] Atualizar docs/adr/README.md adicionando a ADR-018
- [ ] Atualizar README.md, docs/CLI_GUIDE.md e docs/REPOSITORY_BRAIN.md
- [ ] Atualizar .specs/STATE.md com o novo status
- [ ] Validar suíte completa de testes e gates do TLC
