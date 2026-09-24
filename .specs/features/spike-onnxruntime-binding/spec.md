---
title: spike-onnxruntime-binding — CGO build + fixture strategy
category: resource
summary: Spike de validação para decidir entre seguir com `yalue/onnxruntime_go` (CGO in-process) ou pivotar pra GGUF/CrispASR (subprocess JSON-RPC) antes de mergear a foundation de PR #8 e atacar T4 do nemotron-asr-streaming. Decide também estratégia de fixture (~5 MB mock vs 670 MB modelo real) e modelo de deploy (build tag + download lazy).
status: draft
tags: [asr, spike, cgo, onnx, fixture, validation, pre-t4]
---

# spike-onnxruntime-binding — CGO build + fixture strategy

> **Tipo:** Spike de validação (não é feature). Decisões tomadas aqui alimentam `feat/asr-nemotron-real` quando reaberto pós-PR #8 merge.

## Problem Statement

T1-T3 de `.specs/features/nemotron-asr-streaming/` (PR #8 `feat/asr-nemotron-real`) estão fechados. T4 (NemotronClient) está bloqueado em três decisões interligadas que precisam evidência empírica antes de gastar tempo implementando o caminho errado:

1. **`yalue/onnxruntime_go` binding compila em Windows + CGO toolchain disponível?** ADR-045 §Decision Outcome assume "mesmo binding do ADR-035 (embedder builtin)" mas o `internal/embedder/ollama.go` (verificado 2026-09-24) usa apenas `internal API` HTTP, não ONNX in-process. **Nenhum teste de smoke existe pra `onnxruntime_go` neste repo até hoje.** Se binding não compila em Windows (gcc ausente, MSVC toolchain incompatível, ou cross-compile issues), ADR-045 cai e o caminho vira GGUF/CrispASR subprocess (ADR-045 Considered Options #2).

2. **Tamanho do modelo int4 (670 MB) é aceitável como fixture de CI?** ADR-045 §Negative flagra o peso. Se o CI não tiver banda/disk pra modelo real, o caminho de teste precisa de fixture mock (~5 MB) e benchmark do modelo real fica em job separado. Decidir ANTES de T4 evita refactor do test harness depois.

3. **Build tag `//go:build nemotron` (planejado em T2) consegue coexistir com `gofmt -l . && go build ./... && go test -count=1 ./...` no CI atual?** O gate do CI (Go module padrão) precisa passar com `nemotron` tag ausente (= binding skipped) E com tag presente (= binding active). Se o binding falhar em uma das 3 plataformas (Linux x86_64, macOS arm64, Windows amd64) o CI trava e bloqueia PRs não-relacionados.

Sem essa evidência, atacar T4 vira implementação às cegas. O spike dura 1-2 dias e gera 4 entregáveis: (a) `go.mod` adicionando yalue como dep opcional, (b) smoke test que valida binding + ORT environment init, (c) fixture pequena (~5 MB se mock ou link de download do modelo real), (d) relatório de viabilidade escrito em `docs/adr/045-deferred-until-binding-validated.md` (companion to ADR-045).

## Goals

- [ ] Smoke test passa em Windows amd64 (dev machine atual) confirmando binding + ORT init + Session.Load em arquivo `.onnx` válido.
- [ ] Smoke test passa em GitHub Actions Linux (mesma matriz de CI do repo).
- [ ] Decidir estratégia de fixture: `<5 MB mock` vs `link de 670 MB` ou `download lazy on first use`.
- [ ] Decidir build tag policy: `//go:build nemotron` em `internal/asr/nemotron_onnx.go` OU módulo separado em `cmd/mem-asr-server` (analogous ao ADR-042 worker supervisor pattern).
- [ ] Produzir ADR-045 companion documentando a decisão + plano de mitigação se binding falhar.

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
| Binding atual: `yalue/onnxruntime_go` last release | try `v0.0.0-20231110091024-3df7f9d29187` (versão do ADR-035 embedder builtin) | Reuso do mesmo package do ADR-035 | n (validar via `go get`) |
| CGO toolchain em Windows | gcc via MSYS2/MinGW ou strawberryperl | Padrão pra CGO em Windows; verificar `gcc --version` antes de assumir | n |
| Fixture estratégia | Tentar modelo real primeiro (link download em `downloader_test.go`); fallback mock se repo size vira problema | Benchmark do modelo real é mais valioso que mock | n (decidir pós-smoke) |
| ONNX runtime versão | `onnxruntime v1.20.x` ou mais recente | Compatível com Nemotron 3.5 reimplementation | n (validar durante smoke) |
| Build tag placement | `//go:build nemotron` em T4 file (T2 nemotron_onnx.go) + CI matrix com + tag | Paralelo ao pattern `//go:build treesitter` do feat-code-ast | n |

**Open questions (resolver no spike):**

1. **Size of `libonnxruntime.so/dll/dylib` bundle?** Se > 100 MB, justifica build tag + download lazy.
2. **CGO_ENABLED default em CI?** Se CI roda com `CGO_ENABLED=0`, fallback vira obrigatório.
3. **Modelo Nemotron 3.5 ONNX reimplementation disponível publicamente?** Paper 2604.14493 diz int4 quantization cabe em 670 MB; confirmar se já existe `.onnx` file disponível via HuggingFace `nvidia/nemotron-3.5-asr-streaming-0.6b` ou se precisa portar.

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

1. IF the chosen fixture is a real model THEN the system SHALL document the download command (`huggingface-cli download ...` ou URL direta) in `docs/adr/045-deferred-until-binding-validated.md`. (optional-feature)
2. IF the chosen fixture is a mock THEN the system SHALL provide a script `scripts/generate-mock-model.sh` that creates a deterministic 5 MB `.onnx` with 1 input / 1 output tensors. (optional-feature)
3. The fixture size decision SHALL be documented with concrete numbers (download time, git impact, CI cache strategy). (ubiquitous)

**Independent Test:** `TestSpike_FixtureStrategyDocumented` valida que `docs/adr/045-deferred-until-binding-validated.md` existe e tem a seção "Fixture Strategy".

### P3: Build tag coexistence

**User Story**: As a maintainer, I want garantir que `go build ./...` (gate atual do CI) não quebra quando o binding é adicionado e a tag `nemotron` é opcional so that PRs não-relacionados (ex: ADR-044) não sejam bloqueados pela dependência CGO.

**Acceptance Criteria:**

1. WHEN `go build ./...` is invoked without `nemotron` tag THEN the system SHALL skip T4 nemotron_onnx.go via `//go:build nemotron` and exit 0. (unwanted-behavior)
2. WHEN `go test ./internal/asr/...` is invoked without tag THEN the system SHALL use the noop provider stub and pass. (unwanted-behavior)
3. WHEN `go build -tags nemotron ./...` is invoked THEN the system SHALL compile the binding on the current platform. (event-driven)

**Independent Test:** `TestSpoke_NoTag_BuildPasses` + `TestSpike_WithTag_BuildsOnCurrentPlatform`.

## Edge Cases

- IF Windows machine has no gcc (e.g., fresh Win install) THEN spike records `gcc not found in PATH` diagnostic and the PR #8 Foundation remains valid (T2 noop provider).
- IF `libonnxruntime.so` not available in standard search paths THEN spike records `library not found, install via brew/apt/scoop` diagnostic.
- IF model fixture URL returns 404 (model removed from HF) THEN spike falls back to mock generator and documents the 404 for follow-up.

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

## Success Criteria

- [ ] `internal/asr/spike/binding_smoke_test.go` roda em Windows dev machine (target: <30s)
- [ ] Mesmo smoke roda em GitHub Actions Linux runner (target: <60s)
- [ ] `docs/adr/045-deferred-until-binding-validated.md` publicado com: status do binding, fixture strategy, build tag policy, e link pra este spec
- [ ] Decisão final registrada em `STATE.md` §nemotron-asr-streaming: "T4 GO" ou "T4 PIVOT TO GGUF/CRISPASR" + rationale
- [ ] PR #8 (`feat/asr-nemotron-real`) pode mergear independente do spike (T2 noop stub é suficiente)

## Cross-references

- [`.specs/features/nemotron-asr-streaming/spec.md`](../nemotron-asr-streaming/spec.md) — spec upstream que T4-T9 dependem
- [`.specs/features/nemotron-asr-streaming/tasks.md`](../nemotron-asr-streaming/tasks.md) — T4 tem este spike como pré-requisito
- [`docs/adr/045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md`](../../../docs/adr/045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md) — ADR a ser confirmado/atualizado pós-spike
- [`docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md`](../../../docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md) — pattern CGO in-process existente (reuso do mesmo binding se viável)
- PR #8 https://github.com/FelipeMiiller/my-memory/pull/8 — foundation T1/T2/T3 merged-pending

## Outputs (deliverables)

1. `internal/asr/spike/binding_smoke_test.go` — smoke executable
2. `scripts/generate-mock-model.sh` (conditional on P2 mock path) — fixture generator
3. `docs/adr/045-deferred-until-binding-validated.md` — decision record + fixture strategy + build tag rationale
4. `STATE.md` update §nemotron-asr-streaming com verdict final ("T4 GO" ou "PIVOT")

## Estimated Effort

1-2 dias de trabalho focado (assumindo binding compila out-of-the-box). Se binding falhar em Windows: pivot pra GGUF/CrispASR adiciona 1 semana (decision + integration).
