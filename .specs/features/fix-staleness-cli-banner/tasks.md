# Tasks: fix-staleness-cli-banner

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-16
**Issue**: ISSUE-043 (`.specs/ISSUES.md`) · fecha os 4 gaps da feature `staleness-detection-and-banners` (ADR-028)

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Design**: inline na spec (o literal canônico, a categoria "novos" e o gancho de CLI; **nenhum ADR novo**)
**Status**: Draft
**Consolidação (2026-11-16)**: esta planilha é a **versão final única** da ISSUE-043 e absorve a planilha paralela `.specs/features/fix-staleness-banner-wiring/` (12 tasks), que cobria a mesma issue com a decisão **oposta** sobre o literal canônico (emendar a ADR-028 e traduzir o banner MCP para PT-BR). As tasks T1 e T2 da versão paralela — as duas que só faziam sentido sob essa decisão — **não** foram copiadas: a ADR-028 **fica intocada** e é o **código** que se move para o literal dela (`docs/adr/028-...md:45`). O que era material foi incorporado: o contrato de string vazia dos formatadores (T2), a nota explícita do checkbox mentiroso (T11) e o edge case do cache em sequência (T3). A planilha paralela é removida pelo dono; as duas não podem ser implementadas juntas.

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec. Guidelines found: `AGENTS.md` (regra 4 — testes obrigatórios; regra 8 — quality gate), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`, `.github/workflows/ci.yml` (comandos de teste reais do projeto).

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Motor de staleness (domínio: relatório, 3 categorias, formatadores) | unit | 1:1 com FSC-17 a FSC-20 (separação novo/modificado, contagens, `IsStale`), FSC-09/FSC-10 (literal canônico dos dois formatadores) e FSC-11 (string vazia fora do estado stale); todos os edge cases de lista: novo, modificado, deletado, in sync, `.git` excluído, cache TTL | `internal/staleness/detector_test.go` | `go -C D:\repository\my-memory test -count=1 ./internal/staleness/...` |
| Banner MCP (handler + detector) | integration | 1:1 com FSC-09: a resposta da tool carrega o **literal canônico** da ADR-028 §3 e, em sincronia, não carrega banner nenhum | `internal/mcp/staleness_handlers_test.go` | `go -C D:\repository\my-memory test -count=1 ./internal/mcp/...` |
| Gancho do aviso na CLI (helper + `search`/`path`/`inspect`/`impact`) | integration | 1:1 com FSC-01 a FSC-08: banner presente **antes** do 1º resultado, ausente quando in sync, degradação silenciosa na falha da checagem, reuso do cache de 3 s, e stdout **só JSON** com `--json` | `cmd/mem/staleness_banner_test.go` · `cmd/mem/search_staleness_banner_test.go` · `cmd/mem/path_staleness_banner_test.go` · `cmd/mem/inspect_staleness_banner_test.go` · `cmd/mem/impact_staleness_banner_test.go` | `go -C D:\repository\my-memory test -count=1 ./cmd/mem/...` |
| Listagem e literais do `mem status` | integration | 1:1 com FSC-12, FSC-13, FSC-21 a FSC-24: três headings, caminho relativo, ordenação estável, truncamento em 10 + `... e mais N`, e `new_files`/`new_files_count` no `--json` | `cmd/mem/status_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestRunStatusCommand ./cmd/mem/...` |
| Documentação publicada | none | — (build gate only): as ACs de doc (FSC-14, FSC-15) são verificadas por leitura do arquivo e por `grep` do literal aposentado, no mesmo commit | `docs/AGENT_INTEGRATION_GUIDE.md` · `docs/CLI_GUIDE.md` | build gate only |
| ADR-028 (norma, intocada) | none | — (build gate only): FSC-16 é verificado pelo diff (`git diff --stat` não toca a ADR) | `docs/adr/028-staleness-banners-e-deteccao-de-desatualizacao.md` | build gate only |
| Artefatos de spec (bookkeeping da ISSUE e do `tasks.md` de origem) | none | — (build gate only): FSC-25 e FSC-26 são verificadas por leitura do arquivo no mesmo commit | `.specs/ISSUES.md` · `.specs/features/staleness-detection-and-banners/tasks.md` | build gate only |
| Re-verificação independente + sensor | integration | 1:1 com FSC-27 e FSC-28: o sensor mata os mutantes dos **quatro** gap sites e o veredito da feature corrigida é PASS com evidência `file:line` | `.specs/features/staleness-detection-and-banners/validation.md` | `python3 .agents/skills/tlc-spec-driven/scripts/validate_state.py staleness-detection-and-banners --root D:\repository\my-memory` |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após T1 e T2 (pacote `internal/staleness`) | `go -C D:\repository\my-memory test -count=1 ./internal/staleness/...` |
| Quick | Após T3 a T8 (pacote `cmd/mem`) | `go -C D:\repository\my-memory test -count=1 ./cmd/mem/...` |
| Quick | Após cada task (build + vet) | `go -C D:\repository\my-memory build ./...` e `go -C D:\repository\my-memory vet ./...` |
| Full | Gate final | `go -C D:\repository\my-memory test -count=1 ./...` (com `MY_MEMORY_PG_TEST=1` e `MY_MEMORY_PG_URL` preservada) |
| Full | Gate final taggeado | `go -C D:\repository\my-memory test -tags "sqlite_fts5 onnx" -count=1 ./...` |
| Build | Antes de declarar pronto | `go -C D:\repository\my-memory build ./...` |
| Format | Antes de declarar pronto | `gofmt -l` nos `.go` tocados, medido em **cópia LF** (o working tree local mistura CRLF — o `gofmt` do repo aponta 134 arquivos por isso, e a CI roda em Linux) |
| Spec | Antes de entregar | `python3 .agents/skills/tlc-spec-driven/scripts/validate_spec.py fix-staleness-cli-banner --root D:\repository\my-memory` e `python3 .agents/skills/tlc-spec-driven/scripts/validate_tasks.py fix-staleness-cli-banner --root D:\repository\my-memory` |
| State | Antes de entregar | `python3 .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-staleness-cli-banner --root D:\repository\my-memory` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: O relatório distingue três categorias

O motor separa "novo" de "modificado" — é o que dá ao banner e à listagem as três contagens que a ADR-028 §3 promete.

```
T1
```

### Phase 2: O literal canônico

O código passa a emitir o texto da ADR-028 §3 e o `mem status` passa a emitir os literais do spec. T2 depende de T1 porque o banner precisa das três contagens.

```
T1 → T2
```

### Phase 3: O gancho que faltava na CLI (Blocker)

O helper compartilhado (T3) e a ligação dele nos quatro comandos de consulta (T4 a T7). As quatro tasks são **independentes entre si** (arquivos diferentes, nenhum compartilhado) e todas dependem só do T3.

```
T2 → T3
T3 → T4
T3 → T5
T3 → T6
T3 → T7
```

### Phase 4: A listagem e os literais do `mem status`

```
T2 → T8
```

### Phase 5: Documentação, fechamento e re-verificação

```
T2 → T9
T8 → T10
T4 → T11
T5 → T11
T6 → T11
T7 → T11
T8 → T12
T9 → T12
T10 → T12
T11 → T12
T12 → T13
```

---

## Task Breakdown

### Phase 1: O relatório distingue três categorias

### T1: "Novo" e "modificado" viram categorias separadas no relatório

**What**: `StalenessReport` ganha `NewFiles []string` e `NewFilesCount int`; o
`WalkDir` do detector deixa de jogar "arquivo ausente do índice" na lista de
modificados e passa a jogá-lo em `newFiles` (`internal/staleness/detector.go:225-227`),
enquanto o mtime mais novo que o persistido continua indo para `staleFiles`
(`:244-245`). `IsStale` passa a ser verdadeiro quando **qualquer** das três
categorias é não vazia (`internal/staleness/detector.go:286`), e as três listas
continuam ordenadas (`sort.Strings`, `:282-283`).
**Where**: `internal/staleness/detector.go` + `internal/staleness/detector_test.go` (teste co-localizado)
**Depends on**: None
**Reuses**: `store.DocumentMeta` e `db.GetDocumentsMetadata` (o caminho de
persistência **não** muda) · a tolerância de 1 s do mtime (`:243-244`) · o padrão
de fixture `setupTestEnvironment` do próprio `detector_test.go:41`
**Requirement**: FSC-17, FSC-18, FSC-19, FSC-20
**Gap**: P2-2

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] `StalenessReport` tem `NewFiles []string` (tag JSON `new_files`) e `NewFilesCount int` (tag JSON `new_files_count`)
- [x] Um arquivo presente em disco e ausente do índice aparece **só** em `NewFiles`
- [x] Um arquivo com mtime mais novo que o persistido aparece **só** em `StaleFiles`
- [x] `IsStale` é verdadeiro com **qualquer** das três categorias não vazia (inclusive só "novos")
- [x] `TestDetector_NewUnindexedFile` (`internal/staleness/detector_test.go:147`) passa a exigir `NewFilesCount == 1` — **atualizado, nunca removido**
- [x] Um teste novo fixa a separação: vault com 1 arquivo novo + 1 modificado + 1 deletado → 1 em cada lista
- [x] Nenhuma função de teste foi apagada e nenhuma asserção foi afrouxada
- [x] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./internal/staleness/...`
- [x] `go build ./...` e `go vet ./...` saem exit 0

**Tests**: unit
**Gate**: quick

**Commit**: `feat(staleness): separa arquivos novos de modificados no relatorio`

---

### Phase 2: O literal canônico

### T2: O banner passa a emitir o texto da ADR-028 §3

**What**: `internal/staleness` ganha a **fonte única** do literal: o título
(`Staleness Warning`) e a recomendação (`Execute \`mem index\` ou inicie \`mem watch\``)
viram constantes exportadas, e `FormatMarkdownBanner`
(`internal/staleness/detector.go:298-306`) passa a emitir **exatamente** a linha da
ADR-028 §3 (`docs/adr/028-...md:45`), com `N = X + Y + Z` pendentes, `X`
modificados, `Y` novos e `Z` deletados. `FormatCLIBanner` (`:309-315`) passa a
emitir a linha canônica da CLI com as mesmas três contagens e a mesma
recomendação.
**Where**: `internal/staleness/detector.go` + `internal/staleness/detector_test.go` + `internal/mcp/staleness_handlers_test.go` (testes co-localizados)
**Depends on**: T1
**Reuses**: o `StalenessReport` de T1 (as três contagens) · o texto literal da
ADR-028 §3 · o padrão de asserção já existente em `detector_test.go:136-144` e
`staleness_handlers_test.go:94`
**Requirement**: FSC-09, FSC-10, FSC-11
**Gap**: P1-1

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] `FormatMarkdownBanner` emite a linha `> ⚠️ **Staleness Warning**: O índice de memória está desatualizado em relação aos arquivos no disco (N arquivo(s) pendente(s): X modificado(s), Y novo(s), Z deletado(s)). As respostas abaixo podem refletir um estado anterior. Execute \`mem index\` ou inicie \`mem watch\`.`
- [x] `FormatCLIBanner` emite `⚠️ [Aviso] O índice local está desatualizado (X modificado(s), Y novo(s), Z removido(s)). Execute \`mem index\` ou inicie \`mem watch\`.`
- [x] O título e a recomendação vivem em **constantes exportadas** de `internal/staleness`, e os dois formatadores as usam
- [x] O literal **aposentado** `> ⚠️ **AVISO: Memória Desatualizada (Stale Data)**` **não** é emitido por nenhum caminho
- [x] Relatório `nil` ou com `IsStale` falso devolve **string vazia** nos dois formatadores — o banner some
- [x] O teste fixa o **texto literal** (string hard-coded no arquivo de teste) e compara com a constante exportada — **não** afirma a constante contra si mesma
- [x] `internal/mcp/staleness_handlers_test.go:94` passa a asseverar o literal **canônico** (e o `:77` continua provando que não há banner em sincronia)
- [x] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./internal/staleness/... ./internal/mcp/...`
- [x] `go build ./...` e `go vet ./...` saem exit 0

**Tests**: unit
**Gate**: quick

**Commit**: `fix(staleness): adota o literal canonico do banner da ADR-028`

---

### Phase 3: O gancho que faltava na CLI (Blocker)

### T3: O helper compartilhado do aviso de staleness na CLI

**What**: um arquivo novo `cmd/mem/staleness_banner.go` com **um** helper que
todo comando de consulta chama: ele constrói o `staleness.Detector` a partir da
conexão que o comando já tem, chama **`CheckStaleness`** (o caminho cacheado,
TTL 3 s — nunca `ForceCheck`), e escreve `staleness.FormatCLIBanner(report)` no
`io.Writer` **só** quando `report.IsStale`. Qualquer erro na checagem é
**engolido**: o helper não retorna erro, não escreve nada e o comando segue.
**Where**: `cmd/mem/staleness_banner.go` + `cmd/mem/staleness_banner_test.go` (teste co-localizado)
**Depends on**: T2
**Reuses**: `staleness.NewDetector` e `staleness.FormatCLIBanner` (T2) ·
`resolveStorageAndRepo` (`cmd/mem/main.go:1381`) para resolver o engine ·
`staleness.DefaultTTL` (`internal/staleness/detector.go:22`) como TTL padrão
**Requirement**: FSC-05, FSC-06, FSC-07
**Gap**: P3-1

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] Existe **um** helper em `cmd/mem/staleness_banner.go` usado pelos quatro comandos de consulta
- [x] O helper escreve o banner **apenas** quando `report.IsStale` — com o índice em sincronia ele **não** escreve nada
- [x] O helper usa `CheckStaleness` (cache de 3 s) e **nunca** `ForceCheck`
- [x] O helper **não** retorna erro: uma checagem que falha não interrompe o comando nem escreve nada no `out`
- [x] O teste prova o reuso do cache: com TTL longo, a 1ª chamada escreve o banner, o diretório do vault é **removido**, a 2ª chamada **dentro do TTL** **ainda** escreve o banner (com `ForceCheck` a varredura falharia e o banner sumiria)
- [x] O teste prova que o **mesmo** comando em sequência não varre o disco de novo dentro do TTL
- [x] O teste prova a degradação silenciosa: um detector sem base configurada não escreve nada e não devolve erro
- [x] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 -run TestStalenessBanner ./cmd/mem/...`
- [x] `go build ./...` e `go vet ./...` saem exit 0

**Tests**: unit
**Gate**: quick

**Commit**: `feat(cli): adiciona o helper do aviso de staleness na CLI`

---

### T4: `mem search` imprime o aviso antes dos resultados

**What**: `runSearchSQLite` (`cmd/mem/main.go:2448`) e `runSearchPostgres`
(`:2310`) chamam o helper do T3 **antes** da linha de cabeçalho
`🔎 Buscando no SQLite por: …` (`cmd/mem/main.go:2463`), de modo que o aviso
seja a **primeira** linha do stdout. O caminho SQLite e o Postgres usam a
conexão que cada um já abre.
**Where**: `cmd/mem/main.go` + `cmd/mem/search_staleness_banner_test.go` (teste co-localizado)
**Depends on**: T3
**Reuses**: o helper do T3 · `captureStdout` (`cmd/mem/search_tag_test.go:330`) ·
o par `db.InitDB` + inserção de documento (`cmd/mem/status_test.go:19`)
**Requirement**: FSC-01
**Gap**: P3-1

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] `mem search` num vault stale escreve o banner **antes** da linha do primeiro resultado
- [x] Num vault em sincronia, `mem search` **não** escreve banner
- [x] O teste usa `t.TempDir()` e um banco temporário — não toca em nenhum vault nem em `.memory/memory.db`
- [x] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./cmd/mem/...`
- [x] `go build ./...` e `go vet ./...` saem exit 0

**Tests**: integration
**Gate**: quick

**Commit**: `feat(cli): emite o aviso de staleness no mem search`

---

### T5: `mem path` imprime o aviso antes dos resultados

**What**: `runPathCommand` (`cmd/mem/path.go:24`) chama o helper do T3 depois de
abrir a conexão (o `db.InitDB` de `:94` ou o `store.NewPostgresStore` de `:86`) e
**antes** da renderização `=== Descoberta de Rotas & Menor Caminho ===`
(`:114`). Com `--json` (`path.go:107-111`) o banner é **suprimido**: o stdout
continua sendo **só** o documento JSON.
**Where**: `cmd/mem/path.go` + `cmd/mem/path_staleness_banner_test.go` (teste co-localizado)
**Depends on**: T3
**Reuses**: o helper do T3 · o `out io.Writer` que `runPathCommand` já recebe
(`path.go:24`) · o padrão de fixture temporária de `cmd/mem/status_test.go:18`
**Requirement**: FSC-02, FSC-08
**Gap**: P3-1

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] `mem path` num vault stale escreve o banner **antes** do bloco de resultado
- [x] `mem path --json` escreve **só** o JSON: o banner não aparece e o documento continua desserializável
- [x] Num vault em sincronia, `mem path` **não** escreve banner
- [x] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./cmd/mem/...`
- [x] `go build ./...` e `go vet ./...` saem exit 0

**Tests**: integration
**Gate**: quick

**Commit**: `feat(cli): emite o aviso de staleness no mem path`

---

### T6: `mem inspect` imprime o aviso antes dos resultados

**What**: `runInspectCommand` (`cmd/mem/inspect.go:23`) chama o helper do T3
depois de abrir a conexão (`inspect.go:64` ou `:72`) e **antes** da renderização
do tríptico. Com `--json` (`inspect.go`) o banner é **suprimido**.
**Where**: `cmd/mem/inspect.go` + `cmd/mem/inspect_staleness_banner_test.go` (teste co-localizado)
**Depends on**: T3
**Reuses**: o helper do T3 · o `out io.Writer` que `runInspectCommand` já recebe
(`inspect.go:23`)
**Requirement**: FSC-03, FSC-08
**Gap**: P3-1

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [x] `mem inspect` num vault stale escreve o banner **antes** do tríptico
- [x] `mem inspect --json` escreve **só** o JSON: o banner não aparece e o documento continua desserializável
- [x] Num vault em sincronia, `mem inspect` **não** escreve banner
- [x] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./cmd/mem/...`
- [x] `go build ./...` e `go vet ./...` saem exit 0

**Tests**: integration
**Gate**: quick

**Commit**: `feat(cli): emite o aviso de staleness no mem inspect`

---

### T7: `mem impact` imprime o aviso antes dos resultados

**What**: `runImpactCommand` (`cmd/mem/impact.go:24`) chama o helper do T3
depois de abrir a conexão e **antes** da renderização `=== Análise de Impacto
(Blast Radius) ===` (`impact.go:90`). Com `--json` (`impact.go:83-86`) o banner
é **suprimido**.
**Where**: `cmd/mem/impact.go` + `cmd/mem/impact_staleness_banner_test.go` (teste co-localizado)
**Depends on**: T3
**Reuses**: o helper do T3 · o `out io.Writer` que `runImpactCommand` já recebe
(`impact.go:24`)
**Requirement**: FSC-04, FSC-08
**Gap**: P3-1

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] `mem impact` num vault stale escreve o banner **antes** da tabela de impacto
- [ ] `mem impact --json` escreve **só** o JSON: o banner não aparece e o documento continua desserializável
- [ ] Num vault em sincronia, `mem impact` **não** escreve banner
- [ ] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./cmd/mem/...`
- [ ] `go build ./...` e `go vet ./...` saem exit 0

**Tests**: integration
**Gate**: quick

**Commit**: `feat(cli): emite o aviso de staleness no mem impact`

---

### Phase 4: A listagem e os literais do `mem status`

### T8: `mem status` lista as três categorias e emite os literais canônicos

**What**: `runStatusCommand` (`cmd/mem/status.go:24`) passa a imprimir **três**
seções — modificados (`cmd/mem/status.go:132-145`), **novos** (seção nova) e
removidos (`:147-160`) — cada uma com o seu heading e a sua contagem; a linha
`Diferenças:` (`:130`) passa a trazer os **três** números; e as linhas de status
passam a ser `⚠️ DESATUALIZADO (Stale)` (`:129`) e `✅ ATUALIZADO (Fresh)`
(`:124`). O truncamento em 10 com `... e mais N arquivo(s)` (`:141-143`) é
mantido e passa a valer para as três categorias.
**Where**: `cmd/mem/status.go` + `cmd/mem/status_test.go` (teste co-localizado)
**Depends on**: T2
**Reuses**: `StalenessReport` de T1 (as três listas e as três contagens) · o
truncamento existente (`cmd/mem/status.go:134-143`) · o par `--json` (`:97-101`)
**Requirement**: FSC-12, FSC-13, FSC-21, FSC-22, FSC-23, FSC-24
**Gap**: P2-2, P2-3, P2-4

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O output humano tem **três** headings distintos: modificados, novos e removidos, cada um com a sua contagem
- [ ] A linha `Diferenças:` traz os três números (modificados, novos, removidos)
- [ ] O status stale é `⚠️ DESATUALIZADO (Stale)` e o fresh é `✅ ATUALIZADO (Fresh)`
- [ ] Cada caminho listado é **relativo à raiz do vault** e a lista está em **ordem lexicográfica crescente**
- [ ] Uma categoria com mais de 10 arquivos imprime os 10 primeiros e a linha `  ... e mais N arquivo(s)` com N correto
- [ ] O `--json` traz `new_files` e `new_files_count` com os mesmos valores da listagem humana
- [ ] `cmd/mem/status_test.go:46` e `:87` passam a asseverar os literais canônicos — **atualizados, nunca removidos**; e um teste novo assevera a **listagem** (caminho relativo, ordenação, truncamento), que hoje **não tem asserção nenhuma**
- [ ] O gate rápido passa: `go -C D:\repository\my-memory test -count=1 ./cmd/mem/...`
- [ ] `go build ./...` e `go vet ./...` saem exit 0

**Tests**: integration
**Gate**: quick

**Commit**: `fix(cli): lista novos, modificados e removidos no mem status`

---

### Phase 5: Documentação, fechamento e re-verificação

### T9: O guia do agente publica o banner canônico

**What**: o bloco de exemplo do banner em `docs/AGENT_INTEGRATION_GUIDE.md:396-400`
passa a mostrar a linha canônica da ADR-028 §3, no lugar do literal
aposentado `> ⚠️ **AVISO: Memória Desatualizada (Stale Data)**`; e o texto em
volta passa a dizer que a alternativa ao reindex manual é o `mem watch`.
**Where**: `docs/AGENT_INTEGRATION_GUIDE.md`
**Depends on**: T2
**Reuses**: o literal canônico de T2 · o formato de exemplo já existente (bloco
`markdown` com as três linhas)
**Requirement**: FSC-14
**Gap**: P1-1

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O guia mostra a linha canônica do banner MCP, com as três contagens e a recomendação `mem index` **e** `mem watch`
- [ ] O literal aposentado **não** aparece mais no arquivo (conferir com `grep`)
- [ ] `go build ./...` e `go vet ./...` saem exit 0

**Tests**: none
**Gate**: build

**Commit**: `docs(agents): publica o banner canonico de staleness`

---

### T10: O guia da CLI publica os literais e as seções canônicas

**What**: a amostra de `mem status` em `docs/CLI_GUIDE.md:1432-1436` passa a
mostrar o status `⚠️ DESATUALIZADO (Stale)`, a linha `Diferenças:` com os três
números e as três seções de listagem (modificados, novos, removidos).
**Where**: `docs/CLI_GUIDE.md`
**Depends on**: T8
**Reuses**: o output real de `mem status` (o mesmo que o teste do T8 assere)
**Requirement**: FSC-15
**Gap**: P2-3, P2-4

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] A amostra publica `⚠️ DESATUALIZADO (Stale)` e `✅ ATUALIZADO (Fresh)`
- [ ] A amostra publica as três seções de listagem e a linha `Diferenças:` com três números
- [ ] Os literais aposentados `⚠️  Desatualizado (Stale Data)` e `✅ Atualizado (In Sync)` **não** aparecem mais no arquivo (conferir com `grep`)
- [ ] `go build ./...` e `go vet ./...` saem exit 0

**Tests**: none
**Gate**: build

**Commit**: `docs(cli): publica os literais canonicos do mem status`

---

### T11: O `tasks.md` da feature de origem deixa de mentir

**What**: a task T3 da feature `staleness-detection-and-banners` — hoje
marcada `[x] Integrar alerta de staleness nos comandos de consulta (search,
path, inspect, impact)` — passa a citar a **evidência `file:line`** do gancho
que agora existe (o helper e as quatro chamadas), com o link para o commit da
correção.
**Where**: `.specs/features/staleness-detection-and-banners/tasks.md`
**Depends on**: T4, T5, T6, T7
**Reuses**: o `file:line` dos quatro pontos de chamada produzidos por T4 a T7 ·
os hashes dos commits de T4 a T7
**Requirement**: FSC-25
**Gap**: P3-1 (o checkbox que escondeu o Blocker)

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] A T3 da feature de origem cita o `file:line` do helper e das quatro chamadas
- [ ] O `[x]` só sobrevive **porque** o código existe — e o texto diz isso
- [ ] Há uma nota explícita de que o checkbox mentia (achado do Verifier de 2026-11-15) e de que a integração é feita por esta spec
- [ ] Nenhum outro arquivo da feature de origem é alterado (o `spec.md` é histórico)

**Tests**: none
**Gate**: build

**Commit**: `docs(specs): corrige o checkbox da integracao CLI de staleness`

---

### T12: A feature corrigida é re-verificada por um Verifier independente

**What**: despachar um Verifier **independente** (≠ o autor de T1..T11) que
(a) re-derive a cobertura da feature `staleness-detection-and-banners` ancorada
na spec de origem, com evidência `file:line` para as 9 ACs; (b) rode o **sensor
de discriminação** em worktree isolado re-injetando mutações nos **quatro** gap
sites — o gancho da CLI (`cmd/mem/staleness_banner.go` e as quatro chamadas), o
literal do banner (`internal/staleness/detector.go`), a separação de "novo"
(`internal/staleness/detector.go`) e os literais de status (`cmd/mem/status.go`)
— e confirme que **cada uma** é morta pelos testes novos; (c) escreva o
relatório `.specs/features/staleness-detection-and-banners/validation.md` com o
veredito novo.
**Where**: `.specs/features/staleness-detection-and-banners/validation.md`
**Depends on**: T8, T9, T10, T11
**Reuses**: o sensor anterior (6 alvos, worktree isolado, `git stash` nunca
usado) · o padrão de relatório do `validation.md` existente (veredito, tabela
por AC, tabela do sensor, Requirement Traceability Update) · o fallback
standalone do `tlc-spec-driven` para a análise ancorada na spec
**Requirement**: FSC-16, FSC-27, FSC-28
**Gap**: ciclo fix → re-verificar

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] O Verifier é um agente **distinto** do autor de T1..T11 (autor ≠ verificador)
- [ ] O sensor re-injeta mutações nos quatro gap sites e **cada uma** derruba a suíte
- [ ] As 6 mutações do sensor original (M1..M6) constam no relatório: **6/6 mortas**, mais as novas dos quatro gap sites
- [ ] As 9 ACs da feature `staleness-detection-and-banners` têm evidência `file:line` no relatório
- [ ] O diff da correção **não toca** a ADR-028 (`git diff --stat` do range não lista `docs/adr/028-...`)
- [ ] O veredito do relatório é **PASS** e `validate_state.py staleness-detection-and-banners` sai **exit 0**
- [ ] A árvore real está intacta no fim (baseline e final idênticos, `git status --porcelain` limpo) — **sem `git stash`**

**Tests**: integration
**Gate**: full

**Commit**: `docs(specs): re-verificacao da feature staleness-detection-and-banners`

---

### T13: A ISSUE-043 fecha e o estado do projeto é atualizado

**What**: a entrada `## ISSUE-043` em `.specs/ISSUES.md` sai de **open** e passa
a registrar o fechamento com a evidência da re-verificação (o veredito PASS e o
`file:line` do gancho); e `.specs/STATE.md` recebe o snapshot da entrega (a
decisão do literal canônico e o estado das features).
**Where**: `.specs/ISSUES.md` + `.specs/STATE.md`
**Depends on**: T12
**Reuses**: o formato de fechamento das issues anteriores do arquivo (severidade,
status, achado, referência) · a seção de Handoff do `STATE.md`
**Requirement**: FSC-26
**Gap**: processo

**Tools**:
- MCP: NONE
- Skill: `tlc-spec-driven`

**Done when**:

- [ ] A ISSUE-043 não está mais marcada como `open` e cita a evidência da re-verificação
- [ ] O `STATE.md` registra a decisão do literal canônico e o snapshot de handoff
- [ ] Nenhuma outra issue é alterada

**Tests**: none
**Gate**: build

**Commit**: `chore(specs): fecha a ISSUE-043 e atualiza o STATE`

---

## Phase Execution Map

```
Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5

Phase 1:  T1
Phase 2:  T2
Phase 3:  T3 · T4 · T5 · T6 · T7
Phase 4:  T8
Phase 5:  T9 · T10 · T11 · T12 → T13
```

Execution is strictly sequential - there is no intra-phase parallelism. A single agent (or batch worker) works one task at a time, in order.

**Contagem de tarefas**: 13 — cabem em **dois lotes** (~7 por worker: fases 1–3 = 7 tasks, fases 4–5 = 6 tasks). A execução **oferece** sub-agentes de lote ao dono antes de despachar. A T12 despacha o **Verifier** independente, que é parte do fluxo do `tlc-spec-driven` (nunca opcional, nunca perguntado).

---

## Diagram-Definition Cross-Check

| Task | Depends On (task body) | Diagram Shows | Status |
| ---- | ---------------------- | ------------- | ------ |
| T1 | None | (início da fase 1) | ✅ Match |
| T2 | T1 | `T1 → T2` | ✅ Match |
| T3 | T2 | `T2 → T3` | ✅ Match |
| T4 | T3 | `T3 → T4` | ✅ Match |
| T5 | T3 | `T3 → T5` | ✅ Match |
| T6 | T3 | `T3 → T6` | ✅ Match |
| T7 | T3 | `T3 → T7` | ✅ Match |
| T8 | T2 | `T2 → T8` | ✅ Match |
| T9 | T2 | `T2 → T9` | ✅ Match |
| T10 | T8 | `T8 → T10` | ✅ Match |
| T11 | T4, T5, T6, T7 | `T4 → T11` · `T5 → T11` · `T6 → T11` · `T7 → T11` | ✅ Match |
| T12 | T8, T9, T10, T11 | `T8 → T12` · `T9 → T12` · `T10 → T12` · `T11 → T12` | ✅ Match |
| T13 | T12 | `T12 → T13` | ✅ Match |

Nenhuma dependência aponta para uma fase posterior: T8 (fase 4) depende de T2 (fase 2), T11 (fase 5) depende de T4–T7 (fase 3) e T12 (fase 5) depende de T8 (fase 4), T9, T10 e T11.

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | Motor de staleness (domínio) | unit | unit | ✅ OK |
| T2 | Motor de staleness (domínio) + banner MCP | unit | unit | ✅ OK |
| T3 | Gancho do aviso na CLI (helper) | integration | unit | ✅ OK — o helper é testado em unidade (dentro da camada `cmd/mem`); a cobertura de integração dos 4 comandos vem em T4–T7 |
| T4 | Gancho do aviso na CLI (`mem search`) | integration | integration | ✅ OK |
| T5 | Gancho do aviso na CLI (`mem path`) | integration | integration | ✅ OK |
| T6 | Gancho do aviso na CLI (`mem inspect`) | integration | integration | ✅ OK |
| T7 | Gancho do aviso na CLI (`mem impact`) | integration | integration | ✅ OK |
| T8 | Listagem e literais do `mem status` | integration | integration | ✅ OK |
| T9 | Documentação publicada | none | none | ✅ OK |
| T10 | Documentação publicada | none | none | ✅ OK |
| T11 | Artefatos de spec (bookkeeping) | none | none | ✅ OK |
| T12 | Re-verificação independente + sensor | integration | integration | ✅ OK |
| T13 | Artefatos de spec (bookkeeping) | none | none | ✅ OK |

Nenhuma task que toca uma camada com teste exigido fica com `Tests: none`.

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: relatório com 3 categorias | 1 arquivo de produção + o teste co-localizado | ✅ Granular — o teste é **co-localizado por regra do skill** (o gate não pode quebrar entre commits); a task é **um** entregável: a separação das categorias |
| T2: literal canônico | 1 arquivo de produção + 2 arquivos de teste co-localizados | ✅ Granular — idem: sem o `internal/mcp/staleness_handlers_test.go` no mesmo commit, a suíte fica **vermelha** entre T2 e a task seguinte |
| T3: helper do aviso | 1 arquivo novo + o teste co-localizado | ✅ Granular |
| T4: `mem search` | 1 arquivo de produção + o teste co-localizado | ✅ Granular |
| T5: `mem path` | 1 arquivo de produção + o teste co-localizado | ✅ Granular |
| T6: `mem inspect` | 1 arquivo de produção + o teste co-localizado | ✅ Granular |
| T7: `mem impact` | 1 arquivo de produção + o teste co-localizado | ✅ Granular |
| T8: `mem status` | 1 arquivo de produção + o teste co-localizado | ✅ Granular |
| T9: guia do agente | 1 arquivo | ✅ Granular |
| T10: guia da CLI | 1 arquivo | ✅ Granular |
| T11: `tasks.md` de origem | 1 arquivo | ✅ Granular |
| T12: re-verificação | 1 relatório de verificação | ✅ Granular |
| T13: fecho da issue + estado | 2 arquivos de bookkeeping | ✅ Granular — o fecho de issue e o snapshot de estado são **um** entregável de bookkeeping (padrão do repo) |
