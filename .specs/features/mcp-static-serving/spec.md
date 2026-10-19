# Servir artefatos pelo MCP HTTP — URLs clicáveis para o agente

## Problem Statement

O transporte HTTP do MyMemory (`internal/mcp/http_server.go`) é um `http.ServeMux` que expõe exatamente quatro rotas:

```
/health    /sse    /message    /mcp
```

Nenhuma serve arquivo estático. Isso cria uma fricção concreta para o consumidor real do produto — o agente.

`memory_visualize_graph` e `mem graph view` produzem um HTML autossuficiente e devolvem **um caminho de arquivo**. Para um agente, um caminho não é acionável: ele não tem como abrir `/home/user/projeto/grafo.html` a partir de uma sessão que roda em outra máquina, dentro de um contêiner, ou atrás de um filesystem diferente. O agente recebe algo que precisa ser convertido em ação manual pelo humano.

Enquanto isso, a lista de comandos que o MCP **não** expõe continua grande: `properties`, `tasks`, `bases`, `outlines`, `tags`, e o próprio `.base` que o projeto acabou de criar.

Desde 2026-10-19 existe `viewer/` — uma SPA React 19 + Vite 8 + shadcn que compila para `viewer/dist/` (com o stack da ADR-060: 441,5 kB JS / **133.086 bytes gzip -9**; o scaffold sem o stack media 259,52 kB JS / 84,28 kB gzip), registrada na [ADR-059](../../../docs/adr/059-spa-react-vite-em-viewer-servida-pelo-mcp.md). Ela dá um destino concreto ao que esta spec pedia em abstrato: **`viewer/dist/` é o artefato servido**, e é onde um painel de Health Score, órfãs e modularidade pode viver.

Existe uma pergunta adjacente — "o MCP consegue servir uma aplicação React+Vite?" — cuja resposta é sim e é barata (~15 linhas de `http.FileServer`), mas cujo uso é fácil confundir com "trazer o viewer de volta".

O repo já pagou esse experimento uma vez: `b477200` removeu 112 arquivos e 27.858 linhas de viewer Electron com React, Vite, Forge e Playwright. E a ADR-054, escrita em 2026-10-18, declara explicitamente que plugins, themes e `dev:*` estão **fora de escopo** porque "exigem uma instância Electron viva".

## Goals

- [ ] O transporte HTTP do MCP serve artefatos estáticos, com escopo explícito e sem hole de segurança
- [ ] `memory_visualize_graph` passa a devolver uma URL servida, mantendo o caminho como fallback
- [ ] Estabelecer o precedente de que qualquer frontend servido pelo MCP é **build estático**, nunca runtime de aplicação

## Out of Scope

| Feature | Reason |
|---|---|
| **Recriar o viewer Electron** | Revertido em `b477200`; ADR-054 declara plugins e `dev:*` fora de escopo |
| Canvas 3D, chat LLM, IPC streaming | ADRs 051/053 obsoletas; o consumidor Electron foi deletado |
| `mymemoryd` servindo assets | ADR-042 está `Proposed` e o lifecycle é inalcançável (`cmd/mymemoryd/main.go:92` cria o Manager com `nil`) |
| Autenticação no servidor HTTP | Decisão de produto separada; o transporte atual já é loopback sem auth e isso **não** muda aqui |
| Plugin `.base` nativo | Gap real da matriz de paridade, mas é a ADR-056, não esta feature |
| **Construir a SPA** | `viewer/` já existe e compila (ADR-059). Esta feature **serve** o resultado; não o produz |

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | --------------- | --------- | ---------- |
| Forma servida | **build estático** de qualquer stack, servido por `http.FileServer` | Não acopla o produto a React/Vite; quem quiser React compila para estático | y |
| Raiz servida | **`<repo>/.mymemory/www/`**, com `app/` (build da SPA) e `graph/` (artefatos), nunca a raiz do vault | Servir a raiz do vault transformaria o MCP num servidor de arquivos do vault inteiro. `.mymemory/www/` fica **fora do escopo de indexação**, então os artefatos gerados não poluem o grafo | y |
| Autenticação | Fora de escopo, mas o bind padrão continua loopback | Ampliar a exposição é decisão de produto, não consequência desta feature | n |
| `memory_visualize_graph` | Devolve URL **e** caminho | Um cliente MCP não-Windows ou sem acesso ao mesmo filesystem ainda usa o caminho | y |
| React/Vite | **Nenhuma** dependência nova de runtime | `viewer/` já existe (ADR-059) e compila; o MyMemory apenas serve o resultado, não o compila | y |

**Open questions:** none.

### Dimensões de requisito implícito (sweep Large)

| Dimensão | Destino |
|---|---|
| Input validation & bounds | SERV-07 — o path pedido precisa ser resolvido dentro da raiz servida; qualquer escape é erro |
| Failure / partial-failure states | SERV-09, SERV-10 — servidor sem porta, cliente que não acessa o filesystem, diretório ausente |
| Idempotency / retry | Geração de artefato é idempotente; servir é read-only por definição |
| Auth boundaries & rate limits | **SERV-13** — é a lacuna mais séria: o transporte hoje é loopback sem auth, e servir arquivos muda o que está exposto |
| Concurrency / ordering | `http.FileServer` é seguro para concorrência por construção |
| Data lifecycle / expiry | SERV-11 — artefatos gerados ficam no disco; política de limpeza é do vault, não do servidor |
| Observability | SERV-12 — o HTML emitido declara a URL de origem, para diagnóstico |
| External-dependency failure | N/A — sem CDN e sem dependência externa, por decisão |
| State-transition integrity | N/A — nenhuma migration |

---

## User Stories

### P1: Servir artefatos com escopo restrito ⭐ MVP

**User Story**: Como pessoa rodando o MCP em HTTP, quero abrir o artefato gerado no browser sem procurar arquivo no disco, para que a URL seja a interface.

**Why P1**: É o que remove a fricção. Todo o resto depende disso existir.

**Acceptance Criteria**

1. The HTTP server SHALL serve static files under a single dedicated subdirectory of the vault
2. The served root SHALL NOT be the vault root
3. IF a requested path resolves outside the served root THEN the server SHALL refuse it and not return the file
4. IF a requested path resolves outside the served root THEN the server SHALL refuse it and not return the file
5. The served root SHALL default to loopback-only binding, matching the current transport
6. `go build ./...` and `go vet ./...` SHALL exit 0 with the feature absent from the codebase
7. A unit test SHALL assert that a path traversal attempt is refused
8. A unit test SHALL assert that a legitimate file under the served root IS served
9. IF a request for a path that resolves to no existing file under the served root accepts `text/html` THEN the server SHALL respond with the SPA entrypoint and status 200
10. A request that does not accept `text/html` SHALL NOT be answered with the SPA entrypoint; it SHALL return 404
11. The path-traversal check SHALL be evaluated BEFORE the SPA fallback; a path resolving outside the served root SHALL return 404 and never the SPA entrypoint

**Independent Test**: request `../../etc/passwd`-style paths and a known file under the root.

> **Emenda ADR-060 (2026-10-19).** Os ACs 9 a 11 (SERV-23, SERV-24, SERV-25) foram
> adicionados pela [ADR-060](../../../docs/adr/060-stack-do-frontend-da-viewer-e-fallback-spa.md),
> que registra o stack do frontend da viewer com um roteador de cliente. Sem fallback,
> recarregar em `/clusters` devolveria 404 em vez de renderizar a rota.
>
> A ordem de avaliação é o contrato, e ela é **não-negociável**:
> **(1)** resolver e normalizar o path → **(2)** se escapa da raiz servida, **404** →
> **(3)** se é arquivo existente, servir o arquivo → **(4)** se aceita `text/html`,
> devolver o entrypoint com 200 → **(5)** senão, **404**.
>
> O fallback **não** aumenta o que é legível: entrega o mesmo entrypoint que `GET /`
> já entregava. A distinção por `Accept: text/html` existe para que um asset
> faltando (`/assets/nao-existe.js`) continue sendo 404 honesto, e não HTML com 200.
>
> O AC 4 é a duplicata de escrita já registrada na nota de Spec Precision abaixo; o
> comportamento é um só, coberto por SERV-07.

---

### P2: `memory_visualize_graph` devolve URL

**User Story**: Como agente, quero receber um endereço clicável junto com o caminho, para conseguir abrir o grafo sem pedir intervenção humana.

**Acceptance Criteria**

1. WHEN `memory_visualize_graph` runs and the HTTP transport is active THEN the result SHALL include a URL under the served root
2. The result SHALL still include the filesystem path
3. IF the HTTP transport is not active THEN the result SHALL contain only the path, with no error
4. IF the artefact has not been written to the served root THEN the handler SHALL write it before returning
5. The URL SHALL be built from the host and port the transport is actually listening on, not from a hardcoded value
6. A unit test SHALL assert the URL is present when the transport is active and absent when it is not

**Independent Test**: run the handler twice, once with and once without the transport.

---

### P2: Lacuna de exposição acknowledged, não escondida

**User Story**: Como maintainer, quero que a mudança de superfície do servidor esteja escrita antes de existir, para que ninguém descubra depois que o MCP passou a servir arquivos.

**Acceptance Criteria**

1. An ADR SHALL record that the MCP HTTP transport went from serving no files to serving one directory
2. The ADR SHALL state explicitly that the transport has **no authentication** and that this predates the feature
3. The ADR SHALL list what the exposure means: only the served root is readable, never the vault root, never outside it
4. The ADR SHALL record that widening the bind address is out of scope and is a separate product decision
5. The ADR SHALL record that authentication is a known gap, not an oversight of this feature

**Independent Test**: read the ADR; every listed concern has an entry.

---

### P3: Sem dependência de frontend nova

**User Story**: Como maintainer, quero que o MyMemory não passe a depender de React, Vite ou de qualquer `package.json`, para que o histórico de `b477200` não se repita.

**Acceptance Criteria**

1. The Go module SHALL NOT gain a Node dependency: `go build ./...`, `go vet ./...` and `go test -count=1 ./...` SHALL exit 0 without `node_modules` being installed
2. The feature SHALL NOT introduce any runtime CDN reference
3. IF a React or Vite application is served THEN it SHALL have been compiled to static assets outside the MyMemory build
4. The ADR SHALL state that the served artefact is framework-agnostic static output

**Independent Test**: grep the Go-side diff for `package.json`, `node_modules`, `vite`, and remote origins; all must return zero.

> **Emenda ADR-060 (2026-10-19).** O AC 1 era "The feature SHALL NOT add a
> `package.json`, bundler configuration, or Node dependency to the repository".
> Isso deixou de ser verdade — e a culpa é da spec, não do código: quando ela foi
> escrita, `viewer/` ainda era só um plano, e hoje `viewer/package.json` existe e
> vai ser versionado (ADR-059 e ADR-060).
>
> A **intenção** do AC é o que importa, e ela é o que fica: **o build do Go não
> pode depender de Node**. A amendment reescreve o AC em termos verificáveis —
> `go build`, `go vet` e `go test` passam sem `node_modules` instalado. A
> toolchain `viewer/` é uma concern de repositório separada, com a ADR-059 e a
> ADR-060 como registro, e o gate de frontend no CI (hoje inexistente) é a
> task que fecha a conta.
>
> O que separa isso do `b477200` (112 arquivos e 27.858 linhas removidos) é que
> o Electron **entrou no build e no CI** do produto. Aqui a toolchain fica
> **fora** do módulo Go, com o `.gitignore` filtrando `node_modules` e `dist`.
>
> O Independent Test mudou junto: passa a valer para o **diff do lado Go**, e
> não para o repositório inteiro.

---

## Edge Cases

- IF the served root does not exist on disk THEN the server SHALL report that plainly rather than returning 404 for every request
- IF an artefact was deleted after its URL was handed out THEN the URL returns not-found; that is correct behaviour, not a bug
- IF two agents request the same artefact concurrently THEN the write is idempotent because the content is deterministic from the `GraphView`
- IF the vault lives only in Postgres and has no local directory THEN static serving has nothing to serve; the handler SHALL say so instead of pretending

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----------- | ------ | ------- |
| SERV-01 | P1: Servir artefatos | Execute | Pending |
| SERV-02 | P1: Servir artefatos | Execute | Pending |
| SERV-03 | P1: Servir artefatos | Execute | Pending |
| SERV-04 | P1: Servir artefatos | Execute | Pending |
| SERV-05 | P1: Servir artefatos | Execute | Pending |
| SERV-06 | P1: Servir artefatos | Execute | Pending |
| SERV-07 | P1: Servir artefatos | Execute | Pending |
| SERV-08 | P1: Servir artefatos | Execute | Pending |
| SERV-09 | P2: URL no visualize_graph | Execute | Pending |
| SERV-10 | P2: URL no visualize_graph | Execute | Pending |
| SERV-11 | P2: URL no visualize_graph | Execute | Pending |
| SERV-12 | P2: URL no visualize_graph | Execute | Pending |
| SERV-13 | P2: URL no visualize_graph | Execute | Pending |
| SERV-14 | P2: Exposição acknowledged | Execute | Pending |
| SERV-15 | P2: Exposição acknowledged | Execute | Pending |
| SERV-16 | P2: Exposição acknowledged | Execute | Pending |
| SERV-17 | P2: Exposição acknowledged | Execute | Pending |
| SERV-18 | P2: Exposição acknowledged | Execute | Pending |
| SERV-19 | P3: Sem dependência nova | Execute | Pending |
| SERV-20 | P3: Sem dependência nova | Execute | Pending |
| SERV-21 | P3: Sem dependência nova | Execute | Pending |
| SERV-22 | P3: Sem dependência nova | Execute | Pending |
| SERV-23 | P1: Servir artefatos (fallback SPA) | Execute | Pending |
| SERV-24 | P1: Servir artefatos (fallback SPA) | Execute | Pending |
| SERV-25 | P1: Servir artefatos (fallback SPA) | Execute | Pending |

**Coverage:** 25 total, 25 mapped to stories, 0 unmapped.

> **Nota de Spec Precision:** o critério de recusa de path traversal aparece duas vezes
> no P1, por erro de escrita. Está listado como um único comportamento; o requisito
> SERV-07 é o que o cobre. A duplicata não é um requisito separado.

---

## Success Criteria

- [ ] `go build ./...`, `go vet ./...` and `go test -count=1 ./...` exit 0
- [ ] A request for a file under the served root returns it; a traversal attempt does not
- [ ] `memory_visualize_graph` returns a URL when the transport is active
- [ ] The diff contains no `package.json`, no `node_modules`, no bundler config, and no remote origin
- [ ] An ADR records the exposure change and the pre-existing lack of authentication