---
title: "Validation: code-graph-real — Fase 2 (grafo estrutural)"
category: resource
summary: "Prova empírica da Fase 2: arestas calls/imports/inherits/uses_type com call-site, provenance EXTRACTED/INFERRED e bucket de incertas. 7 commits (4f1182a..a680955)."
type: validation
created_at: 2026-10-23T23:40:00-03:00
tags: [validation, code-graph-real, code-ast, epistemic-status, call-site]
---

# Validation: code-graph-real — Fase 2 (grafo estrutural)

**Data**: 2026-10-23
**Worker**: mavis (sub-agente, branch `mvs_138754ab783a4581bbbeb2f33abd08aa`)
**Spec**: `.specs/features/code-graph-real/spec.md`
**Tasks**: `.specs/features/code-graph-real/tasks.md` (T7–T12; T7–T12 desta fase)
**Commits**: `4f1182a` → `a342518` → `b986c37` → `870a323` → `5f52c2d` → `056a3f6` → `a680955`
**Verdict**: **PASS** — o code graph deixou de ser "símbolos + `references` por nome" e virou grafo estrutural com call-site e provenance por aresta.

---

## 1. O que foi entregue (por tarefa)

| Tarefa | Entrega | Arquivo:linha | Commit |
| :--- | :--- | :--- | :--- |
| T7 | Backend AST em Go puro (`go/parser`+`go/ast`+`go/token`, stdlib): símbolos com kind, nome, qualified name, assinatura, início/fim (linha+coluna), doc comment e receiver | `internal/codeast/go_parser.go:1` | `4f1182a` |
| T8 | `newBackendParser()` deixou de ser código morto e devolve o motor Go; o pipeline usa o backend padrão (mock só como fallback das linguagens sem backend AST); log de boot honesto | `internal/codeast/treesitter_disabled.go:33`, `internal/codeast/pipeline.go:80` | `a342518` |
| T9 | Arestas `calls` com **call-site**; coluna `code_edges.epistemic_status` (TEXT, default `EXTRACTED`) + migração idempotente (SQLite e Postgres) + `UpsertCodeEdgeWithStatus`; resolução em 3 níveis | `internal/codeast/go_parser.go:263`, `internal/db/code_migration.go:60` | `b986c37` |
| T10 | Arestas `imports` (com os símbolos sintéticos que dão corpo ao grafo) + `inherits` + `uses_type` | `internal/codeast/go_parser.go:359` | `870a323` |
| T11 | Fechamento do roteamento para `code_edges_uncertain` (confidence + reason), teste de idempotência da migração e default correto das linhas pré-existentes | `internal/codeast/pipeline.go:452`, `internal/db/code_migration_test.go:1` | `5f52c2d` |
| T12 | Resposta MCP com `epistemic_status`, `confidence`, call-site e posição do destino (campos aditivos) | `internal/mcp/code_handlers.go:1` | `056a3f6` |
| — | Correção achada pela própria prova empírica: aresta de TIPO não pode resolver para um MÉTODO de mesmo nome | `internal/codeast/pipeline.go:486` | `a680955` |

---

## 2. Prova empírica (números, não adjetivos)

Code stage real sobre **este repositório**, SQLite local, banco temporário
(`--storage=sqlite --db <tmp>`), escopo `--include=**/*.go --lang=go`:

```
code AST: backend go (stdlib); tree-sitter: disabled
{"engine":"sqlite","repository":"proof/repo","files_scanned":349,"files_indexed":349,
 "files_skipped":0,"files_errors":0,"symbols_total":6754}
```

### 2.1 Arestas por kind (`code_edges`)

| kind | arestas |
| :--- | ---: |
| `calls` | 4055 |
| `imports` | 1984 |
| `uses_type` | 908 |
| `references` | 1 |
| `inherits` | 0 |
| **TOTAL** | **6948** |

* `references` = 1: é a **varredida legada por nome**, que só roda quando o parser
  não devolveu arestas — `internal/supervisor/lock_windows.go:17` (arquivo sem
  nenhuma chamada/import/tipo). Provenance `INFERRED`, confidence 0.80.
* `inherits` = 0 **é o número correto para este repositório**: não existe
  embedding de struct nem de interface no código do repo. A única ocorrência
  textual é dentro da string-fonte do fixture de teste
  (`internal/codeast/go_parser_test.go:284`, `Base` embutido), que o parser Go
  não vê como código. A implementação é coberta pelo teste unitário
  `TestGoParserExtraiImportsEInherits` (2 arestas `inherits` no fixture).

### 2.2 Provenance (`epistemic_status`)

| status | arestas |
| :--- | ---: |
| `EXTRACTED` (resolved by package scope) | 6367 |
| `INFERRED` (resolved by name only) | 581 |

Nas arestas `calls`: `EXTRACTED` 3486, `INFERRED` 569.

`code_edges_uncertain`: **10593 linhas** (nunca promovidas a aresta resolvida)

| kind | linhas | confidence média |
| :--- | ---: | ---: |
| `calls` | 8811 | 0.328 |
| `uses_type` | 1782 | 0.300 |

Por bucket de razão: 9350 `destino não encontrado no repositório` (chamadas a
`testing.T`, `log.Errorf`, `context.Background`, … — destinos fora do repo) e
1243 `ambíguo: N símbolos com o nome "X"` (múltiplos candidatos).

### 2.3 Call-site ≠ linha da definição (exemplos reais)

```
main.newBuiltinEmbedder --calls--> main.defaultModelsDir
   call-site : cmd/mem/builtin_embedder.go:41  (epistemic_status=EXTRACTED, confidence=1.00)
   definicao : cmd/mem/builtin_embedder.go:14

main.newBuiltinEmbedder --calls--> embedder.NewBuiltinClient
   call-site : cmd/mem/builtin_embedder.go:49  (epistemic_status=EXTRACTED, confidence=1.00)
   definicao : internal/embedder/builtin.go:71

main.runClustersCLI --calls--> main.runClustersCommand
   call-site : cmd/mem/clusters.go:30           (epistemic_status=EXTRACTED, confidence=1.00)
   definicao : cmd/mem/clusters.go:34
```

Caso `INFERRED` (destino resolvido só pelo nome, confidence 0.80):

```
main.formatClustersJSON --calls--> embedder.Tokenizer.Encode
   call-site : cmd/mem/clusters.go:118  (INFERRED, conf=0.80)
   definicao : internal/embedder/tokenizer.go:79
```

### 2.4 `imports` com destino que EXISTE

```
import:os --imports--> os (kind do destino=import) em cmd/mem/builtin_embedder.go:6  status=EXTRACTED
import:github.com/FelipeMiiller/my-memory/internal/config --imports-->
    github.com/FelipeMiiller/my-memory/internal/config (kind do destino=import)
    em cmd/mem/builtin_embedder.go:9  status=EXTRACTED
```

`imports: 1984 | com destino inexistente: 0` — nenhuma aresta `imports` aponta
para símbolo ausente ou para o símbolo errado.

### 2.5 Código-fonte NÃO virou nota

Indexação completa (`mem index --storage=sqlite --db <tmp>`) deste repositório:

```
documentos no total: 265
documentos .go: 0
```

Nenhum documento com path `.go`; o code stage é um estágio separado
(`code_files`/`code_symbols`, não `documents`).

### 2.6 Baseline (símbolos por arquivo)

| | baseline Fase 1 | agora | delta |
| :--- | ---: | ---: | ---: |
| arquivos indexados | 331 | 349 | +18 |
| símbolos | ~2595 | 6754 | +4159 |

**Razão do delta de arquivos**: entre o commit de baseline (`340c9cb`) e o HEAD
entraram 8 arquivos `.go` rastreados — 2 meus (`internal/codeast/go_parser.go`,
`internal/codeast/go_parser_test.go`) e 6 de workers paralelos
(`cmd/mem/lesson*.go`, `internal/db/code_ast_hash_test.go`,
`internal/db/hybrid_code_test.go`, `internal/store/code_search*.go`). O resto
(+10) é a diferença de escopo do snapshot de 331 da Fase 1, que não é
reproduzível a partir da árvore atual.

**Razão do delta de símbolos** — decomposição de 6754:

| kind | símbolos |
| :--- | ---: |
| `import` | 3968 (= 1984 specs × 2 nós sintéticos: declaração + pacote) |
| `function` | 1813 |
| `method` | 355 |
| `struct` | 252 |
| `constant` | 172 |
| `variable` | 135 |
| `class` | 49 |
| `interface` | 10 |

Nenhum símbolo foi **perdido**: o mock por regex contava só linhas `func`/
`type`/`import `; o backend AST extrai também métodos, constantes, variáveis,
tipos nomeados e interfaces, e cria os dois nós sintéticos por import
(decisão documentada no item 6 do briefing). É crescimento de cobertura
estrutural, não regressão.

---

## 3. Decisões do gestor aplicadas

1. **Go puro** — `go/parser`/`go/ast`/`go/token` da stdlib. `go.mod` sem
   dependência nova, sem CGO, sem build tag. A ADR-047 **não** foi tocada: o
   deferral do tree-sitter segue intacto (0/3 gatilhos) e o caminho com a tag
   `treesitter` continua isolado e com precedência
   (`internal/codeast/treesitter_enabled.go`).
2. **Contrato do Parser aditivo** — `FileResult.Edges []Edge` (novo campo);
   `MockParser` e os testes existentes seguem funcionando. A forma do `Edge`
   está documentada no doc comment (`internal/codeast/parser.go:1`).
3. **Schema antes das arestas** — a coluna `epistemic_status` e o helper de
   upsert entraram no commit de T9, antes da primeira aresta ser gravada; a
   migração é idempotente nos dois engines (`EnsureCodeTables`, com
   `pragma_table_info` no SQLite e `information_schema.columns` no Postgres).
4. **Semântica do `calls`** — origem = símbolo ENCLOSING; destino = símbolo
   chamado; `file_id/start_line/end_line` = **call-site**. A mudança de
   semântica da coluna está documentada em `UpsertCodeEdgeWithStatus`.
5. **3 níveis de resolução** — escopo de pacote → `EXTRACTED`; só nome (único)
   → `INFERRED`; ambíguo/não resolvido → `code_edges_uncertain` com
   `confidence` + `reason`, **nunca** promovido a aresta resolvida.
6. **`imports` com alvo que existe** — opção (a): cada import cria dois
   símbolos sintéticos de kind `import` (a declaração, origem da aresta, e o
   pacote importado, destino da aresta). A aresta `imports` liga a declaração
   ao pacote; o destino é sempre uma linha real de `code_symbols`
   (1984/1984, zero órfãs). O nó do pacote é por arquivo, e a resolução da
   aresta é escopada por `file_id`.
7. **A varredida por nome não duplica** — `resolveAndInsertEdges` só roda
   quando o parser **não** devolveu arestas; nesse caso o comportamento
   legado continua idêntico (e as arestas `references` são `INFERRED`, por
   terem sido resolvidas por nome).
8. **Log de boot** — agora imprime `code AST: backend go (stdlib);
   tree-sitter: disabled` (não é mais "(code pipeline skipped)"), porque o
   code pipeline roda. `IsTreesitterEnabled()` continua `false` sem a tag e
   `true` com a tag.
9. **Vocabulário de arestas** — `ValidEdgeKinds()` / `ValidEdgeKind()` com
   `calls`, `imports`, `inherits`, `uses_type`, `references`.

---

## 4. Gate executado (verde antes de cada commit)

```
go build ./...                                                    → 0
go vet ./...                                                      → 0
go test -count=1 ./internal/codeast/... ./internal/db/... ./internal/mcp/...   → ok (3/3)
go test '-bench=.' '-benchmem' '-run=^$' ./internal/codeast ./internal/db ./internal/mcp → ok (3/3)
gofmt -l (cópias LF dos arquivos tocados)                          → vazio
```

Suíte completa (`go test ./...`) **não** foi rodada: o gestor roda no final,
sem escritores ativos (a regra da tarefa). Durante a fase, o único teste que
falhou no pacote `internal/db` foi `TestHybridRRFIncludeCodeEntraNoFanIn`,
pertencente ao trabalho **paralelo** (T14) em `internal/db/hybrid.go` +
`internal/store/code_search.go` (arquivos fora da minha propriedade); ele
voltou a passar quando o worker parallel fechou o código.

---

## 5. Desvio declarado (arquivo fora da minha propriedade)

- `internal/store/postgres.go` (proibido para mim) contém o DDL Postgres de
  `code_edges` **sem** a coluna `epistemic_status`. O caminho Postgres não
  quebra por isso: a coluna é criada pela migração idempotente
  `ALTER TABLE ... ADD COLUMN` de `EnsureCodeTables` (dialect-aware), que roda
  no início do pipeline, antes de qualquer escrita. O `CREATE TABLE` do
  Postgres precisa da coluna para bases novas — **pendência para o gestor /
  para o dono do arquivo**.

## 6. O que NÃO foi feito (e por quê)

- **Não** criei ADR novo: a mudança de schema (coluna `epistemic_status`) segue
  o padrão já existente em `graph_edges` (ADR-011) e a migração é aditiva; a
  regra "mudança de banco exige ADR" continua valendo para o gestor decidir.
- **Não** troquei o `CREATE TABLE` Postgres (arquivo de outro worker).
- **Não** alterei `cmd/mem/codegraph.go`: a tabela markdown do CLI ainda mostra
  só `file`/`line` (que já é o call-site); os campos novos de provenance
  aparecem na **tool MCP** `memory_code_neighbors`, que é o contrato da
  feature. O CLI está fora da minha propriedade.
- **Não** implementei satisfação de interface em geral (um tipo que implementa
  uma interface sem embedding): isso exige type-checking (`go/types`), fora do
  escopo do backend AST. O embedding (struct e interface) é coberto.
- **Não** rodei `go test ./...` (regra da tarefa) nem escrevi em
  `.specs/features/code-graph-real/tasks.md` (o gestor atualiza os
  checkboxes).
