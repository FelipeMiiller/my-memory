# fix-doctor-unindexed-links Validation

**Date**: 2026-11-14
**Spec**: `.specs/features/fix-doctor-unindexed-links/spec.md`
**Diff range**: `b8350fd..b6cf2db`
**Verifier**: sub-agente `verifier` (rodada 1: achados e medição) + sub-agente `verifier` (rodada 2: gates e medição) + sensor executado no pai — **ressalva de independência declarada** na seção "Ressalva de independência".

## Validation: PASS

**Result**: PASS

> **Ressalva de independência (leia antes de usar este veredito).** O skill exige
> `author ≠ verifier`. O que aconteceu: a **rodada 1** do Verifier independente
> rodou e achou **3 defeitos reais** (o gate completo estava vermelho, o
> `gofmt` da CI quebrado e um gap de precisão na spec) — a rodada 1 encerrou o
> turno antes do sensor; a **rodada 2** rodou os **dois gates** e a **medição
> real** de forma independente e os confirmou verdes, e também encerrou o turno
> antes do sensor. O **sensor de discriminação** (6 mutações) foi executado
> **no pai**, pelo fallback standalone do próprio skill ("without sub-agents,
> run validate.md as an independent fresh-eyes pass ... including the
> spec-anchored check and discrimination sensor"). Ou seja: **gate e medição
> são independentes; o sensor é do autor.** A evidência `file:line` dos ACs foi
> derivada do código e dos testes, não herdada do autor.
>
> A spec 2 tem a mesma ressalva declarada (`docs/../.specs/STATE.md`).

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1 — resolver a raiz do vault | ✅ Done | `internal/vaultfs/root.go` + 8 testes; precedência flag > catálogo do repo > `central_vault.path` (só o cofre central) > cwd; read-only |
| T2 — resolver o alvo de wikilink em disco | ✅ Done | `internal/vaultfs/target.go` + 9 testes; basename, caminho relativo, `title:`, `alias:`; match exato; devolve **como** casou |
| T3 — categorias novas e Health Score | ✅ Done | `internal/store/doctor_links.go` (classificador compartilhado) + `doctor.go` (fórmula) + 4 testes / 5 sub-casos |
| T4 — classificação no SQLite | ✅ Done | `internal/db/graph.go` + 5 testes |
| T5 — classificação no Postgres | ✅ Done | `internal/store/doctor.go` + interface `Store` + 6 testes contra Postgres real |
| T6 — flag `--vault` e relatório na CLI | ✅ Done | `cmd/mem/main.go` + 6 testes |
| T7 — espelho das categorias no MCP | ✅ Done | `internal/mcp/handlers.go` + `TestFormatDoctorReport_TresCategoriasDeLink` |
| T8 — contrato do wire (gap da rodada 1) | ✅ Done | `74e8ec7` — as duas listas novas normalizadas; guarda-cláusula do `health_score` agora confere a chave |
| T9 — `gofmt` do `DoctorReport` (gap da rodada 1) | ✅ Done | `1af7b4a` |

**Cobertura**: 9 tarefas, 9 concluídas, 0 pendentes. A fase 4 (T8, T9) foi a
rodada de correção aberta pelo Verifier.

---

## Spec-Anchored Acceptance Criteria

| Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion | Result |
| ------------------------- | -------------------- | ----------------------- | ------ |
| **AC1** — alvo em disco **não** é dead link | `DeadLinks == 0`, o alvo vai para a categoria nova | `internal/vaultfs/target_test.go:51` — `TestVaultTargetBasenameEmQualquerProfundidade`; `internal/vaultfs/target_test.go:107` — `TestVaultTargetResolvePeloTitle`; `internal/vaultfs/target_test.go:125` — `TestVaultTargetResolvePeloAlias`; `internal/db/doctor_unindexed_test.go:84` — `"dead links = %d; want 0 (o alvo existe em disco)"`; `internal/store/doctor_unindexed_test.go:80` — mesma asserção no Postgres | ✅ PASS |
| **AC2** — alvo ausente **continua** dead link | `DeadLinks == 1`, `OutOfIndexLinks == 0` | `internal/vaultfs/target_test.go:169` — `TestVaultTargetAlvoInexistenteNaoERro`; `internal/db/doctor_unindexed_test.go:119` — `"dead links = %d; want 1 (o alvo nao existe em disco)"`; `internal/store/doctor_unindexed_test.go:115` — mesma asserção no Postgres | ✅ PASS |
| **AC3** — a categoria nova traz alvo, origem, relação e path | 4 campos conferidos um a um | `internal/db/doctor_unindexed_test.go:90` (`TargetID`), `:93` (`SourceID`), `:96` (`Relation`), `:99` (`ResolvedPath`); `internal/store/doctor_unindexed_test.go:86`/`:89`/`:92`/`:95`; `internal/store/doctor_score_test.go:118` — `"caminho resolvido esperado %q"` | ✅ PASS |
| **AC4** — a categoria nova **não** penaliza o score | score **100** com 1 link fora do índice | `internal/store/doctor_score_test.go:26` — caso `"link fora do indice por decisao nao penaliza"`, `esperado: 100`, asserção em `:66` (`score != c.esperado`); `internal/db/doctor_unindexed_test.go:102` — `"health score = %d; want 100 (a categoria nova nao penaliza)"`; `internal/store/doctor_unindexed_test.go:98` — mesma asserção no Postgres | ✅ PASS |
| **AC5** — raiz inacessível: não verificável, mantém a penalidade, score provisório | `UnverifiableLinks == 1`, `DeadLinks == 0`, score **95**, `HealthScoreProvisional == true` | `internal/vaultfs/root_test.go:131` — `TestVaultRootInacessivelQuandoNaoExiste`; `internal/store/doctor_score_test.go:75` — `TestHealthScoreProvisionalQuandoHaLinkNaoVerificavel` (`:86`, `:89`); `internal/store/doctor_score_test.go:139` — `TestClassifyLinksSemRaizNaoDeclaraRompido` (`:146`, `:149`, `:152`); `internal/db/doctor_unindexed_test.go:194` — `TestDoctorUnindexedSemRaizFicaProvisorio` (`:203`, `:206`, `:212` — `"health score = %d; want 95"`); `internal/store/doctor_unindexed_test.go:186` (`:196`, `:199`, `:205`); `cmd/mem/doctor_vault_test.go:119` — `TestDoctorVaultRaizInacessivelFicaProvisorio` (`:136` — `"o relatório precisa avisar que o score é provisório"`) | ✅ PASS |
| **AC6** — a correção apaga **exatamente** a lista do relatório | aresta viva **preservada** (1), aresta rompida **apagada** (0) | `internal/db/doctor_unindexed_test.go:136` — `TestDoctorUnindexedCorrecaoApagaSoORompido` (`:161` — `"aresta com alvo em disco depois da correcao = %d; want 1"`, `:164` — `"aresta rompida depois da correcao = %d; want 0"`); `internal/store/doctor_unindexed_test.go:128` (`:154`, `:157`) | ✅ PASS |
| **AC7** — a correção **preserva** o não verificável | aresta restante == 1 (sem prova, não se apaga) | `internal/db/doctor_unindexed_test.go:171` — `TestDoctorUnindexedCorrecaoPreservaNaoVerificavel` (`:187` — `"arestas depois da correcao = %d; want 1 (sem prova, a aresta e preservada)"`); `internal/store/doctor_unindexed_test.go:163` (`:179`) | ✅ PASS |
| **AC8** — o MCP expõe as mesmas categorias, com as mesmas contagens | contagem do MCP == contagem da CLI; as três categorias presentes; contrato do wire sem `null` | `cmd/mem/doctor_vault_test.go:142` — `TestDoctorVaultMCPECLIPontamOMesmoNumero` (`:167`, `:170`, `:173`); `cmd/mem/doctor_vault_test.go:180` — `TestDoctorVaultPayloadMCPCarregaAsTresCategorias` (`:199`, `:202`, `:205`, `:208`, `:211`); `internal/mcp/handlers_test.go:1330` — `TestFormatDoctorReport_TresCategoriasDeLink`; `internal/mcp/structured_content_test.go:244` — caso `"doctor"` exige `"out_of_index_links":[]` e `"unverifiable_links":[]`; `internal/mcp/handlers_test.go:959` — o guarda-cláusula confere a **chave** `health_score` | ✅ PASS |
| **AC9** — flag `--vault` no CLI | a flag muda a raiz checada; sem flag, o link continua rompido | `internal/vaultfs/root_test.go:36` — `TestVaultRootFlagVenceConfigGlobal`; `cmd/mem/doctor_vault_test.go:73` — `TestDoctorVaultFlagMudaARaiz` (`:83`, `:86`, `:89`, `:92`); `cmd/mem/doctor_vault_test.go:99` — `TestDoctorVaultSemFlagMantemLinkRompido` (`:109`, `:112`) | ✅ PASS |

**Status**: ✅ 9/9 ACs cobertas com evidência `file:line`, 0 gap de precisão.

---

## Discrimination Sensor

Executado no worktree isolado `D:\.minimax\scratch\subagents\verifier-spec3-r2\wt`
(`HEAD b6cf2db`), com o script `D:\.minimax\scratch\spec3_sensor\sensor.ps1`.
Cada mutação foi aplicada por intervalo de linhas, o teste-alvo foi rodado
contra o worktree e o arquivo foi revertido com `git checkout --`. **Nenhuma
mutação foi aplicada na árvore real**: o `git status --porcelain` da árvore real
e do worktree ficaram **vazios** no fim (baseline conferido).

| Mutation | File:line | Description | Killed? |
| -------- | --------- | ----------- | ------- |
| M1 | `internal/vaultfs/target.go:156` | O match do basename virou **prefixo** (fuzzy) — `[[Kubernetes]]` casaria com qualquer `kube*` | ✅ Killed |
| M2 | `internal/vaultfs/target.go:162` | A resolução por **`title:`/`alias:`** foi removida | ✅ Killed |
| M3 | `internal/store/doctor_links.go:78` | Link **fora do índice** passou a ser classificado como **rompido** | ✅ Killed |
| M4 | `internal/store/doctor.go:45` | A categoria fora do índice voltou a **penalizar** o Health Score | ✅ Killed |
| M5 | `internal/mcp/structured_content.go:65` | As duas normalizações do wire sumiram (as listas voltariam a `null`) | ✅ Killed |
| M6 | `internal/mcp/handlers_test.go:955` | O guarda-cláusula do `health_score` voltou a procurar **substring** | ✅ Killed |

**Sensor depth**: lightweight (6 mutações de comportamento, cobrindo o caminho
novo e **as duas correções** da fase 4).
**Result**: **6/6 killed** — PASS ✅

Nenhum mutante sobreviveu → nenhum gap de teste fraco.

---

## Gate Check

- **Gate command** (Full, com Postgres real): `go test -count=1 ./...`
- **Resultado**: **1325 testes · 1321 passaram · 0 falharam · 4 pulados** ·
  27 pacotes `ok` · exit **0**
- **Gate command** (Tagged, provider ONNX real **e** Postgres):
  `go test -tags "sqlite_fts5 onnx" -count=1 ./...`
  - **Resultado**: **1337 testes · 1337 passaram · 0 falharam · 5 pulados** ·
    27 pacotes `ok` · exit **0** — os **6 testes Postgres da feature**
    (`TestPostgresDoctorUnindexed*`) rodaram e passaram
  - sem Postgres (rodada do Verifier): 1303 passaram · 39 pulados (todos de
    integração Postgres) · 0 falharam
- **Test count before the feature** (`b8350fd`): **+39 funções de teste**, 0 removidas
  (delta medido pelo Verifier com `go test -list` nos dois worktrees)
- **Skipped tests** — todos pré-existentes, fora da superfície da feature, e
  justificados pelo próprio código:
  | Teste | Motivo (no código) |
  | :--- | :--- |
  | `TestAuditSubscriber_ReadOnlyFilesystem_FailsClosed` | `internal/event_runtime/audit_subscriber_test.go:108` — *"skipping read-only FS test on Windows (chmod semantics differ)"* |
  | `TestBuscaSQLitePopulaMTimeEmTodosOsCaminhos/knn` | `internal/watcher/mtime_paths_test.go:142` — *"sqlite-vec indisponível nesta plataforma (o guard de HasSqliteVec impede a query); esta query só é coberta em build cgo com a tag sqlite_fts5"* |
  | `TestVaultRecuperacaoPorSentido` | `cmd/mem/vault_retrieval_test.go:78` — o provider embutido (modelo + ONNX Runtime) não está disponível neste ambiente; o teste pula com motivo explícito |
  | `TestVaultReindexDoZeroPreservaOsVetores` | `cmd/mem/vault_retrieval_test.go:151` — idem (mesmo `providerDoVaultDeTeste`) |
  | `TestBuscaComProviderRealDeclaraFonteVector` | `cmd/mem/search_vector_source_test.go:36` — *"provider embutido indisponível neste ambiente — o teste precisa do modelo e da biblioteca do ONNX Runtime"* |
- **Failures**: nenhum.
- **`go build ./...`**: exit 0 · **`go vet ./...`**: exit 0
- **`gofmt -l`**: limpo nos arquivos tocados (medido em cópia LF; o worktree
  local mistura CRLF e produz 142 falsos positivos). A CI roda `gofmt -l .` em
  checkout Linux — o `internal/store/store.go` estava **quebrado** desde o T3 e
  foi corrigido na T9 (`1af7b4a`).

---

## Medição real (independente, Postgres do cofre central)

| Repositório | Score | Leitura |
| :--- | :---: | :--- |
| `repo_central` | **45/100** | **125** links na categoria "fora do índice por decisão" (112 por basename + 13 por `title:`) e **13** rompidos (conceitos sem nota). O `--fix` apaga **13** arestas, não 138. |
| `repo_e4c8f3b1a2d5` | **75/100** | Os 2 dead links reais (`[[ADR-035]]`) continuam mortos (100 − 10 − 15 dos chunks sem vetor real, que a spec 2 tornou visíveis). |

A medição foi feita de forma independente pelo Verifier (rodada 1 e rodada 2) e
confere com a spec emendada. **O score do cofre não sobe com esta spec**: o
teto de 40 da penalidade de dead link já estava saturado (13 × 5 = 65 > 40) —
o que muda é o **relatório** (13 em vez de 138) e o **`--fix`** (13 em vez de
138). O critério **≥ 90** da ADR-062 §B-2 **não fecha** sem uma decisão de
**conteúdo** sobre os 13 conceitos sem nota (fora do escopo da spec).

---

## Code Quality

| Principle | Status |
| --------- | ------ |
| Minimum code | ✅ — o classificador é compartilhado (`store.ClassifyLinks`), sem duplicação por engine |
| Surgical changes | ✅ — a fase 4 toca 4 arquivos, todos ligados aos gaps abertos |
| No scope creep | ✅ — nenhum schema novo, nenhum dado gravado diferente, nenhuma mudança na matemática da ADR-013 |
| Matches patterns | ✅ — `vaultfs` não importa `store` (sem ciclo); `store`/`db` importam `vaultfs`; `config`/`parser` só usam stdlib |
| Spec-anchored outcome check | ✅ — 9/9 ACs com `file:line` e valor conferido |
| Per-layer coverage expectation | ✅ — domínio 1:1 com os ACs; CLI/MCP com caminho feliz + aresta + erro |
| Every test maps to a spec AC | ✅ — nenhum teste órfão na superfície da feature |
| Documented guidelines followed | ✅ — `AGENTS.md` (regras 4 e 8), `.agents/rules/always-test.md`, `.agents/rules/always-quality-gate.md` |
| Shared classifier, single source of truth | ✅ — relatório e `--fix` consomem a **mesma** lista (`store.ClassifyLinks`); verificado em `internal/store/doctor_links.go:96` |

---

## Edge Cases

- [x] Raiz existe mas está vazia → todo link é rompido (sem achado no disco, sem otimismo): `internal/db/doctor_unindexed_test.go:109`, `internal/store/doctor_unindexed_test.go:104`
- [x] Raiz não existe → **não verificável**, nunca "rompido": `internal/vaultfs/root_test.go:131`, `internal/store/doctor_score_test.go:139`
- [x] Dois arquivos com o mesmo basename → primeiro match, e o relatório nomeia o path resolvido: `internal/vaultfs/target_test.go:188`
- [x] Alvo com prefixo de pasta → caminho relativo primeiro, basename depois: `internal/vaultfs/target_test.go:70`
- [x] Alvo casa com `title:` → resolvido: `internal/vaultfs/target_test.go:107`
- [x] Alvo casa com `alias:` → resolvido: `internal/vaultfs/target_test.go:125`
- [x] Match exato, nunca fuzzy: `internal/vaultfs/target_test.go:145` — `TestVaultTargetNaoFazMatchFuzzy`
- [x] Tag viva (`tagged_as`) nunca é reclassificada pela checagem em disco: `internal/store/doctor_tag_edges_test.go:16`, `internal/store/doctor_code_nodes_test.go:16`
- [x] Vault em outro host que o Postgres → raiz ilegível → não verificável: `internal/vaultfs/root_test.go:146` (raiz que não é diretório)
- [x] A raiz do vault resolvida é **repo-aware** (o catálogo do repo auditado vence o `central_vault.path`): `internal/vaultfs/root_test.go:92`, `internal/vaultfs/root_test.go:111`
- [x] A rotina de correção é read-only no diagnóstico (a raiz não é escrita): `internal/vaultfs/root_test.go:164`

---

## Fix Plans (achados da rodada 1 — todos fechados)

### Fix 1: o gate completo estava vermelho

- **Root cause**: `internal/mcp/structured_content.go` normalizava só 4 arrays do `DoctorReport`; as duas listas novas (`out_of_index_links`, `unverifiable_links`) serializavam como `null`. O `structuredContent` proíbe `null` — e o guarda-cláusula que já existia em `handlers_test.go` falhou. A base `b8350fd` passava e o HEAD falhava: regressão introduzida pela feature.
- **Fix task**: T8 — `74e8ec7`. As duas listas foram normalizadas; o guarda-cláusula do `health_score` passou a conferir a chave pelo nome (a substring `health_score` casava com `health_score_provisional`).
- **Priority**: Blocker
- **Status**: fechado — gate full e taggeado verdes; mutações M5 e M6 mortas.

### Fix 2: `gofmt` da CI quebrado

- **Root cause**: os campos novos entraram com comentários no meio do `DoctorReport` e deixaram o bloco com o alinhamento antigo.
- **Fix task**: T9 — `1af7b4a`.
- **Priority**: Major
- **Status**: fechado.

### Fix 3: gap de precisão na spec

- **Root cause**: o Out of Scope afirmava 90/100 para o vault de repositório; o real é 75/100 desde a spec `fix-zero-vector-chunks` (que tornou visível a degradação de vetor zero).
- **Fix task**: emenda da spec — `b6cf2db`.
- **Priority**: Minor
- **Status**: fechado.

---

## Requirement Traceability Update

| Requirement | Previous Status | New Status |
| ----------- | --------------- | ---------- |
| DUL-01 | Implementing | ✅ Verified (`internal/vaultfs/target.go:139`, `internal/db/doctor_unindexed_test.go:84`) |
| DUL-02 | Implementing | ✅ Verified (`internal/db/doctor_unindexed_test.go:119`) |
| DUL-03 | Implementing | ✅ Verified (`internal/store/store.go:180`, `internal/db/doctor_unindexed_test.go:99`) |
| DUL-04 | Implementing | ✅ Verified (`internal/store/doctor.go:45`, `internal/store/doctor_score_test.go:26`) |
| DUL-05 | Implementing | ✅ Verified (`internal/vaultfs/root.go`, `internal/store/doctor_score_test.go:75`) |
| DUL-06 | Implementing | ✅ Verified (`internal/db/doctor_unindexed_test.go:136`, `internal/store/doctor_unindexed_test.go:128`) |
| DUL-07 | Implementing | ✅ Verified (`internal/db/doctor_unindexed_test.go:171`) |
| DUL-08 | Implementing | ✅ Verified (`internal/mcp/handlers.go:180`, `internal/mcp/structured_content.go:65`) |
| DUL-09 | Implementing | ✅ Verified (`cmd/mem/main.go`, `cmd/mem/doctor_vault_test.go:73`) |

**Coverage**: 9 total, 9 verificados, 0 unmapped.

---

## Lessons distilled

O sinal desta rodada (gate falho na rodada 1 + gap de precisão da spec) virou
duas lições candidatas via `lessons.py add`:

| Lição | Signal | Source |
| :--- | :--- | :--- |
| L-007 — toda chave array nova no payload de uma tool MCP precisa entrar na normalização da fronteira do `structuredContent` e no teste que proíbe `null`: o contrato do wire é **por chave**, não por tipo | `gate_fail` | `internal/mcp/structured_content.go:65` |
| L-008 — número de Health Score citado em Out of Scope envelhece quando outra spec mexe na mesma fórmula: remeça antes de reutilizar o valor medido por outra spec | `spec_precision_gap` | `.specs/features/fix-doctor-unindexed-links/spec.md:68` |

---

## Summary

**Overall**: ✅ Pronto — com a ressalva de independência declarada no topo.

**Spec-anchored check**: 9/9 ACs com `file:line`, 0 gap de precisão
**Sensor**: 6/6 mutações mortas
**Gate**: 1325 testes (full: 1321 ok / 4 skips justificados / 0 falhas) ·
gate taggeado com ONNX + Postgres: **1337 ok / 5 skips justificados / 0 falhas**,
27/27 pacotes `ok`

**O que funciona**: a distinção de três categorias existe nos **dois** engines,
no relatório da CLI e no payload do MCP; a rotina `--fix` consome a mesma lista
do relatório; a checagem em disco é honesta (match exato) e degrada para "não
verificável" em vez de inventar "rompido"; o `--fix` deixou de apagar 138
arestas para apagar 13.

**Achados da rodada 1 (fechados)**: o gate completo estava vermelho desde o
T7 (as listas novas serializavam como `null` no `structuredContent`), o `gofmt`
da CI estava quebrado desde o T3, e o Out of Scope da spec tinha um número
stale.

**Pendências que NÃO são desta spec**: os 13 conceitos sem nota (decisão de
conteúdo do dono) e o `mem index --force` com o binário taggeado, que tira a
penalidade de 15 dos chunks sem vetor e leva o cofre a 60/100.
