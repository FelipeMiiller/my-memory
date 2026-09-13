# Feature: vault-configuration-and-auto-scoping

## Problem Statement
Atualmente, todos os subcomandos do My-Memory dependem de argumentos manuais de linha de comando (`--db`, `--postgres`, `--repo`, `--mode`, `--limit`, `--decay`, etc.) ou de variáveis de ambiente globais. Além disso, a rotina de indexação avalia apenas o sufixo `.md`, sem a capacidade de o usuário ou equipe definir regras declarativas de inclusão e exclusão de diretórios (ex: `.git`, `node_modules`, `vendor`, `.trash`, `.obsidian`, templates ou rascunhos). É necessário introduzir um sistema de configuração declarativa para vaults e repositórios com auto-scoping e resolução ascendente de caminhos.

## Goals
- [ ] Implementar o pacote `internal/config` com structs `Config`, `StorageConfig`, `EmbeddingConfig` e `SearchConfig`.
- [ ] Implementar a função `DefaultConfig()` com valores padrão seguros para armazenamento, embeddings, busca e exclusões glob.
- [ ] Implementar `FindConfigFile` com resolução ascendente (*upward directory search*) procurando `.mem.yaml`, `.mem.yml`, `.mem.json` ou `.memory/config.yaml`.
- [ ] Implementar `LoadConfig` e `SaveConfig` com parsing de YAML/JSON e mescla automática de valores padrão.
- [ ] Implementar o método `(c *Config) ShouldIndex(relPath string) bool` com correspondência glob para regras de `include` e `exclude`.
- [ ] Adicionar o subcomando `mem init` no CLI para inicializar a pasta `.memory/` com `config.yaml`.
- [ ] Integrar o carregamento automático de configuração nos comandos `mem index`, `mem search` e `mem mcp`.
- [ ] Registrar a decisão de arquitetura na ADR-016.

## Out of Scope
- Configurações remotas via servidor HTTP de configuração centralizada (o foco é local e declarativo por repositório).
- Sintaxe personalizada de configuração proprietária (usa exclusivamente padrões abertos YAML e JSON).

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Formato Primário | YAML (`.mem.yaml` ou `.memory/config.yaml`) | Legibilidade e padrão da comunidade Obsidian e Markdown | y |
| Formato Secundário | JSON (`.mem.json` ou `.memory/config.json`) | Interoperabilidade nativa com ferramentas e agentes | y |
| Exclusões Padrão | `.git/**`, `node_modules/**`, `vendor/**`, `.obsidian/**`, `.trash/**`, `.memory/**` | Evita indexação desnecessária de dependências e arquivos de sistema | y |
| Precedência de Configuração | Flags CLI > Env Vars > Arquivo de Config > Defaults de Código | Padrão da indústria que garante retrocompatibilidade | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Motor de Configuração Declarativa e Resolução de Arquivo ⭐ MVP

**User Story**: As a desenvolvedor ou agente, I want carregar e salvar a configuração declarativa do vault a partir de arquivos YAML/JSON so that as preferências do projeto fiquem versionadas no repositório.

**Why P1**: Fornece a fundação de dados desacoplada e as estruturas necessárias para os comandos do sistema.

**Acceptance Criteria**:
1. The system SHALL define `Config` and `DefaultConfig()` in `internal/config` containing storage, embedding, search, and glob rules.
2. The system SHALL provide `FindConfigFile(startDir string)` traversing parent directories up to root to locate configuration files.
3. WHEN `LoadConfig` parses a configuration file with omitted fields THEN the system SHALL merge missing properties with defaults from `DefaultConfig()`.
4. The system SHALL provide `SaveConfig(path string, cfg *Config)` writing formatted YAML to disk.

**Independent Test**: Testes unitários em `internal/config/config_test.go` verificando resolução ascendente, parsing e integridade de defaults.

---

### P2: Filtragem de Arquivos por Padrões Glob (ShouldIndex)

**User Story**: As a sistema de indexação, I want avaliar se um arquivo deve ser indexado com base em regras de `include` e `exclude` so that arquivos indesejados sejam ignorados automaticamente.

**Why P2**: Impede que arquivos de dependências externas (`node_modules`, `vendor`, `.git`) poluam o banco de conhecimento.

**Acceptance Criteria**:
1. The system SHALL provide `(c *Config) ShouldIndex(relPath string) bool` in `internal/config`.
2. WHEN `relPath` matches any pattern in `c.Exclude` THEN the system SHALL return `false`.
3. WHERE `c.Include` is populated AND `relPath` matches an include pattern AND no exclude pattern THEN the system SHALL return `true`.
4. IF `relPath` does not have a valid included extension or matches an exclude rule THEN the system SHALL return `false`.

**Independent Test**: Testes de glob em `internal/config/config_test.go` validando exclusão de `.git`, `node_modules` e aceitação de notas `.md`.

---

### P3: Subcomando CLI mem init

**User Story**: As a usuário, I want executar `mem init` no terminal so that a pasta de configuração `.memory/config.yaml` seja criada automaticamente com instruções claras.

**Why P3**: Reduz o atrito inicial de configuração (*time-to-hello-world*) em novos projetos.

**Acceptance Criteria**:
1. The system SHALL provide `init` subcommand in `cmd/mem/main.go`.
2. WHEN `mem init` is invoked THEN the system SHALL create `.memory/` directory and `.memory/config.yaml` if not already present.
3. IF `.memory/config.yaml` already exists THEN the system SHALL not overwrite it unless `--force` is specified.

**Independent Test**: Execução de `mem init` em diretório temporário verificando criação do arquivo de configuração e idempotência.

---

### P4: Integração dos Defaults de Configuração no CLI

**User Story**: As a usuário no terminal, I want que `mem index`, `mem search` e `mem mcp` usem as configurações declaradas no `.memory/config.yaml` so that eu não precise repetir flags longas em cada comando.

**Why P4**: Conecta a camada de configuração declarativa à operação real do My-Memory.

**Acceptance Criteria**:
1. WHEN `mem index` executes without explicit directory THEN the system SHALL index the root directory of the discovered config.
2. WHEN `mem index` traverses directories THEN the system SHALL filter candidate files using `cfg.ShouldIndex`.
3. WHEN `mem search` executes without explicit search flags THEN the system SHALL adopt mode, decay, limit, and k defaults from `cfg.Search`.
4. WHERE explicit CLI flags are supplied THEN the system SHALL override configuration file values with higher precedence.

**Independent Test**: Verificação de execução integrada no CLI com valores derivados do arquivo de configuração.

---

## Edge Cases
- IF no configuration file is discovered when traversing upward THEN the system SHALL fall back gracefully to `DefaultConfig()` without error.
- IF a configuration file contains invalid YAML/JSON syntax THEN the system SHALL return a descriptive error without crashing.
- IF a directory path contains backslashes on Windows THEN `ShouldIndex` SHALL normalize separators to forward slashes before evaluating glob patterns.

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| CFG-01 | P1: Motor de Configuração Declarativa e Resolução de Arquivo | Tasks | Pending |
| CFG-02 | P1: Motor de Configuração Declarativa e Resolução de Arquivo | Tasks | Pending |
| CFG-03 | P1: Motor de Configuração Declarativa e Resolução de Arquivo | Tasks | Pending |
| CFG-04 | P2: Filtragem de Arquivos por Padrões Glob (ShouldIndex) | Tasks | Pending |
| CFG-05 | P3: Subcomando CLI mem init | Tasks | Pending |
| CFG-06 | P4: Integração dos Defaults de Configuração no CLI | Tasks | Pending |

**Coverage:** 6 total, 6 mapped to tasks, 0 unmapped

---

## Success Criteria
- [ ] Pacote `internal/config` implementado com testes unitários passando a 100%.
- [ ] Suporte a `.mem.yaml`, `.mem.yml`, `.mem.json` e `.memory/config.yaml`.
- [ ] `ShouldIndex` operando com correspondência glob confiável.
- [ ] Subcomando `mem init` funcional e documentado no `printHelp()`.
- [ ] Integração com `mem index` e `mem search` respeitando a precedência de flags.
- [ ] Decisão formal registrada no ADR-016.
