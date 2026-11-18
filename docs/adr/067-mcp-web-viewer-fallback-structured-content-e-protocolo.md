# ADR-067: MCP Web e Viewer — Regra de Fallback SPA, structuredContent Aditivo e Negociação de Protocolo

- **Date**: 2026-11-18
- **Status**: Accepted — Implementado
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: `mcp`, `viewer`, `spa-fallback`, `structured-content`, `blast-radius`, `shortest-path`, `protocol-negotiation`, `adr-059`, `adr-060`
- **Adenda**: [ADR-006](006-integracao-com-agentes-de-ia-via-mcp.md) (MCP baseline) · [ADR-023](023-analise-de-impacto-e-raio-de-destruicao-blast-radius.md) (impacto) · [ADR-027](027-descoberta-de-rotas-e-caminho-minimo-no-grafo.md) (caminho) · [ADR-059](059-spa-react-vite-em-viewer-servida-pelo-mcp.md) (SPA estática) · [ADR-060](060-stack-do-frontend-da-viewer-e-fallback-spa.md) (stack e fallback SPA)
- **Issues**: [ISSUE-021](../../.specs/ISSUES.md#issue-021) · [ISSUE-022](../../.specs/ISSUES.md#issue-022) · [ISSUE-023](../../.specs/ISSUES.md#issue-023)

---

## Context and Problem Statement

A integração entre o servidor MCP em Go e a SPA React/Vite da Viewer (`viewer/`) opera via transporte HTTP estático e chamadas JSON-RPC. A auditoria pós-implementação das ADR-059 e ADR-060 levantou três lacunas no contrato do servidor:

1. **ISSUE-021 (Regra de Fallback SPA)**: O fallback SPA pretendia garantir que *"um asset faltando (`/assets/nao-existe.js`) continue sendo 404 honesto, e não HTML com 200"*. Contudo, navegadores reais sempre enviam `Accept: text/html` ao requisitar scripts e folhas de estilo, fazendo o servidor responder `200 OK` com `index.html` para assets ausentes.
2. **ISSUE-022 (`structuredContent` em `memory_get_impact` e `memory_find_path`)**: A viewer define os schemas `impactSchema` e `pathSchema` para renderizar visualmente a análise de raio de destruição (*blast radius*) e rotas entre nós no painel `/drift`. Todavia, os handlers MCP retornavam apenas texto (`NewTextResult`), forçando a tela a cair sempre no fallback textual de markdown cru.
3. **ISSUE-023 (Versão do Protocolo MCP e Formato de Payloads)**: O handshake `initialize` anunciava estaticamente a versão `2024-11-05`, enquanto a emissão de `structuredContent` foi formalizada na revisão `2025-06-18`. Ademais, 4 ferramentas retornam arrays na raiz do payload estruturado, formato já integrado e validado pelos schemas da viewer.

---

## Decision Drivers

- **DR-1 — 404 honesto para assets**: Um script, CSS ou imagem inexistente não pode devolver `index.html` com status 200, eliminando ruído em logs e evitando mascaramento de falhas de carregamento.
- **DR-2 — Renderização rica na Viewer**: A viewer deve validar os payloads estruturados de impacto e caminho com `impactSchema` e `pathSchema` sem cair no fallback de texto.
- **DR-3 — Ausência de null em arrays**: Em conformidade com o contrato wire (`TestStructuredContentNeverEmitsNullArrays`), slices vazios de nós, arestas ou clusters devem viajar como `[]`, nunca `null`.
- **DR-4 — Negociação de versão de protocolo sem quebra de clientes**: Clientes legados (`2024-11-05`, como Claude Desktop) e clientes modernos (`2025-06-18`) devem coexistir transparentemente.

---

## Decision Outcome

Adotou-se a resolução conjunta em três frentes:

### 1. Discriminação de Fallback por Extensão e Diretório (ISSUE-021)
Em `internal/mcp/static_server.go`, o fallback para `index.html` **só** se aplica a rotas puras de navegação da SPA:
- Se o caminho requisitado estiver sob o prefixo `/assets/` ou tiver extensão de arquivo (`filepath.Ext(last) != ""`), a ausência do arquivo em disco resulta imediatamente em **`404 Not Found`**, independentemente do cabeçalho `Accept` enviado pelo navegador.
- Rotas limpas de navegação da SPA (como `/clusters`, `/drift`, `/graph`) continuam caindo no fallback com `200 OK` e o entrypoint da SPA.

### 2. Emissão de `structuredContent` em Impacto e Rota (ISSUE-022)
- `internal/mcp/impact_handlers.go`: O handler `memory_get_impact` retorna `NewStructuredResult(sb.String(), res)`.
- `internal/mcp/path_handlers.go`: O handler `memory_find_path` retorna `NewStructuredResult(sb.String(), res)` em todos os fluxos (caminho encontrado, não encontrado ou saltos zero).
- `internal/mcp/structured_content.go`: A função `normalizeStructuredContent` normaliza `*graph.ImpactResult` e `*graph.PathResult`, garantindo que slices de nós (`Nodes`), arestas (`Edges`) e clusters (`AffectedClusters`) vazios sejam serializados como `[]` e nunca como `null`.

### 3. Negociação de Versão do Protocolo MCP (ISSUE-023)
- `internal/mcp/server.go`: No handshake `handleInitialize`, o servidor inspeciona `InitializeParams.ProtocolVersion`. Caso o cliente solicite `"2025-06-18"` ou `"2024-11-05"`, o servidor negocia e devolve a versão solicitada; se não informada ou desconhecida, preserva o padrão `"2024-11-05"`.
- Os formatos de array direto em `memory_list_files`, `memory_note_outline`, `memory_get_hubs` e `memory_get_insights` são mantidos e documentados como convenção deliberada de compatibilidade com os schemas leves da Viewer (`viewer/src/lib/schemas.ts`).

---

## Consequences

### Positive
- **Integridade de Cache e Telemetria**: Falhas em assets compilados são reportadas com status HTTP 404 real.
- **Painel `/drift` Plenamente Funcional**: As rotas de impacto e caminho validam com sucesso no TanStack Query e renderizam a interface rica da Viewer.
- **Compatibilidade Ampla**: Clientes estritos de protocolo MCP negociam sua versão preferencial sem atrito.
- **Zero quebras no Frontend**: Todos os schemas do Zod/Mini continuam válidos sem necessidade de refatorações drásticas de contratos.

### Negative / Trade-offs
- A especificação estrita do MCP 2025-06-18 sugere objetos na raiz do `structuredContent`; a coexistência de arrays em 4 ferramentas é uma escolha intencional para preservar a simplicidade e a estabilidade da viewer do repositório.
