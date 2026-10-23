# Paridade de tags com o Obsidian — Tasks

## Execution Protocol (MANDATORY -- do not skip)

- Execution follows the **5 phases** below. Phase 1 runs **in parallel** (T1 and
  T2 touch disjoint files); Phases 2, 3, 4 and 5 are **strictly sequential**.
- **Why sequential from T3 on**: T3, T4 and T5 all touch `cmd/mem/main.go` — T3
  because the batch `mem index` path lives there, T4 because the new subcommands
  are dispatched there, T5 because the `--tag` filter is a flag of the `search`
  case. Parallel workers would write to the same file, so they run in series.
- Every task must satisfy its **Done when** checklist and its **Gate** before the
  next phase starts. A task that fails its gate is fixed immediately; it is never
  carried forward.
- **The API contract in the spec is FIXED.**
  `.specs/features/obsidian-tag-parity/spec.md` §"Contrato de API" defines the
  exact signatures of `store.VaultTag`, `store.TagBinding`, `store.TagFilter`,
  the `internal/db` functions, the `internal/store` methods, `parser.NormalizeTag`
  and `parser.ResolveTagTargets`, the DDL of both engines and the 7 push-down
  sites. A worker that needs a different signature **reports to the orchestrator
  instead of changing it** — changing it would break the other wave.
- **No commit is made by the worker.** The orchestrator audits and commits.
  `git add -A`, `git add .` and `git stash` are forbidden.
- File ownership is **disjoint per task**:
  | Task | Owns |
  | ---- | ---- |
  | T1 | `internal/parser/**` |
  | T2 | `internal/db/**`, `internal/store/**` |
  | T3 | `internal/watcher/**`, `cmd/mem/main.go` (blocos de indexação) |
  | T4 | `cmd/mem/tags.go` (novo), `cmd/mem/vault_source.go`, `cmd/mem/main.go` (dispatch) |
  | T5 | `internal/mcp/**`, `internal/store/**`, `internal/db/**`, `cmd/mem/main.go` (case "search") |
  | T6 | `docs/**`, `README.md`, `CHANGELOG.md`, `AGENTS.md`, `.specs/**` |
- The Go gate must stay green at every task: `go build ./... && go vet ./... && go test -count=1 ./...`.
- **Never** run `gofmt -w .` — apply gofmt only to the specific files the task touches.
  The working tree has CRLF from the old Windows checkout; `gofmt -l .` there lists
  ~200 files that are fine in the repo. Prove formatting on a LF-normalized copy.

## Test Coverage Matrix

| Layer | Test type | File | Status |
| ----- | --------- | ---- | ------ |
| Formato e validação da tag | unit | `internal/parser/tags_test.go` | pronto |
| Resolução de tag-alvo | unit | `internal/parser/tags_test.go` | pronto |
| DDL e API de tags (SQLite) | unit | `internal/db/tags_test.go` | pronto |
| DDL e API de tags (Postgres) | unit | `internal/store/tags_test.go` | pronto |
| Captura de tags nos 4 caminhos de indexação | integration | `internal/watcher/tags_index_test.go` | pronto |
| Comandos `mem tags` / `mem tag` | unit | `cmd/mem/tags_test.go` | pronto |
| Filtro `--tag` na busca (push-down) | integration | `cmd/mem/search_tag_test.go` | pronto |
| Tools MCP de tag | unit | `internal/mcp/tags_handlers_test.go` | pronto |
| Paridade de ponta a ponta com o Obsidian | manual | `mem tags` / `mem tag` / `mem search --tag` sobre um vault de teste | coberto por T6 |
| Verificação adversarial e documentação | none | — | coberto por T6 (a verificação é o gate) |

## Gate Check Commands

```bash
# Gate Go completo (roda em todo task)
cd C:\repository\my-memory
go build ./...
go vet ./...
go test -count=1 ./...

# Gate por pacote, durante o desenvolvimento
go test -count=1 ./internal/db/...
go test -count=1 ./internal/store/...
go test -count=1 ./internal/parser/...
go test -count=1 ./internal/watcher/...
go test -count=1 ./cmd/mem/...
go test -count=1 ./internal/mcp/...

# Prova de formatação (LF normalizado — o working tree do Windows tem CRLF)
# Copiar os arquivos tocados, trocar CRLF por LF e rodar gofmt -l sobre a cópia.
```

## Execution Plan

### Phase 1: Fundações (paralelo — T1 e T2 são disjuntos)

```
T1 → T3 → T4 → T5 → T6
T2 → T3
```

### Phase 2: Captura da tag na indexação

```
T3
```

### Phase 3: Superfície CLI

```
T4
```

### Phase 4: Filtro na busca e tools MCP

```
T5
```

### Phase 5: Verificação e documentação

```
T6
```

---

## Status da execução

| Task | Escopo | Status | Evidência |
| ---- | ------ | ------ | --------- |
| T1: formato da tag e resolução no parser | `internal/parser/tags.go` | ✅ Concluída | `internal/parser/tags.go:44` (`NormalizeTag`), `:107` (`ResolveTagTargets`); `internal/parser/tags_test.go` |
| T2: tabela `document_tags` e API nos dois engines | `internal/store/tags.go`, `internal/db/tags.go` | ✅ Concluída | `internal/db/schema.go:151-162` e `:310-321`; `internal/store/postgres.go:84-94` |
| T3: captura das tags nos 4 caminhos de indexação | `internal/watcher/indexer.go`, `cmd/mem/main.go` | ✅ Concluída | `internal/watcher/indexer.go:109`, `:253`; `cmd/mem/main.go:1730`, `:1940` |
| T4: `mem tags` e `mem tag` na CLI | `cmd/mem/tags.go` | ✅ Concluída | `cmd/mem/tags.go:35` (`runTagsCLI`), `:101` (`runTagCLI`); `cmd/mem/tags_test.go` |
| T5: filtro `--tag` na busca e tools MCP | `internal/mcp/`, `internal/db/`, `internal/store/`, `cmd/mem/main.go` | ✅ Concluída | os 7 pontos de push-down (`internal/db/hybrid.go:43`, `:154`; `internal/db/graph.go:66`, `:124`; `internal/store/postgres.go:502`, `:595`, `:697`); `internal/mcp/tools.go:698`, `:716`; `internal/mcp/server.go:127-128` |
| T6: verificação e documentação | `docs/**`, `README.md`, `CHANGELOG.md`, `skills/**`, `.specs/**` | ✅ Concluída | `.specs/features/obsidian-tag-parity/validation.md` (46/46 ACs PASS); `CHANGELOG.md:12-51`; `docs/CLI_GUIDE.md:145-193` e `:1086-1152`; `docs/adr/README.md:58` |

**Veredito da execução: PASS** — 46/46 critérios de aceitação, gate Go verde
(`go build` 0 · `go vet` 0 · `go test -count=1 ./...` **25/25 pacotes, 0 falhas**).
A prova por critério, com evidência `caminho:linha`, está em
[`validation.md`](validation.md).

> **Por que os checkboxes de "Done when" continuam desmarcados.** É a convenção do
> repositório: as features concluídas `mcp-static-serving`, `viewer-dashboard` e
> `obsidian-parity-foundation` mantêm os checkboxes do plano intactos, e o
> registro de conclusão fica no `validation.md` e no `.specs/STATE.md`. O
> `tasks.md` é o **plano**; o `validation.md` é a **prova**. Desmarcar os
> checkboxes aqui seria inventar um formato que o repo não usa.

---

## Task Breakdown

### T1: Formato da tag e resolução do documento-alvo no parser

**What**: Em `internal/parser/tags.go` (novo), implementar a norma de tag do
Obsidian (`obsidian.md/help/tags`):

- `NormalizeTag(tag string) (canonical string, ok bool)` — devolve a forma
  canônica (minúsculas, sem o `#`) e diz se a tag é válida. Válida = sem espaço
  em branco, só com os caracteres permitidos (letras, números, `_`, `-`, `/` e
  Unicode/emoji) e com **ao menos um caractere não-numérico** (`#1984` é
  inválida, `#y1984` é válida).
- `ResolveTagTargets(tags []string, availableDocs []string) map[string]string` —
  resolve cada tag para o **título** do documento que ela nomeia, com a mesma
  estratégia de fuzzy de `ResolveTagConnections` (reusando `fuzzyResolveTag` e
  `normalizeForFuzzy`, **sem alterá-los**). A chave do mapa é a tag canônica; a
  tag sem casa devolve `""` **e continua no mapa**.
- Ajustar `tagRegex` (`internal/parser/wikilinks.go:15`) para aceitar o conjunto
  de caracteres do Obsidian. Hoje ela só aceita ASCII **iniciando em letra**, o
  que descarta `#1a`, `#_tag`, `#ação`, `#Ω` e `#🚀` — e ainda **trunca**:
  `#ação` vira a tag espúria `"a"`, porque `ç` está fora da classe. A tag
  continua **crua** em `conn.Tags` (a forma de exibição) — a canonicalização é
  da chamada, não da extração.
- Corrigir o strip de blocos de código (`internal/parser/wikilinks.go:20`): ele
  casa só ```` ``` ```` e ignora `~~~`, de modo que uma tag ou um wikilink
  dentro de um fence `~~~` vira aresta e tag reais. O `mem outline`
  (`cmd/mem/outline.go:61`) já trata os dois — o parser é que está atrás.
- Remover os comentários `%%` (inline e em bloco) **antes** da extração de
  tag e de link. Hoje não há nenhum tratamento de `%%` no código: o comentário é
  indexado, buscável, e as tags dentro dele viram tags reais.
  **Limite deliberado:** o strip vale para a **extração**, não para o texto
  indexado. O `mem read` remonta o arquivo a partir dos chunks
  (`internal/store/chunks.go:20-36`), então remover o comentário do índice
  tornaria a leitura lossy. Remover do índice é uma decisão de produto e está
  registrada na [matriz de paridade](../../../docs/OBSIDIAN_PARITY_MATRIX.md),
  decisão nº 5.
- Para a **extração de tag**, as linhas de tabela (`| … |`) **não** devem ser
  removidas: no Obsidian uma tag dentro de uma célula de tabela é uma tag
  válida. O strip de tabelas segue valendo para a extração de **wikilink**
  (defesa contra link fantasma), que não muda nesta task.
- O `stripCodeBlocks` continua sendo um strip **só** para extração: o texto
  indexado permanece o bruto.

Não alterar: `ResolveTagConnections`, `fuzzyResolveTag`, `normalizeForFuzzy`.
A aresta `tagged_as` do grafo continua sendo criada só para tags resolvidas
(ADR-041).

**Where**: `internal/parser/tags.go` (novo)
**Depends on**: None
**Reuses**: `fuzzyResolveTag` e `normalizeForFuzzy` (`internal/parser/fuzzy.go`),
`tagRegex` (`internal/parser/wikilinks.go:15`), `Frontmatter.Tags`
(`internal/parser/frontmatter.go:70`)
**Requirement**: TAG-37, TAG-38, TAG-41, TAG-42, TAG-43

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] `NormalizeTag` aceita `#a`, `#a_b`, `#a-b`, `#a/b`, `#ação`, `#🚀`, `#y1984`
- [ ] `NormalizeTag` aceita tag que **começa** por dígito (`#1a`) ou por `_` (`#_tag`)
- [ ] `NormalizeTag` rejeita `#1984` (só numérica), `#a b` (espaço) e `#a.b` (ponto)
- [ ] `NormalizeTag("#Tag")` e `NormalizeTag("#TAG")` devolvem a **mesma** canônica
- [ ] `NormalizeTag` de uma tag hierárquica preserva a hierarquia na canônica (`a/b`)
- [ ] `tagRegex` captura `#1a`, `#_tag`, `#ação` e `#🚀` no corpo
- [ ] `tagRegex` captura `#ação` **inteiro** — não trunca em `"a"`
- [ ] `tagRegex` **não** captura `#1984` nem `#a b`
- [ ] Tag dentro de bloco ```` ``` ```` **ou** `~~~` não é capturada
- [ ] Tag dentro de comentário `%%` (inline e em bloco) não é capturada
- [ ] Tag dentro de uma célula de tabela **é** capturada
- [ ] O strip de linhas de tabela continua valendo para a extração de wikilink
- [ ] `conn.Tags` mantém a forma crua (exibição), não a canônica
- [ ] `ResolveTagTargets` resolve por match exato, substring e token, como o fuzzy de edges
- [ ] Tag sem casa devolve `""` no mapa, e a chave continua presente
- [ ] A resolução devolve **título**, não id — a conversão título → id é de quem indexa
- [ ] O grafo não mudou: `tagged_as` só para tags resolvidas
- [ ] Testes cobrem: formato, case, numérica, hierárquica, resolução (exato/substring/token/sem casa)
- [ ] Gate check passa: `go test -count=1 ./internal/parser/...`

**Tests**: unit — `internal/parser/tags_test.go`
**Gate**: full

**Commit**: `feat(parser): normalizar a tag no formato do Obsidian e resolver seu alvo`

---

### T2: Tabela `document_tags` e API de tags nos dois engines

**What**: Criar a tabela `document_tags` nos dois engines e a API de leitura e
escrita, exatamente com as assinaturas da spec.

- Tipos em `internal/store/tags.go` (novo): `VaultTag{Tag, Key, Count,
  ResolvedDocID, ResolvedTitle}`, `TagBinding{Tag, Key, ResolvedDocID}` e
  `TagFilter{DocumentID, Folder}`.
- DDL SQLite nas **duas** constantes de `internal/db/schema.go` (`Schema` e
  `FallbackSchema`) e DDL Postgres em `internal/store/postgres.go`
  (`PostgresSchema`) — o DDL exato está na spec. No Postgres a PK inclui
  `repository`, porque a identidade do documento lá é `(repository, id)`.
- API SQLite em `internal/db/tags.go` (novo) e API Postgres em
  `internal/store/tags.go`, com as assinaturas da spec.

Semântica obrigatória:

- `InsertDocumentTags` **substitui** as tags do documento (apaga as anteriores e
  insere as novas, **em transação** nos dois engines) — reindexar não duplica.
- A identidade da linha é `(document_id, tag_key)`: `#Tag` e `#TAG` são a mesma
  linha.
- `ListVaultTags` devolve o inventário com a contagem de documentos distintos por
  tag, agrupado por `tag_key`. O **casing de exibição** é o da tag da linha
  "criada primeiro": menor `created_at`, desempate por `document_id` e depois
  pela própria `tag` (regra determinística, idêntica nos dois engines).
- `ListVaultTags` respeita o `TagFilter`: `DocumentID` restringe a um documento,
  `Folder` restringe aos documentos sob a pasta.
- `FindVaultDocumentsByTag` casa a tag **e as descendentes** (`tag` exato ou
  `tag` seguido de `/`), com o `LIKE` **escapado** (`ESCAPE '\'`): `_` e `%` são
  curingas do `LIKE` e a tag pode conter `_`.
- `ListDocumentIDTitles` devolve o mapa título → `documents.id`.
- `DeleteDocumentTags` remove as tags de um documento (usado nos 3 caminhos de
  remoção, porque o Postgres não tem cascade em `documents`).

**Where**: `internal/store/tags.go` (novo)
**Depends on**: None
**Reuses**: `store.VaultDocument` (`internal/store/vault.go`), o padrão de query de
`internal/db/vault.go` e de `internal/store/vault.go`, `ListDocumentTitles`
(`internal/db/store.go:810`) como referência de formato
**Requirement**: TAG-24, TAG-25, TAG-27, TAG-28, TAG-39, TAG-40

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] A tabela `document_tags` existe em `Schema`, `FallbackSchema` e `PostgresSchema`, com o DDL da spec
- [ ] No Postgres a PK é `(repository, document_id, tag_key)`
- [ ] `InsertDocumentTags` substitui (não acumula) as tags do documento, nos dois engines
- [ ] `#Tag` e `#TAG` viram **uma** linha (mesmo `tag_key`)
- [ ] `FindVaultDocumentsByTag` casa a tag e as descendentes (`a` casa `a/b`)
- [ ] O `LIKE` de descendentes escapa `%`, `_` e `\` — uma tag `a_b` não casa `aXb`
- [ ] `ListVaultTags` conta documentos distintos por tag e respeita `TagFilter`
- [ ] O casing de exibição é o da linha de menor `created_at`, com desempate determinístico
- [ ] `ListDocumentIDTitles` devolve título → id
- [ ] Um vault indexado antes da tabela **não** exige `mem index --force`: abrir o índice cria a tabela
- [ ] Testes cobrem: escrita, substituição, descendentes, contagem, filtro, casing, escape do `LIKE`
- [ ] Gate check passa: `go test -count=1 ./internal/db/... ./internal/store/...`

**Tests**: unit — `internal/db/tags_test.go` e `internal/store/tags_test.go`
**Gate**: full

**Commit**: `feat(store): persistir as tags do vault em document_tags`

---

### T3: Gravar as tags na indexação (os quatro caminhos)

**What**: Gravar as tags de cada documento indexado. São **quatro** pontos de
captura, não dois:

1. `internal/watcher/indexer.go:103` — `IndexSingleFileSQLite`
2. `internal/watcher/indexer.go:238` — `IndexSingleFilePostgres`
3. `cmd/mem/main.go:1689` — `mem index` em lote, Postgres
4. `cmd/mem/main.go:1881` — `mem index` em lote, SQLite

Em cada um: montar os `TagBinding` a partir de `conn.Tags` (canonicalizados com
`parser.NormalizeTag`) e gravá-los **antes** de `parser.ResolveTagConnections`,
para que a tag órfã também sobre. O `resolved_doc_id` vem de
`parser.ResolveTagTargets` convertido para id via `ListDocumentIDTitles`.

A aresta do grafo **não muda**: `tagged_as` continua sendo criada só para as
tags resolvidas (ADR-041).

Remoção: nos **três** caminhos — reindex (o `InsertDocumentTags` já substitui),
purge (`PurgeSingleFileSQLite` e equivalente) e prune — as tags do documento são
removidas junto, por chamada explícita a `DeleteDocumentTags`. O Postgres não
tem cascade em `documents` e o `DeleteDocumentData` Postgres não é transacional
(`internal/store/postgres.go:187`): a limpeza explícita é o que garante.

**Where**: `internal/watcher/indexer.go`
**Depends on**: T1, T2
**Reuses**: `parser.ExtractConnections`, `parser.NormalizeTag`,
`parser.ResolveTagTargets`, a API de tags da T2
**Requirement**: TAG-22, TAG-23, TAG-26

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] Os **quatro** pontos de captura gravam as tags
- [ ] As tags são gravadas **antes** da resolução fuzzy
- [ ] Uma tag órfã (sem documento que a nomeie) **é gravada**, com `resolved_doc_id` nulo
- [ ] Uma tag que nomeia um documento tem `resolved_doc_id` igual ao `documents.id` desse documento
- [ ] `#Tag` e `#TAG` no mesmo documento viram uma linha
- [ ] Reindexar o mesmo arquivo não duplica tags
- [ ] Remover o documento do índice remove as tags dele (reindex, purge e prune)
- [ ] O grafo **não** mudou: a aresta `tagged_as` continua sendo criada só para tags resolvidas
- [ ] Testes cobrem: tag órfã, tag resolvida, reindexação, remoção, os dois engines
- [ ] Gate check passa: `go test -count=1 ./internal/watcher/... ./cmd/mem/...`

**Tests**: unit — `internal/watcher/tags_index_test.go`
**Gate**: full

**Commit**: `feat(indexer): gravar as tags do documento antes da resolucao fuzzy`

---

### T4: Os comandos `mem tags` e `mem tag` na CLI

**What**: Criar `cmd/mem/tags.go` (novo) com os dois subcomandos, no molde de
`cmd/mem/files.go` (`runXCommand(ctx, defaultRepo, args, out io.Writer)` +
wrapper `runXCLI` para `os.Stdout`). Registrar os dois no dispatch de
`cmd/mem/main.go` e na ajuda (`printHelp`).

`mem tags`: uma tag por linha em ordem alfabética; `--counts` imprime
`tag<TAB>contagem`; `--sort=count` ordena por contagem decrescente (padrão: por
nome); `--file <caminho>` lista as tags de um documento; `--folder <pasta>` lista
as tags dos documentos sob a pasta; `--total` imprime só a quantidade de tags
distintas; `--json` imprime o inventário em JSON.

`mem tag <nome>`: imprime a contagem; `--verbose` imprime a contagem e a lista
de documentos; `--total` imprime só a contagem; `--json` imprime nome, contagem,
documentos e documento resolvido. `mem tag` casa a tag **e as descendentes** — a
mesma semântica do Obsidian (`tag:inbox` casa `#inbox` e `#inbox/to-read`).
Tag inexistente → contagem 0, exit 0. Sem `<nome>` → uso e exit != 0.

Estender a interface `vaultReader` (`cmd/mem/vault_source.go:20`) com os métodos
de tag e implementar nos dois readers (SQLite e Postgres). A **ordenação padrão**
mora no engine (a API do store devolve por `tag_key` crescente); o `--sort=count`
é a única reordenação do comando, e ela precisa de desempate determinístico
(contagem DESC, depois tag ASC) para a saída ser estável entre execuções.

**Where**: `cmd/mem/tags.go` (novo)
**Depends on**: T3
**Reuses**: `cmd/mem/files.go` (molde do comando), `registerVaultFlags` e a
interface `vaultReader` (`cmd/mem/vault_source.go`), `store.VaultTag`,
`store.TagFilter`
**Requirement**: TAG-01, TAG-02, TAG-03, TAG-04, TAG-05, TAG-06, TAG-07, TAG-08, TAG-09, TAG-10, TAG-11, TAG-12, TAG-13, TAG-14, TAG-15, TAG-16, TAG-39, TAG-40

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] `mem tags` lista uma tag por linha, em ordem alfabética
- [ ] `mem tags --counts` imprime `tag<TAB>contagem`
- [ ] `mem tags --sort=count` ordena por contagem decrescente
- [ ] `mem tags --file <caminho>` filtra por documento
- [ ] `mem tags --folder <pasta>` filtra por pasta
- [ ] `mem tags --total` imprime só a contagem de tags distintas
- [ ] `mem tags --json` emite o inventário estruturado
- [ ] A tag hierárquica aparece com o caminho completo, como entrada própria
- [ ] Vault sem tags → lista vazia, exit 0
- [ ] A tag órfã aparece no inventário
- [ ] `#Tag` e `#TAG` aparecem como **uma** entrada, com o casing da primeira criação
- [ ] `mem tag <nome>` imprime a contagem; `--verbose` imprime os documentos
- [ ] `mem tag <nome>` conta descendentes
- [ ] `mem tag` sem nome → uso e exit != 0
- [ ] `mem tags` e `mem tag` funcionam nos dois engines
- [ ] Os dois comandos aparecem na ajuda (`printHelp`)
- [ ] Gate check passa: `go test -count=1 ./cmd/mem/...`

**Tests**: unit — `cmd/mem/tags_test.go`
**Gate**: full

**Commit**: `feat(cli): listar e consultar as tags do vault`

---

### T5: Filtro `--tag` na busca e as tools MCP de tag

**What**: Duas superfícies.

**(a) Busca.** `mem search --tag <tag>` restringe os resultados aos documentos
que carregam a tag **e suas descendentes**, combinado com os demais filtros
(`--category`, `--decay`, `--mode`). Sem valor → uso e exit != 0; tag sem nota →
resultado vazio e exit 0. O filtro tem de entrar como **push-down** nos **7**
sites listados na spec — os mesmos onde o filtro de categoria é aplicado — e
**não** como filtro pós-fusão (`internal/store/rrf.go:165-174` roda antes do rank
recíproco e mudaria a ranqueação relativa; `SearchResult` não carrega tags).
Isso exige o campo `Tag` em `store.SearchOptions` e o escape do `LIKE` de
descendentes.

**(b) MCP.** As tools `memory_get_tags` e `memory_get_tag`: definição em
`internal/mcp/tools.go`, schema no catálogo (`internal/mcp/server.go:121`),
executores em `internal/mcp/tags_handlers.go` (novo), wiring nos dois blocos de
`cmd/mem/main.go` (Postgres e SQLite) e entrada na lista explícita
`vaultToolsParity` (`internal/mcp/vault_handlers_test.go:16`). O
`structuredContent` segue a spec (TAG-32: listas vazias como `[]`, nunca `null`) —
e `[]store.VaultTag` precisa entrar no type switch de
`internal/mcp/structured_content.go:26-36`. `memory_get_tags` aceita `path` para
as tags de um documento. O `content` legível tem de explicar o resultado —
inclusive quando o vault não tem tags, sem sinalizar erro.

**Where**: `internal/mcp/tags_handlers.go` (novo)
**Depends on**: T4
**Reuses**: o padrão de `internal/mcp/vault_handlers.go` e de
`internal/mcp/list_files` (handler de listagem), `NewStructuredResult` e
`normalizeStructuredContent` (`internal/mcp/structured_content.go`), o caminho do
filtro `--category` nos 7 sites da spec
**Requirement**: TAG-17, TAG-18, TAG-19, TAG-20, TAG-21, TAG-29, TAG-30, TAG-31, TAG-32, TAG-33

**Tools**:

- MCP: `memory_get_tags`, `memory_get_tag` (novas)
- Skill: `my-memory`

**Done when**:

- [ ] `mem search --tag <tag> "termo"` devolve só documentos com a tag
- [ ] `--tag` casa as descendentes (`--tag a` traz `a/b`)
- [ ] O filtro é push-down nos 7 sites — nenhum filtro pós-fusão foi adicionado
- [ ] `--tag` combina com `--category`, `--decay` e `--mode`
- [ ] `mem search --tag` sem valor → uso e exit != 0
- [ ] Tag sem nota → resultado vazio, exit 0
- [ ] `memory_get_tags` devolve o inventário com contagens, em `content` e `structuredContent`
- [ ] `memory_get_tag` devolve contagem e documentos, em `content` e `structuredContent`
- [ ] `memory_get_tags` com `path` devolve as tags daquele documento
- [ ] O `structuredContent` tem `[]` e nunca `null` (o type switch cobre `[]store.VaultTag`)
- [ ] As tools não sinalizam erro quando o vault não tem tags
- [ ] As duas tools estão nos 3 pontos de registro, nos 2 wirings e na lista `vaultToolsParity`
- [ ] O número de tools MCP subiu de 24 para 26, e o texto do repositório foi conferido
- [ ] Gate check passa: `go test -count=1 ./internal/mcp/... ./cmd/mem/... ./internal/db/... ./internal/store/...`

**Tests**: unit — `internal/mcp/tags_handlers_test.go` e `cmd/mem/search_tag_test.go`
**Gate**: full

**Commit**: `feat(mcp): expor as tags do vault e filtrar a busca por tag`

---

### T6: Verificação independente, gate completo e documentação

**What**: Auditar a feature inteira contra os 43 ACs da spec, com adversarial
probing (inclusive a coexistência com o ADR-041: a tabela `document_tags` não pode
reintroduzir nó morto no grafo nem alterar o Health Score). Rodar o gate Go
completo e o gate de frontend (a viewer não é tocada, mas o CI roda os dois).
Documentar: `docs/CLI_GUIDE.md` (os três comandos), `README.md` e
`docs/adr/README.md` (a paridade implementada e a ADR-055 marcada como
implementada), `CHANGELOG.md` (a adição), `AGENTS.md` (se a skill de vault ganhou
comando). Fechar a spec com `validation.md` e a tabela de rastreabilidade.

**Where**: `docs/CLI_GUIDE.md`
**Depends on**: T5
**Reuses**: o padrão de `docs/CLI_GUIDE.md` e o de `.specs/features/mcp-static-serving/validation.md`
**Requirement**: TAG-34, TAG-35, TAG-36

**Tools**:

- MCP: NONE
- Skill: `my-memory`

**Done when**:

- [ ] Os 43 ACs foram verificados um a um, com evidência `caminho:linha`
- [ ] O grafo e o Health Score não mudaram (o ADR-041 continua valendo)
- [ ] `go build ./... && go vet ./... && go test -count=1 ./...` verde
- [ ] `docs/CLI_GUIDE.md` documenta `mem tags`, `mem tag` e `mem search --tag`
- [ ] `README.md`, `docs/adr/README.md` e `CHANGELOG.md` registram a paridade
- [ ] A ADR-055 aparece como implementada no índice de ADRs
- [ ] `validation.md` existe e a spec está fechada
- [ ] `git status` sem untracked de teste

**Tests**: none — a verificação é o gate
**Gate**: full

**Commit**: `docs: documentar a paridade de tags com o Obsidian`

---

## Traceability (dependências × diagrama)

| Task | Phase | Depends on | Diagram arrow |
| ---- | ----- | ---------- | ------------- |
| T1 | 1 | None | — |
| T2 | 1 | None | — |
| T3 | 2 | T1, T2 | `T1 → T3`, `T2 → T3` |
| T4 | 3 | T3 | `T3 → T4` |
| T5 | 4 | T4 | `T4 → T5` |
| T6 | 5 | T5 | `T5 → T6` |

## Coverage por requisito

| Requirement | Task |
| ----------- | ---- |
| TAG-01..TAG-16 | T4 |
| TAG-17..TAG-21 | T5 |
| TAG-22, TAG-23, TAG-26 | T3 |
| TAG-24, TAG-25, TAG-27, TAG-28, TAG-39, TAG-40 | T2 |
| TAG-29..TAG-33 | T5 |
| TAG-34, TAG-35, TAG-36 | T6 |
| TAG-37, TAG-38, TAG-41, TAG-42, TAG-43, TAG-44, TAG-45, TAG-46 | T1 |

**Cobertura:** 43 requisitos, 43 mapeados, 0 órfãos.
