# Tasks: staleness-detection-and-banners

- [x] **T1**: Motor de Detecção de Desatualização (`internal/staleness/`)
  - [x] Definir modelos de dados: `StalenessReport`, `StalenessOptions`, `DocumentMeta`.
  - [x] Implementar `CheckStaleness(ctx context.Context, db *sql.DB, rootDir string, cfg *config.Config) (*StalenessReport, error)`.
  - [x] Implementar varredura de filesystem com `filepath.WalkDir` e filtragem via `cfg.ShouldIndex`.
  - [x] Implementar detecção de arquivos modificados (`mtime > updated_at + 1s`), novos (ausentes no banco) e deletados (no banco mas ausentes no disco).
  - [x] Implementar cache em memória leve com TTL de 3 segundos para amortecer I/O em rajadas.
  - [x] Implementar formatadores de banner: `FormatMarkdownBanner` e `FormatCLIBanner`.
  - [x] Escrever testes unitários em `internal/staleness/detector_test.go`.

- [x] **T2**: Integração com Camadas de Persistência SQLite e PostgreSQL
  - [x] Adicionar função `GetDocumentsMetadata` em `internal/db/store.go` para extração em lote de `path`, `updated_at`, `content_hash`.
  - [x] Adicionar método à interface `Store` em `internal/store/store.go`.
  - [x] Implementar `GetDocumentsMetadata` em `internal/store/postgres.go`.
  - [x] Escrever testes de persistência em `internal/db/store_test.go` e `internal/store/postgres_test.go`.

- [x] **T3**: Subcomando CLI `mem status` e Warnings na CLI (`cmd/mem/`)
  - [x] Criar `cmd/mem/status.go` com flags `--json`, `--db`, `--postgres`, `--repo`.
  - [x] Renderizar saída amigável no terminal indicando status `Fresh` ou `Stale`, lista de arquivos e recomendações.
  - [x] Integrar alerta de staleness nos comandos de consulta (`search`, `path`, `inspect`, `impact`).
  - [x] Registrar `status` no switch de comandos e em `printHelp()` no `cmd/mem/main.go`.
  - [x] Escrever testes em `cmd/mem/status_test.go`.

- [x] **T4**: Injeção de Staleness Banner no Servidor MCP (`internal/mcp/`)
  - [x] Integrar detector de staleness ao servidor MCP em `internal/mcp/server.go`.
  - [x] Adicionar injeção de `FormatMarkdownBanner` no topo das respostas de ferramentas MCP (`memory_search`, `memory_find_path`, `memory_inspect_node`, `memory_get_impact`).
  - [x] Escrever testes unitários em `internal/mcp/staleness_handlers_test.go`.

- [x] **T5**: Arquitetura (ADR-028) e Documentação Operacional
  - [x] Criar `docs/adr/028-staleness-banners-e-deteccao-de-desatualizacao.md` no padrão MADR.
  - [x] Atualizar catálogo em `docs/adr/README.md` e `docs/README.md`.
  - [x] Atualizar `docs/CLI_GUIDE.md` com o comando `mem status`.
  - [x] Atualizar `docs/AGENT_INTEGRATION_GUIDE.md` com explicações sobre o Staleness Banner.
  - [x] Atualizar estado do projeto em `.specs/STATE.md`.
