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
   - Se o ambiente local possuir PostgreSQL ativo e o repositório necessitar de persistência em pgvector, o agente pode copiar `.memory/.env.example` para `.memory/.env`:
     ```env
     MY_MEMORY_PG_URL=postgres://postgres:postgres@localhost:5432/my_memory?sslmode=disable
     ```
   - O `my-memory` detectará automaticamente o `.memory/.env` e direcionará todas as operações (índice, busca, MCP) para o PostgreSQL sem necessidade de parâmetros manuais.
3. **Indexação Inicial**:
   - Caso o vault tenha acabado de ser inicializado, execute a indexação:
     ```bash
     mem index
     ```

---

## 🤖 Habilidades de IA Disponíveis (`.agents/skills/`)

Antes de realizar ações complexas, consulte e siga as skills instaladas:

| Skill | Localização | Quando o Agente DEVE usar |
| :--- | :--- | :--- |
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