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

## Handoff

- **Feature**: temporal-decay-and-recency-search (.specs/features/temporal-decay-and-recency-search)
- **Phase / Task**: Phase 1 / T3
- **Completed**: T1 (Motor Matemático de Decaimento Temporal e Fusão RRF Ponderada), T2 (Enriquecimento de SearchResult com UpdatedAt e Queries SQLite)
- **In-progress**: T3 (Suporte a Decaimento Temporal no PostgreSQL e Interface Store)
- **Next step**: Implementar SearchHybridRRFWithDecay no PostgresStore e atualizar contrato da interface Store.
- **Blockers**: none
- **Branch**: main






