# Tasks: vault-configuration-and-auto-scoping

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Config Engine | unit | DefaultConfig, FindConfigFile, LoadConfig, SaveConfig | internal/config/*_test.go | go test -v ./internal/config/... |
| Glob Filtering | unit | ShouldIndex with includes, excludes, windows paths | internal/config/*_test.go | go test -v ./internal/config/... |
| CLI Init | unit | mem init command execution and template creation | cmd/mem/* | go test -v ./internal/... |
| CLI Integration | unit | mem index with ShouldIndex filtering, mem search defaults | cmd/mem/* | go test -v ./internal/... |
| CLI / ADR | none | Documentation, help outputs, ADR-016 | docs/adr/* | go test -v ./internal/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | After config engine updates | go test -v ./internal/config/... |
| Store | After glob and filtering updates | go test -v ./internal/config/... ./internal/db/... |
| Full | After CLI integration | go test -count=1 -v ./internal/... |
| Build | After CLI init or ADR updates | go test -v ./internal/... && python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/vault-configuration-and-auto-scoping/spec.md |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T3 -> T4 -> T5 -> T6
```

### Phase 1: Motor de Configuração e Filtragem

## Task Breakdown

### T1: Estruturas de Configuração, Defaults e Resolução Ascendente
**What**: Implementar Config, DefaultConfig, FindConfigFile, LoadConfig e SaveConfig em internal/config/config.go
**Where**: internal/config/config.go
**Depends on**: none
**Requirement**: CFG-01, CFG-02, CFG-03
**Tests**: internal/config/config_test.go
**Gate**: go test -v ./internal/config/...
**Done when**:
- [x] Declarar structs Config, StorageConfig, EmbeddingConfig e SearchConfig
- [x] Implementar DefaultConfig com exclusões seguras e defaults de busca/storage
- [x] Implementar FindConfigFile com busca recursiva ascendente
- [x] Implementar LoadConfig e SaveConfig com parsing YAML/JSON e mescla de defaults
- [x] Adicionar testes unitários validando resolução de caminhos e mescla de valores

### T2: Motor de Filtragem de Arquivos por Regras Glob ShouldIndex
**What**: Implementar ShouldIndex com correspondência glob de include e exclude em internal/config/glob.go
**Where**: internal/config/glob.go
**Depends on**: T1
**Requirement**: CFG-04
**Tests**: internal/config/glob_test.go
**Gate**: go test -v ./internal/config/...
**Done when**:
- [x] Implementar normalização de separadores de caminho (/ e \)
- [x] Implementar ShouldIndex avaliando lista de Exclude e Include via patterns glob
- [x] Ignorar pastas de sistema por padrão (.git, node_modules, vendor, .obsidian, .trash, .memory)
- [x] Adicionar testes unitários para múltiplos cenários de inclusão e exclusão

### T3: Subcomando CLI mem init e Geração de Template
**What**: Adicionar subcomando mem init em cmd/mem/main.go para gerar .memory/config.yaml
**Where**: cmd/mem/main.go
**Depends on**: T2
**Requirement**: CFG-05
**Tests**: cmd/mem/main.go
**Gate**: go test -v ./internal/...
**Done when**:
- [x] Adicionar case init no switch do CLI
- [x] Criar diretório .memory/ e gravar config.yaml com comentários explicativos
- [x] Suportar flags --force e --repo no comando init
- [x] Atualizar printHelp documentando mem init

### T4: Integração de Config e Filtragem no mem index
**What**: Atualizar runIndexSQLite e runIndexPostgres para filtrar arquivos via cfg.ShouldIndex
**Where**: cmd/mem/main.go
**Depends on**: T3
**Requirement**: CFG-06
**Tests**: cmd/mem/main.go
**Gate**: go test -v ./internal/...
**Done when**:
- [x] Carregar configuração ativa via config.FindConfigFile(".")
- [x] Usar cfg.ShouldIndex dentro de WalkDir ignorando diretórios e arquivos excluídos
- [x] Utilizar raiz do config como diretório alvo quando nenhum argumento for passado
- [x] Validar indexação respeitando exclusões declaradas

### T5: Integração de Config Defaults no mem search e mem mcp
**What**: Utilizar preferências de busca do config como defaults para mem search e mem mcp
**Where**: cmd/mem/main.go
**Depends on**: T4
**Requirement**: CFG-06
**Tests**: cmd/mem/main.go
**Gate**: go test -v ./internal/...
**Done when**:
- [x] Utilizar valores de cfg.Search (modo, limit, k, decay, half_life) como fallbacks no flagset de busca
- [x] Utilizar configurações de backend (SQLite/PostgreSQL) e repo do config no servidor MCP
- [x] Garantir que flags explícitas de terminal mantenham prioridade máxima sobre o arquivo

### T6: ADR-016, Validação Final e Documentação
**What**: Registrar decisão ADR-016 e atualizar manuais de documentação
**Where**: docs/adr/016-configuracao-declarativa-e-auto-scoping-de-vault.md
**Depends on**: T5
**Requirement**: CFG-01, CFG-06
**Tests**: docs/adr/016-configuracao-declarativa-e-auto-scoping-de-vault.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/vault-configuration-and-auto-scoping/spec.md
**Done when**:
- [ ] Criar docs/adr/016-configuracao-declarativa-e-auto-scoping-de-vault.md no formato MADR
- [ ] Atualizar docs/adr/README.md, docs/CLI_GUIDE.md, docs/REPOSITORY_BRAIN.md e README.md
- [ ] Atualizar STATE.md e gerar validation.md com veredicto PASS
