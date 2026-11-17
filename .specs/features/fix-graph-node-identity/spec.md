# Feature: fix-graph-node-identity

- **Spec ID**: `fix-graph-node-identity`
- **Created**: 2026-11-16
- **Owner**: Felipe Miiller
- **Source**: ISSUE-047 — `.specs/ISSUES.md` (seção `## ISSUE-047`) · achado original em `.specs/features/graph-path-discovery/validation.md` (**FIX-2**, § Fix Plans)
- **Related**: ADR-027 (descoberta de rotas e caminho mínimo) · **ADR-055** (`documents.id` como referência estável, título como rótulo) · ADR-063 / ISSUE-039 (doctor: link rompido × fora do índice) · ADR-061 (ponte notas↔código) · ADR-041 (tag sem casa não vira nó) · ISSUE-046 (o motor de caminho está correto) · spec `fix-graph-path-discovery-gaps` (os demais gaps da feature)

## Problem Statement

O comando `mem path` **não acha caminho nenhum** no vault real, mesmo entre duas notas
ligadas por wikilink. A feature `graph-path-discovery` (ADR-027) está entregue e
verificada — e ainda assim o comando principal dela não funciona no dado real
(ISSUE-047, severidade 🔴 **critical**).

A causa não é o motor de caminho: é a **identidade do nó**. O indexador dá **duas
identidades** a cada nota:

| Lado da aresta | Identidade gravada | Onde |
| :--- | :--- | :--- |
| **origem** (source) | **id de path** do documento (`documents.id`) | `cmd/mem/main.go:1941` (Postgres) e `cmd/mem/main.go:2184` (SQLite) |
| **destino** (target) | **id de título** (o texto do wikilink) | as mesmas duas linhas: `InsertEdgeWithProps(ctx, …, docID, edge.Target, …)` |

Toda aresta **atravessa essa fenda**. E o `ResolveNodeCanonicalID` — que resolve um
nome informal para o id canônico — cai justamente do lado que **nenhuma** aresta conecta
(`internal/db/graph.go:582` resolve em `documents`, cujo `id` é o path;
`internal/store/postgres.go:1018` faz o mesmo no Postgres).

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

A causa é **anterior** à feature (`171df1e`, um dia antes do commit `9d3e0ee` da feature) —
e a ISSUE-047 é **issue separada** porque a correção toca o **indexador** (a gravação da
aresta) e o **resolvedor canônico**, e a decisão "qual identidade é a canônica" é de
**arquitetura**.

Esta spec decide essa identidade, corrige a gravação na origem, faz o resolvedor tratar
path, `title:` e `alias:` como **aliases do mesmo nó**, dá o procedimento de reparo do
vault já indexado e a **medição** que prova o reparo. A correção dos demais gaps da
feature (resolução ambígua entre nós de mesmo nome, guarda do ramo de nó idêntico,
asserções de detalhe da aresta, cobertura Postgres, mensagens e defaults) é da spec
`fix-graph-path-discovery-gaps` (ISSUE-046).

## Goals

- [x] Cada documento indexado ocupa **uma** identidade de nó no grafo: o seu
      `documents.id`, com o `title:` como rótulo (`graph_nodes.name`).
- [x] Toda aresta entre dois documentos indexados tem **as duas** pontas em id de
      documento — 0 arestas atravessadas.
- [x] O resolvedor canônico trata path, `title:` e `alias:` como **aliases do mesmo nó**.
- [x] `mem path` entre duas notas ligadas por wikilink **acha a rota** (1 salto), e
      nenhum consumidor (`mem impact`, `mem inspect`, a ponte de código, o `mem doctor`)
      fica com a identidade antiga.
- [x] O wikilink **rompido** continua visível (o alvo não indexado não vira um id de
      documento) — a detecção de link morto não regride.
- [x] A decisão fica registrada em **ADR nova** (a identidade do nó é arquitetura).
- [x] O vault já indexado tem **procedimento de reparo** e **query de verificação** — o
      código novo, sozinho, não conserta o dado antigo.

## Out of Scope

| Item | Reason |
| :--- | :--- |
| Corrigir os demais gaps da feature `graph-path-discovery` | São da ISSUE-046 / spec `fix-graph-path-discovery-gaps`: resolução ambígua entre nós de mesmo nome (7 `README`), guarda do ramo de nó idêntico (M7), asserções de `EpistemicStatus`/`Weight`, cobertura Postgres do `FindPath`, asserção das mensagens e o `DefaultPathOptions()` morto. |
| Mexer no **motor** de pathfinding (`internal/graph/path.go`) | O motor está **correto**: o Verifier matou 6/7 mutantes e confirmou que o pruning por dominância, o modo `hops` e a inversão epistêmica se comportam como a ADR-027 pede. O defeito está **antes** dele. |
| Trocar a coluna `documents.id` por um hash | A ADR-055 já escolheu `documents.id` como **a** referência estável do schema. Trocá-la exigiria backfill de `chunks.document_id`, `document_tags.resolved_doc_id` e da ponte de código, nos dois motores. Esta spec **usa** a coluna como ela é. |
| Persistir `aliases:` do frontmatter no índice | A ADR-056 (tabela `document_properties`) **não foi implementada** no código. Persistir exigiria migration em dois motores — é outra feature. O alias é resolvido contra o **disco**, reaproveitando o `internal/vaultfs` da spec `fix-doctor-unindexed-links`. |
| Mexer no nó de código (`code_symbol`, id `code:<repo>:<qualified_name>`) | Não é um documento: tem id próprio por ADR-061. A aresta `references_code` continua apontando para o símbolo de código. |
| Reescrever o conteúdo do cofre (títulos, wikilinks, notas stub) | Decisão de **conteúdo** do dono. A correção é de código. |
| Unificar o `documents.id` com o `title` no banco | Isso é a ISSUE-010, já resolvida: `documents.id` é o id de path e `documents.title` é o título do frontmatter. A correção **usa** essa identidade. |

---

## Assumptions & Open Questions

| Assumption / decisão | Default escolhido | Rationale | Confirmado? |
| :--- | :--- | :--- | :--- |
| **Qual identidade é a canônica** | a coluna **`documents.id`** — o id de path do documento (`cmd/mem/main.go:1868`, `docID := path`) | É a **única** referência estável do schema (ADR-055: "`documents.id` é a única referência estável do schema, e é o que permite consultar por tag sem depender do texto"), e é a identidade que o lado **origem** de toda aresta já usa e que o `ResolveNodeCanonicalID` já devolve (`internal/db/graph.go:594-603`). | y — **decidido** |
| **Alternativa rejeitada: id canônico = id de título** | **não** adotado | Inverte o padrão da ADR-055, torna o id dependente do frontmatter (muda a cada edição de título) e muda o schema do grafo — as arestas passariam a ser voláteis. Registrada como alternativa, não como default. | n |
| **Onde** o wikilink é resolvido | na **gravação** da aresta (indexador), e não na leitura (`FindPath`) | Corrigir na origem conserta **todos** os consumidores (path, doctor, drift, impact), não só o caminho. Canonicalizar na leitura é remendo: deixaria as arestas atravessadas no grafo e o `README` com dois graus (3 e 66). | y — **decidido** |
| **O rótulo do nó** | `graph_nodes.name` = `documents.title`; o id de path é a **chave** e o **alias** | O título é o rótulo humano e o id é a chave estável (ADR-055). Sem o rótulo, a rota do `mem path` passaria a exibir paths em vez de títulos — uma regressão de leitura causada pela própria correção. | y — **decidido** |
| **Wikilink rompido** (alvo não indexado) | o alvo **permanece o texto do wikilink** | É assim que o `mem doctor` reconhece o link rompido (ADR-063 / ISSUE-039: 13 conceitos sem nota + 125 referências fora do índice no cofre central). Se o alvo virasse um id de documento, o doctor perderia a detecção. O conserto não pode cegar o doctor. | y — **decidido** |
| **Resolução do alias de frontmatter** | contra o **disco** do vault, reaproveitando `internal/vaultfs`, e só quando a raiz do vault é conhecida | `aliases:` não é persistido no índice (a ADR-056 não foi implementada) e persistir exigiria migration em dois motores — fora do escopo. O doctor já provou que a resolução em disco funciona (112 por basename + 13 por `title:` no cofre central). | y — **decidido** |
| **Escopo do conserto**: quais relações de aresta | as arestas **documento→documento** (`links_to` e afins de wikilink) e as `tagged_as` que casam com um documento | As arestas `references_code` (ADR-061) apontam para **símbolos de código**, que são nós legítimos de outro tipo — mexer nelas seria errado. As `tagged_as` já resolvem por título (ADR-041); a correção só troca o **id** gravado, e a tag sem casa continua removida. | y — **decidido** |
| **Conserto vale para os dois motores** | sim — SQLite e Postgres, com o **mesmo** alvo gravado | A duplicação existe nos dois indexadores (`cmd/mem/main.go:1941` e `:2184`, `internal/watcher/indexer.go:119` e `:267`) e o vault de produção é Postgres: corrigir só um deixa o defeito vivo no engine que importa. | y — **decidido** |
| **Os 3.200 nós legados de id de título** | o `mem doctor --fix` os remove (nós sem grau que não são `documents.id` nem `code_symbol`) | Eles são o **outro lado** da fenda: enquanto existirem, o passo 2 do resolvedor (`graph_nodes` por nome, `internal/db/graph.go:605-613`) continua devolvendo o id de título e o defeito volta — mesmo com a gravação corrigida. O prune fica na higiene do grafo (ADR-013), que já é do doctor. | y — **decidido** |
| **O vault antigo é corrigido por reindex** | sim — `mem index` no vault, depois `mem doctor --fix` e a query de verificação | Não há migração possível para 40.243 arestas: o grafo é derivado do disco. O reindex é a operação que reconstrói o grafo a partir das notas — e é **por isso** que a prova da correção só fecha depois dele. | y — **decidido** |
| **Quem** executa o reindex do cofre central | o **dono**, fora do ciclo de código | É uma operação de dado no vault de produção (40.243 arestas, 3.601 nós). A spec entrega o procedimento, a query e o gate; a execução é decisão do dono. | y — **decidido** |
| **O ADR é novo** | sim — ADR-066, "a identidade canônica do nó no grafo" | A ADR-027 fixa a **rota** e a ADR-055 a **referência do documento**; nenhuma delas decide a **identidade de nó** no grafo. A regra do repo (`AGENTS.md`, item 3) é: mudança de arquitetura → ADR. | y — **decidido** |

**Open questions**: none — a decisão de arquitetura ("qual identidade é a canônica") está
tomada acima, com default, rationale e a alternativa rejeitada registradas.

---

## User Stories

### P1: A aresta aponta para a identidade canônica ⭐ MVP

**User Story**: Como consumidor do grafo, quero que a aresta entre duas notas aponte
para o **id do documento de destino**, para que o caminho entre elas exista de fato.

**Why P1**: é a causa raiz da ISSUE-047. Hoje o destino é o texto do wikilink, e 0 das
40.243 arestas do vault têm as duas pontas no mesmo lado.

**Acceptance Criteria**:
1. WHEN the indexer stores a wikilink edge whose target resolves to an indexed document THEN the edge target SHALL be the `documents.id` of that document and not the wikilink text.
2. The source of every edge stored by the indexer SHALL be the `documents.id` of the originating document.
3. IF the target of a wikilink resolves to no indexed document THEN the edge target SHALL remain the wikilink text so that the dangling link stays visible to `mem doctor`.
4. The SQLite indexer and the PostgreSQL indexer SHALL store the same edge target for the same document and the same wikilink.
5. The graph SHALL hold exactly one node per indexed document and that node's `id` SHALL be the document's `documents.id`.
6. The graph node of an indexed document SHALL carry the document's `title` in `graph_nodes.name`.

**Independent Test**: indexar duas notas ligadas por `[[Nota B]]` → a aresta gravada tem
`documents.id` como destino (e não `"Nota B"`), com as duas pontas em id de documento;
indexar uma nota com `[[Nota Que Não Existe]]` → o destino continua `"Nota Que Não Existe"`.

---

### P2: O resolvedor canônico aceita as três formas do mesmo nó

**User Story**: Como usuário do `mem path`, quero digitar o path, o título ou o alias de
uma nota e chegar ao **mesmo** nó, para que eu não precise saber como o índice guarda a
identidade.

**Why P2**: o resolvedor é a outra metade da fenda: hoje ele resolve o path para o id de
documento e o título para o nó de título — a mesma nota, dois ids.

**Acceptance Criteria**:
7. WHEN the same document is queried by its path, by its `title:` and by its frontmatter `alias:` THEN the resolver SHALL return the same `documents.id` in all three cases.
8. IF the vault root is unknown and the query names a frontmatter alias THEN the resolver SHALL return the "node not found" error already used by `mem path`, `mem impact` and `mem inspect`.
9. WHEN the query names the title of an indexed document THEN the resolver SHALL return that document's `documents.id` and SHALL NOT return the id of a title-style node.

**Independent Test**: um vault de teste com uma nota que declara `title:` e `alias:` —
as três consultas devolvem o mesmo id.

---

### P3: Nenhum consumidor fica com a identidade antiga

**User Story**: Como agente que consulta o grafo, quero que `mem path`, `mem impact`,
`mem inspect`, a ponte `documents` ↔ `code_symbols` e o `mem doctor` falem com o id
canônico, para que a correção do grafo não deixe nenhum consumidor para trás.

**Why P3**: o blast radius da ISSUE-047 é maior que o `mem path`: todo consumidor que
resolve nó por id canônico é afetado pela mudança da identidade.

**Acceptance Criteria**:
10. WHEN two indexed notes are joined by a wikilink edge THEN `mem path <origem> <destino>` SHALL report the route as found with 1 hop.
11. The canonical id returned by `mem impact`, `mem inspect` and `mem path` for the same document SHALL be the same value.
12. The `references_code` bridge SHALL keep `source_id` equal to the `documents.id` of the note that references the code.
13. The `mem doctor` report SHALL NOT list an edge whose target is an indexed document as a dead link.
14. The `mem doctor` report SHALL expose the count of graph nodes that are neither a `documents.id` nor a `code_symbol` node.

**Independent Test**: no vault de teste, `mem path` acha a rota de 1 salto e o `mem
doctor` não acusa nenhuma das arestas de documento.

---

### P4: A reparação do vault já indexado e a prova no vault real

**User Story**: Como dono do cofre central, quero um **procedimento**, uma **query** e a
**medição** que provem que o reparo aconteceu, para que eu reindexe com confiança e
saiba medir o resultado.

**Why P4**: o código novo não conserta o dado já gravado — sem reindex, o `mem path` do
vault real continua sem achar caminho, e a medição continua em 0 de 40.243.

**Acceptance Criteria**:
15. WHEN the central vault `repo_central` is reindexed after this correction THEN the number of edges whose two endpoints are document path ids SHALL equal the number of edges whose target resolves to an indexed document.
16. WHEN the central vault `repo_central` is reindexed and `mem doctor --fix` runs THEN the number of graph nodes that are neither a `documents.id` nor a `code_symbol` node SHALL be 0.
17. WHEN the repository vault is reindexed THEN `mem path "CONTRIBUTING.md" "README.md" --undirected` SHALL report the route as found.
18. The canonical node identity SHALL be recorded in a new ADR registered in `docs/adr/README.md`.

**Independent Test**: a query de verificação (count de arestas atravessadas) rodada
antes e depois do reindex, no Postgres do cofre central.

---

## Edge Cases

- IF o wikilink aponta para um documento **com o mesmo basename em outra pasta** THEN a
  resolução é a mesma do resto do repo (ADR-063): **match exato**, sem fuzzy — e o índice
  que resolve é o do mesmo vault.
- IF o documento de destino existe mas **não está indexado** (ainda não passou pelo
  `mem index`) THEN ele é tratado como não resolvido: o alvo fica com o texto do
  wikilink e o doctor o aponta.
- IF a nota tem **dois** documentos com o mesmo título no vault THEN a resolução usa o
  mesmo critério do `ResolveNodeCanonicalID` (id exato → título → basename); o desempate
  entre candidatos de mesmo nome é da spec `fix-graph-path-discovery-gaps` (ISSUE-046).
- IF o grafo **já indexado** não é reindexado THEN o defeito permanece — a mudança de
  código só vale para o que for indexado depois dela. Por isso o reindex é uma etapa
  explícita, com query de verificação.
- IF o reindex altera o Health Score do cofre central THEN a comparação antes/depois do
  `mem doctor` faz parte do procedimento — o doctor resolve alvos por documento, e a
  mudança de identidade pode alterar a contagem de links.
- IF o grafo ainda contém nós legados de id de título THEN o `mem doctor --fix` os
  remove (nós sem grau que não são `documents.id` nem `code_symbol`) — enquanto existirem,
  o passo 2 do resolvedor continua devolvendo o id de título.
- IF a nota tem `title:` no frontmatter THEN o rótulo do nó é esse título e o
  `documents.title` gravado é o mesmo (`cmd/mem/main.go:1863-1867`).
- IF a nota é indexada pela indexação cirúrgica (watcher) THEN ela também grava as duas
  pontas no id canônico — o watcher não pode reintroduzir a fenda.

---

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Gap | Where | Status |
| :--- | :--- | :--- | :--- | :--- |
| GNI-01 | AC1 — o alvo resolvido vira o `documents.id` do documento | FIX-2 | `cmd/mem/main.go` · `internal/watcher/indexer.go` | Complete |
| GNI-02 | AC2 — a origem gravada é o `documents.id` do documento de origem | FIX-2 | `cmd/mem/main.go` · `internal/watcher/indexer.go` | Complete |
| GNI-03 | AC3 — alvo não resolvido continua sendo o texto do wikilink | FIX-2 | `cmd/mem/main.go` · `internal/watcher/indexer.go` | Complete |
| GNI-04 | AC4 — SQLite e Postgres gravam o mesmo alvo | FIX-2 | `cmd/mem/main.go` · `internal/watcher/indexer.go` | Complete |
| GNI-05 | AC5 — uma identidade de nó por documento | FIX-2 | `internal/db/store.go` · `internal/store/postgres.go` | Complete |
| GNI-06 | AC6 — o nó do documento carrega o `title:` como rótulo | FIX-2 | `internal/db/store.go` · `internal/store/postgres.go` | Complete |
| GNI-07 | AC7 — path, `title:` e `alias:` do mesmo documento devolvem o mesmo id | FIX-2 | `internal/db/graph.go` · `internal/store/postgres.go` · `internal/vaultfs/target.go` | Complete |
| GNI-08 | AC8 — sem raiz do vault, o alias devolve o erro de nó não encontrado | FIX-2 | `internal/db/graph.go` · `internal/store/postgres.go` | Complete |
| GNI-09 | AC9 — título resolve para o `documents.id`, nunca para um nó de id de título | FIX-2 | `internal/db/graph.go` · `internal/store/postgres.go` | Complete |
| GNI-10 | AC10 — `mem path` acha a rota de 1 salto entre notas ligadas por wikilink | FIX-2 | `cmd/mem/path.go` · `internal/db/graph.go` · `internal/store/postgres.go` | Complete |
| GNI-11 | AC11 — `mem impact`, `mem inspect` e `mem path` devolvem o mesmo id canônico | FIX-2 | `internal/db/graph.go` · `internal/store/postgres.go` | Complete |
| GNI-12 | AC12 — a ponte `references_code` mantém `source_id` = `documents.id` | FIX-2 | `internal/db/code_bridge.go` | Complete |
| GNI-13 | AC13 — o doctor não acusa aresta de documento como dead link | FIX-2 | `internal/db/graph.go` · `internal/store/doctor.go` | Complete |
| GNI-14 | AC14 — o doctor expõe a contagem de nós fora de `documents.id`/`code_symbol` | FIX-2 | `internal/db/graph.go` · `internal/store/doctor.go` | Complete |
| GNI-15 | AC15 — no cofre central reindexado, as arestas divididas são 0 | FIX-2 | indexação em lote + medição SQL em `repo_central` | Complete |
| GNI-16 | AC16 — no cofre central reparado, os nós de id de título são 0 | FIX-2 | `mem doctor --fix` + medição SQL em `repo_central` | Complete |
| GNI-17 | AC17 — `mem path "CONTRIBUTING.md" "README.md" --undirected` acha a rota | FIX-2 | `cmd/mem/path.go` · vault do repositório | Complete |
| GNI-18 | AC18 — a identidade canônica fica em ADR nova | FIX-2 | `docs/adr/README.md` | Complete |

**Coverage**: 18 total, 18 mapeados, 0 unmapped.

---

## Success Criteria

- [x] Num vault de teste com wikilinks, **100%** das arestas entre documentos indexados
      têm as duas pontas em id de documento, e `mem path` entre duas notas ligadas
      devolve `✅ Encontrado` com 1 salto.
- [x] O wikilink rompido continua apontado pelo `mem doctor` (nenhuma regressão na
      detecção de link morto) — as **138** arestas classificáveis do cofre central (125
      fora do índice + 13 rompidas) continuam visíveis.
- [x] O vault real (`repo_central`), depois do reindex e do `mem doctor --fix`, sai de
      **0** arestas com as duas pontas em id de documento e passa a ter **0** dos 3.200
      nós de id de título; o `README` deixa de ter dois graus (3 e 66).
- [x] `mem path "CONTRIBUTING.md" "README.md" --undirected` no vault do repositório
      devolve **"Encontrado"** — hoje devolve "Não encontrado".
- [x] A ADR-066 existe, está no `docs/adr/README.md` e é citada pela doc que descreve a
      identidade do nó.
- [x] O gate completo (`go -C D:\repository\my-memory test -count=1 ./...`) e o taggeado
      (`go -C D:\repository\my-memory test -tags "sqlite_fts5 onnx" -count=1 ./...`)
      saem **exit 0**, com `go build` e `go vet` verdes.
- [x] **O motor de pathfinding não é tocado** (`internal/graph/path.go` fica intacto) — o
      defeito nunca foi nele.
