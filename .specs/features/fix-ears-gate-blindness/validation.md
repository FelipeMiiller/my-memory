# fix-ears-gate-blindness — Validation

**Date**: 2026-11-16
**Spec**: `.specs/features/fix-ears-gate-blindness/spec.md`
**Diff range**: `f76c268..368a077` (4 commits da feature: T1 `f76c268`, T2 `0b2e7dc`, T3 `69595c2`, T4 `368a077`)
**Verifier**: sub-agente independente (autor ≠ verificador), sessão `mvs_20b32f89fab94dccbad5d2bebc010733`
**Scratch do sensor**: `D:\.minimax\scratch\subagents\verifier-ears\` (fora do repo; nenhum arquivo do repo foi mutado)
**Ambiente**: Windows, PowerShell 5.1, Python 3.14.6, Go 1.27.0

---

## Resultado

**Resultado**: PASS

A correção do gate da ISSUE-050 está verificada de forma independente: a prova
empírica se inverte (o gate sai != 0 numa spec com AC sem SHALL depois de uma
linha em branco), o sensor mata 5 de 5 mutantes (incluindo o reset
reintroduzido), a medição fecha em 0 blocos cegos com vistos = presentes, e o
gate roda em 0 erros nas 68 specs do repo.

Quatro achados residuais foram registrados abaixo. Nenhum é bloqueante: três
são fora do escopo declarado da T1 (o Verifier reporta, não amplia escopo) e o
quarto é deriva de documentação.

---

## Task Completion

| Task | Commit | Status | Notes |
| ---- | ------ | ------ | ----- |
| T1 — o coletor volta a ver o bloco inteiro | `f76c268` | Done | reset de linha em branco removido; coleta extraída para `collect_ac_items()` |
| T2 — o teste de regressão do gate | `0b2e7dc` | Done | `test_validate_spec.py`, 13 testes, `main()` próprio sem pytest |
| T3 — a medição republicável | `69595c2` | Done | `count_ears_visibility.py`, sai != 0 quando há bloco cego |
| T4 — a remediação das specs reveladas | `368a077` | Done | 14 specs, +188/-45; 0 specs reprovadas pelo gate |
| T5 — a re-verificação independente | este relatório | Done | veredito PASS; `validate_state.py` exit 0 |

---

## Spec-Anchored Acceptance Criteria

Evidence-or-zero: cada AC abaixo tem citação `file:line` + a expressão
asserida. 10 de 10 ACs casam com o resultado definido na spec.

| AC | Critério (WHEN X THEN Y) | Resultado definido na spec | `file:line` + asserção | Resultado |
| -- | ------------------------ | -------------------------- | ------------------------ | --------- |
| AC1 | linha em branco depois do cabeçalho não zera a coleta | o coletor segue coletando | `.agents/skills/tlc-spec-driven/scripts/validate_spec.py:168` — `elif re.match(r"^#{1,3}\s", ln) or stripped.startswith("**"):` (sem `stripped == ""`) | Coberto |
| AC1 | (comportamento) | AC sem SHALL vira ERROR | `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py:95` — `assert len(errors) == 1` | Coberto |
| AC2 | o bloco termina só em heading ou linha iniciada por `**` | a lista seguinte não é julgada | `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py:139` (heading) e `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py:155` (linha `**`) | Coberto |
| AC3 | AC sem SHALL vira 1 ERROR nomeando linha e critério | mensagem com `L<linha>:` e o texto do critério | `.agents/skills/tlc-spec-driven/scripts/validate_spec.py:188` — `errors.append(f"L{i}: acceptance criterion has no SHALL (not testable): {item[:70]}")`; asserção em `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py:98` — `assert errors[0].startswith(f"L{line}:")` | Coberto |
| AC4 | AC com SHALL e sem keyword EARS vira 1 WARNING | warning, não erro | `.agents/skills/tlc-spec-driven/scripts/validate_spec.py:190`; asserção em `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py:177` — `assert errors == []` e `:178` — `assert len(warnings) == 1` | Coberto |
| AC5 | placeholder de template não vira erro | 0 erros, 0 warnings | `.agents/skills/tlc-spec-driven/scripts/validate_spec.py:166` — `if not PLACEHOLDER_RE.match(item)`; asserção em `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py:190` — `assert errors == []` | Coberto |
| AC6 | os dois formatos de bloco são vistos | mesmo número de erros nos dois formatos | `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py:129` — `assert len(errors_a) == len(errors_b) == 1` | Coberto |
| AC7 | a medição reporta 0 blocos cegos e vistos = presentes | 0 cegos, vistos = presentes | `.agents/skills/tlc-spec-driven/scripts/count_ears_visibility.py:213` — `return 1 if blind_blocks else 0`; execução real em "Gate Check" abaixo | Coberto |
| AC8 | nenhuma spec fica reprovada em silêncio | toda spec reprovada é corrigida ou dispensada por escrito | varredura das 68 specs: 0 reprovadas, 0 dispensas necessárias — ver "Gate Check" | Coberto |
| AC9 | reintroduzir o reset faz o teste falhar | a suite falha | `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py:95` — `assert len(errors) == 1` (falha com `AssertionError: esperado 1 erro, obtido []`) | Coberto |
| AC10 | a verificação é feita por um Verifier distinto do autor | este relatório | `.specs/features/fix-ears-gate-blindness/validation.md` | Coberto |

**Status**: 10 de 10 ACs cobertas, 0 gaps de precisão de spec.

---

## Prova empírica da ISSUE-050 (inversão)

Fixture: cópia do exemplar validado
`.specs/features/fix-doctor-unindexed-links/spec.md` com **um AC sem SHALL**
inserido no primeiro bloco de AC — bloco que tem a linha em branco logo
depois do cabeçalho (`spec.md:98` cabeçalho, `spec.md:99` linha em branco),
exatamente o formato que o template da skill ensina.

| # | Validador | Fixture | Exit | Saída |
| - | --------- | ------- | ---- | ----- |
| D | atual (repo) | exemplar **sem** modificação | **0** | `0 error(s), 0 warning(s)` — sanidade: o exemplar continua limpo |
| A | atual (repo) | com AC sem SHALL | **1** | `ERROR L100: acceptance criterion has no SHALL (not testable): O gate aceita este criterio malformado sem verbo obrigatorio.` |
| B | cópia com o reset reintroduzido | com AC sem SHALL | **0** | `0 error(s), 0 warning(s)` — cego de novo |
| C | versão pré-T1 (`f76c268^`) | com AC sem SHALL | **0** | `0 error(s), 0 warning(s)` — o verde-mentira original |

A inversão está provada: **a mesma fixture, o mesmo caminho de código, só a
condição de reset muda** — exit 1 com o coletor corrigido, exit 0 com o reset
de volta. O caso C reproduz, sobre a árvore real, o "exit 0" que a ISSUE-050
registrou.

---

## Discrimination Sensor

Scratch isolado por **cópia de arquivo** (sem `git stash`, sem worktree, sem
mutar a árvore real): o validador e o arquivo de teste foram copiados para
`D:\.minimax\scratch\subagents\verifier-ears\sensor\<mutante>\`, o validador
foi mutado, e a suite rodou contra a cópia (o `test_validate_spec.py:24` faz
`sys.path.insert(0, dirname(__file__))`, então importa a cópia mutada).

| # | Mutação | Arquivo:linha | Morto? |
| - | ------- | ------------- | ------ |
| CONTROLE | nenhuma (cópia limpa) | — | não aplicável (suite passa, exit 0) |
| M1 | reintroduz o reset `stripped == ""` | `.agents/skills/tlc-spec-driven/scripts/validate_spec.py:168` | **Morto** — `AssertionError: esperado 1 erro, obtido []` |
| M1b | reset reescrito como `not stripped` (não contém a string literal `stripped == ""`) | `.agents/skills/tlc-spec-driven/scripts/validate_spec.py:168` | **Morto** — `AssertionError: esperado 1 erro, obtido []` |
| M2 | bloco deixa de terminar em linha iniciada por `**` | `.agents/skills/tlc-spec-driven/scripts/validate_spec.py:168` | **Morto** — `test_block_ends_at_a_bold_line` (`:164`) |
| M3 | AC sem SHALL deixa de ser ERROR e vira warning | `.agents/skills/tlc-spec-driven/scripts/validate_spec.py:188` | **Morto** — `test_ac_without_shall_after_blank_line_is_error` (`:95`) |
| M4 | placeholder de template deixa de ser ignorado | `.agents/skills/tlc-spec-driven/scripts/validate_spec.py:166` | **Morto** — `test_placeholder_criterion_is_not_an_error` (`:190`) |

**5 de 5 mutantes mortos. 0 sobreviventes.**

M1b é a prova de que a suite discrimina **comportamento**, não texto: o
mutante evita a string literal que o teste de fonte
(`.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py:265`) procura, e
ainda assim é morto pelo teste **comportamental** da linha `:95`.

**Sensor depth**: leve com reforço (5 mutantes de comportamento, ≥ o mínimo de 3
da skill). A superfície da feature é o gate que decide o verde-mentira de 68
specs, por isso o sensor foi além do mínimo.

---

## Gate Check

### Gate da feature

| Gate | Comando | Resultado |
| ---- | ------- | --------- |
| Quick (T1) | `validate_spec.py fix-doctor-unindexed-links` | 0 erro, 0 warning — exit 0 |
| Quick (T2) | `test_validate_spec.py` | **13 testes, 13 verdes**, exit 0 |
| Quick (T3) | `count_ears_visibility.py --root D:\repository\my-memory` | `specs=68 \| vistos=898 presentes=898 cegos=0` · `blocos cegos: 0 \| specs com bloco cego: 0` · `gate ve o AC de prova: sim` — **exit 0** |
| Contrepo (T3) | `count_ears_visibility.py --legacy` | `specs=68 \| vistos=457 presentes=898 cegos=441` · `blocos cegos: 75` — exit 1 (comportamento de gate correto) |
| Full (T4) | `validate_spec.py` em **todas** as 68 specs | **0 specs reprovadas**, 0 erros no total, 73 warnings — exit 0 |
| Build | `go build ./...` | exit 0 |
| Full (T5) | `go test -count=1 ./...` (arvore real) | **28 de 28 pacotes verdes, exit 0** |
| Format | `gofmt -l .` | lista o repo inteiro — **artefato de CRLF do checkout Windows**, não da feature (ver Achado 4) |
| Spec | `validate_spec.py fix-ears-gate-blindness` | 0 erro, 0 warning — exit 0 |
| Tasks | `validate_tasks.py fix-ears-gate-blindness` | 0 erro, 0 warning — exit 0 |
| State | `validate_state.py fix-ears-gate-blindness` | **exit 0** (antes de escrever este relatório: exit 1, `no validation.md` — o gate não é vacuamente verde) |

### Nota sobre concorrencia de agentes na arvore

A **primeira** rodada de `go test -count=1 ./...` na arvore real deu 27 de 28
pacotes verdes, com `cmd/mem` falhando por
`cmd\mem\install.go:40:12: undefined: validateInstallFlags`. Esse arquivo estava
**modificado e nao commitado** por outro agente (`M cmd/mem/install.go` no
porcelain) e **nao faz parte do diff da feature** (`git diff --name-only
f76c268^..368a077` nao lista `cmd/mem/install.go`); a funcao foi definida em
`cmd/mem/install.go:191` quando o outro agente fechou a edicao. A **segunda**
rodada, com a edicao assentada, deu **28 de 28 pacotes verdes, exit 0**. Nenhuma
rodada acusou defeito atribuivel a esta feature.

### Medição final (a claim da T5)

```
count_ears_visibility: specs=68 | vistos=898 presentes=898 cegos=0
blocos cegos: 0 | specs com bloco cego: 0
gate ve o AC de prova (AC sem SHALL depois de linha em branco): sim
```
Exit code 0.

### Baseline da ISSUE-050 reproduzido

Medido na árvore da própria medição original (commit `398344b`, 2026-11-15),
extraída com `git archive` para fora do repo e medida com a regra antiga
(`--legacy`):

| Medida | ISSUE-050 / `spec.md:37-43` | Medido pelo script | Confere |
| ------ | -------------------------- | ------------------ | ------- |
| specs | 62 | 62 | sim |
| ACs vistos | 311 | 311 | sim |
| ACs presentes | 752 | 752 | sim |
| ACs cegos | 441 (59%) | 441 (58,6%) | sim |
| blocos cegos | 75 | 75 | sim |
| specs com bloco cego | **42** de 62 | **19** | **não** (Achado 2) |

### Contagem de testes

- Go: o repo tem **28 pacotes** de teste; nenhum teste foi removido por esta
  feature (o diff da feature não toca nenhum `.go`).
- Python: a feature **introduz** `test_validate_spec.py` com **13** funções de
  teste (antes: 0).

---

## Isolamento da árvore (a claim 5 da T5)

| Momento | HEAD | `git status --porcelain` |
| ------- | ---- | ------------------------ |
| Baseline (antes de tudo) | `368a077` | vazio (limpo) |
| Final | `3569255` | 2 modificados por outro agente (`cmd/mem/install_test.go`, `cmd/mem/pack.go`) + 1 nao rastreado: `validation.md` (este relatorio) |

**Nenhum `git stash` foi usado.** Todo o sensor rodou em cópia de arquivo em
`D:\.minimax\scratch\subagents\verifier-ears\`. As árvores pré-T1 e pré-T1-2026-11-15
foram extraídas com `git archive` (sem tocar `.git`, sem worktree).

A diferença entre o baseline e o final **não é do Verifier**: o HEAD andou
`368a077 → f6bdda9 → 70de4c6 → f465c13` e o conjunto de arquivos modificados
mudou a cada momento (`cmd/mem/install.go` + `internal/graph/pack_test.go` →
`context-packager.../tasks.md` + `docs/adr/029-...` + `internal/autowire/installer.go`
→ `cmd/mem/install_test.go`) porque
**outro agente está commitando e editando o repo em paralelo**. Nenhum arquivo
modificado no final foi tocado por esta verificação, e nada foi revertido. Os
4 commits da feature permanecem ancestrais do HEAD.

---

## Achados

### 1. Latente — AC multi-linha é julgado só pela primeira linha física (mesma família da ISSUE-050)

**Onde**: `.agents/skills/tlc-spec-driven/scripts/validate_spec.py:163` — o
regex `^\s*\d+\.\s+(.*)$` captura **uma** linha física; a continuação
indentada do AC não entra no texto julgado.

**Impacto comprovado**: o gate julga um critério **truncado**. Hoje há **42 ACs
multi-linha** no repo (ex.: `.specs/features/fix-doctor-unindexed-links/spec.md:102`,
`.specs/features/fix-release-onnx-provider/spec.md:78`). O mecanismo já
produziu falso ERROR: o commit `368a077` re-empacotou 6 ACs de
`fix-doctor-unindexed-links` justamente porque o "SHALL" estava na linha de
continuação e o gate acusava "no SHALL" num AC que o tinha.

**Estado atual**: **0 inversões de veredito** — em nenhum dos 42 ACs o
veredito mudaria se o texto completo fosse julgado. Logo, hoje o efeito é
latente, não ativo.

**Gatilho futuro**: uma spec nova com AC quebrado em duas linhas e o "SHALL"
na segunda linha → falso ERROR, e a spec é corrigida por um motivo errado (ou
descartada como dispensa).

**Status**: fora do escopo da T1 (que trata do reset de linha em branco).
Reportado, não corrigido — e não é um AC falho da feature.

### 2. Documentação — "42 de 62 specs" e "75, em 27 specs" não são reproduzíveis

**Onde**: `.specs/features/fix-ears-gate-blindness/spec.md:42` e `:43`, e a
mesma tabela em `.specs/ISSUES.md` (seção ISSUE-050).

**Evidência**: na árvore exata da medição original (`398344b`), o script
versionado mede **19** specs com bloco cego, não 42 nem 27 — e o número é 19
também se a varredura for alargada para **todos** os `.md` sob `.specs/` (201
arquivos) ou para `features+_archive+outros` (71 specs). O número de **blocos
cegos (75)** e de **ACs cegos (441)** reproduz exatamente.

**Impacto**: a headline "59%" (441/752) e os números que motivaram a correção
são exatos; a contagem de specs afetadas não é. Como a medição agora é
**versionada**, o número publicado deveria ser o do script.

**Status**: deriva de documentação; não afeta nenhum AC.

### 3. Residual, fora do escopo da T1 — cabeçalhos de AC que o validador não reconhece

**Onde**: `.agents/skills/tlc-spec-driven/scripts/validate_spec.py:159` — o
regex `^\*{0,2}Acceptance Criteria\*{0,2}\s*:?\s*$` não casa com `## Acceptance
Criteria` nem com `**Acceptance Criteria:**` (o `:` vem antes do `**`).

**Confirmado**: o próprio script de medição reporta
`diagnostico (fora da ISSUE-050): 48 AC(s) sob cabecalho que o validador nao
reconhece, em 11 bloco(s)`, em **6 specs**:

| Spec | Linha do cabeçalho | ACs fora do gate |
| ---- | ------------------ | ----------------- |
| `batch-edge-resolution-lookups` | `spec.md:55` (`## Acceptance Criteria`) | 5 |
| `embedder-builtin-onnx-minilm` | `spec.md:78, 94, 108, 121` (`**Acceptance Criteria:**`) | 17 |
| `fix-doctor-dead-links-code-nodes` | `spec.md:50` (`## Acceptance Criteria`) | 6 |
| `fix-doctor-tag-edges` | `spec.md:51` (`## Acceptance Criteria`) | 4 |
| `fix-drift-repo-context` | `spec.md:45` (`## Acceptance Criteria`) | 5 |
| `spike-onnxruntime-binding` | `spec.md:92, 106, 118` (`**Acceptance Criteria:**`) | 11 |

Total: **48 ACs em 6 specs** fora da leitura do gate — exatamente o número
apontado no briefing, confirmado por medição independente.

**Nota**: `post-release-v1.3.0-bugfixes/spec.md:116` usa `## Acceptance
Criteria (consolidado)`, mas com checklist (`- [x]`), não AC numerada — não
conta como AC para o coletor, e por isso não aparece no diagnóstico.

### 4. Ambiente — `gofmt -l` lista o repo inteiro (artefato de CRLF, não da feature)

`gofmt -l .` lista ~140 arquivos `.go`. A causa é o checkout no Windows com
CRLF: `.agents/.../validate_spec.py` está com 0 CRLF (LF puro), mas
`internal/autowire/detector.go` tem **350 CRLF para 350 LF** — todas as linhas.
O diff da feature **não toca nenhum `.go`** (`git diff --name-only
f76c268^..368a077` mostra só `.py` e `.specs/**/*.md` da feature). Não é um
achado contra a feature.

---

## Code Quality

| Princípio | Status |
| --------- | ------ |
| Escopo mínimo (só o coletor) | ok — o diff de `f76c268` é a extração da coleta + a remoção de uma condição |
| Mudança cirúrgica | ok — uma condição; o resto do validador não muda de comportamento |
| Sem scope creep | ok — sem feature além do pedido; sem "flexibilidade" extra |
| Só arquivos necessários | ok — 3 scripts + 14 specs + spec/tasks da feature |
| Padrão do repo | ok — `assert` + `main()` próprio, sem pytest, como `test_validate_state.py` |
| Testes mapeados a AC | ok — 13 testes, cada um citando o EAG que cobre |
| Spec-anchored outcome check | ok — as asserções miram o valor exato definido na spec (`L<linha>:` + texto do critério) |
| Todo teste mapeado a um AC/done-when | ok — nenhum teste órfão |
| Diretrizes do projeto | `AGENTS.md` (regras 4 e 8), `.agents/rules/always-quality-gate.md`, `.agents/skills/tlc-spec-driven/scripts/test_validate_state.py:15` (convenção do teste de script) |
| Codificação dos arquivos novos | ok — os 3 scripts: UTF-8 **sem BOM**, LF puro, 0 CRLF |

---

## Edge Cases

| Caso (spec.md:175-189) | Verificado | Evidência |
| ----------------------- | ---------- | --------- |
| Vários blocos de AC → contagem é a soma | ok | `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py:237` |
| Bloco de AC vazio → sem erro | ok | `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py:211` (spec bem formada sem erro) |
| AC placeholder → ignorado | ok | `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py:184` |
| Lista numerada fora de bloco de AC → não julgada | ok | `.agents/skills/tlc-spec-driven/scripts/test_validate_spec.py:196` |
| Correção revela erros reais → spec corrigida, nunca afrouxada | ok | 14 specs corrigidas em `368a077`; **nenhum gate foi afrouxado** — o diff do `validate_spec.py` em `f76c268` mexe só na condição de fim de bloco |
| Medição antes da correção reproduz o baseline | ok | `--legacy` reproduz 441 cegos / 75 blocos; ver Achado 2 para a contagem de specs |

---

## Não verificado

- **Atribuição da primeira falha do `go test`.** A primeira rodada na árvore real
  falhou em `cmd/mem` por causa de uma edição concorrente de outro agente
  (arquivo não commitado, fora do diff da feature); a segunda rodada, com a
  edição assentada, deu **28 de 28 pacotes verdes, exit 0**. Uma rodada extra
  foi feita numa cópia isolada do commit `368a077` (`git archive` para fora do
  repo), onde `cmd/mem` passou e só `internal/repo` falhou — artefato da
  extração, sem `.git` para `TestDetectRepository_CurrentRepo` resolver o slug
  (esse pacote passou na árvore real, exit 0). Nada do gate Go fica em aberto;
  o que resta é a impossibilidade de um gate Go determinístico num repo com
  agentes escrevendo em paralelo.
- **Suíte do `viewer/` (frontend)** — fora do diff da feature (nenhum arquivo
  de `viewer/` tocado); não rodada.
- **`lessons.py` / camada de lições** — a skill exige destilar lições após a
  validação; a destilacao (distill) é do orquestrador, não deste relatório.
- **EAG-08 e EAG-10 ainda estão como `Pending`** em
  `.specs/features/fix-ears-gate-blindness/spec.md:202` e `:204`. Os dois são
  objetivamente satisfeitos (verificado acima), mas `spec.md` é do autor — este
  Verifier não altera arquivo da feature além deste relatório.
- **Reprodução do número "42 specs"** da ISSUE-050: o scratch original foi
  descartado pelo autor; a regra de contagem usada nele não é recuperável. O
  número do script versionado é 19 (Achado 2).

---

## Summary

**Overall**: pronto — o gate deixou de ser cego, e isso está provado de forma
independente.

**Resultado**: PASS

**Spec-anchored check**: 10 de 10 ACs casam com o resultado definido na spec;
0 gaps de precisão de spec.
**Sensor**: 5 de 5 mutantes mortos (incluindo o reset reintroduzido e uma
variante reescrita que prova a discriminação comportamental).
**Gate**: 0 erros em 68 specs; 13 de 13 testes Python verdes; medição com
vistos = presentes = 898 e 0 blocos cegos; `validate_state.py` exit 0.

**O que funciona**: o reset de linha em branco saiu do coletor
(`validate_spec.py:168`); a suite de regressão mata o mutante tanto na forma
literal quanto na forma reescrita; a medição é republicável e fecha a conta
(vistos = presentes); nenhuma spec reprova; nenhum gate foi afrouxado.

**Achados**: 4 residuais, nenhum bloqueante — 1 latente (AC multi-linha julgado
só pela 1ª linha), 1 de documentação ("42 specs" não reproduzível), 1 residual
fora de escopo (48 ACs em 6 specs sob cabeçalho não reconhecido — confirmado),
1 de ambiente (CRLF no `gofmt -l`).

**Próximo passo**: fechar os Achados 1 e 2 em feature própria (o Achado 1 é
material para uma ISSUE nova: mesma família da ISSUE-050, gate vendo menos do
que está escrito) e atualizar `spec.md:202` e `spec.md:204` para Verified.