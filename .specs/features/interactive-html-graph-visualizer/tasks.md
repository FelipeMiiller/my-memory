# Tasks: interactive-html-graph-visualizer

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| GraphView Core | unit | BuildGlobalGraph, BuildSubGraph, PageRank calculation, node metrics | internal/graphview/*_test.go | go test -v ./internal/graphview/... |
| HTML Template | unit | RenderHTML, force simulation JS injection, CSS styles, offline Zero-CDN | internal/graphview/*_test.go | go test -v ./internal/graphview/... |
| CLI Commands | unit | mem graph view, mem graph export, mem export --html | cmd/mem/*_test.go | go test -v ./cmd/mem/... |
| MCP Tools | integration | memory_visualize_graph tool registration and execution | internal/mcp/*_test.go | go test -v ./internal/mcp/... |
| Documentation / ADR | none | Documentation, ADR-020, README updates | docs/adr/* | go test -v ./cmd/mem/... ./internal/... |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| GraphView | After builder and template changes | go test -v ./internal/graphview/... |
| CLI Gate | After CLI graph commands implementation | go test -v -run TestGraph ./cmd/mem/... |
| MCP Gate | After MCP visualize tool implementation | go test -v -run TestVisualizeGraph ./internal/mcp/... |
| Full Test | Before documentation and state update | go test -count=1 -v ./cmd/mem/... ./internal/... |
| Spec Gate | Before completing feature | python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/interactive-html-graph-visualizer/spec.md |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T3 -> T4 -> T5
```

### Phase 1: Núcleo Topológico, Renderizador Web, CLI e MCP

## Task Breakdown

### T1: Pacote internal/graphview (Modelo, Builder e PageRank)
**What**: Implementar model.go e builder.go para extração de grafo global e subgrafos locais com PageRank
**Where**: internal/graphview/model.go, internal/graphview/builder.go, internal/graphview/builder_test.go
**Depends on**: none
**Requirement**: VIZ-01, VIZ-02
**Tests**: internal/graphview/builder_test.go
**Gate**: go test -v ./internal/graphview/...
**Done when**:
- [x] Definir estruturas GraphView, Node, Edge e GraphStats em model.go
- [x] Implementar BuildGlobalGraph e BuildSubGraph a partir do SQLite em builder.go
- [x] Integrar cálculo de PageRank ponderado via internal/graph e escala proporcional de raio
- [x] Adicionar suporte a extração e atribuição de paleta de cores por tipo de nota
- [x] Adicionar testes unitários em builder_test.go

### T2: Motor de Renderização HTML/SVG Standalone Autocontido
**What**: Implementar template.go gerando página HTML5 com simulação de forças gravitacionais em JS e CSS puro
**Where**: internal/graphview/template.go, internal/graphview/template_test.go
**Depends on**: T1
**Requirement**: VIZ-03, VIZ-04
**Tests**: internal/graphview/template_test.go
**Gate**: go test -v ./internal/graphview/...
**Done when**:
- [x] Criar template HTML5 com tema escuro (Dark Cyberpunk/Obsidian) sem qualquer CDN externa
- [x] Implementar motor de física de forças (repulsão Coulomb, atração molas e centralização) em JavaScript puro
- [x] Implementar controles de zoom na roda do mouse, pan e arraste de nós
- [x] Implementar campo de busca em tempo real com destaque dinâmico de nós
- [x] Implementar painel lateral de inspeção exibindo conexões (in/out), tags e link Obsidian
- [x] Implementar RenderHTML e ExportHTML
- [x] Adicionar testes unitários em template_test.go

### T3: Interface CLI (mem graph view, export e mem export --html)
**What**: Implementar subcomando mem graph e flag --html em mem export
**Where**: cmd/mem/graph.go, cmd/mem/main.go, cmd/mem/graph_test.go
**Depends on**: T2
**Requirement**: VIZ-05
**Tests**: cmd/mem/graph_test.go
**Gate**: go test -v -run TestGraph ./cmd/mem/...
**Done when**:
- [x] Implementar runGraphCLI com subcomandos view e export
- [x] Implementar abertura automática do navegador padrão com openBrowser
- [x] Adicionar suporte à flag --html em mem export para compatibilidade
- [x] Conectar roteamento e atualizar printHelp() em cmd/mem/main.go
- [x] Adicionar testes unitários em cmd/mem/graph_test.go

### T4: Ferramenta MCP memory_visualize_graph
**What**: Registrar e implementar ferramenta memory_visualize_graph no servidor MCP
**Where**: internal/mcp/tools.go, internal/mcp/visualizer_handlers.go, internal/mcp/visualizer_handlers_test.go
**Depends on**: T3
**Requirement**: VIZ-06
**Tests**: internal/mcp/visualizer_handlers_test.go
**Gate**: go test -v -run TestVisualizeGraph ./internal/mcp/...
**Done when**:
- [x] Declarar schema de memory_visualize_graph em internal/mcp/tools.go
- [x] Implementar handler JSON-RPC gerando arquivo HTML e retornando sumário topológico
- [x] Registrar ferramenta em NewServer
- [x] Adicionar testes de integração MCP em visualizer_handlers_test.go

### T5: ADR-020, Documentação e Validação Final TLC
**What**: Formalizar ADR-020, atualizar guias de uso e executar todos os gates de validação
**Where**: docs/adr/020-visualizador-interativo-de-grafo-em-html-svg.md, docs/CLI_GUIDE.md, README.md, docs/REPOSITORY_BRAIN.md, .specs/STATE.md
**Depends on**: T4
**Requirement**: VIZ-01, VIZ-02, VIZ-03, VIZ-04, VIZ-05, VIZ-06
**Tests**: none
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/interactive-html-graph-visualizer/spec.md && python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py .specs/features/interactive-html-graph-visualizer/tasks.md
**Done when**:
- [x] Registrar ADR-020 no formato MADR
- [x] Atualizar docs/adr/README.md, docs/CLI_GUIDE.md, README.md e docs/REPOSITORY_BRAIN.md
- [x] Gerar validation.md com veredito PASS
- [x] Atualizar .specs/STATE.md com AD-020
- [x] Validar 100% dos testes e gates TLC
