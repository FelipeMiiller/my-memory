# Feature: fix-doctor-unindexed-links

- **Spec ID**: `fix-doctor-unindexed-links`
- **Created**: 2026-11-13
- **Emended**: 2026-11-14 (medição real no Postgres do cofre central — números e premissas corrigidos) · 2026-11-14, rodada 2 (achado do Verifier: o Out of Scope dizia 90/100 para o vault de repositório; o real é **75/100** desde a spec `fix-zero-vector-chunks`, que tornou visível a degradação de vetor zero)
- **Owner**: Felipe Miiller
- **Source**: ISSUE-039 — `.specs/ISSUES.md` (seção ISSUE-039) · fecha o resto da ISSUE-038
- **Related**: ADR-013 (Health Score e suas penalidades) · ADR-041/ADR-055 (tags) · ADR-062 §D-2 e §D-3 · ISSUE-036 (nó de código como alvo vivo)

## Problem Statement

O `mem doctor` conta **138** links do cofre central (`repo_central`) como
"mortos". A maior parte **não** está quebrada: são referências a material que
**existe no disco** e que está **fora do índice por decisão** — o `config.yaml`
do cofre exclui `catalog-skills/skills/**` do escopo de indexação, e as notas
destiladas linkam esse material (`[[review-standards]]`, `[[mcp-guide]]`…). No
Obsidian esses links resolvem; no índice não.

**Medido no Postgres real (2026-11-14, `repo_central`, dados de 2026-11-14):**

| Classe | Arestas | Como resolve | Onde vive o alvo |
| :--- | ---: | :--- | :--- |
| Existe em disco, fora do índice por decisão | **125** | 112 por basename · 13 por `title:` do frontmatter | **todas** em `catalog-skills/skills/**` (área excluída) |
| Não existe em disco | **13** | — | 2 notas de origem, `links_to` |

Os 13 que não existem são **conceitos sem nota** (`Docker`, `Kubernetes`,
`Rollback`, `Observabilidade`, `Banco de Dados`, `Busca Vetorial e RAG`,
`SRE e SLOs`, `Transações e Concorrência`, `Índices SQL`,
`Backup e Recuperação`, `Filas e Processamento Assíncrono`,
`Observabilidade de Banco de Dados`, `Expand and Contract`). São links
**rompidos de verdade** — a nota que os referencia é que não tem destino.

Consequências no estado atual:

- O Health Score real do cofre é **45/100** (penalidade de dead link no teto
  de 40 + penalidade de chunk sem vetor no teto de 15), não 60/100.
- O `mem doctor --fix` de hoje, que decide só por SQL, apagaria **138**
  arestas — das quais **125** apontam para material que existe em disco. Ele
  apagaria links vivos no Obsidian.
- O critério "doctor ≥ 90/100" da ADR-062 §B-2 é inatingível sem uma decisão
  de **conteúdo** sobre os 13 conceitos sem nota (ver Out of Scope).

A correção da ISSUE-038 (a aresta de tag deixou de ser dead link) levou
148 → 138; o que resta depende de uma **checagem em disco**, que o doctor não
faz hoje porque não conhece o caminho do vault.

A decisão do dono (2026-11-13) é explícita: **"fora do índice por decisão" ≠
"link rompido"**. Esta spec implementa essa distinção.

## Goals

- [ ] O doctor classifica cada link suspeito em três estados: **rompido**,
      **fora do índice por decisão** e **não verificável**.
- [ ] Só o link **rompido** é contado como dead link e penaliza o Health Score.
- [ ] A rotina de correção (`--fix`) nunca apaga uma aresta cujo alvo existe em
      disco.
- [ ] O comportamento é o mesmo nos dois engines (SQLite e Postgres).
- [ ] Nenhum dado gravado muda: a correção é **só de medição**.

## Out of Scope

| Item | Reason |
| :--- | :--- |
| Remover os wikilinks para o material bruto | É a **opção 3** da ISSUE-039: edição de conteúdo do cofre, não de código. O dono escolheu a distinção, não a remoção. |
| Mover o material bruto para dentro do índice | Mudaria o escopo do cofre e o `config.yaml` dele — decisão de conteúdo. |
| Gravar o alvo da tag com `#` | Mudaria o **dado** gravado e exigiria reindexar os dois vaults. |
| Mudar a fórmula de penalidades além do necessário | A ADR-013 fixa as penalidades; aqui só muda **o que entra na lista**. |
| O Health Score do vault de repositório | Ele está em **75/100** com 2 dead links reais (`[[ADR-035]]`), que continuam reais — os 2 links custam 10 pontos e os 101 chunks sem vetor real custam os outros 15 (a spec `fix-zero-vector-chunks` tornou essa degradação visível; antes ela estava escondida sob o vetor zero, e o score aparentava 90/100). |
| **Criar nota stub para os 13 conceitos sem nota (ou remover os wikilinks)** | Decisão de **conteúdo do cofre** — é o dono que sabe se quer 13 notas novas ou 13 links removidos. A spec entrega a distinção (rompido / fora-do-índice / não-verificável) e a medição; a escolha é dele. Sem essa escolha, o critério ≥ 90 da ADR-062 §B-2 **não fecha** — e o score honesto do cofre é **60/100** depois do reindex com provider. |

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| Como o doctor descobre a raiz do vault | flag `--vault` > `central_vault.path` do config global > diretório de trabalho atual | o config global já declara `central_vault.path: G:\My Drive\central-memory`; a flag resolve o caso do Postgres sem cópia local | y |
| Semântica de resolução do alvo | **basename** em qualquer profundidade · **caminho relativo** com/sem `.md` · **`title:` do frontmatter** · **`aliases:` do frontmatter** | é a semântica do Obsidian *e* a do próprio indexador: a identidade de um documento no índice é `documents.title` (= `COALESCE(frontmatter.title, basename)`, ISSUE-010) e o resolver reescreve wikilink e tag contra os **títulos** indexados (`parser.ResolveTagConnections`). Medido: 112 resolvem por basename e **13 só por `title:`** — sem o título, esses 13 seriam classificados como rompidos e o `--fix` os apagaria. | y |
| A resolução é **exata**, não fuzzy | match exato (case-insensitive) contra cada uma das 4 chaves | a checagem em disco decide se algo é *rompido*; fuzzy resolveria "aproximadamente" e poderia declarar vivo um link quebrado — o "verde-mentira" que a ISSUE-033 verificou. Medido: **0** dos 125 resolvedores dependem de match case-insensitive, e o match por token do indexador (≥ 4 chars) traria falsos positivos. | y |
| O que fazer com os 13 alvos que não existem em disco | continuam **rompidos** | o doctor não tem evidência de que o link "deveria" resolver; a correção de conteúdo (nota stub ou remoção do wikilink) é decisão do dono | y |
| O que fazer quando a raiz não é acessível | classificar como **não verificável**: entra na contagem de dead link (penalidade) e o relatório diz que o score é provisório | o doctor não tem evidência para limpar a suspeita; limpar a suspeita sem prova é o "verde-mentira" que a ISSUE-033 verificou | y |
| A categoria nova entra no Health Score | não | o score mede a saúde do vault; um link que funciona no Obsidian não é doença | y |
| O `target_id` gravado é o texto do wikilink | sim — a classificação reusa o `target_id` que a query de dead link já devolve | não há dado novo para gravar; o doctor já tem o alvo | y |
| A correção (`--fix`) apaga o quê | só as arestas listadas como **rompidas** | o ISSUE-038 decorreu de a rotina e o relatório usarem critérios diferentes; a lista nova é a **única** lista que a rotina apaga | y |

Open questions: none

## User Stories

### P1: O doctor distingue o que está quebrado do que foi excluído de propósito ⭐ MVP

**User Story**: Como dono do cofre central, quero que o `mem doctor` me diga
quais links estão realmente quebrados e quais apenas apontam para material que
eu escolhi não indexar, para que o Health Score signifique alguma coisa.

**Why P1**: 125 dos 138 "dead links" do cofre central são links que funcionam
no Obsidian; o score de 45/100 é a medida de um vault saudável penalizando uma
decisão de configuração.

**Acceptance Criteria**:

1. WHEN the doctor audits a vault whose root path is known and a link target
   resolves to a file inside it THEN the link SHALL NOT be reported as a dead
   link.
2. IF the link target resolves to no file in the vault and to no indexed
   document THEN the link SHALL continue to be reported as a dead link.
3. The report SHALL expose the new category with the target, the source
   document, the relation, and the path on disk where the target was found.
4. The Health Score SHALL NOT be penalized by links classified as "out of index
   by decision".
5. IF the vault root is unknown or unreadable THEN the affected links SHALL be
   classified as "unverifiable", SHALL keep the dead link penalty, and the
   report SHALL state that the score is provisional because those links could
   not be verified.

**Independent Test**: num vault de teste, uma nota que aponte para um arquivo
que existe no disco mas fora do escopo indexado → o doctor lista o arquivo na
categoria nova e o score não cai.

---

### P1: A correção não apaga o que está vivo

**User Story**: Como dono do cofre, quero que o `mem doctor --fix` apague só o
que está comprovadamente quebrado, para não perder as arestas que ligam as
notas ao material que deixei fora do índice.

**Why P1**: é a mesma lição da ISSUE-038 (o relatório e a rotina usavam
critérios diferentes) e a mesma do ISSUE-036 (a rotina apagava 38 304 arestas
vivas da ponte do code graph).

**Acceptance Criteria**:

6. WHEN the correction routine runs THEN it SHALL delete exactly the edges the
   report lists as dead links, and SHALL preserve every edge whose target
   exists on disk.
7. WHEN the correction routine runs on a vault with unverifiable links THEN it
   SHALL preserve those edges as well, because it cannot prove them broken.

**Independent Test**: `--fix` num vault com a nota apontando para material fora
do índice → a aresta continua lá depois do `--fix`.

---

### P2: O relatório e o payload do MCP contam a mesma história

**User Story**: Como agente que consome o doctor via MCP, quero as mesmas três
categorias que a CLI, para não decidir por um relatório simplificado.

**Why P2**: `internal/mcp/handlers.go` mantém um espelho do `DoctorReport`; sem
a nova categoria, o agente veria o vault central como 45/100.

**Acceptance Criteria**:

8. The MCP doctor payload SHALL expose the same three categories as the CLI
   report, with the same counts.
9. The doctor CLI SHALL accept a `--vault` flag that overrides the vault root
   used for the disk check.

**Independent Test**: `memory_doctor` no MCP e `mem doctor` na CLI devolvem os
mesmos números para o mesmo vault.

## Edge Cases

- IF the vault root exists but is empty THEN every link is classified as broken
  (no file to find) — a checagem é honesta, não otimista.
- IF the vault root does not exist THEN the links are unverifiable (not broken).
- IF two files in the vault share the basename of the target THEN the first
  match is reported, and the report names the resolved path so the ambiguity is
  visible. (Medido: **17** das 138 arestas do cofre central têm basename
  ambíguo.)
- IF the target contains a folder prefix THEN the resolution tries the exact
  relative path first, and only then falls back to the basename search.
- IF the target matches the `title:` of a file in the vault THEN it is
  resolved — a identidade de um documento no índice é o `title`, não o nome do
  arquivo. (Medido: **13** das 138 arestas do cofre central resolvem só assim.)
- IF the target matches an `alias:` of a file in the vault THEN it is resolved.
- IF the target is a tag (relation `tagged_as`) that exists in `document_tags`
  THEN it is alive (regra da ISSUE-038) and is never reclassified by the disk
  check.
- IF the vault is on a different host than the Postgres server THEN the root is
  unreadable and the links are unverifiable — never silently "broken".

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Where |
| :--- | :--- | :--- |
| DUL-01 | AC1 — alvo em disco não é dead link | `internal/vaultfs/target.go` · `internal/db/graph.go` · `internal/store/doctor.go` |
| DUL-02 | AC2 — alvo ausente continua dead link | `internal/db/graph.go` · `internal/store/doctor.go` |
| DUL-03 | AC3 — a categoria nova traz alvo, origem, relação e path | `internal/store/store.go` |
| DUL-04 | AC4 — a categoria nova não penaliza o score | `internal/store/store.go` |
| DUL-05 | AC5 — sem raiz acessível: não verificável e score provisório | `internal/vaultfs/root.go` · `internal/store/doctor.go` |
| DUL-06 | AC6 — a correção apaga só o que está no relatório | `internal/db/graph.go` · `internal/store/doctor.go` |
| DUL-07 | AC7 — a correção preserva o não verificável | `internal/db/graph.go` · `internal/store/doctor.go` |
| DUL-08 | AC8 — o MCP expõe as mesmas categorias | `internal/mcp/handlers.go` |
| DUL-09 | AC9 — flag `--vault` no CLI | `cmd/mem/main.go` |

**Coverage**: 9 total, 9 mapeados, 0 unmapped.

## Success Criteria

- [ ] O cofre central passa a **listar 125** links na categoria "fora do índice
      por decisão" (cada um com o caminho em disco onde o alvo foi encontrado) e
      **13** na categoria "rompida" (conceitos sem nota). O relatório deixa de
      acusar 138 links vivos.
- [ ] O `mem doctor --fix` do cofre central apaga **13** arestas, não 138 — e
      preserva as 125 cujo alvo existe em disco.
- [ ] Os 2 dead links reais do vault do repositório (`[[ADR-035]]`) continuam
      dead links.
- [ ] **O Health Score do cofre central CONTINUA 45/100 com esta spec**: com
      13 dead links a penalidade (5 pts cada) ainda satura o teto de 40 — o
      mesmo teto que os 138 saturavam. O que muda é o relatório (13 em vez de
      138) e o `--fix` (13 em vez de 138).
- [ ] O score chega a **60/100** depois do reindex com o provider (que tira a
      penalidade de 15 dos chunks sem vetor). Para chegar a **≥ 90** (critério
      §B-2 da ADR-062) é preciso ≤ 2 dead links — o que depende da decisão de
      **conteúdo** sobre os 13 conceitos sem nota (ver Out of Scope). A spec
      entrega a distinção e a medição; ela **não fecha** o critério de score.
