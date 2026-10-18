# Fundação de Paridade com o Obsidian — Tasks

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Spec**: `.specs/features/obsidian-parity-foundation/spec.md`
**ADR**: `docs/adr/057-mtime-e-metadados-de-arquivo-no-indice.md`
**Status**: Approved

---

## Test Coverage Matrix

> Generated from codebase sampling, project guidelines, and spec. Guidelines found: `AGENTS.md` (Princípio 4 - testes obrigatórios; Princípio 8 - quality gate), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`, `CONTRIBUTING.md`, `.github/workflows/ci.yml`.

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Schema migration (`internal/db`) | integration | migrate a legacy fixture with no `mtime`; assert build survives and rows default to 0 | `internal/db/*_test.go` | `go test -count=1 ./internal/db/...` |
| Indexer populates `mtime`/`size` | integration | index a fixture file; assert both columns hold the file's real stat values | `internal/watcher/*_test.go`, `cmd/mem/*_test.go` | `go test -count=1 ./internal/watcher/... ./cmd/mem/...` |
| Decay reads `mtime` with `updated_at` fallback | unit | two documents edited at different times, indexed in one run, MUST yield different multipliers; `mtime = 0` MUST fall back | `internal/store/*_test.go` | `go test -count=1 ./internal/store/...` |
| Staleness compares `mtime` | integration | touch a file without reindexing; assert staleness is detected via `mtime`, not `updated_at` | `internal/staleness/*_test.go` | `go test -count=1 ./internal/staleness/...` |
| `mem doctor` missing-mtime count | unit | fixture with one `mtime = 0` row; assert the count is reported | `internal/store/*_test.go` | `go test -count=1 ./internal/store/...` |
| Read-only CLI commands | unit + integration | each command against a three-note fixture vault, compared against the files on disk | `cmd/mem/*_test.go` | `go test -count=1 ./cmd/mem/...` |
| `printHelp` coverage | unit | every dispatch case appears in `printHelp`; a phantom command fails | `cmd/mem/*_test.go` | `go test -count=1 ./cmd/mem/...` |
| New MCP tools | unit | each tool registered with a resolvable schema; missing path returns a tool error, not an empty string | `internal/mcp/*_test.go` | `go test -count=1 ./internal/mcp/...` |
| Documentation | none | no Go layer; verified by the grep assertions in each task | n/a | task-specific grep |

**Coverage Expectation note**: every code-bearing layer above has an automated test. Documentation has none because it declares no runtime behaviour; its criteria are verified by the grep assertions recorded per task.

## Gate Check Commands

> Generated from the repo's own `ci.yml` and `AGENTS.md`, verified locally.

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After tasks touching `cmd/mem` or `internal/mcp` only | `go build ./... && go test -count=1 ./cmd/mem/... ./internal/mcp/...` |
| Full | After any task touching `internal/db`, `internal/store` or `internal/watcher` | `go test -count=1 ./...` |
| Build | After every phase | `go build ./... && go vet ./... && go test -count=1 ./...` |
| Tag-gated | After touching parser or codeast | `go build -tags treesitter ./...` |
| Format | Only valid in a fresh checkout | `gofmt -l .` prints the 239 CRLF files in this working tree — that is the pre-existing condition, not a defect. Prove format cleanliness in a throwaway worktree, never by running `gofmt -w .` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: Persistência de mtime e correção do decaimento

```
T1 → T2 → T3 → T4 → T5
```

### Phase 2: Comandos de leitura

```
T5 → T6 → T7 → T8 → T9 → T10 → T11 → T12
```

### Phase 3: Superfície MCP e documentação

```
T12 → T13 → T14
```

---

## Task Breakdown

### T1: Add `mtime` and `size` to the schema in both engines

**What**: Add `mtime INTEGER` and `size INTEGER` to the `documents` table in the SQLite DDL and in the Postgres DDL, as an idempotent `ADD COLUMN`. Existing rows must end with `mtime = 0` so nothing breaks on an already-indexed vault.
**Where**: `internal/db/schema.go`
**Depends on**: None
**Reuses**: the existing `Schema` / `FallbackSchema` pairing already in this file
**Requirement**: PARITY-01, PARITY-02

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `documents` has `mtime INTEGER` and `size INTEGER` in both the SQLite and Postgres DDL
- [ ] Opening a pre-existing database and running the migration leaves old rows with `mtime = 0`
- [ ] `updated_at` keeps its current meaning and its column definition is unchanged
- [ ] A test migrates a legacy fixture and asserts the columns exist with the 0 default
- [ ] Gate check passes: `go build ./... && go test -count=1 ./internal/db/...`

**Tests**: integration
**Gate**: full

**Commit**: `feat(db): add mtime and size to documents in both engines`

---

### T2: Persist `mtime` and `size` when indexing

**What**: Populate both columns from the file's real stat at index time, in both entry paths: the `mem index` walk in `cmd/mem/main.go` and the watcher in `internal/watcher/indexer.go`.
**Where**: `cmd/mem/main.go`
**Depends on**: T1
**Reuses**: `store.InsertDocumentWithMeta`, the single write point already used by both paths
**Requirement**: PARITY-03

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `mem index` stores the file's actual mtime and byte size
- [ ] The watcher path stores the same values, not zeros
- [ ] No second insert path exists; both callers go through the same function
- [ ] A test indexes a fixture file and asserts `mtime` matches the file on disk
- [ ] Gate check passes: `go test -count=1 ./internal/watcher/... ./cmd/mem/...`

**Tests**: integration
**Gate**: full

**Commit**: `feat(index): persist file mtime and size into documents`

---

### T3: Make time decay read `mtime` with an `updated_at` fallback

**What**: Carry `mtime` through the search result into the RRF fusion and use it for the decay factor, falling back to `updated_at` when `mtime` is 0. This is the corrective core of the feature.
**Where**: `internal/store/rrf.go`
**Depends on**: T2
**Reuses**: `CalculateTimeDecay`, which stays as-is
**Requirement**: PARITY-04, PARITY-05, PARITY-06, PARITY-07

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `SearchResult` carries `MTime`, populated from `COALESCE(d.mtime, 0)` in the vector, TurboQuant and FTS query paths
- [ ] `FuseSearchResultsWithOptions` uses `MTime` when positive and `UpdatedAt` otherwise
- [ ] A test indexes two documents whose mtimes differ by months in a single run and asserts their decay multipliers differ
- [ ] A test with `mtime = 0` asserts the multiplier matches the one computed from `updated_at`
- [ ] Gate check passes: `go test -count=1 ./internal/store/...`

**Tests**: unit
**Gate**: full

**Commit**: `fix(search): rank by file edit time instead of index time`

---

### T4: Base staleness detection on persisted `mtime`

**What**: `internal/staleness/detector.go` currently infers change from `updated_at`. Change it to compare the file's current mtime against the persisted `mtime`.
**Where**: `internal/staleness/detector.go`
**Depends on**: T3
**Reuses**: n/a
**Requirement**: PARITY-08

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] Staleness compares the on-disk mtime against `documents.mtime`
- [ ] A file touched after indexing is reported stale
- [ ] A document with `mtime = 0` does not crash the detector; it falls back to the previous behaviour
- [ ] A test covers both the detected and the not-detected case
- [ ] Gate check passes: `go test -count=1 ./internal/staleness/...`

**Tests**: integration
**Gate**: full

**Commit**: `fix(staleness): detect drift from persisted mtime instead of index time`

---

### T5: Report documents missing `mtime` in `mem doctor`

**What**: A vault indexed before this feature has every row at `mtime = 0`. Without a report, the gap between disk and index is invisible, which is exactly the failure mode ADR-057 warns about in its implementation note.
**Where**: `internal/store/doctor.go`
**Depends on**: T4
**Reuses**: the existing doctor section accumulator
**Requirement**: PARITY-11

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `mem doctor` reports the count of documents with `mtime = 0`
- [ ] A freshly indexed vault reports zero
- [ ] A legacy fixture reports its actual count
- [ ] Gate check passes: `go test -count=1 ./internal/store/...`

**Tests**: unit
**Gate**: full

**Commit**: `feat(doctor): report documents indexed before mtime existed`

---

### T6: `mem files` with extension and folder filters

**What**: List indexed document paths, alphabetically. Obsidian's `files` command, plus a `folders` sibling.
**Where**: `cmd/mem/main.go`
**Depends on**: T5
**Reuses**: `documents.path`, already UNIQUE and indexed
**Requirement**: PARITY-12, PARITY-13, PARITY-14, PARITY-15

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `mem files` lists every path alphabetically
- [ ] `--ext md` filters by extension
- [ ] `--folder <dir>` filters by vault-relative prefix
- [ ] An unknown extension or folder exits zero with an empty list, not an error
- [ ] A test asserts ordering and both filters
- [ ] Gate check passes: `go test -count=1 ./cmd/mem/...`

**Tests**: integration
**Gate**: quick

**Commit**: `feat(cli): add mem files and mem folders`

---

### T7: `mem read` returns a note's raw Markdown

**What**: Print the note's full text. The content is already stored in `chunks.content` as the raw file.
**Where**: `cmd/mem/main.go`
**Depends on**: T6
**Reuses**: the existing read path from `chunks`
**Requirement**: PARITY-16, PARITY-17

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `mem read <path>` prints the raw Markdown
- [ ] An unknown path exits non-zero naming the path
- [ ] A basename that matches two documents reports the ambiguity instead of picking one
- [ ] A test compares output byte-for-byte against the fixture file
- [ ] Gate check passes: `go test -count=1 ./cmd/mem/...`

**Tests**: integration
**Gate**: quick

**Commit**: `feat(cli): add mem read for raw note content`

---

### T8: `mem outline` and `mem wordcount`

**What**: Heading structure with depth, and word/character counts. Both derive from text already stored.
**Where**: `cmd/mem/main.go`
**Depends on**: T7
**Reuses**: the content accessor added in T7
**Requirement**: PARITY-18, PARITY-19, PARITY-20

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `mem outline` prints each heading with its level
- [ ] A note with no headings prints nothing and exits zero
- [ ] `mem wordcount` prints word and character counts
- [ ] Tests cover both commands, including the empty-outline case
- [ ] Gate check passes: `go test -count=1 ./cmd/mem/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(cli): add mem outline and mem wordcount`

---

### T9: `mem random`

**What**: Print a randomly selected document. Must not be silently broken on an empty index.
**Where**: `cmd/mem/main.go`
**Depends on**: T8
**Reuses**: the documents query from T6
**Requirement**: PARITY-21, PARITY-22, PARITY-23

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `mem random` prints a path and title
- [ ] `--json` prints a single JSON object with at least `path` and `title`
- [ ] An empty index exits non-zero saying the index is empty
- [ ] Tests cover the text form, the JSON form and the empty case
- [ ] Gate check passes: `go test -count=1 ./cmd/mem/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(cli): add mem random`

---

### T10: `mem deadends`

**What**: Documents with at least one outgoing edge and no incoming edge. Derivable from `graph_edges` with a `NOT EXISTS` on the target side.
**Where**: `internal/db/graph.go`
**Depends on**: T9
**Reuses**: the CTE pattern already used for `OrphanNotes`
**Requirement**: PARITY-24

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `mem deadends` lists documents that have outgoing edges and no incoming edges
- [ ] A fully isolated document is not listed, matching Obsidian's definition
- [ ] A test with a two-document chain asserts only the head is reported
- [ ] Gate check passes: `go test -count=1 ./cmd/mem/... ./internal/db/...`

**Tests**: integration
**Gate**: full

**Commit**: `feat(graph): add mem deadends for documents with no inbound edges`

---

### T11: `mem file` and `mem recents`

**What**: The two commands ADR-057 exists to unblock. `file` reports path, title, size and mtime; `recents` lists documents ordered by mtime descending.
**Where**: `cmd/mem/main.go`
**Depends on**: T10
**Reuses**: the `mtime` and `size` columns added in T1
**Requirement**: PARITY-09, PARITY-10

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `mem file <path>` reports path, title, size and mtime
- [ ] `mem recents` orders by `mtime` descending
- [ ] Both read from the index only, never the filesystem, so a Postgres-only vault works
- [ ] A document with `mtime = 0` sorts last rather than first
- [ ] Gate check passes: `go test -count=1 ./cmd/mem/...`

**Tests**: integration
**Gate**: quick

**Commit**: `feat(cli): add mem file and mem recents backed by persisted mtime`

---

### T12: Register the new commands in `printHelp()`

**What**: Extend the help output and add the guard test that makes the dispatch table and the help text impossible to desynchronise.
**Where**: `cmd/mem/main.go`
**Depends on**: T11
**Reuses**: the existing `printHelp` function
**Requirement**: PARITY-25, PARITY-26, PARITY-27

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `printHelp()` lists every command added by this spec
- [ ] A test walks the dispatch table and asserts each case appears in the help output
- [ ] The test fails if a command is registered but undocumented — demonstrate this on a scratch copy before committing
- [ ] Gate check passes: `go test -count=1 ./cmd/mem/...`

**Tests**: unit
**Gate**: quick

**Commit**: `test(cli): guard printHelp against an undocumented command`

---

### T13: Expose the new capabilities as MCP tools

**What**: Four tools — list documents, read a note, read an outline, read file metadata — so an agent gets the same surface without shelling out.
**Where**: `internal/mcp/tools.go`
**Depends on**: T12
**Reuses**: the existing tool registration pattern and the store accessors from Phases 1-2
**Requirement**: PARITY-29, PARITY-30, PARITY-31, PARITY-32, PARITY-33, PARITY-34

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] Four tools registered: list documents, read note, note outline, file metadata
- [ ] Each carries a trigger-style description in the style of the existing twenty
- [ ] The metadata tool returns `mtime` and `size`
- [ ] A missing path returns a tool error, not an empty string
- [ ] Tests cover registration, schema resolution and the missing-path error
- [ ] The MCP tool count is updated everywhere it is currently recorded as twenty
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): expose document listing, read, outline and metadata as tools`

---

### T14: Document the new commands and the decay change

**What**: CLI guide, canonical skill, ADR index and changelog. The `--decay` description in particular is now wrong and must be corrected wherever it appears.
**Where**: `docs/CLI_GUIDE.md`
**Depends on**: T13
**Reuses**: `skills/mymemory-cli/SKILL.md` as the canonical command reference
**Requirement**: PARITY-36, PARITY-37, PARITY-38, PARITY-39, PARITY-40

**Tools**:

- MCP: NONE
- Skill: `my-memory-format`

**Done when**:

- [ ] All nine new commands documented in `docs/CLI_GUIDE.md`
- [ ] `--decay` described as ranking by edit time, everywhere it appears
- [ ] `skills/mymemory-cli/SKILL.md` lists the new commands
- [ ] `docs/adr/README.md` records ADR-057 as implemented
- [ ] `CHANGELOG.md` `[Unreleased]` records the commands and the decay correction
- [ ] Every file written is UTF-8 without BOM
- [ ] `grep` for each new command across the guide and the skill returns a hit

**Tests**: none
**Gate**: build

**Commit**: `docs: document the parity foundation commands and the decay correction`

---

## Phase Execution Map

```
Phase 1 → Phase 2 → Phase 3

Phase 1:  T1 → T2 → T3 → T4 → T5
Phase 2:                    T5 → T6 → T7 → T8 → T9 → T10 → T11 → T12
Phase 3:                                                T12 → T13 → T14
```

Execution is strictly sequential - there is no intra-phase parallelism. A single agent (or batch worker) works one task at a time, in order.

**Batch packing** (~7 tasks per worker, cuts only at phase boundaries):

| Batch | Phases | Tasks | Focus |
| ----- | ------ | ----- | ----- |
| 1 | Phase 1 | T1–T5 | Schema, persistence, decay correction |
| 2 | Phase 2 | T6–T12 | Seven read-only commands |
| 3 | Phase 3 | T13-T14 | MCP surface and documentation |

O validador conta as arestas do diagrama como declaradas: 13 setas, 14 tarefas, uma raiz.

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: schema columns | 1 file, both engines | ✅ Granular |
| T2: indexer persistence | 1 write point + its 2 callers | ✅ Granular |
| T3: decay source | 1 file + query paths | ✅ Granular |
| T4: staleness | 1 file | ✅ Granular |
| T5: doctor report | 1 file | ✅ Granular |
| T6: `files` + `folders` | 2 sibling commands, 1 file | ⚠️ Cohesive — both enumerate the same table |
| T7: `read` | 1 command | ✅ Granular |
| T8: `outline` + `wordcount` | 2 small sibling commands, same content source | ⚠️ Cohesive |
| T9: `random` | 1 command | ✅ Granular |
| T10: `deadends` | 1 query | ✅ Granular |
| T11: `file` + `recents` | 2 sibling commands, both read the new columns | ⚠️ Cohesive — both are the ADR-057 payoff |
| T12: printHelp + guard test | 1 file | ✅ Granular |
| T13: MCP tools | 1 file, 4 registrations | ✅ Granular |
| T14: documentation | one atomic doc commit | ✅ Granular |

---

## Diagram-Definition Cross-Check

| Task | Depends On (task body) | Diagram Shows | Status |
| ---- | ---------------------- | ------------- | ------ |
| T1 | None | start | ✅ Match |
| T2 | T1 | T1 → T2 | ✅ Match |
| T3 | T2 | T2 → T3 | ✅ Match |
| T4 | T3 | T3 → T4 | ✅ Match |
| T5 | T4 | T4 → T5 | ✅ Match |
| T6 | T5 | T5 → T6 | ✅ Match |
| T7 | T6 | T6 → T7 | ✅ Match |
| T8 | T7 | T7 → T8 | ✅ Match |
| T9 | T8 | T8 → T9 | ✅ Match |
| T10 | T9 | T9 → T10 | ✅ Match |
| T11 | T10 | T10 → T11 | ✅ Match |
| T12 | T11 | T11 → T12 | ✅ Match |
| T13 | T12 | T12 → T13 | ✅ Match |
| T14 | T13 | T13 → T14 | ✅ Match |

No task depends on a later phase. ✅

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | schema | integration | integration | ✅ OK |
| T2 | indexer | integration | integration | ✅ OK |
| T3 | decay | unit | unit | ✅ OK |
| T4 | staleness | integration | integration | ✅ OK |
| T5 | doctor | unit | unit | ✅ OK |
| T6–T9, T11 | CLI commands | unit/integration | unit/integration | ✅ OK |
| T10 | graph query | integration | integration | ✅ OK |
| T12 | help + guard | unit | unit | ✅ OK |
| T13 | MCP tools | unit | unit | ✅ OK |
| T14 | documentation | none | none | ✅ OK |

Every layer the matrix marks as tested carries its test inside the same task. ✅

---

## MCPs and Skills

- **MCP**: NONE (the feature registers new tools but consumes none)
- **Skills**: `my-memory-format` (T14), `tlc-spec-driven` (this flow), NONE otherwise
