# Tasks: graph-community-detection

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Graph Algorithms | unit | DetectCommunities, modularity Q calculation, tie-breaking, disconnected graphs | internal/graph/*_test.go | go test -v ./internal/graph/... |
| GraphView & Store | unit | Community enrichment in GraphView, core node and dominant type detection | internal/graphview/*_test.go | go test -v ./internal/graphview/... |
| CLI Commands | unit | mem clusters --min-size, --json output formatting | cmd/mem/*_test.go | go test -v ./cmd/mem/... |
| MCP Tools | integration | memory_get_clusters tool registration and execution | internal/mcp/*_test.go | go test -v ./internal/mcp/... |
| Documentation / ADR | none | ADR-022, CLI_GUIDE, STATE.md updates | docs/adr/* | go test -v ./cmd/mem/... ./internal/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Graph Gate | After LPA algorithm and modularity implementation | go test -v -run TestDetectCommunities ./internal/graph/... |
| GraphView Gate | After GraphView community enrichment and store queries | go test -v ./internal/graph/... ./internal/graphview/... |
| CLI Gate | After mem clusters CLI command | go test -v -run TestClustersCLI ./cmd/mem/... |
| Full Test | Before documentation and state update | go test -count=1 -v ./cmd/mem/... ./internal/... |
| Spec Gate | Before completing feature | python C:\Users\Felipe\.cache\agent-skills\skills\tlc-spec-driven\scripts\validate_spec.py graph-community-detection |
| Tasks Gate | Before completing tasks | python C:\Users\Felipe\.cache\agent-skills\skills\tlc-spec-driven\scripts\validate_tasks.py graph-community-detection |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T3 -> T4
```

### Phase 1: Algoritmo de Comunidade, Enriquecimento, CLI e MCP

## Task Breakdown

### T1: Algoritmo LPA Ponderado e Cálculo de Modularidade Q
**What**: Implementar DetectCommunities em internal/graph/community.go com propagação ponderada e modularidade Newman-Girvan
**Where**: internal/graph/community.go
**Depends on**: none
**Requirement**: COMM-01, COMM-02, COMM-03
**Tests**: internal/graph/community_test.go
**Gate**: go test -v -run TestDetectCommunities ./internal/graph/...
**Done when**:
- [x] Implementar estruturas Community, CommunityOptions e DetectCommunities em internal/graph/community.go
- [x] Implementar propagação ponderada de rótulos com suporte a arestas EXTRACTED, INFERRED e TAG
- [x] Implementar desempate determinístico baseado na ordenação lexicográfica de IDs
- [x] Implementar cálculo de Modularidade Newman-Girvan Q
- [x] Adicionar testes unitários em internal/graph/community_test.go cobrindo grafos canônicos, nós isolados e modularidade

### T2: Enriquecimento de Comunidades no GraphView e Bancos
**What**: Adicionar identificação de nó líder, tipo dominante e campos de comunidade em internal/graphview
**Where**: internal/graphview/builder.go
**Depends on**: T1
**Requirement**: COMM-04, COMM-08
**Tests**: internal/graphview/community_test.go
**Gate**: go test -v ./internal/graph/... ./internal/graphview/...
**Done when**:
- [x] Adicionar CommunityID e CommunityLabel na struct Node em internal/graphview/model.go
- [x] Implementar função de identificação de nó líder por PageRank e tipo de nota dominante por comunidade
- [x] Enriquecer BuildGraphView para associar cores categóricas de comunidade quando ativado
- [x] Adicionar testes em internal/graphview/community_test.go

### T3: Subcomando CLI mem clusters
**What**: Implementar subcomando mem clusters com formatação tabular e suporte a --min-size e --json
**Where**: cmd/mem/clusters.go
**Depends on**: T2
**Requirement**: COMM-05, COMM-06
**Tests**: cmd/mem/clusters_test.go
**Gate**: go test -v -run TestClustersCLI ./cmd/mem/...
**Done when**:
- [ ] Implementar runClustersCLI em cmd/mem/clusters.go com suporte a flags --min-size, --json, --db, --postgres, --repo
- [ ] Implementar formatação tabular alinhada com colunas ID, Líder, Tamanho, Tipo Dominante e Membros
- [ ] Conectar subcomando mem clusters no switch de cmd/mem/main.go e documentar em printHelp()
- [ ] Adicionar testes unitários em cmd/mem/clusters_test.go

### T4: Ferramenta MCP memory_get_clusters e ADR-022
**What**: Registrar ferramenta MCP memory_get_clusters, criar ADR-022 e atualizar documentação do projeto
**Where**: docs/adr/022-deteccao-de-comunidades-e-clusters-no-grafo.md
**Depends on**: T3
**Requirement**: COMM-07, COMM-08
**Tests**: internal/mcp/cluster_handlers_test.go
**Gate**: go test -count=1 ./...
**Done when**:
- [ ] Declarar schema de memory_get_clusters em internal/mcp/tools.go e implementar handler
- [ ] Adicionar testes de integração MCP em internal/mcp/cluster_handlers_test.go
- [ ] Criar docs/adr/022-deteccao-de-comunidades-e-clusters-no-grafo.md no padrão MADR
- [ ] Atualizar docs/adr/README.md, docs/CLI_GUIDE.md e .specs/STATE.md
