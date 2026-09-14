# Validation Plan: context-packager-and-subgraph-bundle

## 1. Testes Automatizados Unitários e de Integração

### Motor de Empacotamento Puro (`internal/graph/`)
- [x] `TestEstimateTokens`: Valida cálculo determinístico de tokens para strings vazias, curtas e textos longos.
- [x] `TestPackContext_Basic`: Valida extração de subgrafo com 2 nós em TierCore, mapa Mermaid e Markdown formatado.
- [x] `TestPackContext_BudgetConstraintAndFallback`: Valida degradação graciosa para TierFringe (L0/L1) quando o conteúdo integral excede o limite restante de tokens.
- [x] `TestPackContext_OmittedNodes`: Valida classificação em TierOmitted e listagem na seção de omitidos quando o orçamento se esgota.
- [x] `TestPackContext_DirectionAndCycles`: Valida expansão BFS em grafos cíclicos sem nós duplicados e respeitando direções.
- [x] `TestPackContext_MissingRoot`: Valida retorno de erro apropriado quando o nó raiz não existe.

### Camadas de Persistência SQLite e Postgres (`internal/db/` & `internal/store/`)
- [x] `TestPackContext`: Executa `PackContext` contra banco SQLite real com múltiplos nós, chunks e arestas epistêmicas.
- [x] `TestPostgresStore_Integration`: Valida chamada de `PackContext` no store PostgreSQL com isolamento por repositório.

### Subcomando CLI `mem pack` (`cmd/mem/`)
- [x] `TestRunPackCommand_MissingRoot`: Valida erro elucidativo quando o nó raiz é omitido.
- [x] `TestRunPackCommand_SuccessAndJSON`: Valida emissão em Markdown padrão, saída estruturada com `--json` e salvamento em arquivo com `--out`.
- [x] `TestRearrangePackArgs`: Valida flexibilidade posicional de flags antes e depois do argumento do nó raiz.

### Ferramenta MCP `memory_pack_context` (`internal/mcp/`)
- [x] `TestMemoryPackContextTool_Registered`: Valida registro automático da ferramenta no servidor MCP.
- [x] `TestMemoryPackContextTool_MissingRoot`: Valida rejeição com código de erro `InvalidParams` quando `root_node` não é fornecido.
- [x] `TestMemoryPackContextTool_Success`: Valida retorno formatado em Markdown pronto para prompt via MCP.
- [x] `TestMemoryPackContextTool_PackError`: Valida tratamento e retorno de erro interno quando o empacotamento falha.

---

## 2. Critérios de Conclusão e Regressão
- [x] 100% dos testes novos passando.
- [x] Todos os pacotes Go continuam passando com 100% de sucesso (`go test -count=1 ./...`).
- [x] Recompilação de `bin/mem.exe` bem-sucedida.
