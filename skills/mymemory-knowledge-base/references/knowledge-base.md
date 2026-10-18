# Building a Knowledge Base

How to turn many sources into a linked, verifiable knowledge base inside a MyMemory vault.

---

## 0. Note vs. base

| | Single note | Knowledge base |
| --- | --- | --- |
| Answers | one question | one *topic* |
| Size | 1 file | many linked files |
| Navigated by | search hit | index page + wikilinks |
| Success when | correct | complete, cross-linked, current |

The failure mode is producing a folder of correct, disconnected files. That is a pile, not a
base: nothing can navigate it, and the graph tools return nothing useful.

---

## 1. The five steps

| # | Step | Output |
| --- | --- | --- |
| 1 | **Map** | Inventory of what exists and where the gaps are |
| 2 | **Read** | The source material read **in full**, not skimmed |
| 3 | **Verify** | Each factual claim checked against a primary source |
| 4 | **Distil** | One note per topic: what holds, what fails, what to do |
| 5 | **Link** | Index (MOC), cross-wikilinks, tags, then `mem doctor` |

> [!warning]
> **Step 2 is not optional.** Writing a note from a table of contents produces confident
> statements about a document nobody read. Reading *is* the work.

---

## 2. Step 1 — Map before writing

Answer two questions first: what already exists, and where is the hole?

```bash
# what is in the vault now, by folder
find /path/to/vault -name '*.md' | sed 's|/path/to/vault/||' | cut -d/ -f1 | sort | uniq -c

# is a candidate topic represented at all?
mem search --mode fts "topic" --limit 5
```

The map becomes the work plan. Two strategies:

- **By category** — close one folder completely. Cheaper, keeps every index consistent, easier
  to review. Prefer this.
- **By source** — finish one document. Useful when a single source dominates a topic.

Check for existing coverage before writing anything: `mem search` and `mem clusters` will often
reveal a note that already covers the ground.

---

## 3. Step 2 — Read the source completely

```bash
# size it before reading
wc -c /path/to/source/SKILL.md
wc -l /path/to/source/SKILL.md

# every file in the source, not just the entry point
find /path/to/source -type f
```

Read every file. Entry points routinely link to `references/` that contain the actual
constraints — and those are usually where the incorrect claims live.

---

## 4. Step 3 — Verify against the primary source

This step is what separates a knowledge base from an opinion repository.

> [!tip]
> **Golden rule: a factual claim enters the vault with a link to its primary source.** If the
> claim exists only in a blog post, a forum thread, or another model's output, it does not
> enter as fact — it enters as `[unverified]` in the report, or it does not enter.

Worked examples of claims that fail this test:

| Claim as usually stated | Reality |
| --- | --- |
| "Ranking factors: content 40%, backlinks 25%, technical 15%" | Never published by the search provider. No primary source exists. |
| "Add `changefreq=daily` to speed up crawling" | Field is accepted in the format and **ignored** for crawling. |
| "`FAQPage` markup earns a rich result" | Restricted to authoritative health/government sites. |
| "Set `priority: 1.0` to rank your most important pages" | Field is ignored. |

The verification habit is what makes a base worth more than its sources: it records not only
what is true, but what the source gets wrong and why that matters.

> [!warning]
> **Prefer the vendor's own documentation.** If a claim cannot be found in the project's
> official docs or repository, treat it as folklore until proven otherwise.

---

## 5. Step 4 — Anatomy of a distilled note

The standard note shape (see the `mymemory-notes` skill) plus the blocks that make it a base note:

````markdown
---
title: "Topic — what the source gets right and wrong"
category: resource
summary: "Audit of source X against Y: the corrections that change the decision."
tags:
  - topic
  - audit
status: ativo
---

# Topic — what the source gets right and wrong

> [!important]
> Source: `path/to/source` (author, version). Verified against <primary doc> on <date>.

## What still holds
## What is wrong (with evidence)
## What to do instead
## Related
````

Each callout has one job:

| Callout | Use for |
| --- | --- |
| `> [!important]` | Always first: which source, verified when. This is the provenance that makes the rest trustworthy. |
| `> [!warning]` | The error and its concrete cost: "following this wastes effort here". |
| `> [!tip]` | The alternative. Without it the note only criticises. |

Provenance is not decoration. It is what lets a reader (or a future agent) re-verify in one
step instead of re-reading the whole source.

---

## 6. Step 5 — Linking (what makes it a base)

Three mechanisms, all required:

1. **An index (MOC) per folder.** Every folder gets a `README.md` with frontmatter and a
   list of `[[note]] - one-line summary`. The MOC is what an agent reads first and is the
   entry point to navigation.

2. **Cross-links on both sides.** The new note goes into the MOC **and** gets 2–3
   `[[wikilinks]]` under `## Related`. One-directional links do not close the graph.

3. **`category` in frontmatter.** So `mem search --category resource "..."` retrieves the
   whole set.

```markdown
## Related

- [[sibling-note]] — same topic, other angle
- [[README]] — index for this folder
```

Then validate:

```bash
mem index --force
mem doctor          # dangling links, orphans, Health Score
```

> [!warning]
> **Verify every wikilink target exists** before considering the note done. A link to a note
> that was never created is exactly what `mem doctor` is for.

---

## 7. Text integrity

> [!important]
> **Accents are retrieval, not style.** `sao` and `são` produce different tokens, so a search
> for one never matches the other. This is a recall bug, not a typo. It applies to `title` and
> `summary` exactly as it does to the body.

- UTF-8 **without BOM** — a BOM becomes part of the first token of every note.
- Preserve the vault's existing line endings.
- Do not introduce stray non-Latin characters.

---

## 8. Finishing a batch

```bash
mem index --force
mem doctor
mem search --mode fts "a distinctive phrase from the new note"   # proves it is retrievable
```

> [!warning]
> **A vault is often not under version control.** Copy the target folder before a bulk rewrite,
> and read each file before editing it. Overwriting unread is unrecoverable.

Report per batch: files created, notes added to each index, findings that were deliberately
*not* written, and anything blocked.

---

## 9. Per-note checklist

- [ ] Frontmatter has `title`, `category`, `summary`, `tags`
- [ ] `> [!important]` provenance on the first line
- [ ] Every factual claim has a primary-source link
- [ ] `## Related` with 2–3 wikilinks that **exist**
- [ ] Corresponding entry in the folder `README.md`
- [ ] UTF-8 without BOM, correct accents, no stray characters
- [ ] Retrievable: confirmed by an actual `mem search` hit
