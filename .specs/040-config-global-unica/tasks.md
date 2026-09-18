# Tasks — Spec 040 (ADR-040 implementação)

## T1 — Verificar cascade global→local já funciona (ISSUE-004 base)
**EARS coberto**: EARS-1, EARS-6

- [x] ISSUE-004 já implementa auto-scope para `<repoDir>/.memory/memory.db` (commit `ec1298b`)
- [x] `LoadCascadingConfig` promove default CWD `memory.db` para `<repoDir>/.memory/memory.db` quando `.memory/` existe
- [x] Testes `TestResolveStorageAndRepo_DBPathFallback` e `TestLoadCascadingConfig_*` passam

**Verificação**: rodar `go test -count=1 ./internal/config/ ./cmd/mem/`

## T2 — Verificar detecção Postgres já funciona
**EARS coberto**: EARS-2, EARS-3

- [x] `cmd/mem/main.go` linha ~123+: quando `--postgres` flag OU `cfg.Storage.Engine == "postgres"`, conecta no Postgres+pgvector
- [x] `cfg.Storage.PostgresURL` já é populado do global+env (`MY_MEMORY_PG_URL`, `POSTGRES_URL`, `DATABASE_URL`)

**Verificação**: smoke test com `MY_MEMORY_PG_URL` apontando pra Postgres local

## T3 — Adicionar flag `--storage=sqlite|postgres` para override consciente
**EARS coberto**: EARS-4, EARS-5

- [ ] Adicionar `--storage` flag nos subcomandos: `index`, `search`, `doctor`, `mcp`, `export`, `hubs`, `insights`, `watch`, `status`
- [ ] Quando `--storage=sqlite`: força SQLite local mesmo se Postgres no global; log "SQLite local forçado via --storage"
- [ ] Quando `--storage=postgres`: exige URL no config OU flag `--postgres`; erro explícito se faltar
- [ ] Adicionar testes em `cmd/mem/resolve_storage_test.go`

## T4 — Log visível quando Postgres é detectado (anti-split-brain)
**EARS coberto**: EARS-7

- [ ] Em `resolveStorageAndRepo`, quando detectar Postgres no global E SQLite local existe, emitir log:
  ```
  ℹ️  Postgres detectado em ~/.memory/config.yaml (storage.engine=postgres)
  ℹ️  Central vault será usado; SQLite local em <path> será ignorado nesta sessão
  ```

## T5 — Atualizar docs
- [ ] `docs/CLI_GUIDE.md`: adicionar seção "Storage selection" com matriz de cenários
- [ ] `docs/AGENT_INTEGRATION_GUIDE.md`: atualizar exemplos que assumem `.memory/config.yaml` local
- [ ] `docs/REPOSITORY_BRAIN.md`: diagrama de arquitetura atualizado

## T6 — Smoke test E2E (todos os cenários do ADR)
**EARS coberto**: EARS-1 a EARS-7

- [ ] Cenário A: sem config, com `.memory/` → DB em `<repoDir>/.memory/memory.db`
- [ ] Cenário B: global com Postgres → conecta no Postgres, ignora local
- [ ] Cenário C: `--storage=sqlite --postgres X` → SQLite local, ignora Postgres
- [ ] Cenário D: sem `.memory/`, sem global, sem flag → erro pedindo `mem init`

## T7 — Quality gate + commits atômicos

- [ ] `gofmt -l .` vazio
- [ ] `go build ./...` OK
- [ ] `go test -count=1 ./...` 19/19 pacotes OK
- [ ] Commits separados por fase (T3, T4, T5, T6)
