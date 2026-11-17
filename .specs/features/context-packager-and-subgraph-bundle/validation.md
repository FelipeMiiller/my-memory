# context-packager-and-subgraph-bundle Validation Report

- **Date**: 2026-11-17
- **Spec**: `.specs/features/context-packager-and-subgraph-bundle/spec.md`
- **Correction Spec**: `.specs/features/fix-packager-token-budget/spec.md` (ISSUE-045)
- **Diff range**: `aa1b79c^..HEAD` — commits originais (`aa1b79c`..`6a572bd`) + correções da spec `fix-packager-token-budget` (`6ddeeb8`, `fe24922`, `3899d3e`, `f6bdda9`, `36da388`, `f465c13`, `3569255`, `ebfb255`, `1db797a`).
- **Verifier**: sub-agente independente (author != verifier).
- **Result**: PASS

## Validation: PASS

**Verdict**: PASS — 12/12 critérios com evidência `file:line` e asserção discriminante, 0 gaps, 0 spec-precision gaps remanescentes. Sensor de Discriminação: 6/6 mutantes mortos (0 sobrevivem). Gate 100% verde.

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1 | ✅ Done | Motor em `internal/graph/pack.go:28`, alocação contra o bundle renderizado, priorização por profundidade ↑, PageRank ↓ e ID ↑. Fechado por `fix-packager-token-budget`. |
| T2 | ✅ Done | Persistência SQLite (`internal/db/graph.go:989`) e PostgreSQL (`internal/store/postgres.go:1447`), interface em `internal/store/store.go:271`. |
| T3 | ✅ Done | CLI `mem pack` em `cmd/mem/pack.go:23`, relato do tamanho real do arquivo gravado, defaults da fonte única. |
| T4 | ✅ Done | Tool MCP `memory_pack_context` em `internal/mcp/pack_handlers.go:16`, catálogo em `internal/mcp/tools.go:522`, defaults da fonte única. |
| T5 | ✅ Done | ADR-029 em `docs/adr/029-context-packager-e-subgraph-bundle.md` (emendada para refletir a priorização implementada) + `docs/CLI_GUIDE.md` §21 + `docs/AGENT_INTEGRATION_GUIDE.md`. |

---

## Spec-Anchored Acceptance Criteria (12 Criteria)

| Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion | Result |
| ------------------------- | -------------------- | ----------------------- | ------ |
| R1 UBIQUITOUS — expand from canonical root up to `max_depth` respecting `direction` (`both`/`outbound`/`inbound`) | The collected node set must differ per direction (exact sets per mode) | `internal/graph/pack_test.go:255-303` (`TestPackContext_ConjuntoExatoPorDirecao`), linha 298: `!reflect.DeepEqual(got, expected)`. Mutante M3 morto. | ✅ PASS |
| R2 WHEN — peripheral content over remaining budget -> try L0/L1 micro-abstract; if still does not fit -> `omitted` | Tier becomes `fringe`, then `omitted` | `internal/graph/pack_test.go:115` (`heavyNode.Tier != TierFringe`); `internal/graph/pack_test.go:146,149` (`res.OmittedCount != 1`, `res.OmittedNodes[0] != "Child1"`). Mutante M1 morto. | ✅ PASS |
| R2 UBIQUITOUS — o total de tokens do bundle gerado nunca deve exceder `max_tokens` | `EstimateTokens(bundle) <= max_tokens` e `TotalTokens == EstimateTokens(bundle)` | `internal/graph/pack_test.go:186-191` (`TestPackContext_BundleRespeitaOrcamentoReal`, `real > opts.MaxTokens`, `res.TotalTokens != real`); `cmd/mem/pack_test.go:95-100` (`TestRunPackCommand_OrcamentoRealDoArquivo`). Medido 12 vizinhos `MaxTokens=1000` -> `real <= 1000`. | ✅ PASS |
| R3.1 — metadados e resumo do orçamento (tokens usados, nós principais, periféricos e omitidos) | The bundle carries the budget/tier summary; CLI/JSON report breakdown | `internal/graph/pack.go:449-450` (cabeçalho markdown com root, depth, `~tokens/max`, "Nós Coletados"); `cmd/mem/pack.go:116-124`; `cmd/mem/pack_test.go:82-99` (`Orçamento: ~%d / %d tokens`, contagens de nós centrais, resumidos e omitidos). | ✅ PASS |
| R3.2 — Mermaid topology block (`graph TD`) | The bundle carries a `graph TD` diagram | `internal/graph/pack_test.go:65` (`strings.Contains(res.MermaidGraph, "graph TD")`); impl `internal/graph/pack.go:391`. | ✅ PASS |
| R3.3 — section `## 📄 Documentos Centrais` | The section exists in the bundle when core nodes present | `internal/graph/pack_test.go:402` (`strings.Contains(res.Markdown, "## 📄 Documentos Centrais")`); impl `internal/graph/pack.go:464`. | ✅ PASS |
| R3.4 — section `## 📑 Resumos Periféricos (L0/L1)` | The section exists when peripheral nodes present | `internal/graph/pack_test.go:408` (`strings.Contains(res.Markdown, "## 📑 Resumos Periféricos (L0/L1 Micro-Abstracts)")`); impl `internal/graph/pack.go:490`. | ✅ PASS |
| R3.5 — section `## ⚠️ Nós Omitidos por Orçamento` | The section exists when a node is omitted | `internal/graph/pack_test.go:152` (`strings.Contains(res.Markdown, "Nós Omitidos por Orçamento")`); impl `internal/graph/pack.go:511`. | ✅ PASS |
| R4 WHEN — `mem pack <root> [--out f]` shows metrics and saves the bundle (or prints it) | Metrics on stdout, bundle in the file | `cmd/mem/pack_test.go:82-99` (`Orçamento:`, `reported == real`), `cmd/mem/pack_test.go:76-80` (`os.ReadFile(outFile)`); `cmd/mem/pack.go:112-126`. | ✅ PASS |
| R4 WHEN — `--json` returns nodes + metrics + Markdown as valid JSON | The JSON payload carries nodes, metrics and the Markdown | `cmd/mem/pack_test.go:108-117` (`json.Unmarshal`, `res.TotalTokens == graph.EstimateTokens(res.Markdown)`, `res.TotalTokens == real`); `cmd/mem/pack.go:104-108`. | ✅ PASS |
| R5 WHEN — the agent calls `memory_pack_context` -> the server returns the bundle Markdown | The Markdown is returned as the tool text | `internal/mcp/pack_handlers_test.go:92-95` (`strings.Contains(txt, "Pacote de Contexto: Raiz Teste")`); impl `internal/mcp/pack_handlers.go:73`. | ✅ PASS |
| R5 — avisos de desatualização (staleness banner), se aplicável | The stale index is warned about in the tool output | `internal/mcp/handlers.go:1034` (tool em `isQueryTool`), `:1047-1050` (checagem), `:1053-1061` (banner prepended); teste `internal/mcp/staleness_handlers_test.go:17`. | ✅ PASS |

---

## Resolution of Previous Gaps (ISSUE-045)

Os 7 gaps identificados na bateria de 2026-11-15 foram integralmente resolvidos pela spec `fix-packager-token-budget`:

1. **Gap 1 (Estouro de max_tokens, medido 1370 vs 1000)**: Resolvido por T1 (`6ddeeb8`). O custo estrutural renderizado agora é cobrado, a alocação é baseada em `availableBudget`, `TotalTokens == EstimateTokens(Markdown)`, e orçamentos insuficientes retornam erro. Testes: `internal/graph/pack_test.go:157-195`.
2. **Gap 2 (Filtro de direção sem teste discriminante, M3 sobreviveu)**: Resolvido por T3 (`3899d3e`). Grafo sem ciclo com vizinho de entrada e vizinho de saída; teste `TestPackContext_ConjuntoExatoPorDirecao` assere conjuntos exatos. Mutante M3 morto.
3. **Gap 3 (Ordem de prioridade sem teste discriminante, M4 sobreviveu)**: Resolvido por T4 (`f6bdda9`). Ordem definida normativamente (profundidade ↑, PageRank ↓, ID ↑); teste `TestPackContext_OrdemDePrioridadePorPageRank` assere ordem e posição relativa. Mutante M4 morto.
4. **Gap 4 (Peso epistêmico superestimado na doc)**: Resolvido por T6 (`f465c13`). ADR-029 (`docs/adr/029-context-packager-e-subgraph-bundle.md:24`) e `tasks.md:8` emendados para declarar a ordem real implementada sem prometer peso epistêmico.
5. **Gap 5 (Tier da raiz e cabeçalhos R3.3/R3.4 sem asserção, M2 sobreviveu)**: Resolvido por T5 (`36da388`). `TestPackContext_TierDaRaizECabecalhosDoBundle` assere `root.Tier == TierCore` e os cabeçalhos das seções; `TestPackContext_RaizPreferencialNaoEhOmitida` mata o mutante M2.
6. **Gap 6 (Código morto e defaults duplicados)**: Resolvido por T7 (`3569255`), T8 (`ebfb255`) e T9 (`1db797a`). `nodeIDs` removido, `availableBudget` consumido na alocação, `graph.DefaultPackOptions()` adotada como fonte única na CLI e no MCP.
7. **Gap 7 (CLI reporta tokens contabilizados em vez do tamanho real gravado)**: Resolvido por T2 (`fe24922`). `cmd/mem/pack.go:120` emite `EstimateTokens(written)`, validado por `cmd/mem/pack_test.go:49-118` que lê o arquivo em disco.

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

**Resultado do Sensor**: **6/6 mutantes mortos** (0 sobreviventes).

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

## Requirement Traceability Update

| Requirement | Anterior | Novo |
| ----------- | -------- | ---- |
| R1 — connected subgraph expansion respecting `direction` | ❌ Needs Fix | ✅ Verified (`internal/graph/pack_test.go:255-303`) |
| R2 (WHEN/THEN) — L0/L1 fallback then `omitted` | ✅ Verified | ✅ Verified (`internal/graph/pack_test.go:115,146`) |
| R2 UBIQUITOUS — bundle never exceeds `max_tokens` | ❌ Needs Fix | ✅ Verified (`internal/graph/pack_test.go:186-191`) |
| R3.1 — metadata / budget summary | ⚠️ Partial | ✅ Verified (`internal/graph/pack.go:449`, `cmd/mem/pack_test.go:82-99`) |
| R3.2 — Mermaid `graph TD` | ✅ Verified | ✅ Verified (`internal/graph/pack_test.go:65`) |
| R3.3 — `## 📄 Documentos Centrais` | ⚠️ Unasserted | ✅ Verified (`internal/graph/pack_test.go:402`) |
| R3.4 — `## 📑 Resumos Periféricos (L0/L1)` | ⚠️ Unasserted | ✅ Verified (`internal/graph/pack_test.go:408`) |
| R3.5 — `## ⚠️ Nós Omitidos por Orçamento` | ✅ Verified | ✅ Verified (`internal/graph/pack_test.go:152`) |
| R4 — CLI metrics, file output, `--json` | ✅ Verified | ✅ Verified (`cmd/mem/pack_test.go:49-118`) |
| R5 — MCP returns the Markdown + staleness banner | ✅ Verified | ✅ Verified (`internal/mcp/pack_handlers_test.go:92-95`, `internal/mcp/handlers.go:1034`) |
| G1 — packing engine with token estimate, weighted BFS, tiers, Mermaid | ⚠️ Partial | ✅ Verified (`internal/graph/pack.go:28`, ADR-029:24) |
| G2 — `PackContext` in SQLite and Postgres | ✅ Verified | ✅ Verified (`internal/db/graph.go:989`, `internal/store/postgres.go:1447`) |
| G3 — `mem pack` CLI | ✅ Verified | ✅ Verified (`cmd/mem/pack.go:23`) |
| G4 — `memory_pack_context` MCP tool | ✅ Verified | ✅ Verified (`internal/mcp/pack_handlers.go:16`) |
| G5 — ADR-029 and operational docs | ✅ Verified | ✅ Verified (`docs/adr/029-context-packager-e-subgraph-bundle.md`) |

---

## Summary

- **Spec-anchored check**: 12/12 critérios verificados com asserção discriminante e evidência `file:line`.
- **Sensor**: 6/6 mutantes mortos (M1..M6).
- **Gaps**: 0 gaps remanescentes. Todos os 7 gaps de ISSUE-045 fechados.
- **Veredito**: 🟢 **PASS**.
