# Feature: feat-code-ast

## Problem Statement

O `my-memory` é um *Repository Brain* que indexa Markdown, embeddings, FTS5, e grafo (ADR-001, ADR-003, ADR-004, ADR-005, ADR-008, ADR-009). Hoje código-fonte é tratado como **texto opaco** dentro do pipeline — chunked, embeddado, mas sem estrutura sintática. Isso cria três lacunas concretas, agravadas pela análise dos repositórios `DeusData/codebase-memory-mcp` (43.9k⭐), `MemTensor/MemOS`, `iwe-org/iwe`, `oxigraph/oxigraph` e `deeplethe/forkd` (ver `docs/REFERENCES.md` §1.9, ADR-047):

1. **Sem queries estruturadas sobre código.** `mem search "função que chama X"` retorna matches literais de string; não há como pedir "todos os métodos Go que importam `database/sql`" ou "implementações da interface `Parser`".
2. **Sem grafo de código real.** Wikilinks conectam notas; a relação `foo.go → bar.go` (imports, calls, embeds) só existe se o autor citou `[[bar]]` em prosa — fonte não-confiável que o ADR-011 chama de `EXTRACTED` (vs `INFERRED`).
3. **Sem code-awareness pro LLM.** Quando o agente MCP responde `mem_search`, recebe chunks de prosa com código embeddado, mas sem tipo (`function`/`struct`/`import`) ou posição (linha, range). Desperdício de tokens; força o agente a re-parsear.

A solução proposta por **ADR-047** é integrar **tree-sitter** (binding Go oficial `github.com/tree-sitter/go-tree-sitter`) como dependência opcional (CGO), habilitando:

- Estágio `code_pipeline` opt-in dentro do `mem index` (flag `--code-ast[=true|false]`, default `true` quando tree-sitter compila).
- Novas tabelas SQLite `code_files`, `code_symbols`, `code_edges` no mesmo `memory.db` (coerente com ADR-001).
- Boost no ranqueamento RRF (ADR-009) para matches exatos em `qualified_name`.
- Ferramentas CLI `mem code-index`, `mem code-search`, `mem code-graph`, `mem code-stats`.
- Tools MCP `memory_code_search`, `memory_code_neighbors` (additive; tools existentes ganham flag opcional `include_code`).

## Out of Scope

- Análise semântica completa (type inference, call-graph com tipos resolvidos) — vira ADR futuro; este ADR cobre só parsing sintático + referências literais.
- Auto-correção de código ou refactor assistido.
- Servidor LSP (`gopls`) — deferido como Plano B (ADR-047 §Deferral).
- Cobertura obrigatória das 158 linguagens tree-sitter — v1 entrega top-10 com `--lang=all` opcional.
- Persistência da AST completa em disco — só os símbolos/arestas extraídos; AST bruta fica em memória durante parse.

## Assumptions & Open Questions

| Assumption / Question | Chosen default | Rationale |
| :--- | :--- | :--- |
| Linguagens iniciais (v1) | Go, Python, TypeScript/JavaScript, Rust, Java, C/C++, Ruby, PHP, Shell, C# (top-10) | Cobre ≥90% dos repos reais; demais via `--lang=all` |
| Resolução de referências cross-file | Heurística literal (mesmo qualified name) + escopo do escopo `.memory/config.yaml` | Suficiente para v1; dynamic dispatch fica v2 |
| Threshold de confiança para `code_edges` | confidence < 0.5 vai pra tabela `code_edges_uncertain` (não grafo principal) | Evita grafo poluído por erros de resolução |
| Granularidade de chunking por símbolo | 1 chunk por símbolo (function/struct/etc); arquivos sem símbolos viram 1 chunk | Maximiza granularidade de busca sem inflar cardinalidade |
| Build CGO | `//go:build treesitter` opt-in via `-tags treesitter`; build padrão (sem tag) só tem path no-op | Compatibilidade cross-platform sem quebrar usuários sem toolchain C |
| Detecção de linguagem | Extensão de arquivo + shebang na primeira linha (fallback) | Padrão tree-sitter; sem parsing mágico |
| Coexistência com Markdown pipeline | Mesmo `mem index` orquestra os dois pipelines; `--code-ast=false` desativa code pipeline | Decoupled mas coordenado |
| Reuso de `content_hash` (ADR-010) | `code_files.content_hash` reusa mesmo SHA-256 do ADR-010 | Consistência |
| Tooling MCP | Tools novas (`memory_code_search`, `memory_code_neighbors`) + flag `include_code` em `memory_search` existente (default `false`) | Aditivo, não quebra clientes MCP existentes |

Open questions: none — todas as premissas arquiteturais (10 linhas da tabela acima) estão detalhadas na ADR-047 §Decision Outcome e foram aceitas. Nenhuma decisão pendente para o Execute.

## User Stories

- **US1**: Como engenheiro, quero rodar `mem search "Database.Open"` e ver resultados priorizando o símbolo `*db.Database.Open` (qualified name match + boost RRF) em vez de prosa contendo "database" e "open".
- **US2**: Como tech lead, quero rodar `mem code-graph internal/parser.Parser.Parse` e ver todos os símbolos que `Parser.Parse` chama (grafo AST local) com deep links.
- **US3**: Como agente de IA conectado via MCP, quero invocar `memory_code_search("X", language="go", kind="function")` e receber lista estruturada de funções Go cujo nome casa com `X`.
- **US4**: Como agente de IA, quero invocar `memory_code_neighbors("internal/db.DB.Query", depth=2)` e receber sub-grafo de chamadas (quem chama Query, o que Query chama até depth=2).
- **US5**: Como engenheiro mantendo o vault, quero rodar `mem code-stats` e ver distribuição por linguagem, símbolos por kind, e identificar `code_files` órfãs (sem símbolos extraídos).
- **US6**: Como mantenedor do my-memory em CI sem toolchain C, quero que `go build` (sem tag `treesitter`) continue compilando e o path code-ast seja no-op com warning claro — Markdown pipeline permanece funcional.

## Requirements (EARS Notation)

### CA-01: Parser tree-sitter multi-linguagem
- **UBIQUITOUS**: The system SHALL parse source files of supported languages using `github.com/tree-sitter/go-tree-sitter` (or equivalent official binding) and extract symbols (`function`, `method`, `class`, `interface`, `struct`, `import`, `constant`, `variable`).
- **WHERE**: When a file extension is not in the top-10 supported list, the system SHALL skip parsing silently unless `--lang=all` is provided (which downloads grammar on-demand).
- **WHERE**: When `--lang=<csv>` is provided, the system SHALL restrict parsing to those languages only (allowlist).

### CA-02: Persistência em SQLite (mesmo `memory.db`)
- **UBIQUITOUS**: The system SHALL persist extracted metadata in tables `code_files`, `code_symbols`, `code_edges` (per ADR-047) within the unified `memory.db` (ADR-001).
- **UBIQUITOUS**: The system SHALL add tables `code_edges_uncertain` (low-confidence cross-file references, separate from main graph).
- **UBIQUITOUS**: The system SHALL create all `code_*` indexes (`code_graph_kind_idx`, `code_symbols_kind_name_idx`, `code_files_language_idx`) on migration.

### CA-03: Pipeline `mem index` com estágio code opt-in
- **WHEN**: When the user runs `mem index` and tree-sitter is compiled in (build tag `treesitter` set), the system SHALL execute the `code_pipeline` stage after the existing markdown pipeline.
- **WHERE**: When `--code-ast=false` is provided, the system SHALL skip the code pipeline entirely.
- **WHERE**: When tree-sitter is not compiled (default build, no CGO), the system SHALL log a warning and skip code pipeline — markdown pipeline SHALL run normally.

### CA-04: Cache incremental via SHA-256
- **UBIQUITOUS**: The system SHALL compute `content_hash` (sha256, ADR-010) and `ast_hash` (sha256 of serialized symbol set) per code file.
- **WHEN**: When `content_hash` matches the stored value, the system SHALL skip re-parsing.
- **WHEN**: When `content_hash` differs but `ast_hash` is identical, the system SHALL update `content_hash` and skip downstream re-embedding (whitespace/comment-only changes).

### CA-05: Boost RRF por `qualified_name` match
- **WHEN**: When a search query matches a `qualified_name` in `code_symbols` (e.g., query contains a `.` and follows camelCase/PascalCase pattern), the system SHALL apply a 2x weight boost to that symbol in the RRF score (per ADR-009).
- **WHERE**: When `--no-code-boost` is set, the system SHALL use the standard RRF weight (no boost).

### CA-06: CLI `mem code-index`
- **WHEN**: When the user executes `mem code-index`, the system SHALL scan the configured vault (ADR-016 scope), parse all supported code files, and persist symbols/edges to `memory.db`.
- **UBIQUITOUS**: The system SHALL support flags: `--lang=<csv>`, `--include=<glob>`, `--exclude=<glob>`, `--ast-hash`, `--no-embed`, `--storage=<sqlite|postgres>`.
- **UBIQUITOUS**: The system SHALL respect `.memory/config.yaml` `scope.include` and `scope.exclude` paths.

### CA-07: CLI `mem code-search`
- **WHEN**: When the user executes `mem code-search <query>`, the system SHALL delegate to `mem search` with flag `--kind=code` and apply code-specific ranking (qualified-name boost per CA-05).
- **UBIQUITOUS**: The system SHALL support flags: `--lang=<csv>`, `--kind=<symbol_kind>`, `--limit=<N>` (default 10).

### CA-08: CLI `mem code-graph`
- **WHEN**: When the user executes `mem code-graph <symbol>`, the system SHALL resolve `<symbol>` to a `code_symbols.id` (exact qualified_name match, fallback to suffix `.name` match) and return the depth-N neighborhood.
- **UBIQUITOUS**: The system SHALL support flag `--depth=<N>` (default 1, max 5).
- **UBIQUITOUS**: The system SHALL output a Markdown table with columns: `direction` (in/out), `kind`, `symbol`, `file`, `line`.

### CA-09: CLI `mem code-stats`
- **WHEN**: When the user executes `mem code-stats`, the system SHALL aggregate counts from `code_files` and `code_symbols` and print: language distribution, symbol-kind distribution, top orphan files (code_files with 0 symbols extracted).

### CA-10: Tool MCP `memory_code_search`
- **WHEN**: When an AI agent invokes `memory_code_search(query, language?, kind?)`, the system SHALL return a JSON array of `{symbol_id, qualified_name, kind, file, start_line, end_line, signature, doc_comment, score}` ordered by RRF score (with CA-05 boost).
- **WHERE**: When `language` is provided, the system SHALL filter results to that language.
- **WHERE**: When `kind` is provided, the system SHALL filter results to that symbol kind.

### CA-11: Tool MCP `memory_code_neighbors`
- **WHEN**: When an AI agent invokes `memory_code_neighbors(symbol, depth?)`, the system SHALL resolve `<symbol>` to a code_symbol and return sub-graph (in + out edges) up to `depth` hops.
- **UBIQUITOUS**: The system SHALL return JSON `{root: {...}, neighbors: [{direction, kind, symbol, file, line, hop}]}`.

### CA-12: Flag `include_code` em `memory_search` existente
- **UBIQUITOUS**: The system SHALL add optional `include_code: boolean` (default `false`) parameter to existing MCP tool `memory_search` — when `true`, code_symbols compete in RRF with markdown chunks.

### CA-13: Benchmarks reproduzíveis
- **UBIQUITOUS**: The system SHALL provide `bench/codeast/` with reproducible benchmarks targeting p99 latency:
  - Indexação: 1000 arquivos Go (200KB médio) ≤ 5s.
  - Indexação: 100 arquivos Python ≤ 2s.
  - Query híbrida (100k chunks total, 50% code) ≤ 50ms.
  - `code_graph Database.Open` depth=2 ≤ 100ms.
- **WHERE**: When a benchmark exceeds its target, the system SHALL record the result and the harness SHALL exit non-zero.

### CA-14: Build CGO opt-in
- **UBIQUITOUS**: The system SHALL isolate tree-sitter import behind `//go:build treesitter` so default builds remain CGO-free.
- **UBIQUITOUS**: The system SHALL detect at boot whether tree-sitter is compiled in and log once: `tree-sitter: enabled` or `tree-sitter: disabled (code pipeline skipped)`.
- **WHERE**: When `CGO_ENABLED=0` or tree-sitter binding fails to build, the system SHALL treat code_pipeline as no-op.

### CA-15: Compatibilidade CI cross-platform
- **UBIQUITOUS**: The system SHALL include CI matrix entries for (linux, macos, windows) × (CGO on, CGO off) verifying that:
  - CGO-on: code pipeline runs; benchmarks pass.
  - CGO-off: code pipeline skipped with warning; markdown pipeline passes.
- **WHERE**: When CGO-on fails on a platform, the system SHALL emit a clear error message identifying the missing toolchain (e.g., `gcc not found — install Xcode CLT or set CGO_ENABLED=0`).

### CA-16: Schema migration idempotente
- **UBIQUITOUS**: The system SHALL add `code_*` tables via the same migration mechanism as ADR-001 (idempotent on boot, no manual SQL).
- **UBIQUITOUS**: The system SHALL work identically against SQLite local and Postgres/pgvector (ADR-040).

### CA-17: Documentação atualizada
- **UBIQUITOUS**: The system SHALL update `docs/CLI_GUIDE.md`, `docs/AGENT_INTEGRATION_GGUIDE.md`, `docs/ARCHITECTURE.md` com a nova superfície (`mem code-*`, MCP tools, schema).
- **UBIQUITOUS**: The system SHALL update `README.md` capability table.

## Requirement Traceability

| Requirement ID | Description | Status |
| :--- | :--- | :--- |
| CA-01 | Parser tree-sitter multi-linguagem | verified |
| CA-02 | Persistência SQLite (mesmo `memory.db`) | verified |
| CA-03 | Pipeline `mem index` com code stage opt-in | verified |
| CA-04 | Cache incremental SHA-256 + `ast_hash` | verified |
| CA-05 | Boost RRF por `qualified_name` match | verified |
| CA-06 | CLI `mem code-index` | verified |
| CA-07 | CLI `mem code-search` | verified |
| CA-08 | CLI `mem code-graph` | verified |
| CA-09 | CLI `mem code-stats` | verified |
| CA-10 | Tool MCP `memory_code_search` | verified |
| CA-11 | Tool MCP `memory_code_neighbors` | verified |
| CA-12 | Flag `include_code` em `memory_search` | verified |
| CA-13 | Benchmarks reproduzíveis | verified |
| CA-14 | Build CGO opt-in (`//go:build treesitter`) | verified |
| CA-15 | CI matrix cross-platform × CGO on/off | verified |
| CA-16 | Schema migration idempotente (SQLite + Postgres) | verified |
| CA-17 | Documentação atualizada | verified |