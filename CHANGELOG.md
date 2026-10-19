# Changelog

All notable changes to **my-memory** are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

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

### CI

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
