# Tasks: blast-radius-impact-analysis

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Graph Algorithms | unit | CalculateImpact, reverse dependency closure, depth decay, severity categorization | internal/graph/*_test.go | go test -v ./internal/graph/... |
| DB & Store | unit | Inbound edges traversal, fuzzy target resolver, SQLite & Postgres queries | internal/db/*_test.go | go test -v ./internal/db/... |
| CLI Commands | unit | mem impact <node_id> output formatting, --depth, --json | cmd/mem/*_test.go | go test -v ./cmd/mem/... |
| MCP Tools & ADR | integration | memory_get_impact tool registration, execution and ADR-023 documentation | internal/mcp/*_test.go | go test -v ./internal/mcp/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Graph Gate | After blast radius algorithm implementation | go test -v -run TestCalculateImpact ./internal/graph/... |
| DB Gate | After database inbound edge queries | go test -v ./internal/db/... |
| CLI Gate | After mem impact CLI command | go test -v -run TestImpactCLI ./cmd/mem/... |
| Full Test | Before documentation and state update | go test -count=1 -v ./cmd/mem/... ./internal/... |
| Spec Gate | Before completing feature | python .agents/skills/tlc-spec-driven/scripts/validate_spec.py blast-radius-impact-analysis |
| Tasks Gate | Before completing tasks | python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py blast-radius-impact-analysis |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T3 -> T4
```

### Phase 1: Algoritmo de Impacto, Persistência, CLI e MCP

## Task Breakdown

### T1: Motor de Cálculo de Impacto e Raio de Destruição
**What**: Implementar CalculateImpact em internal/graph/impact.go com travessia reversa, severidade e RiskScore
**Where**: internal/graph/impact.go
**Depends on**: none
**Requirement**: IMPACT-01, IMPACT-02, IMPACT-03, IMPACT-04
**Tests**: internal/graph/impact_test.go
**Gate**: go test -v -run TestCalculateImpact ./internal/graph/...
**Done when**:
- [x] Implementar estruturas ImpactResult, ImpactedNode, ImpactSeverity e ImpactOptions em internal/graph/impact.go
- [x] Implementar travessia de dependências reversas (arestas inbound) até max_depth
- [x] Implementar classificação de severidade (CRITICAL, HIGH, MEDIUM, LOW) baseada em relação semântica
- [x] Implementar cálculo de RiskScore normalizado de 0 a 100 ponderando nós, profundidade e PageRank
- [x] Adicionar testes unitários em internal/graph/impact_test.go cobrindo nós isolados, cadeias lineares, diamantes e ciclos

### T2: Consultas Topológicas de Dependentes no SQLite e Postgres
**What**: Adicionar consultas de dependentes reversos e resolução de nó alvo em internal/db e internal/store
**Where**: internal/db/graph.go
**Depends on**: T1
**Requirement**: IMPACT-05
**Tests**: internal/db/graph_test.go
**Gate**: go test -v ./internal/db/...
**Done when**:
- [x] Implementar GetInboundNeighbors e ResolveNodeCanonicalID em internal/db/graph.go
- [x] Implementar GetInboundNeighbors equivalente em internal/store/postgres.go
- [x] Integrar metadados de nó e documento na montagem do grafo para análise de impacto
- [x] Adicionar testes unitários em internal/db/graph_test.go

### T3: Subcomando CLI mem impact
**What**: Implementar subcomando mem impact com suporte a --depth, --json e visualização hierárquica
**Where**: cmd/mem/impact.go
**Depends on**: T2
**Requirement**: IMPACT-06
**Tests**: cmd/mem/impact_test.go
**Gate**: go test -v -run TestImpactCLI ./cmd/mem/...
**Done when**:
- [x] Implementar runImpactCLI em cmd/mem/impact.go com suporte a flags --depth, --json, --db, --postgres, --repo
- [x] Implementar formatação tabular e em árvore com badges de severidade e resumo de risco
- [x] Conectar subcomando mem impact no switch de cmd/mem/main.go e documentar em printHelp()
- [x] Adicionar testes unitários em cmd/mem/impact_test.go

### T4: Ferramenta MCP memory_get_impact e ADR-023
**What**: Registrar ferramenta MCP memory_get_impact, criar ADR-023 e atualizar documentação do projeto
**Where**: docs/adr/023-analise-de-impacto-e-raio-de-destruicao-blast-radius.md
**Depends on**: T3
**Requirement**: IMPACT-07, IMPACT-08
**Tests**: internal/mcp/impact_handlers_test.go
**Gate**: go test -count=1 ./...
**Done when**:
- [ ] Declarar schema de memory_get_impact em internal/mcp/tools.go e implementar handler em internal/mcp/impact_handlers.go
- [ ] Adicionar testes de integração MCP em internal/mcp/impact_handlers_test.go
- [ ] Criar docs/adr/023-analise-de-impacto-e-raio-de-destruicao-blast-radius.md no padrão MADR
- [ ] Atualizar docs/adr/README.md, docs/CLI_GUIDE.md, docs/REFERENCES.md e .specs/STATE.md
