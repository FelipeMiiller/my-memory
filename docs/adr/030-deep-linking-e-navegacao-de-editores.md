# ADR-030: Deep Linking e Integração de Navegação com Editores (`mem open` e URIs `obsidian://` / `vscode://`)

- **Date**: 2026-09-14
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: deeplink, obsidian, vscode, editor-navigation, cli, mcp, uri-scheme, ux

## Context and Problem Statement

Desenvolvedores e agentes de IA consultam continuamente o grafo de conhecimento e notas de arquitetura através da CLI (`mem search`, `mem inspect`, `mem pack`, `mem path`) e de chamadas MCP (`memory_search`, `memory_inspect_node`, etc.). No entanto, as saídas textuais retornavam apenas caminhos de arquivos relativos (como `docs/adr/001-uso-de-sqlite-como-camada-unificada-de-dados.md`).

Para abrir ou anotar um documento recuperado, o usuário enfrentava atrito constante: copiar o caminho, abrir o Obsidian ou VS Code e navegar manualmente na árvore de pastas até localizar a nota. Além disso, os agentes de IA não tinham uma forma nativa de fornecer links clicáveis (*one-click deep links*) para abrir as notas diretamente no aplicativo do usuário.

## Decision Drivers

- **Navegação em Um Clique (*Click-to-Open UX*)**: Permitir abrir notas instantaneamente a partir do terminal ou chat de IA.
- **Conformidade com Esquemas Padrão de URI (RFC 3986)**:
  - `obsidian://open?vault=<vault>&file=<rel_path>` (com fallback `obsidian://open?path=<abs_path>`).
  - `vscode://file/<abs_path>[:line]`.
  - `file:///<abs_path>`.
- **Configuração Declarativa de Editor no Vault (`.memory/config.yaml`)**:
  - Preferência de aplicativo padrão (`editor.default_app: obsidian | vscode | system`).
  - Nome customizado do vault (`editor.obsidian_vault`), com fallback automático para `vault_name` ou nome da pasta raiz.
- **Comando de Linha de Comando `mem open`**:
  - Suporte a abertura por caminho no disco ou por resolução canônica de título/slug no banco SQLite/PostgreSQL.
  - Flags `--app`, `--line`, `--json` e `--dry-run` para automações e testes determinísticos.
- **Enriquecimento Não-Intrusivo de Saídas**:
  - Bloco `🔗 Links Rápidos` no `mem inspect`.
  - Flag `--links` no `mem search`.
  - Ferramenta MCP `memory_open_node` e metadados de links em inspeções e buscas.

## Decision Outcome

Criou-se o pacote puro `internal/deeplink` e integraram-se os novos recursos na CLI e no servidor MCP.

### 1. Pacote Puro `internal/deeplink`
- `GenerateLinks(repoRoot, vaultName, filePath string, line int) DeepLinks`: calcula e normaliza as três URIs (`Obsidian`, `VSCode`, `File`), aplicando codificação RFC 3986 e separadores universais.
- `Open(targetPathOrURI, app string, line int, repoRoot, vaultName string, launcher Launcher) (string, error)`: executa o launcher apropriado desacoplado do SO:
  - Windows: `cmd.exe /c start "" "<URI>"`
  - macOS: `open "<URI>"`
  - Linux: `xdg-open "<URI>"`

### 2. Configuração em `.memory/config.yaml`
```yaml
editor:
  default_app: "obsidian" # "obsidian", "vscode", "system"
  obsidian_vault: "My-Memory Knowledge Base" # Opcional (fallback automático)
```

### 3. Subcomando CLI `mem open`
```bash
# Abrir diretamente no Obsidian (padrão)
mem open docs/adr/001-uso-de-sqlite-como-camada-unificada-de-dados.md

# Abrir no VS Code com cursor posicionado na linha 42
mem open "Arquitetura Limpa" --app vscode --line 42

# Simular comando sem abrir aplicativo (útil em scripts/CI)
mem open 001-login --dry-run

# Emitir payload com deep links em formato JSON
mem open 001-login --json
```

### 4. Ferramenta MCP `memory_open_node`
Permite que agentes de IA enviem deep links formatados para o usuário ou disparem a abertura de notas:
```json
{
  "name": "memory_open_node",
  "arguments": {
    "node_id": "001-uso-de-sqlite-como-camada-unificada-de-dados",
    "app": "obsidian",
    "action": "links_only"
  }
}
```

## Consequences

### Positive
- Elimina a fricção de navegação manual entre os resultados da CLI/MCP e os editores de texto.
- Permite que chats de IA em terminais ou IDEs apresentem links clicáveis imediatos (`obsidian://` ou `vscode://`).
- Desacoplamento total via `Launcher` interface garante 100% de testabilidade determinística sem abertura de janelas em CI.

### Negative
- O protocolo `obsidian://` depende de o aplicativo Obsidian estar instalado e registrado no sistema operacional do usuário para abrir a janela do app. Caso não esteja instalado, a flag `--app vscode` ou `file` serve como alternativa garantida.
