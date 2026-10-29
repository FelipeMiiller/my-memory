# Validation: fix-doctor-dead-links-code-nodes

- **Feature**: fix-doctor-dead-links-code-nodes (ISSUE-036)
- **Date**: 2026-10-29
- **Spec**: [spec.md](./spec.md)
- **Verdict**: PASS

## Validation: PASS

**Result**: PASS

A correção é de **query**: o alvo de uma aresta que resolve para um nó
`code_symbol` passa a ser considerado vivo — nos **dois** engines e **também no
caminho de correção**, que antes apagava o code graph inteiro.

### O que os testes provam

| Teste | Engine | O que prova |
| --- | --- | --- |
| `internal/db/doctor_code_nodes_test.go:35` | SQLite | o nó `code_symbol` **não** é dead link |
| `internal/db/doctor_code_nodes_test.go:75` | SQLite | o alvo que não é documento nem nó de código **continua** morto |
| `internal/db/doctor_code_nodes_test.go:100` | SQLite | a correção **não** apaga a ponte e **apaga** a órfã |
| `internal/store/doctor_code_nodes_test.go:35` | PostgreSQL (real) | o nó `code_symbol` **não** é dead link |
| `internal/store/doctor_code_nodes_test.go:118` | PostgreSQL (real) | o alvo inexistente **continua** morto (no Postgres o `target_id` não tem FK) |
| `internal/store/doctor_code_nodes_test.go:196` | PostgreSQL (real) | a correção **não** apaga a ponte e **apaga** a órfã |

**Prova de que não é verde vacuamente:** com o critério revertido em
`internal/db/graph.go`, o teste morre com o sintoma exato:

```
--- FAIL: TestDoctorNaoContaNoDeCodigoComoDeadLink (0.10s)
    doctor_code_nodes_test.go:47: dead links = 1; want 0 (o nó de código existe no grafo):
      [{SourceID:nota.md TargetID:code:repo-x:pkg.Alpha Relation:references_code}]
```

### Evidência no vault real (`repo_e4c8f3b1a2d5`, Neon)

| Métrica | Antes | Depois |
| --- | --- | --- |
| Dead links reportados | **38 306** | **2** (ambos reais: `[[ADR-035]]` sem documento correspondente) |
| Health Score | **60/100** | **90/100** |
| Documentos / chunks / arestas / nós | 285 / 2571 / 38461 / 3304 | 285 / 2571 / 38461 / 3304 |

Os 2 dead links restantes são verdadeiros: `.specs/features/embedder-builtin-onnx-minilm/spec.md`
e `tasks.md` apontam para `[[ADR-035]]`, que não existe como documento indexado.

### AC por AC

| AC | Status | Evidência |
| --- | --- | --- |
| AC1 — nó de código não é dead link | PASS | `internal/db/graph.go:277` · `internal/store/doctor.go:116` + os 6 testes |
| AC2 — alvo que não é nota nem código continua morto | PASS | `internal/db/doctor_code_nodes_test.go:75` · `internal/store/doctor_code_nodes_test.go:118` |
| AC3 — a correção não apaga o code graph | PASS | `internal/db/doctor_code_nodes_test.go:100` · `internal/store/doctor_code_nodes_test.go:196` |
| AC4 — a correção apaga a órfã | PASS | os mesmos dois testes |
| AC5 — escopo de repositório no Postgres | PASS | `internal/store/doctor.go:116` (`repository = $1 AND type = $2`) |
| AC6 — os testes morrem contra o código antigo | PASS | saída do FAIL reproduzida acima |

### Gate

- `MY_MEMORY_PG_TEST=1 go test -count=1 -run 'TestDoctorNaoConta\|TestCorrecaoDeDeadLinks' ./internal/db/` → exit 0
- `MY_MEMORY_PG_TEST=1 go test -count=1 -run 'TestPostgresDoctor\|TestPostgresFixHealthIssues' ./internal/store/` → exit 0 (23,2 s — Postgres real)
- `go test -count=1 ./...` → exit 0 (26 pacotes `ok`)
- `go build ./...` → exit 0 · `go vet ./...` → exit 0
- `gofmt -l` na cópia LF de `internal/db/graph.go`, `internal/store/doctor.go` e dos dois testes → vazio
- `validate_spec.py` 0 erros · `validate_tasks.py` 0 erros · `validate_state.py` 0 erros

### Riscos / pendências

- **Nenhum ADR novo**: nenhum schema, coluna, índice ou driver mudou — só o
  critério de "alvo vivo". A ADR-013 (Health Score) e a ADR-061 (code graph)
  continuam válidas.
- O literal `"code_symbol"` aparece em `internal/store/doctor.go` porque o
  package não pode importar `internal/db` (ciclo); o valor é o mesmo de
  `db.CodeSymbolNodeType` e o comentário na constante registra isso.
- Os nós de `type='concept'` continuam contando como dead link — comportamento
  anterior, fora do escopo desta feature.
