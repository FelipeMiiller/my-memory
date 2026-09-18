# Índice Geral: Documentação & Agent Skills

Este documento é o **mapa central de conhecimento** do projeto **My-Memory**. Ele conecta a documentação técnica, as decisões de arquitetura (ADRs) e as habilidades de IA (*Agent Skills*) instaladas para desenvolvedores e agentes autônomos.

---

## 🗺 1. Mapa da Documentação Técnica (`docs/`)

| Documento | Foco | Descrição |
| :--- | :--- | :--- |
| **[`docs/ARCHITECTURE.md`](ARCHITECTURE.md)** | Arquitetura | Diagrama de fluxo de dados, DDL do SQLite, índices FTS5, vetores e consultas recursivas em grafo. |
| **[`docs/CENTRAL_VAULT.md`](CENTRAL_VAULT.md)** | Cofre Central & Federação | Arquitetura do **Global Brain**, setup do `mem setup --central`, URIs `memory://`, fallback FTS quando Ollama está offline e ciclo de promoção do conhecimento. |
| **[`docs/TURBOQUANT.md`](TURBOQUANT.md)** | Matemática & Algoritmo | Teoria da quantização vetorial de 4-bit (Google DeepMind, ICLR 2026), rotações de Householder e produto escalar não-viesado. |
| **[`docs/BENCHMARKS.md`](BENCHMARKS.md)** | Performance & Métricas | Relatório empírico de micro-benchmarks (TurboQuant 4-bit, RRF, hashing SHA-256 e parsing). |
| **[`docs/CLI_GUIDE.md`](CLI_GUIDE.md)** | Operação | Manual prático de comandos da CLI (`mem index`, `mem search`, `mem bench`, `mem insights`). |
| **[`docs/REPOSITORY_BRAIN.md`](REPOSITORY_BRAIN.md)** | Integração com IA | Como utilizar o `my-memory` como memória de contexto dentro de projetos via **Model Context Protocol (MCP)**. |
| **[`docs/AGENT_INTEGRATION_GUIDE.md`](AGENT_INTEGRATION_GUIDE.md)** | Guia para Agentes | Manual completo de integração, comandos CLI, servidor MCP e instruções para `AGENTS.md`. |
| **[`docs/REFERENCES.md`](REFERENCES.md)** | Arte Prévia & Referências | Referências técnicas e ecossistemas que inspiram e refinam o projeto (`graphify`, `ai-memory`, `obsidian-skills`, `codegraph`, `openviking`, `atlas`). |

---

## 📋 2. Registros de Decisão de Arquitetura (`docs/adr/`)

Decisões registradas no formato padronizado **MADR**:

* **[`docs/adr/README.md`](adr/README.md)** — Índice consolidado de ADRs
* **[`ADR-001`](adr/001-uso-de-sqlite-como-camada-unificada-de-dados.md)** — Uso de SQLite como Camada Unificada de Dados
* **[`ADR-002`](adr/002-adocao-de-go-como-linguagem-principal.md)** — Adoção de Go como Linguagem Principal de Implementação
* **[`ADR-003`](adr/003-compressao-vetorial-de-4-bit-via-turboquant.md)** — Compressão Vetorial de 4-bit via TurboQuant
* **[`ADR-004`](adr/004-modelagem-de-grafo-com-recursive-ctes.md)** — Modelagem e Travessia de Grafo com SQL Recursivo (CTEs)
* **[`ADR-005`](adr/005-markdown-com-wikilinks-como-fonte-de-verdade.md)** — Markdown e [[Wikilinks]] como Entrada e Grafo Humano
* **[`ADR-006`](adr/006-integracao-com-agentes-de-ia-via-mcp.md)** — Integração com Agentes de IA via Model Context Protocol (MCP)
* **[`ADR-007`](adr/007-suporte-opcional-a-postgresql-com-pgvector-e-multi-repositorio.md)** — Suporte Opcional a PostgreSQL com pgvector e Referência Multi-Repositório
* **[`ADR-008`](adr/008-interoperabilidade-obsidian-flavored-markdown-e-json-canvas.md)** — Interoperabilidade com Obsidian Flavored Markdown e JSON Canvas 1.0
* **[`ADR-009`](adr/009-busca-hibrida-com-reciprocal-rank-fusion-rrf.md)** — Busca Híbrida com Reciprocal Rank Fusion (RRF)
* **[`ADR-010`](adr/010-cache-incremental-de-indexacao-com-sha256.md)** — Cache Incremental de Indexação com SHA-256
* **[`ADR-011`](adr/011-arestas-epistemicas-e-god-nodes.md)** — Arestas Epistêmicas e God Nodes / Hubs de Conhecimento
* **[`ADR-012`](adr/012-benchmarks-e-conexoes-inesperadas.md)** — Benchmarks de Performance e Conexões Inesperadas (Surprising Connections)
* **[`ADR-013`](adr/013-higiene-de-grafo-pruning-e-doctor.md)** — Higiene de Grafo, Pruning Incremental e Linter Doctor
* **[`ADR-014`](adr/014-centralidade-de-grafo-com-pagerank-ponderado.md)** — Centralidade de Grafo com PageRank Ponderado
* **[`ADR-015`](adr/015-decaimento-temporal-exponencial-na-busca-hibrida.md)** — Decaimento Temporal Exponencial na Busca Híbrida
* **[`ADR-016`](adr/016-configuracao-declarativa-e-auto-scoping-de-vault.md)** — Configuração Declarativa e Auto-Scoping de Vault
* **[`ADR-017`](adr/017-indexacao-continua-com-file-watcher-e-git-hooks.md)** — Indexação Contínua em Tempo Real com File Watcher e Git Hooks
* **[`ADR-018`](adr/018-padrao-compile-not-retrieve-e-escrita-bilateral-mcp.md)** — Padrão Compile-not-Retrieve e Escrita Bilateral na Memória via MCP e CLI
* **[`ADR-019`](adr/019-versionamento-semantico-e-tagging-ci.md)** — Versionamento Semântico Automatizado e Criação de Tags no CI
* **[`ADR-020`](adr/020-visualizador-interativo-de-grafo-em-html-svg.md)** — Visualizador Interativo de Grafo em HTML/SVG Standalone
* **[`ADR-021`](adr/021-servidor-mcp-com-transporte-http-sse.md)** — Servidor MCP com Transporte HTTP e Server-Sent Events (SSE)
* **[`ADR-022`](adr/022-deteccao-de-comunidades-e-clusters-no-grafo.md)** — Detecção de Comunidades e Clusters no Grafo de Conhecimento
* **[`ADR-023`](adr/023-analise-de-impacto-e-raio-de-destruicao-blast-radius.md)** — Análise de Impacto e Raio de Destruição (Blast Radius Analysis)
* **[`ADR-024`](adr/024-visualizacao-cirurgica-em-3-colunas-triptych-inspector.md)** — Visualização Cirúrgica em 3 Colunas (Triptych Node Inspector)
* **[`ADR-025`](adr/025-progressive-context-loading-e-taxonomia-de-memoria.md)** — Carregamento Progressivo de Contexto e Taxonomia de Memória
* **[`ADR-026`](adr/026-auto-wiring-e-instalacao-zero-touch-de-ferramentas-de-ia.md)** — Auto-Wiring e Instalação Zero-Touch de Ferramentas de IA (`mem install` / `mem setup`)
* **[`ADR-027`](adr/027-descoberta-de-rotas-e-caminho-minimo-no-grafo.md)** — Descoberta de Rotas e Caminho Mínimo no Grafo de Conhecimento (`mem path` / `memory_find_path`)
* **[`ADR-028`](adr/028-staleness-banners-e-deteccao-de-desatualizacao.md)** — Staleness Banners e Detecção de Desatualização de Conhecimento (`mem status`)
* **[`ADR-029`](adr/029-context-packager-e-subgraph-bundle.md)** — Context Packager e Subgraph Bundle com Orçamento de Tokens (`mem pack` / `memory_pack_context`)
* **[`ADR-030`](adr/030-deep-linking-e-navegacao-de-editores.md)** — Deep Linking e Integração de Navegação com Editores (`mem open` e URIs `obsidian://` / `vscode://`)
* **[`ADR-031`](adr/031-semantic-drift-e-deteccao-de-desvio-codigo-memoria.md)** — Semantic Drift e Detecção de Desvio Código-Memória (`mem drift` e `memory_get_drift`)
* **[`ADR-032`](adr/032-instalador-universal-one-liner.md)** — Instalador Universal One-Liner para Windows, Linux e macOS
* **[`ADR-033`](adr/033-federated-central-vault-and-repo-identity.md)** — Arquitetura Federada de Vault Central e Identidade Imutável de Repositório (`repo_id`)
* **[`ADR-034`](adr/034-protocolo-canonico-federado-e-wikilinks-cross-vault.md)** — Protocolo Canônico Federado e Wikilinks Cross-Vault (`memory://`)
* **[`ADR-035`](adr/035-embedder-embutido-com-fallback-onnx-minilm.md)** — Embedder Embutido com Fallback Automático (`builtin` / `all-MiniLM-L6-v2` via ONNX Runtime) — *Proposed*



---

## 🤖 3. Habilidades de IA Instaladas (`.agents/skills/`)

Habilidades de engenharia de software configuradas no repositório através do `@tech-leads-club/agent-skills`:

### 3.1. `create-adr`
* **Local:** [`.agents/skills/create-adr/`](../.agents/skills/create-adr/)
* **Descrição:** Guia estruturado para documentar escolhas arquiteturais significativas no formato MADR.
* **Gatilhos para IAs:** *"crie um adr"*, *"documente essa decisão"*, *"registre por que escolhemos X"*.
* **Arquivos chave:**
  - [`.agents/skills/create-adr/SKILL.md`](../.agents/skills/create-adr/SKILL.md): Instruções operacionais e templates.
  - [`.agents/skills/create-adr/README.md`](../.agents/skills/create-adr/README.md): Documentação da skill.

### 3.2. `tlc-spec-driven`
* **Local:** [`.agents/skills/tlc-spec-driven/`](../.agents/skills/tlc-spec-driven/)
* **Descrição:** Metodologia de desenvolvimento orientada a especificações com 4 fases adaptativas (*Specify → Design → Tasks → Execute*), validação em notação EARS e gates determinísticos em Python.
* **Gatilhos para IAs:** *"especifique a feature"*, *"planeje tarefas"*, *"implemente com verificação"*, *"valide requisitos"*.
* **Estrutura interna:**
  - `references/`: Guias de especificação, design, testes e sub-agentes.
  - `scripts/`: Validadores automáticos em Python (`validate_spec.py`, `validate_tasks.py`, `check_commit.py`, etc.).

### 3.3. `my-memory-format`
* **Local:** [`.agents/skills/my-memory-format/`](../.agents/skills/my-memory-format/)
* **Descrição:** Criação de **qualquer tipo de memória** do My-Memory — Markdown Obsidian Flavored (`.md` com wikilinks estruturados, frontmatter YAML `tags`/`aliases`, callouts e embeds) **e** mapas espaciais JSON Canvas 1.0 (`.canvas` com nós e arestas).
* **Gatilhos para IAs:** *"crie uma nota"*, *"adicione wikilinks"*, *"formate em obsidian"*, *"extraia frontmatter"*, *"exporte para canvas"*, *"crie um json canvas"*, *"mapa mental espacial"*.

> Substitui as antigas skills `memory-md` + `json-canvas`, agora unificadas. Cobre todos os formatos de memória do projeto num único source-of-truth.

---

## 🧭 4. Guia para Agentes de IA (Como Navegar no Repositório)

Ao atuar neste repositório:
1. **Antes de propor decisões de arquitetura:** Consulte a pasta [`docs/adr/`](adr/) para verificar precedentes estabelecidos.
2. **Ao registrar novas decisões:** Use a skill [`create-adr`](../.agents/skills/create-adr/SKILL.md) e siga o formato MADR.
3. **Ao planejar e implementar novas features:** Use a metodologia [`tlc-spec-driven`](../.agents/skills/tlc-spec-driven/SKILL.md) mantendo tarefas atômicas e rastreabilidade de requisitos.
4. **Ao alterar a estrutura do banco ou CLI:** Consulte [`docs/ARCHITECTURE.md`](ARCHITECTURE.md) e [`docs/TURBOQUANT.md`](TURBOQUANT.md).