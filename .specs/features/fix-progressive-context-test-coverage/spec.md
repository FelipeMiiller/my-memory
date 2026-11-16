# Feature: fix-progressive-context-test-coverage

- **Spec ID**: `fix-progressive-context-test-coverage`
- **Created**: 2026-11-15
- **Owner**: Felipe Miiller
- **Source**: ISSUE-044 — `.specs/ISSUES.md` (seção ISSUE-044) · fecha os 3 gaps da feature `progressive-context-loading` (ADR-025)
- **Related**: `.specs/features/progressive-context-loading/validation.md` (veredito **PARTIAL**; sensor com 6 mutações, 3 mortas / 3 sobreviveram) · lições L-012 (`cmd/mem`) e L-013 (`internal/mcp`)

## Problem Statement

A feature `progressive-context-loading` (ADR-025) foi implementada, o gate ficou
verde e a verificação independente devolveu veredito **PARTIAL**: **8/9 ACs** com
evidência `file:line` e **3 caminhos sem teste** — PCL-01, PCL-02 e PCL-03
(ISSUE-044). O sensor de discriminação injetou 6 mutações: **3 mortas e 3
sobreviveram** — e as 3 que sobreviveram são exatamente os 3 gaps.

O padrão dos 3 sobreviventes é o **mesmo erro de raciocínio**, não três bugs
diferentes: o comportamento foi testado **na unidade errada**, e o **encano entre
as unidades** ficou sem prova.

| Gap | Onde o teste olha (a unidade) | Onde o comportamento está (o encano) | Mutante |
| :--- | :--- | :--- | :--- |
| **PCL-01** | o **default** da flag — `cmd/mem/search_defaults_test.go:119` e `:139` | a **renderização** da saída L0 — `cmd/mem/main.go:2368` (Postgres) e `cmd/mem/main.go:2517` (SQLite) | **M6** |
| **PCL-02** | o **comportamento** da tool — `internal/mcp/handlers_test.go:1088` | o **schema JSON declarado** — `internal/mcp/tools.go:54` | **M5** |
| **PCL-03** | o **parser** — `internal/parser/parser_test.go:286` | o **indexador** que grava — `internal/watcher/indexer.go:82` | **M2** |

Os 3 gaps são de **cobertura**, não de comportamento: o caminho funcional existe
e o gate passa; o que falta é a prova de que ele continua funcionando. Por isso a
feature `progressive-context-loading` **fica vermelha** (`validate_state.py
progressive-context-loading` sai 1) até os 3 testes existirem.

**Ressalva herdada que esta spec absorve**: o sensor desta feature foi executado
**no pai** — o Verifier independente encerrou o turno antes dele. A
**re-verificação independente** da feature continua pendente. Por isso esta spec
não fecha só com "os 3 testes existem": ela fecha com o **ciclo fix → re-verificar**
(Verifier independente + sensor de discriminação), e uma das ACs exige a **prova
de que os 3 testes novos matam os 3 mutantes que sobreviveram** (M2, M5, M6).

## Goals

- [ ] Cada um dos 3 gaps da ISSUE-044 (PCL-01, PCL-02, PCL-03) ganha um teste
      **no encano**, e não na unidade vizinha.
- [ ] Os 3 testes novos **matam** os 3 mutantes que sobreviveram (M2, M5, M6),
      provado pelo sensor de discriminação.
- [ ] A feature `progressive-context-loading` passa de **8/9** para **9/9** ACs
      com evidência `file:line`, e `validate_state.py progressive-context-loading`
      sai **exit 0**.
- [ ] O sensor de discriminação passa de **3/6 mortas** para **6/6 mortas** nas
      6 mutações originalmente injetadas.
- [ ] **Nenhum arquivo de produção muda**: o diff toca só arquivos `_test.go` e
      os artefatos de `.specs/`.

## Out of Scope

| Item | Reason |
| :--- | :--- |
| Alterar o comportamento de produção (renderização, schema, indexador) | O veredito é PARTIAL por **falta de prova**, não por bug — a ISSUE-044 diz que "o comportamento existe e o gate está verde". A correção é de **cobertura**. |
| Reescrever o relatório antigo `.specs/features/progressive-context-loading/validation.md` sem refazer a verificação | A ressalva de independência não se conserta por edição: ela se conserta **refazendo** a verificação com Verifier independente (T5). |
| Mexer nas outras 3 mutações (M1, M3, M4) | Já são mortas (`TestFrontmatter_CategoryAndAbstract`, `TestSearch_CategoryFilter`, `TestFuseSearchResults_LevelL0`). Refazê-las é re-verificação, não correção. |
| Mudar a feature `progressive-context-loading` (spec, código, `docs/`) | A feature é a **origem** dos gaps; esta spec só fecha os gaps que ela deixou. |
| Fechar a ISSUE-045 (`mem pack` estourando o orçamento de tokens) | Issue separada, com severidade e escopo próprios. |
| Reindexar, reescrever ou tocar em vault | Nenhum dado gravado muda — a correção é só de teste. |
| Criar ADR | Nenhuma decisão de arquitetura: ADR-025 (taxonomia `category`) e a forma do `abstract` continuam valendo. |

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| Onde a asserção da renderização L0 mora | **nos dois engines**: o ramo SQLite (`cmd/mem/main.go:2517`) e o ramo Postgres (`cmd/mem/main.go:2368`) | o mutante **M6** do sensor aponta para o ramo **Postgres** (`cmd/mem/main.go:2368`); um teste só no SQLite **não mataria M6**, e a AC do sensor exigiria uma prova que não existiria. Os dois ramos são cópias da mesma renderização, então os dois precisam de prova. | y — **decidido**: os dois ramos são obrigatórios |
| Como o catálogo da tool é lido no teste | pelo **`GetTools()`** do servidor MCP (o catálogo **servido**), não pela variável de pacote `ToolMemorySearch` | o gap é o **schema declarado**; ler o catálogo servido prova o que o cliente MCP recebe. `GetTools()` é a superfície pública (`internal/mcp/tools.go:792`) e é o que o Fix Plan da ISSUE-044 prescreve. | y |
| Como o teste da CLI monta o vault | **banco temporário por teste** (`t.TempDir()`), com um documento cujo frontmatter declara `category` e `summary` | é o padrão do repo (`internal/watcher/indexer_test.go:14`, `cmd/mem/codeindex_postgres_test.go:56`); o teste não escreve no vault do dono nem no Postgres de produção. | y |
| Onde o teste do Postgres roda | contra o **banco descartável** do `internal/pgtest`, com `MY_MEMORY_PG_TEST=1` e `pgtest.Capture()` no `init()` | o `pgtest` existe exatamente para isso (trava + banco descartável) e é o **único** caminho permitido para o Postgres em teste (`internal/pgtest/pgtest.go:1-23`). | y |
| Se o gate do Postgres é obrigatório | **sim** — o gate completo roda com `MY_MEMORY_PG_TEST=1`; sem a variável o teste Postgres **pulsa** (skip) e o M6 sobrevive sem que o gate avise | a CI roda `go test ./cmd/mem/...` **sem** a variável (o teste pula). A prova do M6 depende do gate local com Postgres e, em cima dele, do sensor do T5. | y |
| Como o sensor é executado no ciclo fix → re-verificar | **worktree isolado** (`git worktree add`), **sem `git stash`**, com a árvore real intocada e o baseline conferido antes e depois | é o padrão do sensor anterior (`.specs/features/progressive-context-loading/validation.md` §Discrimination Sensor) e o que o `tlc-spec-driven` exige do sensor. | y |
| O que o relatório final da feature corrigida deve dizer | **PASS**, com 9/9 ACs, evidência `file:line` e sensor 6/6 | o `validate_state.py` só dá verde com PASS; um PARTIAL novo reabriria a ISSUE-044 e a feature continuaria vermelha. | y |
| A ordem de execução dos 3 testes | schema (T1) → CLI SQLite (T2) → CLI Postgres (T3) → indexador (T4) | as 4 tasks são **independentes** (pacotes diferentes, nenhum arquivo compartilhado); a ordem é só a sequência do plano, dentro da mesma fase. | y |

Open questions: none — todas as decisões acima têm default e rationale.

## User Stories

### P1: O schema da tool MCP é asserido no lugar certo ⭐ MVP

**User Story**: Como cliente MCP da `memory_search`, quero que o **schema declarado**
da tool seja verificado, para que uma chave removida do schema não passe mais
despercebida.

**Why P1**: é o gap PCL-02 — o mutante M5 (remover `detail_level` do schema,
`internal/mcp/tools.go:54`) sobreviveu porque o teste existente (`internal/mcp/handlers_test.go:1088`)
verifica o **comportamento**, que continua idêntico com a chave fora do schema.

**Acceptance Criteria**:
1. WHEN the MCP server exposes its tool catalog THEN the `memory_search` input schema SHALL declare a `detail_level` property whose `enum` contains exactly `l0`, `l1` and `l2`.
2. WHEN the MCP server exposes its tool catalog THEN the `memory_search` input schema SHALL declare a `category` property whose `enum` contains exactly `resource`, `memory` and `skill`.

**Independent Test**: remove a chave do schema em `internal/mcp/tools.go` e o
teste falha com o nome da chave ausente.

---

### P1: A saída L0 da CLI é asserida no lugar certo ⭐ MVP

**User Story**: Como usuário de terminal, quero que a **saída** do `mem search
--level l0` seja verificada, para que a listagem ultracompacta (badge de
categoria + micro-abstract, sem o corpo bruto) não encolha sem ninguém perceber.

**Why P1**: é o gap PCL-01 — o mutante M6 (desligar o ramo L0 da renderização,
`cmd/mem/main.go:2368`) sobreviveu porque o teste existente (`cmd/mem/search_defaults_test.go:119`
e `:139`) verifica o **default da flag**, que não muda quando a renderização morre.

**Acceptance Criteria**:
3. WHEN `mem search --level l0` returns at least one result on SQLite THEN the printed output SHALL contain the uppercase category badge and the micro-abstract of each result and SHALL NOT contain the raw chunk body.
4. WHEN `mem search --level l0` returns at least one result on PostgreSQL THEN the printed output SHALL contain the uppercase category badge and the micro-abstract of each result and SHALL NOT contain the raw chunk body.

**Independent Test**: um vault de teste com uma nota `category: memory` e
`summary:` → a saída L0 mostra `[MEMORY]` e o summary, e o corpo do chunk não
aparece.

---

### P2: O indexador é asserido no lugar certo

**User Story**: Como motor de indexação, quero que a **escolha** do `abstract`
gravado seja verificada, para que o `summary` do frontmatter chegue ao índice
como `abstract` e não como o fallback truncado.

**Why P2**: é o gap PCL-03 — o mutante M2 (o indexador deixar de usar o
`summary` como `abstract`, `internal/watcher/indexer.go:82`) sobreviveu porque o
teste existente (`internal/parser/parser_test.go:286`) verifica o **parser**, que
continua correto com o indexador quebrado.

**Acceptance Criteria**:
5. WHEN the indexer stores a document whose frontmatter declares a `summary` THEN the `documents.abstract` column SHALL hold the `summary` value verbatim.
6. IF the frontmatter declares no `summary` but declares an `abstract` THEN the indexer SHALL store the frontmatter `abstract` in the `documents.abstract` column.
7. IF the frontmatter declares neither `summary` nor `abstract` THEN the indexer SHALL store the parser fallback truncated to at most 160 characters in the `documents.abstract` column.

**Independent Test**: uma nota com `summary:` no frontmatter → a coluna
`documents.abstract` do banco contém **exatamente** o summary.

---

### P1: A correção é re-verificada de forma independente ⭐ MVP

**User Story**: Como dono do repo, quero que a correção destes gaps seja
**re-verificada por um Verifier independente**, com sensor de discriminação, para
que a feature saia do vermelho com prova — e não com a repetição do mesmo
procedimento que a deixou vermelha.

**Why P1**: a ressalva herdada: o sensor da feature original rodou **no pai**,
porque o Verifier encerrou o turno antes dele. Sem um ciclo fix → re-verificar
com Verifier independente, a feature continua com veredito PARTIAL e a
`ISSUE-044` continua aberta.

**Acceptance Criteria**:
8. WHEN the discrimination sensor re-injects the three surviving mutants M2, M5 and M6 THEN the test suite SHALL fail for each one of them.
9. The verification of the `progressive-context-loading` feature SHALL be produced by a Verifier distinct from the author of the three new tests.

**Independent Test**: o relatório da feature corrigida cita `file:line` para as
9 ACs e o resultado do sensor **6/6 mortas**.

## Edge Cases

- IF a busca devolve **zero** resultados THEN a CLI imprime "Nenhum resultado
  encontrado." e nenhuma linha L0 — o fixture do teste sempre tem ao menos 1
  resultado, para que a asserção nunca seja vazia.
- IF o mesmo fixture é buscado no nível `l1` THEN o corpo bruto **aparece** na
  saída — é o **controle negativo** que prova que a asserção do L0 não é
  vacuamente verde.
- IF o frontmatter não tem `summary` nem `abstract` THEN o `abstract` gravado é
  o fallback do parser, truncado em 160 caracteres (AC7).
- IF `MY_MEMORY_PG_TEST` não está no ambiente THEN o teste Postgres **pulsa**
  (skip) e não prova nada — o M6 só morre no gate com Postgres e no sensor do T5.
- IF o documento tem `category` vazia THEN o badge sai como `RESOURCE`
  (default do motor) — o fixture declara `category` explicitamente para que a
  asserção do badge seja sobre o valor gravado, não sobre o default.

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Gap | Mutante | Where | Status |
| :--- | :--- | :--- | :--- | :--- | :--- |
| PCT-01 | AC1 — `detail_level` no schema com enum l0/l1/l2 | PCL-02 | M5 | `internal/mcp/tools_test.go` | Pending |
| PCT-02 | AC2 — `category` no schema com enum resource/memory/skill | PCL-02 | M5 | `internal/mcp/tools_test.go` | Pending |
| PCT-03 | AC3 — saída L0 no SQLite: badge + abstract, sem corpo bruto | PCL-01 | M6 | `cmd/mem/search_level_l0_test.go` | Pending |
| PCT-04 | AC4 — saída L0 no Postgres: badge + abstract, sem corpo bruto | PCL-01 | M6 | `cmd/mem/search_level_l0_postgres_test.go` | Pending |
| PCT-05 | AC5 — o `summary` vira `abstract` verbatim | PCL-03 | M2 | `internal/watcher/indexer_abstract_test.go` | Pending |
| PCT-06 | AC6 — o `abstract` do frontmatter vira `abstract` | PCL-03 | M2 | `internal/watcher/indexer_abstract_test.go` | Pending |
| PCT-07 | AC7 — sem `summary`/`abstract`: fallback de 160 caracteres | PCL-03 | M2 | `internal/watcher/indexer_abstract_test.go` | Pending |
| PCT-08 | AC8 — o sensor mata M2, M5 e M6 | ressalva herdada | M2, M5, M6 | `.specs/features/progressive-context-loading/validation.md` | Pending |
| PCT-09 | AC9 — o Verifier é distinto do autor dos 3 testes | ressalva herdada | — | `.specs/features/progressive-context-loading/validation.md` | Pending |

**Coverage**: 9 total, 9 mapeados, 0 unmapped.

## Success Criteria

- [ ] Os 3 gaps da ISSUE-044 (PCL-01, PCL-02, PCL-03) têm teste com evidência
      `file:line` no relatório do Verifier.
- [ ] A feature `progressive-context-loading` vai de **8/9** ACs com evidência
      para **9/9**, e `validate_state.py progressive-context-loading` sai
      **exit 0**.
- [ ] O sensor de discriminação passa de **3/6 mortas** para **6/6 mortas** — as
      3 novas (M2, M5, M6) morrem e as 3 antigas (M1, M3, M4) continuam mortas.
- [ ] O gate completo (`go -C D:\repository\my-memory test -count=1 ./...`, com
      Postgres) e o gate taggeado (`go -C D:\repository\my-memory test -tags
      "sqlite_fts5 onnx" -count=1 ./...`) saem **exit 0**, com `go build` e
      `go vet` verdes.
- [ ] **Nenhum arquivo de produção muda** — o diff da correção toca só arquivos
      `_test.go` e os artefatos de `.specs/`.
