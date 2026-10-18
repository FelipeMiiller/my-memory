# Quickstart — Copy-Pasteable

Seven short, self-contained snippets. Copy, paste, adapt.

---

## 1. Create a vault and index it

```bash
mkdir my-project && cd my-project
git init

mem init

cat > .memory/config.yaml <<'YAML'
version: 1
repo_id: repo_local
repository: local/project

include:
    - 'docs/**/*.md'
    - 'README.md'

exclude:
    - .git/**
    - node_modules/**
    - .memory/**
YAML

mem index
mem doctor
```

Nothing is searchable until `include` is set. An empty scope indexes zero files, and search
returns zero results in a way that looks like a broken tool.

---

## 2. Write the first note

```bash
mkdir -p docs/concepts

cat docs/concepts/cache-strategy.md <<'NOTE'
---
title: "Cache strategy"
category: resource
summary: "How the cache is keyed, invalidated and measured."
tags:
  - cache
  - performance
status: ativo
---

# Cache strategy

The cache is keyed by content hash, so an unchanged file is skipped on reindex.

Related: [[README]]
NOTE

mem index
mem search "how is the cache invalidated"
```

---

## 3. Build an index page (MOC)

```bash
cat docs/README.md <<'MD'
---
title: "docs"
category: resource
summary: "Index of the docs folder: technical documentation."
tags:
  - navigation
  - docs
status: ativo
---

# docs

Reference documentation for this repository.

## Notes

- [[cache-strategy]] - How the cache is keyed, invalidated and measured.
MD

mem index
mem doctor
```

---

## 4. Search from an agent (MCP)

```json
{
  "mcpServers": {
    "my-memory": {
      "command": "mem",
      "args": ["mcp", "--storage", "sqlite", "--repo", "repo_local"]
    }
  }
}
```

```bash
mem mcp --storage sqlite --repo repo_local
```

A typical agent loop:

```
memory_search "cache invalidation"     # hybrid
memory_search "cache" --detail_level l0  # cheapest layer: summaries only
memory_write_note                       # write the finding back, indexed immediately
memory_doctor                           # confirm no dangling links
```

---

## 5. Compile a topic into one note

```bash
mem search --mode hybrid "postgres federation" --limit 10
# ... synthesise ...

mem compile --topic "Postgres federation" --out docs/concepts/postgres-federation.md
mem index
```

Read the compiled note instead of re-querying every source next time. This is the
compile-not-retrieve pattern and it is the cheapest way to build durable knowledge.

---

## 6. Wire the index into git

```bash
mem hook --install
git add .memory/config.yaml .memory/AGENTS.md
git commit -m "chore: initialize memory vault"
```

Every commit now reindexes. `mem doctor` before pushing keeps the graph clean.

---

## 7. Pre-push check

```bash
mem index --force
mem drift --since HEAD~1     # documentation that fell behind code
mem doctor                   # dangling links, orphans, Health Score
mem status                   # resolved engine and config
```

If `mem drift` reports stale documentation, fix the note before pushing — that is the cheapest
moment to correct it.

---

## Troubleshooting

| Symptom | Cause | Fix |
| --- | --- | --- |
| Search returns nothing | `include` empty or too narrow | Widen `include`, `mem index --force` |
| Search finds exact terms but no concepts | Binary built without `-tags onnx` | Rebuild with the tag; the builtin embedder is compiled out |
| Note never appears | Excluded, or outside the vault | Check `exclude` and the `--vault` path |
| Retrieval feels wrong | Embedding dimension changed | `mem index --force` after any model switch |
| Graph looks disconnected | Orphans and missing indexes | `mem insights`, then add MOC entries |
| MCP reads the wrong content | No `--storage` / `--repo` | Pass both flags |
| Auth fails after a config change | Environment variable not inherited | Restart the shell or editor |
