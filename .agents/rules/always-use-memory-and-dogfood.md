# Regra Obrigatória: Dogfooding do My-Memory, Observabilidade e Validação Ponta a Ponta

Todo Agente de IA atuando neste repositório DEVE seguir rigorosamente esta diretriz:

---

## 1. Uso Mandatório da Memória (Dogfooding Contínuo)
- **Não faça varredura cega de arquivos**: Para investigar arquitetura, decisões técnicas, regras de negócio ou estrutura de código, **sempre utilize o servidor MCP do `my-memory`** ou a CLI antes de ler múltiplos arquivos brutos no disco.
  - `memory_search`: para recuperar conceitos, termos e decisões anteriores via busca híbrida (RRF).
  - `memory_inspect_node`: para obter o tríptico cirúrgico (in-links, nó e out-links) de qualquer componente com *zero file reads*.
  - `memory_find_path`: para rastrear a cadeia de dependências ou rota de menor custo epistêmico entre dois nós.
  - `memory_get_impact`: para calcular o raio de destruição (*blast radius*) antes de alterar arquivos e contratos existentes.
  - `memory_get_hubs`: para identificar os pontos de entrada e nós centrais (*God Nodes*) do vault.
  - `memory_get_clusters`: para entender a divisão macro de domínios e comunidades conceituais.

---

## 2. Monitoramento Ativo de Gargalos e Desempenho
- Durante cada invocação de ferramentas MCP ou CLI, o agente deve **observar a qualidade da resposta e latência**:
  - Respostas vazias ou irrelevantes?
  - Lentidão perceptível em I/O ou cálculos de grafo?
  - Falsos positivos ou dead links nas tabelas?
  - Qualquer gargalo observado deve ser reportado ao usuário ou mitigado nas decisões de arquitetura.

---

## 3. Validação Real em MCP e CLI ao Final de Cada Tarefa
- Sempre que concluir ou modificar qualquer funcionalidade, comando, ferramenta ou refatoração:
  1. **Testes Automatizados**:
     ```bash
     go test -count=1 ./...
     ```
  2. **Recompilação do Binário**:
     ```bash
     go build -v -o bin/mem.exe ./cmd/mem
     ```
  3. **Teste Real na CLI**:
     - Executar o comando CLI correspondente diretamente no terminal (`.\bin\mem.exe <comando>`) e validar a renderização tabular/ASCII e a flag `--json`.
  4. **Teste Real no MCP**:
     - Executar a ferramenta correspondente diretamente via MCP (`call_mcp_tool`) no servidor `my-memory` e validar o retorno formatado em Markdown e conformidade com o JSON Schema.
