# Feature: fix-autowiring-cli-contract

- **Spec ID**: `fix-autowiring-cli-contract`
- **Created**: 2026-11-16
- **Owner**: Felipe Miiller
- **Source**: ISSUE-048 — `.specs/ISSUES.md` (seção ISSUE-048) · fecha os **10 gaps** da feature `ai-tool-auto-wiring` (ADR-026)
- **Verificação de origem**: `.specs/features/ai-tool-auto-wiring/validation.md` — veredito **FAIL** (8/8 ACs com evidência `file:line`, 3 goals divergentes, G3 não cumprido; sensor com **9 mutações, 7 mortas e 2 sobrevividas** — M7 e M8)
- **Related**: ADR-026 (auto-wiring e instalação zero-touch) · ADR-033 (assistente de cofre central — dono do verbo `setup`) · ADR-040 (precedência de storage: flag > env > config) · lições L-027, L-028 e L-029
- **Consolidação (2026-11-16)**: esta é a **versão final única** da ISSUE-048. Ela absorve a spec paralela `.specs/features/fix-ai-tool-auto-wiring-gaps/` (18 ACs / 11 tasks), que cobria os **9 gaps** de comportamento/documentação e **omitia o gap 10** (o plano de validação antigo listava 8 testes que nunca existiram). O que era **material** na versão paralela foi incorporado: o **gap 10** (o relatório novo cita apenas testes que existem), o edge case das **flags de escopo contraditórias**, o edge case da **idempotência** (`already_up_to_date`) e o `--dry-run` que não lista cliente inexistente. Onde as duas versões **discordavam** — o conjunto aceito de `--target`, a forma de detectar a presença do cliente e o comportamento de `mem setup --dry-run` — a decisão está na tabela de Assumptions com o **default, o rationale e a alternativa**, para que o dono possa inverter em uma linha. A spec paralela é removida pelo dono; as duas não podem ser implementadas juntas.

## Problem Statement

O auto-wiring (`mem install`) **escreve** a configuração MCP nos editores — e o
**contrato publicado** promete coisas que o binário não faz. A verificação
independente devolveu veredito **FAIL** com 10 gaps ranqueados: 1 🔴 Blocker,
3 🟠 Major e 6 🟡 Minor.

| # | Sev. | Gap | Onde (evidência `file:linha`) |
| :-: | :--- | :--- | :--- |
| 1 | 🔴 **Blocker** | **`mem setup` não é alias do auto-wiring** — `cmd/mem/main.go:836` roteia `case "install":` e `cmd/mem/main.go:794` roteia `case "setup":` para o **assistente interativo de cofre central** (ADR-033). O alias existiu entre `db02542` e `8685ed8` e foi **repurposado**; a doc continua anunciando-o | `cmd/mem/main.go:794,836` · `docs/CLI_GUIDE.md:625,650-651` · `docs/adr/026-...md:1,18,27,57` |
| 2 | 🟠 Major | **P3.1/P3.2 sem discriminação na camada CLI** — os mutantes **M7** (`cmd/mem/install.go:85` ignora `--target`) e **M8** (`cmd/mem/install.go:39-45` ignora `--workspace`/`--global`) **sobreviveram à suíte inteira**: uma regressão que escreva em todos os clientes, ou no escopo errado, **não quebra nenhum teste** | `cmd/mem/install.go:39-45,85` · `cmd/mem/install_test.go:38-83` |
| 3 | 🟠 Major | **o payload aponta para o SQLite enquanto a CLI declara que o Postgres venceu** — `cmd/mem/install.go:63` descarta a URL resolvida e monta o payload só com `--db`/`--repo` (`:71-77`). Ao vivo: o log diz *"Postgres detectado… será ignorado nesta sessão"* e duas linhas abaixo, `Database: C:\Users\Felipe\.memory\memory.db` | `cmd/mem/install.go:63,71-77` · `cmd/mem/main.go:403-422,1441-1450` |
| 4 | 🟠 Major | **`--target claude-desktop` documentado e não aceito** — `docs/CLI_GUIDE.md:631` lista o valor; `internal/autowire/detector.go:222` só casa `claude`. Execução real → *"Nenhum cliente MCP compatível encontrado"*, **exit 0, sem erro** | `internal/autowire/detector.go:222` · `docs/CLI_GUIDE.md:631` |
| 5 | 🟠 Major | **"auto-detecta ferramentas instaladas" não é implementado** — o detector devolve **todos** os candidatos (`internal/autowire/detector.go:69-158`); `Exists`/`ParentDirOK` (`:199-217`) **nunca** são consultados por `cmd/mem/install.go:109-142`; o instalador cria o diretório e escreve o arquivo de qualquer jeito (`internal/autowire/installer.go:95-101`). Execução real em diretório vazio criou `.cursor/mcp.json` e `.vscode/mcp.json` | `internal/autowire/detector.go:199-217` · `cmd/mem/install.go:109-142` · `internal/autowire/installer.go:95-101` |
| 6 | 🟡 Minor | o backup `.bak` **não é atômico** — `os.WriteFile` sem temporário + rename; a spec (G2) e a ADR-026:19 exigem atomicidade | `internal/autowire/installer.go:105-108` |
| 7 | 🟡 Minor | indentação e ordem das chaves **não são preservadas** — `json.MarshalIndent` sobre `map[string]interface{}` reordena em alfabético. O *conteúdo* foi preservado (a AC P1.2 **não** é violada); o que não se sustenta é o checkbox de T2 | `internal/autowire/installer.go:75` · `.specs/features/ai-tool-auto-wiring/tasks.md:15` |
| 8 | 🟡 Minor | a ADR-026 descreve uma **arquitetura que não existe** — registro declarativo `ClientSpec` (o código tem `if` fixos e constantes `ClientType`) e escopo **Workspace** do Windsurf (só existe o global) | `docs/adr/026-...md:31-35,42` |
| 9 | 🟡 Minor | **T4 parcial** — a linha `mem install` foi adicionada ao `docs/AGENT_INTEGRATION_GUIDE.md` por `db02542` e removida por `f5646fd`; hoje o arquivo só menciona `mem hook install` | `docs/AGENT_INTEGRATION_GUIDE.md:271` |
| 10 | 🟡 Minor | o **plano de validação anterior listava 8 testes que nunca existiram** (de 12 nomes marcados como feitos, só 4 existem) e afirmava "15 pacotes Go" (hoje são 27) | `.specs/features/ai-tool-auto-wiring/validation.md` (plano anterior) |

**Os gaps 2 e 3 são os bloqueantes desta correção.** O gap 2 é um *furo de prova*:
o comportamento certo existe na camada de domínio (`internal/autowire/detector.go:222,226`
já filtram alvo e escopo) mas **nada na fronteira CLI** — a camada que o usuário
toca — distingue "filtrou" de "não filtrou". O gap 3 é um *furo de contrato*: o
comando anuncia um engine e entrega outro, e o store entregue pode ser um SQLite
diferente do vault vivo (`cmd/mem/main.go:403-422` abre o Postgres quando
`resolvedPG != ""` e ignora o `--db`; sem Postgres, abre o SQLite do `--db`).

O padrão dos 10 gaps é o mesmo: **o checkbox e a documentação responderam pelo
comportamento**. Os 8 ACs da feature de origem têm evidência `file:line` — o que
falha não é o que foi implementado, é o que foi **prometido** e o que **não foi
provado**.

## Goals

- [ ] O verbo `setup` tem **um dono só** (o assistente de cofre central, ADR-033) e
      nenhuma documentação do auto-wiring anuncia `mem setup` como alias.
- [ ] A fronteira CLI **discrimina**: `--target`, `--workspace` e `--global` têm
      asserção do **lado negativo** — os mutantes **M7** e **M8** morrem.
- [ ] As flags de escopo **contraditórias** (`--workspace` com `--global`) falham
      com exit ≠ 0, em vez de instalar em todos os escopos.
- [ ] O payload injetado respeita a **precedência de storage** da ADR-040: quando
      o Postgres vence, o payload declara `--storage postgres` e **não** aponta
      para um `.db` que a própria CLI declarou descartar.
- [ ] O instalador **escreve só onde o cliente está instalado** e reporta o que
      pulou e por quê.
- [ ] O backup `.bak` é **atômico** e byte-idêntico ao arquivo substituído.
- [ ] A ADR-026, o CLI guide, o guia de agentes e a spec da feature descrevem
      **exatamente** o que o binário faz.
- [ ] O relatório de validação **cita apenas testes que existem**.
- [ ] O sensor de discriminação fecha em **9/9** mutantes mortas.

## Out of Scope

| Item | Reason |
| :--- | :--- |
| **Restaurar o alias `mem setup`** para o auto-wiring | `mem setup` é hoje o assistente interativo de cofre central (ADR-033), documentado por outra ADR. Restaurar o alias quebraria um comando vivo — a decisão é de **documentação** (Assumption A1). |
| Criar um alias novo para o auto-wiring (ex.: `mem install --all`) | `mem install` sem `--target` já é o "auto-detectar todas as ferramentas" — uma flag sinônima não acrescenta comportamento (Assumption A2). |
| Gravar a **URL do Postgres** no arquivo de configuração do editor | O config do workspace (`.cursor/mcp.json`, `.vscode/mcp.json`) vive **dentro do repositório** e pode ser versionado: gravar ali uma URL com senha vaza credencial no git. Assumption A4. |
| Reescrever a **ADR-026** como ADR nova (substituir/arquivar) | A decisão de arquitetura do auto-wiring continua válida; o que está errado é a **descrição** dela. Emendar é o conserto (Assumption A8, gap 8). |
| Introduzir um registro declarativo `ClientSpec` no código | O gap 8 é de **descrição**, não de arquitetura: alinhar o texto ao código real (`if` fixos + constantes `ClientType`) é o conserto mínimo — implementar um registro novo seria requisito novo. |
| Preservar a **indentação e a ordem original das chaves** do config existente | Exigiria um editor de JSON com ordem estável — escopo novo. O contrato passa a declarar a formatação canônica de 2 espaços (Assumption A8, gap 7). |
| Detectar a ferramenta por **binário/versão** (só diretório e marcador por cliente) | O gap 5 é de *ausência* do marcador de instalação; detectar o executável é feature nova. |
| Implementar a **ordenação por prioridade** de clientes ou qualquer outra feature de UX do instalador | Não está em nenhum dos 10 gaps. |
| Revalidar a feature `ai-tool-auto-wiring` **inteira** do zero | O verificador independente já validou 8/8 ACs com evidência; esta spec fecha os 10 gaps e re-verifica o que mudou (sensor + contrato). |
| Mudar o schema do banco, a tool MCP ou o transporte do servidor | A correção é de contrato de CLI, prova e texto normativo. |
| Tocar na detecção de staleness ou no packager | São as ISSUE-043 e a ISSUE-045, com specs de correção próprias. |
| Tocar no vault, em dados gravados ou no Postgres do dono | Nenhum dado muda: o auto-wiring escreve config de editor, e o teste do Postgres é resolvido por variável de ambiente, sem contato com o banco do dono. |

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| **A1** — Quem é dono do verbo `setup` | O **assistente de cofre central** (ADR-033) mantém o verbo; o alias do auto-wiring **não volta**. A correção é na **documentação** | `cmd/mem/main.go:794` já roteia `setup` para o assistente; restaurar o alias quebraria um comando documentado por outra ADR. `docs/CLI_GUIDE.md:625,650-651` param de prometer o alias (o exemplo `./bin/mem.exe setup --dry-run` dá `flag provided but not defined: -dry-run`) e a ADR-026 passa a registrar que o verbo foi repurposado em `8685ed8`. Decisão do dono. | y |
| **A2** — Alias alternativo para o auto-wiring | **Não**. `mem install` (sem flag) já é o "auto-detectar todas as ferramentas" | Uma flag sinônima (`mem install --all`) não introduz comportamento novo e aumenta a superfície pública da CLI. | y |
| **A3** — Precedência de storage no payload | O payload declara **o engine vencedor** e **nunca** leva `--db` de SQLite quando o Postgres venceu (ADR-040) | `cmd/mem/install.go:63` descarta o terceiro valor devolvido por `resolveStorageAndRepo` (`cmd/mem/main.go:1381`) — a URL resolvida. O log de storage (`cmd/mem/main.go:1439,1444-1450`) e o payload precisam dizer a mesma coisa. | y |
| **A4** — A URL do Postgres vai no payload? | **Não.** O payload leva `--storage postgres` e **não** leva a URL | A URL pode conter senha, e o config do workspace vive dentro do repositório versionado — gravá-la vaza credencial no git. Com `--storage postgres`, o `mem mcp` filho resolve a URL pela **mesma** precedência da ADR-040 e, se não resolver, **falha explicitamente** (`cmd/mem/storage_override.go:30-34`) em vez de abrir um store diferente. | y |
| **A5** — Como o editor resolve a URL do Postgres | Pela **precedência da ADR-040 no processo filho** (flag > env > config global/local), e não por uma cópia no arquivo | Se o editor não herdar `MY_MEMORY_PG_URL` **e** o config global não tiver `storage.postgres_url`, o `mem mcp` falha com mensagem explícita — **falha alto**, nunca um store silenciosamente errado. Esse é o comportamento exigido pelo gap 3. | y |
| **A6** — Conjunto de valores aceitos de `--target` | `all`, `claude`, `claude-desktop`, `cursor`, `vscode`, `windsurf` — com `claude-desktop` como **sinônimo de `claude`** | `claude-desktop` é o ID canônico na ADR-026:32 **e** o valor que `docs/CLI_GUIDE.md:631` ensina; aceitar é o default do dono. Qualquer valor **fora** do conjunto passa a falhar com exit ≠ 0 em vez de sair 0 em silêncio. **As duas fontes divergem:** a ajuda da flag (`cmd/mem/install.go:25`) anuncia só `claude`, `cursor`, `vscode`, `windsurf`, `all`; o guia (`docs/CLI_GUIDE.md:631`) ensina `claude-desktop`. O default é a **união** das duas, para que o valor **documentado** funcione sem quebrar o valor da `--help`. **Alternativa (spec paralela):** aceitar só os 5 valores e corrigir o guia para remover `claude-desktop` — inverte em uma linha nesta tabela, mas torna a documentação que o gap 4 manda corrigir errada de novo. | y |
| **A7** — Como a presença do cliente é apurada | Por um **marcador de instalação por tipo de cliente**, consultado através do `SystemEnv` já injetável (`internal/autowire/detector.go:11-16`) — **não** pelo `ParentDirOK` do alvo | O gap é sobre "ferramentas que o usuário **não tem**". O diretório de workspace (`.cursor/` dentro do repo) não é sinal de instalação: usá-lo tornaria `mem install --workspace` um no-op em todo repositório virgem — o oposto do "zero-touch" da ADR-026:82. O marcador por cliente + probe injetável preserva o zero-touch **e** deixa o teste hermético. **Alternativa (spec paralela):** usar o `Exists`/`ParentDirOK` que o detector já calcula (`internal/autowire/detector.go:199-217`) — mais simples, mas o diretório de workspace é criado pelo próprio instalador (`internal/autowire/installer.go:95-101`), então o sinal seria circular e o gap 5 não fecharia de verdade. | y |
| **A8** — Formatação do config escrito | **2 espaços, chaves de topo em ordem alfabética** — a spec é **reescrita** para dizer isso | O **conteúdo** já é preservado (a AC P1.2 da spec de origem não é violada — o gap 7 é de *checkbox*); preservar indentação e ordem exigiria um editor de JSON ordenado, escopo novo. O contrato passa a declarar a formatação canônica. | y |
| **A9** — O que o comando faz quando nenhum cliente está instalado | **exit 0**, e a saída diz explicitamente que nenhum cliente foi configurado | "Não há ferramenta instalada" não é erro de execução — mas o comando precisa **dizê-lo**, para não parecer que configurou alguma coisa. O valor inválido de `--target` é que é erro (exit ≠ 0). | y |
| **A10** — O `mem setup` mantém o comportamento atual | `mem setup` **sem** flag continua indo para o assistente de cofre central; a correção **não** muda o comportamento desse comando | O gap 1 é de contrato publicado (decisão do dono: "a correção é na documentação"). Tocar no roteamento do assistente seria requisito novo. **Alternativa (spec paralela):** `mem setup --dry-run` responder com uma mensagem que nomeie o assistente de cofre central como dono do verbo e aponte para `mem install`, em vez do erro críptico de flag desconhecida — melhora a UX sem mexer no roteamento, mas **muda comportamento de um comando vivo** e cria um artefato de teste novo (`cmd/mem/setup_owner_test.go`); inverte em uma linha nesta tabela. | y |
| **A11** — Flags de escopo contraditórias (`--workspace` com `--global`) | O comando **falha com exit ≠ 0** e diz que as duas flags são mutuamente exclusivas | Hoje `cmd/mem/install.go:41-44` resolve o escopo só quando **uma** das flags está presente: com as duas, nenhuma condição casa e o escopo fica no padrão, que instala em **todos** os escopos — exatamente a forma do mutante **M8** (filtro de escopo ignorado), que sobreviveu à suíte inteira. **Alternativa (spec paralela):** manter o comportamento atual (ignorar as duas flags) — a spec paralela mencionava o caso como edge case, sem decidir; inverte em uma linha nesta tabela. | y |

Open questions: none — todas as decisões acima têm default e rationale, e as
divergências entre as duas versões da ISSUE-048 estão registradas nas linhas A6,
A7, A10 e A11 com o **default**, o **rationale** e a **alternativa**.

---

## User Stories

### P1: O verbo `setup` tem um dono só ⭐ MVP

**User Story**: Como leitor do CLI guide, quero saber **qual comando** faz o
auto-wiring, para que eu não rode `mem setup` esperando que ele configure meus
editores — e descubra, no meio do caminho, um assistente de cofre central.

**Why P1**: é o gap 1 — 🔴 **Blocker**. O `mem setup` está documentado em
`docs/CLI_GUIDE.md:625,650-651` e `docs/adr/026-...md:1,18,27,57`, e não faz o
que a doc promete. É o único gap que promete um **comando** que não existe.

**Acceptance Criteria**:
1. The auto-wiring documentation SHALL NOT advertise `mem setup` as an alias of the auto-wiring command.
2. The ADR-026 SHALL name `mem install` as the auto-wiring command and SHALL record that the verb `setup` belongs to the central-vault assistant (ADR-033).

**Independent Test**: `grep -rn "mem setup" docs/ .specs/features/ai-tool-auto-wiring/` não
devolve nenhuma promessa de alias; a ADR-026 diz, em uma linha, quem é o dono
do verbo.

---

### P1: A fronteira CLI discrimina alvo e escopo ⭐ MVP

**User Story**: Como usuário que restringe o auto-wiring a um cliente e a um
escopo, quero que `mem install --target cursor --workspace` **realmente** toque
só o Cursor do workspace, para que eu possa confiar que o filtro foi aplicado.

**Why P1**: é o gap 2 — 🟠 Major e **bloqueante**. Os mutantes **M7**
(`cmd/mem/install.go:85`) e **M8** (`cmd/mem/install.go:39-45`) sobreviveram à
suíte inteira: `cmd/mem/install_test.go:38-83` só afirma o lado **positivo**
(nada é criado no VS Code nunca é afirmado) e **não existe** nenhum teste de
`--global`.

**Acceptance Criteria**:
3. WHEN `--target cursor` is given with `--workspace` THEN the installer SHALL NOT create `.vscode/mcp.json`.
4. WHEN `--target cursor` is given with `--workspace` THEN the command output SHALL NOT mention VS Code.
5. WHEN `--global` is given THEN the installer SHALL NOT create `.cursor/mcp.json` nor `.vscode/mcp.json` under the `--dir` root.
6. The command test suite SHALL assert the negative side of the target filter and of the scope filter.
7. IF `--workspace` and `--global` are given together THEN the command SHALL fail with a non-zero exit status.

**Independent Test**: com `--target cursor --workspace` num `--dir` temporário, o
`.vscode/mcp.json` não existe e a saída não fala de VS Code; com `--global`,
nada é criado dentro do `--dir`; com as duas flags de escopo juntas, o comando
falha em vez de instalar em todos os escopos.

---

### P1: O payload respeita a precedência de storage ⭐ MVP

**User Story**: Como usuário com o Postgres como vault central, quero que o
`mem install` injete um payload que **abra o mesmo vault** que a CLI usa, para
que meu editor não conecte num `.db` local que a própria CLI declarou
descartado.

**Why P1**: é o gap 3 — 🟠 Major e **bloqueante**. `cmd/mem/install.go:63`
descarta a URL resolvida; o log diz *"SQLite local em C:\Users\Felipe\.memory\memory.db
será ignorado nesta sessão"* e duas linhas abaixo o payload aponta para
`C:\Users\Felipe\.memory\memory.db`. Se o editor não herdar `MY_MEMORY_PG_URL`,
`mem mcp --db <esse caminho>` abre um **store diferente** do vault vivo
(`cmd/mem/main.go:403-422`).

**Acceptance Criteria**:
8. WHEN the Postgres engine wins the storage resolution THEN the injected MCP payload SHALL declare `--storage postgres`.
9. WHEN the Postgres engine wins the storage resolution THEN the injected MCP payload SHALL NOT carry a `--db` argument.
10. WHEN the SQLite engine wins the storage resolution THEN the injected MCP payload SHALL carry a `--db` argument with the resolved path.
11. WHEN the SQLite engine wins the storage resolution THEN the injected MCP payload SHALL NOT carry a `--storage postgres` argument.
12. The injected MCP payload SHALL NOT contain the Postgres connection URL.
13. The command output SHALL contain the line `Engine: postgres` when the Postgres engine wins the storage resolution.
14. The command output SHALL contain the line `Engine: sqlite` when the SQLite engine wins the storage resolution.
15. The command output SHALL NOT contain a `Database:` line naming a SQLite file when the Postgres engine wins the storage resolution.

**Independent Test**: com `MY_MEMORY_PG_URL` apontando para uma porta fechada,
`mem install --dry-run` imprime `Engine: postgres` e o payload traz
`--storage postgres`, sem `--db` e sem a URL; sem a variável, o payload traz
`--db <caminho>` e a saída imprime `Engine: sqlite`.

---

### P1: O `--target` aceito é o documentado ⭐ MVP

**User Story**: Como usuário que lê o CLI guide, quero que o valor de `--target`
que a doc ensina **funcione**, e que um valor inválido **falhe alto**, para que
eu não leia um "nenhum cliente encontrado" como se a máquina estivesse vazia.

**Why P1**: é o gap 4 — 🟠 Major. `docs/CLI_GUIDE.md:631` lista `claude-desktop`;
`internal/autowire/detector.go:222` só casa `claude`; e o erro sai com
**exit 0**, em silêncio.

**Acceptance Criteria**:
16. The accepted values of `--target` SHALL be exactly `all`, `claude`, `claude-desktop`, `cursor`, `vscode` and `windsurf`.
17. WHEN `--target claude-desktop` is given THEN the installer SHALL resolve the Claude Desktop target.
18. WHEN `--target claude-desktop` is given THEN the installer SHALL NOT resolve the Cursor, VS Code or Windsurf targets.
19. IF `--target` is given a value outside the accepted set THEN the command SHALL exit with a non-zero status.
20. IF `--target` is given a value outside the accepted set THEN the command SHALL print an error message that lists the accepted values.

**Independent Test**: `mem install --target claude-desktop --dry-run` lista o
Claude Desktop; `mem install --target xxxx --dry-run` sai com código ≠ 0 e a
mensagem lista os valores aceitos.

---

### P1: O instalador escreve só onde o cliente está instalado ⭐ MVP

**User Story**: Como usuário que tem duas ferramentas de IA, quero que o
`mem install` **não crie** configuração para ferramentas que eu não tenho, e que
me diga **o que pulou e por quê**, para que eu saiba que a máquina está
configurada — e não que o comando falhou em silêncio.

**Why P1**: é o gap 5 — 🟠 Major. O detector devolve **todos** os candidatos
(`internal/autowire/detector.go:69-158`), `Exists`/`ParentDirOK` (`:199-217`)
**nunca** são consultados por `cmd/mem/install.go:109-142`, e o instalador cria
o diretório e escreve o arquivo de qualquer jeito
(`internal/autowire/installer.go:95-101`).

**Acceptance Criteria**:
21. WHEN a detected client is not installed on the machine THEN the installer SHALL NOT create its configuration file.
22. WHEN a detected client is not installed on the machine THEN the command output SHALL report the client as skipped, naming the client and the reason.
23. WHEN a detected client is installed on the machine THEN the installer SHALL create or update its configuration file.
24. The command test SHALL fix the installation presence of each client through an injected probe, so that the result does not depend on the tools installed on the machine that runs the test.

**Independent Test**: num `--dir` temporário, com o probe Reportando o Cursor
ausente e o VS Code presente, o `.cursor/mcp.json` não é criado, a saída diz
"pulado" com o nome do Cursor, e o `.vscode/mcp.json` é criado.

---

### P2: O backup `.bak` é atômico

**User Story**: Como usuário que já tinha configuração de outros servidores MCP
no editor, quero que o `.bak` esteja **íntegro** mesmo se o processo cair no
meio, para que o backup sirva justamente no cenário em que eu precisar dele.

**Why P2**: é o gap 6 — 🟡 Minor, mas a spec de origem (G2) e a ADR-026:19 já
prometem "backup atômico"; hoje `os.WriteFile` grava direto no destino
(`internal/autowire/installer.go:105-108`) e uma queda deixa o `.bak`
**truncado** — exatamente o caso que o backup existe para evitar.

**Acceptance Criteria**:
25. WHEN the installer backs up an existing configuration file THEN the `.bak` SHALL be written to a temporary file in the same directory and moved into place with a rename.
26. The `.bak` file SHALL be byte-identical to the configuration file it replaced.

**Independent Test**: um config existente com 4 espaços de indentação e dois
servidores → o `.bak` tem os mesmos bytes, e um mutante que troca o rename por
`os.WriteFile` direto morre no teste.

---

### P2: A formatação do config escrito é declarada, não prometida

**User Story**: Como usuário que versiona a configuração do editor no meu
repositório, quero saber **exatamente** como o arquivo vai sair, para que o
diff do meu repo não fique ilegível a cada execução.

**Why P2**: é o gap 7 — 🟡 Minor. O *conteúdo* é preservado (a AC P1.2 da spec
de origem **não** é violada); o que não se sustenta é o checkbox de T2
(`.specs/features/ai-tool-auto-wiring/tasks.md:15`, "preservando indentação") e
a leitura de que a ordem original das chaves é mantida — `json.MarshalIndent`
sobre `map[string]interface{}` reordena em alfabético
(`internal/autowire/installer.go:75`).

**Acceptance Criteria**:
27. The configuration file written by the installer SHALL use 2-space indentation with the top-level keys in alphabetical order.
28. The written configuration SHALL preserve every pre-existing top-level key and every pre-existing entry of `mcpServers`.
29. The specification of the `ai-tool-auto-wiring` feature SHALL state the 2-space alphabetical format and SHALL NOT claim that the original indentation is preserved.

**Independent Test**: um config com `zetaGlobal` antes de `alphaSetting` e 4
espaços → o arquivo escrito sai com 2 espaços, `alphaSetting` primeiro, e as
**duas** chaves e os servidores pré-existentes intactos.

---

### P2: O contrato publicado descreve o código

**User Story**: Como leitor da ADR-026 e do guia de agentes, quero que a
documentação descreva **o código que existe**, para que eu não projete um
`ClientSpec` declarative num repositório que resolve cliente por `if` fixo, nem
 procure um `mem install` no guia de agentes e não encontre.

**Why P2**: são os gaps 8 e 9 — 🟡 Minor. A ADR-026:31-35 documenta um registro
declarativo `ClientSpec` que não existe (o código tem `if` fixos em
`internal/autowire/detector.go:69-158` e as constantes `ClientType` em
`internal/autowire/client.go:6-11`); a ADR-026:42 documenta o escopo
**Workspace** do Windsurf, que só existe no **global**
(`internal/autowire/detector.go:144`); e a linha `mem install` sumiu do
`docs/AGENT_INTEGRATION_GUIDE.md:271` (adicionada em `db02542`, removida em
`f5646fd`).

**Acceptance Criteria**:
30. The ADR-026 SHALL describe the client registry that exists in the code — the four `ClientType` constants and the per-client path resolution — and SHALL NOT describe a declarative `ClientSpec` registry.
31. The ADR-026 SHALL state that the Windsurf client is supported in the global scope only.
32. The `docs/AGENT_INTEGRATION_GUIDE.md` SHALL list the `mem install` command.

**Independent Test**: cada afirmação da ADR-026 aponta para um `file:linha`
que existe; o guia de agentes tem a linha `mem install`.

---

### P2: O relatório de validação diz a verdade

**User Story**: Como dono do repo, quero que o relatório de validação cite
**apenas testes que existem** e diga o **número real** de pacotes, para que o
veredito da feature seja lido como prova — e não como plano.

**Why P2**: é o gap 10 — 🟡 Minor. Dos 12 nomes marcados como feitos no plano
anterior, só **4** existem hoje; o plano ainda afirmava "Todos os 15 pacotes Go
continuam passando" (hoje são 27) e "Teste real no Windows" (não há tal
teste). O `validation.md` atual já substituiu esse plano, mas o **relatório
novo** não pode repetir o erro.

**Acceptance Criteria**:
33. The validation report of the corrected `ai-tool-auto-wiring` feature SHALL cite only test functions that exist in the repository.
34. The validation report of the corrected `ai-tool-auto-wiring` feature SHALL state that the eight test names of the earlier validation plan never existed.

**Independent Test**: cada nome de teste citado no relatório existe no código
(`grep` por `func Test…`), e o relatório nomeia os 8 testes que nunca existiram.

---

### P1: A correção é re-verificada de forma independente ⭐ MVP

**User Story**: Como dono do repo, quero que a correção destes gaps seja
**re-verificada por um Verifier independente**, com sensor de discriminação,
para que a feature saia do vermelho com prova — e não com a repetição do mesmo
procedimento que a deixou vermelha.

**Why P1**: a feature de origem está **vermelha** (`validate_state.py
ai-tool-auto-wiring` sai 1) e o veredito **FAIL** só fecha com um ciclo
fix → re-verificar. O sensor precisa provar que **M7** e **M8** — os dois
mutantes que sobreviveram — agora **morrem**.

**Acceptance Criteria**:
35. WHEN the discrimination sensor re-injects the mutations M7 and M8 THEN the test suite SHALL fail for each one of them.
36. The verification of the `ai-tool-auto-wiring` feature SHALL be produced by a Verifier distinct from the author of the correction.
37. The validation report of the corrected `ai-tool-auto-wiring` feature SHALL record a PASS verdict with `file:line` evidence for the acceptance criteria of that feature.

**Independent Test**: o relatório da feature corrigida cita `file:line` para as
8 ACs de origem, mostra o sensor em **9/9** e tem veredito **PASS**.

## Edge Cases

- IF o valor de `--target` for `claude-desktop` THEN ele é normalizado para
  `claude` — o mesmo alvo, sem duplicar cliente no relatório.
- IF o valor de `--target` for `all` THEN nenhum filtro de cliente é aplicado
  (o filtro de escopo continua valendo).
- IF `--workspace` e `--global` forem passados **juntas** THEN o comando falha
  com exit ≠ 0 — as flags de escopo são mutuamente exclusivas e a combinação
  não pode instalar em todos os escopos (hoje `cmd/mem/install.go:41-44` cai
  no escopo padrão).
- IF **nenhum** cliente estiver instalado THEN o comando sai com **exit 0** e a
  saída diz explicitamente que nada foi configurado (Assumption A9) — não é erro.
- IF o Postgres estiver resolvido **e** o usuário passar `--db` explicitamente
  THEN o `--db` explícito é honrado (a flag vence a config, ADR-040) e a saída
  diz `Engine: sqlite`.
- IF o config existente já tiver o **mesmo** servidor MCP THEN o instalador
  reporta `already_up_to_date` e não reescreve o arquivo (comportamento
  atual, preservado).
- IF o config existente tiver `mcpServers` que **não** é um objeto JSON THEN o
  valor original é substituído em silêncio (`internal/autowire/installer.go:41-49`)
  — comportamento **pré-existente**, fora dos 10 gaps: esta spec **não** o
  corrige, e o Verifier deve registrá-lo como achado remanescente.
- IF o arquivo de config existente tiver JSON inválido THEN o comando falha sem
  escrever nada (`internal/autowire/installer.go:30-36`) — comportamento já
  coberto por teste e preservado.
- IF a escrita do `.bak` falhar THEN o `.bak` **anterior** permanece intacto
  (o rename é a operação que publica o backup novo).
- IF o sensor rodar em worktree isolado THEN a árvore real fica intacta —
  baseline e final idênticos, **sem `git stash`** (o padrão do sensor anterior).

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Gap | Mutante | Where | Status |
| :--- | :--- | :--- | :--- | :--- | :--- |
| AUTOWIRE-01 | AC1 — a doc não anuncia `mem setup` como alias | gap 1 | — | `docs/CLI_GUIDE.md` | Pending |
| AUTOWIRE-02 | AC2 — a ADR-026 nomeia `mem install` e registra o dono do verbo `setup` | gap 1 | — | `docs/adr/026-auto-wiring-e-instalacao-zero-touch-de-ferramentas-de-ia.md` | Pending |
| AUTOWIRE-03 | AC3 — `--target cursor --workspace` não cria `.vscode/mcp.json` | gap 2 | M7 | `cmd/mem/install_test.go` | Pending |
| AUTOWIRE-04 | AC4 — a saída não menciona VS Code no mesmo cenário | gap 2 | M7 | `cmd/mem/install_test.go` | Pending |
| AUTOWIRE-05 | AC5 — `--global` não cria nada dentro do `--dir` | gap 2 | M8 | `cmd/mem/install_test.go` | Pending |
| AUTOWIRE-06 | AC6 — a suíte assere o lado negativo dos dois filtros | gap 2 | M7, M8 | `cmd/mem/install_test.go` | Pending |
| AUTOWIRE-07 | AC7 — `--workspace` com `--global` falha com exit ≠ 0 | gap 2 | M8 | `cmd/mem/install.go` | Pending |
| AUTOWIRE-08 | AC8 — com Postgres, o payload declara `--storage postgres` | gap 3 | — | `cmd/mem/install.go` | Pending |
| AUTOWIRE-09 | AC9 — com Postgres, o payload não leva `--db` | gap 3 | — | `cmd/mem/install.go` | Pending |
| AUTOWIRE-10 | AC10 — com SQLite, o payload leva `--db <caminho>` | gap 3 | — | `cmd/mem/install.go` | Pending |
| AUTOWIRE-11 | AC11 — com SQLite, o payload não leva `--storage postgres` | gap 3 | — | `cmd/mem/install.go` | Pending |
| AUTOWIRE-12 | AC12 — o payload não contém a URL do Postgres | gap 3 | — | `cmd/mem/install.go` | Pending |
| AUTOWIRE-13 | AC13 — a saída contém `Engine: postgres` | gap 3 | — | `cmd/mem/install.go` | Pending |
| AUTOWIRE-14 | AC14 — a saída contém `Engine: sqlite` | gap 3 | — | `cmd/mem/install.go` | Pending |
| AUTOWIRE-15 | AC15 — a saída não nomeia um `.db` de SQLite quando o Postgres venceu | gap 3 | — | `cmd/mem/install.go` | Pending |
| AUTOWIRE-16 | AC16 — o conjunto aceito de `--target` é exatamente os 6 valores | gap 4 | — | `internal/autowire/detector.go` | Pending |
| AUTOWIRE-17 | AC17 — `claude-desktop` resolve o Claude Desktop | gap 4 | — | `internal/autowire/detector.go` | Pending |
| AUTOWIRE-18 | AC18 — `claude-desktop` não resolve Cursor, VS Code nem Windsurf | gap 4 | — | `internal/autowire/detector.go` | Pending |
| AUTOWIRE-19 | AC19 — `--target` inválido sai com código ≠ 0 | gap 4 | — | `cmd/mem/install.go` | Pending |
| AUTOWIRE-20 | AC20 — a mensagem de erro lista os valores aceitos | gap 4 | — | `cmd/mem/install.go` | Pending |
| AUTOWIRE-21 | AC21 — cliente ausente não tem arquivo criado | gap 5 | — | `cmd/mem/install.go` | Pending |
| AUTOWIRE-22 | AC22 — o cliente ausente aparece na saída como pulado, com o motivo | gap 5 | — | `cmd/mem/install.go` | Pending |
| AUTOWIRE-23 | AC23 — cliente presente tem arquivo criado ou atualizado | gap 5 | — | `internal/autowire/detector.go` | Pending |
| AUTOWIRE-24 | AC24 — o teste fixa a presença por probe injetado | gap 5 | — | `cmd/mem/install_test.go` | Pending |
| AUTOWIRE-25 | AC25 — o `.bak` vai por temporário + rename | gap 6 | — | `internal/autowire/installer.go` | Pending |
| AUTOWIRE-26 | AC26 — o `.bak` é byte-idêntico ao arquivo substituído | gap 6 | — | `internal/autowire/installer_test.go` | Pending |
| AUTOWIRE-27 | AC27 — o config escrito usa 2 espaços e chaves em ordem alfabética | gap 7 | — | `internal/autowire/installer_test.go` | Pending |
| AUTOWIRE-28 | AC28 — o conteúdo pré-existente é preservado | gap 7 | — | `internal/autowire/installer_test.go` | Pending |
| AUTOWIRE-29 | AC29 — a spec da origem declara a formatação canônica | gap 7 | — | `.specs/features/ai-tool-auto-wiring/spec.md` | Pending |
| AUTOWIRE-30 | AC30 — a ADR-026 descreve o registro de clientes que existe | gap 8 | — | `docs/adr/026-auto-wiring-e-instalacao-zero-touch-de-ferramentas-de-ia.md` | Pending |
| AUTOWIRE-31 | AC31 — a ADR-026 diz que o Windsurf é só global | gap 8 | — | `docs/adr/026-auto-wiring-e-instalacao-zero-touch-de-ferramentas-de-ia.md` | Pending |
| AUTOWIRE-32 | AC32 — o guia de agentes lista o `mem install` | gap 9 | — | `docs/AGENT_INTEGRATION_GUIDE.md` | Pending |
| AUTOWIRE-33 | AC33 — o relatório cita só testes que existem | gap 10 | — | `.specs/features/ai-tool-auto-wiring/validation.md` | Pending |
| AUTOWIRE-34 | AC34 — o relatório diz que os 8 nomes do plano antigo nunca existiram | gap 10 | — | `.specs/features/ai-tool-auto-wiring/validation.md` | Pending |
| AUTOWIRE-35 | AC35 — o sensor mata M7 e M8 | gaps 2 e 3 | M7, M8 | `.specs/features/ai-tool-auto-wiring/validation.md` | Pending |
| AUTOWIRE-36 | AC36 — o Verifier é distinto do autor da correção | ciclo fix → re-verificar | — | `.specs/features/ai-tool-auto-wiring/validation.md` | Pending |
| AUTOWIRE-37 | AC37 — o relatório novo tem veredito PASS com evidência `file:line` | ciclo fix → re-verificar | — | `.specs/features/ai-tool-auto-wiring/validation.md` | Pending |

**Coverage**: 37 total, 37 mapeados, 0 unmapped.

## Success Criteria

- [ ] Os **10 gaps** da ISSUE-048 têm AC com evidência `file:line` no relatório do
      Verifier — e nenhum dos 10 fica de fora.
- [ ] Os mutantes **M7** e **M8** **morrem**: o sensor fecha em **9/9** mortas
      (os 7 antigos continuam mortos).
- [ ] `mem install --target cursor --workspace` não toca o VS Code, e
      `mem install --global` não toca o `--dir` — ambos **com teste**.
- [ ] `mem install --workspace --global` falha com exit **≠ 0**, em vez de
      instalar em todos os escopos.
- [ ] Com o Postgres resolvido, o payload declara `--storage postgres`, não leva
      `--db` e não leva a URL; a saída imprime `Engine: postgres`.
- [ ] Com o Postgres resolvido, `mem install` **não** cria config para cliente
      ausente e **diz** o que pulou.
- [ ] `grep -rn "ou \`mem setup\`" docs/` não devolve promessa de alias.
- [ ] O gate completo (`go -C D:\repository\my-memory test -count=1 ./...`, com
      Postgres) e o gate taggeado (`go -C D:\repository\my-memory test -tags
      "sqlite_fts5 onnx" -count=1 ./...`) saem **exit 0**, com `go build` e
      `go vet` verdes.
- [ ] `validate_state.py ai-tool-auto-wiring` sai **exit 0** (veredito **PASS**).
