# ai-tool-auto-wiring — Validation Report

**Date**: 2026-11-15
**Spec**: `.specs/features/ai-tool-auto-wiring/spec.md`
**Tasks**: `.specs/features/ai-tool-auto-wiring/tasks.md`
**ADR**: `docs/adr/026-auto-wiring-e-instalacao-zero-touch-de-ferramentas-de-ia.md`
**Diff range**: `db02542` (commit da feature) + follow-ups `1dfd2a6`, `438bfdc`; HEAD `0f5bacd` (branch `develop`)
**Verifier**: sub-agente independente (autor != verificador)
**Verdict**: **FAIL**

> Este arquivo substitui o "Validation Plan" anterior, que nunca foi executado: todos os
> checkboxes dele estavam marcados sem que existisse evidencia. O conteudo abaixo e o
> resultado real da verificacao.

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1 Deteccao de clientes e caminhos | ⚠️ Entregue com ressalva | `internal/autowire/detector.go` cobre Windows/macOS/Linux e os 4 clientes; porem a "verificacao de presenca/instalacao" pedida no checkbox nao e consumida por ninguem (ver Gap 5) |
| T2 Merge JSON + backup | ⚠️ Entregue com ressalva | Merge e backup funcionam (`internal/autowire/installer.go:41-72`, `:103-109`); "backup atomico" e "preservando indentacao" nao sao verdadeiros (Gap 6, Gap 7) |
| T3 CLI `mem install` + alias `mem setup` | ❌ Falha | `mem install` existe e funciona (`cmd/mem/install.go:16-151`, dispatch em `cmd/mem/main.go:836`); o alias `mem setup` NAO existe no HEAD — `mem setup` e o assistente interativo (Gap 1) |
| T4 ADR-026 + documentacao | ⚠️ Entregue com ressalva | ADR e CLI guide existem, mas descrevem capacidades que a implementacao nao tem e omitem o comando (Gap 4, Gap 8, Gap 9) |

---

## Spec-Anchored Acceptance Criteria

| Criterio (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertiva | Resultado |
| ------------------------ | -------------------- | ----------------------- | --------- |
| P1.1 WHEN executa `mem install` THEN localiza o arquivo de config do cliente conforme o SO | caminho exato por SO/cliente (`claude_desktop_config.json`, `.cursor/mcp.json`, `.vscode/mcp.json`, `.codeium/windsurf/mcp_config.json`) | `internal/autowire/detector_test.go:50` — `len(targets) != 5` e `strings.Contains(claude.ConfigPath, "AppData\Roaming\Claude\claude_desktop_config.json")`; `internal/autowire/detector_test.go:129` — `targets[0].ConfigPath != expected` (darwin); `internal/autowire/detector_test.go:150` — idem (linux) | ✅ PASS |
| P1.2 WHEN o JSON ja existe THEN faz merge inserindo `mcpServers["my-memory"]` sem remover outros servidores | `other-server` e as chaves paralelas permanecem | `internal/autowire/installer_test.go:113` — `parsed["globalSetting"] != true`; `:118` — `servers["other-server"]` ausente => erro; `:122` — `servers["my-memory"]` ausente => erro. Impl.: `internal/autowire/installer.go:41-49` + `:71-72` | ✅ PASS |
| P1.3 WHEN o arquivo existe THEN cria backup `.bak` antes de alterar | `.bak` existe e contem o conteudo original | `internal/autowire/installer_test.go:101` — `report.BackupPath == ""` => erro; `:104` — `os.Stat(configFile + ".bak")`. E2E: `cmd/mem/install.go:132` imprime `Backup:` (observado em execucao real) | ✅ PASS |
| P1.4 WHEN o arquivo nao existe THEN cria o diretorio e o JSON formatado | arquivo criado, JSON valido, `mcpServers.my-memory` presente | `internal/autowire/installer_test.go:32` — `report.Status != StatusInstalled`; `:36-44` — arquivo criado e JSON parseavel; `:46-49` — `mcpServers` presente. Impl.: `internal/autowire/installer.go:95-101` (MkdirAll) + `:112-116` (WriteFile) | ✅ PASS |
| P2.1 WHEN `--dry-run` THEN mostra o caminho de cada arquivo detectado e o trecho JSON que seria inserido | "trecho JSON" nao definido na spec (arquivo inteiro? fragmento?) | `internal/autowire/installer_test.go:193` — `report.Diff == ""` => erro; CLI em `cmd/mem/install.go:138-139`. **Sem assertiva de conteudo JSON no nivel CLI** (`cmd/mem/install_test.go:24` so checa o badge `[DRY-RUN]`) | ⚠️ Spec-precision gap |
| P2.2 WHEN `--dry-run` THEN nenhum arquivo em disco e criado ou alterado | zero escrita em disco | `internal/autowire/installer_test.go:198` — `os.Stat(configFile)` deve ser `IsNotExist`; `cmd/mem/install_test.go:33` — `.cursor/mcp.json` nao deve existir. Impl.: `internal/autowire/installer.go:84` retorna antes de `:95-116` | ✅ PASS |
| P3.1 WHEN `--target <cliente>` THEN aceita `claude`, `cursor`, `vscode`, `windsurf` ou `all` | conjunto de valores aceito | `internal/autowire/detector.go:222` — `matchTarget`; `internal/autowire/detector_test.go:125` — `len(targets) != 1` para target `claude`; `internal/autowire/detector_test.go:146` — idem. CLI repassa em `cmd/mem/install.go:85` | ⚠️ PASS na camada de dominio, ❗sem discriminacao na camada CLI (mutante M7 sobreviveu) |
| P3.2 WHEN `--workspace`/`--global` THEN filtram o escopo da instalacao | escopo global x workspace respeitado | `internal/autowire/detector.go:226` — `matchScope`; `internal/autowire/detector_test.go:164` — `len(wsTargets) != 2` e `:175` — `len(globalTargets) != 3`. CLI em `cmd/mem/install.go:39-45` | ⚠️ PASS na camada de dominio, ❗sem discriminacao na camada CLI (mutante M8 sobreviveu) |

**Status**: 8/8 ACs com evidencia `file:line`; 1 spec-precision gap (P2.1); 2 ACs (P3.1, P3.2) protegidas apenas no nivel de dominio — a fronteira CLI (`cmd/mem/install.go`) nao discrimina.

### Goals (spec.md:14-17)

| Goal | Spec-defined outcome | Evidencia | Resultado |
| ---- | -------------------- | --------- | --------- |
| G1 | pacote `internal/autowire/` com deteccao dos 4 clientes em Win/macOS/Linux | `internal/autowire/detector.go:69-158` | ⚠️ deteccao entregue; "auto-detecta ferramentas **instaladas**" nao (Gap 5) |
| G2 | `MergeMCPServerConfig` + backup **atomico** (`.bak`) | funcao real chama-se `InjectMCPServer` (`internal/autowire/installer.go:12`); backup em `:105-108` e `os.WriteFile` simples (nao atomico) | ⚠️ nome e atomicidade divergem da spec (Gap 6) |
| G3 | `mem install` **com alias `mem setup`** | `cmd/mem/main.go:836` — `case "install":`; `cmd/mem/main.go:794` — `case "setup":` chama `runSetupCLI` (assistente interativo, ADR-033) | ❌ FAIL (Gap 1) |
| G4 | ADR-026 + documentacao da CLI | `docs/adr/026-...md:1`, `docs/CLI_GUIDE.md:625` | ⚠️ ADR descreve arquitetura inexistente; docs anunciam valores que nao funcionam (Gap 4, Gap 8, Gap 9) |

---

## Edge Cases

- [x] JSON corrompido: erro seguro sem corromper mais nada — `internal/autowire/installer_test.go:203-221` (`strings.Contains(err.Error(), "JSON invalido")`); impl. `internal/autowire/installer.go:30-36`.
- [x] Idempotencia: segunda injecao identica nao escreve e nao cria `.bak` — `internal/autowire/installer_test.go:154` (`report.Status != StatusAlreadyUpToDate`) e `:163` (`force=true` => `StatusUpdated`).
- [x] Diretorio pai inexistente: criado sob demanda — `internal/autowire/installer.go:96-101`; coberto por `internal/autowire/installer_test.go:13` (caminho em `subdir/config.json`).
- [x] `--target` desconhecido: nao quebra, mas **falha em silencio** — `cmd/mem/install.go:90-93` imprime "Nenhum cliente MCP compativel encontrado" e sai com codigo 0 (Gap 4, severidade baixa).
- [ ] `mcpServers` presente mas **nao** sendo objeto JSON: o valor original e silenciosamente substituido — `internal/autowire/installer.go:41-49` (caminho `else` apaga o dado original sem aviso). Sem teste.
- [ ] Backup **atomico**: `os.WriteFile` direto em `.bak` (`internal/autowire/installer.go:106`) — uma queda no meio da escrita deixa um `.bak` truncado, exatamente o caso que o backup existe para evitar. Sem teste (Gap 6).
- [ ] `mem install` em **cliente nao instalado**: cria o arquivo de configuracao assim mesmo (Gap 5). Comprovado em execucao real: diretorio vazio de scratch, sem Cursor nem VS Code, recebeu `.cursor/mcp.json` e `.vscode/mcp.json`.

---

## Gate Check

- **Gate command**: `tasks.md` **nao tem** secao "Gate Check Commands" — conforme a instrucao, caiu para a suite completa: `go test -count=1 ./...`, mais `go build ./...` e `go vet ./...`. O gate proprio do repo para esta feature no CI e `go test -v -cover ./internal/autowire/...` (`.github/workflows/ci.yml:108-109`) e `go test -v -cover ./cmd/mem/...` (`.github/workflows/ci.yml:120-121`) — os dois pacotes da feature **estao** no CI.
- **Result**: 27 pacotes `ok`, **0 failed**, 0 skipped (suite pura). `go build ./...` exit 0. `go vet ./...` exit 0.
- **Scoped**: `go test -count=1 ./internal/autowire/...` -> ok (0.58s); `go test -count=1 ./cmd/mem/...` -> ok (124s).
- **Test count before feature** (`db02542^` = `03915aa`): `internal/autowire` = 0 (pacote novo), `cmd/mem` = 40.
- **Test count after feature** (HEAD): `internal/autowire` = 9, `cmd/mem` = 261.
- **Delta atribuivel a esta feature**: **+12 funcoes de teste** (9 em `internal/autowire`, 3 em `cmd/mem`: `TestInstallCLI_DryRun`, `TestInstallCLI_WorkspaceExecution`, `TestRearrangeInstallArgs`). O restante do crescimento de `cmd/mem` (40 -> 261) vem de features posteriores.
- **Skipped tests**: nenhum. Nenhum teste desta feature exige Postgres (a feature e de arquivo, nao de banco); a integracao Postgres do repo e opt-in por `MY_MEMORY_PG_TEST=1` (`internal/pgtest/pgtest.go:42`) e **nao foi habilitada** — o Postgres do dono nao foi contatado, nenhum banco foi criado ou apagado.
- **Failures**: nenhum.

---

## Discrimination Sensor

Worktree isolado: `D:\.minimax\scratch\subagents\verifier-atw\wt` (`git worktree add ... HEAD`). Baseline `git status --porcelain` da arvore real registrado ANTES do sensor (vazio) e reconferido DEPOIS (vazio, identico). `git stash`/`git stash pop` nao foram usados. Worktree do scratch restaurado (`git status --porcelain` vazio).

| # | File:line | Mutacao (comportamento) | Morto? |
| - | --------- | ----------------------- | ----- |
| M1 | `internal/autowire/installer.go:71` | merge deixa de preservar os outros servidores (`mcpServers = map[...]{serverName: ...}`) | ✅ Killed |
| M2 | `internal/autowire/installer.go:84` | `--dry-run` deixa de bloquear a escrita (`if dryRun` -> `if false`) | ✅ Killed |
| M3 | `internal/autowire/installer.go:104` | backup `.bak` deixa de ser criado (`if len(originalData) > 0` -> `if false`) | ✅ Killed |
| M4 | `internal/autowire/detector.go:222` | `--target` deixa de filtrar clientes (`matchTarget` -> `return true`) | ✅ Killed |
| M5 | `internal/autowire/detector.go:226` | escopo deixa de ser respeitado (`matchScope` -> `return true`) | ✅ Killed |
| M6 | `internal/autowire/detector.go:79` | deteccao do caminho do Claude Desktop deixa de achar o arquivo certo | ✅ Killed |
| M7 | `cmd/mem/install.go:85` | **CLI ignora a flag `--target`** (sempre detecta tudo) | ❌ **Sobreviveu** |
| M8 | `cmd/mem/install.go:41-45` | **CLI ignora `--workspace`/`--global`** (escopo sempre `all`) | ❌ **Sobreviveu** |
| M9 | `cmd/mem/install.go:110` | CONTROLE: chave gravada deixa de ser `my-memory` | ✅ Killed |

**Sensor depth**: lightweight ampliado (9 mutacoes; o minimo do skill e 3).
**Result**: 7/9 mortos — **FAIL**. M9 morreu, o que prova que o caminho de teste de `cmd/mem` **discrimina de fato**: M7/M8 sobreviveram por teste fraco, nao por execucao quebrada.

### Sobre M7 e M8

As duas flags que a story P3 descreve (`--target` e `--workspace`/`--global`) sao consumidas em `cmd/mem/install.go:39-45` e `:85`, mas **nenhum teste passa por `runInstallCommand` e verifica o lado negativo do filtro**:

- `cmd/mem/install_test.go:38-83` (`TestInstallCLI_WorkspaceExecution`) usa `--target cursor` e `--workspace`, mas so afirma que o `.cursor/mcp.json` foi criado — nunca afirma que o `.vscode/mcp.json` **nao** foi. Com M7/M8 aplicados o teste continua verde.
- Nao existe nenhum teste de `--global` em lugar nenhum.

Consequencia pratica: `mem install --target cursor` pode passar a escrever em todos os clientes e a suite continua verde.

---

## Codigo e Qualidade da Implementacao

| Principio | Status | Observacao |
| --------- | ------ | ---------- |
| Sem features alem do pedido | ❌ | `ServerConfig.Env` (`internal/autowire/client.go:26`) e processado pelo instalador (`internal/autowire/installer.go:56-58`) mas **nenhum produtor** o preenche (`cmd/mem/install.go:79-82`); `StatusSkipped` (`internal/autowire/client.go:47`) nunca e produzido nem consumido |
| Sem abstracao para codigo de uso unico | ⚠️ | `isPayloadEqual` termina com `reflect.DeepEqual(existingMap["command"], next["command"])` (`internal/autowire/installer.go:175`) — cauda **sempre verdadeira**, porque a comparacao `!=` da linha 136 ja teria retornado antes |
| Sem "flexibilidade" desnecessaria | ⚠️ | `ClientTarget.ParentDirOK` (`internal/autowire/client.go:36`) e escrito em `internal/autowire/detector.go:215` e **nao e lido em lugar nenhum** (morto em producao; `Exists` so e lido em `internal/autowire/detector_test.go:93`) |
| Só tocou os arquivos necessarios | ✅ | diff da feature = 17 arquivos, todos dentro do escopo |
| Nao melhorou codigo alheio | ✅ | sem refactor lateral |
| Segue padroes do repo | ✅ | nomes em portugues, `//` comments, injecao de `io.Writer` para teste, `flag.NewFlagSet` |
| Um engenheiro senior aprovaria? | ⚠️ | os 9 gaps abaixo sao bloqueadores ou furos de contrato |
| Testes mapeiam ACs e nao sao rasos | ❌ | M7/M8 sobreviveram: os testes das flags P3 nao sao rasos **o suficiente** |
| Spec-anchored: valor asseverado == valor da spec | ❌ | `--target claude-desktop` documentado nao e aceito (Gap 4); alias `mem setup` documentado nao existe (Gap 1) |
| Coverage por camada: dominio 1:1 AC; rotas cobrem feliz+edge+erro | ❌ | dominio coberto; a fronteira CLI (a que o usuario toca) nao |
| Todo teste do escopo mapeia para AC/criterio | ❌ | 4 testes do plano antigo nao existem (Gap 10); `cmd/mem/install_scripts_test.go` pertence a outra feature (testa `scripts/install.ps1|sh`) |
| Guidelines do projeto seguidos | ✅ | `AGENTS.md` regra 4 (testes) e 8 (quality gate) respeitadas; `go build`/`go vet`/`gofmt` rodados |

---

## Achados (ranqueados)

### Gap 1 — Blocker — o alias `mem setup` nao existe no HEAD (G3 nao cumprida)
- **Condicao**: `mem setup` invocado pelo usuario.
- **Observado**: `cmd/mem/main.go:836` roteia `case "install":` e `cmd/mem/main.go:794` roteia `case "setup":` para `runSetupCLI` — o assistente interativo de cofre central (ADR-033), que nao tem nada a ver com auto-wiring. O alias existiu entre `db02542` e `8685ed8` e foi repurposado depois.
- **Impacto**: a meta G3 da spec nao e atendida; e a documentacao que a feature wrote continua anunciando um comando que nao faz o que promete:
  - `docs/CLI_GUIDE.md:625` — titulo "`mem install` **ou `mem setup`**";
  - `docs/CLI_GUIDE.md:650-651` — "Alias identico de setup: `./bin/mem.exe setup --dry-run`" — **executado de verdade**: `flag provided but not defined: -dry-run`, exit != 0;
  - `docs/adr/026-...md:1`, `:18`, `:27`, `:57` — "`mem install` (com alias de conveniencia `mem setup`)".
- **Evidencia**: `cmd/mem/main.go:794`, `cmd/mem/main.go:836`; `git show 8685ed8 -- cmd/mem/main.go` (remove `case "install", "setup":` e cria `case "setup":`).

### Gap 2 — Major — P3.1/P3.2 sem discriminacao na camada CLI (mutantes M7 e M8 sobreviveram)
- **Condicao**: `mem install --target <x>` ou `mem install --workspace|--global` com o filtro quebrado.
- **Observado**: mutantes M7/M8 sobreviveram a suite completa.
- **Impacto**: uma regressao que faça o comando escrever em todos os clientes, ou em escopo errado, **nao quebra nenhum teste**. As ACs P3.1 e P3.2 do produto sao exatamente esse comportamento.
- **Evidencia**: `cmd/mem/install.go:39-45`, `cmd/mem/install.go:85`; `cmd/mem/install_test.go:38-83` (so afirma o lado positivo); ausencia de qualquer teste de `--global`.

### Gap 3 — Major — o payload aponta para o SQLite enquanto a CLI declara que o Postgres venceu
- **Condicao**: ambiente com `MY_MEMORY_PG_URL` definido (config global `engine: postgres`).
- **Observado**: `cmd/mem/install.go:63` descarta o terceiro valor de `resolveStorageAndRepo` (a URL do Postgres) e monta o payload so com `--db`/`--repo` (`cmd/mem/install.go:71-77`). Execucao real (`--dry-run --global --target claude`): o log diz `Postgres detectado no config/env. Central vault sera usado. SQLite local em C:\Users\Felipe\.memory\memory.db sera ignorado nesta sessao.` e, duas linhas abaixo, `Database: C:\Users\Felipe\.memory\memory.db`.
- **Impacto**: o payload injeta `--db` para um banco SQLite que a propria CLI declarou estar descartando. Se o editor nao herdar `MY_MEMORY_PG_URL`, `mem mcp --db <esse caminho>` abre um **store diferente** do vault vivo (`cmd/mem/main.go:403-422`: com `resolvedPG == ""` ele abre o SQLite em `resolvedDB`; com Postgres, o `--db` e ignorado). A precedencia da ADR-040 nao e respeitada pelo auto-wiring.
- **Evidencia**: `cmd/mem/install.go:63`, `cmd/mem/install.go:71-77`, `cmd/mem/main.go:403-422`, `cmd/mem/main.go:1441-1450`.

### Gap 4 — Major — `--target claude-desktop` documentado e nao aceito
- **Condicao**: `mem install --target claude-desktop`.
- **Observado**: `docs/CLI_GUIDE.md:631` lista `claude-desktop` como valor aceito; `internal/autowire/detector.go:222` so casa `claude`. Execucao real: "Nenhum cliente MCP compativel encontrado para o filtro '--target claude-desktop' e escopo 'all'" — **exit 0, sem erro**. O mesmo nome aparece como ID canonico em `docs/adr/026-...md:32`.
- **Impacto**: o valor que a propria documentacao ensina nao funciona e falha em silencio.

### Gap 5 — Major — "auto-detecta ferramentas instaladas" nao e implementado; T1 entregue pela metade
- **Condicao**: `mem install` numa maquina sem Cursor/VS Code/Windsurf/Claude.
- **Observado**: `internal/autowire/detector.go:69-158` devolve **todos** os candidatos, sem nenhuma trava de instalacao; `Exists`/`ParentDirOK` (`internal/autowire/detector.go:199-217`) nunca sao consultados por `cmd/mem/install.go:109-142`; `internal/autowire/installer.go:95-101` cria o diretorio e escreve o arquivo de qualquer jeito. Execucao real em diretorio de scratch vazio: criou `.cursor/mcp.json` e `.vscode/mcp.json`.
- **Impacto**: o `mem install` cria configuracao para ferramentas que o usuario nao tem — o oposto do "zero-touch" prometido; e o checkbox de T1 "Implementar verificacao de presenca/instalacao de cada ferramenta" esta marcado sem consumidor (campo morto).

### Gap 6 — Minor — o backup nao e atomico
- **Condicao**: queda do processo durante a escrita do `.bak`.
- **Observado**: `internal/autowire/installer.go:105-108` faz `os.WriteFile(backupFile, originalData, 0644)` — sem arquivo temporario + rename.
- **Impacto**: `.bak` truncado, que e justamente o cenario que o backup precisa cobrir. A spec (G2) e o ADR-026:19 exigem "backup atomico".

### Gap 7 — Minor — indentacao e ordem das chaves nao sao preservadas
- **Condicao**: arquivo de configuracao com indentacao de 4 espacos e chaves fora de ordem alfabetica.
- **Observado**: `internal/autowire/installer.go:75` reserializa com `json.MarshalIndent(cfgMap, "", "  ")`; `map[string]interface{}` serializa as chaves em ordem alfabetica. Execucao real: original de 4 espacos com `zetaGlobal` antes de `alphaSetting` saiu com 2 espacos e `alphaSetting` primeiro. O conteudo (servidores e chaves paralelas) foi preservado — a AC P1.2 da spec **nao** e violada; o checkbox de T2 "preservando indentacao" e que nao se sustenta.
- **Evidencia**: `internal/autowire/installer.go:75`, `internal/autowire/installer.go:71-72`.

### Gap 8 — Minor — ADR-026 descreve uma arquitetura e capacidades que nao existem
- **Condicao**: leitura da ADR como documento normativo.
- **Observado**: `docs/adr/026-...md:31-35` documenta um registro declarativo `ClientSpec` (ID/Name/Scope/PlatformPaths/Detector) — o codigo tem blocos `if` fixos (`internal/autowire/detector.go:69-158`) e constantes `ClientType` (`internal/autowire/client.go:6-11`). `docs/adr/026-...md:42` documenta Windsurf com escopo **Global / Workspace** (`<workspace>/.windsurf/mcp.json`) — so existe o escopo global (`internal/autowire/detector.go:143-156`). O ADR ainda repete o alias `mem setup` (Gap 1).

### Gap 9 — Minor — T4 parcial: o guia de integracao nao menciona mais `mem install`
- **Condicao**: leitura de `docs/AGENT_INTEGRATION_GUIDE.md`.
- **Observado**: a linha `| mem install | Auto-wiring zero-touch ... |` foi adicionada pela feature (`db02542`) e removida depois (`f5646fd`); hoje o arquivo so menciona `mem hook install` (`docs/AGENT_INTEGRATION_GUIDE.md:271`). O comando continua documentado no `docs/CLI_GUIDE.md:625`.

### Gap 10 — Minor — o plano de validacao anterior listava 8 testes que nunca existiram
- **Condicao**: auditoria do plano former.
- **Observado**: dos 12 nomes marcados como feitos no plano antigo, so 4 existem hoje (`TestInjectMCPServer_NewFile`, `TestInjectMCPServer_DryRun`, `TestInjectMCPServer_InvalidJSON`, `TestInstallCLI_DryRun`). Nao existem: `TestDetectClients_PathResolution`, `TestDetectClients_ScopeFiltering`, `TestDetectClients_TargetFiltering`, `TestInjectMCPServer_MergeExisting`, `TestInjectMCPServer_BackupCreation`, `TestInstallCLI_WorkspaceOnly`, `TestInstallCLI_TargetFlag`, `TestInstallCLI_AliasSetup`. O plano tambem afirmava "Todos os 15 pacotes Go continuam passando" (hoje sao 27) e "Teste real no Windows" (nao ha tal teste).
- **Evidencia**: `internal/autowire/detector_test.go:50,114,135,156`, `internal/autowire/installer_test.go:11,61,168,203`, `cmd/mem/install_test.go:13,38,85`.

---

## Fix Plans

### Fix 1 — Restore (ou remover) o alias `mem setup`
- **Root cause**: `8685ed8` repurposou o verbo `setup` para o assistente de cofre central sem atualizar spec/ADR/docs do auto-wiring.
- **Tarefa**: escolher **um** dono do verbo `setup`. Se o assistente mantem o verbo, remover o alias da spec (`spec.md:16`), do ADR-026 (`:1,:18,:27,:57`) e do CLI guide (`:625,:651`), e escrever o alias do auto-wiring como `mem install` (ou `mem install --setup`).
- **Onde**: `docs/adr/026-...md`, `docs/CLI_GUIDE.md:625,651`, `spec.md:16`, `tasks.md:23`; opcionalmente `cmd/mem/main.go:836`.
- **Verificar**: `mem setup --dry-run` falha com mensagem explicita OU `mem install --help` documenta o alias; grep por "ou `mem setup`" nos docs nao retorna nada obsoleto.
- **Done when**: nenhum doc afirma `mem setup` como alias do auto-wiring, e o comportamento real bate com o texto.

### Fix 2 — Proteger P3.1 e P3.2 na fronteira CLI
- **Root cause**: os testes exercitam `runInstallCommand` so pelo lado positivo do filtro.
- **Tarefa**: em `cmd/mem/install_test.go`, (a) com `--target cursor --workspace`, afirmar que `.vscode/mcp.json` **nao** existe e que a saida **nao** menciona "VS Code"; (b) com `--global`, afirmar que apenas alvos globais aparecem e que nenhum `.cursor/mcp.json`/`.vscode/mcp.json` e criado no `--dir`; (c) com `--target` invalido, afirmar o texto de erro.
- **Onde**: `cmd/mem/install_test.go`.
- **Verificar**: reaplicar M7 e M8 — os dois precisam morrer.
- **Done when**: M7 e M8 morrem; `go test -count=1 ./cmd/mem/...` verde.

### Fix 3 — Resolver o vault com a mesma precedencia da ADR-040
- **Root cause**: `cmd/mem/install.go:63` descarta o Postgres resolvido e o payload nunca declara o engine.
- **Tarefa**: usar o valor de `pg` devolvido por `resolveStorageAndRepo`; quando houver Postgres, injetar `--storage postgres` (ou `--postgres <url>`, sem vazar a URL no arquivo do editor — nesse caso `--storage postgres` + a URL vinda do ambiente) em vez de `--db <sqlite>`.
- **Onde**: `cmd/mem/install.go:61-82`.
- **Verificar**: com `MY_MEMORY_PG_URL` apontando para uma porta fechada, `mem install --dry-run` mostra `Database:`/`Engine:` coerentes com o log de storage.
- **Done when**: o payload do dry-run nao aponta para um `.db` que a propria CLI declarou descartar.

### Fix 4 — Alinhar `--target` aceito com o documentado
- **Root cause**: `matchTarget` (`internal/autowire/detector.go:222`) so conhece `claude`.
- **Tarefa**: aceitar `claude-desktop` como sinonimo de `claude` (e normalizar), ou corrigir `docs/CLI_GUIDE.md:631` e `docs/adr/026-...md:32` para `claude`; validar o valor e falhar com codigo != 0 em vez de "nenhum cliente encontrado" com exit 0.
- **Onde**: `internal/autowire/detector.go:222-224`, `cmd/mem/install.go:90-93`, `docs/CLI_GUIDE.md:631`, `docs/adr/026-...md:32`.
- **Verificar**: `mem install --target claude-desktop --dry-run` lista o Claude Desktop; `mem install --target xxxx --dry-run` sai != 0.
- **Done when**: o conjunto de valores aceitos == o conjunto documentado.

### Fix 5 — Sobjecutar clientes instalados
- **Root cause**: `Exists`/`ParentDirOK` sao calculados e nunca usados; nao ha trava de instalacao no detector.
- **Tarefa**: usar `ParentDirOK`/`Exists` para marcar o alvo como nao instalado e nao escrever (ou escrever so com flag explicita), e dar um status real (`StatusSkipped` ja existe e nunca e usado).
- **Onde**: `internal/autowire/detector.go:184-220`, `cmd/mem/install.go:109-142`, `internal/autowire/client.go:47`.
- **Verificar**: `mem install --dry-run` num `--dir` vazio nao lista clientes inexistentes; teste novo mata um mutante que ignora a trava.
- **Done when**: `mem install` numa maquina sem as ferramentas nao cria `~/.cursor/mcp.json` et al.

### Fix 6 — Backup atomico + preservar formatacao (opcional)
- **Root cause**: `internal/autowire/installer.go:105-108` escreve direto no destino.
- **Tarefa**: gravar o `.bak` via temporario + rename; opcionalmente preservar indentacao e ordem original das chaves.
- **Onde**: `internal/autowire/installer.go:103-116`.
- **Verificar**: mutante que substitui o rename por `WriteFile` direto morre.
- **Done when**: `.bak` nunca aparece truncado; indentacao do original preservada (ou a spec reescrita para dizer 2 espacos).

### Fix 7 — Corrigir ADR-026 e os docs
- **Root cause**: ADR e guia descrevem capacidades inexistentes.
- **Tarefa**: alinhar `docs/adr/026-...md:31-35` (`ClientSpec` -> implementacao real), `:42` (escopo do Windsurf), e `docs/AGENT_INTEGRATION_GUIDE.md` (incluir `mem install`).
- **Onde**: `docs/adr/026-...md`, `docs/AGENT_INTEGRATION_GUIDE.md`, `docs/adr/README.md:36` (status "Implementado (sem verificacao)").
- **Verificar**: cada linha da ADR aponta para `file:line` existente.
- **Done when**: ADR e CLI guide descrevem exatamente o que o binario faz.

---

## Requirement Traceability

| Requirement | Previous | New |
| ----------- | -------- | --- |
| P1.1 (deteccao de caminho) | Implementing | ✅ Verified |
| P1.2 (merge nao-destrutivo) | Implementing | ✅ Verified |
| P1.3 (backup `.bak`) | Implementing | ✅ Verified (backup, nao atomicidade — Gap 6) |
| P1.4 (cria diretorio/arquivo) | Implementing | ✅ Verified |
| P2.1 (dry-run mostra payload) | Implementing | ⚠️ Spec-precision gap |
| P2.2 (dry-run nao escreve) | Implementing | ✅ Verified |
| P3.1 (`--target`) | Implementing | ❌ Needs Fix (sem discriminacao na CLI — Gap 2; valor documentado invalido — Gap 4) |
| P3.2 (`--workspace`/`--global`) | Implementing | ❌ Needs Fix (sem discriminacao na CLI — Gap 2) |
| G1 (pacote de deteccao) | Implementing | ⚠️ Needs Fix (Gap 5) |
| G2 (merge + backup atomico) | Implementing | ⚠️ Needs Fix (Gap 6, Gap 7) |
| G3 (`mem install` + alias) | Implementing | ❌ Needs Fix (Gap 1) |
| G4 (ADR + docs) | Implementing | ⚠️ Needs Fix (Gap 8, Gap 9) |

---

## Higiene e Isolamento

- **Arvore real**: `git status --porcelain` registrado **antes** do sensor (vazio) e reconferido **depois** (vazio, identico). As unicas escritas na arvore real foram este relatorio e o bookkeeping de lessons (`.specs/lessons.json`, `.specs/LESSONS.md`). Nenhum commit, nenhum push, nenhuma alteracao de codigo, nenhum `go mod tidy`, nenhum `gofmt -w`.
- **Scratch**: tudo em `D:\.minimax\scratch\subagents\verifier-atw\` (worktree `wt` para o sensor, worktree `wt-before` para a contagem de testes "antes", `demo/` e `fmt/` para as execucoes reais). Nada em `%TEMP%` nem na raiz do workspace.
- **Postgres**: a feature e de arquivo; nenhum teste exige Postgres. `MY_MEMORY_PG_TEST` **nao** foi habilitada, nenhuma URL foi impressa, logada ou gravada, nenhum banco foi criado ou apagado, e o Postgres do dono nao foi contatado.
- **Execucoes reais**: `mem install --dry-run` (leitura), `mem install --workspace --dir <scratch>` (escrita em scratch), `mem install --workspace --target cursor --dir <scratch> --db <scratch>` (forca `.bak` em scratch). O `mem install --dry-run --global` leu (somente leitura) o `claude_desktop_config.json` real do usuario — nenhuma escrita nele.
- **Ruido de ambiente (nao e defeito da feature)**: `gofmt -l .` lista 134 arquivos, incluindo os 5 de `internal/autowire` — `git ls-files --eol internal/autowire/client.go` mostra `i/lf w/crlf`: o blob e LF e o working tree e CRLF porque `.gitattributes` (`* text=auto eol=lf`) e posterior ao checkout. O CI roda em Linux e passa. Alem disso, `go test ./...` local inclui `viewer/node_modules/flatted/golang/pkg/flatted` (artefato de `node_modules` no workspace).

---

## Summary

**Overall**: ❌ Not Ready — FAIL

**Spec-anchored check**: 8/8 ACs com evidencia `file:line`; 1 spec-precision gap (P2.1); 2 ACs (P3.1, P3.2) sem discriminacao na camada CLI; 3 goals fora (G1, G2, G4) e 1 goal nao cumprido (G3).

**Sensor**: 9 mutacoes, 7 mortas, **2 sobreviveram** (M7 `--target`, M8 `--workspace`/`--global`).

**Gate**: 27 pacotes `ok`, 0 failed; `go build ./...` e `go vet ./...` exit 0; `internal/autowire` e `cmd/mem` no CI (`.github/workflows/ci.yml:108-109`, `:120-121`).

**O que funciona**: merge nao-destrutivo de `mcpServers`, backup `.bak`, criacao de diretorio/arquivo, idempotencia + `--force`, `--dry-run` sem escrita, deteccao multi-OS de caminho, auto-scoping do binario (`os.Executable()`), saida por cliente com status.

**Achados**: 10 gaps ranqueados (1 Blocker, 3 Major, 6 Minor) com 7 fix plans acima. O Blocker (Gap 1) e' o alias `mem setup`: a feature o implementou, uma feature posterior repurposou o verbo, e a documentacao da feature continua prometendo um comando que nao existe.

**Proximos passos**: Fix 1 (alias) -> Fix 2 (proteger P3 na CLI) -> Fix 3 (precedencia de storage) -> Fix 4 (`--target` aceito == documentado) -> Fix 5 (sobjectar instalados) -> Fix 6/7 (backup atomico + ADR/docs). Re-verificar depois do Fix 2 com os mutantes M7/M8 re-aplicados: os dois precisam morrer.