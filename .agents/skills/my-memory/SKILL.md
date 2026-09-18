---
name: my-memory
description: Skill canônica de uso do My-Memory — ensina a configurar, inicializar, indexar, buscar, analisar grafo, integrar com IA via MCP e criar documentos no vault Obsidian Flavored Markdown. Aplique sempre que o usuário pedir para instalar o mem, rodar mem init/mem index/mem search, configurar Postgres, abrir o visualizador de grafo, criar nota dentro do vault my-memory, ou quando um agente de IA precisar entender o estado cognitivo do repositório.
---

# My-Memory Skill — Como Configurar, Usar e Criar Documentos

> **Esta é a skill canônica de uso do My-Memory.** Qualquer agente (humano ou IA) que precise instalar, configurar, usar ou documentar algo neste projeto deve seguir este guia. Para regras estritas de formatação de `.md` (wikilinks, tags, frontmatter), consulte a skill [`obsidian-markdown`](.agents/skills/obsidian-markdown/SKILL.md) — ela é a referência canônica de markup.

> [!important]
> **Onde estou?** Você está no **repositório raiz do My-Memory** (`C:\repository\my-memory`). A skill foi escrita assumindo Windows/PowerShell; ajustes para Linux/macOS estão sinalizados com `⚠️ SO`.

> [!note]
> **Mapa rápido:**
> - §1 Visão em 60 segundos
> - §2 Instalação · §3 Inicialização · §4 Configuração · §5 Indexação · §6 Busca · §7 Grafo · §8 Visualização · §9 MCP · §10 Federação · §11 Criar Documentos · §12 Comandos · §13 Manutenção · §14 Troubleshooting · §15 Referências

---

## 1. Visão em 60 segundos

O **My-Memory** é uma ferramenta local em Go que transforma qualquer repositório em um *Repository Brain* — um cérebro autoconsciente que combina:

| Camada | Tecnologia | O que resolve |
|---|---|---|
| **Léxica** | SQLite FTS5 (BM25 nativo) | Busca exata de termos, identificadores, nomes de função |
| **Vetorial** | Ollama + TurboQuant 4-bit (Google DeepMind, ICLR 2026) | Busca semântica com 87 % de economia de RAM (388 B/chunk) |
| **Grafo** | `[[wikilinks]]` + `#tags` extraídos via parser Obsidian | Topologia real de dependências entre notas |
| **Persistência** | SQLite local (default) **ou** PostgreSQL + pgvector (opt-in, ver ADR-040) | Zero-config em repo único; multi-tenant distribuído |
| **Integração IA** | Servidor MCP (stdio + HTTP/SSE) | Cursor, VS Code Copilot, Claude Code, Antigravity |

Quando você instala, o My-Memory entrega três coisas que importam para o usuário:
1. **CLI `mem`** — pesquisa/indexa/audita via terminal
2. **Servidor MCP** — expõe o cérebro para IAs entenderem o repo
3. **Vault `.memory/`** — pequena pasta gitignored que mantém o banco e a config

---

## 2. Instalação

> [!tip]
> Use o instalador one-liner. É o caminho mais curto e oficial.

### 2.1 One-liner oficial (recomendado)

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/FelipeMiiller/my-memory/main/scripts/install.ps1 | iex
```

**Linux/macOS (POSIX shell):**
```bash
curl -fsSL https://raw.githubusercontent.com/FelipeMiiller/my-memory/main/scripts/install.sh | sh
```

### 2.2 Via Go toolchain (1.22+)
```bash
go install github.com/FelipeMiiller/my-memory/cmd/mem@latest
```

### 2.3 Compilação manual
```bash
git clone https://github.com/FelipeMiiller/my-memory.git
cd my-memory
go build -o bin/mem.exe ./cmd/mem   # Windows
# ⚠️ SO Linux/macOS:
# go build -o bin/mem ./cmd/mem
```

### 2.4 Pré-requisitos opcionais
- **Ollama** para embeddings locais (`nomic-embed-text` por default):
  ```bash
  ollama pull nomic-embed-text
  ```
- **PostgreSQL 15+** com extensão `pgvector` (somente se for usar Postgres; default é SQLite).

---

## 3. Inicialização do Vault (`mem init`)

Em qualquer repositório, dentro da pasta raiz:

```bash
mem init
```

### 3.1 O que o comando cria

| Arquivo / pasta | Função | Commitado? |
|---|---|---|
| `.memory/config.yaml` | Escopo declarativo (`include`/`exclude`), paths, embedding config | ✅ sim |
| `.memory/memory.db` | Banco SQLite unificado (FTS5 + sqlite-vec + grafo) | ❌ gitignored |
| `.memory/.gitignore` | Garante que `.env` e `.db` nunca sejam commitados | ✅ |
| `.memory/.env.example` | Modelo de variáveis para Postgres/embeddings | ✅ |
| `.memory/AGENTS.md` | Instruções prontas para mover para `AGENTS.md` da raiz | ✅ opcional |

> [!note]
> **ADR-040 (vigente):** o banco é **auto-scope** para `.memory/memory.db` quando o vault existe. Antes desse ADR, `mem index` sem `--db` criava `memory.db` na CWD — comportamento legado eliminado em `ec1298b`.

### 3.2 Flags úteis
```bash
mem init --force        # sobrescreve vault existente sem perguntar
mem init --repo meu-repo # define slug do repo explicitamente
mem init --all          # também gera `.cursor/mcp.json` e `.vscode/mcp.json`
```

### 3.3 Pós-init
```bash
mem index    # primeira indexação — pode levar minutos para vaults grandes
```

---

## 4. Configuração (`.memory/config.yaml`)

> [!important]
> Após `mem init`, edite `.memory/config.yaml` para definir **quais pastas entram no cérebro**. Sem isso, `mem index` não lê nenhuma nota.

### 4.1 Estrutura típica

```yaml
# Escopo declarativo de indexação
include:
  - "docs/**/*.md"      # documentação canônica
  - ".specs/**/*.md"    # specs técnicas (features + ADRs)
  - "README.md"         # raiz do projeto

exclude:
  - ".git/**"
  - "node_modules/**"
  - "bin/**"
  - ".memory/**"
  - "vendor/**"

# Persistência (ver ADR-040 — SQLite auto-scope é o default)
storage:
  engine: sqlite                    # sqlite | postgres
  sqlite_path: .memory/memory.db    # relativo à raiz do repo
  # postgres_url: postgres://USER:PASSWORD@HOST:5432/DBNAME

# Embeddings (Ollama local por default)
embedding:
  provider: ollama
  base_url: http://localhost:11434  # MY_MEMORY_EMBED_URL
  model: nomic-embed-text          # MY_MEMORY_EMBED_MODEL
  dimension: 768                   # MY_MEMORY_EMBED_DIM
```

### 4.2 Ordem de precedência (do mais prioritário ao menos)

1. **Flag CLI explícita** — `--storage=postgres`, `--postgres <url>`, `--db <path>`
2. **Variáveis de ambiente** — `MY_MEMORY_PG_URL`, `POSTGRES_URL`, `DATABASE_URL`, `MY_MEMORY_EMBED_URL`, etc.
3. **`~/.memory/config.yaml` (global)** — source-of-truth da Federação / Central Vault (ADR-040)
4. **`.memory/config.yaml` (local)** — escopo deste repo
5. **Default implícito** — SQLite local em `.memory/memory.db`

### 4.3 PostgreSQL opt-in (ADR-040)

Para trabalhar com **Central Vault** (multi-repo no Google Drive / OneDrive + Postgres):

```bash
# .memory/.env  (local — opcional)
MY_MEMORY_PG_URL=postgres://USER:PASSWORD@HOST:5432/DBNAME?sslmode=require
```

Ou no global:
```yaml
# ~/.memory/config.yaml
storage:
  engine: postgres
  postgres_url: postgres://USER:PASSWORD@HOST:5432/DBNAME?sslmode=require
```

> [!warning]
> **Segredos:** nunca commite `.env`. O `.gitignore` do vault já bloqueia. Use env vars User-scope no Windows ou export no shell. Em mensagens de erro, URLs passam pelo `sanitizePostgresURL()` (logs ocultam `user:password` → `***@host`).

### 4.4 Escolher embedder alternativo

```env
# .env — alternar embedder sem perder o vault
MY_MEMORY_EMBED_BASE_URL=https://api.openai.com/v1
MY_MEMORY_EMBED_MODEL=text-embedding-3-small
MY_MEMORY_EMBED_DIM=1536
```

| Provider | Modelo | Dim | Quando usar |
|---|---|---|---|
| Ollama (default) | `nomic-embed-text` | 768 | Local-first, sem custo |
| Ollama | `bge-m3` | 1024 | Multilíngue robusto |
| OpenAI | `text-embedding-3-small` | 1536 | Quando Ollama não está disponível |
| ONNX MiniLM (ADR-035) | `all-MiniLM-L6-v2` | 384 | Fallback embutido sem rede |

---

## 5. Indexação (`mem index`)

```bash
mem index                # incremental (cache SHA-256 pula o que não mudou)
mem index --force        # reprocessa tudo do zero
mem index --no-prune     # não remove notas deletadas (apenas para auditoria)
mem index --postgres postgres://USER:PASSWORD@HOST:5432/DBNAME   # usa Postgres
```

### 5.1 Pipeline interno (4 etapas)

```
[ Arquivo Markdown ]
        │
        ▼
[ Poda SkipDir: .git, node_modules, vendor, .memory ]
        │
        ▼
[ ShouldIndex (glob include/exclude) ]
        │
        ▼
[ SHA-256 Check no documents.content_hash ]
        │
   ┌────┴────┐
   ▼         ▼
[Cached]  [Hash diferente / novo]
  ⏩ skip   ⚙️ parser → chunking → embedding → TurboQuant → persiste
```

### 5.2 Cache SHA-256 (ADR-010)

Arquivos inalterados são marcados `⏩ [cached]` e **não disparam chamadas de embedding**. Em vaults de 1000+ notas, isso reduz o reprocessamento em > 90 %.

### 5.3 Pruning automático

> [!tip]
> Não precisa limpar manualmente. Quando você apaga uma nota no disco e roda `mem index`, o My-Memory detecta o órfão e remove nós, chunks vetoriais e entradas FTS5 do banco atomicamente.

### 5.4 Pós-index verificar saúde
```bash
mem doctor           # auditoria: dead links, órfãs, self-refs
mem doctor --fix     # auto-cura (remove dead edges)
```

---

## 6. Busca (`mem search`)

```bash
mem search "como funciona o cache de indexação?"          # híbrida (RRF)
mem search --mode fts "CalculateContentHash"              # só léxica (BM25)
mem search --mode vector "algoritmos de comunidades"       # só semântica
mem search --decay --half-life 14 "mudanças recentes"      # privilegia recentes
mem search --level l0 "config"                            # micro-abstract
mem search --category skill "create-adr"                  # filtra taxonomia
```

### 6.1 Modos de busca

| Flag | Back-end | Quando usar |
|---|---|---|
| `--mode hybrid` (default) | RRF (k=60) combinando BM25 + vetor + grafo | Pesquisa geral do dia-a-dia |
| `--mode fts` | Apenas FTS5 BM25 | Quando sabe o termo exato / nome de função |
| `--mode vector` | k-NN cosine em vetores quantizados (TurboQuant se `-tq`) | Perguntas conceituais |
| `-tq` / `--turboquant` | Liga quantização 4-bit ativa na consulta | Benchmarks + comparação de impacto |

### 6.2 Decaimento temporal (ADR-015)

Em vaults vivos, nota de 3 dias atrás frequentemente importa mais que nota de 2 anos. O fator de decaimento:

$$\text{Fator}(t) = e^{-\lambda \Delta t}, \quad \lambda = \frac{\ln 2}{t_{1/2}}$$

```bash
mem search --decay --half-life 30 --decay-weight 0.3 "ADR recente"
```

### 6.3 Resultado com proveniência

Quando o Central Vault está conectado, cada resultado vem com etiqueta:
```text
[local]    docs/CLI_GUIDE.md — §Mem storage
[central]  ~/KnowledgeVault/go-patterns.md — §Repository Brain
```

---

## 7. Análise Cirúrgica e Inteligência de Grafo

### 7.1 `mem inspect <nota>` — visão em 3 colunas

```bash
mem inspect docs/ARCHITECTURE.md
```

Mostra simultaneamente:
- **Coluna 1 (In-Links):** quem aponta para esta nota
- **Coluna 2 (Nó central):** título, status, métricas estruturais, preview
- **Coluna 3 (Out-Links):** notas referenciadas por aqui

### 7.2 `mem impact <nota>` — raio de destruição

```bash
mem impact internal/db/graph.go --depth 2 --json
```

Antes de refatorar ou deletar uma nota/código, lista **quem quebra** e calcula um **Risk Score 0–100**.

### 7.3 `mem hubs` — nós centrais

```bash
mem hubs --algorithm degree --top 10     # God Nodes (grau de entrada)
mem hubs --algorithm pagerank --top 10   # autoridade estrutural (ADR-014)
```

### 7.4 `mem clusters` — detecção de comunidades

```bash
mem clusters --min-size 2 --json
```

Agrupa notas em clusters via *Weighted Label Propagation Algorithm* (ADR-022) e calcula modularidade de Newman-Girvan ($Q$).

### 7.5 `mem doctor` — auditoria de saúde

```bash
mem doctor           # audita (não altera)
mem doctor --fix     # auto-cura (remove dead edges)
```

Detecta:
- **Dead links** — `[[Nota]]` apontando para nota inexistente
- **Notas órfãs** — sem wikilinks de entrada nem saída
- **Self-references** — nota apontando para si mesma
- **Health Score** — escala 0–100 (combina dead link ratio, orphan ratio e demais métricas; ISSUE-006)

### 7.6 `mem insights` — conexões surpreendentes

```bash
mem insights --limit 5
```

Encontra notas com altíssima similaridade semântica que ainda não têm link entre si — brainstorms para o usuário decidir se a conexão merece ser forjada.

### 7.7 `mem drift` — divergência código↔doc (ADR-031, ADR-039)

```bash
mem drift --since HEAD~5..HEAD
mem drift --strict --json
```

Audita divergência semântica entre código e documentação para evitar "memória podre".

### 7.8 `mem path` — caminho mais curto no grafo

```bash
mem path docs/ARCHITECTURE.md docs/CENTRAL_VAULT.md --max-depth 6
```

Mostra a rota conceitual entre duas notas via `WITH RECURSIVE` (ADR-004).

### 7.9 `mem pack` — empacotamento para prompt de IA

```bash
mem pack docs/ARCHITECTURE.md --depth 2 --max-tokens 4000 --out subgrafo.md
```

Empacota subgrafo respeitando orçamento de tokens para prompt de LLM (ADR-029).

---

## 8. Visualização Interativa (`mem graph`)

```bash
mem graph view --open                              # abre HTML padrão no browser
mem graph view --root "docs/ARCHITECTURE.md" --depth 2 --open   # foca subgrafo
mem export --canvas docs/ARCHITECTURE.md --out meu.canvas         # JSON Canvas (Obsidian)
mem open docs/ARCHITECTURE.md                      # abre no editor preferido
```

O visualizador inclui: simulação de forças (Fruchterman-Reingold), busca por texto, filtro de PageRank, alternância de paleta e exportação SVG.

---

## 9. Integração com IA via MCP (ADR-006)

O My-Memory expõe o cérebro via **Model Context Protocol** (stdio ou HTTP/SSE).

### 9.1 Subir o servidor MCP

```bash
mem mcp                       # stdio (uso local pelo editor)
mem mcp --port 38400          # HTTP/SSE (containers / rede)
mem mcp --http 0.0.0.0:38400  # expose para rede
```

### 9.2 Configurar em IDEs

**Cursor** — `.cursor/mcp.json`:
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

**VS Code Copilot** — `.vscode/mcp.json`:
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

> [!tip]
> `mem init --all` gera esses dois arquivos automaticamente.

### 9.3 Tools MCP expostas

| Tool | Função |
|---|---|
| `memory_search` | Busca híbrida federada (RRF + decaimento opcional) |
| `memory_get` | Lê conteúdo integral de uma nota por ID/caminho |
| `memory_get_impact` | Raio de destruição de um nó |
| `memory_compile_note` | `Compile-not-Retrieve` (ADR-018): sintetiza fontes em uma nota atômica |
| `memory_export_context` | Empacota subgrafo respeitando orçamento de tokens |

### 9.4 Padrão Compile-not-Retrieve (ADR-018)

Em vez de a IA reler dezenas de arquivos toda vez:
1. IA pesquisa via `memory_search` sobre o tema
2. Sintetiza em uma nota atômica definitiva com `[[rel:derived_from:...]]`
3. `memory_compile_note` grava em disco e atualiza o grafo
4. Nas próximas interações, IA lê **apenas a nota compilada** — economia de 90 %+ de tokens de contexto

---

## 10. Federação e Central Vault (ADR-033, ADR-034, ADR-040)

Para workspaces distribuídos (Google Drive / OneDrive) unificando múltiplos repos:

```bash
mem setup                  # assistente global → grava ~/.memory/config.yaml
mem central status         # audita Cofre Central
mem central bootstrap      # cria 11 pastas canônicas + templates + README MOC
mem repos                  # lista repos federados registrados
```

Fluxo:
1. Rode `mem setup` uma vez na máquina — define Cofre Central + engine (sqlite|postgres) + porta MCP
2. Rode `mem central bootstrap` para inicializar o cofre (caso esteja vazio)
3. Em cada repositório, rode `mem init` para federar; depois `mem search` consulta local + central em paralelo via RRF, anotando proveniência (`[local]` vs `[central]`)

---

## 11. Como Criar Documentos no Vault

> [!important]
> **Esta seção é a parte mais importante para quem cria `.md`.** Regras detalhadas de markup vivem na skill [`obsidian-markdown`](.agents/skills/obsidian-markdown/SKILL.md) — esta é só a visão geral.

### 11.1 Anatomia mínima de uma nota

```markdown
---
title: "Nome descritivo da nota"
category: resource        # ou memory | skill (ver ADR-024)
summary: "Resumo em 1-2 frases — usado pelo L0 do mem search"
tags:
  - topico-principal
  - subtopico
status: ativo
---

# Nome da Nota

Conteúdo com [[wikilinks]] para outras notas e #tags inline quando fizer sentido.
```

### 11.2 Taxonomia `category` (ADR-024, ADR-025)

| Valor | Quando usar | Exemplos |
|---|---|---|
| `resource` (default) | Documentação técnica persistente | ADRs, RFCs, arquitetura |
| `memory` | Regras/hábitos/preferências do agente | Princípios de qualidade, convenções |
| `skill` | Instruções operacionais / playbooks | Esta skill, COMO_USAR, runbooks |

Use a taxonomia para que `mem search --category skill ...` recupere apenas instruções operacionais, isolando-as de docs técnicas.

### 11.3 Wikilinks (criam arestas no grafo)

```markdown
Está conectado a [[ADR-040]] e substitui [[ADR-016]].
Veja [[TurboQuant 4-bit]] e [[Reciprocal Rank Fusion]].
Conexão tipada: [[rel:implements:adr-040]].
```

| Sintaxe | Efeito |
|---|---|
| `[[Nota]]` | Aresta `links_to` simples |
| `[[Nota\|Texto]]` | Aresta preservando rótulo customizado |
| `[[rel:depends_on:X]]` | Aresta epistêmica tipada (peso 1.0; ver ADR-011) |
| `[[rel:derived_from:Y]]` | Prova de proveniência (peso 0.8) |
| `[[rel:see_also:Z]]` | Sugestão cruzada (peso 0.5) |

> [!warning]
> **Regra ISSUE-001/009 (ADR-041):** tags inline `#tag` ou de frontmatter que **não casam com nenhum doc** via fuzzy resolve são removidas do grafo pelo `internal/parser/fuzzy.go::ResolveTagConnections`. Tags conceituais como `architecture` permanecem pesquisáveis via FTS mas **não viram nó**. Para ter nó do grafo, crie um stub `docs/concepts/<tag>.md` com `title: <tag>`.

### 11.4 Tags inline (`#tag`)

```markdown
Trabalhamos com #memoria-autoconsciente e #fuzzy-resolver.
```

- Inline `#tag` no corpo e tags no frontmatter viram nós do grafo (se houver casa via stub ou fuzzy resolve).
- Tags não casa viram dead links — corrigidos por ADR-041 (removidos do grafo) e/ou criando stub.

### 11.5 Callouts Obsidian (padrão visual rico)

```markdown
> [!note]
> Observação neutra.

> [!warning]
> Atenção — isso quebra fluxos existentes.

> [!tip]
> Boa prática.

> [!important]
> Source-of-truth — divergências devem ser resolvidas a favor deste callout.
```

### 11.6 Comandos rápidos para criar

```bash
mem note create docs/minha-nota.md --title "Título" --tags tag1,tag2
mem note append docs/nota-existente.md --section "Nova Seção"
mem compile --topic "Federação Postgres" --out docs/concepts/federacao-postgres.md
```

> [!tip]
> Use `mem compile --topic ...` quando quiser que a IA sintetize uma nota-base compilada a partir de várias fontes (ADR-018 — Compile-not-Retrieve).

### 11.7 Validar antes de commitar

```bash
mem doctor    # detecta dead links / órfãs
```

> [!warning]
> **Princípio 7 do AGENTS.md:** antes de fazer push de uma feature, **sempre** atualizar docs, specs e README. Reindexe (`mem index`) e rode `mem drift --since HEAD~1` para garantir sincronização código↔doc.

---

## 12. Tabela de Referência Rápida de Comandos

| Comando | Função | Quando usar |
|---|---|---|
| `mem init [--all]` | Cria `.memory/` com `config.yaml`, `.gitignore`, `AGENTS.md` | Setup inicial do vault |
| `mem setup` | Assistente global → grava `~/.memory/config.yaml` | Configurar Central Vault |
| `mem central <status\|bootstrap>` | Audita / inicializa Cofre Central | Trabalhar com Federação multi-repo |
| `mem repos` | Lista repos federados | Auditoria de catálogo |
| `mem install [--tools]` | Auto-configura MCP em Cursor/VS Code/Claude | Plug-and-play de IDEs |
| `mem index [--force]` | Indexa Markdown com cache SHA-256 + pruning | Após criar/editar notas |
| `mem watch` | File watcher + indexação contínua | Manter cérebro vivo sem `mem index` manual |
| `mem hook <install\|uninstall>` | Git pre-commit hook para reindexar | Garantir cérebro atualizado em cada commit |
| `mem search "<q>"` | Busca híbrida federada (RRF + decaimento) | Pesquisa geral |
| `mem search -tq "<q>"` | Modo TurboQuant 4-bit ativo | Benchmarks de quantização |
| `mem path <de> <para>` | Caminho mais curto no grafo | Explicar "como A se conecta com B" |
| `mem inspect <nota>` | Visão cirúrgica em 3 colunas | Entender uma nota isoladamente |
| `mem impact <nota>` | Raio de destruição + Risk Score | Antes de refatorar/deletar |
| `mem hubs` | Degree/PageRank — nós centrais | Mapear autoridade do repo |
| `mem clusters` | LPA — detecção de comunidades temáticas | Mapear módulos conceituais |
| `mem doctor [--fix]` | Dead links, órfãs, Health Score | Auditoria de saúde |
| `mem status` | Saúde do repo + repo_id + central vault + staleness | Status board geral |
| `mem drift [--since]` | Divergência código↔doc | Sincronização código/doc |
| `mem insights` | Conexões surpreendentes sem link | Brainstorm de novas arestas |
| `mem pack <nota>` | Empacota subgrafo com budget de tokens | Prompt de LLM |
| `mem open <nota>` | Abre no editor (VS Code/Cursor/Obsidian) | Navegação rápida |
| `mem graph view [--open]` | HTML interativo do grafo | Visualizar topologia |
| `mem export --canvas <nota>` | Exporta subgrafo em JSON Canvas (Obsidian) | Visualizar no Obsidian Canvas |
| `mem note <create\|append>` | Cria/anexa seções com metadados + indexação atômica | Criar programaticamente |
| `mem compile --topic <termo>` | Sintetiza fontes em nota consolidada | Compile-not-Retrieve |
| `mem mcp [--port]` | Sobe servidor MCP (stdio/HTTP/SSE) | Integrar com IAs |
| `mem bench` | Benchmark TurboQuant/RRF/SHA-256/parsing | Performance profiling |
| `mem version` | Versão + commit + arquitetura | Sanidade |

---

## 13. Manutenção Contínua

### 13.1 Manter o cérebro vivo
- **`mem watch` em background** — reindexa automaticamente ao salvar (debounce 500 ms)
- **`mem hook install`** — pre-commit hook garante cérebro atualizado antes de cada commit
- **`mem status`** — diagnose rápida (saúde, repo_id, staleness)

### 13.2 Atualizar esta skill (regra de ouro)

> [!important]
> **Toda nova feature CLI/MCP ou nova flag ⇒ atualizar esta skill no mesmo PR.**
> Esta skill é a fonte de verdade para uso do My-Memory. Se você adicionar um subcomando `mem foo` ou uma tool MCP `memory_foo`, **atualize §12 e §9.3 antes de declarar "pronto"**.

### 13.3 Roteiro de manutenção ao adicionar uma feature

1. Implementar + testes
2. Atualizar `.agents/skills/my-memory/SKILL.md` (§12 + § específica)
3. Atualizar `docs/CLI_GUIDE.md` (seção do subcomando)
4. Adicionar ADR em `docs/adr/NNN-...md` (se decisão arquitetural; ver AGENTS.md §3)
5. Atualizar `README.md` (seção Capacidades + Comandos + MCP Tools)
6. Atualizar `.specs/.../tasks.md` (T final do spec atômico)
7. `gofmt -l .` + `go build ./...` + `go test -count=1 ./...`
8. `mem index` + `mem doctor`
9. Commit atômico via Conventional Commits

> Esta lista é o **Princípio 7 do AGENTS.md** destilado para esta skill. Ver `.agents/rules/always-update-docs-and-specs-on-pr.md` para a versão formal.

### 13.4 Quando abrir ADR

Se a feature mexe com:
- Schema de banco (nova coluna em `documents`)
- Algoritmo de busca ou quantização
- Formato de markup (nova sintaxe Obsidian)
- Novo protocolo (MCP/JSON Canvas)

→ **Abra um novo ADR** em `docs/adr/` antes de implementar.

Se é bug trivial/cosmético ou ajuste de copy → corrige direto, sem ADR (AGENTS.md Princípio fix imediato).

---

## 14. Troubleshooting

| Sintoma | Causa provável | Solução |
|---|---|---|
| `mem: command not found` | Binário não está no PATH | Rode `go build -o bin/mem.exe ./cmd/mem` ou re-rode o one-liner |
| `.memory/` no CWD em vez de auto-scope | Versão < `ec1298b` | Atualize a CLI; ADR-004 fix aplica esta versão |
| Embeddings lentos / erro Ollama | Ollama não está rodando | `ollama serve` ou defina `MY_MEMORY_EMBED_BASE_URL` remoto |
| Dead links aparecem em `mem doctor` | Tag genérica órfã (ISSUE-001) | Remova tag ou crie stub `docs/concepts/<tag>.md` (ADR-041) |
| Drift reporta 100 % CRITICAL | Calibração antiga (pré-ADR-039) | Atualize para versão pós-`bc70c5f`; pesos recalibrados |
| Postgres URL vazia em log | `sanitizePostgresURL` não aplicado | Verifique versão — fix em `2361a92` |
| `mem search` não retorna nada | Vault recém-criado sem `mem index` | Rode `mem index` primeiro |
| Wikilink não vira aresta | `[texto](./path.md)` em vez de `[[path]]` | Use SEMPRE `[[wikilink]]` para refs internas |

---

## 15. Referências Cruzadas

### Skills irmãs
- [`obsidian-markdown`](.agents/skills/obsidian-markdown/SKILL.md) — **canônica de formatação `.md`** (frontmatter, wikilinks, tags, callouts, embeds)
- [`create-adr`](.agents/skills/create-adr/SKILL.md) — formato MADR para abrir nova decisão arquitetural
- [`tlc-spec-driven`](.agents/skills/tlc-spec-driven/SKILL.md) — fluxo Specify → Design → Tasks → Execute para features não-triviais

### Documentação do projeto (autoritativa)
- [README.md](README.md) — overview + quickstart
- [COMO_USAR.md](COMO_USAR.md) — guia de usuário final (passo a passo)
- [COMO_FUNCIONA.md](COMO_FUNCIONA.md) — arquitetura interna (TurboQuant, RRF, PageRank)
- [docs/CLI_GUIDE.md](docs/CLI_GUIDE.md) — manual detalhado de cada subcomando + storage selection
- [docs/AGENT_INTEGRATION_GUIDE.md](docs/AGENT_INTEGRATION_GUIDE.md) — quando `.memory/` + MCP coexistem em agentes externos
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — schemas DDL + SQL recursivo
- [docs/TURBOQUANT.md](docs/TURBOQUANT.md) — matemática da quantização 4-bit
- [docs/CENTRAL_VAULT.md](docs/CENTRAL_VAULT.md) — arquitetura de Federação e Central Vault
- [docs/REPOSITORY_BRAIN.md](docs/REPOSITORY_BRAIN.md) — filosofia e integração MCP

### Decisões arquiteturais vigentes
- **ADR-001** — SQLite como Camada Unificada
- **ADR-002** — Go como linguagem principal
- **ADR-003** — TurboQuant 4-bit
- **ADR-004** — SQL Recursivo (CTEs) para grafo
- **ADR-005** — Markdown com `[[Wikilinks]]`
- **ADR-006** — MCP como integração com IA
- **ADR-010** — Cache incremental SHA-256
- **ADR-011** — Arestas epistêmicas tipadas
- **ADR-014** — PageRank ponderado
- **ADR-015** — Decaimento temporal exponencial
- **ADR-016** — Auto-scoping de vault (superseded by ADR-040)
- **ADR-018** — Compile-not-Retrieve
- **ADR-024** — Triptych Inspector (`mem inspect`)
- **ADR-025** — Progressive Context Loading + taxonomia `category`
- **ADR-031** + **ADR-039** — Drift detection calibrado
- **ADR-033** + **ADR-034** — Federação e Central Vault
- **ADR-040** — **Config global única + SQLite auto-scope default + Postgres opt-in** (vigente)
- **ADR-041** — Tags sem casa removidas do grafo (ISSUE-009 fix)

### Operacional
- [AGENTS.md](AGENTS.md) — instruções para Agentes de IA no repo
- [.agents/rules/always-quality-gate.md](.agents/rules/always-quality-gate.md) — gate de fim de tarefa
- [.agents/rules/always-update-docs-and-specs-on-pr.md](.agents/rules/always-update-docs-and-specs-on-pr.md) — regra do Princípio 7
- [ISSUES.md](ISSUES.md) — tracking de bugs/limitações
- [`.specs/`](.specs/) — specs atômicas de features (040, 041, etc.)

---

> **Mantida por:** Felipe Miiller · **Slug:** `my-memory` · **Status:** Canônica · **Última atualização:** mantém sincronizada com `docs/CLI_GUIDE.md` a cada PR.
