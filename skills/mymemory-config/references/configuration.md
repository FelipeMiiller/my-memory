# Configuration Reference

---

## 1. Where configuration lives

| File | Scope | In version control |
| --- | --- | --- |
| `~/.memory/config.yaml` | **Global** — machine-wide defaults: central vault, storage engine, MCP port, registered repositories | no |
| `<repo>/.memory/config.yaml` | **Local** — this repository's index scope and overrides | yes |
| `<repo>/.memory/.env` | Local secrets (Postgres password, embedder keys) | no |

> [!important]
> **The global file lives in the user's home directory**, at `~/.memory/config.yaml`. It is the
> one place shared by every repository on the machine and is deliberately kept out of version
> control. The per-repository `.memory/config.yaml` is committed, so a team shares one indexing
> scope.

---

## 2. Precedence

Highest wins:

1. **CLI flag** — `--storage=postgres`, `--db <path>`, `--postgres <url>`
2. **Environment variable** — `MY_MEMORY_PG_URL`, `MY_MEMORY_PG_PASSWORD`, `MY_MEMORY_EMBED_URL`, `MY_MEMORY_EMBED_MODEL`, `MY_MEMORY_EMBED_DIM`
3. **Global config** — `~/.memory/config.yaml`
4. **Local config** — `<repo>/.memory/config.yaml`
5. **Built-in default**

> [!tip]
> Overriding explicitly is usually the right move in scripts and CI. `--storage=sqlite` forces
> a local database even when a machine-wide Postgres is configured, which keeps CI hermetic.

---

## 3. Global config

```yaml
version: 1

central_vault:
    path: /path/to/central-vault

storage:
    engine: sqlite          # sqlite | postgres
    sqlite_path: memory.db
    # postgres_url: "${MY_MEMORY_PG_URL}"
    # postgres_password: "${MY_MEMORY_PG_PASSWORD}"

mcp:
    port: 8080

repositories:
    - id: repo_local
      path: /path/to/repo
      name: owner/repo
```

---

## 4. Local config

```yaml
version: 1
repo_id: repo_central
repository: central/vault
vault_name: Central Knowledge Vault

include:
    - '**/*.md'

exclude:
    - .git/**
    - .obsidian/**
    - .trash/**
    - .memory/**

storage:
    engine: sqlite
    sqlite_path: memory.db

embedding:
    provider: builtin        # único provider vetorial (ADR-035)
    model: paraphrase-multilingual-MiniLM-L12-v2
    dimension: 384

search:
    mode: hybrid
    limit: 5
    k: 60
    half_life: 30
    decay_weight: 0.3

watcher:
    debounce_ms: 500
    interval_ms: 1000

editor:
    default_app: obsidian

mcp:
    port: 8080
```

### 4.1 `include` / `exclude`

The scope of the index. An empty or too-narrow `include` is the most common reason search
returns nothing.

> [!warning]
> **Exclude large volumes of foreign prose.** If the vault sits next to imported source
> material — vendored docs, generated reference pages, another language's corpus — those files
> compete with your own notes in hybrid retrieval and can dominate results. Keep raw material
> out of `include` and let your own notes be the index. This is measurable: a Health Score drop
> of 100 to 57 was traced to exactly this.

Use the narrowest `include` that covers what should be searchable, and exclude everything else
explicitly rather than relying on defaults.

---

## 5. Storage model

| Engine | When | Where the data lives |
| --- | --- | --- |
| `sqlite` | Default. Single machine, single repo, zero setup. | `<repo>/.memory/memory.db` |
| `postgres` | Central vault, multiple machines, multi-tenant. | Postgres + `pgvector` |

### 5.1 The opt-out rule

A repository that declares `storage.engine: sqlite` **discards any inherited `postgres_url`**.
This is what allows one machine to run a Postgres-backed central vault while every individual
repository keeps its own SQLite file with no extra configuration.

```
Global config: engine: postgres  ──▶  central vault (Postgres)
                                         ▲
Repo A: engine: sqlite ──▶ memory.db      │ inherited config discarded
Repo B: engine: sqlite ──▶ memory.db      │
Repo C: engine: sqlite ──▶ memory.db  ────┘
```

---

## 6. Secrets

> [!warning]
> **URL in the config file, password in an environment variable.** The connection URL is not a
> secret — it needs to be readable and shareable, and MCP clients spawn the server as a child
> process that inherits only the `env` block of their config. Keep the password in
> `MY_MEMORY_PG_PASSWORD` and never on disk.

```bash
# macOS / Linux
export MY_MEMORY_PG_PASSWORD='...'

# Windows PowerShell, user scope, survives restart
[Environment]::SetEnvironmentVariable('MY_MEMORY_PG_PASSWORD', '<senha>', 'User')
```

> [!warning]
> After creating or changing an environment variable, **restart the editor or shell**. A
> server already running keeps its inherited environment and will keep failing to authenticate
> until it is restarted.

---

## 7. Embeddings

| Provider | Model | Dimension | Use when |
| --- | --- | --- | --- |
| `builtin` | `paraphrase-multilingual-MiniLM-L12-v2` | 384 | The only vector provider. In-process, no external service. |

The dimension is **not** configurable. `internal/store/postgres.go` declares
`embedding vector(384)`, so honouring a larger value would build vectors the column
cannot hold. A config that declares any other dimension still resolves to 384.

```bash
MY_MEMORY_EMBED_MODEL=...        # parsed, ignored
MY_MEMORY_EMBED_DIM=...          # parsed, ignored — schema wins
MY_MEMORY_EMBED_BASE_URL=...     # parsed, ignored
```

> [!warning]
> These three are **inert**. They are still read into the config struct for backward
> compatibility with existing environments, but `resolveEmbedder` is authoritative on
> the builtin provider and the 384 dimension. Setting them does nothing, and there is
> no second provider left for them to select.
>
> `OLLAMA_URL` / `OLLAMA_HOST` / `OLLAMA_MODEL` remain accepted as aliases mapping onto
> those same variables. The Ollama path itself left the vector pipeline (ADR-035).

> [!warning]
> **A binary built without `-tags onnx` has no `builtin` embedder.** It does not fail loudly —
> it prints a warning and falls back to lexical-only indexing, so `mem search` still returns
> results but with **no semantic layer at all**. Exact-term queries keep working; everything that
> relied on meaning quietly stops working. If search suddenly finds nothing by concept, check
> this first.
>
> ```bash
> go build -tags onnx -o bin/mem ./cmd/mem    # include the embedder
> ```

> [!warning]
> **The dimension must match the schema.** Switching models with a different dimension
> requires a full reindex (`mem index --force`); otherwise vectors and the index disagree
> silently and retrieval quality degrades without any error.

---

## 8. Verify

```bash
mem status          # resolved config and engine
mem repos           # registered repositories
mem doctor          # index health
```
