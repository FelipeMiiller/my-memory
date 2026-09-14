# Tasks: ai-tool-auto-wiring

- [x] **T1**: Motor de Detecção de Clientes e Estruturas de Configuração (`internal/autowire/`)
  - [x] Definir modelos de dados: `ClientTarget`, `ClientType`, `ConfigScope` (`ScopeGlobal`, `ScopeWorkspace`) e `ServerConfig`.
  - [x] Implementar resolvedor de caminhos por sistema operacional (Windows, macOS, Linux) para:
    - Claude Desktop (`claude_desktop_config.json`)
    - Cursor Workspace (`.cursor/mcp.json`) e Cursor Global (`~/.cursor/mcp.json`)
    - VS Code / Copilot Workspace (`.vscode/mcp.json`)
    - Windsurf Global (`~/.codeium/windsurf/mcp_config.json`)
  - [x] Implementar verificação de presença/instalação de cada ferramenta.
  - [x] Escrever testes unitários em `internal/autowire/detector_test.go`.

- [x] **T2**: Motor de Fusão JSON Não-Destrutiva e Backup Atômico (`internal/autowire/`)
  - [x] Implementar `InjectMCPServer(target ClientTarget, serverName string, serverCfg ServerConfig, dryRun bool) (*InstallReport, error)`.
  - [x] Suportar formatos de arquivo com `mcpServers` (Claude Desktop, Cursor, VS Code, Windsurf) preservando indentação e chaves paralelas.
  - [x] Implementar criação de backup `.bak` atômico antes de sobrescrever arquivos existentes.
  - [x] Escrever testes unitários em `internal/autowire/installer_test.go` cobrindo criação nova, mesclagem e simulação `--dry-run`.

- [x] **T3**: Subcomando CLI `mem install` e Alias `mem setup` (`cmd/mem/`)
  - [x] Implementar `runInstallCLI` e `runInstallCommand` com flags `--target`, `--dry-run`, `--global`, `--workspace`, `--db`, `--repo`, `--force`.
  - [x] Auto-detecção de caminho absoluto do binário (`os.Executable()`) e resolução do vault ativo (`.memory/memory.db`).
  - [x] Renderizar saída amigável no terminal com status por cliente (detectado, instalado, ignorado, erro).
  - [x] Registrar alias `mem setup` apontando para o mesmo comando.
  - [x] Escrever testes unitários e de integração em `cmd/mem/install_test.go`.

- [x] **T4**: Decisão de Arquitetura (ADR-026) e Atualização da Documentação
  - [x] Criar `docs/adr/026-auto-wiring-e-instalacao-zero-touch-de-ferramentas-de-ia.md` no padrão MADR.
  - [x] Atualizar índices de ADRs em `docs/adr/README.md` e `docs/README.md`.
  - [x] Atualizar `docs/CLI_GUIDE.md` e `docs/AGENT_INTEGRATION_GUIDE.md` com os novos comandos `mem install` / `mem setup`.
