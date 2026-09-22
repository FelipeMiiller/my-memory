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

- **Feature**: feat-viewer-electron — **Phase 1.5 closed** (smoke gate verde) + **Phase 2 outlined** em `.specs/features/feat-viewer-electron-phase-2/tasks.md`. Phase 1 MVP landed em `0f09ad8` (migration `viewer/+electron/` → `src/` flat com Electron Forge 7 + Vite 8 + React 19 + shadcn v4 + Tailwind 4). Phase 1.5 fechou o smoke loop que estava quebrado: `scripts/electron-smoke.test.mjs` (imports `from "playwright"` sem `expect`, fora do testDir) foi **deletado** (mavis-trash) e consolidado em `src/tests/e2e/example.test.ts` (TS, dentro do testDir), agora lançando electron de `.` (package.json `main` = `.vite/build/main.js`) com screenshot capture → `screenshot-qg-phase15.png`. Gates verdes verificados: `gofmt`, `go build`, `go test` (todos packages PASS), `tsc` (**FIXED** — redundante `tsc -p forge.env.d.ts` removido do script por redeclaração `MAIN_WINDOW_VITE_*`), `vitest` (1/1), `npm run smoke` (1 passed em 7.8s). **Achados pré-existentes (não bloqueantes)**: (1) `forge.config.ts` makers — `MakerSquirrel/Deb/Rpm` filtrados em win32 + `MakerZIP` restrito a `["darwin"]` → nenhum maker roda localmente → `out/` vazio; bundles via `.vite/` funcionam. (2) `biome lint` tem 74 erros (formatação + organizeImports + a11y `<html lang>`) — **não estava na lista de gates verdes do handoff anterior**. (3) `src/types.d.ts` redeclara `MAIN_WINDOW_VITE_DEV_SERVER_URL`/`_NAME` que `node_modules/@electron-forge/plugin-vite/forge-vite-env.d.ts` já declara — worked around removendo o 2º tsc, fix limpo fica pro ADR-049. Phase 2 cobre 5 sub-frentes em ordem: **F4 ADR-049 (CSP strict)** → **F1 (Real IPC `mem:dataset:load`/`mem:cli:run`/`mem:reveal`)** → **F2 (Code symbols — fan-in ADR-047)** → **F3 (Syntax highlight via Shiki)** → **F5 (electron-forge make cross-platform)**. ADR-049 deve ser escrito antes de F1 (CSP é pré-requisito do runtime IPC bootstrap). Achado Cytoscape style warning (`node.community_color` mapping sem dados) é cosmético, decide-se durante F2.
- **Feature**: feat-code-ast (ADR-047) — Code AST multi-linguagem via tree-sitter (binding Go oficial, opt-in via `--tags treesitter`)
- **Phase / Task**: Phase DONE — Specify → Tasks → Execute (3 worker batches + 2 Verifier iterations) → **Validation PASS-WITH-DEFER** (14/17 ACs PASS, 3 PARTIAL documentadas, 5/5 gaps da iter 1 fechados na iter 2). 12 commits merged em `develop`: `c7b81ad` ADR, `76c023d` spec, `3ea93b9`..`615f5e8` Batch 1 (T1-T4), `574c046`..`5b9e212` Batch 2 (T5-T8), `ab2a21b`..`4d20d85` Batch 3 (T9-T10), `ae614be` fix CA-14, `0983d11` fix gaps 2/4/5, `e5e85e6` docs gaps 1/3, `26c8ee1` validation iter 2.
- **Quality gate final**: PASS — gofmt clean, go build exit 0, 28/28 packages PASS em `go test -count=1 ./...`.
- **Deferral compliance** (ADR-047 §Deferral, duplo): (a) Plano A **Real tree-sitter Binding** — 3 gatilhos verificáveis (gcc/CGO toolchain, `go get go-tree-sitter` succeeds, E2E ≥95% match gopls), 5 proibições, status 0 gatilhos disparados, próxima revisão 2026-12-21; (b) Plano B **gopls LSP** — já existia (commit anterior), 3 gatilhos, status 0 disparados, próxima 2027-03-21.
- **Spec-GAPs documentados** (não-bloqueantes): (1) tree-sitter real é stub/mock isolado por `//go:build treesitter` em ambiente sem gcc (deferido por ADR-047 §Deferral Plano A); (2) `include_code=true` em `memory_search` retorna warning loud "⚠ fan-in RRF code+markdown agendado para backlog" em vez de fan-in real (Gap-2 fix de honestidade, fan-in real é follow-up de ADR-040 storage); (3) Postgres/pgvector em `code_*` tables deferred por GAP-3 (CA-16 spec relaxada para `partial (Postgres: deferred per GAP-3)`).
- **Surviving mutants na iter 2** (re-escaner): (E) boot log CLI sem guard automatizado + (G) warning include_code sem guard automatizado. Ambos detectáveis via smoke humano; follow-ups LOW opcionais não-bloqueantes.
- **Verdict**: PASS-WITH-DEFER (recommend promote). Aguarda comando do operador para: (a) considerar `feat-code-ast` DONE e fechar spec, ou (b) abrir follow-ups LOW.
- **Documentation**: README + docs/CLI_GUIDE + docs/AGENT_INTEGRATION_GUIDE + docs/ARCHITECTURE + docs/REFERENCES §1.9 + ADR-047 + spec.md + tasks.md + validation.md + validation-iter2.md todos sincronizados.
- **Cross-references for next session**: ADR-040 (Postgres code_* tables), ADR-046 (Deferred Until framework — usado na seção Deferral do ADR-047), ADR-047 §Deferral Plano A status de revisão em 2026-12-21. **`mem code-*` CLI**: `mem code-index`, `mem code-search`, `mem code-graph`, `mem code-stats` (todos com `--db=<path>` + scope do `.memory/config.yaml`).
- **Previous feature (audit/closed)**: voice-agent-foundation (F0+F1) — ADR-043 Accepted via iter 1 Verifier PASS em 2026-09-20 (commit `e8c0e8e` cosmetic fix). ADRs 042/044/050 seguem Proposed aguardando execução. **Completed Features (extended)**: feat-code-ast (ADR-047 Accepted implemented + Verifier PASS-WITH-DEFER + validation-iter2.md), voice-agent-foundation (042/043/044/050 Proposed + 002 + 036 revisitados + **043 Accepted implementado + Verifier PASS + validation.md**), post-release-v1.3.0-bugfixes (ADR-036 parcial + ADR-038 + ADR-037 spec), benchmarks-and-surprising-connections (ADR-012), federated-canonical-protocol-and-cross-vault-wikilinks (ADR-034), federated-central-vault-and-repo-identity (ADR-033), universal-binary-installer (ADR-032), semantic-drift-detection (ADR-031), deep-linking-and-editor-navigation (ADR-030), context-packager-and-subgraph-bundle (ADR-029), staleness-detection-and-banners (ADR-028), graph-path-discovery (ADR-027), ai-tool-auto-wiring (ADR-026), progressive-context-loading (ADR-025), triptych-node-inspector (ADR-024), blast-radius-impact-analysis (ADR-023), graph-community-detection (ADR-022), mcp-http-sse-server (ADR-021), interactive-html-graph-visualizer (ADR-020)
- **In-progress**: none
- **Blockers**: nenhum. ADR-043 + ADR-047 estão Accepted e validados; ADR-042/044/050 ficam Proposed até suas próprias sessões de Specify+Execute.
- **Branch**: develop
- **Open follow-ups para próxima sessão**: (a) ADR-044 (writer atômico + outbox + reprojeção) é o próximo passo natural — ADR-043 event_runtime já provê o barramento, mas o writer ainda não emite `memory.committed` events; `documents.revision` column existe mas nenhum writer incrementa; (b) ADR-042 (mymemoryd supervisor) só vira Implementable quando ADR-044 promover o outbox como pattern único de escrita; (c) ADR-050 continua transversal — controles já exercitados em T8 (LLM02 redaction) e T7 (LLM10 MaxAckPending), mas outros controles (sandbox, egress, allowlist tools) aguardam voz/agente; (d) divider `@docs/adr/043-...md` Accepted status; (e) atualizar `README.md` e `docs/CLI_GUIDE.md` com a nova flag `mem events` e `mem doctor --events`; (f) divider/atualizar spec.md — confito entre spec P3 AC4 "MarkAcked on success" e fan-out (worker batch 2 resolveu); (g) divider `dedicated acked_reason column` follow-up em vez de string-encode em acked_at. Dívida residual ISSUE-009 (8 dead links `tagged_as`) segue em aberto.
- **Completed Features**: voice-agent-foundation (042/043/044/050 Proposed + 002 + 036 revisitados + **043 Accepted implementado + Verifier PASS + validation.md**), post-release-v1.3.0-bugfixes (ADR-036 parcial + ADR-038 + ADR-037 spec), benchmarks-and-surprising-connections (ADR-012), federated-canonical-protocol-and-cross-vault-wikilinks (ADR-034), federated-central-vault-and-repo-identity (ADR-033), universal-binary-installer (ADR-032), semantic-drift-detection (ADR-031), deep-linking-and-editor-navigation (ADR-030), context-packager-and-subgraph-bundle (ADR-029), staleness-detection-and-banners (ADR-028), graph-path-discovery (ADR-027), ai-tool-auto-wiring (ADR-026), progressive-context-loading (ADR-025), triptych-node-inspector (ADR-024), blast-radius-impact-analysis (ADR-023), graph-community-detection (ADR-022), mcp-http-sse-server (ADR-021), interactive-html-graph-visualizer (ADR-020)
- **In-progress**: none
- **Blockers**: nenhum. ADR-043 está Accepted; ADR-042/044/050 ficam Proposed até suas próprias sessões de Specify+Execute.
- **Branch**: develop







