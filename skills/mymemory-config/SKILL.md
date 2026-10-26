---
name: mymemory-config
description: Resolve how MyMemory decides which vault, storage engine, database, and embedding model to use. Use when writing or debugging `~/.memory/config.yaml` or `.memory/config.yaml`, choosing between SQLite and Postgres, wiring the `MY_MEMORY_*` environment variables, forcing an engine with `--storage`, or diagnosing search that returns exact terms but no concepts.
---

# MyMemory Configuration and Storage

MyMemory resolves **which vault** it is operating on, **which engine** stores it, and **which
embedder** indexes it — through a fixed chain of sources, then two layers of configuration file.

## Precedence

Highest wins:

1. **CLI flag** — `--storage=postgres`, `--db <path>`, `--postgres <url>`, `--repo <slug>`
2. **Environment variable** — `MY_MEMORY_PG_URL`, `MY_MEMORY_PG_PASSWORD`, `MY_MEMORY_REPO`,
   `MY_MEMORY_EMBED_*`
3. **Local config** — `<repo>/.memory/config.yaml` (sobrescreve o global chave a chave, ADR-040)
4. **Global config** — `~/.memory/config.yaml`
5. **Built-in default**

> **A URL do Postgres tem precedência própria:** `--postgres` > `MY_MEMORY_PG_URL` >
> `POSTGRES_URL` > `DATABASE_URL` > config (local > global). O binário anuncia no
> **stderr** qual fonte venceu — `storage: postgres (fonte: MY_MEMORY_PG_URL)` — e o
> stdout do `--json` continua JSON puro. "Eu configurei o config" **não** prevê o engine:
> a variável de ambiente vence o config.

Overriding explicitly is usually the right move in scripts and CI. `--storage=sqlite` forces a
local database even when a machine-wide Postgres is configured, which keeps a pipeline hermetic.

## The two layers

| File | Scope | In version control |
| --- | --- | --- |
| `~/.memory/config.yaml` | **Global** — central vault, storage engine, MCP port, registered repositories | no |
| `<repo>/.memory/config.yaml` | **Local** — this repository's index scope and overrides | yes |
| `<repo>/.memory/.env` | Local secrets (Postgres password, embedder keys) | no |

> [!important]
> The global file lives in the user's home directory at `~/.memory/config.yaml`. It is the one
> place shared by every repository on the machine and is deliberately kept out of version control.
> The per-repository `.memory/config.yaml` **is** committed, so a team shares one indexing scope.

`include` / `exclude` in the local file define the index scope. An empty or too-narrow `include`
is the single most common reason search returns nothing — and it looks like a broken tool rather
than an unconfigured one. Use the narrowest `include` that covers what should be searchable.

## Storage model

| Engine | When | Where the data lives |
| --- | --- | --- |
| `sqlite` | Default. Single machine, single repo, zero setup. | `<repo>/.memory/memory.db` |
| `postgres` | Central vault, multiple machines, multi-tenant. | Postgres + `pgvector` |

**The opt-out rule:** a repository that declares `storage.engine: sqlite` **discards any inherited
`postgres_url`**. This is what lets one machine run a Postgres-backed central vault while every
individual repository keeps its own SQLite file with no extra configuration.

```
Global config: engine: postgres  ──▶  central vault (Postgres)
                                         ▲
Repo A: engine: sqlite ──▶ memory.db      │ inherited config discarded
Repo B: engine: sqlite ──▶ memory.db      │
Repo C: engine: sqlite ──▶ memory.db  ────┘
```

## Embeddings

The **only** vector provider is `builtin`: `paraphrase-multilingual-MiniLM-L12-v2`, 384
dimensions, in-process (ADR-035). Configuration exists for `onnx` and `openai` entries, but the
dimension is not negotiable against the schema.

> [!warning]
> **A binary built without `-tags onnx` has no `builtin` embedder.** It does not fail loudly — it
> prints a warning and falls back to lexical-only indexing, so `mem search` still returns results
> but with **no semantic layer at all**. Exact-term queries keep working; everything that relied on
> meaning quietly stops working. `mem search --mode vector` is the exception: it errors out and
> exits non-zero, because there is no vector layer to query. If search suddenly finds nothing by
> concept, check this first.
>
> ```bash
> go build -tags onnx -o bin/mem ./cmd/mem
> ```

> [!warning]
> **The dimension must match the schema.** Switching to a model with a different dimension
> requires a full reindex (`mem index --force`); otherwise vectors and the index disagree silently
> and retrieval quality degrades with no error at all.

## Secrets

> [!warning]
> **URL in the config file, password in an environment variable.** The connection URL is not a
> secret — it needs to be readable and shareable, and MCP clients spawn the server as a child
> process that inherits only the `env` block of their config. Keep the password in
> `MY_MEMORY_PG_PASSWORD` and never on disk.

```bash
# macOS / Linux
export MY_MEMORY_PG_PASSWORD='...'

# Windows PowerShell, user scope, survives restart
[Environment]::SetEnvironmentVariable('MY_MEMORY_PG_PASSWORD', '<password>', 'User')
```

After creating or changing an environment variable, **restart the shell or editor**. A server
already running keeps its inherited environment and will keep failing to authenticate until it is
restarted.

## Verify

```bash
mem status          # resolved config and engine
mem repos           # registered repositories
mem doctor          # index health
```

## Reference

- [configuration.md](references/configuration.md) — full file layouts, every key, precedence,
  storage model, embedding settings and secrets