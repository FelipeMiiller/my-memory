# Feature: ai-tool-auto-wiring

## Problem Statement
Para que assistentes de codificação de IA (Claude Desktop, Cursor IDE, VS Code / GitHub Copilot, Windsurf) se conectem ao servidor Model Context Protocol (MCP) do **My-Memory**, o desenvolvedor atualmente enfrenta um alto atrito de configuração manual:
1. **Localização Fragmentada no SO**: Cada ferramenta armazena suas configurações em caminhos ocultos distintos dependendo do sistema operacional (ex: `%APPDATA%\Claude\claude_desktop_config.json` no Windows, `~/Library/Application Support/Claude/...` no macOS, `~/.config/Claude/...` no Linux, `.cursor/mcp.json` no workspace ou `~/.cursor/...` globalmente, `.vscode/mcp.json`, etc.).
2. **Resolução Complexa de Caminhos Absolutos**: O arquivo JSON requer o caminho absoluto do executável (`mem` ou `bin/mem.exe`) e o caminho absoluto do banco de dados SQLite (`.memory/memory.db`), exigindo que o desenvolvedor descubra e digite manualmente caminhos absolutos com barras escapadas no Windows.
3. **Risco de Corrupção de JSON Existente**: Usuários que já configuraram outros servidores MCP correm o risco de sobrescrever acidentalmente ou quebrar a sintaxe JSON ao colar blocos de configuração.

Inspirado na funcionalidade `codegraph install` do CodeGraph ([docs/REFERENCES.md#L71](../../../docs/REFERENCES.md#L71)), esta feature estabelece o **Auto-Wiring e Instalação Zero-Touch de Ferramentas de IA (`mem install` / `mem setup`)**, que auto-detecta ferramentas instaladas na máquina do desenvolvedor e no workspace, mescla a configuração do servidor MCP `my-memory` de forma atômica e não-destrutiva, e reduz o tempo de configuração inicial a um único comando no terminal.

---

## Goals
- [x] **G1**: Criar o pacote de domínio `internal/autowire/` com detecção de clientes MCP suportados (Claude Desktop, Cursor IDE, VS Code Copilot, Windsurf) em Windows, macOS e Linux.
- [x] **G2**: Implementar o motor de mesclagem não-destrutiva de JSON (`MergeMCPServerConfig`), preservando servidores e opções pré-existentes, com criação de backup atômico (`.bak`).
- [x] **G3**: Implementar o subcomando CLI `mem install` (com alias `mem setup`) suportando `--target [claude|cursor|vscode|windsurf|all]`, `--dry-run`, `--global`, `--workspace`, `--db`, `--repo` e `--force`.
- [x] **G4**: Registrar a decisão arquitetural no documento `docs/adr/026-auto-wiring-e-instalacao-zero-touch-de-ferramentas-de-ia.md` e atualizar a documentação operacional da CLI.

---

## Out of Scope
- Baixar ou instalar os binários dos clientes de IA (o comando não instala o Cursor ou Claude Desktop; ele apenas auto-detecta os diretórios de configuração e injeta o bloco MCP).
- Gerenciamento de chaves de API secretas de modelos ou serviços de terceiros (o MCP do My-Memory opera localmente).

---

## Assumptions & Open Questions

| Assumption / Decision | Chosen Default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| Alvo padrão (`--target`) | `all` (detecta todas as ferramentas disponíveis) | Proporciona experiência zero-touch em uma única execução | y |
| Escopo padrão | Ambos: workspace (`.cursor/`, `.vscode/`) e global (Claude Desktop, Windsurf) | Garante que a IA funcione tanto no projeto atual quanto globalmente | y |
| Resolução do binário | `os.Executable()` com conversão para caminho absoluto limpo | Garante que o cliente MCP sempre encontre o executável correto | y |
| Resolução do banco | Auto-scoping do vault (`.memory/config.yaml` $\to$ `.memory/memory.db`) | Conecta imediatamente à memória do projeto ativo | y |
| Preservação de JSON | Merge preservando chaves de outros servidores | Evita quebrar configurações pré-existentes de outros MCPs | y |

---

## User Stories

### P1: Auto-Detecção e Injeção Não-Destrutiva ⭐ MVP
**User Story**: Como desenvolvedor usando Claude Desktop ou Cursor, quero executar `mem install` para que o My-Memory adicione seu bloco de configuração MCP automaticamente sem que eu precise procurar pastas ocultas ou editar JSONs à mão.
**Critérios de Aceite**:
1. O comando localiza o caminho de configuração correspondente ao SO em execução.
2. Se o arquivo JSON já existir, faz merge inserindo a chave `mcpServers["my-memory"]` sem remover outros servidores.
3. Cria backup `.bak` do arquivo antes de aplicar qualquer alteração.
4. Se o arquivo não existir, cria o diretório e o arquivo JSON formatado.

### P2: Suporte a Simulação (--dry-run)
**User Story**: Como desenvolvedor cauteloso, quero executar `mem install --dry-run` para visualizar quais arquivos seriam modificados e o payload exato antes da escrita em disco.
**Critérios de Aceite**:
1. Mostra o caminho de cada arquivo detectado e o trecho JSON que seria inserido.
2. Nenhum arquivo em disco é criado ou alterado no modo `--dry-run`.

### P3: Controle Fino por Ferramenta e Escopo
**User Story**: Como usuário de múltiplas ferramentas, quero executar `mem install --target claude` ou `mem install --workspace` para restringir onde a configuração deve ser aplicada.
**Critérios de Aceite**:
1. Flag `--target` aceita `claude`, `cursor`, `vscode`, `windsurf` ou `all`.
2. Flags `--workspace` e `--global` filtram o escopo da instalação.
