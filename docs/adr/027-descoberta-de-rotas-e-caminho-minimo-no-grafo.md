# ADR-027: Descoberta de Rotas e Caminho Mínimo no Grafo de Conhecimento

- **Date**: 2026-09-14
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: graph, pathfinding, dijkstra, shortest-path, epistemic-weights, bfs, mcp, cli

## Context and Problem Statement

Agentes de IA e engenheiros navegando por vaults de notas e bases de código modeladas no **My-Memory** frequentemente precisam entender como dois conceitos, módulos ou decisões arquiteturais aparentemente distantes estão interligados.

Até o momento, as ferramentas analíticas de grafo existentes cobriam:
1. Inspeção local de 1 salto (`memory_get_neighbors`).
2. Fechamento de dependências reversas em leque (`memory_get_impact` / blast radius).
3. Centralidade e autoridade estrutural (`PageRank`).
4. Agrupamento em comunidades conceituais (`LPA Clusters`).

No entanto, quando um agente precisa saber especificamente:
> *"Qual é a cadeia de dependências ou conexões conceituais que leva da nota A até a nota B?"*

O sistema não oferecia uma forma direta de responder. O agente era forçado a consultar os vizinhos da nota A, depois os vizinhos de cada vizinho, acumulando múltiplos turnos de conversação e montando o grafo manualmente em seu contexto LLM — um processo lento, caro em tokens e propenso a alucinações.

Havia a necessidade de um mecanismo nativo de **Pathfinding (Descoberta de Rotas e Caminho Mínimo)** com sensibilidade epistêmica no My-Memory.

## Decision Drivers

- **Ponderação Epistêmica**: Links explícitos intencionais (`EXTRACTED` via `[[wikilink]]`, peso 1.0) devem ser preferidos a conexões inferidas estatisticamente (`INFERRED`, peso 0.6) ou vínculos fracos de categorização (`TAG`, peso 0.3). O algoritmo deve priorizar a certeza epistêmica.
- **Determinismo e Eficiência**: A busca de rota entre dois nós deve executar em milissegundos ($O((V + E) \log V)$), sem sobrecarga de CPU ou dependências de bibliotecas externas de CGO.
- **Flexibilidade Direcional**: Permitir navegação estritamente direcionada ($A \to B$) para análise causal e de dependência técnica, ou bidirecional/não-direcionada para exploração de afinidade conceitual ampla.
- **Corte de Profundidade Seguro**: Suporte a limite de saltos (`max_depth`) para evitar buscas custosas em grafos com nós hiperconectados (*God Nodes*).
- **Consumo Cirúrgico por LLMs**: Respostas prontas via protocolo MCP (`memory_find_path`) e CLI (`mem path`) com representação visual em ASCII legível de imediato.

## Decision Outcome

Adotou-se o algoritmo **Dijkstra com Ponderação Epistêmica Inversa** implementado no pacote de domínio `internal/graph/path.go` e integrado aos armazenamentos SQLite e PostgreSQL, à CLI e ao servidor MCP:

### 1. Modelo de Custo Epistêmico
Para cada aresta $e$ com peso epistêmico $w(e) \in (0, 1]$, define-se o custo de travessia $c(e)$ no modo `epistemic`:
$$c(e) = \frac{1.0}{w(e)}$$
- `EXTRACTED` ($w=1.0$): custo $= 1.0$
- `INFERRED` ($w=0.6$): custo $\approx 1.667$
- `TAG` ($w=0.3$): custo $\approx 3.333$

No modo `hops`, todas as arestas recebem custo unitário $c(e) = 1.0$, equivalendo à menor distância em quantidade de passos (BFS).

### 2. Algoritmo de Busca (`internal/graph/path.go`)
- Utiliza fila de prioridade mínima baseada no pacote padrão `container/heap` do Go.
- Trata ciclos nativamente mantendo tabela de menores custos visitados.
- Suporta busca direcionada ou bidirecional (`Directed: bool`).
- Em caso de nós idênticos ($A = B$), retorna rota imediata de 0 saltos com custo 0.0.

### 3. Camada de Dados (`internal/db/` e `internal/store/`)
- Método `FindPath` com resolução resiliente de aliases, extensões `.md` e barras (`ResolveNodeCanonicalID`).
- Compatibilidade bilateral garantida entre SQLite e PostgreSQL.

### 4. CLI `mem path` (`cmd/mem/path.go`)
- Sintaxe: `mem path <source> <target> [--undirected] [--max-depth <N>] [--mode <epistemic|hops>] [--json]`
- Apresenta representação ASCII encadeada:
  `[auth/session.md] --(implements [EXTRACTED])--> [core/token.md] --(depends_on [EXTRACTED])--> [db/redis.md]`

### 5. Ferramenta MCP `memory_find_path` (`internal/mcp/`)
- Parâmetros: `source` (obrigatório), `target` (obrigatório), `directed` (opcional, default: true), `max_depth` (opcional, default: 6), `mode` (opcional, default: "epistemic").
- Resposta rica em Markdown com metadados de rota, saltos, custo total e tabela de trechos.

### Positive Consequences

- **Navegação Cirúrgica**: Agentes identificam cadeias de dependência entre componentes distantes em uma única chamada MCP.
- **Economia Drástica de Tokens**: Elimina a necessidade de chamadas sucessivas de `memory_get_neighbors`.
- **Respeito à Certeza Epistêmica**: Evita sugerir rotas baseadas em conexões frágeis quando há um caminho explícito e sólido documentado.
- **Zero Alucinação**: Todo o caminho é estritamente derivado da topologia verificada do grafo.

### Negative Consequences / Trade-offs

- **Custo Computacional em Grafos Gigantescos**: Em grafos com centenas de milhares de nós sem direção (`--undirected`), a busca em profundidades altas pode percorrer muitos caminhos alternativos. O parâmetro `max_depth` (default 6) mitiga esse risco.

---

## Links e Referências

- [ADR-011: Arestas Epistêmicas e God Nodes](011-arestas-epistemicas-e-god-nodes.md)
- [ADR-014: Centralidade de Grafo com PageRank Ponderado](014-centralidade-de-grafo-com-pagerank-ponderado.md)
- [ADR-023: Análise de Impacto e Raio de Destruição (Blast Radius)](023-analise-de-impacto-e-raio-de-destruicao-blast-radius.md)
- [ADR-024: Visualização Cirúrgica em 3 Colunas (Triptych Node Inspector)](024-visualizacao-cirurgica-em-3-colunas-triptych-inspector.md)
- [docs/REFERENCES.md - CodeGraph Surgical Context](../../docs/REFERENCES.md#L63)
