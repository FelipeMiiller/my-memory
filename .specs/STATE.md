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
- **AD-047**: Indexação de Código-Fonte via Tree-sitter (Code AST Multi-linguagem) (`docs/adr/047-code-ast-tree-sitter-multi-linguagem.md`) — Spec em `.specs/features/feat-code-ast/` (Specify + Tasks prontas; 11 tasks aguardando Execute)
- **AD-050**: Threat Model — OWASP LLM Top 10 Aplicado ao MyMemory (Proposed, transversal) (`docs/adr/050-threat-model-owasp-llm-aplicado-ao-mymemory.md`)

## Handoff

- **Feature**: voice-agent-foundation (F0+F1) — event_runtime implementado, ADR-043 promovido para Accepted
- **Phase / Task**: Phase DONE — Specify → Design (no design.md; ADR-043 já era o design) → Tasks → Execute (2 worker batches + 1 Verifier) → Validation PASS (19/19 ACs, 4/4 sensors mataram mutations)
- **Completed**: ADR-043 (Envelope de Eventos Canônico) implementado em `internal/event_runtime/`. 13 commits Go merged: `ab3b4d6` schema (T1), `18061c7` envelope (T2), `0687e88` log (T3), `968712a` outbox (T4), `dd5632d` outbox tests (T5), `d02cdc3` Revision field fixup, `d21d5be` subscriber (T6), `1827be0` dispatcher (T7), `f787975` redaction LLM02 (T8), `627056d` audit subscriber (T9), `4f2b0f7` mem events CLI (T10), `5efa0ca` CLI tests (T11), `3f0e610` mem doctor --events (T12), `e8c0e8e` cosmetic fix Verifier-found. **Quality gate final: PASS** (gofmt clean, go build clean, 20/20 packages verde, 54+ testes internos, 9 CLI tests, 3 doctor tests). **Verifier (PASS)** rodou: spec-anchored 19/19 ACs verificados, discrimination sensor 4/4 mutations matadas (Outbox atomicity, Redaction LLM02, ACK→cursor, Dispatcher fan-out), 3 deviations aceitas (D1 fan-out fix correto, D2 Windows chmod skip, D3 replay fail-closed). `validation.md` em `.specs/features/event-runtime-event-log/`. ADRs foundation status: **ADR-043 movido para Accepted** (implementado + Verifier PASS). ADR-042/044/050 seguem Proposed aguardando execução.
- **Open follow-ups para próxima sessão**: (a) ADR-044 (writer atômico + outbox + reprojeção) é o próximo passo natural — ADR-043 event_runtime já provê o barramento, mas o writer ainda não emite `memory.committed` events; `documents.revision` column existe mas nenhum writer incrementa; (b) ADR-042 (mymemoryd supervisor) só vira Implementable quando ADR-044 promover o outbox como pattern único de escrita; (c) ADR-050 continua transversal — controles já exercitados em T8 (LLM02 redaction) e T7 (LLM10 MaxAckPending), mas outros controles (sandbox, egress, allowlist tools) aguardam voz/agente; (d) divider `@docs/adr/043-...md` Accepted status; (e) atualizar `README.md` e `docs/CLI_GUIDE.md` com a nova flag `mem events` e `mem doctor --events`; (f) divider/atualizar spec.md — confito entre spec P3 AC4 "MarkAcked on success" e fan-out (worker batch 2 resolveu); (g) divider `dedicated acked_reason column` follow-up em vez de string-encode em acked_at. Dívida residual ISSUE-009 (8 dead links `tagged_as`) segue em aberto.
- **Completed Features**: voice-agent-foundation (042/043/044/050 Proposed + 002 + 036 revisitados + **043 Accepted implementado + Verifier PASS + validation.md**), post-release-v1.3.0-bugfixes (ADR-036 parcial + ADR-038 + ADR-037 spec), benchmarks-and-surprising-connections (ADR-012), federated-canonical-protocol-and-cross-vault-wikilinks (ADR-034), federated-central-vault-and-repo-identity (ADR-033), universal-binary-installer (ADR-032), semantic-drift-detection (ADR-031), deep-linking-and-editor-navigation (ADR-030), context-packager-and-subgraph-bundle (ADR-029), staleness-detection-and-banners (ADR-028), graph-path-discovery (ADR-027), ai-tool-auto-wiring (ADR-026), progressive-context-loading (ADR-025), triptych-node-inspector (ADR-024), blast-radius-impact-analysis (ADR-023), graph-community-detection (ADR-022), mcp-http-sse-server (ADR-021), interactive-html-graph-visualizer (ADR-020)
- **In-progress**: none
- **Blockers**: nenhum. ADR-043 está Accepted; ADR-042/044/050 ficam Proposed até suas próprias sessões de Specify+Execute.
- **Branch**: develop







