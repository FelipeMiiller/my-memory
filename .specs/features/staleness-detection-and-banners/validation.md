# Validation Plan: staleness-detection-and-banners

## 1. Testes Automatizados Unitários e de Integração

### Motor de Detecção de Staleness (`internal/staleness/`)
- [ ] `TestCheckStaleness_AllFresh`: Valida que quando o banco tem todos os arquivos com timestamps sincronizados, retorna `IsStale: false` e listas vazias.
- [ ] `TestCheckStaleness_ModifiedFile`: Valida detecção de arquivo alterado no disco (`ModTime > UpdatedAt + 1s`).
- [ ] `TestCheckStaleness_NewFile`: Valida identificação de arquivos novos criados no disco ausentes na tabela `documents`.
- [ ] `TestCheckStaleness_DeletedFile`: Valida identificação de arquivos registrados no banco que foram apagados do disco.
- [ ] `TestCheckStaleness_IgnoreRules`: Valida que arquivos em `.git/`, `.obsidian/`, `node_modules/` ou fora das regras de `ShouldIndex` são estritamente ignorados.
- [ ] `TestCheckStaleness_CacheTTL`: Valida que chamadas dentro do TTL de 3 segundos retornam dados em cache sem refazer I/O em disco.
- [ ] `TestFormatBanners`: Valida formatação correta dos textos de alerta para Markdown e CLI.

### Camadas de Persistência SQLite e Postgres (`internal/db/` & `internal/store/`)
- [ ] `TestDB_GetDocumentsMetadata`: Valida retorno do mapa de metadados (`path`, `updated_at`, `content_hash`) no SQLite.
- [ ] `TestPostgres_GetDocumentsMetadata`: Valida recuperação eficiente de metadados no PostgreSQL.

### Subcomando CLI `mem status` (`cmd/mem/`)
- [ ] `TestStatusCLI_FreshVault`: Valida saída amigável quando o vault está 100% sincronizado.
- [ ] `TestStatusCLI_StaleVault`: Valida exibição dos arquivos modificados/novos/deletados quando há divergências.
- [ ] `TestStatusCLI_JSONOutput`: Valida parsing estruturado de `mem status --json`.

### Injeção de Banners no Servidor MCP (`internal/mcp/`)
- [ ] `TestMCP_StalenessBanner_InjectedWhenStale`: Valida que `memory_search` e `memory_find_path` incluem o bloco `> ⚠️ **Staleness Warning**` quando o detector indicar desatualização.
- [ ] `TestMCP_StalenessBanner_OmittedWhenFresh`: Valida que nenhum banner é injetado quando o índice está atualizado.

---

## 2. Critérios de Conclusão e Regressão
- [ ] 100% dos testes novos passando.
- [ ] Todos os 15 pacotes Go continuam passando com 100% de sucesso (`go test -count=1 ./...`).
- [ ] Recompilação de `bin/mem.exe` bem-sucedida.
- [ ] Teste real no Windows demonstrando alerta de desatualização ao alterar uma nota.
