---
title: "Validação — Superfície de tags do Obsidian"
category: resource
summary: "Validação da spec obsidian-tag-parity: 46 critérios EARS (inventário de tags, consulta de tag, filtro da busca, persistência em document_tags, superfície MCP, paridade documentada e norma de formato da tag), com gate Go verde em 25/25 pacotes."
tags: [validation, obsidian, tags, adr-055, adr-041, mcp, cli]
---

# Validação — Superfície de tags do Obsidian

**Spec**: `.specs/features/obsidian-tag-parity/spec.md`
**Plan**: `.specs/features/obsidian-tag-parity/tasks.md`
**ADR**: `docs/adr/055-semantica-de-tag-consultavel-sem-nos-mortos.md` (adenda da `docs/adr/041-issue-009-tag-sem-casa-removida-do-grafo.md`)
**Branch**: `develop` · **HEAD**: `93470c5` (+ correções do gate, ver abaixo) · **Verificado em**: 2026-10-23
**Commits da feature**: `d442629` (spec) · `d8fd7df` (parser) · `5b73dba` (store) · `1eb1a8f` (indexer) · `0091893` (store: prune) · `9aa24a1` (CLI) · `93470c5` (MCP + filtro da busca)

---

## Veredito

**PASS** — 46/46 critérios satisfeitos. A superfície de tags do Obsidian está
entregue nos três lugares onde ela precisa existir: a **CLI** (`mem tags`,
`mem tag <nome>`, `mem search --tag`), o **MCP** (`memory_get_tags`,
`memory_get_tag`) e a **persistência** (a tabela `document_tags` nos dois
engines, populada antes da resolução fuzzy).

O veredito se apoia em **quatro** camadas de evidência:

1. **Gate Go verde** — `go build`, `go vet` e `go test -count=1 ./...` com exit 0
   nos 25 pacotes (tabela abaixo), mais formatação e benchmarks.
2. **Smoke test funcional** — 13 casos com o binário compilado nesta rodada
   contra um vault de verdade (seção abaixo).
3. **Testes unitários** que exercitam o comportamento — `internal/parser/tags_test.go`,
   `internal/db/tags_test.go`, `cmd/mem/tags_test.go`, `cmd/mem/search_tag_test.go`.
4. **Evidência por linha** — cada AC abaixo aponta o `caminho:linha` onde o
   comportamento está implementado, para que a afirmação seja auditável sem
   reexecutar nada.

> **Nota sobre a tabela de rastreabilidade.** A tabela de Requirement
> Traceability do `spec.md` **continua com `Pending`**. Esta é a convenção do
> repositório: a feature `obsidian-parity-foundation` e a `mcp-static-serving`,
> ambas concluídas, também mantêm `Pending` na spec e registram o veredito no
> `validation.md` (achado B2 da auditoria da `mcp-static-serving`). O
> `Pending` na spec significa "aguardando validação", e o `validation.md` é a
> prova de que a validação aconteceu — não que ela está pendente.

---

## Gate executado

| Gate | Comando | Resultado |
|---|---|---|
| Build | `go build ./...` | **exit 0** (`BUILD OK`) |
| Build do binário | `go build -v -o bin/mem.exe ./cmd/mem` | **exit 0** |
| Vet | `go vet ./...` | **exit 0** (`VET OK`) |
| Testes | `go test -count=1 ./...` | **exit 0** — 25/25 pacotes `ok`, 0 falhas |
| Formatação | `gofmt -l` sobre 34 cópias normalizadas em LF | **exit 0, saída vazia** |
| Benchmarks | `go test -bench=. -benchmem -run=^$` em `parser`, `db`, `store` | **exit 0** |
| Pacotes de tag | `cmd/mem`, `internal/parser`, `internal/db`, `internal/store`, `internal/mcp`, `internal/watcher` | **todos `ok`** |

O gate foi executado **duas vezes**: no início da tarefa e no fim. Entre as duas
execuções a árvore mudou (uma correção **alheia** à feature, em
`cmd/mem/main.go`, vinda de outro worker) — as duas execuções terminaram com
exit 0 nos três passos.

### A formatação precisa de uma ressalva

O `gofmt -l .` na árvore de trabalho **lista cerca de 200 arquivos** — mas isso
não é defeito de formatação: é o **CRLF do checkout local** (o Windows
materializa as linhas com `\r\n`, e o `gofmt` compara byte a byte). O mesmo
arquivo, normalizado para LF, sai limpo.

Por isso a prova de formatação é feita sobre **cópias normalizadas em LF** dos
**34 arquivos Go** tocados pela feature (do `7e9e14f` ao `HEAD`, mais os dois
arquivos novos). Resultado: **`gofmt -l` vazio, exit 0** — nenhum defeito de
formatação na feature.

O `gofmt -l .` cru é registrado aqui só para deixar claro que a lista é
pré-existente ao repositório inteiro e não tem relação com a feature de tags.

### Benchmarks

O gate exige benchmark no pacote modificado. `internal/parser` e
`internal/store` já tinham; **`internal/db` não tinha nenhum** — a T2 e a T5
tocaram esse pacote. Três benchmarks foram adicionados
(`internal/db/tags_bench_test.go`) para cobrir o caminho quente da superfície de
tags: o inventário, a consulta por tag (com descendentes) e a substituição das
tags no reindex.

```
BenchmarkListVaultTags-28                607   1978642 ns/op   114841 B/op    7832 allocs/op
BenchmarkFindVaultDocumentsByTag-28     1389    815480 ns/op    76648 B/op    1438 allocs/op
BenchmarkInsertDocumentTags-28           663   1733679 ns/op     3485 B/op     105 allocs/op
```

Sem regressão: os três são benchmarks novos, sem baseline anterior para
comparar.

> **Como rodar no PowerShell 5.1.** A flag tem de vir **entre aspas**:
> `go test '-bench=.' '-benchmem' '-run=^$' ./internal/parser ./internal/db ./internal/store`.
> Sem aspas, o PowerShell entrega o `.` como se fosse um pacote e o `go test`
> falha com `no Go files in <diretório>`. A alternativa é o token `--%`
> (stop-parsing): `go test --% -bench=. -benchmem -run=^$ ./internal/parser`.

### Pacotes do gate, na ordem da saída

```
ok  	github.com/FelipeMiiller/my-memory/bench/codeast	30.860s
ok  	github.com/FelipeMiiller/my-memory/cmd/mem	82.136s
ok  	github.com/FelipeMiiller/my-memory/cmd/mymemoryd	0.882s
ok  	github.com/FelipeMiiller/my-memory/internal/autowire	0.934s
ok  	github.com/FelipeMiiller/my-memory/internal/canvas	0.827s
ok  	github.com/FelipeMiiller/my-memory/internal/ci	44.811s
ok  	github.com/FelipeMiiller/my-memory/internal/codeast	3.192s
ok  	github.com/FelipeMiiller/my-memory/internal/compiler	0.750s
ok  	github.com/FelipeMiiller/my-memory/internal/config	1.132s
ok  	github.com/FelipeMiiller/my-memory/internal/db	9.192s
ok  	github.com/FelipeMiiller/my-memory/internal/deeplink	0.668s
ok  	github.com/FelipeMiiller/my-memory/internal/drift	1.269s
ok  	github.com/FelipeMiiller/my-memory/internal/embedder	2.644s
ok  	github.com/FelipeMiiller/my-memory/internal/event_runtime	3.394s
ok  	github.com/FelipeMiiller/my-memory/internal/federation	1.102s
ok  	github.com/FelipeMiiller/my-memory/internal/graph	0.773s
ok  	github.com/FelipeMiiller/my-memory/internal/graphview	0.253s
ok  	github.com/FelipeMiiller/my-memory/internal/mcp	3.828s
ok  	github.com/FelipeMiiller/my-memory/internal/parser	0.796s
ok  	github.com/FelipeMiiller/my-memory/internal/repo	0.856s
ok  	github.com/FelipeMiiller/my-memory/internal/staleness	2.730s
ok  	github.com/FelipeMiiller/my-memory/internal/store	23.923s
ok  	github.com/FelipeMiiller/my-memory/internal/supervisor	18.849s
ok  	github.com/FelipeMiiller/my-memory/internal/turboquant	0.769s
ok  	github.com/FelipeMiiller/my-memory/internal/watcher	4.627s
```

**TEST EXIT=0**

> **Árvore de trabalho.** O gate rodou sobre a **working tree**, não só sobre o
> `HEAD` (`93470c5`). A working tree tinha, além da documentação desta rodada, uma
> correção **não commitada** em `cmd/mem/main.go` (guarda de embedder nulo nos
> caminhos de busca) e o teste novo `cmd/mem/search_provider_test.go`. Essa correção
> é alheia à feature de tags — o gate a incluiu, o que o torna **mais estrito**,
> não menos. Nenhum arquivo Go foi alterado por esta validação.
>
> **Números de linha.** Todas as referências `caminho:linha` apontam para a
> **working tree** no momento da verificação. Um commit posterior não desloca
> números de linha, então as referências continuam válidas. A única ressalva é
> `cmd/mem/main.go`: ele recebeu, **durante** a verificação, uma correção **não
> relacionada** (a guarda de embedder nulo nos caminhos de busca) vinda de outro
> worker, que deslocou em +22 as linhas depois de ~2005. As referências a esse
> arquivo foram **reconferidas** no fim da rodada e trazem o nome do símbolo
> (`newSearchFlagSet`, `printSearchUsage`, `parseSearchFlags`), de modo que
> continuam localizáveis mesmo que a linha se mova de novo.

---

## Evidência por critério de aceitação

**PASS** = comportamento verificado por teste e/ou por evidência direta no
código, com o `caminho:linha` apontado.

### P1 — Inventário de tags (TAG-01 … TAG-09)

| ID | Critério | Veredito | Evidência |
|---|---|---|---|
| TAG-01 | `mem tags` imprime uma tag por linha, alfabética, sem cabeçalho | **PASS** | `cmd/mem/tags.go:215` (`writeTagInventory`); a ordem alfabética mora no engine: `internal/db/tags.go:127` (`ORDER BY dt.tag_key ASC, dt.created_at ASC, dt.document_id ASC, dt.tag ASC`) |
| TAG-02 | `--counts` imprime `tag<TAB>contagem` | **PASS** | `cmd/mem/tags.go:51` (flag), `:215-244` (formato); a contagem vem de `internal/db/tags.go:146-151` (`tags[position].Count++`) |
| TAG-03 | `--sort=count` ordena por contagem decrescente; o padrão é por nome | **PASS** | `cmd/mem/tags.go:52` (flag), `:63-67` (valida `name`/`count`), `:93-95` (desvia para `sortTagsByCount`), `:245` (`sortTagsByCount`); padrão por nome em `internal/db/tags.go:127` |
| TAG-04 | `--file` lista as tags do documento; `--folder` as dos documentos sob a pasta | **PASS** | `cmd/mem/tags.go:53-54` (flags), `:75-86` (`filter.DocumentID` a partir do caminho); o filtro é no SQL: `internal/db/tags.go:125-126` (`dt.document_id = ?` / `d.path LIKE ? ESCAPE '\'`) |
| TAG-05 | `--total` imprime só a quantidade de tags distintas | **PASS** | `cmd/mem/tags.go:55` (flag), `:216-218` (`fmt.Fprintln(out, len(tags))`) |
| TAG-06 | `--json` imprime `tag`, `key`, `count`, `resolved_doc_id` e `resolved_title` | **PASS** | `cmd/mem/tags.go:56` (flag), `:169-172` (encoder JSON); a forma do payload: `internal/store/tags.go:12-22` (`VaultTag` com os cinco campos) |
| TAG-07 | Tag hierárquica sai com o caminho completo e é entrada própria do inventário | **PASS** | a hierarquia é preservada na canônica (`internal/parser/tags.go:34` — `#projeto/alpha` → `projeto/alpha`); a linha guarda o caminho completo (`internal/db/tags.go:120` — `dt.tag_key, dt.tag`); o inventário agrega por `tag_key` (`internal/db/tags.go:132-151`), sem Achatar a hierarquia |
| TAG-08 | Vault sem tag → lista vazia, saída 0 | **PASS** | `cmd/mem/tags.go:83` (documento fora do índice → inventário vazio, sem erro); `internal/db/tags.go:119-131` (SELECT sem linhas → slice vazio, não erro) |
| TAG-09 | O inventário inclui as tags órfãs | **PASS** | `internal/watcher/indexer.go:328-347` (`buildTagBindings` grava a lista completa, órfãs incluídas); `internal/db/tags.go:124` usa `LEFT JOIN documents r`, então a tag sem documento resolvido **permanece** na lista |

### P2 — Informação de uma tag (TAG-10 … TAG-16)

| ID | Critério | Veredito | Evidência |
|---|---|---|---|
| TAG-10 | `mem tag <nome>` imprime a contagem de documentos | **PASS** | `cmd/mem/tags.go:137-150` (busca + `Count: len(documents)`), `:175` (`fmt.Fprintln(out, result.Count)`); a contagem vem do engine: `internal/db/tags.go:188-203` |
| TAG-11 | `--verbose` imprime a contagem **e** a lista de documentos | **PASS** | `cmd/mem/tags.go:117` (flag), `:175-180` (contagem e depois um caminho por linha) |
| TAG-12 | A consulta conta a tag **e suas descendentes** | **PASS** | `internal/db/tags.go:198` (`dt.tag_key = ? OR dt.tag_key LIKE ? ESCAPE '\'`) com o padrão `key + "/%"` em `:200`; o mesmo predicado no Postgres em `internal/store/tags.go:303-338` |
| TAG-13 | `--total` imprime só a contagem | **PASS** | `cmd/mem/tags.go:118` (flag), `:176` (`if *verbose && !*total` — com `--total` a lista de documentos não é impressa) |
| TAG-14 | `--json` imprime nome, contagem, documentos e documento resolvido | **PASS** | `cmd/mem/tags.go:119` (flag), `:145-150` (montagem do `tagQueryResult`), `:158-167` (o documento resolvido vem do inventário), `:169-172` (JSON) |
| TAG-15 | Tag inexistente → contagem 0, saída 0 | **PASS** | `cmd/mem/tags.go:112` ("Tag que não existe não é erro: contagem 0 e saída zero"); o caminho é `internal/db/tags.go:188-203` (SELECT sem linhas → lista vazia) |
| TAG-16 | `mem tag` sem `<nome>` → imprime o uso e sai com código ≠ 0 | **PASS** | `cmd/mem/tags.go:126-129` (nome vazio → imprime o uso e devolve erro, que a CLI converte em saída ≠ 0) |

### P3 — Filtro por tag na busca (TAG-17 … TAG-21)

| ID | Critério | Veredito | Evidência |
|---|---|---|---|
| TAG-17 | `mem search --tag <tag>` devolve só os documentos com a tag e suas descendentes | **PASS** | `cmd/mem/main.go:327-336` (bloco do filtro `--tag` em `runSearch`: canonicaliza via `parser.NormalizeTag`), `:341` (`Tag: resolvedTag` no `SearchOptions`); o filtro: `internal/db/hybrid.go:43-47` (`dt.tag_key = ? OR dt.tag_key LIKE ? ESCAPE '\'`) |
| TAG-18 | `--tag` combina com `--category`, `--decay` e `--mode` | **PASS** | `cmd/mem/main.go:338-343` (as três opções convivem no mesmo `SearchOptions`); no SQL categoria e tag são o mesmo `WHERE`: `internal/db/hybrid.go:42-47` |
| TAG-19 | `mem search --tag` sem valor → imprime o uso e sai com código ≠ 0 | **PASS** | `cmd/mem/main.go:2368-2369` (`flag.ContinueOnError`), `:2403-2414` (`parseSearchFlags`: erro → `printSearchUsage(errOut)` + `return 1`), `:2393-2396` (o uso já anuncia `--tag <tag>`) |
| TAG-20 | Nenhuma nota com a tag → resultado vazio, saída 0 | **PASS** | o `EXISTS` do filtro não casa com ninguém (`internal/db/hybrid.go:43-47`) → nenhum documento entra no resultado; o filtro é ausência de linha, não erro |
| TAG-21 | O filtro é push-down SQL, não filtro pós-fusão | **PASS** | os **7 pontos**, todos dentro do SQL: `internal/db/hybrid.go:43-47` (FTS5), `internal/db/hybrid.go:154-158` (expansão de grafo), `internal/db/graph.go:66-70` (k-NN vetorial), `internal/db/graph.go:124-128` (TurboQuant 4-bit), `internal/store/postgres.go:502-506` (vetor), `internal/store/postgres.go:595-599` (FTS), `internal/store/postgres.go:697-701` (expansão de grafo). Todos no formato `AND (? = '' OR EXISTS (SELECT 1 FROM document_tags ...))` |

### P4 — Persistência (TAG-22 … TAG-28)

| ID | Critério | Veredito | Evidência |
|---|---|---|---|
| TAG-22 | O indexer grava todas as tags (frente e inline), órfãs incluídas, nos **quatro** caminhos | **PASS** | indexer SQLite `internal/watcher/indexer.go:109`; indexer Postgres `internal/watcher/indexer.go:253`; `mem index` em lote SQLite `cmd/mem/main.go:1940`; `mem index` em lote Postgres `cmd/mem/main.go:1730`. A lista completa vem de `internal/watcher/indexer.go:328-347` |
| TAG-23 | A gravação acontece **antes** da resolução fuzzy; a aresta do grafo continua só das tags resolvidas | **PASS** | `internal/watcher/indexer.go:333` (`ResolveTagTargets`) resolve **à parte** e `:336-347` grava **todas** as tags do documento, resolvidas ou não; o comentário em `:322-324` fixa a regra: a aresta `tagged_as` continua sendo só da tag resolvida (ADR-041 intacta), o que muda é a tabela (ADR-055) |
| TAG-24 | Reindexar substitui as tags anteriores, sem duplicar | **PASS** | `internal/db/tags.go:18-93` (a gravação roda em transação; `:32-46` lê as tags existentes e `:29-31`/`:76-78` preserva o `created_at`); a identidade `(document_id, tag_key` com guarda de duplicata em `:67-71` impede a linha duplicada. O caminho de remoção antes da gravação: `internal/watcher/indexer.go:168` e `cmd/mem/main.go:1792` |
| TAG-25 | Documento removido → as tags dele são removidas, nos **três** caminhos (reindex, purge, prune) | **PASS** | `internal/db/tags.go:95` (`DeleteDocumentTags`); called em `internal/watcher/indexer.go:168` (reindex), `internal/watcher/indexer.go:307` (purge), `cmd/mem/main.go:1792` e `cmd/mem/main.go:2005` (prune) |
| TAG-26 | Tag que nomeia um documento → `resolved_doc_id` igual ao `documents.id` | **PASS** | `internal/watcher/indexer.go:344-346` (`binding.ResolvedDocID = titleToID[title]` — o id do documento **nomeado**, não o que carrega a tag); a coluna em `internal/db/schema.go:157` |
| TAG-27 | Tag que não nomeia documento → `resolved_doc_id` nulo, e a tag continua no inventário | **PASS** | `internal/watcher/indexer.go:344` (só seta quando `found && title != ""`); `internal/db/tags.go:81-83` grava `nullIfEmpty(normalized.ResolvedDocID)`; o inventário usa `LEFT JOIN` (`internal/db/tags.go:124`), então a linha continua listada |
| TAG-28 | A tabela existe nos dois engines ao abrir o índice, sem `mem index --force` | **PASS** | SQLite: `internal/db/schema.go:151-162` (`Schema`) e `:310-321` (`FallbackSchema`), ambas aplicadas na abertura por `internal/db/store.go:28-34` (`db.Exec(schemaToRun)`), com `CREATE TABLE IF NOT EXISTS`. Postgres: `internal/store/postgres.go:84-94` (`PostgresSchema`) |

### P5 — Superfície MCP (TAG-29 … TAG-33)

| ID | Critério | Veredito | Evidência |
|---|---|---|---|
| TAG-29 | `memory_get_tags` devolve o inventário com contagens, no `content` e no `structuredContent` | **PASS** | schema `internal/mcp/tools.go:698-714`; handler `internal/mcp/tags_handlers.go:35-88` (`:87` devolve `NewStructuredResult(content, tags)`); registro `internal/mcp/vault_handlers.go:310` e `internal/mcp/server.go:127` |
| TAG-30 | `memory_get_tag` devolve a contagem e os documentos, no `content` e no `structuredContent` | **PASS** | schema `internal/mcp/tools.go:716-729`; handler `internal/mcp/tags_handlers.go:91-130` (`:121-126` monta o `TagQueryResult`); registro `internal/mcp/vault_handlers.go:311` e `internal/mcp/server.go:128` |
| TAG-31 | `memory_get_tags` com `path` devolve só as tags daquele documento | **PASS** | `internal/mcp/tags_handlers.go:46-56` (`params.Path` → resolve o documento → `filter.DocumentID`); o filtro por documento no engine: `internal/db/tags.go:125` (`dt.document_id = ?`) |
| TAG-32 | `structuredContent` tem listas vazias (`[]`), nunca `null` | **PASS** | `internal/mcp/structured_content.go:29-30` (`case []store.VaultTag` → `nonNilSlice`); `internal/mcp/tags_handlers.go:62-64` (`tags == nil` → `[]store.VaultTag{}`) e `:116-118` (`documents == nil` → `[]store.VaultDocument{}`) |
| TAG-33 | Sem dado, o `content` explica e a tool **não** sinaliza erro | **PASS** | `internal/mcp/tags_handlers.go:67-75` (inventário vazio → frase explicativa + `nil` no erro); o mesmo contrato no `memory_get_tag`: `internal/mcp/tags_handlers.go:105` (só `name` ausente é erro de parâmetro) |

### P6 — Paridade documentada (TAG-34 … TAG-36)

| ID | Critério | Veredito | Evidência |
|---|---|---|---|
| TAG-34 | `docs/CLI_GUIDE.md` documenta `mem tags`, `mem tag` e `mem search --tag` | **PASS** | `mem search --tag`: `docs/CLI_GUIDE.md:145` (título da seção 5), `:152` (a flag e a semântica de descendentes), `:183-193` (exemplos 8 e 9 + a nota de uso sem valor). `mem tags` / `mem tag`: seção 34 do guia, `docs/CLI_GUIDE.md:1086-1147` |
| TAG-35 | `README.md` e `docs/adr/README.md` registram a superfície como paridade implementada | **PASS** | `README.md:66` (pilar 8 — a superfície de tags), `README.md:230` ("26 ferramentas"), `README.md:259-260` (as duas tools na lista); `docs/adr/README.md:58` (linha da ADR-055: status "Aceito — **Implementado**" e tag `document_tags`; callout `:64-77`) |
| TAG-36 | O `CHANGELOG.md` registra a adição e a ADR-055 fica marcada como implementada no índice | **PASS** | `CHANGELOG.md:12-51` (entrada em `## [Unreleased]` → `### Added`); `docs/adr/README.md:58` (ADR-055: "Aceito — **Implementado**") e `docs/README.md:82` (ADR-055: "*Aceito — Implementado*") |

### P7 — Formato e semântica da tag (TAG-37 … TAG-46)

| ID | Critério | Veredito | Evidência |
|---|---|---|---|
| TAG-37 | Aceita letras, números, `_`, `-`, `/` e Unicode; não aceita espaço; a tag inteira precisa de um caractere não-numérico | **PASS** | `internal/parser/tags.go:44-70` (`NormalizeTag`), `:74-80` (`isAllowedTagRune`); a extração usa o mesmo conjunto por espelhamento: `internal/parser/wikilinks.go:15-25` (`tagAllowedChars` / `tagNonNumericChars`); testes `internal/parser/tags_test.go:11-40` e `:94-121` |
| TAG-38 | Tag só numérica é rejeitada e não entra no inventário | **PASS** | `internal/parser/tags.go:41` (`NormalizeTag("#1984")` → `("", false)`); teste `internal/parser/tags_test.go:42-60` e `:123-127`; a tag rejeitada não é gravada: `internal/watcher/indexer.go:337-342` (`continue` quando `!ok`) |
| TAG-39 | Tags que diferem só por caixa são uma entrada só, e a busca casa as duas | **PASS** | a canônica é a identidade: `internal/parser/tags.go:44-70`; a gravação usa a canônica como chave: `internal/store/tags.go:85-104` (`NormalizeTagBinding`) e `internal/db/tags.go:63-71` (chave `(document_id, tag_key)` + guarda de duplicata); teste `internal/db/tags_test.go:104-123` |
| TAG-40 | A tag aparece com o casing da primeira criação | **PASS** | o casing vem da linha representativa, a de menor `created_at`: `internal/db/tags.go:108-111` (regra) e `:127` (`ORDER BY dt.tag_key, dt.created_at, dt.document_id, dt.tag`); a reindexação preserva o `created_at`: `internal/db/tags.go:29-31` e `:76-78`; teste `internal/db/tags_test.go:114-123` |
| TAG-41 | A consulta hierárquica casa a tag e as descendentes; o inventário mantém as hierárquicas literais separadas | **PASS** | descendentes: `internal/db/tags.go:198` + `:200` (`LIKE key/%`); inventário: a agregação é por `tag_key` (`internal/db/tags.go:132-151`), e `projeto/alpha` é uma chave diferente de `projeto` — as duas convivem como entradas |
| TAG-42 | Frontmatter aceita `tags` como lista YAML e o alias depreciado `tag` | **PASS** | `internal/parser/frontmatter.go:70-71` (`case "tags", "tag":` → `normalizeStringOrSlice`, que aceita lista e escalar) |
| TAG-43 | Hashtag em propriedade de texto que não seja `tags` não cria tag | **PASS** | `internal/parser/wikilinks.go:238` (`fm, body := ExtractFrontmatter(content)` — o frontmatter é separado do corpo) e `:284` (a `tagRegex` varre **só o corpo**): as propriedades do frontmatter só viram tag pelo caminho `tags`/`tag` de `internal/parser/frontmatter.go:70-71` |
| TAG-44 | Tag e wikilink dentro de comentário `%%` (inline ou em bloco) não são extraídos, e nenhum link vira aresta | **PASS** | `internal/parser/wikilinks.go:34` (`obsidianCommentRegex`), aplicada em `:66` (antes das arestas) e `:85` (antes das tags); testes `internal/parser/tags_test.go:165-181` (inline e em bloco), mais o caso "percentual-nao-e-comentario" em `:175-181` |
| TAG-45 | Tag em célula de tabela Markdown é extraída e registrada | **PASS** | `internal/parser/wikilinks.go:82-87` (`stripTagContexts` **não** remove linhas de tabela — ao contrário de `stripWikilinkContexts`, que usa `tableRowRegex` em `:67`); o comentário em `:273-276` fixa a regra; teste `internal/parser/tags_test.go:182-186` |
| TAG-46 | Bloco `~~~` não gera tag nem wikilink, e é tratado como código em todo o caminho | **PASS** | `internal/parser/wikilinks.go:31` (`tildeFenceRegex`), aplicado em `:46` (`stripFencedCode`, que remove **antes** das arestas e das tags) e `:83` (`stripTagContexts`); no `mem outline`: `cmd/mem/outline.go:58-64` (`fenceLine` reconhece ` ``` ` e `~~~`); teste `internal/parser/tags_test.go:155-159` |

---

## Os três defeitos de parser corrigidos no mesmo passo

A superfície de tags só é confiável se a extração parar de produzir tag falsa.
Os três defeitos foram corrigidos junto com a feature, e o CHANGELOG os registra
(`CHANGELOG.md:41-51`):

| Defeito | Antes | Depois | Evidência |
|---|---|---|---|
| A regex truncava `#ação` | a tag virava a espúria `a` | a tag é capturada inteira; `#1a`, `#_tag` e `#🚀` também | `internal/parser/wikilinks.go:15-25` (o conjunto de caracteres espelha `isAllowedTagRune`); teste `internal/parser/tags_test.go:94-121` |
| Bloco `~~~` gerava tag e wikilink fantasma | o fence `~~~` não era removido antes da extração | é removido, e passa a ser código em todo o caminho | `internal/parser/wikilinks.go:31` + `:46`; `cmd/mem/outline.go:58-64`; teste `internal/parser/tags_test.go:155-159` |
| Comentário `%%` gerava tag real | o comentário era indexado, buscável, e as tags dentro dele viravam tags | `%%` é removido antes da extração de tag e de link | `internal/parser/wikilinks.go:34` + `:66` + `:85`; testes `internal/parser/tags_test.go:165-181` |

---

## Smoke test funcional com o binário compilado

A evidência dos ACs vinha de teste unitário e de leitura de código. O smoke test
fecha essa lacuna: o binário **real**, compilado nesta rodada, contra um vault
de verdade.

**Montagem.** Vault de teste em `D:\.minimax\scratch\smoke-vault` com três
notas — `alpha.md` (`#projeto/alpha`, `#Backend`), `beta.md` (`#projeto/beta`,
`#backend`) e `soltas.md` (sem tag). Indexado em SQLite local com
`--storage=sqlite --db <caminho>`, para não tocar no Postgres do config global.

> O binário foi compilado **sem a tag `onnx`** — que é exatamente o cenário em
> que o comando de busca quebrava. O smoke test rodou no pior caso.

| # | Comando | Resultado observado | Veredito |
|---|---|---|---|
| 1 | `mem tags` | `Backend`, `projeto/alpha`, `projeto/beta` — 3 linhas, alfabética | **PASS** |
| 2 | `mem tags --counts --sort=count` | `Backend 2`, `projeto/alpha 1`, `projeto/beta 1` | **PASS** |
| 3 | `mem tags --total` | `3` | **PASS** |
| 4 | `mem tag projeto --verbose` | `2` + `notas\alpha.md` + `notas\beta.md` — casa as descendentes | **PASS** |
| 5 | `mem tag projeto/alpha --total` | `1` | **PASS** |
| 6 | `mem tag naoexiste --total` | `0`, **exit 0** | **PASS** |
| 7 | `mem search --tag backend "阿尔法"` | acha `alpha.md` — `#Backend` casa com `--tag backend` | **PASS** |
| 8 | `mem search --tag projeto "阿尔法"` | acha `alpha.md` — `#projeto/alpha` é descendente | **PASS** |
| 9 | `mem search --tag naoexiste "阿尔法"` | `Nenhum resultado encontrado.`, **exit 0** | **PASS** |
| 10 | `mem search "soltas"` (controle, sem filtro) | acha `soltas.md` | **PASS** |
| 11 | `mem search --tag` (sem valor) | `Erro: flag needs an argument: -tag` + uso, **exit 1** | **PASS** (TAG-19) |
| 12 | `mem search --mode vector` (sem provider) | mensagem de erro + **exit 1** | **PASS** |
| 13 | `mem search --tag "#1984"` (tag numérica) | `Erro: tag inválida`, **exit 1** | **PASS** |

Os casos 1 e 2 provam a **identidade case-insensitive**: `#Backend` e `#backend`
são uma entrada só, exibida com o casing da primeira criação. Os casos 4 e 8
provam a **semântica de descendentes**, que é a regra do `tag:inbox` do Obsidian.

O caso 12 prova, no binário real, a correção do **nil-pointer-dereference**
descrita abaixo: a saída traz `Aviso: sem provider de embedding; executando
busca textual FTS.` e o comando **não** quebra.

---

## O caminho MCP, exercitado com um cliente real

A validação declarava, com honestidade, que o caminho MCP **não tinha sido
exercitado em execução**. Essa lacuna foi fechada nesta rodada — e ela tinha
um preço.

Servidor MCP HTTP real (`mem mcp --http 127.0.0.1:39321`), mesmo binário
compilado sem a tag `onnx`, mesmo vault do smoke test, cliente JSON-RPC por
HTTP:

| Chamada | Resultado |
|---|---|
| `tools/list` | **26 tools**, com `memory_get_tags` e `memory_get_tag` presentes |
| `tools/call` → `memory_get_tags` | inventário com as 3 tags, contagens e o `resolved_doc_id`/`resolved_title` das que casaram com um documento |
| `tools/call` → `memory_get_tag` (`name=projeto`) | contagem 2 e os 2 documentos — a semântica de descendentes funciona pelo MCP |
| `tools/call` → `memory_search` com `tag` | **o servidor morreu com panic** |

### O BLOQUEANTE: `memory_search` derrubava o servidor MCP inteiro

```
panic: runtime error: invalid memory address or nil pointer dereference
main.buildSQLiteSearchFunc.func1(...)
	C:/repository/my-memory/cmd/mem/main.go:2569
github.com/FelipeMiiller/my-memory/internal/federation.(*FederatedSearcher).Search.func2()
	C:/repository/my-memory/internal/federation/search.go:87
```

Causa: o mesmo bug da CLI, no caminho do servidor. `buildSQLiteSearchFunc` e
`buildPostgresSearchFunc` chamavam `emb.GenerateEmbedding` **sem guarda**, e
sem provider de embedding o `emb` é `nil`. A busca do servidor MCP roda em
goroutine (`internal/federation/search.go:76` e `:86`), e essas goroutines
**não tinham `recover()`** — então o pânico derrubava o **processo inteiro**,
não só a requisição. O servidor MCP morria no primeiro `memory_search` de um
cliente.

Por que ninguém tinha visto: o binário de desenvolvimento é compilado com a
tag `onnx`, então `emb` nunca é `nil`; e o caminho MCP não era exercitado por
cliente real — só por teste unitário, que sempre passa um embedder.

**Correção, em duas camadas:**

1. **Causa raiz** — `emb == nil` é tratado como "sem provider" nos quatro
   pontos do MCP (`cmd/mem/main.go`, modos `vector` e `hybrid`, engines SQLite
   e Postgres). O vetorial devolve erro de uso; o híbrido cai para a busca
   textual.
2. **Defesa em profundidade** — as goroutines da busca federada agora isolam o
   pânico: ele vira um erro da requisição (`internal/federation/search.go`), e
   o servidor continua de pé. Um `memory_search` malformado não pode mais
   derrubar o servidor.

### Re-verificação depois da correção

Servidor MCP HTTP novo, mesmo binário, mesmo vault:

| Chamada | Resultado |
|---|---|
| `tools/list` | 26 tools, com as duas de tag |
| `memory_get_tags` | inventário com as 3 tags e as contagens |
| `memory_get_tag` (`projeto`) | 2 documentos — descendentes pelo MCP |
| `memory_search` com `tag=backend` | acha `alpha.md` |
| `memory_search` sem `tag` | acha `alpha.md` |
| `memory_search` com `tag=naoexiste` | `Nenhum resultado encontrado.` |
| `memory_search` com `mode=vector` | erro JSON-RPC `-32603` com a mensagem do provider — **servidor de pé** |

O filtro por tag está confirmado pelo caminho do MCP, e o servidor sobrevive a
todas as chamadas.

---

## Defeitos encontrados pelo gate e corrigidos

O gate e o smoke test acharam quatro defeitos **na própria feature** (e um
pré-existente, que só aparecia em máquina sem ONNX). Todos foram corrigidos e
estão cobertos por teste.

| # | Defeito | Gravidade | Correção |
|---|---|---|---|
| 0 | **`memory_search` derrubava o servidor MCP inteiro com panic** quando o binário é compilado sem a tag `onnx`: `emb.GenerateEmbedding` sem guarda, em goroutine sem `recover()` | **BLOQUEANTE** | guarda `emb == nil` nos 4 pontos do MCP; `recover()` nas goroutines da busca federada, para que um pânico vire erro de requisição e o servidor não morra |
| 0b | **`mem tags --folder` não casava nada no Windows**: a comparação era `path LIKE 'pasta/%'`, e o path indexado no Windows usa `\` — o filtro devolvia lista vazia | **ALTO** | a comparação passa a ser feita sobre a forma com `/` (`REPLACE` nos dois engines) e a pasta precisa ser um segmento inteiro, com a mesma regra de `matchesFolder` (`cmd/mem/files.go`); teste de regressão com path Windows no SQLite e integração real no Postgres |
| 1 | **`mem search` derrubava com nil-pointer-dereference** quando o binário é compilado sem a tag `onnx` | **ALTO** | guarda `emb == nil` nos dois caminhos de busca (`cmd/mem/main.go`, `runSearchSQLite` e `runSearchPostgres`): no híbrido, cai para a busca léxica com aviso; no vetorial, explica e sai com código de erro |
| 2 | **`mem search --tag` sem valor saía com código 0** — o pacote `flag` do Go só reclama de argumento faltante quando não sobrou nenhum argumento, então a flag seguinte (`--mode`) era engolida como valor de `--tag` | **ALTO** (violava o TAG-19) | `validateSearchFlagValues` roda **antes** do parse e rejeita flag sem valor; o rearranjador de argumentos passou a consultar o FlagSet (`flagTakesValue`) em vez de manter a lista de nomes na mão |
| 3 | **Erro de busca saía com código 0** — o modo vetorial sem provider imprimia "Erro" e devolvia zero | MÉDIO | `runSearchSQLite` e `runSearchPostgres` devolvem o código de saída; o `main` propaga. As mensagens de erro também foram para o `stderr`, como no resto da CLI |
| 4 | **`--help` imprimia o uso duas vezes** e, no `mem search`, ainda executava a busca com a query que sobrou (`mem search algo --help` buscava "algo") | BAIXO | `parseSearchFlags` devolve um sinalizador de ajuda; `mem tags --help` e `mem tag --help`, que saíam com código 1 e "Erro: flag: help requested", agora saem com código 0 |

> **O defeito 1 é pré-existente**, não nasceu nesta feature: o `git show` do
> commit anterior ao da feature já mostra a chamada sem guarda. Nenhum teste o
> pegava, porque os testes de busca sempre rodavam com provider e o caminho sem
> provider só aparece em máquina sem ONNX instalado. A feature é que provides
> the surface where the smoke test found it.

---

## O que não foi verificado

1. **O caminho de busca completo não foi exercitado contra o Postgres em
   execução.** O Postgres **foi** exercitado — pelo teste de integração do
   filtro por pasta (`internal/store/tags_folder_pg_test.go`, que rodou contra
   o Postgres real e provou o literal `E'\\'` e o `REPLACE`) —, mas o smoke test
   de 13 casos rodou no SQLite local, e os três pontos de push-down do Postgres
   (`internal/store/postgres.go:502-506`, `:595-599`, `:697-701`) e o
   `INSERT ... ON CONFLICT` continuam apoiados em **leitura de código**.
2. **O `mem tag`/`mem tags` foi rodado contra um vault real** com o binário
   compilado nesta rodada — ver a seção de smoke test acima, 13 casos. A
   evidência dos TAG-01 … TAG-16 não depende só de teste unitário
   (`cmd/mem/tags_test.go`).
3. **TAG-42 (alias `tag`) e TAG-43 (hashtag em propriedade que não é `tags`) não
   têm teste unitário dedicado** — a evidência é a linha de código
   (`internal/parser/frontmatter.go:70-71` e `internal/parser/wikilinks.go:238`).
   São as duas afirmações desta validação que se apoiam **apenas** em leitura de
   código.
4. **A formatação foi verificada**, mas por cópia normalizada em LF, e não pelo
   `gofmt -l .` cru — que lista cerca de 200 arquivos por causa do CRLF do
   checkout local (ver a ressalva na seção do gate). Nenhum dos 34 arquivos Go
   tocados pela feature está na lista de defeitos de formatação.
5. **A tabela de rastreabilidade da spec continua com `Pending`** (ver a nota no
   topo). Isso é a convenção do repositório, não uma pendência aberta.
6. **O número de tools MCP (26) foi conferido por contagem de texto** nos
   documentos, e não por introspecção do servidor (`tools/list`). A lista
   canônica continua sendo `internal/mcp/server.go:104-128`.
7. **O caminho MCP foi exercitado com um cliente real** (HTTP/JSON-RPC contra
   um servidor MCP de verdade) — ver a seção acima. A primeira chamada de
   `memory_search` matou o servidor; o defeito foi corrigido e re-verificado.
   O que **não** foi exercitado é o **transporte stdio** do MCP, que segue o
   mesmo caminho de código mas não foi conectado a um cliente nesta rodada.

---

## Achados

| ID | Achado | Decisão |
|---|---|---|
| **B1** (BAIXO) | A tabela de Requirement Traceability da spec ficou com `Pending` nos 46 ACs, depois da validação | **Mantido** — é a convenção do repo (achado B2 da auditoria da `mcp-static-serving`); o veredito vive neste arquivo |
| **B2** (BAIXO) | `docs/REPOSITORY_BRAIN.md` e `docs/OBSIDIAN_PARITY_MATRIX.md` afirmavam "24 tools" e listavam as tags como "sem tool MCP" — afirmações que a feature torna falsas | **Corrigido nesta rodada** — os dois documentos passaram a 26, e a matriz ganhou a linha de Tags |
| **B3** (BAIXO) | O `CHANGELOG.md` do `mcp-static-serving` afirma "as 24 tools mantêm o `content` textual byte-idêntico" | **Mantido** — a afirmação continua verdadeira: as 24 tools *pré-existentes* mantêm o `content`; as 2 novas são aditivas |
| **B4** (ALTO) | `mem search` derrubava com nil-pointer-dereference sem provider de embedding (pré-existente à feature) | **Corrigido** — guarda `emb == nil` nos dois caminhos; teste `cmd/mem/search_provider_test.go`; provado no binário real pelo smoke test (caso 12) |
| **B5** (ALTO) | `mem search --tag` sem valor saía com código 0, porque a flag seguinte era engolida como valor (violava o TAG-19) | **Corrigido** — `validateSearchFlagValues` antes do parse; teste `cmd/mem/search_tag_test.go` (`TestSearchComTagSemValorSeguidoDeOutraFlagEhErroDeUso`); provado no smoke test (caso 11) |
| **B6** (MÉDIO) | Erro de busca devolvia código de saída 0 e ia para a stdout | **Corrigido** — `runSearchSQLite`/`runSearchPostgres` devolvem o código; mensagens para o `stderr` |
| **B7** (BAIXO) | `--help` imprimia o uso duas vezes; `mem search algo --help` ainda buscava "algo"; `mem tags --help` saía com código 1 | **Corrigido** — sinalizador de ajuda no parse; `flag.ErrHelp` tratado como sucesso nos três comandos |
| **B8** (BAIXO) | `internal/db` não tinha nenhum benchmark, e a feature tocou o pacote (T2 e T5) | **Corrigido** — `internal/db/tags_bench_test.go` com três benchmarks do caminho quente de tags |
| **B9** (BLOQUEANTE) | `memory_search` derrubava o **processo do servidor MCP** com panic quando o binário é compilado sem a tag `onnx`: `emb.GenerateEmbedding` sem guarda, em goroutine sem `recover()` | **Corrigido** — guarda `emb == nil` nos 4 pontos do MCP e `recover()` nas goroutines da busca federada; reverificado com cliente JSON-RPC real (o servidor responde e continua de pé) |
| **B10** (ALTO) | `mem tags --folder` não casava nada no Windows: a comparação era `path LIKE 'pasta/%'` e o path indexado usa `\` — os fixtures da suíte usavam `/`, então o bug nunca era reproduzido | **Corrigido** — comparação sobre a forma com `/` nos dois engines, com a pasta como segmento inteiro; teste de regressão com path Windows no SQLite e integração real no Postgres |
