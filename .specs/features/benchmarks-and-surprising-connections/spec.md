# Feature: benchmarks-and-surprising-connections

## Problem Statement
O My-Memory implementa algoritmos de alto desempenho (TurboQuant 4-bit, Reciprocal Rank Fusion RRF, CTEs recursivas de grafo e cache incremental SHA-256), mas carece de uma suíte formal e reproduzível de micro-benchmarks quantitativos (`testing.B`) para aferir throughput, latência e alocação de memória. Além disso, o grafo relacional mapeia apenas links explícitos (`EXTRACTED`), sem revelar aos agentes de IA e usuários conexões conceituais latentes — nós com alta similaridade semântica mas sem links diretos entre si (*Surprising Connections*), conceito chave inspirado em `Graphify-Labs/graphify`.

## Goals
- [ ] Fornecer suíte formal de micro-benchmarks em Go (`testing.B`) para TurboQuant 4-bit, RRF, cache SHA-256 e parsing de Markdown.
- [ ] Implementar algoritmo de detecção de Conexões Inesperadas (*Surprising Connections*) no contrato `Store` (SQLite e PostgreSQL).
- [ ] Disponibilizar a ferramenta MCP `memory_get_insights` para agentes de IA consultarem pontes conceituais latentes do repositório.
- [ ] Adicionar os comandos de terminal `mem bench` e `mem insights` na CLI.
- [ ] Documentar os resultados consolidados de performance em `docs/BENCHMARKS.md` e registrar a decisão na ADR-012.

## Out of Scope
- Otimizações de baixo nível em Assembly AVX-512 (a implementação em Go puro com loop unrolling é o padrão de portabilidade).
- Treinamento online de pesos de grafo não supervisionados.

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Critério de Conexão Inesperada | Alta similaridade semântica ($\ge$ 0.70) sem aresta direta no grafo | Revela pares de notas correlatas que o autor ainda não conectou via wikilinks | y |
| Ferramenta de Benchmark | Go nativo (`testing.B`) e comando programático `mem bench` | Permite testes reproduzíveis via CI/terminal sem ferramentas externas | y |
| Formato de Saída de Insights | Tabela com source, target, similaridade e razão | Fácil leitura para humanos no terminal e formatado para LLMs no MCP | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Suíte de Micro-Benchmarks de Performance ⭐ MVP

**User Story**: As a engenheiro de software, I want executar micro-benchmarks automatizados so that eu possa comprovar quantitativamente a taxa de compressão e velocidade do TurboQuant, RRF e hashing.

**Why P1**: Fornece a base de validação empírica das garantias de desempenho da arquitetura.

**Acceptance Criteria**:
1. The system SHALL provide benchmarks measuring quantization, dequantization, and dot product throughput for 4-bit TurboQuant vs 32-bit float vectors.
2. The system SHALL provide benchmarks measuring Reciprocal Rank Fusion (RRF) execution time and allocations across 100 and 1,000 items.
3. The system SHALL provide benchmarks measuring SHA-256 content hashing throughput and Markdown connection parsing speed.

**Independent Test**: Execução de `go test -bench=. -benchmem ./internal/...` registrando operações por segundo e alocações sem erros.

---

### P2: Algoritmo de Conexões Inesperadas no Store

**User Story**: As a agente ou usuário, I want descobrir pares de notas conceitualmente relacionadas sem links diretos so that eu possa identificar lacunas e sinergias na base de conhecimento.

**Why P2**: Transforma o grafo de um mero repositório estático em um motor ativo de descoberta conceitual.

**Acceptance Criteria**:
1. The system SHALL define `SurprisingConnection` struct in `store` package with source, target, similarity, and reason fields.
2. WHEN `FindSurprisingConnections` is executed THEN the system SHALL return pairs of documents with semantic similarity $\ge$ `minSimilarity` that do NOT have direct edges in `graph_edges`.
3. IF no surprising connections meet the similarity threshold THEN the system SHALL return an empty slice without error.
4. The system SHALL support repository filtering when `repo` parameter is non-empty.

**Independent Test**: Testes unitários montando documentos com alta proximidade vetorial/textual sem aresta no grafo asserindo o retorno esperado de pares.

---

### P3: Ferramenta MCP memory_get_insights

**User Story**: As a modelo de linguagem conectado via MCP, I want chamar a ferramenta `memory_get_insights` so that eu possa sugerir links conceituais e sínteses transversais.

**Why P3**: Permite que modelos como Claude e Cursor atuem proativamente na organização da memória.

**Acceptance Criteria**:
1. The system SHALL expose `memory_get_insights` tool in the MCP catalog with `repository`, `limit`, and `min_similarity` parameters.
2. WHEN `memory_get_insights` is invoked THEN the system SHALL return the formatted list of latent connections with similarity scores.
3. IF `limit` is omitted THEN the system SHALL default to 10 connections.

**Independent Test**: Requisição JSON-RPC `tools/call` com `name: "memory_get_insights"` validando a resposta formatada.

---

### P4: CLI mem bench e mem insights

**User Story**: As a usuário no terminal, I want executar `mem bench` e `mem insights` so that eu possa verificar a velocidade do sistema e inspecionar conexões latentes diretamente.

**Why P4**: Oferece interface interativa imediata para desenvolvedores.

**Acceptance Criteria**:
1. WHEN `mem bench` is executed THEN the system SHALL run the internal performance benchmark suite and display a tabular summary.
2. WHEN `mem insights` is executed THEN the system SHALL query and output an ASCII table of surprising connections.
3. WHERE `--min-similarity` flag is passed to `mem insights` the system SHALL filter connections by the requested threshold.

**Independent Test**: Execução de `mem bench` e `mem insights` no terminal verificando saída formatada e código de saída 0.

---

## Edge Cases
- IF all documents in the repository are already interconnected THEN the system SHALL return zero surprising connections without error.
- IF a document is compared with itself THEN the system SHALL exclude reflexive self-pairs ($A == A$).
- IF vector embeddings are unavailable THEN the system SHALL fallback to lexical/Jaccard term overlap without crashing.

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| BENCH-01 | P1: Suíte de Micro-Benchmarks de Performance | Tasks | Pending |
| BENCH-02 | P1: Suíte de Micro-Benchmarks de Performance | Tasks | Pending |
| BENCH-03 | P1: Suíte de Micro-Benchmarks de Performance | Tasks | Pending |
| SURPR-01 | P2: Algoritmo de Conexões Inesperadas no Store | Tasks | Pending |
| SURPR-02 | P3: Ferramenta MCP memory_get_insights | Tasks | Pending |
| SURPR-03 | P4: CLI mem bench e mem insights | Tasks | Pending |

**Coverage:** 6 total, 6 mapped to tasks, 0 unmapped

---

## Success Criteria
- [ ] Suíte de benchmarks Go executando com sucesso e reportando métricas de ns/op e B/op.
- [ ] Algoritmo `FindSurprisingConnections` implementado e testado em SQLite e PostgreSQL.
- [ ] Ferramenta MCP `memory_get_insights` registrada no catálogo e testada com chamadas JSON-RPC.
- [ ] Subcomandos CLI `mem bench` e `mem insights` operacionais.
- [ ] Relatório `docs/BENCHMARKS.md` e ADR-012 criados e aprovados.
