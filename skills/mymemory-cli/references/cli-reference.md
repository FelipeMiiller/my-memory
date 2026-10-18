# CLI Reference

36 top-level commands. Grouped by what you are trying to do.

---

## Setup

| Command | Purpose |
| --- | --- |
| `mem init` | Create `.memory/` with `config.yaml`, `.gitignore`, `AGENTS.md` |
| `mem init --all` | Also generate MCP configs for Cursor and VS Code |
| `mem setup` | Interactive global setup → writes `~/.memory/config.yaml` |
| `mem install` | Auto-configure MCP across supported editors |

`mem install` has no removal counterpart: `uninstall` exists only as `mem hook uninstall`, for
the Git pre-commit hook.

```bash
mem init --force          # overwrite an existing vault
mem init --repo my-repo   # explicit repo slug
```

| After `mem init` | File | Committed |
| --- | --- | --- |
| `.memory/config.yaml` | Index scope, paths, embedding config | yes |
| `.memory/memory.db` | The index itself | no (gitignored) |
| `.memory/.gitignore` | Keeps `.env` and `*.db` out of git | yes |
| `.memory/AGENTS.md` | Ready-to-paste agent instructions | yes |

---

## Indexing

| Command | Purpose |
| --- | --- |
| `mem index` | Incremental index; SHA-256 cache skips unchanged files |
| `mem index --force` | Reprocess everything from scratch |
| `mem index --no-prune` | Keep entries for deleted files (auditing only) |
| `mem watch` | Continuous reindex on file change |
| `mem hook --install` | Git pre-commit hook that reindexes |
| `mem up` / `mem down` | Start / stop a long-running indexer |
| `mem events` | Inspect the event_runtime `event_log` (JSON Lines) |

`mem events` takes a sub-action: `tail`, `inspect`, `replay`, `trace`, `last-sequence`, `stats`.
`replay` is a sub-action here — there is no top-level `mem replay`.

```bash
mem index --storage sqlite
mem index --postgres "postgres://USER:PASS@HOST:5432/DB"
```

---

## Search

| Command | Purpose |
| --- | --- |
| `mem search "<query>"` | Hybrid search (BM25 + vector + graph) |

`hybrid`, `vector` and `fts` are modes of `mem search`, not separate commands:

```bash
mem search --mode fts "CalculateContentHash"
mem search --mode vector "how ranking works"
mem search --decay --half-life 14 "recent decision"  # ranks by file edit time (mtime), not index time
mem search --level l0 "config"
mem search --category skill "note creation"
```

---

## Notes

```bash
mem note create <path> --title "Title" --tags "a,b" --type concept
mem note create <path> --vault /path/to/vault --title "Title"
cat body.md | mem note create <path> --title "Title"     # body from stdin

mem note append <path> --heading "## Section" --content "..."

mem compile --topic "Topic" --out <path>                # compile-not-retrieve
```

`--type` accepts `concept` (default), `decision`, `summary`, `entity`.
Full flag list and merge semantics: [`mymemory-notes`](../../mymemory-notes/references/notes-format.md).

---

## Graph

| Command | Purpose |
| --- | --- |
| `mem inspect <note>` | Inbound · node · outbound |
| `mem path <from> <to>` | Shortest conceptual route |
| `mem hubs` | Highest-centrality nodes (`--algorithm degree\|pagerank`) |
| `mem clusters` | Thematic communities + modularity |
| `mem impact <target>` | Blast radius and Risk Score |
| `mem insights` | Similar notes with no link yet |
| `mem graph` | Interactive graph visualisation |

```bash
mem hubs --algorithm pagerank --top 10
mem clusters --min-size 2 --json
mem impact <note> --depth 2 --json
mem graph view --open
```

---

## Health

| Command | Purpose |
| --- | --- |
| `mem doctor` | Audit dangling links, orphans, self-loops, Health Score |
| `mem doctor --fix` | Remove dangling edges |
| `mem drift` | Code vs documentation divergence |
| `mem status` | Resolved configuration and engine |
| `mem logs` | Entries from `logs/supervisor.jsonl` (ADR-042) |
| `mem profiles` | Worker execution profiles; sets the vault `active_profile` (ADR-042) |

```bash
mem drift --since HEAD~5..HEAD
mem drift --strict --json
```

---

## Federation

| Command | Purpose |
| --- | --- |
| `mem setup` | Register central vault and repositories |
| `mem central status` | Audit the central vault |
| `mem central bootstrap` | Create canonical folders, templates, index pages |
| `mem repos` | List federated repositories |
| `mem export` | Export a subgraph as JSON Canvas (`.canvas`) |

```bash
mem export --canvas <note> --out graph.canvas
```

---

## Code intelligence

| Command | Purpose |
| --- | --- |
| `mem code-index` | Index source symbols |
| `mem code-search` | Search functions, methods, classes |
| `mem code-graph` | Symbol dependency graph |
| `mem code-stats` | Index statistics |

```bash
mem code-index
mem code-search "renderNote"
```

---

## Server

```bash
mem mcp                                          # stdio, resolves vault from CWD
mem mcp --storage postgres --repo repo_central   # pin a specific vault
mem mcp --port 38400                             # HTTP/SSE
mem mcp --http 0.0.0.0:38400                     # network-exposed
```

> [!warning]
> Without `--storage` / `--repo` the server resolves the vault from the working directory.
> An agent running inside an unrelated repository reads that repository's memory, silently.

---

## Other

| Command | Purpose |
| --- | --- |
| `mem open <note>` | Open in the preferred editor |
| `mem pack <note>` | Package a subgraph under a token budget |
| `mem bench` | Benchmarks |
| `mem version` | Version |

```bash
mem pack <note> --depth 2 --max-tokens 4000 --out subgraph.md
```

---

## Typical sequences

**New repository**

```bash
mem init
# edit .memory/config.yaml
mem index
mem doctor
```

**After writing notes**

```bash
mem index --force
mem doctor
mem search "<a phrase from the new note>"     # prove it is retrievable
```

**Before pushing**

```bash
mem index --force
mem drift --since HEAD~1
mem doctor
```

**Federate a repository**

```bash
mem setup                                  # once per machine
mem central bootstrap                      # if the central vault is empty
# then, in the repo:
mem init
mem search "<query>"                       # local + central, with provenance labels
```
