# Feature: fix-packager-token-budget

- **Spec ID**: `fix-packager-token-budget`
- **Created**: 2026-11-15
- **Owner**: Felipe Miiller
- **Source**: ISSUE-045 — `.specs/ISSUES.md` (seção ISSUE-045) · fecha os 7 gaps da feature `context-packager-and-subgraph-bundle` (ADR-029)
- **Verificação de origem**: `.specs/features/context-packager-and-subgraph-bundle/validation.md` — veredito **FAIL**, sensor 3/6 (M2, M3 e M4 sobreviveram)
- **Related**: ADR-029 (`docs/adr/029-context-packager-e-subgraph-bundle.md`) · `docs/CLI_GUIDE.md` §21 · spec `context-packager-and-subgraph-bundle` (R1–R5, G1–G5)

## Problem Statement

O `mem pack` promete um orçamento rígido de tokens: a ADR-029:22 exige "garantir
matematicamente que o bundle final nao exceda o limite estipulado pelo chamador", e
o `docs/CLI_GUIDE.md` §21 anuncia esse limite ao usuário. O bundle gravado
**estoura o orçamento**: medido **1370 tokens** contra um orçamento de **1000**. A
contabilidade cobra apenas o conteúdo dos nós mais um overhead fixo de 80 tokens
(`internal/graph/pack.go:220-228`) e ignora o enquadramento que o próprio
renderizador escreve — bloco Mermaid, cabeçalho de metadados, linhas de metadados
por nó e linhas da tabela de periféricos (`internal/graph/pack.go:418-497`). A
margem de segurança de cabeçalho que a R2 promete existe como variável
(`availableBudget`, `internal/graph/pack.go:221-224`) e **nunca é lida**.

O verificador independente encontrou 7 gaps na feature `context-packager-and-subgraph-bundle`
(veredito **FAIL**, sensor 3/6 — M2, M3 e M4 sobreviveram à suíte inteira). Dois
deles são violações medidas de requisitos explícitos (R1 e R2 da spec de origem) e
dois são comportamentos que **ninguém teste distingue**.

| # | Gap | Evidência (`file:linha` do `validation.md`) |
| :-: | :--- | :--- |
| 1 | o bundle emitido **estoura `max_tokens`** | medido **1370** tokens contra orçamento de 1000 — `internal/graph/pack.go:220-228` cobra só o conteúdo; o enquadramento renderizado vive em `internal/graph/pack.go:418-497`; `internal/graph/pack_test.go:114` afirma apenas o contador interno |
| 2 | o filtro de **direção** de R1 não tem teste discriminante | `internal/graph/pack_test.go:152-179` usa um ciclo de 3 nós, onde `outbound` e `both` alcançam o mesmo conjunto; o mutante **M3** (`internal/graph/pack.go:147,150`) sobrevive à suíte inteira |
| 3 | a **ordem de prioridade** não tem resultado definido no spec e não é testada | spec-precision gap **SP2**; a ordem está em `internal/graph/pack.go:208-216` e nenhum teste fixa o resultado; o mutante **M4** (`internal/graph/pack.go:213`) sobrevive |
| 4 | o **peso epistêmico** é carregado e descartado | `internal/graph/pack.go:163,402` lê só `e.Type`; `internal/db/graph.go:996,1012-1018` e `internal/store/postgres.go:1454,1473-1479` carregam `weight` e `epistemic_status`; `docs/adr/029-...md:24` promete a ponderação |
| 5 | o tratamento preferencial da **raiz** e os headers R3.3/R3.4 não são asseverados | `internal/graph/pack.go:247` (mutante **M2**), `:442` e `:468` — implementados, mas nenhum teste afirma |
| 6 | **código morto e defaults duplicados** | `internal/graph/pack.go:221-224` (`availableBudget` calculada e nunca lida), `:371,387` (`nodeIDs` escrito e nunca lido), `:28-35` (`DefaultPackOptions()` só é usada em teste) |
| 7 | a CLI reporta os **tokens contabilizados**, não o tamanho do bundle que gravou | `cmd/mem/pack.go:116` imprime `res.TotalTokens`; o arquivo gravado é `res.Markdown` (`cmd/mem/pack.go:109`) — o comando diz um número e escreve outro |

Os gaps **1, 2 e 3 são os bloqueantes**: o gap 1 viola um requisito explícito
(R2) e duas promessas publicadas (ADR-029:22 e `docs/CLI_GUIDE.md` §21); os gaps 2
e 3 são comportamentos que nenhum teste distingue — uma regressão que apague o
filtro de direção ou inverta a prioridade **não quebra nenhum teste** hoje.

## Goals

- [ ] O bundle Markdown **emitido** cabe em `max_tokens`, medido no artefato:
      `EstimateTokens(result.Markdown) <= opts.MaxTokens`.
- [ ] A contagem reportada é o **tamanho real** do bundle: `TotalTokens` ==
      `EstimateTokens(result.Markdown)`, e a CLI imprime o mesmo número que o
      arquivo gravado tem.
- [ ] O filtro de direção de R1 tem **teste discriminante** (o mutante M3 morre).
- [ ] A ordem de prioridade está **definida na spec** e tem teste que fixa o
      resultado (o mutante M4 morre).
- [ ] O tratamento preferencial da raiz e os cabeçalhos R3.3/R3.4 estão
      **asseverados** (o mutante M2 morre).
- [ ] A descrição publicada da priorização diz **o que o código faz**.
- [ ] Não sobra código morto no caminho de emissão e os defaults têm **fonte
      única**.
- [ ] O sensor de discriminação mata **6/6** mutantes.

## Out of Scope

| Item | Reason |
| :--- | :--- |
| Implementar o **peso epistêmico** na seleção | Exigiria definir uma fórmula que não existe em nenhuma ADR do repo — seria requisito novo. O default desta spec (Assumption A4) é alinhar a descrição ao código; a decisão é do dono e pode ser invertida sem mexer no resto do contrato. |
| A flag `--no-abstracts` e o alias `--max-depth` | Escopo extra apontado na seção *Code Quality* do `validation.md`, mas **fora** dos 7 gaps da ISSUE-045. Removê-los muda a superfície pública da CLI — decisão do dono. |
| **SP1** — fixar onde vive o breakdown de tiers (R3.1 da spec de origem) | Spec-precision gap fora dos 7 gaps da ISSUE-045. Registrado como Assumption A7, sem mudança de contrato. |
| Trocar a heurística de tokens (1 token ≈ 4 caracteres) | A ADR-029:37-39 fixa a heurística; o gap 1 é de **contabilidade**, não de medição. |
| Mudar o schema do banco ou o contrato da tool MCP | A correção é de contabilidade, prova e texto normativo. |
| Revalidar a feature `context-packager-and-subgraph-bundle` inteira | O verificador independente já validou tudo; esta spec fecha só os 7 gaps. |
| `README.md`, `docs/README.md` e `docs/CLI_GUIDE.md` | Nenhuma flag, comando ou formato de saída muda — a CLI continua igual, só passa a mentir menos. |

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| **A1** — Onde o orçamento é medido | No **artefato emitido**: `EstimateTokens(result.Markdown)` | A ADR-029:22 promete "garantir matematicamente que o bundle final nao exceda o limite". O contador atual mede só o conteúdo dos nós: 995 contabilizados contra 1370 reais. Medir o artefato é a única leitura que fecha a promessa. | n — padrão do Fix 1 do verificador |
| **A2** — Como o custo estrutural entra na conta | O **custo estrutural renderizado** (cabeçalho, bloco Mermaid, metadados por nó, linhas da tabela de periféricos, seção de omitidos) é cobrado junto com o conteúdo, e a alocação passa a ler o saldo disponível que ela calcula | O Fix 1 do verificador admite dois mecanismos (re-medir o Markdown renderizado ou derivar o custo das seções renderizadas). O requisito fixa o **resultado**; o mecanismo fica a cargo da implementação. | n |
| **A3** — Orçamento abaixo do bundle mínimo | `PackContext` devolve **erro** ("orçamento insuficiente") e não emite bundle | A garantia da ADR-029 é absoluta. Medido: `max_tokens=50` emite hoje um Markdown de 156 tokens (bundle mínimo ≈ título + cabeçalho ≈ 35-80 tokens). Emitir acima do limite ou erguer o limite em silêncio quebraria a promessa; o erro é a única saída honesta. As superfícies existentes (CLI e MCP) já propagam erro — não nasce nenhuma superfície nova. | n |
| **A4** — Peso epistêmico (gap 4) | **Alinhar a descrição ao código**: a ADR-029 e o `tasks.md` T1 da feature de origem passam a declarar a ordem implementada (profundidade → PageRank ↓ → ID) e deixam de afirmar que o peso epistêmico participa da seleção | Implementar o peso exigiria inventar a fórmula (requisito novo, proibido aqui) e mexeria na ordem que o gap 3 precisa fixar. O verificador confirma as duas rotas como legítimas ("either... or..."). **Decisão do dono pendente**: se ele preferir a outra rota, o default A4 inverte e o AC13 vira "o peso epistêmico participa da ordem". | **n — o dono pode inverter** |
| **A5** — Defaults duplicados (gap 6) | `graph.DefaultPackOptions()` vira a **fonte única**: a CLI e o MCP derivam as opções dela, sobrescrevendo só o que o chamador passou; o guard de defaults dentro de `PackContext` (`internal/graph/pack.go:94-102`) permanece | `DefaultPackOptions()` é exportada e hoje só é chamada por teste (`internal/graph/pack_test.go:187`, `internal/store/postgres_test.go:293`), enquanto a produção monta as opções à mão em `cmd/mem/pack.go:57-62` e `internal/mcp/pack_handlers.go:67-72` — os mesmos defaults em 4 lugares. | n |
| **A6** — `nodeIDs` morto | Removido (`internal/graph/pack.go:371,387`) | Mapa escrito e nunca lido; a remoção não muda comportamento observável. | n |
| **A7** — Onde vive o breakdown de tiers (SP1, fora dos 7 gaps) | Inalterado: o breakdown (core / fringe / omitted) continua na saída da CLI e no JSON; o Markdown do bundle mantém o cabeçalho compacto (`~tokens/max` + nós coletados) | SP1 é um spec-precision gap **fora** da ISSUE-045. Mexer no contrato do bundle além dos 7 gaps seria escopo extra. | n |
| **A8** — O número no cabeçalho do bundle | O cabeçalho passa a mostrar o **tamanho real** (`EstimateTokens(Markdown)`), e `TotalTokens` passa a ser exatamente esse número | Hoje o cabeçalho mente junto com a CLI ("~995 / 1000" num Markdown de 1370). Alinhar motor, cabeçalho e CLI é o que fecha o gap 7. | n |
| **A9** — Como o gap 7 é provado | O teste da CLI **lê o arquivo gravado**, calcula `EstimateTokens` e compara com o número impresso no stdout | O gap 7 é literalmente "a CLI diz um número e escreve outro": só a medição do arquivo escrito prova a correção. | n |

**Open questions**: none — todas as decisões acima estão registradas com default e
justificativa. A única que o dono pode inverter é a **A4** (peso epistêmico), e a
inversão não muda nenhum outro contrato desta spec.

---

## User Stories

### P1: O bundle emitido respeita o orçamento que o chamador pediu ⭐ MVP

**User Story**: Como agente que pede um pacote de contexto com `--max-tokens 1000`,
quero que o arquivo que o `mem pack` grava caiba nos 1000 tokens que pedi, para
que eu possa injetá-lo no prompt sem estourar a janela de contexto.

**Why P1**: a restrição que a ADR-029 e o `docs/CLI_GUIDE.md` §21 prometem é
violada em uso real — 1370 tokens contra 1000. É a violação medida de um
requisito explícito (R2 da spec de origem).

**Acceptance Criteria**:
1. WHEN o empacotamento emite um bundle THEN o empacotador SHALL garantir que `EstimateTokens(result.Markdown) <= opts.MaxTokens`.
2. The packer SHALL charge the rendered structural cost of the bundle — header, Mermaid block, per-node metadata lines, fringe table rows and omitted section — in addition to the content of the nodes, in every budget decision it makes.
3. The packer SHALL report `PackResult.TotalTokens` as exactly `EstimateTokens(PackResult.Markdown)`.
4. IF o orçamento pedido não comporta o bundle mínimo THEN o empacotador SHALL devolver um erro que identifique o mínimo exigido e SHALL NOT emitir bundle.

**Independent Test**: um grafo de 12 vizinhos com `MaxTokens=1000` → o Markdown
emitido mede ≤ 1000 tokens; o mesmo grafo com `MaxTokens=50` → erro, nenhum
arquivo.

---

### P1: O filtro de direção é provado pelo conjunto que ele produz ⭐ MVP

**User Story**: Como quem escolhe `--direction inbound` para entender quem depende
de uma nota, quero que o empacotador colete **exatamente** o conjunto que essa
direção promete, para que eu confie no subgrafo que recebo.

**Why P1**: R1 é explícito sobre o filtro de direção, e hoje nenhum teste o
distingue — o mutante M3 sobrevive à suíte inteira.

**Acceptance Criteria**:
5. WHEN a expansão usa `direction=both` THEN o conjunto coletado SHALL conter o vizinho alcançado só por aresta de entrada e o alcançado só por aresta de saída.
6. WHEN a expansão usa `direction=outbound` THEN o conjunto coletado SHALL conter o vizinho de saída e SHALL NOT conter o vizinho de entrada.
7. WHEN a expansão usa `direction=inbound` THEN o conjunto coletado SHALL conter o vizinho de entrada e SHALL NOT conter o vizinho de saída.

**Independent Test**: grafo com um vizinho só de entrada e um só de saída → os
três modos devolvem três conjuntos distintos e conhecidos.

---

### P1: A ordem de prioridade é um resultado definido e testado ⭐ MVP

**User Story**: Como agente que recebe um pacote com vários nós candidatos, quero
saber qual nota vem primeiro **por regra escrita**, para que o pacote seja
previsível e a seleção seja auditável.

**Why P1**: o gap 3 é duplo — o spec nunca disse qual é o resultado esperado
(SP2), então não havia valor a afirmar e o sensor não matou o mutante M4.

**Acceptance Criteria**:
8. The packer SHALL order the selected nodes by depth ascending, then PageRank descending, then node ID ascending.
9. WHEN dois nós têm a mesma profundidade e PageRanks diferentes THEN o nó com o maior PageRank SHALL aparecer antes do outro em `PackResult.Nodes`.

**Independent Test**: dois nós de mesma profundidade com PageRanks invertidos →
o de maior PageRank vem primeiro, e a ordem está escrita nesta spec.

---

### P2: O tratamento da raiz e os cabeçalhos do bundle são asseverados

**User Story**: Como quem lê o bundle gerado, quero que a raiz tenha tratamento
preferencial garantido e que as seções centrality e de periféricos existam
sempre que houver conteúdo, para que eu possa navegar por cabeçalhos previsíveis.

**Why P2**: o mutante M2 sobrevive — o tratamento especial da raiz está no código
(`internal/graph/pack.go:247`) e nenhuma asserção o fixa.

**Acceptance Criteria**:
10. WHEN o conteúdo da raiz cabe no orçamento THEN a raiz SHALL ser classificada como `TierCore`.
11. WHEN o bundle contém ao menos um nó central THEN o Markdown SHALL conter o cabeçalho `## 📄 Documentos Centrais`.
12. WHEN o bundle contém ao menos um nó periférico THEN o Markdown SHALL conter o cabeçalho `## 📑 Resumos Periféricos (L0/L1)`.

**Independent Test**: um bundle com nó central e nó periférico → os dois
cabeçalhos presentes e a raiz com tier `core`.

---

### P2: A descrição publicada da priorização diz o que o código faz

**User Story**: Como leitor da ADR-029, quero que a decisão registrada descreva a
seleção que o código realmente executa, para que a documentação não prometa um
peso epistêmico que nunca é aplicado.

**Why P2**: o gap 4 é de honestidade documental: `tasks.md` T1 e ADR-029:24
superestimam o motor, enquanto o código lê só `e.Type`
(`internal/graph/pack.go:163,402`).

**Acceptance Criteria**:
13. The published description of the prioritization — ADR-029 and the origin feature's `tasks.md` T1 — SHALL declare the implemented order (depth ascending, then PageRank descending, then node ID ascending) and SHALL NOT claim that the epistemic edge weight takes part in the selection.

**Independent Test**: ler a ADR-029 e o `tasks.md` T1 e conferir que a ordem
declarada é a que o teste da P1-3 prova.

---

### P2: Os defaults têm fonte única e o código morto sai

**User Story**: Como mantenedor do motor, quero uma única definição dos defaults de
empacotamento e nenhum estado morto no caminho de emissão, para que CLI e MCP não
possam divergir.

**Why P2**: hoje os mesmos defaults existem em 4 lugares
(`internal/graph/pack.go:28-35,94-102`, `cmd/mem/pack.go:57-62`,
`internal/mcp/pack_handlers.go:67-72`) e `availableBudget` e `nodeIDs` são
código morto.

**Acceptance Criteria**:
14. WHEN `memory_pack_context` é chamada sem os parâmetros opcionais THEN as opções passadas ao motor SHALL ser iguais a `graph.DefaultPackOptions()` campo a campo (MaxDepth 2, MaxTokens 4000, Direction `both`, IncludeFringeAbstracts true).
15. WHEN `mem pack` roda sem `--max-tokens` THEN o orçamento aplicado SHALL ser `graph.DefaultPackOptions().MaxTokens` (4000).
16. The packer SHALL NOT keep in the emission path any state that is written and never read.

**Independent Test**: chamar a tool MCP sem parâmetros opcionais e capturar as
opções no `PackFunc`; rodar a CLI sem flags e comparar com um
`graph.PackContext` usando `graph.DefaultPackOptions()`.

---

### P2: A CLI reporta o tamanho do bundle que gravou

**User Story**: Como quem roda `mem pack --out bundle.md`, quero que o número
impresso no terminal seja o tamanho do arquivo que acabei de receber, para não
decidir com base numa métrica que não corresponde ao artefato.

**Why P2**: hoje a CLI imprime `res.TotalTokens` (995) e grava um Markdown de 1370
tokens — o gap 7.

**Acceptance Criteria**:
17. WHEN `mem pack --out <arquivo>` grava o bundle THEN a linha `Orçamento:` do stdout SHALL reportar o mesmo número de tokens que `EstimateTokens` do arquivo gravado.

**Independent Test**: rodar `mem pack --out`, ler o arquivo, comparar o número
impresso com `EstimateTokens` do conteúdo.

---

## Edge Cases

- IF `max_tokens` é menor que o bundle mínimo (título + cabeçalho) THEN o
  empacotador devolve o erro de orçamento insuficiente — hoje esse caso emite um
  Markdown de **156 tokens** para um orçamento de **50** (medido no
  `validation.md`).
- IF todos os nós candidatos são omitidos THEN o bundle mantém a seção
  `## ⚠️ Nós Omitidos por Orçamento de Tokens` — comportamento já coberto por
  `internal/graph/pack_test.go:147`.
- IF o grafo contém ciclos THEN o conjunto coletado não repete o mesmo nó —
  comportamento preservado (`internal/graph/pack_test.go:176`).
- IF um nó não tem micro-abstract THEN o empacotador usa os primeiros 200
  caracteres do conteúdo (`internal/graph/pack.go:237-244`) — comportamento atual
  preservado e **fora do escopo** (não é gap da ISSUE-045).

---

## Requirement Traceability

| Requirement ID | Gap (ISSUE-045) | Acceptance Criterion | Evidência do gap (`file:linha`) | Where |
| :--- | :-: | :--- | :--- | :--- |
| FPB-01 | 1 | AC1 — o bundle emitido cabe em `max_tokens` | `internal/graph/pack.go:220-228` · `internal/graph/pack_test.go:114` | `internal/graph/pack.go` |
| FPB-02 | 1 | AC2 — o custo estrutural renderizado entra na conta | `internal/graph/pack.go:218-337` · `internal/graph/pack.go:418-497` | `internal/graph/pack.go` |
| FPB-03 | 1 + 7 | AC3 — `TotalTokens` é o tamanho real do Markdown | `internal/graph/pack.go:354` · `cmd/mem/pack.go:116` | `internal/graph/pack.go` |
| FPB-04 | 1 | AC4 — orçamento insuficiente vira erro, não bundle | `internal/graph/pack.go:100` · `validation.md` (edge case `max_tokens=50`) | `internal/graph/pack.go` |
| FPB-05 | 2 | AC5 — `direction=both` coleta entrada e saída | `internal/graph/pack.go:147,150` (M3) | `internal/graph/pack_test.go` |
| FPB-06 | 2 | AC6 — `direction=outbound` exclui a entrada | `internal/graph/pack_test.go:152-179` | `internal/graph/pack_test.go` |
| FPB-07 | 2 | AC7 — `direction=inbound` exclui a saída | `internal/graph/pack_test.go:152-179` | `internal/graph/pack_test.go` |
| FPB-08 | 3 | AC8 — a ordem de prioridade está definida | `internal/graph/pack.go:208-216` (SP2) | esta spec (AC8) · `internal/graph/pack.go` |
| FPB-09 | 3 | AC9 — o PageRank decide o desempate na ordem | `internal/graph/pack.go:213` (M4) | `internal/graph/pack_test.go` |
| FPB-10 | 5 | AC10 — a raiz é `TierCore` quando cabe | `internal/graph/pack.go:247` (M2) | `internal/graph/pack_test.go` |
| FPB-11 | 5 | AC11 — cabeçalho `## 📄 Documentos Centrais` | `internal/graph/pack.go:442` | `internal/graph/pack_test.go` |
| FPB-12 | 5 | AC12 — cabeçalho `## 📑 Resumos Periféricos (L0/L1)` | `internal/graph/pack.go:468` | `internal/graph/pack_test.go` |
| FPB-13 | 4 | AC13 — a descrição publicada diz o que o código faz | `internal/graph/pack.go:163,402` · `docs/adr/029-...md:24` | `docs/adr/029-context-packager-e-subgraph-bundle.md` · `.specs/features/context-packager-and-subgraph-bundle/tasks.md` |
| FPB-14 | 6 | AC14 — o MCP usa `graph.DefaultPackOptions()` | `internal/mcp/pack_handlers.go:67-72` | `internal/mcp/pack_handlers.go` |
| FPB-15 | 6 | AC15 — a CLI usa `graph.DefaultPackOptions()` | `cmd/mem/pack.go:57-62` · `internal/graph/pack.go:28-35` | `cmd/mem/pack.go` |
| FPB-16 | 6 | AC16 — nenhum estado morto no caminho de emissão | `internal/graph/pack.go:221-224` · `internal/graph/pack.go:371,387` | `internal/graph/pack.go` |
| FPB-17 | 7 | AC17 — a CLI reporta o tamanho do arquivo gravado | `cmd/mem/pack.go:109,116` | `cmd/mem/pack.go` |

**Coverage**: 17 total, 17 mapeados, 0 unmapped.

**Bloqueio**: FPB-01 a FPB-04 (gap 1) e FPB-05 a FPB-09 (gaps 2 e 3) são a
corrigenda que trava o resto — a ordem de execução em `tasks.md` começa por eles.

---

## Success Criteria

- [ ] O bundle emitido com `MaxTokens=1000` num grafo de 12 vizinhos mede
      **≤ 1000 tokens** no Markdown final (hoje: **1370**).
- [ ] `res.TotalTokens == EstimateTokens(res.Markdown)` em todos os casos
      (hoje: 995 contra 1370).
- [ ] A linha `Orçamento:` da CLI, o `~tokens/max` do cabeçalho do bundle e o
      `total_tokens` do JSON são o **mesmo número** — o tamanho real do arquivo
      gravado.
- [ ] `max_tokens=50` devolve erro em vez de emitir um bundle de 156 tokens.
- [ ] O sensor de discriminação mata **M2, M3 e M4** (hoje sobrevivem) —
      **6/6** mutantes mortos.
- [ ] A ADR-029 e o `tasks.md` da feature de origem não prometem mais o peso
      epistêmico das arestas, e declaram a ordem implementada.
- [ ] `availableBudget` e `nodeIDs` saem do caminho de emissão, e
      `graph.DefaultPackOptions()` é usada **em produção** pela CLI e pelo MCP.
- [ ] `go -C D:\repository\my-memory test -count=1 ./...` verde (27/27 pacotes) e
      o gate taggeado `sqlite_fts5 onnx` verde.
