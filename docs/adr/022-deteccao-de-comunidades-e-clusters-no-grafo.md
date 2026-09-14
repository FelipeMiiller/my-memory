# ADR-022: Detecção de Comunidades e Clusters no Grafo de Conhecimento

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: graph, community-detection, lpa, label-propagation, modularity, newman-girvan, graphview, clustering, cli, mcp

## Context and Problem Statement

O **My-Memory** disponibiliza análise estrutural e métricas de centralidade individual, como grau de conexões (God Nodes - ADR-011) e autoridade topológica ponderada (PageRank - ADR-014).

Entretanto, métricas puramente locais ou de centralidade individual deixavam lacunas macroestruturais:
1. **Ausência de Visão Modular:** O sistema não detectava agrupamentos orgânicos ou "ilhas conceituais" de conhecimento formadas pela densidade de links entre notas correlatas.
2. **Navegação Visual Monocromática por Domínio:** No visualizador interativo (ADR-020), as notas eram coloridas somente pelo tipo estático (`concept`, `decision`), impossibilitando distinguir limites entre subsistemas de código ou tópicos temáticos de negócio.
3. **Exploração Fragmentada por Agentes:** Agentes de IA via MCP eram forçados a realizar buscas nota a nota ou expansões cegas de vizinhos sem ter uma ferramenta para consultar os módulos ou clusters temáticos consolidados do repositório.

## Decision Drivers

- **Algoritmo em Go Puro (Zero Dependências Externas)**: O cálculo deve ser 100% autônomo, dispensando bibliotecas em C/CGO ou pacotes externos pesados de grafos, mantendo compilação estática multiplataforma.
- **Determinismo Estrito e Reprodutibilidade**: Execuções repetidas sobre o mesmo grafo devem convergir sempre para exatamente os mesmos clusters, empregando ordenação determinística e desempate lexicográfico.
- **Respeito aos Pesos Epistêmicos**: Conexões extraídas explicitamente (`EXTRACTED` peso 1.0) devem exercer maior atração de comunidade do que inferências (`INFERRED` peso 0.6) ou tags (`TAG` peso 0.3).
- **Métrica Qualitativa de Coesão (Modularidade Newman-Girvan \(Q\))**: Avaliar formalmente se a partição do grafo é estatisticamente densa e modular.
- **Ergonomia CLI e MCP**: Disponibilizar o subcomando `mem clusters [--min-size 2] [--json]` e a ferramenta MCP `memory_get_clusters`.

## Decision Outcome

Adotou-se o algoritmo **Weighted Label Propagation Algorithm (LPA)** com cálculo de **Modularidade Newman-Girvan \(Q\)**:

1. **Algoritmo LPA Determinístico (`internal/graph/community.go`):**
   - **Inicialização:** Cada nó inicia como seu próprio cluster (`labels[u] = u`). Os nós são ordenados lexicograficamente.
   - **Propagação Ponderada:** Em cada iteração, para cada nó $u$, calcula-se a atração de cada rótulo vizinho ponderada pelos pesos das arestas epistêmicas ($w_{uv}$).
   - **Desempate Determinístico:** Em caso de empate de atração ponderada máxima entre múltiplos rótulos vizinhos, escolhe-se lexicograficamente o menor identificador (`candidate[0]`).
   - **Convergência Assíncrona:** A atualização é realizada in-place e encerra quando nenhum nó altera de rótulo ou atinge o limite máximo (`MaxIterations = 50`).

2. **Cálculo da Modularidade Newman-Girvan \(Q\):**
   - Calculada em tempo linear $O(V + E)$ sobre a partição:
     $$Q = \sum_{c \in C} \left[ \frac{\Sigma_{in}(c)}{2m} - \left( \frac{\Sigma_{tot}(c)}{2m} \right)^2 \right]$$
     onde $\Sigma_{in}(c)$ é a soma dos pesos das arestas internas da comunidade $c$, $\Sigma_{tot}(c)$ é o grau ponderado total dos nós de $c$, e $2m$ é a soma total de graus do grafo.
   - Valores de $Q > 0.3$ indicam forte coesão temática interna em relação ao esperado por acaso.

3. **Enriquecimento do GraphView (`internal/graphview/`):**
   - Atribuição automática de `CommunityID`, `CommunityLabel` (derivado do nó líder com maior PageRank local) e `CommunityColor` (paleta categórica de 12 cores harmônicas).
   - Inclusão de `Stats.CommunityCount` e `Stats.Modularity` no modelo e nos metadados do visualizador HTML/SVG.

4. **Interface CLI (`cmd/mem/clusters.go`):**
   - `mem clusters`: Exibe tabela alinhada no terminal contendo ID, Nó Líder, Tamanho, Tipo Dominante e Membros.
   - Flag `--min-size <N>`: Oculta nós isolados ou clusters menores que $N$ (padrão: 2).
   - Flag `--json`: Emite payload JSON estruturado com métricas e lista de comunidades.

5. **Ferramenta MCP (`memory_get_clusters`):**
   - Registrada no catálogo do servidor MCP (`internal/mcp/cluster_handlers.go`).
   - Permite que agentes remotos e locais obtenham sumários instantâneos dos macrodomínios conceituais da memória.

### Positive Consequences

- **Identificação Automática de Módulos:** O grafo se auto-organiza em domínios temáticos sem requerer marcação manual de tags em notas.
- **Navegação Visual Clara:** O desenvolvedor pode distinguir visualmente os diferentes subsistemas de código e módulos de notas.
- **Rápido e 100% Determinístico:** Complexidade próxima a $O(V + E)$, executando em milissegundos para centenas de notas sem estocasticidade.

### Negative Consequences / Trade-offs

- **Sensibilidade a Grafos Esparsos:** Em grafos com pouquíssimas arestas, a maioria das notas formará clusters unitários (resolvido na CLI através do filtro padrão `--min-size 2`).

---

## Pros and Cons of the Alternatives

### Louvain / Leiden Algorithm
- *Bom*: Otimiza diretamente a modularidade $Q$ de forma hierárquica.
- *Ruim*: Requer implementação consideravelmente mais complexa de múltiplos passos de contração de grafo e agregação de super-nós, com maior propensão a bugs em Go puro. O LPA determinístico atinge coesão equivalente com menor complexidade de código.

### K-Means ou Spectral Clustering sobre Embeddings
- *Bom*: Agrupa pela proximidade vetorial semântica.
- *Ruim*: Exige fixar o número de clusters $K$ a priori e não reflete as arestas estruturais e epistêmicas estabelecidas explicitamente pelo desenvolvedor no vault.

---

## Links e Referências

- **[CodeGraph (colbymchenry/codegraph)](https://github.com/colbymchenry/codegraph)**: Referência de ponta para grafos de conhecimento locais e pré-indexados integrados a agentes de IA (Claude Code, Cursor, Antigravity, Codex), com sincronização contínua via watcher, SQLite embarcado, visualização interativa e entrega cirúrgica de contexto estrutural sem leituras desnecessárias de arquivos.
- **[Raghavan et al. (2007) - Near linear time algorithm to detect community structures in large-scale networks](https://arxiv.org/abs/0709.2938)**: Fundamentação do algoritmo Label Propagation Algorithm (LPA).
- **[Newman (2006) - Modularity and community structure in networks](https://www.pnas.org/doi/10.1073/pnas.0601602103)**: Formulação canônica da Modularidade $Q$.
- [ADR-011: Arestas Epistêmicas e God Nodes / Hubs de Conhecimento](011-arestas-epistemicas-e-god-nodes.md)
- [ADR-014: Centralidade de Grafo com PageRank Ponderado](014-centralidade-de-grafo-com-pagerank-ponderado.md)
- [ADR-020: Visualizador Interativo de Grafo em HTML/SVG Standalone](020-visualizador-interativo-de-grafo-em-html-svg.md)
