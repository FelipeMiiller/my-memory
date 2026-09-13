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