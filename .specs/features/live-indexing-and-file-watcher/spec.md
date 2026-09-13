# Feature: live-indexing-and-file-watcher

## Problem Statement
Atualmente, a atualização do índice de conhecimento no My-Memory é um processo manual disparado sob demanda (`mem index`). Em fluxos de trabalho reais de escrita de notas (como no Obsidian) ou de edição contínua de código, ocorrem três problemas de fricção:
1. **Desatualização Imediata:** Enquanto o usuário ou agente cria notas e altera conexões, o banco local de embeddings e grafos fica desatualizado até que o usuário se lembre de invocar `mem index` manualmente.
2. **Reindexações Desnecessárias de Árvore Completa:** Mesmo com cache SHA-256, invocar `mem index` percorre todo o filesystem do vault. Quando uma única nota é alterada, apenas aquela nota deveria ser processada de forma cirúrgica.
3. **Risco de Descompasso no Git:** Se notas forem alteradas e comitadas no repositório sem a sincronização do banco local ou dos metadados de grafo, outros desenvolvedores ou agentes autônomos podem consultar índices desatualizados.

## Goals
- [ ] Implementar o motor de monitoramento de arquivos `Watcher` em `internal/watcher/watcher.go` com polling configurável e detecção de criação, modificação e remoção.
- [ ] Implementar mecanismo de debouncing em `internal/watcher/debouncer.go` para agrupar rajadas rápidas de escrita.
- [ ] Integrar filtragem declarativa de arquivos via `cfg.ShouldIndex(relPath)`, ignorando `.git`, `node_modules`, `.obsidian`, etc.
- [ ] Implementar rotina de reindexação cirúrgica em `internal/watcher/indexer.go`, processando unicamente a nota alterada ou removida.
- [ ] Adicionar subcomando CLI `mem watch [<pasta>] [--debounce <ms>] [--interval <ms>] [--db <arq>] [--postgres <url>] [--repo <slug>]` em `cmd/mem/main.go`.
- [ ] Implementar subcomandos de Git Hooks `mem hook install` e `mem hook uninstall` em `cmd/mem/hook.go`.
- [ ] Registrar decisão de arquitetura na ADR-017.

## Out of Scope
- Sincronização remota via WebSockets com servidores na nuvem (o watcher opera exclusivamente no filesystem local do vault).
- Interface gráfica GUI nativa com bandeja do sistema (opera como processo CLI / daemon contínuo).

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| Mecanismo de Watcher | Polling em Go padrão (`time.Ticker` + stat) | 100% multiplataforma (Windows/Linux/macOS) sem dependências CGO ou syscalls instáveis | y |
| Duração Padrão de Debounce | 500ms | Janela ideal para consolidar múltiplos saves automáticos do editor | y |
| Intervalo de Polling | 1000ms (1s) | Balanço excelente entre responsividade e zero impacto de CPU (< 0.1%) | y |
| Ação em Exclusão | Purgar dados (`DeleteDocumentData`) | Mantém o grafo e chunks livres de documentos órfãos | y |
| Hook Instalado | `.git/hooks/pre-commit` | Padrão do Git para validação antes de registrar commits | y |

**Open questions:** none - all resolved or logged above (required before the spec is confirmed).

---

## User Stories

### P1: Motor de Watcher e Debounce ⭐ MVP

**User Story**: As a desenvolvedor ou usuário do vault, I want que o sistema observe as modificações de arquivos em segundo plano de forma estável so that eventos de escrita sejam capturados e consolidados sem sobrecarregar a CPU.

**Why P1**: Fornece a base de detecção de eventos e controle de concorrência necessária para a indexação em tempo real.

**Acceptance Criteria**:
1. The system SHALL define `Watcher` struct in `internal/watcher/watcher.go` supporting start, stop, and event notifications.
2. The system SHALL filter candidate files using `cfg.ShouldIndex(relPath)`, ignoring excluded patterns and system directories.
3. The system SHALL provide debouncing in `internal/watcher/debouncer.go` consolidating multiple writes to the same file within the debounce window.
4. WHEN the watcher context is cancelled THEN the system SHALL shut down gracefully without leaking goroutines.

**Independent Test**: Testes unitários em `internal/watcher/watcher_test.go` simulando criação, modificação e remoção de arquivos com debouncing.

---

### P2: Reindexação Cirúrgica em Tempo Real

**User Story**: As a usuário, I want que apenas o arquivo modificado seja reindexado so that o processo seja instantâneo (< 50ms) e não bloqueie a máquina.

**Why P2**: Elimina a latência de varrer todo o diretório do vault a cada pequena edição de nota.

**Acceptance Criteria**:
1. The system SHALL provide `IndexSingleFile` in `internal/watcher/indexer.go`.
2. WHEN an existing file is modified AND its SHA-256 hash changed THEN the system SHALL recalculate embeddings, chunks, and graph edges for that file only.
3. WHEN a file is deleted THEN the system SHALL invoke `DeleteDocumentData` for that document ID, removing its chunks and relations.
4. IF a file's content hash is identical to stored hash THEN the system SHALL skip embedding generation.

**Independent Test**: Testes unitários em `internal/watcher/indexer_test.go` verificando reindexação pontual e remoção cirúrgica.

---

### P3: Subcomando CLI mem watch

**User Story**: As a usuário no terminal, I want executar `mem watch` so that meu vault seja continuamente monitorado enquanto escrevo.

**Why P3**: Expõe o daemon de monitoramento diretamente para o fluxo diário do desenvolvedor.

**Acceptance Criteria**:
1. The system SHALL provide `watch` subcommand in `cmd/mem/main.go`.
2. WHEN `mem watch` is invoked without directory THEN the system SHALL auto-discover vault root via `config.FindConfigFile`.
3. The system SHALL handle OS interrupt signals (Ctrl+C / SIGINT / SIGTERM) terminating gracefully with a confirmation log.

**Independent Test**: Teste de inicialização e parsing de flags do subcomando watch em `cmd/mem/watch_test.go`.

---

### P4: Gerenciador de Git Hooks (mem hook)

**User Story**: As a equipe ou desenvolvedor, I want instalar hooks de Git com `mem hook install` so that nenhuma alteração de notas seja comitada desatualizada.

**Why P4**: Blinda o repositório contra descompassos entre notas Markdown e índices relacionais.

**Acceptance Criteria**:
1. The system SHALL provide `hook install` and `hook uninstall` subcommands.
2. WHEN `mem hook install` executes in a Git repository THEN the system SHALL write an executable `.git/hooks/pre-commit` script.
3. WHEN `mem hook uninstall` executes THEN the system SHALL cleanly remove the pre-commit script.

**Independent Test**: Testes de instalação e desinstalação de hook em repositório Git temporário em `cmd/mem/hook_test.go`.

---

## Edge Cases
- IF a file is locked temporarily by the editor during write THEN the system SHALL retry after a short delay instead of crashing.
- IF an untracked binary file is added THEN `cfg.ShouldIndex` SHALL reject it immediately without I/O reads.
- IF the Git repository has no `.git/hooks/` directory THEN `mem hook install` SHALL create the folder before writing the script.

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| WATCH-01 | P1: Motor de Watcher e Debounce | Tasks | Pending |
| WATCH-02 | P1: Motor de Watcher e Debounce | Tasks | Pending |
| WATCH-03 | P1: Motor de Watcher e Debounce | Tasks | Pending |
| WATCH-04 | P2: Reindexação Cirúrgica em Tempo Real | Tasks | Pending |
| WATCH-05 | P2: Reindexação Cirúrgica em Tempo Real | Tasks | Pending |
| WATCH-06 | P3: Subcomando CLI mem watch | Tasks | Pending |
| WATCH-07 | P4: Gerenciador de Git Hooks (mem hook) | Tasks | Pending |

**Coverage:** 7 total, 7 mapped to tasks, 0 unmapped

---

## Success Criteria
- [ ] Pacote `internal/watcher` implementado com testes unitários passando a 100%.
- [ ] Debounce funcional prevenindo rajadas desnecessárias de indexação.
- [ ] Reindexação cirúrgica de nota única executando em milissegundos.
- [ ] Subcomando `mem watch` com auto-scoping de vault e cancelamento gracioso.
- [ ] Subcomandos `mem hook install` e `mem hook uninstall` funcionais.
- [ ] Decisão formal registrada no ADR-017.
