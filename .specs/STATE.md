# STATE

## Decisions

- **AD-007**: Suporte Opcional a PostgreSQL com pgvector e Referência Multi-Repositório (`docs/adr/007-suporte-opcional-a-postgresql-com-pgvector-e-multi-repositorio.md`)
- **AD-008**: Interoperabilidade com Obsidian Flavored Markdown e JSON Canvas 1.0 (`docs/adr/008-interoperabilidade-obsidian-flavored-markdown-e-json-canvas.md`)
- **AD-009**: Busca Híbrida com Reciprocal Rank Fusion (RRF) (`docs/adr/009-busca-hibrida-com-reciprocal-rank-fusion-rrf.md`)
- **AD-010**: Cache Incremental de Indexação com SHA-256 (`docs/adr/010-cache-incremental-de-indexacao-com-sha256.md`)
- **AD-011**: Arestas Epistêmicas e God Nodes / Hubs de Conhecimento (`docs/adr/011-arestas-epistemicas-e-god-nodes.md`)
- **AD-012**: Benchmarks de Performance e Conexões Inesperadas (Surprising Connections) (`docs/adr/012-benchmarks-e-conexoes-inesperadas.md`)
- **AD-013**: Higiene de Grafo, Pruning Incremental e Linter Doctor (`docs/adr/013-higiene-de-grafo-pruning-e-doctor.md`)
- **AD-014**: Centralidade de Grafo com PageRank Ponderado (`docs/adr/014-centralidade-de-grafo-com-pagerank-ponderado.md`)
- **AD-015**: Decaimento Temporal Exponencial na Busca Híbrida (`docs/adr/015-decaimento-temporal-exponencial-na-busca-hibrida.md`)

## Handoff

- **Feature**: temporal-decay-and-recency-search (.specs/features/temporal-decay-and-recency-search)
- **Phase / Task**: Complete
- **Completed**: T1 (Motor Matemático de Decaimento Temporal e Fusão RRF Ponderada), T2 (Enriquecimento de SearchResult com UpdatedAt e Queries SQLite), T3 (Suporte a Decaimento Temporal no PostgreSQL e Interface Store), T4 (Parâmetros de Decaimento na Ferramenta MCP memory_search), T5 (Flags de Decaimento no Comando CLI mem search), T6 (ADR-015, Validação Final e Documentação)
- **In-progress**: none
- **Next step**: Feature 100% validada e concluída. Aguardando próximas diretrizes do usuário.
- **Blockers**: none
- **Branch**: main






