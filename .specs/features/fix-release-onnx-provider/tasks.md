# Tasks: fix-release-onnx-provider

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-13
**Issue**: ISSUE-037 (`.specs/ISSUES.md`)

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Design**: inline nesta spec (mudança de pipeline + um comando novo; sem decisão de arquitetura nova — ADR-035 e ADR-032 já decidem o modelo e o instalador)
**Status**: In Progress

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec. Guidelines found: `AGENTS.md` (regras 4, 7 e 8 — gate determinístico, documentação obrigatória e quality gate), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`, `.github/workflows/ci.yml` (comandos de teste reais do projeto).

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Download do modelo e da DLL | unit | 1:1 com os ACs 5 e 6: grava no path, arquivo parcial descartado, tamanho verificado, erro nomeia URL e destino | `internal/embedder/model_downloader_test.go` | `go test -count=1 -run TestDownload ./internal/embedder/...` |
| Comandos `mem embed` | unit | AC8: `mem embed doctor` reporta provider, paths e existência de cada artefato; `mem embed download` propaga o erro do downloader | `cmd/mem/embed_test.go` | `go test -count=1 -run TestEmbed ./cmd/mem/...` |
| Degradação no index e na busca | integration | AC9, AC10 e AC11: um aviso por execução (não um por documento), contagem de documentos sem vetor, fallback léxico declarado na saída | `cmd/mem/embed_provider_degradation_test.go` | `go test -count=1 -run TestProviderDegradation ./cmd/mem/...` |
| Provider com a tag `onnx` | integration (ONNX real) | AC4: o binário com a tag gera vetor e a busca responde com a fonte `vector` | `internal/embedder/builtin_test.go` | `CGO_ENABLED=1 go test -tags "sqlite_fts5 onnx" -count=1 ./internal/embedder/... ./cmd/mem/...` |
| Pipeline (CI e release) | none | o gate do pipeline é o próprio CI; a verificação local é `go build -tags "sqlite_fts5 onnx"` + leitura do YAML | `.github/workflows/*.yml` | build gate only |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após cada tarefa de código | `go build ./...` e `go vet ./...` |
| Quick | Após as tarefas de teste | `go test -count=1 -run TestDownload ./internal/embedder/...` |
| Full | Gate final (T9) | `go test -count=1 ./...` (com `MY_MEMORY_PG_TEST=1` e `MY_MEMORY_PG_URL` preservada) |
| Tagged | T1 e T9 | `$env:PATH="C:\msys64\ucrt64\bin;$env:PATH"; $env:CGO_ENABLED='1'; go build -tags "sqlite_fts5 onnx" -o bin/mem.exe ./cmd/mem` |
| Format | Antes de declarar pronto | `gofmt -l` na cópia LF dos arquivos `.go` tocados (o working tree local mistura CRLF) |
| Spec | Antes de entregar | `python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/fix-release-onnx-provider/spec.md` e `validate_tasks.py .specs/features/fix-release-onnx-provider/tasks.md` |
| State | Antes de entregar | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-release-onnx-provider --root D:\repository\my-memory` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: O binário oficial

```
T1 -> T2 -> T3
```

### Phase 2: Os artefatos do provider na máquina

```
T4 -> T5 -> T6 -> T7
```

### Phase 3: A degradação explícita

```
T8 -> T9
```

---

## Task Breakdown

### T1: Job de CI que compila e testa com a tag `onnx`

**What**: acrescentar ao `ci.yml` um job no padrão do `codeast-cgo-on` que compila
`cmd/mem` com `-tags "sqlite_fts5 onnx"` e roda a suíte do provider sob a tag.
**Where**: `.github/workflows/ci.yml`
**Depends on**: None
**Reuses**: o job `codeast-cgo-on` (mesmo padrão de toolchain, `continue-on-error`
e boot check)
**Requirement**: ROP-03

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] Existe um job com a tag `onnx` no `ci.yml`
- [x] O job instala o toolchain C (`gcc`) e imprime `CGO_ENABLED`/`CC`
- [x] O job roda os testes do provider sob a tag
- [x] `go build -tags "sqlite_fts5 onnx" ./cmd/mem` compila localmente

**Status**: Verified (2026-11-13)

**Tests**: none (o gate é o próprio pipeline; a prova local é o build com a tag)
**Gate**: tagged

**Commit**: `ci(embedder): job com a tag onnx compila e testa o provider`

---

### T2: Build Linux do release com a tag `onnx`

**What**: o build Linux do release passa a compilar com `-tags "sqlite_fts5
onnx"` e `CGO_ENABLED=1`.
**Where**: `.github/workflows/release.yml`
**Depends on**: T1
**Reuses**: o bloco de build Linux existente (`:114-121`)
**Requirement**: ROP-01

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] O build Linux usa as duas tags e `CGO_ENABLED=1`
- [x] O `-ldflags` e o nome do artefato seguem intactos
- [x] O artefato resultante passa no smoke test de `mem search` (fonte `vector`)

**Status**: Verified (2026-11-13)

**Tests**: none
**Gate**: tagged

**Commit**: `build(release): compila o artefato Linux com a tag onnx`

---

### T3: Build Windows do release com a tag `onnx`

**What**: o build Windows do release compila com `-tags "sqlite_fts5 onnx"` e
`CGO_ENABLED=1`; se o runner não tiver toolchain C, o job **falha** com mensagem
explícita em vez de publicar binário sem provider.
**Where**: `.github/workflows/release.yml`
**Depends on**: T2
**Reuses**: o bloco de build Windows existente (`:127-134`)
**Requirement**: ROP-02, ROP-04

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] O build Windows usa as duas tags e `CGO_ENABLED=1`
- [x] Existe um passo que verifica a toolchain C e falha com mensagem explícita
      quando ela não existe
- [x] O `checksums.txt` publicado cobre o novo artefato

**Status**: Verified (2026-11-13)

**Tests**: none
**Gate**: tagged

**Commit**: `build(release): compila o artefato Windows com a tag onnx`

---

### T4: Downloader do modelo e da DLL do ONNX Runtime

**What**: `internal/embedder/model_downloader.go` baixa o modelo
(`paraphrase-multilingual-MiniLM-L12-v2`) e a DLL do ONNX Runtime 1.29.0 para
`~/.memory/models/`, verificando o tamanho de cada arquivo e movendo-o por
rename atômico.
**Where**: `internal/embedder/model_downloader.go`
**Depends on**: None
**Reuses**: `internal/embedder/builtin.go` (os paths que o provider já procura)
**Requirement**: ROP-05, ROP-06

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] O download grava no path que `cmd/mem/builtin_embedder.go` procura
- [x] O tamanho de cada arquivo é verificado antes do rename final
- [x] Falha de download nomeia a URL tentada e o destino, e não deixa parcial
- [x] O teste com `httptest` prova o caminho feliz, o arquivo parcial
      descartado e o erro explícito

**Status**: Verified (2026-11-13)

**Tests**: unit
**Gate**: quick

**Commit**: `feat(embedder): baixa o modelo e a DLL do ONNX Runtime`

---

### T5: Comandos `mem embed download` e `mem embed doctor`

**What**: expor o downloader e o diagnóstico do provider na CLI.
**Where**: `cmd/mem/embed.go`
**Depends on**: T4
**Reuses**: `cmd/mem/builtin_embedder.go` (paths e estado do provider)
**Requirement**: ROP-08

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] `mem embed download` baixa os artefatos e imprime o path final
- [x] `mem embed doctor` reporta provider ativo, path do modelo, path da DLL,
      dimensão e existência de cada artefato
- [x] `mem embed doctor` funciona sem provider (reporta "indisponível", não
      quebra)

**Status**: Verified (2026-11-13)

**Tests**: unit
**Gate**: quick

**Commit**: `feat(cli): mem embed download e mem embed doctor`

---

### T6: O instalador Windows busca os artefatos do provider

**What**: `scripts/install.ps1` invoca `mem embed download` depois de instalar o
binário e imprime o diretório do modelo.
**Where**: `scripts/install.ps1`
**Depends on**: T5
**Reuses**: o bloco de download do release já existente no script
**Requirement**: ROP-07

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] O instalador chama `mem embed download` e não aborta a instalação se o
      download falhar (aviso com o comando para rodar à mão)
- [x] O instalador imprime o path do diretório do modelo
- [x] O script continua sintaticamente válido (`-WhatIf` / parse sem erro)

**Status**: Verified (2026-11-13)

**Tests**: none
**Gate**: build gate only (parse do script)

**Commit**: `feat(installer): o instalador Windows baixa os artefatos do provider`

---

### T7: O instalador POSIX busca os artefatos do provider

**What**: `scripts/install.sh` invoca `mem embed download` e imprime o
diretório do modelo.
**Where**: `scripts/install.sh`
**Depends on**: T6
**Reuses**: o bloco de download do release já existente no script
**Requirement**: ROP-07

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [x] O instalador chama `mem embed download` e não aborta se o download falhar
- [x] O instalador imprime o path do diretório do modelo
- [x] `bash -n scripts/install.sh` passa

**Status**: Verified (2026-11-13)

**Tests**: none
**Gate**: build gate only (parse do script)

**Commit**: `feat(installer): o instalador POSIX baixa os artefatos do provider`

---

### T8: Aviso consolidado no `mem index`

**What**: quando não há provider, `mem index` imprime **um** aviso consolidado
(no lugar de um por documento) e diz quantos documentos foram indexados sem
vetor; o código de saída continua 0.
**Where**: `cmd/mem/main.go`
**Depends on**: None
**Reuses**: `resolveEmbedder` (`cmd/mem/main.go:1455`) e os avisos por documento
existentes (`:1897`, `:2138`)
**Requirement**: ROP-09, ROP-11

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] O aviso aparece uma vez por execução, não uma vez por documento
- [ ] O aviso nomeia a tag ausente e a consequência (chunks sem vetor)
- [ ] A saída final diz quantos documentos ficaram sem vetor
- [ ] O exit code continua 0

**Tests**: integration
**Gate**: quick

**Commit**: `fix(index): aviso consolidado quando o provider semântico está ausente`

---

### T9: A busca declara o fallback léxico

**What**: `mem search --mode hybrid` sem provider declara explicitamente que o
resultado veio do fallback textual.
**Where**: `cmd/mem/main.go`
**Depends on**: T8
**Reuses**: os avisos de fallback já existentes em `runSearch*` (`:2283`, `:2432`)
**Requirement**: ROP-10

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] A saída da busca híbrida sem provider diz que o resultado é só lexical
- [ ] A saída com provider **não** mostra o aviso de fallback
- [ ] O modo `vector` sem provider continua saindo com código ≠ 0

**Tests**: integration
**Gate**: full

**Commit**: `fix(search): declara o fallback léxico na busca híbrida`

---

## Phase Execution Map

```
Phase 1 → Phase 2 → Phase 3

Phase 1:  T1 -> T2 -> T3
Phase 2:  T4 -> T5 -> T6 -> T7
Phase 3:  T8 -> T9
```

Execution is strictly sequential - there is no intra-phase parallelism. A single
agent (or batch worker) works one task at a time, in order.

**Contagem de tarefas**: 9 — cabem em um único lote (≤ ~8 por worker), então a
execução é **inline**, sem sub-agente.

---

## Diagram-Definition Cross-Check

| Task | Depends On (task body) | Diagram Shows | Status |
| ---- | ---------------------- | ------------- | ------ |
| T1 | None | (início da fase 1) | ✅ Match |
| T2 | T1 | T1 -> T2 | ✅ Match |
| T3 | T2 | T2 -> T3 | ✅ Match |
| T4 | None | (início da fase 2) | ✅ Match |
| T5 | T4 | T4 -> T5 | ✅ Match |
| T6 | T5 | T5 -> T6 | ✅ Match |
| T7 | T6 | T6 -> T7 | ✅ Match |
| T8 | None | (início da fase 3) | ✅ Match |
| T9 | T8 | T8 -> T9 | ✅ Match |

Nenhuma dependência aponta para uma fase posterior.

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | CI pipeline | none | none | ✅ OK |
| T2 | CI pipeline | none | none | ✅ OK |
| T3 | CI pipeline | none | none | ✅ OK |
| T4 | Download do modelo e da DLL | unit | unit | ✅ OK |
| T5 | Comandos `mem embed` | unit | unit | ✅ OK |
| T6 | Instalador (script) | none | none | ✅ OK |
| T7 | Instalador (script) | none | none | ✅ OK |
| T8 | Degradação no index | integration | integration | ✅ OK |
| T9 | Degradação na busca | integration | integration | ✅ OK |

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: job de CI | 1 job em 1 arquivo | ✅ Granular |
| T2: build Linux | 1 bloco de build | ✅ Granular |
| T3: build Windows | 1 bloco de build | ✅ Granular |
| T4: downloader | 1 arquivo novo | ✅ Granular |
| T5: comandos `mem embed` | 1 arquivo novo | ✅ Granular |
| T6: instalador Windows | 1 script | ✅ Granular |
| T7: instalador POSIX | 1 script | ✅ Granular |
| T8: aviso do index | 1 função | ✅ Granular |
| T9: aviso da busca | 1 função | ✅ Granular |
