---
title: "Guia Prático de Uso: My-Memory"
category: skill
summary: "Guia prático passo a passo de comandos CLI, inicialização de vault, busca híbrida, inspeção de grafo e integração MCP."
tags: [guide, howto, cli, skill]
---

# 📖 Guia Prático de Uso: My-Memory

> **Transforme seu repositório de notas e documentações em uma memória autoconsciente (*Repository Brain*) com busca híbrida, quantização vetorial de 4-bit (TurboQuant) e grafo de conhecimento.**

---

## 📑 Índice
1. [Visão Geral Rápida](#1-visão-geral-rápida)
2. [Pré-requisitos e Compilação](#2-pré-requisitos-e-compilação)
3. [Passo 1: Inicializando o Repositório (`mem init`)](#3-passo-1-inicializando-o-repositório-mem-init)
4. [Passo 2: Configurando Banco e IA de Embeddings](#4-passo-2-configurando-banco-e-ia-de-embeddings)
5. [Passo 3: Indexando Arquivos (`mem index`)](#5-passo-3-indexando-arquivos-mem-index)
6. [Passo 4: Realizando Buscas Híbridas (`mem search`)](#6-passo-4-realizando-buscas-híbridas-mem-search)
7. [Passo 5: Análise Cirúrgica e Inteligência de Grafo](#7-passo-5-análise-cirúrgica-e-inteligência-de-grafo)
8. [Passo 6: Visualização Interativa do Grafo](#8-passo-6-visualização-interativa-do-grafo)
9. [Passo 7: Monitoramento Contínuo e Git Hooks](#9-passo-7-monitoramento-contínuo-e-git-hooks)
10. [Passo 8: Integração com Agentes de IA via MCP](#10-passo-8-integração-com-agentes-de-ia-via-mcp)
11. [Tabela de Referência Rápida de Comandos](#11-tabela-de-referência-rápida-de-comandos)

---

## 1. Visão Geral Rápida

O **My-Memory** unifica em uma única ferramenta:
- **Busca Léxica (BM25 / FTS5):** Encontra termos exatos, nomes de classes e funções.
- **Busca Vetorial com TurboQuant 4-bit:** Comprime vetores em 4-bit (384 bytes/chunk) preservando > 99% de precisão sem estourar memória.
- **Grafo de Conhecimento Obsidian:** Extrai automaticamente conexões de `[[wikilinks]]` e `#tags`.
- **Servidor MCP:** Expõe todo esse cérebro para assistentes como Cursor, Claude Code, Antigravity e Copilot.

---

## 2. Pré-requisitos e Compilação

### Pré-requisitos
- **Go 1.22+** instalado.
- **Ollama** (para embeddings locais):
  ```bash
  ollama pull nomic-embed-text
  ```
  *(Opcional: você também pode usar PostgreSQL com pgvector ou modelos alternativos como `bge-m3`)*.

### Compilação do Executável
No terminal da raiz do projeto, execute:
```bash
go build -o bin/mem.exe ./cmd/mem
```
*(No Linux/macOS: `go build -o bin/mem ./cmd/mem`)*.

> **Dica:** Adicione a pasta `bin` ao seu `PATH` para poder rodar apenas `mem <comando>` em qualquer terminal.

---

## 3. Passo 1: Inicializando o Repositório (`mem init`)

Para configurar o My-Memory no repositório atual:
```bash
mem init
```

### O que o comando faz:
Ele cria a pasta `.memory/` contendo:
- **`config.yaml`**: Define regras de quais pastas indexar (ex: `docs/`, `specs/`) e localização do banco.
- **`.gitignore`**: Impede que arquivos `.env` e bancos `.db` sejam enviados ao Git.
- **`.env.example`**: Modelo de variáveis de ambiente.
- **`AGENTS.md`**: Instruções prontas para você mover para a raiz (`cp .memory/AGENTS.md ./AGENTS.md`), instruindo qualquer IA a utilizar a memória do projeto.

---

## 4. Passo 2: Configurando Banco e IA de Embeddings

### A) Usando SQLite Local (Padrão - Zero Configuração)
Se não configurar nada, o My-Memory cria e utiliza automaticamente o banco local em:
`.memory/memory.db`

### B) Usando PostgreSQL com pgvector
Crie o arquivo `.memory/.env` a partir do exemplo:
```bash
cp .memory/.env.example .memory/.env
```
Edite `.memory/.env` informando sua string de conexão:
```env
MY_MEMORY_PG_URL=postgres://usuario:senha@localhost:5432/meu_banco?sslmode=disable
```
*(Quando esta variável está presente no `.env`, o My-Memory conecta **automaticamente ao PostgreSQL** em todos os comandos, sem precisar de flags adicionais)*.

### C) Escolhendo o Modelo de Embeddings
Por padrão, o My-Memory utiliza o modelo embutido **`nomic-embed-text`** na porta `11434` local.
Se desejar usar outro modelo ou apontar para um servidor remoto, descomente no `.memory/.env`:
```env
MY_MEMORY_EMBED_URL=http://localhost:11434
MY_MEMORY_EMBED_MODEL=bge-m3
MY_MEMORY_EMBED_DIM=1024
```

### D) Definindo quais pastas indexar
No arquivo `.memory/config.yaml`, configure a lista `include`:
```yaml
include:
  - "docs/**/*.md"      # Todas as notas em docs/
  - "specs/**/*.md"     # Todas as especificações em specs/
  - ".specs/**/*.md"    # Especificações técnicas
  - "README.md"         # Arquivo individual
```

---

## 5. Passo 3: Indexando Arquivos (`mem index`)

Para ler as notas Markdown, gerar o grafo e os embeddings:
```bash
mem index
```

### Características do Indexador:
1. **Cache Incremental SHA-256:** Arquivos inalterados são marcados como `⏩ [cached]` e pulados instantaneamente, sem gastar processamento de IA.
2. **Reindexação Forçada:** Para reprocessar todos os arquivos ignorando o cache:
   ```bash
   mem index --force
   ```
3. **Pruning Automático:** Notas apagadas do disco têm seus nós, chunks e vetores removidos automaticamente do banco.

---

## 6. Passo 4: Realizando Buscas Híbridas (`mem search`)

Execute buscas conceituais ou textuais no terminal:

```bash
# Busca híbrida (BM25 + vetores + grafo fundidos via RRF)
mem search "como funciona o cache incremental?"

# Busca exclusivamente léxica (palavras-chave e código)
mem search --mode fts "CalculateContentHash"

# Busca exclusivamente vetorial com TurboQuant 4-bit ativado
mem search --mode vector -tq "algoritmos de detecção de comunidades"

# Busca priorizando notas recentes (Decaimento Temporal)
mem search --decay --half-life 14 "mudanças recentes de arquitetura"
```

---

## 7. Passo 5: Análise Cirúrgica e Inteligência de Grafo

O My-Memory vai muito além de busca textual; ele entende topologia e dependências:

### Visualização Cirúrgica em 3 Colunas (`mem inspect`)
Exibe os nós que apontam para o arquivo (in-links), o resumo central com cálculo de risco e os nós citados (out-links):
```bash
mem inspect docs/ARCHITECTURE.md
```

### Análise de Raio de Destruição (`mem impact`)
Antes de refatorar ou excluir um arquivo, avalie quem quebra se ele mudar:
```bash
mem impact internal/db/graph.go --depth 2
```
Gera um relatório com **Score de Risco (0 a 100)** e lista de dependentes diretos e indiretos.

### Nós Centrais e Autoridade (`mem hubs`)
Descubra quais são os arquivos mais importantes e conectados do projeto:
```bash
# Por grau de conexões diretas (God Nodes)
mem hubs --algorithm degree

# Por autoridade estrutural de grafo (PageRank ponderado)
mem hubs --algorithm pagerank --top 10
```

### Detecção de Módulos e Comunidades (`mem clusters`)
Agrupa as notas em clusters temáticos através de propagação de rótulos (LPA):
```bash
mem clusters
```

### Auditoria de Saúde do Grafo (`mem doctor`)
Detecta links quebrados (*dead links*), notas órfãs e auto-referências:
```bash
# Apenas auditoria
mem doctor

# Auditoria com remoção e autocura de arestas quebradas
mem doctor --fix
```

### Conexões Inesperadas (`mem insights`)
Descobre notas com altíssima similaridade semântica que ainda não possuem links entre si:
```bash
mem insights --limit 5
```

---

## 8. Passo 6: Visualização Interativa do Grafo

Você pode abrir o grafo de conhecimento visualmente no seu navegador:

```bash
# Gera o visualizador interativo em HTML/SVG e abre no navegador padrão
mem graph view --open

# Foca o grafo a partir de uma nota específica (subgrafo)
mem graph view --root "docs/ARCHITECTURE.md" --depth 2 --open

# Exporta para formato JSON Canvas do Obsidian (.canvas)
mem export --canvas "docs/ARCHITECTURE.md" --out "meu_grafo.canvas"
```

O visualizador inclui simulação de forças físicas, busca por texto, filtro de PageRank, alternância de cores e exportação de imagem SVG.

---

## 9. Passo 7: Monitoramento Contínuo e Git Hooks

### Monitoramento em Segundo Plano (`mem watch`)
Fica observando suas pastas e reindexa notas no momento em que você salva os arquivos:
```bash
mem watch --debounce 500
```

### Indexação Automática no Git (`mem hook`)
Instala um Git *pre-commit hook* para garantir que seu cérebro de notas esteja sempre atualizado antes de cada commit:
```bash
# Instalar o hook
mem hook install

# Desinstalar o hook
mem hook uninstall
```

---

## 10. Passo 8: Integração com Agentes de IA via MCP

O My-Memory implementa o **Model Context Protocol (MCP)**, permitindo que IAs executem buscas e leiam grafos nativamente.

### Iniciar Servidor MCP:
```bash
# Modo stdio (usado por IDEs e editores de IA)
mem mcp

# Modo HTTP / SSE (acesso via rede ou containers)
mem mcp --port 38400
```

### Configuração no Cursor (`.cursor/mcp.json`):
```json
{
  "mcpServers": {
    "my-memory": {
      "command": "mem",
      "args": ["mcp"]
    }
  }
}
```

### Configuração no VS Code Copilot (`.vscode/mcp.json`):
```json
{
  "servers": {
    "my-memory": {
      "type": "stdio",
      "command": "mem",
      "args": ["mcp"]
    }
  }
}
```

*(Dica: você pode gerar esses arquivos de configuração automaticamente executando `mem init --all`)*.

---

## 11. Tabela de Referência Rápida de Comandos

| Comando | Descrição |
| :--- | :--- |
| `mem init [--all]` | Inicializa `.memory/` com `config.yaml`, `.gitignore`, `.env.example` e `AGENTS.md`. |
| `mem install [--tools]` | Auto-configura MCP em ferramentas de IA instaladas (Cursor, VS Code, Claude Desktop). |
| `mem index [--force]` | Indexa arquivos Markdown com cache SHA-256 e pruning de deletados. |
| `mem search "<pergunta>"` | Busca híbrida com RRF, vetores, BM25 e decaimento temporal opcional. |
| `mem path <de> <para>` | Encontra o caminho mais curto entre dois conceitos no grafo de conhecimento. |
| `mem inspect <nota>` | Visão cirúrgica em 3 colunas (in-links, nota central e out-links). |
| `mem impact <nota>` | Calcula o raio de destruição e dependentes reversos com score de risco. |
| `mem hubs` | Lista os nós centrais via conexões (degree) ou autoridade (PageRank). |
| `mem clusters` | Detecta módulos conceituais e clusters temáticos via LPA. |
| `mem doctor [--fix]` | Audita e limpa dead links, notas órfãs e calcula o Health Score. |
| `mem status` | Verifica a saúde do repositório e identifica notas desatualizadas (*staleness*). |
| `mem drift [--since] [--strict]` | Audita divergência semântica entre código e documentação (evita drift de memória). |
| `mem insights` | Descobre conexões surpreendentes entre notas sem links diretos. |
| `mem pack <nota>` | Empacota subgrafo e contexto para prompt de IA respeitando orçamento de tokens. |
| `mem open <nota>` | Abre a nota diretamente no seu editor de código preferido (VS Code, Cursor, Obsidian). |
| `mem graph view [--open]` | Gera visualizador HTML/SVG interativo com física e filtros. |
| `mem export --canvas <nota>` | Exporta subgrafo para formato JSON Canvas do Obsidian (.canvas). |
| `mem note <create\|append>` | Cria notas ou anexa seções com metadados e indexação atômica. |
| `mem compile --topic <termo>` | Sintetiza fontes em uma nota consolidada (*Compile-not-Retrieve*). |
| `mem watch` | File watcher com debounce para indexação automática contínua. |
| `mem hook <install\|uninstall>` | Configura Git pre-commit hook para indexação automática. |
| `mem mcp [--port <num>]` | Inicia o servidor MCP via stdio ou HTTP/SSE. |
| `mem bench` | Executa benchmarks de TurboQuant, RRF, SHA-256 e parsing. |
| `mem version` | Exibe versão, commit Git e arquitetura compilada. |

---

## 🎯 Dúvidas ou Suporte?
Para mais detalhes arquiteturais e matemáticos sobre o TurboQuant e algoritmos de grafo, consulte:
- [Documentação Técnica Completa (`docs/README.md`)](docs/README.md)
- [Guia de Integração de Agentes (`docs/AGENT_INTEGRATION_GUIDE.md`)](docs/AGENT_INTEGRATION_GUIDE.md)
- [Decisões de Arquitetura (`docs/adr/`)](docs/adr/README.md)
