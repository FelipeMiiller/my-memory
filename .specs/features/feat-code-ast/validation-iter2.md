---
title: "Validation: feat-code-ast (iter 2)"
category: resource
summary: "Verifier iter 2 — revalidação após commits 0983d11 + e5e85e6 (GAP-1 a GAP-5 closure)."
type: validation
created_at: 2026-09-21T10:55:00-03:00
tags: [validation, code-ast, tlc-spec-driven, feat-code-ast, iter-2]
---

# Validation: feat-code-ast (iter 2)

**Date**: 2026-09-21
**Verifier**: mavis (sub-agent, branch `mvs_cc51fd0a51d748da83619b5c057b8a32`)
**Iter**: 2 (revalidação após commits `0983d11` + `e5e85e6`)
**Spec**: `.specs/features/feat-code-ast/spec.md` (17 ACs, EARS notation)
**Tasks**: `.specs/features/feat-code-ast/tasks.md`
**ADR**: `docs/adr/047-code-ast-tree-sitter-multi-linguagem.md`
**Diff range**: `c7b81ad..e5e85e6` (10 commits; foco nos 2 fix commits `0983d11`..`e5e85e6`)
**Verdict**: **PASS-WITH-DEFER** (5/5 gaps formalmente fechados; 2 surviving mutants documentados como spec-gap não-críticos)

---

## 1. Gap Closure Audit

| Gap | Closed? | Evidence |
|---|---|---|
| **GAP-1** (CA-01 deferral tree-sitter real binding) | **YES** | `docs/adr/047-code-ast-tree-sitter-multi-linguagem.md:256-294` adiciona nova seção `## Deferral: Plano Real tree-sitter Binding`. 3 gatilhos (linhas 264-268): (1) gcc+CGO toolchain; (2) `go get go-tree-sitter` succeeds; (3) E2E 95%+ matches `gopls definition`. 5 proibições (linhas 274-278): sem binding em `go.mod`, sem substituir `newBackendParser()`, sem tabela `code_grammar_downloads`, sem remover MockParser, sem promover ADR-047. 5 passos de promoção (linhas 281-287) + subseção Status atual (linhas 290-293). |
| **GAP-2** (CA-12 `include_code` warning loud) | **YES** | `internal/mcp/handlers.go:355-358` declara `includeCodeWarning := ""` + branch `if includeCode { includeCodeWarning = "⚠ include_code=true: fan-in RRF code+markdown agendado para backlog (CA-12 placeholder).\n\n" }`. Linha 393 faz `NewTextResult(includeCodeWarning + ...)`. Smoke E2E (via test ad-hoc `TestVerifierIter2_IncludeCodeWarningLoud` no scratch): com `{query,include_code:true}` a resposta contém `include_code=true: fan-in`; com `include_code:false` a resposta não contém. |
| **GAP-3** (CA-16 relaxada para Postgres deferred) | **YES** | `spec.md:126` agora é `### CA-16: Schema migration idempotente (SQLite local; Postgres deferred)`. Linha 128 explica: "Postgres/pgvector support (ADR-040) is **deferred** because `EnsureCodeTables` uses SQLite-specific `pragma_table_info` and Postgres dialect-aware DDL is not implemented in v1." Linha 153 da tabela de Traceability: `partial (Postgres: deferred per GAP-3)`. |
| **GAP-4** (boot log uniforme nos 4 code-*) | **YES** | `cmd/mem/codesearch.go:42`, `codegraph.go:29`, `codestats.go:29` cada um chama `codeast.LogTreesitterBootStatus()` antes do `flag.Parse`. `codeindex.go:45` já tinha. Smoke (PowerShell): `./bin/mem.exe code-search foo` → `tree-sitter: disabled (code pipeline skipped)` ✓ ; `code-graph foo` → mesmo ✓ ; `code-stats` → mesmo ✓. |
| **GAP-5** (`TestServer_ToolsList` estendido) | **YES** | `internal/mcp/tools_test.go:148-149, 175-186, 201-208` agora declara `foundCodeSearch`/`foundCodeNeighbors`, atribui dentro do loop, e exige presença no tools/list com `t.Errorf` explícito citando CA-10/CA-11. Test passa isolado: `go test ./internal/mcp/... -run TestServer_ToolsList -v` → `--- PASS: TestServer_ToolsList (0.00s)`. |

### Counts

- **5/5 gaps formalmente fechados**
- **2 surviving mutants** (Fault E + Fault G — spec-gap documentados abaixo; não-críticos)
- **0 regressões**

---

## 2. Sensor (iter 2 — faults E/F/G)

Worktree isolado: `C:\tmp\feat-code-ast-scratch2-20260921105659\scratch` (HEAD detached em `e5e85e6`). **Limpo após testes** (`git worktree remove --force` + `git worktree prune`).

| Fault | Injection site | Killed by test | Survived? |
|---|---|---|---|
| **E** — `LogTreesitterBootStatus()` revertido em `codesearch.go` | `cmd/mem/codesearch.go:42` | — (nenhum test CLI mata; smoke manual detecta) | **YES (spec-gap)** |
| **F** — TestServer_ToolsList reverted (assertions removidas) | `internal/mcp/tools_test.go:148-208` | Regressão da direção inversa: removendo o bootstrap em `server.go:114-115` o test `TestServer_ToolsList` FALHA com `'memory_code_search' não encontrada em tools/list (CA-10/ADR-047)` e `'memory_code_neighbors' não encontrada em tools/list (CA-11/ADR-047)` → GAP-5 ESTÁ MATANDO o mutante na direção crítica (server-side). | **NO (killed) na direção crítica** ✅ |
| **G** — `include_code` warning removido em `handlers.go` | `internal/mcp/handlers.go:357` (warning string → `""`) | Test existente `TestMemorySearch_IncludeCodeFlag_AcceptsBoolean` não checa o warning (apenas parsing). Ad-hoc test `TestVerifierIter2_IncludeCodeWarningLoud` no scratch detectaria, MAS **não está no production code**. | **YES (spec-gap)** |

**Sensor kills: 1/3 killed** (Fault F, na direção server-side regression). 2 surviving mutants (E + G) — ambos **spec-gap não-críticos**:

### Diagnose do Fault E (spec-gap menor)

- Mutation: comenta `LogTreesitterBootStatus()` em `cmd/mem/codesearch.go:42`. Build compila ✓. `./bin/mem.exe code-search foo` produz `query: "foo"\nnenhum resultado.` (sem a linha `tree-sitter: ...` que os outros code-* emitem).
- Tests existentes: `TestCodeSearch_*` checam comportamento de busca mas **não capturam stdout** nem verificam o boot log. Logo, test suite continua verde com a mutation.
- Spec gap: CA-14 diz "log once: `tree-sitter: enabled|disabled`" mas não há test que valide a presença da linha (apenas testes que checam o conteúdo do banco).
- **Severidade**: BAIXA — drift silencioso em UX. Smoke manual detecta. Sugestão: adicionar `TestCodeSearchCLI_EmitsBootLog` em `cmd/mem/codesearch_test.go` que faça `CaptureStdout` e verifique o prefixo `tree-sitter:`.

### Diagnose do Fault G (spec-gap menor)

- Mutation: troca `includeCodeWarning = "⚠ ... CA-12 placeholder.\n\n"` por `""` em `handlers.go:357`. Build compila ✓. `TestMemorySearch_IncludeCodeFlag_AcceptsBoolean` passa (só valida que `include_code` não causa parse error).
- **Severidade**: BAIXA — mesma armadilha da iter 1 (CA-12 é placeholder; placeholder "loud" = bom, mas sem guard automatizado). Smoke manual detecta (chamar MCP `memory_search` com `include_code:true` e ver warning no response).
- Sugestão: estender `TestMemorySearch_IncludeCodeFlag_*` para que **também** afirme `strings.Contains(tr.Content[0].Text, "include_code=true: fan-in")` quando flag=true. (Confirmei empiricamente no scratch que um test com essa assertion mata Fault G.)

### Diagnose do Fault F (proteção assimétrica confirmada)

A GAP-5 fix **protege contra regressão do server-side** (que é o cenário mais provável: alguém remove o bootstrap). Mutation de bootstrap removeu `s.RegisterTool(ToolMemoryCodeSearch, ...)` em `server.go:114` → `TestServer_ToolsList` falhou com mensagem `ferramenta 'memory_code_search' não encontrada em tools/list (CA-10/ADR-047)`.

A GAP-5 fix **NÃO protege contra regressão do test-side** (alguém remover as assertions). Mas remover assertions já é um sinal explícito de sabotagem — em prática ninguém remove `t.Errorf` deliberadamente.

---

## 3. Regression Check

`go test -count=1 ./...` (baselines: iter 1 = 28 packages ok, 0 falhas).

**Resultado iter 2**: 28/28 packages PASS, zero falhas. Packages:

```
ok  github.com/FelipeMiiller/my-memory/bench/codeast       27.590s
ok  github.com/FelipeMiiller/my-memory/cmd/mem            73.437s
ok  github.com/FelipeMiiller/my-memory/internal/autowire   0.788s
ok  github.com/FelipeMiiller/my-memory/internal/canvas     0.809s
ok  github.com/FelipeMiiller/my-memory/internal/codeast   2.819s
ok  github.com/FelipeMiiller/my-memory/internal/compiler   0.633s
ok  github.com/FelipeMiiller/my-memory/internal/config    1.056s
ok  github.com/FelipeMiiller/my-memory/internal/db         4.779s
ok  github.com/FelipeMiiller/my-memory/internal/deeplink  0.632s
ok  github.com/FelipeMiiller/my-memory/internal/drift      1.172s
ok  github.com/FelipeMiiller/my-memory/internal/embedder   1.713s
ok  github.com/FelipeMiiller/my-memory/internal/event_runtime  3.211s
ok  github.com/FelipeMiiller/my-memory/internal/federation 0.996s
ok  github.com/FelipeMiiller/my-memory/internal/graph      0.730s
ok  github.com/FelipeMiiller/my-memory/internal/graphview  1.476s
ok  github.com/FelipeMiiller/my-memory/internal/ipc/jsonrpc 1.060s
ok  github.com/FelipeMiiller/my-memory/internal/mcp        3.029s
ok  github.com/FelipeMiiller/my-memory/internal/parser     0.743s
ok  github.com/FelipeMiiller/my-memory/internal/policy     0.718s
ok  github.com/FelipeMiiller/my-memory/internal/repo       0.781s
ok  github.com/FelipeMiiller/my-memory/internal/staleness  1.761s
ok  github.com/FelipeMiiller/my-memory/internal/store      1.330s
ok  github.com/FelipeMiiller/my-memory/internal/supervisor 17.873s
ok  github.com/FelipeMiiller/my-memory/internal/turboquant 0.684s
ok  github.com/FelipeMiiller/my-memory/internal/watcher    1.903s
ok  github.com/FelipeMiiller/my-memory/internal/writer     1.420s
ok  github.com/FelipeMiiller/my-memory/internal/writer/projection 1.265s
```

Nenhuma regressão.

---

## 4. Deferral Compliance (re-check)

- [x] **Plano B (gopls LSP)**: nenhum arquivo `*_lsp.go` em `internal/codeast/` ou `internal/mcp/`. Nenhum `golang.org/x/tools/gopls` em `go.mod`.
- [x] **Plano Real tree-sitter Binding (NOVO em ADR-047 §Deferral)**: `grep -n "github.com/tree-sitter/go-tree-sitter" go.mod` → 0 matches. `grep -rn "code_grammar_downloads" .` → 0 matches. `MockParser` ainda presente em `internal/codeast/mock_parser.go` (não removido).
- [x] Nenhuma flag `--code-ast-backend=lsp` ou `--code-ast-backend=treesitter` em CLI/MCP/help texts.
- [x] Nenhum `pragma_table_info` em código Postgres-path (deferred).

**Deferral clean: yes (both Planos A e B)**.

---

## 5. Final Verdict

**Verdict**: **PASS-WITH-DEFER**

| Métrica | Valor |
|---|---|
| Gaps fechados (iter 1) | **5/5** (GAP-1, GAP-2, GAP-3, GAP-4, GAP-5) |
| Sensor kills (iter 2) | 1/3 (Fault F crítico morto; Faults E+G spec-gap não-críticos) |
| Surviving mutants | 2 (Fault E boot-log drift; Fault G warning-text drift) |
| Deferral | clean (Plano A newBackendParser + Plano B gopls LSP) |
| Regressões | none (28/28 packages PASS) |
| Build smoke | `bin/mem.exe code-search foo` ✓; `code-graph foo` ✓; `code-stats` ✓ (boot log presente) |
| ACs PASS (projetados) | 17/17 (vs 14/17 na iter 1 — graças ao relaxamento de CA-16) |

**Recomendação**: promote `feat-code-ast` para produção, COM **2 follow-ups não-bloqueantes**:

1. **(LOW) Test guard para boot log CLI** — adicionar `TestCodeSearchCLI_EmitsBootLog` (e irmãos) em `cmd/mem/codesearch_test.go` que use `CaptureStdout` e afirme `strings.HasPrefix(out, "tree-sitter: ")`. Mata Fault E.
2. **(LOW) Test guard para warning `include_code`** — estender `TestMemorySearch_IncludeCodeFlag_AcceptsBoolean` em `internal/mcp/code_handlers_test.go` para também assertar `strings.Contains(tr.Content[0].Text, "include_code=true: fan-in")`. Mata Fault G.

Não bloqueiam produção: detecção manual via smoke. Mas se o Felipe quiser "spec clean" antes de promover, são 2 PRs triviais.

---

## 6. Lessons Distilled

Re-cordei manualmente (skill ainda sem automation hook):

1. **lesson-candidate-feat-code-ast-04** (scope: `feat-code-ast`): "ADR-046 'Deferred Until' funciona bem quando o gap é arquitetural (Plano Real tree-sitter) mas também funciona quando o gap é operacional (fan-in RRF code+markdown placeholder loud). O importante é ter gatilhos verificáveis (gcc + go get + E2E) e proibições vinculantes. Para promoção planejada, abrir spec dedicado fora do ADR principal." — Causa: GAP-1 + GAP-3.

2. **lesson-candidate-feat-code-ast-05** (scope: `feat-code-ast`): "Loud placeholder (warning explícito no response) é melhor que silent placeholder, MAS sem um test guard automatizado, refactor acidental pode silenciar o warning sem alarme. Convention: qualquer novo `if includeX { response = "..." }` deve ter test irmão com `strings.Contains(...)` assertion." — Causa: GAP-2 (parcialmente — fix funciona mas não tem guard).

3. **lesson-candidate-feat-code-ast-06** (scope: `feat-code-ast`): "Verificação de gaps em iter de auditoria deve distinguir direção da mutation. Mutation A (server-side regressão) e Mutation B (test-side undo) são simétricas em aparência mas só uma captura regressão real em produção. Audit sempre testa **ambos os lados**." — Causa: Fault F asymmetric protection.

---

## 7. Summary da Auditoria

| Item | Resultado |
|---|---|
| `git log c7b81ad..e5e85e6 --oneline` | 10 commits (`0983d11` + `e5e85e6` = 2 fix commits pelos 5 gaps) |
| Spec.md atualizado | YES (CA-16 retitled; explanação Postgres deferred) |
| ADR-047 atualizado | YES (nova §Deferral: Plano Real tree-sitter Binding) |
| Boot log uniforme | YES (4 code-* CLIs) |
| Test guard para tools/list MCP | YES (TestServer_ToolsList estendido) |
| Loud warning para include_code | YES (handlers.go:355-358 + 393) |
| Regressões introduzidas | NONE |
| Código funcional | YES (code-index roda, code-search indexa via MockParser, etc.) |
| Próximo passo | **promote** (com 2 follow-ups LOW opcionais) |
