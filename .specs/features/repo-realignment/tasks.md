# Repositorial — Remoção de Código Morto e Realinhamento Documental — Tasks

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Spec**: `.specs/features/repo-realignment/spec.md`
**Status**: In Progress

---

## Test Coverage Matrix

> Generated from codebase sampling, project guidelines, and spec. Guidelines found: `AGENTS.md` (Princípio 4 - testes obrigatórios; Princípio 8 - quality gate), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`, `CONTRIBUTING.md`, `.github/workflows/ci.yml`.

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Dead-code removal | none | deletion gate only; no new behaviour to test | n/a | `go build ./... && go vet ./... && go test -count=1 ./...` |
| `cmd/mem` config template (`mem init`) | unit | 1:1 to CLEAN-10..CLEAN-16; assert the exact `embedding:` block, not a substring | `cmd/mem/*_test.go` | `go test -count=1 ./cmd/mem/...` |
| `cmd/mem` dispatch/help surface | unit | every registered subcommand appears in `printHelp()` | `cmd/mem/*_test.go` | `go test -count=1 ./cmd/mem/...` |
| CI workflow (`.github/workflows/ci.yml`) | none | config gate; package coverage asserted by an explicit CI-gap script | n/a | CI-gap check command |
| `.gitattributes` | integration | every tracked `*.go` reports `i/lf w/lf`; `gofmt -l .` empty | n/a | `git ls-files --eol "*.go"` |
| Markdown documentation | none | no Go layer; verified by the explicit grep assertions listed in each task's Done-when | n/a | task-specific grep |
| `.specs/` state artifacts | none | no Go layer; verified by grep + file-presence assertions | n/a | task-specific grep |

**Coverage Expectation note**: the documentation and CI layers have no runtime behaviour to unit-test. Their acceptance criteria are verified by the deterministic grep/presence commands recorded per task. Per `AGENTS.md` Princípio 4, this is a build-and-inspection gate, not a silent omission.

## Gate Check Commands

> Generated from the repo's own `ci.yml` and `AGENTS.md`, verified locally.

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After tasks touching `cmd/mem` only | `go build ./... && go test -count=1 ./cmd/mem/...` |
| Full | After any code deletion or dependency change | `go test -count=1 ./...` |
| Build | After every phase | `go build ./... && go vet ./... && go test -count=1 ./... && gofmt -l .` |
| Tag-gated | After deleting packages with build tags | `go build -tags treesitter ./... && go build -tags onnx ./...` |
| Format | Only valid **after T9** | `gofmt -l .` — before `.gitattributes` exists this prints 239 files from local CRLF and is NOT a formatting defect |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: Dead-code removal

```
T1 → T2 → T3 → T4 → T5 → T6
```

### Phase 2: `mem init` config correction

```
T6 → T7 → T8
```

### Phase 3: Infrastructure gates

```
T8 → T9 → T10
```

### Phase 4: Documentation realignment (code-facing)

```
T10 → T11 → T12 → T13 → T14
```

### Phase 5: Project state artifacts

```
T14 → T15 → T16 → T17
```

### Phase 6: Skills consolidation, final cleanup and residual code truth

```
T17 → T18 → T19 → T20 → T21 → T22 → T23
```

---

## Task Breakdown

### T1 ✅ Remove the committed root `main.go` scratch script

**What**: Delete the 36-line `package main` Postgres diagnostic at the repo root that queries `$PGURL` for `repo_central`; nothing imports it and it is not a duplicate of `cmd/mem/main.go`.
**Where**: `main.go`
**Depends on**: None
**Reuses**: n/a
**Requirement**: CLEAN-01, CLEAN-03, CLEAN-09

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `main.go` no longer exists at the repo root
- [ ] `go list ./...` no longer lists the bare module path `github.com/FelipeMiiller/my-memory`
- [ ] Gate check passes: `go build ./... && go vet ./...`
- [ ] Package count drops from 30 to 29

**Tests**: none
**Gate**: build

**Commit**: `chore(repo): remove committed root main.go Postgres scratch script`

---

### T2 ✅ Delete `internal/writer/` and its projection subpackage

**What**: Remove the ADR-044 atomic writer and its outbox projection subscribers; zero importers exist (the only mention is a comment in `internal/mcp/writer_handlers_ifmatch_test.go`).
**Where**: `internal/writer/`
**Depends on**: T1
**Reuses**: n/a
**Requirement**: CLEAN-01, CLEAN-02, CLEAN-09

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `internal/writer/` and `internal/writer/projection/` are gone including their `_test.go` files
- [ ] `internal/mcp/writer_handlers.go` still compiles and still uses `compiler.SyncEngine` — it never imported the writer package
- [ ] A recursive grep for `internal/writer` across `*.go` returns no non-comment hit
- [ ] Gate check passes: `go build ./... && go vet ./... && go test -count=1 ./internal/mcp/...`

**Tests**: none
**Gate**: full

**Commit**: `chore(writer): remove unreachable ADR-044 writer and projection packages`

---

### T3 ✅ Delete `internal/policy/`

**What**: Remove the write-policy engine; its only importer is the writer package deleted in T2.
**Where**: `internal/policy/`
**Depends on**: T2
**Reuses**: n/a
**Requirement**: CLEAN-01, CLEAN-02, CLEAN-09

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `internal/policy/` is gone including `engine.go` and its tests
- [ ] `grep -rn "internal/policy" --include=*.go .` returns no hit
- [ ] Gate check passes: `go build ./... && go vet ./...`

**Tests**: none
**Gate**: build

**Commit**: `chore(policy): remove write-policy engine orphaned with ADR-044`

---

### T4 ✅ Delete `internal/asr/` and `internal/ipc/`

**What**: Remove the ADR-045 ASR config surface (its Electron consumer was deleted in `b477200`) and the ADR-042 JSON-RPC transport that never had a worker to serve.
**Where**: `internal/asr/`
**Depends on**: T3
**Reuses**: n/a
**Requirement**: CLEAN-01, CLEAN-02, CLEAN-09

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `internal/asr/` and `internal/ipc/` are gone including their tests
- [ ] `grep -rn "internal/asr\|internal/ipc" --include=*.go .` returns no non-comment hit
- [ ] ADR-045 and ADR-042 markdown files still exist untouched (history is preserved)
- [ ] Gate check passes: `go build ./... && go vet ./... && go test -count=1 ./...`

**Tests**: none
**Gate**: full

**Commit**: `chore(adr-045): remove ASR scaffolding and ADR-042 JSON-RPC transport`

---

### T5 ✅ Delete the undispatched `mem replay` implementation

**What**: Remove `cmd/mem/replay.go` and `replay_test.go`; no `case "replay"` exists in the dispatch table and `registeredSubscribers()` returns `nil`, so the file always errors.
**Where**: `cmd/mem/replay.go`
**Depends on**: T4
**Reuses**: live replay path `cmd/mem/events.go`
**Requirement**: CLEAN-08, CLEAN-09

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `cmd/mem/replay.go` and `cmd/mem/replay_test.go` are gone
- [ ] `printHelp()` output contains no `replay` entry
- [ ] `cmd/mem/events.go` is untouched — it is the live replay surface
- [ ] Gate check passes: `go build ./... && go test -count=1 ./cmd/mem/...`

**Tests**: none
**Gate**: quick

**Commit**: `chore(cli): remove undispatched mem replay implementation`

---

### T6 ✅ Tidy dependencies and run the full gate

**What**: Run `go mod tidy` after the deletions and verify no build-tagged code lost a dependency.
**Where**: `go.mod`
**Depends on**: T5
**Reuses**: n/a
**Requirement**: CLEAN-03, CLEAN-04, CLEAN-05, CLEAN-06, CLEAN-07

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `go mod tidy` has been run and `go.mod`/`go.sum` are committed
- [ ] `go build ./...`, `go vet ./...` and `go test -count=1 ./...` all exit 0
- [ ] `go build -tags treesitter ./...` exits 0
- [ ] `internal/graphview/template.go`, `internal/supervisor/profile.go`, the `lock_*.go` platform split and `internal/db/vec_*.go` all still exist
- [ ] `go list ./...` reports 24 packages (30 baseline minus 6 removed: root module, writer, writer/projection, policy, asr, ipc/jsonrpc)
- [ ] NOTE: `go build -tags onnx ./...` cannot pass on a machine with `CGO_ENABLED=0` and no C compiler. It fails identically at the pre-cleanup baseline. Record as environmental; do not attempt to fix or weaken it here.

**Tests**: none
**Gate**: full

**Commit**: `chore(deps): tidy go.mod after dead-package removal`

---

### T7 ✅ Fix the `mem init` config template

**What**: Change the generated `embedding:` block from `provider: "ollama"` / `nomic-embed-text` / `dimension: 768` to `provider: "builtin"` / `dimension: 384`, and fix the comment at line ~1013 claiming the builtin model is `nomic-embed-text` at 768.
**Where**: `cmd/mem/main.go`
**Depends on**: T6
**Reuses**: `internal/embedder.EmbeddingDim` and the model id in `internal/embedder/embedder.go`
**Requirement**: CLEAN-10, CLEAN-11, CLEAN-12, CLEAN-13, CLEAN-15, CLEAN-16

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] The generated template contains `provider: "builtin"` and `dimension: 384`
- [ ] The generated template contains neither `nomic-embed-text` nor `localhost:11434`
- [ ] A unit test asserts the exact `embedding:` block produced by the init template writer
- [ ] A unit test asserts a config declaring `dimension: 768` still resolves to `EmbeddingDim`
- [ ] Gate check passes: `go build ./... && go test -count=1 ./cmd/mem/...`

**Tests**: unit
**Gate**: quick

**Commit**: `fix(init): generate builtin/384 embedder config instead of the removed ollama path`

---

### T8 ✅ Correct the repo's own tracked vault config

**What**: Update the committed `.memory/config.yaml`, which still declares `provider: "ollama"`, `nomic-embed-text` and `dimension: 768`.
**Where**: `.memory/config.yaml`
**Depends on**: T7
**Reuses**: the template from T7
**Requirement**: CLEAN-14

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `.memory/config.yaml` declares `provider: "builtin"` and `dimension: 384`
- [ ] The file is UTF-8 without BOM and keeps its existing comments
- [ ] Gate check passes: `go test -count=1 ./internal/config/...`

**Tests**: none
**Gate**: quick

**Commit**: `fix(vault): align tracked .memory/config.yaml with the builtin 384d embedder`

---

### T9 ✅ Add `.gitattributes` to make the format gate passable

**What**: Add `.gitattributes` with `* text=auto eol=lf` so Windows checkouts materialise LF and `gofmt -l .` stops reporting 239 CRLF false positives. Do **not** run `gofmt -w .`.
**Where**: `.gitattributes`
**Depends on**: T8
**Reuses**: n/a
**Requirement**: CLEAN-38, CLEAN-39, CLEAN-40, CLEAN-41

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `.gitattributes` exists and declares `* text=auto eol=lf`
- [ ] No tracked source file was reformatted by this task — the diff touches only `.gitattributes`
- [ ] After `git add --renormalize .`, `git ls-files --eol "*.go"` reports `i/lf` for every tracked Go file
- [ ] The git blob for `HEAD:cmd/mem/graph.go` still contains zero CR bytes
- [ ] `gofmt -l .` is verified green in a **throwaway worktree** freshly materialised from the committed state, not in the current working tree — the current tree keeps its CRLF until a re-checkout, and re-checking out the whole working tree is out of scope and destructive
- [ ] The throwaway worktree is removed afterwards and the real working tree is unchanged (`git status --porcelain` identical to the pre-task baseline)

**Tests**: none
**Gate**: build

**Commit**: `chore(repo): add .gitattributes so the gofmt gate is passable on Windows`

---

### T10 ✅ Close the CI package-coverage gap

**What**: Add the eight packages `ci.yml` never executes — `supervisor`, `event_runtime`, `federation`, `policy`, `asr`, `ipc/jsonrpc`, `writer`, `writer/projection` — adjusting for the ones removed by Phase 1, and add `cmd/mymemoryd` and `bench/codeast`.
**Where**: `.github/workflows/ci.yml`
**Depends on**: T9
**Reuses**: the existing per-package test-step pattern
**Requirement**: CLEAN-34, CLEAN-35, CLEAN-36, CLEAN-37

**Tools**:

- MCP: `github`
- Skill: NONE

**Done when**:

- [ ] Every package from `go list ./...` appears in a CI test step or in a documented exclusion table
- [ ] `internal/supervisor`, `internal/event_runtime` and `internal/federation` are covered
- [ ] No Electron, Node, Vite or Playwright setup step was reintroduced
- [ ] The workflow file remains valid YAML

**Tests**: none
**Gate**: build

**Commit**: `ci: cover the eight packages the workflow never executed`

---

### T11 ✅ Correct README claims

**What**: Fix the two wrong MCP tool counts (19 and 17 → 20) and replace the Ollama/768 narrative with the builtin MiniLM at 384 dimensions.
**Where**: `README.md`
**Depends on**: T10
**Reuses**: `skills/mymemory/references/` for the corrected facts
**Requirement**: CLEAN-17, CLEAN-18

**Tools**:

- MCP: NONE
- Skill: `my-memory-format`

**Done when**:

- [ ] Every tool count in `README.md` reads 20
- [ ] No `nomic-embed-text`, `768` embedding claim, or Ollama dependency remains
- [ ] The documented embedder is `paraphrase-multilingual-MiniLM-L12-v2` at 384 dimensions
- [ ] `grep -rn "nomic-embed-text\|768d\|768 dimensões" README.md` returns zero hits

**Tests**: none
**Gate**: build

**Commit**: `docs(readme): correct MCP tool count and replace ollama/768 with builtin/384`

---

### T12 ✅ Correct the onboarding documents

**What**: Remove the "install Ollama, then `ollama pull nomic-embed-text`" prerequisite from the user-facing guides and fix the conflated 384/Ollama claims.
**Where**: `COMO_USAR.md`
**Depends on**: T11
**Reuses**: `COMO_FUNCIONA.md` and `CONTRIBUTING.md` follow the same pass
**Requirement**: CLEAN-19

**Tools**:

- MCP: NONE
- Skill: `my-memory-format`

**Done when**:

- [ ] `COMO_USAR.md`, `COMO_FUNCIONA.md` and `CONTRIBUTING.md` no longer instruct installing Ollama
- [ ] No document pairs "Ollama" with "384"
- [ ] `grep -rn "ollama" COMO_USAR.md COMO_FUNCIONA.md CONTRIBUTING.md` returns zero hits, except an explicit note that it was removed if kept
- [ ] Files remain UTF-8 without BOM

**Tests**: none
**Gate**: build

**Commit**: `docs(onboarding): drop the ollama prerequisite from user guides`

---

### T13 ✅ Correct the documents under `docs/`

**What**: Remove the Ollama-online/OLSama-offline-fallback model from `CENTRAL_VAULT.md`, `ARCHITECTURE.md`, `README.md` and `BENCHMARKS.md`, and fix the inert `MY_MEMORY_EMBED_PROVIDER` knob in `AGENT_INTEGRATION_GUIDE.md`.
**Where**: `docs/CENTRAL_VAULT.md`
**Depends on**: T12
**Reuses**: `README.md` corrections from T11
**Requirement**: CLEAN-19

**Tools**:

- MCP: NONE
- Skill: `my-memory-format`

**Done when**:

- [ ] No document under `docs/` describes an Ollama fallback path that cannot trigger
- [ ] `MY_MEMORY_EMBED_PROVIDER` is either implemented or explicitly documented as inert
- [ ] `docs/BENCHMARKS.md` attributes the 384 figure to the correct model
- [ ] `grep -rn "nomic-embed-text" docs/` returns zero hits

**Tests**: none
**Gate**: build

**Commit**: `docs(architecture): replace the ollama fallback narrative with the builtin path`

---

### T14 ✅ Document the missing CLI surface and sync `printHelp()`

**What**: Add `mem up`, `mem down`, `mem logs`, `mem profiles`, `mem events`, `mem doctor --events`, `mem search --links` to the guide; fix the `mem code-graph`/`mem code-stats` flag lists; make `printHelp()` list every registered subcommand.
**Where**: `docs/CLI_GUIDE.md`
**Depends on**: T13
**Reuses**: flag registrations in `cmd/mem/codegraph.go` and `cmd/mem/codestats.go`
**Requirement**: CLEAN-20, CLEAN-21, CLEAN-22, CLEAN-23, CLEAN-33

**Tools**:

- MCP: NONE
- Skill: `my-memory-format`

**Done when**:

- [ ] All six missing commands appear in `docs/CLI_GUIDE.md`
- [ ] `--events` and `--links` are documented
- [ ] No documented flag for `mem code-graph` or `mem code-stats` is absent from the real flag registration
- [ ] A unit test asserts every subcommand in the dispatch table appears in `printHelp()`
- [ ] Gate check passes: `go test -count=1 ./cmd/mem/...`

**Tests**: unit
**Gate**: quick

**Commit**: `docs(cli): document the up/down/logs/profiles/events commands and sync help`

---

### T15 ✅ Rewrite `.specs/STATE.md`

**What**: Remove the Electron narrative ("Shipped v2.0.0", "PR #7 draft", "Phase 2"), de-duplicate the two contradictory ADR-035 entries, and reconcile ISSUE-001/009 status with the metrics table.
**Where**: `.specs/STATE.md`
**Depends on**: T14
**Reuses**: `docs/adr/README.md` obsolete marking
**Requirement**: CLEAN-24, CLEAN-25

**Tools**:

- MCP: NONE
- Skill: `my-memory-format`

**Done when**:

- [ ] No line describes the Electron viewer as shipped, in draft, or as an active decision
- [ ] ADR-035 appears exactly once with a status matching `035-*.md`
- [ ] ISSUE-001 and ISSUE-009 no longer contradict `.specs/ISSUES.md`
- [ ] `grep -rn "electron\|Electron" .specs/STATE.md` returns zero hits

**Tests**: none
**Gate**: build

**Commit**: `docs(specs): drop the reverted Electron viewer narrative from STATE.md`

---

### T16 ✅ Repair the ADR index and status headers

**What**: Add the nine unindexed ADRs (035, 036, 039, 040, 041, 042, 043, 044, 050) to `docs/adr/README.md` and `docs/README.md`; flip 045 to Deferred and 035 to Accepted in both the index and the file headers.
**Where**: `docs/adr/README.md`
**Depends on**: T15
**Reuses**: the status callouts already present in `045-*.md`
**Requirement**: CLEAN-26, CLEAN-27, CLEAN-28

**Tools**:

- MCP: NONE
- Skill: `my-memory-format`

**Done when**:

- [ ] All nine ADRs appear in the index with a true status
- [ ] `045-*.md` is indexed as Deferred
- [ ] `035-*.md` declares Accepted and carries a `## Status` or `- **Status**:` header
- [ ] ADR-040 and ADR-041 carry a status header in the same convention
- [ ] Obsolete ADRs 037/038/048/049/051/053 remain listed as obsolete

**Tests**: none
**Gate**: build

**Commit**: `docs(adr): index the nine missing ADRs and correct 035/045 status`

---

### T17 ✅ Reconcile ISSUES.md and CHANGELOG.md

**What**: Add the four missing metric rows, close ISSUE-007 as moot, and write the 96 commits since `v2.0.0` into `[Unreleased]`.
**Where**: `.specs/ISSUES.md`
**Depends on**: T16
**Reuses**: `git log v2.0.0..HEAD`
**Requirement**: CLEAN-29, CLEAN-30

**Tools**:

- MCP: NONE
- Skill: `my-memory-format`

**Done when**:

- [ ] ISSUE-012, ISSUE-013, ISSUE-015 and ISSUE-017 each have a metrics row
- [ ] ISSUE-007 is marked moot with the Playwright-removal reason
- [ ] `[Unreleased]` documents `memory_reconcile`, the Ortografia section, the 768→384 migration and the new commands
- [ ] `CHANGELOG.md` covers the range `v2.0.0..HEAD`

**Tests**: none
**Gate**: build

**Commit**: `docs(changelog): record the post-2.0.0 surface and close obsolete issues`

---

### T18 ✅ Make the public skill canonical and reduce the repo skill to a stub

**What**: Commit `skills/mymemory/` and replace `.agents/skills/my-memory/SKILL.md` with a short pointer that carries no duplicated CLI or MCP reference content.
**Where**: `.agents/skills/my-memory/SKILL.md`
**Depends on**: T17
**Reuses**: `skills/mymemory/SKILL.md` and its `references/`
**Requirement**: CLEAN-47, CLEAN-48, CLEAN-49, CLEAN-50

**Tools**:

- MCP: NONE
- Skill: `my-memory-format`

**Done when**:

- [ ] `skills/mymemory/` is tracked and absent from `git status` as untracked
- [ ] `.agents/skills/my-memory/SKILL.md` is under 60 lines and contains no command or tool reference tables
- [ ] The stub names the canonical skill by an explicit relative path
- [ ] The stub still parses as a skill an agent can load from `.agents/skills/`
- [ ] No skill file presents Ollama as a provider, prerequisite or fallback

**Tests**: none
**Gate**: build

**Commit**: `docs(skill): make skills/mymemory canonical and reduce the repo skill to a pointer`

---

### T19 ✅ Archive viewer-era artifacts and remove working-tree junk

**What**: Move the four viewer-era `.specs/` documents into `.specs/_archive/` and delete `config-global.txt`, `config-local.txt`, `mem.exe` and `bin/scratch_ops/faults.py`.
**Where**: `.specs/_archive/`
**Depends on**: T18
**Reuses**: `git mv` so history follows the files
**Requirement**: CLEAN-42, CLEAN-43, CLEAN-44, CLEAN-45

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] The four artifacts live under `.specs/_archive/` and no longer appear in the active `.specs/` index
- [ ] No active document links to `docs/VIEWER.md`
- [ ] `Test-Path` returns false for `config-global.txt`, `config-local.txt`, `mem.exe` and `bin/scratch_ops/faults.py`
- [ ] The deletion used recoverable removal, not permanent deletion

**Tests**: none
**Gate**: build

**Commit**: `chore(specs): archive viewer-era artifacts and clear working-tree junk`

---

### T20 ✅ Register the idle-daemon gap and run the closing gate

**What**: Add an open issue describing that `mem up` starts a daemon whose `Run` blocks on context cancellation without starting workers, then run the full quality gate.
**Where**: `.specs/ISSUES.md`
**Depends on**: T19
**Reuses**: `cmd/mymemoryd/main.go` and `internal/supervisor/manager.go` as cited evidence
**Requirement**: CLEAN-51, CLEAN-52

**Tools**:

- MCP: NONE
- Skill: `my-memory-format`

**Done when**:

- [ ] The issue is open and cites both `cmd/mymemoryd/main.go` and `internal/supervisor/manager.go`
- [ ] It states that the supervisor lifecycle is unreachable from production and that tests still cover it
- [ ] `go build ./...`, `go vet ./...` and `go test -count=1 ./...` all exit 0
- [ ] `gofmt -l .` prints nothing
- [ ] `git status --porcelain` shows no untracked file
- [ ] Every commit message passes `check_commit.py`

**Tests**: none
**Gate**: build

**Commit**: `docs(issues): register the unreachable supervisor lifecycle as a known gap`

---

### T21 ✅ Remove the Ollama defaults still baked into `internal/config`

**What**: `internal/config/config.go` still declares Ollama as the embedding default in three places, even though `resolveEmbedder` is authoritative on the builtin provider. Found by the T11–T17 doc sweep.
- line ~63: struct comment `// "ollama" (default)`
- line ~64: struct comment `// "nomic-embed-text" (default)`
- lines ~131–132: `DefaultConfig()` sets `Provider: "ollama"` and `Model: "nomic-embed-text"`
- lines ~275–280: after loading a config, empty fields are filled with `Model = "nomic-embed-text"` and `URL = "http://localhost:11434"` (`Dimension = 384` on line ~282 is already correct — leave it)

**Where**: `internal/config/config.go`
**Depends on**: T20
**Reuses**: n/a
**Requirement**: CLEAN-15, CLEAN-16

Align these with the builtin reality. Verify every consumer of `cfg.Embedding.Provider` / `.Model` / `.URL` before changing the default — grep for them and report what actually reads each field. If a field has no remaining consumer, say so rather than keeping it for symmetry.

**Tests**: unit — assert `DefaultConfig()` no longer reports the Ollama provider, and that a config missing the model/URL does not get them backfilled with Ollama values.

**Gate**: `go build ./... && go test -count=1 ./internal/config/... ./cmd/mem/...`

**Commit**: `fix(config): drop the ollama defaults from the embedding config struct`

---

### T22 ✅ Fix user-facing messages that still blame Ollama

**What**: `cmd/mem/main.go` prints six user-facing messages that blame Ollama although the provider is builtin. Found by the T11–T17 doc sweep. Lines ~1629, ~1819, ~1889, ~1897, ~2019, ~2034:
- `"Aviso: Ollama indisponível (%s); indexando em modo léxico FTS"`
- `"Erro ao gerar embedding da busca (verifique se o Ollama está rodando): %v"`
- `"Aviso: Ollama offline ou falha ao gerar embedding (%v). Executando fallback para busca textual FTS."`

Rewrite them to describe the actual failure (the embedder failed) without naming a provider that is no longer in the path. Do NOT touch the explanatory comments at lines ~954, ~1377 and ~1379 — those correctly document why Ollama was removed.

Also fix the same class of error in `docs/BENCHMARKS.md` (~line 38): it claims 384-dim 4-bit packs to "384 bytes, 87,5% de economia". `internal/turboquant/quantizer.go:59` computes `numBytes := (dim + 1) / 2`, so 384 dims pack to **196 bytes** (192 packed + 4 scale), i.e. ~87.2%. Correct the figures.

**Where**: `cmd/mem/main.go`
**Depends on**: T21
**Reuses**: n/a
**Requirement**: CLEAN-12, CLEAN-13, CLEAN-18

**Tests**: none (user-facing strings). Verify by reading the diff.

**Gate**: `go build ./... && go test -count=1 ./cmd/mem/...`

**Commit**: `fix(cli): stop blaming ollama for embedder failures and correct the TurboQuant byte figure`

---

### T23 ✅ Fix the Ollama/768 default in the federation central-vault bootstrap

**What**: Found by the T21/T22 sweeps — `internal/federation/bootstrap.go` (~lines 312–316) still constructs `EmbeddingConfig{Provider: "ollama", Model: "nomic-embed-text", URL: "http://localhost:11434", Dimension: 768}` when generating the central vault config. This path is live: `cmd/mem/setup.go:172` calls `federation.BootstrapCentralVault` during `mem setup`.

**Where**: `internal/federation/bootstrap.go`
**Depends on**: T22
**Reuses**: the pattern already applied at `cmd/mem/main.go:956` and `internal/config/config.go:130`
**Requirement**: CLEAN-10, CLEAN-11, CLEAN-14, CLEAN-15

Align to builtin / `paraphrase-multilingual-MiniLM-L12-v2` / 384, omitting the Ollama URL. The real constants are `embedder.ModelID` (`internal/embedder/embedder.go:32`) and `embedder.EmbeddingDim` (line 28) — read them, do not assume.

**Tests**: unit — `TestBootstrapCentralVault_EmbeddingDefaults` asserting the generated config declares builtin/384 and contains no `nomic-embed-text`, `11434` or `dimension: 768`.

**Gate**: `go build ./... && go vet ./... && go test -count=1 ./internal/federation/...`

**Commit**: `fix(federation): stop writing ollama/768 into the generated central vault config`

---

## Phase Execution Map

```
Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6

Phase 1:  T1 → T2 → T3 → T4 → T5 → T6
Phase 2:                T6 → T7 → T8
Phase 3:                            T8 → T9 → T10
Phase 4:                                        T10 → T11 → T12 → T13 → T14
Phase 5:                                                                T14 → T15 → T16 → T17
Phase 6:                                                                            T17 → T18 → T19 → T20
```

Execution is strictly sequential - there is no intra-phase parallelism. A single agent (or batch worker) works one task at a time, in order.

**Batch packing** (~7 tasks per worker, cuts only at phase boundaries):

| Batch | Phases | Tasks | Focus |
| ----- | ------ | ----- | ----- |
| 1 | Phase 1 | T1–T6 | Pure deletion; ends green or the repo is broken |
| 2 | Phase 2 + 3 | T7–T10 | Behaviour fix + infra gates |
| 3 | Phase 4 + 5 | T11–T17 | Documentation and project-state realignment |
| 4 | Phase 6 | T18–T20 | Skill consolidation and final cleanup |

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: root `main.go` | 1 file | ✅ Granular |
| T2: `internal/writer/` | 1 package dir | ✅ Granular |
| T3: `internal/policy/` | 1 package dir | ✅ Granular |
| T4: `internal/asr/` + `internal/ipc/` | 2 sibling dead packages, one ADR cluster | ⚠️ Cohesive — both are orphaned transport/config shells |
| T5: `cmd/mem/replay.go` | 1 file + its test | ✅ Granular |
| T6: `go mod tidy` + gate | 1 manifest | ✅ Granular |
| T7: init template | 1 file + tests | ✅ Granular |
| T8: vault config | 1 file | ✅ Granular |
| T9: `.gitattributes` | 1 file | ✅ Granular |
| T10: CI workflow | 1 file | ✅ Granular |
| T11–T13: docs | 1–3 cohesive docs per task, same root cause | ⚠️ Cohesive |
| T14: CLI guide + `printHelp()` | doc + one code file, same requirement | ⚠️ Cohesive |
| T15–T17: state artifacts | 1 file each | ✅ Granular |
| T18: skill consolidation | 2 files, one canonical source | ✅ Granular |
| T19: archive + junk | 1 dir + untracked files | ✅ Granular |
| T20: issue + gate | 1 file | ✅ Granular |

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
| T15 | T14 | T14 → T15 | ✅ Match |
| T16 | T15 | T15 → T16 | ✅ Match |
| T17 | T16 | T16 → T17 | ✅ Match |
| T18 | T17 | T17 → T18 | ✅ Match |
| T19 | T18 | T18 → T19 | ✅ Match |
| T20 | T19 | T19 → T20 | ✅ Match |

No task depends on a later phase. ✅

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1–T6 | deletion only | none | none | ✅ OK |
| T7 | `cmd/mem` config template | unit | unit | ✅ OK |
| T8 | config file | none | none | ✅ OK |
| T9 | repo config | integration | none (shell-verified) | ✅ OK |
| T10 | CI config | none | none | ✅ OK |
| T11–T13 | Markdown | none | none | ✅ OK |
| T14 | `cmd/mem` dispatch/help | unit | unit | ✅ OK |
| T15–T17 | Markdown state artifacts | none | none | ✅ OK |
| T18–T20 | skill/docs/archive | none | none | ✅ OK |

Both Go layers that the matrix marks `unit` (T7, T14) carry their tests inside the same task. ✅

---

## MCPs and Skills

- **MCP**: `github` (T10, for workflow inspection), NONE elsewhere
- **Skills**: `my-memory-format` (all Markdown writes), `tlc-spec-driven` (this flow), NONE otherwise