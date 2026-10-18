---
name: mymemory-knowledge-base
description: Turn many sources into a linked, verifiable knowledge base inside a MyMemory vault using a five-step playbook — map, read, verify against the primary source, distil, interlink. Use when building out a topic from documentation or a corpus, writing an audit of a source, or turning disconnected notes into something navigable.
---

# MyMemory Knowledge Base

## A single note is not a base of knowledge

| | Single note | Knowledge base |
| --- | --- | --- |
| Answers | one question | one **topic** |
| Size | 1 file | many linked files |
| Navigated by | a search hit | index page + wikilinks |
| Succeeds when | it is correct | it is complete, cross-linked, and current |

The failure mode is a folder of correct, disconnected files. That is a pile, not a base:
nothing can navigate it and every graph command returns noise.

## The playbook

| # | Step | Output |
| --- | --- | --- |
| 1 | **Map** | Inventory of what exists and where the holes are |
| 2 | **Read** | The source material read **in full**, not skimmed |
| 3 | **Verify** | Each factual claim checked against a primary source |
| 4 | **Distil** | One note per topic: what holds, what fails, what to do |
| 5 | **Interlink** | Index (MOC), cross-wikilinks, tags, then `mem doctor` |

**Step 2 is not optional.** A note written from a table of contents produces confident
statements about a document nobody read. Reading *is* the work. Entry points routinely link to
`references/` that hold the actual constraints — and that is where the incorrect claims live.

## Verify against the primary source

This step is what separates a knowledge base from an opinion repository.

> A factual claim enters the vault with a link to its primary source. If it exists only in a
> blog post, a forum thread, or another model's output, it does not enter as fact — it enters
> as `[unverified]` in the report, or it does not enter.

Prefer the vendor's own documentation. If a claim cannot be found in the project's official
docs or repository, treat it as folklore until proven otherwise.

Recording what a source gets wrong — and what that error would have cost — is what makes a base
worth more than its sources.

## Shape of a base note

```markdown
---
title: "Topic — what the source gets right and wrong"
category: resource
summary: "Audit of source X against Y: the corrections that change the decision."
tags: [topic, audit]
status: ativo
---

# Topic — what the source gets right and wrong

> [!important]
> Source: path/to/source (author, version). Verified against <primary doc> on <date>.

## What still holds
## What is wrong (with evidence)
## What to do instead
## Related
```

The `> [!important]` provenance line always comes first and always names the source, the
verification target, and the date. It is what lets a reader re-verify in one step instead of
re-reading the whole source. Then `> [!warning]` for the error and its concrete cost,
`> [!tip]` for the alternative — without the tip the note only criticises.

Note anatomy and the frontmatter taxonomy are in the `mymemory-notes` skill.

## Three linking mechanisms, all required

1. **An index (MOC) per folder.** Every folder gets a `README.md` with frontmatter and a list
   of `[[note]] - one-line summary`. This is what an agent reads first.
2. **Cross-links on both sides.** The new note goes into the MOC **and** carries 2–3
   `[[wikilinks]]` under `## Related`. One-directional links do not close the graph.
3. **`category` in frontmatter**, so `mem search --category resource "..."` retrieves the whole
   set.

Then validate:

```bash
mem index --force
mem doctor          # dangling links, orphans, Health Score
```

A vault is often not under version control. Copy the target folder before a bulk rewrite, and
read each file before editing it — overwriting unread is unrecoverable.

## Reference

[references/knowledge-base.md](references/knowledge-base.md) — the map/read/verify steps in
full, worked examples of claims that fail verification, text-integrity rules, and the per-note
checklist.