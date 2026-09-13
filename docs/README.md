# Índice Geral: Documentação & Agent Skills

Este documento é o **mapa central de conhecimento** do projeto **My-Memory**. Ele conecta a documentação técnica, as decisões de arquitetura (ADRs) e as habilidades de IA (*Agent Skills*) instaladas para desenvolvedores e agentes autônomos.

---

## 🗺 1. Mapa da Documentação Técnica (`docs/`)

| Documento | Foco | Descrição |
| :--- | :--- | :--- |
| **[`docs/ARCHITECTURE.md`](ARCHITECTURE.md)** | Arquitetura | Diagrama de fluxo de dados, DDL do SQLite, índices FTS5, vetores e consultas recursivas em grafo. |
| **[`docs/TURBOQUANT.md`](TURBOQUANT.md)** | Matemática & Algoritmo | Teoria da quantização vetorial de 4-bit (Google DeepMind, ICLR 2026), rotações de Householder e produto escalar não-viesado. |
| **[`docs/CLI_GUIDE.md`](CLI_GUIDE.md)** | Operação | Manual prático de comandos da CLI (`mem index`, `mem search` e modo `-tq`). |
| **[`docs/REPOSITORY_BRAIN.md`](REPOSITORY_BRAIN.md)** | Integração com IA | Como utilizar o `my-memory` como memória de contexto dentro de projetos via **Model Context Protocol (MCP)**. |
| **[`docs/REFERENCES.md`](REFERENCES.md)** | Arte Prévia & Referências | Referências técnicas e ecossistemas que inspiram e refinam o projeto (`graphify`, `ai-memory`, `obsidian-skills`). |

---

## 📋 2. Registros de Decisão de Arquitetura (`docs/adr/`)

Decisões registradas no formato padronizado **MADR**:

* **[`docs/adr/README.md`](adr/README.md)** — Índice consolidado de ADRs
* **[`ADR-001`](adr/001-uso-de-sqlite-como-camada-unificada-de-dados.md)** — Uso de SQLite como Camada Unificada de Dados
* **[`ADR-002`](adr/002-adocao-de-go-como-linguagem-principal.md)** — Adoção de Go como Linguagem Principal de Implementação
* **[`ADR-003`](adr/003-compressao-vetorial-de-4-bit-via-turboquant.md)** — Compressão Vetorial de 4-bit via TurboQuant
* **[`ADR-004`](adr/004-modelagem-de-grafo-com-recursive-ctes.md)** — Modelagem e Travessia de Grafo com SQL Recursivo (CTEs)
* **[`ADR-005`](adr/005-markdown-com-wikilinks-como-fonte-de-verdade.md)** — Markdown e [[Wikilinks]] como Entrada e Grafo Humano
* **[`ADR-006`](adr/006-integracao-com-agentes-de-ia-via-mcp.md)** — Integração com Agentes de IA via Model Context Protocol (MCP)
* **[`ADR-007`](adr/007-suporte-opcional-a-postgresql-com-pgvector-e-multi-repositorio.md)** — Suporte Opcional a PostgreSQL com pgvector e Referência Multi-Repositório
* **[`ADR-008`](adr/008-interoperabilidade-obsidian-flavored-markdown-e-json-canvas.md)** — Interoperabilidade com Obsidian Flavored Markdown e JSON Canvas 1.0

---

## 🤖 3. Habilidades de IA Instaladas (`.agents/skills/`)

Habilidades de engenharia de software configuradas no repositório através do `@tech-leads-club/agent-skills`:

### 3.1. `create-adr`
* **Local:** [`.agents/skills/create-adr/`](../.agents/skills/create-adr/)
* **Descrição:** Guia estruturado para documentar escolhas arquiteturais significativas no formato MADR.
* **Gatilhos para IAs:** *"crie um adr"*, *"documente essa decisão"*, *"registre por que escolhemos X"*.
* **Arquivos chave:**
  - [`.agents/skills/create-adr/SKILL.md`](../.agents/skills/create-adr/SKILL.md): Instruções operacionais e templates.
  - [`.agents/skills/create-adr/README.md`](../.agents/skills/create-adr/README.md): Documentação da skill.

### 3.2. `tlc-spec-driven`
* **Local:** [`.agents/skills/tlc-spec-driven/`](../.agents/skills/tlc-spec-driven/)
* **Descrição:** Metodologia de desenvolvimento orientada a especificações com 4 fases adaptativas (*Specify → Design → Tasks → Execute*), validação em notação EARS e gates determinísticos em Python.
* **Gatilhos para IAs:** *"especifique a feature"*, *"planeje tarefas"*, *"implemente com verificação"*, *"valide requisitos"*.
* **Estrutura interna:**
  - `references/`: Guias de especificação, design, testes e sub-agentes.
  - `scripts/`: Validadores automáticos em Python (`validate_spec.py`, `validate_tasks.py`, `check_commit.py`, etc.).

### 3.3. `obsidian-markdown`
* **Local:** [`.agents/skills/obsidian-markdown/`](../.agents/skills/obsidian-markdown/)
* **Descrição:** Criação e edição de notas em Obsidian Flavored Markdown (`.md`) com wikilinks estruturados, frontmatter YAML (`tags`, `aliases`), callouts e embeds.
* **Gatilhos para IAs:** *"crie uma nota"*, *"adicione wikilinks"*, *"formate em obsidian"*, *"extraia frontmatter"*.

### 3.4. `json-canvas`
* **Local:** [`.agents/skills/json-canvas/`](../.agents/skills/json-canvas/)
* **Descrição:** Especificação aberta JSON Canvas 1.0 (`.canvas`) para geração e manipulação de mapas mentais espaciais e grafos visuais.
* **Gatilhos para IAs:** *"exporte para canvas"*, *"crie um json canvas"*, *"mapa mental espacial"*.

---

## 🧭 4. Guia para Agentes de IA (Como Navegar no Repositório)

Ao atuar neste repositório:
1. **Antes de propor decisões de arquitetura:** Consulte a pasta [`docs/adr/`](adr/) para verificar precedentes estabelecidos.
2. **Ao registrar novas decisões:** Use a skill [`create-adr`](../.agents/skills/create-adr/SKILL.md) e siga o formato MADR.
3. **Ao planejar e implementar novas features:** Use a metodologia [`tlc-spec-driven`](../.agents/skills/tlc-spec-driven/SKILL.md) mantendo tarefas atômicas e rastreabilidade de requisitos.
4. **Ao alterar a estrutura do banco ou CLI:** Consulte [`docs/ARCHITECTURE.md`](ARCHITECTURE.md) e [`docs/TURBOQUANT.md`](TURBOQUANT.md).