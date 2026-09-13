# Repositório Autoconsciente (Repository Brain)

Este documento descreve como integrar o `my-memory` dentro do repositório de qualquer projeto para fornecer à Inteligência Artificial (Claude Code, Cursor, Antigravity, Copilot) uma memória semântica e relacional viva.

---

## 🎯 Por que colocar o banco no repositório?

1. **Elimina a perda de contexto**: Em bases de código com centenas de arquivos, a janela de contexto das IAs não comporta todo o código. O `my-memory` traz apenas as seções cirúrgicas necessárias.
2. **Evita que a IA quebre dependências**: O grafo (`graph_edges`) avisa quais módulos dependem daquele arquivo antes da IA alterá-lo.
3. **Memória de Decisões (ADRs)**: Evita que a IA proponha soluções ingênuas ou tente refatorar partes do sistema contrariando decisões arquiteturais tomadas no passado.

---

## 📂 Estrutura Padrão Recomendada no Projeto

Em qualquer projeto de software, crie a pasta `.memory/`:

```text
meu-projeto/
├── .memory/
│   ├── config.json          # Regras de inclusão/exclusão de arquivos
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
* Mantenha os arquivos `.md` e o código no Git.
* O arquivo `.memory/memory.db` fica no `.gitignore`.
* Ao clonar ou atualizar o projeto, basta rodar:
  ```bash
  mem index .
  ```
  O Go reconstrói o banco em poucos segundos localmente.

### Opção 2: Banco Versionado Compacto com TurboQuant
* Se quiser commitar o `memory.db` no repositório para evitar etapa de indexação para outros desenvolvedores, o **TurboQuant** reduz o peso dos vetores em 88%, viabilizando manter o arquivo `.db` pequeno e gerenciável no histórico do Git.

---

## 🔌 Integração com Model Context Protocol (MCP)

Ferramentas como Claude Code, Cursor e Antigravity suportam servidores MCP nativamente.
Ao plugar o `my-memory` como servidor MCP no arquivo `.vscode/mcp.json` ou equivalente:

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

A IA ganha acesso automático a ferramentas:
* `memory_search(query)`: Recupera os chunks de maior similaridade semântica.
* `memory_get_neighbors(symbol)`: Retorna nós vizinhos e dependências no grafo.
* `memory_get_decision(topic)`: Consulta os ADRs e decisões arquiteturais vigentes.
