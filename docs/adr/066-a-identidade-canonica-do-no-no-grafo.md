# ADR-066: A identidade canônica do nó no grafo é o id do documento

- **Date**: 2026-11-16
- **Status**: Accepted — Implementado
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: graph, node-identity, documents-id, wikilinks, indexer, doctor, adr-055, adr-027
- **Adenda**: [ADR-027](027-descoberta-de-rotas-e-caminho-minimo-no-grafo.md) (a rota) · [ADR-055](055-semantica-de-tag-consultavel-sem-nos-mortos.md) (`documents.id` como referência estável, título como rótulo) · [ADR-063](063-o-doctor-distingue-link-rompido-de-link-fora-do-indice.md) (o doctor) · [ADR-013](013-higiene-de-grafo-pruning-e-doctor.md) (higiene de grafo)
- **Spec**: `.specs/features/fix-graph-node-identity/` · **Issues**: [ISSUE-047](../.specs/ISSUES.md) · **Origem**: FIX-2 da validação da spec `graph-path-discovery` (`.specs/features/graph-path-discovery/validation.md`)

## Context and Problem Statement

O indexador dava **duas identidades** a cada nota: o lado **origem** da aresta era
gravado com o **id de path** do documento (`documents.id`) e o lado **destino** com o
**id de título** (o texto do wikilink). Toda aresta atravessava essa fenda — e o
`ResolveNodeCanonicalID`, que resolve um nome informal para o id canônico, caía
justamente do lado que **nenhuma** aresta conectava.

**Medido no vault real (`repo_central`, SQL somente-leitura, 2026-11-15):**

| Métrica | Valor |
| :--- | ---: |
| arestas em `graph_edges` | **40.243** |
| arestas com **as duas** pontas em id de path | **0** |
| nós em `graph_nodes` | **3.601** (401 com id de path · 3.200 de estilo título) |
| grau do `README` por id de path | **3** |
| grau do `README` por id de título | **66** |

O mesmo nó, dois graus. Reprodução:

```
mem path "CONTRIBUTING.md" "README" --storage=postgres --repo repo_e4c8f3b1a2d5
→ ❌ Não encontrado
```

A causa é **anterior** à feature `graph-path-discovery` (ADR-027) e a ISSUE-047 é
**issue separada** porque a correção toca o **indexador** (a gravação da aresta) e o
**resolvedor canônico** — e a decisão "qual identidade é a canônica" é de
**arquitetura**, não de bug.

## Decision Drivers

- **DR-1 — Uma identidade por documento.** Um documento indexado ocupa **um** nó no
  grafo. Duas identidades fazem o mesmo README ter grau 3 e grau 66.
- **DR-2 — A identidade é estável.** A chave do nó não pode depender do frontmatter:
  mudar o título de uma nota não pode trocar a identidade do seu nó (e nem quebrar as
  arestas que apontam para ele).
- **DR-3 — O conserto é na origem.** Canonicalizar na leitura (em `FindPath`) é
  remendo: deixaria as arestas atravessadas no grafo e os consumidores continuariam
  discordando sobre o id do mesmo documento.
- **DR-4 — O link rompido continua visível.** O doctor precisa enxergar o alvo que
  não resolve (ADR-063): corrigir a identidade não pode cegar a detecção de link
  morto.
- **DR-5 — Sem mudança de schema.** Trocar a coluna `documents.id` exigiria backfill
  de `chunks.document_id`, `document_tags.resolved_doc_id` e da ponte de código nos
  dois engines.
- **DR-6 — Os dois engines contam a mesma história.** O vault de produção é
  PostgreSQL; corrigir só o SQLite deixa o defeito vivo onde ele importa.

## Considered Options

- **A — `documents.id` (id de path) como identidade canônica, título como rótulo, resolução na gravação (escolhida)**
- **B — O título como identidade canônica**
- **C — Canonicalizar na leitura (`FindPath`)**
- **D — Hash novo como identidade do nó**

## Decision Outcome

Escolhida a **Opção A**. A identidade do nó no grafo passa a ser o **`documents.id`**
do documento — o id de path, a única referência estável do schema (ADR-055).

| Elemento | Decisão |
| :--- | :--- |
| **Identidade do nó** | `documents.id` (o id de path) |
| **Rótulo do nó** (`graph_nodes.name`) | `documents.title` |
| **Aliases do mesmo nó** | o **path**, o `title:` e o `alias:` do frontmatter |
| **Onde o wikilink é resolvido** | **na gravação da aresta** (indexador), não na leitura |
| **Alvo não resolvido** | permanece o **texto do wikilink** (o doctor continua vendo o link rompido) |
| **Correção do dado já gravado** | exige **reindex** do vault |

Consequências imediatas:

- **As duas pontas da aresta** passam a ser id de documento: uma aresta entre dois
  documentos indexados não atravessa mais a fenda.
- **O nó do documento** tem `id` = `documents.id` e `name` = `documents.title` — o
  título vira rótulo humano e o id vira chave. A rota do `mem path` passa a exibir
  títulos, não paths.
- **O resolvedor canônico** (`ResolveNodeCanonicalID`, nos dois engines) trata path,
  `title:` e `alias:` como **aliases do mesmo nó** e devolve **sempre** o
  `documents.id` — nunca o id de um nó de estilo título.
- **O `alias:`** é resolvido contra o **disco** do vault (reaproveitando o
  `internal/vaultfs`), e só quando a raiz do vault é conhecida: sem raiz, o alias
  devolve o erro "nó '...' não encontrado no grafo".
- **Os nós legados de id de título** (os 3.200 do cofre central) são o outro lado
  da fenda: enquanto existirem, o resolvedor continua devolvendo o id de título. O
  `mem doctor --fix` os remove — são nós **sem grau** que não são `documents.id` nem
  `code_symbol`.
- **A correção exige reindex.** Não há migração possível para 40.243 arestas: o
  grafo é derivado do disco. A mudança de código só vale para o que for indexado
  depois dela — o procedimento de reparo está em `docs/CLI_GUIDE.md`.

**Fora do escopo desta decisão:** o nó de código (`code_symbol`, id
`code:<repo>:<qualified_name>`) — não é um documento e tem id próprio (ADR-061); a
aresta `references_code` continua apontando para o símbolo. E o motor de
pathfinding (`internal/graph/path.go`) — o motor está correto, o defeito estava
**antes** dele.

### Positive Consequences

- Cada documento indexado ocupa **uma** identidade de nó: o `README` deixa de ter
  grau 3 e grau 66.
- `mem path` entre duas notas ligadas por wikilink passa a **achar a rota** (1 salto).
- Todo consumidor do id canônico (`mem path`, `mem impact`, `mem inspect`, a ponte
  `documents` ↔ `code_symbols`, o `mem doctor`) fala a **mesma** língua.
- O id do nó sobrevive à renomeação do arquivo e à edição do título.

### Negative Consequences

- O **dado já gravado** não se corrige sozinho: o vault precisa de reindex (e o
  reindex é operação do dono, sobre o vault de produção).
- O `alias:` só resolve com a **raiz do vault** conhecida: sem ela, o alias é
  "não encontrado" — e a honestidade do erro é melhor que um id adivinhado.
- O `graph_nodes.name` deixa de ser o id para o nó de documento: quem lia o nome
  esperando o path passa a ler o **título** (que é o rótulo).

## Pros and Cons of the Options

### Opção A ✅ Chosen

- ✅ Uma identidade por documento, estável e já referenciada pelo schema (DR-1, DR-2).
- ✅ Conserta **todos** os consumidores de uma vez, porque corrige na gravação (DR-3).
- ✅ Preserva a detecção de link rompido (DR-4).
- ✅ Não exige backfill nem mudança de schema (DR-5).
- ✅ Mesmo contrato nos dois engines (DR-6).
- ❌ O vault já indexado precisa de reindex para valer.

### Opção B — o título como identidade canônica

- ✅ O rótulo e a chave viram a mesma coisa: o nome do nó já é o título.
- ❌ Inverte o padrão da ADR-055 e torna o id dependente do frontmatter: mudar o
  título de uma nota troca a identidade do nó e deixa as arestas órfãs (DR-2).
- ❌ Títulos colidem entre pastas (o cofre central tem 7 `README`): o id do nó
  deixaria de ser único.
- ❌ Quebraria a ponte `documents` ↔ `code_symbols`, que usa `documents.id`.

### Opção C — canonicalizar na leitura

- ✅ Não exige reindex: o `mem path` passaria a funcionar no dado antigo.
- ❌ Deixa as arestas atravessadas no grafo (o `README` continua com dois graus) e
  os consumidores continuam discordando sobre o id do mesmo documento (DR-3).
- ❌ Esconde o defeito em vez de corrigir: o doctor e o drift continuariam vendo
  um grafo partido.

### Opção D — hash novo como identidade do nó

- ✅ Identidade estável e independente do caminho do arquivo.
- ❌ Introduz uma terceira identidade no schema e exige backfill de `chunks`,
  `document_tags` e da ponte de código nos dois engines (DR-5).
- ❌ A coluna `documents.id` continua sendo a referência estável — o hash seria um
  segundo critério para a mesma coisa.

## Links

- [ADR-027](027-descoberta-de-rotas-e-caminho-minimo-no-grafo.md) — a rota; o motor
  está correto, o defeito estava antes dele.
- [ADR-055](055-semantica-de-tag-consultavel-sem-nos-mortos.md) — `documents.id` como
  referência estável e o título como rótulo: a base desta decisão.
- [ADR-063](063-o-doctor-distingue-link-rompido-de-link-fora-do-indice.md) — o doctor
  que enxerga o link rompido, que esta decisão preserva.
- [ADR-013](013-higiene-de-grafo-pruning-e-doctor.md) — a higiene de grafo, onde o
  prune dos nós legados mora.
- [ISSUE-047](../.specs/ISSUES.md) · FIX-2 da validação da spec
  `graph-path-discovery` (`.specs/features/graph-path-discovery/validation.md`).
- Spec: `.specs/features/fix-graph-node-identity/` ·
  implementação: `cmd/mem/main.go`, `internal/watcher/indexer.go`,
  `internal/db/graph.go`, `internal/db/store.go`, `internal/store/postgres.go`,
  `internal/store/doctor.go`.
