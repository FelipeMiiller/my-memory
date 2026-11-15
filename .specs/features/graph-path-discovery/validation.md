# graph-path-discovery Validation

**Date**: 2026-11-15
**Spec**: `.specs/features/graph-path-discovery/spec.md`
**ADR**: `docs/adr/027-descoberta-de-rotas-e-caminho-minimo-no-grafo.md`
**Diff range**: `22de8de..9d3e0ee` (feature commit `9d3e0ee feat(graph,staleness): implement path discovery (ADR-027) and staleness banners (ADR-028)`)
**Verified at**: `e498a34` (branch `develop`), worktree clean
**Verifier**: independent sub-agent (author != verifier)

**Overall**: ❌ Not Ready — **FAIL**

**Verdict**: FAIL — the pathfinding engine is correct and every criterion it controls holds, but the feature's primary acceptance criterion (P1.1) does **not** hold end-to-end in the product, and the test suite does not guard one of the branches the spec asks for.

> This report **replaces** the previous `validation.md`, which was a *plan* ("# Validation Plan: graph-path-discovery") that had never been executed. Its claims about test names and coverage did not survive contact with the repository (see "Plan vs. reality" below).

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1 Motor de pathfinding (`internal/graph/`) | ✅ Done | `internal/graph/path.go` (366 lines). Dijkstra over `container/heap`; the depth-aware dominance pruning is sound (verified by trace + M1–M4). |
| T2 Persistência SQLite e PostgreSQL | ⚠️ Partial | Both `FindPath` implementations exist and work, but the PostgreSQL path is covered only by the trivial self-path case (see gap **G5**). |
| T3 Subcomando CLI `mem path` | ✅ Done | `cmd/mem/path.go`; verified live against a real vault (route found, ASCII rendering correct). |
| T4 Ferramenta MCP `memory_find_path` | ✅ Done | Registered in stdio **and** HTTP/SSE (same `runMCPServer`), formal JSON Schema present. |
| T5 ADR-027 e documentação | ✅ Done | ADR + 4 index/manual updates + `STATE.md` all present. |

---

## Spec-anchored Acceptance Criteria

Legend: ✅ PASS · ❌ GAP · ⚠️ Spec-precision gap / partial coverage

### P1: Cálculo de Menor Rota Epistêmica no Grafo ⭐ MVP

| Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion expression | Result |
| ------------------------- | -------------------- | ---------------------------------- | ------ |
| **P1.1** WHEN there is a path inside `max_depth` THEN `found: true` with an ordered node list and detailed edges (relation, epistemic status, weight) | `found=true`; ordered nodes; edges carry relation + epistemic status + weight | `internal/graph/path_test.go:85` `if !res.Found`; `:88` `res.Hops != 2`; `:94-99` exact node order; `:100` `len(res.Edges) != 2`; `:103`/`:106` `res.Edges[i].From/To/Direction` · `cmd/mem/path_test.go:93` `!strings.Contains(outStr,"implements") \|\| !strings.Contains(outStr,"depends_on")` | ❌ **GAP** — the *engine* passes, but on the owner's real vault the criterion **fails end-to-end** (gap **G1**), and `EpistemicStatus`/`Weight` are never asserted (gap **G4**) |
| **P1.2** WHEN there is no path THEN `found: false` with an elucidating message | `found=false` + message | `internal/graph/path_test.go:66` `if res.Found`; `:69` `len(res.Nodes) != 0 \|\| len(res.Edges) != 0` | ⚠️ Partial — `found:false` covered; the message (`internal/graph/path.go:323` `Summary`) is never asserted (gap **G6**) |
| **P1.3** WHEN source == target THEN a 0-hop path with cost 0 | `hops=0`, `total_cost=0` | `internal/graph/path_test.go:18` `!res.Found`; `:21` `res.Hops != 0`; `:24` `res.TotalCost != 0.0`; `:27` `len(res.Nodes) != 1 \|\| res.Nodes[0] != "auth.md"`; `:30` `len(res.Edges) != 0` | ⚠️ Outcome holds, but **not discriminating** — mutant **M7** survives (gap **G3**) |

### P2: Travessia Não-Direcionada e Modos de Custo

| Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion expression | Result |
| ------------------------- | -------------------- | ---------------------------------- | ------ |
| **P2.1** WHEN `--undirected` THEN edges may be traversed in both directions | directed `A→C` not found; undirected `A→C` found | `internal/graph/path_test.go:182` `if resDir.Found` (directed must fail); `:192` `!resUndir.Found`; `:195` `resUndir.Hops != 2` · `cmd/mem/path_test.go:153` `if dirRes.Found`; `:167` `!undirRes.Found`; `:170` `undirRes.Hops != 2` | ✅ PASS (mutants **M2** and **M6** killed) |
| **P2.2** WHEN a route is returned THEN each edge states its orientation | `forward` / `reverse` per edge | `internal/graph/path_test.go:201` `resUndir.Edges[0].Direction != "forward"`; `:204` `resUndir.Edges[1].Direction != "reverse"` | ✅ PASS (domain layer; the CLI/MCP reverse arrow rendering is exercised but not asserted) |
| **P2.3** WHEN `--mode hops` THEN cost is unit per edge, not epistemic | 1 hop chosen; total cost `1.0` | `internal/graph/path_test.go:157` `resHops.Hops != 1`; `:160` `resHops.TotalCost != 1.0` · `internal/db/graph_test.go:300` `resHops.Hops != 1` · `cmd/mem/path_test.go:173` `undirRes.CostMode != graph.CostModeHops` | ✅ PASS (mutants **M3** and **M4** killed) |

### P3: Integração com Servidor MCP

| Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion expression | Result |
| ------------------------- | -------------------- | ---------------------------------- | ------ |
| **P3.1** WHEN the catalogue is listed THEN `memory_find_path` is registered | tool present in the catalogue + schema properties | `internal/mcp/path_handlers_test.go:161` `tools := server.GetTools()`; `:165` `tool.Name == "memory_find_path"`; `:172-186` `props["source"/"target"/"max_depth"/"directed"/"mode"]`; `:191` `if !found` | ✅ PASS — the same registry feeds `tools/list` (`internal/mcp/tools.go:802` returns `s.tools`); mutant **M5** killed |
| **P3.2** WHEN the tool returns THEN Markdown with inline visual route + details table | named visual block + named table | `internal/mcp/path_handlers_test.go:86-97` | ⚠️ Partial — see **G7** |

### Goals (G1–G5)

| Goal | Spec-defined outcome | Evidence | Result |
| ---- | -------------------- | -------- | ------ |
| **G1** Pathfinding engine in `internal/graph/path.go` | Dijkstra + `container/heap`; `epistemic` c=1/w and `hops` c=1; directed/undirected; `max_depth`; cycle termination | `internal/graph/path.go:60-92` (heap); `:109-115` (cost modes); `:196-216` (forward/reverse adjacency); `:139-142`+`:259-261`+`:270-272` (depth cut); `:279-289` (cycle guard) | ✅ PASS |
| **G2** Persistence resolution | SQLite `FindPath`, Postgres `FindPath`, `Store` interface, `ResolveNodeCanonicalID` | `internal/db/graph.go` `FindPath`; `internal/store/postgres.go` `FindPath`; `internal/store/store.go:268` | ✅ PASS (coverage gap **G5**) |
| **G3** CLI `mem path <source> <target>` | `--directed`, `--max-depth`, `--mode`, `--json` + ASCII route renderer | `cmd/mem/path.go:19-174`; flags `:28-37`; dispatch `cmd/mem/main.go:742`; help `cmd/mem/main.go:891` | ✅ PASS (live smoke test) |
| **G4** MCP tool registered in stdio **and** HTTP/SSE with formal JSON Schema | tool in both transports; JSON Schema | `internal/mcp/tools.go:486-520` (schema, `required: [source,target]`); `internal/mcp/server.go:111` (default registration); `cmd/mem/main.go:3229` (Postgres) and `:3433` (SQLite) `SetPathHandler`; a single `runMCPServer` serves stdio and wraps HTTP/SSE at `cmd/mem/main.go:3480-3482` | ✅ PASS |
| **G5** ADR-027 + indexes/manuals | ADR in MADR + catalog + manuals + state | `docs/adr/027-descoberta-de-rotas-e-caminho-minimo-no-grafo.md`; `docs/adr/README.md:37`; `docs/README.md:54`; `docs/CLI_GUIDE.md:656`; `docs/AGENT_INTEGRATION_GUIDE.md:260`+`:306`; `.specs/STATE.md:125` | ✅ PASS |

**Spec-anchored check**: **5/8 ACs fully covered**, **3 partial** (P1.1, P1.2, P3.2), **0 fully unanchored**. Plus **7 spec-precision gaps** listed below.

### Spec-precision gaps (the spec does not define the outcome)

| # | Gap | Where |
| - | --- | ----- |
| S1 | P3's story promises "Markdown **e JSON**", but AC P3.2 only requires Markdown, and the tool returns text-only — `NewTextResult` (`internal/mcp/handlers.go:40`) sets no `structuredContent`. Already tracked as **ISSUE-022** in `.specs/STATE.md:133`. | spec.md:71 |
| S2 | The spec never says how an edge whose relation *is* `EXTRACTED`/`INFERRED`/`TAG` is rendered; the implementation rewrites it to `links_to` (`internal/graph/path.go:191-194`). | spec.md:59 |
| S3 | G3 lists `--directed` while P2/ADR-027 use `--undirected`. Both flags exist (`cmd/mem/path.go:30-31`) and `--undirected` silently wins (`cmd/mem/path.go:58-61`); the spec defines no precedence. | spec.md:30 vs :66 |
| S4 | Behavior for an invalid `--mode` value is unspecified; the implementation silently falls back to `epistemic` (`cmd/mem/path.go:63-66`, `internal/mcp/path_handlers.go:59-65`). | spec.md:68 |
| S5 | Behavior for a non-positive `--max-depth` is unspecified; CLI resets to 6 (`cmd/mem/path.go:54-56`), MCP keeps 6 (`internal/mcp/path_handlers.go:44-49`). | spec.md:30 |
| S6 | The spec never states the expected `relation`/`epistemic_status` strings that the algorithm emits. | spec.md:59 |
| S7 | `PathResult.Summary` — the "mensagem elucidativa" of P1.2 — has no specified text. | spec.md:60 |

---

## Discrimination Sensor

Executed in an isolated scratch worktree (`git worktree add D:\.minimax\scratch\subagents\verifier-gpd\wt HEAD`, detached at `e498a34`). The real tree was never mutated — `git status --porcelain` was empty before and after. **`git stash` was never used.**

**Sensor depth**: lightweight, 7 behavior-level mutations (the brief asked for 3–6; a 7th branch-discrimination probe was added because the brief flags non-discriminating paths as an important finding).

| Mutation | File:line | Description | Killed? |
| -------- | --------- | ----------- | ------- |
| **M1** | `internal/graph/path.go:139` | `maxDepth := opts.MaxDepth` → `maxDepth := opts.MaxDepth * 1000` — the depth cut stops being enforced | ✅ Killed — `--- FAIL: TestFindPath_MaxDepthCutoff` |
| **M2** | `internal/graph/path.go:207` | `if !opts.Directed {` → `if false {` — undirected mode stops inverting edges | ✅ Killed — `--- FAIL: TestFindPath_DirectedVsUndirected` |
| **M3** | `internal/graph/path.go:114` | `cost := 1.0 / weight` → `cost := 1.0` — epistemic cost stops being weighted | ✅ Killed — `--- FAIL: TestFindPath_EpistemicCostTieBreaker` |
| **M4** | `internal/graph/path.go:110` | `return 1.0, weight, status` → `return 1.0 / weight, weight, status` — hops mode stops using unit cost | ✅ Killed — `--- FAIL: TestFindPath_EpistemicCostTieBreaker` |
| **M5** | `internal/mcp/server.go:111` | `s.RegisterTool(ToolMemoryFindPath, NewMemoryFindPathHandler(nil))` removed — tool drops out of the `tools/list` catalogue | ✅ Killed — `--- FAIL: TestMemoryFindPath_ServerRegistration` |
| **M6** | `cmd/mem/path.go:60` | `isDirected = false` → `isDirected = true` — the CLI `--undirected` flag is ignored | ✅ Killed — `--- FAIL: TestPathCLI_UndirectedAndHops` (`path_test.go:168: esperava found=true no modo undirected`) |
| **M7** | `internal/graph/path.go:166` | `if cleanSource == cleanTarget {` → `if false {` — the dedicated identical-node branch is disabled | ❌ **SURVIVED** — target test rc=0 **and** the whole `internal/graph` package rc=0 |

> M6 was first attempted as `if *undirected {` → `if false {`, which produced a **compile error** (`declared and not used`) rather than a behavioral failure. That was rejected as an invalid kill and re-run as the behavioral mutation above.

**Result**: **6/7 killed** — the suite discriminates the depth cut, the undirected inversion, the epistemic weighting, the hops unit cost, the MCP catalogue registration and the CLI flag plumbing. **1 survivor (M7)** → fix task **FIX-1**.

**Why M7 survives**: the generic Dijkstra path already returns `Found=true, Hops=0, TotalCost=0, Nodes=[source], Edges=[]` when source == target (the source is popped at cost 0 and matches the target). The dedicated branch at `internal/graph/path.go:166-179` therefore adds no observable difference **except** its `Summary` string ("Nó de origem e destino são idênticos"), which no test asserts. The branch is effectively redundant *and* unguarded.

---

## Code Quality

| Principle | Status |
| ---------------- | ------ |
| No features beyond what was asked | ✅ — the commit also ships ADR-028 (staleness), but that is a separate spec with its own artifacts |
| No abstractions for single-use code | ⚠️ — `graph.DefaultPathOptions()` (`internal/graph/path.go:26`) has **no production caller** (gap **G8**) |
| No unnecessary "flexibility" added | ⚠️ — the undocumented `--depth` alias (`cmd/mem/path.go:29`) is the one addition beyond the spec'd flags |
| Only touched files required for task | ✅ |
| Didn't "improve" unrelated code | ✅ |
| Matches existing patterns/style | ✅ — the CLI follows the ADR-040 `resolveStorageAndRepo`/`applyStorageOverride` pattern used by the other subcommands |
| Would a senior engineer approve? | ⚠️ — the engine is clean and readable, but a dead exported helper with the defaults re-declared twice, plus a redundant unguarded branch, would draw a comment |
| Tests map to ACs and are non-shallow (spot-check P1) | ❌ — the P1 spot-check exposes the weak spots: no assertion on `EpistemicStatus`/`Weight`/`Summary`, and M7 survives |
| Spec-anchored outcome check | ⚠️ — 5/8 ACs fully anchored; 3 partial; 7 spec-precision gaps flagged |
| Per-layer coverage expectation | ❌ — domain layer is 1:1; the **PostgreSQL layer is only covered in the trivial self-path case**, and the repo-isolation filter is never asserted (gap **G5**) |
| Every test in scope maps to a spec AC / edge case | ✅ — 16 path test functions, all mapped; no unclaimed path test |
| Documented guidelines followed | ✅ — `.agents/rules/always-quality-gate.md`, `.agents/rules/always-test.md` (see the environment notes for the `gofmt` item) |

---

## Edge Cases

| Edge case | Handled? | Evidence |
| --------- | --------- | -------- |
| source == target → 0 hops, cost 0 | ✅ (but unguarded — M7) | `internal/graph/path_test.go:8` |
| non-existent node → clear error | ✅ | `internal/graph/path_test.go:40` `"origem 'inexistente.md' não encontrado"`; `:45` `"destino 'inexistente.md' não encontrado"` |
| empty source/target → error | ✅ | `internal/graph/path_test.go:49` |
| unreachable component → `found:false` | ✅ | `internal/graph/path_test.go:55` |
| cyclic graph → termination, no infinite loop | ✅ | `internal/graph/path_test.go:243` |
| path longer than `max_depth` → not found | ✅ | `internal/graph/path_test.go:209` |
| directed vs undirected | ✅ | `internal/graph/path_test.go:168` |
| CLI missing args | ✅ | `cmd/mem/path_test.go:16` |
| CLI non-existent node | ✅ | `cmd/mem/path_test.go:34` |
| MCP missing `source`/`target` | ⚠️ — the error is raised with `CodeInvalidParams` (`internal/mcp/path_handlers.go:31`, `:40`) but the test only asserts `err != nil` and the message text (`internal/mcp/path_handlers_test.go:140-156`); the **code** is not asserted (gap **G9**) |
| MCP malformed JSON | ⚠️ — error asserted (`internal/mcp/path_handlers_test.go:133`), code not asserted |
| route found end-to-end on a real vault | ❌ — see gap **G1** |

---

## Gate Check

- **Gate command**: `tasks.md` has **no** "Gate Check Commands" section, so the full suite was run per the brief and per `.agents/rules/always-quality-gate.md`.
- **Commands**: `go test -count=1 ./...` (with `MY_MEMORY_PG_TEST=1` and the Postgres URL taken from the global config — never printed, logged or written), then `go build ./...` and `go vet ./...`.
- **Result**: **`go test -count=1 ./...` → exit 0** (28 packages `ok`), **`go build ./...` → exit 0**, **`go vet ./...` → exit 0**.
- **Skipped tests**: none reported as skipped in the run output.
- **Test count before feature**: 252 `func Test` (parent `22de8de`) → **after**: 281 (`9d3e0ee`) — **delta +29**, of which **16 belong to path discovery** (8 `internal/graph/path_test.go`, 4 `cmd/mem/path_test.go`, 3 `internal/mcp/path_handlers_test.go`, 1 `internal/db/graph_test.go:TestFindPath`) and 13 to ADR-028 (staleness), which shipped in the same commit.
- **First run anomaly**: an initial concurrent run (full suite + sensor at the same time) failed `internal/ci` with `enumerar pacotes: pacote "./..." fora do modulo`. **Reproduced as a concurrency artifact, not a defect**: the same package passes standalone in the real tree (`ok … internal/ci 31.299s`) and in the isolated worktree (`ok … 6.294s`). The clean, non-concurrent re-run is green.

### Environment notes (not feature defects)

- **`gofmt -l .` is non-empty in the local working tree** (~130 files, including the feature's). Cause: `core.autocrlf=true` plus a working tree checked out *before* `.gitattributes` was added. The repo's own `.gitattributes` (`* text=auto eol=lf`) documents exactly this. **In a fresh checkout the feature is gofmt-clean**: `gofmt -l .` in the isolated worktree lists **only** the scratch file created by this verification (`cmd/zzscratchcleanup/main.go`). The feature files are LF and correctly formatted (`internal/graph/path.go`: 366 LF / 0 CRLF in the worktree).
- **Live smoke test** (binary built into the scratch area, real SQLite vault, 3 notes, wikilinks): `mem path "a.md" "Nota C"` → `✅ Encontrado, 1 salto, custo 1.000` with the ASCII route (`links_to [EXTRACTED | forward]`) and the segment table — the CLI and renderer behave exactly as the spec asks.

---

## Plan vs. reality (the file this report replaces)

The previous `validation.md` was a plan with checkboxes already ticked. Its claims did not survive verification:

| Plan claim | Reality |
| ---------- | ------- |
| `TestFindPath_EpistemicCostTieBreaker`, `TestFindPath_IdenticalSourceAndTarget`, … | ✅ these exist |
| `TestDB_FindPath` | ❌ the real name is `TestFindPath` (`internal/db/graph_test.go:250`) |
| `TestPostgres_FindPath` — "validando isolamento por repositório" | ❌ **no such test exists**; the only Postgres assertion is the self-path case inside `TestPostgresStore_Integration` (`internal/store/postgres_test.go:272`) |
| `TestMCP_MemoryFindPath_Valid` / `_MissingParams` / `_ToolRegistration` | ❌ real names are `TestMemoryFindPathHandler_Direct` / `_Validation` / `TestMemoryFindPath_ServerRegistration` |
| `TestPathCLI_BasicRoute` / `_JSONOutput` / `_FlagsValidation` | ❌ real names are `TestPathCLI_MissingArgs` / `_NonExistentNode` / `_TextOutputAndJSON` / `_UndirectedAndHops` |
| "Valida erro com código `InvalidParams`" | ⚠️ the error **code** is not asserted (gap **G9**) |

---

## Fix Plans

### FIX-1 — Guard the identical-node branch (or delete it)

- **Root cause**: `internal/graph/path.go:166-179` is redundant with the generic Dijkstra result and nothing distinguishes it.
- **Fix task**: assert the branch-specific `Summary` (or the CLI/MCP "origem e destino são o mesmo nó" rendering) in `internal/graph/path_test.go` and `cmd/mem/path_test.go`; if no distinct behavior is wanted, delete the branch.
- **Verify**: re-run mutant M7 — it must now fail the suite.
- **Priority**: Major (weak test / uncovered AC P1.3).

### FIX-2 — Unify the note identity so `mem path` works on real vaults (gap G1)

- **Root cause**: the indexer writes edges whose source is the document **path id** and whose target is the wikilink **title id** (`cmd/mem/main.go:1941` and `cmd/mem/main.go:2184`: `InsertEdgeWithProps(ctx, …, docID, edge.Target, …)`), so each note occupies two graph nodes (path id and title id) and every edge crosses that split. Measured on the owner's real vault (read-only SQL, 2026-11-15): `graph_edges` = 40,243, **0** with both endpoints on document path ids; `graph_nodes` = 3,601, of which 401 carry a document path id and 3,200 are title-style. `README` has degree **3** by path id vs **66** by title id. The feature's `ResolveNodeCanonicalID` resolves a path-form identifier to the *document* id (`internal/db/graph.go:582`, `internal/store/postgres.go:1018`), i.e. to the side of the split that no edge connects.
- **Reproduction** (read-only, against the live vault): `mem path "CONTRIBUTING.md" "README" --storage=postgres --repo repo_e4c8f3b1a2d5` → `❌ Não encontrado`, even though a `links_to` edge exists between those two notes.
- **Scope note**: the root cause is **outside this feature's diff surface** — the edge-insertion lines were last touched by `171df1e` (2026-09-13), one day *before* the feature commit `9d3e0ee` (2026-09-14). The pathfinding engine needs **no** change: in an isolated vault, `mem path "a.md" "Nota C"` finds the 1-hop route with the correct cost and rendering.
- **Fix task (either)**: (a) resolve the wikilink target to the canonical document id at insertion time (`cmd/mem/main.go:1941`, `:2184`), or (b) make `FindPath`/`ResolveNodeCanonicalID` canonicalize both identities onto one node id.
- **Verify**: re-run the reproduction above and expect `✅ Encontrado`; add a regression test over a vault with real wikilinks.
- **Priority**: Major (AC P1.1 fails in the product).

### FIX-3 — Disambiguate node resolution (gap G2)

- **Root cause**: `internal/store/postgres.go:1030-1039` and `internal/db/graph.go:600-609` resolve by `SELECT id FROM graph_nodes WHERE (id = $2 OR name = $2) LIMIT 1` with **no `ORDER BY`** and no disambiguation. The real vault has **7** nodes named `README`; the query returned `D:\repository\my-memory\.agents\skills\create-adr\README.md` for the query `"README"` — a *different* note, silently.
- **Fix task**: order candidates deterministically (prefer the vault-root document, then shortest path) and/or surface an ambiguity error; the fuzzy fallback (`internal/store/postgres.go` step 4, `internal/db/graph.go` step 4) has the same weakness.
- **Verify**: a test with two same-titled notes asserting which one is chosen.
- **Priority**: Major (silently targets the wrong note).

### FIX-4 — Assert the edge details required by AC P1.1 (gap G4)

- **Fix task**: in `internal/graph/path_test.go`, assert `res.Edges[i].EpistemicStatus` and `res.Edges[i].Weight` on a mixed `EXTRACTED`/`INFERRED`/`TAG` graph (the only `EpistemicStatus` occurrences today are mock *inputs* at `internal/mcp/path_handlers_test.go:48`, `:57`).
- **Priority**: Major.

### FIX-5 — Cover the PostgreSQL path layer (gap G5)

- **Fix task**: add a Postgres integration test with **two repositories** asserting that `FindPath` does not cross the repository boundary (`internal/store/postgres.go` filters `WHERE ($1 = '' OR repository = $1)`) and that a multi-hop route is returned. The current assertion (`internal/store/postgres_test.go:272`) only checks `Found && Hops == 0` for a self-path.
- **Priority**: Major.

### FIX-6 — Assert the not-found message and the CLI/MCP renderings (gaps G6, G7, G9)

- **Fix task**: assert `PathResult.Summary` on the unreachable case (`internal/graph/path.go:323`), the CLI not-found block (`cmd/mem/path.go:126-127`), the MCP `Rota Conectada` and `Segmentos Percorridos` headings (`internal/mcp/path_handlers.go:124`, `:138`), and the `CodeInvalidParams` code (`internal/mcp/path_handlers.go:31`, `:40`).
- **Priority**: Minor.

### FIX-7 — Use or delete `DefaultPathOptions()` (gap G8)

- **Fix task**: `graph.DefaultPathOptions()` (`internal/graph/path.go:26`) has no production caller — the defaults are re-declared as literals in `cmd/mem/path.go:50-72` and `internal/mcp/path_handlers.go:43-65`. Call the helper from both, or delete it. This removes three sources of truth for `max_depth=6`, `directed=true`, `mode=epistemic`.
- **Priority**: Minor.

---

## Requirement Traceability

| Requirement | Previous Status | New Status |
| ----------- | --------------- | ------------ |
| G1 engine | Implementing | ✅ Verified |
| G2 persistence | Implementing | ✅ Verified (coverage gap FIX-5) |
| G3 CLI | Implementing | ✅ Verified |
| G4 MCP tool | Implementing | ✅ Verified |
| G5 ADR + docs | Implementing | ✅ Verified |
| P1.1 | Implementing | ❌ Needs Fix (FIX-2, FIX-4) |
| P1.2 | Implementing | ⚠️ Partial (FIX-6) |
| P1.3 | Implementing | ⚠️ Covered but unguarded (FIX-1) |
| P2.1 | Implementing | ✅ Verified |
| P2.2 | Implementing | ✅ Verified |
| P2.3 | Implementing | ✅ Verified |
| P3.1 | Implementing | ✅ Verified |
| P3.2 | Implementing | ⚠️ Partial (FIX-6) |

---

## Summary

**Overall**: ❌ Not Ready

**Spec-anchored check**: 5/8 ACs fully covered, 3 partial, 7 spec-precision gaps
**Sensor**: 6/7 mutations killed (1 survivor — M7)
**Gate**: `go test -count=1 ./...` exit 0, `go build ./...` exit 0, `go vet ./...` exit 0

**What works**
- The Dijkstra engine is correct. The depth-aware dominance pruning is sound (a state is discarded only when a state with `depth ≤ d` **and** `cost ≤ c` already exists), the `hops` mode is exactly BFS-equivalent, cycles terminate, and the epistemic inversion `c = 1/w` behaves as ADR-027 specifies (`EXTRACTED` 1.0, `INFERRED` ≈1.67, `TAG` ≈3.33).
- The CLI is complete and correct: flags, ASCII route, segment table, JSON output, and both storage engines. Verified live.
- The MCP tool is registered with a formal JSON Schema (`required: [source, target]`), is listed by `tools/list`, and is wired into **both** stdio and HTTP/SSE through the single `runMCPServer`.
- The documentation surface (ADR-027 + 4 indexes/manuals + `STATE.md`) is complete.
- The gate is green, `go vet` is clean, and a fresh checkout is gofmt-clean.

**Issues found**
1. **FIX-2** — AC P1.1 fails end-to-end on real data: the indexer gives each note two graph identities and every edge crosses the split, so `mem path` between two wikilink-connected notes returns "não encontrado" (0 of 40,243 edges in the live vault have both endpoints on document path ids). Root cause is **outside this feature's diff surface** (`171df1e`, one day before the feature commit); the path engine itself needs no change.
2. **FIX-3** — node resolution is ambiguous and arbitrary (`LIMIT 1` with no `ORDER BY` over 7 same-named nodes): `"README"` silently resolved to a skill's README.
3. **FIX-1** — mutant M7 survived: the identical-node branch is redundant and unguarded.
4. **FIX-4** — `EpistemicStatus` and `Weight` on the returned edges are never asserted.
5. **FIX-5** — the PostgreSQL path layer is only covered by a self-path; repo isolation is never asserted.
6. **FIX-6** — the not-found message, the CLI not-found block, the MCP route/table headings and the `InvalidParams` code are unasserted.
7. **FIX-7** — `DefaultPathOptions()` is dead in production; the defaults are duplicated in two other places.
8. **Spec precision** — 7 gaps (S1–S7), including the P3 "Markdown e JSON" promise that the text-only response does not meet (already tracked as ISSUE-022).

**Next steps**: route FIX-1 … FIX-7 to an implementer (FIX-2 and FIX-3 first — they are the ones that make the feature usable on a real vault), then re-verify with the same sensor (M7 must now die) and re-run the gate.

---

## Verification hygiene

- The real tree was never mutated during the sensor: the scratch was a detached `git worktree` at `D:\.minimax\scratch\subagents\verifier-gpd\wt` (`e498a34`); `git status --porcelain` on `D:\repository\my-memory` was empty before the sensor and empty after it. **`git stash` / `git stash pop` were never used.**
- Writes made by this verification in the real tree: this report plus the lessons bookkeeping (`.specs/lessons.json`, `.specs/LESSONS.md`, both owned by `scripts/lessons.py`).
- A third file shows as modified in the final `git status`: `.specs/features/context-packager-and-subgraph-bundle/validation.md`. It was **not** written by this verification - it is the output of a sibling Verifier agent running concurrently on another feature in the same repository (its report references the scratch path `verifier-pack`). It was left untouched and not reverted.
- Side effects created by the live smoke test were removed: the scratch vault's rows were deleted from the owner's Postgres (`documents` 3, `graph_nodes` 5, `graph_edges` 3, `chunks` 3 — all re-verified at 0 for `repo_3311cbc98d9c`) and its entry was removed from the global catalogue `~/.memory/config.yaml` (the pre-existing `repo_e4c8f3b1a2d5` entry was left untouched).