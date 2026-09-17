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
- **AD-035**: Embedder Embutido com Fallback ONNX MiniLM (Proposed) (`docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md`)
- **AD-036**: Consolidação Pós-Release v1.3.0 e Roadmap v1.4.0 (`docs/adr/036-backlog-pos-release-v1.3.0-e-roadmap-v1.4.0.md`)
- **AD-037**: Rewrite do Viewer com Vite + Vanilla TypeScript (`docs/adr/037-rewrite-viewer-com-vite-vanilla-ts.md`)
- **AD-038**: Viewer com Site Único e Dataset Fixo (`docs/adr/038-viewer-site-unico-dataset-fixo.md`)

## Handoff

- **Feature**: post-release-v1.3.0-bugfixes (.specs/features/post-release-v1.3.0-bugfixes)
- **Phase / Task**: Phase DONE — Sessão de validação e correções pós-release v1.3.0 (Trilha A 100%, B1+B2 parciais, B3 revertido por feedback, qualidade validada via gate)
- **Completed**: 8 commits nesta sessão (c45041d → 597e0a9), 19 packages verde, build limpo, gofmt limpo, smoke test exit 0, visual Playwright confirmado, CI success. ADRs 036 (parcial) + 037 + 038 criados. Rule `.agents/rules/always-quality-gate.md` instituída. Benchmarks baselines em graphview + embedder. Drift baseline resetado via `mem index` (68.2/100, 16 críticos legados).
- **Completed Features**: post-release-v1.3.0-bugfixes (ADR-036 parcial + ADR-038 + ADR-037 spec), benchmarks-and-surprising-connections (ADR-012), federated-canonical-protocol-and-cross-vault-wikilinks (ADR-034), federated-central-vault-and-repo-identity (ADR-033), universal-binary-installer (ADR-032), semantic-drift-detection (ADR-031), deep-linking-and-editor-navigation (ADR-030), context-packager-and-subgraph-bundle (ADR-029), staleness-detection-and-banners (ADR-028), graph-path-discovery (ADR-027), ai-tool-auto-wiring (ADR-026), progressive-context-loading (ADR-025), triptych-node-inspector (ADR-024), blast-radius-impact-analysis (ADR-023), graph-community-detection (ADR-022), mcp-http-sse-server (ADR-021), interactive-html-graph-visualizer (ADR-020), **parser-ears-bugfix** (74804c4: stripCodeBlocks estendido + 7 placeholders EARS corrigidos + regression test; -14 dead links no `mem doctor`), **ci-go-version-bump** (aff45ca: workflow `ci.yml` Go 1.23 → 1.25 matching go.mod; desflaquea o setup-go@v5 toolchain extraction), **cross-platform-cmd-test** (53da40b: `TestResolveStorageAndRepo_DerivesRepoFromDBPath` agora seleciona path absoluto por `runtime.GOOS`; falha em Linux/macOS descoberta após o bump do Go)
- **In-progress**: none
- **Next step**: Viewer rewrite (Vite + Vanilla TS) conforme ADR-037. Considerar B2 edit-ability quando começar ADR-037. **Dívida residual de dead links**: ~30 restantes, sendo (a) tags `#algorithms`, `#architecture`, etc. virando dead links por não terem nota de mesmo nome (problema separado, fora do escopo parser+EARS), e (b) `[[Wikilinks]]` no AGENTS.md/ADRs (fuzzy resolve não casa, melhorar Obsidian sync). Nenhum é regressão do fix atual.
- **Blockers**: none
- **Branch**: develop







