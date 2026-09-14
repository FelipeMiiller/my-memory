# Tasks: progressive-context-loading

- [ ] **T1**: Extensão do Schema e Migração Idempotente (`internal/db/` e `internal/store/`)
  - [ ] Adicionar colunas `abstract TEXT` e `category TEXT DEFAULT 'resource'` na tabela `documents` do SQLite (`internal/db/schema.go`).
  - [ ] Adicionar colunas `abstract TEXT` e `category TEXT DEFAULT 'resource'` no PostgreSQL (`internal/store/postgres.go`).
  - [ ] Atualizar struct `Document` e `SearchResult` em `internal/db/store.go` e `internal/store/store.go`.
  - [ ] Escrever testes unitários validando migrações em bancos existentes sem perda de dados.

- [ ] **T2**: Extração Heurística de L0 (Micro-Abstract) e Taxonomia no Parser (`internal/parser/`)
  - [ ] Adicionar campos `Category`, `Summary` e `Abstract` no modelo `Frontmatter` (`internal/parser/frontmatter.go`).
  - [ ] Implementar `ExtractMicroAbstract(body string, maxLen int) string` para extrair o primeiro parágrafo descritivo sem marcações markdown.
  - [ ] Atualizar pipeline de indexação em `internal/db/store.go` e `internal/store/postgres.go` para persistir `abstract` e `category`.
  - [ ] Escrever testes cobrindo extração via frontmatter explícito e via heurística de primeiro parágrafo.

- [ ] **T3**: Motores de Busca com Níveis L0/L1/L2 e Filtro por Categoria (`internal/db/` e `internal/store/`)
  - [ ] Atualizar queries de `SearchFTS`, `SearchVector` e `SearchHybrid` para aceitar filtro opcional `category`.
  - [ ] Implementar projeção de campos: em modo `l0`, omitir blocos massivos de chunks e retornar apenas `Abstract`, metadados e score.
  - [ ] Adicionar suporte a decaimento temporal ponderado preservando o nível selecionado.
  - [ ] Testes automatizados em `internal/db/hybrid_test.go` e `internal/store/rrf_test.go`.

- [ ] **T4**: Subcomando CLI mem search com Flags de Densidade (`cmd/mem/`)
  - [ ] Adicionar flags `--level` (`l0`, `l1`, `l2`) e `--category` (`resource`, `memory`, `skill`) em `cmd/mem/main.go`.
  - [ ] Implementar layout compacto no terminal para L0 (exibição em 1-2 linhas por resultado com badges ANSI).
  - [ ] Atualizar documentação e ajuda (`mem search --help`).
  - [ ] Escrever testes de integração em `cmd/mem/search_defaults_test.go`.

- [ ] **T5**: MCP memory_search com detail_level, Taxonomia e ADR-025 (`internal/mcp/`)
  - [ ] Atualizar schema de `memory_search` em `internal/mcp/tools.go` com `detail_level` e `category`.
  - [ ] Ajustar formatação Markdown no handler MCP (`internal/mcp/handlers.go`) para renderizar tabelas sintéticas em L0.
  - [ ] Conectar os handlers do servidor MCP com suporte aos novos parâmetros.
  - [ ] Escrever testes de integração MCP em `internal/mcp/handlers_test.go`.
  - [ ] Criar `docs/adr/025-progressive-context-loading-e-taxonomia-de-memoria.md` e atualizar índices de ADRs.
