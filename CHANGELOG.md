# Changelog

All notable changes to **my-memory** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.5.0] — 2026-09-22

### Added

- **Chat LLM no viewer Electron (ADR-051)**: nova aba **Chat** com provider
  abstraction (Anthropic default + OpenAI opcional) via Vercel AI SDK,
  streaming token-a-token via IPC `webContents.send`, persistência local de
  múltiplas conversas em IndexedDB (`idb-keyval`), RAG opcional via
  `mem search` rodando como subprocess no main process.
  - API keys armazenadas em `<userData>/chat-config.json` — nunca saem do main.
  - 8 IPC channels: `mem:chat:{send,abort,config-get,config-set,mem-search,delta,done,error}`.
  - Componentes: `chat-view`, `chat-sidebar`, `chat-thread`, `chat-message`
    (MarkdownLite regex), `chat-input` (Enter envia, Shift+Enter quebra linha),
    `chat-settings-modal` (write-only API key field).
  - Zustand store com subscriptions → `idb-keyval` para auto-persistência.

- **Viewer Electron Phase 1 MVP (ADR-048)**: scaffold Electron + React 19 +
  Vite 8 + shadcn/ui v4 + Tailwind 4 + i18next + Cytoscape. Substitui os HTMLs
  legados (`graph.html`, `graph-v2.html`).

- **Code AST multi-linguagem via tree-sitter (ADR-047)**: `mem code-index`,
  `mem code-search`, `mem code-graph`, `mem code-stats` CLI; novos MCP tools
  `code_search` + `code_neighbors`; boost RRF 2× em `qualified_name`.

- **Memory writer atômico + outbox + reprojeção idempotente (ADR-044)**:
  `mem replay --since <seq> [--apply]`, projection SQLite idempotente,
  `If-Match` header + 409 conflict detection.

- **Event runtime (ADR-043)**: `event_log` table + `documents.revision`
  column + envelope canônico + dispatcher com fan-out/ACK/retry +
  redaction PII (LLM02) + audit subscriber JSONL com rotação + `mem events
  {tail,inspect,replay,trace,last-sequence,stats}` + `mem doctor --events`.

- **mymemoryd supervisor (ADR-042)**: `mem up/down/logs/profiles` + sandbox
  best-effort + heartbeat + restart com exponential backoff + profiles YAML
  com herança.

- **Nemotron 3.5 ASR streaming via ONNX Runtime Go (ADR-045)**.

- **Threat model OWASP Top 10 aplicado ao MyMemory (ADR-050)**: controles
  LLM02 (PII redaction) + LLM05 + LLM07 + LLM10 (MaxAckPending).

- **Bugfixes pós-release v1.3.0 (ADR-036)**: parser EARS + fuzzy resolve
  wikilinks + dead-link pruning + drift calibration (ADR-039) +
  `mem index` auto-scope (ADR-040).

### Changed

- **ADR-040 Accepted**: SQLite é o default (auto-scope `.memory/memory.db`);
  PostgreSQL + pgvector é opt-in via `--storage=postgres` ou env var.
- **ADR-041 Accepted**: tags sem `[[nota]]` casa são removidas do grafo
  (não viram dead links).
- **ADR-039 Accepted**: detector de drift recalibrado (PageRank + match de path).
- **ADR-037 Superseded by ADR-048**: viewer reescrito com Electron (era Vite +
  Vanilla TS proposto).
- **Documentos com frontmatter**: `documents.title` agora usa `frontmatter.title`
  quando presente (ISSUE-010).

### Security

- **API keys nunca tocam o renderer** (ADR-051): chave gravada em
  `<userData>/chat-config.json`; IPC `config-get` retorna apenas `hasApiKey: boolean`.
- **Sanitização de URLs Postgres em erros** (commit `2361a92`):
  `sanitizePostgresURL()` oculta `user:password` → `***@host` em logs.
- **ADR-048 contexto isolation**: `contextIsolation: true`, `nodeIntegration: false`,
  `sandbox: true` no Electron BrowserWindow.

### Fixed

- ISSUE-001 (dead links 8 → 0 via tag pruning)
- ISSUE-005, ISSUE-008 (drift detector miscalibration)
- ISSUE-006 (orphan count inclui by-design paths)
- ISSUE-009 (tags sem casa removidas do grafo)
- ISSUE-010 (`documents.title` usa frontmatter.title)
- `pull_request_target` workflow gotcha (nota interna — branch base sempre
  usado pelo GitHub Actions)

## [1.4.0] — prior

Baseline before the work listed above. Used `graph-v2.html` legacy viewer.
