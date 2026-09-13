# Regra Obrigatória: Testes após Conclusão de Tarefas

Toda vez que o agente concluir ou modificar qualquer código, função, arquivo ou documentação técnica:

1. **Executar Testes Imediatamente:** O agente DEVE executar os testes unitários relevantes antes de declarar a tarefa pronta ou propor commits:
   ```bash
   go test -v ./internal/store/... ./internal/parser/... ./internal/turboquant/... ./internal/mcp/... ./internal/canvas/... ./internal/repo/...
   ```
2. **Critério de Aceitação:** Nenhum commit é permitido se houver testes falhando.
3. **Novos Comportamentos Exigem Novos Testes:** Toda nova funcionalidade, caso de borda ou correção de bug deve incluir testes unitários assertivos correspondentes.
