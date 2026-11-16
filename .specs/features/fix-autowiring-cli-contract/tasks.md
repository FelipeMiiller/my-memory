# Tasks: fix-autowiring-cli-contract

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-16
**Issue**: ISSUE-048 (`.specs/ISSUES.md`) · fecha os **10 gaps** da feature `ai-tool-auto-wiring` (ADR-026)

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Design**: inline nesta spec (o verbo `setup` tem um dono só · a fronteira CLI discrimina · o payload respeita a ADR-040 · só escreve onde o cliente está instalado · o backup é atômico · o contrato publicado descreve o código)
**Status**: Draft
**Consolidação (2026-11-16)**: esta planilha é a **versão final única** da ISSUE-048 e absorve a planilha paralela `.specs/features/fix-ai-tool-auto-wiring-gaps/` (11 tasks), que cobria os **9 gaps** de comportamento/documentação e **omitia o gap 10** (o plano de validação antigo listava 8 testes que nunca existiram). O que era **material** na versão paralela foi incorporado: o **gap 10** entra no escopo da T18 (o relatório cita apenas testes que existem e declara que os 8 nomes nunca existiram), as **flags de escopo contraditórias** entram na T10 e na T14, a **idempotência** (`already_up_to_date`) e o **`--dry-run` que não lista cliente inexistente** entram na T12 e na T16. Onde as duas versões **discordavam** — o conjunto aceito de `--target`, a forma de detectar a presença do cliente e o comportamento de `mem setup --dry-run` — a decisão está na tabela de Assumptions da spec (A6, A7 e A10) com o **default, o rationale e a alternativa**. A planilha paralela é removida pelo dono; as duas não podem ser implementadas juntas.

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec. Guidelines found: `AGENTS.md` (regra 4 — testes obrigatórios; regra 8 — quality gate), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`, `.github/workflows/ci.yml` (comandos de teste reais do projeto).

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Contrato publicado (ADR-026, CLI guide, guia de agentes, spec/tasks da feature de origem) | none | build gate only — o contrato é conferido por leitura cruzada entre doc e código (grep por `mem setup`, por `claude-desktop`, pela tabela de comandos) | `docs/adr/026-auto-wiring-e-instalacao-zero-touch-de-ferramentas-de-ia.md` | build gate only |
| Domínio de clientes (conjunto de alvos aceitos, sinônimo, presença) | unit | 1:1 com AUTOWIRE-16 a AUTOWIRE-18 e AUTOWIRE-23: o conjunto aceito de `--target`, o sinônimo `claude-desktop` (resolve o Claude e **nada** mais) e a marcação de presença pelo marcador de instalação | `internal/autowire/detector_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestDetect ./internal/autowire/...` |
| Entidade de cliente (`ClientTarget`) | none | build gate only — o campo novo é **consumido** e asserido pelo detector (T8/T9); a entidade não tem comportamento próprio | `internal/autowire/client.go` | build gate only |
| Fronteira CLI do instalador | integration | 1:1 com AUTOWIRE-03 a AUTOWIRE-07 (lado **negativo** dos filtros de alvo e de escopo, e a falha das flags de escopo contraditórias), AUTOWIRE-08 a AUTOWIRE-15 (payload e linha `Engine:` por engine) e AUTOWIRE-19 a AUTOWIRE-24 (`--target` inválido com exit ≠ 0, pulo por ausência com probe injetado) | `cmd/mem/install_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestInstall ./cmd/mem/...` |
| Escrita do config e backup | integration | 1:1 com AUTOWIRE-25 a AUTOWIRE-28: `.bak` via temporário + rename, byte-idêntico ao original, formatação canônica de 2 espaços e conteúdo pré-existente preservado | `internal/autowire/installer_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestInstaller ./internal/autowire/...` |
| Verificação independente + sensor | integration | 1:1 com AUTOWIRE-33 a AUTOWIRE-37: M7 e M8 morrem, o relatório cita só testes que existem e o veredito é PASS com evidência `file:line` | `.specs/features/ai-tool-auto-wiring/validation.md` | `python3 .agents/skills/tlc-spec-driven/scripts/validate_state.py ai-tool-auto-wiring --root D:\repository\my-memory` |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após cada task de código | `go -C D:\repository\my-memory build ./...` e `go -C D:\repository\my-memory vet ./...` |
| Quick | Após T9 (domínio) | `go -C D:\repository\my-memory test -count=1 -run TestDetect ./internal/autowire/...` |
| Quick | Após T14, T15 e T16 (CLI) | `go -C D:\repository\my-memory test -count=1 -run TestInstall ./cmd/mem/...` |
| Quick | Após T17 (instalador) | `go -C D:\repository\my-memory test -count=1 -run TestInstaller ./internal/autowire/...` |
| Quick | Após T1 a T5 (contrato — não muda código) | `go -C D:\repository\my-memory build ./...` (o gate do contrato é o build + a leitura cruzada doc↔código) |
| Full | Gate final (T18) | `go -C D:\repository\my-memory test -count=1 ./...` (com `MY_MEMORY_PG_TEST=1` e `MY_MEMORY_PG_URL` preservada) |
| Full | Gate final taggeado | `go -C D:\repository\my-memory test -tags "sqlite_fts5 onnx" -count=1 ./...` |
| Build | Antes de declarar pronto | `go -C D:\repository\my-memory build ./...` |
| Format | Antes de declarar pronto | `gofmt -l` nos `.go` tocados, medido em **cópia LF** (o working tree local mistura CRLF — o `gofmt` do repo aponta 134 arquivos por isso, e a CI roda em Linux) |
| Spec | Antes de entregar | `python3 .agents/skills/tlc-spec-driven/scripts/validate_spec.py fix-autowiring-cli-contract --root D:\repository\my-memory` e `python3 .agents/skills/tlc-spec-driven/scripts/validate_tasks.py fix-autowiring-cli-contract --root D:\repository\my-memory` |
| State | Antes de entregar | `python3 .agents/skills/tlc-spec-driven/scripts/validate_state.py ai-tool-auto-wiring --root D:\repository\my-memory` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: O contrato publicado

O verbo `setup` tem um dono só (ADR-033) e a documentação passa a descrever o código que existe.

```
T1 -> T2
T1 -> T3
T1 -> T4
T4 -> T5
```

### Phase 2: O domínio conhece os alvos aceitos e a presença

O conjunto aceito de `--target` e o marcador de instalação do cliente passam a viver no domínio.

```
T1 -> T6
T6 -> T9
T7 -> T8
T8 -> T9
```

### Phase 3: A fronteira CLI e o instalador

`--target` inválido falha alto, o payload respeita a ADR-040, o cliente ausente é pulado e o `.bak` é atômico.

```
T1 -> T13
T2 -> T11
T6 -> T10
T8 -> T12
```

### Phase 4: As asserções que faltavam

O lado negativo dos filtros, o payload por engine, o pulo por ausência e o backup atômico viram teste.

```
T10 -> T14
T11 -> T14
T12 -> T14
T11 -> T15
T10 -> T16
T12 -> T16
T13 -> T17
```

### Phase 5: A re-verificação independente

O Verifier independente roda o sensor de discriminação e reescreve o relatório da feature de origem.

```
T14 -> T18
T15 -> T18
T16 -> T18
T17 -> T18
```

---

## Task Breakdown

### T1: A ADR-026 passa a descrever o auto-wiring que existe

**What**: emendar a ADR-026 para (a) dizer que o comando do auto-wiring é o
`mem install` e registrar que o verbo `setup` pertence ao **assistente de cofre
central** (ADR-033), como o `8685ed8` repurposou; (b) trocar o registro
declarativo `ClientSpec` (`:31-35`) pela implementação real — as quatro
constantes `ClientType` e a resolução de caminho por cliente; (c) corrigir o
escopo **Workspace** do Windsurf (`:42`), que não existe — só há o global.
**Where**: `docs/adr/026-auto-wiring-e-instalacao-zero-touch-de-ferramentas-de-ia.md`
**Depends on**: None
**Reuses**: o formato MADR do repo · as constantes reais (`internal/autowire/client.go:6-11`)
· os caminhos reais (`internal/autowire/detector.go:69-158`) · as decisões A1, A8 e A10 da spec
**Requirement**: AUTOWIRE-02, AUTOWIRE-30, AUTOWIRE-31
**Gap**: gap 1 (Blocker), gap 8

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] A ADR diz que o auto-wiring é o `mem install` e que `mem setup` é do assistente de cofre central (ADR-033)
- [x] A ADR não descreve registro declarativo `ClientSpec` — ela descreve as 4 constantes `ClientType` e a resolução de caminho por cliente
- [x] A ADR diz que o Windsurf é suportado **só no escopo global**
- [x] Cada afirmação técnica da ADR aponta para um `file:linha` que existe no HEAD
- [ ] O status da ADR em `docs/adr/README.md` reflete o estado real (com a spec de correção) — **fora do escopo desta execução**: `docs/adr/README.md` pertence a outra spec (verOwnership) e não pode ser editado aqui
- [x] O gate de build passa: `go -C D:\repository\my-memory build ./...`

**Tests**: none (camada de contrato — o gate é o build + a leitura cruzada)
**Gate**: quick

**Commit**: `docs(adr): a adr-026 descreve o auto-wiring que existe`

---

### T2: O CLI guide para de prometer o alias e passa a listar o contrato real

**What**: o `docs/CLI_GUIDE.md` deixa de anunciar `mem setup` como alias (`:625` e
o exemplo `:650-651`), passa a dizer `mem install` como **único** comando do
auto-wiring, lista o conjunto aceito de `--target` (`all`, `claude`,
`claude-desktop`, `cursor`, `vscode`, `windsurf`) e documenta a linha `Engine:`
e a precedência de storage (ADR-040) do payload injetado.
**Where**: `docs/CLI_GUIDE.md`
**Depends on**: T1
**Reuses**: a Ajuda real da flag (`cmd/mem/install.go:25`) e o conjunto do domínio
(`internal/autowire/client.go:6-11`) como fonte da verdade · a precedência da
ADR-040 · as decisões A3, A4 e A6 da spec
**Requirement**: AUTOWIRE-01, AUTOWIRE-13, AUTOWIRE-14, AUTOWIRE-15, AUTOWIRE-16
**Gap**: gap 1 (Blocker), gap 3, gap 4

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] `grep -n "setup" docs/CLI_GUIDE.md` não devolve nenhuma promessa de alias do auto-wiring (o exemplo `./bin/mem.exe setup --dry-run` saiu)
- [x] O guia documenta `mem install` como o comando do auto-wiring e não menciona `mem setup` como sinônimo
- [x] O guia lista os 6 valores aceitos de `--target`, incluindo `claude-desktop`
- [x] O guia diz que o payload declara o engine vencedor (`Engine: postgres` ou `Engine: sqlite`) e que a URL do Postgres **não** vai para o arquivo do editor
- [x] O gate de build passa: `go -C D:\repository\my-memory build ./...`

**Tests**: none (camada de contrato — o gate é o build + a leitura cruzada)
**Gate**: quick

**Commit**: `docs(cli): o guia documenta mem install e o target aceito`

---

### T3: O guia de agentes volta a listar o `mem install`

**What**: o `docs/AGENT_INTEGRATION_GUIDE.md` volta a ter a linha `mem install`
na tabela de comandos (a que `db02542` adicionou e `f5646fd` removeu), com o
que ele faz e um remate ao CLI guide para as flags.
**Where**: `docs/AGENT_INTEGRATION_GUIDE.md`
**Depends on**: T1
**Reuses**: a tabela de comandos do guia (a mesma que lista `mem hook install`,
`:271`) · a seção de auto-wiring do CLI guide
**Requirement**: AUTOWIRE-32
**Gap**: gap 9

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O guia tem uma linha `mem install` com o que ele faz
- [x] A linha remete ao CLI guide para o detalhe das flags
- [x] O gate de build passa: `go -C D:\repository\my-memory build ./...`

**Tests**: none (camada de contrato — o gate é o build + a leitura cruzada)
**Gate**: quick

**Commit**: `docs(agents): o guia volta a listar o mem install`

---

### T4: A spec da feature de origem para de prometer o alias e declara a formatação

**What**: a spec `.specs/features/ai-tool-auto-wiring/spec.md` para de anunciar
`mem install` / `mem setup` (`:9`) e o goal G3 (`:16`, "com alias `mem setup`"),
passa a declarar o **comando real** (`mem install`) e a **regra de formatação**
do config escrito (2 espaços, chaves de topo em ordem alfabética) — fechando o
gap 7 no texto — e cita esta spec de correção como origem da emenda.
**Where**: `.specs/features/ai-tool-auto-wiring/spec.md`
**Depends on**: T1
**Reuses**: os goals e ACs da spec de origem · o comportamento real do instalador
(`internal/autowire/installer.go:75`) · a decisão A8 da spec
**Requirement**: AUTOWIRE-02, AUTOWIRE-27, AUTOWIRE-29
**Gap**: gap 1 (Blocker), gap 7, gap 8

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] A spec de origem não promete `mem setup` como alias em nenhum ponto (`:9` e `:16` emendados)
- [x] A spec de origem declara que o config escrito usa 2 espaços e chaves de topo em ordem alfabética
- [x] A spec de origem cita a spec `fix-autowiring-cli-contract` e a ISSUE-048 como origem da emenda
- [x] A spec de origem descreve o registro de clientes que **existe** (as 4 constantes `ClientType`)
- [x] O gate de build passa: `go -C D:\repository\my-memory build ./...`

**Tests**: none (camada de contrato — o gate é o build + a leitura cruzada)
**Gate**: quick

**Commit**: `docs(spec): a spec de auto-wiring declara o comando e a formatacao`

---

### T5: Os checkboxes da feature de origem batem com o código

**What**: o `.specs/features/ai-tool-auto-wiring/tasks.md` deixa de marcar como
entregue o que não foi entregue — o alias `mem setup` (`:19`, `:23`, `:29`) e a
"preservação de indentação" (`:15`) — e passa a apontar a spec de correção como
origem da emenda.
**Where**: `.specs/features/ai-tool-auto-wiring/tasks.md`
**Depends on**: T4
**Reuses**: os checkboxes de T1/T2/T3/T4 da feature de origem · o relatório
`.specs/features/ai-tool-auto-wiring/validation.md` (fonte dos gaps)
**Requirement**: AUTOWIRE-01, AUTOWIRE-29
**Gap**: gap 1 (Blocker), gap 7

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] Nenhum checkbox da feature de origem afirma que existe o alias `mem setup`
- [x] Nenhum checkbox da feature de origem afirma que a indentação original é preservada
- [x] A tasks.md da origem cita a spec `fix-autowiring-cli-contract` (ISSUE-048) como origem da emenda
- [x] O gate de build passa: `go -C D:\repository\my-memory build ./...`

**Tests**: none (camada de contrato — o gate é o build + a leitura cruzada)
**Gate**: quick

**Commit**: `docs(spec): os checkboxes da spec batem com o codigo real`

---

### T6: O domínio declara o conjunto de alvos aceitos e o sinônimo `claude-desktop`

**What**: `internal/autowire/detector.go` passa a ter o **conjunto de alvos
aceitos declarado no domínio** (`all`, `claude`, `claude-desktop`, `cursor`,
`vscode`, `windsurf`) e o `matchTarget` (`:222`) normaliza `claude-desktop` como
sinônimo de `claude` — resolvendo o Claude Desktop e **nada** mais.
**Where**: `internal/autowire/detector.go`
**Depends on**: T1
**Reuses**: as constantes `ClientType` (`internal/autowire/client.go:6-11`) · o
`matchTarget` atual (`:222`) · o `matchScope` (`:226`) · a decisão A6 da spec
**Requirement**: AUTOWIRE-16, AUTOWIRE-17, AUTOWIRE-18
**Gap**: gap 4

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] O conjunto aceito de `--target` é exposto pelo domínio (fonte única) e vale `all`, `claude`, `claude-desktop`, `cursor`, `vscode`, `windsurf`
- [x] `claude-desktop` resolve o alvo do Claude Desktop
- [x] `claude-desktop` **não** resolve Cursor, VS Code nem Windsurf
- [x] `claude` continua resolvendo o Claude Desktop (nenhuma regressão)
- [x] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestDetect ./internal/autowire/...`
- [x] Test count: os testes de `internal/autowire` passam (9 no baseline + os novos), sem nenhuma deleção silenciosa

**Tests**: unit
**Gate**: quick

**Commit**: `feat(autowire): o dominio aceita o sinonimo claude-desktop`

---

### T7: O alvo de cliente carrega a presença de instalação

**What**: `internal/autowire/client.go` passa a expor no `ClientTarget` (`:30-37`)
o campo de **presença de instalação** do cliente no destino (`Installed`), ao
lado dos campos `Exists` e `ParentDirOK` — que hoje são calculados e nunca
consumidos. O `StatusSkipped` (`:47`) passa a ter **produtor**.
**Where**: `internal/autowire/client.go`
**Depends on**: None
**Reuses**: a struct `ClientTarget` (`:30-37`) · o `InstallStatus` com
`StatusSkipped` (`:42-49`) · a decisão A7 da spec
**Requirement**: AUTOWIRE-21, AUTOWIRE-23
**Gap**: gap 5

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] `ClientTarget` expõe o campo de presença de instalação, com nome e tipo documentados
- [x] O campo é consumido pelo detector (T8) e pela CLI (T12) — nenhum campo morto
- [x] O gate rápido passa: `go -C D:\repository\my-memory build ./...` e `go -C D:\repository\my-memory vet ./...`

**Tests**: none (camada de entidade — o campo é asserido pelo detector, em T9)
**Gate**: quick

**Commit**: `feat(autowire): o alvo de cliente carrega a presenca de instalacao`

---

### T8: O detector marca a presença pelo marcador de instalação do cliente

**What**: `internal/autowire/detector.go` passa a marcar cada alvo com a
**presença de instalação do cliente na máquina**, consultada através do
`SystemEnv` já injetável (`:11-16`) e do marcador por tipo de cliente (Claude
Desktop, Cursor, VS Code, Windsurf) por sistema operacional — e não pelo
`ParentDirOK` do alvo, que é um diretório de workspace e não diz se a
ferramenta existe.
**Where**: `internal/autowire/detector.go`
**Depends on**: T7
**Reuses**: a interface `SystemEnv` (`internal/autowire/detector.go:11-16`) e o
`Stat` já usado em `makeClientTarget` (`:199-217`) · os marcadores globais já
resolvidos pelo detector (`:79`, `:120`, `:147`) · a decisão A7 da spec
**Requirement**: AUTOWIRE-21, AUTOWIRE-23
**Gap**: gap 5

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] A presença de cada cliente é derivada de um marcador **de máquina** (não do diretório de workspace), resolvido por sistema operacional
- [x] A presença é consultada pelo `SystemEnv` injetável — o detector de teste consegue fixar presença e ausência sem tocar na máquina
- [x] O marcador do Claude Desktop é o diretório de dados do Claude; o do Cursor, o diretório global do Cursor; o do VS Code, o diretório de dados do VS Code; o do Windsurf, o diretório global do Windsurf
- [x] Um alvo whose client is absent sai marcado como ausente, e um whose client is present sai marcado como presente
- [x] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestDetect ./internal/autowire/...`
- [x] Test count: os testes de `internal/autowire` passam (9 no baseline + os novos), sem nenhuma deleção silenciosa

**Tests**: unit
**Gate**: quick

**Commit**: `feat(autowire): o detector marca a presenca do cliente instalado`

---

### T9: O domínio assere o alvo aceito e a presença do cliente

**What**: um conjunto de testes em `internal/autowire/detector_test.go` que
fixa (a) o **conjunto aceito** de `--target`, (b) o sinônimo `claude-desktop`
(resolve o Claude e **não** resolve os outros três), (c) a **presença** de cada
cliente a partir do marcador de máquina, com o `SystemEnv` de teste fixando
presença e ausência, e (d) o **controle negativo**: um cliente ausente sai
marcado como ausente.
**Where**: `internal/autowire/detector_test.go`
**Depends on**: T6, T8
**Reuses**: o `SystemEnv` de teste e o `fakeEnv` já existente no arquivo
(`internal/autowire/detector_test.go`) · os caminhos por SO já asseverados
(`:50`, `:129`, `:150`) · o padrão de asserção do filtro de escopo (`:164`, `:175`)
**Requirement**: AUTOWIRE-16, AUTOWIRE-17, AUTOWIRE-18, AUTOWIRE-23
**Gap**: gap 4, gap 5

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] Há um teste para cada valor do conjunto aceito e um para valor fora dele
- [x] `claude-desktop` resolve **exatamente** o alvo do Claude Desktop (conjunto exato, não contagem)
- [x] Com o cliente marcado como ausente, o alvo sai marcado como ausente; com o cliente presente, sai presente (controle positivo e negativo no mesmo teste)
- [x] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestDetect ./internal/autowire/...`
- [x] Test count: os testes de `internal/autowire` passam (9 no baseline + os novos), sem nenhuma deleção silenciosa

**Tests**: unit
**Gate**: quick

**Commit**: `test(autowire): o alvo aceito e a presenca do cliente sao aseridos`

---

### T10: As flags inválidas do `mem install` falham alto em vez de sair com 0

**What**: `cmd/mem/install.go` passa a validar o valor de `--target` contra o
conjunto aceito do domínio (T6) e, quando o valor não pertence ao conjunto,
**devolve erro** — que a CLI traduz em **exit ≠ 0** (`cmd/mem/main.go:836-840`)
e uma mensagem que **lista os valores aceitos**. A mensagem "Nenhum cliente MCP
compatível encontrado" (`:90-93`) deixa de ser a resposta para um valor
inválido; ela passa a ser o caso "valor válido, nenhum cliente instalado". Na
mesma bloco, o cálculo de escopo (`:39-45`) passa a **rejeitar** `--workspace`
e `--global` juntas, que hoje caem no escopo padrão e instalam em **todos** os
escopos.
**Where**: `cmd/mem/install.go`
**Depends on**: T6
**Reuses**: o conjunto aceito exposto pelo domínio (T6) · o `errOverride`/
padrão de erro da CLI (`cmd/mem/storage_override.go:30-36`) · a decisão A6 e a
decisão A11 da spec
**Requirement**: AUTOWIRE-19, AUTOWIRE-20, AUTOWIRE-07
**Gap**: gap 4, gap 2 (M8)

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] Um `--target` fora do conjunto aceito devolve erro e a CLI sai com código **≠ 0**
- [x] A mensagem de erro lista os 6 valores aceitos
- [x] A mensagem "Nenhum cliente MCP compatível encontrado" **não** é mais a resposta para um valor inválido
- [x] `--workspace` com `--global` devolve erro e a CLI sai com código **≠ 0**, dizendo que as duas flags são mutuamente exclusivas
- [x] `--target all` e `--target claude` continuam funcionando (o caso "nenhum instalado" é tratado em T12)
- [x] O gate rápido passa: `go -C D:\repository\my-memory build ./...` e `go -C D:\repository\my-memory vet ./...`

**Tests**: integration
**Gate**: quick

**Commit**: `fix(cli): o target invalido do mem install falha com exit nao zero`

---

### T11: O payload do `mem install` respeita a precedência de storage

**What**: `cmd/mem/install.go` passa a usar o valor de `pg` devolvido por
`resolveStorageAndRepo` (`:63`, hoje descartado) e a montar o payload
conforme o engine vencedor: com **Postgres**, `--storage postgres` e **nenhum**
`--db`; com **SQLite**, `--db <caminho>` e nenhum `--storage`. A URL do
Postgres **não** vai para o payload. A saída passa a imprimir a linha
`Engine: postgres` / `Engine: sqlite`, e a linha `Database:` só aparece no caso
SQLite.
**Where**: `cmd/mem/install.go`
**Depends on**: T2
**Reuses**: o `pg` de `resolveStorageAndRepo` (`cmd/mem/main.go:1381,1438`) · a
precedência da ADR-040 (`cmd/mem/storage_source.go:34-51`) · o `--storage` do
`mem mcp` (`cmd/mem/main.go:373`) · as decisões A3, A4 e A5 da spec
**Requirement**: AUTOWIRE-08, AUTOWIRE-09, AUTOWIRE-10, AUTOWIRE-11, AUTOWIRE-12, AUTOWIRE-13, AUTOWIRE-14, AUTOWIRE-15
**Gap**: gap 3

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] Com o Postgres resolvido, o payload declara `--storage postgres` e **não** leva `--db`
- [x] Com o SQLite resolvido, o payload leva `--db <caminho>` e **não** leva `--storage postgres`
- [x] A URL do Postgres **não** aparece no payload nem no arquivo escrito no editor
- [x] A saída imprime `Engine: postgres` ou `Engine: sqlite`, e a linha `Database:` só sai no caso SQLite
- [x] O log de storage e o payload dizem o **mesmo** engine (nenhuma linha afirma uma coisa e entrega outra)
- [x] O gate rápido passa: `go -C D:\repository\my-memory build ./...` e `go -C D:\repository\my-memory vet ./...`

**Tests**: integration
**Gate**: quick

**Commit**: `fix(cli): o payload do mem install respeita a precedencia de storage`

---

### T12: O `mem install` escreve só onde o cliente está instalado

**What**: `cmd/mem/install.go` passa a consumir a **presença de instalação**
marcada pelo detector (T8): um alvo cujo cliente não está na máquina é
**pulado** — nenhum arquivo, nenhum diretório criado — e sai na saída com o
status de pulo, o **nome do cliente** e o **motivo**. Um alvo cujo cliente está
instalado é configurado como antes. Em `--dry-run`, o cliente ausente também
**não** é listado como alvo a instalar.
**Where**: `cmd/mem/install.go`
**Depends on**: T8
**Reuses**: a presença marcada no `ClientTarget` (T7/T8) · o `StatusSkipped`
(`internal/autowire/client.go:47`) · o laço de injeção (`:109-142`) · a decisão
A7 e A9 da spec
**Requirement**: AUTOWIRE-21, AUTOWIRE-22
**Gap**: gap 5

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] Um alvo cujo cliente está ausente **não** tem arquivo nem diretório criados
- [x] O cliente pulado aparece na saída com o status de pulo, o nome dele e o motivo
- [x] Em `--dry-run`, um cliente ausente **não** é listado como alvo a instalar
- [x] Um alvo cujo cliente está presente é configurado (controle positivo no mesmo caminho de código)
- [x] Se **nenhum** cliente estiver instalado, o comando sai com **exit 0** e a saída diz que nada foi configurado
- [x] O gate rápido passa: `go -C D:\repository\my-memory build ./...` e `go -C D:\repository\my-memory vet ./...`

**Tests**: integration
**Gate**: quick

**Commit**: `fix(cli): o mem install so escreve onde o cliente esta instalado`

---

### T13: O backup `.bak` passa a ser atômico

**What**: `internal/autowire/installer.go` passa a gravar o backup num arquivo
**temporário no mesmo diretório** e a movê-lo para o destino com `os.Rename`
(`:105-108`), em vez de `os.WriteFile` direto — para que o `.bak` nunca fique
truncado, que é exatamente o cenário que o backup existe para cobrir.
**Where**: `internal/autowire/installer.go`
**Depends on**: T1
**Reuses**: a gravação atual (`:104-109`) · o `report.BackupPath` (`:107`) · a
atomicidade já prometida pela ADR-026:19 e pelo goal G2 da spec de origem
**Requirement**: AUTOWIRE-25
**Gap**: gap 6

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O `.bak` é escrito num temporário no **mesmo** diretório e publicado com `os.Rename`
- [ ] Um `.bak` anterior **não** é apagado se a gravação do novo falhar
- [ ] O `report.BackupPath` continua apontando para o `.bak` final
- [ ] O gate rápido passa: `go -C D:\repository\my-memory build ./...` e `go -C D:\repository\my-memory vet ./...`

**Tests**: integration
**Gate**: quick

**Commit**: `fix(autowire): o backup do config passa a ser atomico`

---

### T14: As asserções que matam M7 e M8

**What**: `cmd/mem/install_test.go` ganha os casos **negativos** que faltavam —
com `--target cursor --workspace`, o `.vscode/mcp.json` **não** existe e a
saída **não** menciona VS Code; com `--global`, nenhum `.cursor/mcp.json` nem
`.vscode/mcp.json` é criado dentro do `--dir`; com `--workspace` e `--global`
juntas, o comando falha com código **≠ 0**. É a guarda que mata os dois
mutantes que sobreviveram à suíte inteira.
**Where**: `cmd/mem/install_test.go`
**Depends on**: T10, T11, T12
**Reuses**: o padrão de teste existente (`cmd/mem/install_test.go:38-83`, só o
lado positivo) · o `t.TempDir()` como `--dir` · o buffer de saída capturado
**Requirement**: AUTOWIRE-03, AUTOWIRE-04, AUTOWIRE-05, AUTOWIRE-06, AUTOWIRE-07
**Gap**: gap 2 (M7, M8)

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] Com `--target cursor --workspace`, o `.vscode/mcp.json` **não** existe e a saída **não** menciona VS Code
- [ ] Com `--global`, nenhum `.cursor/mcp.json` nem `.vscode/mcp.json` é criado dentro do `--dir`
- [ ] Com `--workspace` e `--global` juntas, o comando falha com código **≠ 0**
- [ ] O teste **falha** se o filtro de alvo ou o de escopo for ignorado (é o que mata M7 e M8)
- [ ] O teste usa `t.TempDir()` e não escreve em nenhum config real do usuário
- [ ] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestInstall ./cmd/mem/...`
- [ ] Test count: os testes de `cmd/mem` passam (261 no baseline + os novos), sem nenhuma deleção silenciosa

**Tests**: integration
**Gate**: quick

**Commit**: `test(cli): o mem install ganha as asercoes de alvo e escopo`

---

### T15: O payload por engine é asserido

**What**: `cmd/mem/install_test.go` ganha os casos do payload: com o Postgres
resolvido (`MY_MEMORY_PG_URL` apontando para uma porta fechada — **sem
contato** com banco nenhum), o payload declara `--storage postgres`, **não**
leva `--db`, **não** contém a URL, e a saída imprime `Engine: postgres` e
**não** nomeia um `.db`; com o SQLite, o payload leva `--db <caminho>` e a
saída imprime `Engine: sqlite`.
**Where**: `cmd/mem/install_test.go`
**Depends on**: T11
**Reuses**: o `resolveStorageAndRepo` (`cmd/mem/main.go:1381`) · a precedência da
ADR-040 (`cmd/mem/storage_source.go:34-51`) · o `t.Setenv` para fixar a
variável de ambiente · o `--dry-run` (que não escreve em disco)
**Requirement**: AUTOWIRE-08, AUTOWIRE-09, AUTOWIRE-10, AUTOWIRE-11, AUTOWIRE-12, AUTOWIRE-13, AUTOWIRE-14, AUTOWIRE-15
**Gap**: gap 3

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] Com Postgres resolvido, o payload do dry-run declara `--storage postgres` e **não** contém `--db`
- [ ] Com Postgres resolvido, o payload **não** contém a URL e a saída **não** nomeia um `.db`
- [ ] Com Postgres resolvido, a saída contém `Engine: postgres`
- [ ] Com SQLite, o payload contém `--db <caminho>` e a saída contém `Engine: sqlite`
- [ ] O teste **não** abre conexão com nenhum banco (a porta do Postgres pode estar fechada)
- [ ] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestInstall ./cmd/mem/...`
- [ ] Test count: os testes de `cmd/mem` passam (261 no baseline + os novos), sem nenhuma deleção silenciosa

**Tests**: integration
**Gate**: quick

**Commit**: `test(cli): o payload do mem install e aserido por engine`

---

### T16: A presença do cliente e o `--target` inválido são asseridos

**What**: `cmd/mem/install_test.go` ganha os casos da **presença** e do
**`--target` inválido**: com o probe fixando o Cursor ausente e o VS Code
presente, o `.cursor/mcp.json` **não** é criado, a saída diz "pulado" com o
nome do Cursor, e o `.vscode/mcp.json` é criado; e um `--target` fora do
conjunto falha com código **≠ 0** e mensagem que lista os valores aceitos.
**Where**: `cmd/mem/install_test.go`
**Depends on**: T10, T12
**Reuses**: o probe de presença injetado (T8/T12) · o fake de ambiente com
`Stat` por caminho · o padrão de captura de erro da CLI
**Requirement**: AUTOWIRE-19, AUTOWIRE-20, AUTOWIRE-21, AUTOWIRE-22, AUTOWIRE-23, AUTOWIRE-24
**Gap**: gap 4, gap 5

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O teste fixa a presença de cada cliente por probe injetado — o resultado **não** depende das ferramentas instaladas na máquina que roda o teste
- [ ] Com o cliente ausente, nenhum arquivo dele é criado e a saída o nomeia como pulado
- [ ] Em `--dry-run` num diretório vazio, a saída **não** lista cliente inexistente como alvo a instalar
- [ ] Com o cliente presente, o arquivo dele é criado (controle positivo — o teste não é vacuamente verde)
- [ ] Com `--target` inválido, o comando falha com código **≠ 0** e a mensagem lista os valores aceitos
- [ ] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestInstall ./cmd/mem/...`
- [ ] Test count: os testes de `cmd/mem` passam (261 no baseline + os novos), sem nenhuma deleção silenciosa

**Tests**: integration
**Gate**: quick

**Commit**: `test(cli): a presenca do cliente e o target invalido sao aseridos`

---

### T17: O backup atômico e a formatação canônica são asseridos

**What**: `internal/autowire/installer_test.go` ganha os casos do backup
**atômico** (o `.bak` é byte-idêntico ao arquivo substituído, e um mutante que
troca o rename por `os.WriteFile` direto **morre**) e da **formatação
canônica** (2 espaços, chaves de topo em ordem alfabética, com as chaves e os
servidores pré-existentes preservados).
**Where**: `internal/autowire/installer_test.go`
**Depends on**: T13
**Reuses**: os testes de merge e de idempotência já existentes
(`internal/autowire/installer_test.go:32`, `:101`, `:113`, `:154`) · o fixture de
JSON com 4 espaços e chaves fora de ordem · a serialização real
(`internal/autowire/installer.go:75`)
**Requirement**: AUTOWIRE-25, AUTOWIRE-26, AUTOWIRE-27, AUTOWIRE-28
**Gap**: gap 6, gap 7

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O `.bak` final é **byte-idêntico** ao arquivo de configuração substituído
- [ ] Um mutante que substitui o rename por `os.WriteFile` direto **derruba** o teste
- [ ] O config escrito usa **2 espaços** e as chaves de topo em **ordem alfabética**
- [ ] As chaves de topo e os servidores pré-existentes continuam no arquivo (conteúdo preservado)
- [ ] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestInstaller ./internal/autowire/...`
- [ ] Test count: os testes de `internal/autowire` passam (9 no baseline + os novos), sem nenhuma deleção silenciosa

**Tests**: integration
**Gate**: quick

**Commit**: `test(autowire): o backup atomico e a formatacao sao aseridos`

---

### T18: A feature de origem é re-verificada por um Verifier independente

**What**: despachar um Verifier **independente** (≠ o autor de T1..T17) que
(a) re-derive a cobertura da feature `ai-tool-auto-wiring` ancorada na spec
emendada, com evidência `file:line` para as 8 ACs de origem; (b) rode o
**sensor de discriminação** em worktree isolado re-injetando os **9** alvos do
sensor anterior — com destaque para **M7** e **M8**, os que sobreviveram — e
confirme que **cada um** agora morre; (c) escreva o relatório
`.specs/features/ai-tool-auto-wiring/validation.md` com o veredito novo,
citando **apenas testes que existem** e declarando que os 8 nomes do plano de
validação anterior nunca existiram.
**Where**: `.specs/features/ai-tool-auto-wiring/validation.md`
**Depends on**: T14, T15, T16, T17
**Reuses**: os 9 alvos de mutação do sensor anterior (`cmd/mem/install.go`,
`internal/autowire/detector.go`, `internal/autowire/installer.go`) · o worktree
isolado (sem `git stash`) · as decisões A1 a A11 da spec como contexto do
contrato
**Requirement**: AUTOWIRE-33, AUTOWIRE-34, AUTOWIRE-35, AUTOWIRE-36, AUTOWIRE-37
**Gap**: gap 10 (e o ciclo fix → re-verificar dos gaps 1 a 9)

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O Verifier é um agente **distinto** do autor de T1..T17 (autor ≠ verificador)
- [ ] O sensor mata **M7** e **M8** — os dois que sobreviveram
- [ ] Os 7 mutantes que já morriam continuam mortos, e o sensor fecha em **9/9**
- [ ] As 8 ACs da feature de origem têm evidência `file:line` no relatório
- [ ] O relatório cita **apenas** funções de teste que existem no repositório
- [ ] O relatório declara que os 8 nomes do plano de validação anterior nunca existiram
- [ ] O veredito do relatório é **PASS** e `validate_state.py ai-tool-auto-wiring` sai **exit 0**
- [ ] A árvore real está intacta no fim (baseline e final idênticos, `git status --porcelain` limpo) — **sem `git stash`**
- [ ] O Postgres do dono **não** foi contatado e nenhum banco foi criado ou apagado

**Tests**: integration
**Gate**: full

**Commit**: `docs(specs): re-verificacao da feature ai-tool-auto-wiring`

---

## Phase Execution Map

```
Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5

Phase 1:  T1 · T2 · T3 · T4 · T5
Phase 2:  T6 · T7 · T8 · T9
Phase 3:  T10 · T11 · T12 · T13
Phase 4:  T14 · T15 · T16 · T17
Phase 5:  T18
```

Execution is strictly sequential - there is no intra-phase parallelism. A single agent (or batch worker) works one task at a time, in order.

**Contagem de tarefas**: 18 — empacotam em **3 lotes** de ~6 tasks (fases 1+2 no primeiro, 3+4 no segundo, 5 no terceiro), então a execução **oferece** sub-agentes de lote ao dono. A T18 despacha o **Verifier** independente, que é parte do fluxo do `tlc-spec-driven` (nunca opcional, nunca perguntado).

---

## Diagram-Definition Cross-Check

| Task | Depends On (task body) | Diagram Shows | Status |
| ---- | ---------------------- | ------------- | ------ |
| T1 | None | (início da fase 1) | ✅ Match |
| T2 | T1 | T1 -> T2 | ✅ Match |
| T3 | T1 | T1 -> T3 | ✅ Match |
| T4 | T1 | T1 -> T4 | ✅ Match |
| T5 | T4 | T4 -> T5 | ✅ Match |
| T6 | T1 | T1 -> T6 | ✅ Match |
| T7 | None | (início da fase 2) | ✅ Match |
| T8 | T7 | T7 -> T8 | ✅ Match |
| T9 | T6, T8 | T6 -> T9 · T8 -> T9 | ✅ Match |
| T10 | T6 | T6 -> T10 | ✅ Match |
| T11 | T2 | T2 -> T11 | ✅ Match |
| T12 | T8 | T8 -> T12 | ✅ Match |
| T13 | T1 | T1 -> T13 | ✅ Match |
| T14 | T10, T11, T12 | T10 -> T14 · T11 -> T14 · T12 -> T14 | ✅ Match |
| T15 | T11 | T11 -> T15 | ✅ Match |
| T16 | T10, T12 | T10 -> T16 · T12 -> T16 | ✅ Match |
| T17 | T13 | T13 -> T17 | ✅ Match |
| T18 | T14, T15, T16, T17 | T14 -> T18 · T15 -> T18 · T16 -> T18 · T17 -> T18 | ✅ Match |

Nenhuma dependência aponta para uma fase posterior.

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | Contrato publicado (ADR-026) | none | none | ✅ OK (camada de contrato — o gate é o build + a leitura cruzada) |
| T2 | Contrato publicado (CLI guide) | none | none | ✅ OK (camada de contrato) |
| T3 | Contrato publicado (guia de agentes) | none | none | ✅ OK (camada de contrato) |
| T4 | Contrato publicado (spec da origem) | none | none | ✅ OK (camada de contrato) |
| T5 | Contrato publicado (tasks da origem) | none | none | ✅ OK (camada de contrato) |
| T6 | Domínio de clientes | unit | unit | ✅ OK |
| T7 | Entidade de cliente (`ClientTarget`) | none | none | ✅ OK (camada de entidade — o campo é asserido pelo detector, em T9) |
| T8 | Domínio de clientes | unit | unit | ✅ OK |
| T9 | Domínio de clientes | unit | unit | ✅ OK |
| T10 | Fronteira CLI do instalador | integration | integration | ✅ OK |
| T11 | Fronteira CLI do instalador | integration | integration | ✅ OK |
| T12 | Fronteira CLI do instalador | integration | integration | ✅ OK |
| T13 | Escrita do config e backup | integration | integration | ✅ OK |
| T14 | Fronteira CLI do instalador | integration | integration | ✅ OK |
| T15 | Fronteira CLI do instalador | integration | integration | ✅ OK |
| T16 | Fronteira CLI do instalador | integration | integration | ✅ OK |
| T17 | Escrita do config e backup | integration | integration | ✅ OK |
| T18 | Verificação independente + sensor | integration | integration | ✅ OK |

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: ADR-026 descreve o código real | 1 documento normativo | ✅ Granular |
| T2: CLI guide sem o alias | 1 documento | ✅ Granular |
| T3: guia de agentes com o `mem install` | 1 documento | ✅ Granular |
| T4: spec da origem emendada | 1 documento | ✅ Granular |
| T5: tasks da origem emendadas | 1 documento | ✅ Granular |
| T6: conjunto de alvos + sinônimo | 1 função + 1 tabela no detector | ✅ Granular |
| T7: campo de presença no `ClientTarget` | 1 struct | ✅ Granular |
| T8: marcador de presença no detector | 1 função no detector | ✅ Granular |
| T9: asserções do domínio | 1 arquivo de teste | ✅ Granular |
| T10: validação das flags do instalador | 1 bloco da CLI (alvo inválido + escopo contraditório, no mesmo bloco de parsing de flags) | ✅ Granular |
| T11: payload por engine | 1 bloco da CLI | ✅ Granular |
| T12: pulo por ausência | 1 bloco da CLI | ✅ Granular |
| T13: backup atômico | 1 função do instalador | ✅ Granular |
| T14: asserções de alvo e escopo | 1 arquivo de teste | ✅ Granular |
| T15: asserções do payload | 1 arquivo de teste | ✅ Granular |
| T16: asserções de presença e de erro | 1 arquivo de teste | ✅ Granular |
| T17: asserções do backup e da formatação | 1 arquivo de teste | ✅ Granular |
| T18: re-verificação independente | 1 relatório de verificação | ✅ Granular |
