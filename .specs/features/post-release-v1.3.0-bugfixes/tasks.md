# Tasks: Post-Release v1.3.0 Bugfixes & Viewer Improvements

**Spec**: [spec.md](./spec.md)
**Date**: 2026-09-17

## Trilha A — Bugs críticos (✅ DONE)

### T1: Fix encoding UTF-8 no `mem graph` template
- **Owner**: Felipe Miiller
- **Status**: ✅ DONE
- **Commit**: `f8d3bba` — fix(graph): A1 UTF-8 encoding defense + A2 path-derived repo slug
- **Where**: `internal/graphview/template.go:14-15`, `internal/graphview/template_test.go::TestRenderHTML_UTF8Encoding`
- **Sub-tarefas**:
  - [x] Adicionar `<meta http-equiv="Content-Type" content="text/html; charset=UTF-8">` no `<head>` (defesa em profundidade)
  - [x] Manter `<meta charset="UTF-8">` existente
  - [x] Garantir bytes UTF-8 literais sem BOM (verificado com hex dump)
  - [x] Escrever `TestRenderHTML_UTF8Encoding` que valida "Memória" / "Síntese" / "Usuário" como bytes UTF-8 literais
  - [x] Recompilar `bin/mem.exe` e validar smoke test

### T2: `mem graph --db <outro>` usa slug derivado do path
- **Owner**: Felipe Miiller
- **Status**: ✅ DONE
- **Commit**: `f8d3bba` (mesmo commit, entrega combinada com T1)
- **Where**: `cmd/mem/main.go::resolveStorageAndRepo`, `cmd/mem/graph_test.go::TestResolveStorageAndRepo_DerivesRepoFromDBPath`
- **Sub-tarefas**:
  - [x] Detectar override de DB: `dbPath != "" && filepath.IsAbs(dbPath)`
  - [x] Derivar slug: `filepath.Base(filepath.Dir(dbPath))` com fallback se vazio/`.`/`/`
  - [x] Preservar `cfg.RepoID` quando path relativo (não quebrar workflows locais)
  - [x] Escrever 4 cenários de teste: path absoluto sem --repo, path relativo sem --repo, path absoluto com --repo explícito, sem --db
  - [x] Smoke test: `mem graph --db 'G:\My Drive\central-memory\memory.db'` (sem --repo) gera HTML com `[central-memory]`

### T3: Cluster-label cleanup
- **Owner**: Felipe Miiller
- **Status**: ✅ DONE
- **Commit**: `a7fe4dd` — feat(viewer): B3 multi-repo via ?src= URL param + A3 cleanup
- **Where**: `graph-v2.html` (CSS linhas 14-31, JS `renderClusterLabels` linhas 327-360, JS `redrawClusterLabels` linhas 450-452)
- **Sub-tarefas**:
  - [x] Remover `.cluster-label { ... }` do bloco CSS
  - [x] Remover `renderClusterLabels()` function
  - [x] Remover `redrawClusterLabels()` wrapper
  - [x] Validar `grep cluster-label graph-v2.html` retorna 0 hits
  - [x] Visual: viewer carrega sem overlays

## Trilha B — Viewer gaps (✅ PARTIAL DONE)

### T4: Type filter chips no header
- **Owner**: Felipe Miiller
- **Status**: ✅ DONE
- **Commit**: `dda9d1b` — feat(viewer): B1 type filter chips + B2 repos registry in ConfigModal
- **Where**: `graph-v2.html` (HTML `<div id="type-chips">`, CSS `.type-chip`, JS bootstrap)
- **Sub-tarefas**:
  - [x] Extrair tipos únicos de `DATA.nodes` no bootstrap
  - [x] Renderizar 1 chip por tipo + chip "Todos" (active default)
  - [x] Click handler: filter via `cy.show()/hide()` em nodes + edges
  - [x] Cor do chip = cor da paleta de nodes
  - [x] Visual: chips aparecem abaixo da stats row

### T5: ConfigModal repos registry (read-only)
- **Owner**: Felipe Miiller
- **Status**: ✅ DONE (read-only, edit/save deferred)
- **Commit**: `dda9d1b` (mesmo commit, entrega combinada com T4)
- **Where**: `graph-v2.html` (HTML seção modal, JS `ConfigModal.refreshRepos()`, `ConfigModal.parseRepos()`)
- **Sub-tarefas**:
  - [x] Adicionar seção "Repositórios Registrados" no modal
  - [x] Parser YAML simples extrai bloco `repositories:[]`
  - [x] Renderizar cards com name/path/id + badge "✓ configured"
  - [x] Botão "↻ Atualizar" re-roda o parser
  - [x] Validar com `config-global.txt` (1 repo registrado)
- **DEFERRED (não entregue)**: textarea editável, botão "Salvar", endpoint `PUT /api/config` (Sessão 2+ ou ADR-037)

### T6: Reverter B3 (multi-repo) — site único
- **Owner**: Felipe Miiller
- **Status**: ✅ DONE
- **Commit**: `2bf79bd` — feat(viewer): single-site + fCoSE layout + text-background labels + hide-tags toggle
- **Where**: `graph-v2.html` (linhas 173-220, novo bootstrap async)
- **Sub-tarefas**:
  - [x] Remover `?src=` URL parsing
  - [x] Hardcoded `const SRC_PARAM = 'data-central.json'`
  - [x] Remover tabs vazios no header
  - [x] Remover `data-my-memory.json` (era cópia duplicada)
  - [x] Documentar reversão em **ADR-038**
  - [x] Atualizar **ADR-036** marcando B3 como "REVERTED em 2bf79bd → ADR-038"

## Quality & Infrastructure (✅ DONE)

### T7: Always-quality-gate rule
- **Owner**: Felipe Miiller
- **Status**: ✅ DONE
- **Commit**: `e77911b` — chore(quality): add always-quality-gate rule + benchmarks for graphview/embedder
- **Where**: `.agents/rules/always-quality-gate.md`, `AGENTS.md` seção 8
- **Sub-tarefas**:
  - [x] Definir 10 itens do gate (test/build/gofmt/bench/smoke/visual/pipeline/git-status/docs/cross-ref)
  - [x] Política explícita de erros (trivial → fix direto; arquitetural → ADR; cosmético → fix)
  - [x] Critério de aceitação final claro
  - [x] Referenciar em AGENTS.md

### T8: Benchmarks em pacotes críticos
- **Owner**: Felipe Miiller
- **Status**: ✅ DONE
- **Commit**: `e77911b` (mesmo commit, combinado com T7)
- **Where**: `internal/graphview/builder_bench_test.go`, `internal/embedder/ollama_bench_test.go`
- **Sub-tarefas**:
  - [x] `BenchmarkBuildGraphView_Small/Medium/Large` (50/500/2000 docs)
  - [x] `BenchmarkRenderHTML` (com acentos PT-BR, cobre A1)
  - [x] `BenchmarkGenerateEmbedding_Mocked` (httptest, 768-dim)
  - [x] `BenchmarkEmbedRequestMarshal`, `BenchmarkEmbedResponseUnmarshal`
  - [x] Baselines registrados via `go test -bench=. -benchtime=10x -run='^$'`
  - [x] `gofmt -w` aplicado nos 2 arquivos novos (trailing newline)

### T9: `.memory/config.yaml` cleanup (pendente v1.3.0)
- **Owner**: Felipe Miiller
- **Status**: ✅ DONE
- **Commit**: `e375e94` — chore(memory): cleanup .memory/config.yaml defaults + slim .env.example
- **Where**: `.memory/.env.example`, `.memory/config.yaml`
- **Sub-tarefas**:
  - [x] Include pattern: trocar lista específica por `**/*.md`
  - [x] Remover `.specs/**` e `.agents/**` do exclude (indexados agora)
  - [x] Remover `storage:` block (defaults cobrem)
  - [x] `vault_name`: "My-Memory Knowledge Base" → "Knowledge Vault"
  - [x] Comments redundantes removidos
  - [x] `mem index` pós-commit confirma novos docs indexados (161 total)

### T10: `.gitignore` patterns + ADR-037
- **Owner**: Felipe Miiller
- **Status**: ✅ DONE
- **Commit**: `597e0a9` — chore(deps): .gitignore smoke artifacts + ADR-037 viewer rewrite com Vite
- **Where**: `.gitignore`, `docs/adr/037-rewrite-viewer-com-vite-vanilla-ts.md`, `AGENTS.md`
- **Sub-tarefas**:
  - [x] `.gitignore` patterns: `node_modules/`, `screenshot-*.png`, `smoke-*.html`, `graph-smoke-*.html`, `graph-test-*.html`, `debug-*.cjs`, `test-*.cjs`
  - [x] ADR-037: rewrite viewer com Vite + Vanilla TS + Web Components (5 sprints, plano completo)
  - [x] Atualizar AGENTS.md adicionando ADR-037 e ADR-038

### T11: ADR-038 — site único
- **Owner**: Felipe Miiller
- **Status**: ✅ DONE
- **Commit**: (próximo batch)
- **Where**: `docs/adr/038-viewer-site-unico-dataset-fixo.md`
- **Sub-tarefas**:
  - [x] Documentar reversão do B3 (single-site é decisão arquitetural)
  - [x] Justificar com feedback do Felipe
  - [x] Considerar 3 opções (multi-repo, single-config, single-hardcoded)
  - [x] Marcar B3 como "REVERTED per ADR-038" em ADR-036
  - [x] Cancelar C2 (backend `--all-repos`) — coberto por ADR-038

## ADRs concluídos (cross-ref spec)

| ADR | Status | Commit |
|-----|--------|--------|
| ADR-036 Trilha A | ✅ Done | f8d3bba + a7fe4dd |
| ADR-036 Trilha B1 | ✅ Done | dda9d1b |
| ADR-036 Trilha B2 | ✅ Partial | dda9d1b (read-only) |
| ADR-036 Trilha B3 | ✅ Reverted → ADR-038 | 2bf79bd |
| ADR-037 (viewer rewrite) | ✅ Spec created | 597e0a9 |
| ADR-038 (site único) | ✅ Done | (próximo batch) |

## Out of Scope (marcados para outras specs)

- Edit-ability do ConfigModal (textarea + PUT) → Sessão 2+ ou ADR-037
- Persistência OpenWith reader → Sessão 2+ ou ADR-037
- Mini-mapa interativo (drag-to-navigate) → ADR-037 C3
- Animações de entrada → ADR-037 C4
- Edge labels polish → ADR-037 C5
- Multi-vault support (se necessário no futuro) → reverter ADR-038 criando ADR-039 com use case concreto