---
name: obsidian-markdown
description: Regras canônicas para criar e editar notas no formato Obsidian Flavored Markdown (.md) com wikilinks, embeds, callouts, tags e propriedades de frontmatter YAML. Esta é a **referência canônica** para qualquer agente que manipula documentos .md no vault do My-Memory. Aplique em ADRs, specs, ISSUES, skills, READMEs e qualquer nota que será indexada pelo parser.
---

# Obsidian Flavored Markdown Skill — Referência Canônica

Esta habilidade é a **referência canônica** do projeto My-Memory para formatação de documentos Markdown. Qualquer agente que criar ou editar `.md` deve seguir estas regras para garantir:

1. **Compatibilidade com o parser**: tags, wikilinks e frontmatter são parseados corretamente
2. **Integridade do grafo**: tags viram nós, wikilinks viram arestas — sem ruído
3. **Busca via FTS**: o chunk text permanece pesquisável mesmo após mudanças estruturais
4. **Consistência visual**: wikilinks renderizam como links internos no Obsidian/VS Code

> [!important]
> **Esta skill é a fonte de verdade.** Outras skills (create-adr, tlc-spec-driven, obsidian, etc.) referenciam esta. Se houver conflito entre uma skill específica e esta referência, esta ganha.

---

## 1. Estrutura Padrão de uma Nota

Toda nota `.md` (exceto ADRs, que seguem convenção própria) deve conter um bloco inicial de **frontmatter YAML**:

```markdown
---
title: "Nome da Nota (camel case ou kebab-case)"
category: resource      # resource | memory | skill — opcional mas recomendado em vaults
summary: "Resumo de 1-2 frases. Importante para L0 do progressive context loading."
tags:
  - categoria-principal
  - subcategoria
  - tag-específica
aliases:
  - Sinônimo 1
  - Sinônimo 2
status: ativo            # opcional: ativo | deprecated | draft
---

# Nome da Nota

Conteúdo em Markdown com conexões explícitas (wikilinks, tags inline, callouts).
```

**Campos obrigatórios:**

| Campo | Tipo | Obrigatório | Notas |
|---|---|---|---|
| `title` | string | ✅ | Mesmo critério que `db.ListDocumentTitles` consulta |
| `tags` | lista | ✅ para notas do vault | Lista simples, sem `#`. Inline `#tag` no body é separado |
| `summary` | string | 🟡 recomendado | Usado pelo L0 micro-abstract no `mem search` |
| `category` | enum | 🟡 em vault | `resource`, `memory` ou `skill` (ver ADR-024) |
| `aliases` | lista | opcional | Para resolver wikilinks com nomes alternativos |
| `status` | enum | opcional | Para filtros futuros |

> [!warning]
> **Tags não devem ser genéricas sem nota-casa.** Se uma tag conceitual (`architecture`, `federation`) deve existir como nó do grafo, crie um stub doc com `title: "<tag>"`. Tags órfãs viram dead links (ISSUE-009 → ADR-041 fix em `4c19393`).

---

## 2. Wikilinks (Conexões de Grafo)

Use **sempre** wikilinks `[[...]]` para referências internas (não `[texto](path.md)` quando o destino é nota do vault):

| Sintaxe | Função |
|---|---|
| `[[Nome da Nota]]` | Cria aresta `links_to` no grafo para a nota de destino |
| `[[Nome da Nota\|Texto Exibido]]` | Aresta preservando rótulo customizado |
| `[[Nome da Nota#Seção]]` | Vincula a seção específica |
| `[[Nome da Nota#^bloco-id]]` | Block reference (parágrafo específico) |
| `[[#Seção Local]]` | Navegação interna (não cria aresta) |
| `![[Nota ou Anexo.png]]` | Transclusão / embed visual inline |
| `[[Nome da Nota\|alias]]` | Mostra "alias" mas linka pra "Nome da Nota" |

**Regras importantes:**

- **Resolução case-insensitive**: `[[ADR-040]]` e `[[adr-040]]` batem pro mesmo doc
- **Resolução fuzzy**: parser tenta match exato → substring → token (ver `internal/parser/fuzzy.go`)
- **Não usar `[texto](./path.md)` para refs internas** — perde a aresta do grafo
- **Exceção — ADRs**: usam markdown links `[ADR-NNN](./path)` por convenção histórica do projeto (ver ADR-001..040)

> [!example]
> **Certo:** `Veja [[ADR-040]] para centralização de config.`
> **Errado:** `Veja [ADR-040](./040-config-global-unica-storage-opt-in.md) para centralização de config.`

---

## 3. Tags

### 3.1 Frontmatter (lista sem `#`)

```yaml
---
tags:
  - cli
  - skill
  - howto
---
```

- Tags em YAML **NÃO** levam `#`. Apenas inline.
- Tags devem ser `lowercase-kebab-case` ou `lowercase-snake_case`.
- Tags devem casar com nota-casa OU ser removidas do grafo (ISSUE-009 fix).

### 3.2 Inline (`#tag` no corpo)

```markdown
Este documento cobre #fuzzy-matching e #parser no contexto do vault.
```

- Use `#tag` no corpo quando a tag é contextual ao parágrafo.
- Regex: `(?:^|[\s\(\[\{,;:])#([a-zA-Z][a-zA-Z0-9_\-\/]*)` — começa com letra, alphanumeric + `-_/`.

> [!warning]
> **Syntax examples em inline code.** Quando documentar sintaxe (ex: "a tag `#foo` parseia como..."), wrappear em inline code: `` `#foo` ``. Sem isso, o parser captura o `#foo` como tag de verdade.

---

## 4. Callouts Padronizados

Use callouts para destacar pontos críticos:

```markdown
> [!note]
> Nota explicativa de contexto.

> [!tip]
> Sugestão ou recomendação técnica.

> [!important]
> Requisito crítico ou restrição mandatória.

> [!warning]
> Alerta sobre efeitos colaterais ou quebras de compatibilidade.

> [!example]
> Exemplo concreto de uso.

> [!question]
> Pergunta em aberto (decisão pendente).
```

Tipos disponíveis (Obsidian v1.4+): `note`, `abstract`, `info`, `todo`, `tip`, `success`, `question`, `warning`, `failure`, `danger`, `bug`, `example`, `quote`.

---

## 5. Embeds e Transclusões

```markdown
![[outra-nota]]              # Embed integral da nota
![[outra-nota#Seção]]         # Embed de uma seção específica
![[imagem.png]]               # Embed visual
![[imagem.png|legenda]]       # Embed com legenda
```

**Atenção:** para embeds funcionarem, a nota/imagem referenciada precisa existir no vault. Não use embed pra "criar link pra existir depois" — use wikilink comum.

---

## 6. Anti-Patterns (EVITAR)

### 6.1 Sintaxe de sintaxe (sem inline code)

```markdown
# ERRADO — parser vai capturar como tag real:
Use #foo para marcar uma tag. A regex é `^#[a-zA-Z]`.

# CERTO — wrappado em inline code:
Use `#foo` para marcar uma tag. A regex é `^#[a-zA-Z]`.
```

### 6.2 Markdown table rows com wikilinks

Wikilinks dentro de tabelas Markdown podem ser parseados incorretamente pelo regex. Use **inline code** para wikilinks em tabelas OU reescreva sem `[[...]]`:

```markdown
# ERRADO — wikilink dentro de tabela:
| Origem | Tag |
|---|---|
| COMO_FUNCIONA | [[algorithms]] |

# CERTO — código inline OU texto simples:
| Origem | Tag |
|---|---|
| COMO_FUNCIONA | `[[algorithms]]` (tag removida por ISSUE-009) |
```

### 6.3 Tags genéricas sem nota-casa

```markdown
# ERRADO — `architecture` viraria dead link até ADR-041 fix:
tags: [architecture, federation, storage]

# CERTO — tag específica com casa, ou stub doc criado:
tags: [adr-040, sqlite-auto-scope, central-vault]
# OU criar docs/concepts/architecture.md com title: "architecture"
```

### 6.4 `[texto](./path.md)` para refs internas

```markdown
# ERRADO — perde aresta do grafo:
Ver [AGENTS.md](../AGENTS.md) para regras.

# CERTO — wikilink cria aresta:
Ver [[AGENTS]] para regras.
```

---

## 7. Checklist de Validação Pós-Edição

Após editar qualquer `.md` em vault indexado pelo My-Memory:

- [ ] `mem doctor --db .memory/memory.db` reporta 0 dead links (ou só esperados por docs órfãos conhecidos)
- [ ] `mem index` re-indexa sem erro (se cache SHA-256 estiver OK, deve ser cached; senão, use `--force`)
- [ ] Grep próprio: `grep -rn '\[\[' <arquivo>` para confirmar que wikilinks estão em formato `[[X]]` e não em tabelas
- [ ] Frontmatter tem `title` + `tags` (e `category` em vault)
- [ ] Tags conceituais têm doc-casa (criar stub se necessário)

---

## 8. Convenções por Tipo de Artefato

| Tipo | Padrão de link | Frontmatter especial | Onde mora |
|---|---|---|---|
| ADR (`docs/adr/NNN-*.md`) | Markdown `[ADR-NNN](./path)` (estabelecido) | `status`, `deciders`, `date` | `docs/adr/` |
| Spec (`.specs/NNN/spec.md`) | Wikilinks `[[ADR-NNN]]` | `status: in-progress` | `.specs/NNN-*/` |
| ISSUES | Tabelas + `🔴 🟠 🟡 🟢` emoji para severidade | `status: open/resolved` | `.specs/ISSUES.md` |
| Skill (`SKILL.md`) | Wikilinks pra outros skills | `name`, `description` | `.agents/skills/<skill>/SKILL.md` |
| README | Wikilinks pra docs internos | mínimo | raiz do repo |

---

## 9. Referências Cruzadas

- **`create-adr`** — referencia esta skill para formatação de frontmatter
- **`tlc-spec-driven`** — workflow 4-fases, adota este skill para `spec.md` / `tasks.md` / `validation.md`
- **`obsidian`** — vault operations (mais alto nível)
- **ADR-040** — `~/.memory/config.yaml` global (storage)
- **ADR-041** — tags sem casa removidas do grafo (decisão que motivou §6.3)

---

**Mantido por**: Mavis (orchestrator) + Felipe Miiller (review).
**Última atualização**: 2026-09-18.
