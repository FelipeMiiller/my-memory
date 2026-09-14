# Feature: graph-path-discovery

## Problem Statement
Agentes de IA e engenheiros de software operando em bases de código e grafos de conhecimento densos necessitam frequentemente rastrear como um conceito, módulo ou decisão técnica $A$ se conecta, depende ou influencia outro nó $B$ através de cadeias intermediárias (*multi-hop paths*).

Atualmente, o **My-Memory** oferece ferramentas focadas em nós isolados:
1. `memory_get_neighbors`: explora apenas 1 salto direto ($k=1$).
2. `memory_get_impact` (ADR-023): calcula o raio de destruição reverso em cascata sem isolar uma rota entre dois pontos específicos.
3. `memory_inspect_node` (ADR-024): tríptico cirúrgico focado em um nó individual.

Quando um agente precisa responder a perguntas como:
- *"Como o módulo de autenticação (`auth/session.md`) afeta o driver de banco (`db/postgres.md`)?"*
- *"Qual é a rota de dependência entre a especificação do ADR-009 e a classe de busca vetorial?"*

O agente é forçado a disparar múltiplas chamadas consecutivas de `memory_get_neighbors`, reconstruindo subgrafos mentalmente no prompt, estourando a janela de contexto de tokens e correndo risco de alucinação de rotas inexistentes.

Inspirado nas necessidades de navegação cirúrgica de grafos para agentes autônomos ([docs/REFERENCES.md#L63](../../../docs/REFERENCES.md#L63)), esta feature introduz a **Descoberta de Rotas & Caminho Mínimo no Grafo (`mem path` / `memory_find_path`)**, provendo cálculo de rota determinístico via algoritmo Dijkstra ponderado por pesos epistêmicos (`EXTRACTED`, `INFERRED`, `TAG`) e BFS, disponível via CLI e protocolo MCP.

---

## Goals
- [x] **G1**: Implementar o motor de pathfinding no pacote puro `internal/graph/path.go`, suportando:
  - Modos de custo: `epistemic` (custo inversamente proporcional à confiança epistêmica: $c = 1.0 / w$) e `hops` (custo uniforme por aresta).
  - Grafos direcionados ($A \to B$) e bidirecionais/não-direcionados ($A \leftrightarrow B$).
  - Limitação de profundidade de busca (`max_depth`) e detecção de ciclos com garantias de terminação.
- [x] **G2**: Implementar métodos de resolução e carregamento de rotas nas camadas de persistência:
  - SQLite: `internal/db/graph.go` (`FindPath(ctx, db, sourceQuery, targetQuery, opts)`).
  - PostgreSQL: `internal/store/postgres.go` e interface `internal/store/store.go`.
  - Resolução resiliente de identificadores canônicos para nós de origem e destino (`ResolveNodeCanonicalID`).
- [x] **G3**: Criar o subcomando CLI `mem path <source> <target>` em `cmd/mem/path.go`, com flags `--directed`, `--max-depth`, `--mode` e `--json`, além de renderizador visual de rota em ASCII.
- [x] **G4**: Implementar e expor a ferramenta MCP `memory_find_path` em `internal/mcp/path_handlers.go`, registrada em stdio e HTTP/SSE com esquema formal JSON Schema.
- [x] **G5**: Documentar a decisão arquitetural em `docs/adr/027-descoberta-de-rotas-e-caminho-minimo-no-grafo.md`, atualizando os índices e manuais operacionais.

---

## Out of Scope
- Algoritmos de todos os caminhos possíveis sem restrição de corte (problema NP-completo em grafos cíclicos densos; o foco é o caminho mais curto / de menor custo).
- Algoritmos com pesos negativos (o grafo de conhecimento possui apenas pesos de afinidade estritamente positivos: $w \in (0, 1]$).
- Reindexação de arquivos ou mutação do banco de dados (esta feature é puramente analítica e de consulta).

---

## Assumptions & Open Questions

| Assumption / Decisão | Valor Escolhido | Justificativa | Confirmado? |
| :--- | :--- | :--- | :--- |
| Modo de custo padrão | `epistemic` | Prefere arestas explícitas do autor (`EXTRACTED`, custo 1.0) sobre suposições vetoriais (`INFERRED`, custo 1.67) e `TAG` (custo 3.33) | y |
| Direcionamento padrão | Direcionado (`directed: true`) | Em grafos de dependência e arquitetura, a direção causal importa (A depende de B não significa que B depende de A) | y |
| Profundidade padrão | `max_depth: 6` | Suficiente para a esmagadora maioria das conexões em vaults e bases de código sem degradar performance | y |
| Comportamento em caso de nó inexistente | Erro claro indicando qual nó não foi encontrado | Evita buscas custosas e guia o usuário/agente na correção de nomes | y |

---

## User Stories

### P1: Cálculo de Menor Rota Epistêmica no Grafo ⭐ MVP
**User Story**: Como um agente de IA ou desenvolvedor, quero consultar `mem path auth db` para obter a sequência exata de nós e arestas que conectam os dois pontos com o menor custo de navegação.
**Critérios de Aceite**:
1. Se houver caminho conectando origem e destino dentro do `max_depth`, retorna `found: true` com lista ordenada de nós e arestas detalhadas (relação, status epistêmico e peso).
2. Se não houver caminho, retorna `found: false` com mensagem elucidativa.
3. Se origem for igual ao destino, retorna caminho de 0 saltos com custo 0.

### P2: Travessia Não-Direcionada e Modos de Custo
**User Story**: Como engenheiro investigando afinidade conceitual ampla, quero executar `mem path auth db --undirected --mode hops` para encontrar o caminho com menor número de intermediários, ignorando a direção das setas.
**Critérios de Aceite**:
1. A flag `--undirected` permite navegar pelas arestas em ambos os sentidos.
2. Cada aresta no resultado indica sua orientação relativa (`forward` ou `reverse`).
3. O modo `--mode hops` usa custo unitário por aresta em vez de custo epistêmico ponderado.

### P3: Integração com Servidor MCP para Agentes Autônomos
**User Story**: Como assistente de IA conectado via MCP, quero chamar a ferramenta `memory_find_path(source, target, max_depth, directed)` para receber o caminho estruturado em Markdown e JSON sem precisar disparar múltiplas buscas manuais.
**Critérios de Aceite**:
1. Ferramenta registrada e listada no catálogo `tools/list`.
2. Saída em Markdown formatada com representação visual em linha (ex: `A --[implements]--> B --[depends_on]--> C`) e tabela de detalhes.
