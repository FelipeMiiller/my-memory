---
title: spike-onnxruntime-binding — CGO build + fixture strategy
category: resource
summary: Spike de validação para decidir entre seguir com `yalue/onnxruntime_go` (CGO in-process) ou pivotar pra GGUF/CrispASR (subprocess JSON-RPC) antes de atacar T4 do nemotron-asr-streaming. Decide também estratégia de fixture (~5 MB mock vs 670 MB modelo real) e modelo de deploy (build tag + download lazy). Roda em branch própria, independente do PR #8.
status: draft
tags: [asr, spike, cgo, onnx, fixture, validation, pre-t4]
---

# spike-onnxruntime-binding — CGO build + fixture strategy

> **Tipo:** Spike de validação (não é feature). Decisões tomadas aqui alimentam `feat/asr-nemotron-real` quando reaberto pós-PR #8 merge.

> [!warning] Branch obrigatória: `spike/onnxruntime-binding` (a partir de `develop`)
> O smoke test é um package novo (`internal/asr/spike/`) que importa só `onnxruntime_go` + stdlib — **não tem dependência de T1-T3**. Rodar o spike em cima de `feat/asr-nemotron-real` (head do PR #8) contamina o PR com artefatos do spike e quebra o requisito de independence definido nos Success Criteria.
>
> ```bash
> git checkout develop && git pull
> git checkout -b spike/onnxruntime-binding
> ```
>
> O PR #8 faz merge em qualquer ordem — antes ou depois do spike.

## Problem Statement

T1-T3 de `.specs/features/nemotron-asr-streaming/` (PR #8 `feat/asr-nemotron-real`) estão fechados. T4 (NemotronClient) está bloqueado em três decisões interligadas que precisam evidência empírica antes de gastar tempo implementando o caminho errado:

1. **`yalue/onnxruntime_go` binding compila em Windows + CGO toolchain disponível?** ADR-045 §Decision Outcome assume "mesmo binding do ADR-035 (embedder builtin)" mas o `internal/embedder/ollama.go` (verificado 2026-09-24) usa apenas `internal API` HTTP, não ONNX in-process. **Nenhum teste de smoke existe pra `onnxruntime_go` neste repo até hoje.** Se binding não compila em Windows (gcc ausente, MSVC toolchain incompatível, ou cross-compile issues), ADR-045 cai e o caminho vira GGUF/CrispASR subprocess (ADR-045 Considered Options #2).

2. **Tamanho do modelo int4 (670 MB) é aceitável como fixture de CI?** ADR-045 §Negative flagra o peso. Se o CI não tiver banda/disk pra modelo real, o caminho de teste precisa de fixture mock (~5 MB) e benchmark do modelo real fica em job separado. Decidir ANTES de T4 evita refactor do test harness depois.

3. **Build tag `//go:build nemotron` (planejado em T4) consegue coexistir com `gofmt -l . && go build ./... && go test -count=1 ./...` no CI atual?** O gate do CI (Go module padrão) precisa passar com `nemotron` tag ausente (= binding skipped) E com tag presente (= binding active). Se o binding falhar em uma das 3 plataformas (Linux x86_64, macOS arm64, Windows amd64) o CI trava e bloqueia PRs não-relacionados.

> [!tip] Questão 3 já tem precedente funcionando neste repo
> `.github/workflows/ci.yml` (job `codeast-cgo-on`) roda `CGO_ENABLED=1 go test -tags treesitter` com `continue-on-error: true` e gcc instalado, e o comentário nas linhas 191-196 já prevê exatamente o caso do binding real ("When the real binding lands, the CGO-on job will additionally require gcc..."). O spike não precisa *projetar* o padrão — precisa **copiar** `internal/codeast/treesitter_enabled.go` + o job e trocar a tag. Isso derruba a maior incerteza do plano e reduz o spike de 1-2 dias para meio dia.

Sem essa evidência, atacar T4 vira implementação às cegas. O spike gera 5 entregáveis: (a) `go.mod` adicionando yalue como dep opcional, (b) smoke test que valida binding + ORT environment init, (c) fixture pequena (~5 MB se mock ou link de download do modelo real), (d) verdict em `.specs/features/spike-onnxruntime-binding/validation.md`, (e) amend em ADR-045 §Deferral §Status atual com a evidência.

## Goals

- [ ] Smoke test passa em Windows amd64 (dev machine atual) confirmando binding + ORT init + Session.Load em arquivo `.onnx` válido.
- [ ] Smoke test passa em GitHub Actions Linux (mesma matriz de CI do repo).
- [ ] Decidir estratégia de fixture: `<5 MB mock` vs `link de 670 MB` ou `download lazy on first use`.
- [ ] Decidir build tag policy: `//go:build nemotron` em `internal/asr/nemotron_onnx.go` OU módulo separado em `cmd/mem-asr-server` (analogous ao ADR-042 worker supervisor pattern).
- [ ] Registrar o verdict ("T4 GO" ou "T4 PIVOT TO GGUF/CRISPASR") em `validation.md` do spike + amend em ADR-045 §Deferral.

## Out of Scope

| Item | Reason |
|---|---|
| Streaming inference real (T5 do nemotron-asr-streaming) | Depende das 4 decisões acima. Próximo PR pós-merge. |
| event_runtime integration (T6) | Idem — dependente. |
| HuggingFace downloader (T8) | Caminho GGUF/CrispASR é alternativa; baixar modelo agora amarra a decisão. |
| CLI subcommands (T9) | T8 + T6 primeiro. |
| Cross-platform test matrix (Linux arm64, macOS) | Smoke roda só em Windows (dev) + Linux CI runner. macOS fica para ADR-049 (matriz completa). |

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
|---|---|---|---|
| Binding: `yalue/onnxruntime_go` | `v0.0.0-20231110091024-3df7f9d29187` (versão citada no ADR-035 embedder builtin) | Reuso do mesmo package do ADR-035. **Atenção: pseudo-version de nov/2023 — se for mesmo a latest, são ~3 anos sem tag. Verificar (ver Open Question 4).** | n (validar via `go list -m -versions github.com/yalue/onnxruntime_go`) |
| CGO toolchain em Windows | gcc via MSYS2/MinGW ou strawberryperl | Padrão pra CGO em Windows; verificar `gcc --version` antes de assumir | n |
| Fixture estratégia | Tentar modelo real primeiro (link download em `downloader_test.go`); fallback mock se repo size vira problema | Benchmark do modelo real é mais valioso que mock | n (decidir pós-smoke) |
| ONNX runtime versão | `onnxruntime v1.20.x` ou mais recente | Compatível com Nemotron 3.5 reimplementation | n (validar durante smoke) |
| Build tag placement | `//go:build nemotron` em `internal/asr/nemotron_onnx.go` (arquivo de **T4**, não T2) + job CGO-on no CI | **Precedente já validado no repo:** `ci.yml` job `codeast-cgo-on` (L197-226) roda `CGO_ENABLED=1 go test -tags treesitter` com `continue-on-error: true`. Copiar o pattern, não projetar do zero | n (copiar `internal/codeast/treesitter_enabled.go`) |

**Open questions (resolver no spike):**

1. **Size of `libonnxruntime.so/dll/dylib` bundle?** Se > 100 MB, justifica build tag + download lazy.
2. **CGO_ENABLED no CI?** *Parcialmente respondido pelo precedente:* `ci.yml` já tem job dedicado com `CGO_ENABLED=1` explícito, então não é cenário de "fallback obrigatório". Confirmar apenas se `nemotron` exigir algo além de gcc (ex: headers do ORT).
3. **Modelo Nemotron 3.5 ONNX reimplementation disponível publicamente?** Paper 2604.14493 diz int4 quantization cabe em 670 MB; confirmar se já existe `.onnx` file disponível via HuggingFace `nvidia/nemotron-3.5-asr-streaming-0.6b` ou se precisa portar.
4. **Quão abandonado está o `yalue/onnxruntime_go`?** Se a pseudo-version de nov/2023 for mesmo a latest, auditar supply-chain (open issues, última atividade no repo). Isso pesa direto no verdict: binding CGO sem manutenção + 670 MB de modelo + 3 plataformas é combinação cara de sustentar.
5. **Em qual slot de ADR o verdict é registrado?** Preferência: amend em ADR-045 §Deferral §Status atual (é o mecanismo que o ADR-046 criou exatamente pra isso). Se o amend não couber, o próximo número livre é **054** (052 e 053 já reivindicados por PR #6 e PR #7).

## User Stories

### P1: Smoke test valida o binding ✅ MVP do spike

**User Story**: As a dev, I want um smoke test executável em 30 segundos que confirme "yalue/onnxruntime_go builda + Session.Load funciona em Windows + Linux" so that a decisão de ADR-045 passe de especulação pra evidência.

**Acceptance Criteria:**

1. WHEN `go test -tags nemotron ./internal/asr/spike/...` is invoked THEN the system SHALL load a tiny test ONNX model (≤5 MB, ou link download de modelo real ≤50 MB) and run one inference pass returning without panic. (event-driven)
2. IF the binding fails to compile (missing CGO toolchain) THEN the system SHALL output clear diagnostic `binding load failed: <reason>` and exit 2. (unwanted-behavior)
3. IF ONNX Runtime library fails to initialize THEN the system SHALL output `ORT init failed: <reason>` and exit 3. (unwanted-behavior)
4. WHEN Windows smoke passes THEN the same test SHALL pass on Linux (GitHub Actions) without modification. (ubiquitous)
5. The system SHALL produce a verbose output distinguishing: "binding build OK", "ORT env created", "session loaded", "inference complete" so the next dev reading logs sees exactly where failure occurs. (ubiquitous)

**Independent Test:** `TestSpike_BindingCompilesAndInferenceRuns` mede tempo total (target: <30 s em Windows, <60 s em Linux CI runner com modelo mock).

### P2: Fixture decision (mock vs real vs lazy)

**User Story**: As a dev, I want saber se o repo do my-memory aguenta 670 MB de fixture committed ou se precisa de download on-demand so that test harness seja sustentável em CI sem inflar git history.

**Acceptance Criteria:**

1. IF the chosen fixture is a real model THEN the system SHALL document the download command (`huggingface-cli download ...` ou URL direta) no verdict em `validation.md` do spike e no amend do ADR-045. (optional-feature)
2. IF the chosen fixture is a mock THEN the system SHALL provide a script `scripts/generate-mock-model.sh` that creates a deterministic 5 MB `.onnx` with 1 input / 1 output tensors. (optional-feature)
3. The fixture size decision SHALL be documented with concrete numbers (download time, git impact, CI cache strategy). (ubiquitous)

**Independent Test:** `TestSpike_FixtureStrategyDocumented` valida que `validation.md` do spike existe e tem a seção "Fixture Strategy" com números concretos.

### P3: Build tag coexistence

**User Story**: As a maintainer, I want garantir que `go build ./...` (gate atual do CI) não quebra quando o binding é adicionado e a tag `nemotron` é opcional so that PRs não-relacionados (ex: ADR-044) não sejam bloqueados pela dependência CGO.

**Acceptance Criteria:**

1. WHEN `go build ./...` is invoked without `nemotron` tag THEN the system SHALL skip T4 nemotron_onnx.go via `//go:build nemotron` and exit 0. (unwanted-behavior)
2. WHEN `go test ./internal/asr/...` is invoked without tag THEN the system SHALL use the noop provider stub and pass. (unwanted-behavior)
3. WHEN `go build -tags nemotron ./...` is invoked THEN the system SHALL compile the binding on the current platform. (event-driven)

**Independent Test:** `TestSpike_NoTag_BuildPasses` + `TestSpike_WithTag_BuildsOnCurrentPlatform`.

## Edge Cases

- IF Windows machine has no gcc (e.g., fresh Win install) THEN spike records `gcc not found in PATH` diagnostic and the PR #8 Foundation remains valid (T2 noop provider).
- IF `libonnxruntime.so` not available in standard search paths THEN spike records `library not found, install via brew/apt/scoop` diagnostic.
- IF model fixture URL returns 404 (model removed from HF) THEN spike falls back to mock generator and documents the 404 for follow-up.
- IF the pinned binding version turns out to be stale/archived (Open Question 4) THEN spike records it as evidence and the default verdict becomes **PIVOT** unless a maintained fork is found and documented.

## Requirement Traceability

| Req ID | Story | Status |
|---|---|---|
| SPIKE-01 | P1 | Draft |
| SPIKE-02 | P1 | Draft |
| SPIKE-03 | P1 | Draft |
| SPIKE-04 | P1 | Draft |
| SPIKE-05 | P1 | Draft |
| SPIKE-06 | P2 | Draft |
| SPIKE-07 | P2 | Draft |
| SPIKE-08 | P2 | Draft |
| SPIKE-09 | P3 | Draft |
| SPIKE-10 | P3 | Draft |
| SPIKE-11 | P3 | Draft |
| SPIKE-12 | P1 (OQ-4 supply-chain) | Draft |

## Success Criteria

- [ ] `internal/asr/spike/binding_smoke_test.go` roda em Windows dev machine (target: <30s)
- [ ] Mesmo smoke roda em GitHub Actions Linux runner (target: <60s)
- [ ] Verdict publicado no slot correto de ADR: **amend em `docs/adr/045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md` §Deferral §Status atual** (preferido) **ou** ADR novo numerado **054** se o amend não couber. **NÃO** criar `045-*.md` — o slot 045 está ocupado e o ADR-046 reserva N+1 pro filho.
- [ ] Decisão final registrada em `.specs/features/spike-onnxruntime-binding/validation.md`: "T4 GO" ou "T4 PIVOT TO GGUF/CRISPASR" + rationale + evidência bruta (log do smoke, `go list -m -versions`, tamanho da lib)
- [ ] Seção `§spike-onnxruntime-binding` criada em `.specs/STATE.md` (a seção `§nemotron-asr-streaming` **não existe** — `grep -i "nemotron|asr|spike" .specs/STATE.md` retorna vazio)
- [ ] PR #8 (`feat/asr-nemotron-real`) pode mergear independente do spike (T2 noop stub é suficiente)

## Cross-references

- [`.specs/features/nemotron-asr-streaming/spec.md`](../nemotron-asr-streaming/spec.md) — spec upstream que T4-T9 dependem
- [`.specs/features/nemotron-asr-streaming/tasks.md`](../nemotron-asr-streaming/tasks.md) — T4 tem este spike como pré-requisito
- [`docs/adr/045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md`](../../../docs/adr/045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md) — ADR a ser **amendado** (§Deferral §Status atual) pós-spike
- [`docs/adr/046-deferred-until-padrao-para-fallbacks-opcionais-em-adrs.md`](../../../docs/adr/046-deferred-until-padrao-para-fallbacks-opcionais-em-adrs.md) — convenção que reserva slot N+1 pra ADR filho (proibido repetir `045-`)
- [`docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md`](../../../docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md) — pattern CGO in-process existente (reuso do mesmo binding se viável)
- `.github/workflows/ci.yml` job `codeast-cgo-on` (L197-226) — precedente de job CGO-on com build tag opcional
- `internal/codeast/treesitter_enabled.go` — padrão de stub sob build tag a ser copiado
- PR #8 https://github.com/FelipeMiiller/my-memory/pull/8 — foundation T1/T2/T3 merged-pending

## Outputs (deliverables)

1. `internal/asr/spike/binding_smoke_test.go` — smoke executable
2. `scripts/generate-mock-model.sh` (conditional on P2 mock path) — fixture generator
3. `.specs/features/spike-onnxruntime-binding/validation.md` — verdict final ("T4 GO" ou "PIVOT") + evidência bruta + "Fixture Strategy" com números
4. Amend em `docs/adr/045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md` §Deferral §Status atual (binding status, evidence log, fixture strategy, build tag rationale) — **ou** ADR-**054** novo se o amend não couber
5. `.specs/features/spike-onnxruntime-binding/tasks.md` — breakdown executável do spike, reaproveitando o gate de `.specs/features/nemotron-asr-streaming/tasks.md`
6. Seção `§spike-onnxruntime-binding` em `.specs/STATE.md`

## Estimated Effort

**~meio dia** de trabalho focado, porque P3 (build tag + job CI) copia o precedente `codeast-cgo-on` já validado. O caminho crítico real é P1 (compilar o binding em Windows) + OQ-4 (auditoria de manutenção do `yalue`).

Se o binding falhar em Windows **ou** OQ-4 confirmar bindings abandonado sem fork mantido: o pivot pra GGUF/CrispASR adiciona ~1 semana (decision + integration), e o verdict sai **PIVOT** em vez de GO.
