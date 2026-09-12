# My-Memory (Go + SQLite-Vec + Grafo)

Sistema local de memória semântica e relacional para desenvolvedores e agentes de IA, combinando as melhores características de:
- **Obsidian**: Leitura de notas Markdown locais e preservação de links conceituais (`[[wikilinks]]` e `#tags`).
- **ai-memory**: Indexação rápida, chunking semântico e busca vetorial de alta performance.
- **Graphify**: Estruturação de conhecimento em grafo (nós e arestas) com expansão relacional (Recursive CTEs).

---

## 🏛 Arquitetura

O sistema roda em um **único banco de dados SQLite local** com três pilares:
1. **`sqlite-vec`**: Busca vetorial (k-NN) acelerada por hardware.
2. **`FTS5`**: Busca léxica (Full-Text Search com algoritmo BM25).
3. **`graph_nodes` & `graph_edges`**: Modelagem de grafos consultados via SQL recursivo (`WITH RECURSIVE`).

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

# Indexar uma pasta com notas Markdown
./bin/mem.exe index /caminho/para/vault

# Fazer uma busca semântica com expansão de grafo
./bin/mem.exe search "como funciona a autenticação?"
```
