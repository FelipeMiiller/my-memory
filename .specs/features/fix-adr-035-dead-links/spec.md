# Feature: fix-adr-035-dead-links

- **Spec ID**: `fix-adr-035-dead-links`
- **Created**: 2026-11-16
- **Owner**: Felipe Miiller
- **Source**: ISSUE-049 — `.specs/ISSUES.md` (seção ISSUE-049) · fecha os 2 links rompidos do vault do repositório
- **Related**: ADR-035 (`docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md`) · ADR-013 (Health Score) · a lição de path-style em `.specs/042-stubs-editoriais/validation.md:90`

## Problem Statement

O `mem doctor` do vault do repositório reportou **2 links rompidos** — e ambos
são o **mesmo wikilink** `[[ADR-035]]`, usado em 3 lugares:

| Onde | Ocorrência |
| :--- | :--- |
| `.specs/features/embedder-builtin-onnx-minilm/spec.md` | linha 11 (`> **Tipo:** Feature. Executa o plano de implementação do [[ADR-035]]…`) e linha 188 (`- [[ADR-035]] — a decisão que esta spec executa`) |
| `.specs/features/embedder-builtin-onnx-minilm/tasks.md` | linha 3 (`**ADR:** [[ADR-035]]`) |

**Por que não resolve.** O arquivo da ADR se chama
`docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md` e **não tem
frontmatter `title`** — o `ADR-035` do H1 não é o título indexado. Sem `title`
nem `alias`, o match é por basename, e `ADR-035` ≠
`035-embedder-embutido-com-fallback-onnx-minilm`.

**Custo medido**: o vault do repositório está em **90/100** — a penalidade de
dead link (5 pontos por link, teto de 40) consome **10 pontos** por **2
arestas** (o mesmo arquivo, 2 ocorrências). Os **101 chunks sem vetor real**
(que a ISSUE-040 tornou visíveis) consomem os outros 15; depois da correção da
ISSUE-040 o score do repo vai a **100/100**.

**O conserto já está escrito** — é a lição registrada pelo próprio repo
(`.specs/042-stubs-editoriais/validation.md:90`):

> Plain `[[ADR-001]]` é frágil (depende de basename match). Path-style
> `[[docs/adr/001-…|ADR-001]]` é robusto (casa exato). Alias preserva display
> curto no render. **Regra: pra refs cruzadas entre docs, sempre usar
> path-style com alias.**

O que falta é **aplicar** a regra — e **prová-la**, porque a lição veio do
render do Obsidian, não do indexador do my-memory. Esta spec faz as duas
coisas.

## Goals

- [ ] As 3 ocorrências de `[[ADR-035]]` passam a ser
      `[[docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md|ADR-035]]`.
- [ ] O `mem doctor` do vault do repositório deixa de reportar esses links
      rompidos — **medido**, não presumido.
- [ ] O display curto `ADR-035` é preservado no render.
- [ ] A regra de path-style fica registrada como **lição** do repo, para o
      próximo doc não repetir o erro.
- [ ] **Nenhum texto de ADR é alterado** (o ADR é o registro da decisão).

## Out of Scope

| Item | Reason |
| :--- | :--- |
| Adicionar `title: ADR-035` no frontmatter da ADR | A ADR é **imutável** (ISSUE-024: "ADRs são imutáveis"). O conserto é no **documento que referencia**, não no referenciado. |
| Corrigir o **Health Score** do repo | Os 15 pontos restantes são os chunks sem vetor (ISSUE-040) — outra issue, com correção própria. |
| Trocar o basename dos arquivos de ADR | Mudaria a convenção de nomenclatura do repo inteiro. |
| Reindexar o **cofre central** (`repo_central`) | O defeito é no vault do **repositório**; o cofre central tem outra contagem de dead links (ISSUE-038/039/049). |
| Criar ADR | Nenhuma decisão de arquitetura: é correção de conteúdo com regra já registrada. |

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| **Forma** do wikilink corrigido | `[[docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md\|ADR-035]]` (path-style **com** alias) | é literalmente a regra que o repo já registrou em `.specs/042-stubs-editoriais/validation.md:90`. O alias preserva o display curto (`ADR-035`) no render do Obsidian, que é como o texto aparece ao leitor. | y — **decidido** |
| O parser do my-memory **entende** a forma com alias | **sim** — o alvo extraído é a parte **antes** do `\|` | o parser já normaliza `[[Banco de Dados\|PostgreSQL]]` para o alvo `Banco de Dados` (`internal/parser/parser_test.go:13` e `:23`). O alvo extraído é o caminho, que resolve por path. | y — **decidido** (verificado no código) |
| Como a correção é **provada** | pelo `mem doctor` do vault do repositório, depois do reindex | a lição veio do Obsidian; o indexador do my-memory tem as **suas** regras de resolução (fuzzy por título/base, ADR-063 por disco). A prova é o doctor — não a presunção. | y — **decidido** |
| O reindex do vault do repositório é **inclusão** | sim — a correção só passa a valer depois do reindex | o grafo é derivado do disco: mudar o texto do wikilink não reescreve as arestas já gravadas. Sem reindex, o doctor continua reportando os mesmos 2 links. | y — **decidido** |
| A correção afeta **quantos** links | 3 ocorrências em 2 arquivos — as únicas com `[[ADR-035]]` | o doctor reportou **2** arestas rompidas (o mesmo arquivo, 2 ocorrências indexadas). Os outros `[[ADR-0xx]]` do repo resolvem, porque as ADRs correspondentes têm `title` no frontmatter. | y — **decidido** |
| A regra de path-style vale para **todo** o repo | sim — e o **doctor** é o medidor | o `mem doctor` é o gate que já mede links rompidos: se a regra for violada em outro doc, o doctor acusa. Nenhum outro mecanismo seria necessário. | y — **decidido** |
| Onde a regra fica registrada | no **store de lições** (`.specs/lessons.json`, via `lessons.py`) | o store é machine-owned (o `LESSONS.md` é renderizado); editar à mão quebra a fonte de verdade. A lição já existe num relatório de 2026-09; falta no store, que é o que a skill carrega. | y — **decidido** |

Open questions: none — o conserto, a forma do link e a forma da prova estão
decididos acima.

## User Stories

### P1: O wikilink da ADR-035 resolve ⭐ MVP

**User Story**: Como leitor do vault, quero que o link `ADR-035` leve ao
arquivo da ADR, para que a referência não seja um link morto.

**Why P1**: é o defeito da ISSUE-049: 3 ocorrências de um wikilink que não
resolve, custando 10 pontos de Health Score.

**Acceptance Criteria**:
1. WHEN the wikilink to the ADR-035 decision appears in `.specs/features/embedder-builtin-onnx-minilm/spec.md` THEN it SHALL be `[[docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md|ADR-035]]` in both occurrences.
2. WHEN the wikilink to the ADR-035 decision appears in `.specs/features/embedder-builtin-onnx-minilm/tasks.md` THEN it SHALL be `[[docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md|ADR-035]]`.
3. The ADR-035 file SHALL NOT be modified by this correction.

**Independent Test**: `grep -rn "\[\[ADR-035\]\]" .specs/` não devolve nada, e
o `git diff` não toca `docs/adr/035-*.md`.

---

### P1: A correção é provada pelo doctor ⭐ MVP

**User Story**: Como dono do repo, quero a prova de que o link voltou a
resolver, para que o Health Score do vault volte a 100/100.

**Why P1**: a regra veio do render do Obsidian; o indexador do my-memory tem
regras próprias de resolução. Sem medição, seria "verde-mentira".

**Acceptance Criteria**:
4. WHEN the repository vault is reindexed after the correction THEN the `mem doctor` report SHALL list zero broken links whose target is a wikilink to an ADR file.
5. WHEN the corrected wikilink is rendered THEN the display text SHALL remain `ADR-035`.

**Independent Test**: `mem index --force` + `mem doctor` no vault do repositório
→ 0 links rompidos de ADR.

---

### P2: A regra fica registrada

**User Story**: Como quem escreve o próximo documento, quero a regra de
path-style disponível, para não repetir o erro.

**Acceptance Criteria**:
6. The path-style-with-alias rule for cross-document references SHALL be recorded in the lessons store through the `lessons.py` script.

**Independent Test**: `lessons.py list --status confirmed` mostra a lição.

## Edge Cases

- IF um documento referencia uma ADR **com** `title` no frontmatter THEN o
  wikilink simples já resolve — a regra de path-style vale para **todas** as
  refs cruzadas, porque "resolve hoje" não é "resolve amanhã" (o frontmatter
  pode sair).
- IF o alias do wikilink for omitido THEN o display vira o caminho inteiro
  (`docs/adr/035-embedder-…-minilm.md`) — o alias é o que mantém o texto
  legível.
- IF o reindex for rodado **antes** da correção THEN os links rompidos
  permanecem no grafo — a ordem importa: corrigir o texto, depois reindexar.
- IF o `mem doctor` acusar **outros** links rompidos no vault do repositório
  que não sejam de ADR THEN eles **não** são escopo desta correção (a ISSUE-049
  mediu exatamente 2).

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Where | Status |
| :--- | :--- | :--- | :--- |
| ADL-01 | AC1 — as 2 ocorrências do `spec.md` viram path-style com alias | `.specs/features/embedder-builtin-onnx-minilm/spec.md` | Pending |
| ADL-02 | AC2 — a ocorrência do `tasks.md` vira path-style com alias | `.specs/features/embedder-builtin-onnx-minilm/tasks.md` | Pending |
| ADL-03 | AC3 — o arquivo da ADR-035 não é tocado | `docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md` | Pending |
| ADL-04 | AC4 — o doctor não reporta link rompido de ADR | vault do repositório | Pending |
| ADL-05 | AC5 — o display continua `ADR-035` | `.specs/features/embedder-builtin-onnx-minilm/spec.md` | Pending |
| ADL-06 | AC6 — a regra de path-style fica no store de lições | `.specs/lessons.json` | Pending |

**Coverage**: 6 total, 6 mapeados, 0 unmapped.

## Success Criteria

- [ ] `grep -rn "\[\[ADR-035\]\]" .specs/` **não devolve nada**, e as 3
      ocorrências estão na forma path-style com alias.
- [ ] O `git diff` da correção **não toca** `docs/adr/035-embedder-embutido-com-fallback-onnx-minilm.md`.
- [ ] Depois do reindex, o `mem doctor` do vault do repositório reporta **0**
      links rompidos de ADR (hoje: 2 arestas).
- [ ] A regra de path-style está no store de lições, consultável por
      `lessons.py list`.
- [ ] O Health Score do vault do repositório deixa de perder **10 pontos** por
      esses 2 links (de 90/100 para 100/100 depois da correção da ISSUE-040).
