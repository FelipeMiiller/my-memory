# Regras do Agente - My-Memory

## Testes Obrigatórios Pós-Tarefa (Mandatório)
SEMPRE que você concluir uma tarefa, implementar uma feature, corrigir um bug ou refatorar código:
1. **Execute imediatamente os testes unitários** usando:
   ```bash
   go test -v ./internal/store/... ./internal/parser/... ./internal/turboquant/... ./internal/mcp/... ./internal/canvas/... ./internal/repo/...
   ```
2. **100% dos testes devem passar** antes de dar a tarefa como concluída, marcar `[x]` em listas de tarefas ou realizar commits.
3. Se algum teste falhar, corrija a causa raiz imediatamente antes de avançar.
