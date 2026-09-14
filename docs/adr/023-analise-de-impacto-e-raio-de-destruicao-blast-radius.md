# ADR-023: Análise de Impacto e Raio de Destruição (Blast Radius Analysis)

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: graph, impact-analysis, blast-radius, reverse-dependencies, risk-scoring, pagerank, community-detection, cli, mcp

## Context and Problem Statement

Em projetos complexos de software e bases de conhecimento extensas (vaults), alterar uma nota de arquitetura, contrato de API ou componente de código pode desencadear efeitos colaterais em cascata que quebram dependentes não óbvios.

Até então, o **My-Memory** disponibilizava:
1. Travessia de vizinhos diretos (`memory_get_neighbors` / ADR-004).
2. Detecção de nós centrais (God Nodes / ADR-011).
3. Autoridade estrutural via PageRank (ADR-014).
4. Partições temáticas e clusters (ADR-022).

Entretanto, desenvolvedores e agentes autônomos de IA (Cursor, Antigravity, Claude Code) careciam de uma capacidade fundamental: **analisar preventivamente o "raio de destruição" (*blast radius*) e o fechamento de dependências reversas** antes de submeter uma refatoração ou alteração arquitetural.

Perguntas críticas permaneciam sem resposta automatizada:
- *Quais componentes e documentos dependem direta ou indiretamente deste nó?*
- *Qual o nível de risco e severidade desta alteração para o sistema?*
- *A modificação atinge múltiplos domínios/clusters temáticos isolados?*

## Decision Drivers

- **Algoritmo em Go Puro (Zero Dependências Externas)**: Cálculo determinístico executável em SQLite ou PostgreSQL sem ferramentas externas.
- **Fechamento de Dependências Reversas (Inbound Edges)**: Navegar no sentido inverso das arestas direcionadas ($A \to B \implies B$ afeta $A$ se $A$ depende de $B$), respeitando limites de profundidade (`max_depth`) e imunidade a ciclos.
- **Classificação Semântica de Severidade**: Diferenciar arestas de quebra forte (`implements`, `depends_on`, `supersedes`) de conexões fracas (`links_to`, `tags`).
- **Score de Risco Normalizado e Ponderado**: Produzir métrica contínua de 0.0 a 100.0 que incorpore contagem de nós, decaimento hiperbólico por profundidade ($1/\text{depth}$), bônus de autoridade via PageRank e dispersão entre clusters.
- **Resolução Flexível de Identificadores (Fuzzy Resolution)**: Capacidade de resolver caminhos relativos de arquivos (`concepts/auth.md`), slugs (`auth`) ou títulos exatos (`Autenticação`).
- **Superfície Dupla (CLI e MCP)**: Oferecer o subcomando `mem impact <node_id> [--depth 2] [--json]` para humanos e a ferramenta `memory_get_impact` para agentes de IA.

## Decision Outcome

Adotou-se o motor unificado de **Análise de Impacto e Raio de Destruição (*Blast Radius*)**:

### 1. Algoritmo de Travessia e Score de Risco (`internal/graph/impact.go`)

- **Travessia em Largura (BFS Reversa)**:
  - Constrói o mapa de arestas de entrada (`inboundMap`).
  - Visita os nós dependentes camada por camada até `max_depth` (padrão: 2).
  - Mantém tabela de distâncias mínimas para evitar re-visitações ou loops cíclicos.
- **Severidade Semântica (`DetermineSeverity`)**:
  - `CRITICAL`: Relações de dependência estrita (`implements`, `depends_on`, `contradicts`, `blocks`, `supersedes`) em profundidade 1.
  - `HIGH`: Dependências estritas em profundidade $> 1$.
  - `MEDIUM`: Conexões conceituais (`links_to`, `refers_to`) em profundidade 1.
  - `LOW`: Conexões conceituais profundas ou tags fracas.
- **Fórmula do Score de Risco**:
  $$\text{RawRisk} = \sum_{n \in \text{Impacted}} w(\text{Severity}_n) \cdot \frac{1}{\text{Depth}_n} \cdot (1 + 10 \cdot \text{PageRank}_n)$$
  $$\text{RiskScore} = 100 \cdot \left(1 - \frac{1}{1 + \frac{\text{RawRisk}}{5}}\right)$$
  - Normalizado entre 0.0 e 100.0 através de saturação sigmoidal.
  - Categorizado em faixas:
    - $\ge 75.0$: `CRÍTICO` (🔴)
    - $\ge 50.0$: `ALTO` (🟠)
    - $\ge 25.0$: `MODERADO` (🟡)
    - $< 25.0$: `BAIXO` (🟢)

### 2. Camada de Dados e Resolução Canônica (`internal/db/graph.go` & `internal/store/postgres.go`)

- **`ResolveNodeCanonicalID`**:
  - Busca exata por ID canônico.
  - Busca por `path` relativo ou normalizado sem extensão `.md`.
  - Busca por `title` exato ou `LOWER(title)`.
  - Resolução por substring do caminho, garantindo robustez a inputs parciais de usuários e LLMs.
- **`CalculateImpactForTarget` / `CalculateImpact`**:
  - Carrega arestas do grafo, calcula centralidade via PageRank e mapeia clusters via Weighted LPA (ADR-022).
  - Executa o cálculo e retorna a estrutura `graph.ImpactResult` consolidada.

### 3. Interface de Linha de Comando (`cmd/mem/impact.go`)

- `mem impact <node_id> [--depth 2] [--json]`:
  - Exibe resumo executivo no terminal: Nó alvo, Total impactado (diretos vs indiretos), Score de Risco com badge, Clusters afetados e contagem de severidades.
  - Renderiza tabela hierárquica contendo profundidade, severidade, tipo de relação, nó intermediário e ID afetado.
  - Suporte total a `--json` para scripts de automação e validações em pipelines CI/CD.

### 4. Ferramenta MCP para Agentes de IA (`internal/mcp/impact_handlers.go`)

- **Ferramenta `memory_get_impact`**:
  - Argumentos: `node_id` (obrigatório), `max_depth` (opcional, padrão 2), `repository` (opcional).
  - Responde com sumário conciso em Markdown formatado, permitindo que agentes LLM tomem decisões conscientes de refatoração ou planejem testes direcionados antes de alterar arquivos fundamentais.

---

### Positive Consequences

- **Prevenção Pró-Ativa de Quebras**: Engenheiros e agentes sabem exatamente o impacto antes de modificar contratos ou componentes centrais.
- **Aderência às Melhores Práticas de Agentes (CodeGraph)**: Equipara o My-Memory ao ecossistema do CodeGraph, onde agentes inteligentes inspecionam o raio de impacto via ferramentas dedicadas de grafo.
- **Zero Overhead em Tempo de Execução**: Avaliação BFS sobre estruturas em memória roda em menos de 10 milissegundos para repositórios com centenas de nós.

### Negative Consequences / Trade-offs

- **Dependência da Qualidade das Arestas**: Se um desenvolvedor omitir referências ou links explícitos entre notas/código, o raio de destruição pode subestimar dependentes reais (mitigado pelo decaimento semântico e sugestões do Doctor / ADR-013).

---

## Pros and Cons of the Alternatives

### Análise Estática Exclusiva via CTE Recursivo em SQL
- *Bom*: Executado inteiramente dentro do banco.
- *Ruim*: Extremamente difícil calcular decaimento hiperbólico contínuo, mesclar PageRank em ponto flutuante e cruzar com partições de clusters LPA de maneira portável e eficiente entre SQLite e PostgreSQL.

### Avaliação Não Estruturada via Prompt da LLM
- *Bom*: Não exige código no core.
- *Ruim*: Não determinístico, consome milhares de tokens, hallucina dependências inexistentes e falha completamente em vaults médios e grandes.

---

## Links e Referências

- **[CodeGraph (colbymchenry/codegraph)](https://github.com/colbymchenry/codegraph)**: Referência de vanguarda para grafos de conhecimento locais e pré-indexados integrados a agentes de IA, com análise de impacto estrutural e entrega cirúrgica de contexto sem leitura cega de código.
- [ADR-004: Modelagem de Grafo Relacional com Recursive CTEs](004-modelagem-de-grafo-com-recursive-ctes.md)
- [ADR-011: Arestas Epistêmicas e God Nodes / Hubs de Conhecimento](011-arestas-epistemicas-e-god-nodes.md)
- [ADR-014: Centralidade de Grafo com PageRank Ponderado](014-centralidade-de-grafo-com-pagerank-ponderado.md)
- [ADR-022: Detecção de Comunidades e Clusters no Grafo de Conhecimento](022-deteccao-de-comunidades-e-clusters-no-grafo.md)
