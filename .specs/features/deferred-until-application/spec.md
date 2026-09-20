# Feature: deferred-until-application

> **Status**: Specify phase — methodology is [ADR-046](../../../docs/adr/046-deferred-until-padrao-para-fallbacks-opcionais-em-adrs.md); this spec captures WHAT (testable requirements) for applying the pattern retroactively to existing ADRs that still use "opcional" prose.

## Problem Statement

O [ADR-046](../../../docs/adr/046-deferred-until-padrao-para-fallbacks-opcionais-em-adrs.md) (Accepted em 2026-09-19) codificou o padrão `## Deferral: <Plano B>` como metodologia transversal. A primeira aplicação foi consolidada retroativamente em [ADR-045](../../../docs/adr/045-asr-streaming-engine-nemotron-35-via-onnx-runtime.md) durante a mesma sessão.

Outros dois ADRs já Accepted ainda usam prosa vaga do tipo "opcionalmente disponível como fallback":

- **ADR-035** (Embedder Embutido com Fallback ONNX MiniLM) — descreve "fallback ONNX MiniLM" sem critérios de promoção, lista do que fica proibido, ou procedimento.
- **ADR-042** (mymemoryd com Workers Sidecar Isolados) — trata Python sidecars como contrato permanente pra ASR/TTS/LLM, sem diferenciar quais estão ativos hoje (TTS, LLM) vs. quais são reservas diferidas (ASR — explicitamente movido pra deferred per ADR-045).

Aplicar o padrão a esses dois ADRs fecha o ciclo: a metodologia vira default transversal e leitores futuros não vão tropeçar em prosa vaga ao procurar fallbacks.

## Goals

- [ ] Adicionar seção `## Deferral: <Plano B>` em ADR-035 com 4 subseções obrigatórias (gatilhos, proibições, promoção, status).
- [ ] Adicionar seção `## Deferral: Python Sidecar ASR` em ADR-042 cross-linkando ADR-045 (que é o plano-de-detalhe) e usando a estrutura do ADR-046.
- [ ] Marcar explicitamente no ADR-042 que TTS e LLM sidecars Python permanecem ativos (não deferred); apenas ASR sidecar Python foi movido pra deferred.
- [ ] Atualizar index `docs/adr/README.md` se necessário (entradas já existem; verificar tags).
- [ ] Validar via `validate_spec.py` + reviewer manual que as 4 subseções estão completas em ambos ADRs.
- [ ] Cobrir invariantes com grep-tests: `TestNoOptionalProse_InDeferredSections` falha se palavra "opcionalmente" aparecer em seção `## Deferral` (deveria ser proibida).

## Out of Scope

| Feature | Reason |
| --- | --- |
| Aplicar deferral a outros ADRs além de 035 + 042 | Esta spec cobre os 2 ADRs explicitamente listados em ADR-046 §Implementation Plan #2; demais retrofit são tarefas separadas |
| Mudança de status de ADR-035 ou ADR-042 | Ambos já são Accepted; esta spec adiciona seção, não reabre decisão |
| Implementar código Go / TypeScript / Python | Spec é sobre documentação; zero código de produção envolvido |
| Atualizar ADR-045 novamente | Já foi cross-linkado pra ADR-046 em commit `b27a54a`; sem mudança adicional |
| Renumeração de ADRs | Mantém-se a numeração atual; refatoração massiva de índices seria escopo separado |
| Refactor do `validate_spec.py` pra checar estrutura de ADRs | Spec é manual via review; automação é enhancement posterior |

---

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --- | --- | --- | --- |
| Identidade do "Plano B" no ADR-035 | "Fallback ONNX Runtime CPU-only" (vs GPU/CUDA que ainda não está implementado) | Embedder builtin (ADR-035) hoje tem 1 provider ativo (ONNX MiniLM via `onnxruntime_go`); fallback mencionado é genérico e precisa de critério | n (verificar com Felipe durante Specify se encaixa) |
| Identidade do "Plano B" no ADR-042 | Python sidecar ASR (cross-link com ADR-045) | Já documentado como deferred em ADR-045 §Deferral; só precisa cross-linkar | y |
| Formato das 4 subseções | Template do ADR-046 §Application Template (copy/paste) | Consistência cross-ADR | y |
| Posição da seção `## Deferral` em ADR-035 | Após `## Decision Outcome`, antes de `## Consequences` | Posição canônica do ADR-045 (que é first application) | y |
| Posição da seção `## Deferral` em ADR-042 | Após `## Decision Outcome`, antes de `## Consequences` | Mesma posição canônica | y |
| Cross-link style | Wikilink relativo Markdown `[ADR-046](046-...md)` | Mesmo estilo do ADR-045 (cross-link retroativo) | y |
| Tag nos ADRs modificados | Adicionar tag `deferred` ao frontmatter se ainda não estiver | ADR-038 já usa essa tag; consistente | y |
| Grep-test target | `docs/adr/035-*.md` e `docs/adr/042-*.md` — palavra "opcionalmente" ou "também disponível" em contexto de fallback | Regressão barata pra evitar volta da prosa vaga | y |
| Granularidade da validação | Manual via reviewer + `validate_spec.py` (não é spec.md, mas confirma estrutura) | `validate_spec.py` é focado em spec.md; ADRs são texto livre | n (escolher entre `validate_spec.py` adapted ou manual) |

**Open questions:** two — identidade exata do Plano B no ADR-035 + qual ferramenta valida estrutura de ADR (não spec.md). Resolver antes da T1.

---

## User Stories

### P1: Aplicar `## Deferral` no ADR-035 ⭐ MVP

**User Story**: As a leitor do ADR-035, I want ver explicitamente quais condições promovem o fallback ONNX CPU-only e o que fica proibido até essa promoção so that a decisão seja rastreável e reversível.

**Why P1**: Sem seção `## Deferral`, o ADR-035 continua usando o anti-pattern "fallback opcional" que ADR-046 documentou como dívida arquitetural.

**Acceptance Criteria**:

1. WHERE the ADR-035's "Fallback ONNX MiniLM" prose currently appears THEN the system SHALL replace it with a `## Deferral: <Plano B>` section following the ADR-046 §Application Template. (optional-feature)
2. The system SHALL enumerate 2-3 gatilhos for promoting the fallback (ex: ONNX binding regressão, modelo MiniLM descontinuado, restrição contratual de cadeia de suprimentos). (ubiquitous)
3. The system SHALL enumerate 3+ proibições until promotion (arquivos, configs, flags). (ubiquitous)
4. The system SHALL enumerate 5 numbered steps in `### Como promover (quando um gatilho dispara)`. (ubiquitous)
5. The system SHALL include a `### Status atual` subseção with `Última revisão`, `Gatilhos disparados: 0`, `Evidence log: nenhum ainda`, and `Próxima revisão`. (ubiquitous)
6. The system SHALL cross-link `[ADR-046](../046-...)` na nova seção. (ubiquitous)

**Independent Test**: `TestAdr035_HasDeferralSection` valida presença da seção via `grep "^## Deferral"`; `TestAdr035_DeferralHasFourSubsections` valida 4 subseções enumeradas; `TestAdr035_NoOptionalProse` grep-falha se palavra "opcionalmente" aparecer adjacente a "fallback".

---

### P2: Aplicar `## Deferral` no ADR-042 (ASR Python Sidecar)

**User Story**: As a leitor do ADR-042, I want ver explicitamente que ASR Python sidecar está deferred (per ADR-045) e que TTS/LLM Python sidecars permanecem ativos so that contratos futuros não instanciem ASR sidecar por engano.

**Why P2**: ADR-042 define contratos genéricos (json-rpc, manifest, supervisor); sem clarificação, implementação futura pode instanciar ASR worker Python indevidamente, violando ADR-045 §Deferral.

**Acceptance Criteria**:

1. WHERE the ADR-042's "workers sidecar Python" prose currently treats ASR equally to TTS/LLM THEN the system SHALL add a `## Deferral: Python Sidecar ASR` section clarifying only TTS/LLM are active today. (optional-feature)
2. The system SHALL cross-link to `[ADR-045](../045-...)` (first application) and `[ADR-046](../046-...)` (metodologia). (ubiquitous)
3. The system SHALL NOT add `## Deferral` for TTS or LLM Python sidecars (per ADR-042 §Decision Outcome, ambos permanecem ativos). (ubiquitous)
4. The system SHALL enumerate 3+ proibições specifically for ASR sidecar instantiation (e.g., `internal/voice/asr_worker.py`, `--asr-engine=python`, dependência `pywhispercpp` no `pyproject.toml`). (ubiquitous)
5. The system SHALL include `### Status atual` com cross-link ao ADR-045 §Status atual. (ubiquitous)

**Independent Test**: `TestAdr042_AsrIsDeferred_TtsLlmActive` valida via grep que ASR aparece em seção `## Deferral` mas TTS/LLM NÃO; `TestAdr042_HasAsrProhibitions` valida lista de proibições ASR-specific.

---

### P3: Atualizar README índice + tags

**User Story**: As a leitor do `docs/adr/README.md`, I want ver as tags atualizadas dos ADRs 035 + 042 (com `deferred` adicionado) so que o índice reflita o estado pós-application.

**Why P3**: Sem tag update, filtros por `deferred` no índice não capturam esses dois ADRs.

**Acceptance Criteria**:

1. WHERE the `docs/adr/README.md` row for ADR-035 currently lacks `deferred` tag THEN the system SHALL append it. (optional-feature)
2. WHERE the `docs/adr/README.md` row for ADR-042 currently lacks `deferred` tag THEN the system SHALL append it. (optional-feature)
3. The system SHALL NOT remove existing tags from either row. (ubiquitous)

**Independent Test**: `TestReadme_HasDeferredTag_For035And042` valida via grep que tag `deferred` aparece em ambas as linhas.

---

## Edge Cases

- IF o autor do ADR-035 já tiver reaberto a decisão em outro ADR mais recente THEN this spec SHALL pause and request user clarification before modifying ADR-035. (unwanted-behavior)
- IF o autor do ADR-042 tiver criado ADR-046+ que conflita com o cross-link esperado THEN this spec SHALL cross-link ao ADR mais recente, não ao ADR-046. (unwanted-behavior)
- WHEN applying `## Deferral` causes the ADR file to exceed 500 lines THEN consider splitting into ADR-035-core + ADR-035-deferral. (state-driven)
- IF `validate_spec.py` for adapted-ADR check não existe THEN this spec SHALL use manual review com checklist explícito nas tasks. (unwanted-behavior)

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| --- | --- | --- | --- |
| DEFR-01 | P1 | Design | Pending |
| DEFR-02 | P1 | Design | Pending |
| DEFR-03 | P1 | Design | Pending |
| DEFR-04 | P1 | Design | Pending |
| DEFR-05 | P1 | Design | Pending |
| DEFR-06 | P1 | Design | Pending |
| DEFR-07 | P2 | Design | Pending |
| DEFR-08 | P2 | Design | Pending |
| DEFR-09 | P2 | Design | Pending |
| DEFR-10 | P2 | Design | Pending |
| DEFR-11 | P2 | Design | Pending |
| DEFR-12 | P3 | Design | Pending |
| DEFR-13 | P3 | Design | Pending |
| DEFR-14 | P3 | Design | Pending |

**Coverage:** 14 total, 0 mapped to tasks, 14 unmapped ⚠️ (tasks.md pending).

---

## Success Criteria

- [ ] ADR-035 contém `## Deferral: <Plano B>` com 4 subseções obrigatórias + cross-link pra ADR-046.
- [ ] ADR-042 contém `## Deferral: Python Sidecar ASR` com 4 subseções + cross-link pra ADR-045 + ADR-046.
- [ ] ADR-042 explicitamente marca TTS e LLM Python sidecars como ativos (não deferred).
- [ ] `docs/adr/README.md` tem tag `deferred` em ambos os ADRs.
- [ ] Grep-test `TestNoOptionalProse_InDeferredSections` passa em ambos.
- [ ] Nenhum commit quebra o quality gate (`gofmt`, `go build`, `go test`) — esperado já que mudanças são só em `.md`.
- [ ] Cross-link renderiza corretamente quando indexado pelo Obsidian / Markdown viewer.
