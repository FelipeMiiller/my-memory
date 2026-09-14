# ADR-021: Servidor MCP com Transporte HTTP e Server-Sent Events (SSE)

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: mcp, model-context-protocol, sse, server-sent-events, http, network-server, cors, remote-agents, cli

## Context and Problem Statement

Na ADR-006, o **My-Memory** adotou o **Model Context Protocol (MCP)** via entrada e saída padrão (`stdin`/`stdout`). Esse formato atende perfeitamente ao caso de uso de processo filho local gerenciado por IDEs locais (como Cursor, Claude Desktop e Antigravity).

No entanto, a restrição exclusiva ao transporte `stdio` gerou obstáculos para cenários modernos de desenvolvimento e infraestrutura:
1. **Incompatibilidade com Agentes Remotos:** Agentes de IA hospedados na nuvem, em containers Docker ou em servidores dedicados não conseguem interagir com o MCP do repositório.
2. **Ausência de Concorrência Multi-Cliente:** O `stdio` limita o ciclo de vida a uma única conexão ponto-a-ponto entre cliente e servidor, impedindo que múltiplos assistentes (por exemplo, um agente de pair-programming na IDE e um pipeline de CI autônomo) acessem simultaneamente a mesma base de memória em tempo real.
3. **Falta de Endpoints de Observabilidade:** Não havia suporte a health checks padronizados (`/health`) para monitorar prontidão e métricas em ambientes de orquestração (Kubernetes, Docker Compose).
4. **Isolamento de Aplicações Web:** Painéis web e dashboards interativos não conseguiam executar chamadas JSON-RPC ao MCP sem gateways ou proxies reversos dedicados.

## Decision Drivers

- **Conformidade com a Especificação Oficial do MCP**: O transporte de rede deve seguir fielmente o padrão de Server-Sent Events (SSE) estabelecido pela especificação do Model Context Protocol (Anthropic).
- **Retrocompatibilidade Total (Zero Breaking Changes)**: O comando `mem mcp` sem parâmetros de rede deve continuar operando em modo `stdio`, sem alterar o comportamento com clientes existentes.
- **Transparência e Ergonomia**: Suporte a flags de porta e endereço de rede na CLI (`--port`, `--host`, `--http`).
- **Endpoint Direto e Diagnóstico**: Disponibilizar `/health` para monitoramento e `/mcp` para clientes HTTP simples sem overhead de stream SSE.
- **Suporte Nativo a CORS**: Habilitar preflight requests (`OPTIONS`) e cabeçalhos adequados para integrações com navegadores.
- **Encerramento Gracioso**: Fechamento seguro de streams SSE e liberação de sockets mediante sinais do sistema (`SIGINT`, `SIGTERM`) ou cancelamento de contexto.

## Decision Outcome

Adotou-se o servidor de rede em `internal/mcp/http_server.go` integrado ao comando `mem mcp` em `cmd/mem/main.go`:

1. **Arquitetura do `internal/mcp/http_server.go`:**
   - **`GET /sse`**: Endpoint de Server-Sent Events (`text/event-stream`). Gera um identificador único de sessão (`sessionId`) via `crypto/rand`, registra um canal de buffer em memória e emite o evento obrigatório `endpoint`:
     ```http
     event: endpoint
     data: /message?sessionId=<uuid>
     ```
   - **`POST /message?sessionId=<uuid>`**: Recebe mensagens JSON-RPC 2.0 do cliente, despacha para `Server.HandleRequest` e transmite a resposta correspondente pelo stream SSE ativo da sessão (além de responder HTTP 202 Accepted).
   - **`POST /mcp`**: Endpoint JSON-RPC direto (stateless) para chamadas rápidas e testes via `curl` sem necessidade de abrir streams SSE.
   - **`GET /health`**: Retorna status JSON (`healthy`), versão do protocolo (`2024-11-05`), uptime em segundos, total de ferramentas registradas e repositório padrão.
   - **Middleware `withCORS`**: Insere cabeçalhos `Access-Control-Allow-Origin: *`, `Access-Control-Allow-Methods` e responde preflights `OPTIONS` com HTTP 204 No Content.

2. **Flags na CLI (`cmd/mem/`):**
   - `mem mcp`: Executa no modo clássico `stdio` para processos filhos.
   - `mem mcp --port 38400`: Inicia o servidor HTTP/SSE na porta 38400 (ligado a `127.0.0.1`).
   - `mem mcp --host 0.0.0.0 --port 38400`: Expõe o servidor MCP para a rede local ou containers.
   - `mem mcp --http :38400`: Atalho alternativo para especificação direta de host:porta.

### Positive Consequences

- **Acesso Universal de Agentes Remotos:** Agentes remotos, instâncias de containers e automações em nuvem podem acessar o My-Memory via SSE sobre HTTP.
- **Múltiplos Clientes Simultâneos:** Múltiplas conexões SSE paralelas são gerenciadas concorrentemente em memória com isolamento total por canal.
- **Observabilidade Imediata:** Verificação de saúde e ferramentas ativas em tempo real com `GET /health`.
- **Compatibilidade Preservada:** Nenhuma configuração de Claude Desktop ou Cursor local precisou ser alterada.

### Negative Consequences / Trade-offs

- **Exposição de Rede e Segurança:** Sem autenticação por token ou TLS embutido nesta fase, a exposição direta em redes públicas deve ser mediada por reverse proxies seguros (Nginx, Traefik, Caddy com mTLS ou API Key).

---

## Links

- [Especificação Oficial MCP - Transports](https://modelcontextprotocol.io/docs/concepts/transports)
- [ADR-006: Integração com Agentes de IA via Model Context Protocol](006-integracao-com-agentes-de-ia-via-mcp.md)
- [Guia da Linha de Comando](../CLI_GUIDE.md)
