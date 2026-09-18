---
title: "ADR-040: Configuração única em ~/.memory/ + SQLite opt-in"
category: resource
summary: "Centraliza toda a configuração de storage, indexação e busca em ~/.memory/config.yaml global. SQLite deixa de ser default e vira opt-in (apenas quando explicitamente configurado). Default passa a ser central vault + pgvector remoto."
tags: [architecture, federation, config, storage, central-vault, sqlite, postgres]
---

# ADR-040: Configuração única em `~/.memory/` + SQLite opt-in

- **Date**: 2026-09-18
- **Status**: Proposed
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: architecture, federation, config, storage, central-vault, pgvector

## Context and Problem Statement

O my-memory hoje carrega configuração em **dois lugares** simultaneamente:

1. **Local**: `.memory/config.yaml` por repositório (incluído no git, governa include/exclude e paths)
2. **Global**: `~/.memory/config.yaml` (override system-wide, catálogo de repositórios, central vault)

Esse modelo traz 3 fricções concretas:

- **Repositórios poluídos**: cada clone precisa do `.memory/` (config + DB + logs), dificultando abertura casual de issues/PRs e análise cross-repo.
- **Storage local por default**: `mem index` cria `memory.db` no repo mesmo quando o vault central está disponível — duplicação silenciosa de dados e split-brain entre local e central.
- **Multitenancy limitado**: como SQLite vive no repo, queries cross-repo precisam de federation explícita (ADR-033/034), em vez de serem nativas via Postgres+pgvector.

A ISSUE-004 (corrigida em `ec1298b`) ilustrou parte do problema: o fallback de path do SQLite ainda gravitava para CWD em vez de respeitar `.memory/`, evidenciando que a localização do storage é mal-definida na ausência de config explícita.

## Decision Drivers

- **DRY de configuração**: 1 fonte canônica por máquina, não 1 por repo + 1 global + 1 flag CLI.
- **Workspace limpo**: zero `.memory/` commitável no repo (a não ser override consciente).
- **Cross-repo nativamente**: queries, embeddings e grafos compartilháveis entre repos sem adapter.
- **SQLite só onde faz sentido**: standalone offline, CI sem rede, sandbox descartável. Default deve ser a infra compartilhada.
- **Não quebrar workflow atual sem aviso**: migração tem de ser opt-in e reversível.

## Considered Options

1. **Status quo** — `.memory/config.yaml` local + SQLite local default + global como override.
2. **Global config + SQLite local default** — centraliza config mas mantém SQLite como default de storage.
3. **Global config + central vault default (PostgreSQL + pgvector)** — config 100% em `~/.memory/`, SQLite vira opt-in via flag explícita.
4. **Central vault como única config + storage** — centraliza config E storage no vault (Postgres+pgvector).

## Decision Outcome

Chosen option: **"3 — Global config + central vault default (PostgreSQL + pgvector)"**, porque atende os 3 drivers sem impor o overhead do option 4 (vault como única config) e elimina as fricções do option 1 (split de config + SQLite local padrão).

Concretamente:

- `.memory/config.yaml` deixa de ser necessário na raiz do repo. Repositório fica 100% livre de `.memory/` por default.
- Toda configuração vive em `~/.memory/config.yaml` (global), com cascade via CLI flags quando necessário.
- Default de storage vira **central vault + PostgreSQL+pgvector** (já disponível em `G:\My Drive\central-memory`).
- SQLite só é usado quando:
  - Flag `--db` explícita, OU
  - `storage.engine: sqlite` + `storage.sqlite_path` no config global, OU
  - Repo explícito declara override local via `.memory/config.yaml` (caminho de escape).
- Detecção de ausência do SQLite local deixa de existir — não há fallback silencioso para CWD ou `.memory/memory.db` por default.

### Positive Consequences

- **Repo limpo**: clone de qualquer repositório my-memory é só código. `.memory/`some do gitignore por default.
- **Single source of truth**: `~/.memory/config.yaml` governa tudo — include/exclude, paths, central vault, embeddings.
- **Cross-repo queries nativas**: pgvector indexa embeddings de múltiplos repos simultaneamente; grafo distribuído sem adapter.
- **SQLite preservado como ferramenta certa**: continua sendo a escolha certa para CI, sandbox e uso offline; só não é mais o padrão cego.
- **Alinhamento com ADRs-033/034**: federation deixa de ser código defensivo e vira caminho natural.

### Negative Consequences

- **Operação obrigatória de Postgres+pgvector**: sem ele, o my-memory não funciona out-of-the-box. Setup inicial mais pesado.
- **Ponto único de falha**: queda do central vault paralisa todos os repos federados. Mitigação: read-replica + backup automatizado.
- **Migração**: repos existentes precisam migrar `.memory/config.yaml` → `~/.memory/` ou declarar override. Script de migração será necessário.
- **Latência de rede**: queries no vault remoto são mais lentas que SQLite local (~5-20ms). Mitigação: cache local de embeddings+metadados quentes (fora do escopo deste ADR).
- **DX offline**: trabalhar sem rede (avião, café sem wifi) exige setup explícito de SQLite local antes.

## Pros and Cons of the Options

### Option 1 — Status quo ✅ Baseline

- ✅ Zero atrito de setup: `mem init` cria tudo local.
- ✅ Funciona offline por default.
- ❌ Repo poluído com `.memory/`.
- ❌ Config duplicada em N lugares.
- ❌ Cross-repo precisa de federation explícita.

### Option 2 — Global config + SQLite default

- ✅ Config centralizada (resolve driver 1).
- ✅ Repo limpo.
- ✅ Funciona offline (mantém SQLite local).
- ❌ SQLite local ainda fragmenta dados — não resolve driver 2.
- ❌ Split-brain continua possível (local vs central).

### Option 3 — Global config + central vault default ✅ Chosen

- ✅ Config 100% centralizada (driver 1).
- ✅ Repo limpo (driver 2).
- ✅ Dados unificados por default (driver 2).
- ✅ SQLite preservado como opt-in (driver 4).
- ❌ Exige Postgres+pgvector operacional (driver 4 mitigado).
- ❌ Migração de repos existentes (driver 5 mitigado com script).
- ❌ Latência de rede vs SQLite local.

### Option 4 — Central vault como única config + storage

- ✅ Máxima centralização.
- ✅ Repo 100% livre de qualquer vestígio de config.
- ❌ Setup inicial mais complexo (vault tem que conhecer cada repo).
- ❌ Falha do vault = impossível até criar nova config.
- ❌ Migration friction maior que option 3.

## Implementation Plan (high-level)

1. **`cmd/mem/init.go`**: detectar presença de `.memory/` no repo → sugerir migração para global; não criar nada local por default.
2. **`cmd/mem/main.go::resolveConfig`**: priorizar global; local só se explicitamente solicitado.
3. **`cmd/mem/main.go::resolveStorageAndRepo`**: rever o fix da ISSUE-004 (`ec1298b`) — em vez de fallback `.memory/memory.db`, exigir config explícita ou erro de setup.
4. **`internal/config/config.go`**: refatorar `LoadCascadingConfig` para assumir global como source-of-truth; local como override opt-in.
5. **`docs/CLI_GUIDE.md`**: documentar SQLite opt-in (`--db`, `MY_MEMORY_PG_URL`, `storage.engine`).
6. **`docs/AGENT_INTEGRATION_GUIDE.md`**: atualizar exemplos que assumem `.memory/config.yaml` local.
7. **Script de migração**: `.agents/scripts/migrate-config-to-global.sh` para mover configs locais para global.
8. **`docs/REPOSITORY_BRAIN.md`**: atualizar diagrama de arquitetura.

## Links

- ISSUE-004 — `mem index` sem `--db` criava DB em CWD (corrigida em `ec1298b`, base conceitual deste ADR)
- ADR-007 — Suporte opcional a PostgreSQL com pgvector e multi-repositório
- ADR-016 — Configuração Declarativa e Auto-Scoping de Vault (`.memory/config.yaml`)
- ADR-033 — Federated Central Vault and Repo Identity
- ADR-034 — Protocolo Canônico Federado e Wikilinks Cross-Vault
- ADR-036 — Consolidação Pós-Release v1.3.0 e Roadmap v1.4.0 (cita central vault como prioridade)

---

**Supersedes**: ADR-016 (parcialmente — Auto-Scoping de vault local dá lugar a centralização global)
**Status note**: Proposed — aguardando aprovação do Felipe antes de implementar.
