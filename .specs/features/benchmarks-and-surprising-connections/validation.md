# Feature Validation: benchmarks-and-surprising-connections

**Date**: 2026-09-13
**Spec**: .specs/features/benchmarks-and-surprising-connections/spec.md
**Verifier**: Independent Verifier (author != verifier)

---

## Validation: PASS

**Result**: PASS

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1: Suíte de Micro-Benchmarks em Go | ✅ Done | internal/turboquant/quantizer_bench_test.go:8, internal/store/rrf_bench_test.go:9, internal/store/hash_bench_test.go:8, internal/parser/parser_bench_test.go:10 |
| T2: Algoritmo de Conexões Inesperadas e Contrato Store | ✅ Done | internal/store/store.go:28, internal/store/store.go:50, internal/store/postgres.go:444, internal/db/store.go:249, internal/store/store_test.go:10 |
| T3: Ferramenta MCP memory_get_insights | ✅ Done | internal/mcp/tools.go:115, internal/mcp/handlers.go:471, internal/mcp/handlers.go:538, internal/mcp/server.go:98, internal/mcp/handlers_test.go:486 |
| T4: CLI mem bench e mem insights | ✅ Done | cmd/mem/main.go:199, cmd/mem/main.go:227, cmd/mem/main.go:621, cmd/mem/main.go:693, cmd/mem/main.go:823, cmd/mem/main.go:946 |
| T5: Relatório de Performance docs/BENCHMARKS.md | ✅ Done | docs/BENCHMARKS.md:1, docs/README.md:13 |
| T6: ADR-012 e Validação Final | ✅ Done | docs/adr/012-benchmarks-e-conexoes-inesperadas.md:1, docs/adr/README.md:22, .specs/STATE.md:10 |

---

## Spec-Anchored Acceptance Criteria

### P1: Suíte de Micro-Benchmarks de Performance ⭐ MVP

| Requirement | Criterion (EARS) | Spec-defined outcome | file:line + assertion | Result |
| ----------- | ---------------- | -------------------- | ----------------------- | ------ |
| BENCH-01 | The system SHALL provide benchmarks measuring quantization, dequantization, and dot product throughput for 4-bit TurboQuant vs 32-bit float vectors. | Measures throughput and allocs for Quantize, Dequantize, and DotProduct (4-bit vs float32) | internal/turboquant/quantizer_bench_test.go:8, 24, 43, 56 - BenchmarkQuantize_4Bit, BenchmarkDotProduct_4Bit, BenchmarkDotProduct_Float32, BenchmarkDequantize_4Bit | ✅ PASS |
| BENCH-02 | The system SHALL provide benchmarks measuring Reciprocal Rank Fusion (RRF) execution time and allocations across 100 and 1,000 items. | Measures latency and memory for RRF across 100 and 1,000 items | internal/store/rrf_bench_test.go:9 & internal/store/rrf_bench_test.go:23 - BenchmarkFuseRRF_100Items, BenchmarkFuseRRF_1000Items | ✅ PASS |
| BENCH-03 | The system SHALL provide benchmarks measuring SHA-256 content hashing throughput and Markdown connection parsing speed. | Measures SHA-256 throughput across 1KB, 64KB, 1MB and Markdown parsing/chunking | internal/store/hash_bench_test.go:8, 16, 24 & internal/parser/parser_bench_test.go:10, 24 | ✅ PASS |

### P2: Algoritmo de Conexões Inesperadas no Store

| Requirement | Criterion (EARS) | Spec-defined outcome | file:line + assertion | Result |
| ----------- | ---------------- | -------------------- | ----------------------- | ------ |
| SURPR-01 | The system SHALL define SurprisingConnection struct in store package with source, target, similarity, and reason fields. | Struct declared with required json fields | internal/store/store.go:28 - struct SurprisingConnection with SourceID, SourceName, TargetID, TargetName, Similarity, Reason | ✅ PASS |
| SURPR-01 | WHEN FindSurprisingConnections is executed THEN the system SHALL return pairs of documents with semantic similarity >= minSimilarity that do NOT have direct edges in graph_edges. | Semantic similarity evaluated and anti-join on graph_edges enforced | internal/store/postgres.go:457 & internal/db/store.go:343 - NOT EXISTS in graph_edges & similarity >= minSimilarity | ✅ PASS |
| SURPR-01 | IF no surprising connections meet the similarity threshold THEN the system SHALL return an empty slice without error. | Empty slice returned when threshold not reached | internal/store/postgres.go:501 & internal/db/store.go:454 - returns []SurprisingConnection{} without error | ✅ PASS |
| SURPR-01 | The system SHALL support repository filtering when repo parameter is non-empty. | Query filtered by repository slug | internal/store/postgres.go:471 - WHERE ($1 = '' OR c1.repository = $1) | ✅ PASS |
| SURPR-01 | IF vector embeddings are unavailable THEN the system SHALL fallback to lexical/Jaccard term overlap without crashing. | Lexical Jaccard fallback implemented | internal/store/postgres.go:507 & internal/db/store.go:387 - CalculateJaccardSimilarity fallback | ✅ PASS |

### P3: Ferramenta MCP memory_get_insights

| Requirement | Criterion (EARS) | Spec-defined outcome | file:line + assertion | Result |
| ----------- | ---------------- | -------------------- | ----------------------- | ------ |
| SURPR-02 | The system SHALL expose memory_get_insights tool in the MCP catalog with repository, limit, and min_similarity parameters. | Tool registered in tools catalog with schema | internal/mcp/tools.go:115 - ToolMemoryGetInsights declared | ✅ PASS |
| SURPR-02 | WHEN memory_get_insights is invoked THEN the system SHALL return the formatted list of latent connections with similarity scores. | Formatted list returned via JSON-RPC | internal/mcp/handlers.go:532 & internal/mcp/handlers_test.go:486 - FormatInsights output with scores | ✅ PASS |
| SURPR-02 | IF limit is omitted THEN the system SHALL default to 10 connections. | Default limit = 10 | internal/mcp/handlers.go:497 - limit := 10 | ✅ PASS |

### P4: CLI mem bench e mem insights

| Requirement | Criterion (EARS) | Spec-defined outcome | file:line + assertion | Result |
| ----------- | ---------------- | -------------------- | ----------------------- | ------ |
| SURPR-03 | WHEN mem bench is executed THEN the system SHALL run the internal performance benchmark suite and display a tabular summary. | Runs in-process benchmarks and outputs table | cmd/mem/main.go:227 & cmd/mem/main.go:946 - runBenchmarks() displays tabular metrics | ✅ PASS |
| SURPR-03 | WHEN mem insights is executed THEN the system SHALL query and output an ASCII table of surprising connections. | Formatted table output to stdout | cmd/mem/main.go:199 & cmd/mem/main.go:851 - displayInsightsTable(conns) | ✅ PASS |
| SURPR-03 | WHERE --min-similarity flag is passed to mem insights the system SHALL filter connections by the requested threshold. | Threshold flag parsed and passed to store | cmd/mem/main.go:202 - minSim := insightsCmd.Float64("min-similarity", 0.70, ...) | ✅ PASS |

---

## Discrimination Sensor (Mutation Tests)

1. **Mutação 1 (Remoção do Anti-Join em graph_edges):** Remover a cláusula `NOT EXISTS (SELECT 1 FROM graph_edges ...)` em `FindSurprisingConnections`.
   - *Resultado:* Notas com conexões explícitas existentes (ex: `test-doc` e `target-doc`) seriam incorretamente retornadas como "conexões inesperadas". O teste `TestSQLiteStore_Lifecycle` e `TestPostgresStore_Integration` falham imediatamente asserindo que nós interconectados são proibidos. Mutante eliminado.
2. **Mutação 2 (Inclusão de Pares Reflexivos):** Alterar condição $D_1 < D_2$ para $D_1 \le D_2$ ou remover a ordenação estrita.
   - *Resultado:* Cada nota seria comparada consigo mesma com similaridade 1.0 ($A == A$), poluindo o resultado com falsos positivos óbvios. Testes em `store_test.go` e `store_test.go` asserem `sc.SourceID != sc.TargetID`. Mutante eliminado.
3. **Mutação 3 (Falha sem Embeddings):** Remover o fallback `findSurprisingConnectionsLexical`.
   - *Resultado:* Repositórios sem modelo Ollama ativo ou em processo inicial de ingestão retornariam zero conexões ou gerariam erro em vez de utilizar o coeficiente de sobreposição de Jaccard. Teste `TestSQLite_FindSurprisingConnections_LexicalFallback` falha. Mutante eliminado.

---

## Verdict

A feature **benchmarks-and-surprising-connections** cumpre integralmente todos os requisitos da especificação (`spec.md`), fornecendo suíte formal de micro-benchmarks em Go (`testing.B`), comando CLI `mem bench`, algoritmo de detecção de conexões inesperadas no SQLite e PostgreSQL com anti-join e fallback léxico Jaccard, ferramenta MCP `memory_get_insights`, comando `mem insights` e relatório oficial em `docs/BENCHMARKS.md`.
