# Tasks: ai-tool-auto-wiring

> **Emenda (2026-11-16)**: estes checkboxes foram corrigidos pela spec **`fix-autowiring-cli-contract`** (ISSUE-048), que fecha os 10 gaps da verificação independente. O que mudou aqui é o **contrato publicado**: o alias `mem setup` **não existe** (o verbo pertence ao assistente de cofre central, ADR-033) e a **indentação original do config não é preservada** (a formatação é a canônica: 2 espaços, chaves de topo em ordem alfabética). O comportamento do código e as evidências `file:line` estão na spec de correção.

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
  - [x] Suportar formatos de arquivo com `mcpServers` (Claude Desktop, Cursor, VS Code, Windsurf) preservando as chaves paralelas e os servidores pré-existentes — a **indentação e a ordem originais não são preservadas**: o config escrito sai na formatação canônica (2 espaços, chaves de topo em ordem alfabética).
  - [x] Implementar criação de backup `.bak` atômico antes de sobrescrever arquivos existentes.
  - [x] Escrever testes unitários em `internal/autowire/installer_test.go` cobrindo criação nova, mesclagem e simulação `--dry-run`.

- [x] **T3**: Subcomando CLI `mem install` (`cmd/mem/`)
  - [x] Implementar `runInstallCLI` e `runInstallCommand` com flags `--target`, `--dry-run`, `--global`, `--workspace`, `--db`, `--repo`, `--force`.
  - [x] Auto-detecção de caminho absoluto do binário (`os.Executable()`) e resolução do vault ativo (`.memory/memory.db`).
  - [x] Renderizar saída amigável no terminal com status por cliente (detectado, instalado, ignorado, erro).
  - [x] `mem setup` **não** é alias deste comando: o verbo `setup` pertence ao assistente interativo de cofre central (ADR-033) e o alias que existiu entre `db02542` e `8685ed8` foi repurposado — a correção é de documentação (spec `fix-autowiring-cli-contract`, ISSUE-048).
  - [x] Escrever testes unitários e de integração em `cmd/mem/install_test.go`.

- [x] **T4**: Decisão de Arquitetura (ADR-026) e Atualização da Documentação
  - [x] Criar `docs/adr/026-auto-wiring-e-instalacao-zero-touch-de-ferramentas-de-ia.md` no padrão MADR.
  - [x] Atualizar índices de ADRs em `docs/adr/README.md` e `docs/README.md`.
  - [x] Atualizar `docs/CLI_GUIDE.md` e `docs/AGENT_INTEGRATION_GUIDE.md` com o novo comando `mem install`.
