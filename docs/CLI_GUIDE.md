# Guia da Linha de Comando (CLI)

O executável `mem` fornece uma interface direta para indexação e consulta semântica da base de conhecimento.

---

## 🛠 Compilação

Para compilar o binário em Go:

```bash
# Na raiz do projeto:
go build -o bin/mem.exe ./cmd/mem
```

---

## 📖 Comandos Disponíveis

### 1. `mem index <pasta>`
Percorre recursivamente a pasta especificada procurando arquivos `.md`:
* Salva os metadados do documento no SQLite.
* Extrai os `[[wikilinks]]` e `#tags` inserindo as arestas de relacionamento no grafo.
* Divide o conteúdo em chunks com sobreposição.
* Gera os embeddings de 768 dimensões via Ollama (`nomic-embed-text`).
* Insere no `sqlite-vec` (vetor completo) e no `chunks_turboquant` (comprimido em 4-bits).

**Exemplo:**
```bash
./bin/mem.exe index C:/meu-vault-obsidian
# ou dentro de um projeto:
./bin/mem.exe index ./docs
```

---

### 2. `mem search "<pergunta>"`
Realiza a busca semântica k-NN padrão utilizando a extensão `sqlite-vec` ou `pgvector`:
* Calcula o embedding da pergunta.
* Busca os chunks com menor distância de cosseno.
* Realiza a **expansão de grafo** via SQL recursivo para trazer notas conectadas.

**Exemplo:**
```bash
./bin/mem.exe search "como funciona o fluxo de autenticacao?"
# Filtrando por repositório ou conectando ao PostgreSQL:
./bin/mem.exe search --postgres "postgres://user:pass@localhost:5432/memory?sslmode=disable" --repo "meu-org/meu-projeto" "fluxo de autenticacao"
```

---

### 3. `mem search -tq "<pergunta>"` (Modo TurboQuant)
Realiza a busca ultrarrápida utilizando os blocos compactados em 4-bits no SQLite:
* Rotaciona o vetor da query via Householder.
* Calcula o produto escalar não-viesado diretamente sobre os BLOBs comprimidos.
* Aplica a mesma expansão de grafo sobre os resultados.

**Exemplo:**
```bash
./bin/mem.exe search -tq "qual a regra para calculo de comissao?"
```

---

### 4. `mem mcp [--db <caminho>] [--postgres <url>] [--repo <slug>]`
Inicia o servidor Model Context Protocol (MCP) via `stdio` (JSON-RPC 2.0):
* Exposto para agentes como Claude Code, Cursor, Windsurf e Antigravity.
* Fornece as ferramentas `memory_search` e `memory_get_neighbors`.
* Suporta isolamento por repositório e conexões via SQLite local ou PostgreSQL centralizado.

**Exemplo:**
```bash
./bin/mem.exe mcp --db memory.db
# Ou conectado a uma base central PostgreSQL:
./bin/mem.exe mcp --postgres "postgres://user:pass@localhost:5432/memory?sslmode=disable" --repo "meu-org/meu-projeto"
```

---

### 5. `mem export --canvas <nota> [--depth 1] [--out <arquivo.canvas>]`
Exporta o subgrafo relacional centrado em uma nota para o formato aberto **JSON Canvas 1.0 (`.canvas`)** do Obsidian:
* Realiza a travessia de vizinhos conectados até a profundidade indicada.
* Calcula o posicionamento radial espacial para visualização sem nós sobrepostos.
* Gera arquivo `.canvas` pronto para ser aberto diretamente no Obsidian.

**Exemplo:**
```bash
./bin/mem.exe export --canvas "Arquitetura" --depth 2 --out "mapa_arquitetura.canvas"
```

