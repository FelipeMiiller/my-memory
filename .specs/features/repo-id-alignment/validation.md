# Validation: repo-id-alignment

**Spec**: [spec.md](./spec.md)
**Date**: 2026-10-29
**Issue**: ISSUE-028 (`.specs/ISSUES.md:945`)
**Branch**: `develop` · **Base**: `ef815b5` · **Diff**: working tree (sem commit — o git é do gestor)

## Validation

**Result**: PASS

Escopo do que foi verificado: 12 requisitos (REPOID-01..12), o sensor de discriminação (2 mutantes injetados e mortos), o gate completo (build, vet, test, gofmt) e os 3 validadores determinísticos do fluxo spec-driven.

**Arquivos tocados** (todos UTF-8 sem BOM; `.gitattributes` normaliza o blob para LF):

| Arquivo | Mudança |
| ------- | ------- |
| `internal/config/config.go` | +55 — helper `LoadVaultRepoID` |
| `cmd/mem/setup.go` | +118 / −6 — auditoria de identidade no `mem repos` |
| `internal/config/config_test.go` | +199 — 3 testes (leitura, cascateamento, re-registro) |
| `cmd/mem/setup_test.go` | +252 — 5 testes de regressão do comando |
| `.specs/features/repo-id-alignment/` | `spec.md`, `tasks.md`, `validation.md` |

---

## Evidência por requisito

| Req | Critério | Evidência | Comando | Saída real |
| --- | -------- | --------- | ------- | ---------- |
| REPOID-01 | Erro explícito com id do catálogo, repo_id do disco, path e ação | `cmd/mem/setup.go:321` (monta a mensagem), `cmd/mem/setup.go:415` (`errors.Join` → erro devolvido), `cmd/mem/setup_test.go:154` | `go test -count=1 -run TestRunReposCommand_DivergenciaRepoID ./cmd/mem/...` | `--- PASS: TestRunReposCommand_DivergenciaRepoID (0.02s)` — as 4 asserções (id do catálogo, repo_id do disco, path, `repositories[].id`) passam |
| REPOID-02 | Entrada divergente nunca listada como OK | `cmd/mem/setup.go:384` (`status = "❌"`), `cmd/mem/setup_test.go:190` (asserção "não pode ser listada como OK") | binário real (caso B) | `repo_271444ab0133    \| FelipeMiiller/my-memory   \| ❌ D:\repository\my-memory` |
| REPOID-03 | Saída com código diferente de zero | `cmd/mem/setup.go:415` + `cmd/mem/main.go:801-805` (`os.Exit(1)`) | binário real (caso B) | `exit=1` |
| REPOID-04 | Path sem config de vault → aviso `⚠️`, sem erro | `cmd/mem/setup.go:299-302` (`case errors.Is(err, os.ErrNotExist)`), `cmd/mem/setup.go:387` (`status = "⚠️"`), `cmd/mem/setup_test.go:232` | binário real (caso C) | `repo_sem_vault_01    \| test/sem-vault            \| ⚠️ …\no-vault` · `exit=0` |
| REPOID-05 | Vault sem `repo_id` → listado, sem erro | `cmd/mem/setup.go:304` (`result.Divergent = diskRepoID != "" && …`), `cmd/mem/setup_test.go:282`, `internal/config/config_test.go:840` | `go test -count=1 -run 'TestRunReposCommand_VaultSemRepoID|TestLoadVaultRepoID' ./cmd/mem/... ./internal/config/...` | `--- PASS: TestRunReposCommand_VaultSemRepoID` · `--- PASS: TestLoadVaultRepoID` (subtest "config sem repo_id devolve vazio sem erro") |
| REPOID-06 | Sem divergência: mesma tabela e exit 0 | `cmd/mem/setup.go:378-393` (tabela), `cmd/mem/setup_test.go:200` | binário real (caso A, catálogo real do dono) | `REPO ID \| NOME \| CAMINHO NO DISCO` + `repo_e4c8f3b1a2d5 … ✅ D:\repository\my-memory` · `exit=0` |
| REPOID-07 | Auditoria read-only | `internal/config/config.go:350` (só leitura), `cmd/mem/setup_test.go:313` | `go test -count=1 -run TestRunReposCommand_EhReadOnly ./cmd/mem/...` | `--- PASS: TestRunReposCommand_EhReadOnly` — bytes do catálogo global e do config do vault idênticos antes/depois e nenhum arquivo criado |
| REPOID-08 | Alinhado → `✅` | `cmd/mem/setup.go:382-388`, `cmd/mem/setup_test.go:200` | binário real (caso A) | `repo_e4c8f3b1a2d5    \| FelipeMiiller/my-memory   \| ✅ D:\repository\my-memory` |
| REPOID-09 | Override de storage sobrevive à mudança de path | `internal/config/config.go:750` (`matchesID`), `internal/config/config_test.go:942` | `go test -count=1 -run TestLoadCascadingConfig_CatalogOverrideSobreviveMove ./internal/config/...` | `--- PASS` — entrada de catálogo com path **antigo** ainda aplica `postgres://isolated_repo_db` |
| REPOID-10 | Caminho oficial realinha o id e preserva `storage:` | `cmd/mem/main.go:980-987` (o init preserva o repo_id) + `cmd/mem/main.go:1053-1057` (registra com ele), `internal/config/config.go:621`/`:660` (idempotência por id ou path), `internal/config/config_test.go:993` | `go test -count=1 -run TestRegisterRepositoryInGlobalConfig_RealignsStaleIDByPath ./internal/config/...` | `--- PASS` — entrada `repo_stale_000000` → `repo_real_111111` com `storage.postgres_url` intacto |
| REPOID-11 | Instrui a alinhar e não mexe no catálogo global | `cmd/mem/setup.go:330-356` (bloco com a instrução), `cmd/mem/setup_test.go:313` (read-only) | binário real (caso B) + catálogo real do dono | `→ Alinhe o catálogo global (…): defina repositories[].id como "repo_e4c8f3b1a2d5"` · catálogo real do dono continua `id: repo_e4c8f3b1a2d5` |
| REPOID-12 | Repo_id do disco é o autoritativo e o arquivo do vault é apontado | `cmd/mem/setup.go:321-327` | binário real (caso B) | `repo_id do disco: repo_e4c8f3b1a2d5  (declarado em D:\repository\my-memory\.memory\config.yaml)` e `o repo_id do disco é a fonte da verdade` |

---

## Respostas às 3 perguntas da spec (com evidência, não com opinião)

### 1. Mecanismo de detecção hoje

O cascateamento casa a entrada do catálogo por **ID ou caminho** (`internal/config/config.go:750-752`):

```go
matchesID := localRepoID != "" && repoEntry.ID == localRepoID
matchesPath := absDir != "" && entryAbs == absDir
if matchesID || matchesPath {
```

`localRepoID` é o `repo_id` do `.memory/config.yaml` do repositório (`internal/config/config.go:646-673`) — a **camada 3**. Quando a entrada casa, ela sobrepõe **somente o storage** (`internal/config/config.go:698-708`); o `repo_id` nunca vem do catálogo. **Não existe, em nenhum ponto do código, uma comparação entre o `id` do catálogo e o `repo_id` do disco que produza erro** — a divergência só muda o resultado quando o path também muda. Com o id alinhado, `matchesID` passa a ser verdadeiro e o casamento deixa de depender do path volátil.

### 2. O que acontece hoje se o repositório for movido/clonado

**Comportamento atual (observável)**: com ids divergentes, o moved repo quebra o `matchesPath`; o `matchesID` também é falso → a entrada não casa → o override de storage por repositório é **perdido em silêncio** e o repositório volta ao storage global (`internal/config/config.go:631-643`), sem nenhum aviso — o `mem repos` listava a entrada com `✅` porque o único critério era "o diretório `.memory` existe".

**Comportamento novo**: com o id alinhado, o moved repo **continua casando pelo id** — provado por `internal/config/config_test.go:942`, onde a entrada do catálogo aponta para um path **antigo** e o `postgres_url` do catálogo continua aplicado. E, se o id voltar a divergir, `mem repos` passa a falhar alto (caso B: `exit=1`).

### 3. Diagnóstico ou auto-correção?

**Diagnóstico apenas.** O código detecta e acusa; o alinhamento do `~/.memory/config.yaml` é operação manual do dono, guiada pela instrução que o erro entrega. A auto-correção foi avaliada e rejeitada por 3 razões registradas na spec (Pergunta 3): ela pode **agravar o split-brain** (duas entradas com o mesmo `repo_id`), tem **efeito colateral** no bloco `storage:` (`internal/config/config.go:605-609` só preserva o storage quando a nova entrada vem sem ele), e um default seguro exigiria dry-run + backup sobre um arquivo do usuário. O caminho oficial que já realinha a entrada existe e está provado: `mem init` preserva o `repo_id` do vault (`cmd/mem/main.go:980-987`) e re-registra com ele (`cmd/mem/main.go:1053-1057`) — `internal/config/config_test.go:993` prova que o re-registro realinha o id e preserva o `storage:`.

---

## Prova contra o defeito (sensor de discriminação)

Sensor em arquivo isolado: backup byte-exato de `cmd/mem/setup.go` em `D:\.minimax\scratch\subagents\repo-id-alignment\setup.go.baseline`, mutação injetada **no arquivo real**, teste rodado, arquivo restaurado byte a byte. Baseline do git conferido antes e depois (`118 6 / 252 0 / 55 0 / 199 0` — idêntico).

| Mutante | Falha injetada | Comportamento antigo | Teste que morreu | Saída real |
| ------- | -------------- | ------------------- | ---------------- | ---------- |
| M1 | `cmd/mem/setup.go:304` → `result.Divergent = false` | O `mem repos` **não acusa** a divergência: entrada listada como OK e `exit=0` (o defeito da ISSUE-028) | `TestRunReposCommand_DivergenciaRepoID` | `--- FAIL: TestRunReposCommand_DivergenciaRepoID` → `esperava erro na divergência de repo_id, mas o comando saiu sem erro` |
| M2 | `cmd/mem/setup.go:384` → `status = "✅"` | A entrada divergente é **listada como OK** mesmo com o erro no stderr | `TestRunReposCommand_DivergenciaRepoID` | `--- FAIL: TestRunReposCommand_DivergenciaRepoID` → `a entrada divergente não pode ser listada como OK` |

Com o código restaurado: `go test -count=1 -run TestRunReposCommand ./cmd/mem/...` → `ok github.com/FelipeMiiller/my-memory/cmd/mem 0.260s` (`exit=0`).

---

## Comportamento no binário real

Binário compilado do estado final (`go build -o …\mem.exe ./cmd/mem`). Nenhum arquivo do dono foi alterado: os casos B e C usam um `MY_MEMORY_GLOBAL_CONFIG_DIR` de scratch; o caso A usa o catálogo real do dono **em modo leitura**.

**A) Catálogo real do dono (alinhado pela Parte A) — `exit=0`**

```
=== Catálogo de Repositórios Registrados no My-Memory ===
Arquivo Global: C:\Users\Felipe\.memory\config.yaml

REPO ID              | NOME                      | CAMINHO NO DISCO
---------------------------------------------------------------------------
repo_e4c8f3b1a2d5    | FelipeMiiller/my-memory   | ✅ D:\repository\my-memory
```

**B) Catálogo divergente (`repo_271444ab0133`) contra o repo real (`repo_e4c8f3b1a2d5`) — `exit=1`**

```
REPO ID              | NOME                      | CAMINHO NO DISCO
---------------------------------------------------------------------------
repo_271444ab0133    | FelipeMiiller/my-memory   | ❌ D:\repository\my-memory

❌ Divergência de repo_id no catálogo global (1 entrada(s) divergente(s)):

  [1] id do catálogo:    repo_271444ab0133
      path registrado:  D:\repository\my-memory
      repo_id do disco: repo_e4c8f3b1a2d5  (declarado em D:\repository\my-memory\.memory\config.yaml)
      → Alinhe o catálogo global (…\config.yaml): defina repositories[].id como "repo_e4c8f3b1a2d5"
        O repo_id do disco é a fonte da verdade. Alternativa: rode
        "mem init --force" dentro do repositório (re-registra a entrada com o
        repo_id do vault, mas reescreve o .memory/config.yaml).

Erro: divergência de repo_id no catálogo global: id do catálogo "repo_271444ab0133" (path D:\repository\my-memory) não corresponde ao repo_id "repo_e4c8f3b1a2d5" declarado em D:\repository\my-memory\.memory\config.yaml; alinhe o catálogo global …\config.yaml definindo repositories[].id como "repo_e4c8f3b1a2d5" — o repo_id do disco é a fonte da verdade
```

**C) Catálogo misto (alinhado + entrada sem vault) — `exit=0`**

```
repo_e4c8f3b1a2d5    | FelipeMiiller/my-memory   | ✅ D:\repository\my-memory
repo_sem_vault_01    | test/sem-vault            | ⚠️ …\no-vault
```

---

## Gate

| Gate | Comando | Saída |
| ---- | ------- | ----- |
| Build | `go build ./...` | `exit=0` |
| Vet | `go vet ./...` | `exit=0` |
| Testes | `go test -count=1 ./...` | `exit=0` — 28 pacotes `ok`, nenhum `FAIL` (`cmd/mem` 90.7s, `internal/config` 1.3s) |
| Formatação | `gofmt -l` na **cópia LF** dos 4 arquivos tocados (`D:\.minimax\scratch\subagents\repo-id-alignment\gofmt-check`) | saída vazia — formatação limpa |
| Spec | `python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/repo-id-alignment/spec.md` | `0 error(s), 0 warning(s)` · `exit=0` |
| Tasks | `python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py .specs/features/repo-id-alignment/tasks.md` | `0 error(s), 0 warning(s)` · `exit=0` |
| Estado | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py repo-id-alignment --root D:\repository\my-memory` | `0 error(s)` · `exit=0` |

---

## Notas de escopo e pendências

- **A auditoria é estritamente read-only** — nenhum arquivo do usuário (catálogo global ou vault) é criado, alterado ou removido. Provado por `cmd/mem/setup_test.go:313` e pelo estado do catálogo real do dono depois das execuções (`id: repo_e4c8f3b1a2d5`, intacto).
- **Mudança de comportamento deliberada e compatível**: o aviso `⚠️` passou a significar "não há arquivo de configuração de vault no path" (antes: "não há diretório `.memory`"). Um path com `.memory/` mas sem `config.yaml` agora sai `⚠️` em vez de `✅` — o aviso passa a ser verdadeiro em todos os casos (assumption A5 da spec). As **colunas, o cabeçalho e o separador da tabela não mudaram**.
- **Fora do escopo desta execução** (pendências para o gestor): atualizar `docs/CLI_GUIDE.md` (§ "Catálogo Global de Repositórios"), `COMO_USAR.md`, `README.md`, `skills/mymemory-cli/references/cli-reference.md`, `CHANGELOG.md`, `.specs/ISSUES.md` (status da ISSUE-028) e `.specs/STATE.md`. Nenhum ADR novo é necessário: o comportamento do cascateamento não mudou (ADR-040/ADR-033 seguem válidas).
- **Nenhum commit e nenhum push** — o git é do gestor. As mudanças estão no working tree.
