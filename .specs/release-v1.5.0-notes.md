# my-memory v1.5.0

**Released:** 2026-09-22 · **Tag:** `v1.5.0` · **Commit:** `ba4d14f`

## 🚀 Highlights

### Chat LLM nativo no viewer Electron (ADR-051)

Adiciona uma aba **Chat** ao viewer Electron com:

- **Multi-provider** via [Vercel AI SDK](https://ai-sdk.dev/) — Anthropic (Claude Opus/Sonnet/Haiku 4.x) default + OpenAI (GPT-4o, o3, o3-mini) opcional
- **Streaming token-a-token** via IPC `webContents.send('mem:chat:delta', …)` — latência 5-15ms por hop
- **Persistência local** de múltiplas conversas em IndexedDB via `idb-keyval` (~600B) — sobrevive a restart
- **RAG opcional** via `mem search --json` rodando como subprocess no main process — top-5 snippets injetados no system prompt
- **API keys seguras** — gravadas em `<userData>/chat-config.json`; renderer vê apenas `hasApiKey: boolean`

### Viewer Electron Phase 1 MVP (ADR-048)

Substitui os HTMLs legados (`graph.html`, `graph-v2.html`) por um app desktop nativo:

- **Electron Forge 7** + **React 19** + **Vite 8** + **shadcn/ui v4** + **Tailwind 4**
- CSP strict planejado (ADR-049) na F4 do Phase 2
- IPC tipado entre main e renderer via `contextBridge`
- Bundle zero-CDN (fecha ADR-050 LLM03)

### Code AST multi-linguagem via tree-sitter (ADR-047)

- Novos CLI: `mem code-index`, `mem code-search`, `mem code-graph`, `mem code-stats`
- Novos MCP tools: `code_search` + `code_neighbors`
- Boost RRF 2× em `qualified_name` match
- Pipeline opt-in via `--tags treesitter` (Plano A deferido — ver ADR-047 §Deferral)

### Memory Writer Atômico + Outbox (ADR-044)

- `mem replay --since <seq> [--apply]`
- Projection SQLite idempotente (subscriber resiliante)
- `If-Match` header + 409 conflict detection
- Política de escrita com 3 profiles (`strict`, `balanced`, `permissive-dev`)

### Event Runtime (ADR-043)

- Tabela `event_log` + coluna `documents.revision`
- Envelope canônico com JSON marshal + validação
- Dispatcher com fan-out, ACK/NACK, retry, panic recovery
- Audit subscriber JSONL com rotação
- Redaction PII (LLM02 gate)
- Novos CLI: `mem events {tail,inspect,replay,trace,last-sequence,stats}` + `mem doctor --events`

### mymemoryd Supervisor (ADR-042)

- `mem up/down/logs/profiles` subcommands
- Sandbox best-effort + audit log JSONL com rotação
- Worker lifecycle + heartbeat + restart com exponential backoff
- Profiles YAML com herança + required/optional semantics

### Threat Model OWASP LLM (ADR-050)

Controles aplicados ao MyMemory:

- **LLM02** (PII/Sensitive Disclosure) — redaction rules + audit subscriber
- **LLM05** (Supply Chain) — bundle zero-CDN, dependencies pinned
- **LLM07** (Insecure Output Handling) — sanitização de URLs Postgres em logs
- **LLM10** (Model Theft) — MaxAckPending limit + audit

### Nemotron 3.5 ASR Streaming (ADR-045)

- In-process via ONNX Runtime Go (`yalue/onnxruntime_go`)
- Plano B deferido: Python sidecar ASR (gatilhos em ADR-045 §Deferral)

## 📊 Stats

- **80+ commits** desde v1.4.0
- **50+ ADRs** registrados (001-051, com 11 Accepted)
- **22+ features** shipped
- **Multi-storage**: SQLite (default) + PostgreSQL/pgvector (opt-in via `--storage=postgres`, ADR-040)

## 📥 Instalação

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/FelipeMiiller/my-memory/v1.5.0/scripts/install.ps1 | iex
```

### Linux/macOS

```bash
curl -fsSL https://raw.githubusercontent.com/FelipeMiiller/my-memory/v1.5.0/scripts/install.sh | sh
```

### Go toolchain

```bash
go install github.com/FelipeMiiller/my-memory/cmd/mem@v1.5.0
```

## ⚠️ Notas de upgrade

- v1.5.0 introduz mudança **breaking** no viewer (HTMLs legados removidos em `14c6712`). Use o Electron viewer a partir de agora.
- Auto-scoping do SQLite em `.memory/memory.db` (ADR-040) — comportamento legado `mem index` em CWD removido em `ec1298b`.
- API keys de LLM (Chat feature) ficam em `<userData>/chat-config.json` — primeira execução do chat pede para configurar.

## 🐛 Bugfixes

- ISSUE-001: dead links 8 → 0 (tag pruning + wrap `#Tags` inline code)
- ISSUE-005 + ISSUE-008: drift detector miscalibration (ADR-039 recalibrado)
- ISSUE-006: orphan count excluindo by-design paths
- ISSUE-009: tags sem casa removidas do grafo (ADR-041)
- ISSUE-010: `documents.title` usa `frontmatter.title`

## 📚 Documentação

- [CHANGELOG.md](https://github.com/FelipeMiiller/my-memory/blob/v1.5.0/CHANGELOG.md) — entry completo
- [README.md](https://github.com/FelipeMiiller/my-memory/blob/v1.5.0/README.md) — quickstart
- [docs/CLI_GUIDE.md](https://github.com/FelipeMiiller/my-memory/blob/v1.5.0/docs/CLI_GUIDE.md) — manual de comandos
- [docs/VIEWER.md](https://github.com/FelipeMiiller/my-memory/blob/v1.5.0/docs/VIEWER.md) — viewer desktop
- [docs/ARCHITECTURE.md](https://github.com/FelipeMiiller/my-memory/blob/v1.5.0/docs/ARCHITECTURE.md) — schemas DDL
- [docs/TURBOQUANT.md](https://github.com/FelipeMiiller/my-memory/blob/v1.5.0/docs/TURBOQUANT.md) — matemática 4-bit
- [docs/REPOSITORY_BRAIN.md](https://github.com/FelipeMiiller/my-memory/blob/v1.5.0/docs/REPOSITORY_BRAIN.md) — integração MCP
- `.agents/skills/my-memory/SKILL.md` — canônica de uso

## 🤝 Créditos

- Felipe Miiller — owner + dev principal
- Mavis (orchestrator) — execução, validação, ADRs
- Comunidade open source: Vercel (AI SDK), Anthropic (Claude), tree-sitter, SQLite, pgvector, i18next, shadcn/ui

---

Full Changelog: <https://github.com/FelipeMiiller/my-memory/compare/v1.4.0...v1.5.0>
