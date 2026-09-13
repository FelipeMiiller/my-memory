# Tasks: live-indexing-and-file-watcher

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Watcher Engine | unit | Watcher lifecycle, debounce, event emission | internal/watcher/*_test.go | go test -v ./internal/watcher/... |
| Surgical Indexer | unit | IndexSingleFile (create, modify, delete) | internal/watcher/*_test.go | go test -v ./internal/watcher/... |
| CLI Watch | unit | mem watch command flags and lifecycle | cmd/mem/* | go test -v ./cmd/mem/... |
| Git Hooks | unit | mem hook install and uninstall scripts | cmd/mem/* | go test -v ./cmd/mem/... |
| CLI / ADR | none | Documentation, help outputs, ADR-017 | docs/adr/* | go test -v ./cmd/mem/... ./internal/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After watcher engine updates | go test -v ./internal/watcher/... |
| Store | After surgical indexer updates | go test -v ./internal/watcher/... ./internal/store/... |
| Full | After CLI watch and hooks | go test -count=1 -v ./cmd/mem/... ./internal/... |
| Build | After CLI init or ADR updates | go test -v ./cmd/mem/... ./internal/... && python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/live-indexing-and-file-watcher/spec.md |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T3 -> T4 -> T5 -> T6
```

### Phase 1: Motor de Monitoramento e Reindexação Cirúrgica

## Task Breakdown

### T1: Motor de Watcher e Debounce
**What**: Implementar Watcher e Debouncer em internal/watcher/watcher.go e internal/watcher/debouncer.go
**Where**: internal/watcher/watcher.go
**Depends on**: none
**Requirement**: WATCH-01, WATCH-02, WATCH-03
**Tests**: internal/watcher/watcher_test.go
**Gate**: go test -v ./internal/watcher/...
**Done when**:
- [x] Declarar struct Watcher com suporte a Start, Stop e canais de eventos
- [x] Implementar polling periódico eficiente comparando ModTime e Size de arquivos
- [x] Integrar filtragem declarativa via cfg.ShouldIndex ignorando diretórios de sistema
- [x] Implementar Debouncer consolidando eventos rápidos no mesmo caminho
- [x] Adicionar testes unitários validando detecção de criação, modificação e deleção

### T2: Reindexação Cirúrgica em Tempo Real
**What**: Implementar rotinas de indexação pontual em internal/watcher/indexer.go
**Where**: internal/watcher/indexer.go
**Depends on**: T1
**Requirement**: WATCH-04, WATCH-05
**Tests**: internal/watcher/indexer_test.go
**Gate**: go test -v ./internal/watcher/...
**Done when**:
- [ ] Implementar IndexSingleFile para SQLite e PostgreSQL
- [ ] Atualizar documento, hash SHA-256, arestas de grafo, chunks e embeddings de arquivo único
- [ ] Implementar PurgeSingleFile para limpar chunks e arestas em caso de remoção de nota
- [ ] Otimizar verificação com cache SHA-256 evitando gerar embeddings desnecessários
- [ ] Adicionar testes unitários para atualização e exclusão cirúrgica

### T3: Subcomando CLI mem watch
**What**: Adicionar subcomando mem watch com suporte a cancelamento gracioso em cmd/mem/main.go
**Where**: cmd/mem/main.go
**Depends on**: T2
**Requirement**: WATCH-06
**Tests**: cmd/mem/watch_test.go
**Gate**: go test -v ./cmd/mem/...
**Done when**:
- [ ] Adicionar case watch no switch do CLI
- [ ] Suportar flags --debounce, --interval, --db, --postgres e --repo
- [ ] Tratar sinais do sistema operacional (SIGINT/SIGTERM) para encerramento gracioso
- [ ] Atualizar printHelp documentando o comando mem watch

### T4: Gerenciador de Git Hooks
**What**: Implementar comandos mem hook install e mem hook uninstall em cmd/mem/hook.go
**Where**: cmd/mem/hook.go
**Depends on**: T3
**Requirement**: WATCH-07
**Tests**: cmd/mem/hook_test.go
**Gate**: go test -v ./cmd/mem/...
**Done when**:
- [ ] Implementar rotina para localizar diretório .git e subpasta hooks
- [ ] Gravar script pre-commit com permissões de execução executando validação e indexação
- [ ] Implementar remoção limpa do script via mem hook uninstall
- [ ] Adicionar testes unitários validando instalação e desinstalação

### T5: Integração com Auto-Scoping e Configuração Declarativa
**What**: Conectar mem watch e hooks ao arquivo .memory/config.yaml do vault
**Where**: cmd/mem/main.go
**Depends on**: T4
**Requirement**: WATCH-01, WATCH-06
**Tests**: cmd/mem/watch_test.go
**Gate**: go test -v ./cmd/mem/... ./internal/...
**Done when**:
- [ ] Utilizar auto-scoping de config.FindConfigFile quando nenhum diretório for passado em mem watch
- [ ] Carregar debounce e storage defaults a partir da configuração do vault
- [ ] Validar operação contínua do watcher em conjunto com a configuração declarativa

### T6: ADR-017, Validação Final e Documentação
**What**: Registrar decisão ADR-017 e atualizar manuais de documentação
**Where**: docs/adr/017-indexacao-continua-com-file-watcher-e-git-hooks.md
**Depends on**: T5
**Requirement**: WATCH-01, WATCH-07
**Tests**: docs/adr/017-indexacao-continua-com-file-watcher-e-git-hooks.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/live-indexing-and-file-watcher/spec.md
**Done when**:
- [ ] Criar docs/adr/017-indexacao-continua-com-file-watcher-e-git-hooks.md no formato MADR
- [ ] Atualizar docs/adr/README.md, docs/CLI_GUIDE.md, docs/REPOSITORY_BRAIN.md e README.md
- [ ] Atualizar STATE.md e gerar validation.md com veredicto PASS
