---
name: mymemory
description: Use the MyMemory CLI and MCP server to build and maintain a semantic knowledge base of Markdown notes - hybrid search (BM25 + vectors + graph), Obsidian wikilinks, and graph analysis. This is the router skill: load it to find which of the six focused skills answers the task. Covers vault setup (mem init / mem setup), indexing (mem index / mem watch / mem hook), hybrid search, graph analysis, note authoring, the 26 MCP tools, and building a knowledge base from many sources. Triggers on "my-memory", "mem search", "mem index", "mem init", "knowledge base", "build a knowledge base", "create a note in the vault", "MCP memory tools", "memory_write_note", "memory_search", "wikilink", "Obsidian vault", "note graph", "dangling link", "orphan note", "central vault", "mem doctor", "mem drift", "mem compile", "compile-not-retrieve", "painel de métricas", "dashboard do vault", "mem mcp --http". DO NOT load for general documentation tools that do not index a note graph.
license: MIT
metadata:
  version: 2.0.0
  package: my-memory
---

# MyMemory — Semantic Knowledge Base over Markdown

Turn a folder of Markdown into a searchable, linked knowledge graph. Hybrid retrieval blends
exact-term search (BM25), semantic vectors, and graph traversal, so a query finds a note whether
you know the exact word, only the meaning, or only the neighbourhood.

```
┌──────────────────────┐         ┌───────────────────────────────┐
│  Markdown vault      │         │  Single index (SQLite/PG)     │
│  *.md  +  wikilinks  │  index  │  ├─ FTS5      exact terms     │
│  #tags  frontmatter  │ ──────> │  ├─ vectors   meaning         │
└──────────────────────┘         │  └─ edges     topology        │
                                 └───────────────────────────────┘
        │                                    │
        ▼                                    ▼
  Obsidian / editor                 CLI (mem)  +  MCP server (26 tools)
```

## Pick the skill that answers the task

This package is split by capability. Load exactly one; each is self-contained.

| Skill | Load it when the task is about… |
| --- | --- |
| [`mymemory-cli`](../mymemory-cli/SKILL.md) | Running `mem` from a shell or script: install, `init`, `index`, `search` (including the `--tag` filter), the tag inventory (`mem tags`, `mem tag`), `watch`, `hook`, flags, exit behaviour. Starts with the 60-second quickstart. |
| [`mymemory-mcp`](../mymemory-mcp/SKILL.md) | An agent is calling the MCP server: wiring a client, choosing between a tool and the CLI, all 26 tools with their inputs and outputs. |
| [`mymemory-notes`](../mymemory-notes/SKILL.md) | Writing or editing a note: frontmatter taxonomy, wikilinks, typed edges, tags, and the `mem note` / `mem compile` write paths. |
| [`mymemory-graph`](../mymemory-graph/SKILL.md) | Understanding the graph or auditing the vault: `inspect`, `path`, `hubs`, `clusters`, `impact`, `drift`, `doctor`, and canvas export. |
| [`mymemory-config`](../mymemory-config/SKILL.md) | Where settings live and which one wins: config precedence, the global/local storage split, SQLite vs Postgres, the central vault. |
| [`mymemory-knowledge-base`](../mymemory-knowledge-base/SKILL.md) | Building a base of knowledge from many sources — not a single note, but a topic answered by many interlinked notes with verified provenance. |

> [!tip]
> Unsure which? `mymemory-cli` covers the widest ground. If the question is *"what does this
> repo already know about X"*, that is a `mymemory-cli` search question, not a knowledge-base
> authoring question.

> [!note]
> The MCP server also serves a **metrics dashboard** (a React/Vite SPA) from
> `<base>/.mymemory/www/`, with no extra process. The routes, the `structuredContent`
> shape and the transport's **lack of authentication** are documented in
> [`mymemory-mcp`](../mymemory-mcp/SKILL.md).

## The one idea that matters

> [!important]
> **The vault is a semantic index, not a text dump.** A note is only reachable if it is
> *indexed*, *linked*, and *described*. Three failure modes sink most knowledge bases:
>
> 1. **Not indexed** — the file is outside `include`, so no query will ever return it.
> 2. **Not linked** — a note with no `[[wikilink]]` in or out is an orphan; it has no
>    neighbourhood, so graph traversal cannot reach it either.
> 3. **Not described** — no `summary` means it cannot be retrieved at level L0, which is
>    the cheap context layer.

Everything in every sub-skill serves those three.

## The loop

```
write notes  →  mem index  →  mem doctor  →  mem search
     ↑                                            │
     └────────────  fix what doctor flags  ←─────┘
```

`mem index` is incremental: a SHA-256 cache skips unchanged files, so re-running it after a
batch of edits is cheap. The index is not a text dump — editing files without reindexing is
the single most common reason a search returns stale or missing results.

## Where things live

| Path | Scope | Committed? |
| --- | --- | --- |
| `~/.memory/config.yaml` | **Global** — machine-wide; central vault + storage engine + MCP port | no |
| `<repo>/.memory/config.yaml` | **Local** — per-repository scope and overrides | yes |
| `<repo>/.memory/memory.db` | Local index (SQLite) | no (gitignored) |
| `~/.memory/memory.db` | Global index for the central vault | no |

Precedence, highest first: **CLI flag** → **environment variable** → **global config** →
**local config** → **built-in default**. See `mymemory-config` for what each layer can
override.

## Universal mistakes

1. **Forgetting `include`** — `mem init` creates the config, but an empty `include` indexes
   nothing. Search returns zero and it looks like a broken tool.
2. **Markdown links inside a vault** — `[text](./other.md)` creates no graph edge. Use
   `[[wikilink]]`.
3. **No `summary`** — disables level-L0 retrieval, the cheapest and broadest layer.
4. **Thinking indexed == current** — after editing files, `mem index` before searching, or run
   `mem watch`.
5. **Editing without reading** — a vault is often not under version control. Back up before a
   bulk rewrite.

Each sub-skill carries its own mistake list for its own surface.

## Sibling skills

Generic Obsidian syntax — callouts, embeds, properties, `.canvas` authoring — is **not** in this
package. Use the [`obsidian-markdown`], [`json-canvas`] and [`obsidian-bases`] skills for that.
This package only covers what is specific to a MyMemory index and note graph.

[`obsidian-markdown`]: https://github.com/kepano/obsidian-skills
[`json-canvas`]: https://github.com/kepano/obsidian-skills
[`obsidian-bases`]: https://github.com/kepano/obsidian-skills
