# Arquitetura do My-Memory

O **My-Memory** é uma engine de memória semântica e relacional local desenvolvida em Go, combinando o melhor de múltiplos ecossistemas e inovações:
1. **Obsidian**: Grafo de conhecimento orientado a notas Markdown (`[[wikilinks]]` e `#tags`).
2. **ai-memory**: Indexação rápida, chunking de texto, padrão *compile-not-retrieve* e busca semântica para IAs.
3. **Graphify**: Modelagem de nós e arestas com expansão relacional, status epistêmico e centralidade de nós.
4. **CodeGraph**: Monitoramento contínuo em tempo real com file watcher reativo, contexto cirúrgico para agentes e análise de impacto.
5. **TurboQuant (Google DeepMind, ICLR 2026)**: Compressão vetorial extrema em 4-bits com estimador de produto escalar não-viesado.

---

## 🏛 Visão Geral dos Componentes

```
                               [ Arquivos Markdown ]
                                        │
                    ┌──────────────────┴──────────────────┐
                    ▼                                     ▼
        [ Parser de Wikilinks & Tags ]           [ Chunking de Texto ]
                    │                                     │
                    ▼                                     ▼
           ( graph_nodes & edges )              [ Ollama Embedding ]
                    │                           (nomic-embed-text 768d)
                    │                                     │
                    │                    ┌────────────────┴────────────────┐
                    │                    ▼                                 ▼
                    │            [ sqlite-vec ]                  [ TurboQuant 4-bit ]
                    │            (float32 k-NN)                   (Rotação Householder
                    │                  │                            + Bit-Packing)
                    │                  │                                   │
                    ▼                  ▼                                   ▼
          ┌──────────────────────────────────────────────────────────────────────┐
          │                  Camada de Persistência Unificada                    │
          │  SQLite Local (memory.db) / PostgreSQL com pgvector (Multi-Repo)    │
          │  - documents          - chunks_vec (vec0)      - graph_nodes         │
          │  - chunks             - chunks_turboquant      - graph_edges         │
          │  - chunks_fts (FTS5)                                                 │
          └──────────────────────────────────────────────────────────────────────┘
                                       ▲
                                       │
                      ┌────────────────┴────────────────┐
                      ▼                                 ▼
              Motor de Busca Híbrido          Algoritmos de Grafo
             (RRF + Decaimento Temporal)     (PageRank, LPA, CTEs)
                      │                                 │
                      └────────────────┬────────────────┘
                                       │
                                       ▼
                     ┌───────────────────────────────────┐
                     │     Camada de Interfaces & MCP    │
                     │  - CLI (mem search, clusters...)  │
                     │  - MCP Stdio (Claude/Cursor)      │
                     │  - MCP HTTP/SSE (Rede / Remoto)   │
                     │  - GraphView HTML/SVG Interativo  │
                     └───────────────────────────────────┘
```

---

## 💾 Esquema do Banco de Dados (SQLite / PostgreSQL)

### 1. Documentos e Chunks
- `documents`: Metadados do arquivo (caminho, título, hash SHA-256, repositório, timestamp).
- `chunks`: Pedaços de texto divididos com sobreposição de palavras (*overlap*).

### 2. Busca Léxica (FTS5 / tsvector)
- `chunks_fts`: Tabela virtual nativa do SQLite com tokenizador Porter (stemming) e Unicode61 para busca exata por palavras-chave via BM25 (ou `tsvector` com GIN no PostgreSQL).

### 3. Busca Vetorial Nativa (`sqlite-vec` / `pgvector`)
- `chunks_vec`: Tabela virtual do tipo `vec0` armazenando os vetores originais de 768 floats de 32 bits (ou coluna `vector(768)` no PostgreSQL).

### 4. Busca Vetorial Ultracompacta (`chunks_turboquant`)
- `chunks_turboquant`: Armazena o vetor quantizado para 4-bits (384 bytes por chunk + float de escala). Permite economizar ~88% do espaço em disco.

### 5. Grafo de Conhecimento (`graph_nodes` e `graph_edges`)
- `graph_nodes`: Vértices do grafo (`note`, `tag`, `concept`, `decision`).
- `graph_edges`: Arestas direcionadas tipadas com status epistêmico (`EXTRACTED`, `INFERRED`, `TAG`) e pesos associados.

---

## 🕸 Expansão de Grafo com SQL Recursivo

Em vez de usar bancos de grafos externos pesados (como Neo4j), a navegação por conexões adjacentes é feita diretamente pelo SQLite usando **Recursive Common Table Expressions (CTEs)**:

```sql
WITH RECURSIVE traversal AS (
    -- Ponto de partida: documentos retornados na busca vetorial
    SELECT target_id, 1 AS depth
    FROM graph_edges
    WHERE source_id = :node_id
    
    UNION
    
    -- Expansão de vizinhos até a profundidade máxima
    SELECT e.target_id, t.depth + 1
    FROM graph_edges e
    JOIN traversal t ON e.source_id = t.target_id
    WHERE t.depth < :max_depth
)
SELECT DISTINCT target_id FROM traversal;
```

---

## 🧠 Algoritmos Avançados de Grafo

### 1. Centralidade e PageRank Ponderado (ADR-014)
Identifica a autoridade estrutural de cada nota no grafo através do algoritmo PageRank com suporte a arestas ponderadas por peso epistêmico:
$$PR(u) = \frac{1 - d}{N} + d \sum_{v \in In(u)} \frac{PR(v) \cdot w(v, u)}{\sum_{k \in Out(v)} w(v, k)}$$
* Fator de amortecimento padrão: $d = 0.85$, com 30 iterações e tolerância de convergência $10^{-6}$.
* Pesos epistêmicos: arestas explícitas `EXTRACTED` têm peso 1.0, inferidas `INFERRED` têm peso 0.6 e `TAG` têm peso 0.3.

### 2. Detecção de Comunidades via LPA e Modularidade de Newman-Girvan (ADR-022)
Para particionar a memória em clusters temáticos coesos de forma autônoma:
* **Weighted Label Propagation Algorithm (LPA):** Cada nó propaga e adota iterativamente o rótulo de comunidade mais frequente entre seus vizinhos ponderados:
  $$C(u) = \arg\max_{c} \sum_{v \in N(u), C(v)=c} w(u, v)$$
  com desempate lexicográfico determinístico para garantir idempotência.
* **Modularidade de Newman-Girvan \(Q\):** Avalia quantitativamente a qualidade da partição encontrada contra um modelo nulo aleatório:
  $$Q = \frac{1}{2m} \sum_{i,j} \left[ A_{ij} - \frac{k_i k_j}{2m} \right] \delta(c_i, c_j)$$
  onde $m$ é o peso total das arestas, $k_i$ é o grau ponderado do nó $i$, e $\delta(c_i, c_j) = 1$ quando os nós pertencem à mesma comunidade. Valores de $Q > 0.3$ indicam forte modularidade estrutural.
* Cada comunidade tem seu **nó líder** determinado pelo maior PageRank local e seu **tipo dominante** de nota.

---

## 🌐 Protocolo Model Context Protocol (MCP) Multi-Transporte (ADR-006, ADR-021)

O My-Memory atua como servidor de contexto para LLMs e agentes através de dois mecanismos de transporte:
1. **Stdio (Processo Filho):** Comunicação padrão via `stdin`/`stdout` com framing JSON-RPC 2.0 delimitado por quebras de linha (`\n`), consumido nativamente por Claude Code, Cursor, Windsurf e Antigravity.
2. **HTTP com Server-Sent Events (SSE):** Servidor autônomo de rede (`mem mcp --port 8080`) com:
   * `GET /sse`: Stream SSE unidirecional com handshake `endpoint`.
   * `POST /message?sessionId=<uuid>`: Recepção de comandos JSON-RPC vinculados à sessão ativa.
   * `POST /mcp`: Chamadas diretas (stateless) para scripts rápidos ou ferramentas web.
   * `GET /health`: Monitoramento de status, uptime e ferramentas registradas.
   * Suporte a CORS com preflight `OPTIONS` para conexões diretas via navegador.

---

## 🎨 Visualizador Interativo de Grafo em HTML/SVG Standalone (ADR-020)

O pacote `internal/graphview` compila a topologia do grafo em um artefato HTML/SVG totalmente autocontido (Zero-CDN) executável em qualquer navegador moderno:
* Simulação de física de forças (repulsão Coulombiana e atração de Hooke).
* Nós dimensionados dinamicamente com base no seu PageRank estrutural.
* Nós coloridos por tipo de nota ou particionados pelas cores de cluster LPA detectadas.
* Busca em tempo real com destaque e atenuação de nós não correlacionados.
* Painel lateral retrátil com backlinks, tags e atalho para Obsidian URI (`obsidian://open?file=...`).

---

## 🔄 Higiene e Sincronização Contínua (ADR-013, ADR-017)

* **Indexação Contínua (`mem watch`):** Monitoramento reativo em segundo plano com debouncing de 500ms, reindexação cirúrgica e remoção instantânea de arquivos deletados.
* **Linter de Grafo (`mem doctor`):** Diagnóstico automático de links quebrados (*dead links*), notas órfãs, loops reflexivos (*self-loops*) e cálculo de Health Score (0-100), com reparo automático (`--fix`).

---

## 📚 Arte Prévia e Influências

O My-Memory combina ideias comprovadas da literatura e projetos abertos de ponta:
- **`akitaonrails/ai-memory`**: Padrão *compile-not-retrieve*, busca híbrida com RRF (*Reciprocal Rank Fusion*) e Markdown no Git como fonte soberana da verdade.
- **`Graphify-Labs/graphify`**: Classificação epistêmica de arestas (`EXTRACTED` vs `INFERRED`), identificação de *God Nodes* (conceitos de alta centralidade) e cache incremental SHA-256.
- **`colbymchenry/codegraph`**: Grafo de conhecimento pré-indexado para agentes com eliminação de explorações cegas (*zero file reads*), sincronização viva reativa via watcher, alertas de staleness e análise de impacto.
- **`kepano/obsidian-skills`**: Especificações oficiais de *Obsidian Flavored Markdown* e geração de mapas mentais no formato aberto *JSON Canvas 1.0* (`.canvas`).

Consulte o detalhamento completo em **[`docs/REFERENCES.md`](REFERENCES.md)**.

