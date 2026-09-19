# Repositório Autoconsciente (Repository Brain)

Este documento descreve como integrar o `my-memory` dentro do repositório de qualquer projeto para fornecer à Inteligência Artificial (Claude Code, Cursor, Antigravity, Copilot) uma memória semântica e relacional viva.

---

## 🎯 Por que colocar o banco no repositório?

1. **Elimina a perda de contexto**: Em bases de código com centenas de arquivos, a janela de contexto das IAs não comporta todo o código. O `my-memory` traz apenas as seções cirúrgicas necessárias.
2. **Evita que a IA quebre dependências**: O grafo (`graph_edges`) avisa quais módulos dependem daquele arquivo antes da IA alterá-lo.
3. **Memória de Decisões (ADRs)**: Evita que a IA proponha soluções ingênuas ou tente refatorar partes do sistema contrariando decisões arquiteturais tomadas no passado.
4. **Contexto Cirúrgico (*Zero File Reads* - inspiração CodeGraph)**: A IA elimina explorações cegas e dezenas de leituras manuais de arquivos; as ferramentas de memória entregam a vizinhança e dependências exatas em uma única chamada.

---

## 📂 Estrutura Padrão Recomendada no Projeto

Para inicializar a estrutura recomendada em qualquer projeto de software, basta rodar:

```bash
mem init --repo "minha-org/meu-projeto"
```

O comando cria a pasta `.memory/` e o arquivo de configuração declarativa `config.yaml`:

```text
meu-projeto/
├── .memory/
│   ├── config.yaml          # Configuração declarativa de storage, include/exclude e busca
│   ├── rules.md             # Instruções obrigatórias para a IA
│   ├── decisions/           # Notas de decisões de arquitetura em Markdown
│   │   ├── 001-autenticacao.md
│   │   └── 002-cache-redis.md
│   └── memory.db            # Banco SQLite local gerado (sqlite-vec + TurboQuant)
├── src/
│   └── ...
├── .gitignore               # Recomendado: ignorar .memory/memory.db
└── README.md
```

---

## ⚡ Estratégia de Versionamento

### Opção 1: Banco Compilado Localmente (Recomendada)
* Mantenha os arquivos `.md`, `config.yaml` e o código no Git.
* O arquivo `.memory/memory.db` fica no `.gitignore`.
* Ao clonar ou atualizar o projeto, basta rodar diretamente:
  ```bash
  mem index
  ```
  O My-Memory detecta automaticamente o arquivo de configuração `.memory/config.yaml`, aplica os filtros declarados e reconstrói o banco com cache incremental SHA-256.

### Opção 2: Banco Versionado Compacto com TurboQuant
* Se quiser commitar o `memory.db` no repositório para evitar etapa de indexação para outros desenvolvedores, o **TurboQuant** reduz o peso dos vetores em 88%, viabilizando manter o arquivo `.db` pequeno e gerenciável no histórico do Git.

---

## 🔌 Integração com Model Context Protocol (MCP)

Ferramentas como **VS Code (GitHub Copilot)**, Cursor, Claude Code e Antigravity suportam servidores MCP nativamente.

### Configuração no VS Code & GitHub Copilot (`.vscode/mcp.json`)

O VS Code e o GitHub Copilot Chat utilizam o padrão `"servers"` com especificação do tipo de transporte (`stdio` ou `sse`):

```json
{
  "servers": {
    "my-memory": {
      "type": "stdio",
      "command": "mem",
      "args": ["mcp", "--db", ".memory/memory.db"]
    }
  }
}
```

Ou apontando para o servidor de rede HTTP/SSE (porta `38400`):

```json
{
  "servers": {
    "my-memory": {
      "type": "sse",
      "url": "http://127.0.0.1:38400/sse"
    }
  }
}
```

> 💡 **Auto-wiring:** Você pode gerar o arquivo `.vscode/mcp.json` e as diretrizes do Copilot (`.github/copilot-instructions.md`) diretamente via CLI:
> ```bash
> mem init --vscode --copilot
> ```

### Configuração no Cursor (`.cursor/mcp.json`) e Claude Desktop

```json
{
  "mcpServers": {
    "my-memory": {
      "command": "mem",
      "args": ["mcp", "--db", ".memory/memory.db"]
    }
  }
}
```

### Modo PostgreSQL Centralizado com pgvector (Multi-Repositório)

```json
{
  "mcpServers": {
    "my-memory": {
      "command": "mem",
      "args": [
        "mcp",
        "--postgres", "postgres://user:pass@localhost:5432/memory?sslmode=disable",
        "--repo", "meu-org/meu-projeto"
      ]
    }
  }
}
```

### Modo Servidor Remoto via HTTP/SSE (Rede / Nuvem / Múltiplos Agentes)
Iniciado previamente via `mem mcp --port 38400 [--host 0.0.0.0]`:
```json
{
  "servers": {
    "my-memory": {
      "type": "sse",
      "url": "http://127.0.0.1:38400/sse"
    }
  }
}
```

A IA ganha acesso automático a ferramentas com escopo de repositório:
* `memory_search(query, mode?, limit?, repository?, decay?, half_life?, decay_weight?)`: Recupera os chunks de maior relevância semântica, léxica ou híbrida (RRF), com suporte a decaimento temporal exponencial (ADR-015).
* `memory_get_neighbors(node_id, max_depth?, repository?)`: Retorna nós vizinhos e dependências conectadas no grafo via SQL recursivo.
* `memory_get_clusters(min_size?, repository?)`: Detecta e lista comunidades/clusters temáticos densos no grafo via LPA ponderado e Modularidade Newman-Girvan \(Q\) (ADR-022).
* `memory_write_note(path, content, title?, tags?, aliases?, note_type?, relations?, overwrite?, repository?)`: Cria ou atualiza notas atômicas no vault com frontmatter e conexões tipadas, disparando indexação cirúrgica imediata.
* `memory_append_section(path, heading, content, create_if_missing?, repository?)`: Anexa seções e blocos de conteúdo sob cabeçalhos Markdown existentes ou novos.
* `memory_compile_note(topic, target_path, title?, search_mode?, limit?, tags?, overwrite?, repository?)`: Sintetiza conhecimento sobre um tópico a partir de buscas no repositório (padrão *Compile-not-Retrieve*), gravando nota estruturada com backlinks.
* `memory_export_canvas(node_id, max_depth?, repository?)`: Gera JSON Canvas 1.0 espacial para visualização gráfica no Obsidian.
* `memory_visualize_graph(root_node?, max_depth?, output_path?, repository?)`: Exporta uma visualização interativa do grafo em página HTML/SVG standalone com física de forças, busca em tempo real, PageRank e agrupamento por cores de comunidade.
* `memory_doctor(repository?, fix?)`: Audita a integridade do grafo (dead links, notas órfãs, self-loops e Health Score), com suporte a reparo automático.

---

## 🔄 Sincronização Contínua em Tempo Real e Git Hooks

Para manter a memória do repositório sempre alinhada com as anotações do time sem esforço manual:

### 1. Monitoramento em Segundo Plano (`mem watch`)
Durante sessões de desenvolvimento ou escrita no Obsidian:
```bash
mem watch
```
O `mem watch` utiliza um debouncer inteligente que consolida rajadas de salvamento contínuo e reindexa cirurgicamente as notas modificadas ou deletadas em frações de segundo, mantendo as ferramentas MCP sempre atualizadas para os agentes de IA.

### 2. Automação Pré-Commit com Git Hooks (`mem hook`)
Para garantir que nenhuma alteração em notas de documentação ou decisões seja commitada sem estar indexada no banco vetorial e grafo relacional:
```bash
mem hook install
```
Isso instala um script pre-commit leve e não-invasivo em `.git/hooks/pre-commit` que roda `mem index` antes de finalizar o commit. Para remover a qualquer momento:
```bash
mem hook uninstall
```

---

## 🌱 Concept Stubs: Quando e Como

Tags recorrentes merecem ser nós do grafo, não filtros soltos. **Concept stubs** são notas curtas que servem como "casa" para conceitos centrais do seu vault.

### Decision matrix

| Situação | Ação |
|---|---|
| Tag usada em **≥3 docs** e representa conceito central | ✅ Criar stub em `docs/concepts/<tag>.md` |
| Tag usada em 1-2 docs (específica) | ❌ Aceitar como filtro |
| Tag ambígua (significados diferentes) | 🔧 Renomear pra ser específica |

**Por que ≥3 docs?** Limiar arbitrário mas útil: abaixo disso, a tag é específica demais pra ser hub. Acima, é conceito recorrente e merece nó.

### Walkthrough: criar stub `architecture`

**Cenário:** você nota que a tag `architecture` aparece em 5 docs (`COMO_FUNCIONA.md`, `SKILL.md`, `ARCHITECTURE.md`, `AGENT_INTEGRATION_GUIDE.md`, etc). Sem stub, cada uma dessas tags vira filtro solto — sem nó no grafo.

**Passo 1 — Criar o stub:**

```bash
mkdir -p docs/concepts
```

Crie `docs/concepts/architecture.md` com frontmatter exato:

```markdown
---
title: "architecture"
category: resource
summary: "Visão arquitetural consolidada do projeto"
tags: [concept, architecture, design]
---

# architecture

<descrição de 1-2 frases>

## Onde aparece neste vault

- [[doc-que-referencia-1]] — contexto
- [[doc-que-referencia-2]] — contexto
```

**Regra crítica:** `title` do frontmatter = slug do arquivo (exato, sem fuzzy). O parser casa via exact title match pra stubs.

**Passo 2 — Indexar e validar:**

```bash
bin/mem.exe index --force
bin/mem.exe doctor --db .memory/memory.db
```

**Esperado:**
- Documentos indexados: **+1**
- Dead links: **0** (sem regressão)
- Orphans: pode **cair 1** (a tag stub vira nó resolvido, sai do orphan list)
- Health Score: **não regride**

**Passo 3 — Verificar resolução da tag:**

```sql
SELECT source_id, target_id FROM graph_edges
WHERE relation = 'tagged_as' AND target_id = 'architecture';
-- Deve retornar 5 rows (1 por doc que usa a tag)
```

### Quando NÃO criar stub

- **Tags de domínio específico** (`pipeline-de-ci`, `deploy-production`) — vivem em 1-2 docs só
- **Tags de trabalho** (`adr-040`, `spec-041`) — versionadas com o trabalho, não conceitos
- **Tags-meta** (`parser`, `graph`) — descrevem o sistema, não o domínio

### Evolução do stub

Conforme o stub cresce (passa de 200 linhas, ganha seções próprias, vira doc conceitual maduro), **promova-o pra raiz**:

```bash
mv docs/concepts/architecture.md docs/ARCHITECTURE.md
# atualizar wikilinks se necessário
```

Convenção segrega stubs (placeholders) de concept docs completos (documentação madurecida).

### Ver também

- [`docs/concepts/README.md`](concepts/README.md) — convenção formal completa
- [ADR-041](../adr/041-issue-009-tag-sem-casa-removida-do-grafo.md) — regra original: tags sem casa viram filtros, não edges
- [Spec 042](../specs/042-stubs-editoriais/spec.md) — escopo + EARS
- [ADR-005](../adr/005-markdown-com-wikilinks-como-fonte-de-verdade.md) — wikilinks como entrada canônica
