# Referências de Arquitetura e Arte Prévia (Prior Art)

Este documento registra os projetos, artigos e ecossistemas de referência que fundamentam, inspiram e refinam as decisões arquiteturais do **My-Memory**.

---

## 📚 1. Projetos de Referência

### 1.1. [Graphify](https://github.com/Graphify-Labs/graphify) (`Graphify-Labs/graphify`)
* **Autor:** Safi Shamsi
* **O que é:** Skill para agentes (Claude Code) que lê qualquer código, documentos, PDFs e imagens, construindo um grafo de conhecimento multimodal persistente com redução de até 71.5x em tokens de contexto por query.
* **Pontos de inspiração para o My-Memory:**
  1. **Tipagem Epistêmica de Arestas (`EXTRACTED` vs `INFERRED` vs `AMBIGUOUS`):**
     - O My-Memory deve distinguir arestas explícitas escritas pelo autor (`EXTRACTED` via `[[wikilinks]]`) de arestas semânticas sugeridas por proximidade de embeddings (`INFERRED` via k-NN).
  2. **Métricas de Topologia do Grafo (*God Nodes* & *Surprising Connections*):**
     - Identificar nós centrais (*God Nodes* - conceitos de alto grau de entrada/saída) para servir como pontos de entrada (*entrypoints*) de navegação para agentes de IA.
     - Detectar conexões inesperadas (*Surprising Connections*) cruzando similaridade vetorial com distância no grafo.
  3. **Multi-formato de Exportação:**
     - Exportação para múltiplos destinos visuais: Obsidian (`.canvas`, pastas de vault), formato JSON persistente (`graph.json`) e visualização interativa em HTML/SVG.
  4. **Cache Incremental com Hashing SHA-256:**
     - Indexar apenas arquivos novos ou alterados comparando o SHA-256 do conteúdo, evitando regenerar embeddings caros no Ollama/OpenAI.
  5. **Gatilhos Automáticos (Git Hook & `--watch`):**
     - Manter o grafo atualizado automaticamente a cada `git commit` ou via file-watcher em segundo plano.

---

### 1.2. [ai-memory](https://github.com/akitaonrails/ai-memory) (`akitaonrails/ai-memory`)
* **Autor:** Fabio Akita (@akitaonrails)
* **O que é:** Motor de memória de longo prazo para agentes de IA em Rust, com arquitetura local MCP/HTTP, unificando Markdown versionado em Git como fonte de verdade soberana, SQLite FTS5, vetores e grafos de entidades com fusão RRF.
* **Pontos de inspiração para o My-Memory:**
  1. **Busca Híbrida com RRF (Reciprocal Rank Fusion):**
     - Não depender exclusivamente de distância vetorial (cosseno/L2). Combinar:
       $$\text{RRF Score}(d) = \sum_{m \in M} \frac{1}{k + \text{rank}_m(d)}$$
       onde $M = \{\text{FTS5 (BM25)}, \text{Vetorial (k-NN)}, \text{Grafo (CTE Neighbors)}\}$. Isso elimina calibragem manual de pesos e melhora drasticamente a precisão de recuperação.
  2. **Markdown Git-versionado como Fonte da Verdade Soberana:**
     - O banco de dados (SQLite/PostgreSQL) é apenas um cache derivado e descartável (*disposable index*). Se o banco for apagado, o comando `mem index` reconstrói 100% dos dados a partir dos arquivos `.md`.
  3. **Arestas Tipadas (*Typed Edges*):**
     - Permitir que as arestas do grafo tenham semântica rica: `links_to`, `implements`, `depends_on`, `contradicts`, `fixes`, permitindo queries lógicas sofisticadas.
  4. **Padrão *Compile-not-Retrieve* (Karpathy LLM Wiki Pattern):**
     - Em vez de apenas buscar pedaços de texto soltos, os agentes sintetizam e atualizam páginas de notas atômicas consolidadas com conexões explícitas.
  5. **Auto-scoping com Marker File (`.mem.toml` / `.ai-memory.toml`):**
     - Identificação automática do escopo de trabalho e repositório com isolamento seguro em mono-repositórios e múltiplos clientes.

---

### 1.3. [obsidian-skills](https://github.com/kepano/obsidian-skills) (`kepano/obsidian-skills`)
* **Autor:** Steph Ango (@kepano, CEO do Obsidian)
* **O que é:** Coleção oficial de habilidades para agentes operarem vaults do Obsidian de acordo com as especificações abertas de Markdown e JSON Canvas.
* **Pontos de inspiração para o My-Memory:**
  1. **Obsidian Flavored Markdown Parser:**
     - Suporte a YAML frontmatter (`tags`, `aliases`), âncoras/cabeçalhos (`[[Nota#Seção]]`), block references (`[[Nota#^id]]`) e transclusões (`![[Nota]]`).
  2. **Especificação JSON Canvas 1.0 (`.canvas`):**
     - Padrão aberto oficial para mapas visuais espaciais, permitindo que subgrafos recuperados na busca ou via MCP sejam abertos visualmente dentro do Obsidian.
  3. **Obsidian CLI & Deep Links:**
     - Integração com `obsidian://open?file=...` para focar diretamente na nota consultada pelo agente.

---

## 🚀 2. Matriz de Refinamento Arquitetural para o My-Memory

| Capacidade | Estado Inicial do My-Memory | Refinamento Inspirado | Projeto Referência |
|---|---|---|---|
| **Recuperação** | k-NN vetorial puro + CTE de vizinhos | **Busca Híbrida RRF** (FTS5 + k-NN + Grafo) | `akitaonrails/ai-memory` |
| **Integridade de Dados** | SQLite com tabelas relacionais | **Markdown como Fonte de Verdade** (Banco como projeção reconstruível) | `akitaonrails/ai-memory` |
| **Topologia de Grafo** | Apenas arestas `links_to` | **Arestas Tipadas** (`implements`, `depends_on`) + **Status Epistêmico** (`EXTRACTED` vs `INFERRED`) | `ai-memory` & `graphify` |
| **Identificação de Hubs** | Busca local direta | **God Nodes / Centralidade** (identificar conceitos mais densos) | `graphify` |
| **Cache de Indexação** | Reindexa todos os arquivos | **Incremental Hash (SHA-256)** (processa apenas o que mudou) | `graphify` |
| **Visualização do Grafo** | Apenas texto via terminal / MCP | **Exportação para JSON Canvas (`.canvas`)** e visualizador interativo | `kepano/obsidian-skills` & `graphify` |
| **Parsing de Notas** | Regex simples para `[[wikilinks]]` | **Obsidian Flavored Markdown** (YAML frontmatter, âncoras, aliases) | `kepano/obsidian-skills` |
| **Multi-Repositório** | Detecção de Git Origin | **Auto-scoping com Marker File** (`.mem.toml` / Git slug) | `ai-memory` |
