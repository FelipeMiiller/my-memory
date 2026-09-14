---
title: "Guia de Integração e Uso do My-Memory para Agentes de IA"
category: memory
summary: "Guia definitivo para desenvolvedores e Agentes de IA sobre integração, configuração de ambiente e consumo das ferramentas MCP."
tags: [agents, ai, mcp, integration, memory]
---

# Guia de Integração e Uso do My-Memory para Agentes de IA

Este documento é o guia definitivo para desenvolvedores e **Agentes de IA** (Antigravity, Cursor, Claude Code, GitHub Copilot, Windsurf, Roo Code, etc.) sobre como instalar, inicializar, configurar e utilizar o **My-Memory** como motor de memória semântica e relacional de contexto em qualquer repositório.

---

## 📌 Sumário
1. [Como o Agente sabe que deve usar o My-Memory?](#1-como-o-agente-sabe-que-deve-usar-o-my-memory)
2. [Trecho Pronto para Inserir no `AGENTS.md` ou Regras](#2-trecho-pronto-para-inserir-no-agentsmd-ou-regras)
3. [Ciclo de Inicialização do Vault (`.memory/`)](#3-ciclo-de-inicialização-do-vault-memory)
4. [Configuração de Ambiente e Modelos de IA (`.env`)](#4-configuração-de-ambiente-e-modelos-de-ia-env)
5. [Configuração Declarativa de Escopo (`config.yaml`)](#5-configuração-declarativa-de-escopo-configyaml)
6. [Manual de Comandos do CLI (`mem <comando>`)](#6-manual-de-comandos-do-cli-mem-comando)
7. [Ferramentas do Servidor MCP para IAs](#7-ferramentas-do-servidor-mcp-para-ias)

---

## 1. Como o Agente sabe que deve usar o My-Memory?

Agentes autônomos de IA **não adivinham** a existência de ferramentas em um repositório sem um ponto de entrada textual. No ecossistema moderno de desenvolvimento assistido por IA, todo agente lê arquivos de inicialização do projeto antes do primeiro prompt:

- **Antigravity / Claude Code / Agentes de Terminal**: Lêm o [`AGENTS.md`](../AGENTS.md) ou `CLAUDE.md` na raiz.
- **Cursor IDE**: Lê `.cursorrules` ou arquivos na pasta `.cursor/rules/`.
- **GitHub Copilot**: Lê `.github/copilot-instructions.md`.
- **Windsurf**: Lê `.windsurfrules`.

Portanto, para que o agente saiba que deve checar se a pasta `.memory/` existe e inicializá-la, basta adicionar a diretriz descrita na próxima seção no arquivo de regras do seu repositório.

---

## 2. Trecho Pronto para Inserir no `AGENTS.md` ou Regras

Copie e cole o trecho abaixo no seu [`AGENTS.md`](../AGENTS.md), `.cursorrules` ou `.github/copilot-instructions.md`:

````markdown
<!-- COPIAR A PARTIR DAQUI -->
## 🧠 Memória do Repositório (My-Memory)

Este repositório utiliza o **My-Memory** como motor de contexto semântico, relacional e busca híbrida.

### 1. Verificação de Inicialização Obrigatória
Ao iniciar qualquer tarefa neste repositório:
1. Verifique se o diretório `.memory/` ou o arquivo `.memory/config.yaml` existe.
2. Se **NÃO existir**:
   - Execute no terminal: `mem init` (ou `go run ./cmd/mem init`).
   - O comando gerará de forma segura:
     - `.memory/config.yaml`: Escopo de pastas e banco de dados.
     - `.memory/.gitignore`: Protege credenciais e bancos locais.
     - `.memory/.env.example`: Modelo de variáveis de ambiente.
   - Em seguida, execute `mem index` para alimentar a base.

### 2. Diretrizes de Uso da Memória pelo Agente
1. **Consulte antes de modificar**: Antes de refatorar código ou criar novas funcionalidades, execute `memory_search` (ou `mem search "<tema>"`) para verificar decisões de arquitetura e notas existentes.
2. **Avalie o Raio de Impacto (Blast Radius)**: Ao modificar ou renomear arquivos e conceitos críticos, use `memory_get_impact` (ou `mem impact <id>`) para analisar dependentes diretos e reversos.
3. **Persista Conhecimento Atômico**: Após tomar decisões ou implementar novas features, utilize `memory_write_note` para salvar a síntese no vault com `[[wikilinks]]`.
<!-- ATÉ AQUI -->
````

---

## 3. Ciclo de Inicialização do Vault (`.memory/`)

O comando de inicialização padrão é:

```bash
mem init
```

### O que o `mem init` faz:
1. Cria a pasta `.memory/` na raiz do projeto.
2. Cria `.memory/config.yaml` já com escopo de pastas (`docs/**/*.md`, `specs/**/*.md`, `README.md`) e apontamento para `.memory/memory.db`.
3. Cria `.memory/.gitignore` para garantir que arquivos confidenciais não sejam enviados ao Git.
4. Cria `.memory/.env.example` com modelos de conexão para PostgreSQL e IA de indexação.
5. **Garantia de Segurança:** Se `.memory/config.yaml` já existir, o comando aborta sem sobrescrever nada, a menos que você passe `--force`.

### Flags Opcionais do `mem init`:
```bash
# Inicializar em pasta específica
mem init ./caminho/do/vault

# Gerar integração automática com MCP para VS Code Copilot e Cursor
mem init --all

# Forçar sobrescrita de configurações existentes
mem init --force
```

---

## 4. Configuração de Ambiente e Modelos de IA (`.env`)

Para ativar configurações personalizadas de banco ou modelos de IA, copie o arquivo de exemplo:

```bash
cp .memory/.env.example .memory/.env
```

Edite o arquivo `.memory/.env`:

```env
# ==============================================================================
# My-Memory - Configurações de Ambiente (.memory/.env)
# ==============================================================================

# --- 1. Persistência PostgreSQL com pgvector (Opcional) ---
# Se informada, o My-Memory conecta automaticamente ao PostgreSQL em todos os comandos:
MY_MEMORY_PG_URL=postgres://postgres:postgres@localhost:5432/my_memory?sslmode=disable
# MY_MEMORY_REPO=FelipeMiiller/my-memory

# --- 2. IA de Indexação / Modelo de Embeddings ---
# Substitui o modelo padrão embutido (nomic-embed-text na porta 11434 local).
# Você pode utilizar qualquer modelo ou endpoint do Ollama (local ou remoto):
MY_MEMORY_EMBED_PROVIDER=ollama
MY_MEMORY_EMBED_URL=http://localhost:11434
MY_MEMORY_EMBED_MODEL=nomic-embed-text
MY_MEMORY_EMBED_DIM=768
```

### 💡 Exemplos de Modelos Suportados via Ollama:
| Modelo | Dimensão | Configuração no `.env` |
| :--- | :---: | :--- |
| **Nomic Embed Text** (Padrão) | 768 | `MY_MEMORY_EMBED_MODEL=nomic-embed-text`<br>`MY_MEMORY_EMBED_DIM=768` |
| **BAAI BGE-M3** (Multilíngue denso/esparso) | 1024 | `MY_MEMORY_EMBED_MODEL=bge-m3`<br>`MY_MEMORY_EMBED_DIM=1024` |
| **Mixedbread mxbai-embed-large** | 1024 | `MY_MEMORY_EMBED_MODEL=mxbai-embed-large`<br>`MY_MEMORY_EMBED_DIM=1024` |
| **All-MiniLM-L6-v2** (Super leve) | 384 | `MY_MEMORY_EMBED_MODEL=all-minilm`<br>`MY_MEMORY_EMBED_DIM=384` |

> [!NOTE]
> O My-Memory também aceita aliases universais no `.env`: `EMBEDDING_URL`, `OLLAMA_HOST`, `EMBEDDING_MODEL` e `EMBEDDING_DIM`.

---

## 5. Configuração Declarativa de Escopo (`config.yaml`)

O arquivo `.memory/config.yaml` define as regras de varredura e persistência:

```yaml
version: 1
repository: "FelipeMiiller/my-memory"
vault_name: "Knowledge Vault"

# Pastas e arquivos a serem indexados (Globs)
include:
  - "docs/**/*.md"      # Documentações em docs/
  - "specs/**/*.md"     # Especificações em specs/
  - ".specs/**/*.md"    # Especificações técnicas em .specs/
  - "README.md"         # Arquivo raiz

# Pastas ignoradas
exclude:
  - ".git/**"
  - "node_modules/**"
  - "vendor/**"
  - ".obsidian/**"
  - ".trash/**"
  - ".memory/**"

storage:
  engine: "sqlite"                 # "sqlite" ou "postgres"
  sqlite_path: ".memory/memory.db" # Banco local
  postgres_url: "postgres://postgres:postgres@localhost:5432/my_memory?sslmode=disable"

embedding:
  provider: "ollama"
  model: "nomic-embed-text"
  url: "http://localhost:11434"
  dimension: 768
```

---

## 6. Manual de Comandos do CLI (`mem <comando>`)

| Comando | Descrição | Exemplo de Uso |
| :--- | :--- | :--- |
| `mem init` | Inicializa a pasta `.memory/` com arquivos padrão protegidos. | `mem init` |
| `mem index` | Varre as notas, calcula hash SHA-256 e gera embeddings com cache incremental. | `mem index` *(ou `mem index --force`)* |
| `mem status` | Audita instantaneamente a sincronização entre arquivos em disco e a base indexada. | `mem status` *(ou `mem status --json`)* |
| `mem search` | Busca híbrida (BM25 + vetores + grafo) com Reciprocal Rank Fusion (RRF). | `mem search "como funciona o cache SHA-256?"` |
| `mem inspect` | Inspetor cirúrgico em 3 colunas: in-links (dependências), nó central e out-links. | `mem inspect "docs/ARCHITECTURE.md"` |
| `mem path` | Descoberta de rotas e menor caminho ponderado por custos epistêmicos entre dois nós. | `mem path "auth" "redis"` |
| `mem impact` | Analisa o raio de destruição (*Blast Radius*) e dependentes reversos de um nó. | `mem impact "internal/db/graph.go" --depth 2` |
| `mem doctor` | Audita o grafo em busca de dead links, notas órfãs e self-loops. | `mem doctor` *(ou `mem doctor --fix`)* |
| `mem hubs` | Lista nós centrais por grau estrutural ou PageRank ponderado. | `mem hubs --algorithm pagerank --top 10` |
| `mem clusters` | Detecta comunidades conceituais e módulos temáticos via algoritmo LPA. | `mem clusters --json` |
| `mem insights` | Descobre conexões conceituais inesperadas (alta similaridade sem links diretos). | `mem insights --limit 5` |
| `mem graph view` | Gera e abre no navegador o visualizador interativo em HTML/SVG do grafo. | `mem graph view --open` |
| `mem note create` | Cria uma nova nota atômica com frontmatter padronizado e indexação instantânea. | `mem note create concepts/cache.md --title "Cache"` |
| `mem note append` | Anexa uma seção estruturada a uma nota existente com sincronização atômica. | `mem note append concepts/cache.md --header "Invalidação"` |
| `mem compile` | Síntese de fragmentos recuperados em uma nota consolidada (*Compile-not-Retrieve*). | `mem compile --topic "autenticação" --out auth.md` |
| `mem watch` | Monitora continuamente o sistema de arquivos e reindexa notas em tempo real. | `mem watch --debounce 500` |
| `mem hook install` | Instala Git pre-commit hook para indexação automática prévia ao commit. | `mem hook install` |
| `mem pack` | Empacota subgrafo conexo centrado em nota raiz com controle rígido de tokens. | `mem pack "docs/auth.md" --max-tokens 3000` |
| `mem open` | Abre nota diretamente no Obsidian ou VS Code com cursor opcional na linha. | `mem open "docs/auth.md" --app vscode --line 42` |
| `mem mcp` | Inicia o servidor MCP via stdio (para IDEs) ou HTTP/SSE. | `mem mcp --port 38400` |

---

## 7. Ferramentas do Servidor MCP para IAs

Quando o `my-memory` roda como servidor MCP (`mem mcp`), o Agente de IA tem acesso direto às seguintes ferramentas:

| Ferramenta MCP | Quando o Agente deve chamar | Parâmetros Principais |
| :--- | :--- | :--- |
| `memory_search` | Sempre que precisar recuperar contexto semântico, arquitetural ou regras. | `query` (string), `limit` (int), `mode` ("hybrid" \| "vector" \| "fts") |
| `memory_get_neighbors` | Para inspecionar dependências e conexões diretas de um arquivo ou conceito. | `node_id` (string), `depth` (int) |
| `memory_get_impact` | Antes de refatorar, excluir ou renomear nós para avaliar o raio de destruição. | `node_id` (string), `depth` (int) |
| `memory_get_clusters` | Para entender a divisão macro de módulos e domínios do repositório. | `min_size` (int) |
| `memory_get_hubs` | Para identificar as notas e arquivos mais centrais e influentes do sistema. | `top` (int), `algorithm` ("pagerank" \| "degree") |
| `memory_doctor` | Para verificar a integridade estrutural do grafo e detectar links quebrados. | `fix` (bool) |
| `memory_write_note` | Para criar notas conceituais com backlinks após implementar features. | `path`, `title`, `content`, `tags` |
| `memory_append_section`| Para adicionar exemplos, testes ou logs a uma nota existente. | `path`, `header`, `content` |
| `memory_compile_note` | Para consolidar tópicos dispersos em uma síntese única com fontes. | `topic`, `out_path`, `limit` |
| `memory_visualize_graph`| Para renderizar o grafo de conhecimento em HTML standalone interativo. | `root_node`, `depth` |
| `memory_inspect_node` | Para inspeção cirúrgica em 3 colunas (in-links, nó e out-links) com zero file reads. | `node_id` (string), `max_content_length` (int) |
| `memory_find_path` | Para rastrear a cadeia de dependências ou menor caminho epistêmico entre dois nós. | `source` (string), `target` (string), `max_depth` (int), `directed` (bool), `mode` ("epistemic" \| "hops") |
| `memory_pack_context` | Para extrair e empacotar um subgrafo conexo com limite rígido de tokens em prompt único. | `root_node` (string), `max_depth` (int), `max_tokens` (int), `direction` ("both" \| "outbound" \| "inbound") |
| `memory_open_node` | Para gerar links acionáveis (`obsidian://`, `vscode://`) ou solicitar abertura de nota no editor. | `node_id` (string), `app` ("obsidian" \| "vscode" \| "system"), `line` (int), `action` ("links_only" \| "open") |

---

## 8. Detecção de Desatualização & Staleness Banners

Para evitar alucinações decorrentes de notas editadas no disco sem reindexação (`Context Drift`), o My-Memory inclui um detector ativo com cache em memória (TTL 3s).

### Comportamento do Banner MCP
Quando o agente invoca ferramentas de consulta (`memory_search`, `memory_pack_context`, `memory_find_path`, `memory_inspect_node`, `memory_get_impact`, etc.) e o índice está desatualizado em relação aos arquivos em disco, a resposta inclui automaticamente um aviso não-bloqueante no topo:


```markdown
> ⚠️ **AVISO: Memória Desatualizada (Stale Data)**
> O índice local está defasado em relação aos arquivos no disco (3 modificado(s)/novo(s), 0 removido(s)).
> Execute `mem index` no terminal para sincronizar o grafo e embeddings.
```

### O que o Agente de IA deve fazer ao ver este aviso?
1. Se a tarefa envolver tomada de decisão de alta precisão (ex: refatoração crítica, checagem de regras de negócio), execute `mem index` ou recomende ao usuário executá-lo.
2. Não descarte a resposta: o conteúdo retornado reflete a versão anteriormente indexada e continua sendo útil como contexto histórico ou estrutural preliminar.

---

## Conclusão

Com essas diretrizes no repositório e o arquivo `.memory/config.yaml` ativo, qualquer Agente de IA opera com **visão total do grafo de conhecimento**, sem alucinar dependências e mantendo a documentação sincronizada em tempo real.
