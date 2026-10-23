---
name: mymemory-notes
description: Author Markdown notes inside a MyMemory vault so they are indexed, taxonomised, and connected by the knowledge graph. Use when creating or editing a note in the vault, choosing category/summary/tags frontmatter, wiring [[wikilinks]] or typed [[rel:]] edges, or writing a note through mem note create, mem note append, or mem compile.
---

# MyMemory Notes

Writing a note for a MyMemory vault is not the same as writing Markdown. A note is only
reachable if it is **indexed**, **described**, and **linked**. This skill covers what is
specific to MyMemory. For generic Obsidian syntax — callout types, embed forms, property
syntax, block IDs — use the `obsidian-markdown` skill instead of duplicating it here.

## Minimum viable note

```markdown
---
title: "Descriptive name"
category: resource      # resource | memory | skill
summary: "One or two sentences. This is what level-L0 retrieval returns."
tags:
  - topic
  - subtopic
status: ativo            # ativo | deprecated | draft
---

# Descriptive name

Body with [[wikilinks]] to related notes.
```

`title`, `category`, `summary` and `tags` are what the index actually consumes. A note missing
`summary` is invisible to L0 retrieval, the cheapest and broadest retrieval layer.

> Write `summary` as if it were the only line anyone will ever read. Usually it is: L0 returns
> it alone, with no file read.

## The `category` taxonomy

`category` is not decorative — it is the axis `mem search --category` filters on.

| Value | Use for | Example |
| --- | --- | --- |
| `resource` | Persistent technical documentation | Architecture notes, audits |
| `memory` | Rules, habits, agent conventions | Quality principles |
| `skill` | Operational instructions and playbooks | Runbooks, guides |

`mem search --category skill "..."` returns only operational instructions, separated from
reference documentation. Choosing `resource` for a runbook makes the filter lie.

## Wikilinks become graph edges

Use `[[Note]]` for internal references. `[text](./other.md)` produces **no** edge and the note
stays out of graph traversal.

A bare `[[Note]]` is a `links_to` edge. Prefix the target to type it:

```markdown
[[rel:depends_on:Configuration]]
[[rel:derived_from:ADR-018]]
```

Recognised relations: `links_to`, `implements`, `depends_on`, `supports`, `refutes`, `extends`,
`derived_from`, `fixes`, `causes`, `part_of`, `related_to`, `similar_to`, `tagged_as`.
An unrecognised relation is not a typed edge — it degrades to a plain link. The alias form
`[[Note|rel:depends_on]]` is equivalent.

Edges extracted from a note are written with epistemic status `EXTRACTED` and weight `1.0`;
`derived_from` backlinks are emitted by `mem compile`. The epistemic-weighting model is
`docs/adr/011-arestas-epistemicas-e-god-nodes.md`.

ADRs are the documented exception in this project: they link to file paths with Markdown
links, by long-standing convention.

## Tags are queryable; only the resolved ones become nodes

**ADR-055: every tag of a document is recorded — including the one that names no note.** The
orphan tag, a conceptual `#fuzzy-matching` that resolves to nothing, is still listed by
`mem tags` and answered by `mem tag` / `memory_get_tag`. It is consultable, not dead.

**ADR-041: a tag with no resolvable document is not a graph node.** It creates no edge and no
dead link, so it has no graph presence until a document matches it. To make a tag a real node,
create a stub note whose `title` is exactly that tag.

In frontmatter, write tags without `#`. In the body, inline tags are contextual to a paragraph.
Tags are **case-insensitive** and **hierarchical**: `projeto` also matches `projeto/alpha`.

> When documenting tag syntax, wrap it in inline code — `` `#foo` `` documents it, bare `#foo`
> in prose is captured by the parser as a real tag. This holds inside code blocks too: the
> parser reads text, not Markdown semantics.

## Obsidian syntax the index ignores

Renders fine in Obsidian, contributes nothing retrievable. Human eyes only — never the sole
carrier of something a reader must be able to find.

| Syntax | Consequence |
| --- | --- |
| `%%comment%%` | **Not stripped from the text** — commented-out content is indexed and retrievable, so stale knowledge looks deleted but still answers searches. The `%%` block is removed for **tag extraction only**: a tag inside a comment is not captured. Delete the comment. |
| `![[embed]]` | Embedded content is not indexed and creates **no** edge. To display *and* link, add a separate `[[Note]]`. |
| `==highlight==` | Indexed as text; the `==` is noise. |
| `$$math$$` | Indexed as literal notation. |

## Write paths

```bash
mem note create <path> --title "Title" --tags "a,b" --type concept
mem note append <path> --heading "## Section" --content "..."
mem compile --topic "Topic" --out <path>
```

`--type` accepts `concept` (default), `decision`, `summary`, `entity`. Without `--content`, the
body is read from stdin — the normal path when a script or an agent generates the Markdown.

When the target already has frontmatter, CLI flags are **merged, not substituted**: `title` and
`type` from the CLI win, `tags` and `aliases` are unioned, and keys you did not pass are
preserved.

Accents are retrieval, not style: tokenisers treat `sao` and `são` as different tokens, so a
search for one never matches the other. UTF-8 **without BOM** — a BOM corrupts the first token
of every note.

Full rules, the MOC index pattern, and the pre-delivery checklist:
[references/notes-format.md](references/notes-format.md). Flag semantics live in the
`mymemory-cli` skill.