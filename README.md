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

Agentes de IA (Claude Code, Cursor, Antigravity, Copilot, Zed) sofrem com limites de contexto e alucinações quando navegam em projetos grandes:
1. **Perda de Contexto:** Não é possível enviar centenas de arquivos para a janela de contexto sem estourar limites e aumentar custos.
2. **Dependências Quebradas:** A IA altera um módulo sem saber quem depende dele no grafo.
3. **Decisões Esquecidas:** A IA refatora código ignorando decisões de arquitetura (ADRs) documentadas no passado.

O **My-Memory** resolve isso unificando **busca vetorial**, **grafo de conhecimento** e **busca léxica** em um único arquivo local `memory.db` de altíssima performance, acessível diretamente via linha de comando ou pelo **Model Context Protocol (MCP)**.

---

## ⚡ Os Quatro Pilares

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

1. **📦 SQLite Unificado:** Elimina a necessidade de múltiplos bancos de dados (sem Neo4j, sem Pinecone, sem Elasticsearch). Armazena documentos relacionais, tabela virtual `sqlite-vec`, busca léxica `FTS5 (BM25)` e conexões de grafo em um único arquivo.
2. **🔬 TurboQuant (Google DeepMind, ICLR 2026):** Implementação pioneira em Go da quantização de 4-bits com 32 reflexões ortogonais de Householder ($R^T R = I$) e estimador não-viesado de produto escalar. Reduz o armazenamento vetorial em **~88%** (de 3.072 para 388 bytes por chunk) mantendo correlação > 99%.
3. **🕸 Grafo Estilo Obsidian via SQL Recursivo (`WITH RECURSIVE`):** Extrai conexões explícitas de notas (`[[links]]` e `#tags`), permitindo travessias relacionais e descoberta de vizinhos de dependência em microssegundos com CTEs nativas do SQLite.
4. **🔌 Model Context Protocol (MCP) Nativo:** Conecta-se diretamente aos assistentes de codificação de IA via `stdio` (JSON-RPC 2.0), expondo ferramentas de busca e expansão de contexto sem abrir portas de rede.

---

## 📊 Comparativo de Eficiência: TurboQuant

Para um embedding de 768 dimensões (`nomic-embed-text`):

| Formato | Precisão | Bytes / Chunk | Redução de Espaço | Preservação de Similaridade |
| :--- | :--- | :--- | :--- | :--- |
| **Float32 Padrão** | 32-bit float | 3.072 bytes | Linha de Base | 100.0% |
| **Int8 Clássico** | 8-bit int | 768 bytes | ~75.0% | ~95.0% (sensível a outliers) |
| **TurboQuant 4-bit** | 4-bit packed | **388 bytes** | **~87.4%** | **> 99.0%** (ortogonalmente protegido) |

> O TurboQuant dissipa os canais discrepantes (*outliers*) através de rotações aleatórias ortogonais, garantindo que o empacotamento em 4-bits conserve ângulos e distâncias sem distorção.

---

## 🚀 Como Começar

### Pré-requisitos
- [Go](https://go.dev/) 1.22 ou superior
- [Ollama](https://ollama.ai/) rodando localmente com o modelo de embedding:
  ```bash
  ollama pull nomic-embed-text
  ```

### Instalação

Clone o repositório e compile o binário:

```bash
git clone https://github.com/FelipeMiiller/my-memory.git
cd my-memory
go build -o bin/mem.exe ./cmd/mem
```

### Uso da CLI

### Uso da CLI

#### 1. Inicializar a Configuração Declarativa do Vault (`mem init`)
Cria a pasta `.memory/` e o arquivo `config.yaml` documentado com as regras de indexação, persistência e busca do vault:
```bash
./bin/mem.exe init --repo "minha-org/meu-vault"
```

#### 2. Indexar com Auto-Scoping e Filtragem Glob (`mem index`)
Indexa documentos respeitando as regras declaradas de `include` e `exclude`. Se nenhum caminho for especificado, detecta automaticamente a raiz do vault:
```bash
# Auto-scoping ativo (descobre .memory/config.yaml na pasta atual ou ascendente):
./bin/mem.exe index

# Ou especificando um diretório explícito:
./bin/mem.exe index ./suas-notas
```

#### 3. Monitoramento em Tempo Real (`mem watch`)
Monitora o vault em segundo plano com debouncing inteligente (padrão 500ms) e reindexa cirurgicamente apenas notas criadas, modificadas ou deletadas:
```bash
./bin/mem.exe watch
```

#### 4. Automação Pré-Commit com Git Hooks (`mem hook`)
Garante que notas Markdown sejam sincronizadas no grafo e no índice vetorial antes de cada commit no Git:
```bash
# Instalar pre-commit hook:
./bin/mem.exe hook install

# Desinstalar pre-commit hook:
./bin/mem.exe hook uninstall
```

#### 5. Busca Híbrida e Decaimento Temporal (RRF + FTS5 + k-NN + Grafo)
Executa busca híbrida unificada via Reciprocal Rank Fusion (RRF), com suporte opcional a decaimento temporal exponencial (ADR-015) para priorizar notas mais recentes:
```bash
./bin/mem.exe search "como funciona o fluxo de autenticação?"
# Com decaimento temporal ativado (meia-vida de 15 dias, peso 0.5):
./bin/mem.exe search --decay --half-life 15 --decay-weight 0.5 "decisões recentes de arquitetura"
```

#### 6. Busca Ultracompacta com TurboQuant (4-bits)
Executa a busca ultraveloz projetada sobre os vetores quantizados:
```bash
./bin/mem.exe search -tq "como funciona o fluxo de autenticação?"
```

#### 7. Criação e Edição de Notas Atômicas (`mem note`)
Permite criar notas Markdown com frontmatter estruturado (`title`, `type`, `tags`, `aliases`) e apensar seções sob cabeçalhos, acionando sincronização cirúrgica imediata no banco:
```bash
# Criar nova nota atômica:
./bin/mem.exe note create --title "Arquitetura de Mensageria" --tags "kafka,eventos" --type "concept" concepts/mensageria.md

# Anexar seção cirurgicamente:
./bin/mem.exe note append --heading "## Boas Práticas" concepts/mensageria.md "Utilizar idempotência nos consumidores."
```

#### 8. Síntese e Compilação de Tópicos (`mem compile` / *Compile-not-Retrieve*)
Recupera os fragmentos mais relevantes sobre um tema e compila automaticamente uma nota atômica estruturada com backlinks (`[[rel:derived_from:...]]`), eliminando custos de contexto repetitivo para agentes de IA:
```bash
./bin/mem.exe compile --topic "fluxo de autenticação e tokens" --out syntheses/auth.md --limit 5
```

#### 9. Metadados de Versão (`mem version`)
Exibe versão SemVer, hash Git do commit, data de compilação e arquitetura em texto ou JSON estruturado:
```bash
./bin/mem.exe version
./bin/mem.exe version --json
```

#### 10. Visualização Interativa de Grafo (`mem graph view` / `mem graph export`)
Gera visualização espacial interativa do grafo da memória em HTML/SVG 100% autocontido (Zero-CDN), com simulação de física de forças, escala proporcional por PageRank, busca em tempo real, alternância de cores por cluster e painel lateral com links Obsidian:
```bash
# Abrir visualização no navegador padrão:
./bin/mem.exe graph view

# Visualizar subgrafo de uma nota específica:
./bin/mem.exe graph view --root "concepts/auth.md" --depth 2

# Exportar para arquivo HTML estático:
./bin/mem.exe graph export --out "grafo.html"
```

#### 11. Detecção de Comunidades e Clusters (`mem clusters`)
Detecta agrupamentos temáticos e partições conceituais densas no grafo de conhecimento usando o *Weighted Label Propagation Algorithm* (LPA) ponderado e calcula a Modularidade Newman-Girvan \(Q\):
```bash
# Exibir clusters com tamanho >= 2 em tabela alinhada:
./bin/mem.exe clusters

# Incluir nós isolados (tamanho >= 1):
./bin/mem.exe clusters --min-size 1

# Exportar dados de clusters e modularidade em JSON:
./bin/mem.exe clusters --json
```

#### 12. Servidor MCP Multi-Modo (Stdio ou HTTP/SSE de Rede)
Inicia o servidor Model Context Protocol via `stdio` (padrão local para Cursor e Claude Desktop) ou como servidor de rede HTTP/SSE com suporte a múltiplos clientes concorrentes, CORS e endpoints de diagnóstico:
```bash
# Modo stdio clássico (processo filho):
./bin/mem.exe mcp

# Modo servidor de rede HTTP/SSE na porta 38400:
./bin/mem.exe mcp --port 38400

# Exposto na rede local para múltiplos agentes:
./bin/mem.exe mcp --host 0.0.0.0 --port 38400
```

---

## 🤖 Integração com Agentes de IA (MCP)

O `my-memory` pode ser configurado como servidor **MCP (Model Context Protocol)** em qualquer IDE ou ferramenta de IA compatível.

### Configuração no Cursor (`.cursor/mcp.json`) ou VS Code (`.vscode/mcp.json`):

```json
{
  "mcpServers": {
    "my-memory": {
      "command": "mem",
      "args": ["mcp", "--db", ".memory/memory.db"]
    }
  }
}
```

### Configuração no Claude Code (`~/.claude.json`):

```json
{
  "mcpServers": {
    "my-memory": {
      "command": "/caminho/absoluto/bin/mem",
      "args": ["mcp"]
    }
  }
}
```

### Configuração Remota via HTTP/SSE (Qualquer Agente / Nuvem):

```json
{
  "mcpServers": {
    "my-memory": {
      "url": "http://127.0.0.1:38400/sse"
    }
  }
}
```

### Ferramentas Expostas para a IA:

| Ferramenta | Descrição |
| :--- | :--- |
| `memory_search` | Busca híbrida (RRF) unificando FTS, vetores e grafo, com decaimento temporal exponencial opcional (`decay`, `half_life`, `decay_weight`). |
| `memory_get_neighbors` | Expande nós e documentos conectados no grafo através de travessia recursiva SQL. |
| `memory_get_clusters` | Detecta partições temáticas e comunidades no grafo via LPA ponderado, calculando modularidade Newman-Girvan \(Q\), nós líderes e tipos dominantes. |
| `memory_visualize_graph` | Exporta uma visualização interativa do grafo da memória para uma página HTML/SVG standalone com física de forças, busca e filtros. |
| `memory_write_note` | Grava ou atualiza notas atômicas no vault com frontmatter e conexões tipadas, disparando sincronização imediata no grafo. |
| `memory_append_section` | Anexa cirurgicamente blocos de texto sob seções existentes ou novas sem quebrar a estrutura do documento. |
| `memory_compile_note` | Compila e sintetiza conhecimento sobre um tópico a partir de buscas híbridas (padrão *Compile-not-Retrieve*), gerando nota com backlinks. |
| `memory_get_hubs` | Identifica nós centrais de alta densidade por grau (God Nodes) ou por autoridade estrutural (PageRank ponderado). |
| `memory_get_insights` | Descobre conexões latentes e surpreendentes (*Surprising Connections*) entre conceitos sem links diretos. |
| `memory_doctor` | Audita e repara a integridade do grafo (dead links, notas órfãs, self-loops e Health Score). |
| `memory_export_canvas` | Exporta subgrafos no formato espacial JSON Canvas 1.0 (`.canvas`) do Obsidian. |

---

## 📁 Estrutura do Projeto

```text
my-memory/
├── cmd/
│   └── mem/                # Ponto de entrada da CLI (init, index, search, clusters, mcp, doctor, hubs, insights, export)
├── internal/
│   ├── config/             # Configuração declarativa, descoberta ascendente e filtragem glob
│   ├── db/                 # Schemas SQLite, FTS5, sqlite-vec e queries CTE
│   ├── embedder/           # Integração com Ollama (nomic-embed-text)
│   ├── graph/              # Algoritmos de grafo (LPA, modularidade Q, PageRank ponderado, God Nodes)
│   ├── graphview/          # Construtor de grafo interativo, clusters e templates HTML/SVG
│   ├── mcp/                # Servidor MCP (stdio + HTTP/SSE, framing JSON-RPC 2.0, tools)
│   ├── parser/             # Extração de [[wikilinks]], tags e chunking
│   ├── repo/               # Detecção e normalização de slug de repositório Git
│   ├── store/              # Interfaces unificadas de armazenamento e PostgreSQL com pgvector
│   ├── turboquant/         # Rotações de Householder e quantizador 4-bit
│   └── watcher/            # File watcher em tempo real, debouncing e reindexação cirúrgica
├── docs/                   # Documentação detalhada e ADRs
│   ├── adr/                # Decisões de Arquitetura em formato MADR
│   ├── ARCHITECTURE.md     # Detalhamento de schemas e fluxo de dados
│   ├── TURBOQUANT.md       # Fundamentação matemática e teoria do TurboQuant
│   ├── CLI_GUIDE.md        # Manual da CLI
│   └── REPOSITORY_BRAIN.md # Guia de repositório autoconsciente
└── AGENTS.md               # Instruções de navegação para Agentes de IA
```

---

## 📚 Documentação Técnica e Decisões

- 📘 [**Arquitetura do Sistema (`docs/ARCHITECTURE.md`)**](docs/ARCHITECTURE.md)
- 🔬 [**Matemática do TurboQuant (`docs/TURBOQUANT.md`)**](docs/TURBOQUANT.md)
- 💻 [**Guia Completo da CLI (`docs/CLI_GUIDE.md`)**](docs/CLI_GUIDE.md)
- 🧠 [**Arquitetura do Repository Brain (`docs/REPOSITORY_BRAIN.md`)**](docs/REPOSITORY_BRAIN.md)
- 🏛 [**Registros de Decisões de Arquitetura (`docs/adr/`)**](docs/adr/README.md)
  - [ADR-001: SQLite como Camada Unificada](docs/adr/001-uso-de-sqlite-como-camada-unificada-de-dados.md)
  - [ADR-002: Adoção de Go](docs/adr/002-adocao-de-go-como-linguagem-principal.md)
  - [ADR-003: Compressão 4-bit TurboQuant](docs/adr/003-compressao-vetorial-de-4-bit-via-turboquant.md)
  - [ADR-004: Grafo com SQL Recursivo](docs/adr/004-modelagem-de-grafo-com-recursive-ctes.md)
  - [ADR-005: Markdown e Wikilinks como Fonte](docs/adr/005-markdown-com-wikilinks-como-fonte-de-verdade.md)
  - [ADR-006: Integração via MCP](docs/adr/006-integracao-com-agentes-de-ia-via-mcp.md)
  - [ADR-007: Suporte Opcional a PostgreSQL + pgvector](docs/adr/007-suporte-opcional-a-postgresql-com-pgvector-e-multi-repositorio.md)
  - [ADR-008: Interoperabilidade Obsidian e JSON Canvas 1.0](docs/adr/008-interoperabilidade-obsidian-flavored-markdown-e-json-canvas.md)
  - [ADR-009: Busca Híbrida com Reciprocal Rank Fusion (RRF)](docs/adr/009-busca-hibrida-com-reciprocal-rank-fusion-rrf.md)
  - [ADR-010: Cache Incremental de Indexação com SHA-256](docs/adr/010-cache-incremental-de-indexacao-com-sha256.md)
  - [ADR-011: Arestas Epistêmicas e God Nodes / Hubs](docs/adr/011-arestas-epistemicas-e-god-nodes.md)
  - [ADR-012: Benchmarks de Performance e Conexões Inesperadas](docs/adr/012-benchmarks-e-conexoes-inesperadas.md)
  - [ADR-013: Higiene de Grafo, Pruning Incremental e Linter Doctor](docs/adr/013-higiene-de-grafo-pruning-e-doctor.md)
  - [ADR-014: Centralidade de Grafo com PageRank Ponderado](docs/adr/014-centralidade-de-grafo-com-pagerank-ponderado.md)
  - [ADR-015: Decaimento Temporal Exponencial na Busca Híbrida](docs/adr/015-decaimento-temporal-exponencial-na-busca-hibrida.md)
  - [ADR-016: Configuração Declarativa e Auto-Scoping de Vault](docs/adr/016-configuracao-declarativa-e-auto-scoping-de-vault.md)
  - [ADR-017: Indexação Contínua em Tempo Real com File Watcher e Git Hooks](docs/adr/017-indexacao-continua-com-file-watcher-e-git-hooks.md)
  - [ADR-018: Padrão Compile-not-Retrieve e Escrita Bilateral na Memória via MCP](docs/adr/018-padrao-compile-not-retrieve-e-escrita-bilateral-mcp.md)
  - [ADR-019: Versionamento Semântico Automatizado e Criação de Tags no CI](docs/adr/019-versionamento-semantico-e-tagging-ci.md)
  - [ADR-020: Visualizador Interativo de Grafo em HTML/SVG Standalone](docs/adr/020-visualizador-interativo-de-grafo-em-html-svg.md)
  - [ADR-021: Servidor MCP com Transporte HTTP e Server-Sent Events (SSE)](docs/adr/021-servidor-mcp-com-transporte-http-sse.md)
  - [ADR-022: Detecção de Comunidades e Clusters no Grafo de Conhecimento](docs/adr/022-deteccao-de-comunidades-e-clusters-no-grafo.md)

---

## 🧪 Testes

Execute a suíte de testes com validação de vetores e quantização:

```bash
# Executar todos os testes
go test -v ./internal/...

# Relatório de cobertura dos componentes principais
go test -v -cover ./internal/parser/... ./internal/turboquant/... ./internal/mcp/...
```

---

## 📄 Licença

Distribuído sob a licença [MIT](LICENSE).
