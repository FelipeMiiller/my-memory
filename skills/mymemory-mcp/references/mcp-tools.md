# MCP Tools Reference

The MyMemory MCP server exposes **20 tools**. Start the server with:

```bash
mem mcp                                # stdio, resolves vault from working directory
mem mcp --storage postgres --repo repo_central   # pin the central vault
mem mcp --port 38400                   # HTTP/SSE
mem mcp --http 0.0.0.0:38400           # expose on the network
```

> [!warning]
> **Always pass `--storage` and `--repo` when an agent should read a specific vault.** Without
> them the server resolves from the working directory, so an agent running inside an unrelated
> repository silently reads that repository's memory instead of the intended one.

Client configuration:

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

If the binary is not on `PATH`, use its absolute path — a client that cannot find the
executable usually fails silently, with no visible error in the editor.

---

## 1. Retrieval

### `memory_search`

Hybrid search with Reciprocal Rank Fusion (RRF), combining exact FTS5/tsvector matches,
k-NN vectors, and graph expansion.

| Argument | Type | Default | Notes |
| --- | --- | --- | --- |
| `query` | string | — | Natural language or exact term. Required. |
| `mode` | enum | `hybrid` | `hybrid` (RRF: FTS + vector + graph) · `vector` (semantic k-NN) · `fts` (exact text only) |
| `limit` | integer | `5` | Maximum results. |
| `k` | integer | `60` | RRF smoothing constant. |
| `repository` | string | — | Slug or name; omit to search the default/global repository. |
| `decay` | boolean | `false` | Exponential temporal decay toward recent notes. |
| `half_life` | number | `30` | Half-life in days. |
| `decay_weight` | number | `0.3` | `0.0` = no effect, `1.0` = maximum decay. |
| `detail_level` | enum | `l1` | `l0` micro-abstract (~30-50 tokens, zero file reads) · `l1` overview with metadata + relevant excerpt · `l2` full detail. |
| `category` | enum | — | `resource` · `memory` · `skill`. |

> [!tip]
> `detail_level` is the cheapest lever you have. `l0` returns micro-abstracts with **zero file
> reads** — try it first and escalate to `l1`/`l2` only when the broad layer misses. Note the
> MCP argument is `detail_level`, while the CLI flag is `--level`.

### `memory_code_search`

Structured lookup in `code_symbols` (function / method / class / interface / struct), with a 2x
RRF boost on exact `qualified_name` matches.

### `memory_get_neighbors`

Neighbouring notes in the relational graph from a `node_id`, via recursive CTE.

### `memory_code_neighbors`

Sub-graph (in/out edges) around a `code_symbol`, up to `depth` hops.

---

## 2. Graph analysis

| Tool | Purpose |
| --- | --- |
| `memory_get_hubs` | Highest-degree nodes — the knowledge hubs. |
| `memory_get_clusters` | Thematic communities via weighted label propagation, plus Newman-Girvan modularity. |
| `memory_get_insights` | High-similarity note pairs with **no** link between them — candidate connections to forge. |
| `memory_get_impact` | Blast radius of reverse dependencies before changing or deleting a note or symbol. |
| `memory_inspect_node` | Three-column triptych: inbound (who depends on it) · the node · outbound (what it references). |
| `memory_find_path` | Shortest weighted route between two nodes, with epistemic certainty as cost. |
| `memory_pack_context` | Connected subgraph around a root note, packaged under a token budget (L2 core, L0/L1 periphery, Mermaid diagram). |

---

## 3. Authoring

### `memory_write_note`

Creates or replaces an atomic Markdown note with structured frontmatter and typed connections,
then triggers immediate surgical indexing.

### `memory_append_section`

Appends a block under an existing heading (or at the end) and reindexes only that region.

### `memory_compile_note`

Synthesises knowledge about a topic from hybrid searches across the repository and writes a
single structured note with backlinks. This is **compile-not-retrieve**: retrieve once, compile
once, then read only the compiled note on later interactions.

> [!tip]
> This is the cheapest way to build durable knowledge. It replaces re-reading N sources per
> question with reading one consolidated note.

### `memory_export_canvas`

Exports a subgraph centred on a node as JSON Canvas 1.0 (`.canvas`), which opens in Obsidian
Canvas.

### `memory_open_node`

Canonical deep links (Obsidian, VS Code, file), optionally opening the editor.

---

## 4. Operations

### `memory_doctor`

Audits graph and note-table integrity: dangling links, orphans, self-loops, and the overall
Health Score.

### `memory_get_drift`

Semantic divergence between recent source-code changes (Git) and the notes/ADRs in memory.
Surfaces stale documentation and orphaned code with no architectural decision attached. Run it
after finishing a feature to catch doc rot before it lands.

### `memory_reconcile`

Reconciles the filesystem against the index without a full reindex. Classifies documents as
present, missing from disk, on disk but unindexed, or unindexed due to config exclusion — and
removes confirmed orphans.

### `memory_visualize_graph`

Exports an interactive standalone HTML view of the graph, with force-directed layout, search,
and PageRank filtering.

---

## 5. Suggested flows

**Research into the vault**

```
memory_search (broad)  →  memory_search (narrow)  →  memory_write_note  →  memory_doctor
```

**Understanding an existing topic**

```
memory_get_hubs  →  memory_inspect_node  →  memory_pack_context
```

**Before a risky change**

```
memory_get_impact  →  make the change  →  memory_get_drift
```

**Building a base over many sources**

```
memory_search per source  →  memory_compile_note (one per topic)  →  memory_doctor
```

---

## 6. Gotchas

1. **Silent wrong vault** — no `--storage` / `--repo` means the working directory decides.
2. **`compile` replaces re-reading** — once a compiled note exists, read that instead of
   re-querying every source.
3. **`doctor` is cheap** — run it after any batch of writes.
4. **`drift` is a release gate** — it catches documentation that fell behind code.
5. **`detail_level` is the cost lever** — `l0` returns summaries with no file reads. Escalate
   to `l1`/`l2` only when the broad layer misses. (MCP: `detail_level`; CLI: `--level`.)
