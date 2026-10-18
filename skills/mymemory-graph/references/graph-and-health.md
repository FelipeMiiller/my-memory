# Graph and Health Reference

The graph is what turns a folder of files into something navigable. These commands query it and
audit its integrity.

---

## 1. Inspection

### `mem inspect <note>`

Three columns at once — the fastest way to understand a note's role:

- **Inbound** — who references this note
- **Node** — title, status, structural metrics, preview
- **Outbound** — what this note references

```bash
mem inspect docs/ARCHITECTURE.md
```

> [!tip]
> If the inbound column is empty, the note is an orphan. If the outbound column is empty, it
> contributes nothing to navigation. Both are problems — run `mem doctor`.

### `mem path <from> <to>`

Shortest conceptual route between two notes, resolved with a recursive CTE.

```bash
mem path docs/ARCHITECTURE.md docs/CLI_GUIDE.md --max-depth 6
```

Useful for answering "how does A relate to B?" without reading either file.

---

## 2. Structure

### `mem hubs`

```bash
mem hubs --algorithm degree --top 10     # most-referenced nodes
mem hubs --algorithm pagerank --top 10   # structural authority
```

**Degree** answers "what do people link to?"; **PageRank** answers "what anchors the graph?".
The difference matters: high degree with low PageRank means a note is widely mentioned but not
structurally central.

### `mem clusters`

```bash
mem clusters --min-size 2 --json
```

Partitions the graph into thematic communities via weighted label propagation, reporting
Newman-Girvan modularity. A low modularity value means the notes cluster into one blob — the
folders are not really distinct topics.

> [!tip]
> Compare cluster output against your folder structure. If clusters cut across folder
> boundaries, your folders may not reflect the actual topics.

---

## 3. Risk

### `mem impact <target>`

```bash
mem impact internal/db/graph.go --depth 2 --json
```

Reverse dependencies and a blast-radius Risk Score. Run this **before** refactoring, renaming,
or deleting anything — it answers "what breaks if I change this?" without reading every
dependent file.

---

## 4. Health

### `mem doctor`

```bash
mem doctor
mem doctor --fix     # removes dangling edges
```

Detects:

| Issue | Meaning |
| --- | --- |
| **Dangling links** | A `[[wikilink]]` pointing at a note that does not exist |
| **Orphans** | No inbound and no outbound links — unreachable by traversal |
| **Self-loops** | A note linking to itself |
| **Health Score** | 0–100 aggregate of dead-link and orphan ratios |

> [!warning]
> **Run `mem doctor` after every batch of edits.** Dead links and orphans accumulate silently
> and depress the Health Score, and graph traversal quietly stops reaching parts of the vault.
> `mem doctor --fix` removes dangling edges automatically.

### `mem insights`

```bash
mem insights --limit 5
```

Finds highly similar note pairs with no link between them — latent connections. Each one is a
candidate: decide whether the link deserves to be forged.

> [!tip]
> This is the cheapest way to improve a knowledge base. Every suggestion is a link that would
> make the graph denser and both notes easier to find.

---

## 5. Drift

### `mem drift`

```bash
mem drift --since HEAD~5..HEAD
mem drift --strict --json
```

Compares recent source changes against the notes, surfacing two symmetric failures:

- **Stale documentation** — code changed, the note did not
- **Orphan code** — new code with no architectural decision or note attached

```bash
# typical pre-push sequence
mem index --force
mem drift --since HEAD~1
mem doctor
```

> [!important]
> Drift is the mechanism that stops "memory rot". A base that was accurate at creation and
> never re-checked becomes worse than useless: it is confidently wrong. Schedule drift against
> the same cadence as the code review itself.

---

## 6. Repair loop

```bash
mem doctor             # what is broken
mem doctor --fix       # remove dangling edges
mem insights           # connections that should exist
mem drift              # documentation that fell behind
mem index --force      # reindex after changes
```

Run them in that order: repair structure, then discover missing connections, then check for
staleness, then reindex.

---

## 7. Health checklist

- [ ] No dangling links (`mem doctor`)
- [ ] No orphan notes — every note is reachable from an index
- [ ] Every folder has an index (MOC) page
- [ ] Cross-links exist in both directions
- [ ] `mem drift` reports no stale documentation
- [ ] Health Score stable or improving
