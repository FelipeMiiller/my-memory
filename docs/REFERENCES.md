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

### 1.7. [Always-on Memory Agent](https://github.com/GoogleCloudPlatform/generative-ai/tree/main/gemini/agents/always-on-memory-agent) (`GoogleCloudPlatform/generative-ai`)
* **Autor / Organização:** Shubhamsaboo / Google Cloud Platform (Gemini Enterprise Agent Platform samples)
* **O que é:** Agente Python sempre ativo que vigia uma pasta `inbox/` para ingestão automática de arquivos (text, images, audio, video, PDFs), consolida periodicamente (default 30 min) e serve um **QueryAgent** que sintetiza respostas com citações explícitas `[Memory N]` via HTTP REST API (`localhost:8888`) + Streamlit dashboard. Usa Google ADK + Gemini 3.1 Flash-Lite e SQLite para persistência.
* **Pontos de inspiração para o My-Memory (atualizado 2026-09-19):**
  1. **Loop de Consolidação Periódica:**
     - Polling de 30 min no GCP Agent é o oposto do que queremos (event_runtime ADR-043 é reativo via outbox + dispatcher). Porém o **padrão** de consolidação agendada — resumir/Sintetizar periodicamente, persistir o resumo com proveniência, e oferecer via query — deve existir como **drift watcher + consolidator** no my-memory. Pode virar ADR-046 Proposed após ADR-044.
  2. **Citation Layer Explícita na Resposta:**
     - O QueryAgent do GCP cita `[Memory 2]`, `[Memory 3]` na resposta. Hoje o my-memory tem `documents`/`chunks` com `source_document` mas a camada de apresentação que cita `[Doc 042]` (ou `[memory://repo/doc]`) na resposta do agente ainda não existe. Pode entrar no ModelProvider (ADR-047 planejado) ou virar ADR-051 explícito.
  3. **HTTP REST Paralelo ao MCP:**
     - O endpoint `GET /query?q=...` do GCP Agent é mais discoverable que MCP pra humanos testando com `curl`. Quando o MCP HTTP (ADR-021) for público, vale oferecer endpoint REST complementar (`mem serve --http` em `:8888`) com `/health`, `/query`, `/ingest` — mantendo MCP como transporte preferido pra agentes.
  4. **Inbox Watcher Multi-formato:**
     - O watcher de `inbox/` com auto-ingest (text, image, audio, video, PDF) é uma UX forte. O `internal/watcher/` do my-memory (ADR-017) já cobre `.md` — estender para `.txt`, `.pdf`, `.png`, `.mp3` no F2 (voz) é caminho natural.
  5. **Streaming Dashboard:**
     - Streamlit em `:8501` mostra ingest/query/delete em tempo real. O viewer Vite (ADR-037) cobre o caso geral; pode-se adicionar um **timeline feed live** (via Server-Sent Events sobre `/events`) que empurra novos envelopes conforme o event_runtime os comita — equivalente funcional sem Streamlit.

---

### 1.8. [Roadmap Autoral F0-F5](file:///./adr/043-envelope-de-eventos-canonico-event-runtime.md) — Referências por Camada

A pesquisa autoral de 2026-09-18 (`pesquisa-infraestrutura-autoral-mymemory.md`) definiu F0-F5 como roadmap pós-v1.4.0 e citou dezenas de projetos externos como inspiração por camada. Esta entry agrega essas referências agrupadas por fase/camada, com breve nota sobre o que cada uma contribuiu. **Para acompanhar o estado de evolução destes projetos, marque esta entry como referência viva — quando algum deles lançar release relevante, vale reler a pesquisa e revisar se o my-memory precisa atualizar contratos.**

#### F0-F1 — Fundação (`mymemoryd`, `event_runtime`, writer atômico)

*Sem referências externas significativas.* ADR-043 explicitamente rejeitou NATS JetStream, Redis Streams, ZeroMQ e CRDT libraries em favor de in-process Go + SQLite WAL (single-node MVP). A decisão de in-process está consolidada em ADR-043 §3.9.

#### F2 — Voz (capture, VAD, wake word, ASR, TTS)

| Projeto | URL | Contribuição |
|---|---|---|
| Wyoming | [`OHF-Voice/wyoming`](https://github.com/OHF-Voice/wyoming) | Framing binário JSON + payload; separação serviço de áudio vs processamento; capabilities negotiation. **Não usar** em rede aberta (sem auth/cripto por design). |
| Home Assistant voice pipelines | [`developers.home-assistant.io/docs/voice/pipelines`](https://developers.home-assistant.io/docs/voice/pipelines/) | Máquina de estados explícita: `start_stage`/`end_stage`, VAD no dispositivo, streaming WebSocket, barge-in, eventos de erro. |
| OpenAI Realtime API | [`developers.openai.com/api/docs/guides/realtime`](https://developers.openai.com/api/docs/guides/realtime) | Full-duplex sessionful, VAD server-side, barge-in, resumption, compressão de contexto. **Referência proprietária** — não incorporada. |
| Gemini Live API | [`ai.google.dev/gemini-api/docs/live-api`](https://ai.google.dev/gemini-api/docs/live-api) | Similar ao OpenAI Realtime. **Referência proprietária** — não incorporada. |
| openWakeWord | [`dscripka/openWakeWord`](https://github.com/dscripka/openWakeWord) | PCM mono 16 kHz, scores por frame, threshold calibrável, segundo verificador. Cuidado: modelos pré-treinados CC BY-NC-SA 4.0. |
| Porcupine | [`Picovoice/porcupine`](https://github.com/Picovoice/porcupine) | Engine multiplataforma leve, registry de modelos, sensibilidade calibrável. **Exige AccessKey** — não é Apache. |
| Whisper.cpp | [`ggml-org/whisper.cpp`](https://github.com/ggml-org/whisper.cpp) | Backend local C/C++/GGML, quantização CPU/GPU, multilíngue. **ADR-046 planejado** (F2) avalia como backend primário. |
| faster-whisper | [`SYSTRAN/faster-whisper`](https://github.com/SYSTRAN/faster-whisper) | CTranslate2 worker, INT8/FP16, batching — alta qualidade mas sem streaming nativo. |
| Vosk | [`alphacep/vosk-api`](https://github.com/alphacep/vosk-api) | Streaming stateful, vocabulário customizável, bindings Go. |
| sherpa-onnx | [`k2-fsa/sherpa-onnx`](https://github.com/k2-fsa/sherpa-onnx) | ASR online + endpointing + WebSocket, encoder/decoder/joiner. Bom candidato a sidecar Python. |
| Piper | [`OHF-Voice/piper1-gpl`](https://github.com/OHF-Voice/piper1-gpl) | TTS local enxuto, vozes `pt_BR` e `pt_PT`. **GPL-3.0 + licença por voz** — verificar antes de embedder. |
| Kokoro | [`hexgrad/kokoro`](https://github.com/hexgrad/kokoro) | Open-weight 82M params, vozes brasileiras `pf_dora`, `pm_alex`, `pm_santa`. Qualidade depende do G2P. |
| Coqui XTTS-v2 | [`docs.coqui.ai/en/latest/models/xtts.html`](https://docs.coqui.ai/en/latest/models/xtts.html) | Clonagem de voz, streaming manual. **CPML — uso não-comercial** — não embedder sem licença compatível. |

#### F3 — Agente runtime (loop, tools, policy, approval)

| Projeto | URL | Contribuição |
|---|---|---|
| llama.cpp | [`ggml-org/llama.cpp`](https://github.com/ggml-org/llama.cpp) | GGUF, mmap, quantização, streaming, tool calling via template/parser. **ADR-047 ModelProvider** planejado suporta. |
| Ollama | [`ollama/ollama`](https://github.com/ollama/ollama) | Lifecycle de modelos, API local, tools, JSON Schema. **Não usar como dependência obrigatória** (research autoral §6 rejeita explicitamente). |
| vLLM | [`vllm-project/vllm`](https://github.com/vllm-project/vllm) | PagedAttention, continuous batching, prefix cache, structured outputs. Foco GPU/Linux. |
| MLX-LM | [`ml-explore/mlx-lm`](https://github.com/ml-explore/mlx-lm) | Memória unificada Apple Silicon. Structured output precisa validação por versão. |
| LangGraph | [`langchain-ai/langgraph`](https://github.com/langchain-ai/langgraph) | StateGraph, nodes, edges, reducers, checkpointers, `interrupt/resume`. **Pattern forte** de checkpoint antes de approval. |
| PydanticAI | [`pydantic/pydantic-ai`](https://github.com/pydantic/pydantic-ai) | Agentes tipados, dependencies, output validation, deferred tools. |
| smolagents | [`huggingface/smolagents`](https://github.com/huggingface/smolagents) | ReAct, ToolCallingAgent, CodeAgent. **Cuidado**: execução de Python gerado não é sandbox. |
| pi-mono | [`badlogic/pi-mono`](https://github.com/badlogic/pi-mono) | Eventos de execução serializáveis, tool hooks, RPC JSONL — inspirador pro CLI/viewer do my-memory. |

#### Memória de Longo Prazo (referências conceituais)

| Projeto | URL | Contribuição |
|---|---|---|
| A-MEM | [`agiresearch/a-mem`](https://github.com/agiresearch/a-mem) | Notas atômicas + keywords + tags + descrições contextuais + embeddings + links candidatos. |
| Graphiti | [`getzep/graphiti`](https://github.com/getzep/graphiti) | Grafos temporais com `event_time`/`recorded_at`, validade, invalidação não destrutiva, comunidades. |
| Hindsight | [`vectorize-io/hindsight`](https://github.com/vectorize-io/hindsight) | Separa `retain`/`recall`/`reflect`, vetor + BM25 + grafo temporal, observações com citações e `proof_count`. |
| Cognee | [`topoteretes/cognee`](https://github.com/topoteretes/cognee) | DataPoints, sessões, `remember`/`recall`/`improve`, datasets, ontologias, permissões. |
| LlamaIndex | [`run-llama/llama_index`](https://github.com/run-llama/llama_index) | Memory blocks, flush por orçamento, prioridades. |

#### Barramento / Event Streaming (deferred F5)

| Projeto | URL | Contribuição |
|---|---|---|
| NATS JetStream | [`docs.nats.io/concepts/jetstream`](https://docs.nats.io/concepts/jetstream) | Subjects, queue groups, ACK, replay, redelivery, pull consumers. **ADR-043 §3.9** rejeita pra MVP. |
| ZeroMQ | [`zeromq.org/socket-api/`](https://zeromq.org/socket-api/) | IPC brokerless, `inproc`/IPC/TCP, pub-sub, pipeline. **Não fornece cursor/ACK** — teria que construir. |
| Redis Streams | [`redis.io/docs/latest/develop/data-types/streams/`](https://redis.io/docs/latest/develop/data-types/streams/) | IDs, grupos, PEL, `XACK`, `XAUTOCLAIM`. **Cuidado**: licença SSPL desde Redis 7.4. |

#### Threat Model e Segurança (ADR-050 transversal)

| Projeto | URL | Contribuição |
|---|---|---|
| OWASP LLM Top 10 | [`genai.owasp.org/llm-top-10/`](https://genai.owasp.org/llm-top-10/) | LLM01-LLM10 — prompt injection, info disclosure, supply chain, data poisoning, output handling, excessive agency, system prompt leakage, vector weaknesses, misinformation, unbounded consumption. **ADR-050** mapeia esses vetores pra controles no mymemoryd. |
| NVIDIA garak | [`NVIDIA/garak`](https://github.com/NVIDIA/garak) | Scanner externo de LLM vulnerability (probes, detectors, relatórios JSONL). **Não incorporar como runtime** — usar em staging/F5. |

#### Viewer / Canvas / Protocolo

| Projeto | URL | Contribuição |
|---|---|---|
| Model Context Protocol | [`modelcontextprotocol.io/specification`](https://modelcontextprotocol.io/specification) | Contratos de interoperabilidade: resources, tools, prompts. **ADR-006, ADR-021, ADR-043** já usam; **ADR-048 planejado** (MCP server autoral). |
| BlockSuite | [`toeverything/blocksuite`](https://github.com/toeverything/blocksuite) | Blocos ricos, canvas edgeless, Yjs/CRDT, snapshots. Avaliar licenças por package. |
| tldraw | [`tldraw/tldraw`](https://github.com/tldraw/tldraw) | Shapes tipados, store reativo, snapshots, migrações. Licença source-available — incorporar requer decisão comercial. |
| Obsidian Vault API | [`docs.obsidian.md/Plugins/Vault`](https://docs.obsidian.md/Plugins/Vault) | Edição atômica, MetadataCache, backlinks. **ADR-008, ADR-037** usam. |

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
| **Consolidação Periódica** | `mem index` sob demanda | **Drift Watcher + Consolidator Agendado** (resumos periódicos com proveniência, persistidos como eventos) | `GoogleCloudPlatform/generative-ai` (Always-on Memory Agent) |
| **Citation Layer** | Sem citação explícita na resposta | **Citação `[Doc 042]` / `[memory://repo/doc]` na resposta do agente** | `GoogleCloudPlatform/generative-ai` (QueryAgent cita `[Memory N]`) |
| **HTTP REST Paralelo** | MCP stdio (ADR-021) | **`mem serve --http :8888` complementar** com `/health`, `/query`, `/ingest` | `GoogleCloudPlatform/generative-ai` (REST :8888 + Streamlit) |
| **Inbox Watcher Multi-formato** | Só `.md` via `internal/watcher/` | **Watcher estendido para `.txt`, `.pdf`, `.png`, `.mp3`** (F2 voz) | `GoogleCloudPlatform/generative-ai` & ADR-017 |
| **Event Bus/Outbox (F0)** | Sem barramento interno | **Envelope canônico + outbox transacional + replay nativo** (ADR-043) | `nats-io/nats-server`, `zeromq`, `redis/redis` (rejeitados p/ MVP — ver ADR-043 §3.9) |
| **Threat Model LLM (F5 transversal)** | Sem threat model | **Mapeamento OWASP LLM01-LLM10 → controles no mymemoryd** (ADR-050) | `OWASP LLM Top 10 2026` + `NVIDIA/garak` (scanner externo) |

← [[README]] · [[COMO_FUNCIONA]]

