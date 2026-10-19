# ADR-059: SPA React/Vite em `viewer/`, compilada para estático e servida pelo MCP

- **Data**: 2026-10-19
- **Status**: Accepted
- **Decisores**: Felipe Miiller
- **Tags**: viewer, react, vite, mcp, http, build-estatico, seguranca
- **Relacionada**: [ADR-021](021-servidor-mcp-com-transporte-http-sse.md), [ADR-050](050-threat-model-owasp-llm-aplicado-ao-mymemory.md), [ADR-054](054-escopo-de-paridade-com-o-obsidian.md), [ADR-058](058-renderizador-vis-network-vendorizado-sem-build-step.md), [ADR-048](048-viewer-electron-react-vite-shadcn.md) (obsoleta)

## Contexto e Problema

O transporte HTTP do MCP é um `http.ServeMux` com **exatamente quatro rotas** — `/health`, `/sse`, `/message`, `/mcp` (`internal/mcp/http_server.go:71-75`). Todas servem o protocolo; **nenhuma serve arquivo estático**.

Daí o problema: `memory_visualize_graph` (`internal/mcp/tools.go:392-416`) devolve no markdown o **caminho absoluto** do HTML gerado (`internal/mcp/visualizer_handlers.go:81-94` monta `- **Arquivo HTML**: \`<caminho absoluto>\``). Para um agente remoto esse caminho é inútil — a sessão pode rodar em outra máquina ou num contêiner. O mesmo vale para `mem graph view`, que só abre o browser na máquina do processo (`cmd/mem/graph.go:111`, `:123-126`). Em paralelo, o My-Memory já calcula métricas que existem **apenas como texto no terminal**: Health Score, órfãs, dead links, modularidade, PageRank e drift. Nada as apresenta.

O repositório já tentou um viewer antes: `b477200` (2026-10-18) removeu **112 arquivos e 27.858 linhas** de Electron + React + Vite + Forge + Playwright, tirou o job `frontend` do CI e **não tocou** em `cmd/` nem `internal/`. A lição não é "não use bibliotecas": é **não introduza build step no caminho de release do binário Go**. Esta ADR decide **o que** será servido e **como**, sem implementar: é o contrato da etapa seguinte.

## Drivers da Decisão

- **DR-1**: Dar ao agente uma **URL clicável**, não um caminho que só faz sentido na máquina do processo.
- **DR-2**: Apresentar as métricas que o `mem doctor` já calcula, sem instrumentar o core Go.
- **DR-3**: **Nenhum build step no caminho de release do binário Go.** `go build ./...` funciona sem Node.
- **DR-4**: Preservar a garantia central: **tudo local, tudo offline**. Nenhum CDN, nenhuma requisição externa.
- **DR-5**: **Não repetir `b477200`** — nada de runtime empacotado, daemon obrigatório ou IPC streaming.
- **DR-6**: Conteúdo servido não pode ser lido **de fora** da raiz servida. A barreira de path traversal tem de ser **testada**.
- **DR-7**: A mudança de superfície de ataque precisa estar **registrada por escrito**, inclusive a ausência de autenticação.
- **DR-8**: A decisão precisa ser **verificável por testes** ancorados no que já existe, sem harness novo.

## Opções Consideradas

- **A — SPA React/Vite em `viewer/`, build estático, servida pelo MCP** (escolhida).
- **B — Servir só o HTML do `graphview`, sem SPA** (o HTML standalone de 38 KB de `mem graph view`, `internal/graphview/template.go:11`).
- **C — Trazer de volta o viewer Electron** (reverter `b477200`).
- **D — Servir a SPA de um servidor separado** (ex.: `vite preview`), com o MCP apenas produzindo a URL.

## Resultado da Decisão

Escolhida a **Opção A**.

```
viewer/                     fonte React 19 + Vite 8 + TS 6 + Tailwind 4 + shadcn
  └─ npm run build          etapa de build separada, offline, opcional
       └─ viewer/dist/      ARTEFATO DE BUILD — não versionado
            └─ copiado para <repo>/.mymemory/www/app/
                 ├─ GET /              → app/index.html
                 └─ GET /graph/*.html  → artefatos de grafo
```

### Decisões de execução

| Decisão | Escolha | Racionale |
|---|---|---|
| `viewer/dist/` versionado? | **Não** | Artefato de build: versioná-lo duplica 21 arquivos a cada build e cria conflito de merge binário. Já filtrado pelos dois `.gitignore` |
| Como gerar | hoje: `npm run build` em `viewer/`. O alvo `make viewer` é **estado-alvo a ser criado** — **não existe** no `Makefile` atual | Se `dist/` não existir, o MCP diz que não existe. O alvo no `Makefile` é conveniência, não pré-requisito |
| Nome no `package.json` | `mymemory-dashboard` | Nome do pacote npm, independente do nome da pasta |
| Onde a SPA vive | Dentro do repo, em `viewer/` | Compartilha histórico; um único `git clone` entrega tudo |

### Decisões de projeto

| ID | Tema | Escolha | Rationale |
|---|---|---|---|
| **D-1** | Artefato | build estático de `viewer/` | O MCP **não compila** a SPA; o binário Go não depende de Node |
| **D-2** | Raiz servida | **um** diretório, `<repo>/.mymemory/www/`, com `app/` e `graph/` | Nunca a raiz do vault; fora do escopo de indexação; ignorado pelo git. **Hoje não existe** (`cmd/mem/main.go:2547`) |
| **D-3** | Rotas e URL | as 4 existentes **intocadas** + `GET /` (SPA) + `GET /graph/<arquivo>.html`; SPA na **raiz** | O bundle usa paths absolutos e `viewer/vite.config.ts` não tem `base: './'` — sob prefixo quebraria sem rebuild |
| **D-4** | Path traversal | caminho normalizado tem de continuar **dentro** da raiz; fora → **404** (não 403) + log | Padrão **já existe** em `internal/compiler/note.go:181-184` — modelo a seguir |
| **D-5** | Mecanismo da URL | o `HTTPServer` publica o **endereço efetivamente vinculado** | **Lacuna real**: hoje o handler não descobre host/porta. Ver "Notas", item 4 |
| **D-6** | Degradação | sem `app/index.html` → **503**; sem `graph/` → 404, com mensagem | `ExportHTML` não faz `os.MkdirAll` (`internal/graphview/template.go:1069-1076`) |
| **D-7** | Autenticação | **lacuna pré-existente, registrada** — não criada aqui | Muda **o que é legível dentro do bind**: de "zero arquivos" para "a raiz servida" |
| **D-8** | Conteúdo derivado | o HTML do grafo traz **títulos de nós e conexões** do vault | Exposição real a quem alcance a porta, mesmo só em `graph/` |
| **D-9** | CORS | as rotas estáticas herdam o `Access-Control-Allow-Origin: *` | `withCORS` envolve o mux inteiro (`internal/mcp/http_server.go:79`) e `--cors` é `true` por default |
| **D-10** | CSP | divergência **consciente** com a ADR-050; baseline adiante | `default-src 'none'` é **incompatível com SPA JS**; a ADR-050 foi feita para o viewer Electron |
| **D-11** | Consumo pela SPA | `/mcp` (JSON-RPC direto) e `GET /health`, **mesma origem** | `handleDirectRPC` já existe (`internal/mcp/http_server.go:245`): sem proxy. `GET /health` dá status, versão, `tools_count`, `active_sessions`, `default_repo` (`:39-49`, `:223-242`) |
| **D-12** | Duas toolchains | Go + Node no mesmo repo | Não afeta `go build ./...`. Mas o CI **não tem gate de frontend**, e `go list ./...` passa a enxergar **código Go distribuído por dependência npm** — exige filtro em `internal/ci` |

### Por que isto não é `b477200`

| | `b477200` (revertido) | `viewer/` (esta decisão) |
|---|---|---|
| Runtime | Electron, app distribuído | Estático, servido por HTTP |
| Build step no release | sim, Vite + Forge | **não** — o binário Go não depende do build da SPA |
| Comunicação | IPC streaming | HTTP + MCP |
| Estado | app local com janela | arquivos served sob subpasta restrita |
| Depende de daemon | sim (`mymemoryd`) | **não** — o lifecycle do supervisor está inalcançável (`cmd/mymemoryd/main.go:92` cria o Manager com `nil`) |

O `viewer/` é **painel de métricas sobre dados já calculados**, não editor. A [ADR-054](054-escopo-de-paridade-com-o-obsidian.md) declara plugins, themes e `dev:*` fora de escopo porque exigem instância Electron viva; nada aqui exige.

### O que muda no repositório

1. O repo passa a ter **duas toolchains** (Go + Node). **Estado-alvo:** a pasta `viewer/` **entra versionada**, e `node_modules/` e `dist/` permanecem **fora**. **Hoje ainda não está versionada** — `git ls-files viewer` retorna vazio.
2. `go build ./...` **não é afetado** — Go ignora diretórios sem `.go`. Mas o invariant de `internal/ci` **é** afetado, e a correção já está feita: `go list ./...` varre o filesystem e devolve `viewer/node_modules/flatted/golang/pkg/flatted`, porque o pacote npm `flatted` distribui um port Go. O `internal/ci` passou a descartar qualquer pacote sob `node_modules` (`isThirdPartyPkg`), já que o invariant é sobre código **deste** repositório. Sem o filtro, `go test ./...` ficava **vermelho localmente** para quem rodasse `npm install` em `viewer/`.
3. O transporte HTTP passa a servir um **diretório dedicado**, **nunca a raiz do vault**. Essa é a mudança de superfície real. O escopo e o teste de path traversal são especificados pela spec `.specs/features/mcp-static-serving/spec.md` (22 ACs `SERV-01..SERV-22`) — **não** pela ADR-058, que trata do renderizador vis-network vendorizado e cujo asset ainda **não** existe em disco (`internal/graphview/assets/` não existe e não há `go:embed` em `internal/graphview`).

## Superfície de Ataque e Postura de Segurança

A ADR-021 já registra a lacuna de autenticação (`docs/adr/021-servidor-mcp-com-transporte-http-sse.md:57`); a ADR-050 (`Proposed`) define a postura de longo prazo. Aqui o registro é do que **esta feature** acrescenta.

### Tabela de superfície — antes e depois

| | Antes | Depois |
|---|---|---|
| Rotas no mux | 4 — `/health`, `/sse`, `/message`, `/mcp` | 4 **inalteradas** + `GET /` (SPA) + `GET /graph/<arquivo>.html` |
| Arquivos legíveis via HTTP | **zero** | só `<repo>/.mymemory/www/app/` e `<repo>/.mymemory/www/graph/` |
| Raiz do vault exposta | não | **continua não**: raiz dedicada, fora do escopo de indexação |
| Conteúdo derivado do vault | não | **sim**, via `graph/`: títulos de nós e conexões (D-8) |
| Headers de segurança | nenhum | a aplicar na implementação (baseline abaixo) |
| Autenticação / TLS | nenhum / nenhum | **nenhum / nenhum** — lacuna pré-existente (D-7) |
| CORS | `*` quando `--cors` (default `true`) | **idêntico**: as rotas novas herdam `*` (D-9) |
| Bind default | loopback (`--host` = `127.0.0.1`, `cmd/mem/main.go:403-406`) | **inalterado** |

### Inventário de controles

| Controle | Onde | Existe? |
|---|---|---|
| Bind loopback por default | `cmd/mem/main.go:403-406` | ✅ ativo |
| CORS desligável | `cmd/mem/main.go:406`; `internal/mcp/http_server.go:277-291` | ✅ ativo — **permite `*`** |
| Escopo de indexação restrito | `.memory/config.yaml` (`docs/`, `specs/`, `.specs/`, `README.md`) | ✅ ativo — `.mymemory/` fora |
| Barreira de path traversal (outro domínio) | `internal/compiler/note.go:181-184` | ⚠️ **não** cobre o servidor HTTP — é o **modelo** |
| Timeouts e headers de resposta | `internal/mcp/http_server.go:77-83`, `:139-142`, `:195`/`:212`/`:240`/`:267` | ✅ ativos |
| **Autenticação** | — | ❌ **nenhuma**: sem token, API key ou allowlist (ADR-021, `docs/adr/021-servidor-mcp-com-transporte-http-sse.md:57`) |
| **TLS** | — | ❌ **nenhum**; exposição pública via reverse proxy (mTLS ou API Key) |
| **CSP** | — | ❌ **nenhum** header no repo: 1 único hit, em Markdown obsoleto (`docs/adr/048-...md:81`). `viewer/index.html` e `viewer/dist/index.html` **sem** meta tag CSP |
| **Rate limit**, **headers de segurança**, **allowlist de `Origin`** | — | ❌ **nenhum dos três**; a ADR-050 exige allowlist (`docs/adr/050-...md:67`) |

### Tabela de vetores

| Vetor | O que muda | Mitigação nesta feature | Risco residual aceito |
|---|---|---|---|
| **Path traversal** (`GET /graph/../../vault.md`) | rota que resolve caminho de arquivo | caminho normalizado tem de continuar dentro da raiz; fora disso → **404** (não 403) e motivo no log | **Baixo**, desde que a implementação aplique a barreira — o teste é AC da spec `mcp-static-serving` |
| **Leitura do bundle por outra origem** | as rotas estáticas herdam `Access-Control-Allow-Origin: *` | nenhuma nesta feature | **Médio e consciente** — a superfície exposta é o **bundle público** (artefato de build), não dado de vault. Mitigação agendada na ADR-050: `--cors=false` ou allowlist |
| **Artefato de grafo expondo conteúdo do vault** | o HTML do grafo é servido por HTTP | superfície restrita a **uma subpasta** (`graph/`) de uma raiz dedicada; bind loopback; **nunca** a raiz do vault | **Médio e aceito** — é o aumento de exposição mais relevante desta feature; nenhum controle de leitura foi adicionado |
| **Bind em `0.0.0.0`** | não muda por conta desta feature, mas é o vetor que torna a ausência de auth relevante | nenhuma — decisão de produto separada, fora do escopo | **Alto se ativado** — `--http :38400` ou `--host 0.0.0.0` ignoram o default loopback, porque `--http` recebe o addr verbatim (`cmd/mem/main.go:419-422`); a ADR-021 exige reverse proxy com mTLS ou API key |
| **Ausência de CSP** | a SPA executa JavaScript no browser de quem abre a URL | a implementação deve aplicar a baseline abaixo, mais `X-Content-Type-Options: nosniff` e `Referrer-Policy: no-referrer` | **Médio** — enquanto a baseline não for aplicada, o bundle roda sem restrição de origem |

### Fronteira de escopo — o que esta ADR **não** decide

| Tema | Onde está registrado | Por que não aqui |
|---|---|---|
| Autenticação e TLS do transporte MCP | [ADR-050](050-threat-model-owasp-llm-aplicado-ao-mymemory.md) (`Proposed`), vetor "MCP — transporte" (`docs/adr/050-...md:67`); ADR-021 (`:57`) | lacuna **pré-existente**; esta feature não a cria |
| Ampliar o bind para `0.0.0.0` | decisão de produto separada, fora do escopo | nenhum vínculo com a rota estática |
| Plugins, themes, `dev:*`; editor de notas | [ADR-054](054-escopo-de-paridade-com-o-obsidian.md) | exigem instância Electron viva; o viewer é painel de métricas |
| Runtime Electron | [ADR-048](048-viewer-electron-react-vite-shadcn.md) (obsoleta) | revertido por `b477200`; a lição é absorvida, não o código |
| Gate de frontend no CI | D-12 | dívida de pipeline, não de arquitetura de transporte |

### Divergência consciente com a ADR-050 (CSP)

A ADR-050 exige, no vetor "Viewer" (`docs/adr/050-...md:70`), **CSP strict** com `default-src 'none'; img-src 'self' data:`. Isso é **incomível com uma SPA em JavaScript**: `default-src 'none'` bloquearia o script da própria aplicação. A ADR-050 foi escrita para o viewer Electron, que renderizava Markdown sanitizado num documento sem script próprio — não para um app React. Baseline que a implementação deve aplicar:

```
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'
X-Content-Type-Options: nosniff
Referrer-Policy: no-referrer
```

- `script-src 'self'` **sem** `'unsafe-inline'` — viável: o `viewer/dist/index.html` gerado traz só `<script type="module" crossorigin src="/assets/...">`, e o código-fonte não tem `<script>`, `style={{…}}` nem `dangerouslySetInnerHTML`.
- `style-src 'self' 'unsafe-inline'` — folga deliberada: o Tailwind v4 compila para um CSS **externo** (`/assets/index-*.css`), não style inline; o `'unsafe-inline'` cobre o atributo `style` inline que componentes shadcn usam para valores dinâmicos.
- `connect-src 'self'` — a SPA fala com `/mcp` na **mesma origem**; nenhum destino externo.
- `img-src 'self' data:`, `object-src 'none'`, `base-uri 'none'`, `frame-ancestors 'none'` — nada remoto, nada embutido, não frameável.

> [!note]
> A ADR-050 está em status **Proposed** (`docs/adr/050-...md:4`) e só vira `Accepted` após red-team. A divergência fica registrada agora para que, quando a ADR-050 subir, ela já considere o viewer como **SPA** e não como renderizador de Markdown.

## Consequências Positivas

- O agente recebe **URL**, não caminho — acionável de qualquer máquina que alcance a porta.
- As métricas de grafo e saúde viram visuais, sem instrumentação nova no core Go.
- Frontend iterável com hot reload, contra 38 KB de HTML inline que ninguém mantém.
- `viewer/dist/` é artefato limpo: trocar de framework é apagar uma pasta.

## Consequências Negativas

- **O repo passa a exigir Node para desenvolver o viewer** (não para usar o binário).
- `viewer/dist/` ausente numa clone nova significa MCP sem painel — degradação silenciosa se não for tratada (a D-6 define o 503 para evitar isso).
- **Nenhuma proteção de CSP/auditoria de dependência frontend** enquanto a baseline D-10 não for aplicada; a ADR-050 não cobre esse caminho de build.
- O invariant de `internal/ci` **não** cobre `viewer/` — e o job `frontend` do CI foi removido com o Electron em `b477200`. Em compensação, o invariant precisou de um filtro: `go list ./...` enxerga **código Go distribuído por dependência npm** (`flatted`), que não é código deste repositório.
- O CORS `*` agora cobre também as rotas estáticas (D-9): qualquer origem da máquina lê o bundle.
- A URL entregue ao agente **depende** do transporte HTTP ativo; no modo stdio (default do `mem mcp`) há só caminho, por degradação sem erro.

## Prós e Contras das Opções

### Opção A: SPA React/Vite em `viewer/`, build estático, servida pelo MCP ✅ Escolhida

- ✅ URL clicável para o agente (DR-1); métricas do `mem doctor` apresentáveis (DR-2).
- ✅ Binário Go autossuficiente: `go build ./...` não depende de Node (DR-3, DR-5).
- ✅ Superfície delimitada: **um** diretório dedicado, nunca a raiz do vault.
- ❌ Duas toolchains (D-12), CI sem gate de frontend, e disciplina de path traversal exigida na implementação.

### Opção B: Servir só o HTML do `graphview`, sem SPA

- ✅ Zero build step, zero toolchain nova, nenhum bundle JavaScript para atacar.
- ✅ Reaproveita o HTML standalone já coberto por `cmd/mem/graph_test.go`.
- ❌ Não apresenta Health Score, órfãs, dead links, modularidade, PageRank ou drift: resolve a URL, não o painel (DR-2).
- ❌ Continua um HTML de 38 KB mantido à mão; não posiciona o terreno para o painel de métricas de D-11.

### Opção C: Trazer de volta o viewer Electron

- ✅ App completo, com integração com SO e hot reload de verdade; o código já existiu em `b477200`.
- ❌ Reintroduz build step no release (Vite + Forge) — exatamente o que `b477200` corrigiu (DR-3, DR-5).
- ❌ Depende de `mymemoryd`, cujo lifecycle de supervisor está inalcançável (`cmd/mymemoryd/main.go:92`); empacotamento multiplataforma para um recurso que não precisa de janela.

### Opção D: Servir a SPA de um servidor separado

- ✅ Desacopla totalmente: **zero** mudança no transporte Go, hot reload real, nenhum aumento da superfície do MCP.
- ❌ Duplica processo e duplica a superfície de rede — dois serviços em vez de um.
- ❌ Quebra a promessa de "um binário só": o binário Go deixa de ser autossuficiente.
- ❌ O agente ainda precisa conhecer a porta do servidor separado, o que reintroduz o problema de configuração que a URL deveria resolver.

## Notas de Implementação

Contrato para a etapa futura, em ordem de execução:

1. **Onde entra a rota.** As 4 rotas existentes **permanecem intocadas**; as novas entram no **mesmo** `mux` (`internal/mcp/http_server.go:71-75`), o que as faz herdarem o wrapper `withCORS` (`:79`).
2. **Raiz servida.** `<repo>/.mymemory/www/`, com `app/` e `graph/`. Essa pasta **não existe hoje** — a implementação deve criá-la. `.mymemory/` fica fora do escopo de indexação, então os artefatos não poluem o grafo.
3. **Path traversal.** Caminho resolvido e normalizado tem de continuar dentro da raiz; fora disso, **404** (não 403) e motivo no log. Precedente: `internal/compiler/note.go:181-184`; testes: `internal/compiler/note_test.go:51-60` e `:182-189`.
4. **URL real (D-5) — lacuna a fechar.** O handler de `memory_visualize_graph` **hoje não tem** como descobrir host e porta: `NewMemoryVisualizeGraphHandler(gvFn, defaultRoot)` é uma closure sobre `(gvFn, defaultRoot)` (`internal/mcp/visualizer_handlers.go:17`) e o `HTTPServer` **não expõe** o endereço — todos os campos são privados (`internal/mcp/http_server.go:18-29`), e não existe `Addr()`, `GetAddr()` nem `BaseURL()`. **Decisão de projeto:** o `HTTPServer` publica o endereço efetivamente vinculado, obtido de `net.Listener.Addr()` no caminho `Serve(l)` (`internal/mcp/http_server.go:294-297`), que resolve porta `0` e `:38400` para a porta concreta; o `Server` expõe esse endereço ao handler. Sem transporte HTTP ativo, o handler devolve **apenas o caminho, sem erro** — exigência da spec `mcp-static-serving`.
5. **Criação de diretório.** `graphview.ExportHTML` faz `os.WriteFile(outputPath, htmlBytes, 0644)` **sem** `os.MkdirAll` (`internal/graphview/template.go:1069-1076`) e falha se o destino não existir. Criar `graph/` é **responsabilidade da implementação**, e a falha precisa virar mensagem, não panic. O nome do artefato segue as regras atuais — `graph.html` ou `graph_<rootNode>.html`, com `/` e `\` trocados por `_` e sufixo `.md` removido (`internal/mcp/visualizer_handlers.go:49-58`; espelhado em `cmd/mem/graph.go:71-81`).
6. **Raiz da URL.** A SPA é servida **na raiz** (`GET /`), não sob subpasta, porque `viewer/dist/index.html` referencia `/assets/...` com caminho absoluto e `viewer/vite.config.ts` não tem `base: './'`. Se a subpasta for escolhida depois, é preciso rebuild com `base` relativo.
7. **Makefile.** O alvo `make viewer` **não existe** hoje e é **estado-alvo a ser criado**; o comando real hoje é `npm run build` dentro de `viewer/`. O `Makefile` atual tem `all`, `build`, `test`, `test-all`, `test-coverage`, `fmt`, `lint`, `clean`.
8. **`.gitignore` da raiz.** O comentário das linhas 40-41 ("viewer Electron removido (2026-10-18) … barreira defensiva") está desatualizado: `viewer/` passa a ser **fonte versionada**, e `dist/` continua fora.
9. **CORS herdado (D-9) e degradação (D-6).** As rotas estáticas herdam `Access-Control-Allow-Origin: *` porque `withCORS` envolve o mux inteiro e `--cors` é `true` por default — dívida da ADR-050. Sem `app/index.html` → `GET /` responde **503** com mensagem simples ("viewer não compilado — rode `npm run build` em `viewer/`"); sem `graph/` → 404 com mensagem simples. As flags existentes (`--host`, `--port`, `--http`, `--cors`, em `cmd/mem/main.go:403-406`) cobrem o bind — **nenhuma variável de ambiente nova**.
10. **Pacote Go dentro de `node_modules`.** `go list ./...` varre o filesystem e devolve `viewer/node_modules/flatted/golang/pkg/flatted` — o pacote npm `flatted` distribui um port Go em `golang/pkg/flatted`. O invariant de `internal/ci` (todo pacote roda na CI) quebrava para quem developing localmente. O filtro `isThirdPartyPkg` em `internal/ci/ci.go` descarta qualquer pacote sob `node_modules`, com dois testes em `internal/ci/ci_test.go` fixando a semântica. A alternativa — listar o pacote em `excludedPackages` — foi rejeitada: a árvore npm muda com o lockfile e o `StaleExclusions` reportaria a exclusão como obsoleta.

## Plano de Verificação

**Base de testes atual (não precisa mudar):** `internal/mcp/http_server_test.go` — 6 testes, de `TestHTTPServer_SSE_HandshakeAndMessage` (:25) a `_ServeAndShutdown` (:265). `cmd/mem/mcp_server_test.go:24` — `TestMCPServer_CLI_Network`, que sobe `runMCPServer` real em `127.0.0.1:<porta livre>` e valida `GET /health` com status e `DefaultRepo` (:61-80); hoje **não** testa rota estática, nem CORS, nem path traversal. `internal/mcp/visualizer_handlers_test.go` — 8 testes, de `_Tool_List` (:16) a `TestServer_SetGraphViewHandler` (:202). Precedente de path traversal: `internal/compiler/note_test.go:51-60` ("path traversal rejeitado" / "path traversal profundo rejeitado") e `:182-189` (`TestWriteAtomicNote_TraversalBlocked`).

**Casos concretos a entregar:**

| # | Caso | Resultado esperado |
|---|---|---|
| 1 | Arquivo **dentro** da raiz servida | **200** com o conteúdo e o `Content-Type` correto |
| 2 | Tentativa de traversal (`..`, caminho absoluto, separadores misturados) | **recusada**, e o arquivo **não** é devolvido em nenhuma variante |
| 3 | `GET /` **sem** build | **503** com a mensagem de "viewer não compilado" |
| 4 | `GET /graph/arquivo.html` inexistente | **404** com mensagem simples, sem stack trace |
| 5 | `memory_visualize_graph` com transporte HTTP **ativo** | devolve **URL e caminho** |
| 6 | `memory_visualize_graph` com transporte HTTP **inativo** | devolve **só o caminho**, **sem erro** |
| 7 | Servidor em porta não-default (incluindo porta `0` resolvida) | a URL usa a **porta real**, nunca uma hardcoded |
| 8 | As 4 rotas originais | continuam respondendo como antes (regressão) |
| 9 | CORS e headers de segurança nas rotas estáticas | herdam o `withCORS` atual (registrado, não "corrigido" aqui) e trazem a baseline de CSP da D-10 |

> [!warning]
> A spec `mcp-static-serving` é a fonte da verdade dos critérios de aceitação (22 ACs `SERV-01..SERV-22`). A lista acima é o **núcleo verificável**, não a lista completa.

## Estado da Implementação

- **Esta ADR registra a decisão, não a implementação.** A única linha de Go tocada no mesmo passo foi o filtro de `node_modules` em `internal/ci` — e ele é **pré-requisito de versionar `viewer/`**, não da rota estática.
- O filtro de `node_modules` em `internal/ci/ci.go` **já está implementado** (`isThirdPartyPkg`, com dois testes em `internal/ci/ci_test.go`). Sem ele, `go test ./...` fica vermelho localmente para quem rodar `npm install` em `viewer/`.
- Spec: `.specs/features/mcp-static-serving/spec.md` — **22 ACs, `SERV-01..SERV-22`, todos em `Pending`**. O `tasks.md` **ainda não foi escrito**.
- A spec exige, entre outras coisas, que a ADR registre a mudança de superfície e a ausência de autenticação (ACs de P2: `SERV-14..SERV-18`) — é a seção "Superfície de Ataque e Postura de Segurança" que cumpre esse papel.
- A [ADR-058](058-renderizador-vis-network-vendorizado-sem-build-step.md) (renderizador vis-network vendorizado) também é **decisão, não implementação**: `internal/graphview/assets/` **não existe** em disco e não há `go:embed` em `internal/graphview`.
- O scaffold `viewer/` existe em disco, **100% untracked**; `viewer/dist/` existe como build local.
- A etapa de implementação transforma esta ADR em código: começa pelo `tasks.md` na spec, seguindo o ciclo Specify → Design → Tasks → Execute da skill `tlc-spec-driven`.

## Questões Abertas e Decisões Adiada

| Questão | Situação | Onde está encaminhada |
|---|---|---|
| Autenticação e TLS do transporte MCP | **adiado** — lacuna pré-existente, não introduzida aqui | [ADR-050](050-threat-model-owasp-llm-aplicado-ao-mymemory.md) (`Proposed`), vetor "MCP — transporte" (`docs/adr/050-...md:67`); ADR-021 (`:57`) exige reverse proxy para exposição pública |
| CSP efetivo | **baseline definida** (D-10); a **aplicação** é trabalho da implementação | esta ADR ("Superfície de Ataque") + ADR-050, que precisa ser atualizada para o caso SPA |
| Gate de frontend no CI | **adiado** — dívida conhecida (D-12) | trabalho de pipeline, a definir na implementação |
| Alvo `make viewer` | **estado-alvo**; o `Makefile` atual não o tem | notas de implementação (item 7) |
| Porta 052 reservada | **não sequestrar** por esta feature | segue reservada; não pertence a esta decisão |
| `resolved_doc_id` (ADR-055) | **sem relação** com esta decisão; segue bloqueando `mem tags` | [ADR-055](055-semantica-de-tag-consultavel-sem-nos-mortos.md) — registrado aqui só para não ser confundido com o escopo do viewer |

## Links

- Specs: `.specs/features/mcp-static-serving/spec.md` (22 ACs `SERV-01..SERV-22`) e `.specs/features/graph-view-vis-engine/spec.md` (36 ACs `VIS-01..VIS-36`)
- [ADR-021](021-servidor-mcp-com-transporte-http-sse.md) — o transporte que ganha a rota estática; registra a lacuna de auth/TLS
- [ADR-050](050-threat-model-owasp-llm-aplicado-ao-mymemory.md) — threat model transversal; vetores "MCP — transporte" e "Viewer"
- [ADR-054](054-escopo-de-paridade-com-o-obsidian.md) — a fronteira de escopo que esta decisão **não** atravessa
- [ADR-058](058-renderizador-vis-network-vendorizado-sem-build-step.md) — renderizador do grafo; decisão com asset ainda ausente
- [ADR-048](048-viewer-electron-react-vite-shadcn.md) — obsoleta; o caminho que `b477200` removeu
- Código de referência: `internal/mcp/http_server.go`, `internal/mcp/visualizer_handlers.go`, `internal/graphview/template.go`, `internal/compiler/note.go`, `cmd/mem/main.go`, `cmd/mem/graph.go`
- Índice de ADRs: `docs/adr/README.md`
