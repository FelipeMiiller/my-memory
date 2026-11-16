# ADR-026: Auto-Wiring e Instalação Zero-Touch de Ferramentas de IA (`mem install`)

- **Date**: 2026-09-14
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: autowire, mcp, zero-touch, devx, claude-desktop, cursor, vscode, windsurf, cli
- **Emenda**: 2026-11-16 — a spec `fix-autowiring-cli-contract` (ISSUE-048) corrigiu esta descrição para que ela reflita o código que existe.

## Context and Problem Statement

O ecossistema de agentes e assistentes de inteligência artificial adotou amplamente o **Model Context Protocol (MCP)** como padrão de comunicação entre modelos e fontes de contexto/ferramentas externas. No entanto, a integração do servidor MCP do **My-Memory** (`mem mcp`) com clientes populares (Claude Desktop, Cursor IDE, VS Code com GitHub Copilot, Windsurf) exigia até então a edição manual de arquivos JSON de configuração situados em diretórios com caminhos heterogêneos entre sistemas operacionais (Windows `%APPDATA%`, macOS `~/Library/Application Support`, Linux `~/.config`) ou em pastas ocultas de workspace (`.cursor/mcp.json`, `.vscode/mcp.json`).

Esse processo manual introduzia atrito significativo na adoção (DevX), com alto risco de corrupção acidental de sintaxe JSON, sobrescrita de configurações de outros servidores MCP previamente instalados pelo desenvolvedor e inconsistências nos argumentos passados (como caminhos relativos ao banco de dados SQLite ou identificador de repositório).

Faz-se necessária uma solução integrada à CLI que ofereça **auto-wiring zero-touch**: detecção automática de ferramentas de IA instaladas na máquina do desenvolvedor e no diretório de trabalho, com injeção não-destrutiva e idempotente da configuração do My-Memory.

## Decision Drivers

- **Zero-Touch Onboarding**: Executar um comando único (`mem install`) para identificar e configurar automaticamente os clientes MCP suportados.
- **Injeção Não-Destrutiva e Segura**: Preservar rigorosamente quaisquer outros servidores MCP já configurados no arquivo JSON de destino, gravando com indentação canônica e gerando cópias de backup (`.bak`) **atômicas** antes de qualquer mutação.
- **Transparência e Auditoria (`--dry-run`)**: Permitir que usuários e equipes de infraestrutura visualizem exatamente as ações planejadas e os fragmentos JSON a serem injetados antes de autorizar qualquer gravação em disco.
- **Suporte Multi-OS Nativo**: Resolver caminhos canônicos e variáveis de ambiente em Windows, macOS e Linux sem scripts de casca externos (*zero shell dependencies*).
- **Flexibilidade de Escopo**: Capacidade de direcionar tanto configurações globais a nível de usuário (ex: Claude Desktop, Cursor Global) quanto configurações locais restritas ao repositório atual (Workspace para VS Code e Cursor).
- **Resolução Inteligente de Binário e Contexto**: Detectar automaticamente o caminho absoluto do executável `mem` em execução e inferir o banco SQLite (`.memory/memory.db`) e repositório do vault ativo via `.memory/config.yaml`.
- **Escopo do Cliente Instalado**: Configurar apenas as ferramentas que existem na máquina, pulando as que não estão instaladas — um config para um cliente ausente é ruído, não conveniência.

## Decision Outcome

Decidiu-se pela criação de um subsistema desacoplado em `internal/autowire/` integrado diretamente à CLI `mem` através do comando **`mem install`**.

### 0. O dono do verbo `setup`

O comando do auto-wiring é **`mem install`** (`cmd/mem/main.go:836` roteia `case "install":` para `cmd/mem/install.go:21`). O verbo **`setup` pertence ao assistente interativo de cofre central** (ADR-033): `cmd/mem/main.go:794` roteia `case "setup":` para `runSetupCLI`, um comando vivo e documentado por outra ADR. O alias `mem setup` que o auto-wiring teve entre `db02542` e `8685ed8` foi **repurposado** e não volta: a correção é de **documentação** (spec `fix-autowiring-cli-contract`, ISSUE-048).

### 1. Modelagem de Clientes e Detecção Multi-OS (`internal/autowire/`)

O registro de clientes **não é declarativo**: o código real é formado pelas **quatro constantes de tipo** e por blocos explícitos de resolução de caminho.

- **`ClientType`** (`internal/autowire/client.go:6`): `claude` (Claude Desktop), `cursor`, `vscode` e `windsurf` — os quatro clientes suportados.
- **`ClientTarget`** (`internal/autowire/client.go:30`): tipo, nome de exibição, escopo, caminho do arquivo de configuração, existência do arquivo e **presença de instalação** do cliente na máquina.
- **Blocos de detecção** (`internal/autowire/detector.go:71`): um bloco por cliente e escopo, cada um resolvendo o caminho canônico do sistema operacional.
- **Normalização de SO** (`internal/autowire/detector.go:161`): `joinTargetOSPath` monta o caminho com o separador do SO a partir de `%APPDATA%`, `HOME` e do diretório de workspace.
- **Sondagem do alvo** (`internal/autowire/detector.go:184`): `makeClientTarget` normaliza o caminho e interroga o disco através do `SystemEnv` injetável.
- **Filtros** (`internal/autowire/detector.go:222` e `internal/autowire/detector.go:226`): `matchTarget` filtra por cliente-alvo (aceitando `claude-desktop` como sinônimo de `claude`) e `matchScope` filtra por escopo.
- **Presença de instalação**: derivada de um **marcador de máquina** por tipo de cliente (diretório de dados do Claude, diretório global do Cursor, diretório de dados do VS Code, diretório global do Windsurf), consultado pelo mesmo `SystemEnv` injetável. O diretório de workspace **não** é sinal de instalação — usá-lo tornaria o comando um no-op em todo repositório virgem.

| Cliente MCP | Escopos Suportados | Caminhos Globais / Workspace |
| :--- | :--- | :--- |
| **Claude Desktop** | Global | Windows: `%APPDATA%/Claude/claude_desktop_config.json`<br>macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`<br>Linux: `~/.config/Claude/claude_desktop_config.json` |
| **Cursor IDE** | Global / Workspace | Global: `~/.cursor/mcp.json`<br>Workspace: `<workspace>/.cursor/mcp.json` |
| **VS Code / Copilot** | Workspace | Workspace: `<workspace>/.vscode/mcp.json` |
| **Windsurf** | **Global** | Global: `~/.codeium/windsurf/mcp_config.json` |

O Windsurf é suportado **somente no escopo global** — não existe alvo de workspace para ele.

### 2. Injetor Não-Destrutivo de Configuração (`internal/autowire/installer.go`)

O processo de injeção (`internal/autowire/installer.go:12`) segue um protocolo seguro de 5 etapas:

1. **Verificação de Diretório**: Criação de diretórios-pai se não existirem (com permissão `0755`).
2. **Leitura e Preservação**:
   - Se o arquivo JSON já existir, seu conteúdo é decodificado em uma árvore de objetos genérica (`map[string]interface{}`).
   - Outras chaves de topo e outros servidores sob a propriedade `mcpServers` são integralmente preservados.
3. **Idempotência**: Se `mcpServers["my-memory"]` já contiver exatamente a mesma especificação de comando e argumentos, nenhuma escrita em disco é realizada, reportando status de "já configurado".
4. **Backup Atômico**: Antes de sobrescrever um arquivo existente, o conteúdo original é gravado num **temporário no mesmo diretório** e publicado no destino `<caminho>.bak` com `os.Rename` (`internal/autowire/installer.go:104`) — um `.bak` truncado é exatamente o que o backup existe para evitar.
5. **Serialização e Gravação**: O JSON resultante é formatado com **dois espaços** e com as **chaves de topo em ordem alfabética** (`internal/autowire/installer.go:75`, `json.MarshalIndent`) e gravado com permissão `0644`. A **indentação e a ordem originais do arquivo do usuário não são preservadas**: o contrato é a formatação canônica.

### 3. Interface de Linha de Comando (`cmd/mem/install.go`)

Disponibilizado através de **`mem install`** (`cmd/mem/install.go:21`):

```bash
# Auto-detectar as ferramentas instaladas e configurar automaticamente
mem install

# Simular a instalação sem alterar o disco
mem install --dry-run

# Configurar um cliente específico (ex: cursor ou claude-desktop)
mem install --target cursor
mem install --target claude-desktop

# Forçar configuração apenas em escopo de workspace ou global
mem install --workspace
mem install --global

# Sobrescrever arquivos mesmo se já existirem backups
mem install --force
```

- **`--target`**: aceita `all`, `claude`, `claude-desktop` (sinônimo de `claude`), `cursor`, `vscode` e `windsurf`. Um valor **fora** do conjunto é **erro** (exit ≠ 0), com a lista dos valores aceitos na mensagem.
- **`--workspace` / `--global`**: filtram o escopo e são **mutuamente exclusivos** — informá-los juntos é erro (exit ≠ 0).
- **Escopo do cliente**: um cliente **não instalado** é pulado — nenhum arquivo e nenhum diretório são criados, e a saída nomeia o cliente pulado e o motivo.
- **Precedência de storage (ADR-040)**: o payload injetado declara o **engine vencedor** (`Engine: postgres` ou `Engine: sqlite`). Com Postgres, o payload leva `--storage postgres` e **não** leva `--db`; com SQLite, leva `--db <caminho>`. A **URL do Postgres nunca vai para o arquivo do editor** — o config do workspace vive dentro do repositório, versionado, e uma URL pode carregar senha.
- **Auto-Scoping de Vault**: se um arquivo `.memory/config.yaml` for detectado no diretório atual, o banco SQLite e o nome do repositório correspondentes são passados como argumentos `--db` e `--repo` na entrada do servidor MCP.

## Consequences

### Positive

- **Onboarding em Menos de 5 Segundos**: Usuários e agentes de IA configuram assistentes com um único comando sem navegar manualmente em pastas de sistema ocultas.
- **Risco Zero de Perda de Dados**: A preservação estrita de chaves existentes no JSON somada à criação de arquivos `.bak` **atômicos** elimina o risco de quebrar outros servidores MCP do desenvolvedor.
- **Consistência Operacional**: Garante que o binário correto e os parâmetros de vault (`--repo`, `--db` ou `--storage`) sejam sempre injetados de maneira padronizada, com o mesmo engine que a CLI declarou.
- **Excelente Experiência de Agente**: Agentes autônomos que operam na máquina do usuário podem invocar `mem install --workspace` programaticamente para disponibilizar ferramentas de memória a editores abertos.

### Negative / Trade-offs

- **Diversidade de Clientes MCP**: Novos editores ou clientes que surjam no ecossistema exigirão a inclusão de suas definições em `internal/autowire/client.go` e de um bloco de detecção em `internal/autowire/detector.go`.
- **Necessidade de Reinicialização de Alguns Clientes**: Certos clientes (como Claude Desktop) leem as configurações somente na inicialização, exigindo que o usuário feche e reabra o aplicativo para carregar as novas ferramentas.
