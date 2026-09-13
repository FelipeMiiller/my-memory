# ADR-005: Markdown com [[Wikilinks]] como Entrada e Grafo Humano

- **Date**: 2026-09-12
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: obsidian, parser, markdown, pkm, knowledge-graph

## Context and Problem Statement

Sistemas de inteligência artificial precisam de fontes de verdade organizadas e sustentáveis mantidas por equipes humanas. Formatos binários ou proprietários dificultam a edição manual por desenvolvedores e o versionamento no Git.

O ecossistema do **Obsidian** popularizou o uso de notas Markdown locais com links bidirecionais explícitos no formato `[[Nome da Nota]]` e tags `#tag`, criando um grafo semântico de altíssimo sinal e ruído zero, desenhado intencionalmente por pessoas.

## Decision Drivers

- **Legibilidade e Portabilidade**: O conteúdo deve ser legível por humanos e ferramentas padrão de texto plano.
- **Sinal de Relacionamento Semântico Alto**: Arestas criadas por humanos devem ser preservadas como conexões explícitas no grafo.
- **Interoperabilidade com o Git**: Facilidade de fazer diffs limpos e revisões via Pull Request.

## Considered Options

- **Opção A: Markdown Puro com Suporte a [[Wikilinks]] e #Tags (Estilo Obsidian)**.
- **Opção B: Ontologias RDF / OWL ou Triplestores Semânticos** (formato Turtle / JSON-LD).
- **Opção C: Extração 100% Automatizada por LLM** sem links humanos prévios.

## Decision Outcome

Chosen option: **"Opção A: Markdown Puro com [[Wikilinks]]"**, because permite que desenvolvedores e pesquisadores utilizem editores de texto convencionais ou o Obsidian para criar suas notas e decisões arquiteturais. O parser em Go extrai automaticamente os links `[[...]]` e tags `#...`, convertendo-os em nós e arestas no SQLite sem nenhum custo de token ou latência de LLM.

### Positive Consequences

- **Soberania dos Dados:** O projeto não fica preso a nenhuma ferramenta proprietária; se o `my-memory` deixar de existir, todos os arquivos `.md` continuam intactos e utilizáveis.
- **Grafo Sem Alucinações:** As arestas geradas a partir de `[[wikilinks]]` representam intenções reais de conexão estabelecidas por seres humanos.
- **Baixo Custo de Processamento:** Parsing via expressões regulares otimizadas em Go roda em centenas de notas em menos de 100ms.

### Negative Consequences

- **Esforço Humano:** Se o desenvolvedor esquecer de linkar explicitamente dois tópicos, a aresta do grafo não é criada (embora a busca vetorial ainda consiga encontrar a proximidade semântica).

## Pros and Cons of the Options

### Opção A: Markdown com [[Wikilinks]] ✅ Chosen

- ✅ Arquivos de texto legíveis em qualquer plataforma e amigáveis ao Git.
- ✅ Zero custo computacional para extrair arestas de alta qualidade.
- ✅ Compatibilidade direta com cofres existentes do Obsidian.
- ❌ Depende da disciplina do usuário para criar links manuais.

### Opção B: RDF / OWL / Triplestores

- ✅ Máximo rigor formal para ontologias e web semântica.
- ❌ Complexidade excessiva para o dia a dia de desenvolvedores.
- ❌ Pouco ergonômico para escrita humana rápida.

### Opção C: Extração 100% Automatizada via LLM

- ✅ Zero esforço do usuário.
- ❌ Custo computacional e de tokens alto a cada modificação.
- ❌ Sujeito a alucinações e arestas espúrias no grafo.

## Links

- [Obsidian Help: Internal Links](https://help.obsidian.md/Linking+notes+and+files/Internal+links)
- [ADR-004: Modelagem e Travessia de Grafo com Recursive CTEs](004-modelagem-de-grafo-com-recursive-ctes.md)