---
name: mymemory-mcp
description: Configure and use the MyMemory MCP server, which exposes the 24 `memory_*` tools for hybrid search, graph traversal, note authoring, and drift detection to any MCP-capable agent. Use when wiring an MCP client to the vault, choosing between an MCP tool and shelling out to the `mem` CLI, or mapping an intent such as "search the vault", "write a note", "what breaks if I change this note" onto a `memory_*` tool.
---

# MyMemory MCP Server

`mem mcp` starts a Model Context Protocol server over the same index the CLI uses. It exposes
**24 tools** to the agent so retrieval and note authoring happen as structured tool calls instead
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
| Find notes about a topic | `memory_search` (`detail_level` `l0` → `l1` → `l2`) |
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

## Reference

- [mcp-tools.md](references/mcp-tools.md) — all 24 tools with arguments, defaults, suggested flows
  and gotchas

## Gotchas

1. **Silent wrong vault** — no `--storage` / `--repo` means the working directory decides.
2. **`doctor` is cheap** — run it after any batch of writes.
3. **`drift` is a release gate** — it catches documentation that fell behind code.
4. **Argument naming differs from the CLI** — MCP uses `detail_level`; the CLI uses `--level`.