# Changelog

All notable changes to **my-memory** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [3.0.0] — 2026-10-28

### Added

- **Dois conceitos de memória — Cofre Central e Code Graph (ADR-061)**: o My-Memory passa a
  separar explicitamente **dois conceitos de memória**, servidos pelo **mesmo motor**: o
  **Cofre Central** (memória de aprendizagem, Markdown Obsidian Flavored) e o **Vault de
  Repositório** (o code graph do código). O motor é o mesmo nos dois — busca híbrida FTS +
  TurboQuant 4-bit + RRF, `epistemic_status` (ADR-011), PageRank, LPA + modularidade, blast
  radius, viewer SPA e federação por `repo_id`.
  - **Code graph com backend em Go puro**: o parser de produção é
    `internal/codeast/go_parser.go` — `go/parser` + `go/ast` + `go/token` da stdlib.
    **Sem CGO e sem build tag**: o binário padrão já indexa código. A ADR-047 (tree-sitter)
    **não** foi supersedida — o plano de deferral do binding real segue intacto (0 de 3
    gatilhos) e a tag `treesitter` continua isolada. O build com a tag **não degrada o
    code graph**: como o binding tree-sitter não está implementado, o binário avisa no
    log de boot (`tree-sitter: build tag ativa, mas o binding NAO esta implementado — o
    motor ativo e o backend Go (stdlib)`) e usa o **mesmo** backend Go do build padrão —
    o grafo sai idêntico. Antes, o build com a tag produzia um code graph **vazio, em
    silêncio** (1 símbolo, 0 arestas, `files_errors: 0`).
  - **Arestas estruturais**: o code graph deixou de ser só `references` por nome. Agora
    grava `calls` (com o **ponto de chamada** — arquivo:linha da expressão de chamada, não
    da definição), `imports`, `inherits`, `uses_type` e `references`. As arestas
    estruturais vêm do backend AST, que hoje entende **Go**; as demais linguagens seguem
    no `MockParser` (fallback determinístico da ADR-047) e ficam com as arestas
    `references` por **igualdade de nome** (`INFERRED`, sem ponto de chamada). A chave da
    busca é o `qualified_name` do símbolo quando existe e o `name` quando não — um
    símbolo do `MockParser` só casa com outro de mesmo nome simples **sem**
    `qualified_name`, então num código sem nomes duplicados o fallback produz poucas ou
    nenhuma aresta.
  - **Provenance por aresta**: `code_edges.epistemic_status` distingue `EXTRACTED`
    (resolvido no escopo do pacote) de `INFERRED` (resolvido só por nome), em linha com a
    ADR-011. O destino **ambíguo** vai para `code_edges_uncertain` com `confidence` e
    `reason` — e **nunca** vira aresta resolvida.
  - **Escopo de código próprio**: a seção `code:` do `.memory/config.yaml`
    (`code.include` / `code.exclude`) é **separada** do `include`/`exclude` global de notas.
    O `include` global é compartilhado com o pipeline Markdown, e incluir `**/*.go` nele
    indexaria código como nota — o pipeline Markdown agora ignora arquivo de código por
    definição.
  - **`mem index` roda o code stage**: por padrão quando há escopo de código configurado,
    com a flag `--code-ast[=true|false]`. A falha do code stage **não derruba** o
    `mem index` — o pipeline Markdown já rodou e permanece intacto.
  - **Code graph no engine configurado**: gravado no PostgreSQL quando o vault está no
    PostgreSQL, no SQLite quando está no SQLite (ADR-040), e filtrado por `repository`
    (`repo_id`) como o grafo de notas.
  - **Flags reais do `mem code-index`**: `--repo` vale de verdade (o `repository` viaja até
    as tabelas `code_*`), `--ast-hash[=false]` controla o armazenamento do `ast_hash`, e
    `--json` emite a saída estruturada (engine, repository e contagens) em **JSON puro no
    stdout** — o boot log vai para o stderr, para o cliente conseguir parsear.
  - **`mem code-graph` honrou as flags**: `--repo` era aceito e **descartado** (o valor
    sumia no `_ = repoSlug`); agora o repositório filtrado é o do argumento, e o comando
    ganhou `--storage`, `--db` e `--postgres` — como o `mem code-index`. Sem URL de
    Postgres, ele falha com erro claro e **sem** tentar conexão.
  - **Fonte do storage visível**: o CLI imprime no **stderr** `storage: <engine> (fonte:
    <origem>)` — de onde veio a configuração do engine (flag, env var ou config). A
    precedência real é **flag > env var > config** (`MY_MEMORY_PG_URL` vence o config
    global), e a frase canônica é a mesma em `AGENTS.md`, `docs/CLI_GUIDE.md` e
    `docs/adr/README.md`. O stdout do `--json` continua **JSON puro**.
  - **`--no-embed` saiu do CLI**: a flag era placeholder, sem comportamento (a v1 não
    embute símbolos). Quem usava recebe `flag provided but not defined: -no-embed`.
  - **Fan-in real de `code_symbols`**: `memory_search` com `include_code: true` deixou de
    ser placeholder — `code_symbols` entra no RRF como fonte lexical real, com boost CA-05
    em `qualified_name`. Sem a flag, o ranking é **idêntico** ao de antes.
  - **Tools MCP de código no servidor**: `memory_code_search` e `memory_code_neighbors` são
    injetadas no servidor MCP (CLI e daemon) e respondem de verdade — antes devolviam
    "indisponível". A resposta de `memory_code_neighbors` traz **provenance**
    (`epistemic_status`), **confiança** (`confidence`) e o **ponto de chamada**
    (`call_site_file`/`call_site_line`) de cada relação. As duas tools devolvem o JSON no
    `content` textual **e** preenchem o `structuredContent` do envelope MCP.
  - **Ponte notas ↔ código no grafo único**: cada símbolo vira um nó
    `graph_nodes.type='code_symbol'` (id `code:<repository>:<qualified_name>`) e cada
    documento que o
    menciona ganha uma aresta `graph_edges.relation='references_code'`, com status
    epistêmico (`EXTRACTED` para match por `qualified_name` ou caminho literal do arquivo,
    `INFERRED` para match por nome). O upsert é idempotente e não exige DDL novo. A ponte
    roda no code stage e no indexador de Markdown, e é o que permite ao
    `memory_code_neighbors` listar os documentos que citam um símbolo e ao blast radius
    alcançar as notas que dependem dele.
    - **O nó de código é escopado por repositório**: o id carrega o `repository`, e os
      símbolos da ponte são filtrados por `repository` também no SQLite. Sem isso, dois
      repositórios com o mesmo `qualified_name` colapsavam no **mesmo** nó do grafo e a
      nota de um apontava para o símbolo do outro — o Postgres nunca teve esse problema
      (a identidade do nó lá é `(repository, id)`).
  - **Rota `/code` na viewer**: a quinta visão do painel lista os símbolos de um
    repositório e, ao selecionar um, mostra os vizinhos estruturais com o **tipo da aresta**
    e o **ponto de chamada**. Bundle de **144.667 bytes gzip -9**, contra o orçamento de
    163.840 bytes (160 kB) do job `frontend` do CI. Sem CDN.
  - **`mem lesson`**: novo comando que grava uma lição aprendida no **cofre central**
    (`central_vault.path` da configuração global, ADR-040) como nota Markdown puro, com
    `category: memory`, `summary` (recuperação L0) e timestamp, indexada na hora. Flags:
    `--summary`, `--file`, `--content`, `--tag`/`--tags` e `--folder` (padrão: `lessons`).
    Sem cofre central resolvido, o comando **falha com erro claro e código de saída
    diferente de zero** — a lição nunca cai no vault do repositório.

- **Superfície de tags do Obsidian (ADR-055, adenda da ADR-041)**: a tag deixa de ser
  descartada na indexação e vira **dado consultável**. A promessa DR-5 da ADR-041
  (`mem search --tag X`) e a paridade com `obsidian tags` / `obsidian tag name=`
  ficam entregues.
  - **Tabela `document_tags`** nos dois engines (SQLite e PostgreSQL), com a identidade
    da linha em `(document_id, tag_key)`. Ela é populada **antes** da resolução fuzzy,
    então a tag **órfã** — a que não nomeia nenhuma nota — também sobrevive e aparece
    no inventário. A aresta `tagged_as` do grafo continua sendo criada **só** para as
    tags resolvidas: a tabela não reinstala o nó morto que a ADR-041 eliminou, nem
    altera o Health Score.
  - **`mem tags`** — o inventário de tags do vault: uma tag por linha em ordem
    alfabética, com `--counts`, `--sort=count`, `--file <caminho>`, `--folder <pasta>`,
    `--total` e `--json`.
  - **`mem tag <nome>`** — a contagem de documentos que carregam a tag, com
    `--verbose`, `--total` e `--json`. Casa a tag **e as descendentes** (`projeto` conta
    `projeto/alpha`), como o `tag:inbox` do Obsidian.
  - **`mem search --tag <tag>`** — filtro por tag na busca, combinável com
    `--category`, `--decay` e `--mode`. É **push-down** SQL nos mesmos pontos em que o
    filtro de categoria é aplicado, e não um filtro pós-fusão. `--tag` sem valor é erro
    de uso: o comando imprime o uso e sai com código diferente de zero.
  - **Tools MCP `memory_get_tags` e `memory_get_tag`** — o inventário de tags e a
    consulta de uma tag, no `content` legível e no `structuredContent`. O número de
    tools do servidor MCP subiu de **24 para 26**.
  - **O formato da tag segue a norma do Obsidian** (`obsidian.md/help/tags`):
    **case-insensitive** (`#Tag` e `#TAG` são a mesma tag, exibida com o casing da
    primeira criação), **hierárquica** (`#a/b` preserva o caminho e a consulta casa as
    descendentes) e com o **conjunto de caracteres permitido** — letras de qualquer
    script, números, `_`, `-`, `/` e Unicode/emoji. Uma tag só numérica (`#1984`) é
    inválida; `#y1984` é válida.
- **Correções de parser no mesmo passo** (a superfície de tags só é confiável se a
  extração parar de produzir tag falsa):
  - A regex de tag **truncava** `#ação` para a tag espúria `a` — `ç` estava fora da
    classe de caracteres. Agora a tag é capturada inteira, e `#1a`, `#_tag` e `#🚀`
    também são capturadas.
  - Blocos de código delimitados por `~~~` **não eram removidos** antes da extração:
    uma tag ou um wikilink dentro de um fence `~~~` viravam tag e aresta reais. O
    `mem outline` já tratava os dois formatos; o parser estava atrás.
  - Comentários `%%` do Obsidian **não tinham tratamento nenhum**: o comentário era
    indexado, buscável, e as tags dentro dele viravam tags reais. Agora `%%` é removido
    antes da extração de tag e de link (inline e em bloco).
- **Painel de métricas da memória (`viewer/`, ADR-059 e ADR-060)**: SPA React 19 +
  Vite 8 que **o próprio MCP serve**, sem processo extra, sem CDN e sem runtime de
  aplicação. Cinco visões: **Overview** (prontidão e Health Score com dead links,
  órfãs e self-loops), **Grafo** (hubs por grau e PageRank, comunidades LPA com
  modularidade, conexões latentes), **Notas** (lista indexada com filtro, outline,
  metadados e leitura com `[[wikilinks]]` navegáveis), **Drift** (documentação
  defasada, código sem cobertura, blast radius, caminho entre dois nós) e
  **Código** (rota `/code`: símbolos de um repositório e seus vizinhos estruturais,
  com o tipo da aresta e o ponto de chamada — ADR-061).
  - Stack: TanStack Query, TanStack Router, Zustand, `zod/mini` e
    `@tanstack/markdown`. Bundle final de **144.667 bytes gzip -9** — a mesma
    medição que o job `frontend` do CI faz — contra o orçamento de 160 kB (era
    133.086 bytes antes da rota `/code`). A meta registrada na ADR-060 era ~139 kB;
    o relatório do próprio Vite, que comprime em zlib nível 6, mostra 143,3 kB para
    os mesmos arquivos antes da rota `/code`.
  - `zod/mini` e não `zod`: medido 3,0 kB gzip contra 90,8 kB do barrel. O job de
    frontend do CI impõe orçamento de **160 kB gzip** justamente para pegar essa
    importação errada, que passaria despercebida numa revisão de diff.
  - O pré-processador de wikilinks espelha o parser Go: mesma regex, mesma lista
    de relações e os **mesmos quatro contextos sem link** (bloco de código,
    código inline, comentário HTML, linha de tabela) — linkar ali mostraria
    links que o grafo não tem.
- **O transporte HTTP do MCP serve arquivos estáticos (ADR-059)**: a raiz servida é
  `<base>/.mymemory/www/`, com `app/` (build da viewer) e `graph/` (artefatos de
  grafo) — **nunca** a raiz do vault, e fora do escopo de indexação.
  - Ordem de avaliação é contrato: resolver → recusar escape da raiz (404) →
    servir arquivo existente → fallback da SPA **só** para `Accept: text/html` → 404.
  - A barreira de path traversal roda **antes** do fallback, e `/graph/` é
    prefixo reservado que nunca cai nele.
  - Degradação explícita: sem build, **503** com mensagem simples — não 404 para
    cada arquivo.
  - `make viewer` compila e publica o build na raiz servida.
- **`memory_visualize_graph` devolve a URL servida do artefato** (ADR-059), com o
  caminho em disco como companion. A URL usa o host e a porta **reais** do
  listener. Sem transporte HTTP ativo, devolve **só o caminho, sem erro** — a
  fricção real era um caminho não ser acionável para um agente em outra máquina.
- **Job `frontend` no CI**: instala, typechecka, testa, linta e compila a viewer,
  e impõe o orçamento de 160 kB gzip. O último step roda a invariante do
  `internal/ci` **com `node_modules` presente** — sem essa combinação, ela roda
  num checkout limpo, onde `go list ./...` não vê nenhum pacote de terceiro.
- **`structuredContent` no envelope de resposta MCP** (ADR-060, MCP spec
  2025-06-18): as dez tools de leitura devolvem, além do texto legível, o dado
  estruturado que o handler já tinha antes de formatar em Markdown. A forma é o
  **JSON tags do tipo Go**, que é a fonte da verdade.
  - Mudança **aditiva**: `NewTextResult` não muda e as 24 tools mantêm o `content`
    textual byte-idêntico. O campo é omitido quando uma tool não o preenche.
- **Nove comandos de leitura do vault (ADR-057)**: `mem files` (com `--ext` e
  `--folder`), `mem folders`, `mem read`, `mem outline`, `mem wordcount`,
  `mem random` (`--json`), `mem deadends`, `mem file` e `mem recents`.
  Todos leem **o índice**, nunca o filesystem — por isso funcionam num vault
  federado cujo conteúdo só existe no PostgreSQL.
  - Extensão ou pasta desconhecida sai com zero e lista vazia, como no Obsidian.
  - `mem read`/`outline`/`wordcount`/`file` aceitam path completo ou basename;
    basename ambíguo é reportado com os candidatos, nunca resolvido por escolha
    silenciosa.
- **Quatro ferramentas MCP de leitura do vault (ADR-057)**:
  `memory_list_files`, `memory_read_note`, `memory_note_outline` e
  `memory_file_metadata`. O agente obtém a mesma superfície do CLI sem shelling
  out; caminho ausente do índice retorna **erro de tool**, não string vazia.
  - `memory_file_metadata` expõe `mtime` e `size`; `mtime = 0` é reportado como
    desconhecido em vez de virar uma data de 1970.
- **`documents.mtime` e `documents.size`** persistidos nos dois engines a partir
  do `stat` real do arquivo, e o `mem doctor` passa a reportar quantos
  documentos foram indexados antes dessas colunas existirem.
- **`memory_reconcile` (MCP tool, ISSUE-015)**: reconcilia o vault (filesystem)
  com o índice PostgreSQL **sem reindexar tudo**. Remove apenas os documentos
  cujo arquivo sumiu do disco. `dry_run` é `true` por padrão, e a operação
  aborta se o vault não existir ou se a varredura do disco retornar 0 arquivos.
  - `internal/store`: reconciliação vault↔postgres + guardas de segurança.
  - Contrato da tool coberto por teste em `internal/mcp`.
- **Seção Ortografia no `mem doctor` (ISSUE-017)**: detecta termos em
  português escritos sem acento em `title`, `abstract` e conteúdo dos chunks.
  É **apenas aviso** — não soma nem subtrai do Health Score.
  - A lista de correções é curada (`internal/store/orthography.go`), não derivada
    por regex: `esta` (pronome), `ate`, `ha`, `media`, `sim`, `cache`, `index` e
    `target` são corretos sem acento e nunca são reportados.
  - Ignora blocos de código, code spans, URLs, `[[wikilinks]]` e frontmatter —
    exceto os **valores** de `title` e `summary`.
- **`MY_MEMORY_SKIP_DOTENV` (ISSUE-016)**: desliga a descoberta automática de
  `.env` ao subir a árvore de diretórios, impedindo que um `.memory/.env` do
  projeto repusesse `MY_MEMORY_PG_URL`/`DATABASE_URL` por baixo dos testes.
- **Provider de embeddings embutido (ADR-035)**: `BuiltinClient` com inferência
  ONNX, tokenizer SentencePiece Unigram via `librembed`, e um corpus de
  benchmark PT-BR com guardas de qualidade e critério de abort.
- **`embed_epoch` como segundo eixo do cache incremental**: trocar de modelo ou
  de receita de vetor invalida os vetores gravados mesmo sem mudança no arquivo.

### Changed

- **Migração do embedder: 768d → 384d, Ollama → builtin**. O vetor agora tem
  **384 dimensões**, produzido pelo modelo embutido
  `paraphrase-multilingual-MiniLM-L12-v2`. O Ollama saiu do caminho vetorial.
  - `embedder.EmbeddingDim` (384) passou a ser a **única** dimensão válida:
    `internal/store/postgres.go` declara `embedding vector(384)`, e aceitar um
    `dimension: 768` do config produzia um quantizador do tamanho errado, que
    quebrava ao ler o vetor de volta (e não cabia na coluna no Postgres).
  - `mem init` passa a gerar `provider: "builtin"` no `.memory/config.yaml`.
  - As variáveis `MY_MEMORY_EMBED_PROVIDER`, `MY_MEMORY_EMBED_URL`,
    `MY_MEMORY_EMBED_MODEL` e `MY_MEMORY_EMBED_DIM` continuam sendo lidas do
    `.env` por compatibilidade, mas **não têm efeito** — ver
    `docs/AGENT_INTEGRATION_GUIDE.md`.
- **Senha do PostgreSQL vira campo de DSN, nunca URL**: a URL canônica fica no
  config e a senha em variável de ambiente, sanitizada nos logs.
- **SQLite único e central em `~/.memory/memory.db`** quando não há Postgres no
  cofre central, compartilhado por todos os repositórios.
- **CI**: `setup-go` lê a versão declarada no `go.mod` em vez de fixar `1.25`.

### Fixed

- **`mem repos` não auditava a identidade dos repositórios registrados.** O `id`
  do catálogo global é a chave que casa o repositório com o override de
  storage por repositório (`matchesID || matchesPath`). Com o `id` divergente
  do `repo_id` do `.memory/config.yaml`, o casamento passava a depender **só do
  caminho** — e um repositório **clonado ou movido** perdia o override
  **em silêncio**, sem um único aviso. Pior: o `id` exibido pelo `mem repos`
  não era o `repo_id` real, o que enganava no diagnóstico.
  - O `mem repos` agora compara, para cada entrada do catálogo, o `id` com o
    `repo_id` declarado no vault do path registrado. Divergência é marcada com
    `❌`, imprime os **dois valores**, o path, o arquivo lido e a ação, e o
    comando sai com **código ≠ 0** — trocando o silêncio por erro explícito.
  - Entrada **sem** vault inicializado, ou com vault que não declara
    `repo_id`, **não** é divergência (não há contra o que comparar): continua
    listada com `⚠️` e sai com 0. O formato da tabela não mudou.
  - A auditoria é estritamente **read-only**: não escreve no catálogo global nem
    no vault.
  - Com o `id` alinhado, o repositório **continua casando pelo id** mesmo
    depois de movido — o override de storage sobrevive à mudança de path.
  - O `repo_id` do disco é a fonte da verdade: ele é persistido no
    `.memory/config.yaml` e não muda com a mudança de pasta.
  - Spec: `.specs/features/repo-id-alignment/`.
- **`mem doctor` dava Health Score 100/100 num vault que nunca foi indexado.** A
  métrica é uma razão sobre a população de documentos indexados, mas o cálculo
  inteiro estava dentro de `if report.TotalDocuments > 0` — num vault vazio a
  guarda pulava tudo e o score ficava no valor inicial `100`, a nota máxima. O
  relatório deixou de distinguir "vault saudável" de "vault inexistente", que é
  justamente o uso principal dele. A guarda **e** a matemática de penalidades
  estavam duplicadas nos dois motores (SQLite e PostgreSQL), e foi essa
  duplicação que produziu o defeito.
  - O Health Score agora **só se aplica a um vault com documentos indexados**.
    Sem documentos, o relatório diz **"não se aplica"** e aponta a ação de
    recuperação (`mem index`); no CLI o indicador visual é neutro (⚪) em vez
    de verde. Um vault nunca indexado não é um vault doente.
  - No payload público da tool MCP `memory_doctor`, `health_score` passou a ser
    **anulável**: quando o score não se aplica, a chave **não viaja** no payload.
    Ausência **não** é 0 (saúde catastrófica) e **não** é 100 (saúde perfeita). O
    campo viaja ausente, e não `null`, porque o contrato do wire do
    `structuredContent` proíbe `null` — a viewer valida com `zod/mini`, que aceita
    ausência mas rejeita `null`. O JSON dos vaults com documentos continua
    **byte-idêntico**. A viewer tolera ausência e `null`, e mostra "—" com o
    rótulo "vault sem documentos".
  - A matemática das penalidades da ADR-013 (5 por dead link, até 40; órfãs
    proporcionais, até 30; 5 por self-loop, até 15; 2 por chunk dessincronizado,
    até 15) foi extraída para **uma função só**, compartilhada pelos dois
    motores — para que o defeito não reapareça em um deles só. O caminho
    saudável não regrediu: 100 no relatório perfeito, 90 com 2 dead links, 87
    com órfã + self-loop.
  - Spec e evidências: `.specs/features/doctor-empty-vault-score/`.
- **A suíte de testes escrevia no Postgres real do dono.** `internal/db` e
  `internal/store` não tinham `TestMain` neutralizando `MY_MEMORY_PG_URL` — que tem
  precedência sobre qualquer config — então os testes de integração desses pacotes
  gravavam direto no banco de produção a cada `go test` (foram medidos **126
  repositórios** e **370 documentos** de lixo no store real). Os dois pacotes agora
  usam o mesmo padrão já existente em `cmd/mem`: capturar a URL **antes** de
  neutralizar, limpar `MY_MEMORY_PG_URL`/`MY_MEMORY_PG_PASSWORD`/`POSTGRES_URL`/
  `DATABASE_URL` e restaurar no fim. O harness compartilhado ficou em
  `internal/pgtest`: a integração só roda com `MY_MEMORY_PG_TEST=1` e sempre contra
  um **banco descartável** (`CREATE DATABASE mymemory_pgtest_*` +
  `DROP DATABASE ... WITH (FORCE)`) — o banco `mymemory` do dono nunca é alvo.
  **Hermeticidade provada:** `go test -count=1 ./...` com a variável **presente** no
  ambiente passa (27/27 pacotes) e as contagens do store real ficam **idênticas**
  antes e depois.
- **`mem search --tag=` descartava o filtro silenciosamente.** Com a flag presente e
  o valor vazio (`--tag=` ou `--tag ""`), o comando ignorava o filtro pedido,
  buscava sem ele e saía com código 0 — o resultado parecia o filtrado e não era.
  O `--tag` sem argumento já era erro de uso; agora as duas formas são. O valor
  passa a ser chegado em `canonicalSearchTag` / `validateSearchTag`
  (`cmd/mem/main.go`), antes de abrir o índice.
- **A flag `--storage` não aparecia na ajuda**, apesar de existir e funcionar em
  todos os comandos do vault. Agora ela é listada, com o conjunto de comandos a
  que se aplica.
- **`mem tags --folder` não encontrava nada no Windows.** O filtro comparava
  `path LIKE 'pasta/%'`, mas o caminho indexado no Windows usa `\` como
  separador — o comando devolvia lista vazia em Windows e funcionava em Linux.
  A comparação passa a ser feita sobre a forma com `/` nos dois engines, e a
  pasta precisa ser um segmento inteiro (`docs` não casa `docs-outro`), com a
  mesma regra já usada por `mem files --folder`.
- **`memory_search` derrubava o servidor MCP inteiro com panic** quando o binário é
  compilado sem a tag de build `onnx` (isto é, sem provider de embedding). A busca
  do servidor chamava `emb.GenerateEmbedding` sem guarda — e, como ela roda em
  goroutine sem `recover()`, o pânico derrubava o **processo**, não só a
  requisição: o servidor MCP morria no primeiro `memory_search` de um cliente.
  - A causa raiz foi corrigida nos quatro pontos do MCP (modos `vector` e
    `hybrid`, engines SQLite e Postgres): sem provider, o vetorial devolve erro
    de uso e o híbrido cai para a busca textual.
  - Em profundidade, as goroutines da busca federada agora isolam o pânico: ele
    vira um erro da requisição e o servidor continua de pé.
  - Verificado com um cliente JSON-RPC real contra o servidor MCP HTTP.
- **`mem search` derrubava com nil-pointer-dereference quando o binário é compilado
  sem a tag de build `onnx`** (isto é, sem provider de embedding). A busca chamava
  `emb.GenerateEmbedding` sem guarda; o modo híbrido agora cai para a busca textual
  FTS5 com aviso, e o modo vetorial explica que precisa de provider e sai com
  código de erro. Nenhum teste pegava: os testes de busca sempre rodavam com
  provider, e o caminho sem provider só aparece em máquina sem ONNX instalado.
- **`mem search --tag` sem valor saía com código 0**, e o filtro pedido não era
  aplicado. O pacote `flag` do Go só reclama de argumento faltante quando não
  sobrou nenhum argumento depois da flag, então a flag seguinte (`--mode`) era
  engolida como valor de `--tag`. Agora a checagem roda antes do parse e o
  comando sai com código diferente de zero, como o AC TAG-19 exige.
- **Erro de busca devolvia código de saída 0** e ia para a stdout. O modo vetorial
  sem provider imprimia "Erro" e saía com zero; agora a busca devolve o código de
  saída e as mensagens de erro vão para o `stderr`, como no resto da CLI.
- **`--help` imprimia o uso duas vezes** e, no `mem search`, ainda executava a
  busca com a query que sobrou (`mem search algo --help` buscava "algo"). `mem
  tags --help` e `mem tag --help` saíam com código 1 e a mensagem "Erro: flag:
  help requested". Os três comandos agora imprimem o uso uma vez e saem com
  código 0.
- **A rota `/graph` da viewer devolvia 404 em texto puro quando a URL era aberta
  ou recarregada direto**. O prefixo `/graph/` é reservado para os artefatos de
  grafo, e o path **exato** `/graph` — que é uma rota da SPA — caía dentro desse
  prefixo. A navegação por link funcionava, porque o roteador do cliente não
  passa pelo servidor; abrir ou recarregar a URL não.
  - A área reservada passou a ser o **conteúdo** de `/graph/`: o path exato
    `/graph` e `/graph/` caem no fallback da SPA, como qualquer path de
    navegação — que é o que a ADR-060 exige desde a sua emenda.
  - `GET /graph/<arquivo>` continua servindo o artefato, e `GET
    /graph/nao-existe.html` continua devolvendo 404 sem o entrypoint.
  - Encontrado pela **validação visual** das quatro rotas, que o verificador
    independente não podia fazer (ele não renderiza a SPA). Teste de regressão
    com mutação provada.
- **O servidor MCP serializava uma lista vazia como `null`, e a viewer rejeitava
  o payload**. O `encoding/json` transforma slice nil em `null`, e o schema
  `zod/mini` da viewer aceita `undefined` em `.optional()` mas **não** `null`:
  a validação falhava, o cliente caía no fallback textual e o painel `/drift`
  renderizava markdown cru em vez da UI estruturada.
  - O caso não era hipotético: `memory_get_drift` constrói o relatório com duas
    slices nils sempre que não há drift nenhum — que é o caso comum.
  - A normalização ficou na **fronteira do wire**, em `NewStructuredResult`, e
    não em cada produtor: o contrato do payload é que uma lista vazia é `[]`, e
    vale para toda tool que um dia devolver uma lista. A função é pura — os
    structs são copiados antes de normalizados.
  - O teste de regressão cobre os oito payloads com listas, o handler real de
    drift e a pureza da normalização. **Mutação provada**: sem a correção, os
    oito casos falham com o payload `null` exato.
- **`go list ./...` enxergava código Go de dependência npm, e isso quebrava o
  invariant do CI**. O pacote npm `flatted` distribui um port Go em
  `golang/pkg/flatted`; assim que alguém rodava `npm install` dentro de
  `viewer/`, `go list ./...` passava a devolver
  `viewer/node_modules/flatted/golang/pkg/flatted` como pacote deste módulo, e
  o teste `TestWorkflowCoversEveryPackage` (de `internal/ci`) falhava — o
  `go test ./...` ficava **vermelho localmente**, com uma causa que não tinha
  nada a ver com o workflow.
  - `internal/ci` passa a descartar qualquer pacote sob `node_modules`
    (`isThirdPartyPkg`): o invariant é sobre código **deste** repositório, e
    pacote de dependência só está ali porque mora dentro da árvore de trabalho.
  - A alternativa — declarar o pacote em `excludedPackages` — foi rejeitada: a
    árvore npm muda com o lockfile e o `StaleExclusions` reportaria a exclusão
    como obsoleta.
  - Dois testes novos fixam a semântica (casos de filtro e varredura da lista
    real de pacotes).
- **O último step do job `frontend` do CI passou a declarar o diretório de
  trabalho de forma absoluta**. O job declara `defaults.run.working-directory:
  viewer` e o step que roda a invariante de `internal/ci` usava
  `working-directory: .` para voltar à raiz do repositório.
  - Trocado por `${{ github.workspace }}`, que é absoluto: o step deixa de
    depender de como o runner resolve um caminho relativo.
  - **Correção do registro:** a justificativa original desta entrada estava
    errada. O código do runner (`actions/runner`,
    `src/Runner.Worker/Handlers/ScriptHandler.cs`) combina o `working-directory`
    do step com o **workspace** (`Path.Combine(workspaceDir, workingDirectory)`),
    e o `defaults.run.working-directory` do job só entra quando o step **não**
    define o valor. Ou seja, o `working-directory: .` original já rodava na raiz
    do repositório, e o job não teria ficado vermelho por causa disso. A troca
    continua valendo — explicitar o caminho absoluto é melhor do que depender
    de uma resolução implícita —, mas o defeito que ela supostamente corrigia
    não existia.
- **`--decay` ordena por tempo de edição, não por instante de indexação
  (ADR-057)**. O decaimento temporal lia `documents.updated_at`, que guarda
  `time.Now().Unix()` da indexação: um `mem index` em vault inteiro dava a
  todas as notas timestamps quase idênticos, o delta tendia a zero e o
  multiplicador convergia para 1.0 — a flag virava **no-op silencioso**,
  priorizando "processado por último" em vez de "editado recentemente".
  - A ordenação temporal agora usa `documents.mtime`, com fallback para
    `updated_at` quando o valor é 0 (vault indexado antes da migração, que
    continua funcionando sem exigir `mem index --force`).
  - A detecção de staleness passou a comparar o `mtime` persistido contra o
    `mtime` atual do disco, em vez de inferir mudança do instante de indexação.
  - As descrições de `--decay` na documentação diziam "notas recentes" sem
    dizer qual tempo: `docs/CLI_GUIDE.md`, `COMO_USAR.md`, `COMO_FUNCIONA.md`,
    `README.md`, `docs/REPOSITORY_BRAIN.md`, `skills/` e o help da CLI foram
    corrigidos para "tempo de edição do arquivo".
- **Dimensão do vetor vem de `embedder.EmbeddingDim`**, não de um literal `768`
  que sobreviveu à migração. Quando o embedding falhava, o fallback gravava
  vetor de dimensão errada.
- **BOM do Windows duplicava o frontmatter** e descartava as flags da CLI.
- **`StreamUntilCancel`**: o cancelamento vence o fechamento do canal
  (ISSUE-011). A assinatura promete "stream até o chamador cancelar *ou* `in`
  fechar"; devolver `nil` num cancelamento afirmava que o stream terminou com
  sucesso.
- **Separador de caminho normalizado antes do `filepath.Clean`**.
- **Sentinelas de estado global dos testes medem o arquivo certo**, e não o
  diretório home inteiro.

### Removed

- **Viewer Electron removido do repositório**: removida toda a camada
  TypeScript/Electron — `electron/**`, `src/**` (renderer React), `package.json`,
  `package-lock.json`, `tsconfig.json`, `index.html`, `components.json`,
  `biome.jsonc`, `forge.config.ts`, `forge.env.d.ts`,
  `vite.{main,preload,renderer}.config.mts`, `vitest.config.ts`,
  `playwright.config.ts`, `scripts/dev.mjs`, `scripts/vitest.setup.ts`,
  `scripts/viewer-smoke.test.mjs`, `data-central.json` e
  `resources/chatLanguageModels.default.json`.
  - Docs: `docs/VIEWER.md` removido.
  - Specs: `feat-viewer-electron`, `feat-viewer-electron-phase-2`,
    `feat-viewer-chat`, `feat-viewer-v3-m1-3-pane-layout` e
    `feat-chat-language-models` removidos (todos eram viewer-only).
  - CI: job `frontend` (typecheck + unit tests) removido — rodava `npm ci`,
    `npm run typecheck` e `npm test`, impossível sem `package.json`.
  - ADRs `037`, `038`, `048`, `049`, `051` e `053` marcadas como **Obsolete**
    (texto preservado como memória histórica, fora do índice ativo).
  - O CLI Go (`cmd/`, `internal/`) não foi alterado. A visualização de grafo
    disponível continua sendo a do CLI via `mem graph view` (ADR-020).
- **Implementação de `mem replay` sem dispatch**: existia no código mas não era
  roteada pelo switch de comandos.
- **Scaffolding de ASR e transporte JSON-RPC do ADR-042**; o caminho
  ASR/Nemotron foi adiado (ADR-045 passou a **Deferred**).
- **Write-policy engine do ADR-044** e os pacotes de writer/projection que
  ficaram inalcançáveis com ele.
- Script `main.go` de Postgres na raiz do repositório e
  `postgres_compose_test.go` obsoleto.

### Documentation

- **Limitação conhecida: `mem index` aborta contra um Postgres com pooler em
  modo transacional** (ISSUE-032). O `lib/pq` usa o statement "unnamed" do
  servidor para toda query parametrizada, e o pooler pode mandar o Parse e o
  Bind para backends diferentes — o servidor passa a responder `26000
  (unnamed prepared statement does not exist)` ou `08P01 (bind message
  supplies N parameters, but prepared statement "" requires M)`, e a indexação
  **morre no meio**, deixando o vault parcial. **Workaround verificado:** usar o
  endpoint **direto** (sem `-pooler`) no `~/.memory/config.yaml` — com ele a
  indexação completa (276 documentos). A correção de código (tolerar o erro
  por documento em vez de abortar tudo) está aberta para decisão.
- **Skills do repositório atualizadas com a superfície de tags**: `skills/**` e
  `.agents/skills/my-memory/SKILL.md` ganharam `mem tags --folder` e
  `mem tags --total`, a tabela de **exit codes** da superfície de tags e da busca,
  o filtro `--tag` (CLI e MCP, com o argumento `tag` de `memory_search`), o
  argumento `include_code` de `memory_search` marcado como *placeholder*, e o
  gotcha do binário compilado sem provider de embedding — o híbrido cai para a
  busca textual FTS com aviso, `--mode vector` sai com código diferente de zero, e
  no MCP o `mode=vector` devolve erro JSON-RPC **sem derrubar o servidor**.
- **ADR-055 decidida**: a pergunta que a ADR deixou em aberto — para onde
  `resolved_doc_id` aponta — foi respondida em 2026-10-21. Aponta para
  **`documents.id`**, a única referência estável do schema, com o título
  guardado como rótulo. O título resolveria o vínculo silenciosamente quando o
  frontmatter muda e colide entre pastas; `documents.id` sobrevive à renomeação,
  e a reindexação passa a ser o reconciliador explícito.
  - A ADR sai de **Proposed** para **Aceito**, e a Opção A fica escolhida: a
    tabela `document_tags` é populada *antes* do passo fuzzy de resolução.
  - A **implementação** continua pendente: a migration nos dois engines
    (SQLite e Postgres), a população no indexer e a superfície `mem tags` /
    `mem search --tag` não existem.
- **Nove comandos do CLI finalmente documentados**: `mem up`, `mem down`,
  `mem logs`, `mem profiles`, `mem events` e os quatro `mem code-*`
  (`code-index`, `code-search`, `code-graph`, `code-stats`). Todos já existiam
  desde antes da v2.0.0 — o que faltava era a documentação, não o comando.
- **`mem help` sincronizado com o dispatch**: `up`, `down`, `logs` e `profiles`
  estavam ausentes de `printHelp()`. Um teste novo (`cmd/mem/help_test.go`)
  extrai os `case` do switch de `main.go` e falha se algum não aparecer na ajuda
  — ou se a ajuda inventar um comando que o dispatch não roteia.
- **Flags documentadas conforme o registro real**: `mem code-graph` **não** tem
  `--json` (tem `--depth`, `--direction`, `--db`, `--repo`); `mem code-stats`
  tem `--top`, `--json`, `--db` e `--repo`. Documentados também
  `mem doctor --events` e `mem search --links`.
- **Narrativa do Ollama removida da documentação**: nenhuma guia mais manda
  instalar `nomic-embed-text`, e o fallback léxico é descrito pelo que de fato
  o dispara (build sem a tag `onnx` ou falha do provider embutido).
- **Índice de ADRs corrigido**: as nove ADRs ausentes do índice (035, 036, 039,
  040, 041, 042, 043, 044, 050) foram adicionadas com o status real; ADR-045
  corrigida de Proposed para Deferred; ADR-035 promovida para **Accepted**.
  O slot **ADR-052** está reservado e vazio — nenhum arquivo foi inventado.
- **ADR-059 completada**: a decisão de servir a SPA React/Vite de `viewer/` pelo
  transporte HTTP do MCP sai do esboço de 95 linhas e vira ADR completa (MADR,
  13 seções), com a **superfície de ataque explicitada**.
  - Registra a mudança de superfície — de "zero arquivos servidos" para "um
    diretório dedicado" — e que o transporte **não tem autenticação**: lacuna
    **pré-existente** da ADR-021, não criada por esta feature.
  - O artefato de grafo expor títulos de nós do vault é registrado como o
    aumento de exposição mais relevante, sem eufemismo.
  - Registra a **divergência consciente** com a ADR-050: `default-src 'none'`
    é incompatível com uma SPA JavaScript. A ADR propõe a baseline de CSP
    aplicável e deixa a aplicação para a etapa de implementação.
  - Fecha a lacuna de que o handler de `memory_visualize_graph` **não descobre**
    host/porta: o mecanismo (endereço real do `net.Listener`) vira contrato.
  - Corrige três afirmações do esboço: `viewer/` **ainda não está versionada**,
    o alvo `make viewer` **não existe**, e quem especifica o path traversal é a
    spec `mcp-static-serving` — não a ADR-058.
  - A auditoria das citações encontrou uma **sexta** divergência, agora
    registrada: o invariant de `internal/ci` **é** afetado por `viewer/`, porque
    `go list ./...` enxerga o port Go que o pacote npm `flatted` distribui.
- **ADR-060: stack do frontend da viewer e a rota de fallback SPA**. A viewer
  passa de "scaffold do shadcn" para stack decidida, com custos medidos:
  `@tanstack/react-query` (13,2 kB gzip), `@tanstack/react-router` (30,98 kB),
  `zustand` (489 B), `zod/mini` (3,0 kB — contra **90,8 kB** do `zod` completo,
  medido localmente) e `@tanstack/markdown` (6,94 kB). Total: **+54,7 kB gzip**
  sobre o baseline de 84,28 kB.
  - **Emenda à ADR-059**: o contrato de rotas ganha o fallback SPA, com duas
    regras não-negociáveis — a checagem de path traversal acontece **antes** do
    fallback, e o fallback só se aplica a requisições `Accept: text/html`, de
    modo que um asset faltando continua sendo 404.
  - `@tanstack/config` **não** entra: o pacote está deprecado no npm ("Package
    no longer supported"), e a doc em `tanstack.com/config/latest` continua
    recomendando. O slot é ocupado por `@tanstack/vite-config@0.6.0`, sucessor
    do mesmo produto.
  - Registra a **duplicação da gramática de wikilink** entre o parser Go
    (`internal/parser/wikilinks.go`) e o pré-processador do SPA, com a regra de
    contexto (código cercado, código inline, comentário HTML, linha de tabela)
    que impede a viewer de linkar o que o grafo não tem.
  - As 24 tools MCP mapeadas são a superfície de dados do painel.
- **Índice de ADRs**: a linha em branco entre ADR-057 e ADR-058 foi removida —
  ela encerrava a tabela markdown e deixava ADR-058 e ADR-059 sem cabeçalho.

- **Varredura documental da viewer**: `README.md` passa a descrever os **7 pilares**
  — o painel é o sétimo — e lista as 59 ADRs; `AGENTS.md` registra as ADRs
  054–060, o **gate de frontend** (`npm run typecheck && npm run test &&
  npm run build`, com orçamento de 160 kB gzip) e a regra de que
  `.mymemory/www/` é raiz servida, gerada e **nunca versionada**;
  `docs/README.md`, `docs/CLI_GUIDE.md`, `docs/AGENT_INTEGRATION_GUIDE.md`,
  `docs/ARCHITECTURE.md` e `docs/REPOSITORY_BRAIN.md` recebem o painel, a rota
  de fallback e o `structuredContent`; `skills/mymemory-mcp/SKILL.md` documenta
  as duas camadas da resposta MCP, a URL servida e a **ausência de
  autenticação** do transporte.

### CI

- **A integração PostgreSQL agora é opt-in explícito.** Os jobs de Postgres do
  workflow passam a definir `MY_MEMORY_PG_TEST=1`, que habilita os testes de
  integração (eles são pulados por padrão, mantendo a suíte hermética fora do CI).
  O pacote novo `internal/pgtest` entrou no passo de cobertura — o gate de
  `internal/ci` exige que todo pacote de `go list ./...` apareça em algum step.
- **Cobertura dos pacotes que o workflow nunca executava** (ISSUE-012).
- `.gitattributes` (`* text=auto eol=lf`) adicionado para que o gate de `gofmt`
  seja executável também no Windows.

## [2.0.0] — 2026-09-22

### Added

- **Chat LLM no viewer Electron (ADR-051)**: nova aba **Chat** com provider
  abstraction (Anthropic default + OpenAI opcional) via Vercel AI SDK,
  streaming token-a-token via IPC `webContents.send`, persistência local de
  múltiplas conversas em IndexedDB (`idb-keyval`), RAG opcional via
  `mem search` rodando como subprocess no main process.
  - API keys armazenadas em `<userData>/chat-config.json` — nunca saem do main.
  - 8 IPC channels: `mem:chat:{send,abort,config-get,config-set,mem-search,delta,done,error}`.
  - Componentes: `chat-view`, `chat-sidebar`, `chat-thread`, `chat-message`
    (MarkdownLite regex), `chat-input` (Enter envia, Shift+Enter quebra linha),
    `chat-settings-modal` (write-only API key field).
  - Zustand store com subscriptions → `idb-keyval` para auto-persistência.

- **Viewer Electron Phase 1 MVP (ADR-048)**: scaffold Electron + React 19 +
  Vite 8 + shadcn/ui v4 + Tailwind 4 + i18next + Cytoscape. Substitui os HTMLs
  legados (`graph.html`, `graph-v2.html`).

- **Code AST multi-linguagem via tree-sitter (ADR-047)**: `mem code-index`,
  `mem code-search`, `mem code-graph`, `mem code-stats` CLI; novos MCP tools
  `code_search` + `code_neighbors`; boost RRF 2× em `qualified_name`.

- **Memory writer atômico + outbox + reprojeção idempotente (ADR-044)**:
  `mem replay --since <seq> [--apply]`, projection SQLite idempotente,
  `If-Match` header + 409 conflict detection.

- **Event runtime (ADR-043)**: `event_log` table + `documents.revision`
  column + envelope canônico + dispatcher com fan-out/ACK/retry +
  redaction PII (LLM02) + audit subscriber JSONL com rotação + `mem events
  {tail,inspect,replay,trace,last-sequence,stats}` + `mem doctor --events`.

- **mymemoryd supervisor (ADR-042)**: `mem up/down/logs/profiles` + sandbox
  best-effort + heartbeat + restart com exponential backoff + profiles YAML
  com herança.

- **Nemotron 3.5 ASR streaming via ONNX Runtime Go (ADR-045)**.

- **Threat model OWASP Top 10 aplicado ao MyMemory (ADR-050)**: controles
  LLM02 (PII redaction) + LLM05 + LLM07 + LLM10 (MaxAckPending).

- **Bugfixes pós-release v1.3.0 (ADR-036)**: parser EARS + fuzzy resolve
  wikilinks + dead-link pruning + drift calibration (ADR-039) +
  `mem index` auto-scope (ADR-040).

### Changed

- **ADR-040 Accepted**: SQLite é o default (auto-scope `.memory/memory.db`);
  PostgreSQL + pgvector é opt-in via `--storage=postgres` ou env var.
- **ADR-041 Accepted**: tags sem `[[nota]]` casa são removidas do grafo
  (não viram dead links).
- **ADR-039 Accepted**: detector de drift recalibrado (PageRank + match de path).
- **ADR-037 Superseded by ADR-048**: viewer reescrito com Electron (era Vite +
  Vanilla TS proposto).
- **Documentos com frontmatter**: `documents.title` agora usa `frontmatter.title`
  quando presente (ISSUE-010).

### Security

- **API keys nunca tocam o renderer** (ADR-051): chave gravada em
  `<userData>/chat-config.json`; IPC `config-get` retorna apenas `hasApiKey: boolean`.
- **Sanitização de URLs Postgres em erros** (commit `2361a92`):
  `sanitizePostgresURL()` oculta `user:password` → `***@host` em logs.
- **ADR-048 contexto isolation**: `contextIsolation: true`, `nodeIntegration: false`,
  `sandbox: true` no Electron BrowserWindow.

### Fixed

- ISSUE-001 (dead links 8 → 0 via tag pruning)
- ISSUE-005, ISSUE-008 (drift detector miscalibration)
- ISSUE-006 (orphan count inclui by-design paths)
- ISSUE-009 (tags sem casa removidas do grafo)
- ISSUE-010 (`documents.title` usa frontmatter.title)
- `pull_request_target` workflow gotcha (nota interna — branch base sempre
  usado pelo GitHub Actions)

## [1.4.0] — prior

Baseline before the work listed above. Used `graph-v2.html` legacy viewer.
