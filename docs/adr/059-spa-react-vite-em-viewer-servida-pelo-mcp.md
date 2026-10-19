# ADR-059: SPA React/Vite em `viewer/`, compilada para estático e servida pelo MCP

- **Data**: 2026-10-19
- **Status**: Accepted
- **Decisores**: Felipe Miiller
- **Tags**: viewer, react, vite, mcp, http, build-estatico
- **Relacionada**: [ADR-054](054-escopo-de-paridade-com-o-obsidian.md), [ADR-021](021-servidor-mcp-com-transporte-http-sse.md), [ADR-037](037-rewrite-viewer-com-vite-vanilla-ts.md) (obsoleta), [ADR-038](038-viewer-site-unico-dataset-fixo.md) (obsoleta)

## Contexto e Problema

O transporte HTTP do MCP (`internal/mcp/http_server.go`) é um `http.ServeMux` que expõe quatro rotas — `/health`, `/sse`, `/message`, `/mcp` — e **nenhuma serve arquivo estático**. Hoje `memory_visualize_graph` e `mem graph view` devolvem um **caminho de arquivo**, que para um agente não é acionável: a sessão pode rodar em outra máquina, num contêiner, atrás de um filesystem diferente.

Ao mesmo tempo, o My-Memory já calcula métricas que só existem como texto no terminal: Health Score, contagem de órfãs, dead links, modularidade de Newman-Girvan, PageRank, drift. Nada as apresenta.

O repositório já tentou um viewer antes. `b477200` removeu 112 arquivos e 27.858 linhas de Electron + React + Vite + Forge + Playwright.

## Drivers da Decisão

- Dar ao agente **URL clicável** em vez de caminho de arquivo
- Apresentar as métricas que o `mem doctor` já calcula
- Não repetir `b477200`: **nenhum build step entra no caminho de release do binário Go**
- Preservar a garantia central do projeto: tudo local, tudo offline

## Opções Consideradas

- **A — SPA React/Vite em `viewer/`, build estático, servida pelo MCP** (escolhida)
- **B — Servir só o HTML gerado pelo `graphview`, sem SPA**
- **C — Trazer de volta o viewer Electron**

## Resultado da Decisão

Escolhida a **Opção A**.

```
viewer/           fonte React 19 + Vite 8 + TS 6 + Tailwind 4 + shadcn
  └─ npm run build
       └─ viewer/dist/     ARTEFATO DE BUILD — não versionado
            └─ servido pelo MCP HTTP sob uma subpasta dedicada
```

O scaffold é determinístico e não-interativo:

```
npx shadcn@latest init -t vite --no-monorepo -b base -p nova -n mymemory-dashboard -y
```

Bundle atual: **259,52 kB JS (84,28 kB gzip)**, 22,20 kB CSS, 5 arquivos woff2 da Geist.

### Por que isto não é `b477200`

| | `b477200` (revertido) | `viewer/` (esta decisão) |
|---|---|---|
| Runtime | Electron, app distribuído | Estático, servido por HTTP |
| Build step no release | sim, Vite + Forge | **não** — o binário Go não depende do build da SPA |
| Comunicação | IPC streaming | HTTP + MCP |
| Estado | app local com janela | arquivos served sob subpasta restrita |
| Tam depended de daemon | sim (`mymemoryd`) | **não** — o lifecycle do supervisor está inalcançável (`cmd/mymemoryd/main.go:92` cria o Manager com `nil`) |

O `viewer/` é **painel de métricas sobre dados já calculados**, não editor. A ADR-054 declara plugins, themes e `dev:*` fora de escopo porque exigem instância Electron viva; nada aqui exige.

### O que muda no repositório

1. O repo deixa de ter **zero dependência Node**. `viewer/node_modules/` entra no working tree, já filtrado pelo `viewer/.gitignore` próprio.
2. `go build ./...` **não é afetado** — Go ignora diretórios sem `.go`. `internal/ci` continua válido porque nenhum pacote Go novo aparece em `go list ./...`.
3. O transporte HTTP do MCP passa a servir uma subpasta dedicada. **Nunca a raiz do vault.** Isso é a mudança de superfície real, e a [ADR-058](058-renderizador-vis-network-vendorizado-sem-build-step.md) mais a spec de `mcp-static-serving` especificam o escopo e o teste de path traversal.

### Decisões de execução

| Decisão | Escolha | Racionale |
|---|---|---|
| `viewer/dist/` versionado? | **Não** | É artefato de build. Versioná-lo duplica 21 arquivos a cada `npm run build` e cria conflito de merge binário |
| Como gerar | `make viewer` (ou `npm run build` em `viewer/`) | O MCP degrada com elegância: se `dist/` não existe, diz que não existe |
| Nome no `package.json` | `mymemory-dashboard` | Nome do pacote npm; independente do nome da pasta. Consistente o suficiente para não justificar churn |
| Where a SPA vive | Dentro do repo, em `viewer/` | Compartilha histórico e um único `git clone` entrega tudo; `.gitignore` por pasta isola o peso |

### Consequências Positivas

- Agente recebe URL, não caminho
- Métricas de grafo e saúde finally viram visuais
- Frontend iterável com hot reload, contra 38 KB de HTML inline que ninguém maintém
- `viewer/dist/` é artefato limpo: trocar de framework é apagar uma pasta

### Consequências Negativas

- **O repo passa a exigir Node para desenvolver o viewer** (não para usar o binário). Dois toolchains no mesmo repositório
- `viewer/dist/` ausente numa clone nova significa MCP sem painel — degradação silenciosa se não for tratada
- Nenhuma proteção de CSP/auditoria de dependência frontend; a ADR-050 (threat model) não cobre esse caminho
- O `internal/ci` invariant não cobre `viewer/` — uma pasta Node não tem testes no pipeline Go

## Links

- Spec da integração: `.specs/features/mcp-static-serving/spec.md`
- Spec do renderizador de grafo: `.specs/features/graph-view-vis-engine/spec.md`
- [ADR-021](021-servidor-mcp-com-transporte-http-sse.md) — o transporte que ganha a rota estática
- [ADR-054](054-escopo-de-paridade-com-o-obsidian.md) — a fronteira de escopo que esta decisão **não** atravessa