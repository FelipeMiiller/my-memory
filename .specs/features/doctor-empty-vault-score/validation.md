# Feature Validation: doctor-empty-vault-score

**Date**: 2026-10-28
**Spec**: `.specs/features/doctor-empty-vault-score/spec.md`
**Issue**: ISSUE-019 (`.specs/ISSUES.md:612`)

---

## Validation: PASS

**Result**: PASS

O Health Score deixou de reportar 100/100 num vault sem documentos indexados, o caminho com documentos continua idêntico (penalidades da ADR-013 intactas), e o teste de regressão foi provado contra o defeito revertido em três camadas.

## Evidência por critério

| ID | Critério | Evidência (`arquivo:linha`) | Status |
| -- | -------- | ---------------------------- | ------ |
| EMPTY-01 | `mem doctor` num vault vazio não reporta 100/100 e diz que o score não se aplica | `internal/store/doctor.go:28` (cálculo guardado por `HealthScoreApplicable`), `cmd/mem/main.go:3622` (rótulo "não se aplica"), `cmd/mem/main.go:3635` (aviso + ação), `cmd/mem/doctor_test.go:137` (teste e2e do binário real) | PASS |
| EMPTY-02 | O `structuredContent` do `memory_doctor` não emite `health_score` num vault vazio | `internal/mcp/handlers.go:161` (`*int` + `omitempty`), `internal/mcp/handlers_test.go:918` (payload sem a chave), `cmd/mem/main.go:3733` (conversão store → MCP), `cmd/mem/doctor_test.go:207` (conversão coberta), `viewer/src/lib/schemas.ts:114` (schema tolera ausente e `null`), `viewer/src/lib/schemas.test.ts:326` | PASS |
| EMPTY-03 | O relatório diz o que houve e indica a ação, sem indicador verde | `internal/mcp/handlers.go:846`, `internal/mcp/handlers.go:847`, `cmd/mem/main.go:3622`, `cmd/mem/main.go:3635` | PASS |
| EMPTY-04 | Dead links continuam custando 5 pontos (máx. 40) | `internal/store/doctor.go:36`, `internal/store/doctor_test.go:11` (100 → 90 com 2 dead links) | PASS |
| EMPTY-05 | Órfãs continuam custando razão × 40 (máx. 30) | `internal/store/doctor.go:44`, `internal/store/doctor_test.go:46` (87 = 100 − 5 − 5 − 8) | PASS |
| EMPTY-06 | Self-loops (5, máx. 15) e chunks dessincronizados (2, máx. 15) continuam valendo | `internal/store/doctor.go:52`, `internal/store/doctor.go:58`, `internal/store/doctor_test.go:59`, `internal/store/doctor_test.go:112` | PASS |
| EMPTY-07 | Com documentos, o mesmo valor aparece no CLI, no Markdown e no payload | `cmd/mem/doctor_test.go:166` (CLI), `internal/mcp/handlers_test.go:965` (Markdown + payload), `internal/mcp/structured_content_test.go:102` (JSON byte-idêntico) | PASS |
| EMPTY-08 | Um motor só de matemática, sem guarda duplicada | `internal/store/doctor.go:28` (função compartilhada), `internal/db/graph.go:357` (SQLite), `internal/store/doctor.go:182` (PostgreSQL) | PASS |

## Prova do defeito (antes → depois)

**ANTES** — binário compilado do estado original (2026-10-28):

```
$ mem doctor --storage=sqlite --db <arquivo-novo-vazio.db> --repo acme
 🟢 RELATÓRIO DE SAÚDE DA MEMÓRIA & GRAFO (Health Score: 100/100)
 Documentos: 0 | Chunks: 0 | Arestas: 0 | Nós: 0
```

**DEPOIS** — mesmo comando, binário com a correção:

```
 ⚪ RELATÓRIO DE SAÚDE DA MEMÓRIA & GRAFO (Health Score: não se aplica)
 Documentos: 0 | Chunks: 0 | Arestas: 0 | Nós: 0
 ⚠ Vault sem documentos indexados — o Health Score não se aplica (a métrica exige ao menos um documento indexado).
   Rode 'mem index' para popular o vault e medir a saúde do grafo.
 [🔗 Links Quebrados / Dead Links] (0)
  · Nenhum link quebrado — o vault ainda não tem documentos indexados.
 [🏝️  Notas Órfãs / Sem Conexões] (0)
  · Nenhuma nota órfã — o vault ainda não tem documentos indexados.
```

O Markdown do MCP (`internal/mcp/handlers.go:846`) e o `structuredContent` (`internal/mcp/handlers.go:161`) seguem a mesma regra: o texto diz "Health Score: não se aplica" e a chave `health_score` **não viaja** no payload.

## Prova da regressão (defeito revertido → teste falhando → restaurado)

Defeito reintroduzido em três pontos (o cálculo que voltava a 100, a conversão que voltava a emitir o número e a impressão que voltava a mostrar a nota):

1. `internal/store/doctor.go` — `return 0` → `return 100` no guard (a guarda original que pulava o cálculo);
2. `cmd/mem/main.go` — `healthScoreForMCP` voltou a emitir o score sempre;
3. `cmd/mem/main.go` — `displayDoctorReport` voltou a imprimir o número sem condição.

Resultado com o defeito presente — **três camadas de teste falharam com o sintoma exato**:

```
--- FAIL: TestDoctorHealthScore_VazioNaoSeAplica (0.00s)
    doctor_test.go:87: vault vazio não pode receber a nota máxima, obteve 100

--- FAIL: TestDoctor_EmptyVault_DoesNotReportPerfectScore (2.64s)
    doctor_test.go:146: vault vazio não pode reportar Health Score 100/100:
         🟢 RELATÓRIO DE SAÚDE DA MEMÓRIA & GRAFO (Health Score: 100/100)
         Documentos: 0 | Chunks: 0 | Arestas: 0 | Nós: 0

--- FAIL: TestHealthScoreForMCP_VazioNaoEmiteScore (0.00s)
    doctor_test.go:212: vault sem documentos não pode emitir health_score, obteve 100
```

Com o defeito revertido, o binário real reproduz o sintoma original (`🟢 … Health Score: 100/100` com 0 documentos).

**Restaurado** — os mesmos testes passam:

```
ok  github.com/FelipeMiiller/my-memory/internal/store  1.102s
ok  github.com/FelipeMiiller/my-memory/cmd/mem         4.079s
ok  github.com/FelipeMiiller/my-memory/internal/mcp    0.166s
```

## Refinamento durante a Execute — a spec desenhou `null`, a implementação emite ausência

**A spec e o código não sempre concordaram, e a divergência foi resolvida em favor do código.** Registrado aqui para que o próximo leitor não conclua o contrário.

| Fase | Contrato do payload |
| ---- | ------------------- |
| Specify (spec original) | `health_score: null` — "a chave sai com valor nulo" |
| Execute (implementação entregue) | `health_score` **ausente** — a chave não viaja no payload |

**Motivo da mudança (constraint do repositório, não preferência).** Duas evidências convergiram durante a execução:

1. O contrato do wire do `structuredContent` **proíbe `null`**: `TestStructuredContentNeverEmitsNullArrays` (`internal/mcp/structured_content_test.go:237`) falha se o payload serializar qualquer `:null`. A primeira implementação (`json:"health_score"`, sem `omitempty`) **quebrou esse teste** — o `encoding/json` serializa ponteiro `nil` como `null`.
2. A viewer lê ausência como `undefined` (`zod/mini` `.optional()`) e **rejeita `null`** — a viewer já tinha caído uma vez exatamente por causa disso (o comentário do próprio teste registra o incidente do painel `/drift`).

Por isso a tag final é `json:"health_score,omitempty"` (`internal/mcp/handlers.go:161`): com a chave ausente, o payload dos vaults com documentos continua **byte-idêntico** ao de hoje (`internal/mcp/structured_content_test.go:102`, expectativa intocada) e o payload dos vaults vazios **não emite `null`** — ele apenas não traz o campo.

**A spec foi reconciliada com essa decisão** (seção "Decisão de Design", premissas A1/A6/A7, AC-2 da P1 e critérios de sucesso). A viewer foi deliberadamente deixada **tolerante aos dois** (`null` e ausência → `undefined`), para não acoplar o painel ao encoding do Go.

**Nenhum comportamento de usuário mudou por causa do refinamento:** o texto da CLI e do MCP, a ausência de número no payload e a ação de recuperação (`mem index`) são exatamente os descritos na AC-1 e na AC-3.

## Gate

| Gate | Comando | Resultado |
| ---- | ------- | --------- |
| Build | `go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Testes (suíte completa) | `go test -count=1 ./...` (com `MY_MEMORY_PG_URL` e afins neutralizadas) | exit 0 — 26 pacotes `ok` |
| Formatação | `gofmt -l` sobre cópia dos 8 arquivos Go tocados normalizada para LF | **vazio** (nenhum arquivo listado) |
| Frontend — typecheck | `cd viewer && npm run typecheck` | exit 0 |
| Frontend — testes | `cd viewer && npm run test` | exit 0 — 5 arquivos, 98 testes |
| Frontend — build | `cd viewer && npm run build` | exit 0 — `index-*.js` 453,70 kB (146,21 kB gzip); total JS gzip **144.813 bytes** ≤ 163.840 (orçamento de 160 kB) |
| Spec | `validate_spec.py .specs/features/doctor-empty-vault-score/spec.md` | 0 erro(s), 0 warning(s) — exit 0 |
| Tasks | `validate_tasks.py .specs/features/doctor-empty-vault-score/tasks.md` | 0 erro(s), 0 warning(s) — exit 0 |
| Estado | `validate_state.py doctor-empty-vault-score` | exit 0 (este relatório) |

## Diff range

`git diff --stat` (arquivos desta entrega):

```
cmd/mem/doctor_test.go                      |  99 +++++++++
cmd/mem/main.go                             |  45 ++++--
internal/db/graph.go                        |  43 +-------
internal/mcp/handlers.go                     |  39 +++--
internal/mcp/handlers_test.go                | 127 ++++++++++-
internal/mcp/structured_content_test.go      |   2 +-
internal/store/doctor.go                    | 102 ++++++++---
internal/store/doctor_test.go               |  98 ++++++---
viewer/src/lib/schemas.test.ts              |  61 +++++++
viewer/src/lib/schemas.ts                   |  25 +++-
viewer/src/routes/index.tsx                 |  14 ++-
.specs/features/doctor-empty-vault-score/   |  novo (spec.md, tasks.md, validation.md)
```

## Notas de escopo

- **Arquivos tocados fora da lista de propriedade** (todos forçados pela mudança, nenhum decorativo):
  - `internal/db/graph.go` — **obrigatório**: o `DiagnoseHealth` do SQLite mora aqui, não em `internal/store/doctor.go` (que é a implementação PostgreSQL). Sem tocar nele a CLI continuaria reportando 100/100.
  - `internal/mcp/structured_content_test.go` — 1 linha: o literal `HealthScore: 88` precisa virar ponteiro. **A expectativa do JSON não mudou**: para vault com documentos o payload segue byte-idêntico.
  - `viewer/src/routes/index.tsx` — 2 linhas: o tile do Health Score precisa tratar o valor ausente (`undefined`), senão o `tsc` do gate de frontend quebra.
- `CHANGELOG.md`, `docs/adr/README.md`, `docs/CLI_GUIDE.md`, `docs/ARCHITECTURE.md` e `docs/AGENT_INTEGRATION_GUIDE.md` aparecem modificados no `git status`, mas **não foram tocados por esta execução** — são alterações do gestor, feitas em paralelo.
- Nenhum ADR novo: a ADR-013 continua válida (o Health Score de 0 a 100 com penalidades graduadas segue íntegro); o que muda é a **precondição de aplicabilidade**, registrada pelo gestor em `docs/adr/README.md`.
- **Nenhum commit e nenhum push** — o git é do gestor.
