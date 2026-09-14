# Feature: deep-linking-and-editor-navigation

## Problem Statement
Desenvolvedores e agentes de IA consultam frequentemente a base de conhecimento através do CLI (`mem search`, `mem inspect`, `mem pack`, `mem path`) e de ferramentas MCP (`memory_search`, `memory_inspect_node`, etc.). No entanto, os resultados retornam caminhos textuais simples (ex: `docs/adr/001-uso-de-sqlite-como-camada-unificada-de-dados.md`).

Para visualizar, editar ou anotar o documento encontrado, o usuário precisa copiar o caminho, alternar para o Obsidian ou VS Code e navegar manualmente na árvore de diretórios. Essa fricção prejudica a produtividade e impede fluxos de navegação fluida (*one-click navigation*).

A funcionalidade **Deep Linking e Integração de Navegação com Editores** resolve esse problema ao gerar URIs acionáveis (`obsidian://open?vault=...&file=...`, `vscode://file/...` e `file:///...`) em todas as saídas de inspeção e busca, fornecer o comando CLI `mem open <nota>` para abertura nativa e expor a ferramenta MCP `memory_open_node`.

---

## Out of Scope
- Edição de conteúdo diretamente via URI (os esquemas de deep link focam em abertura e foco no arquivo/linha).
- Plugins nativos internos de terceiros (o sistema usa os manipuladores de protocolo de SO já registrados pelo Obsidian e VS Code).

---

## Assumptions & Open Questions

| Assumption / Question | Chosen default | Rationale |
| :--- | :--- | :--- |
| Formato de URI do Obsidian | `obsidian://open?vault=<vault>&file=<path>` | Protocolo oficial do Obsidian Desktop e Mobile, aceito universalmente. |
| Fallback para Vault Desconhecido | `obsidian://open?path=<abs_path>` ou nome da pasta | Obsidian Desktop suporta o parâmetro `path` com caminho absoluto quando o nome do vault não é configurado explicitamente. |
| Formato de URI do VS Code | `vscode://file/<abs_path>[:line]` | Protocolo padrão do Visual Studio Code e derivados (Cursor, Windsurf). |
| Resolução do nó no `mem open` | Arquivo em disco ou busca na base de dados | Permite abrir tanto por caminho relativo/absoluto quanto por identificador canônico ou título de nota indexada. |
| Modo de teste do launcher | Suporte a mock e flag `--dry-run` | Permite testes unitários e de integração determinísticos sem disparar janelas de aplicativos no ambiente de CI/CD. |

Open questions: none (todas as premissas e opções de protocolo foram alinhadas e pré-aprovadas).

---

## User Stories

- **US1**: Como desenvolvedor navegando no terminal, quero executar `mem open <nota>` para abrir imediatamente o documento no Obsidian ou VS Code na linha relevante.
- **US2**: Como agente de IA ou desenvolvedor inspecionando o grafo, quero receber links clicáveis (`obsidian://` e `vscode://`) no resultado de `mem inspect` e `mem search` para inspecionar notas com um clique.
- **US3**: Como agente conectado via MCP, quero acionar `memory_open_node` ou sugerir deep links clicáveis para o usuário na interface do chat.

---

## Requirements (EARS Notation)

### DL-01: Geração de Deep Links Canônicos
- **UBIQUITOUS**: The system SHALL generate valid RFC 3986 URIs for Obsidian (`obsidian://open?vault=...&file=...`), VS Code (`vscode://file/...`) and OS file (`file:///...`) from any valid document path.
- **WHERE**: Where an explicit line number greater than zero is provided, the system SHALL append line coordinates to the generated VS Code and file URIs.

### DL-02: Resolução Automática de Vault
- **WHEN**: When an Obsidian URI is requested, the system SHALL resolve the vault name prioritizing `editor.obsidian_vault` from configuration, falling back to `vault_name`, and defaulting to the repository root directory name.

### DL-03: Comando CLI `mem open`
- **WHEN**: When the user executes `mem open <target> [--app <app>] [--line <line>]`, the system SHALL resolve the target node and launch the corresponding URI or application via the operating system handler.
- **WHEN**: When the `--dry-run` flag is provided to `mem open`, the system SHALL print the resolved URI and command to standard output without executing the system process.
- **WHEN**: When the `--json` flag is provided to `mem open`, the system SHALL output a JSON payload containing the canonical node, resolved path, and generated deep links.

### DL-04: Enriquecimento de Inspeção e Busca
- **UBIQUITOUS**: The system SHALL display quick clickable deep links in the terminal output of `mem inspect` and include a `links` object with deep links in the structured JSON response.
- **WHEN**: When the user executes `mem search` with the `--links` flag, the system SHALL print Obsidian and VS Code URIs below each matching search result.

### DL-05: Ferramenta MCP `memory_open_node`
- **WHEN**: When an AI agent invokes `memory_open_node` with a valid `node_id`, the system SHALL return the generated deep links and confirmation of the open action.

---

## Requirement Traceability

| Requirement ID | Description | Status |
| :--- | :--- | :--- |
| DL-01 | Geração de Deep Links Canônicos (Obsidian, VS Code, File) | in tasks |
| DL-02 | Resolução Automática de Vault e Configuração do Editor | in tasks |
| DL-03 | Comando CLI `mem open` com `--dry-run`, `--json`, `--app` e `--line` | in tasks |
| DL-04 | Enriquecimento de `mem inspect` e `mem search` com Links Rápidos | in tasks |
| DL-05 | Ferramenta MCP `memory_open_node` e metadados de links em respostas | in tasks |
