# Feature: Post-Release v1.3.0 Bugfixes & Viewer Improvements

- **Spec ID**: `post-release-v1.3.0-bugfixes`
- **Created**: 2026-09-17
- **Owner**: Felipe Miiller
- **Source**: derived from [ADR-036](../../docs/adr/036-backlog-pos-release-v1.3.0-e-roadmap-v1.4.0.md)
- **Related specs**: `.specs/features/interactive-html-graph-visualizer/` (v2 visualizer original — superseded por ADR-037)

## Problem Statement

A release **v1.3.0** (PR #4, tag `v1.3.0`, merge em `4c2e204`) foi congelada sem entregar uma série de bugfixes, gaps de UX e melhorias visuais identificados durante o trabalho de consolidação do cofre central. O **ADR-036** catalogou esses itens em três trilhas (A bugs / B viewer / C diferidos), e esta spec transforma as entregas da Trilha A e B1+B2 em **tarefas rastreáveis com critérios de aceitação**.

Itens fora de escopo desta spec:
- B3 multi-repo viewer → **REVERTIDO** pelo feedback ("só um site") → ver [ADR-038](../../docs/adr/038-viewer-site-unico-dataset-fixo.md)
- B4, B5, C3, C4, C5 → **DEFERRED** pra migração Vite → ver [ADR-037](../../docs/adr/037-rewrite-viewer-com-vite-vanilla-ts.md)
- C1 embedder ONNX → **DEFERRED** pra `tlc-spec-driven`
- C2 backend `--all-repos` → **CANCELLED** (coberto por ADR-038)

## Goals

- ✅ Corrigir bugs críticos que comprometem a função principal (`mem graph`, viewer v2).
- ✅ Adicionar chips de filtro por tipo no header do viewer (UX improvement).
- ✅ Adicionar lista de repositórios registrados no ConfigModal (read-only).
- ✅ Limpar código morto no viewer (cluster-label CSS/JS).
- ✅ Atualizar configuração padrão do `.memory/config.yaml` (cleanup pendente v1.3.0).
- ✅ Criar regra de qualidade (`always-quality-gate.md`) pra evitar regressões futuras.

## Out of Scope

- Migração do viewer pra Vite + Vanilla TypeScript → [ADR-037](../../docs/adr/037-rewrite-viewer-com-vite-vanilla-ts.md).
- Adoção do embedder embutido ONNX MiniLM → ADR-035 + `tlc-spec-driven`.
- Edit-ability do ConfigModal (textarea + PUT /api/config) → Sessão 2+.
- Persistência do reader preferido no OpenWith → Sessão 2+.
- Mini-mapa interativo, animações, edge labels polish → Sessão 2+ (ADR-037).
- Investigação do parser bug (wikilinks em EARS notation sendo parseados) → fora desta sprint, marcado como follow-up.

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| A1 encoding: defesa em profundidade | `<meta http-equiv="Content-Type" content="text/html; charset=UTF-8">` adicional ao `<meta charset="UTF-8">` | Proxies/legacy clients podem ignorar uma das duas; belt-and-suspenders | y |
| A2 slug derivation: apenas path absoluto | `filepath.Base(filepath.Dir(dbPath))` quando `filepath.IsAbs(dbPath) == true` | Path relativo preserva config.RepoID (evita quebrar workflows locais) | y |
| A3 cluster-label: remoção total | CSS + JS + redrawClusterLabels | Decisão anterior do Felipe ("não quero cluster centrais, faça um gráfico") | y |
| B1 chips de tipo: paleta hardcoded | `{concept, decision, guide, synthesis, reference, other}` com cores específicas | Cores devem bater com paleta de nodes; sem tema dinâmico (C3 polish deferred) | y |
| B2 repos registry: read-only | YAML parser simples extrai bloco `repositories:[]` | Editor/save requer endpoint HTTP — escopo separado (Sessão 2+) | y |
| Site único vs multi-repo | **Single-dataset `data-central.json`** | Felipe: "só um site, sem multi-repo" → ADR-038 | y |
| Layout default: fCoSE | `nodeRepulsion: 4800, idealEdgeLength: 180, edgeElasticity: 0.45, gravity: 0.25, numIter: 2500, tile: true` | fCoSE é o que a doc oficial Cytoscape recomenda como "first layout" | y |
| Hide-tags toggle: default ON | Esconde `in_degree=0 AND out_degree≤1` | Reduz clutter visual por default; user pode desligar | y |

**Open questions**: nenhuma (todas resolvidas ou logadas acima).

## Requirements

### Trilha A — Bugs críticos

#### A1: Encoding UTF-8 no `mem graph`
- **EARS**: WHEN o usuário executa `mem graph --repo <slug> --out <arquivo>.html`, THE SYSTEM SHALL gerar HTML com `<meta charset="UTF-8">` AND `<meta http-equiv="Content-Type" content="text/html; charset=UTF-8">` AND bytes UTF-8 literais sem BOM.
- **Arquivos**: `internal/graphview/template.go` (linha 14-15), `internal/graphview/template_test.go` (`TestRenderHTML_UTF8Encoding`).
- **Critério de aceitação**: `mem graph` gera HTML onde "Memória" preserva bytes UTF-8 (`C3 B3`), não `MemÃ³ria`. Test unitário assertivo.
- **Severidade**: Média (afeta readability mas não funcionalidade).

#### A2: `mem graph --db <outro>` usa slug derivado do path
- **EARS**: WHEN o usuário executa `mem graph --db <path-absoluto> --out <arquivo>.html` WITHOUT `--repo`, THE SYSTEM SHALL derivar o slug do repo de `filepath.Base(filepath.Dir(dbPath))` AND usá-lo no `title` e no `repository` do GraphView.
- **Arquivos**: `cmd/mem/main.go::resolveStorageAndRepo`, `cmd/mem/graph_test.go` (`TestResolveStorageAndRepo_DerivesRepoFromDBPath`).
- **Critério de aceitação**: `mem graph --db 'G:\My Drive\central-memory\memory.db'` (sem `--repo`) gera HTML com título `[central-memory]`. Paths relativos preservam `cfg.RepoID`.
- **Severidade**: Alta (bloqueia geração pra cofres não registrados).

#### A3: Cluster-label cleanup
- **EARS**: WHEN o arquivo `graph-v2.html` é carregado, THE SYSTEM SHALL NOT renderizar overlays HTML `.cluster-label` (CSS removido, JS `renderClusterLabels()` removido, `redrawClusterLabels()` removido).
- **Arquivos**: `graph-v2.html` (CSS linhas 14-31, JS `renderClusterLabels` e `redrawClusterLabels`).
- **Critério de aceitação**: `grep cluster-label graph-v2.html` retorna 0 hits. Visual: viewer carrega sem "central cluster" overlays.
- **Severidade**: Baixa (código morto, não afeta usuário).

### Trilha B — Viewer gaps (parcial)

#### B1: Type filter chips
- **EARS**: WHEN DATA.nodes contém `type ∈ {concept, decision, guide, synthesis, reference, other}`, THE SYSTEM SHALL renderizar um chip clicável por tipo no header (abaixo da stats row) AND click no chip SHALL filtrar nodes/edges via `cy.show()/hide()`.
- **Arquivos**: `graph-v2.html` (HTML `<div id="type-chips">`, CSS `.type-chip`, JS bootstrap).
- **Critério de aceitação**: viewer mostra 1 chip "Todos" + 1 chip por tipo único presente nos dados. Click filtra visualmente.
- **Severidade**: Média (UX improvement, Felipe prefere clique a texto livre).

#### B2: ConfigModal repos registry (read-only)
- **EARS**: WHEN o usuário clica no botão ⚙ Configurações, THE SYSTEM SHALL renderizar uma seção "Repositórios Registrados" abaixo do YAML grid com name/path/id de cada `repositories:[]` entry do config global AND badge "✓ configured".
- **Arquivos**: `graph-v2.html` (HTML na seção modal, JS `ConfigModal.refreshRepos()`, `ConfigModal.parseRepos()`).
- **Critério de aceitação**: Modal mostra 1+ cards de repos. Parser YAML extrai corretamente mesmo em config mal-formatado.
- **Severidade**: Média (UX improvement, requisito de visibilidade).

### Quality & Infrastructure

#### Q1: Always-quality-gate rule
- **EARS**: WHEN o agente de IA declara uma tarefa pronta, THE SYSTEM SHALL ter rodado silenciosamente o gate completo (test/build/gofmt/benchmarks/smoke/visual/pipeline/git-status/docs).
- **Arquivos**: `.agents/rules/always-quality-gate.md`, `AGENTS.md` (regra 8).
- **Critério de aceitação**: rule existe e referencia itens verificáveis (test/build/gofmt/etc).

#### Q2: Benchmarks em pacotes críticos
- **EARS**: WHEN o pacote `internal/graphview` ou `internal/embedder` é modificado, THE SYSTEM SHALL ter benchmarks atualizados cobrindo o caminho quente (BuildGraphView, RenderHTML, GenerateEmbedding).
- **Arquivos**: `internal/graphview/builder_bench_test.go`, `internal/embedder/ollama_bench_test.go`.
- **Critério de aceitação**: baselines registrados (BuildGraphView_Small=1.04ms, RenderHTML=1.26ms, GenerateEmbedding_Mocked=588μs).

#### Q3: `.memory/config.yaml` cleanup (pendente v1.3.0)
- **EARS**: WHEN o vault é inicializado, THE SYSTEM SHALL usar `**/*.md` como include pattern (não lista hardcoded) AND remover `.specs/**` e `.agents/**` do exclude (specs e skills agora são indexados).
- **Arquivos**: `.memory/config.yaml`, `.memory/.env.example`.
- **Critério de aceitação**: config simplificado, `mem status` mostra ~160 arquivos indexados vs ~80 com config antigo.

## Acceptance Criteria (consolidado)

### Funcional
- [x] `mem graph` produz HTML com encoding UTF-8 correto (A1)
- [x] `mem graph --db <abs> --out` produz HTML com título derivado do path (A2)
- [x] Viewer carrega sem overlays de cluster (A3)
- [x] Viewer mostra chips de tipo no header (B1)
- [x] ConfigModal mostra lista de repos (B2)
- [x] Viewer é single-site (data-central.json fixo, sem ?src=) — revertido via ADR-038

### Quality
- [x] `go test -count=1 ./...` exit 0, 19 packages
- [x] `go build -v -o bin/mem.exe ./cmd/mem` exit 0, 23.86 MB
- [x] `gofmt -l .` retorna vazio
- [x] Benchmarks baselines registrados em graphview + embedder
- [x] Quality gate rule (`.agents/rules/always-quality-gate.md`) existe e é referenciada em AGENTS.md

### Visual (Playwright)
- [x] Viewer renderiza sem JS error overlay
- [x] Title: "Grafo de Memória [central-memory]"
- [x] Stats: 52 nodes / 68 edges / 15 hubs / density 0.026 / modularity 0.495
- [x] Hubs (amarelo) com text-background labels visíveis
- [x] Diferentes comunidades coloridas (yellow/purple/pink/blue/green)
- [x] Edges com arrows, layout legível
- [x] Toggle "Esconder tags" funciona (default ON)
- [x] Mini-mapa mostra contexto completo

### ADRs concluídos (cross-ref)

- [x] **ADR-036 Trilha A** (A1, A2, A3) — entregue
- [x] **ADR-036 Trilha B1** (B1) — entregue
- [x] **ADR-036 Trilha B2** (B2 read-only) — entregue parcial
- [~] **ADR-036 Trilha B3** (multi-repo) — REVERTIDO por feedback → **ADR-038**
- [x] **ADR-037** (viewer rewrite Vite) — spec criada, implementação deferred
- [x] **ADR-038** (site único) — entregue

## ADR Mapping

| Entrega | ADR referenciada | Status |
|---------|-----------------|--------|
| A1 encoding UTF-8 | ADR-036 §A1 | ✅ Done (commit f8d3bba) |
| A2 path-derived slug | ADR-036 §A2 | ✅ Done (commit f8d3bba) |
| A3 cluster-label cleanup | ADR-036 §A3 | ✅ Done (commit a7fe4dd) |
| B1 type chips | ADR-036 §B1 | ✅ Done (commit dda9d1b) |
| B2 repos registry (read-only) | ADR-036 §B2 | ✅ Partial Done (commit dda9d1b) |
| B3 revert + site único | **ADR-038** (replaces ADR-036 §B3) | ✅ Done (commit 2bf79bd) |
| Single-site + fCoSE visual | **ADR-038** | ✅ Done (commit 2bf79bd) |
| Quality gate rule | New process rule (.agents/rules/always-quality-gate.md) | ✅ Done (commit e77911b) |
| Benchmarks graphview/embedder | New process rule | ✅ Done (commit e77911b) |
| .memory cleanup | ADR-016 (config declarative, cleanup deferred) | ✅ Done (commit e375e94) |
| Viewer rewrite plan (deferred) | **ADR-037** | ✅ Spec (commit 597e0a9) |

## References

- [ADR-036: Consolidação Pós-Release v1.3.0 e Roadmap v1.4.0](../../docs/adr/036-backlog-pos-release-v1.3.0-e-roadmap-v1.4.0.md) — spec origin
- [ADR-037: Rewrite do Viewer com Vite + Vanilla TypeScript](../../docs/adr/037-rewrite-viewer-com-vite-vanilla-ts.md) — viewer future
- [ADR-038: Viewer com Site Único e Dataset Fixo](../../docs/adr/038-viewer-site-unico-dataset-fixo.md) — single-site decision
- [ADR-035: Embedder Embutido com Fallback ONNX MiniLM](../../docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md) — C1 deferred
- [PR #4 — v1.3.0 release](https://github.com/FelipeMiiller/my-memory/pull/4) — baseline
- [docs/CENTRAL_VAULT.md](../../docs/CENTRAL_VAULT.md) — cofre central doc