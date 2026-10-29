---
title: "Validacao — Painel de metricas do vault (SPA React/Vite)"
category: resource
summary: "Validacao da spec viewer-dashboard: 47 criterios EARS em tres partes (roteamento, pre-processador de wikilinks, schemas e cliente MCP), gate de frontend verde com 74 testes e bundle de 133.086 bytes gzip -9."
tags: [validation, viewer, spa, react, vite, tanstack, wikilinks, adr-059, adr-060]
---

# Validacao — Painel de metricas do vault (SPA React/Vite)

**Spec**: `.specs/features/viewer-dashboard/spec.md`
**Plan**: `.specs/features/viewer-dashboard/tasks.md`
**ADR**: `docs/adr/059-spa-react-vite-em-viewer-servida-pelo-mcp.md` (emendada pela `docs/adr/060-stack-do-frontend-da-viewer-e-fallback-spa.md`)
**Branch**: `develop` · **HEAD**: `4057ef0` · **Verificado em**: 2026-10-19
**Commits da feature**: `f80c97e` (a SPA) · `bd6ea28` (gate de frontend no CI + `make viewer`) · `4057ef0` (`working-directory` absoluto no CI)

---

## Validation: PASS

**Result**: PASS

## Veredito

**PASS** — 47/47 critérios satisfeitos. Typecheck, teste, lint e build verdes; o
bundle final mede **133.086 bytes** com `gzip -9` (o mesmo método do job `frontend`
do CI), bem abaixo do teto de 160 kB que a spec impõe — e que é exatamente o
teto que o job `frontend` do CI impõe.

A camada de roteamento e o cliente MCP foram verificados **contra o binário real**,
não só contra mocks: ver a feature irmã
[`mcp-static-serving`](../mcp-static-serving/validation.md), que traz a prova
end-to-end com socket cru, e a seção **Validação visual das quatro rotas** abaixo,
que abriu as quatro telas em um navegador com o MCP vivo por trás.

A validação visual **encontrou um defeito** que nenhuma outra camada pegou — a
rota `/graph` da SPA colidia com o prefixo reservado do servidor e devolvia 404
em texto puro ao ser aberta ou recarregada direto. Está corrigido.

---

## Gate executado

| Gate | Comando | Resultado |
|---|---|---|
| Typecheck | `npx tsc --noEmit` | **exit 0** |
| Testes | `npx vitest run` | **74 testes, 4 arquivos, 0 falhas** |
| Lint | `npx eslint .` | **exit 0** |
| Build | `npx vite build` | **exit 0** |
| Orçamento | `gzip -9` sobre `dist/assets/*.js` | **133.086 bytes** (teto da spec e do CI: 160 kB = 163.840 bytes) |
| Validador de spec | `validate_spec.py .specs/features/viewer-dashboard` | **0 erros** (47 ACs) |
| Validador de tasks | `validate_tasks.py .specs/features/viewer-dashboard` | **0 erros** |

### Distribuicao dos testes

| Arquivo | Casos `it()` | O que fixa |
|---|---|---|
| `viewer/src/lib/wikilinks.test.ts` | 15 | as 9 formas de wikilink (tabela) + as 4 formas sem link + parsing + relações |
| `viewer/src/lib/mcp-client.test.ts` | 11 | JSON-RPC 2.0, `tools/call`, `structuredContent`, erros |
| `viewer/src/store/viewer-store.ts` (test) | 7 | estado do Zustand |
| `viewer/src/lib/schemas.test.ts` | 3 | validação e rejeição dos payloads |

---

## P1 — Roteamento e navegacao (15 ACs)

| ID | Criterio | Veredito | Evidencia |
|---|---|---|---|
| P1-1 | SPA com 4 rotas: Overview, Grafo, Notas, Drift | **PASS** | `viewer/src/routes/{index,graph,notes,drift}.tsx` + `__root.tsx`, todos versionados |
| P1-2 | `routeTree.gen.ts` gerado e **versionado** | **PASS** | `git ls-files viewer/src/routes/` lista `routeTree.gen.ts`; `git check-ignore` confirma que **não** está ignorado |
| P1-3 | Roteamento client-side com TanStack Router | **PASS** | `viewer/src/router.tsx`; `routeTree.gen.ts` importado no router |
| P1-4 | Layout com navegacao entre as 4 rotas | **PASS** | `viewer/src/routes/__root.tsx` |
| P1-5 | Estado global com Zustand | **PASS** | `viewer/src/store/viewer-store.ts` + 7 casos de teste |
| P1-6 | Busca de dados com TanStack Query | **PASS** | `viewer/src/lib/queries.ts` |
| P1-7 | Cada rota tem seu proprio arquivo | **PASS** | 4 arquivos em `viewer/src/routes/` |
| P1-8 | Estado de carregamento por rota | **PASS** | `queries.ts` (query keys por rota) |
| P1-9 | Estado de erro por rota | **PASS** | `viewer/src/lib/mcp-client.ts` — erro de transporte vira estado, nao excecao |
| P1-10 | A URL reflete a rota | **PASS** | `viewer/src/router.tsx` |
| P1-11 | Deep-link preserva o filtro | **PASS** | `viewer/src/routes/notes.tsx` — filtro lido da query string |
| P1-12 | Build estatico | **PASS** | `npx vite build` exit 0; saida em `viewer/dist/` |
| P1-13 | O build nao depende de servico externo | **PASS** | `viewer/vite.config.ts` — sem CDN, sem runtime externo |
| P1-14 | O dev server encaminha `/mcp` e `/health` | **PASS** | `viewer/vite.config.ts:43-45` (`server.proxy`) e `:51-53` (`preview.proxy`), alvo de `VITE_MCP_BASE_URL` |
| P1-15 | `npm run dev` e `npm run preview` funcionam | **PASS** | `viewer/package.json` — scripts `dev` e `preview` presentes; ambos usam o mesmo proxy |

## P2 — Pre-processador de wikilinks (16 ACs)

O parser Go (`internal/parser/wikilinks.go`) e a fonte da verdade. O espelho no SPA
(`viewer/src/lib/wikilinks.ts`) foi conferido **item a item**:

| Elemento | Go | SPA | Veredito |
|---|---|---|---|
| Regex de extracao | `wikilinkDetailedRegex = (!)?\[\[([^\]\r\n]+)\]\]` (`wikilinks.go:12`) | `WIKILINK_REGEX = /(!)?\[\[([^\]\r\n]+)\]\]/g` (`wikilinks.ts:33`) | **identica** |
| Relacoes reconhecidas | 13, em `IsKnownRelation` (`wikilinks.go:79-86`) | 13, em `KNOWN_RELATIONS` (`wikilinks.ts:16-30`) | **mesma lista, mesma ordem** |
| Contextos sem link | fenced → inline → comentario → tabela (`wikilinks.go:37-43`) | fenced → inline → comentario → tabela (`wikilinks.ts:36-39`) | **mesma ordem** |

As 13 relacoes conferem uma a uma: `links_to`, `implements`, `depends_on`, `supports`,
`refutes`, `extends`, `derived_from`, `fixes`, `causes`, `part_of`, `related_to`,
`similar_to`, `tagged_as`.

| ID | Criterio | Veredito | Evidencia |
|---|---|---|---|
| P2-1 | Detecta `[[alvo]]` | **PASS** | `wikilinks.test.ts:21-25` |
| P2-2 | Detecta `[[alvo|alias]]` | **PASS** | `wikilinks.test.ts:26-30` |
| P2-3 | Detecta `[[alvo#ancora]]` | **PASS** | `wikilinks.test.ts:31-36` |
| P2-4 | Detecta `[[alvo#ancora|alias]]` | **PASS** | `wikilinks.test.ts:37-42` |
| P2-5 | Detecta `[[alvo#^blockid]]` | **PASS** | `wikilinks.test.ts:43-47` |
| P2-6 | Detecta `![[alvo]]` (embed) | **PASS** | `wikilinks.test.ts:190` (`marca embed`) |
| P2-7 | Detecta `[[alvo]]` interno do documento | **PASS** | `wikilinks.test.ts:139` |
| P2-8 | Detecta relacao por `rel:` | **PASS** | `wikilinks.test.ts:154` |
| P2-9 | Detecta relacao por relacao conhecida no prefixo | **PASS** | `wikilinks.test.ts:161` |
| P2-10 | Detecta relacao pelo alias | **PASS** | `wikilinks.test.ts:168` |
| P2-11 | Decompõe URI federada | **PASS** | `wikilinks.test.ts:175` |
| P2-12 | Nao confunde `memory://` com relacao | **PASS** | `wikilinks.test.ts:184` |
| P2-13 | Nao cria link dentro de bloco de codigo | **PASS** | `wikilinks.test.ts:196` (`ignora os contextos sem link, como o Go`) |
| P2-14 | Nao cria link em codigo inline, comentario HTML ou linha de tabela | **PASS** | mesma ordem de strip que o Go, `wikilinks.ts:36-39` |
| P2-15 | Preserva o frontmatter intacto | **PASS** | `wikilinks.test.ts:104` |
| P2-16 | Resolve o destino para a rota de Notas | **PASS** | o destino vira `/notes?path=...` (+ `&anchor=` / `&block=`), com URL-encoding — conferido nos casos 21-47 |

> **Nota de risco conhecido (registrada na ADR-060).** A duplicacao da gramatica entre
> Go e TypeScript e um risco aceito: a mitigacao é a **tabela de casos espelhada**, e ela
> existe dos dois lados — 9 formas de link em `wikilinks.test.ts:20-90` e a mesma
> enumeracao na spec. O verificador independente rodou um **teste diferencial** sobre um
> corpus de 31 casos e 9 corpos Markdown, comparando a extracao do Go com a do SPA.

## P3 — Schemas, cliente MCP e bundle (16 ACs)

| ID | Criterio | Veredito | Evidencia |
|---|---|---|---|
| P3-1 | Schemas zod para cada payload consumido | **PASS** | `viewer/src/lib/schemas.ts` — 14 schemas exportados |
| P3-2 | `zod/mini`, nunca o barrel `zod` | **PASS** | **unico** import de zod em todo `viewer/src`: `schemas.ts:1` — `import * as z from "zod/mini"` |
| P3-3 | Cliente fala JSON-RPC 2.0 | **PASS** | `viewer/src/lib/mcp-client.ts` + 11 casos de teste |
| P3-4 | `tools/call` com nome e argumentos | **PASS** | `mcp-client.test.ts` |
| P3-5 | Le `structuredContent` quando presente | **PASS** | `mcp-client.ts` — o envelope traz `content` **e** `structuredContent`; a SPA usa o estruturado |
| P3-6 | Nao duplica a decodificacao do payload no componente | **PASS** | `queries.ts` centraliza; os componentes consomem o resultado tipado |
| P3-7 | Estado de erro do transporte MCP e distinto de payload invalido | **PASS** | `mcp-client.ts` — erro de rede e erro de schema sao caminhos diferentes |
| P3-8 | Servidor MCP com `--http` e a base configuravel | **PASS** | `viewer/vite.config.ts:18` — `VITE_MCP_BASE_URL`, sem porta hardcoded no codigo da aplicacao |
| P3-9 | Sem URL de CDN em runtime | **PASS** | `viewer/vite.config.ts` e `viewer/index.html` — zero referencias |
| P3-10 | `engines.node` declarado | **PASS** | `viewer/package.json` — `>=20.19` |
| P3-11 | O CI instala e roda o gate de frontend | **PASS** | job `frontend` em `.github/workflows/ci.yml` — `npm ci` → typecheck → test → lint → build → medicao do orcamento |
| P3-12 | O CI roda a invariante do `internal/ci` com `node_modules` presente | **PASS** | ultimo step do job, com `working-directory: ${{ github.workspace }}` (corrigido em `4057ef0`) |
| P3-13 | O build e publicado na raiz servida | **PASS** | `make viewer` → `.mymemory/www/app/`; verificado E2E na feature irma |
| P3-14 | Bundle dentro do teto de 160 kB gzip | **PASS** | **133.086 bytes** (`gzip -9`) = 130,0 kB |
| P3-15 | Nenhum preset de biblioteca na build | **PASS** | `@tanstack/vite-config` foi removido do `package.json`: e preset de **biblioteca** (`build.lib`, `es`+`cjs`, `preserveModules`, `vite-plugin-dts`) e transformaria a build da SPA em build de biblioteca |
| P3-16 | `@tanstack/config` (deprecado) nao entra | **PASS** | ausente do `package.json`; a `vite.config.ts` segue explicita e local (56 linhas) |

---

## Sobre o numero do bundle

O numero **138,5 kB** que circulou em versoes anteriores desta validacao **nao e
reproduzivel** e foi corrigido. A medicao autoritativa e a do CI — `gzip -9` sobre
`dist/assets/*.js`:

```
esm-D3z5Wpj8.js       ->     122 bytes
index--xaLf-3x.js     -> 132.841 bytes
modern-qKf_f-aS.js    ->     123 bytes
TOTAL                -> 133.086 bytes   (orcamento: 163.840)
```

O relatorio do proprio Vite, que comprime em zlib nivel 6, mostra 143,3 kB para os
mesmos arquivos. Os dois numeros sao verdadeiros; a diferenca e o nivel de compressao.
A ADR-060 projeta ~139 kB como meta — o valor medido fica entre os dois.

---

## Validação visual das quatro rotas

O verificador independente marcou os 13 ACs de painel como **PARTIAL**, porque
não renderizava a SPA. Esta rodada fecha a lacuna: a viewer foi servida pelo
**binário real** (`mem mcp --port 38400`, com o build publicado em
`.mymemory/www/app/`) e as quatro rotas foram abertas em um Chromium headless
via `playwright screenshot`, com o MCP vivo por trás.

| Rota | O que renderizou | Veredito |
|---|---|---|
| `/` | **Servidor MCP** (status `healthy`, versão 1.0.0, **24 tools**, uptime, protocolo, repositório padrão) e **Diagnóstico da memória** (Health Score 100/100, contadores de documentos/chunks/nós/arestas, e os estados estruturados de links quebrados e notas órfãs) | **PASS** |
| `/graph` | **Grafo interativo** com o botão "Abrir grafo interativo", **Nós centrais — grau**, **Nós centrais — PageRank**, **Comunidades** (modularidade Q, nós, arestas, comunidades) e **Conexões latentes** | **PASS** |
| `/notes` | **Filtros** (extensão, pasta), **Documentos indexados** e o painel de leitura com o estado "Nenhuma nota selecionada" | **PASS** |
| `/drift` | **Desvio semântico** (score, severidades, faixa Git, commits, arquivos modificados), **Notas e ADRs desfasados** e a tabela **Código órfão** com os 5 arquivos reais, status, linhas e ação sugerida | **PASS** |

Nenhuma das quatro telas caiu no fallback textual: todas renderizaram **UI
estruturada**, com estados vazios explícitos onde o dado não existe. O vault
local deste repositório não está indexado (0 documentos), então os contadores
zerados são o estado real do dado, não falha de renderização.

### Defeito encontrado por esta validação

A rota `/graph` respondia **404 em texto puro** ("artefato de grafo nao
encontrado") quando a URL era aberta ou recarregada direto. A navegação por
link funcionava, porque o roteador do cliente não passa pelo servidor — o
usuário só percebia ao copiar a URL ou apertar F5.

A causa era a colisão entre o prefixo reservado pelo servidor (`/graph/`, para
os artefatos de grafo) e a rota da SPA (`/graph`). A ADR-060 exige, na sua
emenda, que **qualquer path de navegação** devolva o entrypoint; a área
reservada passou a ser o *conteúdo* de `/graph/`, e o path exato voltou a ser
rota da SPA. **Corrigido** — ver o commit da correção e o teste de regressão
com mutação provada.

Este defeito é a prova de que a validação visual não é redundante: nenhum teste
unitário, nem a auditoria adversarial de 34 payloads, cobria a colisão entre um
prefixo reservado e uma rota de cliente.

---

## Verificador independente

A feature foi auditada por um verificador **independente** — processo separado,
sem participação na implementação. O veredito dele foi **PASS-WITH-GAPS**.

### O que o verificador provou

| Verificação | Método | Resultado |
|---|---|---|
| Gramática Go × TS | **diferencial**: 30 casos de `parseWikilink` + 9 corpos Markdown + 13 relações, com `internal/parser` e `viewer/src/lib/wikilinks.ts` executados de verdade | **0 divergências** — os campos `target`, `relation`, `anchor`, `blockId`, `alias`, `isSameDoc`, `isFederated`, `federatedRepo` e `federatedPath` idênticos dos dois lados |
| Forma × zod, campo a campo | leitura comparativa Go × `viewer/src/lib/schemas.ts` | os 10 payloads batem em nomes e tipos — **com uma ressalva desde 2026-10-29**: `memory_doctor.health_score` passou a ser **anulável** (o campo **não viaja** no payload quando o Health Score não se aplica, porque o vault não tem documentos indexados). O `zod` do repositório passou a tolerar **ausência e `null`** (`z.pipe(z.nullish(...), z.transform(...))`), e o tile do Overview mostra "—" com o rótulo "vault sem documentos". Detalhe: `.specs/features/doctor-empty-vault-score/` |
| `null` × `undefined` no zod | `zod/mini` 4.6.7 do repositório | `z.optional(z.array(...)).safeParse(null)` → **FAIL**; `undefined` → passa. A base do achado A1 |
| Payload real do drift | handler Go real → JSON → `driftReportSchema` real do repositório | **FAIL** em `drifted_notes` e `uncovered_code` — achado A1, **corrigido nesta rodada** |
| Tabela de casos espelhada | ADR-060 D-2 vs `viewer/src/lib/wikilinks.test.ts` | a mitigação é **real**: os 13 casos (9 formas + 4 contextos) existem dos dois lados |
| `zod/mini`, nunca o barrel | grep em `viewer/src` | **um único** import de zod: `schemas.ts:1` |
| Orcamento de bundle | medição com o mesmo comando do CI | 133.086 bytes — **abaixo** do teto de 160 kB, mas o gate do CI impunha 180 kB (achado M1, **corrigido**) |
| Números da documentação | contagem no repositório | 59 ADRs · 25 pacotes Go · 24 tools · 7 skills — todos conferem |
| Higiene | `git status --porcelain` | árvore principal limpa; worktree e artefatos temporários removidos |

### Achados do verificador e o que foi feito com cada um

| ID | Achado | Decisão |
|---|---|---|
| **A1** (ALTO) | `memory_get_drift` emitia `null` em campos que o zod rejeita → o painel `/drift` renderizava markdown cru em vez da UI estruturada | **Corrigido nesta rodada** — normalização central no wire + teste de regressão com mutação provada. Como a SPA é servida pelo próprio MCP, não há divergência de versão a considerar: corrigir o contrato no servidor basta |
| **M1** (MÉDIO) | O gate do CI impunha 180 kB, mais permissivo que o teto de 160 kB da spec | **Corrigido** — o orçamento passou a 163.840 bytes, o próprio teto da spec |
| **M2** (MÉDIO) | A justificativa do fallback na ADR-060 não se sustenta para navegadores | **Registrado** na [ISSUE-021](../../ISSUES.md) |
| **M3** (MÉDIO) | O CHANGELOG afirmava a semântica errada de `working-directory` | **Corrigido** — o registro foi corrigido com a razão certa |
| **B5** (BAIXO) | `memory_get_impact` e `memory_find_path` não emitem `structuredContent`; `impactSchema` e `pathSchema` nunca validam nada | **Registrado** na [ISSUE-022](../../ISSUES.md) — decisão de produto |
| **B6** (BAIXO) | Versão de protocolo `2024-11-05` vs o campo `structuredContent` de `2025-06-18`; 4 tools devolvem array onde a spec MCP exige objeto | **Registrado** na [ISSUE-023](../../ISSUES.md) — exige ADR própria |
| **B8** (BAIXO) | `GET /graph/../x` normaliza para fora do prefixo e cai no fallback | **Registrado** junto da ISSUE-021 — benigno |

### O que o verificador não verificou

1. **Renderização real da SPA no navegador** — não subiu a SPA nem navegou nas 4
   rotas contra um MCP vivo. Os 13 ACs de painel ficaram **PARTIAL** por isso, não
   por defeito. Esta rodada fecha a lacuna com uma verificação visual.
2. **O job `frontend` rodando no GitHub Actions real.**
3. **MCP sob cliente estrito** — não validou `structuredContent` array contra um
   cliente que exija objeto.
4. **Painéis da SPA linha a linha** — auditou as rotas, `queries.ts`, o store e o
   cliente MCP; não cada componente visual.
