# Validation Plan: ai-tool-auto-wiring

## 1. Testes Automatizados Unitários e de Integração

### Detecção de Clientes e Caminhos (`internal/autowire/`)
- [x] `TestDetectClients_PathResolution`: Valida a resolução de caminhos nos três sistemas operacionais (Windows, macOS, Linux) simulando variáveis de ambiente (`APPDATA`, `USERPROFILE`, `HOME`).
- [x] `TestDetectClients_ScopeFiltering`: Valida filtragem correta por escopo (`--workspace` vs `--global`).
- [x] `TestDetectClients_TargetFiltering`: Valida seleção de clientes individuais (`--target claude`, `--target cursor`, etc.).

### Mesclagem e Backup de JSON (`internal/autowire/`)
- [x] `TestInjectMCPServer_NewFile`: Valida criação limpa de arquivo de configuração com diretórios pais criados sob demanda.
- [x] `TestInjectMCPServer_MergeExisting`: Valida que servidores pré-existentes no arquivo JSON são mantidos intactos e apenas `my-memory` é inserido/atualizado.
- [x] `TestInjectMCPServer_BackupCreation`: Valida que o backup `.bak` é criado antes da escrita física.
- [x] `TestInjectMCPServer_DryRun`: Valida que no modo `dryRun: true` nenhum arquivo ou backup é modificado em disco.
- [x] `TestInjectMCPServer_InvalidJSON`: Valida que arquivos corrompidos geram erro seguro sem corrupção adicional.

### Subcomando CLI `mem install` e `mem setup` (`cmd/mem/`)
- [x] `TestInstallCLI_DryRun`: Executa `mem install --dry-run` e verifica a saída descritiva no terminal.
- [x] `TestInstallCLI_WorkspaceOnly`: Executa `mem install --workspace` em um diretório temporário e valida geração de `.cursor/mcp.json` e `.vscode/mcp.json`.
- [x] `TestInstallCLI_TargetFlag`: Valida isolamento por `--target`.
- [x] `TestInstallCLI_AliasSetup`: Valida que o comando `mem setup` executa idêntico a `mem install`.

---

## 2. Critérios de Conclusão e Regressão
- [x] 100% dos testes do novo pacote `internal/autowire` e de `cmd/mem` passando.
- [x] Todos os 15 pacotes Go continuam passando com 100% de sucesso (`go test -count=1 ./...`).
- [x] Recompilação de `bin/mem.exe` bem-sucedida.
- [x] Teste real no Windows demonstrando detecção e injeção em clientes disponíveis.
