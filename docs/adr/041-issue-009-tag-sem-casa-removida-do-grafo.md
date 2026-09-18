---
title: "ADR-041: Tags sem doc-casa são removidas do grafo (não viram dead links)"
category: resource
summary: "Decisão arquitetural sobre ISSUE-009: quando uma tag do frontmatter (ou inline #tag) não casa com nenhum doc via fuzzy resolve, a edge tagged_as é REMOVIDA do grafo. Tags conceituais ficam no frontmatter (search via FTS funciona) mas não viram nós do grafo."
tags: [architecture, parser, graph, dead-links, tag-resolution]
---

# ADR-041: Tags sem doc-casa são removidas do grafo (não viram dead links)

- **Date**: 2026-09-18
- **Status**: Accepted (implementado em `4c19393`)
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: architecture, parser, graph, dead-links, tag-resolution

## Context and Problem Statement

O parser do my-memory trata QUALQUER valor de tag (frontmatter `tags: [...]` ou inline `#tag` no body) como nó implícito do grafo, criando uma edge `tagged_as` do doc para esse nó. Se nenhum doc existente tem esse exato título, a edge vira dead link no relatório `mem doctor`.

O ISSUE-001 limpou 8 tags específicas (`algorithms`, `internal`, `howto`, `commands`, `flags`, `Tags` em ADR-005, etc.), mas o problema é estrutural: cada novo doc criado com tags descritivas (ex: `architecture`, `storage`, `federation`, `sqlite`, `postgres`) gera novas dead links potenciais. ISSUE-009 foi aberto para rastrear o systemic pattern.

**Estado pré-fix (sessão 2026-09-18):**
- Após ISSUES-001/008 + ADR-040 implementação: 19 dead links no `mem doctor` Health Score 33/100
- Pattern observado: ADR-040 com 8 tags genéricas → 8 dead links; spec.md com 1 self-reference → 1 dead link

## Decision Drivers

- **DR-1**: Cada nova tag genérica não deveria exigir limpeza manual (princípio DRY)
- **DR-2**: Tags conceituais (categorias, não-alvo-de-link) não devem poluir o grafo
- **DR-3**: Search via FTS deve continuar funcionando (frontmatter fica no chunk text)
- **DR-4**: `mem doctor` Health Score deve refletir utilidade real, não ruído
- **DR-5**: Comportamento deve ser conservador — não quebrar features existentes (search, `mem search --tag X` se existir)

## Considered Options

1. **Não criar edge `tagged_as` quando tag não tem casa** ← Chosen
2. Aceitar como débito cosmético — sem mudança de código
3. Renomear nós de tag para `tag:<valor>` (namespacing explícito) — distingue tag-conceito de doc-title
4. Híbrido: criar edge mas marcar `target_kind=tag` (cosmético, não flagado como dead)

## Decision Outcome

Chosen option: **"1 — Não criar edge `tagged_as` quando tag não tem casa via fuzzy resolve"**, porque atende DR-1, DR-2 e DR-4 sem custo de refactor arquitetural (sem mudanças de schema, sem namespace, sem mudanças de doctor SQL).

**Implementação concreta:**

- `internal/parser/fuzzy.go::ResolveTagConnections`: quando `fuzzyResolveTag` retorna `""` (sem match) para uma edge `tagged_as`, marca o `Target` como vazio. Pós-iter, compacta `conn.Edges` removendo entradas com target vazio, e sincroniza `conn.OutgoingLinks` para não apontar para targets removidos.
- Para `links_to` (wikilinks), comportamento existente é preservado (target original mantido se sem match — wikilinks quebrados são dead links conhecidos e o usuário tem controle editorial direto).

**Trade-offs aceitos:**

- Tags conceituais sem doc-casa (ex: ADR-040 com `architecture`, `federation`) deixam de existir como nós do grafo. Search via FTS continua funcionando porque a tag permanece no frontmatter (indexado como chunk text).
- Edge `tagged_as(spec.md, "spec")` é rewriteada para o doc-id ou basename correspondente via fuzzy. Se não houver match, é removida.

### Positive Consequences

- **Health Score realista**: `mem doctor` reflete utilidade, não ruído. Pós-fix: 19 dead links → 0-2 (depende de quantas tags conceituais ainda existem)
- **Zero manutenção manual**: novos docs com tags genéricas não exigem cleanup
- **Compatibilidade total**: search via FTS funciona idêntico; tags permanecem pesquisáveis via conteúdo do chunk
- **Edge cases preservados**: `links_to` continua mostrando dead links quando aplicável (wiki-links quebrados)

### Negative Consequences

- **Perda de "tag-as-node" semantic**: quem usava query `SELECT * FROM graph_edges WHERE relation = 'tagged_as'` pra listar tags conceituais perde isso
- **Stub docs podem ser necessários**: se o usuário QUER uma tag conceitual como nó do grafo (ex: pra grafo de comunidade), precisa criar nota stub com o nome da tag como title
- **Debugging mais sutil**: tags removidas não aparecem em nenhum relatório — usuário pode se perguntar "por que minha tag não aparece no grafo?"

## Pros and Cons of the Options

### Option 1 — Não criar edge sem casa ✅ Chosen

- ✅ Atende DR-1 (zero manutenção) e DR-2 (não polui grafo)
- ✅ Health Score reflete realidade
- ✅ Compatível com search via FTS
- ✅ Implementação isolada no parser, sem mudança de schema
- ❌ Tags conceituais "somem" do grafo (mitigável com stub doc)

### Option 2 — Aceitar como débito

- ✅ Zero código
- ❌ Health Score persistentemente distorcido (drift detector reportando falso positivo, etc.)
- ❌ Cada nova doc exige limpeza manual (viola DR-1)

### Option 3 — Namespace `tag:<valor>`

- ✅ Distingue explicitamente tag-conceito de doc-title
- ✅ Grafo fica completo (nada removido)
- ❌ Requer mudança de schema SQL (`graph_nodes.type` distingue 'note' de 'tag')
- ❌ Requer mudança no `doctor` SQL pra não flagar 'tag' como dead
- ❌ Requer mudança em `mem search` pra interpretar tags (decisão arquitetural separada)

### Option 4 — Híbrido com `target_kind`

- ✅ Compromise: tags ficam mas não flagadas
- ❌ Adiciona coluna nova em `graph_edges` (mudança de schema)
- ❌ Não resolve o ruído: o grafo continua inflado com nós `tag:*`

## Validation

- **Pré-fix**: `mem doctor --db .memory/memory.db` reporta 19 dead links, Health Score 33/100
- **Pós-fix** (commit a ser registrado): dead links reduzidos a 0-2 dependendo do estado do vault
- **Testes**: `TestResolveTagConnections/ISSUE-009: no match removes edge` + `ISSUE-009: tag sem casa é removida` + `ISSUE-009: mistura — tag com casa resolvida, tag sem casa removida` (todos passam)
- **Compatibilidade**: `mem search <query>` continua funcionando — tags pesquisáveis via FTS chunk content
- **Regressão**: nenhum teste quebrado (19/19 pacotes OK)

## Migration Path para Casos Edge

Se um usuário QUISER uma tag conceitual como nó do grafo, cria stub:

```bash
# Cria nota stub com nome da tag como title
echo "# architecture" > docs/concepts/architecture.md
# Frontmatter com title: "architecture" (vira nó do grafo via documents.title)
# Agora tagged_as aponta pra esse nó
```

## Links

- **ISSUE-009** — issue que originou a decisão
- **ISSUE-001** — limpeza anterior de 8 tags específicas (mesma direção, escopo menor)
- **ADR-040** — context de indexação/storage (não relacionada à decisão de parser, mas usou `mem doctor` extensivamente)
- **ae8547b** — commit que introduziu `ResolveTagConnections` (base do fix)
- **fuzzy.go** — `internal/parser/fuzzy.go` (arquivo modificado pelo fix)
- **mermaid graph** — `docs/adr/041-issue-009-tag-sem-casa-removida-do-grafo.md` (este arquivo)

---

**Supersedes**: nenhum (decisão nova)
**Status note**: Accepted em 2026-09-18. Implementação: `4c19393` (`fix(parser): ISSUE-009 — tags sem casa removidas do grafo`). Spec detalhada em `.specs/041-issue-009-tag-resolution/`.
