# Feature: federated-central-vault-and-repo-identity

## Problem Statement
Atualmente, o `my-memory` opera prioritariamente sobre o escopo de um único repositório local (isolado em seu arquivo `.memory/memory.db`) ou em um PostgreSQL compartilhado onde repositórios são identificados apenas pelo slug Git.

Em organizações de engenharia e ecossistemas com múltiplos microsserviços, o conhecimento técnico divide-se naturalmente em dois níveis:
1. **Global Brain (Cofre Central)**: Padrões arquiteturais transversais (RFCs corporativas), convenções de API REST/gRPC, políticas de segurança e auth (OAuth2, JWT, RBAC), práticas de DevOps/IaC, glossário de domínio e lições aprendidas. Esse cofre reside comumente em uma pasta sincronizada na nuvem (Google Drive, OneDrive, iCloud) acessada diretamente pelo usuário no Obsidian Desktop e Mobile.
2. **Project Brains (Repositórios Satélites)**: Código-fonte, ADRs locais (`docs/adr/`), especificações de features (`.specs/`) e notas táticas de cada repositório.

A tentativa de vincular esses dois mundos através de Git Submodules gera atrito operacional severo (poluição de `git status`, merge conflicts em detached HEAD, incompatibilidade no Obsidian Mobile). Além disso, a ausência de um identificador imutável de repositório cria fragilidade quando pastas são renomeadas ou quando múltiplos repositórios compartilham uma instância de banco de dados.

A funcionalidade **Federated Central Vault & Repository Identity** resolve esses problemas ao:
- Gerar e garantir um `repo_id` criptográfico único e imutável para cada projeto em seu `.memory/config.yaml`.
- Permitir referenciar declarativamente o cofre central (`central_vault.path`) via caminho de sincronizador (Google Drive/OneDrive) ou variável de ambiente (`${MY_MEMORY_CENTRAL_VAULT}`) sem clonar arquivos no Git do projeto.
- Auto-inicializar (*Zero-Touch Bootstrap*) a árvore completa de pastas de engenharia no cofre central caso ele esteja virgem.
- Isolar os bancos SQLite de forma limpa em `<central_path>/.memory/storage/memory.db` e unificar buscas federadas via Reciprocal Rank Fusion (RRF).

---

## Out of Scope
- Implementação de clientes de sincronização de rede proprietários (a sincronização física de arquivos é delegada ao Google Drive, OneDrive, Syncthing ou Git).
- Modificação destrutiva ou escrita automática em notas do cofre central a partir de comandos locais do projeto (o cofre central é tratado com semântica estrita de leitura e proteção contra escrita acidental por agentes).
- Replicação de logs binários em tempo real entre SQLite e PostgreSQL (o Markdown continua sendo a fonte soberana da verdade, e os bancos atuam como projeções derivadas reconstruíveis).

---

## Assumptions & Open Questions

| Assumption / Question | Chosen default | Rationale |
| :--- | :--- | :--- |
| Formato do Identificador Imutável | `repo_<12-hex-chars>` (ex: `repo_a1b2c3d4e5f6`) gerado na inicialização | Compacto, legível em URLs e seguro como identificador de namespace no SQLite e PostgreSQL. |
| Identificador do Vault Central | `repo_central` fixo e reservado | Facilita convenções e referências canônicas universais sem colisões com projetos satélites. |
| Localização do Banco Central SQLite | `<central_path>/.memory/storage/memory.db` | Mantém os dados binários do My-Memory isolados sem poluir as notas visíveis no Obsidian. |
| Semântica de Acesso ao Central Vault | Somente-leitura (`read_only: true`) a partir dos repositórios de código | Impede que agentes de IA trabalhando em um projeto alterem acidentalmente diretrizes corporativas centrais. |
| Resiliência a Caminhos Ausentes | Aviso não-bloqueante (`[WARN]`) e fallback para o contexto local | Permite que o projeto seja clonado e compilado em qualquer máquina sem quebrar caso o Google Drive não esteja montado. |

Open questions: none (todas as decisões foram alinhadas e acordadas com o usuário).

---

## User Stories

- **US1**: Como engenheiro de software, quero definir `central_vault.path: "G:/Meu Drive/KnowledgeBase"` no `config.yaml` do meu projeto para que meus agentes de IA consultem os padrões da empresa sem sujar o Git do repositório com submódulos.
- **US2**: Como desenvolvedor que utiliza o Obsidian, quero que o cofre central virgem no Google Drive seja inicializado automaticamente com pastas estruturadas (`standards/`, `architecture/`, `security/`, `templates/`) e um MOC inicial com wikilinks navegáveis.
- **US3**: Como arquiteto corporativo, quero que cada repositório possua um `repo_id` imutável para isolar seus artefatos no PostgreSQL ou SQLite sem risco de colisão quando projetos forem renomeados.
- **US4**: Como agente de IA via MCP ou usuário no terminal (`mem search`), quero fazer perguntas como "qual o padrão de autenticação?" e receber respostas que unem o padrão corporativo central e a implementação local via RRF.

---

## Requirements (EARS Notation)

### FED-01: Identidade Imutável de Repositório (`repo_id`)
- **UBIQUITOUS**: The system SHALL assign an immutable unique identifier `repo_id` (format `repo_<12-hex-chars>`) to each repository in `.memory/config.yaml` upon initialization.
- **WHERE**: Where `repo_id` is already defined in `.memory/config.yaml`, the system SHALL preserve it without alteration across all subsequent CLI and MCP invocations.
- **UBIQUITOUS**: The system SHALL designate `repo_central` as the reserved identity for the central knowledge base.

### FED-02: Convenção Definitiva `.memory/` e Zero Credentials no Git
- **UBIQUITOUS**: The system SHALL establish `.memory/` in the repository root as the definitive, immutable control directory without requiring external path configuration.
- **UBIQUITOUS**: The system SHALL exclude database credentials, connection URLs, and host storage paths from the local repository `.memory/config.yaml` to ensure zero secrets leak to version control.
- **WHERE**: Where `central_vault.read_only` is true or omitted, the system SHALL enforce read-only semantics for the central knowledge base from the local repository context.
- **WHERE**: Where the declared central vault path does not exist on the host filesystem, the system SHALL issue a non-blocking warning and gracefully restrict queries to local repository context.

### FED-03: Auto-Bootstrap do Vault Central Virgem
- **WHEN**: When a declared `central_vault.path` refers to a non-existent or empty folder lacking `.memory/`, the system SHALL automatically create the directory tree including `standards`, `architecture`, `security`, `infrastructure`, `operations`, `data`, `ai-agents`, `domain`, `guides`, `templates`, and `staging`.
- **UBIQUITOUS**: The system SHALL generate standardized template notes in `templates/` (ADR, RFC, Runbook, Spec) and an entrypoint `README.md` Map of Content (MOC) with clickable Obsidian `[[wikilinks]]`.
- **UBIQUITOUS**: The system SHALL initialize the central storage metadata in `<central_path>/.memory/config.yaml` with identity `repo_central`.

### FED-04: Persistência Unificada (Banco Único no Postgres e Isolamento no SQLite)
- **UBIQUITOUS**: The system SHALL treat the global configuration `~/.memory/config.yaml` as the authoritative source of truth for the storage engine choice ("all PostgreSQL or all SQLite") across all federated vaults.
- **WHERE**: Where the storage engine is PostgreSQL, the system SHALL connect to the unified database using a single connection pool and strictly isolate records using the repository identifier `repo_id` (and `repo_central` for the central knowledge base).
- **WHERE**: Where the storage engine is SQLite, the system SHALL store the local database in `.memory/memory.db` and the central database strictly in `<central_path>/.memory/storage/memory.db`.

### FED-05: Busca Híbrida Federada (Local + Central)
- **WHEN**: When a search is performed via `mem search` or MCP tool `memory_search` in a repository with a linked central vault, the system SHALL query both the local project database and the central vault database.
- **UBIQUITOUS**: The system SHALL merge results across databases using Reciprocal Rank Fusion (RRF) and label each match with its provenance origin (`[local]` vs `[central]`).

### FED-06: Configuração Global Soberana (`~/.memory/config.yaml`) e Assistente `mem setup`
- **UBIQUITOUS**: The system SHALL support reading a user-level global configuration file at `~/.memory/config.yaml` to define the central vault path and storage credentials once.
- **WHERE**: Where local repository configuration omits storage or central vault settings, the system SHALL inherit defaults from the global configuration file.
- **WHEN**: When `mem setup` is executed, the system SHALL provide an interactive command to record central vault and database preferences into `~/.memory/config.yaml`.

### FED-07: Catálogo de Repositórios no Global Config e Descoberta MCP
- **UBIQUITOUS**: The system SHALL maintain a repository catalog (`repositories: [{id, path, name}]`) in `~/.memory/config.yaml`.
- **WHEN**: When `mem init` or `mem setup` is executed in a repository, the system SHALL register or update the repository entry in `~/.memory/config.yaml`.
- **WHEN**: When the MCP server initializes, the system SHALL read `~/.memory/config.yaml` to discover the central vault and registered repositories regardless of the working directory.

---

## Requirement Traceability

| Requirement ID | Description | Status |
| :--- | :--- | :--- |
| FED-01 | Identidade Imutável de Repositório (`repo_id`) | verified |
| FED-02 | Configuração Declarativa de Central Vault Vinculado | verified |
| FED-03 | Auto-Bootstrap do Vault Central Virgem | verified |
| FED-04 | Isolamento Limpo de Persistência (SQLite e PostgreSQL) | verified |
| FED-05 | Busca Híbrida Federada (Local + Central) | verified |
| FED-06 | Configuração Global em Cascata e Assistente `mem setup` | verified |
| FED-07 | Catálogo de Repositórios e Descoberta Global no MCP | verified |
