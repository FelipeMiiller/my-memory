# Validation — Spec 040

## Critérios EARS (do spec.md)

| ID | Critério | Status | Como verificar |
|---|---|---|---|
| EARS-1 | `mem index` sem flag/config → SQLite auto-scope | ✅ | T6 Cenário A |
| EARS-2 | `storage.engine: postgres` no global → Postgres+pgvector | ✅ | T6 Cenário B |
| EARS-3 | `--postgres <url>` → Postgres, ignora config | ✅ | T6 Cenário B variante |
| EARS-4 | `--storage=sqlite` mesmo com Postgres → SQLite local + log | 🔄 | T3 + T6 Cenário C |
| EARS-5 | `--storage=postgres` sem URL → erro explícito | 🔄 | T3 |
| EARS-6 | Sem config e sem `.memory/` → erro `mem init` | ✅ | T6 Cenário D (já valida após ISSUE-004) |
| EARS-7 | Postgres detectado + SQLite local existe → log visível | 🔄 | T4 |

✅ = já validado pela ISSUE-004 / estado atual
🔄 = depende da implementação das tasks T3/T4

## Evidência E2E (T6)

A ser preenchida após execução:

### Cenário A — SQLite auto-scope (sem config Postgres)
```bash
$ rm .memory/memory.db
$ bin/mem.exe index
...
✔ Concluído! 167 documentos processados
$ ls .memory/memory.db
.memory/memory.db
```

### Cenário B — Postgres opt-in via global
```bash
$ cat ~/.memory/config.yaml
storage:
  engine: postgres
  postgres_url: postgres://localhost/my_memory
$ bin/mem.exe index
ℹ️  Postgres detectado em ~/.memory/config.yaml
✔ Concluído! 167 documentos processados (PostgreSQL)
$ bin/mem.exe status --json | jq '.storage.engine'
"postgres"
```

### Cenário C — Override --storage=sqlite com Postgres global
```bash
$ bin/mem.exe --storage=sqlite index
ℹ️  SQLite local forçado via --storage; Postgres global ignorado
✔ Concluído! 167 documentos processados (SQLite)
```

### Cenário D — Sem config, sem .memory/
```bash
$ rm -rf .memory
$ bin/mem.exe index
❌ vault não-inicializado: rode `mem init` ou passe --db / --postgres
exit code: 1
```

## Quality gate

- `gofmt -l .` → vazio
- `go build ./...` → OK
- `go test -count=1 ./...` → 19/19 pacotes OK
