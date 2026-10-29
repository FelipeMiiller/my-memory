# Validation: fix-drift-repo-context

- **Feature**: fix-drift-repo-context (ISSUE-035)
- **Date**: 2026-10-29
- **Spec**: [spec.md](./spec.md)
- **Verdict**: PASS

## Validation: PASS

**Result**: PASS

A correção é de **uma linha de escopo** (`cmd/mem/drift.go:85`), com o defeito
demonstrado por medição (não por leitura de código) e coberto por um teste de
regressão que roda contra **PostgreSQL real**.

### O que o teste prova

`cmd/mem/drift_postgres_test.go:96` — `TestDriftCLI_PostgresNaoAcusaMudancaEstruturalEmArquivoInalterado`:

1. indexa `pkg/alpha.go` no banco descartável do `internal/pgtest`, no repositório
   `repo-drift`;
2. roda o `mem drift` com um runner de git falso que diz que `pkg/alpha.go`
   mudou — mas o arquivo é **o mesmo** que foi indexado;
3. exige **zero** mudanças estruturais no relatório.

**Prova de que não é verde vacuamente:** com o fix revertido
(`ctx = db.WithRepository(ctx, resolvedRepo)` removido), o teste morre com o
sintoma exato:

```
--- FAIL: TestDriftCLI_PostgresNaoAcusaMudancaEstruturalEmArquivoInalterado (11.47s)
    drift_postgres_test.go:98: o drift acusou 1 mudança(s) ESTRUTURAL(is) em arquivo INALTERADO:
      [{FilePath:pkg/alpha.go}] (o hash estrutural gravado tem que ser comparado com o atual)
```

### Evidência no vault real (`repo_e4c8f3b1a2d5`)

Range `adfb8cc~5..adfb8cc` (5 commits, 15 arquivos), mesmo vault, mesma árvore:

| Situação | drift reportado |
| --- | --- |
| **Antes** do fix (sem escopo de repositório) | 15 HIGH + 3 MÉDIO · 18 notas · 1 mudança estrutural por arquivo indexado |
| **Depois** do fix (índice atualizado) | **0** · 0 notas · 0 mudanças estruturais |
| Depois + **mudança cosmética** (comentário em `internal/config/config.go`) | **0** — a mudança cosmética é suprimida |
| Depois + **mudança estrutural** (assinatura de `FindConfigFile`) | **244 HIGH + 21 MÉDIO** · 265 notas · **1** mudança estrutural detectada |
| Depois de **reverter** | **0** |

O que a tabela prova: o caminho estrutural **passou a suppresses a mudança
cosmética** (o que o caminho textual não conseguia fazer) e **passou a detectar
a mudança de assinatura**, atribuindo-a às notas que referenciam o arquivo
(atribuição pela ponte, que também estava morta pelo mesmo defeito de escopo).

### AC por AC

| AC | Status | Evidência |
| --- | --- | --- |
| AC1 — o contexto carrega o repositório | PASS | `cmd/mem/drift.go:85` (`ctx = db.WithRepository(ctx, resolvedRepo)`) |
| AC2 — arquivo inalterado não vira mudança estrutural | PASS | `cmd/mem/drift_postgres_test.go:96` + medição no vault (0 drift) |
| AC3 — mudança de assinatura continua detectada | PASS | experimento no vault (1 mudança estrutural) e o teste com o fix revertido |
| AC4 — SQLite inalterado | PASS | `internal/drift/**` não foi tocado; o filtro de repositório não existe no SQLite (`internal/db/code_bridge.go:898`) |
| AC5 — o teste morre contra o código antigo | PASS | saída do FAIL reproduzida acima |

### Gate

- `MY_MEMORY_PG_TEST=1 go test -count=1 -run TestDriftCLI_Postgres ./cmd/mem/` → exit 0
- `go test -count=1 ./...` → exit 0 (26 pacotes `ok`)
- `go build ./...` → exit 0 · `go vet ./...` → exit 0
- `gofmt -l` na cópia LF de `cmd/mem/drift.go` e `cmd/mem/drift_postgres_test.go` → vazio
- `validate_spec.py` 0 erros · `validate_tasks.py` 0 erros · `validate_state.py` 0 erros

### Riscos / pendências

- **Nenhum ADR novo**: a correção é de escopo de contexto, não de schema, driver
  ou arquitetura — a ADR-061 (code graph) e a ADR-039 (calibração do drift)
  continuam válidas.
- O `mem drift` continua reportando **o que mudou desde o último índice** — se o
  índice está atualizado, o drift fica em zero por construção. É a semântica do
  detector (ADR-039), não uma regressão.
