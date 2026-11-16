# Feature: fix-ears-gate-blindness

- **Spec ID**: `fix-ears-gate-blindness`
- **Created**: 2026-11-16
- **Owner**: Felipe Miiller
- **Source**: ISSUE-050 — `.specs/ISSUES.md` (seção ISSUE-050) · fecha a cegueira do gate de EARS
- **Related**: `.agents/skills/tlc-spec-driven/scripts/validate_spec.py:174` · `references/specify.md` (template) · ISSUE-033 (o gate que não distingue relatório de plano) · `.specs/features/fix-doctor-unindexed-links/spec.md:98` (exemplar mais validado do repo)

## Problem Statement

O gate que valida "todo AC tem SHALL e casa com um padrão EARS" **não enxerga
59% dos ACs do repo**. Uma spec com AC malformado passa com **0 erro** — é o
"verde-mentira" que o repo já registrou na ISSUE-033, agora na camada do
validador de spec.

**A causa — uma linha.** Em `.agents/skills/tlc-spec-driven/scripts/validate_spec.py:174`:

```python
elif stripped == "" or re.match(r"^#{1,3}\s", ln) or stripped.startswith("**"):
    in_ac = False
```

A condição que encerra o bloco de Acceptance Criteria inclui `stripped == ""`.
Ou seja: **uma linha em branco logo depois de `**Acceptance Criteria**:` zera o
flag `in_ac`** e **todos** os ACs daquele bloco ficam invisíveis ao gate. O
regex de coleta (`^\s*\d+\.\s+(.*)$`) não é o problema — o problema é o reset.

**A prova (empírica, fora do repo).** Cópia do `spec.md` do exemplar validado
(`fix-doctor-unindexed-links`) com **um AC deliberadamente sem SHALL** inserido:

```
validate_spec: 0 error(s), 0 warning(s)   <- exit 0
```

**O raio de explosão (medido em 2026-11-15).**

| Medida | Valor |
| :--- | :---: |
| ACs que o gate **vê** hoje | **311** |
| ACs que o gate **deveria** ver | **752** |
| **ACs cegos pelo gate** | **441** (59%) |
| specs com pelo menos 1 bloco cego | **42** de 62 |
| blocos `**Acceptance Criteria**:` afetados | **75**, em 27 specs |

**Por que ninguém viu antes.** O padrão (linha em branco depois do cabeçalho) é
o que a **template da skill** produz: o próprio `references/specify.md` mostra
`**Acceptance Criteria** (each line is one EARS pattern):` seguido de lista, e o
exemplar mais validado do repo (`fix-doctor-unindexed-links`, `spec.md:98-99`)
repete o padrão em 3 blocos. **O formato que a skill ensina é exatamente o
formato que cega o gate dela.**

Esta spec fecha a cegueira **no coletor** (não no formato), prova a correção com
um teste de regressão, e **remedia** as specs que o gate passar a apontar —
porque o conserto da linha **revela** 441 ACs que estavam sem prova, e alguns
deles podem estar realmente malformados.

## Goals

- [ ] O gate enxerga **todos** os ACs de um bloco, com ou sem linha em branco
      depois do cabeçalho.
- [ ] O fim do bloco é um **heading** ou uma linha iniciada por `**` — nunca uma
      linha vazia.
- [ ] Existe **teste de regressão** do gate (a convenção `test_validate_state.py`).
- [ ] Existe **medição** republicável da cegueira (quantos ACs o gate vê vs
      quantos existem).
- [ ] Nenhuma spec fica **aprovando por cegueira**: toda spec que o gate passa a
      reprovar está corrigida ou tem dispensa explícita.
- [ ] O template da skill e o exemplar do repo **não** precisam mudar de
      formato.

## Out of Scope

| Item | Reason |
| :--- | :--- |
| Mudar o **template** `references/specify.md` para tirar a linha em branco | o formato do template é **legítimo** (o Obsidian e o GitHub renderizam assim). O defeito é do coletor. Se o coletor passar a ver os dois formatos, nem o template nem as 42 specs precisam mudar. |
| Reescrever as **42 specs** cegas | o conserto do gate é de uma linha; as specs só precisam mudar se o gate, já cego-corrompido, apontar erro real nelas (e aí o AC exige a correção). |
| Transformar o **WARNING** de "sem keyword EARS" em ERROR | isso criaria uma classe nova de falhas além do defeito. O warning continua warning — mas passa a ser **medido**. |
| Tocar no `validate_tasks.py` ou no `validate_state.py` | a cegueira medida é do `validate_spec.py`. O `validate_state.py` tem a lição da ISSUE-033 já endereçada. |
| Mexer no `lessons.py` | é outro script, sem cegueira medida. |

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| **Onde** a cegueira é corrigida | no **coletor** do `validate_spec.py` (a condição da linha 174) | corrigir o formato exigiria mudar 42 specs + o template da skill — e qualquer spec escrita à mão por outro agente voltaria a cegar o gate. O coletor é o ponto único de falha. | y — **decidido** |
| O fim do bloco de AC | **heading** (`#{1,3}`) ou linha iniciada por `**` | são os dois marcos que realmente encerram um bloco. A linha vazia é **espaçamento**, não estrutura. | y — **decidido** |
| O warning de "SHALL sem keyword EARS" | continua **WARNING** | o defeito é a cegueira, não a severidade. Transformar o warning em error passaria a reprovar 62 specs por estilo — e o `--strict` já existe para quem quiser. | y — **decidido** |
| Como as specs que o gate passar a reprovar são tratadas | os ACs são **corrigidos** (SHALL + padrão EARS); **nunca** se afrouxa o gate | a ISSUE-033 já registrou o padrão do repo: "verde-mentira" é proibido. Se um AC é malformado, ele se conserta. | y — **decidido** |
| Dispensa para specs **arquivadas** ou históricas | permitida, **com o motivo registrado dentro da própria spec** | nem toda spec do repo é spec viva (`.specs/_archive/` e as specs de features já encerradas). Uma dispensa silenciosa seria outro verde-mentira; uma dispensa escrita é auditável. | y — **decidido** |
| Onde mora a **medição** | um script ao lado do validador: `.agents/skills/tlc-spec-driven/scripts/count_ears_visibility.py` | a medição precisa ser **republicável** (o número da ISSUE-050 foi medido fora do repo, num scratch descartado). O script fica na mesma família do gate. | y — **decidido** |
| O método da medição | contar os ACs **presentes** pelo mesmo caminho do coletor (número + AC não-placeholder), e os **vistos** pelo gate | é a única comparação que fecha: "vistos = presentes" é o critério de cegueira zero. A medição da ISSUE-050 (311 vs 752) é o baseline a reproduzir. | y — **decidido** |
| O teste do gate | `test_validate_spec.py`, com `assert` + `main()` próprio (sem pytest) | é exatamente a convenção do `test_validate_state.py` que já existe na skill (`.agents/skills/tlc-spec-driven/scripts/test_validate_state.py:15`). | y — **decidido** |
| O template e o exemplar mudam? | **não** | o AC exige que o gate veja os **dois** formatos (com e sem linha em branco). Se o template mudasse, o gate ainda estaria cego para specs escritas por fora — o que seria o mesmo defeito com outro nome. | y — **decidido** |

Open questions: none — as decisões de onde corrigir, como medir e como tratar
as specs que o gate passar a apontar estão tomadas acima.

## User Stories

### P1: O gate enxerga todos os ACs ⭐ MVP

**User Story**: Como quem mantém a spec, quero que o gate leia todos os meus
critérios de aceite, para que um critério malformado seja **erro** e não uma
linha invisível.

**Why P1**: é o defeito da ISSUE-050 — hoje 441 ACs (59%) são invisíveis ao gate.

**Acceptance Criteria**:
1. WHEN the Acceptance Criteria block is followed by a blank line THEN the validator SHALL keep collecting the numbered criteria of that block.
2. The collection of a block SHALL end only at a heading or at a line that starts with `**`.
3. WHEN a numbered criterion has no SHALL THEN the validator SHALL report one ERROR naming the line and the criterion.
4. WHEN a numbered criterion has a SHALL but no EARS lead keyword THEN the validator SHALL report one WARNING.
5. WHEN a numbered criterion is still a template placeholder THEN the validator SHALL NOT report it as an error.

**Independent Test**: um `spec.md` de fixture com AC malformado **depois** de
uma linha em branco → o gate reporta 1 erro.

---

### P1: Os dois formatos são válidos ⭐ MVP

**User Story**: Como autor de spec, quero escrever o bloco do jeito que o
template da skill ensina, para que eu não precise mudar de formato para ser
verificado.

**Why P1**: o formato do template é o que cega o gate — corrigir o formato
deixaria o gate cego para qualquer spec escrita por fora.

**Acceptance Criteria**:
6. The validator SHALL see the criteria of a block written with a blank line after the header and of a block written without it — both formats are valid.

**Independent Test**: a mesma spec, nos dois formatos, produz o **mesmo**
número de erros.

---

### P2: A medição é republicável

**User Story**: Como dono do repo, quero a ceguerreira **medida** por um
script versionado, para que o número não dependa de um scratch descartado.

**Acceptance Criteria**:
7. The measurement of the repository SHALL report zero blind blocks after the correction, and the count of criteria seen by the gate SHALL equal the count of criteria present.

**Independent Test**: o script imprime "vistos = presentes" e 0 blocos cegos.

---

### P2: Nenhuma spec passa por cegueira

**User Story**: Como dono do repo, quero que a revelação dos 441 ACs não deixe
spec reprovada em silêncio, para que a bateria volte a ser confiável.

**Acceptance Criteria**:
8. WHEN the gate reports a spec as failing after the correction THEN that spec SHALL have its criteria fixed or an explicit exemption recorded in it.

**Independent Test**: rodar o gate em todas as specs do repo → nenhuma falha
sem correção ou dispensa.

---

### P1: A correção é re-verificada de forma independente ⭐ MVP

**User Story**: Como dono do repo, quero que a correção do **gate** seja
verificada por um Verifier independente, para que a própria ferramenta de
prova não fique sem prova.

**Acceptance Criteria**:
9. WHEN the blind reset is reintroduced into the gate THEN the regression test suite of the gate SHALL fail.
10. The verification of this correction SHALL be produced by a Verifier distinct from the author of the correction.

**Independent Test**: reintroduzir a linha `stripped == ""` no coletor → o
`test_validate_spec.py` falha.

## Edge Cases

- IF uma spec tem **vários** blocos de Acceptance Criteria THEN cada bloco é
  coletado independentemente, e a contagem é a **soma** deles.
- IF uma spec tem um bloco de AC **vazio** (só o cabeçalho) THEN o gate não
  reporta erro — não há critério para judging.
- IF um AC é um **placeholder** de template (`[Something]`) THEN ele é
  ignorado (a spec ainda não foi preenchida) — a isenção continua.
- IF o AC está **fora** de um bloco de AC (uma lista numerada qualquer) THEN o
  gate não o julga: a coleta só vale dentro de um bloco declarado.
- IF a correção revela **erros reais** em specs antigas THEN a spec é corrigida
  (não afrouxada); se a spec for **histórica/arquivada**, a dispensa vai
  escrita dentro dela, com o motivo.
- IF a medição for rodada **antes** da correção THEN ela reproduz o baseline
  da ISSUE-050 (441 ACs cegos) — é o contra-teste da prova.

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Where | Status |
| :--- | :--- | :--- | :--- |
| EAG-01 | AC1 — a linha em branco não zera a coleta | `.agents/skills/tlc-spec-driven/scripts/validate_spec.py` | Verified |
| EAG-02 | AC2 — o bloco só termina em heading ou `**` | `.agents/skills/tlc-spec-driven/scripts/validate_spec.py` | Verified |
| EAG-03 | AC3 — AC sem SHALL vira ERROR nomeando linha e critério | `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py` | Verified |
| EAG-04 | AC4 — AC com SHALL e sem keyword vira WARNING | `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py` | Verified |
| EAG-05 | AC5 — placeholder de template não vira erro | `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py` | Verified |
| EAG-06 | AC6 — os dois formatos de bloco são vistos | `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py` | Verified |
| EAG-07 | AC7 — a medição reportou 0 blocos cegos e vistos = presentes | `.agents/skills/tlc-spec-driven/scripts/count_ears_visibility.py` | Verified |
| EAG-08 | AC8 — nenhuma spec fica falhando em silêncio | `.specs/features/` | Pending |
| EAG-09 | AC9 — reintroduzir o reset faz o teste falhar | `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py` | Verified |
| EAG-10 | AC10 — o Verifier é distinto do autor | `.specs/features/fix-ears-gate-blindness/validation.md` | Pending |

**Coverage**: 10 total, 10 mapeados, 0 unmapped.

## Success Criteria

- [ ] O gate reporta **0** blocos cegos nas 62 specs do repo, e o número de ACs
      vistos é igual ao de ACs presentes (baseline da ISSUE-050: 311 vistos vs
      752 presentes).
- [ ] Uma spec com um AC sem SHALL, **depois** de uma linha em branco, faz o
      gate sair com código **!= 0** — a prova empírica que o gate deixou de ser
      cego.
- [ ] O `test_validate_spec.py` roda sem pytest e falha se a linha em branco
      voltar a zerar a coleta.
- [ ] Nenhuma spec do repo fica reprovada pelo gate sem correção ou dispensa
      escrita.
- [ ] O template `references/specify.md` e o exemplar
      `fix-doctor-unindexed-links/spec.md` **não** mudaram de formato — e
      continuam passando com 0 erro.
- [ ] A medição é republicável: `count_ears_visibility.py --root <repo>`
      imprime o número.
