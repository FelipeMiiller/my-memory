# ADR-033: Arquitetura Federada de Vault Central e Identidade Imutável de Repositório (`repo_id`)

- **Date**: 2026-09-15
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Antigravity
- **Tags**: federation, central-vault, repo-id, google-drive, multi-vault, postgresql, sqlite, bootstrap, devx

## Context and Problem Statement

À medida que múltiplos projetos e microsserviços utilizam o `my-memory`, surge a necessidade de gerenciar o conhecimento em dois níveis distintos:
1. **Conhecimento Transversal e Perene (*Global Brain*)**: Padrões de arquitetura corporativa, normas de API REST/gRPC, diretrizes de segurança (OAuth2, JWT), políticas de DevOps, glossário de domínio e lições aprendidas.
2. **Conhecimento Específico de Projeto (*Local Brain*)**: Código-fonte, ADRs locais (`docs/adr/`), especificações de features (`.specs/`) e notas táticas de cada repositório.

A tentativa ingênua de sincronizar esse conhecimento via *Git Submodules* ou cópia física de arquivos para dentro dos repositórios de código gera atrito operacional severo: polui o `git status`, provoca merge conflicts em *detached HEAD*, confunde clientes móveis do Obsidian e sobrecarrega os repositórios locais com arquivos desnecessários.

Adicionalmente, se múltiplos repositórios compartilham uma instância de PostgreSQL ou se sincronizam com pastas em nuvem (Google Drive / OneDrive), a identificação por simples nomes de pasta é frágil, sujeita a colisões e quebras quando repositórios são renomeados ou movidos.

## Decision Drivers

- **Zero Poluição de Git**: Nenhum arquivo do cofre central deve ser clonado ou commitado dentro dos repositórios de código locais, eliminando atritos de Git Submodules e merge conflicts.
- **Identidade Criptográfica Imutável (`repo_id`)**: Cada repositório deve receber um identificador único, permanente e imutável gerado no `config.yaml` (`repo_<12-hex-chars>`), utilizado para isolamento em bancos de dados.
- **Armazenamento Central em Nuvem (Google Drive / OneDrive / Obsidian)**: O Vault Central deve residir em uma pasta sincronizada pelo usuário, permitindo navegação fluida no Obsidian Desktop e Mobile.
- **Isolamento e Limpeza no Armazenamento**: Arquivos de banco de dados SQLite central não devem poluir a raiz de notas Markdown, residindo isoladamente em `<central_path>/.memory/storage/memory.db`.
- **Regra de Coerência de Motor ("Ou tudo PostgreSQL, ou tudo SQLite")**: A escolha do motor é configurada centralmente no arquivo global (`~/.memory/config.yaml`), aplicando-se uniformemente a todos os repositórios e ao cofre central, eliminando repetição de credenciais:
  - Se configurado **PostgreSQL**: cada repositório opera em seu próprio banco dedicado batizado a partir do seu `repo_id` (ex: `my_memory_<repo_id>`), e o cofre central opera no banco `my_memory_central`.
  - Se configurado **SQLite**: cada repositório opera com seu banco local isolado (`.memory/memory.db`) e o cofre central opera em `<central_path>/.memory/storage/memory.db`.
- **Descoberta Não-Invasiva e Global pelo Servidor MCP**: O servidor MCP deve ler o arquivo global `~/.memory/config.yaml` no diretório home do usuário para descobrir instantaneamente a localização do cofre central, o motor de banco e o catálogo de repositórios conhecidos, mesmo quando iniciado fora da raiz de um repositório Git.
- **Zero-Touch Auto-Bootstrap**: Se o usuário referenciar uma pasta central virgem, o `my-memory` inicializa automaticamente a árvore de 11 diretórios canônicos de engenharia (`standards/`, `architecture/`, `security/`, `infrastructure/`, `operations/`, `data/`, `ai-agents/`, `domain/`, `guides/`, `templates/`, `staging/`), com templates padrão e MOC navegável.

## Considered Options

1. **Git Submodules / Git Subtree** (Descartado: alto atrito de manutenção, conflitos de branch e incompatibilidade no Obsidian Mobile).
2. **Duplicação Manual de Arquivos** (Descartado: cria divergência de versões e impossibilita atualizações contínuas).
3. **Vault Central Vinculado com Identidade Imutável (`repo_id`), Configuração Global em Cascata e Auto-Bootstrap** (*Opção Escolhida*).

## Decision Outcome

Adotou-se a **Opção 3**:

```mermaid
graph TD
    subgraph UserHome["🏠 Diretório Home do Usuário (~/.memory/config.yaml)"]
        GC_Config["Configuração Global:\n- central_vault.path (Google Drive/OneDrive)\n- storage: postgres (ou sqlite em tudo)\n- mcp: port 8080\n- repositories: [{id, path, name}]"]
    end

    subgraph CentralStorage["☁️ Google Drive / OneDrive (Central Vault)"]
        CV_Dirs["standards/, architecture/, security/, infrastructure/\noperations/, data/, ai-agents/, domain/, guides/, templates/, staging/"]
        CV_DB[".memory/storage/memory.db (SQLite Central)\nou Banco 'my_memory_central' (PostgreSQL)"]
    end

    subgraph ProjectRepo["💻 Repositório Local (c:/meu-projeto)"]
        PR_Config[".memory/config.yaml\nrepo_id: repo_xxx (imutável)\nrepository: empresa/projeto"]
        PR_Docs["docs/adr/, .specs/, README.md"]
        PR_DB[".memory/memory.db (SQLite Local)\nou Banco 'my_memory_repo_xxx' (PostgreSQL)"]
    end

    subgraph MCPServer["🤖 Servidor MCP (Cursor, Claude, Copilot, Antigravity)"]
        MCP_Core["Descoberta Inteligente:\n1. Lê ~/.memory/config.yaml\n2. Localiza Central Vault\n3. Conecta no Repositório Atual\n4. Busca Federada Híbrida (RRF)"]
    end

    GC_Config -.->|"Herança e Catálogo"| ProjectRepo
    GC_Config -.->|"Localização Global"| CentralStorage
    MCPServer -->|"Lê Config Global"| GC_Config
    MCPServer -->|"Consulta Local"| PR_DB
    MCPServer -->|"Consulta Central"| CV_DB
```

### 1. Identidade Criptográfica Imutável e Zero Credentials no Git
Na primeira execução de `mem init`, o sistema gera deterministicamente `repo_id` no formato `repo_<12-hex-chars>` (ex: `repo_a1b2c3d4e5f6`), persistido em `.memory/config.yaml`.
A pasta `.memory/` na raiz do projeto é a **convenção fixa e definitiva**.
Para garantir segurança absoluta e portabilidade no Git, o `.memory/config.yaml` local **NUNCA** contém configurações de banco de dados (`storage`), credenciais ou URLs de conexão:
```yaml
# .memory/config.yaml (no repositório de código - 100% limpo e seguro para commit)
version: 1
repo_id: "repo_a1b2c3d4e5f6"       # Identidade imutável gerada na inicialização
repository: "FelipeMiiller/my-memory"
vault_name: "My Memory Vault"

include:
  - "**/*.md"
exclude:
  - ".git/**"
  - ".memory/**"
```
Nenhuma senha, URL de PostgreSQL ou caminho pessoal do host vaza para o repositório remoto.

### 2. Configuração Global Soberana (`~/.memory/config.yaml`)
Localizada em `$HOME/.memory/config.yaml` (Linux/macOS) ou `%USERPROFILE%\.memory\config.yaml` (Windows).
É a **fonte única da verdade de infraestrutura** para toda a máquina:
```yaml
version: 1

# Caminho para a pasta sincronizada no Google Drive ou OneDrive
central_vault:
  path: "~/Google Drive/Meu Drive/KnowledgeVault"
  read_only: false

# Motor de dados global: "ou tudo PostgreSQL, ou tudo SQLite"
storage:
  engine: "postgres"
  postgres_url: "postgres://postgres:secret@localhost:5432/my_memory?sslmode=disable"

mcp:
  port: 8080

# Catálogo dinâmico de repositórios conhecidos na máquina
repositories:
  - id: "repo_a1b2c3d4e5f6"
    path: "C:/repository/my-memory"
    name: "FelipeMiiller/my-memory"
```

### 3. Topologia e Isolamento Físico de Banco de Dados
- **Modo PostgreSQL ("PostgreSQL em tudo - Conexão Estável")**:
  - Para evitar dezenas de pools de conexão e conexões aleatórias no servidor PostgreSQL, o My-Memory utiliza o **mesmo banco de dados estável** (ex: `my_memory`).
  - O isolamento completo entre projetos e o cofre central é garantido deterministicamente pela coluna `repo_id` (com o valor reservado `repo_central` para o cofre central).
  - Um único pool de conexões de alto desempenho atende o MCP e as buscas federadas, eliminando overhead de rede.
- **Modo SQLite ("SQLite em tudo")**:
  - Repositório local: banco isolado na pasta padrão `.memory/memory.db`.
  - Cofre central: banco isolado exclusivamente na subpasta `<central_path>/.memory/storage/memory.db`, blindando a raiz do vault contra arquivos binários `.db`.

### 4. Auto-Bootstrap Estruturado do Cofre Central
Ao referenciar uma pasta vazia em `central_vault.path`, o `my-memory` cria automaticamente:
- 11 pastas canônicas: `standards/`, `architecture/`, `security/`, `infrastructure/`, `operations/`, `data/`, `ai-agents/`, `domain/`, `guides/`, `templates/`, `staging/`.
- Templates canônicos em `templates/`: `adr.md`, `rfc.md`, `runbook.md`, `spec.md`.
- `README.md` raiz atuando como Mapa de Conteúdo (MOC) com `[[wikilinks]]` clicáveis no Obsidian.
- Isolamento `.memory/config.yaml` (`repo_id: repo_central`) e `.memory/storage/`.

### 5. Descoberta Inteligente no Servidor MCP
Ao inicializar, o servidor MCP:
1. Carrega `~/.memory/config.yaml` para obter as credenciais de banco e a localização do cofre central.
2. Identifica se está rodando dentro de um repositório cadastrado.
3. Se estiver em um projeto: executa busca federada unindo o repositório local e o cofre central via **Reciprocal Rank Fusion (RRF)** com anotação explícita de proveniência (`[local]` vs `[central]`).
4. Se estiver fora de qualquer repositório (ex: aberto em pasta neutra): serve o cofre central de imediato e expõe o catálogo de repositórios disponíveis.

## Positive Consequences

- **Repositórios 100% Limpos e Seguros**: Zero credenciais de banco e zero submódulos no Git dos projetos.
- **Configuração Centralizada**: Infraestrutura definida uma única vez no home do usuário governa todos os projetos sem redundância.
- **Conexão Estável no PostgreSQL**: Um único banco estável sem proliferação de conexões ou pools aleatórios, com separação segura por `repo_id`.
- **Experiência Perfeita no Obsidian**: O cofre central no Google Drive/OneDrive mantém apenas Markdown puro visível, com banco isolado em `.memory/storage/`.
- **Descoberta Instantânea no MCP**: Qualquer agente de IA (Cursor, Claude, Copilot, Antigravity) tem acesso imediato à memória global e local.

## Negative Consequences

- **Dependência de Montagem para SQLite**: No modo SQLite, a pasta do Google Drive/OneDrive precisa estar montada na máquina para que a busca federada acesse o cofre central (com fallback transparente e não-bloqueante para o cofre local caso a pasta esteja ausente).

## Implementation References

- `internal/federation/bootstrap.go`: Motor de auto-bootstrap estruturado do cofre central com 11 pastas canônicas e templates Obsidian.
- `internal/federation/search.go`: Motor de busca federada híbrida unindo fontes locais e centrais via Reciprocal Rank Fusion (RRF).
- `internal/config/config.go`: Identidade `repo_id`, herança em cascata (`LoadCascadingConfig`) e catálogo global de repositórios.
- `cmd/mem/setup.go`: Assistente interativo `mem setup`, comandos `mem central` e `mem repos`.
