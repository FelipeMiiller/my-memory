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
7. **Teste referencia a constante, não a copia.** Se um teste repete um valor que também existe como constante no código de produção, ele envelhece junto com a constante — e falha na migração seguinte, não antes dela.
   - Ruim: `dummyVec := make([]float32, 768)` quando o schema é `embedder.EmbeddingDim`.
   - Bom: `dummyVec := make([]float32, embedder.EmbeddingDim)`.
   - Caso real (2026-10-13): a migração 768 → 384 (`refactor(embedder)`) deixou o teste de integração do Postgres gravando vetor de dimensão errada. Passou 4 dias sem ninguém ver.
8. **Regra que o CI não exercita é decorativa.** Antes de dar uma regra por cumprida, confirmar que existe um job que a executa de fato.
   - O mesmo `768` do item 7 sobreviveu porque o CI estava vermelho desde 2026-10-13: o restore do cache do `setup-go` falhava com `tar: Cannot open: File exists` **antes de qualquer teste rodar**, e os jobs de teste dependem do job de lint. Nenhum teste rodou por 4 dias.
   - Verificar o status real do pipeline (`gh run list`) ao encerrar uma tarefa, não confiar na memória de que "rodou antes".
9. **Teste flaky é teste quebrado.** Um teste que falha em ~40% das execuções treina o time a ignorar vermelho, que é o mesmo efeito de não ter gate nenhum. Flaky é bug a corrigir (ISSUE-011), não caso de borda a tolerar.

---

← [[AGENTS]]
