# ADR-056: Persistência de propriedades de frontmatter em tabela consultável

- **Data**: 2026-10-18
- **Status**: Accepted
- **Decisores**: Felipe Miiller
- **Tags**: schema, frontmatter, propriedades, obsidian

## Contexto e Problema

O Obsidian trata propriedades de frontmatter como objeto de primeira classe, com quatro comandos dedicados: `properties`, `property:read`, `property:set`, `property:remove`.

O My-Memory faz o oposto. `Frontmatter.Properties` é um `map[string]any` populado em `internal/parser/frontmatter.go:86-88` e então **descartado**. Uma varredura por todos os consumidores em `internal/` encontra exatamente dois usos:

| Consumidor | Chave | Finalidade |
|---|---|---|
| `internal/compiler/compile.go:145-146` | `Properties["type"]` | inferir tipo de nota |
| `internal/compiler/note.go:337-338` | `Properties["type"]` | inferir tipo de nota |
| `internal/parser/wikilinks.go:283-305` | `Properties["relations"]` e chaves `IsKnownRelation` | emitir arestas tipadas |

Das seis chaves conhecidas, só três sobrevivem: `title` → `documents.title`, `category` → `documents.category`, `summary` → `documents.abstract` (renomeada). **`tags` e `aliases` são transientes e nunca persistidos.**

O gap mais difícil não é ler: é escrever. `property:set` exigiria reescrever o YAML dentro de `chunks.content`, que é o arquivo bruto fatiado em blocos de 200 palavras sobrepostos (`internal/parser/markdown.go:148-155`), sem garantia de alinhamento com a fronteira do frontmatter. Reescrever ali é destrutivo.

## Drivers da Decisão

- Tornar propriedades consultáveis, para `properties`, `property:read` e filtro em `mem search`
- Tornar `property:set`/`property:remove` possíveis **sem** editar texto chunkado
- Manter a taxonomia `category`/`summary` (ADR-025) funcionando como está
- Não acoplar ao schema — properties são arbitrárias por definição

## Opções Consideradas

- **A — Tabela `document_properties`, persistida em tempo de parse (escolhida)**
- **B — Coluna JSON blob em `documents`**
- **C — Manter em memória e filtrar pós-query**

## Resultado da Decisão

Escolhida a **Opção A**. `document_properties(document_id, key, value_type, value_text, created_at)`, com `UNIQUE(document_id, key)`, populada a partir do `Properties` já existente no parser. Uma migration nova, nos dois motores (`internal/db/schema.go` e `internal/store/postgres.go`).

A gravação passa por um caminho dedicado: `property:set` escreve na tabela **e** no arquivo em disco, na ordem **disco → índice**. O índice é derivado; o arquivo é a fonte. Se o disco falhar, nada é indexado. Isso evita a operação destrutiva de reescrever YAML dentro de chunk.

`value_type` distingue `string`, `number`, `bool`, `list`, `date` — o Obsidian tem tipagem de propriedade e nós hoje não temos nenhuma.

### Consequências Positivas

- `properties`, `property:read`, `property:set`, `property:remove` fecham o maior gap da matriz
- `aliases` ganha coluna própria, que é a menor mudança de schema da lista
- Filtros por propriedade em `mem search` viram-trivial (`EXISTS` + igualdade)
- `knowledge.base` e qualquer view de base passam a ter dado real para filtrar

### Consequências Negativas

- Migration em dois motores, com backfill de vaults já indexados
- Escrita bidirecional (disco + índice) é onde vai aparecer qualquer inconsistência; exige transação ou ordem explícita
- Properties arbitrárias crescem sem limite natural — precisa de política de cardinalidade

## Por Que Não as Outras

### Opção B — Coluna JSON blob ❌

- ✅ Uma coluna, migration menor
- ❌ Filtrar em SQL exige extração por chave em função do motor; Postgres diverge do SQLite
- ❌ Perde a tipagem que o Obsidian oferece nativamente

### Opção C — Filtrar pós-query ❌

- ✅ Zero schema
- ❌ Obliga a carregar todas as propriedades de todos os documentos para filtrar
- ❌ Torna `property:set` impossível sem editar chunked text

## Links

- [ADR-054](054-escopo-de-paridade-com-o-obsidian.md) — a paridade que cria a demanda
- [ADR-055](055-semantica-de-tag-consultavel-sem-nos-mortos.md) — muda o mesmo ponto de parse
- [ADR-025](025-progressive-context-loading-e-taxonomia-de-memoria.md) — taxonomia `category`/`summary`
- Evidência: `internal/parser/frontmatter.go:17,86-88`, `internal/db/store.go:102-108`, `internal/parser/markdown.go:148-155`