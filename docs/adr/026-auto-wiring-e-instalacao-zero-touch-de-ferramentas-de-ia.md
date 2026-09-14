# ADR-026: Auto-Wiring e Instalação Zero-Touch de Ferramentas de IA (`mem install` / `mem setup`)

- **Date**: 2026-09-14
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: autowire, mcp, zero-touch, devx, claude-desktop, cursor, vscode, windsurf, cli

## Context and Problem Statement

O ecossistema de agentes e assistentes de inteligência artificial adotou amplamente o **Model Context Protocol (MCP)** como padrão de comunicação entre modelos e fontes de contexto/ferramentas externas. No entanto, a integração do servidor MCP do **My-Memory** (`mem mcp`) com clientes populares (Claude Desktop, Cursor IDE, VS Code com GitHub Copilot, Windsurf) exigia até então a edição manual de arquivos JSON de configuração situados em diretórios com caminhos heterogêneos entre sistemas operacionais (Windows `%APPDATA%`, macOS `~/Library/Application Support`, Linux `~/.config`) ou em pastas ocultas de workspace (`.cursor/mcp.json`, `.vscode/mcp.json`).

Esse processo manual introduzia atrito significativo na adoção (DevX), com alto risco de corrupção acidental de sintaxe JSON, sobrescrita de configurações de outros servidores MCP previamente instalados pelo desenvolvedor e inconsistências nos argumentos passados (como caminhos relativos ao banco de dados SQLite ou identificador de repositório).

Faz-se necessária uma solução integrada à CLI que ofereça **auto-wiring zero-touch**: detecção automática de ferramentas de IA instaladas na máquina do desenvolvedor e no diretório de trabalho, com injeção não-destrutiva e idempotente da configuração do My-Memory.

## Decision Drivers

- **Zero-Touch Onboarding**: Executar um comando único (`mem install` ou `mem setup`) para identificar e configurar automaticamente todos os clientes MCP suportados.
- **Injeção Não-Destrutiva e Segura**: Preservar rigorosamente quaisquer outros servidores MCP já configurados no arquivo JSON de destino, gravando com indentação limpa e gerando cópias de backup (`.bak`) antes de qualquer mutação.
- **Transparência e Auditoria (`--dry-run`)**: Permitir que usuários e equipes de infraestrutura visualizem exatamente as ações planejadas e os fragmentos JSON a serem injetados antes de autorizar qualquer gravação em disco.
- **Suporte Multi-OS Nativo**: Resolver caminhos canônicos e variáveis de ambiente em Windows, macOS e Linux sem scripts de casca externos (*zero shell dependencies*).
- **Flexibilidade de Escopo**: Capacidade de direcionar tanto configurações globais a nível de usuário (ex: Claude Desktop, Cursor Global) quanto configurações locais restritas ao repositório atual (Workspace para VS Code e Cursor).
- **Resolução Inteligente de Binário e Contexto**: Detectar automaticamente o caminho absoluto do executável `mem` em execução e inferir o banco SQLite (`.memory/memory.db`) e repositório do vault ativo via `.memory/config.yaml`.

## Decision Outcome

Decidiu-se pela criação de um subsistema desacoplado em `internal/autowire/` integrado diretamente à CLI `mem` através dos comandos `mem install` e `mem setup`.

### 1. Modelagem de Clientes e Detecção Multi-OS (`internal/autowire/`)

Foi introduzida a estrutura `ClientSpec` para representar declarativamente clientes MCP suportados:
- **`ID` / `Name`**: Identificador canônico (`claude-desktop`, `cursor`, `vscode`, `windsurf`) e nome de exibição.
- **`Scope`**: Escopo de atuação (`ScopeGlobal`, `ScopeWorkspace` ou ambos).
- **`PlatformPaths`**: Função de mapeamento de caminhos conforme o sistema operacional (`runtime.GOOS`), resolvendo `%APPDATA%`, `%USERPROFILE%`, `HOME` e caminhos relativos de workspace.
- **`Detector`**: Mecanismo que verifica a presença do cliente examinando diretórios de instalação de binários, pastas de dados de aplicativo ou arquivos de configuração pré-existentes.

| Cliente MCP | Escopos Suportados | Caminhos Globais / Workspace |
| :--- | :--- | :--- |
| **Claude Desktop** | Global | Windows: `%APPDATA%/Claude/claude_desktop_config.json`<br>macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`<br>Linux: `~/.config/Claude/claude_desktop_config.json` |
| **Cursor IDE** | Global / Workspace | Global: `%APPDATA%/Cursor/...` ou `~/.cursor/...`<br>Workspace: `<workspace>/.cursor/mcp.json` |
| **VS Code / Copilot** | Workspace | Workspace: `<workspace>/.vscode/mcp.json` |
| **Windsurf** | Global / Workspace | Global: `~/.codeium/windsurf/mcp_config.json`<br>Workspace: `<workspace>/.windsurf/mcp.json` |

### 2. Injetor Não-Destrutivo de Configuração (`internal/autowire/installer.go`)

O processo de injeção segue um protocolo seguro de 5 etapas:
1. **Verificação de Diretório**: Criação de diretórios-pai se não existirem (com permissão `0755`).
2. **Leitura e Preservação**:
   - Se o arquivo JSON já existir, seu conteúdo é decodificado em uma árvore de objetos genérica (`map[string]interface{}`).
   - Outras chaves de topo e outros servidores sob a propriedade `mcpServers` são integralmente preservados.
3. **Idempotência**: Se `mcpServers["my-memory"]` já contiver exatamente a mesma especificação de comando e argumentos, nenhuma escrita em disco é realizada, reportando status de "já configurado".
4. **Backup Automático**: Antes de sobrescrever um arquivo existente, gera-se uma cópia idêntica em `<caminho>.bak`.
5. **Serialização e Gravação**: O JSON resultante é formatado com duas quebras de espaço (`json.MarshalIndent`) e gravado com permissão `0644`.

### 3. Interface de Linha de Comando (`cmd/mem/install.go`)

Disponibilizado através de `mem install` (com alias de conveniência `mem setup`):
```bash
# Auto-detectar todas as ferramentas instaladas e configurar automaticamente
mem install

# Simular a instalação sem alterar o disco
mem install --dry-run

# Configurar um cliente específico (ex: cursor ou claude-desktop)
mem install --target cursor

# Forçar configuração apenas em escopo de workspace ou global
mem install --workspace
mem install --global

# Sobrescrever arquivos mesmo se já existirem backups
mem install --force
```

O comando realiza auto-scoping de vault: se um arquivo `.memory/config.yaml` for detectado no diretório atual, o banco SQLite e o nome do repositório correspondentes são passados como argumentos `--db` e `--repo` na entrada do servidor MCP, garantindo que o agente acesse imediatamente a memória correta daquele projeto.

## Consequences

### Positive

- **Onboarding em Menos de 5 Segundos**: Usuários e agentes de IA configuram assistentes com um único comando sem navegar manualmente em pastas de sistema ocultas.
- **Risco Zero de Perda de Dados**: A preservação estrita de chaves existentes no JSON somada à criação de arquivos `.bak` elimina o risco de quebrar outros servidores MCP do desenvolvedor.
- **Consistência Operacional**: Garante que o binário correto e os parâmetros de vault (`--repo`, `--db`) sejam sempre injetados de maneira padronizada.
- **Excelente Experiência de Agente**: Agentes autônomos que operam na máquina do usuário podem invocar `mem install --workspace` programaticamente para disponibilizar ferramentas de memória a editores abertos.

### Negative / Trade-offs

- **Diversidade de Clientes MCP**: Novos editores ou clientes que surjam no ecossistema exigirão a inclusão de suas definições em `internal/autowire/client.go`. (Minimizado pelo design modular e extensível do `ClientSpec`).
- **Necessidade de Reinicialização de Alguns Clientes**: Certos clientes (como Claude Desktop) leem as configurações somente na inicialização, exigindo que o usuário feche e reabra o aplicativo para carregar as novas ferramentas.
