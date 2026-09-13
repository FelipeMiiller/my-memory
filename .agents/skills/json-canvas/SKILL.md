---
name: json-canvas
description: Criação, validação e manipulação de arquivos JSON Canvas 1.0 (.canvas) no ecossistema Obsidian e My-Memory. Use ao gerar mapas mentais, fluxos de raciocínio ou visualizações espaciais do grafo de memória.
---

# JSON Canvas 1.0 Skill

Esta habilidade descreve a especificação e as regras para gerar arquivos `.canvas` compatíveis com o Obsidian e com a ferramenta de exportação do My-Memory.

## Estrutura do Arquivo

O arquivo `.canvas` é um JSON contendo duas coleções principais:

```json
{
  "nodes": [],
  "edges": []
}
```

## Regras de Nós (`nodes`)

Cada nó deve possuir:
- `id`: string única de **16 caracteres hexadecimais em minúsculo** (ex: `"6f0ad84f44ce9c17"`).
- `type`: `"file"`, `"text"`, `"link"` ou `"group"`.
- `x`, `y`: inteiros representando coordenadas espaciais em pixels (origem no canto superior esquerdo).
- `width`, `height`: dimensões inteiras em pixels.
- `color`: opcional, presets numéricos `"1"` (vermelho), `"2"` (laranja), `"3"` (amarelo), `"4"` (verde), `"5"` (ciano), `"6"` (roxo) ou cor hex (`"#RRGGBB"`).

### Nó de Arquivo (`type: "file"`)
Vincula diretamente a uma nota Markdown no vault:
```json
{
  "id": "a1b2c3d4e5f67890",
  "type": "file",
  "x": 0,
  "y": 0,
  "width": 300,
  "height": 120,
  "file": "Arquitetura.md",
  "color": "1"
}
```

### Nó de Texto (`type: "text"`)
Renderiza conteúdo livre em Markdown:
```json
{
  "id": "b2c3d4e5f6789012",
  "type": "text",
  "x": 380,
  "y": 0,
  "width": 260,
  "height": 100,
  "text": "### Resumo\nExplicação sintetizada pelo agente.",
  "color": "4"
}
```

## Regras de Arestas (`edges`)

Arestas conectam nós existentes:
- `id`: string hex de 16 caracteres.
- `fromNode`: ID do nó de origem.
- `toNode`: ID do nó de destino.
- `fromSide`: `"top"`, `"right"`, `"bottom"` ou `"left"`.
- `toSide`: `"top"`, `"right"`, `"bottom"` ou `"left"`.
- `toEnd`: `"arrow"` (ou `"none"`).
- `label`: texto opcional descrevendo o relacionamento.

```json
{
  "id": "c3d4e5f678901234",
  "fromNode": "a1b2c3d4e5f67890",
  "toNode": "b2c3d4e5f6789012",
  "fromSide": "right",
  "toSide": "left",
  "toEnd": "arrow",
  "label": "detalha"
}
```

## Checklist de Validação
1. Todos os IDs de nós e arestas devem ser únicos.
2. Todo `fromNode` e `toNode` de uma aresta DEVE corresponder a um nó presente em `nodes`.
3. Evite nós sobrepostos; garanta espaçamento mínimo de 80 pixels.
