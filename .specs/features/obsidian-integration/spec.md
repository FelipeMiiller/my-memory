# Feature: obsidian-integration

## Visão Geral
Esta feature provê interoperabilidade completa com o padrão Obsidian Flavored Markdown e a especificação aberta JSON Canvas 1.0 (baseados em `kepano/obsidian-skills`).

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
   - Documentações de skill em `.agents/skills/obsidian-markdown` e `.agents/skills/json-canvas`.
