# ADR-063: O doctor distingue link rompido de link fora do índice por decisão

- **Date**: 2026-11-14
- **Status**: Accepted — Implementado
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: doctor, dead-links, health-score, obsidian, vault, filesystem, adr-013, adr-062
- **Adenda**: [ADR-013](013-higiene-de-grafo-pruning-e-doctor.md) (o doctor e o Health Score) · [ADR-062](062-contrato-dos-dois-cofres-e-o-que-o-ai-memory-ensina.md) (o contrato dos dois cofres)
- **Spec**: `.specs/features/fix-doctor-unindexed-links/` · **Issues**: [ISSUE-039](../.specs/ISSUES.md) (fecha a ISSUE-038)

## Context and Problem Statement

O `mem doctor` tratava **toda** aresta de link que não resolve no índice como
**dead link**. No Cofre Central isso acusava **138** links e dava **45/100** de
Health Score. A medição real (Postgres do cofre `repo_central`, 2026-11-14)
mostrou que **125 dos 138** apontam para material que **existe em disco** e que
está **fora do índice por decisão** — o `config.yaml` do cofre exclui
`catalog-skills/skills/**` do escopo de indexação, e as notas destiladas linkam
esse material (`[[review-standards]]`, `[[mcp-guide]]`). No Obsidian esses links
resolvem; no índice não.

Dois problemas reais, e um deles já tinha sido pago antes:

1. **A métrica media uma decisão de configuração como doença.** O Health Score
   punia o vault por um escopo escolhido pelo dono.
2. **A rotina de correção apagaria links vivos.** O `mem doctor --fix` de hoje
   decide só por SQL e apagaria as **138** arestas — das quais **125** têm alvo
   vivo em disco.

O segundo problema é a **mesma** classe do defeito da ISSUE-038: o relatório e a
rotina de correção usavam **critérios diferentes**, e a rotina (que apaga) é a
que faz estrago. A ISSUE-038 já tinha mostrado que "aresta de tag não é dead
link" (148 → 138) e que o que resta dependia de uma checagem em disco — que o
doctor não fazia, porque não conhecia o caminho do vault.

O dono fixou a direção em 2026-11-13: **"fora do índice por decisão" ≠ "link
rompido"**.

## Decision Drivers

- **DR-1 — Um critério só.** O relatório e a rotina de correção têm que decidir
  pelo **mesmo** classificador. É a lição da ISSUE-038, e é a razão de o
  classificador ser compartilhado em vez de duplicado por engine.
- **DR-2 — O Health Score mede saúde do vault.** Um link que funciona no
  Obsidian não é doença do grafo; ele é uma escolha de escopo.
- **DR-3 — A checagem em disco tem que ser honesta.** Quem decide se algo está
  rompido precisa de **prova de ausência**. Match aproximado ("verde-mentira",
  ISSUE-033) declararia vivo um link quebrado.
- **DR-4 — Os dois engines contam a mesma história.** SQLite e PostgreSQL
  produzem o mesmo relatório, a mesma lista de correção e o mesmo score.
- **DR-5 — A semântica de resolução é a do Obsidian e a do próprio indexador.**
  A identidade de um documento no índice é `documents.title`
  (`COALESCE(frontmatter.title, basename)`, ISSUE-010), e o indexador reescreve
  wikilink e tag contra os **títulos** indexados (`parser.ResolveTagConnections`).
- **DR-6 — Sem mudança de schema e sem mudança de dado gravado.** A spec é de
  **medição**: o que passa a ser gravado (ou não) continua igual.

## Considered Options

- **A — Classificar em três estados com checagem em disco (escolhida)**
- **B — Manter o dead link e apenas rotular no relatório**
- **C — Indexar o material bruto (mudar o escopo do cofre)**
- **D — Remover os wikilinks que apontam para o material bruto**

## Decision Outcome

Escolhida a **Opção A**. O `mem doctor` passa a classificar cada link suspeito
em **três** estados:

| Estado | Significado | Penaliza o Health Score | A `--fix` apaga? |
| :--- | :--- | :---: | :---: |
| **rompido** | não existe em disco **e** não existe no índice | sim (5 pts, teto 40) | sim |
| **fora do índice por decisão** | existe em disco, mas fora do escopo indexado | **não** | **não** |
| **não verificável** | a raiz do vault é desconhecida ou ilegível | sim (5 pts, teto 40) | **não** |

E cada estado tem uma consequência de produto:

- **A raiz do vault** é resolvida por `internal/vaultfs` na precedência:
  flag `--vault` > entrada do **catálogo do repositório auditado** >
  `central_vault.path` do config global (**só** para o cofre central) >
  diretório de trabalho atual. O catálogo entra antes do `central_vault.path`
  porque auditar `repo_xxx` contra os arquivos do cofre central mediria a coisa
  errada.
- **O alvo** é resolvido contra o disco com match **exato** (case-insensitive),
  nunca fuzzy, em quatro chaves: **basename** em qualquer profundidade,
  **caminho relativo** (com e sem `.md`), **`title:`** e **`alias:`** do
  frontmatter. O match exato é o que mantém a checagem honesta (DR-3).
- **A lista é única.** `store.ClassifyLinks` + `store.NewVaultResolver` é o
  **único** classificador; o relatório e a rotina de correção consomem a mesma
  lista (DR-1).
- **A raiz ilegível degrada em vez de inventar.** Os links ficam "não
  verificáveis", **mantêm** a penalidade e o relatório declara o score
  **provisório** — porque sem prova não se limpa a suspeita.

**Medido no Postgres real (`repo_central`, 2026-11-14)**: as 138 arestas suspeitas
se dividem em **125 fora do índice por decisão** (112 resolvem por basename e **13
só por `title:`**) e **13 rompidas** (conceitos sem nota: `Docker`,
`Kubernetes`, `Rollback`, `Observabilidade`, `Banco de Dados`,
`Busca Vetorial e RAG`, `SRE e SLOs`, `Transações e Concorrência`, `Índices SQL`,
`Backup e Recuperação`, `Filas e Processamento Assíncrono`,
`Observabilidade de Banco de Dados`, `Expand and Contract`). O `--fix` passou de
**138** arestas apagadas para **13**. Os 2 dead links reais do vault de
repositório (`[[ADR-035]]`) continuam dead links.

### Positive Consequences

- O relatório deixa de acusar **138** links vivos e passa a acusar os **13** que
  estão realmente quebrados.
- A rotina de correção **não apaga mais** aresta cujo alvo existe em disco: o
  `--fix` de 138 para 13 arestas no cofre central.
- O Health Score volta a medir **saúde do vault** e não escopo de indexação.
- O mesmo contrato nos **dois** engines, com o classificador **compartilhado** —
  a lista que o relatório mostra é literalmente a lista que a correção apaga.
- Nenhum dado gravado mudou: é uma correção de **medição** (DR-6).

### Negative Consequences

- O doctor passa a depender do **sistema de arquivos**: sem a raiz do vault não
  há classificação, e o resultado é um score **provisório** em vez de um número.
- Uma superfície nova (`--vault`) e uma categoria nova no payload do MCP
  (`memory_doctor`), que precisam manter as mesmas contagens da CLI (AC8).
- A classificação custa I/O de disco por link suspeito (a raiz e o índice de
  títulos/aliases são lidos uma vez por auditoria).
- **Os 13 conceitos sem nota continuam pendentes.** Eles são links rompidos de
  verdade, e o score honesto do cofre central é **60/100** depois do reindex com
  o provider — o critério de **≥ 90** da ADR-062 §B-2 **não fecha** sem uma
  decisão de **conteúdo** (criar as notas ou remover os wikilinks), que é do
  dono, não do código.

## Pros and Cons of the Options

### Opção A ✅ Chosen

- ✅ Separa a doença real da decisão de configuração (DR-2).
- ✅ Impede a rotina de correção de apagar material vivo (DR-1).
- ✅ É honesta quando falta evidência: "não verificável" em vez de "rompido"
  (DR-3).
- ✅ Mantém a semântica do Obsidian e do indexador (DR-5).
- ❌ Custa I/O de disco por auditoria.
- ❌ Introduz um score provisório, que é um número que o dono precisa saber
  ler.

### Opção B — rotular sem reclassificar

- ✅ Nenhuma mudança de schema nem de fluxo.
- ❌ O `--fix` continuaria apagando os 125 links vivos: o defeito que a spec
  existe para fechar continua aberto.
- ❌ O Health Score continuaria medindo escopo de configuração.

### Opção C — indexar o material bruto

- ✅ Resolve a ambiguidade na origem.
- ❌ Muda o escopo do cofre e o `config.yaml` dele: decisão de conteúdo, com
  custo de indexação de ~700 arquivos de material bruto.
- ❌ O escopo do cofre é **do dono** — o produto não pode redefini-lo sozinho.

### Opção D — remover os wikilinks do material bruto

- ✅ Deixa o grafo do índice consistente por construção.
- ❌ Edita o **conteúdo** do cofre e quebra o uso no Obsidian, onde os links
  funcionam.
- ❌ O dono escolheu explicitamente a **distinção**, não a remoção.

## Links

- [ADR-013](013-higiene-de-grafo-pruning-e-doctor.md) — Higiene de Grafo,
  Pruning Incremental e Linter Doctor (a matemática das penalidades **não
  muda**: o que muda é **o que entra na lista** de dead link).
- [ADR-062](062-contrato-dos-dois-cofres-e-o-que-o-ai-memory-ensina.md) — o
  contrato dos dois cofres; o critério de prontidão §B-2 continua **aberto**
  por causa dos 13 conceitos sem nota.
- [ADR-041](041-issue-009-tag-sem-casa-removida-do-grafo.md) ·
  [ADR-055](055-semantica-de-tag-consultavel-sem-nos-mortos.md) — a aresta de
  tag viva, que o doctor nunca reclassifica.
- [ISSUE-038](../.specs/ISSUES.md) (o relatório e a rotina discordando) ·
  [ISSUE-039](../.specs/ISSUES.md) (esta spec) ·
  [ISSUE-033](../.specs/ISSUES.md) (verde-mentira em asserção vaga).
- Spec: `.specs/features/fix-doctor-unindexed-links/` ·
  implementação: `internal/vaultfs/`, `internal/store/doctor_links.go`,
  `internal/store/doctor.go`, `internal/db/graph.go`, `internal/mcp/handlers.go`,
  `cmd/mem/main.go`.
