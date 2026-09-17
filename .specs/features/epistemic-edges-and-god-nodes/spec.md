# Feature: epistemic-edges-and-god-nodes

## Problem Statement
No grafo de conhecimento atual do My-Memory, todas as conexões entre notas são tratadas como arestas genéricas links_to, sem distinção semântica da natureza da relação (como dependência, implementação, suporte ou contradição) e sem diferenciação entre arestas explícitas escritas pelo autor e arestas deduzidas pelo sistema. Além disso, agentes de IA e usuários não possuem um mecanismo analítico rápido para descobrir os nós mais importantes e densos do grafo (God Nodes / Hubs de conhecimento), dificultando a escolha de pontos de partida (entrypoints) adequados para exploração e síntese de contexto.

## Goals
- [x] Estender o parser de conexões Markdown para suportar relações tipadas (prefixos, aliases, frontmatter e tags).
- [x] Adicionar suporte a propriedades epistêmicas (epistemic_status e weight) nas tabelas graph_edges do SQLite e PostgreSQL.
- [x] Implementar algoritmo de cálculo de centralidade de grau para identificar God Nodes / Hubs de conhecimento.
- [x] Expor a ferramenta MCP memory_get_hubs para agentes de IA consultarem os principais conceitos centrais do repositório.
- [x] Adicionar o comando CLI mem hubs e atualizar a indexação mem index para persistir arestas tipadas.

## Out of Scope
- Algoritmo de PageRank com fator de amortecimento iterativo em tempo real (centralidade de grau é suficiente e muito mais rápida).
- Inferência vetorial automática não supervisionada de milhares de arestas sintéticas em vaults não indexados.

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Sintaxe de Relações Tipadas | `[[rel:target]]` e `[[target\|rel:tipo]]` | Intuitivo para usuários de Obsidian e totalmente retrocompatível com links comuns | y |
| Status Epistêmico Padrão | EXTRACTED | Arestas extraídas diretamente do conteúdo escrito possuem validade comprovada pelo autor | y |
| Métrica de God Nodes | Degree Centrality (in-degree + out-degree) | Mede diretamente a densidade de conexões conceituais com complexidade SQL O(E) | y |
| Relação de Tags | tagged_as | Transforma tags em nós conceituais conectando documentos com interesses comuns | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Parser de Conexões Tipadas e Semânticas ⭐ MVP

**User Story**: As a autor de notas ou agente, I want declarar wikilinks com relações semânticas e tags so that o grafo registre a intenção exata da conexão entre conceitos.

**Why P1**: É a base que extrai a riqueza das notas antes de qualquer persistência.

**Acceptance Criteria**:
1. The system SHALL parse typed wikilinks with prefix syntax `[[relation:Target]]` into EdgeConnection with target Target and relation relation.
2. The system SHALL parse typed wikilinks with alias syntax `[[Target|rel:relation]]` into EdgeConnection with target Target and relation relation.
3. WHEN a standard wikilink `[[Target]]` is parsed THEN the system SHALL default its relation to links_to and epistemic_status to EXTRACTED.
4. WHEN tags are present in frontmatter or body THEN the system SHALL extract edges with relation tagged_as and epistemic_status EXTRACTED.
5. The system SHALL maintain backward compatibility for OutgoingLinks containing all unique external link targets.

**Independent Test**: Testes unitários com strings Markdown variadas asserindo que conn.Edges contém as relações e alvos corretos.

---

### P2: Persistência de Propriedades Epistêmicas no Store

**User Story**: As a sistema de armazenamento, I want persistir epistemic_status e peso nas arestas do grafo so that o banco distinga conexões extraídas e inferidas.

**Why P2**: Permite consultas seletivas por tipo de relação e confiabilidade epistêmica.

**Acceptance Criteria**:
1. The system SHALL store epistemic_status and weight columns in graph_edges tables for both SQLite and PostgreSQL.
2. WHEN InsertEdgeWithProps is called THEN the system SHALL persist source_id, target_id, relation, epistemic_status, and weight.
3. IF epistemic_status is empty THEN the system SHALL default to EXTRACTED.
4. IF weight is zero THEN the system SHALL default to 1.0.

**Independent Test**: Inserção de arestas com propriedades customizadas verificando recuperação correta nos schemas SQLite e Postgres.

---

### P3: Cálculo de God Nodes e Hubs de Conhecimento

**User Story**: As a agente de IA ou usuário, I want consultar os nós com maior centralidade de conexões so that eu possa identificar os conceitos fundamentais do repositório.

**Why P3**: Permite entender a estrutura global do grafo e encontrar os melhores pontos de entrada de navegação.

**Acceptance Criteria**:
1. WHEN GetGodNodes is executed with limit N THEN the system SHALL return up to N nodes sorted in descending order of total_degree (in_degree + out_degree).
2. The system SHALL compute in_degree as incoming edges count and out_degree as outgoing edges count.
3. The system SHALL filter edges by repository when repository parameter is specified.
4. IF the graph has no edges THEN the system SHALL return an empty slice without error.

**Independent Test**: Testes unitários de banco de dados montando topologias conhecidas (estrela, linha, ciclo) e asserindo o ranqueamento esperado de centralidade.

---

### P4: Ferramenta MCP memory_get_hubs

**User Story**: As a modelo de linguagem conectado via MCP, I want chamar a ferramenta memory_get_hubs so that eu possa descobrir rapidamente o mapa conceitual de um projeto.

**Why P4**: Fornece aos agentes LLM uma visão macro dos hubs conceituais do repositório antes de formular buscas específicas.

**Acceptance Criteria**:
1. The system SHALL expose memory_get_hubs tool in the MCP tool catalog.
2. WHEN memory_get_hubs is invoked with top and repository parameters THEN the system SHALL return the formatted list of central nodes with in/out degrees.
3. IF top parameter is omitted THEN the system SHALL default to top 10 nodes.

**Independent Test**: Chamada JSON-RPC 	ools/call com 
ame: "memory_get_hubs" verificando a resposta estruturada.

---

### P5: Comando CLI mem hubs e Indexação Integrada

**User Story**: As a usuário no terminal, I want executar mem hubs e ter minhas arestas tipadas salvas no mem index so that eu possa explorar os hubs diretamente no terminal.

**Why P5**: Conecta a experiência interativa do desenvolvedor e garante que a indexação popule o grafo enriquecido.

**Acceptance Criteria**:
1. WHEN mem index runs THEN the system SHALL save all extracted edges and tags via InsertEdgeWithProps.
2. WHEN mem hubs is executed THEN the system SHALL output a formatted table of central nodes with degrees.
3. WHERE --top flag is passed to mem hubs the system SHALL restrict the output to the requested count.

**Independent Test**: Execução de mem hubs e verificação da saída tabulada no console.

---

## Edge Cases
- IF a node has zero connections THEN the system SHALL exclude it from God Nodes ranking.
- IF an edge connects a node to itself (self-loop) THEN the system SHALL count in_degree and out_degree consistently without panicking.
- IF frontmatter contains invalid YAML in relations THEN the system SHALL log a warning and continue extracting body wikilinks.

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| EDGE-01 | P1: Parser de Conexões Tipadas e Semânticas | Complete | Implemented |
| EDGE-02 | P2: Persistência de Propriedades Epistêmicas no Store | Complete | Implemented |
| EDGE-03 | P3: Cálculo de God Nodes e Hubs de Conhecimento | Complete | Implemented |
| EDGE-04 | P4: Ferramenta MCP memory_get_hubs | Complete | Implemented |
| EDGE-05 | P5: Comando CLI mem hubs e Indexação Integrada | Complete | Implemented |

**Coverage:** 5 total, 5 mapped to tasks, 0 unmapped

---

## Success Criteria
- [x] Testes do parser de wikilinks tipados passando com 100% de sucesso.
- [x] Colunas epistemic_status e weight operacionais em SQLite e PostgreSQL.
- [x] GetGodNodes retornando ordenação correta por grau de centralidade.
- [x] Ferramenta MCP memory_get_hubs registrada e testada.
- [x] CLI mem hubs exibindo os principais hubs de conhecimento.
