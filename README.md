# My-Memory (Go + SQLite-Vec + TurboQuant + Grafo)

Sistema local de memória semântica e relacional para desenvolvedores e agentes de IA, combinando as melhores características de:
- **Obsidian**: Leitura de notas Markdown locais e preservação de links conceituais (`[[wikilinks]]` e `#tags`).
- **ai-memory**: Indexação rápida, chunking semântico e busca vetorial de alta performance.
- **Graphify**: Estruturação de conhecimento em grafo (nós e arestas) com expansão relacional (Recursive CTEs).
- **TurboQuant (Google DeepMind, ICLR 2026)**: Quantização extrema de vetores para 4-bits com rotação ortogonal aleatória e estimador não-viesado de produto escalar.

---

## 🏛 Arquitetura

O sistema roda em um **único banco de dados SQLite local** com quatro pilares:
1. **`sqlite-vec`**: Busca vetorial (k-NN) acelerada por hardware (3072 bytes por vetor float32).
2. **`TurboQuant (4-bit)`**: Compressão de 768 dimensões para apenas **384 bytes** por chunk (redução de ~88% de espaço) mantendo correlação > 99%.
3. **`FTS5`**: Busca léxica (Full-Text Search com algoritmo BM25).
4. **`graph_nodes` & `graph_edges`**: Modelagem de grafos consultados via SQL recursivo (`WITH RECURSIVE`).

---

## 🚀 Como Começar

### Pré-requisitos
- [Go](https://go.dev/) (1.22+)
- [Ollama](https://ollama.ai/) com o modelo de embedding local:
  ```bash
  ollama pull nomic-embed-text
  ```

### Instalação e Execução
```bash
# Compilar o binário
go build -o bin/mem.exe ./cmd/mem

# Indexar uma pasta com notas Markdown (gera vetores padrão e comprimidos)
./bin/mem.exe index /caminho/para/vault

# Fazer busca padrão via sqlite-vec com expansão de grafo
./bin/mem.exe search "como funciona a autenticação?"

# Fazer busca ultracompacta via TurboQuant 4-bit
./bin/mem.exe search -tq "como funciona a autenticação?"
```

---

## 🔬 Como Funciona o TurboQuant neste Projeto

1. **Rotação Ortogonal Aleatória (Householder)**: Multiplica o vetor por um produto de reflexões de Householder determinísticas ($R^T R = I$). Isso espalha os canais discrepantes (*outliers*) de forma homogênea em todas as dimensões sem alterar a norma e os ângulos.
2. **Quantização de 4-bits**: Mapeia as 768 dimensões para 15 níveis simétricos (de -7 a +7) e empacota 2 valores por byte (*bit-packing*).
3. **Estimador Não-Viesado**: O produto escalar $\langle R q, R x \rangle$ tem esperança matemática exata em relação a $\langle q, x \rangle$.
