# ADR-011: Arestas Epistêmicas e God Nodes / Hubs de Conhecimento

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: epistemic-edges, god-nodes, knowledge-hubs, degree-centrality, obsidian, wikilinks, mcp, graphify, akitaonrails

## Context and Problem Statement

Nas versões anteriores do My-Memory, as conexões extraídas do Markdown eram limitadas a links genéricos não-tipados com relação fixa `links_to` e peso uniforme. Isso apresentava limitações conceituais e operacionais para a navegação de agentes de IA:

1. **Ambiguidade Semântica de Conexões:** Não havia distinção entre uma nota que *implementa* um módulo, *depende* de outro, *refuta* uma hipótese ou é meramente categorizada por uma *tag*.
2. **Ausência de Status Epistêmico:** Relações extraídas diretamente da sintaxe de notas (`EXTRACTED`) eram indistinguíveis de conexões inferidas por modelos de linguagem (`INFERRED`), impedindo auditoria ou calibração de confiança (`weight`).
3. **Dificuldade de Descoberta Estrutural para Agentes:** Ao iniciar um diálogo sobre um repositório grande, o agente precisava adivinhar por onde começar a explorar ou fazer buscas genéricas, desconhecendo quais documentos atuam como nós centrais (God Nodes / Hubs de conhecimento) com maior centralidade estrutural de conexões.

## Decision Drivers

- **Inspiração em Referências Especializadas:** Adotar conceitos de *God Nodes* e *Knowledge Hubs* do [Graphify-Labs/graphify](https://github.com/Graphify-Labs/graphify) e status epistêmico do [akitaonrails/ai-memory](https://github.com/akitaonrails/ai-memory).
- **Interoperabilidade com Obsidian:** Suportar sintaxe nativa de wikilinks tipados (`[[relation:Target]]`, `[[Target|rel:relation]]`) e tags Markdown (`#tag`) sem romper wikilinks convencionais (`[[Target]]`).
- **Desempenho em O(E):** Calcular centralidade de grau (*degree centrality*) de forma eficiente tanto no SQLite quanto no PostgreSQL usando CTEs SQL simples e união de arestas sem complexidade de bibliotecas externas de grafos.
- **Exposição MCP para Agentes:** Disponibilizar ferramenta nativa `memory_get_hubs` para que modelos de IA possam consultar os pontos de entrada cruciais da base de conhecimento.
- **Interface de Linha de Comando:** Oferecer o comando `mem hubs` para visualização tabular formatada dos principais nós centrais.

## Decision Outcome

Adotou-se uma arquitetura unificada de arestas epistêmicas tipadas e cálculo de centralidade de grau em duas camadas:

1. **Parser Sintático com Suporte a Conexões Tipadas:**
   - Adicionada struct `EdgeConnection` no pacote `parser` com campos `Source`, `Target`, `Relation`, `EpistemicStatus` e `Weight`.
   - Suporte a prefixos de relação (`[[implements:Auth]]`), aliases tipados (`[[Database|rel:depends_on]]`), tags Obsidian (`#security` -> aresta `tagged_as`) e metadados frontmatter (`relation: Target` / `rel: Target`).
   - Preservação de 100% de retrocompatibilidade com `OutgoingLinks []string` para chamadas legadas.
2. **Schema Relacional Estendido no SQLite e PostgreSQL:**
   - Adicionadas colunas `epistemic_status TEXT NOT NULL DEFAULT 'EXTRACTED'` e `weight REAL NOT NULL DEFAULT 1.0` na tabela `graph_edges` com migrações idempotentes automáticas (`ADD COLUMN IF NOT EXISTS`).
   - Implementado método `InsertEdgeWithProps(ctx, repo, sourceID, targetID, relation, epistemicStatus, weight)` e adaptado `InsertEdge` como wrapper padrão.
3. **Cálculo de God Nodes / Hubs via CTE de Grau:**
   - Consulta CTE que une direções de entrada (`in_degree`) e saída (`out_degree`), calculando `total_degree = in_degree + out_degree` agrupado por nó e ordenado de forma decrescente:
   ```sql
   WITH degrees AS (
       SELECT source_id AS node_id, 0 AS in_cnt, 1 AS out_cnt FROM graph_edges WHERE ($1 = '' OR repository = $1)
       UNION ALL
       SELECT target_id AS node_id, 1 AS in_cnt, 0 AS out_cnt FROM graph_edges WHERE ($1 = '' OR repository = $1)
   )
   SELECT d.node_id, COALESCE(n.name, d.node_id) AS name,
          SUM(d.in_cnt) AS in_degree,
          SUM(d.out_cnt) AS out_degree,
          COUNT(*) AS total_degree
   FROM degrees d
   LEFT JOIN graph_nodes n ON n.id = d.node_id
   GROUP BY d.node_id, n.name
   ORDER BY total_degree DESC, in_degree DESC
   LIMIT $2;
   ```
4. **Ferramenta MCP `memory_get_hubs`:**
   - Expõe parâmetros `{ repository: string, top: integer }` retornando lista formatada de nós centrais com identificador, nome/título, graus de entrada, saída e total.
5. **Comando CLI `mem hubs`:**
   - Exibe tabela ASCII formatada com colunas `Rank`, `Nó / Documento`, `Entradas`, `Saídas` e `Total Grau`.

### Positive Consequences

- **Capacidade Semântica Avançada:** Modelos e usuários agora diferenciam relações conceituais ricas (ex: `implements`, `depends_on`, `cites`, `tagged_as`) de meros links genéricos.
- **Navegação Orientada por Hubs:** Agentes de IA agora conseguem consultar `memory_get_hubs` no início de uma sessão de exploração para identificar imediatamente os nós arquiteturais mais referenciados do repositório.
- **Retrocompatibilidade e Desempenho:** Nenhuma dependência externa pesada de grafos foi adicionada; as consultas de grau executam em milissegundos sobre os índices de chave primária e chaves estrangeiras existentes.

### Negative Consequences

- Indexação armazena ligeiramente mais arestas por nota quando tags e múltiplas relações tipadas estão presentes, custo negligível dado o tamanho enxuto das linhas da tabela `graph_edges`.

## Links e Referências

- [Graphify-Labs/graphify](https://github.com/Graphify-Labs/graphify)
- [akitaonrails/ai-memory](https://github.com/akitaonrails/ai-memory)
- [docs/REFERENCES.md](../REFERENCES.md)
- [docs/adr/004-modelagem-de-grafo-com-recursive-ctes.md](004-modelagem-de-grafo-com-recursive-ctes.md)
- [docs/adr/005-markdown-com-wikilinks-como-fonte-de-verdade.md](005-markdown-com-wikilinks-como-fonte-de-verdade.md)
- [docs/adr/006-integracao-com-agentes-de-ia-via-mcp.md](006-integracao-com-agentes-de-ia-via-mcp.md)
- [docs/adr/008-interoperabilidade-obsidian-flavored-markdown-e-json-canvas.md](008-interoperabilidade-obsidian-flavored-markdown-e-json-canvas.md)
- [docs/adr/010-cache-incremental-de-indexacao-com-sha256.md](010-cache-incremental-de-indexacao-com-sha256.md)
