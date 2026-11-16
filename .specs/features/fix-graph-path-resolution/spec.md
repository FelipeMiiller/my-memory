# Feature: fix-graph-path-resolution

- **Spec ID**: `fix-graph-path-resolution`
- **Created**: 2026-11-16
- **Merged**: 2026-11-16 — versão final da ISSUE-046: união da spec base `fix-graph-path-resolution` com a spec irmã `fix-graph-path-discovery-gaps` (mesma ISSUE-046, escrita por outra execução e **descartada** pelo dono). Esta spec é a **única** que implementa a ISSUE-046.
- **Owner**: Felipe Miiller
- **Source**: ISSUE-046 — `.specs/ISSUES.md` (seção ISSUE-046) · fecha os gaps FIX-1 e FIX-3..FIX-7 da feature `graph-path-discovery` (ADR-027)
- **Related**: `.specs/features/graph-path-discovery/validation.md` (veredito **FAIL** — 5/8 ACs com evidência, 7 gaps, 7 lacunas de precisão S1–S7; sensor 6/7 mortas, M7 sobreviveu) · ADR-027 (rota e menor caminho) · ADR-063 (semântica de resolução exata) · ISSUE-047 / spec `fix-graph-node-identity` (**FIX-2**, dependência externa) · ISSUE-022 (JSON da tool MCP) · lições L-020, L-021, L-022, L-023, L-024, L-025, L-026

## Problem Statement

O **motor** de descoberta de rotas está correto — o Verifier independente matou 6/7 mutantes e confirmou que o pruning por dominância de profundidade, o modo `hops` (equivalente a BFS) e a inversão epistêmica `c = 1/w` se comportam como a ADR-027 pede. O veredito da feature `graph-path-discovery` é **FAIL** por outra razão: **6 gaps de prova e de contrato** nesta spec, **1 sobrevivente** no sensor de discriminação e **7 lacunas de precisão** no spec da feature — que nunca disse o que o algoritmo emite, nem o que o comando faz quando a entrada é ambígua ou inválida.

| Gap | Sev. | O que falta | Onde (`file:linha`) |
| :--- | :--- | :--- | :--- |
| **FIX-1** | 🟠 Major | o mutante **M7 sobreviveu**: o ramo de nó idêntico é redundante com o Dijkstra genérico e **nada** o distingue — o `Summary` dele não é asseverado por nenhum teste | `internal/graph/path.go:166` · `internal/graph/path_test.go:8` |
| **FIX-3** | 🟠 Major | a resolução de nó é **ambígua e arbitrária**: `LIMIT 1` sem `ORDER BY` sobre **7** nós de mesmo nome — `"README"` resolveu silenciosamente para o README de um skill | `internal/db/graph.go:596-610` · `internal/store/postgres.go:1029-1043` |
| **FIX-4** | 🟠 Major | `EpistemicStatus` e `Weight` das arestas devolvidas **nunca são asseverados** — hoje só existem como *entrada* de mock | `internal/graph/path_test.go:100-106` · `internal/mcp/path_handlers_test.go:48` |
| **FIX-5** | 🟠 Major | o caminho PostgreSQL só é coberto por um **self-path**; o filtro `WHERE ($1 = '' OR repository = $1)` nunca é asseverado | `internal/store/postgres_test.go:272` · `internal/store/postgres.go:1390-1392` |
| **FIX-6** | 🟡 Minor | o `Summary` de não-encontrado, o bloco CLI de not-found, os headings de rota/tabela MCP e o código `InvalidParams` **não são asseverados** | `internal/graph/path.go:323` · `cmd/mem/path.go:125-127` · `internal/mcp/path_handlers.go:31,40,124,138` |
| **FIX-7** | 🟡 Minor | `graph.DefaultPathOptions()` é **morto em produção**; os mesmos três defaults são re-declarados como literais em mais dois lugares | `internal/graph/path.go:26` · `cmd/mem/path.go:50-72` · `internal/mcp/path_handlers.go:43-65` |
| **FIX-2** | 🔴 Critical | **fora do escopo desta spec** — a falha ponta a ponta em dado real é defeito de **identidade de nó**, com spec própria | ISSUE-047 · spec `fix-graph-node-identity` |

E as **7 lacunas de precisão** do spec da feature (S1–S7), que esta spec fecha por escrito: precedência entre `--directed` e `--undirected` (S3), `--mode` inválido (S4), `--max-depth` não-positivo (S5), renderização de uma aresta epistêmica (S2), literais de relação e de status epistêmico (S6), texto do `Summary` (S7) e a promessa P3 de "Markdown **e** JSON" que a resposta só-texto não cumpre (S1 — **já rastreada pela ISSUE-022**, referenciada aqui, não duplicada).

**A dependência que fica registrada:** enquanto a **ISSUE-047** (`fix-graph-node-identity`) não for implementada, o `mem path` **não acha caminho no vault real** — 0 de 40.243 arestas do cofre central têm as duas pontas no mesmo lado da fenda de identidade. Esta spec **não** corrige isso: ela fecha os 6 gaps de prova e contrato e **declara** a dependência. O comando só fica usável no dado real depois da outra spec.

## Goals

- [ ] A resolução de nó é **determinística e explícita**: o mesmo identificador resolve sempre o mesmo nó, o desempate tem **ordem estável** (nunca "o primeiro que casou", nunca `LIMIT 1` sem `ORDER BY`) e a **ambiguidade é reportada** — candidatos e chaves de desambiguação — em vez de escolhida em silêncio.
- [ ] A **CLI** informa a ambiguidade e **segue** com a rota; a **tool MCP** devolve `InvalidParams` **nomeando os candidatos** (uma tool não tem como perguntar — ela devolve o erro que permite ao agente tentar de novo com um path).
- [ ] O ramo de nó idêntico tem **guarda**: o `Summary` é asseverado e o mutante **M7 morre**.
- [ ] As arestas devolvidas têm **status epistêmico, peso e relação asseverados**, com os valores canônicos da ADR-027.
- [ ] O caminho PostgreSQL tem **cobertura de integração** de verdade: rota multi-hop **e** isolamento por repositório.
- [ ] As mensagens e os blocos de saída têm **asserção**: `Summary` de não-encontrado, bloco CLI de not-found, headings de rota/tabela MCP e o código `InvalidParams`.
- [ ] Os defaults de caminho têm **uma única fonte**: `graph.DefaultPathOptions()` deixa de ser código morto.
- [ ] O spec da feature `graph-path-discovery` passa a **definir** os 7 pontos que nunca definiu (S1–S7).
- [ ] A dependência do **FIX-2 / ISSUE-047** fica escrita nesta spec e no spec da feature, sem que esta spec duplique a correção.

## Out of Scope

| Item | Reason |
| :--- | :--- |
| Corrigir a **identidade do nó** (FIX-2 / ISSUE-047) | É defeito de **produto**, com spec própria (`fix-graph-node-identity`): a correção toca o indexador e o resolvedor canônico, e "qual identidade é a canônica" é decisão de arquitetura. Aqui a resolução fica determinística e explícita — **não** unificada. |
| Corrigir o `mem path` **no vault real** | Depende do reparo de identidade (ISSUE-047) e do reindex. Nenhuma aresta do cofre central tem as duas pontas do mesmo lado da fenda; esta spec é de **prova, contrato e spec**. |
| Mexer no **motor** de pathfinding (`internal/graph/path.go`) | O motor está correto (6/7 mutantes mortos; pruning, `hops` e inversão epistêmica conferidos pelo Verifier). A correção é de **prova**, de **contrato** e de **spec**. |
| Trocar o texto do `Summary`, os headings ou as literals | A correção **assevera** o que o código já emite; trocar texto sem gap que o prove seria mudança de comportamento. |
| Implementar `structuredContent` (JSON) na tool MCP | É a **ISSUE-022**, decisão de produto pendente. Esta spec só alinha o *spec* da feature (S1). |
| Desambiguar por `alias:` | O banco **não guarda alias** (`documents`: id/path/title; `graph_nodes`: id/type/name). A 4ª chave do contrato da ADR-063 vive só no resolver de disco (`internal/vaultfs/target.go:115-117`); usá-la no `mem path` exigiria a raiz do vault no comando — flag nova, escopo novo. |
| Criar **flag nova** para desambiguar | A desambiguação usa as chaves que o banco já tem (`caminho relativo` e `title:`). Nenhuma flag nova entra no `mem path` nem na tool MCP. |
| Mexer no `tagged_as` (ADR-041) ou no code graph (ADR-047) | Nenhum dos dois participa do caminho. |
| Remover o alias `--depth` da CLI (`cmd/mem/path.go:29`) | Não é gap da ISSUE-046 nem lacuna S1–S7. Fica documentado como comportamento vigente. |
| Reindexar, reescrever ou tocar em vault do dono | Nenhum dado gravado muda — a correção é de código, prova e spec. |
| Criar ADR nova | Nenhuma decisão de arquitetura nova: ADR-027 (rota) e ADR-063 (resolução exata) continuam valendo. |

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| **Ambiguidade**: erro ou desempate determinístico? | **os dois, por superfície**: o resolvedor é **determinístico** (ordem estável — nunca "o primeiro que casou", nunca `LIMIT 1` sem `ORDER BY`) e **nunca escolhe em silêncio**; a **CLI** `mem path` **informa a ambiguidade** (lista de candidatos + chaves de desambiguação) e **segue** devolvendo a rota; a **tool MCP** `memory_find_path` devolve **`InvalidParams`** **nomeando os candidatos** | é a **decisão do dono nesta rodada** e a **união** das duas versões: o desempate da versão A garante reprodutibilidade, a informação de ambiguidade da versão B garante que a escolha seja **visível** — uma não anula a outra. É a mesma lição do repo (L-023): um `LIMIT 1` sem `ORDER BY` aponta o usuário para o arquivo errado. A CLI **não pode** bloquear o uso — um vault com 7 `README` não pode impedir o comando (argumento da versão A); a tool MCP **não tem como perguntar**, então devolve o erro que permite ao agente tentar de novo com um path (argumento da versão B). **Conflito com a spec irmã `fix-graph-path-discovery-gaps` resolvido**: ela é descartada pelo dono; esta spec é a versão final. | y — **decidido pelo dono (2026-11-16)** |
| Critério de desempate entre candidatos da **mesma** chave de resolução | (1) o nó cujo id é o id de um **documento do mesmo repositório**; (2) o de **menos segmentos de caminho** (o mais raso); (3) o de **id lexicograficamente menor** | o caso medido é 7 nós `README`: sem critério o banco devolve qualquer um. Preferir o lado do documento e depois o caminho mais raso leva ao README da raiz — e é o lado que vira canônico na ISSUE-047. O desempate final por id torna a consulta **reprodutível**. **Alternativa registrada**: ordenar apenas por caminho relativo e id (a mesma ordem do relatório de candidatos), que é mais simples mas **não** privilegia o lado do documento. | n — default escolhido (veio da versão A) |
| Como a ambiguidade chega do resolvedor às superfícies | o resultado do caminho (`graph.PathResult`) carrega o **relatório de ambiguidade** com os candidatos ordenados; a CLI o **imprime**; a tool MCP o **converte** em `InvalidParams` | a CLI precisa **continuar** devolvendo a rota enquanto informa a ambiguidade — um erro no resolvedor a faria falhar, e o dono decidiu que ela não falha. Um canal de **retorno** serve às duas superfícies; um erro tipado exigiria que a CLI capturasse a ambiguidade e seguisse com o vencedor, acoplar o erro ao `fmt.Errorf("falha ao resolver nó de origem: %w")` (`internal/db/graph.go:921-929`) e ainda assim ter o MCP descascar o erro embrulhado. **Alternativa registrada**: erro tipado de ambiguidade no resolvedor, com a CLI capturando-o e continuando. | n — default escolhido (veio da mescla) |
| Chave de desambiguação `alias:` | desambiguação por **caminho relativo** e por **`title:`** — as chaves que o banco tem; o `alias:` **fica fora** do desempate no banco | `documents` guarda id/path/title e `graph_nodes` guarda id/type/name (`internal/db/schema.go:5-11`, `:47-51`; `internal/store/postgres.go:26-30`, `:58-64`). A 4ª chave (`alias:`) só existe no resolver de disco (`internal/vaultfs/target.go:115-117`). Prometer `alias:` num texto sem poder honrá-lo seria outro "verde-mentira". | n — default escolhido |
| O ramo de nó idêntico é **guardado** ou **deletado**? | **guardado**: o `Summary` dele é asseverado | o ramo carrega uma informação que o Dijkstra genérico não dá ("origem e destino são idênticos"), e é **exatamente** essa diferença que mata o M7. A renderização da CLI e do MCP decide por `Hops == 0` (`cmd/mem/path.go:135`, `internal/mcp/path_handlers.go:119`) — só o `Summary` distingue o ramo. Deletá-lo apagaria um comportamento observável (L-020). | y — **decidido** |
| Precedência entre `--directed` e `--undirected` | **`--undirected` ganha** | é o comportamento atual (`cmd/mem/path.go:58-61`) e é o mais seguro: quem pede bidirecional não deve ser ignorado. A correção **escreve** a regra, sem mudar comportamento. | y — **decidido** |
| `--mode` inválido | **cai para `epistemic`** | é o comportamento atual (`cmd/mem/path.go:63-66`, `internal/mcp/path_handlers.go:59-65`) e a queda é **visível**: o modo efetivo é impresso na saída (`Modo:` na CLI, `cost_mode` no JSON). Virar erro seria mudança de comportamento sem gap que a prove. | n — default escolhido |
| `--max-depth` não-positivo | **volta para 6** | as três camadas já concordam em 6 (`cmd/mem/path.go:54-56`, `internal/mcp/path_handlers.go:44-49`, `internal/graph/path.go:140-142`); com o FIX-7 esse 6 passa a ter uma fonte só. | n — default escolhido |
| Como uma aresta epistêmica é renderizada | a **relação** vira `links_to` e o **status epistêmico é preservado** | é o comportamento atual (`internal/graph/path.go:191-194`): relação vazia ou `EXTRACTED`/`INFERRED`/`TAG` é reescrita como `links_to`, e o status continua no campo próprio — é ele que pondera o custo. | y — **decidido** |
| Literais canônicos de relação e de status epistêmico | relação ∈ {`links_to`, `implements`, `depends_on`, `references_code`, `tagged_as`}; status ∈ {`EXTRACTED`, `INFERRED`, `TAG`}; pesos **1.0** / **0.6** / **0.3** | são as constantes da ADR-027 (`internal/graph/pagerank.go:10-12`) e a normalização em `internal/graph/path.go:105-107`; qualquer status fora dos três vira `EXTRACTED`. O custo epistêmico é `1/w` (1.0 · 1.67 · 3.33). | y — **decidido** |
| Texto canônico do `Summary` | não encontrado: `Nenhum caminho encontrado entre '<origem>' e '<destino>' dentro de <N> saltos` · nó idêntico: `Nó de origem e destino são idênticos ('<id>')` | são os textos que o código já emite (`internal/graph/path.go:323` e `:177`); a correção os **assevera** e os escreve no spec (L-010). | y — **decidido** |
| Os defaults têm uma fonte: helper ou remoção? | **`graph.DefaultPathOptions()` é a fonte** — o helper passa a ser chamado pela CLI e pela tool MCP | o helper já existe com os três valores certos (`internal/graph/path.go:26-32`); remover exigiria re-declarar os mesmos defaults em outro lugar. O AC é **observável** (os defaults aplicados vêm do helper), não "arrumar código morto" (L-025). | y — **decidido** |
| Código de erro da ambiguidade na tool MCP | **`InvalidParams`** | ambiguidade é **parâmetro inválido**, não falha de servidor; hoje o handler embrulha qualquer erro do `pathFn` em `CodeInternalError` (`internal/mcp/path_handlers.go:83`), então a distinção precisa ser explícita. | n — default escolhido |
| O motor é tocado? | **não**: nenhuma mudança em `internal/graph/path.go` (Dijkstra, pruning, inversão epistêmica) | o Verifier confirmou que os três estão corretos; a correção é de prova, contrato e spec. | y — **decidido** |
| A promessa de **JSON** na tool MCP (S1) | sai do **spec** da feature, com remissão à **ISSUE-022** | a tool devolve texto (`NewTextResult`, `internal/mcp/handlers.go:40`) e a ISSUE-022 já rastreia o `structuredContent` ausente. Prometer JSON num spec que não entrega é a **mesma** falha que esta spec fecha nos outros 6 pontos — mas a **correção** do payload é da ISSUE-022. | y — **decidido** |
| O alias `--depth` (`cmd/mem/path.go:29`) | **permanece** aceito, com a mesma regra do `--max-depth` | não é gap da ISSUE-046 nem lacuna S1–S7; remover é decisão de produto à parte. A correção apenas o registra como comportamento vigente. | n — default escolhido |

**Open questions**: none — todas as decisões acima têm default e rationale, e as duas divergências que existiam entre as versões da ISSUE-046 (erro × desempate na resolução ambígua; critério de desempate e canal da ambiguidade) estão resolvidas nas três primeiras linhas desta tabela, com alternativa registrada.

## User Stories

### P1: A resolução de nó é determinística e nunca escolhe em silêncio ⭐ MVP

**User Story**: Como usuário do `mem path`, quero que um nome ambíguo **não escolha sozinho**, que a escolha, quando existe, seja **sempre a mesma**, e que eu veja **quais** alternativas existiam, para que eu não analise o resultado de outra nota sem saber.

**Why P1**: é o FIX-3 — 7 nós `README` e um `LIMIT 1` sem `ORDER BY`: o banco devolve o que der. O Verifier mediu `"README"` caindo no README de um skill, **em silêncio**.

**Acceptance Criteria**:
1. WHEN the same identifier is resolved twice against the same repository THEN the resolver SHALL return the same canonical node id in both calls.
2. WHEN more than one node matches the identifier under the same resolution key THEN the resolver SHALL return the ordered candidate list together with the canonical node id.
3. WHEN two or more graph nodes share the requested name THEN the canonical node id SHALL be the id of the node that is a document of the same repository.
4. WHEN several candidate nodes remain after the document-id preference THEN the canonical node id SHALL be the id of the candidate with the fewest path segments, and any remaining tie SHALL be broken by the lexicographically smallest node id.
5. The resolution keys SHALL be evaluated in this order: the document id, the relative path, the node name, the document `title:`, and the fuzzy match.
6. WHEN the identifier matches the relative path of a document THEN the resolver SHALL return that document's id.
7. The candidates of an ambiguity report SHALL be ordered by relative path and then by node id.
8. The SQLite resolver and the PostgreSQL resolver SHALL return the same canonical node id for the same identifier against the same repository.
9. The ambiguity report SHALL list every candidate with its relative path and its title.
10. The ambiguity report SHALL name the disambiguation keys that address a candidate: the relative path, with or without the `.md` suffix, and the `title:`.

**Independent Test**: num vault de teste com dois nós de mesmo nome, inseridos em ordem inversa à esperada → a resolução devolve o **mesmo** id nas duas chamadas, o relatório lista os dois candidatos **ordenados por caminho**, e o desempate favorece o nó que é documento do mesmo repositório; o mesmo identificador resolve igual nos dois motores.

---

### P1: A CLI informa a ambiguidade em vez de escolher em silêncio ⭐ MVP

**User Story**: Como usuário do `mem path`, quero ver **quais** notas casaram com o meu identificador e **como** desambiguar, para que eu nunca analise o resultado de outra nota sem perceber que houve ambiguidade.

**Why P1**: é o lado da CLI da decisão do dono — a escolha é **determinística** (reprodutível) **e** **visível**; a CLI não bloqueia o uso de um vault com muitos nomes repetidos.

**Acceptance Criteria**:
11. WHEN the CLI resolves an identifier THEN its output SHALL name the canonical node id it resolved.
12. WHEN the resolution of an identifier matches more than one candidate THEN the CLI `mem path` output SHALL contain the relative path and the title of every candidate.
13. WHEN the resolution of an identifier matches more than one candidate THEN the CLI `mem path` output SHALL name the disambiguation keys `title:` and the relative path, with the `.md` suffix optional.
14. WHEN the resolution of an identifier matches more than one candidate THEN the CLI `mem path` SHALL still report the route result and SHALL NOT fail.

**Independent Test**: `mem path "README" "alvo"` num vault com 7 `README` → a saída nomeia o id resolvido, lista os 7 candidatos com caminho e título, diz `title:` e o caminho relativo, e **não** sai com erro.

---

### P1: A tool MCP devolve `InvalidParams` em vez de escolher em silêncio ⭐ MVP

**User Story**: Como agente que chama a tool MCP, quero receber um **erro que nomeie os candidatos** quando meu identificador é ambíguo, para que eu possa tentar de novo com um caminho — em vez de receber uma rota silenciosa para a nota errada.

**Why P1**: é o lado da tool da decisão do dono — uma tool não tem como perguntar ao usuário, então ela devolve o erro que permite ao agente tentar de novo com um path. O handler hoje embrulha qualquer erro em `CodeInternalError` (`internal/mcp/path_handlers.go:83`), o que esconderia a causa.

**Acceptance Criteria**:
15. IF the resolution of an identifier matches more than one candidate THEN the MCP tool `memory_find_path` SHALL fail with the error code `InvalidParams`.
16. IF the resolution of an identifier matches more than one candidate THEN the error message of the MCP tool SHALL contain the relative path and the title of every candidate.
17. IF the failure of the path handler is not an ambiguity failure THEN the MCP tool SHALL fail with the error code `CodeInternalError`.

**Independent Test**: a tool com um identificador ambíguo falha com `InvalidParams` e o texto lista os candidatos; a tool com um grafo quebrado falha com `CodeInternalError` (controle negativo).

---

### P1: O ramo de nó idêntico é guardado ⭐ MVP

**User Story**: Como usuário do `mem path`, quero que a resposta para "origem = destino" seja **distinta e previsível**, para que eu saiba que não é um caminho vazio por bug.

**Why P1**: é o FIX-1 — o mutante M7 sobreviveu porque o ramo existe mas nada o distingue do Dijkstra genérico. Só o `Summary` distingue: a CLI e o MCP decidem por `Hops == 0`, então **asseverar a renderização não mataria o M7** — asseverar o `Summary`, sim.

**Acceptance Criteria**:
18. WHEN the source and the target resolve to the same node THEN the result SHALL report 0 hops, a total cost of 0, exactly one node and no edges.
19. WHEN the source and the target resolve to the same node THEN the result summary SHALL be `Nó de origem e destino são idênticos ('<id>')`.
20. WHEN the source and the target resolve to the same node THEN the CLI SHALL print `Origem e destino são o mesmo nó ('<id>').`

**Independent Test**: `mem path "nota.md" "nota.md"` → o `Summary` é `Nó de origem e destino são idênticos ('nota.md')` e a CLI imprime `Origem e destino são o mesmo nó ('nota.md').`; desligar o ramo (`internal/graph/path.go:166`) faz o teste falhar.

---

### P2: Os detalhes da aresta são asseverados

**User Story**: Como consumidor do resultado, quero saber **por que** a rota foi escolhida — relação, status epistêmico e peso de cada trecho.

**Why P2**: é o FIX-4 (+ S2, S6) — o resultado devolve esses campos e **nenhum teste** os olha; hoje só existem como *entrada* de mock.

**Acceptance Criteria**:
21. WHEN a route is returned THEN each returned edge SHALL carry the relation, the epistemic status and the weight of the edge it traversed.
22. The epistemic status of a returned edge SHALL be `EXTRACTED`, `INFERRED` or `TAG`.
23. The weight of a returned edge SHALL be 1.0 when its status is `EXTRACTED`, 0.6 when it is `INFERRED` and 0.3 when it is `TAG`.
24. WHEN the traversed edge has an empty relation or the relation `EXTRACTED`, `INFERRED` or `TAG` THEN the returned relation SHALL be `links_to`.

**Independent Test**: um grafo com as três relações epistêmicas → o teste assevera status e peso de cada aresta devolvida, e a relação `EXTRACTED` aparece como `links_to`.

---

### P2: O caminho PostgreSQL é coberto de verdade

**User Story**: Como dono do vault de produção (Postgres), quero que o `FindPath` do Postgres seja testado como o SQLite, para que a cobertura não seja uma ilusão.

**Why P2**: é o FIX-5 — a única asserção do Postgres é `Found && Hops == 0` (self-path), e o filtro `WHERE ($1 = '' OR repository = $1)` nunca é provado.

**Acceptance Criteria**:
25. WHEN the PostgreSQL `FindPath` resolves a route of 2 hops inside a repository THEN the result SHALL report `Found = true` and `Hops = 2`.
26. WHEN the same PostgreSQL database also holds a node with the same name in another repository THEN the route returned for the first repository SHALL contain no node of the other repository.
27. The PostgreSQL `FindPath` integration test SHALL assert a multi-hop route and not only the self-path case.

**Independent Test**: dois repositórios no mesmo banco, um caminho de 2 saltos dentro de um deles e um nó homônimo no outro → a rota não atravessa.

---

### P2: As mensagens e os blocos têm asserção

**User Story**: Como usuário e como agente, quero que a mensagem de "não encontrado" e os blocos de saída sejam **estáveis**, para que eu possa depender deles.

**Why P2**: é o FIX-6 (+ S7) — hoje nada garante que esses textos existam.

**Acceptance Criteria**:
28. WHEN no route is found THEN the result summary SHALL be `Nenhum caminho encontrado entre '<origem>' e '<destino>' dentro de <N> saltos`.
29. WHEN the CLI reports a not-found result THEN its output SHALL contain the line `Status:          ❌ Não encontrado`.
30. WHEN the CLI reports a not-found result THEN its output SHALL contain the result summary in the `Detalhes:` line.
31. WHEN the MCP tool is called without `source` or without `target` THEN it SHALL fail with the error code `InvalidParams`.
32. WHEN the MCP tool returns a route THEN the Markdown SHALL contain the headings `Rota Conectada` and `Segmentos Percorridos`.

**Independent Test**: cada um dos 5 textos é asseverado no respective teste.

---

### P2: Os defaults têm uma fonte única

**User Story**: Como mantenedor, quero que os defaults do caminho vivam em **um** lugar, para que a CLI e a tool MCP não possam divergir.

**Why P2**: é o FIX-7 — `graph.DefaultPathOptions()` foi construído e nunca foi chamado; os mesmos três defaults estão literais em mais dois arquivos.

**Acceptance Criteria**:
33. The default path options SHALL be exactly `MaxDepth = 6`, `Directed = true` and `CostMode = epistemic`.
34. WHEN the flags `--max-depth`, `--directed`/`--undirected` and `--mode` are absent THEN the CLI `mem path` SHALL apply the values returned by `graph.DefaultPathOptions()`.
35. WHEN the arguments `max_depth`, `directed` and `mode` are absent THEN the MCP tool `memory_find_path` SHALL apply the values returned by `graph.DefaultPathOptions()`.
36. The CLI and the MCP tool SHALL NOT declare a path default as a literal outside `graph.DefaultPathOptions()`.

**Independent Test**: o `grep` dos três valores como default de caminho em `cmd/mem/path.go` e `internal/mcp/path_handlers.go` não acha nada; e o sensor mata a mutação composta que divergiria a CLI do helper.

---

### P1: O spec da feature define o que ele nunca definiu ⭐ MVP

**User Story**: Como quem implementa contra o spec da feature, quero que ele diga o comportamento observável, para que não exista mais "o spec não diz".

**Why P1**: são as 7 lacunas S1–S7 — a mesma classe de falha que o Verifier acusou nos outros gaps.

**Acceptance Criteria**:
37. The spec `graph-path-discovery` SHALL state that `--undirected` takes precedence over `--directed` when both are passed.
38. The spec `graph-path-discovery` SHALL state that an invalid `--mode` value falls back to `epistemic`.
39. The spec `graph-path-discovery` SHALL state that a non-positive `--max-depth` is reset to 6.
40. The spec `graph-path-discovery` SHALL state the canonical relation literals (`links_to`, `implements`, `depends_on`, `references_code`, `tagged_as`), the canonical epistemic-status literals (`EXTRACTED`, `INFERRED`, `TAG`) and their weights (1.0, 0.6, 0.3).
41. The spec `graph-path-discovery` SHALL state the two `Summary` texts: `Nenhum caminho encontrado entre '<origem>' e '<destino>' dentro de <N> saltos` and `Nó de origem e destino são idênticos ('<id>')`.
42. The spec `graph-path-discovery` SHALL state that the MCP tool returns Markdown text only, and SHALL refer the JSON payload to ISSUE-022.

**Independent Test**: a leitura cruzada entre o spec emendado e o código mostra que cada um dos 7 pontos está escrito.

---

### P1: A correção é re-verificada de forma independente ⭐ MVP

**User Story**: Como dono do repo, quero que a correção seja **re-verificada por um Verifier independente**, com sensor de discriminação, para que a feature saia do FAIL com prova.

**Why P1**: o sensor que reprovou a feature rodou **6/7**; o M7 sobreviveu e a correção precisa matá-lo. E o sensor desta correção precisa provar o FIX-7 (helper vivo), que é um estado que **um teste de valor fixo não prova**.

**Acceptance Criteria**:
43. WHEN the discrimination sensor re-injects the mutation that disables the identical-node branch THEN the test suite SHALL fail.
44. WHEN the discrimination sensor re-injects the mutation that makes the default path options return `MaxDepth = 9` and the CLI keeps the literal `6` THEN the test suite SHALL fail.
45. WHEN the discrimination sensor re-injects the six mutations that were already dead THEN the test suite SHALL fail for each of them.
46. The verification of the `graph-path-discovery` feature SHALL be produced by a Verifier distinct from the author of this correction.
47. The verification report of `graph-path-discovery` SHALL record the verdict `PASS` with `file:line` evidence for every acceptance criterion of the feature.

**Independent Test**: o relatório da feature corrigida cita `file:line` para as ACs e o sensor mata o M7 e a mutação composta do FIX-7.

## Edge Cases

- IF o nome consultado casa com **nenhum** nó THEN a saída segue sendo "não encontrado" — a correção muda **qual** nó é escolhido, não a existência da rota.
- IF o nome casa com **um** nó apenas THEN a resolução **não** vira erro: ela devolve o id canônico, e a lista de candidatos fica vazia.
- IF o mesmo documento aparece por **duas chaves** (id e caminho relativo) THEN ele é **um** candidato, não dois — a desambiguação é por nó, não por linha casada.
- IF dois nós têm o **mesmo id** (id de documento e id de título apontando para a mesma nota) THEN a preferência pelo lado do documento vence — é a transição que a ISSUE-047 vai tornar canônica.
- IF o grafo tem **ciclo** THEN a busca termina (o guard de ciclo existe) — a correção não mexe no motor.
- IF a rota passa por uma aresta de **peso customizado** (> 0) THEN o peso devolvido é o customizado (`internal/graph/pagerank.go:44-47`), não o da tabela canônica — a AC23 cobre o caso **sem** peso customizado.
- IF o usuário passa `--directed` e `--undirected` juntos THEN `--undirected` vence e a saída mostra `Bidirecional / Não-direcionado (A <-> B)`.
- IF o usuário passa `--mode` inválido THEN o modo efetivo é `epistemic` e a saída mostra `Modo:            epistemic`.
- IF o usuário passa `--max-depth` não-positivo THEN a profundidade efetiva é 6 e a saída mostra `Profundidade:    6 saltos máx`.
- IF o `--depth` (alias não documentado) e o `--max-depth` são passados juntos THEN o alias só entra em vigor quando **diverge** do default 6 (`cmd/mem/path.go:51-53`) — comportamento vigente, que a correção documenta junto com o resto.
- IF o grafo do Postgres tem **dois** repositórios e o filtro de repositório é exercitado THEN a rota nunca atravessa a fronteira — o self-path não exercita esse filtro (L-021).
- IF a resolução é ambígua **no Postgres** THEN o filtro de repositório vale **antes** da contagem de candidatos: um nó homônimo de outro repositório não vira candidato.

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Gap | Origem | Where | Status |
| :--- | :--- | :--- | :--- | :--- | :--- |
| GPR-01 | AC1 — mesma consulta, mesmo id canônico | FIX-3 | B | `internal/db/graph.go` · `internal/store/postgres.go` | Pending |
| GPR-02 | AC2 — a resolução devolve os candidatos ordenados junto com o id canônico | FIX-3 | Merge | `internal/graph/resolve.go` | Pending |
| GPR-03 | AC3 — preferência pelo nó que é documento do mesmo repositório | FIX-3 | A | `internal/db/graph.go` · `internal/store/postgres.go` | Pending |
| GPR-04 | AC4 — desempate por menos segmentos de caminho e depois id lexicograficamente menor | FIX-3 | A | `internal/db/graph.go` · `internal/store/postgres.go` | Pending |
| GPR-05 | AC5 — ordem das chaves de resolução | FIX-3 | B | `internal/db/graph.go` · `internal/store/postgres.go` | Pending |
| GPR-06 | AC6 — caminho relativo resolve para o documento | FIX-3 | B | `internal/db/graph.go` · `internal/store/postgres.go` | Pending |
| GPR-07 | AC7 — candidatos ordenados por caminho e id | FIX-3 | B | `internal/graph/resolve.go` | Pending |
| GPR-08 | AC8 — SQLite e Postgres resolvem para o mesmo id | FIX-3 | B+A | `internal/store/postgres.go` | Pending |
| GPR-09 | AC9 — o relatório lista cada candidato com caminho relativo e título | FIX-3 | B | `internal/graph/resolve.go` | Pending |
| GPR-10 | AC10 — o relatório nomeia as chaves de desambiguação | FIX-3 | B | `internal/graph/resolve.go` | Pending |
| GPR-11 | AC11 — a CLI nomeia o id canônico resolvido | FIX-3 | A | `cmd/mem/path.go` | Pending |
| GPR-12 | AC12 — a CLI imprime caminho e título de cada candidato | FIX-3 | Merge | `cmd/mem/path.go` | Pending |
| GPR-13 | AC13 — a CLI nomeia as chaves `title:` e caminho relativo | FIX-3 | Merge | `cmd/mem/path.go` | Pending |
| GPR-14 | AC14 — a CLI não falha quando a resolução é ambígua | FIX-3 | Merge | `cmd/mem/path.go` | Pending |
| GPR-15 | AC15 — a tool MCP devolve `InvalidParams` na ambiguidade | FIX-3 | Merge | `internal/mcp/path_handlers.go` | Pending |
| GPR-16 | AC16 — o erro da tool MCP nomeia cada candidato | FIX-3 | Merge | `internal/mcp/path_handlers.go` | Pending |
| GPR-17 | AC17 — erro que não é de ambiguidade continua `CodeInternalError` | FIX-3 | B | `internal/mcp/path_handlers.go` | Pending |
| GPR-18 | AC18 — nó idêntico: 0 saltos, custo 0, 1 nó, 0 arestas | FIX-1 | B+A | `internal/graph/path_test.go` | Pending |
| GPR-19 | AC19 — `Summary` do nó idêntico (mata o M7) | FIX-1, S7 | B+A | `internal/graph/path_test.go` | Pending |
| GPR-20 | AC20 — renderização da CLI do nó idêntico | FIX-1 | B+A | `cmd/mem/path_test.go` | Pending |
| GPR-21 | AC21 — a aresta devolvida carrega relação, status e peso | FIX-4 | A | `internal/graph/path_test.go` | Pending |
| GPR-22 | AC22 — status epistêmico ∈ {EXTRACTED, INFERRED, TAG} | FIX-4, S6 | B+A | `internal/graph/path_test.go` | Pending |
| GPR-23 | AC23 — pesos 1.0 / 0.6 / 0.3 por status | FIX-4, S6 | B+A | `internal/graph/path_test.go` | Pending |
| GPR-24 | AC24 — relação epistêmica vira `links_to` | FIX-4, S2 | B+A | `internal/graph/path_test.go` | Pending |
| GPR-25 | AC25 — rota de 2 saltos no Postgres | FIX-5 | B | `internal/store/postgres_test.go` | Pending |
| GPR-26 | AC26 — a rota do Postgres não atravessa o repositório | FIX-5 | B+A | `internal/store/postgres_test.go` | Pending |
| GPR-27 | AC27 — o teste do Postgres assere rota multi-hop, não só self-path | FIX-5 | A | `internal/store/postgres_test.go` | Pending |
| GPR-28 | AC28 — `Summary` de não-encontrado | FIX-6, S7 | B+A | `internal/graph/path_test.go` | Pending |
| GPR-29 | AC29 — linha `Status:          ❌ Não encontrado` na CLI | FIX-6 | A | `cmd/mem/path_test.go` | Pending |
| GPR-30 | AC30 — linha `Detalhes:` com o `Summary` | FIX-6 | B | `cmd/mem/path_test.go` | Pending |
| GPR-31 | AC31 — `InvalidParams` sem `source`/`target` | FIX-6 | B+A | `internal/mcp/path_handlers_test.go` | Pending |
| GPR-32 | AC32 — headings `Rota Conectada` e `Segmentos Percorridos` | FIX-6 | B+A | `internal/mcp/path_handlers_test.go` | Pending |
| GPR-33 | AC33 — defaults canônicos 6 / true / epistemic | FIX-7 | B | `internal/graph/path_test.go` | Pending |
| GPR-34 | AC34 — a CLI aplica os defaults do helper | FIX-7 | B | `cmd/mem/path.go` | Pending |
| GPR-35 | AC35 — a tool MCP aplica os defaults do helper | FIX-7 | B | `internal/mcp/path_handlers.go` | Pending |
| GPR-36 | AC36 — nenhum default de caminho literal fora do helper | FIX-7 | B+A | `cmd/mem/path.go` · `internal/mcp/path_handlers.go` | Pending |
| GPR-37 | AC37 — o spec define a precedência das flags | S3 | B+A | `.specs/features/graph-path-discovery/spec.md` | Pending |
| GPR-38 | AC38 — o spec define o `--mode` inválido | S4 | B+A | `.specs/features/graph-path-discovery/spec.md` | Pending |
| GPR-39 | AC39 — o spec define o `--max-depth` não-positivo | S5 | B+A | `.specs/features/graph-path-discovery/spec.md` | Pending |
| GPR-40 | AC40 — o spec define literais e pesos canônicos | S2, S6 | B+A | `.specs/features/graph-path-discovery/spec.md` | Pending |
| GPR-41 | AC41 — o spec define os dois textos de `Summary` | S7 | B+A | `.specs/features/graph-path-discovery/spec.md` | Pending |
| GPR-42 | AC42 — o spec define a resposta texto-only (S1 → ISSUE-022) | S1 | B+A | `.specs/features/graph-path-discovery/spec.md` | Pending |
| GPR-43 | AC43 — o sensor mata o M7 | FIX-1 | B+A | `.specs/features/graph-path-discovery/validation.md` | Pending |
| GPR-44 | AC44 — o sensor mata a mutação composta do FIX-7 | FIX-7 | B | `.specs/features/graph-path-discovery/validation.md` | Pending |
| GPR-45 | AC45 — M1..M6 continuam mortas | regressão | B | `.specs/features/graph-path-discovery/validation.md` | Pending |
| GPR-46 | AC46 — Verifier distinto do autor | ciclo fix → re-verificar | B+A | `.specs/features/graph-path-discovery/validation.md` | Pending |
| GPR-47 | AC47 — relatório com veredito PASS e evidência `file:line` | ciclo fix → re-verificar | B | `.specs/features/graph-path-discovery/validation.md` | Pending |

**Coverage**: 47 total, 47 mapeados, 0 unmapped.

## Success Criteria

- [ ] `mem path "<nome ambíguo>" "<alvo>"` num vault com 7 `README` resolve **sempre o mesmo nó** nos **dois** motores, **nomeia o id resolvido** na saída e **informa a ambiguidade** (candidatos com caminho e título + chave `title:`) **sem falhar**; a tool MCP, no mesmo vault, devolve `InvalidParams` **nomeando os candidatos**.
- [ ] O mesmo identificador, resolvido duas vezes, devolve **o mesmo** id canônico; e o caminho relativo resolve para o documento do caminho, sem cair no nome.
- [ ] O mutante **M7** (ramo de nó idêntico desligado) **morre** no sensor; os 6 mutantes que já morriam **continuam** mortos.
- [ ] As arestas devolvidas têm status epistêmico, peso (1.0 / 0.6 / 0.3) e relação (`links_to` para as epistêmicas) asseverados.
- [ ] O `FindPath` do Postgres tem integração com **rota de 2 saltos** e com prova de **isolamento por repositório** (dois repositórios no mesmo banco).
- [ ] O `Summary` de não-encontrado, o bloco `❌ Não encontrado`, a linha `Detalhes:`, os headings MCP e o código `InvalidParams` têm asserção.
- [ ] `graph.DefaultPathOptions()` é **chamado** pela CLI e pela tool MCP — e não há mais default literal de caminho fora dele; a mutação composta do sensor **morre**.
- [ ] O spec `graph-path-discovery` define os **7** pontos que nunca definiu (S1–S7), com a remissão à ISSUE-022 na promessa de JSON.
- [ ] A feature `graph-path-discovery` sai do **FAIL**: relatório com veredito **PASS**, evidência `file:line` para as ACs e `validate_state.py graph-path-discovery` **exit 0**.
- [ ] A **dependência** do FIX-2 fica escrita nesta spec e no spec da feature: o `mem path` só acha caminho no vault real **depois** da spec `fix-graph-node-identity` (ISSUE-047) e do reindex.
- [ ] O gate completo (`go -C D:\repository\my-memory test -count=1 ./...`, com Postgres) e o gate taggeado (`go -C D:\repository\my-memory test -tags "sqlite_fts5 onnx" -count=1 ./...`) saem **exit 0**, com `go build` e `go vet` verdes.
