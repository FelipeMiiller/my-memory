# Feature: triptych-node-inspector

## Problem Statement
No fluxo de trabalho assistido por IA e na navegação de grafos de conhecimento, agentes e desenvolvedores frequentemente sofrem com a exploração cega (*blind exploration*), abrindo e lendo dezenas de arquivos inteiros apenas para descobrir quem os referencia e para onde eles apontam.
Inspirado na arquitetura do **CodeGraph** (Colby McHenry), a **Visualização Cirúrgica em 3 Colunas (Triptych Node Inspector)** fornece um modelo de inspeção instantâneo estruturado em três colunas espaciais:
1. **Inbound Links (Chamadores e Dependentes Reversos)**: Quem aponta para o nó, tipos de relação (`implements`, `depends_on`, `links_to`), severidade de impacto de quebra e autoridade PageRank.
2. **Nó Central (Núcleo e Risco)**: Metadados canônicos, PageRank, comunidade/cluster LPA (ADR-022), score de risco de destruição (ADR-023), tags e preview cirúrgico de conteúdo (*Zero File Reads*).
3. **Outbound Links (Referências de Saída)**: Para onde o nó aponta, status de existência (detectando *dead links* e nós órfãos) e pesos epistêmicos (`EXTRACTED` vs `INFERRED`).

## Goals
- [ ] Implementar as estruturas de dados e motor de montagem do tríptico em `internal/graph/inspector.go`.
- [ ] Implementar métodos `InspectNode` de alta performance na camada de persistência (`internal/db/` e `internal/store/`).
- [ ] Criar o subcomando de terminal `mem inspect <node_id> [--depth 1] [--json] [--full]` em `cmd/mem/inspect.go`.
- [ ] Registrar a ferramenta MCP `memory_inspect_node` para agentes consumirem o contexto cirúrgico sem ler múltiplos arquivos.
- [ ] Aprimorar o visualizador HTML/SVG standalone (`internal/graphview/template.go`) com um modal/painel interativo de 3 colunas.
- [ ] Formalizar as decisões na `ADR-024` e manter os artefatos de especificação e estado sincronizados.

## Out of Scope
- Edição direta de arquivos a partir do visualizador do terminal (a ferramenta é para inspeção e contexto cirúrgico de leitura).
- Parsing sintático específico de ASTs de código-fonte (opera sobre os nós e arestas unificados do grafo de notas e decisões).

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Direção das 3 colunas | `In-links` (Esquerda) \| `Nó Central` (Centro) \| `Out-links` (Direita) | Alinhamento espacial padrão estabelecido pelo CodeGraph | y |
| Truncamento de Preview | Padrão de 500 caracteres / primeiras linhas com opção `--full` | Economiza tokens para agentes de IA mantendo o contexto semântico chave | y |
| Resolução de Identificadores | Resolução flexível por ID, caminho de arquivo, título ou busca difusa | Ergonômico tanto para digitação humana no terminal quanto para chamadas de IA | y |
| Integração com Blast Radius | Exibição do `RiskScore` (0-100) e `RiskLevel` calculados via ADR-023 | O desenvolvedor/agente vê imediatamente o perigo de refatorar o nó inspecionado | y |
| Integração com Comunidades | Exibição do `CommunityID` e rótulo dominante via ADR-022 | Localiza o nó dentro do seu agrupamento conceitual macro | y |

---

## User Stories

### P1: Motor de Montagem do Tríptico em Go ⭐ MVP
**User Story**: As a desenvolvedor ou agente, I want uma função que agregue in-links, nó central e out-links so that eu tenha um modelo consolidado e determinístico do tríptico.
**Acceptance Criteria**:
1. `BuildTriptychView` monta o modelo a partir das arestas e metadados com ordenação consistente por severidade e PageRank.
2. Identifica links de saída para destinos inexistentes (*dangling/dead links*).
3. Testes unitários com topologias canônicas em `internal/graph/inspector_test.go`.

### P2: Consulta de Inspeção no SQLite e PostgreSQL
**User Story**: As a sistema de persistência, I want consultar e enriquecer dados de um nó por ID/caminho so that as conexões e conteúdo sejam extraídos cirurgicamente.
**Acceptance Criteria**:
1. `db.InspectNode` no SQLite e `pgStore.InspectNode` no PostgreSQL suportam identificadores flexíveis.
2. Recuperam chunks/conteúdo, métricas topológicas e arestas adjacentes com precisão.

### P3: Subcomando CLI mem inspect
**User Story**: As a desenvolvedor no terminal, I want executar `mem inspect <node_id>` so that eu veja o tríptico formatado ou em JSON.
**Acceptance Criteria**:
1. Suporte a `mem inspect <node_id> [--depth N] [--json] [--full]`.
2. Exibição tabular/blocos clara com in-links, métricas centrais e out-links.

### P4: Ferramenta MCP memory_inspect_node, Visualizador HTML e ADR-024
**User Story**: As a agente de IA ou usuário web, I want chamar `memory_inspect_node` ou inspecionar no HTML standalone so that eu obtenha o contexto cirúrgico sem ler arquivos avulsos.
**Acceptance Criteria**:
1. Registro de `memory_inspect_node` no catálogo MCP com saída Markdown formatada.
2. Painel/modal triptych no `internal/graphview/template.go`.
3. ADR-024 formalizada.
