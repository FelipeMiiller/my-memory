# central-vault-concept-notes Validation

**Date**: 2026-11-14
**Spec**: `.specs/features/central-vault-concept-notes/spec.md`
**Diff range**: `df1ecc3..HEAD` (artefatos do repo) · **vault**: `G:\My Drive\central-memory`
**Verifier**: execução **inline** (6 tarefas, ≤ ~8 por lote — sem sub-agente, conforme a regra do lote único)

## Validation: PASS

**Result**: PASS

---

## Task Completion

| Task | Status | Notes |
| ---- | ------ | ----- |
| T1 — as 6 notas de `cloud/` | ✅ Done | `Docker.md`, `Kubernetes.md`, `Observabilidade.md`, `Rollback.md`, `Expand and Contract.md`, `SRE e SLOs.md` |
| T2 — as 7 notas de `database/` | ✅ Done | `Banco de Dados.md`, `Índices SQL.md`, `Busca Vetorial e RAG.md`, `Filas e Processamento Assíncrono.md`, `Transações e Concorrência.md`, `Observabilidade de Banco de Dados.md`, `Backup e Recuperação.md` |
| T3 — os READMEs | ✅ Done | `cloud/README.md` atualizado (13 entradas — e a nota blue-green, que **não** estava listada) · `database/README.md` **criado** (a pasta tinha conteúdo e nenhum README) |
| T4 — o reindex com o provider | ✅ Done | `mem index -dir <vault> -storage postgres -repo repo_central -force` com o binário `-tags "sqlite_fts5 onnx"`: **108 documentos** (94 + 13 + 1), 108 indexados, 0 podados, 0 falhos |
| T5 — a medição | ✅ Done | **Health Score 100/100** · **0 link rompido** · **0 órfã** · **0 chunk sem vetor real** (eram 101) |
| T6 — o fechamento | ✅ Done | ADR-064 · ADR-065 · `STATE.md` · `ISSUES.md` · este relatório |

**Cobertura**: 6 tarefas, 6 concluídas, 0 pendentes.

---

## Spec-Anchored Acceptance Criteria

| Criterion (WHEN X THEN Y) | Spec-defined outcome | `file:line` + assertion | Result |
| ------------------------- | -------------------- | ----------------------- | ------ |
| **AC1** — os 13 wikilinks resolvem para documento indexado | as 13 arestas `links_to` saem da lista de suspects | `internal/store/doctor.go:96` — a query de suspects exclui `target_id IN (SELECT title FROM documents)`; medido: **0 link rompido** e as 13 arestas com `target_id` = o nome do conceito | ✅ PASS |
| **AC2** — 0 link rompido no `repo_central` | **0** | `mem doctor --storage=postgres --repo repo_central` → `[🔗 Links Quebrados / Dead Links] (0)` — "Nenhum link quebrado detectado" | ✅ PASS |
| **AC3** — nome do arquivo e `title` iguais ao wikilink | `Docker.md` com `title: "Docker"`, e assim nas 13 | `cloud/Docker.md:2` · `database/Backup e Recuperação.md:2` — verificado por `scripts/verify_vault_notes.py` (campo `title` == conceito, arquivo == conceito) | ✅ PASS |
| **AC4** — frontmatter canônico e `# Título` no corpo | `title`, `category: resource`, `summary`, `tags`, `status: ativo` + H1 igual ao title | `cloud/Kubernetes.md:2` · `database/Transações e Concorrência.md:2` — `scripts/verify_vault_notes.py` (frontmatter completo + `H1 == title`) | ✅ PASS |
| **AC5** — backlink para o README da pasta e para a nota de origem | `[[<pasta>/README]]` e `[[<nota que linka>]]` | `cloud/SRE e SLOs.md:41` (`[[cloud/README]]`) · `database/Busca Vetorial e RAG.md:42` (`[[PostgreSQL — Visão Geral de Recursos e Ferramentas]]`) — verificado pelo script | ✅ PASS |
| **AC6** — entrada no README da pasta; `database/` ganha README | as 13 notas listadas; `database/README.md` existe | `cloud/README.md:18` — as 13 entradas em "## Notas"; `database/README.md:17` — a seção "## Notas" com as 8 notas da pasta | ✅ PASS |
| **AC7** — 0 chunk sem vetor real depois do reindex | **0** | `mem doctor` → a seção de chunks sem vetor real **não aparece** (eram 101, com o teto de 15 pontos); reindex log: 108 documentos, 0 falhas | ✅ PASS |
| **AC8** — Health Score ≥ 90 | **≥ 90** | `mem doctor` → `🟢 RELATÓRIO DE SAÚDE DA MEMÓRIA & GRAFO (Health Score: 100/100)` — o critério §B-2 da ADR-062 **fecha** | ✅ PASS |
| **AC9** — a nota diz o que é e onde entra | seções "O que é" e "Onde entra aqui" | `cloud/Rollback.md:14` ("## O que é") · `cloud/Rollback.md:27` ("## Onde entra aqui") — verificado pelo script nas 13 | ✅ PASS |
| **AC10** — ao menos uma relacionada por nota | seção "Relacionado" com wikilinks existentes | `database/Índices SQL.md:40` — "## Relacionado" com 4 links, todos existentes no vault (o script confere) | ✅ PASS |
| **AC11** — nenhuma nota nova órfã | 0 órfãs | `mem doctor` → `[🏝️ Notas Órfãs / Sem Conexões] (0)` — "Nenhuma nota isolada" | ✅ PASS |

**Status**: ✅ 11/11 ACs cobertas com evidência, 0 gap de precisão.

---

## Discrimination Sensor

O sensor de uma spec de **conteúdo** é o próprio verificador: a prova é que
ele **mata** a primeira versão das notas. Rodado em duas rodadas, ele encontrou
**24 desvios reais** que a leitura não tinha pego:

| Rodada | Defeito detectado | Onde | Killed? |
| ------ | ----------------- | ---- | ------- |
| 1 | 13 notas sem a seção "O que é" (o AC9 estava escrito, o corpo não) | as 13 notas | ✅ Killed |
| 1 | 7 backlinks para `[[database/README]]` apontando para um README que **não existia** | as 7 notas de `database/` | ✅ Killed |
| 1 | 1 caractere CJK espúrio (`尽力`) | `database/Filas e Processamento Assíncrono.md:23` | ✅ Killed |
| 2 | `[[expand-and-contract]]` — link em kebab-case para uma nota cujo nome tem espaço | `cloud/Rollback.md:33` | ✅ Killed |
| 2 | `[[observabilidade-de-banco-de-dados]]` — kebab-case de novo | `database/Busca Vetorial e RAG.md:33` | ✅ Killed |
| 2 | `[[pgvector]]` — link para uma nota **que não existe** no cofre | `database/Busca Vetorial e RAG.md:39` | ✅ Killed |
| 2 | 2 typos de escrita (`tornão`, `doundo`/`etzel`) | `database/Índices SQL.md:23`, `database/Transações e Concorrência.md:18` | ✅ Killed |

O verificador também foi **amploado** na rodada 2: ele só checava a seção
"Relacionado", e passou a checar **todo** wikilink do corpo — foi assim que
apareceram os 3 links quebrados da rodada 2.

**Sensor depth**: lightweight (conteúdo — 7 classes de defeito, 24 ocorrências).
**Resultado**: **7/7 classes mortas** — PASS ✅

O verificador mora no repo: `scripts/verify_vault_notes.py` (roda contra a raiz
do vault e sai com 1 se qualquer nota desviar).

---

## Gate Check

- **Gate (conteúdo)**: `python scripts/verify_vault_notes.py` → **13/13 OK · 0 erros · 0 avisos** · exit 0
- **Gate (medido)**: `mem index -dir "G:\My Drive\central-memory" -storage postgres -repo repo_central -force` → **108 documentos, 108 indexados, 0 em cache, 0 podados, 0 falhos** · exit 0
- **Gate (medido)**: `mem doctor --storage=postgres --repo repo_central` → **Health Score 100/100** · 0 link rompido · 0 não verificável · 0 órfã · 0 chunk sem vetor real · 45 links fora do índice por decisão
- **Spec**: `validate_spec.py` → **0 erros, 0 warnings** · `validate_tasks.py` → **0 erros**, 6 warnings (5 de "`Tests: none` — a matriz declara a verificação como o script — e 1 de granularity smell no T6, que é uma tarefa de documentação por natureza)
- **State**: `validate_state.py central-vault-concept-notes` → 0 erros (rodado com este relatório em disco)

### O que mudou no número de "fora do índice" (125 → 45)

O reindex reconstruiu as arestas, e o resolver do indexador passou a casar
com **documentos indexados** em vez de nomes de arquivo do catálogo bruto.
Comparando as arestas das 2 notas de origem antes e depois:

| Antes | Depois | Efeito |
| :--- | :--- | :--- |
| `deployment-patterns` (tag do catálogo bruto) | `Blue-Green Deployment — Deploy sem indisponibilidade` (título indexado) | deixou de ser suspect |
| `frontend-backend-strategies` (catálogo bruto) | `Shopify: Versão de API é Decisão de Arquitetura` (título indexado) | deixou de ser suspect |

As 13 arestas que **importavam** (os 13 conceitos) mantiveram o texto do
wikilink e agora resolvem por **match exato** de título — que é o ponto da
spec. As 45 restantes continuam apontando para material do catálogo bruto
(`[[SKILL]]`, `[[ci-cd]]`, `[[checks]]`), que é a categoria correta da
ADR-063: existe em disco, está fora do índice **por decisão**.

> [!note]
> O comportamento do resolver é o mesmo de antes (match por token, ≥ 4
> caracteres): o que mudou é **qual** documento ele casa primeiro, porque a
> lista de documentos disponíveis cresceu de 94 para 108. É a limitação
> conhecida do fuzzy do indexador — e ela **não** afeta os 13 conceitos, que
> casam por match exato.

---

## Code Quality

| Principle | Status |
| --------- | ------ |
| Minimum code | ✅ — 13 notas curtas (1,4 a 2,2 KB), sem boilerplate |
| Surgical changes | ✅ — só as 13 notas e os 2 READMEs; nenhuma outra nota do cofre tocada |
| No scope creep | ✅ — os 10 conceitos que o indexador já resolvia **não** ganharam nota |
| Matches patterns | ✅ — padrão do cofre (frontmatter + `# Título`) e do Obsidian (propriedades padrão, wikilink, caminho da pasta) |
| Spec-anchored outcome check | ✅ — 11/11 ACs com evidência `file:line` |
| Per-layer coverage expectation | ✅ — conteúdo verificado por script (13/13); índice verificado pela medição real |
| Every test maps to a spec AC | ✅ — o script cobre AC3, AC4, AC5, AC6, AC9, AC10, AC11 |
| Documented guidelines followed | ✅ — `AGENTS.md` (regras 1, 4 e 7), `.agents/skills/my-memory-format/SKILL.md` §1–§7, `standards/navegacao-moc.md` (regra do MOC), doc oficial do Obsidian (`help/properties`, `help/links`) |
| Sem invenção | ✅ — nenhum número, limite ou afirmação de produto inventado; o conteúdo é conceito geral e o contexto local |

---

## Edge Cases

- [x] Dois conceitos com o mesmo nome → **medido**: nenhum dos 13 títulos colide
      com os 94 documentos indexados (sondagem no Postgres antes de criar).
- [x] `title` diferente do texto do wikilink → evitado: o `title` é **exatamente**
      o conceito, para o match exato do indexador.
- [x] Nota sem backlink para o README da pasta → detectado pelo verificador (AC5).
- [x] Reindex com o binário **sem** a tag `onnx` → evitado: o reindex usou o
      binário taggeado, e o doctor confirma 0 chunk sem vetor real.
- [x] Nota escrita com BOM → detectado pelo verificador (`BOM`); as 13 estão
      UTF-8 sem BOM, LF.
- [x] Pasta `database/` sem README → a regra 1 do MOC exige um; o README foi
      criado (com o padrão de frontmatter das outras pastas).
- [x] Nota que vira órfã → o doctor reporta 0 órfãs; todas as 13 são
      referenciadas por uma nota do vault.

---

## Fix Plans

Nenhum gap aberto. Os defeitos encontrados durante a execução (seção do sensor)
foram corrigidos **antes** do fechamento, e o verificador foi corrigido
(pronto para reavaliar) quando se mostrou que só olhava uma seção das notas.

---

## Requirement Traceability Update

| Requirement | Previous Status | New Status |
| ----------- | --------------- | ---------- |
| CCN-01 | Implementing | ✅ Verified (`internal/store/doctor.go:96`, medição real) |
| CCN-02 | Implementing | ✅ Verified (`mem doctor` → 0 link rompido) |
| CCN-03 | Implementing | ✅ Verified (`cloud/Docker.md:1` e as outras 12) |
| CCN-04 | Implementing | ✅ Verified (`cloud/Kubernetes.md:1` e as outras 12) |
| CCN-05 | Implementing | ✅ Verified (`cloud/SRE e SLOs.md:41`, `database/Busca Vetorial e RAG.md:42`) |
| CCN-06 | Implementing | ✅ Verified (`cloud/README.md:18`, `database/README.md:1`) |
| CCN-07 | Implementing | ✅ Verified (reindex 108/108, 0 chunk sem vetor real) |
| CCN-08 | Implementing | ✅ Verified (Health Score 100/100) |
| CCN-09 | Implementing | ✅ Verified (`cloud/Rollback.md:14` e `:17`) |
| CCN-10 | Implementing | ✅ Verified (`database/Índices SQL.md:40`) |
| CCN-11 | Implementing | ✅ Verified (0 órfã) |

**Coverage**: 11 total, 11 verificados, 0 unmapped.

---

## Summary

**Overall**: ✅ Pronto

**Spec-anchored check**: 11/11 ACs com `file:line`
**Sensor**: 7 classes de defeito, 24 ocorrências, 7/7 mortas
**Gate**: verificador 13/13 (0 erros) · reindex 108/108 · doctor 100/100

**O que funciona**: os 13 conceitos linkados pelo cofre agora são notas de
verdade — resolvem no Obsidian (padrão do arquivo), resolvem no índice (match
exato de título), estão indexadas no README da sua pasta e sobem para o índice
da pasta. O Health Score do Cofre Central saiu de **45/100 para 100/100**, e o
critério de prontidão **"doctor ≥ 90/100"** da ADR-062 §B-2 **fecha**.

**Achados da execução**: o verificador pegou 24 desvios que a leitura não pegou
(seção faltando, backlink para um README inexistente, link em kebab-case, link
para nota inexistente, CJK, typos) — e ele próprio tinha um gap (só olhava a
seção "Relacionado"), corrigido no meio.

**O que NÃO fecha (e é de outro escopo)**: as **45** arestas que apontam para
o material bruto do catálogo continuam fora do índice — é a categoria correta da
ADR-063, não doença. E o bucket GAP-6 a GAP-10 da ADR-062 segue aberto, com o
caminho de decisão item a item registrado na ADR-065.
