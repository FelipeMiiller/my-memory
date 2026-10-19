# Painel de métricas da viewer — SPA que consome o MCP

## Problem Statement

`viewer/` é hoje o **starter padrão do shadcn**: `viewer/src/App.tsx` renderiza "Project ready!" e um botão. Ele não consome o MCP, não tem rota, não tem estado, e não mostra uma única métrica do vault.

Isso deixa um buraco entre duas decisões já tomadas:

- A [ADR-059](../../../docs/adr/059-spa-react-vite-em-viewer-servida-pelo-mcp.md) fixou **que** a viewer existe e **como** ela é servida: build estático em `viewer/dist/`, servido pelo transporte HTTP do MCP, nunca a raiz do vault. Ela define a postura de segurança (mesma origem, sem CDN, sem autenticação — lacuna pré-existente) e a fronteira de escopo: **a viewer é painel de métricas, não editor**.
- A [ADR-060](../../../docs/adr/060-stack-do-frontend-da-viewer-e-fallback-spa.md) fixou **com o que** a SPA é construída (TanStack Query + Router + Zustand + `zod/mini` + Markdown), com custos medidos, e fechou as duas decisões que a ADR-059 deixou abertas: o roteamento de cliente (com a rota de fallback SPA no servidor) e o tratamento de wikilinks.

A ADR-060 registra explicitamente, em "Estado da Implementação": *"A spec do lado do cliente (o painel em si) ainda **não existe** e precisa ser criada."* A spec do servidor (`mcp-static-serving`, `SERV-01..SERV-25`) cobre a **serventia**; ela não cobre o **painel**.

O buraco tem quatro consequências concretas:

1. **As 24 tools MCP produzem métricas que existem só como texto no terminal.** Health Score, órfãs, dead links, modularidade, PageRank, hubs, clusters, conexões latentes, drift, blast radius e caminho entre nós são todos calculados — e nenhum deles é apresentável.
2. **O vault é Obsidian Flavored Markdown, e `[[wikilink]]` é a aresta do grafo** (`internal/parser/wikilinks.go`). O renderer de Markdown escolhido (`@tanstack/markdown`) **não** conhece wikilinks — verificado na doc oficial: nenhuma menção a "wikilink", "Obsidian" ou `[[`. Sem um pré-processador, a viewer ou mostra `[[X]]` como texto literal, ou — se linkar ingenuamente — **mente sobre a topologia**, mostrando links que o grafo não tem.
3. **O Go não tinha `structuredContent`.** O MCP spec (2025-06-18) define `structuredContent` em `CallToolResult`; o `CallToolResult` Go (`internal/mcp/protocol.go:29-32`) só tem `content` e `isError`. A SPA precisa consumir dado estruturado, mas **precisa funcionar com um MCP que não o tem** — durante a transição, e porque a mudança do lado Go é aditiva e independente.
4. **Não há gate de frontend.** O job `frontend` do CI morreu com o viewer Electron em `b477200`. Nada garante que `npm run build` passa.

## Goals

- [ ] As quatro rotas — `/` (Overview), `/graph`, `/notes`, `/drift` — renderizam dados reais do MCP
- [ ] A navegação é por URL, com deep link preservado no histórico
- [ ] O pré-processador de wikilinks espelha a gramática de `internal/parser/wikilinks.go`, com **tabela de casos espelhada** em teste
- [ ] Toda resposta do MCP é validada na fronteira com `zod/mini`, e a ausência de `structuredContent` **não** quebra a tela
- [ ] Estado de servidor no Query, estado de cliente no Zustand, sem mistura
- [ ] A viewer tem gate de frontend: `typecheck`, `test`, `build` e `lint` verdes
- [ ] O bundle de produção fica próximo da meta de ~139 kB gzip registrada na ADR-060

## Out of Scope

| Feature | Reason |
|---|---|
| **A rota de fallback SPA no servidor Go** | É a emenda da ADR-060 à ADR-059, e pertence à spec `mcp-static-serving` (`SERV-23..SERV-25`). Aqui a SPA só consome o contrato |
| **A tool `memory_visualize_graph` devolver URL servida** | Lacuna real da ADR-059 (D-5), do lado Go. Aqui a viewer **abre** a URL retornada; se a URL não vier, mostra o caminho |
| **Edição de notas no vault** | ADR-059: a viewer é painel de métricas, não editor. Nenhuma tool de escrita é consumida |
| **Resolver vault federado (`memory://`)** | A viewer não indexa vault externo; `[[memory://...]]` é texto literal |
| **Renderizar o grafo dentro da SPA** | A ADR-060 mantém o renderer em `internal/graphview`. A viewer **invoca** e **abre** o artefato |
| **Autenticação / TLS do transporte** | Lacuna pré-existente, registrada na ADR-021 e na ADR-050 (D-7) |
| **Job `frontend` no CI** | Dívida de pipeline, fora do diretório `viewer/`. Esta spec entrega o gate local; o job é task do repo |
| **Mudar o Go** | `internal/**` e `cmd/**` não são escopo desta feature |

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | --------------- | --------- | ---------- |
| Fonte de dados | **Exclusivamente o MCP** (`POST {base}/mcp` + `GET {base}/health`), mesma origem | ADR-059 D-11: o MCP é a única fonte; nenhum backend, nenhuma API, nenhum banco no frontend | y |
| Base URL do MCP | `VITE_MCP_BASE_URL` quando definido; **string vazia** (mesma origem) por padrão | Em produção a SPA é servida pelo próprio MCP, então `/mcp` relativo funciona. Em dev o proxy do Vite encaminha para `http://127.0.0.1:38400`. A porta **não** é hardcoded no código | y |
| `structuredContent` ausente | Cai no **renderizador de texto**: mostra o `content` Markdown como está | O `structuredContent` está sendo adicionado em paralelo do lado Go, de forma aditiva. A viewer precisa ser útil antes e depois | y |
| `structuredContent` inválido | Mostra o texto cru, exibe um aviso discreto e registra o erro no `console` — **não** quebra a tela | Divergência de contrato não pode virar tela branca; o painel ainda mostra o que o MCP devolveu | y |
| Estado de servidor vs de cliente | **Query** para dado do MCP; **Zustand** para filtro, seleção, endpoints do path e nó do impacto | ADR-060: a separação é o que a ADR registra. Misturar os dois é o que produz cache inconsistente | y |
| Gramática de wikilink | A do parser Go (`internal/parser/wikilinks.go`), **não** uma segunda gramática | ADR-060 DR-5: o Go é a fonte da verdade. A duplicação só é segura com a tabela de casos espelhada | y |
| Link de nota | `/notes?path=<path percent-encoded>` | Deep link da ADR-060; o `memory_read_note` resolve por path ou por basename | y |
| Relação `rel:` | Não é exibida; fica no `title` do link | A relação é metadado de grafo; o painel já mostra topologia por outro lado | y |
| `[[memory://vault/path]]` | **Texto literal**, sem link | A viewer não resolve vault federado | y |
| Embed `![[X]]` | Link para X (sem `<img>`) | O MCP serve `app/` e `graph/`, não os binários do vault: uma `<img>` apontando para o vault seria 404 garantido | y |
| Validação | `zod/mini`, **nunca** o barrel `zod` | ADR-060 D-2: `zod` = 92,9 kB gzip, `zod/mini` = 3,0 kB. Os dois no bundle invalidariam a meta | y |
| Roteamento | Arquivo, com `routeTree.gen.ts` **commitado** | ADR-060 D-5: o clone não pode depender de rodar o dev server para ter as rotas | y |
| Idioma | Português do Brasil, no código e nos rótulos | O produto inteiro (CLI, tools, docs) é em pt-BR | y |

**Open questions:** none.

---

## User Stories

### P1: Painel de métricas navegável ⭐ MVP

**User Story**: Como pessoa que indexou um vault, quero abrir a viewer e ver a saúde e a topologia do meu grafo sem digitar comando, para que eu saiba o estado do repositório de relance.

**Why P1**: É o motivo de a viewer existir. Sem as rotas renderizando dado real, tudo o mais é enfeite.

**Acceptance Criteria**

1. The `/` route SHALL render the health payload returned by `GET /health` (status, name, version, protocol_version, tools_count, active_sessions, uptime_seconds)
2. The `/` route SHALL render the `memory_doctor` report (health_score, dead_links, orphan_notes)
3. The `/graph` route SHALL render the `memory_get_hubs` result for the `degree` algorithm
4. The `/graph` route SHALL render the `memory_get_hubs` result for the `pagerank` algorithm
5. The `/graph` route SHALL render the `memory_get_clusters` result (communities, modularity, total_nodes, total_edges)
6. The `/graph` route SHALL render the `memory_get_insights` result (surprising connections)
7. WHEN the user activates "Abrir grafo interativo" THEN the SPA SHALL call `memory_visualize_graph` and open the returned URL in a new tab
8. The `/notes` route SHALL render the `memory_list_files` result filtered by the extension filter and by the folder filter
9. WHEN a note is selected THEN the `/notes` route SHALL render the `memory_read_note` content, the `memory_note_outline` structure and the `memory_file_metadata` values
10. The `/drift` route SHALL render the `memory_get_drift` result (drifted notes, uncovered code, overall score)
11. The `/drift` route SHALL render the `memory_get_impact` result for the node informed in the store
12. The `/drift` route SHALL render the `memory_find_path` result between the origin node and the destination node informed in the store
13. The selected note, the extension filter, the folder filter, the path origin node, the path destination node and the impact target node SHALL live in a Zustand store
14. The server data SHALL live in the TanStack Query cache and SHALL NOT be duplicated in the Zustand store
15. The route navigation SHALL update the URL so that the current view is a deep link that survives a reload

**Independent Test**: com um MCP local no ar, abrir `/`, `/graph`, `/notes` e `/drift` e conferir que cada painel mostra o valor real da tool correspondente.

---

### P2: Wikilink navegável

**User Story**: Como pessoa que lê uma nota na viewer, quero que os `[[wikilinks]]` virem link clicável, para que eu navegue pelo grafo a partir do texto.

**Why P2**: A ADR-060 Deduz que o painel mostra o **conteúdo real** da nota, não um resumo. Sem wikilink clicável, a nota renderizada é uma foto morta.

**Acceptance Criteria**

1. The preprocessor SHALL convert `[[X]]` into a link labeled "X"
2. The preprocessor SHALL convert `[[X|Label]]` into a link labeled "Label"
3. The preprocessor SHALL convert `[[X#Section]]` into a link to "X" carrying the anchor "Section"
4. The preprocessor SHALL convert `[[X#Section|Label]]` into a link labeled "Label" carrying the anchor "Section"
5. The preprocessor SHALL convert `[[X#^c182]]` into a link to "X" carrying the block "c182"
6. The preprocessor SHALL convert `[[#Section]]` into a local anchor that does not navigate away from the document
7. The preprocessor SHALL convert `![[X]]` into a link to X
8. The preprocessor SHALL keep `[[memory://vault/path]]` as literal text
9. The preprocessor SHALL label `[[X|rel:implements]]` as "X" and carry the relation in the link title
10. The preprocessor SHALL NOT create a link inside a fenced code block
11. The preprocessor SHALL NOT create a link inside inline code
12. The preprocessor SHALL NOT create a link inside an HTML comment
13. The preprocessor SHALL NOT create a link inside a Markdown table row
14. The destination of a note link SHALL be the SPA route `/notes` with the percent-encoded path in the `path` query parameter
15. A unit test SHALL cover the nine link forms and the four no-link contexts, mirroring the case table of the Go parser
16. The note content SHALL be rendered by the TanStack Markdown renderer after the preprocessor runs

**Independent Test**: rodar a suíte de `viewer` e conferir que a tabela de casos do pré-processador espelha a de `internal/parser/wikilinks.go`.

---

### P3: Contrato MCP tolerante e gate de frontend

**User Story**: Como pessoa que roda a viewer contra um MCP antigo ou contra o MCP novo, quero que o painel funcione nos dois casos, para que a transição não quebre o produto.

**Why P3**: A integração com o lado Go é o **maior risco** desta feature: o `structuredContent` está sendo implementado em paralelo, e a divergência de shape entre o que a SPA assume e o que o Go devolve é o que quebra a tela.

**Acceptance Criteria**

1. The MCP client SHALL send a JSON-RPC 2.0 `tools/call` request to `POST {base}/mcp`
2. The MCP client SHALL return the `content` array of the `tools/call` result
3. IF the result carries `structuredContent` AND the payload validates against the tool schema THEN the client SHALL return the parsed payload
4. IF the result does not carry `structuredContent` THEN the client SHALL return the raw text and the caller SHALL render it as preformatted text
5. IF the `structuredContent` payload does not validate against the tool schema THEN the client SHALL return the raw text, SHALL log the validation error to the console and SHALL NOT throw
6. The health payload SHALL be validated against a schema before being rendered
7. Each tool consumed by the panel SHALL have its own schema declared in `viewer/src/lib/schemas.ts`
8. The `zod` package SHALL be imported through the `zod/mini` subpath and never through the `zod` barrel
9. IF the MCP is unreachable or the tool returns an error THEN the route SHALL render an error state instead of a blank screen
10. The `npm run typecheck` command SHALL exit 0
11. The `npm run test` command SHALL exit 0
12. The `npm run build` command SHALL exit 0
13. The `npm run lint` command SHALL exit 0
14. The production bundle SHALL NOT exceed 160 kB gzip
15. The dev server SHALL proxy the `/mcp` and `/health` paths to the MCP configured by `VITE_MCP_BASE_URL`
16. The generated `routeTree.gen.ts` SHALL be committed to the repository and SHALL NOT be ignored by `.gitignore`

**Independent Test**: rodar os quatro gates em `viewer/` e, com o `fetch` mockado, confirmar que a ausência de `structuredContent` não quebra o cliente.

---

## Edge Cases

- IF the MCP is down THEN the panel shows the error state of the affected route, and the other routes keep working
- IF `memory_doctor` returns zero dead links and zero orphans THEN the Overview shows the empty state instead of an empty table
- IF `memory_list_files` returns no document for the current filters THEN the notes list shows an empty state, and no note is pre-selected
- IF the selected note is a basename that matches more than one document THEN the MCP error is surfaced, and the viewer does not silently pick one
- IF a note contains a wikilink to a note that does not exist THEN the link is still rendered, pointing at the notes route, where the MCP will report the miss
- IF the wikilink target is a path with spaces or accents THEN the link target SHALL be percent-encoded
- IF the wikilink appears inside a fenced code block, inline code, an HTML comment or a table row THEN the text stays literal
- IF `memory_visualize_graph` returns no URL THEN the viewer shows the returned path as text and does not open a tab
- IF the structured payload has a field the schema marks as optional THEN the missing field SHALL NOT break the render
- IF the structured payload has a field of the wrong type THEN the viewer falls back to the raw text and logs the validation error
- IF the bundle grows past 160 kB gzip THEN the cause SHALL be investigated, and the most likely one is an import of the `zod` barrel instead of `zod/mini`

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----------- | ------ | ------- |
| VD-01 | P1: Painel de métricas | Execute | Pending |
| VD-02 | P1: Painel de métricas | Execute | Pending |
| VD-03 | P1: Painel de métricas | Execute | Pending |
| VD-04 | P1: Painel de métricas | Execute | Pending |
| VD-05 | P1: Painel de métricas | Execute | Pending |
| VD-06 | P1: Painel de métricas | Execute | Pending |
| VD-07 | P1: Painel de métricas | Execute | Pending |
| VD-08 | P1: Painel de métricas | Execute | Pending |
| VD-09 | P1: Painel de métricas | Execute | Pending |
| VD-10 | P1: Painel de métricas | Execute | Pending |
| VD-11 | P1: Painel de métricas | Execute | Pending |
| VD-12 | P1: Painel de métricas | Execute | Pending |
| VD-13 | P1: Painel de métricas | Execute | Pending |
| VD-14 | P1: Painel de métricas | Execute | Pending |
| VD-15 | P1: Painel de métricas | Execute | Pending |
| VD-16 | P2: Wikilink navegável | Execute | Pending |
| VD-17 | P2: Wikilink navegável | Execute | Pending |
| VD-18 | P2: Wikilink navegável | Execute | Pending |
| VD-19 | P2: Wikilink navegável | Execute | Pending |
| VD-20 | P2: Wikilink navegável | Execute | Pending |
| VD-21 | P2: Wikilink navegável | Execute | Pending |
| VD-22 | P2: Wikilink navegável | Execute | Pending |
| VD-23 | P2: Wikilink navegável | Execute | Pending |
| VD-24 | P2: Wikilink navegável | Execute | Pending |
| VD-25 | P2: Wikilink navegável | Execute | Pending |
| VD-26 | P2: Wikilink navegável | Execute | Pending |
| VD-27 | P2: Wikilink navegável | Execute | Pending |
| VD-28 | P2: Wikilink navegável | Execute | Pending |
| VD-29 | P2: Wikilink navegável | Execute | Pending |
| VD-30 | P2: Wikilink navegável | Execute | Pending |
| VD-31 | P2: Wikilink navegável | Execute | Pending |
| VD-32 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-33 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-34 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-35 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-36 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-37 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-38 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-39 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-40 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-41 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-42 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-43 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-44 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-45 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-46 | P3: Contrato MCP tolerante | Execute | Pending |
| VD-47 | P3: Contrato MCP tolerante | Execute | Pending |

**Coverage:** 47 total, 47 mapped to stories, 0 unmapped.

---

## Success Criteria

- [ ] `npm run typecheck`, `npm run test`, `npm run build` e `npm run lint` passam em `viewer/`
- [ ] As quatro rotas renderizam dado real do MCP e a navegação por URL funciona
- [ ] O pré-processador de wikilinks tem a tabela de casos completa, incluindo os quatro contextos sem link
- [ ] Os schemas `zod` validam e rejeitam corretamente, e a SPA não quebra quando `structuredContent` está ausente
- [ ] `routeTree.gen.ts` existe e não está no `.gitignore`
- [ ] O bundle final fica próximo de 139 kB gzip
- [ ] Nenhum arquivo fora de `viewer/` e `.specs/features/viewer-dashboard/` foi modificado
