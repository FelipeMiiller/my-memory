---
title: "Validacao — Servir artefatos pelo MCP HTTP"
category: resource
summary: "Validacao da spec mcp-static-serving: 25 criterios EARS, prova end-to-end com o binario real (rotas, fallback SPA, path traversal com socket cru, structuredContent) e gate Go verde em 25/25 pacotes."
tags: [validation, mcp, static-serving, spa-fallback, adr-059, adr-060]
---

# Validacao — Servir artefatos pelo MCP HTTP

**Spec**: `.specs/features/mcp-static-serving/spec.md`
**Plan**: `.specs/features/mcp-static-serving/tasks.md`
**ADR**: `docs/adr/059-spa-react-vite-em-viewer-servida-pelo-mcp.md` (emendada pela `docs/adr/060-stack-do-frontend-da-viewer-e-fallback-spa.md`)
**Branch**: `develop` · **HEAD**: `4057ef0` · **Verificado em**: 2026-10-19
**Commits da feature**: `c3398a0` (ADR-059 completa) · `a344e1a` (ADR-060) · `1a06d75` (emenda da spec) · `e3e5817` (`go list -e`) · `121331e` (lado Go) · `bd6ea28` (gate de frontend no CI) · `4057ef0` (`working-directory` absoluto no CI)

---

## Veredito

**PASS** — 25/25 critérios satisfeitos, com prova de comportamento (não só de código): o
binário `mem` compilado foi subido em HTTP real e sondado com socket cru, e a bateria
de path traversal não produziu vazamento em nenhum payload.

O veredito **não** se apoia apenas em teste unitário: a seção "Prova end-to-end" abaixo
foi executada contra o binário real.

---

## Gate executado

| Gate | Comando | Resultado |
|---|---|---|
| Build | `go build ./...` | **exit 0** |
| Vet | `go vet ./...` | **exit 0** |
| Testes | `go test -count=1 ./...` | **25/25 pacotes ok, 0 falhas** |
| Formatação | `gofmt -l .` em worktree descartável a partir do HEAD | **0 arquivos** |
| Validador de spec | `validate_spec.py .specs/features/mcp-static-serving` | **0 erros** (25 ACs) |
| Validador de tasks | `validate_tasks.py .specs/features/mcp-static-serving` | **0 erros** (1 warning esperado) |
| Commit | `check_commit.py` | **OK** em todos os commits da feature |

> **Nota sobre `gofmt`.** No working tree do Windows, `gofmt -l .` lista ~200 arquivos.
> Isso é artefato do checkout antigo em CRLF, não do código commitado: num worktree
> descartável criado a partir do HEAD (que aplica o `.gitattributes` com
> `* text=auto eol=lf`), a listagem é **vazia**. O índice confirma `i/lf` para todos os
> arquivos Go tocados.

---

## Prova end-to-end com o binário real

Servidor subido com `bin/mem.exe mcp --port 38400` a partir da raiz do repositório
(raiz servida = `<CWD>/.mymemory/www/`), com o build da viewer publicado em
`.mymemory/www/app/`. Todas as requisições foram feitas com **socket cru**
(`System.Net.Sockets.TcpClient`), para que o caminho chegasse ao servidor **sem
normalização** — um cliente HTTP de alto nível limparia `..` antes de enviar e
invalidaria a prova.

### Contrato de rotas

| Requisição | Resultado | Veredito |
|---|---|---|
| `GET /` (`Accept: text/html`) | `200 OK`, corpo = entrypoint da SPA (480 B) | PASS |
| `GET /health` | `200 OK`, JSON com `tools_count: 24` | PASS |
| `GET /assets/index--xaLf-3x.js` | `200 OK`, 441.236 caracteres (bundle real) | PASS |
| `GET /clusters` (`Accept: text/html`) | `200 OK`, corpo = entrypoint da SPA | **fallback SPA** |
| `GET /assets/nao-existe.js` (sem `Accept: text/html`) | `404 Not Found`, corpo `not found` (10 B) | **não cai no fallback** |
| `GET /graph/nao-existe.html` (`Accept: text/html`) | `404 Not Found`, corpo de 33 B | **prefixo reservado nunca cai no fallback** |
| `GET /graph` (`Accept: text/html`) | `200 OK`, corpo = entrypoint da SPA (480 B) | **o path exato é rota da SPA** — a área reservada é o *conteúdo* de `/graph/` |
| `POST /index.html` | `405 Method Not Allowed` | PASS |

### Bateria de path traversal

Todos com `Accept: text/html` — o caso em que o fallback mascararia um escape.

| Payload | Status | Vazou conteúdo fora da raiz? |
|---|---|---|
| `/../../../etc/passwd` | `404` | **não** |
| `/..%2f..%2f..%2fetc%2fpasswd` | `404` | **não** |
| `/assets/../../../index.html` | `404` | **não** |
| `/....//....//etc/passwd` | `200` (entrypoint) | **não** — `....` é um segmento literal, não `..` |
| `/..%5c..%5cwindows%5cwin.ini` | `404` | **não** |
| `/%2e%2e/%2e%2e/etc/passwd` | `404` | **não** |
| `/app/../app/index.html` | `200` (entrypoint) | **não** — normaliza para dentro da raiz |
| `/..;/etc/passwd` | `200` (entrypoint) | **não** — `..;` é literal |
| `/C:/Windows/win.ini` | `404` | **não** |

Nenhum payload devolveu conteúdo de fora da raiz. Os `200` são todos o **entrypoint
da SPA** (480 B, idêntico ao de `GET /`), nunca um arquivo lido do disco.

### Headers de segurança

Presentes em **todas** as respostas, incluindo 404 e 405:

```
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline';
  img-src 'self' data:; font-src 'self'; connect-src 'self'; object-src 'none';
  base-uri 'none'; frame-ancestors 'none'
X-Content-Type-Options: nosniff
Referrer-Policy: no-referrer
```

O `404` respondeu `Content-Type: text/plain; charset=utf-8` — nunca HTML com 200.
O `script-src 'self'` **sem** `'unsafe-inline'` é viável porque o Tailwind v4 gera um
CSS externo; o `'unsafe-inline'` está apenas em `style-src`.

### `structuredContent` na resposta real do MCP

`POST /mcp` com `tools/call` para `memory_visualize_graph`:

```json
{"url":"http://127.0.0.1:38400/graph/graph.html",
 "path":"C:\\repository\\my-memory\\.mymemory\\www\\graph\\graph.html",
 "total_nodes":0,"total_edges":0,"hub_count":0,"density":0}
```

A URL usa a **porta real** do listener (38400), e o caminho em disco está exatamente
sob a raiz servida — o artefato que a tool escreve é o arquivo que o transporte publica.

---

## Evidencia por criterio de aceitacao

**PASS** = comportamento verificado, por teste e/ou pela prova end-to-end acima.

### P1 — Servir artefatos com escopo restrito (SERV-01 … SERV-08)

| ID | Critério | Veredito | Evidencia |
|---|---|---|---|
| SERV-01 | Servir arquivos sob um único subdiretório dedicado | **PASS** | `internal/mcp/static_server.go:20` (`ServedRootDir = ".mymemory/www"`), `:44-48`; prova E2E: `GET /` e `GET /assets/*` |
| SERV-02 | A raiz servida **não** é a raiz do vault | **PASS** | `ServedRoot()` junta `.mymemory/www` à base; `cmd/mem/main.go:44` (`servedRootBase = "."`); a raiz servida é `<CWD>/.mymemory/www/`, nunca `<CWD>/` |
| SERV-03 | Path que resolve fora da raiz é recusado | **PASS** | bateria de traversal: 9 payloads, 0 vazamentos; `internal/mcp/static_server_test.go:80-115` |
| SERV-04 | Duplicata registrada de SERV-03 | **PASS** | mesma barreira; a spec já marca a duplicata (`spec.md:108-109`) |
| SERV-05 | Default loopback-only | **PASS** | `cmd/mem/main.go:412` (`--host` default `127.0.0.1`); `--http` é opt-in explícito e a ADR-059 registra o bind address largo como decisão de produto fora de escopo |
| SERV-06 | `go build` e `go vet` exit 0 | **PASS** | medido: exit 0 nos dois |
| SERV-07 | Teste unitário de recusa de traversal | **PASS** | `internal/mcp/static_server_test.go:80-115` — 4 alvos, com `Accept: text/html` proposital, porque se o fallback fosse avaliado antes da barreira a resposta seria o entrypoint com 200 |
| SERV-08 | Teste unitário de arquivo legítimo servido | **PASS** | `internal/mcp/static_server_test.go:54-65` |

### P1 — Emenda ADR-060: fallback SPA (SERV-23 … SERV-25)

| ID | Critério | Veredito | Evidencia |
|---|---|---|---|
| SERV-23 | Path inexistente + `Accept: text/html` → entrypoint, 200 | **PASS** | E2E: `GET /clusters` → `200` com o entrypoint; `internal/mcp/static_server_test.go:132-143` |
| SERV-24 | Sem `Accept: text/html` → 404, não o entrypoint | **PASS** | E2E: `GET /assets/nao-existe.js` → `404`, corpo `not found`; `static_server_test.go:145-156` |
| SERV-25 | A checagem de traversal roda **antes** do fallback | **PASS** | ordem no código: resolver → escape → arquivo → fallback → 404; os 9 payloads de traversal com `Accept: text/html` devolveram 404 ou o entrypoint, **nunca** um arquivo de fora da raiz |

### P2 — `memory_visualize_graph` devolve URL (SERV-09 … SERV-14)

| ID | Critério | Veredito | Evidencia |
|---|---|---|---|
| SERV-09 | URL presente quando o transporte HTTP está ativo | **PASS** | E2E: `structuredContent.url = http://127.0.0.1:38400/graph/graph.html`; `internal/mcp/visualizer_handlers_test.go:244` |
| SERV-10 | O caminho em disco continua sendo devolvido | **PASS** | E2E: `structuredContent.path`; `internal/mcp/visualizer_handlers.go:26` (`json:"path"`, sem `omitempty`) |
| SERV-11 | Sem transporte: só o caminho, **sem erro** | **PASS** | `internal/mcp/visualizer_handlers.go:22-25` — o campo `url` é `omitempty` e só é preenchido quando há base; `visualizer_handlers_test.go:276` |
| SERV-12 | O artefato é escrito na raiz servida antes de responder | **PASS** | `internal/mcp/visualizer_handlers.go` grava sob a raiz servida; E2E: o arquivo existe em `.mymemory/www/graph/graph.html` e a URL aponta para o mesmo arquivo |
| SERV-13 | A URL usa host e porta **reais** do listener | **PASS** | `net.Listener.Addr()`; E2E mediu a porta 38400 real; `visualizer_handlers_test.go:309` (`UrlUsesRealListenerPort`); `internal/mcp/static_server_test.go:261-273` — o HTTPServer e o Server MCP publicam o **mesmo** endereço |
| SERV-14 | Teste unitário para o par URL presente / ausente | **PASS** | `visualizer_handlers_test.go:244` e `:276` |

### P2 — Lacuna de exposição *acknowledged* (SERV-15 … SERV-18)

| ID | Critério | Veredito | Evidencia |
|---|---|---|---|
| SERV-15 | A ADR registra a mudança de superfície (0 → 1 diretório) | **PASS** | `docs/adr/059-...md` — seção de superfície de ataque, com a mudança explicitada |
| SERV-16 | A ADR diz que **não há autenticação** e que isso é pré-existente | **PASS** | ADR-059 (gap da ADR-021, não criado pela feature) |
| SERV-17 | A ADR lista o que a exposição significa (só a raiz servida, nunca o vault, nunca fora) | **PASS** | ADR-059; e a prova E2E confirma: 9 payloads de traversal, 0 vazamentos |
| SERV-18 | A ADR registra que bind address largo está fora de escopo | **PASS** | ADR-059; confirmado pelo default loopback em `cmd/mem/main.go:412` |

### P3 — Sem dependência de frontend no módulo Go (SERV-19 … SERV-22)

| ID | Critério | Veredito | Evidencia |
|---|---|---|---|
| SERV-19 | O módulo Go **não** ganha dependência de Node | **PASS** | `go build` / `go vet` / `go test` exit 0 **com e sem** `node_modules` presente; o Go não importa nada de `viewer/` |
| SERV-20 | Nenhuma referência de CDN em runtime | **PASS** | grep por `cdn.`, `unpkg`, `jsdelivr`, `googleapis` em `internal/mcp/*.go` e `cmd/mem/*.go` (sem `_test.go`): **0 ocorrências** |
| SERV-21 | A aplicação React/Vite é compilada para estático **fora** do build Go | **PASS** | `viewer/dist/` → `.mymemory/www/app/`, por `make viewer`; o binário Go embute nenhum byte da viewer |
| SERV-22 | A ADR declara que o artefato servido é saída estática agnóstica de framework | **PASS** | ADR-059 e ADR-060 |

---

## Defeito encontrado e corrigido no caminho

| Defeito | Gravidade | Commit | Correção |
|---|---|---|---|
| O path exato `/graph` — que é uma rota da SPA — caía dentro do prefixo reservado `/graph/` e devolvia **404 em texto puro** ao ser aberto ou recarregado direto. A navegação por link funcionava, porque o roteador do cliente não passa pelo servidor | **ALTA** | `abfb087` | A área reservada passou a ser o **conteúdo** de `/graph/`. O path exato `/graph` e `/graph/` caem no fallback da SPA, como qualquer path de navegação — que é o que a ADR-060 exige desde a sua emenda. `GET /graph/<arquivo>` continua servindo o artefato e `GET /graph/nao-existe.html` continua devolvendo 404 sem o entrypoint |
| O último step do job `frontend` do CI usava `working-directory: .` para voltar à raiz do repositório, num job que declara `defaults.run.working-directory: viewer` | **MÉDIA** — a justificativa estava errada (ver abaixo), mas explicitar o caminho absoluto é melhor | `4057ef0` | `working-directory: ${{ github.workspace }}` — absoluto, sem depender de resolução implícita |
| O orçamento do bundle no CI impunha 180 kB, mais permissivo que o teto de 160 kB da spec | **MÉDIA** | `4b0af43` | Orcamento alinhado com o teto da spec: 163.840 bytes |
| O preflight CORS respondia 204 sem os headers de segurança | **BAIXA** | `248ab0d` | A baseline é aplicada antes do early return do preflight |
| `StaticHandler.Root()` não tinha chamador | **BAIXA** | `248ab0d` | Removido |

O primeiro defeito só apareceu na **validação visual** das quatro rotas da
viewer, que o verificador independente não podia fazer (ele não renderiza a
SPA). Os outros quatro foram apontados por ele.

> **Correção de registro:** a justificativa original do `working-directory`
> afirmava que um caminho relativo de step herda o default do job. O código do
> runner (`actions/runner`, `src/Runner.Worker/Handlers/ScriptHandler.cs`)
> mostra o contrário: ele é combinado com o **workspace**
> (`Path.Combine(workspaceDir, workingDirectory)`), e o default do job só entra
> quando o step **não** define o valor. O `working-directory: .` original já
> rodava na raiz do repositório, e o job não teria ficado vermelho por causa
> disso. O CHANGELOG foi corrigido com a razão certa.

---

## Notas não bloqueantes

1. **A spec registra `zod` como 92,9 kB gzip; a ADR-060 e o CHANGELOG registram
   90,8 kB.** Divergência de medição entre dois documentos, sem efeito em
   comportamento. A ADR-060 é a autoridade; a spec é documento de planejamento e
   fica como registro histórico.
2. **`go test ./...` local lista `viewer/node_modules/flatted/golang/pkg/flatted`.**
   É o pacote Go que a dependência npm `flatted` distribui. Ele compila e não tem
   teste, então não quebra nada — mas o invariant de `internal/ci` (o que realmente
   importa) filtra `node_modules` desde `cee9fb1` e `e3e5817`, e é ele que o job
   `frontend` do CI exercita **com `node_modules` presente**, justamente para não
   depender de um checkout limpo.
3. **`total_nodes: 0` na prova E2E** não é defeito do serving: o índice local deste
   repositório está desatualizado, o que já está registrado em
   [ISSUE-019](../../ISSUES.md#issue-019--mem-doctor-dá-health-score-100100-num-grafo-vazio).

---

## Verificador independente

A feature foi auditada por um verificador **independente** — processo separado,
sem participação na implementação, com as ferramentas do repositório em um
worktree descartável. O veredito dele foi **PASS-WITH-GAPS**.

### O que o verificador provou

| Verificação | Método | Resultado |
|---|---|---|
| Barreira de path traversal | **socket cru** (`net.Dial` + linha de requisição manual), 34 requisições com 20+ payloads (`..`, `%2e%2e`, `..%2f`, `..%5c`, `..%20`, `/C:/Windows/win.ini`, `c%3a`, `%00`, `..%00`, UNC, `..;/`, `....//`) | **zero vazamento** de arquivo de fora da raiz |
| Ordem de avaliação | leitura do código + prova | `splitURLPath` → barreira de escape → arquivo → `/graph` 404 → fallback. **O escape nunca chega ao fallback** |
| Fallback só com `text/html` | `GET /assets/nao-existe.js` | sem Accept → **404**; com `Accept: text/html` → **200 + index.html** (achado M2, registrado na ISSUE-021) |
| `/graph/` reservado | `GET /graph/qualquer.html` | **404**, nunca cai no fallback |
| Raiz servida ausente | `GET /` | **503** com mensagem simples + os 3 headers de segurança |
| Headers de segurança | 200, 404 e 503 | presentes em todas; **ausentes só no preflight OPTIONS** (achado B3, corrigido nesta rodada) |
| Texto das 24 tools | `git diff 121331e^ 121331e -- internal/mcp/` | `NewTextResult` **inalterado**; todos os hunks são `NewTextResult(X)` → `NewStructuredResult(X, Y)` com o **mesmo X** |
| 10 tools com `structuredContent` | grep de `NewStructuredResult(` | 11 call-sites → **10 tools distintas**; as outras 14 omitem via `omitempty` |
| Forma × zod, campo a campo | leitura comparativa Go × `viewer/src/lib/schemas.ts` | os 10 payloads batem em nomes e tipos, incluindo `hub_count` e `density` |
| Gramática Go × TS | **diferencial**: 30 casos de `parseWikilink` + 9 corpos + 13 relações, com os dois lados executados de verdade | **0 divergências** |
| Invariante do CI | `go test -count=1 -v ./internal/ci/...` na raiz, com `viewer/node_modules` presente | **PASS** (exit 0) |
| Números da documentação | contagem no repositório | 59 ADRs · 25 pacotes Go · 24 tools · 7 skills — todos conferem |

### Achados do verificador e o que foi feito com cada um

| ID | Achado | Decisão |
|---|---|---|
| **A1** (ALTO) | `memory_get_drift` emitia `null` em `drifted_notes` e `uncovered_code`; o zod v4 **rejeita** `null` (aceita `undefined`), então o painel `/drift` degradava para texto markdown | **Corrigido nesta rodada** — normalização central no wire + teste de regressão, com **mutação provada** (sem a correção, os 8 casos falham com o payload `null` exato) |
| **M1** (MÉDIO) | O gate do CI impunha 180 kB, mais permissivo que o teto de 160 kB da spec | **Corrigido** — o orçamento do CI passou a ser 163.840 bytes, o próprio teto da spec |
| **M2** (MÉDIO) | A justificativa do fallback na ADR-060 não se sustenta para navegadores: eles mandam `Accept: text/html` ao pedir um `.js` | **Registrado** na [ISSUE-021](../../ISSUES.md) — é mudança de contrato, decisão de produto |
| **M3** (MÉDIO) | O CHANGELOG afirmava que um `working-directory` relativo de step herda o default do job. O código do runner (`actions/runner`, `ScriptHandler.cs`) mostra o contrário: ele é combinado com o **workspace** | **Corrigido** — o registro foi corrigido no CHANGELOG, com a razão certa. A troca por caminho absoluto foi mantida (é melhor depender do explícito), mas a justificativa anterior estava errada |
| **B1** (BAIXO) | Caracteres CJK espúrios no texto de duas ADRs e numa spec desta rodada | **Spec corrigida**; as ADRs são imutáveis → [ISSUE-024](../../ISSUES.md) |
| **B2** (BAIXO) | As specs não tinham `validation.md` | **Corrigido** — este arquivo e o da `viewer-dashboard`. A tabela de rastreabilidade continua com `Pending` por convenção do repo (a feature `obsidian-parity-foundation`, concluída, também mantém) |
| **B3** (BAIXO) | O preflight CORS respondia 204 sem os headers de segurança | **Corrigido** — a baseline agora é aplicada antes do early return do preflight |
| **B4** (BAIXO) | `StaticHandler.Root()` não tinha chamador | **Corrigido** — removido |
| **B5** (BAIXO) | `memory_get_impact` e `memory_find_path` não emitem `structuredContent`; dois schemas da viewer nunca validam nada | **Registrado** na [ISSUE-022](../../ISSUES.md) — decisão de produto |
| **B6** (BAIXO) | O servidor anuncia o protocolo `2024-11-05` mas emite `structuredContent` (campo de `2025-06-18`); 4 das 10 tools devolvem array onde a spec MCP exige objeto | **Registrado** na [ISSUE-023](../../ISSUES.md) — exige ADR própria |
| **B7** (BAIXO) | A spec citava o bundle antigo (259,52 kB JS / 84,28 kB gzip) | **Corrigido** |
| **B8** (BAIXO) | `GET /graph/../x` normaliza para fora do prefixo e cai no fallback | **Registrado** junto da ISSUE-021 — benigno (só entrega o entrypoint público) |

### O que o verificador não verificou

1. **Renderização real da SPA no navegador** — não subiu a SPA contra um MCP vivo. Deixado como lacuna desta rodada (ver `viewer-dashboard/validation.md`).
2. **O job `frontend` rodando no GitHub Actions real** — a semântica do `working-directory` foi deduzida do **código-fonte** do runner, não de observância.
3. **MCP sob cliente estrito** — não validou `structuredContent` array contra um cliente que exija objeto, nem contra o Claude Desktop.
4. **A exposição sem autenticação** (reconhecida na ADR-059 D-7) — fora do escopo desta rodada; a barreira de arquivos é que foi atacada.
5. **As 43 features** de `.specs/features/` — só as duas da rodada foram auditadas em detalhe.
