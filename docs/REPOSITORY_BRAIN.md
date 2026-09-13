# Repositório Autoconsciente (Repository Brain)

Este documento descreve como integrar o `my-memory` dentro do repositório de qualquer projeto para fornecer à Inteligência Artificial (Claude Code, Cursor, Antigravity, Copilot) uma memória semântica e relacional viva.

---

## 🎯 Por que colocar o banco no repositório?

1. **Elimina a perda de contexto**: Em bases de código com centenas de arquivos, a janela de contexto das IAs não comporta todo o código. O `my-memory` traz apenas as seções cirúrgicas necessárias.
2. **Evita que a IA quebre dependências**: O grafo (`graph_edges`) avisa quais módulos dependem daquele arquivo antes da IA alterá-lo.
3. **Memória de Decisões (ADRs)**: Evita que a IA proponha soluções ingênuas ou tente refatorar partes do sistema contrariando decisões arquiteturais tomadas no passado.

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

Ferramentas como Claude Code, Cursor e Antigravity suportam servidores MCP nativamente.
Ao plugar o `my-memory` como servidor MCP no arquivo `.vscode/mcp.json` ou de configuração do Claude:

### Modo 1: SQLite Local (Zero-Config)
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

### Modo 2: PostgreSQL Centralizado com pgvector (Multi-Repositório)
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

A IA ganha acesso automático a ferramentas com escopo de repositório:
* `memory_search(query, mode?, limit?, repository?, decay?, half_life?, decay_weight?)`: Recupera os chunks de maior relevância semântica, léxica ou híbrida (RRF), com suporte a decaimento temporal exponencial (ADR-015).
* `memory_get_neighbors(node_id, max_depth?, repository?)`: Retorna nós vizinhos e dependências conectadas no grafo via SQL recursivo.
* `memory_export_canvas(node_id, max_depth?, repository?)`: Gera JSON Canvas 1.0 espacial para visualização gráfica no Obsidian.
* `memory_get_hubs(top?, repository?, algorithm?)`: Retorna os nós com maior centralidade de conexões (*God Nodes* / Hubs) via grau ou autoridade PageRank ponderada.
* `memory_get_insights(limit?, min_similarity?, repository?)`: Descobre conexões latentes (*Surprising Connections*) entre conceitos sem links diretos.
* `memory_doctor(repository?, fix?)`: Audita a integridade do grafo (dead links, notas órfãs, self-loops e Health Score), com suporte a reparo automático.
