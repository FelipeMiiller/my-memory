# Tasks: fix-validate-state-verdict

**Spec**: [spec.md](./spec.md)
**Date**: 2026-10-29
**Issue**: ISSUE-033 (`.specs/ISSUES.md:1336`)

## Test Coverage Matrix

| Code Layer | Required Test Type | Coverage Expectation | Location Pattern | Run Command |
| ---------- | ------------------ | -------------------- | ---------------- | ----------- |
| Gate — leitura do veredito (`_segment`, `_classify`, `_verdict`) | unit | PASS · FAIL · PARTIAL · placeholder `PASS | FAIL` · sem candidata → None · prosa `Resultado:` sem palavra de veredito → None · `PASS-WITH-GAPS` não é placeholder | `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py` | `python .agents/skills/tlc-spec-driven/scripts/test_validate_state.py` |
| Gate — relatório completo (`_check_feature`) | unit | PASS com evidência → 0 erros · PASS sem evidência → 1 erro · FAIL → 1 erro · PARTIAL → 1 erro | `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py` | `python .agents/skills/tlc-spec-driven/scripts/test_validate_state.py` |
| Relatórios normalizados (6 features) | integração | Cada um passa o gate da própria feature com o bloco canônico + evidência `arquivo:linha` | `.specs/features/*/validation.md` | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py <feature> --root D:\repository\my-memory` |
| Cross-check do vault inteiro | regressão | 41 features: exatamente 6 erros (5 "sem veredito" + 1 "PARTIAL") e zero feature que passava antes reprovando | `.specs/features/` | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py --root D:\repository\my-memory` |

## Gate Check Commands

| Gate Level | When to Use | Command |
| ---------- | ----------- | ------- |
| Quick | Depois de mexer na regra de veredito | `python .agents/skills/tlc-spec-driven/scripts/test_validate_state.py` |
| Quick | Depois de normalizar cada relatório | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py <feature> --root D:\repository\my-memory` |
| Full | No fim, contra o vault inteiro | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py --root D:\repository\my-memory` |
| Spec | Antes de entregar | `python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/fix-validate-state-verdict/spec.md` e `python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py .specs/features/fix-validate-state-verdict/tasks.md` |
| State | Antes de entregar | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-validate-state-verdict --root D:\repository\my-memory` |
| Encoding | Antes de entregar | `git ls-files --eol <arquivo>` — a coluna `i/` tem de ser `lf` |

## Execution Plan

Phases are ordered and run sequentially.

```
T1 -> T2
T1 -> T3
T1 -> T4
T1 -> T5
T1 -> T6
T1 -> T7
T1 -> T8
T2 -> T10
T9 -> T10
```

| Fase | Tarefas | Objetivo |
| ---- | ------- | -------- |
| 1 | T1, T2 | Corrigir a regra de veredito do gate e travar o comportamento com a suíte de testes |
| 2 | T3, T4, T5, T6, T7, T8 | Normalizar os 6 relatórios que têm veredito real (5 PASS + 1 PARTIAL) |
| 3 | T9, T10 | Documentar o novo comportamento do gate e fechar o cross-check com o relatório de evidências |

## Task Breakdown

### Phase 1: O gate

### T1: Corrigir a regra de veredito do gate
**What**: Reescrever a leitura do veredito: (a) reconhecer `Result:`/`Resultado:`/`Verdict:`/`Veredito:` e a linha que abre com o token de veredito como candidatas (um token que nomeia um campo, como `- **PARTIAL / SPEC-GAP**: 0`, é contagem e não veredito); (b) remover o fallback que varre o documento inteiro e devolver "sem veredito" quando não há candidata; (c) detectar o placeholder literal `PASS | FAIL` em vez do proxy "a linha tem PASS e FAIL"; (d) tratar `PARTIAL` como veredito próprio, com precedência sobre PASS e FAIL; (e) reportar PARTIAL com mensagem própria.
**Where**: .agents/skills/tlc-spec-driven/scripts/validate_state.py
**Depends on**: none
**Requirement**: VSV-01, VSV-02, VSV-03, VSV-04, VSV-05, VSV-06, VSV-07, VSV-08, VSV-09, VSV-10, VSV-24
**Tests**: .agents/skills/tlc-spec-driven/scripts/test_validate_state.py
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_state.py --root D:\repository\my-memory
**Done when**:
- [x] `_segment` devolve o segmento de veredito das 3 famílias de linha candidata (heading `## Validation`, campo rotulado, linha que abre com o token)
- [x] Um token que **nomeia um campo** (`- **PARTIAL / SPEC-GAP**: 0`) não é candidato — é a contagem da seção de contagens do relatório, não um veredito
- [x] `_verdict` devolve `None` quando não há nenhuma linha candidata — o corpo do documento não é mais palha
- [x] `_classify` lê o **primeiro** palavra de veredito do segmento, com `PARTIAL` antes de `PASS`
- [x] O placeholder `PASS | FAIL` é detectado pelo literal, não pelo proxy "PASS e FAIL na mesma linha"
- [x] `PARTIAL` tem precedência no arquivo inteiro e nunca é reportado como PASS
- [x] `_check_feature` tem um ramo `partial` com mensagem própria
- [x] A regra de evidência (`file:line` obrigatório para PASS) está preservada

### T2: Travar o comportamento com a suíte de testes do gate
**What**: Criar a suíte de testes do `validate_state.py` no padrão do repositório (asserts + `main()` próprio, sem pytest), cobrindo PASS com evidência, PASS sem evidência, FAIL, placeholder não preenchido, ausência de candidata, PARTIAL numa linha que também contém PASS, e as linhas `Veredito:`/`Resultado:`/`Verdict:`.
**Where**: .agents/skills/tlc-spec-driven/scripts/test_validate_state.py
**Depends on**: T1
**Requirement**: VSV-19, VSV-20, VSV-21, VSV-22
**Tests**: .agents/skills/tlc-spec-driven/scripts/test_validate_state.py
**Gate**: python .agents/skills/tlc-spec-driven/scripts/test_validate_state.py
**Done when**:
- [x] PASS com evidência `arquivo:linha` → `_check_feature` sem erros
- [x] PASS sem evidência → erro de evidence-or-zero
- [x] FAIL → erro de FAIL
- [x] `**Result**: [PASS | FAIL]` → erro de template não preenchido
- [x] Relatório sem linha candidata → "sem veredito" (e **não** "unfilled")
- [x] `**Verdict**: **PARTIAL** — PASS for 14 ACs …` → `partial`, nunca `pass`
- [x] `**Veredito**:`, `**Resultado**:`, `**Verdict**:` e `**Result**:` reconhecidos
- [x] Uma linha de contagem (`- **PARTIAL / SPEC-GAP**: 0`) não decide o veredito
- [x] Um rótulo citado em code span não declara veredito
- [x] O token que abre a linha precisa ser enfase (`**PASS**`) para ser uma declaração
- [x] `python .agents/skills/tlc-spec-driven/scripts/test_validate_state.py` sai com exit 0

### Phase 2: Os 6 relatórios com veredito real

### T3: Normalizar o relatório de `event-runtime-event-log`
**What**: Acrescentar o bloco canônico `## Validation: PASS` + `**Result**: PASS`. O relatório já tem `**Verdict**: PASS` e 68 evidências `arquivo:linha` — nada mais a fazer.
**Where**: .specs/features/event-runtime-event-log/validation.md
**Depends on**: T1
**Requirement**: VSV-11
**Tests**: .specs/features/event-runtime-event-log/validation.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_state.py event-runtime-event-log --root D:\repository\my-memory
**Done when**:
- [x] O bloco canônico foi acrescentado
- [x] As 68 evidências e todo o conteúdo original estão intactos (diff aditivo)
- [x] A feature passa o gate

### T4: Normalizar o relatório de `post-release-v1.3.0-bugfixes`
**What**: Acrescentar o bloco canônico e duas citações `arquivo:linha` reais — o relatório hoje só casa com a regex por causa de uma URL (`http://127.0.0.1:8765`), que não é evidência.
**Where**: .specs/features/post-release-v1.3.0-bugfixes/validation.md
**Depends on**: T1
**Requirement**: VSV-12
**Tests**: .specs/features/post-release-v1.3.0-bugfixes/validation.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_state.py post-release-v1.3.0-bugfixes --root D:\repository\my-memory
**Done when**:
- [x] O bloco canônico foi acrescentado
- [x] `internal/graphview/template_test.go:120` e `cmd/mem/graph_test.go:131` foram conferidos abrindo os arquivos
- [x] Nenhum conteúdo original foi apagado
- [x] A feature passa o gate

### T5: Normalizar o relatório de `triptych-node-inspector`
**What**: Acrescentar o bloco canônico e duas citações `arquivo:linha` reais — hoje o relatório só casa com a regex por causa de uma URL (`http://127.0.0.1:38400`).
**Where**: .specs/features/triptych-node-inspector/validation.md
**Depends on**: T1
**Requirement**: VSV-13
**Tests**: .specs/features/triptych-node-inspector/validation.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_state.py triptych-node-inspector --root D:\repository\my-memory
**Done when**:
- [x] O bloco canônico foi acrescentado
- [x] `internal/graph/inspector_test.go:35` e `internal/graph/inspector_test.go:7` foram conferidos abrindo o arquivo
- [x] Nenhum conteúdo original foi apagado
- [x] A feature passa o gate

### T6: Normalizar o relatório de `embedder-builtin-onnx-minilm`
**What**: Acrescentar o bloco canônico e a evidência `arquivo:linha` que faltava — o relatório cita **zero** `arquivo:linha`, o que reprova a regra de evidência. As citações apontam para o código do repositório que materializa os achados do spike (`go.mod`, `internal/embedder/builtin.go`), com o escopo (T1 + T2) explicitado.
**Where**: .specs/features/embedder-builtin-onnx-minilm/validation.md
**Depends on**: T1
**Requirement**: VSV-14
**Tests**: .specs/features/embedder-builtin-onnx-minilm/validation.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_state.py embedder-builtin-onnx-minilm --root D:\repository\my-memory
**Done when**:
- [x] O bloco canônico foi acrescentado
- [x] `go.mod:11`, `internal/embedder/builtin.go:12`, `:51`, `:62`, `:63` e `:143` foram conferidos abrindo os arquivos
- [x] O escopo (T1 + T2) está declarado — o spike rodou em módulo descartável, sem artefato no repositório
- [x] Nenhum conteúdo original foi apagado
- [x] A feature passa o gate

### T7: Normalizar o relatório de `viewer-dashboard`
**What**: Acrescentar o bloco canônico. O relatório já é PASS (47/47 ACs), com 26 evidências; o "template não preenchido" era falso positivo do validador.
**Where**: .specs/features/viewer-dashboard/validation.md
**Depends on**: T1
**Requirement**: VSV-15
**Tests**: .specs/features/viewer-dashboard/validation.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_state.py viewer-dashboard --root D:\repository\my-memory
**Done when**:
- [x] O bloco canônico foi acrescentado
- [x] As 26 evidências e todo o conteúdo original estão intactos (diff aditivo)
- [x] O erro "template não preenchido" desapareceu
- [x] A feature passa o gate

### T8: Normalizar o relatório de `feat-code-ast` para PARTIAL
**What**: Acrescentar o bloco canônico `## Validation: PARTIAL` + `**Result**: PARTIAL`, explicitando que a feature não está pronta (3 ACs em SPEC-GAP + 1 sub-item + 1 mutante sobrevivente) e que `PARTIAL` não é PASS.
**Where**: .specs/features/feat-code-ast/validation.md
**Depends on**: T1
**Requirement**: VSV-16, VSV-18
**Tests**: .specs/features/feat-code-ast/validation.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_state.py feat-code-ast --root D:\repository\my-memory
**Done when**:
- [x] O bloco canônico declara PARTIAL
- [x] As 42 evidências e todo o conteúdo original estão intactos (diff aditivo)
- [x] A feature **continua vermelha**, agora com o motivo "PARTIAL" e não "sem veredito"
- [x] Nenhum PASS foi carimbado

### Phase 3: Documentação e gate final

### T9: Documentar o novo comportamento do gate no SKILL.md
**What**: Atualizar a linha que descreve o gate de conclusão para dizer que PASS, FAIL e PARTIAL são três vereditos distintos e que PARTIAL não é PASS.
**Where**: .agents/skills/tlc-spec-driven/SKILL.md
**Depends on**: T1
**Requirement**: VSV-23
**Tests**: .agents/skills/tlc-spec-driven/SKILL.md
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_state.py --root D:\repository\my-memory
**Done when**:
- [x] A descrição do gate menciona os três vereditos
- [x] A linha diz explicitamente que PARTIAL reprova a feature
- [x] O resto do SKILL.md está intacto (diff aditivo)

### T10: Cross-check final e relatório de evidências
**What**: Rodar o cross-check do vault inteiro, comparar com o baseline de 11 erros, provar que as features que passavam continuam passando e registrar as evidências por critério.
**Where**: .specs/features/fix-validate-state-verdict/validation.md
**Depends on**: T2, T3, T4, T5, T6, T7, T8, T9
**Requirement**: VSV-17
**Tests**: .agents/skills/tlc-spec-driven/scripts/test_validate_state.py
**Gate**: python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-validate-state-verdict --root D:\repository\my-memory
**Done when**:
- [x] A suíte de testes do gate sai com exit 0
- [x] O cross-check sai com **6** erros: 5 "sem veredito" + 1 "PARTIAL"
- [x] A lista de features que passa é a mesma de antes (guard de regressão)
- [x] `feat-code-ast` aparece como PARTIAL
- [x] `validation.md` com evidência `arquivo:linha` por critério
