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
- **AD-016**: Configuração Declarativa e Auto-Scoping de Vault (`docs/adr/016-configuracao-declarativa-e-auto-scoping-de-vault.md`)
- **AD-017**: Indexação Contínua em Tempo Real com File Watcher e Git Hooks (`docs/adr/017-indexacao-continua-com-file-watcher-e-git-hooks.md`)
- **AD-018**: Padrão Compile-not-Retrieve e Escrita Bilateral na Memória via MCP e CLI (`docs/adr/018-padrao-compile-not-retrieve-e-escrita-bilateral-mcp.md`)
- **AD-019**: Versionamento Semântico Automatizado e Criação de Tags no CI (`docs/adr/019-versionamento-semantico-e-tagging-ci.md`)
- **AD-020**: Visualizador Interativo de Grafo em HTML/SVG Standalone (`docs/adr/020-visualizador-interativo-de-grafo-em-html-svg.md`)
- **AD-021**: Servidor MCP com Transporte HTTP e Server-Sent Events (SSE) (`docs/adr/021-servidor-mcp-com-transporte-http-sse.md`)
- **AD-022**: Detecção de Comunidades e Clusters no Grafo de Conhecimento (`docs/adr/022-deteccao-de-comunidades-e-clusters-no-grafo.md`)
- **AD-023**: Análise de Impacto e Raio de Destruição (Blast Radius Analysis) (`docs/adr/023-analise-de-impacto-e-raio-de-destruicao-blast-radius.md`)
- **AD-024**: Visualização Cirúrgica em 3 Colunas (Triptych Node Inspector) (`docs/adr/024-visualizacao-cirurgica-em-3-colunas-triptych-inspector.md`)
- **AD-025**: Carregamento Progressivo de Contexto e Taxonomia de Memória (`docs/adr/025-progressive-context-loading-e-taxonomia-de-memoria.md`)
- **AD-026**: Auto-Wiring e Instalação Zero-Touch de Ferramentas de IA (`docs/adr/026-auto-wiring-e-instalacao-zero-touch-de-ferramentas-de-ia.md`)
- **AD-027**: Descoberta de Rotas e Caminho Mínimo no Grafo de Conhecimento (`docs/adr/027-descoberta-de-rotas-e-caminho-minimo-no-grafo.md`)
- **AD-028**: Staleness Banners e Detecção de Desatualização de Conhecimento (`docs/adr/028-staleness-banners-e-deteccao-de-desatualizacao.md`)
- **AD-029**: Context Packager e Subgraph Bundle com Orçamento de Tokens (`docs/adr/029-context-packager-e-subgraph-bundle.md`)

## Handoff

- **Feature**: context-packager-and-subgraph-bundle (.specs/features/context-packager-and-subgraph-bundle)
- **Phase / Task**: Início da execução (T1 - Motor de Empacotamento)
- **Completed**: Especificação (`spec.md`) e plano de tarefas (`tasks.md`) definidos.
- **In-progress**: T1 (Motor de Empacotamento de Subgrafo e Orçamento de Tokens em `internal/graph/`)
- **Next step**: Implementar `internal/graph/pack.go` e seus testes unitários em `internal/graph/pack_test.go`
- **Blockers**: none
- **Branch**: develop






