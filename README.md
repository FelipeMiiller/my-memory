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

#### 1. Indexar uma pasta com notas Markdown
Indexa documentos, extrai wikilinks, calcula embeddings no Ollama e gera índices `sqlite-vec`, `FTS5` e `TurboQuant`:
```bash
./bin/mem.exe index ./suas-notas
```

#### 2. Busca Semântica Padrão (k-NN + Grafo)
Executa busca por proximidade vetorial float32 com expansão de dependências no grafo:
```bash
./bin/mem.exe search "como funciona o fluxo de autenticação?"
```

#### 3. Busca Ultracompacta com TurboQuant (4-bits)
Executa a busca ultraveloz projetada sobre os vetores quantizados:
```bash
./bin/mem.exe search -tq "como funciona o fluxo de autenticação?"
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

### Ferramentas Expostas para a IA:

| Ferramenta | Descrição |
| :--- | :--- |
| `memory_search` | Busca semântica e contextual de chunks relevantes no banco de conhecimento. |
| `memory_get_neighbors` | Expande nós e documentos conectados no grafo através de travessia recursiva SQL. |

---

## 📁 Estrutura do Projeto

```text
my-memory/
├── cmd/
│   └── mem/                # Ponto de entrada da CLI (index, search, mcp)
├── internal/
│   ├── db/                 # Schemas SQLite, FTS5, sqlite-vec e queries CTE
│   ├── embedder/           # Integração com Ollama (nomic-embed-text)
│   ├── mcp/                # Servidor MCP (JSON-RPC 2.0, framing, tools)
│   ├── parser/             # Extração de [[wikilinks]], tags e chunking
│   └── turboquant/         # Rotações de Householder e quantizador 4-bit
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
