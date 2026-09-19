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
- **AD-039**: Calibração do Detector de Drift (PageRank + match de path) (`docs/adr/039-calibracao-do-detector-de-drift-page-rank-e-match-de-path.md`)
- **AD-040**: Config Global Única + Storage Opt-in (`docs/adr/040-config-global-unica-storage-opt-in.md`)
- **AD-041**: Tags sem doc-casa são removidas do grafo (`docs/adr/041-issue-009-tag-sem-casa-removida-do-grafo.md`)
- **AD-042**: Núcleo Local `mymemoryd` com Workers Sidecar Isolados (Proposed) (`docs/adr/042-nucleo-local-mymemoryd-com-workers-sidecar-isolados.md`)
- **AD-043**: Envelope de Eventos Canônico (event_runtime) (Proposed) (`docs/adr/043-envelope-de-eventos-canonico-event-runtime.md`)
- **AD-044**: Memory Writer Atômico + Outbox + Reprojeção Idempotente (Proposed) (`docs/adr/044-memory-writer-atomico-outbox-e-reprojecao-idempotente.md`)
- **AD-050**: Threat Model — OWASP LLM Top 10 Aplicado ao MyMemory (Proposed, transversal) (`docs/adr/050-threat-model-owasp-llm-aplicado-ao-mymemory.md`)

## Handoff

- **Feature**: voice-agent-foundation (F0+F1) — dokumentação arquitetural a partir da pesquisa autoral 2026-09-18
- **Phase / Task**: Phase SPECIFY DONE — 4 ADRs Proposed (042/043/044/050 transversal) + 2 ADRs antigos revisitados (002 + 036 com Trilha D)
- **Completed**: ADRs 042 (`mymemoryd` supervisor com workers sidecar), 043 (envelope de eventos canônico event_runtime), 044 (writer atômico + outbox + reprojeção idempotente) e **050 (Threat Model OWASP LLM Top 10 aplicado ao mymemoryd, transversal)** criados. ADR-002 ganhou seção "Update 2026-09-19: Limites de Runtime" formalizando Go como núcleo e Python como worker sidecar. ADR-036 ganhou Trilha D — Fundação F0/F1 do roadmap pós-v1.4.0. Cross-references entre os 4 ADRs atualizadas pra apontar pros slugs reais. Estado persistido em `.specs/STATE.md`. Decisão arquitetural: foundation + threat model primeiro, voz/agente/MCP avançado dependem desses 4 alicerces.
- **Completed Features**: voice-agent-foundation (042/043/044/050 Proposed + 002 + 036 revisitados), post-release-v1.3.0-bugfixes (ADR-036 parcial + ADR-038 + ADR-037 spec), benchmarks-and-surprising-connections (ADR-012), federated-canonical-protocol-and-cross-vault-wikilinks (ADR-034), federated-central-vault-and-repo-identity (ADR-033), universal-binary-installer (ADR-032), semantic-drift-detection (ADR-031), deep-linking-and-editor-navigation (ADR-030), context-packager-and-subgraph-bundle (ADR-029), staleness-detection-and-banners (ADR-028), graph-path-discovery (ADR-027), ai-tool-auto-wiring (ADR-026), progressive-context-loading (ADR-025), triptych-node-inspector (ADR-024), blast-radius-impact-analysis (ADR-023), graph-community-detection (ADR-022), mcp-http-sse-server (ADR-021), interactive-html-graph-visualizer (ADR-020)
- **In-progress**: none
- **Next step**: Antes de partir pra F2 (voz) ou F3 (agente), executar `tlc-spec-driven` para Specify → Design → Tasks → Execute de UM dos 4 ADRs foundation. Sugiro começar pelo **ADR-043 (event_runtime)**: é pré-requisito dos outros dois (042 supervisor e 044 writer dependem do envelope). ADR-050 é transversal — não vira `Accepted` sozinho; cada ADR subsequente valida os controles que ele mapeia. Quando ADR-043 virar `Accepted` com testes passando, ativar ADR-044 e em seguida ADR-042. Dívida residual dos 8 dead links de ISSUE-009 (algoritmos, internal, resource, howto, commands, flags, Tags) segue em aberto — mesma solução proposta antes (stub docs por conceito).
- **Blockers**: nenhum. Pesquisa autoral (F0–F5) está totalmente documentada em ADRs Proposed; ADR-050 transversal ancora todos os próximos. Execução real (código Go) é decisão da próxima sessão.
- **Branch**: develop







