# ADR-064: Todo conceito linkado no cofre central tem nota

- **Date**: 2026-11-14
- **Status**: Accepted — Implementado
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: cofre-central, vault, wikilinks, moc, conteudo, adr-062
- **Adenda**: [ADR-062](062-contrato-dos-dois-cofres-e-o-que-o-ai-memory-ensina.md) (o contrato dos dois cofres) · [ADR-063](063-o-doctor-distingue-link-rompido-de-link-fora-do-indice.md) (as três categorias de link)
- **Spec**: `.specs/features/central-vault-concept-notes/` · **Vault**: `G:\My Drive\central-memory`

## Context and Problem Statement

Duas notas do Cofre Central terminam com uma seção **"Notas relacionadas no
Obsidian"**, linkando conceitos que o autor considera relacionados:

- `cloud/blue-green-deployment-obsidian.md` → Docker, Kubernetes, Observabilidade, Rollback, Expand and Contract, SRE e SLOs (entre outros)
- `database/PostgreSQL — Visão Geral de Recursos e Ferramentas.md` → Banco de Dados, Índices SQL, Busca Vetorial e RAG, Filas e Processamento Assíncrono, Transações e Concorrência, Observabilidade de Banco de Dados, Backup e Recuperação

**13 desses conceitos não existiam em lugar nenhum**: não no índice, não em disco
dentro do escopo, não no material bruto do catálogo. A última linha da própria
nota do PostgreSQL diz o que fazer: *"crie uma nota chamada `Nome da nota`"*.

Enquanto isso, o `mem doctor` contava esses 13 como **links rompidos** — e eles
eram rompidos de verdade. A penalidade (5 pontos cada, teto 40) saturava o
Health Score do cofre em **45/100**, e o critério de prontidão **"doctor ≥
90/100"** da ADR-062 §B-2 ficava a um único passo de fechar.

O detalhe que torna a nota mais que um documento: **o wikilink já resolve no
Obsidian** quando a nota existe. A nota ausente não é um documento faltando —
é um **link quebrado** na ferramenta que o dono usa todo dia.

## Decision Drivers

- **DR-1 — Um link é uma promessa.** Um `[[Docker]]` que o Obsidian não
  resolve é um link quebrado, mesmo que o grafo do índice o considere "fora do
  índice por decisão".
- **DR-2 — A identidade da nota é o arquivo.** No Obsidian, `[[Docker]]`
  resolve para `Docker.md`; o `title` do frontmatter é o que o índice usa.
  Nome e título têm de concordar, senão a resolução vira fuzzy.
- **DR-3 — Nota vazia é ruído indexado.** O README do próprio cofre é
  explícito: *"Um documento que não diz nada ainda assim disputa relevância com
  documentos reais"*. A nota nova tem que dizer o que é e onde entra.
- **DR-4 — A estrutura é navegável, não implícita.** A regra do MOC do cofre
  (`standards/navegacao-moc.md`) exige README em toda pasta e backlink de cada
  nota ao índice da sua pasta.
- **DR-5 — O padrão é do Obsidian, não inventado.** O formato vem da
  documentação oficial (`obsidian.md/help/properties`, `obsidian.md/help/links`),
  não de uma convenção local.

## Considered Options

- **A — Criar a nota que falta, uma por conceito (escolhida)**
- **B — Remover o wikilink do texto da nota que linka**
- **C — Deixar como estava e aceitar o score abaixo de 90**

## Decision Outcome

Escolhida a **Opção A**. A regra que passa a valer no Cofre Central:

> **Todo conceito que uma nota do cofre linka tem uma nota.** Se o conceito
> não existe ainda, ele é criado — com o nome do arquivo igual ao texto do
> wikilink.

O formato, no padrão do Obsidian e na convenção do cofre:

| Peça | Forma | Porquê |
| :--- | :--- | :--- |
| Arquivo | `Docker.md`, `SRE e SLOs.md`, `Backup e Recuperação.md` | a identidade de uma nota no Obsidian é o arquivo; `[[Docker]]` resolve sem configuração (`obsidian.md/help/links`) |
| `title` | **exatamente** o conceito | o índice resolve `links_to` contra `documents.title`; o match exato evita a resolução aproximada, que pode casar com outro documento |
| `category` | `resource` | taxonomia do vault (ADR-025) |
| `summary` | 1–2 frases | o micro-abstract L0 do progressive context loading |
| `tags` | lista, sem `#` | propriedade padrão do Obsidian |
| `status` | `ativo` | convenção do cofre |
| Corpo | `# <título>` + "O que é" + "Onde entra aqui" + "Relacionado" | DR-3 e DR-4 |
| Backlinks | `[[<pasta>/README]]` e `[[<nota que linka>]]` | DR-4; o caminho da pasta é a forma que a documentação do Obsidian prescreve para link dentro de pasta |

O **nome do arquivo** é o texto do wikilink — não um slug. É o que a própria
nota do PostgreSQL no cofre instrui, e o cofre já tem notas com espaço no
nome (`performance/CLS - Causas e Correcoes de Layout Shift.md`).

### Positive Consequences

- Os 13 wikilinks passam a resolver **no Obsidian** e no índice: o `mem doctor`
  do `repo_central` vai a **0 link rompido**.
- O critério de prontidão §B-2 da ADR-062 fecha: o Health Score do cofre sai de
  45/100 para **≥ 90** (medido: **100/100**).
- A seção "Notas relacionadas no Obsidian" deixa de ser uma lista de links
  quebrados e vira o que pretendia: um índice navegável.
- A regra é verificável: um link sem nota é um link rompido, e o doctor
  denuncia.

### Negative Consequences

- **Custo de manutenção**: linkar um conceito passa a ter custo de criação de
  documento. A alternativa (não linkar) continua disponível — mas então não
  há link.
- **Nota stub compete por relevância**: 13 documentos novos entram no índice.
  Mitigação: cada um tem conteúdo mínimo útil e está ligado ao contexto que o
  invocou (nenhum fica órfão).
- **A pasta `database/` passou a ter README**: ela já tinha conteúdo (a nota do
  PostgreSQL) e estava em desacordo com a regra 1 do MOC. O índice da pasta
  `cloud/` também não listava a nota blue-green — corrigido na mesma rodada.

## Pros and Cons of the Options

### Opção A ✅ Chosen

- ✅ Fecha o critério §B-2 e o link deixa de estar quebrado no Obsidian.
- ✅ Preserva a intenção do autor da nota: a seção "relacionadas" funciona.
- ❌ Exige 13 documentos novos e a manutenção deles.

### Opção B — remover o wikilink

- ✅ Nenhum documento novo; o score também sobe.
- ❌ Perde a informação: o leitor perde o caminho para o conceito e o autor da
  nota perde a intenção.
- ❌ Troca um link quebrado por uma ausência — e a ausência não é consultável.

### Opção C — deixar como está

- ✅ Zero trabalho.
- ❌ Mantém 13 links quebrados no Obsidian e o cofre abaixo do critério de
  prontidão.

## Links

- [ADR-062](062-contrato-dos-dois-cofres-e-o-que-o-ai-memory-ensina.md) — §B-2,
  o critério "doctor ≥ 90/100" que esta ADR fecha.
- [ADR-063](063-o-doctor-distingue-link-rompido-de-link-fora-do-indice.md) — as
  três categorias que separaram "fora do índice por decisão" de "rompido".
- [ADR-025](025-progressive-context-loading-e-taxonomia-de-memoria.md) — a
  taxonomia `category`/`summary` que o frontmatter alimenta.
- `standards/navegacao-moc.md` (no Cofre Central) — a regra do MOC: README por
  pasta e backlink de cada nota ao índice da pasta.
- Documentação do Obsidian: [Properties](https://help.obsidian.md/properties) ·
  [Internal links](https://help.obsidian.md/links).
- Spec: `.specs/features/central-vault-concept-notes/`.
