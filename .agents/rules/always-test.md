# Regra Obrigatória: Testes após Conclusão de Tarefas e Validação Direta em MCP/CLI

Toda vez que o agente concluir ou modificar qualquer código, função, arquivo ou especificação técnica:

1. **Executar Testes Automatizados Imediatamente:** O agente DEVE executar os testes dos pacotes modificados ou a suíte completa:
   ```bash
   go test -count=1 ./...
   ```
2. **Recompilar o Executável:** Garantir que o binário do projeto esteja atualizado:
   ```bash
   go build -v -o bin/mem.exe ./cmd/mem
   ```
3. **Validação Operacional Direta (CLI & MCP):**
   - Executar o comando correspondente diretamente na CLI (`.\bin\mem.exe <comando>`) avaliando renderização e saída `--json`.
   - Executar a ferramenta correspondente diretamente via MCP (`call_mcp_tool`) no servidor `my-memory` para confirmar a integração de ponta a ponta com o protocolo.
4. **Observabilidade de Gargalos:** Ao executar testes na CLI e no MCP, avaliar ativamente latência, possíveis timeouts, consumo de I/O e precisão semântica das respostas, reportando gargalos encontrados.
5. **Critério de Aceitação:** Nenhum commit ou encerramento de tarefa é permitido se houver testes falhando ou se a execução real na CLI/MCP apresentar erro.
6. **Novos Comportamentos Exigem Novos Testes:** Toda nova funcionalidade, caso de borda ou correção de bug deve incluir testes unitários assertivos correspondentes.

---

← [[AGENTS]]
