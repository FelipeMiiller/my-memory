# ADR-060: Stack do frontend da viewer (TanStack) e a rota de fallback SPA

- **Data**: 2026-10-19
- **Status**: Accepted
- **Decisores**: Felipe Miiller
- **Tags**: viewer, react, tanstack, router, query, zustand, zod, markdown, wikilinks, http, spa
- **Relacionada**: [ADR-059](059-spa-react-vite-em-viewer-servida-pelo-mcp.md), [ADR-058](058-renderizador-vis-network-vendorizado-sem-build-step.md), [ADR-054](054-escopo-de-paridade-com-o-obsidian.md), [ADR-050](050-threat-model-owasp-llm-aplicado-ao-mymemory.md), [ADR-020](020-visualizador-interativo-de-grafo-em-html-svg.md)

---

## Contexto e Problema

A [ADR-059](059-spa-react-vite-em-viewer-servida-pelo-mcp.md) decidiu **que** a viewer existe e **como** ela é servida: uma SPA React/Vite em `viewer/`, compilada para estático, servida pelo transporte HTTP do MCP sob um diretório dedicado, nunca a raiz do vault. A ADR-059 não decidiu **com o que** a SPA é construída além do scaffold do shadcn.

O estado atual é o starter padrão do shadcn: `viewer/src/App.tsx` (685 B) renderiza "Project ready!" e um botão. Ele não consome o MCP, não tem rota, não tem estado, e não renderiza uma única nota. A superfície de tools MCP disponível é de **24 tools** (`internal/mcp/tools.go`), incluindo `memory_get_hubs`, `memory_get_clusters`, `memory_get_impact`, `memory_doctor`, `memory_get_drift`, `memory_list_files`, `memory_read_note`, `memory_note_outline` e `memory_file_metadata` — todas com dados que um painel de métricas consome diretamente.

Duas decisões ficaram abertas por ausência de escolha, e ambas têm consequência de arquitetura:

1. **O roteamento.** A ADR-059 decidiu as rotas do servidor (`GET /` e `GET /graph/<arquivo>.html`). Uma SPA com rotas de cliente precisa que o servidor devolva `index.html` para qualquer path de navegação — caso contrário, recarregar em `/dashboard` devolve 404. Isso é uma **emenda ao contrato de rotas** da ADR-059, não um detalhe de frontend.
2. **O tratamento de wikilinks.** O vault é Obsidian Flavored Markdown, e no My-Memory o `[[wikilink]]` **é a aresta do grafo** (`internal/parser/wikilinks.go`). O renderer de Markdown escolhido não conhece wikilinks.

Este documento fecha as duas, e registra o stack do frontend com os custos medidos.

## Drivers da Decisão

- **DR-1 — Um binário só.** O produto é local. A viewer tem de ser servida pelo mesmo processo Go que serve o MCP, sem servidor de arquivos externo, sem proxy, sem segundo processo. Isso elimina stacks que exigem runtime de aplicação.
- **DR-2 — Zero CDN, zero origem remota.** A ADR-050 é explícita: "nenhuma imagem remota", sem script remoto. Todo byte do bundle tem de ser local e versionável.
- **DR-3 — O MCP é a única fonte de dados.** A SPA fala com `/mcp` (POST JSON-RPC) na mesma origem. Não há backend, não há API, não há banco no frontend.
- **DR-4 — Custo mensurável.** O bundle atual é 259,52 kB JS (84,28 kB gzip). Qualquer adição precisa justificável em bytes, não em preferência.
- **DR-5 — O Go é a fonte da verdade sobre wikilinks.** A gramática de `[[...]]` já está implementada e testada em `internal/parser/wikilinks.go`. O frontend não pode inventar uma segunda gramática sem custo de divergência explícito.
- **DR-6 — A viewer é painel de métricas, não editor.** A ADR-059 define esse limite. Ela lê e mostra; não escreve no vault.
- **DR-7 — Consistência com a superfície de segurança da ADR-059.** A CSP registrada na ADR-059 (`default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self'`) tem de continuar valendo. Isso implica: nenhum `<script>` inline no bundle gerado, nenhuma imagem remota, nenhuma chamada cross-origin.
- **DR-8 — Ferramenta usável antes do código.** O time de produto de um dashboard local é curto. Stack que exige codegen, plugin de rota e leitura de manual não é stack de painel de métricas.
- **DR-9 — O repositório tem duas toolchains desde a ADR-059.** Go e Node. A viewer precisa de um gate de frontend no CI que não existe (o job `frontend` foi removido em `b477200`).

## Opções Consideradas

### Opção A — Stack enxuto: Query + Zustand + zod/mini + Markdown, sem router

SPA de painel único, com abas internas controladas por estado. Sem roteamento de URL.

- ✅ Menor custo: **+23,7 kB gzip** sobre o baseline
- ✅ Zero codegen, zero `routeTree.gen.ts`, zero alteração no contrato de rotas do Go
- ✅ A URL da viewer é sempre a mesma — cola, favorita, abre
- ❌ Não há deep link: não dá para compartilhar "a visão do cluster X"
- ❌ O estado da tela vive no store, não na URL — recarregar perde o contexto
- ❌ Cresce mal se a viewer virar app de verdade

### Opção B — Stack completo: Query + Router + Zustand + zod/mini + Markdown + DevTools

Router de arquivo do TanStack, com fallback SPA no servidor.

- ✅ Deep link por URL, navegação por histórico, estado serializável na URL
- ✅ O único roteador com type-safety ponta a ponta e carregamento de rota por código
- ✅ DevTools do Query e do Router dão introspecção de cache e de rota
- ❌ **+54,7 kB gzip** e um plugin de codegen (`routeTree.gen.ts` commitado)
- ❌ Exige **emendar a ADR-059** com a rota de fallback
- ❌ `@tanstack/router-plugin` usa chokidar + babel: mais uma superfície de build

### Opção C — Sem TanStack: React + fetch + useState, router só se necessário

O mínimo absoluto, sem nenhuma das libs listadas.

- ✅ Zero dependência nova, zero bytes
- ✅ Nenhuma decisão de stack para revisar
- ❌ Sem cache, sem dedup, sem estados de erro prontos — cada painel reimplementa
- ❌ Validação de resposta do MCP na mão, em cada call site
- ❌ O painel precisa de polling manual para `/health` e para as tools de grafo

### Opção D — Trazer de volta o viewer Electron

O caminho de `b477200`.

- ✅ Aplicação desktop de verdade, filesystem e editor
- ❌ 112 arquivos e 27.858 linhas removidos em `b477200`; a ADR-054 declara plugins e `dev:*` fora de escopo
- ❌ Contraria a ADR-059 (que é sobre servir, não empacotar) e o driver DR-1
- ❌ Instala Electron, que é exatamente o peso que o repo se livrou

## Resultado da Decisão

**Adotamos a Opção B**, com a ressalva registrada em D-3 sobre o pacote de config.

### O stack

Versões e custos medidos em 2026-10-19 (registro npm e `bundlephobia`; `zod` medido localmente com esbuild).

| Pacote | Versão | gzip | Dependências | Papel |
|---|---|---|---|---|
| `@tanstack/react-query` | 5.104.4 | 13,2 kB | `@tanstack/query-core` (94,75 kB raw) | Estado **de servidor**: cache, dedup, refetch, estados de erro |
| `@tanstack/react-router` | 1.170.45 | 30,98 kB | `router-core` (143,5 kB), `history`, `react-store`, `store` | Roteamento de arquivo, type-safe |
| `@tanstack/router-plugin` | 1.168.47 | — | chokidar, babel | Codegen da árvore de rotas (devDependency) |
| `zustand` | 5.0.15 | **489 B** | **0** | Estado **de cliente**: filtro, nó selecionado, painel aberto |
| `zod/mini` | 4.6.7 | **3,0 kB** | 0 | Validação das respostas do MCP |
| `@tanstack/markdown` | 1.0.0 | 6,94 kB | **0** | Renderização de Markdown |
| `@tanstack/vite-config` | 0.6.0 | — | `vite-plugin-dts`, `vite-tsconfig-paths`, `externalize-deps` | Preset de Vite compartilhado (devDependency) |
| `@tanstack/react-query-devtools` | 5.104.4 | — | `@tanstack/query-devtools` | Inspeção do cache (devDependency) |
| `@tanstack/react-router-devtools` | 1.167.2 | — | `@tanstack/router-devtools-core` | Inspeção de rota (devDependency) |

**Total de runtime: +54,7 kB gzip** sobre o baseline de 84,28 kB → **~139 kB gzip**. Todo o resto é devDependency e não chega ao bundle.

### Decisões de execução

| ID | Decisão | Rationale |
|---|---|---|
| D-1 | **`@tanstack/config` não entra: o pacote está deprecado.** O slot "Config" do stack é ocupado por **`@tanstack/vite-config@0.6.0`**, sucessor do mesmo produto | `@tanstack/config@0.22.2` está marcado como *"Package no longer supported"* no npm, e a página `tanstack.com/config/latest` **continua recomendando** — 3 menções ao pacote deprecado, 0 ao sucessor. Instalar um pacote que o registro declara sem suporte não é aceitável. O sucessor tem peer `vite ^8.0.0`, que casa com o Vite 8.3.4 do viewer |
| D-2 | **`zod/mini`, não `zod`** | Medido: `zod` completo = 453.897 B raw / **92.955 B gzip**; `zod/mini` = 8.303 B raw / **3.076 B gzip**. São 30× de diferença. O `zod` completo, sozinho, quase dobraria o bundle atual da viewer |
| D-3 | **`@tanstack/react-devtools` (a plataforma) não entra; entram os dois plugins** | A plataforma genérica puxa `@tanstack/devtools` (297 kB) + `solid-js` (46 kB) + `devtools-ui` (89 kB). O que se quer é o plugin do Query e o do Router, que são reativos e leves |
| D-4 | **Markdown cru renderizado, com pré-processador de wikilinks no SPA** | O painel mostra o conteúdo real da nota, não um resumo. O custo é a duplicação da gramática de wikilink — tratada em "O pré-processador de wikilinks" abaixo |
| D-5 | **Roteamento de arquivo, com `routeTree.gen.ts` commitado** | Convenção do TanStack Router: o arquivo gerado vai para o repositório, para que o clone não dependa de rodar o dev server para ter as rotas |
| D-6 | **O DevTools é devDependency e entra só em `npm run dev`** | O bundle de produção não carrega nenhum dos dois |
| D-7 | **A CSP da ADR-059 não muda** | O bundle gerado pelo Vite não tem `<script>` inline, e a SPA fala só com a própria origem. `connect-src 'self'` já cobre `/mcp`. A única diretiva a revisar é `style-src 'self' 'unsafe-inline'`, mantida por folga para o atributo `style` inline |

### Emenda à ADR-059: a rota de fallback SPA

A ADR-059 (D-3) fixou o contrato de rotas do servidor. Esta ADR o **emenda**:

```
ANTES (ADR-059, D-3)
  GET /                    → app/index.html
  GET /graph/<arquivo>.html → artefatos

DEPOIS (esta ADR)
  GET /                    → app/index.html
  GET /assets/...          → build assets (arquivo existente)
  GET /graph/<arquivo>.html → artefatos (arquivo existente)
  GET /<qualquer>          → app/index.html  (fallback SPA)
```

**A regra de discriminação do fallback:** ele se aplica **somente a requisições de navegação** — requisições cujo header `Accept` contenha `text/html`. Uma requisição por `/assets/nao-existe.js` (que não aceita `text/html`) devolve **404**, não `index.html`. Sem essa regra, um asset faltando devolveria HTML com status 200 e o browser reportaria um erro de sintaxe em vez de um 404 honesto.

**A regra de segurança do fallback, e ela é não-negociável:** a verificação de path traversal acontece **antes** do fallback. Se o path pedido, depois de normalizado, escapa da raiz servida, a resposta é **404** — nunca o `index.html`. O fallback só é alcançado por paths que permanecem dentro da raiz. A ordem é: (1) resolver e normalizar o path; (2) se escapa da raiz → 404; (3) se o path é um arquivo existente → servir o arquivo; (4) se aceita `text/html` → `index.html` com 200; (5) senão → 404.

**Consequência para a superfície de ataque:** o fallback **não** aumenta o que é legível. Ele entrega o mesmo `index.html` que `GET /` já entregava, para paths de navegação. A subárea `graph/` continua servindo apenas artefatos existentes. O vetor novo é, no máximo, que um path de navegação desconhecido devolve 200 em vez de 404 — o que não expõe arquivo nenhum.

**Consequência para o ADR-050:** nenhuma. O `index.html` já era servido; o fallback só o reaproveita.

### O pré-processador de wikilinks

`@tanstack/markdown` **não** conhece wikilinks. Verificado: a doc oficial não menciona "wikilink", "Obsidian" ou `[[` em nenhum ponto; o *Syntax Profile* é CommonMark delimitado + frontmatter, com callouts **estilo GitHub** (`> [!TIP]`), não o set do Obsidian (`> [!note|warning|tip|important]`). As extensões exportadas pelo pacote são `docs`, `tabs`, `callouts`, `headings`, `framework`, `streaming` e `comment-components` — **nenhuma de wikilink**.

A ADR-005 e o ADR-009 tratam o `[[wikilink]]` como a aresta do grafo. O painel renderiza a nota, então o `[[...]]` precisa virar link clicável.

**A gramática que o pré-processador precisa cobrir** é a do parser Go (`internal/parser/wikilinks.go`), não uma invenção:

| Forma | Semântica no parser Go | O que a viewer renderiza |
|---|---|---|
| `[[X]]` | link para a nota X | link rotulado "X" |
| `[[X\|Label]]` | link com alias | link rotulado "Label" |
| `[[X#Secao]]` | âncora de cabeçalho | link para "X", âncora "Secao" |
| `[[X#Secao\|Label]]` | âncora + alias | link rotulado "Label", âncora "Secao" |
| `[[X#^c182]]` | referência de bloco | link para "X", bloco "c182" |
| `[[#Secao]]` | link interno do mesmo documento | âncora local, sem navegação |
| `![[X]]` | embed / transclusão | link para X; se X for imagem, imagem local |
| `[[memory://vault/path]]` | link federado cross-vault | **texto literal** — a viewer não resolve vault federado |
| `[[X\|rel:implements]]` | relação tipada | rótulo "X"; a relação fica no `title`, não é exibida |

**Os contextos onde não se cria link** são os mesmos do parser Go, e essa é a parte que ninguém pode inventar: dentro de bloco de código cercado (`` ``` ``), de código inline (`` ` ``), de comentário HTML (`<!-- -->`) e de linha de tabela. O Go já tem `stripCodeBlocks` (`internal/parser/wikilinks.go:26`) exatamente para não criar arestas ali. Se a SPA linkasse nesses contextos, o painel mostraria links que **não existem no grafo** — a viewer passaria a mentir sobre a topologia.

**O custo, dito sem rodeios:** essa regra é uma **duplicação** da gramática do Go, e a duplicação pode divergir. A mitigação não é optional: a tabela de casos do pré-processador no SPA tem de espelhar a tabela de casos do parser Go, e qualquer forma nova aceita pelo parser Go tem de ser aceita pelo pré-processador. É o mesmo modo de falha já registrado para o `internal/ci` nesta sessão — **a afirmação "não afeta" só é segura depois do gate rodando com os dois lados presentes**.

**A alternativa descartada** era processar no servidor, onde o Go já sabe a spec. Ela foi rejeitada por decisão de produto: ela muda o contrato da tool `memory_read_note` (que hoje devolve o Markdown bruto) e cria uma segunda forma de ler o mesmo conteúdo. O pré-processador no SPA mantém a tool honesta.

### O que não muda

- A viewer **continua sendo build estático** servido pelo MCP. Nenhuma dependência de runtime de aplicação (DR-1).
- A raiz servida **continua sendo um diretório dedicado**, nunca a raiz do vault (ADR-059, D-2).
- O transporte **continua sem autenticação**, e a ADR-050 continua sendo o lugar onde isso é resolvido (ADR-059, D-7).
- A viewer **continua sendo painel de métricas, não editor** (ADR-054, ADR-059).
- O grafo **continua sendo renderizado pelo renderer de `internal/graphview`**, com o vis-network vendorizado da ADR-058 como opção `--engine vis`. A viewer não desenha o grafo: ela **invoca** `memory_visualize_graph` e **abre** o artefato servido.
- O `viewer/dist/` **continua não versionado**; `viewer/` (fonte) passa a ser versionado.

## Consequências Positivas

- **Uma decisão, um lugar.** O stack, o custo em bytes e a gramática de wikilink ficam registrados, com números medidos, em vez de espalhados pelo `package.json` e pelo histórico de commits.
- **Deep link de verdade.** `http://127.0.0.1:38400/clusters` é uma URL que abre a visão de clusters, recarrega igual e pode ser colada num ticket.
- **O cache do Query é o que torna o painel viável.** 24 tools de grafo, cada uma com custo de consulta, não podem ser consultadas a cada render. O dedup e o `staleTime` do Query são o que impede o painel de martelar o índice.
- **`zod/mini` compra validação de resposta por 3 kB.** Toda resposta do MCP é validada na fronteira; erro de contrato vira erro de tela, não `undefined` silencioso no meio de um gráfico.
- **O pré-processador de wikilink torna a nota navegável.** O painel deixa de ser uma lista de metadados e passa a ser um grafo clicável a partir do texto.
- **A separação Query/Zustand fica explícita.** Estado de servidor no Query, estado de cliente no Zustand. O DR-6 fica codificado na arquitetura, não só na intenção.

## Consequências Negativas

- **+54,7 kB gzip** — o bundle vai de 84,28 kB para ~139 kB gzip. É irrelevante para um servidor local, mas é um aumento real de 65%.
- **A árvore de rotas passa a ser artefato gerado.** `routeTree.gen.ts` vai para o repositório. Codiplugem vai nele, e um conflito de merge exige regenerar.
- **`@tanstack/router-plugin` adiciona chokidar e babel ao build.** Mais uma superfície que pode quebrar em atualização.
- **A gramática de wikilink está duplicada entre Go e TypeScript.** É um custo de manutenção permanente, e o único jeito de ele não explodir é a tabela de casos espelhada. Se o Go aceitar uma forma nova, a viewer mostra texto literal até a TS acompanhar.
- **O bundle de produção não pode conter script inline.** A CSP `script-src 'self'` (sem `'unsafe-inline'`) da ADR-059 é o que torna isso obrigatório. Qualquer snippet de script inline futuro quebra o painel em silêncio — o browser recusa o script.
- **O CI não tem gate de frontend.** O job `frontend` morreu com o Electron em `b477200`. Nada garante, hoje, que `npm run build` passa. Este é o maior buraco da lista e precisa de task própria.

## Prós e Contras das Opções

### Opção A — Stack enxuto, sem router

- ✅ +23,7 kB gzip em vez de +54,7 kB
- ✅ Zero codegen, zero `routeTree.gen.ts`
- ✅ **Nenhuma emenda à ADR-059** — o contrato de rotas fica intacto
- ✅ Zero risco de gate de frontend (não há build step novo)
- ❌ Sem deep link
- ❌ O estado da tela se perde ao recarregar

### Opção B — Stack completo (escolhida)

- ✅ Deep link, histórico, estado na URL
- ✅ Type-safety ponta a ponta nas rotas
- ✅ DevTools de Query e de Router
- ✅ Compensa o custo em bytes se a viewer virar app de verdade
- ❌ +54,7 kB gzip
- ❌ Codegen com chokidar + babel
- ❌ Exige a emenda de rotas registrada nesta ADR
- ❌ Duplica a gramática de wikilink (mitigada por tabela de casos espelhada)

### Opção C — Sem TanStack

- ✅ Zero dependência, zero byte
- ✅ Zero decisão de stack pendente
- ❌ Sem cache: 24 tools de grafo reconsultadas a cada render
- ❌ Validação de resposta manual em cada call site
- ❌ Polling manual para `/health`
- ❌ Deep link exigiria reimplementar o que o Router dá pronto

### Opção D — Viewer Electron

- ✅ Aplicação desktop completa
- ✅ Editor de verdade, acesso ao filesystem
- ❌ Contraria a ADR-054 e a ADR-059
- ❌ 112 arquivos e 27.858 linhas já foram pagos para remover
- ❌ Introduz o runtime de aplicação que o DR-1 elimina

## Notas de Implementação

1. **A emenda de rota vive no Go, em `internal/mcp/http_server.go`.** A ordem é: resolver → normalizar → checar escape da raiz (404 se escapar) → arquivo existente (servir) → `Accept: text/html` (fallback `index.html`) → 404. **O fallback é o último passo antes do 404, nunca antes da checagem de escape.**
2. **A checagem de escape segue o precedente do repo**: `internal/compiler/note.go:181-184` resolve o path, compara com a raiz e devolve erro quando o resultado sai do diretório. O mesmo padrão, com 404 em vez de erro de vault.
3. **A URL na tool `memory_visualize_graph`** continua a ser a lacuna da ADR-059 (D-5): o handler é uma closure sobre `(gvFn, defaultRoot)` (`internal/mcp/visualizer_handlers.go:17`) e o `HTTPServer` não expõe o endereço (todos os campos privados, `internal/mcp/http_server.go:18-29`). O mecanismo registrado é o `net.Listener.Addr()` do caminho `Serve(l)` (`internal/mcp/http_server.go:294-297`), que resolve porta 0 e `:38400` para a porta concreta.
4. **O `routeTree.gen.ts` é commitado.** A geração vem do `@tanstack/router-plugin` no `vite.config.ts`; o arquivo gerado não entra no `.gitignore`.
5. **A raiz do alias `@` continua no `vite.config.ts`.** O `@tanstack/vite-config` traz `vite-tsconfig-paths`, que pode substituir o alias manual — a escolha é da implementação, desde que o build resolva `@/components/ui/button` (usado por `viewer/src/App.tsx:1`).
6. **`vite-plugin-dts` é inerte para um app.** Ele gera arquivos `.d.ts` que ninguém consome num bundle de aplicação. Vem junto com o preset; a implementação decide se mantém a saída no gitignore ou desliga o plugin.
7. **O gate de frontend no CI é task própria e obrigatória.** Hoje `go list ./...` já enxerga código Go distribuído por dependência npm (o pacote `flatted`), e o `internal/ci` filtra `node_modules` (`isThirdPartyPkg`) justamente por causa disso. Um job de frontend precisa rodar `npm ci` + `npm run build` **e** o invariant do `internal/ci` precisa passar **com** `node_modules` presente — foi exatamente a combinação que expôs o defeito no commit `cee9fb1`.
8. **A tabela de casos do pré-processador de wikilinks espelha a do parser Go.** Os casos são: link simples, alias, âncora, âncora + alias, blockid, link interno, embed, federado, relação no alias, e os quatro contextos sem link (código cercado, código inline, comentário HTML, linha de tabela). A spec registra os IDs.
9. **A resposta do MCP é validada na fronteira**, com `zod/mini`. O esquema de cada tool usada pelo painel é declarado uma vez e reaproveitado.
10. **O painel consome o MCP por `POST /mcp`** (`internal/mcp/http_server.go:245`, `handleDirectRPC`), na mesma origem, sem proxy e sem CORS especial — o CORS já habilitado cobre.

## Plano de Verificação

| Caso | Como se prova |
|---|---|
| Fallback devolve `index.html` em path de navegação | requisição `GET /clusters` com `Accept: text/html` devolve 200 e o HTML do bundle |
| Fallback **não** devolve `index.html` para asset | requisição `GET /assets/nao-existe.js` sem `Accept: text/html` devolve **404** |
| Fallback **não** abre brecha de traversal | `GET /../../etc/passwd` devolve **404** e **não** o `index.html` nem o arquivo |
| Arquivo existente é servido direto | `GET /assets/index-<hash>.js` devolve 200 com o JS |
| Artefato de grafo é servido | `GET /graph/graph.html` devolve 200 com o HTML gerado |
| A URL na tool usa a porta real | a tool devolve URL com a porta do listener, não uma hardcoded |
| A tool devolve caminho e URL | com transporte HTTP ativo, a tool devolve **ambos**; sem ele, devolve **só** o caminho, sem erro |
| Degradação sem build | sem `app/index.html`, `GET /` devolve **503** com mensagem simples |
| Resposta do MCP é validada | um payload fora do esquema produz erro de tela, não `undefined` |
| Wikilink vira link | a tabela de casos do pré-processador roda como teste; o mesmo conjunto de casos existe no parser Go |
| Wikilink **não** vira link em código | `[[X]]` dentro de bloco de código, código inline, comentário HTML e linha de tabela permanecem literais |
| Bundle não tem script inline | o `dist/index.html` gerado não contém `<script>` sem `src` |
| Build do frontend passa | `npm ci && npm run build` verde, e `npm run typecheck` verde |
| Invariant do CI com deps instaladas | `go test -count=1 ./internal/ci/...` verde **com** `node_modules` presente |
| CSP não quebra o bundle | o painel carrega com a CSP da ADR-059 aplicada e sem erro de CSP no console |

## Estado da Implementação

- **Esta ADR registra a decisão, não a implementação.** Nenhuma linha de Go ou de TypeScript foi alterada por ela.
- A spec do **lado do servidor** é `.specs/features/mcp-static-serving/spec.md` (22 ACs, `SERV-01..SERV-22`, todas `Pending`). Esta ADR **emenda** a D-3 dela: o AC que fixava "apenas `/` e `/graph/<arquivo>.html`" passa a admitir o fallback SPA, com a regra de discriminação por `Accept: text/html` e a regra de traversal antes do fallback. O `tasks.md` dessa feature **ainda não foi escrito**.
- A spec do **lado do cliente** (o painel em si) ainda **não existe** e precisa ser criada.
- `viewer/` **ainda não está versionado** (`git ls-files viewer` retorna vazio).
- O job de frontend no CI **não existe** (removido em `b477200`).
- A ADR-058 (renderizador vis-network) segue com o asset **ausente** do disco: `internal/graphview/assets/` não existe e não há `go:embed` em `internal/graphview`.

## Questões Abertas e Decisões Adiada

1. **A porta 052 permanece reservada** e não é sequestrada por esta ADR.
2. **A ADR-055** (`resolved_doc_id`) segue bloqueando `mem tags` e `mem search --tag`. Não é escopo desta ADR, mas um painel de métricas que mostre tags depende dela — o registro é apenas de dependência.
3. **A tabela de casos espelhada entre Go e TypeScript** precisa de um guard automatizado que detecte quando o parser Go aceita uma forma que o pré-processador não aceita. A forma do guard (teste de paridade, ou geração a partir do Go) é decisão da fase de implementação.
4. **A nível de detalhe dos painéis** (quais dos 24 tools viram painel, e com que granularidade) é escopo da spec da feature, não desta ADR.
5. **A CSP efetiva** — a baseline da ADR-059 precisa ser aplicada no servidor. É a implementação da spec `mcp-static-serving`, não desta ADR.
6. **O `vite-plugin-dts` inerte** pode ser desligado na primeira task de build, se a implementação preferir. Registrado como pendência, não como decisão.

## Links

- [ADR-059 — SPA React/Vite em `viewer/`, compilada para estático e servida pelo MCP](059-spa-react-vite-em-viewer-servida-pelo-mcp.md)
- [ADR-058 — Renderizador vis-network vendorizado, sem build step e sem CDN](058-renderizador-vis-network-vendorizado-sem-build-step.md)
- [ADR-054 — Escopo de paridade com o Obsidian é o vault, não o aplicativo](054-escopo-de-paridade-com-o-obsidian.md)
- [ADR-050 — Threat Model: OWASP LLM Top 10 Aplicado ao MyMemory](050-threat-model-owasp-llm-aplicado-ao-mymemory.md)
- [ADR-020 — Visualizador Interativo de Grafo em HTML/SVG Standalone](020-visualizador-interativo-de-grafo-em-html-svg.md)
- [ADR-005 — Markdown e `[[Wikilinks]]` como Entrada e Grafo Humano](005-markdown-com-wikilinks-como-fonte-de-verdade.md)
- [ADR-009 — Busca Híbrida com Reciprocal Rank Fusion (RRF)](009-busca-hibrida-com-reciprocal-rank-fusion-rrf.md)
- [ADR-002 — Adoção de Go como Linguagem Principal de Implementação](002-adocao-de-go-como-linguagem-principal.md)
- Spec do servidor: `.specs/features/mcp-static-serving/spec.md`
- Gramática de wikilink: `internal/parser/wikilinks.go`
