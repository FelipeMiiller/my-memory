# Tasks: repo-id-alignment

**Spec**: [spec.md](./spec.md)
**Date**: 2026-10-29
**Issue**: ISSUE-028 (`.specs/ISSUES.md:945`)

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Config (leitura do repo_id) | unit | `repo_id` lido do vault do path registrado; path sem config → `os.ErrNotExist`; **sem** subir para os diretórios pais; variante JSON | internal/config/config_test.go | go test -count=1 ./internal/config/... |
| Cascateamento (caracterização) | unit | Com id alinhado, o override de storage do catálogo sobrevive à mudança de path; `RegisterRepositoryInGlobalConfig` realinha o id pelo path e preserva o bloco `storage:` | internal/config/config_test.go | go test -count=1 ./internal/config/... |
| CLI `mem repos` (regressão) | unit | Divergência → erro explícito + `❌` + exit ≠ 0; alinhado → `✅` + exit 0; sem vault → `⚠️` sem erro; vault sem `repo_id` → sem erro; auditoria read-only | cmd/mem/setup_test.go | go test -count=1 ./cmd/mem/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após mexer no helper de config | go test -count=1 ./internal/config/... |
| Quick | Após mexer no comando `mem repos` | go test -count=1 ./cmd/mem/... |
| Full | Após a integração completa | go test -count=1 ./... |
| Build | Antes de declarar pronto | go build ./... && go vet ./... |
| Format | Antes de declarar pronto | gofmt -l na cópia LF dos arquivos tocados (o working tree local está em CRLF) |
| Spec | Antes de entregar | python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/repo-id-alignment/spec.md && python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py .specs/features/repo-id-alignment/tasks.md |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2
T3 -> T5
T4 -> T5
```

| Fase | Tarefas | Objetivo |
| ---- | ------- | -------- |
| 1 | T1, T2 | Leitura do `repo_id` do vault e auditoria de identidade no `mem repos` |
| 2 | T3, T4, T5 | Regressão do comando, caracterização do cascateamento e gate final |

## Task Breakdown

### Phase 1: Leitura do repo_id e auditoria do catálogo

### T1: Leitura read-only do repo_id do vault
**What**: Adicionar `config.LoadVaultRepoID(repoDir)`, que lê o arquivo de configuração do vault **restrito ao path informado** (sem subir para os diretórios pais) e devolve o `repo_id` declarado; devolver `os.ErrNotExist` quando não há config, e `repo_id` vazio quando o config existe mas não declara o campo
**Where**: internal/config/config.go
**Depends on**: none
**Requirement**: REPOID-04, REPOID-05
**Tests**: internal/config/config_test.go
**Gate**: go test -count=1 ./internal/config/...
**Done when**:
- [x] `LoadVaultRepoID` lê `repo_id` de `.memory/config.yaml`, `.memory/config.yml`, `.memory/config.json` e `.mem.yaml`/`.mem.yml`/`.mem.json`, na mesma ordem de prioridade de `CandidateConfigFileNames`
- [x] A busca é restrita ao path informado: um path **sem** config devolve `os.ErrNotExist` mesmo quando o diretório pai tem vault
- [x] O path é expandido (`ExpandPath`) antes da leitura, como o cascateamento faz
- [x] Config sem `repo_id` devolve string vazia e `err == nil` (não há contra o que comparar)
- [x] A função **não escreve** nada em disco

### T2: Auditoria de identidade no `mem repos`
**What**: Auditar cada entrada do catálogo contra o `repo_id` do vault no path registrado; marcar a linha divergente com `❌`, imprimir o bloco de erro explícito (id do catálogo, repo_id do disco, path, arquivo do vault, ação para alinhar) e devolver erro para que o comando saia com código diferente de zero
**Where**: cmd/mem/setup.go
**Depends on**: T1
**Requirement**: REPOID-01, REPOID-02, REPOID-03, REPOID-06, REPOID-07, REPOID-08, REPOID-11, REPOID-12
**Tests**: cmd/mem/setup_test.go
**Gate**: go test -count=1 ./cmd/mem/...
**Done when**:
- [x] A auditoria roda para todas as entradas do catálogo e devolve, por entrada, o `repo_id` do disco, o arquivo lido e se há divergência
- [x] Entrada divergente sai com `❌` na tabela e **nunca** com `✅`
- [x] O erro devolvido contém os dois valores, o path e a ação para alinhar o catálogo; o catálogo global **não** é modificado
- [x] Entrada sem vault sai com `⚠️` e sem erro; entrada com vault sem `repo_id` sai sem erro
- [x] Sem divergência, a tabela (`REPO ID | NOME | CAMINHO NO DISCO`) e o código de saída zero continuam como antes
- [x] A auditoria é read-only: nenhum arquivo é criado ou alterado

### Phase 2: Regressão, caracterização e gate final

### T3: Testes de regressão do comando `mem repos`
**What**: Cobrir os cinco comportamentos observáveis do comando (divergente, alinhado, sem vault, vault sem `repo_id`, read-only) contra a implementação real
**Where**: cmd/mem/setup_test.go
**Depends on**: T2
**Requirement**: REPOID-01, REPOID-02, REPOID-03, REPOID-04, REPOID-05, REPOID-06, REPOID-07
**Tests**: cmd/mem/setup_test.go
**Gate**: go test -count=1 ./cmd/mem/...
**Done when**:
- [x] Divergência: `runReposCommand` devolve erro, a saída contém os dois ids, o path e a instrução, e a linha da tabela não contém `✅`
- [x] Alinhado: nenhum erro, saída contém `✅`
- [x] Sem vault: nenhum erro, saída contém `⚠️`
- [x] Vault sem `repo_id`: nenhum erro e nenhuma menção a divergência
- [x] Read-only: os bytes do catálogo global e do `.memory/config.yaml` do vault são idênticos antes e depois do comando
- [x] O teste falha quando o comportamento é revertido (comprovado na validation.md)

### T4: Caracterização do cascateamento com id alinhado
**What**: Fixar por teste as duas propriedades que o alinhamento compra: o override de storage do catálogo sobrevive à mudança de path quando o id casa, e o caminho oficial de registro realinha o id de uma entrada desatualizada sem perder o bloco `storage:`
**Where**: internal/config/config_test.go
**Depends on**: T1
**Requirement**: REPOID-09, REPOID-10
**Tests**: internal/config/config_test.go
**Gate**: go test -count=1 ./internal/config/...
**Done when**:
- [x] Com `id` do catálogo igual ao `repo_id` do vault e path do catálogo **diferente** do path atual, o `postgres_url` do catálogo continua aplicado pelo cascateamento
- [x] `RegisterRepositoryInGlobalConfig` com o `repo_id` do vault realinha o `id` de uma entrada cujo path é o mesmo, preservando o bloco `storage:` da entrada
- [x] A leitura do `repo_id` do vault não sobe para os diretórios pais (path filho sem config devolve `os.ErrNotExist` mesmo com vault no pai)

### T5: Gate final, prova contra o defeito e relatório de evidências
**What**: Rodar o gate completo, provar que o teste de regressão falha contra o comportamento antigo e registrar as evidências por critério
**Where**: .specs/features/repo-id-alignment/validation.md
**Depends on**: T3, T4
**Requirement**: REPOID-01, REPOID-02, REPOID-03, REPOID-04, REPOID-05, REPOID-06, REPOID-07, REPOID-08, REPOID-09, REPOID-10, REPOID-11, REPOID-12
**Tests**: .specs/features/repo-id-alignment/validation.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_state.py repo-id-alignment --root D:\repository\my-memory
**Done when**:
- [x] `go build ./...`, `go vet ./...` e `go test -count=1 ./...` com exit 0
- [x] `gofmt -l` limpo nos arquivos tocados (medido em cópia com LF)
- [x] Defeito revertido → teste falhando com o sintoma exato → restaurado
- [x] `validation.md` com evidência `arquivo:linha` por critério
