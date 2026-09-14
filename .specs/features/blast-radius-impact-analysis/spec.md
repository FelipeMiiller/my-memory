# Feature: blast-radius-impact-analysis

## Problem Statement
O My-Memory atualmente permite busca híbrida (ADR-009), travessia de vizinhos diretos (ADR-004), cálculo de centralidade PageRank (ADR-014) e detecção de comunidades (ADR-022). No entanto, desenvolvedores e agentes de IA ainda enfrentam um desafio crítico antes de alterar, refatorar ou revogar notas arquiteturais e código:
1. **Invisibilidade do Efeito Cascata (*Blast Radius*):** Não há como responder instantaneamente "se eu alterar ou revogar esta decisão (ADR) ou conceito X, quais outros documentos, especificações, contratos e notas em cascata serão impactados?".
2. **Falta de Classificação de Severidade de Ruptura:** Nem todo link tem o mesmo peso. Relações do tipo `implements`, `depends_on` ou `contradicts` representam impacto crítico/alto de quebra (*breaking changes*), enquanto `links_to` ou referências gerais representam impacto médio ou contextual.
3. **Agentes Sem Análise de Risco Prévia:** Assistentes de IA não dispõem de uma ferramenta especializada no servidor MCP para quantificar o risco de refatorações, sendo forçados a realizar múltiplas chamadas ad-hoc sem um score consolidado de risco.

## Goals
- [ ] Implementar motor de cálculo de impacto e raio de destruição (*Blast Radius*) em `internal/graph/impact.go`.
- [ ] Mapear o fechamento transitivo de dependentes reversos (*reverse dependency closure*) a partir de um nó alvo até profundidade $K$ configurável.
- [ ] Classificar arestas e dependências por severidade semântica (`CRITICAL`, `HIGH`, `MEDIUM`, `LOW`) de acordo com as relações (`implements`, `depends_on`, `contradicts`, `links_to`, `tagged_as`).
- [ ] Calcular uma pontuação de risco agregada (*Impact Risk Score*) combinando profundidade, severidade de aresta e centralidade (PageRank) dos nós dependentes.
- [ ] Mapear as comunidades/clusters temáticos (ADR-022) afetados pela alteração proposta.
- [ ] Implementar subcomando CLI `mem impact <node_id> [--depth N] [--json]` com visualização hierárquica e resumo de risco.
- [ ] Registrar ferramenta MCP `memory_get_impact` para agentes de IA consultarem o impacto antes de aplicar edições no repositório.
- [ ] Formalizar as decisões na ADR-023 e atualizar guias técnicos e documentação do projeto.

## Out of Scope
- Bloqueio automático de commits do Git (o comando analisa e reporta o impacto; não barra alterações a menos que configurado externamente por hooks do usuário).
- Análise de impacto sintático AST de linguagens de programação individuais (a análise baseia-se no grafo unificado de notas, decisões e arquivos do repositório).

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Direção da Travessia | Travessia reversa (nós que apontam para o nó alvo: `incoming edges`) | O impacto de mudar X recai sobre quem depende de X | y |
| Profundidade Padrão | `max_depth = 2` (diretos e secundários) | Captura os impactos essenciais sem explosão combinatória em grafos densos | y |
| Classificação de Severidade | `implements`, `depends_on`, `contradicts` -> `CRITICAL`/`HIGH`; `links_to` -> `MEDIUM`; outros -> `LOW` | Alinha o risco à semântica estrutural estabelecida no ecossistema de notas | y |
| Resolução do Nó Alvo | Suporte a caminho de arquivo (`docs/adr/001...`), ID de nota (`001...`) ou nome de conceito | Facilita a consulta tanto via terminal quanto por agentes de IA | y |
| Cruzamento com Clusters | Cada nó impactado exibe seu `CommunityID` / tema dominante (ADR-022) | Permite avaliar quais domínios funcionais do sistema sofrem perturbação | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Algoritmo de Travessia Reversa e Cálculo de Risco em Go ⭐ MVP

**User Story**: As a engenheiro de software ou arquiteto, I want calcular o fechamento de dependências reversas e a pontuação de risco para um nó so that eu descubra instantaneamente todos os documentos afetados por uma mudança.

**Why P1**: Núcleo computacional do cálculo de raio de destruição.

**Acceptance Criteria**:
1. The system SHALL implement `CalculateImpact(nodes, edges, targetID, opts)` in `internal/graph` returning structured impact metrics and affected nodes.
2. The system SHALL traverse reverse dependencies (`incoming edges`) up to the specified `max_depth`.
3. The system SHALL categorize affected dependencies into severity levels (`CRITICAL`, `HIGH`, `MEDIUM`, `LOW`) based on edge relation semantics.
4. The system SHALL compute an aggregate `RiskScore` normalized between `0.0` and `100.0` incorporating node count, depth decay, and edge weights.
5. IF the target node does not exist in the graph THEN the system SHALL return a clear error indicating the missing node.

**Independent Test**: Testes unitários com topologias canônicas em cadeia, diamante e nós isolados em `internal/graph/impact_test.go`.

---

### P2: Integração de Consultas Topológicas com SQLite e PostgreSQL

**User Story**: As a desenvolvedor, I want executar a análise de impacto contra a persistência ativa (`memory.db` ou PostgreSQL) so that o resultado reflita a totalidade do grafo do repositório.

**Why P2**: Conecta o algoritmo com a camada de persistência existente.

**Acceptance Criteria**:
1. The system SHALL provide functions in `internal/db` and `internal/store` to retrieve inbound edges and node metadata for blast radius analysis.
2. The system SHALL resolve fuzzy target identifiers (slug, caminho relativo ou título exato) para o ID canônico correspondente no grafo.

**Independent Test**: Testes de consulta topológica em `internal/db` e `internal/store`.

---

### P3: Subcomando CLI mem impact

**User Story**: As a desenvolvedor no terminal, I want rodar `mem impact <node_id>` so that eu veja uma árvore de dependentes afetados e o score de risco antes de modificar um arquivo.

**Why P3**: Interface de linha de comando ergonômica para desenvolvedores humanos.

**Acceptance Criteria**:
1. The system SHALL provide `mem impact <node_id>` displaying an aligned terminal summary with target node, total affected nodes, risk score and severity breakdown.
2. The system SHALL support `--depth <N>` (default: 2) controlling the maximum propagation depth.
3. WHERE `--json` is specified THEN the system SHALL output structured JSON containing all impact analysis details.

**Independent Test**: Testes de linha de comando em `cmd/mem/impact_test.go`.

---

### P4: Ferramenta MCP memory_get_impact e ADR-023

**User Story**: As a agente de IA, I want chamar a ferramenta MCP `memory_get_impact` so that eu avalie o raio de destruição antes de propor edições ou refatorações complexas de código e documentação.

**Why P4**: Capacidade autônoma de análise de impacto para agentes inteligentes.

**Acceptance Criteria**:
1. The system SHALL register `memory_get_impact` in the MCP tools catalog with arguments `node_id`, `max_depth` and `repository`.
2. The system SHALL format the MCP response with a clear Markdown summary, risk badge and affected node list.
3. The system SHALL document the architecture in `docs/adr/023-analise-de-impacto-e-raio-de-destruicao-blast-radius.md` conforming to MADR.

**Independent Test**: Testes de handler MCP em `internal/mcp/impact_handlers_test.go`.

---

## Requirement Traceability

| Requirement ID | User Story | Acceptance Criteria | Target Component | Status |
| -------------- | ---------- | ------------------- | ---------------- | ------ |
| IMPACT-01 | P1 | AC-1, AC-2 | `internal/graph/impact.go` | complete |
| IMPACT-02 | P1 | AC-3 | `internal/graph/impact.go` | complete |
| IMPACT-03 | P1 | AC-4 | `internal/graph/impact.go` | complete |
| IMPACT-04 | P1 | AC-5 | `internal/graph/impact.go` | complete |
| IMPACT-05 | P2 | AC-1, AC-2 | `internal/db/graph.go`, `internal/store/postgres.go` | complete |
| IMPACT-06 | P3 | AC-1, AC-2, AC-3 | `cmd/mem/impact.go` | complete |
| IMPACT-07 | P4 | AC-1, AC-2 | `internal/mcp/impact_handlers.go` | complete |
| IMPACT-08 | P4 | AC-3 | `docs/adr/023-analise-de-impacto-e-raio-de-destruicao-blast-radius.md` | complete |
