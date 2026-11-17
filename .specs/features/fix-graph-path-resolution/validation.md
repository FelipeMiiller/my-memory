# fix-graph-path-resolution Validation

**Date**: 2026-11-17
**Spec**: `.specs/features/fix-graph-path-resolution/spec.md`
**Tasks**: `.specs/features/fix-graph-path-resolution/tasks.md`
**Issue**: ISSUE-046 (`.specs/ISSUES.md`)
**Diff range**: `7c43246..101062e` (commits T1..T13 da spec de correção)
**Verified at**: `101062e` (branch `develop`), worktree clean
**Verifier**: independent sub-agent (author != verifier, subagent `bc97865a-24fa-49f6-95ce-35135bd6bc1e`)

**Overall**: ✅ Complete — **PASS**

**Verdict**: **PASS** — todas as 47 condições de aceitação (GPR-01 a GPR-47) foram completamente verificadas com evidências `file:line`, os testes cobrem todas as camadas de contrato, persistência (SQLite e PostgreSQL com isolamento por repositório), CLI e MCP, e o sensor de discriminação matou 8/8 mutações.

---

## Acceptance Criteria Evidence (47/47)

| Criterion | Requirement ID | Spec-defined outcome | `file:line` + assertion expression | Result |
| :--- | :--- | :--- | :--- | :--- |
| **AC1** | GPR-01 | Mesma consulta, mesmo id canônico retornado | `internal/db/graph_resolve_test.go:118-132` (`TestResolveNodeCanonicalID_SameIdentifierResolvesSameID`); `internal/store/postgres_resolve_test.go:120-134` | ✅ PASS |
| **AC2** | GPR-02 | Resolução devolve candidatos ordenados junto com id canônico | `internal/graph/resolve.go:30-58` (`AmbiguityReport`); `internal/graph/resolve_test.go:20-55` (`res.Ambiguity != nil`, `len(res.Ambiguity.Candidates) == 2`) | ✅ PASS |
| **AC3** | GPR-03 | Preferência pelo nó que é documento do mesmo repositório | `internal/db/graph.go:650-668` (ordenação com prioridade de documento); `internal/db/graph_resolve_test.go:68-88` (`TestResolveNodeCanonicalID_PrefersDocumentNode`) | ✅ PASS |
| **AC4** | GPR-04 | Desempate por menos segmentos e id lexicograficamente menor | `internal/db/graph.go:669-678`; `internal/db/graph_resolve_test.go:90-116` (`TestResolveNodeCanonicalID_TieBreakFewestSegmentsAndLexicographicID`) | ✅ PASS |
| **AC5** | GPR-05 | Chaves na ordem: doc id, relative path, node name, doc title, fuzzy | `internal/db/graph.go:592-645`; `internal/db/graph_resolve_test.go:22-66` (`TestResolveNodeCanonicalID_ResolutionKeysOrder`) | ✅ PASS |
| **AC6** | GPR-06 | Caminho relativo resolve para o documento correspondente | `internal/db/graph.go:602-612`; `internal/db/graph_resolve_test.go:48-58` | ✅ PASS |
| **AC7** | GPR-07 | Candidatos de ambiguidade ordenados por caminho e depois id | `internal/graph/resolve.go:60-75` (`SortCandidates`); `internal/graph/resolve_test.go:35-50` | ✅ PASS |
| **AC8** | GPR-08 | SQLite e Postgres resolvem para o mesmo id canônico | `internal/store/postgres_resolve_test.go:88-118` (`TestPostgresResolveNodeCanonicalID_ParityWithSQLite`) | ✅ PASS |
| **AC9** | GPR-09 | Relatório lista cada candidato com caminho relativo e título | `internal/graph/resolve.go:80-105` (`FormatCLI`, `FormatMCP`); `internal/graph/resolve_test.go:58-75` | ✅ PASS |
| **AC10** | GPR-10 | Relatório nomeia as chaves de desambiguação (`title:` e caminho relativo com `.md` opcional) | `internal/graph/resolve.go:95-102`; `internal/graph/resolve_test.go:76-88` | ✅ PASS |
| **AC11** | GPR-11 | CLI nomeia o id canônico resolvido na saída | `cmd/mem/path.go:132-133`; `cmd/mem/path_test.go:208-215` (`strings.Contains(outStr, "Origem:          README.md")`) | ✅ PASS |
| **AC12** | GPR-12 | CLI imprime caminho relativo e título de cada candidato | `cmd/mem/path.go:146-148`; `cmd/mem/path_test.go:216-220` (`strings.Contains(outStr, "README.md") && strings.Contains(outStr, "skills/mymemory/README.md")`) | ✅ PASS |
| **AC13** | GPR-13 | CLI nomeia as chaves `title:` e caminho relativo | `cmd/mem/path.go:147`; `cmd/mem/path_test.go:221-227` (`strings.Contains(outStr, "title:") && strings.Contains(outStr, ".md")`) | ✅ PASS |
| **AC14** | GPR-14 | CLI não falha quando a resolução é ambígua (retorna rota) | `cmd/mem/path.go:105-115`; `cmd/mem/path_test.go:201-205` (`err == nil` na ambiguidade) | ✅ PASS |
| **AC15** | GPR-15 | Tool MCP devolve `InvalidParams` na ambiguidade | `internal/mcp/path_handlers.go:80-87`; `internal/mcp/path_handlers_test.go:160-185` (`mcpErr.Code == CodeInvalidParams`) | ✅ PASS |
| **AC16** | GPR-16 | Erro da tool MCP nomeia cada candidato e chaves de desambiguação | `internal/mcp/path_handlers.go:84-86`; `internal/mcp/path_handlers_test.go:175-184` | ✅ PASS |
| **AC17** | GPR-17 | Erro que não é de ambiguidade continua `CodeInternalError` | `internal/mcp/path_handlers.go:88-90`; `internal/mcp/path_handlers_test.go:190-210` (`mcpErr.Code == CodeInternalError`) | ✅ PASS |
| **AC18** | GPR-18 | Origem = destino: 0 saltos, custo 0, 1 nó, 0 arestas | `internal/graph/path.go:167-180`; `internal/graph/path_test.go:18-32` (`res.Hops == 0`, `res.TotalCost == 0.0`, `len(res.Nodes) == 1`, `len(res.Edges) == 0`) | ✅ PASS |
| **AC19** | GPR-19 | `Summary` do nó idêntico (`Nó de origem e destino são idênticos ('<id>')`), matando M7 | `internal/graph/path.go:178`; `internal/graph/path_test.go:34-40` (`res.Summary == "Nó de origem e destino são idênticos ('auth.md')"` — M7 killed) | ✅ PASS |
| **AC20** | GPR-20 | Renderização da CLI do nó idêntico (`Origem e destino são o mesmo nó ('<id>').`) | `cmd/mem/path.go:150-155`; `cmd/mem/path_test.go:290-305` (`strings.Contains(identStr, "Origem e destino são o mesmo nó ('node-iso-a').")`) | ✅ PASS |
| **AC21** | GPR-21 | Cada aresta devolvida carrega relação, status epistêmico e peso | `internal/graph/path.go:191-205`; `internal/graph/path_test.go:306-365` (`TestFindPath_EdgeDetails`) | ✅ PASS |
| **AC22** | GPR-22 | Status epistêmico da aresta ∈ {EXTRACTED, INFERRED, TAG} | `internal/graph/path.go:106-108`; `internal/graph/path_test.go:311,326,341,356` | ✅ PASS |
| **AC23** | GPR-23 | Pesos 1.0 / 0.6 / 0.3 por status epistêmico | `internal/graph/pagerank.go:10-12`; `internal/graph/path_test.go:314,329,344,359` | ✅ PASS |
| **AC24** | GPR-24 | Relação epistêmica ou vazia vira `links_to`, customizadas mantidas verbatim | `internal/graph/path.go:193-195`; `internal/graph/path_test.go:317,332,347,362` (`links_to`, `implements`, `depends_on`) | ✅ PASS |
| **AC25** | GPR-25 | Rota de 2 saltos no Postgres com `Found = true` e `Hops = 2` | `internal/store/postgres.go:1370-1425`; `internal/store/postgres_test.go:365-385` (`pathRes.Found && pathRes.Hops == 2`) | ✅ PASS |
| **AC26** | GPR-26 | Rota do Postgres não atravessa a fronteira de repositório | `internal/store/postgres.go:1390-1393`; `internal/store/postgres_test.go:386-410` (dois repositórios no mesmo banco, 0 nós de B na rota de A) | ✅ PASS |
| **AC27** | GPR-27 | Teste de integração do Postgres assere rota multi-hop, não só self-path | `internal/store/postgres_test.go:340-415` (`TestPostgresStore_FindPath_MultiHopAndRepoIsolation`) | ✅ PASS |
| **AC28** | GPR-28 | `Summary` de não-encontrado (`Nenhum caminho encontrado entre '<origem>' e '<destino>' dentro de <N> saltos`) | `internal/graph/path.go:338`; `internal/graph/path_test.go:74-79` (`res.Summary == "Nenhum caminho encontrado entre 'A' e 'D' dentro de 6 saltos"`) | ✅ PASS |
| **AC29** | GPR-29 | Linha `Status:          ❌ Não encontrado` na CLI | `cmd/mem/path.go:157-159`; `cmd/mem/path_test.go:274-277` | ✅ PASS |
| **AC30** | GPR-30 | Linha `Detalhes:` com o `Summary` na CLI | `cmd/mem/path.go:159`; `cmd/mem/path_test.go:283-288` | ✅ PASS |
| **AC31** | GPR-31 | `InvalidParams` sem `source` ou sem `target` na tool MCP | `internal/mcp/path_handlers.go:31,40`; `internal/mcp/path_handlers_test.go:225-256` (`CodeInvalidParams`) | ✅ PASS |
| **AC32** | GPR-32 | Headings `Rota Conectada` e `Segmentos Percorridos` na resposta MCP | `internal/mcp/path_handlers.go:124,138`; `internal/mcp/path_handlers_test.go:98-125` (com controle negativo) | ✅ PASS |
| **AC33** | GPR-33 | Defaults canônicos 6 / true / epistemic em `DefaultPathOptions()` | `internal/graph/path.go:26-32`; `internal/graph/path_test.go:44-55` (`TestDefaultPathOptions`) | ✅ PASS |
| **AC34** | GPR-34 | CLI aplica os defaults do helper `graph.DefaultPathOptions()` | `cmd/mem/path.go:26,31-35,53-75`; `cmd/mem/path_test.go:278-288` | ✅ PASS |
| **AC35** | GPR-35 | Tool MCP aplica os defaults do helper `graph.DefaultPathOptions()` | `internal/mcp/path_handlers.go:43-65`; `internal/mcp/path_handlers_test.go:70-95` | ✅ PASS |
| **AC36** | GPR-36 | Nenhum default de caminho literal fora de `graph.DefaultPathOptions()` | `cmd/mem/path.go:26-75`; `internal/mcp/path_handlers.go:43-65` (confirmado por grep: 0 literais) | ✅ PASS |
| **AC37** | GPR-37 | Spec `graph-path-discovery` define que `--undirected` prevalece sobre `--directed` | `.specs/features/graph-path-discovery/spec.md:51` | ✅ PASS |
| **AC38** | GPR-38 | Spec `graph-path-discovery` define que `--mode` inválido cai para `epistemic` | `.specs/features/graph-path-discovery/spec.md:52` | ✅ PASS |
| **AC39** | GPR-39 | Spec `graph-path-discovery` define que `--max-depth <= 0` reseta para 6 | `.specs/features/graph-path-discovery/spec.md:53` | ✅ PASS |
| **AC40** | GPR-40 | Spec `graph-path-discovery` define relações, status e pesos canônicos | `.specs/features/graph-path-discovery/spec.md:54` | ✅ PASS |
| **AC41** | GPR-41 | Spec `graph-path-discovery` define os dois textos literais de `Summary` | `.specs/features/graph-path-discovery/spec.md:55` | ✅ PASS |
| **AC42** | GPR-42 | Spec `graph-path-discovery` define resposta text-only e remete JSON à ISSUE-022 | `.specs/features/graph-path-discovery/spec.md:56` | ✅ PASS |
| **AC43** | GPR-43 | Sensor de discriminação re-injeta mutação do ramo de nó idêntico e derruba a suíte | `.specs/features/graph-path-discovery/validation.md:65` (`--- FAIL: TestFindPath_IdenticalSourceAndTarget`) | ✅ PASS |
| **AC44** | GPR-44 | Sensor de discriminação re-injeta mutação composta de defaults (helper 9, CLI 6) e derruba a suíte | `.specs/features/graph-path-discovery/validation.md:66` (`--- FAIL: TestDefaultPathOptions` e `--- FAIL: TestPathCLI_NotFoundAndIdenticalNode`) | ✅ PASS |
| **AC45** | GPR-45 | Sensor de discriminação re-injeta M1..M6 e todas continuam mortas | `.specs/features/graph-path-discovery/validation.md:59-64` (6/6 mortas) | ✅ PASS |
| **AC46** | GPR-46 | Verificação realizada por Verifier independente distinto do autor de T1..T13 | Verificador independente executado em subagente autônomo (`bc97865a-24fa-49f6-95ce-35135bd6bc1e`) | ✅ PASS |
| **AC47** | GPR-47 | Relatório de validação com veredito PASS e evidência `file:line` para todos os critérios | `.specs/features/fix-graph-path-resolution/validation.md` e `.specs/features/graph-path-discovery/validation.md` | ✅ PASS |

---

## Gate Check

- `go test -count=1 ./internal/graph/... ./internal/db/... ./internal/mcp/... ./cmd/mem/...`: **exit 0**
- `go test -count=1 ./internal/...`: **exit 0**
- `go build ./...`: **exit 0**
- `go vet ./...`: **exit 0**
- `python .agents/skills/tlc-spec-driven/scripts/validate_spec.py fix-graph-path-resolution --root .`: **exit 0**
- `python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py fix-graph-path-resolution --root .`: **exit 0**
- `python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-graph-path-resolution --root .`: **exit 0**
