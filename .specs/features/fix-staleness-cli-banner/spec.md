# Feature: fix-staleness-cli-banner

- **Spec ID**: `fix-staleness-cli-banner`
- **Created**: 2026-11-16
- **Owner**: Felipe Miiller
- **Source**: ISSUE-043 — `.specs/ISSUES.md` (seção ISSUE-043) · fecha os 4 gaps da feature `staleness-detection-and-banners` (ADR-028)
- **Related**: `.specs/features/staleness-detection-and-banners/validation.md` (veredito **FAIL** — 4/9 ACs com evidência, 5 gaps, 1 spec-precision gap; sensor 6/6 mutantes mortos) · spec de origem `.specs/features/staleness-detection-and-banners/spec.md` · ADR-028 (`docs/adr/028-staleness-banners-e-deteccao-de-desatualizacao.md`)
- **Consolidação (2026-11-16)**: esta é a **versão final única** da ISSUE-043. Ela absorve a spec paralela `.specs/features/fix-staleness-banner-wiring/` (15 ACs / 12 tasks, escrita na madrugada de 2026-11-16), que cobria a **mesma** issue com a decisão **oposta** sobre o literal canônico. O que era **material** na versão paralela foi incorporado aqui (o contrato de string vazia do formatador, a nota do checkbox mentiroso, o edge case do cache em sequência); o que só fazia sentido sob a decisão contrária — **emendar a ADR-028 e traduzir o banner MCP para PT-BR** — **não** foi copiado e está registrado como **alternativa invertível** na tabela de Assumptions. A spec paralela é removida pelo dono; as duas não podem ser implementadas juntas.

## Problem Statement

A feature `staleness-detection-and-banners` (ADR-028) entregou um **motor de detecção sólido** — o sensor de discriminação matou **6/6 mutantes** — e mesmo assim o veredito independente é **FAIL**: **4/9 ACs** com evidência `file:line` e **5 gaps**. O feature entregou o motor e **deixou de fora o gancho** que o torna visível: o usuário não é avisado de que o índice está desatualizado.

| Gap | Sev. | O que falta | Onde (`file:linha`) |
| :--- | :--- | :--- | :--- |
| **P3-1** | 🔴 **Blocker** | o aviso **não sai** de `mem search` / `mem path` / `mem inspect` / `mem impact`; `FormatCLIBanner` é **código morto** | `internal/staleness/detector.go:309` · `cmd/mem/status.go:75-90` (única construção do detector na CLI) |
| **P1-1** | 🟠 Major | o literal emitido (`> ⚠️ **AVISO: Memória Desatualizada (Stale Data)**`) **não** é o do spec (`spec.md:48`) **nem** o da ADR-028 Accepted (`docs/adr/028-...md:45`); a documentação foi reescrita para acompanhar o código | `internal/staleness/detector.go:302` · `docs/AGENT_INTEGRATION_GUIDE.md:397` |
| **P2-2** | 🟠 Major | a listagem de modificados/novos/deletados **não tem asserção**, e "novo" **não é distinguível** de "modificado" no modelo (`StalenessReport` só tem `StaleFiles` e `DeletedFiles`) | `internal/staleness/detector.go:25-34,227,245` · `cmd/mem/status.go:133` |
| **P2-3 / P2-4** | 🟡 Minor | literais de status divergentes do spec (`⚠️ DESATUALIZADO (Stale)` / `✅ ATUALIZADO (Fresh)` vs `⚠️  Desatualizado (Stale Data)` / `✅ Atualizado (In Sync)`) | `cmd/mem/status.go:124,129` |

Dois fatos agravam o quadro:

1. **O `tasks.md` mente.** A task T3 da feature de origem está marcada como `[x] Integrar alerta de staleness nos comandos de consulta (search, path, inspect, impact)` — e ela **não existe em código**. `FormatCLIBanner` só tem a definição e dois usos em teste (`internal/staleness/detector_test.go:86,141`); nenhum comando de consulta constrói um `staleness.Detector`.
2. **A documentação foi reescrita para acompanhar o bug.** Em vez de corrigir o código, `docs/AGENT_INTEGRATION_GUIDE.md:397` passou a documentar o literal **acidental** — o que faz o gap parecer resolvido para quem lê a documentação. O mesmo aconteceu em `docs/CLI_GUIDE.md:1434-1435`, que publica o status `⚠️  Desatualizado (Stale Data)` e a linha `Diferenças: 8 modificado(s)/novo(s), 0 removido(s)`.

Esta spec fecha os 4 gaps e **decide** as duas escolhas que a ISSUE-043 deixa abertas (o literal canônico e o status de "novo"), para que a ADR-028, a spec de origem, o código, os testes e os guias voltem a falar a mesma língua.

## Goals

- [ ] `mem search`, `mem path`, `mem inspect` e `mem impact` imprimem o aviso de desatualização no stdout **antes** do primeiro resultado quando o índice está stale — e **não** imprimem nada quando está in sync.
- [ ] Existe **um** literal canônico de banner e de status, declarado de forma idêntica no código, na ADR-028, na spec de origem e nos guias.
- [ ] Os dois formatadores devolvem **string vazia** quando o relatório é nulo ou o índice está in sync — nenhum banner sai por acidente.
- [ ] "Novo" é uma **categoria própria** no `StalenessReport` e na listagem do `mem status`, com asserção de teste sobre o output humano.
- [ ] O `tasks.md` da feature de origem deixa de mentir: passa a citar a evidência `file:line` do gancho que existir.
- [ ] Nenhum comando de consulta **quebra** por causa do banner: se a checagem falhar, o comando degrada em silêncio.
- [ ] A saída `--json` dos comandos de consulta continua sendo **JSON válido**.

## Out of Scope

| Item | Reason |
| :--- | :--- |
| Mover o motor de detecção, o cache ou o TTL | O motor é **sólido**: 6/6 mutantes mortos. Esta spec não toca em `computeStaleness`, no cache de 3 s nem na persistência dos metadados. |
| **Emendar a ADR-028** | A ADR-028 já contém o literal canônico (`docs/adr/028-...md:45`). Reescrever a norma para acomodar o código foi **exatamente** o caminho que produziu o gap P1-1 — a ADR é a fonte normativa e o código é que se move. |
| Traduzir o banner MCP para PT-BR | Exigiria emendar a ADR-028 — ver a alternativa invertível na tabela de Assumptions. |
| Mexer na **injeção** do banner MCP | Ela já funciona e é testada (`internal/mcp/handlers.go:1047`, `internal/mcp/staleness_handlers_test.go:94`). O que muda é o **literal** que ela carrega. |
| Criar ADR nova | A decisão de literal **não** é uma decisão arquitetural nova: é a aplicação do ADR-028 Accepted. |
| Auto-reindexar em background | Fora do escopo do ADR-028 original (reindexar durante a query adicionaria latência de segundos). |
| Reindexar, reescrever ou tocar em vault | Nenhum dado gravado muda — a correção é de gancho, literal e listagem. |
| Fechar a ISSUE-044 / ISSUE-045 / ISSUE-046 / ISSUE-048 | Issues separadas, com specs de correção próprias. |
| Touchar a feature `staleness-detection-and-banners` além do `tasks.md` e do `validation.md` | A feature é a **origem** dos gaps; esta spec só fecha o que ela deixou aberto. O `spec.md` de origem é histórico e não muda. |

---

## Assumptions & Open Questions

| Assumption / decisão | Chosen default | Rationale | Confirmado? |
| :--- | :--- | :--- | :--- |
| **Qual é o literal canônico do banner MCP** | o da **ADR-028 §3** (`docs/adr/028-...md:45`), idêntico ao do spec de origem (`spec.md:48`): `` > ⚠️ **Staleness Warning**: O índice de memória está desatualizado em relação aos arquivos no disco (N arquivo(s) pendente(s): X modificado(s), Y novo(s), Z deletado(s)). As respostas abaixo podem refletir um estado anterior. Execute `mem index` ou inicie `mem watch`. `` | A ISSUE-043 abre duas rotas: (a) alinhar o código ao ADR; (b) emendar o ADR com a tradução PT-BR. O briefing decide (a): **a ADR é a fonte normativa** e reescrever a doc para acompanhar o código foi o que produziu o gap. O código se move; a ADR não. | y — **decidido** |
| **Qual é o literal canônico do banner CLI** | uma linha: `` ⚠️ [Aviso] O índice local está desatualizado (X modificado(s), Y novo(s), Z removido(s)). Execute `mem index` ou inicie `mem watch`. `` | A ADR-028 especifica **apenas** o banner Markdown (§3); o formato CLI não tem norma. O default adota a **mesma recomendação** e as **mesmas três contagens** que a ADR já promete (`:45`), mantendo a forma de linha única que cabe no topo de um stdout com resultados. | y — **decidido** |
| **Os literais de status** | o **código** passa a seguir o spec: `⚠️ DESATUALIZADO (Stale)` e `✅ ATUALIZADO (Fresh)` | O spec de origem é a fonte normativa desse AC (`spec.md:57-58`); aqui a divergência é do código (`cmd/mem/status.go:124,129`), não do contrato. `docs/CLI_GUIDE.md:1434` é reescrito junto. | y — **decidido** |
| **"Novo" é categoria própria ou sai do spec** | **categoria própria**: `NewFiles []string` + `NewFilesCount int` no `StalenessReport` e heading próprio no `mem status` | O detector **já distingue** os dois casos internamente (`internal/staleness/detector.go:227` = arquivo no disco ausente do índice; `:245` = arquivo modificado); o que falta é **expor** a distinção. O spec (`:56`) e a ADR (`:51`) pedem as três categorias — o padrão é **implementar a categoria**, não remover o requisito do contrato. | y — **decidido** |
| **O que é `N` no banner** | o **total pendente**: `N = X + Y + Z` (modificados + novos + deletados) | A ADR-028 (`:45`) escreve `N arquivo(s) pendente(s): X modificado(s), Y novo(s), Z deletado(s)` — `N` é a soma das três categorias. | y — **decidido** |
| **Onde vive o literal** | **fonte única** em `internal/staleness`: o título e a recomendação do banner são **constantes exportadas**, e o `mem status` usa literais de status **exportados** do mesmo pacote | O requirement "um literal canônico" só é verificável se houver um só lugar que o produz. A alternativa (cada constante escrita no seu consumidor) é o que produziu os 4 literais divergentes. | y — **decidido** |
| **Como o literal é asseverado sem teste vazio** | o teste **fixa o texto literal** (string hard-coded no arquivo de teste) **e** compara com a constante exportada | Asserir a constante contra si mesma é verde por construção. O teste do repo precisa do texto literal gravado — é o que quebra quando alguém troca o aviso. | y — **decidido** |
| **O que os formatadores devolvem fora do estado stale** | **string vazia** — relatório `nil` ou com `IsStale` falso produz banner vazio, nos dois formatadores | É o contrato que impede o banner de vazar num vault em sincronia (`FormatCLIBanner`, `internal/staleness/detector.go:309`) e é o comportamento que o helper da CLI consome. Sem ele, a suíte só prova o caminho "stale" e deixa o caminho "in sync" sem asserção. | y — **decidido** |
| **Qual é a raiz do vault usada na checagem** | o **diretório de trabalho atual** (o detector resolve `""` → `os.Getwd()`, `internal/staleness/detector.go:55-62`) — o mesmo default do `mem status` | Nenhum comando de consulta tem flag de vault; introduzir uma flag nova seria **inventar superfície de CLI**. | y — **decidido** |
| **Qual checagem a CLI usa** | `CheckStaleness` (cache TTL 3 s), **nunca** `ForceCheck` | `mem status` é o comando de **auditoria** e por isso usa `ForceCheck` (`cmd/mem/status.go:92`). Numa consulta, o custo de I/O tem de ser "insignificante em consultas em lote e sucessivas" (G2 da spec de origem). | y — **decidido** |
| **O que acontece quando a checagem falha** | **degrada em silêncio**: sem banner, sem erro novo, o comando segue e devolve o **mesmo código de saída** | Um aviso de staleness não pode ser a razão de uma consulta falhar. O `mem status` **é** o comando que reporta erro de checagem (`cmd/mem/status.go:92-95`); os comandos de consulta não são. | y — **decidido** |
| **O banner sai em qual saída** | **só no output humano**; com `--json` o stdout continua sendo **só** o documento JSON | `mem path --json` (`cmd/mem/path.go:107-111`), `mem inspect --json` e `mem impact --json` (`cmd/mem/impact.go:83-86`) emitem um documento JSON que integradores fazem parse; injetar o banner quebraria o contrato. `mem search` **não tem** flag `--json` (`cmd/mem/main.go:2622-2644`), então não há decisão a tomar lá. | y — **decidido** |
| **Onde o banner é impresso no stdout** | **antes da primeira linha de resultado** do comando | É o que a ISSUE-043 e o Fix Plan 1 do Verifier pedem ("no topo do stdout **antes** dos resultados"). A posição exata em relação ao cabeçalho do comando (ex.: `🔎 Buscando no SQLite por: …`) não é normativa. | y — **decidido** |
| **A ADR-028 muda?** | **não** — ela fica **intocada** por esta correção | Ela já declara o literal canônico e as três categorias; emendá-la seria trocar a norma pelo comportamento, o caminho que gerou o P1-1. | y — **decidido** |
| **Unificar o banner MCP em PT-BR exigiria emendar a ADR-028** | **não** — a ADR-028 §3 é a fonte normativa e **fica intocada**; o código se move para o literal dela | A ADR-028 Accepted já declara o literal canônico (`docs/adr/028-...md:45`, conferido no HEAD); emendá-la seria trocar a norma pelo comportamento — exatamente o caminho que produziu o gap P1-1. **Alternativa invertível pelo dono (uma linha nesta tabela + três pontos de código):** emendar a ADR-028 e trocar o literal do banner MCP por `> ⚠️ **AVISO: Memória Desatualizada (Stale Data)**` (PT-BR), o que exige reescrever a AC9, a T2 e o teste MCP (`internal/mcp/staleness_handlers_test.go:94`) — esse é o ramo da spec paralela `.specs/features/fix-staleness-banner-wiring/`, hoje absorvida aqui. | y — **decidido** (inverter é 1 linha nesta tabela + 3 pontos citados) |

**Open questions**: none — as duas escolhas que a ISSUE-043 deixava abertas (literal canônico e status de "novo") estão decididas acima, com default e rationale. A **alternativa invertível** do literal do banner MCP está registrada na última linha da tabela, com os três pontos que a inversão toca.

---

## User Stories

### P1: O aviso de desatualização aparece nos comandos de consulta ⭐ MVP

**User Story**: Como usuário de terminal, quero que `mem search`, `mem path`,
`mem inspect` e `mem impact` me avisem que o índice está desatualizado, para que
eu não decida com base em dados velhos — sem precisar rodar `mem status` antes.

**Why P1**: é o gap **P3-1**, o único **Blocker** — o motor existe, o aviso não.
Hoje `FormatCLIBanner` (`internal/staleness/detector.go:309`) é código morto e o
`tasks.md` da feature de origem marca a integração como feita.

**Acceptance Criteria**:
1. WHEN `mem search` runs against a vault whose index is stale THEN the command SHALL print the staleness banner to stdout before the first result line.
2. WHEN `mem path` runs against a vault whose index is stale THEN the command SHALL print the staleness banner to stdout before the first result line.
3. WHEN `mem inspect` runs against a vault whose index is stale THEN the command SHALL print the staleness banner to stdout before the first result line.
4. WHEN `mem impact` runs against a vault whose index is stale THEN the command SHALL print the staleness banner to stdout before the first result line.
5. WHILE the vault index is in sync THEN the four query commands SHALL NOT print the staleness banner.
6. IF the staleness check returns an error THEN the query command SHALL print its normal output without the banner and keep the exit code it would have returned without the check.
7. WHEN a query command obtains its staleness report THEN it SHALL obtain the report through the detector's cached check with a TTL of 3 seconds instead of forcing a fresh disk scan.
8. WHEN `mem path`, `mem inspect` or `mem impact` run with the `--json` flag THEN the stdout SHALL contain only the JSON document, without the banner text.

**Independent Test**: um vault de teste com um arquivo editado e não reindexado →
`mem search "termo"` imprime o banner **antes** da linha do primeiro resultado; o
mesmo vault reindexado → nenhum banner.

---

### P1: O banner e o status têm um literal canônico ⭐ MVP

**User Story**: Como agente que consome o MCP e como usuário que lê a CLI, quero
que o aviso de desatualização seja **sempre o mesmo texto**, para que eu possa
reconhecê-lo (e o sistema possa testá-lo) sem depender de documentação que pode
divergir.

**Why P1**: é o gap **P1-1** (Major) + **P2-3/P2-4** (Minor): hoje o spec, a
ADR-028, o código e os guias dizem quatro coisas diferentes, e o teste MCP
assevera o literal **acidental** (`internal/mcp/staleness_handlers_test.go:94`).

**Acceptance Criteria**:
9. WHEN the MCP tools inject the staleness banner THEN the injected text SHALL be the canonical line `` > ⚠️ **Staleness Warning**: O índice de memória está desatualizado em relação aos arquivos no disco (N arquivo(s) pendente(s): X modificado(s), Y novo(s), Z deletado(s)). As respostas abaixo podem refletir um estado anterior. Execute `mem index` ou inicie `mem watch`. `` with N, X, Y and Z replaced by the counts of the report.
10. WHEN the CLI formats the staleness banner THEN the formatted text SHALL be the canonical line `` ⚠️ [Aviso] O índice local está desatualizado (X modificado(s), Y novo(s), Z removido(s)). Execute `mem index` ou inicie `mem watch`. `` with X, Y and Z replaced by the counts of the report.
11. WHEN the banner formatters receive a nil report or a report whose `IsStale` is false THEN the formatters SHALL return an empty string.
12. WHEN `mem status` reports a stale vault THEN the status line SHALL contain the literal `⚠️ DESATUALIZADO (Stale)`.
13. WHEN `mem status` reports a vault in sync THEN the status line SHALL contain the literal `✅ ATUALIZADO (Fresh)`.
14. The published banner sample in `docs/AGENT_INTEGRATION_GUIDE.md` SHALL show the canonical MCP banner line and no longer contain the retired literal `> ⚠️ **AVISO: Memória Desatualizada (Stale Data)**`.
15. The published `mem status` sample in `docs/CLI_GUIDE.md` SHALL show the canonical status literals and no longer contain the retired literals `⚠️  Desatualizado (Stale Data)` and `✅ Atualizado (In Sync)`.
16. The ADR-028 document SHALL remain unchanged by this correction.

**Independent Test**: o teste do banner MCP, o teste do formatador e os dois testes
do `mem status` asseveram os literais das AC9 a AC13 — trocar o literal no código
quebra o teste; e um formatador chamado com relatório nulo devolve string vazia.

---

### P2: "Novo" é uma categoria própria

**User Story**: Como usuário de terminal, quero ver **separados** os arquivos
novos, modificados e deletados, para que eu saiba se preciso indexar ou se o
vault encolheu.

**Why P2**: é o gap **P2-2** (Major): hoje "novo" e "modificado" caem na mesma
lista (`Arquivos Modificados ou Não-Indexados`, `cmd/mem/status.go:133`), e a
listagem **não tem asserção de teste** nenhuma.

**Acceptance Criteria**:
17. WHEN the staleness report is produced THEN a file present on disk and absent from the index SHALL appear only in `NewFiles`.
18. WHEN the staleness report is produced THEN a file whose current mtime is newer than the persisted mtime SHALL appear only in `StaleFiles`.
19. WHEN the staleness report is produced THEN the counts of the three categories SHALL be reported separately as `StaleFilesCount`, `NewFilesCount` and `DeletedFilesCount`.
20. WHEN any of the three categories is non-empty THEN the report SHALL mark the vault as stale.
21. WHEN `mem status` renders the human-readable report THEN the output SHALL contain one heading per category — modified, new and deleted — each carrying its own count.
22. WHEN `mem status` lists the files of a category THEN the listed paths SHALL be relative to the vault root and sorted in ascending lexicographic order.
23. WHEN a category holds more than 10 files THEN `mem status` SHALL print the first 10 in order followed by the line `  ... e mais N arquivo(s)`.
24. WHEN `mem status` runs with the `--json` flag THEN the JSON document SHALL carry the `new_files` and `new_files_count` fields with the same values shown in the human-readable listing.

**Independent Test**: um vault com 1 arquivo novo, 1 modificado e 1 deletado → o
`mem status --json` traz `new_files` com 1 caminho e o output humano traz **três**
headings distintos.

---

### P2: A correção fecha a lacuna de processo

**User Story**: Como dono do repo, quero que o `tasks.md` da feature de origem
deixe de marcar como feita uma integração que não existe, para que o checkbox
não mascare o próximo gap.

**Why P2**: o `tasks.md` da feature de origem marca `[x] Integrar alerta de
staleness nos comandos de consulta` — foi esse checkbox que escondeu o Blocker
da verificação.

**Acceptance Criteria**:
25. WHEN the correction is delivered THEN the `tasks.md` of the feature `staleness-detection-and-banners` SHALL cite the `file:line` evidence of the banner call in the query commands.
26. WHEN the correction is delivered THEN the ISSUE-043 entry in `.specs/ISSUES.md` SHALL no longer be marked as open.

**Independent Test**: abrir o `tasks.md` da feature de origem e ver a referência
`file:line` do ponto onde o banner é emitido; abrir a ISSUE-043 e ver o status
de fechado com a evidência da re-verificação.

---

### P1: A correção é re-verificada de forma independente ⭐ MVP

**User Story**: Como dono do repo, quero que a correção seja **re-verificada por um
Verifier independente**, com sensor de discriminação, para que a feature saia do
FAIL com prova — e não com a repetição do mesmo procedimento que a deixou
vermelha.

**Why P1**: o ciclo fix → re-verificar é o que fecha a ISSUE-043: um PASS precisa
de evidência `file:line` e de um sensor que mate o mutante do gancho.

**Acceptance Criteria**:
27. WHEN the discrimination sensor re-injects the mutations at the four gap sites THEN the test suite SHALL fail for each of them.
28. The verification of the `staleness-detection-and-banners` feature SHALL be produced by a Verifier distinct from the author of this correction.

**Independent Test**: o relatório da feature corrigida cita `file:line` para as
9 ACs de origem e o sensor mata os mutantes dos quatro gap sites.

## Edge Cases

- IF uma consulta **não** encontra resultado THEN o banner, quando houver, sai
  mesmo assim — o aviso é sobre o índice, não sobre o resultado.
- IF a checagem de staleness **falhar** (sem base configurada, Postgres fora)
  THEN o comando imprime o output normal, sem banner, e **não** quebra.
- IF o vault tem **só** arquivos novos THEN o relatório é stale — a categoria
  "novo" conta como dessincronização (não pode haver regressão para "in sync" por
  causa da separação das listas).
- IF uma categoria da listagem tem **mais de 10** arquivos THEN saem os 10
  primeiros e a linha `... e mais N arquivo(s)` (o truncamento já existe no
  código em `cmd/mem/status.go:141-143` e `:156-158`).
- IF o índice está **in sync** THEN nenhum comando imprime banner — e a
  checagem **não** varre o disco de novo dentro do TTL de 3 s.
- IF o **mesmo** comando roda em sequência THEN a segunda chamada **reaproveita**
  o relatório do cache e não varre o disco de novo — é o que distingue
  `CheckStaleness` de `ForceCheck` no caminho de I/O.
- IF o comando roda com `--json` THEN o stdout é **só** JSON: o banner é
  suprimido, porque quebraria o parse de quem consome a saída.
- IF o diretório do vault **sumir** entre duas checagens dentro do TTL THEN a
  segunda checagem vem do cache e o banner **continua** saindo — é o que
  distingue `CheckStaleness` de `ForceCheck`.
- IF um arquivo **novo** for indexado antes da próxima checagem THEN ele sai de
  "novo" e passa a ser "modificado" na checagem seguinte.

---

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Gap | Where | Status |
| :--- | :--- | :--- | :--- | :--- |
| FSC-01 | AC1 — o banner sai antes do 1º resultado no `mem search` | P3-1 | `cmd/mem/main.go` | Pending |
| FSC-02 | AC2 — o banner sai antes do 1º resultado no `mem path` | P3-1 | `cmd/mem/path.go` | Pending |
| FSC-03 | AC3 — o banner sai antes do 1º resultado no `mem inspect` | P3-1 | `cmd/mem/inspect.go` | Pending |
| FSC-04 | AC4 — o banner sai antes do 1º resultado no `mem impact` | P3-1 | `cmd/mem/impact.go` | Pending |
| FSC-05 | AC5 — nenhum banner quando o índice está in sync | P3-1 | `cmd/mem/staleness_banner.go` | Pending |
| FSC-06 | AC6 — a checagem que falha degrada em silêncio | P3-1 | `cmd/mem/staleness_banner.go` | Pending |
| FSC-07 | AC7 — a checagem vem do cache (TTL 3 s), sem varredura forçada | P3-1 | `cmd/mem/staleness_banner.go` | Pending |
| FSC-08 | AC8 — com `--json` o stdout é só o documento JSON | P3-1 | `cmd/mem/path.go` · `cmd/mem/inspect.go` · `cmd/mem/impact.go` | Pending |
| FSC-09 | AC9 — literal canônico do banner MCP (ADR-028 §3) | P1-1 | `internal/staleness/detector.go` | Pending |
| FSC-10 | AC10 — literal canônico do banner CLI | P1-1 | `internal/staleness/detector.go` | Pending |
| FSC-11 | AC11 — os formatadores devolvem string vazia fora do estado stale | P3-1 | `internal/staleness/detector.go` | Pending |
| FSC-12 | AC12 — status `⚠️ DESATUALIZADO (Stale)` | P2-3 | `cmd/mem/status.go` | Pending |
| FSC-13 | AC13 — status `✅ ATUALIZADO (Fresh)` | P2-4 | `cmd/mem/status.go` | Pending |
| FSC-14 | AC14 — o guia do agente publica o banner canônico | P1-1 | `docs/AGENT_INTEGRATION_GUIDE.md` | Pending |
| FSC-15 | AC15 — o guia da CLI publica os status canônicos | P2-3, P2-4 | `docs/CLI_GUIDE.md` | Pending |
| FSC-16 | AC16 — a ADR-028 fica intocada | P1-1 | `docs/adr/028-staleness-banners-e-deteccao-de-desatualizacao.md` | Pending |
| FSC-17 | AC17 — arquivo ausente do índice vai para `NewFiles` | P2-2 | `internal/staleness/detector.go` | Pending |
| FSC-18 | AC18 — arquivo modificado vai para `StaleFiles` | P2-2 | `internal/staleness/detector.go` | Pending |
| FSC-19 | AC19 — as três contagens são reportadas separadamente | P2-2 | `internal/staleness/detector.go` | Pending |
| FSC-20 | AC20 — qualquer categoria não vazia marca o vault como stale | P2-2 | `internal/staleness/detector.go` | Pending |
| FSC-21 | AC21 — três headings na listagem, um por categoria | P2-2 | `cmd/mem/status.go` | Pending |
| FSC-22 | AC22 — caminho relativo e ordenação estável | P2-2 | `cmd/mem/status.go` | Pending |
| FSC-23 | AC23 — truncamento em 10 + `... e mais N` | P2-2 | `cmd/mem/status.go` | Pending |
| FSC-24 | AC24 — o `--json` traz `new_files` e `new_files_count` | P2-2 | `cmd/mem/status.go` | Pending |
| FSC-25 | AC25 — o `tasks.md` de origem cita a evidência do gancho | P3-1 | `.specs/features/staleness-detection-and-banners/tasks.md` | Pending |
| FSC-26 | AC26 — a ISSUE-043 deixa de estar `open` | processo | `.specs/ISSUES.md` | Pending |
| FSC-27 | AC27 — o sensor mata os mutantes dos quatro gap sites | ciclo fix → re-verificar | `.specs/features/staleness-detection-and-banners/validation.md` | Pending |
| FSC-28 | AC28 — o Verifier é distinto do autor da correção | ciclo fix → re-verificar | `.specs/features/staleness-detection-and-banners/validation.md` | Pending |

**Coverage**: 28 total, 28 mapeados, 0 unmapped.

**Bloqueio**: FSC-01 a FSC-08 (gap P3-1) e FSC-09/FSC-10 (gap P1-1) são a
corrigenda que trava o resto — a ordem de execução em `tasks.md` começa por eles.

---

## Success Criteria

- [ ] `mem search`, `mem path`, `mem inspect` e `mem impact` imprimem o banner de
      desatualização **antes** do primeiro resultado num vault stale, e **nada**
      num vault in sync.
- [ ] O literal do banner e o do status são **idênticos** no código, na ADR-028,
      na spec `staleness-detection-and-banners`, no `docs/CLI_GUIDE.md` e no
      `docs/AGENT_INTEGRATION_GUIDE.md` — e a ADR-028 **não** foi emendada.
- [ ] Os dois formatadores devolvem **string vazia** para relatório nulo ou in
      sync — nenhum banner vaza para fora do estado stale.
- [ ] O `StalenessReport` distingue **novo** de **modificado**, e a listagem do
      `mem status` tem **três** headings com asserção de teste.
- [ ] A feature `staleness-detection-and-banners` sai de **FAIL** com as **9**
      ACs de origem com evidência `file:line`, e `validate_state.py
      staleness-detection-and-banners` sai **exit 0**.
- [ ] O sensor de discriminação mata os mutantes dos **quatro** gap sites (além
      dos 6 mutantes que já morriam).
- [ ] O gate completo (`go -C D:\repository\my-memory test -count=1 ./...`, com
      Postgres) e o gate taggeado (`go -C D:\repository\my-memory test -tags
      "sqlite_fts5 onnx" -count=1 ./...`) saem **exit 0**, com `go build` e
      `go vet` verdes.
