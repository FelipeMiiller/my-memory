# Tasks: semantic-drift-detection

## Test Coverage Matrix

| Task | Requirement | Unit / Integration Test | Target File |
| :--- | :--- | :--- | :--- |
| T1 | SD-01 | `TestOSGitRunner_*`, `TestMockGitRunner_*` | `internal/drift/git_test.go` |
| T2 | SD-02, SD-03 | `TestAnalyzeDrift_*`, `TestUncoveredCode_*` | `internal/drift/drift_test.go` |
| T3 | SD-04 | `TestRunDriftCommand_*` | `cmd/mem/drift_test.go` |
| T4 | SD-05 | `TestMemoryGetDriftTool_*` | `internal/mcp/drift_handlers_test.go` |
| T5 | SD-01..05 | ADR-031, validação e verificação | `.specs/features/semantic-drift-detection/validation.md` |

---

## Gate Check Commands

```bash
# T1 Gate
go test -v ./internal/drift/...

# T2 Gate
go test -v ./internal/drift/...

# T3 Gate
go test -v ./cmd/mem/... -run TestDrift

# T4 Gate
go test -v ./internal/mcp/...

# T5 Gate
go test -count=1 ./...
```

---

## Execution Plan

```mermaid
graph TD
    T1 -> T2
    T2 -> T3
    T3 -> T4
    T4 -> T5
```

---

## Task Breakdown

### Phase 1: Core Drift Engine

### T1: Implementar Extrator de Histórico e Diff do Git
Where: internal/drift/git.go
Depends on: none
Tests: internal/drift/git_test.go
Gate: go test -v ./internal/drift/...
Details:
- Definir estruturas `GitCommit`, `FileChange`, `GitDiffSummary`.
- Definir interface `GitRunner` com métodos `Log(repoRoot, rangeStr string) ([]GitCommit, error)` e `Diff(repoRoot, rangeStr string) ([]FileChange, error)`.
- Implementar `OSGitRunner` baseado em `os/exec` e `MockGitRunner` para testes isolados e determinísticos.

### T2: Implementar Motor de Análise de Desvio Semântico e Código Órfão
Where: internal/drift/drift.go
Depends on: T1
Tests: internal/drift/drift_test.go
Gate: go test -v ./internal/drift/...
Details:
- Implementar correspondência entre arquivos de código modificados e notas de documentação/ADRs através de menções diretas de caminho e correspondência léxica.
- Implementar cálculo de Drift Score ponderado e classificação de severidade (`LOW`, `MEDIUM`, `HIGH`, `CRITICAL`).
- Implementar detecção de código órfão (`UncoveredCode`) para arquivos de código modificados sem nenhuma nota vinculada.
- Retornar estrutura consolidada `DriftReport`.

### Phase 2: CLI & MCP Integration

### T3: Implementar Subcomando CLI mem drift
Where: cmd/mem/drift.go
Depends on: T2
Tests: cmd/mem/drift_test.go
Gate: go test -v ./cmd/mem/... -run TestDrift
Details:
- Implementar comando `mem drift` suportando flags `--since`, `--threshold`, `--uncovered`, `--strict`, `--json`.
- Integrar resolução de banco (SQLite e PostgreSQL) e deep links para as notas afetadas.
- Integrar no roteador `cmd/mem/main.go` e no manual de ajuda.

### T4: Implementar Ferramenta MCP memory_get_drift
Where: internal/mcp/drift_handlers.go
Depends on: T3
Tests: internal/mcp/drift_handlers_test.go
Gate: go test -v ./internal/mcp/...
Details:
- Definir `ToolMemoryGetDrift` em `internal/mcp/tools.go`.
- Implementar handler em `internal/mcp/drift_handlers.go` gerando relatórios Markdown estruturados com deep links e recomendações para o agente.
- Registrar handler em stdio e HTTP/SSE em `cmd/mem/main.go`.

### Phase 3: Governança & Validação

### T5: Documentar ADR-031 e Concluir Verificação do Sistema
Where: docs/adr/031-semantic-drift-e-deteccao-de-desvio-codigo-memoria.md
Depends on: T4
Tests: cmd/mem/drift_test.go
Gate: go test -count=1 ./...
Details:
- Criar `docs/adr/031-semantic-drift-e-deteccao-de-desvio-codigo-memoria.md` no padrão MADR.
- Atualizar `docs/adr/README.md`, `docs/README.md`, `docs/CLI_GUIDE.md` e `docs/AGENT_INTEGRATION_GUIDE.md`.
- Atualizar `.specs/STATE.md` e gerar relatório `validation.md`.
