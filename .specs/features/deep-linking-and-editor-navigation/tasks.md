# Tasks: deep-linking-and-editor-navigation

## Test Coverage Matrix

| Task | Requirement | Unit / Integration Test | Target File |
| :--- | :--- | :--- | :--- |
| T1 | DL-01, DL-02 | `TestGenerateLinks_*`, `TestLauncher_*` | `internal/deeplink/deeplink_test.go` |
| T2 | DL-02 | `TestEditorConfig_DefaultsAndCustom` | `internal/config/config_test.go` |
| T3 | DL-03 | `TestRunOpenCommand_*` | `cmd/mem/open_test.go` |
| T4 | DL-04, DL-05 | `TestInspectLinks_*`, `TestMCPOpenNode_*` | `internal/mcp/open_handlers_test.go` |
| T5 | DL-01..05 | ADR-030, validação e verificação | `.specs/features/deep-linking-and-editor-navigation/validation.md` |

---

## Gate Check Commands

```bash
# T1 Gate
go test -v ./internal/deeplink/...

# T2 Gate
go test -v ./internal/config/...

# T3 Gate
go test -v ./cmd/mem/... -run TestOpen

# T4 Gate
go test -v ./internal/mcp/... ./cmd/mem/...

# T5 Gate
go test -count=1 ./...
```

---

## Execution Plan

```mermaid
graph TD
    T1 -> T2
    T2 -> T3
    T3 -> T4
    T4 -> T5
```

---

## Task Breakdown

### Phase 1: Core Engine & Config

### T1: Implementar Motor de Deep Links RFC 3986 e Launcher Cross-Platform
Where: internal/deeplink/deeplink.go
Depends on: none
Tests: internal/deeplink/deeplink_test.go
Gate: go test -v ./internal/deeplink/...
Details:
- Implementar estrutura `DeepLinks` (`Obsidian`, `VSCode`, `File`).
- Implementar `GenerateLinks(repoRoot, vaultName, filePath string, line int) DeepLinks`.
- Implementar abstração `Launcher` com suporte a `os/exec` e mock para testes determinísticos.
- Implementar `Open(targetPathOrURI string, app string, line int, repoRoot string, vaultName string, launcher Launcher) (string, error)`.

### T2: Integrar Configuração Declarativa de Editor no Vault
Where: internal/config/config.go
Depends on: T1
Tests: internal/config/config_test.go
Gate: go test -v ./internal/config/...
Details:
- Adicionar struct `EditorConfig` com campos `DefaultApp` e `ObsidianVault`.
- Integrar em `Config` e `DefaultConfig()`.
- Implementar método auxiliar para resolução do nome canônico do vault.

### Phase 2: CLI & Enriquecimento

### T3: Implementar Subcomando CLI mem open
Where: cmd/mem/open.go
Depends on: T2
Tests: cmd/mem/open_test.go
Gate: go test -v ./cmd/mem/... -run TestOpen
Details:
- Criar comando `mem open <nota>` suportando flags `--app`, `--line`, `--json`, `--dry-run`.
- Implementar resolução de nó por caminho no disco ou consulta no banco (SQLite/PostgreSQL).
- Integrar no roteador `cmd/mem/main.go` e ajuda CLI.

### T4: Enriquecer Saídas de Inspeção, Busca e Ferramenta MCP
Where: internal/mcp/open_handlers.go
Depends on: T3
Tests: internal/mcp/open_handlers_test.go
Gate: go test -v ./internal/mcp/...
Details:
- Adicionar bloco `🔗 Links Rápidos` na renderização de `mem inspect` e flag `--links` no `mem search`.
- Adicionar `ToolMemoryOpenNode` e handler em `internal/mcp/open_handlers.go`.
- Incluir metadados `links` no `memory_inspect_node` e `memory_search`.

### Phase 3: Governança & Validação

### T5: Documentar ADR-030 e Concluir Verificação do Sistema
Where: docs/adr/030-deep-linking-e-navegacao-de-editores.md
Depends on: T4
Tests: cmd/mem/open_test.go
Gate: go test -count=1 ./...
Details:
- Criar `docs/adr/030-deep-linking-e-navegacao-de-editores.md` no padrão MADR.
- Atualizar `docs/adr/README.md`, `docs/README.md`, `docs/CLI_GUIDE.md` e `docs/AGENT_INTEGRATION_GUIDE.md`.
- Atualizar `.specs/STATE.md` e gerar relatório `validation.md`.
