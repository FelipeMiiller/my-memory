# Guia da Linha de Comando (CLI)

O executável `mem` fornece uma interface direta para indexação e consulta semântica da base de conhecimento.

---

## 🛠 Compilação

Para compilar o binário em Go:

```bash
# Na raiz do projeto:
go build -o bin/mem.exe ./cmd/mem
```

---

## 📖 Comandos Disponíveis

### 1. `mem init [--repo <slug>] [--db <caminho>] [--force] [<pasta>]`
Inicializa um novo vault criando a pasta `.memory/` e gerando o arquivo de configuração declarativa `config.yaml` com comentários explicativos:
* `--repo`: Slug do repositório/vault (padrão: inferido do Git ou `local/vault`).
* `--db`: Caminho do banco SQLite padrão (padrão: `memory.db`).
* `--force`: Sobrescreve o arquivo caso já exista.

**Exemplo:**
```bash
./bin/mem.exe init --repo "empresa/meu-vault"
```

---

### 2. `mem index [<pasta>] [--force] [--no-prune]`
Percorre recursivamente os arquivos do vault respeitando as regras declarativas de `include` e `exclude`:
* **Auto-scoping**: Se `<pasta>` for omitida, descobre automaticamente a raiz do vault procurando `.memory/config.yaml` de forma ascendente.
* Salva os metadados do documento e hash criptográfico SHA-256 no banco.
* Poda automaticamente notas deletadas do disco (`PruneDeletedDocuments`). Use `--no-prune` para preservar registros ausentes.
* Extrai os `[[wikilinks]]` e `#tags` inserindo as arestas de relacionamento no grafo.
* Pula automaticamente diretórios de sistema (`.git`, `node_modules`, `vendor`, `.obsidian`, `.trash`, `.memory`) via `filepath.SkipDir`.
* Gera os embeddings de 768 dimensões via Ollama (`nomic-embed-text`).
* Insere no `sqlite-vec` (vetor completo) e no `chunks_turboquant` (comprimido em 4-bits).

**Exemplo:**
```bash
# Executando no diretório do vault (auto-scoping ativo):
./bin/mem.exe index

# Ou especificando uma pasta explicitamente:
./bin/mem.exe index C:/meu-vault-obsidian
# Sem podar arquivos ausentes:
./bin/mem.exe index --no-prune ./docs
```

---

### 3. `mem watch [--debounce <ms>] [--interval <ms>] [--db <arq>] [--postgres <url>] [--repo <slug>] [<pasta>]`
Monitora continuamente alterações de arquivos Markdown no vault em tempo real:
* **Detecção Automática**: Identifica criação, modificação e deleção de notas Markdown em segundo plano sem necessidade de CGO.
* **Debouncing Amortecido**: Agrupa rajadas de salvamento contínuo (padrão: 500ms) executadas por editores de código e pelo Obsidian para evitar chamadas redundantes ao Ollama.
* **Reindexação Cirúrgica**: Processa exclusivamente o arquivo alterado (documento, arestas, chunks e embeddings), sem reindexar o restante do vault.
* **Purga Automática**: Deleta instantaneamente do índice notas que foram removidas do disco.
* **Encerramento Gracioso**: Suporta encerramento seguro via `Ctrl+C` (`SIGINT`/`SIGTERM`).

**Exemplo:**
```bash
# Iniciar monitoramento em tempo real no vault atual:
./bin/mem.exe watch

# Customizando intervalo de polling e janela de debounce:
./bin/mem.exe watch --debounce 300 --interval 500
```

---

### 4. `mem hook <install|uninstall> [--force] [<pasta>]`
Gerencia a instalação do Git Pre-Commit Hook para garantir integridade do índice de conhecimento antes de cada commit:
* `install`: Localiza a pasta `.git/hooks` e grava o script executável `pre-commit` assinado pelo My-Memory.
* Protege hooks de outros linters existentes, exigindo `--force` apenas se houver conflito com ferramentas externas.
* `uninstall`: Remove de forma limpa e idempotente o hook pre-commit instalado pelo My-Memory.

**Exemplo:**
```bash
# Instalar o hook pre-commit no repositório atual:
./bin/mem.exe hook install

# Forçar instalação sobrescrevendo hooks desconhecidos:
./bin/mem.exe hook install --force

# Desinstalar o hook:
./bin/mem.exe hook uninstall
```

---

### 5. `mem search "<pergunta>" [--mode hybrid|vector|fts] [--decay] [--half-life 30] [--decay-weight 0.3]`
Realiza a busca híbrida via Reciprocal Rank Fusion (RRF) combinando texto exato FTS5/tsvector, vetores semânticos k-NN e expansão de grafo:
* `--mode`: Escolhe entre `hybrid` (padrão, fusão RRF), `vector` (apenas semântico k-NN) ou `fts` (apenas texto exato).
* `--decay`: Ativa o decaimento temporal exponencial ponderado (ADR-015) para priorizar notas mais recentes no ranking final.
* `--half-life <dias>`: Tempo de meia-vida da curva em dias (padrão: 30.0 dias).
* `--decay-weight <w>`: Peso do decaimento entre 0.0 (sem efeito) e 1.0 (decaimento máximo) com piso assintótico $(1 - w)$ (padrão: 0.3).
* Realiza a **expansão de grafo** via SQL recursivo para trazer notas estruturalmente conectadas.

**Exemplo:**
```bash
./bin/mem.exe search "como funciona o fluxo de autenticacao?"
# Busca com decaimento temporal agressivo (meia-vida de 15 dias, peso 0.5):
./bin/mem.exe search --decay --half-life 15 --decay-weight 0.5 "decisoes de arquitetura"
# Filtrando por repositório ou conectando ao PostgreSQL:
./bin/mem.exe search --postgres "postgres://user:pass@localhost:5432/memory?sslmode=disable" --repo "meu-org/meu-projeto" "fluxo de autenticacao"
```

---

### 6. `mem search -tq "<pergunta>"` (Modo TurboQuant)
Realiza a busca ultrarrápida utilizando os blocos compactados em 4-bits no SQLite:
* Rotaciona o vetor da query via Householder.
* Calcula o produto escalar não-viesado diretamente sobre os BLOBs comprimidos.
* Aplica a mesma expansão de grafo sobre os resultados.

**Exemplo:**
```bash
./bin/mem.exe search -tq "qual a regra para calculo de comissao?"
```

---

### 7. `mem mcp [--port <porta>] [--host <ip>] [--http <addr>] [--cors] [--db <caminho>] [--postgres <url>] [--repo <slug>]`
Inicia o servidor Model Context Protocol (MCP) via **`stdio`** (padrão) ou como **servidor de rede HTTP / Server-Sent Events (SSE)** conforme a especificação oficial da Anthropic (ADR-021):
* **Modo Stdio (Padrão):** Ideal para processos filhos locais gerenciados por Claude Code, Cursor, Windsurf e Antigravity.
* **Modo Servidor de Rede (`--port` ou `--http`):** Permite acesso de agentes remotos, múltiplos clientes paralelos e dashboards web.
* **Endpoints HTTP Disponíveis:**
  * `GET /sse`: Inicia stream Server-Sent Events e emite o evento `endpoint`.
  * `POST /message?sessionId=<uuid>`: Envia requisições JSON-RPC 2.0 associadas à sessão SSE ativa.
  * `POST /mcp`: Chamada JSON-RPC direta (stateless) para scripts rápidos ou `curl`.
  * `GET /health`: Diagnóstico com uptime, ferramentas registradas e status operacional.
* **CORS Habilitado:** `--cors` (padrão: ativo) permite chamadas de navegadores web com preflight `OPTIONS`.

**Exemplos:**
```bash
# 1. Modo Stdio local tradicional:
./bin/mem.exe mcp --db memory.db

# 2. Modo Servidor HTTP/SSE na porta 38400:
./bin/mem.exe mcp --port 38400

# 3. Expondo na rede local para múltiplos agentes:
./bin/mem.exe mcp --host 0.0.0.0 --port 38400 --repo "meu-org/projeto"

# 4. Verificando a saúde via curl:
curl http://localhost:38400/health

# 5. Executando busca via endpoint direto:
curl -X POST http://localhost:38400/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"memory_search","arguments":{"query":"arquitetura"}}}'
```

---

### 8. `mem export [--canvas <nota>] [--html <saida.html>] [--depth 1] [--out <arquivo>]`
Exporta o grafo relacional para **JSON Canvas 1.0 (`.canvas`)** do Obsidian ou para uma página **HTML/SVG interativa standalone**:
* Modo Canvas: Gera arquivo `.canvas` para abrir diretamente no Obsidian.
* Modo HTML (`--html`): Gera página web standalone com simulação de física de forças, busca em tempo real e painel lateral de detalhes.

**Exemplos:**
```bash
# Exportar subgrafo para Obsidian JSON Canvas:
./bin/mem.exe export --canvas "Arquitetura" --depth 2 --out "mapa_arquitetura.canvas"

# Exportar grafo completo para página HTML interativa:
./bin/mem.exe export --html "grafo_completo.html"
```

---

### 9. `mem doctor [--fix] [--db <caminho>] [--postgres <url>] [--repo <slug>]`
Audita a saúde do grafo e tabelas relacionais de conhecimento:
* Identifica **Dead Links** (wikilinks apontando para notas inexistentes).
* Identifica **Notas Órfãs** (documentos sem conexões de entrada ou saída).
* Identifica **Self-Loops** (notas apontando reflexivamente para si mesmas).
* Calcula o **Health Score** (0 a 100).
* A flag `--fix` remove automaticamente conexões mortas e loops conhecidos.

**Exemplo:**
```bash
./bin/mem.exe doctor
# Auditando e reparando anomalias no grafo:
./bin/mem.exe doctor --fix
```

---

### 10. `mem hubs [--algorithm degree|pagerank] [--damping 0.85] [--iter 30] [--top 10]` e `mem insights`
* `mem hubs`: Exibe os nós centrais do grafo. Suporta ordenação por grau bruto (`--algorithm degree` - padrão) ou por autoridade estrutural iterativa com pesos epistêmicos (`--algorithm pagerank`).
* `mem insights`: Descobre conexões latentes (*Surprising Connections*) entre notas com alta similaridade sem links diretos.

**Exemplo:**
```bash
./bin/mem.exe hubs --top 5
# Calculando nós centrais via PageRank com amortecimento:
./bin/mem.exe hubs --algorithm pagerank --damping 0.85 --iter 30 --top 5
./bin/mem.exe insights --min-similarity 0.75
```

---

### 11. `mem clusters [--min-size 2] [--json] [--db <caminho>] [--postgres <url>] [--repo <slug>]`
Detecta comunidades temáticas e clusters densamente conectados no grafo relacional usando o algoritmo *Weighted Label Propagation Algorithm* (LPA) e calcula a Modularidade Newman-Girvan \(Q\):
* **LPA Ponderado:** Considera os pesos epistêmicos das arestas (`EXTRACTED` 1.0, `INFERRED` 0.6, `TAG` 0.3) com desempate determinístico lexicográfico.
* **Modularidade \(Q\):** Quantifica o grau de coesão e separação estrutural da memória (valores $> 0.3$ indicam forte coesão).
* **Nó Líder e Tipo Dominante:** Identifica automaticamente o nó central com maior PageRank local e a categoria predominante de nota (`concept`, `decision`, etc.).
* `--min-size <N>`: Filtra clusters menores que $N$ nós (padrão: 2, ocultando nós isolados).
* `--json`: Emite o resultado em formato JSON estruturado com métricas globais e array de comunidades.

**Exemplos:**
```bash
# Detectar clusters temáticos com tamanho >= 2:
./bin/mem.exe clusters

# Incluir nós isolados (tamanho 1):
./bin/mem.exe clusters --min-size 1

# Exportar partições e modularidade em JSON:
./bin/mem.exe clusters --json
```

---

### 12. `mem bench`
Executa a suíte de micro-benchmarks quantitativos da biblioteca (TurboQuant 4-bit, fusão RRF, SHA-256 e parsing) com saída tabular detalhada.

**Exemplo:**
```bash
./bin/mem.exe bench
```

---

### 13. `mem note <create|append> [opções] <caminho>`
Cria ou anexa seções em notas atômicas em Markdown com frontmatter YAML limpo e sincronização cirúrgica imediata no banco de dados e grafo:
* `create`: Cria nota atômica com frontmatter (`title`, `type`, `tags`, `aliases`) e corpo Markdown. Rejeita sobrescrita a menos que `--overwrite` seja passado.
* `append`: Anexa texto cirurgicamente antes da próxima seção de mesmo nível ou cria nova seção caso não exista.

**Exemplo:**
```bash
# Criar nota atômica:
./bin/mem.exe note create --title "Padrão de Autenticação" --tags "auth,security" --type "concept" concepts/auth.md

# Anexar nova seção:
./bin/mem.exe note append --heading "## Sessões e Cookies" concepts/auth.md "Configurar cookies com flags HttpOnly e SameSite=Strict."
```

---

### 14. `mem compile --topic "<termo>" --out "<caminho.md>" [--limit 5] [--mode hybrid|vector|fts]`
Executa o padrão **Compile-not-Retrieve** (Karpathy LLM Wiki): recupera os fragmentos mais relevantes sobre um tópico via busca híbrida e gera uma nota consolidada com seção de síntese e backlinks tipados (`[[rel:derived_from:Doc]]`), sincronizando instantaneamente no grafo.

**Exemplo:**
```bash
./bin/mem.exe compile --topic "decisões de banco de dados e sqlite" --out syntheses/db-decisions.md --limit 5
```

---

### 15. `mem version [--json]`
Exibe a versão do executável, hash Git do commit, data de compilação, versão do Go e arquitetura do sistema operacional. Também acessível através das flags `-v` e `--version`.

**Exemplo:**
```bash
# Saída amigável em texto:
./bin/mem.exe version
# my-memory v1.0.0 (commit: f855dd4, built: 2026-09-13T21:30:00Z, go: go1.23.0, windows/amd64)

# Saída estruturada em JSON (ideal para agentes e scripts):
./bin/mem.exe version --json
```

---

### 16. `mem graph [view|export] [--root <nota>] [--depth 2] [--out <saida.html>] [--open]`
Gera e abre no navegador uma visualização interativa do grafo da memória do repositório, em uma página HTML/SVG 100% autocontida (Zero-CDN) com simulação de física de forças:
* **`mem graph view`**: Compila o grafo e abre imediatamente no navegador padrão do sistema.
* **`mem graph export`**: Compila e grava o arquivo HTML no disco sem abrir o navegador (a menos que `--open` seja passado).
* **Filtro de Subgrafo**: Use `--root <nota>` e `--depth <N>` para isolar a vizinhança de uma nota específica.
* **Recursos da Interface**:
  - Zoom e pan contínuo na tela.
  - Arraste gravitacional interativo de nós.
  - Nós dimensionados pela autoridade estrutural calculada via **PageRank**.
  - Paleta de cores harmoniosa por tipo de nota (`concept`, `decision`, `guide`, `reference`, `synthesis`).
  - Busca instantânea de notas com atenuação visual de nós não correlacionados.
  - Painel lateral retrátil com conexões de entrada/saída e botão direto para abrir no Obsidian (`obsidian://open?file=...`).

**Exemplos:**
```bash
# Visualizar o grafo global completo no navegador padrão:
./bin/mem.exe graph view

# Visualizar subgrafo centrado em uma decisão arquitetural:
./bin/mem.exe graph view --root "decisions/adr-009-busca-hibrida-com-reciprocal-rank-fusion-rrf.md" --depth 2

# Exportar para arquivo específico sem abrir o navegador:
./bin/mem.exe graph export --out "docs/mapa_conhecimento.html"
```

---

## ⚙️ Configuração Declarativa do Vault (`.memory/config.yaml`)

O My-Memory suporta configuração declarativa por projeto ou vault de notas. Ao executar qualquer comando, o binário procura recursivamente de baixo para cima por `.memory/config.yaml`, `.mem.yaml` ou `.mem.json`.

### Exemplo de `.memory/config.yaml`:
```yaml
version: 1
repository: "minha-org/vault-conhecimento"
vault_name: "Knowledge Vault"

# Padrões glob de arquivos a indexar
include:
  - "**/*.md"

# Padrões glob de arquivos e pastas ignorados
exclude:
  - ".git/**"
  - "node_modules/**"
  - "vendor/**"
  - ".obsidian/**"
  - ".trash/**"
  - ".memory/**"

# Persistência
storage:
  engine: "sqlite"          # "sqlite" ou "postgres"
  sqlite_path: "memory.db"

# Embeddings
embedding:
  provider: "ollama"
  model: "nomic-embed-text"
  url: "http://localhost:11434"
  dimension: 768

# Preferências padrão de busca
search:
  mode: "hybrid"            # "hybrid", "vector" ou "fts"
  limit: 5
  k: 60
  decay: false              # Decaimento temporal ativado
  half_life: 30.0           # Meia-vida em dias
  decay_weight: 0.3         # Peso do decaimento
  use_turbo: false          # TurboQuant 4-bit

# Monitoramento em tempo real (mem watch)
watcher:
  debounce_ms: 500          # Janela de amortecimento para agrupar rajadas
  interval_ms: 1000         # Intervalo de polling periódico
```

### 🏆 Ordem de Precedência (Prioridade):
1. **Flags de Terminal**: (`--mode`, `--limit`, `--decay`, `--repo`, `--db`, etc.)
2. **Variáveis de Ambiente**: (`MY_MEMORY_PG_URL`, `MY_MEMORY_REPO`)
3. **Arquivo de Configuração**: (`.memory/config.yaml` ou `.mem.yaml`)
4. **Defaults de Código**: (`DefaultConfig()`)


