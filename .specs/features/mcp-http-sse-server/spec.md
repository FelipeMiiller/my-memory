# Feature: mcp-http-sse-server

## Problem Statement
Atualmente, o servidor Model Context Protocol (**MCP**) do My-Memory opera exclusivamente via entrada e saída padrão (`stdin`/`stdout`, ADR-006). Essa limitação impõe restrições severas ao ecossistema:
1. **Inacessibilidade Remota:** Agentes de inteligência artificial em execução em containers, nuvem ou servidores remotos não conseguem consumir as ferramentas de memória do repositório.
2. **Concorrência Inexistente:** O modelo `stdio` conecta apenas um processo cliente a um processo servidor local, inviabilizando que múltiplos assistentes (ex: Claude Code e Cursor simultaneamente) consultem o mesmo repositório em tempo real.
3. **Falta de Observabilidade HTTP:** Não há endpoints padronizados de health check e métricas (`/health`) para monitorar o ciclo de vida do servidor de memória em pipelines de orquestração.
4. **Desconexão com Aplicações Web:** Interfaces baseadas em navegador não conseguem interagir com o MCP sem gateways ou pontes externas.

## Goals
- [ ] Implementar servidor HTTP nativo em `internal/mcp` suportando transporte Server-Sent Events (SSE) em conformidade com a especificação oficial do Model Context Protocol (Anthropic).
- [ ] Implementar endpoint `GET /sse` com emissão do evento `endpoint` (`event: endpoint\ndata: /message?sessionId=<uuid>\n\n`) e manutenção de stream ativo.
- [ ] Implementar endpoint `POST /message?sessionId=<uuid>` para processar requisições JSON-RPC e despachar respostas tanto via stream SSE quanto resposta HTTP.
- [ ] Implementar endpoint `POST /mcp` para chamadas JSON-RPC diretas e sem estado (stateless RPC).
- [ ] Implementar endpoint `GET /health` reportando status operacional, uptime, total de ferramentas registradas e repositório padrão.
- [ ] Implementar suporte completo a Cross-Origin Resource Sharing (CORS) e preflight requests (`OPTIONS`).
- [ ] Implementar encerramento gracioso (*graceful shutdown*) gerenciado por `context.Context`.
- [ ] Estender a CLI `mem mcp` com flags `--port`, `--host` e `--http`, mantendo `stdio` como transporte padrão retrocompatível.
- [ ] Formalizar as decisões na ADR-021 e atualizar documentação técnica.

## Out of Scope
- Autenticação OAuth2 / JWT para o servidor MCP nesta fase (o servidor é desenhado para redes locais, containers ou atrás de reverse proxies com mTLS/token).
- Transporte WebSocket bidirecional customizado (a especificação oficial do MCP padroniza SSE sobre HTTP, evitando protocolos não normatizados).

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Retrocompatibilidade Stdio | Stdio permanece o padrão quando nenhuma flag de rede for especificada | Preserva integração transparente com clientes locais existentes como Claude Desktop e Cursor | y |
| Especificação SSE | Segue a especificação oficial MCP (GET /sse com evento endpoint e POST /message) | Garante interoperabilidade universal com SDKs oficiais do MCP | y |
| Concorrência de Sessões | Múltiplas sessões SSE simultâneas gerenciadas concorrentemente em memória | Permite conexão de vários agentes de IA paralelos | y |
| Endpoint Direto /mcp | Oferecido como facilidade para clientes HTTP simples sem SSE | Reduz atrito para scripts e testes rápidos | y |
| Política de CORS | Liberado (`*`) por padrão com suporte a preflight OPTIONS | Viabiliza dashboards locais e clientes web sem bloqueios de navegador | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Transporte HTTP com Server-Sent Events (SSE) ⭐ MVP

**User Story**: As a agente de IA conectado via rede, I want abrir uma conexão SSE e enviar requisições JSON-RPC via POST so that eu acesse todas as ferramentas do My-Memory remotamente conforme o protocolo oficial MCP.

**Why P1**: Núcleo indispensável da especificação MCP de rede.

**Acceptance Criteria**:
1. WHEN a client sends a GET request to `/sse` THEN the system SHALL establish a text/event-stream connection, register a unique sessionId, and emit an initial `endpoint` event containing the message URI.
2. WHEN a client sends a POST request with valid JSON-RPC to `/message?sessionId=<id>` THEN the system SHALL process the request and dispatch the JSON-RPC response through the corresponding SSE stream.
3. IF a client sends a POST request to `/message` with an unknown or missing sessionId THEN the system SHALL return HTTP 400 or 404 with a structured JSON error.
4. WHILE a client keeps the SSE connection open, the system SHALL stream events in real time until the client disconnects or the context is cancelled.
5. IF the client disconnects the SSE connection THEN the system SHALL clean up the active session resources immediately.

**Independent Test**: Testes com `httptest.Server` validando handshake SSE e despacho via `/message`.

---

### P2: Endpoints de Saúde e RPC Direto (Health & Direct MCP)

**User Story**: As a orquestrador de infraestrutura ou desenvolvedor, I want consultar `GET /health` e fazer chamadas diretas em `POST /mcp` so that eu verifique a saúde do serviço e execute consultas pontuais sem estabelecer streams SSE.

**Why P2**: Simplifica monitoramento em containers e testes via curl.

**Acceptance Criteria**:
1. WHEN a client sends a GET request to `/health` THEN the system SHALL return HTTP 200 with JSON payload containing status, uptime, registered tools count and repository info.
2. WHEN a client sends a POST request to `/mcp` with a JSON-RPC payload THEN the system SHALL execute the tool or method and return the JSON-RPC response directly in the HTTP body with application/json.
3. IF a request to `/mcp` or `/message` contains invalid JSON THEN the system SHALL return a JSON-RPC ParseError with code -32700.

**Independent Test**: Testes unitários validando `/health` e `/mcp`.

---

### P3: Suporte a CORS e Integração CLI (mem mcp --port / --http)

**User Story**: As a usuário de linha de comando ou criador de interfaces web, I want iniciar o servidor MCP com flags `--port` e poder chamá-lo a partir de browsers so that eu escolha facilmente entre modo local stdio ou servidor de rede.

**Why P3**: Expõe a funcionalidade de forma intuitiva aos operadores humanos.

**Acceptance Criteria**:
1. WHERE CORS is enabled the system SHALL append Access-Control headers to all HTTP responses and answer OPTIONS preflight requests with HTTP 204.
2. WHERE the `--port` or `--http` flag is passed to `mem mcp` the system SHALL start the network HTTP/SSE server instead of the stdio loop.
3. WHERE no network flags are passed to `mem mcp` the system SHALL default to stdio communication for full backward compatibility.
4. The system SHALL support graceful shutdown when receiving interrupt signals or context cancellation.

**Independent Test**: Testes de CLI com flags `--port` e testes de cabeçalhos CORS.

---

### P4: Registro Arquitetural ADR-021 e Documentação

**User Story**: As a engenheiro do projeto, I want consultar a ADR-021 e o CLI_GUIDE atualizado so that eu compreenda as decisões técnicas e exemplos de configuração para Claude Code e Cursor.

**Why P4**: Garante manutenibilidade e integridade da documentação do repositório autoconsciente.

**Acceptance Criteria**:
1. The system SHALL maintain ADR-021 documenting the context, options, consequences and endpoints of the HTTP/SSE server.
2. The system SHALL document network usage and configuration examples in `docs/CLI_GUIDE.md`.

**Independent Test**: Verificação de integridade dos arquivos markdown e links relativos.

---

## Requirement Traceability

| Requirement ID | User Story / Feature Area | Status |
| -------------- | ------------------------- | ------ |
| MCP-01 | Transporte SSE (GET /sse com evento endpoint) | verified |
| MCP-02 | Despacho JSON-RPC via POST /message | verified |
| MCP-03 | Limpeza e isolamento de sessões ativas | verified |
| MCP-04 | Endpoint de diagnóstico e prontidão GET /health | verified |
| MCP-05 | Endpoint RPC direto POST /mcp | verified |
| MCP-06 | Middleware CORS e suporte a preflight OPTIONS | verified |
| MCP-07 | Flags CLI `--port`, `--host`, `--http` e shutdown gracioso | verified |
| MCP-08 | Documentação técnica e ADR-021 | pending |
