# Feature: obsidian-integration

## Problem Statement

O My-Memory precisa interoperar com o padrão Obsidian Flavored Markdown e com a
especificação aberta JSON Canvas 1.0 (baseados em `kepano/obsidian-skills`): ler e
escrever notas que o Obsidian abre sem perda, e exportar o grafo para o formato
JSON Canvas.

## Visão Geral
Esta feature provê interoperabilidade completa com o padrão Obsidian Flavored Markdown e a especificação aberta JSON Canvas 1.0 (baseados em `kepano/obsidian-skills`).

## Out of Scope

Tudo o que não está enumerado em [Requisitos Funcionais](#requisitos-funcionais) fica
fora do escopo desta spec.

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale |
| :--- | :--- | :--- |
| Padrão de interoperabilidade | Obsidian Flavored Markdown + JSON Canvas 1.0 | É o que os requisitos funcionais enumeram: parser, canvas, integração MCP, CLI e Agent Skills |

Open questions: none

## User Stories

### P1: Notas no padrão Obsidian e grafo exportável em JSON Canvas

**User Story**: Como usuário do Obsidian, quero que as notas do vault sejam lidas e
escritas no padrão Obsidian Flavored Markdown, e que eu possa exportar o grafo em
JSON Canvas 1.0.

Os critérios de aceite são os requisitos funcionais listados abaixo.

## Requisitos Funcionais
1. **Parser de Markdown (`internal/parser`)**:
   - Extração de frontmatter YAML: suporte a propriedades como `tags`, `aliases`, `title`, etc.
   - Normalização de Wikilinks:
     - `[[Nome da Nota]]` -> Alvo: `Nome da Nota`
     - `[[Nome da Nota|Rótulo]]` -> Alvo: `Nome da Nota`, Rótulo: `Rótulo`
     - `[[Nome da Nota#Seção]]` -> Alvo: `Nome da Nota`, Âncora: `#Seção`
     - `[[Nome da Nota#Seção|Rótulo]]` -> Alvo: `Nome da Nota`, Âncora: `#Seção`, Rótulo: `Rótulo`
     - `[[Nome da Nota#^block-id]]` -> Alvo: `Nome da Nota`, Bloco: `^block-id`
     - `[[#Seção na mesma nota]]` -> Marcado como link interno sem gerar nó externo espúrio
     - Embeds: `![[Nota]]` ou `![[imagem.png]]` identificados como transclusão
   - Unificação de tags de frontmatter (`tags`) e inline (`#tag`, `#hierarquia/subtag`).

2. **Gerador JSON Canvas 1.0 (`internal/canvas`)**:
   - Conforme à especificação oficial do [JSON Canvas 1.0](https://jsoncanvas.org/spec/1.0/).
   - Estrutura contendo nós (`file`, `text`) e arestas direcionadas com setas.
   - Geração de IDs hexadecimais aleatórios únicos de 16 caracteres.
   - Algoritmo de layout automático espacial (grade ou radial) evitando sobreposição de nós (espaçamento >= 80px).
   - Função `FromNeighbors(centerNode string, neighbors []string) *Canvas`.
   - Serialização JSON e escrita em arquivo `.canvas`.

3. **Integração MCP (`internal/mcp`)**:
   - Registro da ferramenta `memory_export_canvas` em `tools/list`.
   - Handler para gerar canvas a partir de `node_id`, `max_depth` e `repository`.

4. **CLI (`cmd/mem`)**:
   - Subcomando `mem export --canvas <node_id> [--depth 1] [--out <arquivo.canvas>]`.

5. **Agent Skills**:
   - Documentação da skill unificada em `.agents/skills/my-memory-format` (absorve as antigas `memory-md` + `json-canvas`).

---

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Where | Status |
| :--- | :--- | :--- | :--- |
| OBS-01 | Requisito 1 — extração de frontmatter, normalização de wikilinks, embeds e unificação de tags | `internal/parser` | Verified |
| OBS-02 | Requisito 2 — gerador JSON Canvas 1.0 com nós, arestas, layout automático e `FromNeighbors` | `internal/canvas` | Verified |
| OBS-03 | Requisito 3 — integração MCP com a ferramenta `memory_export_canvas` | `internal/mcp` | Verified |
| OBS-04 | Requisito 4 — subcomando `mem export --canvas <node_id>` | `cmd/mem` | Verified |
| OBS-05 | Requisito 5 — Agent Skills com o formato unificado | `.agents/skills/my-memory-format` | Verified |

**Coverage**: 5 total, 5 mapeados, 0 unmapped.
