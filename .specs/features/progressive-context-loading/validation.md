# Validation Plan: progressive-context-loading

## 1. Testes Automatizados Unitários e de Integração

### Parser e Extração L0 (`internal/parser/`)
- [ ] `TestFrontmatter_CategoryAndAbstract`: Valida extração de `category: memory` e `summary: ...`.
- [ ] `TestExtractMicroAbstract`: Valida limpeza de títulos `#`, links `[[...]]`, negrito `**...**` e extração concisa de até 160 caracteres.
- [ ] `TestExtractMicroAbstract_Fallback`: Valida documentos vazios ou sem texto.

### Banco de Dados e Migração (`internal/db/` e `internal/store/`)
- [ ] `TestSchemaMigration_AbstractAndCategory`: Testa abertura de banco SQLite antigo sem as colunas e valida migração sem perda de dados.
- [ ] `TestSearch_CategoryFilter`: Testa busca filtrando por `category="memory"` garantindo exclusão de notas de outros tipos.
- [ ] `TestSearch_LevelL0Projection`: Valida que o resultado de busca em nível L0 retorna `Abstract` preenchido e corpo de conteúdo enxuto.

### Linha de Comando (`cmd/mem/`)
- [ ] `TestSearchCLI_LevelL0`: Executa `mem search --level l0` e valida formato compacto no stdout.
- [ ] `TestSearchCLI_CategoryFilter`: Executa `mem search --category skill` e valida filtragem.

### MCP (`internal/mcp/`)
- [ ] `TestMCP_MemorySearch_LevelL0`: Executa tool call `memory_search` com `detail_level="l0"` e valida resposta semântica concisa sem desperdício de tokens.
- [ ] `TestMCP_MemorySearch_Category`: Executa tool call `memory_search` com `category="resource"`.

---

## 2. Critérios de Conclusão e Regressão
- [ ] Todos os 14 pacotes continuam passando com 100% de sucesso (`go test -count=1 ./...`).
- [ ] Recompilação de `bin/mem.exe` bem sucedida.
- [ ] Reinicialização do servidor MCP em background respondendo com sucesso no `/health`.
