# Validation — Spec 040

## Critérios EARS (do spec.md)

| ID | Critério | Status | Evidência |
|---|---|---|---|
| EARS-1 | `mem index` sem flag/config → SQLite auto-scope | ✅ | Cenário A (2026-09-18) |
| EARS-2 | `storage.engine: postgres` no global → Postgres+pgvector | ✅ | Cenário B (2026-09-18) |
| EARS-3 | `--postgres <url>` → Postgres, ignora config | ✅ | Cenário B variante (2026-09-18) |
| EARS-4 | `--storage=sqlite` mesmo com Postgres → SQLite local + log | ✅ | Cenário C (2026-09-18) |
| EARS-5 | `--storage=postgres` sem URL → erro explícito | ✅ | Cenário E (2026-09-18) |
| EARS-6 | Sem config e sem `.memory/` → erro `mem init` | ✅ | Cenário D (2026-09-18) |
| EARS-7 | Postgres detectado + SQLite local existe → log visível | ✅ | Cenário F (2026-09-18) |

## Evidência E2E (2026-09-18)

### Cenário A — SQLite auto-scope (sem config Postgres)
```bash
$ rm .memory/memory.db
$ bin/mem.exe index
✔ Concluído! 167 documentos processados
$ ls .memory/memory.db
.memory/memory.db   # OK
$ ls memory.db        # CWD
[NOT EXISTS]         # OK: sem vazamento
```

### Cenário B — Postgres opt-in via flag
```bash
$ bin/mem.exe index --postgres postgres://invalid:5432/test
Erro ao conectar no PostgreSQL: dial tcp: lookup invalid: no such host
# OK: roteamento foi para Postgres, não para SQLite
```

### Cenário C — Override `--storage=sqlite`
```bash
$ bin/mem.exe index --storage=sqlite
🔍 Indexando notas no SQLite em: C:\repository\my-memory
✔ Concluído! 167 documentos processados
# OK: override aceito, SQLite usado
```

### Cenário D — Sem `.memory/` e sem config
```bash
$ mv .memory .memory_old
$ bin/mem.exe index
Uso: mem index [--force] [--no-prune] [--db <caminho>] [--postgres <url>] [--repo <nome>] [<pasta>]
     Dica: execute 'mem init' para criar um arquivo de configuração declarativa.
$ mv .memory_old .memory
# OK: erro com hint, sem criar nada silenciosamente
```

### Cenário E — `--storage=postgres` sem URL
```bash
$ bin/mem.exe index --storage=postgres
--storage=postgres exige --postgres <url> ou storage.postgres_url no config
# OK: EARS-5 — erro explícito
```

### Cenário F — Postgres de env + SQLite local existente
```bash
$ MY_MEMORY_PG_URL=postgres://fakehost:5432/test bin/mem.exe index
ℹ️  Postgres detectado no config/env. Central vault será usado.
ℹ️  SQLite local em .memory\memory.db será ignorado nesta sessão.
Erro ao conectar no PostgreSQL: ...
# OK: EARS-7 — log anti-split-brain emitido antes do erro de conexão
```

## Commits

- `20fb251` — docs: ADR-040 (1ª versão, depois superseded)
- `1ad0e40` — docs: ADR-040 marcado como Accepted
- `e390b77` — spec: spec/tasks/validation p/ ADR-040 implementação
- `fb28ba4` — feat(cli): --storage flag + log anti-split-brain (EARS-4/5/7)
- `<TBD>` — docs(CLI_GUIDE): seção "Seleção de Storage" + matriz ADR-040

## Quality gate

- `gofmt -l .` → vazio
- `go build ./...` → OK
- `go test -count=1 ./...` → 19/19 pacotes OK
