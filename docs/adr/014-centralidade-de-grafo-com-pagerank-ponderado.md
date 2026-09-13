# ADR-014: Centralidade de Grafo com PageRank Ponderado

- **Date**: 2026-09-13
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: graph, pagerank, centrality, epistemic-edges, god-nodes, hubs, mcp, cli

## Context and Problem Statement

No My-Memory, a identificação de nós centrais (*God Nodes* / Hubs - ADR-011) baseava-se exclusivamente no cálculo de grau bruto ($D = \text{in-degree} + \text{out-degree}$). Embora eficiente para identificar nós com muitas referências imediatas, essa métrica apresenta duas limitações qualitativas fundamentais na modelagem de conhecimento:

1. **Equivalência Ingênua de Referências:** Todos os links de entrada são tratados com o mesmo peso. Uma menção vinda de um documento fundamental de arquitetura possui o mesmo valor de uma menção secundária em uma nota de rodapé.
2. **Desconsideração Epistêmica:** O sistema não ponderava a certeza semântica entre links explícitos criados intencionalmente pelo autor (`EXTRACTED` - `[[wikilink]]`) e arestas inferidas estatisticamente através de similaridade vetorial (`INFERRED`).

Havia a necessidade de um algoritmo de autoridade estrutural iterativo que transitasse relevância recursivamente através das conexões do grafo, priorizando nós com referências qualificadas.

## Decision Drivers

- **Qualidade Semântica da Autoridade:** Documentos referenciados por outros nós autoritativos devem receber maior prestígio estrutural.
- **Ponderação Epistêmica:** Conexões explícitas (`EXTRACTED`) devem transmitir mais peso e autoridade do que conexões inferidas (`INFERRED`).
- **Performance e Escalabilidade:** O cálculo deve convergir em tempo inferior a 10ms para vaults com até 1.000 nós.
- **Retrocompatibilidade:** O comando `mem hubs` e a ferramenta MCP `memory_get_hubs` devem manter o cálculo por grau (`degree`) como padrão, permitindo a seleção transparente de `pagerank`.

## Decision Outcome

Implementou-se o algoritmo de **PageRank Ponderado com Fator de Amortecimento** no pacote `internal/graph`:

1. **Motor Matemático em Go Puro (`internal/graph/pagerank.go`):**
   - Formulação com fator de amortecimento canônico $d = 0.85$ e tolerância $\epsilon = 10^{-6}$.
   - Tratamento formal de *dangling nodes* (nós sem arestas de saída), redistribuindo sua massa uniformemente entre todos os nós do grafo.
   - Ponderação epistêmica diferenciada por tipo de aresta: `EXTRACTED = 1.0`, `INFERRED = 0.6` e `TAG = 0.3`.
   - Invariante determinístico de conservação de probabilidade: $\sum_{u \in V} PR(u) = 1.0$.

2. **Integração no Contrato `Store`:**
   - Adicionada a struct `PageRankNode` e o método `ComputePageRank(ctx, repo, damping, maxIter)` na interface `Store`.
   - Implementado para **SQLite** (`internal/db/store.go`) e **PostgreSQL** (`internal/store/postgres.go`), com suporte a isolamento por repositório.

3. **Protocolo MCP (`internal/mcp`):**
   - Atualizado o schema da ferramenta `memory_get_hubs` com o parâmetro `algorithm: "degree" | "pagerank"`.
   - Implementado `NewAdvancedMemoryGetHubsHandler` e `SetAdvancedHubsHandler` para formatação em Markdown com scores percentuais.

4. **Interface CLI (`cmd/mem/main.go`):**
   - Subcomando `mem hubs` atualizado com as flags `--algorithm degree|pagerank`, `--damping 0.85` e `--iter 30`.
   - Saída tabular formatada com ranks, scores numéricos e contadores de grau.

### Positive Consequences

- **Identificação Superior de Documentos-Chave:** Documentos fundacionais são destacados mesmo quando possuem menos conexões brutas que notas acessórias hiperlinkadas.
- **Ponderação de Certeza Epistêmica:** O grafo reflete a intencionalidade humana expressa nos wikilinks.
- **Convergência Rápida:** Benchmarks medem 0.50 ms para 100 nós, 3.02 ms para 500 nós e 7.07 ms para 1.000 nós.

### Negative Consequences / Trade-offs

- **Custo Computacional vs Grau Simples:** PageRank executa até 30 iterações em memória sobre as arestas ativas, sendo ligeiramente mais custoso que um `GROUP BY` direto de graus em SQL, embora ainda desprezível em vaults típicos.
