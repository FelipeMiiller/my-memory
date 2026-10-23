---
title: "Spec — Paridade de tags com o Obsidian"
category: resource
summary: "Superficie de tags do Obsidian no My-Memory: `mem tags`, `mem tag <nome>` e `mem search --tag`, mais as tools MCP memory_get_tags e memory_get_tag. Implementa a ADR-055 (resolved_doc_id aponta para documents.id) e destrava a promessa DR-5 da ADR-041."
tags: [spec, tags, obsidian, adr-055, adr-041, mcp, cli]
---

# Spec: Paridade de tags com o Obsidian

- **ADR**: [ADR-055](../../../docs/adr/055-semantica-de-tag-consultavel-sem-nos-mortos.md) (Aceito em 2026-10-21) · [ADR-054](../../../docs/adr/054-escopo-de-paridade-com-o-obsidian.md)
- **Branch**: `develop`
- **Data**: 2026-10-22

---

## Problem Statement

O My-Memory se declarou superset do Obsidian (ADR-054), mas a superfície de **tags**
não existe. Hoje o My-Memory faz o oposto do Obsidian: **descarta** as tags.

| Comportamento | Obsidian | My-Memory hoje |
|---|---|---|
| Lista as tags do vault | `obsidian tags` | **não existe** |
| Consulta uma tag | `obsidian tag name=X` | **não existe** |
| Filtra a busca por tag | `obsidian search tag:X` | **não existe** — a ADR-041 prometeu `mem search --tag X` como DR-5 e nunca entregou |
| Tag órfã (sem nota que a nomeie) | aparece na lista de tags | **descartada na indexação** (ADR-041, ISSUE-009) |
| Tag hierárquica `#a/b` | suportada | o parser aceita, mas a tag morre no mesmo passo |
| `#Tag` e `#TAG` | a mesma tag (case-insensitive) | **duas tags distintas** — o dedup do parser é case-sensitive |
| `#1a`, `#ação`, `#🚀` | tags válidas | **não capturadas** — o regex do parser só aceita ASCII iniciando em letra |
| `#1984` (só numérica) | **tag inválida** | idem (por acidente: o regex exige letra inicial) |

A perda é total e silenciosa. A ADR-041 resolveu um problema real — tag que não
casava com documento virava nó morto no grafo e derrubava o Health Score — mas o
preço foi tornar a tag **não consultável**. A ADR-055 registrou a conta e a
decidiu: uma tabela `document_tags` populada **antes** do passo fuzzy, com o
grafo continuando recebendo só as arestas resolvidas.

Sem essa tabela, a tag é extraída (`internal/parser/wikilinks.go:235-244`) e
deixada no caminho (`conn.Tags`), e o `ResolveTagConnections`
(`internal/parser/fuzzy.go:34`) remove a aresta de quem não casou. O dado
existe por milissegundos e morre.

### Fonte primária (norma adotada)

As regras de tag vêm de `obsidian.md/help/tags`, `obsidian.md/help/properties` e
`obsidian.md/help/cli`. As regras que valem para esta feature:

- **Formato**: letras, números, `_`, `-`, `/` (hierárquica) e Unicode/emoji.
- **Ao menos um caractere não-numérico**: `#1984` é inválida, `#y1984` é válida.
- **Sem espaços** em branco.
- **Case-insensitive**: `#tag` e `#TAG` são a mesma tag.
- **Exibição** com o casing da primeira criação.
- **Hierarquia**: `tag:inbox` casa `#inbox` **e** `#inbox/to-read`.
- **Propriedade `tags`**: lista YAML (`- tag`); `tag` é alias depreciado.
- **Hashtag em propriedade de texto** não cria tag.

---

## Goals

- `mem tags` e `mem tag <nome>` com a mesma semântica do CLI oficial do Obsidian
- `mem search --tag <tag>` — a promessa DR-5 da ADR-041
- As tools MCP `memory_get_tags` e `memory_get_tag`
- A tabela `document_tags` populada no indexer, com `resolved_doc_id` apontando
  para `documents.id` (decisão da ADR-055)
- O **formato** da tagindexada seguindo a norma do Obsidian (caracteres,
  validação, case-insensitive, hierarquia)

---

## Out of Scope

| Item | Motivo |
|---|---|
| Tags como **nó do grafo** | A ADR-041 eliminou isso de propósito: nó morto polui a travessia e infla o Health Score. A tag fica **na tabela**, e a aresta do grafo continua sendo a projeção do vínculo resolvido |
| Editing de tags (`obsidian tag:create`) | O My-Memory é painel de métricas, não editor (ADR-059) |
| Tags em `.base` / `properties` | Pertencem à ADR-056, que tem escopo próprio |
| CLI no estilo `key=value` do Obsidian (`sort=count`) | A CLI do My-Memory usa flags `--` (Go `flag`). O **significado** das opções é o mesmo; a sintaxe acompanha a casa, não a do Obsidian |
| `obsidian tags active` (arquivo ativo) | O My-Memory é headless — não existe "arquivo ativo" |
| `format=tsv\|csv` | O My-Memory usa saída de texto e `--json`; o CSV/TSV é o formato padrão do Obsidian e não acrescenta capacidade |
| Migração de tags de vaults já indexados | A tabela é populada na indexação; um vault antigo precisa de `mem index` para ter as tags. Documentado, não migrado |
| Tags dentro de arquivos não-Markdown (PDF, canvas) | O indexador só trata `.md` nesta feature |

---

## Assumptions & Open Questions

**Assumptions**

- A.1 — `documents.id` é o **path do arquivo** no vault (ver `InsertDocumentWithMeta`
  em `internal/watcher/indexer.go:98`), e é o que `resolved_doc_id` referencia.
- A.2 — O título usado na resolução fuzzy é `COALESCE(title, id)`
  (`internal/db/store.go:811`), então um documento sem `title:` no frontmatter é
  resolvido pelo basename.
- A.3 — A semântica de tag hierárquica do Obsidian é: `obsidian tag name=project`
  devolve as notas com `project` **e** com `project/alpha` (descendentes).
- A.4 — `conn.Tags` **sobrevive** ao `ResolveTagConnections` — o fuzzy só mexe em
  `conn.Edges` e `conn.OutgoingLinks` (`internal/parser/fuzzy.go:52-83`). Logo a
  lista completa (órfãs incluídas) está disponível no indexer.
- A.5 — `obsidian tags counts` conta as notas que carregam a tag **exata**;
  a contagem com descendentes é do `obsidian tag`. As duas referências ficam
  distintas, e a diferença é documentada.
- A.6 — O inventário **inclui** as tags órfãs: é o que o Obsidian lista, e é o
  que a ADR-055 comprou ao popular a tabela antes do passo fuzzy. O grafo
  continua ignorando-as — a ADR-041 não foi revertida.
- A.7 — `resolved_doc_id` de uma tag que nomeia **vários** documentos fica com o
  primeiro (ordem de título, a mesma do fuzzy) e o vínculo é Many-to-One pela
  aresta do grafo. Decidido na ADR-055.
- A.8 — **Case-insensitive com casing de exibição.** A tag é gravada em duas
  formas: `tag` (exibição, o casing da primeira criação) e `tag_key` (canônica,
  minúscula). A chave de identidade é `tag_key`; a deduplicação e a comparação
  são feitas nela. É o que a norma do Obsidian exige ("`#tag` e `#TAG` são a
  mesma tag", exibidas "com o casing com que foram criadas primeiro").
- A.9 — **Descendentes contam em toda a superfície.** `mem tag <nome>` e
  `mem search --tag <nome>` casam a tag e as descendentes, igual ao
  `tag:inbox` do Obsidian, que casa `#inbox` e `#inbox/to-read`. O inventário
  (`mem tags`) lista as tags **literais**, sem expandi-las.
- A.10 — **O filtro por tag é push-down**, no mesmo ponto e com a mesma
  semântica do filtro `--category`. Ele **não** é um filtro pós-fusão: o ponto
  pós-fusão (`internal/store/rrf.go:165-174`) roda **antes** do rank recíproco e
  mudaria a ranqueação relativa, e `SearchResult` não carrega tags para saber
  filtrar.
- A.11 — **A resolução é gravada na indexação.** `resolved_doc_id` é calculado
  quando o documento é indexado, contra os títulos conhecidos naquele instante.
  Um documento novo que case com uma tag já indexada **não** retro-atualiza as
  linhas antigas. Isso não afeta a paridade com o Obsidian: o conceito de
  "documento resolvido" não existe no Obsidian — é uma extensão do My-Memory
  (ADR-055).
- A.12 — **O primeiro caminho de captura é o indexer.** Há **quatro** pontos de
  captura, não dois: `internal/watcher/indexer.go:103` (SQLite, arquivo único),
  `internal/watcher/indexer.go:238` (Postgres, arquivo único) e
  `cmd/mem/main.go:1689` / `cmd/mem/main.go:1881` (`mem index` em lote, Postgres
  e SQLite). Os quatro precisam gravar as tags.
- A.13 — **A limpeza de tags é explícita.** O `ON DELETE CASCADE` do SQLite
  cobre a remoção; o Postgres não tem cascade no `documents` e o
  `DeleteDocumentData` Postgres não é transacional
  (`internal/store/postgres.go:187`). Por isso a remoção de tags é uma chamada
  explícita nos três caminhos (reindex, purge, prune), e não só uma foreign key.

Open questions: none. As perguntas que a spec levava foram respondidas antes
da execução — a inclusão de tags órfãs (A.6), a resolução de `resolved_doc_id`
(A.7), a forma canônica (A.8), a semântica de descendentes (A.9) e o caminho do
filtro (A.10) estão registradas na ADR-055 (aceita em 2026-10-21) e nas
suposições acima.

---

## Contrato de API (fixado — os wavedowns implementam contra isto)

Este contrato é a interface entre os wavedowns. **Ninguém muda a assinatura
sem avisar o gestor.**

### `internal/store` (compartilhado) — arquivo novo `internal/store/tags.go`

```go
// VaultTag é uma tag do vault com a contagem de documentos que a carregam.
type VaultTag struct {
    // Tag é a forma de exibição: o casing com que a tag foi criada primeiro
    // (norma do Obsidian). Key é a forma canônica usada na comparação.
    Tag           string `json:"tag"`
    Key           string `json:"key"`
    // Count é o número de documentos distintos que carregam a tag.
    Count         int    `json:"count"`
    ResolvedDocID string `json:"resolved_doc_id,omitempty"`
    ResolvedTitle string `json:"resolved_title,omitempty"`
}

// TagBinding é o vínculo entre um documento e uma tag literal.
type TagBinding struct {
    Tag           string `json:"tag"`
    Key           string `json:"key"`
    ResolvedDocID string `json:"resolved_doc_id,omitempty"`
}

// TagFilter restringe o inventário de tags do vault.
type TagFilter struct {
    // DocumentID é o caminho de um documento; vazio = todos.
    DocumentID string
    // Folder é uma pasta relativa ao vault; vazio = todas.
    Folder string
}
```

### `internal/db` (SQLite) — arquivo novo `internal/db/tags.go`

```go
func InsertDocumentTags(ctx context.Context, database *sql.DB, documentID string, tags []store.TagBinding) error
func DeleteDocumentTags(ctx context.Context, database *sql.DB, documentID string) error
func ListVaultTags(ctx context.Context, database *sql.DB, filter store.TagFilter) ([]store.VaultTag, error)
func FindVaultDocumentsByTag(ctx context.Context, database *sql.DB, tag string) ([]store.VaultDocument, error)
func ListDocumentIDTitles(ctx context.Context, database *sql.DB) (map[string]string, error)
```

### `internal/store` (Postgres) — mesmo arquivo novo `internal/store/tags.go`

```go
func (s *PostgresStore) InsertDocumentTags(ctx context.Context, repository, documentID string, tags []store.TagBinding) error
func (s *PostgresStore) DeleteDocumentTags(ctx context.Context, repository, documentID string) error
func (s *PostgresStore) ListVaultTags(ctx context.Context, repository string, filter store.TagFilter) ([]store.VaultTag, error)
func (s *PostgresStore) FindVaultDocumentsByTag(ctx context.Context, repository, tag string) ([]store.VaultDocument, error)
func (s *PostgresStore) ListDocumentIDTitles(ctx context.Context, repository string) (map[string]string, error)
```

### `internal/parser` — arquivo novo `internal/parser/tags.go`

```go
// NormalizeTag devolve a forma canônica de uma tag (minúsculas, sem o '#') e
// diz se ela é válida segundo o formato de tag do Obsidian: letras, números,
// '_', '-', '/' e Unicode; ao menos um caractere não-numérico; sem espaços.
func NormalizeTag(tag string) (canonical string, ok bool)

// ResolveTagTargets resolve cada tag literal para o título do documento que ela
// nomeia, com a mesma estratégia de fuzzy match de ResolveTagConnections.
// A chave do mapa é a tag canônica; tag sem casa devolve "" no mapa.
func ResolveTagTargets(tags []string, availableDocs []string) map[string]string
```

### `store.SearchOptions` — campo novo

```go
type SearchOptions struct {
    Category  string `json:"category,omitempty"`
    Tag       string `json:"tag,omitempty"`   // NOVO: tag canônica; "" = sem filtro
    Level     string `json:"level,omitempty"`
    ShowLinks bool   `json:"show_links,omitempty"`
}
```

O filtro por tag é aplicado como **push-down SQL** nos mesmos pontos onde o
filtro de categoria é aplicado (ver abaixo). Nenhuma assinatura de
`*WithOptions` muda de forma incompatível — apenas ganha o parâmetro que já
existe (`searchOpts`).

### DDL (nos dois engines)

```sql
-- SQLite: internal/db/schema.go, nas DUAS constantes (Schema e FallbackSchema)
CREATE TABLE IF NOT EXISTS document_tags (
    document_id    TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    tag           TEXT NOT NULL,
    tag_key       TEXT NOT NULL,
    resolved_doc_id TEXT,
    created_at    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (document_id, tag_key)
);
CREATE INDEX IF NOT EXISTS idx_document_tags_key ON document_tags(tag_key);
CREATE INDEX IF NOT EXISTS idx_document_tags_document ON document_tags(document_id);

-- Postgres: internal/store/postgres.go, em PostgresSchema.
-- A identidade do documento no Postgres é (repository, id): repository vai na PK.
CREATE TABLE IF NOT EXISTS document_tags (
    repository    TEXT NOT NULL,
    document_id   TEXT NOT NULL,
    tag           TEXT NOT NULL,
    tag_key       TEXT NOT NULL,
    resolved_doc_id TEXT,
    created_at    BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (repository, document_id, tag_key)
);
CREATE INDEX IF NOT EXISTS idx_document_tags_key ON document_tags(repository, tag_key);
CREATE INDEX IF NOT EXISTS idx_document_tags_document ON document_tags(repository, document_id);
```

### Push-down do filtro por tag (os 7 pontos)

O filtro de categoria é aplicado em **7** consultas. O filtro por tag entra
**nos mesmos 7**, com a mesma forma de parâmetro (`? = ''` / `$n = ''`):

| # | Arquivo:linha | Consulta |
|---|---|---|
| 1 | `internal/db/hybrid.go:34` | FTS5 (SQLite) |
| 2 | `internal/db/hybrid.go:135` | expansão de grafo (SQLite) |
| 3 | `internal/db/graph.go:57` | busca vetorial por similaridade (SQLite, sqlite-vec) |
| 4 | `internal/db/graph.go:102` | TurboQuant 4-bit (SQLite) |
| 5 | `internal/store/postgres.go:479` | vetorial (Postgres) |
| 6 | `internal/store/postgres.go:559` | FTS (Postgres) |
| 7 | `internal/store/postgres.go:648` | expansão de grafo (Postgres) |

Fragmento (SQLite; o Postgres é o análogo com `$n` e `dt.repository = ?`):

```sql
AND (? = '' OR EXISTS (
    SELECT 1 FROM document_tags dt
    WHERE dt.document_id = d.id
      AND (dt.tag_key = ? OR dt.tag_key LIKE ? ESCAPE '\')
))
```

O `LIKE` casa as **descendentes**: o padrão é `tag_key` escapado + `/%`.
**Armadilha**: `_` e `%` são curingas do `LIKE`; a tag canônica pode conter `_`
(ela é permitida pelo formato do Obsidian). O padrão precisa ser escapado com
`ESCAPE '\'` e os caracteres `%`, `_` e `\` themselves escapados.

### Pontos de registro da tool MCP

Uma tool nova precisa de **3 pontos** de registro e **2** de wiring:

1. definição da tool em `internal/mcp/tools.go`
2. schema no catálogo do servidor, em `internal/mcp/server.go:121`
3. executor em `internal/mcp/vault_handlers.go:290` (`SetVaultAccess`)
4. wiring no servidor Postgres, em `cmd/mem/main.go:2615`
5. wiring no servidor SQLite, em `cmd/mem/main.go:2794`

E a tool precisa entrar na lista explícita `vaultToolsParity`
(`internal/mcp/vault_handlers_test.go:16`), cujo gate exige a propriedade
`"repository"` no `inputSchema`.

### `normalizeStructuredContent` — type switch

`internal/mcp/structured_content.go:26-36` tem um type switch de tipos
concretos. `[]store.VaultTag` precisa ser **adicionado** ao switch (com
`nonNilSlice`), senão o payload volta a serializar `null` — que é exatamente o
defeito que a própria função existe para impedir.

### Seam da CLI

Padrão dos comandos de leitura do vault (`cmd/mem/files.go:103`):

```go
func runTagsCommand(ctx context.Context, defaultRepo string, args []string, out io.Writer) error
func runTagsCLI(ctx context.Context, defaultRepo string, args []string) error  // wrapper para os.Stdout
```

As flags de storage vêm de `registerVaultFlags`
(`cmd/mem/vault_source.go:99`) e o reader vem de `flags.open(defaultRepo)`. A
interface `vaultReader` (`cmd/mem/vault_source.go:20`) ganha os métodos de tag,
implementados nos dois readers (SQLite e Postgres).

**A ordenação mora no engine** (a API do store), não no comando — é o que torna
a saída estável entre execuções e passível de diff.

---

## User Stories

### P1 — Inventário de tags (paridade com `obsidian tags`)

**Story**: Como usuário de um vault, quero listar todas as tags do vault para
reconciliar com o Obsidian e descobrir conteúdo por taxonomia.

1. **TAG-01** — WHEN o usuário executa `mem tags` THEN o comando SHALL imprimir uma tag por linha, em ordem alfabética, sem cabeçalho
2. **TAG-02** — WHEN o usuário executa `mem tags --counts` THEN o comando SHALL imprimir `tag<TAB>contagem` por linha, com o número de documentos que carregam a tag
3. **TAG-03** — WHEN o usuário executa `mem tags --sort=count` THEN o comando SHALL ordenar por contagem decrescente, e o padrão (sem a flag) SHALL ordenar por nome
4. **TAG-04** — WHEN o usuário executa `mem tags --file <caminho>` THEN o comando SHALL listar apenas as tags do documento desse caminho, e WHEN o usuário executa `mem tags --folder <pasta>` THEN o comando SHALL listar apenas as tags dos documentos sob essa pasta
5. **TAG-05** — WHEN o usuário executa `mem tags --total` THEN o comando SHALL imprimir somente a quantidade de tags distintas do vault
6. **TAG-06** — WHEN o usuário executa `mem tags --json` THEN o comando SHALL imprimir o inventário em JSON, com `tag`, `key`, `count`, `resolved_doc_id` e `resolved_title` por tag
7. **TAG-07** — IF uma tag é hierárquica (ex: `projeto/alpha`) THEN o comando SHALL imprimi-la com o caminho completo, e o inventário SHALL conter a tag hierárquica como uma entrada própria
8. **TAG-08** — IF o vault não tem nenhuma tag THEN o comando SHALL imprimir lista vazia e sair com código 0
9. **TAG-09** — WHEN o usuário executa `mem tags` THEN o comando SHALL incluir as tags órfãs (sem documento que as nomeie), que é o que o Obsidian lista

### P2 — Informação de uma tag (paridade com `obsidian tag`)

**Story**: Como usuário, quero consultar uma tag específica para saber quais notas
a carregam e se ela aponta para algum documento.

10. **TAG-10** — WHEN o usuário executa `mem tag <nome>` THEN o comando SHALL imprimir a contagem de documentos que carregam a tag
11. **TAG-11** — WHEN o usuário executa `mem tag <nome> --verbose` THEN o comando SHALL imprimir a contagem e a lista de documentos que carregam a tag
12. **TAG-12** — WHEN o usuário executa `mem tag <nome>` THEN o comando SHALL contar a tag **e suas descendentes** (ex: `projeto` conta `projeto/alpha`)
13. **TAG-13** — WHEN o usuário executa `mem tag <nome> --total` THEN o comando SHALL imprimir somente a contagem
14. **TAG-14** — WHEN o usuário executa `mem tag <nome> --json` THEN o comando SHALL imprimir em JSON o nome da tag, a contagem, a lista de documentos e o documento resolvido
15. **TAG-15** — IF a tag não existe no vault THEN o comando SHALL imprimir contagem 0 e sair com código 0
16. **TAG-16** — WHEN o usuário executa `mem tag <nome>` sem `<nome>` THEN o comando SHALL imprimir o uso e sair com código diferente de 0

### P3 — Filtro por tag na busca (a promessa DR-5 da ADR-041)

**Story**: Como usuário, quero filtrar a busca por tag para achar notas por
taxonomia, e não só por similaridade de texto.

17. **TAG-17** — WHEN o usuário executa `mem search --tag <tag> "termo"` THEN o comando SHALL devolver somente documentos que carregam a tag **e suas descendentes**
18. **TAG-18** — WHEN o usuário combina `--tag` com os demais filtros (`--category`, `--decay`, `--mode`) THEN o comando SHALL aplicar todos simultaneamente
19. **TAG-19** — WHEN o usuário executa `mem search --tag` sem valor THEN o comando SHALL imprimir o uso e sair com código diferente de 0
20. **TAG-20** — IF nenhuma nota carrega a tag THEN o comando SHALL devolver resultado vazio e sair com código 0
21. **TAG-21** — WHEN o filtro por tag é aplicado THEN ele SHALL entrar como push-down SQL nos mesmos pontos em que o filtro de categoria é aplicado, e SHALL NOT ser aplicado como filtro pós-fusão

### P4 — Persistência (o que faz o resto existir)

**Story**: Como implementador, preciso que a tag sobreviva à indexação, inclusive
quando não casa com documento algum.

22. **TAG-22** — WHEN um documento é indexado THEN o indexer SHALL gravar **todas** as suas tags (frontmatter e inline `#tag`), inclusive as órfãs, **nos quatro caminhos de indexação** (indexer SQLite, indexer Postgres, `mem index` em lote SQLite e `mem index` em lote Postgres)
23. **TAG-23** — WHEN o indexer grava as tags THEN ele SHALL fazê-lo **antes** da resolução fuzzy, e a aresta do grafo SHALL continuar sendo gerada apenas a partir das tags resolvidas
24. **TAG-24** — WHEN um documento é reindexado THEN as tags anteriores dele SHALL ser substituídas, sem duplicar
25. **TAG-25** — WHEN um documento é removido do índice THEN as tags dele SHALL ser removidas junto, nos **três** caminhos de remoção (reindex, purge e prune)
26. **TAG-26** — IF uma tag nomeia um documento do vault THEN o registro SHALL ter `resolved_doc_id` igual ao `documents.id` desse documento
27. **TAG-27** — IF uma tag não nomeia nenhum documento THEN o registro SHALL ter `resolved_doc_id` nulo, e a tag SHALL continuar listando no inventário
28. **TAG-28** — WHEN o índice é aberto THEN a tabela `document_tags` SHALL existir nos dois engines, sem exigir `mem index --force`

### P5 — Superfície MCP

**Story**: Como agente que fala MCP, quero consultar as tags do vault sem
processar texto.

29. **TAG-29** — WHEN um agente chama `memory_get_tags` THEN o servidor SHALL devolver o inventário de tags com contagens, no `content` legível e no `structuredContent`
30. **TAG-30** — WHEN um agente chama `memory_get_tag` com um nome THEN o servidor SHALL devolver a contagem e os documentos da tag, no `content` legível e no `structuredContent`
31. **TAG-31** — WHEN um agente chama `memory_get_tags` com um `path` THEN o servidor SHALL devolver apenas as tags daquele documento
32. **TAG-32** — WHEN uma tool MCP devolve `structuredContent` THEN o payload SHALL ter listas vazias (`[]`) e nunca `null` — a regra do `normalizeStructuredContent` de `internal/mcp/structured_content.go`
33. **TAG-33** — WHEN as tools de tag respondem sem dado THEN o `content` legível SHALL explicar que o vault não tem tags, e a tool SHALL NOT sinalizar erro

### P6 — Paridade documentada

**Story**: Como usuário, quero saber que a superfície existe e como usá-la.

34. **TAG-34** — WHEN a feature é entregue THEN `docs/CLI_GUIDE.md` SHALL documentar `mem tags`, `mem tag` e `mem search --tag`
35. **TAG-35** — WHEN a feature é entregue THEN `README.md` e `docs/adr/README.md` SHALL registrar a superfície como paridade implementada
36. **TAG-36** — WHEN a feature é entregue THEN o `CHANGELOG.md` SHALL registrar a adição, e a ADR-055 SHALL ser marcada como implementada no índice de ADRs

### P7 — Formato e semântica da tag (a norma do Obsidian)

**Story**: Como usuário, quero que a tag seja capturada e comparada exatamente
como o Obsidian faz, para que a reconciliação entre os dois seja real.

37. **TAG-37** — WHEN uma tag é extraída THEN ela SHALL aceitar os caracteres do Obsidian: letras, números, `_`, `-`, `/` e caracteres Unicode, e SHALL NOT aceitar espaço em branco; o primeiro caractere SHALL poder ser dígito ou `_`, desde que a tag **inteira** tenha ao menos um caractere não-numérico
38. **TAG-38** — IF uma tag é composta apenas de dígitos (ex: `#1984`) THEN o parser SHALL rejeitá-la, e a tag SHALL NOT ser gravada no inventário
39. **TAG-39** — WHEN duas tags diferem apenas por caixa (ex: `#Tag` e `#TAG`) THEN o inventário SHALL contê-las como **uma** entrada, e a busca por tag SHALL casar as duas
40. **TAG-40** — WHEN o inventário é exibido THEN a tag SHALL aparecer com o casing com que foi criada primeiro no vault
41. **TAG-41** — WHEN uma tag hierárquica é consultada THEN a consulta SHALL casar a tag e as descendentes, e o inventário SHALL manter as tags hierárquicas como entradas literais separadas
42. **TAG-42** — WHEN o frontmatter declara `tags` como lista YAML THEN o parser SHALL aceitar, e WHEN declara `tag` (alias depreciado) THEN o parser SHALL aceitar também
43. **TAG-43** — WHEN um hashtag aparece em uma propriedade de texto que não seja `tags` THEN o parser SHALL NOT criar tag a partir dele
44. **TAG-44** — IF uma tag ou um wikilink aparece dentro de um comentário `%%` (inline ou em bloco) THEN o parser SHALL NOT extraí-la, e nenhum link dentro do comentário SHALL virar aresta
45. **TAG-45** — WHEN uma tag aparece dentro de uma célula de tabela Markdown THEN o parser SHALL extraí-la e registrá-la
46. **TAG-46** — WHEN um bloco de código é delimitado por `~~~` THEN o parser SHALL NOT extrair tag nem wikilink do seu interior, e o bloco SHALL ser tratado como código em todo o caminho (parser e `mem outline`)

---

## Requirement Traceability

| Requirement ID | Story | Phase | Status |
| -------------- | ----------- | ------ | ------- |
| TAG-01 | P1: Inventário de tags | Execute | Pending |
| TAG-02 | P1: Inventário de tags | Execute | Pending |
| TAG-03 | P1: Inventário de tags | Execute | Pending |
| TAG-04 | P1: Inventário de tags | Execute | Pending |
| TAG-05 | P1: Inventário de tags | Execute | Pending |
| TAG-06 | P1: Inventário de tags | Execute | Pending |
| TAG-07 | P1: Inventário de tags | Execute | Pending |
| TAG-08 | P1: Inventário de tags | Execute | Pending |
| TAG-09 | P1: Inventário de tags | Execute | Pending |
| TAG-10 | P2: Informação de uma tag | Execute | Pending |
| TAG-11 | P2: Informação de uma tag | Execute | Pending |
| TAG-12 | P2: Informação de uma tag | Execute | Pending |
| TAG-13 | P2: Informação de uma tag | Execute | Pending |
| TAG-14 | P2: Informação de uma tag | Execute | Pending |
| TAG-15 | P2: Informação de uma tag | Execute | Pending |
| TAG-16 | P2: Informação de uma tag | Execute | Pending |
| TAG-17 | P3: Filtro na busca | Execute | Pending |
| TAG-18 | P3: Filtro na busca | Execute | Pending |
| TAG-19 | P3: Filtro na busca | Execute | Pending |
| TAG-20 | P3: Filtro na busca | Execute | Pending |
| TAG-21 | P3: Filtro na busca | Execute | Pending |
| TAG-22 | P4: Persistência | Execute | Pending |
| TAG-23 | P4: Persistência | Execute | Pending |
| TAG-24 | P4: Persistência | Execute | Pending |
| TAG-25 | P4: Persistência | Execute | Pending |
| TAG-26 | P4: Persistência | Execute | Pending |
| TAG-27 | P4: Persistência | Execute | Pending |
| TAG-28 | P4: Persistência | Execute | Pending |
| TAG-29 | P5: Superfície MCP | Execute | Pending |
| TAG-30 | P5: Superfície MCP | Execute | Pending |
| TAG-31 | P5: Superfície MCP | Execute | Pending |
| TAG-32 | P5: Superfície MCP | Execute | Pending |
| TAG-33 | P5: Superfície MCP | Execute | Pending |
| TAG-34 | P6: Documentação | Execute | Pending |
| TAG-35 | P6: Documentação | Execute | Pending |
| TAG-36 | P6: Documentação | Execute | Pending |
| TAG-37 | P7: Formato da tag | Execute | Pending |
| TAG-38 | P7: Formato da tag | Execute | Pending |
| TAG-39 | P7: Formato da tag | Execute | Pending |
| TAG-40 | P7: Formato da tag | Execute | Pending |
| TAG-41 | P7: Formato da tag | Execute | Pending |
| TAG-42 | P7: Formato da tag | Execute | Pending |
| TAG-43 | P7: Formato da tag | Execute | Pending |
| TAG-44 | P7: Formato da tag | Execute | Pending |
| TAG-45 | P7: Formato da tag | Execute | Pending |
| TAG-46 | P7: Formato da tag | Execute | Pending |

**Coverage:** 46 total, 46 mapped to stories, 0 unmapped.

> **Status `Pending` por convenção do repositório.** A feature foi implementada e
> validada: os 46 ACs estão **PASS**, com evidência `caminho:linha` por critério em
> [`validation.md`](validation.md) e o gate Go verde (`go build` 0 · `go vet` 0 ·
> `go test -count=1 ./...` 25/25 pacotes, 0 falhas). A coluna `Status` continua
> `Pending` porque é assim que o repo registra: as features concluídas
> `obsidian-parity-foundation` e `mcp-static-serving` mantêm o mesmo desenho — o
> veredito vive no `validation.md`, e a spec é documento de planejamento
> (achado B2 da auditoria da `mcp-static-serving`).
