# AGENTS.md

Instruções operacionais e diretrizes para **Agentes de IA** (Antigravity, Claude Code, Cursor, Windsurf, Copilot) neste repositório.

> [!TIP]
> **Dica de Instalação:** Mova ou copie este arquivo para a raiz do seu repositório:
> `cp .memory/AGENTS.md ./AGENTS.md`

---

## 🧠 Memória do Repositório (My-Memory)

Este repositório utiliza o **My-Memory** como motor de contexto semântico, relacional e busca híbrida.

### 1. Verificação de Inicialização Obrigatória
Ao iniciar qualquer tarefa neste repositório:
1. Verifique se o diretório `.memory/` ou o arquivo `.memory/config.yaml` existe.
2. Se **NÃO existir**:
   - Execute no terminal: `mem init` (ou `go run ./cmd/mem init`).
   - O comando gerará de forma segura:
     - `.memory/config.yaml`: Escopo de pastas e banco de dados.
     - `.memory/.gitignore`: Protege credenciais e bancos locais.
     - `.memory/.env.example`: Modelo de variáveis de ambiente.
     - `.memory/AGENTS.md`: Este modelo de instruções.
   - Em seguida, execute `mem index` para alimentar a base.

### 2. Diretrizes de Uso da Memória pelo Agente
1. **Consulte antes de modificar**: Antes de refatorar código ou criar novas funcionalidades, execute `memory_search` (ou `mem search "<tema>"`) para verificar decisões de arquitetura e notas existentes.
2. **Avalie o Raio de Impacto (Blast Radius)**: Ao modificar ou renomear arquivos e conceitos críticos, use `memory_get_impact` (ou `mem impact <id>`) para analisar dependentes diretos e reversos.
3. **Persista Conhecimento Atômico**: Após tomar decisões ou implementar novas features, utilize `memory_write_note` para salvar a síntese no vault com `[[wikilinks]]`.
4. **Higiene e Integridade**: Use `memory_doctor` para auditar a saúde do grafo e detectar links quebrados.

---

## 🛠 Comandos Operacionais para o Agente

```bash
# Indexar alterações no vault de notas
mem index

# Busca híbrida no contexto
mem search "<pergunta ou conceito>"

# Inspecionar nó cirúrgico em 3 colunas (in-links, nó central, out-links)
mem inspect "<caminho ou id>"

# Avaliar raio de impacto de alterações
mem impact "<caminho ou id>" --depth 2

# Iniciar servidor Model Context Protocol (MCP)
mem mcp
```

---

## 📌 Regras de Conduta para Agentes de IA
1. **Codificação:** Arquivos em **UTF-8 sem BOM**.
2. **Commits:** Padrão *Conventional Commits* (`feat:`, `fix:`, `docs:`, `chore:`, `refactor:`).
3. **Testes:** Sempre execute a suíte de testes antes de concluir tarefas.
4. **Memória Atualizada:** Mantenha notas de documentação sincronizadas ao alterar componentes críticos.
