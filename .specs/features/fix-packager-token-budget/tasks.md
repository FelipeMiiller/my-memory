# Tasks: fix-packager-token-budget

**Spec**: [spec.md](./spec.md)
**Date**: 2026-11-15
**Issue**: ISSUE-045 (`.specs/ISSUES.md`) · fecha os 7 gaps de `context-packager-and-subgraph-bundle`

## Execution Protocol (MANDATORY -- do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path. The skill is the source of truth for the full flow (per-task cycle, sub-agent delegation, adequacy review, Verifier, discrimination sensor).

**If the skill cannot be activated, STOP and tell the user - do not proceed without it.**

---

**Design**: inline nesta spec (correção de contabilidade, de prova e de texto normativo; sem mudança de schema, sem tool nova — a ADR-029 é **emendada**, não substituída)
**Status**: Draft — aguardando aprovação do dono. A Assumption **A4** (peso epistêmico: alinhar a descrição ao código) é a única decisão invertível.

**Ordem de execução**: os gaps **1, 2 e 3 são os bloqueantes** e vêm primeiro (Fase 1 e Fase 2). O gap 1 é violação medida de requisito explícito (R2) e de promessa publicada (ADR-029:22, `docs/CLI_GUIDE.md` §21); os gaps 2 e 3 são comportamentos que nenhum teste distingue.

---

## Test Coverage Matrix

> Generated from codebase, project guidelines, and spec. Guidelines found: `AGENTS.md` (regra 4 — testes obrigatórios; regra 8 — quality gate), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md`, `.github/workflows/ci.yml` (comandos de teste reais do projeto).

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Motor de empacotamento — contabilidade de orçamento | unit | 1:1 com FPB-01..FPB-04: o bundle emitido cabe em `max_tokens` medido no Markdown final; o custo estrutural renderizado é cobrado; `TotalTokens` é o tamanho real; orçamento insuficiente vira erro | `internal/graph/pack_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestPackContext ./internal/graph/...` |
| Motor de empacotamento — comportamento selecionável | unit | 1:1 com FPB-05..FPB-12: conjunto exato por direção (não contagem); ordem por profundidade, PageRank e ID (não só o conjunto); tier da raiz (não só as contagens); cabeçalhos R3.3 e R3.4 | `internal/graph/pack_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestPackContext ./internal/graph/...` |
| Superfície CLI — relato do tamanho real | integration | FPB-03 e FPB-17: o `Orçamento:` impresso no stdout é igual ao `EstimateTokens` do arquivo gravado, e o `--json` traz o mesmo número | `cmd/mem/pack_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestRunPackCommand ./cmd/mem/...` |
| Superfície CLI — defaults compartilhados | integration | FPB-15: sem `--max-tokens`, o bundle sai com `graph.DefaultPackOptions().MaxTokens` e o resultado é idêntico ao de um `PackContext` com os defaults compartilhados | `cmd/mem/pack_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestRunPackCommand ./cmd/mem/...` |
| Superfície MCP — defaults compartilhados | integration | FPB-14: as opções que o handler passa ao motor são iguais a `graph.DefaultPackOptions()` campo a campo quando os parâmetros opcionais não são enviados | `internal/mcp/pack_handlers_test.go` | `go -C D:\repository\my-memory test -count=1 -run TestMemoryPackContext ./internal/mcp/...` |
| Documentação normativa (ADR-029 e `tasks.md` da feature de origem) | none | FPB-13: a descrição publicada declara a ordem implementada e não promete o peso epistêmico — verificação por revisão do diff e checagem de desvio (`mem drift`) | `docs/adr/029-context-packager-e-subgraph-bundle.md` | `go -C D:\repository\my-memory run . drift` (checagem de desvio) |
| Higiene de código no caminho de emissão | none | FPB-16: nenhuma variável computada-e-não-lida (`availableBudget`) ou escrita-e-nunca-lida (`nodeIDs`) no caminho de emissão — verificação por revisão do diff, `go build` e `go vet` | `internal/graph/pack.go` | `go -C D:\repository\my-memory build ./...` e `go -C D:\repository\my-memory vet ./...` |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após cada tarefa de código | `go -C D:\repository\my-memory build ./...` e `go -C D:\repository\my-memory vet ./...` |
| Quick | Após as tarefas do motor (T1, T3, T4, T5, T7) | `go -C D:\repository\my-memory test -count=1 -run TestPackContext ./internal/graph/...` |
| Quick | Após as tarefas de CLI (T2, T8) | `go -C D:\repository\my-memory test -count=1 -run TestRunPackCommand ./cmd/mem/...` |
| Quick | Após a tarefa de MCP (T9) | `go -C D:\repository\my-memory test -count=1 -run TestMemoryPackContext ./internal/mcp/...` |
| Full | Gate final (T9) | `go -C D:\repository\my-memory test -count=1 ./...` (com `MY_MEMORY_PG_TEST=1` e `MY_MEMORY_PG_URL` preservada — 27/27 pacotes verdes) |
| Full (tagged) | Gate final (T9) | `go -C D:\repository\my-memory test -tags "sqlite_fts5 onnx" -count=1 ./...` |
| Format | Antes de declarar pronto | `gofmt -l` na cópia LF dos arquivos `.go` tocados (o working tree local mistura CRLF) |
| Spec | Antes de entregar | `python3 .agents/skills/tlc-spec-driven/scripts/validate_spec.py fix-packager-token-budget --root D:\repository\my-memory` e `python3 .agents/skills/tlc-spec-driven/scripts/validate_tasks.py fix-packager-token-budget --root D:\repository\my-memory` |
| State | Antes de entregar | `python3 .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-packager-token-budget --root D:\repository\my-memory` |

---

## Execution Plan

Phases are ordered and run sequentially - each phase completes before the next begins, and tasks within a phase execute in order.

### Phase 1: O orçamento real do bundle (gap 1 — BLOQUEANTE)

```
T1 -> T2
```

### Phase 2: Os comportamentos sem prova (gaps 2 e 3 — BLOQUEANTES)

```
T2 -> T3 -> T4
```

### Phase 3: Asserções e honestidade da descrição (gaps 5 e 4)

```
T4 -> T5 -> T6
```

### Phase 4: A higiene do motor (gap 6)

```
T6 -> T7 -> T8 -> T9
```

---

## Task Breakdown

### T1: O orçamento é medido contra o bundle renderizado

**What**: a alocação do motor passa a ser conferida contra o **artefato emitido**: o custo estrutural que o renderizador escreve (cabeçalho, bloco Mermaid, metadados por nó, linhas da tabela de periféricos, seção de omitidos) é cobrado junto com o conteúdo dos nós; o saldo disponível (`availableBudget`) passa a ser lido em vez de recalculado contra `opts.MaxTokens`; `TotalTokens` passa a ser exatamente `EstimateTokens(result.Markdown)`; e um orçamento que não comporta o bundle mínimo devolve erro em vez de emitir bundle.
**Where**: `internal/graph/pack.go` (motor) · `internal/graph/pack_test.go` (teste co-localizado)
**Depends on**: None
**Reuses**: `graph.EstimateTokens` (`internal/graph/pack.go:79-90`) · `renderMarkdownBundle` (`internal/graph/pack.go:418-497`) · o headerOverhead de `internal/graph/pack.go:220`
**Requirement**: FPB-01, FPB-02, FPB-03, FPB-04

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] A alocação lê o saldo disponível que calcula — não há mais comparação direta com `opts.MaxTokens` no caminho de alocação
- [ ] O custo estrutural renderizado entra na conta do orçamento
- [ ] `PackResult.TotalTokens` é exatamente `EstimateTokens(result.Markdown)`
- [ ] `max_tokens=50` devolve erro e **nenhum** bundle é emitido
- [ ] Teste novo `TestPackContext_BundleRespeitaOrcamentoReal`: grafo de 12 vizinhos com `MaxTokens=1000` → `EstimateTokens(res.Markdown) <= 1000` e `res.TotalTokens == EstimateTokens(res.Markdown)` (hoje o valor real é 1370)
- [ ] Teste novo `TestPackContext_OrcamentoInsuficiente`: `MaxTokens=50` → erro e nenhum bundle
- [ ] Gate check passa: `go -C D:\repository\my-memory build ./...`, `go -C D:\repository\my-memory vet ./...` e a suíte de `./internal/graph/...`
- [ ] Nenhum teste existente foi removido ou afrouxado

**Tests**: unit
**Gate**: quick

**Commit**: `fix(pack): mede o orcamento contra o bundle renderizado`

---

### T2: A CLI reporta o tamanho do bundle que gravou

**What**: a linha `Orçamento:` que o `mem pack --out` imprime passa a reportar o tamanho real do Markdown gravado (o mesmo número de `TotalTokens`), e o `--json` continua emitindo o mesmo valor — o comando para de dizer um número e escrever outro.
**Where**: `cmd/mem/pack.go` (CLI) · `cmd/mem/pack_test.go` (teste co-localizado)
**Depends on**: T1
**Reuses**: `res.TotalTokens` e `res.MaxTokens` de `graph.PackResult` · `graph.EstimateTokens` · o writer injetável `runPackCommand` (`cmd/mem/pack.go:23`)
**Requirement**: FPB-03, FPB-17

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] A linha `Orçamento:` do stdout mostra o tamanho real do bundle emitido, não o contador interno
- [ ] O número impresso nunca ultrapassa `max_tokens`
- [ ] O `--json` emite `total_tokens` igual ao tamanho real do Markdown
- [ ] Teste novo `TestRunPackCommand_OrcamentoRealDoArquivo`: roda `mem pack --out`, **lê o arquivo gravado**, compara `EstimateTokens(arquivo)` com o número impresso no stdout
- [ ] Gate check passa: `go -C D:\repository\my-memory build ./...`, `go -C D:\repository\my-memory vet ./...` e a suíte de `./cmd/mem/...`
- [ ] Nenhum teste existente foi removido ou afrouxado

**Tests**: integration
**Gate**: quick

**Commit**: `fix(cli): o mem pack reporta o tamanho real do bundle`

---

### T3: O filtro de direção tem teste discriminante

**What**: um grafo de teste com um vizinho alcançado **só** por aresta de entrada e outro alcançado **só** por aresta de saída (sem ciclo), com asserção sobre o **conjunto exato** de nós coletados em cada um dos três modos de direção.
**Where**: `internal/graph/pack_test.go`
**Depends on**: T2
**Reuses**: o grafo de teste já existente `TestPackContext_DirectionAndCycles` (`internal/graph/pack_test.go:152-179`) — que **continua** no lugar para cobrir o caso de ciclo
**Requirement**: FPB-05, FPB-06, FPB-07

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] O grafo de teste tem um vizinho de entrada e um de saída, alcançáveis só pelo caminho indicado
- [ ] `direction=both` → conjunto exato {raiz, vizinho de entrada, vizinho de saída}
- [ ] `direction=outbound` → conjunto exato {raiz, vizinho de saída}
- [ ] `direction=inbound` → conjunto exato {raiz, vizinho de entrada}
- [ ] A asserção é sobre o **conjunto**, nunca sobre a contagem
- [ ] O teste **falha** quando o filtro de direção é removido (mutante M3, `internal/graph/pack.go:147,150`)
- [ ] Gate check passa: `go -C D:\repository\my-memory test -count=1 -run TestPackContext ./internal/graph/...`

**Tests**: unit
**Gate**: quick

**Commit**: `test(pack): fixa o conjunto exato de cada direcao`

---

### T4: A ordem de prioridade tem resultado definido e testado

**What**: o resultado esperado da priorização passa a ser asserido: dois nós de **mesma profundidade** com PageRanks invertidos, em que o de maior PageRank tem de aparecer **primeiro** em `PackResult.Nodes`. A ordem (profundidade ↑, PageRank ↓, ID ↑) já está fixada na spec (FPB-08) — esta task fecha a prova.
**Where**: `internal/graph/pack_test.go`
**Depends on**: T3
**Reuses**: `PackOptions.PageRanks` (`internal/graph/pack.go:24`) — hoje nenhum teste do repo o preenche
**Requirement**: FPB-08, FPB-09

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] O teste monta dois nós de mesma profundidade com `PageRanks` invertidos
- [ ] A asserção é sobre a **posição** dos nós em `res.Nodes`, não sobre o conjunto
- [ ] O teste **falha** quando a comparação de PageRank é invertida (mutante M4, `internal/graph/pack.go:213`)
- [ ] A ordem asserted bate com a ordem escrita em FPB-08
- [ ] Gate check passa: `go -C D:\repository\my-memory test -count=1 -run TestPackContext ./internal/graph/...`

**Tests**: unit
**Gate**: quick

**Commit**: `test(pack): fixa a ordem de prioridade por pagerank`

---

### T5: O tratamento da raiz e os cabeçalhos do bundle são asseverados

**What**: três asserções que faltavam — o **tier** da raiz (não as contagens agregadas), a presença do cabeçalho `## 📄 Documentos Centrais` e a presença do cabeçalho `## 📑 Resumos Periféricos (L0/L1)` quando há nós centrais e periféricos.
**Where**: `internal/graph/pack_test.go`
**Depends on**: T4
**Reuses**: os bundles já montados pelos testes existentes (`TestPackContext_Basic`, `TestPackContext_BudgetConstraintAndFallback`)
**Requirement**: FPB-10, FPB-11, FPB-12

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] A asserção fixa o `Tier` da **raiz** (não o `CoreCount` agregado)
- [ ] O bundle com nó central contém `## 📄 Documentos Centrais`
- [ ] O bundle com nó periférico contém `## 📑 Resumos Periféricos (L0/L1)`
- [ ] O teste **falha** quando o tratamento preferencial da raiz é removido (mutante M2, `internal/graph/pack.go:247`)
- [ ] Gate check passa: `go -C D:\repository\my-memory test -count=1 -run TestPackContext ./internal/graph/...`

**Tests**: unit
**Gate**: quick

**Commit**: `test(pack): fixa o tier da raiz e os cabecalhos do bundle`

---

### T6: A descrição publicada da priorização diz o que o código faz

**What**: a ADR-029 passa a declarar a ordem que o motor aplica (profundidade ↑, PageRank ↓, ID ↑), deixa de afirmar que o peso epistêmico das arestas participa da seleção e aponta o teste que prova a ordem; o `tasks.md` T1 da feature de origem deixa de afirmar "pesos epistémicos de arestas".
**Where**: `docs/adr/029-context-packager-e-subgraph-bundle.md` (ADR) · `.specs/features/context-packager-and-subgraph-bundle/tasks.md` (afirmação T1 da origem)
**Depends on**: T5
**Reuses**: a ordem já implementada em `internal/graph/pack.go:208-216` e provada pelo teste da T4
**Requirement**: FPB-13

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] A ADR-029 declara a ordem implementada e **não** afirma que o peso epistêmico entra na seleção
- [ ] A ADR-029 remete ao teste que prova a ordem (o da T4)
- [ ] O `tasks.md` T1 da feature de origem não afirma mais o uso de pesos epistémicos
- [ ] `gofmt -l` não aponta nada e a checagem de desvio (`mem drift`) não acusa o ADR como desatualizado
- [ ] Nenhum outro arquivo da feature de origem é tocado

**Tests**: none (documentação normativa — a verificação é a revisão do diff e a checagem de desvio, não um teste de código)
**Gate**: quick

**Commit**: `docs(adr): a ADR-029 descreve a ordem que o codigo aplica`

---

### T7: O motor sai do estado morto

**What**: o mapa `nodeIDs`, escrito e nunca lido no construtor do diagrama Mermaid, sai do caminho de emissão; e a variável de saldo disponível passa a ser lida pela alocação (nada mais computado-e-não-lido na emissão).
**Where**: `internal/graph/pack.go`
**Depends on**: T6
**Reuses**: — (remoção pura, sem comportamento novo)
**Requirement**: FPB-16

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] `nodeIDs` (`internal/graph/pack.go:371,387`) saiu do construtor do diagrama
- [ ] Nenhuma variável do caminho de emissão é calculada e nunca lida
- [ ] O diagrama Mermaid gerado é **idêntico** ao de antes da remoção
- [ ] Gate check passa: `go -C D:\repository\my-memory build ./...`, `go -C D:\repository\my-memory vet ./...` e a suíte de `./internal/graph/...`

**Tests**: none (higiene de código — a verificação é a revisão do diff mais `go build` e `go vet`)
**Gate**: quick

**Commit**: `refactor(pack): remove o estado morto do caminho de emissao`

---

### T8: A CLI deriva os defaults da fonte única

**What**: as opções que a CLI monta passam a partir de `graph.DefaultPackOptions()`, com os flags sobrescrevendo só o que o usuário passou — os mesmos defaults deixam de existir em dois lugares.
**Where**: `cmd/mem/pack.go` (CLI) · `cmd/mem/pack_test.go` (teste co-localizado)
**Depends on**: T7
**Reuses**: `graph.DefaultPackOptions()` (`internal/graph/pack.go:28-35`) — hoje exportada e usada só em teste
**Requirement**: FPB-15

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] As opções da CLI partem de `graph.DefaultPackOptions()`
- [ ] Sem `--max-tokens`, o bundle sai com `MaxTokens = 4000` (o default compartilhado)
- [ ] Teste novo: a saída da CLI sem flags é idêntica à de um `graph.PackContext` com `graph.DefaultPackOptions()` sobre o mesmo grafo
- [ ] Gate check passa: `go -C D:\repository\my-memory build ./...`, `go -C D:\repository\my-memory vet ./...` e a suíte de `./cmd/mem/...`

**Tests**: integration
**Gate**: quick

**Commit**: `refactor(cli): os defaults do mem pack vem da fonte unica`

---

### T9: O MCP deriva os defaults da fonte única

**What**: o handler de `memory_pack_context` passa a montar as opções a partir de `graph.DefaultPackOptions()`, sobrescrevendo só os parâmetros que o agente enviou — o `MaxDepth=2` / `MaxTokens=4000` / `Direction=both` duplicados no handler saem de cena.
**Where**: `internal/mcp/pack_handlers.go` (handler) · `internal/mcp/pack_handlers_test.go` (teste co-localizado)
**Depends on**: T8
**Reuses**: `graph.DefaultPackOptions()` (`internal/graph/pack.go:28-35`) · o `PackFunc` fake já usado em `internal/mcp/pack_handlers_test.go:32,47`
**Requirement**: FPB-14

**Tools**:
- MCP: NONE
- Skill: NONE

**Done when**:

- [ ] As opções do handler partem de `graph.DefaultPackOptions()`
- [ ] Teste novo: chamada **sem** `max_depth`, `max_tokens` e `direction` → as opções capturadas pelo `PackFunc` fake são iguais a `graph.DefaultPackOptions()` campo a campo (MaxDepth 2, MaxTokens 4000, Direction `both`, IncludeFringeAbstracts true)
- [ ] `memory_pack_context` continua registrada no catálogo e continua devolvendo o Markdown do bundle
- [ ] Gate check passa: `go -C D:\repository\my-memory build ./...`, `go -C D:\repository\my-memory vet ./...` e a suíte de `./internal/mcp/...`
- [ ] Gate full antes de fechar a feature: `go -C D:\repository\my-memory test -count=1 ./...` (com Postgres) e `go -C D:\repository\my-memory test -tags "sqlite_fts5 onnx" -count=1 ./...`

**Tests**: integration
**Gate**: full

**Commit**: `refactor(mcp): os defaults do pack context vem da fonte unica`

---

## Phase Execution Map

```
Phase 1 -> Phase 2 -> Phase 3 -> Phase 4

Phase 1:  T1 -> T2
Phase 2:  T2 -> T3 -> T4
Phase 3:  T4 -> T5 -> T6
Phase 4:  T6 -> T7 -> T8 -> T9
```

Execution is strictly sequential - there is no intra-phase parallelism. A single agent (or batch worker) works one task at a time, in order.

**Contagem de tarefas**: 9 — cabem em um único lote (≤ ~8 por worker), então a execução é **inline**, sem sub-agente.

---

## Task Granularity Check

| Task | Scope | Status |
| ---- | ----- | ------ |
| T1: orçamento contra o bundle renderizado | 1 arquivo de motor + 1 teste co-localizado | ✅ Granular |
| T2: CLI reporta o tamanho real | 1 arquivo de CLI + 1 teste co-localizado | ✅ Granular |
| T3: conjunto exato por direção | 1 arquivo de teste | ✅ Granular |
| T4: ordem de prioridade | 1 arquivo de teste | ✅ Granular |
| T5: tier da raiz e cabeçalhos | 1 arquivo de teste | ✅ Granular |
| T6: descrição publicada | 2 documentos normativos | ✅ Granular (coesos: a mesma afirmação em 2 lugares) |
| T7: estado morto | 1 arquivo | ✅ Granular |
| T8: defaults da CLI | 1 arquivo de CLI + 1 teste co-localizado | ✅ Granular |
| T9: defaults do MCP | 1 arquivo de handler + 1 teste co-localizado | ✅ Granular |

---

## Diagram-Definition Cross-Check

| Task | Depends On (task body) | Diagram Shows | Status |
| ---- | ---------------------- | ------------- | ------ |
| T1 | None | (início da fase 1) | ✅ Match |
| T2 | T1 | T1 -> T2 | ✅ Match |
| T3 | T2 | T2 -> T3 | ✅ Match |
| T4 | T3 | T3 -> T4 | ✅ Match |
| T5 | T4 | T4 -> T5 | ✅ Match |
| T6 | T5 | T5 -> T6 | ✅ Match |
| T7 | T6 | T6 -> T7 | ✅ Match |
| T8 | T7 | T7 -> T8 | ✅ Match |
| T9 | T8 | T8 -> T9 | ✅ Match |

Nenhuma dependência aponta para uma fase posterior.

---

## Test Co-location Validation

| Task | Code Layer Created/Modified | Matrix Requires | Task Says | Status |
| ---- | --------------------------- | --------------- | --------- | ------ |
| T1 | Motor de empacotamento — contabilidade de orçamento | unit | unit | ✅ OK |
| T2 | Superfície CLI — relato do tamanho real | integration | integration | ✅ OK |
| T3 | Motor de empacotamento — comportamento selecionável | unit | unit | ✅ OK |
| T4 | Motor de empacotamento — comportamento selecionável | unit | unit | ✅ OK |
| T5 | Motor de empacotamento — comportamento selecionável | unit | unit | ✅ OK |
| T6 | Documentação normativa (ADR-029 e `tasks.md` da origem) | none | none | ✅ OK (a verificação é a revisão do diff e a checagem de desvio) |
| T7 | Higiene de código no caminho de emissão | none | none | ✅ OK (a verificação é a revisão do diff mais `go build` e `go vet`) |
| T8 | Superfície CLI — defaults compartilhados | integration | integration | ✅ OK |
| T9 | Superfície MCP — defaults compartilhados | integration | integration | ✅ OK |

Nenhuma task de código ficou com `Tests: none`: as duas tarefas com `none` são
camadas que a matriz declara `none` (documentação normativa e higiene).
