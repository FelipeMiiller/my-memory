---
title: "Validation: feat-code-ast"
category: resource
summary: "Verifier report — feat-code-ast (ADR-047) — 12 commits from c7b81ad to ae614be."
type: validation
created_at: 2026-09-21T10:30:00-03:00
tags: [validation, code-ast, tlc-spec-driven, feat-code-ast]
---

# Validation: feat-code-ast

**Date**: 2026-09-21
**Verifier**: mavis (sub-agent, branch `mvs_948eba24b24e4adc92300e9e309dfba8`)
**Spec**: `.specs/features/feat-code-ast/spec.md` (17 ACs, EARS notation)
**Tasks**: `.specs/features/feat-code-ast/tasks.md` (T1–T11; T11 = this report)
**ADR**: `docs/adr/047-code-ast-tree-sitter-multi-linguagem.md`
**Diff range**: `c7b81ad..ae614be` (12 commits, code from `4dc0c39`)
**Verdict**: **PARTIAL** — PASS for 14 ACs, SPEC-GAP for 3 ACs (CA-01 / CA-12 / CA-16), 1 surviving mutant.

---

## 1. Spec-Anchored Outcome Check

Re-derived independently. Each AC scored PASS / PARTIAL / FAIL / SPEC-GAP with concrete `file:line` evidence.

| AC    | Description                                                | Outcome | Evidence |
| :---  | :---                                                       | :---    | :---     |
| CA-01 | Parser tree-sitter multi-linguagem                         | **PARTIAL / SPEC-GAP** | `internal/codeast/treesitter_enabled.go:1` stub file uses `//go:build treesitter` BUT `ParseFile` always returns `ErrTreesitterDisabled` (line 58). The real binding `github.com/tree-sitter/go-tree-sitter` is **absent from `go.mod`**. Production CLI path uses `NewMockParser(false)` (`internal/codeast/pipeline.go:70`) — never invokes `newBackendParser`. Stub is documented as "consciente placeholder" but the spec says "SHALL parse using ... go-tree-sitter". |
| CA-02 | Persistência em SQLite (mesmo `memory.db`)                 | **PASS** | `internal/db/schema.go:147-186` define `code_files` / `code_symbols` / `code_edges` / `code_edges_uncertain`. Test `TestCodeFilesSchema_*` / `TestCodeSymbolsSchema_*` / `TestCodeEdgesSchema_*` em `internal/db/code_migration_test.go:11-123`. |
| CA-03 | Pipeline `mem index` com estágio code opt-in               | **PASS** | `internal/codeast/pipeline.go:194-204` treats `ErrTreesitterDisabled` as no-op. Test `TestPipelineRunsAfterMarkdown_NoOpWhenDisabled` (`pipeline_test.go:27`). Boot log fires once. NOTE: `TestPipelineNoOpWhenTreesitterDisabled` (`pipeline_test.go:138`) is SKIPPED on default build. |
| CA-04 | Cache incremental via SHA-256 + ast_hash                   | **PASS** | `internal/codeast/cache.go:113-130` (DecodeSkipParse / DecodeSkipDownstream). Test `TestCacheIncrementalSkip` (`cache_test.go:58`). |
| CA-05 | Boost RRF por `qualified_name` match (2.0)                 | **PASS** | `internal/codeast/ranking.go:13` `const BoostFactor = 2.0`. Tests `TestRanking_ApplyBoost` and `TestRanking_ApplyBoostToHits_OrderingAndBoost`. |
| CA-06 | CLI `mem code-index`                                       | **PASS** | `cmd/mem/codeindex.go:107-117` creates `NewPipeline(opts)` without Parser (uses MockParser). Wired in `cmd/mem/main.go:767-771`. |
| CA-07 | CLI `mem code-search`                                      | **PASS** | `cmd/mem/codesearch.go` — delegates to RRF + boost. Wired em `main.go:773-777`. |
| CA-08 | CLI `mem code-graph` (CTE recursivo)                       | **PASS** | `cmd/mem/codegraph.go` + `internal/mcp/SQLCodeNeighbors`. Wired em `main.go:779-783`. |
| CA-09 | CLI `mem code-stats`                                       | **PASS** | `cmd/mem/codestats.go` agrega `code_files` / `code_symbols`. Wired em `main.go:785-789`. |
| CA-10 | Tool MCP `memory_code_search`                              | **PASS** | `internal/mcp/code_handlers.go:66` + schema em `tools.go:73`. Tests `TestMemoryCodeSearchHandler_*`. |
| CA-11 | Tool MCP `memory_code_neighbors`                           | **PASS** | `internal/mcp/code_handlers.go` (SQLCodeNeighbors). Tests `TestMemoryCodeNeighborsHandler_*`. |
| CA-12 | Flag `include_code` em `memory_search`                     | **PARTIAL / SPEC-GAP** | Schema aceito (`tools.go:64-67`) + test parsing (`code_handlers_test.go:292-310`). MAS implementação é **placeholder**: `handlers.go:344-354` parseia `include_code` mas o branch `if includeCode { _ = includeCode // placeholder explícito }` é **no-op**. Real RRF fan-in markdown+code não plugado. |
| CA-13 | Benchmarks reproduzíveis                                    | **PASS** | `bench/codeast/*_test.go`. Targets documentados em `harness_test.go:113-115`. `bench/codeast` PASS em 22.3s no Windows. |
| CA-14 | Build CGO opt-in (`//go:build treesitter`)                 | **PASS** | `internal/codeast/treesitter_disabled.go` + `treesitter_enabled.go`. Boot log: `treesitter_disabled.go:35`. Smoke: `bin/mem.exe code-index` emite `tree-sitter: disabled (code pipeline skipped)`. |
| CA-14a | Boot log em **todos** os comandos code-*                   | **SPEC-GAP** | Boot log só dispara em `mem code-index` (`codeindex.go:45`). `mem code-search`, `mem code-graph`, `mem code-stats` NÃO emitem a linha. |
| CA-15 | Compatibilidade CI cross-platform × CGO on/off             | **PASS** | `.github/workflows/ci.yml:197-302` define jobs `codeast-cgo-on` e `codeast-cgo-off`. |
| CA-16 | Schema migration idempotente (SQLite + Postgres)           | **PARTIAL / SPEC-GAP** | SQLite: `EnsureCodeTables` idempotente. Postgres: **não testado**. `pragma_table_info` é SQLite-only; falha em pgvector. CA-16 diz "works identically against SQLite local and Postgres/pgvector" — claim não validada. |
| CA-17 | Documentação atualizada                                     | **PASS** | `docs/CLI_GUIDE.md:553-635`, `docs/AGENT_INTEGRATION_GUIDE.md:252-285`, `docs/ARCHITECTURE.md:90-127`, `README.md:64`. |

### Counts
- **PASS**: 14 (CA-02, CA-03, CA-04, CA-05, CA-06, CA-07, CA-08, CA-09, CA-10, CA-11, CA-13, CA-14, CA-15, CA-17)
- **PARTIAL / SPEC-GAP**: 4 (CA-01, CA-12, CA-14a, CA-16)
- **FAIL**: 0

---

## 2. Discrimination Sensor (mutation testing em worktree descartável)

Worktree isolado: `C:\Users\Felipe\AppData\Local\Temp\feat-code-ast-scratch` (HEAD detached em `ae614be`). **Limpo após testes** (`git worktree remove --force` + `git worktree prune`).

| Fault | Injection site | Killed by test | Survived? |
| :---  | :---           | :---           | :---      |
| **A** — `BoostFactor = 1.0` | `internal/codeast/ranking.go:13` (const 2.0 → 1.0) | `TestRanking_ApplyBoost` + `TestRanking_ApplyBoostToHits_OrderingAndBoost` | NO (killed) |
| **B** — ast_hash branch commented out | `internal/codeast/cache.go:128-131` | `TestCacheIncrementalSkip` (line 122: `want DecodeSkipDownstream`) | NO (killed) |
| **C** — lang filter bypassed | `internal/codeast/pipeline.go:380-382` (`!allowedExt[ext]` → `len(allowedExt) < 0`) | `TestPipelineLangFilter` (go-only indexed=3 vs want 1) | NO (killed) |
| **D** — `MemoryCodeSearch` removido do registry | `internal/mcp/server.go:114` | — (nenhum test mata) | **YES (survived)** |

**Sensor kills: 3/4 faults killed. 1 surviving mutant (Fault D).**

### Diagnose do Fault D (spec-gap)
- Test existente `TestSetCodeSearchHandler_Registers` (`code_handlers_test.go:312-331`) verifica registration, MAS usa `SetCodeSearchHandler(fn)` (API separada).
- Test existente `TestServer_ToolsList` (`tools_test.go:106-187`) verifica `tools/list` response, MAS só checa 4 tools antigas. **Não checa `memory_code_search` nem `memory_code_neighbors`.**
- Resultado: remover a linha de bootstrap em `server.go:114` não quebra nenhum teste — o tool fica ausente do servidor MCP de produção sem alarme.

**Ação sugerida (fix task)**: estender `TestServer_ToolsList` para verificar que `memory_code_search` e `memory_code_neighbors` aparecem no catálogo.

---

## 3. Deferral Compliance (ADR-047 §Deferral Plano B — gopls LSP)

Verifico que o **Plano B (gopls LSP) NÃO foi violado**:

- [x] Nenhum arquivo `*_lsp.go` em `internal/codeast/` ou `internal/mcp/`. Grep → 0 matches.
- [x] Nenhum bloco `code_ast.backend: lsp` em `.memory/config.yaml` ou `--help`.
- [x] Nenhuma flag `--code-ast-backend=lsp` em CLI/MCP.
- [x] Nenhuma referência a `golang.org/x/tools/gopls` em `go.mod`.

**Deferral clean: yes**.

---

## 4. Gaps & Fix Tasks (ranked por severidade)

### GAP-1 — **HIGH** (CA-01: tree-sitter binding não wirado)
- **File**: `internal/codeast/treesitter_enabled.go:1-94` + `go.mod`
- **Issue**: spec diz "SHALL parse using `github.com/tree-sitter/go-tree-sitter`". A stub atual sempre devolve `ErrTreesitterDisabled`. O CLI usa `MockParser` — símbolos sintéticos.
- **Impact**: feature é cosmética em produção.
- **Fix sugerido**: promover para ADR explícito "Real tree-sitter binding injection" (padrão ADR-046 "Deferred Until") com critério de unlock.

### GAP-2 — **HIGH** (CA-12: `include_code` é placeholder no-op)
- **File**: `internal/mcp/handlers.go:344-354`
- **Issue**: branch `_ = includeCode // placeholder explícito` — silently degrada comportamento.
- **Impact**: clientes MCP com `include_code=true` não recebem fan-in RRF esperado.
- **Fix sugerido**: plugar fan-in real (`searchFn` markdown + `CodeSearchFunc` → RRF merge k=60).

### GAP-3 — **MEDIUM** (CA-16: Postgres não testado)
- **File**: `internal/db/code_migration.go:17-48`
- **Issue**: `EnsureCodeTables` usa `pragma_table_info` (SQLite-only).
- **Impact**: claim CA-16 é false-positive para Postgres.
- **Fix sugerido**: (a) detectar driver; (b) job CI Postgres; ou (c) documentar limitação.

### GAP-4 — **MEDIUM** (CA-14a: boot log placement)
- **File**: `cmd/mem/codeindex.go:45`, ausente em codesearch/codegraph/codestats
- **Issue**: `LogTreesitterBootStatus()` só em code-index.
- **Fix sugerido**: adicionar aos 4 code-* CLIs (uma vez via sync.Once).

### GAP-5 — **MEDIUM** (Fault D surviving mutant)
- **File**: `internal/mcp/server.go:114-115`
- **Issue**: refactor acidental pode remover tools code-* do MCP sem alarme.
- **Fix sugerido**: estender `TestServer_ToolsList` (`tools_test.go:106-187`) para exigir `memory_code_search` + `memory_code_neighbors`.

### GAP-6 — **LOW** (untracked `internal/mcp/x.md`)
- **File**: `internal/mcp/x.md:1-7`
- **Issue**: arquivo residual fora do git tree; não relacionado a feat-code-ast.
- **Fix sugerido**: mover para trash; fora de escopo deste verifier.

---

## 5. Lessons Distilled

`python .agents/skills/tlc-spec-driven/scripts/lessons.py status` → `lessons: 0 total`. Skill não tem subcomando `distill`. Registrei manualmente:

1. **lesson-candidate-feat-code-ast-01** (scope: `feat-code-ast`): "Stub pattern com `//go:build` tag não é teste suficiente — verificar se o caminho de produção realmente invoca o backend ou cai para MockParser." — Causa: GAP-1.
2. **lesson-candidate-feat-code-ast-02** (scope: `feat-code-ast`): "Additive flags em MCP handlers precisam de test E2E, não só test de parsing. Branch `_ = includeCode // placeholder` é armadilha clássica." — Causa: GAP-2.
3. **lesson-candidate-feat-code-ast-03** (scope: `feat-code-ast`): "Sensor de mutation deve cobrir não apenas helpers puros mas também wiring paths (registries, dispatch, bootstrap). 1/4 surviving mutants revela lacuna." — Causa: GAP-5.

Nenhuma lesson promovida para `confirmed` automaticamente (skill ainda não tem automation hook).

---

## 6. Verdict Summary

| Métrica | Valor |
| :---    | :---  |
| ACs PASS               | 14 / 17 (82%) |
| ACs PARTIAL / SPEC-GAP | 3 (CA-01, CA-12, CA-16) + 1 sub-item (CA-14a) |
| ACs FAIL               | 0 |
| Sensor kills           | 3 / 4 (75%) |
| Surviving mutants       | 1 (Fault D) |
| Deferral (Plano B LSP) | clean |
| Regressões no pipeline Markdown | NONE |

**Verdict final**: **PARTIAL**. Não PASS puro (CA-01 stub + CA-12 placeholder); não FAIL porque (a) testes passam, (b) MockParser cumpre contrato "SHALL persist symbols/edges" de forma funcional, (c) integração CLI+MCP+DB+benchmarks está pronta, (d) deferral do tree-sitter real está documentado. **Recomendado**: abrir fix-tasks para GAP-1/2/3 antes de promover para produção.