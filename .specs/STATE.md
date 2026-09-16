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
- **AD-030**: Deep Linking e Integração de Navegação com Editores (`docs/adr/030-deep-linking-e-navegacao-de-editores.md`)
- **AD-031**: Semantic Drift e Detecção de Desvio Código-Memória (`docs/adr/031-semantic-drift-e-deteccao-de-desvio-codigo-memoria.md`)
- **AD-032**: Instalador Universal One-Liner para Windows, Linux e macOS (`docs/adr/032-instalador-universal-one-liner.md`)
- **AD-033**: Arquitetura Federada com Cofre Central e Identidade Imutável de Repositório (`docs/adr/033-federated-central-vault-and-repo-identity.md`)
- **AD-034**: Protocolo Canônico Federado e Wikilinks Cross-Vault (`docs/adr/034-protocolo-canonico-federado-e-wikilinks-cross-vault.md`)

## Handoff

- **Feature**: benchmarks-and-surprising-connections (.specs/features/benchmarks-and-surprising-connections)
- **Phase / Task**: Phase 6 (Closure & Refresh) - Re-aferição de micro-benchmarks e refresh do relatório oficial
- **Completed**: Suíte formal de micro-benchmarks re-executada em 2026-09-16 (Xeon E5-2680 v4 @ 2.40GHz, Go 1.24+, `benchtime=2s`), relatório `docs/BENCHMARKS.md` atualizado com as métricas frescas (variação <5% vs aferição anterior, comportamento estável), ADR-012 (Status: Accepted) e o conjunto completo de tarefas T1..T6 reconhecidos como entregues.
- **Completed Features**: benchmarks-and-surprising-connections (ADR-012), federated-canonical-protocol-and-cross-vault-wikilinks (ADR-034), federated-central-vault-and-repo-identity (ADR-033), universal-binary-installer (ADR-032), semantic-drift-detection (ADR-031), deep-linking-and-editor-navigation (ADR-030), context-packager-and-subgraph-bundle (ADR-029), staleness-detection-and-banners (ADR-028), graph-path-discovery (ADR-027), ai-tool-auto-wiring (ADR-026), progressive-context-loading (ADR-025), triptych-node-inspector (ADR-024), blast-radius-impact-analysis (ADR-023), graph-community-detection (ADR-022), mcp-http-sse-server (ADR-021), interactive-html-graph-visualizer (ADR-020)
- **In-progress**: none
- **Next step**: Validar (`go test -count=1 ./...`), reindexar (`mem index`), checar desvio (`mem drift`), commits atômicos (`docs(benchmarks):` e `chore(state):`) e push para `origin/develop`.
- **Blockers**: none
- **Branch**: develop







