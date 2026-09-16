# Feature Specification: federated-canonical-protocol-and-cross-vault-wikilinks

Status: pending
Created: 2026-09-16
Feature: federated-canonical-protocol-and-cross-vault-wikilinks
ADR: docs/adr/034-protocolo-canonico-federado-e-wikilinks-cross-vault.md

---

## Problem Statement

Com a introdução da arquitetura federada de conhecimento ([ADR-033](../../../docs/adr/033-federated-central-vault-and-repo-identity.md)), o My-Memory conecta o **Cofre Central de Conhecimento** (*Global Brain*) a múltiplos **Cofres de Projetos** (*Local Brains*).

Entretanto, desenvolvedores e agentes de IA precisam referenciar padrões corporativos centrais (ex: `standards/oauth2`, `architecture/database-standards`) e especificações de outros repositórios satélites diretamente nas notas de projetos locais. O padrão clássico de wikilinks (`[[Nota]]`) é incapaz de resolver referências fora do diretório raiz local, rotulando os links como órfãos ou desconhecidos. Por outro lado, usar caminhos absolutos do sistema operacional (ex: `C:/Users/...`) destrói a portabilidade no Git, vaza o layout privado do host e quebra a sincronização entre plataformas (Windows, Linux, macOS).

O sistema necessita de um protocolo canônico e universal de URIs (`memory://<repo>/<path>[#anchor]`) integrado ao parser de Markdown, ao grafo de arestas, ao comando `mem open` e ao servidor MCP.

---

## Out of Scope

- Sincronização e push de arquivos Markdown remotos via WebDAV ou APIs proprietárias do Google Drive/OneDrive (o sistema confia na sincronização de pastas locais do host).
- Criação automática de links reversos bidirecionais físicos gravados em arquivos Markdown de repositórios externos somente-leitura.
- Resolução de repositórios que não estejam cadastrados em `~/.memory/config.yaml` ou no contexto local.

---

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale |
| :--- | :--- | :--- |
| Sinônimos do Cofre Central | `repo_central` e `central` | Facilitar a escrita humana (`central`) sem quebrar a identidade técnica (`repo_central`). |
| Resolução de Extensão | `.md` opcional | Permitir tanto `memory://central/standards/auth` quanto `memory://central/standards/auth.md`. |
| Comportamento de Vault Inacessível | Aviso não-bloqueante | Se a nuvem estiver desplugada, o CLI e MCP retornam diagnóstico sem quebrar as operações locais. |

Open questions: none

---

## User Stories

### US-1: Referência a Padrões Centrais em Notas Locais
Como engenheiro de software documentando um microsserviço local,
Quero poder escrever `[[memory://central/standards/oauth2|Padrão de Autenticação]]` ou `[[implements:memory://central/standards/oauth2]]`,
Para que a decisão local cite a diretriz corporativa com portabilidade no Git e sem quebrar o grafo de conhecimento.

### US-2: Navegação e Abertura em Editores
Como desenvolvedor utilizando o terminal,
Quero executar `mem open "memory://central/standards/oauth2#JWT"` ou acionar a ferramenta MCP `memory_open_node`,
Para abrir diretamente o Obsidian ou VS Code na nota do cofre central com foco na seção correspondente.

### US-3: Descoberta de Conexões Federadas pelo Agente de IA
Como um agente de IA (Cursor, Claude Code, Antigravity) analisando uma nota local,
Quero inspecionar os vizinhos via `memory_get_neighbors` e identificar arestas federadas cross-vault,
Para que eu consiga consultar o padrão corporativo no cofre central de forma transparente.

---

## Requirements (EARS Notation)

### FEDLINK-01: Sintaxe Canônica e Dissecação de URIs `memory://`
- **UBIQUITOUS**: The system SHALL recognize and parse canonical federated URIs using the schema `memory://<repo_identifier>/<document_path>[#<anchor_or_block>]`.
- **UBIQUITOUS**: The system SHALL resolve `central` and `repo_central` as valid target identifiers referencing the Central Vault.
- **WHERE**: Where `<repo_identifier>` is a 12-hex repository ID or a registered repository slug from `~/.memory/config.yaml`, the system SHALL identify the target as a satellite repository.

### FEDLINK-02: Blindagem do Analisador de Wikilinks
- **UBIQUITOUS**: The system SHALL preserve `memory://` URI prefixes during wikilink extraction without misclassifying `memory` as a semantic relation.
- **WHERE**: Where a wikilink contains an epistemic relation prefix such as `[[implements:memory://...]]` or an alias such as `[[memory://...|Texto]]`, the system SHALL extract both the semantic relation and the full canonical target URI.
- **UBIQUITOUS**: The system SHALL populate `IsFederated: true`, `FederatedRepo`, and `FederatedPath` in the extracted `LinkTarget` structure.

### FEDLINK-03: Motor de Resolução Federada em Disco
- **WHEN**: When a federated URI is resolved by the system, the system SHALL locate the physical directory of the target vault using `central_vault.path` or the repository catalog in `~/.memory/config.yaml`.
- **WHERE**: Where the resolved target path exists on the host filesystem, the system SHALL return the absolute path, the target vault name, and the relative document path.
- **WHERE**: Where the target vault or document is unmounted or missing, the system SHALL return a structured descriptive error indicating unmounted storage.

### FEDLINK-04: Navegação em Editores e Deep Linking (`mem open`)
- **WHEN**: When `mem open` is executed with a `memory://` URI, the system SHALL resolve the target file and launch the configured editor (`obsidian` or `vscode`).
- **UBIQUITOUS**: The system SHALL construct valid `obsidian://open?vault=<vault>&file=<rel>` and `vscode://file/<abs>` URIs for federated documents.

### FEDLINK-05: Integração com Servidor MCP
- **WHEN**: When the MCP tool `memory_open_node` is called with a `memory://` URI, the system SHALL resolve the target node and perform editor opening.
- **WHEN**: When `memory_get_neighbors` is invoked for a node containing federated links, the system SHALL flag cross-vault edges with `is_federated: true` and the target canonical URI.

---

## Requirement Traceability

| Requirement ID | Description | Status |
| :--- | :--- | :--- |
| FEDLINK-01 | Sintaxe Canônica e Dissecação de URIs `memory://` | in tasks |
| FEDLINK-02 | Blindagem do Analisador de Wikilinks | in tasks |
| FEDLINK-03 | Motor de Resolução Federada em Disco | in tasks |
| FEDLINK-04 | Navegação em Editores e Deep Linking (`mem open`) | in tasks |
| FEDLINK-05 | Integração com Servidor MCP | in tasks |
