# Validation: federated-central-vault-and-repo-identity

Result: PASS

## Evidence Summary

| Requisito | Status | Evidência de Código / Teste |
| :--- | :--- | :--- |
| **FED-01** (Identidade Imutável `repo_id`) | PASS | `internal/config/config.go:94`, `internal/config/config.go:149`, `cmd/mem/main.go:796`, `cmd/mem/init_test.go:49` |
| **FED-02** (Convenção `.memory/` e Zero-Credentials no Git) | PASS | `cmd/mem/main.go:802`, `internal/config/config.go:212`, `docs/adr/033-federated-central-vault-and-repo-identity.md:1` |
| **FED-03** (Auto-Bootstrap do Vault Central Virgem) | PASS | `internal/federation/bootstrap.go:34`, `internal/federation/bootstrap.go:17`, `internal/federation/bootstrap_test.go:19` |
| **FED-04** (Persistência Unificada - Banco Único e Isolamento) | PASS | `internal/federation/bootstrap.go:88`, `internal/federation/search.go:249`, `internal/config/config.go:510` |
| **FED-05** (Busca Híbrida Federada via RRF) | PASS | `internal/federation/search.go:28`, `internal/federation/search.go:149`, `internal/federation/search_test.go:17`, `cmd/mem/main.go:1942`, `cmd/mem/main.go:2208` |
| **FED-06** (Configuração Global em Cascata e `mem setup`) | PASS | `internal/config/config.go:501`, `cmd/mem/setup.go:22`, `cmd/mem/setup_test.go:19`, `internal/config/config_test.go:355` |
| **FED-07** (Catálogo de Repositórios e Descoberta MCP) | PASS | `internal/config/config.go:448`, `cmd/mem/setup.go:275`, `cmd/mem/setup.go:192`, `cmd/mem/setup_test.go:83`, `cmd/mem/main.go:866` |

---

## 1. Testes Automatizados Executados

### Identidade e Configuração em Cascata (`internal/config`)
- [x] **Geração Determinística de `repo_id`**: Identificador no formato `repo_<12-hex-chars>` criptográfico (`internal/config/config_test.go:121`).
- [x] **Imutabilidade de `repo_id`**: Preservação inalterada mesmo durante reindexações ou `mem init --force` (`cmd/mem/init_test.go:80`).
- [x] **Configuração Global e Cascata**: Carregamento em cascata `~/.memory/config.yaml` mesclado com `.memory/config.yaml` local (`internal/config/config_test.go:355`).
- [x] **Descoberta Segura sem Vazamento Global**: `FindConfigFile` ignora `~/.memory/config.yaml` durante busca ascendente em diretórios temporários (`internal/config/config_test.go:55`).

### Auto-Bootstrap do Cofre Central (`internal/federation`)
- [x] **11 Pastas Canônicas**: Criação de `standards/`, `architecture/`, `security/`, `infrastructure/`, `operations/`, `data/`, `ai-agents/`, `domain/`, `guides/`, `templates/` e `staging/` (`internal/federation/bootstrap_test.go:35`).
- [x] **Templates de Engenharia**: Modelos oficiais MADR ADR, RFC, Runbook de Operações e Especificação EARS gerados em `templates/` (`internal/federation/bootstrap_test.go:45`).
- [x] **MOC Inicial com Wikilinks**: Geração de `README.md` raiz com wikilinks prontos para navegação no Obsidian Desktop e Mobile (`internal/federation/bootstrap_test.go:57`).
- [x] **Isolamento do Banco SQLite Central**: Banco SQLite estritamente isolado em `<central_path>/.memory/storage/memory.db` com `repo_id: repo_central` (`internal/federation/bootstrap_test.go:65`).

### Busca Híbrida Federada e MCP (`internal/federation` e `internal/mcp`)
- [x] **Reciprocal Rank Fusion (RRF)**: Fusão concorrente de resultados locais e centrais com cálculo $Score = \sum \frac{1}{k + rank}$ (`internal/federation/search_test.go:17`).
- [x] **Anotação de Proveniência**: Marcação explícita de origem `[local]` e `[central]` em títulos, repositórios e sources (`internal/federation/search_test.go:88`).
- [x] **Resiliência a Desconexão do Google Drive/OneDrive**: Fallback suave para modo local quando o cofre central está unmounted ou inacessível sem quebrar requisições (`internal/federation/search_test.go:125`).
- [x] **Integração no MCP Server**: Servidor MCP conecta o `FederatedSearcher` em runtime tanto para backend SQLite quanto PostgreSQL (`cmd/mem/main.go:1942`, `cmd/mem/main.go:2208`).

### Linha de Comando e Catálogo Global (`cmd/mem`)
- [x] **Assistente `mem setup`**: Configuração interativa e não-interativa (`--yes`) gravando em `~/.memory/config.yaml` (`cmd/mem/setup_test.go:19`).
- [x] **Subcomandos `mem central status` e `mem central bootstrap`**: Auditoria de conectividade e inicialização de vault virgem sob demanda (`cmd/mem/setup_test.go:50`).
- [x] **Catálogo `mem repos`**: Listagem tabular de todos os repositórios registrados com detecção de integridade em disco (`cmd/mem/setup_test.go:83`).
- [x] **Zero-Credentials no Git**: Template de `mem init` não contém bloco de credenciais de banco de dados (`cmd/mem/main.go:802`).
- [x] **Auditoria e Status**: `mem status` exibe `Repo ID` imutável e conectividade do Cofre Central (`cmd/mem/status.go:107`).

---

## 2. Documentação e Governança

- [x] **ADR-033 Aceito**: Registrado em `docs/adr/033-federated-central-vault-and-repo-identity.md` no padrão MADR, atualizado com requisitos de Zero-Credentials, banco único no PostgreSQL e catálogo de repositórios.
- [x] **Índices de Documentação Atualizados**: Registrado em `docs/adr/README.md` e `docs/README.md`.
- [x] **Guia da CLI Atualizado**: Documentação detalhada de `mem setup`, `mem central`, `mem repos` e ordens de precedência em `docs/CLI_GUIDE.md`.
- [x] **Guia Prático Atualizado**: Adicionado "Passo 9: Federação Multirrepositório e Cofre Central" e atualizada a tabela de comandos em `COMO_USAR.md`.
- [x] **README Atualizado**: Adicionado o 5º pilar de arquitetura e atualizado o Quickstart em `README.md`.
- [x] **Reindexação e Validação de Desvio**: Executado `mem index` (46 documentos processados, status in sync) e verificado com `mem status`.
