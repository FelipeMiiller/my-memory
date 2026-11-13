# Validation: fix-doctor-tag-edges

- **Feature**: fix-doctor-tag-edges (ISSUE-038)
- **Date**: 2026-11-13
- **Spec**: [spec.md](./spec.md)
- **Verdict**: PASS

## Validation: PASS

**Result**: PASS

O `mem doctor` parou de contar a aresta de tag como link morto, e — o mais
importante — **não** passou a contar qualquer nó como vivo (o que tornaria o
check vacuamente verde).

### A regra viva (precisa, e com a régua do contra-teste)

Um alvo de aresta está **vivo** quando é:

1. um **documento** (id ou título) — `documents`;
2. uma **tag** conhecida — `document_tags` (por `tag` ou `tag_key`, ADR-055);
3. um **símbolo de código** — `graph_nodes` com `type='code_symbol'` (ADR-061);
4. (futuro, ISSUE-039) um arquivo que **existe no vault** mas está fora do
   índice por decisão — a distinção "fora do índice" × "rompido" que o dono
   definiu em 2026-11-13.

**Não** é vivo "por existir como nó": o grafo cria **nós-placeholder** para
todo link citado (no cofre central: 238 nós para 94 documentos — 94 ids de
documento + 80 títulos + 64 placeholders). Aceitar "qualquer nó" zerava o
contador de dead links **e** mascarava links realmente quebrados — o teste
primeiro passou com essa versão e foi **rejeitado** justamente por isso.

### O que os testes provam

| Teste | Engine | O que prova |
| :--- | :--- | :--- |
| `internal/db/doctor_tag_edges_test.go` · `TestDoctorNaoContaArestaDeTagVivaComoDeadLink` | SQLite | a aresta de tag de uma tag **conhecida** não é dead link |
| `internal/db/doctor_tag_edges_test.go` · `TestCorrecaoDeDeadLinksPreservaArestaDeTagViva` | SQLite | a rotina de correção **preserva** a aresta viva |
| `internal/store/doctor_tag_edges_test.go` · `TestPostgresDoctorNaoContaArestaDeTagVivaComoDeadLink` | PostgreSQL real | a tag viva não é dead link; a tag **inexistente** continua sendo |
| `internal/store/doctor_tag_edges_test.go` · `TestPostgresFixHealthIssuesPreservaArestaDeTagViva` | PostgreSQL real | a correção apaga a morta e preserva a viva |

**Prova de que não é verde vacuamente** (mutante, com o código restaurado em
seguida): removendo a cláusula de `document_tags` do check de dead link, o
teste do SQLite morre com o sintoma exato —

```
--- FAIL: TestDoctorNaoContaArestaDeTagVivaComoDeadLink (0.09s)
    doctor_tag_edges_test.go:64: dead links = 1; want 0 (a tag existe como nó e em document_tags):
      [{SourceID:nota.md TargetID:ddd Relation:tagged_as}]
```

### Medição nos vaults reais (2026-11-13)

| Vault | Antes | Depois |
| :--- | :--- | :--- |
| Cofre central (`repo_central`) | 148 dead links | **138** (os 10 alvos de tag que estão em `document_tags`) |
| Vault do repo (`repo_e4c8f3b1a2d5`) | 38 306 (falso positivo, ISSUE-036) | **2** — os dois `[[ADR-035]]` reais |

O **Health Score do vault do repo** fica em **90/100** com os 2 links
verdadeiramente quebrados ✓ (e não em 100/100, que seria o sinal de um check
vacuamente verde).

### AC por AC

| AC | Status | Evidência |
| --- | --- | --- |
| AC1 — tag viva não é dead link | PASS | os 4 testes acima |
| AC2 — tag morta continua morta | PASS | `TestPostgresDoctorNaoContaArestaDeTagVivaComoDeadLink` (1 dead link) |
| AC3 — a correção usa o mesmo critério | PASS | `TestCorrecaoDeDeadLinksPreservaArestaDeTagViva` + `TestPostgresFixHealthIssuesPreservaArestaDeTagViva` |
| AC4 — vermelho contra o código antigo | PASS | a mutante reproduzida acima |

### O que fica para a próxima (ISSUE-039, decisão do dono)

Os **138** dead links restantes do cofre central são, todos, **referências a
material bruto que existe no disco mas está fora do índice por decisão**
(68 `links_to` + 70 `tagged_as` para notas/tags fora do índice). A correção
é a distinção que o dono definiu: **"fora do índice por decisão" ≠ "link
rompido"** — o que exige a checagem em disco (o alvo do `doctor` precisa do
caminho do vault). Hmm, é a ISSUE-039, com o escopo decidido.

### Gate

- `go test -count=1 -run Tag ./internal/db/... ./internal/store/...` → exit 0
- `go test -count=1 ./...` → exit 0 (26 pacotes `ok`)
- `go build ./...` → exit 0 · `go vet ./...` → exit 0
- `gofmt -l` na cópia LF dos arquivos tocados → vazio
- `validate_spec.py` 0 erros · `validate_tasks.py` 0 erros · `validate_state.py` 0 erros

### Riscos

- O check de dead link ficou **mais largo** (tags e símbolos de código) e
  **mais estreito** (não aceita placeholder) — o comportamento para
  `links_to` continua igual: só resolve para documento.
- Nenhum dado gravado muda ✓ (a correção é só de medição).
