---
title: "Arquitetura do My-Memory"
category: resource
summary: "Visão técnica e estrutural da arquitetura do My-Memory, incluindo persistência SQLite/Postgres, TurboQuant, RRF, Grafo e Carregamento Progressivo (L0/L1/L2)."
tags: [architecture, design, sqlite, postgres, turboquant, progressive-loading]
---

# Arquitetura do My-Memory

O **My-Memory** é uma engine de memória semântica e relacional local desenvolvida em Go, combinando o melhor de múltiplos ecossistemas e inovações:
1. **Obsidian**: Grafo de conhecimento orientado a notas Markdown (`[[wikilinks]]` e `#tags`).
2. **ai-memory**: Indexação rápida, chunking de texto, padrão *compile-not-retrieve* e busca semântica para IAs.
3. **Graphify**: Modelagem de nós e arestas com expansão relacional, status epistêmico e centralidade de nós.
4. **CodeGraph**: Monitoramento contínuo em tempo real com file watcher reativo, contexto cirúrgico para agentes e análise de impacto.
5. **TurboQuant (Google DeepMind, ICLR 2026)**: Compressão vetorial extrema em 4-bits com estimador de produto escalar não-viesado.
6. **OpenViking (ByteDance/Volcengine)**: Carregamento progressivo em 3 camadas (L0 Micro-Abstract, L1 Overview estrutural, L2 Detalhes) e Taxonomia Tripartida (`resource`, `memory`, `skill`).
7. **Atlas (EPAM/Open Knowledge Format - OKF v0.2)**: Arquitetura federada de cofre central (*Global Brain*) vs repositórios satélites (*Project Brains*), governança determinística de schemas e ciclo de vida staging.

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
           ( graph_nodes & edges )              [ Builtin Embedding ]
                    │                           (MiniLM multilíngue 384d)
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
- `chunks_vec`: Tabela virtual do tipo `vec0` armazenando os vetores originais de 384 floats de 32 bits (ou coluna `vector(384)` no PostgreSQL).

### 4. Busca Vetorial Ultracompacta (`chunks_turboquant`)
- `chunks_turboquant`: Armazena o vetor quantizado para 4-bits (384 bytes por chunk + float de escala). Permite economizar ~88% do espaço em disco.

### 5. Grafo de Conhecimento (`graph_nodes` e `graph_edges`)
- `graph_nodes`: Vértices do grafo (`note`, `tag`, `concept`, `decision`).
- `graph_edges`: Arestas direcionadas tipadas com status epistêmico (`EXTRACTED`, `INFERRED`, `TAG`) e pesos associados.

### 6. Tags do documento (`document_tags`, ADR-055)

As tags literais de cada documento ficam numa tabela própria, **em vez de** virarem
um nó do grafo:

```sql
CREATE TABLE IF NOT EXISTS document_tags (
    document_id    TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    tag           TEXT NOT NULL,
    tag_key       TEXT NOT NULL,
    resolved_doc_id TEXT,
    created_at    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (document_id, tag_key)
);
CREATE INDEX IF NOT EXISTS idx_document_tags_key ON document_tags(tag_key);
CREATE INDEX IF NOT EXISTS idx_document_tags_document ON document_tags(document_id);
```

> Fonte: `internal/db/schema.go:151-162` (SQLite) e `internal/store/postgres.go:84-94`
> (PostgreSQL — lá a identidade do documento é `(repository, id)`, e `repository` entra
> na chave primária).

A identidade da linha é `(document_id, tag_key)`: `tag_key` é a forma **canônica**
(minúscula) da tag, o que faz `#Tag` e `#TAG` serem a mesma linha, com `tag`
guardando a forma de exibição (o casing da primeira criação).

**A tag sobrevive à indexação antes da resolução fuzzy.** As linhas são gravadas
*antes* do passo que resolve a tag contra os títulos do vault — por isso a tag
**órfã**, a que não nomeia nenhuma nota, também fica registrada e é consultável
(`mem tags`, `mem tag`, `mem search --tag`, `memory_get_tags` / `memory_get_tag`).
A aresta do grafo continua sendo criada **só** para as tags **resolvidas**: a tabela
não reinstala o nó morto que a ADR-041 eliminou, e o Health Score não muda
(ADR-041 / ADR-055).

### 7. Code Graph do Vault de Repositório (ADR-061 / ADR-047)

O `mem index` orquestra um segundo estágio (`code_pipeline`) que faz parsing sintático dos arquivos-fonte e persiste metadados estruturados nas tabelas `code_*`:

- **`code_files`**: 1 linha por arquivo parseado (`path`, `language`, `content_hash` SHA-256, `ast_hash` SHA-256 dos símbolos, `size_bytes`, `indexed_at`). Reusa o `content_hash` da ADR-010 para cache incremental.
- **`code_symbols`**: símbolos extraídos (`function`, `method`, `class`, `interface`, `struct`, `import`, `constant`, `variable`) com `qualified_name`, `start_line`, `end_line`, `signature`, `doc_comment` e referência ao `code_file_id`.
- **`code_edges`**: arestas tipadas entre símbolos, com `confidence` e `epistemic_status`.
- **`code_edges_uncertain`**: arestas cujo destino **não** pôde ser resolvido unicamente, com `confidence` e `reason` — separadas do grafo principal para não poluí-lo.

```sql
CREATE TABLE IF NOT EXISTS code_files (
    id INTEGER PRIMARY KEY,
    repository TEXT NOT NULL DEFAULT 'default',
    path TEXT NOT NULL UNIQUE,
    language TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    content_hash TEXT NOT NULL,
    ast_hash TEXT,
    indexed_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS code_symbols (
    id INTEGER PRIMARY KEY,
    repository TEXT NOT NULL DEFAULT 'default',
    file_id INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    name TEXT NOT NULL,
    qualified_name TEXT,
    signature TEXT,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    start_col INTEGER,
    end_col INTEGER,
    doc_comment TEXT,
    parent_symbol_id INTEGER REFERENCES code_symbols(id) ON DELETE SET NULL,
    UNIQUE(file_id, kind, qualified_name, start_line)
);

CREATE TABLE IF NOT EXISTS code_edges (
    id INTEGER PRIMARY KEY,
    repository TEXT NOT NULL DEFAULT 'default',
    src_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    dst_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    file_id INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    confidence REAL NOT NULL DEFAULT 1.0,
    epistemic_status TEXT NOT NULL DEFAULT 'EXTRACTED',
    UNIQUE(src_symbol_id, dst_symbol_id, kind)
);

CREATE TABLE IF NOT EXISTS code_edges_uncertain (
    id INTEGER PRIMARY KEY,
    repository TEXT NOT NULL DEFAULT 'default',
    src_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
    dst_candidate_name TEXT NOT NULL,
    kind TEXT NOT NULL,
    confidence REAL NOT NULL,
    reason TEXT,
    file_id INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
    start_line INTEGER NOT NULL,
    UNIQUE(src_symbol_id, dst_candidate_name, kind)
);
```

> Fonte: `internal/db/schema.go` (SQLite) e `internal/store/postgres.go` (PostgreSQL —
> lá as tabelas `code_*` são criadas com os mesmos campos e a mesma identidade).

**Os cinco tipos de aresta.** O vocabulário é fechado (`codeast.ValidEdgeKinds()`):

| `kind` | Significado | `epistemic_status` típico |
| :--- | :--- | :--- |
| `calls` | O símbolo **chama** outro. A aresta guarda o **ponto de chamada** — arquivo:linha da *expressão* de chamada, não da definição. | `EXTRACTED` (callee resolvido no escopo do pacote) ou `INFERRED` (resolvido só por nome) |
| `imports` | O símbolo **importa** outro (via o símbolo `import` do arquivo). | `EXTRACTED` / `INFERRED` |
| `inherits` | O símbolo **herda** de outro (embedding de struct / extensão). | `EXTRACTED` / `INFERRED` |
| `uses_type` | O símbolo **usa** um tipo (campo, parâmetro, retorno, tipo explícito). | `EXTRACTED` / `INFERRED` |
| `references` | referência por **igualdade de nome**, sem evidência estrutural. | `INFERRED` |

**Cobertura por linguagem.** As arestas **estruturais** vêm do backend AST, que
hoje entende **Go** (`internal/codeast/go_parser.go`). Quando o parser não devolve
arestas — as demais linguagens caem no `MockParser`, o fallback determinístico
mantido pela ADR-047 — o pipeline usa o **caminho legado**: a varredida por nome,
que produz arestas `references` com `confidence = 0.8` (um match) ou
`code_edges_uncertain` com `confidence = 0.5` (dois ou mais matches), sempre com
provenance `INFERRED`. As duas rotas nunca correm no mesmo arquivo: quando o
backend entrega arestas, a varredida por nome é pulada (ela duplicaria as arestas
e não carregaria o call-site).

**Provenance epistêmica (ADR-011).** `code_edges.epistemic_status` tem dois valores: `EXTRACTED` (o destino foi resolvido no escopo do pacote — o `qualified_name` bate e é único) e `INFERRED` (o destino só foi resolvido por nome). O terceiro nível — **mais de um candidato, ou nenhum** — **nunca** vira aresta resolvida: a linha vai para `code_edges_uncertain` com `confidence` e `reason`, para revisão. É a mesma separação entre fato e inferência que a ADR-011 aplicou ao grafo de notas.

**Filtro por `repository`.** `code_files`, `code_symbols`, `code_edges` e `code_edges_uncertain` carregam a coluna `repository`. O `repo_id` viaja no contexto até a gravação (o `mem code-index --repo <slug>`) e as consultas do code graph filtram por ele — o mesmo modelo multi-tenant do grafo de notas (ADR-007 / ADR-033 / ADR-040).

Índices canônicos: `code_graph_kind_idx`, `code_symbols_kind_name_idx`, `code_files_language_idx`. Toda a criação é **idempotente** (`CREATE TABLE IF NOT EXISTS` + `CREATE INDEX IF NOT EXISTS`) e funciona identicamente em SQLite local e PostgreSQL/pgvector (ADR-040/ADR-001/ADR-016).

**Backend em Go puro.** O parser de produção é `internal/codeast/go_parser.go`: `go/parser` + `go/ast` + `go/token` da **stdlib**. **Não há CGO nem build tag** — o binário padrão já indexa código. O binário `treesitter_enabled.go` continua isolado atrás de `//go:build treesitter` e o `treesitter_disabled.go` expõe `newBackendParser()` devolvendo `NewGoParser()`; a ADR-047 **não** foi supersedida — o seu plano de deferral (3 gatilhos) segue intacto, hoje **0 de 3** disparados. O boot log do code pipeline diz, uma vez por processo: `code AST: backend go (stdlib); tree-sitter: disabled`.

> [!NOTE]
> A **ponte `documents` ↔ `code_symbols`** mora no **grafo único**
> (`graph_nodes`/`graph_edges`): cada símbolo vira um nó `type='code_symbol'` com
> id `code:<qualified_name>`, e cada documento que o menciona ganha uma aresta
> `relation='references_code'`. O `graph_edges` não tem CHECK de `relation`, então
> a ponte **não exige DDL novo**.
>
> A ponte carrega o status epistêmico (ADR-011) e resolve o match por prioridade:
> `qualified_name` literal no texto → `EXTRACTED`; caminho do arquivo literal →
> `EXTRACTED`; nome do símbolo com fronteira de palavra → `INFERRED` (é um palpite,
> não uma extração). O upsert é **idempotente** (rodar duas vezes não duplica a
> aresta; o conflito atualiza o status). Ela roda no **code stage** (depois de os
> símbolos serem persistidos) e no **indexador de Markdown** (para documento novo),
> e é o que permite ao `memory_code_neighbors` listar os documentos que citam um
> símbolo e ao blast radius alcançar as notas que dependem dele.

---

## 🔌 Estágio `code_pipeline` dentro do `mem index`

O fluxo de indexação do My-Memory é estendido com o estágio `code_pipeline`, executado **após** o pipeline Markdown tradicional:

```
                          [ Arquivos Markdown / Código-fonte ]
                                       │
                  ┌────────────────────┴────────────────────┐
                  ▼                                          ▼
       [ markdown_pipeline ]                       [ code_pipeline ]
       (chunks, embeddings,                      (parser Go puro +
        wikilinks, FTS, turboquant)                cache SHA-256 + ast_hash)
                  │                                          │
                  │                  ┌───────────────────────┴───────────────────────┐
                  ▼                  ▼                                               ▼
       [ graph_nodes/edges ]   [ code_files/code_symbols/code_edges ]      [ code_edges_uncertain ]
                  │                  │                                               │
                  └──────────────────┼───────────────────────────────────────────────┘
                                     ▼
                     SQLite Local (memory.db) / PostgreSQL pgvector
```

* **`mem index`**: roda o `code_pipeline` por padrão **quando há escopo de código configurado** (a seção `code:` do `.memory/config.yaml`), desligável com `--code-ast=false`. Sem escopo de código configurado, o comportamento é exatamente o de antes: só o `markdown_pipeline`. A falha do code stage **não derruba** o `mem index`.
* **Escopo de código (`code:`)**: a seleção de arquivos do code stage usa `code.include` / `code.exclude`, **separados** do `include`/`exclude` global de notas. O `include` global é compartilhado com o pipeline Markdown; incluir `**/*.go` nele indexaria código como nota — o pipeline Markdown ignora arquivo de código por definição (`codeast.IsCodeFile`), e o dono do código é o code stage.
* **`mem code-index`** (ADR-061): executa somente o `code_pipeline`, útil para re-indexações cirúrgicas após refactors massivos de código. Grava no **engine configurado** (PostgreSQL quando o vault está no PostgreSQL, SQLite quando está no SQLite).
* **Resolução de arestas em três níveis**: escopo de pacote (qualified name bate e é único) → `EXTRACTED`; só por nome (único no repositório) → `INFERRED`; mais de um candidato ou nada resolvido → `code_edges_uncertain` com `confidence` e `reason`.
* **Cache incremental**: se `content_hash` bate → skip parse; se difere mas o `ast_hash` é idêntico → atualiza o `content_hash` e skip o parse (mudanças apenas de whitespace/comentários).
* **Ponte notas ↔ código**: depois de persistir os símbolos, o code stage casa os documentos indexados com os `code_symbols` e grava a ponte `references_code` no grafo único (ver §7). O indexador de Markdown faz o mesmo para cada documento novo.

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
2. **HTTP com Server-Sent Events (SSE):** Servidor autônomo de rede (`mem mcp --port 38400`) com:
   * `GET /sse`: Stream SSE unidirecional com handshake `endpoint`.
   * `POST /message?sessionId=<uuid>`: Recepção de comandos JSON-RPC vinculados à sessão ativa.
   * `POST /mcp`: Chamadas diretas (stateless) para scripts rápidos ou ferramentas web.
   * `GET /health`: Monitoramento de status, uptime e ferramentas registradas.
   * Suporte a CORS com preflight `OPTIONS` para conexões diretas via navegador.

---

## 📦 Camada de Serving Estático do Transporte HTTP (ADR-059, ADR-060)

No transporte HTTP, o `HTTPServer` atende **duas** famílias de requisições: as
**quatro rotas do protocolo MCP** (intocadas) e as **rotas estáticas**, que
servem o painel de métricas e os artefatos de grafo.

```
requisição HTTP
      │
      ├── /health, /sse, /message, /mcp  ──►  http.ServeMux do protocolo
      │                                          (JSON-RPC / SSE / health)
      │
      └── qualquer outro path  ────────────►  StaticHandler
                                                 │
                                                 ├─ app/    build da viewer (viewer/dist)
                                                 └─ graph/  artefatos de memory_visualize_graph
                                                   em <repo>/.mymemory/www/
```

**Raiz servida:** `<repo>/.mymemory/www/` — um diretório **dedicado**, nunca a
raiz do vault. `.mymemory/` fica fora do escopo de indexação, de modo que os
artefatos gerados não entram no grafo nem no índice. As duas subáreas são:

| Subárea | Conteúdo | Rota |
|---|---|---|
| `app/` | build estático da SPA (`viewer/dist/`, publicado por `make viewer`) | `/` |
| `graph/` | artefatos HTML do grafo gerados por `memory_visualize_graph` | `/graph/<arquivo>.html` |

**Ordem de avaliação das rotas** (`internal/mcp/static_server.go`) — a ordem é
o contrato, não uma implementação:

1. resolver e normalizar o path (recusando segmentos com nome de volume);
2. se o caminho resolvido **escapa da raiz servida** → **404**;
3. se é um **arquivo existente** → servir o arquivo;
4. se aceita `text/html` **e não é** `/graph/` → `app/index.html` com **200**;
5. senão → **404**.

Duas regras derivam dessa ordem:

* O **`/graph/` é prefixo reservado** e **nunca** cai no fallback da SPA — um
  artefato inexistente é 404, não o `index.html`.
* O fallback só vale para **requisições de navegação** (`Accept: text/html`). Um
  asset faltando continua 404, para que o browser não receba HTML onde esperava
  JavaScript.

**Por que o dispatcher é separado do `http.ServeMux`.** O `http.ServeMux` do
`net/http` **canonicaliza** o path e responde **301** para qualquer path que
contenha `..`. Se as rotas estáticas passassem pelo mux, a tentativa de escape
da raiz servida seria respondida com um *redirect* — o cliente seguiria o
redirect e a barreira de path traversal nunca seria exercida. Por isso
`HTTPServer.route()` (`internal/mcp/http_server.go`) testa primeiro se o path
pertence ao protocolo: só o que é `/health`, `/sse`, `/message` ou `/mcp` vai
ao mux. O resto vai ao `StaticHandler` com o **path cru**, que decide 404
diante de qualquer tentativa de escape. A fronteira de raiz segue o padrão já
existente em `internal/compiler/note.go`.

**Degradação.** Sem `app/index.html`, a raiz da URL responde **503** com
mensagem simples, em vez de 404 para cada arquivo. Sem build, o MCP continua
funcionando — só o painel não existe.

**URL para o agente.** O `HTTPServer` publica o endereço **realmente vinculado**
(o `net.Listener` resolve porta `0` e `:38400` para a porta concreta) e o
`Server` o expõe às tools. Com o transporte ativo, `memory_visualize_graph`
devolve a **URL servida** do artefato e o caminho em disco; sem ele, devolve
só o caminho, sem erro.

## 🧬 `structuredContent` no Envelope de Resposta MCP

O envelope de `tools/call` (`internal/mcp/handlers.go`) carrega **duas**
camadas:

```go
type CallToolResult struct {
    Content           []ToolContent `json:"content"`
    IsError           bool          `json:"isError,omitempty"`
    StructuredContent any           `json:"structuredContent,omitempty"`
}
```

`content[]` é a saída **humana** (blocos de texto). `structuredContent` é a
saída **estruturada**, definida pelo MCP spec 2025-06-18, que a implementação
Go não tinha. Ele é **aditivo**: o campo é omitido quando vazio, de modo que
as tools que não o preenchem emitem exatamente o mesmo JSON de antes.

A forma do payload **é** o JSON tags do tipo Go devolvido pelo handler — não
existe schema paralelo, e o tipo Go é a fonte da verdade. As dez tools que o
preenchem são `memory_doctor`, `memory_get_hubs`, `memory_get_clusters`,
`memory_get_insights`, `memory_get_drift`, `memory_list_files`,
`memory_read_note`, `memory_note_outline`, `memory_file_metadata` e
`memory_visualize_graph`.

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

## 🏛️ Arquitetura Federada e Cofre Central de Conhecimento (ADR-033)

Para conectar o conhecimento corporativo transversal (padrões de engenharia, RFCs, diretrizes de segurança) ao código local de múltiplos repositórios:

```
                      [ ~/.memory/config.yaml ]
                   (Configuração Global Soberana)
                                 │
           ┌─────────────────────┴─────────────────────┐
           ▼                                           ▼
 [ Central Vault (Google Drive/OneDrive) ]   [ Repositório Local (Git) ]
 ├── standards/, architecture/, ...          ├── .memory/config.yaml (repo_id)
 ├── templates/ (ADR, RFC, Runbook, Spec)    ├── docs/adr/, .specs/, src/
 └── .memory/storage/memory.db (SQLite)      └── .memory/memory.db (SQLite)
           │                                           │
           └─────────────────────┬─────────────────────┘
                                 ▼
                     [ FederatedSearcher (RRF) ]
                     - Consultas Concorrentes
                     - Proveniência [local] vs [central]
                     - Fallback Não-Bloqueante
                                 │
                                 ▼
                     [ Servidor MCP / CLI Search ]
```

1. **Identidade Criptográfica Imutável (`repo_id`)**: Formato `repo_<12-hex-chars>` gerado no `mem init` e persistido em `.memory/config.yaml`.
2. **Zero-Credentials no Git**: A configuração local do repositório nunca armazena senhas de banco ou URLs. Toda a infraestrutura herda com segurança de `~/.memory/config.yaml`.
3. **Persistência Estável no PostgreSQL**: Um único banco estável atende todos os repositórios, com separação determinística por `repo_id` (e `repo_central`).
4. **Auto-Bootstrap de Cofre Virgem**: Cria automaticamente 11 pastas canônicas, templates Obsidian e o MOC `README.md` raiz.
5. **Busca Federada com RRF**: Unifica resultados locais e centrais com marcação de proveniência (`[local]` e `[central]`) e tolerância a desconexão de nuvem.
6. **Fallback FTS Não-Bloqueante** (commit [`33feb1d`](../../commit/33feb1d) — `fix(index): fallback to FTS chunk indexing when embedding generation is offline`): quando o provider de embeddings não está disponível, `mem index` continua indexando com vetor zero (`make([]float32, embedder.EmbeddingDim)`) e a busca híbrida cai automaticamente em FTS-only. O usuário é notificado via `sem provider de embedding; <docID> indexado em modo léxico FTS`, e via `fallback para busca textual FTS` no `mem search --mode hybrid`. Desde a migração para o provider embutido (ADR-035), as condições que disparam esse fallback são a ausência da tag `onnx` no build ou uma falha do modelo embutido — não a indisponibilidade de um servidor externo. O grafo, o FTS5 e o TurboQuant continuam 100% operacionais. Detalhes em [`docs/CENTRAL_VAULT.md`](CENTRAL_VAULT.md#4-modos-de-embedding-vetorial-vs-fallback-léxico).

---

## 📚 Arte Prévia e Influências

O My-Memory combina ideias comprovadas da literatura e projetos abertos de ponta:
- **`akitaonrails/ai-memory`**: Padrão *compile-not-retrieve*, busca híbrida com RRF (*Reciprocal Rank Fusion*) e Markdown no Git como fonte soberana da verdade.
- **`Graphify-Labs/graphify`**: Classificação epistêmica de arestas (`EXTRACTED` vs `INFERRED`), identificação de *God Nodes* (conceitos de alta centralidade) e cache incremental SHA-256.
- **`colbymchenry/codegraph`**: Grafo de conhecimento pré-indexado para agentes com eliminação de explorações cegas (*zero file reads*), sincronização viva reativa via watcher, alertas de staleness e análise de impacto.
- **`kepano/obsidian-skills`**: Especificações oficiais de *Obsidian Flavored Markdown* e geração de mapas mentais no formato aberto *JSON Canvas 1.0* (`.canvas`).

Consulte o detalhamento completo em **[`docs/REFERENCES.md`](REFERENCES.md)**.

