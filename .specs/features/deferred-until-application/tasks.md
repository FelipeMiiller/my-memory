# deferred-until-application Tasks

## Execution Protocol (MANDATORY — do not skip)

Implement these tasks with the `tlc-spec-driven` skill: **activate it by name and follow its Execute flow and Critical Rules.** Do not search for skill files by filesystem path.

**If the skill cannot be activated, STOP and tell the user — do not proceed without it.**

---

**Spec**: `.specs/features/deferred-until-application/spec.md`
**Design**: [ADR-046](../../../docs/adr/046-deferred-until-padrao-para-fallbacks-opcionais-em-adrs.md) (metodologia), [ADR-035](../../../docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md), [ADR-042](../../../docs/adr/042-nucleo-local-mymemoryd-com-workers-sidecar-isolados.md) (alvos)
**Status**: Draft → Approved → In Progress → Done

---

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| --- | --- | --- | --- | --- |
| Markdown documentation | none | Manual review + grep regression tests | `.agents/skills/deferred-until-checklist/SKILL.md` (optional) or in-test grep | `bash test/check_deferred_prose.sh` (T5) |

This spec is documentation-only; tests are grep-based regression guards.

---

## Gate Check Commands

| Gate Level | When to Use | Command |
| --- | --- | --- |
| Quick | After documentation tasks (T1, T2, T3) | `go test -count=1 ./...` (no Go change expected; sanity check) |
| Full | After grep regression task (T5) | `go test -count=1 ./... && bash test/check_deferred_prose.sh` |
| Build | After spec | `gofmt -l . && go build ./... && go test -count=1 ./...` |

---

## Execution Plan

### Phase 1: ADR retrofit (documentation only)

Apply `## Deferral` to ADR-035 + ADR-042, update README tags.

```
(no inter-task deps — all can be parallel)
```

Total: **5 tasks in 1 phase**. Documentation-only; can be done in single batch.

---

## Task Breakdown

### Phase 1

### T1: Apply `## Deferral` to ADR-035

**What**: Open `docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md`. Replace "Fallback ONNX MiniLM" prose with `## Deferral: <Plano B>` section following [ADR-046 §Application Template](../../../docs/adr/046-deferred-until-padrao-para-fallbacks-opcionais-em-adrs.md). Enumerate 3 gatilhos (e.g., ONNX binding regressão, MiniLM descontinuado, restrição contratual), 5+ proibições (arquivos, configs, flags), 5 steps de promoção, status subseção.
**Where**: `docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md` (modify)
**Depends on**: None
**Reuses**: ADR-046 §Application Template copy/paste.
**Requirement**: DEFR-01, DEFR-02, DEFR-03, DEFR-04, DEFR-05, DEFR-06
**Status**: Pending

**Tools**: MCP: NONE | Skill: NONE

**Done when**:

- [ ] `## Deferral: <Plano B>` section added to ADR-035 (between `## Decision Outcome` and `## Consequences`)
- [ ] 3 gatilhos enumerated with verification + evidence requirements
- [ ] 5+ proibições enumerated (arquivos, configs, flags específicas ao embedder fallback)
- [ ] 5 numbered steps in `### Como promover (quando um gatilho dispara)`
- [ ] `### Status atual` subseção with `Última revisão: 2026-09-19`, `Gatilhos disparados: 0`, `Evidence log: nenhum ainda`, `Próxima revisão`
- [ ] Cross-link `[ADR-046](../046-deferred-until-padrao-para-fallbacks-opcionais-em-adrs.md)` present

**Tests**: grep-based (T5)
**Gate**: manual review with checklist
**Commit**: `docs(adr): 035 — apply ## Deferral section per ADR-046`

---

### T2: Apply `## Deferral: Python Sidecar ASR` to ADR-042

**What**: Open `docs/adr/042-nucleo-local-mymemoryd-com-workers-sidecar-isolados.md`. Add `## Deferral: Python Sidecar ASR` section clarifying ASR Python sidecar is deferred per ADR-045; TTS/LLM sidecars remain active. Cross-link to [ADR-045](../../../docs/adr/045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md) (first application) and [ADR-046](../../../docs/adr/046-deferred-until-padrao-para-fallbacks-opcionais-em-adrs.md) (metodologia).
**Where**: `docs/adr/042-nucleo-local-mymemoryd-com-workers-sidecar-isolados.md` (modify)
**Depends on**: None
**Reuses**: ADR-046 §Application Template.
**Requirement**: DEFR-07, DEFR-08, DEFR-09, DEFR-10, DEFR-11
**Status**: Pending

**Done when**:

- [ ] `## Deferral: Python Sidecar ASR` section added to ADR-042
- [ ] Cross-link `[ADR-045](../045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md)` present
- [ ] Cross-link `[ADR-046](../046-deferred-until-padrao-para-fallbacks-opcionais-em-adrs.md)` present
- [ ] NO `## Deferral` for TTS or LLM sidecars (per ADR-042 §Decision Outcome)
- [ ] 5+ ASR-specific proibições enumerated (`internal/voice/asr_worker.py`, `--asr-engine=python`, `pywhispercpp` em `pyproject.toml`, etc.)
- [ ] `### Status atual` with cross-link to ADR-045 §Status atual
- [ ] Explicit statement: TTS Piper e LLM (Ollama HTTP / Python binding) sidecars PERMANECEM ATIVOS

**Tests**: grep-based (T5)
**Gate**: manual review
**Commit**: `docs(adr): 042 — apply ## Deferral section for ASR sidecar per ADR-046`

---

### T3: Update `docs/adr/README.md` tags (deferred on 035 + 042)

**What**: Add `deferred` tag to ADR-035 and ADR-042 rows in `docs/adr/README.md` (if not already present). Preserve existing tags.
**Where**: `docs/adr/README.md` (modify)
**Depends on**: T1, T2
**Reuses**: Existing index format.
**Requirement**: DEFR-12, DEFR-13, DEFR-14
**Status**: Pending

**Done when**:

- [ ] ADR-035 row in README contains tag `deferred` (appended to existing tags)
- [ ] ADR-042 row in README contains tag `deferred`
- [ ] No existing tags removed

**Tests**: grep-based (T5)
**Gate**: manual review
**Commit**: `docs(readme): add deferred tag to ADR-035 + ADR-042 rows`

---

### T4: Optional — add frontmatter `deferred` tag to ADR-035 + ADR-042

**What**: Add `deferred` to the `Tags:` line in the YAML frontmatter of both ADRs. Mirror what's added to README. (Optional polish — main work is in T1/T2 body content.)
**Where**: `docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md` (modify frontmatter), `docs/adr/042-nucleo-local-mymemoryd-com-workers-sidecar-isolados.md` (modify frontmatter)
**Depends on**: T1, T2
**Reuses**: Existing ADR frontmatter format.
**Requirement**: Tag consistency
**Status**: Pending

**Done when**:

- [ ] ADR-035 `Tags:` line includes `deferred`
- [ ] ADR-042 `Tags:` line includes `deferred`
- [ ] Original tags preserved

**Tests**: grep-based
**Gate**: manual review
**Commit**: `docs(adr): 035 + 042 — add deferred tag to frontmatter`

---

### T5: Grep-based regression test for "optional prose"

**What**: Create `test/check_deferred_prose.sh` (or Go test in `internal/adr/check_test.go`) that fails if word "opcionalmente" or "também disponível" appears in any `## Deferral` section of any ADR. Add to CI nightly run.
**Where**: `test/check_deferred_prose.sh` (new) OR `internal/adr/check_test.go` (new)
**Depends on**: T1, T2, T3, T4
**Reuses**: `grep`, `awk`, `bash` OR Go testing.
**Requirement**: Grep regression for prose-vague regression.
**Status**: Pending

**Done when**:

- [ ] `bash test/check_deferred_prose.sh` exits 0 on current state (post-T1/T2/T3/T4)
- [ ] Script fails (exit 1) when "opcionalmente" injected into `## Deferral` section
- [ ] Test added to CI nightly or quality gate
- [ ] Test `TestCheck_DeferralProse_NoOptionalWords` validates regression

**Tests**: integration
**Gate**: full
**Commit**: `test(adr): grep regression for "optional prose" in ## Deferral sections`
