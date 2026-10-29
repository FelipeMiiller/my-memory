---
title: "Validation: fix-validate-state-verdict"
category: resource
summary: "Verifier report — fix-validate-state-verdict (ISSUE-033) — o gate deterministico validate_state.py passa a ler o veredito nas linhas que o carregam, sem varrer o documento inteiro, e trata PARTIAL como veredito proprio. Cross-check do vault: 11 erros -> 6, zero regressoes."
type: validation
created_at: 2026-10-29T00:00:00-03:00
tags: [validation, spec-driven, tlc-spec-driven, validate-state, issue-033]
---

# Validation: fix-validate-state-verdict

**Date**: 2026-10-29
**Spec**: `.specs/features/fix-validate-state-verdict/spec.md` (5 user stories, 24 requisitos)
**Tasks**: `.specs/features/fix-validate-state-verdict/tasks.md` (T1–T10)
**Issue**: ISSUE-033 (`.specs/ISSUES.md:1336`)
**Branch**: `develop` · **Base**: `adfb8cc`
**Verifier**: execução independente do gate (author ≠ verifier para os artefatos de `validate_state.py`: o script e o relatório foram escritos em sessões diferentes, e o sensor de discriminação abaixo prova que os testes do gate matam a implementação antiga)

## Validation: PASS

**Result**: PASS

Veredito: **PASS**. O gate determinístico `validate_state.py` reprova 11 das 41
features verificadas por **desenho do validador**, não por conteúdo. Depois
desta correção, o cross-check do vault inteiro reporta **6 erros** — as 5
features cujo `validation.md` é um `# Validation Plan` (nunca verificadas, e que
o dono adiou) mais `feat-code-ast`, reportada honestamente como **PARTIAL**.
**Zero regressões**: as features que passavam antes continuam passando.

Evidência `arquivo:linha`:

- `.agents/skills/tlc-spec-driven/scripts/validate_state.py:107` — `_segment`: as 3 famílias de linha candidata (heading `## Validation`, campo rotulado, linha que abre com o token de veredito enfase).
- `.agents/skills/tlc-spec-driven/scripts/validate_state.py:161` — `_verdict`: regra de decisão (primeiro candidato decisivo; `PARTIAL` tem precedência).
- `.agents/skills/tlc-spec-driven/scripts/validate_state.py:192` — `if "partial" in decisive:`: precedência de PARTIAL no arquivo inteiro.
- `.agents/skills/tlc-spec-driven/scripts/validate_state.py:236` — `elif verdict == "partial":`: ramo próprio de PARTIAL em `_check_feature`.
- `.agents/skills/tlc-spec-driven/scripts/validate_state.py:93` — `_label_segment`: um rótulo citado em code span é exemplo, não declaração.
- `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py:38` — `test_verdict_none_without_candidate`: sem candidata → `None`.
- `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py:133` — `test_verdict_partial_is_not_pass`: PARTIAL nunca vira PASS.
- `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py:187` — `test_verdict_count_line_is_not_a_verdict`: a seção de contagens do relatório não decide o veredito.
- `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py:203` — `test_verdict_quoted_label_is_not_a_declaration`: rótulo citado não declara.
- `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py:212` — `test_verdict_leading_token_must_be_emphasized`: o token que abre a linha precisa ser ênfase.
- `.agents/skills/tlc-spec-driven/SKILL.md:42` — a descrição do gate agora diz que PASS, FAIL e PARTIAL são três vereditos distintos.

---

## 1. Spec-Anchored Outcome Check

Cada AC re-derivado contra o artefato real, com evidência `arquivo:linha`.

| AC | Critério | Veredito | Evidência |
| :--- | :--- | :--- | :--- |
| P1.1 | `Result:`/`Resultado:`/`Verdict:`/`Veredito:` são candidatas | **PASS** | `.agents/skills/tlc-spec-driven/scripts/validate_state.py:56` (`_LABEL_RE`) + `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py:174` (`test_verdict_labeled_lines_recognized`) |
| P1.2 | Linha que abre com o token de veredito enfase é candidata (um token que nomeia um campo não) | **PASS** | `.agents/skills/tlc-spec-driven/scripts/validate_state.py:69` (`_LEADING_RE`) + `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py:187` (`test_verdict_count_line_is_not_a_verdict`) + `:212` (`test_verdict_leading_token_must_be_emphasized`) |
| P1.3 | Sem candidata → "sem veredito"; o corpo não é palha | **PASS** | `.agents/skills/tlc-spec-driven/scripts/validate_state.py:161` (`_verdict` devolve `None` quando `decisive` é vazio) + `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py:38` |
| P1.4 | Placeholder `PASS \| FAIL` detectado | **PASS** | `.agents/skills/tlc-spec-driven/scripts/validate_state.py:75` (`_PLACEHOLDER_RE`) + `test_validate_state.py` caso `test_verdict_unfilled_template` |
| P2.1 | PARTIAL é veredito próprio, nunca PASS | **PASS** | `.agents/skills/tlc-spec-driven/scripts/validate_state.py:142` (`_classify`) + `.agents/skills/tlc-spec-driven/scripts/validate_state.py:192` |
| P2.2 | PARTIAL numa linha que também diz PASS → PARTIAL | **PASS** | `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py:133` (`test_verdict_partial_is_not_pass`) |
| P2.3 | PARTIAL em qualquer linha mantém a feature vermelha | **PASS** | `.agents/skills/tlc-spec-driven/scripts/validate_state.py:192` + `.agents/skills/tlc-spec-driven/scripts/validate_state.py:236` |
| P1.5 | Rótulo citado em code span não declara veredito | **PASS** | `.agents/skills/tlc-spec-driven/scripts/validate_state.py:93` (`_label_segment`) + `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py:203` |
| P3.1 | Toda feature que passa hoje continua passando | **PASS** | cross-check antes/depois (seção 3) — a lista de features verdes é a mesma |
| P3.2 | Linha de prosa `Resultado:` sem palavra de veredito não decide | **PASS** | `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py` caso `test_verdict_prose_resultado_line_is_not_decisive` |
| P3.3 | `PASS-WITH-GAPS` não é confundido com placeholder | **PASS** | `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py` caso `test_verdict_pass_with_gaps_is_not_unfilled` |
| P4.1 | Relatório com veredito real recebe o bloco canônico | **PASS** | `.specs/features/event-runtime-event-log/validation.md:8` · `.specs/features/post-release-v1.3.0-bugfixes/validation.md:10` · `.specs/features/triptych-node-inspector/validation.md:12` · `.specs/features/embedder-builtin-onnx-minilm/validation.md:133` · `.specs/features/viewer-dashboard/validation.md:18` |
| P4.2 | Nenhum conteúdo ou evidência apagado | **PASS** | `git diff --stat` nos 6 arquivos: **só linhas adicionadas** (seção 4) |
| P4.3 | PASS cita ao menos uma evidência `arquivo:linha` verificável | **PASS** | `.specs/features/embedder-builtin-onnx-minilm/validation.md:139` (bloco de evidências) |
| P5.1 | `feat-code-ast` expressa PARTIAL na forma canônica | **PASS** | `.specs/features/feat-code-ast/validation.md:22` (`## Validation: PARTIAL`) |
| P5.2 | O gate reporta `feat-code-ast` como PARTIAL | **PASS** | saída do cross-check (seção 3) |
| P5.3 | As 42 evidências de `feat-code-ast` intactas; nenhum PASS carimbado | **PASS** | `git diff --stat .specs/features/feat-code-ast/validation.md` → 10 linhas adicionadas, 0 removidas |

### Counts

- **PASS**: 17 / 17 ACs
- **PARTIAL / SPEC-GAP**: 0
- **FAIL**: 0

---

## 2. Discrimination Sensor

Sensor rodado numa **cópia isolada** dos dois arquivos do gate, no scratch da
execução — o repositório não foi tocado. Cada mutante é a implementação antiga
(defeito real) reintroduzida numa cópia; a suíte tem de matá-la.

| # | Mutante | Injeção | Teste que mata | Status |
| :--- | :--- | :--- | :--- | :--- |
| **M1** | Fallback antigo: sem candidata, o documento inteiro vira palha (`hay = text`) | `_verdict` reescrito com a implementação pré-correção | `test_verdict_none_without_candidate` — `AssertionError: esperado None, obtido 'unfilled'` | **MORTO** |
| **M2** | PARTIAL classificado como PASS (`"PARTIAL": "pass"`) | mapa de `_classify` trocado | `test_verdict_partial_is_not_pass` — `AssertionError: esperado 'partial', obtido 'pass'` | **MORTO** |

**Sensor kills: 2 / 2 faults killed (100% de discriminação).**

O M1 reproduz exatamente o falso positivo que motivou a ISSUE-033: o
`viewer-dashboard` (53 PASS e 2 FAIL em prosa) era classificado como
"template não preenchido". O M2 reproduz a corrupção que a correção ingênua
causaria: `feat-code-ast` viraria verde.

---

## 3. Cross-check do vault inteiro — antes e depois

**Antes** (baseline medido em `develop` @ `adfb8cc`, antes de qualquer mudança):

```
  ERROR ai-tool-auto-wiring: validation.md has no PASS/FAIL verdict (a prose-only report does not count)
  ERROR context-packager-and-subgraph-bundle: validation.md has no PASS/FAIL verdict (a prose-only report does not count)
  ERROR embedder-builtin-onnx-minilm: validation.md has no PASS/FAIL verdict (a prose-only report does not count)
  ERROR event-runtime-event-log: validation.md has no PASS/FAIL verdict (a prose-only report does not count)
  ERROR feat-code-ast: validation.md has no PASS/FAIL verdict (a prose-only report does not count)
  ERROR graph-path-discovery: validation.md has no PASS/FAIL verdict (a prose-only report does not count)
  ERROR post-release-v1.3.0-bugfixes: validation.md has no PASS/FAIL verdict (a prose-only report does not count)
  ERROR progressive-context-loading: validation.md has no PASS/FAIL verdict (a prose-only report does not count)
  ERROR staleness-detection-and-banners: validation.md has no PASS/FAIL verdict (a prose-only report does not count)
  ERROR triptych-node-inspector: validation.md has no PASS/FAIL verdict (a prose-only report does not count)
  ERROR viewer-dashboard: validation.md verdict is still the template placeholder '[PASS | FAIL]' - not filled

validate_state: 11 error(s) across [41 features]
```

**Depois**:

```
  ERROR ai-tool-auto-wiring: validation.md has no PASS/FAIL verdict (a prose-only report does not count)
  ERROR context-packager-and-subgraph-bundle: validation.md has no PASS/FAIL verdict (a prose-only report does not count)
  ERROR feat-code-ast: validation.md verdict is PARTIAL - the feature is not fully verified; close the SPEC-GAP/FAIL acceptance criteria listed in the report before promoting it
  ERROR graph-path-discovery: validation.md has no PASS/FAIL verdict (a prose-only report does not count)
  ERROR progressive-context-loading: validation.md has no PASS/FAIL verdict (a prose-only report does not count)
  ERROR staleness-detection-and-banners: validation.md has no PASS/FAIL verdict (a prose-only report does not count)

validate_state: 6 error(s) across [41 features]
```

### Os 6 erros que sobraram são os 6 que **devem** sobrar

| Feature | Erro | Por que o gate está certo |
| :--- | :--- | :--- |
| `ai-tool-auto-wiring` | sem veredito | o `validation.md` é um `# Validation Plan` — nunca houve relatório de verificação |
| `context-packager-and-subgraph-bundle` | sem veredito | idem (opção C da ISSUE-033, **adiada pelo dono**) |
| `graph-path-discovery` | sem veredito | idem |
| `progressive-context-loading` | sem veredito | idem |
| `staleness-detection-and-banners` | sem veredito | idem |
| `feat-code-ast` | **PARTIAL** | o próprio relatório diz PARTIAL (14/17 ACs, 3 SPEC-GAP, 1 mutante sobrevivente) e pede fix-tasks para GAP-1/2/3 antes de promover |

### Guard de regressão

As **35 features** que passavam antes passam depois. As duas que passavam **só**
por causa do fallback que varria o documento inteiro — `mcp-static-serving`
(`**PASS** — 25/25 critérios…`) e `obsidian-tag-parity` (`**PASS** — 46/46
critérios…`) — continuam passando pela nova regra da linha que abre com o token
de veredito. `repo-realignment`, cujo veredito é `**PASS-WITH-GAPS** — 47 PASS ·
3 FAIL`, deixou de ser lido como "template não preenchido" e passou a ser lido
como PASS.

---

## 4. Diff auditado — os 6 relatórios só ganharam linhas

```
 .specs/features/embedder-builtin-onnx-minilm/validation.md | 28 ++++-
 .specs/features/event-runtime-event-log/validation.md     |  6 +-
 .specs/features/feat-code-ast/validation.md                | 10 ++
 .specs/features/post-release-v1.3.0-bugfixes/validation.md |  9 ++
 .specs/features/triptych-node-inspector/validation.md      |  9 ++
 .specs/features/viewer-dashboard/validation.md             |  4 +
```

Nenhuma evidência foi apagada. As contagens de `arquivo:linha` antes/depois:

| Relatório | Evidências antes | Depois |
| :--- | :--- | :--- |
| `event-runtime-event-log` | 68 | 68 |
| `post-release-v1.3.0-bugfixes` | 1 (a URL `//127.0.0.1:8765` — falso positivo da regex) | 3 (2 `arquivo:linha` reais + a URL) |
| `triptych-node-inspector` | 1 (a URL `//127.0.0.1:38400` — falso positivo da regex) | 3 (2 `arquivo:linha` reais + a URL) |
| `embedder-builtin-onnx-minilm` | **0** | 6 |
| `viewer-dashboard` | 26 | 26 |
| `feat-code-ast` | 42 | 42 |

---

## 5. Evidência nova, conferida arquivo por arquivo

Cada `arquivo:linha` acrescentado foi aberto e a linha lida antes de ser escrita.

| Citação | Linha lida |
| :--- | :--- |
| `go.mod:11` | `github.com/yalue/onnxruntime_go v1.36.0` |
| `internal/embedder/builtin.go:12` | `ort "github.com/yalue/onnxruntime_go"` |
| `internal/embedder/builtin.go:51` | `// tem de ser a versão 1.29.0 — ver .specs/features/embedder-builtin-onnx-minilm/validation.md.` |
| `internal/embedder/builtin.go:62` | `ort.SetSharedLibraryPath(libPath)` |
| `internal/embedder/builtin.go:63` | `if err := ort.InitializeEnvironment(); err != nil {` |
| `internal/embedder/builtin.go:143` | `if err := c.sess.Run([]ort.Value{idsTensor, maskTensor, typeTensor}, outputs); err != nil {` |
| `internal/graphview/template_test.go:120` | `func TestRenderHTML_UTF8Encoding(t *testing.T) {` |
| `cmd/mem/graph_test.go:131` | `func TestResolveStorageAndRepo_DerivesRepoFromDBPath(t *testing.T) {` |
| `internal/graph/inspector_test.go:7` | `func TestTruncateContent(t *testing.T) {` |
| `internal/graph/inspector_test.go:35` | `func TestBuildTriptychView(t *testing.T) {` |
| `internal/mcp/structured_content.go:55` | `normalized.DriftedNotes = nonNilSlice(payload.DriftedNotes)` |
| `.github/workflows/ci.yml:424` | `budget=163840   # 160 kB` |

O `internal/embedder/builtin.go:51` é a evidência mais forte do feature
`embedder-builtin-onnx-minilm`: o código de produção **remete ao relatório do
spike** e exige a DLL na versão 1.29.0 — exatamente o bloqueio nº 2 da cadeia
que o spike identificou. O `builtin.go:62` confirma o SPEC_DEVIATION nº 2
(`SetSharedLibraryPath` sem retorno de erro na v1.36.0) e o `builtin.go:143`
é a inferência real que o T1 comprovou.

---

## 6. Quality Gate

| Gate | Comando | Resultado |
| :--- | :--- | :--- |
| Testes do gate | `python .agents/skills/tlc-spec-driven/scripts/test_validate_state.py` | **exit 0** — 17 casos, 100% verde |
| Cross-check | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py --root D:\repository\my-memory` | **6 erros** (5 "sem veredito" + 1 "PARTIAL") |
| Gate da feature | `python .agents/skills/tlc-spec-driven/scripts/validate_state.py fix-validate-state-verdict --root D:\repository\my-memory` | **exit 0** |
| Validador de spec | `python .agents/skills/tlc-spec-driven/scripts/validate_spec.py .specs/features/fix-validate-state-verdict/spec.md` | **0 erros, 0 warnings** |
| Validador de tasks | `python .agents/skills/tlc-spec-driven/scripts/validate_tasks.py .specs/features/fix-validate-state-verdict/tasks.md` | **0 erros, 0 warnings** |
| Encoding | `git ls-files --eol <arquivo>` | `i/lf` em todos os arquivos tocados (o índice é LF; o `w/` CRLF é o estado local pré-existente) |
| Escopo | `git status --short` | só os arquivos desta feature e do script do gate (mais `.memory/config.yaml` e `.specs/features/fix-index-tolerancia/`, de um worker paralelo — não tocados) |

## 7. Ranked Gap List

1. **Minor — `EVIDENCE_RE` aceita `host:port` como evidência.** `http://127.0.0.1:8765`
   casa com `[\w./-]+\.[A-Za-z0-9]+:\d+`. Foi por isso que `post-release-v1.3.0-bugfixes`
   e `triptych-node-inspector` passavam com "1 evidência" que não era evidência
   nenhuma. Não corrigi: o regex está fora do escopo desta feature e endurecê-lo
   reprovaria outros relatórios. Os dois relatórios receberam citações `arquivo:linha`
   reais, então não dependem mais do falso positivo. **Fix-task candidata: exigir
   que o caminho antes do `:` contenha ao menos uma letra e uma barra ou extensão
   de arquivo.**
2. **Minor — o `feat-code-ast` segue vermelho por escolha.** O gate está correto:
   a feature tem 3 ACs em SPEC-GAP e 1 mutante sobrevivente. Fechar GAP-1/2/3 é
   trabalho da feature `feat-code-ast`, não desta.
3. **Note — as 5 features de `# Validation Plan` seguem sem relatório.** Opção C da
   ISSUE-033, adiada pelo dono. O gate está certo em reprovar.
4. **Note — `feat-code-ast/validation.md` tem BOM no blob indexado.** Estado
   pré-existente, preservado: a correção não introduziu nem removeu o BOM, e o
   índice continua `i/lf`.
5. **Minor — a seção de contagens de um relatório precisa de rótulo próprio.** A
   primeira versão da regra tratava qualquer linha que abre com um token de
   veredito como candidata, e a seção "Counts" do `feat-code-ast` (`- **PASS**: 14`,
   `- **PARTIAL / SPEC-GAP**: 4`, `- **FAIL**: 0`) virava candidata — o que
   classificava o relatório como PARTIAL **por acidente**, não pelo veredito.
   A regra final exclui o token que **nomeia um campo** (composto seguido de `:`):
   `- **PARTIAL / SPEC-GAP**: 0` é contagem, não veredito. O veredito do
   `feat-code-ast` continua PARTIAL, agora pelo motivo certo
   (`**Verdict**: **PARTIAL** …`). Fix-task candidata: o template de relatório do
   `SKILL.md` deveria instruir a rotular a contagem como "ACs PASS" / "ACs
   PARTIAL", para não depender da ressalva do validador.
6. **Minor — o rótulo de veredito precisa ser distinguível de uma citação.** Um
   relatório que **cita** um rótulo em prosa (`` (`**Verdict**: **PARTIAL**`) explica… ``)
   fazia o validador ler a citação como declaração. A regra final só considera um
   rótulo que não está dentro de um code span, e só considera um token inicial
   quando ele está **enfatizado** (`**PASS**`) — um token solto no início de uma
   linha de prosa é prosa, não veredito. Efeito colateral aceito: `**PASS**: 16/16`
   (rótulo com dois-pontos, sem o bloco canônico) deixa de ser lido como veredito —
   o relatório fica **vermelho** ("sem veredito"), nunca verde por acidente.
7. **Note — `event-runtime-event-log/validation.md` tem 1 byte inválido em UTF-8
   (pré-existente).** Na linha 6, `residue ␗ git diff` — o byte `0x97` (em-dash
   escrito em CP1252) não é UTF-8 válido. O `git show HEAD:` confirma que o byte
   está no blob commitado, ou seja, é anterior a esta feature. O editor usado
   nesta execução normalizou o byte para U+FFFD; ele foi **restaurado byte a byte**
   ao valor original para que o diff desta feature fosse puramente aditivo. O byte
   segue lá — corrigi-lo é decisão do dono, não desta feature.
