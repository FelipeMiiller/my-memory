---
title: "Guia da Linha de Comando (CLI)"
category: skill
summary: "Referência completa de comandos, sintaxe, flags e exemplos práticos para o executável mem (CLI do My-Memory)."
tags: [cli, skill]
---

# Guia da Linha de Comando (CLI)

O executável `mem` fornece uma interface direta para indexação e consulta semântica da base de conhecimento.

---

## ⚡ Instalação e Compilação

### 1. Instalador Automático One-Liner (Recomendado)

O My-Memory fornece scripts de instalação automática que detectam o sistema operacional e a arquitetura, baixam o binário pré-compilado das Releases oficiais do GitHub, validam a integridade criptográfica **SHA-256** e adicionam o comando ao `PATH` do usuário sem exigir privilégios de administrador:

#### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/FelipeMiiller/my-memory/main/scripts/install.ps1 | iex
```

Parâmetros suportados no Windows:
* `-Version <vX.Y.Z>`: Instala uma versão específica da release (ex: `-Version v1.2.0`).
* `-InstallDir <caminho>`: Define o diretório de destino (padrão: `$env:USERPROFILE\.mem\bin`).
* `-NoPath`: Não altera o PATH do registro nem da sessão.
* `-Force`: Sobrescreve o binário existente.

#### Linux e macOS (POSIX Shell)
```bash
curl -fsSL https://raw.githubusercontent.com/FelipeMiiller/my-memory/main/scripts/install.sh | sh
```

Parâmetros suportados no Linux/macOS:
* `--version <vX.Y.Z>` / `-v`: Instala uma versão específica da release.
* `--dir <caminho>` / `-d`: Define o diretório de destino (padrão: `~/.local/bin` ou `/usr/local/bin` se root).
* `--no-path`: Não sugere inclusão no PATH.
* `--force` / `-f`: Sobrescreve o binário existente.

### 2. Instalação via Go Toolchain
```bash
go install github.com/FelipeMiiller/my-memory/cmd/mem@latest
```

### 3. Compilação do Código-Fonte
Para compilar o binário em Go localmente:

```bash
# Na raiz do projeto:
go build -o bin/mem.exe ./cmd/mem
```

---

## 📖 Comandos Disponíveis

### 1. `mem init [--repo <slug>] [--db <caminho>] [--force] [--vscode] [--copilot] [--cursor] [--all] [<pasta>]`
Inicializa um novo vault criando a pasta `.memory/` e gerando o arquivo de configuração declarativa `config.yaml` com comentários explicativos:
* `--repo`: Slug do repositório/vault (padrão: inferido do Git ou `local/vault`).
* `--db`: Caminho do banco SQLite padrão (padrão: `memory.db`).
* `--force`: Sobrescreve arquivos caso já existam.
* `--vscode`: Gera `.vscode/mcp.json` configurado para integração com o VS Code e Copilot Chat via MCP.
* `--copilot`: Gera `.github/copilot-instructions.md` com instruções obrigatórias de contexto para o GitHub Copilot.
* `--cursor`: Gera `.cursor/mcp.json` para o Cursor IDE.
* `--all`: Gera todas as configurações de IDE acima de uma só vez.

**Exemplos:**
```bash
# Inicialização básica (apenas .memory/config.yaml):
./bin/mem.exe init --repo "empresa/meu-vault"

# Inicialização com integração completa para VS Code e GitHub Copilot:
./bin/mem.exe init --vscode --copilot

# Inicialização universal para todas as IDEs suportadas:
./bin/mem.exe init --all
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

### 5. `mem search "<pergunta>" [--level l0|l1|l2] [--category resource|memory|skill] [--mode hybrid|vector|fts] [--decay] [--half-life 30] [--decay-weight 0.3]`
Realiza a busca híbrida via Reciprocal Rank Fusion (RRF) combinando texto exato FTS5/tsvector, vetores semânticos k-NN, Progressive Context Loading e expansão de grafo:
* `--level <l0|l1|l2>`: Define a densidade de contexto do resultado:
  * `l0`: Micro-abstract cirúrgico (1 a 2 frases, ~30-50 tokens) com deduplicação por documento e sem despejo de texto bruto (*Zero File Reads*).
  * `l1`: Overview estrutural padrão com metadados, resumo L0, trecho relevante e conexões do grafo.
  * `l2`: Detalhes completos e conteúdo integral.
* `--category <resource|memory|skill>`: Filtra por taxonomia de conhecimento (recursos técnicos, memórias de regras/hábitos ou habilidades operacionais).
* `--mode`: Escolhe entre `hybrid` (padrão, fusão RRF), `vector` (apenas semântico k-NN) ou `fts` (apenas texto exato).
* `--decay`: Ativa o decaimento temporal exponencial ponderado (ADR-015) para priorizar notas mais recentes no ranking final.
* `--half-life <dias>`: Tempo de meia-vida da curva em dias (padrão: 30.0 dias).
* `--decay-weight <w>`: Peso do decaimento entre 0.0 (sem efeito) e 1.0 (decaimento máximo) com piso assintótico $(1 - w)$ (padrão: 0.3).
* Realiza a **expansão de grafo** via SQL recursivo para trazer notas estruturalmente conectadas.

**Exemplos:**
```bash
# 1. Busca padrão L1:
./bin/mem.exe search "como funciona o fluxo de autenticacao?"

# 2. Busca cirúrgica L0 (ultracompacta para Agentes e triagem rápida):
./bin/mem.exe search "protocolos" --level l0

# 3. Busca filtrada por categoria (apenas regras e condutas do agente):
./bin/mem.exe search "agentes" --level l0 --category memory

# 4. Busca por habilidades operacionais (deploy, comandos e procedimentos):
./bin/mem.exe search "deploy" --level l0 --category skill

# 5. Busca com decaimento temporal agressivo (meia-vida de 15 dias, peso 0.5):
./bin/mem.exe search --decay --half-life 15 --decay-weight 0.5 "decisoes de arquitetura"

# 6. Filtrando por repositório ou conectando ao PostgreSQL:
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

### 17. `mem impact <node_id> [--depth 2] [--json] [--db <caminho>] [--postgres <url>] [--repo <slug>]`
Analisa preventivamente o **raio de destruição (*blast radius*)** e o fechamento de dependências reversas antes de alterar arquivos de código ou notas arquiteturais:
* **Travessia Reversa em Camadas:** Identifica todos os documentos e serviços que dependem direta ou indiretamente do nó alvo até a profundidade especificada (`--depth`, padrão: 2).
* **Score de Risco Normalizado (0 a 100):** Pondera a quantidade de nós afetados, a severidade semântica das arestas, o decaimento hiperbólico por profundidade ($1/\text{depth}$) e a autoridade estrutural calculada via **PageRank**.
* **Badges de Severidade Semântica:** Diferencia impactos `[CRITICAL]` (contratos de dependência e implementação direta), `[HIGH]`, `[MEDIUM]` (links conceituais) e `[LOW]`.
* **Detecção de Domínios Cruzados:** Relaciona os clusters temáticos de conhecimento (ADR-022) atingidos pelo raio de destruição.
* **Resolução Canônica Tolerante:** O identificador `<node_id>` aceita caminhos relativos de arquivos (`internal/graph/impact.go`), títulos de notas ou slugs parciais.
* **Formato JSON Estruturado:** Com `--json`, exporta métricas e árvore de dependentes pronta para automações e pipelines CI/CD.

**Exemplos:**
```bash
# Analisar impacto de uma nota central de autenticação (profundidade padrão: 2):
./bin/mem.exe impact "concepts/auth.md"

# Avaliar impacto profundo (até 3 níveis de dependências):
./bin/mem.exe impact "internal/db/database.go" --depth 3

# Exportar relatório de blast radius em JSON:
./bin/mem.exe impact "decisions/adr-001.md" --json
```

---

### 18. `mem inspect <node_id> [--full] [--json] [--db <caminho>] [--postgres <url>] [--repo <slug>]`
Inspeciona cirurgicamente qualquer nó da base de conhecimento e do grafo no padrão **Tríptico Cirúrgico em 3 Colunas** (*Zero File Reads*):
* **Coluna 1 (Conexões de Entrada):** Precedentes estruturais, dependentes reversos e callers diretos (`[[wikilinks]]` e referências que apontam para este nó).
* **Coluna 2 (Foco Central):** Metadados primários, autoridade topológica (**PageRank** normalizado), cluster temático (ADR-022) e corpo textual (resumo cirúrgico ou `--full` para conteúdo integral).
* **Coluna 3 (Conexões de Saída):** Relações downstream, contratos de dependência e links epistêmicos derivados.
* `--full`: Exibe o corpo completo do documento sem truncamento.
* `--json`: Emite o tríptico em formato JSON estruturado para consumo de agentes autônomos.

**Exemplos:**
```bash
# Inspecionar nó por caminho relativo ou título:
./bin/mem.exe inspect "concepts/auth.md"

# Inspecionar nó com corpo textual completo:
./bin/mem.exe inspect "internal/graph/impact.go" --full

# Exportar tríptico em formato JSON:
./bin/mem.exe inspect "decisions/adr-001.md" --json
```

---

### 19. `mem install` ou `mem setup` `[--target <cliente>] [--dry-run] [--workspace] [--global] [--force] [--db <caminho>] [--repo <slug>]`
Configuração e **Auto-Wiring Zero-Touch** das ferramentas de IA e clientes MCP suportados (**Claude Desktop**, **Cursor IDE**, **VS Code / GitHub Copilot**, **Windsurf**):
* **Detecção Automática:** Escaneia o sistema operacional (Windows, macOS, Linux) e o diretório de trabalho atual identificando clientes instalados e arquivos de configuração existentes.
* **Injeção Não-Destrutiva:** Preserva integralmente outros servidores MCP já configurados sob a chave `mcpServers` e gera backups automáticos com extensão `.bak` antes de qualquer alteração.
* **Auto-Scoping de Vault:** Detecta `.memory/config.yaml` no diretório atual e auto-injeta os argumentos `--db` e `--repo` correspondentes.
* `--dry-run`: Simula a detecção e exibe os fragmentos JSON de payload sem modificar arquivos em disco.
* `--target <cliente>`: Restringe a instalação a um cliente específico (`claude-desktop`, `cursor`, `vscode`, `windsurf` ou `all`).
* `--workspace`: Limita a configuração exclusivamente ao escopo de workspace local (`.cursor/mcp.json`, `.vscode/mcp.json`, etc.).
* `--global`: Limita a configuração aos diretórios globais do usuário/sistema.
* `--force`: Permite sobrescrever configurações existentes sem confirmação interativa.

**Exemplos:**
```bash
# Auto-detectar todas as ferramentas instaladas e configurar automaticamente:
./bin/mem.exe install

# Simular a configuração sem modificar o disco:
./bin/mem.exe install --dry-run

# Configurar exclusivamente o Cursor IDE:
./bin/mem.exe install --target cursor

# Configurar apenas o workspace atual:
./bin/mem.exe install --workspace

# Alias idêntico de setup:
./bin/mem.exe setup --dry-run
```

---

### 20. `mem path <origem> <destino> [--undirected] [--max-depth 6] [--mode epistemic|hops] [--json] [--db <caminho>] [--postgres <url>] [--repo <slug>]`
Encontra a menor rota e calcula a cadeia de conexões entre dois nós arbitrários no grafo de conhecimento:
* **Ponderação Epistêmica (`--mode epistemic`, padrão):** Pondera arestas inversamente à certeza semântica: conexões explícitas intencionais (`EXTRACTED`, custo $1.0$) têm menor custo do que suposições semânticas (`INFERRED`, custo $1.67$) ou tags (`TAG`, custo $3.33$).
* **Menor Contagem de Saltos (`--mode hops`):** Aplica custo unitário ($1.0$) por aresta, encontrando a rota com o menor número absoluto de nós intermediários.
* **Direcionamento Flexível:** Opera por padrão de forma estritamente causal/direcionada ($A \to B$). A flag `--undirected` ativa a exploração bidirecional ($A \leftrightarrow B$) para descobrir conexões conceituais amplas.
* **Limite de Profundidade:** Restringe a exploração até a profundidade máxima desejada (`--max-depth`, padrão: 6 saltos).
* **Diagrama de Rota ASCII:** Exibe no terminal a sequência de nós, tipos de relação, status e sentido de travessia (`──>` forward ou `<──` reverse).
* **Exportação JSON:** Com a flag `--json`, emite o payload estruturado contendo nós, arestas, saltos e custo acumulado.

**Exemplos:**
```bash
# Descobrir rota epistêmica entre o módulo de autenticação e o redis:
./bin/mem.exe path "concepts/auth.md" "infra/redis.md"

# Encontrar menor caminho bidirecional entre duas notas:
./bin/mem.exe path "ARCHITECTURE" "COMO_FUNCIONA" --undirected --mode hops

# Consultar rota profunda com saída em JSON:
./bin/mem.exe path "decisions/adr-001.md" "decisions/adr-027.md" --max-depth 8 --json
```

---

### 21. `mem pack <nota_raiz> [--depth 2] [--max-tokens 4000] [--direction both] [--out <arquivo>] [--json]`
Empacota um subgrafo de contexto completo e auto-contido centrado em uma nota raiz, consolidando o conteúdo com controle rígido de orçamento de tokens:
* **Orçamento de Tokens (`--max-tokens`, padrão: 4000):** Limita o tamanho total do bundle Markdown gerado, prevenindo estouro de janela de contexto em prompts de IA.
* **Degradação Graciosa em 3 Tiers:**
  - **TierCore (L2):** Nós centrais próximos recebem texto integral.
  - **TierFringe (L0/L1):** Nós periféricos que estourariam o orçamento são resumidos automaticamente através de seus micro-abstracts L0/L1.
  - **TierOmitted:** Nós secundários que não couberem são listados como omitidos, mantendo a rastreabilidade estrutural.
* **Topologia Mermaid:** Incorpora um diagrama visual Mermaid (`graph TD`) representando as relações entre todos os nós incluídos.
* **Gravação em Arquivo (`--out`):** Salva o bundle diretamente no disco, ideal para injeção em prompts ou sub-agentes.
* **Saída JSON (`--json`):** Emite a estrutura de nós, métricas e texto serializados em JSON.

**Exemplos:**
```bash
# Empacotar subgrafo de autenticação com limite de 3000 tokens:
./bin/mem.exe pack "concepts/auth.md" --depth 2 --max-tokens 3000

# Salvar bundle diretamente em arquivo para alimentar um agente:
./bin/mem.exe pack "ARCHITECTURE" --out context_bundle.md

# Obter o subgrafo em formato JSON estruturado:
./bin/mem.exe pack "ARCHITECTURE" --json
```

---

### 22. `mem open <nota_ou_caminho> [--app obsidian|vscode|system] [--line <n>] [--dry-run] [--json]`
Abre diretamente qualquer nota, ADR ou arquivo no editor configurado ou exibe deep links acionáveis:
* **Resolução Flexível de Nó:** Aceita caminho direto no disco, título da nota, identificador canônico no grafo ou wikilink (`[[Nota]]`).
* **Seleção de Aplicativo (`--app`, padrão: definido em `.memory/config.yaml` ou `obsidian`):**
  - `obsidian`: Dispara a URI `obsidian://open?vault=<vault>&file=<rel_path>`.
  - `vscode`: Dispara a URI `vscode://file/<abs_path>[:line]`.
  - `system`: Abre no visualizador padrão do sistema operacional.
* **Foco em Linha (`--line <n>`):** Posiciona o cursor do editor na linha exata indicada.
* **Modo Simulação (`--dry-run`):** Exibe a URI calculada e comando nativo sem disparar processos.
* **Saída Estruturada (`--json`):** Retorna payload com URIs canônicas prontas para scripts.

**Exemplos:**
```bash
# Abrir nota no Obsidian (padrão):
./bin/mem.exe open docs/adr/001-uso-de-sqlite-como-camada-unificada-de-dados.md

# Abrir no VS Code com cursor posicionado na linha 42:
./bin/mem.exe open "Arquitetura Limpa" --app vscode --line 42

# Simulação dry-run para validar o comando a ser executado:
./bin/mem.exe open 001-login --dry-run

# Obter deep links estruturados em JSON:
./bin/mem.exe open 001-login --json
```

---

### 23. `mem drift [--since <faixa>] [--threshold <0.0-1.0>] [--uncovered] [--strict] [--json]`
Analisa o desvio semântico e estrutural entre alterações recentes no histórico do Git e as notas da base de memória:
* **Cruzamento Código-Memória:** Mapeia arquivos de código modificados (`*.go`, `*.py`, `*.ts`, etc.) contra notas e ADRs que os mencionam ou pertencem ao mesmo componente.
* **Cálculo de Drift Score (0 a 100):** Pondera a quantidade de commits posteriores ao `updated_at` da nota, volume de linhas modificadas (+adições/-deleções) e PageRank.
* **Badges de Severidade:** Classifica em `[CRITICAL]` ($\ge 65$), `[HIGH]`, `[MEDIUM]` e `[LOW]`.
* **Detecção de Código Órfão:** Identifica novos módulos ou arquivos de código alterados que não possuem nenhuma nota ou decisão associada no grafo.
* **Flag `--strict` para CI/CD:** Interrompe a execução com código de saída `1` se houver qualquer desvio em nível `CRITICAL`, ideal para GitHub Actions e pré-commits.
* **Saída Estruturada (`--json`):** Exporta métricas, notas defasadas e arquivos órfãos em JSON.

**Exemplos:**
```bash
# Analisar desvio nos últimos 5 commits (padrão):
./bin/mem.exe drift

# Analisar faixa Git customizada e filtrar por threshold mínimo:
./bin/mem.exe drift --since "HEAD~10..HEAD" --threshold 0.30

# Gate rigoroso para pipelines de CI:
./bin/mem.exe drift --strict

# Exportar relatório completo em JSON:
./bin/mem.exe drift --json
```

---


## 🔍 Status e Detecção de Desatualização (`mem status`)

O comando `mem status` realiza uma auditoria instantânea entre os arquivos físicos no disco e os documentos indexados no banco (SQLite ou PostgreSQL), identificando discrepâncias temporais e drift de contexto:

```bash
# Verificar status de sincronização do vault atual:
./bin/mem.exe status

# Exportar status estruturado em JSON para automações ou scripts:
./bin/mem.exe status --json

# Verificar um vault específico ou banco customizado:
./bin/mem.exe status --dir ./notas --db .memory/memory.db
```

### Exemplo de Saída:
```text
=== Status de Integridade e Sincronização do Vault ===
Diretório do Vault: C:\repository\my-memory
Repositório:        FelipeMiiller/my-memory
Arquivos no Disco:  41
Arquivos Indexados: 38
Status:             ⚠️  Desatualizado (Stale Data)
Diferenças:         8 modificado(s)/novo(s), 0 removido(s)

Arquivos Modificados ou Não-Indexados (8):
  • README.md
  • docs/AGENT_INTEGRATION_GUIDE.md
  • docs/CLI_GUIDE.md
  • docs/adr/028-staleness-banners-e-deteccao-de-desatualizacao.md

💡 Recomendação: Execute 'mem index' para sincronizar o grafo e embeddings.
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
2. **Variáveis de Ambiente**: (`MY_MEMORY_PG_URL`, `MY_MEMORY_REPO`, `MY_MEMORY_CENTRAL_VAULT`)
3. **Configuração Local do Repositório**: (`.memory/config.yaml`)
4. **Configuração Global do Usuário**: (`~/.memory/config.yaml`)
5. **Defaults de Código**: (`DefaultConfig()`)

---

## 🏛️ Federação e Cofre Central de Conhecimento (`mem setup`, `mem central`, `mem repos`)

O My-Memory implementa uma arquitetura federada corporativa conectando um **Cofre Central de Conhecimento** (Global Brain sincronizado no Google Drive, OneDrive ou nuvem) e **Cofres de Projeto** (satélites de código local):

### 1. Assistente Interativo Global (`mem setup`)
Executa o assistente guiado para configurar suas preferências em `~/.memory/config.yaml` (fonte soberana de verdade herdada por todos os seus projetos locais):

```bash
# Modo interativo com prompts amigáveis:
mem setup

# Modo não-interativo via linha de comando:
mem setup --central "~/Google Drive/Meu Drive/KnowledgeVault" --engine sqlite --yes
```

Parâmetros suportados:
* `--central <pasta>`: Caminho da pasta do Cofre Central no host.
* `--engine <sqlite|postgres>`: Motor de banco de dados unificado ("ou tudo PostgreSQL, ou tudo SQLite").
* `--postgres-url <url>`: URL de conexão unificada com pgvector.
* `--mcp-port <porta>`: Porta padrão do servidor MCP HTTP/SSE (padrão: 8080).
* `--yes`: Executa em modo silencioso sem confirmação interativa.

### 2. Gestão do Cofre Central (`mem central`)
Audita e inicializa o Cofre Central virgem com estrutura canônica completa e templates para Obsidian:

```bash
# Verificar status de conexão e saúde do cofre central:
mem central status

# Inicializar estrutura canônica em cofre central virgem:
mem central bootstrap
```

O comando de bootstrap cria automaticamente:
* **11 Pastas Canônicas**: `standards/`, `architecture/`, `security/`, `infrastructure/`, `operations/`, `data/`, `ai-agents/`, `domain/`, `guides/`, `templates/` e `staging/`.
* **Templates Prontos**: Modelos de ADR (MADR), RFC, Runbook de Operações e Especificação EARS em `templates/`.
* **MOC Inicial (`README.md`)**: Mapa de conteúdo com `[[wikilinks]]` prontos para navegação no Obsidian Desktop/Mobile.
* **Isolamento de Dados**: Banco SQLite armazenado em `.memory/storage/memory.db` para não poluir as notas visíveis.

### 3. Catálogo Global de Repositórios (`mem repos`)
Exibe todos os projetos locais registrados e rastreados pelo My-Memory:

```bash
mem repos
```

### 4. Zero-Credentials no Git & Identidade Imutável (`repo_id`)
* Cada repositório recebe um identificador criptográfico imutável (`repo_id: repo_<12-hex-chars>`) gerado automaticamente no `mem init`.
* O `.memory/config.yaml` local **nunca** armazena senhas, URLs de banco ou caminhos absolutos do host, garantindo que o versionamento via Git seja 100% limpo e seguro para commits públicos ou equipes corporativas.

### 5. Navegação em Editores e Wikilinks Federados (`mem open`)
Abre notas locais ou notas federadas cross-vault no editor de preferência do usuário (Obsidian, VS Code ou aplicativo padrão do sistema operacional):

```bash
# Abrir nota local no aplicativo padrão ou configurado:
mem open "docs/architecture.md"

# Abrir nota em linha específica no VS Code:
mem open "docs/spec.md" --app vscode --line 42

# Abrir nota canônica do Cofre Central no Obsidian:
mem open "memory://central/standards/oauth2" --app obsidian

# Abrir nota em repositório satélite registrado no catálogo global:
mem open "memory://payments-service/docs/api" --app vscode

# Simular comando de abertura sem executar processo (dry-run):
mem open "memory://central/architecture/pgvector" --dry-run

# Obter links canônicos e metadados de resolução em JSON estruturado:
mem open "memory://central/standards/oauth2" --json
```

Parâmetros suportados:
* `--app <obsidian|vscode|system>`: Define o aplicativo alvo da abertura (padrão: `obsidian`).
* `--line <número>`: Posiciona o cursor na linha indicada (suportado no VS Code).
* `--dry-run`: Simula a abertura e exibe comando de SO e URI gerada sem iniciar o editor.
* `--json`: Retorna objeto JSON com campos `uri`, `vault`, `file`, `path`, `deep_link` e `is_federated: true`.

### 4. Modo de Embedding: Online vs Fallback FTS

O `mem index` e o `mem search --mode hybrid` tentam gerar embeddings via Ollama (`http://localhost:11434`). Se Ollama estiver offline ou a URL configurada em `.memory/config.yaml` for inalcançável:

* **Indexação continua funcionando** — cada chunk é armazenado com vetor zero (`make([]float32, 768)`); FTS5 indexa normalmente e TurboQuant comprime perfeitamente (vetor zero comprime sem perda).
* **Aviso emitido em cada chunk**: `Ollama indisponível (<chunkID>); indexando em modo léxico FTS`.
* **Search híbrido cai automaticamente em FTS-only** — você verá `Executando fallback para busca textual FTS` na saída.
* **Resultados ainda funcionam**: FTS5 + RRF + expansão de grafo retornam resultados relevantes, só perdem o ranqueamento semântico.
* Para voltar ao modo semântico completo: suba `ollama serve`, confirme `nomic-embed-text` disponível e reindexe o vault.

Comportamento garantido pelo commit [`33feb1d`](../../commit/33feb1d) (`fix(index): fallback to FTS chunk indexing when embedding generation is offline`). Documentação completa em [`docs/CENTRAL_VAULT.md` — Seção 4](CENTRAL_VAULT.md#4-modos-de-embedding-online-vs-fallback).




