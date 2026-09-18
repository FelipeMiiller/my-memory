---
title: "Spec 040: Implementação ADR-040 — Config global única + SQLite auto-scope + Postgres opt-in"
category: resource
summary: "Implementação do ADR-040. Centraliza config em ~/.memory/config.yaml, mantém SQLite auto-scope em .memory/memory.db por repo como default, e torna Postgres opt-in via flag --postgres ou storage.engine no config."
tags: [spec, adr-040]
---

# Spec 040 — Implementação ADR-040

**Status**: in-progress
**ADR de referência**: [ADR-040](../../docs/adr/040-config-global-unica-storage-opt-in.md) (Accepted)
**Issue relacionada**: ISSUE-004 (base conceitual via auto-scope, `ec1298b`)

## Escopo

Implementar a matriz de detecção de storage do ADR-040:

| Cenário | Comportamento esperado |
|---|---|
| Sem `.memory/`, sem global, sem flag | Erro explícito pedindo `mem init` |
| Sem `.memory/`, com global sem Postgres | `mem init` cria `.memory/` (gitignored), SQLite auto-scope |
| Com `.memory/` mas sem config Storage | SQLite auto-scope em `<repo>/.memory/memory.db` (já funciona pós ISSUE-004) |
| `storage.engine: postgres` + URL no global | Postgres+pgvector (central vault) |
| Flag `--postgres <url>` | Postgres+pgvector (override consciente) |
| Flag `--storage=sqlite` mesmo com Postgres global | SQLite local, ignora global (CI/sandbox) |
| Flag `--db <path>` | Override consciente para path específico |

## Fora do escopo (neste spec)

- Migration script pra mover `.memory/config.yaml` local → `~/.memory/config.yaml` global (será spec 041)
- Implementação de TTL/cleanup de DBs inativos
- Cache local de embeddings quentes (perf)
- Mudanças no protocolo federation (ADR-034)

## Critérios de aceitação (EARS)

- **EARS-1**: When o usuário roda `mem index` sem nenhuma flag e sem config Postgres, the system MUST criar/usar `<repoDir>/.memory/memory.db` (auto-scope).
- **EARS-2**: When `~/.memory/config.yaml` declara `storage.engine: postgres` com URL válida, the system MUST conectar no Postgres+pgvector e ignorar SQLite local.
- **EARS-3**: When o usuário passa `--postgres <url>`, the system MUST conectar no Postgres e ignorar storage.engine do config.
- **EARS-4**: When o usuário passa `--storage=sqlite` mesmo com Postgres global, the system MUST usar SQLite local e emitir log "SQLite local forçado via flag --storage".
- **EARS-5**: When o usuário passa `--storage=postgres` sem URL, the system MUST usar a URL do config ou erro explícito.
- **EARS-6**: When nenhuma config e `.memory/` ausente, `mem index` MUST retornar erro pedindo `mem init` (sem fallback silencioso CWD).
- **EARS-7**: When Postgres é detectado (config global) e SQLite local existe, the system MUST emitir log visível "Postgres detectado no global, usando central vault; SQLite local em <path> será ignorado".

## Tarefas

Ver `tasks.md`.

## Validação

Ver `validation.md`.
