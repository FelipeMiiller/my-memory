# Painel de métricas da viewer — Tasks

## Execution Protocol (MANDATORY -- do not skip)

- Execution is **strictly sequential** — there is no intra-phase parallelism. The chain is a chain: each task consumes the artifacts of the previous one.
- Every task must satisfy its **Done when** checklist and its **Gate** before the next task starts. A task that fails its gate is fixed immediately; it is never carried forward.
- After every task, run the gate command of the task and record the result in the task's checklist. A task is not "done" because the code compiles; it is done when the gate is green.
- The frontend gate lives in `viewer/`: `npm run typecheck`, `npm run test`, `npm run build`, `npm run lint`. The Go gate (`go build ./... && go vet ./...`) must stay green too — this feature does not touch Go, so a Go failure belongs to another worker and is **reported, not fixed** here.
- **No commit is made by the worker.** The orchestrator audits and commits. `git add -A`, `git add .` and `git stash` are forbidden.
- Scope boundary: only `viewer/**` and `.specs/features/viewer-dashboard/**` are writable. `internal/**`, `cmd/**`, `docs/**`, `CHANGELOG.md`, `README.md`, `AGENTS.md` and the root `.gitignore` are out of scope.

## Test Coverage Matrix

| Layer | Test type | File | Status |
| ----- | --------- | ---- | ------ |
| Toolchain e dependências | none | `viewer/package.json` | none — não há comportamento a testar; o gate de T11 cobre a configuração |
| Schemas zod do contrato MCP | unit | `viewer/src/lib/schemas.test.ts` | pronto |
| Cliente MCP (JSON-RPC, fallback) | unit | `viewer/src/lib/mcp-client.test.ts` | pronto |
| Pré-processador de wikilinks | unit | `viewer/src/lib/wikilinks.test.ts` | pronto |
| Store Zustand | unit | `viewer/src/store/viewer-store.test.ts` | pronto |
| Shell, rotas e navegação | integration | smoke test do bundle em `viewer/dist/` | coberto por T11 |
| Verificação final e versionamento | none | `viewer/src/routes/routeTree.gen.ts` | none — o gate de T11 é a verificação |

## Gate Check Commands

```bash
# Frontend (dentro de viewer/)
cd viewer
npm run typecheck
npm run test
npm run build
npm run lint

# Go: não pode quebrar (não é escopo desta feature)
cd ..
go build ./... && go vet ./...
```

## Execution Plan

### Phase 1: Toolchain e contrato de dados

```
T1 → T2 → T3
```

### Phase 2: Estado de cliente e pré-processador de wikilinks

```
T4 → T5
```

### Phase 3: As quatro rotas da viewer

```
T6 → T7 → T8 → T9 → T10
```

### Phase 4: Verificação e versionamento

```
T11
```

---

## Task Breakdown

### T1: Instalar o stack da ADR-060 e criar o gate de frontend

**What**: Adicionar ao `package.json` o stack decidido na ADR-060 — `@tanstack/react-query@5.104.4`, `@tanstack/react-router@1.170.45`, `zustand@5.0.15`, `zod@4.6.7`, `@tanstack/markdown@1.0.0`; em dev, `@tanstack/router-plugin@1.168.47`, `@tanstack/react-query-devtools@5.104.4`, `@tanstack/react-router-devtools@1.167.2`, `@tanstack/vite-config@0.6.0` e `vitest`. **Não** instalar `@tanstack/config`: está deprecado no npm ("Package no longer supported"). Adicionar o script `test` (`vitest run`). Configurar o plugin de rotas (`tanstackRouter`) com `generatedRouteTree` dentro de `src/routes/` e `routeFileIgnorePattern` para que o arquivo gerado não vire rota fantasma. Configurar o proxy de `/mcp` e `/health` no dev server, com o destino vindo de `VITE_MCP_BASE_URL` (default `http://127.0.0.1:38400`).
**Where**: `viewer/package.json`
**Depends on**: None
**Reuses**: o alias `@` já existente em `viewer/vite.config.ts`; o plugin React e o Tailwind já instalados
**Requirement**: VD-44, VD-45, VD-46

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] O `package.json` lista as versões exatas do stack da ADR-060
- [ ] `@tanstack/config` não está no `package.json`
- [ ] `npm run test` existe e roda o vitest
- [ ] O dev server encaminha `/mcp` e `/health` para `VITE_MCP_BASE_URL`
- [ ] Nenhuma porta está hardcoded no código da aplicação
- [ ] Gate check passa: `cd viewer && npm install && npm run test`

**Tests**: none
**Gate**: full

**Commit**: `chore(viewer): instalar o stack da ADR-060 e criar o gate de frontend`

---

### T2: Declarar os schemas zod do contrato MCP

**What**: Um schema por tool em `viewer/src/lib/schemas.ts`, espelhando os JSON tags dos tipos Go (`mcp.DoctorReport`, `mcp.GodNode`, `mcp.PageRankNode`, `graph.CommunityResult`, `mcp.SurprisingConnection`, `store.VaultDocument`, `mcp.ReadNoteResult`, `mcp.OutlineHeading`, `mcp.FileMetadataResult`, `drift.DriftReport`, `graph.ImpactResult`, `graph.PathResult`, `mcp.VisualizeGraphResult`) mais o payload de `/health`. Importar **apenas** `zod/mini`. Marcar com `.optional()` os campos que o Go omite ou que podem faltar.
**Where**: `viewer/src/lib/schemas.ts`
**Depends on**: T1
**Reuses**: os JSON tags de `internal/mcp/handlers.go`, `internal/mcp/vault_handlers.go`, `internal/mcp/cluster_handlers.go`, `internal/mcp/drift_handlers.go`, `internal/mcp/visualizer_handlers.go`, `internal/graph/community.go`, `internal/graph/impact.go`, `internal/graph/path.go`, `internal/drift/drift.go`
**Requirement**: VD-36, VD-37, VD-38

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] Existe um schema por tool consumida pela viewer
- [ ] O import é `zod/mini`, e nenhum arquivo importa o barrel `zod`
- [ ] O shape de cada schema está comentado em cima dele, com o tipo Go de origem
- [ ] Um teste valida um payload válido e rejeita um payload inválido de cada schema
- [ ] Gate check passa: `cd viewer && npm run test && npm run typecheck`

**Tests**: unit
**Gate**: full

**Commit**: `feat(viewer): declarar os schemas zod do contrato MCP`

---

### T3: Implementar o cliente MCP com fallback de texto

**What**: Cliente JSON-RPC 2.0 em `viewer/src/lib/mcp-client.ts`. `POST {base}/mcp` com `tools/call`, e `GET {base}/health`. A base vem de `VITE_MCP_BASE_URL`; vazio significa mesma origem. O resultado carrega o texto do `content`, o `structuredContent` validado, e um booleano `hasStructured`. Se o `structuredContent` faltar (MCP antigo), devolve só o texto. Se falhar a validação, devolve o texto cru, registra o erro no `console` e **não** lança.
**Where**: `viewer/src/lib/mcp-client.ts`
**Depends on**: T2
**Reuses**: os schemas de T2; o formato de `mcp.CallToolResult` (`internal/mcp/handlers.go:29-36`)
**Requirement**: VD-32, VD-33, VD-34, VD-35, VD-36

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] A requisição tem `jsonrpc`, `id`, `method: "tools/call"` e `params: {name, arguments}`
- [ ] O `content` da resposta volta como texto
- [ ] O `structuredContent` volta quando existe e valida
- [ ] A ausência de `structuredContent` não quebra: o texto cru é devolvido
- [ ] Um `structuredContent` inválido não quebra: texto cru + erro no `console`
- [ ] Um erro JSON-RPC ou `isError` vira exceção, e o MCP fora do ar vira erro legível
- [ ] Gate check passa: `cd viewer && npm run test`

**Tests**: unit
**Gate**: full

**Commit**: `feat(viewer): implementar o cliente MCP com fallback de texto`

---

### T4: Implementar o store Zustand de estado de cliente

**What**: Store em `viewer/src/store/viewer-store.ts` com a nota selecionada, o filtro de extensão, o filtro de pasta, o nó de origem e o de destino do caminho, e o nó alvo do impacto — mais as ações que mudam cada um. O dado do servidor **não** entra aqui: ele vive no cache do TanStack Query.
**Where**: `viewer/src/store/viewer-store.ts`
**Depends on**: T3
**Reuses**: o padrão de `zustand` (`create`) e a separação registrada na ADR-060
**Requirement**: VD-13, VD-14

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] A nota selecionada, os dois filtros, os dois nós do caminho e o nó do impacto vivem no store
- [ ] Cada ação muda exatamente o campo esperado, e as demais ficam intactos
- [ ] Nenhum dado vindo do MCP é guardado no store
- [ ] Gate check passa: `cd viewer && npm run test && npm run typecheck`

**Tests**: unit
**Gate**: full

**Commit**: `feat(viewer): implementar o store Zustand de estado de cliente`

---

### T5: Implementar o pré-processador de wikilinks

**What**: Módulo `viewer/src/lib/wikilinks.ts` que espelha a gramática de `internal/parser/wikilinks.go`: a regex `(!)?\[\[([^\]\r\n]+)\]\]`, a dissecação de `ParseWikilink` (alias, âncora, block id, relação por prefixo e por alias, URI federada) e as quatro Strippagens de `stripCodeBlocks` (bloco cercado, código inline, comentário HTML, linha de tabela). `@tanstack/markdown` não conhece wikilinks, então a conversão para link Markdown acontece **antes** do render. O destino do link é `/notes?path=<path percent-encoded>`, com `anchor` e `block` opcionais; `[[#Secao]]` vira âncora local; `[[memory://...]]` fica como texto literal; a relação vai para o `title`, nunca para o rótulo.
**Where**: `viewer/src/lib/wikilinks.ts`
**Depends on**: T4
**Reuses**: a regex e a dissecação de `internal/parser/wikilinks.go`; o contrato de destino definido em `viewer/src/lib/note-search.ts`
**Requirement**: VD-16, VD-17, VD-18, VD-19, VD-20, VD-21, VD-22, VD-23, VD-24, VD-25, VD-26, VD-27, VD-28, VD-29, VD-30, VD-31

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] A tabela de casos do teste espelha a do Go: link simples, alias, âncora, âncora+alias, block id, link interno, embed, federado, relação no alias, e os quatro contextos sem link
- [ ] Wikilink em bloco cercado, código inline, comentário HTML ou linha de tabela permanece texto literal
- [ ] O frontmatter é preservado intacto
- [ ] O path do destino é percent-encoded
- [ ] Gate check passa: `cd viewer && npm run test`

**Tests**: unit
**Gate**: full

**Commit**: `feat(viewer): implementar o pré-processador de wikilinks`

---

### T6: Montar o shell da SPA com router, providers e tratamento de erro

**What**: `viewer/src/routes/__root.tsx` com o layout (header de navegação, `QueryClientProvider`, `ThemeProvider`, `<Outlet/>`), `errorComponent` por rota e `notFoundComponent`. `viewer/src/router.tsx` cria o `createRouter` sobre a árvore gerada. `viewer/src/main.tsx` monta o `RouterProvider`. Os devtools entram por import dinâmico, para não pesar no bundle de produção. A navegação é por URL: cada tela é um deep link.
**Where**: `viewer/src/routes/__root.tsx`
**Depends on**: T5
**Reuses**: o `ThemeProvider` já existente em `viewer/src/components/theme-provider.tsx`; o `Button` do shadcn
**Requirement**: VD-15, VD-39, VD-46

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] As quatro rotas aparecem na navegação do header, e a rota ativa fica destacada
- [ ] `QueryClientProvider` e o tema estão no layout raiz
- [ ] Cada rota tem tratamento de erro próprio, e a raiz tem `notFoundComponent`
- [ ] Os devtools só carregam em desenvolvimento
- [ ] Gate check passa: `cd viewer && npm run build && npm run lint`

**Tests**: integration
**Gate**: full

**Commit**: `feat(viewer): montar o shell da SPA com router e providers`

---

### T7: Rota Overview (`/`)

**What**: `viewer/src/routes/index.tsx`. Consome `GET /health` (status, nome, versão, `tools_count`, `active_sessions`, uptime) e `memory_doctor` (health score, dead links, notas órfãs). Quando o MCP está fora do ar, a rota mostra o estado de erro em vez de tela branca.
**Where**: `viewer/src/routes/index.tsx`
**Depends on**: T6
**Reuses**: os hooks de `viewer/src/lib/queries.ts`; o `ToolPayload` de `viewer/src/components/tool-payload.tsx`
**Requirement**: VD-01, VD-02

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] A rota mostra o health do servidor e o relatório do doctor
- [ ] O MCP fora do ar produz estado de erro, não tela em branco
- [ ] Sem `structuredContent`, o texto cru do MCP é exibido
- [ ] Gate check passa: `cd viewer && npm run build && npm run lint`

**Tests**: integration
**Gate**: full

**Commit**: `feat(viewer): rota Overview com health e diagnóstico`

---

### T8: Rota Grafo (`/graph`)

**What**: `viewer/src/routes/graph.tsx`. Consome `memory_get_hubs` (degree e pagerank), `memory_get_clusters` (comunidades e modularidade) e `memory_get_insights` (conexões latentes). O botão "Abrir grafo interativo" chama `memory_visualize_graph` e abre a URL devolvida em nova aba — a URL servida, nunca um caminho local.
**Where**: `viewer/src/routes/graph.tsx`
**Depends on**: T7
**Reuses**: os hooks de `viewer/src/lib/queries.ts`; a `mutation` de `memory_visualize_graph`
**Requirement**: VD-03, VD-04, VD-05, VD-06, VD-07

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] Hubs de grau e de PageRank são renderizados em tabelas distintas
- [ ] As comunidades, a modularidade e as conexões latentes são renderizadas
- [ ] O botão abre a URL retornada por `memory_visualize_graph` em nova aba
- [ ] Sem URL no resultado, o caminho é exibido como texto e nenhuma aba é aberta
- [ ] Gate check passa: `cd viewer && npm run build && npm run lint`

**Tests**: integration
**Gate**: full

**Commit**: `feat(viewer): rota Grafo com hubs, clusters e insights`

---

### T9: Rota Notas (`/notes`)

**What**: `viewer/src/routes/notes.tsx`. Lista com `memory_list_files` (filtro de extensão e de pasta, vindos do store) e, para a nota selecionada, `memory_read_note` (renderizado com TanStack Markdown depois do pré-processador), `memory_note_outline` e `memory_file_metadata`. O path da nota vive na URL (`/notes?path=...`), para que a tela seja um deep link.
**Where**: `viewer/src/routes/notes.tsx`
**Depends on**: T8
**Reuses**: o pré-processador de T5; o store de T4; `viewer/src/components/note-markdown.tsx`
**Requirement**: VD-08, VD-09, VD-16, VD-31

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] A lista respeita o filtro de extensão e o de pasta
- [ ] A nota selecionada mostra conteúdo, outline e metadados
- [ ] O conteúdo renderizado tem os wikilinks clicáveis, apontando para `/notes?path=...`
- [ ] Selecionar uma nota atualiza a URL, e abrir a URL seleciona a nota
- [ ] Gate check passa: `cd viewer && npm run build && npm run lint`

**Tests**: integration
**Gate**: full

**Commit**: `feat(viewer): rota Notas com listagem e leitura de nota`

---

### T10: Rota Drift (`/drift`)

**What**: `viewer/src/routes/drift.tsx`. Consome `memory_get_drift` (notas defasadas, código órfão, score geral), `memory_get_impact` (blast radius do nó informado no store) e `memory_find_path` (caminho entre origem e destino, ambos no store). As consultas de impacto e de caminho só rodam quando o usuário preencheu oRespectivo campo.
**Where**: `viewer/src/routes/drift.tsx`
**Depends on**: T9
**Reuses**: os hooks de `viewer/src/lib/queries.ts`; o store de T4
**Requirement**: VD-10, VD-11, VD-12

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] O relatório de drift mostra score, severidades, notas defasadas e código órfão
- [ ] O blast radius só é consultado quando há nó alvo, e mostra risco e nós afetados
- [ ] O caminho só é consultado quando há origem e destino, e mostra os segmentos
- [ ] Gate check passa: `cd viewer && npm run build && npm run lint`

**Tests**: integration
**Gate**: full

**Commit**: `feat(viewer): rota Drift com desvio, impacto e caminho`

---

### T11: Gate de frontend completo e versionamento da árvore de rotas

**What**: Rodar os quatro gates em `viewer/` e confirmar que `viewer/src/routes/routeTree.gen.ts` existe e **não** está no `.gitignore`. Conferir o tamanho final do bundle contra o baseline de 84,28 kB gzip e a meta de ~139 kB gzip da ADR-060. Se o bundle estiver muito acima, investigar a causa — a mais provável é um import do barrel `zod` em vez de `zod/mini`. Confirmar também que o gate do Go segue verde.
**Where**: `viewer/`
**Depends on**: T10
**Reuses**: o proxy e o gate montados em T1
**Requirement**: VD-40, VD-41, VD-42, VD-43, VD-44, VD-45, VD-46, VD-47

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] `npm run typecheck` sai com 0
- [ ] `npm run test` sai com 0, com a tabela de casos do pré-processador verde
- [ ] `npm run build` sai com 0
- [ ] `npm run lint` sai com 0
- [ ] `routeTree.gen.ts` existe em `viewer/src/routes/` e `git check-ignore` não o encontra
- [ ] O tamanho final do bundle foi medido e registrado
- [ ] `go build ./... && go vet ./...` segue verde
- [ ] Gate check passa: `cd viewer && npm run typecheck && npm run test && npm run build && npm run lint`

**Tests**: none
**Gate**: full

**Commit**: `chore(viewer): gate de frontend e versionamento da árvore de rotas`

---

## Phase Execution Map

```
Phase 1 → Phase 2 → Phase 3 → Phase 4

T1 → T2 → T3 → T4 → T5 → T6 → T7 → T8 → T9 → T10 → T11
```

O validador conta as arestas do diagrama como declaradas: **10 setas, 11 tarefas, uma raiz** (T1).

## Task Granularity Check

| Task | `Where` | Veredito |
| ---- | ------- | -------- |
| T1 | `viewer/package.json` | 1 arquivo primário; `vite.config.ts` e `vitest.config.ts` são adjusting do mesmo gate de toolchain, described no "What" |
| T2 | `viewer/src/lib/schemas.ts` | 1 arquivo |
| T3 | `viewer/src/lib/mcp-client.ts` | 1 arquivo |
| T4 | `viewer/src/store/viewer-store.ts` | 1 arquivo |
| T5 | `viewer/src/lib/wikilinks.ts` | 1 arquivo |
| T6 | `viewer/src/routes/__root.tsx` | 1 arquivo primário; `router.tsx` e `main.tsx` são o wiring do mesmo shell |
| T7 | `viewer/src/routes/index.tsx` | 1 arquivo |
| T8 | `viewer/src/routes/graph.tsx` | 1 arquivo |
| T9 | `viewer/src/routes/notes.tsx` | 1 arquivo |
| T10 | `viewer/src/routes/drift.tsx` | 1 arquivo |
| T11 | `viewer/` | gate de feature, não unidade de código |

Nenhuma task toca dois domínios de comportamento. As tasks de rota são separadas justamente porque cada uma consome um conjunto diferente de tools MCP.

## Diagram-Definition Cross-Check

| Aresta | `Depends on` correspondente | Fase |
| ------ | -------------------------- | ---- |
| T1 → T2 | T2: `Depends on: T1` | 1 |
| T2 → T3 | T3: `Depends on: T2` | 1 |
| T3 → T4 | T4: `Depends on: T3` | 2 |
| T4 → T5 | T5: `Depends on: T4` | 2 |
| T5 → T6 | T6: `Depends on: T5` | 3 |
| T6 → T7 | T7: `Depends on: T6` | 3 |
| T7 → T8 | T8: `Depends on: T7` | 3 |
| T8 → T9 | T9: `Depends on: T8` | 3 |
| T9 → T10 | T10: `Depends on: T9` | 3 |
| T10 → T11 | T11: `Depends on: T10` | 4 |

As arestas do diagrama e as declarações `Depends on` são o mesmo conjunto: 10 arestas, nenhum lado sem par.

## Test Co-location Validation

| Task | Tipo | Teste mora onde? |
| ---- | ---- | ----------------- |
| T1 | none | sem comportamento testável |
| T2 | unit | `viewer/src/lib/schemas.test.ts`, ao lado do `schemas.ts` |
| T3 | unit | `viewer/src/lib/mcp-client.test.ts`, ao lado do `mcp-client.ts` |
| T4 | unit | `viewer/src/store/viewer-store.test.ts`, ao lado do `viewer-store.ts` |
| T5 | unit | `viewer/src/lib/wikilinks.test.ts`, ao lado do `wikilinks.ts` |
| T6–T10 | integration | smoke test do bundle em `viewer/dist/`, executado em T11 |
| T11 | none | é o próprio gate |

Nenhum teste mora em um diretório de testes separado: o padrão do repositório é o arquivo `*.test.ts` ao lado do módulo.

## MCPs and Skills

- MCP: NONE — a viewer **consome** o MCP, não é uma tool. Nenhuma tool nova é criada, e nenhuma tool de escrita é consumida: a viewer é painel de métricas, não editor (ADR-059).
- Skill: `my-memory` — para a semântica do produto (formatação de wikilink, taxonomia do vault).
- Skill: `tlc-spec-driven` — para o ciclo Specify → Design → Tasks → Execute desta própria spec.
