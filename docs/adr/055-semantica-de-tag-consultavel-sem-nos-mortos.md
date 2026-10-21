# ADR-055: Semântica de tag consultável sem nós mortos

- **Data**: 2026-10-18
- **Status**: Accepted (decidido em 2026-10-21; a pergunta em aberto foi resolvida — `resolved_doc_id` aponta para `documents.id`, com o título como rótulo)
- **Decisores**: Felipe Miiller
- **Tags**: grafo, tags, adr-041, semantica
- **Adenda**: [ADR-041](041-issue-009-tag-sem-casa-removida-do-grafo.md)

## Contexto e Problema

A ADR-041 resolveu um problema real: tags que não casavam com documento viravam nós mortos no grafo, inflando o Health Score e poluindo a travessia. A solução foi resolver por fuzzy match e **descartar o que não casasse** (`internal/parser/fuzzy.go:61-75`).

A consequência aceita na época — registrada na própria ADR-041 linha 63 — foi que **tag deixa de ser consultável como tag**. Hoje não existe `mem tags` nem `memory_get_tags`, e reconstrução via grafo é impossível:

| Comportamento | Evidência |
|---|---|
| Tag órfã é apagada na indexação | `fuzzy.go:67-75` compacta a aresta para fora |
| Tag sobrevivente é **renomeada** para o título do documento | `fuzzy.go:59-60` substitui o alvo |
| `documents` não tem coluna de tag | `internal/db/schema.go:5-14` |
| Tags só existem como aresta `tagged_as` | `internal/parser/wikilinks.go:277-280` |

O Obsidian lista **todas** as tags, inclusive órfãs. Logo, `mem tags` literal seria incompatível com o grafo atual — não é tarefa de exposição, é mudança de modelo.

## Drivers da Decisão

- `mem tags` e `mem search --tag` devem existir; a ADR-041 já prometeu `mem search --tag X` como DR-5 e nunca foi implementado
- Não reintroduzir nós mortos no grafo (o problema original da ADR-041)
- Permitir reconciliação com o Obsidian sobre o mesmo vault
- Não duplicar dados: a aresta `tagged_as` continua sendo a projeção do grafo

## Opções Consideradas

- **A — Tabela `document_tags` populada antes da resolução fuzzy (escolhida)**
- **B — Redefinir "tag" no My-Memory como "nota que é alvo de tag"**
- **C — Manter o descarte e nunca expor tags**

## Resultado da Decisão

Escolhida a **Opção A**. Uma tabela `document_tags(document_id, tag, resolved_doc_id, created_at)` é populada no `internal/watcher/indexer.go`, **antes** do passo fuzzy. O grafo continua recebendo apenas arestas resolvidas — ADR-041 intacta. A tabela serve de fonte para `mem tags` e para o filtro de `mem search`.

O mapeamento passa a ser Many-to-One: uma tag literal pode apontar para zero, um ou vários documentos resolvidos, e a aresta do grafo é uma projeção desse vínculo, não a fonte da verdade.

### Consequências Positivas

- `mem tags` é compatível com o Obsidian, incluindo tags órfãs
- `mem search --tag X` passa a filtrar por tag resolvida, não por substring
- A ADR-041 não precisa ser revertida: o problema dela (nó morto) continua resolvido
- Reconciliação com o Obsidian passa a ser possível

### Consequências Negativas

- Nova migration no schema, no SQLite **e** no Postgres (`internal/store/postgres.go`)
- Uma tag aparece em duas consultas — `documents` e `graph_edges` — e exige cuidado para não divergirem
- `resolved_doc_id` é uma referência frouxa: renomear um documento deixa a tag órfã de novo, exigindo reindexação para reconciliar

## Por Que Não as Outras

### Opção B — Redefinir "tag" ❌

- ✅ Zero mudança de schema
- ❌ `mem tags` passa a listar títulos de documentos, o que não é uma lista de tags — confunde o usuário e quebra reconciliação
- ❌ Torna a ADR-041 ambígua: "tag" passa a significar duas coisas

### Opção C — Nunca expor ❌

- ✅ Status quo, zero risco
- ❌ Mantém uma promessa formal não cumprida (DR-5 da ADR-041) e o gap `#20` da matriz

## Pergunta Resolvida

**A coluna `resolved_doc_id` aponta para `documents.id` (hash estável), com o título
guardado como rótulo.** Decidido em 2026-10-21.

Renomear o documento deixa a tag órfã até a próxima reindexação, que é o
reconciliador. A alternativa — apontar para o título resolvido — quebraria o
vínculo silenciosamente: o título muda com o frontmatter, colide entre pastas e
não é uma chave. `documents.id` é a única referência estável do schema, e é o
que permite consultar por tag sem depender do texto.

## Links

- [ADR-041](041-issue-009-tag-sem-casa-removida-do-grafo.md) — esta ADR é uma adenda, não uma reversão
- [ADR-054](054-escopo-de-paridade-com-o-obsidian.md) — define o escopo que cria a demanda
- [ADR-011](011-arestas-epistemicas-e-god-nodes.md) — pesos e relações do grafo
- Evidência: `internal/parser/fuzzy.go:59-94`, `internal/watcher/indexer.go:100-103`