# Feature: Code Graph Real — o vault de repositório como code graph de primeira classe

- **Status**: in tasks
- **Decisão de arquitetura**: [ADR-061](../../docs/adr/061-dois-conceitos-de-memoria-vault-central-e-code-graph.md)
- **Issue**: [ISSUE-027](../ISSUES.md) (12 gaps, medidos com evidência)
- **Referência conceitual**: [Graphify-Labs/graphify](https://github.com/Graphify-Labs/graphify) `v0.9.94` — **referência de modelo, não dependência**
- **Escopo**: as três fases da ADR-061 §Implementation Plan — ligação, motor, superfície

## Problem Statement

O `my-memory` tem dois conceitos de memória: **memória de aprendizagem** (vault
central, Obsidian Flavored Markdown) e **code graph** (vault de repositório).
A separação está decidida e implementada desde a ADR-033. O **modelo interno do
vault de repositório** é que está errado: ele foi especificado como "mais notas"
(ADR-047, ADR-054) e nunca como code graph de primeira classe.

A consequência é mensurável. A feature de code AST da ADR-047 está **Accepted**
mas **inerte**:

- O motor nunca foi implementado — `newBackendParser()` devolve
  `ErrTreesitterDisabled` mesmo compilado com `-tags treesitter` e **não tem
  nenhum call site**; o caminho de produção é sempre o `MockParser` regex.
- **Não existe call graph.** As arestas são `references` por igualdade de nome;
  `imports` é pulado explicitamente.
- As **2 tools MCP de código estão mortas**: registradas com handler `nil`,
  respondem "indisponível" em produção — 2 das 26 tools MCP são decorativas.
- **`mem code-index` indexa 0 arquivos por default** (filtra por
  `cfg.ShouldIndex`, cujo `include` default é `**/*.md`), e `mem index` nunca
  chama o code pipeline.
- **Em Postgres** (engine do ambiente real), as tabelas `code_*` são criadas e
  **permanecem vazias**: o pipeline abre o store PG e o descartar, escrevendo num
  SQLite local que nada consulta.
- `code_*` **não tem coluna `repository`** — o code graph está fora da federação.
- `graph_edges` (notas) e `code_edges` (símbolos) **não têm ponte** — o
  "Repository Brain" não existe como grafo único.
- A viewer **não tem superfície de código**; o drift **não usa `ast_hash`**; o
  fan-in de código na busca híbrida é *placeholder*; 4 flags do CLI de código são
  aceitas e descartadas.

Medição do verificador independente sobre o próprio repo: `mem code-index
--include='**/*.go' --lang=go` → **331 arquivos, 2595 símbolos, 120 arestas**;
`db.InitDB`, a função mais chamada do repo, **não tem nenhum vizinho**.

A ADR-061 decide o caminho: **adotar o conceito do Graphify (arestas
estruturais com call-site, proveniência por aresta, hub threshold, orçamento de
tokens) mantendo o motor do my-memory** (busca híbrida FTS + TurboQuant 4-bit +
RRF, `epistemic_status` da ADR-011, PageRank, LPA + Q, blast radius, viewer
SPA, federação `repo_id`). O backend do AST vira **Go puro** (`go/parser`/
`go/ast`), coerente com a ADR-022, e o tree-sitter (CGO) permanece **deferido**
conforme o plano da ADR-047.

Esta spec fecha os 12 gaps da ISSUE-027.

## User Stories

### US-1 — Como agente, quero que o code graph indexe o código do repo, para orientá-lo sem grep

Como usuário de um repositório com código, quero que o `my-memory` indexe o
código-fonte do vault de repositório — com o **escopo de código separado** do
escopo de notas — e que o indexador comum (`mem index`) rode esse estágio, para
que o code graph exista sem um comando extra.

**Acceptance Criteria**:

1. WHEN `mem code-index` runs, the system SHALL select the files to parse using a **code scope** (`code.include`/`code.exclude` in the config) and SHALL NOT reuse the Markdown `include`/`exclude` filter for that selection.
2. WHEN the config has no `code.include`, the system SHALL default the code scope to the source extensions of the **active backend** and SHALL index a non-empty file set for a Go repository.
3. WHEN `mem index` runs with the code stage enabled, the system SHALL execute the code pipeline for the resolved repository and SHALL persist `code_files`/`code_symbols`/`code_edges` alongside the Markdown documents.
4. IF the Markdown `include` glob matches a source file (e.g. `**/*.go`), THEN the system SHALL NOT index that file as a document, and SHALL NOT create embeddings, graph edges or drift entries for it as a note.

### US-2 — Como operador, quero que o code graph fique no mesmo banco que o resto, para poder consultá-lo

Como operador, quero que o code graph seja gravado no **engine configurado**
(Postgres quando Postgres, SQLite quando SQLite) e isolado por repositório,
para que as consultas do vault de repositório encontrem os dados.

**Acceptance Criteria**:

5. WHEN the configured storage engine is Postgres, the code pipeline SHALL write `code_files`, `code_symbols` and `code_edges` to the Postgres database instead of a local SQLite file.
6. The system SHALL store a `repository` column on `code_files`, `code_symbols` and `code_edges`, and SHALL populate it with the resolved `repo_id` of the vault.
7. WHEN a code query runs with a repository filter, the system SHALL restrict results to the rows whose `repository` matches, and SHALL NOT return rows from another repository.

### US-3 — Como agente, quero que as tools MCP de código respondam, para consultar o grafo

Como agente conectado ao servidor MCP, quero que `memory_code_search` e
`memory_code_neighbors` devolvam dados reais, e que as flags do CLI de código
façam o que dizem, para não depender de adivinhação.

**Acceptance Criteria**:

8. WHEN the MCP server starts, the system SHALL inject the code search and code neighbors executors into the server, and `memory_code_search` and `memory_code_neighbors` SHALL return results instead of the "unavailable" message.
9. WHEN `mem code-index --repo <slug>` is invoked, the system SHALL write to the database of that repository, and SHALL NOT discard the parsed `--repo` value.
10. The CLI of the code stage SHALL NOT accept a flag whose value it discards: each remaining flag SHALL either change behaviour or be removed from the usage text.

### US-4 — Como agente, quero arestas estruturais reais, para saber quem chama o quê

Como agente, quero que o code graph tenha arestas `calls`, `imports` e
`inherits` extraídas do AST, com o **ponto de chamada**, para responder "quem
chama isto" com prova.

**Acceptance Criteria**:

11. The system SHALL extract, from the Go AST, the symbols with their exact positions, and SHALL register `calls` edges carrying the **call site** (file and line of the call expression), not only the position of the definition.
12. The system SHALL register `imports` edges between the importing file's symbols and the imported package's symbols, and SHALL register `inherits`/`uses_type` edges for Go embedding and interface satisfaction.
13. The system SHALL provide a real AST backend in the default build, implemented with the Go standard library (`go/parser`/`go/ast`), and SHALL NOT require a C compiler or a build tag for the code graph to work.
14. WHERE a build provides the `treesitter` tag, the system SHALL NOT degrade the code graph: if the tree-sitter binding is not implemented (the current state — the ADR-047 deferral is intact with 0 of 3 triggers fired), the system SHALL fall back to the Go backend and say so in the boot log, and SHALL NOT silently index fewer symbols or zero edges. The Go backend SHALL remain the fallback when the tag is absent.
    *Amendment 2026-10-24 (verificação independente):* a AC original exigia que o backend tree-sitter substituísse o de Go quando a tag estivesse presente — o que é **insatisfazível**, porque o binding não existe. O requisito real é o oposto: a tag **não pode** degradar o grafo em silêncio.

### US-5 — Como auditor, quero proveniência em cada aresta de código, para confiar no que o grafo afirma

Como auditor, quero saber se uma aresta foi extraída do AST ou inferida, e com
que confiança, para não tratar uma coincidência de nome como uma chamada.

**Acceptance Criteria**:

15. The system SHALL record an `epistemic_status` on every `code_edges` row, using `EXTRACTED` for edges derived from the AST and `INFERRED` for edges resolved by name or scope, aligning the code graph with `graph_edges` (ADR-011).
16. The system SHALL route an edge whose target cannot be resolved uniquely into `code_edges_uncertain` with its confidence and reason, and SHALL NOT expose it as a resolved edge.
17. WHEN a code tool returns an edge, the response SHALL carry the edge's `epistemic_status`, its `confidence` and the call site, in addition to the destination symbol's file and lines.

### US-6 — Como agente, quero que nota e código se liguem, para ter um Repository Brain de verdade

Como agente, quero que uma nota que menciona um símbolo se conecte a ele, para
navegar do porquê para o onde.

**Acceptance Criteria**:

18. WHEN a document references a symbol that exists in `code_symbols`, the system SHALL create a typed edge from the document to that symbol during indexing.
19. WHEN `memory_code_neighbors` is called for a symbol, the response SHALL include the documents that reference that symbol, and `memory_get_impact` SHALL reach code symbols starting from a document.

### US-7 — Como agente, quero que a busca híbrida considere o código, para não perder resultado

Como agente, quero que a busca híbrida inclua os símbolos de código no RRF,
para que uma pergunta sobre comportamento encontre a função, não só a prosa.

**Acceptance Criteria**:

20. WHEN `memory_search` runs with `include_code=true`, the system SHALL merge `code_symbols` into the RRF ranking as a real fan-in source, and SHALL NOT emit the "CA-12 placeholder" warning.
21. WHEN `include_code` is omitted or false, the system SHALL preserve the current ranking behaviour and SHALL NOT change any existing result set.

### US-8 — Como agente, quero que a travessia do grafo respeite orçamento, para não despejar tokens

Como agente, quero que a travessia de código corte o subgrafo por orçamento de
tokens e por grau dos hubs, para receber contexto denso em vez de uma explosão
de nós centrais.

**Acceptance Criteria**:

22. The code traversal SHALL apply a hub threshold (percentile of node degree) so that a handful of high-degree nodes SHALL NOT dominate the returned subgraph.
23. The code tools SHALL respect a token budget on the response, SHALL order the results by hop distance from the seed, and WHEN the response is truncated SHALL emit a hint that the query should be narrowed.

### US-9 — Como humano, quero ver o code graph na viewer, para inspecionar o que o agente enxerga

Como humano, quero uma superfície de código na viewer, para navegar o mesmo
grafo que o agente consulta.

**Acceptance Criteria**:

24. The viewer SHALL provide a route that lists the symbols of a repository and shows, for a selected symbol, its structural neighbours with the kind of edge and the call site.
25. The viewer SHALL NOT require a build-time dependency on a CDN to render the code graph, and the bundle SHALL respect the 160 kB gzip budget of the CI job.

### US-10 — Como mantenedor, quero que a mudança estrutural do código apareça no drift, para saber que a nota ficou para trás

Como mantenedor, quero que o drift aponte mudança de AST, para que a nota que
descreve o código seja revisada quando a estrutura muda.

**Acceptance Criteria**:

26. WHEN `ast_hash` of a file changes while the note graph stays unchanged, the drift detector SHALL report the structural change of that file and SHALL attribute it to the documents that reference its symbols.
27. WHEN the drift detector runs, it SHALL read `ast_hash` from `code_files` and SHALL NOT require the code stage to have been run in the same session.

### US-11 — Como mantenedor, quero registrar uma lição no vault central, para a aprendizagem acumular

Como mantenedor, quero transformar uma lição aprendida em nota do vault central,
com taxonomia e resumo, para que ela volte em consultas futuras.

**Acceptance Criteria**:

28. The system SHALL provide a write path that records a lesson in the central vault as a note with `category: memory`, a `summary` for L0 retrieval and a timestamp, and SHALL index it immediately.
29. WHEN a lesson is recorded, the system SHALL NOT require it to live in the repository vault, and the central vault SHALL remain navigable as plain Markdown in Obsidian.

## Out of Scope

- **Promover o binding tree-sitter (CGO).** O plano de deferral da ADR-047
  permanece intacto, com 0 de 3 gatilhos disparados. A ADR-047 **não** é
  supersedida por esta feature, e nenhuma dependência CGO entra no `go.mod`.
- **Implementar `gopls`/LSP.** Continua proibido pela ADR-047 até os gatilhos.
- **Parsing de linguagens além de Go.** O backend v1 é Go puro; as outras 9
  linguagens declaradas (`python`, `typescript`, `javascript`, `rust`, `java`,
  `cpp`, `ruby`, `php`, `shell`) seguem no mock até o tree-sitter ser promovido
  ou um parser puro existir. O escopo de código padrão cobre as **10**
  linguagens declaradas, mas **apenas Go tem extração AST precisa**
  (`go/parser`/`go/ast`); nas outras 9 os símbolos saem do mock heurístico
  (`def `, `function `, `fn `, `class `, `struct `, `interface `) e **não** têm
  call-site nem aresta estrutural confiável. O escopo padrão **não** promete
  AST para todas elas.
- **Mudança no vault central.** O conceito de memória de aprendizagem do cofre
  central (Obsidian Flavored Markdown, taxonomia `category`, `summary`) está
  decidido e não muda aqui. A US-11 só abre a **via de escrita** de uma lição.
- **Embeddings por símbolo.** `code_symbols` continua sem vetor próprio; o
  fan-in da US-7 entra pelo RRF com o score do símbolo.
- **PR review, triagem de incidente e verificação formal.** São produto
  fechado do Graphify, sem equivalente open source.
- **Navegador multi-repo do Graphify.** O equivalente já existe pela federação
  `repo_id` (ADR-033/034).

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale |
|---|---|---|
| Linguagem do backend AST v1 | Go (`go/parser`/`go/ast`), Go puro | ADR-022 exige Go puro sem CGO; a máquina de desenvolvimento não tem compilador C (`CGO_ENABLED=0`); o repositório e sua audiência são Go-first |
| Granularidade da resolução de chamada | `EXTRACTED` quando o callee resolve unicamente no escopo do pacote; `INFERRED` quando resolve só por nome; ambíguo vai para `code_edges_uncertain` | É o que o schema `confidence` + `code_edges_uncertain` já expressa; evita fingir precisão que o mock não tem |
| Escopo de código vs escopo de notas | Seções separadas `code.include`/`code.exclude`, com `include` global intocado | O `include` global é compartilhado com o pipeline Markdown; alterá-lo indexaria código como nota |
| Onde o code graph é gravado | No engine configurado (Postgres quando Postgres) | A ADR-047 promete "roda igual sobre SQLite local ou Postgres/pgvector"; hoje a promessa não vale |
| Granularidade do fan-in de código | `code_symbols` entra no RRF com o mesmo formato de resultado das notas | Mantém uma única resposta para o agente, sem tool nova |
| Orçamento de tokens da travessia | Eveno por token, truncamento com dica de estreitar a query | O Graphify usa `chars = budget*3`; o my-memory já tem empacotamento por orçamento (ADR-029) |
| Surface da viewer | Uma rota de código, sem renderizador novo | A viewer já é SPA servida pelo MCP (ADR-059/060); vis-network vendorizado é a ADR-058, **não implementada** |
| Repositório canônico da análise | O clone do Graphify `v0.9.94` em `D:\.minimax\scratch\subagents\graphify-study\graphify` | Referência de modelo; nenhuma dependência de runtime é criada |

**Open questions: none.** As decisões de produto foram tomadas pelo dono em
2026-10-23 e registradas na ADR-061.

## Requirement Traceability

| Requirement | Gap | Fase | Artefato |
|---|---|---|---|
| CG-01 | C4 | ligação | escopo de código no config |
| CG-02 | C4 | ligação | default do escopo de código |
| CG-03 | C4 | ligação | code pipeline como estágio do `mem index` |
| CG-04 | C4 | ligação | `include` global não indexa código como nota |
| CG-05 | C5 | ligação | escrita no engine configurado |
| CG-06 | C6 | ligação | coluna `repository` em `code_*` |
| CG-07 | C6 | ligação | filtro por repositório nas queries de código |
| CG-08 | C3 | ligação | handlers MCP de código injetados |
| CG-09 | C12 | ligação | flags do CLI de código deixam de ser descartadas |
| CG-10 | C1 | motor | backend AST Go puro no build default |
| CG-11 | C2 | motor | arestas `calls` com call-site |
| CG-12 | C2 | motor | arestas `imports` e `inherits`/`uses_type` |
| CG-13 | C1 | motor | `newBackendParser()` deixa de ser código morto |
| CG-14 | C8 | motor | `epistemic_status` em `code_edges` |
| CG-15 | C8 | motor | aresta ambígua vai para `code_edges_uncertain` |
| CG-16 | C8 | motor | resposta MCP com proveniência, confiança e call-site |
| CG-17 | C7 | superfície | ponte `documents` ↔ `code_symbols` |
| CG-18 | C7 | superfície | documentos que referenciam o símbolo |
| CG-19 | C9 | superfície | fan-in RRF real de `code_symbols` |
| CG-20 | C9 | superfície | ranking sem `include_code` inalterado |
| CG-21 | — | superfície | hub threshold na travessia |
| CG-22 | — | superfície | orçamento de tokens e dica de estreitar |
| CG-23 | C10 | superfície | rota de código na viewer |
| CG-24 | C10 | superfície | viewer sem CDN e dentro do orçamento de bundle |
| CG-25 | C11 | superfície | drift de mudança estrutural |
| CG-26 | C11 | superfície | drift lê `ast_hash` sem exigir code stage no mesmo run |
| CG-27 | — | superfície | escrita de lição no vault central |
| CG-28 | — | superfície | lição indexada com `category: memory` e `summary` |
