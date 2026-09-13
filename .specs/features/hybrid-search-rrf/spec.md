# Feature: hybrid-search-rrf

## Problem Statement
A recuperação de contexto baseada exclusivamente em embeddings vetoriais (k-NN) sofre com baixa precisão em correspondências léxicas exatas (como nomes de funções, IDs, hashes e siglas específicas) e ignora a topologia relacional das notas conectadas. Uma busca híbrida que funde os rankings de busca léxica exata (BM25 via FTS5 / tsvector), busca vetorial densa (k-NN via sqlite-vec, TurboQuant ou pgvector) e expansão estrutural de nós vizinhos no grafo (CTE recursivo) via Reciprocal Rank Fusion (RRF) resolve a dependência de pesos calibrados manualmente e aumenta expressivamente a qualidade do contexto recuperado para agentes de IA.

## Goals
- [ ] Implementar o algoritmo genérico e determinístico de Reciprocal Rank Fusion (RRF) em Go com pontuação parametrizada k (padrão 60).
- [ ] Implementar busca léxica FTS5 (BM25) no SQLite e busca textual no PostgreSQL.
- [ ] Fundir as três modalidades de busca (FTS5/BM25, k-NN vetorial e expansão de vizinhos no grafo) no SQLite e no PostgreSQL via RRF.
- [ ] Expor a busca híbrida via ferramenta MCP memory_search com suporte a modos (hybrid, vector, fts).
- [ ] Integrar o comando CLI mem search para suportar busca híbrida com RRF.

## Out of Scope
- Re-ranking neural pesado via modelos Cross-Encoder locais (mantendo baixa latência e consumo zero de GPU/VRAM adicional).
- Modificação do formato de compressão vetorial do TurboQuant.
- Implementação de cache de indexação por SHA-256 (objeto da Frente 2).

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Constante k do RRF | k = 60 | Valor padrão comprovado na literatura de RI (Cormack et al.) e adotado pelo Elasticsearch / ai-memory | y |
| Modo padrão de busca no MCP | hybrid | Garante máxima qualidade na recuperação para agentes sem exigir novos parâmetros obrigatórios | y |
| Expansão de grafo na fusão RRF | Chunks pertencentes aos nós vizinhos das top sementes recebem ranking estrutural | Permite que documentos adjacentes no grafo reforcem a relevância sem poluir resultados irrelevantes | y |
| Fallback para falta de embeddings | Executar apenas ranking léxico FTS5 se vetor não estiver disponível | Mantém a usabilidade mesmo quando Ollama estiver inacessível | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Algoritmo de Fusão RRF e Estrutura Unificada ⭐ MVP

**User Story**: As a desenvolvedor ou agente, I want fundir múltiplos rankings ordenados através da fórmula de Reciprocal Rank Fusion (RRF) so that resultados relevantes em diferentes modalidades recebam pontuações justas sem calibração manual de pesos.

**Why P1**: É a fundação matemática e agnóstica necessária para qualquer busca híbrida no sistema.

**Acceptance Criteria**:
1. The system SHALL compute the score of candidate d as the sum of 1 / (k + rank_m(d)) across all ranked source lists m containing d.
2. WHEN multiple ranked lists contain the same candidate THEN the system SHALL accumulate reciprocal scores and assign the fused candidate higher priority.
3. IF a candidate is missing from a ranked list THEN the system SHALL contribute zero reciprocal score from that specific list without causing an error.
4. WHEN the fusion completes THEN the system SHALL return candidates sorted in descending order of fused RRF score.

**Independent Test**: Testes unitários puros com listas de entrada conhecidas e asserção exata das pontuações e ordem de classificação calculadas.

---

### P2: Busca Híbrida no SQLite (FTS5 + k-NN/TurboQuant + Grafo CTE)

**User Story**: As a usuário do CLI ou MCP sobre SQLite local, I want realizar busca híbrida que combine FTS5, vetores (sqlite-vec ou TurboQuant) e vizinhos no grafo so that eu obtenha os trechos mais precisos do meu vault.

**Why P2**: O SQLite é o backend primário padrão local do My-Memory.

**Acceptance Criteria**:
1. WHEN a text query is executed on SQLite THEN the system SHALL retrieve matching chunks from chunks_fts sorted by BM25 rank.
2. WHEN hybrid search is requested on SQLite THEN the system SHALL execute FTS5 search, vector search, expand top seed neighbors via CTE, and fuse all candidates via RRF.
3. WHERE TurboQuant mode is enabled on SQLite the system SHALL use TurboQuant dot-product ranking as the vector component in RRF fusion.
4. IF a query contains syntax error characters for FTS5 THEN the system SHALL sanitize the query before execution to prevent SQL errors.

**Independent Test**: Teste de integração consultando banco SQLite com dados sintéticos e verificando presença de correspondências léxicas e semânticas ordenadas por RRF.

---

### P3: Busca Híbrida no PostgreSQL com pgvector

**User Story**: As a usuário de ambientes corporativos ou multi-repositório no PostgreSQL, I want executar busca híbrida com RRF combinando tsvector/tsquery, pgvector e o grafo de edges so that múltiplos repositórios tenham recuperação de alta fidelidade.

**Why P3**: Mantém a paridade funcional entre os armazenamentos SQLite e PostgreSQL definida no ADR-007.

**Acceptance Criteria**:
1. WHEN hybrid search is executed on PostgreSQL THEN the system SHALL execute text search via ts_rank, vector search via pgvector cosine distance, and graph CTE traversal.
2. The system SHALL fuse the PostgreSQL search results using the unified RRF algorithm and filter candidates by repository when provided.
3. IF repository filter is specified THEN the system SHALL restrict all three modalities (lexical, vector, graph) to that repository.

**Independent Test**: Testes unitários com mocks ou formatação e testes de integração com Postgres se disponível.

---

### P4: Integração MCP e CLI (memory_search e mem search)

**User Story**: As a agente de IA via MCP ou desenvolvedor via terminal, I want invocar a busca híbrida como padrão e inspecionar os scores RRF e conexões so that eu tenha contexto preciso para geração de respostas.

**Why P4**: É a interface externa que agentes (Claude, Cursor) e usuários utilizam para acessar a memória.

**Acceptance Criteria**:
1. WHEN memory_search is called via MCP without mode parameter THEN the system SHALL execute hybrid RRF search by default.
2. WHERE mode parameter is set to vector or fts in memory_search the system SHALL execute only the requested search modality.
3. WHEN mem search is executed in CLI THEN the system SHALL display RRF scores, individual ranking sources, and discovered graph connections.
4. IF Ollama is offline during hybrid search THEN the system SHALL gracefully degrade to lexical FTS5 search and inform the user.

**Independent Test**: Testes unitários do MCP handler cobrindo fallback, chamadas válidas e formatação de resultados com score RRF.

---

## Edge Cases
- IF query string is completely empty THEN the system SHALL return an empty result list without querying the database.
- IF no matches are found in any modality THEN the system SHALL return an empty slice with zero errors.
- IF query contains unmatched quotation marks or FTS5 boolean operators THEN the system SHALL sanitize input to safe plain tokens.
- WHEN k parameter is less than or equal to zero THEN the system SHALL default to k = 60.

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| RRF-01 | P1: Algoritmo de Fusão RRF e Estrutura Unificada | Tasks | Verified |
| RRF-02 | P1: Algoritmo de Fusão RRF e Estrutura Unificada | Tasks | Verified |
| RRF-03 | P2: Busca Híbrida no SQLite (FTS5 + k-NN/TurboQuant + Grafo CTE) | Tasks | Verified |
| RRF-04 | P2: Busca Híbrida no SQLite (FTS5 + k-NN/TurboQuant + Grafo CTE) | Tasks | Verified |
| RRF-05 | P3: Busca Híbrida no PostgreSQL com pgvector | Tasks | Verified |
| RRF-06 | P4: Integração MCP e CLI (memory_search e mem search) | Tasks | Verified |
| RRF-07 | P4: Integração MCP e CLI (memory_search e mem search) | Tasks | Pending |

**Coverage:** 7 total, 7 mapped to tasks, 0 unmapped

---

## Success Criteria
- [ ] 100% de precisão nos testes unitários do algoritmo RRF com validação de empates, pesos e múltiplos rankings.
- [ ] Busca híbrida funcional no SQLite e PostgreSQL com pontuação combinada de FTS, vetores e grafo.
- [ ] Handlers do MCP memory_search atualizados com modo híbrido por padrão.
- [ ] CLI mem search exibindo resultados híbridos com fontes de ranking e scores.
