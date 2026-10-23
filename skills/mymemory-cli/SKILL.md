---
name: mymemory-cli
description: Drive the MyMemory `mem` command line to create and index a Markdown vault, search it, author notes, and audit the note graph. Use when running or scripting `mem` commands such as mem init, mem index, mem search, mem doctor, mem drift, mem note create, mem compile, mem graph, mem hubs, or mem code-index, or when deciding which `mem` subcommand to reach for.
---

# MyMemory CLI

`mem` is a single Go binary that turns a folder of Markdown into an indexed, searchable, linked
knowledge graph. It is the primary interface for setting up a vault, keeping the index current,
retrieving notes, and checking graph health.

## Get it

```bash
go install github.com/FelipeMiiller/my-memory/cmd/mem@latest

# or from source — the onnx tag is required for the semantic (vector) layer
git clone https://github.com/FelipeMiiller/my-memory.git
cd my-memory
go build -tags onnx -o bin/mem ./cmd/mem
```

A binary built **without** `-tags onnx` still runs and still returns exact-term results, but it
silently has no vector layer — meaning-based retrieval quietly stops working. See
`mymemory-config` for details.

## The command surface

Run `mem help` to see every command with its current flags. **That output is always accurate;
this file is not a substitute for it.** Read `mem help` before assuming a flag exists.

Two shapes are worth memorising because most other commands are variations on them:

- `mem search "<query>"` is one command with **modes**, not three commands. `--mode hybrid`
  (default, RRF over FTS + vector + graph), `--mode vector` (meaning only), `--mode fts` (exact
  terms). There is no `mem fts` or `mem vector`.
- Sub-verbs take an action as the first argument: `mem note <create|append>`,
  `mem hook <install|uninstall>`, `mem central <status|bootstrap>`,
  `mem profiles <list|use>`, `mem events <tail|inspect|replay|trace|last-sequence|stats>`.

## Task shapes

**New repository**

```bash
mem init              # writes .memory/config.yaml, .gitignore, AGENTS.md
# edit .memory/config.yaml -> include / exclude
mem index
mem doctor
```

Nothing is searchable until `include` is set. An empty scope indexes zero files and search returns
zero results, which looks like a broken tool rather than an unconfigured one.

**Keep the index current**

```bash
mem index             # incremental; SHA-256 cache skips unchanged files
mem index --force     # reprocess everything, e.g. after an embedding model switch
mem watch             # continuous reindex on change
mem hook --install    # git pre-commit reindex
mem up / mem down     # background supervisor (mymemoryd)
```

**Retrieve**

```bash
mem search "how does incremental indexing work"        # hybrid
mem search --mode fts "CalculateContentHash"           # identifiers, exact terms
mem search --mode vector "how ranking works"           # meaning
mem search --level l0 "config"                        # summaries only, cheapest layer
mem search --decay --half-life 14 "recent ADR"        # bias toward recently EDITED notes (file mtime)
mem search --category skill "note creation"           # resource | memory | skill
mem search --tag projeto "architecture"              # Obsidian tag; matches the tag AND its descendants
```

**Read the vault** — every one of these queries the index, never the filesystem, so they
work on a Postgres-only vault with no local copy of the file.

```bash
mem files                              # every indexed path, alphabetical
mem files --ext md --folder docs/adr   # filter by extension and folder segment
mem folders                            # every folder holding an indexed document
mem read <path>                        # the note's raw Markdown
mem outline <path>                     # heading tree with depth
mem wordcount <path>                   # words and characters
mem random [--json]                    # one randomly selected note
mem deadends                           # notes with out-edges but no in-edges
mem file <path>                        # path, title, size, mtime (ADR-057)
mem recents [--limit 10]               # most recently edited first
mem tags                               # every tag in the vault, one per line
mem tags --counts --sort=count         # tag<TAB>count, most used first
mem tags --file <path>                 # only that note's tags
mem tag <name>                         # how many notes carry the tag (and its descendants)
mem tag <name> --verbose               # the count plus the notes themselves
```

`read`, `outline`, `wordcount` and `file` take a full path or just the basename. An exact
path always wins; when only a basename matches and several notes share it, the command
reports the ambiguity with the candidate paths instead of picking one. An extension or
folder filter that matches nothing exits zero with an empty list — that is a valid answer
about the vault, not a failure.

**Author**

```bash
cat body.md | mem note create notes/subject.md --title "Title" --tags "a,b" --type concept
mem note append existing.md --heading "## New Section" --content "..."
mem compile --topic "Postgres federation" --out docs/concepts/federation-postgres.md
```

**Graph and health**

```bash
mem inspect <note>              # in-links | node | out-links
mem path <from> <to>            # shortest conceptual route
mem hubs --algorithm pagerank   # authoritative nodes
mem clusters --min-size 2       # thematic communities
mem impact <note> --depth 2     # blast radius before changing or deleting
mem insights                    # similar notes with no link yet
mem doctor                      # dangling links, orphans, self-loops, Health Score
mem doctor --fix                # removes dangling edges
mem drift --since HEAD~5..HEAD  # code vs documentation divergence
```

**Code symbols**

```bash
mem code-index
mem code-search "renderNote"
mem code-graph "SomeFunction"
mem code-stats
```

## Storage flags

Commands that touch the index accept `--storage sqlite|postgres`, `--db <path>`,
`--postgres <url>` and `--repo <slug>`. Without them, resolution follows config precedence —
see `mymemory-config`. In CI, pass `--storage=sqlite` explicitly to stay hermetic.

## References

- [quickstart.md](references/quickstart.md) — seven copy-pasteable snippets, setup to pre-push check
- [cli-reference.md](references/cli-reference.md) — every command grouped by task, plus typical sequences

## Before pushing

```bash
mem index --force
mem drift --since HEAD~1     # documentation that fell behind code
mem doctor
mem status                   # resolved engine and config
```

## Gotchas

1. **Indexing is not caching your writes** — edit files, then `mem index`, or search stale content.
2. **Every retrieval command accepts the same storage flags** — omitting them is how a script
   silently writes to the wrong database.
3. **`mem help` over any documentation**, including this skill, when a flag is in doubt.
4. **Run `mem doctor` after batches of writes** — dangling links accumulate silently and depress
   the Health Score.