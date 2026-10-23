---
title: "Matriz de Paridade com o Obsidian"
category: resource
summary: "Levantamento completo da distância entre a superfície do Obsidian e o My-Memory, por área, com evidência caminho:linha. Data base 2026-10-22, fontes primárias obsidian.md/help. Substitui a matriz de 2026-10-18 citada na ADR-054, que nunca foi registrada no repositório."
tags: [obsidian, parity, gap-matrix, adr-054, roadmap]
---

# Matriz de Paridade com o Obsidian

- **Data base**: 2026-10-22
- **Branch**: `develop` (HEAD `7e9e14f`)
- **Escopo**: a ADR-054 decide o que é paridade. Este documento mede a distância.
- **Método**: leitura estática do código, com auditoria independente em três
  frentes (sintaxe/parser · links/embeds/aliases/PDF/propriedades ·
  CLI/MCP/documentação). Nenhuma afirmação abaixo é sem `caminho:linha`.

## Fontes primárias

Toda a coluna "Obsidian" vem das páginas oficiais, lidas em 2026-10-22:

| Fonte | URL |
| ----- | --- |
| Basic formatting syntax | `obsidian.md/help/syntax` |
| Internal links | `obsidian.md/help/links` |
| Tags | `obsidian.md/help/tags` |
| Properties | `obsidian.md/help/properties` |
| Embed files | `obsidian.md/help/embeds` |
| Aliases | `obsidian.md/help/aliases` |
| PDFs | `obsidian.md/help/pdf` |
| Obsidian CLI (lista de comandos) | `obsidian.md/help/cli` |

## Como ler a matriz

| Status | Significado |
| ------ | ----------- |
| **COBERTO** | Capacidade equivalente ou superior, com evidência |
| **PARCIAL** | Existe, mas falta uma variação que o Obsidian tem |
| **AUSENTE** | Não existe no código |
| **FORA** | Fora de escopo por decisão (ADR-054:35) — app/editor, não conteúdo de vault |

A coluna **Destino** dá o próximo passo: **Construir** (dentro do escopo da
ADR-054), **Adiar**, **Recusar** (fora de escopo) ou **Decidir** (exige o dono).

---

## 1. Matriz — sintaxe e parsing

| # | Capacidade | Obsidian | My-Memory | Evidência | Destino |
|---|-----------|----------|-----------|-----------|---------|
| S1 | Parágrafos / quebras de linha | ✅ | PARCIAL | `internal/parser/markdown.go:149-179` — `ChunkText` usa `strings.Fields`, colapsa todo whitespace; não há modelo de parágrafo | Adiar |
| S2 | Headings `#`–`######` | ✅ | PARCIAL | `cmd/mem/outline.go:27-56` extrai ATX, mas **fora do `internal/parser`**, sem persistência; `internal/parser` não tem nenhuma regex de heading | Construir (schema) |
| S3 | Setext heading (`===`) | ✅ | AUSENTE | `cmd/mem/outline.go:23-26` | Adiar |
| S4 | Bold / itálico / riscado | ✅ | PARCIAL | `internal/parser/markdown.go:12-14` — só no abstract L0; no texto indexado os `**` permanecem literais | Adiar |
| S5 | Realce `==x==` / `==🔴x==` | ✅ | AUSENTE | grep `==` em `internal/` → zero matches | Adiar |
| S6 | Callout `> [!info]` / `[-]` / `[+]` | ✅ | AUSENTE | grep `callout\|\[!` → zero matches; o marcador vai cru para o FTS e para o abstract (`markdown.go:78`) | Adiar |
| S7 | Task list `- [ ]` `[x]` `[?]` `[-]` | ✅ | AUSENTE | grep `\[x\]` → zero matches; nenhum estado no schema | Construir (schema) |
| S8 | Lists `-` `*` `+` `1.` `1)` | ✅ | PARCIAL | `internal/parser/markdown.go:78` — marcador cru no chunk; o cutset `">- *+0123456789."` do abstract é largo demais (`"1. Intro"` → `"Intro"`) | Adiar |
| S9 | Fenced code ```` ``` ```` com linguagem | ✅ | PARCIAL | `internal/parser/wikilinks.go:20` — a linguagem **nunca** é extraída | Adiar |
| S10 | Fenced `~~~` | ✅ | **PARCIAL (bug)** | `internal/parser/wikilinks.go:20` casa só ```` ``` ```` → `[[Nota]]` dentro de um fence `~~~` **vira aresta fantasma**; `cmd/mem/outline.go:61` trata os dois → inconsistência entre outline e parser | **Construir** (T1) |
| S11 | Fences aninhados | ✅ | PARCIAL | toggle por prefixo — um ```` ``` ```` dentro de ```` ```` ```` fecha o bloco prematuramente (`cmd/mem/outline.go:39`) | Adiar |
| S12 | Código indentado (tab/4 espaços) | ✅ | AUSENTE | `internal/store/orthography.go:261-262` só no doctor de ortografia | Adiar |
| S13 | Inline code `` `x` `` | ✅ | PARCIAL | `internal/parser/wikilinks.go:21,39` — removido só da extração de link/tag; o **conteúdo** entra inteiro no chunk indexado | Adiar |
| S14 | Footnotes `[^1]` / `^[texto]` | ✅ | AUSENTE | grep `footnote` → zero matches; o marcador vira token `1` no FTS5 | Adiar |
| S15 | Comentários `%%inline%%` / `%%bloco%%` | ✅ (invisível) | **AUSENTE (bug)** | grep `%%` em `*.go` → zero matches → comentário é **indexado, buscável** e suas `#tag` **viram tags reais** (`wikilinks.go:228-235` roda sobre `cleanBody`, que não remove `%%`) | **Construir** (T1) |
| S16 | Escapes `\*` `\_` `\#` `` \` `` `\|` `\~` | ✅ | AUSENTE | `internal/parser/wikilinks.go:12,100` — sem lógica de escape; `[[Note\|Alias]]` produz target `"Note\"` | Adiar |
| S17 | Link Markdown `[t](url)` → aresta | ✅ (não é aresta no Obsidian também, mas é link) | AUSENTE | `internal/parser/markdown.go:10-11` — as regexes existem mas são usadas **só** por `CleanMarkdownText`; `ExtractConnections` usa só a regex de wikilink | Construir |
| S18 | Heading de nota por link `[[##header]]` / `[[^^block]]` | ✅ | AUSENTE | `internal/parser/wikilinks.go:116-118` + `:267-268` — `IsSameDoc` impede a aresta | Adiar |

## 2. Matriz — links, embeds, blocos, âncoras

| # | Capacidade | Obsidian | My-Memory | Evidência | Destino |
|---|-----------|----------|-----------|-----------|---------|
| L1 | `[[Note]]` | ✅ | COBERTO | `internal/parser/wikilinks.go:12,203` | — |
| L2 | `[[Note.md]]` | ✅ | PARCIAL | resolve por *acaso* (substring fuzzy: `"note"` ⊂ `"note.md"`) — `internal/parser/fuzzy.go:119-128` | Construir |
| L3 | `[[Pasta/Nota]]` | ✅ | PARCIAL | sem resolução por caminho; só fuzzy por substring, **ambíguo** com títulos sobrepostos (`internal/parser/fuzzy.go:119-128`) | Construir |
| L4 | `[[Note#Heading]]` | ✅ | PARCIAL | `Anchor` é preenchido (`wikilinks.go:123`) e **descartado** na criação da aresta (`:267-275`); `graph_edges` não tem coluna (`internal/db/schema.go:53-60`) | Construir (schema) |
| L5 | `[[Note#^bloco]]` | ✅ | PARCIAL | `BlockID` preenchido (`:120-121`) e descartado; **nada extrai** o `^id` do documento alvo (grep `BlockID` → só o parser) | Construir (schema) |
| L6 | `[[Note#H1#H2]]` | ✅ | AUSENTE | `wikilinks.go:110` divide só no **primeiro** `#` → `Anchor = "H1#H2"` | Adiar |
| L7 | `[[#Secao]]` (intra-nota) | ✅ | AUSENTE | `wikilinks.go:212-214,267-268` — `IsSameDoc` impede a aresta | Adiar |
| L8 | `[[Note\|Rótulo]]` | ✅ | PARCIAL | parser ok (`wikilinks.go:100-107`); o rótulo não persiste | Construir (schema) |
| L9 | Caracteres que quebram link (`# \| ^ : %% [[ ]]`) | ✅ | AUSENTE | `wikilinks.go:12` — a classe `[^\]\r\n]+` aceita todos eles; `[[a [[b]]` trunca em `[[a [[b` | Adiar |
| E1 | `![[Note]]` — embed de nota | ✅ | PARCIAL | `wikilinks.go:12` detecta o `!` (`IsEmbed`, `:61`), mas o índice grava uma aresta `links_to` **sem relação própria** (`:267-275` não checa `IsEmbed`) | Construir (schema) |
| E2 | `![[img.jpg\|640x480]]` | ✅ | **AUSENTE (bug)** | `wikilinks.go:100-107` — o valor vira **`Alias`** (teste `internal/parser/parser_test.go:106-108` confirma `Alias=="500"`) | Construir (parser) |
| E3 | `![[doc.pdf#page=3&height=400]]` | ✅ | AUSENTE | cai em `Anchor` cru (`wikilinks.go:109-124`); `#selection=` → zero hits | Decidir |
| E4 | `![[canvas.canvas]]` | ✅ | AUSENTE | vira target "canvas.canvas"; nada lê `.canvas` (só escrita, `internal/canvas/canvas.go:201-208`) | Decidir |
| E5 | **Embed de asset não cria dead link** | ✅ | **AUSENTE (bug)** | `internal/db/graph.go:237-243` — embed vira link para alvo inexistente → `doctor` reporta dead link, `doctor --fix` apaga a aresta, Health Score −5 por dead link (`internal/store/doctor.go:117-121`) | **Construir** |
| E6 | `[[Note#^lista]]` (embed de lista) | ✅ | PARCIAL | `BlockID` lido, não resolvido | Construir (schema) |

## 3. Matriz — aliases e propriedades

| # | Capacidade | Obsidian | My-Memory | Evidência | Destino |
|---|-----------|----------|-----------|-----------|---------|
| A1 | Propriedade `aliases` como lista | ✅ | PARCIAL | `internal/parser/frontmatter.go:72-73` lê (`aliases`/`alias`); não persiste | Construir (ADR-056) |
| A2 | `[[AI]]` resolve para a nota com alias `AI` | ✅ | **AUSENTE (bug)** | a resolução só conhece `documents.title` (`internal/db/store.go:810-811`, `internal/db/graph.go:435-508`); `AI` (2 chars) também falha no token match (exige ≥4, `internal/parser/fuzzy.go:133-148`) → **dead link** | **Construir** (ADR-056) |
| A3 | Comando `mem aliases` | ✅ | AUSENTE | nenhum `case "aliases"` em `cmd/mem/main.go:60-856` | Construir |
| A4 | Propriedades como objeto consultável (`obsidian properties`) | ✅ | AUSENTE | ADR-056 aceita (`docs/adr/056:39`) com a tabela `document_properties` — **a tabela não existe** em `internal/db/schema.go` nem em `internal/store/postgres.go` (grep `document_properties` → zero hits) | Construir (ADR-056) |
| A5 | Tipos de propriedade (Text/List/Number/Checkbox/Date/Date & time) | ✅ | AUSENTE | `internal/parser/frontmatter.go:86-88` → `Properties[k] = v` cru; `normalizeStringOrSlice` (`:100-129`) **descarta** número/bool/data em lista; não há `value_type` | Construir (ADR-056) |
| A6 | Propriedades padrão `tags`/`aliases`/`cssclasses` | ✅ | PARCIAL | `tags`/`tag` e `aliases`/`alias` lidos; `cssclasses` **não** lida (grep → zero hits) | Construir (ADR-056) |
| A7 | Hashtag em propriedade de texto não cria tag | ✅ | COBERTO | o frontmatter não é varrido por tag inline (`internal/parser/wikilinks.go:190-244` usa o `body`) | — |
| A8 | Propriedades aninhadas | ❌ (não suporta) | COBERTO (superset) | `frontmatter.go:56` (yaml.v3) aceita; usado de propósito para `relations` (`wikilinks.go:284`) | — |

## 4. Matriz — tags

| # | Capacidade | Obsidian | My-Memory | Evidência | Destino |
|---|-----------|----------|-----------|-----------|---------|
| T1 | Lista as tags do vault | ✅ `obsidian tags` | AUSENTE | nenhum `case "tags"` em `cmd/mem/main.go:60-856` | **Construir** (T4) |
| T2 | Consulta uma tag | ✅ `obsidian tag name=` | AUSENTE | idem | **Construir** (T4) |
| T3 | Filtra a busca por tag | ✅ `tag:` | AUSENTE | flag `--tag` não existe (`main.go:263-276`); promessa DR-5 da ADR-041 | **Construir** (T5) |
| T4 | Tag sobrevive à indexação | ✅ | AUSENTE | a tag é extraída (`wikilinks.go:235-244`) e **descartada** no passo fuzzy (`internal/parser/fuzzy.go:61-75`); a tabela `document_tags` **não existe** em nenhum motor | **Construir** (T2/T3) |
| T5 | Tag órfã aparece na lista | ✅ | AUSENTE | ADR-041 remove a aresta; sem tabela, a tag não é consultável | **Construir** (T2) |
| T6 | Hierarquia: `tag:inbox` casa `#inbox` e `#inbox/to-read` | ✅ | AUSENTE | sem semântica de prefixo | **Construir** (T4/T5) |
| T7 | Caracteres permitidos (letras, números, `_`, `-`, `/`, Unicode) | ✅ | PARCIAL | `wikilinks.go:15` — ASCII apenas, **primeiro caractere** obrigatoriamente letra | **Construir** (T1) |
| T8 | `#1a`, `#_tag`, `#ação`, `#Ω`, `#🚀` são válidas | ✅ | **AUSENTE (bug)** | a regex não captura; e `#ação` **trunca** para `"a"` (o `ç` está fora da classe) — cria tag espúria | **Construir** (T1) |
| T9 | `#1984` (só numérica) é inválida | ✅ | COBERTO (por acidente) | a regex exige letra inicial | Construir (T1, regra explícita) |
| T10 | Case-insensitive (`#tag` ≡ `#TAG`) | ✅ | **AUSENTE (bug)** | dedup case-sensitive (`wikilinks.go:230-233`); o frontmatter só remove o `#` (`frontmatter.go:92-94`) | **Construir** (T1/T2) |
| T11 | Exibição com o casing da primeira criação | ✅ | AUSENTE | nada guarda a forma canônica | **Construir** (T2) |
| T12 | Tag dentro de bloco de código não conta | ✅ | COBERTO | `stripCodeBlocks` roda antes da extração de tag (`wikilinks.go:228,235`) | — |
| T13 | Tag dentro de comentário `%%` não conta | ✅ | **AUSENTE (bug)** | comentário não é removido → vira tag real | **Construir** (T1) |
| T14 | Tag dentro de célula de tabela conta | ✅ | **AUSENTE (bug)** | `tableRowRegex` (`wikilinks.go:23`) remove as linhas `\|…\|` **antes** da extração de tag | Construir (T1) |

## 5. Matriz — assets, PDF e arquivos

| # | Capabilidade | Obsidian | My-Memory | Evidência | Destino |
|---|-----------|----------|-----------|-----------|---------|
| F1 | Indexa `.md` | ✅ | COBERTO | `internal/config/config.go:115-117` (`Include: ["**/*.md"]`) | — |
| F2 | Indexa `.pdf` / imagem / áudio / `.canvas` | ✅ | AUSENTE | nenhuma lib de PDF em `go.mod:5-14`; `cmd/mem/main.go:1504` só coleta `.md` | **Decidir** |
| F3 | Busca dentro do PDF | ✅ (desktop) | AUSENTE | grep `pdf\|pdftotext\|pdfcpu` → único hit é um teste (`internal/mcp/vault_handlers_test.go:231`) | **Decidir** |
| F4 | Link a página do PDF `#page=N` / seleção | ✅ | AUSENTE | §E3 | Decidir |
| F5 | Exporta nota para PDF | ✅ | AUSENTE | `mem export` (`cmd/mem/main.go:459-510`) só faz `--canvas` e `--html` do **grafo** | Decidir |
| F6 | Importa `.canvas` | ✅ | AUSENTE | só escrita (`internal/canvas/canvas.go:201-208`) | Decidir |
| F7 | Guarda contra binário no índice | — (não se aplica) | **AUSENTE (risco)** | `cmd/mem/main.go:1815-1820` e `internal/watcher/indexer.go:40-46,71-72` fazem `string(contentBytes)` sem detectar binário: se o usuário alargar o `include`, indexa lixo | **Construir** |

## 6. Matriz — comandos da CLI

A lista oficial do Obsidian tem **116 comandos**. Desses, a ADR-054:classifica
**~48** como *conteúdo de vault* (consulta) e o resto como *estado de aplicativo*
(fora de escopo). O My-Memory expõe **45** comandos (`cmd/mem/main.go:60-856`).

### Dentro do escopo (conteúdo de vault)

| Comando Obsidian | Status My-Memory | Evidência | Destino |
|---|---|---|---|
| `files`, `folders`, `file` | COBERTO | `main.go:672,678,720` · `cmd/mem/files.go:103-179` | — |
| `read` | COBERTO | `main.go:714` · `cmd/mem/read.go:16-59` | — |
| `outline` | COBERTO (ATX) | `main.go:702` · `cmd/mem/outline.go:114-150` | — |
| `wordcount` | PARCIAL | `main.go:708` — sem `words`/`characters` selecionáveis, sem escopo `file=`/`path=`, sem `--json` | Construir |
| `random` | COBERTO | `main.go:696` · `cmd/mem/random.go:35-78` | — |
| `random:read` | PARCIAL | `random.go:73-76` devolve path+título; conteúdo exige 2º passo | Construir |
| `recents` | COBERTO | `main.go:684` | — |
| `deadends` | COBERTO | `main.go:690` · `cmd/mem/deadends.go:21` | — |
| `search` | **SUPERIOR** | `main.go:261` — RRF híbrido (ADR-054:37) | — |
| `search:context` | **SUPERIOR** | `mem pack` `main.go:762` — orçamento de tokens | — |
| `backlinks`, `links` | PARCIAL (superior, mas sem comando isolado) | `cmd/mem/inspect.go:147-190` — só via tríptico | Construir |
| `unresolved` | PARCIAL | dentro de `mem doctor` (`main.go:3119+`), truncado em 15 (`main.go:3125-3127`); a ADR-054:33 pede "isolado" | Construir |
| `orphans` | PARCIAL | `main.go:3120-3129` — exclui docs by-design (`internal/store/doctor.go:51-54`) | Construir |
| **`tags`, `tag`** | **AUSENTE** | §T1–T2 | **Construir (T4)** |
| **`search --tag`** | **AUSENTE** | §T3 | **Construir (T5)** |
| `aliases` | AUSENTE | §A3 | Construir |
| `properties`, `property:read` | AUSENTE | §A4 | Construir |
| `property:set`, `property:remove` | FORA* | ADR-056:39 prevê; é escrita | **Decidir** |
| `tasks`, `task` | AUSENTE | sem tabela, sem comando, sem tool | Construir (schema) |
| `templates`, `template:read` | AUSENTE | sem índice de templates | Construir |
| `bases`, `base:views`, `base:query` | AUSENTE | sem leitura de `.base` | Decidir |
| `history`, `history:list`, `history:read` | AUSENTE | o índice guarda `updated_at`/`mtime`, mas não snapshot | Adiar |
| `move`, `rename` | FORA* | ADR-054:33 lista como "a construir"; ADR-054:35 não lista como fora — **contradição interna** | **Decidir** |
| `daily`, `daily:path` | AUSENTE | convenção de nome por data | Adiar |
| `unique` | AUSENTE | | Adiar |
| `vault` (info) | PARCIAL | `mem status` é sync/staleness (`cmd/mem/status.go:23-32`), não agregado name/path/files/folders/size | Construir |
| `vaults` | PARCIAL | `mem repos` lista repos federados, não vaults locais | Adiar |
| `create`, `append` | PARCIAL | `mem note create` (`cmd/mem/note.go:32-60`) — atômico, com frontmatter, não é criação de arquivo arbitrário | Adiar |
| `prepend`, `delete` | FORA | escrita | Recusar |

### Fora de escopo por decisão (ADR-054:35)

`plugins*` (9), `theme*` (5), `snippets*` (4), `commands`, `command`, `hotkeys`,
`hotkey`, `workspace*` (5), `tabs`, `tab:open`, `bookmarks`, `bookmark`, `sync*`
(7), `publish*` (6), `dev:*` (7), `devtools`, `eval`, `reload`, `restart`,
`search:open`, `template:insert`, `vault:open`, `web`, `daily:append`,
`daily:prepend`, `history:restore`, `history:open`, `base:create`.

## 7. Matriz — tools MCP

O servidor registra **26 tools** (`internal/mcp/server.go:98-128`). Cobertura da
superfície de vault:

| Capacidade | Tool MCP | Evidência |
| ---------- | -------- | --------- |
| Ler/listar notas | `memory_list_files`, `memory_read_note`, `memory_note_outline`, `memory_file_metadata` | `internal/mcp/tools.go:614-693` |
| Buscar | `memory_search` (sem filtro por tag) | `tools.go:16` |
| Gravar | `memory_write_note`, `memory_append_section`, `memory_compile_note` | `tools.go:248-392` |
| Grafo | `memory_get_neighbors`, `memory_get_hubs`, `memory_get_clusters`, `memory_get_impact`, `memory_inspect_node`, `memory_find_path`, `memory_get_insights`, `memory_get_drift`, `memory_doctor`, `memory_export_canvas`, `memory_visualize_graph`, `memory_pack_context`, `memory_open_node`, `memory_reconcile` | `tools.go:106-613` |
| Código | `memory_code_search`, `memory_code_neighbors` | `tools.go:74-129` |
| Tags | `memory_get_tags`, `memory_get_tag` | `tools.go:698-729` |

**Sem tool MCP**: aliases, properties, tasks, backlinks/links
dedicados, wordcount, random, folders, deadends, recents, `unresolved`/`orphans`
isolados.

## 8. Alegações infladas na documentação

Cada item é uma promessa que o código não cumpre. As correções estão no
CHANGELOG desta rodada.

| # | Onde | Promete | Reality | Severidade |
|---|------|---------|----------|-----------|
| A1 | `docs/CLI_GUIDE.md:1059-1062` | "Estas **nove** capacidades também estão expostas via MCP" e lista 4 tools | Só 4 dos 9 comandos de leitura de vault têm tool. `folders`, `wordcount`, `random`, `deadends`, `recents` não têm tool nenhuma | **ALTA** |
| A2 | `docs/adr/054-...:42` | "`mem tags` e `mem search --tag` deixam de prometer compatibilidade literal" — como se já existissem | `mem tags` **não existe** | **ALTA** — **resolvido** em 2026-10-22 pela feature `obsidian-tag-parity` (`mem tags`, `mem tag`, `mem search --tag` existem) |
| A3 | `docs/adr/054-...:76` | A matriz de gap está "registrada em `.specs/ISSUES.md`" | **Não está em lugar nenhum** do repo — este documento é a correção | **ALTA** |
| A4 | `docs/adr/054-...:10,12` | "O Obsidian expõe ~115 comandos" · "48 comandos de conteúdo de vault" + "os ~45 comandos restantes são de estado do aplicativo" | Aritmética inconsistente: 48 + 45 = 93, não ~115. A lista oficial tem **116** | **ALTA** |
| A5 | `docs/adr/054-...:33` | Escopo "a construir": `files`, `folders`, `read`, `wordcount`, `random`, `outline`, `deadends`, … | **7 dos 16 itens listados já existem** — a lista sobrestima o trabalho pendente | MÉDIA |
| A6 | `README.md:61` | "Extrai conexões explícitas de notas (`[[links]]` e `#tags`)" | `#tags` só vira aresta **se casar com um título** (`internal/parser/fuzzy.go:61-75`); sem tabela de tags, tags **não são consultáveis** | MÉDIA — **resolvido** em 2026-10-22: a tabela `document_tags` existe e a tag é consultável (`mem tags`, `mem search --tag`); a aresta continua só para a tag resolvida |
| A7 | `docs/REPOSITORY_BRAIN.md:141-149` | Lista de tools MCP "com escopo de repositório" (9 tools) | O servidor registra **26** | MÉDIA |
| A8 | `cmd/mem/note.go:54` | Erro manda usar "`mem write` (T11)" | **`mem write` não existe** — nenhum `case "write"`; referência órfã | MÉDIA |
| A9 | `docs/README.md:81` vs `docs/adr/README.md:58` | ADR-055 = *Proposed* vs *Aceito* | Divergência direta; o estado real é Aceito (2026-10-21) e **implementado** (2026-10-22) — os dois documentos já dizem *Aceito — Implementado* | BAIXA — **resolvido** |
| A10 | `cmd/mem/codeindex.go:34,42` | `--json` "Saída em JSON (placeholder)" | Flag parseada e **descartada** (`_ = jsonOut`) → no-op silencioso | BAIXO |

**Verificadas e NÃO infladas**: "26 tools MCP" (confere em `README.md:293`,
`AGENTS.md:99`, `skills/mymemory-mcp/SKILL.md:9`, `CHANGELOG.md`) e "45 comandos"
(ADR-054:10). O README **não** afirma "paridade total" nem "superset" — essa
alegação vive só na ADR-054:10 e é contradita pela própria ADR (:31, :63).

## 9. Já superior — não replicar

| Capacidade | Evidência |
| ---------- | --------- |
| Relações tipadas + status epistêmico + peso | `internal/parser/wikilinks.go:78-87` · `internal/db/schema.go:53-60` |
| Fuzzy resolve com normalização de acento PT-BR (`[[Memória]]` ↔ "Memoria") | `internal/parser/fuzzy.go:181-197` |
| Busca híbrida RRF + decaimento temporal por mtime real | `internal/store/rrf.go` · `internal/db/hybrid.go:13-33` |
| Deep links `obsidian://` + VS Code + arquivo | `internal/deeplink/deeplink.go:94-108` |
| URI federada `memory://<repo>/<path>#anchor` | `internal/deeplink/deeplink.go:41-51` |
| Tríptico inbound/outbound com severidade e PageRank | `internal/graph/inspector.go:164-190` |
| Export JSON Canvas 1.0 | `internal/canvas/canvas.go:131-208` |
| Tags órfãs **não** viram dead link (ADR-041) | `internal/parser/fuzzy.go:61-75` |
| Doctor de ortografia PT-BR com lista curada | `internal/store/orthography.go:196-253` |
| `mem doctor` repara self-loops e dead links | `internal/db/graph.go:401` · `internal/store/doctor.go:196` |
| Metadata de arquivo (path, title, size, mtime) — ADR-057 | `internal/store/vault.go:12-21` |
| `mem read` remonta o Markdown dos chunks (remove sobreposição) | `internal/store/chunks.go:20-36` |

## 10. Decisões que exigem o dono

Cinco decisões **não são minhas** e travam o roadmap:

1. **Assets não-Markdown**: indexar `.pdf`/imagem/áudio/`.canvas`, ou declará-los
   fora do escopo ("o My-Memory indexa só Markdown; assets são referenciados mas
   não ingeridos")? Isso decide se entra busca dentro de PDF, viewer e
   exportação de nota — e muda a migration (precisa de `asset_kind`/`mime`).
2. **`move`/`rename`**: a ADR-054:33 lista como "a construir" e a :35 lista
   `move`/`rename` fora de escopo. A ADR contradiz a si mesma.
3. **`property:set`/`property:remove`**: a ADR-056:39 prevê escrita de
   propriedade; é escrita, mas é o que torna `properties` útil.
4. **`unresolved`/`orphans` isolados**: a ADR-054:33 pede; hoje vivem dentro do
   `doctor` e truncados em 15 itens.
5. **Comentário `%%` no texto indexado**: removê-lo do FTS (paridade com o
   Obsidian, que esconde) muda resultados de busca existentes. Remover só da
   *extração* de tag/link é seguro; remover do índice é decisão de produto.

## 11. Roadmap derivado

Ordenado por (custo de schema ÷ valor), porque **cinco dos dez achados críticos
são a mesma migration**:

1. **Tags** — `document_tags` nos dois motores + `mem tags`/`mem tag` +
   `mem search --tag` + 2 tools MCP. *(ADR-055 já decidida; em execução.)*
2. **Alias e propriedades** — `document_aliases` + `document_properties` + colunas
   `anchor`/`block_id`/`is_embed` em `graph_edges`. Fecha de uma vez os
   achados L4, L5, E1, A1, A2, A4 e o dead link de embed (E5).
3. **Guarda de não-texto** no indexador — 1 linha de decisão, elimina o risco F7.
4. **Links Markdown** como aresta — as regexes já existem
   (`internal/parser/markdown.go:10-11`); é movê-las para `ExtractConnections`.
5. **Parser de parâmetros de embed** (`|WxH`, `#page=`, `#height=`) — separar os
   casos que hoje caem em `Alias`/`Anchor` cru.
6. **Tasks** — tabela + comando + tool (ADR-054:33).
7. **Templates, bases, history** — cada um com seu custo.

## 12. Procedência

Este documento **substitui** a matriz citada na ADR-054:76 (2026-10-18,
"3 cobertos · 13 parciais · 32 ausentes"), que nunca foi registrada no
repositório. O número de 2026-10-18 não é reproduzível a partir do código
atual e não é re-verificado aqui — a matriz acima é a medição vigente.
