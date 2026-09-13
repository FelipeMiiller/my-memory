# ADR-004: Modelagem e Travessia de Grafo com SQL Recursivo (CTEs)

- **Date**: 2026-09-12
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: graph, sqlite, sql, knowledge-graph, recursive-cte

## Context and Problem Statement

A busca por similaridade puramente vetorial sofre de "cegueira relacional": ela encontra documentos tematicamente próximos, mas é incapaz de navegar por relacionamentos em múltiplos saltos (*multi-hop reasoning*), como dependências de componentes, hierarquias ou notas conectadas por links conceituais.

Para superar isso (inspirado em ferramentas como Graphify e Obsidian), precisamos de uma estrutura de grafo de conhecimento que conecte nós (documentos, conceitos, tags) e permita navegar por seus vizinhos adjacentes.

## Decision Drivers

- **Zero dependência de bancos dedicados**: Não adicionar Neo4j, Dgraph ou Memgraph.
- **Consultas diretas**: A expansão de vizinhos deve acontecer na mesma transação/conexão da busca vetorial.
- **Profundidade configurável**: Capacidade de buscar 1 salto, 2 saltos ou N saltos de distância.

## Considered Options

- **Opção A: Modelagem em Tabelas Relacionais com Recursive CTEs (`WITH RECURSIVE`)** no SQLite.
- **Opção B: Banco de Grafos Embutido em C/Go** (ex: Cayley ou BadWolf).
- **Opção C: Processamento de Grafos em Memória com NetworkX/Go-Graph**.

## Decision Outcome

Chosen option: **"Opção A: Modelagem com Recursive CTEs"**, because permite representar nós (`graph_nodes`) e arestas (`graph_edges`) diretamente em tabelas relacionais do SQLite e realizar travessias recursivas profundas em SQL puro através da cláusula nativa `WITH RECURSIVE traversal AS (...)`.

### Positive Consequences

- **Atomicidade e Integridade:** As arestas possuem restrições de chave estrangeira (`REFERENCES graph_nodes ON DELETE CASCADE`), garantindo que nós excluídos limpem suas arestas automaticamente.
- **Expansão Dinâmica em SQL:** Uma única query SQL expande vizinhos para todos os chunks retornados na busca vetorial.
- **Indexação por B-Tree:** Índices sobre `source_id` e `target_id` tornam a travessia de 1 ou 2 saltos instantânea (< 1ms).

### Negative Consequences

- **Algoritmos Complexos de Grafo:** Algoritmos muito avançados de ciência de redes (como PageRank global ou Louvain Community Detection) são mais difíceis de expressar puramente em SQL do que em bibliotecas especializadas.

## Pros and Cons of the Options

### Opção A: Recursive CTEs no SQLite ✅ Chosen

- ✅ Nativo do SQLite, sem bibliotecas adicionais.
- ✅ Consistência transacional com o restante dos dados.
- ✅ Extremamente rápido para buscas de vizinhos de 1-2 saltos.
- ❌ Consultas SQL complexas para algoritmos de agrupamento em larga escala.

### Opção B: Banco de Grafo Dedicado Embutido

- ✅ Estrutura de grafos nativa.
- ❌ Fragmenta os dados em dois bancos separados.
- ❌ Aumenta o tamanho do binário e complexidade de manutenção.

## Links

- [Documentação do SQLite sobre Recursive CTEs](https://www.sqlite.org/syntax/recursive-cte.html)
- [ADR-001: Uso de SQLite como Camada Unificada de Dados](001-uso-de-sqlite-como-camada-unificada-de-dados.md)