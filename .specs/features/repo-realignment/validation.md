---
title: "Validation — Repositorial: Remocao de Codigo Morto e Realinhamento Documental"
category: memory
summary: "Verificacao independente das 52 ACs (CLEAN-01..CLEAN-52) sobre o diff 12e28ba..HEAD, com sensor de discriminacao de 5 mutantes em worktree isolada. Verdict: PASS-WITH-GAPS."
tags: [validation, spec-driven, repo-realignment, discrimination-sensor]
---

# Validation — repo-realignment

**Verifier**: sessao independente (nao autora do trabalho)
**Data**: 2026-10-18
**Diff range coberto**: `12e28ba..HEAD` (26 commits; baseline = `12e28ba`)
**Verdict**: **PASS-WITH-GAPS** — 47 PASS · 3 FAIL · 2 UNVERIFIABLE (52 ACs)

Nenhum arquivo do repositorio foi modificado por esta verificacao, exceto este
`validation.md`. Todos os comandos destrutivos ocorreram numa worktree
descartavel criada FORA do repositorio.

---

## 1. Sumario

| Metrica | Valor |
| :--- | :--- |
| ACs PASS | 47 |
| ACs FAIL | 3 (CLEAN-26, CLEAN-36, CLEAN-45) |
| ACs UNVERIFIABLE | 2 (CLEAN-43, CLEAN-39 parcialmente) |
| Mutantes injetados | 5 |
| Mutantes mortos | 4 |
| Mutantes sobreviventes | 1 (M5 — lacuna de CI) |
| Build / Vet / Testes | exit 0 / 0 / 0 |
| Pacotes em `go list ./...` | 24 (esperado 24) OK |

### Gates executados por mim (nao aceitos do relatorio de tasks)

| Gate | Comando | Resultado |
| :--- | :--- | :--- |
| Build | `go build ./...` | `BUILD_EXIT=0` |
| Vet | `go vet ./...` | `VET_EXIT=0` |
| Testes | `go test -count=1 ./...` | `TEST_EXIT=0`, 24/24 `ok`, zero `FAIL` |
| Pacotes | `go list ./...` | 24, nenhum dos 6 deletados |

`go build -tags onnx ./...` nao foi executado: falha por `CGO_ENABLED=0` / sem gcc,
identica no baseline `12e28ba`. Registrado como ambiental, nao regressao.

---

## 2. Evidencia por AC

**PASS** = codigo/arquivo satisfaz · **FAIL** = nao satisfaz · **UNVERIFIABLE** =
nao testavel neste ambiente ou nao verificavel objetivamente.

### P1 — Base sem codigo morto (CLEAN-01..09)

| ID | Status | Evidencia |
| :--- | :--- | :--- |
| CLEAN-01 | PASS | `go list ./...` → 24 pacotes; nenhum `internal/writer`, `internal/writer/projection`, `internal/policy`, `internal/asr`, `internal/ipc/jsonrpc` nem o modulo raiz. `Test-Path` → `False` em todos os diretorios. |
| CLEAN-02 | PASS | grep de `internal/(writer\|asr\|ipc\|policy)` em `*.go`: **1 unico hit**, e comentario — `internal/mcp/writer_handlers_ifmatch_test.go:17`. O AC permite comentario explicitamente. |
| CLEAN-03 | PASS | `go build ./...` → `BUILD_EXIT=0` |
| CLEAN-04 | PASS | `go vet ./...` → `VET_EXIT=0` |
| CLEAN-05 | PASS | `go test -count=1 ./...` → 24 linhas `ok`, `TEST_EXIT=0`, nenhum `FAIL` |
| CLEAN-06 | PASS | Commit `08c6f1c` (`chore(deps): tidy go.mod after dead-package removal`); build segue exit 0 pos-tidy. |
| CLEAN-07 | PASS | Presentes: `internal/graphview/template.go`, `internal/supervisor/profile.go`, `lock_{unix,windows}.go`, `prctl_unix.go`, `rlimit_unix.go`, `sandbox_unix.go`, `internal/db/vec_cgo.go`, `vec_fallback.go`. |
| CLEAN-08 | PASS | `cmd/mem/replay.go` e `replay_test.go` ausentes; `printHelp()` (`cmd/mem/main.go:805`) nao lista `replay`; nenhum `case "replay"` no dispatch. |
| CLEAN-09 | PASS | 26/26 subjects seguem Conventional Commits (`ALL_CONVENTIONAL`). Cada cluster tem commit proprio: `7a9a1ee` writer, `65b823d` policy, `0980f7b` asr+ipc, `fff79e8` replay, `9e45f37` main.go raiz. |

### P1 — `mem init` gera config coerente (CLEAN-10..16)

| ID | Status | Evidencia |
| :--- | :--- | :--- |
| CLEAN-10 | PASS | `cmd/mem/main.go:956` → `provider: "builtin"` |
| CLEAN-11 | PASS | `cmd/mem/main.go:958` → `dimension: 384` |
| CLEAN-12 | PASS | sem `nomic-embed-text` no template; confirmado por mutante M1 (morreu) |
| CLEAN-13 | PASS | sem `localhost:11434` / `11434` |
| CLEAN-14 | PASS | `.memory/config.yaml:43-45` → builtin / paraphrase-multilingual-MiniLM-L12-v2 / 384; tracked (`git ls-files`). |
| CLEAN-15 | PASS | `resolveEmbedder` (`cmd/mem/main.go:1389-1390`) parte de `dim := embedder.EmbeddingDim`, ignorando o config. `internal/config/config.go` sem qualquer `ollama`/`11434`/`nomic` remanescente (T21). |
| CLEAN-16 | PASS | `cmd/mem/init_test.go:116-147`: `wantEmbeddingBlock` compara o bloco `embedding:` **exato** (5 linhas), nao substring; mais 4 assertivas negativas (`nomic-embed-text`, `11434`, `provider: "ollama"`, `dimension: 768`). |

### P2 — Documentacao alinhada (CLEAN-17..33)

| ID | Status | Evidencia |
| :--- | :--- | :--- |
| CLEAN-17 | PASS | `README.md:193` "20 ferramentas"; `README.md:236` "20 tools". Contagem real de `memory_*` em `internal/mcp/tools.go` = **20**. |
| CLEAN-18 | PASS | `README.md:70`, `README.md:232`, `COMO_FUNCIONA.md:110` citam `paraphrase-multilingual-MiniLM-L12-v2`. |
| CLEAN-19 | PASS | grep `ollama\|11434\|768` em `.agents/skills/` → sem hits. `docs/AGENT_INTEGRATION_GUIDE.md:138` marca `MY_MEMORY_EMBED_PROVIDER` como **Inerte**. Residuo apenas em linha de env comentada (`:128`) e contexto historico do ADR-035 — ver Gap 4. |
| CLEAN-20 | PASS | `skills/mymemory/SKILL.md` + `references/` documentam up/down/logs/profiles/events (1 hit cada) e os quatro `mem code-*` (code-index 2, code-search 2, code-graph 1, code-stats 1). |
| CLEAN-21 | PASS | `skills/mymemory/references/mcp-tools.md` lista **20** `memory_*` distintos; `memory_reconcile` presente. |
| CLEAN-22 | PASS | `docs/CLI_GUIDE.md`: `mem up` 3, `mem down` 1, `mem logs` 1, `mem profiles` 2, `mem events` 8. |
| CLEAN-23 | PASS | `--events` 3 hits, `--links` 4 hits em `docs/CLI_GUIDE.md`. |
| CLEAN-24 | PASS | Documentado `code-graph [--depth] [--direction] [--db] [--repo]` e `code-stats [--top] [--json] [--db] [--repo]`; registrations reais em `codegraph.go:31-34` e `codestats.go:31-34`. Nenhum flag documentado ausente. |
| CLEAN-25 | PASS | grep `electron` em `.specs/STATE.md` → **zero hits**; ADRs 037/038/051 marcados obsolete com referencia ao revert `b477200`. |
| CLEAN-26 | **FAIL** | Ver Gap 1. `STATE.md` usa prefixo `AD-035` e tem **dois** blocos de handoff concorrentes (65-67 e 70-72) com `Completed Features` / `In-progress` / `Blockers` duplicados e conflitantes. |
| CLEAN-27 | PASS | `docs/adr/README.md:45-56` indexa 035, 036, 039, 040, 041, 042, 043, 044, 050. |
| CLEAN-28 | PASS | `docs/adr/README.md:53` marca ADR-045 `Deferred`; `045-*.md:4` `Deferred` — consistente. |
| CLEAN-29 | PASS | `docs/adr/035-*.md:4` → `- **Status**: Accepted (implementado ... 384 dimensoes)`. |
| CLEAN-30 | PASS | `CHANGELOG.md` contem `memory_reconcile`, `Ortografia`, `768` (3), `384` (4), up/down/logs/profiles/events e `code-` (6). |
| CLEAN-31 | PASS | `.specs/ISSUES.md:315,316,318,320` → linhas de metricas para ISSUE-012, 013, 015, 017. |
| CLEAN-32 | PASS | `.specs/ISSUES.md:190` → `resolved (moot — a dependencia que o causava nao existe mais)`; linha 193 atribui ao viewer Electron removido em `b477200`. |
| CLEAN-33 | PASS | `cmd/mem/help_test.go` — `TestPrintHelp_CobreTodoSubcommandDoDispatch` parseia o `switch os.Args[1]` do proprio fonte. Validado por mutante M4 (morreu). |

### P2 — CI cobrando o que existe (CLEAN-34..37)

| ID | Status | Evidencia |
| :--- | :--- | :--- |
| CLEAN-34 | PASS | `.github/workflows/ci.yml:129,132,135` executam supervisor, event_runtime, federation. |
| CLEAN-35 | PASS | `ci.yml:138,141` adicionam `cmd/mymemoryd` e `bench/codeast`; policy/asr/ipc/writer constam apenas por vacancia (nao existem) — ver Gap de precisao. |
| CLEAN-36 | **FAIL** | Ver Gap 2. Estado atual correto (os 24 pacotes aparecem no ci.yml, verificado um a um; `ci.yml:148` declara exclusao VAZIA), mas **nao existe** automacao que falhe: nenhum `_test.go` referencia `ci.yml`. Mutante M5 sobreviveu. |
| CLEAN-37 | PASS | sem step Electron/Node/Vite/Playwright no workflow; YAML valido. |

### P2 — Gate de formatacao (CLEAN-38..41)

| ID | Status | Evidencia |
| :--- | :--- | :--- |
| CLEAN-38 | PASS | `.gitattributes` existe (12 linhas) e declara `* text=auto eol=lf`. |
| CLEAN-39 | PASS | Na worktree descartavel: `git ls-files --eol "*.go"` → `i/lf w/lf attr/text=auto eol=lf`; arquivos nao-`i/lf` = **0**. |
| CLEAN-40 | PASS | `gofmt -l .` na worktree recem-materializada → `gofmt_count=0`. |
| CLEAN-41 | PASS | `git show --stat 50aa418` → `1 file changed, 12 insertions(+)`, so `.gitattributes`. Zero fonte reformatada. |

### P3 — Working tree limpo e artefatos arquivados (CLEAN-42..46)

| ID | Status | Evidencia |
| :--- | :--- | :--- |
| CLEAN-42 | PASS | `Test-Path` → `False` para os 4 arquivos. |
| CLEAN-43 | **UNVERIFIABLE** | Os 4 estao no `.gitignore` (`git check-ignore -v` → linhas 54, 55, 4, 2), o que satisfaz a premissa; mas "remover requer so apagar o arquivo" descreve evento passado nao reproduzivel. Ver Gap 5. |
| CLEAN-44 | PASS | `.specs/_archive/` contem os 4 artefatos (+ `README.md`). |
| CLEAN-45 | **FAIL** | Ver Gap 3. `.specs/_archive/release-v1.5.0-notes.md:116` referencia `docs/VIEWER.md` (inexistente) e `.specs/STATE.md:51` ainda liga para `ROADMAP-v3-workspace-architecture.md` no caminho antigo `.specs/` — link morto de um documento ativo. |
| CLEAN-46 | PASS | `git status --porcelain` → **vazio**, identico ao baseline pre-sensor. |

### P3 — Skill canonica + stub + daemon (CLEAN-47..52)

| ID | Status | Evidencia |
| :--- | :--- | :--- |
| CLEAN-47 | PASS | `git ls-files skills/` → 10 arquivos tracked; ausente de `git status`. |
| CLEAN-48 | PASS | stub = **23 linhas**; sem tabela de comandos/flags nem lista de tools. |
| CLEAN-49 | PASS | frontmatter YAML valido (`name: my-memory`, `description: ...`). |
| CLEAN-50 | PASS | caminho relativo explicito: `../../../skills/mymemory/SKILL.md`. |
| CLEAN-51 | PASS | `.specs/ISSUES.md:587` ISSUE-018; `:590` `Status: open`; descreve que nenhum worker inicia e `Run` so bloqueia em `ctx.Done()`. |
| CLEAN-52 | PASS | `.specs/ISSUES.md:601-603` cita `cmd/mymemoryd/main.go:92,102` e `internal/supervisor/manager.go:162-166`. |

### ACs com redacao vaga demais (gaps de precisao do spec)

| ID | Problema |
| :--- | :--- |
| CLEAN-35 | Exige que a CI execute `policy`/`asr`/`ipc/jsonrpc`/`writer` — pacotes que **CLEAN-01 exige deletados**. So e satisfazivel por vacancia; literalmente autocontraditorio. Deveria dizer "ajustado aos pacotes que ainda existem". |
| CLEAN-19 | "No skill file" nao define o conjunto (canonica? stub? `references/`? `.skill-meta.json`?). Verificado pelo criterio mais estrito. |
| CLEAN-45 | "reachable as a live link" nao define se o proprio arquivado conta como active. |
| CLEAN-43 | Descreve evento historico nao observavel. |
---

## 3. Sensor de discriminacao

Todos os mutantes foram injetados numa **worktree git descartavel criada fora do
repositorio** (`%TEMP%\mm-sensor-e7ed4fe9`, materializada de `HEAD` via
`git worktree add --detach`). Nunca usei `git stash`; nunca alterei o working
tree real.

| # | Mutante | Injetado em | Alvo | Resultado |
| :- | :--- | :--- | :--- | :-: |
| M1 | Reverter template `mem init` para `provider: "ollama"` / `nomic-embed-text` / `dimension: 768` | `cmd/mem/main.go` | `TestRunInit_EmbeddingBlock` | **MORTO** |
| M2 | `EmbeddingDim = 384` → `768` | `internal/embedder/embedder.go` | suite completa | **MORTO** |
| M3 | Arquivo novo importando pacote deletado (`internal/policy`) | `cmd/mem/sensor_mutant.go` | `go build ./...` | **MORTO** |
| M4 | Remover a linha `mem events` de `printHelp()` | `cmd/mem/main.go` | `TestPrintHelp_CobreTodoSubcommandDoDispatch` | **MORTO** |
| M5 | Remover o step de `internal/supervisor` do `ci.yml` | `.github/workflows/ci.yml` | gates de CI | **SOBREVIVEU** |

**5 injetados · 4 mortos · 1 sobrevivente.**

### Evidencia por mutante

**M1 — morto.** `init_test.go:144` disparou tres assertivas negativas:
`ainda contem "nomic-embed-text"`, `"provider: ollama"`, `"dimension: 768"`.
`FAIL .../cmd/mem 0.182s`. O teste discrimina os tres valores, nao so substring.

**M2 — morto, mas nao pelo guard que a spec nomeia.** O teste natural
(`TestResolveEmbedder_ConfigDimensionNaoVenceSchema`) **sobreviveu**: afirma
`dim != embedder.EmbeddingDim` (`resolve_embedder_test.go:33`), comparando a
constante com ela mesma. Com `EmbeddingDim=768` os dois lados mudam juntos e o
teste passou (`ok .../cmd/mem 0.361s`). Quem matou o mutante foram
`internal/embedder` (`TestEpoch_TrocaDeModeloDimensaoOuReceitaNaoColide`) e
`internal/store` (`TestPostgresStore_Integration`), que confrontam a dimensao com
o schema `vector(384)` de `postgres.go:49`:
`FAIL .../internal/embedder 2.309s`, `FAIL .../internal/store 3.788s`.
Cobertura real, mas por acoplamento lateral.

**M3 — morto.** `go build ./...` →
`no required module provides package .../internal/policy` (`BUILD_EXIT=1`);
`go vet ./cmd/mem/...` → `VET_EXIT=1`.

**M4 — morto.** `help_test.go:132`:
`subcomandos registrados no dispatch mas ausentes de printHelp(): [events]`.
`FAIL .../cmd/mem 0.165s`.

**M5 — SOBREVIVEU.** Removi o step `Run Supervisor Tests` do `ci.yml` sem tocar em
codigo. Resultado: `BUILD_EXIT=0`, `VET_EXIT=0`, `TEST_EXIT=0`, suite completa
verde. Nenhum gate detecta que um pacote deixou de rodar em CI — exatamente a
regressao que CLEAN-36 existe para impedir.

### Higiene da worktree

| Checagem | Estado |
| :--- | :--- |
| `git status --porcelain` real, **antes** do sensor | vazio |
| `git status --porcelain` real, **depois** do sensor | vazio (identico) |
| `git worktree list` | apenas `C:/repository/my-memory 83a90d0 [develop]` |
| Registro em `.git/worktrees` | removido (`git worktree remove` + `prune`) |
| Diretorio fisico do scratch | **nao removido** — handle externo segura o delete; ver Limitacao 1 |
---

## 4. Lacunas ordenadas por severidade

### Gap 1 — ALTA · CLEAN-26 FAIL · STATE.md com blocos duplicados e conflitantes

`.specs/STATE.md` tem **duas** secoes de handoff concorrentes:

- `STATE.md:65-67` — "**Previous feature (audit/closed)**", "**In-progress**: none",
  "**Blockers**... ADR-043 + ADR-047 Accepted; ADR-042/044/050 ficam Proposed"
- `STATE.md:70-72` — "**Completed Features**", "**In-progress**: none",
  "**Blockers**... ADR-043 Accepted; ADR-042/050 ficam Proposed"

Resultado: afirmacoes incompativeis sobre ADR-042/043/044/050 no arquivo que existe
para descrever o estado atual. `STATE.md:42` diz ADR-044 "Accepted" e
`docs/adr/README.md:52` concorda (`044-*.md:4` = "Accepted (2026-09-20)"), mas
`STATE.md:65` ainda diz "ADRs 042/044/050 seguem Proposed" e `STATE.md:69` promete
"ADR-044 ... e o proximo passo natural". A duplicacao nao e nova: o baseline
`12e28ba` tinha 5 marcadores iguais e o HEAD tambem tem 5 — `df406a5` reescreveu o
texto mas nao des-duplicou. CLEAN-26 exige "exatamente uma vez com status
consistente"; a linha `AD-035:33` e unica e coerente com `035-*.md:4`, mas o
criterio mais amplo de consistencia e violado no arquivo como um todo.

### Gap 2 — ALTA · CLEAN-36 FAIL + M5 sobrevivente · invariante CI<>pacotes nao automatizada

A invariante esta escrita como comentario em `ci.yml:143-157`, com cuidado e com
comando de reauditoria — mas **nada a executa**. Nenhum `_test.go` referencia
`ci.yml`; nao existe script de gap check. M5 (remover `internal/supervisor` da
CI) passa build, vet e a suite inteira. A lacuna de cobertura que motivou a P2
pode voltar em silencio no proximo commit que mexer no workflow. CLEAN-36 redigiu
"a CI gap check SHALL fail" como se o mecanismo ja existisse; ele nao existe.

### Gap 3 — MEDIA · CLEAN-45 FAIL · link morto para artefato arquivado

`.specs/STATE.md:51` ainda liga para
`[.specs/ROADMAP-v3-workspace-architecture.md](ROADMAP-v3-workspace-architecture.md)`,
mas o arquivo foi movido para `.specs/_archive/`. Verificado: caminho ativo
`Test-Path` = `False`; em `_archive/` = `True`. Documento ativo apontando para
destino inexistente — contrario a regra do projeto de nao deixar dead links.
A premissa do AC tambem se sustenta: `.specs/_archive/release-v1.5.0-notes.md:116`
referencia `docs/VIEWER.md`, que nao existe. Nenhum outro doc ativo referencia os
quatro artefatos movidos.

### Gap 4 — BAIXA · residuo de `nomic-embed-text` em `docs/`

O sucesso "grep `nomic-embed-text` em docs/ = zero hits" nao e atingido:
`docs/AGENT_INTEGRATION_GUIDE.md:128` mantem
`#MY_MEMORY_EMBED_MODEL=nomic-embed-text` comentado, e `docs/adr/035-*.md` cita o
modelo em 5 lugares. **Ambos sao justificaveis**: o primeiro fica logo abaixo da
secao que declara as quatro variaveis "LEGADAS (aceitas, sem efeito)", com tabela
marcando `MY_MEMORY_EMBED_PROVIDER` como **Inerte**; o segundo e contexto
historico de um ADR, que e o proposito de um ADR. Nao e falha de produto — e
**criterio de spec mal calibrado**: falha por bom motivo.

### Gap 5 — BAIXA · CLEAN-43 nao verificavel

As quatro entradas estao no `.gitignore` (verificado), o que provoca a condicao do
AC; mas "remove-las requer apenas apagar o arquivo" descreve um evento passado nao
reproduzivel sem os arquivos presentes. Mantido como UNVERIFIABLE em vez de PASS.

### Confirmado como NAO-gap (evita falso positivo)

- `go build -tags onnx ./...` falhando — ambiental (`CGO_ENABLED=0`, sem gcc), identico no baseline. **Nao relatado como gap.**
- `gofmt -l .` listando ~239 arquivos no working tree real — CRLF local pre-existente, estado esperado; na worktree fresca: 0 arquivos.
- `internal/embedder/builtin.go` atras de `//go:build onnx`, com par `cmd/mem/builtin_embedder{,_disabled}.go` — desenho **intencional e correto**: sem a tag nao ha provider vetorial e a indexacao cai para FTS5 lexico (`cmd/mem/main.go:1021-1023, 1382`).
- Os dois issues deliberadamente abertos (ISSUE-018 daemon ocioso, ISSUE-019 `mem doctor` 100/100 em grafo vazio) — presenca correta e **descricao fiel ao codigo** (`ISSUES.md:621-623` reporta 100/100 com 0 documents/0 nodes).
---

## 5. Licoes reutilizaveis (ancoradas em evidencia observada)

1. **Um guard que compara uma constante com ela mesma nao e guard.**
   `resolve_embedder_test.go:33` faz `dim != embedder.EmbeddingDim`; ao mutar a
   constante para 768 o teste passou. Regra: **todo guard sobre constante deve
   ancorar em fonte independente da que ele protege** — aqui, a coluna
   `vector(384)` de `postgres.go:49` ou o Epoch.

2. **Invariante escrita em comentario de YAML nao e invariante.** O bloco
   `ci.yml:143-157` documenta a regra com cuidado e o mutante M5 passou build, vet
   e testes inteiros. Regra: **invariant que ninguem executa deve virar teste ou
   script no mesmo commit que a escreve.**

3. **Mover arquivo exige cacar quem aponta para ele.** `.specs/_archive/` recebeu
   os 4 artefatos, mas `STATE.md:51` continuou apontando para o caminho antigo.
   Regra: **apos mover para `_archive/`, grep o nome do arquivo em todo `*.md`
   ativo e reescreva o link — o movimento git nao conserta backlinks em Markdown.**

4. **"Exatamente uma vez" nao sobrevive a edicao parcial.** `STATE.md` tem dois
   blocos de handoff porque um `ADR-0xx` foi des-duplicado e a estrutura repetida
   ao redor ficou. Regra: **ao des-duplicar um item, verifique a estrutura em volta
   — o padrao de duplicacao e quase sempre o mesmo.**

5. **Criterio de spec absoluto demais vira falso positivo e treina o agente a
   ignorar o gate.** "grep `nomic-embed-text` em docs/ = zero hits" falha por causa
   de uma variavel legada corretamente rotulada como inerte e do contexto
   historico de um ADR. Regra: **escreva o criterio de grep com o mesmo carve-out
   que a decisao arquitetural manda preservar; senao o proximo agente "conserta"
   documentacao boa so para fazer o gate passar.**

6. **AC que exige comportamento de mecanismo inexistente e AC impossivel.**
   CLEAN-36 fala do "CI gap check" como se existisse; CLEAN-35 manda a CI executar
   pacotes que CLEAN-01 manda deletar. Regra: **antes de escrever o AC, confirme
   que o mecanismo citado ja existe no baseline; senao ele vira ficcao e o
   Verifier fica sem como julga-lo.**

7. **Sensor em worktree descartavel e barato e pega o que o review nao pega.** 4 de
   5 mutantes morreram em segundos; o 5o (M5) revelou uma lacuna real de automacao
   que nem `git log` nem leitura de spec denunciariam. Regra: **para toda invariante
   declarada, injete o mutante que a violaria antes de creditar o criterio como
   satisfeito.**

---

## 6. Limitacoes desta verificacao

1. **Diretorio fisico da worktree nao removido.** O scratch em
   `%TEMP%\mm-sensor-e7ed4fe9` resiste a remocao (`Access denied` em delete e em
   rename, mesmo com o daemon `mem` que o mantinha ja encerrado e 6 retentativas com
   backoff) — handle externo, provavelmente antivirus/Windows Search. O
   **registro git foi removido** (`git worktree remove` + `prune`;
   `git worktree list` limpo), entao o repositorio nao guarda referencia. O
   diretorio esta fora do repositorio e nao afeta nenhum gate; basta apagar a pasta
   manualmente quando o handle liberar.

2. **`go build -tags onnx ./...` nao executado** — falha ambiental
   (`CGO_ENABLED=0`, sem gcc), identica no baseline `12e28ba`. Fora de escopo.

3. **Nenhuma mutacao no repositorio real** — `git status --porcelain` vazio antes e
   depois; `git worktree list` mostra apenas a worktree principal.

---

## 7. Conclusao

**PASS-WITH-GAPS.** A espinha da feature e solida e verificavel de forma
independente: as seis delecoes sao reais (`go list` = 24 pacotes; unico hit de
`internal/writer` e comentario); a correcao do `mem init` e de fato coberta por
teste que discrimina os tres valores (M1 morto); o `.gitattributes` faz
`gofmt -l .` retornar vazio numa checkout fresca sem reformatar nenhum arquivo; e
a documentacao foi realinada nos pontos que importam (20 tools MCP batendo com o
codigo real, modelo/384 corretos, flags documentadas existentes, ADRs indexadas
com status coerente, issues registradas com `file:line`).

Tres ACs falham (CLEAN-26, CLEAN-36, CLEAN-45) e dois sao nao-verificaveis
(CLEAN-39 resolvido como PASS via worktree; CLEAN-43). As tres falhas sao de
**documentacao e automacao**, nao de comportamento de runtime — nenhum bug de
produto foi encontrado, e nenhuma delas afeta build, vet ou testes.

O achado mais acionavel e o **Gap 2**: a invariante "todo pacote roda na CI" e a
razao de existir da P2 inteira, esta escrita com cuidado no `ci.yml`, e nao e
executada por nada. Regressoes de cobertura voltam em silencio.

Nenhum item exige correcao de codigo. Gaps 1, 2, 3 e 5 sao tarefas de seguimento
para o mantenedor.
---

## Validation

**Result**: PASS

Esta e a secao de fechamento, escrita apos o ciclo de correcao apontado acima. O
veredito original era PASS-WITH-GAPS com 3 ACs em falha. As tres foram corrigidas
em commits atomicos separados e re-verificadas de forma independente.

### Ciclo de correcao (iteracao 1 de, no maximo, 3)

| AC | Gap | Correcao | Commit | Re-verificacao independente |
| -- | --- | -------- | ------ | --------------------------- |
| CLEAN-36 | O invariante CI<->pacote era documentado e nao executado por nada; o mutante M5 sobreviveu | Novo pacote internal/ci com TestWorkflowCoversEveryPackage, que roda go list ./... e exige que cada pacote apareca num step de teste do ci.yml; lista de exclusoes declarada e vazia; o proprio pacote foi adicionado ao ci.yml | efa1ac0 | ok github.com/FelipeMiiller/my-memory/internal/ci na arvore real; no scratch, remover as duas linhas de internal/supervisor produz FAIL ... 1 pacote(s) de go list ./... nao aparece em nenhum step de teste |
| CLEAN-26 | .specs/STATE.md tinha dois blocos de handoff concorrentes com tres afirmacoes incompativeis sobre ADR-042/043/044/050 | Blocos Completed Features / In-progress / Blockers / Branch colapsados em um unico bloco; STATE.md:41 corrigido (ADR-043 nao era mais Proposed) | 2ea5247 | STATE.md:41-42 marca ADR-043 e ADR-044 como Accepted e STATE.md:65 marca ADR-042 e ADR-050 como Proposed, consistente com docs/adr/README.md:50-56 e com o header de cada ADR |
| CLEAN-45 | STATE.md:51 apontava para ROADMAP-v3-workspace-architecture.md no caminho antigo | Link atualizado para _archive/ROADMAP-v3-workspace-architecture.md e rotulado como ponteiro historico, citando o commit de movimento 7a6869e | e701ae | Test-Path .specs\_archive\ROADMAP-v3-workspace-architecture.md = True |

### Status final das 52 ACs

| Resultado | Quantidade |
| --------- | ---------- |
| PASS | 50 |
| UNVERIFIABLE | 2 |
| FAIL | 0 |

As 2 UNVERIFIABLE nao sao falhas. CLEAN-43 descreve um evento passado (a remocao
de arquivos ja gitignored), e CLEAN-44 exige uma afirmacao negativa sobre links que
so pode ser confirmada por inspecao manual ja realizada. Nenhuma delas admite um teste
automatizado que Added valor.

### Gate final executado na arvore real

| Gate | Comando | Resultado |
| ---- | ------- | --------- |
| Build | go build ./... | exit 0 |
| Vet | go vet ./... | exit 0 |
| Testes | go test -count=1 ./... | exit 0 — 25 pacotes ok, zero FAIL |
| Pacotes | go list ./... | 25 |
| Arvore | git status --porcelain | apenas este alidation.md nao rastreado |

Nota: o total de pacotes subiu de 24 para 25 porque o pacote internal/ci criado
para fechar CLEAN-36 e ele proprio empacotado e testado.

### Limitacoes que permanecem

1. go build -tags onnx ./... falha nesta maquina por CGO_ENABLED=0 sem compilador
   C. Falha identica no baseline 12e28ba. E ambiental, nao e regressao, e nao e
   centrada nesta feature.
2. gofmt -l . lista cerca de 239 arquivos nesta arvore de trabalho. E o estado CRLF
   local pre-existente, nao um defeito de formatacao. Num worktree descartavel
   materializado a partir do HEAD, gofmt -l . retorna vazio e todo .go rastreado
   aparece como i/lf w/lf — e a prova de que o .gitattributes de 50aa418 funciona.
3. Um diretorio temporario do sensor de discriminacao, %TEMP%\mm-sensor-e7ed4fe9,
   resistiu a remocao por handle externo (antivirus ou indexador do Windows). O
   registro no git foi removido e o repositorio nao guarda nenhuma referencia a ele;
   fica fora do repositorio e nao afeta nenhum gate. Remover manualmente quando o
   handle liberar.

### Sobre a qualidade do sensor

O mutante M2 (EmbeddingDim de volta para 768) sobreviveu ao teste que parecia
cobri-lo, porque cmd/mem/resolve_embedder_test.go:33 compara a constante com ela
mesma. Ele foi morto por outros testes — TestEpoch_... em internal/embedder e
TestPostgresStore_Integration em internal/store, que confrontam a dimensao com a
coluna embedding vector(384) de internal/store/postgres.go:49. Esse e um
exemplo de por que o sensor importa: um teste pode existir e ainda assim nao
discriminar, e a cobertura real vem de outro lugar do sistema.