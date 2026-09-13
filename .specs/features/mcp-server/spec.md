# MCP Server Specification

## Problem Statement

Agentes de Inteligência Artificial como Claude Code, Cursor, Antigravity e Windsurf precisam consultar a memória semântica e relacional de projetos de forma autônoma. Atualmente, o `my-memory` só pode ser operado via comandos manuais do terminal, sem fornecer uma interface padronizada de chamadas de ferramentas (*Tool Use*) para IAs.

## Goals

- [ ] Implementar um servidor local compatível com a especificação Model Context Protocol (MCP) via `stdio` (JSON-RPC 2.0).
- [ ] Expor as ferramentas `memory_search` e `memory_get_neighbors` com esquemas JSON Schema válidos.
- [ ] Garantir isolamento estrito entre mensagens do protocolo (`stdout`) e registros de log/erros (`stderr`).
- [ ] Fornecer o comando de entrada `mem mcp [--db <caminho>]`.

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
| ------- | ------ |
| Transporte HTTP / SSE / WebSockets | O foco inicial é execução local via processos filhos em `stdio`. |
| Autenticação e tokens de acesso | Processo local filho executado no mesmo ambiente de segurança do usuário. |
| Ingestão/Modificação de notas via MCP | O servidor MCP inicial atuará em modo somente-leitura (*read-only*) para busca e grafo. |

---

## Assumptions & Open Questions

Every ambiguity is resolved or recorded here - nothing is left silently unclear.

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Protocolo de transporte | stdio (stdin/stdout) | Padrão nativo do Claude Code e Cursor para ferramentas locais | y |
| Formato de codificação | JSON-RPC 2.0 com delimitação por quebra de linha | Especificação oficial do Model Context Protocol | y |
| Caminho padrão do banco | memory.db no diretório de trabalho atual | Mantém compatibilidade com a CLI existente | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Handshake e Descoberta de Ferramentas ⭐ MVP

**User Story**: As a AI coding agent, I want to connect to `mem mcp` via stdio so that I can discover available memory tools.

**Why P1**: Sem o handshake e o catálogo de ferramentas, o cliente de IA não consegue registrar nem invocar nenhuma capacidade.

**Acceptance Criteria**:

1. WHEN the AI client sends an `initialize` JSON-RPC request THEN system SHALL return server capabilities with protocol version "2024-11-05" and server info. <!-- event-driven -->
2. WHEN the AI client sends a `tools/list` request THEN system SHALL return the schema definition of `memory_search` and `memory_get_neighbors`. <!-- event-driven -->
3. WHILE the MCP server is running system SHALL send all JSON-RPC responses exclusively to stdout. <!-- state-driven -->
4. The system SHALL redirect all diagnostic logs and debugging messages to stderr. <!-- ubiquitous -->
5. IF the AI client sends an invalid JSON-RPC payload THEN system SHALL return a standard ParseError or InvalidRequest error object. <!-- unwanted-behavior -->

**Independent Test**: Iniciar o processo `mem mcp` e enviar requisições de `initialize` e `tools/list` via pipe stdin, validando as respostas correspondentes em stdout.

---

### P2: Execução de Busca e Travessia de Grafo

**User Story**: As a AI coding agent, I want to call `memory_search` and `memory_get_neighbors` so that I can retrieve context and connected notes.

**Why P2**: Permite que a IA execute perguntas em linguagem natural e explore relações no grafo durante a edição de código.

**Acceptance Criteria**:

1. WHEN the AI client sends a `tools/call` request for `memory_search` with a query string THEN system SHALL execute vector search on SQLite and return formatted text chunks with distances and document paths. <!-- event-driven -->
2. WHEN the AI client sends a `tools/call` request for `memory_get_neighbors` with a node ID THEN system SHALL execute recursive CTE traversal and return connected neighbors. <!-- event-driven -->
3. IF the requested tool name is unknown THEN system SHALL return an MCP error response with code -32601 (Method not found). <!-- unwanted-behavior -->
4. IF a required argument is missing from `tools/call` THEN system SHALL return an error response indicating the missing parameter. <!-- unwanted-behavior -->

**Independent Test**: Executar requisição de `tools/call` com argumentos válidos e verificar se o JSON de resultado contém os dados consultados no banco SQLite.

---

## Edge Cases

- IF the SQLite database file does not exist at startup THEN system SHALL emit an explanatory error message to stderr and return an error response.
- IF an empty query is provided to `memory_search` THEN system SHALL return an empty result list without crashing.
- WHEN multiple requests are sent sequentially through the same stdin stream THEN system SHALL process each request and output the corresponding response with matching ID.

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| MCP-01 | P1: Handshake e Descoberta de Ferramentas | Design | Pending |
| MCP-02 | P1: Handshake e Descoberta de Ferramentas | Design | Pending |
| MCP-03 | P1: Handshake e Descoberta de Ferramentas | Design | Pending |
| MCP-04 | P2: Execução de Busca e Travessia de Grafo | Design | Pending |
| MCP-05 | P2: Execução de Busca e Travessia de Grafo | Design | Pending |

**Coverage:** 5 total, 5 mapped to stories, 0 unmapped

---

## Success Criteria

- [ ] AI client can complete initialization handshake in < 50ms.
- [ ] Zero non-JSON-RPC output emitted on stdout.
- [ ] Claude Code or Cursor can inspect available tools and invoke `memory_search` successfully.