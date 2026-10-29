# Feature: doctor-empty-vault-score

- **Spec ID**: `doctor-empty-vault-score`
- **Created**: 2026-10-28
- **Owner**: Felipe Miiller
- **Source**: ISSUE-019 — `.specs/ISSUES.md:612`
- **Related**: ADR-013 (`docs/adr/013-higiene-de-grafo-pruning-e-doctor.md`) · ISSUE-006 (calibração de pesos — **outro** defeito, não corrigido aqui)

## Problem Statement

`mem doctor` reporta **Health Score: 100/100** num vault que nunca foi indexado. A métrica deixa de distinguir "vault saudável" de "vault inexistente" — que é justamente o uso principal dela (decidir se vale reindexar).

Evidência reproduzida em 2026-10-28 com o binário real compilado do estado atual do repositório:

```
$ mem doctor --storage=sqlite --db <arquivo-novo-vazio.db> --repo acme
 🟢 RELATÓRIO DE SAÚDE DA MEMÓRIA & GRAFO (Health Score: 100/100)
 Documentos: 0 | Chunks: 0 | Arestas: 0 | Nós: 0
```

**Causa raiz.** O cálculo do Health Score é todo guardado por `if report.TotalDocuments > 0`. Num vault vazio a guarda pula o cálculo inteiro e o score fica no valor inicial `100` — o número máximo. A guarda está duplicada nos dois motores:

- SQLite: `internal/db/graph.go:352-388` (`score := 100` … `if report.TotalDocuments > 0 { … }`)
- PostgreSQL: `internal/store/doctor.go:114-146` (mesma guarda, mesma cópia da matemática de penalidades)

O defeito é a guarda, **não** os pesos: as penalidades (dead links, órfãs, self-loops, chunks dessincronizados) estão corretas e continuam válidas para um vault com documentos.

## Goals

- [ ] Fazer o Health Score deixar de reportar um número quando a métrica não tem denominador (vault sem documentos indexados).
- [ ] Preservar integralmente o comportamento de penalidades para vault com documentos (ADR-013).
- [ ] Manter o JSON do `structuredContent` do `memory_doctor` **byte-idêntico** para vault com documentos (contrato público).
- [ ] Entregar teste de regressão que **falha contra o defeito** (100/100 em vault vazio) e passa com a correção.

## Out of Scope

- **ISSUE-006** — Health Score persistentemente *baixo* num vault indexado. É calibração de pesos, defeito oposto e independente. Não corrigir aqui.
- Reweigh das penalidades (5/40 dead links, 40/30 órfãs, 5/15 self-loops, 2/15 chunks dessync) — a ADR-013 continua valendo como está.
- Reindexação do vault do dono (o segundo defeito descrito na ISSUE-019 — "o vault deste repositório está efetivamente não indexado" — é operação, não código).
- Criação de ADR nova: a ADR-013 é `Accepted` e a leitura de "Health Score de 0 a 100 com penalidades graduadas" continua válida; a correção apenas impede que a métrica seja aplicada sem denominador. Registro em `docs/adr/README.md` fica a cargo do gestor (ADRs são imutáveis).
- Atualização de `docs/`, `README.md`, `CHANGELOG.md`, `.specs/ISSUES.md` e `.specs/STATE.md` — fora da propriedade desta execução.

## Decisão de Design — alternativa (A): score nulo + estado explícito

As três saídas defensáveis eram: **(A)** score nulo/ausente com estado explícito, **(B)** score 0 com rótulo de "vazio", **(C)** score 100 com aviso.

**Escolhida: (A).**

Por quê:

1. **Só (A) não deixa número nenhum no payload.** Num vault vazio não há nada a penalizar: o Health Score é uma razão sobre a população de documentos indexados. Sem denominador, a métrica não tem valor — nem 0 (que significa "saúde catastrófica" e acusaria um vault que está intacto) nem 100 (que significa "saúde perfeita" e é exatamente o defeito). (B) troca uma mentira por outra; (C) mantém a mentira original e só acrescenta um aviso que o número continua contradizendo.
2. **A distinção vira consultável por máquina.** Com `health_score` ausente do payload, um agente que só fala MCP distingue "não indexado" de "saudável" sem ler prosa. Com (B) ou (C) ele precisa parsear o texto — e o payload JSON continua affirmando 0 ou 100.
3. **O JSON dos vaults saudáveis não muda.** A chave só some quando a métrica não se aplica; para qualquer vault com documentos o payload é byte-idêntico ao de hoje. A compatibilidade do contrato público é preservada no caminho que importa.
4. **O texto passa a dizer o que aconteceu**, incluindo a ação: "rode `mem index`". Um relatório que só diz "não se aplica" deixa o usuário sem próximo passo.

**O que muda no contrato público:** o campo `health_score` do `structuredContent` da ferramenta MCP `memory_doctor` passa a **viajar ausente** quando o vault não tem documentos indexados — a chave não existe no JSON. O tipo Go correspondente (`mcp.DoctorReport.HealthScore`) vira `*int` — `nil` significa "não se aplica" — e a tag é `json:"health_score,omitempty"`. Nenhum outro campo, nem o Markdown, muda para vaults com documentos.

**Por que ausente e não `null` — é uma constraint do repositório, não uma preferência.** O contrato do wire deste repositório proíbe `null` no `structuredContent`: `TestStructuredContentNeverEmitsNullArrays` (`internal/mcp/structured_content_test.go:237`) falha se o payload serializar qualquer `:null`, e a viewer lê ausência como `undefined` (`zod/mini` `.optional()`), nunca `null`. Agrava o mecanismo: o `encoding/json` serializa um ponteiro `nil` como `null` quando o campo **não** tem `omitempty` — ou seja, um `*int` sem `omitempty` produziria exatamente o `null` que o contrato do wire proíbe e que a viewer rejeitaria. Por isso a tag é `json:"health_score,omitempty"`: a chave simplesmente **não viaja**. "Ausente" é o idioma que o repo já usa para "sem valor" (arrays viajam como `[]`, nunca `null`); `null` seria um contrato novo e quebraria um teste que existe justamente para isso.

A viewer foi deixada **tolerante aos dois** (`null` e ausência), normalizando para `undefined` (`z.pipe(z.nullish(z.number()), z.transform(...))`), do jeito que `codeSymbolListSchema` (`viewer/src/lib/schemas.ts:619`) já tolera o `null` que o Go manda em outros campos. Assim a viewer não quebra se o encoding do Go mudar, se um cliente antigo falar com outro servidor, ou se um payload vindo de fora trouxer `null` — e o caminho de produção continua sem `null`.

**Decisões de apoio (ver `Assumptions`):** o tipo `store.DoctorReport.HealthScore` continua `int` (o contrato do pacote `store`, definido na ADR-013, não muda de forma); a matemática de penalidades vai para uma função única `store.CalculateHealthScore`, usada pelos dois motores, para que SQLite e PostgreSQL não voltem a divergir; e a viewer passa a tolerar `health_score` ausente **ou** `null`.

## Assumptions & Open Questions

| Assumption / decision | Chosen default | Rationale | Confirmed? |
| --------------------- | -------------- | --------- | ---------- |
| A1 — Representação do "score não se aplica" | `health_score` ausente do payload MCP; CLI imprime "não se aplica" | (A) é a única saída que não deixa número enganoso no payload; ausente ≠ 0 e ≠ 100, e é o idioma que o contrato do wire deste repo já usa (null é proibido no payload) | y |
| A2 — Escopo da mudança de tipo | `mcp.DoctorReport.HealthScore` vira `*int`; `store.DoctorReport.HealthScore` continua `int` | O contrato público (payload MCP) precisa de "ausência de valor"; o tipo interno do pacote `store` não muda de forma, o que mantém o raio da mudança dentro do que a ISSUE-019 pede | y |
| A3 — Origem da regra de aplicabilidade | Um único método `DoctorReport.HealthScoreApplicable()` (= `TotalDocuments > 0`) + uma função única `store.CalculateHealthScore` | Duas cópias da matemática de penalidades (SQLite e PostgreSQL) já divergiram no passado; uma função só impede a regressão e dá um alvo testável ao defeito | y |
| A4 — Valor do score no relatório interno quando não aplicável | `0` (inerte) + `HealthScoreApplicable() == false` | `int` não tem valor nulo; 0 marcado como inerte é o menor valor posible e nunca é apresentado como veredito | y |
| A5 — Indicador visual no CLI | `⚪` (neutro) em vez de `🟢` quando não aplicável | Verde + 100/100 é o que fez o defeito passar despercebido; um indicador neutro com rótulo explícito não repete o erro | y |
| A6 — Compatibilidade do payload | `health_score` ausente quando não aplicável; demais campos e Markdown inalterados para vault com documentos | O teste de estabilidade do `structuredContent` (`internal/mcp/structured_content_test.go`) continua valendo sem alteração | y |
| A7 — Viewer | Schema tolera `null` **e** ausência, normalizando para `undefined`; o tile de Health Score mostra "—" e o rótulo "vault sem documentos" | O payload mudou de forma; a viewer é a única outra leitora do `structuredContent` e não pode quebrar com o valor novo — nem se o encoding do Go mudar | y |
| A8 — Escopo da execução | Sem ADR nova, sem alteração de pesos, sem docs | A ADR-013 continua válida; docs e ADRs estão fora da propriedade desta execução | y |

**Open questions**: none — todas as decisões estão resolvidas e registradas acima (A1 a A8).

## User Stories

### P1: Diagnóstico de vault vazio (a falha que a ISSUE-019 abriu)

**User Story**: Como agente de IA ou pessoa que rodou `mem doctor`, quero saber se o vault está doente ou apenas não indexado, so that eu não veja um veredito de saúde perfeito num vault que não existe.

**Why P1**: O `memory_doctor` foi a origem do achado (ISSUE-019) — a métrica precisa servir ao caso "não indexado", que é o primeiro estado em que qualquer vault se encontra.

**Acceptance Criteria**:
1. WHEN `mem doctor` audita um vault sem documentos indexados THEN the system SHALL NOT report a Health Score of 100/100 AND SHALL report explicitamente que o score não se aplica porque o vault não tem documentos indexados.
2. WHEN a ferramenta MCP `memory_doctor` audita um vault sem documentos indexados THEN the system SHALL NOT emit the `health_score` key in the structuredContent AND SHALL state in the Markdown that o vault não tem documentos indexados.
3. WHEN o relatório apresenta um vault sem documentos indexados THEN the system SHALL indicar a ação de recuperação (`mem index`) AND SHALL NOT usar o indicador visual de saúde verde.

**Independent Test**: Rodar o binário real `mem doctor --storage=sqlite --db <arquivo-vazio.db> --repo acme` e conferir que a saída não contém "Health Score: 100/100" e contém "não se aplica".

---

### P2: O caminho saudável não pode regredir

**User Story**: Como pessoa que já usa `mem doctor` para curar o grafo, quero que as penalidades de dead links, órfãs, self-loops e chunks dessincronizados continuem valendo, so that a correção do caso "vault vazio" não embaralhe a métrica que eu já uso.

**Why P2**: A correção toca a mesma função que produz o score dos vaults reais. Sem regressão coberta, "não se aplica" pode virar "sempre 100".

**Acceptance Criteria**:
1. WHEN um relatório tem documentos indexados e dead links THEN the system SHALL aplicar a penalidade de 5 pontos por dead link (máximo de 40) conforme a ADR-013.
2. WHEN um relatório tem documentos indexados e notas órfãs THEN the system SHALL aplicar a penalidade proporcional de órfãs (razão × 40, máximo de 30) conforme a ADR-013.
3. WHEN um relatório tem documentos indexados e self-loops e chunks dessincronizados THEN the system SHALL aplicar 5 pontos por self-loop (máximo de 15) e 2 pontos por chunk dessincronizado (máximo de 15) conforme a ADR-013.
4. WHEN um relatório tem documentos indexados THEN the system SHALL report a Health Score entre 0 e 100 AND the same value in the CLI, no Markdown do MCP e no structuredContent.

**Independent Test**: Rodar `store.CalculateHealthScore` sobre relatórios montados (perfeito, 2 dead links, órfã + self-loop) e conferir os scores 100, 90 e 87.

---

### P3: Uma regra só, dois motores

**User Story**: Como mantenedor do repositório, quero que SQLite e PostgreSQL compartilhem o cálculo do Health Score, so that o defeito não reapareça em um dos motores só.

**Why P3**: A guarda que causa o defeito está duplicada nos dois motores; corrigir uma cópia e esquecer a outra deixa metade do produto quebrada.

**Acceptance Criteria**:
1. WHEN o SQLite ou o PostgreSQL calcula o Health Score THEN the system SHALL usar a mesma função compartilhada de cálculo AND SHALL NOT duplicar a matemática de penalidades no chamador.
2. WHEN o relatório de qualquer motor tem zero documentos indexados THEN the system SHALL report the score como não aplicável AND SHALL NOT report 100/100.

**Independent Test**: Auditar os dois call sites (`internal/db/graph.go` e `internal/store/doctor.go`) e confirmar que ambos chamam a função compartilhada; rodar o diagnóstico SQLite num banco vazio e confirmar o score não aplicável.

## Edge Cases

- IF o vault tem documentos mas zero anomalias THEN the system SHALL report 100/100 (caminho saudável preservado).
- IF o vault tem documentos e penalidades suficientes para zerar o score THEN the system SHALL report 0/100 — 0 continua sendo um score legítimo e não pode ser confundido com "não aplicável".
- IF o relatório chega ao formatador MCP sem nenhum documento (inclusive o caso `report == nil`, tratado como relatório vazio) THEN the system SHALL report the score como não aplicável.
- IF o banco não tem tabela `documents` (banco não inicializado) THEN the contagem vem como 0 e o score é não aplicável — sem panic.

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----- | ----- | ------ |
| EMPTY-01 | P1: Diagnóstico de vault vazio | Execute | Verified |
| EMPTY-02 | P1: Diagnóstico de vault vazio | Execute | Verified |
| EMPTY-03 | P1: Diagnóstico de vault vazio | Execute | Verified |
| EMPTY-04 | P2: O caminho saudável não pode regredir | Execute | Verified |
| EMPTY-05 | P2: O caminho saudável não pode regredir | Execute | Verified |
| EMPTY-06 | P2: O caminho saudável não pode regredir | Execute | Verified |
| EMPTY-07 | P2: O caminho saudável não pode regredir | Execute | Verified |
| EMPTY-08 | P3: Uma regra só, dois motores | Execute | Verified |

**Coverage:** 8 total, 8 mapeados para tarefas, 0 não mapeados.

## Success Criteria

- [x] `mem doctor` num vault vazio não reporta mais "Health Score: 100/100" (CLI e MCP).
- [x] `health_score` no `structuredContent` do `memory_doctor` viaja **ausente** quando o vault não tem documentos indexados (e nunca `null` — o contrato do wire do repo proíbe `null`).
- [x] O JSON do `structuredContent` para vault com documentos continua byte-idêntico (`internal/mcp/structured_content_test.go` sem alteração de expectativa).
- [x] Teste de regressão que falha contra o defeito revertido.
- [x] `go build ./...`, `go vet ./...` e `go test -count=1 ./...` com exit 0.
