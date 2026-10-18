# ADR-054: Escopo de paridade com o Obsidian é o vault, não o aplicativo

- **Data**: 2026-10-18
- **Status**: Accepted
- **Decisores**: Felipe Miiller
- **Tags**: escopo, produto, vault, obsidian

## Contexto e Problema

O objetivo declarado do My-Memory é ser um **superset das funcionalidades do Obsidian**, com a busca semântica por embeddings como complemento. O CLI oficial do Obsidian (v1.12+, introduzido em 2026-02-27) expõe ~115 comandos. O My-Memory expõe 36 comandos e 20 tools MCP.

A matriz de gap levantada em 2026-10-18, sobre 48 comandos de conteúdo de vault, deu: **3 cobertos · 13 parciais · 32 ausentes**. Os ~45 comandos restantes são de estado do aplicativo (plugins, themes, workspace, bookmarks, sync, `dev:*`, CDP).

Sem uma fronteira explícita, "ser superset do Obsidian" é uma direção sem fim: replicar `dev:cdp` significaria transformar um binário Go headless em um segundo Obsidian, competindo com um aplicativo Electron que já existe e é bom nisso.

## Drivers da Decisão

- Manter o My-Memory headless — sem exigir instância do Obsidian rodando
- Cobrir toda a **superfície de conteúdo do vault**, que é o que um agente realmente consulta
- Não duplicar o aplicativo; competir apenas onde o My-Memory já é superior
- Cada gap deve ter um destino declarado: construir, adiar ou recusar

## Opções Consideradas

- **A — Paridade total, incluindo estado do aplicativo**
- **B — Paridade do vault, com fronteira explícita (escolhida)**
- **C — Sem paridade; manter só o que já existe**

## Resultado da Decisão

Escolhida a **Opção B**. O My-Memory replica a superfície de **conteúdo** do vault e **recusa explicitamente** a superfície de **aplicativo**.

**Escopo de paridade** (a construir, por ordem de custo÷valor): `files`/`folders`, `read`, `wordcount`, `random`, `outline`, `deadends`, `aliases`, `move`/`rename`, `unresolved` isolado, `mem search --tag`, `tasks`, `templates`, `properties`, `bases`, `history`.

**Fora de escopo por decisão**: `plugins*`, `theme*`, `snippets`, `commands`, `hotkeys`, `workspace`, `tabs`, `bookmarks`, `sync`, `dev:*`, `devtools`, `dev:cdp`, `restart`, `reload`, `search:open`, `template:insert`. Requerem uma instância Electron viva ou um protocolo de debug que nenhum módulo do `internal/` fala.

**Superiores já hoje, não a replicar**: `search` (RRF híbrido contra textual), `search:context` (empacotamento com orçamento de tokens), `backlinks`/`links` (triptych), `unresolved`/`orphans` (via `doctor`), `hubs` (PageRank), `clusters` (LPA + modularidade), `impact` (raio de destruição), `drift`, `code-*` (AST tree-sitter).

### Consequências Positivas

- O escopo de cada gap fica decidido, não em aberto
- `mem tags` e `mem search --tag` deixam de prometer compatibilidade literal com o Obsidian (ver [ADR-055](055-semantica-de-tag-consultavel-sem-nos-mortos.md))
- A propriedade de frontmatter ganha dono explícito (ver [ADR-056](056-persistencia-de-propriedades-de-frontmatter.md))

### Consequências Negativas

- Usuários que esperem `obsidian`-paridade de interface vão frustar — o My-Memory não é substituto do editor
- Paridade de conteúdo exige mudança de schema em três frentes (tags, propriedades, mtime), não é só consulta
- A lista de gaps assume dívida de manutenção: sem buildar a lista, a distância do Obsidian só cresce

## Prós e Contras das Opções

### Opção A — Paridade total ❌

- ✅ Literalmente satisfaz a frase "todas as funcionalidades"
- ❌ Exige reimplementar CDP/Electron; `internal/deeplink` só emite URIs `obsidian://`
- ❌ Entrega uma interface pior que a do Obsidian, que é o produto maduro nessa frente

### Opção B — Paridade do vault ✅ Escolhida

- ✅ Cobre o que agente e humano realmente consultam
- ✅ Fronteira defensável e verificável
- ❌ Não é "tudo"; a frase do objetivo precisa ser lida com essa nuance

### Opção C — Sem paridade

- ✅ Zero trabalho
- ❌ Deixa 32 gaps sem destino e contradicts o objetivo declarado

## Links

- [ADR-041](041-issue-009-tag-sem-casa-removida-do-grafo.md) — origem da perda de tag-como-nó
- [ADR-055](055-semantica-de-tag-consultavel-sem-nos-mortos.md) — desdobramento da semântica de tag
- [ADR-056](056-persistencia-de-propriedades-de-frontmatter.md) — maior gap de schema
- [ADR-057](057-mtime-e-metadados-de-arquivo-no-indice.md) — mtime nunca persistido
- Matriz de gap de 2026-10-18: 48 comandos de conteúdo de vault mapeados — **3 cobertos · 13 parciais · 32 ausentes**. Levantado por Verifier sobre o diff `90b5c32..9bc28c7`; registrado em `.specs/ISSUES.md`