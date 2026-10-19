# Servir artefatos pelo MCP HTTP — Tasks

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Spec**: `.specs/features/mcp-static-serving/spec.md`
**ADR**: `docs/adr/059-spa-react-vite-em-viewer-servida-pelo-mcp.md` (emendada por `docs/adr/060-stack-do-frontend-da-viewer-e-fallback-spa.md`)
**Status**: Approved

---

## Test Coverage Matrix

> Generated from codebase sampling, project guidelines, and spec. Guidelines found: `AGENTS.md` (Princípio 4 - testes obrigatórios; Princípio 8 - quality gate), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`, `CONTRIBUTING.md`, `.github/workflows/ci.yml`.

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Raiz servida e barreira de path (`internal/mcp/static_server.go`) | unit | arquivo existente servido com 200; escape da raiz recusado com 404 e **sem** o entrypoint; `GET /` sem build responde 503; `/graph/` inexistente responde 404 e nunca o entrypoint; asset faltando sem `Accept: text/html` responde 404 | `internal/mcp/static_server_test.go` | `go test -count=1 ./internal/mcp/...` |
| Roteamento do transporte HTTP (`internal/mcp/http_server.go`) | unit | as 4 rotas do protocolo continuam respondendo; as rotas estáticas herdam CORS; o endereço vinculado é publicado com a porta real | `internal/mcp/http_server_test.go` | `go test -count=1 ./internal/mcp/...` |
| URL do artefato na tool (`internal/mcp/visualizer_handlers.go`) | unit | com transporte ativo a resposta traz URL **e** caminho; sem transporte, só o caminho e nenhum erro; a URL usa a porta real do listener | `internal/mcp/visualizer_handlers_test.go` | `go test -count=1 ./internal/mcp/...` |
| `structuredContent` das tools de leitura (`internal/mcp/*_handlers.go`) | unit | o payload estruturado é emitido ao lado do mesmo texto; tools sem payload não emitem a chave; o JSON do payload é estável (string esperada) | `internal/mcp/structured_content_test.go` | `go test -count=1 ./internal/mcp/...` |
| CLI do servidor MCP (`cmd/mem/main.go`) | integration | `mem mcp --http` sobe e responde em `/health`; a raiz servida é a mesma usada pelos artefatos de grafo | `cmd/mem/mcp_server_test.go` | `go test -count=1 ./cmd/mem/...` |
| Documentação (ADRs) | none | no Go layer; verified by the grep assertions recorded per task | n/a | task-specific grep |

**Coverage Expectation note**: every code-bearing layer above has an automated test. Documentation has none because it declares no runtime behaviour; its criteria are verified by the grep assertions recorded per task.

## Gate Check Commands

> Generated from the repo's own `ci.yml` and `AGENTS.md`, verified locally.

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After tasks touching `internal/mcp` or `cmd/mem` only | `go build ./... && go test -count=1 ./internal/mcp/... ./cmd/mem/...` |
| Full | After any task touching `internal/db`, `internal/store` or `internal/watcher` | `go test -count=1 ./...` |
| Build | After every phase | `go build ./... && go vet ./... && go test -count=1 ./...` |
| Format | Only valid in a fresh checkout | `gofmt -l internal/mcp cmd/mem` prints the pre-existing CRLF files in this working tree — that is the pre-existing condition, not a defect. Prove format cleanliness in a throwaway worktree, never by running `gofmt -w .` |
| No-Node | After every phase | `go build ./... && go vet ./... && go test -count=1 ./...` must pass without `viewer/node_modules` installed (SERV-19) |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: Raiz servida e segurança de path

```
T1 → T2 → T3
```

### Phase 2: Rotas e fallback SPA

```
T3 → T4 → T5
```

### Phase 3: URL na tool e degradação

```
T5 → T6 → T7 → T8 → T9
```

### Phase 4: Headers de segurança, testes e documentação

```
T9 → T10 → T11 → T12
```

### Phase 5: `structuredContent` para o painel

```
T12 → T13 → T14 → T15 → T16 → T17 → T18 → T19
```

---

## Task Breakdown

### T1: Create the dedicated served root and the static handler with the path barrier

**What**: Create the served root `<base>/.mymemory/www/` with the two subareas `app/` (SPA build) and `graph/` (graph artifacts), and the static handler that resolves a requested path inside that root. The path is resolved and normalized segment by segment; anything that escapes the root is refused with 404 and never receives the SPA entrypoint. The pattern follows `internal/compiler/note.go:181-184` (resolve, compare with the root, refuse when it leaves the directory).
**Where**: `internal/mcp/static_server.go`
**Depends on**: None
**Reuses**: the traversal barrier precedent of `internal/compiler/note.go`
**Requirement**: SERV-01, SERV-02, SERV-03, SERV-04

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] The served root is `<base>/.mymemory/www/`, with `app/` and `graph/` under it, and never the vault root
- [ ] A file that exists under the root is served with 200 and its content
- [ ] A path that escapes the root after normalization is refused with 404
- [ ] The refusal happens **before** any fallback: the refused request returns neither the file nor the entrypoint
- [ ] The barrier is implemented in the handler, not delegated to `http.FileServer`/`http.ServeMux`
- [ ] Gate check passes: `go build ./... && go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): serve the viewer and graph artifacts from a dedicated root`

---

### T2: Route static files alongside the four protocol endpoints

**What**: Wire the static handler into the HTTP transport without touching the four protocol routes (`/health`, `/sse`, `/message`, `/mcp`). The static paths must reach the handler with the raw request path, because `http.ServeMux` canonicalizes paths containing `..` and would redirect (301) the traversal attempt instead of letting the barrier refuse it.
**Where**: `internal/mcp/http_server.go`
**Depends on**: T1
**Reuses**: the existing `http.ServeMux` and the `withCORS` wrapper, which the static routes inherit
**Requirement**: SERV-05, SERV-06, SERV-19, SERV-21

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `/health`, `/sse`, `/message` and `/mcp` keep answering exactly as before
- [ ] Every other path is handled by the static handler
- [ ] The static routes inherit `Access-Control-Allow-Origin` from the existing CORS wrapper
- [ ] The Go build has no dependency on `viewer/node_modules` and no bundler is invoked by the binary
- [ ] `go build ./...`, `go vet ./...` and `go test -count=1 ./...` pass without `node_modules` installed
- [ ] Gate check passes: `go build ./... && go test -count=1 ./internal/mcp/... ./cmd/mem/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): route static files alongside the protocol endpoints`

---

### T3: Cover the path barrier and the served file with unit tests

**What**: Test the two behaviors the surface depends on: a legitimate file under the root is served, and a traversal attempt is refused without returning the file nor the entrypoint.
**Where**: `internal/mcp/static_server_test.go`
**Depends on**: T2
**Reuses**: the reference cases of `internal/compiler/note_test.go:51-60` and `:182-189`
**Requirement**: SERV-07, SERV-08

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] A test asserts 200 and the file content for a file under the root
- [ ] A test asserts 404 for a traversal attempt, and asserts the body is neither the outside file nor the entrypoint
- [ ] The traversal test covers `..`, an absolute path and mixed separators
- [ ] The test does not depend on the vault or on a built SPA
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `test(mcp): cover path traversal refusal and static file serving`

---

### T4: Reserve the `/graph/` route for graph artifacts

**What**: Map `/graph/<arquivo>` to the `graph/` subarea of the served root. A path under `/graph/` that does not match an existing artifact answers 404 and **never** falls back to the SPA entrypoint: the reserved prefix serves artifacts only.
**Where**: `internal/mcp/static_server.go`
**Depends on**: T3
**Reuses**: the served root resolution from T1
**Requirement**: SERV-01, SERV-23

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] An existing artifact under `graph/` is served with 200
- [ ] A missing artifact under `graph/` answers 404
- [ ] The 404 body is not the SPA entrypoint, even when the request accepts `text/html`
- [ ] A path that merely starts with the same letters (`/graphs/...`) is not treated as the reserved prefix
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): reserve the /graph/ route for graph artifacts`

---

### T5: Apply the Accept-discriminated SPA fallback and the degradation responses

**What**: Add the SPA fallback of ADR-060: a path that matches no file answers with `app/index.html` and 200 **only** when the request accepts `text/html`; otherwise it answers 404. Add the degradation of ADR-059 D-6: without `app/index.html`, the root answers 503 with a plain message instead of 404 for every file; without the `graph/` area, the artifact route answers 404 with a plain message.
**Where**: `internal/mcp/static_server.go`
**Depends on**: T4
**Reuses**: the entrypoint resolution and the reserved prefix from T4
**Requirement**: SERV-02, SERV-23, SERV-24, SERV-25

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `GET /clusters` with `Accept: text/html` answers 200 with the entrypoint
- [ ] `GET /assets/nao-existe.js` without `Accept: text/html` answers 404, not HTML
- [ ] The fallback is never reached by a path that escaped the root
- [ ] `GET /` without a built SPA answers 503 with a plain message
- [ ] A missing `graph/` area answers 404 with a plain message
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): serve the SPA fallback only for navigation requests`

---

### T6: Publish the address the transport is actually bound to

**What**: The `HTTPServer` must publish the effective address, resolved from `net.Listener.Addr()` in the `Serve(l)` path, so a request for port `0` or `:38400` yields the concrete port. The `ListenAndServe()` path has no listener, so it is changed to create one and go through the same path, making both publish the same address.
**Where**: `internal/mcp/http_server.go`
**Depends on**: T5
**Reuses**: the existing `net.Listener` argument of `Serve(l)`
**Requirement**: SERV-13

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] After `Serve(l)`, the server exposes the address of the listener, with the concrete port
- [ ] `ListenAndServe()` publishes the same kind of address, resolved from a real listener
- [ ] The published address is `host:port`, never a hardcoded value
- [ ] A wildcard bind is published as a loopback host, so the resulting URL is clickable
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): publish the bound listener address`

---

### T7: Expose the transport base URL to the MCP server

**What**: The MCP `Server` needs to hand the transport address to the tool handlers. It gets a `SetBaseURL(string)` / `BaseURL() string` pair, and the graph view handler reads it through that accessor, so the value is resolved at call time and not at registration time.
**Where**: `internal/mcp/server.go`
**Depends on**: T6
**Reuses**: the existing `SetStalenessDetector` / `StalenessDetector` accessor pattern
**Requirement**: SERV-09, SERV-13

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `Server.SetBaseURL` stores the address and `Server.BaseURL` returns it
- [ ] `BaseURL` returns an empty string when no transport is active
- [ ] The graph view handler registered on the server reads the base URL at call time
- [ ] Concurrent access is safe (the accessors use the server mutex)
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): expose the transport base URL to the MCP server`

---

### T8: Write the graph artifact to the served root and return its URL

**What**: `memory_visualize_graph` must write the artifact inside `<base>/.mymemory/www/graph/`, creating the directory before the write (`graphview.ExportHTML` uses `os.WriteFile` without `os.MkdirAll`), and answer with the served URL **and** the filesystem path when the HTTP transport is active. Without the transport it answers with the path only, without an error. A failure to create the directory becomes a message, never a panic.
**Where**: `internal/mcp/visualizer_handlers.go`
**Depends on**: T7
**Reuses**: the artifact naming rules of the current handler (`graph.html` / `graph_<nota>.html`)
**Requirement**: SERV-09, SERV-10, SERV-11, SERV-12

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] The artifact is written inside `<base>/.mymemory/www/graph/`, and the directory is created when missing
- [ ] With the transport active, the result contains the URL `http://<host>:<port>/graph/<arquivo>.html` and the absolute path
- [ ] Without the transport, the result contains only the path, and no error is raised
- [ ] The URL is built from the address the transport published, never from a hardcoded host or port
- [ ] A failure to create the artifact directory is returned as a tool error
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): write graph artifacts to the served root and return their URL`

---

### T9: Wire the served root and the address publication into the CLI

**What**: The `mem mcp` command must hand the served root to the HTTP transport and use the same base for the graph artifacts, so the artifact directory and the served area are the same tree. The bind stays untouched: loopback by default, `--host`/`--port`/`--http` unchanged, and no new environment variable.
**Where**: `cmd/mem/main.go`
**Depends on**: T8
**Reuses**: the existing `mcp.HTTPServerOptions` and the `SetGraphViewHandler` call sites
**Requirement**: SERV-05, SERV-13

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] The HTTP transport receives the served root derived from the same base the graph artifacts use
- [ ] The base URL published by the transport reaches the graph view handler
- [ ] The default bind remains loopback and no new flag is introduced
- [ ] `mem mcp --http` still answers `/health` with the same payload
- [ ] Gate check passes: `go test -count=1 ./cmd/mem/... ./internal/mcp/...`

**Tests**: integration
**Gate**: quick

**Commit**: `feat(cli): wire the served root into the MCP HTTP transport`

---

### T10: Apply the security header baseline to the served responses

**What**: Apply the baseline registered in ADR-059 D-10 to the served responses: the CSP (`default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'`), `X-Content-Type-Options: nosniff` and `Referrer-Policy: no-referrer`. No remote origin is referenced by the server, and the policy forbids it at runtime.
**Where**: `internal/mcp/static_server.go`
**Depends on**: T9
**Reuses**: the response writer of the static handler
**Requirement**: SERV-20

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] The three headers are present on every served response, including the fallback
- [ ] The CSP is exactly the baseline of ADR-059
- [ ] No remote origin appears in the Go code that serves the responses
- [ ] The headers are set before the body is written
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): apply the CSP baseline to served responses`

---

### T11: Cover the served URL with and without the transport, and the real listener port

**What**: Test the URL contract of `memory_visualize_graph`: URL and path with the transport active, path only without it, and a URL that carries the real port of a listener bound to a free port.
**Where**: `internal/mcp/visualizer_handlers_test.go`
**Depends on**: T10
**Reuses**: the real listener path of `TestHTTPServer_ServeAndShutdown`
**Requirement**: SERV-09, SERV-10, SERV-11, SERV-12, SERV-13

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] A test asserts the URL and the path are both present when the transport is active
- [ ] A test asserts only the path is present, with no error, when the transport is inactive
- [ ] A test binds a free port and asserts the URL carries that concrete port
- [ ] The eight pre-existing tests of `internal/mcp/http_server_test.go` and `cmd/mem/mcp_server_test.go` still pass
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/... ./cmd/mem/...`

**Tests**: unit
**Gate**: quick

**Commit**: `test(mcp): cover the served URL with and without the HTTP transport`

---

### T12: Confirm the exposure and the missing authentication are recorded in the ADR

**What**: The exposure ACs (SERV-14..SERV-18) and the framework-agnostic artifact AC (SERV-22) are satisfied by the text of ADR-059 and ADR-060, which are already in the repository. This task is the verification that each required item is recorded — the change of surface, the absence of authentication and its pre-existing nature, what the exposure means, the widening of the bind being a separate decision, and the served artifact being static output.
**Where**: `docs/adr/059-spa-react-vite-em-viewer-servida-pelo-mcp.md`
**Depends on**: T11
**Reuses**: the sections "Superfície de Ataque e Postura de Segurança" and "Fronteira de escopo" of ADR-059
**Requirement**: SERV-14, SERV-15, SERV-16, SERV-17, SERV-18, SERV-22

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] The ADR states that the transport went from serving no file to serving one directory
- [ ] The ADR states that the transport has no authentication and that this predates the feature
- [ ] The ADR states that only the served root is readable, never the vault root
- [ ] The ADR records that widening the bind address is a separate product decision
- [ ] The ADR records that the served artifact is static output, not a runtime application
- [ ] Gate check passes: `go build ./... && go vet ./... && go test -count=1 ./...`

**Tests**: none
**Gate**: build

**Commit**: `docs(adr): confirmar o registro da exposição do transporte MCP`

> [!note]
> A ADR-059 e a ADR-060 já registram o exigido por estes ACs, e a propriedade dos arquivos de ADR é de outro escritor nesta rodada. A task é de **verificação**: nenhuma alteração de documento é necessária e o commit correspondente fica vazio.

---

### T13: Add `structuredContent` to the tool result, additively

**What**: The MCP spec (2025-06-18) defines `structuredContent` in `CallToolResult`, and this implementation does not have the field. Add it as `StructuredContent any` with `json:"structuredContent,omitempty"`, so the JSON of the tools that do not fill it is unchanged, and add `NewStructuredResult(text, structured)` next to `NewTextResult`, which fills both the human text and the machine payload.
**Where**: `internal/mcp/handlers.go`
**Depends on**: T12
**Reuses**: `NewTextResult`, whose output is preserved byte for byte
**Requirement**: n/a — the `structuredContent` requirement is additional to the `mcp-static-serving` spec and has no `SERV-nn` of its own

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `CallToolResult` has `StructuredContent any` with the tag `json:"structuredContent,omitempty"`
- [ ] `NewTextResult` is unchanged, and the 24 tools keep returning the same textual content
- [ ] `NewStructuredResult(text, structured)` fills the text and the payload
- [ ] A tool that does not fill the payload does not emit the `structuredContent` key in the JSON
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): add structuredContent to tool results`

---

### T14: Fill `structuredContent` in the four vault read tools

**What**: `memory_list_files`, `memory_read_note`, `memory_note_outline` and `memory_file_metadata` must expose the data the handler already has, before formatting: the filtered document list, the read document, the headings, and the metadata. The shape of each is documented in a comment in the handler, because the panel reads the JSON tags of the Go types.
**Where**: `internal/mcp/vault_handlers.go`
**Depends on**: T13
**Reuses**: `store.VaultDocument` and `OutlineHeading`, which already carry JSON tags
**Requirement**: n/a — the `structuredContent` requirement is additional to the `mcp-static-serving` spec and has no `SERV-nn` of its own

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] Each of the four tools returns the payload next to the same text
- [ ] The list tool returns the whole filtered slice
- [ ] The read tool returns path, title and content
- [ ] The outline tool returns the headings
- [ ] The metadata tool returns path, title, size and mtime
- [ ] Each handler carries a comment documenting the shape of its payload
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): expose vault read data as structuredContent`

---

### T15: Fill `structuredContent` in doctor, hubs and insights

**What**: `memory_doctor` exposes the report it already formats, `memory_get_hubs` exposes the hubs slice, or the PageRank slice when the algorithm is `pagerank`, and `memory_get_insights` exposes the connections.
**Where**: `internal/mcp/handlers.go`
**Depends on**: T14
**Reuses**: `DoctorReport`, `GodNode` and `PageRankNode`, which already carry JSON tags
**Requirement**: n/a — the `structuredContent` requirement is additional to the `mcp-static-serving` spec and has no `SERV-nn` of its own

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] The doctor tool returns the report next to the same text
- [ ] The hubs tool returns the hubs slice, and the PageRank slice when the algorithm is `pagerank`
- [ ] The insights tool returns the connections
- [ ] Each handler carries a comment documenting the shape of its payload
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): expose doctor, hubs and insights data as structuredContent`

---

### T16: Fill `structuredContent` in the clusters tool

**What**: `memory_get_clusters` exposes the clusters it already filters and the modularity, reusing `graph.CommunityResult` and its JSON tags instead of a parallel type.
**Where**: `internal/mcp/cluster_handlers.go`
**Depends on**: T15
**Reuses**: `graph.CommunityResult`, which already carries JSON tags
**Requirement**: n/a — the `structuredContent` requirement is additional to the `mcp-static-serving` spec and has no `SERV-nn` of its own

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] The clusters tool returns the modularity, the totals and the filtered clusters
- [ ] The payload reuses `graph.CommunityResult`, with no parallel type
- [ ] The handler carries a comment documenting the shape of the payload
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): expose cluster data as structuredContent`

---

### T17: Fill `structuredContent` in the drift tool

**What**: `memory_get_drift` exposes the drift report it already formats, reusing `drift.DriftReport` and its JSON tags.
**Where**: `internal/mcp/drift_handlers.go`
**Depends on**: T16
**Reuses**: `drift.DriftReport`, which already carries JSON tags
**Requirement**: n/a — the `structuredContent` requirement is additional to the `mcp-static-serving` spec and has no `SERV-nn` of its own

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] The drift tool returns the report next to the same text
- [ ] The payload reuses `drift.DriftReport`, with no parallel type
- [ ] The handler carries a comment documenting the shape of the payload
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): expose drift report as structuredContent`

---

### T18: Fill `structuredContent` in the graph visualization tool

**What**: `memory_visualize_graph` exposes the URL and the path it already returns, so the panel can open the artifact it just generated. The `url` field exists only when the HTTP transport is active.
**Where**: `internal/mcp/visualizer_handlers.go`
**Depends on**: T17
**Reuses**: the URL and the path produced by T8
**Requirement**: SERV-09, SERV-10

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] The payload carries `url` and `path` when the transport is active
- [ ] The payload carries `path` and no `url` when the transport is inactive
- [ ] The payload reuses the metrics already reported in the text
- [ ] The handler carries a comment documenting the shape of the payload
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `feat(mcp): expose the served graph artifact as structuredContent`

---

### T19: Lock the shape of `structuredContent` with tests

**What**: Test the compatibility and the stability of the payload: a tool that fills it returns the payload and keeps the same text; a tool that does not fill it does not emit the key; the JSON of the payload is compared against the expected string, so a tag change is detected as a break.
**Where**: `internal/mcp/structured_content_test.go`
**Depends on**: T18
**Reuses**: the JSON tags of the types reused in T14..T18
**Requirement**: n/a — the `structuredContent` requirement is additional to the `mcp-static-serving` spec and has no `SERV-nn` of its own

**Tools**:

- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] A test asserts the payload and the unchanged text of a tool that fills it
- [ ] A test asserts the `structuredContent` key is absent for a tool that does not fill it
- [ ] A test asserts the JSON of the doctor payload equals the expected string
- [ ] A test asserts the JSON of the hubs payload equals the expected string
- [ ] A test asserts the payload of the graph visualization carries `url` and `path`
- [ ] Gate check passes: `go test -count=1 ./internal/mcp/...`

**Tests**: unit
**Gate**: quick

**Commit**: `test(mcp): lock the structuredContent shape of the read tools`

---

## Phase Execution Map

```
Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5

Phase 1: T1 → T2 → T3
Phase 2:          T3 → T4 → T5
Phase 3:                    T5 → T6 → T7 → T8 → T9
Phase 4:                                          T9 → T10 → T11 → T12
Phase 5:                                                                T12 → T13 → T14 → T15 → T16 → T17 → T18 → T19
```

Execution is strictly sequential - there is no intra-phase parallelism. A single agent (or batch worker) works one task at a time, in order.

**Batch packing** (~5 tasks per worker, cuts only at phase boundaries):

| Batch | Phases | Tasks | Focus |
| ----- | ------ | ----- | ----- |
| 1 | Phase 1 | T1–T3 | Served root, path barrier, first tests |
| 2 | Phase 2 | T4–T5 | Reserved `/graph/` route, Accept-discriminated fallback |
| 3 | Phase 3 | T6–T9 | Published address, URL in the tool, CLI wiring |
| 4 | Phase 4 | T10–T12 | Security headers, URL tests, ADR verification |
| 5 | Phase 5 | T13–T19 | `structuredContent` field, payload per tool, shape tests |

O validador conta as arestas do diagrama como declaradas: 18 setas, 19 tarefas, uma raiz.

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: served root + static handler | 1 file, 1 new type | ✅ Granular |
| T2: transport routing | 1 file | ✅ Granular |
| T3: traversal + file tests | 1 test file | ✅ Granular |
| T4: reserved `/graph/` route | 1 file | ✅ Granular |
| T5: fallback + degradation | 1 file | ✅ Granular |
| T6: published address | 1 file | ✅ Granular |
| T7: base URL on the server | 1 file | ✅ Granular |
| T8: artifact + URL in the tool | 1 file | ✅ Granular |
| T9: CLI wiring | 1 file | ✅ Granular |
| T10: security headers | 1 file | ✅ Granular |
| T11: URL tests | 1 test file | ✅ Granular |
| T12: ADR verification | 1 document, read only | ✅ Granular |
| T13: `structuredContent` field | 1 file, additive | ✅ Granular |
| T14: vault tools payload | 1 file, 4 tools | ⚠️ Cohesive — the four share the same accessors |
| T15: doctor, hubs, insights payload | 1 file, 3 tools | ⚠️ Cohesive — same pattern, same file |
| T16: clusters payload | 1 file | ✅ Granular |
| T17: drift payload | 1 file | ✅ Granular |
| T18: graph visualization payload | 1 file | ✅ Granular |
| T19: payload shape tests | 1 test file | ✅ Granular |

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

No task depends on a later phase. ✅

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1, T4, T5, T10 | static handler | unit | unit | ✅ OK |
| T2, T6 | HTTP transport | unit | unit | ✅ OK |
| T3 | traversal + serving tests | unit | unit | ✅ OK |
| T7 | MCP server accessor | unit | unit | ✅ OK |
| T8, T18 | graph visualization tool | unit | unit | ✅ OK |
| T9 | CLI wiring | integration | integration | ✅ OK |
| T11 | URL tests | unit | unit | ✅ OK |
| T12 | ADR verification | none | none | ✅ OK |
| T13–T18 | `structuredContent` per tool | unit | unit | ✅ OK |
| T19 | payload shape tests | unit | unit | ✅ OK |

Every layer the matrix marks as tested carries its test inside the same task. ✅

---

## MCPs and Skills

- **MCP**: NONE (the feature registers no new tool and consumes none)
- **Skills**: `tlc-spec-driven` (this flow), NONE otherwise

---

## Registro de desvio (a ser preenchido na execução)

_(vazio no momento da escrita deste plano)_
