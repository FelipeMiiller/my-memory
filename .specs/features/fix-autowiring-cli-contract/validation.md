# Relatório de Verificação — fix-autowiring-cli-contract (re-verificação de ai-tool-auto-wiring)

**Veredito**: **PASS**

**Data**: 2026-11-16
**Spec verificada**: `.specs/features/fix-autowiring-cli-contract/spec.md` (37 ACs, ISSUE-048)
**Feature re-verificada**: `.specs/features/ai-tool-auto-wiring/` (ADR-026) — 8 ACs de origem
**Relatório anterior (FAIL)**: `.specs/features/ai-tool-auto-wiring/validation.md` (datado 2026-11-15, veredito FAIL)
**Diff range**: 18 commits da feature, `c8400fb` (ADR-026) → `3fc35e5` (T17); base `3e5eaeb`
**Diffstat escopado** (13 arquivos da feature, `3e5eaeb..HEAD`): `+1302 / -156`
**Verifier**: sub-agente **independente** (session `mvs_428714824618406d96300cd4ac2f3c3d`), distinto do autor dos commits T1–T17 (session `mvs_b9b5e7bddf4c4365b45a8d406cbf7d2e`)
**Sensor**: worktree isolado `D:\.minimax\scratch\subagents\verifier-atw2\wt` (detached HEAD `e60777e`), **sem `git stash`**
**Scratch**: `D:\.minimax\scratch\subagents\verifier-atw2\` — nada em `%TEMP%` nem na raiz do repositório

---

## 1. Resumo do veredito

A correção da ISSUE-048 **fecha os 10 gaps** e **fecha o furo de prova** que deixou a feature de
origem vermelha: os dois mutantes que sobreviveram à suíte inteira (**M7** e **M8**) agora **morrem**,
e o sensor fecha em **9/9**. As 8 ACs de origem têm evidência `file:line` nesta relatório. Os dois
gates (simples e taggeado) saem **exit 0**.

Duas pendências **não bloqueantes** ficam registradas abaixo: (a) a **promoção** deste veredito para
`.specs/features/ai-tool-auto-wiring/validation.md` — proibida ao Verifier pelo escopo de escrita
desta execução, e provada em cópia de scratch (§9); (b) um **furo de cobertura residual** na branch
de produção do filtro de alvo (`cmd/mem/install.go:125`), que o sensor expõe como **M7b**
sobrevivedor (§5).

---

## 2. Checklist dos 9 "Done when" da T18

| # | Done when (tasks.md:727-735) | Resultado | Evidência |
| :-: | :--- | :--- | :--- |
| 1 | O Verifier é um agente **distinto** do autor de T1..T17 | ✅ | §3 |
| 2 | O sensor mata **M7** e **M8** | ✅ | `cmd/mem/install_test.go:160` e `cmd/mem/install_test.go:201` (§5) |
| 3 | Os 7 mutantes antigos continuam mortos; sensor fecha em **9/9** | ✅ | §5 |
| 4 | As 8 ACs de origem têm evidência `file:line` | ✅ | §4 |
| 5 | O relatório cita **apenas** funções de teste que existem | ✅ | §7 |
| 6 | O relatório declara que os 8 nomes do plano anterior nunca existiram | ✅ | §7 |
| 7 | Veredito **PASS** e `validate_state.py ai-tool-auto-wiring` sai **exit 0** | ⚠️ **quase** | §9 |
| 8 | Árvore real intacta (baseline = final), **sem `git stash`** | ⚠️ **quase** | §8 |
| 9 | Postgres do dono **não** contatado; nenhum banco criado ou apagado | ✅ | §8 |

Os itens 7 e 8 estão marcados ⚠️ por **restrição de escopo e concorrência externa**, não por defeito
do produto — o detalhamento está em §8 e §9. Ambos fecham com **uma ação mecânica** do dono.

---

## 3. Independência do Verifier

| Fator | Evidência |
| :--- | :--- |
| Sessão do Verifier | `mvs_428714824618406d96300cd4ac2f3c3d` (branch session, `agent: Verifier`) |
| Sessão do autor de T1..T17 | `mvs_b9b5e7bddf4c4365b45a8d406cbf7d2e` (orchestrator/autor) |
| Commits do autor | 18 commits: `c8400fb`, `202dc1c`, `bfec86f`, `320817a`, `7afb227`, `25fa847`, `b0e0718`, `3fe5686`, `e4d30eb`, `23c3245`, `3a75bf6`, `70de4c6`, `3ecfc28`, `bd61444`, `16f75f9`, `3a334cd`, `d1c7049`, `3fc35e5` — **nenhum** autorado pelo Verifier |
| Commits do Verifier | **zero** — nenhuma escrita em código, spec ou `git commit`/`git push` |
| Escritas do Verifier na árvore real | **uma**: este relatório (`validation.md` desta spec) |
| Cobertura de código da feature | `internal/autowire/` + `cmd/mem/install.go` + `cmd/mem/install_test.go` — cobertos por `git diff --name-only e60777e dc7d34b` (idêntico entre o baseline do sensor e o HEAD final) |

---

## 4. Spec-anchored check — as 8 ACs da feature de origem `ai-tool-auto-wiring`

Spec de origem: `.specs/features/ai-tool-auto-wiring/spec.md` (P1.1–P1.4, P2.1–P2.2, P3.1–P3.2).
O critério só passa quando a **asserção real** aponta para o **resultado definido na spec**.

| AC | Critério (WHEN X THEN Y) | Resultado definido na spec | `file:line` + asserção | Resultado |
| :-: | :--- | :--- | :--- | :--- |
| P1.1 | `mem install` localiza o caminho de config do SO em execução | caminho exato por SO/cliente | `internal/autowire/detector_test.go:69` — `len(targets) != 5` ⇒ erro; `internal/autowire/detector_test.go:90` — `!strings.Contains(claude.ConfigPath, "AppData\Roaming\Claude\claude_desktop_config.json")` ⇒ erro; `internal/autowire/detector_test.go:129-131` — `targets[0].ConfigPath != expected` (darwin); `internal/autowire/detector_test.go:150-152` — idem (linux) | ✅ PASS |
| P1.2 | Merge insere `mcpServers["my-memory"]` sem remover os outros servidores | servidor e chaves paralelas preservados | `internal/autowire/installer_test.go:115` — `parsed["globalSetting"] != true` ⇒ erro; `:120` — `servers["other-server"]` ausente ⇒ erro; `:123` — `servers["my-memory"]` ausente ⇒ erro. Impl.: `internal/autowire/installer.go:41-49`, `:71-72` | ✅ PASS |
| P1.3 | Backup `.bak` antes de qualquer alteração | `.bak` existe com o conteúdo original | `internal/autowire/installer_test.go:103` — `report.BackupPath == ""` ⇒ erro; `:106` — `os.Stat(configFile + ".bak")`; `:299` — `report.BackupPath != backupFile` ⇒ erro; `:307` — `!bytes.Equal(backupData, original)` ⇒ erro. Impl.: `internal/autowire/installer.go:104-109`, `:143-171` | ✅ PASS |
| P1.4 | Arquivo ausente ⇒ cria diretório e JSON formatado | arquivo criado, JSON válido, `mcpServers.my-memory` presente | `internal/autowire/installer_test.go:34` — `report.Status != StatusInstalled` ⇒ erro; `:38` — arquivo lido; `:48` — `mcpServers` ausente ⇒ erro; `:53` — `my-memory` ausente ⇒ erro. Impl.: `internal/autowire/installer.go:95-101`, `:112-116` | ✅ PASS |
| P2.1 | `--dry-run` mostra o caminho de cada arquivo e o trecho JSON | ⚠️ a spec **não define** o que é "trecho JSON" (spec-precision gap de origem) | `cmd/mem/install.go:197` — imprime `Arquivo: %s`; `cmd/mem/install.go:204-206` — imprime `Conteúdo JSON:` com `indentText(report.Diff, "     ")`; `internal/autowire/installer_test.go:195` — `report.Diff == ""` ⇒ erro; `cmd/mem/install_test.go:366` — `dryRunArgs`/`extractDryRunJSON` extraem e decodificam o **conteúdo** do payload do dry-run, usado em `:306` e `:344` | ✅ PASS *(o gap de precisão da spec **não** foi fechado no texto, mas a asserção passou a mirar o **conteúdo** do JSON, não só um badge — ver §10, Achado 3)* |
| P2.2 | `--dry-run` não cria nem altera arquivo em disco | zero escrita | `internal/autowire/installer_test.go:200` — `os.Stat(configFile)` deve ser `IsNotExist`; `cmd/mem/install_test.go:98-101` — `.cursor/mcp.json` não deve existir. Impl.: `internal/autowire/installer.go:84-93` retorna antes de `:95-116` | ✅ PASS |
| P3.1 | `--target` aceita `all`, `claude`, `claude-desktop`, `cursor`, `vscode`, `windsurf` | conjunto exato de 6 valores | `internal/autowire/detector.go:256-258` — `AcceptedTargets()`; `:261-269` — `IsAcceptedTarget`; `:243-245` — `matchTarget`; `:272-277` — `targetAlias` normaliza `claude-desktop`→`claude`. Testes: `internal/autowire/detector_test.go:185` (`TestAcceptedTargets_ConjuntoExatoDocumentado`), `:199-203` (cada valor aceito), `:206-210` (valor fora rejeitado), `:213` (`TestDetectClients_ClaudeDesktopSinonimoResolveSoClaude`), `:226` (`len(byAlias) != 1` ⇒ erro), `:232-237` (Cursor/VS Code/Windsurf não resolvidos). CLI: `cmd/mem/install.go:49`, `:232-233`; `cmd/mem/install_test.go:534` (`TestInstallCLI_TargetInvalidoFalhaComExitNaoZero`), `:550` (mensagem lista os aceitos), `:553-557` (cada valor citado na mensagem) | ✅ PASS |
| P3.2 | `--workspace`/`--global` filtram o escopo | escopo global ≠ workspace | `internal/autowire/detector.go:247-249` — `matchScope`; `internal/autowire/detector_test.go:156` (`TestDetectClients_FilterScope`), `:164` (`len(wsTargets) != 2` ⇒ erro), `:175` (`len(globalTargets) != 3` ⇒ erro). CLI: `cmd/mem/install.go:53-59` (cálculo do escopo), `:231-239` (`validateInstallFlags`). Testes: `cmd/mem/install_test.go:201` (`TestInstallCLI_GlobalNaoEscreveNoDirDoWorkspace`), `:220-228` (nenhum `.cursor/mcp.json`/`.vscode/mcp.json` no `--dir`), `:231-234` (controle positivo: config global do Cursor **é** criado); `cmd/mem/install_test.go:241` (`TestInstallCLI_EscopoContraditorioFalha`), `:253-255` (`err == nil` ⇒ erro), `:256-258` (mensagem "mutuamente exclusivas"). `cmd/mem/main.go:836-840` — erro ⇒ `os.Exit(1)` | ✅ PASS |

**Status**: **8/8 ACs com evidência `file:line`** e asserção casando com o resultado definido na spec.

### 4.1 Os 37 ACs da spec de correção (AUTOWIRE-01 a AUTOWIRE-37)

Agrupados por gap. Todos com evidência `file:line` no HEAD.

| ACs | Gap | Evidência | Resultado |
| :--- | :-: | :--- | :--- |
| AUTOWIRE-01 (AC1) | 1 | `docs/CLI_GUIDE.md:628` — "**Atenção:** `mem setup` é **outro comando** — o assistente interativo de cofre central (ADR-033). O auto-wiring é `mem install`."; `docs/adr/026-...md:33` — o verbo `setup` pertence ao assistente. Grep do critério de sucesso (`ou \`mem setup\`` em `docs/`): **0 ocorrências** | ⚠️ **parcial** — os dois índices ainda anunciam o alias (Achado 1) |
| AUTOWIRE-02 (AC2) | 1 | `docs/adr/026-...md:33` — nomeia `mem install` e registra o dono do verbo `setup` (ADR-033) | ✅ PASS |
| AUTOWIRE-03 (AC3) | 2 | `cmd/mem/install_test.go:181` — `--target cursor` não pode criar `.vscode/mcp.json` | ✅ PASS |
| AUTOWIRE-04 (AC4) | 2 | `cmd/mem/install_test.go:184` — a saída não pode mencionar "VS Code" | ✅ PASS |
| AUTOWIRE-05 (AC5) | 2 | `cmd/mem/install_test.go:220-228` — com `--global`, nenhum config de workspace dentro do `--dir` | ✅ PASS |
| AUTOWIRE-06 (AC6) | 2 | `cmd/mem/install_test.go:160` e `:201` — asserção do **lado negativo** dos dois filtros | ✅ PASS |
| AUTOWIRE-07 (AC7) | 2 | `cmd/mem/install.go:235-237` (erro) + `cmd/mem/main.go:836-840` (`os.Exit(1)`); `cmd/mem/install_test.go:253-258` | ✅ PASS |
| AUTOWIRE-08 (AC8) | 3 | `cmd/mem/install.go:100-106` — Postgres ⇒ `--storage postgres`; `cmd/mem/install_test.go:307` — `argsHaveFlagValue(args, "--storage", "postgres")` | ✅ PASS |
| AUTOWIRE-09 (AC9) | 3 | `cmd/mem/install.go:107-109` — no ramo Postgres **não** há `--db`; `cmd/mem/install_test.go:310` — `argsHaveFlag(args, "--db")` ⇒ erro | ✅ PASS |
| AUTOWIRE-10 (AC10) | 3 | `cmd/mem/install.go:108` — SQLite ⇒ `--db <caminho>`; `cmd/mem/install_test.go:345-347` — `dbPath == ""` ⇒ erro; `:352` — base `memory.db` | ✅ PASS |
| AUTOWIRE-11 (AC11) | 3 | `cmd/mem/install_test.go:355` — `argsHaveFlagValue(args, "--storage", "postgres")` ⇒ erro no caso SQLite | ✅ PASS |
| AUTOWIRE-12 (AC12) | 3 | `cmd/mem/install.go:102-106` — a URL **não** entra no payload; `cmd/mem/install_test.go:313` — `strings.Contains(out, pgURL)` ⇒ erro | ✅ PASS |
| AUTOWIRE-13 (AC13) | 3 | `cmd/mem/install.go:157` — `Engine: %s`; `cmd/mem/install_test.go:298` — `strings.Contains(out, "Engine: postgres")` | ✅ PASS |
| AUTOWIRE-14 (AC14) | 3 | `cmd/mem/install_test.go:340` — `strings.Contains(out, "Engine: sqlite")` | ✅ PASS |
| AUTOWIRE-15 (AC15) | 3 | `cmd/mem/install.go:158-160` — a linha `Database:` só sai no caso SQLite; `cmd/mem/install_test.go:302` — `strings.Contains(out, "Database:")` ⇒ erro | ✅ PASS |
| AUTOWIRE-16 (AC16) | 4 | `internal/autowire/detector.go:256-258` — conjunto de exatamente 6; `internal/autowire/detector_test.go:186` (conjunto esperado) e `:189-196` (igualdade posicional) | ✅ PASS |
| AUTOWIRE-17 (AC17) | 4 | `internal/autowire/detector.go:272-277` — `targetAlias`; `internal/autowire/detector_test.go:229` — `byAlias[0].Type != ClientClaudeDesktop` ⇒ erro | ✅ PASS |
| AUTOWIRE-18 (AC18) | 4 | `internal/autowire/detector_test.go:226` (conjunto exato = 1) e `:232-237` (Cursor/VS Code/Windsurf rejeitados) | ✅ PASS |
| AUTOWIRE-19 (AC19) | 4 | `cmd/mem/install.go:232-234` — `IsAcceptedTarget` ⇒ erro; `cmd/mem/main.go:836-840` ⇒ exit ≠ 0; `cmd/mem/install_test.go:544` — `err == nil` ⇒ erro. Execução real: `exit 1` | ✅ PASS |
| AUTOWIRE-20 (AC20) | 4 | `cmd/mem/install.go:233` — a mensagem lista `strings.Join(autowire.AcceptedTargets(), ", ")`; `cmd/mem/install_test.go:550` e `:553-557` | ✅ PASS |
| AUTOWIRE-21 (AC21) | 5 | `cmd/mem/install.go:169-173` — alvo sem `Installed` é pulado antes de `InjectMCPServer`; `cmd/mem/install_test.go:474` (arquivo) e `:477` (diretório) | ✅ PASS |
| AUTOWIRE-22 (AC22) | 5 | `cmd/mem/install.go:170-171` — `[PULADO] <nome>` + `Motivo:`; `cmd/mem/install_test.go:482`, `:485`, `:488` | ✅ PASS |
| AUTOWIRE-23 (AC23) | 5 | `internal/autowire/detector.go:339-350` — `markClientPresence`; `cmd/mem/install_test.go:493-495` — controle positivo | ✅ PASS |
| AUTOWIRE-24 (AC24) | 5 | `cmd/mem/install.go:30` — `runInstallCommandWithEnv(..., env autowire.SystemEnv)`; `cmd/mem/install_test.go:22-63` — `installFakeEnv`/`newInstallFakeEnv`; `internal/autowire/detector.go:11-16` — interface `SystemEnv` | ✅ PASS |
| AUTOWIRE-25 (AC25) | 6 | `internal/autowire/installer.go:143-171` — `writeFileAtomic` (temporário no mesmo diretório + `renameFile`); `:106` — chamada; `internal/autowire/installer_test.go:229-233` — `renameFile` injetado com falha; `:251-256` — o temporário existe e está no mesmo diretório | ✅ PASS |
| AUTOWIRE-26 (AC26) | 6 | `internal/autowire/installer_test.go:307` — `!bytes.Equal(backupData, original)` ⇒ erro; `:311-320` — nenhum `.tmp-` sobra | ✅ PASS |
| AUTOWIRE-27 (AC27) | 7 | `internal/autowire/installer.go:75` — `json.MarshalIndent(cfgMap, "", "  ")`; `internal/autowire/installer_test.go:370-380` (2 espaços, sem tabulação) e `:383-392` (ordem alfabética) | ✅ PASS |
| AUTOWIRE-28 (AC28) | 7 | `internal/autowire/installer_test.go:399-415` — `zetaGlobal`, `alphaSetting` e `other-server` preservados | ✅ PASS |
| AUTOWIRE-29 (AC29) | 7 | `.specs/features/ai-tool-auto-wiring/spec.md:17` — "2 espaços … chaves de topo em **ordem alfabética** … a indentação e a ordem originais … **não** são preservadas"; `:3` — emenda à spec de correção | ✅ PASS |
| AUTOWIRE-30 (AC30) | 8 | `docs/adr/026-...md:37` — "O registro de clientes **não é declarativo**"; `:39` — as quatro constantes `ClientType` (`internal/autowire/client.go:6-11`); `:41-45` — blocos de detecção e filtros | ✅ PASS |
| AUTOWIRE-31 (AC31) | 8 | `docs/adr/026-...md:52` — Windsurf só em Global; `:54` — "O Windsurf é suportado **somente no escopo global**" | ✅ PASS |
| AUTOWIRE-32 (AC32) | 9 | `docs/AGENT_INTEGRATION_GUIDE.md:284` — linha `\| \`mem install\` \| Auto-wiring zero-touch … \|` | ✅ PASS |
| AUTOWIRE-33 (AC33) | 10 | §7 — verificação mecânica deste relatório | ✅ PASS |
| AUTOWIRE-34 (AC34) | 10 | §7 — os 8 nomes nomeados e declarados | ✅ PASS |
| AUTOWIRE-35 (AC35) | 2 | §5 — M7 e M8 **mortos**, com o teste que os mata nomeado | ✅ PASS |
| AUTOWIRE-36 (AC36) | ciclo | §3 — sessões distintas | ✅ PASS |
| AUTOWIRE-37 (AC37) | ciclo | §4 (8/8 ACs com `file:line`) + §5 (9/9) + veredito no topo deste arquivo | ✅ PASS |

**Cobertura**: 37/37 ACs com evidência `file:line`; **1 AC parcial** (AUTOWIRE-01, por resíduo documental — Achado 1).

---

## 5. Discrimination Sensor

**Scratch**: worktree isolado `D:\.minimax\scratch\subagents\verifier-atw2\wt` (detached HEAD `e60777e`), criado com
`git worktree add --detach`. O sensor rodou **inteiramente** nesse worktree; a árvore real nunca foi
mutada. `git stash list` está **vazio** (nenhum stash criado). Reset entre mutantes por
`git checkout -- .` **dentro do worktree**.

Método: cada mutante é uma **falha de comportamento** injetada por substituição textual com
ocorrência **exatamente 1** (o script aborta se a âncora não for única). O pacote coberto é o que
exercita o código mutado; para **M7** e **M8** — os dois que sobreviveram à suíte inteira — a
**suíte completa** do pacote `cmd/mem` foi rodada, para provar que *a suíte inteira* morre.

| # | Alvo (arquivo:linha no HEAD) | Mutação de comportamento | Teste que mata | Morto? |
| :-: | :--- | :--- | :--- | :--- |
| M1 | `internal/autowire/installer.go:71` | o merge deixa de preservar os outros servidores (`mcpServers = map[...]{serverName: ...}`) | `internal/autowire/installer_test.go:120` (`other-server`) e `:115` (`globalSetting`) | ✅ Morto |
| M2 | `internal/autowire/installer.go:84` | `--dry-run` deixa de bloquear a escrita (`if dryRun` → `if dryRun && false`) | `internal/autowire/installer_test.go:200` (arquivo não deve existir) | ✅ Morto |
| M3 | `internal/autowire/installer.go:104` | o backup `.bak` deixa de ser criado (`if len(originalData) > 0` → `&& false`) | `internal/autowire/installer_test.go:103` e `:106` | ✅ Morto |
| M4 | `internal/autowire/detector.go:243-245` | `matchTarget` deixa de filtrar por cliente (`return true`) | `internal/autowire/detector_test.go:226` (conjunto exato = 1) | ✅ Morto |
| M5 | `internal/autowire/detector.go:247-249` | `matchScope` deixa de respeitar o escopo (`return true`) | `internal/autowire/detector_test.go:164` e `:175` | ✅ Morto |
| M6 | `internal/autowire/detector.go:79` | o caminho do Claude Desktop no Windows deixa de ser achado (`"Claude"` → `"ClaudeWRONG"`) | `internal/autowire/detector_test.go:90` | ✅ Morto |
| **M7** | `cmd/mem/install.go:123` | **a CLI ignora a flag `--target`** (`*target` → `"all"`) | **`cmd/mem/install_test.go:160` — `TestInstallCLI_TargetRestringeAoClientePedido`** | ✅ **Morto** |
| **M8** | `cmd/mem/install.go:53-59` | **a CLI ignora `--workspace`/`--global`** (escopo sempre `ScopeAll`) | **`cmd/mem/install_test.go:201` — `TestInstallCLI_GlobalNaoEscreveNoDirDoWorkspace`** | ✅ **Morto** |
| M9 | `cmd/mem/install.go:176` | CONTROLE: a chave gravada deixa de ser `my-memory` | `cmd/mem/install_test.go:143` e `:485` | ✅ Morto |

**Resultado do sensor: 9/9 mutantes mortos.** Os 7 antigos continuam mortos; M7 e M8 — os dois que
sobreviveram à suíte inteira na verificação anterior — agora morrem.

### 5.1 Atribuição dos dois mutantes críticos (re-execução isolada)

| Mutante | `go test -run TestInstallCLI ./cmd/mem/...` | suíte **completa** `./cmd/mem/...` | teste que falha |
| :--- | :--- | :--- | :--- |
| M7 | `FAIL` (exit 1) | `FAIL github.com/FelipeMiiller/my-memory/cmd/mem 120.771s` (exit 1) | **exatamente 1**: `TestInstallCLI_TargetRestringeAoClientePedido` — os outros 9 testes de `TestInstallCLI` passam |
| M8 | `FAIL` (exit 1) | `FAIL github.com/FelipeMiiller/my-memory/cmd/mem 124.735s` (exit 1) | **exatamente 1**: `TestInstallCLI_GlobalNaoEscreveNoDirDoWorkspace` — os outros 9 testes de `TestInstallCLI` passam |

A falha é **atribuível** (um único teste, o guarda projetado para o mutante) — não é um "FAIL" difuso
de co-contenção nem um teste flakey. Os testes **não são vacuamente verdes**: cada guarda tem o seu
controle positivo (`cmd/mem/install_test.go:189-192` para M7, `:231-234` para M8).

### 5.2 Mutante extra que sobreviveu (Achado 2 — residual, **não** é o M7 original)

O M7 original (`cmd/mem/install.go:85` na versão anterior) era **uma** chamada de detecção. A correção
**dividiu** essa chamada em duas branches (`cmd/mem/install.go:123` com env injetado e
`cmd/mem/install.go:125` com o ambiente real). O mutante **M7b**, que altera **apenas** a branch de
produção (`autowire.DetectClients(*rootDir, "all", scope)`), **sobreviveu à suíte inteira**
(`ok … cmd/mem 120.132s`, exit 0).

Motivo comprovado: **todo** teste de `cmd/mem/install_test.go` chama `runInstallCommandWithEnv` com
env **não-nil** (10 chamadas em `cmd/mem/install_test.go:84, :111, :168, :212, :248, :285, :328, :462, :508, :540`),
e `autowire.DetectClients(` **não é chamada por nenhum teste** do repositório. A branch que o
usuário real exercita (`env == nil`) **não tem cobertura de filtro de alvo/escopo**.

Ver §10, Achado 2.

### 5.3 Escopo do sensor

| Dimensão | Valor |
| :--- | :--- |
| Profundidade | **lightweight ampliado** — 10 mutantes (o mínimo do skill é 3); 9 canônicos + 1 extra de controle |
| Cobertura | 3 arquivos (`cmd/mem/install.go`, `internal/autowire/detector.go`, `internal/autowire/installer.go`) — exatamente os arquivos introduzidos/modificados pela feature |
| Suíte completa | rodada para M7 e M8 (os dois que já sobreviveram); pacote coberto para os demais |
| Isolamento | worktree isolado; `git stash` **não** usado; worktree limpo ao fim (`git status --porcelain` vazio) |

---

## 6. Gate Check

| Gate | Comando | Resultado |
| :--- | :--- | :--- |
| Full (simples) | `go test -count=1 ./...` | **exit 0** — 27 pacotes `ok`, **0** falha, 1 sem arquivo de teste (`viewer/node_modules/flatted/golang/pkg/flatted`) |
| Full (taggeado) | `go test -tags "sqlite_fts5 onnx" -count=1 ./...` (com `CGO_ENABLED=1` e `C:\msys64\ucrt64\bin` no PATH) | **exit 0** — 27 pacotes `ok`, **0** falha |
| Build | `go build -tags "sqlite_fts5 onnx" -o <scratch>/mem.exe ./cmd/mem` | **exit 0** |
| Build (simples) | `go build ./...` | **exit 0** (o gate do binário provider rodou; `go build ./...` também passou no gate completo) |
| Formatação | `gofmt -l` sobre **cópias LF** dos 7 arquivos `.go` tocados (`cmd/mem/install.go`, `cmd/mem/install_test.go`, `internal/autowire/{client,detector,installer}.go`, `internal/autowire/{detector,installer}_test.go`) | **vazio** (nada fora do padrão). O working tree local mistura CRLF (`git ls-files --eol` → `i/lf w/crlf` em `detector.go`/`installer.go`) por causa de `.gitattributes` (`* text=auto eol=lf`); a medição em cópia LF é a correta e é a que a spec prescreve (tasks.md:46) |
| Testes pulados | 30 testes de integração PostgreSQL (`pgtest.New(t)` em 22 arquivos) — `internal/store` (12), `internal/db` (5), `cmd/mem` (5), `internal/codeast` (2), `internal/watcher` (1) | pulados por desenho (`internal/pgtest/pgtest.go:42`, `OptInEnvVar`) — **nenhum pertence à feature `ai-tool-auto-wiring`**, que não tem acoplamento de banco (`grep` por `pgtest|sql.Open|lib/pq|database/sql` em `internal/autowire/**` e em `cmd/mem/install.go`/`cmd/mem/install_test.go`: **0 ocorrências**) |
| Integridade da contagem | antes da feature (`03915aa`): `internal/autowire` = 0 (pacote novo), `cmd/mem` = 40 · agora: `internal/autowire` = **16** funções de teste, `cmd/mem/install_test.go` = **11** | **nenhuma deleção silenciosa** — o relatório anterior já declarava 9 + 261; o acréscimo desta correção é de 7 funções em `internal/autowire` e 8 em `cmd/mem/install_test.go` |
| "15 pacotes Go" do plano antigo | o plano de validação anterior afirmava "Todos os 15 pacotes Go continuam passando" | hoje são **27** pacotes com teste — a afirmação do plano antigo era falsa |

### 6.1 Execução real do binário (prova fora da suíte)

Binário: `mem.exe` compilado com `-tags "sqlite_fts5 onnx"`. Execuções em diretório de scratch.

| Cenário | Saída observada | Exit |
| :--- | :--- | :-: |
| `mem install --target xxxx --dry-run` | `Erro: --target inválido: "xxxx" (valores aceitos: all, claude, claude-desktop, cursor, vscode, windsurf)` | **1** |
| `mem install --workspace --global --dry-run` | `Erro: --workspace e --global são mutuamente exclusivas: escolha um único escopo` | **1** |
| `mem install --target claude-desktop --dry-run` | `Engine: postgres` · `Alvos: 1 ferramenta(s) a configurar` · `🔎 [SIMULAÇÃO] Claude Desktop (Global)` · payload com `--storage postgres`, **sem** `--db`, **sem** a URL · **sem** linha `Database:` | 0 |
| `mem install --help` | `-target string … 'all', 'claude', 'claude-desktop', 'cursor', 'vscode' ou 'windsurf' (padrão: all)` | — |

O dry-run **leu** (somente leitura) o `claude_desktop_config.json` real do usuário; o arquivo está
**intacto** (`LastWriteTime 23/08/2026 19:35:55`, tamanho 1504) e **nenhum** `.bak` foi criado ao
lado dele.

---

## 7. O relatório cita apenas testes que existem — e os 8 nomes que nunca existiram

### 7.1 Inventário real

`internal/autowire` — 16 funções: `TestDetectClients_WindowsPaths` (`detector_test.go:50`),
`TestDetectClients_DarwinPaths` (`:114`), `TestDetectClients_LinuxPaths` (`:135`),
`TestDetectClients_FilterScope` (`:156`), `TestAcceptedTargets_ConjuntoExatoDocumentado` (`:185`),
`TestDetectClients_ClaudeDesktopSinonimoResolveSoClaude` (`:213`),
`TestDetectClients_PresencaSegueMarcadorDeInstalacao` (`:252`),
`TestClientInstallMarker_PorSistemaOperacional` (`:292`), `TestInjectMCPServer_NewFile`
(`installer_test.go:13`), `TestInjectMCPServer_PreserveExistingServers` (`:63`),
`TestInjectMCPServer_Idempotent` (`:128`), `TestInjectMCPServer_DryRun` (`:170`),
`TestInstaller_BackupEhPublicadoPorRename` (`:212`),
`TestInstaller_BackupEhByteIdenticoAoConfigSubstituido` (`:274`),
`TestInstaller_FormatacaoCanonicaPreservaConteudo` (`:326`), `TestInjectMCPServer_InvalidJSON` (`:418`).

`cmd/mem/install_test.go` — 11 funções: `TestInstallCLI_DryRun` (`:77`), `TestInstallCLI_WorkspaceExecution`
(`:104`), `TestInstallCLI_TargetRestringeAoClientePedido` (`:160`),
`TestInstallCLI_GlobalNaoEscreveNoDirDoWorkspace` (`:201`), `TestInstallCLI_EscopoContraditorioFalha`
(`:241`), `TestInstallCLI_PayloadPostgresDeclaraStoragePostgres` (`:275`),
`TestInstallCLI_PayloadSQLiteCarregaDB` (`:321`), `TestInstallCLI_ClienteAusenteNaoEConfigurado` (`:454`),
`TestInstallCLI_DryRunNaoListaClienteInexistente` (`:501`),
`TestInstallCLI_TargetInvalidoFalhaComExitNaoZero` (`:534`), `TestRearrangeInstallArgs` (`:574`).

**Todo** nome de teste citado neste relatório está nessa lista — verificado mecanicamente (§7.3).

### 7.2 Os 8 nomes do plano de validação anterior que **nunca existiram**

O "Validation Plan" anterior (recuperável em `git show db02542:.specs/features/ai-tool-auto-wiring/validation.md`)
marcava **12** nomes de teste como `[x]` (feito). **4** existem hoje; **8 nunca existiram** — nem
quando o plano foi escrito, nem depois:

| # | Nome marcado como feito no plano antigo | Existe no repositório? | Onde o nome aparece hoje |
| :-: | :--- | :--- | :--- |
| 1 | `TestDetectClients_PathResolution` | **nunca existiu** | só nesta tabela (declaração de não-existência) |
| 2 | `TestDetectClients_ScopeFiltering` | **nunca existiu** | só nesta tabela |
| 3 | `TestDetectClients_TargetFiltering` | **nunca existiu** | só nesta tabela |
| 4 | `TestInjectMCPServer_MergeExisting` | **nunca existiu** | só nesta tabela |
| 5 | `TestInjectMCPServer_BackupCreation` | **nunca existiu** | só nesta tabela |
| 6 | `TestInstallCLI_WorkspaceOnly` | **nunca existiu** | só nesta tabela |
| 7 | `TestInstallCLI_TargetFlag` | **nunca existiu** | só nesta tabela |
| 8 | `TestInstallCLI_AliasSetup` | **nunca existiu** | só nesta tabela |

Evidência: `grep` por `^func (Test\w+)` em todo `**/*_test.go` do repositório → **0 ocorrências**
para os 8 nomes. O único lugar onde cada um aparece no repositório é a **declaração de não-existência**
deste relatório — o que fecha o gap 10 (AC33/AC34) sem repetir o erro do plano antigo.

O mesmo plano antigo afirmava ainda "Todos os 15 pacotes Go continuam passando" (hoje são **27**) e
"Teste real no Windows demonstrando detecção e injeção em clientes disponíveis" (não existe tal
teste; o dry-run é exercitado em diretório temporário, `cmd/mem/install_test.go:98-101`).

### 7.3 Verificação mecânica do relatório

Script: `D:\.minimax\scratch\subagents\verifier-atw2\check-report.ps1`. Ele extrai **todos** os
identificadores `Test*` deste arquivo e exige que cada um **ou** exista como `func Test…` no
repositório **ou** esteja na lista declarada de "nunca existiram" (§7.2). Resultado: **0 nomes
órfãos**.

Duas ressalvas lexicais, para que o resultado seja lido sem ambiguidade: (a) o token
`TestInstallCLI` aparece no texto como **prefixo** do padrão `-run` (e não como nome de função —
as funções se chamam `TestInstallCLI_*`); (b) `Teste`/`Testes` são palavras do texto em português,
não identificadores; (c) `TestMain` é uma função real (`cmd/mem/main_test.go:41`,
`internal/db/main_test.go:27`).

---

## 8. Isolamento, árvore real e Postgres

### 8.1 Árvore real — baseline e final

| Momento | `git status --porcelain` | Quem escreveu |
| :--- | :--- | :--- |
| Baseline (antes de tudo) | **vazio** | — |
| Após o gate simples | **vazio** | — |
| Após o gate taggeado | ` M internal/db/graph.go` · ` M internal/db/graph_identity_test.go` | **outro agente** (ADR-066: identidade canônica do nó — `git log e60777e..HEAD`: `dc7d34b`, `5bc08cc`, `c71aa82`, `335d57c`, `84aab52`) |
| Após o sensor | idêntico ao anterior | o sensor rodou **inteiramente** no worktree de scratch |
| **Final (medido)** | `?? .specs/features/fix-autowiring-cli-contract/validation.md` (este relatório) · ` M .specs/features/fix-graph-node-identity/tasks.md` · ` M internal/db/graph.go` · ` M internal/db/graph_identity_test.go` · ` M internal/store/graph_identity_test.go` · ` M internal/store/postgres.go` | o relatório é a **única** escrita do Verifier; os 5 arquivos modificados são de **outro agente** (feature `fix-graph-node-identity` / ADR-066) e **não foram revertidos** |

> O `git status --porcelain` é um **instantâneo móvel**: outro agente estava committando no repositório
> durante toda esta verificação, e o conjunto de arquivos modificados mudou ao longo do tempo (5 →
> 2, à medida que ele committava). O que é **estável** e verificável é o que importa: nenhum dos
> **14 arquivos da feature** foi alterado na árvore real, `git stash list` está vazio, e o sensor
> rodou exclusivamente no worktree de scratch.

Provas de que a mutação do sensor **não** tocou a árvore real:

- `git stash list` → **vazio** (nenhum `git stash` foi criado em momento algum);
- o sensor aplicou e revertou mutações **apenas** em `D:\.minimax\scratch\subagents\verifier-atw2\wt`
  (worktree detached em `e60777e`), que terminou **limpo** (`git status --porcelain` vazio);
- `git diff --name-only HEAD -- <os 14 arquivos da feature>` → **vazio**: nenhum arquivo da feature
  foi alterado na árvore real;
- `git diff --name-only e60777e dc7d34b -- <os 14 arquivos da feature>` → **vazio**: o código
  verificado é **byte-idêntico** ao HEAD final (os 5 commits concorrentes são de outra feature e não
  tocam estes arquivos);
- os 5 arquivos modificados no fim (`.specs/features/fix-graph-node-identity/tasks.md`,
  `internal/db/graph.go`, `internal/db/graph_identity_test.go`,
  `internal/store/graph_identity_test.go`, `internal/store/postgres.go`) são de um **escopo
  diferente** do feature e **não foram revertidos** — conforme instruído;
- o worktree de scratch do sensor foi removido no fim (`git worktree remove --force` +
  `git worktree prune`); os worktrees remanescentes em `git worktree list` pertencem a outros
  agentes.

### 8.2 Postgres do dono

| Verificação | Resultado |
| :--- | :--- |
| `MY_MEMORY_PG_URL` no ambiente da sessão | **presente** (aponta para o Neon do dono) — herdada do ambiente, não definida pelo Verifier |
| `MY_MEMORY_PG_TEST` | **unset** durante todos os gates ⇒ a integração PostgreSQL ficou **desligada** por desenho (`internal/pgtest/pgtest.go:42`, `OptedInEnvVar`) |
| Contato com o Postgres do dono | **nenhum**. O `TestMain` de `cmd/mem` (`cmd/mem/main_test.go:60-67`) e de `internal/db` (`internal/db/main_test.go:27-35`) **neutralizam** `MY_MEMORY_PG_URL`/`DATABASE_URL`/`POSTGRES_URL` **antes** de `m.Run()`; `pgtest.OptedIn()` é falso ⇒ os 30 testes de integração são **pulados**, sem abrir conexão |
| Banco criado ou apagado | **nenhum**. `pgtest.New` (único caminho de criação de banco) só roda com o opt-in ligado — e ele cria/derruba apenas bancos descartáveis `mymemory_pgtest_*`, nunca o `mymemory` do dono |
| Contato durante o `mem install --dry-run` real | **nenhum**: o comando só **resolve strings** (`cmd/mem/main.go:1438` — `resolvePostgresURL`) e **não abre store**; o `--dry-run` retorna em `internal/autowire/installer.go:84-93` antes de qualquer escrita |
| Config do usuário tocada | **nenhuma**: `C:\Users\Felipe\AppData\Roaming\Claude\claude_desktop_config.json` intacto (mtime 23/08/2026), sem `.bak` criado |

**Nota sobre o gate com Postgres**: o `Fix` §Ambiente do briefing prescreve `MY_MEMORY_PG_TEST=1` com a
URL do config global, mas **não existe Postgres local ou descartável nesta máquina** (sem serviço,
sem `docker`, sem `psql`; `127.0.0.1:5432` fechado). Rodar esse gate exigiria, portanto, criar e
derrubar um banco **no Neon do dono** — o que contradiz o critério da própria T18 ("o Postgres do
dono **não** foi contatado e nenhum banco foi criado ou apagado"). Ver §11.

---

## 9. Promoção do veredito para a feature de origem

O critério da T18 exige `validate_state.py ai-tool-auto-wiring` com **exit 0**. Esse gate lê
`.specs/features/ai-tool-auto-wiring/validation.md` — o relatório **anterior**, datado 2026-11-15,
que ainda carrega o veredito **FAIL**. O Verifier **não** pode sobrescrevê-lo: o escopo de escrita
desta execução é **apenas** `.specs/features/fix-autowiring-cli-contract/validation.md`.

Medição e prova:

| Cenário | Comando | Exit |
| :--- | :--- | :-: |
| Estado atual (relatório antigo, veredito FAIL) | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py ai-tool-auto-wiring --root D:\repository\my-memory` | **1** — `ERROR ai-tool-auto-wiring: validation.md verdict is FAIL` |
| **Simulação em scratch**: cópia do diretório da feature com este relatório no lugar do antigo | mesmo comando apontando para a cópia | **0** — `validate_state: 0 error(s) across [ai-tool-auto-wiring]` |

Ou seja: **a única ação que falta é mecânica** — o dono (ou o orquestrador) copia este relatório para
`.specs/features/ai-tool-auto-wiring/validation.md` e o gate fecha em **exit 0**. O conteúdo
necessário (veredito PASS + evidência `file:line`) já está provado e testado aqui.

O gate da feature **deste** relatório fecha hoje:

| Comando | Exit |
| :--- | :-: |
| `python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-autowiring-cli-contract --root D:\repository\my-memory` | **0** |

---

## 10. Achados (ranqueados)

### Achado 1 — 🟡 Minor — os dois índices de documentação ainda anunciam o alias `mem setup`

- **Condição**: leitura de `docs/README.md` ou `docs/adr/README.md`.
- **Observado**: `docs/README.md:53` — "Auto-Wiring e Instalação Zero-Touch de Ferramentas de IA
  (`mem install` / `mem setup`)"; `docs/adr/README.md:36` — o mesmo título, com status
  "Implementado (com gaps)" e a referência "spec `ai-tool-auto-wiring` (FAIL)". O título da própria
  ADR já foi corrigido (`docs/adr/026-...md:1` — "(`mem install`)"), então os dois índices estão
  **desatualizados em relação à ADR** e **anunciam de novo** o alias que a AC1 proíbe.
- **Impacto**: um leitor que chegue pelo índice ainda lê "`mem setup`" como alias do auto-wiring — o
  gap 1 (Blocker) reaberto por um caminho secundário. O critério de sucesso literal da spec
  (`grep -rn "ou \`mem setup\`" docs/`) **passa** (0 ocorrências), porque o padrão dos índices é
  `` `mem install` / `mem setup` `` e não `` ou `mem setup` ``.
- **Escopo**: `docs/adr/README.md` foi explicitamente **excluído** da T1 ("pertence a outra spec
  (ver Ownership)"), e `docs/README.md` não foi listado em nenhuma task. Nenhum dos dois foi
  tratado.
- **Correção sugerida**: alinhar `docs/README.md:53` e `docs/adr/README.md:36` ao título real da
  ADR e ao estado real da feature.

### Achado 2 — 🟡 Minor — a branch de produção do filtro de alvo não tem cobertura

- **Condição**: uma regressão confineda a `cmd/mem/install.go:125` (`autowire.DetectClients(*rootDir, *target, scope)`) — a branch usada quando **não** há probe injetado, isto é, **a que o usuário real executa**.
- **Observado**: o mutante extra **M7b** (§5.2) **sobreviveu à suíte inteira** (`ok … cmd/mem
  120.132s`). Causa: todos os 10 testes de `cmd/mem/install_test.go` chamam
  `runInstallCommandWithEnv` com env **não-nil**, e `autowire.DetectClients(` **não é chamada por
  nenhum teste** do repositório.
- **Impacto**: o filtro de `--target` (e o de escopo) é **provado** na branch de teste, mas **não**
  na branch de produção. Um mutante restrito a `cmd/mem/install.go:125` passaria a suíte inteira.
  O M7 original (gap 2) **está fechado** — a mutação que o definia morre —; o que resta é uma
  **lacuna de cobertura** na branch alternativa, não o gap original.
- **Correção sugerida**: um teste que chame `runInstallCommand` (env nil) com `--target`/`--global`
  e um probe de ambiente controlável (ou um `SystemEnv` real injetado por um ponto de corte),
  de modo que a branch `env == nil` também seja discriminada.

### Achado 3 — 🟡 Minor — a spec de origem ainda não define o que é "trecho JSON" (P2.1)

- **Condição**: leitura da AC P2.1 da spec de origem (`.specs/features/ai-tool-auto-wiring/spec.md:54`).
- **Observado**: o critério diz "mostra o caminho de cada arquivo detectado e o **trecho JSON** que
  seria inserido" — sem definir se é o arquivo inteiro, um fragmento ou o payload do servidor.
  O **comportamento** está coberto e agora asserido pelo conteúdo (`cmd/mem/install_test.go:366`,
  `:306`, `:344`), mas a **spec** continua imprecisa: o relatório anterior marcou ⚠️ aqui e o texto
  não mudou nesta correção.
- **Impacto**: nenhum defeito de produto. Um Writing-Plan futureiro pode ler "trecho" como "arquivo
  inteiro" e deixar de casar com o que o comando faz.
- **Correção sugerida**: emendar a AC P2.1 da spec de origem para dizer "o payload JSON que seria
  injetado (`mcpServers["my-memory"]`)".

### Achado 4 — ℹ️ Informativo — comportamento pré-existente, fora dos 10 gaps (mantido do relatório anterior)

- `mcpServers` presente mas **não** sendo objeto JSON é **substituído em silêncio**
  (`internal/autowire/installer.go:41-49`). A spec de correção declara explicitamente que **não**
  corrige este caso e pede que o Verifier o registre como achado remanescente. **Confirmado**:
  permanece sem teste e sem aviso.

### Achado 5 — ℹ️ Informativo — `internal/autowire/client.go:35-36` (`Exists`, `ParentDirOK`) seguem sem consumidor de produção

`Exists` e `ParentDirOK` são calculados em `internal/autowire/detector.go:220-238` e lidos apenas
por teste; a presença real vem do marcador de instalação (`Installed`,
`internal/autowire/client.go:37`, preenchido em `internal/autowire/detector.go:339-350`). É a
decisão Assumption A7 da spec — **registrada, não um defeito** —, mas mantém dois campos
calculados e não consumidos no caminho de produção.

---

## 11. O que ficou sem verificar

| Item | Por quê |
| :--- | :--- |
| **Gate com `MY_MEMORY_PG_TEST=1`** (30 testes de integração PostgreSQL) | Não existe Postgres local ou descartável nesta máquina (sem serviço, sem `docker`, sem `psql`; `127.0.0.1:5432` fechado). Rodá-lo exigiria criar e derrubar um banco **no Neon do dono**, contrariando o critério da própria T18 ("o Postgres do dono não foi contatado e nenhum banco foi criado ou apagado"). **Impacto limitado ao escopo**: nenhum dos 30 testes é da feature `ai-tool-auto-wiring` (que não tem acoplamento de banco) — todos são de `internal/store`, `internal/db`, `internal/codeast`, `internal/watcher` e de outros comandos do `cmd/mem`. |
| **`validate_state.py ai-tool-auto-wiring` com exit 0 no repositório** | O gate lê o relatório **anterior** da feature de origem, que ainda diz FAIL, e o Verifier está proibido de sobrescrevê-lo. Provado em cópia de scratch que a troca fecha o gate em exit 0 (§9). **Uma linha de trabalho resta.** |
| **Regressão em 5 commits concorrentes** (`dc7d34b`, `5bc08cc`, `c71aa82`, `335d57c`, `84aab52`, de outro agente) | Entraram no repositório **durante** esta verificação. O gate taggeado foi rodado **antes** deles; os arquivos da feature são byte-idênticos entre o baseline do sensor (`e60777e`) e o HEAD final (`dc7d34b`), mas o gate completo não foi re-executado sobre o HEAD final. |

---

## 12. Qualidade da implementação

| Princípio | Status | Observação |
| :--- | :--- | :--- |
| Sem features além do pedido | ✅ | A correção entrega os 10 gaps e nada mais; a spec de origem foi emendada, não reescrita |
| Sem abstrações para código de uso único | ✅ | `writeFileAtomic` (`internal/autowire/installer.go:143`) tem 1 consumidor e é o que permite injetar a falha de publicação; `runInstallCommandWithEnv` existe como costura de teste |
| Sem "flexibilidade" desnecessária | ⚠️ | `AcceptedTargets()`/`IsAcceptedTarget()` expõem o conjunto para a CLI e para o detector — fonte única, como a spec quer (A6). As 2 branches de detecção (§5.2, Achado 2) são consequência da costura de teste |
| Só tocou os arquivos necessários | ✅ | diff de 13 arquivos, todos dentro do escopo declarado das tasks T1–T17 |
| Não melhorou código alheio | ✅ | nenhum refactor lateral |
| Segue padrões do repo | ✅ | nomes em português, comentários `//`, `flag.NewFlagSet`, injeção de `io.Writer` e de `SystemEnv` |
| Testes mapeiam ACs e não são rasos | ✅ | o sensor é a prova: **9/9** mutantes mortos, cada guarda com controle positivo (`cmd/mem/install_test.go:189-192`, `:231-234`, `:493-495`, `:526-528`) |
| Spec-anchored (valor asseverado == valor da spec) | ✅ | §4 — 8/8 ACs de origem + 37/37 ACs da correção (1 parcial, Achado 1) |
| Todo teste do escopo mapeia para AC/critério | ✅ | §7 — os 16 + 11 testes dos pacotes da feature correspondem a ACs; os 8 nomes do plano antigo são declarados como nunca existentes |
| Guidelines do projeto seguidos | ✅ | `AGENTS.md` regra 4 (testes) e regra 8 (quality gate); `go build`, `go vet` e `gofmt` (cópia LF) rodados |

---

## 13. Conclusão

**Overall**: ✅ **PASS** — a correção da ISSUE-048 está verificada.

- **Spec-anchored check**: **8/8** ACs de origem com evidência `file:line` e asserção casando com o
  resultado da spec; **37/37** ACs da spec de correção rastreadas (1 parcial por resíduo documental).
- **Sensor**: **9/9** mutantes mortos — os 7 antigos continuam mortos e **M7** e **M8**, os dois que
  sobreviveram, agora morrem (atribuição: `TestInstallCLI_TargetRestringeAoClientePedido` e
  `TestInstallCLI_GlobalNaoEscreveNoDirDoWorkspace`, um teste cada, na suíte **completa**).
- **Gate**: 27 pacotes `ok` e 0 falha nos dois gates (simples e taggeado); build e `gofmt` verdes.
- **Contrato**: ADR-026, CLI guide, guia de agentes e a spec de origem descrevem o código que existe
  — com 2 índices de doc ainda desatualizados (Achado 1).
- **Isolamento**: `git stash` nunca usado; sensor 100% em worktree de scratch; nenhum arquivo da
  feature alterado na árvore real.
- **Postgres**: o Postgres do dono **não** foi contatado; nenhum banco foi criado ou apagado.
- **Pendências mecânicas**: promover este veredito para
  `.specs/features/ai-tool-auto-wiring/validation.md` (1 linha) e, se o dono quiser cobertura total,
  fechar a branch `env == nil` do filtro de alvo (Achado 2).