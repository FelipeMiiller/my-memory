# Arquitetura do My-Memory

O **My-Memory** é uma engine de memória semântica e relacional local desenvolvida em Go, combinando o melhor de três mundos:
1. **Obsidian**: Grafo de conhecimento orientado a notas Markdown (`[[wikilinks]]` e `#tags`).
2. **ai-memory**: Indexação rápida, chunking de texto e busca semântica para IAs.
3. **Graphify**: Modelagem de nós e arestas com expansão relacional para navegação em contexto profundo.
4. **TurboQuant (Google DeepMind, ICLR 2026)**: Compressão vetorial extrema em 4-bits com estimador de produto escalar não-viesado.

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
          │                       SQLite Local (memory.db)                       │
          │  - documents          - chunks_vec (vec0)      - graph_nodes         │
          │  - chunks             - chunks_turboquant      - graph_edges         │
          │  - chunks_fts (FTS5)                                                 │
          └──────────────────────────────────────────────────────────────────────┘
                                       ▲
                                       │
                              [ Motor de Busca ]
                                       │
                      ┌────────────────┴────────────────┐
                      ▼                                 ▼
              Busca Semântica                  Expansão de Grafo
              (sqlite-vec / TQ)            (SQL WITH RECURSIVE CTE)
```

---

## 💾 Esquema do Banco de Dados (SQLite)

### 1. Documentos e Chunks
- `documents`: Metadados do arquivo (caminho, título, timestamp).
- `chunks`: Pedaços de texto divididos com sobreposição de palavras (*overlap*).

### 2. Busca Léxica (FTS5)
- `chunks_fts`: Tabela virtual nativa do SQLite com tokenizador Porter (stemming) e Unicode61 para busca exata por palavras-chave via algoritmo BM25.

### 3. Busca Vetorial Nativa (`sqlite-vec`)
- `chunks_vec`: Tabela virtual do tipo `vec0` armazenando os vetores originais de 768 floats de 32 bits (3.072 bytes por chunk).

### 4. Busca Vetorial Ultracompacta (`chunks_turboquant`)
- `chunks_turboquant`: Armazena o vetor quantizado para 4-bits (384 bytes por chunk + float de escala). Permite economizar ~88% do espaço em disco.

### 5. Grafo de Conhecimento (`graph_nodes` e `graph_edges`)
- `graph_nodes`: Vértices do grafo (`note`, `tag`, `concept`).
- `graph_edges`: Arestas direcionadas (`links_to`, `tagged_as`).

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

## 📚 Arte Prévia e Influências

O My-Memory combina ideias comprovadas da literatura e projetos abertos de ponta:
- **`akitaonrails/ai-memory`**: Padrão *compile-not-retrieve*, busca híbrida com RRF (*Reciprocal Rank Fusion*) e Markdown no Git como fonte soberana da verdade.
- **`Graphify-Labs/graphify`**: Classificação epistêmica de arestas (`EXTRACTED` vs `INFERRED`), identificação de *God Nodes* (conceitos de alta centralidade) e cache incremental SHA-256.
- **`kepano/obsidian-skills`**: Especificações oficiais de *Obsidian Flavored Markdown* e geração de mapas mentais no formato aberto *JSON Canvas 1.0* (`.canvas`).

Consulte o detalhamento completo em **[`docs/REFERENCES.md`](REFERENCES.md)**.

