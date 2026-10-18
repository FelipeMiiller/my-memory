# MyMemory Agent Skills

Agent skills for [MyMemory](https://github.com/FelipeMiiller/my-memory) — a local Go tool that turns any
folder of Markdown into a searchable, linked knowledge graph. Hybrid retrieval blends exact-term
search (BM25), semantic vectors, and graph traversal; the result is exposed through a CLI (`mem`) and
an MCP server with 20 tools.

These skills follow the [Agent Skills specification](https://agentskills.io/specification), the same
format used by the [`obsidian-skills`](https://github.com/kepano/obsidian-skills) package, so they load
in any skills-compatible agent.

## Layout

The package is split by capability. An agent loads the **router** to find its way, then loads exactly
one focused skill.

```
skills/
├── README.md                     this file
├── mymemory/                     router — finds the right skill, holds the cross-cutting model
├── mymemory-cli/                 the mem command line
├── mymemory-mcp/                 MCP server and its 20 tools
├── mymemory-notes/               authoring notes in the vault
├── mymemory-graph/               graph queries, vault health, canvas export
├── mymemory-config/              config precedence and storage
└── mymemory-knowledge-base/      building a base of knowledge from many sources
```

## Skills

| Skill | Description |
| --- | --- |
| [`mymemory`](mymemory/) | Router. Load when you need to find which skill answers the task, or for the cross-cutting model: a note is only reachable if it is **indexed, linked and described**. |
| [`mymemory-cli`](mymemory-cli/) | Run `mem` from a shell or script: install, `init`, `index`, `search`, `watch`, `hook`, every command and flag. Starts with a 60-second quickstart. |
| [`mymemory-mcp`](mymemory-mcp/) | Wire an MCP client, decide between a tool and the CLI, and read the full contract of all 20 tools. |
| [`mymemory-notes`](mymemory-notes/) | Write and edit notes: the `category` taxonomy, `summary`, wikilinks, typed edges, tags, and the `mem note` / `mem compile` write paths. |
| [`mymemory-graph`](mymemory-graph/) | Query the graph and audit the vault: `inspect`, `path`, `hubs`, `clusters`, `impact`, `drift`, `doctor`, and `.canvas` export. |
| [`mymemory-config`](mymemory-config/) | Where settings live and which one wins: precedence, the global/local split, SQLite vs Postgres, and the central vault. |
| [`mymemory-knowledge-base`](mymemory-knowledge-base/) | Build a base of knowledge from many sources — map, read, verify against primary sources, distil, interlink. |

## Relationship to `obsidian-skills`

This package deliberately does **not** duplicate generic Obsidian syntax. For callouts, embeds,
properties, `.canvas` authoring and `.base` views, use the companion package:

| Topic | Use instead |
| --- | --- |
| Callouts, embeds, properties, wikilink syntax | [`obsidian-markdown`](https://github.com/kepano/obsidian-skills) |
| Authoring `.canvas` files | [`json-canvas`](https://github.com/kepano/obsidian-skills) |
| Authoring `.base` files | [`obsidian-bases`](https://github.com/kepano/obsidian-skills) |

These skills only cover what is specific to a MyMemory index and note graph.

## Installation

```
# Claude Code / any skills-compatible agent
npx skills add https://github.com/FelipeMiiller/my-memory
```

Or copy the `skills/` directory into your agent's skills path. The `mem` binary itself is
installed separately — `go install github.com/FelipeMiiller/my-memory/cmd/mem@latest`.

## License

MIT. See [LICENSE](../LICENSE).
