# fix-packager-token-budget Validation Report

- **Date**: 2026-11-17
- **Spec**: `.specs/features/fix-packager-token-budget/spec.md`
- **Issue**: ISSUE-045 (`.specs/ISSUES.md`)
- **Diff range**: `6ddeeb8`..`1db797a` (commits `6ddeeb8`, `fe24922`, `3899d3e`, `f6bdda9`, `36da388`, `f465c13`, `3569255`, `ebfb255`, `1db797a`)
- **Verifier**: sub-agente independente (author != verifier).
- **Result**: PASS

## Validation: PASS

**Verdict**: PASS — 17/17 Acceptance Criteria (FPB-01 a FPB-17) verificados com evidência `file:line` e asserção discriminante. 9/9 Tasks (T1 a T9) concluídas. Sensor de Discriminação: 6/6 mutantes mortos (0 sobrevivem).

---

## Task Completion (T1..T9)

| Task | Status | Commit | Evidência / Descrição |
| ---- | ------ | ------ | --------------------- |
| **T1**: O orçamento é medido contra o bundle renderizado | ✅ Done | `6ddeeb8` | `internal/graph/pack.go:223-395`, `internal/graph/pack_test.go:157-220` (`TestPackContext_BundleRespeitaOrcamentoReal`, `TestPackContext_OrcamentoInsuficiente`). |
| **T2**: A CLI reporta o tamanho do bundle que gravou | ✅ Done | `fe24922` | `cmd/mem/pack.go:111-125`, `cmd/mem/pack_test.go:46-118` (`TestRunPackCommand_OrcamentoRealDoArquivo`). |
| **T3**: O filtro de direção tem teste discriminante | ✅ Done | `3899d3e` | `internal/graph/pack_test.go:251-303` (`TestPackContext_ConjuntoExatoPorDirecao`). Mata o mutante M3. |
| **T4**: A ordem de prioridade tem resultado definido e testado | ✅ Done | `f6bdda9` | `internal/graph/pack_test.go:305-361` (`TestPackContext_OrdemDePrioridadePorPageRank`). Mata o mutante M4. |
| **T5**: O tratamento da raiz e os cabeçalhos do bundle são asseverados | ✅ Done | `36da388` | `internal/graph/pack_test.go:363-452` (`TestPackContext_TierDaRaizECabecalhosDoBundle`, `TestPackContext_RaizPreferencialNaoEhOmitida`). Mata o mutante M2. |
| **T6**: A descrição publicada da priorização diz o que o código faz | ✅ Done | `f465c13` | `docs/adr/029-context-packager-e-subgraph-bundle.md:24`, `.specs/features/context-packager-and-subgraph-bundle/tasks.md:8`. |
| **T7**: O motor sai do estado morto | ✅ Done | `3569255` | `internal/graph/pack.go:271,277,284,375` (`nodeIDs` removido, `availableBudget` consumido). |
| **T8**: A CLI deriva os defaults da fonte única | ✅ Done | `ebfb255` | `cmd/mem/pack.go:29,61`, `cmd/mem/pack_test.go:120-160` (`TestRunPackCommand_DefaultsVemDaFonteUnica`). |
| **T9**: O MCP deriva os defaults da fonte única | ✅ Done | `1db797a` | `internal/mcp/pack_handlers.go:36`, `internal/mcp/pack_handlers_test.go:98-126` (`TestMemoryPackContextTool_DefaultsVemDaFonteUnica`). |

---

## Acceptance Criteria Verification (FPB-01 .. FPB-17)

| Requirement ID | Gap | Acceptance Criterion | Evidência (`file:line` + asserção discriminante) | Resultado |
| :--- | :-: | :--- | :--- | :--- |
| **FPB-01** | 1 | AC1 — O bundle emitido cabe em `max_tokens` (`EstimateTokens(result.Markdown) <= opts.MaxTokens`) | `internal/graph/pack.go:391-394`; `internal/graph/pack_test.go:186-188` (`real := EstimateTokens(res.Markdown); if real > opts.MaxTokens { t.Errorf(...) }`). Medido 12 vizinhos `MaxTokens=1000` -> Markdown cabe em 1000 tokens. | ✅ PASS |
| **FPB-02** | 1 | AC2 — O custo estrutural renderizado (cabeçalho, Mermaid, metadados por nó, linhas da tabela e omitidos) é cobrado junto com o conteúdo | `internal/graph/pack.go:239-242` (`renderState`), `internal/graph/pack.go:270-285` (`availableBudget := opts.MaxTokens - current`, medição incremental); `internal/graph/pack_test.go:185-188`. | ✅ PASS |
| **FPB-03** | 1 + 7 | AC3 — `PackResult.TotalTokens` reporta exatamente `EstimateTokens(PackResult.Markdown)` | `internal/graph/pack.go:384-390,405` (convergência por ponto fixo); `internal/graph/pack_test.go:189-191` (`res.TotalTokens != real`); `cmd/mem/pack_test.go:112-114` (`res.TotalTokens != graph.EstimateTokens(res.Markdown)`). | ✅ PASS |
| **FPB-04** | 1 | AC4 — Orçamento insuficiente para o pacote mínimo devolve erro identificando o mínimo exigido e não emite bundle | `internal/graph/pack.go:343-346,391-394`; `internal/graph/pack_test.go:211-219` (`TestPackContext_OrcamentoInsuficiente`, `res != nil`, erro contém "orçamento insuficiente"). | ✅ PASS |
| **FPB-05** | 2 | AC5 — `direction=both` coleta vizinhos de entrada e de saída | `internal/graph/pack.go:152-157`; `internal/graph/pack_test.go:270,298-300` (`TestPackContext_ConjuntoExatoPorDirecao/both`, conjunto exato `["root", "vizinho-entrada", "vizinho-saida"]`). | ✅ PASS |
| **FPB-06** | 2 | AC6 — `direction=outbound` contém o vizinho de saída e exclui o de entrada | `internal/graph/pack.go:152-154`; `internal/graph/pack_test.go:271,298-300` (`TestPackContext_ConjuntoExatoPorDirecao/outbound`, conjunto exato `["root", "vizinho-saida"]`). | ✅ PASS |
| **FPB-07** | 2 | AC7 — `direction=inbound` contém o vizinho de entrada e exclui o de saída | `internal/graph/pack.go:155-157`; `internal/graph/pack_test.go:272,298-300` (`TestPackContext_ConjuntoExatoPorDirecao/inbound`, conjunto exato `["root", "vizinho-entrada"]`). | ✅ PASS |
| **FPB-08** | 3 | AC8 — Ordem de prioridade definida: profundidade ascendente, PageRank descendente, ID ascendente | `internal/graph/pack.go:213-221` (`sort.Slice`); `internal/graph/pack_test.go:349-354` (`expected := []string{"root", "n2", "n3", "n4", "n1"}`, `!reflect.DeepEqual(got, expected)`). | ✅ PASS |
| **FPB-09** | 3 | AC9 — Dois nós de mesma profundidade: nó com maior PageRank vem antes em `PackResult.Nodes` | `internal/graph/pack.go:217-219`; `internal/graph/pack_test.go:357-360` (`if position["n2"] > position["n1"] { t.Errorf(...) }`). | ✅ PASS |
| **FPB-10** | 5 | AC10 — Raiz classificada como `TierCore` quando cabe no orçamento, com tratamento preferencial | `internal/graph/pack.go:264-267,289`; `internal/graph/pack_test.go:398-400` (`root.Tier != TierCore`); `internal/graph/pack_test.go:440-451` (`TestPackContext_RaizPreferencialNaoEhOmitida`). | ✅ PASS |
| **FPB-11** | 5 | AC11 — Cabeçalho `## 📄 Documentos Centrais` presente quando há nó central | `internal/graph/pack.go:464`; `internal/graph/pack_test.go:402-404` (`!strings.Contains(res.Markdown, "## 📄 Documentos Centrais")`). | ✅ PASS |
| **FPB-12** | 5 | AC12 — Cabeçalho `## 📑 Resumos Periféricos (L0/L1 Micro-Abstracts)` presente quando há nó periférico | `internal/graph/pack.go:490`; `internal/graph/pack_test.go:408-410` (`!strings.Contains(res.Markdown, "## 📑 Resumos Periféricos (L0/L1 Micro-Abstracts)")`). | ✅ PASS |
| **FPB-13** | 4 | AC13 — Descrição publicada da priorização alinhada ao código (ADR-029 e tasks.md T1 não prometem peso epistêmico) | `docs/adr/029-context-packager-e-subgraph-bundle.md:24`; `.specs/features/context-packager-and-subgraph-bundle/tasks.md:8`. Ordem real documentada e provada. | ✅ PASS |
| **FPB-14** | 6 | AC14 — MCP `memory_pack_context` usa `graph.DefaultPackOptions()` quando chamada sem opcionais | `internal/mcp/pack_handlers.go:36`; `internal/mcp/pack_handlers_test.go:113-125` (compara campo a campo com os defaults compartilhados). | ✅ PASS |
| **FPB-15** | 6 | AC15 — CLI `mem pack` sem `--max-tokens` usa `graph.DefaultPackOptions().MaxTokens` (4000) | `cmd/mem/pack.go:29,33`; `cmd/mem/pack_test.go:155-160` (compara saída com `PackContext(DefaultPackOptions())`). | ✅ PASS |
| **FPB-16** | 6 | AC16 — Nenhum estado morto no caminho de emissão (`nodeIDs` removido, `availableBudget` consumido) | `internal/graph/pack.go:271,277,284,375`. Validação de diff limpo e compilação sem alertas. | ✅ PASS |
| **FPB-17** | 7 | AC17 — CLI `mem pack --out` reporta no stdout exatamente o tamanho do arquivo gravado em disco | `cmd/mem/pack.go:120`; `cmd/mem/pack_test.go:95-97` (`reported != real`, onde `real` é `EstimateTokens(os.ReadFile(outFile))`). | ✅ PASS |

---

## Discrimination Sensor (6/6 Mutants Killed)

Executado em throwaway worktree isolado (`git worktree add`). **`git stash` nunca utilizado.** Árvore de trabalho verificada como limpa antes e depois.

| # | Mutação | Arquivo:linha | Descrição | Status | Teste Discriminante |
| - | ------- | ------------- | --------- | ------ | ------------------- |
| M1 | Orçamento burlado no TierCore | `internal/graph/pack.go:277,284` | `charged > availableBudget*4` e `<= availableBudget*4` | ✅ Morto | `TestPackContext_OmittedNodes`, `TestPackContext_BundleRespeitaOrcamentoReal` |
| M2 | Raiz perde tratamento preferencial | `internal/graph/pack.go:264` | `if false && cand.depth == 0 {` | ✅ Morto | `TestPackContext_RaizPreferencialNaoEhOmitida` (`internal/graph/pack_test.go:441`) |
| M3 | Filtro de direção removido do BFS | `internal/graph/pack.go:152,155` | Condições `opts.Direction == ...` viradas para `if true {` | ✅ Morto | `TestPackContext_ConjuntoExatoPorDirecao` (`internal/graph/pack_test.go:299`) |
| M4 | PageRank invertido na ordenação | `internal/graph/pack.go:218` | `pageRank > pageRank` virado para `<` | ✅ Morto | `TestPackContext_OrdemDePrioridadePorPageRank` (`internal/graph/pack_test.go:353,358`) |
| M5 | Tool removida do catálogo MCP | `internal/mcp/server.go:112` | Registro de `ToolMemoryPackContext` comentado | ✅ Morto | `TestMemoryPackContextTool_Registered` (`internal/mcp/pack_handlers_test.go:27`) |
| M6 | BFS nunca expande vizinhos | `internal/graph/pack.go:147` | `curr.depth >= opts.MaxDepth` virado para `if true { continue }` | ✅ Morto | `TestPackContext_Basic`, `TestPackContext_BudgetConstraintAndFallback`, `TestPackContext_OmittedNodes`, `TestPackContext_DirectionAndCycles`, `TestPackContext_ConjuntoExatoPorDirecao`, `TestPackContext_OrdemDePrioridadePorPageRank`, `TestPackContext_TierDaRaizECabecalhosDoBundle` |

---

## Gate Check

| Comando | Resultado |
| ------- | --------- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test -count=1 ./internal/graph/...` | exit 0 (11/11 testes PASS) |
| `go test -count=1 ./cmd/mem/... -run "TestRunPackCommand|TestRearrangePackArgs"` | exit 0 (5/5 testes PASS) |
| `go test -count=1 ./internal/mcp/... -run "TestMemoryPackContext"` | exit 0 (5/5 testes PASS) |

---

## Summary

- **Tasks**: 9/9 concluídas (T1 a T9).
- **Acceptance Criteria**: 17/17 verificados com asserção discriminante e evidência `file:line`.
- **Sensor**: 6/6 mutantes mortos (M1..M6).
- **Veredito**: 🟢 **PASS**.
