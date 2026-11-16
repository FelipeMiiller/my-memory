# Feature: central-vault-concept-notes

- **Spec ID**: `central-vault-concept-notes`
- **Created**: 2026-11-14
- **Owner**: Felipe Miiller
- **Source**: decisão do dono (2026-11-14, no questionário da spec `fix-doctor-unindexed-links`): **criar 13 notas stub, uma por conceito**
- **Related**: ADR-062 §B-2 (critério de prontidão do Cofre Central) · ADR-063 (as três categorias de link) · ADR-055 (tag consultável) · ISSUE-039
- **Vault**: `G:\My Drive\central-memory` (repo `repo_central`, Postgres)

## Problem Statement

O `mem doctor` do Cofre Central reporta **13 links rompidos** — e eles são
rompidos de verdade: o alvo **não existe em disco, nem no índice, nem no
material bruto do catálogo**. São conceitos linkados por 2 notas e que nunca
tiveram nota:

| Nota de origem | Conceitos linkados que não existem |
| :--- | :--- |
| `cloud/blue-green-deployment-obsidian.md` | `Docker`, `Kubernetes`, `Observabilidade`, `Rollback`, `Expand and Contract`, `SRE e SLOs` |
| `database/PostgreSQL — Visão Geral de Recursos e Ferramentas.md` | `Banco de Dados`, `Índices SQL`, `Busca Vetorial e RAG`, `Filas e Processamento Assíncrono`, `Transações e Concorrência`, `Observabilidade de Banco de Dados`, `Backup e Recuperação` |

Isso é o último obstáculo do critério **"doctor ≥ 90/100"** da ADR-062 §B-2:
com os 13 rompidos, a penalidade de dead link satura o teto de 40 e o score
fica em **45/100** (e em **85/100** mesmo depois de tirar a penalidade de chunk
sem vetor). Sem eles, o cofre passa de 90.

**A medição que fecha o diagnóstico** (Postgres `repo_central`, 2026-11-14):
das 23 arestas `links_to` das 2 notas, **13 mantêm o texto do wikilink** (não
resolveram) e **10 foram reescritas** pelo indexador para tags ou títulos já
existentes (por isso não são suspects). Criar uma nota para cada um dos 13 faz
as 13 arestas resolverem por **match exato** de título — e o doctor deixa de
contá-las.

## Goals

- [ ] Cada um dos 13 conceitos tem uma nota no vault, com o **nome do arquivo
      igual ao texto do wikilink** (padrão Obsidian: a identidade da nota é o
      arquivo) e o `title` exatamente igual ao conceito.
- [ ] As notas seguem o padrão do cofre: frontmatter canônico
      (`title`, `category`, `summary`, `tags`, `status`), `# Título` no corpo,
      e um link de volta para o README da pasta (regra MOC).
- [ ] As 13 notas entram no README da sua pasta, e a pasta `database/` ganha o
      README que faltava (regra 1 do MOC: toda pasta tem README).
- [ ] Depois do reindex, o `mem doctor` do `repo_central` reporta **0 link
      rompido** e o Health Score fica **≥ 90**.

## Out of Scope

| Item | Reason |
| :--- | :--- |
| Os 10 conceitos que o indexador já reescreveu (`CI-CD`, `Load Balancer`, `Health Checks`, `Cache e Consistência`, `Full Text Search`, `Arquitetura Backend`, …) | Não são link rompido: o indexador resolveu os alvos por tag/título. Criar nota para eles seria inventar escopo. |
| O material bruto (`catalog-skills/skills/**`) | Está fora do índice **por decisão** (ADR-063); não é disease do grafo. |
| Reescrever as 2 notas de origem | Os links já estão certos; é a nota de destino que falta. |
| A limpeza de governança do repo (§7 do handover) | Frente separada, com o caminho **A** escolhido pelo dono (inventário → podar → simplificar). |
| O bucket GAP-6 a GAP-10 da ADR-062 | Decidido item a item com medição antes de cada corte (decisão do dono, 2026-11-14). |

## Assumptions & Open Questions

| Assumption | Chosen default | Rationale | Confirmed? |
| :--- | :--- | :--- | :--- |
| Onde a nota nasce | **Na pasta da nota que a linka** (`cloud/`, `database/`) | É o comportamento nativo do Obsidian: um link que não existe cria a nota na pasta do link (`obsidian.md/help/links`). Também mantém a nota perto do contexto que a invoca. | y |
| Nome do arquivo | **O texto do wikilink, literal** (`Docker.md`, `SRE e SLOs.md`) | A identidade de uma nota no Obsidian é o arquivo; `[[Docker]]` passa a resolver sem configuração. A própria nota do PostgreSQL no cofre diz: *"crie uma nota chamada `PostgreSQL — Visão Geral de Recursos e Ferramentas`"*. O cofre já tem notas com espaço no nome (`performance/CLS - Causas e Correcoes de Layout Shift.md`). | y |
| `title` no frontmatter | **Igual ao texto do wikilink** | O indexador resolve `links_to` contra `documents.title` (`COALESCE(frontmatter.title, basename)`) — match exato resolve na primeira etapa, sem depender do fuzzy. | y |
| Frontmatter | `title`, `category: resource`, `summary`, `tags`, `status: ativo` | O padrão do cofre (as 94 notas existentes) e a skill `my-memory-format`. `category` e `summary` alimentam a taxonomia L0 (ADR-025); `tags` é a propriedade padrão do Obsidian. | y |
| Escopo do corpo | **Stub com conteúdo, não nota vazia** | O README do cofre é explícito: *"Um documento que não diz nada ainda assim disputa relevância com documentos reais"*. Cada nota tem "O que é", "Onde entra aqui" e "Relacionado". | y |
| Como entra no índice | `mem note create` (escrita atômica, UTF-8 sem BOM) e depois `mem index --force` | O ciclo de vida do cofre diz que a nota "passa por `mem note create` para entrar no índice". O `--force` é necessário porque **101 chunks** do cofre foram gravados sem vetor real (pelo binário oficial, sem a tag `onnx`) e só um reindex com o provider os conserta. | y |

Open questions: none

## User Stories

### P1: O conceito linkado tem nota ⭐ MVP

**User Story**: Como dono do cofre, quero que todo conceito que minhas notas
linkam exista como nota, para que o grafo não acuse link quebrado e eu possa
navegar de uma nota a outra no Obsidian.

**Why P1**: são exatamente os 13 que faltam; sem eles o critério de prontidão
§B-2 não fecha.

**Acceptance Criteria**:

1. WHEN o cofre for reindexado THEN os 13 wikilinks das 2 notas de origem SHALL resolver para um documento indexado (match exato de `title`).
2. WHEN o doctor auditar o `repo_central` THEN ele SHALL reportar **0 link
   rompido**.
3. EACH nota criada SHALL ter o nome do arquivo **igual** ao texto do wikilink
   que a referencia e o `title` igual ao conceito.
4. EACH nota criada SHALL ter frontmatter com `title`, `category: resource`,
   `summary`, `tags` e `status: ativo`, e o corpo SHALL começar com `# <título>`
   igual ao `title`.
5. EACH nota criada SHALL conter um link para o README da sua pasta e para a
   nota que a referencia.
6. EACH nota criada SHALL estar listada no README da sua pasta, e a pasta
   `database/` SHALL ter README (regra 1 do MOC).
7. WHEN o cofre for reindexado com o provider ONNX THEN ele SHALL reportar
   **0 chunk sem vetor real**.
8. WHEN o doctor auditar o `repo_central` THEN o Health Score SHALL ser
   **≥ 90** (critério §B-2 da ADR-062).

**Independent Test**: `mem doctor --storage=postgres --repo repo_central`
depois do reindex → 0 link rompido e score ≥ 90.

---

### P2: A nota é útil, não um placeholder vazio

**User Story**: Como dono do cofre, quero que a nota nova diga **o que é o
conceito** e **onde ele entra no meu contexto**, para que ela valha a
relevância que ocupa no índice.

**Acceptance Criteria**:

9. EACH nota criada SHALL ter uma seção "O que é" com a definição do conceito
   e uma seção "Onde entra aqui" que nomeia o contexto que a linka.
10. EACH nota criada SHALL listar ao menos uma nota relacionada do cofre.
11. NONE das notas criadas SHALL ficar órfã no grafo (toda uma é referenciada
    por uma nota do vault).

## Edge Cases

- IF dois conceitos tivessem o mesmo nome THEN they'd colidir no índice —
  **medido**: nenhum dos 13 títulos colide com os 94 documentos indexados.
- IF o `title` da nota fosse diferente do texto do wikilink THEN a resolução
  cairia no fuzzy (substring/token) e poderia casar com outro documento — por
  isso o `title` é **exatamente** o conceito.
- IF a nota não tivesse link de volta ao README da pasta THEN a busca reversa
  ("quem referencia isto?") quebraria — por isso o backlink é obrigatório.
- IF o reindex rodasse com o binário **sem** a tag `onnx` THEN os chunks
  voltariam a ser gravados sem vetor e o score perderia os 15 pontos — por
  isso o reindex usa o binário taggeado.
- IF a nota fosse escrita com BOM THEN o parser não reconheceria o
  frontmatter — a escrita é UTF-8 sem BOM (regra do `AGENTS.md` e do
  `compiler.WriteAtomicNote`).

## Requirement Traceability

| Requirement ID | Acceptance Criterion | Where |
| :--- | :--- | :--- |
| CCN-01 | AC1 — os 13 wikilinks resolvem para documento indexado | `cloud/*.md` · `database/*.md` · o índice |
| CCN-02 | AC2 — 0 link rompido no `repo_central` | `mem doctor --storage=postgres --repo repo_central` |
| CCN-03 | AC3 — nome do arquivo e `title` iguais ao wikilink | as 13 notas |
| CCN-04 | AC4 — frontmatter canônico e `# Título` no corpo | as 13 notas |
| CCN-05 | AC5 — backlink para o README da pasta e para a nota de origem | as 13 notas |
| CCN-06 | AC6 — entrada no README da pasta; `database/` ganha README | `cloud/README.md` · `database/README.md` |
| CCN-07 | AC7 — 0 chunk sem vetor real depois do reindex | o reindex com o binário taggeado |
| CCN-08 | AC8 — Health Score ≥ 90 | `mem doctor` |
| CCN-09 | AC9 — a nota diz o que é e onde entra | as 13 notas |
| CCN-10 | AC10 — ao menos uma relacionada por nota | as 13 notas |
| CCN-11 | AC11 — nenhuma nota nova órfã | o grafo |

**Coverage**: 11 total, 11 mapeados, 0 unmapped.

## Success Criteria

- [ ] O `mem doctor` do `repo_central` reporta **0 link rompido** (eram 13).
- [ ] O Health Score do `repo_central` fica **≥ 90** (está em 45).
- [ ] O cofre reindexado não tem chunk sem vetor real (eram 101).
- [ ] As 13 notas existem, com o nome do arquivo igual ao texto do wikilink, e
      cada uma aparece no README da sua pasta.
