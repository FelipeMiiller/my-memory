# ADR-047: Indexação de Código-Fonte via Tree-sitter (Code AST Multi-linguagem)

- **Date**: 2026-09-21
- **Status**: Accepted
- **Deciders**: Felipe Miiller, Mavis (orchestrator)
- **Tags**: code-intelligence, tree-sitter, ast, parser, multi-language, search, hybrid-ranking, code-graph

## Context and Problem Statement

O `my-memory` foi desenhado como um * Repository Brain * que unifica SQLite + FTS5 + sqlite-vec + grafo recursivo + Obsidian Markdown (ADRs 001/003/004/005). O parser Markdown é robusto (ADR-008), mas **código-fonte é tratado como texto opaco**: indexado como chunk, embeddado, mas sem estrutura.

Isso cria 3 gaps concretos, agravados pela referência cruzada com o codebase MCP `DeusData/codebase-memory-mcp` (43.9k⭐) e o iwe `iwe-org/iwe` (1.7k⭐):

1. **Sem queries semânticas sobre código.** Hoje `mem search "função que chama X"` retorna matches literais de string; não é possível pedir "todos os métodos Go que importam `database/sql`" ou "implementações da interface `Parser`".
2. **Sem grafo de código real.** Wikilinks conectam notas. Mas a relação `foo.go → bar.go` (imports, calls, embeds) só existe se o autor citou `[[bar]]` em prosa — fonte não-confiável que o ADR-011 chama de EXTRACTED (vs INFERRED).
3. **Sem "code awareness" pro LLM.** Quando o agente MCP responde `mem_search`, ele recebe chunks de prosa com o código embeddado — mas sem o tipo (function/struct/import) ou a posição (linha, range). Isso desperdiça tokens e força o agente a re-parsear.

A pesquisa autoral de 2026-09-21 mapeou 5 repositórios concorrentes/adjacentes. Os relevantes ao escopo:

- **`DeusData/codebase-memory-mcp`** (Rust, 43.9k⭐) — prova que **tree-sitter** indexa 158 linguagens em ms com sub-ms queries via Cypher-like. É o concorrente mais próximo em "code memory".
- **`iwe-org/iwe`** (Rust, 1.7k⭐) — markdown + grafo + LSP, mas usa AST próprio (`liwe` lib) só pra **markdown**, não pra código.
- `MemTensor/MemOS` — memory OS, não resolve code AST.
- `oxigraph/oxigraph` — RDF/SPARQL, fora do escopo (ADR-004 fixa SQL recursivo).
- `deeplethe/forkd` — microVM sandbox, fora do escopo.

O `go-tree-sitter` (`github.com/tree-sitter/go-tree-sitter`) é o binding Go oficial do tree-sitter, mantido pelo time tree-sitter, com CGO para a biblioteca C nativa. Bindings alternativos como `smacker/go-tree-sitter` (mais leve, mas menos mantido) foram considerados.

A pergunta que este ADR responde: **como adicionar indexação estruturada de código-fonte ao my-memory, mantendo a stack Go-first (ADR-002), respeitando o SQLite unificado (ADR-001), e habilitando queries híbridas código↔markdown (ADR-009)?**

## Decision Drivers

- **DR-1**: Fechar o gap de code intelligence (sem mexer no parser Markdown que está sólido).
- **DR-2**: Manter Go-first (ADR-002). Tree-sitter é CGO, mas o binding oficial (`go-tree-sitter`) é maduro e mantido pelo time upstream.
- **DR-3**: SQLite unificado continua sendo fonte de verdade (ADR-001). AST é uma tabela a mais, não um banco paralelo.
- **DR-4**: Reusar o motor de busca híbrida (RRF, ADR-009) — chunks de código e chunks de Markdown competem no mesmo ranking.
- **DR-5**: Não ser dogmático sobre linguagens — começar com top-10 (Go, Python, TS/JS, Rust, Java, C/C++, Ruby, PHP, Shell) e expor `--lang=all` para as 158 suportadas pelo tree-sitter.
- **DR-6**: Performance — indexar 1000 arquivos Go em ≤5s e resolver query em ≤50ms p99 (alinhado com o codebase-memory-mcp, que é referência de mercado).
- **DR-7**: Sem lock-in: se tree-sitter regredir, o parser de código pode ser desligado por flag (`--code-ast=false`) sem impacto no Markdown path.

## Considered Options

1. **Status quo (código como texto opaco)** — descartado: gap de queries estruturadas; desperdício de tokens no LLM; sem grafo de código.
2. **Tree-sitter binding `smacker/go-tree-sitter` (puro Go via gyp)** — descartado: comunidade migrou para `go-tree-sitter` oficial; menos releases; cobertura de linguagens menor.
3. **AST custom em Go para cada linguagem** — descartado: reinventa a roda; custo de manutenção proibitivo para 10+ linguagens; tree-sitter já tem gramáticas comunitárias estáveis.
4. **Bindings via sidecar Python (`py-tree-sitter`)** — descartado: viola ADR-042 (workers sidecar isolados) — sidecar só faz sentido para I/O pesado (ASR/TTS), não para parsing de 1000 arquivos locais; CGO em Go já dá a mesma cobertura.
5. **`github.com/tree-sitter/go-tree-sitter` (oficial, CGO para a lib tree-sitter C)** — *Opção Escolhida*. Madura, mantida pelo time upstream, cobertura idêntica ao `py-tree-sitter`/CLI, sem sidecar.

## Decision Outcome

**Opção 5 escolhida**: adicionar **tree-sitter como dependência opcional** (CGO) via `github.com/tree-sitter/go-tree-sitter`. O parsing de código vira um **estágio opcional** dentro do pipeline `mem index` (controlado por `--code-ast[=true|false]`, default `true` quando tree-sitter está disponível).

### Tabelas SQLite adicionadas

```sql
-- Nó de código: arquivo de source. 1:1 com arquivo físico dentro do escopo (ADR-016).
CREATE TABLE code_files (
  id INTEGER PRIMARY KEY,
  path TEXT NOT NULL UNIQUE,
  language TEXT NOT NULL,           -- 'go', 'python', 'typescript', etc
  size_bytes INTEGER NOT NULL,
  content_hash TEXT NOT NULL,       -- sha256 (reuso ADR-010)
  ast_hash TEXT,                    -- sha256 da AST serializada — se mudou, drift de estrutura mesmo com bytes idênticos
  indexed_at INTEGER NOT NULL
);

-- Símbolo extraído pela AST: function, method, struct, class, interface, import, constant.
CREATE TABLE code_symbols (
  id INTEGER PRIMARY KEY,
  file_id INTEGER NOT NULL REFERENCES code_files(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,               -- 'function','method','class','interface','struct','import','constant','variable'
  name TEXT NOT NULL,
  qualified_name TEXT,              -- 'pkg.Foo' para Go, 'ClassName.method' para OO — usado em joins
  signature TEXT,                   -- assinatura completa (para preview)
  start_line INTEGER NOT NULL,
  end_line INTEGER NOT NULL,
  start_col INTEGER,
  end_col INTEGER,
  doc_comment TEXT,                 -- docstring adjacente (se houver)
  parent_symbol_id INTEGER REFERENCES code_symbols(id),  -- método dentro de classe, etc
  UNIQUE(file_id, kind, qualified_name, start_line)
);

-- Aresta estrutural: relação derivada da AST, com proveniência distinta de EXTRACTED (wikilink).
CREATE TABLE code_edges (
  id INTEGER PRIMARY KEY,
  src_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
  dst_symbol_id INTEGER NOT NULL REFERENCES code_symbols(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,               -- 'calls','imports','embeds','implements','extends','references','uses_type'
  file_id INTEGER NOT NULL REFERENCES code_files(id),
  start_line INTEGER NOT NULL,
  end_line INTEGER NOT NULL,
  UNIQUE(src_symbol_id, dst_symbol_id, kind)
);

CREATE INDEX code_graph_kind_idx ON code_edges(kind);
CREATE INDEX code_symbols_kind_name_idx ON code_symbols(kind, name);
CREATE INDEX code_files_language_idx ON code_files(language);
```

### Pipeline de indexação (estendido)

```
mem index
  ├── markdown_pipeline (existente)
  │   ├── parse Markdown (ADR-008)
  │   ├── extract wikilinks → edges (ADR-011)
  │   ├── FTS5 chunking
  │   └── embed + index
  └── code_pipeline (NOVO, opt-in via --code-ast)
      ├── detect language (extensão + shebang)
      ├── parse com tree-sitter → AST
      ├── extract symbols → code_symbols
      ├── resolve references entre arquivos → code_edges
      ├── FTS5 chunking (por símbolo, não por linha)
      └── embed + index (opcional, vetor por símbolo)
```

### Busca híbrida ampliada (RRF ADR-009)

```
mem search "X"  →
    FTS5 (BM25)  ─┐
    vec (k-NN)   ─┤── RRF ── resultados ranqueados
    graph CTE    ─┤
    code_symbols ─┘    (NOVO: match exato em qualified_name quando query parece nome de símbolo)
                       ex: "mem search 'Database.Open'" prioriza Go:db.Database.Open
```

### CLI nova superfície

```bash
mem code-index [--lang=go,py,...] [--include=glob] [--exclude=glob]
                [--ast-hash] [--no-embed] [--storage=sqlite|postgres]
mem code-search <query>           # nova fachada opcional; delega para mem search com flag --kind=code
mem code-graph <symbol>           # novo: mostra vizinhos AST (calls/imports/etc) — implementa sub-grafo
mem code-stats                    # novo: distribuição por linguagem, símbolos por kind, drift AST
```

MCP server ganha:

- `memory_code_search(query, language?, kind?)` — busca estruturada
- `memory_code_neighbors(symbol, depth?)` — grafo AST local (calls/imports) com deep links

### Compatibilidade CGO

- `tree-sitter` (CGO) é **opcional**: o build detecta se `CGO_ENABLED=1` e a lib C está presente. Se faltar, código de code-ast vira no-op com log warning; pipeline Markdown segue intacto.
- Windows/MSVC: usuário precisa de `gcc` (mingw) ou MSVC; build script em `Makefile` ou `Taskfile.yml` documenta.
- Linux/macOS: build padrão funciona out-of-the-box.
- CI: novo step `go build -tags treesitter` no `.github/workflows/` para compilar e testar o path CGO; o path padrão (sem tag) testa o no-op.

### Granularidade de busca por símbolo

Quando a query casa exatamente um `qualified_name` (ex: "Database.Open" tem ponto e camelCase), o motor dá **boost** a esse símbolo no RRF (peso 2x). Isso aproveita a estrutura sem obrigar sintaxe especial do usuário.

### Performance esperada (alvos)

| Operação | Alvo p99 |
|---|---|
| Indexar 1000 arquivos Go (200KB médio) | ≤ 5s |
| Indexar 100 arquivos Python | ≤ 2s |
| Query híbrida (100k chunks total, 50% code) | ≤ 50ms |
| Resolver `code_graph Database.Open` depth=2 | ≤ 100ms |

Benchmarks viram `bench/` com harness reproduzível (similar aos `bench/embed-*` existentes).

### Cross-references com ADRs existentes

| ADR | Como interage com ADR-047 |
|---|---|
| **ADR-001** (SQLite unificado) | `code_*` tabelas são parte do mesmo `memory.db` |
| **ADR-002** (Go-first) | CGO permitido para dependências nativas críticas |
| **ADR-004** (Grafo SQL recursivo) | `code_edges`参加的 na CTE — `--include-code-edges` flag global |
| **ADR-009** (RRF) | RRF passa a incluir `code_symbols` no ranqueamento |
| **ADR-010** (Cache SHA-256) | `ast_hash` detecta mudanças estruturais mesmo com bytes idênticos |
| **ADR-011** (Arestas epistemicas) | Arestas `code_edges` têm proveniência distinta: `kind: AST` (não EXTRACTED nem INFERRED) |
| **ADR-016** (Auto-scoping) | `code_index` respeita `.memory/config.yaml` `scope.include/exclude` |
| **ADR-031** (Semantic drift) | `ast_hash` alimenta drift detector: nota descrevendo X mudou quando AST de X mudou |
| **ADR-040** (Storage global) | Code AST roda igual sobre SQLite local ou Postgres/pgvector |
| **ADR-046** (Deferred until) | Plano B = `gopls` LSP se tree-sitter regredir (gated) |

## Consequences

### Positive

- **Code intelligence real** — queries estruturadas sobre código com boost por qualified name.
- **Code-aware LLM** — agente recebe tipo/posição do símbolo, não só prosa embeddada.
- **Drift detection mais preciso** — `ast_hash` detecta mudanças estruturais ignoradas por `content_hash`.
- **Reuso de infra** — RRF, FTS5, sqlite-vec, file-watcher (ADR-017) ganham code path com pouco código novo.
- **Compatibilidade mantida** — `code-ast` é opt-in; sem CGO o sistema segue funcionando só com Markdown.
- **Performance alinhada com mercado** — alvos p99 baseados em codebase-memory-mcp (43.9k⭐ referência).

### Negative

- **CGO introduz dependência de toolchain C** — usuário precisa de gcc/MSVC; build script documenta. Mitigação: build sem CGO continua funcionando (só sem code AST).
- **Complexidade do grafo cresce** — `code_edges` multiplica cardinalidade. Mitigação: índice por `(src, kind)` e lazy-load de sub-grafos.
- **Falsos positivos de referência** — tree-sitter pode resolver nomes via heurística que erra em dynamic dispatch. Mitigação: campo `confidence` em `code_edges`, edges com confidence < 0.5 ficam fora do grafo principal e entram em tabela `code_edges_uncertain` para revisão manual.
- **Cobertura inicial limitada** — só top-10 linguagens na v1; resto via `--lang=all` (que baixa gramática on-demand na primeira execução).

### Neutral

- **Não substitui parser Markdown** — Markdown path intocado.
- **Não muda formato de notas** — wikilinks e tags funcionam idêntico.
- **Inferência de tipos fica para v2** — `uses_type` no v1 cobre só referências literais; análise semântica (call graph com types) fica como ADR futuro.

## Compatibility

- **Backward compatible**: sem CGO ou com `--code-ast=false`, comportamento idêntico ao v1.x.
- **Schema migration** automática no boot (mesmo padrão do `InitDB` em `internal/db/db.go`).
- **MCP**: tools novas são aditivas; tools existentes (`memory_search`, `memory_get_drift`) ganham flag opcional `include_code` (default false para não surpreender clientes).
- **Postgres**: tabelas `code_*` criadas via mesma migração (idempotente).

## Implementation Plan

1. ✅ Spec `tlc-spec-driven` `.specs/features/feat-code-ast/` — Specify → Tasks → Execute.
2. ADR-047 (este doc) referenciado em `docs/adr/README.md`, `docs/README.md`, `README.md`, `AGENTS.md`.
3. Tabela `code_*` na próxima migração (`internal/db/db.go::InitDB`).
4. Binding `github.com/tree-sitter/go-tree-sitter` em `internal/codeast/parser.go`.
5. Build tag `//go:build treesitter` para isolar path CGO.
6. CI matrix: (linux/macos/windows) × (CGO on/off).
7. Benchmarks em `bench/codeast/` com harness reproduzível.
8. Atualizar `docs/CLI_GUIDE.md`, `docs/AGENT_INTEGRATION_GUIDE.md`, `docs/ARCHITECTURE.md`.

## Deferral: Plano B (gopls LSP)

### Condições-gatilho para promoção

1. **tree-sitter quebrar em CGO binding** — issue em `github.com/tree-sitter/go-tree-sitter` sem resposta por ≥90 dias que afete o my-memory (build quebrado, panic em runtime).
2. **cobertura de linguagens insuficiente** — para uma linguagem top-10 (Go, Python, TS, Rust, Java, C/C++, Ruby, PHP, Shell, C#) não houver gramática tree-sitter estável por ≥180 dias.
3. **gopls adicionar parser oficial exposto via biblioteca** — projeto `golang/tools` (gopls) ou `gopls/internal/golang` expor API pública de parsing de Go com cobertura ≥95% das construções Go (structs, generics, interfaces, embed) até 2027-06-21.

**Não é gatilho válido**: "queremos melhor qualidade", "vamos esperar", "talvez precise". Plano B só promove com evidência (issue + log + binário quebrado, ou gramática ausente, ou API pública gopls).

### O que fica proibido até a promoção

- ❌ Criar `internal/codeast/lsp_adapter.go` ou qualquer arquivo com sufixo `_lsp.go`.
- ❌ Adicionar bloco `code_ast.backend: lsp` em `.memory/config.yaml` schemas.
- ❌ Adicionar flag `--code-ast-backend=lsp` em qualquer help text ou doc.
- ❌ Adicionar dependência `golang.org/x/tools/gopls` em `go.mod` (só `tree-sitter`).
- ❌ Implementar IPC entre Go e processo gopls (`net/rpc`, `jsonrpc2`).

### Como promover (quando um gatilho dispara)

1. Abrir issue em `.specs/issues/047-lsp-fallback.md` referenciando ADR-047 + evidência do gatilho.
2. Criar ADR novo (slot ADR-052+ quando ADR-048/049/050/051 forem atribuídos) com análise do gopls como alternativa — incluindo trade-offs de dependência de processo externo, latência de IPC, cobertura.
3. Atualizar este ADR movendo Status da subseção "Status atual" para **Promoted** com link cruzado.
4. Implementar via spec `tlc-spec-driven` (spec.md → tasks.md → batches com quality gate).
5. Atualizar `docs/REFERENCES.md` seção 1.9 + `docs/adr/README.md`.

### Status atual

- **Última revisão**: 2026-09-21
- **Gatilhos disparados**: 0
- **Evidence log**: nenhum ainda — `tree-sitter/go-tree-sitter` ativo e mantido, gramáticas top-10 estáveis.
- **Próxima revisão**: 2027-03-21 (6 meses) ou quando 1ª issue significativa de CGO binding aparecer.

## References

- **`DeusData/codebase-memory-mcp`** — referência de mercado para performance (43.9k⭐). Influenciou alvos p99 e o pattern Cypher-like.
- **`iwe-org/iwe`** — markdown knowledge graph concorrente (1.7k⭐); confirmou que tree-sitter é o caminho certo (eles usam AST próprio só pra markdown, deixando code path aberto).
- **`github.com/tree-sitter/go-tree-sitter`** — binding Go escolhido.
- **ADR-001** (SQLite unificado) — code_* tabelas no mesmo banco.
- **ADR-002** (Go-first) — justifica CGO aqui; ADR-002 permite CGO para dependências nativas críticas.
- **ADR-009** (RRF) — passa a incluir code_symbols no ranqueamento.
- **ADR-010** (Cache SHA-256) — `ast_hash` complementa `content_hash`.
- **ADR-011** (Arestas epistemicas) — `code_edges` ganha proveniência `kind: AST`.
- **ADR-016** (Auto-scoping) — code_index respeita `.memory/config.yaml`.
- **ADR-031** (Semantic drift) — `ast_hash` alimenta drift detector.
- **ADR-040** (Storage global) — code AST roda em SQLite local ou Postgres.
- **ADR-046** (Deferred until) — template aplicado no Plano B (gopls).
- **Pesquisa autoral `pesquisa-infraestrutura-autoral-mymemory.md`** §6 (escolhas tecnológicas) — base para CGO permitido.