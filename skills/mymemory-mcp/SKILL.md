---
name: mymemory-mcp
description: Configure and use the MyMemory MCP server, which exposes the 26 `memory_*` tools for hybrid search, graph traversal, note authoring, and drift detection to any MCP-capable agent. Use when wiring an MCP client to the vault, choosing between an MCP tool and shelling out to the `mem` CLI, or mapping an intent such as "search the vault", "write a note", "what breaks if I change this note" onto a `memory_*` tool.
---

# MyMemory MCP Server

`mem mcp` starts a Model Context Protocol server over the same index the CLI uses. It exposes
**26 tools** to the agent so retrieval and note authoring happen as structured tool calls instead
of shell output that has to be re-parsed.

## Configure a client

The server speaks **stdio by default** — the client spawns `mem` as a child process and talks to
it over stdin/stdout.

```json
{
  "mcpServers": {
    "my-memory": {
      "command": "mem",
      "args": ["mcp", "--storage", "postgres", "--repo", "repo_central"]
    }
  }
}
```

`mem install --target <claude|cursor|vscode|windsurf|all>` writes this configuration for you.

> [!warning]
> **Always pass `--storage` and `--repo` when the agent should read a specific vault.** Without
> them the server resolves the vault from the working directory, so an agent running inside an
> unrelated repository silently reads *that* repository's memory instead of the intended one.
> There is no error — just the wrong content.

If `mem` is not on `PATH`, use the absolute path to the binary. A client that cannot spawn the
executable usually fails silently, with nothing visible in the editor.

HTTP/SSE is available when a long-lived shared server is wanted instead of a child process:

```bash
mem mcp --port 38400             # HTTP/SSE
mem mcp --http 0.0.0.0:38400     # network-exposed
```

The same HTTP transport also **serves the metrics dashboard and the graph
artefacts** (ADR-059, ADR-060). The viewer build is published under
`<repo>/.mymemory/www/app/` and answers on `/`; generated graph artefacts
answer on `/graph/`. Run `make viewer` to build and publish it.

> [!warning]
> **The HTTP transport has no authentication.** The default bind is loopback
> (`127.0.0.1`), but `--http 0.0.0.0:38400` exposes it to the network, and
> anything that can reach the port can read the served root — including the
> graph artefacts, whose titles come from vault notes. This gap predates the
> viewer (ADR-021) and is tracked in ADR-050. Put a reverse proxy with
> authentication in front of it before exposing it anywhere.

> [!important]
> **The connection URL and the credential are separate things.** The Postgres URL in
> `storage.postgres_url` is not a secret — it must be readable and shareable. The password is a
> secret and belongs in the `MY_MEMORY_PG_PASSWORD` environment variable, never on disk, because
> MCP clients spawn the server with only the `env` block of their own config. After changing an
> environment variable, restart the client: a server already running keeps its inherited
> environment and keeps failing to authenticate. Full layout in `mymemory-config`.

## When to use a tool instead of the CLI

Prefer the MCP tool when the agent is doing the work. Shell to `mem` when a human is at the
prompt, when the output is meant to be read directly (a graph view, a benchmark, an HTML export),
or when the operation has no tool equivalent — `mem install`, `mem watch`, `mem up`/`mem down`,
`mem events` have no MCP counterpart.

| Situation | Reach for |
| --- | --- |
| The agent is retrieving, writing, or reasoning about vault content | `memory_*` tool |
| The output is for a human to look at | `mem` CLI |
| Setup, wiring, or process supervision | `mem` CLI |
| Code symbols rather than notes | `memory_code_*` tools |

## Intent → tool

| Intent | Tool |
| --- | --- |
| Find notes about a topic | `memory_search` (`detail_level` `l0` → `l1` → `l2`; `tag` filters by Obsidian tag) |
| List the vault's tags | `memory_get_tags` (with `path` for one note) |
| Which notes carry a tag | `memory_get_tag` (matches the tag **and** its descendants) |
| Find functions, methods, classes | `memory_code_search` |
| What links to / from this note | `memory_get_neighbors`, `memory_inspect_node` |
| Code dependency neighbourhood | `memory_code_neighbors` |
| What is most authoritative / central here | `memory_get_hubs` |
| What does this vault cluster into | `memory_get_clusters` |
| Which notes should probably be linked | `memory_get_insights` |
| What breaks if I change this note or symbol | `memory_get_impact` |
| How do I get from note A to note B | `memory_find_path` |
| Load a coherent subgraph under a token budget | `memory_pack_context` |
| Write a finding back into the vault | `memory_write_note` |
| Add a section to an existing note | `memory_append_section` |
| Consolidate many sources into one note | `memory_compile_note` |
| Export a subgraph to Obsidian | `memory_export_canvas` |
| Open a note or emit a deep link | `memory_open_node` |
| Check graph health after writes | `memory_doctor` |
| Check whether docs fell behind code | `memory_get_drift` |
| Reconcile filesystem against the index | `memory_reconcile` |
| Produce a visual graph | `memory_visualize_graph` |

## Two behaviours that change the cost of a conversation

**`detail_level` is the cost lever.** `l0` returns micro-abstracts with **zero file reads**.
Start there, escalate only when the broad layer misses.

**`memory_compile_note` replaces re-reading.** Retrieve once, compile once, then read only the
compiled note on later interactions instead of re-querying N sources.

## Two layers in one response

The twelve read tools return **both** a human-readable `content` block and a
machine-readable **`structuredContent`** — the field the MCP spec (2025-06-18)
defines and this server now fills (ADR-060).

| Layer | Who reads it | Shape |
| --- | --- | --- |
| `content[].text` | the human, the LLM | Markdown, formatted for reading |
| `structuredContent` | a client that parses JSON | the handler's own data, as JSON |

**Prefer `structuredContent` when the client can parse JSON.** Scraping a number
out of a Markdown table is brittle: the formatting is for humans and changes for
readability, not for machine stability. The shape of `structuredContent` is the
**JSON tags of the Go type** the handler already had — that type is the source
of truth, not the documentation.

Tools that fill it: `memory_doctor`, `memory_get_hubs`, `memory_get_clusters`,
`memory_get_insights`, `memory_get_drift`, `memory_list_files`,
`memory_read_note`, `memory_note_outline`, `memory_file_metadata`,
`memory_visualize_graph`, `memory_get_tags` and `memory_get_tag`.

The field is **omitted** when a tool does not fill it, so the JSON of the other
fourteen tools is byte-identical to what it was before.

> [!tip]
> **A response without `structuredContent` is not an error.** A client written
> against an older server still gets the text. Treat the structured payload as an
> upgrade, and keep the text fallback.

## A URL, not just a path

`memory_visualize_graph` returns the **served URL** of the artefact alongside the
filesystem path — `http://127.0.0.1:38400/graph/<file>.html` when the HTTP
transport is running. A path is not actionable for an agent that runs on another
machine, in a container, or behind a different filesystem; a URL is.

Without the HTTP transport (the `stdio` default), the result carries **only the
path, and no error** — the transport simply is not there.

## Reference

- [mcp-tools.md](references/mcp-tools.md) — all 26 tools: argument tables for the retrieval
  and vault tools, a summary table for the graph tools, plus suggested flows and gotchas

## Gotchas

1. **Silent wrong vault** — no `--storage` / `--repo` means the working directory decides.
2. **`doctor` is cheap** — run it after any batch of writes.
3. **`drift` is a release gate** — it catches documentation that fell behind code.
4. **Argument naming differs from the CLI** — MCP uses `detail_level`; the CLI uses `--level`.
5. **The HTTP transport is unauthenticated** — loopback by default, but `--http 0.0.0.0:...`
   puts the served root on the network, graph artefact titles included.
6. **`structuredContent` is optional** — a tool that does not fill it omits the field
   entirely; keep the text fallback for clients that talk to an older server.
7. **A binary without an embedding provider still serves** — a server built without
   `-tags onnx` has no vector layer. `memory_search` with `mode=vector` then answers with a
   JSON-RPC error (`-32603`) and the server keeps running; `mode=hybrid` falls back to the
   lexical FTS search, so the call succeeds with weaker results.