# Tasks: mcp-http-sse-server

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| MCP HTTP Core | unit | NewHTTPServer, handleSSE, handleMessage, session management, shutdown | internal/mcp/http_server_test.go | go test -v ./internal/mcp/... |
| MCP Endpoints & CORS | unit | handleHealth, handleDirectRPC, CORS preflight and headers | internal/mcp/http_server_test.go | go test -v ./internal/mcp/... |
| CLI MCP Network | integration | mem mcp --port, health check over HTTP, stdio fallback | cmd/mem/mcp_server_test.go | go test -v ./cmd/mem/... |
| Documentation / ADR | none | ADR-021, CLI_GUIDE, STATE.md updates | docs/adr/* | go test -v ./cmd/mem/... ./internal/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| MCP Gate | After HTTP server core and endpoints | go test -v ./internal/mcp/... |
| CLI Gate | After CLI flag wiring and network runner | go test -v -run TestMCPServer ./cmd/mem/... |
| Full Test | Before documentation and state update | go test -count=1 -v ./cmd/mem/... ./internal/... |
| Spec Gate | Before completing feature | python C:\Users\Felipe\.cache\agent-skills\skills\tlc-spec-driven\scripts\validate_spec.py mcp-http-sse-server |
| Tasks Gate | Before completing tasks | python C:\Users\Felipe\.cache\agent-skills\skills\tlc-spec-driven\scripts\validate_tasks.py mcp-http-sse-server |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T3 -> T4
```

### Phase 1: Servidor de Rede MCP (HTTP / SSE / Direct RPC)

## Task Breakdown

### T1: Servidor HTTP/SSE Core e Gerenciamento de Sessões
**What**: Implementar HTTPServer em internal/mcp/http_server.go com suporte a GET /sse e POST /message
**Where**: internal/mcp/http_server.go
**Depends on**: none
**Requirement**: MCP-01, MCP-02, MCP-03
**Tests**: internal/mcp/http_server_test.go
**Gate**: go test -v -run TestHTTPServer_SSE ./internal/mcp/...
**Done when**:
- [x] Implementar estrutura HTTPServer e NewHTTPServer em internal/mcp/http_server.go
- [x] Implementar handleSSE com cabeçalhos text/event-stream, geração de sessionId e emissão do evento endpoint
- [x] Implementar handleMessage recebendo JSON-RPC via POST e despachando pelo canal SSE da sessão
- [x] Implementar gerenciamento e descarte de sessões quando cliente desconectar
- [x] Implementar método Shutdown(ctx) gracioso
- [x] Adicionar testes em internal/mcp/http_server_test.go validando o ciclo SSE e POST /message

### T2: Endpoints Auxiliares (/health, /mcp) e Middleware CORS
**What**: Implementar endpoint de diagnóstico /health, endpoint RPC direto /mcp e cabeçalhos CORS
**Where**: internal/mcp/http_server.go
**Depends on**: T1
**Requirement**: MCP-04, MCP-05, MCP-06
**Tests**: internal/mcp/http_server_test.go
**Gate**: go test -v -run "TestHTTPServer_Health|TestHTTPServer_DirectRPC|TestHTTPServer_CORS" ./internal/mcp/...
**Done when**:
- [x] Implementar handleHealth retornando JSON com status, uptime, total de tools e repositório padrão
- [x] Implementar handleDirectRPC permitindo invocar ferramentas via POST /mcp sem necessidade de stream SSE
- [x] Implementar middleware withCORS com suporte a preflight OPTIONS (HTTP 204)
- [x] Adicionar testes unitários para /health, /mcp e CORS em internal/mcp/http_server_test.go

### T3: Extensão da CLI mem mcp com Flags de Rede
**What**: Estender comando mem mcp em cmd/mem/main.go para suportar --port, --host e --http com fallback stdio
**Where**: cmd/mem/main.go
**Depends on**: T2
**Requirement**: MCP-07
**Tests**: cmd/mem/mcp_server_test.go
**Gate**: go test -v -run TestMCPServer_CLI ./cmd/mem/...
**Done when**:
- [ ] Adicionar flags --port, --host e --http no subcomando mem mcp
- [ ] Iniciar HTTPServer quando uma porta for especificada, ou cair no loop stdio quando omitida
- [ ] Atualizar printHelp() documentando a nova funcionalidade de servidor de rede
- [ ] Adicionar teste de integração em cmd/mem/mcp_server_test.go testando inicialização em porta dinâmica e shutdown

### T4: ADR-021, Documentação Técnica e Atualização do STATE.md
**What**: Registrar a ADR-021, atualizar guias de uso e documentar no STATE.md do TLC
**Where**: docs/adr/021-servidor-mcp-com-transporte-http-sse.md
**Depends on**: T3
**Requirement**: MCP-08
**Tests**: none
**Gate**: go test -count=1 ./...
**Done when**:
- [ ] Criar docs/adr/021-servidor-mcp-com-transporte-http-sse.md no padrão MADR
- [ ] Atualizar índice em docs/adr/README.md
- [ ] Atualizar exemplos em docs/CLI_GUIDE.md com curl e configuração de clientes MCP remotos
- [ ] Atualizar .specs/STATE.md com a decisão AD-021 e status do handoff
