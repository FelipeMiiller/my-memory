# 🏛️ Central Vault — Global Brain do My-Memory

> **Status**: v1.3.0 — Implementado pelos ADRs [ADR-033](adr/033-federated-central-vault-and-repo-identity.md) e [ADR-034](adr/034-protocolo-canonico-federado-e-wikilinks-cross-vault.md).

Este documento descreve o **Cofre Central de Conhecimento** (também chamado de **Global Brain** ou **central vault**) do My-Memory — a camada que conecta repositórios satélites em uma federação de grafos unificada, com persistência opcional na nuvem (Google Drive, OneDrive, Dropbox) e wikilinks cross-vault via URIs canônicas `memory://`.

---

## 1. O que é o Central Vault?

O **Cofre Central** é uma pasta Markdown local — sincronizada opcionalmente pelo seu provedor de nuvem preferido — que funciona como o **repositório canônico de conhecimento corporativo/pessoal**. Ele é registrado no catálogo global (`~/.memory/config.yaml`) e recebe um identificador imutável (`repo_central`) que todos os outros repositórios reconhecem.

Concretamente, ele é:

1. **Um vault Obsidian comum** — pastas + `.md` + wikilinks `[[...]]` + tags, aberto nativamente no Obsidian.
2. **Resolvido por uma URI canônica** `memory://central/...` — toda referência a uma nota canônica do cofre central funciona a partir de qualquer repositório satélite.
3. **Indexado pelo `mem index`** — todos os arquivos Markdown viram nós do grafo + chunks FTS5 + embeddings vetoriais.
4. **Auto-bootstrappado** — ao primeiro `mem setup --central <pasta>`, o `mem` cria 11 pastas canônicas (`standards/`, `architecture/`, `security/`, `infrastructure/`, `operations/`, `data/`, `ai-agents/`, `domain/`, `guides/`, `templates/`, `staging/`) com READMEs e templates prontos.
5. **Opcionalmente compartilhado** — você pode apontar o cofre central para uma pasta sincronizada (Google Drive, OneDrive, Dropbox) para colaboração multi-device ou multi-usuário.

---

## 2. Quando usar o Central Vault?

| Cenário | Use central vault? |
| --- | --- |
| Knowledge management pessoal único | Não obrigatório, mas recomendado |
| Múltiplos projetos/repositórios com decisões compartilhadas (ADRs, RFCs, runbooks) | ✅ Sim — evita duplicação |
| Times distribuídos que precisam compartilhar padrões | ✅ Sim — o cofre central fica na nuvem |
| Pesquisa acadêmica com notas multi-paper | ✅ Sim — `domain/` centraliza conceitos |
| Projeto único sem necessidade de federação | Opcional — o vault local do repo já funciona sozinho |

---

## 3. Setup do Cofre Central

### 3.1. Comando principal
```bash
mem setup --central "<caminho_do_cofre>" --engine sqlite --yes
```
- `--central`: caminho absoluto da pasta que será o cofre central (ex.: `"G:\My Drive\central-memory"`, `"~/Documents/central-memory"`).
- `--engine`: `sqlite` (default) ou `postgres` (com pgvector).
- `--yes`: executa sem prompts interativos.

O comando grava em `~/.memory/config.yaml`:
```yaml
version: 1
central_vault:
  path: G:\My Drive\central-memory
  vault_name: central
storage:
  engine: sqlite
mcp:
  port: 8080
```

### 3.2. Estrutura canônica criada

Após `mem central bootstrap` (executado automaticamente pelo setup), você terá:

```
central-memory/
├── .memory/                 # Config local + memory.db + .env (ignorado pelo git)
├── standards/               # Padrões corporativos (auth, segurança, naming)
├── architecture/            # ADRs, topologias, diagramas C4, microsserviços
├── security/                # Threat models, controles, auditoria
├── infrastructure/          # IaC, redes, deploy
├── operations/              # Runbooks, on-call, SLOs
├── data/                    # Schemas, dicionário de dados, linhagem
├── ai-agents/               # Skills, prompts, MCP servers por agente
├── domain/                  # Conceitos de domínio (DDD), glossários
├── guides/                  # Playbooks, onboarding, how-to
├── templates/               # Modelos: ADR, RFC, runbook, spec
├── staging/                 # Captura rápida antes de promoção
└── README.md                # MOC raiz (Map of Content)
```

Cada pasta canônica recebe um `README.md` com frontmatter L0/L1/L2, links para a raiz via `[[../README]]` e metadados `category: memory` ou `resource`.

### 3.3. Repositórios satélites
Repositórios de projeto (não-centrais) também podem ser registrados no catálogo global:
```bash
# A partir do diretório do projeto
mem init
# (idempotente — registra o repo se ainda não estiver no catálogo)
```
Eles passam a ser endereçáveis via `memory://<repo-slug>/...`.

---

## 4. Modos de Embedding (vetorial vs fallback léxico)

O `mem index` requer embeddings para o componente semântico da busca híbrida. O provider é o **modelo embutido** `paraphrase-multilingual-MiniLM-L12-v2` (ADR-035), com vetores de **384 dimensões** — não há servidor de embeddings remoto a configurar.

### 4.1. Vetorial (padrão)
```yaml
# .memory/config.yaml do projeto
embedding:
  provider: "builtin"
  # model e dimension são fixos pelo schema do banco (vector(384))
```
Com o provider embutido disponível, cada chunk é vetorizado e indexado em `chunks_vec` (sqlite-vec). A busca híbrida combina FTS5 + RRF + distância vetorial.

### 4.2. Fallback léxico FTS (provider indisponível)

O fallback existe, mas **não é mais disparado por um servidor externo ficar offline** — a migração para o provider embutido removeu essa dependência de rede. Ele dispara em dois casos reais:

1. O binário foi compilado **sem a tag `onnx`**, então `newBuiltinEmbedder` não existe (`cmd/mem/builtin_embedder_disabled.go`);
2. O modelo embutido falha ao gerar um vetor (artefatos ausentes, erro de inferência).

> [!IMPORTANT]
> O binário oficial **já vem com a tag `onnx`**: o release compila com `-tags "sqlite_fts5 onnx"` e a CI tem um job que compila e testa o provider sob essa tag. Se o seu binário avisa que a tag falta, ele foi compilado à mão — recompile com a tag (precisa de toolchain C/CGO) ou baixe o artefato do release.

```go
// cmd/mem/main.go — resolveEmbedderStatus()
emb, err := newBuiltinEmbedder(cfg)
if err != nil {
    emb = nil // quem decide o que avisar é o comando: o index imprime UM
              // aviso consolidado e a busca declara o fallback léxico
}
```

Comportamento:
- ⚠️ **Um** aviso por execução de `mem index` (não uma linha por documento):
  `Provider vetorial indisponível: N documento(s) indexado(s) SEM VETOR — a busca semântica não funciona para eles.`
  A mensagem nomeia a **tag ausente** (`onnx`) e a consequência.
- ✅ Indexação **continua normalmente** no FTS5 + grafo, e o `mem index` sai com **código 0** — o modo léxico é um modo válido.
- ✅ O chunk é gravado **sem vetor**: nada em `chunks_vec` e nada em `chunks_turboquant` (nenhuma linha comprimida com vetor zero). O `mem doctor` passa a listar esse chunk como **sem vetor real** — a ausência é visível, e o Health Score penaliza (2 pontos por chunk, ADR-013).
- ❌ Busca semântica fica degradada — `mem search --mode hybrid` cai em FTS-only
- ✅ `--mode fts` continua 100%

> [!IMPORTANT]
> O reparo de um cofre indexado sem provider é `mem index --force` **com** o provider presente (binário com a tag `onnx`): ele regrava os chunks com vetor real. Isso já valia antes — o que mudou é que agora o `mem doctor` **mostra** quais chunks estão sem vetor, em vez de declarar o cofre saudável.

**Verifique se está em modo fallback** rodando:
```bash
mem search --mode hybrid "qualquer coisa"
# Saída esperada:
# ⚠️  Resultado apenas LEXICAL — sem provider de embedding. Esta busca NÃO usou similaridade vetorial; o resultado vem só do FTS textual.
# 📊 Fontes RRF: [fts:1]
```

> [!TIP]
> `mem embed doctor` diagnostica o ambiente (provider ativo, caminhos, dimensão e a presença de cada artefato) e `mem embed download` busca o que falta. Ver [`docs/CLI_GUIDE.md` — `mem embed`](CLI_GUIDE.md).

---

## 5. Comandos do Central Vault

### 5.1. `mem central`
```bash
mem central status          # Verifica status de conexão e saúde do cofre central
mem central bootstrap       # (Re-)inicializa estrutura canônica em cofre virgem
```

### 5.2. `mem setup`
```bash
mem setup                                                     # Modo interativo
mem setup --central "~/KnowledgeVault" --engine sqlite --yes  # Modo não-interativo
```

### 5.3. `mem repos`
Lista todos os repositórios registrados no catálogo global (`~/.memory/config.yaml`):
```bash
mem repos
# Saída:
#   repo_central      G:\My Drive\central-memory      (central)
#   payments-service  C:\src\payments-service          (satélite)
#   docs-platform     C:\src\docs-platform             (satélite)
```

### 5.4. `mem open` com URI federada
O `mem open` aceita URIs `memory://` para abrir notas no editor configurado:

```bash
# Nota canônica do cofre central
mem open "memory://central/standards/oauth2" --app obsidian

# Nota em repositório satélite
mem open "memory://payments-service/docs/api" --app vscode

# Dry-run (mostra o que vai abrir sem abrir)
mem open "memory://central/architecture/pgvector" --dry-run

# JSON com metadados
mem open "memory://central/standards/oauth2" --json
```
Saída JSON de exemplo:
```json
{
  "uri": "memory://central/standards/oauth2",
  "vault": "central",
  "file": "standards/oauth2.md",
  "path": "G:\\My Drive\\central-memory\\standards\\oauth2.md",
  "deep_link": "obsidian://open?vault=central&file=standards/oauth2",
  "is_federated": true
}
```

---

## 6. Federação e wikilinks cross-vault

A URI canônica `memory://` permite que wikilinks `[[...]]` em **qualquer repositório** apontem para notas do cofre central sem precisar copiar conteúdo:

```markdown
# Em qualquer arquivo .md de QUALQUER repositório federado

Veja o padrão canônico em [[memory://central/standards/oauth2|nosso ADR de OAuth2]].

Compare com [[memory://payments-service/docs/api|a documentação interna do serviço de pagamentos]].
```

O parser (`internal/parser/wikilinks.go`) reconhece o esquema `memory://`, extrai o vault (segmento após `memory://`), o caminho do arquivo, e cria uma aresta federada no grafo.

Para habilitar esse cross-vault na sua config de vault:
```yaml
# .memory/config.yaml local do projeto satélite
vault:
  central_ref: "memory://central"
  federated_search: true
```

---

## 7. Persistência opcional na nuvem

A camada de storage (`storage.engine`) é independente da localização da pasta. Você pode:

- **SQLite local** (default): o `memory.db` fica dentro de `.memory/`. Sincronização via Google Drive/OneDrive copia o arquivo binário — risco de corrupção se edições simultâneas.
- **PostgreSQL com pgvector**: o schema vive num servidor Postgres centralizado, acessado por todos os satélites. Recomendado para times distribuídos.

Configuração:
```bash
# Via variáveis de ambiente
export MY_MEMORY_PG_URL="postgres://user:pass@db.example.com:5432/memory?sslmode=require"

# Ou via flag
mem index --postgres "postgres://..."
```

Mais detalhes em [docs/POSTGRES_SETUP.md](POSTGRES_SETUP.md) (quando aplicável).

---

## 8. Ciclo de vida do conhecimento

O cofre central implementa um **fluxo de promoção**:

1. **Captura** (`staging/`) — qualquer ideia, rascunho, anotação de reunião.
2. **Curadoria** (refinamento com agente de IA + humanos) — adiciona YAML, wikilinks, tags.
3. **Promoção** (mover para pasta temática definitiva) — `standards/`, `architecture/`, etc.

Isso é descrito no `README.md` raiz do cofre central e referenciado pela taxonomia `category: memory | resource | skill` no frontmatter.

---

## 9. MCP Server federado

O `mem mcp` expõe todas as operações do cofre central via Model Context Protocol:

| Tool MCP | Descrição |
| --- | --- |
| `memory_search` | Busca híbrida com RRF e expansão de grafo |
| `memory_open_node` | Abre nó por URI `memory://` |
| `memory_get_neighbors` | Vizinhos diretos no grafo (federados) |
| `memory_inspect_node` | Visualização triptych (in / nó / out) |
| `memory_get_drift` | Desvio código-memória |
| `memory_get_insights` | Conexões surpreendentes (Surprising Connections) |

O servidor pode rodar em stdio (default — para clientes locais como Claude Desktop) ou HTTP/SSE (porta configurável).

---

## 10. Verificação e diagnóstico

```bash
# Status do cofre central
mem central status

# Status da indexação
mem status --db "<caminho_para_memory.db>"

# Saúde do grafo (dead links, órfãos, self-loops)
mem doctor --db "<caminho_para_memory.db>"

# Drift código-memória (HEAD~5..HEAD por default)
mem drift --db "<caminho_para_memory.db>"

# Hubs estruturais
mem hubs --algorithm pagerank --top 10

# Clusters
mem clusters --min-size 2
```

---

## 11. Troubleshooting

| Sintoma | Causa provável | Solução |
| --- | --- | --- |
| `unable to open database file (14)` ao indexar | Path com caracteres especiais / falta de permissão | Verifique aspas, encoding e permissões de escrita |
| Aviso `Provider vetorial indisponível` (uma linha por execução) | Binário compilado sem a tag `onnx`, ou artefatos do modelo embutido ausentes | `mem embed doctor` mostra o diagnóstico; `mem embed download` busca o que falta. O binário do release já vem com a tag `onnx` — se o aviso aparece, recompile com `-tags "sqlite_fts5 onnx"` (ver [ADR-035](adr/035-embedder-embutido-com-fallback-onnx-minilm.md)) |
| `mem open memory://...` não acha o nó | Repo satélite não registrado no catálogo global | Rode `mem init` no diretório do repo |
| Drift mostra ADRs órfãos | ADRs ficaram desatualizados vs código novo | Atualize ADR, commite, reindexe |
| `--db` mostra help em vez de erro | Argumento mal posicionado | `mem index --db <arq> <pasta>` (DB antes do path) |

---

## 12. Referências

- [ADR-033 — Arquitetura Federada com Cofre Central e Identidade Imutável de Repositório](adr/033-federated-central-vault-and-repo-identity.md)
- [ADR-034 — Protocolo Canônico Federado e Wikilinks Cross-Vault](adr/034-protocolo-canonico-federado-e-wikilinks-cross-vault.md)
- [CLI Guide — Seção Federação e Cofre Central](CLI_GUIDE.md#-federação-e-cofre-central-de-conhecimento-mem-setup-mem-central-mem-repos)
- [Architecture — Camadas de persistência e federação](ARCHITECTURE.md)
- [Repository Brain — Visão geral do modelo My-Memory](REPOSITORY_BRAIN.md)
