---
name: mymemory-graph
description: Query and repair the knowledge graph MyMemory builds from wikilinks, using mem inspect, path, hubs, clusters, impact, doctor, insights, drift, and canvas export. Use when understanding how notes connect, finding orphans or dangling links, measuring centrality, checking blast radius before a change, exporting a subgraph to a .canvas, or auditing documentation drift.
---

# MyMemory Graph and Health

The graph is what turns a folder of files into something navigable. Nodes are documents,
concepts and tags; edges are wikilinks typed by relation and epistemic status. These commands
query that topology and audit its integrity.

## Which analysis, when

| Question | Command |
| --- | --- |
| What is this note, and what touches it? | `mem inspect <note>` |
| How does A reach B? | `mem path <from> <to> --max-depth 6` |
| What anchors this vault? | `mem hubs --algorithm pagerank --top 10` |
| What do people link to most? | `mem hubs --algorithm degree --top 10` |
| Are these folders real topics? | `mem clusters --min-size 2 --json` |
| What breaks if I change this? | `mem impact <target> --depth 2` |
| What is broken? | `mem doctor` / `mem doctor --fix` |
| What links should exist? | `mem insights --limit 5` |
| What fell behind the code? | `mem drift --since HEAD~5..HEAD` |
| Is this shape worth drawing? | `mem export --canvas <note> --out graph.canvas` |

## Tags are edges, not nodes

A tag becomes a graph node only when it **resolves to a document** — the `tagged_as` edge is
written for the resolved tags alone. A tag that names no note, a conceptual
`#fuzzy-matching`, is **consultable** (`mem tags`, `mem tag`, `memory_get_tag`) but it is not
a node, it has no edge, and it does not show up as a dangling link (ADR-041, ADR-055). To give
it a graph presence, create a stub note whose `title` is exactly that tag.

## Reading the outputs

`mem inspect` prints three columns: inbound, the node itself, and outbound. **Inbound empty
means orphan; outbound empty means the note contributes nothing to navigation.** Both are
problems.

`degree` counts references — "what do people link to?". `pagerank` measures structural
authority — "what anchors the graph?". High degree with low PageRank means a note is widely
mentioned without being central; only the second tells you what traversal will pass through.

`mem clusters` partitions by weighted label propagation and reports Newman-Girvan modularity.
Compare cluster boundaries against folder boundaries: if clusters cut across folders, the
folders are not the topics. Low modularity means one blob, which means the structure is fake.

`mem doctor` detects dangling links, orphans, self-loops, and a 0–100 Health Score aggregating
dead-link and orphan ratios. Run it after every batch of edits — the failures accumulate
silently and traversal quietly stops reaching parts of the vault. `--fix` removes dangling
edges.

`mem insights` finds highly similar note pairs with no link between them. Each is a candidate
link, and this is the cheapest way to densify a knowledge base.

`mem drift` compares recent source changes against the notes and surfaces two symmetric
failures: stale documentation (code moved, note did not) and orphan code (new code with no
architectural decision attached). This is the mechanism that prevents memory rot — a base that
was accurate at creation and never re-checked becomes confidently wrong.

## Repair loop

```bash
mem doctor             # what is broken
mem doctor --fix       # remove dangling edges
mem insights           # connections that should exist
mem drift              # documentation that fell behind
mem index --force      # reindex after changes
```

Run them in that order: repair structure, then discover missing connections, then check
staleness, then reindex.

## Canvas export

`mem export --canvas <note>` writes a subgraph as JSON Canvas 1.0, which opens directly in
Obsidian Canvas and in any reader implementing the spec.

**A `.canvas` file is not indexed** — `include` patterns target `.md`. Anything a future search
must find belongs in a note, even when a canvas visualises it. A canvas complements notes, it
does not replace them.

Rule of thumb: if the value is the **shape** of the information, draw it; if the value is the
**content**, write it down.

> A hand-drawn Mermaid diagram is usually unnecessary: `mem pack` emits one for the subgraph it
> packages.

## References

- [references/graph-and-health.md](references/graph-and-health.md) — every graph command with
  its flags, what each output means, and the health checklist.
- [references/canvas-export.md](references/canvas-export.md) — JSON Canvas 1.0 node and edge
  schema, colour presets, layout guidance, and validation.