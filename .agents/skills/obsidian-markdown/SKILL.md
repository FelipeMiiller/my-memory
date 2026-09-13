---
name: obsidian-markdown
description: Regras e padrões para criar e editar notas no formato Obsidian Flavored Markdown (.md) com wikilinks, embeds, callouts, tags e propriedades de frontmatter YAML. Use ao manipular documentos de memória ou notas no cofre do My-Memory.
---

# Obsidian Flavored Markdown Skill

Esta habilidade orienta agentes de IA na manipulação de notas em **Obsidian Flavored Markdown** para garantir integridade estrutural e semântica com o grafo do My-Memory.

## Estrutura Padrão de uma Nota

Toda nota criada ou atualizada deve conter um bloco inicial de **frontmatter YAML**:

```markdown
---
title: Nome da Nota
tags:
  - categoria
  - subcategoria
aliases:
  - Sinônimo 1
  - Sinônimo 2
status: ativo
---

# Nome da Nota

Conteúdo em Markdown com conexões explícitas.
```

## Wikilinks (Conexões de Grafo)

Use sempre o padrão de links internos do Obsidian:

| Sintaxe | Função no My-Memory |
|---|---|
| `[[Nome da Nota]]` | Cria aresta direcionada no grafo para a nota de destino |
| `[[Nome da Nota\|Texto Exibido]]` | Cria aresta no grafo preservando o rótulo de leitura |
| `[[Nome da Nota#Seção]]` | Vincula a uma seção específica dentro da nota alvo |
| `[[Nome da Nota#^bloco-id]]` | Vincula a um parágrafo ou bloco específico com block reference |
| `[[#Seção Local]]` | Link de navegação interna (não cria nó externo espúrio) |
| `![[Nota ou Anexo.png]]` | Transclusão ou incorporação visual inline |

## Tags

- **Frontmatter**: declaradas na lista `tags:` (sem `#`).
- **Inline**: no corpo do texto usando `#tag` ou `#hierarquia/subtag` (sempre iniciando com letra).

## Callouts Padronizados

Destaque pontos críticos usando callouts do Obsidian:

```markdown
> [!note]
> Nota explicativa de contexto.

> [!tip]
> Sugestão ou recomendação técnica.

> [!important]
> Requisito crítico ou restrição mandatória.

> [!warning]
> Alerta sobre efeitos colaterais ou quebras de compatibilidade.
```
