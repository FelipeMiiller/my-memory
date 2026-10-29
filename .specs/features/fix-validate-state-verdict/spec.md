# Feature: fix-validate-state-verdict

- **Spec ID**: `fix-validate-state-verdict`
- **Created**: 2026-10-29
- **Owner**: Felipe Miiller
- **Source**: ISSUE-033 (`.specs/ISSUES.md:1336`)
- **Related**: skill `tlc-spec-driven` (`.agents/skills/tlc-spec-driven/`) · sem ADR nova (ver `Out of Scope`)

## Problem Statement

O gate determinístico `validate_state.py` reprova **11 das 41 features** verificadas
do repositório — mas por **desenho do validador**, não por conteúdo dos relatórios.
A ISSUE-033 separa os dois grupos:

| Grupo | Features | Realidade |
| :--- | :--- | :--- |
| **Têm veredito de verdade, o validador não acha** | `embedder-builtin-onnx-minilm`, `event-runtime-event-log`, `feat-code-ast`, `post-release-v1.3.0-bugfixes`, `triptych-node-inspector`, `viewer-dashboard` | O relatório existe e é real; só o formato não é lido |
| **O `validation.md` é um PLANO, não um relatório** | `ai-tool-auto-wiring`, `context-packager-and-subgraph-bundle`, `graph-path-discovery`, `progressive-context-loading`, `staleness-detection-and-banners` | Nunca houve relatório de verificação — aqui o gate está **certo** |

**Dois defeitos no validador**, ambos em `.agents/skills/tlc-spec-driven/scripts/validate_state.py`:

1. **O fallback varre o documento inteiro.** Sem nenhuma linha candidata, o script usa
   `hay = text` — o texto inteiro como palha. Um relatório real que menciona as duas
   palavras em prosa (`viewer-dashboard`: 53 PASS e 2 FAIL) é classificado como
   **"template não preenchido"**. É um **falso positivo** do validador.
2. **O filtro não reconhece o formato do relatório.** As linhas candidatas eram só as
   que casavam com `^#{1,4}\s*validation\b` ou `\*{0,2}result\*{0,2}\s*:`. O veredito
   escrito como `**Verdict**: PASS`, `**Veredito**: PASS` ou `**Resultado**: …` — e
   uma linha que abre com `**PASS** — …` — não era candidata nenhuma.

**Um terceiro defeito, latente:** `PARTIAL` não existe como veredito. A linha de
veredito de `feat-code-ast` é `**Verdict**: **PARTIAL** — PASS for 14 ACs, SPEC-GAP
for 3 ACs (CA-01 / CA-12 / CA-16), 1 surviving mutant.` — ela contém a palavra
"PASS". Uma correção ingênua (só acrescentar `Verdict:`/`Veredito:`/`Resultado:`
como candidatas) transformaria uma feature **PARTIAL** em verde, apagando um gap
real do repositório.

**Estado inicial medido** (2026-10-29, `develop` @ `adfb8cc`):

```
validate_state: 11 error(s) across [41 features]
```

**Estado alvo medido**:

```
validate_state: 6 error(s) across [41 features]
  - 5 features de '# Validation Plan'  -> 'no PASS/FAIL verdict' (o gate está certo)
  - feat-code-ast                      -> 'verdict is PARTIAL' (o gate está certo)
  - 0 regressões entre as features que passam hoje
```

## Goals

- [ ] Fazer o validador ler o veredito das linhas que de fato o carregam, em todos os
      formatos de relatório já presentes no vault.
- [ ] Remover o fallback que varre o documento inteiro (falso "template não preenchido").
- [ ] Tornar `PARTIAL` um veredito próprio, reportado como tal e **nunca** como PASS.
- [ ] Normalizar os **6** relatórios que têm veredito real na forma canônica
      `## Validation: PASS` + `**Result**: PASS`, sem apagar conteúdo nem evidências.
- [ ] Preservar integralmente o conteúdo e as evidências dos 6 relatórios.
- [ ] Garantir **zero regressões**: toda feature que passa hoje continua passando.

## Out of Scope

- **As 5 features de `# Validation Plan`** (`ai-tool-auto-wiring`,
  `context-packager-and-subgraph-bundle`, `graph-path-discovery`,
  `progressive-context-loading`, `staleness-detection-and-banners`). O dono **adiou**
  a opção C da ISSUE-033 (escrever o relatório de verificação que nunca existiu).
  Elas **não** foram tocadas e o gate continua vermelho para elas — que é o estado
  honesto. Trabalho de verdade, item separado.
- **`.specs/ISSUES.md`, `.specs/STATE.md`, `docs/**`, `README.md`, `CHANGELOG.md`,
  `.memory/**`, `cmd/mem/**`, `internal/**`** — fora da propriedade desta execução.
- **Qualquer `validation.md` além dos 6 listados em P4** — em especial os 5 do grupo
  "Validation Plan".
- **ADR nova.** A mudança é de **comportamento de um script de gate**, não de schema
  de banco, driver de vetor ou arquitetura do produto. Nenhuma ADR-001/003/040 muda
  de sentido; criar uma ADR seria inventar uma decisão arquitetural que não existe.
  A decisão está registrada aqui, em `Assumptions & Open Questions` (A8).
- **`git commit` / `git push`** — o commit é do gestor (pai).

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| A1 — O que é uma "linha candidata" | `## Validation` (heading) · `Result:`/`Resultado:`/`Verdict:`/`Veredito:` (campo rotulado, com ou sem negrito, qualquer caixa) · linha que **abre** com o token de veredito (`**PASS** — …`) **como sujeito da frase** — um token que nomeia um campo (`- **PARTIAL / SPEC-GAP**: 0`) é contagem do relatório, não veredito | Os três formatos já existem no vault. O terceiro formato é o que segura `mcp-static-serving` e `obsidian-tag-parity`, que hoje só passam porque o fallback varre o texto inteiro — sem ele, esses dois virariam "sem veredito". A ressalva do campo evita que a seção de contagens de um relatório ("PASS: 16, PARTIAL: 0, FAIL: 0") decida o veredito por acidente | Sim |
| A2 — Sem nenhuma linha candidata | `None` → "no PASS/FAIL verdict" | Um relatório em prosa não é veredito. O documento inteiro deixa de ser palha — é a causa do falso "template não preenchido" em `viewer-dashboard` | Sim |
| A3 — Detecção de template não preenchido | O placeholder literal `PASS | FAIL` (com ou sem colchetes) | O proxy antigo ("a linha tem PASS **e** FAIL") é o que confunde `**Verdict**: **PASS-WITH-GAPS** — 47 PASS · 3 FAIL` com template. O placeholder literal é o que o template realmente escreve | Sim |
| A4 — `PARTIAL` | Veredito próprio, com precedência sobre PASS e FAIL no arquivo inteiro | `feat-code-ast` é PARTIAL de verdade: 14/17 ACs passam, 3 são SPEC-GAP e 1 mutante sobrevive. Deixar isso verde é apagar um gap real | Sim |
| A5 — Desempate | Sem `PARTIAL` no arquivo, vence o **primeiro** candidato decisivo (ordem do documento) | Determinístico e explicável: o veredito do autor vem primeiro no relatório | Sim |
| A6 — Evidência obrigatória | PASS continua exigindo ≥1 citação `arquivo:linha`; PARTIAL e FAIL não | Evidence-or-zero é o que distingue PASS de "ninguém checou" — mas só o PASS é um veredito de aprovação | Sim |
| A7 — Evidência dos 6 relatórios | Cada citação nova é conferida abrindo o arquivo e confirmando a linha | Não inventar evidência é o requisito; a citação do `embedder-builtin-onnx-minilm` aponta para o código que materializa os achados do spike | Sim |
| A8 — ADR | **Nenhuma** | Não há mudança de schema, driver ou arquitetura; a decisão (A1–A7) fica registrada nesta spec | Sim |

**Open questions**: none — todas as decisões estão resolvidas e registradas acima (A1 a A8).

## User Stories

### P1: O gate lê o veredito onde ele realmente está

**User Story**: Como pessoa que mantém o fluxo spec-driven, quero que o gate leia o
veredito nos formatos que os relatórios realmente usam, so that um relatório real
não seja reprovado por formato.

**Why P1**: 6 features com veredito real reprovavam por desenho. O gate é a
integridade do fluxo; um gate que reprova o certo é pior que gate nenhum.

**Acceptance Criteria**:
1. WHEN a linha do relatório traz `Result:`, `Resultado:`, `Verdict:` ou `Veredito:` THEN the system SHALL tratá-la como linha candidata de veredito.
2. WHEN uma linha do relatório começa com o token de veredito **enfatizado** como sujeito da frase THEN the system SHALL tratá-la como linha candidata de veredito AND SHALL NOT tratá-la como candidata quando o token nomeia um campo seguido de `:`.
3. WHEN o relatório não tem nenhuma linha candidata THEN the system SHALL reportar "sem veredito" AND SHALL NOT varrer o corpo do documento.
4. WHEN a linha de veredito contém o placeholder `PASS | FAIL` THEN the system SHALL reportar o veredito como template não preenchido.
5. WHEN um rótulo de veredito aparece dentro de um code span THEN the system SHALL NOT tratá-lo como declaração de veredito.

**Independent Test**: `python .agents/skills/tlc-spec-driven/scripts/test_validate_state.py` — casos
`sem candidata -> None`, `linha Veredito:/Resultado: reconhecida`, `template detectado`.

---

### P2: PARTIAL é um veredito próprio, nunca PASS

**User Story**: Como pessoa que decide se uma feature pode ir para produção, quero
que o gate diferencie "aprovado" de "aprovado pela metade", so that um PARTIAL nunca
seja lido como verde.

**Why P2**: `feat-code-ast` é PARTIAL de verdade (14/17 ACs, 3 SPEC-GAP, 1 mutante
sobrevivente). A linha de veredito dela contém a palavra "PASS". Sem um veredito
PARTIAL explícito, a correção de P1 transformaria um gap real em verde.

**Acceptance Criteria**:
1. WHEN o relatório declara PARTIAL THEN the system SHALL reportar o veredito como PARTIAL AND SHALL NOT reportá-lo como PASS.
2. WHEN uma linha de veredito declara PARTIAL e também menciona PASS THEN the system SHALL classificar a linha como PARTIAL.
3. WHEN qualquer linha candidata do relatório declara PARTIAL THEN the system SHALL reportar a feature como PARTIAL AND SHALL mantê-la vermelha.

**Independent Test**: `test_validate_state.py` — casos `PARTIAL com 'PASS' na mesma linha`
e `PARTIAL tem precedência sobre um PASS posterior`.

---

### P3: Zero regressões — o que passa hoje continua passando

**User Story**: Como dono do repositório, quero que a correção do gate não quebre
nenhuma feature já verificada, so that eu possa confiar no cross-check.

**Why P3**: O gate é o instrumento de confiança do fluxo. Se a correção reverte
features verdes, ela troca um falso positivo por um falso negativo.

**Acceptance Criteria**:
1. WHEN a feature passa o gate antes da mudança THEN the system SHALL continuar passando depois da mudança.
2. WHEN o relatório tem uma linha de prosa `Resultado:` sem palavra de veredito THEN the system SHALL NOT derivar veredito dela.
3. WHEN o relatório tem uma linha `**Verdict**: **PASS-WITH-GAPS** — 47 PASS · 3 FAIL` THEN the system SHALL classificá-la como PASS AND SHALL NOT classificá-la como template não preenchido.

**Independent Test**: cross-check do repositório inteiro antes e depois — a lista de
features que passa tem de ser a mesma, e o total de erros tem de cair de 11 para 6.

---

### P4: Os 6 relatórios com veredito real, normalizados

**User Story**: Como pessoa que lê o vault, quero que os relatórios que têm veredito
real falem a língua canônica do gate, so that o gate e o documento concordem.

**Why P4**: A opção B da ISSUE-033. Sem o bloco canônico, o gate continua Depends
de adivinhação de formato.

**Acceptance Criteria**:
1. WHEN um relatório tem veredito real PASS THEN the system SHALL acrescentar o bloco canônico `## Validation: PASS` + `**Result**: PASS`.
2. WHEN um relatório é normalizado THEN the system SHALL preservar todo o conteúdo e todas as evidências existentes AND SHALL NOT apagar nenhuma linha.
3. WHEN um relatório normalizado declara PASS THEN the system SHALL citar ao menos uma evidência `arquivo:linha` verificável no repositório.

**Independent Test**: abrir cada um dos 6 arquivos e conferir o bloco canônico, a
evidência `arquivo:linha` e que o diff é **aditivo**.

---

### P5: `feat-code-ast` fica PARTIAL, e vermelho

**User Story**: Como pessoa responsável por promover `feat-code-ast` a produção,
quero que o gate me diga "PARTIAL", so that eu não promote uma feature com 3 ACs
em SPEC-GAP.

**Why P5**: O próprio relatório pede fix-tasks para GAP-1/2/3 antes de promover. O
gate deve refletir esse pedido, não contradi-lo.

**Acceptance Criteria**:
1. WHEN o relatório de `feat-code-ast` é normalizado THEN the system SHALL expressar o veredito PARTIAL na forma canônica.
2. WHEN o gate roda o cross-check THEN the system SHALL reportar `feat-code-ast` como PARTIAL AND SHALL NOT reportá-la como PASS.
3. WHEN o relatório de `feat-code-ast` é normalizado THEN the system SHALL preservar as 42 evidências existentes AND SHALL NOT carimbar PASS.

**Independent Test**: `validate_state.py --root <repo>` → a saída tem de conter
`feat-code-ast: validation.md verdict is PARTIAL`.

## Edge Cases

- IF o relatório tem uma linha `## Validation` (sem veredito) e, 200 linhas abaixo,
  uma linha `**Result**: PASS` THEN the system SHALL ler o PASS da linha `**Result**`.
- IF o relatório tem um `## Validation: PASS` canônico e, antes dele, uma linha de
  prosa que contém "PASS" solto THEN the system SHALL NOT tratar a prosa como veredito.
- IF o relatório tem veredito `PASS` mas nenhuma citação `arquivo:linha` THEN the
  system SHALL reprovar por evidence-or-zero.
- IF o relatório é vazio THEN the system SHALL reportar "sem veredito".
- IF a feature não tem `validation.md` THEN the system SHALL reportar o erro de
  relatório ausente (comportamento já existente, preservado).

## Requirement Traceability

| Requirement ID | AC | Story | Arquivo | Fase | Status |
| -------------- | -- | ----- | ------- | ---- | ------ |
| VSV-01 | P1.1 | P1: O gate lê o veredito onde ele está | .agents/skills/tlc-spec-driven/scripts/validate_state.py | Execute | Verified |
| VSV-02 | P1.2 | P1: O gate lê o veredito onde ele está | .agents/skills/tlc-spec-driven/scripts/validate_state.py | Execute | Verified |
| VSV-03 | P1.3 | P1: O gate lê o veredito onde ele está | .agents/skills/tlc-spec-driven/scripts/validate_state.py | Execute | Verified |
| VSV-04 | P1.4 | P1: O gate lê o veredito onde ele está | .agents/skills/tlc-spec-driven/scripts/validate_state.py | Execute | Verified |
| VSV-05 | P2.1 | P2: PARTIAL é veredito próprio | .agents/skills/tlc-spec-driven/scripts/validate_state.py | Execute | Verified |
| VSV-06 | P2.2 | P2: PARTIAL é veredito próprio | .agents/skills/tlc-spec-driven/scripts/validate_state.py | Execute | Verified |
| VSV-07 | P2.3 | P2: PARTIAL é veredito próprio | .agents/skills/tlc-spec-driven/scripts/validate_state.py | Execute | Verified |
| VSV-08 | P3.1 | P3: Zero regressões | .agents/skills/tlc-spec-driven/scripts/validate_state.py | Execute | Verified |
| VSV-09 | P3.2 | P3: Zero regressões | .agents/skills/tlc-spec-driven/scripts/validate_state.py | Execute | Verified |
| VSV-10 | P3.3 | P3: Zero regressões | .agents/skills/tlc-spec-driven/scripts/validate_state.py | Execute | Verified |
| VSV-11 | P4.1 | P4: Os 6 relatórios normalizados | .specs/features/event-runtime-event-log/validation.md | Execute | Verified |
| VSV-12 | P4.2 | P4: Os 6 relatórios normalizados | .specs/features/post-release-v1.3.0-bugfixes/validation.md | Execute | Verified |
| VSV-13 | P4.3 | P4: Os 6 relatórios normalizados | .specs/features/triptych-node-inspector/validation.md | Execute | Verified |
| VSV-14 | P4.3 | P4: Os 6 relatórios normalizados | .specs/features/embedder-builtin-onnx-minilm/validation.md | Execute | Verified |
| VSV-15 | P4.1 | P4: Os 6 relatórios normalizados | .specs/features/viewer-dashboard/validation.md | Execute | Verified |
| VSV-16 | P5.1 | P5: feat-code-ast fica PARTIAL | .specs/features/feat-code-ast/validation.md | Execute | Verified |
| VSV-17 | P5.2 | P5: feat-code-ast fica PARTIAL | .specs/features/fix-validate-state-verdict/validation.md | Execute | Verified |
| VSV-18 | P5.3 | P5: feat-code-ast fica PARTIAL | .specs/features/feat-code-ast/validation.md | Execute | Verified |
| VSV-19 | P1.1 | P1: O gate lê o veredito onde ele está | .agents/skills/tlc-spec-driven/scripts/test_validate_state.py | Execute | Verified |
| VSV-20 | P2.2 | P2: PARTIAL é veredito próprio | .agents/skills/tlc-spec-driven/scripts/test_validate_state.py | Execute | Verified |
| VSV-21 | P3.1 | P3: Zero regressões | .agents/skills/tlc-spec-driven/scripts/test_validate_state.py | Execute | Verified |
| VSV-22 | P4.3 | P4: Os 6 relatórios normalizados | .agents/skills/tlc-spec-driven/scripts/test_validate_state.py | Execute | Verified |
| VSV-23 | P1.3 | P1: O gate lê o veredito onde ele está | .agents/skills/tlc-spec-driven/SKILL.md | Execute | Verified |
| VSV-24 | P5.2 | P5: feat-code-ast fica PARTIAL | .agents/skills/tlc-spec-driven/scripts/validate_state.py | Execute | Verified |
| VSV-25 | P1.5 | P1: O gate lê o veredito onde ele está | .agents/skills/tlc-spec-driven/scripts/validate_state.py | Execute | Verified |

**Coverage**: 25 total, 25 mapeados para tarefas, 0 não mapeados.

## Success Criteria

- [x] `python .agents/skills/tlc-spec-driven/scripts/test_validate_state.py` sai com exit 0.
- [x] `python .agents/skills/tlc-spec-driven/scripts/validate_state.py --root D:\repository\my-memory`
      reporta **6** erros: 5 "no PASS/FAIL verdict" (as features de `# Validation Plan`)
      + 1 "verdict is PARTIAL" (`feat-code-ast`).
- [x] Nenhuma das features que passavam antes reprovou depois (guard de regressão).
- [x] `feat-code-ast` continua **vermelha**, agora com o motivo certo.
- [x] Os 6 relatórios têm o bloco canônico, com evidência `arquivo:linha` verificável.
- [x] `validate_spec.py` e `validate_tasks.py` passam para esta feature.
