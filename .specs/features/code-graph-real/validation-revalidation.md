# Revalidação pós-correção — `code-graph-real`

> **Cópia consolidada** do relatório do verificador independente (read-only),
> que roda fora do repositório em `D:\.minimax\scratch\subagents\revalidacao\`.
> A evidência bruta (binários, fixtures, logs) fica no scratch; o veredito e as
> provas ficam aqui.
>
> **Leitura do gestor:** as **8 correções D1..D8 sobreviveram** à refutação
> (8/8). Das 29 checagens, **26 confirmadas com evidência empírica**; 3 ficam
> por verificar por **limite de ambiente** (Postgres real), não por defeito.
> O achado residual **R-1** foi reproduzido e a análise do gestor está no
> `ISSUE-030`.

Verifier independente (read-only): sessão `mvs_7f50fda216dd4cc489ad94e95bf75b93`
Repo: `D:\repository\my-memory` · branch `develop` · HEAD `339d14eee118d43ea23086f8013eb7a6056325b7`
Data: 2026-10-26 (07:22 -> 07:40 BRT)

## Atualização 2026-10-28 — veredito revisto: 🟢 PASS

> Esta seção **supera** o veredito original de 2026-10-26 (registrado abaixo e
> repetido no fim do relatório). O **corpo do relatório continua intacto**, como
> registro do que era verificável na data da revalidação (HEAD `339d14e`).
> Nada foi reinterpretado: os dois itens que estavam abertos por limite de
> ambiente foram **fechados com prova empírica** depois.

**O que mudou desde 2026-10-26:** dois dos três itens que estavam abertos por
**limite de ambiente** foram fechados com prova — não por reinterpretação.

| Item | Em 2026-10-26 | Hoje | Evidência |
| :--- | :--- | :--- | :--- |
| **P1** — escrita real em Postgres: `ON CONFLICT (repository, path)`, índice único composto de `code_files`, `?`→`$1` em runtime real | **NÃO VERIFICÁVEL NESTE AMBIENTE** | ✅ **FECHADO** (2026-10-27) | Verificado contra um **PostgreSQL 16.15 real** (Neon), sempre em **banco descartável** criado e derrubado pelo próprio teste (`internal/pgtest`: `CREATE DATABASE mymemory_pgtest_*` + `DROP ... WITH (FORCE)`). Cada asserção foi provada **contra o defeito**. Detalhe: `ISSUE-031` → *P1* |
| **H4** (metade Postgres) — code graph gravado no engine configurado, dialeto certo | **PARCIAL** — SQLite confirmado, Postgres não verificável | ✅ **CONFIRMADA** — H4 passa a **16/16** confirmadas | `cmd/mem/codeindex_postgres_test.go::TestCodeIndexCLI_PostgresGravaCodeGraphNoEngineConfigurado` — a CLI real grava `code_files`/`code_symbols`/`code_edges` no **Postgres** (`"engine": "postgres"`), com `content_hash` SHA-256 e a aresta `calls` no **ponto de chamada** (linha 4, não a da definição, linha 3) |
| **P5** — colisão cross-repo no nó `code:<qualified_name>` | **REPRODUZIDO** (só em SQLite) | ✅ **CORRIGIDO** (2026-10-26, `c3c6267`) — decisão do dono: alinhar o SQLite com o Postgres; o id do nó passou a ser `code:<repository>:<qualified_name>` nos dois engines | Teste de regressão **provado contra o defeito** (com o fix revertido, falha com o sintoma exato do P5). Migração zero |
| **P4** — viewer sem MCP no ar: ramo "MCP indisponível" | **PARCIAL** — o ramo existe e está coberto por teste unitário (95/95 verdes), mas o **fluxo end-to-end** (viewer carregando `/code` com o MCP fora do ar) **não foi exercitado** | 🔶 **ÚNICO ITEM AINDA ABERTO** — **limite de ambiente** (exige subir o dev server e dirigir o browser), não defeito | `viewer/src/routes/code.tsx:196-201` e `:317-322`; `viewer/src/lib/mcp-client.test.ts`; `viewer/src/lib/code-view.test.ts` |
| **R-1** — achado residual: o fallback do `MockParser` só casa símbolos de mesmo nome simples | achado novo, **pré-existente** | 📌 **documentado** — virou **ISSUE-030**; o dono decidiu **deixar como está** e a documentação foi corrigida (`README.md`, `docs/CLI_GUIDE.md`, `docs/ARCHITECTURE.md`, `docs/adr/README.md`, `CHANGELOG.md`) | `internal/codeast/pipeline.go:607-622` |

### Veredito revisto (2026-10-28)

🟢 **PASS** — nenhuma das 16 hipóteses foi refutada, as 8 correções sobreviveram
à refutação, e os dois itens que dependiam do Postgres real foram **provados**
contra um PostgreSQL real em 2026-10-27. **Resta 1 item** (P4 end-to-end da
viewer), que é **limite de ambiente**: a lógica do ramo é coberta por teste
unitário verde.

O que foi provado em 2026-10-27 está em `.specs/ISSUES.md` → *ISSUE-031 → P1*; a
evidência de execução das duas fases está em
`.specs/features/code-graph-real/validation.md`.

---

## Veredito global (original, 2026-10-26): **PARTIAL**

As **8 correções D1..D8 sobreviveram** a refutação (Fase B: 8/8). Das 29 checagens,
**26 confirmadas com evidência empírica**. **3 itens ficam por verificar** por
limite de ambiente/acesso, não por defeito: **P1** (Postgres real), a **metade
Postgres de H4**, e o **ramo end-to-end de P4**. Um achado residual novo (**R-1**)
foi reproduzido e está detalhado no fim.

---

## Premissa inicial (medida, nao herdada)

```
$ git status --porcelain      -> (vazio)
$ git rev-parse HEAD          -> 339d14eee118d43ea23086f8013eb7a6056325b7
$ git log --oneline -12
  339d14e docs(spec): close the 8 defects and point the handoff to the revalidation
  aac0c0b fix(cli): precedencia do storage documentada e a fonte vencedora visivel
  e64e389 fix(viewer): a rota /code aponta para a ADR-061, nao a ADR-047 deferida
  aadd791 fix(drift): separar o hash estrutural do hash de cache
  b84dfce fix(cli): mem code-graph honra --repo e ganha --storage/--db/--postgres
  1dc845a fix(mcp): o corte por hub so vale em grafo grande
  b34abd8 fix(codeast): o build com a tag treesitter nao pode esvaziar o code graph
  fbd3d62 fix(mcp): propagar o repo_id do vault na busca com include_code
  b3ed53f fix(db): migrar code_edges_uncertain.repository em banco existente
```

Desvio do briefing: o texto original declarava HEAD `aac0c0b`; o HEAD real e
`339d14e`, commit **documental** sobreposto a `aac0c0b`. Os 8 commits de correcao
estao todos presentes e sao a base testada.

### Isolamento (regra 2 - Postgres do dono jamais tocado)
`MY_MEMORY_PG_URL` removido do escopo de **todo** processo filho;
`MY_MEMORY_GLOBAL_CONFIG_DIR` apontado para um diretorio vazio (o config global
real **tem** `postgres_url` - confirmado por booleano, **sem imprimir o valor**).
Todo armazenamento usado: `--storage=sqlite` / `--db=<arquivo em scratch>`.
Scan final: **0** ocorrencias de `postgres://`/`postgresql://`/`MY_MEMORY_PG_URL=`
em todos os `.log`/`.out`/`.err` do scratch.

### Artefatos de teste (fora do repo)
`scratch\bin\mem.exe` e `scratch\bin\mem-ts.exe` (binarios compilados **fora** do
repo). `scratch\fx\` = fixture de ataque; `scratch\mirror\` = copia LF dos 361
`.go` do HEAD (via `git archive`); `scratch\fxbig\` = grafo de 201 simbolos.

### Fixture de ataque (`scratch\fx\`)
`alpha.go` (pkg.Alpha -> Beta, def na 4 / chamada na 5) - `beta.go` (pkg.Beta ->
pkg.Gamma, def na 4 / chamada na 5, Gamma def na 9) - `a_one/one.go` (one.Shared,
one.Unique) - `b_two/two.go` (two.Shared) - `z_ambig.go` (chama `Shared` -> 2
candidatos) - `z_infer.go` (chama `Unique` -> 1 candidato) - `z_missing.go` (chama
`NeverExists` -> 0 candidatos) - `broken/bad.go` (Go quebrado) - `py/main.py`,
`py/two.py` (nao-Go) - `notes/alpha.md` (documento que cita `pkg.Alpha`).

---

# Fase A - revalidacao das 16 hipoteses H1..H16

| # | Hipotese | Veredito | Evidencia empirica (comando + saida) |
| :-- | :--- | :--- | :--- |
| **H1** | backend AST e Go puro; MockParser = caminho prod. p/ nao-Go; tag treesitter nao degrada | **CONFIRMADA** | Binario **sem CGO**: stderr `code AST: backend go (stdlib); tree-sitter: disabled`; indexou 5 arquivos/6 simbolos. Nao-Go caiu no MockParser: `code_files` tem `('py/main.py','python')` e 3 simbolos (`alpha`,`beta`,`gamma`) com `qualified_name=NULL`. Tag `treesitter`: resultado **identico** (7/9/3/2) + aviso `tree-sitter: build tag ativa, mas o binding NAO esta implementado - o motor ativo e o backend Go (stdlib)`. Teste `TestBuildTagIsolatesTreesitter_BothTagsCompile` PASS. **Ressalva R-1** (ver fim). |
| **H2** | aresta `calls` gravada com o **ponto de chamada** | **CONFIRMADA** | `code_edges` id=1: `src=3 (pkg.Alpha, def linha 4) -> dst=5 (pkg.Beta)`, `start_line=5` = a **linha da chamada**, nao a da definicao (4). id=2: `pkg.Beta`(def 4)->`pkg.Gamma`(def 9) com `start_line=5`. No MCP, `call_site_file=alpha.go, call_site_line=5` e `dst_file=beta.go, dst_line=4`. |
| **H3** | `epistemic_status`+`confidence`; ambiguo vai p/ uncertain e **nunca** e promovido | **CONFIRMADA (3 niveis)** | `code_edges`: id1 `EXTRACTED/1.0`, id2 `EXTRACTED/1.0`, id3 `INFERRED/0.8` (src=11 pkg.Infers -> dst=2 one.Unique). `code_edges_uncertain`: id1 `pkg.Shared` **0.5** `ambiguo: 2 simbolos com o nome "Shared"`; id2 `pkg.NeverExists` **0.3** `destino nao encontrado no repositorio`. **Nenhum** `dst_symbol_id` de `code_edges` e `one.Shared`(1)/`two.Shared`(4) -> o ambiguo **nao** foi promovido. |
| **H4** | code graph gravado no **engine configurado**, dialeto certo (`?` vs `$1`) | **PARCIAL** | SQLite: `--json` -> `"engine": "sqlite"`, tabelas `code_*` no arquivo SQLite. Dialeto: `dialect.Bind` troca `?`->`$1` (codigo). **Postgres: nao verificavel** (regra 2 - ver P1). |
| **H5** | servidor MCP sobe; tools de codigo respondem | **CONFIRMADA** | `GET /health` -> `{"status":"healthy","name":"my-memory","protocol_version":"2024-11-05","default_repo":"acme","tools_count":26}`. `tools/list` -> 26 tools, incluindo `memory_code_search` e `memory_code_neighbors`. `memory_code_search` devolveu `pkg.Alpha` (alpha.go, linha 4). |
| **H6** | `structuredContent` traz epistemic_status/confidence/call_site_*/dst_* | **CONFIRMADA** | `memory_code_neighbors`: `epistemic_status=EXTRACTED`, `confidence=1`, `call_site_file=alpha.go`, `call_site_line=5`, `dst_file=beta.go`, `dst_line=4`; no vizinho de hop 2, `call_site_line=5`, `dst_line=9`. |
| **H7** | `include_code=true` muda o ranking; sem a flag o ranking e identico | **CONFIRMADA** | `memory_search "Alpha"`: sem a flag -> **1** resultado (a nota), `Fontes RRF: [fts:1]`; com `include_code=true` -> **2** resultados, `code:pkg.Alpha` em 1o com `Fontes RRF: [code:1]`. O ranking **muda de fato**. |
| **H8** | ponte documents<->code_symbols no grafo unico, idempotente | **CONFIRMADA** | `mem index --code-ast=true` -> `[code-stage] ponte notas-codigo: 2 aresta(s) em 1 documento(s)`. `graph_nodes`: `code:pkg.Alpha|code_symbol|Alpha`, `code:pkg.Beta|code_symbol|Beta`. `graph_edges`: `...alpha.md -> code:pkg.Alpha | references_code | EXTRACTED` e `-> code:pkg.Beta | references_code | INFERRED`. 2a rodada -> **mesmas 2 arestas**, `graph_nodes=3`, `graph_edges=2` -> idempotente. |
| **H9** | corte por hub: p99 + ADR-029 + ordem por hop + hint; **nao remove** o vizinho direto | **CONFIRMADA** | Grafo pequeno (3 nos): `hub_threshold=0`, `nodes_pruned=0`, hop 1 (`pkg.Beta`) **e** hop 2 (`pkg.Gamma`) intactos. Grafo grande (201 simbolos/400 arestas): `hub_threshold=2`, `nodes_pruned=200`, `truncated=true`, hint `... (200 no(s)-hub removido(s) (grau acima do percentil); 128 vizinho(s) fora do orcamento de 4000 tokens)`, **72 vizinhos de hop 1 preservados**, e `big.Hub` (maior grau) **ausente** da resposta. Ordem por hop confirmada (so hop 1 no corte). |
| **H10** | drift estrutural: attribution "bridge"; ignora cosmetico, detecta assinatura | **CONFIRMADA** | `go test -v -run 'Structural\|Drift' ./internal/drift/` -> `TestDriftDetecaMudancaEstrutural` PASS e `TestDriftSemAstHashNaoQuebra` PASS. O teste exige `attribution == "bridge"`, `note_ids`, e **exclui** o arquivo so-comentario de `structural_changes` **e** de `affected_files`. |
| **H11** | `mem lesson` grava no **cofre central**, frontmatter ok, indexado | **CONFIRMADA** | `mem lesson` (cwd = vault de repo) -> `Licao registrada no cofre central!` `Arquivo: ...\central\lessons\titulo-teste-verificador.md`. **0** pastas `lessons/` dentro do vault do repo. Frontmatter: `category: memory`, `created_at`, `updated_at`, `summary`, `title`. `Indexacao: indexed (6.0498ms)`; `documents` tem a linha com `category='memory'`. |
| **H12** | bundle viewer no orcamento (163.840 B gzip-9), zero CDN, rota `/code`, typecheck+build | **CONFIRMADA** | gzip-9 (python): `index-*.js` 144.881 + `index-*.css` 6.733 + 112 + 108 = **151.834 B <= 163.840** (dentro). **0** hits de CDN (`unpkg`/`jsdelivr`/`cdn.`). `npm run typecheck` exit 0 - `npm run test` **95/95** (5 arquivos) - `npm run build` exit 0. `git status` **vazio apos o build**. |
| **H13** | code-index/code-graph honram `--repo`; `--ast-hash=false` -> NULL; `--json` stdout puro | **CONFIRMADA** | `--ast-hash=false` -> `ast_hash` **NULL em 7/7** `code_files`. `--json` -> stdout faz `ConvertFrom-Json` **sem erro**; stderr separado (nada de `storage:` no stdout). `repository=acme` gravado em `code_files/code_symbols/code_edges`. (O `--repo` do `code-graph` em R5.) |
| **H14** | docs honestas (backend Go, MockParser, ADR-047 deferida "gatilhos 0") | **CONFIRMADA** | `docs/adr/README.md:102` "backend Go puro"; `:109` "`MockParser` (fallback deterministico da ADR-047)"; `:154-156` "3 gatilhos, hoje **0 de 3** disparados". `docs/CLI_GUIDE.md:719` "Nao ha CGO nem build tag"; `:723` MockParser. `docs/ARCHITECTURE.md:217` cita o boot log **exatamente** como observado (`code AST: backend go (stdlib); tree-sitter: disabled`). `README.md:64` idem. **Ressalva R-1** (ver fim). |
| **H15** | sem codigo morto; `newBackendParser()` vivo; gofmt provado por copia LF | **CONFIRMADA** | `gofmt -l` sobre a copia LF dos **361** `.go` do HEAD -> **vazio** (controle: worktree lista 187, artefato de `core.autocrlf=true`). `newBackendParser()` e chamado em producao (`pipeline.go:79`) e nos testes (`parser_test.go`, `treesitter_build_test.go`). Sem TODO/FIXME/panic no escopo (o grep inicial so achou falso-positivo de "me**todo**"). |
| **H16** | go build + go vet + go test verdes; gate front-end verde | **CONFIRMADA** | `go build ./...` -> `build_exit=0`; `go vet ./...` -> `vet_exit=0`; `go test -count=1 ./...` -> `test_exit=0`, **26/26 pacotes `ok`** (incluindo `internal/ci`, que **flakeou** na rodada anterior; e `cmd/mem` 87,6s). Front-end: typecheck 0 / 95 testes / build 0 (ver H12). |

---

# Fase B - tentativa de refutar as 8 correcoes D1..D8

**Resultado: 8/8 sobreviveram.** Nenhuma correcao foi refutada.

| # | Correcao | Hipotese de refutacao | Veredito | Evidencia |
| :-- | :--- | :--- | :--- | :--- |
| **R1** | D1 `b3ed53f` - migra `code_edges_uncertain.repository` | banco pre-feature quebra? a migracao e idempotente? | **CORRECAO SEGUIU** | Banco pre-feature **real**, reconstruido do schema de `1bf7e75~1` (via `git show`): `code_edges_uncertain` **sem** `repository` (`id,src_symbol_id,dst_candidate_name,kind,confidence,reason,file_id,start_line`). `mem code-index` contra ele -> exit **0**, sem erro; pos-migracao as 4 tabelas `code_*` ganharam `repository` (e `code_edges` ganhou `epistemic_status`); dados gravados com `repository='acme'`, incluindo as 2 linhas uncertain. **2a rodada: exit 0, `files_indexed: 0`** -> idempotente. `EnsureCodeTables` e chamado no caminho do code-index (`internal/codeast/pipeline.go:67`) e no MCP (`cmd/mem/main.go:2821`). |
| **R2** | D2 `fbd3d62` - propaga `repo_id` do vault na busca com `include_code` | repo != default faz o fan-in de codigo falhar? | **CORRECAO SEGUIU** | Vault com `repository: acme` (!= "default"). `memory_search "Alpha"` sem a flag -> **1** resultado, `Fontes RRF: [fts:1]`; com `include_code=true` -> **2** resultados, `code:pkg.Alpha` em 1o com `Fontes RRF: [code:1]`. O `repo_id` do vault (nao so a flag) e o que alimenta o filtro - exatamente a claim do fix. |
| **R3** | D3 `b34abd8` - build com tag `treesitter` nao esvazia o code graph | a tag sombreia o backend Go? | **CORRECAO SEGUIU** | Binario `-tags treesitter` vs padrao: **mesmos** numeros (7 arquivos / 9 simbolos / 3 arestas / 2 uncertain) e stderr explicito `tree-sitter: build tag ativa, mas o binding NAO esta implementado - o motor ativo e o backend Go (stdlib)`. `TestBuildTagIsolatesTreesitter_BothTagsCompile` PASS e `TestPipelineNoOpWhenTreesitterDisabled` PASS. |
| **R4** | D4 `1dc845a` - o corte por hub so vale em grafo grande | grafo pequeno poda o vizinho direto? grafo grande remove o relevante? | **CORRECAO SEGUIU** | Pequeno: `hub_threshold=0`, `nodes_pruned=0`, hop 1 **e** hop 2 intactos. Grande (201 simbolos): `hub_threshold=2`, `nodes_pruned=200`, `big.Hub` (maior grau) **ausente**, **72 vizinhos de hop 1 preservados**. Os dois limites do fix (`code_handlers.go:895-925`, `MinHubPruneNodes`) valem empiricamente. |
| **R5** | D5 `b84dfce` - `mem code-graph` honra `--repo` e aceita `--storage/--db/--postgres` | `--repo` e ignorado? as flags novas existem? | **CORRECAO SEGUIU** | `code-graph --repo=acme pkg.Alpha` -> acha e devolve hop 1 (`pkg.Beta`) e hop 2 (`pkg.Gamma`). `code-graph --repo=outro pkg.Alpha` -> `Erro: simbolo "pkg.Alpha" nao encontrado em code_symbols`, exit 1 -> o filtro por repo **e** aplicado. O `--help` lista `--storage`, `--db`, `--postgres`, `--repo`. `--storage=postgres` **sem** URL -> `engine=postgres sem URL: defina storage.postgres_url, use --postgres=<url> ou force --storage=sqlite` - **nenhuma conexao tentada**, nenhuma URL impressa. |
| **R6** | D6 `aadd791` - separa hash estrutural do hash de cache | o drift acusaria mudanca cosmetica? o cache quebrou? | **CORRECAO SEGUIU** | `go test -v -run HashSymbols ./internal/codeast/`: `TestHashSymbolsStructural_IgnoraPosicaoEVeAssinatura` **PASS** (hash estrutural ignora deslocamento de linha e muda na assinatura) e `TestHashSymbols_StableAndOrderIndependent` **PASS** (o hash de cache continua estavel/independente de ordem). No drift: `TestDriftDetecaMudancaEstrutural` **PASS** - assinatura detectada com `attribution="bridge"`; arquivo so-comentario **excluido** de `structural_changes` e de `affected_files`. |
| **R7** | D7 `e64e389` - viewer `/code` cita ADR-061 | sobrou ADR-047 na viewer? | **CORRECAO SEGUIU** | `viewer/src/routes/code.tsx:200` e `:321` -> `(ADR-061)`; `lib/code-view.ts:30` e `lib/queries.ts:234` -> ADR-061. Grep em `viewer/src/**`: **0** ocorrencias de ADR-047. Os dois ramos de erro ("MCP indisponivel") citam ADR-061. |
| **R8** | D8 `aac0c0b` - precedencia documentada + fonte visivel no stderr | a precedencia real mudou? o log polui o stdout? | **CORRECAO SEGUIU** | Precedencia real **inalterada** (`storage_source.go:34-51`: flag > env > config). Frase canonica unica em `AGENTS.md:47`, `docs/CLI_GUIDE.md` (`:1251`, `:1410`, `:1432`) e `docs/adr/README.md:236`. Log observado no **stderr**: `storage: sqlite (fonte: flag --storage=sqlite)`; `mem status --json` -> stdout faz `ConvertFrom-Json` **sem erro** e **sem** `storage:`. **Ressalva declarada:** `mem code-index`/`code-graph` **nao** logam a fonte - mas isso e uma divergencia **documentada** (`.specs/features/code-graph-real/validation-d7-d8.md:122-125`: resolvem o engine pelo *config*, e o code stage nao pode ser redirecionado pelo ambiente). Nao e regressao. |

---

# Fase C - riscos que "ninguem olhou" (P1..P5)

| # | Risco | Veredito | Evidencia |
| :-- | :--- | :--- | :--- |
| **P1** | escrita real em Postgres + `ON CONFLICT (repository,path)` + indice unico composto | **NAO VERIFICAVEL NESTE AMBIENTE** | Ha um Postgres **em escuta** em `127.0.0.1:5432` (processo `wslrelay.exe`), mas ele e o **store real do dono** (`MY_MEMORY_PG_URL` -> cofre central `repo_central`) - conectar violaria a regra 2. Nao ha Postgres **descartavel**: sem servico Windows registrado, sem `psql`/`pg_ctl`/`initdb`, sem `docker`. **Nao criei infraestrutura nova** (regra da tarefa). Fica **nao verificado**: `ON CONFLICT (repository, path)` no Postgres, o indice unico composto, e o `?`->`$1` em runtime real. |
| **P2** | idempotencia de `mem index` e `mem code-index` | **CONFIRMADA** | `code-index` rodada A -> `files_indexed:3, files_skipped:4`; contagens `code_files 7, code_symbols 9, code_edges 3, code_edges_uncertain 2`. Rodada B -> `files_indexed:0, files_skipped:7, symbols_total:0`; contagens **identicas** (delta 0). `mem index` 2a rodada -> `[cached] Nota sobre Alpha (inalterado)`, `0 indexados, 1 em cache`, e a ponte devolveu **as mesmas 2 arestas** (`graph_nodes=3`, `graph_edges=2`). |
| **P3** | `files_errors` forcado: arquivo Go quebrado vira erro visivel | **CONFIRMADA** | `code-index` em `fx\broken`: `"files_errors": 2`, `"files_indexed": 0`, exit **0** e stderr `[code-pipeline] bad.go: ParseFile bad.go: go_parser: parse bad.go: bad.go:3:14: expected ')', found '{' (and 1 more errors)`. O erro e **visivel**, nao some em silencio. No `mem index` o code stage tambem o reporta e **nao derruba** o index (`Concluido! 1 documentos processados`, exit 0). |
| **P4** | viewer sem MCP no ar: ramo "MCP indisponivel" exercitado, sem estourar | **PARCIAL** | O ramo existe e esta correto em `viewer/src/routes/code.tsx:196-201` e `:317-322` (`title="MCP indisponivel"`, hint ADR-061). Coberto por teste unitario: `mcp-client.test.ts` (`isError` -> `McpError`; connection error) e `code-view.test.ts` (texto "memory_code_search indisponivel" -> `data` nulo, sem estourar) - **95/95** testes verdes. **Nao exercitei** o fluxo end-to-end (viewer carregando `/code` com o MCP fora do ar): exigiria subir o dev server e dirigir o browser. |
| **P5** | colisao cross-repo no grafo (no `code:<qualified_name>` global) | **REPRODUZIDO (so em SQLite)** | Cenario: dois vaults/repos (`acme` e `globex`) no **mesmo** arquivo SQLite, ambos com `pkg.Alpha` em paths distintos. Resultado: `code_symbols` tem **so** a linha do *acme*; mas `graph_edges` tem **duas** linhas - a nota do repo A **e** a nota do repo B apontando para o **mesmo** no `code:pkg.Alpha` -> os dois repos colapsam no mesmo no. Causa: em SQLite `graph_nodes` **nao tem** coluna `repository` e o insert e `INSERT OR IGNORE` (`code_bridge.go:317-321`); `loadBridgeSymbols` so filtra por `repository` **no Postgres** (`code_bridge.go:353-357`). **No Postgres nao ha colisao**: `graph_nodes` tem `repository` e o conflito e `(repository, id)` (`code_bridge.go:311-313`), e `graph_edges` usa `(repository, source_id, target_id, relation)` (`:331-335`). O proprio codigo declara a assimetria (`code_bridge.go:91-92`: "a ponte e local ao vault; no PostgreSQL tudo e filtrado pelo `repository`"). |

---

# Achado residual novo - R-1 (baixa/media, **pre-existente**, nao e regressao D1..D8)

**O caminho legado `references` (MockParser) nao produz arestas para simbolos sem
`qualified_name`.**

Medido: 2 arquivos Python (`py/main.py`, `py/two.py`) -> 5 simbolos MockParser
(`alpha`,`beta`,`gamma`,`Beta`,`Gamma`), todos com `qualified_name = NULL`, e
**0** arestas `references` e **0** linhas em `code_edges_uncertain`. Nenhum par
por nome foi resolvido - nem entre `Beta`(Python) e `pkg.Beta`(Go), que tem o
**mesmo** nome.

Causa (`internal/codeast/pipeline.go:607-622`): o indice `byName` e chaveado pelo
`qualified_name` quando o simbolo **tem** um (`key = qn.String`) e pelo `name`
quando **nao** tem; mas a busca usa `s.QualifiedName` e, na ausencia dele,
`s.Name`. Um simbolo MockParser (sem qualified name) procura a chave `"Beta"`,
enquanto o alvo Go esta sob a chave `"pkg.Beta"` -> **0 candidatos** -> nenhuma
aresta. O caminho legado so acha algo quando **outro** simbolo tambem nao tem
qualified name (ai a chave bate por `name`).

Impacto sobre as docs: `README.md:64`, `docs/CLI_GUIDE.md:723` e
`docs/ARCHITECTURE.md:203-204` prometem que "as demais linguagens caem no
`MockParser` ... e ficam com as arestas `references` por nome - provenance
`INFERRED`". Na pratica, um simbolo nao-Go **nao** gera aresta por nome contra
um simbolo Go. **Nao** e regressao das correcoes D1..D8 (o `MockParser` e a
varredida por nome sao da ADR-047, pre-existentes) e **nao** afeta o backend Go
nem nenhum dos grafos multi-linguagem ja indexados - mas e uma divergencia
doc<->realidade que a rodada anterior nao mediu.

---

# Correcoes do proprio verificador (auto-auditoria)

Registradas porque tres conclusoes intermediarias minhas estavam erradas e foram
corrigidas antes de virar veredito:

1. **R1 "validado" com banco pre-feature falso.** A primeira extracao do schema
   antigo usou redirecionamento `>` do PowerShell, que grava **UTF-16** -> meu
   extrator achou 0 blocos SQL -> `old.db` ficou **vazio** e o `InitDB` criou
   tabelas **novas**. Esse teste nao provava nada. Refeito com extracao UTF-8 do
   schema real de `1bf7e75~1` (resultado no quadro R1).
2. **"a secao `code:` do config nao habilita o code stage" - falso, era o meu YAML.**
   Meu `config.yaml` tinha `sqlite_path: "D:\.minimax\..."`; em YAML com aspas
   duplas `\.` e escape **invalido** -> `yaml.Unmarshal` falhava -> o config caia
   no default. Com YAML valido (aspas simples) o code stage roda **so** com a
   secao `code:` no config, sem `--code-ast`. A "descoberta" era 100% artefato do
   meu fixture - **nao** foi reportada como defeito do produto.
3. **P5 "sem colisao" na primeira tentativa.** A primeira versao do fixture
   reindexava o **mesmo** path relativo em outro repo -> o upsert por path
   sobrescreveu e nao houve coexistencia. Refeito com paths distintos.
4. **"MockParser sem arestas"** - a primeira leitura sugeria que o caminho legado
   nem rodava; a causa real e a inconsistencia de chave do indice (R-1), nao a
   ausencia de execucao do caminho.

---

# Integridade (regra read-only)

```
$ git -C D:\repository\my-memory status --porcelain   -> VAZIO
$ git -C D:\repository\my-memory rev-parse HEAD        -> 339d14eee118d43ea23086f8013eb7a6056325b7
```
`git status` vazio **antes e depois** do build da viewer, dos 2 binarios
compilados e de todas as execucoes. Nenhuma URL de Postgres foi impressa
(0 hits). Servidores MCP que eu subi foram encerrados (pids 2280 e 30540).
Todo scratch em `D:\.minimax\scratch\subagents\revalidacao\`; nada em `%TEMP%`,
na raiz do workspace ou dentro do repo. Nenhum comando de exclusao de arquivo;
nenhum `gofmt -w`.

# Veredito global (original, 2026-10-26): **PARTIAL**

> **Revisto em 2026-10-28 → 🟢 PASS.** P1 e a metade Postgres de H4 foram
> fechados com prova empírica; P5 foi corrigido. Resta 1 item por limite de
> ambiente (P4 end-to-end da viewer). Ver a atualização no topo do relatório.

- **Fase A:** 14/16 confirmadas, H4 parcial (metade Postgres) - nenhuma **refutada**.
- **Fase B:** **8/8 correcoes sobreviveram** - nenhuma refutacao consumada.
- **Fase C:** P2 e P3 confirmadas, P4 parcial, P5 reproduzido (SQLite),
  **P1 nao verificavel** (limite de ambiente).
- **R-1**: achado residual novo (doc<->realidade no caminho legado do MockParser).
- **Nao verificavel neste ambiente:** (1) escrita real em Postgres com
  `ON CONFLICT (repository,path)` e indice unico composto; (2) `?`->`$1` em
  runtime Postgres; (3) ramo da viewer end-to-end com o MCP fora do ar.
