---
title: "ADR-040: Configuração única em ~/.memory/ + SQLite auto-scope por padrão (Postgres opt-in)"
category: resource
summary: "Centraliza configuração em ~/.memory/config.yaml global. SQLite local em .memory/memory.db permanece como default por repo (auto-scope). Postgres + central vault + pgvector vira opt-in via config explícita — quando detectado, substitui o SQLite local."
tags: [architecture, federation, config, storage, central-vault, sqlite, postgres, fallback]
---

# ADR-040: Configuração única em `~/.memory/` + SQLite auto-scope por padrão (Postgres opt-in)

- **Date**: 2026-09-18
- **Status**: Proposed (revisado após clarificação do Felipe)
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: architecture, federation, config, storage, central-vault, pgvector, fallback

## Context and Problem Statement

O my-memory hoje carrega configuração em **dois lugares** simultaneamente:

1. **Local**: `.memory/config.yaml` por repositório (incluído no git, governa include/exclude e paths)
2. **Global**: `~/.memory/config.yaml` (override system-wide, catálogo de repositórios, central vault)

Esse modelo traz 3 fricções concretas:

- **Repositórios poluídos**: cada clone precisa do `.memory/` (config + DB + logs), dificultando abertura casual de issues/PRs e análise cross-repo.
- **Storage local por default mal-resolvido**: até `ec1298b` (ISSUE-004), `mem index` criava `memory.db` no CWD; agora cai em `.memory/memory.db` (auto-scope), mas a **intenção** ainda é ambígua — o usuário precisa entender se o DB é "dele" (local) ou da "infra" (compartilhado).
- **Central vault subutilizado**: PostgreSQL+pgvector já está disponível em `G:\My Drive\central-memory` mas o default ainda é SQLite local, mesmo quando o usuário já tem Postgres configurado globalmente.

A ISSUE-004 ilustrou parte do problema: o fallback de path do SQLite gravitava para CWD em vez de respeitar `.memory/`, mostrando que a **detecção automática** de onde o DB deve viver é frágil.

## Decision Drivers

- **DRY de configuração**: 1 fonte canônica por máquina, não 1 por repo + 1 global + 1 flag CLI.
- **Zero-friction out-of-the-box**: clonar repo → `mem index` funciona imediatamente, sem Postgres obrigatório.
- **Auto-scope por repo quando local**: cada repo tem seu próprio `.memory/memory.db`, sem conflito entre clones.
- **Centralização opt-in**: quem QUER cross-repo / analytics distribuído configura Postgres; quem não quer, segue com SQLite local sem ruído.
- **Não quebrar workflow atual sem aviso**: SQLite local continua funcionando; Postgres é aditivo.

## Considered Options

1. **Status quo** — `.memory/config.yaml` local + SQLite local default + global como override.
2. **Global config + SQLite local default (sem mudança de storage)** — só centraliza config; storage igual ao atual.
3. **Global config + Postgres como default** — força Postgres como única opção; SQLite vira opt-in raro.
4. **Global config + auto-fallback Postgres→SQLite** — global manda, se Postgres configurado usa Postgres, senão SQLite auto-scope. ✅ Chosen.

## Decision Outcome

Chosen option: **"4 — Global config + auto-fallback Postgres→SQLite"**, porque preserva o **zero-friction out-of-the-box** enquanto torna a centralização real e opt-in.

Concretamente:

- `.memory/config.yaml` deixa de ser obrigatório na raiz do repo. Quando presente, age como override local consciente; quando ausente, tudo vem do `~/.memory/config.yaml` global.
- **Detecção de storage na inicialização** (ordem de prioridade):
  1. Flag CLI `--db <path>` ou `--postgres <url>` → override consciente.
  2. `storage.engine` + path/url no config (global ou local) → respeitado.
  3. **`postgres` URL presente no config global** → usa Postgres+pgvector (central vault), `.memory/memory.db` neste repo é ignorado.
  4. **Caso contrário** → SQLite em `<repoDir>/.memory/memory.db` (auto-scope). `db.InitDB` cria o arquivo e o diretório se preciso.
- **Sem fallback silencioso para CWD**: se o usuário não tem config E não tem `.memory/` (vault não-inicializado), `mem init` é exigido ou erro explícito é retornado.
- **FLAG explícita pra forçar local**: `--storage=sqlite` (ou env `MY_MEMORY_FORCE_SQLITE=1`) permite usar SQLite local mesmo com Postgres configurado — útil pra CI/sandbox sem rede.

### Positive Consequences

- **Zero-friction preservado**: clonar repo + `mem init` + `mem index` continua funcionando do jeito que está hoje.
- **Repo limpo por default**: `.memory/` continua fora do git (já está em `.gitignore`), mas agora é **auto-gerado** em vez de versionado.
- **Single source of truth pra config**: `~/.memory/config.yaml` governa tudo.
- **Postgres como upgrade, não migração**: usuários que configuram Postgres no global ganham central vault automaticamente; quem não configurou não percebe diferença.
- **CI/sandbox friendly**: flag `--storage=sqlite` permite isolar um job sem tocar no vault compartilhado.

### Negative Consequences

- **DB local ainda existe em todo repo**: quem tem muitos clones tem N DBs SQLite. Mitigação: TTL de inatividade + cleanup via `mem prune` (já existente, fora do escopo).
- **Detecção de "Postgres presente?" tem que ser explícita e rápida**: precisa de cache do resolved config pra não re-parsear YAML a cada chamada.
- **Risco de split-brain**: usuário que tem Postgres configurado globalmente mas esquece de commit e roda `mem index` em outro clone → dados vão pro vault sem aviso. Mitigação: log/warning visível ao detectar Postgres.
- **Configurar Postgres passa a ser responsabilidade do setup inicial**: onboarding precisa de docs claras pra "quero central vault" vs "quero só local".

## Pros and Cons of the Options

### Option 1 — Status quo ✅ Baseline

- ✅ Zero atrito de setup: `mem init` cria tudo local.
- ✅ Funciona offline por default.
- ❌ Repo poluído com `.memory/` (config versionada).
- ❌ Config duplicada em N lugares.
- ❌ Postgres fica enterrado, subutilizado.

### Option 2 — Só centraliza config (storage igual ao atual)

- ✅ Config centralizada (resolve driver 1).
- ✅ Comportamento atual preservado.
- ✅ Zero risco de regressão.
- ❌ Não resolve driver 3 (central vault continua subutilizado).
- ❌ Detecção ambígua de "qual DB usar" continua.

### Option 3 — Postgres default (proposto na 1ª iteração do ADR)

- ✅ Cross-repo nativo por default.
- ✅ Força modernização do setup.
- ❌ Exige Postgres+pgvector operacional antes de qualquer uso — quebra zero-friction.
- ❌ Migração de quem já tinha SQLite funcionando.
- ❌ Rejeitado após clarificação do Felipe: o fallback deve ser SQLite local.

### Option 4 — Global config + auto-fallback Postgres→SQLite ✅ Chosen

- ✅ Config centralizada.
- ✅ Zero-friction preservado (SQLite auto-scope por default).
- ✅ Postgres vira upgrade opt-in (não migração forçada).
- ✅ Repo limpo: `.memory/` é auto-gerado, não versionado.
- ❌ Detecção "Postgres presente?" precisa ser explícita e cacheada.
- ❌ Risco de split-brain sem aviso (mitigável com log visível).

## Detection Logic (ordem de resolução)

```go
func resolveStorage(repoCtx RepoContext) StorageDecision {
    // 1. Override consciente via flag/env
    if db := os.Getenv("MY_MEMORY_FORCE_SQLITE"); db != "" {
        return SQLiteLocal{Path: deriveRepoLocalPath(repoCtx)}
    }
    if pg := os.Getenv("MY_MEMORY_FORCE_POSTGRES"); pg != "" {
        return PostgresRemote{URL: pg}
    }

    // 2. Config global/local explícito
    cfg := loadCascadingConfig(repoCtx)
    if cfg.Storage.Engine == "postgres" && cfg.Storage.PostgresURL != "" {
        return PostgresRemote{URL: cfg.Storage.PostgresURL}
    }
    if cfg.Storage.Engine == "sqlite" && cfg.Storage.SQLitePath != "" {
        return SQLiteLocal{Path: cfg.Storage.SQLitePath}
    }

    // 3. Heurística: Postgres configurado no global? → central vault
    if globalCfg := loadGlobalConfig(); globalCfg != nil && globalCfg.Storage.PostgresURL != "" {
        log.Info("Postgres detectado no global config, usando central vault")
        return PostgresRemote{URL: globalCfg.Storage.PostgresURL}
    }

    // 4. Fallback gracioso: SQLite auto-scope por repo
    return SQLiteLocal{Path: filepath.Join(repoCtx.RepoDir, ".memory", "memory.db")}
}
```

## Implementation Plan (high-level)

1. **`internal/config/config.go::LoadCascadingConfig`** — refatorar pra deixar `storage.sqlite_path` vazio por default quando YAML local não declara; a heurística mora no storage resolver.
2. **`cmd/mem/main.go::resolveStorageAndRepo`** — implementar a `Detection Logic` acima; remover fallback CWD que ficou obsoleto pelo auto-scope.
3. **`cmd/mem/main.go`** — adicionar flag `--storage=sqlite|postgres` para override consciente.
4. **`cmd/mem/init.go`** — criar `.memory/` (gitignored) automaticamente quando SQLite vai ser usado; detectar vault central via `~/.memory/config.yaml` antes de sugerir setup local.
5. **`internal/logger`** — log visível quando Postgres é detectado e SQLite local é ignorado (anti-split-brain).
6. **Documentação**:
   - `docs/CLI_GUIDE.md` — atualizar seção de `--db`/`--postgres` com nova flag `--storage`.
   - `docs/AGENT_INTEGRATION_GUIDE.md` — exemplos pra "vou usar central vault" vs "vou usar SQLite local".
   - `docs/REPOSITORY_BRAIN.md` — atualizar diagrama de arquitetura.
   - `.agents/skills/prova-md/SKILL.md` e `resumo-materias/SKILL.md` — alinhar com novo fluxo.
7. **`.gitignore`** — garantir `.memory/` continua ignorado (já está).
8. **Script de migração opcional** — pra quem quiser mover `.memory/config.yaml` local pra `~/.memory/config.yaml`.

## Validation Criteria

- `mem init` em repo novo sem `.memory/` → cria `.memory/config.yaml` mínimo OU usa global.
- `mem index` sem flag → DB criado em `.memory/memory.db` (auto-scope). Smoke test E2E.
- `mem index --postgres <url>` → conecta no Postgres, dados vão pro vault.
- Com `~/.memory/config.yaml` apontando Postgres + sem `--db`: log "Postgres detectado, usando central vault"; DB criado no Postgres, não em `.memory/`.
- `mem index --storage=sqlite --postgres <url>` (forçar local mesmo com global Postgres) → DB criado em `.memory/memory.db`, ignora global.
- Sem `.memory/`, sem global, sem flag → erro explícito pedindo `mem init`.

## Links

- ISSUE-004 — `mem index` sem `--db` criava DB em CWD (corrigida em `ec1298b`, base conceitual deste ADR)
- ADR-007 — Suporte opcional a PostgreSQL com pgvector e multi-repositório
- ADR-016 — Configuração Declarativa e Auto-Scoping de Vault (`.memory/config.yaml`)
- ADR-033 — Federated Central Vault and Repo Identity
- ADR-034 — Protocolo Canônico Federado e Wikilinks Cross-Vault
- ADR-036 — Consolidação Pós-Release v1.3.0 e Roadmap v1.4.0 (cita central vault como prioridade)

---

**Supersedes**: ADR-016 (parcialmente — auto-scoping de vault local dá lugar a centralização global + opt-in Postgres)
**Status note**: Proposed — revisado após clarificação do Felipe (fallback gracioso SQLite quando Postgres ausente). Aguardando aprovação final antes de implementar.
