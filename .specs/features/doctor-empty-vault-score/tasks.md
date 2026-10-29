# Tasks: doctor-empty-vault-score

**Spec**: [spec.md](./spec.md)
**Date**: 2026-10-28
**Issue**: ISSUE-019 (`.specs/ISSUES.md:612`)

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Store (cálculo compartilhado) | unit | Vault vazio → score não aplicável (nunca 100); vault com documentos → penalidades da ADR-013 | internal/store/doctor_test.go | go test -count=1 ./internal/store/... |
| Engine SQLite | unit (via CLI e2e) | `db.DiagnoseHealth` em banco vazio devolve score não aplicável | cmd/mem/doctor_test.go | go test -count=1 ./cmd/mem/... |
| MCP Contract | unit | `health_score` ausente + Markdown "não se aplica"; JSON dos vaults saudáveis byte-idêntico | internal/mcp/handlers_test.go | go test -count=1 ./internal/mcp/... |
| CLI Dashboard | integration | Binário real em vault vazio não imprime "Health Score: 100/100" | cmd/mem/doctor_test.go | go test -count=1 ./cmd/mem/... |
| Viewer | unit + build | Schema aceita `health_score` anulável; tile mostra "—" | viewer/src/lib/schemas.test.ts | npm run typecheck && npm run test |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Após mexer no cálculo ou no contrato MCP | go test -count=1 ./internal/store/... ./internal/mcp/... |
| Full | Após a integração completa | go test -count=1 ./... |
| Build | Antes de declarar pronto | go build ./... && go vet ./... |
| Frontend | Após tocar a viewer | cd viewer && npm run typecheck && npm run test && npm run build |
| Spec | Antes de entregar | python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/doctor-empty-vault-score/spec.md && python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py .specs/features/doctor-empty-vault-score/tasks.md |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2 -> T3 -> T4
T4 -> T5
T4 -> T6
T6 -> T7
```

### Phase 1: Correção do Health Score para vault sem documentos indexados

## Task Breakdown

### T1: Regra única de aplicabilidade e cálculo compartilhado do Health Score
**What**: Extrair a matemática de penalidades para `store.CalculateHealthScore`, adicionar `DoctorReport.HealthScoreApplicable()` (denominador = `TotalDocuments > 0`) e ligar o `DiagnoseHealth` do PostgreSQL à função compartilhada
**Where**: internal/store/doctor.go
**Depends on**: none
**Requirement**: EMPTY-04, EMPTY-05, EMPTY-06, EMPTY-07, EMPTY-08
**Tests**: internal/store/doctor_test.go
**Gate**: go test -count=1 ./internal/store/...
**Done when**:
- [x] `CalculateHealthScore` devolve 0 (inerte) quando `TotalDocuments == 0` — nunca 100
- [x] `HealthScoreApplicable()` é a única regra de aplicabilidade, usada pelo cálculo e pelos consumidores
- [x] A cópia de teste `CalculateDoctorHealthScore` some: o teste passa a exercitar a função de produção
- [x] Penalidades da ADR-013 preservadas: 100 no relatório perfeito, 90 com 2 dead links, 87 com órfã + self-loop
- [x] `DiagnoseHealth` (PostgreSQL) usa a função compartilhada

### T2: Ligar o DiagnoseHealth do SQLite à função compartilhada
**What**: Trocar o bloco de cálculo duplicado do SQLite pela chamada a `store.CalculateHealthScore`, eliminando a guarda `if report.TotalDocuments > 0` que produzia o 100/100
**Where**: internal/db/graph.go
**Depends on**: T1
**Requirement**: EMPTY-01, EMPTY-02, EMPTY-08
**Tests**: cmd/mem/doctor_test.go
**Gate**: go test -count=1 ./cmd/mem/... ./internal/store/...
**Done when**:
- [x] O bloco de cálculo local é removido e substituído por `report.HealthScore = store.CalculateHealthScore(*report)`
- [x] A matemática de penalidades não existe mais duplicada no pacote `db`
- [x] O motivo da mudança fica registrado em comentário no call site (ISSUE-019)

### T3: Contrato MCP — `health_score` ausente quando não se aplica e Markdown explícito
**What**: Trocar `mcp.DoctorReport.HealthScore` para `*int` (com `omitempty`: a chave sai ausente do payload) e fazer o `FormatDoctorReport` dizer que o vault não tem documentos indexados
**Where**: internal/mcp/handlers.go
**Depends on**: T2
**Requirement**: EMPTY-01, EMPTY-02, EMPTY-03
**Tests**: internal/mcp/handlers_test.go
**Gate**: go test -count=1 ./internal/mcp/...
**Done when**:
- [x] `HealthScore *int` com `json:"health_score,omitempty"` — a chave fica ausente quando o score não se aplica
- [x] O payload nunca serializa `null` (contrato do wire: `TestStructuredContentNeverEmitsNullArrays`)
- [x] `FormatDoctorReport` imprime "Health Score: não se aplica" e a ação de recuperação quando o score não se aplica (ponteiro nulo)
- [x] O Markdown dos vaults com documentos não muda
- [x] `internal/mcp/structured_content_test.go` adaptado ao novo tipo (literal do score) — expectativa do JSON inalterada

### T4: Dashboard do CLI e conversão para o payload MCP
**What**: Fazer o `displayDoctorReport` não mostrar número quando o score não se aplica e mapear o score interno para `nil` na conversão store → MCP
**Where**: cmd/mem/main.go
**Depends on**: T3
**Requirement**: EMPTY-01, EMPTY-03
**Tests**: cmd/mem/doctor_test.go
**Gate**: go test -count=1 ./cmd/mem/...
**Done when**:
- [x] O dashboard usa indicador neutro (⚪) e o rótulo "não se aplica" num vault sem documentos
- [x] O dashboard mostra a ação de recuperação (`mem index`)
- [x] `convertStoreDoctorReportToMCP` não emite `health_score` para vault sem documentos (a chave não viaja no payload)
- [x] O caminho com documentos continua imprimindo "Health Score: N/100" com o indicador por faixa

### T5: Regressão end-to-end no binário real
**What**: Cobrir o defeito no binário compilado — vault vazio não pode reportar 100/100, e vault com documentos tem de continuar reportando o score
**Where**: cmd/mem/doctor_test.go
**Depends on**: T4
**Requirement**: EMPTY-01, EMPTY-04
**Tests**: cmd/mem/doctor_test.go
**Gate**: go test -count=1 ./cmd/mem/...
**Done when**:
- [x] Teste do vault vazio: a saída não contém "Health Score: 100/100" e contém "não se aplica"
- [x] Teste do vault com documentos: a saída contém "Health Score: 100/100"
- [x] Teste da conversão store → MCP: vault sem documentos não emite `health_score`; com documentos emite o número
- [x] O teste falha quando o comportamento é revertido (comprovado na validation.md)

### T6: Viewer tolera `health_score` ausente ou `null`
**What**: Tornar o schema da viewer tolerante à ausência de valor (chave ausente **ou** `null` do Go), normalizando para `undefined`, e deixar o tile de Health Score capaz de mostrar "sem score"
**Where**: viewer/src/lib/schemas.ts
**Depends on**: T4
**Requirement**: EMPTY-02
**Tests**: viewer/src/lib/schemas.test.ts
**Gate**: cd viewer && npm run typecheck && npm run test && npm run build
**Done when**:
- [x] `DoctorReport.health_score` aceita `null` e ausência, normalizando para `undefined` (`z.pipe(z.nullish(z.number()), z.transform(...))`)
- [x] O tile da viewer mostra "—" e "vault sem documentos" quando o score está ausente ou nulo (`viewer/src/routes/index.tsx`)
- [x] `viewer/src/lib/schemas.test.ts` cobre o payload real do Go com `health_score: null`, a chave ausente, e mantém `health_score: "alta"` inválido
- [x] `npm run typecheck`, `npm run test` e `npm run build` verdes, com o bundle dentro do orçamento de 160 kB gzip

### T7: Validação final e relatório de evidências
**What**: Rodar o gate completo, provar o teste de regressão contra o defeito revertido e registrar as evidências por critério
**Where**: .specs/features/doctor-empty-vault-score/validation.md
**Depends on**: T6
**Requirement**: EMPTY-01, EMPTY-02, EMPTY-03, EMPTY-04, EMPTY-05, EMPTY-06, EMPTY-07, EMPTY-08
**Tests**: .specs/features/doctor-empty-vault-score/validation.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_state.py doctor-empty-vault-score
**Done when**:
- [x] `go build ./...`, `go vet ./...` e `go test -count=1 ./...` com exit 0
- [x] `gofmt -l` limpo nos arquivos tocados (medido em cópia com LF)
- [x] Defeito revertido → teste falhando com o sintoma exato → restaurado
- [x] `validation.md` com evidência `arquivo:linha` por critério
