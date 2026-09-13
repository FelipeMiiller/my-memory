# ADR-012: Benchmarks de Performance e Conexões Inesperadas (Surprising Connections)

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: benchmarks, performance, surprising-connections, latent-knowledge, turboquant, rrf, sha256, mcp, graphify

## Context and Problem Statement

O My-Memory implementou algoritmos e estruturas de dados de alto desempenho para gestão de conhecimento (TurboQuant 4-bit, Reciprocal Rank Fusion RRF, CTEs recursivas de grafo e cache incremental SHA-256). No entanto, o projeto apresentava duas necessidades críticas de evolução:

1. **Ausência de Comprovação Empírica Formal de Performance:** Não existia uma suíte padronizada e reproduzível de micro-benchmarks em Go (`testing.B`) para aferir throughput, latência por operação e alocação de memória (B/op e allocs/op) entre a quantização 4-bit e vetores 32-bit, fusão RRF em diferentes escalas e throughput de hashing/parsing.
2. **Grafo Restrito a Conexões Explícitas:** O grafo de conhecimento navegava exclusivamente conexões expressas sintaticamente (`EXTRACTED` via wikilinks e tags). Agentes de IA e desenvolvedores ficavam sem visibilidade de **pontes conceituais latentes** — pares de documentos com alta similaridade semântica ($\ge 0.70$) que ainda não foram interconectados pelo autor (*Surprising Connections*), conceito central inspirado em [Graphify-Labs/graphify](https://github.com/Graphify-Labs/graphify).

## Decision Drivers

- **Comprovação Empírica das Garantias de Engenharia:** Fornecer benchmarks formais e comando de terminal para aferir quantitativamente a taxa de compressão (8x), latência do produto escalar (zero alocações) e velocidade do RRF.
- **Descoberta de Conhecimento Latente:** Implementar algoritmo agnóstico e de alto desempenho para detectar conexões conceituais não linkadas no SQLite e PostgreSQL.
- **Resiliência e Fallback Léxico:** Permitir que a detecção de conexões inesperadas funcione via sobreposição léxica de Jaccard caso embeddings vetoriais não estejam disponíveis.
- **Disponibilização via MCP e CLI:** Expor a ferramenta `memory_get_insights` no protocolo MCP para LLMs e os subcomandos `mem bench` e `mem insights` para desenvolvedores no terminal.

## Decision Outcome

Adotou-se uma arquitetura integrada de benchmarking quantitativo e descoberta de conhecimento latente:

1. **Suíte Formal de Micro-Benchmarks (`testing.B` e `mem bench`):**
   - Implementados benchmarks padronizados em `internal/turboquant/quantizer_bench_test.go`, `internal/store/rrf_bench_test.go`, `internal/store/hash_bench_test.go` e `internal/parser/parser_bench_test.go`.
   - Adicionado método `Dequantize` na struct `Quantizer` e `RotateInverse` no `Rotator` para descompressão e testes de fidelidade.
   - Criado subcomando CLI `mem bench` com execução in-process, isolamento de garbage collection e exibição tabular detalhada.
2. **Contrato de Conexões Inesperadas no `Store`:**
   - Adicionada struct `SurprisingConnection` em `internal/store/store.go` e método `FindSurprisingConnections(ctx, repo, limit, minSimilarity)` na interface `Store`.
   - Implementado no PostgreSQL via cálculo de cosseno vetorial `(1.0 - (c1.embedding <=> c2.embedding))` com anti-join estrito em `graph_edges` (`NOT EXISTS (SELECT 1 FROM graph_edges ...)`), ordenação decrescente e corte por `minSimilarity`.
   - Implementado no SQLite com comparação vetorial via `chunks_vec` / `chunks_turboquant` e exclusão de arestas existentes.
   - Implementado fallback léxico baseado no coeficiente de Jaccard (`CalculateJaccardSimilarity`) para operação mesmo sem embeddings.
3. **Ferramenta MCP `memory_get_insights`:**
   - Declarada `ToolMemoryGetInsights` no catálogo `tools.go` com parâmetros `repository`, `limit` (default: 10) e `min_similarity` (default: 0.70).
   - Implementados `NewMemoryGetInsightsHandler`, `FormatInsights` e método `SetInsightsHandler` no servidor MCP, integrado a backends SQLite e PostgreSQL.
4. **Subcomando CLI `mem insights`:**
   - Adicionado comando `mem insights [--limit 10] [--min-similarity 0.70] [--db <arq>] [--postgres <url>] [--repo <slug>]` com exibição em tabela ASCII formatada.
5. **Relatório Oficial de Performance:**
   - Criado documento de referência `docs/BENCHMARKS.md` registrando as métricas consolidadas.

### Positive Consequences

- **Eficiência Comprovada:** TurboQuant 4-bit comprovou redução de 87,5% em memória (384B vs 3.072B) com produto escalar rodando a ~890.000 ops/s com 0 alocações.
- **RRF de Baixa Latência:** Fusão RRF de 100 itens executa em ~153 µs/op e 1.000 itens em ~1.87 ms/op, garantindo resposta interativa em tempo real.
- **Assistência Ativa de IA:** Agentes conectados via MCP agora chamam `memory_get_insights` para sugerir novas conexões conceituais e identificar lacunas de documentação.
- **Transparência para Desenvolvedores:** Qualquer desenvolvedor pode executar `mem bench` no terminal para verificar a velocidade da biblioteca em seu hardware específico.

### Negative Consequences / Trade-offs

- **Custo Computacional do Anti-Join em Bases Muito Grandes:** Comparar pares de documentos não linkados escala com $O(N^2)$ chunks se não indexado. Mitigado no PostgreSQL agrupando por documentos e limitando a chunks principais, e no SQLite mantendo mapa de arestas em memória.
