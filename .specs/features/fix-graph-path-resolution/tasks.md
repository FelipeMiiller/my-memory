# Tasks: fix-graph-path-resolution

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-16
**Issue**: ISSUE-046 (`.specs/ISSUES.md`) · fecha os gaps FIX-1 e FIX-3..FIX-7 da feature `graph-path-discovery` (ADR-027). **FIX-2 (ISSUE-047) fica de fora** — é dependência externa, spec `fix-graph-node-identity`.
**Merged**: 2026-11-16 — união das duas versões da ISSUE-046. A spec irmã `fix-graph-path-discovery-gaps` foi **descartada** pelo dono; esta é a versão final. O desempate determinístico e o relatório de ambiguidade (CLI) e o `InvalidParams` (tool MCP) convivem aqui.

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Design**: inline nesta spec (resolução determinística **e** relatório de ambiguidade + 2 resolvers + asserções + defaults em fonte única + emenda do spec da feature + ciclo de re-verificação; **o motor de pathfinding não é tocado**)
**Status**: Draft

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec. Guidelines found: `AGENTS.md` (regra 4 — testes obrigatórios; regra 8 — quality gate), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`, `.github/workflows/ci.yml` (comandos de teste reais do projeto). Lições aplicadas: L-020 (branch dedicado só é coberto quando a asserção fixa o que ele produz de diferente), L-021 (filtro por repositório exige dois repositórios no teste), L-022 (pino no critério o campo de saída que o usuário precisa), L-023 (resolução ambígua precisa de desambiguação explícita), L-024 (flag e sua inversa: qual vence), L-025 (helper de default chamado só por teste está morto em produção), L-026 (resolva o nó para a identidade que as arestas usam — FIX-2, ISSUE-047).

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Contrato de resolução e relatório de ambiguidade | unit | 1:1 com GPR-02, GPR-07, GPR-09 e GPR-10: todo candidato listado com caminho relativo e título, ordem por caminho e depois por id, chaves de desambiguação nomeadas, predicado de erro | `internal/graph/resolve_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestAmbiguousNode ./internal/graph/...` |
| Resolução de nó no SQLite | integration | 1:1 com GPR-01, GPR-02, GPR-03, GPR-04, GPR-05, GPR-06 e GPR-07: ordem das chaves, desempate determinístico (documento do mesmo repositório → menos segmentos → id menor), ambiguidade **reportada** e não fatal, caminho relativo resolvendo para o documento | `internal/db/graph_resolve_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestResolveNodeCanonicalID ./internal/db/...` |
| Resolução de nó no PostgreSQL | integration (PostgreSQL real) | 1:1 com GPR-01, GPR-02, GPR-03, GPR-04, GPR-05, GPR-06, GPR-07 e GPR-08: a mesma matriz do SQLite, mais o filtro de repositório e o **mesmo** id canônico esperado nos dois motores | `internal/store/postgres_resolve_test.go` | `MY_MEMORY_PG_TEST=1 go -C D:\repository\my-memory test -count=1 -run TestPostgresResolveNodeCanonicalID ./internal/store/...` |
| Superfície da CLI (relatório de ambiguidade) | integration | 1:1 com GPR-11, GPR-12, GPR-13 e GPR-14: id canônico nomeado, caminho e título de cada candidato, chaves `title:` e caminho relativo nomeadas, e a CLI **não** falha | `cmd/mem/path_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestPathCLI ./cmd/mem/...` |
| Superfície da tool MCP (ambiguidade) | unit | 1:1 com GPR-15, GPR-16 e GPR-17: `InvalidParams` na ambiguidade com os candidatos nomeados, `CodeInternalError` no controle negativo | `internal/mcp/path_handlers_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestMemoryFindPath ./internal/mcp/...` |
| Superfície da tool MCP (headings e código de erro) | unit | 1:1 com GPR-31 e GPR-32: código `InvalidParams` sem `source`/`target` e os headings `Rota Conectada` e `Segmentos Percorridos` | `internal/mcp/path_handlers_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestMemoryFindPath ./internal/mcp/...` |
| Motor — ramo do nó idêntico | unit | 1:1 com GPR-18 e GPR-19: 0 saltos, custo 0, 1 nó, 0 arestas e o `Summary` literal — **é o `Summary` que distingue o ramo do Dijkstra genérico e mata o M7** | `internal/graph/path_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestFindPath_Identical ./internal/graph/...` |
| Motor — detalhes da aresta devolvida | unit | 1:1 com GPR-21, GPR-22, GPR-23 e GPR-24: relação, status epistêmico e peso (1.0 / 0.6 / 0.3) de **cada** aresta devolvida, com a relação (`links_to` para as epistêmicas) | `internal/graph/path_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestFindPath_EdgeDetails ./internal/graph/...` |
| Superfície da CLI (not-found e nó idêntico) | integration | 1:1 com GPR-20, GPR-28, GPR-29 e GPR-30: linha `Status:          ❌ Não encontrado`, linha `Detalhes:` com o `Summary` e a renderização do nó idêntico | `cmd/mem/path_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestPathCLI ./cmd/mem/...` |
| Defaults de caminho (CLI + tool MCP) | integration | 1:1 com GPR-33, GPR-34, GPR-35 e GPR-36: os defaults aplicados vêm de `graph.DefaultPathOptions()` e nenhum default de caminho fica literal fora dele | `cmd/mem/path_test.go` · `internal/mcp/path_handlers_test.go` | `go -C D:\repository\my-memory test -count=1 ./cmd/mem/... ./internal/mcp/...` |
| Caminho no PostgreSQL (isolamento por repositório) | integration (PostgreSQL real) | 1:1 com GPR-25, GPR-26 e GPR-27: rota de 2 saltos **e** nenhum nó do outro repositório — dois repositórios no mesmo banco (o self-path não exercita o filtro) | `internal/store/postgres_test.go` | `MY_MEMORY_PG_TEST=1 go -C D:\repository\my-memory test -count=1 -run TestPostgresStore_Integration ./internal/store/...` |
| Texto do spec da feature | none (documento) | build gate only — as 7 decisões (S1–S7) e a dependência do FIX-2 estão escritas no spec da feature | `.specs/features/graph-path-discovery/spec.md` | `python3 D:\repository\my-memory\.agents\skills\tlc-spec-driven\scripts\validate_spec.py graph-path-discovery --root D:\repository\my-memory` |
| Verificação independente + sensor | integration | 1:1 com GPR-43 a GPR-47: M7 morre, a mutação composta do FIX-7 morre, M1–M6 continuam mortas, veredito PASS com evidência `file:line` | `.specs/features/graph-path-discovery/validation.md` | `python3 D:\repository\my-memory\.agents\skills\tlc-spec-driven\scripts\validate_state.py graph-path-discovery --root D:\repository\my-memory` |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após cada task de teste sem Postgres | `go -C D:\repository\my-memory test -count=1 ./internal/graph/... ./internal/db/... ./internal/mcp/... ./cmd/mem/...` |
| Quick | Após cada task de produção | `go -C D:\repository\my-memory build ./...` e `go -C D:\repository\my-memory vet ./...` |
| Full | Após T3 e T12 (Postgres) e no gate final | `MY_MEMORY_PG_TEST=1 go -C D:\repository\my-memory test -count=1 ./...` (com `MY_MEMORY_PG_URL` preservada — nunca impressa) |
| Full | Gate final taggeado | `go -C D:\repository\my-memory test -tags "sqlite_fts5 onnx" -count=1 ./...` |
| Build | Antes de declarar pronto | `go -C D:\repository\my-memory build ./...` |
| Format | Antes de declarar pronto | `gofmt -l` nos `.go` tocados, medido em **cópia LF** (o working tree local mistura CRLF por causa do checkout antigo; a CI roda em Linux) |
| Spec | Antes de entregar | `python3 D:\repository\my-memory\.agents\skills\tlc-spec-driven\scripts\validate_spec.py fix-graph-path-resolution --root D:\repository\my-memory` e `python3 D:\repository\my-memory\.agents\skills\tlc-spec-driven\scripts\validate_tasks.py fix-graph-path-resolution --root D:\repository\my-memory` |
| Spec | Depois de T13 | `python3 D:\repository\my-memory\.agents\skills\tlc-spec-driven\scripts\validate_spec.py graph-path-discovery --root D:\repository\my-memory` |
| State | Antes de entregar | `python3 D:\repository\my-memory\.agents\skills\tlc-spec-driven\scripts\validate_state.py graph-path-discovery --root D:\repository\my-memory` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: O contrato da resolução e da ambiguidade

O tipo que carrega os candidatos e o relatório de ambiguidade; depois os dois resolvers, que coletam candidatos em vez de escolher o primeiro e aplicam o desempate determinístico.

```
T1
T1 ──→ T2
T1 ──→ T3
```

### Phase 2: As superfícies (CLI e tool MCP)

A CLI passa a **informar** a ambiguidade e a **nomear** o id resolvido, sem falhar; a tool MCP passa a devolver `InvalidParams` com os candidatos nomeados.

```
T2 ──→ T4
T3 ──→ T4
T2 ──→ T5
T3 ──→ T5
T5 ──→ T6
T4 ──→ T7
```

### Phase 3: Defaults em fonte única, motor e Postgres

A CLI e a tool MCP passam a ler os defaults de `graph.DefaultPathOptions()`; o motor ganha as asserções que matam o M7 e que fixam o status/peso/relação; o Postgres ganha a integração de dois repositórios.

```
T7 ──→ T8
T5 ──→ T9
T6 ──→ T9
T10 ──→ T11
T3 ──→ T12
```

### Phase 4: Emenda do spec e re-verificação independente

O spec da feature recebe as 7 decisões (S1–S7) e a dependência do FIX-2; depois o Verifier independente roda o sensor. A T14 fecha **todas** as tasks anteriores.

```
T2 ──→ T14
T3 ──→ T14
T4 ──→ T14
T5 ──→ T14
T6 ──→ T14
T7 ──→ T14
T8 ──→ T14
T9 ──→ T14
T10 ──→ T14
T11 ──→ T14
T12 ──→ T14
T13 ──→ T14
```

---

## Task Breakdown

### T1: O contrato da resolução e do relatório de ambiguidade

**What**: em `internal/graph`, um tipo novo que carrega o **relatório de ambiguidade** — o identificador consultado e **todos** os candidatos casados (id, caminho relativo sem a raiz do vault, título) — ordenado por caminho relativo e depois por id, com as **chaves de desambiguação** nomeadas (caminho relativo, com ou sem `.md`, e `title:`), mais um predicado que reconhece um erro de ambiguidade; o `graph.PathResult` ganha o campo que carrega esse relatório.
**Where**: `internal/graph/resolve.go`
**Depends on**: None
**Reuses**: as 4 chaves de match do resolver de disco (`internal/vaultfs/target.go:14-24`) e a ordem Obsidian de resolução (`internal/vaultfs/target.go:127-167`) · o padrão de erro do próprio pacote `internal/graph` · `graph.PathResult` (`internal/graph/path.go:45-57`)
**Requirement**: GPR-02, GPR-07, GPR-09, GPR-10
**Gap**: FIX-3

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O relatório carrega **todos** os candidatos casados — nenhum é descartado nem truncado
- [x] Cada candidato carrega o id, o caminho relativo (sem a raiz do vault) e o título do documento
- [x] A mensagem do relatório lista um candidato por linha e nomeia as chaves que o endereçam (caminho relativo, com ou sem `.md`, e `title:`)
- [x] A ordem dos candidatos é por caminho relativo e, no empate, por id
- [x] Há um predicado exported que reconhece o erro de ambiguidade (o `errors.As` funciona)
- [x] O `graph.PathResult` tem onde carregar o relatório, e a CLI e a tool MCP leem **o mesmo** relatório
- [x] Teste unitário novo em `internal/graph/resolve_test.go` cobre 1 candidato, 2 candidatos e 0 candidato
- [x] Gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./internal/graph/...`
- [x] Test count: os testes de `internal/graph` passam, sem nenhuma deleção silenciosa
- [x] O **algoritmo** de `internal/graph/path.go` não foi tocado — o motor (Dijkstra, pruning, inversão epistêmica) fica intacto; só o `PathResult` ganhou o campo

**Tests**: unit
**Gate**: quick

**Commit**: `feat(graph): add typed ambiguity report for node resolution`

---

### T2: O resolver do SQLite coleta candidatos e ordena o desempate

**What**: `ResolveNodeCanonicalID` no SQLite passa a **coletar** os candidatos de cada chave — id do documento, caminho relativo, nome do nó, `title:` do documento e fuzzy — em vez de pegar o primeiro com `LIMIT 1`; ordena os candidatos pelo critério de desempate (documento do mesmo repositório → menos segmentos de caminho → id lexicograficamente menor) e devolve o id canônico **junto com** o relatório de ambiguidade.
**Where**: `internal/db/graph.go`
**Depends on**: T1
**Reuses**: o relatório de ambiguidade do T1 · a semântica de resolução exata da ADR-063 (`internal/vaultfs/target.go:134-167`) · `db.QueryContext` e `db.QueryRowContext`
**Requirement**: GPR-01, GPR-02, GPR-03, GPR-04, GPR-05, GPR-06, GPR-07
**Gap**: FIX-3

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] Nenhuma query de resolução usa mais `LIMIT 1` sem `ORDER BY` para escolher entre candidatos (`internal/db/graph.go:596-600`, `:606-610`, `:616-620` e `:644-649`)
- [x] O passo difuso (LIKE, `internal/db/graph.go:628-638`) tem a **mesma** ordem de desempate — e o `ORDER BY` que ele já tinha ganha um desempate estável por id
- [x] A chave do caminho relativo resolve para o documento do caminho (com ou sem `.md`) e **não** cai no nome
- [x] Entre nós de mesmo nome, vence o que é documento do mesmo repositório; persistindo o empate, vence o caminho com menos segmentos e depois o id lexicograficamente menor
- [x] Dois nós de mesmo nome **não** viram erro: o resultado traz o id canônico e o relatório com os dois candidatos
- [x] O mesmo identificador resolvido duas vezes devolve o mesmo id
- [x] Teste novo em `internal/db/graph_resolve_test.go` cobre: chave exata, caminho relativo, desempate, ambiguidade (2 candidatos) e ordem dos candidatos
- [x] Gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./internal/db/...`
- [x] Test count: os testes de `internal/db` passam, sem nenhuma deleção silenciosa

**Tests**: integration
**Gate**: quick

**Commit**: `fix(db): collect node candidates and order the tie-break`

---

### T3: O resolver do Postgres coleta candidatos e ordena o desempate

**What**: o mesmo contrato de T2 no `PostgresStore.ResolveNodeCanonicalID`, **preservando** o filtro `WHERE ($1 = '' OR repository = $1)` — a coleta de candidatos acontece dentro do repositório, e o id canônico devolvido é o mesmo que o SQLite devolve para o mesmo identificador.
**Where**: `internal/store/postgres.go`
**Depends on**: T1
**Reuses**: o relatório de ambiguidade do T1 · o filtro de repositório já existente (`internal/store/postgres.go:1031`, `:1041`, `:1051`) · o padrão de banco descartável `internal/pgtest`
**Requirement**: GPR-01, GPR-02, GPR-03, GPR-04, GPR-05, GPR-06, GPR-07, GPR-08
**Gap**: FIX-3

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] Nenhuma query de resolução usa mais `LIMIT 1` sem `ORDER BY` para escolher entre candidatos (`internal/store/postgres.go:1029-1033`, `:1039-1043`, `:1049-1053` e `:1076-1081`)
- [x] O passo difuso (LIKE, `internal/store/postgres.go:1060-1070`) tem a **mesma** ordem de desempate de T2
- [x] A coleta de candidatos **respeta** o filtro de repositório: candidatos de outro repositório nunca entram na lista
- [x] Dois nós de mesmo nome **no mesmo repositório** produzem o relatório de ambiguidade com os dois candidatos
- [x] O id canônico devolvido é **o mesmo** que o teste do SQLite afirma para o mesmo identificador (GPR-08)
- [x] O teste usa `pgtest.New(t)` (banco descartável) e **nunca** o Postgres configurado do dono; sem `MY_MEMORY_PG_TEST=1` ele **pulsa** (skip) em vez de falhar
- [x] Gate completo passa: `MY_MEMORY_PG_TEST=1 go -C D:\repository\my-memory test -count=1 ./internal/store/...`
- [x] Test count: os testes de `internal/store` passam, sem nenhuma deleção silenciosa

**Tests**: integration
**Gate**: full

**Commit**: `fix(store): collect node candidates and order the tie-break in postgres`

---

### T4: A CLI informa a ambiguidade e nomeia o id resolvido

**What**: `mem path` passa a **imprimir** o relatório de ambiguidade quando a resolução do identificador casou com mais de um candidato — caminho relativo e título de cada candidato e as chaves de desambiguação (`title:` e caminho relativo, com o sufixo `.md` opcional) — nomeia o **id canônico resolvido** e **segue** devolvendo a rota em vez de falhar.
**Where**: `cmd/mem/path.go`
**Depends on**: T2, T3
**Reuses**: o relatório de ambiguidade do T1 · `runPathCommand` com `io.Writer` injetável (`cmd/mem/path.go:24`) · o cabeçalho da saída (`cmd/mem/path.go:114-123`)
**Requirement**: GPR-11, GPR-12, GPR-13, GPR-14
**Gap**: FIX-3

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] A saída nomeia o **id canônico resolvido** (origem e destino)
- [x] Quando há mais de um candidato, a saída traz o caminho relativo e o título de **todos** eles
- [x] A saída nomeia as chaves: o literal `title:` e o caminho relativo, deixando o sufixo `.md` opcional
- [x] A CLI **não falha** com a ambiguidade: termina com sucesso e mostra o resultado da rota
- [x] O relatório sai **antes** do bloco `Status:`, para que o usuário veja a ambiguidade antes do resultado
- [x] Com um único candidato, **nada** de relatório é impresso (controle negativo)
- [x] Nenhuma flag nova é criada para desambiguar — a desambiguação usa o caminho relativo e o `title:`
- [x] Gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./cmd/mem/...`
- [x] Test count: os testes de `cmd/mem` passam, sem nenhuma deleção silenciosa

**Tests**: integration
**Gate**: quick

**Commit**: `feat(cli): report path ambiguity and name the resolved node`

---

### T5: A tool MCP devolve `InvalidParams` quando o nó é ambíguo

**What**: o handler de `memory_find_path` deixa de embrulhar o erro de ambiguidade em `CodeInternalError` e o devolve como `InvalidParams`, com no texto o caminho relativo e o título de cada candidato e as chaves de desambiguação — ambiguidade é parâmetro inválido, não falha de servidor.
**Where**: `internal/mcp/path_handlers.go`
**Depends on**: T2, T3
**Reuses**: `NewError` e `CodeInvalidParams` (`internal/mcp/path_handlers.go:31`, `:40`) · o predicado de erro de ambiguidade do T1 · `errors.As` para separar o caso
**Requirement**: GPR-02, GPR-14, GPR-15, GPR-16, GPR-17
**Gap**: FIX-3

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O erro de ambiguidade sai com o código `InvalidParams` (e **não** `CodeInternalError`)
- [x] O texto do erro mantém o caminho relativo e o título de cada candidato e as chaves de desambiguação
- [x] A tool **não** devolve rota nenhuma quando a resolução é ambígua — ela falha
- [x] Um erro que **não** é de ambiguidade continua saindo como `CodeInternalError` (controle negativo)
- [x] Gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./internal/mcp/...`
- [x] Test count: os testes de `internal/mcp` passam, sem nenhuma deleção silenciosa

**Tests**: unit
**Gate**: quick

**Commit**: `fix(mcp): return invalid params for ambiguous path nodes`

---

### T6: A tool MCP tem asserção dos headings e do código de erro

**What**: asserções na tool MCP para o código `InvalidParams` (sem `source`, sem `target`) e para os headings `Rota Conectada` e `Segmentos Percorridos` na resposta Markdown de uma rota encontrada.
**Where**: `internal/mcp/path_handlers_test.go`
**Depends on**: T5
**Reuses**: o fixture de handler do próprio arquivo (`internal/mcp/path_handlers_test.go:13`) · `NewTextResult` (`internal/mcp/handlers.go:40`) · `CodeInvalidParams`
**Requirement**: GPR-31, GPR-32
**Gap**: FIX-6

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O teste sem `source` e o teste sem `target` afirmam o **código** `InvalidParams` (hoje só o texto é afirmar)
- [x] A resposta de uma rota encontrada contém `Rota Conectada` **e** `Segmentos Percorridos`
- [x] O caso "nenhum caminho" **não** contém `Rota Conectada` (controle negativo que prova que o heading não é decoration)
- [x] Gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./internal/mcp/...`
- [x] Test count: os testes de `internal/mcp` passam, sem nenhuma deleção silenciosa

**Tests**: unit
**Gate**: quick

**Commit**: `test(mcp): assert path headings and invalid params code`

---

### T7: A saída de not-found e de nó idêntico da CLI tem asserção

**What**: um teste na CLI que monta um grafo sem rota e um grafo com origem = destino, e afirma a linha de status `Status:          ❌ Não encontrado`, a linha `Detalhes:` carregando o `Summary` e a renderização `Origem e destino são o mesmo nó ('<id>').`.
**Where**: `cmd/mem/path_test.go`
**Depends on**: T4
**Reuses**: o fixture de banco temporário do próprio arquivo (`cmd/mem/path_test.go:34-52`) · `db.InsertDocument` e `db.InsertEdgeWithProps` · `runPathCommand` com `io.Writer` injetável
**Requirement**: GPR-20, GPR-28, GPR-29, GPR-30
**Gap**: FIX-1, FIX-6

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O caso sem rota afirma a linha `Status:          ❌ Não encontrado` **e** o `Summary` na linha `Detalhes:`
- [x] O caso origem = destino afirma `Origem e destino são o mesmo nó ('<id>').`
- [x] O caso origem = destino é o **controle negativo** do M7: a renderização da CLI decide por `Hops == 0` (`cmd/mem/path.go:135`), então ela **não** substitui a asserção do `Summary` (T10) — as duas juntas provam o ramo
- [x] O teste usa `t.TempDir()` e não escreve em nenhum vault nem em `.memory/memory.db`
- [x] Gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./cmd/mem/...`
- [x] Test count: os testes de `cmd/mem` passam, sem nenhuma deleção silenciosa

**Tests**: integration
**Gate**: quick

**Commit**: `test(cli): assert path not-found and identical-node output`

---

### T8: A CLI lê os defaults de `graph.DefaultPathOptions()`

**What**: `mem path` deixa de re-declarar `max_depth=6`, `directed=true` e `mode=epistemic` como literais e passa a derivar os três valores de `graph.DefaultPathOptions()`; o default efetivo continua sendo 6 / true / epistemic, e a saída mostra o modo e a profundidade aplicados.
**Where**: `cmd/mem/path.go`
**Depends on**: T7
**Reuses**: `graph.DefaultPathOptions()` (`internal/graph/path.go:26-32`) · a precedência `--undirected` já existente (`cmd/mem/path.go:58-61`) · o reset de `--max-depth` não-positivo (`cmd/mem/path.go:54-56`)
**Requirement**: GPR-33, GPR-34, GPR-36
**Gap**: FIX-7

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O default de `--max-depth`, de direção e de `--mode` vem de `graph.DefaultPathOptions()`
- [x] O default efetivo continua sendo **6** / **true** / **epistemic** e a saída mostra `Profundidade:    6 saltos máx` e `Modo:            epistemic`
- [x] O `grep` não acha mais literal de default de caminho em `cmd/mem/path.go` (GPR-36)
- [x] A precedência `--undirected` sobre `--directed` e o reset de `--max-depth` ≤ 0 continuam valendo (comportamento vigente, ver Edge Cases)
- [x] Gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./cmd/mem/...`
- [x] Test count: os testes de `cmd/mem` passam, sem nenhuma deleção silenciosa

**Tests**: integration
**Gate**: quick

**Commit**: `refactor(cli): read path defaults from the graph helper`

---

### T9: A tool MCP lê os defaults de `graph.DefaultPathOptions()`

**What**: `memory_find_path` deixa de re-declarar `max_depth=6`, `directed=true` e `mode=epistemic` como literais e passa a derivar os três valores de `graph.DefaultPathOptions()`; o default efetivo continua sendo 6 / true / epistemic, e a resposta Markdown mostra o modo e a profundidade aplicados.
**Where**: `internal/mcp/path_handlers.go`
**Depends on**: T5, T6
**Reuses**: `graph.DefaultPathOptions()` (`internal/graph/path.go:26-32`) · o reset de `max_depth` não-positivo já existente (`internal/mcp/path_handlers.go:44-49`)
**Requirement**: GPR-33, GPR-35, GPR-36
**Gap**: FIX-7

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O default de `max_depth`, de `directed` e de `mode` vem de `graph.DefaultPathOptions()`
- [x] O default efetivo continua sendo **6** / **true** / **epistemic** e a resposta mostra `Limite de Profundidade: 6 saltos` e `Modo de Custo: \`epistemic\``
- [x] O `grep` não acha mais literal de default de caminho em `internal/mcp/path_handlers.go` (GPR-36)
- [x] `max_depth` não-positivo continua sendo ignorado em favor do default (comportamento vigente)
- [x] Gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./internal/mcp/...`
- [x] Test count: os testes de `internal/mcp` passam, sem nenhuma deleção silenciosa

**Tests**: unit
**Gate**: quick

**Commit**: `refactor(mcp): read path defaults from the graph helper`

---

### T10: O `Summary` do nó idêntico é asseverado (mata o M7)

**What**: um teste que fixa o `Summary` literal do ramo de nó idêntico — `Nó de origem e destino são idênticos ('<id>')` — e o resultado do ramo (0 saltos, custo 0, 1 nó, 0 arestas), de modo que desligar o ramo em `internal/graph/path.go:166` derrube a suíte.
**Where**: `internal/graph/path_test.go`
**Depends on**: None
**Reuses**: o fixture de grafo do próprio arquivo (`internal/graph/path_test.go:8-33`) · `graph.ResolveEdgeCost` e as constantes de peso (`internal/graph/pagerank.go:10-12`)
**Requirement**: GPR-18, GPR-19
**Gap**: FIX-1

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O teste afirma o `Summary` **exato** do nó idêntico, com o id interpolado
- [x] O teste afirma 0 saltos, custo total 0, exatamente 1 nó e 0 arestas
- [x] O teste afirma que o `Summary` do nó idêntico é **diferente** do `Summary` do caminho encontrado (`Caminho encontrado com ...`) — é essa diferença que distingue o ramo do Dijkstra genérico
- [x] **Prova do sensor**: desligar o ramo (`internal/graph/path.go:166` → `if false {`) faz **este** teste falhar — a asserção do `Summary`, e não a renderização da CLI (que decide por `Hops == 0`), é o que mata o M7
- [x] Gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./internal/graph/...`
- [x] Test count: os testes de `internal/graph` passam, sem nenhuma deleção silenciosa
- [x] Nenhum arquivo de produção foi tocado — o motor não muda

**Tests**: unit
**Gate**: quick

**Commit**: `test(graph): assert identical-node summary`

---

### T11: O status epistêmico, o peso e a relação das arestas são asseverados

**What**: um teste com um grafo que mistura os três status epistêmicos (`EXTRACTED`, `INFERRED`, `TAG`) e afirma, para **cada** aresta devolvida, que ela carrega a relação, o status epistêmico e o peso da aresta que percorreu, com o status exato de cada uma, o peso (1.0 / 0.6 / 0.3) e a relação — `links_to` quando a relação de origem é vazia ou epistêmica.
**Where**: `internal/graph/path_test.go`
**Depends on**: T10
**Reuses**: `graph.WeightedEdge` (`internal/graph/path.go:34-43`) · `ResolveEdgeWeight` (`internal/graph/pagerank.go:44-58`) · a normalização de status (`internal/graph/path.go:105-107`)
**Requirement**: GPR-21, GPR-22, GPR-23, GPR-24
**Gap**: FIX-4, S2, S6

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] Cada aresta devolvida tem status `EXTRACTED`, `INFERRED` ou `TAG` — o teste afirma o valor exato de cada uma
- [x] O peso de cada aresta é **1.0**, **0.6** ou **0.3**, conforme o status (caso sem peso customizado)
- [x] Cada aresta devolvida carrega a relação, o status e o peso **da aresta que ela percorreu**, e não de outra do grafo
- [x] Uma aresta cuja relação de origem é `EXTRACTED` (ou vazia) volta como `links_to`
- [x] Uma aresta de relação própria (`implements`, `depends_on`) volta **verbatim** — controle negativo da reescrita
- [x] Gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./internal/graph/...`
- [x] Test count: os testes de `internal/graph` passam, sem nenhuma deleção silenciosa
- [x] Nenhum arquivo de produção foi tocado

**Tests**: unit
**Gate**: quick

**Commit**: `test(graph): assert epistemic status and weight of returned edges`

---

### T12: O caminho no PostgreSQL tem rota multi-hop e isolamento por repositório

**What**: um teste de integração do `FindPath` do Postgres com **dois** repositórios no mesmo banco: uma rota de 2 saltos dentro do repositório A, um nó de mesmo nome no repositório B, e a asserção de que a rota devolvida para A não contém nenhum nó de B.
**Where**: `internal/store/postgres_test.go`
**Depends on**: T3
**Reuses**: o banco descartável `internal/pgtest` (`internal/pgtest/pgtest.go`) · `store.InsertDocument` e `store.InsertEdgeWithProps` (`internal/store/postgres_test.go:231`) · o filtro de repositório (`internal/store/postgres.go:1390-1392`)
**Requirement**: GPR-25, GPR-26, GPR-27
**Gap**: FIX-5

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O banco do teste tem **dois** repositórios (o self-path não exercita o filtro — L-021)
- [x] A rota de 2 saltos dentro do repositório A é devolvida com `Found = true` e `Hops = 2`
- [x] O teste **assere** a rota multi-hop — e não só o caso self-path (GPR-27)
- [x] Nenhum nó devolvido pertence ao repositório B
- [x] O nó homônimo de B **não** aparece como candidato de ambiguidade na resolução dentro de A (o filtro de repositório vale antes da contagem de candidatos)
- [x] O teste usa `pgtest.New(t)` e **nunca** o Postgres configurado do dono; sem `MY_MEMORY_PG_TEST=1` ele **pulsa** (skip)
- [x] Gate completo passa: `MY_MEMORY_PG_TEST=1 go -C D:\repository\my-memory test -count=1 ./internal/store/...`
- [x] Test count: os testes de `internal/store` passam, sem nenhuma deleção silenciosa

**Tests**: integration
**Gate**: full

**Commit**: `test(store): cover postgres path isolation and multi-hop route`

---

### T13: O spec da feature recebe as 7 decisões e a dependência do FIX-2

**What**: emenda em `.specs/features/graph-path-discovery/spec.md` que escreve as 7 decisões que o spec nunca tomou — precedência `--undirected` sobre `--directed` (S3), `--mode` inválido cai para `epistemic` (S4), `--max-depth` não-positivo volta para 6 (S5), literais de relação e de status epistêmico com seus pesos (S2, S6), os dois textos de `Summary` (S7) e a resposta **texto-only** da tool MCP com remissão à ISSUE-022 (S1) — e que registra a **dependência** do FIX-2 (ISSUE-047 / `fix-graph-node-identity`).
**Where**: `.specs/features/graph-path-discovery/spec.md`
**Depends on**: None
**Reuses**: as decisões e os literais já fixados nesta spec (seção Assumptions) · a ISSUE-022 e a ISSUE-047 em `.specs/ISSUES.md`
**Requirement**: GPR-37, GPR-38, GPR-39, GPR-40, GPR-41, GPR-42
**Gap**: S1–S7

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] As 7 decisões estão escritas no spec da feature, com os **literais** exatos (não "o texto de não-encontrado", mas o texto)
- [x] A história P3 da feature deixa de prometer "Markdown **e** JSON" e remete à ISSUE-022
- [x] O spec da feature declara que o `mem path` só acha caminho no vault real **depois** da spec `fix-graph-node-identity` (ISSUE-047) e do reindex
- [x] Nenhum AC do spec da feature é apagado ou afrouxado pela emenda
- [x] Gate de spec passa: `python3 D:\repository\my-memory\.agents\skills\tlc-spec-driven\scripts\validate_spec.py graph-path-discovery --root D:\repository\my-memory`
- [x] Test count: **nenhum teste** — é texto; o gate é o validador de spec

**Tests**: none
**Gate**: build

**Commit**: `docs(spec): pin the path resolution decisions in the feature spec`

---

### T14: A feature é re-verificada por um Verifier independente, com sensor

**What**: despachar um Verifier **independente** (≠ o autor de T1..T13) que (a) re-derive a cobertura da feature `graph-path-discovery` ancorada no spec emendado, com evidência `file:line` para as ACs; (b) rode o **sensor de discriminação** em worktree isolado re-injetando o M7 (`internal/graph/path.go:166`), a mutação composta do FIX-7 (helper devolvendo `MaxDepth = 9` com a CLI mantendo o literal `6`) e as 6 mutações já mortas (M1–M6, nos alvos `internal/graph/path.go`, `cmd/mem/path.go` e `internal/mcp/server.go`); (c) escreva o relatório `.specs/features/graph-path-discovery/validation.md` com o veredito novo.
**Where**: `.specs/features/graph-path-discovery/validation.md`
**Depends on**: T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13
**Reuses**: o sensor anterior (7 alvos, worktree isolado, sem `git stash`) · `internal/pgtest` (o sensor precisa do Postgres para T3 e T12) · o fallback standalone do `tlc-spec-driven` para a análise ancorada na spec
**Requirement**: GPR-43, GPR-44, GPR-45, GPR-46, GPR-47
**Gap**: FIX-1, FIX-7 e o ciclo de re-verificação

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O Verifier é um agente **distinto** do autor de T1..T13 (autor ≠ verificador)
- [x] O sensor re-injeta o M7 e **derruba** a suíte (GPR-43)
- [x] O sensor re-injeta a mutação composta do FIX-7 e **derruba** a suíte (GPR-44)
- [x] As 6 mutações do sensor original (M1–M6) constam no relatório: **6/6 mortas** (GPR-45)
- [x] As ACs da feature `graph-path-discovery` têm evidência `file:line` no relatório
- [x] O veredito do relatório é **PASS** e `validate_state.py graph-path-discovery` sai **exit 0**
- [x] A árvore real está intacta no fim (baseline e final idênticos, `git status --porcelain` limpo) — **sem `git stash`**
- [x] O relatório registra a dependência que **resta**: o `mem path` no vault real só funciona depois da ISSUE-047

**Tests**: integration
**Gate**: full

**Commit**: `docs(specs): re-verificacao independente da feature graph-path-discovery`

---

## Phase Execution Map

```
Phase 1 → Phase 2 → Phase 3 → Phase 4

Phase 1:  T1
          T1 ──→ T2
          T1 ──→ T3
Phase 2:  T2 ──→ T4
          T3 ──→ T4
          T2 ──→ T5
          T3 ──→ T5
          T5 ──→ T6
          T4 ──→ T7
Phase 3:  T7 ──→ T8
          T5 ──→ T9
          T6 ──→ T9
          T10 ──→ T11
          T3 ──→ T12
Phase 4:  T2 ──→ T14
          T3 ──→ T14
          T4 ──→ T14
          T5 ──→ T14
          T6 ──→ T14
          T7 ──→ T14
          T8 ──→ T14
          T9 ──→ T14
          T10 ──→ T14
          T11 ──→ T14
          T12 ──→ T14
          T13 ──→ T14
```

Execution is strictly sequential - there is no intra-phase parallelism. A single agent (or batch worker) works one task at a time, in order.

**Contagem de tarefas**: 14 em 4 fases — cabem em **2 lotes** (≤ ~7 tasks por worker, sempre em fronteira de fase): lote 1 = fases 1–2 (T1–T7, 7 tasks), lote 2 = fases 3–4 (T8–T14, 7 tasks). A execução oferece sub-agentes de lote ao dono; a T14 despacha o **Verifier** independente, que nunca é opcional.

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: contrato da resolução e da ambiguidade | 1 arquivo novo + 1 teste | ✅ Granular |
| T2: resolver do SQLite | 1 função + 1 teste | ✅ Granular |
| T3: resolver do Postgres | 1 função + 1 teste | ✅ Granular |
| T4: ambiguidade na CLI | 1 arquivo + 1 teste | ✅ Granular |
| T5: ambiguidade na tool MCP | 1 handler + 1 teste | ✅ Granular |
| T6: headings e InvalidParams | 1 arquivo de teste | ✅ Granular |
| T7: saída da CLI | 1 arquivo de teste | ✅ Granular |
| T8: defaults da CLI | 1 arquivo + 1 teste | ✅ Granular |
| T9: defaults da tool MCP | 1 arquivo + 1 teste | ✅ Granular |
| T10: `Summary` do nó idêntico | 1 arquivo de teste | ✅ Granular |
| T11: detalhes da aresta | 1 arquivo de teste | ✅ Granular |
| T12: isolamento por repositório | 1 arquivo de teste | ✅ Granular |
| T13: emenda do spec da feature | 1 documento | ✅ Granular |
| T14: re-verificação independente | 1 relatório de verificação | ✅ Granular |

---

## Diagram-Definition Cross-Check

| Task | Depends On (task body) | Diagram Shows | Status |
| ---- | ---------------------- | ------------- | ------ |
| T1 | None | (início da fase 1) | ✅ Match |
| T2 | T1 | T1 ──→ T2 | ✅ Match |
| T3 | T1 | T1 ──→ T3 | ✅ Match |
| T4 | T2, T3 | T2 ──→ T4 · T3 ──→ T4 | ✅ Match |
| T5 | T2, T3 | T2 ──→ T5 · T3 ──→ T5 | ✅ Match |
| T6 | T5 | T5 ──→ T6 | ✅ Match |
| T7 | T4 | T4 ──→ T7 | ✅ Match |
| T8 | T7 | T7 ──→ T8 | ✅ Match |
| T9 | T5, T6 | T5 ──→ T9 · T6 ──→ T9 | ✅ Match |
| T10 | None | (início da fase 3) | ✅ Match |
| T11 | T10 | T10 ──→ T11 | ✅ Match |
| T12 | T3 | T3 ──→ T12 | ✅ Match |
| T13 | None | (início da fase 4) | ✅ Match |
| T14 | T2, T3, T4, T5, T6, T7, T8, T9, T10, T11, T12, T13 | T2..T13 ──→ T14 (12 setas na fase 4) | ✅ Match |

Nenhuma dependência aponta para uma fase posterior: T2/T3 (fase 2) dependem de T1 (fase 1); T4/T5 (fase 3) dependem de T2/T3 (fase 2); T6/T7 (fase 3) dependem de T5/T4 (mesma fase); T8/T9/T11/T12 (fase 4) dependem de T7/T5/T6/T10 (fases 3 e 2) e de T10 (mesma fase); T14 (fase 5) depende de T2..T13 (fases anteriores).

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | Contrato de resolução e relatório de ambiguidade | unit | unit | ✅ OK |
| T2 | Resolução de nó no SQLite | integration | integration | ✅ OK |
| T3 | Resolução de nó no PostgreSQL | integration | integration | ✅ OK |
| T4 | Superfície da CLI (relatório de ambiguidade) | integration | integration | ✅ OK |
| T5 | Superfície da tool MCP (handler) | unit | unit | ✅ OK |
| T6 | Superfície da tool MCP (asserção) | unit | unit | ✅ OK |
| T7 | Superfície da CLI (not-found e nó idêntico) | integration | integration | ✅ OK |
| T8 | Defaults de caminho na CLI | integration | integration | ✅ OK |
| T9 | Defaults de caminho na tool MCP | unit | unit | ✅ OK |
| T10 | Motor — ramo do nó idêntico | unit | unit | ✅ OK |
| T11 | Motor — detalhes da aresta | unit | unit | ✅ OK |
| T12 | Caminho no PostgreSQL | integration | integration | ✅ OK |
| T13 | Texto do spec da feature | none (documento) | none | ✅ OK |
| T14 | Verificação independente + sensor | integration | integration | ✅ OK |

---

## Nota de escopo para o Execute

- **O motor não é tocado**: `internal/graph/path.go` (Dijkstra, pruning por dominância, inversão epistêmica) fica intacto — o Verifier confirmou que os três estão corretos. T10 e T11 são **só** asserções.
- **A resolução nunca escolhe em silêncio**: o desempate é **determinístico** (T2, T3) e a ambiguidade é **informada** — a CLI imprime o relatório e segue (T4), a tool MCP devolve `InvalidParams` com os candidatos (T5). **Não** criar flag nova para desambiguar e **não** usar `alias:` (o banco não guarda alias).
- **FIX-2 (ISSUE-047) não entra aqui**: a correção da identidade do nó é da spec `fix-graph-node-identity`. Esta feature continua **não usável no vault real** até aquela spec e o reindex — o T13 escreve essa dependência no spec da feature.
- **Escopo da emenda**: o T13 é a **única** task que toca `.specs/features/graph-path-discovery/spec.md`. Nenhuma outra task emenda esse arquivo, e a spec irmã `fix-graph-path-discovery-gaps` foi descartada — não há dois planos disputando o mesmo arquivo.
