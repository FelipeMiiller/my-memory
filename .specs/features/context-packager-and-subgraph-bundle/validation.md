# context-packager-and-subgraph-bundle Validation

**Date**: 2026-11-15
**Spec**: `.specs/features/context-packager-and-subgraph-bundle/spec.md`
**Diff range**: `aa1b79c^..6a572bd` (5 commits, 2026-09-14 — T1..T5)
**Verifier**: independent sub-agent (author != verifier)
**Branch / HEAD**: `develop` @ `e498a34`
**Method**: read-only over the real tree. Every mutation ran in a throwaway worktree (`D:\.minimax\scratch\subagents\verifier-pack\wt`), removed afterwards. No `git stash`. The real tree was `git status --porcelain` empty before and after.

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1 | ⚠️ Partial | Engine is complete, but the "pesos epistemicos de arestas" priority input is never read (`internal/graph/pack.go` reads only `e.Type`), and 3 of its behaviors have no discriminating test (M2/M3/M4 survive the whole suite). |
| T2 | ✅ Done | `internal/db/graph.go:989`, `internal/store/postgres.go:1447`, interface at `internal/store/store.go:271`. The Postgres path really executes (see Gate). |
| T3 | ✅ Done | `cmd/mem/pack.go:23`; switch case at `cmd/mem/main.go:748`. |
| T4 | ✅ Done | `internal/mcp/pack_handlers.go:16`; catalog `internal/mcp/tools.go:522`; registration `internal/mcp/server.go:112`; both transports wired at `cmd/mem/main.go:3236` (Postgres) and `cmd/mem/main.go:3437` (SQLite). |
| T5 | ✅ Done | ADR-029 + `docs/CLI_GUIDE.md` §21 + `docs/AGENT_INTEGRATION_GUIDE.md` + `docs/README.md` + `docs/adr/README.md` + `.specs/STATE.md`. |

---

## Spec-Anchored Acceptance Criteria

The spec has no "Acceptance Criteria" section; its normative content is the EARS block **R1–R5** (plus Goals G1–G5). R1–R5 are treated as the ACs and expanded into 12 checkable criteria. Evidence-or-zero: a criterion with no `file:line` assertion counts as NOT covered.

| Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion | Result |
| ------------------------- | -------------------- | ----------------------- | ------ |
| R1 UBIQUITOUS — expand from the canonical root up to `max_depth` **respecting `direction`** (`both`/`outbound`/`inbound`) | the collected node set must differ per direction | `internal/graph/pack_test.go:176` — `if len(res.Nodes) != 3` (a node count only, inside a 3-cycle where `outbound` and `both` yield the same set) | ❌ GAP — the assertion does not target the direction outcome. Mutant **M3** (direction filter deleted) survives the entire suite. The *implementation* is correct (probe: `both -> [root in out]`, `outbound -> [root out]`, `inbound -> [root in]`); the gap is in the test. |
| R2 WHEN — peripheral content over the remaining budget -> try the L0/L1 micro-abstract; if it still does not fit -> `omitted` | tier becomes `fringe`, then `omitted` | `internal/graph/pack_test.go:111` — `heavyNode.Tier != TierFringe`; `internal/graph/pack_test.go:141,144` — `res.OmittedCount != 1`, `res.OmittedNodes[0] != "Child1"` | ✅ PASS — both branches asserted; mutant **M1** is killed by these. |
| R2 UBIQUITOUS — "o total de tokens do bundle gerado nunca deve exceder `max_tokens`" | `EstimateTokens(bundle) <= max_tokens` | `internal/graph/pack_test.go:114` — `res.TotalTokens > opts.MaxTokens` (asserts the *accounted counter*, not the bundle) | ❌ GAP — **measured violation**: `max_tokens=1000` -> `res.TotalTokens=995` (assertion passes) but `EstimateTokens(res.Markdown)=1370` (**+370, 37% over budget**). ADR-029:22 removes the ambiguity: "garantir matematicamente que o bundle final nao exceda o limite". |
| R3.1 — "metadados e resumo do orcamento (tokens usados, nos principais, perifericos e omitidos)" | the bundle carries the budget/tier summary | `internal/graph/pack.go:427-428` (header: root, depth, `~tokens/max`, "Nos Coletados"); sections at `internal/graph/pack.go:442`, `:468`, `:489` | ⚠️ Partial — tokens used and the collected count are in the Markdown, but the core/fringe/omitted **breakdown** exists only in the CLI stdout (`cmd/mem/pack.go:117-119`) and in the JSON, not in the bundle. Spec-precision: R3.1 does not say where the summary must live (see SP1). |
| R3.2 — Mermaid topology block (`graph TD`) | the bundle carries a `graph TD` diagram | `internal/graph/pack_test.go:62` — `strings.Contains(res.MermaidGraph, "graph TD")`; impl `internal/graph/pack.go:369` | ✅ PASS |
| R3.3 — section `## 📄 Documentos Centrais` | the section exists in the bundle | impl `internal/graph/pack.go:442`; **no test asserts this header** (probe-verified only) | ⚠️ GAP — implemented but unasserted |
| R3.4 — section `## 📑 Resumos Periféricos (L0/L1)` | the section exists when there are peripheral nodes | impl `internal/graph/pack.go:468`; **no test asserts this header** (probe-verified only) | ⚠️ GAP — implemented but unasserted |
| R3.5 — section `## ⚠️ Nós Omitidos por Orçamento` | the section exists when a node is omitted | `internal/graph/pack_test.go:147` — `strings.Contains(res.Markdown, "Nós Omitidos por Orçamento")`; impl `internal/graph/pack.go:489` | ✅ PASS |
| R4 WHEN — `mem pack <root> [--out f]` shows the metrics and saves the bundle (or prints it) | metrics on stdout, bundle in the file | `cmd/mem/pack_test.go:71` (title in stdout), `:108` (file content), `:111` (`"Arquivo Salvo:"`) | ✅ PASS |
| R4 WHEN — `--json` returns nodes + metrics + Markdown as valid JSON | the JSON payload carries nodes, metrics and the Markdown | `cmd/mem/pack_test.go:86` (valid JSON), `:89` (`res.RootID`), `:92` (`res.CoreCount >= 2`) | ⚠️ Partial — the test asserts only the root and the core count; neither `Nodes` nor `Markdown` is asserted, although the struct does emit them (`internal/graph/pack.go:71,75`). |
| R5 WHEN — the agent calls `memory_pack_context` -> the server returns the bundle Markdown | the Markdown is returned as the tool text | `internal/mcp/pack_handlers_test.go:92-95` — `strings.Contains(txt, "Pacote de Contexto: Raiz Teste")` | ✅ PASS |
| R5 — "avisos de desatualizacao (staleness banner), se aplicavel" | the stale index is warned about in the tool output | `internal/mcp/handlers.go:1034` (the tool is in `isQueryTool`), `:1047-1050` (check), `:1053-1061` (banner prepended); generic test `internal/mcp/staleness_handlers_test.go:17` | ✅ PASS — provided by the generic server-level injection, not by the handler itself. |

**Status**: ❌ Gaps present — **6/12** criteria matched the spec outcome with a discriminating assertion; **2** measured violations (R1, R2 UBIQUITOUS); **2** unasserted-but-implemented (R3.3, R3.4); **2** partial (R3.1, R4 --json); **2** spec-precision gaps (SP1, SP2).

### Spec-precision gaps

| # | Gap | Consequence |
| - | --- | ----------- |
| SP1 | R3.1 does not say **where** the budget/tier summary must live (inside the Markdown, or only in the CLI/JSON). | The implementation puts only `~tokens/max` and "Nos Coletados" in the Markdown; the tier breakdown lives in the CLI/JSON. A test cannot decide which is correct. |
| SP2 | Neither the spec nor ADR-029 defines an expected **order** for the selected nodes (which note is "more useful"), although G1/T1/ADR-029 demand a priority by PageRank and epistemic weights. | There is no spec-defined value to assert, which is exactly why mutant **M4** survives. The spec must state the order (depth -> PageRank desc -> ID) before it can be tested. |

---

## Discrimination Sensor

Scratch: `git worktree add D:\.minimax\scratch\subagents\verifier-pack\wt HEAD` (removed after). **No `git stash` was used.** Baseline `git status --porcelain` on the real tree was empty before the sensor and is empty after cleanup; HEAD is still `e498a34` on `develop`.

| # | Mutation | File:line | Description | Killed? |
| - | -------- | --------- | ----------- | ------- |
| M1 | budget bypassed in the TierCore allocation | `internal/graph/pack.go:284` | `totalTokensUsed+contentTokens+20 <= opts.MaxTokens` -> `<= opts.MaxTokens*4` | ✅ Killed — `TestPackContext_BudgetConstraintAndFallback`, `TestPackContext_OmittedNodes` |
| M2 | root loses its preferential treatment | `internal/graph/pack.go:247` | `if cand.depth == 0 {` -> `if false && cand.depth == 0 {` | ❌ **Survived** — narrow scope (`internal/graph`) and broad scope (`graph`+`mcp`+`db`+`store`+`cmd/mem`) -> weak test |
| M3 | direction filter removed from the BFS | `internal/graph/pack.go:147,150` | both `if opts.Direction == ...` conditions -> `if true {` | ❌ **Survived** — narrow and broad scope -> weak test (breaks R1) |
| M4 | PageRank priority inverted | `internal/graph/pack.go:213` | `candidates[i].pageRank > candidates[j].pageRank` -> `<` | ❌ **Survived** — narrow and broad scope -> weak test |
| M5 | `memory_pack_context` removed from the tool catalog | `internal/mcp/server.go:112` | `s.RegisterTool(ToolMemoryPackContext, ...)` deleted | ✅ Killed — `TestMemoryPackContextTool_Registered` |
| M6 | BFS never expands a neighbor | `internal/graph/pack.go:142` | `if curr.depth >= opts.MaxDepth {` -> `if true {` | ✅ Killed — `TestPackContext_Basic`, `TestPackContext_BudgetConstraintAndFallback`, `TestPackContext_OmittedNodes`, `TestPackContext_DirectionAndCycles` |

**Sensor depth**: lightweight (6 manual behaviour-level mutations; no mutation tooling is available in this repo).
**Result**: **3/6 killed** — ❌ FAIL. Three surviving mutants are fix tasks (see Fix Plans 2, 3, 5).

### Dead-in-production path (test-only)

`graph.DefaultPackOptions()` (`internal/graph/pack.go:28`) is exported but its **only callers are tests**: `internal/graph/pack_test.go:187` and `internal/store/postgres_test.go:293`. Production builds the options literally in two more places — `cmd/mem/pack.go:57-62` and `internal/mcp/pack_handlers.go:67-72` — and `graph.PackContext` re-applies its own defaults at `internal/graph/pack.go:94-102`. The defaults are therefore duplicated in four places with no single source of truth. (`graph.EstimateTokens` is **not** dead: it is reused in production by `internal/mcp/code_handlers.go:1032`.)

---

## Code Quality

| Principle | Status | Evidence |
| --------- | ------ | -------- |
| No features beyond what was asked | ⚠️ | `--no-abstracts` (`cmd/mem/pack.go:32`) and the `--max-depth` alias (`cmd/mem/pack.go:28`, `:50-52`) are not in the spec, ADR-029 or G3 (which lists `--depth`, `--max-tokens`, `--direction`, `--out`, `--json`). Minor scope creep. |
| No abstractions for single-use code | ✅ | `buildMermaidSubgraph` / `renderMarkdownBundle` are single-use but are cohesive seams with clear contracts. |
| No unnecessary "flexibility" added | ⚠️ | the `--max-depth` alias duplicates `--depth` with an awkward resolution rule; `DefaultPackOptions()` is exported but unused in production (above). |
| Only touched files required for the task | ✅ | 24 files in `aa1b79c^..6a572bd`, all inside the feature surface. |
| Didn't "improve" unrelated code | ✅ | no unrelated refactor in the range. |
| Matches existing patterns / style | ✅ | `gofmt` clean, `go vet` clean, same conventions as the sibling handlers. |
| Would a senior engineer approve? | ⚠️ | dead `availableBudget` (`internal/graph/pack.go:221-224`, computed and never read) and dead `nodeIDs` (`internal/graph/pack.go:371,387`, written and never read); the budget accounting does not match the documented promise in `docs/CLI_GUIDE.md` §21. |
| Tests map to ACs and are non-shallow | ❌ | R1, R2 UBIQUITOUS, R3.3 and R3.4 have no discriminating assertion; 3 mutants survive. |
| Spec-anchored outcome check (asserted values match the spec) | ❌ | see the AC table. |
| Per-layer coverage expectation | ⚠️ | domain layer partial (direction, priority and real bundle size untested); MCP layer good (happy + invalid-params + internal error); CLI layer good (stdout + JSON + file + missing root). |
| Every test maps to a spec AC (no unclaimed tests) | ✅ | 14 new test functions, each mapping to R1–R5. Note: `TestPackContext_DirectionAndCycles` *claims* to cover the direction but does not discriminate it (M3). |
| Documented project guidelines followed | ✅ | AGENTS.md "Comandos Operacionais" (`go test`, `go build`, `gofmt`) and `.agents/rules/always-test.md` — the gate is green and `gofmt` reports nothing. |

---

## Edge Cases

| Edge case | Result | Evidence |
| --------- | ------ | -------- |
| Root content larger than the whole budget -> the root is degraded to `TierFringe` while a peripheral node takes `TierCore` | ⚠️ measured, spec silent (no AC covers it) | probe: root 4000 chars, `max_tokens=200` -> `core=1` (the child), `fringe=1` (the root) |
| `max_tokens <= 80` -> the hardcoded 80-token header overhead consumes the entire budget, so **the root itself is omitted** and the bundle has zero content | ⚠️ measured | probe: `max_tokens=50` -> `core=0 fringe=0 omitted=[root n1]`, `TotalTokens=80` (already over the 50 budget), real Markdown 156 tokens; `max_tokens=80` -> identical |
| Cyclic graph -> no duplicated nodes | ✅ | `internal/graph/pack_test.go:176` |
| Missing root -> error | ✅ | `internal/graph/pack_test.go:181-193`; `internal/mcp/pack_handlers_test.go:31-44` (`CodeInvalidParams`) |
| Node without an abstract -> fallback to the first 200 characters | ✅ implemented, untested | `internal/graph/pack.go:237-244`, `:302-308` |
| Omitted nodes are kept out of the Mermaid diagram | ✅ implemented | `internal/graph/pack.go:373-376` (skip) and `:398-407` (an edge is drawn only when both ends are included) |

---

## Gate Check

`tasks.md` has **no "Gate Check Commands" section**, so the full suite was run, as the validation brief requires.

| Command | Result |
| ------- | ------ |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test -count=1 ./...` | **exit 0** — 28 packages, **27 with tests, all `ok`, 0 failed, 0 skipped** (the 28th, `viewer/node_modules/flatted/...`, has no test files) |

- **Postgres**: `MY_MEMORY_PG_TEST=1` plus `MY_MEMORY_PG_URL` read from `~/.memory/config.yaml`. The URL was never printed, logged or written to any file. `TestPostgresStore_Integration` really executed (PASS, 15.94s) — not skipped — and it covers `PackContext` against Postgres (`internal/store/postgres_test.go:293-299`).
- **Test count before the feature** (`aa1b79c^`): **281** test functions. **After** (`6a572bd`): **295**. **Delta: +14** — exactly the 14 test functions the feature adds; no test was deleted or weakened.
- At HEAD (`e498a34`) the repository holds **1136** test functions and all 14 feature tests are still present.
- **Failures**: none. **Skipped**: none in the feature's packages.

---

## Fix Plans

### Fix 1 (Major) — the emitted bundle can exceed `max_tokens`

- **Root cause**: the budget only accounts for node content/abstract plus a hardcoded `headerOverhead := 80` (`internal/graph/pack.go:220-228`). The rendered Markdown additionally contains the Mermaid block (~73 tokens), the metadata header, the per-core-node metadata lines (ID / path / category / distance / tokens) and the fringe table rows — none of which are counted. `availableBudget` (`internal/graph/pack.go:221-224`) is computed and never read, so the "margem de seguranca de cabecalho" promised by R2 is never applied.
- **Fix task**: charge the structural cost of the bundle (derive it from the sections actually rendered, or re-measure `EstimateTokens(renderedMarkdown)` before accepting each node) and make the allocation use `availableBudget` instead of `opts.MaxTokens`.
- **Where**: `internal/graph/pack.go:218-337` (`PackContext`) and `internal/graph/pack.go:418-497` (`renderMarkdownBundle`).
- **Verify**: a test that builds a 12-neighbor graph with `MaxTokens=1000` and asserts `EstimateTokens(res.Markdown) <= 1000`. Today that value is **1370**.
- **Done when**: the assertion is green and the "Orcamento" the CLI prints equals the real size of the Markdown it wrote.
- **Priority**: Major — it violates an explicit spec requirement (R2 UBIQUITOUS) and an explicit ADR promise (`docs/adr/029-context-packager-e-subgraph-bundle.md:22`), and `docs/CLI_GUIDE.md` §21 advertises the guarantee to the user.

### Fix 2 (Major) — R1's direction filter has no discriminating test

- **Root cause**: `TestPackContext_DirectionAndCycles` (`internal/graph/pack_test.go:152-179`) uses a 3-cycle, in which `outbound` and `both` reach the same 3 nodes, so the assertion cannot tell the directions apart.
- **Fix task**: add a test with one inbound-only and one outbound-only neighbour, asserting the exact node set for `both`, `outbound` and `inbound`.
- **Verify**: the new test must fail when mutant M3 is applied.
- **Done when**: M3 is killed.
- **Priority**: Major — R1 is explicit about the direction filter.

### Fix 3 (Major) — the priority order has neither an assertion nor a spec-defined outcome

- **Root cause**: no test sets `opts.PageRanks`, and the spec defines no expected order (SP2). Mutant M4 survives.
- **Fix task**: (a) state the expected order in `spec.md` (depth -> PageRank descending -> ID); (b) add a test with two same-depth nodes whose PageRanks are inverted and assert the resulting order.
- **Verify**: the new test must fail when mutant M4 is applied.
- **Done when**: M4 is killed and the spec carries the order.
- **Priority**: Major.

### Fix 4 (Minor) — the epistemic edge weight is loaded and then discarded

- **Root cause**: `tasks.md` T1 claims "ordenacao prioritaria por distancia, PageRank e **pesos epistemicos de arestas**", and both persistence layers load `weight` and `epistemic_status` from SQL (`internal/db/graph.go:996,1012-1018`; `internal/store/postgres.go:1454,1473-1479`), but `internal/graph/pack.go` never reads `e.Weight` or `e.EpistemicStatus` — only `e.Type` (`:163`, `:402`). ADR-029:24 promises the same weighting.
- **Fix task**: either implement the epistemic weighting in the priority order, or correct `tasks.md` T1 and ADR-029 to describe what the code does.
- **Priority**: Minor.

### Fix 5 (Minor) — the root's preferential treatment and the R3.3/R3.4 headers are unasserted

- **Root cause**: mutant M2 survives; no test asserts `## 📄 Documentos Centrais` or `## 📑 Resumos Periféricos (L0/L1)`.
- **Fix task**: assert the root's tier explicitly (e.g. the root is `TierCore` when its content fits) and assert the two section headers.
- **Priority**: Minor.

### Fix 6 (Minor) — dead code inside the feature

- **Root cause**: `availableBudget` (`internal/graph/pack.go:221-224`) is computed and never read; `nodeIDs` (`internal/graph/pack.go:371,387`) is written and never read.
- **Fix task**: remove both (or make `availableBudget` actually drive the allocation, per Fix 1).
- **Priority**: Minor.

### Fix 7 (Minor) — the CLI reports the accounted tokens, not the size of the file it wrote

- **Root cause**: `cmd/mem/pack.go:116` prints `res.TotalTokens`, which excludes the structural cost, so the user sees "Orcamento: ~995 / 1000 tokens" for a 1370-token Markdown.
- **Fix task**: after Fix 1, report the real size of the emitted bundle.
- **Priority**: Minor.

---

## Requirement Traceability Update

| Requirement | Previous | New |
| ----------- | -------- | ---- |
| R1 — connected subgraph expansion respecting `direction` | Implementing | ❌ Needs Fix — no discriminating assertion (M3 survives) |
| R2 (WHEN/THEN) — L0/L1 fallback then `omitted` | Implementing | ✅ Verified |
| R2 UBIQUITOUS — bundle never exceeds `max_tokens` | Implementing | ❌ Needs Fix — measured violation (1370 > 1000) |
| R3.1 — metadata / budget summary | Implementing | ⚠️ Partial — breakdown only in CLI/JSON (SP1) |
| R3.2 — Mermaid `graph TD` | Implementing | ✅ Verified |
| R3.3 — `## 📄 Documentos Centrais` | Implementing | ⚠️ Implemented, unasserted |
| R3.4 — `## 📑 Resumos Periféricos (L0/L1)` | Implementing | ⚠️ Implemented, unasserted |
| R3.5 — `## ⚠️ Nós Omitidos por Orçamento` | Implementing | ✅ Verified |
| R4 — CLI metrics, file output, `--json` | Implementing | ✅ Verified (⚠️ Partial on the JSON payload) |
| R5 — MCP returns the Markdown + staleness banner | Implementing | ✅ Verified |
| G1 — packing engine with token estimate, weighted BFS, tiers, Mermaid | Implementing | ⚠️ Partial — epistemic weights missing (Fix 4) |
| G2 — `PackContext` in SQLite and Postgres | Implementing | ✅ Verified |
| G3 — `mem pack` CLI | Implementing | ✅ Verified |
| G4 — `memory_pack_context` MCP tool | Implementing | ✅ Verified |
| G5 — ADR-029 and operational docs | Implementing | ✅ Verified |

---

## Summary

**Overall**: ❌ Not Ready

- **Spec-anchored check**: 6/12 criteria matched the spec outcome with a discriminating assertion; 2 measured violations; 2 implemented-but-unasserted; 2 partial; 2 spec-precision gaps.
- **Sensor**: 3/6 mutants killed (M1, M5, M6). M2, M3 and M4 survived the whole suite.
- **Gate**: 27/27 test packages green, 0 failed, 0 skipped; `go build` and `go vet` clean; +14 tests, none removed.

**What works**: the subgraph expansion, the 3-tier degradation (core / fringe / omitted), the Mermaid topology, the CLI (`mem pack` with `--out` / `--json`), the MCP tool with its JSON-Schema catalog entry and the server-level staleness banner, both persistence layers, the ADR and the operational docs. The engine is deterministic, local and free of external dependencies, as the spec requires.

**Issues found** (ranked):

1. **The emitted bundle exceeds `max_tokens`** (R2 UBIQUITOUS + ADR-029:22 + `docs/CLI_GUIDE.md` §21) — measured 1370 tokens against a 1000-token budget; the accounting omits the Mermaid block, the header and the per-node metadata. Fix 1.
2. **R1's direction filter is untested** — mutant M3 survives. Fix 2.
3. **The priority order is untested and has no spec-defined outcome** — mutant M4 survives; SP2. Fix 3.
4. **The epistemic edge weight is loaded and discarded** — T1/ADR-029 overstate what the code does. Fix 4.
5. **The root's preferential treatment and the R3.3/R3.4 headers are unasserted** — mutant M2 survives. Fix 5.
6. **Dead code and duplicated defaults** — `availableBudget`, `nodeIDs`, `DefaultPackOptions()` (test-only). Fix 6.
7. **The CLI reports the accounted tokens, not the size of the bundle it wrote** — Fix 7.

**Next steps**: fixes 1–3 are the blocking ones (a violation of an explicit requirement plus two behaviours with no discriminating test). After they land, re-run this validation; the sensor must reach 6/6 and the budget probe must report `real Markdown tokens <= max_tokens`.