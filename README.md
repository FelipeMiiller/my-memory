# 🧠 My-Memory

> **Transforme qualquer repositório ou vault Markdown em uma memória autoconsciente (*Repository Brain*) para você e seus Agentes de IA.**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![SQLite](https://img.shields.io/badge/SQLite-Unified_Engine-003B57?style=flat&logo=sqlite)](https://sqlite.org/)
[![sqlite-vec](https://img.shields.io/badge/sqlite--vec-Vector_Search-4169E1?style=flat)](https://github.com/asg017/sqlite-vec)
[![Paper](https://img.shields.io/badge/ICLR_2026-TurboQuant-FF6F00?style=flat)](https://arxiv.org/abs/2504.19874)
[![MCP](https://img.shields.io/badge/MCP-Model_Context_Protocol-8A2BE2?style=flat)](https://modelcontextprotocol.io/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 💡 Por que o My-Memory?

Agentes de IA (Claude Code, Cursor, Antigravity, GitHub Copilot, Windsurf) sofrem com limites de contexto e alucinações relacionais quando navegam em projetos grandes:
1. **Perda de Contexto:** Não é viável enviar centenas de arquivos para a janela de contexto sem estourar tokens e aumentar custos.
2. **Dependências Quebradas:** A IA altera um módulo sem saber quais componentes dependem dele no grafo.
3. **Decisões Esquecidas:** A IA refatora código ignorando decisões de arquitetura (ADRs) documentadas no passado.

O **My-Memory** resolve isso unificando **busca vetorial**, **grafo de conhecimento** e **busca léxica** em uma camada unificada de altíssima performance, acessível diretamente via terminal ou pelo **Model Context Protocol (MCP)**.

---

## ⚡ Arquitetura em 4 Pilares

```
                                [ Arquivos Markdown / Notas ]
                                              │
                         ┌────────────────────┴────────────────────┐
                         ▼                                         ▼
             [ Parser [[wikilinks]] & #tags ]            [ Chunking Semântico ]
                         │                                         │
                         ▼                                         ▼
               ( graph_nodes & edges )                  [ Ollama Embeddings ]
                         │                               (nomic-embed-text 768d)
                         │                                         │
                         │                         ┌───────────────┴───────────────┐
                         │                         ▼                               ▼
                         │                 [ sqlite-vec ]                 [ TurboQuant 4-bit ]
                         │                 (float32 k-NN)               (32 Reflexões Householder
                         │                         │                        + Bit-Packing)
                         │                         │                               │
                         ▼                         ▼                               ▼
       ┌───────────────────────────────────────────────────────────────────────────────────────┐
       │                              SQLite Local (memory.db)                                 │
       │  • documents         • chunks_vec (vec0)          • graph_nodes                       │
       │  • chunks            • chunks_turboquant (4-bit)  • graph_edges                       │
       │  • chunks_fts (BM25)                                                                  │
       └───────────────────────────────────────────────────────────────────────────────────────┘
                                                   ▲
                                                   │
                         ┌─────────────────────────┴─────────────────────────┐
                         ▼                                                   ▼
            [ CLI: mem search [-tq] ]                           [ Servidor MCP: mem mcp ]
             (Desenvolvedor no terminal)                           (Claude Code, Cursor, IAs)
```

1. **📦 SQLite Unificado & PostgreSQL Opcional:** Sem dependências pesadas (sem Neo4j, sem Pinecone, sem Elasticsearch). Armazena documentos, tabela virtual `sqlite-vec`, índice léxico `FTS5 (BM25)` e conexões de grafo em um arquivo local único ou em PostgreSQL corporativo com `pgvector`.
2. **🔬 TurboQuant (Google DeepMind, ICLR 2026):** Implementação pioneira em Go da quantização de 4-bits com 32 reflexões ortogonais de Householder ($R^T R = I$). Reduz o consumo vetorial em **~88%** (de 3.072 para 388 bytes por chunk) mantendo fidelidade $> 99\%$.
3. **🕸 Grafo Estilo Obsidian via SQL Recursivo:** Extrai conexões explícitas de notas (`[[links]]` e `#tags`), permitindo travessias relacionais, detecção de comunidades (LPA) e análise de raio de impacto em microssegundos.
4. **🔌 Model Context Protocol (MCP) Nativo:** Conecta-se diretamente aos assistentes de codificação de IA via `stdio` (JSON-RPC 2.0) ou rede HTTP/SSE, expondo ferramentas de busca e expansão de contexto.

---

## 📊 Eficiência TurboQuant (Google DeepMind, ICLR 2026)

Para um vetor de 768 dimensões (`nomic-embed-text`):

| Formato | Precisão | Bytes / Chunk | Redução de Espaço | Fidelidade Angular |
| :--- | :--- | :--- | :--- | :--- |
| **Float32 Padrão** | 32-bit float | 3.072 bytes | Linha de Base | 100.0% |
| **Int8 Clássico** | 8-bit int | 768 bytes | ~75.0% | ~95.0% (sensível a outliers) |
| **TurboQuant 4-bit** | 4-bit packed | **388 bytes** | **~87.4%** | **> 99.0%** (ortogonalmente protegido) |

---

## 🚀 Início Rápido (Quickstart)

### 1. Compilar
```bash
git clone https://github.com/FelipeMiiller/my-memory.git
cd my-memory
go build -o bin/mem.exe ./cmd/mem
```

### 2. Inicializar o Vault
```bash
./bin/mem.exe init
```
*(Gera a pasta `.memory/` com escopo de pastas, `.gitignore`, `.env.example` e `AGENTS.md`)*.

### 3. Indexar e Buscar
```bash
# Indexar notas com cache incremental SHA-256
./bin/mem.exe index

# Busca híbrida unificada (BM25 + Vetores + Grafo)
./bin/mem.exe search "como funciona o cache incremental?"
```

---

## 📚 Navegação da Documentação

Para mergulhar nos detalhes operacionais, matemáticos e de integração, consulte os guias dedicados:

| Guia | Para quem é | Descrição |
| :--- | :--- | :--- |
| 📖 [**`COMO_USAR.md`**](COMO_USAR.md) | **Desenvolvedores** | Manual prático de comandos CLI, exemplos de busca, configuração de PostgreSQL/SQLite e monitoramento em tempo real. |
| 🔍 [**`COMO_FUNCIONA.md`**](COMO_FUNCIONA.md) | **Engenheiros & Arquitetos** | Explicação profunda da arquitetura, matemática do TurboQuant, algoritmo RRF, CTEs recursivas e ciclo de vida do cache. |
| 🤖 [**`AGENT_INTEGRATION_GUIDE.md`**](docs/AGENT_INTEGRATION_GUIDE.md) | **Agentes de IA & Integrações** | Como integrar o My-Memory com Cursor, Claude Code, Copilot e Antigravity via MCP e regras `AGENTS.md`. |
| 🏛 [**`docs/adr/`**](docs/adr/README.md) | **Decisões de Engenharia** | 24 Registros de Decisão de Arquitetura (ADRs) documentados no formato padrão MADR. |

---

## 🤖 Ferramentas MCP para Assistentes de IA

Quando executado como servidor MCP (`mem mcp`), o My-Memory disponibiliza para a IA:

- `memory_search`: Busca híbrida (RRF) unificando FTS, vetores e grafo com decaimento temporal opcional.
- `memory_get_neighbors`: Expansão recursiva de nós e dependências conectadas.
- `memory_get_impact`: Análise de raio de destruição (*Blast Radius*) e dependentes reversos.
- `memory_get_clusters`: Detecção de comunidades e módulos temáticos via LPA ponderado.
- `memory_get_hubs`: Identificação de God Nodes e nós líderes por PageRank ponderado.
- `memory_doctor`: Auditoria de integridade do grafo com detecção de dead links e notas órfãs.
- `memory_write_note`: Criação de notas atômicas estruturadas com sincronização instantânea.
- `memory_compile_note`: Síntese de fragmentos recuperados (*Compile-not-Retrieve*).
- `memory_visualize_graph`: Exportação de visualizador interativo em HTML/SVG.

---

## 📁 Estrutura do Repositório

```text
my-memory/
├── cmd/mem/            # Ponto de entrada CLI (init, index, search, inspect, impact, mcp, etc.)
├── internal/
│   ├── config/         # Configuração declarativa, descoberta de vault e variáveis de ambiente
│   ├── db/             # Schemas SQLite, virtual tables sqlite-vec e queries recursivas CTE
│   ├── embedder/       # Cliente Ollama e resolução dinâmica de modelos de embedding
│   ├── graph/          # Algoritmos de grafo (LPA, modularidade Q, PageRank, Blast Radius, Inspector)
│   ├── graphview/      # Visualizador interativo HTML/SVG standalone com física de forças
│   ├── mcp/            # Servidor Model Context Protocol (stdio + HTTP/SSE)
│   ├── parser/         # Extração de wikilinks, tags, metadados e chunking
│   ├── store/          # Camada de armazenamento unificada e suporte a PostgreSQL com pgvector
│   ├── turboquant/     # Rotações ortogonais de Householder e quantização de 4-bit
│   └── watcher/        # File watcher em segundo plano com debouncing inteligente
├── docs/               # Documentação técnica detalhada e 24 ADRs
├── COMO_USAR.md        # Manual prático passo a passo para o usuário
├── COMO_FUNCIONA.md    # Explicação detalhada da arquitetura e funcionamento interno
├── AGENTS.md           # Regras operacionais para Agentes de IA
└── README.md           # Apresentação geral do projeto
```

---

## 🧪 Validação e Testes

O My-Memory conta com cobertura de testes unitários e de integração em todos os 14 pacotes Go:

```bash
# Executar todos os testes do repositório
go test -count=1 ./...

# Executar testes com relatório de cobertura
go test -v -cover ./internal/turboquant/... ./internal/graph/... ./internal/parser/...
```

---

## 📄 Licença

Distribuído sob a licença [MIT](LICENSE).
