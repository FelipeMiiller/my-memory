# Tasks: fix-graph-node-identity

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-16
**Issue**: ISSUE-047 (`.specs/ISSUES.md`) · é o **FIX-2** da validação da `graph-path-discovery` (ADR-027)

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Design**: inline nesta spec (o `documents.id` vira a identidade canônica do nó; o
`title:` é o rótulo; path e `alias:` são aliases do mesmo nó; a resolução do alvo do
wikilink acontece **na gravação**; o alvo não resolvido permanece o texto do wikilink;
**o motor de pathfinding não é tocado**)
**Status**: Draft

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec. Guidelines found: `AGENTS.md` (regra 4 — testes obrigatórios; regra 8 — quality gate), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`, `.github/workflows/ci.yml` (comandos de teste reais do projeto).

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Indexação em lote (SQLite) | unit | 1:1 com GNI-01 a GNI-04: destino resolvido vira o `documents.id`, destino não resolvido mantém o texto do wikilink, a origem continua o id do documento | `cmd/mem/index_identity_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestIndexIdentity ./cmd/mem/...` |
| Indexação em lote (Postgres) | integration (PostgreSQL real) | o mesmo contrato no engine de produção (GNI-01 a GNI-04) | `cmd/mem/index_identity_pg_test.go` | `MY_MEMORY_PG_TEST=1 go -C D:\repository\my-memory test -count=1 -run TestPostgresIndexIdentity ./cmd/mem/...` |
| Indexação cirúrgica (watcher) | integration | 1:1 com GNI-01 a GNI-04 nos dois engines do watcher — o watcher não pode reintroduzir a fenda | `internal/watcher/index_identity_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestIndexIdentity ./internal/watcher/...` |
| Nó canônico e rótulo (SQLite) | unit | 1:1 com GNI-05 e GNI-06: um nó por documento, `name` = `title`, a FK de `graph_edges` satisfeita, o nó de código preservado | `internal/db/graph_identity_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestNodeIdentity ./internal/db/...` |
| Nó canônico e rótulo (Postgres) | integration (PostgreSQL real) | o mesmo contrato no engine de produção (GNI-05 e GNI-06) | `internal/store/graph_identity_test.go` | `MY_MEMORY_PG_TEST=1 go -C D:\repository\my-memory test -count=1 -run TestPostgresNodeIdentity ./internal/store/...` |
| Resolvedor canônico (SQLite) | unit | 1:1 com GNI-07 a GNI-09: path, `title:` e `alias:` do mesmo documento devolvem o mesmo id; sem raiz do vault o alias devolve o erro; o título nunca devolve o id de um nó de estilo título | `internal/db/graph_identity_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestResolveCanonical ./internal/db/...` |
| Resolvedor canônico (Postgres) | integration (PostgreSQL real) | o mesmo contrato no engine de produção (GNI-07 a GNI-09) | `internal/store/graph_identity_test.go` | `MY_MEMORY_PG_TEST=1 go -C D:\repository\my-memory test -count=1 -run TestPostgresResolveCanonical ./internal/store/...` |
| Doctor: nós legados (SQLite) | unit | 1:1 com GNI-13, GNI-14 e GNI-16: a contagem aparece no relatório, o `--fix` remove só os órfãos, o nó de código fica, aresta de documento não é dead link | `internal/db/graph_identity_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestDoctorLegacyNodes ./internal/db/...` |
| Doctor: nós legados (Postgres) | integration (PostgreSQL real) | o mesmo contrato no engine de produção (GNI-13, GNI-14 e GNI-16) | `internal/store/doctor_identity_test.go` | `MY_MEMORY_PG_TEST=1 go -C D:\repository\my-memory test -count=1 -run TestPostgresDoctorLegacyNodes ./internal/store/...` |
| Superfície CLI (ponta a ponta) | e2e | 1:1 com GNI-10 a GNI-13: `mem path` acha a rota de 1 salto, `mem impact` e `mem inspect` devolvem o mesmo id canônico, a ponte `references_code` mantém o `source_id`, o doctor não acusa dead link de documento | `cmd/mem/path_identity_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestPathIdentity ./cmd/mem/...` |
| Contrato (ADR-066) | none | build gate only — o contrato é conferido por leitura cruzada (ADR ↔ código ↔ docs) | `docs/adr/066-a-identidade-canonica-do-no-no-grafo.md` | build gate only |
| Procedimento de reparo | none | o gate é a **query de medição** (count de arestas divididas) + o build | `docs/CLI_GUIDE.md` | build gate only |
| Medição no vault real | integration (E2E no Postgres de produção) | 1:1 com GNI-15, GNI-16 e GNI-17: a query de verificação, a contagem de nós e o `mem path` no vault reindexado | a medição de `repo_central` (sem arquivo no repo) | `mem path "CONTRIBUTING.md" "README.md" --undirected --storage=postgres --repo repo_e4c8f3b1a2d5` |
| Alinhamento do código | format | o `gofmt` não aponta nenhum arquivo `.go` tocado (medido em cópia LF) | os arquivos `.go` tocados | `gofmt -l` |

## Gate Check Commands

> Generated from codebase — confirm before Execute.

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após cada task de código | `go -C D:\repository\my-memory build ./...` e `go -C D:\repository\my-memory vet ./...` |
| Quick | Após as tasks de indexação (T2 e T3) | `go -C D:\repository\my-memory test -count=1 -run TestIndexIdentity ./cmd/mem/... ./internal/watcher/...` |
| Quick | Após as tasks de nó e resolvedor (T4 a T7) | `go -C D:\repository\my-memory test -count=1 -run TestNodeIdentity ./internal/db/... ./internal/store/...` e `go -C D:\repository\my-memory test -count=1 -run TestResolveCanonical ./internal/db/... ./internal/store/...` |
| Full | Gate final (T12) | `go -C D:\repository\my-memory test -count=1 ./...` (com `MY_MEMORY_PG_TEST=1` e `MY_MEMORY_PG_URL` preservada — 27/27 pacotes) |
| Full | Gate final taggeado | `go -C D:\repository\my-memory test -tags "sqlite_fts5 onnx" -count=1 ./...` |
| Medido | T12 | `mem index --storage=postgres --repo repo_central` + `mem doctor --fix --storage=postgres --repo repo_central` + a query de verificação da T11 |
| Format | Antes de declarar pronto | `gofmt -l` nos `.go` tocados, medido em **cópia LF** (o working tree local mistura CRLF — a CI roda em Linux) |
| Spec | Antes de entregar | `python3 .agents/skills/tlc-spec-driven/scripts/validate_spec.py fix-graph-node-identity --root D:\repository\my-memory` e `validate_tasks.py fix-graph-node-identity --root D:\repository\my-memory` |
| State | Antes de entregar | `python3 .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-graph-node-identity --root D:\repository\my-memory` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: A decisão de arquitetura e a gravação em lote

```
T1 -> T2
```

### Phase 2: A gravação completa e o nó canônico

```
T2 -> T3 -> T4 -> T5
```

### Phase 3: O resolvedor canônico

```
T5 -> T6 -> T7
```

### Phase 4: A limpeza do legado

```
T7 -> T8 -> T9
```

### Phase 5: A prova e o reparo do vault

```
T9 -> T10 -> T11 -> T12
```

---

## Task Breakdown

### T1: A ADR da identidade canônica do nó

**What**: registrar a decisão de arquitetura — o `documents.id` (id de path) é a
**identidade canônica** do nó no grafo, o `title:` é o **rótulo** do nó, e path e
`alias:` são **aliases do mesmo nó**; a resolução do alvo do wikilink acontece **na
gravação** da aresta; o alvo não resolvido permanece o texto do wikilink; a correção
exige **reindex** do vault para valer no dado já gravado.
**Where**: `docs/adr/066-a-identidade-canonica-do-no-no-grafo.md`
**Depends on**: None
**Reuses**: o formato MADR das ADRs do repo (`docs/adr/063-...md`, `docs/adr/065-...md`)
· a ADR-027 (a rota) · a ADR-055 (`documents.id` como referência estável, título como
rótulo) · a ADR-063 (o doctor) · a ADR-013 (higiene de grafo)
**Requirement**: GNI-18
**Gap**: FIX-2

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] A ADR decide que o `documents.id` é a identidade canônica do nó no grafo
- [x] A ADR diz que o `title:` é o rótulo e que path e `alias:` são aliases do mesmo nó
- [x] A ADR diz que a resolução acontece **na gravação** da aresta, não na leitura
- [x] A ADR diz que o alvo **não resolvido** permanece o texto do wikilink (o doctor
      continua enxergando o link rompido)
- [x] A ADR registra que a correção exige **reindex** do vault para valer no dado já
      gravado
- [x] A ADR cita a ISSUE-047, o FIX-2 da validação da `graph-path-discovery` e a ADR-055
- [x] A ADR está listada em `docs/adr/README.md` com o status real
- [x] O gate de build passa: `go -C D:\repository\my-memory build ./...`

**Tests**: none (camada de contrato — o gate é o build + a leitura cruzada)
**Gate**: quick

**Commit**: `docs(adr): a identidade canonica do no no grafo e o id de documento`

---

### T2: A indexação em lote grava o id de documento como destino

**What**: nos dois indexadores em lote, o **destino** da aresta passa a ser o
`documents.id` do documento resolvido (e não o texto do wikilink). A origem já é o id
de path (`docID`) — só o destino estava do outro lado da fenda.
**Where**: `cmd/mem/main.go`
**Depends on**: T1
**Reuses**: o mapa título→id já disponível no ponto da gravação (`titleToID`,
`cmd/mem/main.go:1928-1930`) · `parser.ResolveTagConnections`
(`internal/parser/fuzzy.go:34`) · a semântica de título/base do
`ResolveNodeCanonicalID` (`internal/db/graph.go:594-609`)
**Requirement**: GNI-01, GNI-02, GNI-03, GNI-04
**Gap**: FIX-2

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O indexador Postgres (`cmd/mem/main.go:1941`) grava o id do documento como destino
      quando o wikilink resolve
- [x] O indexador SQLite (`cmd/mem/main.go:2184`) grava o **mesmo** destino
- [x] Um wikilink que não resolve para documento indexado mantém o texto do wikilink
      como destino
- [x] A origem gravada continua sendo o `documents.id` do documento indexado
- [x] A resolução respeita o **repositório** — o título procurado é o do mesmo vault
- [x] As arestas `references_code` (ADR-061) não são tocadas, e a `tagged_as` que casa
      com um documento passa a apontar para o id dele
- [x] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestIndexIdentity ./cmd/mem/...`

**Tests**: integration
**Gate**: quick

**Commit**: `fix(index): a aresta de wikilink aponta para o id do documento`

---

### T3: A indexação cirúrgica grava o mesmo id

**What**: o indexador cirúrgico (watcher) grava o `documents.id` do destino nos dois
motores — sem isso, o primeiro arquivo que o watcher tocar reintroduz a fenda.
**Where**: `internal/watcher/indexer.go`
**Depends on**: T2
**Reuses**: `db.ListDocumentIDTitles` (`internal/watcher/indexer.go:108`) ·
`s.ListDocumentIDTitles` (`:258`) · o mesmo mapa de resolução da T2
**Requirement**: GNI-01, GNI-02, GNI-03, GNI-04
**Gap**: FIX-2

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] `internal/watcher/indexer.go:119` (SQLite) grava o id do documento como destino
- [x] `internal/watcher/indexer.go:267` (Postgres) grava o **mesmo** destino
- [x] Um wikilink que não resolve mantém o texto do wikilink como destino
- [x] A aresta `tagged_as` que casa com um documento aponta para o id dele, e a tag sem
      casa continua removida (ADR-041)
- [x] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestIndexIdentity ./internal/watcher/...`

**Tests**: integration
**Gate**: quick

**Commit**: `fix(watcher): a aresta de wikilink aponta para o id do documento`

---

### T4: O nó do documento no SQLite carrega o título como rótulo

**What**: o nó criado para um documento passa a ter `id` = `documents.id` e `name` =
`documents.title` — o título vira rótulo, o id de path vira a chave e o alias.
**Where**: `internal/db/store.go`
**Depends on**: T3
**Reuses**: `InsertEdgeWithProps` (`internal/db/store.go:388-408`) · a FK
`graph_edges.target_id → graph_nodes.id` do schema (`internal/db/schema.go:53-57`)
**Requirement**: GNI-05, GNI-06
**Gap**: FIX-2

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O nó de um documento indexado tem `id` = `documents.id` e `name` = `documents.title`
- [x] Um documento não ocupa dois nós (nenhum nó extra com o id = título do documento)
- [x] O nó de destino não resolvido (texto do wikilink) continua sendo criado, com o
      texto como nome — a FK continua satisfeita e o doctor continua enxergando o link
- [x] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestNodeIdentity ./internal/db/...`

**Tests**: unit
**Gate**: quick

**Commit**: `fix(graph): o no do documento carrega o titulo como rotulo`

---

### T5: O nó do documento no Postgres carrega o título como rótulo

**What**: o mesmo contrato do nó canônico no engine de produção.
**Where**: `internal/store/postgres.go`
**Depends on**: T4
**Reuses**: `PostgresStore.InsertEdgeWithProps` (`internal/store/postgres.go:344-372`) ·
o escopo de repositório do nó (`(repository, id)`)
**Requirement**: GNI-05, GNI-06
**Gap**: FIX-2

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O nó de um documento indexado tem `id` = `documents.id` e `name` = `documents.title`
- [ ] Um documento não ocupa dois nós no Postgres
- [x] O nó de destino não resolvido continua sendo criado, com o texto como nome
- [x] O gate rápido passa: `MY_MEMORY_PG_TEST=1 go -C D:\repository\my-memory test -count=1 -run TestPostgresNodeIdentity ./internal/store/...`

**Tests**: integration
**Gate**: quick

**Commit**: `fix(graph): o no canonico no postgres carrega o titulo como rotulo`

---

### T6: O resolvedor canônico no SQLite aceita path, título e alias

**What**: o `ResolveNodeCanonicalID` do SQLite passa a tratar path, `title:` e `alias:`
como **aliases do mesmo nó** — devolvendo sempre o `documents.id` do documento, e nunca
o id de um nó de estilo título.
**Where**: `internal/db/graph.go`
**Depends on**: T5
**Reuses**: `ResolveNodeCanonicalID` (`internal/db/graph.go:582`) · a resolução de alvo
em disco do `internal/vaultfs` (spec `fix-doctor-unindexed-links`) para o `alias:` do
frontmatter
**Requirement**: GNI-07, GNI-08, GNI-09
**Gap**: FIX-2

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O path, o `title:` e o `alias:` do **mesmo** documento devolvem o mesmo
      `documents.id`
- [x] Uma consulta pelo título devolve o `documents.id` e **não** o id de um nó de
      estilo título
- [x] Sem raiz do vault, um alias devolve o erro "nó '...' não encontrado no grafo"
- [x] A resolução por título continua determinística quando há um único documento com
      esse título (o desempate entre homônimos é do FIX-3, da spec
      `fix-graph-path-discovery-gaps`)
- [x] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestResolveCanonical ./internal/db/...`

**Tests**: unit
**Gate**: quick

**Commit**: `fix(graph): o resolvedor canonico aceita path, titulo e alias do mesmo no`

---

### T7: O resolvedor canônico no Postgres aceita path, título e alias

**What**: o mesmo contrato do resolvedor no engine de produção.
**Where**: `internal/store/postgres.go`
**Depends on**: T6
**Reuses**: `PostgresStore.ResolveNodeCanonicalID` (`internal/store/postgres.go:1018`) ·
o filtro por repositório dos dois engines
**Requirement**: GNI-07, GNI-08, GNI-09
**Gap**: FIX-2

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O path, o `title:` e o `alias:` do mesmo documento devolvem o mesmo
      `documents.id`, com o filtro de repositório respeitado
- [x] Uma consulta pelo título devolve o `documents.id` e **não** o id de um nó de
      estilo título
- [x] Sem raiz do vault, um alias devolve o erro de nó não encontrado
- [x] O gate rápido passa: `MY_MEMORY_PG_TEST=1 go -C D:\repository\my-memory test -count=1 -run TestPostgresResolveCanonical ./internal/store/...`

**Tests**: integration
**Gate**: quick

**Commit**: `fix(graph): o resolvedor canonico do postgres aceita path, titulo e alias`

---

### T8: O doctor do SQLite conta e remove os nós legados

**What**: o relatório do doctor passa a expor a contagem de nós que **não** são
`documents.id` nem `code_symbol` (os 3.200 nós de id de título do cofre central), e a
rotina `--fix` os remove — são o outro lado da fenda: enquanto existirem, o passo 2 do
resolvedor continua devolvendo o id de título.
**Where**: `internal/db/graph.go`
**Depends on**: T7
**Reuses**: `FixHealthIssues` (`internal/db/graph.go:540-579`) · `deadLinkCandidates`
(`:372`) · a query de dead link que já conhece `document_tags` e `code_symbol`
**Requirement**: GNI-13, GNI-14, GNI-16
**Gap**: FIX-2

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O relatório expõe a contagem de nós que não são `documents.id` nem `code_symbol`
- [x] A rotina `--fix` remove exatamente esses nós **sem grau**, e nenhum outro
- [x] Um nó de código (`code_symbol`) nunca é removido
- [x] Uma aresta cujo destino é um documento indexado não é tratada como dead link
- [x] O gate passa: `go -C D:\repository\my-memory test -count=1 -run TestDoctorLegacyNodes ./internal/db/...`

**Tests**: unit
**Gate**: quick

**Commit**: `fix(doctor): o sqlite conta e remove os nos legados de titulo`

---

### T9: O doctor do Postgres conta e remove os nós legados

**What**: o mesmo contrato de higiene do grafo no engine de produção.
**Where**: `internal/store/doctor.go`
**Depends on**: T8
**Reuses**: `deadLinkCandidates` (`internal/store/doctor.go:95`) · `DiagnoseHealth` e
`FixHealthIssues` do Postgres · o filtro por `repository`
**Requirement**: GNI-13, GNI-14, GNI-16
**Gap**: FIX-2

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O relatório expõe a contagem de nós que não são `documents.id` nem `code_symbol`
- [x] A rotina `--fix` remove exatamente esses nós **sem grau**, escopada por
      repositório, e nenhum outro
- [x] Um nó de código (`code_symbol`) nunca é removido
- [x] Uma aresta cujo destino é um documento indexado não é tratada como dead link
- [x] O gate passa: `MY_MEMORY_PG_TEST=1 go -C D:\repository\my-memory test -count=1 -run TestPostgresDoctorLegacyNodes ./internal/store/...`

**Tests**: integration
**Gate**: quick

**Commit**: `fix(doctor): o postgres conta e remove os nos legados de titulo`

---

### T10: A regressão ponta a ponta da identidade

**What**: o teste que fecha o ciclo — um vault de teste com notas ligadas por wikilink é
indexado e o `mem path` entre elas **acha a rota de 1 salto**, com uma **única**
identidade de nó por documento; e nenhum consumidor (impact, inspect, ponte de código,
doctor) fica com a identidade antiga.
**Where**: `cmd/mem/path_identity_test.go`
**Depends on**: T9
**Reuses**: `db.InitDB` + o indexador já usados pelos testes de `cmd/mem` ·
`runPathCommand` (`cmd/mem/path.go:24`, com `io.Writer` injetável) · a captura de
stdout já existente no pacote
**Requirement**: GNI-10, GNI-11, GNI-12, GNI-13
**Gap**: FIX-2

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O teste indexa 2 notas ligadas por `[[wikilink]]` e o `mem path` entre elas reporta
      `✅ Encontrado` com **1** salto
- [ ] O teste prova que as duas notas ocupam **uma** identidade de nó cada e que a aresta
      gravada tem as duas pontas em id de documento
- [ ] O teste prova que `mem impact` e `mem inspect` devolvem o **mesmo** id canônico que
      o `mem path` para o mesmo documento
- [ ] O teste prova que a ponte `references_code` mantém `source_id` = `documents.id`
- [ ] O teste prova que o `mem doctor` não acusa as arestas de documento como dead link
- [ ] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestPathIdentity ./cmd/mem/...`

**Tests**: e2e
**Gate**: quick

**Commit**: `test(cli): o mem path acha a rota entre notas ligadas por wikilink`

---

### T11: O procedimento de reparo do vault indexado

**What**: documentar o reparo do dado já gravado — o comando de reindex, o
`mem doctor --fix` que remove os nós legados, a **query** que mede as arestas
atravessadas (antes e depois) e o que comparar no `mem doctor` (o Health Score pode
mudar, porque as arestas passam a apontar para documentos).
**Where**: `docs/CLI_GUIDE.md`
**Depends on**: T10
**Reuses**: a query de medição usada na verificação de 2026-11-15 (`graph_edges` ×
`graph_nodes` × `documents`) · a seção de comandos de manutenção da CLI
**Requirement**: GNI-15, GNI-16, GNI-17
**Gap**: FIX-2

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] A doc traz a **query** que conta as arestas com as duas pontas em id de documento
- [ ] A doc diz que o valor atual do vault `repo_central` é **0** de 40.243 (a medição
      de 2026-11-15)
- [ ] A doc traz o procedimento (reindex → `mem doctor --fix` → query) e avisa que ele
      reescreve o grafo
- [ ] A doc manda comparar o `mem doctor` (Health Score e links rompidos) antes e depois
- [ ] A doc deixa claro que a mudança de código **não** repara o vault sem reindex
- [ ] O gate de build passa: `go -C D:\repository\my-memory build ./...`

**Tests**: none (camada de contrato — o gate é o build + a query de medição)
**Gate**: quick

**Commit**: `docs(cli): o procedimento de reparo do grafo apos a unificacao de identidade`

---

### T12: O reparo do vault real e a medição

**What**: executar o reindex do cofre central (com o go-ahead explícito do dono — é
operação de dado no vault de produção), o `mem doctor --fix` e a medição; e registrar o
resultado. Sem esta etapa a correção não se prova: o código novo não conserta o dado já
gravado.
**Where**: `mem index --storage=postgres --repo repo_central` (a medição; sem arquivo no repo)
**Depends on**: T11
**Reuses**: a query de medição da T11 · a medição de 2026-11-15 (40.243 arestas, 0
atravessadas, 3.200 nós de título) como baseline
**Requirement**: GNI-15, GNI-16, GNI-17
**Gap**: FIX-2

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O dono deu o go-ahead explícito para o reindex do cofre central
- [ ] A query de verificação devolve **0** arestas com as pontas divididas — o total de
      arestas com as duas pontas em id de documento é igual ao total de arestas cujo
      alvo resolve para um documento indexado (baseline: 0 de 40.243)
- [ ] A contagem de nós que não são `documents.id` nem `code_symbol` é **0** depois do
      `mem doctor --fix` (baseline: 3.200)
- [ ] O `mem doctor` do cofre central continua reportando as arestas classificáveis
      (125 fora do índice por decisão + 13 rompidas) — a correção não cega o doctor
- [ ] `mem path "CONTRIBUTING.md" "README.md" --undirected --storage=postgres --repo repo_e4c8f3b1a2d5`
      devolve `✅ Encontrado`
- [ ] O gate completo passa: `go -C D:\repository\my-memory test -count=1 ./...` e
      `go -C D:\repository\my-memory test -tags "sqlite_fts5 onnx" -count=1 ./...`
- [ ] A medição antes/depois fica registrada no `tasks.md`, no status desta task

**Tests**: integration
**Gate**: full

**Commit**: `docs(specs): a medicao do reparo do grafo no cofre central`

---

## Phase Execution Map

```
Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5

Phase 1:  T1 -> T2
Phase 2:  T2 -> T3 -> T4 -> T5
Phase 3:  T5 -> T6 -> T7
Phase 4:  T7 -> T8 -> T9
Phase 5:  T9 -> T10 -> T11 -> T12
```

Execution is strictly sequential - there is no intra-phase parallelism. A single agent (or batch worker) works one task at a time, in order.

**Contagem de tarefas**: 12 — cabem em **dois lotes** (≤ ~7 tasks por worker), então a
execução oferece sub-agente de lote: lote 1 = fases 1 a 3 (7 tasks), lote 2 = fases 4 e 5
(5 tasks). A seta `T1 -> T2` na fase 1 é a **passagem de fase** (T2 abre a fase 2) e o
grafo de dependências é uma cadeia linear.

---

## Diagram-Definition Cross-Check

| Task | Depends On (task body) | Diagram Shows | Status |
| ---- | ---------------------- | ------------- | ------ |
| T1 | None | (início da fase 1) | ✅ Match |
| T2 | T1 | T1 -> T2 | ✅ Match |
| T3 | T2 | T2 -> T3 | ✅ Match |
| T4 | T3 | T3 -> T4 | ✅ Match |
| T5 | T4 | T4 -> T5 | ✅ Match |
| T6 | T5 | T5 -> T6 | ✅ Match |
| T7 | T6 | T6 -> T7 | ✅ Match |
| T8 | T7 | T7 -> T8 | ✅ Match |
| T9 | T8 | T8 -> T9 | ✅ Match |
| T10 | T9 | T9 -> T10 | ✅ Match |
| T11 | T10 | T10 -> T11 | ✅ Match |
| T12 | T11 | T11 -> T12 | ✅ Match |

Nenhuma dependência aponta para uma fase posterior.

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | Contrato (ADR-066) | none | none | ✅ OK (camada de contrato — o gate é o build + a leitura cruzada) |
| T2 | Indexação em lote (SQLite + Postgres) | integration | integration | ✅ OK |
| T3 | Indexação cirúrgica (watcher) | integration | integration | ✅ OK |
| T4 | Nó canônico e rótulo (SQLite) | unit | unit | ✅ OK |
| T5 | Nó canônico e rótulo (Postgres) | integration | integration | ✅ OK |
| T6 | Resolvedor canônico (SQLite) | unit | unit | ✅ OK |
| T7 | Resolvedor canônico (Postgres) | integration | integration | ✅ OK |
| T8 | Doctor: nós legados (SQLite) | unit | unit | ✅ OK |
| T9 | Doctor: nós legados (Postgres) | integration | integration | ✅ OK |
| T10 | Superfície CLI (ponta a ponta) | e2e | e2e | ✅ OK |
| T11 | Procedimento de reparo | none | none | ✅ OK (camada de contrato — o gate é o build + a query de medição) |
| T12 | Medição no vault real | integration | integration | ✅ OK |

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: ADR da identidade | 1 documento | ✅ Granular |
| T2: gravação em lote | 2 pontos do mesmo arquivo (Postgres + SQLite) | ✅ Granular (mudança idêntica nos dois indexadores) |
| T3: gravação cirúrgica | 2 pontos do mesmo arquivo | ✅ Granular |
| T4: nó canônico SQLite | 1 função | ✅ Granular |
| T5: nó canônico Postgres | 1 função | ✅ Granular |
| T6: resolvedor SQLite | 1 função | ✅ Granular |
| T7: resolvedor Postgres | 1 função | ✅ Granular |
| T8: doctor SQLite | 1 arquivo (relatório + prune) | ✅ Granular |
| T9: doctor Postgres | 1 arquivo (relatório + prune) | ✅ Granular |
| T10: regressão ponta a ponta | 1 arquivo de teste | ✅ Granular |
| T11: procedimento de reparo | 1 documento | ✅ Granular |
| T12: reparo do vault real | 1 operação + a medição | ✅ Granular |