# MCP Server Tasks

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Spec**: `.specs/features/mcp-server/spec.md`
**Status**: Approved

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec - confirm before Execute. Guidelines found: `AGENTS.md`, `.github/workflows/ci.yml`.

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Protocol / RPC | unit | All frame types; parse errors; valid serialization | `internal/mcp/*_test.go` | `go test -v ./internal/mcp/...` |
| Tool Handlers | unit | Happy paths for search & neighbors; missing args; unknown tool | `internal/mcp/*_test.go` | `go test -v ./internal/mcp/...` |
| CLI Wiring | none | - (build gate only) | `cmd/mem/main.go` | `go build -o bin/mem.exe ./cmd/mem` |

## Gate Check Commands

> Generated from codebase - confirm before Execute.

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After tasks with unit tests only | `go test -v ./internal/mcp/...` |
| Full | After tasks with full test suite | `go test -v ./internal/...` |
| Build | After CLI command wiring or phase completion | `go test -v ./internal/... && go build -o bin/mem.exe ./cmd/mem` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: Protocol & Core Server

Base JSON-RPC 2.0 framing and standard MCP handshake.

```
T1 -> T2
```

### Phase 2: Tools & Handlers

Tool registration, JSON Schema definitions and handler execution.

```
T2 -> T3 -> T4
```

### Phase 3: CLI Integration

Exposing the `mem mcp` command in the CLI.

```
T4 -> T5
```

---

## Task Breakdown

### T1: Implement JSON-RPC 2.0 Protocol Framing

**What**: Define request, response and error types for JSON-RPC 2.0 with reader/writer on io.Reader/io.Writer
**Where**: `internal/mcp/protocol.go`
**Depends on**: None
**Reuses**: Standard library `encoding/json`
**Requirement**: MCP-01, MCP-02

**Tools**:

- MCP: `filesystem`
- Skill: NONE

**Done when**:

- [x] JSON-RPC 2.0 Request, Response, and Error structs defined
- [x] ReadMessage and WriteMessage helpers implemented
- [x] Unit tests cover valid frames and malformed JSON
- [x] Quick gate passes: `go test -v ./internal/mcp/...`

**Tests**: unit
**Gate**: quick
**Commit**: `feat(mcp): implement JSON-RPC 2.0 protocol types and framing`

---

### T2: Implement Server Lifecycle and Handshake

**What**: Implement MCPServer running stdio loop with initialize and ping handlers
**Where**: `internal/mcp/server.go`
**Depends on**: T1
**Reuses**: `internal/mcp/protocol.go`
**Requirement**: MCP-01, MCP-02

**Tools**:

- MCP: `filesystem`
- Skill: NONE

**Done when**:

- [x] Server handles `initialize` returning protocol version and capabilities
- [x] Server handles `notifications/initialized` and `ping`
- [x] Diagnostic logs routed to stderr
- [x] Unit tests cover handshake flow
- [x] Quick gate passes: `go test -v ./internal/mcp/...`

**Tests**: unit
**Gate**: quick
**Commit**: `feat(mcp): implement server lifecycle and initialize handshake`

---

### T3: Implement Tool Registration and Schema Listing

**What**: Register memory_search and memory_get_neighbors schemas and handle tools/list
**Where**: `internal/mcp/tools.go`
**Depends on**: T2
**Reuses**: `internal/mcp/server.go`
**Requirement**: MCP-03

**Tools**:

- MCP: `filesystem`
- Skill: NONE

**Done when**:

- [x] Tool definition with JSON Schema input definition
- [x] `tools/list` returns memory_search and memory_get_neighbors
- [x] Unit tests verify tool schema definitions
- [x] Quick gate passes: `go test -v ./internal/mcp/...`

**Tests**: unit
**Gate**: quick
**Commit**: `feat(mcp): implement tool registration and tools/list endpoint`

---

### T4: Implement Tool Call Dispatcher with SQLite Handlers

**What**: Dispatch tools/call executing search and graph neighbor expansion against SQLite
**Where**: `internal/mcp/handlers.go`
**Depends on**: T3
**Reuses**: `internal/db/store.go`, `internal/db/graph.go`
**Requirement**: MCP-04, MCP-05

**Tools**:

- MCP: `filesystem`
- Skill: NONE

**Done when**:

- [x] Handler executes vector search and returns formatted content
- [x] Handler executes graph traversal and returns neighbors
- [x] Unit tests verify tools/call dispatching and error responses
- [x] Full gate passes: `go test -v ./internal/...`

**Tests**: unit
**Gate**: full
**Commit**: `feat(mcp): implement tool call execution for search and graph`

---

### T5: Wire mem mcp Command to CLI

**What**: Add mcp subcommand to CLI connecting stdio server with database
**Where**: `cmd/mem/main.go`
**Depends on**: T4
**Reuses**: `internal/mcp/server.go`
**Requirement**: MCP-01, MCP-04

**Tools**:

- MCP: `filesystem`
- Skill: NONE

**Done when**:

- [x] `mem mcp [--db path]` subcommand starts server
- [x] Stdio correctly pipes stdin/stdout while logs use stderr
- [x] Build gate passes: `go test -v ./internal/... && go build -o bin/mem.exe ./cmd/mem`

**Tests**: none
**Gate**: build
**Commit**: `feat(cli): add mem mcp command for AI agent integration`

---

## Pre-Approval Validation Checks

### Check 1: Task Granularity

| Task | File(s) Changed | Single Deliverable? | Status |
| ---- | --------------- | ------------------- | ------ |
| T1 | `internal/mcp/protocol.go` | Yes (RPC framing) | âœ… Pass |
| T2 | `internal/mcp/server.go` | Yes (Server loop) | âœ… Pass |
| T3 | `internal/mcp/tools.go` | Yes (Tool schemas) | âœ… Pass |
| T4 | `internal/mcp/handlers.go` | Yes (Tool execution) | âœ… Pass |
| T5 | `cmd/mem/main.go` | Yes (CLI command) | âœ… Pass |

### Check 2: Diagram-Definition Cross-Check

| Task | Diagram Depends On | Task Definition Depends On | Matches? |
| ---- | ------------------ | -------------------------- | -------- |
| T1 | None | None | âœ… Yes |
| T2 | T1 | T1 | âœ… Yes |
| T3 | T2 | T2 | âœ… Yes |
| T4 | T3 | T3 | âœ… Yes |
| T5 | T4 | T4 | âœ… Yes |

### Check 3: Test Co-location Validation

| Task | Code Layer | Required Test Type | Tests Field | Matches Matrix? |
| ---- | ---------- | ------------------ | ----------- | --------------- |
| T1 | Protocol / RPC | unit | unit | âœ… Yes |
| T2 | Protocol / RPC | unit | unit | âœ… Yes |
| T3 | Tool Schemas | unit | unit | âœ… Yes |
| T4 | Tool Handlers | unit | unit | âœ… Yes |
| T5 | CLI Wiring | none | none | âœ… Yes |
