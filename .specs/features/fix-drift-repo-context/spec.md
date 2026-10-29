# Feature: fix-drift-repo-context

- **Spec ID**: `fix-drift-repo-context`
- **Created**: 2026-10-29
- **Owner**: Felipe Miiller
- **Source**: ISSUE-035 — `.specs/ISSUES.md` (seção ISSUE-035)
- **Related**: ADR-061 (code graph) · ADR-039 (calibração do drift) · ISSUE-034 (code graph em lote)

## Problem Statement

`cmd/mem/drift.go` resolve o repositório e **o descarta** (`_, resolvedDB, resolvedPG := resolveStorageAndRepo(...)`). O `ctx` que chega em `drift.AnalyzeDrift` **não carrega o `repository`** — ao contrário de `cmd/mem/codeindex.go:84`, `cmd/mem/codegraph.go:68` e `cmd/mem/main.go:1620`, que fazem `ctx = db.WithRepository(ctx, repository)`.

No PostgreSQL, `internal/db/code_bridge.go` → `CodeSymbolSignatures` filtra por repositório (`f.repository = ? AND s.repository = ?`). Com o contexto sem repositório a consulta devolve **0 linhas**, e `internal/drift/drift.go:561` calcula o hash estrutural gravado como `""` — contra um hash atual não-vazio. Resultado: `res.Structural = true` para **todo** arquivo indexado, e a atribuição pela ponte erra o alvo pelo mesmo motivo (o id do nó é prefixado com o repositório, `db.CodeSymbolNodeID`).

**Medido** em `internal/config/config_test.go`, cujo `content_hash` gravado era idêntico ao SHA-256 do arquivo em disco (o arquivo **não** tinha mudado):

| Contexto | Lado gravado | Lado atual | Veredito |
| --- | --- | --- | --- |
| com `WithRepository` | 37 símbolos · `1ef039df…` | 37 símbolos · `1ef039df…` | iguais → cosmético → fora do drift |
| sem repositório | 0 símbolos · `""` | 37 símbolos · `1ef039df…` | divergentes → estrutural (falso) |

No SQLite não há filtro de repositório em `CodeSymbolSignatures` — por isso os testes existentes (fixture SQLite) nunca pegaram o defeito: ele só aparece no engine de produção.

### O que muda

O `mem drift` passa a rodar com escopo de repositório: a comparação estrutural usa os símbolos gravados **do repositório certo**, e a atribuição pela ponte volta a achar os nós.

## Goals

- [x] O `mem drift` SHALL colocar o repositório resolvido no contexto antes da análise.
- [x] Arquivo indexado e inalterado desde o último índice SHALL NOT ser reportado como mudança estrutural.
- [x] Mudança estrutural (assinatura) SHALL continuar sendo detectada e atribuída às notas que referenciam o arquivo.
- [x] A atribuição pela ponte (`references_code`) SHALL voltar a funcionar no PostgreSQL.
- [x] O comportamento no SQLite SHALL permanecer o mesmo (lá não há filtro de repositório).
- [x] Teste de regressão em PostgreSQL real que falha contra o código antigo.

## Out of Scope

- **Mudar o cálculo do drift** (`internal/drift/**`) — o algoritmo está certo; o que faltava era o escopo no contexto.
- **Mudar `internal/db`** — `CodeSymbolSignatures` está correto; o `mem drift` passa a ter o escopo certo por inteiro.
- **O `mem doctor`** — é a ISSUE-036, feature separada.
- **`mem code-stats`** — agregador por desenho: descarta `defaultRepo` e `repoSlug` de propósito (`cmd/mem/codestats.go:39-40`) e reporta o engine inteiro.
- **Mudar schema, tabelas ou ADR** — nenhum ADR novo é necessário (ver A10).

## Acceptance Criteria

1. WHEN o `mem drift` roda no PostgreSQL THEN o contexto da análise SHALL carregar o repositório resolvido por `--repo` (ou o `repo_id` do vault).
2. IF um arquivo de código está indexado e inalterado desde o último índice THEN o relatório SHALL NOT listar mudança estrutural para ele.
3. IF a assinatura de um símbolo indexado mudou THEN o relatório SHALL listar a mudança estrutural e atribuí-la às notas que referenciam o arquivo.
4. IF o engine é SQLite THEN o comportamento SHALL permanecer o de antes (o filtro de repositório não existe nesse engine).
5. The regression test SHALL fail against the pre-fix code and pass after the fix, running against a real PostgreSQL (the `internal/pgtest` disposable database).

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale |
| --- | --- | --- |
| O repositório do drift é o resolvido por `--repo`, ou o `repo_id`/`repository` do vault | usar `resolveStorageAndRepo`, como os outros comandos | é o mesmo caminho de precedência do ADR-040 |
| `db.WithRepository(ctx, "")` normaliza para `default` | confiar no helper | `internal/db/repository.go:22` já normaliza o valor vazio |
| O `--since` do comando define quais arquivos são analisados | não mudar | é contrato do drift (ADR-039) |

Open questions: none

## User Stories

- Como dono do vault, quero que o drift diferencie mudança cosmética de mudança estrutural, para não receber alertas de arquivos que não mudaram de verdade.
- Como leitor das specs, quero que o drift atribua a mudança às notas que realmente dependem do arquivo, para saber o que revisar.

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Where |
| --- | --- | --- |
| DRF-1 | AC1 — o contexto carrega o repositório | `cmd/mem/drift.go:85` |
| DRF-2 | AC2 — arquivo inalterado não vira mudança estrutural | `cmd/mem/drift_postgres_test.go:96` |
| DRF-3 | AC3 — mudança de assinatura continua detectada | `cmd/mem/drift_postgres_test.go:96` (fix revertido) + medição no vault |
| DRF-4 | AC5 — o teste morre contra o código antigo | `cmd/mem/drift_postgres_test.go:96` |
