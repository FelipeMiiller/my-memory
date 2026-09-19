---
title: "architecture"
category: resource
summary: "Visão arquitetural do my-memory: SQLite como camada unificada, FTS5 + sqlite-vec + grafo recursivo, MCP como superfície de integração."
tags: [concept, architecture, design, system-design]
---

# architecture

Visão arquitetural consolidada do my-memory: SQLite unificado + FTS5 (BM25) + sqlite-vec (embeddings) + SQL recursivo (grafo) + MCP (integração com agentes de IA). Single-binary, single-file deployment.

## Camadas

1. **Storage unificado** — SQLite com `sqlite-vec` + `chunks_turboquant` (4-bit) + `graph_nodes`/`graph_edges`
2. **Parser** — Markdown Obsidian Flavored → wikilinks, tags, edges (`links_to`, `tagged_as`, `implements`)
3. **Indexação** — SHA-256 incremental, file watcher, sync/async paths
4. **Higiene** — `mem doctor` (Health Score 0-100, dead links, orphans), pruning automático
5. **MCP** — transporte HTTP/SSE, 13+ tools (`memory_search`, `memory_get_drift`, `memory_doctor`, etc)

## Onde aparece neste vault

- [[COMO_FUNCIONA]] — overview top-down
- [[docs/adr/001-uso-de-sqlite-como-camada-unificada-de-dados|ADR-001: SQLite como camada unificada]]
- [[docs/adr/002-adocao-de-go-como-linguagem-principal|ADR-002: Adoção de Go]]
- [[docs/adr/003-compressao-vetorial-de-4-bit-via-turboquant|ADR-003: TurboQuant 4-bit]]
- [[docs/adr/004-modelagem-de-grafo-com-recursive-ctes|ADR-004: Grafo recursivo]]
- [[docs/adr/006-integracao-com-agentes-de-ia-via-mcp|ADR-006: MCP]]
- [[docs/adr/013-higiene-de-grafo-pruning-e-doctor|ADR-013: Higiene de grafo + Doctor]]
- [[docs/adr/035-embedder-embutido-com-fallback-onnx-minilm|ADR-035: Embedder ONNX]]
