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

### 1.4. [CodeGraph](https://github.com/colbymchenry/codegraph) (`colbymchenry/codegraph`)
* **Autor:** Colby McHenry (@colbymchenry)
* **O que é:** Grafo de conhecimento pré-indexado local-first com kernel em Rust e armazenamento SQLite, projetado especificamente para agentes de IA (Claude Code, Cursor, Antigravity, Codex, Gemini) com sincronização em tempo real via watcher e eliminação completa de explorações cegas (*zero file reads* em benchmarks).
* **Pontos de inspiração para o My-Memory:**
  1. **Contexto Cirúrgico (*Surgical Context*):**
     - Entrega caminhos de dependência e trechos exatos de notas e código em uma única chamada MCP, impedindo que o agente desperdice tokens e turnos re-derivando estrutura por varredura manual.
  2. **Sincronização Reativa com Staleness Banners:**
     - File watcher com debouncing inteligente para agrupar rajadas de salvamento contínuo, e injeção de avisos explícitos (`⚠️ Staleness Banner`) nas ferramentas MCP para alertar agentes caso um arquivo ainda esteja em processamento na fila.
  3. **Análise de Impacto e Raio de Destruição (*Blast Radius / Impact Analysis*):**
     - Rastreamento em cascata de callers, dependentes e conceitos correlatos antes de aplicar alterações ou revogar decisões de arquitetura.
  4. **Visualização Espacial de 3 Colunas (`In-links | Nota | Out-links`):**
     - Layout ergonômico em três colunas espelhadas, alinhando chamadores à esquerda, corpo da nota no centro e referências de saída à direita.
   5. **Auto-Wiring de Ferramentas de IA (`codegraph install`):**
      - Descoberta automática de configurações de IDEs locais (Cursor, Claude Code, Antigravity) para registrar servidores MCP sem atrito manual.

---

### 1.5. [OpenViking](https://github.com/volcengine/OpenViking) (`volcengine/OpenViking`)
* **Autor / Organização:** Volcengine / ByteDance
* **O que é:** Context Database de código aberto para agentes de IA que unifica memória de longo prazo, RAG de conhecimento e skills operacionais sob uma hierarquia virtual de arquivos com carregamento progressivo de contexto.
* **Pontos de inspiração para o My-Memory:**
  1. **Carregamento Progressivo em Camadas (Context Tiers L0 / L1 / L2):**
     - **L0 (Micro-Abstract):** Resumo sintético de 1 frase para triagem de relevância instantânea com consumo mínimo de tokens.
     - **L1 (Overview):** Visão geral estrutural, decisões arquiteturais e sumário do nó para planejamento.
     - **L2 (Full Details):** O conteúdo integral do documento, lido cirurgicamente apenas sob demanda estrita.
  2. **Tripartição de Contexto do Agente (`Resources` vs `Memories` vs `Skills`):**
     - Diferenciação clara entre documentação técnica/código (`resources`), hábitos e preferências de arquitetura (`memories`) e rotinas operacionais executáveis (`skills`).
  3. **Busca Guiada por Comunidades e Domínios (Hierarchical Retrieval):**
     - Triagem preliminar de domínio/cluster de conhecimento antes da recuperação granular de trechos.

### 1.6. [Atlas](https://github.com/sergio-sisternes-epam/atlas) (`sergio-sisternes-epam/atlas`)
* **Autor / Organização:** Sergio Sisternes (@sergio-sisternes-epam, EPAM) / Open Knowledge Format (OKF)
* **O que é:** Knowledge Substrate durável e distribuído para agentes LLM e skills APM baseado na especificação aberta **OKF v0.2** (*Open Knowledge Format*), utilizando Git submodules/branches, gates determinísticos de compilação/validação (`atlas compile` / `validate`), protocolo de endereçamento federado `atlas://` e ciclo de vida higiênico de memórias via `staging/` e promoção.
* **Pontos de inspiração para o My-Memory:**
  1. **Governança Estrita de Schemas e Gates de Qualidade (OKF & Deterministic Gates):**
     - O `my-memory` deve dispor de validação formal de schema (`mem lint` / `mem doctor --strict`) para garantir que notas, ADRs e especificações criadas por agentes de IA obedeçam à taxonomia exigida (L0/L1/L2, category, summary, tags, aliases) antes do commit.
  2. **Protocolo Canônico Federado Cross-Repository (`memory://<repo>/<doc>`):**
     - Estabelecer uma notação canônica de URIs para cruzar referências entre repositórios e vaults distintos (`[[memory://central-brain/auth-standard]]` ou `[[memory://repo-b/api-contracts]]`), viabilizando a navegação federada sem quebrar a autonomia de cada repositório local.
  3. **Higiene de Conhecimento e Workflow de Staging (`staging/` -> Promoção):**
     - Proteger o grafo canônico da poluição de notas e rascunhos rasos gerados por agentes, mantendo notas recém-escritas em quarentena/staging até revisão ou promoção (`mem promote`).
  4. **Topologia Multi-Store (Hub Central vs Repositórios Satélites):**
     - Separação deliberada entre um *Vault Central de Conhecimento* (Global Brain com padrões transversais, aprofundamento e regras corporativas) e *Subvaults de Projeto* (Local Brains isolados com código, ADRs locais e especificações cirúrgicas).

---

## 🚀 2. Matriz de Refinamento Arquitetural para o My-Memory

| Capacidade | Estado Inicial do My-Memory | Refinamento Inspirado | Projeto Referência |
|---|---|---|---|
| **Recuperação** | k-NN vetorial puro + CTE de vizinhos | **Busca Híbrida RRF** (FTS5 + k-NN + Grafo) | `akitaonrails/ai-memory` |
| **Integridade de Dados** | SQLite com tabelas relacionais | **Markdown como Fonte de Verdade** (Banco como projeção reconstruível) | `akitaonrails/ai-memory` |
| **Topologia de Grafo** | Apenas arestas `links_to` | **Arestas Tipadas** (`implements`, `depends_on`) + **Status Epistêmico** (`EXTRACTED` vs `INFERRED`) | `ai-memory` & `graphify` |
| **Identificação de Hubs** | Busca local direta | **God Nodes / PageRank** (autoridade estrutural e hubs densos) | `graphify` & ADR-014 |
| **Detecção Modular** | Sem agrupamento macroestrutural | **LPA Ponderado e Modularidade \(Q\)** (identificação de clusters conceituais) | ADR-022 & literatura de redes |
| **Cache de Indexação** | Reindexa todos os arquivos | **Incremental Hash (SHA-256)** (processa apenas o que mudou) | `graphify` |
| **Sincronização Viva** | Indexação manual sob demanda | **File Watcher com Debounce e Staleness Banners** | `colbymchenry/codegraph` & ADR-017 |
| **Visualização do Grafo** | Apenas texto via terminal / MCP | **Exportação JSON Canvas e HTML/SVG Interativo** com modo de clusters | `kepano/obsidian-skills`, `graphify` & `codegraph` |
| **Parsing de Notas** | Regex simples para `[[wikilinks]]` | **Obsidian Flavored Markdown** (YAML frontmatter, âncoras, aliases) | `kepano/obsidian-skills` |
| **Multi-Repositório** | Detecção de Git Origin | **Auto-scoping com Marker File** (`.memory/config.yaml` / Git slug) | `ai-memory` |
| **Transporte MCP** | Apenas stdio local | **Multi-transporte (stdio + HTTP/SSE)** para agentes remotos e locais | ADR-021 |
| **Raio de Destruição** | Sem análise de dependências reversas | **Análise de Impacto Reversa e Risk Scoring** (BFS reversa, severidade, PageRank, clusters) | `colbymchenry/codegraph` & ADR-023 |
| **Carregamento em Camadas** | Recuperação de texto plano integral | **Progressive Context Loading (L0/L1/L2)** com Micro-Abstracts e Triptych | `volcengine/OpenViking` & `codegraph` |
| **Taxonomia de Conhecimento** | Notas tratadas de forma homogênea | **Tripartição `Resource` vs `Memory` vs `Skill`** no frontmatter | `volcengine/OpenViking` |
| **Governança de Schema** | Validação informal | **Gates Determinísticos e Validação OKF** (`mem doctor --strict` / `mem lint`) | `sergio-sisternes-epam/atlas` |
| **Federação Multi-Vault** | Isolamento por repositório local | **Protocolo Canônico Federado (`memory://`)** e Topologia Hub & Spoke | `sergio-sisternes-epam/atlas` & OKF v0.2 |
| **Higiene de Memórias** | Escrita direta no cofre | **Ciclo de Vida Staging → Promote** para notas geradas por agentes | `sergio-sisternes-epam/atlas` |

← [[README]] · [[COMO_FUNCIONA]]

