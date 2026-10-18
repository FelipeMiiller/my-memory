# Architecture Decision Records (ADRs)

Este diretório contém os Registros de Decisão de Arquitetura (**ADRs**) do projeto **My-Memory**, estruturados no formato **MADR** conforme as melhores práticas de engenharia de software.

---

## 📋 Índice de Decisões

| Número | Título | Status | Data | Tags |
| :--- | :--- | :--- | :--- | :--- |
| **[ADR-001](001-uso-de-sqlite-como-camada-unificada-de-dados.md)** | Uso de SQLite como Camada Unificada de Dados | Aceito | 2026-09-12 | `database`, `sqlite`, `fts5`, `architecture` |
| **[ADR-002](002-adocao-de-go-como-linguagem-principal.md)** | Adoção de Go como Linguagem Principal de Implementação | Aceito | 2026-09-12 | `language`, `go`, `cli`, `performance` |
| **[ADR-003](003-compressao-vetorial-de-4-bit-via-turboquant.md)** | Compressão Vetorial de 4-bit via TurboQuant | Aceito | 2026-09-12 | `vectors`, `quantization`, `turboquant`, `math` |
| **[ADR-004](004-modelagem-de-grafo-com-recursive-ctes.md)** | Modelagem e Travessia de Grafo com SQL Recursivo (CTEs) | Aceito | 2026-09-12 | `graph`, `sqlite`, `sql`, `knowledge-graph` |
| **[ADR-005](005-markdown-com-wikilinks-como-fonte-de-verdade.md)** | Markdown e [[Wikilinks]] como Entrada e Grafo Humano | Aceito | 2026-09-12 | `obsidian`, `parser`, `markdown`, `pkm` |
| **[ADR-006](006-integracao-com-agentes-de-ia-via-mcp.md)** | Integração com Agentes de IA via Model Context Protocol (MCP) | Aceito | 2026-09-12 | `ai`, `mcp`, `claude`, `cursor`, `integration` |
| **[ADR-007](007-suporte-opcional-a-postgresql-com-pgvector-e-multi-repositorio.md)** | Suporte Opcional a PostgreSQL com pgvector e Referência Multi-Repositório | Aceito | 2026-09-13 | `database`, `postgres`, `pgvector`, `multi-repository` |
| **[ADR-008](008-interoperabilidade-obsidian-flavored-markdown-e-json-canvas.md)** | Interoperabilidade com Obsidian Flavored Markdown e JSON Canvas 1.0 | Aceito | 2026-09-13 | `obsidian`, `markdown`, `wikilinks`, `json-canvas`, `agent-skills` |
| **[ADR-009](009-busca-hibrida-com-reciprocal-rank-fusion-rrf.md)** | Busca Híbrida com Reciprocal Rank Fusion (RRF) | Aceito | 2026-09-13 | `rrf`, `hybrid-search`, `fts5`, `pgvector`, `graph` |
| **[ADR-010](010-cache-incremental-de-indexacao-com-sha256.md)** | Cache Incremental de Indexação com SHA-256 | Aceito | 2026-09-13 | `cache`, `indexing`, `sha256`, `performance`, `ollama` |
| **[ADR-011](011-arestas-epistemicas-e-god-nodes.md)** | Arestas Epistêmicas e God Nodes / Hubs de Conhecimento | Aceito | 2026-09-13 | `epistemic-edges`, `god-nodes`, `knowledge-hubs`, `degree-centrality`, `graphify` |
| **[ADR-012](012-benchmarks-e-conexoes-inesperadas.md)** | Benchmarks de Performance e Conexões Inesperadas (Surprising Connections) | Aceito | 2026-09-13 | `benchmarks`, `performance`, `surprising-connections`, `turboquant`, `rrf`, `mcp` |
| **[ADR-013](013-higiene-de-grafo-pruning-e-doctor.md)** | Higiene de Grafo, Pruning Incremental e Linter Doctor | Aceito | 2026-09-13 | `lifecycle`, `pruning`, `garbage-collection`, `graph-doctor`, `health-score`, `mcp`, `cli` |
| **[ADR-014](014-centralidade-de-grafo-com-pagerank-ponderado.md)** | Centralidade de Grafo com PageRank Ponderado | Aceito | 2026-09-13 | `graph`, `pagerank`, `centrality`, `epistemic-edges`, `god-nodes`, `hubs`, `mcp`, `cli` |
| **[ADR-015](015-decaimento-temporal-exponencial-na-busca-hibrida.md)** | Decaimento Temporal Exponencial na Busca Híbrida | Aceito | 2026-09-13 | `search`, `rrf`, `time-decay`, `recency`, `sqlite`, `postgres`, `mcp`, `cli` |
| **[ADR-016](016-configuracao-declarativa-e-auto-scoping-de-vault.md)** | Configuração Declarativa e Auto-Scoping de Vault | Aceito | 2026-09-13 | `config`, `yaml`, `cli`, `auto-scoping`, `glob`, `vault`, `mcp` |
| **[ADR-017](017-indexacao-continua-com-file-watcher-e-git-hooks.md)** | Indexação Contínua em Tempo Real com File Watcher e Git Hooks | Aceito | 2026-09-13 | `watcher`, `live-indexing`, `debounce`, `git-hooks`, `pre-commit`, `sqlite`, `postgres`, `mcp` |
| **[ADR-018](018-padrao-compile-not-retrieve-e-escrita-bilateral-mcp.md)** | Padrão Compile-not-Retrieve e Escrita Bilateral na Memória via MCP e CLI | Aceito | 2026-09-13 | `compile-not-retrieve`, `llm-wiki`, `mcp`, `bilateral-memory`, `atomic-notes`, `cli` |
| **[ADR-019](019-versionamento-semantico-e-tagging-ci.md)** | Versionamento Semântico Automatizado e Criação de Tags no CI | Aceito | 2026-09-13 | `semver`, `conventional-commits`, `github-actions`, `ci-cd`, `releases`, `ldflags` |
| **[ADR-020](020-visualizador-interativo-de-grafo-em-html-svg.md)** | Visualizador Interativo de Grafo em HTML/SVG Standalone | Aceito | 2026-09-13 | `graphview`, `interactive-graph`, `force-directed`, `svg`, `zero-cdn`, `pagerank`, `cli`, `mcp` |
| **[ADR-021](021-servidor-mcp-com-transporte-http-sse.md)** | Servidor MCP com Transporte HTTP e Server-Sent Events (SSE) | Aceito | 2026-09-13 | `mcp`, `sse`, `server-sent-events`, `http`, `network-server`, `cors`, `remote-agents`, `cli` |
| **[ADR-022](022-deteccao-de-comunidades-e-clusters-no-grafo.md)** | Detecção de Comunidades e Clusters no Grafo de Conhecimento | Aceito | 2026-09-13 | `graph`, `community-detection`, `lpa`, `modularity`, `newman-girvan`, `graphview`, `clustering`, `cli`, `mcp` |
| **[ADR-023](023-analise-de-impacto-e-raio-de-destruicao-blast-radius.md)** | Análise de Impacto e Raio de Destruição (Blast Radius Analysis) | Aceito | 2026-09-13 | `graph`, `impact-analysis`, `blast-radius`, `reverse-dependencies`, `risk-scoring`, `pagerank`, `cli`, `mcp` |
| **[ADR-024](024-visualizacao-cirurgica-em-3-colunas-triptych-inspector.md)** | Visualização Cirúrgica em 3 Colunas (Triptych Node Inspector) | Aceito | 2026-09-14 | `graph`, `triptych`, `inspector`, `surgical-context`, `zero-file-reads`, `codegraph`, `pagerank`, `cli`, `mcp` |
| **[ADR-025](025-progressive-context-loading-e-taxonomia-de-memoria.md)** | Carregamento Progressivo de Contexto e Taxonomia de Memória | Aceito | 2026-09-14 | `progressive-context`, `context-stuffing`, `l0-abstract`, `taxonomy`, `openviking`, `codegraph`, `search`, `mcp`, `cli` |
| **[ADR-026](026-auto-wiring-e-instalacao-zero-touch-de-ferramentas-de-ia.md)** | Auto-Wiring e Instalação Zero-Touch de Ferramentas de IA (`mem install` / `mem setup`) | Aceito | 2026-09-14 | `autowire`, `mcp`, `zero-touch`, `devx`, `claude-desktop`, `cursor`, `vscode`, `windsurf`, `cli` |
| **[ADR-027](027-descoberta-de-rotas-e-caminho-minimo-no-grafo.md)** | Descoberta de Rotas e Caminho Mínimo no Grafo de Conhecimento (`mem path` / `memory_find_path`) | Aceito | 2026-09-14 | `graph`, `pathfinding`, `dijkstra`, `shortest-path`, `epistemic-weights`, `bfs`, `mcp`, `cli` |
| **[ADR-028](028-staleness-banners-e-deteccao-de-desatualizacao.md)** | Staleness Banners e Detecção de Desatualização de Conhecimento (`mem status`) | Aceito | 2026-09-14 | `staleness`, `drift-detection`, `banners`, `watcher`, `mcp`, `cli`, `consistency`, `codegraph` |
| **[ADR-029](029-context-packager-e-subgraph-bundle.md)** | Context Packager e Subgraph Bundle com Orçamento de Tokens (`mem pack` / `memory_pack_context`) | Aceito | 2026-09-14 | `graph`, `context-packager`, `subgraph`, `token-budget`, `mermaid`, `progressive-context`, `mcp`, `cli` |
| **[ADR-030](030-deep-linking-e-navegacao-de-editores.md)** | Deep Linking e Integração de Navegação com Editores (`mem open` e URIs `obsidian://` / `vscode://`) | Aceito | 2026-09-14 | `deeplink`, `obsidian`, `vscode`, `editor-navigation`, `cli`, `mcp`, `uri-scheme`, `ux` |
| **[ADR-031](031-semantic-drift-e-deteccao-de-desvio-codigo-memoria.md)** | Semantic Drift e Detecção de Desvio Código-Memória (`mem drift` e `memory_get_drift`) | Aceito | 2026-09-14 | `semantic-drift`, `knowledge-drift`, `git-diff`, `uncovered-code`, `cli`, `mcp`, `governance`, `ci-cd` |
| **[ADR-032](032-instalador-universal-one-liner.md)** | Instalador Universal One-Liner para Windows, Linux e macOS (`install.ps1` e `install.sh`) | Aceito | 2026-09-14 | `installer`, `one-liner`, `devx`, `release-automation`, `powershell`, `posix-shell`, `sha256`, `security` |
| **[ADR-033](033-federated-central-vault-and-repo-identity.md)** | Arquitetura Federada de Vault Central e Identidade Imutável de Repositório (`repo_id`) | Aceito | 2026-09-15 | `federation`, `central-vault`, `repo-id`, `google-drive`, `multi-vault`, `postgresql`, `sqlite`, `bootstrap` |
| **[ADR-034](034-protocolo-canonico-federado-e-wikilinks-cross-vault.md)** | Protocolo Canônico Federado e Wikilinks Cross-Vault (`memory://`) | Aceito | 2026-09-16 | `federation`, `cross-vault`, `canonical-uri`, `wikilinks`, `graph`, `parser`, `deeplink`, `mcp` |
| **[ADR-035](035-embedder-embutido-com-fallback-onnx-minilm.md)** | Embedder Embutido com Fallback Automático (`paraphrase-multilingual-MiniLM-L12-v2` via ONNX Runtime) | Aceito | 2026-09-16 | `embedding`, `semantic-search`, `onnx`, `minilm`, `frugal`, `cpu-only` |
| **[ADR-036](036-backlog-pos-release-v1.3.0-e-roadmap-v1.4.0.md)** | Consolidação Pós-Release v1.3.0 — Backlog Técnico e Roadmap para v1.4.0 | Aceito | 2026-09-16 | `backlog`, `technical-debt`, `release-planning`, `embedder`, `multi-repo` |
| **[ADR-039](039-calibracao-do-detector-de-drift-page-rank-e-match-de-path.md)** | Calibração do Detector de Drift (PageRank e Match de Path) | Aceito | 2026-09-17 | `semantic-drift`, `calibration`, `pagerank`, `path-matching`, `false-positive`, `ci-cd` |
| **[ADR-040](040-config-global-unica-storage-opt-in.md)** | Configuração única em `~/.memory/` + SQLite auto-scope por padrão (Postgres opt-in) | Aceito | 2026-09-18 | `architecture`, `federation`, `config`, `storage`, `central-vault`, `pgvector` |
| **[ADR-041](041-issue-009-tag-sem-casa-removida-do-grafo.md)** | Tags sem doc-casa são removidas do grafo (não viram dead links) | Aceito | 2026-09-18 | `architecture`, `parser`, `graph`, `dead-links`, `tag-resolution` |
| **[ADR-042](042-nucleo-local-mymemoryd-com-workers-sidecar-isolados.md)** | Núcleo Local `mymemoryd` com Workers Sidecar Isolados | Proposed | 2026-09-19 | `runtime`, `supervision`, `sidecar`, `sandbox`, `mymemoryd` |
| **[ADR-043](043-envelope-de-eventos-canonico-event-runtime.md)** | Envelope de Eventos Canônico (`event_runtime`) | Aceito | 2026-09-19 | `events`, `event-driven`, `envelope`, `outbox`, `replay`, `idempotency` |
| **[ADR-044](044-memory-writer-atomico-outbox-e-reprojecao-idempotente.md)** | Memory Writer Atômico, Outbox e Reprojeção Idempotente | Aceito | 2026-09-19 | `writer`, `atomicity`, `outbox`, `idempotency`, `projection`, `crash-safety` |
| **[ADR-045](045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md)** | ASR Streaming Engine — Nemotron 3.5 (0.6B) Embutido via ONNX Runtime | Deferred | 2026-09-19 | `asr`, `voice`, `streaming`, `nemotron`, `onnx-runtime`, `frugal`, `language-pt-br` |
| **[ADR-046](046-deferred-until-padrao-para-fallbacks-opcionais-em-adrs.md)** | Padrão "Deferred Until" para Fallbacks Opcionais em ADRs | Aceito | 2026-09-19 | `methodology`, `architecture`, `yagni`, `adr-038`, `deferral`, `scope-reduction`, `cross-cutting` |
| **[ADR-047](047-code-ast-tree-sitter-multi-linguagem.md)** | Indexação de Código-Fonte via Tree-sitter (Code AST Multi-linguagem) | Aceito | 2026-09-21 | `code-intelligence`, `tree-sitter`, `ast`, `multi-language`, `cgo`, `search`, `hybrid-ranking`, `code-graph` |
| **[ADR-050](050-threat-model-owasp-llm-aplicado-ao-mymemory.md)** | Threat Model — OWASP LLM Top 10 Aplicado ao MyMemory | Proposed | 2026-09-19 | `security`, `threat-model`, `owasp`, `llm`, `sandbox`, `egress`, `policy`, `red-team` |
| **[ADR-054](054-escopo-de-paridade-com-o-obsidian.md)** | Escopo de paridade com o Obsidian é o vault, não o aplicativo | Aceito | 2026-10-18 | `scope`, `product`, `vault`, `obsidian`, `parity` |
| **[ADR-055](055-semantica-de-tag-consultavel-sem-nos-mortos.md)** | Semântica de tag consultável sem nós mortos (adenda da ADR-041) | Proposed | 2026-10-18 | `graph`, `tags`, `adr-041`, `semantics` |
| **[ADR-056](056-persistencia-de-propriedades-de-frontmatter.md)** | Persistência de propriedades de frontmatter em tabela consultável | Aceito | 2026-10-18 | `schema`, `frontmatter`, `properties`, `obsidian` |
| **[ADR-057](057-mtime-e-metadados-de-arquivo-no-indice.md)** | mtime e metadados de arquivo no índice | Aceito — **Implementado** | 2026-10-18 | `schema`, `metadata`, `staleness`, `obsidian` |

> [!success]
> **ADR-057: implementada.** `documents.mtime` e `documents.size` existem nos dois
> engines (SQLite e PostgreSQL) e são preenchidos a partir do `stat` real do arquivo em
> ambos os caminhos de indexação (`mem index` e watcher). O decaimento temporal do
> `--decay` lê `mtime` com fallback para `updated_at`, a detecção de staleness compara o
> `mtime` persistido contra o do disco, e `mem doctor` reporta quantos documentos foram
> indexados antes das colunas existirem. A superfície de leitura acompanha em
> `mem files`, `mem folders`, `mem read`, `mem outline`, `mem wordcount`, `mem random`,
> `mem deadends`, `mem file` e `mem recents`, com o mesmo comportamento exposto via MCP
> em `memory_list_files`, `memory_read_note`, `memory_note_outline` e
> `memory_file_metadata`.
>
> Vault indexado antes da migração continua funcionando sem `mem index --force`: os
> documentos com `mtime = 0` caem no `updated_at` e vão para o fim de `mem recents`.

> [!warning]
> **ADR-044: decisão mantida, implementação removida.** Em `9bc28c7` os pacotes
> `internal/writer`, `internal/writer/projection` e `internal/policy` foram apagados por
> não terem importadores, e o comando `mem write` nunca chegou a existir. A decisão de
> arquitetura continua válida, mas **a precondição de escrita (`--if-match` / `IfMatch`)
> não é implementada em nenhum caminho hoje** — o parâmetro é aceito e ignorado pelo CLI
> e pelo MCP. Trate a ADR-044 como especificação não implementada, não como estado atual.
> A observer levantou isso na análise de gap de 2026-10-18.

---

## 🗄️ ADRs Obsoletas — viewer Electron (removido em 2026-10-18)

O viewer Electron foi removido do repositório por decisão de produto. O texto
das ADRs abaixo é **mantido como memória histórica**, mas nenhuma delas
representa mais o estado vigente do projeto.

| Número | Título | Status | Data |
| :--- | :--- | :--- | :--- |
| [ADR-037](037-rewrite-viewer-com-vite-vanilla-ts.md) | Rewrite do Viewer com Vite + Vanilla TypeScript | Obsoleta (antes *Superseded by ADR-048*) | 2026-09-17 |
| [ADR-038](038-viewer-site-unico-dataset-fixo.md) | Viewer com Site Único e Dataset Fixo (B3 do ADR-036 Revertido) | Obsoleta | 2026-09-17 |
| [ADR-048](048-viewer-electron-react-vite-shadcn.md) | Viewer Desktop — Electron + React + Vite + shadcn/ui | Obsoleta | 2026-09-21 |
| [ADR-049](049-viewer-restructure-electron-forge-and-src-split.md) | Viewer Restructure — Electron Forge 7 + `electron/` ↔ `src/` split | Obsoleta | 2026-09-22 |
| [ADR-051](051-viewer-chat-llm-provider-and-ipc-streaming.md) | Chat LLM no Viewer — Provider Abstraction + IPC Streaming + RAG via `mem search` | Obsoleta | 2026-09-22 |
| [ADR-053](053-viewer-v3-3-pane-layout-shell.md) | Layout Shell 3-pane + Dock (M1 do Roadmap v3) | Obsoleta | 2026-09-22 |

> [!note]
> **Slot ADR-052 reservado, sem arquivo.** O número 052 é citado por documentos
> do repositório (a spec `feat-chat-language-models` e notas de handoff em
> [`.specs/STATE.md`](../../.specs/STATE.md)), mas **não existe** nenhum
> `docs/adr/052-*.md`. A decisão sobre a extensão multi-provider de chat foi
> escrita junto do código que a implementava, e esse código foi removido com o
> viewer em `b477200`. Nenhum ADR foi inventado para preencher o buraco: se a
> decisão voltar a valer, escreva o `052-*.md` na primeira commit; enquanto isso,
> trate qualquer referência a "ADR-052" como órfã.
>
> O `data-central.json` fixo do ADR-038 e o `resources/chatLanguageModels.default.json`
> foram removidos junto com o viewer. A visualização de grafo
> disponível hoje é a do CLI (`mem graph view`, ADR-020), que segue **Aceita**.

