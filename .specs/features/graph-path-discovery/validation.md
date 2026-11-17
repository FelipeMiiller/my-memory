# graph-path-discovery Validation

**Date**: 2026-11-17
**Spec**: `.specs/features/graph-path-discovery/spec.md`
**ADR**: `docs/adr/027-descoberta-de-rotas-e-caminho-minimo-no-grafo.md`
**Diff range**: `22de8de..101062e` (inclui feature commit `9d3e0ee` e commits de correção `7c43246..101062e` da ISSUE-046 / `fix-graph-path-resolution`)
**Verified at**: `101062e` (branch `develop`), worktree clean
**Verifier**: independent sub-agent (author != verifier, subagent `bc97865a-24fa-49f6-95ce-35135bd6bc1e`)

**Overall**: ✅ Complete — **PASS**

**Verdict**: **PASS** — todos os gaps identificados na verificação anterior (FIX-1 e FIX-3..FIX-7) foram resolvidos, as decisões de precisão S1..S7 foram documentadas na spec, todos os critérios de aceitação possuem evidências determinísticas `file:line`, e o sensor de discriminação matou 8/8 mutantes (incluindo o M7 e a mutação composta do FIX-7). A dependência externa de identidade de nós (FIX-2 / ISSUE-047) permanece formalmente documentada como pendência de produto independente.

---

## Acceptance Criteria Evidence

| Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion expression | Result |
| :--- | :--- | :--- | :--- |
| **P1.1** WHEN there is a path inside `max_depth` THEN `found: true` with an ordered node list and detailed edges (relation, epistemic status, weight) | `found=true`; ordered nodes; edges carry relation + epistemic status + weight | `internal/graph/path_test.go:85-106` (`res.Found`, `res.Hops == 2`, exact order `["A", "B", "C"]`, `res.Edges[0].From/To/Direction`); `internal/graph/path_test.go:306-365` (`TestFindPath_EdgeDetails`: asserts exact `EpistemicStatus` ∈ {EXTRACTED, INFERRED, TAG}, `Weight` ∈ {1.0, 0.6, 0.3}, `Relation` links_to / implements / depends_on); `cmd/mem/path_test.go:109-120` (`res.Found`, `res.Hops == 2`, `len(res.Nodes) == 3`, `len(res.Edges) == 2`) | ✅ PASS |
| **P1.2** WHEN there is no path THEN `found: false` with an elucidating message | `found=false` + summary `Nenhum caminho encontrado entre '<origem>' e '<destino>' dentro de <N> saltos` | `internal/graph/path_test.go:66-79` (`!res.Found`, `len(res.Nodes) == 0 && len(res.Edges) == 0`, `res.Summary == "Nenhum caminho encontrado entre 'A' e 'D' dentro de 6 saltos"`); `cmd/mem/path_test.go:275-288` (`strings.Contains(notFStr, "Status:          ❌ Não encontrado")`, `strings.Contains(notFStr, "Detalhes:        Nenhum caminho encontrado entre 'node-iso-a' e 'node-iso-b' dentro de 6 saltos")`) | ✅ PASS |
| **P1.3** WHEN source == target THEN a 0-hop path with cost 0 and distinct summary | `hops=0`, `total_cost=0`, `Summary = "Nó de origem e destino são idênticos ('<id>')"` | `internal/graph/path_test.go:18-40` (`!res.Found`, `res.Hops != 0`, `res.TotalCost != 0.0`, `len(res.Nodes) != 1`, `len(res.Edges) != 0`, `res.Summary == "Nó de origem e destino são idênticos ('auth.md')"` — mata M7); `cmd/mem/path_test.go:290-305` (`strings.Contains(identStr, "Origem e destino são o mesmo nó ('node-iso-a').")`, `Status:          ✅ Encontrado`, `Saltos:          0`) | ✅ PASS |
| **P2.1** WHEN `--undirected` THEN edges may be traversed in both directions | directed `A→C` not found; undirected `A→C` found; `--undirected` takes precedence | `internal/graph/path_test.go:190-210` (`!resDir.Found`, `resUndir.Found`, `resUndir.Hops == 2`); `cmd/mem/path.go:62-65` (`if *undirected { isDirected = false }`); `cmd/mem/path_test.go:167-172` (`if dirRes.Found { t.Errorf(...) }`, `if !undirRes.Found`, `undirRes.Hops == 2`) | ✅ PASS |
| **P2.2** WHEN a route is returned THEN each edge states its orientation | `forward` / `reverse` per edge | `internal/graph/path_test.go:210-214` (`resUndir.Edges[0].Direction != "forward"`, `resUndir.Edges[1].Direction != "reverse"`) | ✅ PASS |
| **P2.3** WHEN `--mode hops` THEN cost is unit per edge, not epistemic | 1 hop chosen; total cost `1.0`; invalid mode falls back to `epistemic` | `internal/graph/path_test.go:157-174` (`resHops.Hops != 1`, `resHops.TotalCost != 1.0`); `internal/db/graph_test.go:300` (`resHops.Hops != 1`); `cmd/mem/path_test.go:173` (`undirRes.CostMode != graph.CostModeHops`) | ✅ PASS |
| **P3.1** WHEN the catalogue is listed THEN `memory_find_path` is registered | tool present in the catalogue + schema properties | `internal/mcp/server.go:111` (`s.RegisterTool(ToolMemoryFindPath, NewMemoryFindPathHandler(nil))`); `internal/mcp/tools.go:486-520` (formal JSON Schema); `internal/mcp/path_handlers_test.go:260-293` (`tools := server.GetTools()`, `tool.Name == "memory_find_path"`, properties asserted) | ✅ PASS |
| **P3.2** WHEN the tool returns THEN Markdown with inline visual route + details table | named visual block + named table | `internal/mcp/path_handlers.go:124,138`; `internal/mcp/path_handlers_test.go:98-125` (`strings.Contains(content, "Rota Conectada")`, `strings.Contains(content, "Segmentos Percorridos")`, negative control on not found) | ✅ PASS |

---

## Goals (G1–G5) Evidence

| Goal | Spec-defined outcome | Evidence | Result |
| :--- | :--- | :--- | :--- |
| **G1** Pathfinding engine in `internal/graph/path.go` | Dijkstra + `container/heap`; `epistemic` c=1/w and `hops` c=1; directed/undirected; `max_depth`; cycle termination | `internal/graph/path.go:60-92` (heap); `:109-116` (cost modes); `:197-217` (forward/reverse); `:140-143,260-272` (depth cut); `:280-290` (cycle guard) | ✅ PASS |
| **G2** Persistence resolution | SQLite `FindPath`, Postgres `FindPath`, `Store` interface, `ResolveNodeCanonicalID` | `internal/db/graph.go:577-670,916-965`; `internal/store/postgres.go:1015-1100,1370-1420`; `internal/db/graph_resolve_test.go:14-165`; `internal/store/postgres_test.go:340-415` (multi-hop + repo isolation); `internal/store/postgres_resolve_test.go:15-175` | ✅ PASS |
| **G3** CLI `mem path <source> <target>` | `--directed`, `--undirected`, `--max-depth`, `--mode`, `--json` + ASCII route renderer | `cmd/mem/path.go:25-222`; `cmd/mem/path_test.go:16-305` | ✅ PASS |
| **G4** MCP tool registered in stdio **and** HTTP/SSE with formal JSON Schema | tool in both transports; JSON Schema; typed error codes | `internal/mcp/tools.go:486-520` (schema); `internal/mcp/server.go:111` (registration); `internal/mcp/path_handlers.go:19-158`; `internal/mcp/path_handlers_test.go:13-293` (`InvalidParams` on ambiguity and missing params, `CodeInternalError` on system failure) | ✅ PASS |
| **G5** ADR-027 + indexes/manuals | ADR in MADR + catalog + manuals + state | `docs/adr/027-descoberta-de-rotas-e-caminho-minimo-no-grafo.md`; `docs/adr/README.md:37`; `docs/README.md:54`; `docs/CLI_GUIDE.md:656`; `docs/AGENT_INTEGRATION_GUIDE.md:260,306`; `.specs/STATE.md:125` | ✅ PASS |

---

## Closure of Gaps (ISSUE-046 / `fix-graph-path-resolution`)

| Gap | Description | Resolution & Evidence | Status |
| :--- | :--- | :--- | :--- |
| **FIX-1** | M7 sobreviveu: ramo de nó idêntico redundante e sem guarda | `internal/graph/path_test.go:34-40` assere o `Summary` literal exato (`Nó de origem e destino são idênticos ('auth.md')`) e desabilita prefixo `Caminho encontrado`. M7 morre no sensor. | ✅ Closed |
| **FIX-3** | Resolução de nó ambígua e arbitrária (`LIMIT 1` sem `ORDER BY`) | SQLite (`internal/db/graph.go:650-678`) e Postgres (`internal/store/postgres.go:1080-1095`) agora coletam candidatos e aplicam desempate determinístico estável: documento do mesmo repositório → menos segmentos → id lexicográfico. Relatório tipado de ambiguidade (`internal/graph/resolve.go:30-105`) exibido na CLI (`cmd/mem/path.go:146-148`) e retornado como `InvalidParams` no MCP (`internal/mcp/path_handlers.go:80-87`). Testes: `internal/db/graph_resolve_test.go:14-165` e `internal/store/postgres_resolve_test.go:15-175`. | ✅ Closed |
| **FIX-4** | `EpistemicStatus` e `Weight` não eram asseverados nas arestas devolvidas | `internal/graph/path_test.go:306-365` (`TestFindPath_EdgeDetails`) assere para cada aresta percorrida: `EpistemicStatus` (EXTRACTED, INFERRED, TAG), `Weight` (1.0, 0.6, 0.3) e `Relation` (`links_to` normalizado, relações customizadas mantidas verbatim). | ✅ Closed |
| **FIX-5** | Caminho PostgreSQL coberto apenas por self-path trivial | `internal/store/postgres_test.go:340-415` (`TestPostgresStore_FindPath_MultiHopAndRepoIsolation`) implementa teste com dois repositórios no mesmo banco, provando rota multi-hop de 2 saltos e garantia de que nenhum nó do outro repositório vaza na rota ou no relatório de ambiguidade. | ✅ Closed |
| **FIX-6** | Mensagens de saída, headings e códigos de erro sem asserção | `internal/graph/path_test.go:74-79` assere `Summary` de não-encontrado; `cmd/mem/path_test.go:275-288` assere linha `Status: ❌ Não encontrado` e linha `Detalhes:`; `internal/mcp/path_handlers_test.go:98-125` assere headings `Rota Conectada` e `Segmentos Percorridos`; `:225-256` assere `CodeInvalidParams` para parâmetros obrigatórios ausentes. | ✅ Closed |
| **FIX-7** | `DefaultPathOptions()` morto em produção com defaults duplicados | `cmd/mem/path.go:26` e `internal/mcp/path_handlers.go:43` derivam todos os defaults de `graph.DefaultPathOptions()`. Asserções diretas adicionadas em `internal/graph/path_test.go:44-55` (`TestDefaultPathOptions`) e `cmd/mem/path_test.go:278-288`. Mutação composta morta no sensor. | ✅ Closed |
| **FIX-2** | Fenda de identidade de nós entre path ID e title ID no store real | Dependência externa mantida formalmente documentada em `.specs/features/graph-path-discovery/spec.md:57`, `.specs/features/fix-graph-path-resolution/spec.md:22,44` e rastreada via ISSUE-047 (`fix-graph-node-identity`). | 📋 Tracked (ISSUE-047) |

---

## Precision Decisions (S1–S7)

Todas as 7 lacunas de precisão foram formalmente integradas ao contrato da feature em `.specs/features/graph-path-discovery/spec.md:43-57`:
- **S1**: Resposta MCP text-only em Markdown; entrega de payload estruturado JSON formalmente diferida para a ISSUE-022 (`spec.md:56`).
- **S2**: Relações epistêmicas (`EXTRACTED`, `INFERRED`, `TAG`) ou vazias normalizam para `links_to`; relações customizadas (`implements`, `depends_on`) preservadas verbatim (`spec.md:54`, `internal/graph/path.go:193-195`).
- **S3**: `--undirected` prevalece sobre `--directed` quando ambas são fornecidas (`spec.md:51`, `cmd/mem/path.go:62-65`).
- **S4**: Valor inválido de `--mode` adota fallback seguro para `epistemic` (`spec.md:52`, `cmd/mem/path.go:67-70`, `internal/mcp/path_handlers.go:59-65`).
- **S5**: Valor não-positivo de `--max-depth` reseta para o default canônico de 6 saltos (`spec.md:53`, `cmd/mem/path.go:58-60`, `internal/mcp/path_handlers.go:44-49`).
- **S6**: Relações canônicas (`links_to`, `implements`, `depends_on`, `references_code`, `tagged_as`), status (`EXTRACTED`, `INFERRED`, `TAG`) e pesos canônicos (1.0, 0.6, 0.3) definidos formalmente (`spec.md:54`).
- **S7**: Textos literais de `Summary` padronizados para caso não encontrado e nó idêntico (`spec.md:55`).

---

## Discrimination Sensor (8/8 Killed)

Executado em worktree isolado e descartável (`git worktree add --detach D:\.minimax\scratch\subagents\verifier-gpd2\wt HEAD`). A árvore principal nunca foi mantida mutada — `git status --porcelain` verificado limpo. **`git stash` nunca foi utilizado.**

| Mutation | File:line | Description | Killed? | Evidence |
| :--- | :--- | :--- | :--- | :--- |
| **M1** | `internal/graph/path.go:140` | `maxDepth := opts.MaxDepth * 1000` (desabilita corte de profundidade) | ✅ Killed | `--- FAIL: TestFindPath_MaxDepthCutoff` (`path_test.go:234: com limite de 3 saltos não deveria alcançar 4 saltos`) |
| **M2** | `internal/graph/path.go:208` | `if false {` em vez de `if !opts.Directed {` (desabilita arestas reversas) | ✅ Killed | `--- FAIL: TestFindPath_DirectedVsUndirected` (`path_test.go:201: deveria encontrar caminho não-direcionado de A para C`) |
| **M3** | `internal/graph/path.go:115` | `cost := 1.0` em vez de `cost := 1.0 / weight` (desabilita peso epistêmico) | ✅ Killed | `--- FAIL: TestFindPath_EpistemicCostTieBreaker` (`path_test.go:143: modo epistêmico deveria escolher caminho de 2 saltos sólidos, obteve 1`) |
| **M4** | `internal/graph/path.go:110` | `return 1.0 / weight, weight, status` em `CostModeHops` (desabilita custo unitário) | ✅ Killed | `--- FAIL: TestFindPath_EpistemicCostTieBreaker` (`path_test.go:166: modo hops deveria escolher caminho direto de 1 salto, obteve 2`) |
| **M5** | `internal/mcp/server.go:111` | Remove registro de `ToolMemoryFindPath` do catálogo | ✅ Killed | `--- FAIL: TestMemoryFindPath_ServerRegistration` (`path_handlers_test.go:291: ferramenta memory_find_path não registrada no servidor`) |
| **M6** | `cmd/mem/path.go:63` | `isDirected = true` mesmo com `*undirected == true` | ✅ Killed | `--- FAIL: TestPathCLI_UndirectedAndHops` (`path_test.go:168: esperava found=true no modo undirected`) |
| **M7** | `internal/graph/path.go:167` | `if false && cleanSource == cleanTarget {` (desabilita ramo dedicado de nó idêntico) | ✅ Killed | `--- FAIL: TestFindPath_IdenticalSourceAndTarget` (`path_test.go:36: esperado Summary "Nó de origem e destino são idênticos ('auth.md')", obteve "Caminho encontrado com 0 salto(s)..."`) |
| **FIX-7 Composite** | `internal/graph/path.go:28` + `cmd/mem/path.go:26` | `DefaultPathOptions().MaxDepth = 9` e CLI força literal `6` | ✅ Killed | `--- FAIL: TestDefaultPathOptions` (`path_test.go:47: esperado MaxDepth=6, obteve 9`) e `--- FAIL: TestPathCLI_NotFoundAndIdenticalNode` (`path_test.go:282: linha 'Profundidade:' esperada na saída:`) |

**Sensor Result**: **8/8 mutações mortas** (100% de discriminação comportamental comprovada).

---

## Gate Check

- `go test -count=1 ./internal/graph/... ./internal/db/... ./internal/mcp/... ./cmd/mem/...`: **exit 0**
- `go test -count=1 ./internal/...`: **exit 0**
- `go build ./...`: **exit 0**
- `go vet ./...`: **exit 0**
- `python .agents/skills/tlc-spec-driven/scripts/validate_spec.py graph-path-discovery --root .`: **exit 0**
- `python .agents/skills/tlc-spec-driven/scripts/validate_state.py graph-path-discovery --root .`: **exit 0**

---

## Summary

- **Veredito Final**: **PASS**
- **Cobertura Ancorada na Spec**: 8/8 ACs totalmente comprovados com evidências determinísticas `file:line`.
- **Sensor de Discriminação**: 8/8 mutações mortas (M1..M7 e mutação composta FIX-7).
- **Decisões Arquiteturais**: S1..S7 formalmente documentadas e ancoradas no código.
- **Dependências**: FIX-2 formalmente delegada à ISSUE-047 (`fix-graph-node-identity`).
