---
title: "Guia Operacional para Agentes de IA"
category: memory
summary: "Instruções e mapa de navegação operacional para Agentes de IA (Antigravity, Claude Code, Cursor, Windsurf, Copilot) no repositório my-memory."
tags: [agents, ai, operational-guide, memory]
---

# AGENTS.md

Instruções e mapa de navegação operacional para **Agentes de IA** (Antigravity, Claude Code, Cursor, Windsurf, Copilot) atuando no repositório **`my-memory`**.

---

## 🎯 Visão do Projeto

O **My-Memory** é uma ferramenta local em Go que transforma qualquer repositório em um *Repositório Autoconsciente* (*Repository Brain*), unificando:
1. **SQLite Unificado**: Armazenamento relacional, FTS5 (BM25), `sqlite-vec` e nós/arestas de grafo em um único arquivo `memory.db`.
2. **TurboQuant (Google DeepMind, ICLR 2026)**: Compressão vetorial para 4-bits com 32 reflexões de Householder e estimador não-viesado (384 bytes por chunk).
3. **Obsidian Graph**: Parsing de `[[wikilinks]]` e `#tags` gerando conexões reais sem alucinações.
4. **SQL Recursivo (`WITH RECURSIVE`)**: Expansão de vizinhos em grafo direto pelo SQLite.
5. **PostgreSQL + pgvector**: Persistência híbrida opcional de alta escala para repositórios distribuídos e ambientes multi-tenant.

---

## 🧠 Inicialização do Vault de Conhecimento (`.memory/`)

No início de qualquer sessão ou tarefa no repositório, o Agente de IA **DEVE** verificar se o vault local está inicializado:

1. **Detecção e Criação Automática**:
   - Verifique se a pasta `.memory/` ou o arquivo `.memory/config.yaml` existe no repositório.
   - Se **NÃO existir**:
     - Execute imediatamente:
       ```bash
       mem init
       ```
       *(ou execute `go run ./cmd/mem init` caso o binário compilado não esteja no PATH)*.
     - O comando criará a estrutura padrão com proteção de segurança:
       - `.memory/config.yaml`: Escopo declarativo pré-configurado para pastas `docs/`, `specs/`, `.specs/` e `README.md`, com isolamento do SQLite em `.memory/memory.db`.
       - `.memory/.gitignore`: Garante que `.env` e arquivos `.db` nunca sejam commitados.
       - `.memory/.env.example`: Modelo de configuração com variáveis para PostgreSQL.
2. **Conexão com PostgreSQL (pgvector)**:
   - **Source-of-truth (ADR-040)**: configuração canônica vive em `~/.memory/config.yaml` (global). Seção `storage:` com `engine: postgres` + `postgres_url` ativa o central vault.
   - **Override local opcional**: o agente pode copiar `.memory/.env.example` para `.memory/.env` para configurar Postgres via env vars:
     ```env
     MY_MEMORY_PG_URL=postgres://postgres:postgres@localhost:5432/my_memory?sslmode=disable
     ```
   - O `my-memory` detecta automaticamente (global > local > env) e direciona todas as operações (índice, busca, MCP) para o PostgreSQL sem parâmetros manuais.
   - **Override consciente**: flag `--storage=postgres` força Postgres mesmo sem config; `--storage=sqlite` força SQLite local (útil pra CI/sandbox). Ver `docs/CLI_GUIDE.md` seção "Seleção de Storage".
3. **Indexação Inicial**:
   - Caso o vault tenha acabado de ser inicializado, execute a indexação:
     ```bash
     mem index
     ```

---

## 📌 Decisões Arquiteturais Vigentes (`docs/adr/`)
Consulte estes registros antes de sugerir mudanças estruturais:
* **ADR-001**: Uso de SQLite como Camada Unificada de Dados.
* **ADR-002**: Adoção de Go como Linguagem Principal de Implementação.
* **ADR-003**: Compressão Vetorial de 4-bit via TurboQuant.
* **ADR-004**: Modelagem e Travessia de Grafo com SQL Recursivo (CTEs).
* **ADR-005**: Markdown com [[Wikilinks]] como Entrada e Grafo Humano.
* **ADR-006**: Integração com Agentes de IA via Model Context Protocol (MCP).
* **ADR-016**: Configuração Declarativa e Auto-Scoping de Vault (`.memory/config.yaml`).
* **ADR-035**: Embedder Embutido com Fallback ONNX MiniLM (Proposed).
* **ADR-036**: Consolidação Pós-Release v1.3.0 e Roadmap v1.4.0.
* **ADR-037**: Rewrite do Viewer com Vite + Vanilla TypeScript (deferred).
* **ADR-038**: Viewer com Site Único e Dataset Fixo (B3 do ADR-036 revertido).

## 🤖 Habilidades de IA Disponíveis (`.agents/skills/`)

Antes de realizar ações complexas, consulte e siga as skills instaladas:

| Skill | Localização | Quando o Agente DEVE usar |
| :--- | :--- | :--- |
| **`my-memory`** ⭐ canônica | [`.agents/skills/my-memory/SKILL.md`](.agents/skills/my-memory/SKILL.md) | **Primeiro a ser carregada em qualquer sessão do projeto.** Ensina a instalar, configurar, usar a CLI, integrar via MCP e **criar documentos** no vault. Fonte única de verdade para uso do My-Memory. |
| **`memory-md`** | [`.agents/skills/memory-md/SKILL.md`](.agents/skills/memory-md/SKILL.md) | Referência canônica para formatação `.md` (frontmatter, wikilinks, tags, callouts). Usar SEMPRE que criar/editar uma nota — a skill `my-memory` aponta para esta para markup detalhado. |
| **`create-adr`** | [`.agents/skills/create-adr/SKILL.md`](.agents/skills/create-adr/SKILL.md) | Ao tomar decisões arquiteturais significativas, registrar escolhas técnicas ou quando o usuário pedir para documentar o motivo de uma escolha. Formato obrigatório: **MADR**. |
| **`tlc-spec-driven`** | [`.agents/skills/tlc-spec-driven/SKILL.md`](.agents/skills/tlc-spec-driven/SKILL.md) | Ao planejar ou implementar features não-triviais. Seguir as 4 fases adaptativas (*Specify → Design → Tasks → Execute*), critérios em notação EARS, commits atômicos e gates determinísticos em Python. |

---

## 📚 Mapa da Documentação e Decisões

### Documentação Técnica (`docs/`)
* **[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)**: Schemas DDL, tabelas relacionais, índices FTS5 e consultas recursivas em SQL.
* **[`docs/TURBOQUANT.md`](docs/TURBOQUANT.md)**: Teoria matemática da quantização de 4-bit, rotações ortogonais e tabelas de compressão.
* **[`docs/CLI_GUIDE.md`](docs/CLI_GUIDE.md)**: Manual dos comandos `mem index`, `mem search` e modo `-tq`.
* **[`docs/REPOSITORY_BRAIN.md`](docs/REPOSITORY_BRAIN.md)**: Arquitetura de integração com IAs via Model Context Protocol (MCP).
* **[`docs/README.md`](docs/README.md)**: Índice unificado de documentação.

### Decisões Arquiteturais Vigentes (`docs/adr/`)
Consulte estes registros antes de sugerir mudanças estruturais:
* **ADR-001**: Uso de SQLite como Camada Unificada de Dados.
* **ADR-002**: Adoção de Go como Linguagem Principal de Implementação.
* **ADR-003**: Compressão Vetorial de 4-bit via TurboQuant.
* **ADR-004**: Modelagem e Travessia de Grafo com SQL Recursivo (CTEs).
* **ADR-005**: Markdown com [[Wikilinks]] como Entrada e Grafo Humano.
* **ADR-006**: Integração com Agentes de IA via Model Context Protocol (MCP).
* **ADR-016**: Configuração Declarativa e Auto-Scoping de Vault (`.memory/config.yaml`).

---

## 🛠 Comandos Operacionais para o Agente

### Testes
```bash
# Executar todos os testes unitários
go test -v ./internal/...

# Executar com relatório de cobertura
go test -v -cover ./internal/parser/... ./internal/turboquant/...

# Executar suíte completa
go test -count=1 ./...
```

### Formatação e Estilo
```bash
# Verificar arquivos fora do padrão (deve retornar vazio)
gofmt -l .

# Aplicar formatação oficial Go automaticamente
gofmt -w .
```

### Compilação
```bash
# Compilar o executável CLI
go build -v -o bin/mem.exe ./cmd/mem
```

---

## 📌 Regras de Conduta para Agentes de IA

1. **Codificação de Arquivos:** Todo arquivo criado ou modificado deve ser estritamente **UTF-8 sem BOM**.
2. **Commits Atômicos:** Utilize a convenção *Conventional Commits* (`feat:`, `fix:`, `docs:`, `chore:`, `refactor:`).
3. **Não altere convenções sem ADR:** Qualquer mudança de banco de dados, biblioteca de vetores ou arquitetura exige a criação de um novo ADR na pasta `docs/adr/`.
4. **Execução Obrigatória de Testes após Cada Tarefa:** SEMPRE que finalizar uma tarefa, alteração de código ou refatoração, o agente DEVE OBRIGATORIAMENTE executar os testes (`go test -v ./...` ou os pacotes impactados). Nenhuma tarefa é considerada pronta nem pode ser commitada sem que os testes passem com 100% de aprovação.
5. **Verificação de Inicialização da Memória:** Ao iniciar qualquer trabalho neste repositório, o agente DEVE checar se `.memory/config.yaml` existe. Caso não exista, deve executar `mem init` antes de qualquer outra tarefa.
6. **Taxonomia e Carregamento Progressivo (L0/L1/L2):** Ao criar ou documentar notas, inclua preferencialmente no frontmatter a taxonomia `category: resource | memory | skill` e `summary: <resumo>` para viabilizar indexação em camadas e recuperação econômica L0.
7. **Atualização Mandatória de Documentação, Specs e README ao Subir PR:** Antes de finalizar uma funcionalidade, abrir PR ou submeter alterações para o repositório remoto (`git push`), o agente DEVE OBRIGATORIAMENTE atualizar todos os artefatos de documentação impactados:
   - `README.md`: Atualizar tabela de capacidades, novos comandos da CLI, novas ferramentas MCP e exemplos de uso.
   - `docs/`: Atualizar `docs/README.md`, `docs/CLI_GUIDE.md`, `docs/AGENT_INTEGRATION_GUIDE.md`, `docs/ARCHITECTURE.md` e registrar o respectivo ADR em `docs/adr/`.
   - `.specs/`: Concluir especificação (`spec.md`), tarefas (`tasks.md`), validação com evidências (`validation.md`) e atualizar o snapshot em `STATE.md`.
   - **Reindexação e Checagem de Desvio:** Executar `mem index` e verificar com `mem drift` que nenhum código novo permaneceu órfão e que a documentação está sincronizada antes do push (consulte [.agents/rules/always-update-docs-and-specs-on-pr.md](.agents/rules/always-update-docs-and-specs-on-pr.md)).
8. **Quality Gate de Fim de Tarefa:** Após qualquer entrega (spec concluída, task fechada, fix aplicado, viewer modificado), o agente DEVE rodar silenciosamente o gate completo antes de declarar "pronto" (consulte [.agents/rules/always-quality-gate.md](.agents/rules/always-quality-gate.md)). Itens do gate: `go test -count=1 ./...`, `go build`, `gofmt -l .`, benchmarks onde faltarem, smoke test do binário, validação visual via Playwright (se viewer mudou), status do pipeline CI, `git status` sem untracked de teste, e checagem cruzada com ADR-036. Erros são corrigidos imediatamente; só viram ADR quando a decisão for arquitetural.